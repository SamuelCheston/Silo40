# Agent 操作员模式（外部 AI 真实跑局采样）

> 本文档由 AI 维护 (project-source)。实现：`internal/service/agent_game.go`、
> `internal/api/debug_http.go`（/debug/agent/* 路由）、`cmd/agenthost`（无头宿主）。
> 面向 agent 的短协议在 `/g/silo40-npc-ml/data/GENERATION_PROMPT.md` 第 1 节。

## 目的

让有工具调用能力的外部 AI agent **真实坐在游戏里操作**：引擎在每个 NPC 决策点
暂停，把局面与候选发给 agent，agent 提交选择，引擎执行并记录。决策记录是
`/g/silo40-npc-ml` 数据管线的直接输入（真实分布样本），不是合成数据。

## 架构

- `AgentSession`（service 层）实现引擎钩子 `engine.NpcPolicy`（Advise/Committed）
  与 `engine.ExecutionObserver`（NpcExecuted）。
- 会话自带 runner goroutine 持有并推进 `PassTime(1)`；HTTP 侧经
  `stepCh/parkCh/quitCh` 与 runner 乒乓：
  - `step`/`choose` 发送信号后等待下一个 park（决策点或月末），永远不持有 `s.mu`
    等待（`agentSession` 指针由独立的 `agentMu` 保护——runner 在 PassTime 内
    持有 `s.mu` 等待决策，若 HTTP 侧也走 `s.mu` 会死锁）。
  - 决策点挂起时 `parkCh` 把 `AgentDecisionContext` 交给 HTTP 响应；
    `choose` 校验后经 `pending.doneCh` 唤醒 tick。
- agent 的选择通过 `NpcAdvice.Force` 进入引擎：被选候选**就是**执行的候选
  （加权抽签照常消耗，rng 流顺序不变）。
- 每决策等待预算缺省 30 分钟（`SILO40_AGENT_TIMEOUT_MS` 或 start 的
  `timeout_ms` 覆盖）。超时 = 会话进入 error（该决策记 `timeout_fallback`，
  本月以原权重收尾，记录保留可导出）。

## 会话状态机

`idle`（等 step）→ `running`（tick 内）→ `awaiting_choice`（挂起等 choose）→
`running` → … → 月末回 `idle`；终态：`game_over` / `stopped` / `error`。
终态后只能 export，或 start 新局（CreateGame 会整体替换存档图）。

## 决策记录格式（管线契约）

每条记录 = `pipeline.py validate_raw` 的归一化形状（JSON 键与 SQLite 导出路径
产物一致）：

```json
{
  "decision_id": "agent-7-20261114T000440-0003",
  "game_id": "agent-7-20261114T000440",
  "occurred_at": "Y123-M04",
  "npc_id": "NPC:3",
  "decision_version": "agent-operator-v1",
  "prompt": "{\"actor\":{...},\"silo\":{...},\"departments\":[...],\"factions\":[...]}",
  "candidates": [
    {"action_id": "GATHER_INFO", "args": {"target_dept": "Medical"},
     "executable": true, "original_weight": 2.1, "model_score": 8.0,
     "willingness": 1.0, "final_multiplier": 1.7}
  ],
  "executed_action": {"action_id": "GATHER_INFO", "args": {"target_dept": "Medical"}},
  "execution_status": "executed",
  "execution_reason": "<agent 的 rationale>",
  "response": {"action_id": "GATHER_INFO", "args": {"target_dept": "Medical"}}
}
```

字段语义：

- `prompt`：决策时的状态快照（与 agent 看到的同源、同一战雾白名单：actor 自身
  切片 + 公开堡指标 + 部门公开数据 + 公开派系；多堡时含堡 id/name）。**不含候选**
  ——管线自行拼接候选公共视图。
- `candidates[].model_score`：agent 给该候选的 0-10 分；`final_multiplier` =
  `0.5 + score/10*1.5`（与 LLM 顾问的 score→乘数映射一致）；`willingness` 恒 1.0
  （agent 模式无通道 B）；`original_weight` = 引擎基础权重。
- `response` 恒等于 `executed_action`：服务端把 agent 的 `candidate_id` 映射回
  严格动作结构，agent 的自由文本不直接当标签。
- `execution_status`：`executed`（通过）/ `execution_failed`（SubmitAction 拒绝，
  带原因）/ `timeout_fallback` / `aborted`。后三者被管线丢弃。
- 记录时点：`Committed`（引擎官方提交点）追加；`NpcExecuted` 用真实执行结果回填。

## action_id 词表

管线要求 `action_id` 在单条记录内唯一，因此：

- 六个基础类型直接用类型字符串：`GATHER_INFO` / `SHARE_INFO` /
  `BUILD_CONNECTION` / `CONDUCT_PROPAGANDA` / `INCITE_REBELLION` /
  `PUBLICIZE_FACTION`（每类在候选集中至多出现一次）。
- 职业行动用其（全局唯一的）定义 ID，如 `MECHANICAL_OVERHAUL`，
  而不是共享的 `PROFESSION_ACTION`。
- `args` 只含目标参数：`target_dept` / `resource_target` / `fragments`
  （SHARE_INFO 的共享碎片，逗号连接）。
- 反向映射（训练后的模型输出 → 引擎动作）：action_id 属于七类型枚举 →
  `Type=该类型`；否则 `Type=PROFESSION_ACTION, ProfessionAction=action_id`。

## 引擎修复（2026-11-14，随本特性落地）

`scoreCandidates` 过去只把目标写进候选的 message 文本，不进动作本身，导致
NPC 的 GATHER_INFO/SHARE_INFO/BUILD_CONNECTION 被选中后在执行端
（`events/mechanics/common_actions.js` 要求 `target_dept`）**必然失败**。
现在候选评分时 rng 选中的目标随动作下发（`TargetDept`/`FragmentIds`），
rng 消耗顺序不变，但动作真实生效。

**影响**：模拟轨迹自此变化，`references/parity-v6-three-silo.json` 基准失效。
完整影响面回溯与基准重建约定见 `references/known-issues.md` 的 KI-1/KI-2
（结论：基准不单独重建，等加权结局与 v7 存档提交后与新三堡构成一起重建）。

## 无头宿主 cmd/agenthost

```bash
go build -o agenthost ./cmd/agenthost && ./agenthost -addr 127.0.0.1:18080 -db /tmp/agenthost.db
```

- 从含 `events/` 的目录运行（内容解析顺序：二进制同目录 → cwd）。
- 与 GUI 应用共用同一套 debug 路由；不加载本地模型、无显存占用。
- 单会话串行；SIGINT/SIGTERM 干净退出。

## 产率实测（种子 42/43/44，机械策略，单堡）

| 指标 | 数值 |
|---|---|
| 单局时长 | 31-43 月（机械打法快速 information 结局） |
| 决策点/局 | 31-41（100% 执行） |
| 候选/决策 | 5.4-5.8 |
| 状态输入/决策 | ~1.1 KB ≈ 280 token |
| 管线转换 | 110 条 → 110 SFT + 107 偏好对 + 613 回归，0 丢弃 |

复现：`SILO40_AGENT_YIELD=1 go test ./internal/service/ -run TestAgentYieldMeasurement -v`。

## 测试

`internal/service/agent_game_test.go`：完整对局 + 管线形状断言、choose 严格校验、
超时会话终止、stop 中止路径；`-race` 通过。

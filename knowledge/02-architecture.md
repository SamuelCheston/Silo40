# 架构与模块边界

## 1. 权威状态与调用路径

游戏权威状态由 `GameService` 持有：一个 World、按 SiloID 排序的多个 Silo、每堡一个 Agent、事件历史、内容状态、行动门控、延迟任务和当前操作。React 不保存另一份规则实现。

```text
Wails UI ───────────────┐
curl / cmd/sim ─────────┼─→ App / service API → GameService → GameEngine ↔ events/**/*.js
cmd/agenthost ─→ HTTP ──┘             │                 │
外部 AI ─→ /debug/agent/* ─→ AgentSession/NpcPolicy      │
                                             SQLite + FreeCache
```

Wails UI 使用 Begin/ProcessNext 队列接口；普通 debug HTTP 与 sim 使用同步 PassTime/ExecuteAction；Agent 模式用 runner goroutine 调用同步 PassTime，并在 NPC 决策点挂起等待选择。

## 2. 模块职责

| 路径 | 当前职责 |
| --- | --- |
| `main.go` | 嵌入前端和 events，启动 Wails |
| `app.go` | 启动升级链、内容加载、Resume、Wails API、LLM/Agent HTTP 适配 |
| `internal/model/` | World/Silo/Agent、结局仲裁、历史影响、存档版本与 DTO |
| `internal/service/game_service.go` | 单会话、三堡对象图、操作队列、内容、保存恢复、行动完成节点 |
| `internal/service/world.go` | 多堡月度分相、AI 代操作、跨堡队列、operation 顺序 |
| `internal/service/station.go` | 驻地派生、玩家知识裁剪、身份校验与行动目标 |
| `internal/service/silo1.go` | 完整 Silo 1、Archives、世界事实同步与情报产品 |
| `internal/service/silo18*.go` | 正典实体、N1–N9、具名人物和跨界信号 |
| `internal/service/action_gates.go` | 配置编译、阶段推进、三态求值、执行旁路拒绝 |
| `internal/service/weighted_endings.go` | 多主体贡献、候选分数与确定性仲裁 |
| `internal/service/agent_game.go` | Agent 会话状态机、停车/选择、执行记录和导出 |
| `internal/engine/` | 本地管线、统一 Actor、NPC、派系、脚本宿主与 mutation |
| `internal/api/` | 普通 game、LLM 和 Agent 的 debug HTTP 路由 |
| `events/` | 内容事件、核心 mechanic、行动公式与 action gate 注册表 |
| `frontend/` | 快照展示、操作输入、门控灰项原因和结局解释 |
| `cmd/sim/` | 通过普通 HTTP 驱动批量游戏 |
| `cmd/agenthost/` | 无 GUI 的真实游戏服务与 Agent HTTP surface |

## 3. Go、JS 与声明式 JSON 的边界

### 内容事件

`defineEvent` 提供结构化 trigger/effects 和受限 `script.canTrigger/apply`。脚本只读 Go 构造的快照并返回声明式结果，不能直接改 Go 对象。多堡 scope、世界 flag 和 cross signal 也都是封闭集，新增类型必须改 Go。

### 核心 mechanic

`defineMechanics` 承载 Agent/资源/思潮/指标/结局的月度公式、通用与职业行动、AgentStats 和评分。行动以 JS 为唯一权威，缺失或报错显式失败；核心月度系统仍有 Go fallback，改任一侧必须同步另一侧。

### 行动注册表

`events/action_gates.json` 不是事件，也不定义行动效果。它声明有序阶段、行动解锁条件和实时可用条件；实际效果仍来自 mechanic、内容事件或 Go operation 定义。生产启动缺配置会 fail-closed。

### Go 独占

三堡创建与驻地身份、世界阶段顺序、派系聚类与持久身份、Silo 18 正典、operation、加权仲裁、存档升级、Agent 会话以及所有 trigger/effect/mutation 的物理应用都在 Go。不能通过新增 JS 字段绕过这些边界。

## 4. 多堡月度结算顺序

普通三堡走 `runOneMonthWorld`：

1. **0a 延迟任务 staging**：全局延迟序号排序，按目标堡分组。
2. **0b 非玩家堡 AI 代操作**：按 SiloID 升序执行零时长动作。
3. **0c Silo 18 正典**：到期 N 节点强制生效，覆写 18 号状态并入队跨界信号。
4. **阶段 1 各堡本地相**：按 SiloID 升序，发布该堡延迟事件，执行 Agent → 资源 → 思潮 → NPC → 指标 → VICTORY_CHECK，并 Drain。
5. **2a 跨堡派发**：按 `(到期年月, CrossSeq)` 顺序，目标仍按 SiloID 升序。
6. **2b Silo 1 裁决**：读取本月各堡本地相后的状态。
7. **阶段 3 时钟**：世界时钟为权威，逐堡推进并镜像。
8. **玩家 operation**：驻地先于 Silo 1，按目标 SiloID 稳定结算。
9. **阶段 4 各堡后置相**：按 SiloID 升序执行旧规则/随机事件兼容路径和 TIME_CONTENT。
10. 清空本月 receivedSignals 与逐堡上下文，LLM 只观察玩家驻地本月日志。

单堡测试场景走 `runOneMonthSingle`，不进入跨堡阶段。每堡本地 VICTORY_CHECK 后，service 的 WeightedEndingResolver 只处理玩家驻地；Silo 18 不接入，Silo 1 的 N9 覆灭事实也不能被 peer 同步复活。

## 5. 队列、保存与接口一致性

- BeginAction/ExecuteAction 先由服务端解析真实成本，再检查 gate；通过后才扣 AP、排队或应用效果。
- 成功普通行动写入 `player_action:<mechanic id>` 内容状态，阶段推进依赖这些完成节点，而不是按钮点击或空转。
- UI 队列在完整月份/耗时操作结束以及 game over 时落 SQLite；同步 debug API 与部分即时动作路径只更新 FreeCache，除非 game over。需要立即保存完整快照时使用 Wails `SaveGame`，它拒绝 pending operation。
- 快照包含 `available_actions` 和 `world`；LOCKED 不下发，UNLOCKED/AVAILABLE 通过 visibility、enabled 和 disabled_reason 表达。

## 6. Agent 操作员是另一条控制面

`AgentSession` 实现 NpcPolicy 与 ExecutionObserver。每个 NPC 决策点把受限状态与候选交给外部 Agent，选择经 `NpcAdvice.Force` 回到同一行动管线；加权 RNG 抽取仍消耗，保证流位置不被绕过。HTTP goroutine 不拿 `GameService.mu` 等待，避免与 runner 死锁。

Agent 记录保存在会话内存，导出 JSON/JSONL；它不复用 LLM 的 SQLite 决策日志。普通游戏、LLM advisor、Agent 操作员共用引擎，但控制权和持久记录格式不同。

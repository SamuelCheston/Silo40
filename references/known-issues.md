# 已知问题与基准状态台账

> 本文档由 AI 维护 (project-source)：记录已修复但影响面需要回溯的旧缺陷，
> 以及当前失效的基准/产物。新条目按 KI-<序号> 追加，不删除历史条目。

## KI-1：NPC 目标动作丢失（已修复，2026-11-14）

**缺陷**：`engine/npc.go scoreCandidates` 为 GATHER_INFO / SHARE_INFO /
BUILD_CONNECTION 生成候选时，rng 选中的目标只写进了候选的 message 文本，
没有写进 `AgentAction`（`TargetDept` / `FragmentIds` 为空）。执行端
（`events/mechanics/common_actions.js` 的 action_gather_info /
action_share_info / action_build_connection）要求 `ctx.action.target_dept`
（share 还要求 `fragment_ids`），因此这三类 NPC 动作**被选中后必然执行失败**
（"Invalid target department." / "Invalid target or no fragments selected."）。
自 Go 引擎迁移（f497b60）起即存在，非近期回归。

**影响面（修复前的所有数据均失真）**：

- 受影响动作：仅 NPC 决策路径的 GATHER_INFO / SHARE_INFO / BUILD_CONNECTION。
  玩家行动不受影响（前端/执行 API 一直正确携带 target）；
  CONDUCT_PROPAGANDA / INCITE_REBELLION / PUBLICIZE_FACTION / PROFESSION_ACTION
  不需要目标，不受影响。
- 失真后果：NPC 之间的情报搜集、碎片共享、关系结交**从未真实发生过**——
  碎片扩散、部门间 connection 网络、跨部门知情度只由玩家行动与事件驱动；
  这三类动作的日志行也从未出现（日志以 `result.Executed` 为前提）；
  失败发生在 mutation 之前，NPC 行动点未被扣除。
- 受影响的历史产物：v6 parity 基准（见 KI-2）、此前所有 `cmd/sim` 批量模拟、
  `outputs/` 下的分析导出、LLM on/off 配对实验与 `llm_decision_logs` 里
  这些动作类型的"执行"记录——其轨迹都运行在"这三类 NPC 动作永远失败"的
  世界里，相关统计（碎片传播速度、部门关系演化、情报类动作占比与效果）
  系统性偏低。
- rng 注意：决策侧 rng 消耗顺序与修复后一致，但执行端处理器提前返回，
  其内部随机抽取（碎片选择/接受率掷骰）从未消耗——修复前后轨迹在首个
  命中的此类动作处即分岔。

**修复方式**（随 agent-operator 特性落地，`internal/engine/npc.go`）：
候选评分时把 rng 选中的目标直接写入动作——GATHER_INFO/BUILD_CONNECTION 带
`TargetDept`，SHARE_INFO 带 `TargetDept` + `FragmentIds`；`CandidateInfo`
新增 `FragmentIds` 透出。决策侧 rng 消耗不变。配套在 `RunNpcTurn` 增加
`engine.ExecutionObserver` 钩子，把真实执行结果回报给记录层（agent 模式用
它给训练样本回填 `execution_status`）。

## KI-2：v6 parity 基准失效（已销记，v7 基准已重建）

`references/parity-v6-three-silo.json`（commit 08eb21a 生成）曾**失效**，
原因有二：

1. KI-1 修复改变了 NPC 动作的真实执行，模拟轨迹自此分岔；
2. 行动门控与多实体加权结局（见 `references/mechanics-gating-endings.md`）
   已于 2026-11-14 落地（6e1d7fb → 8235a10，v7 存档），结局判定与开局
   行动集合均已改变。

**2026-11-14 重建完成**：新基准 `references/parity-v7-three-silo.json`，
新三堡构成（种子派生随机驻地 2–17 + Silo 1 + Silo 18，种子 7–12），含
KI-1 目标修复与加权结局仲裁；门控按单测惯例不加载（旧机制 fixture）。
双跑逐字节一致，SHA-256
`9f37dfa409a7efcf863819d6231de66042c6f8af3a4d9f742f8fa3e9196b7f84`（仅将绿地事件 ID 更新为 reconnaissance）。
旧 v6 文件已从仓库删除（不保留两份有效基准）。重建命令（注意用绝对
路径，测试 cwd 在 internal/service）：
`SILO40_PARITY_DUMP=<绝对路径> go test ./internal/service/ -run TestParityHarnessDump`。

## 采样跑局暂停令（2026-11-14）

agent-operator 采样跑局曾**暂停**，2026-11-14 恢复条件满足（双方改动均
已落提交；v7 基准已重建）：门控/加权结局一路 6e1d7fb → 18730b1，
agent-operator 一路 a660429 → 46ccae4。

- 恢复后先做 **smoke 采样**（3–5 局，仅验证链路与产率，数据标记 smoke、
  不进训练集）；
- **大规模采样仍不得开始**：信息分层、AP 惩罚细化、IT 幕后指标等机制
  尚在途中，机制定型前铺开采样会随机制一起作废，等作者另行下令。

暂停原因：两个会话曾在同一工作树并行改码，发生过 stash 裹挟对方未提交
改动的事故；且 KI-1/KI-2 未落定前产出的样本与基准都会被作废。

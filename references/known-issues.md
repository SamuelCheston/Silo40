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

---

## KI-3：v7 parity 基准失效，已重建为 v8

`references/parity-v7-three-silo.json`（ec88062 代码树生成，SHA-256
`9f37dfa409a7efcf863819d6231de66042c6f8af3a4d9f742f8fa3e9196b7f84`）已**失效**：
ec88062 之后主线合入大量机制线，结局判定、行动集合、RNG 消费路径与存档结构
均已改变，旧轨迹与当前代码不再可比。机制线清单（`git log --first-parent
--oneline ec88062..origin/main` 归纳）：

- **结局线**：结局链机制基础 G1–G11（dc6be5e，存档 v9）、G12 裁定与事件链
  修复（269e01f）、可玩结局链（db72c01）、**失败结局**（87a964e）、
  reset/fracture/information/exclusionist/agent-compromised/death 各可玩链修复
  （e0340fc、b3bbce1、2bab376、446ad27、ffd3e6e、9bc906b）、1 号反应链可达性
  与主线平局固定序（a379952、05850de，存档 v12）、移除 Silo 1 渗透情报
  operation（c9049d7）。
- **逃出/逃亡线**：escape line（a51f9d4）、REBELLION 逃亡接纳（50d6f49）、
  可玩性修复（e559f49）、正典对齐（d02b866）、怀疑度只由组织行为驱动
  （bc9a785）。
- **谈判**：市长与叛乱方谈判/妥协层，三方带权决策、信道回合（139d63c）。
- **渗透**：时间渗透状态机，部门信念层渗透成熟态，存档 v13→v14（8ce5ef2）。
- **反制**：死亡线远程终止渠道绑定值夜检修（332a12f）、本地整肃后果与名册
  反制（5086a97）、流通摩擦改为按动作分流（T1-2 建联 / T2-3 交接）并补部门
  戒备月度冷却（c87a34e）。
- **其他可玩性/世界机制**：rule-of-law 线（4295916）、个人规则线（2783fea）、
  职业 crossplay（78c8309）、时间线（5fd5c78）、IT 秩序司法附着（55f2d74）、
  前十分钟（19898d6）、堡语音（de37c4d）、外部给养（3ce5521）、遗物目录
  （fb7081f）、剧情抉择兄弟动作互斥（03efc2c）。

**行为差异**：旧基线 6 种子以 INFORMATION×4 / REBELLION×2 胜利（42–81 月）；
新转储 6 种子全部于 37–38 月以 RESET 失败结局终结。parity harness 的固定策略
只做 GatherInfo 与 BuildConnection，不做 reset 反制也不分享情报，因此 RESET
是当前行为的确定性事实，**不是**结局分布验收。因早于第 100 月结局，harness
的第 100 月 persist/Resume 分支本次轨迹未到达（该分支代码仍保留）。

**新基线生成条件**：

- 构建版本：`c87a34e`（转储即在该提交上生成；其后只有引用面文档提交，行为中性）。
- 种子 7–12；三堡场景 = 种子派生随机驻地（2–17）+ Silo 1 + Silo 18，
  全 service 路径、真实内容、脚本动作；按单测惯例不加载生产 action gate
  （旧机制 fixture）。
- harness 版本：含 `gameOver` 守卫（结局后拒绝脚本动作并退出循环）、200 月上限、
  独立 policy RNG（`seed ^ 0x5110`，不消耗游戏流）、第 100 月 persist/Resume
  分支（当前轨迹未到达，见上）。
- 双跑逐字节一致：`cmp` 通过。
- SHA-256：`c63e0a57b9e374611f5b74ae267a8e3b24f18bfd5c6dd0c8b096d3df5de09cfe`
  （两份转储 13588 字节）。

**旧基线废弃理由**：上述机制线改变了结局判定、行动集合与 RNG 消费顺序，
v7 轨迹运行在「无结局链/失败结局/逃出/谈判/时间渗透/反制」的世界定义里，
对当前代码不可比；按纪律不保留两份都宣称有效的基准。`parity-v7-three-silo.json`
已删除，当前唯一有效基准为 `parity-v8-three-silo.json`。

重建命令（绝对路径）：
`SILO40_PARITY_DUMP=<绝对路径> go test ./internal/service -run '^TestParityHarnessDump$' -v`。

**基线口径（[ADR-0017](../knowledge/adr/0017-no-backward-compat-before-release.md)）**：
正式上线前 parity/轨迹基线不承担兼容义务——机制变化时旧基线可直接作废，
重建是求当前正确的手段而非义务；「不保留两份都宣称有效的基准」与台账失效
记录的纪律保留。

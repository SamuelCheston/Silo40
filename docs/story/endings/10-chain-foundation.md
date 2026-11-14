# 结局事件链 · 机制底座与表达约定

> [返回总索引](../silo40-endings.md) · [共同主干](./11-chain-common-trunk.md) · [十线仲裁](./22-chain-cross-arbitration.md)
> 性质：事件链设计，不包含代码、事件文件或数值改动。所有数值只作 sim 校准起点。

## 1. 本卷回答什么

十条结局线的剧情台阶（01–09 分册与好结局分册）回答"剧情上发生什么"；本卷回答"事件机制上如何发生"：每条线从开局到终局依次经过哪些节点、由什么触发、在哪里分岔、十条线在同一局里怎样互斥与交叉。

硬约束：**只使用项目现有的事件机制**。现有机制表达不了的地方，在 [§5 能力缺口清单](#5-能力缺口清单) 登记，不假设它能跑。

## 2. 可用的机制原语（现状事实）

| 原语 | 出处 | 事件链中的用途 |
| --- | --- | --- |
| 事件分组 `histories / special / crisis / rules` | [events/README.md](../../../events/README.md) | 里程碑、条件节点、升级后续、常驻规则 |
| `player_actions/` 文件驱动玩家动作 | 同上（管线就绪、目录未建） | 玩家决断时刻：接触卡、固定选择、步骤行动 |
| 触发器封闭集 | [作者手册 §3](../../research/07-content-authoring-guide/02-format-triggers-effects.md) | timestamp、silo 指标/资源/人口阈值、`silo1_destroyed` flag、`event_triggered`、`player_action_is`、部门恐慌/思潮广度计数、all/any、`script.canTrigger` |
| 效果封闭集 | 作者手册 §4 | 持久杠杆：`rebellion` `history_burden` `panic_value`、资源、人口、碎片收发、楼层稳定度、`schedule_event`；flag 仅 `silo1_destroyed` |
| 行动门控 `action_gates.json` | [mechanics-gating-endings.md](../../../references/mechanics-gating-endings.md) | 阶段（`arrival→first_signal→archive_awakened`）与行动解锁；条件节点支持 `event_completed` `stage_at_least` `metric` `relation` |
| 玩家行动完成键 `player_action:<id>` | 行动设计 15 §5.3 | 链条状态：选择、步骤、结果都以完成键记录，可被 `event_completed` 与 `event_triggered` 消费（已核实 `event_triggered` 按 `category:event_id` 匹配，`player_action` 类别归一化成立） |
| 脚本上下文 `ctx` | [content_events_script.go](../../../internal/engine/content_events_script.go) | `canTrigger/apply` 只读：silo 指标、资源、人口、部门恐慌与派生思潮、派系状态、事件状态 `states`、`world_flags` |
| 结局判定 | `core_systems.js` victory_check + `weighted_endings.go` | 硬失败护栏与历史候选加权；现状候选仅 `INFORMATION / TIME / REBELLION / EXCLUSIONIST` |

## 3. 链条状态的表达约定

内容事件**不能自建 flag**（flag 白名单只有 `silo1_destroyed`）。链条状态一律用以下三种既有载体表达：

1. **事件完成键**：某节点"已经发生"＝对应 `category:event_id` 在 `ContentEventState` 中 `Triggered`。分支选择用不同的玩家行动完成键区分（accept/decline/defer 各一键）。
2. **持久杠杆阈值**：叛乱、历史包袱、恐慌广度、资源、人口、碎片、楼层稳定度。只有持久杠杆能承担"世界真的变了"；派生指标（凝聚力、合法性、部门紧张、部门思潮）每 tick 重设，不得作为链条状态。
3. **阶段与门控**：`arrival / first_signal / archive_awakened` 与 `UnlockedActions`。

**超时与守卫约定**（现状事实：`schedule_event` 到期**无条件派发**，延迟任务不可取消）：

- 守卫不放在触发条件里，放在事件内部：超时事件的 `script.apply` 读取 `ctx.states` 中的完成键，**已完成则返回空效果空转，未完成才结算后果**。
- 超时事件本身无论如何都会点火，因此**不得订阅超时事件作为"失败已发生"的条件**；要区分"超时失败"与"已完成"，读完成键，不读超时事件。
- 需要"失败事实键"时，由超时事件在守卫未通过时 `emit` 一个独立事实事件（如 `special:reset_completed`），订阅者只订阅该事实事件。

**时序纪律**（作者手册 §6）：可玩窗口 Y122M1 → Y130M1（`silo1_destroyed_signal`），约 96 个月。所有节点与延迟落点必须在 Y130 前；延迟发起不晚于 ~Y128；延迟档 2–3 月（短）/ 4–6 月（中）；大事件 ≤1 个/月，循环事件合计 ≤ ~0.55 次/月。

## 4. 与既有事件生态的衔接

事件链不重造已有节点，直接挂接：

| 既有节点 | 时点 | 被哪些线消费 |
| --- | --- | --- |
| `archive_truth_broadcast`（histories） | Y122M1 | 全线开局背景：历史包袱入局 |
| `neighbor_silo_silence` → `network_data_cache` | Y124M1 起 | 《无人裁决》衰亡前兆、《共证》受限信道材料 |
| `silo1_ultimatum` → `ultimatum_aftermath` | Y126M1 起 | 《寂堡》终止征兆、《无人裁决》压力、全线暴露风险 |
| `archive_purge_order` → `purge_backlash` | hb≥0.12 | 《遗忘》缓慢绞杀路径、《共证》宿主威胁 |
| `underground_coordination` | rebellion≥0.62 且 Y126+ | 《越岭》滚雪球、《断层》升级 |
| `silo1_destroyed_signal` | Y130M1 | 《无人裁决》权威覆灭事实、《深闭》死后复制分岔 |

## 5. 能力缺口清单

以下需求**现有机制表达不了**，事件链在对应节点标注缺口编号，不假设已落地：

| # | 缺口 | 性质 |
| --- | --- | --- |
| G1 | `RULE_OF_LAW / PERSONAL_RULE / RESET / FRACTURE` 不在结局候选（现状候选仅四个历史候选加两个硬失败） | 引擎：新结局候选与仲裁 |
| G2 | 公开约章资格供给：`PublicEvidenceQ / IndependentPublicFacts / PactQualified` 无任何内容效果或机制变异可写（Authority 状态无写入入口） | 引擎：Authority 变异或碎片→`Q` 桥接 |
| G3 | 内容触发器与脚本读不到 `W`、Judicial 附着、玩家怀疑/威望（`ctx` 无 Authority 与 actor 字段） | 引擎：ctx 扩展或机制层桥接 |
| G4 | cohort 思潮直写与覆灭前夕快照：内容只有短暂的 `profession_ideology_delta_all`；多轴思潮未实现 | 引擎 |
| G5 | 逃出群体模型与堡外存续结算：人口只有堡内总数，无"已离开且存续"状态 | 引擎/模型 |
| G6 | 内容 → `safeguard_risk` 桥接：内容效果封闭集不含它，只有机制变异层可写 | 引擎 |
| G7 | 跨堡出站信号投递：外溢与外部独立复制的结算需要多堡编排管线（入站 `SignalsReceived` 已有，出站未实现） | 多堡编排 |
| G8 | 事实性尾声标签：主结局之外的尾声（约章已锁、原驻地毁灭等）无展示载体 | 展示层 |
| G9 | `victory_check` 的 REBELLION/TIME 占位语义需按结局体系 12 修正（基础触发只请求终局评估；高叛乱无逃离不等于《越岭》；覆灭须过思潮闸） | 引擎 |
| G10 | `action_gates` 的 `first_signal` 来源迁移：现状仍消费 `player_action:action_gather_info`，行动设计 15 要求改为首份本地 Observation 键 | 实现迁移 |
| G11 | 世界 flag `network_jammed`（终止信号被干扰）已预留但无写入与消费方 | 编排层 |

## 6. 命名与书写纪律

- 事件 `id` 用 snake_case 与文件名一致；新节点按线取前缀（`charter_*` `reset_*` `fracture_*` `termination_*` 等），不与既有 ID 重复。
- 代码注释用英文（仓库约定）；设计文档中的数值一律标注"sim 校准起点"，不作为平衡结论。
- 不新增专属反抗组织，不恢复 40 号指挥链；协作一律走既有部门、派系与联系人。
- 纯调度事件写 `canTrigger: () => false`；不为依赖测试订阅延迟队列派发的事件（作者手册 §12.5）。

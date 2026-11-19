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
| 触发器封闭集 | [作者手册 §3](../../research/07-content-authoring-guide/02-format-triggers-effects.md) | timestamp、silo 指标/资源/人口阈值、`silo1_destroyed` flag、`event_triggered`、`player_action_is`、部门恐慌/思潮广度计数、all/any、`script.canTrigger`；机制底座后 `ctx` 另暴露 `authority`（W/附着/约章）、`actor`（怀疑/威望/宣传）、`safeguard_risk` 与逃出群体状态 |
| 效果封闭集 | 作者手册 §4 | 持久杠杆：`rebellion` `history_burden` `panic_value`、资源、人口、碎片收发、楼层稳定度、`schedule_event`；flag 仅 `silo1_destroyed`；机制底座新增：`authority_evidence_delta / authority_public_fact / authority_pact_qualify`、`cohort_ideology_delta`、`safeguard_risk` 指标、`escape_group_depart/status`；内容事件另已有 `world_flag_set`（现可写 `network_jammed`） |
| 行动门控 `action_gates.json` | [mechanics-gating-endings.md](../../../references/mechanics-gating-endings.md) | 阶段（`arrival→first_signal→archive_awakened`）与行动解锁；条件节点支持 `event_completed` `stage_at_least` `metric` `relation` |
| 玩家行动完成键 `player_action:<id>` | 行动设计 15 §5.3 | 链条状态：选择、步骤、结果都以完成键记录，可被 `event_completed` 与 `event_triggered` 消费（已核实 `event_triggered` 按 `category:event_id` 匹配，`player_action` 类别归一化成立） |
| 脚本上下文 `ctx` | [content_events_script.go](../../../internal/engine/content_events_script.go) | `canTrigger/apply` 只读：silo 指标、资源、人口、部门恐慌与派生思潮、派系状态、事件状态 `states`、`world_flags`；机制底座后另含 `authority`、`actor`、`safeguard_risk`、逃出群体 |
| 结局判定 | `core_systems.js` victory_check + `weighted_endings.go` | 机制底座已实现：硬失败护栏（P0：寂堡→遗忘→断层→除名，先读逃出群体）＋六候选资格池（P1）＋四类主体加权（P2）＋同分顺序（P3）；权重为 TODO(balance) 占位 |

## 3. 链条状态的表达约定

内容事件**不能自建 flag**（flag 白名单只有 `silo1_destroyed`）。链条状态一律用以下三种既有载体表达：

1. **事件完成键**：某节点"已经发生"＝对应 `category:event_id` 在 `ContentEventState` 中 `Triggered`。分支选择用不同的玩家行动完成键区分（accept/decline/defer 各一键）。
2. **持久杠杆阈值**：叛乱、历史包袱、恐慌广度、资源、人口、碎片、楼层稳定度、合法性（市长个人正统性）。只有持久杠杆能承担"世界真的变了"；派生指标（凝聚力、部门紧张、部门思潮）每 tick 重设，不得作为链条状态。
3. **阶段与门控**：`arrival / first_signal / archive_awakened` 与 `UnlockedActions`。

**超时与守卫约定**（现状事实：`schedule_event` 到期**无条件派发**，延迟任务不可取消）：

- 守卫不放在触发条件里，放在事件内部：超时事件的 `script.apply` 读取 `ctx.states` 中的完成键，**已完成则返回空效果空转，未完成才结算后果**。
- 超时事件本身无论如何都会点火，因此**不得订阅超时事件作为"失败已发生"的条件**；要区分"超时失败"与"已完成"，读完成键，不读超时事件。
- 需要"失败事实键"时，由超时事件在守卫未通过时 `emit` 一个独立事实事件（如 `special:reset_completed`），订阅者只订阅该事实事件。

**时序纪律**（用户裁决口径；作者手册 §6 的旧窗口约束以此为准）：

- **相对历**：以开局为 Y0，不再对齐原著绝对年份；2301→2345 的四十余年只存在于前史，不在游戏中走过。相对年与现行代码时间戳的映射为 Y0=代码 Y122 … Y8=代码 Y130；代码迁移属实现事项，本轮不改代码。
- **可玩窗口**：Y0M1 开局，约 96 个月。窗口以三个剧情锚点定节奏：**朱丽叶当上警长是中期事件，当选市长是末期事件，覆灭是结算事件**（锚点取 18 号正典链进度；现行链上"当选市长"对应 N5=相对 Y3M1，"覆灭结算"对应 N9=相对 Y8M1，"警长上任"当前链上无独立节点，属实现缺口）。
- **覆灭是结算动作，不是有日期的世界事件**：窗口走完或结算条件满足，即视为覆灭发生，直接进仲裁。原固定年份覆灭信号（`silo1_destroyed_signal`／N9）降级为**结算触发器**，不再兼任剧情节点。**覆灭后不留内容窗口**：仲裁紧随结算，不安排覆灭之后的接管、余波等内容环节；一切需要在覆灭前完成的事，必须在结算前完成。
- **延迟链允许溢出**：不强制收进窗口内。事件链可以按自身情况自然触发，也可以按年份强制推进。**必须在窗口内发生的特殊事件设兜底**：到指定时间若还没被事件链唤起，就强制拉起（兜底节点仍须在结算前）。覆灭结算仍是终局时钟：落到结算之后的延迟在本局不再有结算意义。
- 密度预算不变：大事件 ≤1 个/月，循环事件合计 ≤ ~0.55 次/月；延迟档 2–3 月（短）/ 4–6 月（中）。

## 4. 与既有事件生态的衔接

事件链不重造已有节点，直接挂接：

| 既有节点 | 时点（相对历） | 被哪些线消费 |
| --- | --- | --- |
| `archive_truth_broadcast`（histories） | Y0M1 | 全线开局背景：历史包袱入局 |
| `neighbor_silo_silence` → `network_data_cache` | Y2M1 起 | 《无人裁决》衰亡前兆、《共证》受限信道材料 |
| `silo1_ultimatum` → `ultimatum_aftermath` | Y4M1 起 | 《寂堡》终止征兆、《无人裁决》压力、全线暴露风险 |
| `archive_purge_order` → `purge_backlash` | hb≥0.12 | 《遗忘》缓慢绞杀路径、《共证》宿主威胁 |
| `underground_coordination` | rebellion≥0.62 且 Y4+ | 《越岭》滚雪球、《断层》升级 |
| `silo1_destroyed_signal` | Y8M1（窗口走完） | **结算触发器**：置 `silo1_destroyed` flag 即进仲裁，不再是剧情节点 |

## 5. 能力缺口清单

机制底座（`dc6be5e`，save v9）已落地原 G1–G11 的机制部分：六候选 P0→P3 仲裁、覆灭前夕快照、逃出群体最小状态、内容→Authority/cohort/`safeguard_risk` 写入通路、跨堡出站信号日志、`network_jammed` 消费、`first_signal` 迁移。下表是**仍然存在的缺口**：

| # | 缺口 | 性质 |
| --- | --- | --- |
| R1 | **逃离模型未实装（用户裁决，本轮不做，已登 roadmap）**：`EscapedGroup` 最小状态基础已有，但路线事实、分批撤离与堡外存续的完整结算需要单独设计；依赖它的链条节点只预留注释 | 单独设计轮次 |
| R2 | **外溢准备工作未实装（用户裁决，本轮不做，已登 roadmap）**：外溢不进胜负判定、只作评分项（仲裁已按此实现）；受限渠道、外部接收与独立复制结算需要单独设计 | 单独设计轮次 |
| G2' | 公开约章资格的**具体供给事件**未编写：写入通路（`authority_evidence_delta / authority_public_fact / authority_pact_qualify`）已有，供给内容待写 | 内容编写 |
| G8 | 事实性尾声标签：主结局之外的尾声（约章已锁、原驻地毁灭、外溢评分等）无展示载体 | 展示层 |
| G12 | 【裁决对齐】`PERSONAL_RULE` 资格当前读 `Judicial.Regime == "MAYOR"`（裁决前口径）；按裁决应改为警察服从＋市长正统性（司法部只附着 IT 或约章），结语文案同步 | 引擎对齐 |
| G13 | 相对历呈现迁移：代码时间戳仍为 Y122/Y130，文档口径已改相对历（Y0/Y8）；呈现层迁移属实现事项 | 实现迁移 |
| — | 十条结局链的**内容事件节点群大部分尚未编写**；仲裁只消费其完成键/事实键。已落地的内容节点：Silo 1 反应链（`termination_channel_test / termination_channel_disrupted / termination_release / strike_reconnaissance / bombing_terminal`、`silo1_purge_omen / silo1_purge_exposed / silo1_purge_execution`）。权重系数全部为 TODO(balance) 占位 | 内容编写 |

## 6. 命名与书写纪律

- 事件 `id` 用 snake_case 与文件名一致；新节点按线取前缀（`charter_*` `reset_*` `fracture_*` `termination_*` 等），不与既有 ID 重复。
- 代码注释用英文（仓库约定）；设计文档中的数值一律标注"sim 校准起点"，不作为平衡结论。
- 不新增专属反抗组织，不恢复 40 号指挥链；协作一律走既有部门、派系与联系人。
- 纯调度事件写 `canTrigger: () => false`；不为依赖测试订阅延迟队列派发的事件（作者手册 §12.5）。

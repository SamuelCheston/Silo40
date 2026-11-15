# 《无人裁决》事件链 · 字段级事件规格

> [返回总索引](../silo40-endings.md) · [节点设计](./33-time-nodes.md) · [作者手册](../../research/07-content-authoring-guide.md)
> 性质：**实装规格**。沿用 [27《公议》](./27-rule-of-law-event-spec.md)／[31《人治》](./31-personal-rule-event-spec.md) 的格式与纪律：数值一律 `TODO(balance)`；依赖未落地机制的节点只留注释；功能位、无人名。

## 0. 通用约定

- **目录与命名**：里程碑与抉择时刻入 `events/special/`，三选择入 `events/player_actions/`。文件名＝事件 `id`，snake_case，本线前缀 `open_discourse_*`／`pre_fall_*`。
- **完成键**：事件节点为 `special:<id>`；玩家动作为 `player_actions:<事件 id>`（内容完成键按 `组:事件 ID` 命名）。一个键只记事实。
- **兜底**：`deadline_year`／`deadline_month` 字段（硬伤 #3）：到期未触发则强制拉起一次（绕过 trigger 与 canTrigger）。相对 Y7M10 = 代码 `129/10`。注意：服务层 DB 同步路径曾丢失这两个字段（引擎有、生产装载没有），本批已修（模型加列、同步/加载携带）；此前一切依赖兜底的实装在生产路径上都不生效。
- **布线事实**（与 27/31 号相同）：完成键发总线事件；`event_triggered` 参与月度持续评估；订阅既有动作的节点放 `special/`。
- **快照纪律**：资格读引擎 `PreFallSnapshot`（`silo1_destroyed` 首次置真时截取，cohort 人口加权真源）；内容不重算思潮、不在覆灭后追加内容窗口。`cohort_ideology_delta` 只用于抉择的即时效果（定格前），不用于"补快照"。
- **现有 DSL 盘点**：触发器含 `timestamp_at_or_after / silo_metric_gte|lte / silo_flag_true|false / event_triggered / player_action_is / profession_metric_gte_count / profession_ideology_gte_count / all / any / canTrigger`；效果含 `silo_metric_delta`、`profession_metric_delta_all`、`floor_stability_delta`、`cohort_ideology_delta`、`schedule_event`、`player_action_complete` 等（封闭集见作者手册 §4）。
- **ctx 可读**：`silo_metrics`、`professions[].ideologies`（派生视图）、`states`、`authority`、`year/month`。快照本身在定格前不存在、定格后不参与内容触发（结算紧随覆灭，无内容窗口）。

## 1. 节点规格

### A-1 open_discourse_milestone（开放表达里程碑）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/open_discourse_milestone.js` |
| 触发条件 | `professionIdeologyGteCount("pro_foreign", 0.55, 2)`（`TODO(balance)`）；`fire_mode: "REPEATABLE"`，`cooldown_months: 6`（`TODO(balance)`） |
| 效果 | `siloMetricDelta("history_burden", +0.02)`（`TODO(balance)`） |
| 完成键 | `special:open_discourse_milestone`（计数键） |
| DSL 能力 | 全部现有（`profession_ideology_gte_count` 读 cohort 派生视图） |
| 依赖 | 无 |

### T-1 pre_fall_choice（覆灭前夕抉择时刻）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/pre_fall_choice.js` |
| 触发条件 | `all(timestampAtOrAfter(129,1), eventTriggered("special","open_discourse_milestone"))`（相对 Y7 起，`TODO(balance)`）；`fire_mode: "ONCE"` |
| 兜底 | `deadline_year: 129`、`deadline_month: 10`（相对 Y7M10，`TODO(balance)`）：事件链未唤起时强制拉起——覆灭前夕的抉择不可被前置时序杀死 |
| 效果 | 无机制效果：`script.apply` 返回 `{ effects: [], emit: [] }`（纯时刻节点） |
| 完成键 | `special:pre_fall_choice` |
| DSL 能力 | 全部现有（deadline 字段已落地） |
| 依赖 | 无 |

### T-2 pre_fall_loyalty_mobilization（忠诚动员）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/pre_fall_loyalty_mobilization.js` |
| 动作规格 | `id: "PRE_FALL_LOYALTY_MOBILIZATION"`、`PLAYER_EVENT`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 0`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | `playerActionIs("PRE_FALL_LOYALTY_MOBILIZATION")`；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("rebellion", -0.05)`；`professionMetricDeltaAll("panic_value", -0.04)`；`cohortIdeologyDelta("loyalty", +0.10)`（profession 空＝全部 cohort；均 `TODO(balance)`） |
| 资格后果 | 快照倾向纯种：《无人裁决》绝对失格（由快照判定，不由本键判定） |
| 完成键 | `player_actions:pre_fall_loyalty_mobilization` |
| 门控 | `unlock: event_completed "special:pre_fall_choice"` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### T-2 pre_fall_total_riot（全面暴乱）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/pre_fall_total_riot.js` |
| 动作规格 | `id: "PRE_FALL_TOTAL_RIOT"`，其余同上 |
| 触发条件 | `playerActionIs("PRE_FALL_TOTAL_RIOT")`；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("rebellion", +0.15)`；`floorStabilityDelta`（zone 空＝全部楼层，`-0.1`）（均 `TODO(balance)`） |
| 资格后果 | 维存受损：社会维存不可逆崩溃且无逃出群体时入《断层》判定（既有口径），否则继续模拟至覆灭结算 |
| 完成键 | `player_actions:pre_fall_total_riot` |
| 门控 | `unlock: event_completed "special:pre_fall_choice"` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### T-2 pre_fall_open_inquiry（维持公开质询）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/pre_fall_open_inquiry.js` |
| 动作规格 | `id: "PRE_FALL_OPEN_INQUIRY"`，其余同上 |
| 触发条件 | `playerActionIs("PRE_FALL_OPEN_INQUIRY")`；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("history_burden", +0.03)`（`TODO(balance)`） |
| 资格后果 | 开放快照合格的叙事前提；实际资格由快照判定，不由本键授予 |
| 完成键 | `player_actions:pre_fall_open_inquiry` |
| 门控 | `unlock: event_completed "special:pre_fall_choice"` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

## 2. 清单1：可直接写的节点

不依赖新机制，全部现有能力：

| 节点 | 文件 | 备注 |
| --- | --- | --- |
| A-1 open_discourse_milestone | `events/special/open_discourse_milestone.js` | 全部现有能力 |
| T-1 pre_fall_choice | `events/special/pre_fall_choice.js` | deadline 兜底已落地 |
| T-2 三选一 | `events/player_actions/pre_fall_{loyalty_mobilization,total_riot,open_inquiry}.js` | 兄弟选择隐藏是已知缺口 |
| 门控条目 | `events/action_gates.json` 追加 3 条 | 三选择同解 |

## 3. 清单2：必须等机制的节点

| 节点/能力 | 等的机制 | 未到位前的处理 |
| --- | --- | --- |
| 终止来路的反制绑定 | 已落地：`termination_release` 读 `network_jammed` 守卫，玩家动作 `TERMINATION_CHANNEL_JAM` 调度 `termination_channel_disrupted` | — |
| 整肃来路的反制绑定 | 已落地：名册调换动作 `PURGE_ROSTER_TAMPER` → `silo1_purge_softened`；执行节点按名册是否被动过分档（人口/历史包袱/凝聚/恐慌） | — |
| 三选一兄弟选择隐藏 | 门控否定谓词 | 三键同解，下游只读各键事实（与《公议》《人治》共用缺口） |
| 覆灭的更早来路（1 号正典链提前覆灭） | Silo 18 正典链进度节点（"警长上任"等实现缺口，机制底座 §3 已登记） | 本线只消费 `silo1_destroyed` 事实，不关心其来路时序 |

## 4. 缺口汇总（给机制批次）

1. 反制绑定（终止／整肃两路）：等"日常动作中的反制"批次。
2. 门控否定谓词：三选一互斥呈现（三线共用缺口）。
3. 覆灭来路的提前化：1 号正典链进度节点落地前，覆灭只有结算时钟一条实际来路；终止/物理来路以 DEATH 事实形式存在（威胁成立，先于结算）。

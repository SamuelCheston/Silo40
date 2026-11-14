# 《公议》事件链 · 字段级事件规格

> [返回总索引](../silo40-endings.md) · [节点设计](./26-rule-of-law-nodes.md) · [作者手册](../../research/07-content-authoring-guide.md)
> 性质：**实装规格**。沿用 [24《共证》事件规格](./24-information-event-spec.md) 的格式与纪律：只写规格，不动 `events/` 下任何实际文件与引擎代码；数值一律 `TODO(balance)`；依赖未落地机制的节点只留注释；人名用功能位。

## 0. 通用约定

- **目录与命名**：条件/里程碑节点入 `events/special/`，守卫超时入 `events/crisis/`，接触卡与三选择入 `events/player_actions/`。文件名＝事件 `id`，snake_case，本线前缀 `order_chain_*`／`charter_*`（机制底座 §6 命名纪律）。
- **完成键**：事件节点为 `special:<id>`／`crisis:<id>`；玩家动作为 `player_action:<action_id>`。一个键只记事实，不预写路线归属。共享键不重造：法警证据链（`bailiff_*`）与两份核心事实（`Judicial_1`／`Judicial_2`）跨线消费。
- **fire_mode**：里程碑 ONCE；起草步骤 REPEATABLE＋`cooldown_months`。
- **守卫**：纯调度事件写 `canTrigger: () => false`；守卫逻辑放 `apply` 内读 `ctx.states`；超时事件不作失败条件被订阅（机制底座 §3）。
- **关键布线事实**（与 24 号规格时点的差异，均已落地）：完成键写入即发总线事件（硬伤 #1 已修）；带 `event_triggered` 的定义**参与月度持续评估**（硬伤 #1/#2 修复，注释明引 charter_lock_day 前置时序为动机）；`player_actions/` 组事件在动作派发时点火且**会接管该次动作**——订阅既有动作（Share Info）的节点必须放 `special/`，不得放 `player_actions/`，否则宿主动作结算被吞。
- **现有 DSL 盘点**（实装时以此为准）：触发器含 `timestamp_at_or_after / silo_metric_gte|lte / silo_flag_true|false / world_flag_true|false / event_triggered / player_action_is / profession_metric_gte_count / profession_ideology_gte_count / all / any / canTrigger`；效果含 `silo_metric_delta`（持久杠杆）、`profession_metric_delta_all`（panic_value）、`profession_fragment_add|remove`、`schedule_event`、`belief_observe`、`public_evidence_register`、`authority_evidence_delta / authority_public_fact / authority_pact_qualify`、`player_action_complete`、`world_flag_set`、`cross_signal`。
- **ctx 可读**：`authority`（`w / police_obedience / regime / pact_qualified / pact_locked / public_evidence_q / independent_public_facts / a_it / a_pact`）、`public_evidence`（已登记证据 ID 表）、`actor.beliefs`、`states`（完成键状态含 `trigger_count`）、silo 指标、派系。**不可读**：部门信念层、暴露高水位 E、引擎配置阈值。
- **门控**（`action_gates.json`）：条件节点支持 `event_completed / stage_at_least / metric / relation` 与 `all/any` 递归；**无否定谓词**（见 T-2 缺口）。

## 1. 节点规格

### B-1 order_chain_breach（规则断点）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/order_chain_breach.js` |
| 触发条件 | `all(eventTriggered("player_action","action_share_info"), siloMetricGte("history_burden", 0.10))`；`fire_mode: "ONCE"` |
| 效果 | `professionFragmentAdd("Judicial")`；`siloMetricDelta("history_burden", +0.02)`（`TODO(balance)`） |
| 完成键 | `special:order_chain_breach` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### T-1 charter_contact_offer（接触卡）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/charter_contact_offer.js` |
| 动作规格 | `id: "CHARTER_CONTACT_OFFER"`、`action_type: "PLAYER_EVENT"`、`scope: "common"`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 1`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | `playerActionIs("CHARTER_CONTACT_OFFER")`；`fire_mode: "ONCE"` |
| 效果 | 无机制效果：`script.apply` 返回 `{ effects: [], emit: [] }`（纯接触节点） |
| 完成键 | `player_action:charter_contact_offer` |
| 门控 | `unlock: all(event_completed "special:order_chain_breach", relation Judicial min 0.2)`（`TODO(balance)`）；`available: {}` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### T-2 charter_contact_accept / decline / defer（三选一）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/player_actions/charter_contact_accept.js`／`charter_contact_decline.js`／`charter_contact_defer.js` |
| 动作规格 | 各为 `PLAYER_EVENT`、`target_type: "NONE"`、`unavailable_behavior: "hide"`；`ap_cost: 0`、`duration_months: 0`（`TODO(balance)`） |
| 触发条件 | 各自 `playerActionIs(<动作 ID>)`；`fire_mode: "ONCE"` |
| 接受效果 | `scheduleEvent("charter_window_expiry", 6)`（`TODO(balance)`） |
| 拒绝/暂缓效果 | 无机制效果（`apply` 返回空）；注释指依赖：材料转喂 Mayor 圈层属《人治》节点群（未编写） |
| 完成键 | `player_action:charter_contact_accept`／`..._decline`／`..._defer` |
| 门控 | 三键同解：`unlock: event_completed "player_actions:charter_contact_offer"`（内容完成键按 `组:事件 ID` 命名，注意与内置动作键 `player_action:*` 的区别） |
| DSL 能力 | 全部现有 |
| 依赖 | **缺口**：门控无否定谓词，选定其一后兄弟选择不能隐藏；下游只读接受键，多选不构成机制矛盾。隐藏能力等门控批次，届时在 `available` 加"未选其他键"条件 |

### D-1 charter_draft_step（起草步骤心跳）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/charter_draft_step.js`（**必须放 special/**：订阅完成键、不拦截 Share Info 结算；布线事实见 §0） |
| 触发条件 | `all(eventTriggered("player_action","charter_contact_accept"), eventTriggered("player_action","action_share_info"))`；`fire_mode: "REPEATABLE"`，`cooldown_months: 2`（`TODO(balance)`） |
| 效果 | `apply` 按 `ctx.runtime.trigger_count` 奇偶轮换事实 ID（奇→`Judicial_1`，偶→`Judicial_2`，确定性），`beliefObserve("Judicial", <事实>, 0.5)`（`TODO(balance)`）：制度内喂证深化司法层信念，是本线的速率杠杆 |
| 完成键 | `special:charter_draft_step`（计数键；步骤档由 `trigger_count` 表达） |
| DSL 能力 | 全部现有 |
| 依赖 | 无。心跳语义与 `replication_progress` 同型：两完成键持久存在即按冷却推进，不要求"接受后的新分享"——若未来要精确计数新分享，需机制批次提供时序比较 |

### D-2 charter_draft_convened（起草会议召开）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/charter_draft_convened.js` |
| 触发条件 | `eventTriggered("special","charter_draft_step")` ＋ `canTrigger`：`ctx.states["special:charter_draft_step"].trigger_count >= 2`（`TODO(balance)`）；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("history_burden", +0.03)`（`TODO(balance)`） |
| 完成键 | `special:charter_draft_convened` |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

### K-1 charter_lock_day（锁定日）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/special/charter_lock_day.js` |
| 触发条件 | `all(eventTriggered("special","charter_draft_convened"), timestampAtOrAfter(127,1))`（相对 Y5＝代码 127，`TODO(balance)`）＋ `canTrigger` 读锁定窗：`ctx.authority.w <= 18 && ctx.authority.public_evidence_q >= 0.85 && ctx.authority.independent_public_facts >= 2`（`TODO(balance)`，三值镜像引擎 `PactOpenW / PactMinQ / PactMinFacts`）；`fire_mode: "ONCE"` |
| 效果 | `siloMetricDelta("rebellion", -0.04)`；`professionMetricDeltaAll("panic_value", -0.04)`（`TODO(balance)`） |
| 完成键 | `special:charter_lock_day` |
| DSL 能力 | 全部现有（`ctx.authority` 已暴露） |
| 依赖 | **依赖并行轨**：`W ≤ 18` 的达成当前靠暴露高水位 E 的自然累积；W 离散负冲击通道（1 号覆灭／终止链失效／大规模公开证据）落地后锁定窗才有稳定打开路径。内容不写 W、不自造 W 判定，节点在窗外等待 |
| 备注 | 锁定事实（`PactLocked`）由引擎动力学写入，不是本节点效果；结局资格读引擎事实，不读本完成键 |

### K-2 charter_window_expiry（守卫超时）

| 字段 | 规格 |
| --- | --- |
| 事件文件 | `events/crisis/charter_window_expiry.js` |
| 触发条件 | `script.canTrigger: () => false`（纯调度）；由 T-2 接受键 `scheduleEvent(+6 月)` 唤起，到期无条件派发 |
| 守卫逻辑 | `apply` 读 `ctx.states["special:charter_lock_day"].triggered`：已触发 → 返回空效果空转；未触发 → 未锁定分支 |
| 未锁定效果 | `siloMetricDelta("rebellion", +0.04)`；`professionMetricDeltaAll("panic_value", +0.04)`（`TODO(balance)`） |
| 完成键 | `crisis:charter_window_expiry`（不被订阅为失败条件） |
| DSL 能力 | 全部现有 |
| 依赖 | 无 |

## 2. 清单1：可直接写的节点

不依赖新机制，全部现有能力（实装时按本规格填值）：

| 节点 | 文件 | 备注 |
| --- | --- | --- |
| B-1 order_chain_breach | `events/special/order_chain_breach.js` | 全部现有能力 |
| T-1 charter_contact_offer | `events/player_actions/charter_contact_offer.js` | 接触卡管线就绪 |
| T-2 三选一 | `events/player_actions/charter_contact_{accept,decline,defer}.js` | 兄弟选择隐藏是已知缺口，不阻塞 |
| D-1 charter_draft_step | `events/special/charter_draft_step.js` | 心跳语义同 replication_progress |
| D-2 charter_draft_convened | `events/special/charter_draft_convened.js` | 全部现有能力 |
| K-1 charter_lock_day | `events/special/charter_lock_day.js` | 触发全现有；锁定窗能否打开受 W 冲击通道依赖影响，不影响本文件实装 |
| K-2 charter_window_expiry | `events/crisis/charter_window_expiry.js` | 守卫范式同机制底座 §3 |
| 门控条目 | `events/action_gates.json` 追加 4 条 | offer＋三选择 |

## 3. 清单2：必须等机制的节点

| 节点/能力 | 等的机制 | 未到位前的处理 |
| --- | --- | --- |
| K-1 锁定窗的稳定打开 | **W 离散负冲击通道**（并行轨：1 号覆灭／终止链失效／大规模公开证据三类触发） | 内容按现有接口写（读 `ctx.authority.w`），文档标依赖；不自造 W 判定，不用内容效果写 W |
| T-2 兄弟选择隐藏 | 门控条件否定谓词（"未发生"） | 三键同解，下游只读接受键 |
| D-1 精确计数"接受后的新分享" | 完成键时序比较能力 | 用心跳语义（冷却推进），与 replication_progress 一致 |
| 拒绝/暂缓后的转线路径 | 《人治》节点群（未编写） | 事件内留注释指依赖 |
| 供给侧接口换代 | 并行轨把 `bailiff_evidence_public_*` 换为玩家发布动作 | 本线读登记状态（`ctx.public_evidence`／`ctx.authority`），不订阅供给事件键，换代免疫 |

## 4. 缺口汇总（给机制批次）

1. W 离散负冲击通道（并行轨在做）：落地前 `W ≤ 18` 只能靠 E 自然累积，锁定窗打开不可保。
2. 门控否定谓词：接触卡三选一的互斥呈现。
3. 完成键时序比较：心跳节点改精确计数（可选，低优先）。

# 文件驱动事件

当前所有文件驱动事件都收敛到 `events/` 根目录下，并按子目录分组。当前主格式为 `JavaScript`，每个事件文件使用 `defineEvent({...})` 导出一个事件定义。

当前约定的分组包括：

- `events/histories/`: 历史事件
- `events/special/`: 条件性特殊事件
- `events/crisis/`: 危机事件
- `events/player_actions/`: 玩家操作事件（管线与内容均已启用：拆台路线的调查/
  执行失败/核验/发布动作，见 10-it-control-credibility §7.4）

后端会自动扫描 `events/` 下的一级子目录作为事件分组；旧的平铺布局仅作为兼容读取保留。

运行时约定：

- 所有文件驱动事件都会先转换成一个带名字的 `EventBus` 事件，再由对应 handler 应用效果。
- 推荐使用 `category:event_id` 作为运行时事件名，例如 `special:history_burden_awakened`。
- `crisis` 事件是 `special` 事件的后续事件，这是一条约定。实现上通常通过 `event_triggered` 监听上游 `special` 事件名来唤起。

当前支持的顶层字段：

- `id`: 事件唯一 ID
- `title`: 事件标题
- `description`: 事件描述
- `type`: 事件类型，例如 `SOCIAL` / `TECHNICAL` / `EXTERNAL`
- `fire_mode`: `ONCE` 或 `REPEATABLE`
- `cooldown_months`: 可重复事件的冷却月数（可选）
- `deadline_year` / `deadline_month`: 到点强制拉起兜底（可选）。到该日历时点仍
  未触发过的事件在月度评估里无条件点火一次（绕过 `trigger` 与
  `script.canTrigger`）。只用于剧情上必须发生的窗口内事件（如
  `charter_lock_day`），防止前置早于时点完成时被窗口超时先杀线
- `trigger`: 结构化触发条件
- `effects`: 结构化事件效果列表
- `player_action`: 仅用于 `events/player_actions/`，声明该动作在 UI 中如何展示与提交
- `script`: 可选的受限脚本 hook 容器，当前支持：
  - `canTrigger(ctx) => boolean`
  - `apply(ctx) => { effects?: ContentEffect[], emit?: string[] }`

最小示例：

```js
defineEvent({
  id: "silo1_fallout",
  title: "一号地堡失联余波",
  type: "EXTERNAL",
  fire_mode: "ONCE",
  trigger: all(
    siloFlagTrue("silo1_destroyed"),
    siloMetricGte("cohesion", 0.31),
  ),
  effects: [
    siloMetricDelta("cohesion", -0.05),
    professionMetricDeltaAll("panic_value", 0.05),
  ],
})
```

`script` hook 约束：

- `script.canTrigger(ctx)` 只负责额外判断，返回 `boolean`
- `script.apply(ctx)` 不能直接修改 Go 状态；它只能返回额外 `effects` 和 `emit`
- `emit` 里的事件名必须是运行时事件名，例如 `special:history_burden_awakened`
- 推荐优先使用结构化 `trigger/effects`；只有复杂条件分支和后续链路才进入 `script`

当前支持的 `trigger.type`：

- `timestamp_at_or_after`
- `silo_metric_gte`
- `silo_metric_lte`
- `silo_resource_gte`（无 JS helper，见下方"裸对象触发器"）
- `silo_resource_lte`（无 JS helper，见下方"裸对象触发器"）
- `silo_population_gte`（无 JS helper，见下方"裸对象触发器"）
- `silo_population_lte`（无 JS helper，见下方"裸对象触发器"）
- `silo_flag_true`
- `silo_flag_false`
- `event_triggered`
- `player_action_is`
- `profession_metric_gte_count`
- `profession_ideology_gte_count`
- `all`
- `any`

裸对象触发器：

- 引擎（`internal/engine/content_events.go` 的 `validateTrigger` / `evaluateContentTrigger`）
  支持 `silo_resource_gte|lte` 与 `silo_population_gte|lte`，但 JS prelude 没有为它们
  提供 helper 函数。事件文件可直接写裸对象，与引擎支持的类型一一对应：

```js
trigger: { type: "silo_resource_lte", metric: "Supplies", value: 3000 },
trigger: { type: "silo_population_lte", value: 6000 },
```

新增触发器说明：

- `event_triggered`: 用于 `crisis` 等升级事件，要求某个上游事件已经在本局内触发过。常用字段：
  - `category`: 上游事件分类，例如 `special`
  - `event_id`: 上游事件 ID
  - 评估通道：上游事件点火时的总线唤醒（即时链）＋月度通道的持续评估（完成键
    状态持久存在）。含 `event_triggered` 的组合不再被月度通道排除，`any()` 里
    的数值分支可独立点火；内建动作完成键（`player_action:*`、
    `observation:first_local`）与 `player_action_complete` 效果写入的完成键都会
    发布同名总线事件唤醒后续链
- `player_action_is`: 用于 `player_actions` 目录，要求当前玩家执行的 action 与指定值一致。常用字段：
  - `action`: 可以是内建动作名，例如 `CONDUCT_PROPAGANDA`，也可以是文件驱动动作自己的 `player_action.id`

`player_action` 元数据当前支持的字段：

- `id`: 动作 ID；默认使用事件 `id`
- `label`: 按钮标题；默认使用事件 `title`
- `description`: 按钮说明；默认使用事件 `description`
- `scope`: `common` / `profession` / `profession_group` / `faction_member` / `faction_leader`
- `profession`: 当 `scope=profession` 时必填
- `profession_group`: 当 `scope=profession_group` 时必填；当前复用职业的 `ClassType`
- `action_type`: 前端提交的动作类型；默认是 `PLAYER_EVENT`
- `target_type`: `NONE` / `DEPT` / `RESOURCE`
- `ap_cost`: 动作 AP 消耗
- `duration_months`: 动作执行后推进的月数
- `unavailable_behavior`: `hide` / `disable`

运行时展示约定：

- 后端会把当前可展示的动作统一整理成 `available_actions`
- 当前阶段会把静态内建动作和 `player_actions` 文件驱动动作合并后一起返回
- `faction_member` 作用域的玩家动作暂不启用；`faction_leader` 依据当前 `AgentStats.is_faction_leader` 判定

当前支持的 `effects.type`：

- `silo_metric_delta`
- `silo_resource_delta`
- `silo_population_delta`
- `profession_metric_delta_all`
- `profession_ideology_delta_all`
- `silo_flag_set`（仅 `silo1_destroyed`）
- `profession_fragment_add`（`profession` 空 = 补给碎片最少的部门；`fragment` 空 = 随机发一个该部门未知碎片）
- `profession_fragment_remove`（必须同时给 `profession` 与 `fragment`）
- `floor_stability_delta`（`zone` 空 = 全部楼层；否则按 `Upper`/`Mid`/`Lower` 过滤；`value` 为稳定度增量，0–1 截断）
- `schedule_event`（`event_id` + `delay_months > 0`）
- `player_action_complete`（`action` = 动作 ID；写入 `player_action:<action>`
  完成键并发同名总线事件，供 `event_triggered` 与行动门控读取。JS helper：
  `completePlayerAction(actionId)`）
- `belief_observe`（`observer` = `"player"` / `"residents"` / 部门名；
  `fragment` = 事实 ID；`reliability` = 渠道可靠度。向观察者信念层注入一次
  观察，走统一的吸收公式。JS helper：`beliefObserve(observer, fragment, reliability)`）
- `public_evidence_register`（`fragment` = 事实 ID；`provenance` / `coverage`
  ∈ (0,1]。登记公开证据；`Q` 与独立公开事实资格只读登记。JS helper：
  `publicEvidenceRegister(fragment, provenance, coverage)`）
- `evidence_verify`（`fragment` = 事实 ID。把事实记入已核验名单，是
  provenance=1.0 发布的前置。JS helper：`evidenceVerify(fragment)`）
- `authority_w_shock`（`value` = 冲击值。进入威势冲击队列 Σδe，下一次月度
  权威更新消费一次，单月合计 cap ±12。JS helper：`authorityWShock(value)`）

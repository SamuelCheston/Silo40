# 文件驱动事件

当前所有文件驱动事件都收敛到 `events/` 根目录下，并按子目录分组。

当前约定的分组包括：

- `events/histories/`: 历史事件
- `events/special/`: 条件性特殊事件
- `events/crisis/`: 危机事件
- `events/player_actions/`: 玩家操作事件

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
- `trigger`: 结构化触发条件
- `effects`: 结构化事件效果列表
- `player_action`: 仅用于 `events/player_actions/`，声明该动作在 UI 中如何展示与提交

当前支持的 `trigger.type`：

- `timestamp_at_or_after`
- `silo_metric_gte`
- `silo_metric_lte`
- `silo_flag_true`
- `silo_flag_false`
- `event_triggered`
- `player_action_is`
- `profession_metric_gte_count`
- `profession_ideology_gte_count`
- `all`
- `any`

新增触发器说明：

- `event_triggered`: 用于 `crisis` 等升级事件，要求某个上游事件已经在本局内触发过。常用字段：
  - `category`: 上游事件分类，例如 `special`
  - `event_id`: 上游事件 ID
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
- `profession_metric_delta_all`
- `profession_ideology_delta_all`
- `silo_flag_set`

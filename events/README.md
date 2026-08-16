# 文件驱动事件

当前所有文件驱动事件都收敛到 `events/` 根目录下，并按子目录分组。

当前约定的分组包括：

- `events/histories/`: 历史事件
- `events/special/`: 条件性特殊事件
- `events/crisis/`: 危机事件
- `events/player_actions/`: 玩家操作事件

后端会自动扫描 `events/` 下的一级子目录作为事件分组；旧的平铺布局仅作为兼容读取保留。

当前支持的顶层字段：

- `id`: 事件唯一 ID
- `title`: 事件标题
- `description`: 事件描述
- `type`: 事件类型，例如 `SOCIAL` / `TECHNICAL` / `EXTERNAL`
- `fire_mode`: `ONCE` 或 `REPEATABLE`
- `cooldown_months`: 可重复事件的冷却月数（可选）
- `trigger`: 结构化触发条件
- `effects`: 结构化事件效果列表

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
  - `action`: 例如 `CONDUCT_PROPAGANDA`

当前支持的 `effects.type`：

- `silo_metric_delta`
- `profession_metric_delta_all`
- `profession_ideology_delta_all`
- `silo_flag_set`

# 文件驱动事件

`events/` 与 `histories/` 目录都使用 JSON 事件定义。

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
- `profession_metric_gte_count`
- `profession_ideology_gte_count`
- `all`
- `any`

当前支持的 `effects.type`：

- `silo_metric_delta`
- `profession_metric_delta_all`
- `profession_ideology_delta_all`
- `silo_flag_set`

# Silo40 内容作者手册 · 冷却、延迟与事件 ID 解析

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 6. 冷却与延迟

- `fire_mode: "REPEATABLE"` + `cooldown_months`：按游戏时间戳强制（30 天/月），冷却内条件
  满足也不触发。可从以下档位开始 sim 校准：小麻烦 cd 3–4、中等 cd 6、大灾 cd 9–12、
  唯一事件 `ONCE`。
- `schedule_event(event_id, N)`：N 个月后**无条件**派发目标事件（不检查其触发条件），
  走 `delayTasks` 队列。用于延迟代价链：`silo1_ultimatum → ultimatum_aftermath(+3 月)`、
  `materials_cannibalization → lower_deck_collapse(+4 月)`。
- **时间窗约束**：可玩窗口 = Y122M1 开局 → Y130M1 `silo1_destroyed_signal` → TIME 结局，
  约 96 个月。所有事件与延迟落点必须在 Y130 前；延迟档用 2–3 月（短）/ 4–6 月（中），
  且发起时机不晚于 ~Y128。
- **第 N 次质变**：`script.apply` 读 `ctx.runtime.trigger_count`（本次触发后的累计次数，
  首次=1）分支，范例 `events/special/materials_cannibalization.js`（第 5 次起出事故）。
- 密度预算：大事件 ≤1 个/月（脚本里程碑间隔 ≥3 月）；新增循环事件合计 ≤ ~0.55 次/月。

## 7. 事件 ID 与 `schedule_event` 解析

- `schedule_event` 与状态查找按 `EventID` **首个匹配**解析（`findContentDefinition`）：
  ID 重复 = 静默影子覆盖。防呆闸会报重复（`TestContentGuardEveryEventLoadsValidAndTriggerable`）。
- 运行时事件名 = `category:event_id`（如 `crisis:purge_backlash`）；`script.apply` 的
  `emit` 与 `event_triggered` 匹配都用这个名字。
- `event_triggered` 订阅链：父事件派发 → 订阅者立即评估跟随触发（同 tick）。
  **不要订阅经延迟队列派发的事件做依赖测试以外的用途**——历史崩溃路径，见 §12.5。


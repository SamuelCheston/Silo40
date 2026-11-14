# 结局事件链 · `PERSONAL_RULE`《人治》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./02-personal-rule.md)

链条形态：**正统性调查 → 喂证 Mayor 圈层 → 公开担当时刻 → 继承抉择**。与《公议》共用"低 `W` 后 Judicial 附着谁"的互斥分叉。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `legitimacy_survey` | 门控接触卡 | 解锁：`stage_at_least first_signal` + `relation Mayor min 0.15` | 抉择 P-A：把 Mayor 当障碍绕过还是当部分盟友；接触方是 Mayor 联系人 | 完成键二选一 | 接受后进入喂证步骤 |
| `mayor_evidence_step` | player_actions | `all(event_triggered 接受键, playerActionIs(Share Info))` | 抉择 P-B：责任链材料喂给 Mayor 圈层；玩家保留跨堡层 | 职务结果照常 | 担当时刻前置 |
| `mayor_crisis_stage` | special，ONCE | `any(event_triggered("crisis","food_poisoning"), event_triggered("crisis","water_leak"), event_triggered("special","energy_rolling_blackout"), siloMetricGte("rebellion", 0.3))` | 危机给 Mayor 合法舞台；玩家可选择推他提前对抗（额外完成键）或等舞台成熟 | `panic_value +0.03`（sim 校准起点） | 担当时刻解锁 |
| `mayor_public_stand` | special，ONCE | `all(event_triggered mayor_crisis_stage, siloMetricGte("legitimacy", 0.5))` | 抉择 P-C 的结算：Mayor 以个人权威公开拒绝不合法命令 | `rebellion −0.05`、`panic_value −0.05`（sim 校准起点） | 收束前置 |
| `succession_choice` | 接触卡，ONCE | 门控：`event_completed mayor_public_stand` | 抉择 P-D：接受纯粹人治（本线收束）或埋程序种子（转《公议》节点群） | 完成键二选一 | 收束或改道 |

## 2. 分支与改道

| 完成键／事实 | 去向 |
| --- | --- |
| 埋程序种子 | 转《公议》节点群：`charter_contact_offer` 解锁条件追加 `event_completed succession_choice` 种子键 |
| Mayor 成为撤离公开领头 | 《人治》失去持有者：本线关闭，随《越岭》节点群 |
| `W` 回升、IT 重新接管（设计口径） | 秩序回摆：转《深闭》惯性节点 |
| Mayor 死亡或更替且无继承安排 | 视后续状态转《深闭》（旧秩序回填）或《断层》（争夺空缺） |

## 3. 超时与失败结算

- 本线无全局超时；`mayor_crisis_stage` 是机会窗口而非倒计时——危机可以被别的线消费（同一危机也是《断层》的燃料、《遗忘》的掩护）。
- 失败形态是"担当时刻没有发生"：Mayor 正统性不足（`legitimacy < 0.5` 时 `mayor_public_stand` 永不点火），本线停在喂证步骤，局面按实际权力状态滑向《深闭》。

## 4. 与结局判定的对接

- 引擎已有 Mayor 附着动力学：`W ≤ 30` 且 Mayor 正统性 ≥0.35 时 Judicial 可迁移附着（[authority.go](../../../internal/engine/authority.go)）。`legitimacy` 可被内容触发器读取（`ctx.silo_metrics.legitimacy` 即 `MayorLegitimacy()`），这是本线唯一能直接读到的资格输入。
- `W` 读不到（G3）：内容无法以"`W` 已降"为触发；`mayor_public_stand` 用正统性阈值＋危机事实近似"担当成立"，`W` 的实际下降由机制层输入承担（执行失败、调度暴露等合格事实如何进 `W`，见 IT 控制可信度设计）。
- `PERSONAL_RULE` 不在仲裁候选（G1）；Mayor 更替时的 `resetJudicialForNewMayor` 已存在，继承空洞的剧情后果有机制对应物。

## 5. 能力缺口

G1（候选缺失）、G3（读不到 `W`）、G8（尾声展示）。Mayor 正统性的内容写入也受限：`legitimacy` 属派生类指标，内容 delta 活不过下一 tick，正统性的实际变化只能经机制层与权威引擎（G3 的延伸）。

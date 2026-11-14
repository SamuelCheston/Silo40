# 结局事件链 · `RESET`《遗忘》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./04-reset.md)

链条形态：**征兆 → 反制窗口（守卫超时）→ 执行或空转 → 完成事实键**。本线是守卫超时范式的样板：执行节点无条件到期，守卫决定它结算还是空转。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `reset_omen_census` | special，ONCE | `all(siloMetricGte("history_burden", 0.2), timestampAtOrAfter(2,1))`（相对 Y2 起） | 抉择 F-A：当作噪音还是当作准备 | `panic_value +0.04`（sim 校准起点） | `schedule_event(reset_omen_requisition, 3)` |
| `reset_omen_requisition` | crisis，纯调度 | `canTrigger: () => false` | 第二个无法解释的征兆：镇静与配额类物资异常请领 | `panic_value +0.03`、`rebellion +0.02`（sim 校准起点） | 反制接触解锁；`schedule_event(reset_execution, 9)` |
| `reset_countermeasure_offer` | 门控接触卡 | 解锁：`event_completed reset_omen_requisition` | 抉择 F-B 三向：分发托管（本卡）／渠道反制（见《寂堡》线 `termination_*`，共用接触骨架）／抬高执行成本（部门关系步骤） | 完成键 | 接受解锁步骤 |
| `copies_distributed` | special，ONCE | `all(event_triggered 接受键, event_triggered("player_action","action_share_info"))`（`trigger_count` 达配置次数） | 副本离开清洗范围：把材料分发给多个部门接收者 | `professionFragmentAdd()`×2 | 写完成键 `player_action:reset_countermeasure_done` |
| `reset_execution` | crisis，纯调度 | `canTrigger: () => false` | **守卫**：`apply` 读 `ctx.states["player_action:reset_countermeasure_done"]`；已完成→空转；未完成→结算 | 结算分支：`siloPopulationDelta(−1500)`（减口）、`history_burden −0.15`（清档）、`panic_value +0.10`、`rebellion +0.06`（sim 校准起点）；并 `emit ["special:reset_completed"]` | 完成事实 |
| `reset_completed` | special，纯调度 | `canTrigger: () => false` | 重置完成事实键：只被 `reset_execution` 的未守卫分支唤起 | `history_burden +0.02` | 供终局仲裁读取（已实现，P0.2） |

## 2. 分支与改道

| 事实 | 去向 |
| --- | --- |
| 守卫通过（反制完成） | 不判《遗忘》；处置评估继续，局面可滑向其他线；IT 可能改用缓慢绞杀（二次 `archive_purge_order` 链） |
| 独立逃出群体已先成立 | 原驻地重置只作《越岭》尾声；`reset_completed` 仍记录本地事实（仲裁例外读取已实现；完整存续结算依赖逃离模型，roadmap） |
| 执行分支结算且玩家仍在驻地 | 接《除名》线：玩家材料失去全部接收者，封顶压力上升 |
| 清洗链（`archive_purge_order`→`purge_backlash`） | 缓慢绞杀路径：不经过本线节点，直接以既有清洗链表达局部清档；只有全量执行才走 `reset_execution` |
| 本地整肃链（已实现：`silo1_purge_omen`→`silo1_purge_execution`，守卫 `silo1_purge_exposed`） | 通牒之后叛乱仍居高（≥0.8）时的本地执行链内容实现：投药、清档、减口；被曝光则整肃停滞。与本线 `reset_execution` 设计同构（守卫范式），落地时二者择一或合并 |

## 3. 超时与失败结算

- `reset_execution` 在第二次征兆后 +9 月无条件到期——这是"执行开始"决断（抉择 F-4）的机制化身：到期前玩家有多少反制，决定守卫结果。
- 守卫空转时 `reset_execution` 仍点火并留痕：**不得**把 `reset_execution` 的触发当作"重置已发生"；一切订阅只认 `reset_completed`（底座 §3 约定）。
- 延迟链允许溢出，但覆灭结算是终局时钟：重置要成为终局事实，`reset_completed` 必须在覆灭结算前成立；结算之后不再有内容窗口。若征兆链迟迟未启动而剧情又必须发生，按底座约定设兜底节点在结算前强制拉起。

## 4. 与结局判定的对接

- 机制底座已实现 RESET 候选：仲裁读取 `special:reset_completed` 事实键（P0.2）。
- 硬失败优先级（已实现）：`reset_completed` 先于《断层》《除名》与一切历史加权；先检查逃出群体（最小状态基础已落地，完整逃离模型未实装，见 roadmap）。
- 抹忆无机制对应物：药物失忆属剧情层（红线 5），机制上以 `history_burden` 深降与碎片清除近似，不假装等价。

## 5. 能力缺口

G8（终局画面展示）。原 G1（RESET 候选）已随机制底座落地；逃出群体例外读取已实现，完整逃离模型见 roadmap。碎片清除的精确表达也受限：`profession_fragment_remove` 需要指明部门与碎片，内容作者只能写死少量代表性清除对，无法"按持有状态动态清档"——登记为碎片系统的策略缺口（作者手册 §9）。

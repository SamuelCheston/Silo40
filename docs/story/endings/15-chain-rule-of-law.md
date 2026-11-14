# 结局事件链 · `RULE_OF_LAW`《公议》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./01-rule-of-law.md)

链条形态：**规则断点 → 制度内喂证 → 约章起草 → 锁定日**。本线完整采用行动设计 15 §5.3 的接触链范式：接触 → 固定选择 → 步骤 → 结果 → 守卫超时。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `order_chain_breach` | special，ONCE | `all(event_triggered("player_action","action_share_info"), siloMetricGte("history_burden", 0.10))` | 抉择 L-A 的机制投影：材料指向规则断点而非个人罪行 | `professionFragmentAdd()`（Judicial 侧第一份断点材料）、`history_burden +0.02`（sim 校准起点） | 接触解锁 |
| `charter_contact_offer` | 门控接触卡 | 解锁：`event_completed order_chain_breach` + `relation Judicial min 0.2` | 接触方是 Judicial 程序派联系人；可接、拒、暂缓 | 完成键三选一 | 接受解锁步骤与窗口 |
| `charter_contact_accept` | player_actions | 玩家选择 | — | 完成键 | `schedule_event(charter_window_expiry, 6)` |
| `charter_draft_step` | player_actions | `all(event_triggered 接受键, playerActionIs(Share Info))`，需两次 | 抉择 L-C：约章由本地人起草，玩家供证不代笔 | 职务结果照常 | `charter_draft_convened` |
| `charter_draft_convened` | special，ONCE | `event_triggered("player_action","charter_draft_step")`（`trigger_count ≥ 2`） | 起草会议召开 | `history_burden +0.03`（sim 校准起点） | 锁定日解锁 |
| `charter_lock_day` | special，ONCE | `all(event_triggered charter_draft_convened, timestampAtOrAfter(127,1))` | 抉择 L-D：妥协版本对不可逆锁定 | `rebellion −0.04`、`panic_value −0.04`（sim 校准起点） | **缺口 G2**：锁定事实（`PactLocked`）无内容写入入口 |
| `charter_window_expiry` | crisis，纯调度 | `canTrigger: () => false` | 守卫：`charter_lock_day` 已触发则空转；未触发则动能流失 | 未锁定分支：`rebellion +0.04`、`panic_value +0.04`（sim 校准起点） | — |

## 2. 分支与改道

| 完成键／事实 | 去向 |
| --- | --- |
| 拒绝或暂缓接触卡 | 本线关闭；材料可转喂 Mayor 圈层（转《人治》节点群） |
| 锁定日前 `W` 仍高（设计口径） | 约章锁不定：剧情上 IT 反扑成立；机制上 G2 未落地前以叙事节点表达 |
| 锁定完成 | 作为《共证》的制度尾声候选（同分时《共证》优先）；候选资格属 G1 |
| 清洗链在锁定日前完成 | 转《遗忘》：`reset_completed` 事实键先于制度成就 |

## 3. 超时与失败结算

- `charter_window_expiry` 是守卫超时：6 个月内未锁定则动能流失；已锁定则空转。超时事件本身不作失败条件被订阅（底座 §3 约定）。
- 锁定日失败（IT 反扑得逞）不设独立节点：以 `charter_window_expiry` 的未锁定分支与后续清洗链表达。

## 4. 与结局判定的对接

- 引擎已有 Pact 附着动力学：`W ≤ 18` 且 `Q ≥ 0.85` 且独立公共事实 ≥2 且 `PactQualified` 时 Judicial 可附着约章并锁定（[authority.go](../../../internal/engine/authority.go)）。
- 但 `PublicEvidenceQ / IndependentPublicFacts / PactQualified` 无任何内容或机制写入入口（G2），`RULE_OF_LAW` 也不在仲裁候选（G1）。本线事件链因此只落到**政治剧情层**：它生产"锁定应当发生"的全部剧情事实，把资格供给留给引擎工程。

## 5. 能力缺口

G1（候选缺失）、G2（约章资格供给）、G3（内容读不到 `W`，锁定日无法以 `W` 为触发条件，只能用时间＋完成键近似）、G8（制度尾声展示）。

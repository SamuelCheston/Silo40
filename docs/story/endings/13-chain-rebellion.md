# 结局事件链 · `REBELLION`《越岭》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](../good-endings/02-rebellion.md)
>
> 【裁决标注 · 已被取代】本分册的"逃离模型未实装、节点只预留"口径**已被取代**：准备侧按 [16 准备度换算设计](../../research/16-preparation-outcome-design.md) 的 Layer 0/1 落地，四节点（意愿→路线→开窗→离开）、顶层立场、准备账本与出发结算点的最小级联均已实装，见 [36 节点设计](./36-rebellion-nodes.md)／[37 字段级规格](./37-rebellion-event-spec.md)。本分册保留为链骨架的历史底稿；堡外月度维持、追踪四档与群体质量加权已随 Layer 2 落地（`escape_outside.go`），剩余缺口（实体交接×逃出群体耦合、Attention 承载、M4 延迟定性）见 roadmap。

链条形态：**找想走的人 → 证明路线 → 轮班表 → 开窗 → 分批离开 → 外部存续**。本线曾是十条线中机制缺口最集中的一条；如今组织、反抗与"离开并存续"均已可表达（出发结算与堡外月度维持已落地）。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `departure_willingness` | special，ONCE | `all(timestampAtOrAfter(1,1), professionMetricGteCount("panic_value", 0.5, 3))`（相对 Y1 起） | 抉择 R-A 的前置：离开意愿出现，尚不是队伍 | `rebellion +0.03`、`history_burden +0.02`（sim 校准起点） | 路线调查解锁 |
| `route_survey_offer` | 门控接触卡 | 解锁：`stage_at_least first_signal` + `relation Mechanical min 0.2` + `event_completed departure_willingness` | 接触方是 Mechanical 联系人（既有实体，不新增组织）；可接、拒、暂缓 | 完成键三选一 | 接受后解锁步骤 |
| `route_survey_step` | player_actions | `all(event_triggered route_survey_offer 接受键, playerActionIs(机械/矿业职务))` | 抉择 R-B 的机制投影：路线信息公开程度由后续宣传选择承担 | 职务主结果照常 | `route_confirmed` |
| `route_confirmed` | special，ONCE | `event_triggered("player_action","route_survey_step")` | 先遣事实成立 | `rebellion +0.04` | 解锁先遣派遣 |
| `advance_party_dispatch` | player_actions，ONCE | 门控：`event_completed route_confirmed` | 抉择 R-E 的前奏 | 完成键 | `schedule_event(advance_party_report, 4)` |
| `advance_party_report` | crisis，纯调度 | `canTrigger: () => false` | `apply` 守卫：若期间 `event_triggered("crisis","lower_deck_collapse")` 则路线受损分支 | 完好分支：`rebellion +0.04`；受损分支：`panic_value +0.05` | 开窗前置 |
| 开窗 | 既有行动与事件 | Incite Rebellion（情境化）＋ `underground_coordination`（rebellion≥0.62 且相对 Y4+） | 抉择 R-D：有序占用撤离通道（低结构损耗）对立于全面暴乱（高损耗） | 叛乱滚雪球由既有节点承担 | 分批撤离前置 |
| 分批撤离与外部存续 | — | — | — | **已落地（Layer 0/1/2）**：出发结算与堡外存续结算见 36/37（`escape_outside.go`） | 已写成可运行内容 |

## 2. 分支与改道

| 完成键／事实 | 去向 |
| --- | --- |
| 高叛乱但路线未确认 | 不得进入《越岭》；按社会状态继续模拟或滑向《断层》（`fracture_*` 节点） |
| 开窗选择全面暴乱 | `floor_stability_delta` 与资源损耗进入《断层》节点群 |
| 撤离组织者与核心复制者重叠 | 与《共证》争夺同一批人：离开即掏空宿主复制链（完成键层面的互斥见[十线仲裁](./22-chain-cross-arbitration.md)） |
| 原驻地随后被处置 | 逃出群体已成立并存续时只作惨胜尾声（仲裁例外读取与完整存续结算均已随 Layer 2 落地，`escape_outside.go`） |

## 3. 超时与失败结算

- `advance_party_report` 是延迟档：4 个月后无条件回报，守卫区分路线完好与受损。延迟链允许溢出，但涉及终局事实的节点必须赶在覆灭结算前。
- 开窗之后"分批离开"的结算已落地：出发结算点与堡外月度维持见 36/37（`escape_outside.go`）。本底稿"最后节点是开窗、其后只预留注释"的口径已被取代。

## 4. 与结局判定的对接

- 机制底座已实现：`victory_check` 对叛乱触发只**请求终局评估**；仲裁的《越岭》资格读逃出群体存续（`EscapedGroup` 存续结算，已随 Layer 2 落地），高叛乱无逃离不再判《越岭》。
- 完整资格（实际离开＋成规模存续）已随逃离模型落地（Layer 0/1/2）。本线事件链如今覆盖**从组织反抗到离开存续的全程**；剩余缺口（实体交接×逃出群体耦合、Attention 承载）见 roadmap。

## 5. 能力缺口

逃离模型已落地（Layer 0/1/2：路线、批次与堡外存续结算，`escape_outside.go`）；剩余缺口为实体交接×逃出群体耦合、Attention 承载与 M4 延迟定性（roadmap）、G8（惨胜尾声展示）。原 G9（占位语义修正）与逃出群体最小状态已随机制底座落地。

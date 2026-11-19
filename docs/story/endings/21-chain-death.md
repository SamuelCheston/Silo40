# 结局事件链 · `DEATH`《寂堡》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](./07-death.md)

链条形态：**终止征兆 → 反制窗口 → 执行**。本线一半挂在既有通牒链上，一半是终局破坏事件；分渠道结算的纪律贯穿全链。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| `silo1_ultimatum` 链 | 既有 histories | Y4M1 | 第一终止征兆：定向警告与信号测试；抉择 D-A | 既有 | 本线入口 |
| `termination_channel_test` | special，ONCE | `all(eventTriggered("special","silo1_purge_omen"), monthsSinceEventGte(...,6), siloMetricGte("order_pressure",0.45), actorMetricGte("propaganda_level",2), silo_resource_gte Energy ≥1)`（整肃前兆后 +6 月；`events/special/termination_channel_test.js`） | "没有成功"的关闭指令：渠道被试探 | `panic_value +0.05`、`rebellion +0.03`（sim 校准起点） | 反制动作解锁 |
| `termination_channel_jam` | 门控日常动作（`PLAYER_EVENT`，AP 2） | 解锁：`event_completed special:termination_channel_test` + `relation Mechanical/IT min 0.2`（`events/action_gates.json`） | 抉择 D-B：渠道反制（本动作）／降低偏离／让人离开（转《越岭》节点群）；藏在值夜检修里，不新增组织 | 调度 `termination_channel_disrupted`（+1 月） | 反制结算 |
| `termination_channel_disrupted` | special，纯调度 | `canTrigger: () => false`；由玩家动作 `termination_channel_jam` 调度（+1 月，不自触发；`events/player_actions/termination_channel_jam.js`） | 单一终止渠道失效——只免疫该渠道，不免疫物理手段（分渠道纪律） | `history_burden +0.03`、`worldFlagSet("network_jammed", true)`、`decisiveShock(...)`（sim 校准起点） | **已实现**：`termination_release`（纯调度投放节点）读 `network_jammed`，被干扰则空转 |
| `bombing_terminal` | crisis，纯调度 | `canTrigger: () => false`；由 `strike_reconnaissance`（终止自检后 +6 月、`order_pressure ≥ 0.45`、`propaganda_level ≥ 2`、Energy ≥ 1）调度（+3 月；`events/crisis/bombing_terminal.js`） | 物理摧毁：分渠道结算下基本无解的渠道 | **不写人口**：`apply` 返回空效果，只留下完成键 | `crisis:bombing_terminal` 被仲裁在 P0 读作 DEATH 事实 |
| Safeguard 路径 | 机制层 | `safeguard_risk ≥ 1.0` | 既有硬护栏；`safeguard_risk` 已可由内容效果写入（机制底座） | DEATH | — |

## 2. 分支与改道

| 事实 | 去向 |
| --- | --- |
| 独立逃出群体已先成立且存续 | 毁灭只作《越岭》惨胜尾声，不判《寂堡》；仲裁的例外读取已实现（最小状态基础），完整存续结算依赖逃离模型（roadmap，未实装） |
| 渠道反制成立后 Silo 1 改用物理手段 | 分渠道纪律的叙事时刻：反制成功不等于获救；`bombing_terminal` 不受 `termination_channel_disrupted` 影响 |
| `fracture_confirmed` 之后人口耗尽 | 本线的维生死寂变体：不经终止节点，由人口杠杆自然抵达 0 |
| 驻地活到覆灭结算 | 本线关闭：上游可用性归零，终止不再可能 |

## 3. 超时与失败结算

- `bombing_terminal` 是纯调度终局事件：调度方已实现（`strike_reconnaissance`：终止自检后局势仍不收敛、`order_pressure ≥ 0.45` 且宣传到位时为物理投送校位，+3 月调度终局节点）。延迟链允许溢出，但覆灭结算是终局时钟：终局破坏节点必须在覆灭结算前成立，否则在本局不再有结算意义。
- 无守卫：物理摧毁没有反制守卫（设计口径）；唯一的"守卫"是人已经离开（逃离模型，roadmap 未实装）。远程终止渠道则有守卫：`termination_release` 读 `network_jammed`，被干扰的渠道投放空转。

## 4. 与结局判定的对接

- 现状：人口 ≤0、`safeguard_risk ≥ 1.0`，以及仲裁读到的 `crisis:termination_release`／`crisis:bombing_terminal` 事实键都判 DEATH——本线复用既有硬护栏，无 G1 缺口（`internal/service/weighted_endings.go` `resolveWeightedEnding` P0）。
- `bombing_terminal` 不写人口、不归零人口：它只返回空效果并留下完成键，由仲裁在 P0 把完成键读作 DEATH 事实。它不进入常态控制力叙事（一次性摧毁，作者手册与 Silo 1 控制设计的口径）。
- 残余应答母题（几十年后微弱的线路应答）不进事件链：它不改变本结局，也不许诺。

## 5. 能力缺口

- 逃出群体例外：仲裁读取已实现（最小状态基础）；完整存续结算依赖逃离模型（roadmap，未实装）。
- `network_jammed`：写入（`worldFlagSet`）与消费（通牒暂停、`termination_release` 守卫）均已实现。
- G8（终局画面展示）。原 G6（内容→`safeguard_risk`）已随机制底座落地。
- 本线的终止征兆→渠道反制→物理投送链已有内容实现（`termination_channel_test / termination_channel_jam / termination_channel_disrupted / termination_release / strike_reconnaissance / bombing_terminal`）；本分册原设计的接触卡（`termination_countermeasure_offer`）已由日常动作 `termination_channel_jam` 承载：同一门控（自检点亮＋Mechanical/IT 最低信任），效果改为 +1 月后信道干扰成立。

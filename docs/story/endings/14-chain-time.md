# 结局事件链 · `TIME`《无人裁决》

> [返回总索引](../silo40-endings.md) · [机制底座](./10-chain-foundation.md) · [剧情台阶](../good-endings/03-time.md)

链条形态：**长期维持 → 覆灭前夕抉择 → 权威覆灭事实 → 快照判定**。本线的戏在"等待"，但等待不是空转：存续与开放都要靠实际行动维持。

## 1. 节点表

| 节点 | 分组 | 触发 | 分岔与抉择 | 效果（封闭集） | 后续 |
| --- | --- | --- | --- | --- | --- |
| 维持动作 | 既有行动与事件 | STATION_TRADE、职务行动、`grief_release_ritual`（危机后减压） | 抉择 T-A／T-C 的日常层：开放进入普通 Cohort、镇压来临时保住还能说话的人 | 资源、恐慌、楼层稳定度等既有杠杆 | 存续保持 |
| `open_discourse_milestone` | special，REPEATABLE cd6 | `professionIdeologyGteCount("pro_foreign", 0.55, 2)` | 开放思潮取得公共表达的里程碑；读的是部门派生视图（cohort 真源的投影） | `history_burden +0.02`（sim 校准起点） | 开放计数 |
| `pre_fall_choice` | 接触卡，ONCE | 门控：`timestamp_at_or_after(129,1)` 对应阶段条件 + `event_completed open_discourse_milestone` | 抉择 T-E 三选一，各写一个完成键：**忠诚动员**／**全面暴乱**／**维持公开质询** | 见 §2 | 覆灭前快照的叙事投影 |
| `silo1_destroyed_signal` | 既有 histories | Y130M1 | 权威覆灭事实；不受玩家改写 | 置 `silo1_destroyed` flag | 终局评估 |

## 2. 覆灭前夕三选一的机制投影

| 选择完成键 | 立即效果（封闭集内可表达部分） | 资格后果（设计口径） |
| --- | --- | --- |
| 忠诚动员 | `rebellion −0.05`、`panic_value −0.04`（sim 校准起点） | 覆灭前夕为纯种地堡思潮：《无人裁决》绝对失格；按实际权力状态落入《人治》或《深闭》 |
| 全面暴乱 | `rebellion +0.15`、`floor_stability_delta −0.1`（sim 校准起点） | 维存受损；按社会状态进入《断层》判定或继续模拟 |
| 维持公开质询 | `history_burden +0.03`（sim 校准起点） | 开放快照合格的叙事前提；实际资格由思潮快照判定（缺口 G4） |

注意：`profession_ideology_delta_all` 是短暂效果（每 tick 被 cohort 覆写），因此三选一**不能**用思潮 delta 表达资格变化；资格后果只能登记为设计口径，等 G4 落地。

## 3. 分支与改道

| 事实 | 去向 |
| --- | --- |
| 等待期间宿主因《共证》外溢暴露被处置 | 《无人裁决》失去存续驻地；按实际结果进入《遗忘》或《寂堡》 |
| 等待期间传播链外部复制成立 | 《共证》与《无人裁决》共同进入资格检查，统一仲裁只选一个主结局 |
| 覆灭前全面忠诚动员 | 转《深闭》死后复制节点（`post_fall_takeover`）或《人治》节点群 |

## 4. 超时与失败结算

- 本线没有独立超时：终局时钟是 `silo1_destroyed_signal`（Y130M1），全局唯一。
- 失败不是"没等到"，而是"等到了但资格不合格"或"没等到（驻地先被处置）"。

## 5. 与结局判定的对接

- 现状：`victory_check` 见 `silo1_destroyed` 即判 TIME——占位语义。目标判定要求覆灭前夕思潮闸（纯种地堡绝对排除），属缺口 G4＋G9。
- `open_discourse_milestone` 与 `pre_fall_choice` 的完成键是快照判定的**叙事输入**：G4 落地后，快照审计应能引用这两个键作为开放建设的证据链。

## 6. 能力缺口

G4（cohort 思潮直写与覆灭前夕快照）、G9（TIME 占位语义修正）、G8（尾声展示）。

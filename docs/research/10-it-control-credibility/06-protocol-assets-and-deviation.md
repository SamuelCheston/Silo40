# IT 幕后权力：威势、Protocol 能动性与 Judicial 三方附着 · Protocol 内容资产、三类偏离与介入链

> 拆分自 [IT 幕后权力：威势、Protocol 能动性与 Judicial 三方附着](../10-it-control-credibility.md)；目录、章节映射与出处见该索引。

## 6. Protocol 内容资产、偏离与上层控制方 认知

### 6.1 Protocol 作为内容资产

建议 `events/protocol/*.json`，与公开 Covenant 分离。每条条款：

```text
id / version / category / precondition
required_action_or_forbidden_action / deadline
target_metric / target_range / review_interval
severity / safeguard_delta / intervention_points
reporting_requirement / known_gaps
```

| 条款类别 | 覆盖情境 | 判定例 |
| --- | --- | --- |
| `prohibition` | 未授权越界、隐瞒必需报告、禁止的跨堡接触、禁止清理 | 实际行为匹配禁止谓词即构成违反 |
| `required_response` | 高危处置失败、异常信号、指定目击、失控前兆 | 在 deadline 前完成规定处置与上报 |
| `outcome_target` | 无失控叛乱、官方叙事仍可持续、指定异常被隔离 | 到 review 日读取真实结果，不看 IT 是否努力 |
| `reporting` | 何种异常必须记录、何时留档、何时允许极端接触 | 报告缺失单独计时，不能由伪造本地副本消除 |
| `gap/exception` | Protocol 没写或条件互相冲突 | 标为未知，不自动视为安全；IT 需用碎片推断 |

玩家能否读到：Protocol 不在公共图书馆。IT 学徒即使读遍普通藏书也不会得知上层控制方；主任也只能通过 Protocol、密室日志、任命培训 记忆和稀有接触逐步理解。非 IT 玩家可经法警调度调查找到密室位置、从倒戈 IT 内部人员取得副本、或在 IT 崩解后回收档案。

玩家可篡改/伪造**本地副本**，影响 IT 的 believed Protocol；不能篡改上层控制方 用来判定真实偏离的权威条款与结果。伪造副本同样具有 `B/π/provenance`，被识破会生成新事实。

### 6.2 三类偏离的判定与后果

| 类型 | 确定性判定 | 分级 | 对 `R_s` | 对 `W` | 对介入分 `J` |
| --- | --- | --- | --- | --- | --- |
| **违反 Protocol** | 实际动作命中有效禁止条款 | 轻微/重大/存在级 | +0.02 / +0.08 / +0.25 | 若联盟知悉，-2 / -5 / -10 | +0.2 / +0.6 / +1.2 |
| **未贯彻 Protocol** | 必需动作、清理或报告超过 deadline 未完成 | 轻微/重大/关键 | 每逾期月 +0.01 / +0.03 / +0.06 | -1 / -3 / -5 | +0.15 / +0.40 / +0.80 |
| **未达成 Protocol 目标** | review 日真实目标越界，即使每步合规 | 轻微/重大/关键 | 每次 review +0.02 / +0.05 / +0.10 | -2 / -4 / -6 | +0.25 / +0.60 / +1.00 |

- 违反强调行为本身，未贯彻强调时限，未达成强调结果；同一事件可分别命中，但每个条款每类只记一次。
- IT 对条款不知情不会改变真实判定，只影响其预测；伪造报告可推迟“未贯彻”被发现，不能消除“未达成目标”。
- `W` 惩罚需要关键联盟知悉，`R_s/J` 是权威系统结果，不依赖全民舆论。

### 6.3上层控制方 介入：罕见、确定、可预测

```text
J = Σ intervention_points_of_open_deviations
```

常态无接触。满足以下之一才进入介入链：

1. **存在级违反**：立即进入一次权威审查；
2. `J≥1.0`：生成带具体条款原因的异常预警，给玩家和 IT 可观察征兆；
3. `J≥1.5` 且关键目标连续两个 review 未恢复：进入“限期纠正 → 权威覆写/重置 → Safeguard 评估”序列。

介入不使用随机抽取，不因单月小偏离直接天罚。预警必须显示缺失条款、剩余 review 数和可能后果；上层控制方 有人出于良心直接提醒属于 `J≥1.0` 后的稀有具名事件，不是日常指令通道。`R_s≥1` 仍是独立硬失败。


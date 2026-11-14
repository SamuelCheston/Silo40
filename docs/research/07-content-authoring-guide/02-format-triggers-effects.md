# Silo40 内容作者手册 · 事件文件格式、触发器与效果封闭集

> 拆分自 [Silo40 内容作者手册](../07-content-authoring-guide.md)；目录、章节映射与出处见该索引。

## 2. 事件文件格式

- 每文件恰好一次 `defineEvent({...})`；JS 由 goja 解析，函数被剥离、只留结构化数据 +
  `script` hook 标记。
- 必填：`id`、`title`、（`trigger` 或 `script.canTrigger`）、（`effects` 或 `script.apply`）；
  `fire_mode` 缺省 `ONCE`。
- 校验在加载时执行；**反例**：`events/special/outside_signal.js` 初版无 trigger 无脚本 →
  挡住全部内容加载（`bd18be1` 修复）。
- 代码注释用英文（仓库约定）；`id` 用 snake_case 并与文件名一致。

## 3. 触发器封闭集

| 触发器 | JS helper | 备注 |
| --- | --- | --- |
| `timestamp_at_or_after` | `timestampAtOrAfter(year, month)` | 游戏历基准 Year 100，30 天/月 |
| `silo_metric_gte/lte` | `siloMetricGte/Lte(metric, value)` | metric 见 §4 指标表 |
| `silo_resource_gte/lte` | **无 helper**，写裸对象 | `{ type: "silo_resource_lte", metric: "Supplies", value: 3000 }` |
| `silo_population_gte/lte` | **无 helper**，写裸对象 | `{ type: "silo_population_lte", value: 6000 }` |
| `silo_flag_true/false` | `siloFlagTrue/False(flag)` | flag 只有 `silo1_destroyed`（§9） |
| `event_triggered` | `eventTriggered(category, event_id)` | 上游须已在本局触发；带它的定义**不参与月度系统评估**，只在父事件派发时跟随触发 |
| `player_action_is` | `playerActionIs(action)` | 仅 `player_actions/` |
| `profession_metric_gte_count` | `professionMetricGteCount(metric, value, count, profession?)` | 满足条件的部门数 ≥ count |
| `profession_ideology_gte_count` | `professionIdeologyGteCount(ideology, value, count, profession?)` | 读的是 cohort 派生值（§8） |
| `all` / `any` | `all(...)` / `any(...)` | 组合 |
| （脚本） | `script.canTrigger(ctx)` | ctx 只读：`silo_metrics`/`resources`/`professions`/`runtime`/`states`/`total_population`/`year`/`month` |

**纯调度事件约定**：只经 `schedule_event` 唤起的事件，写
`script.canTrigger: function(ctx) { return false; }`——延迟队列派发不检查触发条件，
而月度评估因恒假不会提前点燃。反例：不加它则校验报 "missing trigger or script.canTrigger"。

## 4. 内容事件效果封闭集

| 效果 | JS helper | 可用字段 | 持久性 |
| --- | --- | --- | --- |
| `silo_metric_delta` | `siloMetricDelta(metric, v)` | `legitimacy` `cohesion` `rebellion` `dept_tension` `class_fragmentation` `history_burden` `event_trigger` `countdown` | 见下表 |
| `silo_resource_delta` | `siloResourceDelta(res, v)` | `Energy` `Materials` `Supplies`（下限 0） | 持久 |
| `silo_population_delta` | `siloPopulationDelta(v)` | 总人口（下限 0） | 持久 |
| `profession_metric_delta_all` | `professionMetricDeltaAll(metric, v)` | `panic_value`（年衰减 10%）/ `productivity` | 持久 |
| `profession_ideology_delta_all` | `professionIdeologyDeltaAll(ideology, v)` | `pro_foreign` / `loyalty` | **短暂**（§8） |
| `silo_flag_set` | `siloFlagSet(flag, bool)` | 仅 `silo1_destroyed` | 持久 |
| `profession_fragment_add` | `professionFragmentAdd(profession?, fragment?)` | profession 空=补给碎片最少的部门；fragment 空=种子随机抽该部门未知碎片 | 持久 |
| `profession_fragment_remove` | `professionFragmentRemove(profession, fragment)` | 两字段必填；不存在则无操作 | 持久 |
| `floor_stability_delta` | `floorStabilityDelta(zone?, v)` | zone 空=全部楼层，否则 `Upper`/`Mid`/`Lower`；0–1 截断 | 持久 |
| `schedule_event` | `scheduleEvent(event_id, delay_months)` | `delay_months > 0` | 见 §6 |

指标持久性（`metrics_update`/`ideology_update` 每 tick 重设派生值，内容 delta 活不过下一 tick）：

| 持久（内容事件的有效杠杆） | 派生/短暂（写了只有当次快照可见） |
| --- | --- |
| `rebellion`、`history_burden`、`countdown`、资源、人口、部门 `panic_value`、`productivity`、部门碎片、楼层稳定度 | `cohesion`、`legitimacy`（=派系加权凝聚力）、`dept_tension`（=1−关系均值）、`class_fragmentation`（=忠诚差）、部门 `Ideologies`（§8） |

**别碰 `event_trigger`**：空池虽已防护（`bd04aac`），正向 delta 仍无收益、只有风险。


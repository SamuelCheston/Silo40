# Silo40 事件内容清单与实现计划 · 机制基线（只读探针实测）

> 拆分自 [Silo40 事件内容清单与实现计划](../06-silo40-event-plan.md)；目录、章节映射与出处见该索引。

## 2. 机制基线（只读探针实测，种子 7 / IT 特工 / 无玩家动作 / 97 个月）

以下全部为实测数据（`internal/service` 临时探针，已删除），不是推断：

### 2.1 无人值守基线轨迹

| 时点 | 人口 | Energy | Materials | Supplies | 最大部门恐慌 | 凝聚力 | 叛乱 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Y122 开局 | 10000 | 5000 | 2000 | 5000 | 0 | 1.0 | 0 |
| Y123 | 9389 | 0 | 0 | 3719 | 0.04 | 0.88 | 0 |
| Y125 | 7297 | 0 | 0 | 0 | 0.39 | 0.68 | 0 |
| Y127 | 5395 | 0 | 0 | 0 | 0.93 | 0.48 | 0 |
| Y130 | 3781 | 0 | 0 | 0 | 0.97 | 0.32 | 0.003 |

- **资源首年崩溃**：结构性赤字（人口消耗 10000/年 > 各链产能，Materials 链尤甚——Mines
  1000 人 × 4.5 = 4500/年 < 10000/年消耗），Energy/Materials 约 1 年内归零，随后人口以
  ~15%/年 衰减直到消耗与产能重新咬合。
- **结局**：第 97 个月 TIME（胜），人口约 3700。无人值守结局分布 = 100% TIME。
- **恐慌饱和**：Y126 起最大部门恐慌 ≥0.76，终局 0.87–1.0。
- **思潮**：终局各部门 `pro_foreign` 0.50–0.81（恐慌驱动漂移，`ideology_update`），
  `pro_foreign_awakening` 于 ~Y127M2 触发。
- **叛乱≈0**：激进派系压力（派系凝聚力 <0.4 的影响力占比）直到终局才微量累积。
- **既有事件触发次数**：water_leak ×17、food_poisoning ×14、outside_signal ×9、
  各 ONCE 事件 ×1（broadcast / awakened / pro_foreign_awakening / silo1_destroyed / fallout）。
  总密度 ≈ 0.46 次/月。

### 2.2 内容事件可用的"持久杠杆"与"派生表"（代码事实）

`metrics_update` / `ideology_update`（`core_systems.js`）每 tick 会**重设**以下指标，内容事件
对它们的 delta 只能在落点当月的快照里可见，不构成持久机制：

- `cohesion` / `legitimacy`（= 派系影响力加权凝聚力；无政治派系时不重设，delta 才残留）
- `dept_tension`（= 1 − 部门关系均值）、`class_fragmentation`（= 精英/平民忠诚差）
- 部门 `Ideologies`（`sync_profession_ideologies_from_cohorts` 每 tick 用 cohort 人口加权覆写）

**持久杠杆**（内容事件 delta 会留存并继续参与后续结算）：

| 杠杆 | 后续机制咬合 |
| --- | --- |
| 部门 `panic_value` | 每年自然衰减 10%；>0 时驱动 cohort `pro_foreign` 漂移（panic×0.10/年），>0.3 且凝聚力 <0.5 时附加漂移；可被 `profession_metric_gte_count` 读取 |
| `rebellion` | 每 tick ±（激进派系压力 >0.05 则 +，否则 −0.02/年）；>0.7 产能 ×0.3、>0.8 额外死亡、≥1.0 → REBELLION 结局 |
| `history_burden` | 无自然衰减；供阈值事件、评分与结局叙事读取 |
| 资源（Energy/Materials/Supplies） | 生产链、枯竭死亡（每种 +0.05/年）、资源阈值触发器 |
| 人口 | 枯竭死亡基数、人口阈值触发器、≤0 → DEATH |
| `schedule_event` / `event_triggered` | 延迟后果与升级链（`panic_storm → delayed_riot` 既有范式） |

**设计原则**：新事件只把数值投到持久杠杆上；`cohesion/legitimacy` 类 delta 不再用于新事件
（既有 8 事件里的保留原样，不做语义重设计）。`event_trigger` 累积器一律不碰——空池采样
（`rng.Intn(0)`）风险见 §6 待决项。

### 2.3 加载面事实（影响事件放在哪个目录）

- 服务端只加载 `histories / special / crisis / player_actions(/events)` 四个分组
  （`contentEventDirectories` 白名单）。**`events/rules/` 不会被加载**：既有 4 个规则事件
  （panic_storm / rebellion_escalation / supplies_shortage / delayed_riot）在当前运行时是死内容。
  本计划不依赖它们；修复需改 `internal/service` 白名单 → 记为待决项（§6.2），不顺手改。
- 校验要求每个事件有 `trigger` 或 `script.canTrigger`（既有 3 个事件曾因此挡住整个
  bootstrap，已在 batch-0 修复：`bd18be1`）。
- `silo_resource_gte|lte` / `silo_population_gte|lte` 触发器引擎支持但无 JS helper，
  用裸对象字面量书写（已写入 `events/README.md`）。
- **纯调度事件约定**：只经 `schedule_event` 唤起的事件，写 `script.canTrigger: () => false`
  作为名义触发器——延迟队列唤起时走 `handleContentEvent`，不检查触发条件；而月度系统评估
  因 canTrigger 恒假永不点燃。这样既过校验又不提前误触。


# Silo40 事件内容清单与实现计划

> 用途：`events/` 新事件的选题、校准与分批实现依据。本篇是内容计划，不是设定考据；
> 设定依据引自本目录 01/02/05 篇，机制依据引自仓库代码与只读探针实测。
>
> 硬约束（来自任务）：只写**第 40 号地堡**自己的故事——借世界观/结构/部门/禁忌/思潮/技术设定，
> 不搬 Silo 18 剧情线（不写原著具体人物命运）；只用现有 `events/` DSL 与封闭效果集，
> 不新增效果/变异/事件类型或管线阶段；不改 `internal/engine/`；与现有 8 个内容事件不重复。

## 1. 时间窗 → 游戏历映射

### 1.1 任务时间窗

"原著主角出生前 10 年 → 原著结束（Dust 收尾）"。按 02 篇的小说纪年：

| 锚点 | 小说纪年 |
| --- | --- |
| 原著主角出生前 10 年（窗起点） | ~2301（原著正篇约 34 岁，推得出生 ~2311）【推断】 |
| 正篇开始 | 2345 |
| Dust 收尾（窗终点） | ~2346 |

窗口**排除**：建堡与封堡期（2049–2052）、改编侧原创前史线，以及正篇结束之后的内容。窗口内事件可以**提及**更早的历史事实（大革命、抹忆重置作为"历史包袱"存在），但不把那些时代当作事件发生的舞台。

### 1.2 游戏历

游戏自有纪年（代码事实）：

- 开局 `CurrentYear = 122 / Month 1`（`game_service.go:500-503` 默认值；`cmd/debug`、各测试同）。
- `histories` 既有锚点：`archive_truth_broadcast` Y122M1、`silo1_destroyed_signal` Y130M1。
- `silo1_destroyed` flag 置位 → 下一 tick `victory_check` 判 **TIME** 结局
  （`core_systems.js:368-377`）。因此**无人干预的默认局在 Y130M2 结束，可玩窗口 ≈ 96 个月**。
- 500 年 `Countdown` 是叙事框架（每年 −1），不参与任何结局判定——不作为事件杠杆。

结论：任务窗口 [窗起点, 窗终点] 完全覆盖可玩窗口 [Y122, Y130]。所有新事件必须落在
Y122–Y130 内，且延迟后果链的落点也要在 Y130 之前（延迟档设计见 §3.3）。

### 1.3 Silo 40 自己的历史定位（本游戏叙事基线）

小说中 Silo 40 曾率 11 堡断联、被 Silo 1 轰炸（2212，【原著明确】，01 篇 §6.2）。本游戏采用
**覆灭后特工视角**：40 号实体在 2212 年轰炸中被毁；主角是 40 号本人在覆灭前派出的特工，经简陋
休眠舱在约 2291–2301 年强制唤醒，记忆完整继续执行任务（前史见仓库 `docs/story/silo40-backstory.md`）。
40 号不作为可运转实体存在，只保留叙事中的派出方与"断联战争"的历史包袱；特工带着这份包袱，面对
Silo 1 仍在运转的控制网络。Y130 的 `silo1_destroyed_signal` 是任务的终局契机。新事件的叙事主线围绕
"旧网络正在死去、控制者正在失能"的弧线展开——由 40 号派出的特工亲眼见证，与 Silo 18 剧情无交集。

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

## 3. 量化校准（基准：quantitative-event-design-reference.md）

### 3.1 粒度换算

Silo40 回合粒度 = 1 个月（30 天；`deltaYears = 1/12`）。RimWorld 参考以天为单位
（其"年"= 60 天季度制），换算按 **30 天 = 1 回合**折算并向上取整到整月：

| 参考档位（RimWorld 冷却） | 天数 | Silo40 折算 | 本计划采用 |
| --- | --- | --- | --- |
| 小麻烦 | 8–15 天 | 0.3–0.5 月 | 冷却 3–4 月（低于 1 月的冷却在月粒度下等价于每回合，过密） |
| 中等 | 30 天 | 1 月 | 冷却 6 月（既有 crisis 档，保持一致） |
| 大灾 | 90–140 天 | 3–4.7 月 | 冷却 9–12 月 |
| 传奇/唯一 | 60–360 天或每局一次 | 2–12 月或 ONCE | ONCE + 时间闸 |

采用冷却不低于 3 月的原因：可玩窗口仅 96 个月，冷却 1 月的可重复事件会贡献 ~1 次/月的密度，
单个事件即吃满参考密度预算。

### 3.2 密度预算

- **大事件（脚本里程碑 + 一次性转折）≤1 个/月**：本计划所有大事件间隔 ≥3 个月
  （Y122 既有 broadcast → Y124 邻堡静默 → Y126 最后通牒 → Y130 既有 silo1 失联），
  延迟落点另行错峰（§4 各批注）。
- **小事件均值目标 0.2–0.5 个/月（新增部分）**：基线既有小事件已占 0.32/月
  （water_leak + food_poisoning + outside_signal），新增循环事件合计预算 ≤0.55/月
  （§4 表内逐个标注预估触发次数）。总体密度 ~0.9–1.0/月高于参考的随机池建议值——
  确定性触发系统没有"抽取"环节，冷却即节奏；此为**有记录的偏离**，用 sim 回归兜底。
- **高级内容用状态门槛锁**（参考 §4.1.2 的预算门槛思路）：后期事件用人口/思潮/历史包袱
  阈值门，保证前期见不到后期内容。

### 3.3 后果延迟档位（参考 §4.2 → 月粒度）

| 档位 | 参考原型 | 本计划实现 |
| --- | --- | --- |
| 立即 | Frostpunk 法律即改计量 | 事件主效果 |
| 短延迟 2–3 月 | 限时 quest 3 天 | `schedule_event` 2–3 月（通牒余波、停电级联） |
| 中延迟 4–6 月 | Problem→Protest→Riot 链 | `schedule_event` 4–6 月（清洗余波、菌株副作用） |
| 第 N 次质变 | Emergency Shift 第二次死亡 | `script.apply` 读 `ctx.runtime.trigger_count` 分支（拆解事件第 5 次起质变） |
| 受创泄压 | RimWorld adaptation −30 天 | 危机→互助事件（`event_triggered` 反向链：灾后立即减压） |

所有延迟落点 ≤ Y130：最晚的调度发生在 Y128 前（延迟 ≤6 月）。

## 4. 事件清单

命名与分组遵循 `events/README.md` 约定；"纯调度"= `script.canTrigger: () => false`、
只经 `schedule_event` 唤起。预估次数按基线轨迹估算（±30%），以 sim 回归为准。

### 批次 1：邻堡网络与记忆政治（5 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果（持久杠杆） | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `neighbor_silo_silence` | histories | Y124M1 起（脚本里程碑） | 旧断联同盟里的邻堡逐个失联——当年 Silo 40 带头断联、随后在轰炸中毁灭，如今网络在它派出的特工眼前死去 | 脚本里程碑抬升历史包袱与叛乱，为 Y130 大结局铺弧 | `timestampAtOrAfter(124,1)`，ONCE | hb +0.05、rebellion +0.04、全部门 panic +0.04 | `archive_truth_broadcast` 是堡内档案真相；本事件是**外部网络**信号；`silo1_destroyed_signal` 是 Y130 的结果，本事件是其前兆 |
| `silo1_ultimatum` | histories | Y126M1 起（脚本里程碑） | Silo 1 经旧协议通道发来定向警告：终止信号测试 | 大威胁立即压顶 + 3 月后"适应与积怨"分流（受创泄压的镜像） | `timestampAtOrAfter(126,1)`，ONCE | panic +0.08、rebellion +0.06、`schedule_event(ultimatum_aftermath, 3)` | 既有事件无一涉及 Silo 1 的**主动施压**（失联是它的下场，不是它的动作） |
| `ultimatum_aftermath` | special | Y126M4 前后（纯调度） | 恐慌回落（适应），怨气沉淀 | 延迟档：威胁后恐慌泄压、叛乱续涨 | 纯调度 | panic −0.04、rebellion +0.05 | — |
| `archive_purge_order` | special | hb ≥0.12 后（基线 ~Y124M2：broadcast 0.08 + 邻堡 0.05） | IT 对觉醒发动二次档案清洗 | 历史包袱的"对抗性操作"：清洗立即降 hb，但埋下反弹 | `siloMetricGte("history_burden", 0.12)`，ONCE | hb −0.06、rebellion +0.05、panic +0.06、`schedule_event(purge_backlash, 4)` | `history_burden_awakened` 是 hb 的**读出**；本事件是 hb 的**反向操作**（抹除），构成拉锯 |
| `purge_backlash` | crisis | 清洗后 4 月（纯调度） | 被抹除的目录在楼梯间口口相传，抹除本身成为新的历史包袱 | 延迟反弹：清洗净收益≈0（−0.06 后 +0.04） | 纯调度 | rebellion +0.06、hb +0.04、panic +0.04 | 既有 crisis 均为资源型灾害，无记忆政治链 |

批次 1 大事件错峰：neighbor Y124M1 → purge ~Y124M2（条件触发，随局浮动）→ purge_backlash ~Y124M6 → ultimatum Y126M1 → aftermath ~Y126M4。

### 批次 2：资源与劳动（9 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果 | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `energy_rolling_blackout` | special | Y123 起（Energy ≤300 常驻） | 分区轮流停电 | 资源阈值 → 恐慌 + 2 月后级联到食物链（Energy→Supplies 咬合） | `{silo_resource_lte Energy 300}`，REPEATABLE cd9 | panic +0.06、`schedule_event(blackout_cascade, 2)` | `water_leak` 是 Materials→Supplies 链；本事件是 **Energy→Supplies** 链，且带延迟级联 |
| `blackout_cascade` | crisis | 停电后 2 月（纯调度） | 水培照明离线，一批作物报废 | 延迟档：能源危机传导为食物损失 | 纯调度 | Supplies −250、panic +0.04 | — |
| `materials_cannibalization` | special | Y123 起（Materials ≤200 常驻） | 拆解底层闲置设施应急 | 即时回血 + 第 N 次质变（`trigger_count ≥5` 起出现拆解事故）+ 每次都埋 4 月后的住户事故 | `{silo_resource_lte Materials 200}`，REPEATABLE cd12 | Materials +300、`schedule_event(lower_deck_collapse, 4)`；apply 分支：count≥5 追加 panic +0.05、人口 −50 | 既有事件只**消耗**资源；本事件首次提供资源侧**应急手段**并附延迟代价（Frostpunk 式"同一手段第 N 次质变"） |
| `lower_deck_collapse` | crisis | 拆解后 4 月（纯调度） | 被拆解的住户区发生结构事故 | 延迟代价 | 纯调度 | 人口 −50、panic +0.05 | — |
| `labor_conscription` | special | 人口 ≤6000 后（基线 ~Y126） | 强制轮班：把丧亲者编入生产线 | 即时产出 + 3 月后怨气反弹；人口阈值门（后期内容锁） | `{silo_population_lte 6000}`，REPEATABLE cd9 | Supplies +800、`schedule_event(conscription_backlash, 3)` | 既有事件无人口阈值触发，也无"即时收益+延迟政治代价"结构 |
| `conscription_backlash` | crisis | 强制轮班后 3 月（纯调度） | 疲劳与怨气累积 | 延迟政治代价 | 纯调度 | rebellion +0.04、panic +0.05 | — |
| `geothermal_surge` | special | Y123M6 起（Energy ≤500 常驻） | 地热井季节性脉冲 | **馈赠通道**（正面事件与危机共用时钟）：缓解但不解决能源缺口 | `all(timestampAtOrAfter(123,6), silo_resource_lte Energy 500)`，REPEATABLE cd12 | Energy +600、panic −0.03 | 既有 8 事件全部是负面/中性；本事件是首个正面事件 |
| `hydroponics_mutant_strain` | special | Y123M6 起（Supplies ≤800） | 水培区选育出高产变异株 | 一次性大幅回血 + 6 月后副作用（延迟健康代价） | `all(timestampAtOrAfter(123,6), silo_resource_lte Supplies 800)`，ONCE | Supplies +1000、`schedule_event(strain_side_effects, 6)` | — |
| `strain_side_effects` | crisis | 变异株后 6 月（纯调度） | 长期食用的副作用显现 | 延迟代价 | 纯调度 | panic +0.04、rebellion +0.03 | — |

### 批次 3：社会与禁忌（4 个文件）

| id | 分组 | 时间子窗 | 主题 | 机制内核 | 触发 | 效果 | 与既有事件的区别 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `stairwell_rumor_cascade` | special | 中后期（≥5 部门恐慌 ≥0.6；基线 ~Y126） | 楼梯间流言网把事故串成"被隐瞒的真相" | 恐慌广度阈值 → 叛乱/历史包袱转化（恐慌→政治化的显性节点） | `professionMetricGteCount("panic_value", 0.6, 5)`，ONCE | rebellion +0.06、hb +0.04、panic +0.03 | `history_burden_awakened` 读 hb；本事件读**恐慌广度**，是恐慌轴的升级节点 |
| `grief_release_ritual` | special | 任一资源危机之后 | 灾后互助与悼念 | **危机→减压反向链**（`event_triggered` 的缓解用法）：灾难当月减压，社区把创伤转为团结 | `any(eventTriggered("crisis","food_poisoning"), eventTriggered("crisis","water_leak"), eventTriggered("crisis","blackout_cascade"))`，REPEATABLE cd9 | panic −0.10、rebellion −0.02 | 既有 crisis 链只有升级方向；本事件是首个**缓解方向**的链 |
| `public_cleaning_sentence` | special | Y124 后且 ≥2 部门 pro_foreign ≥0.55（基线 ~Y127–128） | 当局执行一次公开清洁判决以重申信条 | 禁忌的双面性：短期压恐慌（处刑即宣传），长期激进化；+5 月后目击者余波 | `all(timestampAtOrAfter(124,1), professionIdeologyGteCount("pro_foreign", 0.55, 2))`，ONCE | panic −0.06、rebellion +0.10、hb +0.04、`schedule_event(cleaning_witness_unrest, 5)` | `outside_signal` 是外部信号的被动接收；本事件是当局**主动执行禁忌**，且带思潮阈值门 |
| `cleaning_witness_unrest` | crisis | 公开清洁后 5 月（纯调度） | 目击者在底层复述"他走出去了" | 延迟政治代价 | 纯调度 | rebellion +0.06、panic +0.06、hb +0.03 | — |

### 叛乱预算核对（无人值守基线预估）

新事件合计注入 ≈ +0.11（通牒链）+0.11（清洗链）+0.06（流言）+0.16（清洁链）+0.12（轮班链，
3 次×0.04）+0.03（菌株）−0.10（互助，~5 次×0.02）≈ **+0.49**；叠加基线自然漂移（≈0），
终局 rebellion ≈ 0.5 < 1.0——**REBELLION 结局在无人值守下保持不可达**，仍需玩家动作推高
（是否要让它可达 → §6.3 待决项）。

## 5. 验证与回归

每批执行：

1. `go build ./...` + `go test ./internal/...`（含内容事件加载校验、既有确定性契约测试）。
2. 无头探针（临时 `internal/service` 测试，跑完删除）：加载真实 `events/`、96+1 个月推进、
   断言——(a) 本批事件按预期触发（次数/时点落在预估区间）；(b) 无事件 0 触发（除纯调度
   事件只经链路唤起）或开局必触发；(c) 结局仍为 TIME 且月份 = 97（无人值守基线不被打破）。
3. `cmd/sim` 需要 GUI 进程（`SILO40_DEBUG_HTTP_ADDR`），沙箱若不可用则如实报告，
   以第 2 步探针 + 既有测试替代（当前计划：先尝试 `wails build` + 真实 sim）。
4. 分支 `content/silo40-events-batchN-20261112` → 原子提交 → `--no-ff` 合并回 main → 推送 → 报告。

防呆闸：`internal/service/content_guard_test.go`（`d69cdcb`）持续守卫"静默内容丢失"——
磁盘上任何含事件文件的分组若被白名单漏掉（死内容）、任何定义加载/校验失败、缺可用触发器或
效果、事件 ID 重复，测试都会直接红并报出分组/文件/事件名。新增事件分组或改动加载白名单时
无需再手动核对。

## 6. 待决项与缺陷处理记录

状态标记：【已修复】/【判定：设计如此】/【方案待决】/【留待作者】。

1. 【已修复 `8eec461`】**`events/rules/` 不加载**：`contentEventDirectories` 白名单缺 `rules`，
   4 个规则事件是死内容。修复=白名单加入 `rules`；`delayed_riot` 补 `canTrigger: () => false`
   过校验（纯调度约定）。新增测试：规则组加载 + `panic_storm → delayed_riot` 时延链路。
   **sim 对照（20 局，种子 7–26，IT）**：修复前 TIME 100%、avg pop 4486 → 修复后 TIME 100%、
   avg pop 4480——分布无变化（delayed_riot 的 panic+0.15 在基线已饱和，legitimacy 为派生值）。
2. 【判定：设计如此，代码不动】**部门思潮每 tick 被 cohort 覆写**：git 考据确认这是有意设计——
   `2255de4`（2026-08-11，"refactor(ideology): use cohorts for ideology tracking"）提交说明
   明确："track ideologies on population Cohorts as **single source of truth**"、"auto-sync of
   department ideology aggregates from cohort data"、"Update all profession actions to modify
   cohort ideologies instead of direct silo/profession values"。即 `Profession.Ideologies` 自
   该重构起就是 cohort 的人口加权派生视图，`sync_profession_ideologies_from_cohorts` 每 tick
   覆写是**特性而非 bug**。影响：内容事件的 `profession_ideology_delta_all` 写入派生视图，活不过
   下一 tick（`outside_signal` 的 pro_foreign +0.05 无持久机制作用，仅当次快照可见）；内容事件
   想持久影响思潮，现有唯一杠杆是 `panic_value`（驱动 cohort 漂移）。若未来要内容事件直达
   cohort 思潮，需新增内容效果类型（见第 6 条同类扩展），由作者立项。
3. 【已解决（作者拍板 adjust，`bfcbada`/`18e3b47`）】**REBELLION 无人值守可达**：
   实现=门槛式滚雪球事件 `underground_coordination`（rebellion ≥0.62 且 Y126 起，cd6，
   每次 +0.10）。基础叛乱预算（无人值守终局均值 ~0.78、双峰分布 0.69–0.96）决定越线时机，
   越早越线越能在 Y130 TIME 钟之前滚到 1.0。**实测（20 局种子 7–26）：REBELLION 30% /
   TIME 70%；40 局复测 35%/65%**。调参敏感度实测：门槛 0.65→95%、0.67→5%、0.62→30%
   （暴乱注入 ±0.01 同级别敏感）——门槛位于轨迹双峰之间才有分水岭效果。
   另注意判定优先级 TIME 先于 REBELLION：Y130 同刻叛乱到 1.0 也算 TIME（部分局实测如此）。
4. 【已修复 `bd04aac`】**`EventTrigger` 空池采样**：`TriggerRandomEvent` 在池为空时改为直接不触发
   （预算保留），消除 `rng.Intn(0)` 崩溃。测试：`TestTriggerRandomEventEmptyPoolNoTrigger`。
5. 【已修复 `fd03bf7` + `326241d` + `d70dca6`】**内容事件故事不进 `TickResult.Stories`**：
   根因有两层——(a) `engine.Drain` 只回收 `tickStories`、丢弃事件自带的
   `step_stories`/`step_logs`（service 层 `drainQueuedEvents` 一直在收，引擎侧没收），
   `fd03bf7` 对齐回收；(b) step 流（`BeginPassTime → ProcessNextEvent`）里
   `runOneMonth` 的返回值被整段丢弃、且月度 tick 排空队列后 `Step` 报
   "no pending event"、操作卡死在 `phaseQueued`，`326241d` 改为回传该月故事/日志并推进
   操作状态。呈现端：`App.tsx` 结果行原本只显示 `step.stories[0]`，`d70dca6` 改为按既有
   句式连显全部故事（无新交互）。日志端本就消费（`recordStories` → `eventLogs` → 落库 →
   `GetEventHistory`/`/debug/game/events`），无需改动。sim 分布无变化（纯反馈通道）。
6. 【已实现（作者立项，`9923dce`/`393a385`/`18e3b47`）】**楼层/碎片对内容事件的可达性**：

   **碎片**：内容效果封闭集新增 `profession_fragment_add` / `profession_fragment_remove`
   （`Fragment` 字段；add 的 profession 空=补给碎片最少部门、fragment 空=种子随机抽未知碎片，
   均确定性可复现）。机制层变异本就有同名实现，桥只是接入内容面。
   **可达性实测**：引擎测试 `TestFragmentBridgeMakesInformationEndingReachable` 证明连发
   laggard 补给可把全部门抬到 ≥5 碎片并触发 INFORMATION；**但无人值守 sim 中 INFORMATION
   仍为 0%**——内容侧碎片事件（`network_data_cache` 2 枚 + `archive_salvage_operation`
   约 7 枚）不足以把 9 个部门全部补到 ≥5（缺口约 19 枚），该结局仍以玩家/NPC 的
   SHARE_INFO 为主路径。这是实测现状，不是缺陷。

   **楼层**：消费层已立（`core_systems.js`）——(a) 产出：各部门效率 ×(0.5+0.5×本区平均
   稳定度)，稳定度 1.0 时因子=1 完全中性；(b) 死亡：全堡平均稳定度 <0.8 时每低 1.0 加
   +0.25/年 死亡率（上限 +0.20/年）。内容入口 `floor_stability_delta`（zone 过滤，0–1 截断）。
   衰减来源=互调后的 `water_leak`/`blackout_cascade`（Lower −0.04）与 `lower_deck_collapse`
   （Lower −0.08），修复渠道=`structural_reinforcement`（耗 600 Materials，Lower +0.10）。
   实测基线保持：无衰减时分布逐位不变；接入衰减后 avg pop 4480→3197（见 §7 对照）。
   前端无楼层展示面，按作者指示不新造 UI。
7. 【已修复 `51d312f`】**延迟队列事件缺 `story_sink` 的空指针崩溃**：`emitContentEvent` 对
   nil `stories` 计数解引用加防护（`handleContentEvent` 本就 nil 检查 sinks、故事经
   content_source 路径照常记录）。回归测试按延迟派发原样构造无 sink 事件 + `event_triggered`
   订阅者复现原崩溃路径；真实内容路径由 `TestDelayedDispatchFollowupDoesNotCrash` 锁定。
   `grief_release_ritual` 对 `blackout_cascade` 的订阅已在互调中恢复（`bfcbada`）。

## 7. 全局互调轮（2026-11-12，作者授权改存量）

目标与验收：旧事件 + 18 新事件 + 规则事件互调成一张网；结局不再 100% 单一；触发分布更均匀；
同种子同局数前后对照。实现提交：`bfcbada`（存量互调）/ `18e3b47`（4 个新事件）/ `57d080f`（测试）。

### 7.1 量级标尺（本轮统一）

| 杠杆 | 小 | 中 | 大 | 备注 |
| --- | --- | --- | --- | --- |
| rebellion | +0.03 | +0.05 | +0.08 | 滚雪球门槛事件 +0.10 为唯一例外（终局机制） |
| panic | ±0.04 | ±0.06 | ±0.08 | 泄压渠道 −0.06~−0.10 |
| history_burden | ±0.03 | ±0.05 | +0.08（仅 broadcast） | |
| 资源 | ±300 | ±600 | ±1000 | |
| 人口 | −40 | −80 | （−0.5%/月 规则档） | |
| 稳定度 | ±0.04 | ±0.08 | +0.10（修复） | 仅 Lower 区承受衰减 |

派生指标（cohesion/legitimacy/dept_tension/class_fragmentation/部门思潮）的 delta 已全部
从存量事件中清除，换成持久杠杆。

### 7.2 因果网与母题分工

- 记忆政治：broadcast →(hb≥0.08) awakened →(hb≥0.12) purge →(+4月) backlash；neighbor_silence
  →(订阅) network_data_cache（碎片×2）。
- Silo 1 线：ultimatum(Y126) →(+3月) aftermath；destroyed(Y130) → fallout（去掉派生凝聚力门槛）。
- 能源线：blackout(cd12) →(+2月) cascade（食物−300 + Lower 稳定度−0.04）；geothermal 为馈赠变奏。
- 材料线：cannibalization(cd12，第5次质变) →(+4月) lower_deck_collapse（人口−80 + Lower −0.08）；
  water_leak 降级为被动衰变变奏（cd9，Lower −0.04）。
- 食物线：food_poisoning（门槛收紧 ≤2000、cd9、人口−40）；mutant_strain →(+6月) side_effects；
  supplies_shortage 规则改三档温水煮青蛙（<1500/<800/<200，cd4，只注恐慌不注叛乱）。
- 恐慌线：panic_storm(cd9) →(+3月) delayed_riot——暴乱重设计为"恐慌转化为政治压力"：
  rebellion +0.04、panic −0.06（泄压）、人口 −80。
- 互助线：grief_release（cd9，订阅 food_poisoning/water_leak/blackout_cascade，恐慌 −0.08、
  叛乱 −0.02）。
- 禁忌线：cleaning_sentence →(+5月) witness_unrest。
- 劳动线：conscription(cd9) →(+3月) backlash。
- 流言节点：rumor 增加前置 event_triggered(awakened)——先有叙事种子再有流言网。
- 终局门槛：underground_coordination（§6.3）。

### 7.3 sim 前后对照（同参数：20 局，种子 7–26，IT，months 200）

| 指标 | 互调前 | 互调后 | 说明 |
| --- | --- | --- | --- |
| 结局分布 | TIME 100% | **TIME 70% / REBELLION 30%** | 40 局复测 65%/35% |
| 平均终局人口 | 4480 | 3197 | 楼层衰减 + 暴乱/镇压伤亡 |
| 平均终局叛乱 | 0.541 | 0.980 | 门槛滚雪球后双峰收敛到 1.0 附近 |
| 平均结束月份 | 90.0 | 88.9 | 叛乱局提前结束（77–88 月） |
| supplies_shortage 触发 | 63.35/局 | 20.40/局 | 三档化 + cd4 |
| water_leak | 16.95 | 10.95 | cd9 + 门槛 |
| food_poisoning | 14.45 | 9.55 | cd9 + 门槛收紧 |
| 新增事件 | — | underground 2.85、salvage 7.0、cache 1.0、reinforcement 0.60 | reinforcement 依赖材料富余，稀疏属预期 |

无"从不触发"事件；`silo1_destroyed_signal`/`silo1_fallout` 降至 0.75/0.70 是叛乱局提前结束
（未到 Y130）的自然结果，非失效。INFORMATION 仍 0%（见 §6.6 说明）。

## Sources

- 机制事实：`internal/engine/content_events.go`（触发/效果/校验）、`internal/engine/mechanics.go`
  （变异应用）、`events/mechanics/core_systems.js`（每 tick 重设逻辑、victory_check）、
  `internal/service/game_service.go`（加载白名单、调度与订阅）、`events/README.md`（DSL 约定）。
- 量化基准：RimWorld/Frostpunk 数值研究（本地归档，未随仓库发布；26 来源，[事实]/[社区反推]/[推断] 分级）。
- 设定依据：本目录 01（结构/禁忌/处置机制）、02（小说纪年与改编侧阶段）、05（桥段映射）篇；
  Silo 40 史实（2212 断联战争）见 01 篇 §6.2【原著明确】。
- 基线数据：本轮只读探针实测（种子 7、IT 特工、无玩家动作、97 个月）。

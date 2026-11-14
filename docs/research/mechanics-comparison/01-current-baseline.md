# 横向机制对照与 Silo40 借鉴建议 · Silo40 现状基线

> 拆分自 [横向机制对照与 Silo40 借鉴建议](../mechanics-comparison.md)；目录、章节映射与出处见该索引。

## 1. Silo40 现状基线

### 1.1 回合管线与时间模型

- **按月 tick**：每次 PassTime 推进 1 个月（`deltaYears = 1/12`）。管线顺序硬编码在
  [engine.go runTickPipeline](../../../internal/engine/engine.go#L269-L316)：
  `AGENT_UPDATE → RESOURCE_UPDATE → IDEOLOGY_UPDATE → NPC 回合 → METRICS_UPDATE → VICTORY_CHECK → 时间推进 → 触发器/随机事件 → 内容事件`。
- **0 月行动**：`GATHER_INFO` / `SHARE_INFO` 不推进时间，只执行 NPC 回合 + 胜利判定
  （[engine.go EnqueueInstantResolution](../../../internal/engine/engine.go#L190-L204)）。
- **长时间尺度**：500 年倒计时（`Countdown`）；脚本化里程碑用游戏内时间戳触发
  （Year 122 档案广播、Year 130 一号地堡失联，`events/histories/*.js`，`timestamp_at_or_after`）。
- JS 主路径 + Go 降级副本：核心系统事件（`events/mechanics/core_systems.js`）与 Go 侧
  同语义 fallback 并存，由 `victory_fallback_test.go` 锁定语义一致性。

### 1.2 状态与资源建模

- **三层粒度**：Silo 宏观指标 ← Profession 部门（9 个部门、总人口 10,000）←
  `PopulationCohort`（每部门按 `pro_foreign` 切 Low/Medium/High 三 bucket）+ 关键
  `Resident`（每 cohort 一名 delegate）。没有个体级市民模拟。
- **宏观指标**：`Legitimacy` / `Cohesion` / `Rebellion` / `HistoryBurden` /
  `DeptTension` / `ClassFragmentation` / `SafeguardRisk` / `EventTrigger`，全部 0–1
  （除 Countdown/EventTrigger）。
- **资源**：Energy / Materials / Supplies。人均消耗 1.0/年；产率 `4.5 × 工人数量/年`
  （[engine.go L1206](../../../internal/engine/engine.go#L1205-L1207)，
  注意 `references/balance.md` 写的 2.1 已过期，以代码为准）；转化链：Agricultural
  耗 Energy→Supplies、Supply 耗 Materials→Supplies（1:1，受劳力与原料双限）；
  `Rebellion > 0.7` 时生产效率降至 0.3。
- **人口**：基础死亡率 0.001/年；每种枯竭资源 +0.05/年；`Rebellion > 0.8` 额外死亡；
  死亡按 cohort 人数比例扣除并移除空 cohort。

### 1.3 思潮与派系

- **cohort 思潮演化**（[engine.go updateIdeology](../../../internal/engine/engine.go#L1092-L1180)
  + `core_systems.js ideology_update`）：`pro_foreign` 受恐慌驱动——部门恐慌 > 0.3 且
  凝聚力 < 0.5 时漂移 `panic × (1−cohesion) × 0.01/年`，另有 `panic × 0.10/年` 的转化项；
  恐慌自身每年衰减 10%。`psychoactive_meds` 地堡特质会让全体思潮向 IT 部门收敛。
- **忠诚度双段更新**：cohort `loyalty` 先向 `legitimacy − panic×0.35×敏感度 + 阶层偏差`
  收敛（0.45/年），派系聚类后再做二次回调——低合法性惩罚 dissent 阵营、高合法性利好
  order 阵营（[residents.go applyFactionLoyaltyDynamics](../../../internal/engine/residents.go#L1114-L1151)）。
- **派系聚类**（每回合 NPC 阶段重建，
  [residents.go RebuildImplicitFactions](../../../internal/engine/residents.go#L658-L907)）：
  signature = `pro_foreign` 档位（Medium/High）+ drive 标签；政治化条件 = `pro_foreign ≥ 0.7`
  或存在潜在领袖（`Ambition ≥ 0.62 且 Influence ≥ 0.25`）；最小 200 人；部门间关系
  < 0.05 拒绝同阵营；未成型者并入 `Unaffiliated`（0 政治影响力）。
- **派系凝聚力** = 内部部门关系均值 × 0.6 + 领袖威望修正（≤0.4）− 规模惩罚
  （`log10(pop/100+1) × 0.05`）；代表按 `影响力×0.4 + 野心×0.2 + 忠诚×0.2 + 威望×0.15 + AP×0.1` 选出。
- **政治指标派生**：`Cohesion` = 阵营凝聚力按影响力加权；`Legitimacy = Cohesion`；
  `Rebellion` 由激进阵营（凝聚力 < 0.4）影响力占比超 0.05 的部分累积，否则 −0.02/年。
- **阵营可见性**：对每个观测者分 Hidden/Aware/Visible 三级（关系阈值 0.15/0.4，
  公开阵营恒可见）——Silo40 已有"信息不对称"基础设施。

### 1.4 事件系统（`events/` 文件驱动）

- **分组**：`histories/`（脚本化里程碑）· `special/`（条件特殊事件）· `crisis/`
  （special 的后续升级，经 `event_triggered` 唤起）· `rules/`（常驻规则，带冷却）·
  `mechanics/`（每回合核心系统与公式）· `player_actions/`（文件驱动玩家动作，当前目录未建）。
- **触发器**：`timestamp_at_or_after`、指标/资源/人口阈值、flag、`event_triggered`、
  `player_action_is`、部门指标/思潮计数、`all/any` 组合（[content_events.go](../../../internal/engine/content_events.go#L352-L399)）。
- **效果**：silo 指标/资源/人口 delta、部门恐慌/思潮 delta、flag 置位、
  `schedule_event`（延迟 N 月）——**延迟后果已有原语**。
- **火力模式**：`ONCE` / `REPEATABLE + cooldown_months`。
- **现有规则事件**：`panic_storm`（任一部门恐慌 > 0.7 → 3 个月后暴乱，冷却 6 月）、
  `rebellion_escalation`（叛乱 > 0.6 → 每月损失 0.5% 人口）、`supplies_shortage`
  （Supplies < 200 → 凝聚力 −0.02、亲外 +0.02，冷却 1 月）。
- **脚本边界**：`canTrigger` 只做判断；`apply` 只返回声明式 effects/emit，不能直接改 Go 状态。
- **关键缺口**：没有自适应事件导演。所有事件纯条件触发；旧随机剧情池已清空
  （[story_events.go L29](../../../internal/engine/story_events.go#L26-L30)），
  `EventTrigger` 累积值（`(1−cohesion)×0.1/年`）不再有可触发事件，达到 1.0 会命中空池采样
  （`rng.Intn(0)`，潜在崩溃）。

### 1.5 玩家与 NPC

- **玩家 = 秘密特工**：AP 上限 100、恢复 `10 + 威望×0.05`/年；行动有 AP 成本 + 月数耗时；
  怀疑度随行动增长（职业倍率：Mayor ×2.0、IT/Mines ×0.3、Police ×0.5~0.9 随机），
  每年自然衰减 0.05；`suspicion ≥ 1.0` → `AGENT_COMPROMISED` 失败结局。
- **信息碎片**：9 部门 × 5 碎片；`INFORMATION` 胜利 = 每个部门各掌握 ≥5 个他部门碎片。
- **NPC**：`NpcBrain` 效用加权随机（每年约 1.2 次行动尝试），部门与阵营代表共用同一
  动作管线；代表有强动机公开阵营（权重 2.5）。

### 1.6 随机性与可复现性

- 单一可注入随机源（[rng.go](../../../internal/engine/rng.go)）；种子随存档持久化
  （`Silo.RNGSeed`）；`SILO40_SEED` 派生每局种子；同种子 + 同操作序列可复现；
  JS 侧 `randomFloat/randomInt` 同源；浮点求和顺序显式固定（派系凝聚力排序后求和）。
- `cmd/sim` 无头模拟器可批量跑结局分布（固定种子整批可复现）。

### 1.7 结局判定（JS 主路径 + Go fallback）

判定优先级（[core_systems.js victory_check](../../../events/mechanics/core_systems.js#L325-L414)）：

| 优先级 | 条件 | 结局 | 胜负 |
|---|---|---|---|
| 1 | `SafeguardRisk ≥ 1.0` | DEATH（Safeguard 清洗） | 败 |
| 2 | 所有部门碎片 ≥5 | INFORMATION | 胜 |
| 3 | `silo1_destroyed` flag | TIME | 胜 |
| 4 | `suspicion ≥ 1.0` | AGENT_COMPROMISED | 败 |
| 5 | `rebellion ≥ 1.0` | REBELLION | 胜 |
| 6 | 人口 ≤ 0 | DEATH | 败 |

- **评分** = (生存 + 多样性 + 历史包袱 + 思潮) × 结局倍率（INFORMATION ×2.0、TIME ×1.5、
  REBELLION ×1.2、EXCLUSIONIST ×0.5、DEATH/COMPROMISED ×0）。
- 结局叙事按 `pro_foreign` 部门占比与 `HistoryBurden` 附加段落。

### 1.8 玩家反馈通道

- `agent_stats` 由后端下发（AP 恢复、威望拆解、宣传倍率、叛乱基础效果等），前端不复算公式。
- 每季度输出政治日志（凝聚力/叛乱/部门紧张度）；`StoryEventLog` 记录事件历史；
  快照含 `cohorts`/`factions`（完整 residents 被裁剪）。
- 隐藏状态：他阵营精确凝聚力/影响力、他部门恐慌值等不完全对玩家暴露（可见性模型）。

### 1.9 硬编码 vs 脚本边界

- **Go 硬编码**：管线顺序、延迟任务调度、世界生成、派系聚类、NPC 决策、存档结构。
- **events/ 可覆写（免编译）**：事件数值/触发条件/冷却、行动成本与耗时、
  `agent_stats`/`score` 公式、既有结局类型内的判定条件。


# 横向机制对照与 Silo40 借鉴建议

> 研究日期：2026-11-12。
>
> 配套文档：[逐游戏机制拆解](game-mechanics-breakdown.md)（来源编号 [A*]/[B*]/[C*]/[D*] 见该篇 Sources）。
>
> 本文所有 Silo40 现状描述均以当前代码为准（`internal/engine/`、`events/`、`internal/service/`），引用格式为 `文件:行号`。

## 1. Silo40 现状基线

### 1.1 回合管线与时间模型

- **按月 tick**：每次 PassTime 推进 1 个月（`deltaYears = 1/12`）。管线顺序硬编码在
  [engine.go runTickPipeline](../../internal/engine/engine.go#L269-L316)：
  `AGENT_UPDATE → RESOURCE_UPDATE → IDEOLOGY_UPDATE → NPC 回合 → METRICS_UPDATE → VICTORY_CHECK → 时间推进 → 触发器/随机事件 → 内容事件`。
- **0 月行动**：`GATHER_INFO` / `SHARE_INFO` 不推进时间，只执行 NPC 回合 + 胜利判定
  （[engine.go EnqueueInstantResolution](../../internal/engine/engine.go#L190-L204)）。
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
  （[engine.go L1206](../../internal/engine/engine.go#L1205-L1207)，
  注意 `references/balance.md` 写的 2.1 已过期，以代码为准）；转化链：Agricultural
  耗 Energy→Supplies、Supply 耗 Materials→Supplies（1:1，受劳力与原料双限）；
  `Rebellion > 0.7` 时生产效率降至 0.3。
- **人口**：基础死亡率 0.001/年；每种枯竭资源 +0.05/年；`Rebellion > 0.8` 额外死亡；
  死亡按 cohort 人数比例扣除并移除空 cohort。

### 1.3 思潮与派系

- **cohort 思潮演化**（[engine.go updateIdeology](../../internal/engine/engine.go#L1092-L1180)
  + `core_systems.js ideology_update`）：`pro_foreign` 受恐慌驱动——部门恐慌 > 0.3 且
  凝聚力 < 0.5 时漂移 `panic × (1−cohesion) × 0.01/年`，另有 `panic × 0.10/年` 的转化项；
  恐慌自身每年衰减 10%。`psychoactive_meds` 地堡特质会让全体思潮向 IT 部门收敛。
- **忠诚度双段更新**：cohort `loyalty` 先向 `legitimacy − panic×0.35×敏感度 + 阶层偏差`
  收敛（0.45/年），派系聚类后再做二次回调——低合法性惩罚 dissent 阵营、高合法性利好
  order 阵营（[residents.go applyFactionLoyaltyDynamics](../../internal/engine/residents.go#L1114-L1151)）。
- **派系聚类**（每回合 NPC 阶段重建，
  [residents.go RebuildImplicitFactions](../../internal/engine/residents.go#L658-L907)）：
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
  `player_action_is`、部门指标/思潮计数、`all/any` 组合（[content_events.go](../../internal/engine/content_events.go#L352-L399)）。
- **效果**：silo 指标/资源/人口 delta、部门恐慌/思潮 delta、flag 置位、
  `schedule_event`（延迟 N 月）——**延迟后果已有原语**。
- **火力模式**：`ONCE` / `REPEATABLE + cooldown_months`。
- **现有规则事件**：`panic_storm`（任一部门恐慌 > 0.7 → 3 个月后暴乱，冷却 6 月）、
  `rebellion_escalation`（叛乱 > 0.6 → 每月损失 0.5% 人口）、`supplies_shortage`
  （Supplies < 200 → 凝聚力 −0.02、亲外 +0.02，冷却 1 月）。
- **脚本边界**：`canTrigger` 只做判断；`apply` 只返回声明式 effects/emit，不能直接改 Go 状态。
- **关键缺口**：没有自适应事件导演。所有事件纯条件触发；旧随机剧情池已清空
  （[story_events.go L29](../../internal/engine/story_events.go#L26-L30)），
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

- 单一可注入随机源（[rng.go](../../internal/engine/rng.go)）；种子随存档持久化
  （`Silo.RNGSeed`）；`SILO40_SEED` 派生每局种子；同种子 + 同操作序列可复现；
  JS 侧 `randomFloat/randomInt` 同源；浮点求和顺序显式固定（派系凝聚力排序后求和）。
- `cmd/sim` 无头模拟器可批量跑结局分布（固定种子整批可复现）。

### 1.7 结局判定（JS 主路径 + Go fallback）

判定优先级（[core_systems.js victory_check](../../events/mechanics/core_systems.js#L325-L414)）：

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

## 2. 横向对比表（机制维度 × 对照游戏 × Silo40）

| 维度 | Frostpunk | TWoM | CK3 | Democracy 4 | RimWorld | Cultist Sim | Suzerain | Reigns | **Silo40 现状** |
|---|---|---|---|---|---|---|---|---|---|
| 状态粒度 | 城市级双条 + 人口类别计数 | 个体级（具名角色+隐藏 morale） | 个体角色（opinion 矩阵） | 2,000 模拟选民 + 群体 | 个体 pawn（needs/thought） | 卡牌（万物皆卡） | 隐藏 flag 数百个 + 四维意识形态追踪 | 四条全局计量 | **cohort 级（部门×思潮档位）+ 关键居民代表**，介于两者之间 |
| 核心驱动 | 法律树 + 温度日程 | 昼夜双相位 + 稀缺 | 角色寿命 + 派系累积 | 政策滑条 + 选举周期 | 事件导演 + 财富复利 | 动词计时器 + 合成 | 读文本—做选择—看新闻 | 抽卡—二元选择 | **月 tick 管线 + 玩家特工行动 + 条件事件** |
| 时间推进 | 实时可暂停，小时刻度 | 实时可暂停，天刻度 | 可暂停实时，月度结算 | 回合=季度 | 连续 tick（60/秒） | 真实时间，动词各异 | 离散 turn（≈4 月/turn） | 一卡=一年 | **月 tick（1/12 年）+ 0 月即时行动** |
| 随机性 | 天气脚本化、可背板 | 战争长度开局预定 | 种子存档内，读档复现 | 确定性公式为主 | 种子仅地形；导演调度有保底 | 抽卡随机、无种子证据 | 【推断】确定性 flag | 概率卡池、不可复现 | **单一 RNG 源 + 存档种子 + 无头模拟器，可复现性最强一档** |
| 延迟后果 | 法律不可逆 + 温水煮青蛙 | 道德创伤滚雪球 | 派系 discontent 累积→最后通牒 | inertia 移动平均（≤32 回合） | thought 时限 + adaptation 橡皮筋 | 潜伏链（证据链/3 张 Dread） | flag 累积 + 阈值组合判定 | 压力眨眼 + 跨朝代解锁 | **schedule_event + event_triggered 链 + 冷却**（原语已有，用量少） |
| 失败/结局 | 希望/不满最后通牒；结局审判法律 | 全员死亡/离开；停火即胜 | 无胜利条件，涌现结局 | 选举失败/暗杀 | 全灭；5 种过职员表结局 | 计数达标式失败；健全/疯狂双胜利 | 9 大结局 + 25+ 子结局 | 任一计量触底/触顶即死 | **6 类结局（优先级判定）+ 评分倍率 + 叙事段落** |
| 反馈通道 | 双计量条 + 天气预报 | 文字状态行 + 台词，无数值条 | 嵌套 tooltip + 全分解 | 因果网络图 + 民调 + 焦点组 | mood=thought 纯函数，全可解释 | 刻意隐晦，面包屑暗示 | 报纸/报告/面板，定性为主 | 图标填充度，无数字 | **agent_stats 下发 + 季度日志 + 事件历史**；他阵营隐藏 |
| 刻意不做 | 个体模拟、战斗、即时说教 | 战斗爽感、即时道德评分 | 国家模拟、微观经济 | 外交建模、简化信息过载 | 脚本主线、强制胜利、真随机 | 教程、任务日志、显式数值 | 数值国家模拟、读档回滚 | 数值透明、分支树、多选项 | **个体市民模拟、空间/战斗、政策明面立法**（玩家走暗线） |

## 3. 三类结论

### 3.1 Silo40 已有的机制（可对照，无需重做）

| # | 机制 | 对照标杆 | Silo40 现状（代码事实） |
|---|---|---|---|
| 1 | 群体级思潮/情绪计量 | Frostpunk Hope/Discontent [A1] | `Cohesion/Legitimacy/Rebellion` + 部门 `PanicValue`，cohort 粒度演化（`engine.go:1092-1180`） |
| 2 | 派系自组织与诉求 | Frostpunk Londoners [A5]、CK3 factions [B2] | signature 聚类 + 代表选举 + 可见性模型 + 公开/威望（`residents.go:658-907`） |
| 3 | 延迟后果原语 | RimWorld incident queue [C11]、Suzerain flag 链 [D5] | `schedule_event`（延迟 N 月）+ `event_triggered` 链（`content_events.go:421-427`）；实例：`panic_storm → delayed_riot` |
| 4 | 冷却/火力模式 | RimWorld incident cooldown [C5] | `ONCE` / `REPEATABLE + cooldown_months`（`content_events.go:448-457`） |
| 5 | 隐藏计量 + 极值即死 | Reigns 四计量 [D12] | `suspicion ≥ 1.0 → AGENT_COMPROMISED`、`SafeguardRisk ≥ 1.0 → DEATH`（`core_systems.js:333-388`） |
| 6 | 确定性复现 | CK3 存档种子 [B14]、RimWorld 世界种子 [C12] | 单一 RNG 源 + 种子随存档 + `cmd/sim` 批量验证——**比多数标杆更完整** |
| 7 | 数据驱动事件/公式 | Democracy 4 CSV 对象-效果图 [B26] | `events/**/*.js` 免编译覆写数值/触发/公式（README"调内容"节） |
| 8 | 结局回读雏形 | Frostpunk 法律审判 [A4]、TWoM Epilogue [A13] | 评分分项 + 结局叙事按思潮/历史包袱变体（`engine.go:897-987`） |

### 3.2 低成本可借鉴（按优先级）

| # | 借鉴点 | 来源依据 | 落点 | 成本/风险 |
|---|---|---|---|---|
| 1 | **事件导演：压力预算 + 冷却 + 危机/馈赠双通道**。规模与频率两条独立管线：规模随状态缩放（类比 raid_points=财富函数 [C4]），频率用周期/保底混合（Cassandra 周期、Randy 保底 [C8]） | RimWorld [C3][C4][C8] | Silo40 缺口明确：所有事件纯条件触发，旧随机池已空、`EventTrigger` 累积无出口（`story_events.go:29`，达 1.0 触发 `rng.Intn(0)` 潜在崩溃）。可在内容事件层做加权抽取 + 冷却，复用现有 `EventTrigger` 作预算 | 中。涉及 `internal/service/` 选取逻辑（属代码改动，需单独立项）；风险：随机事件打乱平衡，用 `cmd/sim` 回归 |
| 2 | **群体情绪档位化显示 + 可分解反馈**：隐藏数值→离散档位文案（TWoM morale 五档 [A10]），任何变化可追溯（RimWorld mood=thought 纯函数 [C2]；CK3"挑战来自掌握系统而非寻找信息" [B3]） | TWoM / RimWorld / CK3 | 不改模拟，只加反馈层：cohort 恐慌/忠诚映射为档位词 + 事件日志说明"为什么"。已有 `agent_stats` 下发管线可复用 | 低。纯 DTO/UI 增量 |
| 3 | **温水煮青蛙式规则事件分级**：同一议题随状态恶化逐级升级（Frostpunk 童工"从此向下滑" [A3]） | Frostpunk [A3][A5] | 纯 `events/rules/` 内容：如 `supplies_shortage` 分三档（<200/<100/枯竭），每档效果与文案递进、各自冷却 | 低。纯内容，DSL 已支持 |
| 4 | **结局回读强化**：结局时逐条回读玩家关键行动/触发过的规则事件（"was it worth it" 式延迟审判） | Frostpunk [A4]、TWoM Epilogue [A13]、Suzerain 结局总结屏 [D11] | `StoryEventLog` 已记录全部事件历史；结局界面按类别汇总 + 评分叙事化。过程中不说教（符合 [A3] "lack of judgment"） | 低-中。前端 + 文案 |
| 5 | **inertia 式思潮惯性**：思潮变更按 N 月移动平均生效（D4 inertia≤32 回合 [B21]） | Democracy 4 [B21][B26] | 近似实现可纯内容：关键思潮变动拆成 `schedule_event` 分期落地；精确实现需引擎小改（记为待决项） | 低（近似）/ 中（精确） |
| 6 | **群体级愤世/麻木双计量**：对管理层承诺反复 → cynicism 累积、长期配给 → complacency（D4 [B30]） | Democracy 4 [B30][B24] | 可先用现有 `loyalty` 动态 + 事件 flag 近似；独立计量需模型扩展（待决项） | 低（近似）/ 中（精确） |
| 7 | **暗线面包屑**：玩家"已有半个答案"时给暗示（Cultist [C16]），碎片系统已有，缺的是暗示文案节奏 | Cultist Simulator [C14][C16] | 纯内容：信息碎片/特殊事件文案加引导性措辞 | 低 |

### 3.3 看着好但不适合本项目规模

| 机制 | 来源 | 不适合的原因 |
|---|---|---|
| 个体角色全模拟（opinion 矩阵、联姻、继承） | CK3 [B5] | Silo40 万人规模必须 cohort 聚合；个体矩阵是 O(n²) 状态，与"玩家是幕后特工"的信息粒度冲突 |
| 空间/战斗/物理模拟 | RimWorld、ONI [D22] | Silo40 是政治叙事游戏，楼层已是抽象职能分布；接空间模拟会淹没核心玩法 |
| 实时卡牌桌面操作 | Cultist Simulator [C14] | UI 范式与回合制策略界面冲突；只借其"暗线+潜伏链"思路，不借交互 |
| 百万字手写分支 | Suzerain [D1] | 内容产能不匹配；其"flag+阈值结局判定"方法 Silo40 已在用 |
| 高频换主角轮回 | Reigns [D12] | 单局短循环与 500 年长程经营冲突；只借"隐性极值即死 + 死法差异化" |
| 个体级涌现叙事沙盒 | Dwarf Fortress [D21] | 无结局结构、无政治决策主线，与多结局目标背道而驰 |

## 4. Top 3 借鉴点（详述）

### 4.1 给事件系统装一个"导演"（压力预算 × 冷却 × 双通道）

**依据**：RimWorld 的事件导演是"规模"与"频率"两条独立管线——规模 = 殖民地状态
（财富/人口）的函数 [C4]，频率 = 周期 + 保底 + 冷却 [C3][C5][C8]；且导演既发危机也发
礼物，不做纯恶意曲线 [C10]。Frostpunk 则证明主线压力应该脚本化、可预知 [A1]。

**Silo40 现状缺口**（代码事实）：当前所有事件纯条件触发；旧随机剧情池已清空
（`story_events.go:29`），`EventTrigger` 累积值（`(1−cohesion)×0.1/年`，
`engine.go:1268`）没有任何事件消费，达到 1.0 会命中 `rng.Intn(0)` 空池采样（潜在
崩溃）。即"压力在累积，但没有出口"。

**建议形态**：主线里程碑（histories）保持脚本化；规则/危机事件引入加权随机选取——
预算用现有 `EventTrigger`（低凝聚力攒得快，语义天然契合），规模按地堡状态缩放，
频率带保底与冷却（复用 `cooldown_months`），危机与馈赠（如外部物资、士气事件）同池。
**注意**：选取逻辑在 `internal/service/` 层，属代码改动，需单独立项并过 `cmd/sim`
结局分布回归。

### 4.2 反馈可读性：档位化 + 可分解，而不是更多数值

**依据**：三个不同品类的标杆收敛到同一原则——TWoM 把隐藏 morale 映射为五档文字状态
而非数值条 [A10]；RimWorld 的 mood 是活跃 thought 的纯函数，任何崩溃都有因可查
[C2][C9]；CK3 官方设计原则"挑战应来自掌握系统，而非寻找信息"，配套嵌套 tooltip 全
分解 [B3]。反面教材是 Reigns：计量全隐藏导致媒体批评"后果不明" [D15]。

**Silo40 落点**：玩家目前能看到 `agent_stats`（自己的数值全分解），但群体侧只有季度
日志一行。低成本做法：cohort 恐慌/忠诚/派系凝聚力映射为离散档位词（麻木/不安/躁动/
一触即发），事件日志统一"现象 + 归因"句式。不改模拟公式，纯反馈层——与"玩家是幕后
特工、信息不完全"的设定兼容（隐藏精确值、暴露趋势档）。

### 4.3 延迟审判：把道德清算留到结局

**依据**：Frostpunk 过程中不即时说教、结局逐条回读法律（"lack of judgment" [A3]、
"was it worth it" [A4]）；TWoM 的 Epilogue 用角色战后命运回读行为 [A13]；Suzerain
1.1 补丁专门追加结局总结屏与 political compass [D11]。三者共同点：**过程中给事实，
结局给审判**。

**Silo40 落点**：素材已齐——`StoryEventLog` 记录全部事件、评分已有四个分项与结局
倍率（`engine.go:897-954`）、结局叙事已有思潮/历史包袱变体（`engine.go:957-987`）。
缺的是"回读"：结局界面按类别汇总玩家关键行动（煽动/宣传/分享未解释知识）、触发过的
规则事件、以及它们与最终结局的因果句。这是内容+前端工作，不动引擎。

## 5. 待决项（涉及设计意图，不在本研究拍板）

1. 事件导演的随机事件是否允许直接改变结局可达性（如随机馈赠避免资源枯竭死局）？
2. inertia 精确实现（引擎小改）vs `schedule_event` 近似（纯内容）：选哪个？
3. 结局回读是否展示"未走过的路"（对比性审判），还是只回读已发生事实？
4. `EventTrigger` 空池潜在崩溃（`rng.Intn(0)`）：在事件导演立项前先修还是合并处理？
   （本研究只记录，不改代码。）

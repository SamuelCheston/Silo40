# 横向机制对照与 Silo40 借鉴建议 · 三类结论、Top 3 借鉴点与待决项

> 拆分自 [横向机制对照与 Silo40 借鉴建议](../mechanics-comparison.md)；目录、章节映射与出处见该索引。

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

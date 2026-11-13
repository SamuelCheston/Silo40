# 架构参考：数据存储与计算逻辑后置到 Go 后端

> 本文档由 AI 维护 (project-source)。`docs/` 为外部来源，不予修改。

## 迁移前 (前端驱动)

- 全部游戏逻辑位于 `frontend/src/logic/`：
  - `engine.ts` (数值演化/胜利/评分)、`eventbus/` (EventBus/Scheduler/RuleEngine/ConditionEngine/ScriptEngine/TriggerEngine)、`npc.ts` (NPC 决策)、`professionActions.ts` (职业行动注册表)、`events.ts` (剧情事件)、`initializer.ts` (初始化)、`models.ts` (模型)
- Go 后端仅提供抽象 CRUD (`internal/service/data_service.go`) + FreeCache，不含游戏逻辑
- `App.tsx` 持有全部游戏状态 (React state)，每次操作深拷贝后交给 TS 引擎原地修改

## 迁移后 (后端驱动)

### Go 后端 (唯一事实来源)

| 包 | 职责 |
| --- | --- |
| `internal/model/models.go` | 后端权威数据模型与 DTO (snake_case JSON)，含 `PopulationCohort`/`Resident`/`Faction`、`VictoryStatus`/`GameScore`/`StoryEvent`/`StoryEventLog`、`AgentAction` 及通信 DTO |
| `internal/engine/` | 移植的全部游戏引擎：事件驱动核心 + 数值演化 + cohort/faction 聚类 + profession 决策/职业行动 + 剧情事件 + 初始化 + 胜利/评分 |
| `internal/service/game_service.go` | 有状态会话服务：内存态 `GameEngine` + Silo/Agent + `StoryEventLog` 历史；FreeCache 会话快照；关键节点落库 |
| `app.go` | Wails 绑定：保留同步接口 `PassTime` / `ExecuteAction`，并新增显式队列接口 `BeginPassTime` / `BeginAction` / `ProcessNextEvent`；其余状态查询接口保持不变 |

### 前端 (纯视图)

- `frontend/src/logic/models.ts`：优先 re-export `frontend/wailsjs/go/models.ts` 生成的 DTO 类型，只保留渲染用常量 (`ACTION_COSTS`/`ACTION_DURATIONS`/`ALL_FRAGMENTS`) 与少量前端 union type
- `frontend/src/App.tsx`：纯调用方，所有状态来自 Go 返回的快照；Header 说明性指标通过后端 `agent_stats` 下发。普通开局显示“40 号派出，现驻 X”，UI 自动 drain 操作队列，并提供驻地/Silo 1 operation、反间谍、Silo 18 正典观察和逐 silo 历史评分视图
- 其余 `logic/*` 实现文件与 `eventbus/` 已删除
- wailsjs 绑定由 `wails generate module` 重新生成

### 关键设计决策

1. **有状态 + 缓存层**：会话状态驻留后端内存；每次操作后 JSON 快照写入 FreeCache (`game_session_v1`)；应用启动 `Resume()` 优先从缓存恢复，其次 SQLite。
2. **结算节点保存**：新建游戏、游戏结束及每个完整结算操作结束时落库 SQLite；`SaveGame` 仅允许无 pending operation 的完整状态。`StoryEventLog` 与世界快照采用同一策略，中途重启可恢复到最近一个完整月份/动作。
3. **事件时间戳语义**：事件历史额外记录整数 `timestamp`，以 `Year 100 / Month 1 / Day 1 = 0` 为基准；当前游戏仍按月推进，因此事件统一映射到“当月第 1 天”，按 `30 天 / 月` 折算，即 `timestamp = ((year - 100) * 12 + month - 1) * 30`。
4. **单活跃游戏**：固定 `World.ID=1` / `Silo.ID=1` / `Agent.ID=1`，部门 `Profession.ID=1..9`；落库为整图替换式 (事务内删旧建新)，避免 PK 冲突，与现有单窗口 UI 匹配。`Silo`/`Agent` 含 `DeletedAt` 字段，删旧必须用 `Unscoped()` 物理删除，否则软删除旧行仍占用固定主键，重建时触发 `UNIQUE constraint failed`。
   - **世界层 (存档 v2)**：`World` 是一局游戏的顶层聚合——主种子、世界时钟 (当前镜像玩家堡时钟)、节奏配置 `WorldPacing` (默认全 1.0，只缩放速率不触碰日历)、Silo 1 占位状态机、空邻堡拓扑。v1 单堡档经 `UpgradeV1ToWorldSave` 显式升级 (备份+事务+回读校验)，空邻堡配置行为逐位不变。
   - **双地堡原型 (存档 v3，P0.5)**：`World` 增 `CrossQueue`/`CrossSeq` (跨堡事件队列，按到期月+Seq 排序)、`Flags` (世界事实白名单)；`Agent` 增 `SiloID` (每堡一个操作者)。`CreateGameRequest.Neighbor` 非空时建第二座**完整模拟堡**并由确定性规则 AI 代操作 (无热座、无第二套 UI)。单堡 (空邻堡) 配置不进入任何多堡路径，行为逐位不变。v2 档经 `UpgradeV2ToTwoSiloSave` 接力升级 (见第 26 条)。**第二堡固定为 18 号地堡并跑原著事件链** (作者裁决，见第 27 条)；配置字段仅保留 API 兼容。
   - **派系持久身份 (存档 v4，P1)**：`Faction` 增 `Stance`/`Memory`/`Demands`；派系 ID 为持久身份 (lineage 继承、新派系单调新 ID，见第 28 条)。v3 档经 `UpgradeV3ToFactionIdentitySave` 接力升级 (备份+单事务+回读校验+失败回滚，旧档字段由签名派生回填)；升级链每步独立留痕、可独立回滚，`UpgradeV2ToTwoSiloSave` 固定写版本 3 而非当前版本。
   - **外派特工身份 (存档 v6)**：可试玩世界固定为驻地 (2–17 号，内部 ID 1)、Silo 1 (ID 2)、Silo 18 (ID 3)。驻地顶替旧 40 号主 silo；40 号不作为实体生成或持久化。`World.PlayerSiloID`、玩家 `Agent.SiloID`、`Agent.StationSiloID` 一致指向驻地。v5 的“40 号实体 + 18 号可操作”语义不可迁移，`Resume` 必须显式拒绝并提示开新局；legacy→v5 升级链仅保留备份/事务/回读校验用途，不代表升级后可续玩。
5. **统一 Actor 管线**：玩家特工、聚合 `Profession`、以及关键 `Resident` 共用 `ActorView` + `SubmitAction`。
6. **生产部门与政治阵营的解耦**：
   - `Profession` (生产部门) 仅承担资源生产力、恐慌值等经济/社会统计；其 `Ideologies` 字段作为宏观分布的统计视图展示；
   - `Faction` (政治阵营) 驱动政治系统：地堡的 `Cohesion` (凝聚力)、`Legitimacy` (合法性) 和 `Rebellion` (叛乱) 指标完全由阵营的影响力 (Influence) 与凝聚力 (Cohesion) 派生；
7. **核心意识形态**：当前核心维度为 `pro_foreign` (亲外) 与 `loyalty` (忠诚)；核心模拟在 `PopulationCohort` (人口单元) 级别进行，而非部门级别；
8. **人口聚合模拟**：初始化按两个意识形态维度的三个等级进行 cohort 切分；P1 起政治阵营按**特征向量距离聚类** (亲外×1.0 / 忠诚×0.5 / 阶层×0.4 加权欧氏距离)，`loyalty` 参与聚类；faction 形成后再进行忠诚度动态回调；
9. **阵营生成与影响**：政治 faction 由 `pro_foreign` 强度门槛 (进 0.7 / 出 0.6 滞回带) 准入，距离聚类成组；总人数不足 `MinFactionSize` 的簇并入 `Unaffiliated`；签名 (质心档位+最大成员 drive) 每 tick 派生，只供展示/命名与既有消费方，不再是分组键；
10. **代表机制**：每个 faction 通过动态选举产生一个 representative；
11. **政治系统驱动**：Tick 执行顺序为：资源结算 -> 思潮演化 (Cohort级别) -> 阵营重建 -> **派系忠诚度演化（显式时间缩放步骤 `ApplyFactionLoyaltyDynamics`）** -> 指标结算。`RebuildImplicitFactions` 为纯分组（不改忠诚度）；忠诚度回调按 `1-(1-base)^(12*dt)` 折算到实际时间片，零时长步骤（0 月动作的即时结算）不推进演化。`loyalty` 在 faction 划分之后，基于地堡合法性 (`Legitimacy`) 与 faction 的政治倾向进行动态回调，因此政治指标反映的是“结盟后的忠诚度”而不是静态初值。职业行动 (Profession Actions) 的政治后果通过修改人口忠诚度间接起效。派系聚类的全部可调常数集中在 `engine.FactionConfig`（`internal/engine/faction_config.go`），政治化准入门槛带滞回带（进 0.7 / 出 0.6）。P1 起分组为距离聚类 + 归属惯性，派系为持久实体（稳定 ID/名/立场/记忆/诉求，见第 28 条）。
12. **响应裁剪**：后端在会话/持久化中保留 `cohorts` 与关键 `residents`；UI 快照会回传 `cohorts` 与 `factions` 供前端展示阵营构成，但仍裁掉完整 `residents` 列表以控制响应体积。
13. **派生展示指标后端化**：`GameState` / `TickResult` / `ActionOutcome` 额外包含 `agent_stats`，由 Go 直接下发 AP 恢复、威望拆解、宣传倍率等 Tooltip 所需数值，前端不再重算这些引擎公式。
14. **程序流与领域事件的分离 (Program Flow vs. Domain Events)**：为了解决高频脚本调用下的性能瓶颈与结算顺序的不确定性，系统将“程序逻辑”从 `EventBus` 中剥离。
   - **硬编码流水线 (Hardcoded Pipeline)**：`GameEngine.runTickPipeline` 显式定义了结算顺序（特工 -> 资源 -> 思潮 -> NPC -> 指标 -> 胜利判定 -> 时间推进），不再通过事件订阅编排。
   - **事件系统降级**：`EventBus` 现在仅作为“领域通告器”（如 `EVENT_GAME_OVER`、`EVENT_STORY_EVENT`）和动作入口（`EVENT_ACTOR_ACTION`），不再驱动程序生命周期。
15. **物理变更直接应用 (Direct Mutation)**：物理属性的变更（如资源增减、特工属性修改）不再通过 `EVENT_APPLY_MUTATION` 事件转发，而是由机制引擎执行 JS 后直接调用 Go 函数修改 Model，实现了 Go-JS 的高效交互。
16. **脚本预编译 (Script Pre-compilation)**：为了提升执行效率，JS 脚本在加载时被预编译为 `goja.Program` 字节码并缓存，避免了每次执行时的解析开销。
17. **文件驱动事件分组**：文件驱动事件统一存放在 `events/` 根目录下，并按一级子目录分组；当前主格式为 `events/**/*.js`，每个文件通过 `defineEvent({...})` 导出一个事件定义。后端 bootstrap 会自动扫描这些分组目录，并在数据库中停用已不存在的旧定义，避免目录重组后重复加载旧事件。
18. **事件分类字段**：运行时 `StoryEvent` / `StoryEventLog` 现在显式携带 `category`，用于区分 `history` / `special` / `crisis` / `player_action`。文件驱动事件的 category 由目录分组派生；旧的 `events:*` 键会按兼容规则映射到 `special`。
19. **文件驱动事件总线化**：文件驱动事件不再由 service 直接 apply；运行时会先 enqueue 一个按事件名命名的 `EventBus` 事件，再由注册好的 handler 在后续 signal 中应用效果并记录结果。推荐事件名格式为 `category:event_id`。`crisis` 作为 `special` 的后续事件，会通过监听上游事件名在同一条因果链里被唤起。
20. **玩家动作列表后端化**：`GameState` / `TickResult` / `ActionOutcome` 现在会统一返回 `available_actions`。这个列表会合并现有静态动作（通用动作、职业动作、阵营领袖动作）和 `events/player_actions/` 中声明的文件驱动动作，前端不再硬编码动作分组。
21. **文件驱动玩家动作元数据**：`events/player_actions/*.js` 除了 `trigger` / `effects`，还可以通过 `player_action` 字段声明 UI 元数据，例如 `scope`、`target_type`、`ap_cost`、`duration_months` 与 `unavailable_behavior`。当前 `profession_group` 复用职业 `ClassType`，`faction_member` 作用域先保留接口但不启用，`faction_leader` 由 `AgentStats.is_faction_leader` 判定。
22. **事件脚本边界**：事件定义支持受限 `script` hook。`script.canTrigger(ctx)` 只做额外触发判断；`script.apply(ctx)` 只能返回额外结构化 `effects` 和 `emit` 目标，具体状态修改与后续事件发射仍由 Go 侧统一执行，避免脚本直接改写运行时对象。
23. **Linux 渲染路径显式配置**：`main.go` 显式设置 `options.Linux.WebviewGpuPolicy = WebviewGpuPolicyOnDemand`。Wails v2 在 `options.Linux == nil` 时缺省 `Never`，会使 WebKitGTK 关闭 accelerated compositor、滚动/动画逐帧 CPU 重光栅（审计实测 12.3× 绘制成本）。前端侧配套约束：避免大面积/大 blur 半径的 `box-shadow`、禁止 `transition: all`（改定向属性）、canvas 动画禁止每帧 `shadowBlur`（预渲染 offscreen 后 blit）；转盘实现见 `frontend/src/components/PrizeWheelLocal.tsx`。详见 `docs/research/08-linux-frontend-rendering-perf.md`。
24. **延迟事件队列确定性派发**：`GameService.checkDelayedEvents` 按任务创建序号（`delay#N` 的数字后缀）排序后再递减/派发，保证同月多个延迟任务到期时按"先调度先触发"的固定顺序入队。此前直接 `range` map 导致同种子长跑（约 100 月后延迟任务同月到期）结果不可复现，违反"同种子 + 同操作序列可复现"契约。锁定测试：`internal/service/delay_determinism_test.go`。
25. **LLM 辅助 NPC 决策（可选，默认关闭）**：`SILO40_LLM_MODE=shadow|on` 启用。引擎侧 `NpcBrain.Decide` 挂可选 `NpcPolicy`（双通道建议：候选打分 + 意愿系数，均 clamp；`Force` 用于回放，加权抽样照常消耗以保 rng 流次序）；service 侧 `internal/llm` Advisor 按参与度四档（off/events/key/full，锚定现有事件类别与计量阈值）门控决策点，`llm_decision_logs` append-only 表记录每次介入（含模型标识）与参与度切换，Resume 按日志回放、断序/指纹冲突显式失败。shadow 对轨迹零影响、on 的可复现性由日志承担；`cmd/sim`/parity/锁定测试不受影响。详见 [LLM NPC 决策](../references/llm-npc.md)。
26. **双地堡世界 tick (P0.5)**：`runOneMonth` 按 `multiSilo()` 分流——单堡走原 `UpdateSiloState` 组合 (逐位不变)；多堡走 `runOneMonthWorld` 的分相编排：阶段 0 AI 代操作 (确定性规则，零时长动作) → **阶段 0c 18 号正典链条** (时间戳强制触发，权威覆写 18 号状态，见第 27 条) → 阶段 1 各堡本地结算 (SiloID 升序，流水线 1-6) → 阶段 2a 跨堡队列派发 (**本地结算后、时钟推进前**，按到期月+Seq) → 阶段 2b 一号地堡裁决 (观测各堡叛乱、一次性通牒) → 阶段 3 时钟推进 (世界时钟为权威，各堡镜像) → 玩家 operation 按目标 SiloID 升序结算 (驻地 ID 1 → Silo 1 ID 2) → 阶段 4 各堡后置 (SiloID 升序，流水线 8-9)。
   - **确定性**：单一共享 RNG 流不分片 (方案 A)，靠固定堡序获得复现；跨堡队列按 `(到期月, Seq)` 排序；延迟任务按目标堡分组、在归属堡本地阶段前发布。
   - **子表主键偏移**：引擎在内存中按堡内 1 起始分配子表 ID，落库时对第 `siloID` 堡的 professions/cohorts/factions/residents 主键与堡内外键统一加 `base=(siloID-1)*10000`，读档减回；玩家堡 `base=0` 逐位不变。resources/floors 无外键、ID 数据库自增，无需偏移。
   - **内容效果 scope**：`ContentEffect.Scope` (self/world/silo:<ref>/neighbors/all) 在施加期解析；新增 `cross_signal` (入跨堡队列)、`world_flag_set` (世界事实白名单) 效果与 `world_flag_true/false`、`signal_received` 触发器。`silo1_destroyed` 升为世界事实 (`World.Silo1.Alive` 别名)，`silo_flag_set/silo_flag_true` 对其保留兼容别名。
   - **迷雾**：`WorldView.Neighbors` 下发 `RumorView` (量化档 status + 最后确认时间 + 信道来源)，玩家只知道自己堡的真实值；IT 职业附加半量化档位。
   - **内容**：`events/special/silo18_*.js` / `silo1_ultimatum_cross.js` 均为 schedule-only (`script.canTrigger=false`)，只经跨堡队列投递、从不自发触发，保证单堡奇偶校验不受影响。(原型期的 `neighbor_probe` 链已随 18 号链条落地而删除。)
   - **验收测试**：`internal/service/two_silo_test.go` (双跑逐字节一致/阶段顺序/计量隔离/存档 Resume)、`two_silo_sim_test.go` (存档体积 + 单双堡 sim 对照)、`silo18_chain_test.go` (逐节点触发表/耦合信号实测/具名人物生死/读档防重入)；v6 三堡构成、驻地范围、动作边界、operation 顺序与存档拒绝由 `station_operations_test.go` / `save_version_test.go` 锁定；新三堡逐位基线由 `parity_harness_test.go` 生成。
27. **Silo 18 原著事件链 (第三堡正典)**：可试玩世界第三堡固定为 18 号 (`createSilo18`，带 `silo18_canon` 标记特质)，跑 Wool→Dust 正篇链条 N1–N9 (`internal/service/silo18.go` / `silo18_chain.go`)。
   - **前史承载**：大革命/17 号陷落/Allison 之死不做节点，由开局计量承载 (`history_burden=0.55` 等)；可玩弧段 Y122M2 (Holston 清洁死亡) → Y130M1 (Donald 引爆，与既有 `silo1_destroyed_signal` 合流)。
   - **强制触发 + 权威覆写**：节点按时间戳在阶段 0c 强制生效，直接覆写 18 号计量/人物生死/结局 (N8 设 DEATH)，覆盖 AI 漂移；生效记录写 `contentStates` (key `silo18_chain:N*`)，随 v3 存档持久化、读档防重入——**存档格式不变**。
   - **具名人物是真实个体**：Juliette/Bernard/Lukas/Holston/Jahns/Marnes/Walker/Knox/Shirly/George 为带 `named` 标签的 `Resident`，生死唯一来源是链条节点 (AI 群演只做零时长通用动作；引擎无模拟杀居民路径)。其他堡居民保留系统生成名。
   - **耦合"少数关键、多数独立"**：仅 8 个跨堡信号 (谣言/强信号/广播/断联)，每个有明确因果节点；N8 难民按原著流向 17 号、不入当前驻地；旧“14 种子 9 TIME / 5 rebellion”属于已作废的 40 号主 silo 基线，不作为 v6 分布结论。
   - **零 RNG 消耗**：链条与具名人物创建不碰共享 RNG 流；确定性继承 P0.5 契约 (双跑逐字节一致)。
28. **P1 派系距离聚类与持久身份**：派系从签名分桶重构为距离聚类 + 持久实体 (`internal/engine/faction_cluster.go` / `faction_explain.go`，`RebuildImplicitFactions` 重写)。
   - **距离聚类 (S2)**：cohort 特征向量 = 亲外×1.0 / 忠诚×0.5 / 阶层×0.4 (ELITE=1, COMMONER=0)，加权欧氏距离；并入半径 `ClusterJoinRadius=0.30`，归属惯性 (留在上一 tick 派系) 半径放大 `1+ClusterInertiaBonus=1.5`；敌意部门约束 (双向关系 < 0.05 拒绝同组) 与政治门槛滞回带不变。签名退化为每 tick 派生的展示键 (质心档位 + 最大成员 drive)。双胞胎派系与签名坍缩在距离模型下结构上不可能。
   - **持久身份 (S3)**：lineage 簇继承上一 tick 派系记录 (ID/名/IsPublic/Prestige/Stance/Memory)；新派系单调新 ID 永不复用；无阵营桶同样继承 ID。组序与 S2 严格一致 (规模降序；平局先继承者按上一 tick 名次升序，与旧 lineage 升序单调等价)，保证代表创建的 RNG 消耗序不变——**行为中性，parity 逐字节一致**。`Stance` 创建时固定 (质心档位 join，空则 `status_quo`)；`Memory` 携带 `founded:Y%dM%d`；`Demands` 每 tick 由最大成员 drive 派生 (`demand:<drive>`)。派系因此可被事件系统消费 (稳定 ID/立场/记忆/诉求)：事件脚本上下文新增只读 `ctx.factions`。行动系统未动。
   - **可解释归属 (S4)**：`faction_explain.go` 只读接口，对任意 cohort 输出"到本派系质心距离 vs 次近派系"的解释，与引擎版本无关、不消耗 RNG。
   - **存档 v4 (S5)**：见第 4 条世界层；`UpgradeV3ToFactionIdentitySave` 与既有升级器同构 (备份+单事务+回读校验+失败回滚)，旧档签名回填身份字段，派系 ID 原样保留。
   - **验收数字**：审计 (6 种子×120 月) 双胞胎/坍缩/ABA 恒 0；政治派系数均值 0.38→0.51 (区间上限 0.48→0.78)；S2 激活步结局分布逐种子不变 (5 TIME / 1 rebellion)；S3 后 id_flips 8–13/种子 = 门槛跨越+真实换组 (身份翻转分量归零)；S3/S4/S5 对 S2 基线 parity 逐字节一致。
29. **Relations map 浮点累加确定性 (事故修复，2026-11-13)**：`Relations` 改为 `map[string]float64` 后，`DeptTension` 的部门关系均值 (`engine.go` 政治结算) 与 `ConnectionValues`（威望均值输入，`actor.go`）按 Go map 迭代序累加浮点，加法次序随机 ⇒ 末位 (ULP) 漂移，跨进程同种子复现间歇性逐位不一致（LLM 回放门槛测试捕获：仅 DeptTension 第 16 位有效数字不同）。与第 24 条延迟队列事故同类：**凡对 map 做浮点累加，必须按键排序后进行**。初修 `26cd7fd` 排序关系键；v6 门禁进一步发现跨部门累加还依赖 DB preload 的 `Professions` 切片顺序，现按持久 `Profession.ID` 排序副本后再按关系 key 累加，不改原切片或正常开局顺序。`TestDeptTensionIgnoresProfessionStorageOrder` 锁定顺序无关性，LLM 回放连跑 6/6 通过。
30. **外派驻地与 v6**：玩家是 40 号派出的特工，普通 UI 世界只有驻地、完整模拟 Silo 1（含 Archives）与 Silo 18。驻地由独立随机流从 2–17 号确定，不消耗 gameplay RNG；40 号只保留派出方文案，不生成实体、不写 silo 表、不参与 tick。普通动作与驻地 operation 统一经 `playerActionSilo()` 指向驻地；驻地开放情报/策反/欺骗/内部黑市交易，Silo 1 只开放情报/欺骗，40 和 18 无行动入口。operation 使用服务端权威定价、1–2 月延迟、真实目标 delta 与独立 Exposure/反间谍，月度结算按目标 SiloID 升序。`World.HistoryImpacts` 只记录驻地/Silo 1 已结算行动的实测变化，首次结局一次性加入历史分。存档升级为 v6，v5 显式拒绝，拒绝文案为“旧档基于旧身份模型，无法继续，请开新局”。
   - **parity 基线作废与重建**：旧基线以 40 号为真实主 silo，且包含 18 号 operation，世界构成、RNG 消费路径、行动集合与结局轨迹均不再具有可比性，因此不得宣称与 v5 parity 相同。新基线 `references/parity-v6-three-silo.json` 由 6 种子（7–12）三堡全 service 路径生成，包含驻地身份、三堡 ID/名称、脚本动作、100 月 save/resume 与结局记录；双跑逐字节一致，SHA-256 `272774433cdae5067e6e8cdb68fa8cbb402c873f25133cca441c13fc64af60ed`。
   - **既有启动保护**：空档探测只忽略 GORM `ErrRecordNotFound` 自动日志，真实错误仍显式失败；ready channel 防止早期 `HasActiveGame` 空指针。

31. **行动门控与多实体结局 (v7)**：`events/action_gates.json` 声明有序阶段、动作完成节点/属性/关系条件和行动解锁；`action_gates.go` 统一求值并在快照与两条执行 API 中判定。LOCKED 不下发，UNLOCKED 带 `disabled_reason`，AVAILABLE 才可执行；`World.ProgressStage` 与 `UnlockedActions` 持久化且只向前。玩家开局仅有本职业 2 个碎片，成功动作才写 `player_action:<id>` 完成节点，空转月份不推进阶段。生产启动加载配置，旧机制单测不加载时保留原 fixture。
   - **多主体仲裁**：`WeightedEndingResolver` 在 `VICTORY_CHECK` 后仅处理驻地的非灾难 trigger，JS 小写类型先规范化。玩家、驻地派系、具名关键人物和 Silo 1 按稳定 ID 顺序贡献权重，固定候选顺序解决同分；`VictoryStatus.Decision` 保存主体明细和候选总分。权重与阈值集中于 `weighted_endings.go`，全部 TODO(balance)，不代表已调平衡。
   - **正典边界**：Silo 18 不接入仲裁钩子，N1–N9 时序不变；Silo 1 的 N9 世界覆灭事实优先于其他本地结局，禁止 peer 同步将其复活。耦合测试使用观察 fixture，不要求固定玩家结局类型。
   - **发布与显示护栏**：`syncActionGateConfig` 在已有 events 目录上也同步嵌入注册表，缺失配置显式报错并 fail-closed；`start-playable.sh` 拒绝缺文件的旧包。AP 灰项保留两位小数，避免把 19.925 显示成 20。
   - **存档**：v7 持久化阶段、解锁和仲裁记录；v6 显式拒绝，v5 继续旧身份拒绝。覆盖 `action_gates_test.go`、`weighted_endings_test.go`、engine hook 和 v6 拒绝测试；本轮不做批量对局或 parity 重建。

32. **权威状态骨架 (v8)**：`MayorOffice` 绑定现任具名 Resident 与个人正统性，旧 `Silo.Legitimacy` 仅为兼容镜像，结算统一经 `MayorLegitimacy()` / `SetMayorLegitimacy()`。`ShadowAuthority` 独立保存 W、K/D/P/X/E、Police 服从和 Judicial `A/T/O`；`AuthorityConfig` 集中文档参数。Police 单向因果响应 `O←O+0.35(L−O)`（0.35 为 TODO(balance)）；W 按 `50+25K+12D+10P+12X−40E` 与 8 月半衰期更新，OrderDeviation 仅保留输入位。
   - **可观测输入与边界**：K 来自 Judicial/IT cohort 忠诚，D 暂由 Judicial 恐慌映射，E 暂由 SafeguardRisk 映射；P/X 为 0。每月思潮后、NPC 前更新，零月不漂移，不消耗 RNG。pact 仅有资格/锁定结构，当前无内容能授予资格，也不进入结局。
   - **Judicial 滞回**：`S(W)=1/(1+exp(−(W−38)/3.5))`，`A_next=0.4A+0.6T`；Mayor 需 `W≤30/L≥0.35` 连续 2 月且 `A_Mayor≥0.60`，31–44 保持，`W≥45` 连续 3 月才回 IT。权重始终非负且总和 1，现任者更替进入 SWING 并释放 Mayor 权重。
   - **迁移与 parity**：`UpgradeV7ToAuthoritySave` 按 SiloID 顺序迁移全部堡，备份+单事务+回读校验+失败回滚；Mayor 继承旧 Legitimacy，W 固定为来源 `v7_conservative_w50` 的保守 50。个人正统性、Police/Judicial 关系和 W 已进入月度轨迹，因此 v6 parity 基线自 v8 起失效，本轮未重建。

### 运行时数据流

```
UI 点击 → App.BeginAction / BeginPassTime
  → GameService 记录 PendingOperation (不入队事件，仅记录意图)
  → App.ProcessNextEvent (按需推进流水线)
  → GameEngine.UpdateSiloState (执行 runTickPipeline)
    → 顺序结算：特工 -> 资源 -> 思潮 -> 权威状态 -> NPC -> 指标 -> 胜利 -> 时间推进
    → 发布领域事件 (EVENT_TIME_CONTENT / EVENT_GAME_OVER)
  → GameEngine.Drain (消费 EventBus 中的领域事件)
  → 每步返回完整快照 (Silo/Agent/available_actions/日志/剧情事件/剩余队列)
  → 事件历史追加到 `StoryEventLog`
  → 关键节点 (新建/结束) 落库 SQLite + 更新 FreeCache
  → 前端 setState 渲染
```

### Debug HTTP 镜像接口

- 保留 `Wails` 原生桥接作为主调用路径，不替换现有前端通信。
- 应用启动时额外在本机启动一个 debug-only `HTTP server`，默认监听 `127.0.0.1:18080`。
- 详细使用说明与接口示例请参考 [Debug API 指南](../references/debug_api_guide.md)。
- 地址可通过环境变量 `SILO40_DEBUG_HTTP_ADDR` 覆盖；传空时回退到默认值。
- `HTTP` 路由与 `Wails` 绑定方法基本镜像；当前 debug server 仍保留同步接口，单步队列接口优先从 `Wails` 绑定验证：
  - `GET /debug/health`
  - `GET /debug/game/active`
  - `GET /debug/game/state`
  - `POST /debug/game/create`
  - `POST /debug/game/pass-time`
  - `POST /debug/game/action`
  - `GET /debug/game/ending`
  - `GET /debug/game/events?limit=<n>`
  - `GET /debug/game/profession-actions?profession=<name>`
  - `GET /debug/game/profession-action?id=<action_id>`
- `HTTP` 层不复制业务逻辑，只做 JSON decode/encode 和参数校验，实际调用仍落到 `App`/`GameService`。
- 为了便于本地调试，debug server 开启了宽松 `CORS`；但只绑定本机地址，不作为正式对外 API。

### 验证

- `go build ./...` 通过
- `go test ./internal/service ./internal/api ./cmd/debug ./debug` 通过
- `go test ./internal/engine -run TestEngineSmoke` 通过
- 前端 `npm run build` 通过
- `go test -count=1 ./internal/...` 与关键路径 `go test -race` 全绿
- v6 身份重构后 `go build ./...`、`go vet ./...`、`go test ./...`、前端 build、Wails build 全绿
- 新三堡 6 种子 parity 双跑逐字节一致，SHA-256 `272774433cdae5067e6e8cdb68fa8cbb402c873f25133cca441c13fc64af60ed`；Silo 18 N1–N9 在新三堡下按时触发，旧 v5 档明确拒绝
- 真实桌面干净档 E2E：seed 10 驻地策反 + Silo 1 欺骗均 RESOLVED，TIME 结局历史分 41/C、总分 6723；N1–N9 时间戳正确，SQLite 仅驻地/1 号/18 号三行，退出重启后完整恢复

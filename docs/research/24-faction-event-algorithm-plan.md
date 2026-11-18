# 24 · 派系事件算法实现方案

> 日期：2026-11-17
> 性质：实现方案（裁决已闭合，按阶段原子提交落地）。
> 裁决依据：ADR-0007（带权重随机、阈值不锁死决策）、ADR-0010（玩家是干涉者）、
> ADR-0017（上线前无兼容义务，存档硬切断）、ADR-0018（后端无玩家/NPC 之分，玩家即代表人物）；
> 项目所有者会话裁决（2026-11-17）：废除独立政治威望、ambition 开局定死仅事件可改、
> ambition 非强制门槛仅为权重、发起消耗 AP 无冷却、层级=部门间+阶级间、筹备期解散、
> 相近领袖可合并、领袖死亡加权接任。
> 取代：14 篇 §5.2/§5.3 的"每 tick 重聚类产生派系/成员流动"生成规则与 §6.6 玩家排除条款（后端部分）。
> 14 篇的满意/积怨、需求事件、七挂载节点不在本轮范围。

## 1. 目标算法

派系不再是每 tick 距离聚类的统计副产品，而是**领袖发起的事件产物**，成员资格持久：

1. **发起**：个体（玩家或 NPC 代表）满足 派生威望 ≥ 50 && 意识形态强度 ≥ 0.25 && AP ≥ 20
   → 消耗 20 AP，创立派系（创始人=该 Actor，立场=其思潮，零成员合法）。无冷却、无野心硬门槛。
   NPC 是否发起走带权重随机决策（野心、威望、强度为权重输入）；玩家直选，不掷骰。
2. **感知**：发起时按三通道合成每个观测者的感知度 ∈ [0,1]：
   - 关系通道：观测者对领袖所在部门的关系（居民另取领袖↔代表个人关系，取高者）；
   - 意识形态通道：1 − 思潮距离；
   - 层级通道：PowerLevel 接近度与同阶级加成。
   感知度以 `faction:<id>:founded` 事实写入观测者信念层（reliability=感知度）。
   `faction:` 事实永不成熟、不计入《共证》资格门（沿用 `leak:` 豁免模式）。
3. **传播**：月度口碑扩散（高确信者向关系网传播，每跳 reliability 衰减）；
   GATHER_INFO / SHARE_INFO / Field Gossip / Medical 被动搭载低可靠度派系传闻分支。
4. **加入**：已感知（B ≥ QualRumor）的 cohort 每月加权随机掷骰，意识形态相近度主导（^k）；
   关系 < 0.1 一票否决；跨层级（PowerLevel 差 ≥ 3）需关系 ≥ 0.4；跨阶级需关系 ≥ 0.6；
   部门双向敌意 < 0.05 硬否决。玩家通过 JOIN_FACTION 直选加入，不掷骰。
   退出：意识形态漂移出半径 + 惯性保护的对称掷骰。
5. **生命周期**：成立 18 个月内 MemberCount < 200 → 解散，成员回无阵营桶。
   领袖死亡 → 派系内 `0.5×(prestige/100) + 0.5×ambition` 最高的代表自动接任（平局取 ID 小者）；
   无可接任者 → 解散。
6. **合并**：双方领袖个人关系 ≥ 0.6 && 思潮距离 ≤ 亲缘半径 && AP 成本 → 领袖发起 MERGE_FACTION；
   存续方保留 ID，被并方成员转移、Memory 追加合并记录。
7. **保留**：凝聚力/忠诚动力学（ApplyFactionLoyaltyDynamics）、PUBLICIZE_FACTION、
   持久 ID 与身份继承、事件脚本 factions 只读上下文、可见性三级模型。

距离聚类（faction_cluster.go）降级为**亲缘场**：特征向量与加权距离保留，
只用于计算加入概率与退出压力，不再直接产生派系。

## 2. 属性层裁决

- **政治威望全层派生**：`prestige = 50 × (Ambition + Influence)`（0–100）。
  Agent / Profession / Resident 删除独立 PoliticalPrestige 字段；Cohort 借其代表居民派生。
  Faction.Prestige（组织威望）保留不动。
- **Ambition / Influence 补齐到 Agent 与 Profession**（ADR-0018 同构要求）。
- **Ambition 开局定死**（现状已固化，balance.md），**仅事件可改**：
  新增 `actor_metric_delta/set: ambition` mutation 通路（PLAYER/RESIDENT/NPC 同构）。
- **Influence 可演化**：居民/玩家沿用 `(0.005 + Ambition×0.05)/年` 增长；
  部门向 PowerLevel/10 收敛（沿用 cohort 公式）。
- **年度结算以期初威望为口径**：Go/JS agent_update 先按期初 prestige 结算
  政治点数/AP，再落影响力演化（下一期生效），与 agent_stats 展示口径逐位一致。
- **循环依赖拆除**：generateResidentAmbition 不再以 cohort 威望为输入，改读 PowerLevel。
- **删除两段威望动力学**：cohort 威望漂移（residents.go updatePopulationCohorts）与
  居民威望同步（updateKeyResidents）。
- **prestige mutation 折算**：既有 `political_prestige` delta/set（relics 等）折算为
  influence delta/set（value/50），保持内容语义近似。
- **玩家开局属性**（定死，特质修正）：
  Mayor A=0.8/I=0.8（prestige 80）；Judicial/IT/Medical/Mines A=0.05/I=0.05（prestige 5）；
  其余 A=0.1/I=0.1（prestige 10）；特质 charismatic → A+0.2、I+0.2（prestige +20，对齐旧值）。
- **部门开局属性**：A = 0.2 + 0.03×PowerLevel + 0.025（定死，开局不掷骰——新增 RNG
  消耗会平移共享随机流，破坏种子调优的可玩性测试）；I = 0.6×PowerLevel/10。
- **居民开局 ambition**：AmbitionBase + min(PowerLevel,10)/10×AmbitionPowerWeight +
  ideologicalIntensity×AmbitionIntensityWeight + rng×AmbitionNoise（公式结构不变，输入换轴）。

## 3. 数据模型变更（存档硬切断 v15 → v16）

- Agent：+Ambition、+Influence；−PoliticalPrestige。
- Profession：+Ambition、+Influence；−PoliticalPrestige。
- Resident：−PoliticalPrestige。
- PopulationCohort：−PoliticalPrestige。
- Faction：
  - RepresentativeResidentID → **RepresentativeKind + RepresentativeID**（Actor 引用，玩家可任代表）；
  - +FounderKind、+FounderID、+FoundedYear、+FoundedMonth（筹备期计时）；
  - +LeaderlessSince（可选：无主宽限，用 FoundedYear/Month 同构字段）。
- Resident：+LeaderRelations map[uint]float64（仅领袖↔代表子矩阵，键=对方居民 ID；
  玩家作为领袖时以 AgentID 编码进同一 map，约定键空间：居民 ID 与 Agent ID 不冲突需校验，
  实现时改用 `map[string]float64`，键 `res:<id>` / `agent:<id>`，与部门关系键隔离）。
- 信念层：`faction:<id>:founded` 事实（无新字段，复用 FactBelief）。
- CurrentSaveVersion = 16；旧档拒载（现有校验逻辑天然硬切断，不写迁移）。
  原拟 v15，与 22/23 黑箱系列合并时顺延：v15 已被黑箱学习层占用
  （含 v14→v15 升级步），威望派生硬切断改占 v16，黑箱升级链保留不动。
  **C7 再顺延 v17**：v16 存档的聚类派系/归属是已废除算法的产物，
  语义不可延续；C9 的 LeaderRelations 等新增字段零值安全，不再升版。

## 4. 参数表（FactionConfig 新增/变更）

| 键 | 默认 | 语义 |
| --- | --- | --- |
| FoundPrestigeThreshold | 50 | 发起所需派生威望 |
| FoundIntensityThreshold | 0.25 | 发起所需意识形态强度 |
| FoundAPCost | 20 | 发起 AP 成本 |
| JoinAPCost | 10 | 玩家直选加入 AP 成本（C9 落地决策：低于发起 20，加入是轻动作；与 SHARE/PROBE 同量级，月收入可支撑） |
| JoinAffinityExponent | 2.0 | 加入概率 = 相近度^k |
| JoinRelationVeto | 0.10 | 关系一票否决 |
| CrossTierPowerGap | 3 | PowerLevel 差 ≥ 此值算跨层级 |
| CrossTierRelationMin | 0.40 | 跨层级所需关系 |
| CrossClassRelationMin | 0.60 | 跨阶级所需关系 |
| FactionGraceMonths | 18 | 筹备期月数 |
| MergeRelationMin | 0.60 | 合并所需领袖关系 |
| MergeAPCost | 15 | 合并 AP 成本 |
| SuccessionPrestigeWeight | 0.5 | 接任评分威望权重（ambition 占 1−w） |
| GossipReliabilityDecay | 0.6 | 口碑每跳衰减 |
| RumorPiggybackChance | 0.15 | 信息动作搭载传闻概率 |
| AwarenessJoinFloor | QualRumor(0.30) | 可考虑加入的信念地板 |
| AffinityRadius | 0.30 | 亲缘半径（沿用 14 篇旧聚类加入半径）；加入相近度=1−dist/半径，退出压力按超出部分 |
| ExitInertiaFactor | 0.5 | 退出惯性系数：退出概率=超出度^k×(1−系数)，恒低于加入概率，防单月往返 |
| AmbitionPowerWeight | 0.40 | 原 AmbitionPrestigeWeight 改名换轴 |
| 删除 | — | AmbitionHighThreshold/LeaderInfluenceThreshold 硬门槛语义废除（标签阈值保留为纯 tagging 参数） |

关系三档沿用 BadRelationThreshold=0.05（部门敌意硬否决）。
所有新参数进 FactionConfig，无持久化。

## 5. 阶段拆分与原子提交

每个提交独立编译、全量测试绿；破坏性变更不写迁移（ADR-0017）。

| # | 提交 | 内容 | 主要文件 |
| --- | --- | --- | --- |
| C1 | docs(adr) | ADR-0018 + 索引 | knowledge/adr/ |
| C2 | docs | 本方案 | docs/research/24 |
| C3 | feat(model) | Agent/Profession 补 Ambition/Influence + 开局赋值 + ActorView 访问器 + ambition mutation 通路（增量，不破坏） | models.go、initializer.go、actor.go、mechanics.go、residents.go |
| C4 | refactor(prestige) | 威望全层派生：删独立字段与两段动力学、拆循环依赖、重锚全部消费点（Go/JS/门控/结局/展示）、prestige mutation 折算 influence、存档 v16（原 v15，黑箱层合并顺延） | models.go、actor.go、engine.go、residents.go、mechanics.go、agent_stats.go、initializer.go、action_gates.go、weighted_endings.go、station_operations.go、content_events.go、core_systems.js、formulas.js、save 相关测试 |
| C5 | feat(factions) | Faction 代表 Actor 化 + Founder/成立年月 + 全部消费点适配 | models.go、residents.go、faction_explain.go、mechanics.go、common_actions.js、npc.go、service 快照 |
| C6 | feat(factions) | FOUND_FACTION 动作 + 资格门 + NPC 发起候选（此时自动聚类仍在，C7 切换） | common_actions.js、action_gates.json、npc.go、faction_config.go |
| C7 | refactor(factions) | 事件驱动成员资格：废除每 tick 聚类建派系；RebuildImplicitFactions → 维护管线；亲缘场；筹备期解散；领袖死亡接任；解释接口改新语义；审计测试重建 | residents.go、faction_cluster.go、faction_explain.go、engine.go、faction_test.go、faction_audit_test.go |
| C8 | feat(factions) | 三通道感知 + faction: 信念事实 + 月度口碑 + 信息动作搭载 + MaturedFactCount 豁免 | faction_awareness.go(新)、information.go、game_service.go、common_actions.js、core_systems.js、profession_actions.js |
| C9 | feat(factions) | 领袖↔代表关系子矩阵 + JOIN_FACTION + NPC 月度加权加入/退出 | faction_join.go(新)、models.go、common_actions.js、npc.go、residents.go |
| C10 | feat(factions) | MERGE_FACTION | faction_merge.go(新)、common_actions.js、npc.go |
| C11 | docs/test | balance.md、knowledge 主题文档、14 篇取代标注、全量绿 | references/、knowledge/ |

## 6. 确定性契约

- 单 RNG 流（rng.go）不变；新增掷骰点固定顺序：发起扫描按 Actor 稳定序（居民 ID 升序、玩家最后），
  加入掷骰按 cohort ID 升序，口碑传播按 (源 ID, 目标 ID) 升序。
- 玩家动作不消耗 RNG（ADR-0018）；NPC 发起/加入掷骰即使被 Force 覆盖也消耗（沿用 npc.go 契约）。
- 浮点求和固定排序（沿用 factionCohesion 的 ID 排序纪律）。
- 同种子双跑逐字节一致；审计测试（6 种子 × 120 月）在 C7 重建基线，
  断言改为：零发起则零派系、筹备期解散、无单月归属往返、接任确定性。

## 7. 测试计划

- C3：ambition mutation 通路单测（事件改 ambition、模拟不改）。
- C4：派生威望单测（四层同构）；prestige→influence 折算单测；既有结局/门控测试重锚
  （PoliticalPrestige=100 → Ambition=1/Influence=1）。存档升级链测试与 4 个种子调优
  可玩性测试的预期内失败留档 [25-v16-test-debt.md](25-v16-test-debt.md)，
  种子重调建议等 C7/C9 落地后一次完成。
- C5：玩家任代表/创始人的序列化与快照单测。
- C6：发起资格门单测（威望/强度/AP 三闸、零成员派系合法）✓；端到端执行单测
  （门控 → JS 预检 → faction_found_actor → 建派系/扣 AP）✓；NPC 同构与
  rebuild 保留单测 ✓。NPC 发起候选平移加权随机落点，新增 3 个种子调优
  可玩性测试失败，留档 [25-v16-test-debt.md](25-v16-test-debt.md)，随 C7/C9 一并重调。
- C7：零发起零派系；筹备期解散；死亡接任（含平局）；亲缘场不产生派系；双跑确定性 ✓
  （faction_test.go / faction_lifecycle_test.go / faction_actor_test.go /
  faction_found_test.go / faction_explain_test.go 重建；引擎全绿。
  存档 v17；两处真实回归随修复闭合、5 个可玩性漂移留档
  [25-v16-test-debt.md](25-v16-test-debt.md) §五，随 C9 重调）。
- C8：三通道感知度单调性（关系/思潮/层级越近感知越高）；faction: 事实不成熟、不计共证门；
  搭载传播可让远端观测者逐步获知 ✓（faction_awareness.go + engine/service 两侧测试；
  发起播种零 RNG、口碑按 (派系, 源, 目标) 固定序、玩家搭载不掷骰、无派系零消耗；
  引擎全绿，service 全量失败清单与 §五相同，漂移留档
  [25-v16-test-debt.md](25-v16-test-debt.md) §六）。
  C9 合并后接线已闭合：居民观察者的 LeaderRelations 个人关系并入关系通道
  （取部门人脉与个人关系的高者，§1.2），事实键统一走 FactionFoundedFactID。
- C9：加入概率意识形态主导；三档关系门槛与跨层级/跨阶级闸；玩家直选不掷骰；退出惯性 ✓
  （faction_join_test.go 9 项；引擎全绿、前端 tsc 通过。感知读取走引擎 BeliefIn 直读
  model 信念层 `faction:<id>:founded`，无 service 回调；C8 未落地时该事实缺省 B=0，
  NPC 加入不触发，单测直接播种信念事实。玩家成员资格新增 Agent.FactionID（零值安全，
  存档仍 v17）；月度加入/退出挂 RunNpcTurn 的 ApplyFactionLoyaltyDynamics 之后，
  退出相先于加入相、退出 cohort 当月不再加入，杜绝单月往返）。
- C10：合并资格与成员转移、ID 存续 ✓（faction_merge_test.go 4 项：三闸 + 领袖身份/
  自身排除、成员转移 + 存续 ID 不变 + 被并方除名 + Memory 追加 + 领袖降级、玩家领袖
  读 agent:<id> 键且不掷骰、零成员派系可合并。合并关系闸对玩家同样生效——合意需双方
  领袖达标，ADR-0010 直选豁免只适用加入。引擎全绿、前端 tsc 通过；service 全量复测
  仍为 13 个已知失败，无新增漂移。开局门控回归随 political_faction_count 指标修复，
  留档 [25-v16-test-debt.md](25-v16-test-debt.md) §七）。

## 8. 下游影响清单

- 解释接口 ExplainCohortAffiliation/ExplainResidentAffiliation：改"感知+加入"叙事（C7）。
- 结局加权 weighted_endings.go：prestige 改派生读法；派系贡献结构不变。
- 动作门控 action_gates.json：political_prestige 度量改派生值（JSON 不动，求值侧换）。
- LLM advisor / 快照 / 前端 AgentStats：prestige 展示改派生分解（Ambition/Influence 两因子）。
- JS agent_update / agent_stats：删除 connections→prestige 公式，改读派生值。
- relics.json political_prestige 效果：折算 influence（C4）。
- balance.md / knowledge 03/07：重锚（C11）。
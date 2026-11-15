# Silo40「世界的反制」设计：权力方对情报流通与逃跑组织的反应

> 日期：2026-11-15
> 性质：机制设计与规格，不包含代码改动。
> 范围：玩家主动推动情报流通（《共证》线）或组织逃跑（《越岭》线）时，地堡里的权力方怎么反应、怎么发现、反制到什么程度，以及这些反制如何藏在日常小动作里而不是弹窗告知。
> 本轮边界：确定反制的主体谱系与动机差异、发现口径对现有检测机制的映射、三档反制的效果形态、呈现通道纪律、数值锚点推导与受影响测试账。凡需 sim 校准的数值一律标 `TODO(balance)`，不给平衡数值。
> 上游约束：[ADR-0009 情报会淡忘](../../knowledge/adr/0009-intel-fading.md)、[ADR-0010 玩家是干涉者不是指挥者](../../knowledge/adr/0010-player-as-intervener.md)、[ADR-0011 开局偏向必须与可玩路线测试重调同时落地](../../knowledge/adr/0011-opening-bias-needs-test-retuning.md)、[16 准备度→成果换算](./16-preparation-outcome-design.md)、[17 《共证》线经济设计](./17-information-economy-design.md)、[18 时间渗透设计](./18-time-permeation-design.md)、[10 IT 控制可信度](./10-it-control-credibility.md) §3/§5/§8、[20 《除名》事件链](../story/endings/20-chain-agent-compromised.md)、[29 五个失败结局](../story/endings/29-failure-endings.md)、[37 《越岭》事件规格](../story/endings/37-rebellion-event-spec.md)。
> 本册是 [roadmap](../roadmap.md)「逃出群体与《越岭》逃离模型」剩余项"追踪的反制入口（'反制藏进日常小动作'批次）"与 [29 §5](../story/endings/29-failure-endings.md) `silo1_purge_exposed` 反制绑定 TODO 的设计交付。

## 0. 标注口径

- **【现状】**：当前仓库已经存在的机制、状态与数值（基线 `a89a084`）。
- **【推导】**：由既有数值或实测锚点推出的设计数值与区间，全部标 `TODO(balance)`，等待 sim 校准覆盖。
- **【项目外推】**：为本设计新增的规则，不声称已经实现。
- **【裁决】**：用户既定口径，必须遵守。

## 1. 命题与总判断

**命题**：玩家推动情报流通、组织逃跑时，地堡的权力方不是被动进度条——它们会反应、会追查、会反制。但反制必须满足三条既定纪律：不弹窗告知（开局与界面设计禁止廉价引导，失败线提示只报告已发生事实的口径同源）、不直接判负（资格先于加权，玩家是干涉者不是指挥者【裁决，ADR-0010】）、不读玩家私有状态（10 §8.4：玩家 `B` 只决定行动与输出证据强度，反向不成立）。

**总判断（推荐方案）**：世界的反制是**对公开事实的确定性状态反应集合**——每个权力方读自己能观察到的公开事实（曝光 `E`、公开证据 `Q`、痕迹完成键、筹备账本、立场、部门戒备），经既有月度结算产生**成本型后果**（AP 变贵、月份变长、信道变窄、关系变冷、怀疑度增量），后果全部经既有反馈通道（月度故事、行动回执、门控灰置、账本数字）呈现，让玩家从日常里"读出来"而不是被"告知"。

三条理由：

1. **反制的输入必须是权力方真的能知道的东西**。IT 主任看不到玩家心里的确信（`B`），它看得到的是访问日志、流量形状、部门戒备与"有人在传某件事"的风声（部门信念层对泄漏事实的 `B`）。用公开事实做输入，反制才既有世界观合法性（10 §3.3：外界通常不知道 IT，IT 的归因也要经证据链），又有玩家可反推的因果——能被反推的反制才是策略要素，看不见的惩罚只是噪音。
2. **反制的输出必须是成本而不是判决**。结局资格全部由事实经仲裁判定（12 §2.2）；反制若直接写资格或直接判负，等于在仲裁之外新增一个投票主体，与"玩家是干涉者不是指挥者"直接冲突。正确形态是：反制让后面的路**变贵**（AP/月份）、**变窄**（信道/目标/窗口），走不走、怎么绕仍是玩家的决策。唯一通向失败结局的路径是既有的怀疑度封顶硬护栏（`victory_check`，【现状】），反制只能按既有公式贡献增量，不能设置封顶。
3. **反制的呈现必须藏在日常小动作里**。已裁决的反制形态是"有对象、地点、时机、材料与痕迹的情境化动作"（[silo1_purge_exposed.js](../../events/special/silo1_purge_exposed.js) 的 TODO(Countermeasure binding) 注释）；呈现走月度故事、行动回执与门控灰置这些既有通道，玩家看到的是"住处被例行检查""熟悉的联系人调去了夜班""该部门近期严查内外往来"，而不是"你被发现了"。

**明确不做**：不新造"隐蔽值"这类单一不可见总计量；不新增结局类型；不新增随机源（反制判定全部确定性，立场重估复用既有单一 RNG 流口径，[top_stance.go](../../internal/engine/top_stance.go)）；不改动任何初始思潮值（门控耦合旋钮，[ADR-0011](../../knowledge/adr/0011-opening-bias-needs-test-retuning.md)）；不改动既有资格口径与仲裁逻辑；宣传不是反制、不得充当反制（[silo1_purge_exposed.js](../../events/special/silo1_purge_exposed.js) 既有裁决）。

## 2. 谁在反制：五个主体、五种动机

反制不等于"发现玩家"。五个主体里只有两个（本地 IT、本地执法）有可能把反制指向玩家本人，且指向的前提是玩家自己的公开痕迹越线；其余反制全部指向**群体、信道与部门**——权力方镇压的是"有人在传东西、有人想走"这件事本身，不是某个名字。

| 主体 | 怕什么（动机） | 读什么（可观察输入）【现状】 | 能做什么（反制手段） | 指向 |
| --- | --- | --- | --- | --- |
| **本地 IT** | 怕自己的操控被证实（`E`/`Q` 上升，法警调度证据链成形，10 §3.3/§3.7）；怕失控招来 1 号问责（`R_s`）；怕 Protocol 偏离 | `playerThreatVisible`：公开证据 >0 或核验证据 >0 或 `E ≥ 0.2`（[it_authority.go](../../internal/engine/it_authority.go)）；部门戒备；泄漏事实的部门信念层（§3.2） | 月度权威回合的 17 个候选动作（维护 8＋隐蔽 7＋反击 2），`R_op` 预算约束；立场阻挠（[top_stance.go](../../internal/engine/top_stance.go)）；审计没收（[escape_prep_audit_seizure.js](../../events/crisis/escape_prep_audit_seizure.js)） | 群体/信道为主；玩家本人仅在威胁可见时（`counter_investigate_player`） |
| **Judicial（司法部）** | 怕程序不自洽、怕被某一方工具化——它的权力就是程序本身；归属在 IT/SWING/PACT 三态间迁移，最终归属只有 IT 或约章，SWING 是争夺态，市长仅有影响力修正、永不构成归属（[authority.go](../../internal/engine/authority.go)，[ADR-0005](../../knowledge/adr/0005-judicial-not-mayors.md)） | 纸面痕迹（身份链完成键计数，[20 分册](../story/endings/20-chain-agent-compromised.md)）；搜查中见到的不一致 | 复核来历（`identity_query_local`）、档案复查（`archive_review_opened`）、搜查与逮捕（职业动作面）、封顶时签发逮捕令（`victory_check` 演出） | 痕迹指向谁就指向谁；多数时候指向"对不上的纸面"，不指向人 |
| **市长** | 怕正统性流失（`MayorLegitimacy`）与叛乱失控——镇压可行性随叛乱下降（[top_stance.go](../../internal/engine/top_stance.go) `mayorStanceWeights`） | 叛乱值；正统性；公开秩序事件 | 直接命令、警察处置（`PoliceObedience` 是执行前提）、公开叙事；立场三档（阻挠/不阻挠/配合）塑形逃跑人数（裁决 F，16 §3.2） | 群体；市长反制的是"秩序松动"，不是某个人 |
| **1 号（上层控制方）** | 怕失序——它读的是复合压力，不分辨个人：`order_pressure = 0.6·叛乱 + 0.4·平均恐慌 ≥ 0.45` 且宣传等级 `≥ 2`（有组织煽动门，[content_events.go](../../internal/engine/content_events.go)） | 各堡叛乱值（注意力 = 叛乱，[world.go](../../internal/service/world.go)）；通牒后压力是否回落 | 通牒 → 三条反应链：本地整肃（前兆 U+4/执行 U+7）、终止投放（自检 U+10/投放 U+14）、投送打击（侦察 U+16/打击 U+19）（[silo1_purge_omen.js](../../events/special/silo1_purge_omen.js) 时点注释） | 整个堡；玩家只是被波及 |
| **部门内部** | 怕被牵连——恐慌与戒备是部门自保的两个读数 | 部门信念层对泄漏事实的 `B`（"有人在我们中间传东西"成为部门知识）；恐慌值 | 戒备抬升（`Alertness`，压低探查成功率 `1−0.60·alert_d`，[information.go](../../internal/engine/information.go)）；关系收紧（接触变谨慎）；恐慌漂向忠诚（既有漂移，17 §3 修复 D） | 群体；部门反制没有主体意识，是"组织收缩"的投影 |

两点谱系原则（【项目外推】，本册核心纪律）：

1. **档位越高，越要求归属到玩家个人行为；档位越低，越指向群体。** 第一档反制只读群体级事实（戒备、恐慌、泄漏事实 `B`），完全不归属玩家；第二档读信道与账本（仍不归属人）；第三档读痕迹计数与公开事实（身份链完成键、怀疑度），仍永不读取玩家私有 `B`。
2. **动机差异决定触发输入，不决定强度。** 同一玩家行为（例如一次公开分享）会同时喂多个主体的输入（IT 的 `E`、部门的泄漏 `B`、市长的恐慌），但每个主体按自己的门槛与预算独立反应——不存在一个统一的"世界反制条"被同时填满。这保证反制节奏由各主体的既有资源曲线（`R_op` 收入、立场重估时点、反应链时点阶梯）自然错开，不新造协调器。

## 3. 怎么发现：复用现有检测口径，不新造隐蔽值

玩家的每个推进行为都在喂既有的检测量。本节把"什么行为提高被注意的概率"写成**行为 → 检测量 → 读取方**的映射表，全部落在既有公式与既有状态上；新增的只是把流通类行为接进既有因子表（【项目外推】），公式本身不改。

### 3.1 三个检测量及其生产者

| 检测量 | 语义 | 生产者【现状】 | 读取方【现状】 |
| --- | --- | --- | --- |
| **痕迹**（`suspicion_level`，actor 层） | 玩家个人纸面接缝的累积；只升是设计口径（20 分册 §3），年衰减 0.05（[core_systems.js](../../events/mechanics/core_systems.js)） | `sampledSuspicion = base·V_identity·F_fit·W_witness·R_record·(1+mismatch)`（[mechanics.go](../../internal/engine/mechanics.go) 注入 prelude，10 §8.3）；固定增量 `suspicionDelta`（Build 0.01、Incite 0.05、职务动作 0.01–0.03）；秘密行动开场 0.002–0.008＋结算 0.01–0.05（[station_operations.go](../../internal/service/station_operations.go)） | 《除名》链三级节点（完成键计数代理）；封顶 `≥1.0` 硬护栏；`agent_erasure_staged` 读 `≥0.7` |
| **方向**（反制方对"泄漏事实"的 `B/π`） | 反制方知道"有人在传某件事"，不知道是谁——确信随流通次数上升，归因由版本分布 `π` 表达 | 玩家流通行为生成观察，经既有吸收公式 `s_i = rel·(0.85+0.15·related_f)·independence`、`B_new = 1−(1−B_old)·(1−s_i/(1+0.25·n))`（10 §3.5，[information.go](../../internal/engine/information.go)）进入 IT/Judicial 层 | §4 各档反制的触发输入；IT 的 `π` 归因（外来渗透者 vs 内部不满，同构 `it_directed/mundane` 版本对）决定它选哪类反制动作 |
| **账本**（可见筹备与流量） | 看得见的准备与看得见的流量：修了多少服、划拨了多少物资、信道用了多少次 | `prep_exposure`：修服 +0.08、演练 +0.06、预案 +0.02、划拨 +0.04（市长非配合时）、照护/生产 +0.03（[37 分册](../story/endings/37-rebellion-event-spec.md) 动作表）；`StationOps.Exposure`/`Silo1.Exposure`：跨堡行动反向追查 0.04–0.14（[station_operations.go](../../internal/service/station_operations.go)）；信道暴露成本（[16 §2.3](./16-preparation-outcome-design.md)） | 审计没收（`prep_exposure ≥ 0.5`，[escape_prep_audit_seizure.js](../../events/crisis/escape_prep_audit_seizure.js)）；出发追踪（`≥0.6` 折减 30%，[escape_settlement.go](../../internal/service/escape_settlement.go)）；堡外追踪热度（筹备暴露权重 0.8，[escape_outside.go](../../internal/service/escape_outside.go)） |

### 3.2 方向量怎么累积：泄漏事实的 B/π 复用（本册新增映射）

【项目外推】每次玩家完成一次流通或公开动作（Share／verify／publish／宣传性动作），为驻地反制方（IT 层与 Judicial 层）生成一条针对**泄漏事实** `leak:<fact_id>` 的观察：

- **吸收走既有公式**：首次获知该事实被传为独立来源（`independence = 1.0`），此后同一事实的重复流通按同源转述（`independence = 0.25`，既有 `IndependenceRe`）——反制方对"有人在传"的确信随流通次数上升但边际递减，与部门信念层的淡忘（0.01/月，17 §3 修复 A）同口径：**流通停了，风声也会淡**。
- **观察去重走既有口径**：去重键 `leak:<fact_id>:<actor>:<年-月>`，同一人同一事实同月只记一次观察（同构 `gather:<fragment>` 去重与 `observation_id` 去重纪律，[game_service.go](../../internal/service/game_service.go) `applyBeliefObservations`）；重复流通不刷反制确信。
- **目击档映射既有因子表**：对单部门分享 = 小圈子（`W_witness = 1.0`）；跨部门串联 = 部门（1.6）；发布公开证据 = 公开（2.2）＋监控记录档（`R_record = 1.6`）——公开动作"被注意"的概率天然高于私下流通，因子全部取自 [information.go](../../internal/engine/information.go) 既有档位，不新增系数。
- **π 归因**：泄漏事实带两个解释版本——"外部渗透者在传"与"内部不满在传"，先验各 0.5（同构碎片目录 `it_directed/mundane` 版本对，[information.go](../../internal/engine/information.go) `buildFragmentCatalog`）。玩家的秘密行动痕迹（身份链计数）抬高"外部渗透者"似然，公开的社会不满（叛乱、恐慌）抬高"内部不满"似然——**归因决定反制指向**：信"外部渗透者"时 IT 倾向查人（第三档），信"内部不满"时倾向控面（第一、二档）。

### 3.3 行为—注意映射总表

| 玩家行为 | 喂哪个检测量 | 既有数值/因子 | 谁会因此动 |
| --- | --- | --- | --- |
| Gather（探查） | 痕迹（`SAMPLED` 时）＋目标部门戒备（失败 +0.10，复核减半） | `ProbeBaseSuspicion 0.01`、`F_fit` 跨 1.2/本 0.6、戒备惩罚 `1−0.60·alert_d` | 部门（戒备）；IT（戒备是其月度回合的输入之一） |
| Build Network | 痕迹 +0.01 | `suspicionDelta(ctx, 0.01)` | 部门（关系面）；痕迹累计进身份链 |
| Share Info | 方向（§3.2 泄漏观察）＋痕迹（小圈子档，【项目外推】接入既有因子表） | 吸收 `B_player·transmit·receptivity`；`W_witness 1.0` | 部门（泄漏 `B`）；IT/Judicial（方向） |
| Incite Rebellion | 痕迹 +0.05＋叛乱＋恐慌 | 既有固定增量 | 市长（叛乱）；1 号（`order_pressure` 复合）；部门（恐慌） |
| verify/publish（公开证据） | `E`/`Q` 聚合（10 §3.3/§3.7）＋方向（公开档 2.2） | `playerThreatVisible` 门 | IT（反击候选集解锁，既有）；Judicial（`Q` 是约章门输入） |
| 秘密行动（STATION_*/SILO1_DECEIVE） | 痕迹（开场+结算）＋身份链完成键计数＋`Exposure` | 0.002–0.008 开场、0.01–0.05 结算；`identity_trace_accumulates` cd2 | Judicial（纸面接缝）；1 号（`Exposure` 反向追查） |
| 逃跑筹备各动作 | `prep_exposure` 账本 | 0.02–0.08（37 分册表） | IT（审计没收门 0.5）；出发追踪（0.6） |
| 外溢信道建立/使用 | 信道暴露成本 | 可靠度 0.75/0.55/0.35 三档（[escape_spillover_*.js](../../events/player_actions/)） | 1 号（异常流量观察，Layer 3，16 §2.3 预留） |

**矿工结构性豁免不变**：`UNSAMPLED` 类（矿工普通行为）不进入检测流程、无怀疑增量（10 §8.2，[information.go](../../internal/engine/information.go)）；本册不改动检测分类。

## 4. 反制分级与后果：三档，全部是成本

三档反制对应三种目标粒度：**指标悄悄恶化（群体）→ 链路被掐断（信道）→ 具体的人被盯上（个体）**。每档给出触发输入、执行主体、结算效果、玩家可读的因果链。所有效果都是成本型：让路变贵、变窄，不直接判负（【裁决】）。

### 4.1 第一档：指标悄悄恶化（目标 = 部门/群体）

| 编号 | 反制 | 触发输入 | 效果形态 | 现状 |
| --- | --- | --- | --- | --- |
| T1-1 | 戒备收紧 | IT 月度回合选中 `counter_seize_materials`（威胁可见时）；玩家失败探查自喂 | 目标部门 `Alertness +0.15`（反击）/+0.10（失败探查）→ 探查成功率 `×(1−0.60·alert_d)` | **既有**（[it_authority.go](../../internal/engine/it_authority.go)、[common_actions.js](../../events/mechanics/common_actions.js)） |
| T1-2 | 流通摩擦 | 目标部门 `Alertness ≥` 门槛（起点 0.30，【推导，TODO(balance)】） | 对该部门的 Share/Build AP 成本按既有 mismatch 公式上浮：`ceil(base·(1+0.5·mismatch_alert))`，`mismatch_alert` 起点 0.5（即 Share 3→4 AP）；**吸收效果不变，只贵不堵** | 【项目外推】公式既有（10 §8.1），新增的是把戒备接成 mismatch 项 |
| T1-3 | 关系冷却 | 部门对泄漏事实 `B ≥` 门槛（起点 0.45，【推导，TODO(balance)】） | 该部门关系增长减半、持续 N 月（起点 12 月，§6.3 推导）；Build Network 回执措辞变化 | 【项目外推】 |
| T1-4 | 恐慌收拢 | 反制事件附带小量恐慌增量（单次 0.02–0.04，同构既有征兆事件量级） | 恐慌经**既有漂移**漂向忠诚（17 §3 修复 D）→ 全堡接收性自然变差 | 漂移通道**既有**，增量走既有月度通道 |

**玩家可读的因果链**：探查失败回执已写明"该部门戒备上升"（既有文案）；T1-2 的成本上浮直接显示在行动行 AP 数字上（既有 UI）；T1-3 读关系数字与回执措辞；T1-4 读部门恐慌/思潮面板（既有）。**不需要任何新提示**——这一档的反制就是"数字变难看了，而且能看出为什么"。

### 4.2 第二档：链路被掐断（目标 = 信道/链条）

| 编号 | 反制 | 触发输入 | 效果形态 | 现状 |
| --- | --- | --- | --- | --- |
| T2-1 | 筹备审计没收 | `prep_exposure ≥ 0.5` 且 IT 立场阻挠（`canTrigger` 读 `top_stance.it === "block"`）；REPEATABLE cd6 | 已修服 −2、`prep_exposure −0.3`、恐慌 +0.02 | **既有**（[escape_prep_audit_seizure.js](../../events/crisis/escape_prep_audit_seizure.js)） |
| T2-1b | 囤积审计没收（同构扩展） | 同上＋`supplies_packed/materials_packed ≥` 门槛（起点 150/80，【推导，TODO(balance)】） | 封存补给/材料按比例没收（起点 20%，【推导】），`prep_exposure −0.3` | 【项目外推】效果接口既有（`preparationDelta`） |
| T2-2 | 信道审计 | 信道使用次数 × 信道暴露成本 ≥ 门槛（起点：IT 专线类 3 次、信号类 5 次、实体类 8 次，【推导，TODO(balance)】，与三档可靠度 0.75/0.55/0.35 反相关） | 该信道可靠度降一档或中断；信道卡上的数字变化玩家可见（既有账本字段） | 【项目外推】16 §2.3 预留的"外溢暴露与宿主风险结算"的首个落点 |
| T2-3 | 分享目标收紧 | 目标部门 `Alertness ≥` 高门槛（起点 0.50，【推导】） | 针对该部门的流通类动作进入灰置：`available` 增加戒备上限条件，`DisabledReason` 自然可读（"该部门近期严查内外往来"）；戒备回落后自动恢复——**灰置不是锁定**，动作不重新隐藏（15 分册 §3.3 三态纪律） | 【项目外推】机制既有（[action_gates.go](../../internal/service/action_gates.go) `available` 每次快照重算） |
| T2-4 | 谈判可信度侵蚀 | `prep_exposure ≥` 门槛（起点 0.4，【推导】） | 保障可信度 `Guarantee −δ`（起点 0.15，【推导】）经既有 `addTopStanceGuarantee` 接口 → 顶层立场权重自然偏向阻挠（既有 `mayorStanceWeights/itStanceWeights` 读 `Guarantee`）→ 出发人数自然变少（裁决 F：立场只塑形人数，不做判定闸） | 【项目外推】接口与权重既有（[top_stance.go](../../internal/engine/top_stance.go)） |

**玩家可读的因果链**：T2-1/1b 是账本数字直接减少＋月度故事"审计"条目；T2-2 是信道卡可靠度数字变化＋"例行流量审计"故事；T2-3 是动作灰置原因文本；T2-4 是谈判接触卡措辞变模糊（内容层），没有任何数值弹窗。

### 4.3 第三档：具体的人被盯上（目标 = 个体）

| 编号 | 反制 | 触发输入 | 效果形态 | 现状 |
| --- | --- | --- | --- | --- |
| T3-1 | 玩家身份线（四级递进） | 秘密行动完成键计数（`identity_trace_accumulates` cd2）与怀疑度 | 第 3 次 = 本地执法复核来历；第 5 次 = IT 档案复查；怀疑度 ≥0.7 = 抹除演出；≥1.0 = 封顶硬护栏 | **既有**（[20 分册](../story/endings/20-chain-agent-compromised.md)四节点全部落地） |
| T3-1b | 身份线的"日常小动作"绑定 | 同上（各级节点的既有触发不变） | 每级节点附带一个情境化小动作后果：复核期 = 一处地点类动作次月灰置（"例行核查期间不便前往"）；复查期 = 一项职务动作当月灰置（"职务暂时交接"）；抹除准备期 = 秘密行动类动作灰置（`available: suspicion < 0.7`），逼玩家选择收手或最后一搏 | 【项目外推】——这正是 [silo1_purge_exposed.js](../../events/special/silo1_purge_exposed.js) TODO 等待的绑定；灰置机制既有 |
| T3-2 | 联系人被盯上 | 某部门对泄漏事实 `B ≥` 高门槛（起点 0.60，【推导】，同构 `QualInvestigate` 口径）——部门知道了"有人在传"，IT 就有了排查方向 | 该部门联系人被调离/问话：关系 −δ（起点 0.10，【推导】）、接触类动作 N 月不可用（起点 3 月，cd 保护）；月度故事"你熟悉的那个人调去了夜班" | 【项目外推】 |
| T3-3 | 1 号反应链（谱系极端） | `order_pressure ≥ 0.45` 且宣传 `≥ 2`（通牒门）；通牒后压力不回落 | 整肃/终止/投送三链，时点阶梯 U+4/+10/+16 | **既有**；本册只定位，不新增 |

**硬约束**：第三档只能动**匿名联系人**。正典具名人物的生死唯一来源是事件链节点，不得由反制逻辑触碰（项目硬约束）；反制也不得改写具名人物的信念与状态。

**玩家可读的因果链**：身份线四节点的叙事已经是"日常小动作"范本（"你的住处被例行检查，你的职务被建议'暂时交接'"）；T3-2 读月度故事与关系数字。第三档是唯一可能通向失败结局的档——但通路是既有的怀疑度封顶，反制贡献的增量受 §6.2 预算约束，保证《除名》仍是"玩家一直赌下一次行动是最后一次"的选择后果（29 §3），不是世界自动收敛。

### 4.4 档间关系

- **不跳档**：第三档的触发必须经过第一、二档同款公开事实（痕迹计数、泄漏 `B`），不存在"跳过恶化直接抓人"的路径。
- **可反制**：每档都有玩家侧的应对——第一档靠等（戒备衰减）与换目标；第二档靠降暴露（低暴露动作、IT 立场转向后风险消失，既有 `canTrigger` 口径）；第三档靠收手（痕迹只升不降，但停止秘密行动即停止计数，既有 cd2 口径）。应对全部是既有玩法，不新增反反制按钮。
- **不同主体在各档的分布**：第一档以 IT 与部门为主，第二档以 IT 为主（信道审计涉及 1 号），第三档以 Judicial/IT 为主（身份线）与 IT 为主（联系人）；市长与 1 号的反制始终是群体级（市长在第二档只经立场权重间接出现）。

## 5. 藏在日常小动作里：呈现通道纪律

【裁决】反制的呈现必须是玩家从已有叙事与日常反馈里**读出来**的，不能靠一条"你被发现了"的提示。反制事件就是"一些不属于任何流程的小事同时发生了"（[agent_erasure_staged.js](../../events/special/agent_erasure_staged.js) 既有文案范式）。

### 5.1 可承载信号的既有通道（盘点）

| 通道 | 机制 | 承载哪类反制信号 |
| --- | --- | --- |
| 月度故事（`StoryEvent`，月度反馈必达，[world.go](../../internal/service/world.go) 延迟事件固定排空） | 内容事件点火 → `recordStories` → 当月反馈 | 反制事件的叙事本体："例行检查""流量审计""联系人调岗"；走 `category=crisis/special`，多堡场景经谣言过滤与同月合并（[perception.go](../../internal/service/perception.go)） |
| 行动回执（`action_result.message`） | 动作结算同步返回 | 摩擦与冷却的即时反馈："对方比上次谨慎""该部门没有回应你的接触" |
| 行动门控（`available` 灰置＋`DisabledReason`） | 每次快照重算，执行前再校验（[action_gates.go](../../internal/service/action_gates.go)） | T2-3/T3-1b/T3-2 的"动作暂时做不了"，原因文本就是叙事 |
| 数字面板（AP 成本、关系、戒备经回执、账本、信道可靠度） | 既有 UI | 第一、二档的量化面——数字变化本身就是信号，不加标注 |
| 线面板征兆聚合（`failureOmens`，[lines_progress.go](../../internal/service/lines_progress.go)） | 只聚合 12 个既有征兆键的点亮状态 | 身份线三节点已在列（`identity_trace_accumulates`/`archive_review_opened`/`agent_erasure_staged`）；新增反制事件**不新造预警机制**，够格的按 29 §2 口径增列征兆键 |

### 5.2 呈现禁令（逐条）

1. **不出现"你被发现了"类告知**：任何反制事件文案不得指认玩家为对象，只描述发生的事（既有范本：`identity_query_local`"他们没有提到另一个堡，只把几次对不上的记录放在一起"）。
2. **不显示概率与数值预告**：行动列表不显示"反制概率"，灰置只写当前缺项（15 分册 §6.2 同口径）。
3. **不点名结局、不预测**：反制提示仅作为已发生事实的汇总，出路仍由玩家行动决定（失败线界面提示约束，项目既有口径）。
4. **不弹窗、不强制引导**：反制信号全部走上述被动通道；玩家没读出来是设计允许的——读得出来是高手，读不出来是代价。
5. **不造新通道**：月度故事、回执、灰置、数字、征兆聚合之外不新增呈现面；LLM 文本变体失败不得影响反制结算（15 分册 §4.6 同口径）。

### 5.3 文案示范（每档一例，供内容批次使用）

- 第一档（T1-2 生效）：回执"Supply 的人接过材料时多问了一句'谁让你来的'。这次沟通花的时间比往常长。"（AP +1 已在行动行可见）
- 第二档（T2-1b 生效）：月度故事"后勤进行了一次库存例行盘点。封存区的一部分补给以'账实不符'的名义被重新登记进了公共配额。"
- 第三档（T3-2 生效）：月度故事"Medical 那边你常找的那个人调去了夜班。接替她的人礼貌、客气，问你有没有登记过的预约。"

## 6. 数值推导：概率、强度、冷却的锚点

所有数值锚定三类既有依据：17 号 AP 经济、18 号渗透月数、既有代码数值。未定值一律 `TODO(balance)`。

### 6.1 AP 摩擦预算（锚：17 §3/§5）

- 生涯 AP ≈ 380–410（开局 50＋36/年×8 年＋威望）；门槛需求 ≈162（基础）＋81（渗透期补流通）= 243（17 §5、18 §5）；**富余 ≈ 137–167 AP**。
- 反制摩擦总额 ≤ 富余的一半 ≈ **70–80 AP**（留出策略失误与《越岭》线的 AP 需求）；再留一半安全边际后，本册取**全程摩擦总预算 ≤ 36 AP**【推导，TODO(balance)】。
- 推导链：摩擦窗口单次时长 ≤ 12 月（§6.3），窗口内流通频率 ~1 次/月（17 §3 修复 C "成本≈月收入"节奏），单次成本上浮 +1 AP（Share 3→4）→ 单窗口 ≈ 12 AP → **全程摩擦窗口 ≤ 3 次，窗口间冷却 ≥ 12 月**【推导，TODO(balance)】。
- 显式失败纪律：若重调后保守策略结算月被摩擦推出地平线（>94），回本节降窗口数/时长，**不放宽测试**（18 §7.2 判据 4 同构）。

### 6.2 怀疑度预算（锚：既有增量表与封顶 1.0）

- 封顶 1.0；年衰减 0.05（≈0.0042/月）；既有增量：Build 0.01、Incite 0.05、探查 ~0.01×因子、秘密行动 0.002–0.008 开场＋0.01–0.05 结算、IT 反击 `counter_investigate_player` 0.08（既有）。
- **反制系统的全程净贡献 ≤ 0.25（封顶值的四分之一）**【推导，TODO(balance)】：T3-2 单次 +0.03（起点，同构身份链单次痕迹量级）；第一、二档反制**不贡献怀疑度**（指向群体，不归属玩家——档位原则 §2）；T3-1 沿用既有链，不新增增量。
- 理由：《除名》的叙事是"一直赌下一次行动是最后一次"（29 §3）——失败必须来自玩家的贪，不能来自世界的自动累积；0.25 的预算保证反制把玩家**推向**悬崖，但跳下去的那一步永远是玩家自己迈的。

### 6.3 时长与衰减（锚：18 §3.3 单次分享自然寿命 25–30 月、渗透区间 [24, 36]）

- **戒备衰减**起点 0.015/月【推导，TODO(balance)】：与淡忘系数 0.01 同构（淡忘是信念的天敌，衰减是戒备的天敌）；0.30 的戒备 ~20 月自然消散——≤ 单次分享自然寿命（25–30 月）且 ≤ `PermeationMonths` 下限（24），保证**单个恶化窗口不会打断渗透**：玩家的应对（再流通）始终在既有玩法内可行。
- **关系冷却时长**起点 12 月 ≤ 24（同上界推导）；**联系人不可用**起点 3 月（同构演练 cd3 与实体信道冷却量级）。
- 上界纪律：任何第一档恶化的持续时间超过 24 月即判过界——那意味着渗透槽位必然清零、可玩性保证受威胁（18 §3.4 粘性成熟保住的下限），必须回本节调参。

### 6.4 冷却与事件密度（锚：既有 cd 序列与文本墙治理）

- 既有冷却序列：身份痕迹 cd2、演练 cd3、外溢信道 cd4、审计没收 cd6、常态硬化 cd12。本册新增事件冷却全部落在 **[4, 12] 月**区间【推导，TODO(balance)】：T1 事件 cd ≥ 6、T2-1b cd6（与 T2-1 共用审计名义，同月不并发）、T2-2 cd ≥ 6、T3-2 cd4。
- **同源去重**：同一触发源（同一部门戒备越线、同一信道审计）每冷却周期只生效一次，不叠加（同构决定性冲击同源去重裁决）；审计记录永久保留（供线面板与结算行引用）。
- **新增事件条目预算 ≤ 8 个**：多堡事件重构已确立"同月同类重复事件必须合并、避免文本墙"的治理口径（[perception.go](../../internal/service/perception.go) 合并逻辑）；反制事件全部走该通道，条目数受同一密度约束。

### 6.5 IT 资源一致性（锚：it_authority.go 既有预算）

- 凡 IT 执行的反制（T1-1/2/3、T2-1/1b/2、T3-2）必须走既有 `R_op` 预算（初始 60、收入 8/月、上限 100）与弱势成本曲线 `ceil(base·(1+3·(1−W/100)²))`——**没有免费反制**。
- 新增候选动作成本取既有目录区间 [5, 12] 内的值【推导，TODO(balance)】；反击加成、双动机 utility、资源枯竭衰减全部沿用（10 §5.5："玩家须承受至少一次反击；强度随资源枯竭衰减"）。
- 叙事同构：W 越低 IT 越反制不起——弱势的权力方收手是自保，不是仁慈。

### 6.6 初始思潮与门槛耦合（锚：ADR-0011）

- 反制效果**不触碰初始思潮**；T1-4 的恐慌增量走既有月度漂移通道（恐慌→忠诚，17 §3 修复 D），单次 0.02–0.04（同构既有征兆事件），**全程恐慌净贡献 ≤ 0.1**【推导，TODO(balance)】——恐慌是门控耦合旋钮的供给输入（TIME 覆灭前夕思潮闸），增量必须进 §7 测试账核算，不得视为自由旋钮。
- 1 号通牒门槛（`order_pressure ≥ 0.45` 且宣传 ≥ 2）与三链时点阶梯**不改**；本册不新增 1 号反制。

## 7. 测试账（本册核心）

上一次开局可变性原型被回退，就是因为没有提前算测试账（[19 号分册](./19-opening-variability-design.md) §6、[ADR-0011](../../knowledge/adr/0011-opening-bias-needs-test-retuning.md)）。本册提前立账。

### 7.1 受影响的测试清单

| 测试 | 位置 | 受影响方式 |
| --- | --- | --- |
| `TestInformationEndingPlayableFromNewGame` | [info_endgame_test.go](../../internal/service/info_endgame_test.go) | **核心受影响**：摩擦窗口与联系人监视改变流通节奏；策略需预留摩擦 AP（§6.1 预算内）；断言链（内容键＋公开证据＋INFORMATION 结算）不变，结算月可能后移 |
| 三策略基线 | [info_economy_harness_test.go](../../internal/service/info_economy_harness_test.go) | **基线重测**：17 §4 对比表在反制落地后再次失效，需重建（零操作/最低维持仍应输，有节奏推进应赢） |
| `escape_chain_test.go`／`escape_outside_test.go` | [escape_chain_test.go](../../internal/service/escape_chain_test.go)、[escape_outside_test.go](../../internal/service/escape_outside_test.go) | T2-1b/T2-4 改变筹备节奏与保障可信度；断言（逃出人数、追踪档位、群体存续）不变 |
| `rule_of_law_chain_test.go`／结局验收宿主 | [rule_of_law_chain_test.go](../../internal/service/rule_of_law_chain_test.go)、[ending_acceptance_harness_test.go](../../internal/service/ending_acceptance_harness_test.go) | 间接：玩家时间分配竞争（同 18 §6 说明 1 口径）；**必须重跑**，dump 基线 JSON 重建 |
| `compromised_endgame_test.go`／`failure_endings_test.go` | [compromised_endgame_test.go](../../internal/service/compromised_endgame_test.go)、[failure_endings_test.go](../../internal/service/failure_endings_test.go) | T3-1b 绑定改变抹除准备期的动作可用性；封顶口径与断言不变 |
| 同种子 parity 双跑 | [parity_harness_test.go](../../internal/service/parity_harness_test.go) | 新增反制状态（戒备衰减计数、摩擦窗口计数、联系人不可用计数、信道审计状态）**必须进快照**；双跑逐字节一致契约不变 |
| 1 号与多堡回归 | [silo1_reaction_chain_test.go](../../internal/service/silo1_reaction_chain_test.go)、[two_silo_sim_test.go](../../internal/service/two_silo_sim_test.go) | 门槛不改，预期不变；全量回归 |
| 故事通道纪律 | [cross_story_dedup_test.go](../../internal/service/cross_story_dedup_test.go)、[story_silo_attribution_test.go](../../internal/service/story_silo_attribution_test.go) | 新反制事件走月度故事通道，必须通过归属戳记与同月去重 |
| 可玩性保证矩阵 | 职业 × 种子全覆盖（任务书口径） | 反制落地后**同口径复跑全绿**；任何组合掉出按 §7.2 判据 4 处理——修参数，不删组合 |

### 7.2 公平重调判据（全文沿用 18 §7.2，增一条）

1. **不改断言语义**：结局资格口径、内容键清单、公开证据清单不得放宽；只能改"策略何时做什么"，不能改"什么算赢"。
2. **不白盒反制状态**：禁止在测试里直接写戒备值、摩擦窗口或"关闭反制"的配置开关让测试变绿——反制必须真实发生、策略真实绕行；绕不过去说明参数过紧，回 §6 调参。
3. **策略改动限于适配反制**：允许预留摩擦 AP、错开摩擦窗口、在戒备高企期换流通目标；不允许新增游戏内不存在的动作、不允许改 AP 基础成本、不允许延长循环上限。
4. **循环上限只可逼近地平线、不可越过**：重调后结算月 > 94 判为反制参数过紧，回 §6 降窗口数/时长/增量，不放宽测试（显式失败纪律）。
5. **可玩性保证矩阵不降级**（18 §7.2 判据 5）。
6. **（新增）反制不得单独落地**：反制机制必须与本节重调**同批合并**，不允许"先落反制、测试红了再说"的中间态进 main（同 18 §8 Layer 纪律）。若 18 号渗透先行落地，两批的重调必须合并计算，不得各自松参数凑绿。

### 7.3 重调后的验收观测项

1. **反制触发分布**：每档每局的触发次数应落在冷却允许的上限之内（§6.4）；若某档全程 0 次触发，说明门槛脱离真实玩法（反制形同虚设），显式失败。
2. **结算月偏移**：信息线保守策略结算月相对无反制基线的后移 ≤ 6 月（§6.1 预算的日历投影，【推导，TODO(观测阈值)】）。
3. **失败结局触发率**：反制拉长暴露窗口后失败线触发率的上升幅度（同 18 §7.3 观测项 3）；陡增说明反制过狠，回 §6 降强度。
4. **摩擦 AP 实际占用**：harness dump 记录全程摩擦 AP 总量，≤ 36（§6.1）；超支即回 §6。

## 8. 硬约束核对

| 约束 | 本册核对 |
| --- | --- |
| 不得削弱现有可玩性保证 | §6.1 摩擦预算在 AP 富余内、§6.3 恶化窗口 ≤ 渗透下界、§7.1 矩阵同口径复跑、§7.2 判据 4/5 显式失败纪律 |
| 不得改任何初始思潮值（ADR-0011） | §6.6：反制不触初始思潮；恐慌增量走既有漂移且全程 ≤ 0.1 并进测试账；本册所有新旋钮均为自由旋钮（资源/冷却/门槛），只需 parity 回归 |
| 不新增随机源 | 反制判定全部确定性（读状态、比门槛、走既有月度 dispatch）；立场重估复用既有单一 RNG 流口径，不新增取样点 |
| 不改资格口径与仲裁 | 反制只产成本；唯一失败通路是既有怀疑度封顶，增量受 §6.2 预算 |
| 正典具名人物 | §4.3：反制只动匿名联系人；具名人物生死唯一来源仍是事件链节点 |
| 空配置保持原行为 | 反制事件未配置/门槛未越线时逐位不变（既有事件系统口径）；新账本字段零值 = 旧档语义 |

## 9. 明确不做

1. 不新造"隐蔽值"或任何单一不可见总计量；检测只有三个既有检测量（§3.1）。
2. 不让反制直接判负、直接写资格、直接锁死动作（灰置必须可恢复，锁定只来自既有三态门控的 `unlock` 语义）。
3. 不新增结局类型、不新增随机源、不改初始思潮、不改 1 号通牒门槛与反应链时点。
4. 不让宣传充当反制（既有裁决）；不新造专用反制组织（15 分册 §5.1 同口径）。
5. 不在本轮写代码、事件、配置与数值；本册只产出机制设计与规格。
6. 不给反制加"教学"：读不出信号是设计允许的代价（§5.2 禁令 4）。

## 10. 主要依据

- 检测三类与因子表、吸收公式、E/Q 聚合：[10-it-control-credibility](./10-it-control-credibility.md) §3/§8（子文件 03/13）、[information.go](../../internal/engine/information.go)、[mechanics.go](../../internal/engine/mechanics.go) prelude。
- IT 月度回合、`R_op` 预算、反击目录与衰减：[it_authority.go](../../internal/engine/it_authority.go)（10 §5 的实现）。
- 立场三档与权重输入：[top_stance.go](../../internal/engine/top_stance.go)、[16 §3.2](./16-preparation-outcome-design.md)。
- 筹备暴露、审计没收、出发与堡外追踪：[37 分册](../story/endings/37-rebellion-event-spec.md)、[escape_prep_audit_seizure.js](../../events/crisis/escape_prep_audit_seizure.js)、[escape_settlement.go](../../internal/service/escape_settlement.go)、[escape_outside.go](../../internal/service/escape_outside.go)。
- 身份线四节点与封顶：[20-chain-agent-compromised.md](../story/endings/20-chain-agent-compromised.md)、[events/special/identity_*.js 与 agent_erasure_staged.js](../../events/special/)。
- 1 号反应链门槛与时点阶梯：[content_events.go](../../internal/engine/content_events.go)（`OrderPressure`/`Silo1UltimatumPressure`/`Silo1AgitationPropaganda`）、[world.go](../../internal/service/world.go)、[silo1_purge_omen.js](../../events/special/silo1_purge_omen.js)。
- 征兆聚合与呈现纪律：[lines_progress.go](../../internal/service/lines_progress.go)、[29-failure-endings.md](../story/endings/29-failure-endings.md)。
- AP 经济与淡忘：[17-information-economy-design.md](./17-information-economy-design.md)。
- 渗透月数与公平重调判据：[18-time-permeation-design.md](./18-time-permeation-design.md) §3.3/§4/§7。
- 测试账纪律与旋钮分类：[ADR-0011](../../knowledge/adr/0011-opening-bias-needs-test-retuning.md)、[19-opening-variability-design.md](./19-opening-variability-design.md)。
- 反制绑定的既有 TODO：[silo1_purge_exposed.js](../../events/special/silo1_purge_exposed.js)、[roadmap](../roadmap.md)（"追踪的反制入口"）。

# 同类策略游戏机制拆解（供 Silo40 借鉴）

> 研究日期：2026-11-12。覆盖 8 款对照游戏（每款 8 个机制维度）+ 3 款落选游戏简评。
>
> 配套文档：[横向机制对照与借鉴建议](mechanics-comparison.md)

## 0. 研究目的与方法

**目的**：为 Silo40（《羊毛战记》世界观单机策略游戏：万人地堡、cohort 级思潮演化、
政治派系、人均资源经济、按月 tick、事件驱动叙事、多结局）挑选机制层可对照的同类
游戏，拆解其设计事实，提炼可借鉴点。

**选型标准**：以"群体/思潮/派系演化 + 资源约束 + 事件驱动 + 多结局"为核心玩法的
策略/模拟经营游戏；每款必须至少在一个机制维度上与 Silo40 直接同构。

**方法与来源规则**：

- 来源优先级：P0 官方（官网/官方 wiki/开发者博客/访谈/GDC）> P1 一线媒体深度报道
  > P2 可信社区 wiki（含解包数值）与严肃设计分析 > P3 论坛（仅佐证）。
- 每条结论标注来源编号 `[n]`，文末 Sources 列表给出名称与 URL。
- 数值与公式若来自社区解包/二手分析，标注可信度（高/中/低）。
- 明确区分两类陈述：
  - **【设计事实】**：有可靠来源支撑的设计描述；
  - **【推断】**：研究者基于多个事实的推理，附推理依据，不作为设计依据直接使用。
- 未找到可靠来源的内容标注"未找到可靠来源"，不编造。

## 1. 选型说明

| 游戏 | 年份/开发商 | 与 Silo40 的对照点 |
|---|---|---|
| Frostpunk | 2018 / 11 bit studios | 社会思潮计量（希望/不满）+ 道德法律树 + 资源约束 + 结算式结局评判 |
| This War of Mine | 2014 / 11 bit studios | 小群体心理状态链 + 道德抉择的延迟后果 + 战争结束式结局 |
| Crusader Kings 3 | 2020 / Paradox | opinion 数值体系 + 派系演化 + 人物能动性 + 跨代延迟后果 |
| Democracy 4 | 2020-22 / Positech | 政策→民意反馈回路 + 选民群体建模 + 选举判定 |
| RimWorld | 2018 / Ludeon | 事件导演（storyteller）+ 个体心理崩溃链 + 威胁缩放 |
| Cultist Simulator | 2018 / Weather Factory | 卡牌化事件 + 计时器进程 + 暗线叙事 + 刻意隐晦的反馈 |
| Suzerain | 2020 / Torpor Games | 政治抉择后果链 + 隐藏状态 flag + 多结局网络 |
| Reigns | 2016 / Nerial | 二元抉择 + 隐性计量条平衡 + 即死判定 + 轮回结构 |

**落选说明**（候选但机制重心错位）：

- **Dwarf Fortress**：个体级超深模拟（每个生物独立身心模型），机制重心在物理/历史
  模拟而非思潮-抉择闭环，实现规模远超本项目。
- **Oxygen Not Included**：物理/热力学/气体液体模拟是核心玩法， duplicant 压力系统
  只是附属约束，与 Silo40 的政治叙事重心错位。
- **Fallout Shelter**：浅层计时器房间管理，居民无思潮分化、无派系、无多结局叙事，
  可对照维度太少。

（以上落选判断为【推断】，依据为三款游戏公开的核心玩法描述；详细机制见正文 B 部分。）

## 2. 逐游戏拆解

### 2.1 Frostpunk（2018 / 11 bit studios）

**核心循环**【设计事实】
官方自称"the first society survival game"：以中央 Generator 为圆心建设环形城市 →
采集/生产资源（coal、wood、steel、raw food→rations）→ 通过 Workshop 研发科技树、
通过 Book of Laws 立法（有冷却时间）→ 维持两条全局情绪条 Hope 与 Discontent →
应对按日程到来的降温与最终 Great Storm。立法解锁新建筑或能力，形成
"危机→立法→新建筑/新代价"的循环 [A1][A3]。

**状态与资源建模**【设计事实】
粒度为**群体/抽象**：人口按类别计数（工人/工程师/儿童/病人/重伤员），不逐个模拟
个体需求；Hope、Discontent 是城市级单一数值条。个体市民仅作视觉单位呈现（可行走、
可冻伤截肢），无独立人格/情绪状态机。资源为离散整数（coal/wood/steel/rations/
steam cores）。温度以 10°C 为一档 Heat level 建模，-20°C 设为"零点"，需累计
+2 Heat 才达最舒适档（解包分析，可信度：高）[A1][A2]。

**时间推进**【设计事实】
实时可暂停、可加速，以"小时"为最小刻度、以"天"为叙事节点。工作日制度：默认
10h（8:00–18:00），Extended Shift 14h，Emergency Shift 24h。降温按固定日程发生
（如 Day4→-40°C、Day6→-50°C），Great Storm 约在 Day 43 前后到来并持续数日、
期间温度骤降至约 -150°C（攻略口径，可信度：中）[A1][A7]。

**随机性与可复现性**【设计事实+推断】
天气为**脚本化/确定性**而非随机：有天气预报可预知未来降温，便于规划；【推断】
这使单局高度可复现、可被"背板"优化。部分事件含随机成分（决斗可能死人、童工
事故），但核心压力曲线是设计好的。有论坛证据显示触发 Winterhome 事件过早会使
天气日程"重新同步"，说明日程与剧情事件挂钩而非纯按天（可信度：低，P3）[A8]。

**抉择后果的延迟与叠加**【设计事实】
后果显著延迟且叠加：签署解锁建筑的法律即"许诺建造"，违约累积不满；Extended
Shift 持续小幅抬升 Discontent；Emergency Shift 第二次使用触发脚本化过劳死。
开发者 Czaplarski："你设定的法律不可逆，但未来的法律可以修正过去的决定"
（如先 Radical Treatment 致截肢、后研发义肢让其重新有用）；Kubica 强调玩家
**在当下无法预知分支法律的后续后果**。官方明确采用"温水煮青蛙"（boiling frog）
设计：激进化与威权化是**渐进而非突变**，直至临界点玩家自立为先知/独裁者
（P0，开发者 Jakub Stokalski 与 Marta Fijak 在《Procedural Storytelling in Game
Design》中的表述，经官方 Steam 帖转述）[A3][A5]。

**失败/结局判定**【设计事实】
失败：Hope 归零或 Discontent 满 → 给出最后通牒，通常要求 2 天（48h）内把某条
改善 15%，否则被流放（banished/deposed）。New Home 胜利条件 = 撑过 Great
Storm。结局**逐条回读你签过的法律**作道德审判，显示"The city survived"及
（若签署越界法律）"We have crossed the line / But was it worth it" [A4][A7]。

**玩家反馈通道**【设计事实】
底部 Hope/Discontent 双条、建筑温度悬停提示、天气预报横幅、事件弹窗与市民请愿、
热区积雪融化的视觉反馈、结局审判文字。反馈以"城市整体情绪+温度"为主，不做
个体级心情播报 [A1][A4]。

**刻意不做什么**【设计事实+推断】
无战斗/军事；不做个体深度模拟；过程中**不即时道德说教**，审判留到结局（学者
Jaime Oliveros 称其"lack of judgment"，P0 官方帖）；以供暖半径约束城市扩张，
刻意抑制传统城建的无限蔓延 [A3][A6]。

**Law/Purpose 双树（Faith vs Order）**【设计事实】
Book of Laws 分 Adaptation（适应，开局可用）与 Purpose（目的，后置）。Purpose 下
Faith 与 Order **互斥**。两树末端法律（New Faith / New Order）**彻底移除 Hope**，
改以 obedience 替代，并自动建造 Execution Platform（处决台）用于大幅压低
Discontent。"越界"法律 = Protector of the Truth、Righteous Denunciation、Pledge
of Loyalty、Forceful Persuasion，签署后结局显示 crossed the line [A1]。

**越界法律设计意图（child labor 等）**【设计事实，P0/P1】
官方渠道（学者访谈经官方 Steam 帖发布）解释：童工是**最早**提供的法令之一，
"从此向下滑"（From here downward），为后续更极端法令设定心理阈值；从"only
safe jobs"滑向"all jobs"极其自然。设计意图是制造"道德测试场"，让玩家在生存
压力下自我审视，而非直接谴责。开发者 Czaplarski："我坚信当涉及群体时，道德
规则是不同的"——领导者须为整个城邦负责，某些对个体不可接受的决定在群体存续
压力下会变得"危险地容易"（PC Gamer 开发者访谈，官方 Steam 页托管）[A3][A5]。

**派系与诉求反馈（与 Silo40 政治派系直接相关）**【设计事实】
后期当生存压力缓解，市民会**自行结成派系（factions）并制造麻烦**，官方开发者
日志展示了 Londoners 派系。开发者强调"不存在单向沟通——你的人民也会对你说话"：
市民以请愿/请求形式主动提出诉求（如要求童工、要求探索外界），玩家选择回应或
忽视，忽视会累积 Discontent [A5][A6]。

### 2.2 This War of Mine（2014 / 11 bit studios）

**核心循环**【设计事实】
双相位循环：**白天**（约 6–8 时起至 20 时）因狙击手封锁无法外出，只能在避难所
内制造/修缮/做饭/治疗/睡觉/接待来访者；**夜晚**（约 20 时至次日 5–6 时）派一名
幸存者外出 scavenging（搜刮），其余人睡觉或守夜。循环压力来自食物/燃料/药品
稀缺、伤病、夜间袭击与道德抉择。官方《The Little Ones》页明确"白天狙击手把你
困在避难所，夜晚才能外出"（P0）[A12]。

**状态与资源建模**【设计事实】
粒度为**个体级**：每个幸存者是具名实体，有独立饥饿/疲劳/伤病/情绪状态与背包
格数，性格（psychotype）硬编码于角色模板、全程不变。情绪为隐藏数值"morale"，
按阈值映射到 5 档显示状态（解包，可信度：高）：Content（0~‑19）、普通
（‑20~‑39）、Sad（‑40~‑59）、Depressed（‑60~‑79）、Broken（‑80~‑99）；全员
初始 ‑30，技术上不存在高于 0 的情绪。性格四型：Altruistic（最敏感）/Homie/
Survivalist/Egoist（最麻木），叠加 Apathic/Insensitive/Aggressive 情绪修正。
同一事件按性格扣不同分（如杀平民：Altruistic ‑27、Egoist ‑14，未含修正）
[A10][A18]。

**时间推进**【设计事实】
实时可暂停/可加速，以"天"为单位推进，昼夜交替驱动玩法切换。战争总长**随机但
开局预定**，依剧本/所选角色组而定，常见 35–45 天（社区口径，可信度：中）；
收音机会在 ceasefire 临近时给出预告 [A16][A17]。

**随机性与可复现性**【设计事实+推断】
战争天数、搜刮点物资刷新、夜间事件与访客敲门含随机成分；但角色性格、剧本结构
固定。【推断】可复现性低于 Frostpunk，单局更依赖临场应变而非背板。

**抉择后果的延迟与叠加**【设计事实】
后果延迟且滚雪球：偷窃/杀伤平民会按性格扣 morale，且**在场与不在场者扣分不同**；
拒绝求助（如拒给儿童药物）同样累积创伤。情绪会传染——一人崩溃会拖累他人。
抑郁（约 ‑50 起）会触发夜间斗殴（Egoist 概率最高，x4；Egoist+Aggressive 阈值
‑35、概率 x8–12）。Broken 角色失控，可能离家出走或自杀，进而引发连锁崩溃
[A10][A14]。

**失败/结局判定**【设计事实】
失败 = 全员死亡或全部离开，画面打出 "You didn't make it"。胜利 = 至少一人活到
ceasefire 日。结局 **Epilogue** 仅在抵达停火日后可看，按 5 类事件回读每个角色
的行为：杀人/抢劫计入 Karma（坏）结局点数，助人计入 Good 结局点数，死亡/
abandonment 计入 Sad/Karma；角色战后的照片/命运即其行为的后果 [A13][A14]。

**玩家反馈通道**【设计事实】
以**文字状态行+角色台词**为主（hungry/tired/sad/depressed/broken，及角色自言
自语的绝望/抱怨），无可见数值情绪条、游戏也不提示如何化解情绪危机（解包站特别
指出"缺乏可见指示器"）。道德后果通过角色对话与结局照片传达，而非计量条
[A10][A18]。

**刻意不做什么**【设计事实，P0/P1】
开发者 Pawel Miechowski 明确：灵感来自萨拉热窝围城与幸存者回忆录，核心洞察是
"最糟的不是身体挑战，而是人必须背负的情绪代价"；立意是"游戏不必好玩"（a
game doesn't need to be fun），严肃对待题材、"不做浅薄说教"（without shallow
moralization）。刻意不做：战斗爽感（玩家是平民，战斗是绝望自卫）、即时道德
评分/数值情绪条、对玩家选择的即时审判 [A11][A15]。
### 2.3 Crusader Kings 3（2020 / Paradox Development Studio）

官方设计目标（Dev Diary #0，CCO Doomdark）：Character Focus（"CK 明确且毫不含糊地
关于个体角色"）、Player Freedom and Progression、Player Stories（事件须与底层
模拟相关/有影响/沉浸，玩家记住的是"自己的故事"）、Approachability [B1]。

**核心循环**【设计事实】
扮演统治者 → 通过联姻、授予头衔、战争、scheme（密谋）管理封臣与家族 → 角色死亡
后自动扮演继承人，王朝血脉不可中断 [B1][B16]；无可玩继承人即 game over
[B15][B16]。无胜利条件，目标即王朝存续本身。

**状态与资源建模**【设计事实】
状态全部挂在角色上：五维技能（diplomacy/martial/stewardship/intrigue/learning，
上限 100）、互斥人格特质、health、stress、dread、gold/prestige/piety [B6]。

- **opinion**：每个角色对每个其他角色各持一个 opinion，数值 = 修饰符之和（CK2
  设计源头为 -100~+100 单值 [B12]）；CK3 实际可越界（如 Nemesis -120）[B5]。
  多数修饰符随时间衰减至 0 [B5]。继承时继承人临时继承封臣对先君的 opinion：
  正面 25%、负面 50% [B5]。clan 封臣的税收/兵力贡献直接由对领主 opinion 决定 [B4]。
- **vassal stance**：7 种（Belligerent/Courtly/Glory Hound/Parochial/Zealot/
  Minor Landholder/Minority），由 AI 人格决定，偏好对应继承人（继位时 +30 或
  -20 opinion）[B4]。powerful vassal 未进 council 则 -40 opinion [B4]。
- **county 层**：control 0–100（+0.1/月，每点低于 100 → -1% 税、-0.5% 兵力）；
  popular opinion 过低 → populist/peasant faction [B7]。

**时间推进**【设计事实】
可暂停实时（pausable real-time），5 档速度，重要事件自动暂停 [B9]；开局
867/1066，终局 1453（可设 No End Date）[B8][B10]。月度 tick 驱动：dread 以
0.5/月向 natural dread 漂移 [B6]、control +0.1/月 [B7]、opinion 修饰符衰减 [B5]。

**随机性与可复现性**【设计事实】
全局 RNG 种子存于存档，读回同一存档随机序列复现，save-scumming 基本无效
[B14]（P3 佐证）；debug 模式可用 random_seed 参数指定种子，官方用它跑自动化
benchmark 保证行为可复现 [B13]。Ironman：禁手动存档与切换角色，每 6 个月自动
存档 + 退出时存档 [B8]。

**抉择后果的延迟与叠加**【设计事实】
factions 是后果累积的标准模型：faction military power = 成员军力总和/领主军力；
比值 > 80% 阈值 → discontent 累积（超出越多越快，>110% 明显加速），< 80% 则回落；
discontent 100% → 发 ultimatum，领主接受（-20 dread）或拒绝 → 内战 [B2][B4][B11]。
五类：independence（归属感：de jure/文化/宗教）、claimant（opinion 驱动）、
liberty（降王权）、populist 与 peasant（county opinion 驱动；peasant 无最低军力
要求、discontent 恒速增长）[B2]。加入硬条件：无地、未成年、友/情人/盟友、停战、
Strong Hook、神权政体、opinion ≥ 80 → 不得加入；terrified 不能加入（discontent
已在上涨时除外），intimidated 降低加入意愿 [B17][B18][B11]。

stress：违背人格特质行事累积，突破 100/200/300 触发 mental break（1–3 级），
达 400 强制 3 级并 -100 stress（上限 400）；选项为获得 coping mechanism 特质或
改变人格特质；冷却 5 年（Mental Resilience perk 8 年）；3 级可致自伤、死亡、
杀害廷臣、强制退位 [B6]。

dread：酷刑/处决增加，释放囚犯或接受派系要求减少；dread ≥ 对方 boldness+20 →
intimidated，≥ boldness+45 → terrified（Brave 永不 intimidated，Craven 恒
terrified）[B6]。

**失败/结局判定**【设计事实】
无胜利条件。game over：死亡/被废后无可玩王朝继承人、继承人政体不可玩、失去全部
头衔且无王朝成员可接手、到达 1453 [B15][B8]。"结局"是涌现状态（王朝灭绝、国土
分裂），非脚本化结局。

**玩家反馈通道**【设计事实】
嵌套 tooltip（"tooltips in tooltips"，蓝色高亮文本逐层悬停）+ 游戏内 encyclopedia
（内容由脚本文件自动生成，恒与版本一致）+ reactive advice + alerts（仅提示可采取
行动解决的事项）+ Issues/Suggestions/Notifications/Toasts 分层 [B3]。opinion 可
完全分解为修饰符列表，faction UI 展示军力/discontent/成员 [B11]。设计原则：
"挑战应来自掌握系统，而非寻找信息" [B3]。

**刻意不做什么**【设计事实+推断】
官方明言 "go deep rather than wide"：发售即砍掉 CK2 多个系统（游牧、商业共和国）
[B1]；Character Focus 即刻意不做国家模拟器；省份叛乱不再随机，完全归因于
county opinion（取代 CK2 随机省份起义）[B2]；无胜利条件、终局日期可关 [B8]。
【推断】经济刻意抽象为 holdings 税收 + 领地收入，无人均生产/贸易模拟。

### 2.4 Democracy 4（EA 2020 / 正式版 2022 / Positech Games）

定位：单人开发者（Cliff Harris）政治策略游戏，"run the country simulation"。
官方描述："Built on a custom-built neural network designed to model the
opinions, beliefs, thoughts and biases of thousands of virtual citizens" [B31]。

**核心循环**【设计事实】
扮演总理/总统 → 调整政策滑条、立法/废法、应对 events/dilemmas/situations →
每回合一个季度 → 约每 16 回合（4 年）大选 → 胜选连任继续执政 [B30][B21]。
无传统胜利条件："游戏不能被'赢'，只能因选举失败或被暗杀而输——All political
careers end in failure" [B30]。任期数可配置（有限/无限）[B30]。

**状态与资源建模**【设计事实】
模拟引擎为对象-效果图（"neural network"式）：选民、选民群体、政策、统计值、
事件皆为对象，以 effects 连接，全部由 CSV 定义，数值归一化 0–1 [B26]。

- **选民**：2,000 名模拟 AI 选民，每人代表 真实人口/2000 公民 [B19][B27]。
  每人处于 liberal↔conservative、socialist↔capitalist 光谱与三档收入组
  （low/middle/wealthy）；同时隶属多个 voter group（约 20 个 + Everyone），
  认同度 >50% 才加入；成员身份随政策流动（如农民脱贫后从 poor 移入 middle
  income）[B30]。
- 群体 happiness 是理论平均值；个体投票是其所属全部群体影响的净结果
  （"socialist 群体 87% 满意也可能投反对票"）[B30]。
- 每群体独立追踪 **cynicism**（选前讨好/选后背叛/政策反复所致，随时间衰减）与
  **complacency**（把好政策视为理所当然；每次连任后上限提高）[B30]。
- 个体选民有 **party loyalty**（忠诚可压倒理性分析，可悬停查看分解）；approval
  在阈值附近摆动的即 floating voters（摇摆层）[B20][B19]。
- **政策效果公式**格式 `[target],[含 x 的表达式],[inertia]`，可引用其他对象
  （如酒税收入×酒类消费量），运算符 + - / ^ [B26]。
- 财政：政策成本/收入随滑条缩放，部长能力修正效果，债务/GDP → 信用评级
  [B26][B27]；全球经济按正弦波周期运行 [B27]。注意：min/max_income 纯展示用，
  "对模拟无影响" [B27]。

**时间推进**【设计事实】
回合制，1 回合 = 1 季度（3 个月）[B30][B19]；选举约每 16 回合 [B21]。inertia
使效果按 N 回合移动平均生效，上限 32 回合（8 年，技术限制）[B21][B26]；政策实施
另有爬坡期（ghost slider 显示目标值）[B30]。

**随机性与可复现性**【设计事实】
开发者原话："Ironically there are very very few random numbers in the
game…"——模拟层是确定性公式系统（CSV 方程），玩家常因高 inertia 误以为游戏
随机 [B21]。是否存在全局种子/可复现机制：未找到可靠来源。

**抉择后果的延迟与叠加**【设计事实】
① inertia 为核心延迟机制：例 AdultEducationSubsidies 对 Education 的效果
inertia=4，滑条变更需 4 回合完全生效 [B26]；官方长链示例：提高科学预算 → 2 年后
量子计算突破 → 股市 → GDP → 工资 → 支持率 [B19]。② 选举时机效应：选前讨好/选后
背叛/反复横跳 → 对应群体 cynicism [B30]。③ complacency 跨任期累积：执政越久
核心支持者越难取悦 [B30]。④ 思潮转化极慢：socialist↔capitalist 成员变化受
inertia 约束，另设可选 "opinion inertia boost" 滑条（0–3×）三倍化该延迟 [B21]。
⑤ 部分政策效果隐藏，待情境触发才显现 [B30]。

**失败/结局判定**【设计事实】
两种失败：选举落败、被暗杀 [B30]。暗杀风险源于激进改革激怒极端组织，投资安保
可降低，intel 屏幕可预警 [B29]。无脚本化结局。连任判定：两党制下 approval >50%
的选民投给玩家；三党制投票阈值为 40/60（比例代表制 33/66），且阈值随选民满意度
重心漂移（反对党向中间靠拢）[B25]。多党模式：得票 >50%（不计弃权）直接获胜；
不足 50% 但为最大党 → 联合执政，每回合 political capital 大幅削减，盟友党以
政治资本为筹码要求政策交易 [B32]。

**玩家反馈通道**【设计事实】
主界面为图标网络 + 动态箭头（绿=正效应/红=负效应，箭头速度=强度），悬停追溯
因果链 [B30]；polling 屏双线：voting intention vs average approval + approval
分布气泡（20 桶，区分铁票与软支持）[B19]；focus group 随机抽样选民可个体下钻
[B30]；parties 屏散点绘制全部 2,000 选民，悬停看 loyalty 分解 [B20]；每群体有
满意度条 + 成员变化图 [B30]；"changes since election" 图（红/绿/黑情绪标记，
刻意避免党派价值判断）[B28]。

**刻意不做什么**【设计事实】
"Overcomplexity is a feature, not a bug"：刻意不简化信息过载，要玩家体验真实
总理的压力，"经常不得不凭直觉而非分析做决策" [B22]。不建模外交/地缘政治：各国外交
情境太独特、太复杂，而 "income tax is basically income tax" 具有跨文化普适性
[B32]。模拟只覆盖"让游戏可玩且相对准确的变量子集"，文化差异用国家脚本打补丁
而非建模（如德国生产力加成）[B23]。自认未建模"选民按趋势评判"（选民只评判绝对
水平）与社会议题水涨船高现象 [B24]。定位："a game ABOUT politics, not a
'political' game"，不意图输出立场 [B29]。
### 2.5 RimWorld（2018 / Ludeon Studios）

**核心循环**【设计事实】
三层嵌套时间尺度的实时殖民地模拟：瞬间事件（秒级）/ 日循环（60,000 ticks=1 天）/
战役曲线（15 天 quadrum、60 天一年）[C10]。循环链：建造→分配工作优先级→保障
食物→研究→防御→扩张人口→财富复利；财富正是事件导演（AI storyteller）读取的
变量，因此"循环自我升级，没有外部计时器驱动"[C10]。设计师 Tynan Sylvester 的
公开定位是"故事生成器而非模拟本身"，核心方法论是 apophenia（玩家自行从随机
事件中读出意义）与 Player Model Principle [C7]。

**状态与资源建模**【设计事实】
Pawn = needs + skills + traits + health。Needs 共 16 条（Food/Rest/Recreation/
Beauty/Comfort/Space/Outdoors 等，DLC 增删）[C9]。核心管线 **need → thought →
mood → mental break**：mood 为 0–100 条，游戏从不直接写 mood，而是计算
mood target = 难度基础 mood + 全部活跃 thought 偏移之和，条以 +12/游戏时、
−8/游戏时的速率逼近目标，睡眠时冻结 [C2][C9]。基础 mood 按难度：Peaceful 42 /
Adventure 37 / Strive to Survive 32 / Blood and Dust 27 / Losing is Fun 22 [C2]。
Thought 是纯数据（偏移/时长/阶段/叠加上限），共 667 个阶段（解包统计，可信度中）[C9]。

**时间推进**【设计事实】
连续 tick：60 tick/秒，2,500 ticks=1 小时，60,000 ticks=1 天（1× 速度约 16 分 40
秒现实时间）；可随时暂停。无回合、无月 tick [C10]。

**随机性与可复现性**【设计事实】
世界生成支持用户输入种子：同一种子总是生成相同地形（大陆/河流/山脉），但道路、
派系城镇等人工要素可能变化，即"地形可复现、对局不可复现"[C12]。对局内事件由
storyteller 调度。三个 storyteller 共用同一 incident 池与点数公式，只改"何时开火"
[C3][C8]：

- **Cassandra Classic**：10.6 天固定周期（4.6 天 on / 6 天 off），on 期内 1–2 次
  大威胁、间隔 ≥1.9 天；
- **Phoebe Chillax**：16 天周期（8 on / 8 off），每周期恰好 1 次大威胁；
- **Randy Random**：无周期，每 1,000 ticks 掷一次骰，13 天无大威胁则强制补一次
  ——"带保底的高方差，不是真随机"（解包，可信度：高）[C8]。

威胁预算（官方 wiki 公式）：`raid_points = (wealth_points + pawn_points) ×
threat_scale × starting_factor × adaptation_factor`，clamp 至 [35, 10000]；Randy
在封顶后额外 ×0.5–1.5，任务可按星级 ×1–3 [C4]。财富缩放：storyteller wealth
（建筑折半）经曲线转点数——≤14,000 为 0 点，400,000 为 2,400 点，≥1,000,000
封顶 4,200 点；每名殖民者另计 pawn points，随财富从 15 点升至 140（40 万）/
200（百万）[C4]。Threat scale：Peaceful 0.10 → Losing is Fun 2.20（Custom 0–5）；
starting factor 前 10 天 0.7、第 40 天升至 1.0；adaptation 为隐藏 AdaptDays∈
[−60,+100] 计数，映射 0.4–1.47 乘数，殖民者死亡按人口扣 30 天左右、击倒扣至多
8，前 30 天为保护期 [C4]。

**Incident 冷却与排队**：各事件有独立 repeat cooldown——toxic fallout 第 60 天前
不出现且 90 天不复发，heat wave/cold snap/alphabeavers 30 天，eclipse/psychic
soothe 15 天，Man in Black 60 天，rare thrumbos 13 天，short circuit ≥8 天 [C5]。
代码存在 incident queue 用于延迟/强制事件（补丁日志佐证，机制细节无官方文档，
可信度：中）[C11]。

**抉择后果的延迟与叠加**【设计事实】
Thought 有时长与叠加上限，单日饥饿不会滚成不可逆螺旋；mental break 结束后给
Catharsis +40（最多 5 层、递减）防止连环崩溃 [C1][C9]。Adaptation 是延迟反馈的
橡皮筋：顺风缓慢升难，死人立刻降难 [C4]。财富的惩罚是追溯性的——"你建造的一切
最终都会变成对付你的点数"[C10]。社交记忆长期存在（定向侮辱最多叠 10 次、−33
mood，可连锁触发对方崩溃）[C1]。

**失败/结局判定**【设计事实】
失败 = 全体殖民者死亡/被绑架/离开（Colony End；可以新流浪者重启，最多 7 次）；
结局共 5 种过职员表方式（发射飞船、Royal Ascent、Archonexus、Void Awakening、
Mechhive），场景可选 Planetkiller 倒计时强制死线；Commitment 模式为单存档
permadeath，且可无限沙盒不设强制结局 [C6][C3]。

**玩家反馈通道**【设计事实】
Needs 面板 mood 条 + mood target 三角；每个 thought 可检视，tooltip 能完整解释
任何 mood 变化（"mood 是活跃 thought 列表的纯函数"，这是可读性的结构保证）
[C2][C9]；警报系统（如 "Major break risk x2"）、事件 Letter、History 统计；Dev
模式可直接查看 storyteller 的 Base Points [C4]。

**刻意不做什么**【设计事实】
不做脚本化战役/主线任务；不做强制胜利（存档可无限延续）；storyteller 没有游戏内
实体，只是"调度人格 + 一组乘数"，没有单一难度数字；mood 从不被直接赋值；
Cassandra/Phoebe 用确定性周期而非随机，Randy 也带 13 天保底——刻意拒绝"真随机"；
事件导演既发危机也发礼物（同一时钟、同一注意力池），不做纯恶意难度曲线
[C3][C8][C10]。

**关键阈值**【设计事实】
mental break 阈值 minor 35% / major 20%（=minor×4/7）/ extreme 5%（=minor×1/7），
traits 可移动阈值且封顶 1%–50%（Iron-willed −18、Volatile +15 等）；低于阈值后按
MTB 掷骰：官方 wiki 记 minor 4 天 / major 0.8 天 / extreme 0.5 天 [C1]。
注：P2 分析站 teemo.dev 给出 10/3/0.7 天，与官方 wiki 冲突，本文采信官方 wiki。

### 2.6 Cultist Simulator（2018 / Weather Factory）

**核心循环**【设计事实】
实时"occult solitaire"：一张桌面，两类对象——**卡牌（cards，实体/名词）**与
**动词（verbs，带计时器的进程）**；把卡拖入动词槽位、等倒计时结束、产出新卡、
解锁新动词。早期链路：Work 挣 Funds → 应付 Time 动词的生活成本 → Study 学知识
→ Dream 入 Mansus（漫宿）→ 建立密教 → 仪式飞升。官方定位"roguelike 式叙事
卡牌游戏"，单局 20–40 小时 [C13]。Kennedy 自述最初按 idle game 构想（受 A Dark
Room、A Kittens Game 影响），主题为 "yearning and apocalypse" [C17][C19]。

**状态与资源建模**【设计事实】
**万物皆卡牌**：资源（Health/Reason/Passion/Funds）、人物、知识（8 大法则
Lantern/Forge/Edge/Winter/Heart/Grail/Moth/Knock + Secret Histories，等级 2–14
共 7 档）、证据、季节、情绪全是卡；**动词即进程**：Work/Study/Dream/Explore/
Talk/Time 等，各带独立计时器与多个槽位，同时并行 [C20][C22]。Reason/Passion 是
可再生卡：投入动词期间被占用、完成后归还；升级属性靠 Erudition/Glimmering/
Vitality 中间卡 + Funds [C22]。卡牌组合即配方（官方示例：Secret Histories 1 +
Occult Scrap 1 → 招募卡，GDC 幻灯片）[C14]。

**时间推进**【设计事实】
真实时间连续推进，无回合。各动词计时器独立、时长不一：常见约 60 秒（职员工作
60s、Explore 60s+60s 两段、Time 每 60s 扣 1 Funds），但存在 29.7s、80.2s 等实例
（GDC 幻灯片截图）、Sleep 空转 90s（可信度：中高）[C14][C21][C22][C24]。
**Time 动词自动运行、不可停**：每 60 秒消耗 1 Funds（生活成本），无钱→饥饿→
损耗 Health→可能死亡；同时每分钟左右掷出一个"季节"事件 [C21][C22][C25]。
可暂停与快进；Kennedy 曾想限制暂停，最终为无障碍保留 [C17]。

**随机性与可复现性**【设计事实】
未找到可靠来源说明存在可输入种子；随机性体现在：探索/远征的随机遭遇与抽卡、
Time 动词的季节抽取、学习副产物（Erudition 可能变 Dread）。结构上 "deliberately
roguelike"，单局不可逆，跨局由 Legacy 继承系统衔接 [C14][C13][C19]。

**抉择后果的延迟与叠加**【设计事实】
负面状态多为潜伏链：**Restlessness 60 秒后转为 Dread**；**证据链** Notoriety/
Mystique →（Suspicion 季节）Tentative Evidence → Damning Evidence → 逮捕 =
game over；Dread/Fascination 卡本身 180 秒自然消散，但一旦被 Season of
Despair/Visions 抽中，"The Clouds Part/Trembling in the Air" 会持续吸卡，
**集满 3 张即结局**；Season of Sickness 把 Health 转 Affliction，不治转
Decrepitude；Season of Ambitions 给 Restlessness [C21][C23]。后果跨局叠加：
本局的教团/遗产成为下一局开局（Legacy）[C13][C19]。

**失败/结局判定**【设计事实】
失败判定多为"计数达标"式：3 张 Dread（绝望）、证据链走完被捕、Health 耗尽等。
胜利 = 沿 Mansus 层层开门、以对应法则仪式飞升；官方宣传明确"在健全与疯狂的
胜利条件之间选择"（sane and insane victory conditions）[C26]。另有非飞升结局
"Evening Falls"（文本自评 "This might be considered a victory."，见 GDC 幻灯片）；
社区 wiki 列 Ascension/Change/Threshold/The Palest/Apostle/Exile 等结局（含 DLC）
[C14][C20]。

**玩家反馈通道**【设计事实】
**刻意隐晦**：无教程、无任务日志、无目标清单。反馈通道只剩：卡牌 flavor text、
点击空槽位的提示（"会告诉你该放什么……有时"）、卡面标签与图标。开发最后一个月
才补"面包屑"：文本母题 + 机制模式 + UI 暗示，引导但不点破 [C15]。Kennedy 原话：
"我想做'如果魔法真实存在、学习魔法'的体验……当游戏显得晦涩、玩家感到困惑时，
'aha!' 时刻最激动人心。但困惑的玩家很快就会失去耐心"——解法是在玩家"已有半个
答案"时给 UI 暗示，"让交互像游戏与玩家的对话"[C16]。

**刻意不做什么**【设计事实】
不做教程（"与 Cultist 的设计根本冲突"；若给了引导，玩家会期待处处有提示，毁掉
"来之不易的矛盾信息营造的悬疑体验"）[C15][C19]；不给类型标签（"no genre"）；
不用过场/直述讲故事，只做暗示（洛夫克拉夫特式）[C19]；GDC 总结的核心原则
**"Don't simulate where you can imply"（能暗示就不模拟）**，自称 "apophenian
design"，与 RimWorld 的 apophenia 理念同源但方向相反——RimWorld 用透明系统让
玩家投射，Cultist 用信息缺口让玩家脑补 [C14]；信息过载是刻意的（"同时有多件
进行中的相关事务……这是 deliberate 的"）[C17]。
### 2.7 Suzerain（2020 / Torpor Games）

**定位**：文本驱动的政治叙事 RPG，扮演虚构国家 Sordland 新任总统 Anton Rayne，
在 1954–1957 的四年任期内处理经济衰退、宪政改革、边境冲突与派系斗争。官方自称
"100 万+词分支政治史诗、50+ 角色、9 个主要结局 + 25 个以上子结局"（P0，Torpor
官方 press kit [D1]；PSN 商店页同口径 [D2]）。词数口径随版本增长：发行商 Fellow
Traveller 早期 press kit 为 "500K 词"（P0 [D26]），中文 Steam 页早期文案为 "20 万字"
（P0 [D28]），现行官方口径为 "1M+"（P0 [D1]）——内容经 2.0/3.x 大版本持续扩写。
设计立意上，总监 Ata Sergey Nowak 在访谈中明确"把理解与共情作为 Suzerain 的核心
概念"（P1 [D25]）。

**核心循环**【设计事实】
对话/会议 → 做出决策（法案签署、人事任命、经济指令、外交表态）→ 通过新闻、报告、
局势面板观察后果 → 进入下一 turn。没有资源微操，几乎所有交互都是"读文本—做选择"。
开发团队明确以"领导的人性层面、通过对话而非数值来玩"为立基点，刻意区别于传统数值
策略游戏（P0，制作人 Ata Sergey Nowak 自述 [D3]）。

**状态与资源建模**【设计事实】
双层状态系统：

- **意识形态追踪值**：官方 2020 年 7 月开发月报披露，后台追踪 Democratic /
  Capitalist / Sollist / Malenyevist 四类倾向，"在选项背后加值"（adding values
  on choices behind the scenes），四者互不排斥，从序章开始累积，"用于塑造选择、
  结局与其他应对玩家的元素"（P0 [D4]）。结局界面展示基于该系统的 political
  compass 四象限（P2 [D5]）。
- **变量规模**：Ata 自述游戏用 Articy v2（可视化脚本 + 数据库）承载 "45 万词与
  数千个变量"（thousands of variables），并明言"游戏建立在数百个选择之上"
  （hundreds of choices）（P0 [D3]）。这是"数百决策 flag"说法的直接开发者出处。
- **开局状态**：序章通过出身地、难度、大学专业、学生团体、青年组织等选择设定初始
  Personal Wealth (PW)、Public Opinion (PO)、Bludish Opinion (BO)、Economy (E)、
  Government Budget (GB) 等数值（P2，社区 wiki 解包整理 [D6]）。

**时间推进**【设计事实】
离散回合制：Ata 自述"把游戏建为 11 个 turn，以三个 turn 为一组来管理叙事钩子与
峰值"，多数议题在第三、四章收束形成高潮（P0 [D3]）。现行版本社区 wiki 记为序章
+ 12 个正章 turn + 尾声 + 总结，每个 turn 覆盖任期四个月（P2 [D6]）——11 与 12
的差异【推断】为版本更新（2.0/3.x）增补所致，2020 年 10 月官方广播明确 "Turn 11
是最后一个游戏回合"（P0 [D7]）。官方还特意移除了章节开头的年份标注，"Suzerain
的大部分以 moments（时刻）呈现"，以减少年份带来的叙事约束（P0 [D4]）。

**随机性与可复现性**【设计事实+推断】
游戏采用单一自动存档、"每个决定都是最终的，不允许读取更早的存档"（官方商店文案
口径，P2 媒体转引 [D8]；P2 评测确认"单自动存档槽，所有选择永久" [D9]）。
【推断】全部结果由 flag/变量确定性决定，未见任何 RNG 机制的描述；相同选择序列应
导向相同结局。未找到官方对"确定性"的直接表述，此为基于机制描述的推断。

**抉择后果的延迟与叠加**【设计事实】
Ata 自述团队把故事拆成家族、Bludish 少数族裔、国防、经济、改革等多条 storyline，
"由于游戏建立在数百个选择上，一条故事线可以强烈影响其他故事线"，交叉管理是开发
最大难点之一（P0 [D3]）。官方广播亦提及"决策与场景之间的空隙被慢慢缝合，以增强
故事元素的互联性"（P0 [D7]）。社区 wiki 记录的典型延迟链（P2 解包 [D5]）：
国有化两大财团且不部署秘密警察 → 选举前演讲遇刺（Oligarch Assassination）；
向 Rumburg 割地 → 触发 Iosef/Valken 政变或流亡线；腐败交易被反腐机构反查 →
弹劾。这些条件链表明后果以"flag 累积 + 阈值/组合判定"实现，而非即时结算。

**失败/结局判定**【设计事实】
官方口径 9 大结局 + 25+ 子结局（P0 [D1][D2]）；社区 wiki 口径为 9 大结局 + 20 个
子结局并逐一列出（P2 [D5]）。Sordland 篇 9 类：Reelection（连任）、Assassination
（遇刺）、Lost Election（败选）、Impeached（弹劾）、Early Retirement（退休）、
Path to Exile（流亡）、Coup d'état（政变）、Hail the Queen（战败亡国）、
World At War（核战）。失败多为"触怒关键势力"的阈值判定：激怒寡头/Blud 人/军方/
议会即分别导向刺杀、政变、弹劾；对 Rumburg 战争准备不足则亡国，与核大国结盟开战
则核冬天（P2 [D5]）。【推断】结局判定是"多 flag 组合的优先级树"而非单一分数。

**玩家反馈通道**【设计事实】
多层反馈：六份立场各异的报纸（各有字体与文风，作为"更个人化的选择反馈"）；
overview 面板分 active policies / situations 两类并带严重度等级（如 nation-wide
revolt）；新闻、报告、codex 提供世界观与后果信息（P0 [D3]）。官方月报称报告/新闻
的作用是"给玩家预警、让玩家能反应、传递 lore"（P0 [D10]）。1.1 补丁追加结局总结屏
与 political compass（P0 [D11]）。注意：反馈以定性文本为主，刻意不给全局数值仪表盘。

**刻意不做什么**【设计事实+推断】
(a) 删除了"政治关系面板"，官方理由："它暴露太多，剥夺了玩家自行揣摩幕后权力关系
的发现感"（P0 [D4]）；(b) 移除章节年份标注，避免年份制造叙事义务（P0 [D4]）；
(c) 不做数值化国家模拟——立项初衷即"对话而非数值"（P0 [D3]）；(d) 不允许读档
回滚，强制玩家承担决定（P2 [D8][D9]）。【推断】这些取舍共同服务于"政治抉择的重量
感"：信息不完全 + 结果不可逆 + 反馈延迟，是其后果链可信的根源。

### 2.8 Reigns（2016 / Nerial）

**定位**：二元抉择 + 隐性计量的"滑动治国"游戏。Nerial（François Alliot 一人负责
开发与文案）开发、Devolver Digital 发行，2016-08-11 上线（P0 [D12][D13]）。
Alliot 自述立项动机："嘲讽我们的社会处理复杂性的方式"，以 Brexit 式的二元公投为
靶子，刻意制造"滑动的轻浮与后果的残酷之间的落差"（P0，Alliot 为 Polygon 撰写的
亲笔文章 [D14]）。

**核心循环**【设计事实】
抽到一张卡（顾问/事件）→ 左滑（拒绝）或右滑（同意）→ 四个计量条（church 教会 /
people 人民 / army 军队 / treasury 国库）增减 → 任何一条**清零或拉满即当场死亡**，
换下一任国王继续（P1，The Guardian / Washington Post 评测口径，经 P2 百科转述
[D15]；官方商店文案同口径 [D12]）。一张卡 = 统治一年。Alliot 概括："我们围绕这个
循环构建整个游戏，确保玩家无论做什么都能继续滑下一张"（P1 [D16]）。

**状态与资源建模**【设计事实】
(a) 四计量条只显示图标填充程度，**不给数值**；(b) 角色级隐藏状态：顾问被连续拒绝
会累积"压力"，表现为眨眼加快，随后带来的负面后果加重，最终可能行刺（Alliot 亲述
的"没人会注意到的小系统"）（P0 [D14]，P1 访谈再次确认 [D16]）；(c) 内容规模：
初版 **750 张卡**（P0 [D14]）；官方 1.07 更新日志明确"新增 100+ 张卡、3 种新死法"
（P0 [D17]）；续作 Her Majesty "1200+ 张卡，比 Reigns 多 30%"（P1 [D16]）；
(d) 早期版本曾在选项旁显示精确数值后果，开发后期 Alliot 刻意"打破 Reigns"、把数值
与信息从界面中拿掉，让焦点回到决策本身（P2，Alliot 2017 年在巴黎高师 InGame
研讨会的发言记录 [D18]）。

**时间推进**【设计事实】
离散年进制：官方文案"统治的每一年都会带来另一个重要——看似随机——的请求"
（P0 [D12]）。王朝跨代延续：死亡后由继承人开新局，但世界状态（已解锁角色、事件线、
诅咒进度）跨朝代保留；部分事件横跨数个世纪（火刑女巫→科学启蒙→恶魔线）
（P0 [D12][D14]）。

**随机性与可复现性**【设计事实】
**不可复现**：叙事采用"概率驱动"而非分支树——每个选择把玩家导向一个卡池，池内按
权重随机抽卡，"有些选项比其他更可能出现"（P2 [D18]；Alliot 自述"用概率在 750
张卡间制造不同路径：窄巷（线性链）、大路（跨世纪主题混合）、旷野（纯随机问题）"
（P0 [D14]））。Alliot 强调卡牌的随机性"对制造意义感与随时可能脱轨的可能性很重要"，
做法是在随机表象下埋隐藏结构，"一次会话里耍三四次这种把戏，就会让玩家觉得每张卡
都可能有深意"（P1 [D20]；P0 [D14]）。官方文案则刻意用 "seemingly random" 措辞
（P0 [D12]）。

**抉择后果的延迟与叠加**【设计事实】
Alliot："很快你就会明白某些决定有长期影响：添加角色、故事碎片、效果或不同玩法"
（P0 [D14]）。典型链：拒绝顾问→压力累积→刺杀；与 Senator 的绑架线→数年后的恶魔
弱点情报（P2 [D19]）；恶魔线本身横跨 666 / 1332 / 1998 三个年份节点（见下）。
跨朝代解锁（Royal Challenges 完成目标→解锁新卡）把单局后果升级为永久内容
（P0 [D12]）。

**失败/结局判定**【设计事实】
四计量条触底/触顶即死，且**同一极值按附加 flag 给出不同死法**（如 army 满：有儿子
→被绞死；无视刺客→被斩首）（P2，SuperSoluce 攻略逐条列出 [D19]）。死法数量：
游戏内 "Memento mori" 页面收录 **26 种死法**（P2，SuperSoluce 逐条列出解锁条件
并注明其在游戏选项页内 [D19]；官方文案只说 "find new ways to die"，未给数字
（P0 [D12]）——26 为社区攻略对游戏内收集页的计数，可信度中高）。恶魔机制：666 年
恶魔首次现身，揭示国王曾出卖灵魂换取永恒权力、被困死亡轮回；1332 年再现，预告
1998 年最后到访并留下诅咒；真结局需完成多步隐藏任务（女巫、Pungeon、骷髅 Kloc、
草莓），把诅咒用在"不可能再死"的骷髅上，迫使恶魔履约释放国王（P2 [D15][D19]）。
Alliot 自述恶魔是开发后期加入的，"需要一个东西解释为什么这些国王记得一切、顾问会
重复——你被困在诅咒里，而玩游戏的方式就是设法骗过恶魔"（P1 [D16]）。

**玩家反馈通道**【设计事实】
极简反馈：顶部四个图标（无数字）、角色眨眼、死亡结算文字、Memento mori 收集页、
成就/目标。Alliot 的设计观："完美系统（唯一最优解）是把玩家关进监狱；系统必须不
平衡，给犹豫、好坏选择、长短期后果留空间"（P2 [D18]）。P1 批评视角：The Guardian
认为部分选择"无厘头、后果不明"（经 [D15] 转述）——【推断】这正是隐性计量的代价：
反馈不足时玩家失去因果感。

**刻意不做什么**【设计事实+推断】
(a) 不做数值透明——删掉了早期的数值后果预览（P2 [D18]）；(b) 不做传统分支树——
用概率卡池替代，牺牲可复现性换取"随时可能脱轨"的张力（P0 [D14]）；(c) 不做多选项
——只给二元选择，作为对"复杂政治被压成 yes/no"的讽刺本体（P0 [D14]）；(d) 不做
教程——卡牌物理交互自解释（P1 [D16]）。【推断】对 Silo40 的核心启示：Reigns 证明
"计量条隐藏 + 极值即死 + 死法差异化"能用最小界面承载强重开性；但其单局短命与
Silo40 的长程地堡经营相反，需保留"隐性阈值死亡"而放弃"高频换主角"。

### 2.9 落选游戏简评（机制重心错位）

**Dwarf Fortress（Bay 12 Games，2006 经典版 / 2022 Steam 版）**
机制核心：**个体级超深模拟**。官方自称"有史以来最深、最复杂的世界模拟"：每个矮人
有独立模拟的性格、技能、身体部位与社交关系，世界生成包含文明兴衰、历史人物与文化；
玩法是殖民地建造 + 涌现式叙事（P0 [D21]；自 2003 年持续开发至今 [D24]）。
为何不适合：Silo40 的万人地堡必须在 **cohort（群体）粒度**上建模思潮与派系，而 DF
把算力与玩家注意力全部投向两三百个微观个体；其叙事是无目标的涌现沙盒，没有政治决策
主线、没有结局结构，奇幻基调也与《羊毛战记》的封闭社会政治叙事不兼容。可借鉴的只有
"个体记忆/性格驱动涌现事件"的思路，整体规模与主题不匹配。

**Oxygen Not Included（Klei Entertainment，2019）**
机制核心：**物理/热力学为核心的殖民地生存**。官方描述聚焦氧气、温度与食物的持续
威胁：气体/液体管网模拟、热力学温控是常驻挑战；duplicant（复制人）的压力系统是
附属层——"每个 duplicant 面对压力有不同且可能具破坏性的反应"，靠休闲、住宿与食物
压制（P0 [D22]）。
为何不适合：ONI 的核心挑战是工程闭环优化（气体、液体、温度的物理场管理），人物心理
只是物理系统的衍生效应；Silo40 的"人均资源经济"本质是**政治分配问题**（谁得到多少、
凭什么），而非实时物理模拟。把月 tick 的叙事决策游戏接到连续物理引擎上会淹没政治
玩法，主题上 ONI 是科学求生而非社会契约。

**Fallout Shelter（Bethesda Game Studios，2015）**
机制核心：**浅层计时器管理**。建造功能房间、按 S.P.E.C.I.A.L. 属性安置 dweller、
等待资源产出计时器、派探险者去废土（也是计时回报）、处理突袭与生育繁衍；免费 + 内购
（P0 [D23]）。
为何不适合：题材上它离 Silo40 最近（同为地下避难所社会），但机制深度最浅——决策只有
"房间布局 + 人员分配"，没有思潮、派系与道德困境，也没有结局叙事；其循环为 F2P 等待
计时服务，与 Silo40 的单机月 tick 叙事体验根本冲突。它证明了"地堡"题材的受众存在，
但不能作为机制参照。

## 3. Sources

> 编号前缀对应研究线程：A = Frostpunk/TWoM，B = CK3/Democracy 4，C = RimWorld/
> Cultist Simulator，D = Suzerain/Reigns/落选游戏。分级：P0 官方 / P1 一线媒体 /
> P2 社区 wiki 与严肃设计分析 / P3 论坛（仅佐证）。所有 URL 均为研究时实际访问过的
> 链接（个别经 Wayback Machine 快照读取，已注明）。

### Track A：Frostpunk / This War of Mine

- [A1] P2 — Frostpunk Wiki and Guides: Book of laws — https://frostpunk.game-vault.net/wiki/Book_of_laws
- [A2] P2 — Vigaroe: Frostpunk Research Analysis: Heating（解包分析）— https://vigaroe.com/Analyses/Frostpunk/Heat
- [A3] P0 — Steam 官方新闻: Revisiting New London #6: The Legacy of Frostpunk（含学者访谈与 boiling frog 转述）— https://store.steampowered.com/news/posts/?appids=323190
- [A4] P1 — gamepressure: Frostpunk Guide — What happens when Discontent or Hope reach critical values — https://guides.gamepressure.com/frostpunk/guide.asp?ID=44412
- [A5] P1 — Steam 官方新闻页（托管 PC Gamer 开发者访谈 "You can't please everyone in survival city builder Frostpunk"）— https://store.steampowered.com/news/posts/?appids=323190&enddate=1522343332
- [A6] P1 — GamesBeat: Why survival sim Frostpunk is eerily relevant — https://gamesbeat.com/why-survival-sim-frostpunk-is-eerily-relevant/
- [A7] P2 — PwrDown: How to Survive the Final Storm（攻略口径）— https://www.pwrdown.com/gaming/frostpunk-how-to-survive-the-final-storm/
- [A8] P3 — GOG 论坛: Temp drop schedule changed（天气与事件挂钩佐证）— https://www.gog.com/forum/frostpunk/temp_drop_schedule_changed_in_saved_game_after_upgrade_to_101/post2
- [A10] P2 — This War of Mine 俄语社区站: Мораль（士气解包数值）— https://thiswarofmine.ru/comment/2297
- [A11] P1 — TechRaptor: Art and This War of Mine: Interview with 11 bit Studios（Pawel Miechowski）— https://techraptor.net/gaming/interview/art-and-war-mine-interview-11-bit-studios
- [A12] P0 — This War of Mine: The Little Ones 官方页（昼夜/狙击手设定）— https://tlo.thiswarofmine.com/de/
- [A13] P2 — blackgawer: This war of mine game end（Epilogue/结局点数）— https://blackgawer.weebly.com/this-war-of-mine-game-end.html
- [A14] P1 — Kill Screen: This War of Mine is not fun — https://www.killscreen.com/war-mine-not-fun-you-should-still-play-it/
- [A15] P1 — Ganker: How Polish devs turned war into a groundbreaking game（萨拉热窝灵感）— https://ganker.com/this-war-of-mine-how-a-group-of-polish-devs-turned-war-into-a-deeply-personal-and-groundbreaking-game-43896/
- [A16] P3 — GOG 论坛: Beat the game tips（战争天数随机）— https://www.gog.com/forum/this_war_of_mine/beat_the_game_my_selected_tips_possible_spoilers/post6
- [A17] P3 — Fuwanovel 论坛（停火常见 35–45 天）— https://forums.fuwanovel.moe/topic/16904-this-war-of-mine/
- [A18] P2 — atwiki: This War Of Mine WIKI: TIPS（情绪状态链）— https://w.atwiki.jp/thiswarofmine/pages/16.html

### Track B：Crusader Kings 3 / Democracy 4

- [B1] P0 — CK3 Dev Diary #0 – The Vision（Doomdark/CCO，2019-10-24）— https://forum.paradoxplaza.com/forum/developer-diary/ck3-dev-diary-0-the-vision.1265472/
- [B2] P0 — CK3 Dev Diary #19 – Factions and Civil Wars（Baron von Shoes，2020-03-24）— https://forum.paradoxplaza.com/forum/developer-diary/ck3-dev-diary-19-factions-and-civil-wars.1363951/
- [B3] P0 — CK3 Dev Diary #16 – Tutorials and Tooltips and Encyclopedias, Oh My!（2020-03-03）— https://forum.paradoxplaza.com/forum/developer-diary/ck3-dev-diary-16-tutorials-and-tooltips-and-encyclopedias-oh-my.1345581/
- [B4] P0 — CK3 官方 Wiki: Subjects（vassal stances、powerful vassals、clan opinion→税收；大页面存在抓取截断，经多个索引修订交叉印证）— https://ck3.paradoxwikis.com/Subjects
- [B5] P0 — CK3 官方 Wiki: Character（opinion 衰减、先君继承 25%/50%、Nemesis -120）— https://ck3.paradoxwikis.com/Character
- [B6] P0 — CK3 官方 Wiki: Attributes（Dread/Stress：0–100、0.5/月漂移、+20/+45 判定、stress 阈值与 mental break）— https://ck3.paradoxwikis.com/Dread
- [B7] P0 — CK3 官方 Wiki: County（control、popular opinion）— https://ck3.paradoxwikis.com/Corruption
- [B8] P0 — CK3 官方 Wiki: Game rules（ironman、end date；页面直接抓取失败，经多个索引修订一致）— https://ck3.paradoxwikis.com/Game_rules
- [B9] P0 — CK3 官方 Wiki: Keyboard shortcuts（速度档位）— https://ck3.paradoxwikis.com/Keyboard_shortcuts
- [B10] P0 — CK3 官方 Wiki: Achievements（"End of an Era: Play until 1453"）— https://ck3.paradoxwikis.com/Achievements
- [B11] P1 — GameWatcher: Crusader Kings 3 Factions Guide（2020-09-14；discontent 加速、加入条件清单为媒体观察，中等可信）— https://www.gamewatcher.com/news/crusader-kings-3-faction
- [B12] P0 — CK2 Dev Diary #1（opinion -100~+100 单值设计源头）— https://forum.paradoxplaza.com/forum/developer-diary/crusader-kings-dev-diary-1.503167/page-12
- [B13] P0 — Paradox 论坛: Anatomy of a Game – Startup and Loading（random_seed 命令行参数用于自动化 benchmark）— https://forum.paradoxplaza.com/forum/threads/anatomy-of-a-game-startup-and-loading.1482960/
- [B14] P3 — Paradox 论坛: When does the game calculate child stats（RNG 种子存于存档、读档复现，佐证）— https://forum.paradoxplaza.com/forum/threads/when-does-the-game-calculate-child-stats.1496975/
- [B15] P3 — Paradox 论坛: CK III – NO PLAYER HEIR AVAILABLE & GAME OVER（game over 条件清单，佐证）— https://forum.paradoxplaza.com/forum/threads/ck-iii-no-player-heir-available-game-over-bug.1447510/
- [B16] P2 — HandWiki: Software:Crusader Kings III（死亡后扮演继承人、1453 终局）— https://www.handwiki.org/wiki/Software:Crusader_Kings_III
- [B17] P0 — CK3 官方 Wiki: Vassals（faction 加入条件）— https://ck3.paradoxwikis.com/Vassals
- [B18] P3 — Paradox 论坛: CK III – Terrified character is in faction（intimidated/terrified 与派系加入，佐证）— https://forum.paradoxplaza.com/forum/threads/ck-iii-terrified-character-is-in-faction.1536064/
- [B19] P0 — Cliffski's Blog: On the visualization of voter approval distribution in Democracy 4（2022-03-02：2,000 AI 选民、floating voters、延迟链条示例）— https://www.positech.co.uk/cliffsblog/2022-03-02/on-the-visualization-of-voter-approval-distribution-in-democracy-4/
- [B20] P0 — Cliffski's Blog: Modelling voter happiness in a video game is… interesting but also horrendous（2020-10-21：50% 投票阈值、turnout、party loyalty）— https://www.positech.co.uk/cliffsblog/2020-10-21/modelling-voter-happiness-in-a-video-game-is-interesting-but-also-horrendous/
- [B21] P0 — Cliffski's Blog: Inertia scaling in Democracy 4 (new feature)（2022-11-20：inertia 上限 32=8 年、16 回合选举、"随机数极少"、opinion inertia boost 0–3×）— https://www.positech.co.uk/cliffsblog/2022-11-20/inertia-scaling-in-democracy-4-new-feature/
- [B22] P0 — Cliffski's Blog: Democracy 4's overcomplexity is by design（2023-02-19）— https://www.positech.co.uk/cliffsblog/2023-02-19/democracy-4s-overcomplexity-is-by-design/
- [B23] P0 — Cliffski's Blog: Modelling cultural differences in Democracy 4（2020-11-16：单一模拟模型、国家脚本补丁）— https://www.positech.co.uk/cliffsblog/2020-11-16/modelling-cultural-differences-in-democracy-4/
- [B24] P0 — Cliffski's Blog: An update on balancing Democracy 4 for longer games（2022-02-24：complacency、绝对水平评判的局限）— https://www.positech.co.uk/cliffsblog/2022-02-24/an-update-on-balancing-democracy-4-for-longer-games/
- [B25] P0 — Cliffski's Blog: Yet more refinement and complexity regarding party politics in Democracy 4（2022-04-08：40/60 与 33/66 阈值、阈值漂移）— https://www.positech.co.uk/cliffsblog/2022-04-08/yet-more-refinement-and-complexity-regarding-party-politics-in-democracy-4/
- [B26] P0 — Democracy 4 官方 Modding 文档: Basics/Effects（neural network 引擎、CSV 驱动、效果公式格式与 inertia 示例）— https://www.positech.co.uk/democracy4/modding.html
- [B27] P0 — Democracy 4 官方 Modding 文档: Countries（2,000 选民与人口换算、经济正弦波、min/max_income 纯展示）— https://www.positech.co.uk/democracy4/mod_countries.html
- [B28] P0 — Democracy 4 官方 Modding 文档: Simulation values（simulation.csv、0–1 数值、Emotion 标记）— https://www.positech.co.uk/democracy4/mod_simulation.html
- [B29] P0 — Democracy 4 官方 FAQ（中立性声明、暗杀机制解释）— https://www.positech.co.uk/democracy4/faq.html
- [B30] P1 — GamePretty: Democracy 4 Basic Gameplay Guide（2020-09-29；内容与游戏内帮助一致）— https://gamepretty.com/democracy-4-basic-gameplay-guide/
- [B31] P0 — GOG 商店页描述（"custom-built neural network… thousands of virtual citizens"）— https://www.gog.com/es/game/democracy_4
- [B32] P0 — Cliffski's Blog: Democracy 4 – Multi Party Support（2019-11-19：联合政府与 political capital、不做外交政策的设计理由）— https://www.positech.co.uk/cliffsblog/2019-11-19/democracy-4-multi-party-support/
- [B33] P0 — Steam 官方公告（D4 新特性 coalition governments、media reports 等）— https://store.steampowered.com/oldnews/?appids=591370&feed=steam_community_announcements
- [B34] P3 — Positech 论坛: Perceptions 帖（失败状态仅选举/暗杀两种，佐证）— https://forums.positech.co.uk/t/perceptions/17290/8

### Track C：RimWorld / Cultist Simulator

- [C1] P0 — RimWorld Official Wiki: Mental break（阈值 35%/20%/5%、MTB 4/0.8/0.5 天、Catharsis）— https://rimworldwiki.com/wiki/Mental_break
- [C2] P0 — RimWorld Official Wiki: Mood（mood target、难度基础值、趋近速率）— https://rimworldwiki.com/wiki/Mood
- [C3] P0 — RimWorld Official Wiki: AI Storytellers（三 storyteller、无游戏内实体、结局方式）— https://rimworldwiki.com/wiki/Storyteller
- [C4] P0 — RimWorld Official Wiki: Raid points（威胁预算公式、财富曲线、threat scale、adaptation）— https://rimworldwiki.com/wiki/Raid_points
- [C5] P0 — RimWorld Official Wiki: Events（各事件 repeat cooldown）— https://rimworldwiki.com/wiki/Events
- [C6] P0 — RimWorld Official Wiki: Endings（5 种过职员表方式、Planetkiller、Colony End）— https://rimworldwiki.com/wiki/Endings
- [C7] P0 — Tynan Sylvester: The Simulation Dream（2013 演讲文稿，apophenia）— https://tynansylvester.com/2013/06/the-simulation-dream/
- [C8] P2 — Game Design Lab: The AI Storyteller (Event Director)（解包：调度周期、Randy 1000 ticks 掷骰+13 天保底）— https://teemo.dev/game-design/rimworld/systems/the-ai-storyteller/
- [C9] P2 — Game Design Lab: Needs, Mood & Mental Breaks（解包：16 needs、667 thought 阶段；其 MTB 10/3/0.7 天与官方 wiki 冲突，本文采信官方）— https://teemo.dev/game-design/rimworld/systems/needs-mood-and-mental-breaks/
- [C10] P2 — Game Design Lab: Core Loop: Three Nested Timescales — https://teemo.dev/game-design/rimworld/systems/core-loop-three-nested/
- [C11] P3 — UpdateMonitor: RimWorld 1.6.4543 补丁日志（incident queue 存在佐证）— https://www.updatemonitor.com/app/2074920/news/1807332909521747
- [C12] P0 — RimWorld Official Wiki: World generation（种子与可复现性）— https://rimworldwiki.com/wiki/World_generation
- [C13] P0 — Steam 商店页: Cultist Simulator（官方定位、单局时长）— https://store.steampowered.com/app/718670/Cultist_Simulator/
- [C14] P0 — Alexis Kennedy, GDC 2019 演讲幻灯片（卡牌=实体/动词=进程、配方示例、"Don't simulate where you can imply"、Evening Falls）— https://media.gdcvault.com/gdc2019/presentations/Kennedy-Alexis-Cultist-Simulator.pdf
- [C15] P1 — Lottie Bevan: Cultist Simulator: the retrospective（Game Developer/Gamasutra，2018；面包屑、不做教程的理由）— https://www.gamasutra.com/business/cultist-simulator-the-retrospective
- [C16] P0 — Alexis Kennedy Q&A（alexiskennedy.co.uk，2020；"学习魔法"体验、半个答案时给暗示）— https://www.alexiskennedy.co.uk/post/in-the-news-q-a-with-alexis-kennedy-co-founder-of-weather-factory
- [C17] P0 — Weather Factory 博客: The Hands of Time（2017；idle game 起源、暂停的取舍、信息过载是刻意）— https://weatherfactory.biz/hands-of-time
- [C18] P0 — Weather Factory 博客: Devlog: Loose, Tight（2016）— https://weatherfactory.biz/cultist-simulator-devlog-loose-tight/
- [C19] P1 — GameRes:《密教模拟器》Weather Factory 专访（2019 访谈转载；no genre、暗示叙事）— https://www.gameres.com/842914.html
- [C20] P2 — namu.moe: Cultist Simulator 条目（社区 wiki：法则体系、结局列表）— https://m.namu.moe/w/Cultist%20Simulator
- [C21] P3 — Gamerzy Magazine: The Cultist Simulator Guide（季节机制、Dread/Fascination 180 秒、3 张终局整理）— https://gamerzymag.com/the-cultist-simulator-guide-you-need-before-starting/
- [C22] P3 — gameplay.tips: Cultist Simulator Getting Started Guide（Time 动词 60 秒/1 币、动词计时器）— https://gameplay.tips/guides/1937-cultist-simulator.html
- [C23] P3 — Something Awful 论坛 LP 帖（Season of Despair 吸 Dread 机制佐证）— https://forums.somethingawful.com/showthread.php?threadid=3888951
- [C24] P3 — cheatinfo.de: Cultist Simulator（Explore 60s+60s 两段计时）— https://www.cheatinfo.de/files/cultist-simulator.htm
- [C25] P3 — tiflo-games.ru: Cultist Simulator（Time 动词 60 秒/1 币）— https://tiflo-games.ru/cultist-simulator/
- [C26] P0 — Weather Factory 官网游戏介绍页（"Choose between sane and insane victory conditions…"）— https://weatherfactory.biz/page/24/?ak_action=hixjqamo

### Track D：Suzerain / Reigns / 落选游戏

- [D1] P0 — Torpor Games 官方 Press Kit（9 大结局 + 25+ 子结局、100 万+词、50+ 角色）— https://www.torporgames.com/presskit
- [D2] P0 — PlayStation Store《Suzerain》商品页（同口径官方特性列表）— https://store.playstation.com/en-qa/product/UP8520-PPSA31764_00-0933294946319513
- [D3] P0 — Ata Sergey Nowak 自述开发复盘《Suzerain: What Would You Do as President?》（Unity Container；正文经 Wayback 快照读取；含 "11 turns、三 turn 一组"、"450k 词与数千变量（Articy v2）"、"hundreds of choices"、storyline 交叉）— https://unitycontainer.org/suzerain-what-would-you-do-as-president/ （快照：https://web.archive.org/web/2026/https://unitycontainer.org/suzerain-what-would-you-do-as-president/ ）
- [D4] P0 — Suzerain Steam 官方开发月报（2020-07：Democratic/Capitalist/Sollist/Malenyevist 后台追踪值；移除年份标注与政治关系面板）— https://store.steampowered.com/news/posts/?feed=steam_community_announcements&appids=1207650&enddate=1598558407
- [D5] P2 — Suzerain Fandom Wiki: Endings（9 大结局 + 20 子结局清单与触发条件、political compass）— https://suzerain.fandom.com/wiki/Endings
- [D6] P2 — Suzerain Fandom Wiki: Turn（序章 + 12 turn + 尾声；每 turn = 4 个月；序章初始值解包）— https://suzerain.fandom.com/wiki/Turn
- [D7] P0 — Torpor Games 2020 年开发广播合集（10 月广播："Turn 11 是最后一个回合"、"决策—场景空隙缝合"）— https://www.torporgames.com/broadcasts-2020
- [D8] P2（转引官方商店文案）— NSG Reviews《Suzerain》："Every decision you make is final: there is an autosave feature that does not allow loading earlier saves." — https://www.nsgreviews.com/game/5946837
- [D9] P2 — NoobFeed《Suzerain》PC 评测（单自动存档槽、选择永久；三大派系）— https://www.noobfeed.com/games/suzerain
- [D10] P0 — Suzerain Steam 官方开发月报（2020-06：报告/新闻的作用 = 预警/反应/lore；同 [D4] 页面）— 见 [D4]
- [D11] P0 — Torpor Games 2020 年 12 月广播（1.1 "Rayne Train" 补丁新增结局总结屏与 political compass；同 [D7] 页面）— 见 [D7]
- [D12] P0 — Steam《Reigns》英文商店页（四势力平衡、"seemingly random"、跨世纪事件与恶魔、Royal Challenges）— https://store.steampowered.com/app/474750/Reigns/?l=english
- [D13] P0 — Steam《Reigns》中文商店页 — https://store.steampowered.com/app/474750/Reigns/
- [D14] P0 — François Alliot 亲笔文章《How we mixed Tinder and politics to make a premium hit on mobile》（Polygon，2016-09-15：750 张卡、概率路径"窄巷/大路/旷野"、眨眼压力系统、长期后果）— https://www.polygon.com/2016/9/15/12927968/reigns-mobile-pc-ios-royalty-tinder
- [D15] P2 — HandWiki: Software:Reigns (video game)（转述 The Guardian / Washington Post / TouchArcade 评测；666/1332/1998 恶魔线）— https://handwiki.org/wiki/Software:Reigns_(video_game)
- [D16] P1 — PC Gamer: How Reigns and its forthcoming sequel make their 'Tinderocracy' work（Alliot 访谈：循环定义、眨眼=压力、恶魔为后期加入、1.8M 销量、Her Majesty 1200+ 卡）— https://www.pcgamer.com/how-reigns-and-its-forthcoming-sequel-make-their-tinderocracy-work/
- [D17] P0 — App Store《Reigns》页面（版本历史：1.07 "新增 100+ 卡、1 角色、3 种新死法、5 个目标"）— https://apps.apple.com/ge/app/reigns/id1114127463
- [D18] P2 — Julien Bazile《[EN] What Reigns are made of》（Alliot 2017-03 巴黎高师 InGame 研讨会实录：早期数值预览被移除、概率驱动叙事池、"完美系统即监狱"）— https://www.chroniquesvideoludiques.com/en-what-reigns-are-made-of-the-building-and-breaking-of-a-medieval-political-simulator/
- [D19] P2 — SuperSoluce: Astuces Reigns（26 种死法逐条解锁条件，对应游戏内 Memento mori 页；恶魔线完整流程）— https://www.supersoluce.com/astuce/reigns/astuces-reigns
- [D20] P1 — Game Developer（原 Gamasutra）: How Reigns' lead developer made politics feel as goofy as Tinder（随机性下的隐藏深度）— https://www.gamasutra.com/design/how-i-reigns-i-lead-developer-made-politics-feel-as-goofy-as-tinder
- [D21] P0 — Steam《Dwarf Fortress》商店页（"deepest, most intricate simulation"）— https://store.steampowered.com/app/975370/Dwarf_Fortress/?l=english
- [D22] P0 — Steam《Oxygen Not Included》商店页（氧气/温度/食物威胁、duplicant 压力反应）— https://store.steampowered.com/app/457140/Oxygen_Not_Included/?l=english
- [D23] P0 — Steam《Fallout Shelter》商店页（房间建造、dweller 岗位匹配、F2P）— https://store.steampowered.com/app/588430/Fallout_Shelter/?l=english
- [D24] P0 — Bay 12 Games 官网（Dwarf Fortress Classic 持续更新）— https://www.bay12games.com/dwarves/
- [D25] P1 — I.N.T. Magazine 对 Ata Sergey Nowak 的深度访谈（设计哲学：理解与共情为核心；抓取时正文有截断）— https://int-magazine.com/interview/ata-sergey-nowak-of-suzerain/
- [D26] P0 — Fellow Traveller（发行商）Press Kit《Suzerain》（早期口径 "500K-word"）— https://www.fellowtravellerpresskit.com/suzerain
- [D28] P0 — Steam《Suzerain》商店页（中文；早期口径 "20 万字"；1954 年设定）— https://store.steampowered.com/app/1207650/Suzerain/

### 未找到可靠来源的条目（诚实留白）

- Frostpunk 天气日程与剧情事件"重新同步"：仅 P3 论坛佐证（[A8]），正文已标可信度低。
- TWoM 停火具体天数：无官方公开数值，取社区口径（[A16][A17]），已标可信度中。
- Democracy 4 全局种子/可复现机制：开发者称随机数极少，但未找到种子机制的官方说明。
- Suzerain "数百个决策 flag" 的精确数字：开发者只给定性表述（[D3]）。
- Suzerain "无 RNG/确定性" 的直接官方表述：未找到，相关结论已标【推断】。
- Reigns 官方口径的精确死法数：官方仅 "find new ways to die"（[D12]）；26 为社区对
  游戏内收集页的计数（[D19]）。
- Reigns 四计量条数值区间/每卡增减量的官方解包：未找到可靠来源。
- Cultist Simulator 可输入种子的存在性：未找到可靠来源。

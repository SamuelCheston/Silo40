# 同类策略游戏机制拆解（供 Silo40 借鉴） · Frostpunk

> 拆分自 [同类策略游戏机制拆解（供 Silo40 借鉴）](../game-mechanics-breakdown.md)；目录、章节映射与出处见该索引。来源编号（[A*]/[B*]/[C*]/[D*]）见 [10-sources.md](10-sources.md)。

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


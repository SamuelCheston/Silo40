# 同类策略游戏机制拆解（供 Silo40 借鉴） · Cultist Simulator

> 拆分自 [同类策略游戏机制拆解（供 Silo40 借鉴）](../game-mechanics-breakdown.md)；目录、章节映射与出处见该索引。来源编号（[A*]/[B*]/[C*]/[D*]）见 [10-sources.md](10-sources.md)。

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

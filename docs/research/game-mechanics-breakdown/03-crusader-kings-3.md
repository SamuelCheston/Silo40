# 同类策略游戏机制拆解（供 Silo40 借鉴） · Crusader Kings 3

> 拆分自 [同类策略游戏机制拆解（供 Silo40 借鉴）](../game-mechanics-breakdown.md)；目录、章节映射与出处见该索引。来源编号（[A*]/[B*]/[C*]/[D*]）见 [10-sources.md](10-sources.md)。

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


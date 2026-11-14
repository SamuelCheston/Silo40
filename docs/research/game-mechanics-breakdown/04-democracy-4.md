# 同类策略游戏机制拆解（供 Silo40 借鉴） · Democracy 4

> 拆分自 [同类策略游戏机制拆解（供 Silo40 借鉴）](../game-mechanics-breakdown.md)；目录、章节映射与出处见该索引。来源编号（[A*]/[B*]/[C*]/[D*]）见 [10-sources.md](10-sources.md)。

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

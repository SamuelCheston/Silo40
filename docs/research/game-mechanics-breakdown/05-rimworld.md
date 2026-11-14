# 同类策略游戏机制拆解（供 Silo40 借鉴） · RimWorld

> 拆分自 [同类策略游戏机制拆解（供 Silo40 借鉴）](../game-mechanics-breakdown.md)；目录、章节映射与出处见该索引。来源编号（[A*]/[B*]/[C*]/[D*]）见 [10-sources.md](10-sources.md)。

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


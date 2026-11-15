package engine

// 盲区经营 (23 系列) 数值集中定义：决策模型 (23-03 §2.10 参数汇总) 与累积状态
// (23-04 §1/§6.5)。全部为推荐初值，等待 sim 校准覆盖 (TODO(balance))。
//
// 属格划分：本文件属盲区经营格 (W5)——03 卷六因子决策与 04 卷 §1 累积状态
// (Cov/Bias/Traces)。注入系数 (23-05) 属观测核心格，见 blackbox_injection_constants.go；
// 拆穿/怀疑数值 (23-04 §3/§4) 属缺口通道格，不在本文件。
//
// 时间常数纪律 (23-00 裁决 5)：τ_cov=30 与 22 系列 τ_decay=15 (BlackBoxTauDecay)
// 是两个刻意独立的常数——盲区是制度化存量 (口径、排班表、维护规程、回执格式)，
// 比快起快落的戒心更耐久。不得复用 22 系列常量。

// —— 非零裁量地板与基线权重 (23-03 §2.9，本系列灵魂) ——

const (
	// BlindSpotWeightFloor ε_w：候选权重地板。任何处境下任何候选永不归零——
	// 比 w_idle 低三个数量级：地板候选抽中概率极低但裁量余地永在。TODO(balance)
	BlindSpotWeightFloor = 0.001
	// BlindSpotIntensityFloor ε_ℓ：力度权重地板。TODO(balance)
	BlindSpotIntensityFloor = 0.02
	// BlindSpotIdleWeight w_idle：「本月不动手」基线权重，与手段类基础权重同量级，
	// 静默是常态默认。TODO(balance)
	BlindSpotIdleWeight = 1.0
	// BlindSpotIdleID 「不动手」候选 ID (遍历序恒最后)。
	BlindSpotIdleID = "idle"
)

// —— f_mot 自保动机 (23-03 §2.3)：M(Θ)=4Θ(1−Θ) 中段最强、两端受抑 ——

const (
	// BlindSpotMotEps ε_mot：动机地板，任何 Θ 下伪装候选保留最小动机分量。
	// TODO(balance)
	BlindSpotMotEps = 0.02
	// BlindSpotThetaRamp θ_ramp：低段斜坡锚点 (与 B2 阳奉阴违下界对齐)。
	// TODO(balance)
	BlindSpotThetaRamp = 0.30
	// BlindSpotThetaFade θ_fade：高段衰减锚点 (与 B4 抗命/θ_defy 对齐：转向摊牌
	// 筹备，伪装退居次位)。TODO(balance)
	BlindSpotThetaFade = 0.70
)

// —— f_risk 被查风险 (23-03 §2.4)：本地感知监督强度 P_s ——

const (
	// BlindSpotRiskWR / BlindSpotRiskWI w_r / w_i：评级与事实各占一半，
	// P_s 上界 1.2 容许叠加。TODO(balance)
	BlindSpotRiskWR = 0.6
	BlindSpotRiskWI = 0.6
	// BlindSpotAuditWindow K_win：稽查事实记忆窗口 (月)。TODO(balance)
	BlindSpotAuditWindow = 6
	// BlindSpotAuditSat I_sat：窗口内稽查类事实感知饱和计数。TODO(balance)
	BlindSpotAuditSat = 3.0
	// BlindSpotRiskNeed a_need：监督越紧 → 报告面越需要打理 (动机侧抬升)。
	// TODO(balance)
	BlindSpotRiskNeed = 0.6
	// BlindSpotRiskDanger a_danger：监督越紧 → 高拆穿率手段越危险 (手段侧压制)。
	// a_danger > a_need：净效果是紧监督下向低拆穿率通道迁移而非停手。TODO(balance)
	BlindSpotRiskDanger = 1.2
)

// —— f_attn 上级注意力投影 (23-03 §2.5)：只用可观测投影，不读 A(t) ——

const (
	// BlindSpotAttnZeta ζ：注意力弹性 (流量翻倍时 u=0.5 手段 ×0.61)。TODO(balance)
	BlindSpotAttnZeta = 1.0
	// BlindSpotUltWindow K_ult：通牒记忆窗口 (月)。TODO(balance)
	BlindSpotUltWindow = 3
	// BlindSpotUltShock Δ_ult：通牒等价流量翻倍的注意力冲击。TODO(balance)
	BlindSpotUltShock = 1.0
	// BlindSpotTrafficLo / BlindSpotTrafficHi 流量比截断：骤降是机会窗口但防极端化。
	// TODO(balance)
	BlindSpotTrafficLo = -0.5
	BlindSpotTrafficHi = 1.5
	// BlindSpotIdleAttnSlope idle 静默斜率：注意力上抬时「不动手」轻微抬升。
	// TODO(balance)
	BlindSpotIdleAttnSlope = 0.15
	// BlindSpotShowComply b_show：「表面合规」标签手段在通牒窗口内的额外加成
	// (本轮最小候选集无表面合规手段，参数随谱系声明保留)。TODO(balance)
	BlindSpotShowComply = 1.5
)

// —— f_situ 当前处境 (23-03 §2.6)：R_op 可负担性 + W 成本曲线 + 部门恐慌 ——

const (
	// BlindSpotRReserve R_reserve：负担余量标尺 (≈5 个月净收入量级)。TODO(balance)
	BlindSpotRReserve = 40.0
	// BlindSpotPanicK k_panic：恐慌把倾向推向「稳」的手段，幅度温和。TODO(balance)
	BlindSpotPanicK = 0.6
	// BlindSpotAffordMargin / BlindSpotAffordPow f_afford 形状：余量项系数与开方
	// 指数——区分「伤筋动骨」与「宽裕」，近付不起处陡升。TODO(balance)
	BlindSpotAffordMargin = 0.2
	BlindSpotAffordPow    = 0.5
	// BlindSpotIdlePanicSlope 恐慌对 idle 的抬升：乱局里少做少错。TODO(balance)
	BlindSpotIdlePanicSlope = 0.1
)

// BlindSpotAuditFactKey 近 6 月稽查事实计数键 (SafeguardIncidents)：由缺口通道格
// (W6) 写入，本格只读；缺省 0 (23-03 §2.4 I_recent)。
const BlindSpotAuditFactKey = "audit_fact"

// BlindSpotExposeDivide u_c ≤ 0.3 低拆穿率 / > 0.3 高拆穿率的性情乘子分界
// (23-03 §2.7)。TODO(balance)
const BlindSpotExposeDivide = 0.3

// —— f_disp 长官性情 (23-03 §2.7)：逐堡确定性系数，种子派生、零运行时 RNG ——

const (
	BlindSpotDispCautious    = "cautious"    // 谨慎
	BlindSpotDispSpeculative = "speculative" // 投机
	BlindSpotDispPlodding    = "plodding"    // 守拙
)

// 性情乘子表：列序 = 谨慎/投机/守拙；一个候选可同时命中多行，命中行连乘
// (23-03 §2.7)。全部 TODO(balance)。
var (
	// BlindSpotDispLowExpose 低拆穿率手段 (u_c ≤ 0.3)。
	BlindSpotDispLowExpose = [3]float64{1.10, 1.05, 1.00}
	// BlindSpotDispHighExpose 高拆穿率手段 (u_c > 0.3)。
	BlindSpotDispHighExpose = [3]float64{0.60, 1.15, 0.80}
	// BlindSpotDispTiming timing 通道 (时机类，投机者所长)。
	BlindSpotDispTiming = [3]float64{1.00, 1.30, 1.00}
	// BlindSpotDispInertia 惯性类手段 (对「流程」演戏)。
	BlindSpotDispInertia = [3]float64{1.00, 1.00, 1.30}
	// BlindSpotDispTargeted 针对性类手段。
	BlindSpotDispTargeted = [3]float64{1.00, 1.05, 0.80}
	// BlindSpotDispIdleMul 「不动手」。
	BlindSpotDispIdleMul = [3]float64{1.40, 0.85, 1.20}
)

// —— f_G 认知梯度 (23-03 §2.8)：μ(G) 锚点线性插值，连续斜坡而非硬开关 ——

// BlindSpotMuGAnchors (G, μ) 锚点：G≤0.30 平台 0.15 (残余概率) → 0.40 解锁 1.00
// (斜坡中心压 22-02 §4.2 的 G=0.35 解锁线) → G≥0.65 平台 1.25 (高精度伪装小幅
// 加成)。惯性类 f_G≡1，本表只作用于针对性类。TODO(balance)
var BlindSpotMuGAnchors = [][2]float64{
	{0.00, 0.15},
	{0.30, 0.15},
	{0.40, 1.00},
	{0.65, 1.25},
	{1.00, 1.25},
}

// —— f_inertia 惯性延续 (23-03 §2.9) ——

// BlindSpotInertiaBonus 上月同一选择 (含 idle 延续 idle) 的延续加成：伪装是持续
// 经营，但加成必须温和——倾向性 ≠ 锁死。TODO(balance)
const BlindSpotInertiaBonus = 1.3

// —— 力度三档 (23-03 §3.2)：轻/标准/重 ——

const (
	// BlindSpotW1Base / BlindSpotW2Base / BlindSpotW3Base 力度基权：轻动作为主、
	// 重动作稀有。TODO(balance)
	BlindSpotW1Base = 0.45
	BlindSpotW2Base = 0.40
	BlindSpotW3Base = 0.15
	// BlindSpotCautiousW1 谨慎性情对轻档的加成。TODO(balance)
	BlindSpotCautiousW1 = 0.25
	// BlindSpotW1PSlope 紧监督偏向轻动作的 P_s 系数。TODO(balance)
	BlindSpotW1PSlope = 0.30
	// BlindSpotW3Courage 重动作胆量 (性情) 系数。TODO(balance)
	BlindSpotW3Courage = 0.20
	// BlindSpotW3Margin 重动作余量 (R_op) 系数。TODO(balance)
	BlindSpotW3Margin = 0.10
	// BlindSpotHeavyCostMult 重动作成本乘子 (= m_cost(3))。TODO(balance)
	BlindSpotHeavyCostMult = 1.6
	// BlindSpotCapGOnset / BlindSpotCapGSpan cap_s 解锁斜坡锚点 0.25→0.40 (中心压
	// G=0.35 解锁线)。TODO(balance)
	BlindSpotCapGOnset = 0.25
	BlindSpotCapGSpan  = 0.15
	// BlindSpotCapBase / BlindSpotCapWeight 能力基底/权重：能力不足不禁止重动作，
	// 只压低其权重。TODO(balance)
	BlindSpotCapBase   = 0.4
	BlindSpotCapWeight = 0.6
)

// BlindSpotBoldness b_χ 胆量值：谨慎/投机/守拙。TODO(balance)
var BlindSpotBoldness = [3]float64{0.1, 0.6, 0.3}

// 力度结算乘子 (ℓ=1 轻 / 2 标准 / 3 重)，消费归 01/02/04 卷。TODO(balance)
var (
	// BlindSpotMCost m_cost：力度对成本的乘子。
	BlindSpotMCost = [3]float64{0.6, 1.0, 1.6}
	// BlindSpotMExpose m_u：力度对拆穿率/痕迹量级的乘子。
	BlindSpotMExpose = [3]float64{0.7, 1.0, 1.4}
	// BlindSpotMEff m_eff：力度对效果的乘子。
	BlindSpotMEff = [3]float64{0.5, 1.0, 1.8}
)

// —— 累积状态 (23-04 §1)：世界侧簿记，零 RNG ——

const (
	// BlindSpotTauCov τ_cov：盲区厚度衰减时间常数 (月)，半衰期 ≈21 月。长于 22
	// 系列 τ_decay=15 (BlackBoxTauDecay)：戒心随减压快速回落，经营出的流程跨人员
	// 存续 (23-00 裁决 5)。两常量独立定义，禁止复用。TODO(balance)
	BlindSpotTauCov = 30.0
	// BlindSpotDeltaCov δ_cov：单次手段月度盲区增量基准 (强度 1)。单次收益必须
	// 极小，持续经营 30–40 月才成厚盲区。TODO(balance)
	BlindSpotDeltaCov = 0.02
	// BlindSpotDeltaCap δ_cap：单通道单月增量上限 (边际递减，防「一月速成盲区」)。
	// TODO(balance)
	BlindSpotDeltaCap = 0.03
	// BlindSpotBiasMax b̄_c：单通道被经营偏置上限 (多通道叠加对齐 22-02 b_max 量级)。
	// TODO(balance)
	BlindSpotBiasMax = 0.10
	// BlindSpotBiasGamma γ_b：Bias–Cov 曲率 (线性起点，留 sim 调)。TODO(balance)
	BlindSpotBiasGamma = 1.0
	// BlindSpotTraceRetain T_retain：痕迹保留期 (月)——纸面档案归档量级，也是统计
	// 形状检测窗口长度；到期条目按月确定性移除，保留期内不衰减。TODO(balance)
	BlindSpotTraceRetain = 36
	// BlindSpotCapAbility ΔB 能力项系数：Δcov ∝ m_eff(ℓ)·(BlindSpotCapAbility +
	// BlindSpotCapAbility·cap_s)——能力归零仍留一半「蛮干」效果 (23-03 §6.2)。
	// TODO(balance)
	BlindSpotCapAbility = 0.5
	// BlindSpotTraceMagBase 痕迹量级基准 (× m_u(ℓ))。TODO(balance)
	BlindSpotTraceMagBase = 0.05
)

// —— 决策日志 (23-03 §5) ——

// BlindSpotLogCap 决策日志环形上限 (NegotiationLogEntry 模式，每堡独立)。
// TODO(balance)
const BlindSpotLogCap = 24

// —— 手段类标签 (23-03 §2.8) ——

const (
	// BlindSpotClassInertia 惯性类：对「流程」演戏，不需要知道流程背后是谁，
	// f_G ≡ 1。
	BlindSpotClassInertia = "inertia"
	// BlindSpotClassTargeted 针对性类：揣摩观察者的伪装，f_G = μ(G)。
	BlindSpotClassTargeted = "targeted"
)

// —— G-01 监控设备最小承载簿记键 (23-01 §6) ——

// BlindSpotEquipmentUseKey suppress_signal 类动作执行时为 equipment 通道登记的
// 世界侧簿记键 (SafeguardIncidents)。
const BlindSpotEquipmentUseKey = "equipment_use"

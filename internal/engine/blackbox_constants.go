package engine

// 22 黑箱学习系列 01 卷 §9 参数总表：世界侧数值集中定义于此。
// 所有数值为推荐初值，等待 sim 校准覆盖 (TODO(balance))。
// 叙事与 UI 不得写死时间描述 (00 卷 §10 纪律)。
//
// 属格划分 (00 卷 §5.2 真值/信念隔离)：
//   - 世界侧物理参数 (真值)：Θ 动力学、E(r,Θ) 曲线、服从脉冲与观测信号
//     特性——黑箱回合 (信念与决策) 禁止引用 (01 卷 §5.5 禁止读心审计)。
//   - 1 号内部参数 (信念/决策)：先验、内部似然、漂移上界假设、伪装率
//     假设形式、决策菜单、压力代价、冷却与日志容量——集中在
//     service/silo1_blackbox_constants.go (黑箱壁禁止决策侧 import engine)，
//     与本表保持独立同值基线 (01 卷 §6.2)。

// —— 世界侧：Θ 动力学 (01 卷 §3.2 候选 A 分段线性加性) ——

const (
	// BlackBoxRCalm r_calm：正常带上界。r ≤ r_calm 时 growth = 0、全速衰减。
	// TODO(balance)
	BlackBoxRCalm = 0.35
	// BlackBoxRFull r_full：增长饱和点。0.85–1.0 留为纯危险余量 (DEATH 线 1.0
	// 之下)。TODO(balance)
	BlackBoxRFull = 0.85
	// BlackBoxGrowthMax g_max：最大月增长率。反应快 (00 卷 §3 定稿、裁决 13)：
	// 满压 1–2 月明显抬头，5 月到危险区，约 6 月进叫板档、8–9 月到 Θ=1。
	// TODO(balance)
	BlackBoxGrowthMax = 0.12
	// BlackBoxTauDecay τ_decay：衰减时间常数 (月)。裁决 12–18 月量级；一年安抚
	// 0.60 → ≈0.27；建立快消解慢 = 1:3 (00 卷 §3 定稿)。TODO(balance)
	BlackBoxTauDecay = 15.0
)

// —— 世界侧：执行强度单峰 E(r, Θ) (01 卷 §4 候选 A 高斯型) ——

const (
	// BlackBoxPeakR0 r_0：Θ=0 时峰位。略高于 r_calm：例行压力稍带紧迫感时
	// 执行最卖力。TODO(balance)
	BlackBoxPeakR0 = 0.45
	// BlackBoxPeakShiftK k_shift：峰移斜率 (单调左移)。Θ=1 时峰位 0.15：只剩
	// 温和指派还能命中甜蜜点。TODO(balance)
	BlackBoxPeakShiftK = 0.30
	// BlackBoxPeakE0 e_0：Θ=0 时峰高。最忠诚 95%。TODO(balance)
	BlackBoxPeakE0 = 0.95
	// BlackBoxPeakE1 e_1：Θ=1 时峰高。彻底离心也有 40% 表面功夫 (阳奉阴违封顶
	// 出力)。TODO(balance)
	BlackBoxPeakE1 = 0.40
	// BlackBoxPeakSigma0 σ_0：Θ=0 时峰宽。顺从期容错窗口约 ±0.2。TODO(balance)
	BlackBoxPeakSigma0 = 0.20
	// BlackBoxPeakWidthK k_width：峰宽收窄斜率 (单调收窄)。Θ=1 时宽 0.10：
	// 容错减半。TODO(balance)
	BlackBoxPeakWidthK = 0.50
)

// —— 世界侧：G10 表面服从脉冲 (22-02 §3.5 ΔE_show、02 卷 §6 缺口清单) ——

const (
	// BlackBoxObediencePulseDeltaE ΔE_show：通牒后 E 的一次性脉冲幅值 (持续
	// 1–2 月，随脉冲剩余月递减)，使通牒刚结束时的执行观测量系统性偏乐观。
	// TODO(balance)
	BlackBoxObediencePulseDeltaE = 0.25
	// BlackBoxObediencePulseMonths 通牒签发时置入 ObediencePulse 的初始月数
	// (22-02 §3.5「持续 1–2 月」)。TODO(balance)
	BlackBoxObediencePulseMonths = 2
)

// —— 共享阈值：Θ 语义档 (01 卷 §1) ——

const (
	// BlackBoxThetaDanger θ_danger：危险区阈值。越过之后人人自危，公开叫板成为
	// 可能；与 N=20 箱边界 (12/20) 对齐。TODO(balance)
	BlackBoxThetaDanger = 0.60
	// BlackBoxThetaDefy θ_defy：叫板阈值。03 卷 §2.1 B4 抗命档下界；满压约
	// 6 月到达。TODO(balance)
	BlackBoxThetaDefy = 0.70
)

// —— 1 号内部：信念先验/漂移与伪装假设/月度决策 (01 卷 §5.1/§5.3/§5.4/§6) ——
// 集中在 service/silo1_blackbox_constants.go (黑箱壁禁止决策侧 import engine)。

// —— 观测信号特性 (01 卷 §5.2；信号类型键见 model.BlackBoxSignal*) ——

const (
	// BlackBoxDelayReceiptDelay δ：回执/执行回报时延 (月)。TODO(balance)
	BlackBoxDelayReceiptDelay = 1
	// BlackBoxDelayExecutionGap δ：复核差异 (月)。TODO(balance)
	BlackBoxDelayExecutionGap = 2
	// BlackBoxDelayLineTone δ：例行汇报态度 (月)。与 SILO1_DECEIVE
	// DelayMonths=2 同量级。TODO(balance)
	BlackBoxDelayLineTone = 2
)

const (
	// BlackBoxNoiseReceiptDelay σ：回执最硬。TODO(balance)
	BlackBoxNoiseReceiptDelay = 0.10
	// BlackBoxNoiseExecutionGap σ：复核差异。TODO(balance)
	BlackBoxNoiseExecutionGap = 0.12
	// BlackBoxNoiseLineTone σ：态度最软。TODO(balance)
	BlackBoxNoiseLineTone = 0.15
)

const (
	// BlackBoxDisguiseOffsetReceiptDelay d：回执偏移。TODO(balance)
	BlackBoxDisguiseOffsetReceiptDelay = 0.05
	// BlackBoxDisguiseOffsetExecutionGap d：复核偏移。TODO(balance)
	BlackBoxDisguiseOffsetExecutionGap = 0.15
	// BlackBoxDisguiseOffsetLineTone d：越主观的信号越容易被粉饰。TODO(balance)
	BlackBoxDisguiseOffsetLineTone = 0.25
)

// —— 世界侧：伪装概率形式 p_disg(Θ) = clamp01((Θ − onset)/span) (01 卷 §5.2) ——

const (
	// BlackBoxDisguiseOnset 伪装启动点：戒心过半才开始系统性粉饰太平。
	// TODO(balance)
	BlackBoxDisguiseOnset = 0.30
	// BlackBoxDisguiseSpan 伪装爬升跨度。TODO(balance)
	BlackBoxDisguiseSpan = 0.40
)

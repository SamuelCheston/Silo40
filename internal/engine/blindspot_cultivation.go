package engine

// 盲区经营决策 (23 系列 03 卷) + 累积状态写入 (23 系列 04 卷 §1)。属盲区经营格
// (W5) 实现。
//
// 纪律 (23-03 §3.5 确定性清单)：
//   - 手段选择是六因子连乘权重 + ε 地板的三级抽取 (是否动手 → 手段类 → 力度)，
//     不是确定性 argmax——权重被可读状态塑形但不锁死 (ADR-0007 同构)；
//   - 权重计算全程零 RNG (纯确定性浮点)，随机只出现在三次抽取；因子按 §2.2 总表
//     序连乘、候选按谱系表 ID 升序累加，无 map 序浮点累加；
//   - 每堡每月 DrawFloat64 消耗 1–3 次，堡内顺序写死 (动手 → 手段 → 力度)；堡按
//     SiloID 升序 (调用方保证)；
//   - 门控：BlindSpot.Initialized 为 false (功能关闭/旧档) → 直接返回，零 RNG、
//     状态零变化，与既有 parity 基线逐位一致。
//
// 真值/信念隔离：本函数是世界侧簿记，读 Θ 真值合法 (22-00 §5)；不读 1 号内部
// 状态 (b_s、L、A(t) 真值一律禁用，23-03 §2.4/§2.5 裁决)。

import (
	"fmt"
	"math"
	"strings"

	"silo40/internal/model"
)

// blindSpotMethod 候选手段类声明 (23-03 §2.1/§6.1)：ID、通道、类标签、基础权重
// B_c、基础成本、基础拆穿率 u_c、痕迹模板。本轮取最小集：每通道一个惯性类手段。
// 数值为 sim 起点 (B_c/u_c 量级参照 01/02 卷谱系表)，全部 TODO(balance)。
type blindSpotMethod struct {
	id         string
	channel    string
	classTag   string
	baseWeight float64
	baseCost   float64
	exposeRate float64 // u_c ∈ [0,1]
	traceKind  string  // 04 卷 §1.4 痕迹类型
}

// blindSpotMethods 候选谱系表，固定按 ID 升序 (确定性纪律；idle 不入表、恒排最后)。
//
//	equipment「维修单制造盲区」(01 卷 M1 形态) / schedule「迎检排班」(01 卷 M2) /
//	reports「报表口径裁剪」(02 卷 M01/M04 形态) / ledger「回执整形」(02 卷 M06) /
//	census「底册口径」(02 卷 M12) / timing「坏消息压点上报」(02 卷 M14/M15)。
var blindSpotMethods = []blindSpotMethod{
	{id: "census_roster", channel: model.BlindSpotChannelCensus, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 3, exposeRate: 0.25, traceKind: "census_diff"},
	{id: "equipment_repair", channel: model.BlindSpotChannelEquipment, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 4, exposeRate: 0.15, traceKind: "repair_slip"},
	{id: "ledger_receipt", channel: model.BlindSpotChannelLedger, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 2, exposeRate: 0.20, traceKind: "receipt_shape"},
	{id: "reports_trim", channel: model.BlindSpotChannelReports, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 2, exposeRate: 0.10, traceKind: "standard_change"},
	{id: "schedule_roster", channel: model.BlindSpotChannelSchedule, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 5, exposeRate: 0.15, traceKind: "roster_change"},
	{id: "timing_hold", channel: model.BlindSpotChannelTiming, classTag: BlindSpotClassInertia, baseWeight: 1.0, baseCost: 8, exposeRate: 0.35, traceKind: "submit_time"},
}

// blindSpotFactorReads 一个候选的六因子读数 (可归因，进决策日志)。
type blindSpotFactorReads struct {
	mot     float64 // f_mot 自保动机
	risk    float64 // f_risk 被查风险
	attn    float64 // f_attn 上级注意力投影
	situ    float64 // f_situ 当前处境
	disp    float64 // f_disp 长官性情
	cog     float64 // f_G 认知梯度
	inertia float64 // f_inertia 惯性延续
}

// blindSpotInputs 因子输入快照 (全部为堡内可观测世界事实，零 RNG 读取)。
type blindSpotInputs struct {
	theta        float64
	g            float64
	rOp          float64 // 4.6 支出后余额
	w            float64
	rEff         float64
	auditFacts   int
	pS           float64 // 感知监督强度 ∈ [0,1.2]
	deltaTraffic float64
	panicMean    float64
	disp         string
	dispIdx      int
	prevActive   string
}

// RunBlindSpotCultivationMonthly RunLocalPhase 步骤 4.62 (23 系列 03 卷)：
// 每堡每月一次带权重的随机伪装决策；抽中的 (手段类 + 力度) 写入 04 卷累积状态
// (盲区厚度月度增量 + 痕迹账本) 并留决策审计日志。未动手月只写日志与零增量
// 衰减。门控：功能关闭零 RNG、状态零变化。
func RunBlindSpotCultivationMonthly(silo *model.Silo) {
	// 门控 (23-03 §3.3)：功能关闭 / 已退出结算 → 整体跳过，零 RNG。
	if silo == nil || silo.VictoryStatus != nil || !silo.BlindSpot.Initialized {
		return
	}
	blind := &silo.BlindSpot

	// (0) 候选与因子 —— 零 RNG。
	in := blindSpotReadInputs(silo)
	motCore := blindSpotMotCore(in.theta) // M(Θ)·S_low·S_high

	methodReads := make([]blindSpotFactorReads, len(blindSpotMethods))
	methodWeights := make([]float64, len(blindSpotMethods))
	sumMethods := 0.0
	for i, m := range blindSpotMethods { // 谱系表 ID 升序 (固定累加序)
		methodReads[i] = blindSpotMethodReads(in, m, motCore)
		methodWeights[i] = blindSpotFloorWeight(m.baseWeight, methodReads[i])
		sumMethods += methodWeights[i]
	}
	idleReads := blindSpotIdleReads(in, motCore)
	wIdle := blindSpotFloorWeight(BlindSpotIdleWeight, idleReads)
	total := wIdle + sumMethods // ε 地板保证 total > 0 恒成立

	// (1) 第一抽：动手与否 —— RNG ×1。
	acted := false
	chosenIdx := -1
	ell := 0
	capS := 0.0
	selChannel := -1
	delta := 0.0
	if DrawFloat64()*total < wIdle {
		// 不动手：本月结束 (只耗 1 次 RNG)。
		blind.ActiveDisguise = BlindSpotIdleID
	} else {
		acted = true
		// (2) 第二抽：手段类 —— RNG ×1，谱系表 ID 升序累减。
		u2 := DrawFloat64() * sumMethods
		acc := 0.0
		chosenIdx = len(blindSpotMethods) - 1 // 浮点边界兜底：取升序最后一个
		for i := range blindSpotMethods {
			acc += methodWeights[i]
			if u2 < acc {
				chosenIdx = i
				break
			}
		}
		m := blindSpotMethods[chosenIdx]
		cost := ItCostCurve(m.baseCost, in.w) // 既有成本曲线：W 经此进入

		// (3) 第三抽：力度 —— RNG ×1，ℓ=1→2→3 升序区间。
		capS = blindSpotCapS(in.g, in.rOp)
		w1 := BlindSpotW1Base + BlindSpotW1PSlope*in.pS
		if in.dispIdx == 0 { // 谨慎
			w1 += BlindSpotCautiousW1
		}
		w2 := BlindSpotW2Base
		w3 := (BlindSpotW3Base +
			BlindSpotW3Courage*BlindSpotBoldness[in.dispIdx] +
			BlindSpotW3Margin*clamp01((in.rOp-cost*BlindSpotHeavyCostMult)/BlindSpotRReserve)) *
			(BlindSpotCapBase + BlindSpotCapWeight*capS)
		w1 = math.Max(w1, BlindSpotIntensityFloor)
		w2 = math.Max(w2, BlindSpotIntensityFloor)
		w3 = math.Max(w3, BlindSpotIntensityFloor)
		u3 := DrawFloat64() * (w1 + w2 + w3)
		ell = 1
		if u3 >= w1 {
			ell = 2
		}
		if u3 >= w1+w2 {
			ell = 3
		}

		// 预算衔接 (23-03 §3.1-5)：不剔除、不重抽——成本超余额时按力度自动降档，
		// 连轻档也付不起时保持 ℓ=1、R_op 扣到 0。
		for ell > 1 && cost*BlindSpotMCost[ell-1] > in.rOp {
			ell--
		}
		spend := cost * BlindSpotMCost[ell-1]
		if spend > in.rOp {
			spend = in.rOp
		}
		silo.ItAuthority.ROp = clamp(silo.ItAuthority.ROp-spend, 0, rOpMax)

		// 选中写入：ActiveDisguise、Coverage 通道月度增量 (τ_cov 动力学)、
		// 痕迹条目 (04 卷 §1.4)。
		blind.ActiveDisguise = m.id
		selChannel = blindSpotChannelIndex(m.channel)
		// ΔB ∝ δ_cov · m_eff(ℓ) · (0.5 + 0.5·cap_s)：能力封顶，能力归零仍留一半
		// 「蛮干」效果 (23-03 §6.2)；单通道单月增量受 δ_cap 截断 (23-04 §1.2)。
		delta = BlindSpotDeltaCov * BlindSpotMEff[ell-1] *
			(BlindSpotCapAbility + BlindSpotCapAbility*capS)
		if delta > BlindSpotDeltaCap {
			delta = BlindSpotDeltaCap
		}
		blind.Traces = append(blind.Traces, model.BlindSpotTraceEntry{
			Year:      silo.CurrentYear,
			Month:     silo.CurrentMonth,
			Channel:   m.channel,
			Kind:      m.traceKind,
			Magnitude: BlindSpotTraceMagBase * BlindSpotMExpose[ell-1],
			Seq:       blind.TraceSeq,
		})
		blind.TraceSeq++
	}

	// 04 卷累积推进 (零 RNG)：未动手月也有自然衰减 (τ_cov)，动手月叠加入选通道
	// 增量；Bias 由 Cov 确定性派生 (23-04 §1.3)。
	blindSpotAdvanceCoverage(blind, selChannel, delta)
	// 痕迹保留期裁剪 (T_retain，到期条目按月确定性移除)。
	blindSpotTrimTraces(blind, silo.CurrentYear, silo.CurrentMonth)

	// 决策日志 (环形 24，因子摘要可归因)。
	blindSpotAppendLog(blind, silo, in, motCore, acted, chosenIdx, ell,
		wIdle, total, methodWeights, methodReads, idleReads)
}

// blindSpotReadInputs 读取因子输入 (全部世界事实，零 RNG)。
func blindSpotReadInputs(silo *model.Silo) blindSpotInputs {
	in := blindSpotInputs{}
	// f_mot 输入：Θ 世界侧真值 (本堡自己的隐藏量，世界侧读合法)。
	in.theta = silo.BlackBox.Theta
	// f_G 输入：主任认知 G 读 silo.Protocol.G (既有字段，恒可读；零值 = 低认知
	// 是真实状态而非缺口——G<0.35 只有惯性伪装，22-02 §4.2)。
	in.g = silo.Protocol.G
	// f_situ 输入：R_op 读 4.6 支出后余额、W 读威势。
	in.rOp = silo.ItAuthority.ROp
	in.w = silo.Authority.W
	// f_risk 输入：r_eff = 上月指派、本月生效；零值 → 当前 SafeguardRisk (与 4.55
	// 同口径的连续性处理)。
	in.rEff = silo.BlackBox.LastAssignedRisk
	if in.rEff == 0 {
		in.rEff = silo.SafeguardRisk
	}
	// f_risk 输入：近 6 月稽查事实计数读 SafeguardIncidents["audit_fact"]
	// (W6 缺口通道格写入；缺省 0)。
	in.auditFacts = silo.SafeguardIncidents[BlindSpotAuditFactKey]
	in.pS = BlindSpotRiskWR*in.rEff +
		BlindSpotRiskWI*clamp01(float64(in.auditFacts)/BlindSpotAuditSat)
	// f_attn 输入：通牒事实。代理登记——引擎侧读不到内容事件状态
	// (special:silo1_ultimatum_cross 的 ContentEventState 由服务层 contentStates
	// 持有)，以 BlackBox.ObediencePulse > 0 代理：通牒签发时服务层置
	// ObediencePulse (world.go)，4.55 每月递减，4.62 读到 >0 对应「上月有通牒
	// 到达」，是 K_ult=3 月窗口的保守近似 (窗口偏窄、方向一致)。
	ultimatum := silo.BlackBox.ObediencePulse > 0
	// 专线流量比无载体 (22-02 G6 专线心跳放弃、G7 注意力预算载体未落地)，取基线
	// 1：流量项 Δ 贡献为 0，注意力变化只经通牒代理投影。
	traffic := 1.0
	in.deltaTraffic = clamp(traffic-1, BlindSpotTrafficLo, BlindSpotTrafficHi)
	if ultimatum {
		in.deltaTraffic += BlindSpotUltShock
	}
	// f_situ 输入：部门恐慌均值 (OrderPressure 恐慌分量同源口径，部门序累加)。
	in.panicMean = blindSpotPanicMean(silo)
	// f_disp：种子派生性情 (零 RNG)。
	in.disp, in.dispIdx = blindSpotDisposition(silo)
	// f_inertia：上月 ActiveDisguise (存档事实)。
	in.prevActive = silo.BlindSpot.ActiveDisguise
	return in
}

// blindSpotMotCore M(Θ)·S_low(Θ)·S_high(Θ)：中段最强、两端受抑 (23-03 §2.3)。
func blindSpotMotCore(theta float64) float64 {
	m := 4 * theta * (1 - theta)
	sLow := clamp01(theta / BlindSpotThetaRamp)
	sHigh := 1 - clamp01((theta-BlindSpotThetaFade)/(1-BlindSpotThetaFade))
	return m * sLow * sHigh
}

// blindSpotFloorWeight 未归一权重 = 基础权重 × 六因子连乘 (§2.2 总表序)，经 ε
// 地板保非零裁量 (23-03 §2.1)。连乘次序固定写死 (浮点纪律)。
func blindSpotFloorWeight(base float64, f blindSpotFactorReads) float64 {
	w := base * f.mot * f.risk * f.attn * f.situ * f.disp * f.cog * f.inertia
	if w < BlindSpotWeightFloor {
		return BlindSpotWeightFloor
	}
	return w
}

// blindSpotMethodReads 手段类候选的六因子读数 (23-03 §2)。
func blindSpotMethodReads(in blindSpotInputs, m blindSpotMethod, motCore float64) blindSpotFactorReads {
	f := blindSpotFactorReads{}
	// f_mot = ε_mot + M·S_low·S_high。
	f.mot = BlindSpotMotEps + motCore
	// f_risk = 1 + a_need·P_s − a_danger·P_s·u_c：紧监督下向低拆穿率通道迁移。
	f.risk = 1 + BlindSpotRiskNeed*in.pS - BlindSpotRiskDanger*in.pS*m.exposeRate
	// f_attn = exp(−ζ·Δ_traffic·u_c)：注意力上抬主要压制高风险动作。
	f.attn = math.Exp(-BlindSpotAttnZeta * in.deltaTraffic * m.exposeRate)
	// f_situ = f_afford · f_panic：W 经 ItCostCurve 进入成本。
	cost := ItCostCurve(m.baseCost, in.w)
	afford := math.Pow(
		clamp01((in.rOp-cost)/BlindSpotRReserve+BlindSpotAffordMargin*(in.rOp/cost)),
		BlindSpotAffordPow)
	f.situ = afford * (1 + BlindSpotPanicK*in.panicMean*(0.5-m.exposeRate))
	// f_disp：性情乘子 (命中行连乘)。
	f.disp = blindSpotDispFactor(in.dispIdx, m)
	// f_G：惯性类 ≡ 1；针对性类 μ(G)。
	f.cog = blindSpotCogFactor(in.g, m.classTag)
	// f_inertia：上月同一选择延续加成。
	f.inertia = 1
	if in.prevActive == m.id {
		f.inertia = BlindSpotInertiaBonus
	}
	return f
}

// blindSpotIdleReads 「不动手」候选的因子读数 (23-03 §2 各节 idle 行)。
func blindSpotIdleReads(in blindSpotInputs, motCore float64) blindSpotFactorReads {
	f := blindSpotFactorReads{}
	// 反向形状：动机越低越倾向不动手。
	f.mot = 1 + (1 - motCore)
	f.risk = 1 // ≡ 1
	// 注意力上抬时略偏向静默。
	f.attn = 1 + BlindSpotIdleAttnSlope*clamp(in.deltaTraffic, 0, BlindSpotTrafficHi)
	// idle 免费：负担项 ≡ 1，恐慌轻微抬升。
	f.situ = 1 + BlindSpotIdlePanicSlope*in.panicMean
	f.disp = BlindSpotDispIdleMul[in.dispIdx]
	f.cog = 1 // ≡ 1
	f.inertia = 1
	if in.prevActive == BlindSpotIdleID {
		f.inertia = BlindSpotInertiaBonus
	}
	return f
}

// blindSpotDispFactor 性情乘子：候选可同时命中多行 (拆穿率档/timing 通道/类标签)，
// 命中行连乘 (23-03 §2.7)。
func blindSpotDispFactor(dispIdx int, m blindSpotMethod) float64 {
	f := 1.0
	if m.exposeRate <= BlindSpotExposeDivide {
		f *= BlindSpotDispLowExpose[dispIdx]
	} else {
		f *= BlindSpotDispHighExpose[dispIdx]
	}
	if m.channel == model.BlindSpotChannelTiming {
		f *= BlindSpotDispTiming[dispIdx]
	}
	if m.classTag == BlindSpotClassInertia {
		f *= BlindSpotDispInertia[dispIdx]
	} else {
		f *= BlindSpotDispTargeted[dispIdx]
	}
	return f
}

// blindSpotCogFactor f_G：惯性类对「流程」演戏 ≡ 1；针对性类按 μ(G) 连续斜坡
// (23-03 §2.8)。
func blindSpotCogFactor(g float64, classTag string) float64 {
	if classTag == BlindSpotClassInertia {
		return 1
	}
	return blindSpotMuG(g)
}

// blindSpotMuG μ(G) 锚点线性插值 (23-03 §2.8)。
func blindSpotMuG(g float64) float64 {
	g = clamp01(g)
	anchors := BlindSpotMuGAnchors
	for i := 0; i+1 < len(anchors); i++ {
		x0, y0 := anchors[i][0], anchors[i][1]
		x1, y1 := anchors[i+1][0], anchors[i+1][1]
		if g <= x1 {
			if x1 == x0 {
				return y1
			}
			return y0 + (g-x0)/(x1-x0)*(y1-y0)
		}
	}
	return anchors[len(anchors)-1][1]
}

// blindSpotCapS 软化能力上限 cap_s = clamp01((G−0.25)/0.15)·clamp01(R_op/100)^0.5
// (22-02 §4.3 的软化版，23-03 §3.2)。
func blindSpotCapS(g, rOp float64) float64 {
	return clamp01((g-BlindSpotCapGOnset)/BlindSpotCapGSpan) *
		math.Pow(clamp01(rOp/rOpMax), 0.5)
}

// blindSpotDisposition 逐堡性情 χ_s：开局由 (RNGSeed, SiloID) 纯函数派生，
// 不耗运行时 RNG、不创建随机源 (23-03 §2.7；同种子同堡序 → 同性情，复现契约)。
func blindSpotDisposition(silo *model.Silo) (string, int) {
	h := blindSpotSplitMix64(uint64(silo.RNGSeed) ^ (uint64(silo.ID) * 0x9E3779B97F4A7C15))
	u := float64(h) / (1 << 64)
	switch {
	case u < 1.0/3.0:
		return BlindSpotDispCautious, 0
	case u < 2.0/3.0:
		return BlindSpotDispSpeculative, 1
	default:
		return BlindSpotDispPlodding, 2
	}
}

// blindSpotSplitMix64 标准 SplitMix64 finalizer (确定性整数哈希)。
func blindSpotSplitMix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

// blindSpotPanicMean 堡内部门恐慌均值 (部门序累加，确定性；无部门时 0)。
func blindSpotPanicMean(silo *model.Silo) float64 {
	if len(silo.Professions) == 0 {
		return 0
	}
	total := 0.0
	for i := range silo.Professions {
		total += silo.Professions[i].PanicValue
	}
	return total / float64(len(silo.Professions))
}

// blindSpotChannelIndex 通道名 → 固定通道序下标 (与存档数组序一致)。
func blindSpotChannelIndex(channel string) int {
	switch channel {
	case model.BlindSpotChannelEquipment:
		return bbChannelEquipment
	case model.BlindSpotChannelSchedule:
		return bbChannelSchedule
	case model.BlindSpotChannelReports:
		return bbChannelReports
	case model.BlindSpotChannelLedger:
		return bbChannelLedger
	case model.BlindSpotChannelCensus:
		return bbChannelCensus
	case model.BlindSpotChannelTiming:
		return bbChannelTiming
	}
	return -1
}

// blindSpotAdvanceCoverage 04 卷 §1.2 月度递推 (零 RNG)：
//
//	Cov_c(t+1) = clamp01( Cov_c(t) − Cov_c(t)/τ_cov + Δcov_c(t) )
//	Bias_c(t)  = b̄_c · Cov_c(t)^γ_b                        // §1.3
//
// τ_cov=30 是 23 系列独立常数 (制度化存量比戒心耐久，23-00 裁决 5)，与 22 系列
// τ_decay=15 (BlackBoxTauDecay) 刻意不等、不得复用。未动手月 Δ=0：放任衰减。
// 通道序固定 (equipment→…→timing)，禁止 map 序。
func blindSpotAdvanceCoverage(blind *model.BlindSpotCultivationState, selChannel int, delta float64) {
	for c := 0; c < bbChannelCount; c++ {
		next := blind.Coverage[c] - blind.Coverage[c]/BlindSpotTauCov
		if c == selChannel {
			next += delta
		}
		blind.Coverage[c] = clamp01(next)
		blind.Bias[c] = clamp01(BlindSpotBiasMax * math.Pow(blind.Coverage[c], BlindSpotBiasGamma))
	}
}

// blindSpotTrimTraces 痕迹保留期裁剪：到期 (age ≥ T_retain) 条目按月确定性移除；
// 保留期内不衰减——事实就是事实 (23-04 §1.4)。
func blindSpotTrimTraces(blind *model.BlindSpotCultivationState, year, month int) {
	current := year*12 + (month - 1)
	cutoff := current - BlindSpotTraceRetain
	kept := blind.Traces[:0]
	for _, tr := range blind.Traces {
		if tr.Year*12+(tr.Month-1) > cutoff {
			kept = append(kept, tr)
		}
	}
	blind.Traces = kept
}

// blindSpotAppendLog 决策日志：归一化权重 (round3，谈判层同口径) + 因子读数摘要；
// 环形上限 24 (NegotiationLogEntry 模式)。世界侧审计日志，不进玩家界面。
func blindSpotAppendLog(blind *model.BlindSpotCultivationState, silo *model.Silo,
	in blindSpotInputs, motCore float64, acted bool, chosenIdx, ell int,
	wIdle, total float64, methodWeights []float64,
	methodReads []blindSpotFactorReads, idleReads blindSpotFactorReads) {

	var weights strings.Builder
	fmt.Fprintf(&weights, "%s=%.3f", BlindSpotIdleID, round3(wIdle/total))
	for i, m := range blindSpotMethods { // ID 升序 (固定序)
		fmt.Fprintf(&weights, "|%s=%.3f", m.id, round3(methodWeights[i]/total))
	}

	choice := BlindSpotIdleID
	channel := ""
	selReads := idleReads
	if acted {
		choice = blindSpotMethods[chosenIdx].id
		channel = blindSpotMethods[chosenIdx].channel
		selReads = methodReads[chosenIdx]
	}
	factors := fmt.Sprintf(
		"Θ=%.3f M·S=%.4f G=%.3f r_eff=%.3f I_recent=%d P_s=%.3f Δtraffic=%.3f "+
			"R_op=%.1f W=%.1f panic=%.3f χ=%s | %s: mot=%.3f risk=%.3f attn=%.3f "+
			"situ=%.3f disp=%.3f G=%.3f inertia=%.3f",
		in.theta, motCore, in.g, in.rEff, in.auditFacts, in.pS, in.deltaTraffic,
		in.rOp, in.w, in.panicMean, in.disp,
		choice, selReads.mot, selReads.risk, selReads.attn, selReads.situ,
		selReads.disp, selReads.cog, selReads.inertia)

	blind.Log = append(blind.Log, model.BlindSpotDecisionLogEntry{
		Year:      silo.CurrentYear,
		Month:     silo.CurrentMonth,
		Acted:     acted,
		Choice:    choice,
		Channel:   channel,
		Intensity: ell,
		Weights:   weights.String(),
		Factors:   factors,
	})
	if len(blind.Log) > BlindSpotLogCap {
		blind.Log = blind.Log[len(blind.Log)-BlindSpotLogCap:]
	}
}

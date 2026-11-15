package engine

import (
	"encoding/json"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"silo40/internal/model"
)

// bsTestSilo 构造启用盲区经营的最小堡 (黑箱/IT/部门状态齐备)。
func bsTestSilo(id uint, seed int64) *model.Silo {
	silo := &model.Silo{
		ID:            id,
		RNGSeed:       seed,
		CurrentYear:   122,
		CurrentMonth:  1,
		SafeguardRisk: 0.4,
		Professions: []model.Profession{
			{Name: "IT", PanicValue: 0.25},
			{Name: "Mechanical", PanicValue: 0.15},
		},
	}
	silo.BlackBox.Initialized = true
	silo.BlackBox.Theta = 0.45
	silo.BlackBox.LastAssignedRisk = 0.5
	silo.Protocol.G = 0.5
	silo.ItAuthority.Initialized = true
	silo.ItAuthority.ROp = 60
	silo.Authority.W = 68
	silo.BlindSpot.Initialized = true
	return silo
}

// bsReplayDecision 在参考流上重放三级抽取 (只读堡状态，不消耗引擎 RNG)，
// 返回与实现同口径的预期决策。
func bsReplayDecision(t *testing.T, silo *model.Silo, ref *rand.Rand) (bool, string, int) {
	t.Helper()
	in := blindSpotReadInputs(silo)
	motCore := blindSpotMotCore(in.theta)
	weights := make([]float64, len(blindSpotMethods))
	sumMethods := 0.0
	for i, m := range blindSpotMethods {
		weights[i] = blindSpotFloorWeight(m.baseWeight, blindSpotMethodReads(in, m, motCore))
		sumMethods += weights[i]
	}
	wIdle := blindSpotFloorWeight(BlindSpotIdleWeight, blindSpotIdleReads(in, motCore))
	total := wIdle + sumMethods

	if ref.Float64()*total < wIdle {
		return false, BlindSpotIdleID, 0
	}
	u2 := ref.Float64() * sumMethods
	acc := 0.0
	chosenIdx := len(blindSpotMethods) - 1
	for i := range blindSpotMethods {
		acc += weights[i]
		if u2 < acc {
			chosenIdx = i
			break
		}
	}
	m := blindSpotMethods[chosenIdx]
	cost := ItCostCurve(m.baseCost, in.w)
	capS := blindSpotCapS(in.g, in.rOp)
	w1 := BlindSpotW1Base + BlindSpotW1PSlope*in.pS
	if in.dispIdx == 0 {
		w1 += BlindSpotCautiousW1
	}
	w2 := BlindSpotW2Base
	w3 := (BlindSpotW3Base + BlindSpotW3Courage*BlindSpotBoldness[in.dispIdx] +
		BlindSpotW3Margin*clamp01((in.rOp-cost*BlindSpotHeavyCostMult)/BlindSpotRReserve)) *
		(BlindSpotCapBase + BlindSpotCapWeight*capS)
	w1 = math.Max(w1, BlindSpotIntensityFloor)
	w2 = math.Max(w2, BlindSpotIntensityFloor)
	w3 = math.Max(w3, BlindSpotIntensityFloor)
	u3 := ref.Float64() * (w1 + w2 + w3)
	ell := 1
	if u3 >= w1 {
		ell = 2
	}
	if u3 >= w1+w2 {
		ell = 3
	}
	for ell > 1 && cost*BlindSpotMCost[ell-1] > in.rOp {
		ell--
	}
	return true, m.id, ell
}

// TestBlindSpotRNGConsumptionAndReplay 每堡每月 RNG 消耗 1–3 次，堡内顺序写死
// (动手 → 手段 → 力度)：参考流重放逐位一致，消耗后的下一抽对齐参考流。
func TestBlindSpotRNGConsumptionAndReplay(t *testing.T) {
	const seed = int64(2301)
	for trial := 0; trial < 8; trial++ {
		silo := bsTestSilo(2, seed)
		silo.CurrentMonth = trial%12 + 1
		silo.BlackBox.Theta = 0.1 + 0.11*float64(trial) // 不同处境覆盖动手/不动手

		ref := rand.New(rand.NewSource(seed))
		wantActed, wantChoice, wantEll := bsReplayDecision(t, silo, ref)

		SetRNG(rand.New(rand.NewSource(seed)))
		RunBlindSpotCultivationMonthly(silo)

		if len(silo.BlindSpot.Log) != 1 {
			t.Fatalf("trial %d: 日志条数 = %d，想要 1", trial, len(silo.BlindSpot.Log))
		}
		entry := silo.BlindSpot.Log[0]
		if entry.Acted != wantActed || entry.Choice != wantChoice || entry.Intensity != wantEll {
			t.Fatalf("trial %d: 决策 (%v,%s,%d) != 参考流重放 (%v,%s,%d)",
				trial, entry.Acted, entry.Choice, entry.Intensity, wantActed, wantChoice, wantEll)
		}
		if entry.Acted != (silo.BlindSpot.ActiveDisguise != BlindSpotIdleID) {
			t.Fatalf("trial %d: ActiveDisguise %q 与 Acted=%v 不一致",
				trial, silo.BlindSpot.ActiveDisguise, entry.Acted)
		}

		// 消耗次数 = 动手 3 次 / 不动手 1 次：重放已把 ref 推到决策消耗位，
		// 下一抽必须与引擎流的下一抽逐位一致。
		consumed := 1
		if wantActed {
			consumed = 3
		}
		if got, want := DrawFloat64(), ref.Float64(); got != want {
			t.Fatalf("trial %d: RNG 消耗 %d 次后下一抽 %v != 参考 %v", trial, consumed, got, want)
		}
	}
}

// TestBlindSpotSiloOrderSharedStream 堡序纪律：多堡按 SiloID 升序依次结算时，
// 各堡消耗在同一 RNG 流上连续、顺序固定——逐堡重放与逐位对齐均成立。
func TestBlindSpotSiloOrderSharedStream(t *testing.T) {
	const seed = int64(2302)
	silos := []*model.Silo{bsTestSilo(2, seed), bsTestSilo(3, seed+1), bsTestSilo(7, seed+2)}
	silos[1].BlackBox.Theta = 0.8
	silos[2].BlackBox.Theta = 0.1

	ref := rand.New(rand.NewSource(seed))
	type want struct {
		acted  bool
		choice string
		ell    int
	}
	wants := make([]want, len(silos))
	totalConsumed := 0
	for i, silo := range silos { // 升序重放
		a, c, e := bsReplayDecision(t, silo, ref)
		wants[i] = want{a, c, e}
		if a {
			totalConsumed += 3
		} else {
			totalConsumed++
		}
	}

	SetRNG(rand.New(rand.NewSource(seed)))
	for _, silo := range silos { // 升序结算 (调用方纪律)
		RunBlindSpotCultivationMonthly(silo)
	}
	for i, silo := range silos {
		entry := silo.BlindSpot.Log[0]
		if entry.Acted != wants[i].acted || entry.Choice != wants[i].choice || entry.Intensity != wants[i].ell {
			t.Fatalf("堡 %d: 决策 (%v,%s,%d) != 升序重放 (%v,%s,%d)", silo.ID,
				entry.Acted, entry.Choice, entry.Intensity, wants[i].acted, wants[i].choice, wants[i].ell)
		}
	}
	ref2 := rand.New(rand.NewSource(seed))
	for i := 0; i < totalConsumed; i++ {
		ref2.Float64()
	}
	if got, want := DrawFloat64(), ref2.Float64(); got != want {
		t.Fatalf("三堡升序共消耗 %d 次后下一抽 %v != 参考 %v", totalConsumed, got, want)
	}
}

// TestBlindSpotWeightFloorExtremeSituations ε 地板：任何处境下所有候选权重
// ≥ ε_w (23-03 §2.9 非零裁量)——构造极端处境验证。
func TestBlindSpotWeightFloorExtremeSituations(t *testing.T) {
	extremes := []func() *model.Silo{
		// 忠顺 + 枯竭 + 满监督 + 通牒脉冲 + 满恐慌。
		func() *model.Silo {
			s := bsTestSilo(2, 1)
			s.BlackBox.Theta = 0
			s.BlackBox.LastAssignedRisk = 1.0
			s.SafeguardIncidents = map[string]int{BlindSpotAuditFactKey: 100}
			s.BlackBox.ObediencePulse = 2
			s.ItAuthority.ROp = 0
			s.Authority.W = 0
			s.Protocol.G = 0
			for i := range s.Professions {
				s.Professions[i].PanicValue = 1
			}
			return s
		},
		// 离心 + 富裕 + 无监督 + 高认知。
		func() *model.Silo {
			s := bsTestSilo(3, 2)
			s.BlackBox.Theta = 1
			s.BlackBox.LastAssignedRisk = 0
			s.SafeguardRisk = 0
			s.ItAuthority.ROp = rOpMax
			s.Authority.W = 100
			s.Protocol.G = 1
			return s
		},
		// 中段 + 惯性延续。
		func() *model.Silo {
			s := bsTestSilo(4, 3)
			s.BlackBox.Theta = 0.5
			s.BlindSpot.ActiveDisguise = "reports_trim"
			return s
		},
	}
	for i, mk := range extremes {
		silo := mk()
		in := blindSpotReadInputs(silo)
		motCore := blindSpotMotCore(in.theta)
		for _, m := range blindSpotMethods {
			w := blindSpotFloorWeight(m.baseWeight, blindSpotMethodReads(in, m, motCore))
			if w < BlindSpotWeightFloor {
				t.Fatalf("极端处境 %d 手段 %s 权重 %v < ε_w %v", i, m.id, w, BlindSpotWeightFloor)
			}
			if math.IsNaN(w) || math.IsInf(w, 0) {
				t.Fatalf("极端处境 %d 手段 %s 权重非有限值 %v", i, m.id, w)
			}
		}
		wIdle := blindSpotFloorWeight(BlindSpotIdleWeight, blindSpotIdleReads(in, motCore))
		if wIdle < BlindSpotWeightFloor {
			t.Fatalf("极端处境 %d idle 权重 %v < ε_w %v", i, wIdle, BlindSpotWeightFloor)
		}
	}
}

// TestBlindSpotGatingZeroRNGZeroState 门控关闭 (未启用/已终局)：状态零变化、零 RNG。
func TestBlindSpotGatingZeroRNGZeroState(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*model.Silo)
	}{
		{"未启用", func(s *model.Silo) { s.BlindSpot.Initialized = false }},
		{"已终局", func(s *model.Silo) { s.VictoryStatus = &model.VictoryStatus{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const seed = int64(2303)
			silo := bsTestSilo(2, seed)
			tc.mut(silo)
			before := silo.BlindSpot

			SetRNG(rand.New(rand.NewSource(seed)))
			RunBlindSpotCultivationMonthly(silo)

			ref := rand.New(rand.NewSource(seed))
			if got, want := DrawFloat64(), ref.Float64(); got != want {
				t.Fatalf("门控关闭必须零 RNG：下一抽 %v != 参考首抽 %v", got, want)
			}
			if !reflect.DeepEqual(silo.BlindSpot, before) {
				t.Fatalf("门控关闭必须状态零变化：got %+v want %+v", silo.BlindSpot, before)
			}
		})
	}
	// nil 堡不 panic。
	RunBlindSpotCultivationMonthly(nil)
}

// TestBlindSpotTauCovDistinctAndDirections τ_cov 与 22 系列 τ_decay 是两个独立
// 常量 (23-00 裁决 5)；Cov 动手月增长、未动手月按 τ_cov 松弛、Bias 随 Cov 派生。
func TestBlindSpotTauCovDistinctAndDirections(t *testing.T) {
	if BlindSpotTauCov == BlackBoxTauDecay {
		t.Fatalf("τ_cov (%v) 不得等于 22 系列 τ_decay (%v)：两个独立时间常数",
			BlindSpotTauCov, BlackBoxTauDecay)
	}
	if BlindSpotTauCov <= BlackBoxTauDecay {
		t.Fatalf("τ_cov (%v) 必须长于 τ_decay (%v)：制度化存量比戒心耐久",
			BlindSpotTauCov, BlackBoxTauDecay)
	}

	// 增长方向：动手月入选通道 +Δ (零存量起)。
	var blind model.BlindSpotCultivationState
	blindSpotAdvanceCoverage(&blind, bbChannelReports, 0.02)
	if blind.Coverage[bbChannelReports] <= 0 {
		t.Fatalf("动手月 reports 通道必须增长：%v", blind.Coverage[bbChannelReports])
	}
	if got, want := blind.Coverage[bbChannelReports], 0.02; math.Abs(got-want) > 1e-12 {
		t.Fatalf("零存量起首月增量 = %v，想要 %v", got, want)
	}
	for c := 0; c < bbChannelCount; c++ {
		if c == bbChannelReports {
			continue
		}
		if blind.Coverage[c] != 0 {
			t.Fatalf("未入选通道 %d 不得增长：%v", c, blind.Coverage[c])
		}
	}
	// Bias = b̄_c·Cov^γ_b 同步派生。
	if want := BlindSpotBiasMax * math.Pow(0.02, BlindSpotBiasGamma); math.Abs(blind.Bias[bbChannelReports]-want) > 1e-12 {
		t.Fatalf("Bias = %v，想要 %v", blind.Bias[bbChannelReports], want)
	}

	// 松弛方向：未动手月按 τ_cov 衰减 (Δ=0)。
	blind.Coverage[bbChannelReports] = 0.5
	blindSpotAdvanceCoverage(&blind, -1, 0)
	if got, want := blind.Coverage[bbChannelReports], 0.5-0.5/BlindSpotTauCov; math.Abs(got-want) > 1e-12 {
		t.Fatalf("松弛后 = %v，想要 %v", got, want)
	}
	if blind.Coverage[bbChannelReports] >= 0.5 {
		t.Fatalf("未动手月必须松弛：%v", blind.Coverage[bbChannelReports])
	}

	// 持续经营趋向平衡厚度 Δ·τ_cov (23-04 §1.2 推导：0.02×30=0.6)。
	var eq model.BlindSpotCultivationState
	for i := 0; i < 300; i++ {
		blindSpotAdvanceCoverage(&eq, bbChannelEquipment, BlindSpotDeltaCov)
	}
	equilibrium := BlindSpotDeltaCov * BlindSpotTauCov
	if math.Abs(eq.Coverage[bbChannelEquipment]-equilibrium) > 0.01 {
		t.Fatalf("300 月持续经营 Cov = %v，想要趋向 %v", eq.Coverage[bbChannelEquipment], equilibrium)
	}
}

// TestBlindSpotTraceRetain 痕迹账本：动手月记一条 (kind 按 04 §1.4 表)，
// T_retain=36 月外的条目确定性移除。
func TestBlindSpotTraceRetain(t *testing.T) {
	var blind model.BlindSpotCultivationState
	// 40 个月前的痕迹 + 本月新痕：裁剪后只留 36 月内。
	blind.Traces = []model.BlindSpotTraceEntry{
		{Year: 118, Month: 10, Channel: model.BlindSpotChannelReports, Kind: "standard_change", Seq: 0},
		{Year: 122, Month: 1, Channel: model.BlindSpotChannelLedger, Kind: "receipt_shape", Seq: 1},
	}
	blind.TraceSeq = 2
	blindSpotTrimTraces(&blind, 122, 2) // 当前 122-02：118-10 已超 36 月
	if len(blind.Traces) != 1 || blind.Traces[0].Seq != 1 {
		t.Fatalf("裁剪后应只剩 1 条 (Seq=1)：got %+v", blind.Traces)
	}
	// 恰好 36 月龄的条目被移除 (age ≥ T_retain)，35 月龄保留。
	blind.Traces = []model.BlindSpotTraceEntry{
		{Year: 119, Month: 2, Seq: 2}, // age = 36
		{Year: 119, Month: 3, Seq: 3}, // age = 35
	}
	blindSpotTrimTraces(&blind, 122, 2)
	if len(blind.Traces) != 1 || blind.Traces[0].Seq != 3 {
		t.Fatalf("36 月龄应移除、35 月龄应保留：got %+v", blind.Traces)
	}
}

// TestBlindSpotLogRingAndFields 决策日志：字段齐全、环形上限 24。
func TestBlindSpotLogRingAndFields(t *testing.T) {
	const seed = int64(2304)
	silo := bsTestSilo(2, seed)
	SetRNG(rand.New(rand.NewSource(seed)))

	months := 0
	for y := 122; y <= 124; y++ {
		for m := 1; m <= 12 && months < 30; m, months = m+1, months+1 {
			silo.CurrentYear = y
			silo.CurrentMonth = m
			RunBlindSpotCultivationMonthly(silo)
		}
	}

	log := silo.BlindSpot.Log
	if len(log) != BlindSpotLogCap {
		t.Fatalf("30 月后日志长度 = %d，想要环形上限 %d", len(log), BlindSpotLogCap)
	}
	// 前 6 条被挤出：最旧条目是第 7 次决策 (122 年 7 月)。
	if log[0].Year != 122 || log[0].Month != 7 {
		t.Fatalf("环形最旧条目 = %d-%d，想要 122-7", log[0].Year, log[0].Month)
	}
	for i, entry := range log {
		if entry.Year == 0 || entry.Month == 0 {
			t.Fatalf("条目 %d 缺年/月：%+v", i, entry)
		}
		if entry.Weights == "" || entry.Factors == "" {
			t.Fatalf("条目 %d 缺权重表或因子摘要：%+v", i, entry)
		}
		if entry.Choice == "" {
			t.Fatalf("条目 %d 缺选中候选：%+v", i, entry)
		}
		if entry.Acted {
			if entry.Intensity < 1 || entry.Intensity > 3 {
				t.Fatalf("条目 %d 动手时力度 %d 越出 1–3", i, entry.Intensity)
			}
			if entry.Channel == "" || entry.Choice == BlindSpotIdleID {
				t.Fatalf("条目 %d 动手时缺通道/选择非法：%+v", i, entry)
			}
		} else {
			if entry.Choice != BlindSpotIdleID || entry.Channel != "" || entry.Intensity != 0 {
				t.Fatalf("条目 %d 不动手时应为 idle/空通道/力度 0：%+v", i, entry)
			}
		}
	}
}

// TestBlindSpotSameSeedByteIdentical 同种子双跑逐字节一致 (复现契约)。
func TestBlindSpotSameSeedByteIdentical(t *testing.T) {
	const seed = int64(2305)
	run := func() []byte {
		silo := bsTestSilo(5, seed)
		SetRNG(rand.New(rand.NewSource(seed)))
		for i := 0; i < 12; i++ {
			RunBlindSpotCultivationMonthly(silo)
			silo.CurrentMonth++
			if silo.CurrentMonth > 12 {
				silo.CurrentMonth = 1
				silo.CurrentYear++
			}
		}
		out, err := json.Marshal(struct {
			BlindSpot   model.BlindSpotCultivationState `json:"blind_spot"`
			ROp         float64                         `json:"r_op"`
			SafeguardRI map[string]int                  `json:"safeguard_incidents"`
		}{silo.BlindSpot, silo.ItAuthority.ROp, silo.SafeguardIncidents})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return out
	}
	if a, b := run(), run(); string(a) != string(b) {
		t.Fatalf("同种子双跑不逐字节一致：\n%s\n%s", a, b)
	}
}

// TestBlindSpotDispositionDeterministic 性情派生：种子纯函数、零 RNG、同种子同堡
// 必同值 (23-03 §2.7)。
func TestBlindSpotDispositionDeterministic(t *testing.T) {
	const seed = int64(2306)
	silo := bsTestSilo(9, seed)

	SetRNG(rand.New(rand.NewSource(seed)))
	d1, idx1 := blindSpotDisposition(silo)
	// 派生零 RNG：下一抽必须是参考流首抽。
	ref := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("性情派生必须零 RNG：下一抽 %v != 参考首抽 %v", got, want)
	}
	d2, idx2 := blindSpotDisposition(silo)
	if d1 != d2 || idx1 != idx2 {
		t.Fatalf("同种子同堡性情不一致：(%s,%d) vs (%s,%d)", d1, idx1, d2, idx2)
	}
	// 三型覆盖：不同 SiloID 派生结果都落在合法三型内。
	valid := map[string]bool{BlindSpotDispCautious: true, BlindSpotDispSpeculative: true, BlindSpotDispPlodding: true}
	seen := map[string]bool{}
	for id := uint(1); id <= 60; id++ {
		s := bsTestSilo(id, seed)
		d, idx := blindSpotDisposition(s)
		if !valid[d] || idx < 0 || idx > 2 {
			t.Fatalf("堡 %d 派生非法性情 (%s,%d)", id, d, idx)
		}
		seen[d] = true
	}
	if len(seen) < 2 {
		t.Fatalf("60 个堡只派生出 %v 一种性情，SplitMix64 分桶可疑", seen)
	}
}

// TestBlindSpotBudgetDocking 预算衔接 (23-03 §3.1-5)：枯竭堡长期运行 R_op 不为负、
// 决策照常产生 (非零裁量：付不起也不禁选，只降档/扣到 0)。
func TestBlindSpotBudgetDocking(t *testing.T) {
	const seed = int64(2307)
	silo := bsTestSilo(2, seed)
	silo.ItAuthority.ROp = 3
	silo.Authority.W = 100 // 成本曲线最低点
	SetRNG(rand.New(rand.NewSource(seed)))
	for i := 0; i < 24; i++ {
		RunBlindSpotCultivationMonthly(silo)
		if silo.ItAuthority.ROp < 0 || silo.ItAuthority.ROp > rOpMax {
			t.Fatalf("第 %d 月后 R_op = %v 越出 [0,%d]", i, silo.ItAuthority.ROp, rOpMax)
		}
		if len(silo.BlindSpot.Log) == 0 {
			t.Fatalf("第 %d 月无决策日志", i)
		}
	}
}

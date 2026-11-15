package engine

import (
	"math"
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// bbTestSilo 构造启用黑箱的最小堡：含 IT 部门 (line_tone 代理) 与恐慌分布。
func bbTestSilo(theta float64) *model.Silo {
	silo := &model.Silo{
		CurrentYear:  122,
		CurrentMonth: 3,
		Rebellion:    0.2,
		Professions: []model.Profession{
			{Name: "IT", PanicValue: 0.25},
			{Name: "Mechanical", PanicValue: 0.15},
		},
	}
	silo.BlackBox.Initialized = true
	silo.BlackBox.Theta = theta
	return silo
}

// TestGenerateBlackBoxObservationsSequence 锁定 4.65 的逐位结构：信号类型升序
// [receipt_delay, execution_gap, line_tone]，每信号恰好 3 次 DrawFloat64
// (1 伪装判定 + 2 Box-Muller 噪声)，判定先于噪声；观测值与参考流逐位一致。
func TestGenerateBlackBoxObservationsSequence(t *testing.T) {
	const seed = int64(2026)
	silo := bbTestSilo(0.55)

	SetRNG(rand.New(rand.NewSource(seed)))
	GenerateBlackBoxObservations(silo)

	// 消耗总数锁定为 9 (3 信号 × 3 次)：下一次取样必须等于参考流第 10 抽。
	ref := rand.New(rand.NewSource(seed))
	for i := 0; i < len(bbSignalSpecs)*3; i++ {
		ref.Float64()
	}
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("RNG 消耗次数不是每信号 3 次：下一抽 %v != 参考第 10 抽 %v", got, want)
	}

	// 逐信号重放参考流，校验值/顺序/延迟入队字段。
	ref = rand.New(rand.NewSource(seed))
	wantOrder := []string{
		model.BlackBoxSignalReceiptDelay,
		model.BlackBoxSignalExecutionGap,
		model.BlackBoxSignalLineTone,
	}
	if len(silo.BlackBox.PendingObservations) != len(wantOrder) {
		t.Fatalf("观测入队数量 = %d，想要 %d", len(silo.BlackBox.PendingObservations), len(wantOrder))
	}
	for i, obs := range silo.BlackBox.PendingObservations {
		spec := bbSignalSpecs[i]
		if obs.SignalType != wantOrder[i] {
			t.Fatalf("信号 %d 类型 %s 不在升序位 %s", i, obs.SignalType, wantOrder[i])
		}
		if !blackBoxSignalProduced(silo, spec.signalType) {
			t.Fatalf("信号 %s 当月应产生", spec.signalType)
		}
		pDisg, dEff, sigmaEff := blackboxInjectionParams(silo, spec, silo.BlackBox.Theta)
		disguised := ref.Float64() < pDisg
		bias := 0.0
		if disguised {
			bias = -dEff
		}
		eta := boxMuller(ref.Float64(), ref.Float64())
		want := clamp01(silo.BlackBox.Theta + bias + sigmaEff*eta)
		if obs.Value != want {
			t.Fatalf("信号 %s 观测值 %v != 参考重放 %v", spec.signalType, obs.Value, want)
		}
		if obs.Seq != i {
			t.Fatalf("信号 %s Seq = %d，想要 %d", spec.signalType, obs.Seq, i)
		}
		arrive := AdvanceTime(GameTime{Year: 122, Month: 3}, spec.delayMonths)
		if obs.ArriveYear != arrive.Year || obs.ArriveMonth != arrive.Month {
			t.Fatalf("信号 %s 到达月 %d-%d != 当月+δ %d-%d", spec.signalType,
				obs.ArriveYear, obs.ArriveMonth, arrive.Year, arrive.Month)
		}
		if obs.ReflectedYear != 122 || obs.ReflectedMonth != 3 {
			t.Fatalf("信号 %s 反映月 %d-%d != 当月", spec.signalType, obs.ReflectedYear, obs.ReflectedMonth)
		}
	}
	if silo.BlackBox.ObsSeq != len(wantOrder) {
		t.Fatalf("ObsSeq = %d，想要 %d", silo.BlackBox.ObsSeq, len(wantOrder))
	}
}

// TestGenerateBlackBoxObservationsGatingZeroRNG 门控关闭：零 RNG、
// PendingObservations 不增长、ObsSeq 不变。
func TestGenerateBlackBoxObservationsGatingZeroRNG(t *testing.T) {
	const seed = int64(11)
	silo := bbTestSilo(0.5)
	silo.BlackBox.Initialized = false // 功能未启用 (旧档语义)

	SetRNG(rand.New(rand.NewSource(seed)))
	GenerateBlackBoxObservations(silo)

	ref := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("门控关闭必须零 RNG：下一抽 %v != 参考首抽 %v", got, want)
	}
	if len(silo.BlackBox.PendingObservations) != 0 {
		t.Fatalf("门控关闭时 PendingObservations 不得增长：%+v", silo.BlackBox.PendingObservations)
	}
	if silo.BlackBox.ObsSeq != 0 {
		t.Fatalf("门控关闭时 ObsSeq 不得变化：%d", silo.BlackBox.ObsSeq)
	}
}

// TestInjectionIdentityWhenBlindSpotZero 注入恒等：BlindSpot 未启用 (旧档零值)
// 或已启用但全零时，三参数 == 22 基线。
func TestInjectionIdentityWhenBlindSpotZero(t *testing.T) {
	const theta = 0.62
	wantP := blackBoxDisguiseProb(theta)
	for _, spec := range bbSignalSpecs {
		// 未启用 (含旧档)。
		siloOff := bbTestSilo(theta)
		siloOff.BlindSpot = model.BlindSpotCultivationState{}
		p, d, s := blackboxInjectionParams(siloOff, spec, theta)
		if p != wantP || d != spec.disguise || s != spec.sigma {
			t.Fatalf("信号 %s 未启用注入必须恒等：got (%v,%v,%v) want (%v,%v,%v)",
				spec.signalType, p, d, s, wantP, spec.disguise, spec.sigma)
		}
		// 已启用但全零。
		siloZero := bbTestSilo(theta)
		siloZero.BlindSpot.Initialized = true
		p, d, s = blackboxInjectionParams(siloZero, spec, theta)
		if p != wantP || d != spec.disguise || s != spec.sigma {
			t.Fatalf("信号 %s 全零注入必须恒等：got (%v,%v,%v) want (%v,%v,%v)",
				spec.signalType, p, d, s, wantP, spec.disguise, spec.sigma)
		}
	}
}

// TestInjectionNonZeroAmplifiesWithinCaps 注入非零：p_disg_eff ≥ p_disg、
// d_eff ≥ d、σ_eff ≥ σ，且各自不超过上限 (23-05 §3.1)。
func TestInjectionNonZeroAmplifiesWithinCaps(t *testing.T) {
	const theta = 0.62
	const eps = 1e-12
	for _, spec := range bbSignalSpecs {
		silo := bbTestSilo(theta)
		silo.BlindSpot.Initialized = true
		for c := 0; c < bbChannelCount; c++ {
			silo.BlindSpot.Coverage[c] = 0.8
			silo.BlindSpot.Bias[c] = 0.7
		}
		p, d, s := blackboxInjectionParams(silo, spec, theta)
		p0 := blackBoxDisguiseProb(theta)
		if p < p0-eps || p > 1+eps {
			t.Fatalf("信号 %s p_disg_eff = %v 越出 [%v, 1]", spec.signalType, p, p0)
		}
		if d < spec.disguise-eps || d > spec.dCap+eps {
			t.Fatalf("信号 %s d_eff = %v 越出 [%v, %v]", spec.signalType, d, spec.disguise, spec.dCap)
		}
		sigmaMax := spec.sigma * (1 + bbInjectSMax)
		if s < spec.sigma-eps || s > sigmaMax+eps {
			t.Fatalf("信号 %s σ_eff = %v 越出 [%v, %v]", spec.signalType, s, spec.sigma, sigmaMax)
		}
	}
}

// TestObservationBaseIsTheta 22-01 §7.1 权威形式：观测基值是结算后的 Θ'，
// 「测到的是服从不是自保」由伪装混合与噪声产生 (22-02 §5)，不改基值。
// 三信号均每月产生。
func TestObservationBaseIsTheta(t *testing.T) {
	for _, signal := range []string{
		model.BlackBoxSignalReceiptDelay,
		model.BlackBoxSignalExecutionGap,
		model.BlackBoxSignalLineTone,
	} {
		if !blackBoxSignalProduced(bbTestSilo(0.4), signal) {
			t.Fatalf("信号 %s 应每月产生", signal)
		}
	}

	// 同种子下 Θ 高的堡观测均值必须显著高于 Θ 低的堡 (基值 = Θ' 的直接证据)。
	mean := func(theta float64) float64 {
		total := 0.0
		n := 0
		for i := 0; i < 60; i++ {
			silo := bbTestSilo(theta)
			GenerateBlackBoxObservations(silo)
			for _, obs := range silo.BlackBox.PendingObservations {
				total += obs.Value
				n++
			}
		}
		return total / float64(n)
	}
	hi, lo := mean(0.8), mean(0.2)
	if hi-lo < 0.3 {
		t.Fatalf("观测均值差 %v 过小 (高 Θ %v / 低 Θ %v)：基值未跟随 Θ'", hi-lo, hi, lo)
	}
}

// TestBoxMullerDeterministic Box-Muller 变换的确定性与边界 (u1=0 不产生 NaN)。
func TestBoxMullerDeterministic(t *testing.T) {
	got := boxMuller(0.5, 0.5) // sqrt(-2·ln(0.5))·cos(π)
	want := -math.Sqrt(2 * math.Ln2)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("boxMuller(0.5,0.5) = %v，想要 %v", got, want)
	}
	if v := boxMuller(0, 0.25); math.IsNaN(v) || math.IsInf(v, 0) {
		t.Fatalf("boxMuller(0,·) 产生非有限值 %v", v)
	}
}

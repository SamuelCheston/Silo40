package engine

import (
	"math"
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// blackBoxTestSilo 已启用黑箱的最小测试堡。
func blackBoxTestSilo() *model.Silo {
	silo := &model.Silo{}
	silo.BlackBox.Initialized = true
	return silo
}

// TestBlackBoxThetaFullPressureGrowth 锁定 01 卷 §3.2 硬要求校验：满压
// (x=1) 净增 +g_max/月——1 个月明显抬头，5 个月到危险区 0.60。
func TestBlackBoxThetaFullPressureGrowth(t *testing.T) {
	silo := blackBoxTestSilo()
	silo.BlackBox.LastAssignedRisk = 1.0

	UpdateBlackBoxMonthly(silo)
	if math.Abs(silo.BlackBox.Theta-0.12) > 1e-12 {
		t.Fatalf("full pressure 1 month: Θ = %v, want 0.12", silo.BlackBox.Theta)
	}
	for i := 0; i < 4; i++ {
		UpdateBlackBoxMonthly(silo)
	}
	if math.Abs(silo.BlackBox.Theta-0.60) > 1e-12 {
		t.Fatalf("full pressure 5 months: Θ = %v, want 0.60 (danger zone)", silo.BlackBox.Theta)
	}
}

// TestBlackBoxThetaDecaysAtCalm 锁定 01 卷 §3.2：r ≤ r_calm 时 growth = 0，
// Θ 按 τ=15 月指数衰减 (一年安抚 0.60 → ≈0.27)。
func TestBlackBoxThetaDecaysAtCalm(t *testing.T) {
	silo := blackBoxTestSilo()
	silo.BlackBox.Theta = 0.60
	silo.BlackBox.LastAssignedRisk = BlackBoxRCalm // x = 0：全速衰减

	UpdateBlackBoxMonthly(silo)
	if want := 0.60 * (1 - 1/BlackBoxTauDecay); math.Abs(silo.BlackBox.Theta-want) > 1e-12 {
		t.Fatalf("calm 1 month: Θ = %v, want %v", silo.BlackBox.Theta, want)
	}
	for i := 0; i < 11; i++ {
		UpdateBlackBoxMonthly(silo)
	}
	if want := 0.60 * math.Pow(1-1/BlackBoxTauDecay, 12); math.Abs(silo.BlackBox.Theta-want) > 1e-12 {
		t.Fatalf("calm 12 months: Θ = %v, want %v", silo.BlackBox.Theta, want)
	}
	if silo.BlackBox.Theta > 0.30 {
		t.Fatalf("one year of calm must halve Θ≈0.60: %v", silo.BlackBox.Theta)
	}

	// r=0 (深度回落区) 同样只衰减不增长。
	deep := blackBoxTestSilo()
	deep.BlackBox.Theta = 0.5
	deep.BlackBox.LastAssignedRisk = 0
	UpdateBlackBoxMonthly(deep)
	if want := 0.5 * (1 - 1/BlackBoxTauDecay); math.Abs(deep.BlackBox.Theta-want) > 1e-12 {
		t.Fatalf("r=0 must decay at full speed: Θ = %v, want %v", deep.BlackBox.Theta, want)
	}
}

// TestBlackBoxThetaEquilibriumMatchesFormula 锁定 01 卷 §3.2 平衡点：
// Θ_eq(x) = g_max·τ·x/(1−x) = 1.8x/(1−x)。轻度超压慢性滋生，数值逐位验证。
func TestBlackBoxThetaEquilibriumMatchesFormula(t *testing.T) {
	cases := []struct {
		r    float64
		want float64
	}{
		{0.40, 0.20}, // x = 0.1 → 1.8·0.1/0.9
		{0.45, 0.45}, // x = 0.2 → 1.8·0.2/0.8 (深入阳奉阴违带)
	}
	for _, tc := range cases {
		silo := blackBoxTestSilo()
		silo.BlackBox.LastAssignedRisk = tc.r
		for i := 0; i < 2000; i++ {
			UpdateBlackBoxMonthly(silo)
		}
		if math.Abs(silo.BlackBox.Theta-tc.want) > 1e-9 {
			t.Fatalf("r=%v equilibrium: Θ = %v, want %v", tc.r, silo.BlackBox.Theta, tc.want)
		}
	}
}

// TestBlackBoxExecutionCurveShape 锁定 01 卷 §4.2 数值校验：Θ=0 峰在 0.45
// 高 0.95；Θ=1 峰移到 0.15、高 0.40、宽减半；两端低。
func TestBlackBoxExecutionCurveShape(t *testing.T) {
	// Θ=0：峰位 0.45、峰高 0.95。
	if got := blackBoxExecution(0.45, 0); math.Abs(got-0.95) > 1e-12 {
		t.Fatalf("E(0.45, 0) = %v, want 0.95", got)
	}
	// 两端低：低 r 怠工、高 r 急性失能。
	if got := blackBoxExecution(0, 0); got > 0.10 {
		t.Fatalf("E(0, 0) = %v, want low tail (≈0.08)", got)
	}
	if got := blackBoxExecution(1, 0); got > 0.05 {
		t.Fatalf("E(1, 0) = %v, want low tail (≈0.02)", got)
	}
	// Θ=1：峰移 0.15、高 0.40。
	if got := blackBoxExecution(0.15, 1); math.Abs(got-0.40) > 1e-12 {
		t.Fatalf("E(0.15, 1) = %v, want 0.40", got)
	}
	// 沿用旧甜蜜点指派 → 彻底失能 (01 卷 §4.2 ≈0.004)。
	if got := blackBoxExecution(0.45, 1); got > 0.01 {
		t.Fatalf("E(0.45, 1) = %v, want near-total dysfunction", got)
	}
	// 峰宽：1σ 落点幅值 = 峰高·exp(−1/2)。σ(0)=0.20，σ(1)=0.10 (减半)。
	sigmaDrop := math.Exp(-0.5)
	if got, want := blackBoxExecution(0.45+0.20, 0), 0.95*sigmaDrop; math.Abs(got-want) > 1e-12 {
		t.Fatalf("Θ=0 width: E(r_0+σ_0) = %v, want %v", got, want)
	}
	if got, want := blackBoxExecution(0.15+0.10, 1), 0.40*sigmaDrop; math.Abs(got-want) > 1e-12 {
		t.Fatalf("Θ=1 width: E(r*+σ(1)) = %v, want %v", got, want)
	}
}

// TestBlackBoxGateClosedZeroChange 门控：未启用 (单堡/关闭/旧档) 逐位不变，
// 零消耗直接返回。
func TestBlackBoxGateClosedZeroChange(t *testing.T) {
	UpdateBlackBoxMonthly(nil) // 不得 panic

	silo := &model.Silo{} // 零值 = 未启用
	silo.SafeguardRisk = 0.5
	silo.BlackBox.Theta = 0.3
	silo.BlackBox.LastAssignedRisk = 0.8
	silo.BlackBox.CurrentExecution = 0.7
	silo.BlackBox.ObediencePulse = 2
	before := silo.BlackBox

	UpdateBlackBoxMonthly(silo)

	if silo.SafeguardRisk != 0.5 {
		t.Fatalf("gated settlement must not touch SafeguardRisk: %v", silo.SafeguardRisk)
	}
	if silo.BlackBox.Theta != before.Theta ||
		silo.BlackBox.CurrentExecution != before.CurrentExecution ||
		silo.BlackBox.ObediencePulse != before.ObediencePulse ||
		silo.BlackBox.LastAssignedRisk != before.LastAssignedRisk ||
		silo.BlackBox.Initialized {
		t.Fatalf("gated settlement must leave BlackBox bit-identical: %+v -> %+v",
			before, silo.BlackBox)
	}
}

// TestBlackBoxZeroRNGConsumption 零 RNG：结算前后 RNG 流状态一致
// (固定序列抽样对比)。
func TestBlackBoxZeroRNGConsumption(t *testing.T) {
	prev := rng
	defer SetRNG(prev)

	silo := blackBoxTestSilo()
	silo.BlackBox.LastAssignedRisk = 0.6
	silo.BlackBox.Theta = 0.2
	silo.BlackBox.ObediencePulse = BlackBoxObediencePulseMonths

	reference := rand.New(rand.NewSource(777))
	SetRNG(rand.New(rand.NewSource(777)))
	UpdateBlackBoxMonthly(silo)
	UpdateBlackBoxMonthly(silo)
	for i := 0; i < 64; i++ {
		if got, want := DrawFloat64(), reference.Float64(); got != want {
			t.Fatalf("black-box settlement consumed the RNG stream: draw %d = %v, want %v",
				i, got, want)
		}
	}
}

// TestBlackBoxObediencePulseRaisesExecutionAndDecays G10 表面服从脉冲
// (22-02 §3.5)：Θ 不变、E 暂时抬高，随脉冲剩余月递减直至耗尽。
func TestBlackBoxObediencePulseRaisesExecutionAndDecays(t *testing.T) {
	base := blackBoxTestSilo()
	base.BlackBox.LastAssignedRisk = 0.45
	pulsed := blackBoxTestSilo()
	pulsed.BlackBox.LastAssignedRisk = 0.45
	pulsed.BlackBox.ObediencePulse = BlackBoxObediencePulseMonths

	// 第 1 月：满幅 ΔE_show，脉冲减一；Θ 动力学不受脉冲影响。
	UpdateBlackBoxMonthly(base)
	UpdateBlackBoxMonthly(pulsed)
	if want := clamp01(base.BlackBox.CurrentExecution + BlackBoxObediencePulseDeltaE); math.Abs(pulsed.BlackBox.CurrentExecution-want) > 1e-12 {
		t.Fatalf("pulse month 1: E = %v, want %v", pulsed.BlackBox.CurrentExecution, want)
	}
	if pulsed.BlackBox.ObediencePulse != BlackBoxObediencePulseMonths-1 {
		t.Fatalf("pulse must decrement after settlement: %v", pulsed.BlackBox.ObediencePulse)
	}
	if pulsed.BlackBox.Theta != base.BlackBox.Theta {
		t.Fatalf("pulse must not move Θ: %v vs %v", pulsed.BlackBox.Theta, base.BlackBox.Theta)
	}

	// 第 2 月：半幅。
	UpdateBlackBoxMonthly(base)
	UpdateBlackBoxMonthly(pulsed)
	if want := clamp01(base.BlackBox.CurrentExecution + BlackBoxObediencePulseDeltaE*0.5); math.Abs(pulsed.BlackBox.CurrentExecution-want) > 1e-12 {
		t.Fatalf("pulse month 2: E = %v, want %v", pulsed.BlackBox.CurrentExecution, want)
	}
	if pulsed.BlackBox.ObediencePulse != 0 {
		t.Fatalf("pulse must exhaust after %d months: %v", BlackBoxObediencePulseMonths, pulsed.BlackBox.ObediencePulse)
	}

	// 第 3 月：脉冲耗尽，与无脉冲基线逐位一致。
	UpdateBlackBoxMonthly(base)
	UpdateBlackBoxMonthly(pulsed)
	if pulsed.BlackBox.CurrentExecution != base.BlackBox.CurrentExecution ||
		pulsed.BlackBox.Theta != base.BlackBox.Theta {
		t.Fatalf("after pulse exhaustion must match baseline: E=%v/%v Θ=%v/%v",
			pulsed.BlackBox.CurrentExecution, base.BlackBox.CurrentExecution,
			pulsed.BlackBox.Theta, base.BlackBox.Theta)
	}
}

// TestBlackBoxFirstMonthUsesCurrentSafeguardRisk 01 卷 §8 连续性：
// LastAssignedRisk 零值 (首次启用月) 以当前 SafeguardRisk 为生效 r。
func TestBlackBoxFirstMonthUsesCurrentSafeguardRisk(t *testing.T) {
	silo := blackBoxTestSilo() // LastAssignedRisk 零值
	silo.SafeguardRisk = 1.0

	UpdateBlackBoxMonthly(silo)

	if math.Abs(silo.BlackBox.Theta-0.12) > 1e-12 {
		t.Fatalf("first month at SafeguardRisk=1.0: Θ = %v, want 0.12", silo.BlackBox.Theta)
	}
	if want := blackBoxExecution(1.0, silo.BlackBox.Theta); math.Abs(silo.BlackBox.CurrentExecution-want) > 1e-12 {
		t.Fatalf("first month E must evaluate at r_eff = SafeguardRisk: %v, want %v",
			silo.BlackBox.CurrentExecution, want)
	}
}

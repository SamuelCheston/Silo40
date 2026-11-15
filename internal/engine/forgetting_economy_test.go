package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

// TestDiminishDenominatorCapsAtObsCap 锁定 cap 尾部：n=cap 与 n>cap 的同强度
// 观察使用同一分母；n=1 的早期递减公式保持不变。
func TestDiminishDenominatorCapsAtObsCap(t *testing.T) {
	cfg := DefaultInformationConfig()
	if cfg.DiminishObsCap != 4 {
		t.Fatalf("DiminishObsCap = %d, want the calibrated value 4", cfg.DiminishObsCap)
	}
	build := func(obsCount int) *model.FactBelief {
		return &model.FactBelief{
			FactID: "Judicial_1", B: 0.5, ObsCount: obsCount,
			Versions: InitialVersions(FragmentSpecFor("Judicial_1")),
		}
	}
	spec := FragmentSpecFor("Judicial_1")

	atCap, overCap := build(cfg.DiminishObsCap), build(cfg.DiminishObsCap+6)
	ObserveFact(cfg, atCap, spec, "obs#cap", 0.4)
	ObserveFact(cfg, overCap, spec, "obs#over", 0.4)
	if atCap.B != overCap.B {
		t.Fatalf("denominator must cap at ObsCount=%d: B(n=cap)=%v vs B(n=cap+6)=%v",
			cfg.DiminishObsCap, atCap.B, overCap.B)
	}

	early := build(1)
	ObserveFact(cfg, early, spec, "obs#early", 0.4)
	want := clamp01(1 - (1-0.5)*(1-0.4/(1+cfg.DiminishPerN)))
	if math.Abs(early.B-want) >= 1e-12 {
		t.Fatalf("n=1 denominator changed by the cap: B=%v, want %v", early.B, want)
	}
}

// TestDiminishingReturnsBoundedAtCap 锁定有界递减：n≤cap 时每新增一次观察，
// 同强度吸收增量仍严格递减；n>cap 后增量不再趋零，而固定在 cap 对应水平。
func TestDiminishingReturnsBoundedAtCap(t *testing.T) {
	cfg := DefaultInformationConfig()
	spec := FragmentSpecFor("Judicial_1")
	gains := make([]float64, cfg.DiminishObsCap+4)
	for n := range gains {
		belief := &model.FactBelief{
			FactID: "Judicial_1", B: 0.1, ObsCount: n,
			Versions: InitialVersions(spec),
		}
		ObserveFact(cfg, belief, spec, "obs", 0.4)
		gains[n] = belief.B - 0.1
		if n > 0 && n <= cfg.DiminishObsCap && gains[n] >= gains[n-1] {
			t.Fatalf("early diminishing returns changed at n=%d: gain %v >= previous %v", n, gains[n], gains[n-1])
		}
		if n > cfg.DiminishObsCap && math.Abs(gains[n]-gains[cfg.DiminishObsCap]) >= 1e-12 {
			t.Fatalf("tail gain is not bounded at n=%d: gain=%v cap gain=%v", n, gains[n], gains[cfg.DiminishObsCap])
		}
	}
	wantTail := (1 - 0.1) * 0.4 / (1 + cfg.DiminishPerN*float64(cfg.DiminishObsCap))
	if gains[cfg.DiminishObsCap] <= 0 || math.Abs(gains[cfg.DiminishObsCap]-wantTail) >= 1e-12 {
		t.Fatalf("cap tail gain = %v, want positive bounded gain %v", gains[cfg.DiminishObsCap], wantTail)
	}
}

// TestCirculationOutpacesForgetting 锁定长期流通经济：典型强度 0.25 的每月
// 同源转述能把部门信念从 0.85 托住并继续抬升，而不是在大 n 尾部输给淡忘。
func TestCirculationOutpacesForgetting(t *testing.T) {
	cfg := DefaultInformationConfig()
	if cfg.IndependenceRe != 0.5 {
		t.Fatalf("IndependenceRe = %v, want the calibrated value 0.5", cfg.IndependenceRe)
	}
	silo := permeationTestSilo(model.FactBelief{FactID: "Judicial_1", B: 0.85})

	for month := 1; month <= 240; month++ {
		AbsorbSharedFact(cfg, &silo.Professions[0].Beliefs, "Judicial_1", "player", 0.25)
		ApplyBeliefForgetting(silo, cfg)
		if b := silo.Professions[0].Beliefs[0].B; b < 0.85 {
			t.Fatalf("month %d: monthly circulation lost to forgetting: B=%v < 0.85", month, b)
		}
	}
	if final := silo.Professions[0].Beliefs[0].B; final < 0.90 {
		t.Fatalf("circulation must not merely hold 0.85 but converge upward: B=%v", final)
	}
}

// TestForgettingWindowAtDesignRate 锁定 0.005 下的渗透窗口：B₀=0.4525 时，
// 无干预滑行第 30 月仍在地板上，第 31 月跌破，成熟前存在明确补流通压力。
func TestForgettingWindowAtDesignRate(t *testing.T) {
	cfg := DefaultInformationConfig()
	if cfg.ForgetRateMonthly != 0.005 {
		t.Fatalf("ForgetRateMonthly = %v, want the calibrated value 0.005", cfg.ForgetRateMonthly)
	}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.4525})

	for month := 1; month <= cfg.PermeationMonths; month++ {
		ApplyBeliefForgetting(silo, cfg)
		if b := silo.Professions[0].Beliefs[0].B; b < cfg.QualRumor {
			t.Fatalf("month %d: B=%v fell below the floor before the %d-month coast ended", month, b, cfg.PermeationMonths)
		}
	}
	ApplyBeliefForgetting(silo, cfg)
	if b := silo.Professions[0].Beliefs[0].B; b >= cfg.QualRumor {
		t.Fatalf("the coast must end once (B0-floor)/r months pass: B=%v still above the floor", b)
	}
}

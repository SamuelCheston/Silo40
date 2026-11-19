package engine

import (
	"reflect"
	"testing"

	"silo40/internal/model"
)

// permeationTestSilo builds a silo with one department holding the given
// beliefs (department layer — the only layer permeation may touch).
func permeationTestSilo(beliefs ...model.FactBelief) *model.Silo {
	return &model.Silo{
		ID:   1,
		Name: "Permeation Test",
		Professions: []model.Profession{
			{ID: 1, Name: "IT", Beliefs: beliefs},
		},
	}
}

// TestPermeationMaturesAfterFullRun locks 18-time-permeation §3.1: a fact
// whose B stays at or above the sharing floor matures exactly when the
// consecutive-month counter reaches PermeationMonths — not one month earlier.
func TestPermeationMaturesAfterFullRun(t *testing.T) {
	cfg := DefaultInformationConfig()
	if cfg.PermeationMonths < 24 || cfg.PermeationMonths > 36 {
		t.Fatalf("PermeationMonths %d outside the designed [24, 36] band", cfg.PermeationMonths)
	}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})

	for month := 1; month <= cfg.PermeationMonths; month++ {
		ApplyPermeation(silo, cfg)
		belief := silo.Professions[0].Beliefs[0]
		if month < cfg.PermeationMonths {
			if belief.Matured {
				t.Fatalf("matured at month %d, before the full %d-month run", month, cfg.PermeationMonths)
			}
			if belief.PermeationCounter != month {
				t.Fatalf("month %d: counter = %d, want %d", month, belief.PermeationCounter, month)
			}
		}
	}
	belief := silo.Professions[0].Beliefs[0]
	if !belief.Matured {
		t.Fatalf("counter reached %d but the fact never matured: %+v", cfg.PermeationMonths, belief)
	}
}

// TestPermeationStickyAfterDecay locks §3.4 (粘性成熟): once matured, a fact
// stays matured while B decays below the floor and all the way to zero —
// "证据一旦成立就不可收回".
func TestPermeationStickyAfterDecay(t *testing.T) {
	cfg := DefaultInformationConfig()
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55, PermeationCounter: cfg.PermeationMonths - 1})
	ApplyPermeation(silo, cfg)
	if !silo.Professions[0].Beliefs[0].Matured {
		t.Fatal("setup: the fact must mature on the first tick")
	}

	// Forget the fact to the floor and below; maturity must not regress. The
	// horizon derives from the configured rate so the lock survives retunes.
	horizon := int(0.55/cfg.ForgetRateMonthly) + 5
	for i := 0; i < horizon; i++ {
		ApplyBeliefForgetting(silo, cfg)
		ApplyPermeation(silo, cfg)
	}
	belief := silo.Professions[0].Beliefs[0]
	if belief.B != 0 {
		t.Fatalf("forgetting did not run B down to 0: %v", belief.B)
	}
	if !belief.Matured {
		t.Fatal("maturity regressed after B decayed below the floor")
	}
}

// TestPermeationCounterResetsBelowFloor locks §3.1 counting semantics: a month
// below the floor zeroes the counter (restart, not pause) — "断了就要重新渗透".
func TestPermeationCounterResetsBelowFloor(t *testing.T) {
	cfg := DefaultInformationConfig()
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	for i := 0; i < 5; i++ {
		ApplyPermeation(silo, cfg)
	}
	if got := silo.Professions[0].Beliefs[0].PermeationCounter; got != 5 {
		t.Fatalf("counter = %d, want 5", got)
	}

	// One month under the floor (simulate heavy forgetting) resets the run.
	belief := &silo.Professions[0].Beliefs[0]
	belief.B = cfg.QualRumor - 0.01
	ApplyPermeation(silo, cfg)
	if belief.PermeationCounter != 0 || belief.Matured {
		t.Fatalf("below-floor month must zero the counter without maturing: %+v", *belief)
	}

	// Back above the floor the run restarts from 1.
	belief.B = 0.45
	ApplyPermeation(silo, cfg)
	if belief.PermeationCounter != 1 {
		t.Fatalf("restarted counter = %d, want 1", belief.PermeationCounter)
	}
}

// TestPermeationFloorBoundary locks the shared-floor rule (§3.3 第 3 条):
// B exactly at QualRumor still counts; one float step below does not.
func TestPermeationFloorBoundary(t *testing.T) {
	cfg := DefaultInformationConfig()
	atFloor := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: cfg.QualRumor})
	ApplyPermeation(atFloor, cfg)
	if got := atFloor.Professions[0].Beliefs[0].PermeationCounter; got != 1 {
		t.Fatalf("B exactly at the floor must count: counter = %d", got)
	}

	under := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: cfg.QualRumor - 1e-9})
	ApplyPermeation(under, cfg)
	if got := under.Professions[0].Beliefs[0].PermeationCounter; got != 0 {
		t.Fatalf("B under the floor must not count: counter = %d", got)
	}
}

// TestPermeationDoesNotTouchBeliefValues locks §3.3 第 1 条: permeation is a
// read-only derivation — it never rewrites B or π, and it consumes no RNG
// (same input state twice → bit-identical output).
func TestPermeationDoesNotTouchBeliefValues(t *testing.T) {
	cfg := DefaultInformationConfig()
	build := func() *model.Silo {
		return permeationTestSilo(model.FactBelief{
			FactID:   "Judicial_1",
			B:        0.6,
			Versions: map[string]float64{VersionITDirected: 0.5, VersionMundane: 0.5},
		})
	}
	a, b := build(), build()
	for i := 0; i < 3; i++ {
		ApplyPermeation(a, cfg)
		ApplyPermeation(b, cfg)
	}
	beliefA, beliefB := a.Professions[0].Beliefs[0], b.Professions[0].Beliefs[0]
	if beliefA.B != 0.6 || beliefB.B != 0.6 {
		t.Fatalf("permeation rewrote B: %v / %v", beliefA.B, beliefB.B)
	}
	if beliefA.Versions[VersionITDirected] != 0.5 || beliefA.Versions[VersionMundane] != 0.5 {
		t.Fatalf("permeation rewrote π: %+v", beliefA.Versions)
	}
	if !reflect.DeepEqual(beliefA, beliefB) {
		t.Fatalf("same state diverged across two runs: %+v vs %+v", beliefA, beliefB)
	}
}

// TestPermeationIgnoresNonDepartmentLayers locks §3.2: only Profession.Beliefs
// permeate. Agent beliefs (the player's own evidence chain) and director
// beliefs keep their zero permeation state, preserving the opening share feel.
func TestPermeationIgnoresNonDepartmentLayers(t *testing.T) {
	cfg := DefaultInformationConfig()
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	agent := &model.Agent{ID: 1, Beliefs: []model.FactBelief{{FactID: "IT_1", B: 0.9}}}
	silo.Protocol.DirectorBeliefs = []model.FactBelief{{FactID: "Order_1", B: 0.9}}

	for i := 0; i < cfg.PermeationMonths+2; i++ {
		ApplyPermeation(silo, cfg)
	}
	if agent.Beliefs[0].PermeationCounter != 0 || agent.Beliefs[0].Matured {
		t.Fatalf("agent layer must not permeate: %+v", agent.Beliefs[0])
	}
	if silo.Protocol.DirectorBeliefs[0].PermeationCounter != 0 || silo.Protocol.DirectorBeliefs[0].Matured {
		t.Fatalf("director layer must not permeate: %+v", silo.Protocol.DirectorBeliefs[0])
	}
	if !silo.Professions[0].Beliefs[0].Matured {
		t.Fatal("department layer must mature in the same window")
	}
}

// TestPermeationDisabledConfig locks the no-op edges: PermeationMonths <= 0 or
// a nil silo leaves everything untouched (config kill-switch semantics).
func TestPermeationDisabledConfig(t *testing.T) {
	disabled := DefaultInformationConfig()
	disabled.PermeationMonths = 0
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	ApplyPermeation(silo, disabled)
	if got := silo.Professions[0].Beliefs[0]; got.PermeationCounter != 0 || got.Matured {
		t.Fatalf("PermeationMonths=0 must be a no-op: %+v", got)
	}
	ApplyPermeation(nil, DefaultInformationConfig()) // must not panic
}

// TestPermeationMonthsLeftLocked locks the UI helper (§7.2 反馈面): remaining
// months count down with the counter, floor at 1 before maturity, and report
// 0 once matured.
func TestPermeationMonthsLeftLocked(t *testing.T) {
	cfg := DefaultInformationConfig()
	fresh := model.FactBelief{FactID: "IT_1", B: 0.5}
	if got := PermeationMonthsLeft(fresh, cfg); got != cfg.PermeationMonths {
		t.Fatalf("fresh belief months-left = %d, want %d", got, cfg.PermeationMonths)
	}
	mid := model.FactBelief{FactID: "IT_1", B: 0.5, PermeationCounter: cfg.PermeationMonths - 1}
	if got := PermeationMonthsLeft(mid, cfg); got != 1 {
		t.Fatalf("one tick before maturity months-left = %d, want 1", got)
	}
	matured := model.FactBelief{FactID: "IT_1", B: 0.1, PermeationCounter: cfg.PermeationMonths, Matured: true}
	if got := PermeationMonthsLeft(matured, cfg); got != 0 {
		t.Fatalf("matured belief months-left = %d, want 0", got)
	}
}

// TestMaturedFactCountLocked locks the gate-side reader: only Matured entries
// count, regardless of their current B.
func TestMaturedFactCountLocked(t *testing.T) {
	prof := &model.Profession{Name: "IT", Beliefs: []model.FactBelief{
		{FactID: "a", B: 0.9, Matured: true},
		{FactID: "b", B: 0.0, Matured: true}, // decayed to zero, still matured (sticky)
		{FactID: "c", B: 0.9},                // above floor but not yet matured
	}}
	if got := MaturedFactCount(prof); got != 2 {
		t.Fatalf("matured count = %d, want 2", got)
	}
	if got := MaturedFactCount(nil); got != 0 {
		t.Fatalf("nil department count = %d, want 0", got)
	}
}

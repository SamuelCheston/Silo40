package engine

import (
	"math/rand"
	"testing"
)

// TestPressureReleaseLetsAuthorityRebound locks the 回摆 batch scenario:
// after sustained pressure stops, the authority recovers — IT maintenance
// resumes, loyalty drifts back, and W climbs off the trough. Determinism:
// no RNG on this path except the IT loss-of-control rolls (seeded below).
func TestPressureReleaseLetsAuthorityRebound(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(99)))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Rebound Fixture", 122, nil)
	InitializeAuthorityState(silo, false)
	InitializeItAuthority(silo)

	// Pressure phase: monthly shocks and loyalty attacks.
	for i := 0; i < 6; i++ {
		QueueAuthorityShock(silo, -10)
		silo.ItAuthority.BailiffLoyalty = clamp01(silo.ItAuthority.BailiffLoyalty - 0.1)
		AdvanceSiloClock(silo)
		AdvanceAuthorityMonthly(silo)
		e.AdvanceProtocolMonthly(silo)
		e.RunITAuthorityMonthly(silo, nil)
	}
	trough := silo.Authority.W
	if trough >= 70 {
		t.Fatalf("pressure phase must push W down: %v", trough)
	}

	// Release phase: no more shocks; IT maintenance and loyalty reversion act.
	for i := 0; i < 8; i++ {
		AdvanceSiloClock(silo)
		AdvanceAuthorityMonthly(silo)
		e.AdvanceProtocolMonthly(silo)
		e.RunITAuthorityMonthly(silo, nil)
	}
	if silo.Authority.W <= trough {
		t.Fatalf("authority must rebound after pressure release: trough=%.2f after=%.2f",
			trough, silo.Authority.W)
	}
}

// TestOrderDeviationSafeguardRaceEndsInDeath locks the Safeguard 竞赛 batch
// scenario: accumulated Protocol deviations drive R_s to 1.0 and the hard
// guardrail ends the silo in DEATH (§6.2/§6.3 tables).
func TestOrderDeviationSafeguardRaceEndsInDeath(t *testing.T) {
	e, silo, clauses := protocolFixture(t)

	RecordProtocolAction(silo, clauses, "unauthorized_cross_contact", true)
	RecordProtocolAction(silo, clauses, "required_report_cleanup", true)
	RecordProtocolAction(silo, clauses, "unauthorized_disappearance", false)
	RecordProtocolAction(silo, clauses, "evidence_forgery", false)

	// Sustained bad outcomes: unmet reviews (rebellion/cohesion) and an
	// overdue required report keep adding R_s every month.
	silo.Rebellion = 0.9
	silo.Cohesion = 0.05
	silo.EventTrigger = 2
	for i := 0; i < 60 && silo.SafeguardRisk < 1.0; i++ {
		AdvanceSiloClock(silo)
		e.AdvanceProtocolMonthly(silo)
	}
	if silo.SafeguardRisk < 1.0 {
		t.Fatalf("safeguard race did not reach the hard fail: R_s=%.3f", silo.SafeguardRisk)
	}

	e.checkVictoryConditions(silo, nil)
	if silo.VictoryStatus == nil || silo.VictoryStatus.Type != "DEATH" {
		t.Fatalf("R_s >= 1.0 must end in DEATH: %+v", silo.VictoryStatus)
	}
}

package engine

import (
	"testing"
)

// dailyTroughInputs 逼近校准测量的日常压力谷底带 (W_eq ≈ 25–32)：暴露取
// 中高位、二元联盟忠诚低位、交付与人事中性。这是"持续施压但无决定性事件"
// 的引擎级近似。
func dailyTroughInputs() AuthorityInputs {
	return AuthorityInputs{
		BailiffLoyalty:    0.2,
		ITInternalLoyalty: 0.2,
		Exposure:          0.6,
	}
}

func TestDailyPressureAloneNeverOpensPactGate(t *testing.T) {
	silo := CreateInitialSilo("station", 2401, nil)
	InitializeAuthorityState(silo, false)
	j := &silo.Authority.Judicial
	j.PactQualified = true
	j.PublicEvidenceQ = 0.9
	j.IndependentPublicFacts = 2

	inputs := dailyTroughInputs()
	for month := 0; month < 24; month++ {
		AdvanceAuthority(silo, inputs)
		if silo.Authority.W <= 18 {
			t.Fatalf("daily pressure alone reached the covenant gate: W=%.2f at month %d (W_eq floor must stay above 18)",
				silo.Authority.W, month+1)
		}
	}
	if j.PactLocked {
		t.Fatal("pact locked without any decisive shock")
	}
	if len(silo.Authority.DecisiveShocks) != 0 {
		t.Fatalf("unexpected decisive shocks recorded: %+v", silo.Authority.DecisiveShocks)
	}
}

func TestSingleDecisiveShockBreaksFloorAndLocksPact(t *testing.T) {
	silo := CreateInitialSilo("station", 2401, nil)
	InitializeAuthorityState(silo, false)
	j := &silo.Authority.Judicial
	j.PactQualified = true
	j.PublicEvidenceQ = 0.9
	j.IndependentPublicFacts = 2

	inputs := dailyTroughInputs()
	// Settle onto the daily floor first.
	for month := 0; month < 24; month++ {
		AdvanceAuthority(silo, inputs)
	}
	floor := silo.Authority.W
	if floor <= 18 {
		t.Fatalf("daily floor already below the gate before the shock: W=%.2f", floor)
	}

	if !QueueDecisiveShock(silo, DecisiveSourceMassEvidence, DecisiveShockMagnitude) {
		t.Fatal("first decisive shock was not queued")
	}
	if QueueDecisiveShock(silo, DecisiveSourceMassEvidence, DecisiveShockMagnitude) {
		t.Fatal("same source queued twice (same-source shocks must not stack)")
	}

	lockedMonth := 0
	lowStreak, maxLowStreak := 0, 0
	for month := 1; month <= 12 && !j.PactLocked; month++ {
		AdvanceAuthority(silo, inputs)
		if silo.Authority.W <= 18 {
			lowStreak++
			if lowStreak > maxLowStreak {
				maxLowStreak = lowStreak
			}
		} else {
			lowStreak = 0
		}
		if j.PactLocked {
			lockedMonth = month
		}
	}
	if !j.PactLocked {
		t.Fatalf("decisive shock did not lock the pact: W=%.2f lowStreakMax=%d", silo.Authority.W, maxLowStreak)
	}
	if maxLowStreak < 3 {
		t.Fatalf("pact locked without a sustained W<=18 window: maxLowStreak=%d", maxLowStreak)
	}
	if j.Regime != "PACT" {
		t.Fatalf("regime after lock = %q, want PACT", j.Regime)
	}
	// Audit trail: the record stays even after consumption; the cursor advanced.
	if len(silo.Authority.DecisiveShocks) != 1 || silo.Authority.DecisiveApplied != 1 {
		t.Fatalf("audit trail wrong: shocks=%+v applied=%d", silo.Authority.DecisiveShocks, silo.Authority.DecisiveApplied)
	}
	t.Logf("floor=%.2f lockedAtMonth=%d maxLowStreak=%d", floor, lockedMonth, maxLowStreak)
}

func TestDecisiveShockQueueDedupAndAudit(t *testing.T) {
	silo := CreateInitialSilo("station", 2401, nil)
	InitializeAuthorityState(silo, false)
	silo.CurrentYear, silo.CurrentMonth = 2402, 3

	if !QueueDecisiveShock(silo, DecisiveSourceSilo1Destroyed, DecisiveShockMagnitude) {
		t.Fatal("silo1 shock not queued")
	}
	if !QueueDecisiveShock(silo, DecisiveSourceTerminationFailed, DecisiveShockMagnitude) {
		t.Fatal("termination shock not queued (distinct sources must both queue)")
	}
	if QueueDecisiveShock(silo, DecisiveSourceSilo1Destroyed, DecisiveShockMagnitude) {
		t.Fatal("silo1 shock queued twice")
	}
	if QueueDecisiveShock(silo, "", DecisiveShockMagnitude) || QueueDecisiveShock(silo, "x", 0) {
		t.Fatal("empty source or zero value must not queue")
	}

	got := silo.Authority.DecisiveShocks
	if len(got) != 2 || got[0].Source != DecisiveSourceSilo1Destroyed || got[1].Source != DecisiveSourceTerminationFailed {
		t.Fatalf("queue order/content wrong: %+v", got)
	}
	if got[0].Year != 2402 || got[0].Month != 3 {
		t.Fatalf("queue timestamp wrong: %+v", got[0])
	}

	// One monthly update consumes both pending shocks (deterministic order)
	// and the records remain for audit.
	wBefore := silo.Authority.W
	AdvanceAuthority(silo, AuthorityInputs{})
	if len(silo.Authority.DecisiveShocks) != 2 || silo.Authority.DecisiveApplied != 2 {
		t.Fatalf("consumption wrong: shocks=%+v applied=%d", silo.Authority.DecisiveShocks, silo.Authority.DecisiveApplied)
	}
	if silo.Authority.W >= wBefore {
		t.Fatalf("two negative shocks did not lower W: before=%.2f after=%.2f", wBefore, silo.Authority.W)
	}
}

package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestSampledSuspicionKeepsVisibilityFactors locks the SAMPLED path (§8.3):
// duty actions by sampled identities keep their visibility factor (Mayor ×2.0
// on the 0.01 base penalty). The old blanket profession multiplier is gone,
// but SAMPLED magnitudes are preserved this round.
func TestSampledSuspicionKeepsVisibilityFactors(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MAYOR_PUBLIC_ADDRESS", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if !almostEqual(mayor.SuspicionLevel, 0.02) {
		t.Errorf("Mayor suspicion gain = %v, want 0.02 (0.01 x V_identity 2.0)", mayor.SuspicionLevel)
	}
}

// TestMinesAreStructurallyUnsampled locks §8.2: miners' ordinary behavior is
// UNSAMPLED — no sampling RNG, no ordinary suspicion increment. AP still
// settles independently (paid as usual below).
func TestMinesAreStructurallyUnsampled(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	miner := CreateInitialAgent("MinerAgent", "Mines", nil, silo)
	apBefore := miner.ActionPoints

	res := e.ExecuteAgentAction(silo, miner, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MINES_DEEP_EXCAVATION", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if miner.SuspicionLevel != 0 {
		t.Errorf("Mines suspicion gain = %v, want 0 (structurally UNSAMPLED)", miner.SuspicionLevel)
	}
	if miner.ActionPoints != apBefore-15 {
		t.Errorf("UNSAMPLED must not waive AP: before=%v after=%v", apBefore, miner.ActionPoints)
	}
}

// TestClassifyDetection locks the three-class routing: Protocol touch wins,
// Mines identity or Mines location is structurally unsampled, everything else
// is sampled. IT touching Protocol must NOT take the ordinary multiplier.
func TestClassifyDetection(t *testing.T) {
	cases := []struct {
		name          string
		actor         string
		target        string
		protocolTouch bool
		want          string
	}{
		{"protocol touch wins", "IT", "Judicial", true, DetectDirectProtocol},
		{"miner identity", "Mines", "Judicial", false, DetectUnsampled},
		{"mines location", "IT", "Mines", false, DetectUnsampled},
		{"ordinary sampled", "IT", "Judicial", false, DetectSampled},
		{"mayor sampled", "Mayor", "Supply", false, DetectSampled},
	}
	for _, tc := range cases {
		if got := ClassifyDetection(tc.actor, tc.target, tc.protocolTouch); got != tc.want {
			t.Errorf("%s: ClassifyDetection(%q,%q,%v) = %s, want %s",
				tc.name, tc.actor, tc.target, tc.protocolTouch, got, tc.want)
		}
	}
}

// TestITSafeguardUnifiedPenaltyRemoved locks §11.8: the old blanket
// "every IT action adds safeguard risk" coupling is gone — safeguard
// consequences of IT behavior now come from Protocol deviation judgments.
func TestITSafeguardUnifiedPenaltyRemoved(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	itAgent := CreateInitialAgent("ITAgent", "IT", nil, silo)

	res := e.ExecuteAgentAction(silo, itAgent, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "IT_SURVEILLANCE", TargetDept: "Supply", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if silo.SafeguardRisk != 0 {
		t.Errorf("safeguard_risk = %v, want 0 (unified IT penalty removed)", silo.SafeguardRisk)
	}
	if !almostEqual(itAgent.SuspicionLevel, 0) {
		t.Errorf("IT_SURVEILLANCE suspicion = %v, want 0", itAgent.SuspicionLevel)
	}
}

// TestSafeguardRiskCouplingOnlyForIT keeps non-IT actors out of any safeguard
// coupling (regression guard after the removal).
func TestSafeguardRiskCouplingOnlyForIT(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MAYOR_PUBLIC_ADDRESS", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if silo.SafeguardRisk != 0 {
		t.Errorf("safeguard_risk = %v, want 0 for non-IT action", silo.SafeguardRisk)
	}
}

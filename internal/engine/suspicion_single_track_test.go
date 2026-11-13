package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Profession actions now apply the profession suspicion factor on the live
// JS path (previously they added the raw penalty; only the dead Go fallback
// applied the factor).
func TestSuspicionFactorAppliesToProfessionActions(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MAYOR_PUBLIC_ADDRESS", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	// Mayor factor x2.0 on the 0.01 base penalty.
	if !almostEqual(mayor.SuspicionLevel, 0.02) {
		t.Errorf("Mayor suspicion gain = %v, want 0.02 (0.01 x 2.0)", mayor.SuspicionLevel)
	}
}

func TestSuspicionFactorAppliesToMinesActions(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	miner := CreateInitialAgent("MinerAgent", "Mines", nil, silo)

	res := e.ExecuteAgentAction(silo, miner, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MINES_DEEP_EXCAVATION", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	// Mines factor x0.3 on the 0.005 base penalty.
	if !almostEqual(miner.SuspicionLevel, 0.0015) {
		t.Errorf("Mines suspicion gain = %v, want 0.0015 (0.005 x 0.3)", miner.SuspicionLevel)
	}
}

// Common actions keep using the same factor (regression guard for the
// prelude move).
func TestSuspicionFactorStillAppliesToCommonActions(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionGatherInfo, TargetDept: "IT", Cost: 10,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	// Mayor factor x2.0 on the 0.005 gather base.
	if !almostEqual(mayor.SuspicionLevel, 0.01) {
		t.Errorf("Mayor gather suspicion gain = %v, want 0.01 (0.005 x 2.0)", mayor.SuspicionLevel)
	}
}

// IT passive: every successful IT action worsens safeguard risk by
// cost x 0.002, for profession and common actions alike. IT actions gain no
// suspicion themselves (factor x0.3 on zero-base penalties).
func TestITSafeguardRiskCoupling(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Suspicion Silo", 122, nil)
	itAgent := CreateInitialAgent("ITAgent", "IT", nil, silo)

	res := e.ExecuteAgentAction(silo, itAgent, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "IT_SURVEILLANCE", TargetDept: "Mines", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if !almostEqual(silo.SafeguardRisk, 15*0.002) {
		t.Errorf("safeguard_risk = %v, want %v", silo.SafeguardRisk, 15*0.002)
	}
	if !almostEqual(itAgent.SuspicionLevel, 0) {
		t.Errorf("IT_SURVEILLANCE suspicion = %v, want 0", itAgent.SuspicionLevel)
	}

	// A common action by IT also feeds the coupling.
	before := silo.SafeguardRisk
	res = e.ExecuteAgentAction(silo, itAgent, model.AgentAction{
		Type: model.ActionBuildConnection, TargetDept: "Mines", Cost: 15,
	})
	if !res.Executed {
		t.Fatalf("expected execution, got %q", res.Message)
	}
	if !almostEqual(silo.SafeguardRisk-before, 15*0.002) {
		t.Errorf("safeguard_risk delta = %v, want %v", silo.SafeguardRisk-before, 15*0.002)
	}
}

// Non-IT actors must not feed the safeguard coupling.
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

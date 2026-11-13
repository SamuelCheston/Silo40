package engine

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// Profession actions must be validated server-side against the executing
// actor's profession. Routing alone only looks the action up by id, which
// used to let any profession execute any profession action.
func TestProfessionActionOwnershipEnforced(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Guard Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "IT_COVER_UP", TargetDept: "Mines", Cost: 25,
	})
	if res.Executed {
		t.Fatalf("expected Mayor to be rejected from IT_COVER_UP, got success: %s", res.Message)
	}
	if !strings.Contains(res.Message, "not available to Mayor") {
		t.Fatalf("expected explicit rejection message, got %q", res.Message)
	}
	if silo.SafeguardRisk != 0 {
		t.Fatalf("rejected action must not apply effects, safeguard_risk=%v", silo.SafeguardRisk)
	}
}

func TestProfessionActionOwnershipAllowsOwner(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Guard Silo", 122, nil)
	itAgent := CreateInitialAgent("ITAgent", "IT", nil, silo)

	res := e.ExecuteAgentAction(silo, itAgent, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "IT_COVER_UP", TargetDept: "Mines", Cost: 25,
	})
	if !res.Executed {
		t.Fatalf("expected IT to execute IT_COVER_UP, got: %s", res.Message)
	}
	if silo.SafeguardRisk <= 0 {
		t.Fatalf("expected IT_COVER_UP effects to apply, safeguard_risk=%v", silo.SafeguardRisk)
	}
}

func TestProfessionActionUnknownIDRejected(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Guard Silo", 122, nil)
	mayor := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	res := e.ExecuteAgentAction(silo, mayor, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "NOT_AN_ACTION", Cost: 15,
	})
	if res.Executed {
		t.Fatalf("expected unknown profession action to be rejected")
	}
	if !strings.Contains(res.Message, "Unknown profession action") {
		t.Fatalf("expected explicit unknown-action message, got %q", res.Message)
	}
}

// NPC departments submit their own profession actions through the same
// pipeline; the ownership check must not break them.
func TestProfessionActionOwnershipAllowsNpcOwner(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Guard Silo", 122, nil)

	dept := findDept(silo, "Mechanical")
	before := dept.ActionPoints
	view, err := CreateActorView(CreateActorRefForProfession(dept), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	res := e.SubmitAction(view.Ref, silo, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL", Cost: 15,
	}, nil)
	if !res.Executed {
		t.Fatalf("expected Mechanical dept to execute its own action, got: %s", res.Message)
	}
	if dept.ActionPoints >= before {
		t.Fatalf("expected AP to be charged, before=%v after=%v", before, dept.ActionPoints)
	}
}

package engine

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// Actions must fail explicitly when the JS mechanic definitions are
// unavailable. There is no silent Go fallback: action semantics live in the
// JS definitions as the single source of truth (the legacy hardcoded Go
// action implementations were removed).

func newMechaniclessEngine() *GameEngine {
	e := NewGameEngine()
	// Strip all loaded definitions to simulate mechanics being unavailable.
	e.MechanicEngine = &MechanicEngine{
		byEventType:        map[string]MechanicDefinition{},
		byActionType:       map[model.AgentActionType]MechanicDefinition{},
		byProfessionAction: map[string]MechanicDefinition{},
		byFormula:          map[string]MechanicDefinition{},
	}
	return e
}

func TestCommonActionFailsExplicitlyWithoutMechanics(t *testing.T) {
	e := newMechaniclessEngine()
	silo := CreateInitialSilo("Mechanicless", 122, nil)
	agent := CreateInitialAgent("Agent", "Mechanical", nil, silo)

	apBefore := agent.ActionPoints
	fragmentsBefore := len(agent.KnownFragments)

	res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 10,
	})
	if res.Executed {
		t.Fatalf("expected explicit failure without mechanic definitions, got success: %s", res.Message)
	}
	if res.Message == "" {
		t.Errorf("expected an explicit error message, got empty")
	}
	if agent.ActionPoints != apBefore {
		t.Errorf("AP changed on failed action: %v -> %v", apBefore, agent.ActionPoints)
	}
	if len(agent.KnownFragments) != fragmentsBefore {
		t.Errorf("fragments changed on failed action: %v -> %v", fragmentsBefore, len(agent.KnownFragments))
	}
}

func TestProfessionActionFailsExplicitlyWithoutMechanics(t *testing.T) {
	e := newMechaniclessEngine()
	silo := CreateInitialSilo("Mechanicless", 122, nil)
	agent := CreateInitialAgent("Agent", "Mechanical", nil, silo)

	res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL", Cost: 15,
	})
	if res.Executed {
		t.Fatalf("expected explicit failure without mechanic definitions, got success: %s", res.Message)
	}
	if !strings.Contains(res.Message, "Unknown profession action") && res.Message == "" {
		t.Errorf("expected an explicit error message, got %q", res.Message)
	}
}

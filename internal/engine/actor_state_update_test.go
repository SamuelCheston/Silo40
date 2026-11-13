package engine

import (
	"testing"
)

// UpdateActorState applies the JS AGENT_UPDATE mutations by re-resolving the
// actor ref through ApplyMutation. For PLAYER actors that resolution used to
// fail (agent pointer not passed along), so every mutation was silently
// dropped: no AP regeneration, no prestige recompute, no suspicion decay.
// Lock in that player updates actually land.
func TestUpdateActorStatePlayerMutationsLand(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Update Silo", 122, nil)
	agent := CreateInitialAgent("Updater", "Mayor", nil, silo)

	view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}

	agent.ActionPoints = 0
	initialPrestige := agent.PoliticalPrestige
	agent.SuspicionLevel = 0.5

	e.UpdateActorState(view, silo, 1.0, nil)

	if agent.ActionPoints <= 0 {
		t.Errorf("expected AP regeneration to land, got %v", agent.ActionPoints)
	}
	// Prestige is recomputed from connections/structure, not left at the
	// creation-time seed value.
	if agent.PoliticalPrestige == initialPrestige {
		t.Errorf("expected prestige to be recomputed, still %v", initialPrestige)
	}
	// Suspicion decays 0.05 per year.
	if agent.SuspicionLevel >= 0.5 {
		t.Errorf("expected suspicion decay, got %v", agent.SuspicionLevel)
	}
}

// NPC department views resolve from the silo alone; keep that path working.
func TestUpdateActorStateProfessionMutationsLand(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Update Silo", 122, nil)

	dept := findDept(silo, "Mines")
	dept.ActionPoints = 0
	view, err := CreateActorView(CreateActorRefForProfession(dept), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.UpdateActorState(view, silo, 1.0, nil)
	if dept.ActionPoints <= 0 {
		t.Errorf("expected department AP regeneration to land, got %v", dept.ActionPoints)
	}
}

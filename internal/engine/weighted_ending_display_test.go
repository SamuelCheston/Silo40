package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestApplyWeightedEndingShowsNarrative locks 25-information-endgame §2.2 at
// the engine boundary: once the arbitrator writes the settlement narrative
// into Reason, the ending screen field (VictoryStatus.Description) carries
// the story instead of the composed audit line; an empty Reason still falls
// back to the audit line.
func TestApplyWeightedEndingShowsNarrative(t *testing.T) {
	e := NewGameEngine()
	e.WeightedEndingResolver = func(silo *model.Silo, agent *model.Agent) *model.EndingDecision {
		return &model.EndingDecision{Selected: "INFORMATION", Reason: "链不再需要你。《共证》成立。"}
	}
	silo := &model.Silo{VictoryStatus: &model.VictoryStatus{Type: "INFORMATION", IsWon: true}}
	e.applyWeightedEnding(EVENT_VICTORY_CHECK, silo, &model.Agent{})
	if silo.VictoryStatus.Description != "链不再需要你。《共证》成立。" {
		t.Fatalf("description must carry the narrative, got %q", silo.VictoryStatus.Description)
	}

	e.WeightedEndingResolver = func(silo *model.Silo, agent *model.Agent) *model.EndingDecision {
		return &model.EndingDecision{Selected: "INFORMATION"}
	}
	silo2 := &model.Silo{VictoryStatus: &model.VictoryStatus{Type: "INFORMATION", IsWon: true}}
	e.applyWeightedEnding(EVENT_VICTORY_CHECK, silo2, &model.Agent{})
	if silo2.VictoryStatus.Description == "" {
		t.Fatal("empty Reason must fall back to the audit line, not blank the screen")
	}
}

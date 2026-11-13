package engine

import (
	"testing"

	"silo40/internal/model"
)

func TestWeightedEndingHookAppliesDecision(t *testing.T) {
	e := &GameEngine{}
	silo := &model.Silo{VictoryStatus: &model.VictoryStatus{IsWon: true, Type: "INFORMATION"}}
	decision := &model.EndingDecision{Selected: "EXCLUSIONIST"}
	e.WeightedEndingResolver = func(*model.Silo, *model.Agent) *model.EndingDecision {
		return decision
	}
	e.applyWeightedEnding(EVENT_VICTORY_CHECK, silo, &model.Agent{})
	if silo.VictoryStatus.Type != "EXCLUSIONIST" || silo.VictoryStatus.Decision != decision {
		t.Fatalf("weighted decision was not applied: %+v", silo.VictoryStatus)
	}
}

func TestWeightedEndingHookLeavesFailureUntouched(t *testing.T) {
	e := &GameEngine{}
	status := &model.VictoryStatus{IsWon: false, Type: "DEATH", Description: "failure"}
	silo := &model.Silo{VictoryStatus: status}
	e.WeightedEndingResolver = func(*model.Silo, *model.Agent) *model.EndingDecision {
		return nil
	}
	e.applyWeightedEnding(EVENT_VICTORY_CHECK, silo, &model.Agent{})
	if silo.VictoryStatus != status || silo.VictoryStatus.Type != "DEATH" || silo.VictoryStatus.Decision != nil {
		t.Fatalf("hard failure was unexpectedly replaced: %+v", silo.VictoryStatus)
	}
}

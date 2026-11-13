package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
)

func drainOperationForTest(t *testing.T, svc *GameService) *model.EventStepResult {
	t.Helper()
	for i := 0; i < 256; i++ {
		step, err := svc.ProcessNextEvent()
		if err != nil {
			t.Fatalf("process event: %v", err)
		}
		if step.OperationComplete {
			return step
		}
	}
	t.Fatal("operation did not settle within 256 steps")
	return nil
}

func advanceMonthForTest(t *testing.T, svc *GameService) *model.EventStepResult {
	t.Helper()
	if _, err := svc.BeginPassTime(1); err != nil {
		t.Fatalf("begin pass time: %v", err)
	}
	return drainOperationForTest(t, svc)
}

func TestSilo1OperationUsesDelayedLiveTargetPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "operation.db")
	svc := newTwoSiloService(t, dbPath)
	state, err := svc.CreateGame(threeSiloRequest(61))
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	found := false
	for _, action := range state.AvailableActions {
		if action.ActionType == model.ActionSilo1Operation && action.ID == Silo1OpDeceive && action.Enabled {
			found = true
		}
	}
	if !found {
		t.Fatal("live Silo 1 deception action is not available")
	}

	target := svc.fullSilo1()
	before := target.Rebellion
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpDeceive, Cost: 0}); err != nil {
		t.Fatalf("begin cross-silo action: %v", err)
	}
	step := drainOperationForTest(t, svc)
	if step.ActionResult == nil || !step.ActionResult.Executed {
		t.Fatalf("cross-silo action was not executed: %+v", step.ActionResult)
	}
	if svc.agent.ActionPoints != 30 {
		t.Fatalf("expected server-authoritative cost 20 AP, remaining=%v", svc.agent.ActionPoints)
	}
	if target.Rebellion != before || len(svc.world.Silo1.Operations) != 1 || svc.world.Silo1.Operations[0].Status != "PENDING" {
		t.Fatal("target changed before the operation due month")
	}

	advanceMonthForTest(t, svc)
	if svc.world.Silo1.Operations[0].Status != "PENDING" {
		t.Fatal("two-month operation resolved after only one month")
	}
	before = target.Rebellion
	step = advanceMonthForTest(t, svc)
	op := svc.world.Silo1.Operations[0]
	if op.Status != "RESOLVED" || !strings.Contains(op.Resolution, "叛乱") {
		t.Fatalf("operation did not resolve with an observable target report: %+v", op)
	}
	if target.Rebellion < before+0.15-1e-9 {
		t.Fatalf("live target rebellion did not rise: before=%v after=%v", before, target.Rebellion)
	}
	if svc.agent.SuspicionLevel < 0.04 {
		t.Fatalf("expected a reverse trace on the player, suspicion=%v", svc.agent.SuspicionLevel)
	}
	if step.World == nil || len(step.World.Silo1Operations) != 1 {
		t.Fatal("UI step result does not expose the resolved operation")
	}

	if err := svc.SaveGame(); err != nil {
		t.Fatalf("save game: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(fresh.world.Silo1.Operations) != 1 || fresh.world.Silo1.Operations[0].Status != "RESOLVED" {
		t.Fatal("resolved operation was not preserved across save/resume")
	}
}

func TestSilo1OperationEffects(t *testing.T) {
	for _, tc := range []struct {
		actionID string
		verify   func(*testing.T, *GameService, string)
	}{
		{Silo1OpIntel, func(t *testing.T, svc *GameService, report string) {
			if !strings.Contains(report, "合法性") || !strings.Contains(report, "Archives") {
				t.Fatalf("intel report lacks target facts: %s", report)
			}
		}},
		{Silo1OpDeceive, func(t *testing.T, svc *GameService, report string) {
			if !strings.Contains(report, "叛乱 0.04→0.19") || svc.fullSilo1().Rebellion != 0.19 {
				t.Fatalf("deception did not apply the measured rebellion delta: report=%s rebellion=%v", report, svc.fullSilo1().Rebellion)
			}
		}},
	} {
		t.Run(tc.actionID, func(t *testing.T) {
			svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "effect.db"))
			if _, err := svc.CreateGame(threeSiloRequest(62)); err != nil {
				t.Fatalf("create game: %v", err)
			}
			def, _ := silo1OperationDefinition(tc.actionID)
			result := svc.scheduleSilo1Operation(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: tc.actionID, Cost: float64(def.APCost)})
			if !result.Executed {
				t.Fatalf("schedule failed: %s", result.Message)
			}
			svc.world.Silo1.Operations[0].DueYear = svc.world.CurrentYear
			svc.world.Silo1.Operations[0].DueMonth = svc.world.CurrentMonth
			var logs []string
			var stories []model.StoryEvent
			svc.resolveSilo1Operations(&logs, &stories)
			op := svc.world.Silo1.Operations[0]
			if op.Status != "RESOLVED" || len(stories) != 1 {
				t.Fatalf("operation did not produce a player report: %+v", op)
			}
			tc.verify(t, svc, op.Resolution)
		})
	}
}

func TestSilo1CounterintelligenceEscalates(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "counterintel.db"))
	if _, err := svc.CreateGame(threeSiloRequest(63)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	stationBefore := svc.stationSilo().Legitimacy
	for i := 0; i < 3; i++ {
		result := svc.scheduleSilo1Operation(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpDeceive, Cost: 0})
		if !result.Executed {
			t.Fatalf("schedule %d failed: %s", i, result.Message)
		}
		svc.world.Silo1.Operations[i].DueYear = svc.world.CurrentYear
		svc.world.Silo1.Operations[i].DueMonth = svc.world.CurrentMonth
	}
	var logs []string
	var stories []model.StoryEvent
	svc.resolveSilo1Operations(&logs, &stories)
	if svc.world.Silo1.CounterintelStage != 1 {
		t.Fatalf("expected counterintelligence stage 1, got %d", svc.world.Silo1.CounterintelStage)
	}
	if svc.agent.SuspicionLevel < 0.19 {
		t.Fatalf("expected opening + operation + counterintel suspicion, got %v", svc.agent.SuspicionLevel)
	}
	if got := svc.stationSilo().Legitimacy; got != stationBefore-0.02 {
		t.Fatalf("counterintelligence must penalize the assigned station: before=%v after=%v", stationBefore, got)
	}
}

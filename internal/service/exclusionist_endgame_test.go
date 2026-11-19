package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

func TestExclusionistEndingPlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testExclusionistEndingPlayableFromNewGame)
}

func testExclusionistEndingPlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "exclusionist-playable.db"))
	if _, err := svc.CreateGame(threeSiloArchetypeRequest(2801, archetype)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance EXCLUSIONIST line at month %d: %v", month, err)
		}
	}

	t.Logf("settled at %d-%02d: status=%+v rebellion=%.3f population=%d",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus,
		svc.silo.Rebellion, svc.silo.TotalPopulation)
	if !contentTriggered(svc, "special:normalcy_hardens") {
		t.Fatalf("normalcy omen never fired through real play at %d-%02d",
			svc.silo.CurrentYear, svc.silo.CurrentMonth)
	}
	if svc.silo.Silo1Destroyed {
		t.Fatal("closed-order convergence only settled through the post-fall TIME request")
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingExclusionist {
		t.Fatalf("zero-input route did not settle as EXCLUSIONIST at %d-%02d: status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal EXCLUSIONIST state accepted another action: %+v", outcome.Result)
	}
}

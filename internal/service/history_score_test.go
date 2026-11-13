package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
)

func TestHistoryInfluenceUsesMeasuredDeltasAndAppliesOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history-score.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(81)); err != nil {
		t.Fatalf("create game: %v", err)
	}

	var logs []string
	var stories []model.StoryEvent
	if result := svc.scheduleSilo1Operation(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpDeceive, Cost: 20}); !result.Executed {
		t.Fatalf("schedule Silo 1: %s", result.Message)
	}
	svc.world.Silo1.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.Silo1.Operations[0].DueMonth = svc.world.CurrentMonth
	svc.resolveSilo1Operations(&logs, &stories)
	if result := svc.scheduleStationOperation(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive, Cost: 20}); !result.Executed {
		t.Fatalf("schedule station: %s", result.Message)
	}
	svc.world.StationOps.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.StationOps.Operations[0].DueMonth = svc.world.CurrentMonth
	svc.resolveStationOperations(&logs, &stories)

	if len(svc.world.HistoryImpacts) != 2 {
		t.Fatalf("expected two measured impacts, got %d", len(svc.world.HistoryImpacts))
	}
	if svc.world.HistoryImpacts[0].Points != 21 || svc.world.HistoryImpacts[1].Points != 21 {
		t.Fatalf("unexpected measured impact points: %+v", svc.world.HistoryImpacts)
	}

	svc.silo.VictoryStatus = &model.VictoryStatus{
		IsWon: true, Type: "time", Score: &model.GameScore{Total: 100},
	}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist ending: %v", err)
	}
	score := svc.silo.VictoryStatus.Score
	if score.Total != 142 || score.HistoryInfluence == nil || score.HistoryInfluence.Total != 42 || score.HistoryInfluence.Grade != "C" {
		t.Fatalf("unexpected ending score: %+v", score)
	}
	if len(score.HistoryInfluence.Silos) != 2 || score.HistoryInfluence.Silos[0].Points != 21 || score.HistoryInfluence.Silos[1].Points != 21 {
		t.Fatalf("unexpected per-silo breakdown: %+v", score.HistoryInfluence.Silos)
	}
	if !strings.Contains(score.HistoryInfluence.Silos[1].Summary, "叛乱 0.00→0.15") {
		t.Fatalf("station evidence lacks measured delta: %s", score.HistoryInfluence.Silos[1].Summary)
	}

	if err := svc.persist(); err != nil {
		t.Fatalf("persist ending again: %v", err)
	}
	if svc.silo.VictoryStatus.Score.Total != 142 {
		t.Fatalf("history bonus applied twice: %d", svc.silo.VictoryStatus.Score.Total)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	resumed := fresh.silo.VictoryStatus.Score.HistoryInfluence
	if resumed == nil || resumed.Total != 42 || len(resumed.Silos) != 2 {
		t.Fatalf("history score was not restored: %+v", resumed)
	}
}

func TestHistoryInfluenceWithoutOperationsIsZero(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "history-zero.db"))
	if _, err := svc.CreateGame(threeSiloRequest(82)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.silo.VictoryStatus = &model.VictoryStatus{
		IsWon: true, Type: "time", Score: &model.GameScore{Total: 77},
	}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist ending: %v", err)
	}
	score := svc.silo.VictoryStatus.Score
	if score.Total != 77 || score.HistoryInfluence == nil || score.HistoryInfluence.Total != 0 || score.HistoryInfluence.Grade != "D" {
		t.Fatalf("natural evolution should not award history points: %+v", score)
	}
	for _, breakdown := range score.HistoryInfluence.Silos {
		if breakdown.Points != 0 || breakdown.OperationCount != 0 {
			t.Fatalf("unexpected unearned breakdown: %+v", breakdown)
		}
	}
}

func TestHistoryInfluenceIntelRecordsEvidenceWithoutMetricPoints(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "history-intel.db"))
	if _, err := svc.CreateGame(threeSiloRequest(83)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if result := svc.scheduleSilo1Operation(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpIntel, Cost: 10}); !result.Executed {
		t.Fatalf("schedule intel: %s", result.Message)
	}
	svc.world.Silo1.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.Silo1.Operations[0].DueMonth = svc.world.CurrentMonth
	var logs []string
	var stories []model.StoryEvent
	svc.resolveSilo1Operations(&logs, &stories)
	if len(svc.world.HistoryImpacts) != 1 || svc.world.HistoryImpacts[0].Points != 0 {
		t.Fatalf("intel should leave zero-delta evidence: %+v", svc.world.HistoryImpacts)
	}
}

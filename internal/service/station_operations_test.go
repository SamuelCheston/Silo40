package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
)

func TestAssignedStationRange(t *testing.T) {
	for seed := int64(0); seed < 256; seed++ {
		number := assignedStationNumber(seed)
		if number < 2 || number > 17 {
			t.Fatalf("seed %d assigned excluded silo %d; want 2-17", seed, number)
		}
	}
}

func TestStationOperationUsesDelayedLiveTargetPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "station-operation.db")
	svc := newTwoSiloService(t, dbPath)
	state, err := svc.CreateGame(threeSiloRequest(71))
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	target := svc.stationSilo()
	if target == nil || target.ID != ActiveSiloID || svc.silo != target || svc.agent.SiloID != target.ID {
		t.Fatalf("station did not replace the primary silo: station=%+v agent=%+v", target, svc.agent)
	}
	if len(svc.silos) != 3 {
		t.Fatalf("expected exactly three silos, got %d", len(svc.silos))
	}
	if target.Name != fmt.Sprintf("%d 号地堡", assignedStationNumber(71)) {
		t.Fatalf("unexpected assigned station name: %s", target.Name)
	}
	if svc.world.PlayerSiloID != target.ID || svc.agent.StationSiloID != target.ID {
		t.Fatalf("world/agent do not point to the station: world=%d agent=%+v", svc.world.PlayerSiloID, svc.agent)
	}
	var persisted []model.Silo
	if err := svc.db.Order("id asc").Find(&persisted).Error; err != nil {
		t.Fatalf("load persisted silos: %v", err)
	}
	if len(persisted) != 3 {
		t.Fatalf("expected exactly three persisted silos, got %d", len(persisted))
	}
	for _, silo := range persisted {
		if silo.Name == "Silo 40" || silo.Name == "40 号地堡" {
			t.Fatalf("Silo 40 was created as an entity: %+v", silo)
		}
	}
	if state.World == nil || state.World.StationSilo == nil || state.World.StationSilo.ID != target.ID {
		t.Fatal("assigned station is missing from the UI snapshot")
	}

	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive, Cost: 0}); err != nil {
		t.Fatalf("begin station action: %v", err)
	}
	step := drainOperationForTest(t, svc)
	if step.ActionResult == nil || !step.ActionResult.Executed || svc.agent.ActionPoints != 30 {
		t.Fatalf("station action failed: result=%+v AP=%v", step.ActionResult, svc.agent.ActionPoints)
	}
	advanceMonthForTest(t, svc)
	if svc.world.StationOps.Operations[0].Status != "PENDING" {
		t.Fatal("station operation resolved before its due month")
	}
	before := target.Rebellion
	advanceMonthForTest(t, svc)
	op := svc.world.StationOps.Operations[0]
	if op.Status != "RESOLVED" || op.TargetSiloID != target.ID || !strings.Contains(op.Resolution, "叛乱") {
		t.Fatalf("station operation lacks a real target result: %+v", op)
	}
	if !strings.Contains(op.Resolution, "叛乱 0.04→0.19") || target.Rebellion < before+0.15-1e-9 {
		t.Fatalf("station operation must add exactly 0.15 before natural drift: before=%v after=%v report=%s", before, target.Rebellion, op.Resolution)
	}
	if err := svc.SaveGame(); err != nil {
		t.Fatalf("save: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(fresh.world.StationOps.Operations) != 1 || fresh.world.StationOps.Operations[0].Status != "RESOLVED" {
		t.Fatal("station operation did not survive save/resume")
	}
}

func TestOperationTargetBoundaries(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "target-boundaries.db"))
	state, err := svc.CreateGame(threeSiloRequest(72))
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	stationCount, silo1Count := 0, 0
	silo1IDs := map[string]bool{}
	for _, action := range state.AvailableActions {
		switch action.ActionType {
		case model.ActionStationOperation:
			stationCount++
		case model.ActionSilo1Operation:
			silo1Count++
			silo1IDs[action.ID] = true
		case model.AgentActionType("SILO18_OPERATION"):
			t.Fatal("Silo 18 operation entry is still exposed")
		}
	}
	if stationCount != 4 || silo1Count != 2 || !silo1IDs[Silo1OpIntel] || !silo1IDs[Silo1OpDeceive] {
		t.Fatalf("unexpected legal targets: station=%d silo1=%d ids=%v", stationCount, silo1Count, silo1IDs)
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.AgentActionType("SILO18_OPERATION"), ActionID: "SILO18_DECEIVE"}); err == nil {
		t.Fatal("legacy Silo 18 operation was not rejected")
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpInfiltrate}); err == nil {
		t.Fatal("Silo 1 infiltration must not be available through the limited channel")
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpTrade}); err == nil {
		t.Fatal("Silo 1 trade must not be available through the limited channel")
	}
}

func TestPlayerOperationsResolveInSiloIDOrder(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "operation-order.db"))
	if _, err := svc.CreateGame(threeSiloRequest(74)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if result := svc.scheduleStationOperation(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive, Cost: 20}); !result.Executed {
		t.Fatalf("schedule station: %s", result.Message)
	}
	if result := svc.scheduleSilo1Operation(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpDeceive, Cost: 20}); !result.Executed {
		t.Fatalf("schedule Silo 1: %s", result.Message)
	}
	svc.world.StationOps.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.StationOps.Operations[0].DueMonth = svc.world.CurrentMonth
	svc.world.Silo1.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.Silo1.Operations[0].DueMonth = svc.world.CurrentMonth

	var logs []string
	var stories []model.StoryEvent
	svc.resolvePlayerOperations(&logs, &stories)
	if len(logs) != 2 || !strings.HasPrefix(logs[0], "[StationOperation]") || !strings.HasPrefix(logs[1], "[Silo1Operation]") {
		t.Fatalf("operations did not resolve in ascending SiloID order: %v", logs)
	}
}

func TestStationAdditionalOperationEffects(t *testing.T) {
	for _, tc := range []struct {
		actionID string
		verify   func(*testing.T, *GameService, string, historyMetricSnapshot)
	}{
		{StationOpIntel, func(t *testing.T, svc *GameService, report string, before historyMetricSnapshot) {
			if !strings.Contains(report, "Mechanical") || !strings.Contains(report, "合法性") {
				t.Fatalf("intel lacks station evidence: %s", report)
			}
		}},
		{StationOpInfiltrate, func(t *testing.T, svc *GameService, report string, before historyMetricSnapshot) {
			target := svc.stationSilo()
			if target.Legitimacy != before.legitimacy-0.12 || target.Cohesion != before.cohesion-0.08 {
				t.Fatalf("infiltration delta mismatch: before=%+v legitimacy=%v cohesion=%v", before, target.Legitimacy, target.Cohesion)
			}
		}},
		{StationOpTrade, func(t *testing.T, svc *GameService, report string, before historyMetricSnapshot) {
			if got := siloResourceAmount(svc.stationSilo(), "Supplies"); got != before.supplies-100 {
				t.Fatalf("internal trade must consume exactly 100 supplies: before=%v after=%v", before.supplies, got)
			}
			if svc.silo.ID != svc.stationSilo().ID {
				t.Fatal("trade affected a separate Silo 40 entity")
			}
		}},
	} {
		t.Run(tc.actionID, func(t *testing.T) {
			svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "station-effect.db"))
			if _, err := svc.CreateGame(threeSiloRequest(73)); err != nil {
				t.Fatalf("create game: %v", err)
			}
			def, _ := stationOperationDefinition(tc.actionID)
			before := snapshotHistoryMetrics(svc.stationSilo())
			result := svc.scheduleStationOperation(model.AgentAction{Type: model.ActionStationOperation, ActionID: tc.actionID, Cost: float64(def.APCost)})
			if !result.Executed {
				t.Fatalf("schedule: %s", result.Message)
			}
			svc.world.StationOps.Operations[0].DueYear = svc.world.CurrentYear
			svc.world.StationOps.Operations[0].DueMonth = svc.world.CurrentMonth
			var logs []string
			var stories []model.StoryEvent
			svc.resolveStationOperations(&logs, &stories)
			op := svc.world.StationOps.Operations[0]
			if op.Status != "RESOLVED" {
				t.Fatalf("operation failed: %+v", op)
			}
			tc.verify(t, svc, op.Resolution, before)
		})
	}
}

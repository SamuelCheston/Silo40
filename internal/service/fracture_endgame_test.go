package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

func TestFractureEndingPlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testFractureEndingPlayableFromNewGame)
}

func testFractureEndingPlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "fracture-playable.db"))
	req := threeSiloArchetypeRequest(2601, archetype)
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
	}
	execute(model.AgentAction{Type: model.ActionConductPropaganda})

	deceives := 0
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		deceive := actionByID(svc.availablePlayerActions(), StationOpDeceive)
		if deceive != nil && deceive.Enabled && svc.agent.ActionPoints >= float64(deceive.APCost) {
			execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive})
			deceives++
			t.Logf("deceive %d at %d-%02d: rebellion=%.3f energy=%.0f population=%d suspicion=%.3f",
				deceives, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				siloResourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.SuspicionLevel)
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance FRACTURE line at month %d: %v", month, err)
		}
	}
	if deceives == 0 {
		t.Fatalf("real gates never allowed STATION_DECEIVE at %d-%02d: status=%+v prestige=%.3f AP=%.3f connections=%+v actions=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus, engine.DerivedPrestige(svc.agent.Ambition, svc.agent.Influence), svc.agent.ActionPoints, svc.agent.Connections, svc.availablePlayerActions())
	}
	for _, key := range []string{
		"crisis:fracture_infrastructure",
		"crisis:fracture_point_of_no_return",
		"special:fracture_confirmed",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire through real play", key)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingFracture {
		t.Fatalf("real route did not settle as FRACTURE at %d-%02d after %d deceives: status=%+v rebellion=%.3f energy=%.0f population=%d suspicion=%.3f",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, deceives, svc.silo.VictoryStatus,
			svc.silo.Rebellion, siloResourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.SuspicionLevel)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal FRACTURE state accepted another action: %+v", outcome.Result)
	}
}

func TestFractureRequiresActualSustenanceCollapse(t *testing.T) {
	boot := bootstrapRealContent(t)
	var defs []engine.ContentEventDefinition
	for _, def := range boot.contentDefinitions {
		switch def.Key {
		case "crisis:fracture_infrastructure", "crisis:fracture_point_of_no_return", "special:fracture_confirmed":
			defs = append(defs, def)
		}
	}
	if len(defs) != 3 {
		t.Fatalf("loaded %d FRACTURE definitions, want 3", len(defs))
	}

	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 124, 1
	svc.silo.Rebellion = 0.95
	svc.silo.TotalPopulation = 8000
	svc.silo.Resources = []model.Resource{{Type: "Energy", Amount: 1000}}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "crisis:fracture_infrastructure") {
		t.Fatal("high rebellion did not fire the infrastructure damage node")
	}
	if got := siloResourceAmount(svc.silo, "Energy"); got != 600 {
		t.Fatalf("infrastructure damage left Energy %.0f, want 600", got)
	}
	if contentTriggered(svc, "crisis:fracture_point_of_no_return") || contentTriggered(svc, fractureFactKey) {
		t.Fatal("high rebellion alone confirmed FRACTURE while Energy and population remained recoverable")
	}

	svc.silo.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: endingRebellion}
	decision := svc.resolveWeightedEnding(svc.silo, svc.agent)
	if decision == nil || decision.Selected != "" {
		t.Fatalf("recoverable high rebellion should continue simulation, got %+v", decision)
	}
}

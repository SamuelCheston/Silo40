package service

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"silo40/internal/model"
)

func weightedEndingFixture(trigger string) (*GameService, *model.Silo, *model.Agent) {
	station := &model.Silo{ID: 1, Name: "10 号地堡", VictoryStatus: &model.VictoryStatus{IsWon: true, Type: trigger}}
	peer := &model.Silo{ID: 2, Name: "1 号地堡", Cohesion: 0.8, Rebellion: 0.1, Traits: []string{silo1FullTrait}}
	agent := &model.Agent{ID: 1, Name: "Operative", SiloID: 1}
	svc := &GameService{
		world: &model.World{PlayerSiloID: 1, Silo1: model.Silo1State{Alive: true}},
		silo:  station, silos: []*model.Silo{station, peer}, agent: agent,
	}
	return svc, station, agent
}

func TestWeightedEndingRecordsAllEntityKinds(t *testing.T) {
	svc, station, _ := weightedEndingFixture("TIME")
	station.Factions = []model.Faction{{
		ID: 2, Name: "Open Circle", Signature: "pro_foreign:High", Influence: 1, Cohesion: 1,
	}}
	station.Residents = []model.Resident{{
		ID: 3, Name: "Named Resident", Alive: true, Influence: 1,
		Ideologies: map[string]float64{model.IdeologyProForeign: 1},
	}}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil {
		t.Fatal("expected a weighted decision")
	}
	kinds := map[string]bool{}
	var total float64
	for _, contribution := range decision.Contributions {
		kinds[contribution.Kind] = true
		for _, outcome := range endingOutcomeOrder {
			total += contribution.Scores[outcome]
		}
	}
	for _, kind := range []string{"player", "faction", "resident", "silo1"} {
		if !kinds[kind] {
			t.Fatalf("missing %s contribution: %+v", kind, decision.Contributions)
		}
	}
	for _, outcome := range endingOutcomeOrder {
		var outcomeTotal float64
		for _, contribution := range decision.Contributions {
			outcomeTotal += contribution.Scores[outcome]
		}
		if math.Abs(outcomeTotal-decision.Scores[outcome]) > 1e-12 {
			t.Fatalf("score audit mismatch for %s: contributions=%v total=%v", outcome, outcomeTotal, decision.Scores[outcome])
		}
	}
	if total == 0 || decision.Selected == "" {
		t.Fatalf("empty decision: %+v", decision)
	}
}

func TestWeightedEndingFactionsAndNamedResidentsCanOverridePlayer(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	baseline := svc.resolveWeightedEnding(station, svc.agent)
	station.Factions = []model.Faction{{ID: 4, Name: "Revolutionaries", Influence: 20, Cohesion: 0}}
	withFaction := svc.resolveWeightedEnding(station, svc.agent)
	if withFaction.Selected != "REBELLION" || withFaction.Selected == baseline.Selected {
		t.Fatalf("faction weight did not change the outcome: baseline=%s faction=%s", baseline.Selected, withFaction.Selected)
	}

	svc2, station2, _ := weightedEndingFixture("INFORMATION")
	station2.Residents = []model.Resident{{
		ID: 5, Name: "Ambitious Resident", Alive: true, Influence: 50, Ambition: 1,
		Ideologies: map[string]float64{},
	}}
	withResident := svc2.resolveWeightedEnding(station2, svc2.agent)
	if withResident.Selected != "REBELLION" {
		t.Fatalf("named resident weight did not affect the outcome: %+v", withResident)
	}
}

func TestWeightedEndingUsesSilo1AndDeterministicTieOrder(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	svc.world.Silo1.Alive = false
	first := svc.resolveWeightedEnding(station, svc.agent)
	second := svc.resolveWeightedEnding(station, svc.agent)
	if first.Selected != "TIME" {
		t.Fatalf("Silo 1 destruction did not contribute to TIME: %+v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical entities produced different decisions: %+v vs %+v", first, second)
	}
}

func TestWeightedEndingAcceptsLowercaseEngineTrigger(t *testing.T) {
	svc, station, _ := weightedEndingFixture("time")
	if decision := svc.resolveWeightedEnding(station, svc.agent); decision == nil {
		t.Fatal("lowercase JS victory trigger was not arbitrated")
	}
}

func TestEndingDecisionSurvivesSaveResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "weighted-ending.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(103)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	decision := &model.EndingDecision{
		ConfigVersion: 1, Selected: "TIME", MinimumScore: 1,
		Scores:        map[string]float64{"TIME": 4},
		Contributions: []model.EndingEntityContribution{{EntityID: "silo:1", Kind: "silo1", Name: "1 号地堡", Scores: map[string]float64{"TIME": 4}}},
	}
	svc.silo.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "TIME", Decision: decision}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if fresh.silo.VictoryStatus == nil || !reflect.DeepEqual(fresh.silo.VictoryStatus.Decision, decision) {
		t.Fatalf("ending decision was not restored: %+v", fresh.silo.VictoryStatus)
	}
}

func TestWeightedEndingPreservesCatastrophicFailure(t *testing.T) {
	svc, station, _ := weightedEndingFixture("DEATH")
	station.VictoryStatus.IsWon = false
	if decision := svc.resolveWeightedEnding(station, svc.agent); decision != nil {
		t.Fatalf("hard failure must not enter weighted arbitration: %+v", decision)
	}
}

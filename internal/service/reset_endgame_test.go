package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// progressResetCountermeasureForRoute lets playable tests for other endings
// respond to the newly reachable RESET chain through the same real action
// gates as a player. It never writes completion facts directly.
func progressResetCountermeasureForRoute(t *testing.T, svc *GameService, execute func(model.AgentAction)) {
	t.Helper()
	if svc.gameOver() || contentTriggered(svc, "player_action:reset_countermeasure_done") ||
		!contentTriggered(svc, "crisis:reset_omen_requisition") {
		return
	}
	if !contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") {
		offer := actionByID(svc.availablePlayerActions(), "RESET_COUNTERMEASURE_OFFER")
		if offer == nil || !offer.Enabled {
			return
		}
		execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RESET_COUNTERMEASURE_OFFER"})
	}
	if contentTriggered(svc, "player_action:reset_countermeasure_done") ||
		svc.agent.ActionPoints < model.ACTION_COSTS[model.ActionShareInfo] {
		return
	}
	for _, dept := range svc.silo.Professions {
		fragment, ok := strongestShareableFactFor(svc.agent, departmentByAddress(svc.silo, dept.Name))
		if !ok {
			continue
		}
		execute(model.AgentAction{
			Type: model.ActionShareInfo, TargetDept: dept.Name,
			FragmentIds: []string{fragment},
		})
		return
	}
}

func TestResetEndingPlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testResetEndingPlayableFromNewGame)
}

func testResetEndingPlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "reset-playable.db"))
	if _, err := svc.CreateGame(threeSiloArchetypeRequest(2701, archetype)); err != nil {
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

	for month := 1; month <= 48 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance RESET line at month %d: %v", month, err)
		}
	}

	for _, key := range []string{
		"special:reset_omen_census",
		"crisis:reset_omen_requisition",
		"crisis:reset_execution",
		resetFactKey,
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire through real play", key)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingReset {
		t.Fatalf("real route did not settle as RESET at %d-%02d: status=%+v history=%.3f population=%d",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus,
			svc.silo.HistoryBurden, svc.silo.TotalPopulation)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal RESET state accepted another action: %+v", outcome.Result)
	}
}

func TestResetCountermeasurePlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testResetCountermeasurePlayableFromNewGame)
}

func testResetCountermeasurePlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "reset-countermeasure.db"))
	if _, err := svc.CreateGame(threeSiloArchetypeRequest(2702, archetype)); err != nil {
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

	for month := 1; month <= 24 && !contentTriggered(svc, "crisis:reset_omen_requisition") && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance to RESET countermeasure window at month %d: %v", month, err)
		}
	}
	if !contentTriggered(svc, "crisis:reset_omen_requisition") {
		t.Fatalf("requisition never fired at %d-%02d: history=%.3f status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.HistoryBurden, svc.silo.VictoryStatus)
	}
	offer := actionByID(svc.availablePlayerActions(), "RESET_COUNTERMEASURE_OFFER")
	if offer == nil || !offer.Enabled {
		t.Fatalf("countermeasure offer was not available through its real gate: %+v", offer)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RESET_COUNTERMEASURE_OFFER"})
	if !contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") {
		t.Fatal("accepting the offer did not record its real completion key")
	}
	if contentTriggered(svc, "special:copies_distributed") || contentTriggered(svc, "player_action:reset_countermeasure_done") {
		t.Fatal("one pre-offer share incorrectly completed the two-share countermeasure")
	}

	var shared bool
	for _, dept := range svc.silo.Professions {
		fragment, ok := strongestShareableFactFor(svc.agent, departmentByAddress(svc.silo, dept.Name))
		if !ok {
			continue
		}
		execute(model.AgentAction{
			Type: model.ActionShareInfo, TargetDept: dept.Name,
			FragmentIds: []string{fragment},
		})
		shared = true
		break
	}
	if !shared {
		t.Fatalf("no real second share was possible: known=%v beliefs=%+v AP=%.2f",
			svc.agent.KnownFragments, svc.agent.Beliefs, svc.agent.ActionPoints)
	}
	for _, key := range []string{"special:copies_distributed", "player_action:reset_countermeasure_done"} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s did not fire after the second real Share", key)
		}
	}

	for month := 1; month <= 12 && !contentTriggered(svc, "crisis:reset_execution") && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance guarded RESET execution at month %d: %v", month, err)
		}
	}
	if !contentTriggered(svc, "crisis:reset_execution") {
		t.Fatal("scheduled RESET execution never fired")
	}
	if contentTriggered(svc, resetFactKey) {
		t.Fatal("guarded RESET execution emitted reset_completed instead of idling")
	}
	if svc.silo.VictoryStatus != nil && svc.silo.VictoryStatus.Decision != nil &&
		svc.silo.VictoryStatus.Decision.Selected == endingReset {
		t.Fatalf("completed countermeasure still settled as RESET: %+v", svc.silo.VictoryStatus)
	}
}

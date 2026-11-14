package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// chainFixtureDef builds a normalized-shape content definition (lowercase
// trigger/effect types, Key set) the way the loader would produce it.
func chainFixtureDef(group, eventID string, trigger engine.ContentTrigger, effects []engine.ContentEffect) engine.ContentEventDefinition {
	def := engine.ContentEventDefinition{
		EventID:     eventID,
		SourceGroup: group,
		Key:         group + ":" + eventID,
		Title:       eventID,
		FireMode:    engine.ContentFireModeOnce,
		Trigger:     trigger,
		Effects:     effects,
	}
	return def
}

// chainTestService wires a minimal session: one silo, one agent, custom
// content definitions and the real bus subscriptions.
func chainTestService(defs ...engine.ContentEventDefinition) *GameService {
	svc := &GameService{
		engine:        engine.NewGameEngine(),
		contentStates: map[uint]map[string]model.ContentEventState{},
		delayTasks:    map[string]*delayTask{},
	}
	svc.silo = &model.Silo{ID: 1, Name: "Chain Fixture", CurrentYear: 105, CurrentMonth: 1}
	svc.agent = &model.Agent{ID: 1, Name: "Operative", SiloID: 1}
	svc.silos = []*model.Silo{svc.silo}
	svc.agents = map[uint]*model.Agent{1: svc.agent}
	svc.contentDefinitions = defs
	svc.bindContentEventHandlers()
	return svc
}

func contentTriggered(svc *GameService, key string) bool {
	states := svc.contentStates[svc.silo.ID]
	if states == nil {
		return false
	}
	return states[key].Triggered
}

// TestCompletionKeyEmitsBusEventForFollowups locks 硬伤 #1: built-in action
// completion keys publish their bus event, so event_triggered follow-ups fire
// in the same drain instead of never firing at all.
func TestCompletionKeyEmitsBusEventForFollowups(t *testing.T) {
	followup := chainFixtureDef("special", "mayor_evidence_step",
		engine.ContentTrigger{Type: "event_triggered", Category: "player_action", EventID: "SHARE_INFO_STEP"},
		[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "cohesion", Value: -0.05}},
	)
	svc := chainTestService(followup)
	svc.silo.Cohesion = 0.5 // negative delta must survive clampUnit
	before := svc.silo.Cohesion

	svc.completePlayerAction(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "SHARE_INFO_STEP"})
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "player_action:SHARE_INFO_STEP") {
		t.Fatal("completion key was not recorded")
	}
	if !contentTriggered(svc, "special:mayor_evidence_step") {
		t.Fatal("event_triggered follow-up never fired after the completion key")
	}
	if svc.silo.Cohesion != before-0.05 {
		t.Fatalf("follow-up effect not applied: cohesion %v, want %v", svc.silo.Cohesion, before-0.05)
	}
}

// TestMixedAnyBranchSurvivesMonthlySweep locks 硬伤 #2: an any() combo that
// mixes event_triggered with numeric branches is no longer excluded wholesale
// from the monthly channel — the numeric branch ignites on its own.
func TestMixedAnyBranchSurvivesMonthlySweep(t *testing.T) {
	crisis := chainFixtureDef("special", "mayor_crisis_stage",
		engine.ContentTrigger{Type: "any", Conditions: []engine.ContentTrigger{
			{Type: "event_triggered", Category: "crisis", EventID: "food_poisoning"},
			{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.3},
		}},
		[]engine.ContentEffect{{Type: "profession_metric_delta_all", Metric: "panic_value", Value: 0.03}},
	)
	svc := chainTestService(crisis)
	svc.silo.Rebellion = 0.4

	if count := svc.queueSystemContentEvents("TEST"); count == 0 {
		t.Fatal("numeric any() branch was dropped with its event_triggered sibling")
	}
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:mayor_crisis_stage") {
		t.Fatal("mixed any() event never fired via the monthly sweep")
	}
}

// TestEventTriggeredChainStaysContinuouslyEvaluated locks the other half of
// 硬伤 #2/#3: a pure event_triggered follower whose parent fired earlier is
// still picked up by the monthly sweep (continuous evaluation), not only in
// the instant its parent emits.
func TestEventTriggeredChainStaysContinuouslyEvaluated(t *testing.T) {
	followup := chainFixtureDef("special", "charter_draft_convened",
		engine.ContentTrigger{Type: "event_triggered", Category: "player_action", EventID: "charter_draft_step"},
		[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "history_burden", Value: 0.03}},
	)
	svc := chainTestService(followup)
	// Parent completed earlier; no bus event accompanies this sweep.
	svc.statesMapFor(svc.silo.ID)["player_action:charter_draft_step"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "player_action:charter_draft_step", Triggered: true,
	}

	if count := svc.queueSystemContentEvents("TEST"); count == 0 {
		t.Fatal("event_triggered follower was excluded from the monthly sweep")
	}
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:charter_draft_convened") {
		t.Fatal("continuous evaluation did not pick up the earlier parent completion")
	}
}

// TestDeadlineForcesLateWindowEvent locks 硬伤 #3: an event whose precondition
// completed before its lock time-point still fires (continuous evaluation),
// and an event still untriggered at the deadline is force-raised once.
func TestDeadlineForcesLateWindowEvent(t *testing.T) {
	newLockDay := func() engine.ContentEventDefinition {
		def := chainFixtureDef("special", "charter_lock_day",
			engine.ContentTrigger{Type: "all", Conditions: []engine.ContentTrigger{
				{Type: "event_triggered", Category: "special", EventID: "charter_draft_convened"},
				{Type: "timestamp_at_or_after", Year: 106, Month: 1},
			}},
			[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "rebellion", Value: -0.04}},
		)
		def.DeadlineYear = 106
		def.DeadlineMonth = 1
		return def
	}

	// A: drafting completed before the lock time-point; at the time-point the
	// monthly sweep ignites the node through normal continuous evaluation.
	svc := chainTestService(newLockDay())
	svc.statesMapFor(svc.silo.ID)["special:charter_draft_convened"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:charter_draft_convened", Triggered: true,
	}
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 106, 2
	if count := svc.queueSystemContentEvents("TEST"); count == 0 {
		t.Fatal("lock day with an early-completed precondition must fire at the time-point")
	}
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:charter_lock_day") {
		t.Fatal("lock day with an early-completed precondition never fired")
	}

	// B: nothing triggered by the deadline — the fallback force-raises it.
	svc2 := chainTestService(newLockDay())
	svc2.silo.CurrentYear, svc2.silo.CurrentMonth = 106, 2
	if count := svc2.queueSystemContentEvents("TEST"); count == 0 {
		t.Fatal("deadline fallback must force-raise an untriggered window event")
	}
	svc2.drainQueuedEvents()
	if !contentTriggered(svc2, "special:charter_lock_day") {
		t.Fatal("deadline fallback never force-raised the window event")
	}

	// C: before the deadline with the precondition missing, nothing fires.
	svc3 := chainTestService(newLockDay())
	svc3.silo.CurrentYear, svc3.silo.CurrentMonth = 105, 6
	if count := svc3.queueSystemContentEvents("TEST"); count != 0 || contentTriggered(svc3, "special:charter_lock_day") {
		t.Fatal("lock day must not fire before its time-point without the precondition")
	}
}

// TestContentEventWritesPlayerCompletionKey locks 硬伤 #4: content events can
// write player completion keys (player_action_complete), and the written key
// wakes event_triggered follow-ups like a built-in completion.
func TestContentEventWritesPlayerCompletionKey(t *testing.T) {
	writer := chainFixtureDef("special", "copies_distributed",
		engine.ContentTrigger{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.5},
		[]engine.ContentEffect{{Type: "player_action_complete", Action: "reset_countermeasure_done"}},
	)
	guarded := chainFixtureDef("crisis", "reset_execution_settled",
		engine.ContentTrigger{Type: "event_triggered", Category: "player_action", EventID: "reset_countermeasure_done"},
		[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "cohesion", Value: 0.02}},
	)
	svc := chainTestService(writer, guarded)
	svc.silo.Rebellion = 0.6
	before := svc.silo.Cohesion

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "player_action:reset_countermeasure_done") {
		t.Fatal("player_action_complete effect did not write the completion key")
	}
	if !contentTriggered(svc, "crisis:reset_execution_settled") {
		t.Fatal("completion key written by content did not wake its event_triggered follow-up")
	}
	if svc.silo.Cohesion != before+0.02 {
		t.Fatalf("follow-up effect not applied: cohesion %v, want %v", svc.silo.Cohesion, before+0.02)
	}
}

// TestPlayerActionCompleteValidation locks the effect contract: the write path
// requires an action id, both in structured definitions and script results.
func TestPlayerActionCompleteValidation(t *testing.T) {
	def := chainFixtureDef("special", "bad_writer",
		engine.ContentTrigger{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.5},
		[]engine.ContentEffect{{Type: "player_action_complete"}},
	)
	if err := engine.ValidateContentEventDefinition(def); err == nil {
		t.Fatal("player_action_complete without action must fail validation")
	}
	def.Effects[0].Action = "reset_countermeasure_done"
	if err := engine.ValidateContentEventDefinition(def); err != nil {
		t.Fatalf("player_action_complete with action must validate: %v", err)
	}
}

// TestDeadlineValidation locks the deadline field contract.
func TestDeadlineValidation(t *testing.T) {
	def := chainFixtureDef("special", "deadline_fixture",
		engine.ContentTrigger{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.5},
		[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "rebellion", Value: -0.01}},
	)
	def.DeadlineYear = 106
	def.DeadlineMonth = 13
	if err := engine.ValidateContentEventDefinition(def); err == nil {
		t.Fatal("deadline month out of range must fail validation")
	}
	def.DeadlineMonth = 1
	if err := engine.ValidateContentEventDefinition(def); err != nil {
		t.Fatalf("valid deadline must pass validation: %v", err)
	}
}

// TestDeadlineAndCompletionHelperLoadFromJS locks the content-author contract
// end to end: deadline fields normalize (month defaults to 1) and the
// completePlayerAction() prelude helper produces a valid effect.
func TestDeadlineAndCompletionHelperLoadFromJS(t *testing.T) {
	dir := t.TempDir()
	source := `defineEvent({
  id: "charter_lock_day",
  title: "锁定日",
  type: "SPECIAL",
  fire_mode: "ONCE",
  deadline_year: 106,
  trigger: siloMetricGte("rebellion", 0.5),
  effects: [ completePlayerAction("reset_countermeasure_done") ],
});`
	if err := os.WriteFile(filepath.Join(dir, "charter_lock_day.js"), []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	defs, err := engine.LoadContentEventDefinitions(map[string]string{"special": dir})
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected one definition, got %d", len(defs))
	}
	def := defs[0]
	if def.DeadlineYear != 106 || def.DeadlineMonth != 1 {
		t.Fatalf("deadline must normalize month to 1: %+v", def)
	}
	if len(def.Effects) != 1 || def.Effects[0].Type != "player_action_complete" || def.Effects[0].Action != "reset_countermeasure_done" {
		t.Fatalf("completePlayerAction helper produced %+v", def.Effects)
	}
}

// TestPlayerEventFeedbackCarriesEventStory locks 修复1: executing a
// PLAYER_EVENT action must return the fired event's title and narrative in the
// action receipt, not the opaque "Triggered player action event." The story
// itself only lands in history after the drain, so the receipt text is built
// from the definition to stay readable at execution time.
func TestPlayerEventFeedbackCarriesEventStory(t *testing.T) {
	def := chainFixtureDef("player_actions", "receiver_verify_accompany",
		engine.ContentTrigger{Type: "player_action_is", Action: "RECEIVER_VERIFY_ACCOMPANY"},
		[]engine.ContentEffect{{Type: "silo_metric_delta", Metric: "history_burden", Value: 0.01}},
	)
	def.Title = "陪同回查"
	def.Description = "你陪第一位接收者把材料重新查了一遍：来源、缺口、可以核验的部分。"
	def.PlayerAction = &engine.ContentPlayerActionSpec{
		ID:          "RECEIVER_VERIFY_ACCOMPANY",
		Label:       "陪同回查",
		ActionType:  string(model.ActionPlayerEvent),
		TargetType:  "NONE",
		Scope:       "common",
		APCost:      2,
	}
	svc := chainTestService(def)
	svc.agent.ActionPoints = 10

	handled, result, err := svc.queuePlayerActionEvents(model.AgentAction{
		Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY", Cost: 2,
	})
	if err != nil || !handled {
		t.Fatalf("player event action not handled: handled=%v err=%v", handled, err)
	}
	if !result.Executed {
		t.Fatalf("player event action not executed: %+v", result)
	}
	if !strings.Contains(result.Message, "陪同回查") {
		t.Fatalf("receipt lost the event title: %q", result.Message)
	}
	if !strings.Contains(result.Message, "重新查了一遍") {
		t.Fatalf("receipt lost the event narrative: %q", result.Message)
	}
	if strings.Contains(result.Message, "Triggered player action event") {
		t.Fatalf("receipt fell back to system jargon: %q", result.Message)
	}
}

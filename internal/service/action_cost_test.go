package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// newActionCostTestService builds a minimal in-memory session: a real engine
// (JS mechanics loaded from the repo), an initialized silo and agent, no DB.
func newActionCostTestService(t *testing.T, profession string) *GameService {
	t.Helper()
	svc := &GameService{
		engine:        engine.NewGameEngine(),
		contentStates: map[uint]map[string]model.ContentEventState{},
		delayTasks:    map[string]*delayTask{},
	}
	svc.silo = engine.CreateInitialSilo("Cost Test", 122, nil)
	svc.agent = engine.CreateInitialAgent("CostTester", profession, nil, svc.silo)
	return svc
}

// Server-authoritative cost: the client-submitted cost must be replaced by
// the definition cost, so cost=0 can no longer execute for free.
func TestResolveActionCostUsesServerDefinitions(t *testing.T) {
	svc := newActionCostTestService(t, "Mechanical")

	cases := []struct {
		action model.AgentAction
		want   float64
	}{
		{model.AgentAction{Type: model.ActionGatherInfo, Cost: 0}, model.ACTION_COSTS[model.ActionGatherInfo]},
		{model.AgentAction{Type: model.ActionShareInfo, Cost: 999}, model.ACTION_COSTS[model.ActionShareInfo]},
		{model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL", Cost: 0}, 15},
		{model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MECHANICAL_PIPE_TAP", Cost: 1}, 15},
	}
	for _, tc := range cases {
		action := tc.action
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolveActionCost(%+v): %v", tc.action, err)
		}
		if action.Cost != tc.want {
			t.Errorf("resolveActionCost(%+v) = %v, want %v", tc.action, action.Cost, tc.want)
		}
	}

	if err := svc.resolveActionCost(&model.AgentAction{Type: model.ActionProfession, ProfessionAction: "NO_SUCH_ACTION"}); err == nil {
		t.Errorf("expected error for unknown profession action")
	}
	if err := svc.resolveActionCost(&model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "NO_SUCH_ACTION"}); err == nil {
		t.Errorf("expected error for unknown player action")
	}
}

// The charged cost must equal the PlayerActionMeta value the UI displayed:
// both come from the same mechanic-engine definitions.
func TestResolvedCostMatchesPlayerActionMeta(t *testing.T) {
	svc := newActionCostTestService(t, "Mechanical")
	metas := svc.availablePlayerActions()
	if len(metas) == 0 {
		t.Fatalf("expected available player actions")
	}
	for _, meta := range metas {
		if meta.Source != "static" {
			continue
		}
		action := model.AgentAction{Type: meta.ActionType, Cost: 0}
		if meta.ActionType == model.ActionProfession {
			action.ProfessionAction = meta.ID
		}
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolveActionCost for meta %s: %v", meta.ID, err)
		}
		if action.Cost != float64(meta.APCost) {
			t.Errorf("meta %s: resolved cost %v != meta ap_cost %d", meta.ID, action.Cost, meta.APCost)
		}
	}
}

// End to end: submitting cost=0 must still charge the full definition cost.
func TestExecuteActionChargesServerCostDespiteClientZero(t *testing.T) {
	svc := newActionCostTestService(t, "Mechanical")
	before := svc.agent.ActionPoints

	outcome, err := svc.ExecuteAction(model.AgentAction{
		Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 0,
	})
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if !outcome.Result.Executed {
		t.Fatalf("expected action to execute, got %q", outcome.Result.Message)
	}
	want := model.ACTION_COSTS[model.ActionGatherInfo]
	if got := before - svc.agent.ActionPoints; got != want {
		t.Errorf("AP deduction = %v, want server cost %v (client sent 0)", got, want)
	}
}

// End to end via the queued path (BeginAction stores the action and executes
// it later through ProcessNextEvent): the stored action must carry the
// server-resolved cost.
func TestBeginActionStoresServerCost(t *testing.T) {
	svc := newActionCostTestService(t, "Mechanical")

	if _, err := svc.BeginAction(model.AgentAction{
		Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 0,
	}); err != nil {
		t.Fatalf("BeginAction: %v", err)
	}
	if svc.pendingOperation == nil || svc.pendingOperation.Action == nil {
		t.Fatalf("expected pending action operation")
	}
	if got := svc.pendingOperation.Action.Cost; got != model.ACTION_COSTS[model.ActionGatherInfo] {
		t.Errorf("pending action cost = %v, want %v", got, model.ACTION_COSTS[model.ActionGatherInfo])
	}
}

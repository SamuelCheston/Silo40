package service

import (
	"path/filepath"
	"reflect"
	"testing"

	"silo40/internal/model"
)

func TestCompromisedEndingPlayableFromNewGame(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "compromised-playable.db"))
	req := threeSiloRequest(2801)
	req.Profession = "Mayor"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}

	omenKeys := []string{
		"special:identity_trace_accumulates",
		"special:identity_query_local",
		"special:archive_review_opened",
		"special:agent_erasure_staged",
	}
	seen := make(map[string]bool, len(omenKeys))
	var omenOrder []string
	observeOmens := func() {
		t.Helper()
		states := svc.contentStates[svc.silo.ID]
		for _, key := range omenKeys {
			if seen[key] || !states[key].Triggered {
				continue
			}
			seen[key] = true
			omenOrder = append(omenOrder, key)
			t.Logf("omen %s at %d-%02d: traces=%d suspicion=%.3f",
				key, svc.silo.CurrentYear, svc.silo.CurrentMonth,
				states["special:identity_trace_accumulates"].TriggerCount, svc.agent.SuspicionLevel)
		}
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
		observeOmens()
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mechanical"})
	}
	execute(model.AgentAction{Type: model.ActionConductPropaganda})

	infiltrations, suspicionBuilds := 0, 0
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}

		actions := svc.availablePlayerActions()
		if infiltrations == 0 {
			infiltrate := actionByID(actions, StationOpInfiltrate)
			if infiltrate != nil && infiltrate.Enabled && svc.agent.ActionPoints >= float64(infiltrate.APCost) {
				execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpInfiltrate})
				infiltrations++
				t.Logf("infiltrate at %d-%02d: suspicion=%.3f AP=%.1f legitimacy=%.3f cohesion=%.3f rebellion=%.3f",
					svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.agent.SuspicionLevel,
					svc.agent.ActionPoints, svc.silo.MayorLegitimacy(), svc.silo.Cohesion, svc.silo.Rebellion)
				continue
			}
		} else {
			build := actionByID(actions, "action_build_connection")
			if build != nil && build.Enabled && svc.agent.ActionPoints >= float64(build.APCost) {
				execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
				suspicionBuilds++
				if suspicionBuilds%10 == 0 {
					t.Logf("connection build %d at %d-%02d: suspicion=%.3f AP=%.1f",
						suspicionBuilds, svc.silo.CurrentYear, svc.silo.CurrentMonth,
						svc.agent.SuspicionLevel, svc.agent.ActionPoints)
				}
				continue
			}
		}

		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance AGENT_COMPROMISED line at month %d: %v", month, err)
		}
		observeOmens()
	}

	if infiltrations == 0 || suspicionBuilds == 0 {
		t.Fatalf("real gates did not execute the covert probe and sustained relation building: infiltrations=%d builds=%d status=%+v AP=%.3f connections=%+v actions=%+v",
			infiltrations, suspicionBuilds, svc.silo.VictoryStatus, svc.agent.ActionPoints,
			svc.agent.Connections, svc.availablePlayerActions())
	}
	if !reflect.DeepEqual(omenOrder, omenKeys) {
		t.Errorf("omen order = %v, want %v", omenOrder, omenKeys)
	}
	if traces := svc.contentStates[svc.silo.ID]["special:identity_trace_accumulates"].TriggerCount; traces < 5 {
		t.Errorf("identity traces accumulated %d times, want >= 5", traces)
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingCompromised {
		t.Fatalf("real route did not settle as AGENT_COMPROMISED at %d-%02d after %d infiltration and %d connection builds: status=%+v suspicion=%.3f legitimacy=%.3f cohesion=%.3f rebellion=%.3f omens=%v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, infiltrations, suspicionBuilds,
			svc.silo.VictoryStatus, svc.agent.SuspicionLevel, svc.silo.MayorLegitimacy(),
			svc.silo.Cohesion, svc.silo.Rebellion, omenOrder)
	}
	if svc.agent.SuspicionLevel < 1.0 {
		t.Fatalf("AGENT_COMPROMISED settled below the victory_check suspicion cap: %.3f", svc.agent.SuspicionLevel)
	}
	t.Logf("real AGENT_COMPROMISED route settled at %d-%02d: suspicion=%.3f infiltrations=%d connection_builds=%d traces=%d",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.agent.SuspicionLevel,
		infiltrations, suspicionBuilds, svc.contentStates[svc.silo.ID]["special:identity_trace_accumulates"].TriggerCount)
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal AGENT_COMPROMISED state accepted another action: %+v", outcome.Result)
	}
}

package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// reactionChainKeys lists the Silo 1 post-ultimatum reaction chain, grouped by
// independent enforcement channel (本地执行 / 远程终止 / 远程物理打击).
var reactionChainKeys = []string{
	// Local execution channel.
	"special:silo1_purge_omen",
	"special:silo1_purge_exposed",
	"crisis:silo1_purge_execution",
	// Remote termination channel.
	"special:termination_channel_test",
	"special:termination_channel_disrupted",
	"crisis:termination_release",
	// Remote physical strike channel.
	"special:strike_reconnaissance",
	"crisis:bombing_terminal",
}

func findContentDefinitionByKey(t *testing.T, svc *GameService, key string) engine.ContentEventDefinition {
	t.Helper()
	for _, def := range svc.contentDefinitions {
		if def.Key == key {
			return def
		}
	}
	t.Fatalf("content event %q was not loaded", key)
	return engine.ContentEventDefinition{}
}

// TestSilo1ReactionChainEventsLoad locks that every reaction-chain node loads
// and passes content validation (a malformed trigger/effect would fail boot).
func TestSilo1ReactionChainEventsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	have := map[string]bool{}
	for _, def := range svc.contentDefinitions {
		have[def.Key] = true
	}
	for _, key := range reactionChainKeys {
		if !have[key] {
			t.Fatalf("reaction chain event %q not loaded", key)
		}
	}
}

// TestReactionSettlementsAreNonNumericalHooks locks the ruling that settlements
// write no population figures and no safeguard-risk deltas: all three settlement
// nodes are apply-hooks with no population/risk effects. The remote-termination
// and remote-strike hooks connect to the ending purely through their completion
// keys; the local purge is a no-op TODO hook.
func TestReactionSettlementsAreNonNumericalHooks(t *testing.T) {
	svc := bootstrapRealContent(t)
	for _, key := range []string{
		"crisis:bombing_terminal",
		"crisis:termination_release",
		"crisis:silo1_purge_execution",
	} {
		def := findContentDefinitionByKey(t, svc, key)
		if !def.ScriptHooks.Apply {
			t.Fatalf("%s must be an apply settlement hook", key)
		}
		for _, effect := range def.Effects {
			if effect.Type == "silo_population_delta" {
				t.Fatalf("%s must not reduce population", key)
			}
			if effect.Type == "silo_metric_delta" && effect.Metric == "safeguard_risk" {
				t.Fatalf("%s must not touch safeguard_risk", key)
			}
		}
	}
}

// TestReactionCountermeasureHooksAreDeclaredButUnbound locks that the
// countermeasure windows are declared but NOT bound: Share Info and propaganda
// are not countermeasures, so even with the omen fired and those actions
// completed the hooks must not trigger this round.
func TestReactionCountermeasureHooksAreDeclaredButUnbound(t *testing.T) {
	svc := bootstrapRealContent(t)
	for _, key := range []string{
		"special:termination_channel_disrupted",
		"special:silo1_purge_exposed",
	} {
		def := findContentDefinitionByKey(t, svc, key)
		if !def.ScriptHooks.CanTrigger {
			t.Fatalf("%s must declare a canTrigger hook (declared-but-unbound)", key)
		}
		silo := &model.Silo{ID: 1, CurrentYear: 127, CurrentMonth: 1}
		ok, err := engine.RunContentEventCanTriggerScript(def, engine.ContentEvaluationContext{
			Silo: silo,
			States: map[string]engine.ContentEventRuntime{
				"special:termination_channel_test":        {Triggered: true},
				"special:silo1_purge_omen":                {Triggered: true},
				"player_action:action_share_info":         {Triggered: true},
				"player_action:action_conduct_propaganda": {Triggered: true},
			},
		}, false)
		if err != nil {
			t.Fatalf("%s canTrigger: %v", key, err)
		}
		if ok {
			t.Fatalf("%s must stay unbound (canTrigger false) this round", key)
		}
	}
}

// TestReactionSettlementFactKeysArbitrateDeath locks the ending wiring: a fired
// remote-termination or remote-strike settlement is a DEATH fact in arbitration,
// and a surviving escape group still converts it into the REBELLION pyrrhic
// epilogue (P0 escape exception).
func TestReactionSettlementFactKeysArbitrateDeath(t *testing.T) {
	for _, factKey := range []string{terminationFactKey, strikeFactKey} {
		svc, station, _ := weightedEndingFixture("TIME")
		svc.contentStates = map[uint]map[string]model.ContentEventState{
			station.ID: {factKey: {Triggered: true}},
		}
		decision := svc.resolveWeightedEnding(station, svc.agent)
		if decision == nil || decision.Selected != "DEATH" {
			t.Fatalf("settlement fact %s must arbitrate DEATH, got %+v", factKey, decision)
		}

		// With a surviving escape group the same fact becomes a pyrrhic REBELLION.
		svc2, station2, _ := weightedEndingFixture("TIME")
		station2.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 120, Surviving: true}
		svc2.contentStates = map[uint]map[string]model.ContentEventState{
			station2.ID: {factKey: {Triggered: true}},
		}
		pyrrhic := svc2.resolveWeightedEnding(station2, svc2.agent)
		if pyrrhic == nil || pyrrhic.Selected != "REBELLION" {
			t.Fatalf("settlement fact %s with escape group must avert DEATH into REBELLION, got %+v", factKey, pyrrhic)
		}
	}
}

// TestLocalPurgeSettlementIsNotADeathFact locks that the local purge settlement
// is deliberately NOT wired into the DEATH facts this round (no modeled
// consequence yet — it must not end the game on its own).
func TestLocalPurgeSettlementIsNotADeathFact(t *testing.T) {
	svc, station, _ := weightedEndingFixture("TIME")
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {"crisis:silo1_purge_execution": {Triggered: true}},
	}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision != nil && decision.Selected == "DEATH" {
		t.Fatalf("local purge settlement must not arbitrate DEATH this round, got %+v", decision)
	}
}

// TestDeathPhysicalStrikePlayableFromNewGame drives the complete Silo 1 physical
// termination channel through public game operations. Deception raises real
// rebellion; Supply rationing keeps FRACTURE from winning the race before the
// scheduled strike lands. No metric, content fact, escape group, ending status,
// or delay task is seeded by the test.
func TestDeathPhysicalStrikePlayableFromNewGame(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "death-physical-strike.db"))
	req := threeSiloRequest(2101)
	req.Profession = "Supply"
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

	observed := map[string]bool{}
	observedAt := map[string]int{}
	observe := func() {
		t.Helper()
		for _, key := range []string{
			"histories:silo1_ultimatum",
			"special:strike_reconnaissance",
			"crisis:bombing_terminal",
		} {
			if observed[key] || !contentTriggered(svc, key) {
				continue
			}
			observed[key] = true
			observedAt[key] = gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth)
			t.Logf("%s fired at %d-%02d: rebellion=%.3f energy=%.0f supplies=%.0f population=%d suspicion=%.3f",
				key, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), resourceAmount(svc.silo, "Supplies"),
				svc.silo.TotalPopulation, svc.agent.SuspicionLevel)
		}
	}

	deceivePending := func() bool {
		for _, op := range svc.world.StationOps.Operations {
			if op.ActionID == StationOpDeceive && op.Status == "PENDING" {
				return true
			}
		}
		return false
	}
	deceives, rations := 0, 0
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		observe()
		if svc.gameOver() {
			break
		}

		buildupOpen := svc.silo.CurrentYear >= 128
		buildingRebellion := buildupOpen && svc.silo.Rebellion < 0.82
		pending := deceivePending()
		deceive := actionByID(svc.availablePlayerActions(), StationOpDeceive)
		energyGuard := svc.silo.Rebellion >= 0.6 && resourceAmount(svc.silo, "Energy") < 1500
		if buildingRebellion && deceives < 2 && !pending && !energyGuard && deceive != nil && deceive.Enabled &&
			svc.agent.ActionPoints >= float64(deceive.APCost+15) {
			execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive})
			deceives++
			t.Logf("deceive %d at %d-%02d: rebellion=%.3f energy=%.0f population=%d AP=%.1f suspicion=%.3f",
				deceives, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.ActionPoints, svc.agent.SuspicionLevel)
			observe()
			continue
		}

		// Once the repeatable underground-coordination node can snowball
		// rebellion, keep Energy positive so it cannot confirm FRACTURE.
		if energyGuard && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{
				Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Energy",
			})
			rations++
			t.Logf("ration %d at %d-%02d: rebellion=%.3f energy=%.0f supplies=%.0f materials=%.0f population=%d AP=%.1f",
				rations, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), resourceAmount(svc.silo, "Supplies"),
				resourceAmount(svc.silo, "Materials"), svc.silo.TotalPopulation, svc.agent.ActionPoints)
			observe()
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance DEATH physical strike at month %d: %v", month, err)
		}
		observe()
	}

	if deceives == 0 || rations == 0 {
		t.Fatalf("route did not use its real rebellion and survival levers: deceives=%d rations=%d", deceives, rations)
	}
	states := svc.contentStates[svc.silo.ID]
	for _, key := range []string{
		"histories:silo1_ultimatum",
		"special:strike_reconnaissance",
		"crisis:bombing_terminal",
	} {
		if !states[key].Triggered || states[key].TriggerCount != 1 {
			t.Fatalf("%s did not fire exactly once through real play: state=%+v status=%+v", key, states[key], svc.silo.VictoryStatus)
		}
	}
	ultimatum := states["histories:silo1_ultimatum"]
	strike := states["special:strike_reconnaissance"]
	bombing := states["crisis:bombing_terminal"]
	if ultimatum.LastTriggeredTimestamp >= strike.LastTriggeredTimestamp {
		t.Fatalf("physical strike did not follow the ultimatum: ultimatum=%+v strike=%+v", ultimatum, strike)
	}
	if bombing.LastTriggeredTimestamp <= strike.LastTriggeredTimestamp {
		t.Fatalf("bombing did not follow reconnaissance: strike=%+v bombing=%+v", strike, bombing)
	}
	// Reconnaissance is recorded by the post-clock content sweep, while its
	// scheduled target is dispatched before the third subsequent local phase.
	// Measure at completed public operations so both ends use the same clock
	// boundary and exactly three real monthly advances separate their observation.
	if got, want := observedAt["crisis:bombing_terminal"]-observedAt["special:strike_reconnaissance"], 3*daysPerMonth; got != want {
		t.Fatalf("publicly observed bombing delay = %d days, want %d: observed_at=%v strike=%+v bombing=%+v",
			got, want, observedAt, strike, bombing)
	}
	if contentTriggered(svc, fractureFactKey) {
		t.Fatal("FRACTURE settled before the physical strike could land")
	}
	if svc.silo.EscapedGroup.SurvivingEscapeGroup() {
		t.Fatalf("route unexpectedly produced a surviving escape group: %+v", svc.silo.EscapedGroup)
	}
	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil || status.Decision.Selected != endingDeath {
		t.Fatalf("physical termination chain did not select DEATH at %d-%02d: status=%+v rebellion=%.3f energy=%.0f population=%d suspicion=%.3f",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, status, svc.silo.Rebellion,
			resourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.SuspicionLevel)
	}
	t.Logf("DEATH settled at %d-%02d after %d deceives and %d rations: decision=%s escaped_group=%+v",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, deceives, rations, status.Decision.Selected, svc.silo.EscapedGroup)

	terminalAction := model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}
	if outcome, err := svc.ExecuteAction(terminalAction); err == nil {
		t.Fatalf("terminal DEATH state accepted ExecuteAction: %+v", outcome.Result)
	}
	if _, err := svc.BeginAction(terminalAction); err == nil {
		t.Fatal("terminal DEATH state accepted BeginAction")
	}
}

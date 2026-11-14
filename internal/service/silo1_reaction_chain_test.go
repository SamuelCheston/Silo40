package service

import (
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
				"special:termination_channel_test":          {Triggered: true},
				"special:silo1_purge_omen":                  {Triggered: true},
				"player_action:action_share_info":           {Triggered: true},
				"player_action:action_conduct_propaganda":   {Triggered: true},
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

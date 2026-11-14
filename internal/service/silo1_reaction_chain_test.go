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

// TestTerminationReleaseGuardedByNetworkJammed locks the remote-termination
// countermeasure: a disrupted channel (network_jammed, G11) makes the release a
// no-op, while an intact channel drives safeguard_risk to the DEATH guardrail.
func TestTerminationReleaseGuardedByNetworkJammed(t *testing.T) {
	svc := bootstrapRealContent(t)
	def := findContentDefinitionByKey(t, svc, "crisis:termination_release")
	if !def.ScriptHooks.Apply {
		t.Fatalf("termination_release must expose an apply guard")
	}

	hasSafeguardSpike := func(result engine.ContentScriptApplyResult) bool {
		for _, effect := range result.Effects {
			if effect.Type == "silo_metric_delta" && effect.Metric == "safeguard_risk" && effect.Value > 0 {
				return true
			}
		}
		return false
	}

	silo := &model.Silo{ID: 1, CurrentYear: 127, CurrentMonth: 6}

	// Intact channel: the release spikes intervention risk toward DEATH.
	intact, err := engine.RunContentEventApplyScript(def, engine.ContentEvaluationContext{
		Silo:  silo,
		World: &model.World{Flags: map[string]bool{}},
	})
	if err != nil {
		t.Fatalf("apply intact channel: %v", err)
	}
	if !hasSafeguardSpike(intact) {
		t.Fatalf("intact termination release must raise safeguard_risk, got %+v", intact.Effects)
	}

	// Jammed channel: the delivery handshake never completes, no risk spike.
	jammed, err := engine.RunContentEventApplyScript(def, engine.ContentEvaluationContext{
		Silo:  silo,
		World: &model.World{Flags: map[string]bool{model.WorldFlagNetworkJammed: true}},
	})
	if err != nil {
		t.Fatalf("apply jammed channel: %v", err)
	}
	if hasSafeguardSpike(jammed) {
		t.Fatalf("jammed termination release must not raise safeguard_risk, got %+v", jammed.Effects)
	}
}

// TestLocalPurgeExecutionGuardedByExposure locks the local-execution
// countermeasure: an exposed plot stalls the crackdown, an unopposed one
// carries out attrition.
func TestLocalPurgeExecutionGuardedByExposure(t *testing.T) {
	svc := bootstrapRealContent(t)
	def := findContentDefinitionByKey(t, svc, "crisis:silo1_purge_execution")
	if !def.ScriptHooks.Apply {
		t.Fatalf("silo1_purge_execution must expose an apply guard")
	}

	hasPopulationLoss := func(result engine.ContentScriptApplyResult) bool {
		for _, effect := range result.Effects {
			if effect.Type == "silo_population_delta" && effect.Value < 0 {
				return true
			}
		}
		return false
	}

	silo := &model.Silo{ID: 1, CurrentYear: 127, CurrentMonth: 1}

	// Unopposed: the crackdown kills and clears records.
	unopposed, err := engine.RunContentEventApplyScript(def, engine.ContentEvaluationContext{
		Silo:   silo,
		States: map[string]engine.ContentEventRuntime{},
	})
	if err != nil {
		t.Fatalf("apply unopposed purge: %v", err)
	}
	if !hasPopulationLoss(unopposed) {
		t.Fatalf("unopposed purge must reduce population, got %+v", unopposed.Effects)
	}

	// Exposed: the plot loses surprise, no attrition.
	exposed, err := engine.RunContentEventApplyScript(def, engine.ContentEvaluationContext{
		Silo: silo,
		States: map[string]engine.ContentEventRuntime{
			"special:silo1_purge_exposed": {Triggered: true},
		},
	})
	if err != nil {
		t.Fatalf("apply exposed purge: %v", err)
	}
	if hasPopulationLoss(exposed) {
		t.Fatalf("exposed purge must not reduce population, got %+v", exposed.Effects)
	}
}

// TestBombingTerminalIgnoresNetworkJamming locks channel independence: jamming
// the remote-termination channel does NOT block the physical strike. The
// bombing is a plain schedule-only population-collapse node with no network
// guard; the only mitigation is a surviving escape group, read at arbitration.
func TestBombingTerminalIgnoresNetworkJamming(t *testing.T) {
	svc := bootstrapRealContent(t)
	def := findContentDefinitionByKey(t, svc, "crisis:bombing_terminal")
	if def.ScriptHooks.Apply {
		t.Fatalf("bombing_terminal must not gain an apply guard; physical strike is undefendable")
	}
	foundCollapse := false
	for _, effect := range def.Effects {
		if effect.Type == "silo_population_delta" && effect.Value < 0 {
			foundCollapse = true
		}
	}
	if !foundCollapse {
		t.Fatalf("bombing_terminal must collapse the in-silo population, got %+v", def.Effects)
	}
}

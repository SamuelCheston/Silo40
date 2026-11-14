package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestBeliefObserveEffectRoutesToObservers locks 子项 3 routing: belief_observe
// reaches the player agent, a named department, and the named residents.
func TestBeliefObserveEffectRoutesToObservers(t *testing.T) {
	def := chainFixtureDef("special", "belief_route_fixture",
		engine.ContentTrigger{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.5},
		[]engine.ContentEffect{
			{Type: "belief_observe", Observer: "player", Fragment: "Judicial_1", Reliability: 0.95},
			{Type: "belief_observe", Observer: "Judicial", Fragment: "Judicial_1", Reliability: 0.9},
			{Type: "belief_observe", Observer: "residents", Fragment: "Judicial_1", Reliability: 0.6},
		},
	)
	svc := chainTestService(def)
	svc.silo.Rebellion = 0.6
	svc.silo.Residents = []model.Resident{
		{ID: 1, SiloID: svc.silo.ID, Name: "Named Witness", Alive: true, Influence: 10},
	}
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Judicial", Population: 200},
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if belief := engine.BeliefFor(svc.agent, "Judicial_1"); belief == nil || belief.B <= 0 {
		t.Fatalf("player belief not written: %+v", belief)
	}
	var dept *model.Profession
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == "Judicial" {
			dept = &svc.silo.Professions[i]
		}
	}
	if dept == nil {
		t.Fatal("Judicial department missing")
	}
	if belief := engine.BeliefIn(&dept.Beliefs, "Judicial_1"); belief == nil || belief.B <= 0 {
		t.Fatalf("department belief not written: %+v", belief)
	}
	if belief := engine.BeliefIn(&svc.silo.Residents[0].Beliefs, "Judicial_1"); belief == nil || belief.B <= 0 {
		t.Fatalf("resident belief not written: %+v", belief)
	}
}

// TestPublicEvidenceRegisterEffect locks §3.7 registration through content.
func TestPublicEvidenceRegisterEffect(t *testing.T) {
	def := chainFixtureDef("special", "publish_fixture",
		engine.ContentTrigger{Type: "silo_metric_gte", Metric: "rebellion", Value: 0.5},
		[]engine.ContentEffect{
			{Type: "public_evidence_register", Fragment: "Judicial_1", Provenance: 1.0, Coverage: 0.9},
		},
	)
	svc := chainTestService(def)
	svc.silo.Rebellion = 0.6
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 5

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if len(svc.silo.PublicEvidence) != 1 {
		t.Fatalf("public evidence not registered: %+v", svc.silo.PublicEvidence)
	}
	record := svc.silo.PublicEvidence[0]
	if record.FactID != "Judicial_1" || record.Provenance != 1.0 || record.Coverage != 0.9 {
		t.Fatalf("unexpected record: %+v", record)
	}
	if record.Year != 122 || record.Month != 5 {
		t.Fatalf("record must carry the silo calendar: %+v", record)
	}
}

// TestBailiffEvidenceChainsLoadAndFlow locks the two independent dispatch
// evidence chains end to end: the evidence node fires on local unrest after
// the first local observation, and the player-driven verify/publish actions
// (子项 6) carry the material from private belief to public evidence.
func TestBailiffEvidenceChainsLoadAndFlow(t *testing.T) {
	svc := bootstrapRealContent(t)
	svc.engine = engine.NewGameEngine()
	svc.silo = engine.CreateInitialSilo("Evidence Fixture", 122, nil)
	svc.silo.ID = ActiveSiloID // CreateInitialSilo leaves ID 0; the session uses 1
	svc.agent = engine.CreateInitialAgent("Operative", "IT", nil, svc.silo)
	svc.agent.SiloID = ActiveSiloID
	svc.silos = []*model.Silo{svc.silo}
	svc.agents = map[uint]*model.Agent{ActiveSiloID: svc.agent}
	svc.bindContentEventHandlers()

	for _, key := range []string{
		"special:bailiff_dispatch_evidence_a",
		"special:bailiff_dispatch_evidence_b",
		"player_actions:investigate_dispatch",
		"player_actions:expose_execution_failure",
		"player_actions:verify_evidence",
		"player_actions:publish_evidence",
	} {
		findContentDefinitionByKey(t, svc, key)
	}

	// First local observation done; unrest rising.
	svc.statesMapFor(svc.silo.ID)["observation:first_local"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "observation:first_local", Triggered: true,
	}
	svc.silo.Rebellion = 0.2

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "special:bailiff_dispatch_evidence_a") {
		t.Fatal("dispatch evidence chain A did not fire under unrest")
	}
	belief := engine.BeliefFor(svc.agent, "Judicial_1")
	if belief == nil || belief.B <= 0 {
		t.Fatalf("evidence chain A did not reach the player belief layer: %+v", belief)
	}
	if len(svc.silo.PublicEvidence) != 0 {
		t.Fatalf("publication must not precede the belief floor: %+v", svc.silo.PublicEvidence)
	}

	// The player verifies the material up to the publish floor (≥0.90).
	cfg := engine.DefaultInformationConfig()
	spec := engine.FragmentSpecFor("Judicial_1")
	for i := 0; i < 4 && belief.B < 0.90; i++ {
		engine.ObserveFact(cfg, belief, spec, "verify#"+string(rune('a'+i)), 0.9)
	}
	if belief.B < 0.90 {
		t.Fatalf("verification fixture did not reach the publish floor: %v", belief.B)
	}

	// Verify action: B >= 0.85 floor (§3.6) -> verified list.
	verifyAction := model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence", Cost: 15}
	handled, result, err := svc.queuePlayerActionEvents(verifyAction)
	if err != nil || !handled || !result.Executed {
		t.Fatalf("verify action failed: handled=%v result=%+v err=%v", handled, result, err)
	}
	svc.drainQueuedEvents()
	svc.completePlayerAction(verifyAction)
	svc.drainQueuedEvents()
	if len(svc.silo.VerifiedEvidence) != 1 || svc.silo.VerifiedEvidence[0] != "Judicial_1" {
		t.Fatalf("verify action did not register the verified fact: %+v", svc.silo.VerifiedEvidence)
	}

	// Publish action: verified + B >= 0.90 -> public evidence (provenance 1.0).
	publishAction := model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence", Cost: 20}
	handled, result, err = svc.queuePlayerActionEvents(publishAction)
	if err != nil || !handled || !result.Executed {
		t.Fatalf("publish action failed: handled=%v result=%+v err=%v", handled, result, err)
	}
	svc.drainQueuedEvents()
	svc.completePlayerAction(publishAction)
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "player_action:publish_evidence") {
		t.Fatal("publication completion key missing")
	}
	if len(svc.silo.PublicEvidence) != 1 || svc.silo.PublicEvidence[0].FactID != "Judicial_1" {
		t.Fatalf("publication did not register public evidence: %+v", svc.silo.PublicEvidence)
	}
	if record := svc.silo.PublicEvidence[0]; record.Provenance != 1.0 || record.Coverage != 0.9 {
		t.Fatalf("publication provenance/coverage unexpected: %+v", record)
	}
	var judicial *model.Profession
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == "Judicial" {
			judicial = &svc.silo.Professions[i]
		}
	}
	if judicial == nil {
		t.Fatal("Judicial department missing")
	}
	jBelief := engine.BeliefIn(&judicial.Beliefs, "Judicial_1")
	if jBelief == nil || jBelief.B < cfg.PactFactBelief {
		t.Fatalf("publication must bring the Judicial layer over the pact-fact belief bar: %+v", jBelief)
	}

	env := engine.ComputeInformationEnvironment(svc.silo)
	if env.PublicEvidenceQ <= 0 || env.QualifyingFacts != 1 {
		t.Fatalf("one published core fact must count once toward Q: %+v", env)
	}
}

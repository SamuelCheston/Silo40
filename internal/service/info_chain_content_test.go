package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestInformationLineContentLoads locks that the first batch of INFORMATION
// line content (24-spec §2 "can write today" list) loads from events/ and
// passes validation alongside the rest of the content set.
func TestInformationLineContentLoads(t *testing.T) {
	svc := bootstrapRealContent(t)

	wantSpecials := []string{
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	}
	defs := map[string]engine.ContentEventDefinition{}
	for _, def := range svc.contentDefinitions {
		defs[def.Key] = def
	}
	for _, key := range wantSpecials {
		if _, ok := defs[key]; !ok {
			t.Fatalf("expected content definition %q to load from events/", key)
		}
	}
	action, ok := defs["player_actions:receiver_verify_accompany"]
	if !ok {
		t.Fatal("expected player_actions:receiver_verify_accompany to load from events/player_actions/")
	}
	if action.PlayerAction == nil || action.PlayerAction.ID != "RECEIVER_VERIFY_ACCOMPANY" {
		t.Fatal("receiver_verify_accompany must expose a player_action spec with id RECEIVER_VERIFY_ACCOMPANY")
	}
}

// TestReceiverVerifyAccompanyListed locks that the accompanied re-check shows
// up as a runnable content player action once its definition is loaded.
func TestReceiverVerifyAccompanyListed(t *testing.T) {
	svc := bootstrapRealContent(t)
	svc.silo = &model.Silo{CurrentYear: 122, CurrentMonth: 1}
	svc.agent = &model.Agent{Profession: "Mechanical"}

	seen := false
	for _, action := range svc.availablePlayerActions() {
		if action.ID == "RECEIVER_VERIFY_ACCOMPANY" {
			seen = true
		}
	}
	if !seen {
		t.Fatal("RECEIVER_VERIFY_ACCOMPANY missing from available player actions")
	}
}

// TestInformationLineChainFires drives the INFORMATION line end to end on the
// real bus wiring: a completed share ignites the replication heartbeat, the
// seeded public-evidence registry satisfies the receiver milestone, and the
// living chain plus public fact nodes light up through continuous evaluation.
func TestInformationLineChainFires(t *testing.T) {
	boot := bootstrapRealContent(t)

	pick := func(key string) engine.ContentEventDefinition {
		for _, def := range boot.contentDefinitions {
			if def.Key == key {
				return def
			}
		}
		t.Fatalf("definition %q not loaded", key)
		return engine.ContentEventDefinition{}
	}
	svc := chainTestService(
		pick("special:replication_progress"),
		pick("special:independent_receivers_met"),
		pick("special:living_chain_confirmed"),
		pick("special:public_fact_cross_dept"),
	)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Rebellion = 0.2
	svc.silo.Authority.Judicial.PublicEvidenceQ = 0.2
	// Seed the registry with the three distinct facts the chain registers
	// (replication / verification / receivers). The registry write path
	// itself is covered by the evidence chain tests; here we verify trigger
	// wiring only.
	svc.silo.PublicEvidence = []model.PublicEvidenceRecord{
		{FactID: "fact:info_chain_replication", Provenance: 0.6, Coverage: 0.1},
		{FactID: "fact:receiver_verification", Provenance: 1, Coverage: 0.1},
		{FactID: "fact:independent_receivers", Provenance: 0.6, Coverage: 0.1},
	}
	// First completed share (completion key emission is locked by
	// TestCompletionKeyEmitsBusEventForFollowups).
	svc.statesMapFor(svc.silo.ID)["player_action:action_share_info"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "player_action:action_share_info", Triggered: true, TriggerCount: 1,
	}

	// Advance 14 months: replication_progress (REPEATABLE cd3) must fire at
	// least 4 times so living_chain_confirmed's milestone passes.
	for i := 0; i < 14; i++ {
		svc.silo.CurrentMonth++
		if svc.silo.CurrentMonth > 12 {
			svc.silo.CurrentMonth = 1
			svc.silo.CurrentYear++
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
	}

	for _, key := range []string{
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s never fired", key)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:replication_progress"].TriggerCount; count < 4 {
		t.Fatalf("replication_progress fired %d times, want >= 4 for the living chain milestone", count)
	}
}

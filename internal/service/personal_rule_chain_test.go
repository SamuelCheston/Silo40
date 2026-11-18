package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestPersonalRuleEndingNarrativeAndCoexistence locks 32-personal-rule-endgame
// §2: the PERSONAL_RULE settlement tells the protected-sleep story (not a
// score formula), reports the order's factual basis with the three-state
// judicial position, and — when a charter is locked at the same time — states
// the coexistence instead of denying it.
func TestPersonalRuleEndingNarrativeAndCoexistence(t *testing.T) {
	setup := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("PERSONAL_RULE")
		station.Authority.W = 20
		station.Authority.PoliceObedience = 0.7
		station.Authority.Judicial.Regime = "SWING"
		station.SetMayorLegitimacy(0.6)
		station.Mayor.HolderResidentID = 7
		station.Mayor.HolderName = "Named Mayor"
		svc.agent.Ambition = 1
		svc.agent.Influence = 1
		return svc, station
	}

	// Variant A: contested judiciary, no charter.
	svc, station := setup()
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingPersonalRule {
		t.Fatalf("expected PERSONAL_RULE selection, got %+v", decision)
	}
	if strings.Contains(decision.Reason, "P2 加权") {
		t.Fatalf("score formula still on the ending screen: %q", decision.Reason)
	}
	for _, want := range []string{
		"《人治》成立",
		"被保护的睡眠",
		"警察服从 0.70，市长正统性 0.60，幕后威势 W=20.0，司法部处于争夺态",
		"永不归属市长",
	} {
		if !strings.Contains(decision.Reason, want) {
			t.Fatalf("narrative missing %q: %q", want, decision.Reason)
		}
	}

	// Variant B: a locked charter coexists with personal rule in the pool.
	// Under the calibrated weights the institutional fact outranks personal
	// rule when the pact is locked (prestige parity 0.5/0.5 plus the pact
	// bonus; the tie-break order also yields to RULE_OF_LAW), so the
	// settlement is RULE_OF_LAW stating the coexistence — no hard-exclusion
	// fiction on either side.
	svc2, station2 := setup()
	station2.Authority.Judicial.PactLocked = true
	station2.Authority.Judicial.Regime = "PACT"
	station2.Authority.Judicial.IndependentPublicFacts = 2
	station2.Authority.Judicial.PublicEvidenceQ = 0.9
	decision2 := svc2.resolveWeightedEnding(station2, svc2.agent)
	if decision2 == nil {
		t.Fatal("no decision")
	}
	poolHas := func(want string) bool {
		for _, outcome := range decision2.Pool {
			if outcome == want {
				return true
			}
		}
		return false
	}
	if !poolHas(endingRuleOfLaw) || !poolHas(endingPersonalRule) {
		t.Fatalf("charter lock and personal rule must coexist in the pool: pool=%v excluded=%v",
			decision2.Pool, decision2.Excluded)
	}
	if decision2.Selected != endingRuleOfLaw {
		t.Fatalf("expected the locked charter to outrank personal rule here, got %s (%v)",
			decision2.Selected, decision2.Scores)
	}
	if !strings.Contains(decision2.Reason, "公开约章与人治并存") {
		t.Fatalf("coexistence line missing: %q", decision2.Reason)
	}
}

// TestPersonalRuleLineContentLoads locks that the PERSONAL_RULE batch
// (31-spec §2 "can write today" list) loads from events/ and passes
// validation alongside the rest of the content set.
func TestPersonalRuleLineContentLoads(t *testing.T) {
	svc := bootstrapRealContent(t)

	wantSpecials := []string{
		"special:mayor_evidence_step",
		"special:mayor_crisis_stage",
		"special:mayor_public_stand",
		"special:police_obedience_shift",
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
	for actionID, key := range map[string]string{
		"LEGITIMACY_SURVEY":         "player_actions:legitimacy_survey",
		"MAYOR_ALLY_ACCEPT":         "player_actions:mayor_ally_accept",
		"MAYOR_ALLY_BYPASS":         "player_actions:mayor_ally_bypass",
		"SUCCESSION_PURE_RULE":      "player_actions:succession_pure_rule",
		"SUCCESSION_SEED_PROCEDURE": "player_actions:succession_seed_procedure",
	} {
		def, ok := defs[key]
		if !ok {
			t.Fatalf("expected content definition %q to load from events/player_actions/", key)
		}
		if def.PlayerAction == nil || def.PlayerAction.ID != actionID {
			t.Fatalf("%s must expose a player_action spec with id %s", key, actionID)
		}
	}
}

// pickPersonalRuleDefs extracts the real special-group definitions from the
// loaded content set so the chain test runs the shipped files.
func pickPersonalRuleDefs(t *testing.T, boot *GameService) []engine.ContentEventDefinition {
	t.Helper()
	want := []string{
		"special:mayor_evidence_step",
		"special:mayor_crisis_stage",
		"special:mayor_public_stand",
		"special:police_obedience_shift",
	}
	picked := []engine.ContentEventDefinition{}
	for _, key := range want {
		picked = append(picked, pickContentDef(t, boot, key))
	}
	return picked
}

// TestPersonalRuleChainFires drives the line end to end on the real bus
// wiring: the seeded contact/accept/share keys start the feeding heartbeat,
// unrest opens the crisis stage, legitimacy above the bar lights the public
// stand, and the police shift milestone follows once obedience crosses the
// qualification bar. Qualification itself stays an engine read.
func TestPersonalRuleChainFires(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(pickPersonalRuleDefs(t, boot)...)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Rebellion = 0.35 // crisis-stage numeric branch
	states := svc.statesMapFor(svc.silo.ID)
	for _, key := range []string{
		"player_actions:legitimacy_survey",
		"player_actions:mayor_ally_accept",
		"player_action:action_share_info",
	} {
		states[key] = model.ContentEventState{SiloID: svc.silo.ID, EventKey: key, Triggered: true, TriggerCount: 1}
	}

	// Seven months: crisis stage on the first sweep (rebellion branch), the
	// feeding heartbeat (cd 3) at least twice.
	for i := 0; i < 7; i++ {
		svc.silo.CurrentMonth++
		if svc.silo.CurrentMonth > 12 {
			svc.silo.CurrentMonth = 1
			svc.silo.CurrentYear++
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
	}
	for _, key := range []string{"special:mayor_crisis_stage", "special:mayor_evidence_step"} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s never fired", key)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:mayor_evidence_step"].TriggerCount; count < 2 {
		t.Fatalf("mayor_evidence_step fired %d times, want >= 2", count)
	}

	// Legitimacy above the bar: the stand lights; obedience above the bar:
	// the police shift follows.
	svc.silo.SetMayorLegitimacy(0.55)
	svc.silo.Authority.PoliceObedience = 0.65
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "special:mayor_public_stand") {
		t.Fatalf("mayor_public_stand did not fire at legitimacy %v", svc.silo.MayorLegitimacy())
	}
	if !contentTriggered(svc, "special:police_obedience_shift") {
		t.Fatalf("police_obedience_shift did not fire at obedience %v", svc.silo.Authority.PoliceObedience)
	}
	if svc.silo.Rebellion >= 0.35 {
		t.Fatalf("stand must calm the station: rebellion %v", svc.silo.Rebellion)
	}
}

// TestPersonalRuleStandWaitsForLegitimacy locks the failure shape (30-nodes
// §7): below the legitimacy bar the stand never lights and the line stalls
// at the feeding step.
func TestPersonalRuleStandWaitsForLegitimacy(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(pickPersonalRuleDefs(t, boot)...)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 123, 6
	svc.silo.Rebellion = 0.35
	svc.silo.SetMayorLegitimacy(0.3) // below the 0.5 bar
	svc.statesMapFor(svc.silo.ID)["special:mayor_crisis_stage"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:mayor_crisis_stage", Triggered: true,
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if contentTriggered(svc, "special:mayor_public_stand") {
		t.Fatal("mayor_public_stand fired below the legitimacy bar")
	}
}

// TestSuccessionSeedOpensCharterContact locks the reroute (30-nodes §6): the
// procedure-seed key unlocks the charter contact card through the gate's
// seed path even without the rule breach, while the Judicial relation bar
// still applies. New games start with a Judicial connection, so the negative
// side is "no breach and no seed", not "no relation".
func TestSuccessionSeedOpensCharterContact(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "succession-seed.db"))
	if _, err := svc.CreateGame(threeSiloRequest(3101)); err != nil {
		t.Fatalf("create game: %v", err)
	}

	// No breach, no seed: the relation alone must not unlock the charter
	// contact.
	svc.refreshActionProgress()
	if svc.actionUnlocked("CHARTER_CONTACT_OFFER") {
		t.Fatal("charter contact unlocked without breach or seed")
	}

	// The seed alone (relation already held) must unlock it.
	svc.statesMapFor(svc.silo.ID)["player_actions:succession_seed_procedure"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "player_actions:succession_seed_procedure", Triggered: true,
	}
	svc.refreshActionProgress()
	if !svc.actionUnlocked("CHARTER_CONTACT_OFFER") {
		t.Fatal("procedure seed plus Judicial relation did not unlock the charter contact")
	}
}

// TestPersonalRuleLinePlayableFromNewGame requires the real action policy to
// settle PERSONAL_RULE from a fresh game. The policy uses only CreateGame,
// ExecuteAction and PassTime to change the simulation; in particular it never
// seeds authority, evidence, content completion or victory state.
func TestPersonalRuleLinePlayableFromNewGame(t *testing.T) {
	runner := playPersonalRuleAcceptance(t, 2401)
	svc := runner.svc
	status := svc.silo.VictoryStatus
	if status == nil {
		t.Fatal("PERSONAL_RULE line did not settle from the real new-game policy")
	}
	if status.Decision == nil {
		t.Fatalf("PERSONAL_RULE settlement has no arbitration decision: %+v", status)
	}
	if status.Decision.Selected != endingPersonalRule {
		t.Fatalf("selected ending = %s, want %s; status=%+v decision=%+v",
			status.Decision.Selected, endingPersonalRule, status, status.Decision)
	}
	if svc.silo.Authority.Judicial.PactLocked {
		t.Fatal("pure PERSONAL_RULE line must not lock the public pact")
	}
	for _, key := range []string{
		"player_actions:legitimacy_survey",
		"player_actions:mayor_ally_accept",
		"special:mayor_evidence_step",
		"special:mayor_crisis_stage",
		"special:mayor_public_stand",
		"special:police_obedience_shift",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the real PERSONAL_RULE line", key)
		}
	}

	_, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if err == nil || !strings.Contains(err.Error(), "the game has ended") {
		t.Fatalf("post-terminal action must be rejected, got %v", err)
	}

	t.Logf("PERSONAL_RULE real new-game settlement: cal=%s ending=%s selected=%s pact_locked=%v W=%.2f legitimacy=%.2f obedience=%.2f regime=%s pool=%v",
		runner.cal(), status.Type, status.Decision.Selected, svc.silo.Authority.Judicial.PactLocked,
		svc.silo.Authority.W, svc.silo.MayorLegitimacy(), svc.silo.Authority.PoliceObedience,
		svc.silo.Authority.Judicial.Regime, status.Decision.Pool)
	t.Logf("PERSONAL_RULE terminal guard: subsequent ExecuteAction rejected: %v", err)
}

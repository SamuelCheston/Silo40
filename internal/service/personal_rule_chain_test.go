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
		svc.agent.PoliticalPrestige = 100
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

	// Variant B: a locked charter coexists with personal rule; the weighting
	// (prestige favors personal rule here) settles the concurrence — no
	// hard-exclusion fiction, and the tie-break rule is stated.
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
	if decision2.Selected != endingPersonalRule {
		t.Fatalf("expected the personal-rule weighting to win here, got %s (%v)",
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

// TestPersonalRuleLinePlayableFromNewGame drives the line from a real new
// game through the real gates: the survey unlocks through the Mayor
// relation, the accept choice opens the feeding heartbeat, and the crisis
// stage arrives with real unrest. The stand and the police shift depend on
// natural legitimacy/obedience levels, so they are asserted only if the run
// reaches their bars (the W <= 30 qualification side shares the RULE_OF_LAW
// line's shock-channel dependency and is logged, not asserted).
func TestPersonalRuleLinePlayableFromNewGame(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "personal-rule-playable.db"))
	req := threeSiloRequest(3201)
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// Build the Mayor relation until the survey contact unlocks (real gates).
	var survey *model.PlayerActionMeta
	for i := 0; i < 12; i++ {
		survey = actionByID(svc.availablePlayerActions(), "LEGITIMACY_SURVEY")
		if survey != nil && survey.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance for survey contact: %v", err)
		}
	}
	if survey == nil || !survey.Enabled {
		t.Fatalf("legitimacy survey never unlocked through real gates: %+v", survey)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "LEGITIMACY_SURVEY"})
	accept := actionByID(svc.availablePlayerActions(), "MAYOR_ALLY_ACCEPT")
	if accept == nil || !accept.Enabled {
		t.Fatalf("ally accept choice not available after the survey: %+v", accept)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "MAYOR_ALLY_ACCEPT"})

	// Sustained circulation drives the feeding heartbeat. Deterministic
	// policy, no policy RNG.
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	legitimacyPeak := 0.0
	obediencePeak := 0.0
	wMin := 100.0
	for month := 1; month <= 100 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance PERSONAL_RULE line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		if l := svc.silo.MayorLegitimacy(); l > legitimacyPeak {
			legitimacyPeak = l
		}
		if o := svc.silo.Authority.PoliceObedience; o > obediencePeak {
			obediencePeak = o
		}
		if w := svc.silo.Authority.W; w < wMin {
			wMin = w
		}

		ap := svc.agent.ActionPoints
		shareCost := model.ACTION_COSTS[model.ActionShareInfo]
		gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
		gatherFrom := rotation[month%len(rotation)]
		switch {
		case svc.silo.Rebellion >= 0.7:
			// Lie low while rebellion runs hot.
		case resourceAmount(svc.silo, "Supplies") < 2500 && ap >= 15:
			svc.ExecuteAction(model.AgentAction{
				Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
			})
		case ap >= shareCost:
			dept := neediestDepartment(svc.silo)
			fragment, ok := strongestShareableFactFor(svc.agent, departmentByAddress(svc.silo, dept))
			if ok {
				if outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{fragment},
				}); err != nil || !outcome.Result.Executed {
					svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
				}
			} else {
				svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
			}
		case ap >= gatherCost:
			svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
		}
	}

	for _, key := range []string{
		"player_actions:legitimacy_survey",
		"player_actions:mayor_ally_accept",
		"special:mayor_evidence_step",
		"special:mayor_crisis_stage",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the playable line (legitimacy_peak=%.2f obedience_peak=%.2f)",
				key, legitimacyPeak, obediencePeak)
		}
	}
	// The stand is asserted only if legitimacy actually reached the bar.
	if legitimacyPeak >= 0.5 && !contentTriggered(svc, "special:mayor_public_stand") {
		t.Errorf("legitimacy peaked at %.2f but mayor_public_stand never fired", legitimacyPeak)
	}
	// The police shift is asserted only if obedience reached the bar after a stand.
	if contentTriggered(svc, "special:mayor_public_stand") && obediencePeak >= 0.60 &&
		!contentTriggered(svc, "special:police_obedience_shift") {
		t.Errorf("obedience peaked at %.2f after a stand but the police shift never fired", obediencePeak)
	}
	if status := svc.silo.VictoryStatus; status != nil {
		selected := ""
		if status.Decision != nil {
			selected = status.Decision.Selected
		}
		t.Logf("PERSONAL_RULE playable settled: ending=%s selected=%s legitimacy_peak=%.2f obedience_peak=%.2f w_min=%.1f regime=%s",
			status.Type, selected, legitimacyPeak, obediencePeak, wMin, svc.silo.Authority.Judicial.Regime)
	} else {
		t.Logf("PERSONAL_RULE playable: no settlement within the window; legitimacy_peak=%.2f obedience_peak=%.2f w_min=%.1f regime=%s",
			legitimacyPeak, obediencePeak, wMin, svc.silo.Authority.Judicial.Regime)
	}
}

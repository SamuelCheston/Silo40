package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestRuleOfLawEndingNarrativeReplacesFormula locks 28-rule-of-law-endgame
// §2: the RULE_OF_LAW settlement shows the story, not the P2 score formula,
// and the charter evidence-base line reports the fact count and Q (§3).
func TestRuleOfLawEndingNarrativeReplacesFormula(t *testing.T) {
	svc, station, _ := weightedEndingFixture("RULE_OF_LAW")
	station.Authority.Judicial.PactLocked = true
	station.Authority.Judicial.Regime = "PACT"
	station.Authority.Judicial.IndependentPublicFacts = 2
	station.Authority.Judicial.PublicEvidenceQ = 0.95
	svc.agent.PoliticalPrestige = 100

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingRuleOfLaw {
		t.Fatalf("expected RULE_OF_LAW selection, got %+v", decision)
	}
	if strings.Contains(decision.Reason, "P2 加权") {
		t.Fatalf("score formula still on the ending screen: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "《公议》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "这条命令到底从哪里来") {
		t.Fatalf("honest epilogue (the question the charter cannot answer) missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "2 项独立公共事实，公开证据强度 Q=0.95") {
		t.Fatalf("charter evidence-base settlement line missing: %q", decision.Reason)
	}
}

// TestRuleOfLawAndPersonalRuleCoexistInPool locks the new three-state soft
// exclusion (26-nodes §6): a locked charter (Regime=PACT) does not block
// PERSONAL_RULE qualification — both candidates enter the pool, and the
// weighting plus fixed tie-break order settle the concurrence. No judgment
// reads the abolished "Judicial belongs to the Mayor" state.
func TestRuleOfLawAndPersonalRuleCoexistInPool(t *testing.T) {
	svc, station, _ := weightedEndingFixture("RULE_OF_LAW")
	station.Authority.Judicial.PactLocked = true
	station.Authority.Judicial.Regime = "PACT"
	station.Authority.Judicial.IndependentPublicFacts = 2
	station.Authority.Judicial.PublicEvidenceQ = 0.95
	// PERSONAL_RULE side: low W, legitimate incumbent, obedient police.
	station.Authority.W = 10
	station.Authority.PoliceObedience = 0.7
	station.SetMayorLegitimacy(0.6)
	station.Mayor.HolderResidentID = 7
	station.Mayor.HolderName = "Mayor"
	// An open-leaning resident pushes the charter score ahead of personal
	// rule (deterministic weighting, no tie involved).
	station.Residents = []model.Resident{
		{ID: 3, Name: "Open Voice", Alive: true, Influence: 1,
			Ideologies: map[string]float64{model.IdeologyProForeign: 1}},
	}
	svc.agent.PoliticalPrestige = 100

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil {
		t.Fatal("no decision")
	}
	poolHas := func(want string) bool {
		for _, outcome := range decision.Pool {
			if outcome == want {
				return true
			}
		}
		return false
	}
	if !poolHas(endingRuleOfLaw) || !poolHas(endingPersonalRule) {
		t.Fatalf("charter lock and personal rule must coexist in the pool: pool=%v excluded=%v",
			decision.Pool, decision.Excluded)
	}
	if decision.Selected != endingRuleOfLaw {
		t.Fatalf("expected the charter candidate to win the open-voice weighting, got %s (%v)",
			decision.Selected, decision.Scores)
	}
}

// TestRuleOfLawLineContentLoads locks that the RULE_OF_LAW batch (27-spec §2
// "can write today" list) loads from events/ and passes validation alongside
// the rest of the content set.
func TestRuleOfLawLineContentLoads(t *testing.T) {
	svc := bootstrapRealContent(t)

	wantSpecials := []string{
		"special:order_chain_breach",
		"special:charter_draft_step",
		"special:charter_draft_convened",
		"special:charter_lock_day",
		"crisis:charter_window_expiry",
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
		"CHARTER_CONTACT_OFFER":   "player_actions:charter_contact_offer",
		"CHARTER_CONTACT_ACCEPT":  "player_actions:charter_contact_accept",
		"CHARTER_CONTACT_DECLINE": "player_actions:charter_contact_decline",
		"CHARTER_CONTACT_DEFER":   "player_actions:charter_contact_defer",
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

// pickRuleOfLawDefs extracts the real charter definitions from the loaded
// content set so the chain test runs the shipped files, not fixtures.
func pickRuleOfLawDefs(t *testing.T, boot *GameService) []engine.ContentEventDefinition {
	t.Helper()
	want := []string{
		"special:order_chain_breach",
		"special:charter_draft_step",
		"special:charter_draft_convened",
		"special:charter_lock_day",
		"crisis:charter_window_expiry",
	}
	picked := []engine.ContentEventDefinition{}
	for _, key := range want {
		found := false
		for _, def := range boot.contentDefinitions {
			if def.Key == key {
				picked = append(picked, def)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("definition %q not loaded", key)
		}
	}
	return picked
}

// TestRuleOfLawChainFires drives the charter chain end to end on the real bus
// wiring: a completed share under enough history burden ignites the breach,
// the seeded contact/accept keys open the drafting heartbeat, the convened
// milestone follows the second step, and the lock day lights once the engine
// lock window (W/Q/facts) is open. The lock fact itself (PactLocked) stays an
// engine dynamic and is deliberately not asserted here.
func TestRuleOfLawChainFires(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(pickRuleOfLawDefs(t, boot)...)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.HistoryBurden = 0.12
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Judicial", Population: 200},
	}
	// First completed share plus the contact trilemma resolved as accept
	// (completion-key emission is locked by the wiring tests; here we verify
	// the charter chain's own trigger wiring).
	states := svc.statesMapFor(svc.silo.ID)
	for _, key := range []string{
		"player_action:action_share_info",
		"player_action:charter_contact_offer",
		"player_action:charter_contact_accept",
	} {
		states[key] = model.ContentEventState{SiloID: svc.silo.ID, EventKey: key, Triggered: true, TriggerCount: 1}
	}

	// Six months: breach on the first sweep, the drafting heartbeat (cd 2)
	// fires at least twice and the convened milestone follows.
	for i := 0; i < 6; i++ {
		svc.silo.CurrentMonth++
		if svc.silo.CurrentMonth > 12 {
			svc.silo.CurrentMonth = 1
			svc.silo.CurrentYear++
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
	}

	for _, key := range []string{
		"special:order_chain_breach",
		"special:charter_draft_step",
		"special:charter_draft_convened",
	} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s never fired", key)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:charter_draft_step"].TriggerCount; count < 2 {
		t.Fatalf("charter_draft_step fired %d times, want >= 2 for the convened milestone", count)
	}
	// Institutional feeding landed on the Judicial belief layer (rate lever).
	var judicial *model.Profession
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == "Judicial" {
			judicial = &svc.silo.Professions[i]
		}
	}
	if judicial == nil {
		t.Fatal("Judicial department missing")
	}
	if belief := engine.BeliefIn(&judicial.Beliefs, "Judicial_1"); belief == nil || belief.B <= 0 {
		t.Fatalf("drafting steps did not feed the Judicial belief layer: %+v", judicial.Beliefs)
	}

	// The lock day waits for the engine lock window: low W plus the shared
	// bailiff supply (Q and independent facts). Seeded here; in play the W
	// side waits for the shock channel (27-spec §3 dependency).
	svc.silo.Authority.W = 10
	svc.silo.Authority.Judicial.PublicEvidenceQ = 0.9
	svc.silo.Authority.Judicial.IndependentPublicFacts = 2
	svc.silo.Rebellion = 0.2
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 127, 3

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "special:charter_lock_day") {
		t.Fatalf("charter_lock_day did not fire inside the open lock window (W=%v Q=%v facts=%v)",
			svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
			svc.silo.Authority.Judicial.IndependentPublicFacts)
	}
	// Effect direction sanity: the lock day calms the station.
	if svc.silo.Rebellion != 0.16 {
		t.Fatalf("lock day rebellion effect not applied: %v, want 0.16", svc.silo.Rebellion)
	}
}

// TestRuleOfLawLockDayWaitsForWindow locks the negative side: with the window
// closed (W still high) the lock day does not fire even after the time point.
func TestRuleOfLawLockDayWaitsForWindow(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(pickRuleOfLawDefs(t, boot)...)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 127, 3
	svc.silo.Authority.W = 60 // IT grip intact: window closed
	svc.silo.Authority.Judicial.PublicEvidenceQ = 0.9
	svc.silo.Authority.Judicial.IndependentPublicFacts = 2
	svc.statesMapFor(svc.silo.ID)["special:charter_draft_convened"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:charter_draft_convened", Triggered: true,
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if contentTriggered(svc, "special:charter_lock_day") {
		t.Fatal("charter_lock_day fired while W was still high; content must wait for the engine window")
	}
}

// dispatchScheduledCrisis fires a schedule-only crisis exactly the way the
// delay queue does (bus name, no trigger check).
func dispatchScheduledCrisis(svc *GameService, key string) {
	stories := []model.StoryEvent{}
	logs := []string{}
	event := engine.CreateEvent("scheduled#"+key, key, map[string]interface{}{
		"silo":            svc.silo,
		"agent":           svc.agent,
		"content_source":  "SCHEDULED",
		"step_story_sink": &stories,
		"step_log_sink":   &logs,
	})
	svc.engine.Bus.Emit(event, engine.NewEventContext())
	svc.drainQueuedEvents()
}

// TestRuleOfLawGuardBranches locks the K-2 guard: dispatched unconditionally,
// it spins down when the lock day already fired and bleeds momentum otherwise.
func TestRuleOfLawGuardBranches(t *testing.T) {
	boot := bootstrapRealContent(t)

	var guard engine.ContentEventDefinition
	found := false
	for _, def := range boot.contentDefinitions {
		if def.Key == "crisis:charter_window_expiry" {
			guard, found = def, true
		}
	}
	if !found {
		t.Fatal("crisis:charter_window_expiry did not load from events/crisis/")
	}

	// A: unlocked branch — momentum bleeds.
	svc := chainTestService(guard)
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Judicial", Population: 200},
	}
	rebellionBefore := svc.silo.Rebellion
	dispatchScheduledCrisis(svc, "crisis:charter_window_expiry")
	if !contentTriggered(svc, "crisis:charter_window_expiry") {
		t.Fatal("guard timeout never fired from the delay dispatch")
	}
	if svc.silo.Rebellion != rebellionBefore+0.04 {
		t.Fatalf("unlocked branch must bleed momentum: rebellion %v, want %v", svc.silo.Rebellion, rebellionBefore+0.04)
	}
	if svc.silo.Professions[0].PanicValue != 0.04 {
		t.Fatalf("unlocked branch must raise panic: %v", svc.silo.Professions[0].PanicValue)
	}

	// B: locked branch — the guard spins down harmlessly.
	svc2 := chainTestService(guard)
	svc2.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc2.silo.ID, Name: "Judicial", Population: 200},
	}
	svc2.statesMapFor(svc2.silo.ID)["special:charter_lock_day"] = model.ContentEventState{
		SiloID: svc2.silo.ID, EventKey: "special:charter_lock_day", Triggered: true,
	}
	dispatchScheduledCrisis(svc2, "crisis:charter_window_expiry")
	if svc2.silo.Rebellion != 0 || svc2.silo.Professions[0].PanicValue != 0 {
		t.Fatalf("locked branch must be a no-op: rebellion %v panic %v",
			svc2.silo.Rebellion, svc2.silo.Professions[0].PanicValue)
	}
}

// TestRuleOfLawLinePlayableFromNewGame drives the line from a real new game
// through the real gates: the breach ignites from circulation, the contact
// card unlocks through the Judicial relation, and the drafting heartbeat
// convenes the charter. Qualification supply lands through verify/publish.
// Everyday pressure is established before the second publication; that
// publication and a final exposure combine the large-scale-evidence decisive
// shock with an ordinary negative shock in one monthly settlement. The route
// also performs the real RESET countermeasure and must settle through weighted
// arbitration as RULE_OF_LAW, after which further actions are rejected.
func TestRuleOfLawLinePlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testRuleOfLawLinePlayableFromNewGame)
}

func testRuleOfLawLinePlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "rule-of-law-playable.db"))
	req := threeSiloArchetypeRequest(2401, archetype)
	req.Profession = "Mines"
	// Start before the fixed archive broadcast so real tunnel actions have time
	// to build the resident support needed by weighted arbitration.
	req.StartYear = 120
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v, known=%v)", action, err, svc.agent.ActionPoints, svc.agent.KnownFragments)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v, known=%v)", action, outcome.Result, svc.agent.ActionPoints, svc.agent.KnownFragments)
		}
	}

	// Opening: gather, then a first share — the breach needs the share key
	// plus history_burden >= 0.10 (archive broadcast + early circulation).
	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	// Build the Judicial relation until the contact card unlocks (real gates).
	var offer *model.PlayerActionMeta
	for i := 0; i < 36; i++ {
		offer = actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_OFFER")
		if offer != nil && offer.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled && svc.agent.ActionPoints >= model.ACTION_COSTS[model.ActionBuildConnection] {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance for contact card: %v", err)
		}
	}
	if offer == nil || !offer.Enabled {
		t.Fatalf("charter contact card never unlocked through real gates: %+v", offer)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_OFFER"})
	accept := actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_ACCEPT")
	if accept == nil || !accept.Enabled {
		t.Fatalf("accept choice not available after the contact offer: %+v", accept)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_ACCEPT"})

	windowOpenAt := 0
	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	tunnels := 0
	decisiveFired := false
	decisivePublishW, decisiveImpactW := 0.0, 0.0
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		return err == nil && outcome.Result.Executed
	}
	for month := 1; month <= 140 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance RULE_OF_LAW line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		// Publication can put every currently held core fact above the receiving
		// floor in every department, leaving the generic helper no "new" fact to
		// choose. The countermeasure only requires a real second Share, so reuse
		// any genuinely held fact rather than inventing a completion key.
		if contentTriggered(svc, "crisis:reset_omen_requisition") &&
			contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") &&
			!contentTriggered(svc, "player_action:reset_countermeasure_done") &&
			svc.agent.ActionPoints >= model.ACTION_COSTS[model.ActionShareInfo] {
			for _, belief := range svc.agent.Beliefs {
				if belief.B < 0.3 {
					continue
				}
				execute(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: "Mayor", FragmentIds: []string{belief.FactID},
				})
				break
			}
		}
		// Establish social support through the real Mines profession action before
		// concentrating AP on evidence; continue it after the pact locks as AP allows.
		// Rift retune (19 §6.3): the amplified order volatility matures the
		// rebellion spiral around Y126-127, so the pre-lock network stops at 5
		// tunnels — the ~60 AP of the last three tunnels is exactly the expose
		// fund the decisive sequence needs to land before the spiral does; the
		// network keeps growing after the pact locks as AP allows.
		tunnelTarget := 5
		if svc.silo.Authority.Judicial.PactLocked {
			tunnelTarget = 14
		}
		if tunnels < tunnelTarget && svc.agent.ActionPoints >= 20 {
			tunnel := actionByID(svc.availablePlayerActions(), "MINES_TUNNEL_NETWORK")
			if tunnel != nil && tunnel.Enabled {
				execute(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MINES_TUNNEL_NETWORK"})
				tunnels++
				continue
			}
		}
		// Rift retune (19 §6.3 重心迁移): the amplified order volatility matures
		// the rebellion spiral around Y126, so once the tunnels are done the
		// expose bursts outrank portfolio growth — investigate and the supply
		// shares only spend the true AP surplus while the expose fund is due.
		exposeReserve := 0.0
		if !decisiveFired && exposes < 3 {
			exposeReserve = 25
		}
		gameMonth := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
		if windowOpenAt == 0 && svc.silo.Authority.W <= 18 &&
			svc.silo.Authority.Judicial.PublicEvidenceQ >= 0.85 &&
			svc.silo.Authority.Judicial.IndependentPublicFacts >= 2 {
			windowOpenAt = month
		}

		// Verify the first chain and publish it before spending AP on the second.
		// The second release waits for three real exposures, never for a W value
		// produced by its own shock. Rift retune (19 §6.3): the AP cushion drops
		// from 40 to the publish cost itself — the final exposure can wait one
		// month for AP, but the decisive shock cannot wait for the rebellion
		// spiral, which matures two months earlier than the publish-40 timeline.
		secondPublishReady := exposes >= 3 && gameMonth-lastExposeMonth >= 3 &&
			svc.agent.ActionPoints >= 20
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(svc.silo, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		if factPublished(svc.silo, "Judicial_1") && !factVerified(svc.silo, "Judicial_2") &&
			playerCoreBelief(svc, "Judicial_2") >= 0.85 && svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if fact == "Judicial_2" && !secondPublishReady {
				continue
			}
			if playerCoreBelief(svc, fact) >= 0.9 && factVerified(svc.silo, fact) &&
				!factPublished(svc.silo, fact) && svc.agent.ActionPoints >= 20 {
				publishW := svc.silo.Authority.W
				if attempt("publish_evidence", 20) && fact == "Judicial_2" {
					decisiveFired = true
					decisivePublishW = publishW
				}
				break
			}
		}
		if svc.agent.ActionPoints >= 15+exposeReserve &&
			(playerCoreBelief(svc, "Judicial_1") < 0.9 || playerCoreBelief(svc, "Judicial_2") < 0.9) &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			attempt("investigate_dispatch", 15)
		}

		exposeForPressure := !decisiveFired && exposes < 3 && svc.agent.ActionPoints >= 25
		exposeForDecisive := decisiveFired && decisiveImpactW == 0 && svc.agent.ActionPoints >= 20
		if (exposeForPressure || exposeForDecisive) && gameMonth-lastExposeMonth >= 3 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			if attempt("expose_execution_failure", 20) {
				exposes++
				lastExposeMonth = gameMonth
				if decisiveFired && decisiveImpactW == 0 {
					decisiveImpactW = svc.silo.Authority.W
				}
			}
		}
		for _, dept := range []string{"Mechanical", "Supply"} {
			for _, fact := range []string{"Judicial_1", "Judicial_2"} {
				if playerCoreBelief(svc, fact) < 0.6 || svc.agent.ActionPoints < 25+exposeReserve ||
					shares[dept+":"+fact] >= 2 || (exposes >= 3 && !decisiveFired) {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{fact},
				})
				if err == nil && outcome.Result.Executed {
					shares[dept+":"+fact]++
				}
			}
		}
	}

	for _, key := range []string{
		"special:order_chain_breach",
		"player_actions:charter_contact_offer",
		"player_actions:charter_contact_accept",
		"special:charter_draft_step",
		"special:charter_draft_convened",
		"special:charter_lock_day",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the playable line (W=%v Q=%v facts=%v)", key,
				svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
				svc.silo.Authority.Judicial.IndependentPublicFacts)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:charter_draft_step"].TriggerCount; count < 2 {
		t.Errorf("drafting heartbeat fired %d times, want at least 2", count)
	}
	// Shared supply chain must have registered the core dispatch facts.
	for _, factID := range []string{"Judicial_1", "Judicial_2"} {
		found := false
		for _, record := range svc.silo.PublicEvidence {
			if record.FactID == factID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("shared supply did not register %s (evidence=%+v)", factID, svc.silo.PublicEvidence)
		}
	}
	if svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 {
		t.Errorf("natural authority updates did not produce Q from the shared supply: Q=%v evidence=%+v",
			svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
	}
	if !svc.silo.Authority.Judicial.PactQualified {
		t.Errorf("qualifying facts did not set PactQualified (facts=%v)",
			svc.silo.Authority.Judicial.IndependentPublicFacts)
	}
	status := svc.silo.VictoryStatus
	ending, selected := "", ""
	var pool []string
	var scores map[string]float64
	if status != nil {
		ending = status.Type
		if status.Decision != nil {
			selected = status.Decision.Selected
			pool = status.Decision.Pool
			scores = status.Decision.Scores
		}
	}
	t.Logf("RULE_OF_LAW playable settled: ending=%s selected=%s date=%d-%02d W=%.2f Q=%.4f facts=%d pact_locked=%v rebellion=%.3f suspicion=%.3f window_open_month=%d exposes=%d tunnels=%d publishW=%.2f impactW=%.2f reset_countermeasure=%v reset_execution=%v shocks=%+v pool=%v scores=%v",
		ending, selected, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
		svc.silo.Authority.Judicial.IndependentPublicFacts, svc.silo.Authority.Judicial.PactLocked,
		svc.silo.Rebellion, svc.agent.SuspicionLevel, windowOpenAt, exposes, tunnels, decisivePublishW, decisiveImpactW,
		contentTriggered(svc, "player_action:reset_countermeasure_done"),
		contentTriggered(svc, "crisis:reset_execution"), svc.silo.Authority.DecisiveShocks, pool, scores)
	if !decisiveFired {
		t.Fatal("the second qualifying fact was never published through the real action")
	}
	if windowOpenAt == 0 {
		t.Fatal("the real decisive event never opened the covenant window")
	}
	if !contentTriggered(svc, "player_action:reset_countermeasure_done") {
		t.Fatal("the route did not complete the RESET countermeasure through real actions")
	}
	if contentTriggered(svc, resetFactKey) {
		t.Fatal("the completed RESET countermeasure still emitted reset_completed")
	}
	if !svc.silo.Authority.Judicial.PactLocked {
		t.Fatal("the playable route ended without a locked pact")
	}
	if status == nil || status.Decision == nil {
		t.Fatalf("playable route did not produce a real arbitration decision: status=%+v", status)
	}
	if status.Decision.Selected != endingRuleOfLaw {
		for _, contribution := range status.Decision.Contributions {
			t.Logf("contribution: kind=%s entity=%s name=%s scores=%v", contribution.Kind, contribution.EntityID, contribution.Name, contribution.Scores)
		}
		t.Fatalf("playable route selected %q instead of RULE_OF_LAW: pool=%v scores=%v excluded=%v",
			status.Decision.Selected, status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal RULE_OF_LAW state accepted another action: %+v", outcome.Result)
	}
}

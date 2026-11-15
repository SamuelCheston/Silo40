package service

// Ending acceptance harness for the two lines the decisive-shock channel
// unlocks: 《公议》 (RULE_OF_LAW) and 《人治》 (PERSONAL_RULE). Headless,
// fixed seed, full service stack with real content and real gates — the
// strategies play from a fresh game and never white-box state. Opt-in via
// SILO40_ENDING_ACCEPTANCE_DUMP=<output.json> so it never runs in normal CI.
//
// Policies are deterministic functions of observable state (no policy RNG):
//
//   - ruleoflaw: charter contact -> circulation -> investigate/verify/publish
//     the first core fact, keep everyday pressure (expose/share), publish the
//     second core fact only once W has reached the daily trough band — that
//     second registration IS the large-scale-evidence decisive shock. Then
//     circulate to the belief gate so arbitration settles.
//   - personalrule: mayor ally route (survey -> accept) + the SAME everyday
//     pressure but only ONE published core fact — no decisive event. The
//     line must settle on the daily-pressure retreat of W (<= MayorOpenW)
//     with police obedience + legitimacy, and shows whether the covenant
//     competes in the same pool.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

type acceptanceBeat struct {
	Month  int     `json:"month"`
	Cal    string  `json:"cal"`
	Note   string  `json:"note"`
	W      float64 `json:"w,omitempty"`
	E      float64 `json:"e,omitempty"`
	Q      float64 `json:"q,omitempty"`
	Regime string  `json:"regime,omitempty"`
	Ob     float64 `json:"obedience,omitempty"`
	Leg    float64 `json:"legitimacy,omitempty"`
	Reb    float64 `json:"rebellion,omitempty"`
	Sus    float64 `json:"suspicion,omitempty"`
	AP     float64 `json:"ap,omitempty"`
}

type acceptanceRecord struct {
	Line            string             `json:"line"`
	Seed            int64              `json:"seed"`
	Profession      string             `json:"profession"`
	Ending          string             `json:"ending"`
	Won             bool               `json:"won"`
	SettledMonth    int                `json:"settled_month"` // 0 = no settlement within the cap
	Pool            []string           `json:"pool,omitempty"`
	Excluded        map[string]string  `json:"excluded,omitempty"`
	Scores          map[string]float64 `json:"scores,omitempty"`
	PactLocked      bool               `json:"pact_locked"`
	PactLockedMonth int                `json:"pact_locked_month,omitempty"`
	WMin            float64            `json:"w_min"`
	W30Month        int                `json:"w_le_30_month,omitempty"`
	W18Month        int                `json:"w_le_18_month,omitempty"`
	DecisiveShocks  []string           `json:"decisive_shocks,omitempty"`
	MayorHolder     string             `json:"mayor_holder,omitempty"`
	MayorHolderID   uint               `json:"mayor_holder_id,omitempty"`
	Prestige        float64            `json:"prestige,omitempty"`
	Timeline        []acceptanceBeat   `json:"timeline"`
	Fires           map[string]int     `json:"fires,omitempty"`
	Actions         map[string]int     `json:"actions,omitempty"`
}

type acceptanceRunner struct {
	t       *testing.T
	svc     *GameService
	rec     *acceptanceRecord
	start   int
	seen    map[string]bool
	w30     bool
	w18     bool
	actions map[string]int
}

func newAcceptanceRunner(t *testing.T, line string, seed int64, profession string) *acceptanceRunner {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), line+"-acceptance.db"))
	req := threeSiloRequest(seed)
	req.Profession = profession
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game %s seed %d: %v", line, seed, err)
	}
	r := &acceptanceRunner{
		t:    t,
		svc:  svc,
		rec:  &acceptanceRecord{Line: line, Seed: seed, Profession: profession, Fires: map[string]int{}, Actions: map[string]int{}},
		seen: map[string]bool{},
	}
	r.start = svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	return r
}

func (r *acceptanceRunner) month() int {
	return r.svc.silo.CurrentYear*12 + r.svc.silo.CurrentMonth - r.start
}

func (r *acceptanceRunner) cal() string {
	return fmt.Sprintf("%d-%02d", r.svc.silo.CurrentYear, r.svc.silo.CurrentMonth)
}

func (r *acceptanceRunner) beat(note string) {
	silo := r.svc.silo
	r.rec.Timeline = append(r.rec.Timeline, acceptanceBeat{
		Month: r.month(), Cal: r.cal(), Note: note,
		W: silo.Authority.W, E: silo.Authority.E, Q: silo.Authority.Judicial.PublicEvidenceQ,
		Regime: silo.Authority.Judicial.Regime, Ob: silo.Authority.PoliceObedience,
		Leg: silo.MayorLegitimacy(), Reb: silo.Rebellion,
		Sus: r.svc.agent.SuspicionLevel, AP: r.svc.agent.ActionPoints,
	})
}

// snapshot records newly lit content keys and threshold crossings.
func (r *acceptanceRunner) snapshot() {
	silo := r.svc.silo
	states := r.svc.contentStates[silo.ID]
	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		state := states[key]
		if !state.Triggered || r.seen[key] {
			continue
		}
		r.seen[key] = true
		r.beat("node lit: " + key)
	}
	if !r.w30 && silo.Authority.W <= 30 {
		r.w30 = true
		r.rec.W30Month = r.month()
		r.beat("W crossed 30 downward (MayorOpenW / personal-rule retreat bar)")
	}
	if !r.w18 && silo.Authority.W <= 18 {
		r.w18 = true
		r.rec.W18Month = r.month()
		r.beat("W crossed 18 downward (covenant gate)")
	}
	if silo.Authority.W < r.rec.WMin || r.rec.WMin == 0 {
		r.rec.WMin = silo.Authority.W
	}
	if !r.rec.PactLocked && silo.Authority.Judicial.PactLocked {
		r.rec.PactLocked = true
		r.rec.PactLockedMonth = r.month()
		r.beat("PactLocked (covenant locked, regime=PACT)")
	}
}

func (r *acceptanceRunner) execute(label string, action model.AgentAction) bool {
	outcome, err := r.svc.ExecuteAction(action)
	if err != nil {
		return false
	}
	if outcome.Result.Executed {
		r.rec.Actions[label]++
	}
	return outcome.Result.Executed
}

func (r *acceptanceRunner) passTime() {
	if _, err := r.svc.PassTime(1); err != nil {
		r.t.Fatalf("pass time at %s: %v", r.cal(), err)
	}
	r.snapshot()
}

func (r *acceptanceRunner) settle() {
	silo := r.svc.silo
	if silo.VictoryStatus != nil {
		r.rec.Ending = silo.VictoryStatus.Type
		r.rec.Won = silo.VictoryStatus.IsWon
		r.rec.SettledMonth = r.month()
		if decision := silo.VictoryStatus.Decision; decision != nil {
			r.rec.Pool = decision.Pool
			r.rec.Scores = decision.Scores
			r.rec.Excluded = decision.Excluded
		}
		r.beat("settled: " + r.rec.Ending)
	}
	r.rec.MayorHolder = silo.Mayor.HolderName
	r.rec.MayorHolderID = silo.Mayor.HolderResidentID
	r.rec.Prestige = r.svc.agent.PoliticalPrestige
	for _, shock := range silo.Authority.DecisiveShocks {
		r.rec.DecisiveShocks = append(r.rec.DecisiveShocks,
			fmt.Sprintf("%s@%d-%02d(value=%.0f,applied=%v)", shock.Source, shock.Year, shock.Month, shock.Value,
				len(r.rec.DecisiveShocks) < silo.Authority.DecisiveApplied))
	}
	for _, entry := range r.svc.eventLogs {
		key := entry.EventID
		if entry.Category != "" {
			key = entry.Category + ":" + entry.EventID
		}
		r.rec.Fires[key]++
	}
}

// evidenceRoute drives verify/publish for the core dispatch facts.
// publishSecond=false never publishes the second qualifying fact (no decisive
// event); with publishSecond=true the second fact waits for the trough band so
// the decisive blow lands on an already-ground-down W. Monthly priority order
// across the loop: verify/publish -> expose -> investigate -> circulation.
func (r *acceptanceRunner) evidenceRoute(publishSecond, waitTrough bool) {
	svc, silo := r.svc, r.svc.silo
	// Trough gate for the decisive blow: the −24 shock only sustains the
	// 3-month W<=18 window when it lands on a W already ground to ~36 or
	// below (14-parameters-and-audit.md §9.1).
	trough := silo.Authority.W <= 36
	beliefOf := func(factID string) float64 {
		if belief := engine.BeliefFor(svc.agent, factID); belief != nil {
			return belief.B
		}
		return 0
	}
	allowSecond := publishSecond && (!waitTrough || trough)
	for _, fact := range []string{"Judicial_1", "Judicial_2"} {
		if fact == "Judicial_2" && !allowSecond {
			continue
		}
		if beliefOf(fact) >= 0.85 && !factVerified(silo, fact) && svc.agent.ActionPoints >= 15 {
			if r.execute("verify_evidence", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence"}) {
				r.beat("verified " + fact)
			}
			break
		}
	}
	for _, fact := range []string{"Judicial_1", "Judicial_2"} {
		if fact == "Judicial_2" && !allowSecond {
			continue
		}
		if beliefOf(fact) >= 0.9 && factVerified(silo, fact) && !factPublished(silo, fact) && svc.agent.ActionPoints >= 20 {
			if r.execute("publish_evidence", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence"}) {
				note := "published " + fact
				if fact == "Judicial_2" {
					note += " (second qualifying fact = large-scale-evidence DECISIVE EVENT)"
				}
				r.beat(note)
			}
			break
		}
	}
}

// investigateLagging deepens the weaker core chain; it runs before the expose
// burst because the publications it unlocks are the bigger E lever. Covert
// actions pause while suspicion runs hot (skilled play does not push the
// trace over the line).
func (r *acceptanceRunner) investigateLagging() {
	svc := r.svc
	if svc.agent.SuspicionLevel >= 0.6 {
		return
	}
	beliefOf := func(factID string) float64 {
		if belief := engine.BeliefFor(svc.agent, factID); belief != nil {
			return belief.B
		}
		return 0
	}
	if svc.agent.ActionPoints >= 15 &&
		(beliefOf("Judicial_1") < 0.9 || beliefOf("Judicial_2") < 0.9) &&
		(beliefOf("Judicial_1") >= 0.6 || beliefOf("Judicial_2") >= 0.6) {
		r.execute("investigate_dispatch", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "investigate_dispatch"})
	}
}

// exposeBurst fires one public capability check when the interval and AP allow
// (up to cap total exposes).
func (r *acceptanceRunner) exposeBurst(m int, exposes, lastExpose *int, cap int) {
	svc := r.svc
	if svc.agent.SuspicionLevel >= 0.6 {
		return
	}
	beliefFloor := func(factID string) bool {
		belief := engine.BeliefFor(svc.agent, factID)
		return belief != nil && belief.B >= 0.6
	}
	if *exposes < cap && m-*lastExpose >= 3 && svc.agent.ActionPoints >= 20 &&
		(beliefFloor("Judicial_1") || beliefFloor("Judicial_2")) {
		if r.execute("expose_execution_failure", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "expose_execution_failure"}) {
			*exposes++
			*lastExpose = m
		}
	}
}

// everydayPressure runs rationing plus circulation. Burst mode
// (circulationOnly=false) circulates only on true surplus so the 15-20 AP
// bursts keep getting funded; circulation mode switches to the cheap full-time
// share/gather cadence that grows department beliefs toward the settlement
// gate. reserve keeps that much AP untouched (the contention run funds its
// post-lock W top-ups from it).
func (r *acceptanceRunner) everydayPressure(m int, shareAttempts map[string]int, circulationOnly bool, reserve float64) {
	svc, silo := r.svc, r.svc.silo
	ap := svc.agent.ActionPoints - reserve

	if resourceAmount(silo, "Supplies") < 1500 && ap >= 15 {
		r.execute("supply_ration", model.AgentAction{
			Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
		})
		return
	}

	shareCost := model.ACTION_COSTS[model.ActionShareInfo]
	gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	gatherFrom := rotation[m%len(rotation)]

	shareNeediest := func() bool {
		order := append([]model.Profession(nil), silo.Professions...)
		sort.SliceStable(order, func(i, j int) bool {
			return informationHeldFacts(&order[i]) < informationHeldFacts(&order[j])
		})
		for i := range order {
			dept := &order[i]
			fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
				return shareAttempts[dept.Name+":"+fact] >= 2
			})
			if !ok {
				continue
			}
			shareAttempts[dept.Name+":"+fragment]++
			return r.execute("share", model.AgentAction{
				Type: model.ActionShareInfo, TargetDept: dept.Name, FragmentIds: []string{fragment},
			})
		}
		return false
	}

	if circulationOnly {
		if ap >= shareCost && m%2 == 0 && shareNeediest() {
			return
		}
		if ap >= gatherCost && svc.agent.SuspicionLevel < 0.6 {
			r.execute("gather", model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
		}
		return
	}

	// Burst mode: circulate only on genuine surplus so expose/publish keep
	// getting funded.
	if ap >= 25 {
		if m%2 == 0 && shareNeediest() {
			return
		}
		r.execute("gather", model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
	}
}

// countermeasures answers the RESET chain with the real route: the offer
// card, then the two shares that complete it. When the broadcast-heavy
// strategy has already shared everything it holds (nothing shareable left),
// it gathers fresh material first — the countermeasure must not fail for
// lack of a shareable fact.
func (r *acceptanceRunner) countermeasures() {
	svc := r.svc
	if svc.gameOver() || contentTriggered(svc, "player_action:reset_countermeasure_done") ||
		!contentTriggered(svc, "crisis:reset_omen_requisition") {
		return
	}
	if !contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") {
		if card := actionByID(svc.availablePlayerActions(), "RESET_COUNTERMEASURE_OFFER"); card != nil && card.Enabled {
			r.execute("RESET_COUNTERMEASURE_OFFER", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RESET_COUNTERMEASURE_OFFER"})
		}
		return
	}
	if svc.agent.ActionPoints < model.ACTION_COSTS[model.ActionShareInfo] {
		return
	}
	// The completion key counts ANY share (trigger_count >= 2): re-share the
	// strongest held fact to the neediest department even when it already
	// holds it — the countermeasure must not fail for lack of novelty.
	best, bestB := "", 0.0
	for _, belief := range svc.agent.Beliefs {
		if belief.B >= 0.30 && belief.B > bestB {
			best, bestB = belief.FactID, belief.B
		}
	}
	if best == "" {
		rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
		r.execute("countermeasure_gather", model.AgentAction{
			Type: model.ActionGatherInfo, TargetDept: rotation[r.month()%len(rotation)],
		})
		return
	}
	r.execute("countermeasure_share", model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: neediestDepartment(svc.silo), FragmentIds: []string{best},
	})
}

// waitForActionCard builds the given connection until the card unlocks, then
// executes it. Returns false when it never unlocks within the budget.
func (r *acceptanceRunner) waitForActionCard(cardID, buildDept string, budget int) bool {
	svc := r.svc
	for i := 0; i < budget; i++ {
		card := actionByID(svc.availablePlayerActions(), cardID)
		if card != nil && card.Enabled {
			return r.execute(cardID, model.AgentAction{Type: model.ActionPlayerEvent, ActionID: cardID})
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			r.execute("build_connection", model.AgentAction{Type: model.ActionBuildConnection, TargetDept: buildDept})
			continue
		}
		r.passTime()
		if svc.gameOver() {
			return false
		}
	}
	return false
}

// relationTo reads the agent's connection value to a department (the gates'
// "relation" condition source).
func (r *acceptanceRunner) relationTo(dept string) float64 {
	svc := r.svc
	for _, prof := range svc.silo.Professions {
		if prof.Name != dept {
			continue
		}
		for _, conn := range svc.agent.Connections {
			if conn.ProfessionID == prof.ID {
				return conn.Value
			}
		}
		return 0
	}
	return 0
}

// buildRelation builds a department connection up to min (the investigate
// gate needs Judicial >= 0.2 — the evidence route is unreachable without it).
func (r *acceptanceRunner) buildRelation(dept string, min float64, budget int) {
	svc := r.svc
	for i := 0; i < budget && r.relationTo(dept) < min; i++ {
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			r.execute("build_connection", model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept})
			continue
		}
		r.passTime()
		if svc.gameOver() {
			return
		}
	}
}

const acceptanceMaxMonths = 132

func runRuleOfLawAcceptance(t *testing.T, seed int64) *acceptanceRecord {
	return runRuleOfLawFamily(t, "ruleoflaw", seed, false)
}

// runContentionAcceptance plays BOTH lines in one game: the charter route and
// the mayor ally route advance together (22-chain-cross-arbitration §3), the
// decisive event locks the pact, and post-lock expose top-ups keep W under the
// personal-rule bar so RULE_OF_LAW and PERSONAL_RULE sit in the same
// settlement pool and the weighting decides the concurrence.
func runContentionAcceptance(t *testing.T, seed int64) *acceptanceRecord {
	return runRuleOfLawFamily(t, "contention", seed, true)
}

func runRuleOfLawFamily(t *testing.T, line string, seed int64, withMayorRoute bool) *acceptanceRecord {
	t.Helper()
	r := newAcceptanceRunner(t, line, seed, "Judicial")
	svc := r.svc
	r.beat("game created (profession=Judicial)")

	// Opening: first local observation + one share open the stage ladder.
	r.execute("gather", model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	r.execute("share", model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	r.snapshot()

	// Charter contact through the real gate (Judicial relation), then accept.
	if !r.waitForActionCard("CHARTER_CONTACT_OFFER", "Judicial", 12) {
		t.Fatalf("charter contact never unlocked (seed %d)", seed)
	}
	r.beat("CHARTER_CONTACT_OFFER executed")
	if !r.execute("CHARTER_CONTACT_ACCEPT", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_ACCEPT"}) {
		t.Fatalf("charter accept rejected (seed %d)", seed)
	}
	r.beat("CHARTER_CONTACT_ACCEPT executed (guard scheduled)")

	// Contention variant: the mayor ally route advances alongside the charter.
	// NOTE: the mayor's public stand calms rebellion, which can starve the
	// bailiff chain A (rebellion >= 0.15) — investigate must carry Judicial_1
	// over the publish floor instead.
	successionTaken := false
	if withMayorRoute {
		if !r.waitForActionCard("LEGITIMACY_SURVEY", "Mayor", 12) {
			t.Fatalf("legitimacy survey never unlocked (seed %d)", seed)
		}
		r.beat("LEGITIMACY_SURVEY executed")
		if !r.execute("MAYOR_ALLY_ACCEPT", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "MAYOR_ALLY_ACCEPT"}) {
			t.Fatalf("mayor ally accept rejected (seed %d)", seed)
		}
		r.beat("MAYOR_ALLY_ACCEPT executed (feeding heartbeat opens)")
	}

	exposes, lastExpose := 0, -3
	shareAttempts := map[string]int{}
	for m := 1; m <= acceptanceMaxMonths && !svc.gameOver(); m++ {
		r.passTime()
		if svc.gameOver() {
			break
		}
		r.countermeasures()
		if svc.gameOver() {
			break
		}
		if withMayorRoute && !successionTaken && contentTriggered(svc, "special:mayor_public_stand") {
			if card := actionByID(svc.availablePlayerActions(), "SUCCESSION_PURE_RULE"); card != nil && card.Enabled {
				if r.execute("SUCCESSION_PURE_RULE", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "SUCCESSION_PURE_RULE"}) {
					successionTaken = true
					r.beat("SUCCESSION_PURE_RULE executed (pure personal-rule convergence)")
				}
			}
		}
		// The decisive blow waits for the trough: everyday pressure first.
		circulationOnly := factPublished(svc.silo, "Judicial_2")
		if svc.silo.Rebellion < 0.7 {
			r.evidenceRoute(true, true)
			if !circulationOnly {
				r.investigateLagging()
				r.exposeBurst(m, &exposes, &lastExpose, 8)
			} else if withMayorRoute && svc.silo.Authority.Judicial.PactLocked &&
				svc.silo.Authority.W >= 28 && svc.agent.ActionPoints >= 20 {
				// Contention: keep W under the personal-rule bar after the
				// lock so both lines stay eligible at the settlement.
				r.exposeBurst(m, &exposes, &lastExpose, 14)
			}
			reserve := 0.0
			if withMayorRoute && circulationOnly {
				reserve = 20 // the post-lock top-up fund
			}
			r.everydayPressure(m, shareAttempts, circulationOnly, reserve)
		}
		if m%12 == 0 {
			r.beat("year sample")
		}
	}
	r.settle()
	return r.rec
}

func runPersonalRuleAcceptance(t *testing.T, seed int64) *acceptanceRecord {
	t.Helper()
	// Judicial profession: the single-RNG stream means the profession choice
	// shifts every downstream draw; this seed+profession station keeps its
	// supplies healthy (no rationing spiral) while the evidence route stays
	// the line's pressure engine.
	r := newAcceptanceRunner(t, "personalrule", seed, "Judicial")
	svc := r.svc
	r.beat("game created (profession=Judicial)")

	r.execute("gather", model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	r.execute("share", model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	r.snapshot()

	// Mayor ally route through the real gate (Mayor relation).
	if !r.waitForActionCard("LEGITIMACY_SURVEY", "Mayor", 12) {
		t.Fatalf("legitimacy survey never unlocked (seed %d)", seed)
	}
	r.beat("LEGITIMACY_SURVEY executed")
	if !r.execute("MAYOR_ALLY_ACCEPT", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "MAYOR_ALLY_ACCEPT"}) {
		t.Fatalf("mayor ally accept rejected (seed %d)", seed)
	}
	r.beat("MAYOR_ALLY_ACCEPT executed (feeding heartbeat opens)")

	// The evidence route's investigate gate needs a Judicial connection —
	// without it the core beliefs never deepen, nothing is published, and IT
	// is free to hunt the player instead of maintaining its alliance.
	r.buildRelation("Judicial", 0.2, 12)
	if r.relationTo("Judicial") >= 0.2 {
		r.beat("Judicial connection built (investigate gate open)")
	}

	// Personal-rule succession: when the stand has happened and the succession
	// card unlocks, take the pure-rule convergence (the line's own ending
	// choice). The card gates on mayor_public_stand.
	successionTaken := false

	exposes, lastExpose := 0, -3
	for m := 1; m <= acceptanceMaxMonths && !svc.gameOver(); m++ {
		r.passTime()
		if svc.gameOver() {
			break
		}
		r.countermeasures()
		if svc.gameOver() {
			break
		}
		if !successionTaken && contentTriggered(svc, "special:mayor_public_stand") {
			if card := actionByID(svc.availablePlayerActions(), "SUCCESSION_PURE_RULE"); card != nil && card.Enabled {
				if r.execute("SUCCESSION_PURE_RULE", model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "SUCCESSION_PURE_RULE"}) {
					successionTaken = true
					r.beat("SUCCESSION_PURE_RULE executed (pure personal-rule convergence)")
				}
			}
		}
		// Everyday pressure publishes ONLY Judicial_1: one qualifying fact, no
		// decisive shock — W must retreat on daily pressure alone. Publishing is
		// also the cover: the alliance erosion it causes keeps IT busy with
		// maintenance instead of hunting the player (counter_investigate_player
		// is +0.08 suspicion per hit; without maintenance demand the hunt
		// compounds to AGENT_COMPROMISED within ~3 years).
		//
		// Once W retreats toward the MayorOpenW bar (or the expose budget is
		// spent), the window phase races the belief gate: circulation spreads
		// the QE-weighted facts through the departments, which keeps pushing
		// W_eq down — the window sustains itself — and expose only tops W back
		// up near the exit bar. Circulation must not be postponed until the
		// expose budget is empty: the AP budget funds either pressure or
		// circulation, and the gate needs ~25 gathers + ~10 shares.
		windowOpen := svc.silo.Authority.W <= 33 || exposes >= 6
		// Horizon phase: the forced settlement at Silo 1's destruction (Y130M1)
		// is the personal-rule settlement point. From Y129 on, hoard everything
		// for expose top-ups timed so W sits under the 30 bar at settlement —
		// IT maintenance outpaces affordable erosion, so the bar is only held
		// transiently and the timing is the play.
		horizon := svc.silo.CurrentYear >= 129
		if svc.silo.Rebellion < 0.7 {
			r.evidenceRoute(false, false)
			switch {
			case horizon:
				// Expose timing outranks the supply buffer in the final months
				// (rationing is a Supply-profession action anyway — a Judicial
				// agent cannot ration, so it must never shadow the top-up).
				if svc.agent.ActionPoints >= 20 {
					// Time the top-ups to the forced settlement (Y130M1): one
					// expose ~6 months out drops W toward the bar, the second
					// ~2 months out lands W under 30 and the post-shock drift
					// keeps it there through settlement (W_eq ~43 drifts back
					// only ~2 points over the last 2 months).
					cal := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
					if 130*12+1-cal <= 8 {
						r.exposeBurst(m, &exposes, &lastExpose, 14)
					}
				}
				if resourceAmount(svc.silo, "Supplies") < 500 && svc.agent.ActionPoints >= 15 {
					r.execute("supply_ration", model.AgentAction{
						Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
					})
				}
			case windowOpen:
				// Keep IT busy while hoarding: without ongoing alliance
				// erosion IT's maintenance demand collapses and it turns to
				// counter_investigate_player (+0.08 suspicion per hit), which
				// compounds to AGENT_COMPROMISED before the horizon. One
				// expose every ~9 months sustains the maintenance cycle; the
				// rest of the income is saved for the timed top-ups. The
				// settlement request comes from Silo 1's destruction at
				// Y130M1, so circulation buys nothing this line needs.
				// Rationing stays a survival duty throughout — letting supplies
				// collapse here would make the horizon rationing spiral eat
				// every AP before the top-up fund reaches 20. But the cover
				// expose outranks the supply buffer: when an expose is due,
				// hoard for it unless supplies are already critical.
				exposeDueWindow := m-lastExpose >= 9 && svc.agent.SuspicionLevel < 0.6 &&
					svc.silo.CurrentYear < 129 // keep the last interval slot for the horizon top-ups
				supplies := resourceAmount(svc.silo, "Supplies")
				switch {
				case exposeDueWindow && svc.agent.ActionPoints >= 25:
					r.exposeBurst(m, &exposes, &lastExpose, 10)
				case supplies < 1000 && svc.agent.ActionPoints >= 15:
					r.execute("supply_ration", model.AgentAction{
						Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
					})
				case supplies < 2500 && svc.agent.ActionPoints >= 15 && !exposeDueWindow:
					r.execute("supply_ration", model.AgentAction{
						Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
					})
				}
			default:
				// Austere pressure phase: expose first (E growth + alliance
				// erosion + W shocks), survival rationing second, investigate
				// third. No circulation here — at 3 AP/month it would stretch
				// the expose cycle past the horizon; department beliefs for E
				// come from the publication broadcast and the exposes.
				exposeDue := exposes < 6 && m-lastExpose >= 3 && svc.agent.SuspicionLevel < 0.6
				switch {
				case exposeDue && svc.agent.ActionPoints >= 20:
					r.exposeBurst(m, &exposes, &lastExpose, 6)
				case resourceAmount(svc.silo, "Supplies") < 2500 && svc.agent.ActionPoints >= 15:
					r.execute("supply_ration", model.AgentAction{
						Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
					})
				case !exposeDue || svc.agent.ActionPoints >= 35:
					// Never let investigate eat the expose fund.
					r.investigateLagging()
				}
			}
		}
		if m%12 == 0 {
			r.beat("year sample")
		}
	}
	r.settle()
	return r.rec
}

func TestEndingAcceptanceDump(t *testing.T) {
	outPath := os.Getenv("SILO40_ENDING_ACCEPTANCE_DUMP")
	if outPath == "" {
		t.Skip("set SILO40_ENDING_ACCEPTANCE_DUMP=<file> to dump acceptance records")
	}
	records := []*acceptanceRecord{
		runRuleOfLawAcceptance(t, 2401),
		runPersonalRuleAcceptance(t, 2401),
		runContentionAcceptance(t, 2401),
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatalf("write dump: %v", err)
	}
	for _, rec := range records {
		t.Logf("%s seed=%d ending=%s settled_month=%d pact_locked=%v(%d) w_min=%.1f w30=%d w18=%d shocks=%v pool=%v",
			rec.Line, rec.Seed, rec.Ending, rec.SettledMonth, rec.PactLocked, rec.PactLockedMonth,
			rec.WMin, rec.W30Month, rec.W18Month, rec.DecisiveShocks, rec.Pool)
	}
}

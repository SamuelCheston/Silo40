package service

// Escape acceptance harness for 《越岭》 (REBELLION): the full chain
// willingness -> route (contact/survey/formation/vanguard) -> window opening
// -> departure -> monthly outside sustenance -> tracking tiers -> settlement.
// Headless, fixed seed, full service stack with real content and real gates —
// the strategies play from a fresh game and never white-box state: stances are
// the weighted-random draw at window/guarantee/launch, the ledger is filled by
// the real preparation actions, and the settlement reads only what the
// departure settlement produced. Opt-in via SILO40_ESCAPE_ACCEPTANCE_DUMP.
//
// Runs:
//   - escapefull:   full preparation (route confirmed+stocked, plans, drill,
//     care, production, transfers, suit repair) — the line's intended play.
//   - escapesparse: the same seed with minimal preparation (no vanguard, half
//     the plan seats, no drill/care/production, one transfer) — preparation
//     must be visible in departed headcount and months survived.
//   - sweep-<seed>: the full strategy across seeds; the stance at launch is
//     recorded per seed so obstruct / non-obstruct / cooperate all surface as
//     random outcomes, not set states.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

type escapeStanceBeat struct {
	Month     int     `json:"month"`
	Cal       string  `json:"cal"`
	Trigger   string  `json:"trigger"`
	Mayor     string  `json:"mayor"`
	IT        string  `json:"it"`
	Guarantee float64 `json:"guarantee"`
	Rebellion float64 `json:"rebellion"`
	W         float64 `json:"w"`
}

type escapeRecord struct {
	Line       string `json:"line"`
	Seed       int64  `json:"seed"`
	Profession string `json:"profession"`
	PrepLevel  string `json:"prep_level"`

	Ending       string             `json:"ending"`
	Won          bool               `json:"won"`
	SettledMonth int                `json:"settled_month"`
	Pool         []string           `json:"pool,omitempty"`
	Excluded     map[string]string  `json:"excluded,omitempty"`
	Scores       map[string]float64 `json:"scores,omitempty"`
	Reason       string             `json:"reason,omitempty"`

	// Chain milestones (calendar months from game start).
	WillingnessMonth int `json:"willingness_month,omitempty"`
	UndergroundMonth int `json:"underground_month,omitempty"`
	WindowMonth      int `json:"window_month,omitempty"`
	AdvanceDispatch  int `json:"advance_dispatch_month,omitempty"`
	AdvanceReport    int `json:"advance_report_month,omitempty"`
	LaunchMonth      int `json:"launch_month,omitempty"`

	// Stance history: every re-evaluation with its draw.
	Stances            []escapeStanceBeat `json:"stances"`
	StanceAtLaunch     string             `json:"stance_at_launch,omitempty"`
	ReevaluationsTotal int                `json:"reevaluations_total"`

	// Departure settlement (the sole output arbitration may see).
	Capacity      int      `json:"capacity"`
	Departed      int      `json:"departed"`
	Survivors     int      `json:"survivors"`
	Surviving     bool     `json:"group_surviving"`
	Fragile       bool     `json:"group_fragile"`
	Notes         []string `json:"settlement_notes,omitempty"`
	SettlementLog string   `json:"settlement_log,omitempty"`

	// The sealed ledger at launch — must never reach the settlement screen.
	LedgerAtLaunch model.EscapePreparation `json:"ledger_at_launch"`

	// Outside phase.
	OutsideLines   []string `json:"outside_lines,omitempty"`
	OutsideStories []string `json:"outside_stories,omitempty"`
	MonthsOut      int      `json:"months_out"`
	Demise         string   `json:"demise,omitempty"`

	// Escaped-group facts at settle time (the strict REBELLION acceptance reads
	// these: the ending may only settle after the group faced real outside
	// sustenance and cleared the disband judgment).
	GroupPopulationAtSettlement int  `json:"group_population_at_settlement"`
	GroupSurvivingAtSettlement  bool `json:"group_surviving_at_settlement"`

	PopulationAtSettlement int              `json:"population_at_settlement"`
	Prestige               float64          `json:"prestige"`
	Suspicion              float64          `json:"suspicion"`
	Timeline               []acceptanceBeat `json:"timeline"`
	Fires                  map[string]int   `json:"fires,omitempty"`
	Actions                map[string]int   `json:"actions,omitempty"`
}

type escapeRunner struct {
	t       *testing.T
	svc     *GameService
	rec     *escapeRecord
	start   int
	seen    map[string]bool
	lastRee int

	willingness bool
	underground bool
	window      bool

	actions map[string]int

	// stanceTrigger labels the next re-evaluation (set just before executing
	// window-adjacent actions; content-driven re-evaluations keep the label of
	// the month's last candidate).
	stanceTrigger string

	launched      bool
	orderlyDone   bool
	guaranteeDone bool
	lastRepair    int
}

func newEscapeRunner(t *testing.T, line string, seed int64, prepLevel string) *escapeRunner {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), line+"-escape.db"))
	req := threeSiloRequest(seed)
	req.Profession = "Mechanical"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game %s seed %d: %v", line, seed, err)
	}
	r := &escapeRunner{
		t:       t,
		svc:     svc,
		rec:     &escapeRecord{Line: line, Seed: seed, Profession: "Mechanical", PrepLevel: prepLevel, Fires: map[string]int{}, Actions: map[string]int{}},
		start:   svc.silo.CurrentYear*12 + svc.silo.CurrentMonth,
		seen:    map[string]bool{},
		actions: map[string]int{},
	}
	return r
}

func (r *escapeRunner) month() int {
	return r.svc.silo.CurrentYear*12 + r.svc.silo.CurrentMonth - r.start
}

func (r *escapeRunner) cal() string {
	return fmt.Sprintf("%d-%02d", r.svc.silo.CurrentYear, r.svc.silo.CurrentMonth)
}

func (r *escapeRunner) calIndex() int {
	return r.svc.silo.CurrentYear*12 + r.svc.silo.CurrentMonth
}

func (r *escapeRunner) beat(note string) {
	silo := r.svc.silo
	r.rec.Timeline = append(r.rec.Timeline, acceptanceBeat{
		Month: r.month(), Cal: r.cal(), Note: note,
		W: silo.Authority.W, E: silo.Authority.E,
		Regime: silo.Authority.Judicial.Regime,
		Reb:    silo.Rebellion,
		Sus:    r.svc.agent.SuspicionLevel, AP: r.svc.agent.ActionPoints,
		Sup: resourceAmount(silo, "Supplies"), Nrg: resourceAmount(silo, "Energy"),
	})
}

func (r *escapeRunner) snapshot() {
	svc, silo := r.svc, r.svc.silo
	states := svc.contentStates[silo.ID]
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
		switch key {
		case "special:departure_willingness":
			r.willingness = true
			r.rec.WillingnessMonth = r.month()
		case "special:underground_coordination":
			r.underground = true
			r.rec.UndergroundMonth = r.month()
		case "special:escape_window_open":
			r.window = true
			r.rec.WindowMonth = r.month()
		case "player_actions:advance_party_dispatch":
			r.rec.AdvanceDispatch = r.month()
		case "crisis:advance_party_report":
			r.rec.AdvanceReport = r.month()
		}
	}
	// Yearly diagnostic sample (loop-level continues would skip it otherwise).
	if r.month()%12 == 0 {
		r.beat("year sample")
	}
	// Stance re-evaluations are the random draws; record every one.
	if silo.TopStance.Reevaluations != r.lastRee {
		r.lastRee = silo.TopStance.Reevaluations
		trigger := r.stanceTrigger
		if trigger == "" {
			trigger = "content"
		}
		r.rec.Stances = append(r.rec.Stances, escapeStanceBeat{
			Month: r.month(), Cal: r.cal(), Trigger: trigger,
			Mayor: silo.TopStance.MayorStance(), IT: silo.TopStance.ITStance(),
			Guarantee: silo.TopStance.Guarantee, Rebellion: silo.Rebellion, W: silo.Authority.W,
		})
		r.beat(fmt.Sprintf("stance re-eval #%d (%s): mayor=%s it=%s guarantee=%.2f",
			r.lastRee, trigger, silo.TopStance.MayorStance(), silo.TopStance.ITStance(), silo.TopStance.Guarantee))
		r.stanceTrigger = ""
	}
}

func (r *escapeRunner) execute(label string, action model.AgentAction) bool {
	outcome, err := r.svc.ExecuteAction(action)
	if err != nil {
		return false
	}
	if outcome != nil {
		// Duration>0 actions (the survival-duty overhaul) auto-advance the
		// month, so that tick's outside sustenance returns here, not in a
		// PassTime result — capture both channels.
		for _, line := range outcome.Logs {
			if strings.Contains(line, "[Outside]") {
				r.rec.OutsideLines = append(r.rec.OutsideLines, r.cal()+" "+line)
			}
			if strings.Contains(line, "[EscapeSettlement]") {
				r.rec.SettlementLog = r.cal() + " " + line
			}
		}
		for _, story := range outcome.Stories {
			if story.Category == "EXTERNAL" {
				r.rec.OutsideStories = append(r.rec.OutsideStories,
					fmt.Sprintf("%s %s：%s", r.cal(), story.Title, story.Description))
			}
		}
	}
	if outcome.Result.Executed {
		r.rec.Actions[label]++
		r.actions[label]++
	}
	return outcome.Result.Executed
}

func (r *escapeRunner) passTime() {
	result, err := r.svc.PassTime(1)
	if err != nil {
		r.t.Fatalf("pass time at %s: %v", r.cal(), err)
	}
	if result != nil {
		for _, line := range result.Logs {
			if strings.Contains(line, "[Outside]") {
				r.rec.OutsideLines = append(r.rec.OutsideLines, r.cal()+" "+line)
			}
			if strings.Contains(line, "[EscapeSettlement]") {
				r.rec.SettlementLog = r.cal() + " " + line
			}
		}
		for _, story := range result.Stories {
			if story.Category == "EXTERNAL" {
				r.rec.OutsideStories = append(r.rec.OutsideStories,
					fmt.Sprintf("%s %s：%s", r.cal(), story.Title, story.Description))
			}
		}
	}
	r.snapshot()
}

func (r *escapeRunner) relationTo(dept string) float64 {
	for _, prof := range r.svc.silo.Professions {
		if prof.Name != dept {
			continue
		}
		for _, conn := range r.svc.agent.Connections {
			if conn.ProfessionID == prof.ID {
				return conn.Value
			}
		}
		return 0
	}
	return 0
}

func (r *escapeRunner) cardEnabled(id string) bool {
	card := actionByID(r.svc.availablePlayerActions(), id)
	return card != nil && card.Enabled
}

func (r *escapeRunner) playerEvent(id string) model.AgentAction {
	return model.AgentAction{Type: model.ActionPlayerEvent, ActionID: id}
}

// suitTarget caps the repair campaign. Departed headcount is suit-limited,
// and survivors must clear the outside minimum group size (20): full prep
// repairs to 26, sparse to 16 (under IT cooperation the settlement doubles
// suit capacity, which is the realistic path to a group of 20+).
func suitTarget(level string) int {
	if level == "sparse" {
		return 16
	}
	return 26
}

// prepActions fills the sealed ledger through the real producer actions.
func (r *escapeRunner) prepActions(level string) {
	svc := r.svc
	prep := &svc.silo.EscapePrep

	// Transfers first: the silo's supplies collapse early on most seeds, so
	// sealing happens the moment the gate is open, before plans and drills.
	if prep.SuppliesPacked < suppliesTarget(level) && r.cardEnabled("ESCAPE_PREP_TRANSFER_SUPPLIES") &&
		svc.agent.ActionPoints >= 8 && resourceAmount(svc.silo, "Supplies") >= 250 {
		if r.execute("transfer_supplies", r.playerEvent("ESCAPE_PREP_TRANSFER_SUPPLIES")) {
			return
		}
	}

	// Plan seats need the matching department relation (0.25 gate). Three
	// seats clear the M4 organization threshold (0.5); chasing all six burns
	// the AP and months the sealed ledger needs, so stop at three.
	plans := []struct {
		action, dept string
	}{
		{"ESCAPE_PLAN_MECHANICAL", "Mechanical"},
		{"ESCAPE_PLAN_MINES", "Mines"},
		{"ESCAPE_PLAN_SUPPLY", "Supply"},
		{"ESCAPE_PLAN_MEDICAL", "Medical"},
		{"ESCAPE_PLAN_AGRICULTURAL", "Agricultural"},
		{"ESCAPE_PLAN_ORDER", "Mayor"},
	}
	planTarget := 3
	if len(prep.PlanRoles) < planTarget {
		for _, p := range plans {
			if svc.agent.ActionPoints < 4 {
				break
			}
			has := false
			for _, role := range prep.PlanRoles {
				if strings.EqualFold(role, p.dept) || role == strings.ToLower(p.dept) {
					has = true
					break
				}
			}
			if has {
				continue
			}
			if r.relationTo(p.dept) < 0.25 {
				r.buildOneRelation(p.dept)
				return // one relation per month: leave AP turns for the ledger
			}
			if r.cardEnabled(p.action) {
				if r.execute("plan_"+strings.ToLower(p.dept), r.playerEvent(p.action)) {
					return
				}
			}
		}
	}

	// Suit repair runs at top level (cooldown-limited, outranks the feed).
	// Supplies transfer runs first (above): materials are next.
	if level == "full" {
		if prep.MaterialsPacked < 60 && r.cardEnabled("ESCAPE_PREP_TRANSFER_MATERIALS") &&
			svc.agent.ActionPoints >= 8 && resourceAmount(svc.silo, "Materials") >= 250 {
			if r.execute("transfer_materials", r.playerEvent("ESCAPE_PREP_TRANSFER_MATERIALS")) {
				return
			}
		}
		if !prep.ProductionStart && prep.MaterialsPacked >= 60 && r.cardEnabled("ESCAPE_PREP_PRODUCTION") &&
			svc.agent.ActionPoints >= 6 {
			if r.execute("prep_production", r.playerEvent("ESCAPE_PREP_PRODUCTION")) {
				return
			}
		}
		if prep.DrillLevel < 0.45 && r.cardEnabled("ESCAPE_DRILL") && svc.agent.ActionPoints >= 8 {
			if r.execute("drill", r.playerEvent("ESCAPE_DRILL")) {
				return
			}
		}
		if prep.CareReady < 0.4 && r.cardEnabled("ESCAPE_PREP_CARE") && svc.agent.ActionPoints >= 6 {
			if r.execute("prep_care", r.playerEvent("ESCAPE_PREP_CARE")) {
				return
			}
		}
		// Reception stocking lifts the route to stocked (route safety 0.9 at
		// arrival) once the vanguard has confirmed it.
		if prep.ReceptionStage == model.ReceptionConfirmed &&
			r.cardEnabled("ESCAPE_RECEPTION_STOCKING") && svc.agent.ActionPoints >= 8 &&
			resourceAmount(svc.silo, "Supplies") >= 600 {
			if r.execute("reception_stocking", r.playerEvent("ESCAPE_RECEPTION_STOCKING")) {
				return
			}
		}
	}
}

func suppliesTarget(level string) float64 {
	if level == "sparse" {
		return 200
	}
	return 400
}

// countermeasures defuses the RESET chain (offer + 2 shares; the completion
// key counts ANY share). The escape line cannot afford a reset: it burns the
// months the route and the rebellion snowball need.
func (r *escapeRunner) countermeasures() {
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

func (r *escapeRunner) buildOneRelation(dept string) {
	if r.svc.agent.ActionPoints < 5 {
		return
	}
	if r.cardEnabled("action_build_connection") {
		r.execute("build_connection", model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept})
	}
}

// rebellionEngine feeds the rebellion snowball. The heavy lever is the
// forged-archive station operation (+0.15 rebellion, +panic, gated by
// propaganda level 1); propaganda opens and amplifies it. Incite stays a
// bonus lever (its prestige-60 unlock is rarely worth chasing: prestige
// reads the average connection value, so single-department ladders dilute).
//
// Pacing is the line's core skill: rebellion >= 0.8 invites the fracture
// chain, while the window needs 0.72 with underground (year >= 126,
// rebellion >= 0.62, then +0.10 snowball) already lit. So the engine parks
// rebellion at ~0.55 until the underground year gate, then pushes to the
// snowball threshold and lets the content deltas carry it to the window.
// Returns true when it spent the month's AP.
func (r *escapeRunner) rebellionEngine() bool {
	svc := r.svc
	if r.window {
		return false // the window is open: every further point of rebellion feeds fracture, not the line
	}
	target := 0.35 // content deltas snowball hard in Y125; park well below them
	if svc.silo.CurrentYear >= 126 {
		target = 0.62 // underground's own +0.10 carries the snowball to the window
	}
	if svc.silo.Rebellion < target && r.cardEnabled("STATION_DECEIVE") && svc.agent.ActionPoints >= 22 {
		return r.execute("station_deceive", model.AgentAction{Type: model.ActionStationOperation, ActionID: "STATION_DECEIVE"})
	}
	if svc.agent.PropagandaLevel < 1 && r.cardEnabled("action_conduct_propaganda") && svc.agent.ActionPoints >= 24 {
		return r.execute("propaganda", model.AgentAction{Type: model.ActionConductPropaganda})
	}
	if svc.silo.Rebellion < target && r.cardEnabled("action_incite_rebellion") && svc.agent.ActionPoints >= 35 {
		return r.execute("incite_rebellion", model.AgentAction{Type: model.ActionInciteRebellion})
	}
	return false
}

// routeActions advances contact -> survey -> formation -> vanguard.
func (r *escapeRunner) routeActions() {
	svc := r.svc
	if r.cardEnabled("ROUTE_SURVEY_OFFER") && svc.agent.ActionPoints >= 5 {
		r.execute("route_survey_offer", r.playerEvent("ROUTE_SURVEY_OFFER"))
		return
	}
	if r.cardEnabled("ROUTE_SURVEY_ACCEPT") {
		r.execute("route_survey_accept", r.playerEvent("ROUTE_SURVEY_ACCEPT"))
		return
	}
	// The survey step rides a real Mechanical profession action.
	if contentTriggered(svc, "player_actions:route_survey_accept") &&
		!contentTriggered(svc, "special:route_survey_step") &&
		svc.agent.ActionPoints >= 20 && r.svc.silo.Rebellion < 0.9 {
		r.execute("mechanical_overhaul", model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL"})
		return
	}
	// Sparse preparation skips the vanguard: the route stays unconfirmed, the
	// departure falls to the forced channel (capacity 30) and the survival
	// rate to its floor — preparation must be visible in the headcount.
	if r.rec.PrepLevel != "sparse" && r.cardEnabled("ADVANCE_PARTY_DISPATCH") && svc.agent.ActionPoints >= 6 {
		r.execute("advance_party_dispatch", r.playerEvent("ADVANCE_PARTY_DISPATCH"))
	}
}

func (r *escapeRunner) settle() {
	silo := r.svc.silo
	group := silo.EscapedGroup
	r.rec.Departed = group.Departed
	r.rec.MonthsOut = group.Outside.MonthsOut
	r.rec.Demise = group.Outside.DemiseCause
	r.rec.PopulationAtSettlement = silo.TotalPopulation
	r.rec.GroupPopulationAtSettlement = group.Population
	r.rec.GroupSurvivingAtSettlement = group.Surviving
	r.rec.Prestige = engine.DerivedPrestige(r.svc.agent.Ambition, r.svc.agent.Influence)
	r.rec.Suspicion = r.svc.agent.SuspicionLevel
	r.rec.ReevaluationsTotal = silo.TopStance.Reevaluations
	// The launch draw is recorded by the snapshot AFTER the launch action; at
	// settle time the last stance beat is the launch re-evaluation itself.
	if r.rec.LaunchMonth > 0 && len(r.rec.Stances) > 0 {
		last := r.rec.Stances[len(r.rec.Stances)-1]
		r.rec.StanceAtLaunch = "mayor=" + last.Mayor + " it=" + last.IT
	}
	// Parse the settlement audit line for capacity and notes.
	if r.rec.SettlementLog != "" {
		var capacity, departed, survivors int
		var surviving, fragile bool
		var mayorStance, itStance, notes string
		if _, err := fmt.Sscanf(r.rec.SettlementLog,
			"%*s %*s [EscapeSettlement] 容量=%d 出发=%d 幸存=%d 存续=%t 脆弱=%t 立场(市长/IT)=%s 备注=%s",
			&capacity, &departed, &survivors, &surviving, &fragile, &mayorStance, &itStance, &notes); err == nil {
			r.rec.Capacity = capacity
			if notes != "" && notes != "[]" {
				r.rec.Notes = strings.Split(strings.Trim(notes, "[]"), " ")
			}
		}
	}
	if silo.VictoryStatus != nil {
		r.rec.Ending = silo.VictoryStatus.Type
		r.rec.Won = silo.VictoryStatus.IsWon
		r.rec.SettledMonth = r.month()
		r.rec.Reason = silo.VictoryStatus.Description
		if decision := silo.VictoryStatus.Decision; decision != nil {
			r.rec.Pool = decision.Pool
			r.rec.Scores = decision.Scores
			r.rec.Excluded = decision.Excluded
			if decision.Reason != "" {
				r.rec.Reason = decision.Reason
			}
		}
		r.beat("settled: " + r.rec.Ending)
	}
	for _, entry := range r.svc.eventLogs {
		key := entry.EventID
		if entry.Category != "" {
			key = entry.Category + ":" + entry.EventID
		}
		r.rec.Fires[key]++
	}
}

func runEscapeLine(t *testing.T, line string, seed int64, level string) *escapeRecord {
	t.Helper()
	r := newEscapeRunner(t, line, seed, level)
	svc := r.svc
	r.beat("game created (profession=Mechanical, prep=" + level + ")")

	// Opening: first local observation + one share open the stage ladder.
	// Without it the progress stage stays "arrival" and the 《深闭》 normalcy
	// convergence settles the game before the escape chain can start — the
	// Witness must exist before the line can branch.
	r.execute("gather", model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	r.execute("share", model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	r.snapshot()

	for m := 1; m <= acceptanceMaxMonths && !svc.gameOver(); m++ {
		r.passTime()
		if svc.gameOver() {
			break
		}
		r.countermeasures()
		if svc.gameOver() {
			break
		}

		// Survival duty, endgame form: with energy at 0 the silo limps on for
		// years, but the fracture point of no return reads Energy <= 0 AND
		// rebellion >= 0.9 — and crises evaluate during passTime, BEFORE that
		// month's actions. So once rebellion runs hot, keep a preemptive
		// energy buffer above the line; this runs after launch too, because
		// every month the silo holds is a month the group lives outside.
		if svc.silo.Rebellion >= 0.85 && resourceAmount(svc.silo, "Energy") < 400 &&
			svc.agent.ActionPoints >= 18 {
			r.execute("mechanical_overhaul", model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL"})
		}

		if !r.launched {
			// Window choices and the guarantee (once each) — both shape the
			// launch draw, so they outrank the launch itself.
			if r.window && !r.orderlyDone && r.cardEnabled("ESCAPE_WINDOW_ORDERLY") {
				if r.execute("window_orderly", r.playerEvent("ESCAPE_WINDOW_ORDERLY")) {
					r.orderlyDone = true
				}
			}
			if r.window && !r.guaranteeDone && r.cardEnabled("ESCAPE_NEGOTIATION_GUARANTEE") &&
				svc.agent.ActionPoints >= 6 {
				r.stanceTrigger = "guarantee"
				if r.execute("negotiation_guarantee", r.playerEvent("ESCAPE_NEGOTIATION_GUARANTEE")) {
					r.guaranteeDone = true
				}
			}
			// IT cooperation opens the suit warehouse: draw it down while it
			// lasts (stance can swing back at the launch re-evaluation).
			if r.window && svc.silo.TopStance.ITStance() == model.TopStanceCooperate &&
				r.cardEnabled("ESCAPE_PREP_SUIT_WAREHOUSE") && svc.agent.ActionPoints >= 5 {
				r.execute("suit_warehouse", r.playerEvent("ESCAPE_PREP_SUIT_WAREHOUSE"))
			}
			// Departure: the settlement point. Stances get one last
			// re-evaluation inside the launch action itself — the draw at
			// launch, not the stance on display, shapes the headcount.
			//
			// Timing: the Silo-1 backlash chain (ultimatum → conscription /
			// purge backlash) locks rebellion at 1.0 ~6 months after the
			// window, and the terminal evaluation then settles the game.
			// Every month hoarded before launch is a month the group does
			// not live outside. Under IT cooperation the warehouse months
			// (+4 suits each) are the fastest path to a group of 20+
			// survivors — the repair campaign is cooldown-limited (one suit
			// per 2 months) — so at most two warehouse months are worth it;
			// without cooperation waiting only feeds fracture.
			if r.window && r.orderlyDone && r.guaranteeDone && r.cardEnabled("ESCAPE_DEPARTURE_LAUNCH") {
				prep := svc.silo.EscapePrep
				itCoop := svc.silo.TopStance.ITStance() == model.TopStanceCooperate
				sinceWindow := r.month() - r.rec.WindowMonth
				ready := false
				switch {
				case itCoop && (prep.SuitsFunctional >= 18 || sinceWindow >= 3):
					ready = true // two warehouse months at most: 20+ survivors vs months outside
				case !itCoop && sinceWindow >= 1:
					ready = true
				case r.calIndex() >= 128*12+9:
					ready = true
				}
				if ready && prep.SuppliesPacked >= 50 && prep.PlanCompleteness >= 0.5 {
					r.stanceTrigger = "launch"
					r.rec.LedgerAtLaunch = svc.silo.EscapePrep
					if r.execute("departure_launch", r.playerEvent("ESCAPE_DEPARTURE_LAUNCH")) {
						r.launched = true
						r.rec.LaunchMonth = r.month()
						group := svc.silo.EscapedGroup
						r.rec.Surviving = group.Surviving
						r.rec.Fragile = group.Fragile
						r.rec.Survivors = group.Population
						if len(r.rec.Stances) > 0 {
							last := r.rec.Stances[len(r.rec.Stances)-1]
							r.rec.StanceAtLaunch = "mayor=" + last.Mayor + " it=" + last.IT
						}
						r.beat(fmt.Sprintf("ESCAPE_DEPARTURE_LAUNCH settled: departed=%d survivors=%d surviving=%v (%s)",
							group.Departed, group.Population, group.Surviving, r.rec.StanceAtLaunch))
					}
					continue
				} else if ready && m%3 == 0 {
					r.beat(fmt.Sprintf("launch ready, gated: packed=%.0f plan=%.2f",
						prep.SuppliesPacked, prep.PlanCompleteness))
				}
			}
			// Suit repair is cooldown-limited (2 months): take every slot
			// (2 AP — cheap enough to keep even while hoarding AP for the
			// post-launch overhauls). Zero-duration actions stack within a
			// month, so the branches below fall through.
			if r.willingness && svc.silo.EscapePrep.SuitsFunctional < suitTarget(level) &&
				r.cardEnabled("ESCAPE_PREP_SUIT_REPAIR") && r.month()-r.lastRepair >= 2 &&
				svc.agent.ActionPoints >= 6 {
				if r.execute("suit_repair", r.playerEvent("ESCAPE_PREP_SUIT_REPAIR")) {
					r.lastRepair = r.month()
				}
			}
			// Rebellion is the line's engine: the snowball to underground
			// (0.62) and the window (0.72) needs sustained incitement — it
			// outranks preparation, which spends leftover AP. Incite also
			// hastens willingness itself (3 panicked departments).
			r.rebellionEngine()
			if r.willingness {
				r.routeActions()
				if svc.agent.ActionPoints >= 14 {
					r.prepActions(level)
				}
			} else if r.relationTo("Mechanical") < 0.25 || r.relationTo("Mayor") < 0.15 ||
				r.relationTo("Supply") < 0.25 {
				// Before willingness: open the offer gate (Mechanical
				// 0.2), the propaganda gate (Mayor 0.15) and the transfer
				// gate (Supply 0.25) — the sealed ledger must be filled
				// while the silo still has supplies to seal.
				switch {
				case r.relationTo("Mechanical") < 0.25:
					r.buildOneRelation("Mechanical")
				case r.relationTo("Mayor") < 0.15:
					r.buildOneRelation("Mayor")
				default:
					r.buildOneRelation("Supply")
				}
			}
		}
	}
	r.settle()
	return r.rec
}

// TestEscapeAcceptanceDump plays the 《越岭》 line end to end and dumps the
// records. Opt-in: SILO40_ESCAPE_ACCEPTANCE_DUMP=<output.json>.
func TestEscapeAcceptanceDump(t *testing.T) {
	out := os.Getenv("SILO40_ESCAPE_ACCEPTANCE_DUMP")
	if out == "" {
		t.Skip("SILO40_ESCAPE_ACCEPTANCE_DUMP not set; escape acceptance is opt-in")
	}

	// Full vs sparse pairs share the seed (not the RNG path: the strategies
	// diverge it), so preparation must show against each pair's own draws.
	var records []*escapeRecord
	records = append(records, runEscapeLine(t, "escapefull", 2401, "full"))
	records = append(records, runEscapeLine(t, "escapesparse", 2401, "sparse"))
	for _, seed := range []int64{2402, 2403, 2404, 2405, 2406, 2407, 2408, 2409, 2411, 2412, 2413, 2414, 2415, 2416, 2417, 2418, 2419, 2420, 2421, 2422, 2423, 2424, 2425, 2426, 2427, 2428, 2429, 2430} {
		records = append(records, runEscapeLine(t, fmt.Sprintf("sweep-%d", seed), seed, "full"))
		records = append(records, runEscapeLine(t, fmt.Sprintf("sparse-%d", seed), seed, "sparse"))
	}

	for _, rec := range records {
		t.Logf("%s seed=%d ending=%s launch=%s departed=%d survivors=%d months_out=%d demise=%s pool=%v",
			rec.Line, rec.Seed, rec.Ending, rec.StanceAtLaunch, rec.Departed, rec.Survivors,
			rec.MonthsOut, rec.Demise, rec.Pool)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatalf("write dump: %v", err)
	}
}

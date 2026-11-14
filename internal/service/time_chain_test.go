package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestTimeLineContentLoads locks that the TIME batch (34-spec §2 "can write
// today" list) loads from events/ with the deadline fallback field parsed.
func TestTimeLineContentLoads(t *testing.T) {
	svc := bootstrapRealContent(t)

	defs := map[string]engine.ContentEventDefinition{}
	for _, def := range svc.contentDefinitions {
		defs[def.Key] = def
	}
	for _, key := range []string{
		"special:open_discourse_milestone",
		"special:pre_fall_choice",
	} {
		if _, ok := defs[key]; !ok {
			t.Fatalf("expected content definition %q to load from events/", key)
		}
	}
	choice := defs["special:pre_fall_choice"]
	if choice.DeadlineYear != 129 || choice.DeadlineMonth != 10 {
		t.Fatalf("pre_fall_choice must carry the Y7M10 fallback deadline: %+v", choice)
	}
	for actionID, key := range map[string]string{
		"PRE_FALL_LOYALTY_MOBILIZATION": "player_actions:pre_fall_loyalty_mobilization",
		"PRE_FALL_TOTAL_RIOT":           "player_actions:pre_fall_total_riot",
		"PRE_FALL_OPEN_INQUIRY":         "player_actions:pre_fall_open_inquiry",
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

// TestTimeChainFires drives the line end to end on the real bus wiring: open
// discourse lights under pro-foreign departments, the pre-fall moment arrives
// at the Y7 time point, and the deadline fallback force-raises it even when
// the milestone never fired.
func TestTimeChainFires(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(
		pickContentDef(t, boot, "special:open_discourse_milestone"),
		pickContentDef(t, boot, "special:pre_fall_choice"),
	)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 129, 1
	for i := 1; i <= 2; i++ {
		svc.silo.Professions = append(svc.silo.Professions, model.Profession{
			ID: uint(i), SiloID: svc.silo.ID, Name: string(rune('A' + i)),
			Ideologies: map[string]float64{"pro_foreign": 0.6},
		})
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:open_discourse_milestone") {
		t.Fatal("open_discourse_milestone did not fire under pro-foreign departments")
	}
	if !contentTriggered(svc, "special:pre_fall_choice") {
		t.Fatal("pre_fall_choice did not follow at the Y7 time point")
	}

	// Deadline fallback: no milestone ever fired, but Y7M10 force-raises the
	// moment anyway (硬伤 #3).
	svc2 := chainTestService(pickContentDef(t, boot, "special:pre_fall_choice"))
	svc2.silo.CurrentYear, svc2.silo.CurrentMonth = 129, 11
	svc2.queueSystemContentEvents("TEST")
	svc2.drainQueuedEvents()
	if !contentTriggered(svc2, "special:pre_fall_choice") {
		t.Fatal("deadline fallback did not force-raise pre_fall_choice")
	}
}

// TestPreFallChoicesEffects locks the three response effects (34-spec §1 T-2):
// loyalty mobilization writes the cohort loyalty tilt, total riot damages
// sustenance, open inquiry records the moment. Runs on a real gated service
// (ExecuteAction persists).
func TestPreFallChoicesEffects(t *testing.T) {
	newGame := func(name string) *GameService {
		svc := newGatedService(t, filepath.Join(t.TempDir(), name))
		req := threeSiloRequest(3301)
		req.Profession = "Supply"
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create game: %v", err)
		}
		// Let the opening cascade (archive broadcast and its followers) settle
		// before measuring choice effects: one extra sweep drains the lagging
		// history_burden_awakened follow-up.
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("settle opening month: %v", err)
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
		svc.silo.Rebellion = 0.2 // headroom so the calming delta is not clamped
		// The pre-fall moment has arrived: unlock the three responses.
		svc.statesMapFor(svc.silo.ID)["special:pre_fall_choice"] = model.ContentEventState{
			SiloID: svc.silo.ID, EventKey: "special:pre_fall_choice", Triggered: true,
		}
		svc.refreshActionProgress()
		return svc
	}
	executeChoice := func(svc *GameService, actionID string) {
		t.Helper()
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		if err != nil {
			t.Fatalf("execute %s: %v", actionID, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("%s rejected: %+v", actionID, outcome.Result)
		}
	}

	// Loyalty mobilization: calms the station, tilts every cohort loyal.
	svc := newGame("prefall-loyalty.db")
	before := svc.silo.Rebellion
	cohortBefore := svc.silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]
	cohortWant := cohortBefore + 0.10
	if cohortWant > 1 {
		cohortWant = 1 // clampUnit
	}
	executeChoice(svc, "PRE_FALL_LOYALTY_MOBILIZATION")
	if !contentTriggered(svc, "player_actions:pre_fall_loyalty_mobilization") {
		t.Fatal("loyalty mobilization completion key missing")
	}
	if svc.silo.Rebellion != before-0.05 {
		t.Fatalf("loyalty mobilization must calm rebellion: %v, want %v", svc.silo.Rebellion, before-0.05)
	}
	if got := svc.silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]; got != cohortWant {
		t.Fatalf("cohort loyalty tilt not applied: %v, want %v", got, cohortWant)
	}

	// Total riot: raises rebellion, damages every floor.
	svc2 := newGame("prefall-riot.db")
	before2 := svc2.silo.Rebellion
	floorsBefore := make([]float64, len(svc2.silo.Floors))
	for i, floor := range svc2.silo.Floors {
		floorsBefore[i] = floor.Stability
	}
	executeChoice(svc2, "PRE_FALL_TOTAL_RIOT")
	if svc2.silo.Rebellion != before2+0.15 {
		t.Fatalf("total riot must raise rebellion: %v, want %v", svc2.silo.Rebellion, before2+0.15)
	}
	for i, floor := range svc2.silo.Floors {
		want := floorsBefore[i] - 0.1
		if want < 0 {
			want = 0
		}
		if floor.Stability != want {
			t.Fatalf("total riot must damage every floor by 0.1: floor %d got %v want %v", i, floor.Stability, want)
		}
	}

	// Open inquiry: records the moment in the history burden. Keep the burden
	// low enough that the choice's +0.03 does not trip archive_purge_order
	// (hb >= 0.12) during the same sweep.
	svc3 := newGame("prefall-inquiry.db")
	svc3.silo.HistoryBurden = 0.05
	before3 := svc3.silo.HistoryBurden
	executeChoice(svc3, "PRE_FALL_OPEN_INQUIRY")
	if svc3.silo.HistoryBurden != before3+0.03 {
		t.Fatalf("open inquiry must record the moment: %v, want %v", svc3.silo.HistoryBurden, before3+0.03)
	}
}

// TestTimeEndingNarrativeAndOverlap locks 35-time-endgame: the TIME
// settlement tells the no-one-left-to-sign story with the snapshot facts,
// and the INFORMATION/TIME overlap settles in one pool — different facts,
// no substitution, fixed tie-break.
func TestTimeEndingNarrativeAndOverlap(t *testing.T) {
	newTimeStation := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("TIME")
		// Consistent world state: the station reports the destruction and the
		// world agrees Silo 1 is gone (destroyed-base weighting).
		svc.world.Silo1.Alive = false
		station.Silo1Destroyed = true
		station.PreFallSnapshot = model.PreFallIdeologySnapshot{
			Taken: true, Year: 130, Month: 1, Population: 5000,
			Ideologies:    map[string]float64{model.IdeologyProForeign: 0.55, model.IdeologyLoyalty: 0.35},
			DominantAxis:  model.IdeologyProForeign,
			OpenQualified: true,
		}
		station.TotalPopulation = 5000
		svc.agent.PoliticalPrestige = 100
		return svc, station
	}

	svc, station := newTimeStation()
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingTime {
		t.Fatalf("expected TIME selection, got %+v", decision)
	}
	if strings.Contains(decision.Reason, "P2 加权") {
		t.Fatalf("score formula still on the ending screen: %q", decision.Reason)
	}
	for _, want := range []string{
		"《无人裁决》成立",
		"没有后续签批",
		"亲外 0.55 对忠诚 0.35",
		"外部保持沉默",
		"仍有居民 5000 人",
	} {
		if !strings.Contains(decision.Reason, want) {
			t.Fatalf("narrative missing %q: %q", want, decision.Reason)
		}
	}

	// Overlap: the same station also holds the INFORMATION gate facts — both
	// candidates share one pool; neither qualification substitutes the other.
	svc2, station2 := newTimeStation()
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
		station2.Professions = append(station2.Professions, prof)
	}
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
	if !poolHas(endingInformation) || !poolHas(endingTime) {
		t.Fatalf("INFORMATION and TIME must share the pool when both facts hold: pool=%v excluded=%v",
			decision2.Pool, decision2.Excluded)
	}
	if decision2.Selected != endingInformation && decision2.Selected != endingTime {
		t.Fatalf("overlap must settle to one of the two: %+v", decision2)
	}

	// Pure-silo snapshot: TIME excluded, the post-fall replication converges
	// to EXCLUSIONIST (Silo 1 destroyed makes it eligible).
	svc3, station3 := newTimeStation()
	station3.PreFallSnapshot.OpenQualified = false
	station3.PreFallSnapshot.DominantAxis = model.IdeologyLoyalty
	station3.PreFallSnapshot.Ideologies = map[string]float64{model.IdeologyProForeign: 0.2, model.IdeologyLoyalty: 0.7}
	decision3 := svc3.resolveWeightedEnding(station3, svc3.agent)
	if decision3 == nil || decision3.Selected != endingExclusionist {
		t.Fatalf("pure-silo snapshot must fall to post-fall replication, got %+v", decision3)
	}
	if _, excluded := decision3.Excluded[endingTime]; !excluded {
		t.Fatalf("TIME must carry an exclusion reason under a pure-silo snapshot: %+v", decision3.Excluded)
	}
}

// TestTimeLinePlayableFromNewGame drives the line from a real new game: the
// station is kept alive and calm (the lethal routes all need rebellion >=
// 0.8), the farm keeps supplies up and gossips openness into the commoner
// cohorts, the reset countermeasure is taken when the purge chain stirs, and
// the pre-fall moment arrives (deadline fallback at the latest). The final
// settlement must read the engine snapshot: TIME in the pool when
// open-qualified, excluded when not.
//
// Balance note (TODO(balance)): under the current calibration the run below
// settles with a closed snapshot — resource attrition plus the panic-to-
// loyalty drift outrun the single agent's openness levers, so the honest
// outcome demonstrated here is "outlasted into EXCLUSIONIST" (33-time-nodes
// §2). The positive TIME path is locked by the fixture-driven arbitration
// test above; making it reachable in play is a tuning batch (drift rate,
// lever strength, NPC economy), not a wiring gap.
func TestTimeLinePlayableFromNewGame(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "time-playable.db"))
	req := threeSiloRequest(3401)
	req.Profession = "Agricultural"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	tryExec := func(action model.AgentAction) bool {
		outcome, err := svc.ExecuteAction(action)
		return err == nil && outcome.Result.Executed
	}

	// Opening: gather, two shares (the reset countermeasure later needs
	// share trigger_count >= 2).
	tryExec(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if len(svc.agent.KnownFragments) > 0 {
		tryExec(model.AgentAction{Type: model.ActionShareInfo, TargetDept: "Mayor", FragmentIds: []string{svc.agent.KnownFragments[0]}})
		tryExec(model.AgentAction{Type: model.ActionShareInfo, TargetDept: "Supply", FragmentIds: []string{svc.agent.KnownFragments[0]}})
	}

	for month := 1; month <= 100 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance TIME line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		ap := svc.agent.ActionPoints
		countermeasure := actionByID(svc.availablePlayerActions(), "RESET_COUNTERMEASURE_OFFER")
		inquiry := actionByID(svc.availablePlayerActions(), "PRE_FALL_OPEN_INQUIRY")
		switch {
		case countermeasure != nil && countermeasure.Enabled:
			// The purge countermeasure outranks every other concern.
			tryExec(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RESET_COUNTERMEASURE_OFFER"})
		case inquiry != nil && inquiry.Enabled:
			tryExec(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "PRE_FALL_OPEN_INQUIRY"})
		case svc.silo.Rebellion >= 0.7:
			// Lie low: the lethal routes all feed on rebellion >= 0.8.
		case resourceAmount(svc.silo, "Supplies") < 4500 && ap >= 15:
			tryExec(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "AGRICULTURAL_INTENSIVE_HARVEST"})
		case ap >= 10:
			tryExec(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "AGRICULTURAL_FIELD_GOSSIP", TargetDept: "Mechanical"})
		}
	}

	// The pre-fall moment must arrive (deadline fallback at relative Y7M10 at
	// the latest) and the settlement must have consumed the destruction fact.
	if !contentTriggered(svc, "special:pre_fall_choice") {
		t.Errorf("pre_fall_choice never arrived (deadline fallback failed)")
	}
	if !svc.silo.Silo1Destroyed {
		t.Fatalf("silo1_destroyed_signal did not settle the run at %d-%02d", svc.silo.CurrentYear, svc.silo.CurrentMonth)
	}
	if !svc.silo.PreFallSnapshot.Taken {
		t.Fatal("destruction did not take the pre-fall ideology snapshot")
	}
	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil {
		t.Fatalf("destruction did not settle an ending: %+v", status)
	}
	inPool, excluded := false, false
	for _, outcome := range status.Decision.Pool {
		if outcome == endingTime {
			inPool = true
		}
	}
	_, excluded = status.Decision.Excluded[endingTime]
	if svc.silo.PreFallSnapshot.OpenQualified && !inPool {
		t.Errorf("open-qualified snapshot must put TIME in the pool: %+v", status.Decision)
	}
	if !svc.silo.PreFallSnapshot.OpenQualified && !excluded {
		t.Errorf("pure-silo snapshot must exclude TIME with a reason: %+v", status.Decision)
	}
	t.Logf("TIME playable settled: selected=%s open_qualified=%v pf=%.2f loyalty=%.2f dominant=%s population=%d month=%d",
		status.Decision.Selected, svc.silo.PreFallSnapshot.OpenQualified,
		svc.silo.PreFallSnapshot.Ideologies[model.IdeologyProForeign],
		svc.silo.PreFallSnapshot.Ideologies[model.IdeologyLoyalty],
		svc.silo.PreFallSnapshot.DominantAxis, svc.silo.TotalPopulation, svc.silo.CurrentMonth)
}

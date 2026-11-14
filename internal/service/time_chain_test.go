package service

import (
	"path/filepath"
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

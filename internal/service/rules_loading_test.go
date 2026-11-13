package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"
)

// bootstrapRealContent loads the repository events/ directories the same way
// BootstrapContent does, minus the embedded-FS extraction step (which is a
// packaging concern). Tests run with cwd inside internal/service, and
// resolveContentDirectories searches upward to find <repo>/events.
func bootstrapRealContent(t *testing.T) *GameService {
	t.Helper()
	db, err := repository.InitDB(filepath.Join(t.TempDir(), "rules.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := NewGameService(db)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.syncContentDefinitions(dirs); err != nil {
		t.Fatalf("sync content definitions: %v", err)
	}
	if err := svc.loadContentDefinitions(); err != nil {
		t.Fatalf("load content definitions: %v", err)
	}
	return svc
}

// TestRulesGroupIsLoaded locks the fix for the rules whitelist gap:
// events/rules/ used to be filtered out by contentEventDirectories, leaving
// the four rule events as dead content. They must load and pass validation.
func TestRulesGroupIsLoaded(t *testing.T) {
	svc := bootstrapRealContent(t)

	want := map[string]bool{
		"rules:panic_storm_delay_rule":    false,
		"rules:delayed_riot":              false,
		"rules:rebellion_escalation_rule": false,
		"rules:supplies_shortage_rule":    false,
	}
	for _, def := range svc.contentDefinitions {
		if _, ok := want[def.Key]; ok {
			want[def.Key] = true
		}
	}
	for key, seen := range want {
		if !seen {
			t.Fatalf("expected rule event %q to be loaded", key)
		}
	}
}

// TestPassTimeReturnsContentStories locks the story pass-through fix: content
// events fired during a PassTime tick must surface in TickResult.Stories
// (previously engine.Drain dropped the event-carried step_stories, so only
// the empty random-story pool was returned).
func TestPassTimeReturnsContentStories(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   "StoryPassThrough",
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       13,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	res, err := svc.PassTime(1)
	if err != nil {
		t.Fatalf("pass time: %v", err)
	}
	found := false
	for _, story := range res.Stories {
		if story.ID == "archive_truth_broadcast" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected archive_truth_broadcast story in TickResult.Stories on month 1, got %+v", res.Stories)
	}
}

// TestProcessNextEventSurfacesMonthStories locks the presentation fix: the
// step flow (BeginPassTime -> ProcessNextEvent) used to discard runOneMonth's
// stories and then error with "no pending event", leaving the operation stuck
// in phaseQueued. Now the month's content stories surface in the step result
// and the operation completes.
func TestProcessNextEventSurfacesMonthStories(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   "StepStories",
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       7,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	if _, err := svc.BeginPassTime(1); err != nil {
		t.Fatalf("begin pass time: %v", err)
	}
	res, err := svc.ProcessNextEvent()
	if err != nil {
		t.Fatalf("process next event: %v", err)
	}
	if !res.OperationComplete {
		t.Fatalf("expected single-month pass-time operation to complete, pending=%d", res.PendingEvents)
	}
	found := false
	for _, story := range res.Stories {
		if story.ID == "archive_truth_broadcast" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected archive_truth_broadcast among step stories, got %+v", res.Stories)
	}

	// The operation must not stay stuck: a second pass-time can be queued.
	if _, err := svc.BeginPassTime(1); err != nil {
		t.Fatalf("second begin pass time (operation stuck?): %v", err)
	}
	if _, err := svc.ProcessNextEvent(); err != nil {
		t.Fatalf("second process next event: %v", err)
	}
}

// TestDelayedDispatchFollowupDoesNotCrash locks the restored
// grief_release_ritual -> blackout_cascade subscription with real content:
// blackout_cascade fires via the delay queue (no story sinks), and its
// event_triggered follower must apply without crashing (nil-sink guard,
// 51d312f).
func TestDelayedDispatchFollowupDoesNotCrash(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   "DelayFollowup",
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       7,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	// Dispatch blackout_cascade exactly like the delay queue does: bus name
	// plus step sinks only, no story_sink / log_sink.
	stepStories := []model.StoryEvent{}
	stepLogs := []string{}
	event := engine.CreateEvent("scheduled#blackout_cascade", "crisis:blackout_cascade", map[string]interface{}{
		"silo":            svc.silo,
		"agent":           svc.agent,
		"content_source":  "SCHEDULED",
		"step_story_sink": &stepStories,
		"step_log_sink":   &stepLogs,
	})
	svc.engine.Bus.Emit(event, engine.NewEventContext())
	svc.drainQueuedEvents()

	if !svc.contentStates[ActiveSiloID]["crisis:blackout_cascade"].Triggered {
		t.Fatalf("expected blackout_cascade to be applied from the delay-queue dispatch")
	}
	if !svc.contentStates[ActiveSiloID]["special:grief_release_ritual"].Triggered {
		t.Fatalf("expected grief_release_ritual to follow blackout_cascade without crashing")
	}
}

// TestPanicStormSchedulesDelayedRiot verifies the rule execution chain end to
// end: a rule evaluated on the monthly system pass schedules delayed_riot,
// which fires only after its delay elapses (schedule-only canTrigger=false).
func TestPanicStormSchedulesDelayedRiot(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   "RulesChain",
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       11,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	// Force the panic_storm condition (any profession panic > 0.7).
	for i := range svc.silo.Professions {
		svc.silo.Professions[i].PanicValue = 0.8
	}

	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time month 1: %v", err)
	}
	storm := svc.contentStates[ActiveSiloID]["rules:panic_storm_delay_rule"]
	if !storm.Triggered {
		t.Fatalf("expected panic_storm_delay_rule to trigger on the first monthly pass")
	}
	if riot := svc.contentStates[ActiveSiloID]["rules:delayed_riot"]; riot.Triggered {
		t.Fatalf("delayed_riot fired immediately; expected a 3-month delay")
	}

	for m := 2; m <= 5; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", m, err)
		}
	}
	riot := svc.contentStates[ActiveSiloID]["rules:delayed_riot"]
	if !riot.Triggered || riot.TriggerCount < 1 {
		t.Fatalf("expected delayed_riot to fire after its delay, got %+v", riot)
	}
}

package engine

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

func TestLoadContentEventDefinitions(t *testing.T) {
	root := t.TempDir()
	specialDir := filepath.Join(root, "events", "special")
	historiesDir := filepath.Join(root, "events", "histories")
	if err := os.MkdirAll(specialDir, 0o755); err != nil {
		t.Fatalf("mkdir special: %v", err)
	}
	if err := os.MkdirAll(historiesDir, 0o755); err != nil {
		t.Fatalf("mkdir histories: %v", err)
	}

	eventJS := `defineEvent({
		id: "history_burden_awakened",
		title: "History Burden Awakened",
		description: "desc",
		type: "social",
		fire_mode: "once",
		trigger: siloMetricGte("history_burden", 0.08),
		effects: [siloMetricDelta("cohesion", -0.02)]
	})`
	historyJS := `defineEvent({
		id: "archive_truth_broadcast",
		title: "Archive Truth Broadcast",
		description: "desc",
		type: "external",
		fire_mode: "once",
		trigger: timestampAtOrAfter(122, 1),
		effects: [siloMetricDelta("history_burden", 0.08)]
	})`

	if err := os.WriteFile(filepath.Join(specialDir, "history_burden_awakened.js"), []byte(eventJS), 0o644); err != nil {
		t.Fatalf("write event: %v", err)
	}
	if err := os.WriteFile(filepath.Join(historiesDir, "archive_truth_broadcast.js"), []byte(historyJS), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	defs, err := LoadContentEventDefinitions(map[string]string{
		"special":   specialDir,
		"histories": historiesDir,
	})
	if err != nil {
		t.Fatalf("load definitions: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
	if defs[0].Key == "" || defs[1].Key == "" {
		t.Fatalf("expected keys to be populated")
	}
}

func TestCanTriggerAndApplyContentEvent(t *testing.T) {
	silo := &model.Silo{
		CurrentYear:   122,
		CurrentMonth:  1,
		HistoryBurden: 0.08,
		Cohesion:      1,
		Professions: []model.Profession{
			{Name: "IT", PanicValue: 0.1},
			{Name: "Medical", PanicValue: 0.2},
		},
	}

	def := ContentEventDefinition{
		EventID:     "history_burden_awakened",
		Title:       "History Burden Awakened",
		Description: "desc",
		Type:        "SOCIAL",
		FireMode:    ContentFireModeOnce,
		Trigger: ContentTrigger{
			Type:   "silo_metric_gte",
			Metric: "history_burden",
			Value:  0.08,
		},
		Effects: []ContentEffect{
			{Type: "profession_metric_delta_all", Metric: "panic_value", Value: 0.04},
			{Type: "silo_metric_delta", Metric: "cohesion", Value: -0.02},
		},
	}

	if !CanTriggerContentEvent(def, ContentEvaluationContext{Silo: silo}) {
		t.Fatalf("expected event to trigger")
	}

	story, _ := ApplyContentEvent(def, silo)
	if story.ID != def.EventID {
		t.Fatalf("expected story id %q, got %q", def.EventID, story.ID)
	}
	if story.Category != "" {
		t.Fatalf("expected empty category without source group, got %q", story.Category)
	}
	if math.Abs(silo.Cohesion-0.98) > 1e-9 {
		t.Fatalf("expected cohesion to become 0.98, got %v", silo.Cohesion)
	}
	if math.Abs(silo.Professions[0].PanicValue-0.14) > 1e-9 || math.Abs(silo.Professions[1].PanicValue-0.24) > 1e-9 {
		t.Fatalf("expected panic values to increase, got %+v", silo.Professions)
	}
	if CanTriggerContentEvent(def, ContentEvaluationContext{
		Silo: silo,
		Runtime: ContentEventRuntime{
			Triggered: true,
		},
	}) {
		t.Fatalf("expected once event not to re-trigger after being marked triggered")
	}
}

func TestCanTriggerCrisisEventAfterSpecialEvent(t *testing.T) {
	silo := &model.Silo{CurrentYear: 122, CurrentMonth: 1, Cohesion: 0.4}
	def := ContentEventDefinition{
		SourceGroup: "crisis",
		EventID:     "deep_unrest",
		Title:       "Deep Unrest",
		FireMode:    ContentFireModeOnce,
		Trigger: ContentTrigger{
			Type: "all",
			Conditions: []ContentTrigger{
				{Type: "event_triggered", Category: "special", EventID: "history_burden_awakened"},
				{Type: "silo_metric_lte", Metric: "cohesion", Value: 0.5},
			},
		},
	}
	if !CanTriggerContentEvent(def, ContentEvaluationContext{
		Silo: silo,
		States: map[string]ContentEventRuntime{
			"special:history_burden_awakened": {Triggered: true},
		},
	}) {
		t.Fatalf("expected crisis event to trigger after special event")
	}
}

func TestCanTriggerPlayerActionEvent(t *testing.T) {
	silo := &model.Silo{CurrentYear: 122, CurrentMonth: 1}
	action := model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "PROPAGANDA_BROADCAST"}
	def := ContentEventDefinition{
		SourceGroup: "player_actions",
		EventID:     "propaganda_broadcast",
		Title:       "Propaganda Broadcast",
		FireMode:    ContentFireModeRepeatable,
		PlayerAction: &ContentPlayerActionSpec{
			ID:                  "PROPAGANDA_BROADCAST",
			Label:               "Propaganda Broadcast",
			Scope:               "common",
			ActionType:          string(model.ActionPlayerEvent),
			TargetType:          "NONE",
			UnavailableBehavior: "disable",
		},
		Trigger: ContentTrigger{
			Type:   "player_action_is",
			Action: "PROPAGANDA_BROADCAST",
		},
	}
	if !CanTriggerContentEvent(def, ContentEvaluationContext{
		Silo:   silo,
		Action: &action,
	}) {
		t.Fatalf("expected player action event to trigger for matching action")
	}
	if !CanDisplayPlayerAction(def, ContentEvaluationContext{Silo: silo}) {
		t.Fatalf("expected player action to stay displayable before a concrete click action exists")
	}
	if ContentPlayerActionID(def) != "PROPAGANDA_BROADCAST" {
		t.Fatalf("expected player action id to come from metadata, got %q", ContentPlayerActionID(def))
	}
}

func TestCanTriggerContentEventWithScriptHook(t *testing.T) {
	def, err := parseContentEventJSDefinition(`defineEvent({
		id: "script_branch",
		title: "Script Branch",
		description: "desc",
		type: "social",
		fire_mode: "once",
		script: {
			canTrigger(ctx) {
				return ctx.silo_metrics.cohesion < 0.5 && !!ctx.states["special:history_burden_awakened"];
			}
		},
		effects: [siloMetricDelta("legitimacy", -0.02)]
	})`)
	if err != nil {
		t.Fatalf("parse js definition: %v", err)
	}
	def.ScriptSource = `defineEvent({
		id: "script_branch",
		title: "Script Branch",
		description: "desc",
		type: "social",
		fire_mode: "once",
		script: {
			canTrigger(ctx) {
				return ctx.silo_metrics.cohesion < 0.5 && !!ctx.states["special:history_burden_awakened"];
			}
		},
		effects: [siloMetricDelta("legitimacy", -0.02)]
	})`

	ok := CanTriggerContentEvent(def, ContentEvaluationContext{
		Silo: &model.Silo{
			CurrentYear:  122,
			CurrentMonth: 1,
			Cohesion:     0.4,
		},
		States: map[string]ContentEventRuntime{
			"special:history_burden_awakened": {Triggered: true},
		},
	})
	if !ok {
		t.Fatalf("expected script canTrigger hook to allow the event")
	}
}

func TestRunContentEventApplyScript(t *testing.T) {
	source := `defineEvent({
		id: "script_apply",
		title: "Script Apply",
		description: "desc",
		type: "social",
		fire_mode: "once",
		trigger: siloMetricGte("cohesion", 0.1),
		script: {
			apply(ctx) {
				if (ctx.silo_metrics.cohesion < 0.3) {
					return {
						effects: [siloMetricDelta("rebellion", 0.15)],
						emit: ["crisis:deep_unrest"]
					};
				}
				return {
					effects: [siloMetricDelta("legitimacy", -0.05)]
				};
			}
		},
		effects: [siloMetricDelta("cohesion", -0.02)]
	})`
	def, err := parseContentEventJSDefinition(source)
	if err != nil {
		t.Fatalf("parse js definition: %v", err)
	}
	def.ScriptSource = source

	result, err := RunContentEventApplyScript(def, ContentEvaluationContext{
		Silo: &model.Silo{
			CurrentYear:  122,
			CurrentMonth: 1,
			Cohesion:     0.2,
		},
	})
	if err != nil {
		t.Fatalf("run apply hook: %v", err)
	}
	if len(result.Effects) != 1 || result.Effects[0].Metric != "rebellion" {
		t.Fatalf("expected rebellion effect from script hook, got %+v", result.Effects)
	}
	if len(result.Emit) != 1 || result.Emit[0] != "crisis:deep_unrest" {
		t.Fatalf("expected deep unrest followup emit, got %+v", result.Emit)
	}
}

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

	eventJSON := `{
                "id": "history_burden_awakened",
                "title": "History Burden Awakened",
                "description": "desc",
                "type": "social",
                "fire_mode": "once",
                "trigger": {"type": "silo_metric_gte", "metric": "history_burden", "value": 0.08},
                "effects": [{"type": "silo_metric_delta", "metric": "cohesion", "value": -0.02}]
        }`
	historyJSON := `{
                "id": "archive_truth_broadcast",
                "title": "Archive Truth Broadcast",
                "description": "desc",
                "type": "external",
                "fire_mode": "once",
                "trigger": {"type": "timestamp_at_or_after", "year": 122, "month": 1},
                "effects": [{"type": "silo_metric_delta", "metric": "history_burden", "value": 0.08}]
        }`

	if err := os.WriteFile(filepath.Join(specialDir, "history_burden_awakened.json"), []byte(eventJSON), 0o644); err != nil {
		t.Fatalf("write event: %v", err)
	}
	if err := os.WriteFile(filepath.Join(historiesDir, "archive_truth_broadcast.json"), []byte(historyJSON), 0o644); err != nil {
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

	story := ApplyContentEvent(def, silo)
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

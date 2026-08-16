package service

import (
	"os"
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

func TestDiscoverContentDirectoriesGroupedLayout(t *testing.T) {
	root := t.TempDir()
	eventsRoot := filepath.Join(root, "events")
	if err := os.MkdirAll(filepath.Join(eventsRoot, "histories"), 0o755); err != nil {
		t.Fatalf("mkdir histories: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(eventsRoot, "special"), 0o755); err != nil {
		t.Fatalf("mkdir special: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(eventsRoot, "crisis"), 0o755); err != nil {
		t.Fatalf("mkdir crisis: %v", err)
	}

	dirs := discoverContentDirectories(root)
	if got := dirs["histories"]; got != filepath.Join(eventsRoot, "histories") {
		t.Fatalf("expected histories dir, got %q", got)
	}
	if got := dirs["special"]; got != filepath.Join(eventsRoot, "special") {
		t.Fatalf("expected special dir, got %q", got)
	}
	if got := dirs["crisis"]; got != filepath.Join(eventsRoot, "crisis") {
		t.Fatalf("expected crisis dir, got %q", got)
	}
}

func TestContentStateForDefinitionFallsBackByEventID(t *testing.T) {
	svc := &GameService{
		contentStates: map[string]model.ContentEventState{
			"events:history_burden_awakened": {
				EventKey:     "events:history_burden_awakened",
				Triggered:    true,
				TriggerCount: 1,
			},
		},
	}

	state, key := svc.contentStateForDefinition(engine.ContentEventDefinition{
		Key:     "special:history_burden_awakened",
		EventID: "history_burden_awakened",
	})
	if key != "events:history_burden_awakened" {
		t.Fatalf("expected legacy key fallback, got %q", key)
	}
	if !state.Triggered || state.TriggerCount != 1 {
		t.Fatalf("expected legacy state to be reused, got %+v", state)
	}
}

func TestContentEventsDispatchThroughEventBusByName(t *testing.T) {
	svc := &GameService{
		engine: engine.NewGameEngine(),
		silo: &model.Silo{
			CurrentYear:  122,
			CurrentMonth: 1,
			Cohesion:     0.4,
		},
		agent:         &model.Agent{},
		contentStates: map[string]model.ContentEventState{},
		contentDefinitions: []engine.ContentEventDefinition{
			{
				ID:          1,
				Key:         "special:history_burden_awakened",
				SourceGroup: "special",
				EventID:     "history_burden_awakened",
				Title:       "History Burden Awakened",
				Effects: []engine.ContentEffect{
					{Type: "silo_metric_delta", Metric: "legitimacy", Value: -0.05},
				},
			},
			{
				ID:          2,
				Key:         "crisis:deep_unrest",
				SourceGroup: "crisis",
				EventID:     "deep_unrest",
				Title:       "Deep Unrest",
				Trigger: engine.ContentTrigger{
					Type: "all",
					Conditions: []engine.ContentTrigger{
						{Type: "event_triggered", EventID: "history_burden_awakened"},
						{Type: "silo_metric_lte", Metric: "cohesion", Value: 0.5},
					},
				},
				Effects: []engine.ContentEffect{
					{Type: "silo_metric_delta", Metric: "cohesion", Value: -0.1},
				},
			},
		},
	}
	svc.bindContentEventHandlers()

	var stories []model.StoryEvent
	var logs []string
	fired := svc.emitContentEvent(svc.contentDefinitions[0], "TEST", engine.NewEventContext(), nil, &stories, &logs)
	if !fired {
		t.Fatalf("expected special event to be emitted through event bus")
	}
	if len(stories) != 2 {
		t.Fatalf("expected special event to wake crisis event, got %d stories", len(stories))
	}
	if stories[0].ID != "history_burden_awakened" || stories[1].ID != "deep_unrest" {
		t.Fatalf("unexpected story order: %+v", stories)
	}
	if !svc.contentStates["special:history_burden_awakened"].Triggered {
		t.Fatalf("expected special event state to be marked triggered")
	}
	if !svc.contentStates["crisis:deep_unrest"].Triggered {
		t.Fatalf("expected crisis event state to be marked triggered")
	}
	if svc.silo.Cohesion >= 0.4 {
		t.Fatalf("expected crisis effect to reduce cohesion, got %v", svc.silo.Cohesion)
	}
}

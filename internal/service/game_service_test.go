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

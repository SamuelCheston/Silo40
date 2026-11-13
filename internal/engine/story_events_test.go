package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestTriggerRandomEventEmptyPoolNoTrigger locks the empty-pool guard: the
// story pool is currently externalized to events/ and empty in-code, so a
// fully charged EventTrigger must not trigger (rng.Intn(0) panic) and the
// accumulated budget must be preserved for a future non-empty pool.
func TestTriggerRandomEventEmptyPoolNoTrigger(t *testing.T) {
	e := NewEventEngine()
	silo := &model.Silo{EventTrigger: 1.5}

	story := e.TriggerRandomEvent(silo, NewEventBus(), NewEventContext())
	if story != nil {
		t.Fatalf("expected no story from an empty pool, got %+v", story)
	}
	if silo.EventTrigger != 1.5 {
		t.Fatalf("expected EventTrigger budget to be preserved, got %v", silo.EventTrigger)
	}
}

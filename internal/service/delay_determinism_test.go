package service

import (
	"reflect"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestCheckDelayedEventsFiresInCreationOrder locks the deterministic dispatch
// order of the delay queue: tasks coming due in the same month must fire in
// first-scheduled-first-fired order regardless of map iteration order.
func TestCheckDelayedEventsFiresInCreationOrder(t *testing.T) {
	svc := NewGameService(nil)

	mkEvent := func(id, typ string) *engine.GameEvent {
		return engine.CreateEvent(id, typ, map[string]interface{}{})
	}
	// Insert out of creation order on purpose; map iteration order is
	// randomized anyway, the sort must win.
	svc.delayCounter = 3
	svc.delayTasks = map[string]*delayTask{
		"delay#3": {ID: "delay#3", Status: "scheduled", DelayMonths: 1, Event: mkEvent("e3", "AUDIT_T3")},
		"delay#1": {ID: "delay#1", Status: "scheduled", DelayMonths: 1, Event: mkEvent("e1", "AUDIT_T1")},
		"delay#2": {ID: "delay#2", Status: "scheduled", DelayMonths: 2, Event: mkEvent("e2", "AUDIT_T2")},
	}

	staged := svc.checkDelayedEventsStaged()
	svc.publishStagedDelayedEvents(staged, ActiveSiloID)

	// delay#1 and delay#3 are due after this pass; delay#2 still has a month.
	if got := svc.delayTasks["delay#2"].Status; got != "scheduled" {
		t.Fatalf("expected delay#2 to stay scheduled, got %q", got)
	}
	if got := svc.delayTasks["delay#2"].DelayMonths; got != 1 {
		t.Fatalf("expected delay#2 to have 1 month left, got %d", got)
	}

	processed := svc.engine.Bus.Drain()
	if len(processed) != 2 {
		t.Fatalf("expected 2 due events, got %d", len(processed))
	}
	if processed[0].ID != "e1" || processed[1].ID != "e3" {
		t.Fatalf("expected fire order [e1 e3] (creation order), got [%s %s]",
			processed[0].ID, processed[1].ID)
	}
	if svc.delayTasks["delay#1"].Status != "triggered" || svc.delayTasks["delay#3"].Status != "triggered" {
		t.Fatalf("expected due tasks to be marked triggered, got %+v", svc.delayTasks)
	}
}

// TestDelayedEventsSameMonthTwoRunsIdentical locks the reproducibility
// contract end to end: when several delayed content events come due in the
// same month, two runs with the same seed must produce the same fire order
// and the same resulting state. Before the delay-queue sort fix the shared
// RNG was consumed in a run-dependent order and long runs diverged.
func TestDelayedEventsSameMonthTwoRunsIdentical(t *testing.T) {
	scheduledKeys := []string{
		"crisis:blackout_cascade",
		"crisis:water_leak",
		"crisis:food_poisoning",
	}

	run := func(t *testing.T) ([]string, map[string]bool, model.Silo) {
		t.Helper()
		svc := bootstrapRealContent(t)
		if _, err := svc.CreateGame(model.CreateGameRequest{
			SiloName:   "DelayDeterminism",
			AgentName:  "Agent",
			Profession: "IT",
			Seed:       42,
		}); err != nil {
			t.Fatalf("create game: %v", err)
		}
		svc.bindContentEventHandlers()

		// Schedule three distinct content events with the same delay so they
		// all come due on the same monthly pass.
		for _, key := range scheduledKeys {
			event := engine.CreateEvent("scheduled#"+key, key, map[string]interface{}{
				"silo":           svc.silo,
				"agent":          svc.agent,
				"content_source": "SCHEDULED",
			})
			svc.scheduleDelayedEvent(event, 1, nil)
		}

		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time: %v", err)
		}

		fireOrder := make([]string, 0, len(svc.eventLogs))
		for _, entry := range svc.eventLogs {
			fireOrder = append(fireOrder, entry.Category+":"+entry.EventID)
		}
		triggered := make(map[string]bool)
		for key, state := range svc.contentStates[ActiveSiloID] {
			triggered[key] = state.Triggered
		}
		return fireOrder, triggered, *svc.silo
	}

	order1, triggered1, silo1 := run(t)
	order2, triggered2, silo2 := run(t)

	// 1. The three scheduled events must fire in creation order (relative to
	// each other) in both runs.
	filter := func(order []string) []string {
		out := []string{}
		want := map[string]bool{}
		for _, key := range scheduledKeys {
			want[key] = true
		}
		for _, entry := range order {
			if want[entry] {
				out = append(out, entry)
			}
		}
		return out
	}
	wantOrder := append([]string{}, scheduledKeys...)
	if got := filter(order1); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("run 1: expected scheduled events in creation order %v, got %v (full order %v)",
			wantOrder, got, order1)
	}
	if got := filter(order2); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("run 2: expected scheduled events in creation order %v, got %v (full order %v)",
			wantOrder, got, order2)
	}

	// 2. Both runs must be fully identical: event sequence, triggered content
	// state set, and resulting silo metrics.
	if !reflect.DeepEqual(order1, order2) {
		t.Fatalf("event history diverged between identical runs:\nrun1=%v\nrun2=%v", order1, order2)
	}
	if !reflect.DeepEqual(triggered1, triggered2) {
		t.Fatalf("triggered content states diverged between identical runs:\nrun1=%v\nrun2=%v",
			triggered1, triggered2)
	}
	for _, field := range []struct {
		name   string
		v1, v2 interface{}
	}{
		{"Cohesion", silo1.Cohesion, silo2.Cohesion},
		{"Legitimacy", silo1.Legitimacy, silo2.Legitimacy},
		{"Rebellion", silo1.Rebellion, silo2.Rebellion},
		{"DeptTension", silo1.DeptTension, silo2.DeptTension},
		{"TotalPopulation", silo1.TotalPopulation, silo2.TotalPopulation},
		{"EventTrigger", silo1.EventTrigger, silo2.EventTrigger},
	} {
		if !reflect.DeepEqual(field.v1, field.v2) {
			t.Fatalf("silo metric %s diverged between identical runs: %v vs %v",
				field.name, field.v1, field.v2)
		}
	}
}

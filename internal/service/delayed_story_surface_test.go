package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// scheduleDelayedContentForStation queues a real content event as a delayed
// (schedule) event targeting the player station, mirroring how
// scheduleContentEvents builds scheduled events.
func scheduleDelayedContentForStation(t *testing.T, svc *GameService, key string, delayMonths int) {
	t.Helper()
	def := findContentDefinitionByKey(t, svc, key)
	stepStories := []model.StoryEvent{}
	stepLogs := []string{}
	data := map[string]interface{}{
		"silo":            svc.silo,
		"agent":           svc.agent,
		"content_source":  "SCHEDULED",
		"step_story_sink": &stepStories,
		"step_log_sink":   &stepLogs,
	}
	event := engine.CreateEvent("scheduled#"+def.EventID, engine.ContentEventBusName(def), data)
	event.Data["step_stories"] = &stepStories
	event.Data["step_logs"] = &stepLogs
	svc.scheduleDelayedEvent(event, delayMonths, nil)
}

// TestDelayedContentStorySurfacesInMonthFeedback locks the P0-1 fix: a delayed
// (scheduled) content event that comes due during a multi-silo month MUST have
// its story appear in the returned month feedback, regardless of whether any
// NPC acts that month. Before the fix the delayed event lingered on the bus and
// was swallowed by an NPC SubmitAction's internal Drain, which discarded the
// story — so visibility depended on NPC behaviour.
func TestDelayedContentStorySurfacesInMonthFeedback(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(threeSiloRequest(40)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	// Schedule the canonical delayed follow-up (ultimatum aftermath) for the
	// station, due next month.
	scheduleDelayedContentForStation(t, svc, "special:ultimatum_aftermath", 1)

	result, err := svc.PassTime(1)
	if err != nil {
		t.Fatalf("pass time: %v", err)
	}

	found := false
	for _, story := range result.Stories {
		if story.ID == "ultimatum_aftermath" {
			found = true
			break
		}
	}
	if !found {
		titles := make([]string, 0, len(result.Stories))
		for _, story := range result.Stories {
			titles = append(titles, story.ID)
		}
		t.Fatalf("delayed content story ultimatum_aftermath missing from month feedback; got stories %v", titles)
	}

	// The event must also have actually applied (state changed) — the fix must
	// not merely surface the story but still settle the effect.
	if state, ok := svc.contentStates[ActiveSiloID]["special:ultimatum_aftermath"]; !ok || !state.Triggered {
		t.Fatalf("ultimatum_aftermath content state not triggered, got %+v", svc.contentStates[ActiveSiloID])
	}
}

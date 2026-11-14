package service

import (
	"testing"

	"silo40/internal/engine"
)

// TestCrossSignalStoryAppearsOnceInMonthFeedback locks the P0-2 fix: a
// cross-silo content event delivered to a single target silo must contribute
// exactly ONE story to the month feedback. Before the fix fireCrossEvent passed
// a story_sink to emitContentEvent while also harvesting step_stories from the
// drain, so handleContentEvent wrote the same story into both and the player
// saw it twice (the persisted history was already single).
func TestCrossSignalStoryAppearsOnceInMonthFeedback(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(threeSiloRequest(40)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

	// Deliver a cross-only ONCE event to the player silo with no delay, so it is
	// due during the next monthly pass.
	svc.enqueueCrossSignal(engine.CrossSignalRequest{
		SourceSilo:  3,
		TargetScope: "silo:player",
		EventID:     "silo1_ultimatum_cross",
		DelayMonths: 0,
	})

	result, err := svc.PassTime(1)
	if err != nil {
		t.Fatalf("pass time: %v", err)
	}

	count := 0
	for _, story := range result.Stories {
		if story.ID == "silo1_ultimatum_cross" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected cross story silo1_ultimatum_cross exactly once in month feedback, got %d", count)
	}
}

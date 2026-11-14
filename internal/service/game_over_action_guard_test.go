package service

import (
	"testing"

	"silo40/internal/model"
)

// TestActionsRejectedAfterGameOver locks the P0-4 fix: once the station has a
// victory status (game over), the server must refuse player actions on both the
// synchronous (ExecuteAction) and queued (BeginAction) paths. Before the fix the
// month was frozen but actions were still accepted, deducting AP and granting
// fragments after the ending.
func TestActionsRejectedAfterGameOver(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(threeSiloRequest(40)); err != nil {
		t.Fatalf("create game: %v", err)
	}

	action := model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 10}

	// Sanity: the same action is accepted before the ending.
	if _, err := svc.ExecuteAction(action); err != nil {
		t.Fatalf("action before game over should succeed, got %v", err)
	}

	// Force the ending.
	svc.silo.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "INFORMATION"}
	if !svc.gameOver() {
		t.Fatalf("expected gameOver() to be true once VictoryStatus is set")
	}

	if _, err := svc.ExecuteAction(action); err == nil {
		t.Fatalf("ExecuteAction must be rejected after game over")
	}
	if _, err := svc.BeginAction(action); err == nil {
		t.Fatalf("BeginAction must be rejected after game over")
	}
}

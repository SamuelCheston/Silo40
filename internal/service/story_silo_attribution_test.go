package service

import (
	"strings"
	"testing"

	"silo40/internal/engine"
)

// TestMonthlyStoriesCarrySiloAttribution locks the P0-3 fix: every story in the
// month feedback of a multi-silo game must carry its source-silo attribution
// (SiloID + SiloName), so the player can tell which silo an event happened in
// when several silos fire identical content in the same month.
func TestMonthlyStoriesCarrySiloAttribution(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(threeSiloRequest(40)); err != nil {
		t.Fatalf("create game: %v", err)
	}

	siloName := map[uint]string{}
	for _, silo := range svc.silos {
		siloName[silo.ID] = silo.Name
	}

	collected := 0
	for m := 0; m < 6; m++ {
		result, err := svc.PassTime(1)
		if err != nil {
			t.Fatalf("pass time month %d: %v", m, err)
		}
		for _, story := range result.Stories {
			collected++
			if story.SiloID == 0 {
				t.Fatalf("story %q (%s) missing SiloID attribution", story.Title, story.ID)
			}
			if story.SiloName == "" {
				t.Fatalf("story %q (%s) missing SiloName attribution", story.Title, story.ID)
			}
			// 同月多堡合并条目以顿号列出全部来源堡；单堡条目必须与 SiloID 精确一致。
			if want := siloName[story.SiloID]; want != "" && !strings.Contains(story.SiloName, want) {
				t.Fatalf("story %q SiloName %q does not attribute SiloID %d (%q)", story.Title, story.SiloName, story.SiloID, want)
			}
		}
		if svc.gameOver() {
			break
		}
	}
	if collected == 0 {
		t.Fatalf("expected at least one story across six months, got none")
	}
}

// TestCrossSignalStoryAttributedToTargetSilo locks that a cross-delivered story
// is attributed to the silo that received it (the target), not left blank.
func TestCrossSignalStoryAttributedToTargetSilo(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.CreateGame(threeSiloRequest(40)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.bindContentEventHandlers()

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

	station := svc.stationSilo()
	for _, story := range result.Stories {
		if story.ID == "silo1_ultimatum_cross" {
			if story.SiloID != station.ID || story.SiloName != station.Name {
				t.Fatalf("cross story attributed to silo %d/%q, want station %d/%q",
					story.SiloID, story.SiloName, station.ID, station.Name)
			}
			return
		}
	}
	t.Fatalf("cross story silo1_ultimatum_cross not found in month feedback")
}

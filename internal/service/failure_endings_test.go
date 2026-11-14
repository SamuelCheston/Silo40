package service

import (
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// pickContentDef extracts one real definition from the loaded content set so
// failure-line tests run the shipped files, not fixtures.
func pickContentDef(t *testing.T, boot *GameService, key string) engine.ContentEventDefinition {
	t.Helper()
	for _, def := range boot.contentDefinitions {
		if def.Key == key {
			return def
		}
	}
	t.Fatalf("definition %q not loaded", key)
	return engine.ContentEventDefinition{}
}

// TestExclusionistEndingNarrativeAndVariants locks 29-failure-endings §3: the
// EXCLUSIONIST settlement tells the story of the inertial slide (not a score
// formula), branches into the two canon variants (Silo 1 alive / post-fall
// replication), and reports what still runs at settlement.
func TestExclusionistEndingNarrativeAndVariants(t *testing.T) {
	newStation := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("EXCLUSIONIST")
		// Closed dominance: population-weighted loyalty above pro-foreign.
		station.Cohorts = []model.PopulationCohort{
			{ID: 1, Count: 100, Ideologies: map[string]float64{
				model.IdeologyLoyalty: 0.7, model.IdeologyProForeign: 0.2,
			}},
		}
		station.TotalPopulation = 100
		station.SetMayorLegitimacy(0.55)
		svc.agent.PoliticalPrestige = 100
		return svc, station
	}

	// Variant A: Silo 1 still alive — the closed order keeps its external
	// support and the new supervisor inherits the old ritual.
	svc, station := newStation()
	station.Silo1Destroyed = false
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingExclusionist {
		t.Fatalf("expected EXCLUSIONIST selection, got %+v", decision)
	}
	if strings.Contains(decision.Reason, "P2 加权") {
		t.Fatalf("score formula still on the ending screen: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "《深闭》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "新一任 IT 主管在旧仪式中就职") {
		t.Fatalf("alive-Silo-1 variant missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "仍有居民 100 人，正统性 0.55") {
		t.Fatalf("settlement line missing: %q", decision.Reason)
	}

	// Variant B: Silo 1 fallen — the order has internalized its master.
	svc2, station2 := newStation()
	svc2.world.Silo1.Alive = false
	station2.Silo1Destroyed = true
	decision2 := svc2.resolveWeightedEnding(station2, svc2.agent)
	if decision2 == nil || decision2.Selected != endingExclusionist {
		t.Fatalf("expected EXCLUSIONIST after the fall, got %+v", decision2)
	}
	if !strings.Contains(decision2.Reason, "内化了它的主人") {
		t.Fatalf("post-fall replication variant missing: %q", decision2.Reason)
	}
}

// TestResetEndingNarrative locks 29-failure-endings §4: the RESET settlement
// tells the formatting story (what went wrong: the omens were read as noise)
// and reports what remains — the people survive, the memory does not.
func TestResetEndingNarrative(t *testing.T) {
	svc, station, _ := weightedEndingFixture("RESET")
	station.VictoryStatus.IsWon = false
	station.TotalPopulation = 5200
	station.HistoryBurden = 0.02
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {resetFactKey: {SiloID: station.ID, EventKey: resetFactKey, Triggered: true}},
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingReset {
		t.Fatalf("expected RESET selection, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "《遗忘》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "被当成了噪音") {
		t.Fatalf("what-went-wrong beat missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "仍有居民 5200 人，历史包袱降至 0.02") {
		t.Fatalf("post-failure settlement line missing: %q", decision.Reason)
	}
}

// TestNormalcyHardensFires locks the EXCLUSIONIST omen (29-failure-endings
// §2.1): from relative Y3 on, once four departments' loyalty stands above the
// public-dominance bar, the hardening heartbeat lights and records that
// another year passed without change.
func TestNormalcyHardensFires(t *testing.T) {
	boot := bootstrapRealContent(t)
	var def = pickContentDef(t, boot, "special:normalcy_hardens")
	svc := chainTestService(def)

	svc.silo.CurrentYear, svc.silo.CurrentMonth = 125, 2
	svc.silo.HistoryBurden = 0.10
	svc.silo.Rebellion = 0.10
	for i := 1; i <= 4; i++ {
		svc.silo.Professions = append(svc.silo.Professions, model.Profession{
			ID: uint(i), SiloID: svc.silo.ID, Name: string(rune('A' + i)),
			Ideologies: map[string]float64{"loyalty": 0.7},
		})
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	if !contentTriggered(svc, "special:normalcy_hardens") {
		t.Fatal("normalcy_hardens did not fire under closed dominance")
	}
	if svc.silo.HistoryBurden != 0.08 || svc.silo.Rebellion != 0.08 {
		t.Fatalf("hardening effects not applied: hb=%v rebellion=%v",
			svc.silo.HistoryBurden, svc.silo.Rebellion)
	}

	// Negative side: before the Y3 time point nothing fires.
	svc2 := chainTestService(pickContentDef(t, boot, "special:normalcy_hardens"))
	svc2.silo.CurrentYear, svc2.silo.CurrentMonth = 124, 6
	svc2.silo.Professions = svc.silo.Professions
	svc2.queueSystemContentEvents("TEST")
	svc2.drainQueuedEvents()
	if contentTriggered(svc2, "special:normalcy_hardens") {
		t.Fatal("normalcy_hardens fired before its time point")
	}
}

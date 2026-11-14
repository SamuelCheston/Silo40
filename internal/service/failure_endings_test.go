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

// TestFractureEndingNarrative locks 29-failure-endings §5: the FRACTURE
// settlement tells the self-destruction story (anger without organization)
// and reports what is left — people, energy, floors: technically alive,
// tomorrow gone. The omen side is already covered by the shipped chain tests
// (fracture_infrastructure precedes fracture_confirmed).
func TestFractureEndingNarrative(t *testing.T) {
	svc, station, _ := weightedEndingFixture("FRACTURE")
	station.VictoryStatus.IsWon = false
	station.TotalPopulation = 2400
	station.Resources = []model.Resource{{Type: "Energy", Amount: 120}}
	station.Floors = []model.Floor{
		{Zone: "Upper", Stability: 0.5},
		{Zone: "Lower", Stability: 0.2},
	}
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {fractureFactKey: {SiloID: station.ID, EventKey: fractureFactKey, Triggered: true}},
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingFracture {
		t.Fatalf("expected FRACTURE selection, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "《断层》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "期待跑得比组织快") {
		t.Fatalf("what-went-wrong beat missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "仍有居民 2400 人，剩余能源 120，楼层平均稳定度 0.35") {
		t.Fatalf("post-failure settlement line missing: %q", decision.Reason)
	}
}

// TestCompromisedEndingNarrative locks 29-failure-endings §6: the
// AGENT_COMPROMISED settlement tells the exposure story (the traces were
// there from the start; the erasure is packaged under a local pretext) and
// reports that the station goes on — the bitter-epilogue variant.
func TestCompromisedEndingNarrative(t *testing.T) {
	svc, station, _ := weightedEndingFixture("AGENT_COMPROMISED")
	station.VictoryStatus.IsWon = false
	station.TotalPopulation = 5600
	station.Professions = []model.Profession{
		{ID: 1, Name: "Judicial", Beliefs: []model.FactBelief{{FactID: "Judicial_1", B: 0.5}}},
		{ID: 2, Name: "IT", Beliefs: []model.FactBelief{{FactID: "IT_1", B: 0.4}, {FactID: "IT_2", B: 0.1}}},
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingCompromised {
		t.Fatalf("expected AGENT_COMPROMISED selection, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "《除名》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "没有人会知道你真正是什么") {
		t.Fatalf("local-pretext staging constraint missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "5600 名居民，2 个部门仍在履职，部门信念层里仍有 2 份可被引用的事实") {
		t.Fatalf("post-failure settlement line missing: %q", decision.Reason)
	}
}

// TestCompromisedOmenChainFires locks the three designed exposure omens
// (20-chain-agent-compromised §1): covert-operation traces accumulate, the
// archive review opens at the fifth trace, and the erasure staging lights as
// suspicion approaches the cap — all before the failure itself.
func TestCompromisedOmenChainFires(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(
		pickContentDef(t, boot, "special:identity_trace_accumulates"),
		pickContentDef(t, boot, "special:archive_review_opened"),
		pickContentDef(t, boot, "special:agent_erasure_staged"),
	)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 123, 1
	svc.agent.SuspicionLevel = 0.75
	// A covert operation completed earlier; its completion key persists.
	svc.statesMapFor(svc.silo.ID)["player_action:STATION_INFILTRATE"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "player_action:STATION_INFILTRATE", Triggered: true, TriggerCount: 1,
	}

	// Eleven months: the trace heartbeat (cd 2) fires at least five times and
	// both follower omens pick the count/suspicion up via continuous
	// evaluation.
	for i := 0; i < 11; i++ {
		svc.silo.CurrentMonth++
		if svc.silo.CurrentMonth > 12 {
			svc.silo.CurrentMonth = 1
			svc.silo.CurrentYear++
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
	}

	if count := svc.contentStates[svc.silo.ID]["special:identity_trace_accumulates"].TriggerCount; count < 5 {
		t.Fatalf("identity traces accumulated %d times, want >= 5", count)
	}
	for _, key := range []string{"special:archive_review_opened", "special:agent_erasure_staged"} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s never fired before the exposure cap", key)
		}
	}
}

// TestDeathEndingNarrativeVariants locks 29-failure-endings §7: the DEATH
// settlement names the cause (which step went wrong), keeps the restrained
// final moment, and reports what remains — the escape attempt that did not
// survive, the outside world, the facility.
func TestDeathEndingNarrativeVariants(t *testing.T) {
	// Variant A: remote termination (Silo 1 settlement fact), Silo 1 alive.
	svc, station, _ := weightedEndingFixture("TIME")
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {terminationFactKey: {SiloID: station.ID, EventKey: terminationFactKey, Triggered: true}},
	}
	station.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 40, Surviving: false}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingDeath {
		t.Fatalf("expected DEATH selection, got %+v", decision)
	}
	for _, want := range []string{
		"终止来自控制网的另一端",
		"《寂堡》成立",
		"曾有 40 人离开，但他们没能存续到最后",
		"这座驻地被标记为已处理",
	} {
		if !strings.Contains(decision.Reason, want) {
			t.Fatalf("termination variant missing %q: %q", want, decision.Reason)
		}
	}

	// Variant B: safeguard ceiling — the station's own last device.
	svc2, station2, _ := weightedEndingFixture("DEATH")
	station2.VictoryStatus.IsWon = false
	station2.SafeguardRisk = 1.0
	decision2 := svc2.resolveWeightedEnding(station2, svc2.agent)
	if decision2 == nil || decision2.Selected != endingDeath {
		t.Fatalf("expected DEATH by safeguard, got %+v", decision2)
	}
	if !strings.Contains(decision2.Reason, "保障机制在冲突中被触发") {
		t.Fatalf("safeguard cause missing: %q", decision2.Reason)
	}

	// Variant C: sustenance silence — population exhausted, no war.
	svc3, station3, _ := weightedEndingFixture("DEATH")
	station3.VictoryStatus.IsWon = false
	decision3 := svc3.resolveWeightedEnding(station3, svc3.agent)
	if decision3 == nil || decision3.Selected != endingDeath {
		t.Fatalf("expected DEATH by exhaustion, got %+v", decision3)
	}
	if !strings.Contains(decision3.Reason, "最后一班没有人接") {
		t.Fatalf("sustenance-silence cause missing: %q", decision3.Reason)
	}
	if !strings.Contains(decision3.Reason, "驻地仍然矗立") {
		t.Fatalf("facility-remains line missing: %q", decision3.Reason)
	}
}

// TestSafeguardProtocolWarmingFires locks the DEATH safeguard omen
// (29-failure-endings §2.5): the warning lights on the existing monthly
// channel once safeguard risk crosses the bar, and stays silent below it.
func TestSafeguardProtocolWarmingFires(t *testing.T) {
	boot := bootstrapRealContent(t)

	svc := chainTestService(pickContentDef(t, boot, "special:safeguard_protocol_warming"))
	svc.silo.SafeguardRisk = 0.65
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:safeguard_protocol_warming") {
		t.Fatal("safeguard omen did not fire above the risk bar")
	}

	svc2 := chainTestService(pickContentDef(t, boot, "special:safeguard_protocol_warming"))
	svc2.silo.SafeguardRisk = 0.3
	svc2.queueSystemContentEvents("TEST")
	svc2.drainQueuedEvents()
	if contentTriggered(svc2, "special:safeguard_protocol_warming") {
		t.Fatal("safeguard omen fired below the risk bar")
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

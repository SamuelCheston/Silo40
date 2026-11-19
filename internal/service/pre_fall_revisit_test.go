package service

import (
	"testing"

	"silo40/internal/model"
)

// 抉择延迟回访验收（覆灭前夜批次）：三选一不是快照投影完就结束，
// 三个月后按所选分支回来算账。时序硬墙（晚于 Y7M7 的选择追不进
// Y8M1 结算）见 pre_fall_echo.js 注释，不为此放宽延迟。

func TestPreFallEchoDefinitionLoads(t *testing.T) {
	svc := bootstrapRealContent(t)
	found := false
	for _, def := range svc.contentDefinitions {
		if def.Key == "special:pre_fall_echo" {
			found = true
		}
	}
	if !found {
		t.Fatal("special:pre_fall_echo not loaded")
	}
}

func TestPreFallEchoBranches(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:pre_fall_echo")

	newRun := func(actionID string) *GameService {
		svc := chainTestService(defs...)
		svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
		svc.silo.Rebellion = 0.1
		svc.silo.HistoryBurden = 0.1
		svc.silo.Cohorts = []model.PopulationCohort{
			{ID: 1, SiloID: svc.silo.ID, Ideologies: map[string]float64{"loyalty": 0.2}},
		}
		svc.silo.Floors = []model.Floor{
			{ID: 1, SiloID: svc.silo.ID, Level: 70, Zone: "Mid", Stability: 0.9},
		}
		seedChoiceCompletion(svc, actionID, 122, 2)
		return svc
	}

	// 忠诚动员：队列惯性继续凝成忠诚，叛乱再压一档。
	loyal := newRun("PRE_FALL_LOYALTY_MOBILIZATION")
	advanceSweepMonths(loyal, 3)
	if !contentTriggered(loyal, "special:pre_fall_echo") {
		t.Fatal("pre_fall_echo never fired on the loyalty branch")
	}
	if got := loyal.silo.Cohorts[0].Ideologies["loyalty"]; !charterRevisitAlmostEqual(got, 0.25) {
		t.Fatalf("loyalty ideology = %v, want 0.25", got)
	}
	if !charterRevisitAlmostEqual(loyal.silo.Rebellion, 0.08) {
		t.Fatalf("loyalty branch rebellion = %v, want 0.08", loyal.silo.Rebellion)
	}

	// 公开质询：为公共讨论担的风险继续到期。
	inquiry := newRun("PRE_FALL_OPEN_INQUIRY")
	advanceSweepMonths(inquiry, 3)
	if !contentTriggered(inquiry, "special:pre_fall_echo") {
		t.Fatal("pre_fall_echo never fired on the inquiry branch")
	}
	if !charterRevisitAlmostEqual(inquiry.silo.HistoryBurden, 0.13) {
		t.Fatalf("inquiry branch history_burden = %v, want 0.13", inquiry.silo.HistoryBurden)
	}

	// 全面暴乱：维存损耗的第二期。
	riot := newRun("PRE_FALL_TOTAL_RIOT")
	advanceSweepMonths(riot, 3)
	if !contentTriggered(riot, "special:pre_fall_echo") {
		t.Fatal("pre_fall_echo never fired on the riot branch")
	}
	if got := riot.silo.Floors[0].Stability; !charterRevisitAlmostEqual(got, 0.85) {
		t.Fatalf("riot branch floor stability = %v, want 0.85", got)
	}

	// 延迟纪律：两个月时不得提前回来。
	early := newRun("PRE_FALL_OPEN_INQUIRY")
	advanceSweepMonths(early, 2)
	if contentTriggered(early, "special:pre_fall_echo") {
		t.Fatal("pre_fall_echo fired before the 3-month delay")
	}
}

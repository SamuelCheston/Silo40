package service

import (
	"testing"

	"silo40/internal/model"
)

// 抉择延迟回访验收（撤离窗口批次）：有序占用的红利与全面暴乱的账单
// 都在三个月后到期——后者兑现 escape_window_riot 文案「分两次到期」的承诺。

func TestEscapeRevisitDefinitionsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	byKey := map[string]bool{}
	for _, def := range svc.contentDefinitions {
		byKey[def.Key] = true
	}
	for _, key := range []string{"special:escape_window_orderly_yield", "special:escape_window_riot_bill"} {
		if !byKey[key] {
			t.Fatalf("%s not loaded", key)
		}
	}
}

// 有序回访（+3 月）：秩序被反复看见，恐慌回落、凝聚上升。
func TestEscapeOrderlyYield(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:escape_window_orderly_yield")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Cohesion = 0.5
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Mechanical", Population: 200, PanicValue: 0.1},
	}
	seedChoiceCompletion(svc, "ESCAPE_WINDOW_ORDERLY", 122, 2)

	advanceSweepMonths(svc, 2)
	if contentTriggered(svc, "special:escape_window_orderly_yield") {
		t.Fatal("orderly yield fired before the 3-month delay")
	}
	advanceSweepMonths(svc, 1)
	if !contentTriggered(svc, "special:escape_window_orderly_yield") {
		t.Fatal("escape_window_orderly_yield never fired")
	}
	if got := svc.silo.Professions[0].PanicValue; !charterRevisitAlmostEqual(got, 0.08) {
		t.Fatalf("panic = %v, want 0.08", got)
	}
	if !charterRevisitAlmostEqual(svc.silo.Cohesion, 0.51) {
		t.Fatalf("cohesion = %v, want 0.51", svc.silo.Cohesion)
	}
}

// 暴乱回访（+3 月）：账单第二期——全楼层结构稳定再降，恐慌余波未平。
func TestEscapeRiotBill(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:escape_window_riot_bill")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Floors = []model.Floor{
		{ID: 1, SiloID: svc.silo.ID, Level: 10, Zone: "Upper", Stability: 0.9},
		{ID: 2, SiloID: svc.silo.ID, Level: 70, Zone: "Mid", Stability: 0.9},
		{ID: 3, SiloID: svc.silo.ID, Level: 130, Zone: "Lower", Stability: 0.9},
	}
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Mechanical", Population: 200, PanicValue: 0.1},
	}
	seedChoiceCompletion(svc, "ESCAPE_WINDOW_RIOT", 122, 2)

	advanceSweepMonths(svc, 3)
	if !contentTriggered(svc, "special:escape_window_riot_bill") {
		t.Fatal("escape_window_riot_bill never fired")
	}
	for _, floor := range svc.silo.Floors {
		if !charterRevisitAlmostEqual(floor.Stability, 0.86) {
			t.Fatalf("floor %d stability = %v, want 0.86 (all floors billed)", floor.Level, floor.Stability)
		}
	}
	if got := svc.silo.Professions[0].PanicValue; !charterRevisitAlmostEqual(got, 0.12) {
		t.Fatalf("panic = %v, want 0.12", got)
	}
	// ONCE：再扫不重复计账。
	advanceSweepMonths(svc, 3)
	if state := svc.statesMapFor(svc.silo.ID)["special:escape_window_riot_bill"]; state.TriggerCount != 1 {
		t.Fatalf("riot bill must fire once, count = %d", state.TriggerCount)
	}
}

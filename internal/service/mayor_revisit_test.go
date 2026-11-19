package service

import (
	"testing"
)

// 抉择延迟回访验收（市长批次）：接受的同盟有价码，绕过的漂移有归因。

func TestMayorRevisitDefinitionsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	byKey := map[string]bool{}
	for _, def := range svc.contentDefinitions {
		byKey[def.Key] = true
	}
	for _, key := range []string{"special:mayor_ally_price", "special:mayor_bypass_drift"} {
		if !byKey[key] {
			t.Fatalf("%s not loaded", key)
		}
	}
}

// 接受回访（+4 月）：同盟的代价记在部门张力上。
func TestMayorAllyPriceEcho(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:mayor_ally_price")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.DeptTension = 0.1
	seedChoiceCompletion(svc, "MAYOR_ALLY_ACCEPT", 122, 2)

	advanceSweepMonths(svc, 3)
	if contentTriggered(svc, "special:mayor_ally_price") {
		t.Fatal("mayor_ally_price fired before the 4-month delay")
	}
	advanceSweepMonths(svc, 1)
	if !contentTriggered(svc, "special:mayor_ally_price") {
		t.Fatal("mayor_ally_price never fired")
	}
	if !charterRevisitAlmostEqual(svc.silo.DeptTension, 0.12) {
		t.Fatalf("dept_tension = %v, want 0.12", svc.silo.DeptTension)
	}
}

// 绕过回访（+6 月）：世界明确引用这次绕过，惯性漂移一次性归因。
func TestMayorBypassDriftEcho(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:mayor_bypass_drift")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.HistoryBurden = 0.1
	svc.silo.Rebellion = 0.1
	seedChoiceCompletion(svc, "MAYOR_ALLY_BYPASS", 122, 2)

	advanceSweepMonths(svc, 6)
	if !contentTriggered(svc, "special:mayor_bypass_drift") {
		t.Fatal("mayor_bypass_drift never fired")
	}
	if !charterRevisitAlmostEqual(svc.silo.HistoryBurden, 0.08) {
		t.Fatalf("history_burden = %v, want 0.08", svc.silo.HistoryBurden)
	}
	if !charterRevisitAlmostEqual(svc.silo.Rebellion, 0.09) {
		t.Fatalf("rebellion = %v, want 0.09", svc.silo.Rebellion)
	}
	// ONCE：再扫不重复计账。
	advanceSweepMonths(svc, 3)
	if state := svc.statesMapFor(svc.silo.ID)["special:mayor_bypass_drift"]; state.TriggerCount != 1 {
		t.Fatalf("bypass drift must fire once, count = %d", state.TriggerCount)
	}
}
package service

import (
	"testing"

	"silo40/internal/model"
)

// 抉择延迟回访验收（谈判批次）：妥协成立与信道破裂两类非动作结局在 +3 月
// 回来算账；妥协后的接管/带组织离开/悬置三种状态各有不同后果，悬置不是
// 死胡同——世界明确引用这次沉默（保障可信度磨损）。

// seedNegotiationOutcome 点亮一个谈判结局完成键（与玩家动作完成键同口径）。
func seedNegotiationOutcome(svc *GameService, key string, year, month int) {
	svc.statesMapFor(svc.silo.ID)[key] = model.ContentEventState{
		SiloID:                 svc.silo.ID,
		EventKey:               key,
		Triggered:              true,
		TriggerCount:           1,
		LastTriggeredTimestamp: gameTimestamp(year, month),
	}
}

func TestNegotiationRevisitDefinitionsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	byKey := map[string]bool{}
	for _, def := range svc.contentDefinitions {
		byKey[def.Key] = true
	}
	for _, key := range []string{"special:negotiation_compromise_weight", "special:negotiation_rupture_echo"} {
		if !byKey[key] {
			t.Fatalf("%s not loaded", key)
		}
	}
}

// 妥协回访（+3 月）·接管分支：旧班底的不安记在部门张力上。
func TestNegotiationCompromiseTakeoverWeight(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:negotiation_compromise_weight")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.DeptTension = 0.1
	svc.silo.Negotiation.CompromiseEstablished = true
	svc.silo.Negotiation.Takeover = model.NegotiationTakeover
	seedNegotiationOutcome(svc, negotiationCompromiseKey, 122, 2)

	advanceSweepMonths(svc, 2)
	if contentTriggered(svc, "special:negotiation_compromise_weight") {
		t.Fatal("compromise weight fired before the 3-month delay")
	}
	advanceSweepMonths(svc, 1)
	if !contentTriggered(svc, "special:negotiation_compromise_weight") {
		t.Fatal("compromise weight never fired")
	}
	if !charterRevisitAlmostEqual(svc.silo.DeptTension, 0.12) {
		t.Fatalf("dept_tension = %v, want 0.12", svc.silo.DeptTension)
	}
}

// 妥协回访（+3 月）·带组织离开分支：组织完整转向，凝聚小量上行。
func TestNegotiationCompromiseHandsOffWeight(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:negotiation_compromise_weight")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Cohesion = 0.5
	svc.silo.Negotiation.CompromiseEstablished = true
	svc.silo.Negotiation.Takeover = model.NegotiationHandsOff
	seedNegotiationOutcome(svc, negotiationCompromiseKey, 122, 2)

	advanceSweepMonths(svc, 3)
	if !contentTriggered(svc, "special:negotiation_compromise_weight") {
		t.Fatal("compromise weight never fired")
	}
	if !charterRevisitAlmostEqual(svc.silo.Cohesion, 0.52) {
		t.Fatalf("cohesion = %v, want 0.52", svc.silo.Cohesion)
	}
}

// 妥协回访（+3 月）·悬置分支：承诺无人认领，保障可信度磨损。
func TestNegotiationCompromiseStalledWeight(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:negotiation_compromise_weight")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.TopStance.Initialized = true
	svc.silo.TopStance.Guarantee = 0.4
	svc.silo.Negotiation.CompromiseEstablished = true
	// Takeover 留空：妥协成立但一直未表态。
	seedNegotiationOutcome(svc, negotiationCompromiseKey, 122, 2)

	advanceSweepMonths(svc, 3)
	if !contentTriggered(svc, "special:negotiation_compromise_weight") {
		t.Fatal("compromise weight never fired")
	}
	if !charterRevisitAlmostEqual(svc.silo.TopStance.Guarantee, 0.35) {
		t.Fatalf("guarantee = %v, want 0.35", svc.silo.TopStance.Guarantee)
	}
	// ONCE：再扫不重复计账。
	advanceSweepMonths(svc, 3)
	if state := svc.statesMapFor(svc.silo.ID)["special:negotiation_compromise_weight"]; state.TriggerCount != 1 {
		t.Fatalf("compromise weight must fire once, count = %d", state.TriggerCount)
	}
}

// 破裂回访（+3 月）：断线的回声一次性计入全部门恐慌与叛乱。
func TestNegotiationRuptureEcho(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:negotiation_rupture_echo")
	svc := chainTestService(defs...)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
	svc.silo.Rebellion = 0.1
	svc.silo.Professions = []model.Profession{
		{ID: 1, SiloID: svc.silo.ID, Name: "Mechanical", Population: 200, PanicValue: 0.1},
	}
	seedNegotiationOutcome(svc, negotiationRuptureKey, 122, 2)

	advanceSweepMonths(svc, 2)
	if contentTriggered(svc, "special:negotiation_rupture_echo") {
		t.Fatal("rupture echo fired before the 3-month delay")
	}
	advanceSweepMonths(svc, 1)
	if !contentTriggered(svc, "special:negotiation_rupture_echo") {
		t.Fatal("rupture echo never fired")
	}
	if got := svc.silo.Professions[0].PanicValue; !charterRevisitAlmostEqual(got, 0.12) {
		t.Fatalf("panic = %v, want 0.12", got)
	}
	if !charterRevisitAlmostEqual(svc.silo.Rebellion, 0.12) {
		t.Fatalf("rebellion = %v, want 0.12", svc.silo.Rebellion)
	}
	// ONCE：再扫不重复计账。
	advanceSweepMonths(svc, 3)
	if state := svc.statesMapFor(svc.silo.ID)["special:negotiation_rupture_echo"]; state.TriggerCount != 1 {
		t.Fatalf("rupture echo must fire once, count = %d", state.TriggerCount)
	}
}

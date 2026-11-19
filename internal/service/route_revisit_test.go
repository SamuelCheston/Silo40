package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 抉择延迟回访验收（《越岭》批次）：接受后按勘察事实分流算账；
// 拒绝/暂缓的拖延付出真实暴露代价，并在第二扇窗里换条件重开选择。

// 内容加载：两个回访事件与再接受动作必须随真实目录入库。
func TestRouteRevisitDefinitionsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	byKey := map[string]engine.ContentEventDefinition{}
	for _, def := range svc.contentDefinitions {
		byKey[def.Key] = def
	}
	for _, key := range []string{"special:route_accept_cost_due", "special:route_second_window"} {
		if _, ok := byKey[key]; !ok {
			t.Fatalf("%s not loaded", key)
		}
	}
	def, ok := byKey["player_actions:route_survey_reaccept"]
	if !ok || def.PlayerAction == nil || def.PlayerAction.ID != "ROUTE_SURVEY_REACCEPT" {
		t.Fatalf("route_survey_reaccept missing or wrong action id: %+v", def.PlayerAction)
	}
}

// 接受回访（+5 月）：勘察过付余波，没去付恐慌。
func TestRouteAcceptCostDueBranches(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot,
		"special:route_accept_cost_due", "special:route_survey_step")

	newRun := func() *GameService {
		svc := chainTestService(defs...)
		svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
		svc.silo.Rebellion = 0.1
		svc.silo.Professions = []model.Profession{
			{ID: 1, SiloID: svc.silo.ID, Name: "Mechanical", Population: 200, PanicValue: 0.1},
		}
		seedChoiceCompletion(svc, "ROUTE_SURVEY_ACCEPT", 122, 2)
		return svc
	}

	surveyed := newRun()
	surveyed.statesMapFor(surveyed.silo.ID)["special:route_survey_step"] = model.ContentEventState{
		SiloID: surveyed.silo.ID, EventKey: "special:route_survey_step", Triggered: true, TriggerCount: 1,
	}
	advanceSweepMonths(surveyed, 5)
	if !contentTriggered(surveyed, "special:route_accept_cost_due") {
		t.Fatal("route_accept_cost_due never fired on the surveyed branch")
	}
	if !charterRevisitAlmostEqual(surveyed.silo.Rebellion, 0.11) {
		t.Fatalf("surveyed branch rebellion = %v, want 0.11", surveyed.silo.Rebellion)
	}

	idle := newRun()
	advanceSweepMonths(idle, 5)
	if !contentTriggered(idle, "special:route_accept_cost_due") {
		t.Fatal("route_accept_cost_due never fired on the idle branch")
	}
	if got := idle.silo.Professions[0].PanicValue; !charterRevisitAlmostEqual(got, 0.12) {
		t.Fatalf("idle branch panic = %v, want 0.12", got)
	}

	early := newRun()
	advanceSweepMonths(early, 4)
	if contentTriggered(early, "special:route_accept_cost_due") {
		t.Fatal("route_accept_cost_due fired before the 5-month delay")
	}
}

// 拒绝全链路（+8 月）：第二扇窗记暴露代价、解锁更贵的再接受，
// 再接受经新完成键重开勘察心跳（宿主职务动作照常承载）。
func TestRouteSecondWindowDeclinePath(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "route-window-decline.db"))
	req := threeSiloRequest(4601)
	req.Profession = "Mechanical"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	completeGateEvent(svc, "player_actions:route_survey_offer")
	executePlayerEventAction(t, svc, "ROUTE_SURVEY_DECLINE")
	if meta := actionByID(svc.availablePlayerActions(), "ROUTE_SURVEY_REACCEPT"); meta != nil && meta.Enabled {
		t.Fatalf("reaccept visible before the second window: %+v", meta)
	}

	if _, err := svc.PassTime(8); err != nil {
		t.Fatalf("advance to the second window: %v", err)
	}
	if !contentTriggered(svc, "special:route_second_window") {
		t.Fatal("route_second_window did not fire 8 months after the decline")
	}
	if !charterRevisitAlmostEqual(svc.silo.EscapePrep.PrepExposure, 0.03) {
		t.Fatalf("delay exposure = %v, want 0.03", svc.silo.EscapePrep.PrepExposure)
	}
	meta := actionByID(svc.availablePlayerActions(), "ROUTE_SURVEY_REACCEPT")
	if meta == nil || !meta.Enabled {
		t.Fatalf("reaccept must unlock after the second window: %+v", meta)
	}
	if meta.APCost != 2 {
		t.Fatalf("reaccept AP cost = %d, want 2 (the cost of waiting)", meta.APCost)
	}

	svc.agent.ActionPoints = 30
	svc.refreshActionProgress()
	executePlayerEventAction(t, svc, "ROUTE_SURVEY_REACCEPT")
	if !svc.statesMapFor(svc.silo.ID)["player_action:ROUTE_SURVEY_REACCEPT"].Triggered {
		t.Fatal("reaccept completion key missing")
	}

	// 勘察心跳订阅新接受键：打开职务动作阶梯后，宿主动作照常承载。
	svc.agent.ActionPoints = 30
	svc.refreshActionProgress()
	if _, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("opening gather: %v", err)
	}
	if _, err := svc.ExecuteAction(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	}); err != nil {
		t.Fatalf("opening share: %v", err)
	}
	svc.agent.ActionPoints = 30
	svc.refreshActionProgress()
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL"})
	if err != nil || outcome == nil || !outcome.Result.Executed {
		t.Fatalf("overhaul rejected: %+v %v", outcome, err)
	}
	if !contentTriggered(svc, "special:route_survey_step") {
		t.Fatal("route_survey_step must ride the reaccept key")
	}
	if svc.silo.EscapePrep.ReceptionStage != model.ReceptionSurveyed {
		t.Fatalf("reception stage = %q, want surveyed", svc.silo.EscapePrep.ReceptionStage)
	}
}

// 暂缓路径（+4 月）：时间也是成本——更短的回访延迟，同样的暴露与重开。
func TestRouteSecondWindowDeferPath(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "route-window-defer.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4602)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	completeGateEvent(svc, "player_actions:route_survey_offer")
	executePlayerEventAction(t, svc, "ROUTE_SURVEY_DEFER")

	if _, err := svc.PassTime(3); err != nil {
		t.Fatalf("advance 3 months: %v", err)
	}
	if contentTriggered(svc, "special:route_second_window") {
		t.Fatal("second window fired before the 4-month defer delay")
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("advance to the second window: %v", err)
	}
	if !contentTriggered(svc, "special:route_second_window") {
		t.Fatal("route_second_window did not fire 4 months after the defer")
	}
	if !charterRevisitAlmostEqual(svc.silo.EscapePrep.PrepExposure, 0.03) {
		t.Fatalf("delay exposure = %v, want 0.03", svc.silo.EscapePrep.PrepExposure)
	}
	if meta := actionByID(svc.availablePlayerActions(), "ROUTE_SURVEY_REACCEPT"); meta == nil || !meta.Enabled {
		t.Fatalf("reaccept must unlock after the defer window: %+v", meta)
	}
}
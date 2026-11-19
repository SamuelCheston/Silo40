package service

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

func charterRevisitAlmostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// 抉择延迟回访验收（《公议》批次）：接受/拒绝/暂缓三个完成键在数月后以
// 事件形态回来算账；暂缓不是死胡同——第二次敲门带新完成键重开选择。

func pickCharterRevisitDefs(t *testing.T, boot *GameService, keys ...string) []engine.ContentEventDefinition {
	t.Helper()
	var picked []engine.ContentEventDefinition
	for _, key := range keys {
		found := false
		for _, def := range boot.contentDefinitions {
			if def.Key == key {
				picked = append(picked, def)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("definition %q not loaded", key)
		}
	}
	return picked
}

// seedChoiceCompletion 写入完成键原形（大写动作 ID），时间戳落在指定月份，
// 供 monthsSinceEventGte 精确查表。
func seedChoiceCompletion(svc *GameService, actionID string, year, month int) {
	key := "player_action:" + actionID
	svc.statesMapFor(svc.silo.ID)[key] = model.ContentEventState{
		SiloID:                 svc.silo.ID,
		EventKey:               key,
		Triggered:              true,
		TriggerCount:           1,
		LastTriggeredTimestamp: gameTimestamp(year, month),
	}
}

func advanceSweepMonths(svc *GameService, months int) {
	for i := 0; i < months; i++ {
		svc.silo.CurrentMonth++
		if svc.silo.CurrentMonth > 12 {
			svc.silo.CurrentMonth = 1
			svc.silo.CurrentYear++
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()
	}
}

// 内容加载：三个回访事件与两个二次敲门动作必须随真实目录入库。
func TestCharterRevisitDefinitionsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	byKey := map[string]engine.ContentEventDefinition{}
	for _, def := range svc.contentDefinitions {
		byKey[def.Key] = def
	}
	for _, key := range []string{
		"special:charter_accept_momentum",
		"special:charter_decline_reroute",
		"special:charter_defer_knock",
	} {
		if _, ok := byKey[key]; !ok {
			t.Fatalf("%s not loaded", key)
		}
	}
	for key, wantID := range map[string]string{
		"player_actions:charter_second_accept":  "CHARTER_SECOND_ACCEPT",
		"player_actions:charter_second_decline": "CHARTER_SECOND_DECLINE",
	} {
		def, ok := byKey[key]
		if !ok || def.PlayerAction == nil || def.PlayerAction.ID != wantID {
			t.Fatalf("%s missing or wrong action id: %+v", key, def.PlayerAction)
		}
	}
}

// 接受回访（+5 月）：按起草心跳分流——沉默付恐慌与叛乱，供证喂信念层。
func TestCharterAcceptMomentumBranches(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot,
		"special:charter_accept_momentum", "special:charter_draft_step")

	newRun := func() *GameService {
		svc := chainTestService(defs...)
		svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
		svc.silo.Rebellion = 0.1
		svc.silo.Professions = []model.Profession{
			{ID: 1, SiloID: svc.silo.ID, Name: "Judicial", Population: 200, PanicValue: 0.1},
		}
		seedChoiceCompletion(svc, "CHARTER_CONTACT_ACCEPT", 122, 2)
		return svc
	}

	// 沉默分支：接受后从未供证，五个月后动能原地流失。
	silent := newRun()
	advanceSweepMonths(silent, 5)
	if !contentTriggered(silent, "special:charter_accept_momentum") {
		t.Fatal("charter_accept_momentum never fired on the silent branch")
	}
	if !charterRevisitAlmostEqual(silent.silo.Rebellion, 0.12) {
		t.Fatalf("silent branch rebellion = %v, want 0.12", silent.silo.Rebellion)
	}
	if got := silent.silo.Professions[0].PanicValue; !charterRevisitAlmostEqual(got, 0.12) {
		t.Fatalf("silent branch panic = %v, want 0.12", got)
	}

	// 供证分支：起草心跳走过一步，回访把供给记进司法信念层。
	fed := newRun()
	fed.statesMapFor(fed.silo.ID)["special:charter_draft_step"] = model.ContentEventState{
		SiloID: fed.silo.ID, EventKey: "special:charter_draft_step", Triggered: true, TriggerCount: 1,
	}
	advanceSweepMonths(fed, 5)
	if !contentTriggered(fed, "special:charter_accept_momentum") {
		t.Fatal("charter_accept_momentum never fired on the fed branch")
	}
	if !charterRevisitAlmostEqual(fed.silo.HistoryBurden, 0.02) {
		t.Fatalf("fed branch history_burden = %v, want 0.02", fed.silo.HistoryBurden)
	}
	if belief := engine.BeliefIn(&fed.silo.Professions[0].Beliefs, "Judicial_1"); belief == nil || belief.B <= 0 {
		t.Fatalf("fed branch did not feed the Judicial belief layer: %+v", fed.silo.Professions[0].Beliefs)
	}

	// 延迟纪律：四个月时不得提前回来。
	early := newRun()
	advanceSweepMonths(early, 4)
	if contentTriggered(early, "special:charter_accept_momentum") {
		t.Fatal("charter_accept_momentum fired before the 5-month delay")
	}
}

// 拒绝回访（+4 月）：世界明确引用这次拒绝，材料改道的合法性拉力落地。
// 二次拒绝走同一条回声。
func TestCharterDeclineRerouteEcho(t *testing.T) {
	boot := bootstrapRealContent(t)
	defs := pickCharterRevisitDefs(t, boot, "special:charter_decline_reroute")

	for _, actionID := range []string{"CHARTER_CONTACT_DECLINE", "CHARTER_SECOND_DECLINE"} {
		svc := chainTestService(defs...)
		svc.silo.CurrentYear, svc.silo.CurrentMonth = 122, 2
		svc.silo.Legitimacy = 0.5
		seedChoiceCompletion(svc, actionID, 122, 2)
		advanceSweepMonths(svc, 4)
		if !contentTriggered(svc, "special:charter_decline_reroute") {
			t.Fatalf("%s: reroute echo never fired", actionID)
		}
		if !charterRevisitAlmostEqual(svc.silo.Legitimacy, 0.52) {
			t.Fatalf("%s: legitimacy = %v, want 0.52", actionID, svc.silo.Legitimacy)
		}
	}
}

// 暂缓全链路：拒绝不是死胡同——六个月后第二次敲门解锁新完成键，
// 再接受重开起草线（心跳订阅新键），动能窗口重新挂表。
func TestCharterDeferSecondKnockAccept(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "charter-knock-accept.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4501)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	completeGateEvent(svc, "player_actions:charter_contact_offer")
	executePlayerEventAction(t, svc, "CHARTER_CONTACT_DEFER")
	if !svc.statesMapFor(svc.silo.ID)["player_action:CHARTER_CONTACT_DEFER"].Triggered {
		t.Fatal("defer completion key missing")
	}
	// 敲门前：二次选择不得提前可见。
	if meta := actionByID(svc.availablePlayerActions(), "CHARTER_SECOND_ACCEPT"); meta != nil && meta.Enabled {
		t.Fatalf("second accept visible before the knock: %+v", meta)
	}

	if _, err := svc.PassTime(6); err != nil {
		t.Fatalf("advance to the knock: %v", err)
	}
	if !contentTriggered(svc, "special:charter_defer_knock") {
		t.Fatal("charter_defer_knock did not fire 6 months after the defer")
	}
	for _, id := range []string{"CHARTER_SECOND_ACCEPT", "CHARTER_SECOND_DECLINE"} {
		if meta := actionByID(svc.availablePlayerActions(), id); meta == nil || !meta.Enabled {
			t.Fatalf("%s must unlock after the knock: %+v", id, meta)
		}
	}

	executePlayerEventAction(t, svc, "CHARTER_SECOND_ACCEPT")
	if !svc.statesMapFor(svc.silo.ID)["player_action:CHARTER_SECOND_ACCEPT"].Triggered {
		t.Fatal("second accept completion key missing")
	}
	// 互斥：再接受后第二次拒绝收起。
	if meta := actionByID(svc.availablePlayerActions(), "CHARTER_SECOND_DECLINE"); meta != nil && meta.Enabled {
		t.Fatalf("second decline must hide after the second accept: %+v", meta)
	}
	// 动能窗口重新挂表。
	expiryScheduled := false
	for _, task := range svc.delayTasks {
		if strings.Contains(task.EndEventID, "charter_window_expiry") && task.Status == "scheduled" {
			expiryScheduled = true
		}
	}
	if !expiryScheduled {
		t.Fatal("second accept must re-arm the charter window expiry")
	}
	// 起草心跳订阅新接受键：供一次证即走步（分享完成键为小写原形）。
	seedChoiceCompletion(svc, "action_share_info", svc.silo.CurrentYear, svc.silo.CurrentMonth)
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:charter_draft_step") {
		t.Fatal("charter_draft_step must ride the second accept key")
	}
}

// 暂缓→二次拒绝：终局拒绝不是空键——材料真的流向市长圈层，
// 喂证心跳经拒绝键照常走 Share Info。
func TestCharterDeferSecondKnockDeclineFeedsMayor(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "charter-knock-decline.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4502)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	completeGateEvent(svc, "player_actions:charter_contact_offer")
	executePlayerEventAction(t, svc, "CHARTER_CONTACT_DEFER")
	if _, err := svc.PassTime(6); err != nil {
		t.Fatalf("advance to the knock: %v", err)
	}
	if !contentTriggered(svc, "special:charter_defer_knock") {
		t.Fatal("charter_defer_knock did not fire")
	}
	executePlayerEventAction(t, svc, "CHARTER_SECOND_DECLINE")
	if !svc.statesMapFor(svc.silo.ID)["player_action:CHARTER_SECOND_DECLINE"].Triggered {
		t.Fatal("second decline completion key missing")
	}

	// 材料改道：拒绝键 + 供证 → 市长喂证心跳点火（分享完成键为小写原形）。
	seedChoiceCompletion(svc, "action_share_info", svc.silo.CurrentYear, svc.silo.CurrentMonth)
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:mayor_evidence_step") {
		t.Fatal("mayor_evidence_step must fire from the second decline reroute")
	}
}
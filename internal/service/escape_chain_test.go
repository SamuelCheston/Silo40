package service

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestEscapeLineContentLoads locks that the REBELLION batch (37-spec §2) loads
// from events/ with the four Witness nodes, the stance producers and the
// preparation actions all present.
func TestEscapeLineContentLoads(t *testing.T) {
	svc := bootstrapRealContent(t)

	defs := map[string]engine.ContentEventDefinition{}
	for _, def := range svc.contentDefinitions {
		defs[def.Key] = def
	}
	for _, key := range []string{
		// 分支二节点一～三（意愿/路线/开窗）与结算点（离开）
		"special:departure_willingness",
		"player_actions:route_survey_offer",
		"player_actions:route_survey_accept",
		"player_actions:route_survey_decline",
		"player_actions:route_survey_defer",
		"special:route_survey_step",
		"special:route_confirmed",
		"player_actions:advance_party_dispatch",
		"crisis:advance_party_report",
		"special:escape_window_open",
		"player_actions:escape_window_orderly",
		"player_actions:escape_window_riot",
		"player_actions:escape_departure_launch",
		// 立场生产者（顶层链）
		"player_actions:escape_negotiation_guarantee",
		// 准备账本生产者（Layer 1）
		"player_actions:escape_prep_suit_repair",
		"player_actions:escape_prep_suit_warehouse",
		"crisis:escape_prep_audit_seizure",
		"player_actions:escape_prep_transfer_supplies",
		"player_actions:escape_prep_transfer_materials",
		"player_actions:escape_plan_mechanical",
		"player_actions:escape_plan_mines",
		"player_actions:escape_plan_supply",
		"player_actions:escape_plan_medical",
		"player_actions:escape_plan_agricultural",
		"player_actions:escape_plan_order",
		"player_actions:escape_drill",
		"player_actions:escape_prep_care",
		"player_actions:escape_prep_production",
		"player_actions:escape_reception_stocking",
		"player_actions:escape_spillover_it_line",
		"player_actions:escape_spillover_cross_signal",
		"player_actions:escape_spillover_physical",
	} {
		if _, ok := defs[key]; !ok {
			t.Fatalf("expected content definition %q to load from events/", key)
		}
	}
	for actionID, key := range map[string]string{
		"ESCAPE_DEPARTURE_LAUNCH":       "player_actions:escape_departure_launch",
		"ESCAPE_WINDOW_ORDERLY":         "player_actions:escape_window_orderly",
		"ESCAPE_PREP_TRANSFER_SUPPLIES": "player_actions:escape_prep_transfer_supplies",
	} {
		def := defs[key]
		if def.PlayerAction == nil || def.PlayerAction.ID != actionID {
			t.Fatalf("%s must expose a player_action spec with id %s", key, actionID)
		}
	}
}

// TestEscapeWillingnessAndWindowFire drives the first half of the chain on the
// real bus wiring: willingness lights under panic, the window opens only when
// rebellion is organized AND a route fact exists (无路径的叛乱锁死开窗).
func TestEscapeWillingnessAndWindowFire(t *testing.T) {
	boot := bootstrapRealContent(t)
	svc := chainTestService(
		pickContentDef(t, boot, "special:departure_willingness"),
		pickContentDef(t, boot, "special:escape_window_open"),
	)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 123, 1
	for i := 1; i <= 3; i++ {
		svc.silo.Professions = append(svc.silo.Professions, model.Profession{
			ID: uint(i), SiloID: svc.silo.ID, Name: string(rune('A' + i)),
			PanicValue: 0.6,
		})
	}

	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:departure_willingness") {
		t.Fatal("departure_willingness did not fire under panicked departments")
	}

	// 高叛乱但无路线事实：开窗不得成立（Witness：无路径的叛乱锁死开窗）。
	svc.silo.Rebellion = 0.8
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if contentTriggered(svc, "special:escape_window_open") {
		t.Fatal("escape_window_open must stay locked without a confirmed route")
	}

	// 路线事实 + 组织（地下协调网络）齐备：开窗成立，并触发首次立场重估。
	states := svc.statesMapFor(svc.silo.ID)
	states["special:route_confirmed"] = model.ContentEventState{SiloID: svc.silo.ID, EventKey: "special:route_confirmed", Triggered: true}
	states["special:underground_coordination"] = model.ContentEventState{SiloID: svc.silo.ID, EventKey: "special:underground_coordination", Triggered: true}
	engine.SetRNG(deterministicTestRNG(4101))
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:escape_window_open") {
		t.Fatal("escape_window_open did not fire with rebellion + route + organization")
	}
	if !svc.silo.TopStance.Initialized || svc.silo.TopStance.Reevaluations != 1 {
		t.Fatalf("window opening must produce the first stance re-evaluation: %+v", svc.silo.TopStance)
	}
}

// TestEscapePreparationActionsCosts locks Layer 1 discipline on a real gated
// service: every ledger record has a producer, transfers deduct immediately,
// and insufficient balance fails explicitly (no partial transfer).
func TestEscapePreparationActionsCosts(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "escape-prep.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4101)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	// 意愿事实解锁准备动作；关系满足门控。
	completeGateEvent(svc, "special:departure_willingness")
	for _, name := range []string{"Mechanical", "Supply", "Agricultural"} {
		for i := range svc.silo.Professions {
			if svc.silo.Professions[i].Name == name {
				svc.agent.Connections = append(svc.agent.Connections, model.Connection{
					ProfessionID: svc.silo.Professions[i].ID, Value: 0.5,
				})
			}
		}
	}
	svc.agent.ActionPoints = 200
	svc.refreshActionProgress()

	exec := func(actionID string) bool {
		t.Helper()
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		if err != nil {
			t.Fatalf("execute %s: %v", actionID, err)
		}
		return outcome.Result.Executed
	}

	// 修服：+1 可用防护服，累积筹备暴露。
	if !exec("ESCAPE_PREP_SUIT_REPAIR") {
		t.Fatal("suit repair must execute")
	}
	if svc.silo.EscapePrep.SuitsFunctional != 1 {
		t.Fatalf("suits = %d, want 1", svc.silo.EscapePrep.SuitsFunctional)
	}
	if svc.silo.EscapePrep.PrepExposure <= 0 {
		t.Fatal("suit repair must accumulate prep exposure")
	}

	// 划拨即扣减：堡内补给立即减少，封存账本增加。
	suppliesBefore := resourceAmount(svc.silo, "Supplies")
	if !exec("ESCAPE_PREP_TRANSFER_SUPPLIES") {
		t.Fatal("supplies transfer must execute")
	}
	if got := resourceAmount(svc.silo, "Supplies"); got != suppliesBefore-200 {
		t.Fatalf("supplies after transfer = %v, want %v (划拨即扣减)", got, suppliesBefore-200)
	}
	if svc.silo.EscapePrep.SuppliesPacked != 200 {
		t.Fatalf("packed supplies = %v, want 200", svc.silo.EscapePrep.SuppliesPacked)
	}

	// 余额不足 = 显式失败：不部分划拨。
	for i := range svc.silo.Resources {
		if svc.silo.Resources[i].Type == "Supplies" {
			svc.silo.Resources[i].Amount = 50
		}
	}
	packedBefore := svc.silo.EscapePrep.SuppliesPacked
	exec("ESCAPE_PREP_TRANSFER_SUPPLIES")
	if resourceAmount(svc.silo, "Supplies") != 50 || svc.silo.EscapePrep.SuppliesPacked != packedBefore {
		t.Fatal("insufficient balance must fail explicitly without partial transfer")
	}

	// 预案功能位：完备度 = 覆盖率。
	if !exec("ESCAPE_PLAN_MECHANICAL") {
		t.Fatal("plan mechanical must execute")
	}
	if svc.silo.EscapePrep.PlanCompleteness != 1.0/model.PlanRoleCount {
		t.Fatalf("plan completeness = %v, want 1/6", svc.silo.EscapePrep.PlanCompleteness)
	}

	// 生产起点：材料门槛未达时不执行；达标后写入。
	if exec("ESCAPE_PREP_PRODUCTION") {
		t.Fatal("production start must stay gated below the materials threshold")
	}
	svc.silo.EscapePrep.MaterialsPacked = 60
	if !exec("ESCAPE_PREP_PRODUCTION") {
		t.Fatal("production start must execute above the materials threshold")
	}
	if !svc.silo.EscapePrep.ProductionStart {
		t.Fatal("production start flag missing")
	}
}

// TestEscapeDepartureSettlement locks the departure settlement point
// (16 §3.4 minimal form): stances shape the inputs, never a gate; the
// settlement's sole output is the escaped headcount; arbitration facts come
// from EscapedGroup alone.
func TestEscapeDepartureSettlement(t *testing.T) {
	newStation := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("REBELLION")
		station.TotalPopulation = 10000
		station.Rebellion = 0.8
		station.EscapePrep = model.EscapePreparation{
			Initialized:      true,
			SuitsFunctional:  200,
			SuppliesPacked:   500,
			MaterialsPacked:  80,
			PlanCompleteness: 4.0 / 6.0,
			PlanRoles:        []string{"agricultural", "mechanical", "medical", "supply"},
			DrillLevel:       0.6,
			CareReady:        0.5,
			ProductionStart:  true,
			ReceptionStage:   model.ReceptionConfirmed,
		}
		return svc, station
	}

	// 不阻挠双方 + 齐备账本：成规模群体成立并存续。
	svc, station := newStation()
	station.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceNonBlock, IT: model.TopStanceNonBlock}
	settle := svc.settleEscapeDeparture(station, nil)
	if settle == nil || settle.Departed <= 0 || settle.Survivors <= 0 {
		t.Fatalf("nonblock settlement must produce a group: %+v", settle)
	}
	if !station.EscapedGroup.SurvivingEscapeGroup() {
		t.Fatalf("prepared nonblock departure must yield a surviving group: %+v", station.EscapedGroup)
	}
	if station.EscapedGroup.Departed != settle.Departed || station.EscapedGroup.Population != settle.Survivors {
		t.Fatalf("group audit fields mismatch: %+v vs %+v", station.EscapedGroup, settle)
	}
	if station.TotalPopulation != 10000-settle.Departed {
		t.Fatalf("population must move out of the silo: %d", station.TotalPopulation)
	}

	// IT 阻挠：通道封闭 + 防护服库封 → 容量自然为 0，无群体，无显式否决。
	svc2, station2 := newStation()
	station2.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceNonBlock, IT: model.TopStanceBlock}
	settle2 := svc2.settleEscapeDeparture(station2, nil)
	if settle2.Departed != 0 || station2.EscapedGroup.SurvivingEscapeGroup() {
		t.Fatalf("IT block must settle to headcount 0 naturally: %+v %+v", settle2, station2.EscapedGroup)
	}

	// 市长阻挠：动员被压制 + 秩序侧追缉，幸存人数压向近 0。
	svc3, station3 := newStation()
	station3.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceBlock, IT: model.TopStanceNonBlock}
	settle3 := svc3.settleEscapeDeparture(station3, nil)
	if settle3.Survivors >= settle.Survivors/3 {
		t.Fatalf("mayor block must crush the outcome via mobilization + pursuit: survivors %d vs nonblock %d",
			settle3.Survivors, settle.Survivors)
	}

	// 配合双方：带走更多（容量与防护服改善）。
	svc4, station4 := newStation()
	station4.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceCooperate, IT: model.TopStanceCooperate}
	settle4 := svc4.settleEscapeDeparture(station4, nil)
	if settle4.Departed <= settle.Departed {
		t.Fatalf("cooperation must take more people: %d vs %d", settle4.Departed, settle.Departed)
	}

	// 补给断供：落地即散（M1 失败 → 群体不存续）。
	svc5, station5 := newStation()
	station5.EscapePrep.SuppliesPacked = 0
	station5.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceNonBlock, IT: model.TopStanceNonBlock}
	settle5 := svc5.settleEscapeDeparture(station5, nil)
	if settle5.Departed == 0 {
		t.Fatalf("missing supplies must not stop departure itself: %+v", settle5)
	}
	if station5.EscapedGroup.Surviving {
		t.Fatalf("zero supplies must scatter the group on arrival: %+v", station5.EscapedGroup)
	}

	// 路线未确认强行出发：存活率地板造成重损耗。
	svc6, station6 := newStation()
	station6.EscapePrep.ReceptionStage = model.ReceptionSurveyed
	station6.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceNonBlock, IT: model.TopStanceNonBlock}
	settle6 := svc6.settleEscapeDeparture(station6, nil)
	if settle6.Departed > 0 && settle6.Survivors >= settle6.Departed/2 {
		t.Fatalf("unconfirmed route must hit the survival floor: %+v", settle6)
	}
}

// TestEscapeDepartureLaunchAction drives the settlement through the real
// action path: the departure order re-evaluates stances one last time
// (最后一刻妥协也算) and then settles. Same seed ⇒ same outcome.
func TestEscapeDepartureLaunchAction(t *testing.T) {
	run := func(seed int64, name string) model.EscapeGroup {
		svc := newGatedService(t, filepath.Join(t.TempDir(), name))
		req := threeSiloRequest(seed)
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create game: %v", err)
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("settle opening month: %v", err)
		}
		svc.queueSystemContentEvents("TEST")
		svc.drainQueuedEvents()

		completeGateEvent(svc, "special:escape_window_open")
		completeGateEvent(svc, "player_actions:escape_window_orderly")
		svc.silo.Rebellion = 0.8
		svc.silo.EscapePrep = model.EscapePreparation{
			Initialized: true, SuitsFunctional: 150, SuppliesPacked: 400,
			PlanCompleteness: 0.5, DrillLevel: 0.5, CareReady: 0.4,
			ProductionStart: true, ReceptionStage: model.ReceptionConfirmed,
		}
		svc.agent.ActionPoints = 50
		svc.refreshActionProgress()

		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "ESCAPE_DEPARTURE_LAUNCH"})
		if err != nil || !outcome.Result.Executed {
			t.Fatalf("departure launch rejected: %+v %v", outcome.Result, err)
		}
		if !contentTriggered(svc, "player_actions:escape_departure_launch") {
			t.Fatal("departure completion key missing")
		}
		if svc.silo.TopStance.Reevaluations == 0 {
			t.Fatal("departure must re-evaluate stances one last time")
		}
		return svc.silo.EscapedGroup
	}

	first := run(4201, "escape-launch-a.db")
	second := run(4201, "escape-launch-b.db")
	if first != second {
		t.Fatalf("same seed must reproduce the departure outcome: %+v vs %+v", first, second)
	}
	// 结算只看逃出人数：立场由带权随机产生，阻挠时人数自然为 0 (无群体)，
	// 不阻挠时成立群体——两种形态都是结算的合法输出。
	if first.Departed > 0 && first.Population > first.Departed {
		t.Fatalf("survivors cannot exceed departed: %+v", first)
	}
	if first.Departed == 0 && first.Surviving {
		t.Fatalf("zero headcount cannot yield a surviving group: %+v", first)
	}
	if first.Departed > 0 && !first.Initialized {
		t.Fatalf("a departure with people must initialize the group: %+v", first)
	}
	t.Logf("departure settled under seed 4201: %+v", first)
}

// TestEscapeLedgerNotReadByArbitration locks the Layer 1 discipline: a rich
// preparation ledger alone never qualifies REBELLION — arbitration reads only
// the escape group. Hoarding preparation is not a victory progress bar.
func TestEscapeLedgerNotReadByArbitration(t *testing.T) {
	svc, station, _ := weightedEndingFixture("REBELLION")
	station.TotalPopulation = 8000
	station.EscapePrep = model.EscapePreparation{
		Initialized: true, SuitsFunctional: 500, SuppliesPacked: 2000,
		MaterialsPacked: 300, PlanCompleteness: 1, DrillLevel: 1,
		CareReady: 1, ProductionStart: true, ReceptionStage: model.ReceptionStocked,
	}
	station.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceCooperate, IT: model.TopStanceCooperate}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil {
		t.Fatal("no decision")
	}
	if _, excluded := decision.Excluded[endingRebellion]; !excluded {
		t.Fatalf("ledger alone must not qualify REBELLION: pool=%v", decision.Pool)
	}

	// 出发结算产出存续群体之后，资格才成立——仲裁只读群体事实。
	station.EscapedGroup = model.EscapeGroup{
		Initialized: true, Population: 120, Departed: 150, Surviving: true,
		DepartedYear: 126, DepartedMonth: 3,
	}
	decision2 := svc.resolveWeightedEnding(station, svc.agent)
	inPool := false
	for _, outcome := range decision2.Pool {
		if outcome == endingRebellion {
			inPool = true
		}
	}
	if !inPool {
		t.Fatalf("a surviving escape group must qualify REBELLION: %+v", decision2.Excluded)
	}
}

// TestRebellionEndingNarrative locks the REBELLION settlement presentation:
// the ending screen tells the second-roll-call story with the departure
// facts (departed/survived/fragile/what remains), and the pyrrhic branch
// keeps the same settlement line when the origin station falls afterwards.
func TestRebellionEndingNarrative(t *testing.T) {
	newGroupStation := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("REBELLION")
		station.TotalPopulation = 5000
		station.EscapedGroup = model.EscapeGroup{
			Initialized: true, Population: 120, Departed: 150, Surviving: true,
			DepartedYear: 126, DepartedMonth: 3,
		}
		svc.agent.PoliticalPrestige = 100
		return svc, station
	}

	// 正常成立：资格读群体存续，叙事读同一组事实。
	svc, station := newGroupStation()
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingRebellion {
		t.Fatalf("expected REBELLION selection, got %+v", decision)
	}
	for _, want := range []string{
		"《越岭》成立",
		"第二次公开清点",
		"表上没有你的名字",
		"本局共有 150 人出发，120 人幸存落地，路上损耗 30 人",
		"他们于 126 年 3 月离开",
		"驻地仍有居民 5000 人",
	} {
		if !strings.Contains(decision.Reason, want) {
			t.Fatalf("narrative missing %q: %q", want, decision.Reason)
		}
	}

	// 脆弱群体：结算行标注脆弱。
	svc2, station2 := newGroupStation()
	station2.EscapedGroup.Fragile = true
	decision2 := svc2.resolveWeightedEnding(station2, svc2.agent)
	if decision2.Selected != endingRebellion || !strings.Contains(decision2.Reason, "脆弱的") {
		t.Fatalf("fragile group must carry the fragile settlement line: %+v", decision2)
	}

	// 惨胜尾声：原驻地毁灭，存续群体不被追溯判死，结算行仍在。
	svc3, station3 := newGroupStation()
	svc3.contentStates = map[uint]map[string]model.ContentEventState{
		station3.ID: {terminationFactKey: {SiloID: station3.ID, EventKey: terminationFactKey, Triggered: true}},
	}
	decision3 := svc3.resolveWeightedEnding(station3, svc3.agent)
	if decision3 == nil || decision3.Selected != endingRebellion {
		t.Fatalf("death fact with a surviving group must settle as pyrrhic REBELLION, got %+v", decision3)
	}
	for _, want := range []string{"惨胜尾声", "本局共有 150 人出发"} {
		if !strings.Contains(decision3.Reason, want) {
			t.Fatalf("pyrrhic narrative missing %q: %q", want, decision3.Reason)
		}
	}
}

// deterministicTestRNG 为引擎单一 RNG 流注入固定种子。
func deterministicTestRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

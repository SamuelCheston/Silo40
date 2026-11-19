package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestOrgSuspicionDrivenByOrganizationNotHoarding 实跑验证怀疑度驱动边界
// (20 修订/ADR-0014)：囤积 (修服/划拨) 不抬高组织怀疑度；组织行为
// (演练/预案功能位) 抬高。准备本身不惹眼，惹眼的是组织行为。
func TestOrgSuspicionDrivenByOrganizationNotHoarding(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "org-suspicion.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4201)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	completeGateEvent(svc, "special:departure_willingness")
	for _, name := range []string{"Mechanical", "Supply"} {
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

	// 囤积：修服与划拨执行成功，但组织怀疑度保持为 0。
	if !exec("ESCAPE_PREP_SUIT_REPAIR") {
		t.Fatal("suit repair must execute")
	}
	if !exec("ESCAPE_PREP_TRANSFER_SUPPLIES") {
		t.Fatal("supplies transfer must execute")
	}
	if svc.silo.EscapePrep.SuitsFunctional == 0 || svc.silo.EscapePrep.SuppliesPacked == 0 {
		t.Fatalf("hoarding must land in the ledger: %+v", svc.silo.EscapePrep)
	}
	if svc.silo.EscapeResponse.Initialized || svc.silo.EscapeResponse.OrgSuspicion != 0 {
		t.Fatalf("hoarding must not raise org suspicion: %+v", svc.silo.EscapeResponse)
	}

	// 组织行为：演练抬高怀疑度 (动员范围)。
	if !exec("ESCAPE_DRILL") {
		t.Fatal("drill must execute")
	}
	afterDrill := svc.silo.EscapeResponse.OrgSuspicion
	if !svc.silo.EscapeResponse.Initialized || afterDrill <= 0 {
		t.Fatalf("drill must raise org suspicion: %+v", svc.silo.EscapeResponse)
	}

	// 组织行为：预案功能位继续抬高 (把一个部门拉进计划)。
	if !exec("ESCAPE_PLAN_MECHANICAL") {
		t.Fatal("plan mechanical must execute")
	}
	if svc.silo.EscapeResponse.OrgSuspicion <= afterDrill {
		t.Fatalf("plan seat must raise org suspicion further: %+v", svc.silo.EscapeResponse)
	}
}

// TestOrgSuspicionDecaysWithQuietMonths 组织行为停了，风声会淡：
// 连续安静月份后怀疑度按月衰减。
func TestOrgSuspicionDecaysWithQuietMonths(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "org-suspicion-decay.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4202)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	completeGateEvent(svc, "special:departure_willingness")
	svc.agent.ActionPoints = 200
	svc.refreshActionProgress()
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "ESCAPE_DRILL"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("drill must execute: %v", err)
	}
	peak := svc.silo.EscapeResponse.OrgSuspicion
	if peak <= 0 {
		t.Fatalf("drill must raise org suspicion: %+v", svc.silo.EscapeResponse)
	}

	// 安静月份：怀疑度按月衰减 (0.01/月)，且不会衰减到负。
	for i := 0; i < 3; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", i+1, err)
		}
	}
	if got := svc.silo.EscapeResponse.OrgSuspicion; got >= peak || got < 0 {
		t.Fatalf("suspicion after 3 quiet months = %v, want in (0, %v)", got, peak)
	}
}

// TestEscapeResponseDecisionFiresAndLogs 怀疑度到门槛后 IT 必须做选择：
// 决策日志带归一化权重与因子摘要，叙事走月度故事通道。
func TestEscapeResponseDecisionFiresAndLogs(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "escape-response-decide.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4203)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	svc.queueSystemContentEvents("TEST")
	svc.drainQueuedEvents()

	completeGateEvent(svc, "special:departure_willingness")
	svc.agent.ActionPoints = 200
	svc.refreshActionProgress()

	// 真实演练把怀疑度推过门槛 (0.34 白盒垫底 + 演练 +0.06)。
	svc.silo.EscapeResponse.OrgSuspicion = 0.34
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "ESCAPE_DRILL"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("drill must execute: %v", err)
	}

	result, err := svc.PassTime(1)
	if err != nil {
		t.Fatalf("pass time: %v", err)
	}

	state := svc.silo.EscapeResponse
	if len(state.Decisions) != 1 {
		t.Fatalf("expected exactly one decision log entry, got %+v", state.Decisions)
	}
	entry := state.Decisions[0]
	switch entry.Decision {
	case model.EscapeResponseSuppress, model.EscapeResponseConceal, model.EscapeResponseIgnore:
	default:
		t.Fatalf("unknown decision %q", entry.Decision)
	}
	sum := entry.Weights[model.EscapeResponseSuppress] + entry.Weights[model.EscapeResponseConceal] + entry.Weights[model.EscapeResponseIgnore]
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("decision weights must normalize: %+v", entry.Weights)
	}
	if entry.Factors == "" {
		t.Fatal("decision log must carry the factor summary (attribution)")
	}

	// 叙事走月度故事通道：当月反馈里有一条响应层故事。
	found := false
	for _, story := range result.Stories {
		if len(story.ID) > 16 && story.ID[:16] == "escape_response:" {
			found = true
			if story.Title == "" || story.Description == "" {
				t.Fatalf("response story must carry narrative: %+v", story)
			}
		}
	}
	if !found {
		t.Fatalf("decision must surface through the monthly story channel: %+v", result.Stories)
	}
}

// TestEscapeResponseCoverCollapseViaMonthlyTick 蒙蔽付不起月费时崩塌：
// 登记蒙蔽崩塌事件 (ADR-0019，不再直接累加风险度)，决策日志记录结果。
func TestEscapeResponseCoverCollapseViaMonthlyTick(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "escape-response-cover.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4204)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}

	state := &svc.silo.EscapeResponse
	state.Initialized = true
	// 打开黑箱功能语义 (ADR-0019 事件登记)，同时用专线干扰挡住 1 号月度指派
	// 对风险基线的改写，隔离响应层副作用。
	svc.silo.BlackBox.Initialized = true
	svc.world.Flags[model.WorldFlagNetworkJammed] = true
	state.OrgSuspicion = 0.5
	state.CoverActive = true
	state.Decisions = []model.EscapeResponseLogEntry{{Decision: model.EscapeResponseConceal}}
	// 月度 IT 回合会把 0 的 R_op 回收到 8；W=20 时蒙蔽月成本 ≥12，付不起
	// (W 单月漂移不改变量级)。
	svc.silo.ItAuthority.ROp = 0
	svc.silo.Authority.W = 20
	rsBefore := svc.silo.SafeguardRisk

	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if state.CoverActive {
		t.Fatal("unaffordable cover must collapse during the monthly tick")
	}
	if svc.silo.SafeguardRisk != rsBefore {
		t.Fatalf("collapse must not write safeguard risk directly: %v -> %v", rsBefore, svc.silo.SafeguardRisk)
	}
	if svc.silo.SafeguardIncidents[engine.SafeguardIncidentCoverCollapse] == 0 {
		t.Fatalf("collapse must register a cover_collapse incident: %+v", svc.silo.SafeguardIncidents)
	}
	if state.Decisions[len(state.Decisions)-1].Outcome == "" {
		t.Fatal("collapse must be recorded in the decision log outcome")
	}
}

// TestEscapeResponseWindowOpenEndsLayer 开窗后筹备期结束：响应层停摆，
// 进行中的蒙蔽收尾，不再产生新决策。
func TestEscapeResponseWindowOpenEndsLayer(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "escape-response-window.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4205)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}

	state := &svc.silo.EscapeResponse
	state.Initialized = true
	state.OrgSuspicion = 0.8 // 远超门槛：若层未停摆必然点火
	state.CoverActive = true
	states := svc.statesMapFor(svc.silo.ID)
	states["special:escape_window_open"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:escape_window_open", Triggered: true,
	}

	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if state.CoverActive {
		t.Fatal("window opening must end the active cover")
	}
	if len(state.Decisions) != 0 {
		t.Fatalf("no decision may fire after the window opens: %+v", state.Decisions)
	}
}

// TestEscapeResponseSafeguardBudget 预算不变式 (ADR-0019 新语义)：组织活动
// 持续不断的最坏情况下，响应层对 safeguard 风险的全程净贡献为 0——不作为与
// 蒙蔽崩塌改登记事件、经观测管线进入 1 号信念，本通道永不直接触碰风险度，
// 更到不了硬护栏 1.0，后果只能是"日后更贵、更窄"，不困死玩家。
func TestEscapeResponseSafeguardBudget(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "escape-response-budget.db"))
	if _, err := svc.CreateGame(threeSiloRequest(4206)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("settle opening month: %v", err)
	}
	svc.silo.EscapeResponse.Initialized = true
	// 功能开启语义 (ADR-0019)：登记事件而非写风险度；专线干扰隔离 1 号月度
	// 指派对风险基线的改写，便于断言响应层净贡献。
	svc.silo.BlackBox.Initialized = true
	svc.world.Flags[model.WorldFlagNetworkJammed] = true
	rsStart := svc.silo.SafeguardRisk

	// 24 月筹备窗 (16 §6.3 渗透下界口径)：每月把怀疑度垫回门槛上方，
	// 模拟"组织活动不停"的最坏情形。
	for m := 0; m < 24; m++ {
		svc.silo.EscapeResponse.OrgSuspicion = 0.5
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", m+1, err)
		}
		if svc.silo.SafeguardRisk >= 1.0 {
			t.Fatalf("response channel alone must never reach the safeguard hard guardrail: R_s=%v",
				svc.silo.SafeguardRisk)
		}
	}
	delta := svc.silo.SafeguardRisk - rsStart
	if delta != 0 {
		t.Fatalf("response channel must not write safeguard risk directly: delta=%v", delta)
	}
	if len(svc.silo.EscapeResponse.Decisions) == 0 {
		t.Fatal("budget scenario must actually fire decisions (no empty green)")
	}
	incidents := svc.silo.SafeguardIncidents[engine.SafeguardIncidentIgnore] +
		svc.silo.SafeguardIncidents[engine.SafeguardIncidentCoverCollapse]
	if incidents == 0 {
		t.Fatalf("fired decisions with consequences must register incidents: %+v",
			svc.silo.SafeguardIncidents)
	}
}

// TestEscapePreparationGradientNotReversed 反向梯度消除的实跑验收：
// 同种子下，准备更充分的一局 (full：路线确认+囤备、预案、演练、照护、
// 生产、划拨、修服) 在出发人数与幸存人数上都不差于准备稀疏的一局
// (sparse：无前遣、半数预案、无演练/照护/生产、单次划拨)。
// 审计没收时代该对比可能反转 (准备越重被没收越多)；囤积不再被没收、
// 怀疑度只由组织行为驱动后，梯度必须单调。
func TestEscapePreparationGradientNotReversed(t *testing.T) {
	full := runEscapeLine(t, "escapefull", 2401, "full", "")
	sparse := runEscapeLine(t, "escapesparse", 2401, "sparse", "")

	t.Logf("full:   ending=%s launch=%s departed=%d survivors=%d months_out=%d demise=%s",
		full.Ending, full.StanceAtLaunch, full.Departed, full.Survivors, full.MonthsOut, full.Demise)
	t.Logf("sparse: ending=%s launch=%s departed=%d survivors=%d months_out=%d demise=%s",
		sparse.Ending, sparse.StanceAtLaunch, sparse.Departed, sparse.Survivors, sparse.MonthsOut, sparse.Demise)

	if full.Departed <= 0 {
		t.Fatalf("full prep run must depart: %+v", full)
	}
	if full.Departed < sparse.Departed {
		t.Fatalf("gradient reversed on departed headcount: full=%d sparse=%d (full notes=%v)",
			full.Departed, sparse.Departed, full.Notes)
	}
	if full.Survivors < sparse.Survivors {
		t.Fatalf("gradient reversed on road survivors: full=%d sparse=%d (full notes=%v)",
			full.Survivors, sparse.Survivors, full.Notes)
	}
}

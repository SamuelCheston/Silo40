package service

import (
	"path/filepath"
	"testing"

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

package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
)

// TestSilentProbeGatedByShadowyTrait locks 19 §4.2：无声探查仅隐秘行事可见
// 可用，检测强制 UNSAMPLED (无怀疑增量)，AP 为普通探查 2 倍，且与普通探查
// 同样计入首观察键。
func TestSilentProbeGatedByShadowyTrait(t *testing.T) {
	// 无特质：动作不可见、执行被拒。
	plain := newGatedService(t, filepath.Join(t.TempDir(), "silent-plain.db"))
	if _, err := plain.CreateGame(archetypeRequest(201, "")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if meta := actionByID(plain.availablePlayerActions(), "action_silent_probe"); meta != nil {
		t.Fatalf("silent probe must stay hidden without the trait: %+v", meta)
	}
	if _, err := plain.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"}); err == nil || !strings.Contains(err.Error(), "not been unlocked") {
		t.Fatalf("silent probe bypass without trait must be rejected: %v", err)
	}

	// 隐秘行事：解锁、2 倍 AP、零怀疑、计入首观察。
	svc := newGatedService(t, filepath.Join(t.TempDir(), "silent-shadowy.db"))
	if _, err := svc.CreateGame(archetypeRequest(201, "", "shadowy")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	meta := actionByID(svc.availablePlayerActions(), "action_silent_probe")
	if meta == nil || meta.Visibility != model.ActionAvailable || !meta.Enabled {
		t.Fatalf("silent probe must be available for shadowy agents: %+v", meta)
	}

	gatherCost := resolveProbeCost(t, svc, model.ActionGatherInfo, "Mayor")
	silentCost := resolveProbeCost(t, svc, model.ActionSilentProbe, "Mayor")
	if silentCost != 2*gatherCost {
		t.Fatalf("silent probe cost = %v, want 2× gather cost %v", silentCost, gatherCost)
	}

	apBefore := svc.agent.ActionPoints
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("silent probe execution failed: %v %+v", err, outcome)
	}
	if got := apBefore - svc.agent.ActionPoints; got != silentCost {
		t.Fatalf("silent probe charged %v AP, want %v", got, silentCost)
	}
	if svc.agent.SuspicionLevel != 0 {
		t.Fatalf("silent probe must be UNSAMPLED (zero suspicion), got %v", svc.agent.SuspicionLevel)
	}
	if state := svc.statesMapFor(svc.silo.ID)[observationFirstLocalKey]; !state.Triggered {
		t.Fatalf("silent probe must count as local observation (first_signal key)")
	}

	// 对照：同种子同职业的普通探查是 SAMPLED，必然产生怀疑增量。
	compare := newGatedService(t, filepath.Join(t.TempDir(), "silent-compare.db"))
	if _, err := compare.CreateGame(archetypeRequest(201, "")); err != nil {
		t.Fatalf("create compare game: %v", err)
	}
	outcome, err = compare.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("gather execution failed: %v %+v", err, outcome)
	}
	if compare.agent.SuspicionLevel <= 0 {
		t.Fatalf("ordinary gather by Judicial must be SAMPLED and raise suspicion, got %v", compare.agent.SuspicionLevel)
	}
}

// resolveProbeCost 走服务端权威成本口径 (含规范冲突上浮)。
func resolveProbeCost(t *testing.T, svc *GameService, actionType model.AgentActionType, targetDept string) float64 {
	t.Helper()
	action := model.AgentAction{Type: actionType, TargetDept: targetDept}
	if err := svc.resolveActionCost(&action); err != nil {
		t.Fatalf("resolve cost for %s: %v", actionType, err)
	}
	return action.Cost
}

// TestNativeTraitWaivesSameClassRelationGate locks 19 §4.3：地堡土著对同阶层
// 关系门视为已满足，跨阶层关系门与阶段门不受影响。
func TestNativeTraitWaivesSameClassRelationGate(t *testing.T) {
	min02 := 0.2

	native := newGatedService(t, filepath.Join(t.TempDir(), "native.db"))
	if _, err := native.CreateGame(model.CreateGameRequest{
		SiloName: "NativeStation", AgentName: "Player", Profession: "Mines",
		Seed: 301, TraitIds: []string{"native"},
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	// Supply 与 Mines 同属 COMMONER：关系门豁免，动作直接解锁。
	if !native.evaluateActionCondition(actionCondition{Relation: "relation:Supply", Min: &min02}) {
		t.Fatalf("native commoner must waive the Supply relation gate")
	}
	if !native.actionUnlocked("STATION_TRADE") {
		t.Fatalf("STATION_TRADE (Supply relation gate) must unlock for native Mines without building network")
	}
	// Judicial 属 ELITE：跨阶层不豁免，关系不足仍被挡。
	if native.evaluateActionCondition(actionCondition{Relation: "relation:Judicial", Min: &min02}) {
		t.Fatalf("cross-class relation gate must not be waived")
	}

	plain := newGatedService(t, filepath.Join(t.TempDir(), "native-plain.db"))
	if _, err := plain.CreateGame(model.CreateGameRequest{
		SiloName: "PlainStation", AgentName: "Player", Profession: "Mines", Seed: 301,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if plain.evaluateActionCondition(actionCondition{Relation: "relation:Supply", Min: &min02}) {
		t.Fatalf("without the native trait the Supply relation gate must hold (0.15 < 0.20)")
	}
	if plain.actionUnlocked("STATION_TRADE") {
		t.Fatalf("STATION_TRADE must stay locked for non-native Mines at game start")
	}
}

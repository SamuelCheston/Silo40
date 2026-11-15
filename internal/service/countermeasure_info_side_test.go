package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 20 分册 T2-3 分享目标收紧（等价实现）：戒备 ≥ 0.50 的部门交接成本翻倍，
// 但动作仍可执行——反制只抬成本，不永久锁死路径；戒备由月度冷却回落。
func TestShareCostDoublesWhenTargetDeptScrutinized(t *testing.T) {
	svc := newActionCostTestService(t, "Mayor")
	it := findProfessionByName(svc.silo, "IT")
	if it == nil {
		t.Fatal("IT profession missing from initial silo")
	}
	shareBase := model.ACTION_COSTS[model.ActionShareInfo]
	buildBase := model.ACTION_COSTS[model.ActionBuildConnection]

	shareCases := []struct {
		alertness float64
		want      float64
	}{
		{alertness: 0.20, want: shareBase},
		{alertness: 0.49, want: shareBase},
		{alertness: 0.50, want: 2 * shareBase}, // T2-3 目标收紧
		{alertness: 0.95, want: 2 * shareBase},
	}
	for _, tc := range shareCases {
		it.Alertness = tc.alertness
		action := model.AgentAction{Type: model.ActionShareInfo, TargetDept: "IT"}
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolveActionCost(share, alertness=%.2f): %v", tc.alertness, err)
		}
		if action.Cost != tc.want {
			t.Errorf("share cost at alertness %.2f = %.1f, want %.1f", tc.alertness, action.Cost, tc.want)
		}
	}

	buildCases := []struct {
		alertness float64
		want      float64
	}{
		{alertness: 0.20, want: buildBase},
		{alertness: 0.30, want: buildBase + 1}, // T1-2 建联摩擦
		{alertness: 0.95, want: buildBase + 1},
	}
	for _, tc := range buildCases {
		it.Alertness = tc.alertness
		action := model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "IT"}
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolveActionCost(build, alertness=%.2f): %v", tc.alertness, err)
		}
		if action.Cost != tc.want {
			t.Errorf("build cost at alertness %.2f = %.1f, want %.1f", tc.alertness, action.Cost, tc.want)
		}
	}
}

// 冷却是 T2-3/T1-2 可恢复的前提：戒备每月回落，不会永久停在高位。
func TestProfessionAlertnessCoolsMonthly(t *testing.T) {
	svc := newActionCostTestService(t, "Mayor")
	it := findProfessionByName(svc.silo, "IT")
	if it == nil {
		t.Fatal("IT profession missing from initial silo")
	}
	it.Alertness = 0.03
	engine.DecayProfessionAlertness(svc.silo)
	if got := it.Alertness; got < 0.014 || got > 0.016 {
		t.Errorf("alertness after one cooldown = %.3f, want ≈0.015", got)
	}
	engine.DecayProfessionAlertness(svc.silo)
	if it.Alertness != 0 {
		t.Errorf("alertness must clamp at 0, got %.3f", it.Alertness)
	}
}

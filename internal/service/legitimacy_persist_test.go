package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// TestLegitimacySurvivesRealMonthlyTick 在真实内容栈上锁定 P0 修复的两端：
//
//  1. events/mechanics/core_systems.js 的 metrics_update 不再把市长正统性
//     重设为派系加权凝聚力——有派系的月度结算后正统性必须原样留存；
//  2. 叛乱侵蚀已接到真实月度路径——有叛乱的月度结算后正统性必须下降
//     （修复前侵蚀只挂在不可达的 Go 兜底，真实流程从不发生）。
//
// 精确幅度由引擎层 legitimacy_dispatch_test 锁定；这里只锁真实路径行为。
func TestLegitimacySurvivesRealMonthlyTick(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "legitimacy-persist.db"))
	if _, err := svc.CreateGame(threeSiloRequest(2401)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	silo := svc.silo

	// 有派系、无叛乱：修复前此月过后正统性会被覆写为凝聚力 (0.9)，
	// 修复后必须原样留存。正统性 0.2 同时低于 mayor_public_stand 的
	// 0.5 门槛，排除担当时刻写入的干扰。
	silo.Factions = []model.Faction{{ID: 1, Name: "Persist Probe", Influence: 1, Cohesion: 0.9}}
	silo.Rebellion = 0
	silo.SetMayorLegitimacy(0.2)
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time with factions: %v", err)
	}
	if silo.MayorLegitimacy() != 0.2 {
		t.Fatalf("legitimacy must survive the real monthly metrics update: want 0.2, got %v (cohesion=%v)",
			silo.MayorLegitimacy(), silo.Cohesion)
	}

	// 有叛乱：真实月度路径必须发生侵蚀（修复前逐位不变）。
	silo.Factions = nil
	silo.Rebellion = 0.5
	silo.SetMayorLegitimacy(0.2)
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time with rebellion: %v", err)
	}
	if silo.MayorLegitimacy() >= 0.2 {
		t.Fatalf("active rebellion must erode legitimacy on the real monthly path: got %v", silo.MayorLegitimacy())
	}
}

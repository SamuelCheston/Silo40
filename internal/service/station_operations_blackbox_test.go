package service

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestStationIntelBandedWhenBlackBoxEnabled 00 卷裁决 5：功能开启时
// STATION_INTEL 精确读数改为分档读数；成本/暴露不变。
func TestStationIntelBandedWhenBlackBoxEnabled(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "intel-on.db"))
	req := threeSiloRequest(83)
	req.Silo1BlackBox = true
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	def, ok := stationOperationDefinition(StationOpIntel)
	if !ok {
		t.Fatal("missing STATION_INTEL definition")
	}
	exposureBefore := svc.world.StationOps.Exposure
	apBefore := svc.agent.ActionPoints
	result := svc.scheduleStationOperation(model.AgentAction{
		Type: model.ActionStationOperation, ActionID: StationOpIntel, Cost: float64(def.APCost),
	})
	if !result.Executed {
		t.Fatalf("schedule: %s", result.Message)
	}
	// 成本/暴露语义不变。
	if svc.world.StationOps.Exposure != exposureBefore+def.Exposure {
		t.Fatalf("暴露变化 %v != 预期 +%v", svc.world.StationOps.Exposure-exposureBefore, def.Exposure)
	}
	if svc.agent.ActionPoints != apBefore-float64(def.APCost) {
		t.Fatalf("AP 变化 %v != 预期 -%v", apBefore-svc.agent.ActionPoints, def.APCost)
	}
	svc.world.StationOps.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.StationOps.Operations[0].DueMonth = svc.world.CurrentMonth
	engine.SetRNG(rand.New(rand.NewSource(83)))
	var logs []string
	var stories []model.StoryEvent
	svc.resolveStationOperations(&logs, &stories)
	op := svc.world.StationOps.Operations[0]
	if op.Status != "RESOLVED" {
		t.Fatalf("operation not resolved: %+v", op)
	}
	report := op.Resolution
	if !strings.Contains(report, "档") {
		t.Fatalf("开启时报告应为分档读数：%s", report)
	}
	if strings.Contains(report, "0.") {
		t.Fatalf("开启时报告不得含精确小数值：%s", report)
	}
	if !strings.Contains(report, "Mechanical") {
		t.Fatalf("部门构成仍应精确保留：%s", report)
	}
}

// TestStationIntelPreciseWhenBlackBoxDisabled 00 卷裁决 5 关闭侧：功能关闭时
// 精确读数逐位不变。
func TestStationIntelPreciseWhenBlackBoxDisabled(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "intel-off.db"))
	if _, err := svc.CreateGame(threeSiloRequest(85)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	def, _ := stationOperationDefinition(StationOpIntel)
	result := svc.scheduleStationOperation(model.AgentAction{
		Type: model.ActionStationOperation, ActionID: StationOpIntel, Cost: float64(def.APCost),
	})
	if !result.Executed {
		t.Fatalf("schedule: %s", result.Message)
	}
	svc.world.StationOps.Operations[0].DueYear = svc.world.CurrentYear
	svc.world.StationOps.Operations[0].DueMonth = svc.world.CurrentMonth
	target := svc.stationSilo()
	var logs []string
	var stories []model.StoryEvent
	svc.resolveStationOperations(&logs, &stories)
	report := svc.world.StationOps.Operations[0].Resolution
	want := fmt.Sprintf("合法性 %.2f", target.MayorLegitimacy())
	if !strings.Contains(report, want) {
		t.Fatalf("关闭时报告应保持精确读数 %q：%s", want, report)
	}
}

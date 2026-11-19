package engine

import (
	"math"
	"testing"
)

// TestLegitimacyErosionMechanicPathsMutuallyExclusive 锁定叛乱侵蚀正统性的
// 唯一月度路径：JS 成功路与 Go 兜底路各恰好执行一次侵蚀，读当月更新后的
// 叛乱值。此前侵蚀只挂在 Go 兜底的 updateSiloMetrics 里，而 JS 的
// metrics_update 总是成功、兜底不可达——真实流程里侵蚀从未发生。任一路
// 漏挂或双跑（例如在兜底里恢复旧调用），精确幅度断言都会立刻失败。
func TestLegitimacyErosionMechanicPathsMutuallyExclusive(t *testing.T) {
	const start = 0.7
	want := start - rebellionLegitimacyErosion*0.5*(1.0/12)

	t.Run("JS success", func(t *testing.T) {
		e := NewGameEngine()
		const id = "legitimacy_dispatch_test"
		// JS 只重写叛乱（模拟当月 metrics_update 的结算结果），不碰正统性。
		e.MechanicEngine.byEventType[EVENT_METRICS_UPDATE] = MechanicDefinition{
			ID: id, EventType: EVENT_METRICS_UPDATE,
			ScriptSource: `defineMechanics([{id:"legitimacy_dispatch_test",event_type:"METRICS_UPDATE",apply:function(){return {mutations:[{type:"silo_metric_set",metric:"rebellion",value:0.5}]};}}]);`,
		}
		silo := CreateInitialSilo("ErosionJS", 122, nil)
		silo.SetMayorLegitimacy(start)
		e.runEventMechanic(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), NewEventContext())
		if silo.Rebellion != 0.5 {
			t.Fatalf("JS success-path sentinel missing (Go fallback may have run): rebellion=%v", silo.Rebellion)
		}
		if got := silo.MayorLegitimacy(); math.Abs(got-want) > 1e-12 {
			t.Fatalf("erosion must run exactly once on the post-update rebellion: legitimacy=%v want %v", got, want)
		}
	})

	t.Run("Go fallback", func(t *testing.T) {
		e := NewGameEngine()
		e.MechanicEngine = nil
		silo := CreateInitialSilo("ErosionGo", 122, nil)
		silo.SetMayorLegitimacy(start)
		silo.Rebellion = 0.5 // 无派系 -> totalInf=0，兜底不改叛乱
		e.runEventMechanic(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), NewEventContext())
		if silo.Rebellion != 0.5 {
			t.Fatalf("fallback must keep rebellion unchanged without factions: %v", silo.Rebellion)
		}
		if got := silo.MayorLegitimacy(); math.Abs(got-want) > 1e-12 {
			t.Fatalf("fallback path must erode exactly once: legitimacy=%v want %v", got, want)
		}
	})
}

// TestLegitimacyPoliticalDispatchGuards 锁定派发器的三个不变边界：零月路径、
// 非 METRICS_UPDATE 事件、零叛乱局都必须逐位不变（纯算术不耗 RNG）。
func TestLegitimacyPoliticalDispatchGuards(t *testing.T) {
	e := &GameEngine{}
	silo := CreateInitialSilo("PoliticalGuard", 122, nil)
	silo.SetMayorLegitimacy(0.7)

	e.runMonthlyPoliticalDispatch(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 0), silo)
	if silo.MayorLegitimacy() != 0.7 {
		t.Fatalf("deltaYears=0 must not settle: %v", silo.MayorLegitimacy())
	}

	e.runMonthlyPoliticalDispatch(EVENT_AGENT_UPDATE, metricsUpdateEvent(silo, 1.0/12), silo)
	if silo.MayorLegitimacy() != 0.7 {
		t.Fatalf("non-METRICS_UPDATE eventType must be a no-op: %v", silo.MayorLegitimacy())
	}

	silo.Rebellion = 0
	e.runMonthlyPoliticalDispatch(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), silo)
	if silo.MayorLegitimacy() != 0.7 {
		t.Fatalf("zero rebellion must not erode: %v", silo.MayorLegitimacy())
	}
}

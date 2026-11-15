package engine

import (
	"math"
	"reflect"
	"testing"

	"silo40/internal/model"
)

// metricsUpdateEvent 按 CreateMetricsUpdateEvent 的形态合成月度事件。
func metricsUpdateEvent(silo *model.Silo, deltaYears float64) *GameEvent {
	return CreateEvent("metrics_update#t", EVENT_METRICS_UPDATE, map[string]interface{}{
		"silo": silo, "deltaYears": deltaYears,
	})
}

// dispatchMonth 走真实月度路径的挂钩点结算一个月。
func dispatchMonth(e *GameEngine, silo *model.Silo) {
	e.runMonthlyBeliefDispatch(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), silo)
}

func TestForgettingDispatchExactlyOnceAndOrder(t *testing.T) {
	cfg := DefaultInformationConfig()
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	dispatchMonth(e, silo)
	belief := silo.Professions[0].Beliefs[0]
	wantB := 0.55 - cfg.ForgetRateMonthly
	if math.Abs(belief.B-wantB) >= 1e-12 {
		t.Fatalf("one dispatch must decay B exactly once: B = %v, want %v", belief.B, wantB)
	}
	if belief.PermeationCounter != 1 {
		t.Fatalf("same-tick order must be forget-then-count: counter = %d, want 1", belief.PermeationCounter)
	}
	if belief.Matured {
		t.Fatal("a single dispatch must not mature the fact")
	}
}

// TestForgettingDispatchOrderDiscriminates 用地板加半个淡忘速率构造判别值：
// 淡忘前在地板上，淡忘后在地板下。先淡忘再计数得到 0；顺序颠倒会得到 1。
func TestForgettingDispatchOrderDiscriminates(t *testing.T) {
	cfg := DefaultInformationConfig()
	e := &GameEngine{}
	startB := cfg.QualRumor + cfg.ForgetRateMonthly/2
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: startB})
	dispatchMonth(e, silo)
	belief := silo.Professions[0].Beliefs[0]
	wantB := startB - cfg.ForgetRateMonthly
	if math.Abs(belief.B-wantB) >= 1e-12 {
		t.Fatalf("B = %v, want %v", belief.B, wantB)
	}
	if belief.PermeationCounter != 0 {
		t.Fatalf("counter = %d, want 0 (count-before-forget would give 1)", belief.PermeationCounter)
	}
}

// TestForgettingDispatchMaturesAfterCount 判别成熟判定在当月计数之后：已有
// PermeationMonths-1 个真实计数的事实，必须在本月淡忘并计数后成熟，而非再等一月。
func TestForgettingDispatchMaturesAfterCount(t *testing.T) {
	cfg := DefaultInformationConfig()
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	for month := 1; month < cfg.PermeationMonths; month++ {
		ApplyPermeation(silo, cfg)
	}
	dispatchMonth(e, silo)
	belief := silo.Professions[0].Beliefs[0]
	if belief.PermeationCounter != cfg.PermeationMonths || !belief.Matured {
		t.Fatalf("maturity must follow this month's count: %+v", belief)
	}
}

func TestForgettingDispatchFloorBoundary(t *testing.T) {
	cfg := DefaultInformationConfig()
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: cfg.QualRumor})
	dispatchMonth(e, silo)
	belief := &silo.Professions[0].Beliefs[0]
	wantB := cfg.QualRumor - cfg.ForgetRateMonthly
	if math.Abs(belief.B-wantB) >= 1e-12 {
		t.Fatalf("B = %v, want %v", belief.B, wantB)
	}
	if belief.PermeationCounter != 0 {
		t.Fatalf("B exactly at the floor must not count after decay: counter = %d", belief.PermeationCounter)
	}

	// 真实再流通把 B 拉回地板以上后，下一月计数恢复。
	AbsorbSharedFact(cfg, &silo.Professions[0].Beliefs, "IT_1", "player", 0.1)
	dispatchMonth(e, silo)
	if belief.PermeationCounter != 1 {
		t.Fatalf("counter must resume once circulation holds B above the floor: %d", belief.PermeationCounter)
	}
}

func TestForgettingDispatchZeroMonthGuard(t *testing.T) {
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	e.runMonthlyBeliefDispatch(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 0), silo)
	belief := silo.Professions[0].Beliefs[0]
	if belief.B != 0.55 || belief.PermeationCounter != 0 {
		t.Fatalf("deltaYears=0 must not settle: %+v", belief)
	}
}

func TestForgettingDispatchEventTypeGuard(t *testing.T) {
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	e.runMonthlyBeliefDispatch(EVENT_AGENT_UPDATE, metricsUpdateEvent(silo, 1.0/12), silo)
	belief := silo.Professions[0].Beliefs[0]
	if belief.B != 0.55 || belief.PermeationCounter != 0 {
		t.Fatalf("non-METRICS_UPDATE eventType must be a no-op: %+v", belief)
	}
}

// TestForgettingDispatchMechanicPathsMutuallyExclusive 分别跑 JS 成功路与 Go
// 兜底路。JS 的 sentinel countdown=42 证明没有跌入兜底；两路都必须只淡忘、计数一次。
func TestForgettingDispatchMechanicPathsMutuallyExclusive(t *testing.T) {
	cfg := DefaultInformationConfig()
	assertSettledOnce := func(t *testing.T, silo *model.Silo) {
		t.Helper()
		belief := silo.Professions[0].Beliefs[0]
		wantB := 0.55 - cfg.ForgetRateMonthly
		if math.Abs(belief.B-wantB) >= 1e-12 || belief.PermeationCounter != 1 {
			t.Fatalf("monthly dispatch ran more or less than once: %+v, want B=%v counter=1", belief, wantB)
		}
	}

	t.Run("JS success", func(t *testing.T) {
		e := NewGameEngine()
		const id = "monthly_dispatch_test"
		e.MechanicEngine.byEventType[EVENT_METRICS_UPDATE] = MechanicDefinition{
			ID: id, EventType: EVENT_METRICS_UPDATE,
			ScriptSource: `defineMechanics([{id:"monthly_dispatch_test",event_type:"METRICS_UPDATE",apply:function(){return {mutations:[{type:"silo_metric_set",metric:"countdown",value:42}]};}}]);`,
		}
		silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
		e.runEventMechanic(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), NewEventContext())
		if silo.Countdown != 42 {
			t.Fatalf("JS success-path sentinel missing (Go fallback may have run): countdown=%v", silo.Countdown)
		}
		assertSettledOnce(t, silo)
	})

	t.Run("Go fallback", func(t *testing.T) {
		e := NewGameEngine()
		e.MechanicEngine = nil
		silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
		e.runEventMechanic(EVENT_METRICS_UPDATE, metricsUpdateEvent(silo, 1.0/12), NewEventContext())
		assertSettledOnce(t, silo)
	})
}

func TestForgettingDispatchLayerIsolation(t *testing.T) {
	e := &GameEngine{}
	silo := permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	agent := &model.Agent{ID: 1, Beliefs: []model.FactBelief{{FactID: "IT_1", B: 0.9}}}
	silo.Protocol.DirectorBeliefs = []model.FactBelief{{FactID: "Order_1", B: 0.9}}
	agentBefore := append([]model.FactBelief(nil), agent.Beliefs...)
	directorBefore := append([]model.FactBelief(nil), silo.Protocol.DirectorBeliefs...)

	dispatchMonth(e, silo)

	if !reflect.DeepEqual(agent.Beliefs, agentBefore) {
		t.Fatalf("agent layer must be untouched by the dispatch: %+v", agent.Beliefs)
	}
	if !reflect.DeepEqual(silo.Protocol.DirectorBeliefs, directorBefore) {
		t.Fatalf("director layer must be untouched by the dispatch: %+v", silo.Protocol.DirectorBeliefs)
	}
}

func TestForgettingDispatchDeterminism(t *testing.T) {
	e := &GameEngine{}
	build := func() *model.Silo {
		return permeationTestSilo(model.FactBelief{FactID: "IT_1", B: 0.55})
	}
	a, b := build(), build()
	for i := 0; i < 3; i++ {
		dispatchMonth(e, a)
		dispatchMonth(e, b)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same-state silos diverged across two dispatch runs:\n%+v\n%+v", a, b)
	}
}

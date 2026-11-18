package service

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// pressureFixture 建真实会话 (门控+内容+mechanics 全加载)，威望置于 incite
// 解锁线，AP 充足；返回服务与第一个平民部门下标。
func pressureFixture(t *testing.T) (*GameService, int) {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "pressure.db"))
	req := threeSiloRequest(701)
	req.Profession = "Mechanical"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.agent.ActionPoints = 500
	svc.agent.Ambition = 0.6
	svc.agent.Influence = 0.6
	// incite 的可用门还要求 Mayor 关系 ≥0.25：fixture 直接给足。
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == "Mayor" {
			svc.agent.Connections = append(svc.agent.Connections,
				model.Connection{ProfessionID: svc.silo.Professions[i].ID, Value: 0.4})
		}
	}
	commoner := -1
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].ClassType == "COMMONER" {
			commoner = i
			break
		}
	}
	if commoner < 0 {
		t.Fatal("fixture silo has no commoner profession")
	}
	return svc, commoner
}

func inciteOutcome(t *testing.T, svc *GameService) (*model.ActionOutcome, float64) {
	t.Helper()
	before := svc.silo.Rebellion
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionInciteRebellion})
	if err != nil {
		t.Fatalf("execute incite: %v", err)
	}
	if !outcome.Result.Executed {
		t.Fatalf("incite rejected: %+v", outcome.Result)
	}
	return outcome, before
}

// TestInciteRebellionHasObservableDelta 锁定效应量落地：煽动对叛乱有可观测、
// 可复现的即时增量 = round((0.05+威望*0.002)*(1+宣传*0.2)*0.5, 3)，文案报出
// 数值与乘数；恐慌的部门加权效应保留。
func TestInciteRebellionHasObservableDelta(t *testing.T) {
	svc, commoner := pressureFixture(t)
	svc.agent.PropagandaLevel = 2
	applied := round3((0.05 + 60*0.002) * (1 + 2*0.2) * 1.0)
	panicBefore := svc.silo.Professions[commoner].PanicValue
	outcome, before := inciteOutcome(t, svc)
	// incite 耗时 2 月，月度派系漂移会叠加：断言即时增量落在以 applied 为中心
	// 的小窗口内 (漂移量级 ≤0.05/两月)，精确值由文案锁定。
	got := svc.silo.Rebellion - before
	if got < applied-0.02 || got > applied+0.06 {
		t.Fatalf("rebellion delta = %.4f, want %.3f ± drift", got, applied)
	}
	msg := outcome.Result.Message
	if !strings.Contains(msg, fmt.Sprintf("Rebellion +%.3f", applied)) {
		t.Fatalf("message must report the delta, got %q", msg)
	}
	if !strings.Contains(msg, "multiplier x1.4") {
		t.Fatalf("message must report the propaganda multiplier, got %q", msg)
	}
	if svc.silo.Professions[commoner].PanicValue <= panicBefore {
		t.Fatalf("commoner panic must still rise, got %.3f from %.3f",
			svc.silo.Professions[commoner].PanicValue, panicBefore)
	}
}

// TestPropagandaMultiplierScalesIncite 锁定宣传的乘数效应可观测：同威望下
// 宣传 0 与 3 的即时增量之比 = 1.6/1.0。
func TestPropagandaMultiplierScalesIncite(t *testing.T) {
	w0 := round3((0.05 + 60*0.002) * 1.0 * 1.0)
	w3 := round3((0.05 + 60*0.002) * 1.6 * 1.0)

	svc, _ := pressureFixture(t)
	svc.agent.PropagandaLevel = 0
	out0, before0 := inciteOutcome(t, svc)
	d0 := svc.silo.Rebellion - before0

	svc2, _ := pressureFixture(t)
	svc2.agent.PropagandaLevel = 3
	out2, before3 := inciteOutcome(t, svc2)
	d3 := svc2.silo.Rebellion - before3

	// 文案锁定精确增量与乘数 (漂移无关)。
	if !strings.Contains(out0.Result.Message, fmt.Sprintf("Rebellion +%.3f", w0)) ||
		!strings.Contains(out0.Result.Message, "multiplier x1.0") {
		t.Fatalf("propaganda-0 message must report %.3f and x1.0, got %q", w0, out0.Result.Message)
	}
	if !strings.Contains(out2.Result.Message, fmt.Sprintf("Rebellion +%.3f", w3)) ||
		!strings.Contains(out2.Result.Message, "multiplier x1.6") {
		t.Fatalf("propaganda-3 message must report %.3f and x1.6, got %q", w3, out2.Result.Message)
	}
	// 实测增量之差 = 乘数差 (漂移在同长度窗口内相消)。
	diff := d3 - d0
	if diff < (w3-w0)-0.01 || diff > (w3-w0)+0.01 {
		t.Fatalf("delta difference = %.4f, want %.3f", diff, w3-w0)
	}
}

// TestInciteHonestMessageAtCap 锁定诚实反馈：叛乱顶到上限时煽动报"没有起作用"，
// 不再假装成功。
func TestInciteHonestMessageAtCap(t *testing.T) {
	svc, _ := pressureFixture(t)
	svc.silo.Rebellion = 1.0
	outcome, before := inciteOutcome(t, svc)
	if svc.silo.Rebellion > before {
		t.Fatalf("rebellion must not rise at cap, got %.3f from %.3f", svc.silo.Rebellion, before)
	}
	if !strings.Contains(outcome.Result.Message, "没有起作用") {
		t.Fatalf("capped incite must say so, got %q", outcome.Result.Message)
	}
}

// TestUltimatumIssuedUnderPressureInFullSim 锁定错线修复：全模拟 1 号模式下
// 状态机不再早退，复合压力越线即签发带条件的通牒 (跨堡事件入队)；压力回落
// 则不签发 (门槛两侧都验)。
func TestUltimatumIssuedUnderPressureInFullSim(t *testing.T) {
	svc, station, _ := weightedEndingFixture("TIME")
	station.VictoryStatus = nil // fixture 带结局态，这里只测状态机
	station.Professions = []model.Profession{
		{ID: 1, Name: "Mechanical", ClassType: "COMMONER", PanicValue: 0.7},
		{ID: 2, Name: "Mines", ClassType: "COMMONER", PanicValue: 0.7},
	}
	station.Rebellion = 0.5 // pressure = 0.6*0.5 + 0.4*0.7 = 0.58 >= 0.45
	svc.agent.PropagandaLevel = 3            // 有组织煽动条件满足
	svc.agents = map[uint]*model.Agent{station.ID: svc.agent}
	svc.world.Silo1.OrderPhase = "LOCAL_SIM" // 全模拟模式的真实观察相位
	if svc.fullSilo1() == nil {
		t.Fatal("fixture must provide a full Silo 1 peer")
	}
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1StateMachine(&logs, &stories)
	if svc.world.Silo1.OrderPhase != "ULTIMATUM" {
		t.Fatalf("order phase = %q, want ULTIMATUM under pressure in full-sim mode", svc.world.Silo1.OrderPhase)
	}
	if len(svc.world.CrossQueue) != 1 || svc.world.CrossQueue[0].EventID != "silo1_ultimatum_cross" {
		t.Fatalf("cross queue must carry the conditional ultimatum, got %+v", svc.world.CrossQueue)
	}

	// 负例：压力越线但没有有组织煽动 (宣传等级 0)——背景热度不招致通牒。
	svc2, station2, _ := weightedEndingFixture("TIME")
	station2.VictoryStatus = nil
	station2.Professions = []model.Profession{
		{ID: 1, Name: "Mechanical", ClassType: "COMMONER", PanicValue: 0.7},
		{ID: 2, Name: "Mines", ClassType: "COMMONER", PanicValue: 0.7},
	}
	station2.Rebellion = 0.5 // pressure = 0.58 >= 0.45，但 propaganda = 0
	svc2.agents = map[uint]*model.Agent{station2.ID: svc2.agent}
	svc2.world.Silo1.OrderPhase = "LOCAL_SIM"
	logs2, stories2 := []string{}, []model.StoryEvent{}
	svc2.runSilo1StateMachine(&logs2, &stories2)
	if svc2.world.Silo1.OrderPhase != "LOCAL_SIM" || len(svc2.world.CrossQueue) != 0 {
		t.Fatalf("high pressure without organized agitation must not issue an ultimatum: phase=%q queue=%+v",
			svc2.world.Silo1.OrderPhase, svc2.world.CrossQueue)
	}
}

// TestOrderPressureAndRelativeMonthTrigger 锁定复合度量与相对月触发器：
// 压力 = 0.6*叛乱 + 0.4*平均恐慌；相对门锚定事件点亮月，未点亮不成立。
func TestOrderPressureAndRelativeMonthTrigger(t *testing.T) {
	silo := &model.Silo{
		CurrentYear: 124, CurrentMonth: 5,
		Rebellion:    0.5,
		Professions:  []model.Profession{{PanicValue: 0.3}, {PanicValue: 0.5}},
	}
	want := 0.6*0.5 + 0.4*0.4
	if got := engine.OrderPressure(silo); got != want {
		t.Fatalf("OrderPressure = %.3f, want %.3f", got, want)
	}

	states := map[string]engine.ContentEventRuntime{
		"special:silo1_ultimatum_cross": {Triggered: true, LastTriggeredTimestamp: gameTimestamp(124, 1)},
	}
	ctx := engine.ContentEvaluationContext{Silo: silo, States: states}
	holds := func(months int) bool {
		def := engine.ContentEventDefinition{
			Key:     "special:probe",
			Trigger: engine.ContentTrigger{Type: "months_since_event_gte", Category: "special", EventID: "silo1_ultimatum_cross", Months: months},
		}
		return engine.CanTriggerContentEvent(def, ctx)
	}
	if !holds(4) {
		t.Fatal("4 months after the ultimatum the relative gate must hold")
	}
	if holds(5) {
		t.Fatal("5-month gate must not hold at 4 months elapsed")
	}
	missing := engine.ContentEventDefinition{
		Key:     "special:probe",
		Trigger: engine.ContentTrigger{Type: "months_since_event_gte", Category: "special", EventID: "nope", Months: 0},
	}
	if engine.CanTriggerContentEvent(missing, ctx) {
		t.Fatal("unlit event must not satisfy the relative gate")
	}
}

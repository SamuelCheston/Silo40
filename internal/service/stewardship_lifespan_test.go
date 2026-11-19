package service

// 指标二 (剩余人员平均寿命) 测试：22 设计卷 §4 的五类断言——
// 自然衰减、三档成本分档、一次性成本台账去重、耗尽退化级联与耗尽守卫
// (多堡服务真实驱动)、未启用状态空转。

import (
	"math"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// newLifespanGame 真实三堡会话 (内容全加载)。
func newLifespanGame(t *testing.T, seed int64) *GameService {
	t.Helper()
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "lifespan.db"))
	req := threeSiloRequest(seed)
	req.Silo1BlackBox = true // 功能开关：权衡层启用
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

// runLifespanMonths 直接驱动指标二月度回合 n 次。
func runLifespanMonths(t *testing.T, svc *GameService, n int) []string {
	t.Helper()
	var logs []string
	for i := 0; i < n; i++ {
		stories := []model.StoryEvent{}
		svc.stewardshipLifespanMonth(&logs, &stories)
	}
	return logs
}

func lifespanAlmostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestStewardshipLifespanNaturalDecayWithoutAction 不操作也自然下降：真实管线
// 推进 N 月，无档位时下降量 = 自然衰减 × N。
func TestStewardshipLifespanNaturalDecayWithoutAction(t *testing.T) {
	svc := newLifespanGame(t, 901)
	svc.initSilo1Stewardship()
	ls := &svc.world.Silo1Stewardship.Lifespan
	ls.Postures = map[uint]string{} // 无档位：只剩自然衰减
	start := ls.AvgLifespan
	if start != stewardshipLifespanStart {
		t.Fatalf("initial lifespan = %.3f, want %.1f", start, stewardshipLifespanStart)
	}

	const months = 7
	for i := 0; i < months; i++ {
		advanceMonthForTest(t, svc)
	}
	if svc.world.CurrentYear != 122 || svc.world.CurrentMonth != 1+months {
		t.Fatalf("world clock did not advance: Y%d M%d", svc.world.CurrentYear, svc.world.CurrentMonth)
	}
	drop := start - ls.AvgLifespan
	if !lifespanAlmostEqual(drop, months*lifespanNaturalDecay) {
		t.Fatalf("drop over %d months = %.6f, want natural decay %.6f only",
			months, drop, months*lifespanNaturalDecay)
	}
	if len(ls.Ledger) != 0 || len(ls.Postures) != 0 {
		t.Fatalf("no-op months must not touch ledger/postures: ledger=%+v postures=%+v", ls.Ledger, ls.Postures)
	}
}

// TestStewardshipLifespanPostureCostsTiered 三档成本分档：同月数下总消耗
// HIGH > NORMAL > REPORT_ONLY (对比数字)。
func TestStewardshipLifespanPostureCostsTiered(t *testing.T) {
	const months = 5
	dropFor := func(posture string) (float64, int) {
		svc := newLifespanGame(t, 902)
		svc.initSilo1Stewardship()
		ls := &svc.world.Silo1Stewardship.Lifespan
		for siloID := range ls.Postures {
			ls.Postures[siloID] = posture
		}
		start := ls.AvgLifespan
		runLifespanMonths(t, svc, months)
		return start - ls.AvgLifespan, len(ls.Postures)
	}
	dropHigh, siloCount := dropFor(model.Silo1PostureHigh)
	dropNormal, _ := dropFor(model.Silo1PostureNormal)
	dropMin, _ := dropFor(model.Silo1PostureReportOnly)
	if siloCount != 3 {
		t.Fatalf("three-silo game must monitor three silos, got %d", siloCount)
	}

	if !(dropHigh > dropNormal && dropNormal > dropMin) {
		t.Fatalf("tiered drops must be strictly ordered, got HIGH=%.4f NORMAL=%.4f REPORT_ONLY=%.4f",
			dropHigh, dropNormal, dropMin)
	}
	// 三堡同档：每月消耗 = 自然衰减 + 3×档位成本。
	wantHigh := months * (lifespanNaturalDecay + float64(siloCount)*postureCostHigh)
	wantNormal := months * (lifespanNaturalDecay + float64(siloCount)*postureCostNormal)
	wantMin := months * (lifespanNaturalDecay + float64(siloCount)*postureCostMin)
	if !lifespanAlmostEqual(dropHigh, wantHigh) ||
		!lifespanAlmostEqual(dropNormal, wantNormal) ||
		!lifespanAlmostEqual(dropMin, wantMin) {
		t.Fatalf("drops = (%.4f, %.4f, %.4f), want (%.4f, %.4f, %.4f)",
			dropHigh, dropNormal, dropMin, wantHigh, wantNormal, wantMin)
	}
}

// TestStewardshipLifespanOneTimeCostsChargeOnce 一次性成本按台账只计一次：
// 整肃/终止点火按堡去重，通牒按全局键去重。
func TestStewardshipLifespanOneTimeCostsChargeOnce(t *testing.T) {
	svc := newLifespanGame(t, 904)
	svc.initSilo1Stewardship()
	ls := &svc.world.Silo1Stewardship.Lifespan
	start := ls.AvgLifespan
	postureMonthly := 0.0
	for _, posture := range ls.Postures {
		postureMonthly += postureMonthlyCost(posture)
	}

	// 点火事实：1 号堡整肃征兆、3 号堡终止信道测试。
	svc.statesMapFor(1)["special:silo1_purge_omen"] = model.ContentEventState{
		SiloID: 1, EventKey: "special:silo1_purge_omen", Triggered: true, TriggerCount: 1,
	}
	svc.statesMapFor(3)["special:termination_channel_test"] = model.ContentEventState{
		SiloID: 3, EventKey: "special:termination_channel_test", Triggered: true, TriggerCount: 1,
	}

	runLifespanMonths(t, svc, 1)
	want := start - (lifespanNaturalDecay + postureMonthly + opCostPurge + opCostTermination)
	if !lifespanAlmostEqual(ls.AvgLifespan, want) {
		t.Fatalf("after ignition month lifespan = %.6f, want %.6f", ls.AvgLifespan, want)
	}
	if ls.Ledger["purge:1"] != 1 || ls.Ledger["termination:3"] != 1 {
		t.Fatalf("ignition charges must be ledgered once per silo, got %+v", ls.Ledger)
	}

	// 点火事实仍在 (内容状态长期为真)，第二月不得重复计费。
	before := ls.AvgLifespan
	runLifespanMonths(t, svc, 1)
	if !lifespanAlmostEqual(before-ls.AvgLifespan, lifespanNaturalDecay+postureMonthly) {
		t.Fatalf("second month must charge only decay+posture: drop=%.6f, want %.6f",
			before-ls.AvgLifespan, lifespanNaturalDecay+postureMonthly)
	}
	if ls.Ledger["purge:1"] != 1 || ls.Ledger["termination:3"] != 1 {
		t.Fatalf("ledger entries must stay 1 (never recharged), got %+v", ls.Ledger)
	}

	// 通牒成本：重复调用只计一次；未知种类不扣。
	before = ls.AvgLifespan
	svc.stewardshipChargeLifespan("ultimatum")
	svc.stewardshipChargeLifespan("ultimatum")
	svc.stewardshipChargeLifespan("bogus")
	if !lifespanAlmostEqual(before-ls.AvgLifespan, opCostUltimatum) {
		t.Fatalf("ultimatum must charge exactly %.1f once, drop=%.6f", opCostUltimatum, before-ls.AvgLifespan)
	}
	if ls.Ledger["ultimatum"] != 1 {
		t.Fatalf("ultimatum ledger = %d, want 1", ls.Ledger["ultimatum"])
	}
}

// TestStewardshipLifespanExhaustionCascadeBlocksUltimatums 耗尽退化级联与
// exhausted 守卫阻断新通牒 (多堡服务真实管线驱动)。
func TestStewardshipLifespanExhaustionCascadeBlocksUltimatums(t *testing.T) {
	svc := newLifespanGame(t, 905)

	// 真实管线第 1 月：调度器初始化并挂载指标二。
	advanceMonthForTest(t, svc)
	st := &svc.world.Silo1Stewardship
	if !st.Initialized || !st.Lifespan.Initialized {
		t.Fatalf("stewardship must initialize through the real pipeline: %+v", st)
	}
	want := stewardshipLifespanStart - lifespanNaturalDecay - 3*postureCostNormal
	if !lifespanAlmostEqual(st.Lifespan.AvgLifespan, want) {
		t.Fatalf("month-1 lifespan = %.6f, want %.6f (mount point active)", st.Lifespan.AvgLifespan, want)
	}

	// 级联 A：跌破退化阈值 → 所有 HIGH 降 NORMAL (其余档位不动)。
	ls := &st.Lifespan
	ls.AvgLifespan = postureDowngradeThreshold + lifespanNaturalDecay +
		postureCostHigh + postureCostNormal + postureCostMin - 0.05
	siloIDs := sortedPostureIDs(ls.Postures)
	if len(siloIDs) != 3 {
		t.Fatalf("expected three monitored silos, got %v", siloIDs)
	}
	ls.Postures[siloIDs[0]] = model.Silo1PostureHigh
	ls.Postures[siloIDs[1]] = model.Silo1PostureNormal
	ls.Postures[siloIDs[2]] = model.Silo1PostureReportOnly
	advanceMonthForTest(t, svc)
	if ls.AvgLifespan > postureDowngradeThreshold || ls.AvgLifespan <= 0 {
		t.Fatalf("lifespan %.6f must land in (0, %.1f] to exercise the downgrade branch",
			ls.AvgLifespan, postureDowngradeThreshold)
	}
	if ls.Postures[siloIDs[0]] != model.Silo1PostureNormal {
		t.Fatalf("HIGH posture must be downgraded to NORMAL at threshold, got %q", ls.Postures[siloIDs[0]])
	}
	if ls.Postures[siloIDs[1]] != model.Silo1PostureNormal || ls.Postures[siloIDs[2]] != model.Silo1PostureReportOnly {
		t.Fatalf("non-HIGH postures must be untouched, got %+v", ls.Postures)
	}
	if svc.stewardshipLifespanExhausted() {
		t.Fatal("downgrade threshold alone must not set exhausted")
	}

	// 级联 B：跌破 0 → 全部强制 REPORT_ONLY、台账 exhausted=1。
	ls.AvgLifespan = 0.05
	advanceMonthForTest(t, svc)
	if ls.AvgLifespan != 0 {
		t.Fatalf("exhausted lifespan must clamp to 0, got %.6f", ls.AvgLifespan)
	}
	for _, siloID := range siloIDs {
		if ls.Postures[siloID] != model.Silo1PostureReportOnly {
			t.Fatalf("exhaustion must force silo %d to REPORT_ONLY, got %q", siloID, ls.Postures[siloID])
		}
	}
	if ls.Ledger[lifespanLedgerExhausted] != 1 || !svc.stewardshipLifespanExhausted() {
		t.Fatalf("exhausted flag must be ledgered and readable: %+v", ls.Ledger)
	}

	// 耗尽守卫：通牒条件全部满足也不得签发新通牒。
	svc.silo.Rebellion = 0.95
	svc.agent.PropagandaLevel = 3
	svc.world.Silo1.OrderPhase = "LOCAL_SIM"
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1StateMachine(&logs, &stories)
	if svc.world.Silo1.OrderPhase != "LOCAL_SIM" {
		t.Fatalf("exhausted stewardship must keep order phase LOCAL_SIM, got %q", svc.world.Silo1.OrderPhase)
	}
	for _, entry := range svc.world.CrossQueue {
		if entry.EventID == "silo1_ultimatum_cross" {
			t.Fatalf("exhausted stewardship must block new ultimatums, queue=%+v", svc.world.CrossQueue)
		}
	}
	if ls.Ledger[lifespanLedgerUltimatum] != 0 {
		t.Fatalf("no ultimatum charge may be ledgered while blocked: %+v", ls.Ledger)
	}

	// 对照：解除耗尽后同样条件照常签发并计费 (已点火阶梯不受守卫影响的另一面)。
	ls.Ledger[lifespanLedgerExhausted] = 0
	ls.AvgLifespan = 5.0
	before := ls.AvgLifespan
	logs2, stories2 := []string{}, []model.StoryEvent{}
	svc.runSilo1StateMachine(&logs2, &stories2)
	if svc.world.Silo1.OrderPhase != "ULTIMATUM" {
		t.Fatalf("non-exhausted stewardship must still issue the ultimatum, phase=%q", svc.world.Silo1.OrderPhase)
	}
	found := false
	for _, entry := range svc.world.CrossQueue {
		if entry.EventID == "silo1_ultimatum_cross" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ultimatum must be enqueued when not exhausted, queue=%+v", svc.world.CrossQueue)
	}
	if ls.Ledger[lifespanLedgerUltimatum] != 1 || !lifespanAlmostEqual(before-ls.AvgLifespan, opCostUltimatum) {
		t.Fatalf("ultimatum issue must charge %.1f once: drop=%.6f ledger=%+v",
			opCostUltimatum, before-ls.AvgLifespan, ls.Ledger)
	}
}

// TestStewardshipLifespanDisabledStatesAreNoOps 未启用状态空转：零值状态与
// 单堡配置下本层不写任何状态。
func TestStewardshipLifespanDisabledStatesAreNoOps(t *testing.T) {
	// 零值世界：所有入口早退，不 panic、不落状态。
	svc := &GameService{world: &model.World{}}
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1StewardshipMonth(&logs, &stories)
	svc.stewardshipLifespanMonth(&logs, &stories)
	svc.stewardshipChargeLifespan("ultimatum")
	if svc.stewardshipLifespanExhausted() {
		t.Fatal("zero-value stewardship must never report exhausted")
	}
	if svc.world.Silo1Stewardship.Initialized || svc.world.Silo1Stewardship.Lifespan.Initialized {
		t.Fatalf("zero-value world must stay uninitialized: %+v", svc.world.Silo1Stewardship)
	}

	// 单堡配置：真实管线推进数月，调度器早退，权衡层保持零值。
	single := newTwoSiloService(t, filepath.Join(t.TempDir(), "lifespan-single.db"))
	if _, err := single.CreateGame(model.CreateGameRequest{
		SiloName: "Lone", AgentName: "Player", Profession: "IT", Seed: 906,
	}); err != nil {
		t.Fatalf("create single-silo game: %v", err)
	}
	for i := 0; i < 3; i++ {
		advanceMonthForTest(t, single)
	}
	if single.world.Silo1Stewardship.Initialized || single.world.Silo1Stewardship.Lifespan.AvgLifespan != 0 {
		t.Fatalf("single-silo config must leave stewardship untouched: %+v", single.world.Silo1Stewardship)
	}
}

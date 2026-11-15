package service

// 黑箱回合 (格三) 测试：22 黑箱学习系列 01 卷的验收面——
// 先验形状、后验移动方向 (含伪装与 ε 地板防锁死)、抽干排序、argmax 平局与
// 菜单上界、日志字段与环形截断、专线干扰丢弃、门控零变化零 RNG、探索接口
// 与冷却位置、AST 黑箱壁守卫。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// newBlackBoxGame 功能开关开启的真实三堡会话 (内容全加载)。
func newBlackBoxGame(t *testing.T, seed int64) *GameService {
	t.Helper()
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "blackbox.db"))
	req := threeSiloRequest(seed)
	req.Silo1BlackBox = true // 功能开关：黑箱回合启用
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

// blackBoxObs 构造一条 (到达月 122/1、反映月 121/12) 的观测。
func blackBoxObs(signal string, y float64, seq int) model.BlackBoxPendingObservation {
	return model.BlackBoxPendingObservation{
		ArriveYear:     122,
		ArriveMonth:    1,
		ReflectedYear:  121,
		ReflectedMonth: 12,
		SignalType:     signal,
		Value:          y,
		Seq:            seq,
	}
}

// blackBoxOnMenu r 是否落在决策菜单上。
func blackBoxOnMenu(r float64) bool {
	for _, m := range blackBoxMenu {
		if r == m {
			return true
		}
	}
	return false
}

// blackBoxSampleBeliefs 一组覆盖形状的信念样本 (先验、各箱点质量、均匀)。
func blackBoxSampleBeliefs() [][]float64 {
	beliefs := [][]float64{blackBoxPrior()}
	for k := 0; k < blackBoxBins; k++ {
		b := make([]float64, blackBoxBins)
		b[k] = 1
		beliefs = append(beliefs, b)
	}
	uniform := make([]float64, blackBoxBins)
	for i := range uniform {
		uniform[i] = 1.0 / float64(blackBoxBins)
	}
	return append(beliefs, uniform)
}

// TestBlackBoxPriorShape 先验：归一化、低箱重 (单调递减)、ε 地板严格为正。
func TestBlackBoxPriorShape(t *testing.T) {
	belief := blackBoxPrior()
	if len(belief) != blackBoxBins {
		t.Fatalf("prior bins = %d, want %d", len(belief), blackBoxBins)
	}
	sum := 0.0
	for i, p := range belief {
		if p <= 0 {
			t.Fatalf("prior floor must keep every bin strictly positive, bin %d = %v", i, p)
		}
		sum += p
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("prior must normalize, sum = %.15f", sum)
	}
	for i := 1; i < len(belief); i++ {
		if belief[i] >= belief[i-1] {
			t.Fatalf("prior must concentrate on low bins (strictly decreasing): %v", belief)
		}
	}
	// 地板下界：b(i) = [(1−ε)exp(−λθ_i)+ε]/Z ≥ ε/N (Z ≤ N)。
	if belief[len(belief)-1] < blackBoxPriorFloor/float64(blackBoxBins) {
		t.Fatalf("top bin %.6f below the ε-floor bound %.6f", belief[len(belief)-1], blackBoxPriorFloor/float64(blackBoxBins))
	}
}

// TestBlackBoxPosteriorRisesWithHighObservations 高 y 观测推高 Θ̂ 与危险区概率。
func TestBlackBoxPosteriorRisesWithHighObservations(t *testing.T) {
	prior := blackBoxPrior()
	thetaBefore, dangerBefore, _, _ := blackBoxSummarize(prior)

	belief := blackBoxPrior()
	for i := 0; i < 3; i++ {
		blackBoxUpdateBelief(belief, blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.9, i+1))
	}
	thetaAfter, dangerAfter, _, _ := blackBoxSummarize(belief)
	if thetaAfter <= thetaBefore {
		t.Fatalf("high observations must raise Θ̂: %.4f -> %.4f", thetaBefore, thetaAfter)
	}
	if dangerAfter <= dangerBefore {
		t.Fatalf("high observations must raise P_danger: %.4f -> %.4f", dangerBefore, dangerAfter)
	}
}

// TestBlackBoxDisguisedObservationsStillRaiseSuspicion 被伪装偏移的观测
// (高 Θ 粉饰后的中等读数) 仍有机会被识别为高 Θ：混合似然的伪装分量承接它。
func TestBlackBoxDisguisedObservationsStillRaiseSuspicion(t *testing.T) {
	prior := blackBoxPrior()
	thetaBefore, dangerBefore, _, _ := blackBoxSummarize(prior)

	// Θ' = 0.80 的 line_tone 被伪装 (−0.25) 后落在 y ≈ 0.55。
	belief := blackBoxPrior()
	for i := 0; i < 4; i++ {
		blackBoxUpdateBelief(belief, blackBoxObs(model.BlackBoxSignalLineTone, 0.55, i+1))
	}
	thetaAfter, dangerAfter, _, _ := blackBoxSummarize(belief)
	if thetaAfter <= thetaBefore {
		t.Fatalf("disguised high-Θ readings must still raise Θ̂: %.4f -> %.4f", thetaBefore, thetaAfter)
	}
	if dangerAfter <= dangerBefore {
		t.Fatalf("disguised high-Θ readings must still raise P_danger: %.4f -> %.4f", dangerBefore, dangerAfter)
	}
}

// TestBlackBoxFloorPreventsLockIn ε 地板防锁死：一长串低读数把后验压到低箱后，
// 所有箱仍严格为正，随后的高读数序列能把 Θ̂ 实质性翻转回来。
func TestBlackBoxFloorPreventsLockIn(t *testing.T) {
	belief := blackBoxPrior()
	for i := 0; i < 12; i++ {
		blackBoxUpdateBelief(belief, blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.05, i+1))
	}
	lowHat, _, _, _ := blackBoxSummarize(belief)
	for i, p := range belief {
		if p <= 0 {
			t.Fatalf("floor violated after low readings: bin %d = %v", i, p)
		}
	}
	for i := 0; i < 6; i++ {
		blackBoxUpdateBelief(belief, blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.9, i+101))
	}
	recovered, _, _, _ := blackBoxSummarize(belief)
	if recovered <= lowHat+0.1 {
		t.Fatalf("posterior must recover from the low lock-in: %.4f -> %.4f", lowHat, recovered)
	}
}

// TestBlackBoxDrainOrderAndFutureKept 抽干判据与排序键 (到达月, 信号类型, seq)
// 升序；未到月的观测留在途。
func TestBlackBoxDrainOrderAndFutureKept(t *testing.T) {
	pending := []model.BlackBoxPendingObservation{
		{ArriveYear: 122, ArriveMonth: 2, SignalType: model.BlackBoxSignalReceiptDelay, Value: 0.5, Seq: 9},
		{ArriveYear: 122, ArriveMonth: 1, SignalType: model.BlackBoxSignalReceiptDelay, Value: 0.5, Seq: 2},
		{ArriveYear: 122, ArriveMonth: 1, SignalType: model.BlackBoxSignalLineTone, Value: 0.5, Seq: 1},
		{ArriveYear: 122, ArriveMonth: 1, SignalType: model.BlackBoxSignalReceiptDelay, Value: 0.5, Seq: 1},
		{ArriveYear: 121, ArriveMonth: 12, SignalType: model.BlackBoxSignalExecutionGap, Value: 0.5, Seq: 0},
	}
	arrived := blackBoxDrainArrived(&pending, 122, 1)
	type key struct {
		year, month int
		signal      string
		seq         int
	}
	want := []key{
		{121, 12, model.BlackBoxSignalExecutionGap, 0},
		{122, 1, model.BlackBoxSignalLineTone, 1},
		{122, 1, model.BlackBoxSignalReceiptDelay, 1},
		{122, 1, model.BlackBoxSignalReceiptDelay, 2},
	}
	if len(arrived) != len(want) {
		t.Fatalf("drained %d observations, want %d (future must stay pending)", len(arrived), len(want))
	}
	for i, w := range want {
		got := key{arrived[i].ArriveYear, arrived[i].ArriveMonth, arrived[i].SignalType, arrived[i].Seq}
		if got != w {
			t.Fatalf("drain order[%d] = %+v, want %+v", i, got, w)
		}
	}
	if len(pending) != 1 || pending[0].ArriveMonth != 2 {
		t.Fatalf("future observation must remain pending, got %+v", pending)
	}
}

// TestBlackBoxMenuArgmaxTieKeepsSmallerR 平局保留更小 r (升序扫描、只有严格
// 更高分才替换)。
func TestBlackBoxMenuArgmaxTieKeepsSmallerR(t *testing.T) {
	// 全域平分：每一点都平局 → 保留最小 r。
	if r, _ := blackBoxMenuArgmax(func(float64) float64 { return 0.5 }); r != blackBoxMenu[0] {
		t.Fatalf("full tie must keep the smallest r, got %v", r)
	}
	// 两点同高：保留更小的那一个。
	score := func(r float64) float64 {
		if r == blackBoxMenu[6] || r == blackBoxMenu[10] {
			return 1
		}
		return 0
	}
	if r, _ := blackBoxMenuArgmax(score); r != blackBoxMenu[6] {
		t.Fatalf("two-way tie must keep the smaller r %v, got %v", blackBoxMenu[6], r)
	}
}

// TestBlackBoxArgmaxSelectsSmallestMaximizer 真实效用下 argmax 恒选最小最优解
// (对多种信念逐一重算全部菜单点核对)。
func TestBlackBoxArgmaxSelectsSmallestMaximizer(t *testing.T) {
	for _, belief := range blackBoxSampleBeliefs() {
		chosen, best := blackBoxArgmax(belief)
		wantR := math.Inf(-1)
		for _, r := range blackBoxMenu {
			if u := blackBoxUtility(belief, r); u == best {
				wantR = r
				break
			}
		}
		if math.IsInf(wantR, -1) {
			t.Fatalf("no menu point reproduces the reported utility %v", best)
		}
		if chosen != wantR {
			t.Fatalf("argmax must pick the smallest maximizer: chose %v, want %v", chosen, wantR)
		}
	}
}

// TestBlackBoxMenuUpperBoundNeverCrossed 菜单上界 0.95 永不被越过。
func TestBlackBoxMenuUpperBoundNeverCrossed(t *testing.T) {
	if len(blackBoxMenu) != blackBoxMenuPoints {
		t.Fatalf("menu size = %d, want %d", len(blackBoxMenu), blackBoxMenuPoints)
	}
	if blackBoxMenu[len(blackBoxMenu)-1] != 0.95 {
		t.Fatalf("menu upper bound = %.17f, want exactly 0.95", blackBoxMenu[len(blackBoxMenu)-1])
	}
	for i, r := range blackBoxMenu {
		if r < 0 || r > 0.95 {
			t.Fatalf("menu[%d] = %.17f outside [0, 0.95]", i, r)
		}
		if i > 0 && r <= blackBoxMenu[i-1] {
			t.Fatalf("menu must be strictly ascending at %d", i)
		}
	}
	for _, belief := range blackBoxSampleBeliefs() {
		if r, _ := blackBoxArgmax(belief); r > 0.95 {
			t.Fatalf("argmax crossed the menu upper bound: %v", r)
		}
	}
}

// TestBlackBoxRoundAssignsAndLogs 回合集成：逐堡初始化先验、抽干到期观测、
// 指派落点与日志字段齐全；日志不进故事事件与玩家日志；未来观测留待下月。
func TestBlackBoxRoundAssignsAndLogs(t *testing.T) {
	svc := newBlackBoxGame(t, 301)
	target := svc.siloByID(2)
	if target == nil {
		t.Fatal("three-silo game must contain silo 2")
	}
	target.BlackBox.PendingObservations = []model.BlackBoxPendingObservation{
		blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.8, 1),
		{ArriveYear: 122, ArriveMonth: 2, ReflectedYear: 121, ReflectedMonth: 12,
			SignalType: model.BlackBoxSignalExecutionGap, Value: 0.6, Seq: 2},
	}

	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1BlackBoxRound(&logs, &stories)

	if len(stories) != 0 {
		t.Fatalf("black-box round must not emit story events: %+v", stories)
	}
	if len(logs) != 0 {
		t.Fatalf("black-box round must not emit player logs: %+v", logs)
	}
	for _, silo := range svc.silos {
		bb := &silo.BlackBox
		if len(bb.Belief) != blackBoxBins {
			t.Fatalf("silo %d belief must be initialized to %d bins, got %d", silo.ID, blackBoxBins, len(bb.Belief))
		}
		if len(bb.Log) != 1 {
			t.Fatalf("silo %d log entries = %d, want 1", silo.ID, len(bb.Log))
		}
		entry := bb.Log[0]
		if entry.Year != 122 || entry.Month != 1 {
			t.Fatalf("silo %d log stamp = %d/%d, want 122/1", silo.ID, entry.Year, entry.Month)
		}
		if entry.Reason != "exploit" {
			t.Fatalf("silo %d reason = %q, want exploit", silo.ID, entry.Reason)
		}
		if entry.Factors == "" {
			t.Fatalf("silo %d log must carry the factor summary", silo.ID)
		}
		if !blackBoxOnMenu(entry.ChosenR) {
			t.Fatalf("silo %d chosen r = %v not on menu", silo.ID, entry.ChosenR)
		}
		if entry.ChosenR != silo.SafeguardRisk || entry.ChosenR != bb.LastAssignedRisk {
			t.Fatalf("silo %d assignment mismatch: log=%v safeguard=%v last=%v",
				silo.ID, entry.ChosenR, silo.SafeguardRisk, bb.LastAssignedRisk)
		}
		if entry.Lo > entry.ThetaHat || entry.ThetaHat > entry.Hi {
			t.Fatalf("silo %d summary inconsistent: lo=%.3f θ̂=%.3f hi=%.3f",
				silo.ID, entry.Lo, entry.ThetaHat, entry.Hi)
		}
		if entry.PDanger < 0 || entry.PDanger > 1 {
			t.Fatalf("silo %d P_danger = %v outside [0,1]", silo.ID, entry.PDanger)
		}
		if math.IsInf(entry.Utility, 0) || math.IsNaN(entry.Utility) {
			t.Fatalf("silo %d utility must be finite, got %v", silo.ID, entry.Utility)
		}
	}
	if len(target.BlackBox.PendingObservations) != 1 || target.BlackBox.PendingObservations[0].ArriveMonth != 2 {
		t.Fatalf("future observation must stay pending: %+v", target.BlackBox.PendingObservations)
	}
}

// TestBlackBoxLogRingCap 日志环形截断：超过 24 条丢弃最旧。
func TestBlackBoxLogRingCap(t *testing.T) {
	svc := newBlackBoxGame(t, 302)
	silo := svc.siloByID(2)
	if silo == nil {
		t.Fatal("three-silo game must contain silo 2")
	}
	const rounds = blackBoxLogCap + 6
	for m := 1; m <= rounds; m++ {
		svc.runSilo1BlackBoxSiloMonth(silo, 122, m)
	}
	bb := &silo.BlackBox
	if len(bb.Log) != blackBoxLogCap {
		t.Fatalf("log length = %d, want ring cap %d", len(bb.Log), blackBoxLogCap)
	}
	if bb.Log[0].Month != rounds-blackBoxLogCap+1 {
		t.Fatalf("oldest retained entry month = %d, want %d", bb.Log[0].Month, rounds-blackBoxLogCap+1)
	}
	if bb.Log[len(bb.Log)-1].Month != rounds {
		t.Fatalf("newest entry month = %d, want %d", bb.Log[len(bb.Log)-1].Month, rounds)
	}
}

// TestBlackBoxJammedDropsInFlightObservations 专线干扰：早退、丢弃在途观测、
// 不更新信念、不签发新指派。
func TestBlackBoxJammedDropsInFlightObservations(t *testing.T) {
	svc := newBlackBoxGame(t, 303)
	for _, silo := range svc.silos {
		silo.BlackBox.PendingObservations = []model.BlackBoxPendingObservation{
			blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.9, 1),
		}
	}
	svc.silos[0].SafeguardRisk = 0.33 // 非菜单值：若回合误跑必被覆写
	svc.world.Flags[model.WorldFlagNetworkJammed] = true

	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1BlackBoxRound(&logs, &stories)

	if len(stories) != 0 || len(logs) != 0 {
		t.Fatalf("jammed round must stay silent: logs=%v stories=%v", logs, stories)
	}
	for _, silo := range svc.silos {
		bb := &silo.BlackBox
		if len(bb.PendingObservations) != 0 {
			t.Fatalf("silo %d in-flight observations must be dropped: %+v", silo.ID, bb.PendingObservations)
		}
		if bb.Belief != nil {
			t.Fatalf("silo %d belief must not be initialized while jammed", silo.ID)
		}
		if len(bb.Log) != 0 || bb.LastAssignedRisk != 0 {
			t.Fatalf("silo %d must not receive a new assignment while jammed: %+v", silo.ID, bb)
		}
	}
	if svc.silos[0].SafeguardRisk != 0.33 {
		t.Fatalf("jammed round must not touch SafeguardRisk, got %v", svc.silos[0].SafeguardRisk)
	}
}

// TestBlackBoxGatedOffZeroChangeZeroRNG 门控关闭 (功能关 / 1 号覆灭)：
// 零变化、零 RNG。
func TestBlackBoxGatedOffZeroChangeZeroRNG(t *testing.T) {
	check := func(name string, svc *GameService) {
		t.Helper()
		for _, silo := range svc.silos {
			silo.BlackBox.PendingObservations = []model.BlackBoxPendingObservation{
				blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.9, 1),
			}
		}
		const seed = 4242
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		logs, stories := []string{}, []model.StoryEvent{}
		svc.runSilo1BlackBoxRound(&logs, &stories)
		if len(stories) != 0 || len(logs) != 0 {
			t.Fatalf("%s: gated-off round must stay silent", name)
		}
		for _, silo := range svc.silos {
			bb := &silo.BlackBox
			if bb.Belief != nil || len(bb.Log) != 0 || bb.LastAssignedRisk != 0 {
				t.Fatalf("%s: gated-off round must not write state: %+v", name, bb)
			}
			if len(bb.PendingObservations) != 1 {
				t.Fatalf("%s: gated-off round must not touch pending observations", name)
			}
		}
		ref := rand.New(rand.NewSource(seed))
		for i := 0; i < 64; i++ {
			if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
				t.Fatalf("%s: gated-off round consumed RNG (draw %d: %v != %v)", name, i, got, want)
			}
		}
	}

	off := newBlackBoxGame(t, 304)
	off.world.Silo1BlackBoxEnabled = false
	check("feature-off", off)

	dead := newBlackBoxGame(t, 305)
	dead.world.Silo1.Alive = false
	check("silo1-dead", dead)
}

// TestBlackBoxRoundConsumesNoRNG 功能开启、回合真实工作 (抽干观测、更新信念、
// 指派、落日志) 也零 RNG：回合前后引擎 RNG 流状态不变 (固定种子抽序列对比)。
func TestBlackBoxRoundConsumesNoRNG(t *testing.T) {
	svc := newBlackBoxGame(t, 306)
	for _, silo := range svc.silos {
		silo.BlackBox.PendingObservations = []model.BlackBoxPendingObservation{
			blackBoxObs(model.BlackBoxSignalReceiptDelay, 0.7, 1),
			blackBoxObs(model.BlackBoxSignalLineTone, 0.4, 2),
		}
	}
	const seed = 777
	engine.SetRNG(rand.New(rand.NewSource(seed)))

	logs, stories := []string{}, []model.StoryEvent{}
	svc.runSilo1BlackBoxRound(&logs, &stories)

	for _, silo := range svc.silos {
		if silo.BlackBox.Belief == nil || len(silo.BlackBox.Log) != 1 {
			t.Fatalf("round must actually do its work for silo %d", silo.ID)
		}
	}
	ref := rand.New(rand.NewSource(seed))
	for i := 0; i < 64; i++ {
		if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
			t.Fatalf("black-box round consumed RNG (draw %d: %v != %v)", i, got, want)
		}
	}
}

// fixedBlackBoxProbe 测试用探索提议者：恒定提议 r。
type fixedBlackBoxProbe struct{ r float64 }

func (p fixedBlackBoxProbe) ProbeProposer(*model.Silo, []float64, BlackBoxProbeContext) (float64, bool) {
	return p.r, true
}

// TestBlackBoxProbeInterfaceAndCooldown 探索接口与冷却位置就位：冷却 >0 时
// 逐月递减且不咨询提议者；归零后提议覆盖 argmax (00 裁决 10)，使用后设冷却。
func TestBlackBoxProbeInterfaceAndCooldown(t *testing.T) {
	svc := newBlackBoxGame(t, 307)
	silo := svc.siloByID(2)
	if silo == nil {
		t.Fatal("three-silo game must contain silo 2")
	}
	original := silo1BlackBoxProber
	t.Cleanup(func() { silo1BlackBoxProber = original })
	silo1BlackBoxProber = fixedBlackBoxProbe{r: 0.90}

	// 冷却 2 → 1：递减，提议者不被咨询。
	silo.BlackBox.ProbeCooldown = 2
	svc.runSilo1BlackBoxSiloMonth(silo, 122, 1)
	if silo.BlackBox.ProbeCooldown != 1 {
		t.Fatalf("cooldown must decrement: %d", silo.BlackBox.ProbeCooldown)
	}
	if silo.BlackBox.Log[0].Reason != "exploit" || silo.BlackBox.LastAssignedRisk == 0.90 {
		t.Fatalf("cooldown>0 must skip the proposer: %+v", silo.BlackBox.Log[0])
	}

	// 冷却 1 → 0：仍走利用 (递减月不咨询提议者)。
	svc.runSilo1BlackBoxSiloMonth(silo, 122, 2)
	if silo.BlackBox.ProbeCooldown != 0 || silo.BlackBox.Log[1].Reason != "exploit" {
		t.Fatalf("decrement month must stay exploit: %+v", silo.BlackBox.Log[1])
	}

	// 冷却归零：提议覆盖 argmax，理由码 explore，使用后设冷却。
	svc.runSilo1BlackBoxSiloMonth(silo, 122, 3)
	if silo.BlackBox.LastAssignedRisk != 0.90 || silo.SafeguardRisk != 0.90 {
		t.Fatalf("probe must override argmax: last=%v safeguard=%v",
			silo.BlackBox.LastAssignedRisk, silo.SafeguardRisk)
	}
	if silo.BlackBox.Log[2].Reason != "explore" {
		t.Fatalf("probe month reason = %q, want explore", silo.BlackBox.Log[2].Reason)
	}
	if silo.BlackBox.ProbeCooldown != blackBoxProbeCooldownStart {
		t.Fatalf("cooldown must be set after probe use: %d", silo.BlackBox.ProbeCooldown)
	}
}

// TestBlackBoxRoundRunsThroughWorldPipeline 真实多堡管线驱动：黑箱回合随月度
// tick 挂载运行，日志逐月累积。
func TestBlackBoxRoundRunsThroughWorldPipeline(t *testing.T) {
	svc := newBlackBoxGame(t, 308)
	advanceMonthForTest(t, svc)
	for _, silo := range svc.silos {
		if len(silo.BlackBox.Log) != 1 {
			t.Fatalf("silo %d must have one black-box entry after month 1, got %d", silo.ID, len(silo.BlackBox.Log))
		}
		if !blackBoxOnMenu(silo.BlackBox.LastAssignedRisk) {
			t.Fatalf("silo %d assigned r = %v not on menu", silo.ID, silo.BlackBox.LastAssignedRisk)
		}
	}
	advanceMonthForTest(t, svc)
	for _, silo := range svc.silos {
		if len(silo.BlackBox.Log) != 2 {
			t.Fatalf("silo %d must have two black-box entries after month 2, got %d", silo.ID, len(silo.BlackBox.Log))
		}
		if e := silo.BlackBox.Log[1]; e.Year != 122 || e.Month != 2 {
			t.Fatalf("silo %d month-2 entry stamp = %d/%d, want 122/2", silo.ID, e.Year, e.Month)
		}
	}
}

// blackBoxGuardViolations 扫描源码中的黑箱壁违规：物理层标识符 (真值结算、
// 观测产生、E 曲线)、Theta 真值字段读取与 engine 包导入。
func blackBoxGuardViolations(t *testing.T, filename string, src any) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	// 物理层标识符 (真值结算、观测产生、E 曲线) 一律不得出现。
	forbiddenIdents := map[string]bool{
		"UpdateBlackBoxMonthly":        true,
		"GenerateBlackBoxObservations": true,
		"ExecutionIntensity":           true,
		"CurrentExecution":             true,
	}
	// 字段选择器：Theta 真值与 E 生效槽不得被读取。
	forbiddenSelectors := map[string]bool{
		"Theta":              true,
		"CurrentExecution":   true,
		"ExecutionIntensity": true,
	}
	var violations []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ImportSpec:
			if strings.Contains(node.Path.Value, "internal/engine") {
				violations = append(violations, "imports engine (physics) package "+node.Path.Value)
			}
		case *ast.SelectorExpr:
			if forbiddenSelectors[node.Sel.Name] {
				violations = append(violations, "reads physics identifier ."+node.Sel.Name+
					" at "+fset.Position(node.Pos()).String())
			}
		case *ast.Ident:
			if forbiddenIdents[node.Name] {
				violations = append(violations, "references physics identifier "+node.Name+
					" at "+fset.Position(node.Pos()).String())
			}
		}
		return true
	})
	return violations
}

// TestBlackBoxMindReadingGuard AST 守卫：黑箱回合文件禁止引用物理层真值与
// 曲线标识符——决策输入只有 Belief、PendingObservations 与 1 号内部参数。
func TestBlackBoxMindReadingGuard(t *testing.T) {
	for _, v := range blackBoxGuardViolations(t, "silo1_blackbox.go", nil) {
		t.Errorf("black-box round violates the black-box wall: %s", v)
	}

	// 守卫自检：同一扫描器对违规片段必须报警 (防止守卫空转)。
	bad := `package service

import "silo40/internal/engine"

func bad(s *engine.Something) float64 {
	s.BlackBox.Theta = 1
	s.BlackBox.CurrentExecution = 1
	engine.UpdateBlackBoxMonthly(nil)
	return s.ExecutionIntensity
}
`
	violations := blackBoxGuardViolations(t, "bad_example.go", bad)
	if len(violations) < 5 {
		t.Fatalf("guard self-check must flag the bad example (engine import, .Theta, "+
			".CurrentExecution, UpdateBlackBoxMonthly, ExecutionIntensity), got %v", violations)
	}
}

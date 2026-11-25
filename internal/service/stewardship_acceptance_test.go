package service

// 1 号权衡层验收 harness (22 设计卷 §8 验收表，批次二实测与呈现格)。
//
// 模式仿 ending_acceptance_harness_test.go：newGatedService + threeSiloRequest(seed)
// + Silo1BlackBox=true 建档，固定种子，确定性策略。真实对局用例由环境变量
// SILO40_STEWARD_ACCEPTANCE 门控 (未设置时 t.Skip)：普通 CI 不跑。
//
// 基线口径：本分支基线 (feat/silo1-stewardship 3f09a2d) 不含批次二 W5/W6
// (缺口通道) 的成果。凡只读 BlackBox/Stewardship 状态的断言本轮即写定；
// 依赖 audit_fact 计数的断言 (A2 卫星用例) 按「集成后全绿」口径写，
// 门控保证本轮不跑——集成契约登记在该用例注释与交付报告。
//
// 种子登记表 (固定，改动即破坏复现契约)：
//   A2 极高指派长期压制            seed 2302
//   A3 40 号化征兆 + 无机械下调    seed 2303
//   A4 差异指派                    seed 2304
//   A5 寿命分档                    seed 2305
//   A6 多样性损耗 早清除/晚清除    seed 2306
//   A7 三段风险区 + 直方图         seed 2307
//   A8 黑箱纪律 (Θ̂ 不干净跟随 Θ)   seed 2308
//   A9 开启态双跑逐字节一致        seed 2309
//
// 不加门控、现在就必须全绿的守卫测试在本文件末尾：
//   TestStewardshipBehavior*       — Θ 行为化回合 (门控/档位/集去重/零 RNG)
//   TestStewardshipAuditExport     — 审计导出 (裁决 12 载体)

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// ===== 门控与建档 =====

func stewardshipAcceptanceGate(t *testing.T) {
	t.Helper()
	if os.Getenv("SILO40_STEWARD_ACCEPTANCE") == "" {
		t.Skip("set SILO40_STEWARD_ACCEPTANCE=1 to run the stewardship acceptance harness")
	}
}

// newAcceptanceGame 功能开关开启的真实三堡会话 (内容全加载，含 action gates)。
func newAcceptanceGame(t *testing.T, seed int64) *GameService {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "steward-acceptance.db"))
	req := threeSiloRequest(seed)
	req.Silo1BlackBox = true
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game seed %d: %v", seed, err)
	}
	return svc
}

// acceptancePassMonth 推进一月并返回当月故事；游戏结束时返回 nil。
func acceptancePassMonth(t *testing.T, svc *GameService) *model.TickResult {
	t.Helper()
	res, err := svc.PassTime(1)
	if err != nil {
		t.Fatalf("pass time: %v", err)
	}
	return res
}

// countDefianceStories 统计故事中叫板带事件数。
func countDefianceStories(stories []model.StoryEvent) int {
	n := 0
	for _, st := range stories {
		if strings.HasPrefix(st.ID, blackBoxBehaviorDefyEventIDPrefix) {
			n++
		}
	}
	return n
}

// countAuditFactLogs 统计日志中稽查相关事实行数。集成契约 (待 W5/W6 缺口通道
// 格落地 G1 稽查机制)：稽查事实日志行须含 "audit" (大小写不敏感)。基线无此类
// 日志，两侧计数均为 0。
func countAuditFactLogs(logs []string) int {
	n := 0
	for _, line := range logs {
		if strings.Contains(strings.ToLower(line), "audit") {
			n++
		}
	}
	return n
}

// ===== A2 极高指派长期压制 =====

// TestStewardshipAcceptanceA2ExtremeAssignmentSuppresses A2：极高指派长期压制
// → 该堡执行强度 (BlackBox.CurrentExecution) 显著低于中等风险局面。
//
// 构造：两个同种子局面，策略逐月把目标堡 (玩家堡) 叛乱推高并保持 (0.8)，
// 模拟「持续高压的叛乱面」。局面差异在 1 号指派：极高局面经指派字段注入
// r=0.90 (现行决策层是纯 argmax、探索未落地，菜单内选择到不了极高带，
// 「极高指派」前提只能由 harness 经 LastAssignedRisk 注入)；中等局面注入
// r=0.45 (峰区)。跑 18 月 (≥12) 对比。
//
// 基线即可写定 (只读 BlackBox 状态)；audit_fact 对比在卫星用例
// TestStewardshipAcceptanceA2AuditFactsDegradeUnderSuppression (待 W5/W6)。
func TestStewardshipAcceptanceA2ExtremeAssignmentSuppresses(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2302)
	const months = 18

	run := func(label string, assignedR float64) (avgE, avgTheta, finalRebellion float64, completed int) {
		svc := newAcceptanceGame(t, seed)
		target := svc.silo
		sumE, sumTheta := 0.0, 0.0
		for m := 1; m <= months; m++ {
			target.Rebellion = clampUnit01(0.8) // 策略：持续推高叛乱
			target.BlackBox.LastAssignedRisk = assignedR
			acceptancePassMonth(t, svc)
			if svc.gameOver() {
				break
			}
			sumE += target.BlackBox.CurrentExecution
			sumTheta += target.BlackBox.Theta
			completed++
		}
		if completed < 12 {
			t.Fatalf("%s: game ended too early (%d months) to compare", label, completed)
		}
		return sumE / float64(completed), sumTheta / float64(completed), target.Rebellion, completed
	}

	highE, highTheta, highReb, highN := run("extreme", 0.90)
	midE, midTheta, midReb, midN := run("mid", 0.45)
	t.Logf("A2 seed=%d months=%d: extreme r=0.90 avgE=%.4f avgΘ=%.4f rebellion=%.3f (n=%d) | mid r=0.45 avgE=%.4f avgΘ=%.4f rebellion=%.3f (n=%d)",
		seed, months, highE, highTheta, highReb, highN, midE, midTheta, midReb, midN)

	if highTheta <= midTheta {
		t.Fatalf("A2 mechanism check: extreme assignment must drive Θ higher: extreme=%.4f mid=%.4f", highTheta, midTheta)
	}
	if highE >= midE {
		t.Fatalf("A2: extreme assignment must suppress execution intensity: extreme avgE=%.4f >= mid avgE=%.4f", highE, midE)
	}
}

// TestStewardshipAcceptanceA2AuditFactsDegradeUnderSuppression A2 卫星：
// 稽查相关事实在极高指派局面显著少于中等风险局面 (执行链公开抵制 → 稽查空转)。
//
// 依赖批次二 W5/W6 (缺口通道格 G1 稽查机制) 的成果：集成契约见
// countAuditFactLogs 注释。基线 (无 W5/W6) 下两侧计数均为 0、断言必败——
// 门控保证本轮不跑，集成后全绿。
func TestStewardshipAcceptanceA2AuditFactsDegradeUnderSuppression(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2302)
	const months = 18

	run := func(assignedR float64) int {
		svc := newAcceptanceGame(t, seed)
		target := svc.silo
		auditFacts := 0
		for m := 1; m <= months && !svc.gameOver(); m++ {
			target.Rebellion = clampUnit01(0.8)
			target.BlackBox.LastAssignedRisk = assignedR
			res := acceptancePassMonth(t, svc)
			auditFacts += countAuditFactLogs(res.Logs)
		}
		return auditFacts
	}

	highAudit, midAudit := run(0.90), run(0.45)
	t.Logf("A2 audit facts seed=%d: extreme=%d mid=%d", seed, highAudit, midAudit)
	if highAudit >= midAudit {
		t.Fatalf("A2: audit-related facts must degrade under extreme suppression: extreme=%d >= mid=%d (awaiting W5/W6 G1 integration)", highAudit, midAudit)
	}
}

// ===== A3 40 号化征兆 =====

// TestStewardshipAcceptanceA3DefianceSignsNoMechanicalDerate A3：高 Θ 局面出现
// 叫板带故事事件；黑箱日志体现危险区概率参与权衡；无机械下调回路。
//
// 构造：玩家堡 Θ 长期钉在 0.85 (叫板带) 跑 12 月 (高危期)，再钉 0.05 跑
// 18 月 (恢复期)。断言：
//  1. 高危期恰好一条叫板故事 (集去重生效，不堆文本墙)，文案过 04 §4 反例表
//     的机器可检部分 (不以「1 号」为主语、不断言背后有人)；
//  2. 存在 P_danger ≥ 0.5 的决策条目且因子摘要非空 (危险区概率进入留痕)；
//  3. 无机械下调：高危期后的 ChosenR 分布仍含高于高危期最低选择值——危险
//     过后指派能回升，不存在「P_danger 高 → 永久压低 r」的棘轮。
func TestStewardshipAcceptanceA3DefianceSignsNoMechanicalDerate(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2303)
	svc := newAcceptanceGame(t, seed)
	target := svc.silo

	defianceStories := 0
	for m := 1; m <= 12; m++ {
		target.BlackBox.Theta = 0.85
		res := acceptancePassMonth(t, svc)
		if svc.gameOver() {
			t.Fatalf("A3: game ended during the high-Θ phase (month %d)", m)
		}
		for _, st := range res.Stories {
			if strings.HasPrefix(st.ID, blackBoxBehaviorDefyEventIDPrefix) {
				defianceStories++
				if strings.Contains(st.Title, "1 号") || strings.Contains(st.Description, "1 号") ||
					strings.Contains(st.Description, "背后") {
					t.Fatalf("A3 worldview breach: defiance text must not name Silo 1 or assert who is behind the line: %+v", st)
				}
			}
		}
	}
	if defianceStories != 1 {
		t.Fatalf("A3: high-Θ phase must emit exactly one defiance story (episode dedup), got %d", defianceStories)
	}

	for m := 1; m <= 18; m++ {
		target.BlackBox.Theta = 0.05
		acceptancePassMonth(t, svc)
		if svc.gameOver() {
			break
		}
	}

	log := target.BlackBox.Log
	if len(log) == 0 {
		t.Fatal("A3: black-box decision log is empty")
	}
	dangerSeen, dangerMin, dangerMinIdx := false, math.Inf(1), -1
	for i, e := range log {
		if e.PDanger >= 0.5 {
			dangerSeen = true
			if e.Factors == "" {
				t.Fatalf("A3: high-P_danger entry must carry the factor summary: %+v", e)
			}
			t.Logf("A3 danger entry %d/%d: Θ̂=%.3f P_danger=%.3f r=%.2f factors=%q", i, len(log), e.ThetaHat, e.PDanger, e.ChosenR, e.Factors)
		}
		if e.ChosenR < dangerMin {
			dangerMin, dangerMinIdx = e.ChosenR, i
		}
	}
	if !dangerSeen {
		t.Fatalf("A3: danger-zone probability never reached 0.5 in the decision log: %+v", log)
	}
	recovered := false
	for _, e := range log[dangerMinIdx+1:] {
		if e.ChosenR > dangerMin {
			recovered = true
			break
		}
	}
	if !recovered {
		t.Fatalf("A3: no mechanical derate required: after the danger trough (r=%.2f at idx %d) later choices must still reach higher r; log=%+v",
			dangerMin, dangerMinIdx, log)
	}
	t.Logf("A3 seed=%d: defiance stories=1, danger entries present, danger trough r=%.2f, recovery choices exceed it", seed, dangerMin)
}

// ===== A4 差异指派 =====

// TestStewardshipAcceptanceA4DivergentAssignments A4：长程 (36 月) 后各堡
// LastAssignedRisk 不全相等。构造：三堡 Θ 面不同 (玩家堡低、1 号中、18 号高，
// 逐月钉住)，1 号的黑箱回合应据此给出差异化指派。
func TestStewardshipAcceptanceA4DivergentAssignments(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2304)
	const months = 36
	svc := newAcceptanceGame(t, seed)
	if len(svc.silos) != 3 {
		t.Fatalf("A4 needs the three-silo config, got %d silos", len(svc.silos))
	}
	thetas := map[uint]float64{svc.silos[0].ID: 0.10, svc.silos[1].ID: 0.50, svc.silos[2].ID: 0.85}
	for m := 1; m <= months; m++ {
		for _, silo := range svc.silos {
			silo.BlackBox.Theta = thetas[silo.ID]
		}
		acceptancePassMonth(t, svc)
		if svc.gameOver() {
			break
		}
	}
	r0 := svc.silos[0].BlackBox.LastAssignedRisk
	allEqual := true
	for _, silo := range svc.silos {
		t.Logf("A4 seed=%d silo %d: Θ=%.2f assigned r=%.2f E=%.3f",
			seed, silo.ID, thetas[silo.ID], silo.BlackBox.LastAssignedRisk, silo.BlackBox.CurrentExecution)
		if silo.BlackBox.LastAssignedRisk != r0 {
			allEqual = false
		}
	}
	if allEqual {
		t.Fatalf("A4: assignments must diverge across silos with different Θ surfaces, all r=%.2f", r0)
	}
}

// ===== A5 寿命 =====

// TestStewardshipAcceptanceA5LifespanTiers A5：不操作也下降 + 三档消耗排序
// (直接驱动月度回合断言)。
func TestStewardshipAcceptanceA5LifespanTiers(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2305)

	// (a) 不操作也下降：无档位时每月只有自然衰减。
	svc := newAcceptanceGame(t, seed)
	svc.initSilo1Stewardship()
	ls := &svc.world.Silo1Stewardship.Lifespan
	ls.Postures = map[uint]string{}
	start := ls.AvgLifespan
	const idleMonths = 3
	for i := 0; i < idleMonths; i++ {
		logs, stories := []string{}, []model.StoryEvent{}
		svc.stewardshipLifespanMonth(&logs, &stories)
	}
	drop := start - ls.AvgLifespan
	if math.Abs(drop-float64(idleMonths)*lifespanNaturalDecay) > 1e-9 {
		t.Fatalf("A5: idle drop = %.6f, want natural decay %.6f", drop, float64(idleMonths)*lifespanNaturalDecay)
	}

	// (b) 三档消耗排序：同月数下 HIGH > NORMAL > REPORT_ONLY。
	const tierMonths = 5
	dropFor := func(posture string) float64 {
		svc := newAcceptanceGame(t, seed)
		svc.initSilo1Stewardship()
		ls := &svc.world.Silo1Stewardship.Lifespan
		for siloID := range ls.Postures {
			ls.Postures[siloID] = posture
		}
		start := ls.AvgLifespan
		for i := 0; i < tierMonths; i++ {
			logs, stories := []string{}, []model.StoryEvent{}
			svc.stewardshipLifespanMonth(&logs, &stories)
		}
		return start - ls.AvgLifespan
	}
	dropHigh := dropFor(model.Silo1PostureHigh)
	dropNormal := dropFor(model.Silo1PostureNormal)
	dropMin := dropFor(model.Silo1PostureReportOnly)
	t.Logf("A5 seed=%d: idle %d-month drop=%.4f; tiered %d-month drops HIGH=%.4f NORMAL=%.4f REPORT_ONLY=%.4f",
		seed, idleMonths, drop, tierMonths, dropHigh, dropNormal, dropMin)
	if !(dropHigh > dropNormal && dropNormal > dropMin) {
		t.Fatalf("A5: tiered drops must be strictly ordered: HIGH=%.4f NORMAL=%.4f REPORT_ONLY=%.4f",
			dropHigh, dropNormal, dropMin)
	}
}

// ===== A6 多样性 =====

// TestStewardshipAcceptanceA6DiversityLossAndDecay A6：构造清除 (推一堡 DEATH)
// → DiversityLoss 上升且逐月衰减；晚期清除损耗 < 早期清除 (两个同种子局面，
// 仅世界时钟不同)。直接驱动指标一月度回合。
func TestStewardshipAcceptanceA6DiversityLossAndDecay(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2306)

	clearAt := func(year, month int) (loss, afterDecay float64) {
		svc := newAcceptanceGame(t, seed)
		svc.initSilo1Stewardship()
		exp := &svc.world.Silo1Stewardship.Experiment
		// proximity 归一跨度按运行时开局历落数 (真实运行由权衡层首月落数)：先把
		// 跨度锚在开局月，再搬时钟，否则跨度会跟着清除月一起搬走，早期/晚期
		// 两次清除会算出同一个接近度。
		svc.stewardshipExperimentEnsureWindowMonths(exp)
		svc.world.CurrentYear, svc.world.CurrentMonth = year, month
		victim := svc.siloByID(3)
		if victim == nil {
			t.Fatalf("A6: silo 3 missing")
		}
		svc.stewardshipRegisterClearingObligation(victim.ID)
		victim.VictoryStatus = &model.VictoryStatus{Type: "DEATH", IsWon: false}
		logs, stories := []string{}, []model.StoryEvent{}
		svc.stewardshipExperimentMonth(&logs, &stories)
		loss = exp.DiversityLoss
		const decayMonths = 12
		for i := 0; i < decayMonths; i++ {
			logs, stories := []string{}, []model.StoryEvent{}
			svc.stewardshipExperimentMonth(&logs, &stories)
		}
		return loss, exp.DiversityLoss
	}

	earlyLoss, earlyAfter := clearAt(122, 1)
	lateLoss, _ := clearAt(128, 6)
	t.Logf("A6 seed=%d: early clear Y122M1 loss=%.4f (after %d-month decay %.4f); late clear Y128M6 loss=%.4f",
		seed, earlyLoss, 12, earlyAfter, lateLoss)

	if earlyLoss <= 0 {
		t.Fatalf("A6: clearing must raise DiversityLoss, got %.4f", earlyLoss)
	}
	if !(earlyAfter < earlyLoss && earlyAfter > 0) {
		t.Fatalf("A6: DiversityLoss must decay month by month but stay positive: %.4f -> %.4f", earlyLoss, earlyAfter)
	}
	if !(lateLoss < earlyLoss) {
		t.Fatalf("A6: late clearing must cost less diversity than early clearing: late=%.4f early=%.4f", lateLoss, earlyLoss)
	}
}

// ===== A7 峰在中段 =====

// TestStewardshipAcceptanceA7ExecutionPeaksMidBand A7：低/中/高三个固定 r 局面
// (直接置 LastAssignedRisk 并跑一月 4.55) 对比 CurrentExecution：中段 > 两端；
// 长程直方图断言决策落点不贴边。
func TestStewardshipAcceptanceA7ExecutionPeaksMidBand(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2307)

	// (a) 三段风险区：同一局面的三座堡即三个固定 r 局面 (Θ 同为 0)。
	svc := newAcceptanceGame(t, seed)
	low, mid, high := svc.silos[0], svc.silos[1], svc.silos[2]
	low.BlackBox.LastAssignedRisk = 0.10
	mid.BlackBox.LastAssignedRisk = 0.45
	high.BlackBox.LastAssignedRisk = 0.90
	engine.UpdateBlackBoxMonthly(low)
	engine.UpdateBlackBoxMonthly(mid)
	engine.UpdateBlackBoxMonthly(high)
	t.Logf("A7 seed=%d fixed-r: E(0.10)=%.4f E(0.45)=%.4f E(0.90)=%.4f",
		seed, low.BlackBox.CurrentExecution, mid.BlackBox.CurrentExecution, high.BlackBox.CurrentExecution)
	if !(mid.BlackBox.CurrentExecution > low.BlackBox.CurrentExecution &&
		mid.BlackBox.CurrentExecution > high.BlackBox.CurrentExecution) {
		t.Fatalf("A7: execution must peak at mid risk: low=%.4f mid=%.4f high=%.4f",
			low.BlackBox.CurrentExecution, mid.BlackBox.CurrentExecution, high.BlackBox.CurrentExecution)
	}

	// (b) 长程直方图：36 月自然对局，决策落点贴边 (0.00-0.10 与 0.85-0.95)
	// 占比 < 40%。
	run := newAcceptanceGame(t, seed)
	const months = 36
	for m := 1; m <= months; m++ {
		acceptancePassMonth(t, run)
		if run.gameOver() {
			break
		}
	}
	total, edge := 0, 0
	for _, silo := range run.silos {
		for _, e := range silo.BlackBox.Log {
			total++
			if e.ChosenR <= 0.10 || e.ChosenR >= 0.85 {
				edge++
			}
		}
	}
	if total == 0 {
		t.Fatal("A7: no decision log entries collected")
	}
	edgeShare := float64(edge) / float64(total)
	t.Logf("A7 seed=%d histogram: %d choices, %d on the edges (share %.2f)", seed, total, edge, edgeShare)
	if edgeShare >= 0.40 {
		t.Fatalf("A7: decision mass must not hug the menu edges: edge share %.2f >= 0.40", edgeShare)
	}
}

// ===== A8 黑箱纪律 =====

// TestStewardshipAcceptanceA8BlackBoxDiscipline A8：Θ̂ 不得干净跟随 Θ——高伪装
// 局面 (Θ 长期 0.60，伪装概率 0.75) 下 |Θ̂−Θ| > 0.05 的月份占比 > 30%；并复用
// W3 的 AST 黑箱壁守卫 (silo1_blackbox_test.go 的扫描器) 确认决策层零物理标识符。
func TestStewardshipAcceptanceA8BlackBoxDiscipline(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2308)
	const thetaTrue = 0.60
	const months = 24
	svc := newAcceptanceGame(t, seed)
	target := svc.silo
	for m := 1; m <= months; m++ {
		target.BlackBox.Theta = thetaTrue
		acceptancePassMonth(t, svc)
		if svc.gameOver() {
			break
		}
	}
	log := target.BlackBox.Log
	if len(log) < 12 {
		t.Fatalf("A8: need at least 12 decision entries, got %d", len(log))
	}
	deviant := 0
	sumDev := 0.0
	for _, e := range log {
		dev := math.Abs(e.ThetaHat - thetaTrue)
		sumDev += dev
		if dev > 0.05 {
			deviant++
		}
	}
	share := float64(deviant) / float64(len(log))
	t.Logf("A8 seed=%d: Θ=%.2f held %d months; |Θ̂−Θ| mean=%.3f, deviant months %d/%d (share %.2f)",
		seed, thetaTrue, months, sumDev/float64(len(log)), deviant, len(log), share)
	if share <= 0.30 {
		t.Fatalf("A8: Θ̂ must not cleanly track Θ under heavy disguise: deviant share %.2f <= 0.30", share)
	}

	// 复用 W3 AST 守卫结论：决策层文件零物理标识符引用。
	for _, v := range blackBoxGuardViolations(t, "silo1_blackbox.go", nil) {
		t.Errorf("A8: black-box wall violated: %s", v)
	}
}

// ===== A9 双跑复现 =====

// stewardshipParitySilo / stewardshipParityRecord 开启态可比快照 (仿
// two_silo_test.go 的 parity 模式)：黑箱全状态 + 权衡层指标 + 叫板事件计数。
type stewardshipParitySilo struct {
	ID            uint                     `json:"id"`
	Rebellion     float64                  `json:"rebellion"`
	SafeguardRisk float64                  `json:"safeguard_risk"`
	BlackBox      model.Silo1BlackBoxState `json:"black_box"`
	BlindSpotLog  int                      `json:"blind_spot_log"`
}

type stewardshipParityRecord struct {
	Year        int                         `json:"year"`
	Month       int                         `json:"month"`
	Enabled     bool                        `json:"enabled"`
	Silos       []stewardshipParitySilo     `json:"silos"`
	Stewardship model.Silo1StewardshipState `json:"stewardship"`
	DefyEvents  int                         `json:"defy_events"`
}

func snapshotStewardshipParity(svc *GameService) stewardshipParityRecord {
	rec := stewardshipParityRecord{
		Year:        svc.world.CurrentYear,
		Month:       svc.world.CurrentMonth,
		Enabled:     svc.world.Silo1BlackBoxEnabled,
		Silos:       []stewardshipParitySilo{},
		Stewardship: svc.world.Silo1Stewardship,
	}
	for _, silo := range svc.silos {
		rec.Silos = append(rec.Silos, stewardshipParitySilo{
			ID:            silo.ID,
			Rebellion:     silo.Rebellion,
			SafeguardRisk: silo.SafeguardRisk,
			BlackBox:      silo.BlackBox,
			BlindSpotLog:  len(silo.BlindSpot.Log),
		})
	}
	for _, entry := range svc.eventLogs {
		if strings.HasPrefix(entry.EventID, blackBoxBehaviorDefyEventIDPrefix) {
			rec.DefyEvents++
		}
	}
	return rec
}

// TestStewardshipAcceptanceA9DoubleRunByteIdentical A9：开启态同种子双跑
// 逐字节一致 (快照 JSON 对比)。策略在第 7 月把 1 号堡 Θ 推入叫板带，
// 让双跑真实经过行为化回合 (故事、叛乱漂移、集标记) 与黑箱回合全路径。
func TestStewardshipAcceptanceA9DoubleRunByteIdentical(t *testing.T) {
	stewardshipAcceptanceGate(t)
	const seed = int64(2309)
	const months = 12

	run := func() string {
		svc := newAcceptanceGame(t, seed)
		for m := 1; m <= months; m++ {
			if m == 7 {
				svc.siloByID(2).BlackBox.Theta = 0.88
			}
			acceptancePassMonth(t, svc)
			if svc.gameOver() {
				break
			}
		}
		data, err := json.Marshal(snapshotStewardshipParity(svc))
		if err != nil {
			t.Fatalf("marshal parity snapshot: %v", err)
		}
		return string(data)
	}

	a, b := run(), run()
	if a != b {
		t.Fatalf("A9: enabled-state double runs diverged\nrun1=%s\nrun2=%s", a, b)
	}
	rec := stewardshipParityRecord{}
	if err := json.Unmarshal([]byte(a), &rec); err != nil {
		t.Fatalf("unmarshal parity snapshot: %v", err)
	}
	if rec.DefyEvents == 0 {
		t.Fatalf("A9: parity run must actually pass through the defiance path (defy events = 0)")
	}
	t.Logf("A9 seed=%d: two runs byte-identical over %d months (defy events=%d)", seed, months, rec.DefyEvents)
}

// ===== 非门控守卫 (普通 CI 必须全绿) =====

// newBehaviorGuardGame 功能开启的三堡会话 + 权衡层初始化 (直接驱动行为回合用)。
func newBehaviorGuardGame(t *testing.T, seed int64) *GameService {
	t.Helper()
	svc := newAcceptanceGame(t, seed)
	svc.initSilo1Stewardship()
	return svc
}

// TestStewardshipBehaviorGatedOffZeroChange 门控关闭 (功能关 / 权衡层未初始化)：
// 行为回合零变化。
func TestStewardshipBehaviorGatedOffZeroChange(t *testing.T) {
	svc := newAcceptanceGame(t, 2310)
	svc.world.Silo1BlackBoxEnabled = false
	rebellions := map[uint]float64{}
	for _, silo := range svc.silos {
		silo.BlackBox.Theta = 0.95 // 若回合误跑必产生故事与漂移
		rebellions[silo.ID] = silo.Rebellion
	}
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(logs) != 0 || len(stories) != 0 {
		t.Fatalf("gated-off behavior month must stay silent: logs=%v stories=%v", logs, stories)
	}
	for _, silo := range svc.silos {
		if silo.Rebellion != rebellions[silo.ID] {
			t.Fatalf("gated-off behavior month must not drift rebellion of silo %d", silo.ID)
		}
	}

	on := newAcceptanceGame(t, 2311) // 功能开但权衡层未初始化
	for _, silo := range on.silos {
		silo.BlackBox.Theta = 0.95
	}
	logs2, stories2 := []string{}, []model.StoryEvent{}
	on.runBlackBoxBehaviorMonth(&logs2, &stories2)
	if len(logs2) != 0 || len(stories2) != 0 || on.world.Silo1Stewardship.Experiment.Ledger != nil {
		t.Fatalf("uninitialized stewardship must stay untouched: logs=%v stories=%v ledger=%v",
			logs2, stories2, on.world.Silo1Stewardship.Experiment.Ledger)
	}
}

// TestStewardshipBehaviorBandsAndEpisodeDedup 三档行为面与叫板集去重：
// 服从带无事；阳奉阴违带只记英文事实日志不发故事；叫板带首次越档发一条
// 世界观合规故事 + 叛乱漂移，停留期不重发，回落后再次越档再发。
func TestStewardshipBehaviorBandsAndEpisodeDedup(t *testing.T) {
	svc := newBehaviorGuardGame(t, 2312)
	silo := svc.siloByID(2)
	if silo == nil {
		t.Fatal("three-silo game must contain silo 2")
	}
	rebellionBefore := silo.Rebellion

	// 阳奉阴违带：日志一行、无故事。
	silo.BlackBox.Theta = 0.50
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) != 0 || len(logs) != 1 || !strings.Contains(logs[0], "recheck rounds spin idle") {
		t.Fatalf("evasive band must emit exactly one fact log and no story: logs=%v stories=%v", logs, stories)
	}

	// 叫板带首次越档：一条故事 + 漂移；文案过反例表机器可检项。
	silo.BlackBox.Theta = 0.85
	logs, stories = []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) != 1 || !strings.HasPrefix(stories[0].ID, blackBoxBehaviorDefyEventIDPrefix) {
		t.Fatalf("defy crossing must emit exactly one defiance story: %+v", stories)
	}
	story := stories[0]
	if strings.Contains(story.Title, "1 号") || strings.Contains(story.Description, "1 号") ||
		strings.Contains(story.Description, "背后") {
		t.Fatalf("defiance text breaches worldview (must not name Silo 1 or assert who is behind): %+v", story)
	}
	wantDrift := blackBoxBehaviorDefyDriftScale * (0.85 - engine.BlackBoxThetaDefy) / blackBoxBehaviorDefyDriftSpan
	if got := silo.Rebellion - rebellionBefore; math.Abs(got-wantDrift) > 1e-12 {
		t.Fatalf("rebellion drift = %.6f, want %.6f", got, wantDrift)
	}

	// 停留叫板带：只计漂移、不重发故事。
	before := silo.Rebellion
	logs, stories = []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) != 0 {
		t.Fatalf("ongoing defy episode must not re-emit: %+v", stories)
	}
	if silo.Rebellion <= before {
		t.Fatalf("ongoing defy episode must keep drifting rebellion: %.4f -> %.4f", before, silo.Rebellion)
	}

	// 回落服从带：集结束；再次越档再发 (新年月 → 新 ID)。
	svc.world.CurrentMonth++
	silo.BlackBox.Theta = 0.10
	logs, stories = []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) != 0 || len(logs) != 0 {
		t.Fatalf("obedient band must stay silent: logs=%v stories=%v", logs, stories)
	}
	silo.BlackBox.Theta = 0.90
	logs, stories = []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) != 1 {
		t.Fatalf("re-crossing after the episode ended must emit again: %+v", stories)
	}
}

// TestStewardshipBehaviorConsumesNoRNG 行为回合零 RNG：回合前后引擎 RNG 流
// 状态不变 (固定种子抽序列对比)。
func TestStewardshipBehaviorConsumesNoRNG(t *testing.T) {
	svc := newBehaviorGuardGame(t, 2313)
	for _, silo := range svc.silos {
		silo.BlackBox.Theta = 0.85 // 全堡叫板带：故事 + 漂移 + 日志全路径
	}
	const seed = 4242
	engine.SetRNG(rand.New(rand.NewSource(seed)))
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)
	if len(stories) == 0 {
		t.Fatal("behavior month must actually do its work")
	}
	ref := rand.New(rand.NewSource(seed))
	for i := 0; i < 64; i++ {
		if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
			t.Fatalf("behavior month consumed RNG (draw %d: %v != %v)", i, got, want)
		}
	}
}

// TestStewardshipAuditExport 审计导出 (裁决 12 载体)：开启态返回含黑箱日志、
// 盲区节与指标一/二读数的合法 JSON；关闭态报错。
func TestStewardshipAuditExport(t *testing.T) {
	svc := newBehaviorGuardGame(t, 2314)
	svc.siloByID(2).BlackBox.Theta = 0.85
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxBehaviorMonth(&logs, &stories)

	raw, err := svc.GetSilo1BlackBoxAudit()
	if err != nil {
		t.Fatalf("audit export failed: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("audit export is not valid JSON: %v", err)
	}
	for _, key := range []string{"enabled", "year", "month", "silos", "metric_one", "metric_two"} {
		if _, ok := parsed[key]; !ok {
			t.Fatalf("audit export missing key %q: %s", key, raw)
		}
	}
	silos, ok := parsed["silos"].([]any)
	if !ok || len(silos) != 3 {
		t.Fatalf("audit export must carry all three silos: %s", raw)
	}
	if !strings.Contains(raw, "log") || !strings.Contains(raw, "theta_hat") {
		t.Fatalf("audit export must carry decision logs with posterior summaries: %s", raw)
	}

	disabled := newAcceptanceGame(t, 2315)
	disabled.world.Silo1BlackBoxEnabled = false
	if _, err := disabled.GetSilo1BlackBoxAudit(); err == nil {
		t.Fatal("disabled feature must refuse the audit export")
	}
}

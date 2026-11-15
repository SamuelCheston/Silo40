package service

import (
	"math"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// stewardshipExperimentFixture 构建一个已启用权衡层 (Initialized=true) 的最小三堡
// 服务：驻地 (ID 1)、1 号 (ID 2, 带 silo1FullTrait)、18 号 (ID 3)。时钟默认
// Y122M1。map 由被测代码防御性初始化，这里不预建，以覆盖存档回读 nil 表路径。
func stewardshipExperimentFixture() (*GameService, *model.Silo, *model.Silo) {
	station := &model.Silo{ID: 1, Name: "驻地"}
	silo1 := &model.Silo{ID: 2, Name: "1 号地堡", Traits: []string{silo1FullTrait}}
	silo18 := &model.Silo{ID: 3, Name: "18 号地堡"}
	world := &model.World{
		CurrentYear:  122,
		CurrentMonth: 1,
		PlayerSiloID: 1,
		Silo1:        model.Silo1State{Alive: true, OrderPhase: "LOCAL_SIM"},
		Silo1Stewardship: model.Silo1StewardshipState{
			Initialized: true,
		},
		Flags: map[string]bool{},
	}
	svc := &GameService{
		world: world,
		silos: []*model.Silo{station, silo1, silo18},
		silo:  station,
	}
	return svc, station, silo18
}

// TestStewardshipRegisterDirectCall 直接调用登记函数：启用时落登记，重复签发不
// 重复登记，未启用时空转。
func TestStewardshipRegisterDirectCall(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 3

	svc.stewardshipRegisterClearingObligation(station.ID)
	exp := &svc.world.Silo1Stewardship.Experiment
	want := stewardshipExperimentMonthIndex(122, 3)
	if got, ok := exp.Scheduled[station.ID]; !ok || got != want {
		t.Fatalf("registration must record the current month: got=%v ok=%v want=%v", got, ok, want)
	}

	// 重复签发不重复登记 (值不被覆写)。
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 5
	svc.stewardshipRegisterClearingObligation(station.ID)
	if got := exp.Scheduled[station.ID]; got != want {
		t.Fatalf("duplicate issuance must not re-register: got=%v want=%v", got, want)
	}
	if len(exp.Scheduled) != 1 {
		t.Fatalf("expected exactly one scheduled entry, got %d", len(exp.Scheduled))
	}
}

// TestStewardshipRegisterRequiresEnabled 未启用 (零值) 时登记空转，不写状态。
func TestStewardshipRegisterRequiresEnabled(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.Silo1Stewardship.Initialized = false
	svc.stewardshipRegisterClearingObligation(station.ID)
	exp := &svc.world.Silo1Stewardship.Experiment
	if len(exp.Scheduled) != 0 {
		t.Fatalf("disabled stewardship must not register, got %+v", exp.Scheduled)
	}
	if got := svc.stewardshipRetentionDiversity(); got != 1 {
		t.Fatalf("disabled retention diversity must be the neutral 1, got %v", got)
	}
}

// TestStewardshipUltimatumRegistersViaPipeline 真实 CreateGame + advanceMonth
// 驱动：首月启用权衡层，随后高压力 + 有组织煽动触发通牒签发，产生清除义务登记。
func TestStewardshipUltimatumRegistersViaPipeline(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "stewardship-register.db"))
	if _, err := svc.CreateGame(threeSiloRequest(61)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	// 首月：阶段 2b' 启用权衡层 (此时时钟仍在 Y122M1，无通牒)。
	advanceMonthForTest(t, svc)
	if !svc.world.Silo1Stewardship.Initialized {
		t.Fatal("stewardship must be initialized after the first multi-silo month")
	}

	// 持续施压并推进，直到通牒签发并产生登记 (上限 8 月防呆)。复合压力
	// OrderPressure = 0.6*叛乱 + 0.4*平均恐慌，需越过 0.45 门槛；叛乱 0.8 + 恐慌
	// 0.5 实测可稳定触发且驻地不覆灭。
	svc.agent.PropagandaLevel = 3
	registered := false
	for i := 0; i < 8 && !registered; i++ {
		svc.silo.Rebellion = 0.8
		for pi := range svc.silo.Professions {
			svc.silo.Professions[pi].PanicValue = 0.5
		}
		advanceMonthForTest(t, svc)
		if len(svc.world.Silo1Stewardship.Experiment.Scheduled) > 0 {
			registered = true
		}
		if svc.gameOver() {
			break
		}
	}
	if !registered {
		t.Fatal("ultimatum issuance never produced a clearing-obligation registration within 8 months")
	}
	// 登记的必须是真实存在的非 1 号堡。
	for id := range svc.world.Silo1Stewardship.Experiment.Scheduled {
		silo := svc.siloByID(id)
		if silo == nil {
			t.Fatalf("registered silo %d does not exist", id)
		}
		if svc.stewardshipExperimentIsSilo1(silo) {
			t.Fatalf("silo 1 itself must not be a clearing target")
		}
	}
}

// clearDiversityLossAt 在指定月份清除一个已登记堡，返回当月结算后的多样性损耗。
// 义务在游戏开局 (Y122M1) 登记，清除发生在目标月，以隔离 proximity 权重。
func clearDiversityLossAt(clearYear, clearMonth int) float64 {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 1
	svc.stewardshipRegisterClearingObligation(station.ID)
	svc.world.CurrentYear, svc.world.CurrentMonth = clearYear, clearMonth
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "DEATH"}
	var logs []string
	var stories []model.StoryEvent
	svc.stewardshipExperimentMonth(&logs, &stories)
	return svc.world.Silo1Stewardship.Experiment.DiversityLoss
}

// TestStewardshipClearProximityWeighting 同样一次清除，发生在早期 (远离 Y130M1)
// 的多样性损耗大于接近 Y130M1 (proximity 更小)。
func TestStewardshipClearProximityWeighting(t *testing.T) {
	early := clearDiversityLossAt(122, 6) // monthsRemaining=91, proximity≈0.948
	late := clearDiversityLossAt(129, 12) // monthsRemaining=1,  proximity≈0.010
	if early <= 0 || late <= 0 {
		t.Fatalf("clearing must add positive diversity loss: early=%v late=%v", early, late)
	}
	if early <= late {
		t.Fatalf("early clear must cost more diversity than a near-end clear: early=%v late=%v", early, late)
	}
	// proximity 归一：开局 (Y122M1) proximity=1，终局参照 (Y130M1) proximity=0。
	if got := stewardshipExperimentProximity(122, 1); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("proximity at Y122M1 = %v, want 1.0", got)
	}
	if got := stewardshipExperimentProximity(130, 1); got != 0 {
		t.Fatalf("proximity at Y130M1 = %v, want 0", got)
	}
}

// TestStewardshipClearDetection 清除成立：被登记堡 DEATH (大小写不敏感) 记入
// Cleared 并计入损耗；已清除堡不重复计费 (次月仅衰减)。
func TestStewardshipClearDetection(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 1
	svc.stewardshipRegisterClearingObligation(station.ID)
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "death"} // 小写，验 EqualFold
	var logs []string
	var stories []model.StoryEvent
	svc.stewardshipExperimentMonth(&logs, &stories)
	exp := &svc.world.Silo1Stewardship.Experiment
	if _, ok := exp.Cleared[station.ID]; !ok {
		t.Fatal("destroyed registered silo must be recorded as cleared")
	}
	if exp.DiversityLoss <= 0 {
		t.Fatalf("clearing must add diversity loss, got %v", exp.DiversityLoss)
	}
	before := exp.DiversityLoss
	svc.stewardshipExperimentMonth(&logs, &stories)
	if exp.DiversityLoss >= before {
		t.Fatalf("already-cleared silo must not be charged again (expect pure decay): %v -> %v", before, exp.DiversityLoss)
	}
}

// TestStewardshipUnregisteredDeathNoLoss 未被登记的堡覆灭不计多样性损耗 (清除
// 义务只来自通牒登记)。
func TestStewardshipUnregisteredDeathNoLoss(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 1
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "DEATH"}
	var logs []string
	var stories []model.StoryEvent
	svc.stewardshipExperimentMonth(&logs, &stories)
	exp := &svc.world.Silo1Stewardship.Experiment
	if exp.DiversityLoss != 0 || len(exp.Cleared) != 0 {
		t.Fatalf("unregistered death must not be charged: loss=%v cleared=%+v", exp.DiversityLoss, exp.Cleared)
	}
}

// TestStewardshipMissSettlesOnce miss 一次性结算：登记超过窗口仍未覆灭记一次
// 缺口并写审计；窗口内不记；之后月份不重复计费。
func TestStewardshipMissSettlesOnce(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.CurrentYear, svc.world.CurrentMonth = 122, 1
	svc.stewardshipRegisterClearingObligation(station.ID)
	var logs []string
	var stories []model.StoryEvent

	// 窗口内 (23 个月后，Y123M12)：仍存活 → 不构成 miss。
	svc.world.CurrentYear, svc.world.CurrentMonth = 123, 12
	svc.stewardshipExperimentMonth(&logs, &stories)
	exp := &svc.world.Silo1Stewardship.Experiment
	if len(exp.Misses) != 0 || exp.Missed[station.ID] {
		t.Fatalf("no miss before the deadline: misses=%+v missed=%v", exp.Misses, exp.Missed)
	}

	// 窗口到期 (24 个月后，Y124M1)：构成 miss。
	svc.world.CurrentYear, svc.world.CurrentMonth = 124, 1
	svc.stewardshipExperimentMonth(&logs, &stories)
	if len(exp.Misses) != 1 || !exp.Missed[station.ID] {
		t.Fatalf("deadline reached must settle exactly one miss: misses=%+v missed=%v", exp.Misses, exp.Missed)
	}
	if exp.Misses[0].SiloID != station.ID || exp.Misses[0].Year != 124 || exp.Misses[0].Month != 1 {
		t.Fatalf("miss audit record must carry silo and settlement month: %+v", exp.Misses[0])
	}
	if exp.FulfillmentDeficit <= 0 {
		t.Fatalf("miss must raise fulfillment deficit, got %v", exp.FulfillmentDeficit)
	}

	// 再过一月：不重复结算 (仍只有一条审计)。
	svc.world.CurrentYear, svc.world.CurrentMonth = 124, 2
	svc.stewardshipExperimentMonth(&logs, &stories)
	if len(exp.Misses) != 1 {
		t.Fatalf("miss must settle exactly once, got %d records", len(exp.Misses))
	}
}

// TestStewardshipDecayMonotonic 无新事件时两项指标随月度回合单调下降。
func TestStewardshipDecayMonotonic(t *testing.T) {
	svc, _, _ := stewardshipExperimentFixture()
	exp := &svc.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	exp.FulfillmentDeficit = 0.5
	exp.DiversityLoss = 0.5
	var logs []string
	var stories []model.StoryEvent
	prevF, prevD := exp.FulfillmentDeficit, exp.DiversityLoss
	for i := 0; i < 6; i++ {
		svc.stewardshipExperimentMonth(&logs, &stories)
		if exp.FulfillmentDeficit >= prevF || exp.DiversityLoss >= prevD {
			t.Fatalf("decay must be strictly monotonic (month %d): F %v->%v D %v->%v",
				i, prevF, exp.FulfillmentDeficit, prevD, exp.DiversityLoss)
		}
		if exp.FulfillmentDeficit < 0 || exp.DiversityLoss < 0 {
			t.Fatalf("decay must not go negative: F=%v D=%v", exp.FulfillmentDeficit, exp.DiversityLoss)
		}
		prevF, prevD = exp.FulfillmentDeficit, exp.DiversityLoss
	}
}

// TestStewardshipRetentionDiversity 保留多样性读数：满多样性为 1，候选覆灭按比例
// 下降，损耗把它压向 0，且恒在 [0,1]。
func TestStewardshipRetentionDiversity(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	exp := &svc.world.Silo1Stewardship.Experiment

	// 初始候选 = 非 1 号堡数 (驻地 + 18 号)。
	if got := svc.stewardshipRetentionDiversity(); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("full diversity must read 1.0, got %v", got)
	}
	if initial := exp.Ledger[stewardshipExperimentLedgerInitialCandidates]; initial != 2 {
		t.Fatalf("initial candidates must count non-silo-1 silos, got %d", initial)
	}

	// 覆灭一个候选：ratio = 1/2。
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "DEATH"}
	if got := svc.stewardshipRetentionDiversity(); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("one destroyed candidate must read 0.5, got %v", got)
	}

	// 损耗叠加把它夹到 0。
	exp.DiversityLoss = 2.0
	exp.FulfillmentDeficit = 2.0
	if got := svc.stewardshipRetentionDiversity(); got != 0 {
		t.Fatalf("heavy losses must clamp to 0, got %v", got)
	}

	// 中间值：全活 (ratio=1) − 0.5*0.5 = 0.75。
	station.VictoryStatus = nil
	exp.DiversityLoss = 0.5
	exp.FulfillmentDeficit = 0
	if got := svc.stewardshipRetentionDiversity(); math.Abs(got-0.75) > 1e-9 {
		t.Fatalf("expected 0.75, got %v", got)
	}

	// 任意极端组合恒在 [0,1]。
	for _, dl := range []float64{0, 0.3, 1.0, 5.0} {
		for _, fd := range []float64{0, 0.3, 1.0, 5.0} {
			exp.DiversityLoss = dl
			exp.FulfillmentDeficit = fd
			if got := svc.stewardshipRetentionDiversity(); got < 0 || got > 1 {
				t.Fatalf("retention diversity out of [0,1]: dl=%v fd=%v got=%v", dl, fd, got)
			}
		}
	}
}

// TestStewardshipIdleWhenDisabled 未启用 (零值) 状态下月度回合并读数全部空转，
// 状态零值不变。
func TestStewardshipIdleWhenDisabled(t *testing.T) {
	svc, station, _ := stewardshipExperimentFixture()
	svc.world.Silo1Stewardship.Initialized = false
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "DEATH"}
	var logs []string
	var stories []model.StoryEvent
	svc.stewardshipExperimentMonth(&logs, &stories)
	svc.stewardshipRegisterClearingObligation(station.ID)
	exp := &svc.world.Silo1Stewardship.Experiment
	if exp.FulfillmentDeficit != 0 || exp.DiversityLoss != 0 ||
		len(exp.Scheduled) != 0 || len(exp.Cleared) != 0 || len(exp.Misses) != 0 {
		t.Fatalf("disabled stewardship must keep zero state: %+v", exp)
	}
	if len(logs) != 0 {
		t.Fatalf("disabled stewardship must not log, got %v", logs)
	}
}

// TestStewardshipIdleSingleSilo 单堡 (空邻堡) 配置下权衡层从不启用，月度推进后
// 状态仍为零值。
func TestStewardshipIdleSingleSilo(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "stewardship-single.db"))
	req := model.CreateGameRequest{
		SiloName:   "Single",
		AgentName:  "Player",
		Profession: "IT",
		Seed:       7,
	}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	advanceMonthForTest(t, svc)
	st := svc.world.Silo1Stewardship
	if st.Initialized {
		t.Fatal("single-silo configuration must never enable the stewardship layer")
	}
	if st.Experiment.FulfillmentDeficit != 0 || st.Experiment.DiversityLoss != 0 ||
		len(st.Experiment.Scheduled) != 0 {
		t.Fatalf("single-silo stewardship must stay zero, got %+v", st.Experiment)
	}
}

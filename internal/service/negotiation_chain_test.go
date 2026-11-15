package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 谈判/妥协层验收 (18 设计卷)：信道生命周期、带权决策轨迹的确定性、
// 妥协可达性、妥协后二选一与出发结算加成、未开信道的零 RNG 保证。

func newNegotiationGame(t *testing.T, name string, seed int64) *GameService {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), name))
	if _, err := svc.CreateGame(threeSiloRequest(seed)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

func executePlayerEventAction(t *testing.T, svc *GameService, actionID string) *model.ActionOutcome {
	t.Helper()
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
	if err != nil {
		t.Fatalf("execute %s: %v", actionID, err)
	}
	if !outcome.Result.Executed {
		t.Fatalf("execute %s rejected: %+v", actionID, outcome.Result)
	}
	return outcome
}

// 门控：叛乱不足时开信道动作不可用；叛乱达标后解锁并可执行。
func TestNegotiationOpenChannelGatedByRebellion(t *testing.T) {
	svc := newNegotiationGame(t, "neg-gate.db", 4201)
	svc.silo.Rebellion = 0.1
	if meta := actionByID(svc.availablePlayerActions(), "NEGOTIATION_OPEN_CHANNEL"); meta != nil && meta.Enabled {
		t.Fatalf("open channel must stay locked at rebellion 0.1: %+v", meta)
	}

	svc.silo.Rebellion = 0.5
	meta := actionByID(svc.availablePlayerActions(), "NEGOTIATION_OPEN_CHANNEL")
	if meta == nil || !meta.Enabled {
		t.Fatalf("open channel must unlock at rebellion 0.5: %+v", meta)
	}

	executePlayerEventAction(t, svc, "NEGOTIATION_OPEN_CHANNEL")
	negotiation := svc.silo.Negotiation
	if !negotiation.Initialized || !negotiation.ChannelOpen {
		t.Fatalf("channel must be open after the action: %+v", negotiation)
	}
	switch negotiation.MayorDisposition {
	case model.MayorDispositionHardliner, model.MayorDispositionPragmatist, model.MayorDispositionSurvivor:
	default:
		t.Fatalf("disposition %q out of enum", negotiation.MayorDisposition)
	}
	if len(negotiation.Log) != 1 {
		t.Fatalf("opening must leave exactly one decision-log entry: %+v", negotiation.Log)
	}
	if negotiation.MayorLean <= 0 {
		t.Fatalf("mayor lean must be perceptible after opening: %v", negotiation.MayorLean)
	}
}

// 确定性：同种子 + 同动作序列 → 同样的谈判轨迹 (月度回合＋玩家回合)。
func TestNegotiationTrajectoryDeterministic(t *testing.T) {
	run := func(seed int64, name string) string {
		svc := newNegotiationGame(t, name, seed)
		svc.silo.Rebellion = 0.6
		executePlayerEventAction(t, svc, "NEGOTIATION_OPEN_CHANNEL")
		for i := 0; i < 6 && svc.silo.Negotiation.NegotiationActive(); i++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("advance month %d: %v", i, err)
			}
		}
		raw, err := json.Marshal(svc.silo.Negotiation)
		if err != nil {
			t.Fatalf("marshal negotiation: %v", err)
		}
		return string(raw)
	}
	a := run(4202, "neg-det-a.db")
	b := run(4202, "neg-det-b.db")
	if a != b {
		t.Fatalf("same seed diverged:\n%s\nvs\n%s", a, b)
	}
}

// 月度回合：信道畅通时每月通气一次 (回合数随月份增长或信道崩塌/妥协收场)。
func TestNegotiationMonthlyRoundRuns(t *testing.T) {
	svc := newNegotiationGame(t, "neg-month.db", 4203)
	svc.silo.Rebellion = 0.6
	executePlayerEventAction(t, svc, "NEGOTIATION_OPEN_CHANNEL")
	for i := 0; i < 8; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance month %d: %v", i, err)
		}
	}
	negotiation := svc.silo.Negotiation
	if negotiation.Rounds < 1 {
		t.Fatalf("monthly rounds must run while the channel is open: %+v", negotiation)
	}
	if !negotiation.NegotiationActive() && !negotiation.CompromiseEstablished && !negotiation.Closed {
		t.Fatalf("rounds stopped without a legal ending state: %+v", negotiation)
	}
	if len(negotiation.Log) == 0 || len(negotiation.Log) > model.NegotiationLogCap {
		t.Fatalf("decision log must stay within the ring cap: %d entries", len(negotiation.Log))
	}
	// 决策日志必须可复盘：每条市长决策带归一化权重与因子摘要。
	for _, entry := range negotiation.Log {
		if entry.Decider == "mayor" {
			if len(entry.Weights) != 3 {
				t.Fatalf("mayor log entry must carry three normalized weights: %+v", entry)
			}
			if !strings.Contains(entry.Factors, "正统性") {
				t.Fatalf("mayor log entry must carry the factor summary: %+v", entry)
			}
		}
	}
}

// 妥协可达：真实 RNG 下存在可达路径——开信道、递条款、月度通气，
// 妥协在某个种子上成立 (失败与拖延合法，但机制不得不可达)。
func TestNegotiationCompromiseReachable(t *testing.T) {
	var found *GameService
	for seed := int64(1); seed <= 80 && found == nil; seed++ {
		svc := newNegotiationGame(t, fmt.Sprintf("neg-reach-%d.db", seed), seed)
		svc.silo.Rebellion = 0.75
		executePlayerEventAction(t, svc, "NEGOTIATION_OPEN_CHANNEL")
		executePlayerEventAction(t, svc, "NEGOTIATION_OFFER_TERMS")
		for i := 0; i < 12 && svc.silo.Negotiation.NegotiationActive(); i++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("seed %d month %d: %v", seed, i, err)
			}
			if svc.silo.Negotiation.CompromiseEstablished {
				break
			}
		}
		if svc.silo.Negotiation.CompromiseEstablished {
			found = svc
		}
	}
	if found == nil {
		t.Fatal("compromise never established across 80 seeds: the mechanism is unreachable and must be fixed, not loosened")
	}

	negotiation := found.silo.Negotiation
	if negotiation.CompromiseYear == 0 && negotiation.CompromiseMonth == 0 {
		t.Fatalf("compromise must record when it happened: %+v", negotiation)
	}
	if found.silo.TopStance.Reevaluations < 1 {
		t.Fatalf("compromise must trigger a stance re-evaluation: %+v", found.silo.TopStance)
	}
	// 妥协解锁二选一。
	takeover := actionByID(found.availablePlayerActions(), "NEGOTIATION_TAKEOVER")
	handsOff := actionByID(found.availablePlayerActions(), "NEGOTIATION_HANDS_OFF")
	if takeover == nil || !takeover.Enabled || handsOff == nil || !handsOff.Enabled {
		t.Fatalf("post-compromise choices must unlock: takeover=%+v hands_off=%+v", takeover, handsOff)
	}

	// 选择不接管：组织转向离开，外溢信道可靠度上升。
	found.silo.SpilloverPrep.Channels = append(found.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: "physical_handoff", TargetSilo: 3, Reliability: 0.30,
	})
	executePlayerEventAction(t, found, "NEGOTIATION_HANDS_OFF")
	if found.silo.Negotiation.Takeover != model.NegotiationHandsOff {
		t.Fatalf("hands_off choice not recorded: %+v", found.silo.Negotiation)
	}
	if rel := found.silo.SpilloverPrep.Channels[0].Reliability; rel <= 0.30 {
		t.Fatalf("hands_off must raise spillover channel reliability: %v", rel)
	}
	// 互斥：接管不再可选。
	if meta := actionByID(found.availablePlayerActions(), "NEGOTIATION_TAKEOVER"); meta != nil && meta.Enabled {
		t.Fatalf("takeover must be locked after hands_off: %+v", meta)
	}
}

// 接管：委员会正统性改写、叛乱回落、市长立场以新权重重估 (仍是带权取样)。
func TestNegotiationTakeoverEffects(t *testing.T) {
	svc := newNegotiationGame(t, "neg-takeover.db", 4204)
	svc.silo.Rebellion = 0.7
	svc.silo.Negotiation.Initialized = true
	svc.silo.Negotiation.CompromiseEstablished = true
	beforeRebellion := svc.silo.Rebellion
	beforeReevals := svc.silo.TopStance.Reevaluations
	engine.SetRNG(deterministicTestRNG(4204))

	executePlayerEventAction(t, svc, "NEGOTIATION_TAKEOVER")

	negotiation := svc.silo.Negotiation
	if negotiation.Takeover != model.NegotiationTakeover {
		t.Fatalf("takeover not recorded: %+v", negotiation)
	}
	if svc.silo.Rebellion >= beforeRebellion {
		t.Fatalf("takeover must settle part of the grievance: %v -> %v", beforeRebellion, svc.silo.Rebellion)
	}
	if svc.silo.TopStance.Reevaluations != beforeReevals+1 {
		t.Fatalf("takeover must re-evaluate the mayor stance once: %d -> %d",
			beforeReevals, svc.silo.TopStance.Reevaluations)
	}
	if svc.silo.MayorLegitimacy() <= 0 {
		t.Fatalf("committee legitimacy must be positive: %v", svc.silo.MayorLegitimacy())
	}
}

// 出发结算：不接管加成只改变组织动员，不改变结算的读取对象 (18 §3.7)。
// 夹具把组织动员造成容量瓶颈 (小规模人口、低预案/演练、宽路线与服体)，
// 让组织加成的差异可见。
func TestNegotiationHandsOffBonusInSettlement(t *testing.T) {
	newStation := func() (*GameService, *model.Silo) {
		svc, station, _ := weightedEndingFixture("REBELLION")
		station.TotalPopulation = 300
		station.Rebellion = 0.5
		station.TopStance = model.TopStanceState{Initialized: true, Mayor: model.TopStanceNonBlock, IT: model.TopStanceNonBlock}
		station.EscapePrep = model.EscapePreparation{
			Initialized:     true,
			SuitsFunctional: 200,
			SuppliesPacked:  500,
			MaterialsPacked: 80,
			DrillLevel:      0,
			CareReady:       0.5,
			ReceptionStage:  model.ReceptionStocked,
		}
		return svc, station
	}

	svcBase, base := newStation()
	settleBase := svcBase.settleEscapeDeparture(base, nil)

	svcBonus, bonus := newStation()
	bonus.Negotiation.CompromiseEstablished = true
	bonus.Negotiation.Takeover = model.NegotiationHandsOff
	settleBonus := svcBonus.settleEscapeDeparture(bonus, nil)

	if settleBonus.Departed < settleBase.Departed {
		t.Fatalf("hands_off must not reduce the headcount: %d vs %d", settleBonus.Departed, settleBase.Departed)
	}
	if settleBonus.Capacity <= settleBase.Capacity {
		t.Fatalf("hands_off organization bonus must raise capacity: %d vs %d", settleBonus.Capacity, settleBase.Capacity)
	}
	noted := false
	for _, note := range settleBonus.Notes {
		if strings.Contains(note, "不接管") {
			noted = true
		}
	}
	if !noted {
		t.Fatalf("hands_off bonus must be auditable in settlement notes: %+v", settleBonus.Notes)
	}

	// 接管不加组织加成 (顶层改由叛乱方主导经立场重估体现，不走加成)。
	svcTakeover, takeover := newStation()
	takeover.Negotiation.CompromiseEstablished = true
	takeover.Negotiation.Takeover = model.NegotiationTakeover
	settleTakeover := svcTakeover.settleEscapeDeparture(takeover, nil)
	if settleTakeover.Capacity != settleBase.Capacity {
		t.Fatalf("takeover must not read the hands_off bonus: %d vs %d", settleTakeover.Capacity, settleBase.Capacity)
	}
}

// TestNegotiationAcceptancePlaythrough 无头宿主、固定种子、真实内容与真实
// 门控下的完整对局：叛乱成形 → 开信道 → 试探 → 递条款 → 月度通气 →
// 妥协成立 → 不接管 → 出发结算。谈判只改变"谁在阻碍谁"：结算仍只看
// 逃出去多少人。同种子重跑逐位一致。
func TestNegotiationAcceptancePlaythrough(t *testing.T) {
	run := func(seed int64, name string) (model.EscapeGroup, string) {
		svc := newNegotiationGame(t, name, seed)
		// 叛乱成形＋路线与组织事实齐备 (与出发验收同一夹具口径)：
		// 谈判层接在叛乱链的中后段，前面的里程碑以完成键事实表达。
		svc.silo.Rebellion = 0.75
		completeGateEvent(svc, "special:route_confirmed")
		completeGateEvent(svc, "special:underground_coordination")
		completeGateEvent(svc, "special:escape_window_open")
		completeGateEvent(svc, "player_actions:escape_window_orderly")
		svc.silo.EscapePrep = model.EscapePreparation{
			Initialized: true, SuitsFunctional: 150, SuppliesPacked: 400,
			PlanCompleteness: 0.5, DrillLevel: 0.5, CareReady: 0.4,
			ProductionStart: true, ReceptionStage: model.ReceptionConfirmed,
		}
		svc.agent.ActionPoints = 50

		softExecute := func(actionID string) bool {
			if svc.gameOver() {
				return false
			}
			outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
			return err == nil && outcome.Result.Executed
		}

		// 1) 开信道 (真实门控：叛乱 ≥ 0.30)。
		if !softExecute("NEGOTIATION_OPEN_CHANNEL") {
			return model.EscapeGroup{}, ""
		}
		// 2) 试探一轮：通气不逼表态。
		if !softExecute("NEGOTIATION_PROBE") {
			return model.EscapeGroup{}, ""
		}
		// 3) 递出条款：保障可信度上升，条款在桌。
		if !softExecute("NEGOTIATION_OFFER_TERMS") {
			return model.EscapeGroup{}, ""
		}
		// 4) 月度通气，直到妥协成立或信道崩塌 (两者都是合法结局)。
		for i := 0; i < 12 && svc.silo.Negotiation.NegotiationActive(); i++ {
			if _, err := svc.PassTime(1); err != nil {
				return model.EscapeGroup{}, ""
			}
			if svc.silo.Negotiation.CompromiseEstablished || svc.gameOver() {
				break
			}
		}
		if !svc.silo.Negotiation.CompromiseEstablished {
			return model.EscapeGroup{}, ""
		}
		// 5) 不接管市长职位：带组织转向离开。
		if !softExecute("NEGOTIATION_HANDS_OFF") {
			return model.EscapeGroup{}, ""
		}
		// 6) 出发：终局立场重估 + 出发结算 (真实动作路径)。
		svc.agent.ActionPoints = 50
		if !softExecute("ESCAPE_DEPARTURE_LAUNCH") {
			return model.EscapeGroup{}, ""
		}
		raw, err := json.Marshal(svc.silo.Negotiation)
		if err != nil {
			t.Fatalf("marshal negotiation: %v", err)
		}
		return svc.silo.EscapedGroup, string(raw)
	}

	// 扫描种子：要求真的有人逃出去 (存续群体)。妥协只改变阻碍关系，
	// 阻挠下人数自然为 0 是合法输出，但验收要证明这条线能走通到有人离开。
	var foundSeed int64
	var group model.EscapeGroup
	var negotiationJSON string
	launched, escaped := 0, 0
	for seed := int64(1); seed <= 200 && foundSeed == 0; seed++ {
		g, nj := run(seed, fmt.Sprintf("neg-playthrough-%d.db", seed))
		if nj == "" {
			continue
		}
		launched++
		if g.Departed > 0 && g.Surviving {
			escaped++
			foundSeed, group, negotiationJSON = seed, g, nj
		}
	}
	t.Logf("playthrough seeds: %d reached departure, %d escaped with a surviving group", launched, escaped)
	if foundSeed == 0 {
		t.Fatal("no seed produced a surviving escape group through the negotiation line in 200 tries: the line is unreachable and must be fixed, not loosened")
	}

	// 结算一致性：存续群体与人数口径自洽 (资格只读群体事实)。
	if group.Departed == 0 && group.Surviving {
		t.Fatalf("zero headcount cannot yield a surviving group: %+v", group)
	}
	if group.Departed > 0 && !group.Initialized {
		t.Fatalf("a departure with people must initialize the group: %+v", group)
	}

	// 确定性：同种子重跑，逃出群体与谈判轨迹逐位一致。
	group2, negotiationJSON2 := run(foundSeed, "neg-playthrough-replay.db")
	if group2 != group {
		t.Fatalf("same seed must reproduce the escape outcome: %+v vs %+v", group, group2)
	}
	if negotiationJSON2 != negotiationJSON {
		t.Fatalf("same seed must reproduce the negotiation trajectory:\n%s\nvs\n%s", negotiationJSON, negotiationJSON2)
	}
	t.Logf("playthrough settled under seed %d: %+v", foundSeed, group)
}

// 非回归核心保证：未开信道的运行不消耗任何新 RNG。
// 同一 RNG 种子下，走若干次未激活的月度钩子后取样，必须与不走钩子的
// 基线逐位一致——钩子在守卫处返回，不触碰随机流。
func TestNegotiationInactiveConsumesNoRNG(t *testing.T) {
	svc := newNegotiationGame(t, "neg-rng.db", 4205)
	if svc.silo.Negotiation.NegotiationActive() {
		t.Fatalf("fresh game must have an inactive negotiation layer: %+v", svc.silo.Negotiation)
	}

	engine.SetRNG(deterministicTestRNG(9999))
	for i := 0; i < 5; i++ {
		svc.runNegotiationMonth(svc.silo)
	}
	after := make([]float64, 8)
	for i := range after {
		after[i] = engine.DrawFloat64()
	}

	engine.SetRNG(deterministicTestRNG(9999))
	for i := range after {
		if baseline := engine.DrawFloat64(); after[i] != baseline {
			t.Fatalf("inactive negotiation hook consumed RNG: draw %d = %v, baseline %v", i, after[i], baseline)
		}
	}
}

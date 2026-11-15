package engine

import (
	"math"
	"sort"
	"strconv"

	"silo40/internal/model"
)

// §5.1 维护方式与忠诚响应系数。
const (
	MaintGoods  = "goods"
	MaintFear   = "fear"
	MaintCareer = "career"
	MaintTruth  = "truth"
)

const (
	etaGoods  = 0.20
	etaFear   = 0.30
	etaCareer = 0.25
	etaTruth  = 0.35

	loyaltyBaseline  = 0.8  // 常规忠诚月回归基线 (§3.2 初始两块忠诚均为 0.8)
	regularReversion = 0.03 // 非恐惧维护每月向基线回归 3%
	fearReversion    = 0.08 // 恐惧维护每月回归 8%

	rOpInitial = 60
	rOpIncome  = 8
	rOpMax     = 100

	// §5.5 反击上限 (TODO(balance): 档位为设计值)。
	rOpTwoActionsMin  = 50
	rOpOneActionMin   = 12
	lowIntensityCost  = 7 // <12 时只做负担得起的低强度动作
	auditActionKeeper = 12
)

// ItAction IT 候选动作 (§5.1 维护 + §5.3 七类隐蔽动作 + 反击)。结构性边界
// (§5.4)：公开亮相与反上层控制方不在候选集——目录里不存在这类条目，不是成本高。
type ItAction struct {
	ID                string
	BaseCost          float64
	WShock            float64 // ΔW_pred (经冲击队列结算)
	RSDelta           float64 // ΔR_s_pred (负=保护性)
	DirectExposure    float64 // 直接暴露概率基数 (只有法警调度非零)
	LossOfControlProb float64 // 失控概率 (可能迫使再次调度法警)
	Group             string  // 维护动作目标组：bailiff / it_internal
	Method            string  // 维护方式：goods/fear/career/truth
	RegimeAverting    bool    // Judicial 游离时是否属于"挽留"动作
	Counter           bool    // 反击动作：只在玩家威胁出现时可选 (§7.4)
}

// ItActionCatalog 返回按 ID 升序的候选动作集 (确定性)。
// TODO(balance): 维护动作基础成本 5 为 sim 起点 (§5.3 表未列维护成本)。
func ItActionCatalog() []ItAction {
	return []ItAction{
		{ID: "control_information", BaseCost: 6, WShock: 2, LossOfControlProb: 0.03},
		{ID: "counter_investigate_player", BaseCost: 8, Counter: true},
		{ID: "counter_seize_materials", BaseCost: 9, WShock: 2, Counter: true},
		{ID: "delay_enforcement", BaseCost: 7, WShock: 3, LossOfControlProb: 0.06},
		{ID: "dispatch_bailiff", BaseCost: 10, WShock: 4, DirectExposure: 0.18},
		{ID: "forge_records", BaseCost: 7, WShock: 2, LossOfControlProb: 0.04},
		{ID: "maintain_bailiff_career", BaseCost: 5, Group: "bailiff", Method: MaintCareer},
		{ID: "maintain_bailiff_fear", BaseCost: 5, Group: "bailiff", Method: MaintFear},
		{ID: "maintain_bailiff_goods", BaseCost: 5, Group: "bailiff", Method: MaintGoods},
		{ID: "maintain_bailiff_truth", BaseCost: 5, Group: "bailiff", Method: MaintTruth},
		{ID: "maintain_itinternal_career", BaseCost: 5, Group: "it_internal", Method: MaintCareer},
		{ID: "maintain_itinternal_fear", BaseCost: 5, Group: "it_internal", Method: MaintFear},
		{ID: "maintain_itinternal_goods", BaseCost: 5, Group: "it_internal", Method: MaintGoods},
		{ID: "maintain_itinternal_truth", BaseCost: 5, Group: "it_internal", Method: MaintTruth},
		{ID: "protocol_disposal", BaseCost: 7, WShock: 2, RSDelta: -0.04, LossOfControlProb: 0.02},
		{ID: "quiet_disappearance", BaseCost: 12, WShock: 6, LossOfControlProb: 0.10, RegimeAverting: true},
		{ID: "suppress_signal", BaseCost: 8, WShock: 3, RSDelta: -0.03, LossOfControlProb: 0.05},
	}
}

// InitializeItAuthority §3.2/§5.2 初始：两块忠诚 0.8，R_op 60。
func InitializeItAuthority(silo *model.Silo) {
	if silo == nil || silo.ItAuthority.Initialized {
		return
	}
	silo.ItAuthority = model.ItAuthorityState{
		Initialized:       true,
		ROp:               rOpInitial,
		BailiffLoyalty:    loyaltyBaseline,
		ITInternalLoyalty: loyaltyBaseline,
	}
}

// ItCostCurve §5.2 弱势成本曲线：ceil(base·(1+3·(1-W/100)^2))。
func ItCostCurve(baseCost, w float64) float64 {
	return math.Ceil(baseCost * (1 + 3*math.Pow(1-w/100, 2)))
}

// itGroupWeight §5.1 联盟权重。
func itGroupWeight(group string) float64 {
	if group == "bailiff" {
		return 0.65
	}
	return 0.35
}

func itGroupLoyalty(state *model.ItAuthorityState, group string) (regular, fear float64) {
	if group == "bailiff" {
		return state.BailiffLoyalty, state.BailiffFear
	}
	return state.ITInternalLoyalty, state.ITInternalFear
}

func applyItMaintenance(state *model.ItAuthorityState, group, method string) {
	eta := 0.0
	switch method {
	case MaintGoods:
		eta = etaGoods
	case MaintFear:
		eta = etaFear
	case MaintCareer:
		eta = etaCareer
	case MaintTruth:
		eta = etaTruth
	}
	if eta == 0 {
		return
	}
	if group == "bailiff" {
		if method == MaintFear {
			state.BailiffFear = clamp01(state.BailiffFear + eta*(1-state.BailiffFear))
		} else {
			state.BailiffLoyalty = clamp01(state.BailiffLoyalty + eta*(1-state.BailiffLoyalty))
		}
		return
	}
	if method == MaintFear {
		state.ITInternalFear = clamp01(state.ITInternalFear + eta*(1-state.ITInternalFear))
	} else {
		state.ITInternalLoyalty = clamp01(state.ITInternalLoyalty + eta*(1-state.ITInternalLoyalty))
	}
}

// playerThreatVisible IT 可观察到的玩家进展 (§5.5 反击触发)：公开证据、
// 核验名单或已成形的曝光。IT 读不到玩家私人信念。
func playerThreatVisible(silo *model.Silo) bool {
	return len(silo.PublicEvidence) > 0 || len(silo.VerifiedEvidence) > 0 || silo.Authority.E >= 0.2
}

// itUtility §5.5 双动机 utility。IT 读自己的预测 (believed 一侧)，不读真实
// 全知结果；本轮预测值取动作声明效果 (TODO(balance): 预测误差之后接 believed
// 条款与碎片)。反击动作只在玩家威胁可见时进入候选。
func itUtility(action ItAction, silo *model.Silo, cost float64) float64 {
	if action.Counter && !playerThreatVisible(silo) {
		return -1000
	}
	deltaK := 0.0
	if action.Group != "" {
		state := &silo.ItAuthority
		regular, fear := itGroupLoyalty(state, action.Group)
		lg := clamp01(regular + fear)
		eta := etaGoods
		switch action.Method {
		case MaintFear:
			eta = etaFear
		case MaintCareer:
			eta = etaCareer
		case MaintTruth:
			eta = etaTruth
		}
		deltaK = itGroupWeight(action.Group) * eta * (1 - lg)
	}
	regimeAverted := 0.0
	if action.RegimeAverting && silo.Authority.Judicial.Regime != "IT" && silo.Authority.Judicial.Regime != "PACT" {
		regimeAverted = 1
	}
	// W 收益边际递减：威势接近上限时继续加码的价值趋零，IT 不会在安全期
	// 无限度消耗资源抬 W (TODO(balance): 线性递减为 sim 起点)。
	wValue := action.WShock / 6 * (1 - clamp(silo.Authority.W, 0, 100)/100)
	self := wValue + 1.5*regimeAverted + 0.5*deltaK
	public := -action.RSDelta / 0.05
	utility := 0.5*self + 0.5*public - 0.01*cost
	if action.Counter {
		// TODO(balance): 反击加成 1.5 为 sim 起点 (§7.4：承受至少一次反击)。
		utility += 1.5
	}
	return utility
}

// RunITAuthorityMonthly §5 月度 IT 权威回合：忠诚回归、动作选择与执行、
// 资源结算。确定性选择 + 单一 RNG 流 (失控/暴露骰)。在 Protocol 结算之后。
func (e *GameEngine) RunITAuthorityMonthly(silo *model.Silo, agent *model.Agent) {
	if silo == nil || silo.VictoryStatus != nil {
		return
	}
	InitializeItAuthority(silo)
	state := &silo.ItAuthority

	// 1. 忠诚回归：非恐惧向基线 3%，恐惧衰减 8%。
	state.BailiffLoyalty = clamp01(state.BailiffLoyalty + regularReversion*(loyaltyBaseline-state.BailiffLoyalty))
	state.ITInternalLoyalty = clamp01(state.ITInternalLoyalty + regularReversion*(loyaltyBaseline-state.ITInternalLoyalty))
	state.BailiffFear = clamp01(state.BailiffFear * (1 - fearReversion))
	state.ITInternalFear = clamp01(state.ITInternalFear * (1 - fearReversion))

	// 2. 反击上限 (§5.5 末段)。
	maxActions := 0
	lowIntensityOnly := false
	switch {
	case state.ROp >= rOpTwoActionsMin:
		maxActions = 2
	case state.ROp >= rOpOneActionMin:
		maxActions = 1
	case state.ROp > 0:
		maxActions = 1
		lowIntensityOnly = true
	}
	if maxActions == 0 {
		state.ROp = math.Min(rOpMax, state.ROp+rOpIncome)
		return
	}

	// 3. 候选评估：utility>0 最高分，同分按 action ID (升序稳定)。
	// E 主消费点 (22-00 裁决 1)：Order 意图方向的执行候选 (声明效果降低风险度，
	// public 效用 > 0) 的效用乘本堡当月执行强度 E (4.55 写入生效槽)。未启用时
	// 按 1.0 处理，行为逐位不变；反应链触发判定不读 E。
	execution := 1.0
	if silo.BlackBox.Initialized {
		execution = silo.BlackBox.CurrentExecution
	}
	catalog := ItActionCatalog()
	type scoredAction struct {
		action ItAction
		cost   float64
		score  float64
	}
	var candidates []scoredAction
	budget := state.ROp
	for _, action := range catalog {
		if lowIntensityOnly && action.BaseCost > lowIntensityCost {
			continue
		}
		cost := ItCostCurve(action.BaseCost, silo.Authority.W)
		if cost > budget {
			continue
		}
		score := itUtility(action, silo, cost)
		if action.RSDelta < 0 {
			score *= execution
		}
		if score <= 0 {
			continue
		}
		candidates = append(candidates, scoredAction{action: action, cost: cost, score: score})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].action.ID < candidates[j].action.ID
	})

	spent := 0.0
	executed := 0
	for _, candidate := range candidates {
		if executed >= maxActions || candidate.cost > state.ROp-spent {
			continue
		}
		spent += candidate.cost
		executed++
		e.executeItAction(silo, agent, candidate.action)
		state.LastActions = append(state.LastActions, candidate.action.ID)
		if len(state.LastActions) > auditActionKeeper {
			state.LastActions = state.LastActions[len(state.LastActions)-auditActionKeeper:]
		}
	}

	// 4. 资源结算：收入恒定，成本随 W 下降上升 → 反击随资源枯竭衰减。
	state.ROp = clamp(state.ROp+rOpIncome-spent, 0, rOpMax)
}

// executeItAction 执行一个隐蔽动作的真实后果。
func (e *GameEngine) executeItAction(silo *model.Silo, agent *model.Agent, action ItAction) {
	state := &silo.ItAuthority

	// 维护动作：忠诚响应。
	if action.Group != "" {
		applyItMaintenance(state, action.Group, action.Method)
		return
	}

	// 反击动作 (§7.4：玩家须承受至少一次反击；强度随资源枯竭衰减)。
	// TODO(balance): 怀疑度 0.08 与戒备 0.15 为 sim 起点。
	switch action.ID {
	case "counter_investigate_player":
		if agent != nil {
			agent.SuspicionLevel = math.Max(0, agent.SuspicionLevel+0.08)
		}
	case "counter_seize_materials":
		for i := range silo.Professions {
			prof := &silo.Professions[i]
			if prof.Name == "Judicial" || prof.Name == "IT" {
				prof.Alertness = clampUnit(prof.Alertness + 0.15)
			}
		}
	}

	// W 效果经冲击队列在下一次权威更新消费。
	if action.WShock != 0 {
		QueueAuthorityShock(silo, action.WShock)
	}

	// 条款衔接 (§5.3 最后一列)。
	switch action.ID {
	case "forge_records":
		// 篡改必需报告构成违反 (无授权，隐蔽执行)。
		if hasPendingReporting(silo) {
			RecordProtocolAction(silo, e.ProtocolClauses, "evidence_forgery", false)
		}
	case "protocol_disposal":
		// 恢复一项 Protocol 目标：关闭最早一条未达成偏离 (条款 ID 升序)。
		closeEarliestUnmet(silo)
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+action.RSDelta)
	case "suppress_signal":
		if hasPendingReporting(silo) {
			silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+action.RSDelta)
		}
	case "quiet_disappearance":
		// Protocol 未授权即属违反 (隐蔽执行，联盟不完全知悉)。
		RecordProtocolAction(silo, e.ProtocolClauses, "unauthorized_disappearance", false)
	}

	// 失控骰：失控可能迫使再次调度法警 (二次暴露风险)。
	lostControl := false
	if action.LossOfControlProb > 0 && rng.Float64() < action.LossOfControlProb {
		lostControl = true
	}

	// 直接暴露：唯一日常泄漏链是法警调度 (§5.3/§11.2)；失控的二次收场同样
	// 走调度暴露。暴露只生成事实观察，仍须经 B/π 与公开传播才影响 E/Q。
	if action.ID == "dispatch_bailiff" || lostControl {
		e.exposeJudicialDispatch(silo, action.DirectExposure)
	}
}

// exposeJudicialDispatch p_expose = clamp(0.18·(1+0.5·alert)·(1+0.5·failed), 0, 0.85)。
// 暴露生成法警调度事实观察 (进入见证者信念层)，仍须经 B/π 与公开传播才影响 E/Q。
func (e *GameEngine) exposeJudicialDispatch(silo *model.Silo, base float64) {
	if base <= 0 {
		base = 0.18
	}
	state := &silo.ItAuthority
	alert := 0.0
	for i := range silo.Professions {
		if silo.Professions[i].Name == "Judicial" {
			alert = silo.Professions[i].Alertness
			break
		}
	}
	p := clamp(base*(1+0.5*alert), 0, 0.85)
	if rng.Float64() >= p {
		return
	}
	state.DispatchCount++
	factID := "Judicial_1"
	if state.DispatchCount%2 == 0 {
		factID = "Judicial_2"
	}
	obsID := "dispatch:" + strconv.Itoa(state.DispatchCount)
	spec := FragmentSpecFor(factID)
	cfg := DefaultInformationConfig()
	strength := AbsorptionStrength(cfg, 0.9, 0, cfg.IndependenceNew)
	// 司法部自己执行了调度；在场目击 (具名人物层) 也得到观察。
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Name != "Judicial" {
			continue
		}
		belief := EnsureBeliefIn(&prof.Beliefs, factID)
		ObserveFact(cfg, belief, spec, obsID, strength)
	}
	for i := range silo.Residents {
		resident := &silo.Residents[i]
		if !resident.Alive || resident.Influence <= 0 {
			continue
		}
		belief := EnsureBeliefIn(&resident.Beliefs, factID)
		ObserveFact(cfg, belief, spec, obsID, strength)
	}
}

func hasPendingReporting(silo *model.Silo) bool {
	return len(silo.Protocol.Pending) > 0
}

func closeEarliestUnmet(silo *model.Silo) {
	state := &silo.Protocol
	var target *model.ProtocolDeviation
	for i := range state.Deviations {
		dev := &state.Deviations[i]
		if dev.Kind != DeviationUnmet || !dev.Open {
			continue
		}
		if target == nil || dev.ClauseID < target.ClauseID {
			target = dev
		}
	}
	if target != nil {
		target.Open = false
	}
}

package engine

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// 市长与叛乱方谈判/妥协层的决策核心 (18 设计卷 §3、ADR-0011)。
//
// 设计纪律：
//   - 三方 (市长/叛乱方/IT) 的取舍都是带权重的随机决策：权重读状态但
//     不被状态锁死——不存在"叛乱 ≥ X → 必须妥协"的阈值。
//   - 随机源走引擎单一 RNG 流 (地堡种子)：同种子＋同动作序列 → 同样的
//     谈判轨迹。未开信道的运行不进入本文件任何取样，RNG 消耗逐位不变。
//   - 每个决策点产出归一化权重与因子摘要，由服务层写入决策日志，
//     留下可复盘的依据。
//   - 本层只改变"谁在阻碍谁"：产出写入立场权重输入与组织事实，
//     不直接发放胜利；最终结算仍只看逃出去多少人。
//
// 月度回合内的固定取样顺序 (确定性纪律)：
//
//	市长路径 → [镇压: 信道断裂抽取 | 妥协: 叛乱方接受抽取 | 谈判: 叛乱方姿态抽取]
//	          → IT 反应 → [破坏: 信道曝光抽取]

// 市长路径决策的回合强度：月度低强度通气、玩家试探、玩家递条款。
const (
	NegotiationKindMonth = "month"
	NegotiationKindProbe = "probe"
	NegotiationKindTerms = "terms"
)

// 谈判效果的全部数值为初始校准值。TODO(balance)
const (
	negotiationSuppressRebellionDelta   = -0.04 // 镇压：叛乱被压回
	negotiationSuppressPanicDelta       = 0.03  // 镇压：全部门恐慌
	negotiationSuppressLegitimacyDelta  = -0.02 // 镇压：市长付出正统性代价
	negotiationSuppressGuaranteeDelta   = -0.10 // 镇压：背信，保障可信度下降
	negotiationSuppressBreakChance      = 0.30  // 镇压打断信道的概率
	negotiationCompromiseGuaranteeDelta = 0.30  // 妥协成立：最大的保障
	negotiationCompromiseRebellionDelta = -0.08 // 妥协成立：诉求部分兑现
	negotiationPressRebellionDelta      = 0.01  // 叛乱方施压：叛乱小量上升
	negotiationProbeLeanDelta           = 0.05  // 叛乱方试探：倾向观测信号
	negotiationUndermineGuaranteeDelta  = -0.05 // IT 破坏：削保障可信度
	negotiationExposureChance           = 0.20  // IT 破坏时信道被曝光的概率
	negotiationExposurePanicDelta       = 0.04  // 信道曝光：恐慌
	negotiationExposureRebellionDelta   = 0.03  // 信道曝光：叛乱
	// 市长倾向 (玩家可感知) 的初值与漂移。
	negotiationLeanHardliner  = 0.25
	negotiationLeanPragmatist = 0.40
	negotiationLeanSurvivor   = 0.50
	negotiationLeanSuppress   = -0.05 // 镇压后倾向信号下行
	negotiationLeanConcede    = 0.15  // 开出/接受提案后倾向信号上行
)

// 叛乱侵蚀现任市长个人正统性的月率 (ADR-0007 后果条、18 §4)：
// 纯算术不消耗 RNG；叛乱为 0 的局逐位不变。TODO(balance)
const rebellionLegitimacyErosion = 0.10 // 每年、满叛乱时的侵蚀量

// negotiationDiscretionFloor 自由裁量地板 (18 §1.5)：任何状态下每个档位
// 都保留不可压没的最小权重——被围困的市长仍可能孤注一掷镇压，占优的
// 叛乱方仍可能拒绝妥协。这不是强制阈值，而是保证裁量空间永不塌缩。
// TODO(balance)
const negotiationDiscretionFloor = 0.03

func floorDiscretion(v float64) float64 {
	if v < negotiationDiscretionFloor {
		return negotiationDiscretionFloor
	}
	return v
}

// ApplyRebellionLegitimacyErosion 按叛乱水平侵蚀市长正统性 (月度政治更新调用)。
func ApplyRebellionLegitimacyErosion(silo *model.Silo, deltaYears float64) {
	if silo == nil || silo.Rebellion <= 0 || deltaYears <= 0 {
		return
	}
	silo.SetMayorLegitimacy(silo.MayorLegitimacy() - rebellionLegitimacyErosion*silo.Rebellion*deltaYears)
}

// AddTopStanceGuarantee 服务层写入保障可信度的导出入口 (夹 [0,1])。
// 与内容效果 top_stance_guarantee 走同一实现。
func AddTopStanceGuarantee(silo *model.Silo, delta float64) {
	addTopStanceGuarantee(silo, delta)
}

// ReevalTopStanceAfterTakeover 叛乱方接管市长职位后重估市长立场 (18 §3.7)。
// 顶层改由叛乱方主导：权重输入显著偏向配合，但仍是带权取样而非写死——
// 接管不保证配合，只改变权重。随机源走引擎单一 RNG 流。TODO(balance)
func ReevalTopStanceAfterTakeover(silo *model.Silo, world *model.World) {
	if silo == nil {
		return
	}
	silo.TopStance.Initialized = true
	w := mayorStanceWeights(silo, world)
	w.block -= 0.40     // TODO(balance)
	w.nonblock += 0.20  // TODO(balance)
	w.cooperate += 0.50 // TODO(balance)
	silo.TopStance.Mayor = drawStance(w)
	silo.TopStance.Reevaluations++
}

// mayorPathWeights 市长三条路的未归一权重。
type mayorPathWeights struct {
	suppress float64
	talk     float64
	concede  float64
}

func (w mayorPathWeights) clamp() mayorPathWeights {
	if w.suppress < 0 {
		w.suppress = 0
	}
	if w.talk < 0 {
		w.talk = 0
	}
	if w.concede < 0 {
		w.concede = 0
	}
	return w
}

// MayorPathWeights 市长路径决策的权重输入 (18 §3.1)。输入归因：
//   - 个人正统性：高 → 镇压仍可行；
//   - 叛乱水平：高 → 镇压不可行、妥协是现实选项 (读权重，不设强制阈值)；
//   - 相对实力：IT 残余 W 与警察服从构成的镇压力量 vs 叛乱；
//   - 既有承诺：保障可信度 → 妥协有可信出路；
//   - 性格/特质：市长性情 (开信道时一次性抽取)；
//   - 压力：秩序压力 (1 号通牒口径) 与拖延疲劳。
//
// kind 为回合强度 (月度/试探/递条款)；全部系数为初始校准值。TODO(balance)
func MayorPathWeights(silo *model.Silo, world *model.World, kind string) mayorPathWeights {
	w := mayorPathWeights{suppress: 0.40, talk: 0.40, concede: 0.20} // TODO(balance)
	negotiation := silo.Negotiation
	legitimacy := silo.MayorLegitimacy()
	rebellion := silo.Rebellion
	guarantee := silo.TopStance.Guarantee
	force := 0.5*(silo.Authority.W/100.0) + 0.5*silo.Authority.PoliceObedience

	// 正统性：高 → 镇压仍可行。
	w.suppress += 0.35 * legitimacy // TODO(balance)
	w.concede -= 0.10 * legitimacy  // TODO(balance)

	// 叛乱：镇压可行性下降，妥协权重上升。
	w.suppress -= 0.40 * rebellion // TODO(balance)
	w.talk += 0.15 * rebellion     // TODO(balance)
	w.concede += 0.30 * rebellion  // TODO(balance)

	// 相对实力：镇压力量仍在 → 镇压可行。
	w.suppress += 0.30 * force // TODO(balance)
	w.concede -= 0.10 * force  // TODO(balance)

	// 既有承诺：可信的出路才敢放下镇压。
	w.suppress -= 0.10 * guarantee // TODO(balance)
	w.talk += 0.15 * guarantee     // TODO(balance)
	w.concede += 0.35 * guarantee  // TODO(balance)

	// 压力：秩序压力升高 → 妥协的现实性上升。
	pressure := OrderPressure(silo)
	w.concede += 0.20 * pressure // TODO(balance)
	w.suppress -= 0.10 * pressure

	// 性格/特质：性情是持久权重输入。
	switch negotiation.MayorDisposition {
	case model.MayorDispositionHardliner:
		w.suppress *= 1.40 // TODO(balance)
		w.concede *= 0.60  // TODO(balance)
	case model.MayorDispositionPragmatist:
		w.talk *= 1.40 // TODO(balance)
	case model.MayorDispositionSurvivor:
		if silo1CommandSourceGone(silo, world) || silo.Authority.W < 30 { // TODO(balance)
			w.concede *= 1.60 // TODO(balance)
		} else {
			w.talk *= 1.15 // TODO(balance)
		}
	}

	// 拖延疲劳：谈判可以一直拖，但拖不是永远免费的 (权重软衰减，不是阈值闸)。
	w.talk *= 1.0 / (1.0 + 0.15*float64(negotiation.StalledRounds)) // TODO(balance)

	// 回合强度：试探只通气不逼表态；条款在桌 → 妥协的现实性上升。
	switch kind {
	case NegotiationKindProbe:
		w.talk += 0.05 // TODO(balance)
	case NegotiationKindTerms:
		w.concede += 0.25 // TODO(balance)
	}

	w = w.clamp()
	w.suppress = floorDiscretion(w.suppress)
	w.talk = floorDiscretion(w.talk)
	w.concede = floorDiscretion(w.concede)
	return w
}

// DrawMayorPath 按权重在引擎单一 RNG 流上取样一次。遍历顺序固定：
// 镇压 → 谈判 → 妥协。总权重为零时兜底为谈判 (中性档)。
func DrawMayorPath(w mayorPathWeights) string {
	w = w.clamp()
	total := w.suppress + w.talk + w.concede
	if total <= 0 {
		return model.NegotiationPathTalk
	}
	r := DrawFloat64() * total
	if r < w.suppress {
		return model.NegotiationPathSuppress
	}
	if r < w.suppress+w.talk {
		return model.NegotiationPathTalk
	}
	return model.NegotiationPathConcede
}

// NormalizeMayorPath 归一化权重供决策日志复盘。
func NormalizeMayorPath(w mayorPathWeights) map[string]float64 {
	w = w.clamp()
	total := w.suppress + w.talk + w.concede
	if total <= 0 {
		return map[string]float64{model.NegotiationPathSuppress: 0, model.NegotiationPathTalk: 0, model.NegotiationPathConcede: 0}
	}
	return map[string]float64{
		model.NegotiationPathSuppress: round3(w.suppress / total),
		model.NegotiationPathTalk:     round3(w.talk / total),
		model.NegotiationPathConcede:  round3(w.concede / total),
	}
}

// DrawMayorDisposition 开信道时一次性抽取市长性情 (三档近等权)。
// 随机源走引擎单一 RNG 流。TODO(balance)
func DrawMayorDisposition() string {
	r := DrawFloat64()
	switch {
	case r < 1.0/3.0:
		return model.MayorDispositionHardliner
	case r < 2.0/3.0:
		return model.MayorDispositionPragmatist
	default:
		return model.MayorDispositionSurvivor
	}
}

// rebelAcceptWeights 叛乱方接受/拒绝市长妥协提案的未归一权重。
type rebelAcceptWeights struct {
	accept float64
	reject float64
}

func (w rebelAcceptWeights) clamp() rebelAcceptWeights {
	if w.accept < 0 {
		w.accept = 0
	}
	if w.reject < 0 {
		w.reject = 0
	}
	return w
}

// RebelAcceptWeights 叛乱方接受决策的权重输入 (18 §3.3)：
//   - 叛乱水平：高 → 更敢拒绝继续打；
//   - 既有承诺：保障可信度与条款内容 → 接受可信的出路；
//   - 近期镇压：背信记忆 → 拒绝权重上升。
//
// 全部系数为初始校准值。TODO(balance)
func RebelAcceptWeights(silo *model.Silo) rebelAcceptWeights {
	w := rebelAcceptWeights{accept: 0.45, reject: 0.35} // TODO(balance)
	rebellion := silo.Rebellion
	guarantee := silo.TopStance.Guarantee
	suppressed := silo.Negotiation.SuppressRounds

	w.reject += 0.25 * rebellion // TODO(balance)
	w.accept -= 0.10 * rebellion // TODO(balance)
	w.accept += 0.40 * guarantee // TODO(balance)
	if suppressed > 5 {
		suppressed = 5
	}
	w.reject += 0.08 * float64(suppressed) // TODO(balance)
	w = w.clamp()
	w.accept = floorDiscretion(w.accept)
	w.reject = floorDiscretion(w.reject)
	return w
}

// DrawRebelAccept 按权重取样叛乱方是否接受。遍历顺序固定：接受 → 拒绝。
// 总权重为零时兜底为拒绝 (失败是合法结果，不默认放行)。
func DrawRebelAccept(w rebelAcceptWeights) bool {
	w = w.clamp()
	total := w.accept + w.reject
	if total <= 0 {
		return false
	}
	return DrawFloat64()*total < w.accept
}

// NormalizeRebelAccept 归一化权重供决策日志复盘。
func NormalizeRebelAccept(w rebelAcceptWeights) map[string]float64 {
	w = w.clamp()
	total := w.accept + w.reject
	if total <= 0 {
		return map[string]float64{"accept": 0, "reject": 0}
	}
	return map[string]float64{"accept": round3(w.accept / total), "reject": round3(w.reject / total)}
}

// rebelPostureWeights 叛乱方姿态的未归一权重。
type rebelPostureWeights struct {
	press float64
	probe float64
	offer float64
}

func (w rebelPostureWeights) clamp() rebelPostureWeights {
	if w.press < 0 {
		w.press = 0
	}
	if w.probe < 0 {
		w.probe = 0
	}
	if w.offer < 0 {
		w.offer = 0
	}
	return w
}

// RebelPostureWeights 叛乱方姿态决策的权重输入 (18 §3.3)。
// 全部系数为初始校准值。TODO(balance)
func RebelPostureWeights(silo *model.Silo) rebelPostureWeights {
	w := rebelPostureWeights{press: 0.35, probe: 0.40, offer: 0.25} // TODO(balance)
	rebellion := silo.Rebellion
	guarantee := silo.TopStance.Guarantee
	stalled := silo.Negotiation.StalledRounds

	w.press += 0.20 * rebellion       // TODO(balance)
	w.offer += 0.15 * (1 - guarantee) // TODO(balance)
	if stalled > 4 {
		stalled = 4
	}
	w.offer += 0.05 * float64(stalled) // TODO(balance)
	w = w.clamp()
	w.press = floorDiscretion(w.press)
	w.probe = floorDiscretion(w.probe)
	w.offer = floorDiscretion(w.offer)
	return w
}

// DrawRebelPosture 按权重取样叛乱方姿态。遍历顺序固定：施压 → 试探 → 递条款。
// 总权重为零时兜底为试探 (维持通气)。
func DrawRebelPosture(w rebelPostureWeights) string {
	w = w.clamp()
	total := w.press + w.probe + w.offer
	if total <= 0 {
		return model.NegotiationPostureProbe
	}
	r := DrawFloat64() * total
	if r < w.press {
		return model.NegotiationPosturePress
	}
	if r < w.press+w.probe {
		return model.NegotiationPostureProbe
	}
	return model.NegotiationPostureOffer
}

// NormalizeRebelPosture 归一化权重供决策日志复盘。
func NormalizeRebelPosture(w rebelPostureWeights) map[string]float64 {
	w = w.clamp()
	total := w.press + w.probe + w.offer
	if total <= 0 {
		return map[string]float64{model.NegotiationPosturePress: 0, model.NegotiationPostureProbe: 0, model.NegotiationPostureOffer: 0}
	}
	return map[string]float64{
		model.NegotiationPosturePress: round3(w.press / total),
		model.NegotiationPostureProbe: round3(w.probe / total),
		model.NegotiationPostureOffer: round3(w.offer / total),
	}
}

// itReactionWeights IT 对谈判信道反应的未归一权重。
type itReactionWeights struct {
	standAside float64
	undermine  float64
	watch      float64
}

func (w itReactionWeights) clamp() itReactionWeights {
	if w.standAside < 0 {
		w.standAside = 0
	}
	if w.undermine < 0 {
		w.undermine = 0
	}
	if w.watch < 0 {
		w.watch = 0
	}
	return w
}

// ITReactionWeights IT 反应决策的权重输入 (18 §3.4)：
//   - 残余力量 W：越高、Protocol 执行核心越完整，越倾向破坏；
//   - 1 号命令源：专线仍在下令 → 破坏加权；静默/覆灭 → 不干涉加权；
//   - 既有承诺：保障可信度是 IT 的求生路径 (握手段换安全)；
//   - 警察服从已转移时，破坏失去执行手段。
//
// 全部系数为初始校准值。TODO(balance)
func ITReactionWeights(silo *model.Silo, world *model.World) itReactionWeights {
	w := itReactionWeights{standAside: 0.40, undermine: 0.30, watch: 0.30} // TODO(balance)
	wResidual := silo.Authority.W / 100.0
	guarantee := silo.TopStance.Guarantee
	rebellion := silo.Rebellion

	w.undermine += 0.30 * wResidual  // TODO(balance)
	w.standAside -= 0.15 * wResidual // TODO(balance)

	if silo1CommandSourceGone(silo, world) {
		w.standAside += 0.20 // TODO(balance)
		w.undermine -= 0.10  // TODO(balance)
	} else {
		w.undermine += 0.15 // TODO(balance)
	}

	w.standAside += 0.30 * guarantee // TODO(balance)
	w.watch += 0.15 * rebellion      // TODO(balance)

	if silo.Authority.PoliceObedience < 0.5 { // TODO(balance)
		w.undermine -= 0.15 // TODO(balance)
	}
	w = w.clamp()
	w.standAside = floorDiscretion(w.standAside)
	w.undermine = floorDiscretion(w.undermine)
	w.watch = floorDiscretion(w.watch)
	return w
}

// DrawITReaction 按权重取样 IT 反应。遍历顺序固定：不干涉 → 破坏 → 观望。
// 总权重为零时兜底为观望。
func DrawITReaction(w itReactionWeights) string {
	w = w.clamp()
	total := w.standAside + w.undermine + w.watch
	if total <= 0 {
		return model.NegotiationITWatch
	}
	r := DrawFloat64() * total
	if r < w.standAside {
		return model.NegotiationITStandAside
	}
	if r < w.standAside+w.undermine {
		return model.NegotiationITUndermine
	}
	return model.NegotiationITWatch
}

// NormalizeITReaction 归一化权重供决策日志复盘。
func NormalizeITReaction(w itReactionWeights) map[string]float64 {
	w = w.clamp()
	total := w.standAside + w.undermine + w.watch
	if total <= 0 {
		return map[string]float64{model.NegotiationITStandAside: 0, model.NegotiationITUndermine: 0, model.NegotiationITWatch: 0}
	}
	return map[string]float64{
		model.NegotiationITStandAside: round3(w.standAside / total),
		model.NegotiationITUndermine:  round3(w.undermine / total),
		model.NegotiationITWatch:      round3(w.watch / total),
	}
}

// NegotiationFactorSummary 决策时刻的因子读数摘要 (决策日志归因)。
func NegotiationFactorSummary(silo *model.Silo) string {
	return fmt.Sprintf("正统性 %.2f、叛乱 %.2f、W %.0f、警察服从 %.2f、保障可信度 %.2f、秩序压力 %.2f、性情 %s、拖延 %d 回合",
		silo.MayorLegitimacy(), silo.Rebellion, silo.Authority.W, silo.Authority.PoliceObedience,
		silo.TopStance.Guarantee, OrderPressure(silo), silo.Negotiation.MayorDisposition, silo.Negotiation.StalledRounds)
}

// NegotiationRoundResult 一个谈判回合的结构化结果 (服务层转叙事)。
// 指针字段为 nil 表示本回合没有发生对应决策。
type NegotiationRoundResult struct {
	Kind                 string
	MayorPath            string
	MayorWeights         map[string]float64
	MayorFactors         string
	SuppressBrokeChannel bool   // 镇压打断了信道
	ConcedeOfferMade     bool   // 市长开出了妥协提案
	RebelAccept          *bool  // 叛乱方接受决策 (仅市长主动让步时发生)
	RebelWeights         map[string]float64
	RebelPosture         string // 叛乱方姿态 (仅市长路径为谈判时抽取)
	RebelPostureWeights  map[string]float64
	CompromiseEstablished bool
	ITReaction           string
	ITWeights            map[string]float64
	ITExposedChannel     bool // IT 破坏导致信道曝光
}

// NegotiationRound 执行一个完整回合并施加状态改变。
//
// 固定取样顺序 (确定性纪律)：
//
//	市长路径 → [镇压: 信道断裂抽取 | 妥协: 叛乱方接受抽取 | 谈判: 叛乱方姿态抽取]
//	          → IT 反应 → [破坏: 信道曝光抽取]
//
// 未激活 (信道未开/已崩塌/已妥协) 的堡不进入本函数——调用方负责守卫，
// 保证未开信道的运行不消耗任何 RNG。
func NegotiationRound(silo *model.Silo, world *model.World, kind string) *NegotiationRoundResult {
	if silo == nil || !silo.Negotiation.NegotiationActive() {
		return nil
	}
	negotiation := &silo.Negotiation
	result := &NegotiationRoundResult{Kind: kind}
	year, month := silo.CurrentYear, silo.CurrentMonth

	// ---- 1) 市长路径决策 ----
	factors := NegotiationFactorSummary(silo)
	pathWeights := MayorPathWeights(silo, world, kind)
	path := DrawMayorPath(pathWeights)
	negotiation.MayorPath = path
	negotiation.Rounds++
	result.MayorPath = path
	result.MayorWeights = NormalizeMayorPath(pathWeights)
	result.MayorFactors = factors

	pathOutcome := ""
	switch path {
	case model.NegotiationPathSuppress:
		silo.Rebellion = clampUnit(silo.Rebellion + negotiationSuppressRebellionDelta)
		addPanicAllProfessions(silo, negotiationSuppressPanicDelta)
		silo.SetMayorLegitimacy(silo.MayorLegitimacy() + negotiationSuppressLegitimacyDelta)
		addTopStanceGuarantee(silo, negotiationSuppressGuaranteeDelta)
		negotiation.SuppressRounds++
		negotiation.StalledRounds = 0
		negotiation.MayorLean = clampUnit(negotiation.MayorLean + negotiationLeanSuppress)
		// 镇压背信：叛乱方在桌条款撤回。
		if negotiation.OfferOnTable && negotiation.OfferedBy == "rebels" {
			negotiation.OfferOnTable = false
			negotiation.OfferedBy = ""
			negotiation.OfferTerms = ""
		}
		if DrawFloat64() < negotiationSuppressBreakChance {
			negotiation.ChannelOpen = false
			negotiation.Closed = true
			negotiation.CloseReason = model.NegotiationClosedCrackdown
			result.SuppressBrokeChannel = true
			pathOutcome = "镇压打断了信道"
		} else {
			pathOutcome = "镇压后信道勉强维持"
		}
	case model.NegotiationPathTalk:
		negotiation.StalledRounds++
		negotiation.MayorLean = clampUnit(negotiation.MayorLean + negotiationLeanDrift(negotiation.MayorDisposition))
		pathOutcome = "保持通气，无决议"
	case model.NegotiationPathConcede:
		negotiation.MayorLean = clampUnit(negotiation.MayorLean + negotiationLeanConcede)
		if negotiation.OfferOnTable && negotiation.OfferedBy == "rebels" {
			// 市长接受叛乱方已在桌上的条款：提议方不否决自己的提案，
			// 妥协直接成立。
			establishCompromise(silo, world)
			result.CompromiseEstablished = true
			pathOutcome = "市长接受了叛乱方的条款，妥协成立"
		} else {
			// 市长主动让步：开出提案，随后由叛乱方带权决定是否接受。
			negotiation.OfferOnTable = true
			negotiation.OfferedBy = "mayor"
			negotiation.OfferTerms = "赦免名单、职能保留、不清算求生者"
			result.ConcedeOfferMade = true
			acceptWeights := RebelAcceptWeights(silo)
			accepted := DrawRebelAccept(acceptWeights)
			result.RebelAccept = &accepted
			result.RebelWeights = NormalizeRebelAccept(acceptWeights)
			if accepted {
				establishCompromise(silo, world)
				result.CompromiseEstablished = true
				pathOutcome = "市长开出提案，叛乱方接受，妥协成立"
				negotiation.AppendLogEntry(model.NegotiationLogEntry{
					Year: year, Month: month, Decider: "rebels", Decision: "accept",
					Weights: result.RebelWeights, Factors: factors,
					Outcome: "接受市长妥协提案",
				})
			} else {
				negotiation.OfferOnTable = false
				negotiation.OfferedBy = ""
				negotiation.OfferTerms = ""
				pathOutcome = "市长开出提案，叛乱方拒绝，提案作废"
				negotiation.AppendLogEntry(model.NegotiationLogEntry{
					Year: year, Month: month, Decider: "rebels", Decision: "reject",
					Weights: result.RebelWeights, Factors: factors,
					Outcome: "拒绝市长妥协提案",
				})
			}
		}
	}
	negotiation.AppendLogEntry(model.NegotiationLogEntry{
		Year: year, Month: month, Decider: "mayor", Decision: path,
		Weights: result.MayorWeights, Factors: factors, Outcome: pathOutcome,
	})

	// ---- 2) 叛乱方姿态 (市长路径为谈判且回合仍活跃时) ----
	if path == model.NegotiationPathTalk && negotiation.NegotiationActive() {
		postureWeights := RebelPostureWeights(silo)
		posture := DrawRebelPosture(postureWeights)
		negotiation.RebelPosture = posture
		result.RebelPosture = posture
		result.RebelPostureWeights = NormalizeRebelPosture(postureWeights)
		postureOutcome := ""
		switch posture {
		case model.NegotiationPosturePress:
			silo.Rebellion = clampUnit(silo.Rebellion + negotiationPressRebellionDelta)
			postureOutcome = "叛乱方施压"
		case model.NegotiationPostureProbe:
			negotiation.MayorLean = clampUnit(negotiation.MayorLean + negotiationProbeLeanDelta)
			postureOutcome = "叛乱方试探市长倾向"
		default:
			postureOutcome = "叛乱方酝酿递出条款"
		}
		negotiation.AppendLogEntry(model.NegotiationLogEntry{
			Year: year, Month: month, Decider: "rebels", Decision: posture,
			Weights: result.RebelPostureWeights, Factors: factors, Outcome: postureOutcome,
		})
	}

	// ---- 3) IT 反应 (回合仍活跃时) ----
	if negotiation.NegotiationActive() {
		itWeights := ITReactionWeights(silo, world)
		reaction := DrawITReaction(itWeights)
		result.ITReaction = reaction
		result.ITWeights = NormalizeITReaction(itWeights)
		itOutcome := ""
		switch reaction {
		case model.NegotiationITUndermine:
			addTopStanceGuarantee(silo, negotiationUndermineGuaranteeDelta)
			if DrawFloat64() < negotiationExposureChance {
				negotiation.ChannelOpen = false
				negotiation.Closed = true
				negotiation.CloseReason = model.NegotiationClosedITExposure
				addPanicAllProfessions(silo, negotiationExposurePanicDelta)
				silo.Rebellion = clampUnit(silo.Rebellion + negotiationExposureRebellionDelta)
				result.ITExposedChannel = true
				itOutcome = "IT 破坏导致信道曝光"
			} else {
				itOutcome = "IT 暗中破坏，保障可信度受损"
			}
		case model.NegotiationITStandAside:
			itOutcome = "IT 不干涉"
		default:
			itOutcome = "IT 观望"
		}
		negotiation.AppendLogEntry(model.NegotiationLogEntry{
			Year: year, Month: month, Decider: "it", Decision: reaction,
			Weights: result.ITWeights, Factors: factors, Outcome: itOutcome,
		})
	}

	return result
}

// establishCompromise 妥协成立的即时效果 (18 §3.6)：保障可信度增量、
// 诉求部分兑现 (叛乱回落但组织能力不受损)、立场重估 (保障输入已抬升)。
// 妥协不发放胜利：只改变立场权重输入。
func establishCompromise(silo *model.Silo, world *model.World) {
	negotiation := &silo.Negotiation
	negotiation.CompromiseEstablished = true
	negotiation.CompromiseYear = silo.CurrentYear
	negotiation.CompromiseMonth = silo.CurrentMonth
	addTopStanceGuarantee(silo, negotiationCompromiseGuaranteeDelta)
	silo.Rebellion = clampUnit(silo.Rebellion + negotiationCompromiseRebellionDelta)
	ReevalTopStance(silo, world, "both")
}

// PlaceRebelTerms 叛乱方把妥协条款摆上桌面 (18 §3.2：递条款回合的前置)。
// 条款在桌时市长可以一直不表态 (拖延是合法结果)；市长让步即接受条款。
// 信道不活跃或已有提案在桌时幂等返回 false。
func PlaceRebelTerms(silo *model.Silo) bool {
	if silo == nil || !silo.Negotiation.NegotiationActive() {
		return false
	}
	negotiation := &silo.Negotiation
	if negotiation.OfferOnTable {
		return false
	}
	negotiation.OfferOnTable = true
	negotiation.OfferedBy = "rebels"
	negotiation.OfferTerms = "赦免名单、职能保留、不清算求生者"
	negotiation.AppendLogEntry(model.NegotiationLogEntry{
		Year: silo.CurrentYear, Month: silo.CurrentMonth, Decider: "rebels", Decision: "offer_terms",
		Factors: NegotiationFactorSummary(silo),
		Outcome: "叛乱方条款摆上桌面：" + negotiation.OfferTerms,
	})
	return true
}

// NegotiationOpenResult 开信道的结构化结果。
type NegotiationOpenResult struct {
	Disposition string
	AlreadyOpen bool
}

// OpenNegotiationChannel 开通谈判信道并一次性抽取市长性情 (18 §2)。
// 已开通或已崩塌的信道不重复开通 (幂等)。性情抽取走引擎单一 RNG 流。
func OpenNegotiationChannel(silo *model.Silo) *NegotiationOpenResult {
	if silo == nil {
		return nil
	}
	negotiation := &silo.Negotiation
	if negotiation.ChannelOpen || negotiation.Closed {
		return &NegotiationOpenResult{AlreadyOpen: true, Disposition: negotiation.MayorDisposition}
	}
	negotiation.Initialized = true
	negotiation.ChannelOpen = true
	negotiation.OpenedYear = silo.CurrentYear
	negotiation.OpenedMonth = silo.CurrentMonth
	negotiation.MayorDisposition = DrawMayorDisposition()
	negotiation.MayorLean = initialNegotiationLean(negotiation.MayorDisposition)
	negotiation.AppendLogEntry(model.NegotiationLogEntry{
		Year: silo.CurrentYear, Month: silo.CurrentMonth, Decider: "channel", Decision: "opened",
		Factors: NegotiationFactorSummary(silo),
		Outcome: fmt.Sprintf("信道建立，市长性情：%s", negotiation.MayorDisposition),
	})
	return &NegotiationOpenResult{Disposition: negotiation.MayorDisposition}
}

// addPanicAllProfessions 全部门恐慌值增量 (夹 [0,1])。
func addPanicAllProfessions(silo *model.Silo, delta float64) {
	if silo == nil {
		return
	}
	for i := range silo.Professions {
		silo.Professions[i].PanicValue = clampUnit(silo.Professions[i].PanicValue + delta)
	}
}

// negotiationLeanDrift 谈判拖延期间市长倾向的性情漂移 (确定性，不抽签)。
func negotiationLeanDrift(disposition string) float64 {
	switch disposition {
	case model.MayorDispositionHardliner:
		return -0.01 // TODO(balance)
	case model.MayorDispositionSurvivor:
		return 0.03 // TODO(balance)
	default:
		return 0.02 // TODO(balance)
	}
}

// initialNegotiationLean 开信道时市长倾向的初值 (按性情)。TODO(balance)
func initialNegotiationLean(disposition string) float64 {
	switch disposition {
	case model.MayorDispositionHardliner:
		return negotiationLeanHardliner
	case model.MayorDispositionSurvivor:
		return negotiationLeanSurvivor
	default:
		return negotiationLeanPragmatist
	}
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

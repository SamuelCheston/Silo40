package engine

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// 逃跑筹备的反制层 (20 分册修订、ADR-0014)。
//
// 设计纪律：
//   - 审计没收已整体移除：它造成反向梯度 (准备越重、暴露越高、被没收越多，
//     重准备反而更吃亏)。囤积账本 (防护服/补给/材料) 永不被反制扣减。
//   - 组织怀疑度只由组织行为驱动：动员范围 (演练/预案功能位/照护与生产
//     组织) 与通信频次规模 (外溢信道/保障磋商)。囤积不抬高怀疑度——
//     准备本身不惹眼，惹眼的是组织行为。
//   - 怀疑度到门槛后 IT 必须做选择：镇压 / 蒙住 1 号 / 不作为。选择是
//     带权重的随机裁量 (ADR-0007 同构)，不是阈值 if-else；权重因子可归因，
//     决策日志留痕。
//   - 任何机制都不把玩家困死：全部后果是成本与风险的上浮 (戒备/恐慌/
//     safeguard 风险)，不直接判负、不没收、不永久锁定；玩家永远还有路走。
//   - 随机源走引擎单一 RNG 流：未产生组织怀疑度的运行不进入任何取样，
//     RNG 消耗逐位不变 (同谈判层纪律)。

// 怀疑度累积与衰减的全部数值为初始校准值。TODO(balance)
const (
	// escapeResponseSuspicionDecay 月度衰减：组织行为停了，风声也会淡
	// (与信念淡忘 0.01/月同构，17 §3 修复 A)。
	escapeResponseSuspicionDecay = 0.01
)

// AddEscapeOrgSuspicion 组织行为累积怀疑度 (内容效果 escape_org_suspicion_delta
// 的唯一写入口)。夹到 [0,1]；首次累积初始化响应层。
func AddEscapeOrgSuspicion(silo *model.Silo, delta float64) {
	if silo == nil || delta <= 0 {
		return
	}
	state := &silo.EscapeResponse
	state.Initialized = true
	state.OrgSuspicion = clampUnit(state.OrgSuspicion + delta)
}

// DecayEscapeOrgSuspicion 月度衰减 (月度响应回合调用)。未初始化的运行
// 逐位不变。
func DecayEscapeOrgSuspicion(silo *model.Silo) {
	if silo == nil || !silo.EscapeResponse.Initialized {
		return
	}
	silo.EscapeResponse.OrgSuspicion = clampUnit(silo.EscapeResponse.OrgSuspicion - escapeResponseSuspicionDecay)
}

// IT 响应决策的全部数值为初始校准值。TODO(balance)
const (
	// escapeResponseDecisionThreshold 决策点火门槛：怀疑度到此程度，
	// IT 必须做选择 (选什么是带权裁量，不是阈值 if-else)。
	escapeResponseDecisionThreshold = 0.35
	// escapeResponseCooldownMonths 决策冷却：响应后风声暂歇，层需要
	// 重新积累才会再次点火。
	escapeResponseCooldownMonths = 4
	// escapeResponseDiscretionFloor 自由裁量地板 (同谈判层口径)：任何
	// 状态下每个档位都保留不可压没的最小权重——IT 永远保留镇压/蒙蔽/
	// 不作为三种裁量空间。
	escapeResponseDiscretionFloor = 0.05

	// 镇压效果：只压组织，不碰囤积。
	escapeResponseSuppressSuspicionDelta = -0.25 // 组织暂时散去，风声被消费
	escapeResponseSuppressAlertness      = 0.10  // Judicial/IT 部门戒备
	escapeResponseSuppressPanic          = 0.02  // 全部门恐慌 (同既有征兆事件量级)

	// 蒙蔽维持：占 IT 资源，暴露风险随月上浮。
	escapeResponseCoverCostROp    = 5.0  // 月度基础成本 (经弱势成本曲线)
	escapeResponseCoverLeakBase   = 0.06 // 当月暴露概率基数
	escapeResponseCoverLeakGrowth = 0.03 // 暴露概率逐月增量 (蒙得越久越危险)
	escapeResponseCoverLeakCap    = 0.50
	escapeResponseCoverCollapseRS = 0.08 // 蒙蔽崩塌：safeguard 风险上浮

	// 不作为后果：异常悬而未决，safeguard 风险上浮。全程预算 ≤0.25：
	// 决策冷却 4 月 → 24 月筹备窗内至多 6 次决策，6×0.03 = 0.18，叠加
	// 一次蒙蔽崩塌 (+0.08) 仍在预算内——本通道单独永远到不了硬护栏 1.0。
	escapeResponseIgnoreRS = 0.03
)

// EscapeResponseWeights 一次决策的三档权重 (未归一)。
type EscapeResponseWeights struct {
	Suppress float64
	Conceal  float64
	Ignore   float64
}

// clamp 把权重压回非负。
func (w EscapeResponseWeights) clamp() EscapeResponseWeights {
	if w.Suppress < 0 {
		w.Suppress = 0
	}
	if w.Conceal < 0 {
		w.Conceal = 0
	}
	if w.Ignore < 0 {
		w.Ignore = 0
	}
	return w
}

// floor 自由裁量地板：每个档位保留不可压没的最小权重。
func (w EscapeResponseWeights) floor() EscapeResponseWeights {
	w = w.clamp()
	if w.Suppress < escapeResponseDiscretionFloor {
		w.Suppress = escapeResponseDiscretionFloor
	}
	if w.Conceal < escapeResponseDiscretionFloor {
		w.Conceal = escapeResponseDiscretionFloor
	}
	if w.Ignore < escapeResponseDiscretionFloor {
		w.Ignore = escapeResponseDiscretionFloor
	}
	return w
}

// NormalizeEscapeResponseWeights 归一化权重供决策日志复盘 (三档之和为 1)。
func NormalizeEscapeResponseWeights(w EscapeResponseWeights) map[string]float64 {
	w = w.clamp()
	total := w.Suppress + w.Conceal + w.Ignore
	if total <= 0 {
		total = 1
	}
	return map[string]float64{
		model.EscapeResponseSuppress: w.Suppress / total,
		model.EscapeResponseConceal:  w.Conceal / total,
		model.EscapeResponseIgnore:   w.Ignore / total,
	}
}

// EscapeResponseDecisionWeights IT 响应的权重因子。全部因子可归因：决策日志记录
// 决策时刻的因子读数摘要。权重读状态但不被状态锁死——不存在"怀疑度 ≥ X
// → 必须镇压"。
//
// 因子 (各自 TODO(balance))：
//   - 残余力量 W：高 → 有执行能力镇压；低 → 收手自保 (不作为)。
//   - 怀疑度深度：动向越大越难蒙住，镇压冲动越强。
//   - 资源预算 R_op：蒙蔽要按月付费，穷得付不起时蒙蔽权重下降。
//   - safeguard 压力：1 号已经在注视时，IT 倾向尽快了结异常。
//   - 保障可信度：叛乱方保障越可信，镇压意愿越低。
//   - IT 立场：阻挠者倾向镇压；配合者倾向蒙蔽 (放人走但瞒住 1 号)。
func EscapeResponseDecisionWeights(silo *model.Silo) EscapeResponseWeights {
	w := EscapeResponseWeights{Suppress: 0.38, Conceal: 0.34, Ignore: 0.28} // TODO(balance)
	wResidual := clampUnit(silo.Authority.W / 100)
	suspicion := silo.EscapeResponse.OrgSuspicion
	guarantee := silo.TopStance.Guarantee

	// 残余力量：W 高 → 转向慢、镇压可行；W 低 → 弱势的权力方收手是自保。
	w.Suppress += 0.30 * wResidual          // TODO(balance)
	w.Ignore += 0.20 * (1 - wResidual)      // TODO(balance)

	// 怀疑度深度：动向越大越难蒙住。
	w.Suppress += 0.25 * suspicion // TODO(balance)
	w.Conceal -= 0.20 * suspicion  // TODO(balance)

	// 资源预算：蒙蔽按月占 R_op，付不起时这条路的可信度下降。
	if silo.ItAuthority.ROp < escapeResponseCoverCostROp*3 { // TODO(balance)
		w.Conceal -= 0.25 // TODO(balance)
	}

	// safeguard 压力：1 号已在注视，异常拖得越久越危险。
	rs := clampUnit(silo.SafeguardRisk)
	w.Suppress += 0.30 * rs // TODO(balance)
	w.Ignore -= 0.20 * rs   // TODO(balance)

	// 保障可信度：求生算计压低镇压意愿。
	w.Suppress -= 0.15 * guarantee // TODO(balance)
	w.Conceal += 0.10 * guarantee  // TODO(balance)

	// IT 立场：阻挠 → 镇压；配合 → 放人走但把 1 号蒙住。
	switch silo.TopStance.ITStance() {
	case model.TopStanceBlock:
		w.Suppress += 0.20 // TODO(balance)
	case model.TopStanceCooperate:
		w.Suppress -= 0.30 // TODO(balance)
		w.Conceal += 0.25  // TODO(balance)
	}
	return w.floor()
}

// escapeResponseFactors 决策时刻的因子读数摘要 (决策日志归因)。
func escapeResponseFactors(silo *model.Silo) string {
	return fmt.Sprintf("suspicion=%.2f W=%.0f R_op=%.0f R_s=%.2f it=%s guarantee=%.2f",
		silo.EscapeResponse.OrgSuspicion, silo.Authority.W, silo.ItAuthority.ROp,
		silo.SafeguardRisk, silo.TopStance.ITStance(), silo.TopStance.Guarantee)
}

// drawEscapeResponse 按权重在引擎单一 RNG 流上取样一次。遍历顺序固定：
// 镇压 → 蒙蔽 → 不作为。
func drawEscapeResponse(w EscapeResponseWeights) string {
	w = w.clamp()
	total := w.Suppress + w.Conceal + w.Ignore
	if total <= 0 {
		return model.EscapeResponseIgnore
	}
	r := DrawFloat64() * total
	if r < w.Suppress {
		return model.EscapeResponseSuppress
	}
	if r < w.Suppress+w.Conceal {
		return model.EscapeResponseConceal
	}
	return model.EscapeResponseIgnore
}

// EscapeResponseDecisionResult 一次 IT 响应决策的结构化结果 (服务层写
// 决策日志与叙事)。
type EscapeResponseDecisionResult struct {
	Decision string
	Weights  map[string]float64
	Factors  string
}

// escapeResponseCooldownElapsed 决策冷却是否已过 (未做过决策 = 已过)。
func escapeResponseCooldownElapsed(silo *model.Silo) bool {
	state := &silo.EscapeResponse
	if state.LastDecisionYear == 0 {
		return true
	}
	now := silo.CurrentYear*12 + silo.CurrentMonth
	last := state.LastDecisionYear*12 + state.LastDecisionMonth
	return now-last >= escapeResponseCooldownMonths
}

// RunEscapeResponseDecision 决策点火与效果施加。怀疑度未到门槛或冷却
// 未过时返回 nil (不取样、不消耗 RNG)。
func RunEscapeResponseDecision(silo *model.Silo) *EscapeResponseDecisionResult {
	if silo == nil {
		return nil
	}
	state := &silo.EscapeResponse
	if !state.Initialized || state.OrgSuspicion < escapeResponseDecisionThreshold {
		return nil
	}
	if state.CoverActive || !escapeResponseCooldownElapsed(silo) {
		return nil
	}

	w := EscapeResponseDecisionWeights(silo)
	decision := drawEscapeResponse(w)
	factors := escapeResponseFactors(silo)

	switch decision {
	case model.EscapeResponseSuppress:
		// 镇压只压组织：驱散集结、抬戒备与恐慌。结构性不变式——
		// 囤积账本 (防护服/补给/材料) 与外溢信道永不被扣减。
		state.OrgSuspicion = clampUnit(state.OrgSuspicion + escapeResponseSuppressSuspicionDelta)
		for i := range silo.Professions {
			prof := &silo.Professions[i]
			if prof.Name == "Judicial" || prof.Name == "IT" {
				prof.Alertness = clampUnit(prof.Alertness + escapeResponseSuppressAlertness)
			}
			prof.PanicValue = clampUnit(prof.PanicValue + escapeResponseSuppressPanic)
		}
	case model.EscapeResponseConceal:
		// 蒙蔽：不镇压，自己把 1 号蒙住。组织继续活动 (怀疑度不重置)，
		// IT 按月付资源，暴露风险随月上浮。
		state.CoverActive = true
		state.CoverMonths = 0
	case model.EscapeResponseIgnore:
		// 不作为是 IT 的自由裁量，但异常悬而未决：1 号可能自己注意到。
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+escapeResponseIgnoreRS)
	}
	state.LastDecisionYear = silo.CurrentYear
	state.LastDecisionMonth = silo.CurrentMonth

	return &EscapeResponseDecisionResult{
		Decision: decision,
		Weights:  NormalizeEscapeResponseWeights(w),
		Factors:  factors,
	}
}

// EscapeResponseCoverMonth 蒙蔽维持推进一个月：支付 R_op (弱势成本曲线)，
// 抽取暴露。返回是否崩塌 (付不起或暴露)。崩塌时 safeguard 风险上浮——
// 后果落在"日后更贵、更窄"，不是当场判负。
func EscapeResponseCoverMonth(silo *model.Silo) bool {
	if silo == nil {
		return false
	}
	state := &silo.EscapeResponse
	if !state.CoverActive {
		return false
	}
	state.CoverMonths++

	cost := ItCostCurve(escapeResponseCoverCostROp, silo.Authority.W)
	if silo.ItAuthority.ROp < cost {
		// 付不起：蒙蔽维持不住，异常暴露。
		state.CoverActive = false
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+escapeResponseCoverCollapseRS)
		return true
	}
	silo.ItAuthority.ROp -= cost

	p := escapeResponseCoverLeakBase + escapeResponseCoverLeakGrowth*float64(state.CoverMonths-1)
	if p > escapeResponseCoverLeakCap {
		p = escapeResponseCoverLeakCap
	}
	if DrawFloat64() < p {
		state.CoverActive = false
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+escapeResponseCoverCollapseRS)
		return true
	}
	return false
}

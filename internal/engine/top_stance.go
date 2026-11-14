package engine

import (
	"silo40/internal/model"
)

// 顶层立场的带权重随机决策 (16 §3.2 裁决 D/E/F、ADR-0008)。
//
// 设计纪律：
//   - 权重读状态，但不被状态锁死：市长正统性、叛乱、W、Silo 1 状态、保障
//     可信度都是权重的输入，不是强制阈值——不存在"叛乱 ≥ X → 必须妥协"。
//   - 随机源走引擎单一 RNG 流 (地堡种子)：同种子 + 同动作序列 → 同样的
//     立场轨迹。重估只在内容事件点火时发生 (开窗/磋商/出发)，不在月度
//     tick 里取样，因此不改变既有基线运行的 RNG 消耗顺序。
//   - 立场可反复重估：磋商是持续过程，妥协时点是权重与随机种子的产物；
//     重估不保证单调 (W 残余时存在玉石俱焚回摆到阻挠的概率)。
//   - 立场是内部状态：不作交互选项、不作显式判定闸；仲裁不读立场，
//     只读出发结算产出的逃出人数。

// stanceWeights 一次重估的三档权重 (未归一)。
type stanceWeights struct {
	block     float64
	nonblock  float64
	cooperate float64
}

// clamp 把权重压回非负；总权重为零时由 drawStance 兜底为阻挠。
func (w stanceWeights) clamp() stanceWeights {
	if w.block < 0 {
		w.block = 0
	}
	if w.nonblock < 0 {
		w.nonblock = 0
	}
	if w.cooperate < 0 {
		w.cooperate = 0
	}
	return w
}

// ReevalTopStance 对 actor ("mayor" / "it" / 其它 = both) 重估立场。
// both 时固定先市长后 IT 取样 (RNG 消耗顺序固定，确定性)。
func ReevalTopStance(silo *model.Silo, world *model.World, actor string) {
	if silo == nil {
		return
	}
	silo.TopStance.Initialized = true
	switch actor {
	case "mayor":
		silo.TopStance.Mayor = drawStance(mayorStanceWeights(silo, world))
	case "it":
		silo.TopStance.IT = drawStance(itStanceWeights(silo, world))
	default:
		silo.TopStance.Mayor = drawStance(mayorStanceWeights(silo, world))
		silo.TopStance.IT = drawStance(itStanceWeights(silo, world))
	}
	silo.TopStance.Reevaluations++
}

// drawStance 按权重在引擎单一 RNG 流上取样一次，返回立场档位。
// 遍历顺序固定：阻挠 → 不阻挠 → 配合。
func drawStance(w stanceWeights) string {
	w = w.clamp()
	total := w.block + w.nonblock + w.cooperate
	if total <= 0 {
		return model.TopStanceBlock
	}
	r := DrawFloat64() * total
	if r < w.block {
		return model.TopStanceBlock
	}
	if r < w.block+w.nonblock {
		return model.TopStanceNonBlock
	}
	return model.TopStanceCooperate
}

// silo1CommandSourceGone 报告 Silo 1 的命令源是否枯竭：覆灭或终止信道被干扰
// (既有世界事实，不新造状态)。命令源枯竭时，继续阻挠的组织前提瓦解。
func silo1CommandSourceGone(silo *model.Silo, world *model.World) bool {
	if silo != nil && silo.Silo1Destroyed {
		return true
	}
	if world != nil && world.Flags[model.WorldFlagNetworkJammed] {
		return true
	}
	return false
}

// mayorStanceWeights 市长立场权重。输入 (均 16 §3.2)：
//   - 个人正统性 (MayorLegitimacy，既有)：越高越能维持镇压；
//   - 叛乱水平 (既有)：侵蚀镇压可行性，妥协权重上升 (裁决：叛乱降低正统性
//     接既有状态，这里直接读叛乱值作权重输入)；
//   - 保障可信度：赦免/职能保留承诺使不阻挠/配合可信；
//   - Silo 1 状态：命令源枯竭时阻挠权重下降。
//
// 全部系数为初始校准值。TODO(balance)
func mayorStanceWeights(silo *model.Silo, world *model.World) stanceWeights {
	w := stanceWeights{block: 0.55, nonblock: 0.30, cooperate: 0.15} // TODO(balance)
	legitimacy := silo.MayorLegitimacy()
	rebellion := silo.Rebellion
	guarantee := silo.TopStance.Guarantee

	// 正统性：高 → 镇压仍可行 (阻挠升)；低 → 妥协是现实选项。
	w.block += 0.30 * legitimacy          // TODO(balance)
	w.nonblock += 0.20 * (1 - legitimacy) // TODO(balance)
	w.cooperate += 0.10 * (1 - legitimacy)

	// 叛乱：镇压可行性下降。读权重，不设强制阈值。
	w.block -= 0.35 * rebellion     // TODO(balance)
	w.nonblock += 0.25 * rebellion  // TODO(balance)
	w.cooperate += 0.15 * rebellion // TODO(balance)

	// 保障可信度：承诺可信才敢放下阻挠。
	w.block -= 0.15 * guarantee     // TODO(balance)
	w.nonblock += 0.25 * guarantee  // TODO(balance)
	w.cooperate += 0.30 * guarantee // TODO(balance)

	if silo1CommandSourceGone(silo, world) {
		w.block -= 0.15     // TODO(balance)
		w.nonblock += 0.10  // TODO(balance)
		w.cooperate += 0.05 // TODO(balance)
	}
	return w.clamp()
}

// itStanceWeights IT 立场权重。叛乱胜利拆掉 IT 维持阻挠的三根支柱
// (执法合法性失效、命令源枯竭、生存算计)，权重输入对应：
//   - 残余力量 W (既有)：越高、Protocol 执行核心越完整，转向越慢，
//     且保留玉石俱焚 (回摆阻挠) 的概率；
//   - 叛乱水平＋警察服从转移：执法装置事实抗命/瘫痪 → 阻挠在物理上不可行；
//   - 保障可信度：不阻挠乃至配合是 IT 的求生路径 (握气闸/库/盲区换安全)；
//   - Silo 1 状态：专线仍在下令 → 阻挠权重被抬高；静默/覆灭 → 瓦解。
//
// 全部系数为初始校准值。TODO(balance)
func itStanceWeights(silo *model.Silo, world *model.World) stanceWeights {
	w := stanceWeights{block: 0.55, nonblock: 0.28, cooperate: 0.17} // TODO(balance)
	wResidual := silo.Authority.W
	rebellion := silo.Rebellion
	guarantee := silo.TopStance.Guarantee
	obedience := silo.Authority.PoliceObedience

	// 残余力量：W 高 → 转向慢、玉石俱焚概率保留。
	w.block += 0.35 * wResidual    // TODO(balance)
	w.nonblock -= 0.15 * wResidual // TODO(balance)

	// 叛乱胜利：镇压支柱坍塌。
	w.block -= 0.35 * rebellion     // TODO(balance)
	w.nonblock += 0.20 * rebellion  // TODO(balance)
	w.cooperate += 0.15 * rebellion // TODO(balance)

	// 执法合法性失效：警察服从已转移时，继续阻挠失去执行手段。
	if obedience < 0.5 { // TODO(balance)
		w.block -= 0.20     // TODO(balance)
		w.nonblock += 0.12  // TODO(balance)
		w.cooperate += 0.08 // TODO(balance)
	}

	// 生存算计：可信保障把"握手段换安全"变成合作。
	w.block -= 0.15 * guarantee     // TODO(balance)
	w.nonblock += 0.20 * guarantee  // TODO(balance)
	w.cooperate += 0.35 * guarantee // TODO(balance)

	// 命令源枯竭：专线静默 / Silo 1 覆灭 → "为谁镇压"消失。
	if silo1CommandSourceGone(silo, world) {
		w.block -= 0.25     // TODO(balance)
		w.nonblock += 0.15  // TODO(balance)
		w.cooperate += 0.10 // TODO(balance)
	}
	return w.clamp()
}

// addTopStanceGuarantee 增减保障可信度并夹到 [0,1]。
func addTopStanceGuarantee(silo *model.Silo, delta float64) {
	if silo == nil {
		return
	}
	silo.TopStance.Initialized = true
	g := silo.TopStance.Guarantee + delta
	if g < 0 {
		g = 0
	} else if g > 1 {
		g = 1
	}
	silo.TopStance.Guarantee = g
}

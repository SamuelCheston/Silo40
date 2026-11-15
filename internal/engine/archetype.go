package engine

import (
	"math"

	"silo40/internal/model"
)

// 驻地原型 (19-opening-variability §3)：重开一局的差异落在玩家可感知的玩法
// 面上，而不是文案。三档各强化且只强化一个核心压力源——IT 控制 (磐石)、
// 秩序波动 (裂隙)、资源维生 (荒年)，其余两轴保持基线，档间差异可归因、可
// 单独测试。
//
// 铁律 (19 §5 / ADR-0011)：任何原型旋钮都不得修改初始思潮值；旋钮只许落
// 在门槛、动作可用性、资源、控制强度这些可归因的量上。初始 W 与其衰减速
// 率是门控耦合旋钮，档间偏移必须满足 19 §6.2 的可达性约束 (a)(b)(c)。
const (
	ArchetypeBedrock = "bedrock" // 磐石：IT 控制强化
	ArchetypeRift    = "rift"    // 裂隙：控制松弛、秩序波动放大
	ArchetypeFamine  = "famine"  // 荒年：资源匮乏
)

// ArchetypeTableVersion 号数→档位查表的内容版本 (19 §3.3"查表内容进配置并
// 版本化")：配置改版不追溯改写旧档——原型是建档时写入存档的既成事实。
const ArchetypeTableVersion = 1

// ArchetypeConfig 单档原型的全部旋钮。零值语义：每个旋钮的基线值 (1.0 /
// 0) 表示"不动"；空原型 (silo.Archetype == "") 不读取本结构，行为逐位不变。
type ArchetypeConfig struct {
	ID          string
	DisplayName string // 玩家可见档名 (文案层可再议，机制不受影响)
	Description string // 一句话机制说明，进开局面板原生呈现
	// W0Delta 初始幕后威势偏移 (门控耦合旋钮，19 §6.2)：建档时 W = NewGameW +
	// W0Delta。上调延长《公议》(W≤18) 与《人治》(W≤30) 的磨 W 期，偏移幅度
	// 受可达性约束 (a)(b)(c) 限制，逐档标定记录见 archetypeRegistry。
	W0Delta float64
	// WHalfLifeMonths 覆写 W 向平衡点收敛的半衰期 (月)；0 = 沿用基线
	// (DefaultAuthorityConfig.WHalfLifeMonths)。值越大衰减越慢。
	WHalfLifeMonths float64
	// DetectionFactor SAMPLED 怀疑增量系数 (带宽内取值，19 §3.2)：只乘
	// SAMPLED 路径的怀疑结算，不改变检测分类结构。1.0 = 基线。
	DetectionFactor float64
	// ResourceFactor 初始资源水位倍率 (自由旋钮)：建档时一次性乘上
	// Energy/Materials/Supplies。1.0 = 基线。
	ResourceFactor float64
	// CrisisCooldownFactor 危机类可重复事件冷却倍率 (自由旋钮)：<1 = 危机
	// 加密。1.0 = 基线。仅作用于 crisis 组事件，不改变其他事件节奏。
	CrisisCooldownFactor float64
	// PanicRebellionAmp 恐慌抬升与叛乱累积放大系数 (自由旋钮)：只放大正向
	// 增量 (抬升/累积)，不放大回落。1.0 = 基线。
	PanicRebellionAmp float64
}

// archetypeRegistry 三档旋钮。方向由 19 §3.2 确定，量级由各原型验收宿主 sim
// 标定 (19 §6.2)。
var archetypeRegistry = map[string]ArchetypeConfig{
	ArchetypeBedrock: {
		ID:          ArchetypeBedrock,
		DisplayName: "磐石",
		Description: "IT 控制强化：幕后威势更高且退得更慢，探查更容易被抽样怀疑——信息线变贵，磨 W 期更长。",
		// 标定量级 (19 §6.2 (a)(b)(c)，archetype_bedrock_test.go 钉档实测)：
		// - W0 +8 → +10：+10 时三线地平线内可达且磨 W 动作数不增——《公议》
		//   (120 开局) 73/120 月结算、决定性发布仍在谷底带 (W≈39)、磨 W 曝光
		//   7 次不变；《人治》13 次曝光顶压结算 W≈12.7 (门槛 30)；《共证》82
		//   月结算、怀疑峰 0.73。+11 起《共证》种子 2401 的 IT 反击轨迹换相：
		//   高 W 使 ItCostCurve 更便宜、反击更密，怀疑峰跳到 0.95+ (距《除名》
		//   硬线 1.0 仅 0.03–0.05)，故 +10 是保留健康余量的最大偏移。
		// - 半衰期 12 月保留：13/14 月同样经"高 W → 反击便宜"通路把《共证》
		//   怀疑峰推至 0.80/0.95+、《断层》怀疑峰推至 0.78，12 是健康区上限。
		// - 检测 1.25 → 1.5：钉档各线 (含无声探查《共证》与《断层》) 至 2.0
		//   仍全绿——这些策略结构性规避 SAMPLED 探查，检测轴在钉档线上不咬合；
		//   1.5 为未测的探查密集型矩阵组合 (§6.4 职业×种子) 保留 ≥2× 怀疑预算
		//   余量，对裂隙 0.8 的档间比值 1.875 已充分可感知。
		W0Delta:         10,
		WHalfLifeMonths: 12,
		DetectionFactor: 1.5,
	},
	ArchetypeRift: {
		ID:          ArchetypeRift,
		DisplayName: "裂隙",
		Description: "控制松弛、秩序波动放大：幕后威势更低、探查更不易被抽样，但恐慌与叛乱更容易抬升——容易传话，不容易稳住。",
		// 方向不变 (19 §3.2)，量级按逐线可达性收敛 (19 §6.2/§6.3，ADR-0011
		// 塌缩即回炉)：
		// - 波动放大 1.3 → 1.03：1.2 起《共证》(Judicial 无压恐慌杠杆) 与《无人
		//   裁决》塌进恐慌→维生螺旋的《断层》；1.05 起零输入《深闭》输掉与重置
		//   链的同月竞速 (放大把清洁判决链整体提前一个月)，《公议》的叛乱螺旋
		//   也早于约章锁定成熟。1.03 是全部线可达时保留的最大放大。
		// - 检测 0.75 → 0.8：0.75 下《除名》线的抽样怀疑积累 (建连 ×0.75) 赶不
		//   上一号堡覆灭地平线 (130-01)；0.8 时纯建连节奏可达，方向仍 <1。
		// 失败线 (侦察链/断层) 触发率不上陡坡仍为界 (19 §8)。
		W0Delta:           -8,
		DetectionFactor:   0.8,
		PanicRebellionAmp: 1.03,
	},
	ArchetypeFamine: {
		ID:          ArchetypeFamine,
		DisplayName: "荒年",
		Description: "资源匮乏：初始资源水位更低、短缺危机更早且更密——日历与 AP 被维生挤占，留给流通与组织的时间变少。",
		// TODO(balance)：水位 ×0.6 与冷却 ×0.75 为 sim 起点；危机提前量以
		// "首月可感知、不早于玩家首次行动窗口" 为界 (19 §8)。
		ResourceFactor:       0.6,
		CrisisCooldownFactor: 0.75,
	},
}

// ArchetypeByID 返回已注册原型；未注册 (含空串) 返回 false = 无原型。
func ArchetypeByID(id string) (ArchetypeConfig, bool) {
	cfg, ok := archetypeRegistry[id]
	return cfg, ok
}

// stationArchetypeTable 号数 (2–17) → 档位的固定查表 (19 §3.3)：每档 5–6
// 个号，相邻号数不同档 (重开手感)。2/9/17 号分属三档，保证重开对比可见。
var stationArchetypeTable = map[int]string{
	2:  ArchetypeBedrock,
	3:  ArchetypeRift,
	4:  ArchetypeFamine,
	5:  ArchetypeBedrock,
	6:  ArchetypeRift,
	7:  ArchetypeFamine,
	8:  ArchetypeBedrock,
	9:  ArchetypeRift,
	10: ArchetypeBedrock,
	11: ArchetypeFamine,
	12: ArchetypeBedrock,
	13: ArchetypeRift,
	14: ArchetypeFamine,
	15: ArchetypeBedrock,
	16: ArchetypeRift,
	17: ArchetypeFamine,
}

// ArchetypeForStation 号数 → 原型的确定性派生 (独立查表，不消耗游戏内
// RNG)。未知号数返回空串 = 无原型。
func ArchetypeForStation(number int) string {
	return stationArchetypeTable[number]
}

// ApplyArchetypeOpening 在建档时把原型旋钮落到初始状态。只在新局建档调用
// 一次 (原型是既成事实)；空原型直接返回，行为逐位不变。不消耗 RNG。
func ApplyArchetypeOpening(silo *model.Silo) {
	if silo == nil {
		return
	}
	arch, ok := ArchetypeByID(silo.Archetype)
	if !ok {
		return
	}
	cfg := DefaultAuthorityConfig()
	if arch.W0Delta != 0 && silo.Authority.Initialized {
		silo.Authority.W = clamp(cfg.NewGameW+arch.W0Delta, 0, 100)
		// InitializeAuthorityState 已按 NewGameW 算过 CommandIT；W 偏移后按
		// 同一 sigmoid 重算，保持与月度推进同公式。
		s := 1 / (1 + math.Exp(-(silo.Authority.W-cfg.SigmoidMidpoint)/cfg.SigmoidWidth))
		silo.Authority.Judicial.CommandIT = silo.Authority.Judicial.IT * s
	}
	if arch.ResourceFactor > 0 && arch.ResourceFactor != 1 {
		for i := range silo.Resources {
			r := &silo.Resources[i]
			if r.Type == "Energy" || r.Type == "Materials" || r.Type == "Supplies" {
				r.Amount *= arch.ResourceFactor
			}
		}
	}
}

// AuthorityConfigFor 按原型覆写权威配置。空原型/未注册返回逐位不变的默认
// 配置——旧档与基线行为不受影响。
func AuthorityConfigFor(silo *model.Silo) AuthorityConfig {
	cfg := DefaultAuthorityConfig()
	if silo == nil {
		return cfg
	}
	if arch, ok := ArchetypeByID(silo.Archetype); ok && arch.WHalfLifeMonths > 0 {
		cfg.WHalfLifeMonths = arch.WHalfLifeMonths
	}
	return cfg
}

// ArchetypeDetectionFactor SAMPLED 怀疑增量系数：空原型恒为 1 (逐位不变)。
func ArchetypeDetectionFactor(silo *model.Silo) float64 {
	if silo == nil {
		return 1
	}
	if arch, ok := ArchetypeByID(silo.Archetype); ok && arch.DetectionFactor > 0 {
		return arch.DetectionFactor
	}
	return 1
}

// AmplifyOrderDelta 裂隙档放大恐慌抬升与叛乱累积的正向增量；空原型、非正
// 增量 (回落/镇压) 与其余档位逐位不变。metric 仅识别 "panic_value" 与
// "rebellion"，其余度量原样返回。
func AmplifyOrderDelta(silo *model.Silo, metric string, delta float64) float64 {
	if silo == nil || delta <= 0 {
		return delta
	}
	if metric != "panic_value" && metric != "rebellion" {
		return delta
	}
	if arch, ok := ArchetypeByID(silo.Archetype); ok && arch.PanicRebellionAmp > 0 && arch.PanicRebellionAmp != 1 {
		return delta * arch.PanicRebellionAmp
	}
	return delta
}

// CrisisCooldownScale 危机组可重复事件的冷却倍率；空原型恒为 1。
func CrisisCooldownScale(silo *model.Silo) float64 {
	if silo == nil {
		return 1
	}
	if arch, ok := ArchetypeByID(silo.Archetype); ok && arch.CrisisCooldownFactor > 0 && arch.CrisisCooldownFactor != 1 {
		return arch.CrisisCooldownFactor
	}
	return 1
}

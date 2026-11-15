package service

import (
	"fmt"
	"math"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 出发结算点 (16 §3.4 的最小形态，《越岭》链的结算接线)。
//
// 口径 (用户裁决 + ADR-0008)：
//   - 结算只看逃出去了多少人。顶层立场不是判定闸：任一方阻挠时，容量/
//     存活率/追踪把人数自然压到 0，不需要额外判定宣布失败。
//   - 立场不直接换算人数：它塑形准备可得性与过程环境 (防护服/物资/
//     监控/通道)，经容量 D、存活率 S、维存 M、追踪 T 四个侧面落到人数。
//   - 仲裁永不读本结算的输入 (立场/账本)，只读产出的 EscapedGroup 事实；
//     囤积准备不产生任何结局影响，直到出发真的发生 (Layer 1 纪律)。
//
// 本文件是 Layer 2 逃离模型主体的第一段结算的最小实现：堡外月度维持检查、
// 追踪四档判定与仲裁读群体质量仍属 Layer 2，未实装 (依赖标注见 37 号规格)。

// 出发结算阈值。全部为初始校准值。TODO(balance)
const (
	// 容量 D：路线吞吐按接应档位 (未确认路线只保留强行出发的狭窄通道)。
	escapeRouteCapacityConfirmed = 150.0 // 先遣往返确认
	escapeRouteCapacityStocked   = 220.0 // 接应点已囤物资
	escapeRouteCapacityForced    = 30.0  // 未确认/受损路线强行出发
	// 容量 D：防护服覆盖 (路线含暴露段，外界致命)。
	escapeSuitPassagesPerSuit = 1.0 // 单套可掩护通行次数 TODO(balance)
	// 存活率 S。
	escapeBaseSurvival           = 0.90 // S_base
	escapeRouteFactorConfirmed   = 1.00
	escapeRouteFactorUnconfirmed = 0.25 // 未确认/受损：地板值 (近乎全损)
	escapeDrillBase              = 0.70 // 演练系数下限 (0.7 + 0.3×演练水平)
	escapeOrderFactorOrderly     = 1.00 // R-D 有序占用
	escapeOrderFactorRiot        = 0.70 // R-D 全面暴乱
	escapeOrderFactorDefault     = 0.85 // 未作 R-D 选择
	escapeSuitFactorFloor        = 0.30 // 覆盖率 0 时的存活系数下限
	// 维存 M。
	escapeSuppliesPerSurvivor = 1.0  // 人均补给阈值 (人·月)
	escapeCareThreshold       = 0.30 // 照护能力阈值
	escapeOrgThreshold        = 0.50 // 组织留存阈值 (预案完备度)
	// 追踪 T (最小形态：单一阈值 + 折减；四档判定属 Layer 2)。
	escapeTrackingThreshold       = 0.60 // 筹备暴露阈值
	escapeTrackingLoss            = 0.30 // 折减比例
	escapeCooperateTrackingFactor = 0.40 // IT 配合：本地追踪大降
	// 立场塑形系数。
	escapeCooperateSuitFactor  = 2.0  // IT 配合：开防护服库
	escapeCooperateRouteFactor = 1.5  // IT 配合：正式调度提高路线吞吐
	escapeCooperateOrgFactor   = 1.5  // 市长配合：正式调拨与动员
	escapeBlockOrgFactor       = 0.10 // 市长阻挠：秩序侧追缉，动员被压制
	escapeBlockPursuitFactor   = 0.20 // 市长阻挠：追缉对路上幸存的损耗
	// 动员：愿意离开的人口比例。
	escapeWillingBase = 0.4 // TODO(balance)
	// 谈判层 (18 §3.7)：叛乱方不接管市长职位、带组织转向离开时的组织动员加成。
	escapeNegotiationHandsOffOrgBonus = 0.15 // TODO(balance)
)

// escapeSettlement 一次出发结算的审计结果 (呈现与测试用)。
type escapeSettlement struct {
	Capacity  int
	Departed  int
	Survivors int
	Surviving bool
	Fragile   bool
	Notes     []string
}

// settleEscapeDeparture 执行出发结算：读立场＋账本＋路线事实，产出逃出
// 群体事实。只在出发事件 (含 escape_departure_settle 效果) 施加完毕后调用。
func (s *GameService) settleEscapeDeparture(silo *model.Silo, event *engine.GameEvent) *escapeSettlement {
	if silo == nil {
		return nil
	}
	prep := silo.EscapePrep
	mayorStance := silo.TopStance.MayorStance()
	itStance := silo.TopStance.ITStance()
	settle := &escapeSettlement{}

	// ---- 容量 D = min(路线, 组织, 防护服) ----
	// 路线吞吐：接应档位读账本 (先遣链写入)。未确认路线 = 强行出发窄通道。
	routeCap := escapeRouteCapacityForced
	routeFactor := escapeRouteFactorUnconfirmed
	switch prep.ReceptionStage {
	case model.ReceptionConfirmed:
		routeCap, routeFactor = escapeRouteCapacityConfirmed, escapeRouteFactorConfirmed
	case model.ReceptionStocked:
		routeCap, routeFactor = escapeRouteCapacityStocked, escapeRouteFactorConfirmed
	}
	// 组织动员：愿意离开的人口 × 组织系数 (预案＋演练)。
	willingFraction := clampUnit(silo.Rebellion) * (escapeWillingBase + (1-escapeWillingBase)*prep.PlanCompleteness) // TODO(balance)
	willing := int(float64(silo.TotalPopulation) * willingFraction)
	// 防护服覆盖。
	suitCap := float64(prep.SuitsFunctional) * escapeSuitPassagesPerSuit

	// 立场塑形 (阻挠 → 人数自然趋 0；配合 → 带走更多)。
	switch itStance {
	case model.TopStanceBlock:
		// 封闸、气闸即处刑场、防护服库封闭 → 物理通行能力与服体归零。
		routeCap = 0
		suitCap = 0
		settle.Notes = append(settle.Notes, "IT 阻挠：通道封闭、防护服库未开")
	case model.TopStanceCooperate:
		suitCap *= escapeCooperateSuitFactor
		routeCap *= escapeCooperateRouteFactor
		settle.Notes = append(settle.Notes, "IT 配合：防护服库开启、监控被压、正式调度")
	}
	switch mayorStance {
	case model.TopStanceBlock:
		// 离开被定义为叛乱与出逃：秩序侧追缉正当，动员在追缉下坍塌。
		willing = int(float64(willing) * escapeBlockOrgFactor)
		settle.Notes = append(settle.Notes, "市长阻挠：秩序侧追缉，动员被压制")
	case model.TopStanceCooperate:
		willing = int(float64(willing) * escapeCooperateOrgFactor)
		settle.Notes = append(settle.Notes, "市长配合：正式调拨与公开动员")
	}
	orgFactor := 0.4 + 0.3*prep.PlanCompleteness + 0.3*prep.DrillLevel // TODO(balance)
	// 谈判层 (18 §3.7)：妥协成立且叛乱方不接管 → 组织完整转向离开，
	// 动员获得组织加成。谈判只改变"谁在阻碍谁"，不改变结算的读取对象。
	if silo.Negotiation.CompromiseEstablished && silo.Negotiation.Takeover == model.NegotiationHandsOff {
		orgFactor += escapeNegotiationHandsOffOrgBonus // TODO(balance)
		settle.Notes = append(settle.Notes, "叛乱方不接管：组织完整保留，动员获得加成")
	}
	orgCap := float64(willing) * orgFactor

	capacity := int(math.Min(math.Min(routeCap, orgCap), suitCap))
	if capacity > silo.TotalPopulation {
		capacity = silo.TotalPopulation
	}
	departed := capacity // 出发人数 N = min(愿意离开, D)；愿意离开已含于 orgCap
	if departed < 0 {
		departed = 0
	}
	settle.Capacity = capacity
	settle.Departed = departed

	// ---- 存活率 S ----
	survivors := 0
	fragile := false
	suitCoverage := 0.0 // 出发时的防护服覆盖：堡外维持的初始完整性
	if departed > 0 {
		coverage := clampUnit(suitCap / float64(departed))
		suitCoverage = coverage
		suitFactor := escapeSuitFactorFloor + (1-escapeSuitFactorFloor)*coverage // TODO(balance)
		drillFactor := escapeDrillBase + (1-escapeDrillBase)*prep.DrillLevel     // TODO(balance)
		orderFactor := escapeOrderFactorDefault
		if s.contentFactTriggered(silo.ID, "player_actions:escape_window_orderly") {
			orderFactor = escapeOrderFactorOrderly
		} else if s.contentFactTriggered(silo.ID, "player_actions:escape_window_riot") {
			orderFactor = escapeOrderFactorRiot
		}
		survival := escapeBaseSurvival * suitFactor * routeFactor * drillFactor * orderFactor
		survivors = int(math.Round(float64(departed) * clampUnit(survival)))
		if routeFactor < 1 {
			settle.Notes = append(settle.Notes, "路线未确认：存活率走地板值")
		}
		// 市长阻挠：秩序侧追缉正当，路上损耗陡增 (16 §3.2 立场到人数的因果链)。
		if mayorStance == model.TopStanceBlock && survivors > 0 {
			survivors = int(math.Round(float64(survivors) * escapeBlockPursuitFactor))
			fragile = true
			settle.Notes = append(settle.Notes, "秩序侧追缉：路上损耗陡增")
		}
	}

	// ---- 维存 M (逐项检查；最小形态为即时定性，延迟定性属 Layer 2) ----
	surviving := false
	if survivors > 0 {
		m1 := prep.SuppliesPacked/float64(survivors) >= escapeSuppliesPerSurvivor
		m2 := prep.CareReady >= escapeCareThreshold
		m3 := prep.ProductionStart
		m4 := prep.PlanCompleteness >= escapeOrgThreshold
		switch {
		case !m1:
			surviving = false
			settle.Notes = append(settle.Notes, "补给不足：落地即散")
		case !m4:
			surviving = false
			settle.Notes = append(settle.Notes, "组织缺位：走出去的是人群，不是群体")
		default:
			surviving = true
			if !m2 {
				fragile = true
				settle.Notes = append(settle.Notes, "照护缺口：群体脆弱")
			}
			if !m3 {
				fragile = true
				settle.Notes = append(settle.Notes, "无生产起点：能活一季，没有明天")
			}
		}
	}

	// ---- 追踪 T (最小形态：暴露阈值 + 折减；四档判定属 Layer 2) ----
	if survivors > 0 {
		exposure := prep.PrepExposure
		if itStance == model.TopStanceCooperate {
			exposure *= escapeCooperateTrackingFactor
		}
		if exposure >= escapeTrackingThreshold {
			lost := int(math.Round(float64(survivors) * escapeTrackingLoss))
			survivors -= lost
			fragile = true
			settle.Notes = append(settle.Notes, fmt.Sprintf("筹备暴露过高：被追踪折减 %d 人", lost))
		}
	}

	// ---- 写回：堡内人口移出，逃出群体事实成立 ----
	if departed > silo.TotalPopulation {
		departed = silo.TotalPopulation
	}
	// 堡外维持状态初始化 (Layer 2)：在人口移出前调用——出发清点占比读的是
	// 出发时刻的堡内人口。只有存续群体才进入堡外月度结算。
	if survivors > 0 && surviving {
		attention := 0.0
		if s.world != nil {
			attention = s.world.Silo1.Attention[silo.ID]
		}
		initOutsideState(silo, prep, suitCoverage, departed, survivors, attention, itStance)
	}
	silo.TotalPopulation -= departed
	group := &silo.EscapedGroup
	group.Departed = departed
	if departed > 0 {
		group.Initialized = true
		group.Population = survivors
		group.Surviving = surviving && survivors > 0
		group.Fragile = fragile
		if group.DepartedYear == 0 {
			group.DepartedYear = silo.CurrentYear
			group.DepartedMonth = silo.CurrentMonth
		}
	}
	settle.Survivors = survivors
	settle.Surviving = group.Surviving
	settle.Fragile = fragile

	s.appendContentLog(event, fmt.Sprintf(
		"[EscapeSettlement] 容量=%d 出发=%d 幸存=%d 存续=%v 脆弱=%v 立场(市长/IT)=%s/%s 备注=%v",
		settle.Capacity, departed, survivors, settle.Surviving, fragile, mayorStance, itStance, settle.Notes))
	return settle
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

package service

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// 堡外维持 (16 §6.4，Layer 2)：出发结算之后，逃出群体逐月结算三件事实——
// 维持 (补给/防护服/路线)、追踪 (四档)、组合 (质量)。
//
// 纪律：
//   - 结算是事实不是系数：三个维持侧面各自记录、各自出结果；"能撑几个月"
//     是三个侧面预测的最小值，撑不住时覆没原因写明是哪一面塌的
//     (冻死/耗尽/被地形困住，外加追捕歼灭)。
//   - 追踪不是必然失败：档位改变后续的选择空间——被追着跑的人带不走东西
//     (弃置补给)、走不快 (路线退化加速、不能觅食)，但群体仍可能存续。
//   - 确定性：全部输入来自状态与日历 (天气 = 月份的纯函数)，不新增 RNG
//     取样；同种子同动作序列逐月复现。
//   - 资格仍只读群体存续：质量只进 P2 加权 (玩家贡献项)，不作第二道资格门。

// TODO(balance)：以下全部为初始校准值 (16 §7)。
const (
	// 维持侧面。
	outsidePerCapitaSupplies = 1.0  // 人均月消耗 (人·月口径)
	outsideForageFactor      = 0.6  // 觅食抵消 = 路线安全 × 本系数 (追捕/被困时不可用)
	outsideSuitWearBase      = 0.04 // 防护服完整性月磨损基数
	outsideRouteDegradeBase  = 0.03 // 路线安全月退化基数
	outsideTrapThreshold     = 0.15 // 路线安全低于此值 = 被地形困住
	outsideFreezeMargin      = 0.10 // 严寒超出防护的边际
	outsideFreezeRate        = 0.10 // 冻损月比率
	outsideStarveRate        = 0.15 // 断粮月比率
	outsideTrapRate          = 0.10 // 被困月比率
	outsidePursuitRate       = 0.05 // 追捕月损耗比率
	outsideMinGroupSize      = 20   // 低于此规模群体解体
	outsideDisbandIntegrity  = 0.20 // 灾难性解体：防护服完整性下限
	// 追踪四档。
	outsideTrackingAware           = 0.25 // 热度阈值：已被察觉
	outsideTrackingTrailing        = 0.50 // 咬住痕迹
	outsideTrackingPursuit         = 0.75 // 正在追捕
	outsideTrackingDecay           = 0.02 // 距离随月数增长：热度衰减
	outsideTrackingWeightExposure  = 0.8  // 筹备暴露权重 (出发时定档)
	outsideTrackingWeightDeparted  = 0.5  // 出发清点权重 (人口占比越大越显眼)
	outsideTrackingWeightAttention = 0.6  // 1 号注意力权重 (逐月读现值)
	outsideTrailingSupplyDrop      = 0.25 // 进入"咬住痕迹"：弃置补给比例
	// 群体组合 (质量)。
	outsideLeaderRate   = 0.02 // 领头人目标配比
	outsideArtisanRate  = 0.05 // 工匠目标配比
	outsideElderShare   = 0.12 // 老人人口占比 (地形学常数)
	outsideWoundedCarry = 0.5  // 路上损耗中随队伤者比例
	outsideScaleFull    = 150  // 规模档位满分人口
)

// outsideWeatherSeverity 当月严寒程度 (0-1)：月份的纯函数——1 月最冷 (0.9)，
// 7 月最暖 (0.1)。确定性纪律：天气不抽签，读日历。
func outsideWeatherSeverity(month int) float64 {
	m := ((month-1)%12 + 12) % 12
	return 0.5 + 0.4*math.Cos(2*math.Pi*float64(m)/12)
}

// outsideTrackingTier 热度定档 (四档)。
func outsideTrackingTier(heat float64) string {
	switch {
	case heat >= outsideTrackingPursuit:
		return model.TrackingPursuit
	case heat >= outsideTrackingTrailing:
		return model.TrackingTrailing
	case heat >= outsideTrackingAware:
		return model.TrackingAware
	default:
		return model.TrackingNone
	}
}

func trackingTierRank(tier string) int {
	switch tier {
	case model.TrackingAware:
		return 1
	case model.TrackingTrailing:
		return 2
	case model.TrackingPursuit:
		return 3
	default:
		return 0
	}
}

// outsideTrackingHeat 当月追踪热度 = 出发定档基数 + 1 号注意力现值 − 距离衰减。
func (s *GameService) outsideTrackingHeat(station *model.Silo, g *model.EscapeGroup) float64 {
	attention := 0.0
	if s.world != nil {
		attention = s.world.Silo1.Attention[station.ID]
	}
	return g.Outside.TrackingBase + outsideTrackingWeightAttention*attention -
		outsideTrackingDecay*float64(g.Outside.MonthsOut)
}

// initOutsideState 出发结算初始化堡外维持状态 (仅当群体存续时调用)。
// 补给余量 = 封存账本；防护服完整性 = 出发时的覆盖率；路线安全按接应档位；
// 追踪基数由筹备暴露与出发清点定档 (并读 1 号注意力现值)；群体组合由准备
// 事实派生。
func initOutsideState(silo *model.Silo, prep model.EscapePreparation, suitCoverage float64, departed, survivors int, attention float64) {
	g := &silo.EscapedGroup
	out := &g.Outside
	out.Active = true
	out.SuppliesLeft = prep.SuppliesPacked
	out.SuitIntegrity = clampUnit01(suitCoverage)
	switch prep.ReceptionStage {
	case model.ReceptionStocked:
		out.RouteSafety = 0.9 // TODO(balance)
	case model.ReceptionConfirmed:
		out.RouteSafety = 0.8 // TODO(balance)
	case model.ReceptionSurveyed:
		out.RouteSafety = 0.4 // TODO(balance)
	default:
		out.RouteSafety = 0.25 // TODO(balance)
	}
	// 出发清点：占比越大，缺口越显眼。
	departedShare := 0.0
	if departed > 0 {
		departedShare = float64(departed) / float64(silo.TotalPopulation+departed)
	}
	out.TrackingBase = outsideTrackingWeightExposure*prep.PrepExposure +
		outsideTrackingWeightDeparted*departedShare // TODO(balance)
	out.Tracking = outsideTrackingTier(out.TrackingBase + outsideTrackingWeightAttention*attention)
	// 群体组合：领头人来自预案组织，工匠来自演练与材料，伤者来自路上损耗，
	// 老人是地形学事实。全部 TODO(balance)。
	out.Leaders = int(math.Round(float64(departed) * outsideLeaderRate * (0.5 + 0.5*prep.PlanCompleteness)))
	artisanInput := 0.5*prep.DrillLevel + 0.5*clampUnit01(prep.MaterialsPacked/100)
	out.Artisans = int(math.Round(float64(departed) * outsideArtisanRate * artisanInput))
	roadLoss := departed - survivors
	if roadLoss < 0 {
		roadLoss = 0
	}
	out.Wounded = clampInt(int(math.Round(float64(roadLoss)*outsideWoundedCarry)), survivors)
	out.Elders = clampInt(int(math.Round(float64(survivors)*outsideElderShare)), survivors)
}

// outsideGroupQuality 群体质量 (0-1)：组合事实的派生读数。领头人与工匠的
// 覆盖率是正资产，负担 (伤者+老人) 与规模是结构项。只进 P2 加权，
// 绝不进资格。TODO(balance)：四项权重。
func outsideGroupQuality(g *model.EscapeGroup) float64 {
	pop := g.Population
	if pop <= 0 {
		return 0
	}
	leaderCov := clampUnit01(float64(g.Outside.Leaders) / (float64(pop) * outsideLeaderRate))
	artisanCov := clampUnit01(float64(g.Outside.Artisans) / (float64(pop) * outsideArtisanRate))
	burden := clampUnit01(float64(g.Outside.Wounded+g.Outside.Elders) / float64(pop))
	scale := clampUnit01(float64(pop) / outsideScaleFull)
	return clampUnit01(0.30*leaderCov + 0.30*artisanCov + 0.20*scale + 0.20*(1-burden))
}

// outsideForecast 预测"还能撑几个月"：三个侧面各自的月数，取最小者并指明
// 约束侧面。预测是呈现量，由事实推得，不写回状态。
func outsideForecast(g *model.EscapeGroup) (int, string) {
	out := g.Outside
	pop := g.Population
	if pop <= 0 {
		return 0, ""
	}
	// 补给面：净消耗 = 人均需求 − 觅食抵消 (追捕/被困时觅食不可用)。
	forage := 0.0
	if out.Tracking != model.TrackingPursuit && out.RouteSafety >= outsideTrapThreshold {
		forage = out.RouteSafety * outsideForageFactor * float64(pop) * outsidePerCapitaSupplies
	}
	net := float64(pop)*outsidePerCapitaSupplies - forage
	monthsSupplies := math.MaxInt32
	if net > 0 {
		monthsSupplies = int(out.SuppliesLeft / net)
	}
	// 保暖面：完整性磨到冻损边际以下的月数 (平均严寒 0.5)。
	monthsWarmth := math.MaxInt32
	wear := outsideSuitWearBase * 1.0 // TODO(balance)：平均严寒
	if wear > 0 && out.SuitIntegrity > outsideFreezeMargin {
		monthsWarmth = int((out.SuitIntegrity - outsideFreezeMargin) / wear)
	} else if out.SuitIntegrity <= outsideFreezeMargin {
		monthsWarmth = 0
	}
	// 机动面：路线安全退化到被困阈值的月数。
	monthsRoute := math.MaxInt32
	if out.RouteSafety <= outsideTrapThreshold {
		monthsRoute = 0
	} else {
		monthsRoute = int((out.RouteSafety - outsideTrapThreshold) / outsideRouteDegradeBase)
	}
	months, facet := monthsSupplies, "补给"
	if monthsWarmth < months {
		months, facet = monthsWarmth, "保暖"
	}
	if monthsRoute < months {
		months, facet = monthsRoute, "路线"
	}
	if months < 0 {
		months = 0
	}
	if months > 999 {
		months = 999 // 呈现上限："长期自持"
	}
	return months, facet
}

// runOutsideSustenance 堡外月度结算 (世界编排阶段 2c：1 号裁决后、时钟推进前)。
// 只对激活的存续群体生效；零值状态 (旧档/未出发) 逐位不变。
func (s *GameService) runOutsideSustenance(logs *[]string, stories *[]model.StoryEvent) {
	station := s.silo
	if station == nil || s.gameOver() {
		return
	}
	g := &station.EscapedGroup
	if !g.Initialized || !g.Surviving || !g.Outside.Active {
		return
	}
	out := &g.Outside
	out.MonthsOut++
	month := station.CurrentMonth
	severity := outsideWeatherSeverity(month)

	// ---- 追踪档：热度 = 定档基数 + 1 号注意力现值 − 距离衰减 ----
	prevTier := out.Tracking
	out.Tracking = outsideTrackingTier(s.outsideTrackingHeat(station, g))
	if out.Tracking != prevTier {
		if trackingTierRank(out.Tracking) > trackingTierRank(prevTier) &&
			trackingTierRank(out.Tracking) >= trackingTierRank(model.TrackingTrailing) {
			// 被追着跑的人带不走东西：弃置补给。
			out.SuppliesLeft *= (1 - outsideTrailingSupplyDrop) // TODO(balance)
		}
		s.appendOutsideStory(stories, station, "追踪档位变化", fmt.Sprintf(
			"堡外第 %d 个月：追踪从%s变为%s。", out.MonthsOut, trackingTierName(prevTier), trackingTierName(out.Tracking)))
	}
	trailMult := 1.0
	switch out.Tracking {
	case model.TrackingTrailing:
		trailMult = 1.5 // TODO(balance)
	case model.TrackingPursuit:
		trailMult = 2.0 // TODO(balance)
	}

	// ---- 维持侧面一：补给 (觅食受路线安全与追踪塑形) ----
	pop := g.Population
	forage := 0.0
	if out.Tracking != model.TrackingPursuit && out.RouteSafety >= outsideTrapThreshold {
		forage = out.RouteSafety * outsideForageFactor * float64(pop) * outsidePerCapitaSupplies
	}
	need := float64(pop) * outsidePerCapitaSupplies
	consumption := need - forage
	if consumption < 0 {
		consumption = 0
	}
	out.SuppliesLeft -= consumption
	if out.SuppliesLeft < 0 {
		out.SuppliesLeft = 0
	}
	starving := out.SuppliesLeft <= 0 && forage < need

	// ---- 维持侧面二：防护服 (随天气磨损) ----
	out.SuitIntegrity = clampUnit01(out.SuitIntegrity - outsideSuitWearBase*(0.5+severity)) // TODO(balance)
	freezing := severity > out.SuitIntegrity+outsideFreezeMargin

	// ---- 维持侧面三：路线安全 (随天气与地形退化，追踪加速) ----
	out.RouteSafety = clampUnit01(out.RouteSafety - outsideRouteDegradeBase*(0.5+severity)*trailMult) // TODO(balance)
	trapped := out.RouteSafety < outsideTrapThreshold
	if trapped {
		out.TrappedMonths++
	} else {
		out.TrappedMonths = 0
	}

	// ---- 损耗：四个来源各自记账，覆没原因读主导项 ----
	loss := 0
	freezeLoss, starveLoss, trapLoss, pursuitLoss := 0, 0, 0, 0
	if freezing {
		freezeLoss = int(math.Round(float64(pop) * outsideFreezeRate * (severity - out.SuitIntegrity))) // TODO(balance)
	}
	if starving {
		starveLoss = int(math.Round(float64(pop) * outsideStarveRate))
	}
	if trapped {
		trapLoss = int(math.Round(float64(pop) * outsideTrapRate))
	}
	if out.Tracking == model.TrackingPursuit {
		pursuitLoss = int(math.Round(float64(pop) * outsidePursuitRate))
	}
	loss = freezeLoss + starveLoss + trapLoss + pursuitLoss
	if loss > pop {
		loss = pop
	}
	if loss > 0 {
		g.Population -= loss
		g.Fragile = true
		// 累计损耗审计：覆没归因读累计主导项，不读单月快照。
		out.AttritionFreeze += freezeLoss
		out.AttritionStarve += starveLoss
		out.AttritionTrap += trapLoss
		out.AttritionPursuit += pursuitLoss
		// 组合事实随人口收缩：负担与资产都不超过在世人数。
		out.Wounded = clampInt(out.Wounded, g.Population)
		out.Elders = clampInt(out.Elders, g.Population)
		out.Leaders = clampInt(out.Leaders, g.Population)
		out.Artisans = clampInt(out.Artisans, g.Population)
	}

	// ---- 解体判定：人口耗尽 / 三面同时坍塌 / 低于最小规模 ----
	// 归因读累计损耗的主导项 (并列按固定序，确定性)；无损耗的小型群体
	// (出发即不足最小规模) 按约束侧面归因。
	demise := ""
	switch {
	case g.Population <= 0:
		demise = dominantDemise(out.AttritionFreeze, out.AttritionStarve, out.AttritionTrap, out.AttritionPursuit)
	case out.SuppliesLeft <= 0 && trapped && out.SuitIntegrity <= outsideDisbandIntegrity:
		demise = model.DemiseExhaust
		if freezing {
			demise = model.DemiseFreeze
		}
	case g.Population < outsideMinGroupSize:
		if total := out.AttritionFreeze + out.AttritionStarve + out.AttritionTrap + out.AttritionPursuit; total > 0 {
			demise = dominantDemise(out.AttritionFreeze, out.AttritionStarve, out.AttritionTrap, out.AttritionPursuit)
		} else {
			demise = bindingDemise(out)
		}
	}
	if demise != "" {
		g.Surviving = false
		out.Active = false
		out.DemiseCause = demise
		s.appendOutsideStory(stories, station, "逃出群体覆没", fmt.Sprintf(
			"堡外第 %d 个月：%s。最后清点：%d 人出发，无人存续。%s",
			out.MonthsOut, demiseNarrative(demise), g.Departed, outsideEpitaph(demise)))
		if logs != nil {
			*logs = append(*logs, fmt.Sprintf("[Outside] 群体覆没（%s），堡外存续 %d 个月", demise, out.MonthsOut))
		}
		return
	}

	// ---- 压力标记与月度呈现 ----
	if out.Tracking == model.TrackingPursuit || out.SuppliesLeft < 2*need ||
		out.SuitIntegrity < 0.3 || out.RouteSafety < 0.3 {
		g.Fragile = true
	}
	forecastMonths, facet := outsideForecast(g)
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"[Outside] 堡外第 %d 个月：%d 人，补给约可支撑 %d 个月（约束：%s），防护服完整性 %.0f%%，路线安全 %.0f%%，追踪：%s",
			out.MonthsOut, g.Population, forecastMonths, facet,
			out.SuitIntegrity*100, out.RouteSafety*100, trackingTierName(out.Tracking)))
	}
}

// dominantDemise 损耗主导项即覆没原因；并列按固定序 (确定性)。
func dominantDemise(freeze, starve, trap, pursuit int) string {
	cause, max := model.DemiseFreeze, freeze
	if starve > max {
		cause, max = model.DemiseExhaust, starve
	}
	if trap > max {
		cause, max = model.DemiseTrap, trap
	}
	if pursuit > max {
		cause = model.DemisePursuit
	}
	return cause
}

// bindingDemise 规模解体时的约束侧面：哪一面先塌，群体就死于哪一面。
func bindingDemise(out *model.OutsideState) string {
	switch {
	case out.SuppliesLeft <= 0:
		return model.DemiseExhaust
	case out.RouteSafety < outsideTrapThreshold:
		return model.DemiseTrap
	case out.SuitIntegrity <= outsideDisbandIntegrity:
		return model.DemiseFreeze
	case out.Tracking == model.TrackingPursuit:
		return model.DemisePursuit
	default:
		return model.DemiseExhaust
	}
}

func trackingTierName(tier string) string {
	switch tier {
	case model.TrackingAware:
		return "已被察觉"
	case model.TrackingTrailing:
		return "被咬住痕迹"
	case model.TrackingPursuit:
		return "正在被追捕"
	default:
		return "没人注意"
	}
}

func demiseNarrative(cause string) string {
	switch cause {
	case model.DemiseFreeze:
		return "严寒比防护服更持久，队伍在风里失温"
	case model.DemiseTrap:
		return "路线在塌方与滑坡里断了，队伍被困在走不出去的地形里"
	case model.DemisePursuit:
		return "追捕者咬住了队尾，损耗一个月比一个月重"
	default:
		return "封存的补给吃完了，觅食补不上缺口"
	}
}

func outsideEpitaph(cause string) string {
	switch cause {
	case model.DemiseFreeze:
		return "他们冻死在离安全更近的地方。"
	case model.DemiseTrap:
		return "大山留下了他们。"
	case model.DemisePursuit:
		return "旧权力追到了最后一个人。"
	default:
		return "他们饿死在还能看见绿色的地方。"
	}
}

func (s *GameService) appendOutsideStory(stories *[]model.StoryEvent, station *model.Silo, title, description string) {
	if stories == nil {
		return
	}
	*stories = append(*stories, model.StoryEvent{
		ID:          fmt.Sprintf("outside:%d:%d:%s", station.CurrentYear, station.CurrentMonth, title),
		Category:    "EXTERNAL",
		Title:       title,
		Description: description,
		Type:        "EXTERNAL",
		SiloID:      station.ID,
		SiloName:    station.Name,
	})
}

func clampInt(v, max int) int {
	if v < 0 {
		return 0
	}
	if v > max {
		return max
	}
	return v
}

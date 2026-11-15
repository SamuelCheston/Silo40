package service

// 线信息架构：五条结局线的统一进度快照与主线判定。
//
// 设计口径 (对应界面信息架构任务)：
//  1. 一局只有一条主线——玩家实际在走的那条。主线由里程碑事实确定性推出
//     (最深者胜；平层取最近推进；再平按固定序)，不消耗 RNG，不参与结算。
//  2. 改道是玩家可见的事实：主线每次变化写入 World.LineShifts (持久化)，
//     呈现端据此渲染改道横幅与日志，而不是让"某个数字悄悄不涨"。
//  3. 失败线征兆只报告已经发生的事实 (29-failure-endings §2 的既有征兆键)，
//     不点名结局、不预测概率——提示的分寸止于"事实已被看见"。
//
// 跨线共享的事实 (信念覆盖、登记簿、W、警察服从……) 不在本快照里重复：
// 它们已在 Silo/InformationProgress 快照中单点下发，呈现端只读一次。

import (
	"silo40/internal/model"
)

// lineMilestoneDef 线上一格里程碑。AlternateKeys 里任一键点亮即算该格点亮
// (如开窗方式的秩序/暴乱两条互斥支路)。
type lineMilestoneDef struct {
	Key           string
	AlternateKeys []string
}

type lineDef struct {
	Line       string
	Title      string
	Milestones []lineMilestoneDef
}

// lineDefs 五条可玩结局线与其里程碑事实键 (链序即展示序)。固定序与仲裁
// P3 同分序 (endingOutcomeOrder 去掉 EXCLUSIONIST) 保持一致，平层时兜底。
var lineDefs = []lineDef{
	{
		Line:  model.LineInformation,
		Title: "《共证》",
		Milestones: []lineMilestoneDef{
			{Key: "player_actions:receiver_verify_accompany"},
			{Key: "special:replication_progress"},
			{Key: "special:independent_receivers_met"},
			{Key: "special:living_chain_confirmed"},
			{Key: "special:public_fact_cross_dept"},
			{Key: "special:info_vigil_eve"},
		},
	},
	{
		Line:  model.LineRebellion,
		Title: "《越岭》",
		Milestones: []lineMilestoneDef{
			{Key: "special:departure_willingness"},
			{Key: "special:route_confirmed"},
			{Key: "special:escape_window_open"},
			{Key: "player_actions:escape_window_orderly", AlternateKeys: []string{"player_actions:escape_window_riot"}},
			{Key: "player_actions:escape_departure_launch"},
		},
	},
	{
		Line:  model.LineRuleOfLaw,
		Title: "《公议》",
		Milestones: []lineMilestoneDef{
			{Key: "special:order_chain_breach"},
			{Key: "player_actions:charter_contact_accept"},
			{Key: "special:charter_draft_convened"},
			{Key: "special:charter_lock_day"},
		},
	},
	{
		Line:  model.LineTime,
		Title: "《无人裁决》",
		Milestones: []lineMilestoneDef{
			{Key: "special:open_discourse_milestone"},
			{Key: "special:pre_fall_choice"},
		},
	},
	{
		Line:  model.LinePersonalRule,
		Title: "《人治》",
		Milestones: []lineMilestoneDef{
			{Key: "special:mayor_evidence_step"},
			{Key: "special:mayor_crisis_stage"},
			{Key: "special:mayor_public_stand"},
			{Key: "special:police_obedience_shift"},
		},
	},
}

// failureOmenKeys 五条失败线的既有事前征兆 (29-failure-endings §2)。
// 全部是已实现的内容事件键；快照只读其点亮状态，不新造预警机制。
var failureOmenKeys = []string{
	"special:normalcy_hardens",        // 《深闭》：又一年没有改变
	"special:reset_omen_census",       // 《遗忘》：异常普查
	"crisis:reset_omen_requisition",   // 《遗忘》：异常请领
	"special:silo1_purge_omen",        // 《遗忘》并行路径：本地整肃准备
	"special:archive_purge_order",     // 《遗忘》并行路径：缓慢绞杀
	"crisis:fracture_infrastructure",  // 《断层》：破坏落到维生基础
	"special:identity_trace_accumulates", // 《除名》：纸面接缝
	"special:archive_review_opened",   // 《除名》：档案复查
	"special:agent_erasure_staged",    // 《除名》：抹除准备
	"special:termination_channel_test", // 《寂堡》：远程终止通道
	"special:strike_reconnaissance",   // 《寂堡》：物理打击侦察
	"special:safeguard_protocol_warming", // 《寂堡》：保障机制升温
}

// lineMilestoneState 读取一格里程碑的点亮、计数与最近点亮时刻。
func (s *GameService) lineMilestoneState(siloID uint, def lineMilestoneDef) (lit bool, count, lastTS int) {
	states := s.contentStates[siloID]
	primary := states[def.Key]
	lit = primary.Triggered
	count = primary.TriggerCount
	lastTS = primary.LastTriggeredTimestamp
	for _, alt := range def.AlternateKeys {
		state := states[alt]
		if state.Triggered {
			lit = true
		}
		if state.TriggerCount > count {
			count = state.TriggerCount
		}
		if state.LastTriggeredTimestamp > lastTS {
			lastTS = state.LastTriggeredTimestamp
		}
	}
	return lit, count, lastTS
}

// lineTitle 从内容定义取里程碑的人话标题；未知键退回键名本身。
func (s *GameService) lineMilestoneTitle(key string) string {
	for _, def := range s.contentDefinitions {
		if def.Key == key {
			return def.Title
		}
	}
	return key
}

// lineQualification 单线硬资格判定——与仲裁 P1 读同一组事实、同一口径，
// 界面因此永远与结算一致。返回资格与未成立时的具体原因。
func (s *GameService) lineQualification(line string, station *model.Silo) (bool, string) {
	switch line {
	case model.LineInformation:
		if allDepartmentsHoldFragments(station) {
			return true, ""
		}
		return false, "跨部门公共事实未成立（部门碎片覆盖不足）"
	case model.LineRebellion:
		if station.EscapedGroup.SurvivingEscapeGroup() {
			return true, ""
		}
		return false, "无实际逃离并存续的成规模群体"
	case model.LineRuleOfLaw:
		if station.Authority.Judicial.PactLocked {
			return true, ""
		}
		return false, "公开约章未锁定"
	case model.LineTime:
		switch {
		case !station.Silo1Destroyed:
			return false, "一号地堡尚未覆灭"
		case !station.PreFallSnapshot.Taken:
			return false, "缺少覆灭前夕思潮快照"
		case !station.PreFallSnapshot.OpenQualified:
			return false, "覆灭前夕思潮闸未通过（纯种地堡思潮主导）"
		}
		return true, ""
	case model.LinePersonalRule:
		return personalRuleEligible(station)
	}
	return false, ""
}

// buildLineProgress 汇总单条线的玩家可见进度。只读内容状态与堡状态。
func (s *GameService) buildLineProgress(def lineDef, station *model.Silo) model.LineProgress {
	progress := model.LineProgress{Line: def.Line, Title: def.Title}
	for _, milestone := range def.Milestones {
		lit, count, lastTS := s.lineMilestoneState(station.ID, milestone)
		title := s.lineMilestoneTitle(milestone.Key)
		progress.Milestones = append(progress.Milestones, model.LineMilestone{
			Key:   milestone.Key,
			Title: title,
			Lit:   lit,
			Count: count,
		})
		if lit {
			progress.Depth++
			if lastTS > progress.LastAdvanced {
				progress.LastAdvanced = lastTS
			}
		}
	}
	progress.Qualified, progress.QualNote = s.lineQualification(def.Line, station)

	switch def.Line {
	case model.LinePersonalRule:
		// 回摆：公开表态已经发生 (里程碑常亮)，成立条件却不再成立——
		// 路没有消失，是秩序摆回去了。这正是 mayor_public_stand 可重复
		// 点亮所抵御的东西 (30-personal-rule-nodes §抉择后节点)。
		stand := s.contentFactTriggered(station.ID, "special:mayor_public_stand")
		if stand && !progress.Qualified {
			progress.Regressed = true
		}
	case model.LineRuleOfLaw:
		// 开窗期已过而约章未锁定：线被事实关闭。
		if s.contentFactTriggered(station.ID, "crisis:charter_window_expiry") &&
			!s.contentFactTriggered(station.ID, "special:charter_lock_day") {
			progress.Closed = true
			progress.CloseNote = "开窗期已过而约章未锁定"
		}
	case model.LineTime:
		// 覆灭已成定局且思潮闸未过：硬条件在覆灭瞬间定格，线被事实关闭。
		if station.Silo1Destroyed && station.PreFallSnapshot.Taken && !station.PreFallSnapshot.OpenQualified {
			progress.Closed = true
			progress.CloseNote = "覆灭前夕思潮闸未通过，硬条件已定格"
		}
	}
	return progress
}

// deriveMainLine 从五条线的进度推出当下主线：最深者胜；平层取最近推进；
// 再平按 lineDefs 固定序。全部里程碑层为 0 时主线未定 (返回空)。
func deriveMainLine(lines []model.LineProgress) string {
	best := ""
	bestDepth := 0
	bestTS := -1
	for _, line := range lines {
		if line.Depth == 0 {
			continue
		}
		better := false
		switch {
		case best == "":
			better = true
		case line.Depth > bestDepth:
			better = true
		case line.Depth == bestDepth && line.LastAdvanced > bestTS:
			better = true
		}
		if better {
			best = line.Line
			bestDepth = line.Depth
			bestTS = line.LastAdvanced
		}
	}
	return best
}

// linesProgress 汇总五条线的统一进度快照。只读内容状态，不消耗 RNG。
//
// 主线读持久化值 (World.MainLine)。持久化值与当前事实的同步由
// syncMainLine 在本快照构建时幂等执行——快照构建位于服务互斥锁内且
// 紧随每一次状态落点 (月度 tick、动作、步骤、读档) 之后，因此这是
// 唯一的同步点，不需要在各状态落点分别挂钩。
func (s *GameService) linesProgress() *model.LinesProgress {
	if s.silo == nil {
		return nil
	}
	progress := &model.LinesProgress{}
	for _, def := range lineDefs {
		progress.Lines = append(progress.Lines, s.buildLineProgress(def, s.silo))
	}
	s.syncMainLine(progress.Lines)
	if s.world != nil {
		progress.MainLine = s.world.MainLine
		progress.Shifts = append([]model.LineShift(nil), s.world.LineShifts...)
	}
	progress.Omens = s.failureOmens()
	return progress
}

// failureOmens 已发生的失败线征兆列表 (只读)。征兆走既有月度事件与故事
// 通道，这里只是把它们聚合成玩家可见的一等事实——不点名结局，不预测。
func (s *GameService) failureOmens() []model.OmenFact {
	states := s.contentStates[s.silo.ID]
	var omens []model.OmenFact
	for _, key := range failureOmenKeys {
		state := states[key]
		if !state.Triggered {
			continue
		}
		year, month := decodeGameTimestamp(state.LastTriggeredTimestamp)
		omens = append(omens, model.OmenFact{
			Key:   key,
			Title: s.lineMilestoneTitle(key),
			Count: state.TriggerCount,
			Year:  year,
			Month: month,
		})
	}
	return omens
}

// syncMainLine 把持久化主线与当前事实对齐：推导值与持久值不同时，更新
// World.MainLine 并追加一条改道日志 (含首次确立，From 为空)。确定性、
// 幂等、不消耗 RNG；结局已定后不再改动。
func (s *GameService) syncMainLine(lines []model.LineProgress) {
	if s.world == nil || s.silo == nil || s.gameOver() {
		return
	}
	derived := deriveMainLine(lines)
	if derived == s.world.MainLine {
		return
	}
	s.world.LineShifts = append(s.world.LineShifts, model.LineShift{
		Year:  s.world.CurrentYear,
		Month: s.world.CurrentMonth,
		From:  s.world.MainLine,
		To:    derived,
	})
	s.world.MainLine = derived
}

// decodeGameTimestamp 是 gameTimestamp 的逆运算：把世界时刻 (天) 还原为
// 年/月，供征兆与里程碑的展示使用。
func decodeGameTimestamp(ts int) (year, month int) {
	if ts <= 0 {
		return 0, 0
	}
	totalMonths := ts / daysPerMonth
	return timestampBaseYear + totalMonths/monthsPerYear, totalMonths%monthsPerYear + 1
}

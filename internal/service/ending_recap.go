package service

// 结局回读：把"本局关键抉择"投影成带时序的 3–6 条条目，挂在 EndingDecision
// 上随结算下发。设计口径：
//  1. 只读既有事实——contentStates 的完成键/事实键 (何时、哪一支)、内容事件
//     定义的既有文案 (标题/叙事)、StoryEventLog 的点火记录 (缺定义时的文案
//     与时点兜底)、以及与仲裁同口径的堡内事实 (后果句)。不新造状态、不抽 RNG。
//  2. 选材先回读"玩家做过的抉择"：窗口/路口/反制/资源投入四类抉择键跨线
//     全量收齐 (recapDecisionKeys) 优先占位，再按结局候选键表补本线里程碑
//     与征兆；不足 3 条时用其余已点亮里程碑/征兆补齐，仍不足 3 条返回 nil
//     ——呈现端退化为既有结算行，不给空面板。总量仍夹在 6 条内。
//  3. 后果句一律由完成键事实与结算事实拼成 (与仲裁读同一组事实)，不另写
//     抒情模板。

import (
	"fmt"
	"sort"
	"strings"

	"silo40/internal/model"
)

// recapEndingNames 十个结局的正典书名 (与呈现端标题一致)。
var recapEndingNames = map[string]string{
	endingInformation:  "《共证》",
	endingRebellion:    "《越岭》",
	endingRuleOfLaw:    "《公议》",
	endingTime:         "《无人裁决》",
	endingPersonalRule: "《人治》",
	endingExclusionist: "《深闭》",
	endingReset:        "《遗忘》",
	endingFracture:     "《断层》",
	endingDeath:        "《寂堡》",
	endingCompromised:  "《除名》",
}

// recapDecisionKeys 本局关键抉择目录：玩家亲手做过选择的完成键，跨线全量
// 回读 (不受结局候选表限制)。四类：
//   - 窗口：抉择机会出现过 (窗本身也是事实)；
//   - 路口：互斥支里实际点亮的那一支 (未点亮的支自然跳过)；
//   - 反制：对重置/整肃/终止链条的玩家反制动作；
//   - 资源投入：把补给/材料/工时花进准备账本的动作。
//
// 顺序即同刻排序的相关度序；全部是既有内容事件键/完成键，只读点亮状态。
var recapDecisionKeys = []string{
	// 窗口
	"player_actions:charter_contact_offer",
	"player_actions:route_survey_offer",
	"special:route_second_window",
	"crisis:charter_window_expiry",
	"special:escape_window_open",
	"player_actions:reset_countermeasure_offer",
	// 路口：约章接触
	"player_actions:charter_contact_accept",
	"player_actions:charter_contact_decline",
	"player_actions:charter_contact_defer",
	"player_actions:charter_second_accept",
	"player_actions:charter_second_decline",
	// 路口：路线勘察
	"player_actions:route_survey_accept",
	"player_actions:route_survey_decline",
	"player_actions:route_survey_defer",
	"player_actions:route_survey_reaccept",
	// 路口：继承与市长
	"player_actions:succession_pure_rule",
	"player_actions:succession_seed_procedure",
	"player_actions:mayor_ally_accept",
	"player_actions:mayor_ally_bypass",
	// 路口：覆灭前夕
	"player_actions:pre_fall_open_inquiry",
	"player_actions:pre_fall_loyalty_mobilization",
	"player_actions:pre_fall_total_riot",
	// 路口：证据处置
	"player_actions:publish_evidence",
	"player_actions:verify_evidence",
	// 路口：开窗方式
	"player_actions:escape_window_orderly",
	"player_actions:escape_window_riot",
	// 路口：谈判层 (开信道/递条款/试探是过程抉择，接管/不接管互斥)
	"player_actions:negotiation_open_channel",
	"player_actions:negotiation_offer_terms",
	"player_actions:negotiation_probe",
	"player_actions:negotiation_hands_off",
	"player_actions:negotiation_takeover",
	// 反制
	"player_action:reset_countermeasure_done",
	"player_actions:silo1_purge_countermeasure",
	"player_actions:silo1_purge_plot_leak",
	"player_actions:termination_channel_jam",
	// 资源投入：划拨与接应
	"player_actions:escape_prep_transfer_supplies",
	"player_actions:escape_prep_transfer_materials",
	"player_actions:escape_reception_stocking",
	// 资源投入：能力准备
	"player_actions:escape_prep_care",
	"player_actions:escape_prep_production",
	"player_actions:escape_prep_suit_repair",
	"player_actions:escape_prep_suit_warehouse",
	"player_actions:escape_drill",
	// 资源投入：预案功能位
	"player_actions:escape_plan_agricultural",
	"player_actions:escape_plan_mechanical",
	"player_actions:escape_plan_medical",
	"player_actions:escape_plan_mines",
	"player_actions:escape_plan_order",
	"player_actions:escape_plan_supply",
	// 资源投入：先遣/出发/保障
	"player_actions:advance_party_dispatch",
	"player_actions:escape_departure_launch",
	"player_actions:escape_negotiation_guarantee",
}

// recapCompletionTitles 完成键的人话标题：完成键没有内容定义，点火记录也
// 可能缺位——回读时不给玩家看裸键名。
var recapCompletionTitles = map[string]string{
	"player_action:reset_countermeasure_done": "托管反制完成",
}

// recapPlanRoleNames 预案功能位键 → 人话部门名 (与内容定义标题同口径)。
var recapPlanRoleNames = map[string]string{
	"player_actions:escape_plan_agricultural": "农业",
	"player_actions:escape_plan_mechanical":   "机械",
	"player_actions:escape_plan_medical":      "医疗",
	"player_actions:escape_plan_mines":        "矿业",
	"player_actions:escape_plan_order":        "秩序",
	"player_actions:escape_plan_supply":       "补给",
}

// recapCandidates 每个结局的候选键表，按因果相关度排序 (前面的先入选)。
// 键全部是既有内容事件键/完成键；未点亮的候选自然跳过。
func recapCandidates(selected string) []string {
	lineKeys := func(line string) []string {
		var keys []string
		for _, def := range lineDefs {
			if def.Line != line {
				continue
			}
			for _, m := range def.Milestones {
				keys = append(keys, m.Key)
				keys = append(keys, m.AlternateKeys...)
			}
		}
		return keys
	}
	switch selected {
	case endingInformation:
		return append(lineKeys(model.LineInformation),
			"player_actions:publish_evidence", "player_actions:verify_evidence")
	case endingRebellion:
		keys := append(lineKeys(model.LineRebellion),
			"player_actions:route_survey_accept", "player_actions:escape_drill")
		// 惨胜尾声：原驻地的毁灭/重置事实也是本结局的因果条目。
		return append(keys, resetFactKey, fractureFactKey, terminationFactKey, strikeFactKey)
	case endingRuleOfLaw:
		return append(lineKeys(model.LineRuleOfLaw),
			"player_actions:charter_contact_decline", "player_actions:charter_contact_defer",
			"crisis:charter_window_expiry")
	case endingTime:
		return append(lineKeys(model.LineTime),
			"player_actions:pre_fall_open_inquiry", "player_actions:pre_fall_loyalty_mobilization",
			"player_actions:pre_fall_total_riot")
	case endingPersonalRule:
		return append(lineKeys(model.LinePersonalRule),
			"player_actions:succession_pure_rule", "player_actions:succession_seed_procedure",
			"player_actions:mayor_ally_accept", "player_actions:mayor_ally_bypass")
	case endingExclusionist:
		return []string{normalcyFactKey, "crisis:charter_window_expiry", "special:pre_fall_choice"}
	case endingReset:
		return []string{
			resetFactKey,
			"player_action:reset_countermeasure_done", "player_actions:reset_countermeasure_offer",
			"special:reset_omen_census", "crisis:reset_omen_requisition",
			"special:silo1_purge_omen", "special:archive_purge_order",
		}
	case endingFracture:
		keys := []string{fractureFactKey, "crisis:fracture_infrastructure", "crisis:fracture_point_of_no_return"}
		return append(keys, lineKeys(model.LineRebellion)...)
	case endingDeath:
		keys := []string{
			terminationFactKey, strikeFactKey,
			"special:last_interlock_warming", "special:termination_channel_test",
			"special:strike_reconnaissance", "player_actions:termination_channel_jam",
		}
		return append(keys, lineKeys(model.LineRebellion)...)
	case endingCompromised:
		return []string{
			"special:identity_trace_accumulates", "special:archive_review_opened",
			"special:agent_erasure_staged", "player_actions:expose_execution_failure",
		}
	default:
		return nil
	}
}

// recapPick 一条入选候选：键、点亮时刻与相关度序 (同刻排序兜底)。
type recapPick struct {
	key  string
	ts   int
	rank int
}

// buildEndingRecap 在仲裁收口处投影本局关键抉择。只读事实，确定性输出；
// 缺料 (不足 3 条) 返回 nil，呈现端退化为既有结算行。
func (s *GameService) buildEndingRecap(station *model.Silo, selected string) []model.EndingRecapEntry {
	if station == nil || selected == "" || len(s.contentStates[station.ID]) == 0 {
		return nil
	}
	states := s.contentStates[station.ID]
	seen := map[string]bool{}
	var picks []recapPick
	add := func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true
		state := states[key]
		if !state.Triggered {
			return
		}
		picks = append(picks, recapPick{key: key, ts: state.LastTriggeredTimestamp, rank: len(picks)})
	}
	// 抉择优先：窗口/路口/反制/资源投入跨线全量回读，先占位。
	for _, key := range recapDecisionKeys {
		add(key)
	}
	for _, key := range recapCandidates(selected) {
		add(key)
	}
	// 补齐：其余已点亮的线里程碑与失败征兆也是玩家做过的事实。
	if len(picks) < 3 {
		for _, def := range lineDefs {
			for _, m := range def.Milestones {
				add(m.Key)
				for _, alt := range m.AlternateKeys {
					add(alt)
				}
			}
		}
		for _, key := range failureOmenKeys {
			add(key)
		}
	}
	if len(picks) < 3 {
		return nil
	}
	if len(picks) > 6 {
		picks = picks[:6]
	}
	// 呈现按时序；同刻按相关度序，保证确定性。
	sort.SliceStable(picks, func(i, j int) bool {
		if picks[i].ts != picks[j].ts {
			return picks[i].ts < picks[j].ts
		}
		return picks[i].rank < picks[j].rank
	})

	entries := make([]model.EndingRecapEntry, 0, len(picks))
	for _, p := range picks {
		entry := model.EndingRecapEntry{Key: p.key, Line: recapKeyLine(p.key)}
		if p.ts > 0 {
			entry.Year, entry.Month = decodeGameTimestamp(p.ts)
		}
		entry.Title = s.lineMilestoneTitle(p.key)
		for _, def := range s.contentDefinitions {
			if def.Key == p.key {
				entry.Description = def.Description
				break
			}
		}
		// 缺定义/缺时点：用实际点火的叙事记录兜底 (完成键没有内容定义)。
		if entry.Year == 0 || entry.Title == p.key || entry.Description == "" {
			if log := s.recapEventLog(station.ID, p.key); log != nil {
				if entry.Year == 0 {
					entry.Year, entry.Month = log.Year, log.Month
				}
				if entry.Title == p.key && log.Title != "" {
					entry.Title = log.Title
				}
				if entry.Description == "" {
					entry.Description = log.Description
				}
			}
		}
		if entry.Title == p.key {
			if title, ok := recapCompletionTitles[p.key]; ok {
				entry.Title = title
			} else {
				entry.Title = strings.SplitN(p.key, ":", 2)[1]
			}
		}
		entry.Consequence = s.recapConsequence(p.key, station, selected)
		entries = append(entries, entry)
	}
	return entries
}

// recapEventLog 找该键最近一次实际点火的叙事记录 (EventID 即键的后缀)。
func (s *GameService) recapEventLog(siloID uint, key string) *model.StoryEventLog {
	eventID := strings.SplitN(key, ":", 2)[1]
	for i := len(s.eventLogs) - 1; i >= 0; i-- {
		log := &s.eventLogs[i]
		if log.SiloID == siloID && log.EventID == eventID {
			return log
		}
	}
	return nil
}

// recapKeyLine 键归属的线 (含互斥支)；不属五线返回空。
func recapKeyLine(key string) string {
	for _, def := range lineDefs {
		for _, m := range def.Milestones {
			if m.Key == key {
				return def.Line
			}
			for _, alt := range m.AlternateKeys {
				if alt == key {
					return def.Line
				}
			}
		}
	}
	return ""
}

// recapLineTitle 线的正典书名。
func recapLineTitle(line string) string {
	for _, def := range lineDefs {
		if def.Line == line {
			return def.Title
		}
	}
	return line
}

// recapEscapeFact 逃离群体的结算事实 (与 rebellionSettlementLine 同口径)。
func recapEscapeFact(station *model.Silo) string {
	g := station.EscapedGroup
	if !g.Initialized || g.Departed <= 0 {
		return "出发未成事实"
	}
	loss := g.Departed - g.Population
	if loss < 0 {
		loss = 0
	}
	return fmt.Sprintf("%d 人出发，%d 人幸存落地，路上损耗 %d 人", g.Departed, g.Population, loss)
}

// recapLineFact 一条线在结算时的资格事实——与仲裁 P1 读同一组事实。
func (s *GameService) recapLineFact(line string, station *model.Silo) string {
	switch line {
	case model.LineInformation:
		if allDepartmentsHoldFragments(station) {
			return fmt.Sprintf("跨部门公共事实成立，信念层合格事实 %d 项", beliefCoverageFactCount(station))
		}
		return "跨部门公共事实未成立"
	case model.LineRebellion:
		return recapEscapeFact(station)
	case model.LineRuleOfLaw:
		if station.Authority.Judicial.PactLocked {
			return "公开约章已锁定"
		}
		return "公开约章未锁定"
	case model.LineTime:
		switch {
		case !station.Silo1Destroyed:
			return "一号地堡尚未覆灭"
		case !station.PreFallSnapshot.Taken:
			return "缺少覆灭前夕思潮快照"
		case !station.PreFallSnapshot.OpenQualified:
			return "覆灭前夕思潮闸未通过"
		default:
			return "覆灭前夕思潮闸已通过，硬条件定格"
		}
	case model.LinePersonalRule:
		if ok, reason := personalRuleEligible(station); ok {
			return fmt.Sprintf("市长正统性 %.2f，警察服从 %.2f", station.MayorLegitimacy(), station.Authority.PoliceObedience)
		} else {
			return reason
		}
	}
	return ""
}

// recapConsequence 一条因果句：完成键事实 + 与仲裁同口径的结算事实。
func (s *GameService) recapConsequence(key string, station *model.Silo, selected string) string {
	pactFact := s.recapLineFact(model.LineRuleOfLaw, station)
	infoFact := s.recapLineFact(model.LineInformation, station)
	timeFact := s.recapLineFact(model.LineTime, station)
	personalFact := s.recapLineFact(model.LinePersonalRule, station)
	countermeasureDone := s.contentFactTriggered(station.ID, "player_action:reset_countermeasure_done")
	countermeasureFact := "反制未完成"
	if countermeasureDone {
		countermeasureFact = "反制已完成"
	}

	switch key {
	case "player_actions:escape_window_orderly":
		return "以秩序方式开窗：" + recapEscapeFact(station) + "。"
	case "player_actions:escape_window_riot":
		return "以暴乱方式开窗：" + recapEscapeFact(station) + "。"
	case "player_actions:charter_contact_accept":
		return "接受了约章接触；" + pactFact + "。"
	case "player_actions:charter_contact_decline":
		return "拒绝了约章接触；" + pactFact + "。"
	case "player_actions:charter_contact_defer":
		return "推迟了约章接触；" + pactFact + "。"
	case "player_actions:succession_pure_rule":
		return "继承抉择选了纯人治；" + personalFact + "。"
	case "player_actions:succession_seed_procedure":
		return "继承抉择选了播种程序；" + pactFact + "。"
	case "player_actions:mayor_ally_accept":
		return "选择与市长结盟；" + personalFact + "。"
	case "player_actions:mayor_ally_bypass":
		return "选择绕开市长；" + personalFact + "。"
	case "player_actions:pre_fall_open_inquiry":
		return "覆灭前夕选了公开质询；" + timeFact + "。"
	case "player_actions:pre_fall_loyalty_mobilization":
		return "覆灭前夕选了忠诚动员；" + timeFact + "。"
	case "player_actions:pre_fall_total_riot":
		return "覆灭前夕选了全面暴乱；" + timeFact + "。"
	case "player_actions:publish_evidence":
		return "选择公开证据；" + infoFact + "。"
	case "player_actions:verify_evidence":
		return "选择核验证据；" + infoFact + "。"
	case "player_actions:charter_contact_offer":
		return "程序派联系人出现过；" + pactFact + "。"
	case "player_actions:charter_second_accept":
		return "第二扇窗里接受了约章接触；" + pactFact + "。"
	case "player_actions:charter_second_decline":
		return "第二扇窗里拒绝了约章接触；" + pactFact + "。"
	case "crisis:charter_window_expiry":
		return "约章开窗期到期关闭；" + pactFact + "。"
	case "player_actions:route_survey_offer":
		return "路线勘察委托窗口出现过；" + s.recapRouteFact(station.ID) + "。"
	case "player_actions:route_survey_accept":
		return "接受了路线勘察委托；" + s.recapRouteFact(station.ID) + "。"
	case "player_actions:route_survey_decline":
		return "拒绝了路线勘察委托；" + s.recapRouteFact(station.ID) + "。"
	case "player_actions:route_survey_defer":
		return "推迟了路线勘察委托；" + s.recapRouteFact(station.ID) + "。"
	case "player_actions:route_survey_reaccept":
		return "在第二扇窗里重新委托勘察；" + s.recapRouteFact(station.ID) + "。"
	case "special:route_second_window":
		return "第二扇勘察窗在收窄后打开；" + s.recapRouteFact(station.ID) + "。"
	case "special:escape_window_open":
		return "逃离窗口打开：" + recapEscapeFact(station) + "。"
	case "player_actions:negotiation_open_channel":
		return "打开了与市长的谈判信道；" + recapNegotiationFact(station) + "。"
	case "player_actions:negotiation_offer_terms":
		return "把妥协条款摆上了桌面；" + recapNegotiationFact(station) + "。"
	case "player_actions:negotiation_probe":
		return fmt.Sprintf("试探市长底线 %d 次；%s。",
			s.contentStates[station.ID][key].TriggerCount, recapNegotiationFact(station))
	case "player_actions:negotiation_hands_off":
		return "妥协后选择不接管、带组织离开；" + recapEscapeFact(station) + "。"
	case "player_actions:negotiation_takeover":
		return "妥协后选择接管市长职位；" + personalFact + "。"
	case "player_actions:silo1_purge_countermeasure":
		count := s.contentStates[station.ID][key].TriggerCount
		if s.contentFactTriggered(station.ID, "special:silo1_purge_softened") {
			return fmt.Sprintf("名册反制触发 %d 次并已生效：整肃降档执行，少带走了一些人。", count)
		}
		return fmt.Sprintf("名册反制触发 %d 次，但篡改尚未生效。", count)
	case "player_actions:silo1_purge_plot_leak":
		count := s.contentStates[station.ID][key].TriggerCount
		if s.contentFactTriggered(station.ID, "special:silo1_purge_exposed") {
			return fmt.Sprintf("信息反制触发 %d 次并已生效：整肃图谋被公开，执行失去突然性。", count)
		}
		return fmt.Sprintf("信息反制触发 %d 次，但图谋尚未被公开。", count)
	case "player_actions:reset_countermeasure_offer":
		return "反制窗口出现过；" + countermeasureFact + "。"
	case "player_action:reset_countermeasure_done":
		if s.contentFactTriggered(station.ID, resetFactKey) {
			return "反制已完成，但重置仍然发生。"
		}
		return "反制已完成，重置没有发生。"
	case "player_actions:termination_channel_jam":
		count := s.contentStates[station.ID][key].TriggerCount
		jamFact := "信道干扰未生效"
		if s.contentFactTriggered(station.ID, "special:termination_channel_disrupted") {
			jamFact = "信道干扰已生效"
		}
		if s.contentFactTriggered(station.ID, terminationFactKey) {
			return fmt.Sprintf("通道反制触发 %d 次，%s，但远程终止仍然执行。", count, jamFact)
		}
		return fmt.Sprintf("通道反制触发 %d 次，%s，远程终止没有执行。", count, jamFact)
	case "player_actions:expose_execution_failure":
		if s.agent != nil {
			return fmt.Sprintf("执行失败被暴露，怀疑度升至 %.2f。", s.agent.SuspicionLevel)
		}
		return "执行失败被暴露。"
	case "player_actions:escape_prep_transfer_supplies":
		return fmt.Sprintf("累计划拨 %d 次；结算封存补给共 %.0f 人·月。",
			s.contentStates[station.ID][key].TriggerCount, station.EscapePrep.SuppliesPacked)
	case "player_actions:escape_prep_transfer_materials":
		return fmt.Sprintf("累计划拨 %d 次；结算封存装备材料共 %.0f。",
			s.contentStates[station.ID][key].TriggerCount, station.EscapePrep.MaterialsPacked)
	case "player_actions:escape_reception_stocking":
		return fmt.Sprintf("接应点完成囤备；结算接应档位为%s，封存补给共 %.0f 人·月。",
			recapReceptionStageName(station.EscapePrep.ReceptionStage), station.EscapePrep.SuppliesPacked)
	case "player_actions:escape_prep_care":
		return fmt.Sprintf("照护储备已建立；结算照护能力 %.2f。", station.EscapePrep.CareReady)
	case "player_actions:escape_prep_production":
		fact := "未备"
		if station.EscapePrep.ProductionStart {
			fact = "已备"
		}
		return "生产起点动作完成；结算时生产起点" + fact + "。"
	case "player_actions:escape_prep_suit_repair":
		return fmt.Sprintf("累计修复 %d 次；结算可用防护服 %d 套。",
			s.contentStates[station.ID][key].TriggerCount, station.EscapePrep.SuitsFunctional)
	case "player_actions:escape_prep_suit_warehouse":
		return fmt.Sprintf("累计启用预置缓存 %d 次；结算可用防护服 %d 套。",
			s.contentStates[station.ID][key].TriggerCount, station.EscapePrep.SuitsFunctional)
	case "player_actions:escape_drill":
		return fmt.Sprintf("累计演练 %d 次；结算演练水平 %.2f。",
			s.contentStates[station.ID][key].TriggerCount, station.EscapePrep.DrillLevel)
	case "player_actions:advance_party_dispatch":
		return fmt.Sprintf("先遣小组已派出；结算接应档位为%s。",
			recapReceptionStageName(station.EscapePrep.ReceptionStage))
	case "player_actions:escape_departure_launch":
		return "出发已启动：" + recapEscapeFact(station) + "。"
	case "player_actions:escape_negotiation_guarantee":
		return fmt.Sprintf("向顶层递出了保障清单；结算时市长立场为%s。",
			recapTopStanceName(station.TopStance.MayorStance()))
	case resetFactKey:
		return fmt.Sprintf("重置完成：驻地仍有居民 %d 人，历史包袱降至 %.2f。", station.TotalPopulation, station.HistoryBurden)
	case fractureFactKey:
		return fmt.Sprintf("断层确认：驻地仍有居民 %d 人，技术上还活着，已经失去了明天。", station.TotalPopulation)
	case terminationFactKey:
		return "远程终止执行，驻地被关闭——这一事实直接触发《寂堡》护栏。"
	case strikeFactKey:
		return "物理打击执行，堡体被摧毁——这一事实直接触发《寂堡》护栏。"
	case normalcyFactKey:
		state := s.contentStates[station.ID][key]
		return fmt.Sprintf("封闭叙事硬化已点亮 %d 次，封闭秩序主导了危机时刻的选择。", state.TriggerCount)
	}

	// 预案功能位：报告对接了哪个部门与结算预案完备度。
	if role, ok := recapPlanRoleNames[key]; ok {
		prep := station.EscapePrep
		return fmt.Sprintf("完成%s预案功能位对接；结算预案完备度 %.2f（已对接 %d 个功能位）。",
			role, prep.PlanCompleteness, len(prep.PlanRoles))
	}

	// 失败征兆：报告已发生的次数与该失败线的可归属事实。
	for _, omen := range failureOmenKeys {
		if key != omen {
			continue
		}
		state := s.contentStates[station.ID][key]
		fact := ""
		switch selected {
		case endingReset:
			fact = countermeasureFact
		case endingCompromised:
			if s.agent != nil {
				fact = fmt.Sprintf("怀疑度 %.2f", s.agent.SuspicionLevel)
			}
		case endingDeath:
			fact = fmt.Sprintf("保障风险 %.2f", station.SafeguardRisk)
		case endingFracture:
			fact = fmt.Sprintf("叛乱值 %.2f", station.Rebellion)
		}
		if fact == "" {
			return fmt.Sprintf("征兆已点亮 %d 次。", state.TriggerCount)
		}
		return fmt.Sprintf("征兆已点亮 %d 次；%s。", state.TriggerCount, fact)
	}

	// 线里程碑：第几格点亮 + 该线结算事实。
	if line := recapKeyLine(key); line != "" {
		depth := 0
		for _, def := range lineDefs {
			if def.Line != line {
				continue
			}
			for i, m := range def.Milestones {
				if m.Key == key {
					depth = i + 1
					break
				}
				for _, alt := range m.AlternateKeys {
					if alt == key {
						depth = i + 1
						break
					}
				}
			}
		}
		return fmt.Sprintf("点亮%s第 %d 格；结算时：%s。", recapLineTitle(line), depth, s.recapLineFact(line, station))
	}
	return fmt.Sprintf("该事实计入%s的结算。", recapEndingNames[selected])
}

// recapRouteFact 路线勘察的结算事实 (读既有路线确认键)。
func (s *GameService) recapRouteFact(siloID uint) string {
	if s.contentFactTriggered(siloID, "special:route_confirmed") {
		return "路线已确认"
	}
	return "路线未确认"
}

// recapNegotiationFact 谈判层的结算事实 (只读既有谈判账本)。
func recapNegotiationFact(station *model.Silo) string {
	n := station.Negotiation
	if !n.Initialized {
		return "谈判层未开启"
	}
	compromise := "妥协未成立"
	if n.CompromiseEstablished {
		compromise = "妥协已成立"
	}
	return fmt.Sprintf("谈判已交换 %d 回合，%s", n.Rounds, compromise)
}

// recapReceptionStageName 接应档位的人话名。
func recapReceptionStageName(stage string) string {
	switch stage {
	case "surveyed":
		return "已勘察"
	case "confirmed":
		return "已确认"
	case "stocked":
		return "已囤备"
	}
	return "未建立"
}

// recapTopStanceName 顶层立场的人话名。
func recapTopStanceName(stance string) string {
	switch stance {
	case model.TopStanceNonBlock:
		return "不阻挠"
	case model.TopStanceCooperate:
		return "配合"
	}
	return "阻挠"
}

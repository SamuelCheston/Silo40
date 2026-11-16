package service

// 结局回读：把"本局关键抉择"投影成带时序的 3–6 条条目，挂在 EndingDecision
// 上随结算下发。设计口径：
//  1. 只读既有事实——contentStates 的完成键/事实键 (何时、哪一支)、内容事件
//     定义的既有文案 (标题/叙事)、StoryEventLog 的点火记录 (缺定义时的文案
//     与时点兜底)、以及与仲裁同口径的堡内事实 (后果句)。不新造状态、不抽 RNG。
//  2. 按对结局的因果相关度选材：每个结局一张候选键表 (本线里程碑与抉择支
//     优先，失败结局读征兆与反制)，不足 3 条时用其余已点亮里程碑/征兆补齐，
//     仍不足 3 条返回 nil——呈现端退化为既有结算行，不给空面板。
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
			entry.Title = strings.SplitN(p.key, ":", 2)[1]
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
	case "player_actions:reset_countermeasure_offer":
		return "反制窗口出现过；" + countermeasureFact + "。"
	case "player_action:reset_countermeasure_done":
		if s.contentFactTriggered(station.ID, resetFactKey) {
			return "反制已完成，但重置仍然发生。"
		}
		return "反制已完成，重置没有发生。"
	case "player_actions:termination_channel_jam":
		if s.contentFactTriggered(station.ID, terminationFactKey) {
			return "干扰了终止通道，但远程终止仍然执行。"
		}
		return "干扰了终止通道，远程终止没有执行。"
	case "player_actions:expose_execution_failure":
		if s.agent != nil {
			return fmt.Sprintf("执行失败被暴露，怀疑度升至 %.2f。", s.agent.SuspicionLevel)
		}
		return "执行失败被暴露。"
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
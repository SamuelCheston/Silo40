package service

import (
	"fmt"
	"sort"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// Ending candidates and arbitration follow docs/research/12-ending-system-design.md:
// qualification precedes weighting, hard failures precede history, and ties
// resolve in a fixed deterministic order.

const (
	endingInformation  = "INFORMATION"
	endingRebellion    = "REBELLION"
	endingRuleOfLaw    = "RULE_OF_LAW"
	endingTime         = "TIME"
	endingPersonalRule = "PERSONAL_RULE"
	endingExclusionist = "EXCLUSIONIST"
	endingReset        = "RESET"
	endingFracture     = "FRACTURE"
	endingDeath        = "DEATH"
	endingCompromised  = "AGENT_COMPROMISED"

	// Fact keys produced by content chains (18/19 分册); arbitration only reads them.
	resetFactKey    = "special:reset_completed"
	fractureFactKey = "special:fracture_confirmed"

	// Silo 1 reaction-chain settlement hooks (《寂堡》链 21). The remote-termination
	// and remote-physical-strike settlements carry no numerical effect of their
	// own; firing one records its completion key, which arbitration reads as a
	// DEATH fact. The local-purge settlement is deliberately NOT a DEATH fact —
	// it has no modeled consequence this round (see its TODO).
	terminationFactKey = "crisis:termination_release"
	strikeFactKey      = "crisis:bombing_terminal"
)

// endingOutcomeOrder is the P3 tie-break order (结局体系 12 §6.2 P3). It exists
// purely for determinism; it deliberately changes the old skeleton behaviour
// where TIME outranked REBELLION on ties.
var endingOutcomeOrder = []string{
	endingInformation, endingRebellion, endingRuleOfLaw,
	endingTime, endingPersonalRule, endingExclusionist,
}

// endingOutcomeWeights keeps every tunable outcome coefficient in one config.
type endingOutcomeWeights struct {
	Information  float64
	Time         float64
	Rebellion    float64
	RuleOfLaw    float64
	PersonalRule float64
	Exclusionist float64
}

type endingWeightConfig struct {
	Version                 int
	MinimumScore            float64
	PrestigeScale           float64
	PlayerBase              endingOutcomeWeights
	PlayerPerPrestige       endingOutcomeWeights
	PlayerPerFragment       endingOutcomeWeights
	PlayerPerPropaganda     endingOutcomeWeights
	PlayerPerSpilloverSilo  endingOutcomeWeights
	FactionPerInfluence     endingOutcomeWeights
	FactionPerOpenInfluence endingOutcomeWeights
	FactionPerLowCohesion   endingOutcomeWeights
	ResidentPerInfluence    endingOutcomeWeights
	ResidentPerOpenIdeology endingOutcomeWeights
	ResidentPerAmbition     endingOutcomeWeights
	Silo1AliveBase          endingOutcomeWeights
	Silo1DestroyedBase      endingOutcomeWeights
	Silo1PerRebellion       endingOutcomeWeights
	Silo1PerCohesion        endingOutcomeWeights
}

// TODO(balance): all values below are placeholders for future measured tuning
// (结局体系 12 §9 leaves weights to sim). This round proves the qualification
// and arbitration structure, not the ending distribution.
var defaultEndingWeightConfig = endingWeightConfig{
	Version:       2,
	MinimumScore:  1,
	PrestigeScale: 100,
	PlayerBase: endingOutcomeWeights{
		Information: 3, Time: 2, Rebellion: 3, RuleOfLaw: 1, PersonalRule: 1, Exclusionist: 1,
	},
	PlayerPerPrestige: endingOutcomeWeights{
		Information: 0.5, Time: 0.2, Rebellion: 0.8, RuleOfLaw: 0.2, PersonalRule: 0.3, Exclusionist: 0.1,
	},
	PlayerPerFragment:      endingOutcomeWeights{Information: 0.05},
	PlayerPerPropaganda:    endingOutcomeWeights{Rebellion: 0.4},
	PlayerPerSpilloverSilo: endingOutcomeWeights{Information: 0.5}, // G7: each distinct silo reached by station-originated signals.
	FactionPerInfluence: endingOutcomeWeights{
		Information: 0.2, Time: 0.1, Rebellion: 0.2, RuleOfLaw: 0.15, PersonalRule: 0.15, Exclusionist: 0.2,
	},
	FactionPerOpenInfluence: endingOutcomeWeights{
		Information: 0.8, Rebellion: 0.2, RuleOfLaw: 0.3, Exclusionist: -0.4,
	},
	FactionPerLowCohesion: endingOutcomeWeights{Rebellion: 1.2, Exclusionist: -0.2},
	ResidentPerInfluence: endingOutcomeWeights{
		Information: 0.2, Time: 0.1, Rebellion: 0.2, RuleOfLaw: 0.15, PersonalRule: 0.2, Exclusionist: 0.2,
	},
	ResidentPerOpenIdeology: endingOutcomeWeights{Information: 0.6, RuleOfLaw: 0.3, Exclusionist: -0.4},
	ResidentPerAmbition:     endingOutcomeWeights{Rebellion: 0.8, PersonalRule: 0.4},
	Silo1AliveBase:          endingOutcomeWeights{Exclusionist: 2},
	Silo1DestroyedBase:      endingOutcomeWeights{Time: 4},
	Silo1PerRebellion:       endingOutcomeWeights{Time: 2},
	Silo1PerCohesion:        endingOutcomeWeights{Exclusionist: 1},
}

func outcomeWeight(weights endingOutcomeWeights, outcome string) float64 {
	switch outcome {
	case endingInformation:
		return weights.Information
	case endingTime:
		return weights.Time
	case endingRebellion:
		return weights.Rebellion
	case endingRuleOfLaw:
		return weights.RuleOfLaw
	case endingPersonalRule:
		return weights.PersonalRule
	case endingExclusionist:
		return weights.Exclusionist
	default:
		return 0
	}
}

// endingDescriptions are the settlement narratives per main ending. Factual
// epilogues (charter locked, station destroyed, escape group surviving) are
// appended by the arbitrator, not encoded as separate endings (结局体系 12 §6.1).
var endingDescriptions = map[string]string{
	endingInformation:  "核心真相已经成为跨部门、可独立复核的公共事实：它有了自己的来源说明、自己的核验路径、自己的保管人。你的沉默、离开或失忆，都不再能把它收回——链不再需要你，这正是任务要的结果。驻地之外是否有人接过这份事实，只写进结算，不改变这个结局。《共证》成立。",
	endingRebellion:    "有组织的不满转化为了实际离开：一个成规模的群体在堡外继续存续——《越岭》成立。",
	endingRuleOfLaw:    "裁决依据从隐蔽命令与个人正统性转向了可公开复核的规则。第一次按约章裁决的案件里，裁决者引用约章与公开证据，而不是等待上级指示；隐蔽调度不再能进入执行——法警第一次问'依据哪一条'。约章有本地主人，以不可被单方收回的程序锁定；你坐在旁听席，不在裁决席。约章法庭迟早会遇到一个它回答不了的问题——'这条命令到底从哪里来'——答案通向专线与控制网的另一端，但本结局停在这个问题被第一次提出之前。《公议》成立。",
	endingTime:         "按 Protocol 应当到来的复核没有到来，控制通道不再给出下一项处置。桌上的旧命令第一次没有后续签批，谁也不能再用下一道外部指令推迟选择。在覆灭之前就已存在的公开质询场所里，有人把那份旧命令放到桌上，问它今天是否仍然适用；没有人能从控制网另一端得到答案。争论没有立即停止，外面也没有因此变得安全——但讨论继续下去，不需要谁替全堡批准。你坐在其中，不是新裁决者。《无人裁决》成立：不是世界终于给出正确答案，而是唯一垄断裁决权的来源消失时，这里已经具备不被另一套封闭答案立即吞没的公共条件。",
	endingPersonalRule: "驻地秩序系于一位具备个人正统性的现任者。一次危机由他的个人判断收束：那道无人能解释来源的旧命令摆在桌上，他没有向居民解释全部，只以本地责任公开拒绝执行；法警没有去执行它——司法部没有站在 IT 一边，哪怕它也没有站到他一边。当夜驻地安静，居民安睡在他们并不完全理解的秩序里。这个秩序会随一个人老去，而居民此刻不需要知道更多：被保护的睡眠，与一个知道代价的守夜人。《人治》成立。",
	endingExclusionist: "什么都没有毁灭，也什么都没有改变。封闭秩序不是靠镇压守住的，是靠习惯：证人老去，接收者调离，封闭的叙事逐年硬化，始终没有遇到它的对抗版本。危机到来时驻地集体选择了旧规则，是自愿的——提供可核验替代方案的最后窗口，就这样安静地关上了。任务没有战败，它失去了战场。《深闭》成立。",
	endingReset:        "驻地安静而有序——重置总是伴随'恢复正常'。有人喊邻居的名字，对方迟疑了一下才应答；新刷的墙面盖住了旧刻痕；官方记录里只有一次'燃气事故'，或者什么都没有。减口、抹忆与清档系统性完成了：你的任务赖以继续的公共事实，与能核验它们的人，被一并拆除。征兆其实从未缺席——变长的档案'维护'窗口、远超平常的请领、深夜的人数清点——它们被当成了噪音，或者反制窗口被花在了别处。你输掉的不是一场对抗，是整块记忆。《遗忘》成立。",
	endingFracture:     "撕碎驻地的不是攻击，是它自己的手。无人控制的破坏落到了维生基础上：轮班断裂，部门停止共享，关键岗位弃守——愤怒达到了它的目的，目的却跟着塌了。你点燃的是愤怒，没有来得及建起接住它的组织：期待跑得比组织快，而维存是一张协作网，每撕掉一条网眼，都不会因为任何一方'胜利'而自动复原。驻地的灯一层一层熄灭，不是因为攻击，是因为没有人再在开关旁边。《断层》成立。",
	endingDeath:        "驻地的最后时刻不是爆炸，是顺序：灯、门、静默。没有人剩下可以承接任何东西——里面没有人知道结束他们的东西叫什么。《寂堡》成立。",
	endingCompromised:  "你的身份先于任务死去了。每一次深度行动都在纸面上留下与过去对不上的痕迹：某年的空缺，某段对不上的履历，某次回指到你自己的暴露。痕迹从一开始就在那里，你一直赌下一次行动是最后一次——直到复查到来。抹除以本地名义包装：来历存疑、刑事罪名，或者无声失踪；没有人会知道你真正是什么。驻地以'一个来历存疑的人'的名义归档——档案合上的声音，就是任务合上的声音。《除名》成立。",
}

type endingScoreAccumulator struct {
	decision *model.EndingDecision
	pool     map[string]bool
}

func (a *endingScoreAccumulator) add(kind, id, name string, weights endingOutcomeWeights, scale float64) {
	if scale == 0 {
		return
	}
	var contribution *model.EndingEntityContribution
	for i := range a.decision.Contributions {
		entry := &a.decision.Contributions[i]
		if entry.EntityID == id {
			contribution = entry
			break
		}
	}
	if contribution == nil {
		a.decision.Contributions = append(a.decision.Contributions, model.EndingEntityContribution{
			EntityID: id, Kind: kind, Name: name, Scores: map[string]float64{},
		})
		contribution = &a.decision.Contributions[len(a.decision.Contributions)-1]
	}
	// P2: the four entity types weight only qualified candidates.
	for _, outcome := range endingOutcomeOrder {
		if !a.pool[outcome] {
			continue
		}
		delta := outcomeWeight(weights, outcome) * scale
		contribution.Scores[outcome] += delta
		a.decision.Scores[outcome] += delta
	}
}

// contentFactTriggered reports whether a content fact key (category:event_id)
// has fired for the given silo. Fact keys are the only way content chains
// register terminal facts such as reset_completed / fracture_confirmed.
func (s *GameService) contentFactTriggered(siloID uint, key string) bool {
	states := s.contentStates[siloID]
	if states == nil {
		return false
	}
	return states[key].Triggered
}

// spilloverSiloCount G7: distinct non-station silos that received at least one
// station-originated cross signal. Read from the persisted outbound log, which
// is appended in deterministic dispatch order.
func (s *GameService) spilloverSiloCount(stationID uint) int {
	if s.world == nil {
		return 0
	}
	reached := map[uint]bool{}
	for _, signal := range s.world.OutboundSignals {
		if signal.SourceSilo != stationID || signal.TargetSilo == stationID || signal.TargetSilo == 0 {
			continue
		}
		reached[signal.TargetSilo] = true
	}
	return len(reached)
}

// spilloverSignalCount counts every station-originated outbound signal
// (deterministic log order). Together with spilloverSiloCount it renders the
// spillover settlement line: how much was sent out, how many silos it
// reached. Spillover never gates victory; this is settlement information
// only (25-information-endgame §3).
func (s *GameService) spilloverSignalCount(stationID uint) int {
	if s.world == nil {
		return 0
	}
	count := 0
	for _, signal := range s.world.OutboundSignals {
		if signal.SourceSilo == stationID && signal.TargetSilo != stationID && signal.TargetSilo != 0 {
			count++
		}
	}
	return count
}

// exclusionistEligible reports whether the closed order still dominates the
// station. Silo 1 destruction alone also qualifies it as the non-victory
// convergence of last resort. Closed dominance is read from the population-
// weighted cohort ideologies (cohorts iterated in ascending ID order); with no
// readable population the closed order is not established.
func exclusionistEligible(station *model.Silo) bool {
	if station.Silo1Destroyed {
		return true
	}
	cohorts := append([]model.PopulationCohort(nil), station.Cohorts...)
	sort.SliceStable(cohorts, func(i, j int) bool { return cohorts[i].ID < cohorts[j].ID })
	loyalty, proForeign, weight := 0.0, 0.0, 0.0
	for i := range cohorts {
		cohort := &cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		w := float64(cohort.Count)
		loyalty += clampUnitIdeology(cohort.Ideologies[model.IdeologyLoyalty]) * w
		proForeign += clampUnitIdeology(cohort.Ideologies[model.IdeologyProForeign]) * w
		weight += w
	}
	if weight == 0 {
		return false
	}
	return loyalty/weight > proForeign/weight
}

func clampUnitIdeology(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// informationGateFactCount is the re-anchored structural bridge depth: every
// department must hold at least this many facts at or above the sharing floor
// in its belief layer. TODO(balance): 5 → 3. Derivation (17-information-economy
// §5): one share absorbs B_player·transmit·receptivity ≈ 0.22–0.34, so a fact
// needs ~2 shares to cross the 0.30 floor and stay there against forgetting;
// 3 facts × 9 departments × ~2 shares ≈ 54 shares ≈ 162 AP, reachable inside
// the ~410 AP lifetime income while 5 facts (≈270 AP + upkeep) is not.
const informationGateFactCount = 3

// allDepartmentsHoldFragments is the local public-fact structural bridge,
// re-anchored to the belief layer after the belief rework: the legacy
// KnownFragments layer is no longer fed by Share Info (which now absorbs into
// Profession.Beliefs), so the gate reads the live layer — every department
// must hold at least informationGateFactCount facts with belief at or above
// the sharing floor. Department beliefs decay at the constant forgetting rate
// (engine.ApplyBeliefForgetting), so the gate only stays true while
// circulation outpaces forgetting (25-information-economy §修复A/B).
func allDepartmentsHoldFragments(station *model.Silo) bool {
	if len(station.Professions) == 0 {
		return false
	}
	qualRumor := engine.DefaultInformationConfig().QualRumor
	for _, prof := range station.Professions {
		count := 0
		for _, belief := range prof.Beliefs {
			if belief.B >= qualRumor {
				count++
			}
		}
		if count < informationGateFactCount {
			return false
		}
	}
	return true
}

// personalRuleEligible implements the G12 ruling (02-personal-rule §4,
// 16-chain-personal-rule §4): PERSONAL_RULE rests on police obedience and the
// Mayor's personal legitimacy. The Judicial never attaches to the Mayor, so no
// judgment reads "Judicial belongs to the Mayor" — its neutrality or hostility
// does not block establishment; only its return to IT control does. Thresholds
// reuse the authority config's named bars (MayorOpenW doubles as the "low W"
// bar; PersonalRuleMinObedience mirrors the pre-ruling committed bar).
func personalRuleEligible(station *model.Silo) (bool, string) {
	cfg := engine.DefaultAuthorityConfig()
	authority := &station.Authority
	if station.Mayor.HolderResidentID == 0 {
		return false, "无现任具名 Mayor，秩序无人承载"
	}
	if authority.W > cfg.MayorOpenW {
		return false, "幕后威势未退，秩序仍由 IT 隐蔽调度"
	}
	if station.MayorLegitimacy() < cfg.MayorMinLegitimacy {
		return false, "Mayor 个人正统性不足"
	}
	if authority.PoliceObedience < cfg.PersonalRuleMinObedience {
		return false, "警察服从不足"
	}
	if authority.Judicial.Regime == "IT" {
		return false, "Judicial 已回到 IT 手里"
	}
	return true, ""
}

// resolveWeightedEnding arbitrates the terminal evaluation requested by
// victory_check (G1/G9). It implements the fixed pipeline of 结局体系 12 §6.2:
//
//	P0 hard-failure guardrails (DEATH → RESET → FRACTURE → AGENT_COMPROMISED),
//	reading the escape group first; P1 qualification pool; P2 four-entity
//	weighting restricted to the pool; P3 fixed tie-break order. A nil result
//	keeps the engine's status untouched; Selected == "" means "no qualified
//	candidate — continue simulating" and the engine clears the request.
func (s *GameService) resolveWeightedEnding(station *model.Silo, agent *model.Agent) *model.EndingDecision {
	if station == nil || agent == nil {
		return nil
	}
	trigger := ""
	if station.VictoryStatus != nil {
		trigger = strings.ToUpper(station.VictoryStatus.Type)
	} else if !s.contentFactTriggered(station.ID, resetFactKey) {
		// Most endings still require a legacy victory_check request. RESET is a
		// content-only hard failure, so its completion fact must be able to open
		// arbitration even while the ordinary simulation status is nil.
		return nil
	}
	config := defaultEndingWeightConfig
	decision := &model.EndingDecision{
		ConfigVersion: config.Version,
		MinimumScore:  config.MinimumScore,
		Scores:        map[string]float64{},
		Contributions: []model.EndingEntityContribution{},
		Excluded:      map[string]string{},
	}
	for _, outcome := range endingOutcomeOrder {
		decision.Scores[outcome] = 0
	}

	escapeAlive := station.EscapedGroup.SurvivingEscapeGroup()
	resetFact := s.contentFactTriggered(station.ID, resetFactKey)
	fractureFact := s.contentFactTriggered(station.ID, fractureFactKey)
	// Silo 1 reaction-chain settlements are DEATH facts: the engine's own DEATH
	// trigger (safeguard/population) OR a fired remote-termination /
	// remote-physical-strike settlement hook. Both keep the escape-group
	// exception below.
	terminationFact := s.contentFactTriggered(station.ID, terminationFactKey)
	strikeFact := s.contentFactTriggered(station.ID, strikeFactKey)
	deathFact := trigger == endingDeath || terminationFact || strikeFact

	// P0 (结局体系 12 §6.2): the first three guardrails must read the escape
	// group before condemning anyone; a surviving group is never retroactively
	// killed with its origin silo.
	switch {
	case deathFact && !escapeAlive:
		decision.Selected = endingDeath
		decision.Reason = deathNarrative(s, station, trigger, terminationFact, strikeFact)
		return s.finishArbitration(decision)
	case resetFact && !escapeAlive:
		decision.Selected = endingReset
		decision.Reason = endingDescriptions[endingReset] + fmt.Sprintf("重置之后驻地仍有居民 %d 人，历史包袱降至 %.2f——人还在，记忆不在了；任务死于接收者的消失，不是死于镇压。",
			station.TotalPopulation, station.HistoryBurden)
		return s.finishArbitration(decision)
	case fractureFact && !escapeAlive:
		decision.Selected = endingFracture
		decision.Reason = endingDescriptions[endingFracture] + fractureSettlementLine(station)
		return s.finishArbitration(decision)
	case trigger == endingCompromised:
		decision.Selected = endingCompromised
		decision.Reason = endingDescriptions[endingCompromised] + compromisedSettlementLine(station)
		return s.finishArbitration(decision)
	case deathFact || resetFact || fractureFact:
		// escapeAlive == true here: the local fall becomes a pyrrhic epilogue.
		decision.Selected = endingRebellion
		decision.Reason = "逃出群体已先成立并存续：原驻地的毁灭/重置/崩坏只作《越岭》惨胜尾声。"
		return s.finishArbitration(decision)
	}

	// P1: build the qualification pool. Old triggers only request the terminal
	// evaluation; unqualified candidates never enter scoring (G9).
	pool := map[string]bool{}
	if allDepartmentsHoldFragments(station) {
		pool[endingInformation] = true
	} else {
		decision.Excluded[endingInformation] = "跨部门公共事实未成立（部门碎片覆盖不足）"
	}
	if escapeAlive {
		pool[endingRebellion] = true
	} else {
		decision.Excluded[endingRebellion] = "无实际逃离并存续的成规模群体"
	}
	switch {
	case !station.Silo1Destroyed:
		decision.Excluded[endingTime] = "一号地堡尚未覆灭"
	case !station.PreFallSnapshot.Taken:
		decision.Excluded[endingTime] = "缺少覆灭前夕思潮快照"
	case !station.PreFallSnapshot.OpenQualified:
		decision.Excluded[endingTime] = "覆灭前夕思潮闸未通过（纯种地堡思潮主导）"
	default:
		pool[endingTime] = true
	}
	if station.Authority.Judicial.PactLocked {
		pool[endingRuleOfLaw] = true
	} else {
		decision.Excluded[endingRuleOfLaw] = "公开约章未锁定"
	}
	if ok, reason := personalRuleEligible(station); ok {
		pool[endingPersonalRule] = true
	} else {
		decision.Excluded[endingPersonalRule] = reason
	}
	// EXCLUSIONIST eligibility (结局体系 12 §6.2 P1): the closed order still
	// dominates, or Silo 1 is destroyed without another qualified new order
	// (it then doubles as the non-victory historical convergence, tie-break
	// last). Closed dominance reads the population-weighted cohort ideologies.
	if exclusionistEligible(station) {
		pool[endingExclusionist] = true
	} else {
		decision.Excluded[endingExclusionist] = "封闭秩序未主导，且一号地堡尚未覆灭"
	}

	decision.Pool = make([]string, 0, len(pool))
	for _, outcome := range endingOutcomeOrder {
		if pool[outcome] {
			decision.Pool = append(decision.Pool, outcome)
		}
	}
	if len(decision.Pool) == 0 {
		decision.Selected = ""
		decision.Reason = "无合格候选：继续模拟，不硬凑结局。"
		return s.finishArbitration(decision)
	}

	// P2: four entity types weight only qualified candidates.
	accumulator := &endingScoreAccumulator{decision: decision, pool: pool}
	playerID := fmt.Sprintf("agent:%d", agent.ID)
	accumulator.add("player", playerID, agent.Name, config.PlayerBase, 1)
	prestige := agent.PoliticalPrestige / config.PrestigeScale
	accumulator.add("player", playerID, agent.Name, config.PlayerPerPrestige, prestige)
	accumulator.add("player", playerID, agent.Name, config.PlayerPerFragment, float64(len(agent.KnownFragments)))
	accumulator.add("player", playerID, agent.Name, config.PlayerPerPropaganda, agent.PropagandaLevel)
	accumulator.add("player", playerID, agent.Name, config.PlayerPerSpilloverSilo, float64(s.spilloverSiloCount(station.ID)))

	factions := append([]model.Faction(nil), station.Factions...)
	sort.SliceStable(factions, func(i, j int) bool { return factions[i].ID < factions[j].ID })
	for _, faction := range factions {
		if faction.Signature == "special:unaffiliated" {
			continue
		}
		id := fmt.Sprintf("faction:%d", faction.ID)
		accumulator.add("faction", id, faction.Name, config.FactionPerInfluence, faction.Influence)
		openInfluence := 0.0
		if strings.Contains(faction.Signature, "pro_foreign:High") || strings.Contains(strings.ToLower(faction.Stance), "open") {
			openInfluence = faction.Influence
		}
		accumulator.add("faction", id, faction.Name, config.FactionPerOpenInfluence, openInfluence)
		lowCohesion := 1 - faction.Cohesion
		if lowCohesion < 0 {
			lowCohesion = 0
		}
		accumulator.add("faction", id, faction.Name, config.FactionPerLowCohesion, faction.Influence*lowCohesion)
	}

	residents := append([]model.Resident(nil), station.Residents...)
	sort.SliceStable(residents, func(i, j int) bool { return residents[i].ID < residents[j].ID })
	for _, resident := range residents {
		if !resident.Alive || (resident.Influence == 0 && !resident.IsRepresentative) {
			continue
		}
		id := fmt.Sprintf("resident:%d", resident.ID)
		accumulator.add("resident", id, resident.Name, config.ResidentPerInfluence, resident.Influence)
		openIdeology := resident.Ideologies[model.IdeologyProForeign]
		accumulator.add("resident", id, resident.Name, config.ResidentPerOpenIdeology, resident.Influence*openIdeology)
		accumulator.add("resident", id, resident.Name, config.ResidentPerAmbition, resident.Influence*resident.Ambition)
	}

	// Silo 1 contributes external structural conditions. Its TIME weight only
	// reaches scoring when TIME is qualified (pool filter), so a failed
	// ideology gate automatically excludes the destruction contribution.
	silo1Alive := s.world != nil && s.world.Silo1.Alive
	if silo1Alive {
		accumulator.add("silo1", "silo:1", "1 号地堡", config.Silo1AliveBase, 1)
	} else {
		accumulator.add("silo1", "silo:1", "1 号地堡", config.Silo1DestroyedBase, 1)
	}
	if peer := s.fullSilo1(); peer != nil {
		accumulator.add("silo1", "silo:1", peer.Name, config.Silo1PerRebellion, peer.Rebellion)
		accumulator.add("silo1", "silo:1", peer.Name, config.Silo1PerCohesion, peer.Cohesion)
	}

	// P3: strict ">" in fixed tie-break order keeps earlier candidates on ties;
	// the minimum score only rejects qualified candidates without social support.
	best := ""
	bestScore := config.MinimumScore
	for _, outcome := range endingOutcomeOrder {
		if !pool[outcome] {
			continue
		}
		if decision.Scores[outcome] > bestScore {
			best = outcome
			bestScore = decision.Scores[outcome]
		}
	}
	if best == "" {
		decision.Selected = ""
		decision.Reason = "合格候选均未达最低社会支撑分：继续模拟。"
		return s.finishArbitration(decision)
	}
	decision.Selected = best
	// Settlement narrative replaces the score formula on the ending screen:
	// the structured audit (scores, contributions, pool, exclusions) stays in
	// the decision fields, Reason carries the story (25-information-endgame
	// §2.2). INFORMATION appends the spillover settlement line and RULE_OF_LAW
	// appends the charter evidence-base line — neither gates victory; both are
	// settlement information only.
	narrative := endingDescriptions[best]
	if best == endingInformation {
		sent := s.spilloverSignalCount(station.ID)
		reached := s.spilloverSiloCount(station.ID)
		if sent == 0 {
			narrative += "本局没有信号送出驻地——外溢为零，不影响本次裁决。"
		} else {
			narrative += fmt.Sprintf("本局共有 %d 条信号送出驻地，到达 %d 座他堡。", sent, reached)
		}
	}
	if best == endingRuleOfLaw {
		narrative += fmt.Sprintf("本局约章的证据底座为 %d 项独立公共事实，公开证据强度 Q=%.2f。",
			station.Authority.Judicial.IndependentPublicFacts, station.Authority.Judicial.PublicEvidenceQ)
	}
	if best == endingExclusionist {
		// Two settlement variants (03-exclusionist §5): with Silo 1 alive the
		// closed order keeps its external support; after the fall, local power
		// replicates the closure without orders — the order has internalized
		// its master. The settlement line reports what still runs: the station
		// survives, the possibility of change is what died.
		if s.world != nil && !s.world.Silo1.Alive {
			narrative += "主人没了，秩序却已经内化了它的主人：本地权力在没有命令的情况下执行封闭——为旧仪式发明本地理由，引用'祖制'。封闭从此不再需要外部命令，它自我维持，且连一个可以追责的外部对象都不存在。"
		} else {
			narrative += "若干年后，新一任 IT 主管在旧仪式中就职，背诵他不完全理解的词句。你的材料被归档为一个无害怪人的遗物。驻地运转良好。任务没有。"
		}
		narrative += fmt.Sprintf("结算时驻地仍有居民 %d 人，正统性 %.2f——驻地仍在运转，死掉的是改变的可能。",
			station.TotalPopulation, station.MayorLegitimacy())
	}
	if best == endingPersonalRule {
		narrative += personalRuleSettlementLine(station)
	}
	if best == endingTime {
		narrative += timeSettlementLine(s, station)
	}
	decision.Reason = narrative
	return s.finishArbitration(decision)
}

// finishArbitration records the audit copy on the world (结局体系 12 §9) and
// returns the decision for the engine hook.
func (s *GameService) finishArbitration(decision *model.EndingDecision) *model.EndingDecision {
	if s.world != nil {
		s.world.LastEndingAudit = decision
	}
	return decision
}

// fractureSettlementLine reports what is left after the maintenance collapse:
// the people, the energy, the floors. Failure does not erase the world — the
// station is technically alive and has lost its tomorrow (05-fracture §5).
func fractureSettlementLine(station *model.Silo) string {
	energy := 0.0
	for _, res := range station.Resources {
		if res.Type == "Energy" {
			energy = res.Amount
		}
	}
	stability := 0.0
	if len(station.Floors) > 0 {
		for _, floor := range station.Floors {
			stability += floor.Stability
		}
		stability /= float64(len(station.Floors))
	}
	return fmt.Sprintf("断层之后驻地仍有居民 %d 人，剩余能源 %.0f，楼层平均稳定度 %.2f——技术上还活着，已经失去了明天。",
		station.TotalPopulation, energy, stability)
}

// compromisedSettlementLine reports that the station goes on without the
// agent (06-agent-compromised §4 bitter-epilogue variant): the residents,
// the running departments, and the shareable facts the agent planted — the
// method keeps being used by people who never learn where it came from.
func compromisedSettlementLine(station *model.Silo) string {
	shareableFacts := 0
	for _, prof := range station.Professions {
		for _, belief := range prof.Beliefs {
			if belief.B >= 0.30 {
				shareableFacts++
			}
		}
	}
	return fmt.Sprintf("驻地继续运转：%d 名居民，%d 个部门仍在履职，部门信念层里仍有 %d 份可被引用的事实——你教过的方法还在被人使用，只是没有人知道它来自哪里。",
		station.TotalPopulation, len(station.Professions), shareableFacts)
}

// personalRuleSettlementLine reports the order's factual basis at
// settlement: police obedience, the mayor's personal legitimacy, the
// withdrawn W, and the judicial three-state position — the Judicial never
// belongs to the mayor (ADR-0005), its stance is not an establishment
// condition. A locked charter coexists with personal rule; on ties the
// fixed order yields to RULE_OF_LAW (30-personal-rule-nodes §6).
func personalRuleSettlementLine(station *model.Silo) string {
	var regimeNote string
	switch station.Authority.Judicial.Regime {
	case "PACT":
		regimeNote = "附着约章"
	case "SWING":
		regimeNote = "处于争夺态"
	default:
		regimeNote = station.Authority.Judicial.Regime
	}
	line := fmt.Sprintf("结算时：警察服从 %.2f，市长正统性 %.2f，幕后威势 W=%.1f，司法部%s——它永不归属市长，其立场也不是成立条件。",
		station.Authority.PoliceObedience, station.MayorLegitimacy(), station.Authority.W, regimeNote)
	if station.Authority.Judicial.PactLocked {
		line += "公开约章与人治并存：两种秩序同时成立，本次加权倾向人治一侧；同分时按既定序让《公议》。"
	}
	return line
}

// timeSettlementLine reports the facts the TIME ending stands on and what
// remains after the fall (03-time §3, 33-time-nodes §2): the pre-fall
// snapshot that passed the open gate, whatever reached other silos (residual
// responses — no promises), and the population that stays to answer for
// itself. Failure does not erase the world; even victory leaves a silence.
func timeSettlementLine(s *GameService, station *model.Silo) string {
	snap := station.PreFallSnapshot
	axis := snap.DominantAxis
	switch axis {
	case model.IdeologyProForeign:
		axis = "亲外"
	case model.IdeologyLoyalty:
		axis = "忠诚"
	}
	line := fmt.Sprintf("覆灭前夕快照：亲外 %.2f 对忠诚 %.2f（主导轴：%s），快照人口 %d——硬条件在结算瞬间定格。",
		snap.Ideologies[model.IdeologyProForeign], snap.Ideologies[model.IdeologyLoyalty], axis, snap.Population)
	sent := s.spilloverSignalCount(station.ID)
	reached := s.spilloverSiloCount(station.ID)
	if sent == 0 {
		line += "外面：本局没有信号送出驻地——外部保持沉默。"
	} else {
		line += fmt.Sprintf("外面：本局共有 %d 条信号送出驻地，到达 %d 座他堡——残余应答不是建制恢复，也不许诺。", sent, reached)
	}
	line += fmt.Sprintf("驻地内仍有居民 %d 人，从今天起，没有人再替他们签批。", station.TotalPopulation)
	return line
}

// deathNarrative composes the P0.1 settlement story (07-death §4): a
// cause-specific sentence (which step went wrong), the restrained final
// moment, and what remains — the escape attempt that did not survive, the
// outside world, the facility itself. The residual-response motif is allowed
// but promises nothing.
func deathNarrative(s *GameService, station *model.Silo, trigger string, terminationFact, strikeFact bool) string {
	var cause string
	switch {
	case terminationFact:
		cause = "终止来自控制网的另一端——驻地的偏离招来了处置：外溢与高调叛乱送出的可操作观察，最终换来了这次关闭。"
	case strikeFact:
		cause = "侦察确认了责任链，物理摧毁随之到来——分渠道结算里，这是几乎无解的渠道：干扰一种终止手段，不豁免物理手段。"
	case trigger == endingDeath && station.SafeguardRisk >= 1.0:
		cause = "驻地自身的保障机制在冲突中被触发——旧秩序的最后装置，杀死了它要保护的驻地。过度干预把风险推到了封顶。"
	default:
		cause = "没有战争。最后一班没有人接——人口在叛乱、流失与沉寂里耗尽，驻地再也凑不齐下一轮轮值。"
	}
	narrative := cause + endingDescriptions[endingDeath]
	if g := station.EscapedGroup; g.Initialized && g.Population > 0 && !g.Surviving {
		narrative += fmt.Sprintf("曾有 %d 人离开，但他们没能存续到最后——逃离成了残酷的注脚。", g.Population)
	}
	if s.world != nil && !s.world.Silo1.Alive {
		narrative += "外面同样寂静：1 号地堡已经覆灭，没有人来清点这份寂静。"
	} else {
		narrative += "控制网的某一端，这座驻地被标记为已处理；Pact 的计数减一。"
	}
	if strikeFact {
		narrative += "堡体不再矗立，原地只剩残骸的形状。几十年后也许有人会碰到一条还给出微弱应答的线路——但那不是胜利，不改变这个结局。"
	} else {
		narrative += "驻地仍然矗立——作为正常运转的地堡，覆灭了。几十年后也许有人会碰到一条还给出微弱应答的线路——但那不是胜利，不改变这个结局。"
	}
	return narrative
}

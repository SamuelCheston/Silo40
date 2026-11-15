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

	// Fact keys produced by content chains; arbitration only reads them.
	resetFactKey    = "special:reset_completed"
	fractureFactKey = "special:fracture_confirmed"
	normalcyFactKey = "special:normalcy_hardens"

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
	Version       int
	MinimumScore  float64
	PrestigeScale float64
	// PactLockedBonus 约章锁定后给 RULE_OF_LAW 的结构性加分。约章是单向制度
	// 事实 ("它的价值恰在于不能被单方收回")，其资格已包含《共证》的证据底座
	// (Q 与独立公开事实正是约章门)：锁定成立时制度成就应压过一般认知成就，
	// 否则《公议》在与《共证》同池时结构性不可赢 (声望项 0.2 vs 0.5、底分
	// 1 vs 3，而声望在开局后几乎不再增长)。取 5：覆盖对 INFORMATION 的底分差
	// 2 + 终局碎片/外溢项的现实上限 (~0.05×40 碎片 + 0.5×2 堡 ≈ 3) 并留余量；
	// 若同池仍输给《共证》，整条约章线将结构性不可赢。TODO(balance): sim 复测。
	PactLockedBonus float64
	// PersonalRuleSuccessionBonus 《人治》继承抉择 (SUCCESSION_PURE_RULE) 成立后
	// 给 PERSONAL_RULE 的结构性加分。继承抉择是本线的收束事实 (30-nodes §4：
	// "本线收束"，单向承诺)，与约章锁定之于《公议》对称：没有它，占位权重
	// (底分 1 + 声望 0.3) 永远输给《共证》(底分 3 + 0.5 + 碎片/外溢) 与《深闭》
	// (silo1 存续 +2)，本线结构性不可赢。TODO(balance): sim 复测。
	PersonalRuleSuccessionBonus float64
	PlayerBase                  endingOutcomeWeights
	PlayerPerPrestige           endingOutcomeWeights
	PlayerPerFragment           endingOutcomeWeights
	PlayerPerPropaganda         endingOutcomeWeights
	PlayerPerSpilloverSilo      endingOutcomeWeights
	// PlayerEscapeGroupQuality (16 Layer 2)：逃出群体质量作为玩家贡献项。
	// 只对合格候选计分 (pool 过滤)——资格仍只读群体存续，质量不是第二道资格门。
	PlayerEscapeGroupQuality endingOutcomeWeights
	FactionPerInfluence      endingOutcomeWeights
	FactionPerOpenInfluence  endingOutcomeWeights
	FactionPerLowCohesion    endingOutcomeWeights
	ResidentPerInfluence     endingOutcomeWeights
	ResidentPerOpenIdeology  endingOutcomeWeights
	ResidentPerAmbition      endingOutcomeWeights
	Silo1AliveBase           endingOutcomeWeights
	Silo1DestroyedBase       endingOutcomeWeights
	Silo1PerRebellion        endingOutcomeWeights
	Silo1PerCohesion         endingOutcomeWeights
}

// TODO(balance): all values below are placeholders for future measured tuning
// (结局体系 12 §9 leaves weights to sim). This round proves the qualification
// and arbitration structure, not the ending distribution.
var defaultEndingWeightConfig = endingWeightConfig{
	Version:                     2,
	MinimumScore:                1,
	PrestigeScale:               100,
	PactLockedBonus:             5, // TODO(balance): 见 endingWeightConfig.PactLockedBonus 注释。
	PersonalRuleSuccessionBonus: 5, // TODO(balance): 见 endingWeightConfig.PersonalRuleSuccessionBonus 注释。
	PlayerBase: endingOutcomeWeights{
		Information: 3, Time: 2, Rebellion: 3, RuleOfLaw: 1, PersonalRule: 1, Exclusionist: 1,
	},
	PlayerPerPrestige: endingOutcomeWeights{
		// RuleOfLaw 与 Information 同斜率 (0.5)：声望是《公议》锁定日的核心
		// 资源 ("你用剩余的关系与威望为签署程序争取那一段时间")，而声望在开局
		// 后几乎不再增长——0.2 的占位斜率使《公议》在同池时结构性不可赢。
		// TODO(balance): 以 sim 复测为准。
		Information: 0.5, Time: 0.2, Rebellion: 0.8, RuleOfLaw: 0.5, PersonalRule: 0.3, Exclusionist: 0.1,
	},
	PlayerPerFragment:        endingOutcomeWeights{Information: 0.05},
	PlayerPerPropaganda:      endingOutcomeWeights{Rebellion: 0.4},
	PlayerPerSpilloverSilo:   endingOutcomeWeights{Information: 0.5}, // G7: each distinct silo reached by station-originated signals.
	PlayerEscapeGroupQuality: endingOutcomeWeights{Rebellion: 2.0},   // TODO(balance): 群体质量 (组合事实 0-1) 的 REBELLION 加权。
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
	endingRebellion:    "有组织的不满转化为了实际离开：一个成规模的群体沿着被证明过的路走了出去，在堡外继续存续。终局不是闸门打开的那一刻，而是抵达后的第二次公开清点——有人核对下一班物资，有人照顾伤者，有人开始安排第二天的维护与生产。新社会的轮班表由他们自己改写，表上没有你的名字。人真的走出去了，也真的开始过自己必须负责的生活——《越岭》成立。",
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
// department must hold at least this many MATURED facts in its belief layer.
// TODO(balance): 5 → 3. Derivation (17-information-economy §5): one share
// absorbs B_player·transmit·receptivity ≈ 0.22–0.34, so a fact needs ~2 shares
// to cross the 0.30 floor and stay there against forgetting; 3 facts ×
// 9 departments × ~2 shares ≈ 54 shares ≈ 162 AP, reachable inside the ~410 AP
// lifetime income while 5 facts (≈270 AP + upkeep) is not. Permeation adds the
// time dimension on top without changing this depth (18-time-permeation §2).
const informationGateFactCount = 3

// allDepartmentsHoldFragments is the local public-fact structural bridge,
// reading permeation maturity after the time-permeation rework
// (18-time-permeation §3.1): every department must hold at least
// informationGateFactCount facts whose belief B stayed at or above the sharing
// floor for the full permeation window (engine.ApplyPermeation marks them
// Matured). Maturity is sticky — once a department has held a fact long
// enough it counts as usable evidence even if B later decays — so the gate is
// a planning target (permeate all 27 slots ahead of settlement), not a
// real-time juggling act (§3.4). The depth (3 facts) and the floor (QualRumor)
// are unchanged; only the time dimension is added.
func allDepartmentsHoldFragments(station *model.Silo) bool {
	if len(station.Professions) == 0 {
		return false
	}
	for _, prof := range station.Professions {
		if engine.MaturedFactCount(&prof) < informationGateFactCount {
			return false
		}
	}
	return true
}

// beliefCoverageFactCount 修复7：跨部门统计信念浓度达到分享地板的事实总数。
// 与 allDepartmentsHoldFragments 读同一层、同一地板——资格重锚到信念层后，
// 玩家加权贡献也必须读同一层 (旧 KnownFragments 层不再被 Share Info 写入)。
func beliefCoverageFactCount(station *model.Silo) int {
	qualRumor := engine.DefaultInformationConfig().QualRumor
	count := 0
	for _, prof := range station.Professions {
		for _, belief := range prof.Beliefs {
			if belief.B >= qualRumor {
				count++
			}
		}
	}
	return count
}

// ruleOfLawExplanation 补齐《公议》候选的四维解释 (10 §9 审计口径)：
// 证据 = Q 与独立合格事实；附着 = Judicial 的约章附着与锁定；介入 = Order
// 介入链阶段与 J；动机 = 推动公开规则的势力 (玩家威望与开放派系)。
func ruleOfLawExplanation(station *model.Silo, agent *model.Agent) string {
	judicial := station.Authority.Judicial
	evidence := fmt.Sprintf("证据: Q=%.2f, 合格独立事实 %d 项", judicial.PublicEvidenceQ, judicial.IndependentPublicFacts)
	if len(judicial.PublicFactKeys) > 0 {
		evidence += " (" + strings.Join(judicial.PublicFactKeys, ", ") + ")"
	}
	adhesion := fmt.Sprintf("附着: Judicial regime=%s, 约章附着=%.2f, 锁定=%v", judicial.Regime, judicial.Pact, judicial.PactLocked)
	stage := station.Protocol.InterventionStage
	if stage == "" {
		stage = "无接触"
	}
	intervention := fmt.Sprintf("介入: 介入链=%s, J=%.2f, 主任认知 G=%.2f", stage, station.Protocol.J, station.Protocol.G)
	openFactions := 0
	for _, faction := range station.Factions {
		if strings.Contains(faction.Signature, "pro_foreign:High") || strings.Contains(strings.ToLower(faction.Stance), "open") {
			openFactions++
		}
	}
	prestige := 0.0
	if agent != nil {
		prestige = agent.PoliticalPrestige
	}
	motive := fmt.Sprintf("动机: 玩家威望=%.0f, 开放派系 %d 个", prestige, openFactions)
	return motive + "; " + intervention + "; " + evidence + "; " + adhesion
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

// resolveWeightedEnding arbitrates terminal evaluations requested by
// victory_check or content facts (G1/G9). It implements 结局体系 12 §6.2:
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
	normalcyConvergence := s.contentFactTriggered(station.ID, normalcyFactKey) &&
		s.world != nil && s.world.ProgressStage == "arrival"
	if station.VictoryStatus != nil {
		trigger = strings.ToUpper(station.VictoryStatus.Type)
	} else if !s.contentFactTriggered(station.ID, resetFactKey) && !normalcyConvergence {
		// Most endings still require a legacy victory_check request. RESET is a
		// content-only hard failure. EXCLUSIONIST may converge from normalcy only
		// while no other chain has left arrival; an active chain is a crawl-out.
		return nil
	}
	config := defaultEndingWeightConfig
	decision := &model.EndingDecision{
		ConfigVersion: config.Version,
		MinimumScore:  config.MinimumScore,
		Scores:        map[string]float64{},
		Contributions: []model.EndingEntityContribution{},
		Excluded:      map[string]string{},
		Explanations:  map[string]string{},
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
		decision.Reason = "逃出群体已先成立并存续：原驻地的毁灭/重置/崩坏只作《越岭》惨胜尾声。" + rebellionSettlementLine(station)
		return s.finishArbitration(decision)
	}

	// P1: build the qualification pool. Old triggers only request the terminal
	// evaluation; unqualified candidates never enter scoring (G9).
	pool := map[string]bool{}
	if allDepartmentsHoldFragments(station) {
		pool[endingInformation] = true
	} else {
		decision.Excluded[endingInformation] = "跨部门公共事实未成立（部门成熟证据不足）"
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
	// 子项 7：《公议》候选的四维审计解释 (动机/介入/证据/附着)，无论入池与否。
	decision.Explanations[endingRuleOfLaw] = ruleOfLawExplanation(station, agent)
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
	// 修复7：碎片贡献重锚到信念覆盖 (beliefCoverageFactCount)。旧口径读
	// agent.KnownFragments——该层不再被 Share Info 写入，低威望职业完成全部门
	// 信念门后贡献仍只有零点几分，会被 Silo 1 存续期的 EXCLUSIONIST 固定贡献
	// 以毫厘压过 (职业横评 Mechanical-40：3.985 vs 3.996)。
	accumulator.add("player", playerID, agent.Name, config.PlayerPerFragment, float64(beliefCoverageFactCount(station)))
	accumulator.add("player", playerID, agent.Name, config.PlayerPerPropaganda, agent.PropagandaLevel)
	accumulator.add("player", playerID, agent.Name, config.PlayerPerSpilloverSilo, float64(s.spilloverSiloCount(station.ID)))
	// 16 Layer 2：群体质量 (老人/伤者/工匠/领头人的组合事实) 作为玩家贡献项。
	// add() 只对 pool 内候选计分——无存续群体时 REBELLION 不在 pool，质量
	// 不产生任何分数；资格判定本身不读质量。
	accumulator.add("player", playerID, agent.Name, config.PlayerEscapeGroupQuality, outsideGroupQuality(&station.EscapedGroup))

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

	// 约章锁定是单向制度事实 (22-chain-cross-arbitration §3)：锁定成立时给
	// RULE_OF_LAW 结构性加分，使其能与《共证》同池竞争 (见
	// endingWeightConfig.PactLockedBonus 的推导注释)。
	if station.Authority.Judicial.PactLocked {
		decision.Scores[endingRuleOfLaw] += config.PactLockedBonus
	}
	// 《人治》继承抉择是本线收束的单向承诺 (30-nodes §4)：与约章锁定对称，
	// 给 PERSONAL_RULE 结构性加分 (见 PersonalRuleSuccessionBonus 注释)。
	if s.contentFactTriggered(station.ID, "player_actions:succession_pure_rule") {
		decision.Scores[endingPersonalRule] += config.PersonalRuleSuccessionBonus
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
		if ok, _ := personalRuleEligible(station); ok {
			narrative += "公开约章与人治并存：两种秩序同时成立，本次加权倾向公议一侧；同分时按既定序亦让《公议》。"
		}
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
	if best == endingRebellion {
		narrative += rebellionSettlementLine(station)
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

// rebellionSettlementLine reports the departure facts as settled: how many
// left, how many survived the road, whether the group is fragile, and what
// remains behind. It reads only EscapedGroup and station population — the
// same facts arbitration reads (ADR-0008: stances and ledgers never reach
// the settlement screen).
func rebellionSettlementLine(station *model.Silo) string {
	g := station.EscapedGroup
	if !g.Initialized || g.Population <= 0 {
		return ""
	}
	loss := g.Departed - g.Population
	if loss < 0 {
		loss = 0
	}
	line := fmt.Sprintf("本局共有 %d 人出发，%d 人幸存落地，路上损耗 %d 人。", g.Departed, g.Population, loss)
	if g.Fragile {
		line += "群体的存续是脆弱的：照护、补给或追踪的压力在他们身上留下了印记。"
	}
	if g.DepartedYear > 0 {
		line += fmt.Sprintf("他们于 %d 年 %d 月离开。", g.DepartedYear, g.DepartedMonth)
	}
	// Layer 2 堡外事实：存续月数、追踪档位与群体组合 (结算只读事实)。
	out := g.Outside
	if out.MonthsOut > 0 {
		line += fmt.Sprintf("他们已在堡外存续 %d 个月，追踪档位：%s。", out.MonthsOut, trackingTierName(out.Tracking))
	}
	if out.Leaders > 0 || out.Artisans > 0 || out.Wounded > 0 || out.Elders > 0 {
		line += fmt.Sprintf("这支队伍不是一个人：%d 名领头人、%d 名工匠、%d 名伤者、%d 名老人——走出去的是群体，不是人群。",
			out.Leaders, out.Artisans, out.Wounded, out.Elders)
	}
	line += fmt.Sprintf("驻地仍有居民 %d 人——离开没有掏空驻地，只是让社会关系的延续在堡外成了事实。", station.TotalPopulation)
	return line
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
	if g := station.EscapedGroup; g.Initialized && !g.Surviving && g.Outside.DemiseCause != "" {
		// Layer 2：堡外覆灭的群体——写明是哪种死法 (冻死/耗尽/被困/追捕)。
		narrative += fmt.Sprintf("曾有 %d 人离开，堡外存续 %d 个月后覆没：%s。",
			g.Departed, g.Outside.MonthsOut, demiseNarrative(g.Outside.DemiseCause))
	} else if g := station.EscapedGroup; g.Initialized && g.Population > 0 && !g.Surviving {
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

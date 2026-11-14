package service

import (
	"fmt"
	"sort"
	"strings"

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
	endingInformation:  "核心真相已成为跨部门、可独立复核的公共事实。玩家的沉默或离开不再能收回它——《共证》成立。",
	endingRebellion:    "有组织的不满转化为了实际离开：一个成规模的群体在堡外继续存续——《越岭》成立。",
	endingRuleOfLaw:    "裁决依据从隐蔽命令与个人正统性转向了可公开复核的规则。公开约章已经锁定——《公议》成立。",
	endingTime:         "一号地堡权威覆灭，驻地仍有形成历史的能力，覆灭前夕的思潮具备面向外部的一极——《无人裁决》成立。",
	endingPersonalRule: "驻地秩序系于一位具备个人正统性的现任者，Judicial 附着于 Mayor——《人治》成立。",
	endingExclusionist: "公开求证与外部经验没有取得公共正当性，封闭秩序延续或自我复制——《深闭》成立。",
	endingReset:        "减口、抹忆与清档系统性完成，任务赖以继续的公共事实被拆除——《遗忘》成立。",
	endingFracture:     "叛乱与损耗造成了不可逆的维存崩溃，且没有独立幸存的逃出群体——《断层》成立。",
	endingDeath:        "没有任何可确认的幸存社会——《寂堡》成立。",
	endingCompromised:  "特工身份彻底暴露，任务不可继续——《除名》成立。",
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

// allDepartmentsHoldFragments is the local public-fact structural bridge: every
// department holds at least five unique fragments (12 分册 §4 现状口径).
func allDepartmentsHoldFragments(station *model.Silo) bool {
	if len(station.Professions) == 0 {
		return false
	}
	for _, prof := range station.Professions {
		unique := map[string]bool{}
		for _, f := range prof.KnownFragments {
			unique[f] = true
		}
		if len(unique) < 5 {
			return false
		}
	}
	return true
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
	if station == nil || station.VictoryStatus == nil || agent == nil {
		return nil
	}
	trigger := strings.ToUpper(station.VictoryStatus.Type)
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
		decision.Reason = "P0.1 寂堡：无可确认的幸存社会，先于一切历史加权。"
		return s.finishArbitration(decision)
	case resetFact && !escapeAlive:
		decision.Selected = endingReset
		decision.Reason = "P0.2 遗忘：reset_completed 事实键成立，先于断层、除名与历史加权。"
		return s.finishArbitration(decision)
	case fractureFact && !escapeAlive:
		decision.Selected = endingFracture
		decision.Reason = "P0.3 断层：fracture_confirmed 事实键成立，先于除名与历史加权。"
		return s.finishArbitration(decision)
	case trigger == endingCompromised:
		decision.Selected = endingCompromised
		decision.Reason = "P0.4 除名：怀疑度封顶，身份暴露不可挽回。"
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
	if station.Authority.Judicial.Regime == "MAYOR" {
		pool[endingPersonalRule] = true
	} else {
		decision.Excluded[endingPersonalRule] = "Judicial 未附着于现任 Mayor"
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
	decision.Reason = fmt.Sprintf("P2 加权：%s 得分 %.4f 达到最低分 %.4f；同分顺序按固定裁决序。", best, bestScore, config.MinimumScore)
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

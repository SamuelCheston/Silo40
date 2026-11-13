package service

import (
	"fmt"
	"sort"
	"strings"

	"silo40/internal/model"
)

// endingOutcomeWeights keeps every tunable outcome coefficient in one config.
type endingOutcomeWeights struct {
	Information  float64
	Time         float64
	Rebellion    float64
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

// TODO(balance): all values below are placeholders for future measured tuning.
// This round proves the decision skeleton, not the ending distribution.
var defaultEndingWeightConfig = endingWeightConfig{
	Version:       1,
	MinimumScore:  1,
	PrestigeScale: 100,
	PlayerBase:    endingOutcomeWeights{Information: 3, Time: 2, Rebellion: 3, Exclusionist: 1},
	PlayerPerPrestige: endingOutcomeWeights{
		Information: 0.5, Time: 0.2, Rebellion: 0.8, Exclusionist: 0.1,
	},
	PlayerPerFragment:   endingOutcomeWeights{Information: 0.05},
	PlayerPerPropaganda: endingOutcomeWeights{Rebellion: 0.4},
	FactionPerInfluence: endingOutcomeWeights{
		Information: 0.2, Time: 0.1, Rebellion: 0.2, Exclusionist: 0.2,
	},
	FactionPerOpenInfluence: endingOutcomeWeights{
		Information: 0.8, Rebellion: 0.2, Exclusionist: -0.4,
	},
	FactionPerLowCohesion:   endingOutcomeWeights{Rebellion: 1.2, Exclusionist: -0.2},
	ResidentPerInfluence:    endingOutcomeWeights{Information: 0.2, Time: 0.1, Rebellion: 0.2, Exclusionist: 0.2},
	ResidentPerOpenIdeology: endingOutcomeWeights{Information: 0.6, Exclusionist: -0.4},
	ResidentPerAmbition:     endingOutcomeWeights{Rebellion: 0.8},
	Silo1AliveBase:          endingOutcomeWeights{Exclusionist: 2},
	Silo1DestroyedBase:      endingOutcomeWeights{Time: 4},
	Silo1PerRebellion:       endingOutcomeWeights{Time: 2},
	Silo1PerCohesion:        endingOutcomeWeights{Exclusionist: 1},
}

var endingOutcomeOrder = []string{"INFORMATION", "TIME", "REBELLION", "EXCLUSIONIST"}

func outcomeWeight(weights endingOutcomeWeights, outcome string) float64 {
	switch outcome {
	case "INFORMATION":
		return weights.Information
	case "TIME":
		return weights.Time
	case "REBELLION":
		return weights.Rebellion
	case "EXCLUSIONIST":
		return weights.Exclusionist
	default:
		return 0
	}
}

type endingScoreAccumulator struct {
	decision *model.EndingDecision
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
	for _, outcome := range endingOutcomeOrder {
		delta := outcomeWeight(weights, outcome) * scale
		contribution.Scores[outcome] += delta
		a.decision.Scores[outcome] += delta
	}
}

// resolveWeightedEnding converts a non-catastrophic engine ending trigger into
// a multi-entity historical direction. Nil preserves hard failures/no-trigger.
func (s *GameService) resolveWeightedEnding(station *model.Silo, agent *model.Agent) *model.EndingDecision {
	if station == nil || station.VictoryStatus == nil || !station.VictoryStatus.IsWon || agent == nil {
		return nil
	}
	triggered := strings.ToUpper(station.VictoryStatus.Type)
	validTrigger := false
	for _, outcome := range endingOutcomeOrder {
		if outcome == triggered {
			validTrigger = true
			break
		}
	}
	if !validTrigger {
		return nil
	}

	config := defaultEndingWeightConfig
	decision := &model.EndingDecision{
		ConfigVersion: config.Version,
		MinimumScore:  config.MinimumScore,
		Scores:        map[string]float64{},
		Contributions: []model.EndingEntityContribution{},
	}
	for _, outcome := range endingOutcomeOrder {
		decision.Scores[outcome] = 0
	}
	accumulator := &endingScoreAccumulator{decision: decision}
	playerID := fmt.Sprintf("agent:%d", agent.ID)
	accumulator.add("player", playerID, agent.Name, config.PlayerBase, 1)
	prestige := agent.PoliticalPrestige / config.PrestigeScale
	accumulator.add("player", playerID, agent.Name, config.PlayerPerPrestige, prestige)
	accumulator.add("player", playerID, agent.Name, config.PlayerPerFragment, float64(len(agent.KnownFragments)))
	accumulator.add("player", playerID, agent.Name, config.PlayerPerPropaganda, agent.PropagandaLevel)

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

	decision.Selected = triggered
	bestScore := config.MinimumScore
	for _, outcome := range endingOutcomeOrder {
		if decision.Scores[outcome] > bestScore {
			decision.Selected = outcome
			bestScore = decision.Scores[outcome]
		}
	}
	return decision
}

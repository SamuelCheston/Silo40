package engine

import "silo40/internal/model"

const (
	agentAPBaseRecovery = 10.0
	agentAPMax          = 100.0
)

// BuildAgentStats returns the derived values that the UI needs to explain
// current Agent metrics without reimplementing engine formulas in TypeScript.
func (e *GameEngine) BuildAgentStats(agent *model.Agent, silo *model.Silo) model.AgentStats {
	if agent == nil || silo == nil {
		return model.AgentStats{}
	}

	connValues := connectionsByProfession(agent, silo)
	totalConnection := 0.0
	if len(connValues) > 0 {
		for _, value := range connValues {
			totalConnection += value
		}
		totalConnection /= float64(len(connValues))
	}

	professionFactor := e.professionFactors[agent.Profession]
	traitFactor := 0.0
	for _, trait := range agent.Traits {
		traitFactor += e.traitFactors[trait]
	}

	powerLevel := 0
	classType := ""
	for _, profession := range silo.Professions {
		if profession.Name == agent.Profession {
			powerLevel = profession.PowerLevel
			classType = profession.ClassType
			break
		}
	}

	prestigeBase := totalConnection * 100
	structuralPrestige := float64(powerLevel)*1.5 + classBias(classType, 2.0)
	prestige := prestigeBase*(1+professionFactor)*(1+traitFactor) + structuralPrestige
	if prestige < 0 {
		prestige = 0
	}

	apPrestigeBonus := prestige * 0.05
	propagandaMultiplier := 1 + (agent.PropagandaLevel * 0.2)

	return model.AgentStats{
		AvgConnection:        totalConnection,
		PrestigeBase:         prestigeBase,
		ProfessionFactor:     professionFactor,
		TraitFactor:          traitFactor,
		StructuralPrestige:   structuralPrestige,
		PoliticalPrestige:    prestige,
		APBaseRecovery:       agentAPBaseRecovery,
		APPrestigeBonus:      apPrestigeBonus,
		APTotalRecovery:      agentAPBaseRecovery + apPrestigeBonus,
		APMax:                agentAPMax,
		PropagandaLevel:      agent.PropagandaLevel,
		PropagandaMultiplier: propagandaMultiplier,
		RebellionBaseEffect:  0.05 + (prestige * 0.002),
	}
}

func connectionsByProfession(agent *model.Agent, silo *model.Silo) []float64 {
	if agent == nil || silo == nil {
		return nil
	}
	out := make([]float64, 0, len(silo.Professions))
	for _, profession := range silo.Professions {
		value := 0.0
		for _, connection := range agent.Connections {
			if connection.ProfessionID == profession.ID {
				value = connection.Value
				break
			}
		}
		out = append(out, value)
	}
	return out
}

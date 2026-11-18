package engine

import "silo40/internal/model"

const (
	// agentAPBaseRecovery 必须与 core_systems.js agent_update 的实际恢复公式
	// (年恢复 36 + 威望*0.05，TODO(balance)) 保持一致：展示值不能比实际经济
	// 多一个量级，否则玩家会按错误的预算规划流通节奏。
	agentAPBaseRecovery = 36.0
	agentAPMax          = 100.0
)

// BuildAgentStats returns the derived values that the UI needs to explain
// current Agent metrics without reimplementing engine formulas in TypeScript.
func (e *GameEngine) BuildAgentStats(agent *model.Agent, silo *model.Silo) model.AgentStats {
	if agent == nil || silo == nil {
		return model.AgentStats{}
	}
	if e.MechanicEngine != nil {
		if def, ok := e.MechanicEngine.FormulaDefinition("agent_stats"); ok {
			actor, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
			if err == nil {
				result, runErr := e.MechanicEngine.Run(def, MechanicContext{
					Silo:  silo,
					Agent: agent,
					Actor: actor,
				})
				if runErr == nil && result.Stats != nil {
					return *result.Stats
				}
			}
		}
	}

	connValues := connectionsByProfession(agent, silo)
	totalConnection := 0.0
	if len(connValues) > 0 {
		for _, value := range connValues {
			totalConnection += value
		}
		totalConnection /= float64(len(connValues))
	}

	// 威望全层派生 (24 篇 §2)：50×(Ambition+Influence)，展示同口径。
	prestige := DerivedPrestige(agent.Ambition, agent.Influence)

	apPrestigeBonus := prestige * 0.05
	propagandaMultiplier := 1 + (agent.PropagandaLevel * 0.2)

	isFactionLeader := false
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if res.Alive && res.Profession == agent.Profession && res.IsRepresentative {
			isFactionLeader = true
			break
		}
	}

	return model.AgentStats{
		AvgConnection:        totalConnection,
		Ambition:             agent.Ambition,
		Influence:            agent.Influence,
		PoliticalPrestige:    prestige,
		APBaseRecovery:       agentAPBaseRecovery,
		APPrestigeBonus:      apPrestigeBonus,
		APTotalRecovery:      agentAPBaseRecovery + apPrestigeBonus,
		APMax:                agentAPMax,
		PropagandaLevel:      agent.PropagandaLevel,
		PropagandaMultiplier: propagandaMultiplier,
		RebellionBaseEffect:  0.05 + (prestige * 0.002),
		IsFactionLeader:      isFactionLeader,
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

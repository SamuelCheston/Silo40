package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

func TestBuildAgentStatsMatchesEngineFormula(t *testing.T) {
	eng := NewGameEngine()
	silo := &model.Silo{
		Professions: []model.Profession{
			{ID: 1, Name: "Mayor", ClassType: "ELITE", PowerLevel: 10},
			{ID: 2, Name: "Mechanical", ClassType: "COMMONER", PowerLevel: 6},
		},
	}
	agent := &model.Agent{
		Profession:      "Mayor",
		Traits:          []string{"一号地堡密使", "守旧派"},
		PropagandaLevel: 2,
		Ambition:        0.8,
		Influence:       0.6,
		Connections: []model.Connection{
			{ProfessionID: 1, Value: 0.8},
			{ProfessionID: 2, Value: 0.4},
		},
	}

	stats := eng.BuildAgentStats(agent, silo)

	assertClose(t, "avg connection", stats.AvgConnection, 0.6)
	assertClose(t, "ambition", stats.Ambition, 0.8)
	assertClose(t, "influence", stats.Influence, 0.6)
	// 威望全层派生：50×(0.8+0.6) = 70。
	assertClose(t, "political prestige", stats.PoliticalPrestige, 70)
	assertClose(t, "ap prestige bonus", stats.APPrestigeBonus, 3.5)
	// 修复2：展示口径必须与实际恢复公式一致 (年 36 + 威望*0.05)，
	// 不再是旧经济的 10 + 威望*0.05。
	assertClose(t, "ap base recovery", stats.APBaseRecovery, 36)
	assertClose(t, "ap total recovery", stats.APTotalRecovery, 39.5)
	assertClose(t, "propaganda multiplier", stats.PropagandaMultiplier, 1.4)
	assertClose(t, "rebellion base effect", stats.RebellionBaseEffect, 0.19)
}

// TestDisplayedAPRecoveryMatchesActualRecovery locks 修复2 against drift: the
// recovery the UI explains (agent_stats formula) must equal, to the digit,
// what agent_update actually grants. The old display (10/yr) survived the
// economy re-anchor (36/yr) and told players to plan around a budget four
// times larger than the real one.
func TestDisplayedAPRecoveryMatchesActualRecovery(t *testing.T) {
	eng := NewGameEngine()
	if eng.MechanicEngine == nil {
		t.Fatal("mechanic engine unavailable")
	}
	if _, ok := eng.MechanicEngine.EventDefinition(EVENT_AGENT_UPDATE); !ok {
		t.Fatal("agent_update mechanic not loaded from events/mechanics")
	}
	silo := &model.Silo{
		Professions: []model.Profession{
			{ID: 1, Name: "Mayor", ClassType: "ELITE", PowerLevel: 10},
		},
	}
	agent := &model.Agent{
		Profession:   "Mayor",
		ActionPoints: 20,
		Connections:  []model.Connection{{ProfessionID: 1, Value: 0.5}},
	}
	stats := eng.BuildAgentStats(agent, silo)

	before := agent.ActionPoints
	eng.UpdateAgentState(agent, 1.0, silo, nil)
	actual := agent.ActionPoints - before

	assertClose(t, "displayed yearly recovery vs actual agent_update recovery", stats.APTotalRecovery, actual)
}

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("%s mismatch: got %.6f want %.6f", label, got, want)
	}
}

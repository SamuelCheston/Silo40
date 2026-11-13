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
		Connections: []model.Connection{
			{ProfessionID: 1, Value: 0.8},
			{ProfessionID: 2, Value: 0.4},
		},
	}

	stats := eng.BuildAgentStats(agent, silo)

	assertClose(t, "avg connection", stats.AvgConnection, 0.6)
	assertClose(t, "prestige base", stats.PrestigeBase, 60)
	assertClose(t, "profession factor", stats.ProfessionFactor, 0.5)
	assertClose(t, "trait factor", stats.TraitFactor, 0.4)
	assertClose(t, "structural prestige", stats.StructuralPrestige, 17)
	assertClose(t, "political prestige", stats.PoliticalPrestige, 143)
	assertClose(t, "ap prestige bonus", stats.APPrestigeBonus, 7.15)
	assertClose(t, "ap total recovery", stats.APTotalRecovery, 17.15)
	assertClose(t, "propaganda multiplier", stats.PropagandaMultiplier, 1.4)
	assertClose(t, "rebellion base effect", stats.RebellionBaseEffect, 0.336)
}

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("%s mismatch: got %.6f want %.6f", label, got, want)
	}
}

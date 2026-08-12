package engine

import (
	"testing"

	"silo40/internal/model"
)

func TestFactionGroupingIgnoresLoyaltyAndRequiresAmbition(t *testing.T) {
	silo := &model.Silo{
		ID:         1,
		Legitimacy: 0.65,
		Cohesion:   0.60,
		Professions: []model.Profession{
			{
				ID:         1,
				Name:       "Mechanical",
				ClassType:  "COMMONER",
				PowerLevel: 8,
				Zone:       "Lower",
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyDemocracy:  0.80,
					model.IdeologyLoyalty:    0.35,
				},
				Relations: map[string]float64{"Mechanical": 1.0, "Supply": 0.60, "Mines": 0.40},
			},
			{
				ID:         2,
				Name:       "Supply",
				ClassType:  "COMMONER",
				PowerLevel: 7,
				Zone:       "Mid",
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.78,
					model.IdeologyDemocracy:  0.75,
					model.IdeologyLoyalty:    0.80,
				},
				Relations: map[string]float64{"Supply": 1.0, "Mechanical": 0.60, "Mines": 0.40},
			},
			{
				ID:         3,
				Name:       "Mines",
				ClassType:  "COMMONER",
				PowerLevel: 4,
				Zone:       "Lower",
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.76,
					model.IdeologyDemocracy:  0.78,
					model.IdeologyLoyalty:    0.20,
				},
				Relations: map[string]float64{"Mines": 1.0, "Mechanical": 0.40, "Supply": 0.40},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Mechanical Bloc",
				Count:           260,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyDemocracy + ":High", model.IdeologyLoyalty + ":Low"},
				Ambition:        0.82,
				Influence:       0.72,
				ActionPoints:    35,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyDemocracy:  0.80,
					model.IdeologyLoyalty:    0.25,
				},
				PanicSensitivity: 1.0,
			},
			{
				ID:              2,
				ProfessionID:    2,
				Name:            "Supply Bloc",
				Count:           260,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyDemocracy + ":High", model.IdeologyLoyalty + ":High"},
				Ambition:        0.79,
				Influence:       0.69,
				ActionPoints:    34,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.78,
					model.IdeologyDemocracy:  0.75,
					model.IdeologyLoyalty:    0.82,
				},
				PanicSensitivity: 1.0,
			},
			{
				ID:              3,
				ProfessionID:    3,
				Name:            "Mines Crew",
				Count:           260,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyDemocracy + ":High"},
				Ambition:        0.30,
				Influence:       0.42,
				ActionPoints:    20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.76,
					model.IdeologyDemocracy:  0.78,
					model.IdeologyLoyalty:    0.20,
				},
				PanicSensitivity: 1.0,
			},
		},
	}

	for i := range silo.Cohorts {
		prof := getProfessionByID(silo, silo.Cohorts[i].ProfessionID)
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}

	RebuildImplicitFactions(silo)

	if len(silo.Factions) != 2 {
		t.Fatalf("expected 2 factions, got %d", len(silo.Factions))
	}

	if silo.Cohorts[0].FactionID == nil || silo.Cohorts[1].FactionID == nil {
		t.Fatalf("expected ambitious cohorts to be assigned to political factions")
	}
	if *silo.Cohorts[0].FactionID != *silo.Cohorts[1].FactionID {
		t.Fatalf("expected cohorts with different loyalty but same political profile to share a faction")
	}
	if silo.Cohorts[2].FactionID == nil || *silo.Cohorts[2].FactionID == *silo.Cohorts[0].FactionID {
		t.Fatalf("expected low-ambition cohort to stay out of the political faction")
	}

	var political, unaffiliated *model.Faction
	for i := range silo.Factions {
		faction := &silo.Factions[i]
		if faction.Signature == "special:unaffiliated" {
			unaffiliated = faction
		} else {
			political = faction
		}
	}

	if political == nil || unaffiliated == nil {
		t.Fatalf("expected both a political faction and an unaffiliated faction")
	}
	if political.MemberCount != 520 {
		t.Fatalf("expected merged political faction size 520, got %d", political.MemberCount)
	}
	if unaffiliated.MemberCount != 260 {
		t.Fatalf("expected unaffiliated size 260, got %d", unaffiliated.MemberCount)
	}
	if political.Signature != "democracy:High|pro_foreign:High|drive:ambition" {
		t.Fatalf("unexpected political signature: %s", political.Signature)
	}
}

func TestFactionRebuildAppliesPostGroupingLoyaltyDynamics(t *testing.T) {
	silo := &model.Silo{
		ID:         1,
		Legitimacy: 0.20,
		Cohesion:   0.35,
		Rebellion:  0.30,
		Professions: []model.Profession{
			{
				ID:         1,
				Name:       "Mechanical",
				ClassType:  "COMMONER",
				PowerLevel: 8,
				Zone:       "Lower",
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.80,
					model.IdeologyDemocracy:  0.82,
					model.IdeologyLoyalty:    0.55,
				},
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Restless Mechanics",
				Count:           320,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyDemocracy + ":High"},
				Ambition:        0.92,
				Influence:       0.76,
				ActionPoints:    38,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.80,
					model.IdeologyDemocracy:  0.82,
					model.IdeologyLoyalty:    0.58,
				},
				PanicSensitivity: 1.0,
			},
		},
	}

	prof := &silo.Professions[0]
	silo.Cohorts[0].Tags = buildCohortTags(&silo.Cohorts[0], prof, silo)

	before := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]
	RebuildImplicitFactions(silo)
	after := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]

	if after >= before {
		t.Fatalf("expected loyalty to drop after political faction mobilization, before=%.3f after=%.3f", before, after)
	}
	if len(silo.Factions) != 1 {
		t.Fatalf("expected one political faction, got %d", len(silo.Factions))
	}
	if silo.Factions[0].Ambition < POLITICAL_AMBITION_THRESHOLD {
		t.Fatalf("expected resulting faction ambition to exceed threshold, got %.2f", silo.Factions[0].Ambition)
	}
}

package engine

import (
	"testing"

	"silo40/internal/model"
)

func TestFactionGroupingIgnoresLoyaltyWhenPoliticalProfileMatches(t *testing.T) {
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
					model.IdeologyDemocracy:  0.20,
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
				IdeologyProfile: []string{model.IdeologyProForeign + ":High"},
				Influence:       0.42,
				ActionPoints:    20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.76,
					model.IdeologyDemocracy:  0.20,
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
		t.Fatalf("expected paired-ideology cohorts to be assigned to political factions")
	}
	if *silo.Cohorts[0].FactionID != *silo.Cohorts[1].FactionID {
		t.Fatalf("expected cohorts with different loyalty but same political profile to share a faction")
	}
	if silo.Cohorts[2].FactionID == nil || *silo.Cohorts[2].FactionID == *silo.Cohorts[0].FactionID {
		t.Fatalf("expected single-axis cohort to stay out of the political faction")
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
	if political.Signature != "democracy:High|pro_foreign:High|drive:solidarity" {
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
		t.Fatalf("expected loyalty to drop after political faction pressure, before=%.3f after=%.3f", before, after)
	}
	if len(silo.Factions) != 1 {
		t.Fatalf("expected one political faction, got %d", len(silo.Factions))
	}
}

func TestSingleIdeologyCannotFormPoliticalFaction(t *testing.T) {
	silo := &model.Silo{
		ID:         1,
		Legitimacy: 0.55,
		Cohesion:   0.55,
		Professions: []model.Profession{
			{
				ID:         1,
				Name:       "Mechanical",
				ClassType:  "COMMONER",
				PowerLevel: 8,
				Zone:       "Lower",
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.82,
					model.IdeologyDemocracy:  0.10,
					model.IdeologyLoyalty:    0.45,
				},
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Single-Axis Bloc",
				Count:           340,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyLoyalty + ":Medium"},
				Influence:       0.74,
				ActionPoints:    32,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.82,
					model.IdeologyDemocracy:  0.10,
					model.IdeologyLoyalty:    0.45,
				},
				PanicSensitivity: 1.0,
			},
		},
	}

	prof := &silo.Professions[0]
	silo.Cohorts[0].Tags = buildCohortTags(&silo.Cohorts[0], prof, silo)

	RebuildImplicitFactions(silo)

	if len(silo.Factions) != 1 {
		t.Fatalf("expected only unaffiliated faction, got %d factions", len(silo.Factions))
	}
	if silo.Factions[0].Signature != "special:unaffiliated" {
		t.Fatalf("expected single-ideology cohort to stay unaffiliated, got %s", silo.Factions[0].Signature)
	}
	if silo.Factions[0].Influence != 0 {
		t.Fatalf("expected unaffiliated faction to have zero political influence, got %.2f", silo.Factions[0].Influence)
	}
}

func TestPoliticalFormationTier(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "medium high qualifies",
			tags: []string{model.IdeologyDemocracy + ":Medium", model.IdeologyProForeign + ":High"},
			want: "formation:medium_high",
		},
		{
			name: "high high qualifies",
			tags: []string{model.IdeologyDemocracy + ":High", model.IdeologyProForeign + ":High"},
			want: "formation:high_high",
		},
		{
			name: "medium medium does not qualify",
			tags: []string{model.IdeologyDemocracy + ":Medium", model.IdeologyProForeign + ":Medium"},
			want: "",
		},
		{
			name: "single axis does not qualify",
			tags: []string{model.IdeologyProForeign + ":High"},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := politicalFormationTier(tc.tags)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRepresentativeResidentGetsNativeFixedAmbition(t *testing.T) {
	silo := &model.Silo{
		ID: 1,
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
					model.IdeologyLoyalty:    0.55,
				},
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:                1,
				ProfessionID:      1,
				Name:              "Mechanics",
				Count:             300,
				IdeologyProfile:   []string{model.IdeologyProForeign + ":High", model.IdeologyDemocracy + ":High"},
				Influence:         0.72,
				ActionPoints:      35,
				PoliticalPrestige: 18,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyDemocracy:  0.80,
					model.IdeologyLoyalty:    0.55,
				},
				PanicSensitivity: 1.0,
			},
		},
	}

	rep := ensureRepresentativeResident(silo, &silo.Cohorts[0])
	if rep == nil {
		t.Fatalf("expected representative resident to be created")
	}
	if rep.Ambition <= 0 {
		t.Fatalf("expected representative resident ambition to be generated")
	}

	before := rep.Ambition
	silo.Cohorts[0].Influence = 0.10
	silo.Cohorts[0].PoliticalPrestige = 90
	silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = 0.10
	updateKeyResidents(silo, 1.0)

	if rep.Ambition != before {
		t.Fatalf("expected resident ambition to remain fixed, before=%.3f after=%.3f", before, rep.Ambition)
	}
}

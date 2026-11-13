package engine

import (
	"strings"
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
					model.IdeologyProForeign: 0.10,
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
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyLoyalty + ":Low"},
				Influence:       0.72,
				ActionPoints:    35,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyLoyalty:    0.25,
				},
				PanicSensitivity: 1.0,
			},
			{
				ID:              2,
				ProfessionID:    2,
				Name:            "Supply Bloc",
				Count:           260,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyLoyalty + ":High"},
				Influence:       0.69,
				ActionPoints:    34,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.78,
					model.IdeologyLoyalty:    0.82,
				},
				PanicSensitivity: 1.0,
			},
			{
				ID:              3,
				ProfessionID:    3,
				Name:            "Mines Crew",
				Count:           260,
				IdeologyProfile: []string{model.IdeologyLoyalty + ":Low"},
				Influence:       0.42,
				ActionPoints:    20,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.10,
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
	if political.Signature != "pro_foreign:High|drive:solidarity" {
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
				IdeologyProfile: []string{model.IdeologyProForeign + ":High"},
				Influence:       0.76,
				ActionPoints:    38,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.80,
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
	if len(silo.Factions) != 1 {
		t.Fatalf("expected one political faction, got %d", len(silo.Factions))
	}
	// Rebuild is grouping-only: it must not mutate loyalty on its own.
	if got := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]; got != before {
		t.Fatalf("rebuild must not change loyalty, before=%.3f after=%.3f", before, got)
	}
	// Zero-duration step (instant action resolution): no evolution either.
	ApplyFactionLoyaltyDynamics(silo, 0)
	if got := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]; got != before {
		t.Fatalf("zero-duration dynamics must be a no-op, before=%.3f after=%.3f", before, got)
	}
	// One monthly step applies the post-grouping pressure.
	ApplyFactionLoyaltyDynamics(silo, 1.0/12.0)
	after := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]

	if after >= before {
		t.Fatalf("expected loyalty to drop after political faction pressure, before=%.3f after=%.3f", before, after)
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
					model.IdeologyProForeign: 0.10,
					model.IdeologyLoyalty:    0.45,
				},
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Status-Quo Bloc",
				Count:           340,
				IdeologyProfile: []string{model.IdeologyLoyalty + ":Medium"},
				Influence:       0.74,
				ActionPoints:    32,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.10,
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
		t.Fatalf("expected non-political cohort to stay unaffiliated, got %s", silo.Factions[0].Signature)
	}
}

func TestPoliticalFormationTier(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "high qualifies",
			tags: []string{model.IdeologyProForeign + ":High"},
			want: "formation:high",
		},
		{
			name: "medium qualifies",
			tags: []string{model.IdeologyProForeign + ":Medium"},
			want: "formation:medium",
		},
		{
			name: "no political tags does not qualify",
			tags: []string{"ideology:neutral"},
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
				IdeologyProfile:   []string{model.IdeologyProForeign + ":High"},
				Influence:         0.72,
				ActionPoints:      35,
				PoliticalPrestige: 18,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
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

// TestPoliticalGateHysteresisPreventsFlipFlop locks the entry/exit band: a
// cohort politicizes at pro_foreign >= 0.7, stays political while >= 0.6,
// and must climb back to 0.7 to re-enter after falling below 0.6. Without
// the band, cohorts hovering near the gate flip between political and
// unaffiliated every tick (audit observed 4 flips in 5 months).
func TestPoliticalGateHysteresisPreventsFlipFlop(t *testing.T) {
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
					model.IdeologyProForeign: 0.70,
					model.IdeologyLoyalty:    0.50,
				},
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Hovering Mechanics",
				Count:           320,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High"},
				Influence:       0.50,
				ActionPoints:    30,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.72,
					model.IdeologyLoyalty:    0.50,
				},
				PanicSensitivity: 1.0,
			},
		},
	}

	politicalFactions := func() int {
		n := 0
		for i := range silo.Factions {
			if silo.Factions[i].Signature != "special:unaffiliated" {
				n++
			}
		}
		return n
	}
	setPF := func(v float64) {
		silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = v
	}

	// 0.72 >= entry 0.7 -> politicizes.
	setPF(0.72)
	RebuildImplicitFactions(silo)
	if politicalFactions() != 1 {
		t.Fatalf("pf=0.72: expected cohort to politicize, factions=%d", politicalFactions())
	}

	// Drops into the hysteresis band [0.6, 0.7) -> stays political.
	setPF(0.65)
	RebuildImplicitFactions(silo)
	if politicalFactions() != 1 {
		t.Fatalf("pf=0.65: expected cohort to stay political inside the band, factions=%d", politicalFactions())
	}

	// Falls below exit 0.6 -> depoliticizes.
	setPF(0.55)
	RebuildImplicitFactions(silo)
	if politicalFactions() != 0 {
		t.Fatalf("pf=0.55: expected cohort to depoliticize below exit threshold, factions=%d", politicalFactions())
	}

	// Back into the band from below -> still unaffiliated (entry needs 0.7).
	setPF(0.65)
	RebuildImplicitFactions(silo)
	if politicalFactions() != 0 {
		t.Fatalf("pf=0.65 from below: expected cohort to stay unaffiliated until entry threshold, factions=%d", politicalFactions())
	}

	// Reaches entry again -> politicizes.
	setPF(0.72)
	RebuildImplicitFactions(silo)
	if politicalFactions() != 1 {
		t.Fatalf("pf=0.72 again: expected cohort to re-politicize, factions=%d", politicalFactions())
	}
}

// TestFactionIdentityStableAcrossRebuilds verifies P1-S3 persistent identity:
// lineage factions keep their ID/name/memory across rebuilds, and a size
// inversion swaps ranks without swapping identities.
func TestFactionIdentityStableAcrossRebuilds(t *testing.T) {
	silo := &model.Silo{
		ID:         1,
		Legitimacy: 0.60,
		Cohesion:   0.60,
		Professions: []model.Profession{
			{
				ID:        1,
				Name:      "Mechanical",
				ClassType: "COMMONER",
				Zone:      "Lower",
				Relations: map[string]float64{"Mechanical": 1.0, "Supply": 0.60},
			},
			{
				ID:        2,
				Name:      "Supply",
				ClassType: "COMMONER",
				Zone:      "Mid",
				Relations: map[string]float64{"Supply": 1.0, "Mechanical": 0.60},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Mechanical Bloc",
				Count:           400,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyLoyalty + ":Low"},
				Influence:       0.70,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.90,
					model.IdeologyLoyalty:    0.20,
				},
			},
			{
				ID:              2,
				ProfessionID:    2,
				Name:            "Supply Bloc",
				Count:           400,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High", model.IdeologyLoyalty + ":High"},
				Influence:       0.70,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.72,
					model.IdeologyLoyalty:    0.85,
				},
			},
		},
	}
	for i := range silo.Cohorts {
		prof := getProfessionByID(silo, silo.Cohorts[i].ProfessionID)
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}

	factionOf := func(cohortID uint) *model.Faction {
		for i := range silo.Cohorts {
			if silo.Cohorts[i].ID != cohortID || silo.Cohorts[i].FactionID == nil {
				continue
			}
			for j := range silo.Factions {
				if silo.Factions[j].ID == *silo.Cohorts[i].FactionID {
					return &silo.Factions[j]
				}
			}
		}
		return nil
	}

	RebuildImplicitFactions(silo)
	if len(silo.Factions) != 2 {
		t.Fatalf("expected 2 political factions, got %d", len(silo.Factions))
	}
	fa, fb := factionOf(1), factionOf(2)
	if fa == nil || fb == nil || fa.ID == fb.ID {
		t.Fatalf("expected two distinct factions for the two distant cohorts")
	}
	idA, idB := fa.ID, fb.ID
	nameA, nameB := fa.Name, fb.Name
	memA := append([]string(nil), fa.Memory...)
	for _, f := range []*model.Faction{fa, fb} {
		if f.Stance == "" {
			t.Fatalf("faction %d: expected non-empty stance", f.ID)
		}
		if len(f.Demands) == 0 {
			t.Fatalf("faction %d: expected non-empty demands", f.ID)
		}
		if len(f.Memory) == 0 || !strings.HasPrefix(f.Memory[0], "founded:") {
			t.Fatalf("faction %d: expected founding memory, got %v", f.ID, f.Memory)
		}
	}

	// Small ideological drift: identity (ID/name/memory) must not change.
	silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = 0.89
	RebuildImplicitFactions(silo)
	fa, fb = factionOf(1), factionOf(2)
	if fa == nil || fb == nil {
		t.Fatalf("expected both factions to survive the drift rebuild")
	}
	if fa.ID != idA || fb.ID != idB {
		t.Fatalf("expected stable IDs after drift, got %d/%d want %d/%d", fa.ID, fb.ID, idA, idB)
	}
	if fa.Name != nameA || fb.Name != nameB {
		t.Fatalf("expected stable names after drift, got %q/%q", fa.Name, fb.Name)
	}
	if len(fa.Memory) != len(memA) || fa.Memory[0] != memA[0] {
		t.Fatalf("expected inherited memory after drift, got %v want %v", fa.Memory, memA)
	}

	// Size inversion: rank swaps, but identity follows the ID, not the rank.
	silo.Cohorts[1].Count = 600
	RebuildImplicitFactions(silo)
	if len(silo.Factions) != 2 {
		t.Fatalf("expected 2 factions after size inversion, got %d", len(silo.Factions))
	}
	if silo.Factions[0].ID != idB || silo.Factions[1].ID != idA {
		t.Fatalf("expected size-descending rank with persistent IDs, got [%d %d] want [%d %d]",
			silo.Factions[0].ID, silo.Factions[1].ID, idB, idA)
	}
	if silo.Factions[0].Name != nameB || silo.Factions[1].Name != nameA {
		t.Fatalf("expected names to follow identity after rank swap, got %q/%q",
			silo.Factions[0].Name, silo.Factions[1].Name)
	}
}

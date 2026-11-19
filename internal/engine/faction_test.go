package engine

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// C7 审计基线：距离聚类已废除——高亲外思潮的 cohort 不再自动政治化，
// 零发起则零派系 (仅无阵营桶)。
func TestMaintainFactionsDoesNotCluster(t *testing.T) {
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
				Relations: map[string]float64{"Mechanical": 1.0, "Supply": 0.60},
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
				Relations: map[string]float64{"Supply": 1.0, "Mechanical": 0.60},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:           1,
				ProfessionID: 1,
				Name:         "Mechanical Bloc",
				Count:        260,
				Influence:    0.72,
				ActionPoints: 35,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyLoyalty:    0.25,
				},
			},
			{
				ID:           2,
				ProfessionID: 2,
				Name:         "Supply Bloc",
				Count:        260,
				Influence:    0.69,
				ActionPoints: 34,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.78,
					model.IdeologyLoyalty:    0.82,
				},
			},
		},
	}

	MaintainFactions(silo, nil)

	if len(silo.Factions) != 1 || silo.Factions[0].Signature != unaffiliatedSignature {
		t.Fatalf("zero founding must yield only the unaffiliated bucket, got %+v", silo.Factions)
	}
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		if c.FactionID == nil || *c.FactionID != silo.Factions[0].ID {
			t.Fatalf("cohort %d must sit in the unaffiliated bucket: %v", c.ID, c.FactionID)
		}
	}
	if silo.Factions[0].MemberCount != 520 {
		t.Fatalf("unaffiliated bucket must aggregate all population, got %d", silo.Factions[0].MemberCount)
	}
	// 每个 cohort 都有代表 (人口算法不变)，但无派系代表标志。
	if len(silo.Residents) != 2 {
		t.Fatalf("expected one delegate per cohort, got %d", len(silo.Residents))
	}
	for _, r := range silo.Residents {
		if r.IsRepresentative {
			t.Fatalf("delegate without a faction must not carry the representative flag: %+v", r)
		}
	}
}

// 忠诚动力学保留：维护管线只分组不演化；月度步骤施加派系压力。
func TestFactionMaintenanceAppliesPostGroupingLoyaltyDynamics(t *testing.T) {
	silo := &model.Silo{
		ID:           1,
		Legitimacy:   0.20,
		Cohesion:     0.35,
		Rebellion:    0.30,
		CurrentYear:  122,
		CurrentMonth: 5,
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
				ID:           1,
				ProfessionID: 1,
				Name:         "Restless Mechanics",
				Count:        320,
				Influence:    0.76,
				ActionPoints: 38,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.80,
					model.IdeologyLoyalty:    0.58,
				},
			},
		},
		Residents: []model.Resident{
			{
				ID: 1, Alive: true, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical",
				Ambition: 0.7, Influence: 0.6,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.8, model.IdeologyLoyalty: 0.58},
				Relations:  map[string]float64{"Mechanical": 1.0},
			},
		},
	}
	prof := &silo.Professions[0]
	silo.Cohorts[0].Tags = buildCohortTags(&silo.Cohorts[0], prof, silo)

	// 模拟已发起并加入的派系 (C9 前的手工成员资格)。
	factionID := uint(7)
	silo.Factions = []model.Faction{{
		ID: factionID, SiloID: 1, Name: "发起的派系",
		Signature: "founded:RESIDENT:1|" + model.IdeologyProForeign + ":High",
		Tags:      []string{model.IdeologyProForeign + ":High", "stance:dissent"},
		TagStats:  map[string]int{},
		Stance:    "stance:dissent",
		Memory:    []string{"founded:Y122M5"},
		FoundedYear: 122, FoundedMonth: 5,
		RepresentativeKind: string(ActorResident), RepresentativeID: 1, RepresentativeName: "Rep",
	}}
	silo.Cohorts[0].FactionID = uintPtr(factionID)
	silo.Residents[0].FactionID = uintPtr(factionID)

	before := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]
	MaintainFactions(silo, nil)
	if got := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]; got != before {
		t.Fatalf("maintenance must not change loyalty, before=%.3f after=%.3f", before, got)
	}
	ApplyFactionLoyaltyDynamics(silo, 0)
	if got := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]; got != before {
		t.Fatalf("zero-duration dynamics must be a no-op, before=%.3f after=%.3f", before, got)
	}
	ApplyFactionLoyaltyDynamics(silo, 1.0/12.0)
	after := silo.Cohorts[0].Ideologies[model.IdeologyLoyalty]
	if after >= before {
		t.Fatalf("expected loyalty to drop after faction pressure, before=%.3f after=%.3f", before, after)
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
				ID:           1,
				ProfessionID: 1,
				Name:         "Mechanics",
				Count:        300,
				Influence:    0.72,
				ActionPoints: 35,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.75,
					model.IdeologyLoyalty:    0.55,
				},
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
	silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = 0.10
	updateKeyResidents(silo, 1.0)

	if rep.Ambition != before {
		t.Fatalf("expected resident ambition to remain fixed, before=%.3f after=%.3f", before, rep.Ambition)
	}
}

// C7 成员资格持久：思潮漂移出旧聚类半径也不翻转归属 (无单月往返)。
func TestMembershipPersistentAcrossIdeologyDrift(t *testing.T) {
	silo := &model.Silo{
		ID:           1,
		Legitimacy:   0.55,
		Cohesion:     0.55,
		CurrentYear:  122,
		CurrentMonth: 5,
		Professions: []model.Profession{
			{
				ID:         1,
				Name:       "Mechanical",
				ClassType:  "COMMONER",
				PowerLevel: 4,
				Zone:       "Lower",
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.70, model.IdeologyLoyalty: 0.50},
				Relations:  map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:           1,
				ProfessionID: 1,
				Name:         "Hovering Mechanics",
				Count:        320,
				Influence:    0.50,
				ActionPoints: 30,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.72, model.IdeologyLoyalty: 0.50},
			},
		},
	}
	factionID := uint(3)
	silo.Factions = []model.Faction{{
		ID: factionID, SiloID: 1, Name: "持久派系",
		Signature: "founded:PLAYER:42|" + model.IdeologyProForeign + ":High",
		Tags:      []string{model.IdeologyProForeign + ":High"},
		TagStats:  map[string]int{},
		Memory:    []string{"founded:Y122M5"},
		FoundedYear: 122, FoundedMonth: 5,
		RepresentativeKind: string(ActorPlayer), RepresentativeID: 42,
	}}
	silo.Cohorts[0].FactionID = uintPtr(factionID)

	for _, pf := range []float64{0.65, 0.55, 0.65, 0.72} {
		silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = pf
		MaintainFactions(silo, nil)
		if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != factionID {
			t.Fatalf("pf=%.2f: membership must persist, got %v", pf, silo.Cohorts[0].FactionID)
		}
	}
}

// C7 持久身份：发起派系的 ID/名称/立场/ founding 记忆跨维护不变，
// 成员规模变化只改统计，不改身份。
func TestFactionIdentityStableAcrossMaintenance(t *testing.T) {
	silo := &model.Silo{
		ID:           1,
		Legitimacy:   0.60,
		Cohesion:     0.60,
		CurrentYear:  122,
		CurrentMonth: 5,
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
				ID:           1,
				ProfessionID: 1,
				Name:         "Mechanical Bloc",
				Count:        400,
				Influence:    0.70,
				Ideologies: map[string]float64{
					model.IdeologyProForeign: 0.90,
					model.IdeologyLoyalty:    0.20,
				},
			},
			{
				ID:           2,
				ProfessionID: 2,
				Name:         "Supply Bloc",
				Count:        400,
				Influence:    0.70,
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

	agent := &model.Agent{ID: 42, Name: "Founder", Profession: "Mechanical", Ambition: 0.8, Influence: 0.6, ActionPoints: 30}
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, CreateActorRefForAgent(agent, silo), MechanicMutation{Type: "faction_found_actor"})
	if len(silo.Factions) != 1 {
		t.Fatalf("expected the founded faction, got %d", len(silo.Factions))
	}
	founded := silo.Factions[0]
	id, name, stance := founded.ID, founded.Name, founded.Stance
	mem := append([]string(nil), founded.Memory...)

	// 两个 cohort 加入 (C9 前的手工成员资格)，随后思潮漂移 + 规模反转。
	for i := range silo.Cohorts {
		silo.Cohorts[i].FactionID = uintPtr(id)
	}
	silo.Cohorts[0].Ideologies[model.IdeologyProForeign] = 0.89
	silo.Cohorts[1].Count = 600
	MaintainFactions(silo, nil)

	var f *model.Faction
	for i := range silo.Factions {
		if silo.Factions[i].ID == id {
			f = &silo.Factions[i]
		}
	}
	if f == nil {
		t.Fatal("founded faction must survive maintenance")
	}
	if f.Name != name || f.Stance != stance {
		t.Fatalf("identity drifted: name=%q stance=%q want %q/%q", f.Name, f.Stance, name, stance)
	}
	if len(f.Memory) != len(mem) || f.Memory[0] != mem[0] {
		t.Fatalf("founding memory changed: %v want %v", f.Memory, mem)
	}
	if f.MemberCount != 1000 {
		t.Fatalf("member count must track cohorts, got %d", f.MemberCount)
	}
	if !strings.HasPrefix(f.Signature, foundedSignaturePrefix) {
		t.Fatalf("signature vocabulary lost: %q", f.Signature)
	}
}
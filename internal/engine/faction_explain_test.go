package engine

// C7 可解释归属测试：解释接口只读、自洽；距离只是亲缘参考，
// 归属由发起/加入事件确立 (无惯性保留语义)。

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// explainTestSilo 两个发起派系 (各有一个成员 cohort) + 一个无阵营 cohort。
func explainTestSilo() *model.Silo {
	silo := &model.Silo{
		ID:           1,
		Legitimacy:   0.60,
		Cohesion:     0.60,
		CurrentYear:  122,
		CurrentMonth: 1,
		Professions: []model.Profession{
			{ID: 1, Name: "Mechanical", ClassType: "COMMONER", Relations: map[string]float64{"Mechanical": 1.0, "Supply": 0.6, "Mines": 0.6}},
			{ID: 2, Name: "Supply", ClassType: "COMMONER", Relations: map[string]float64{"Supply": 1.0, "Mechanical": 0.6, "Mines": 0.6}},
			{ID: 3, Name: "Mines", ClassType: "COMMONER", Relations: map[string]float64{"Mines": 1.0, "Mechanical": 0.6, "Supply": 0.6}},
		},
		Cohorts: []model.PopulationCohort{
			{ID: 1, ProfessionID: 1, Name: "Radicals", Count: 400, Influence: 0.5,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.90, model.IdeologyLoyalty: 0.20}},
			{ID: 2, ProfessionID: 2, Name: "Moderates", Count: 400, Influence: 0.5,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.72, model.IdeologyLoyalty: 0.85}},
			{ID: 3, ProfessionID: 3, Name: "Quiet", Count: 400, Influence: 0.5,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.10, model.IdeologyLoyalty: 0.80}},
		},
		Factions: []model.Faction{
			{
				ID: 11, SiloID: 1, Name: "激进派",
				Signature: "founded:PLAYER:42|" + model.IdeologyProForeign + ":High",
				Tags:      []string{model.IdeologyProForeign + ":High"},
				TagStats:  map[string]int{},
				Memory:    []string{"founded:Y122M1"},
				FoundedYear: 122, FoundedMonth: 1,
				RepresentativeKind: string(ActorPlayer), RepresentativeID: 42,
			},
			{
				ID: 12, SiloID: 1, Name: "温和派",
				Signature: "founded:PLAYER:43|" + model.IdeologyProForeign + ":High",
				Tags:      []string{model.IdeologyProForeign + ":High"},
				TagStats:  map[string]int{},
				Memory:    []string{"founded:Y122M1"},
				FoundedYear: 122, FoundedMonth: 1,
				RepresentativeKind: string(ActorPlayer), RepresentativeID: 43,
			},
		},
	}
	for i := range silo.Cohorts {
		prof := getProfessionByID(silo, silo.Cohorts[i].ProfessionID)
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}
	silo.Cohorts[0].FactionID = uintPtr(11)
	silo.Cohorts[1].FactionID = uintPtr(12)
	return silo
}

func TestExplainCohortAffiliationDistances(t *testing.T) {
	silo := explainTestSilo()
	MaintainFactions(silo, nil)

	// Radicals 是激进派唯一成员 → 自身即质心 (距离 0)，次近为温和派。
	expl, err := ExplainCohortAffiliation(silo, 1)
	if err != nil {
		t.Fatalf("explain cohort 1: %v", err)
	}
	if expl.FactionID != 11 {
		t.Fatalf("expected event-established affiliation with faction 11, got %+v", expl)
	}
	if expl.OwnDistance != 0 {
		t.Fatalf("sole member cohort must sit on its own centroid, got %.3f", expl.OwnDistance)
	}
	if expl.NearestID != 12 || expl.NearestDist <= 0 {
		t.Fatalf("expected the other founded faction as nearest, got %+v", expl)
	}
	if expl.Margin <= 0 {
		t.Fatalf("margin must be positive when own faction is nearest, got %.3f", expl.Margin)
	}
	if !strings.Contains(expl.Reason, "事件驱动成员资格") || !strings.Contains(expl.Reason, "亲缘参考") {
		t.Fatalf("reason must state event-driven membership with affinity-only distances, got %q", expl.Reason)
	}

	// 无阵营 cohort 的解释。
	explQuiet, err := ExplainCohortAffiliation(silo, 3)
	if err != nil {
		t.Fatalf("explain cohort 3: %v", err)
	}
	if explQuiet.FactionID != 0 || !strings.Contains(explQuiet.Reason, "无阵营") {
		t.Fatalf("quiet cohort must be explained as unaffiliated, got %+v", explQuiet)
	}
}

// 零成员筹备期派系：无质心可算，解释必须明说而不是报错。
func TestExplainZeroMemberFaction(t *testing.T) {
	silo := explainTestSilo()
	silo.Factions = append(silo.Factions, model.Faction{
		ID: 14, SiloID: 1, Name: "空壳派系",
		Signature: "founded:PLAYER:44|" + model.IdeologyProForeign + ":High",
		Tags:      []string{model.IdeologyProForeign + ":High"},
		TagStats:  map[string]int{},
		Memory:    []string{"founded:Y122M1"},
		FoundedYear: 122, FoundedMonth: 1,
		RepresentativeKind: string(ActorPlayer), RepresentativeID: 44,
	})
	MaintainFactions(silo, nil)
	silo.Residents = append(silo.Residents, model.Resident{
		ID: 900, SiloID: 1, Name: "FounderProbe", Alive: true,
		ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(14),
		Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
	})

	expl, err := ExplainResidentAffiliation(silo, "FounderProbe")
	if err != nil {
		t.Fatalf("explain zero-member faction resident: %v", err)
	}
	if expl.FactionID != 14 || !strings.Contains(expl.Reason, "零成员筹备期") {
		t.Fatalf("zero-member faction must be explained as in grace period, got %+v", expl)
	}
}

func TestExplainResidentAffiliationReadOnly(t *testing.T) {
	silo := explainTestSilo()
	MaintainFactions(silo, nil)

	// 为 Radicals cohort 造一个具名居民 (模拟 Silo 18 具名个体的解释路径)。
	cohort := &silo.Cohorts[0]
	resident := model.Resident{
		ID:           uint(len(silo.Residents) + 1),
		SiloID:       silo.ID,
		Name:         "Probe",
		CohortID:     uintPtr(cohort.ID),
		ProfessionID: cohort.ProfessionID,
		Profession:   "Mechanical",
		FactionID:    cohort.FactionID,
		Ideologies: map[string]float64{
			model.IdeologyProForeign: 0.88,
			model.IdeologyLoyalty:    0.22,
		},
		Alive: true,
	}
	silo.Residents = append(silo.Residents, resident)

	before := *cohort // 深拷贝关键字段，验证只读
	expl, err := ExplainResidentAffiliation(silo, "Probe")
	if err != nil {
		t.Fatalf("explain resident: %v", err)
	}
	if expl.FactionID != 11 || expl.OwnDistance <= 0 {
		t.Fatalf("resident explanation must carry faction and distance, got %+v", expl)
	}
	if cohort.ID != before.ID || cohort.Count != before.Count ||
		cohort.Ideologies[model.IdeologyProForeign] != before.Ideologies[model.IdeologyProForeign] {
		t.Fatalf("explanation must be read-only; cohort mutated")
	}
	if _, err := ExplainResidentAffiliation(silo, "Nobody"); err == nil {
		t.Fatalf("unknown resident must return an error")
	}
}
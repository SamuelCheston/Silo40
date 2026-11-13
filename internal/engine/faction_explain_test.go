package engine

// P1-S4 可解释归属测试：解释接口只读、自洽，且能区分"距离更近"与"惯性保留"。

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// explainTestSilo 三个政治派系 (质心相距较远) + 一个无阵营 cohort。
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
	}
	for i := range silo.Cohorts {
		prof := getProfessionByID(silo, silo.Cohorts[i].ProfessionID)
		silo.Cohorts[i].Tags = buildCohortTags(&silo.Cohorts[i], prof, silo)
	}
	return silo
}

func TestExplainCohortAffiliationDistances(t *testing.T) {
	silo := explainTestSilo()
	RebuildImplicitFactions(silo)

	// Radicals 与 Moderates 各自成派 (距离 > 半径)，Quiet 无阵营。
	expl, err := ExplainCohortAffiliation(silo, 1)
	if err != nil {
		t.Fatalf("explain cohort 1: %v", err)
	}
	if expl.FactionID == 0 {
		t.Fatalf("expected political affiliation, got unaffiliated")
	}
	if expl.NearestID == 0 {
		t.Fatalf("expected a nearest other political faction")
	}
	if expl.OwnDistance >= expl.NearestDist {
		t.Fatalf("freshly clustered cohort must be closer to its own faction: own=%.3f nearest=%.3f",
			expl.OwnDistance, expl.NearestDist)
	}
	if expl.HeldByInertia {
		t.Fatalf("freshly clustered cohort must not be inertia-held")
	}
	if expl.Margin <= 0 {
		t.Fatalf("margin must be positive when own faction is nearest, got %.3f", expl.Margin)
	}
	if !strings.Contains(expl.Reason, "本派系更近") {
		t.Fatalf("reason must explain nearest-own case, got %q", expl.Reason)
	}

	// 无阵营 cohort 的解释。
	explQuiet, err := ExplainCohortAffiliation(silo, 3)
	if err != nil {
		t.Fatalf("explain cohort 3: %v", err)
	}
	if explQuiet.FactionID != 0 || !strings.Contains(explQuiet.Reason, "unaffiliated") {
		t.Fatalf("quiet cohort must be explained as unaffiliated, got %+v", explQuiet)
	}
}

func TestExplainResidentAffiliationReadOnly(t *testing.T) {
	silo := explainTestSilo()
	RebuildImplicitFactions(silo)

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
	if expl.FactionID == 0 || expl.OwnDistance <= 0 {
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

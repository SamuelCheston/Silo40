package engine

import (
	"testing"

	"silo40/internal/model"
)

// 威望全层派生 (24 篇 §2)：PLAYER / NPC / RESIDENT 经同一 ActorView 接口
// 读出 50×(A+I)；cohort 层借代表居民的 A/I 用同一公式派生。四层同构，
// 不存在独立存储的威望字段。
func TestDerivedPrestigeIsomorphicAcrossLayers(t *testing.T) {
	const ambition, influence = 0.7, 0.3
	want := DerivedPrestige(ambition, influence)
	if want != 50 {
		t.Fatalf("DerivedPrestige(0.7, 0.3) = %v, want 50", want)
	}

	cohortID := uint(3)
	silo := &model.Silo{
		Professions: []model.Profession{
			{ID: 1, Name: "Mayor", Ambition: ambition, Influence: influence},
		},
		Residents: []model.Resident{
			{ID: 2, Name: "Rep", Alive: true, CohortID: &cohortID, Ambition: ambition, Influence: influence},
		},
		Cohorts: []model.PopulationCohort{{ID: cohortID}},
	}
	agent := &model.Agent{ID: 4, Profession: "Mayor", Ambition: ambition, Influence: influence}

	views := []*ActorView{}
	player, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, player)
	npc, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[0]), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, npc)
	resident, err := CreateActorView(CreateActorRefForResident(&silo.Residents[0]), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, resident)

	for _, view := range views {
		if got := view.PoliticalPrestige(); got != want {
			t.Errorf("kind %v prestige = %v, want %v", view.Ref.Kind, got, want)
		}
	}
	if got := cohortPrestige(silo, &silo.Cohorts[0]); got != want {
		t.Errorf("cohort prestige = %v, want %v", got, want)
	}
}

// 旧 political_prestige mutation 折算为 influence (value/50)：delta 与 set
// 两种口径对三类个体一致，且受 clamp01 约束；ambition 不被 prestige
// mutation 触碰（开局定死，仅事件显式改）。
func TestPrestigeMutationConvertsToInfluence(t *testing.T) {
	e := NewGameEngine()
	cohortID := uint(3)
	silo := &model.Silo{
		Professions: []model.Profession{
			{ID: 1, Name: "Mayor", Ambition: 0.5, Influence: 0.5},
		},
		Residents: []model.Resident{
			{ID: 2, Alive: true, CohortID: &cohortID, Ambition: 0.5, Influence: 0.5},
		},
	}
	agent := &model.Agent{ID: 4, Profession: "Mayor", Ambition: 0.5, Influence: 0.5}

	views := []*ActorView{}
	for _, ref := range []ActorRef{
		CreateActorRefForAgent(agent, silo),
		CreateActorRefForProfession(&silo.Professions[0]),
		CreateActorRefForResident(&silo.Residents[0]),
	} {
		view, err := CreateActorView(ref, silo, agent)
		if err != nil {
			t.Fatal(err)
		}
		views = append(views, view)
	}

	for _, view := range views {
		e.applyActorMetricDelta(view, "political_prestige", 12.5)
		if got := view.Influence(); got != 0.75 {
			t.Errorf("kind %v delta +12.5 influence = %v, want 0.75", view.Ref.Kind, got)
		}
		if got := view.Ambition(); got != 0.5 {
			t.Errorf("kind %v ambition touched by prestige delta: %v", view.Ref.Kind, got)
		}
		if got := view.PoliticalPrestige(); got != 62.5 {
			t.Errorf("kind %v derived prestige = %v, want 62.5", view.Ref.Kind, got)
		}
		e.applyActorMetricSet(view, "political_prestige", 40)
		if got := view.Influence(); got != 0.8 {
			t.Errorf("kind %v set 40 influence = %v, want 0.8", view.Ref.Kind, got)
		}
		e.applyActorMetricDelta(view, "political_prestige", 100)
		if got := view.Influence(); got != 1 {
			t.Errorf("kind %v delta clamp influence = %v, want 1", view.Ref.Kind, got)
		}
		e.applyActorMetricSet(view, "political_prestige", -20)
		if got := view.Influence(); got != 0 {
			t.Errorf("kind %v set clamp influence = %v, want 0", view.Ref.Kind, got)
		}
	}
}
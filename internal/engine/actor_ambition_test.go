package engine

import (
	"math"
	"testing"
)

// 野心开局定死：NPC 部门与玩家特工的 Ambition/Influence 由初始化赋值，
// 月度人口/居民动力学不得漂移 Ambition (仅事件可改)。
func TestAmbitionInfluenceInitialValues(t *testing.T) {
	silo := CreateInitialSilo("Ambition Silo", 122, nil)
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		wantAmbition := clamp01(0.2 + 0.03*float64(prof.PowerLevel) + 0.025)
		if prof.Ambition != wantAmbition {
			t.Fatalf("%s ambition=%v want %v", prof.Name, prof.Ambition, wantAmbition)
		}
		wantInfluence := 0.6 * float64(prof.PowerLevel) / 10.0
		if math.Abs(prof.Influence-wantInfluence) > 1e-9 {
			t.Fatalf("%s influence=%v want %v", prof.Name, prof.Influence, wantInfluence)
		}
	}

	cases := []struct {
		profession      string
		ambition, influ float64
	}{
		{"Mayor", 0.8, 0.8},
		{"Judicial", 0.05, 0.05},
		{"IT", 0.05, 0.05},
		{"Medical", 0.05, 0.05},
		{"Mines", 0.05, 0.05},
		{"Supply", 0.1, 0.1},
	}
	for _, c := range cases {
		agent := CreateInitialAgent("Agent"+c.profession, c.profession, nil, silo)
		if agent.Ambition != c.ambition || agent.Influence != c.influ {
			t.Fatalf("%s agent ambition/influence=%v/%v want %v/%v",
				c.profession, agent.Ambition, agent.Influence, c.ambition, c.influ)
		}
	}

	charismatic := CreateInitialAgent("Charisma", "Mayor", []string{"charismatic"}, silo)
	if charismatic.Ambition != 1.0 || charismatic.Influence != 1.0 {
		t.Fatalf("charismatic Mayor should clamp to 1.0/1.0, got %v/%v",
			charismatic.Ambition, charismatic.Influence)
	}
	commoner := CreateInitialAgent("CharismaSupply", "Supply", []string{"charismatic"}, silo)
	if math.Abs(commoner.Ambition-0.3) > 1e-9 || math.Abs(commoner.Influence-0.3) > 1e-9 {
		t.Fatalf("charismatic Supply should be 0.3/0.3, got %v/%v",
			commoner.Ambition, commoner.Influence)
	}
}

func TestActorAmbitionInfluenceMutationAllKinds(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Mutation Silo", 122, nil)
	agent := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)
	dept := findDept(silo, "Mechanical")

	views := []*ActorView{}
	playerView, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, playerView)
	npcView, err := CreateActorView(CreateActorRefForProfession(dept), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, npcView)
	if len(silo.Residents) == 0 {
		t.Fatal("expected key residents after init")
	}
	resView, err := CreateActorView(CreateActorRefForResident(&silo.Residents[0]), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	views = append(views, resView)

	for _, view := range views {
		e.applyActorMetricDelta(view, "ambition", 0.1)
		e.applyActorMetricDelta(view, "influence", -0.05)
		e.applyActorMetricSet(view, "ambition", 1.5)
		if view.Ambition() != 1.0 {
			t.Fatalf("%s ambition set should clamp to 1.0, got %v", view.Ref.Kind, view.Ambition())
		}
		e.applyActorMetricSet(view, "influence", -0.2)
		if view.Influence() != 0.0 {
			t.Fatalf("%s influence set should clamp to 0, got %v", view.Ref.Kind, view.Influence())
		}
		e.applyActorMetricDelta(view, "ambition", -2.0)
		if view.Ambition() != 0.0 {
			t.Fatalf("%s ambition delta should clamp to 0, got %v", view.Ref.Kind, view.Ambition())
		}
	}

	if agent.Ambition != 0.0 || dept.Ambition != 0.0 || silo.Residents[0].Ambition != 0.0 {
		t.Fatal("ambition mutation must write through to the underlying entity")
	}
}

func TestAmbitionDoesNotDriftInSimulation(t *testing.T) {
	silo := CreateInitialSilo("Drift Silo", 122, nil)
	agent := CreateInitialAgent("MayorAgent", "Mayor", nil, silo)

	profAmbitions := map[string]float64{}
	for i := range silo.Professions {
		profAmbitions[silo.Professions[i].Name] = silo.Professions[i].Ambition
	}
	resAmbitions := map[uint]float64{}
	for i := range silo.Residents {
		resAmbitions[silo.Residents[i].ID] = silo.Residents[i].Ambition
	}
	agentAmbition := agent.Ambition

	for month := 0; month < 24; month++ {
		updatePopulationCohorts(silo, 1.0/12.0)
		updateKeyResidents(silo, 1.0/12.0)
	}

	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Ambition != profAmbitions[prof.Name] {
			t.Fatalf("%s ambition drifted: %v -> %v", prof.Name, profAmbitions[prof.Name], prof.Ambition)
		}
	}
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if before, ok := resAmbitions[res.ID]; ok && res.Ambition != before {
			t.Fatalf("resident %d ambition drifted: %v -> %v", res.ID, before, res.Ambition)
		}
	}
	if agent.Ambition != agentAmbition {
		t.Fatalf("agent ambition drifted: %v -> %v", agentAmbition, agent.Ambition)
	}
}
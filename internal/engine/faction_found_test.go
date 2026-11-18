package engine

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

func foundTestSilo() *model.Silo {
	return &model.Silo{
		ID:           1,
		Legitimacy:   0.60,
		Cohesion:     0.60,
		CurrentYear:  122,
		CurrentMonth: 5,
		Professions: []model.Profession{
			{
				ID:         1,
				Name:       "Mechanical",
				ClassType:  "COMMONER",
				Zone:       "Lower",
				PowerLevel: 4,
				Relations:  map[string]float64{"Mechanical": 1.0},
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			},
		},
	}
}

// 发起资格三闸 (威望/强度/AP) + 无政治派系；无阵营派系不算政治派系。
func TestFoundFactionEligibilityGates(t *testing.T) {
	silo := foundTestSilo()
	agent := &model.Agent{ID: 42, Profession: "Mechanical", Ambition: 0.6, Influence: 0.5, ActionPoints: 20}
	view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}
	if ok, reason := foundFactionEligible(view, silo); !ok {
		t.Fatalf("prestige 55 / intensity 0.4 / AP 20 must be eligible, got %q", reason)
	}

	agent.Ambition = 0.4 // prestige 45 < 50
	if ok, _ := foundFactionEligible(view, silo); ok {
		t.Fatal("prestige below threshold must be ineligible")
	}
	agent.Ambition = 0.6

	silo.Professions[0].Ideologies[model.IdeologyProForeign] = 0.6 // intensity 0.1 < 0.25
	if ok, _ := foundFactionEligible(view, silo); ok {
		t.Fatal("intensity below threshold must be ineligible")
	}
	silo.Professions[0].Ideologies[model.IdeologyProForeign] = 0.9

	agent.ActionPoints = 19
	if ok, _ := foundFactionEligible(view, silo); ok {
		t.Fatal("AP below cost must be ineligible")
	}
	agent.ActionPoints = 20

	// 已代表政治派系 → 不可再发起。
	silo.Factions = []model.Faction{{
		ID: 7, Signature: "founded:PLAYER:42", MemberCount: 0,
		RepresentativeKind: string(ActorPlayer), RepresentativeID: 42,
	}}
	if ok, _ := foundFactionEligible(view, silo); ok {
		t.Fatal("representative of a political faction must be ineligible")
	}
	// 无阵营派系不算政治派系 → 仍可发起。
	silo.Factions[0].Signature = unaffiliatedSignature
	if ok, reason := foundFactionEligible(view, silo); !ok {
		t.Fatalf("unaffiliated faction must not block founding, got %q", reason)
	}
}

// faction_found_actor：权威三闸 + AP 扣除原子完成；零成员派系合法，
// 代表/创始人/成立年月/立场/签名齐备；重复发起被拒。
func TestApplyFactionFoundActorCreatesZeroMemberFaction(t *testing.T) {
	silo := foundTestSilo()
	agent := &model.Agent{ID: 42, Name: "Test Agent", Profession: "Mechanical", Ambition: 0.6, Influence: 0.5, ActionPoints: 25}
	ref := CreateActorRefForAgent(agent, silo)
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_found_actor"})

	if len(silo.Factions) != 1 {
		t.Fatalf("expected 1 founded faction, got %d", len(silo.Factions))
	}
	f := silo.Factions[0]
	if f.MemberCount != 0 {
		t.Fatalf("founded faction must start with zero members, got %d", f.MemberCount)
	}
	if f.RepresentativeKind != string(ActorPlayer) || f.RepresentativeID != 42 ||
		f.FounderKind != string(ActorPlayer) || f.FounderID != 42 {
		t.Fatalf("representative/founder must be the founding player: %+v", f)
	}
	if f.FoundedYear != 122 || f.FoundedMonth != 5 {
		t.Fatalf("founding stamp = Y%dM%d, want Y122M5", f.FoundedYear, f.FoundedMonth)
	}
	if !strings.HasPrefix(f.Signature, "founded:PLAYER:42") || !strings.Contains(f.Signature, "pro_foreign:High") {
		t.Fatalf("signature vocabulary mismatch: %q", f.Signature)
	}
	if f.Name != "Test Agent的派系" || f.RepresentativeName != "Test Agent" {
		t.Fatalf("naming mismatch: %+v", f)
	}
	if len(f.Memory) == 0 || !strings.HasPrefix(f.Memory[0], "founded:") {
		t.Fatalf("founding memory missing: %+v", f.Memory)
	}
	if agent.ActionPoints != 5 {
		t.Fatalf("AP = %v, want 25-20=5", agent.ActionPoints)
	}

	view, err := CreateActorView(ref, silo, agent)
	if err != nil {
		t.Fatal(err)
	}
	if !view.IsRepresentative() || view.FactionID() == nil || *view.FactionID() != f.ID {
		t.Fatal("founder must resolve as representative of the new faction")
	}

	// 重复发起：已是政治派系代表 → 拒绝，AP 不再扣。
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_found_actor"})
	if len(silo.Factions) != 1 || agent.ActionPoints != 5 {
		t.Fatalf("repeat founding must be rejected: factions=%d AP=%v", len(silo.Factions), agent.ActionPoints)
	}
}

// NPC (部门) 发起与玩家同构：RepresentativeKind=NPC，视图解析一致。
func TestNPCFoundFactionIsomorphic(t *testing.T) {
	silo := foundTestSilo()
	silo.Professions[0].Ambition = 0.6
	silo.Professions[0].Influence = 0.5
	silo.Professions[0].ActionPoints = 30
	silo.Professions[0].Ideologies[model.IdeologyProForeign] = 0.1 // intensity 0.4

	ref := CreateActorRefForProfession(&silo.Professions[0])
	e := NewGameEngine()
	e.ApplyMutation(silo, nil, ref, MechanicMutation{Type: "faction_found_actor"})

	if len(silo.Factions) != 1 {
		t.Fatalf("expected 1 founded faction, got %d", len(silo.Factions))
	}
	f := silo.Factions[0]
	if f.RepresentativeKind != string(ActorNPC) || f.RepresentativeID != 1 ||
		f.FounderKind != string(ActorNPC) || f.FounderID != 1 {
		t.Fatalf("NPC founding fields mismatch: %+v", f)
	}
	if silo.Professions[0].ActionPoints != 10 {
		t.Fatalf("NPC AP = %v, want 30-20=10", silo.Professions[0].ActionPoints)
	}
	view, err := CreateActorView(ref, silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !view.IsRepresentative() || view.FactionID() == nil || *view.FactionID() != f.ID {
		t.Fatal("NPC founder must resolve as representative of the new faction")
	}
}

// C6 共存期：发起派系跨 RebuildImplicitFactions 保留，居民代表重新绑定，
// 聚类派系不得重复认领同一居民代表。
func TestFoundedFactionSurvivesRebuild(t *testing.T) {
	silo := foundTestSilo()
	silo.Cohorts = []model.PopulationCohort{
		{
			ID:           1,
			ProfessionID: 1,
			Name:         "Mechanical Bloc",
			Count:        400,
			Influence:    0.70,
			Ideologies:   map[string]float64{model.IdeologyProForeign: 0.90},
		},
	}
	prof := getProfessionByID(silo, 1)
	silo.Cohorts[0].Tags = buildCohortTags(&silo.Cohorts[0], prof, silo)
	silo.Residents = []model.Resident{
		{
			ID: 9, Alive: true, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical",
			Ambition: 0.7, Influence: 0.5, ActionPoints: 20,
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			Relations:  map[string]float64{"Mechanical": 1.0},
		},
	}

	resRef := CreateActorRefForResident(&silo.Residents[0])
	e := NewGameEngine()
	e.ApplyMutation(silo, nil, resRef, MechanicMutation{Type: "faction_found_actor"})
	if len(silo.Factions) != 1 {
		t.Fatalf("resident founding failed: factions=%d", len(silo.Factions))
	}
	foundedID := silo.Factions[0].ID

	silo.CurrentYear = 123
	silo.CurrentMonth = 9
	RebuildImplicitFactions(silo)

	var founded, political *model.Faction
	for i := range silo.Factions {
		f := &silo.Factions[i]
		switch {
		case strings.HasPrefix(f.Signature, foundedSignaturePrefix):
			founded = f
		case f.Signature != unaffiliatedSignature:
			political = f
		}
	}
	if founded == nil || founded.ID != foundedID {
		t.Fatal("founded faction must survive rebuild with its persistent ID")
	}
	if founded.FoundedYear != 122 || founded.FoundedMonth != 5 ||
		founded.RepresentativeKind != string(ActorResident) || founded.RepresentativeID != 9 {
		t.Fatalf("rebuild reset founding facts: %+v", founded)
	}
	if political == nil {
		t.Fatal("auto-clustering must still produce a political faction in C6 coexistence")
	}
	if political.RepresentativeKind == string(ActorResident) && political.RepresentativeID == 9 {
		t.Fatal("clustered faction must not double-claim the resident founder")
	}
	r := &silo.Residents[0]
	if !r.IsRepresentative || r.FactionID == nil || *r.FactionID != foundedID {
		t.Fatalf("resident founder must rebind to founded faction: %+v", r)
	}
}
package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// mergeTestSilo 两个居民领袖派系 (7/8)，领袖 mutual 个人关系 0.7，思潮同轴
// (pf0.9/loy0.2，Mechanical COMMONER) → 质心 dist=0。cohort 1 属 7、cohort 2
// 属 8，居民 9 是 8 的普通成员 (验证成员转移)。
func mergeTestSilo() *model.Silo {
	silo := joinTestSilo()
	ideo := map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}
	silo.Residents = []model.Resident{
		{ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical", ActionPoints: 20,
			IsRepresentative: true, FactionID: uintPtr(7), Ideologies: ideo,
			LeaderRelations: map[string]float64{"res:101": 0.7}},
		{ID: 101, Alive: true, ProfessionID: 1, Profession: "Mechanical", ActionPoints: 20,
			IsRepresentative: true, FactionID: uintPtr(8), Ideologies: ideo,
			LeaderRelations: map[string]float64{"res:100": 0.7}},
		{ID: 9, Alive: true, CohortID: uintPtr(2), ProfessionID: 1, Profession: "Mechanical",
			FactionID: uintPtr(8), Ideologies: ideo},
	}
	silo.Cohorts = []model.PopulationCohort{
		{ID: 1, ProfessionID: 1, Count: 300, FactionID: uintPtr(7), Ideologies: ideo},
		{ID: 2, ProfessionID: 1, Count: 200, FactionID: uintPtr(8), Ideologies: ideo},
	}
	silo.Factions = []model.Faction{
		{ID: 7, Name: "Bloc Seven", Signature: "founded:RESIDENT:100",
			RepresentativeKind: string(ActorResident), RepresentativeID: 100,
			Memory: []string{"founded:Y122M1"}},
		{ID: 8, Name: "Bloc Eight", Signature: "founded:RESIDENT:101",
			RepresentativeKind: string(ActorResident), RepresentativeID: 101,
			Memory: []string{"founded:Y122M2"}},
	}
	return silo
}

// 合并资格三闸 (关系/思潮/AP) + 领袖身份 + 自身/无阵营排除。
func TestMergeEligibilityThreeGates(t *testing.T) {
	// 基线全通过。
	silo := mergeTestSilo()
	ref := CreateActorRefForResident(&silo.Residents[0])
	view, _ := CreateActorView(ref, silo, nil)
	target := factionByID(silo, 8)
	if ok, reason := mergeFactionEligible(view, silo, target, nil); !ok {
		t.Fatalf("baseline merge must pass, got %q", reason)
	}
	// 关系闸：mutual 最小值 < 0.60 → 拒 (单向 0.7 不够，需双方达标)。
	silo = mergeTestSilo()
	silo.Residents[1].LeaderRelations["res:100"] = 0.5
	ref = CreateActorRefForResident(&silo.Residents[0])
	view, _ = CreateActorView(ref, silo, nil)
	if ok, _ := mergeFactionEligible(view, silo, factionByID(silo, 8), nil); ok {
		t.Fatal("mutual relation min 0.5 < 0.60 must be rejected")
	}
	// AP 闸：AP < 15 → 拒。
	silo = mergeTestSilo()
	silo.Residents[0].ActionPoints = 10
	ref = CreateActorRefForResident(&silo.Residents[0])
	view, _ = CreateActorView(ref, silo, nil)
	if ok, _ := mergeFactionEligible(view, silo, factionByID(silo, 8), nil); ok {
		t.Fatal("AP below MergeAPCost must be rejected")
	}
	// 思潮闸：目标质心漂移出亲缘半径 → 拒。
	silo = mergeTestSilo()
	silo.Cohorts[1].Ideologies = map[string]float64{model.IdeologyProForeign: 0.1, model.IdeologyLoyalty: 0.2}
	ref = CreateActorRefForResident(&silo.Residents[0])
	view, _ = CreateActorView(ref, silo, nil)
	if ok, _ := mergeFactionEligible(view, silo, factionByID(silo, 8), nil); ok {
		t.Fatal("ideology distance beyond affinity radius must be rejected")
	}
	// 领袖身份闸：普通成员 (居民 9，非领袖) 不能发起。
	silo = mergeTestSilo()
	ref = CreateActorRefForResident(&silo.Residents[2])
	view, _ = CreateActorView(ref, silo, nil)
	if ok, _ := mergeFactionEligible(view, silo, factionByID(silo, 8), nil); ok {
		t.Fatal("non-leader must not initiate merge")
	}
	// 自身合并 → 拒。
	silo = mergeTestSilo()
	ref = CreateActorRefForResident(&silo.Residents[0])
	view, _ = CreateActorView(ref, silo, nil)
	if ok, _ := mergeFactionEligible(view, silo, factionByID(silo, 7), nil); ok {
		t.Fatal("self merge must be rejected")
	}
}

// 成员转移 + 存续 ID 不变 + 被并方除名 + Memory 追加 + AP 扣除 + 领袖降级。
func TestMergeMemberTransferAndIDStable(t *testing.T) {
	silo := mergeTestSilo()
	ref := CreateActorRefForResident(&silo.Residents[0])
	e := NewGameEngine()
	e.ApplyMutation(silo, nil, ref, MechanicMutation{Type: "faction_merge_actor", IntValue: 8})

	if factionByID(silo, 7) == nil {
		t.Fatal("surviving faction 7 must keep its ID")
	}
	if factionByID(silo, 8) != nil {
		t.Fatal("absorbed faction 8 must be removed")
	}
	if silo.Cohorts[1].FactionID == nil || *silo.Cohorts[1].FactionID != 7 {
		t.Fatalf("absorbed cohort must transfer to survivor: %+v", silo.Cohorts[1].FactionID)
	}
	if silo.Residents[1].FactionID == nil || *silo.Residents[1].FactionID != 7 {
		t.Fatalf("absorbed leader must become member of survivor: %+v", silo.Residents[1].FactionID)
	}
	if silo.Residents[1].IsRepresentative {
		t.Fatal("absorbed leader must lose representative flag")
	}
	if silo.Residents[2].FactionID == nil || *silo.Residents[2].FactionID != 7 {
		t.Fatalf("absorbed ordinary member must transfer: %+v", silo.Residents[2].FactionID)
	}
	survivor := factionByID(silo, 7)
	if survivor.MemberCount != 500 {
		t.Fatalf("survivor MemberCount = %d, want 300+200=500", survivor.MemberCount)
	}
	found := false
	for _, m := range survivor.Memory {
		if m == "merged:8:Y122M5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("survivor Memory must append merge record, got %v", survivor.Memory)
	}
	if silo.Residents[0].ActionPoints != 5 {
		t.Fatalf("AP = %v, want 20-15=5", silo.Residents[0].ActionPoints)
	}
}

// 玩家领袖合并：关系读对方居民子矩阵 agent:<id> 键；玩家直选不掷骰。
func TestMergePlayerLeaderRelationAndNoRNG(t *testing.T) {
	silo := joinTestSilo()
	ideo := map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}
	silo.Residents = []model.Resident{
		{ID: 101, Alive: true, ProfessionID: 1, Profession: "Mechanical",
			IsRepresentative: true, FactionID: uintPtr(8), Ideologies: ideo,
			LeaderRelations: map[string]float64{"agent:42": 0.7}},
	}
	silo.Cohorts = []model.PopulationCohort{
		{ID: 2, ProfessionID: 1, Count: 200, FactionID: uintPtr(8), Ideologies: ideo},
	}
	silo.Factions = []model.Faction{
		{ID: 7, Name: "Player Bloc", Signature: "founded:PLAYER:42",
			RepresentativeKind: string(ActorPlayer), RepresentativeID: 42},
		{ID: 8, Name: "Resident Bloc", Signature: "founded:RESIDENT:101",
			RepresentativeKind: string(ActorResident), RepresentativeID: 101},
	}
	agent := &model.Agent{ID: 42, Profession: "Mechanical", ActionPoints: 20}
	ref := CreateActorRefForAgent(agent, silo)

	const seed = 555444333
	SetRNG(rand.New(rand.NewSource(seed)))
	control := rng.Float64()
	SetRNG(rand.New(rand.NewSource(seed)))
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_merge_actor", IntValue: 8})
	if after := rng.Float64(); after != control {
		t.Fatalf("player direct merge must not consume RNG: after=%v control=%v", after, control)
	}
	if factionByID(silo, 8) != nil || factionByID(silo, 7) == nil {
		t.Fatal("player-led merge must absorb 8 into 7")
	}
	if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != 7 {
		t.Fatalf("absorbed cohort must transfer to player faction: %+v", silo.Cohorts[0].FactionID)
	}
	if agent.ActionPoints != 5 {
		t.Fatalf("AP = %v, want 20-15=5", agent.ActionPoints)
	}
	// 关系不足 (对方居民子矩阵 agent 键 < 0.60) → 拒，玩家亦不豁免。
	silo2 := mergeTestSilo()
	silo2.Factions[0].RepresentativeKind = string(ActorPlayer)
	silo2.Factions[0].RepresentativeID = 42
	silo2.Factions[0].Signature = "founded:PLAYER:42"
	silo2.Residents[1].LeaderRelations = map[string]float64{"agent:42": 0.4}
	agent2 := &model.Agent{ID: 42, Profession: "Mechanical", ActionPoints: 20}
	ref2 := CreateActorRefForAgent(agent2, silo2)
	view2, _ := CreateActorView(ref2, silo2, agent2)
	if ok, _ := mergeFactionEligible(view2, silo2, factionByID(silo2, 8), agent2); ok {
		t.Fatal("player merge must still respect leader relation gate")
	}
}

// 零成员派系可发起/被合并 (亲缘回退领袖特征)。
func TestMergeZeroMemberFaction(t *testing.T) {
	silo := mergeTestSilo()
	// 清空两派系成员 cohort → 零成员，亲缘回退领袖特征 (同轴 → dist 0)。
	silo.Cohorts = nil
	silo.Residents[2].FactionID = uintPtr(8)
	ref := CreateActorRefForResident(&silo.Residents[0])
	view, _ := CreateActorView(ref, silo, nil)
	if ok, reason := mergeFactionEligible(view, silo, factionByID(silo, 8), nil); !ok {
		t.Fatalf("zero-member factions must be mergeable, got %q", reason)
	}
	e := NewGameEngine()
	e.ApplyMutation(silo, nil, ref, MechanicMutation{Type: "faction_merge_actor", IntValue: 8})
	if factionByID(silo, 8) != nil || factionByID(silo, 7) == nil {
		t.Fatal("zero-member merge must absorb 8 into 7")
	}
	if silo.Residents[1].FactionID == nil || *silo.Residents[1].FactionID != 7 {
		t.Fatal("absorbed zero-member leader must transfer to survivor")
	}
}
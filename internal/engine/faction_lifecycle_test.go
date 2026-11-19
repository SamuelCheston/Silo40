package engine

import (
	"fmt"
	"testing"

	"silo40/internal/model"
)

// C7 生命周期审计：筹备期解散、领袖死亡接任 (加权评分/平局取小 ID/
// 无候选解散)、零发起则零派系。

func lifecycleSilo() *model.Silo {
	return &model.Silo{
		ID:           1,
		Legitimacy:   0.60,
		Cohesion:     0.60,
		CurrentYear:  122,
		CurrentMonth: 1,
		Professions: []model.Profession{
			{
				ID: 1, Name: "Mechanical", ClassType: "COMMONER", Zone: "Lower",
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
				Relations:  map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID: 1, ProfessionID: 1, Name: "Mechanical Bloc", Count: 150, Influence: 0.5,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			},
		},
	}
}

func lifecycleFaction(id uint, repKind string, repID uint) model.Faction {
	return model.Faction{
		ID: id, SiloID: 1, Name: "筹备中的派系",
		Signature: fmt.Sprintf("%s%s:%d|%s:High", foundedSignaturePrefix, repKind, repID, model.IdeologyProForeign),
		Tags:      []string{model.IdeologyProForeign + ":High"},
		TagStats:  map[string]int{},
		Memory:    []string{"founded:Y122M1"},
		FoundedYear: 122, FoundedMonth: 1,
		RepresentativeKind: repKind, RepresentativeID: repID,
	}
}

// 筹备期 18 个月：未凑够 MinFactionSize 则解散，成员回无阵营桶；
// 届满前一个月仍存续。
func TestGraceDissolution(t *testing.T) {
	silo := lifecycleSilo()
	f := lifecycleFaction(7, string(ActorResident), 1)
	silo.Factions = []model.Faction{f}
	silo.Cohorts[0].FactionID = uintPtr(7)
	silo.Residents = []model.Resident{
		{ID: 1, Alive: true, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.7, Influence: 0.5},
	}

	// 17 个月：尚未届满。
	silo.CurrentYear = 123
	silo.CurrentMonth = 6
	MaintainFactions(silo, nil)
	if factionByID(silo, 7) == nil {
		t.Fatal("faction must survive until the grace deadline")
	}

	// 18 个月：150 人 < 200 → 解散。
	silo.CurrentMonth = 7
	var logs []string
	MaintainFactions(silo, func(msg string) { logs = append(logs, msg) })
	if factionByID(silo, 7) != nil {
		t.Fatal("under-sized faction must dissolve at the grace deadline")
	}
	if len(logs) == 0 {
		t.Fatal("dissolution must be logged")
	}
	unaff := silo.Factions[0]
	if unaff.Signature != unaffiliatedSignature {
		t.Fatalf("only the unaffiliated bucket must remain, got %+v", silo.Factions)
	}
	if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != unaff.ID {
		t.Fatalf("dissolved members must return to the unaffiliated bucket: %v", silo.Cohorts[0].FactionID)
	}
	for _, r := range silo.Residents {
		if r.FactionID != nil && *r.FactionID == 7 {
			t.Fatalf("resident still bound to dissolved faction: %+v", r)
		}
		if r.IsRepresentative {
			t.Fatalf("dissolved faction's representative flag must be cleared: %+v", r)
		}
	}
}

// 领袖死亡：派系内存活居民按 0.5×(prestige/100)+0.5×ambition 接任。
func TestLeaderDeathSuccessionByWeightedScore(t *testing.T) {
	silo := lifecycleSilo()
	silo.Cohorts[0].Count = 400 // 越过筹备期人数门槛
	f := lifecycleFaction(7, string(ActorResident), 1)
	silo.Factions = []model.Faction{f}
	silo.Cohorts[0].FactionID = uintPtr(7)
	silo.Residents = []model.Resident{
		{ID: 1, Alive: false, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.9, Influence: 0.9},
		{ID: 2, Alive: true, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.9, Influence: 0.5}, // score 0.80
		{ID: 3, Alive: true, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.5, Influence: 0.9}, // score 0.60
	}

	MaintainFactions(silo, nil)

	got := factionByID(silo, 7)
	if got == nil {
		t.Fatal("faction with living members must survive leader death")
	}
	if got.RepresentativeKind != string(ActorResident) || got.RepresentativeID != 2 {
		t.Fatalf("highest weighted score must succeed, got %+v", got)
	}
	if !silo.Residents[1].IsRepresentative || silo.Residents[2].IsRepresentative {
		t.Fatalf("representative flags must follow the succession: %+v", silo.Residents)
	}
}

// 平局取 ID 小者：接任确定性审计。
func TestSuccessionTieBreaksToSmallerID(t *testing.T) {
	silo := lifecycleSilo()
	silo.Cohorts[0].Count = 400
	f := lifecycleFaction(7, string(ActorResident), 1)
	silo.Factions = []model.Faction{f}
	silo.Cohorts[0].FactionID = uintPtr(7)
	silo.Residents = []model.Resident{
		{ID: 1, Alive: false, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7)},
		{ID: 5, Alive: true, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.6, Influence: 0.6},
		{ID: 4, Alive: true, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7), Ambition: 0.6, Influence: 0.6},
	}

	MaintainFactions(silo, nil)

	got := factionByID(silo, 7)
	if got == nil || got.RepresentativeID != 4 {
		t.Fatalf("tie must break to the smaller resident ID, got %+v", got)
	}
}

// 领袖死亡且无存活成员 → 派系解散。
func TestLeaderDeathWithoutSuccessorDissolves(t *testing.T) {
	silo := lifecycleSilo()
	silo.Cohorts[0].Count = 400
	f := lifecycleFaction(7, string(ActorResident), 1)
	silo.Factions = []model.Faction{f}
	silo.Cohorts[0].FactionID = uintPtr(7)
	silo.Residents = []model.Resident{
		{ID: 1, Alive: false, ProfessionID: 1, Profession: "Mechanical", FactionID: uintPtr(7)},
	}

	var logs []string
	MaintainFactions(silo, func(msg string) { logs = append(logs, msg) })

	if factionByID(silo, 7) != nil {
		t.Fatal("faction without a successor must dissolve")
	}
	if len(logs) == 0 {
		t.Fatal("succession failure dissolution must be logged")
	}
	if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID == 7 {
		t.Fatal("members must return to the unaffiliated bucket")
	}
}

// PLAYER/NPC 领袖不死亡：零成员派系在筹备期内恒有主、存续。
func TestPlayerAndNPCLeadersNeverDie(t *testing.T) {
	for _, kind := range []string{string(ActorPlayer), string(ActorNPC)} {
		silo := lifecycleSilo()
		silo.Factions = []model.Faction{lifecycleFaction(7, kind, 42)}
		MaintainFactions(silo, nil)
		f := factionByID(silo, 7)
		if f == nil || f.RepresentativeKind != kind || f.RepresentativeID != 42 {
			t.Fatalf("%s leader must persist in a zero-member faction: %+v", kind, f)
		}
	}
}

// 零发起审计：连续 24 个月维护，始终只有无阵营桶，ID 与归属稳定。
func TestZeroFoundingAuditStableAcrossMonths(t *testing.T) {
	silo := lifecycleSilo()
	silo.Cohorts[0].Count = 400

	MaintainFactions(silo, nil)
	if len(silo.Factions) != 1 || silo.Factions[0].Signature != unaffiliatedSignature {
		t.Fatalf("zero founding must yield only the unaffiliated bucket, got %+v", silo.Factions)
	}
	bucketID := silo.Factions[0].ID

	for month := 0; month < 24; month++ {
		silo.CurrentMonth++
		if silo.CurrentMonth > 12 {
			silo.CurrentMonth = 1
			silo.CurrentYear++
		}
		MaintainFactions(silo, nil)
		if len(silo.Factions) != 1 || silo.Factions[0].ID != bucketID {
			t.Fatalf("month %d: factions drifted without any founding: %+v", month, silo.Factions)
		}
		if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != bucketID {
			t.Fatalf("month %d: cohort membership drifted: %v", month, silo.Cohorts[0].FactionID)
		}
	}
}
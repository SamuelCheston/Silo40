package engine

import (
	"encoding/json"
	"testing"

	"silo40/internal/model"
)

// ADR-0018：玩家任派系代表与居民同构——IsRepresentative / FactionID /
// actorFaction / 派系 mutation 全部经 RepresentativeKind=PLAYER 引用解析，
// 零成员派系同样合法。
func TestPlayerRepresentativeResolution(t *testing.T) {
	silo := &model.Silo{
		ID: 1,
		Factions: []model.Faction{
			{
				ID:                 7,
				Name:               "玩家的小圈子",
				Signature:          "player:founded",
				MemberCount:        0,
				RepresentativeKind: string(ActorPlayer),
				RepresentativeID:   42,
				FounderKind:        string(ActorPlayer),
				FounderID:          42,
				FoundedYear:        122,
				FoundedMonth:       3,
			},
		},
	}
	agent := &model.Agent{ID: 42, Profession: "Mechanical"}
	view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}

	if !view.IsRepresentative() {
		t.Fatal("player representing a zero-member faction must be a representative")
	}
	fid := view.FactionID()
	if fid == nil || *fid != 7 {
		t.Fatalf("player FactionID = %v, want 7", fid)
	}
	if f := actorFaction(silo, view); f == nil || f.ID != 7 {
		t.Fatalf("actorFaction did not resolve player's faction: %v", f)
	}

	e := NewGameEngine()
	e.applyActorFactionPublic(silo, view, true)
	e.applyActorFactionMetricDelta(silo, view, "prestige", 15)
	if !silo.Factions[0].IsPublic || silo.Factions[0].Prestige != 15 {
		t.Fatalf("faction mutations did not land on player's faction: %+v", silo.Factions[0])
	}

	// 非代表玩家：无派系身份。
	other := &model.Agent{ID: 43, Profession: "Supply"}
	otherView, err := CreateActorView(CreateActorRefForAgent(other, silo), silo, other)
	if err != nil {
		t.Fatal(err)
	}
	if otherView.IsRepresentative() || otherView.FactionID() != nil {
		t.Fatal("non-representative player must have no faction identity")
	}

	// 居民代表路径不受影响。
	silo.Residents = []model.Resident{{ID: 9, Alive: true, IsRepresentative: true, FactionID: uintPtr(7)}}
	resView, err := CreateActorView(CreateActorRefForResident(&silo.Residents[0]), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resView.IsRepresentative() || resView.FactionID() == nil || *resView.FactionID() != 7 {
		t.Fatal("resident representative resolution regressed")
	}
}

// 代表/创始人 Actor 引用的序列化快照契约：PLAYER 与 RESIDENT 两种
// Kind 经 JSON 往返逐字段保真 (存档持有行快照，前端 models.ts 同构)。
func TestFactionActorFieldsJSONRoundTrip(t *testing.T) {
	for _, kind := range []string{string(ActorPlayer), string(ActorResident)} {
		in := model.Faction{
			ID:                 3,
			SiloID:             1,
			Name:               "_roundtrip",
			RepresentativeKind: kind,
			RepresentativeID:   11,
			RepresentativeName: "代表",
			FounderKind:        kind,
			FounderID:          12,
			FoundedYear:        123,
			FoundedMonth:       7,
		}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		var out model.Faction
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		if out.RepresentativeKind != kind || out.RepresentativeID != 11 ||
			out.FounderKind != kind || out.FounderID != 12 ||
			out.FoundedYear != 123 || out.FoundedMonth != 7 {
			t.Fatalf("kind %s roundtrip lost actor fields: %+v", kind, out)
		}
	}
}

// 成立年月与创始人随亲缘 ID 继承：重建不重置 founding 事实；
// 新派系打上当期年月戳。
func TestFactionFounderInheritedAcrossRebuilds(t *testing.T) {
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
				Relations: map[string]float64{"Mechanical": 1.0},
			},
		},
		Cohorts: []model.PopulationCohort{
			{
				ID:              1,
				ProfessionID:    1,
				Name:            "Mechanical Bloc",
				Count:           400,
				IdeologyProfile: []string{model.IdeologyProForeign + ":High"},
				Influence:       0.70,
				Ideologies:      map[string]float64{model.IdeologyProForeign: 0.90},
			},
		},
	}
	prof := getProfessionByID(silo, 1)
	silo.Cohorts[0].Tags = buildCohortTags(&silo.Cohorts[0], prof, silo)

	RebuildImplicitFactions(silo)
	var political *model.Faction
	for i := range silo.Factions {
		if silo.Factions[i].Signature != unaffiliatedSignature {
			political = &silo.Factions[i]
		}
	}
	if political == nil {
		t.Fatal("expected a political faction")
	}
	if political.FoundedYear != 122 || political.FoundedMonth != 5 {
		t.Fatalf("new faction founding stamp = Y%dM%d, want Y122M5", political.FoundedYear, political.FoundedMonth)
	}

	political.FounderKind = string(ActorPlayer)
	political.FounderID = 42
	political.FoundedYear = 121
	political.FoundedMonth = 2

	silo.CurrentYear = 123
	silo.CurrentMonth = 9
	RebuildImplicitFactions(silo)

	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == unaffiliatedSignature {
			continue
		}
		if f.FounderKind != string(ActorPlayer) || f.FounderID != 42 || f.FoundedYear != 121 || f.FoundedMonth != 2 {
			t.Fatalf("lineage rebuild reset founding facts: %+v", f)
		}
	}
}
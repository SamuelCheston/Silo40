package service

import (
	"testing"

	"silo40/internal/model"
)

// 多堡存档合并的 ID 重映射对 Actor 化派系引用的口径：RESIDENT 引用
// 随居民 ID 平移，PLAYER 引用指向特工 (跨堡唯一)，绝不平移。
func TestFactionActorIDOffsetOnlyResidents(t *testing.T) {
	cohortID := uint(5)
	in := []model.Faction{
		{
			ID:                 1,
			RepresentativeKind: "RESIDENT",
			RepresentativeID:   10,
			RepresentativeCohortID: &cohortID,
			FounderKind:        "RESIDENT",
			FounderID:          11,
		},
		{
			ID:                 2,
			RepresentativeKind: "PLAYER",
			RepresentativeID:   42,
			FounderKind:        "PLAYER",
			FounderID:          42,
		},
	}

	out := offsetFactions(in, 99, 1000)
	if out[0].RepresentativeID != 1010 || out[0].FounderID != 1011 {
		t.Fatalf("resident refs not offset: %+v", out[0])
	}
	if out[0].RepresentativeCohortID == nil || *out[0].RepresentativeCohortID != 1005 {
		t.Fatalf("cohort ref not offset: %+v", out[0])
	}
	if out[1].RepresentativeID != 42 || out[1].FounderID != 42 {
		t.Fatalf("player refs must not be offset: %+v", out[1])
	}

	silo := &model.Silo{Factions: out}
	unoffsetSiloSubIDs(silo, 1000)
	if silo.Factions[0].RepresentativeID != 10 || silo.Factions[0].FounderID != 11 {
		t.Fatalf("resident refs not unoffset: %+v", silo.Factions[0])
	}
	if silo.Factions[1].RepresentativeID != 42 || silo.Factions[1].FounderID != 42 {
		t.Fatalf("player refs must not be unoffset: %+v", silo.Factions[1])
	}
}
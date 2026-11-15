package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// seedForStation 找一个派生到指定号数的种子 (确定性查表，测试内自包含)。
func seedForStation(t *testing.T, station int) int64 {
	t.Helper()
	for seed := int64(1); seed < 100000; seed++ {
		if assignedStationNumber(seed) == station {
			return seed
		}
	}
	t.Fatalf("no seed found for station %d", station)
	return 0
}

func archetypeRequest(seed int64, archetype string, traitIDs ...string) model.CreateGameRequest {
	return model.CreateGameRequest{
		SiloName:   "ArchetypeStation",
		AgentName:  "Player",
		Profession: "Judicial",
		Seed:       seed,
		Archetype:  archetype,
		TraitIds:   traitIDs,
	}
}

// TestCreateGameDerivesArchetypeFromStationNumber locks 19 §3.3：缺省按号数
// 确定性查表派生原型并写入存档，档间开局旋钮即时生效；同种子派生稳定。
func TestCreateGameDerivesArchetypeFromStationNumber(t *testing.T) {
	cases := []struct {
		station int
	}{
		{2},  // 磐石
		{9},  // 裂隙
		{17}, // 荒年
	}
	for _, tc := range cases {
		seed := seedForStation(t, tc.station)
		want := engine.ArchetypeForStation(tc.station)
		svc := newGatedService(t, filepath.Join(t.TempDir(), "archetype-derive.db"))
		state, err := svc.CreateGame(archetypeRequest(seed, ""))
		if err != nil {
			t.Fatalf("create game (station %d): %v", tc.station, err)
		}
		if state == nil {
			t.Fatalf("create game returned nil state")
		}
		if svc.silo.Archetype != want {
			t.Fatalf("station %d (seed %d) derived archetype %q, want %q", tc.station, seed, svc.silo.Archetype, want)
		}
		cfg, ok := engine.ArchetypeByID(want)
		if !ok {
			t.Fatalf("derived archetype %q is not registered", want)
		}
		base := engine.DefaultAuthorityConfig()
		if wantW := base.NewGameW + cfg.W0Delta; svc.silo.Authority.W != wantW {
			t.Fatalf("station %d archetype %s W = %v, want %v", tc.station, want, svc.silo.Authority.W, wantW)
		}
		// 号数派生不消耗游戏内 RNG：同种子重建逐位一致。
		svc2 := newGatedService(t, filepath.Join(t.TempDir(), "archetype-derive-2.db"))
		if _, err := svc2.CreateGame(archetypeRequest(seed, "")); err != nil {
			t.Fatalf("recreate game (station %d): %v", tc.station, err)
		}
		if svc2.silo.Archetype != svc.silo.Archetype || svc2.silo.Authority.W != svc.silo.Authority.W {
			t.Fatalf("same seed must derive identical archetype state")
		}
	}
}

// TestCreateGameArchetypeOverride locks 覆写入口 (测试与未来玩家选档同一口
// 径) 与非法档位的显式失败。
func TestCreateGameArchetypeOverride(t *testing.T) {
	// 找一个派生结果不是荒年的种子，才能区分覆写与派生。
	seed := int64(0)
	for candidate := int64(1); candidate < 100000; candidate++ {
		if engine.ArchetypeForStation(assignedStationNumber(candidate)) != engine.ArchetypeFamine {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatalf("no non-famine deriving seed found")
	}

	svc := newGatedService(t, filepath.Join(t.TempDir(), "archetype-override.db"))
	if _, err := svc.CreateGame(archetypeRequest(seed, engine.ArchetypeFamine)); err != nil {
		t.Fatalf("create game with override: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeFamine {
		t.Fatalf("override archetype = %q, want %q", svc.silo.Archetype, engine.ArchetypeFamine)
	}

	svc2 := newGatedService(t, filepath.Join(t.TempDir(), "archetype-override-bad.db"))
	if _, err := svc2.CreateGame(archetypeRequest(seed, "not_a_tier")); err == nil || !strings.Contains(err.Error(), "unknown archetype") {
		t.Fatalf("unknown archetype must fail loudly, got %v", err)
	}
}

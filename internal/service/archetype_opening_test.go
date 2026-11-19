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

// runPlayableArchetypeMatrix 逐档跑一条可玩线 (19 §6.4：可玩性保证矩阵升级
// 为逐原型——每个原型都要跑通现有各条主线的可玩测试)。三档全部经覆写字段
// 钉档 (§3.3 与选档同一入口)；号数派生档由
// TestCreateGameDerivesArchetypeFromStationNumber 单独锁定。
func runPlayableArchetypeMatrix(t *testing.T, run func(*testing.T, string)) {
	t.Helper()
	for _, archetype := range []string{engine.ArchetypeBedrock, engine.ArchetypeRift, engine.ArchetypeFamine} {
		t.Run(archetype, func(t *testing.T) {
			run(t, archetype)
		})
	}
}

// threeSiloArchetypeRequest 在三堡建档请求上叠加原型覆写 (钉档)。
func threeSiloArchetypeRequest(seed int64, archetype string) model.CreateGameRequest {
	req := threeSiloRequest(seed)
	req.Archetype = archetype
	return req
}

// TestFamineOpeningAndFirstCrisisPerceptible locks 19 §3.2/§8：荒年在首月
// 资源面即可与非荒年区分，并在首个真实行动窗口后迎来第一波资源危机。
func TestFamineOpeningAndFirstCrisisPerceptible(t *testing.T) {
	type observation struct {
		svc             *GameService
		initial         map[string]float64
		firstMonth      map[string]float64
		openingTime     int
		firstCrisis     string
		firstCrisisTime int
	}
	run := func(t *testing.T, archetype string) observation {
		t.Helper()
		svc := newGatedService(t, filepath.Join(t.TempDir(), "famine-perception.db"))
		req := threeSiloArchetypeRequest(2401, archetype)
		req.Profession = "Supply"
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create %s game: %v", archetype, err)
		}
		obs := observation{
			svc: svc, initial: map[string]float64{}, firstMonth: map[string]float64{},
			openingTime: gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth),
		}
		for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
			obs.initial[resourceType] = resourceAmount(svc.silo, resourceType)
		}
		for month := 1; month <= 72; month++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("advance %s opening at month %d: %v", archetype, month, err)
			}
			if month == 1 {
				for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
					obs.firstMonth[resourceType] = resourceAmount(svc.silo, resourceType)
				}
			}
			for _, key := range []string{"crisis:food_poisoning", "crisis:water_leak"} {
				state := svc.contentStates[svc.silo.ID][key]
				if state.Triggered {
					obs.firstCrisis = key
					obs.firstCrisisTime = state.LastTriggeredTimestamp
					return obs
				}
			}
			if svc.gameOver() {
				t.Fatalf("%s opening ended before its first resource crisis: %+v", archetype, svc.silo.VictoryStatus)
			}
		}
		t.Fatalf("%s opening produced no resource crisis within 72 months", archetype)
		return obs
	}

	control := run(t, engine.ArchetypeBedrock)
	famine := run(t, engine.ArchetypeFamine)
	cfg, _ := engine.ArchetypeByID(engine.ArchetypeFamine)
	for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
		if got, want := famine.initial[resourceType], control.initial[resourceType]*cfg.ResourceFactor; got != want {
			t.Fatalf("famine opening %s = %v, want %v", resourceType, got, want)
		}
		if famine.firstMonth[resourceType] >= control.firstMonth[resourceType] {
			t.Fatalf("famine %s was not lower after the first real month: famine=%v control=%v",
				resourceType, famine.firstMonth[resourceType], control.firstMonth[resourceType])
		}
	}
	if famine.firstCrisis == "" || control.firstCrisis == "" {
		t.Fatalf("both openings must reach a real resource crisis: control=%+v famine=%+v", control, famine)
	}
	if famine.firstCrisisTime > control.firstCrisisTime {
		t.Fatalf("famine first crisis must not land later than the control opening: famine %s at %d, control %s at %d",
			famine.firstCrisis, famine.firstCrisisTime, control.firstCrisis, control.firstCrisisTime)
	}
	if famine.firstCrisisTime <= famine.openingTime {
		t.Fatalf("famine first crisis must follow the opening month: %s at %d, opening %d",
			famine.firstCrisis, famine.firstCrisisTime, famine.openingTime)
	}
	t.Logf("famine opening perceptible: resources ×%.2f, first crisis %s at +%d months (control %s at +%d months)",
		cfg.ResourceFactor, famine.firstCrisis, (famine.firstCrisisTime-famine.openingTime)/daysPerMonth,
		control.firstCrisis, (control.firstCrisisTime-control.openingTime)/daysPerMonth)
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

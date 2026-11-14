package service

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"silo40/internal/model"
)

// newCatalogService 在标准双堡服务上加载遗物清单（生产路径等价）。
// 不走该 helper 的旧机制测试保持清单关闭，开局 RNG 行为逐位不变。
func newCatalogService(t *testing.T, dbPath string) *GameService {
	t.Helper()
	svc := newTwoSiloService(t, dbPath)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.loadRelicCatalog(dirs); err != nil {
		t.Fatalf("load relic catalog: %v", err)
	}
	return svc
}

// relicSignature 可比快照：按实例化顺序取内容字段（不含自增主键与时间戳）。
func relicSignature(relics []model.Relic) []string {
	out := make([]string, len(relics))
	for i, r := range relics {
		out[i] = fmt.Sprintf("%s|%s|%s|%d", r.Name, r.LocationTag, r.SourceDept, r.DiscoveryYear)
	}
	return out
}

func TestCreateGameSeedsRelicsDeterministically(t *testing.T) {
	svc1 := newCatalogService(t, filepath.Join(t.TempDir(), "relic-d1.db"))
	if _, err := svc1.CreateGame(twoSiloRequest(42)); err != nil {
		t.Fatalf("create game 1: %v", err)
	}
	svc2 := newCatalogService(t, filepath.Join(t.TempDir(), "relic-d2.db"))
	if _, err := svc2.CreateGame(twoSiloRequest(42)); err != nil {
		t.Fatalf("create game 2: %v", err)
	}

	r1, r2 := svc1.silo.Relics, svc2.silo.Relics
	if len(r1) == 0 {
		t.Fatal("expected seeded relics in player silo")
	}
	if len(r1) < svc1.relicCatalog.Spawn.Min || len(r1) > svc1.relicCatalog.Spawn.Max {
		t.Fatalf("relic count %d outside spawn budget %+v", len(r1), svc1.relicCatalog.Spawn)
	}
	if !reflect.DeepEqual(relicSignature(r1), relicSignature(r2)) {
		t.Fatalf("same seed produced different relics:\n%v\nvs\n%v", relicSignature(r1), relicSignature(r2))
	}
	for _, relic := range r1 {
		if _, ok := svc1.relicCatalog.Locations[relic.LocationTag]; !ok {
			t.Fatalf("relic %q placed at unknown location %q", relic.Name, relic.LocationTag)
		}
	}
	// 双堡局里的非玩家堡（18 号）绝不注入。
	for _, silo := range svc1.silos {
		if silo.ID == ActiveSiloID {
			continue
		}
		if len(silo.Relics) != 0 {
			t.Fatalf("protected silo %d received %d relics", silo.ID, len(silo.Relics))
		}
	}
}

// TestDifferentSeedsProduceDifferentRelics 不同种子应布出不同遗物组合。
// 两条种子的结果由代码与清单完全决定，不存在偶发碰撞。
func TestDifferentSeedsProduceDifferentRelics(t *testing.T) {
	svc1 := newCatalogService(t, filepath.Join(t.TempDir(), "relic-s1.db"))
	if _, err := svc1.CreateGame(twoSiloRequest(42)); err != nil {
		t.Fatalf("create game seed 42: %v", err)
	}
	svc2 := newCatalogService(t, filepath.Join(t.TempDir(), "relic-s2.db"))
	if _, err := svc2.CreateGame(twoSiloRequest(43)); err != nil {
		t.Fatalf("create game seed 43: %v", err)
	}
	if reflect.DeepEqual(relicSignature(svc1.silo.Relics), relicSignature(svc2.silo.Relics)) {
		t.Fatalf("seeds 42 and 43 produced identical relic layouts: %v", relicSignature(svc1.silo.Relics))
	}
}

// TestRelicsNeverInjectedIntoProtectedSilos 三堡局（含完整一号与 18 号）
// 下遗物只进玩家堡。
func TestRelicsNeverInjectedIntoProtectedSilos(t *testing.T) {
	svc := newCatalogService(t, filepath.Join(t.TempDir(), "relic-protected.db"))
	if _, err := svc.CreateGame(threeSiloRequest(77)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if len(svc.silos) != 3 {
		t.Fatalf("expected three silos, got %d", len(svc.silos))
	}
	if len(svc.silo.Relics) == 0 {
		t.Fatal("player silo must receive relics")
	}
	for _, silo := range svc.silos {
		if silo.ID == ActiveSiloID {
			continue
		}
		if len(silo.Relics) != 0 {
			t.Fatalf("protected silo %d received %d relics", silo.ID, len(silo.Relics))
		}
	}
}

// TestRelicsSurviveSaveLoadRoundTrip 遗物行经落库/读档完整还原，且读档
// 不依赖清单加载（旧档语义）。
func TestRelicsSurviveSaveLoadRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "relic-roundtrip.db")
	svc := newCatalogService(t, dbPath)
	if _, err := svc.CreateGame(twoSiloRequest(99)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	want := relicSignature(svc.silo.Relics)
	if len(want) == 0 {
		t.Fatal("expected seeded relics before save")
	}

	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got := relicSignature(fresh.silo.Relics); !reflect.DeepEqual(got, want) {
		t.Fatalf("relics changed across save/load:\nbefore %v\nafter  %v", want, got)
	}
}

package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validTestRelicFile 最小合法清单：两条遗物、两个位置，覆盖主要字段。
func validTestRelicFile() relicFile {
	return relicFile{
		Version: 1,
		Spawn:   relicSpawnBudget{Min: 1, Max: 2},
		Locations: []relicLocation{
			{ID: "sealed_vault", Name: "地下封存设施", Zone: "Lower"},
			{ID: "evidence_room", Name: "司法部证物库", Zone: "Upper", Dept: "Judicial"},
		},
		Relics: []relicDefinition{
			{
				ID: "test_drive", Name: "测试硬盘", Description: "用于校验的存储介质",
				Category: "storage_medium", Locations: []string{"sealed_vault"}, Weight: 3,
				Effects: map[string]float64{"suspicion_level": 0.05}, Fragments: []string{"IT_3"},
				Confiscable: true,
			},
			{
				ID: "test_book", Name: "测试旧书", Description: "用于校验的旧版书籍",
				Category: "old_book", Locations: []string{"evidence_room"}, Weight: 2,
				Fragments: []string{"Judicial_3"}, Confiscable: true,
			},
		},
	}
}

func TestCompileRelicCatalogAcceptsValidFile(t *testing.T) {
	runtime, err := compileRelicCatalog(validTestRelicFile())
	if err != nil {
		t.Fatalf("valid catalog rejected: %v", err)
	}
	if !runtime.Enabled || len(runtime.Relics) != 2 || len(runtime.Locations) != 2 {
		t.Fatalf("unexpected runtime: enabled=%v relics=%d locations=%d", runtime.Enabled, len(runtime.Relics), len(runtime.Locations))
	}
	if runtime.TotalWeight != 5 {
		t.Fatalf("total weight = %d, want 5", runtime.TotalWeight)
	}
	if runtime.Locations["evidence_room"].Dept != "Judicial" {
		t.Fatalf("dept anchor lost: %+v", runtime.Locations["evidence_room"])
	}
}

func TestCompileRelicCatalogRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*relicFile)
		wantErr string
	}{
		{"wrong version", func(f *relicFile) { f.Version = 2 }, "unsupported version"},
		{"duplicate location", func(f *relicFile) { f.Locations[1].ID = "sealed_vault" }, "duplicate location"},
		{"empty location id", func(f *relicFile) { f.Locations[0].ID = "  " }, "empty id"},
		{"empty location name", func(f *relicFile) { f.Locations[0].Name = "" }, "empty name"},
		{"invalid zone anchor", func(f *relicFile) { f.Locations[0].Zone = "Bottom" }, "invalid zone"},
		{"unknown dept anchor", func(f *relicFile) { f.Locations[1].Dept = "Sheriff" }, "unknown dept anchor"},
		{"duplicate relic id", func(f *relicFile) { f.Relics[1].ID = "test_drive" }, "duplicate relic id"},
		{"empty relic id", func(f *relicFile) { f.Relics[0].ID = "" }, "empty id"},
		{"empty relic name", func(f *relicFile) { f.Relics[0].Name = "" }, "empty name"},
		{"empty description", func(f *relicFile) { f.Relics[0].Description = " " }, "empty description"},
		{"unknown category", func(f *relicFile) { f.Relics[0].Category = "artifact" }, "unknown category"},
		{"no locations declared", func(f *relicFile) { f.Relics[0].Locations = nil }, "no locations declared"},
		{"unknown location ref", func(f *relicFile) { f.Relics[0].Locations = []string{"moon_base"} }, "unknown location"},
		{"weight too low", func(f *relicFile) { f.Relics[0].Weight = 0 }, "out of range"},
		{"weight too high", func(f *relicFile) { f.Relics[0].Weight = 11 }, "out of range"},
		{"unsupported effect metric", func(f *relicFile) { f.Relics[0].Effects = map[string]float64{"rebellion_speed": 0.1} }, "unsupported effect metric"},
		{"effect value out of range", func(f *relicFile) { f.Relics[0].Effects = map[string]float64{"cohesion": 1.5} }, "out of range [-1,1]"},
		{"unknown fragment", func(f *relicFile) { f.Relics[0].Fragments = []string{"IT_99"} }, "unknown fragment"},
		{"spawn min above max", func(f *relicFile) { f.Spawn = relicSpawnBudget{Min: 3, Max: 1} }, "invalid budget"},
		{"spawn max above twice relic count", func(f *relicFile) { f.Spawn = relicSpawnBudget{Min: 1, Max: 5} }, "exceeds twice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := validTestRelicFile()
			tc.mutate(&file)
			if _, err := compileRelicCatalog(file); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestCompileRelicCatalogAggregatesProblems 多个错误必须一次全部报出，
// 不能挤牙膏式一次暴露一个。
func TestCompileRelicCatalogAggregatesProblems(t *testing.T) {
	file := validTestRelicFile()
	file.Relics[0].Category = "artifact"
	file.Relics[1].Weight = 99
	_, err := compileRelicCatalog(file)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !strings.Contains(err.Error(), "unknown category") || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error must aggregate all problems, got: %v", err)
	}
}

func TestRelicCatalogLoadsFromRepo(t *testing.T) {
	svc := NewGameService(nil)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.loadRelicCatalog(dirs); err != nil {
		t.Fatalf("load relic catalog: %v", err)
	}
	catalog := svc.relicCatalog
	if !catalog.Enabled {
		t.Fatal("catalog must be enabled after load")
	}
	if len(catalog.Relics) != 12 || len(catalog.Locations) != 10 {
		t.Fatalf("unexpected catalog size: relics=%d locations=%d", len(catalog.Relics), len(catalog.Locations))
	}
	if catalog.Spawn.Min != 6 || catalog.Spawn.Max != 9 {
		t.Fatalf("unexpected spawn budget: %+v", catalog.Spawn)
	}
	var drive *relicDefinition
	for i := range catalog.Relics {
		if catalog.Relics[i].ID == "sealed_hard_drive" {
			drive = &catalog.Relics[i]
			break
		}
	}
	if drive == nil {
		t.Fatal("sealed_hard_drive missing from catalog")
	}
	if drive.Category != "storage_medium" || drive.Weight != 1 || len(drive.Fragments) != 1 || drive.Fragments[0] != "IT_3" {
		t.Fatalf("sealed_hard_drive parsed wrong: %+v", drive)
	}
	for _, ref := range drive.Locations {
		if _, ok := catalog.Locations[ref]; !ok {
			t.Fatalf("sealed_hard_drive references unknown location %q", ref)
		}
	}
}

func TestRelicCatalogMissingFailsClosed(t *testing.T) {
	svc := NewGameService(nil)
	tmp := t.TempDir()
	dirs := map[string]string{"mechanics": filepath.Join(tmp, "events", "mechanics")}
	err := svc.loadRelicCatalog(dirs)
	if err == nil || !strings.Contains(err.Error(), "relic catalog is missing") {
		t.Fatalf("missing catalog must fail closed, got: %v", err)
	}
	if svc.relicCatalog.Enabled {
		t.Fatal("catalog must stay disabled after failed load")
	}
}

func TestRelicCatalogRejectsUnknownFields(t *testing.T) {
	svc := NewGameService(nil)
	tmp := t.TempDir()
	eventsDir := filepath.Join(tmp, "events")
	if err := os.MkdirAll(filepath.Join(eventsDir, "mechanics"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":1,"spawn":{"min":0,"max":0},"locations":[],"relics":[],"loot":true}`)
	if err := os.WriteFile(filepath.Join(eventsDir, "relics.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	dirs := map[string]string{"mechanics": filepath.Join(eventsDir, "mechanics")}
	err := svc.loadRelicCatalog(dirs)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown fields must be rejected, got: %v", err)
	}
}

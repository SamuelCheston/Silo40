package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// validTestFragmentFile 最小合法内容库：覆盖 ALL_FRAGMENTS 全部 26 条。
func validTestFragmentFile() fragmentFile {
	file := fragmentFile{Version: 1}
	for _, id := range model.ALL_FRAGMENTS {
		file.Fragments = append(file.Fragments, fragmentTemplate{
			ID:           id,
			Title:        "标题 " + id,
			Department:   strings.SplitN(id, "_", 2)[0],
			Tier:         "中层",
			Theme:        "Covenant 与执法",
			DocumentForm: "档案",
			BodyTemplate: "正文 " + id,
			LockedClaims: []string{"事实锚点 " + id},
			NextLeads:    "指向 " + id,
		})
	}
	return file
}

func TestCompileFragmentLibraryAcceptsValidFile(t *testing.T) {
	runtime, err := compileFragmentLibrary(validTestFragmentFile())
	if err != nil {
		t.Fatalf("valid library rejected: %v", err)
	}
	if !runtime.Enabled || len(runtime.Fragments) != len(model.ALL_FRAGMENTS) {
		t.Fatalf("unexpected runtime: enabled=%v fragments=%d", runtime.Enabled, len(runtime.Fragments))
	}
	entry, ok := runtime.byID["Judicial_1"]
	if !ok || entry.Title != "标题 Judicial_1" {
		t.Fatalf("byID lookup failed: %+v", entry)
	}
}

func TestCompileFragmentLibraryRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*fragmentFile)
		wantErr string
	}{
		{"wrong version", func(f *fragmentFile) { f.Version = 2 }, "unsupported version"},
		{"duplicate id", func(f *fragmentFile) { f.Fragments[1].ID = f.Fragments[0].ID }, "duplicate fragment id"},
		{"empty id", func(f *fragmentFile) { f.Fragments[0].ID = "  " }, "empty id"},
		{"unknown id", func(f *fragmentFile) { f.Fragments[0].ID = "Mayor_99" }, "not in ALL_FRAGMENTS"},
		{"missing entry", func(f *fragmentFile) { f.Fragments = f.Fragments[1:] }, "missing from library"},
		{"empty title", func(f *fragmentFile) { f.Fragments[0].Title = "" }, "empty title"},
		{"unknown department", func(f *fragmentFile) { f.Fragments[0].Department = "Sheriff" }, "unknown department"},
		{"invalid tier", func(f *fragmentFile) { f.Fragments[0].Tier = "顶层" }, "invalid tier"},
		{"unknown theme", func(f *fragmentFile) { f.Fragments[0].Theme = "未登记主题" }, "unknown theme"},
		{"unknown document form", func(f *fragmentFile) { f.Fragments[0].DocumentForm = "口供" }, "unknown document form"},
		{"empty body", func(f *fragmentFile) { f.Fragments[0].BodyTemplate = " " }, "empty body_template"},
		{"no locked claims", func(f *fragmentFile) { f.Fragments[0].LockedClaims = nil }, "no locked_claims"},
		{"empty locked claim", func(f *fragmentFile) { f.Fragments[0].LockedClaims = []string{"  "} }, "empty locked_claims"},
		{"empty next leads", func(f *fragmentFile) { f.Fragments[0].NextLeads = "" }, "empty next_leads"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := validTestFragmentFile()
			tc.mutate(&file)
			if _, err := compileFragmentLibrary(file); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestCompileFragmentLibraryAggregatesProblems 多个错误必须一次全部报出。
func TestCompileFragmentLibraryAggregatesProblems(t *testing.T) {
	file := validTestFragmentFile()
	file.Fragments[0].Title = ""
	file.Fragments[1].Tier = "顶层"
	_, err := compileFragmentLibrary(file)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !strings.Contains(err.Error(), "empty title") || !strings.Contains(err.Error(), "invalid tier") {
		t.Fatalf("error must aggregate all problems, got: %v", err)
	}
}

// TestFragmentLibraryLoadsFromRepo 仓库内真实文件必须通过校验，并把标题
// 登记进引擎注册表 (mechanic 消息可读化的来源)。
func TestFragmentLibraryLoadsFromRepo(t *testing.T) {
	svc := NewGameService(nil)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.loadFragmentLibrary(dirs); err != nil {
		t.Fatalf("load fragment library: %v", err)
	}
	library := svc.fragmentLibrary
	if !library.Enabled || len(library.Fragments) != len(model.ALL_FRAGMENTS) {
		t.Fatalf("unexpected library: enabled=%v fragments=%d", library.Enabled, len(library.Fragments))
	}
	entry := library.byID["Judicial_1"]
	if entry.Title == "" || entry.Department != "Judicial" || entry.BodyTemplate == "" {
		t.Fatalf("Judicial_1 parsed wrong: %+v", entry)
	}
	if got := engine.FragmentTitle("Judicial_1"); got != entry.Title {
		t.Fatalf("engine title registry = %q, want %q", got, entry.Title)
	}
	meta := svc.FragmentLibraryMeta()
	if len(meta) != len(model.ALL_FRAGMENTS) {
		t.Fatalf("meta size = %d, want %d", len(meta), len(model.ALL_FRAGMENTS))
	}
}

func TestFragmentLibraryMissingFailsClosed(t *testing.T) {
	svc := NewGameService(nil)
	tmp := t.TempDir()
	dirs := map[string]string{"mechanics": filepath.Join(tmp, "events", "mechanics")}
	err := svc.loadFragmentLibrary(dirs)
	if err == nil || !strings.Contains(err.Error(), "fragment library is missing") {
		t.Fatalf("missing library must fail closed, got: %v", err)
	}
	if svc.fragmentLibrary.Enabled {
		t.Fatal("library must stay disabled after failed load")
	}
}

func TestFragmentLibraryRejectsUnknownFields(t *testing.T) {
	svc := NewGameService(nil)
	tmp := t.TempDir()
	eventsDir := filepath.Join(tmp, "events")
	if err := os.MkdirAll(filepath.Join(eventsDir, "mechanics"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":1,"fragments":[],"loot":true}`)
	if err := os.WriteFile(filepath.Join(eventsDir, "fragments.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	dirs := map[string]string{"mechanics": filepath.Join(eventsDir, "mechanics")}
	err := svc.loadFragmentLibrary(dirs)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown fields must be rejected, got: %v", err)
	}
}

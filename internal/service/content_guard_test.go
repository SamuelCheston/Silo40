package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"silo40/internal/engine"
)

// These tests are the foolproof gate against silent content loss. The
// historical failure mode (commit bd18be1 era): three events lacked triggers,
// ValidateContentEventDefinition rejected one of them, syncContentDefinitions
// aborted, and EVERY file-driven event silently stopped loading. A second
// failure mode: events/rules/ existed on disk but was filtered out of the
// content whitelist, so the four rule events were dead content for months.
//
// The guards below fail loudly and name the offending file/event when:
//   - any content group on disk is neither content-loaded nor the mechanics
//     group (whitelist regression / dead content),
//   - any definition fails to load or validate,
//   - any loaded definition lacks a usable trigger or effects,
//   - two definitions share an event id (silent shadowing in
//     findContentDefinition / schedule_event resolution).

// eventsRootFromDirs derives the events/ root directory from discovered
// group paths (grouped layout: <root>/<group>).
func eventsRootFromDirs(t *testing.T, dirs map[string]string) string {
	t.Helper()
	for _, dir := range dirs {
		return filepath.Dir(dir)
	}
	t.Fatalf("no content directories discovered")
	return ""
}

func TestContentGuardNoDeadGroups(t *testing.T) {
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	root := eventsRootFromDirs(t, dirs)
	loaded := contentEventDirectories(dirs)

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read events root %s: %v", root, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		group := entry.Name()
		groupDir := filepath.Join(root, group)
		hasFiles, err := directoryHasContentEventFiles(groupDir)
		if err != nil || !hasFiles {
			continue
		}
		if group == "mechanics" {
			// Loaded separately by the MechanicEngine; guard it below.
			continue
		}
		if group == "protocol" {
			// Order 条款资产，由 loadProtocolClauses 独立加载 (不是事件组)；
			// guard it below.
			continue
		}
		if loaded[group] == "" {
			t.Errorf("dead content: group %q has event files on disk (%s) but is excluded from the content whitelist", group, groupDir)
		}
	}

	// Positive anchors: the known groups must actually load, so a broken
	// discovery (empty result) cannot pass the dead-group scan vacuously.
	for _, group := range []string{"histories", "special", "crisis", "rules"} {
		if loaded[group] == "" {
			t.Errorf("expected content group %q to be loaded", group)
		}
	}
}

func TestContentGuardMechanicsLoad(t *testing.T) {
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	mechanicsDir := dirs["mechanics"]
	if mechanicsDir == "" {
		t.Fatalf("mechanics group not discovered under %s", eventsRootFromDirs(t, dirs))
	}
	defs, err := engine.LoadMechanicDefinitions(mechanicsDir)
	if err != nil {
		t.Fatalf("mechanics definitions failed to load: %v", err)
	}
	if len(defs) == 0 {
		t.Fatalf("expected mechanics definitions to be loaded from %s", mechanicsDir)
	}
}

// TestContentGuardProtocolLoads guards the Order clause asset: the protocol
// group must exist and load versioned clauses (dead-protocol regression gate,
// mirroring the mechanics guard).
func TestContentGuardProtocolLoads(t *testing.T) {
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	protocolDir := dirs["protocol"]
	if protocolDir == "" {
		t.Fatalf("protocol group not discovered under %s", eventsRootFromDirs(t, dirs))
	}
	clauses, version, err := engine.LoadProtocolDir(protocolDir)
	if err != nil {
		t.Fatalf("protocol clauses failed to load: %v", err)
	}
	if len(clauses) == 0 || version == 0 {
		t.Fatalf("expected versioned protocol clauses from %s", protocolDir)
	}
}

func TestContentGuardEveryEventLoadsValidAndTriggerable(t *testing.T) {
	svc := bootstrapRealContent(t)
	if len(svc.contentDefinitions) == 0 {
		t.Fatalf("no content definitions loaded; bootstrap silently lost content")
	}

	seenIDs := map[string]string{}
	for _, def := range svc.contentDefinitions {
		where := def.SourceGroup + "/" + def.SourceFile + " (event " + def.EventID + ")"

		// Defense in depth: validation already ran at load time; re-check the
		// two usability conditions so a future validator regression still
		// fails here with per-event diagnostics.
		if def.Trigger.Type == "" && !def.ScriptHooks.CanTrigger {
			t.Errorf("%s: no usable trigger (needs structural trigger or script.canTrigger)", where)
		}
		if len(def.Effects) == 0 && !def.ScriptHooks.Apply {
			t.Errorf("%s: no effects and no script.apply", where)
		}

		if prev, dup := seenIDs[def.EventID]; dup {
			t.Errorf("duplicate event id %q: %s shadows %s (schedule_event and state lookups resolve to the first match)", def.EventID, where, prev)
		}
		seenIDs[def.EventID] = where
	}

	// Report the loaded inventory so a shrinking content set is visible in
	// test output when this guard is investigated.
	keys := make([]string, 0, len(svc.contentDefinitions))
	for _, def := range svc.contentDefinitions {
		keys = append(keys, def.Key)
	}
	sort.Strings(keys)
	t.Logf("loaded %d content definitions: %v", len(keys), keys)
}

// TestContentGuardEventTriggerReferencesResolve 防止"有内容却触发不到"的键
// 拼写漂移：每个 event_triggered 的 category:event_id 必须指向已加载的
// 内容定义，或指向明确的运行时完成键组 (player_action / observation)。
// 例：histories 组被误写成 history 时，条件永远不会成立，事件静默死亡。
func TestContentGuardEventTriggerReferencesResolve(t *testing.T) {
	svc := bootstrapRealContent(t)
	// 引擎在载入时会把目录组折叠成运行时类别：histories→history、
	// player_actions→player_action、events→special。引用比对必须用
	// 同一口径，否则 history 组会被误报成死键。
	normalize := func(group string) string {
		switch strings.ToLower(strings.TrimSpace(group)) {
		case "histories", "history":
			return "history"
		case "player_actions", "player_action":
			return "player_action"
		case "events":
			return "special"
		default:
			return strings.ToLower(strings.TrimSpace(group))
		}
	}
	known := map[string]string{}
	for _, def := range svc.contentDefinitions {
		known[normalize(def.SourceGroup)+":"+def.EventID] = def.SourceGroup + "/" + def.SourceFile
	}
	// 运行时写入的完成键组：不是内容文件，由服务层动作完成时落账。
	runtimeGroups := map[string]bool{
		"player_action": true, // 普通/职务/职业动作完成键（含职业动作）
		"observation":   true, // 首次本地观察等观察键
	}

	var walk func(def engine.ContentEventDefinition, tr engine.ContentTrigger)
	walk = func(def engine.ContentEventDefinition, tr engine.ContentTrigger) {
		if tr.Type == "event_triggered" {
			key := tr.Category + ":" + tr.EventID
			if _, ok := known[key]; !ok && !runtimeGroups[tr.Category] {
				t.Errorf("%s/%s (event %s): event_triggered references unknown key %q",
					def.SourceGroup, def.SourceFile, def.EventID, key)
			}
		}
		for _, child := range tr.Conditions {
			walk(def, child)
		}
	}
	for _, def := range svc.contentDefinitions {
		walk(def, def.Trigger)
	}
}

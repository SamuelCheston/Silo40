package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseContextLevel(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want ContextLevel
		ok   bool
	}{
		{"", DefaultContextLevel, true},
		{"minimal", ContextMinimal, true},
		{"MIN", ContextMinimal, true},
		{"standard", ContextStandard, true},
		{"std", ContextStandard, true},
		{"rich", ContextRich, true},
		{"world", ContextWorld, true},
		{"bogus", DefaultContextLevel, false},
	} {
		got, err := ParseContextLevel(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Fatalf("ParseContextLevel(%q) = %v,%v want %v,ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
	if !ContextWorld.AtLeast(ContextMinimal) || ContextStandard.AtLeast(ContextRich) {
		t.Fatalf("context level nesting broken")
	}
}

func TestLoadEnvContextDefaults(t *testing.T) {
	// Unset: standard level + historical-shape switches.
	for _, k := range []string{EnvContext, EnvCtxMemory, EnvCtxProtocol, EnvCtxEventLimit, EnvCtxPeerIntel} {
		t.Setenv(k, "")
	}
	cfg := LoadEnv()
	if cfg.Context != ContextStandard {
		t.Fatalf("default context level: got %q want %q", cfg.Context, ContextStandard)
	}
	want := DefaultContextOptions()
	if cfg.ContextOptions != want {
		t.Fatalf("default context options: got %+v want %+v", cfg.ContextOptions, want)
	}

	// Explicit values win.
	t.Setenv(EnvContext, "world")
	t.Setenv(EnvCtxMemory, "off")
	t.Setenv(EnvCtxProtocol, "on")
	t.Setenv(EnvCtxEventLimit, "12")
	t.Setenv(EnvCtxPeerIntel, "on")
	cfg = LoadEnv()
	if cfg.Context != ContextWorld {
		t.Fatalf("explicit context level ignored: %q", cfg.Context)
	}
	if cfg.ContextOptions.PersonalMemory || !cfg.ContextOptions.ProtocolClauses ||
		cfg.ContextOptions.EventSummaryLimit != 12 || !cfg.ContextOptions.PeerSiloIntel {
		t.Fatalf("explicit context switches ignored: %+v", cfg.ContextOptions)
	}

	// Invalid level falls back to the default with a parse error surfaced.
	t.Setenv(EnvContext, "bogus")
	cfg = LoadEnv()
	if cfg.ContextParseErr == nil || cfg.Context != DefaultContextLevel {
		t.Fatalf("invalid context level must fall back with error: %v %q", cfg.ContextParseErr, cfg.Context)
	}
}

// fullContextData returns a provider payload with every section populated.
func fullContextData() ContextData {
	return ContextData{
		RecentEvents: []EventSummary{
			{Year: 122, Month: 3, Category: "crisis", Title: "e1", Description: "d1"},
			{Year: 122, Month: 4, Category: "special", Title: "e2", Description: "d2"},
			{Year: 122, Month: 5, Category: "history", Title: "e3", Description: "d3"},
		},
		MainlineEvents: []EventSummary{
			{Year: 121, Month: 1, Category: "history", Title: "m1"},
		},
		ProtocolClauses: []ClauseSummary{
			{ID: "P-01", Category: "prohibition", Severity: "major", Description: "clause"},
		},
		WorldSetting:  "setting text",
		PeerSilos:     []PeerSiloBrief{{ID: 18, Name: "18 号地堡", Relation: "peer", Status: "normal"}},
		LongTermLines: []string{"line A"},
		PeerIntel:     []string{"rumor text"},
	}
}

func buildRequestAt(t *testing.T, level ContextLevel, opts ContextOptions, withProvider bool) map[string]any {
	t.Helper()
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelFull,
		ContextLevel: level, ContextOptions: &opts})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	if withProvider {
		a.SetContextProvider(func(uint) ContextData { return fullContextData() })
	}
	silo, view, _ := testSiloAndView(t)
	_, user := a.buildRequest(view, silo, testCandidates())
	var payload map[string]any
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user payload not JSON: %v", err)
	}
	return payload
}

func TestBuildContextLevels(t *testing.T) {
	opts := DefaultContextOptions()

	minimal := buildRequestAt(t, ContextMinimal, opts, true)
	for _, key := range []string{"actor", "candidates"} {
		if _, ok := minimal[key]; !ok {
			t.Fatalf("minimal payload missing %q", key)
		}
	}
	for _, key := range []string{"silo", "departments", "factions", "recent_events", "mainline",
		"world_setting", "peer_silos", "long_term_lines", "protocol_clauses", "peer_intel"} {
		if _, ok := minimal[key]; ok {
			t.Fatalf("minimal payload must not carry %q", key)
		}
	}

	standard := buildRequestAt(t, ContextStandard, opts, true)
	for _, key := range []string{"silo", "departments", "factions", "recent_events"} {
		if _, ok := standard[key]; !ok {
			t.Fatalf("standard payload missing %q", key)
		}
	}
	if events := standard["recent_events"].([]any); len(events) != 3 {
		t.Fatalf("recent_events not capped by provider data: %d", len(events))
	}
	for _, key := range []string{"mainline", "world_setting", "peer_silos"} {
		if _, ok := standard[key]; ok {
			t.Fatalf("standard payload must not carry %q", key)
		}
	}

	rich := buildRequestAt(t, ContextRich, opts, true)
	if _, ok := rich["mainline"]; !ok {
		t.Fatalf("rich payload missing mainline")
	}
	if _, ok := rich["world_setting"]; ok {
		t.Fatalf("rich payload must not carry world_setting")
	}

	world := buildRequestAt(t, ContextWorld, opts, true)
	for _, key := range []string{"world_setting", "peer_silos", "long_term_lines"} {
		if _, ok := world[key]; !ok {
			t.Fatalf("world payload missing %q", key)
		}
	}
}

func TestBuildContextSwitches(t *testing.T) {
	// Personal memory off: no traits/fragments/relations at any level.
	opts := DefaultContextOptions()
	opts.PersonalMemory = false
	payload := buildRequestAt(t, ContextWorld, opts, true)
	actor := payload["actor"].(map[string]any)
	for _, key := range []string{"traits", "known_fragments", "relations"} {
		if _, ok := actor[key]; ok {
			t.Fatalf("personal memory off must drop actor.%s", key)
		}
	}

	// Protocol + peer intel switches are orthogonal to the level.
	opts = DefaultContextOptions()
	opts.ProtocolClauses = true
	opts.PeerSiloIntel = true
	payload = buildRequestAt(t, ContextMinimal, opts, true)
	if _, ok := payload["protocol_clauses"]; !ok {
		t.Fatalf("protocol switch must add clauses even at minimal")
	}
	if _, ok := payload["peer_intel"]; !ok {
		t.Fatalf("peer intel switch must add intel even at minimal")
	}

	// Event limit: caps entries; 0 disables the section.
	opts = DefaultContextOptions()
	opts.EventSummaryLimit = 2
	payload = buildRequestAt(t, ContextStandard, opts, true)
	if events := payload["recent_events"].([]any); len(events) != 2 {
		t.Fatalf("event limit 2: got %d entries", len(events))
	}
	opts.EventSummaryLimit = 0
	payload = buildRequestAt(t, ContextStandard, opts, true)
	if _, ok := payload["recent_events"]; ok {
		t.Fatalf("event limit 0 must drop recent_events")
	}

	// No provider wired: standard level degrades to silo-only sections,
	// no panic, no event section.
	payload = buildRequestAt(t, ContextWorld, DefaultContextOptions(), false)
	for _, key := range []string{"silo", "departments", "factions"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("provider-less payload missing %q", key)
		}
	}
	if _, ok := payload["recent_events"]; ok {
		t.Fatalf("provider-less payload must not carry recent_events")
	}
}

// The minimal level must truly serialize less than the richer levels —
// sections are omitted, not blanked.
func TestBuildContextSizeMonotonic(t *testing.T) {
	opts := DefaultContextOptions()
	size := func(level ContextLevel) int {
		db := newAdvisorTestDB(t)
		a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelFull,
			ContextLevel: level, ContextOptions: &opts})
		if err != nil {
			t.Fatalf("advisor: %v", err)
		}
		defer a.Close()
		a.SetContextProvider(func(uint) ContextData { return fullContextData() })
		silo, view, _ := testSiloAndView(t)
		_, user := a.buildRequest(view, silo, testCandidates())
		return len(user)
	}
	minimal, standard := size(ContextMinimal), size(ContextStandard)
	rich, world := size(ContextRich), size(ContextWorld)
	if !(minimal < standard && standard < rich && rich < world) {
		t.Fatalf("payload sizes not monotonic: minimal=%d standard=%d rich=%d world=%d",
			minimal, standard, rich, world)
	}
}

func TestSetContextLoggedAndRestored(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelOff})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	opts := ContextOptions{PersonalMemory: false, ProtocolClauses: true, EventSummaryLimit: 9, PeerSiloIntel: true}
	if err := a.SetContext(ContextRich, opts, "api"); err != nil {
		t.Fatalf("set context: %v", err)
	}
	if a.GetContextLevel() != ContextRich || a.GetContextOptions() != opts {
		t.Fatalf("context config not applied: %q %+v", a.GetContextLevel(), a.GetContextOptions())
	}
	a.Close()

	entries, err := LoadEntries(db)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d (%v)", len(entries), err)
	}
	e := entries[0]
	if e.Kind != KindContextConfig || e.ConfigSource != "api" || e.ConfigContext == "" {
		t.Fatalf("bad context config entry: %+v", e)
	}

	// A fresh advisor restores the switches from the log on resume; a later
	// participation config entry must still apply on top.
	b, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelOff})
	if err != nil {
		t.Fatalf("advisor b: %v", err)
	}
	defer b.Close()
	if err := b.SetLevel(LevelKey, "api"); err != nil {
		t.Fatalf("set level: %v", err)
	}
	entries, _ = LoadEntries(db)

	c, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelOff})
	if err != nil {
		t.Fatalf("advisor c: %v", err)
	}
	defer c.Close()
	c.ResumeFrom(entries)
	if c.GetContextLevel() != ContextRich || c.GetContextOptions() != opts {
		t.Fatalf("resume did not restore context config: %q %+v", c.GetContextLevel(), c.GetContextOptions())
	}
	if c.GetLevel() != LevelKey {
		t.Fatalf("resume lost participation level: %q", c.GetLevel())
	}
}

// A malformed context payload in the log must never wedge replay: the entry
// is consumed and defaults stay in force.
func TestResumeSurvivesBadContextPayload(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelOff})
	if err != nil {
		t.Fatalf("advisor: %v", err)
	}
	if err := AppendEntry(db, &DecisionLog{Kind: KindContextConfig, ConfigContext: "{not json"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	entries, _ := LoadEntries(db)
	a.ResumeFrom(entries)
	if err := a.Error(); err != nil {
		t.Fatalf("bad context payload must not poison replay: %v", err)
	}
	if a.GetContextLevel() != DefaultContextLevel {
		t.Fatalf("bad payload must leave defaults in force, got %q", a.GetContextLevel())
	}
	a.Close()
}

// The system prompt (output contract) is identical across all levels: the
// level only changes the user payload.
func TestSystemPromptStableAcrossLevels(t *testing.T) {
	opts := DefaultContextOptions()
	var systems []string
	for _, level := range []ContextLevel{ContextMinimal, ContextStandard, ContextRich, ContextWorld} {
		db := newAdvisorTestDB(t)
		a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelFull,
			ContextLevel: level, ContextOptions: &opts})
		if err != nil {
			t.Fatalf("advisor: %v", err)
		}
		silo, view, _ := testSiloAndView(t)
		system, _ := a.buildRequest(view, silo, testCandidates())
		systems = append(systems, system)
		a.Close()
	}
	for i := 1; i < len(systems); i++ {
		if systems[i] != systems[0] {
			t.Fatalf("system prompt differs at level index %d", i)
		}
	}
	if !strings.Contains(systems[0], "JSON") {
		t.Fatalf("system prompt lost the output contract")
	}
}

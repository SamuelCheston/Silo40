package llm

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"

	"gorm.io/gorm"
)

func TestLoadEnvReleaseDefaults(t *testing.T) {
	// Unset participation defaults to key (author-approved release form);
	// explicit values still win, including off.
	for _, tc := range []struct {
		env  string
		want Level
	}{
		{"", LevelKey}, {"off", LevelOff}, {"events", LevelEvent},
		{"key", LevelKey}, {"full", LevelFull},
	} {
		t.Setenv("SILO40_LLM_PARTICIPATION", tc.env)
		cfg := LoadEnv()
		if cfg.Level != tc.want {
			t.Fatalf("participation %q: got %v want %v", tc.env, cfg.Level, tc.want)
		}
	}
	// Timeout default is sized for local endpoints.
	a, err := NewAdvisor(Config{DB: newAdvisorTestDB(t), Client: &FakeClient{}, Mode: ModeShadow, Level: LevelKey})
	if err != nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	if a.timeout != DefaultTimeoutMS*time.Millisecond {
		t.Fatalf("default timeout: got %v want %v", a.timeout, DefaultTimeoutMS*time.Millisecond)
	}
}

func TestLoadEnvLatencyMitigationDefaults(t *testing.T) {
	t.Setenv(EnvWarmup, "")
	t.Setenv(EnvPrefetch, "")
	cfg := LoadEnv()
	if !cfg.Warmup {
		t.Fatal("warmup must remain enabled by default")
	}
	if cfg.Prefetch {
		t.Fatal("prefetch must default off to avoid background model spend")
	}

	t.Setenv(EnvPrefetch, "on")
	if cfg := LoadEnv(); !cfg.Prefetch {
		t.Fatal("explicit prefetch=on must enable prefetch")
	}
	t.Setenv(EnvPrefetch, "off")
	if cfg := LoadEnv(); cfg.Prefetch {
		t.Fatal("explicit prefetch=off must disable prefetch")
	}
	t.Setenv(EnvWarmup, "off")
	if cfg := LoadEnv(); cfg.Warmup {
		t.Fatal("explicit warmup=off must disable warmup")
	}
}

func TestParseModeAndLevel(t *testing.T) {
	for _, tc := range []struct {
		in   string
		mode Mode
		ok   bool
	}{
		{"", ModeOff, true}, {"off", ModeOff, true}, {"SHADOW", ModeShadow, true},
		{"on", ModeOn, true}, {"bogus", ModeOff, false},
	} {
		m, err := ParseMode(tc.in)
		if (err == nil) != tc.ok || m != tc.mode {
			t.Fatalf("ParseMode(%q) = %v,%v want %v,ok=%v", tc.in, m, err, tc.mode, tc.ok)
		}
	}
	for _, tc := range []struct {
		in    string
		level Level
		ok    bool
	}{
		{"", LevelOff, true}, {"events", LevelEvent, true}, {"key", LevelKey, true},
		{"full", LevelFull, true}, {"bogus", LevelOff, false},
	} {
		l, err := ParseLevel(tc.in)
		if (err == nil) != tc.ok || l != tc.level {
			t.Fatalf("ParseLevel(%q) = %v,%v want %v,ok=%v", tc.in, l, err, tc.level, tc.ok)
		}
	}
	if !LevelFull.AtLeast(LevelEvent) || LevelEvent.AtLeast(LevelKey) {
		t.Fatalf("level nesting broken")
	}
}

func TestParseResponseValidation(t *testing.T) {
	ok, err := ParseResponse(`{"scores":[{"candidate_id":0,"score":3},{"candidate_id":1,"score":8}]}`, 2)
	if err != nil || len(ok.Scores) != 2 {
		t.Fatalf("valid response rejected: %v", err)
	}
	// Prose-wrapped JSON is salvaged.
	if _, err := ParseResponse("Sure! Here you go:\n```\n{\"scores\":[{\"candidate_id\":0,\"score\":5}]}\n```", 1); err != nil {
		t.Fatalf("salvage failed: %v", err)
	}
	for name, raw := range map[string]string{
		"missing candidate":  `{"scores":[{"candidate_id":0,"score":3}]}`,
		"extra candidate":    `{"scores":[{"candidate_id":0,"score":3},{"candidate_id":1,"score":1},{"candidate_id":2,"score":1}]}`,
		"score out of range": `{"scores":[{"candidate_id":0,"score":11},{"candidate_id":1,"score":1}]}`,
		"duplicate":          `{"scores":[{"candidate_id":0,"score":3},{"candidate_id":0,"score":4}]}`,
		"bad factor key":     `{"scores":[{"candidate_id":0,"score":3},{"candidate_id":1,"score":1}],"factors":[{"action_type":"destroy_silo","factor":1}]}`,
		"not json":           `I cannot help with that.`,
	} {
		if _, err := ParseResponse(raw, 2); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func TestParseResponseRepair(t *testing.T) {
	// qwen3.x non-thinking quirk: the final closing brace is dropped.
	raw := `{"scores":[{"candidate_id":0,"score":3},{"candidate_id":1,"score":7}],"factors":[{"action_type":"gather_info","factor":1.5}]`
	resp, err := ParseResponse(raw, 2)
	if err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if !resp.Repaired || len(resp.Scores) != 2 || len(resp.Factors) != 1 {
		t.Fatalf("bad repaired response: %+v", resp)
	}
	// Truncation mid-scores: salvage keeps the complete prefix, but the
	// candidate-coverage check still rejects it — repair never bypasses the
	// schema gate.
	raw2 := `{"scores":[{"candidate_id":0,"score":3},{"candi`
	if _, err := ParseResponse(raw2, 2); err == nil {
		t.Fatalf("expected validation failure for truncated scores")
	}
	// Complete replies are not marked repaired.
	resp3, err := ParseResponse(raw+"}", 2)
	if err != nil || resp3.Repaired {
		t.Fatalf("complete reply misclassified: %v %+v", err, resp3)
	}
}

func TestScoreToMultiplierMapping(t *testing.T) {
	adv := ToAdvice(&RawResponse{Scores: []RawScore{
		{CandidateID: 0, Score: 0}, {CandidateID: 1, Score: 5}, {CandidateID: 2, Score: 10},
	}})
	want := []float64{0.5, 1.25, 2.0}
	for i, w := range want {
		if got := adv.Scores[i]; got != w {
			t.Fatalf("score mapping %d: got %v want %v", i, got, w)
		}
	}
}

func TestFakeClientDeterminism(t *testing.T) {
	fc := &FakeClient{}
	user := `{"candidates":[{"candidate_id":0,"type":"GATHER_INFO"},{"candidate_id":1,"type":"PROFESSION_ACTION"}]}`
	a, _, err := fc.Complete(nil, "", user)
	if err != nil {
		t.Fatalf("fake complete: %v", err)
	}
	b, _, err := fc.Complete(nil, "", user)
	if err != nil || a != b {
		t.Fatalf("fake client not deterministic: %v / %q vs %q", err, a, b)
	}
	if _, err := ParseResponse(a, 2); err != nil {
		t.Fatalf("fake client output invalid: %v", err)
	}
}

func newAdvisorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := repository.InitDB(filepath.Join(t.TempDir(), "llm-test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	return db
}

func testSiloAndView(t *testing.T) (*model.Silo, *engine.ActorView, *model.Agent) {
	t.Helper()
	engine.SetRNG(nil) // keep whatever source; determinism irrelevant here
	silo := engine.CreateInitialSilo("LlmTest", 122, nil)
	agent := engine.CreateInitialAgent("Agent", "IT", nil, silo)
	view, err := engine.CreateActorView(engine.CreateActorRefForProfession(&silo.Professions[6]), silo, nil)
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	return silo, view, agent
}

func testCandidates() []engine.CandidateInfo {
	return []engine.CandidateInfo{
		{Index: 0, Type: model.ActionGatherInfo, TargetDept: "Mayor", APCost: 10, BaseWeight: 2},
		{Index: 1, Type: model.ActionBuildConnection, TargetDept: "Supply", APCost: 15, BaseWeight: 1},
		{Index: 2, Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL", APCost: 15, BaseWeight: 0.8},
	}
}

func TestAdvisorLiveRecordAndReplay(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeOn, Level: LevelFull})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()

	silo, view, agent := testSiloAndView(t)
	a.ObserveMonth(silo, agent, nil) // arms the full gate for month 1

	cands := testCandidates()
	advice := a.Advise(view, silo, cands)
	if advice == nil || advice.Force != nil {
		t.Fatalf("expected live advice without Force, got %+v", advice)
	}
	chosen := 1
	a.Committed(view, silo, cands, advice, chosen)
	if err := a.Error(); err != nil {
		t.Fatalf("advisor error: %v", err)
	}

	entries, err := LoadEntries(db)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d (%v)", len(entries), err)
	}
	if entries[0].Source != SourceLLM || entries[0].FinalChoice != chosen ||
		entries[0].ScoresJSON == "" || entries[0].Model == "" {
		t.Fatalf("bad entry: %+v", entries[0])
	}

	// Replay on a fresh advisor: the same point must be forced, and the
	// commit must consume the entry without appending a new one.
	b, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeOn, Level: LevelFull})
	if err != nil {
		t.Fatalf("advisor b: %v", err)
	}
	defer b.Close()
	b.ResumeFrom(entries)
	b.ObserveMonth(silo, agent, nil)

	replayAdvice := b.Advise(view, silo, cands)
	if replayAdvice == nil || replayAdvice.Force == nil || *replayAdvice.Force != chosen {
		t.Fatalf("replay did not force recorded choice: %+v", replayAdvice)
	}
	b.Committed(view, silo, cands, replayAdvice, *replayAdvice.Force)
	if err := b.Error(); err != nil {
		t.Fatalf("replay commit error: %v", err)
	}
	entries2, _ := LoadEntries(db)
	if len(entries2) != 1 {
		t.Fatalf("replay appended entries: %d", len(entries2))
	}

	// Divergence must fail explicitly.
	c, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeOn, Level: LevelFull})
	if err != nil {
		t.Fatalf("advisor c: %v", err)
	}
	defer c.Close()
	c.ResumeFrom(entries)
	c.ObserveMonth(silo, agent, nil)
	adv := c.Advise(view, silo, cands)
	if adv == nil || adv.Force == nil {
		t.Fatalf("expected force")
	}
	c.Committed(view, silo, cands, adv, (*adv.Force+1)%len(cands)) // wrong choice
	if c.Error() == nil {
		t.Fatalf("expected explicit replay divergence error")
	}
}

func TestAdvisorShadowRecordsWithoutInfluence(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelFull})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	silo, view, agent := testSiloAndView(t)
	a.ObserveMonth(silo, agent, nil)

	cands := testCandidates()
	if adv := a.Advise(view, silo, cands); adv != nil {
		t.Fatalf("shadow must never advise, got %+v", adv)
	}
	a.Committed(view, silo, cands, nil, 2)
	if err := a.Error(); err != nil {
		t.Fatalf("advisor error: %v", err)
	}

	// The async shadow worker fills the LLM-side fields; poll briefly.
	var entries []DecisionLog
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ = LoadEntries(db)
		if len(entries) == 1 && entries[0].ScoresJSON != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.Close()
	if len(entries) != 1 {
		t.Fatalf("expected 1 shadow entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Source != SourceShadow || e.FinalChoice != 2 || e.ScoresJSON == "" ||
		e.Validation != "ok" || e.ShadowChoice == nil {
		t.Fatalf("bad shadow entry: %+v", e)
	}
}

func TestAdvisorLevelSwitchIsLogged(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelOff})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	if err := a.SetLevel(LevelKey, "api"); err != nil {
		t.Fatalf("set level: %v", err)
	}
	if a.GetLevel() != LevelKey {
		t.Fatalf("level not applied")
	}
	entries, _ := LoadEntries(db)
	if len(entries) != 1 || entries[0].Kind != KindConfig ||
		entries[0].ConfigLevel != "key" || entries[0].ConfigSource != "api" {
		t.Fatalf("config entry missing/bad: %+v", entries)
	}
}

func TestAdvisorGatingByLevel(t *testing.T) {
	db := newAdvisorTestDB(t)
	// events level: no gate without an ONCE fire, gated after one.
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelEvent, EventWindow: 1})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	a.SetEventFilter(func(category, _ string) bool { return category == "special" })
	silo, _, agent := testSiloAndView(t)

	a.ObserveMonth(silo, agent, nil)
	if a.gatedNow() {
		t.Fatalf("events gate open without any ONCE fire")
	}
	a.ObserveMonth(silo, agent, []model.StoryEventLog{{Category: "special", EventID: "outside_signal"}})
	if !a.gatedNow() {
		t.Fatalf("events gate closed right after ONCE fire")
	}

	// key level: rebellion band opens the gate.
	b, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeShadow, Level: LevelKey})
	if err != nil || b == nil {
		t.Fatalf("advisor b: %v", err)
	}
	defer b.Close()
	hot := *silo
	hot.Rebellion = 0.8
	b.ObserveMonth(&hot, agent, nil)
	if !b.gatedNow() {
		t.Fatalf("key gate closed despite rebellion>0.7")
	}
}

// gatedNow exposes the current gate for tests.
func (a *Advisor) gatedNow() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.gateReason != ""
}

// capturingClient records the system prompt of every Complete call so tests
// can assert the shadow path does not drop it.
type capturingClient struct {
	FakeClient
	mu      sync.Mutex
	systems []string
}

func (c *capturingClient) Complete(ctx context.Context, system, user string) (string, Usage, error) {
	c.mu.Lock()
	c.systems = append(c.systems, system)
	c.mu.Unlock()
	return c.FakeClient.Complete(ctx, system, user)
}

func (c *capturingClient) lastSystem() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.systems) == 0 {
		return ""
	}
	return c.systems[len(c.systems)-1]
}

// Regression: the shadow worker must forward the system prompt (it carries
// the task + output schema). Dropping it made a real model emit 94%
// schema_invalid responses in a gate run.
func TestShadowForwardsSystemPrompt(t *testing.T) {
	db := newAdvisorTestDB(t)
	cap := &capturingClient{}
	a, err := NewAdvisor(Config{DB: db, Client: cap, Mode: ModeShadow, Level: LevelFull})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	silo, view, agent := testSiloAndView(t)
	a.ObserveMonth(silo, agent, nil)

	cands := testCandidates()
	a.Committed(view, silo, cands, nil, 1)

	// Wait for the async worker to issue the call.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && cap.lastSystem() == "" {
		time.Sleep(20 * time.Millisecond)
	}
	a.Close()

	sys := cap.lastSystem()
	if sys == "" {
		t.Fatalf("shadow call sent an empty system prompt")
	}
	if !strings.Contains(sys, "scores") || !strings.Contains(sys, "JSON") {
		t.Fatalf("system prompt lacks schema instructions: %q", sys)
	}
	// Alignment instructions: anchor around base_weight, no type labeling.
	for _, want := range []string{"anchor_score", "base_weight", "action type"} {
		if !strings.Contains(sys, want) {
			t.Fatalf("system prompt lacks alignment instruction %q", want)
		}
	}
}

// TestPrefetchHitMissAndAttribution covers the mode=on prefetch path:
// a cached prediction is applied without any synchronous call, attributed as
// llm_prefetch; an unpredicted candidate set misses and falls back to the
// original weights (audit: prefetch_miss).
func TestPrefetchHitMissAndAttribution(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeOn, Level: LevelFull, Prefetch: true})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	silo, view, agent := testSiloAndView(t)
	a.ObserveMonth(silo, agent, nil) // arms the gate and queues prefetches
	silo.CurrentMonth++              // decisions happen the month after observation

	predicted := engine.PredictCandidates(view, silo)
	if len(predicted) == 0 {
		t.Fatalf("expected predicted candidates")
	}
	var advice *engine.NpcAdvice
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if advice = a.Advise(view, silo, predicted); advice != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if advice == nil {
		t.Fatalf("prefetch cache never produced a hit")
	}
	a.Committed(view, silo, predicted, advice, 0)

	odd := []engine.CandidateInfo{{Index: 0, Type: model.ActionInciteRebellion, APCost: 30, BaseWeight: 0.8}}
	if adv := a.Advise(view, silo, odd); adv != nil {
		t.Fatalf("prefetch miss must fall back to original weights, got %+v", adv)
	}
	a.Committed(view, silo, odd, nil, 0)
	a.Close()

	entries, _ := LoadEntries(db)
	var hit, miss *DecisionLog
	for i := range entries {
		e := &entries[i]
		if e.Kind != KindDecision {
			continue
		}
		if e.Source == SourceLLMPrefetch {
			hit = e
		}
		if e.Validation == "prefetch_miss" {
			miss = e
		}
	}
	if hit == nil || hit.Validation == "" || hit.ScoresJSON == "" {
		t.Fatalf("prefetch hit not attributed as llm_prefetch: %+v", hit)
	}
	if miss == nil || miss.Source != SourceOriginal {
		t.Fatalf("prefetch miss not logged as original fallback: %+v", miss)
	}
}

func TestWarmupRecordsOutcome(t *testing.T) {
	db := newAdvisorTestDB(t)
	a, err := NewAdvisor(Config{DB: db, Client: &FakeClient{}, Mode: ModeOn, Level: LevelFull, Warmup: true})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if done, _ := a.WarmupStatus(); done {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	done, werr := a.WarmupStatus()
	if !done || werr != nil {
		t.Fatalf("warmup not recorded: done=%v err=%v", done, werr)
	}
}

// rawClient returns a fixed raw reply regardless of the prompt.
type rawClient struct{ raw string }

func (r *rawClient) Complete(_ context.Context, _, _ string) (string, Usage, error) {
	return r.raw, Usage{}, nil
}
func (r *rawClient) ModelID() string { return "raw" }

// Regression: repaired (missing trailing brace) advice is applied by Advise,
// so the log must attribute it to the LLM, not to the original algorithm.
func TestModeOnRepairedAdviceAttributedToLLM(t *testing.T) {
	db := newAdvisorTestDB(t)
	raw := `{"scores":[{"candidate_id":0,"score":3},{"candidate_id":1,"score":7}]`
	a, err := NewAdvisor(Config{DB: db, Client: &rawClient{raw: raw}, Mode: ModeOn, Level: LevelFull})
	if err != nil || a == nil {
		t.Fatalf("advisor: %v", err)
	}
	defer a.Close()
	silo, view, agent := testSiloAndView(t)
	a.ObserveMonth(silo, agent, nil)

	cands := testCandidates()[:2]
	advice := a.Advise(view, silo, cands)
	if advice == nil {
		t.Fatalf("repaired response must produce advice")
	}
	a.Committed(view, silo, cands, advice, 1)
	entries, _ := LoadEntries(db)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Source != SourceLLM || entries[0].Validation != "ok_repaired" {
		t.Fatalf("repaired advice misattributed: source=%q validation=%q",
			entries[0].Source, entries[0].Validation)
	}
}

func TestCandidateAnchors(t *testing.T) {
	infos := []engine.CandidateInfo{
		{Index: 0, Type: model.ActionGatherInfo, APCost: 10, BaseWeight: 3.0},
		{Index: 1, Type: model.ActionBuildConnection, APCost: 15, BaseWeight: 1.0},
		{Index: 2, Type: model.ActionProfession, ProfessionAction: "X", APCost: 15, BaseWeight: 2.0},
	}
	out := ContextBuilder{}.BuildCandidateList(infos)
	if len(out) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(out))
	}
	// Anchors spread over [2,8] preserving the weight order.
	want := []float64{8, 2, 5}
	for i, cc := range out {
		if cc.BaseWeight != infos[i].BaseWeight {
			t.Fatalf("candidate %d base_weight: got %v want %v", i, cc.BaseWeight, infos[i].BaseWeight)
		}
		if cc.AnchorScore != want[i] {
			t.Fatalf("candidate %d anchor: got %v want %v", i, cc.AnchorScore, want[i])
		}
	}
	// Equal weights map to the neutral middle.
	equal := ContextBuilder{}.BuildCandidateList([]engine.CandidateInfo{
		{Index: 0, Type: model.ActionGatherInfo, BaseWeight: 1.2},
		{Index: 1, Type: model.ActionBuildConnection, BaseWeight: 1.2},
	})
	for _, cc := range equal {
		if cc.AnchorScore != 5 {
			t.Fatalf("equal weights should anchor at 5, got %v", cc.AnchorScore)
		}
	}
}

package service

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"silo40/internal/llm"
	"silo40/internal/model"
)

// promptCaptureClient records every user prompt the advisor sends, so tests
// can assert what each context level actually ships to the model.
type promptCaptureClient struct {
	llm.FakeClient
	mu    sync.Mutex
	users []string
}

func (c *promptCaptureClient) Complete(ctx context.Context, system, user string) (string, llm.Usage, error) {
	c.mu.Lock()
	c.users = append(c.users, user)
	c.mu.Unlock()
	return c.FakeClient.Complete(ctx, system, user)
}

func (c *promptCaptureClient) promptUsers() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.users...)
}

// TestSetLlmContextMidGameIsLoggedAndRestored locks the switch contract:
// changing the context config mid-game appends a context_config entry, takes
// effect immediately, and survives resume (participation level untouched).
func TestSetLlmContextMidGameIsLoggedAndRestored(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ctxswitch.db")
	svc := newLlmTestService(t, dbPath, llm.ModeShadow, llm.LevelOff, &llm.FakeClient{})
	createLlmGame(t, svc, 5)

	if _, err := svc.SetLlmContext(LlmContextConfig{Level: "bogus"}); err == nil {
		t.Fatalf("invalid context level must be rejected")
	}

	got, err := svc.SetLlmContext(LlmContextConfig{
		Level:             "rich",
		PersonalMemory:    false,
		ProtocolClauses:   true,
		PeerSiloIntel:     true,
		EventSummaryLimit: 9,
	})
	if err != nil {
		t.Fatalf("set context: %v", err)
	}
	if got.Level != "rich" || got.PersonalMemory || !got.ProtocolClauses ||
		!got.PeerSiloIntel || got.EventSummaryLimit != 9 {
		t.Fatalf("applied config mismatch: %+v", got)
	}
	// Participation stays orthogonal.
	if svc.llmAdvisor.GetLevel() != llm.LevelOff {
		t.Fatalf("context switch must not touch participation: %q", svc.llmAdvisor.GetLevel())
	}

	entries, err := llm.LoadEntries(svc.db)
	if err != nil {
		t.Fatalf("load log: %v", err)
	}
	var ctxEntries []llm.DecisionLog
	for _, e := range entries {
		if e.Kind == llm.KindContextConfig {
			ctxEntries = append(ctxEntries, e)
		}
	}
	if len(ctxEntries) != 2 || ctxEntries[0].ConfigSource != "create" || ctxEntries[1].ConfigSource != "api" {
		t.Fatalf("context config entries wrong: %+v", ctxEntries)
	}
	if !strings.Contains(ctxEntries[1].ConfigContext, `"level":"rich"`) ||
		!strings.Contains(ctxEntries[1].ConfigContext, `"event_limit":9`) {
		t.Fatalf("context payload missing fields: %s", ctxEntries[1].ConfigContext)
	}

	// Resume restores the last recorded context config.
	svc2 := newLlmTestService(t, dbPath, llm.ModeShadow, llm.LevelOff, &llm.FakeClient{})
	if err := svc2.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if svc2.llmAdvisor.GetContextLevel() != llm.ContextRich {
		t.Fatalf("resume lost the context level: %q", svc2.llmAdvisor.GetContextLevel())
	}
	opts := svc2.llmAdvisor.GetContextOptions()
	if opts.PersonalMemory || !opts.ProtocolClauses || !opts.PeerSiloIntel || opts.EventSummaryLimit != 9 {
		t.Fatalf("resume lost the context switches: %+v", opts)
	}

	// Status snapshot carries the config for the UI.
	st := svc2.GetLlmStatus()
	if !st.Enabled || st.Context.Level != "rich" || st.Context.EventSummaryLimit != 9 {
		t.Fatalf("status context mismatch: %+v", st)
	}
}

// TestSetLlmContextRequiresAdvisor: with LLM disabled the switch fails
// explicitly instead of pretending to apply.
func TestSetLlmContextRequiresAdvisor(t *testing.T) {
	svc := newLlmTestService(t, filepath.Join(t.TempDir(), "ctx-off.db"), llm.ModeOff, llm.LevelOff, &llm.FakeClient{})
	if _, err := svc.SetLlmContext(LlmContextConfig{Level: "minimal"}); err == nil {
		t.Fatalf("SetLlmContext must fail when LLM is disabled")
	}
}

// TestLlmContextProviderFeedsPrompt drives real ticks with the service-side
// provider wired and asserts the world level + switches actually reach the
// model payload (sections present, not just declared).
func TestLlmContextProviderFeedsPrompt(t *testing.T) {
	cap := &promptCaptureClient{}
	svc := newLlmTestService(t, filepath.Join(t.TempDir(), "ctxfeed.db"), llm.ModeOn, llm.LevelFull, cap)
	svc.llmAdvisor.SetContextProvider(svc.llmContextData)

	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content dirs: %v", err)
	}
	if err := svc.loadProtocolClauses(dirs); err != nil {
		t.Fatalf("load protocol clauses: %v", err)
	}

	opts := llm.ContextOptions{PersonalMemory: true, ProtocolClauses: true, EventSummaryLimit: 5, PeerSiloIntel: true}
	if err := svc.llmAdvisor.SetContext(llm.ContextWorld, opts, "test"); err != nil {
		t.Fatalf("set context: %v", err)
	}
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:     "CtxStation",
		AgentName:    "CtxAgent",
		Profession:   "IT",
		Seed:         42,
		IncludeSilo1: true,
		Neighbor:     &model.NeighborSiloConfig{},
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("month %d: %v", i+1, err)
		}
	}
	if err := svc.llmError(); err != nil {
		t.Fatalf("advisor error: %v", err)
	}

	users := cap.promptUsers()
	if len(users) == 0 {
		t.Fatalf("no LLM calls captured")
	}
	joined := strings.Join(users, "\n")
	for _, key := range []string{`"world_setting"`, `"peer_silos"`, `"protocol_clauses"`, `"peer_intel"`} {
		if !strings.Contains(joined, key) {
			t.Fatalf("world-level prompt never carried %s", key)
		}
	}
	// 其他地堡情报 must stay rumor-grade (status + source), never exact values.
	if !strings.Contains(joined, "消息来源") {
		t.Fatalf("peer intel section lacks rumor-grade phrasing")
	}
}

// TestMinimalContextSkipsProvider locks the cost contract: at the minimal
// level the provider is never called, so cheap prompts stay cheap.
func TestMinimalContextSkipsProvider(t *testing.T) {
	svc := newLlmTestService(t, filepath.Join(t.TempDir(), "ctxmin.db"), llm.ModeOn, llm.LevelFull, &llm.FakeClient{})
	calls := 0
	svc.llmAdvisor.SetContextProvider(func(uint) llm.ContextData {
		calls++
		return llm.ContextData{}
	})
	opts := llm.DefaultContextOptions()
	if err := svc.llmAdvisor.SetContext(llm.ContextMinimal, opts, "test"); err != nil {
		t.Fatalf("set context: %v", err)
	}
	createLlmGame(t, svc, 9)
	for i := 0; i < 2; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("month %d: %v", i+1, err)
		}
	}
	if calls != 0 {
		t.Fatalf("minimal level must not call the context provider, got %d calls", calls)
	}
}

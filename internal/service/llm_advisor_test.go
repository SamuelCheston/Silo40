package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"silo40/internal/llm"
	"silo40/internal/model"
	"silo40/internal/repository"
)

// nondetClient returns random (but schema-valid) scores: used to prove
// that replay does not depend on the model at all.
type nondetClient struct {
	r    *rand.Rand
	name string
}

func (c *nondetClient) Complete(_ context.Context, _, user string) (string, llm.Usage, error) {
	var payload struct {
		Candidates []struct {
			CandidateID int `json:"candidate_id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		return "", llm.Usage{}, err
	}
	type item struct {
		CandidateID int     `json:"candidate_id"`
		Score       float64 `json:"score"`
	}
	out := struct {
		Scores []item `json:"scores"`
	}{}
	for _, cand := range payload.Candidates {
		out.Scores = append(out.Scores, item{CandidateID: cand.CandidateID, Score: c.r.Float64() * 10})
	}
	data, err := json.Marshal(out)
	return string(data), llm.Usage{}, err
}

func (c *nondetClient) ModelID() string { return "nondet#" + c.name }

// newLlmTestService builds a full-content service with an explicitly injected
// advisor (env vars stay untouched so parallel tests are unaffected).
func newLlmTestService(t *testing.T, dbPath string, mode llm.Mode, level llm.Level, client llm.Client) *GameService {
	t.Helper()
	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := NewGameService(db)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content dirs: %v", err)
	}
	if err := svc.syncContentDefinitions(dirs); err != nil {
		t.Fatalf("sync content: %v", err)
	}
	if err := svc.loadContentDefinitions(); err != nil {
		t.Fatalf("load content: %v", err)
	}
	svc.bindContentEventHandlers()
	if mode != llm.ModeOff {
		advisor, err := llm.NewAdvisor(llm.Config{DB: db, Client: client, Mode: mode, Level: level})
		if err != nil || advisor == nil {
			t.Fatalf("advisor: %v", err)
		}
		t.Cleanup(advisor.Close)
		svc.llmAdvisor = advisor
		svc.engine.NpcPolicy = advisor
		svc.bindLlmEventFilter()
	}
	return svc
}

func createLlmGame(t *testing.T, svc *GameService, seed int64) {
	t.Helper()
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   fmt.Sprintf("Llm-%d", seed),
		AgentName:  "LlmAgent",
		Profession: "IT",
		Seed:       seed,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
}

// TestShadowModeLeavesTrajectoryUntouched locks the shadow contract: with
// the LLM recording but never acting, the trajectory must stay bit-for-bit
// identical to a plain run, while the log fills with shadow entries.
func TestShadowModeLeavesTrajectoryUntouched(t *testing.T) {
	run := func(mode llm.Mode) (reproSnapshot, *GameService) {
		svc := newLlmTestService(t, filepath.Join(t.TempDir(), "shadow.db"), mode, llm.LevelFull, &llm.FakeClient{})
		createLlmGame(t, svc, 42)
		for i := 0; i < 24; i++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("pass time month %d: %v", i+1, err)
			}
		}
		return snapshot(svc), svc
	}

	base, _ := run(llm.ModeOff)
	shadow, svc := run(llm.ModeShadow)
	if !reflect.DeepEqual(base, shadow) {
		t.Fatalf("shadow mode changed the trajectory:\n base=%+v\n shadow=%+v", base, shadow)
	}

	entries, err := llm.LoadEntries(svc.db)
	if err != nil {
		t.Fatalf("load log: %v", err)
	}
	shadowCount := 0
	for _, e := range entries {
		if e.Kind == llm.KindDecision && e.Source == llm.SourceShadow {
			shadowCount++
		}
	}
	if shadowCount == 0 {
		t.Fatalf("no shadow decision entries recorded (log=%d entries)", len(entries))
	}
	// Async shadow evaluations must land; poll until scores appear.
	deadline := time.Now().Add(3 * time.Second)
	filled := 0
	for time.Now().Before(deadline) {
		entries, _ = llm.LoadEntries(svc.db)
		filled = 0
		for _, e := range entries {
			if e.Source == llm.SourceShadow && e.ScoresJSON != "" && e.ShadowChoice != nil {
				filled++
			}
		}
		if filled == shadowCount {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if filled != shadowCount {
		t.Fatalf("shadow evaluations incomplete: %d/%d filled", filled, shadowCount)
	}
}

// TestReplayRestoresLLMDecisions locks the decision-log contract end to
// end: months played with a non-deterministic model reproduce exactly when
// the same save is resumed and replayed from the log.
func TestReplayRestoresLLMDecisions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "replay.db")

	// Phase 1: deterministic fake client plays months 1-12 (mode=on, full).
	svcA := newLlmTestService(t, dbPath, llm.ModeOn, llm.LevelFull, &llm.FakeClient{})
	createLlmGame(t, svcA, 7)
	for i := 0; i < 12; i++ {
		if _, err := svcA.PassTime(1); err != nil {
			t.Fatalf("phase1 month %d: %v", i+1, err)
		}
	}
	if err := svcA.persist(); err != nil {
		t.Fatalf("phase1 persist: %v", err)
	}

	// Phase 2: resume; a NON-deterministic client plays months 13-24 live.
	// State is deliberately not persisted afterwards: the db keeps the month
	// 12 checkpoint while the log grows with phase-2 decisions.
	svcB := newLlmTestService(t, dbPath, llm.ModeOn, llm.LevelFull, &nondetClient{r: rand.New(rand.NewSource(1)), name: "B"})
	if err := svcB.Resume(); err != nil {
		t.Fatalf("phase2 resume: %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := svcB.PassTime(1); err != nil {
			t.Fatalf("phase2 month %d: %v", i+1, err)
		}
	}
	snapB := snapshot(svcB)

	// Phase 3: fresh resume of the SAME checkpoint with a differently-seeded
	// non-deterministic client. Months 13-24 must replay from the log, so
	// the client's randomness cannot change the outcome.
	svcC := newLlmTestService(t, dbPath, llm.ModeOn, llm.LevelFull, &nondetClient{r: rand.New(rand.NewSource(999)), name: "C"})
	if err := svcC.Resume(); err != nil {
		t.Fatalf("phase3 resume: %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := svcC.PassTime(1); err != nil {
			t.Fatalf("phase3 month %d: %v", i+1, err)
		}
	}
	snapC := snapshot(svcC)

	if !reflect.DeepEqual(snapB, snapC) {
		t.Fatalf("replay diverged from live play:\n live=%+v\n replay=%+v", snapB, snapC)
	}

	entries, err := llm.LoadEntries(svcC.db)
	if err != nil {
		t.Fatalf("load log: %v", err)
	}
	llmEntries := 0
	for _, e := range entries {
		if e.Kind == llm.KindDecision && e.Source == llm.SourceLLM {
			llmEntries++
			if e.Model == "" || e.ScoresJSON == "" {
				t.Fatalf("llm entry missing audit fields: %+v", e)
			}
		}
	}
	if llmEntries == 0 {
		t.Fatalf("no LLM decision entries recorded")
	}
}

// TestLevelSwitchMidGameIsLoggedAndRestored locks §11: switching
// participation mid-game appends a config entry, takes effect from the next
// decision point, and survives resume.
func TestLevelSwitchMidGameIsLoggedAndRestored(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "switch.db")
	svc := newLlmTestService(t, dbPath, llm.ModeShadow, llm.LevelOff, &llm.FakeClient{})
	createLlmGame(t, svc, 5)
	for i := 0; i < 6; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("month %d: %v", i+1, err)
		}
	}
	got, err := svc.SetLlmParticipation("full")
	if err != nil || got != "full" {
		t.Fatalf("switch: %v / %q", err, got)
	}
	for i := 0; i < 6; i++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("month %d: %v", i+7, err)
		}
	}

	entries, err := llm.LoadEntries(svc.db)
	if err != nil {
		t.Fatalf("load log: %v", err)
	}
	var configs []llm.DecisionLog
	decisions := 0
	for _, e := range entries {
		if e.Kind == llm.KindConfig {
			configs = append(configs, e)
		}
		if e.Kind == llm.KindDecision {
			decisions++
		}
	}
	if len(configs) != 2 || configs[0].ConfigLevel != "off" || configs[0].ConfigSource != "create" ||
		configs[1].ConfigLevel != "full" || configs[1].ConfigSource != "api" {
		t.Fatalf("config entries wrong: %+v", configs)
	}
	if decisions == 0 {
		t.Fatalf("no decisions recorded after switching to full")
	}

	// Resume restores the last recorded level.
	svc2 := newLlmTestService(t, dbPath, llm.ModeShadow, llm.LevelOff, &llm.FakeClient{})
	if err := svc2.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if svc2.llmAdvisor.GetLevel() != llm.LevelFull {
		t.Fatalf("resume lost the level switch: %v", svc2.llmAdvisor.GetLevel())
	}
}

// TestLlmOffKeepsContractBitForBit sanity-checks that merely constructing
// the advisor path (mode off) cannot perturb the seeded trajectory.
func TestLlmOffKeepsContractBitForBit(t *testing.T) {
	run := func(seed int64) reproSnapshot {
		svc := newSeededService(t, seed)
		for i := 0; i < 18; i++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("pass time month %d: %v", i+1, err)
			}
		}
		return snapshot(svc)
	}
	a, b := run(42), run(42)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("baseline contract broken:\n A=%+v\n B=%+v", a, b)
	}
}

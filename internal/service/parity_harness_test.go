package service

// Parity harness: plays deterministic seeded three-silo games through the full
// service stack (real content, real DB, save/resume round-trip, scripted actions)
// and dumps records as JSON. The v6 baseline replaces the invalidated v5
// identity-model records; later refactors must diff against the new baseline.
//
// Opt-in via SILO40_PARITY_DUMP=<output.json>; skipped otherwise so it never
// runs in normal CI.

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"silo40/internal/model"
	"silo40/internal/repository"
)

type parityRecord struct {
	Seed          int64            `json:"seed"`
	Station       string           `json:"station"`
	SiloIDs       []uint           `json:"silo_ids"`
	SiloNames     []string         `json:"silo_names"`
	Ending        string           `json:"ending"`
	Won           bool             `json:"won"`
	Months        int              `json:"months"`
	Year          int              `json:"year"`
	Month         int              `json:"month"`
	Population    int              `json:"population"`
	Rebellion     float64          `json:"rebellion"`
	Cohesion      float64          `json:"cohesion"`
	Legitimacy    float64          `json:"legitimacy"`
	HistoryBurden float64          `json:"history_burden"`
	SafeguardRisk float64          `json:"safeguard_risk"`
	AgentAP       float64          `json:"agent_ap"`
	Suspicion     float64          `json:"suspicion"`
	Fragments     int              `json:"fragments"`
	Fires         map[string]int   `json:"fires"`
	Resources     map[string]int64 `json:"resources"`
}

// newParityService builds a service bound to a fixed DB path (so Resume can
// be tested against the same file) with the repository's real content.
func newParityService(t *testing.T, dbPath string) *GameService {
	t.Helper()
	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := NewGameService(db)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.syncContentDefinitions(dirs); err != nil {
		t.Fatalf("sync content definitions: %v", err)
	}
	if err := svc.loadContentDefinitions(); err != nil {
		t.Fatalf("load content definitions: %v", err)
	}
	return svc
}

func TestParityHarnessDump(t *testing.T) {
	outPath := os.Getenv("SILO40_PARITY_DUMP")
	if outPath == "" {
		t.Skip("set SILO40_PARITY_DUMP=<file> to dump parity records")
	}
	seeds := []int64{7, 8, 9, 10, 11, 12}
	records := make([]parityRecord, 0, len(seeds))
	for _, seed := range seeds {
		records = append(records, runParityGame(t, seed))
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatalf("write dump: %v", err)
	}
	t.Logf("parity records written to %s", outPath)
}

func runParityGame(t *testing.T, seed int64) parityRecord {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "parity.db")
	svc := newParityService(t, dbPath)

	req := threeSiloRequest(seed)
	req.AgentName = "ParityAgent"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game seed %d: %v", seed, err)
	}

	// Independent policy RNG: must never consume the game's stream.
	policy := rand.New(rand.NewSource(seed ^ 0x5110))

	months := 0
	for m := 1; m <= 200; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("seed %d pass time month %d: %v", seed, m, err)
		}
		months = m

		// Scripted deterministic actions.
		if m%7 == 0 && svc.agent.ActionPoints >= 10 {
			if _, err := svc.ExecuteAction(model.AgentAction{
				Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 0,
			}); err != nil {
				t.Fatalf("seed %d gather month %d: %v", seed, m, err)
			}
		}
		if m%23 == 0 && svc.agent.ActionPoints >= 15 && policy.Float64() < 0.8 {
			if _, err := svc.ExecuteAction(model.AgentAction{
				Type: model.ActionBuildConnection, TargetDept: "Mines", Cost: 0,
			}); err != nil {
				t.Fatalf("seed %d build month %d: %v", seed, m, err)
			}
		}

		// Mid-game save/load round-trip: persist, then resume in a fresh
		// service bound to the same DB.
		if m == 100 {
			if err := svc.persist(); err != nil {
				t.Fatalf("seed %d persist: %v", seed, err)
			}
			fresh := newParityService(t, dbPath)
			if err := fresh.Resume(); err != nil {
				t.Fatalf("seed %d resume: %v", seed, err)
			}
			svc = fresh
		}

		if svc.gameOver() {
			break
		}
	}

	state, err := svc.GetState()
	if err != nil {
		t.Fatalf("seed %d get state: %v", seed, err)
	}
	silo := state.Silo

	rec := parityRecord{
		Seed:          seed,
		Station:       silo.Name,
		SiloIDs:       sortedSiloIDs(svc.silos),
		SiloNames:     make([]string, 0, len(svc.silos)),
		Months:        months,
		Year:          silo.CurrentYear,
		Month:         silo.CurrentMonth,
		Population:    silo.TotalPopulation,
		Rebellion:     silo.Rebellion,
		Cohesion:      silo.Cohesion,
		Legitimacy:    silo.Legitimacy,
		HistoryBurden: silo.HistoryBurden,
		SafeguardRisk: silo.SafeguardRisk,
		AgentAP:       state.Agent.ActionPoints,
		Suspicion:     state.Agent.SuspicionLevel,
		Fragments:     len(state.Agent.KnownFragments),
		Fires:         map[string]int{},
		Resources:     map[string]int64{},
	}
	if silo.VictoryStatus != nil {
		rec.Ending = silo.VictoryStatus.Type
		rec.Won = silo.VictoryStatus.IsWon

	} else {
		rec.Ending = "NO_ENDING(max_months)"
	}
	for _, worldSilo := range svc.silos {
		rec.SiloNames = append(rec.SiloNames, worldSilo.Name)
	}
	for _, entry := range svc.eventLogs {
		key := entry.EventID
		if entry.Category != "" {
			key = entry.Category + ":" + entry.EventID
		}
		rec.Fires[key]++
	}
	for _, res := range silo.Resources {
		rec.Resources[res.Type] = int64(res.Amount)
	}
	return rec
}

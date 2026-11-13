package service

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestAgentYieldMeasurement plays mechanical full games on several seeds and
// reports per-game yield (months, decision points, prompt size) plus writes
// the combined pipeline-ready JSONL for transform verification.
//
// Run explicitly: go test ./internal/service/ -run TestAgentYieldMeasurement -v
func TestAgentYieldMeasurement(t *testing.T) {
	if os.Getenv("SILO40_AGENT_YIELD") == "" {
		t.Skip("set SILO40_AGENT_YIELD=1 to run the yield measurement")
	}

	outPath := os.Getenv("SILO40_AGENT_YIELD_OUT")
	if outPath == "" {
		outPath = "/tmp/agent_yield.jsonl"
	}

	var all strings.Builder
	seeds := []int64{42, 43, 44}
	for _, seed := range seeds {
		svc := bootstrapRealContent(t)
		if _, err := svc.StartAgentGame(AgentStartRequest{
			Seed: seed, SiloName: fmt.Sprintf("Yield-%d", seed),
			AgentName: "Mech", Profession: "IT",
		}); err != nil {
			t.Fatalf("start seed %d: %v", seed, err)
		}

		decisions := 0
		months := 0
		gameOver := false
		for !gameOver && months < 1500 {
			out, err := svc.AgentStep()
			if err != nil {
				t.Fatalf("seed %d step: %v", seed, err)
			}
			for {
				m := out.(map[string]any)
				switch m["status"] {
				case "decision":
					dec := m["decision"].(map[string]any)
					out, err = svc.AgentChoose(mechanicalChoose(t, dec))
					if err != nil {
						t.Fatalf("seed %d choose: %v", seed, err)
					}
					decisions++
					continue
				case "month_end":
					months++
				case "game_over":
					months++
					gameOver = true
				}
				break
			}
		}

		export, err := svc.AgentExport()
		if err != nil {
			t.Fatalf("seed %d export: %v", seed, err)
		}
		exp := export.(map[string]any)
		records := exp["records"].([]agentRecord)

		var promptBytes, candTotal, executed int
		for _, r := range records {
			promptBytes += len(r.Prompt)
			candTotal += len(r.Candidates)
			if r.ExecutionStatus == agentStatusExecuted {
				executed++
			}
		}
		avgCand := 0.0
		avgPrompt := 0
		if len(records) > 0 {
			avgCand = float64(candTotal) / float64(len(records))
			avgPrompt = promptBytes / len(records)
		}
		t.Logf("seed=%d months=%d decisions=%d executed=%d game_over=%v avg_candidates=%.2f avg_prompt_bytes=%d total_prompt_bytes=%d (~%d tokens)",
			seed, months, len(records), executed, gameOver, avgCand, avgPrompt, promptBytes, promptBytes/4)

		jsonl, err := svc.AgentExportJSONL()
		if err != nil {
			t.Fatalf("seed %d jsonl: %v", seed, err)
		}
		all.WriteString(jsonl)
	}

	if err := os.WriteFile(outPath, []byte(all.String()), 0o644); err != nil {
		t.Fatalf("write yield jsonl: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(all.String()), "\n") + 1
	t.Logf("combined JSONL: %d records -> %s", lines, outPath)
}

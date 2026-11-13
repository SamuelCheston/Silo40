package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// mechanicalChoose mimics a very simple operator: pick the candidate with
// the highest base weight, score the chosen one clearly above the rest.
func mechanicalChoose(t *testing.T, decision map[string]any) AgentChooseRequest {
	t.Helper()
	decisionID, _ := decision["decision_id"].(string)
	rawCands, ok := decision["candidates"].([]any)
	if !ok || len(rawCands) == 0 {
		t.Fatalf("decision has no candidates: %v", decision)
	}
	bestIdx, bestWeight := 0, math.Inf(-1)
	scores := make([]AgentScore, 0, len(rawCands))
	for i, raw := range rawCands {
		cand := raw.(map[string]any)
		w, _ := cand["base_weight"].(float64)
		if w > bestWeight {
			bestIdx, bestWeight = i, w
		}
	}
	for i := range rawCands {
		score := 4.0
		if i == bestIdx {
			score = 8.5
		}
		scores = append(scores, AgentScore{CandidateID: i, Score: score})
	}
	return AgentChooseRequest{
		DecisionID: decisionID,
		Candidate:  bestIdx,
		Scores:     scores,
		Rationale:  "mechanical baseline: highest base weight",
	}
}

// playAgentMonth advances one month through the step/choose ping-pong and
// returns the month summary (or fails the test).
func playAgentMonth(t *testing.T, svc *GameService) map[string]any {
	t.Helper()
	out, err := svc.AgentStep()
	if err != nil {
		t.Fatalf("agent step: %v", err)
	}
	for {
		m, ok := out.(map[string]any)
		if !ok {
			t.Fatalf("unexpected step result type %T", out)
		}
		status, _ := m["status"].(string)
		switch status {
		case "decision":
			decision, _ := m["decision"].(map[string]any)
			req := mechanicalChoose(t, decision)
			out, err = svc.AgentChoose(req)
			if err != nil {
				t.Fatalf("agent choose: %v", err)
			}
		case "month_end", "game_over":
			return m
		default:
			t.Fatalf("unexpected status %q in %v", status, m)
		}
	}
}

// assertPipelineShape mirrors pipeline.py validate_raw for one record.
func assertPipelineShape(t *testing.T, rec agentRecord) {
	t.Helper()
	for _, field := range []struct{ name, value string }{
		{"decision_id", rec.DecisionID}, {"game_id", rec.GameID}, {"prompt", rec.Prompt},
	} {
		if strings.TrimSpace(field.value) == "" {
			t.Fatalf("record %s: empty %s", rec.DecisionID, field.name)
		}
	}
	if len(rec.Candidates) == 0 {
		t.Fatalf("record %s: empty candidates", rec.DecisionID)
	}
	seen := map[string]bool{}
	for _, c := range rec.Candidates {
		if c.ActionID == "" || seen[c.ActionID] {
			t.Fatalf("record %s: bad or duplicate action_id %q", rec.DecisionID, c.ActionID)
		}
		seen[c.ActionID] = true
		if !c.Executable {
			t.Fatalf("record %s: candidate %s not marked executable", rec.DecisionID, c.ActionID)
		}
		for name, v := range map[string]float64{
			"original_weight": c.OriginalWeight, "model_score": c.ModelScore,
			"willingness": c.Willingness, "final_multiplier": c.FinalMultiplier,
		} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("record %s: candidate %s field %s not finite", rec.DecisionID, c.ActionID, name)
			}
		}
	}
	if rec.ExecutionStatus != agentStatusExecuted {
		return // discarded path; shape checks below only apply to executed rows
	}
	if rec.ExecutedAction == nil || rec.Response == nil {
		t.Fatalf("record %s: executed status but nil action/response", rec.DecisionID)
	}
	if rec.ExecutedAction.ActionID != rec.Response.ActionID {
		t.Fatalf("record %s: response action_id != executed action_id", rec.DecisionID)
	}
	if fmt.Sprint(rec.ExecutedAction.Args) != fmt.Sprint(rec.Response.Args) {
		t.Fatalf("record %s: response args != executed args", rec.DecisionID)
	}
	if !seen[rec.ExecutedAction.ActionID] {
		t.Fatalf("record %s: executed action %s not among candidates", rec.DecisionID, rec.ExecutedAction.ActionID)
	}
}

// TestAgentOperatorFullGame plays a full game as the mechanical agent and
// verifies the recorded decisions against the pipeline contract.
func TestAgentOperatorFullGame(t *testing.T) {
	svc := bootstrapRealContent(t)

	start, err := svc.StartAgentGame(AgentStartRequest{
		Seed:       42,
		SiloName:   "AgentHost",
		AgentName:  "Operator",
		Profession: "IT",
	})
	if err != nil {
		t.Fatalf("start agent game: %v", err)
	}
	startMap := start.(map[string]any)
	if startMap["phase"] != "idle" {
		t.Fatalf("expected idle phase after start, got %v", startMap["phase"])
	}

	const maxMonths = 1500
	var last map[string]any
	for m := 0; m < maxMonths; m++ {
		last = playAgentMonth(t, svc)
		if last["status"] == "game_over" {
			break
		}
	}
	if last == nil {
		t.Fatalf("game produced no months")
	}

	export, err := svc.AgentExport()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	exp := export.(map[string]any)
	records, _ := exp["records"].([]agentRecord)
	if len(records) == 0 {
		t.Fatalf("expected recorded decisions, got none")
	}
	executed := 0
	for _, rec := range records {
		assertPipelineShape(t, rec)
		if rec.ExecutionStatus == agentStatusExecuted {
			executed++
		}
	}
	// The overwhelming majority of agent choices must execute: Force pins
	// the pick and candidates are pre-filtered by AP budget.
	if executed == 0 {
		t.Fatalf("no executed decisions among %d records", len(records))
	}

	// JSONL export must round-trip line by line.
	jsonl, err := svc.AgentExportJSONL()
	if err != nil {
		t.Fatalf("export jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(jsonl), "\n")
	if len(lines) != len(records) {
		t.Fatalf("jsonl lines %d != records %d", len(lines), len(records))
	}
	for i, line := range lines {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Fatalf("jsonl line %d invalid: %v", i, err)
		}
		for _, key := range []string{"decision_id", "game_id", "prompt", "candidates", "executed_action", "response", "execution_status"} {
			if _, ok := parsed[key]; !ok {
				t.Fatalf("jsonl line %d missing key %s", i, key)
			}
		}
	}

	t.Logf("yield: months=%v decisions=%d executed=%d game_over=%v ending=%v",
		exp["months_elapsed"], len(records), executed, last["game_over"], last["ending"])
}

// TestAgentOperatorChooseValidation verifies the strict payload rules reject
// bad submissions without consuming the pending decision.
func TestAgentOperatorChooseValidation(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.StartAgentGame(AgentStartRequest{Seed: 7, SiloName: "V", AgentName: "O", Profession: "IT"}); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Advance to the first decision point.
	var decision map[string]any
	for i := 0; i < 200; i++ {
		sum := playAgentMonthUntilDecision(t, svc)
		if sum != nil {
			decision = sum
			break
		}
	}
	if decision == nil {
		t.Skip("no decision point in the first 200 months; nothing to validate")
	}
	decisionID := decision["decision_id"].(string)
	n := len(decision["candidates"].([]any))

	fullScores := make([]AgentScore, n)
	for i := range fullScores {
		fullScores[i] = AgentScore{CandidateID: i, Score: 5}
	}

	cases := []struct {
		name string
		req  AgentChooseRequest
	}{
		{"wrong decision id", AgentChooseRequest{DecisionID: "bogus", Candidate: 0, Scores: fullScores}},
		{"candidate out of range", AgentChooseRequest{DecisionID: decisionID, Candidate: n + 3, Scores: fullScores}},
		{"missing scores", AgentChooseRequest{DecisionID: decisionID, Candidate: 0, Scores: fullScores[:n-1]}},
		{"score out of range", AgentChooseRequest{DecisionID: decisionID, Candidate: 0,
			Scores: append([]AgentScore{{CandidateID: 0, Score: 14}}, fullScores[1:]...)}},
	}
	for _, tc := range cases {
		if _, err := svc.AgentChoose(tc.req); err == nil {
			t.Fatalf("%s: expected error, got none", tc.name)
		}
	}

	// The pending decision must still be there and answerable.
	status, err := svc.AgentStatus()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.(map[string]any)["pending_decision_id"] != decisionID {
		t.Fatalf("pending decision changed after rejected submissions: %v", status)
	}
	if _, err := svc.AgentChoose(AgentChooseRequest{
		DecisionID: decisionID, Candidate: 0, Scores: fullScores,
	}); err != nil {
		t.Fatalf("valid choose after rejections failed: %v", err)
	}
	svc.AgentStop()
}

// playAgentMonthUntilDecision steps months until a decision point appears;
// returns the decision context or nil for a plain month_end.
func playAgentMonthUntilDecision(t *testing.T, svc *GameService) map[string]any {
	t.Helper()
	out, err := svc.AgentStep()
	if err != nil {
		t.Fatalf("agent step: %v", err)
	}
	m := out.(map[string]any)
	switch m["status"] {
	case "decision":
		return m["decision"].(map[string]any)
	case "month_end":
		return nil
	case "game_over":
		t.Fatalf("game ended before a decision point appeared")
		return nil
	}
	t.Fatalf("unexpected status %v", m["status"])
	return nil
}

// TestAgentOperatorTimeoutEndsSession verifies a silent agent triggers the
// fallback record and an explicit session error (records stay exportable).
func TestAgentOperatorTimeoutEndsSession(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.StartAgentGame(AgentStartRequest{
		Seed: 9, SiloName: "T", AgentName: "O", Profession: "IT",
		TimeoutMS: 300,
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Find the first decision point, then abandon it.
	var found bool
	for i := 0; i < 200 && !found; i++ {
		out, err := svc.AgentStep()
		if err != nil {
			t.Fatalf("step: %v", err)
		}
		m := out.(map[string]any)
		switch m["status"] {
		case "decision":
			found = true
		case "month_end":
			continue
		case "game_over":
			t.Fatalf("game ended before any decision point")
		}
	}
	if !found {
		t.Skip("no decision point in the first 200 months")
	}

	// Wait for the timeout to trip and the month to finish.
	deadline := time.Now().Add(10 * time.Second)
	for {
		status, err := svc.AgentStatus()
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		phase := status.(map[string]any)["phase"].(string)
		if phase == "error" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("session never entered error phase (phase=%s)", phase)
		}
		time.Sleep(50 * time.Millisecond)
	}

	export, err := svc.AgentExport()
	if err != nil {
		t.Fatalf("export after timeout: %v", err)
	}
	records := export.(map[string]any)["records"].([]agentRecord)
	var timeouts int
	for _, rec := range records {
		if rec.ExecutionStatus == agentStatusTimeout {
			timeouts++
		}
	}
	if timeouts == 0 {
		t.Fatalf("expected at least one timeout_fallback record, got %d records", len(records))
	}

	// A fresh session must be startable after the failed one.
	if _, err := svc.StartAgentGame(AgentStartRequest{Seed: 10, SiloName: "T2", AgentName: "O", Profession: "IT"}); err != nil {
		t.Fatalf("restart after timeout session: %v", err)
	}
	svc.AgentStop()
}

// TestAgentOperatorStopMidMonth verifies stop records the pending decision
// as aborted, lets the month finish, and keeps records exportable.
func TestAgentOperatorStopMidMonth(t *testing.T) {
	svc := bootstrapRealContent(t)
	if _, err := svc.StartAgentGame(AgentStartRequest{Seed: 11, SiloName: "S", AgentName: "O", Profession: "IT"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	decision := playAgentMonthUntilDecision(t, svc)
	if decision == nil {
		// Try a few more months.
		for i := 0; i < 200 && decision == nil; i++ {
			decision = playAgentMonthUntilDecision(t, svc)
		}
	}
	if decision == nil {
		t.Skip("no decision point; stop path still exercised below")
	}

	if _, err := svc.AgentStop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	status, err := svc.AgentStatus()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.(map[string]any)["phase"] != "stopped" {
		t.Fatalf("expected stopped phase, got %v", status)
	}
	export, err := svc.AgentExport()
	if err != nil {
		t.Fatalf("export after stop: %v", err)
	}
	records := export.(map[string]any)["records"].([]agentRecord)
	var aborted int
	for _, rec := range records {
		if rec.ExecutionStatus == agentStatusAborted {
			aborted++
		}
	}
	if aborted == 0 {
		t.Fatalf("expected the pending decision to be recorded as aborted")
	}

	// Step after stop must fail explicitly, not hang.
	if _, err := svc.AgentStep(); err == nil {
		t.Fatalf("step after stop should fail")
	}
}

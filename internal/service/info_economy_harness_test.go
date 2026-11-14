package service

// INFORMATION line economy harness: plays deterministic seeded three-silo
// games through the full service stack under three fixed strategies and dumps
// measurements as JSON. Opt-in via SILO40_INFO_ECON_DUMP=<output.json> so it
// never runs in normal CI. The policy is fully deterministic (no policy RNG):
// every strategy is a fixed function of observable state.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/model"
)

type econStrategy int

const (
	strategyIdle econStrategy = iota
	strategyMinimal
	strategyRhythmic
)

func (s econStrategy) String() string {
	switch s {
	case strategyIdle:
		return "idle"
	case strategyMinimal:
		return "minimal"
	default:
		return "rhythmic"
	}
}

type econRecord struct {
	Strategy           string         `json:"strategy"`
	Seed               int64          `json:"seed"`
	MonthsToGate       int            `json:"months_to_gate"`        // first month allDepartmentsHoldFragments true; 0 = never
	MonthsToBeliefGate int            `json:"months_to_belief_gate"` // first month every dept holds >=3 facts with B>=0.30 (informationGateFactCount); 0 = never
	MonthsToEnding     int            `json:"months_to_ending"`
	Ending             string         `json:"ending"`
	Won                bool           `json:"won"`
	APSpent            float64        `json:"ap_spent"`
	ActionsExecuted    int            `json:"actions_executed"`
	ActionFailures     map[string]int `json:"action_failures"`
	RebellionAtEnd     float64        `json:"rebellion_at_end"`
	RebellionMax       float64        `json:"rebellion_max"`
	PanicMax           float64        `json:"panic_max"`
	DecisionReason     string         `json:"decision_reason"`
	DecisionPool       []string       `json:"decision_pool"`
	Fires              map[string]int `json:"fires"`
	DeptFragmentsAtEnd map[string]int `json:"dept_fragments_at_end"`
	DeptBeliefsAtEnd   map[string]int `json:"dept_beliefs_at_end"` // facts with B>=0.30 per dept
}

func resourceAmount(silo *model.Silo, resType string) float64 {
	for _, res := range silo.Resources {
		if res.Type == resType {
			return res.Amount
		}
	}
	return 0
}

// strongestShareableFactFor returns the agent's highest-B fact at or above
// the sharing floor (0.30) that the target department does not already hold
// above the floor; deterministic tie-break by fact ID.
func strongestShareableFactFor(agent *model.Agent, dept *model.Profession) (string, bool) {
	held := map[string]bool{}
	if dept != nil {
		for _, belief := range dept.Beliefs {
			if belief.B >= 0.30 {
				held[belief.FactID] = true
			}
		}
	}
	best := ""
	bestB := 0.0
	for _, belief := range agent.Beliefs {
		if belief.B < 0.30 || held[belief.FactID] {
			continue
		}
		if belief.B > bestB || (belief.B == bestB && belief.FactID < best) {
			best = belief.FactID
			bestB = belief.B
		}
	}
	return best, best != ""
}

// departmentByAddress returns the department view for a share target.
func departmentByAddress(silo *model.Silo, name string) *model.Profession {
	for i := range silo.Professions {
		if silo.Professions[i].Name == name {
			return &silo.Professions[i]
		}
	}
	return nil
}

// neediestDepartment returns the department with the fewest facts at or above
// the sharing floor in its belief layer (deterministic tie-break: slice
// order). Falls back to fewest legacy fragments when no beliefs exist.
func neediestDepartment(silo *model.Silo) string {
	best := ""
	bestCount := -1
	for _, prof := range silo.Professions {
		count := 0
		for _, belief := range prof.Beliefs {
			if belief.B >= 0.30 {
				count++
			}
		}
		if bestCount < 0 || count < bestCount {
			best = prof.Name
			bestCount = count
		}
	}
	return best
}

// beliefGateReached reports whether every department holds at least k facts
// with belief at or above the sharing floor (0.30) — the re-anchored gate the
// harness measures alongside the legacy fragment gate.
func beliefGateReached(silo *model.Silo, k int) bool {
	if len(silo.Professions) == 0 {
		return false
	}
	for _, prof := range silo.Professions {
		count := 0
		for _, belief := range prof.Beliefs {
			if belief.B >= 0.30 {
				count++
			}
		}
		if count < k {
			return false
		}
	}
	return true
}

func runEconGame(t *testing.T, seed int64, strategy econStrategy) econRecord {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "econ.db")
	svc := newParityService(t, dbPath)

	req := threeSiloRequest(seed)
	req.AgentName = "EconAgent"
	// Supply profession for every strategy: comparable across strategies, and
	// the minimal strategy can run SUPPLY_RATION (server resolves the cost).
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game seed %d: %v", seed, err)
	}

	rec := econRecord{
		Strategy:           strategy.String(),
		Seed:               seed,
		Fires:              map[string]int{},
		DeptFragmentsAtEnd: map[string]int{},
		DeptBeliefsAtEnd:   map[string]int{},
		ActionFailures:     map[string]int{},
	}
	initialAP := svc.agent.ActionPoints

	// try executes an action and records executed/failed counts keyed by a
	// short label (deterministic diagnostics).
	try := func(label string, action model.AgentAction) {
		res, err := svc.ExecuteAction(action)
		if err != nil {
			rec.ActionFailures[label+":err"]++
			return
		}
		if res.Result.Executed {
			rec.ActionsExecuted++
		} else {
			rec.ActionFailures[label+":"+res.Result.Message]++
		}
	}

	const maxMonths = 120
	for m := 1; m <= maxMonths; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("seed %d strategy %s month %d: %v", seed, strategy, m, err)
		}

		if rec.MonthsToGate == 0 && allDepartmentsHoldFragments(svc.silo) {
			rec.MonthsToGate = m
		}
		if rec.MonthsToBeliefGate == 0 && beliefGateReached(svc.silo, 3) {
			rec.MonthsToBeliefGate = m
		}

		switch strategy {
		case strategyIdle:
			// zero actions for the whole run
		case strategyMinimal:
			// Keep the station alive: one Supply duty when Supplies run low.
			// 15 = SUPPLY_RATION registry cost (server re-resolves anyway).
			if resourceAmount(svc.silo, "Supplies") < 2500 && svc.agent.ActionPoints >= 15 {
				try("ration", model.AgentAction{
					Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies", Cost: 0,
				})
			}
		case strategyRhythmic:
			// Deterministic progression rotation: share the agent's strongest
			// not-yet-held fact to the neediest department; gather across
			// departments in fixed rotation to build a diverse fact base;
			// build one connection early; lie low while rebellion runs hot.
			// Fully deterministic: no policy RNG.
			ap := svc.agent.ActionPoints
			shareCost := model.ACTION_COSTS[model.ActionShareInfo]
			gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
			buildCost := model.ACTION_COSTS[model.ActionBuildConnection]
			rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
			gatherFrom := rotation[m%len(rotation)]
			switch {
			case svc.silo.Rebellion >= 0.7:
				// lie low: no actions while rebellion runs hot
			case ap >= shareCost && m >= 4:
				dept := neediestDepartment(svc.silo)
				fragment, ok := strongestShareableFactFor(svc.agent, departmentByAddress(svc.silo, dept))
				if ok {
					try("share", model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept,
						FragmentIds: []string{fragment}, Cost: 0,
					})
				} else if ap >= gatherCost {
					try("gather", model.AgentAction{
						Type: model.ActionGatherInfo, TargetDept: gatherFrom, Cost: 0,
					})
				}
			case ap >= buildCost && m < 12:
				try("build", model.AgentAction{
					Type: model.ActionBuildConnection, TargetDept: "Judicial", Cost: 0,
				})
			case ap >= gatherCost:
				try("gather", model.AgentAction{
					Type: model.ActionGatherInfo, TargetDept: gatherFrom, Cost: 0,
				})
			}
		}

		if svc.silo.Rebellion > rec.RebellionMax {
			rec.RebellionMax = svc.silo.Rebellion
		}
		// The gate can also cross during the action's own resolution
		// (absorption lands before the instant victory check), so re-measure.
		if rec.MonthsToBeliefGate == 0 && beliefGateReached(svc.silo, 3) {
			rec.MonthsToBeliefGate = m
		}

		for _, prof := range svc.silo.Professions {
			if prof.PanicValue > rec.PanicMax {
				rec.PanicMax = prof.PanicValue
			}
		}

		if svc.gameOver() {
			rec.MonthsToEnding = m
			break
		}
	}

	rec.APSpent = initialAP - svc.agent.ActionPoints
	if silo := svc.silo; silo != nil {
		rec.RebellionAtEnd = silo.Rebellion
		if silo.VictoryStatus != nil {
			rec.Ending = silo.VictoryStatus.Type
			rec.Won = silo.VictoryStatus.IsWon
			if silo.VictoryStatus.Decision != nil {
				rec.DecisionReason = silo.VictoryStatus.Decision.Reason
				rec.DecisionPool = silo.VictoryStatus.Decision.Pool
			}
		} else {
			rec.Ending = "NO_ENDING(max_months)"
		}
		for _, prof := range silo.Professions {
			rec.DeptFragmentsAtEnd[prof.Name] = len(prof.KnownFragments)
			count := 0
			for _, belief := range prof.Beliefs {
				if belief.B >= 0.30 {
					count++
				}
			}
			rec.DeptBeliefsAtEnd[prof.Name] = count
		}
	}
	for _, entry := range svc.eventLogs {
		key := entry.EventID
		if entry.Category != "" {
			key = entry.Category + ":" + entry.EventID
		}
		rec.Fires[key]++
	}
	return rec
}

func TestInfoEconomyDump(t *testing.T) {
	outPath := os.Getenv("SILO40_INFO_ECON_DUMP")
	if outPath == "" {
		t.Skip("set SILO40_INFO_ECON_DUMP=<file> to dump economy records")
	}
	records := []econRecord{}
	for _, seed := range []int64{7, 8} {
		for _, strategy := range []econStrategy{strategyIdle, strategyMinimal, strategyRhythmic} {
			records = append(records, runEconGame(t, seed, strategy))
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Seed != records[j].Seed {
			return records[i].Seed < records[j].Seed
		}
		return records[i].Strategy < records[j].Strategy
	})
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatalf("write dump: %v", err)
	}
	for _, rec := range records {
		t.Logf("seed=%d strategy=%s belief_gate=%d ending_month=%d ending=%s rebellion=%.2f panic_max=%.2f ap_spent=%.1f actions=%d failures=%v pool=%v",
			rec.Seed, rec.Strategy, rec.MonthsToBeliefGate, rec.MonthsToEnding, rec.Ending,
			rec.RebellionAtEnd, rec.PanicMax, rec.APSpent, rec.ActionsExecuted, rec.ActionFailures, rec.DecisionPool)
	}
}

package engine

import (
	"math/rand"
	"strings"
	"testing"

	"silo40/internal/model"
)

// TestGatherInfoAccumulatesBelief locks the §3.4/§3.5 rework end to end on a
// fixed seed: probes roll p_get, successes absorb observations into B (never
// reaching 1), and every known fragment carries a belief entry.
func TestGatherInfoAccumulatesBelief(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(42)))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Gather Silo", 122, nil)
	// Medical agent: unlike IT agents it does not start knowing every fragment.
	agent := CreateInitialAgent("MedicAgent", "Medical", nil, silo)
	initialKnown := map[string]bool{}
	for _, fragment := range agent.KnownFragments {
		initialKnown[fragment] = true
	}

	for i := 0; i < 12; i++ {
		res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
			Type: model.ActionGatherInfo, TargetDept: "Judicial", Cost: 10,
		})
		if !res.Executed {
			if strings.Contains(res.Message, "already knows everything") {
				break // fragment pool exhausted
			}
			t.Fatalf("probe %d: %s", i+1, res.Message)
		}
	}
	observed := []string{}
	for _, fragment := range agent.KnownFragments {
		if !initialKnown[fragment] {
			observed = append(observed, fragment)
		}
	}
	if len(observed) == 0 {
		t.Fatal("12 seeded probes must yield at least one observation")
	}
	for _, fragment := range observed {
		belief := BeliefFor(agent, fragment)
		if belief == nil {
			t.Fatalf("observed fragment %s has no belief entry", fragment)
		}
		if belief.B <= 0 || belief.B >= 1 {
			t.Fatalf("belief B out of (0,1) for %s: %v", fragment, belief.B)
		}
		if belief.ObsCount < 1 || len(belief.Observations) != belief.ObsCount {
			t.Fatalf("observation bookkeeping broken for %s: %+v", fragment, belief)
		}
	}
}

// TestGatherInfoFailureRaisesAlertness locks §3.4: failed probes settle AP and
// raise the target's alertness without granting belief.
func TestGatherInfoFailureRaisesAlertness(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(7)))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Alert Silo", 122, nil)
	agent := CreateInitialAgent("MedicAgent", "Medical", nil, silo)

	judicial := findDept(silo, "Judicial")
	if judicial == nil {
		t.Fatal("fixture missing Judicial department")
	}
	failures := 0
	for i := 0; i < 20 && failures == 0; i++ {
		before := len(agent.KnownFragments)
		res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
			Type: model.ActionGatherInfo, TargetDept: "Judicial", Cost: 10,
		})
		if !res.Executed {
			t.Fatalf("probe %d: %s", i+1, res.Message)
		}
		if len(agent.KnownFragments) == before {
			failures++
		}
	}
	if failures == 0 {
		t.Skip("seed produced no failures; alertness path covered by mutation tests")
	}
	if judicial.Alertness <= 0 {
		t.Fatalf("failed probes must raise alertness, got %v", judicial.Alertness)
	}
}

// TestGatherInfoMinesTargetUnsampled locks §8.2 at the action level: probing
// the Mines complex never adds ordinary suspicion (structural exemption), AP
// still charged.
func TestGatherInfoMinesTargetUnsampled(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(3)))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Mines Silo", 122, nil)
	agent := CreateInitialAgent("MedicAgent", "Medical", nil, silo)

	for i := 0; i < 6; i++ {
		res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
			Type: model.ActionGatherInfo, TargetDept: "Mines", Cost: 10,
		})
		if !res.Executed {
			if strings.Contains(res.Message, "already knows everything") {
				break // Mines only carries two fragments; exhaustion is fine
			}
			t.Fatalf("probe %d: %s", i+1, res.Message)
		}
		if agent.SuspicionLevel != 0 {
			t.Fatalf("Mines probe raised ordinary suspicion: %v", agent.SuspicionLevel)
		}
	}
}

// TestGatherInfoRecheckDeepensKnownBelief locks 修复6: once a department's
// new material is exhausted, Gather switches to re-checking known material
// and deepens the agent's existing belief instead of dead-ending. Without
// this path professions whose home domain holds only two fragments (Supply,
// Mechanical, Police, Medical) can never raise B_player high enough for
// share absorption to be effective — the 《共证》 line was de facto unwinnable
// for them (职业横评).
func TestGatherInfoRecheckDeepensKnownBelief(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(42)))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Recheck Silo", 122, nil)
	agent := CreateInitialAgent("SupplyAgent", "Supply", nil, silo)
	// Supply domain carries exactly two fragments, both known from the start:
	// the new-material pool is exhausted on the very first probe.
	if len(agent.KnownFragments) != 2 {
		t.Fatalf("expected the Supply agent to start with 2 fragments, got %v", agent.KnownFragments)
	}
	SeedLegacyBeliefs(agent)
	fragment := agent.KnownFragments[0]
	before := BeliefFor(agent, fragment).B

	deepened := false
	for month := 1; month <= 10 && !deepened; month++ {
		silo.CurrentMonth = month
		res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
			Type: model.ActionGatherInfo, TargetDept: "Supply", Cost: 2,
		})
		if !res.Executed {
			t.Fatalf("re-check probe rejected: %s", res.Message)
		}
		if strings.Contains(res.Message, "already knows everything") {
			t.Fatalf("exhausted department must switch to re-check instead of failing: %s", res.Message)
		}
		if BeliefFor(agent, fragment).B > before {
			deepened = true
		}
	}
	if !deepened {
		t.Fatal("re-check never deepened the known fragment belief")
	}
}

// TestShareInfoRequiresBeliefFloor locks §3.6: material below the rumor floor
// (B < 0.30) is private suspicion and cannot be shared as a fact.
func TestShareInfoRequiresBeliefFloor(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Share Silo", 122, nil)
	agent := CreateInitialAgent("ITAgent", "IT", nil, silo)
	agent.KnownFragments = []string{"Judicial_1"} // known, but no belief entry yet

	res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Supply",
		FragmentIds: []string{"Judicial_1"}, Cost: 20,
	})
	if res.Executed {
		t.Fatalf("sharing private suspicion must be rejected: %s", res.Message)
	}

	SeedLegacyBeliefs(agent) // B=0.65 > 0.30 floor
	res = e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Supply",
		FragmentIds: []string{"Judicial_1"}, Cost: 20,
	})
	if !res.Executed {
		t.Fatalf("seeded belief must pass the share floor: %s", res.Message)
	}
}

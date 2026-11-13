package engine

import (
	"math/rand"
	"testing"
)

// PredictCandidates and DecisionActors feed the LLM prefetch layer and must
// never consume the engine rng (R1): the live draw order has to stay
// identical with and without prefetching.
func TestPredictCandidatesDoesNotConsumeRng(t *testing.T) {
	silo := CreateInitialSilo("PredictRng", 122, nil)
	agent := CreateInitialAgent("Agent", "IT", nil, silo)
	view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[6]), silo, nil)
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	_ = agent

	draw := func(predict bool) []float64 {
		SetRNG(rand.New(rand.NewSource(42)))
		if predict {
			for _, v := range DecisionActors(silo, agent) {
				PredictCandidates(v, silo)
			}
		}
		out := make([]float64, 8)
		for i := range out {
			out[i] = rng.Float64()
		}
		return out
	}
	withPredict := draw(true)
	without := draw(false)
	for i := range withPredict {
		if withPredict[i] != without[i] {
			t.Fatalf("prediction consumed the rng stream at draw %d: %v != %v",
				i, withPredict[i], without[i])
		}
	}
	// Sanity: the initial silo actually yields prediction material.
	if got := PredictCandidates(view, silo); len(got) == 0 {
		t.Fatalf("expected non-empty prediction for a fresh department actor")
	}
}

// DecisionActors must enumerate exactly the RunNpcTurn decision population:
// every profession except the player's, plus alive representatives.
func TestDecisionActorsEnumeration(t *testing.T) {
	silo := CreateInitialSilo("Enumerate", 122, nil)
	// The player profession must be an existing one for the exclusion to apply.
	agent := CreateInitialAgent("Agent", silo.Professions[0].Name, nil, silo)
	views := DecisionActors(silo, agent)
	reps := 0
	for i := range silo.Residents {
		if silo.Residents[i].Alive && silo.Residents[i].IsRepresentative {
			reps++
		}
	}
	want := len(silo.Professions) - 1 + reps // player profession excluded
	if len(views) != want {
		t.Fatalf("DecisionActors: got %d views, want %d (%d profs + %d reps)",
			len(views), want, len(silo.Professions)-1, reps)
	}
	for _, v := range views {
		if v.Profession() == agent.Profession {
			t.Fatalf("player profession %q must not be enumerated", agent.Profession)
		}
	}
	// A nil agent enumerates every profession.
	if got := DecisionActors(silo, nil); len(got) != len(silo.Professions)+reps {
		t.Fatalf("nil agent: got %d views, want %d", len(got), len(silo.Professions)+reps)
	}
}

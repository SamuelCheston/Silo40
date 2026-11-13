package engine

import (
	"math/rand"
	"testing"
	"time"

	"silo40/internal/model"
)

// seedForTest swaps the engine rng for a seeded source and restores a fresh
// time-seeded source (the package default) on cleanup.
func seedForTest(t *testing.T, seed int64) {
	t.Helper()
	SetRNG(rand.New(rand.NewSource(seed)))
	t.Cleanup(func() { SetRNG(rand.New(rand.NewSource(time.Now().UnixNano()))) })
}

// recordingPolicy captures Advise/Committed calls for assertions.
type recordingPolicy struct {
	adviseCalls    int
	committedCalls int
	lastChosen     int
	lastInfos      []CandidateInfo
	advice         *NpcAdvice
	// pickByType, when set, forces the first candidate of that type.
	forceType model.AgentActionType
	// boostType, when set, returns Scores=multiplier for candidates of type.
	boostType  model.AgentActionType
	boostMult  float64
	factorType model.AgentActionType
	factorMult float64
}

func (p *recordingPolicy) Advise(_ *ActorView, _ *model.Silo, candidates []CandidateInfo) *NpcAdvice {
	p.adviseCalls++
	p.lastInfos = candidates
	if p.forceType != "" {
		for _, c := range candidates {
			if c.Type == p.forceType {
				idx := c.Index
				return &NpcAdvice{Force: &idx}
			}
		}
		return nil
	}
	adv := &NpcAdvice{}
	if p.boostType != "" {
		adv.Scores = map[int]float64{}
		for _, c := range candidates {
			if c.Type == p.boostType {
				adv.Scores[c.Index] = p.boostMult
			}
		}
	}
	if p.factorType != "" {
		adv.Factors = map[model.AgentActionType]float64{p.factorType: p.factorMult}
	}
	if len(adv.Scores) == 0 && len(adv.Factors) == 0 {
		return nil
	}
	return adv
}

func (p *recordingPolicy) Committed(_ *ActorView, _ *model.Silo, _ []CandidateInfo, _ *NpcAdvice, chosenIndex int) {
	p.committedCalls++
	p.lastChosen = chosenIndex
}

func newPolicyTestSilo(t *testing.T, seed int64) (*model.Silo, *ActorView) {
	t.Helper()
	seedForTest(t, seed)
	silo := CreateInitialSilo("PolicyTest", 122, nil)
	view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[6]), silo, nil) // Mechanical
	if err != nil {
		t.Fatalf("create view: %v", err)
	}
	return silo, view
}

// TestAdviceForcePreservesRngStream locks the determinism contract (R1): a
// forced (replayed) decision must consume exactly the same rng draws as the
// unadvised path, so subsequent state evolution stays bit-for-bit identical.
func TestAdviceForcePreservesRngStream(t *testing.T) {
	draws := func(policy NpcPolicy) []float64 {
		seedForTest(t, 7)
		silo := CreateInitialSilo("Stream", 122, nil)
		view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[6]), silo, nil)
		if err != nil {
			t.Fatalf("create view: %v", err)
		}
		brain := NpcBrain{policy: policy}
		var out []float64
		for i := 0; i < 40; i++ {
			brain.Decide(view, silo, 1.0) // deltaYears=1 => attempt always
			out = append(out, rng.Float64())
		}
		return out
	}

	base := draws(nil)
	forced := draws(&recordingPolicy{forceType: model.ActionBuildConnection})
	if len(base) != len(forced) {
		t.Fatalf("length mismatch")
	}
	for i := range base {
		if base[i] != forced[i] {
			t.Fatalf("rng stream diverged at draw %d: base=%v forced=%v", i, base[i], forced[i])
		}
	}
}

// TestAdviceNilPolicyIsOriginalBehavior: nil advice and nil policy must
// produce identical choice sequences under the same seed.
func TestAdviceNilPolicyIsOriginalBehavior(t *testing.T) {
	run := func(policy NpcPolicy) []model.AgentActionType {
		seedForTest(t, 21)
		silo := CreateInitialSilo("NilPolicy", 122, nil)
		view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[1]), silo, nil)
		if err != nil {
			t.Fatalf("create view: %v", err)
		}
		brain := NpcBrain{policy: policy}
		var picks []model.AgentActionType
		for i := 0; i < 60; i++ {
			if d := brain.Decide(view, silo, 1.0); d != nil {
				picks = append(picks, d.Action.Type)
			}
		}
		return picks
	}
	a := run(nil)
	b := run(&recordingPolicy{}) // returns nil advice every time
	if len(a) == 0 {
		t.Fatalf("expected some decisions")
	}
	if len(a) != len(b) {
		t.Fatalf("decision count diverged: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("pick %d diverged: %v vs %v", i, a[i], b[i])
		}
	}
}

// TestAdviceScoresBiasSelection: boosting one candidate type raises its
// selection share; Committed reports the matching index.
func TestAdviceScoresBiasSelection(t *testing.T) {
	count := func(policy NpcPolicy) (map[model.AgentActionType]int, *recordingPolicy) {
		seedForTest(t, 33)
		silo := CreateInitialSilo("Bias", 122, nil)
		view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[6]), silo, nil)
		if err != nil {
			t.Fatalf("create view: %v", err)
		}
		rec, _ := policy.(*recordingPolicy)
		brain := NpcBrain{policy: policy}
		picks := map[model.AgentActionType]int{}
		for i := 0; i < 300; i++ {
			if d := brain.Decide(view, silo, 1.0); d != nil {
				picks[d.Action.Type]++
			}
		}
		return picks, rec
	}

	base, _ := count(&recordingPolicy{})
	boosted, rec := count(&recordingPolicy{boostType: model.ActionGatherInfo, boostMult: AdviceFactorMax})
	if boosted[model.ActionGatherInfo] <= base[model.ActionGatherInfo] {
		t.Fatalf("boost did not raise selection share: base=%d boosted=%d",
			base[model.ActionGatherInfo], boosted[model.ActionGatherInfo])
	}
	if rec.committedCalls == 0 {
		t.Fatalf("Committed never called")
	}
	// Channel B (factor) on a type with a single candidate must also shift.
	factored, _ := count(&recordingPolicy{factorType: model.ActionConductPropaganda, factorMult: AdviceFactorMax})
	if factored[model.ActionConductPropaganda] <= base[model.ActionConductPropaganda] {
		t.Fatalf("factor did not raise selection share: base=%d factored=%d",
			base[model.ActionConductPropaganda], factored[model.ActionConductPropaganda])
	}
}

// TestApplyAdviceClamp verifies per-channel and combined clamp bounds.
func TestApplyAdviceClamp(t *testing.T) {
	mk := func() []WeightedDecision {
		return []WeightedDecision{
			{decision: NpcDecision{Action: model.AgentAction{Type: model.ActionGatherInfo}}, weight: 1.0},
			{decision: NpcDecision{Action: model.AgentAction{Type: model.ActionShareInfo}}, weight: 1.0},
		}
	}
	// Per-channel clamp: extreme values collapse to the [0.5, 2] band.
	cs := mk()
	applyAdvice(cs, &NpcAdvice{
		Factors: map[model.AgentActionType]float64{model.ActionGatherInfo: 100},
		Scores:  map[int]float64{0: 100},
	})
	if cs[0].weight != AdviceCombinedMax {
		t.Fatalf("expected combined max %v, got %v", AdviceCombinedMax, cs[0].weight)
	}
	if cs[1].weight != 1.0 {
		t.Fatalf("untouched candidate changed: %v", cs[1].weight)
	}
	cs = mk()
	applyAdvice(cs, &NpcAdvice{
		Factors: map[model.AgentActionType]float64{model.ActionGatherInfo: -5},
		Scores:  map[int]float64{0: -5},
	})
	if cs[0].weight != AdviceCombinedMin {
		t.Fatalf("expected combined min %v, got %v", AdviceCombinedMin, cs[0].weight)
	}
	// nil advice leaves weights intact.
	cs = mk()
	applyAdvice(cs, nil)
	if cs[0].weight != 1.0 || cs[1].weight != 1.0 {
		t.Fatalf("nil advice mutated weights")
	}
}

// TestPolicySkippedOnGateMiss: when the attempt gate misses, neither Advise
// nor Committed may run (no rng-free side channels into the decision layer).
func TestPolicySkippedOnGateMiss(t *testing.T) {
	seedForTest(t, 5)
	silo := CreateInitialSilo("Gate", 122, nil)
	view, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[0]), silo, nil)
	if err != nil {
		t.Fatalf("create view: %v", err)
	}
	rec := &recordingPolicy{}
	brain := NpcBrain{policy: rec}
	for i := 0; i < 50; i++ {
		if d := brain.Decide(view, silo, 0); d != nil {
			t.Fatalf("deltaYears=0 must never decide")
		}
	}
	if rec.adviseCalls != 0 || rec.committedCalls != 0 {
		t.Fatalf("policy called on gate miss: advise=%d committed=%d", rec.adviseCalls, rec.committedCalls)
	}
}

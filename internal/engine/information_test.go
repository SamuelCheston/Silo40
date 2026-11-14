package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

// TestAbsorptionSaturatesThirdAndFifth locks the §3.5 table: with s=0.65 the
// third observation is near-certain (≈0.905) and the fifth still below 1
// (≈0.960) — belief saturates fast but never reaches 1.
func TestAbsorptionSaturatesThirdAndFifth(t *testing.T) {
	cfg := DefaultInformationConfig()
	spec := FragmentSpecFor("Judicial_1")
	belief := &model.FactBelief{FactID: "Judicial_1", Versions: InitialVersions(spec)}

	want := []float64{0.650, 0.832, 0.905, 0.940, 0.960}
	for i, w := range want {
		observed := ObserveFact(cfg, belief, spec, obsID(i), 0.65)
		if !observed {
			t.Fatalf("observation %d was not absorbed", i+1)
		}
		if math.Abs(belief.B-w) > 0.001 {
			t.Fatalf("after %d observations B = %.4f, want ≈ %.3f", i+1, belief.B, w)
		}
	}
	if belief.B >= 1 {
		t.Fatalf("belief must never reach 1: %v", belief.B)
	}
	if belief.ObsCount != 5 || len(belief.Observations) != 5 {
		t.Fatalf("observation bookkeeping mismatch: %+v", belief)
	}
}

func obsID(i int) string {
	return "obs#" + string(rune('a'+i))
}

// TestObservationDeduplication locks §3.4: the same observation_id never
// replays its benefit (no RNG or gain replay).
func TestObservationDeduplication(t *testing.T) {
	cfg := DefaultInformationConfig()
	spec := FragmentSpecFor("IT_2")
	belief := &model.FactBelief{FactID: "IT_2", Versions: InitialVersions(spec)}

	if !ObserveFact(cfg, belief, spec, "gather:IT_2", 0.65) {
		t.Fatal("first observation must absorb")
	}
	b, n := belief.B, belief.ObsCount
	if ObserveFact(cfg, belief, spec, "gather:IT_2", 0.65) {
		t.Fatal("duplicate observation_id must not replay")
	}
	if belief.B != b || belief.ObsCount != n {
		t.Fatalf("duplicate observation changed belief: B %v->%v n %d->%d", b, belief.B, n, belief.ObsCount)
	}
}

// TestVersionBayesUpdate locks §3.5: observing an IT/Judicial fragment shifts
// π toward the it_directed version and π stays normalized.
func TestVersionBayesUpdate(t *testing.T) {
	cfg := DefaultInformationConfig()
	spec := FragmentSpecFor("Judicial_1")
	belief := &model.FactBelief{FactID: "Judicial_1", Versions: InitialVersions(spec)}

	ObserveFact(cfg, belief, spec, "obs#1", 0.65)
	directed := belief.Versions[VersionITDirected]
	mundane := belief.Versions[VersionMundane]
	if directed <= 0.5 || mundane >= 0.5 {
		t.Fatalf("π must shift toward it_directed after observation: %+v", belief.Versions)
	}
	if math.Abs(directed+mundane-1) > 1e-12 {
		t.Fatalf("π must stay normalized: %+v", belief.Versions)
	}

	// A single-version (mundane) fragment keeps π trivial.
	specM := FragmentSpecFor("Medical_1")
	beliefM := &model.FactBelief{FactID: "Medical_1", Versions: InitialVersions(specM)}
	ObserveFact(cfg, beliefM, specM, "obs#m", 0.65)
	if math.Abs(beliefM.Versions[VersionMundane]-1) > 1e-12 {
		t.Fatalf("mundane-only fragment must keep trivial π: %+v", beliefM.Versions)
	}
}

// TestPGetBoundsAndFactors locks §3.4: clamped to [0.05, 0.95], monotone in
// alertness/mismatch and in related bonus.
func TestPGetBoundsAndFactors(t *testing.T) {
	cfg := DefaultInformationConfig()
	base := PGet(cfg, 1.0, 0, 0, 0, 1.0)
	if math.Abs(base-0.55) > 1e-12 {
		t.Fatalf("neutral p_get = %v, want 0.55", base)
	}
	if PGet(cfg, 1.0, 1, 0, 0, 1.0) >= base {
		t.Fatal("mismatch must lower p_get")
	}
	if PGet(cfg, 1.0, 0, 1, 0, 1.0) >= base {
		t.Fatal("alertness must lower p_get")
	}
	if PGet(cfg, 1.0, 0, 0, 1, 1.0) <= base {
		t.Fatal("related facts must raise p_get")
	}
	if got := PGet(cfg, 10, 0, 0, 10, 10); got != cfg.PGetMax {
		t.Fatalf("p_get must clamp at %v, got %v", cfg.PGetMax, got)
	}
	if got := PGet(cfg, 0.01, 1, 1, 0, 0.1); got != cfg.PGetMin {
		t.Fatalf("p_get must clamp at %v, got %v", cfg.PGetMin, got)
	}
}

// TestMismatchAndAPIndependence locks §8.1: mismatch = |a|·max(0,-sign(a)·d)
// and AP = ceil(base·(1+0.5·mismatch)), computed independently of suspicion.
func TestMismatchAndAPIndependence(t *testing.T) {
	if got := Mismatch(0.4, -0.5); math.Abs(got-0.2) > 1e-12 {
		t.Fatalf("closed-norm mismatch = %v, want 0.2", got)
	}
	if got := Mismatch(0.4, 0.5); got != 0 {
		t.Fatalf("open-norm mismatch = %v, want 0", got)
	}
	if got := Mismatch(0, -1); got != 0 {
		t.Fatalf("zero deviation must give zero mismatch, got %v", got)
	}

	cfg := DefaultInformationConfig()
	if got := APCostWithMismatch(cfg, 10, 0); got != 10 {
		t.Fatalf("AP without mismatch = %v, want 10", got)
	}
	if got := APCostWithMismatch(cfg, 10, 0.2); got != 11 {
		t.Fatalf("AP with mismatch 0.2 = %v, want ceil(11)=11", got)
	}
	if got := APCostWithMismatch(cfg, 10, 1); got != 15 {
		t.Fatalf("AP with max mismatch = %v, want 15", got)
	}
}

// TestQualificationTiers locks the §3.6 ladder.
func TestQualificationTiers(t *testing.T) {
	cfg := DefaultInformationConfig()
	cases := []struct {
		b    float64
		want string
	}{
		{0.10, "private"},
		{0.30, "rumor"},
		{0.59, "rumor"},
		{0.60, "investigate"},
		{0.85, "report"},
		{0.90, "publish"},
	}
	for _, tc := range cases {
		if got := QualificationTier(cfg, tc.b); got != tc.want {
			t.Errorf("tier(%.2f) = %s, want %s", tc.b, got, tc.want)
		}
	}
}

// TestSeedLegacyBeliefs locks the save-compat path: KnownFragments without
// belief entries are seeded at one conservative observation (B=0.65), and
// seeding is idempotent.
func TestSeedLegacyBeliefs(t *testing.T) {
	agent := &model.Agent{KnownFragments: []string{"Judicial_1", "IT_3"}}
	SeedLegacyBeliefs(agent)
	for _, fragment := range agent.KnownFragments {
		belief := BeliefFor(agent, fragment)
		if belief == nil {
			t.Fatalf("legacy fragment %s has no belief", fragment)
		}
		if math.Abs(belief.B-DefaultInformationConfig().LegacySeedB) > 1e-12 {
			t.Fatalf("legacy seed B = %v, want %v", belief.B, DefaultInformationConfig().LegacySeedB)
		}
		if belief.ObsCount != 1 {
			t.Fatalf("legacy seed must count one observation: %+v", belief)
		}
	}
	before := len(agent.Beliefs)
	SeedLegacyBeliefs(agent)
	if len(agent.Beliefs) != before {
		t.Fatal("seeding must be idempotent")
	}
}

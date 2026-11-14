package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

// envFixtureSilo builds a silo with the initial professions (populations from
// the initializer) and one influential named resident.
func envFixtureSilo() *model.Silo {
	silo := CreateInitialSilo("Env Fixture", 122, nil)
	silo.Residents = []model.Resident{
		{ID: 1, SiloID: silo.ID, Name: "Named Witness", Alive: true, Influence: 10},
	}
	return silo
}

func departmentBelief(t *testing.T, silo *model.Silo, dept, factID string) *model.FactBelief {
	t.Helper()
	for i := range silo.Professions {
		if silo.Professions[i].Name != dept {
			continue
		}
		return BeliefIn(&silo.Professions[i].Beliefs, factID)
	}
	t.Fatalf("department %s not found", dept)
	return nil
}

// TestPrivatePlayerBeliefNeverRaisesExposure locks §3.3: the player's own B
// never enters E — only department and named-resident belief layers aggregate.
func TestPrivatePlayerBeliefNeverRaisesExposure(t *testing.T) {
	silo := envFixtureSilo()
	agent := &model.Agent{ID: 1}
	cfg := DefaultInformationConfig()
	belief := EnsureBelief(agent, "Judicial_1")
	spec := FragmentSpecFor("Judicial_1")
	for i := 0; i < 5; i++ {
		ObserveFact(cfg, belief, spec, obsID(i), 0.9)
	}
	if belief.B < 0.9 {
		t.Fatalf("player belief did not build: %v", belief.B)
	}
	env := ComputeInformationEnvironment(silo)
	if env.ExposureNow != 0 || env.Exposure != 0 {
		t.Fatalf("private player belief leaked into exposure: %+v", env)
	}
}

// TestExposureAggregatesLayeredBeliefs locks §3.3: department and resident
// beliefs with an it_directed attribution raise E; facts without attribution
// (mundane-only domains) never do; repeated facts count once.
func TestExposureAggregatesLayeredBeliefs(t *testing.T) {
	silo := envFixtureSilo()
	cfg := DefaultInformationConfig()

	// Supply (large department) absorbs the core dispatch fact.
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Name != "Supply" {
			continue
		}
		belief := EnsureBeliefIn(&prof.Beliefs, "Judicial_1")
		spec := FragmentSpecFor("Judicial_1")
		for k := 0; k < 4; k++ {
			ObserveFact(cfg, belief, spec, obsID(k), 0.9)
		}
		if belief.B < 0.9 {
			t.Fatalf("department belief did not build: %v", belief.B)
		}
	}
	// The named resident absorbs it too.
	for i := range silo.Residents {
		belief := EnsureBeliefIn(&silo.Residents[i].Beliefs, "Judicial_1")
		ObserveFact(cfg, belief, FragmentSpecFor("Judicial_1"), obsID(0), 0.9)
	}

	env := ComputeInformationEnvironment(silo)
	if env.ExposureNow <= 0 {
		t.Fatalf("layered it-directed beliefs must raise exposure: %+v", env)
	}

	// A mundane-only domain fact can never attribute to IT.
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Name != "Supply" {
			continue
		}
		belief := EnsureBeliefIn(&prof.Beliefs, "Medical_1")
		ObserveFact(cfg, belief, FragmentSpecFor("Medical_1"), obsID(0), 0.9)
	}
	envMundane := ComputeInformationEnvironment(silo)
	if envMundane.ExposureNow != env.ExposureNow {
		t.Fatalf("mundane fact changed exposure: %v -> %v", env.ExposureNow, envMundane.ExposureNow)
	}

	// High-water mark: exposure never decays when beliefs are wiped.
	silo.Authority.E = env.ExposureNow
	for i := range silo.Professions {
		silo.Professions[i].Beliefs = nil
	}
	for i := range silo.Residents {
		silo.Residents[i].Beliefs = nil
	}
	envAfter := ComputeInformationEnvironment(silo)
	if envAfter.ExposureNow != 0 {
		t.Fatalf("wiped beliefs must give E_now=0, got %+v", envAfter)
	}
	if envAfter.Exposure != env.ExposureNow {
		t.Fatalf("exposure high-water mark must hold: %+v", envAfter)
	}
}

// TestPublicEvidenceQReachesCovenantThreshold locks §3.7: two core dispatch
// facts with high judicial belief, verified provenance and high coverage reach
// Q ≥ 0.85 and count as two qualifying independent facts.
func TestPublicEvidenceQReachesCovenantThreshold(t *testing.T) {
	silo := envFixtureSilo()
	cfg := DefaultInformationConfig()

	for _, factID := range []string{"Judicial_1", "Judicial_2"} {
		// The Judicial department absorbs two observations per fact (witnessed
		// dispatch + verified publication).
		for i := range silo.Professions {
			prof := &silo.Professions[i]
			if prof.Name != "Judicial" {
				continue
			}
			belief := EnsureBeliefIn(&prof.Beliefs, factID)
			spec := FragmentSpecFor(factID)
			if !ObserveFact(cfg, belief, spec, "witness:"+factID, AbsorptionStrength(cfg, 0.9, 0, 1)) {
				t.Fatalf("witness observation rejected for %s", factID)
			}
			if !ObserveFact(cfg, belief, spec, "publish:"+factID, AbsorptionStrength(cfg, 1.0, 0, 1)) {
				t.Fatalf("publication observation rejected for %s", factID)
			}
			if belief.B < cfg.PactFactBelief {
				t.Fatalf("judicial belief for %s = %v, want >= %v", factID, belief.B, cfg.PactFactBelief)
			}
		}
		RegisterPublicEvidence(silo, model.PublicEvidenceRecord{
			FactID: factID, Provenance: ProvenanceVerified, Coverage: 0.9, Year: 122, Month: 3,
		})
	}

	env := ComputeInformationEnvironment(silo)
	if env.PublicEvidenceQ < 0.85 {
		t.Fatalf("two verified core evidences must reach Q >= 0.85, got %v", env.PublicEvidenceQ)
	}
	if env.QualifyingFacts != 2 {
		t.Fatalf("qualifying independent facts = %d, want 2 (%+v)", env.QualifyingFacts, env.QualifyingFactIDs)
	}

	// Unpublished evidence never counts, however strong the belief.
	silo2 := envFixtureSilo()
	for i := range silo2.Professions {
		if silo2.Professions[i].Name != "Judicial" {
			continue
		}
		belief := EnsureBeliefIn(&silo2.Professions[i].Beliefs, "Judicial_1")
		ObserveFact(cfg, belief, FragmentSpecFor("Judicial_1"), "witness", 0.99)
	}
	env2 := ComputeInformationEnvironment(silo2)
	if env2.PublicEvidenceQ != 0 || env2.QualifyingFacts != 0 {
		t.Fatalf("unpublished belief must not produce Q: %+v", env2)
	}
}

// TestRegisterPublicEvidenceDeduplicates locks that re-publication only raises
// provenance/coverage (same fact registered once, deterministic max).
func TestRegisterPublicEvidenceDeduplicates(t *testing.T) {
	silo := envFixtureSilo()
	RegisterPublicEvidence(silo, model.PublicEvidenceRecord{FactID: "Judicial_1", Provenance: 0.6, Coverage: 0.4})
	RegisterPublicEvidence(silo, model.PublicEvidenceRecord{FactID: "Judicial_1", Provenance: 1.0, Coverage: 0.2})
	if len(silo.PublicEvidence) != 1 {
		t.Fatalf("same fact must register once: %+v", silo.PublicEvidence)
	}
	record := silo.PublicEvidence[0]
	if record.Provenance != 1.0 || record.Coverage != 0.4 {
		t.Fatalf("re-publication must take the max of each field: %+v", record)
	}
}

// TestShareAbsorptionDiminishesOnRepeat locks §3.5/§3.6: the first share from
// a source counts as an independent observation; repeats are retold at 0.25.
func TestShareAbsorptionDiminishesOnRepeat(t *testing.T) {
	cfg := DefaultInformationConfig()
	var beliefs []model.FactBelief

	first := AbsorbSharedFact(cfg, &beliefs, "Judicial_1", "player", 0.8)
	if !first {
		t.Fatal("first share must absorb")
	}
	b1 := BeliefIn(&beliefs, "Judicial_1").B

	AbsorbSharedFact(cfg, &beliefs, "Judicial_1", "player", 0.8)
	b2 := BeliefIn(&beliefs, "Judicial_1").B
	gain2 := b2 - b1
	if gain2 <= 0 {
		t.Fatal("retold share must still add (diminished) belief")
	}
	// A fresh independent observation of the same strength must add more than
	// the diminished retell.
	before := b2
	ObserveFact(cfg, BeliefIn(&beliefs, "Judicial_1"), FragmentSpecFor("Judicial_1"), "independent#x", 0.8*cfg.IndependenceNew)
	if BeliefIn(&beliefs, "Judicial_1").B-before <= gain2 {
		t.Fatalf("independent observation must outweigh the retold share: %v vs %v", BeliefIn(&beliefs, "Judicial_1").B-before, gain2)
	}
	if math.Abs(BeliefIn(&beliefs, "Judicial_1").B-1) < 1e-12 {
		t.Fatal("belief must never reach 1")
	}
}

// TestAdvanceAuthorityMonthlyFeedsExposureIntoW locks the wiring: belief-layer
// exposure (not safeguard risk) feeds the W equation this round.
func TestAdvanceAuthorityMonthlyFeedsExposureIntoW(t *testing.T) {
	silo := envFixtureSilo()
	InitializeAuthorityState(silo, false)
	cfg := DefaultInformationConfig()

	// Strong it-directed belief in the largest department.
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if prof.Name != "Mechanical" {
			continue
		}
		belief := EnsureBeliefIn(&prof.Beliefs, "Judicial_1")
		for k := 0; k < 5; k++ {
			ObserveFact(cfg, belief, FragmentSpecFor("Judicial_1"), obsID(k), 0.9)
		}
	}

	wBefore := silo.Authority.W
	AdvanceAuthorityMonthly(silo)
	env := ComputeInformationEnvironment(silo)
	if env.Exposure <= 0 {
		t.Fatalf("expected positive exposure, got %+v", env)
	}
	if silo.Authority.E != env.Exposure {
		t.Fatalf("authority E must carry the high-water exposure: %v vs %v", silo.Authority.E, env.Exposure)
	}
	// With K held high the exposure term must still pull W below the no-exposure
	// equilibrium: recompute the expected step with the same inputs.
	alpha := 1 - math.Pow(2, -1.0/8)
	wEq := 50 + 25*silo.Authority.K + 12*silo.Authority.D + 10*silo.Authority.P + 12*silo.Authority.X - 40*env.Exposure
	want := clamp(wBefore+alpha*(wEq-wBefore), 0, 100)
	if math.Abs(silo.Authority.W-want) > 1e-9 {
		t.Fatalf("W step with exposure = %v, want %v", silo.Authority.W, want)
	}
}

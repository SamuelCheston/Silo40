package engine

import (
	"math"
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// TestItAllianceStructuralBoundaries locks §5.1/§5.4/§11.1/§11.2: maintenance
// registers only bailiff and IT-internal groups; the candidate set contains no
// public-appearance or anti-order actions; the only direct daily leak is the
// bailiff dispatch action.
func TestItAllianceStructuralBoundaries(t *testing.T) {
	for _, action := range ItActionCatalog() {
		switch action.Group {
		case "", "bailiff", "it_internal":
		default:
			t.Fatalf("maintenance outside the dual alliance: %+v", action)
		}
		if action.DirectExposure > 0 && action.ID != "dispatch_bailiff" {
			t.Fatalf("direct exposure outside bailiff dispatch: %+v", action)
		}
	}
	// Police/Mayor/technical groups have no maintenance entry at all.
	for _, group := range []string{"police", "mayor", "mechanical"} {
		for _, action := range ItActionCatalog() {
			if action.Group == group {
				t.Fatalf("group %s must not be buyable: %+v", group, action)
			}
		}
	}
}

// TestItCostCurve locks §5.2: ceil(base·(1+3·(1-W/100)^2)) with the documented
// multiplier landmarks.
func TestItCostCurve(t *testing.T) {
	cases := []struct {
		w    float64
		want float64 // base 10
	}{
		{100, 10},
		{70, 13}, // 10·1.27 = 12.7 -> 13
		{30, 25}, // 10·2.47 = 24.7 -> 25
		{0, 40},  // 10·4.00
	}
	for _, tc := range cases {
		if got := ItCostCurve(10, tc.w); got != tc.want {
			t.Errorf("cost curve at W=%v = %v, want %v", tc.w, got, tc.want)
		}
	}
}

// TestItAuthorityInitializationAndIncome locks §5.2: initial R_op 60, monthly
// income 8, cap 100; loyalty baselines 0.8.
func TestItAuthorityInitializationAndIncome(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("IT Fixture", 122, nil)
	InitializeItAuthority(silo)
	state := &silo.ItAuthority
	if state.ROp != 60 || state.BailiffLoyalty != 0.8 || state.ITInternalLoyalty != 0.8 {
		t.Fatalf("initial authority state mismatch: %+v", state)
	}
	// Exhaust resources: no action, income only.
	state.ROp = 0
	e.RunITAuthorityMonthly(silo)
	if state.ROp != 8 {
		t.Fatalf("depleted R_op must recover by the income 8, got %v", state.ROp)
	}
	if len(state.LastActions) != 0 {
		t.Fatalf("no action affordable at R_op=0: %+v", state.LastActions)
	}
}

// TestItCounterattackDecaysWithResources locks §5.5: with depleted resources
// only low-intensity actions remain affordable; expensive crackdowns vanish.
func TestItCounterattackDecaysWithResources(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("IT Fixture", 122, nil)
	InitializeItAuthority(silo)
	silo.ItAuthority.ROp = 9 // below the 12 threshold: low-intensity only
	e.RunITAuthorityMonthly(silo)
	for _, id := range silo.ItAuthority.LastActions {
		for _, action := range ItActionCatalog() {
			if action.ID != id {
				continue
			}
			if action.BaseCost > lowIntensityCost {
				t.Fatalf("resource-depleted IT ran a high-intensity action: %+v", action)
			}
		}
	}
}

// TestItMaintenanceMovesLoyaltyAndReverts locks §5.1: η responses raise
// loyalty; monthly reversion pulls the regular part toward the 0.8 baseline
// and decays fear-built loyalty faster.
func TestItMaintenanceMovesLoyaltyAndReverts(t *testing.T) {
	state := &model.ItAuthorityState{Initialized: true, BailiffLoyalty: 0.5, ITInternalLoyalty: 0.8}
	applyItMaintenance(state, "bailiff", MaintTruth)
	if want := 0.5 + etaTruth*(1-0.5); math.Abs(state.BailiffLoyalty-want) > 1e-12 {
		t.Fatalf("truth maintenance = %v, want %v", state.BailiffLoyalty, want)
	}
	applyItMaintenance(state, "bailiff", MaintFear)
	if want := etaFear; math.Abs(state.BailiffFear-want) > 1e-12 {
		t.Fatalf("fear maintenance = %v, want %v", state.BailiffFear, want)
	}

	// Reversion: regular drifts 3% toward baseline; fear decays 8%.
	before := state.BailiffLoyalty
	fearBefore := state.BailiffFear
	state.BailiffLoyalty = clamp01(state.BailiffLoyalty + regularReversion*(loyaltyBaseline-state.BailiffLoyalty))
	state.BailiffFear = clamp01(state.BailiffFear * (1 - fearReversion))
	if state.BailiffLoyalty <= before {
		t.Fatalf("regular loyalty below baseline must revert upward: %v -> %v", before, state.BailiffLoyalty)
	}
	if state.BailiffFear >= fearBefore {
		t.Fatalf("fear loyalty must decay: %v -> %v", fearBefore, state.BailiffFear)
	}
}

// seedForFirstRollBelow finds the first seed whose initial RNG draw is below p
// (deterministic scan, keeps tests reproducible).
func seedForFirstRollBelow(p float64) int64 {
	for seed := int64(1); seed < 100000; seed++ {
		if rand.New(rand.NewSource(seed)).Float64() < p {
			return seed
		}
	}
	return 1
}

// TestDispatchIsTheOnlyDirectLeak locks §5.3: a forced successful dispatch
// exposure produces Judicial-department and witness observations of the core
// dispatch facts — and nothing else raises exposure directly.
func TestDispatchIsTheOnlyDirectLeak(t *testing.T) {
	prev := rng
	SetRNG(rand.New(rand.NewSource(seedForFirstRollBelow(0.18))))
	defer SetRNG(prev)

	e := NewGameEngine()
	silo := CreateInitialSilo("Leak Fixture", 122, nil)
	silo.Residents = []model.Resident{
		{ID: 1, SiloID: silo.ID, Name: "Witness", Alive: true, Influence: 10},
	}
	e.exposeJudicialDispatch(silo, 0.18)

	if silo.ItAuthority.DispatchCount != 1 {
		t.Fatalf("dispatch count = %d, want 1", silo.ItAuthority.DispatchCount)
	}
	var judicial *model.Profession
	for i := range silo.Professions {
		if silo.Professions[i].Name == "Judicial" {
			judicial = &silo.Professions[i]
		}
	}
	if judicial == nil {
		t.Fatal("Judicial department missing")
	}
	if belief := BeliefIn(&judicial.Beliefs, "Judicial_1"); belief == nil || belief.ObsCount != 1 {
		t.Fatalf("judicial dispatch belief missing: %+v", belief)
	}
	if belief := BeliefIn(&silo.Residents[0].Beliefs, "Judicial_1"); belief == nil || belief.ObsCount != 1 {
		t.Fatalf("witness dispatch belief missing: %+v", belief)
	}
}

// TestProtocolDisposalProtectsAndRestores locks the §5.5 same-direction case:
// protocol disposal lowers safeguard risk and closes the earliest unmet target.
func TestProtocolDisposalProtectsAndRestores(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Disposal Fixture", 122, nil)
	silo.SafeguardRisk = 0.2
	silo.Protocol.Deviations = []model.ProtocolDeviation{
		{ClauseID: "O-02", Kind: DeviationUnmet, Open: true},
		{ClauseID: "O-01", Kind: DeviationUnmet, Open: true},
	}
	e.executeItAction(silo, ItAction{ID: "protocol_disposal", BaseCost: 7, WShock: 2, RSDelta: -0.04})
	if math.Abs(silo.SafeguardRisk-0.16) > 1e-12 {
		t.Fatalf("safeguard after protective disposal = %v, want 0.16", silo.SafeguardRisk)
	}
	for _, dev := range silo.Protocol.Deviations {
		if dev.ClauseID == "O-01" && dev.Open {
			t.Fatalf("earliest unmet (O-01) must be closed first: %+v", silo.Protocol.Deviations)
		}
		if dev.ClauseID == "O-02" && !dev.Open {
			t.Fatalf("later unmet (O-02) must stay open: %+v", silo.Protocol.Deviations)
		}
	}
}

// TestQuietDisappearanceConflictsWithProtocol locks the §5.5 conflict case:
// the disappearance action preserves prestige but records an unauthorized
// deviation (intervention risk rises even without public awareness).
func TestQuietDisappearanceConflictsWithProtocol(t *testing.T) {
	e := NewGameEngine()
	dir, err := filepathAbsProtocolDir()
	if err != nil {
		t.Fatalf("resolve protocol dir: %v", err)
	}
	clauses, version, err := LoadProtocolDir(dir)
	if err != nil {
		t.Fatalf("load clauses: %v", err)
	}
	e.ProtocolClauses = clauses
	e.ProtocolVersion = version

	silo := CreateInitialSilo("Disappearance Fixture", 122, nil)
	EnsureProtocolInitialized(silo, clauses, version)
	silo.Authority.Judicial.Regime = "SWING"

	e.executeItAction(silo, ItAction{ID: "quiet_disappearance", BaseCost: 12, WShock: 6, RegimeAverting: true})

	dev := findDeviation(&silo.Protocol, "P-03", DeviationViolation)
	if dev == nil || !dev.Open {
		t.Fatalf("unauthorized disappearance must record a violation: %+v", silo.Protocol.Deviations)
	}
	if dev.Acknowledged {
		t.Fatal("covert disappearance must not be coalition-acknowledged")
	}
	if silo.Protocol.J != 0.6 {
		t.Fatalf("J = %v, want 0.6", silo.Protocol.J)
	}
	// Utility must favor it while the Judicial is contested.
	silo.ItAuthority = model.ItAuthorityState{Initialized: true, ROp: 60, BailiffLoyalty: 0.8, ITInternalLoyalty: 0.8}
	score := itUtility(ItAction{ID: "quiet_disappearance", BaseCost: 12, WShock: 6, RegimeAverting: true}, silo, ItCostCurve(12, silo.Authority.W))
	if score <= 0 {
		t.Fatalf("contested Judicial must make disappearance utility-positive: %v", score)
	}
}

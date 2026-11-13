package engine

import (
	"math"
	"testing"

	"silo40/internal/model"
)

func TestAuthorityInitializationNamesMayorAndKeepsPactClosed(t *testing.T) {
	silo := CreateInitialSilo("Authority Fixture", 122, nil)
	if !silo.Mayor.Initialized || silo.Mayor.HolderResidentID == 0 || silo.Mayor.HolderName == "" {
		t.Fatalf("Mayor must be a named initialized office holder: %+v", silo.Mayor)
	}
	if !silo.Authority.Initialized || silo.Authority.W != 70 || silo.Authority.InitializationSource != "new_game" {
		t.Fatalf("new authority state mismatch: %+v", silo.Authority)
	}
	if silo.Authority.Judicial.PactQualified || silo.Authority.Judicial.PactLocked {
		t.Fatal("v8 content must not qualify or lock the pact channel")
	}
	assertAttachmentSum(t, silo)
}

func TestPoliceObedienceRespondsOnlyToMayorPersonalLegitimacy(t *testing.T) {
	silo := CreateInitialSilo("Police Fixture", 122, nil)
	silo.SetMayorLegitimacy(0.2)
	inputs := AuthorityInputs{BailiffLoyalty: 0.8, ITInternalLoyalty: 0.8}
	AdvanceAuthority(silo, inputs)
	want := 1 + DefaultAuthorityConfig().PoliceObedienceResponseRate*(0.2-1)
	if math.Abs(silo.Authority.PoliceObedience-want) > 1e-12 {
		t.Fatalf("first response = %v, want %v", silo.Authority.PoliceObedience, want)
	}

	// The compatibility mirror alone must not drive any settlement.
	silo.Legitimacy = 0.95
	before := silo.Authority.PoliceObedience
	AdvanceAuthority(silo, inputs)
	want = before + DefaultAuthorityConfig().PoliceObedienceResponseRate*(0.2-before)
	if math.Abs(silo.Authority.PoliceObedience-want) > 1e-12 {
		t.Fatalf("Police used Silo.Legitimacy instead of Mayor: got %v want %v", silo.Authority.PoliceObedience, want)
	}
}

func TestShadowAuthorityFormulaAndReservedOrderInput(t *testing.T) {
	silo := CreateInitialSilo("W Fixture", 122, nil)
	inputs := AuthorityInputs{BailiffLoyalty: 0.8, ITInternalLoyalty: 0.8, Exposure: 0.25, Delivery: 0.5, Personnel: -0.2, ExternalSignal: 0.1}
	AdvanceAuthority(silo, inputs)
	wEq := 50 + 25*0.8 + 12*0.5 + 10*(-0.2) + 12*0.1 - 40*0.25
	alpha := 1 - math.Pow(2, -1.0/8)
	want := 70 + alpha*(wEq-70)
	if math.Abs(silo.Authority.W-want) > 1e-12 {
		t.Fatalf("W = %.15f, want %.15f", silo.Authority.W, want)
	}

	inputs.OrderDeviation = 1
	before := silo.Authority.W
	AdvanceAuthority(silo, inputs)
	want = before + alpha*(wEq-before)
	if math.Abs(silo.Authority.W-want) > 1e-12 || silo.Authority.OrderDeviation != 1 {
		t.Fatalf("reserved Order input changed the W formula: W=%v want=%v", silo.Authority.W, want)
	}
}

func TestJudicialThresholdCrossingAndHysteresis(t *testing.T) {
	silo := CreateInitialSilo("Judicial Fixture", 122, nil)
	silo.SetMayorLegitimacy(0.8)
	cfg := DefaultAuthorityConfig()

	for month := 1; month <= 2; month++ {
		silo.Authority.W = 30
		advanceJudicialAttachment(silo, cfg)
		assertAttachmentSum(t, silo)
	}
	if silo.Authority.Judicial.Regime != "MAYOR" || silo.Authority.Judicial.Mayor < cfg.MayorAttachmentThreshold {
		t.Fatalf("two qualifying months must complete the institutional shift: %+v", silo.Authority.Judicial)
	}

	for _, w := range []float64{31, 44, 30.5, 44.9, 37} {
		silo.Authority.W = w
		advanceJudicialAttachment(silo, cfg)
		if silo.Authority.Judicial.Regime != "MAYOR" {
			t.Fatalf("hysteresis oscillated at W=%v: %+v", w, silo.Authority.Judicial)
		}
		assertAttachmentSum(t, silo)
	}

	for month := 1; month <= 2; month++ {
		silo.Authority.W = 45
		advanceJudicialAttachment(silo, cfg)
		if silo.Authority.Judicial.Regime != "MAYOR" {
			t.Fatalf("returned before three high months at month %d", month)
		}
	}
	silo.Authority.W = 45
	advanceJudicialAttachment(silo, cfg)
	if silo.Authority.Judicial.Regime != "IT" {
		t.Fatalf("three high months must return attachment to IT: %+v", silo.Authority.Judicial)
	}
}

func TestHighMayorLegitimacyCannotOpenLowWGate(t *testing.T) {
	silo := CreateInitialSilo("Mayor Gate Fixture", 122, nil)
	silo.SetMayorLegitimacy(1)
	cfg := DefaultAuthorityConfig()
	for month := 0; month < 12; month++ {
		silo.Authority.W = 50
		advanceJudicialAttachment(silo, cfg)
	}
	if silo.Authority.Judicial.Regime != "IT" || silo.Authority.Judicial.MayorLowMonths != 0 {
		t.Fatalf("Mayor legitimacy alone opened the low-W gate: %+v", silo.Authority.Judicial)
	}
}

func TestPactQualificationRemainsStructuralOnly(t *testing.T) {
	silo := CreateInitialSilo("Pact Fixture", 122, nil)
	silo.Authority.W = 10
	silo.Authority.Judicial.PublicEvidenceQ = 1
	silo.Authority.Judicial.IndependentPublicFacts = 3
	advanceJudicialAttachment(silo, DefaultAuthorityConfig())
	if silo.Authority.Judicial.PactLocked || silo.Authority.Judicial.TargetPact != 0 {
		t.Fatalf("unqualified pact channel acquired content effects: %+v", silo.Authority.Judicial)
	}
}

func assertAttachmentSum(t *testing.T, silo *model.Silo) {
	t.Helper()
	j := silo.Authority.Judicial
	if j.IT < 0 || j.Mayor < 0 || j.Pact < 0 || math.Abs(j.IT+j.Mayor+j.Pact-1) > 1e-12 {
		t.Fatalf("attachment weights invalid: %+v", j)
	}
}

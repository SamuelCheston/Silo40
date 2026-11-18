package engine

import (
	"math"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

func filepathAbsProtocolDir() (string, error) {
	return filepath.Abs("../../events/protocol")
}

func protocolFixture(t *testing.T) (*GameEngine, *model.Silo, []model.ProtocolClause) {
	t.Helper()
	dir, err := filepathAbsProtocolDir()
	if err != nil {
		t.Fatalf("resolve protocol dir: %v", err)
	}
	clauses, version, err := LoadProtocolDir(dir)
	if err != nil {
		t.Fatalf("load protocol clauses: %v", err)
	}
	if len(clauses) == 0 || version == 0 {
		t.Fatalf("protocol content missing: clauses=%d version=%d", len(clauses), version)
	}
	e := NewGameEngine()
	e.ProtocolClauses = clauses
	e.ProtocolVersion = version
	silo := CreateInitialSilo("Protocol Fixture", 122, nil)
	silo.BlackBox.Initialized = true // ADR-0019 功能开启语义：后果登记事件
	EnsureProtocolInitialized(silo, clauses, version)
	return e, silo, clauses
}

func advanceMonths(e *GameEngine, silo *model.Silo, months int) {
	for i := 0; i < months; i++ {
		AdvanceSiloClock(silo)
		e.AdvanceProtocolMonthly(silo)
	}
}

// TestProtocolLoadsVersionedClauses locks §6.1: the content file loads with a
// version and all five categories validate.
func TestProtocolLoadsVersionedClauses(t *testing.T) {
	_, silo, clauses := protocolFixture(t)
	categories := map[string]bool{}
	for _, clause := range clauses {
		categories[clause.Category] = true
	}
	for _, want := range []string{ProtocolProhibition, ProtocolRequiredResponse, ProtocolOutcomeTarget, ProtocolGap} {
		if !categories[want] {
			t.Fatalf("category %s missing from the protocol content", want)
		}
	}
	// believed copy starts identical to the true copy (no forgery yet).
	if len(silo.Protocol.BelievedClauses) != len(clauses) || silo.Protocol.BelievedVersion == 0 {
		t.Fatalf("believed copy not initialized: %+v", silo.Protocol)
	}
}

// TestViolationIgnoresIgnoranceAndForgery locks §6.2/§6.1: the true judgment
// reads the authoritative clauses — emptying or forging the believed copy
// changes nothing about the recorded deviation.
func TestViolationIgnoresIgnoranceAndForgery(t *testing.T) {
	e, silo, clauses := protocolFixture(t)

	silo.Protocol.BelievedClauses = nil // IT "knows" nothing
	RecordProtocolAction(silo, clauses, "unauthorized_cross_contact", true)

	dev := findDeviation(&silo.Protocol, "P-01", DeviationViolation)
	if dev == nil || !dev.Open || !dev.Acknowledged {
		t.Fatalf("ignorance must not change the true judgment: %+v", silo.Protocol.Deviations)
	}
	// ADR-0019：偏离不再直接累加风险度，改登记事件 (一期一次)。
	if silo.SafeguardRisk != 0 {
		t.Fatalf("violation must not accumulate into R_s: %v", silo.SafeguardRisk)
	}
	if silo.SafeguardIncidents[SafeguardIncidentDeviation] != 1 {
		t.Fatalf("major violation must register one deviation incident: %+v", silo.SafeguardIncidents)
	}
	if silo.Protocol.J != 0.6 {
		t.Fatalf("J = %v, want 0.6", silo.Protocol.J)
	}

	// Same clause twice: still one deviation record.
	RecordProtocolAction(silo, clauses, "unauthorized_cross_contact", true)
	if len(silo.Protocol.Deviations) != 1 {
		t.Fatalf("each clause records once per kind: %+v", silo.Protocol.Deviations)
	}

	// Forgery of the believed copy cannot erase or alter the true record.
	silo.Protocol.BelievedClauses = []model.ProtocolClause{}
	e.AdvanceProtocolMonthly(silo)
	if dev := findDeviation(&silo.Protocol, "P-01", DeviationViolation); dev == nil || !dev.Open {
		t.Fatal("forged believed copy must not touch the true deviation record")
	}
}

// TestExistentialViolationGoesStraightToReview locks §6.3: an existential
// violation enters the authoritative review immediately.
func TestExistentialViolationGoesStraightToReview(t *testing.T) {
	_, silo, clauses := protocolFixture(t)
	RecordProtocolAction(silo, clauses, "required_report_cleanup", true)
	if silo.Protocol.InterventionStage != InterventionAssessment {
		t.Fatalf("existential violation must enter assessment: %+v", silo.Protocol)
	}
	// ADR-0019：存在级违反同样只登记事件，不直接累加风险度。
	if silo.SafeguardRisk != 0 {
		t.Fatalf("existential violation must not accumulate into R_s: %v", silo.SafeguardRisk)
	}
	if silo.SafeguardIncidents[SafeguardIncidentDeviation] != 1 {
		t.Fatalf("existential violation must register one deviation incident: %+v", silo.SafeguardIncidents)
	}
}

// TestLapseDeadlineAndCompletion locks §6.2: an overdue required response
// becomes a lapse that worsens monthly until completed; completion closes it.
func TestLapseDeadlineAndCompletion(t *testing.T) {
	e, silo, clauses := protocolFixture(t)

	silo.EventTrigger = 1.5 // trips R-01's trigger metric
	e.AdvanceProtocolMonthly(silo)
	if len(silo.Protocol.Pending) != 1 {
		t.Fatalf("requirement not activated: %+v", silo.Protocol.Pending)
	}

	// Deadline is 2 months; months 1-2 pass without completion.
	advanceMonths(e, silo, 2)
	dev := findDeviation(&silo.Protocol, "R-01", DeviationLapse)
	if dev == nil || !dev.Open {
		t.Fatalf("overdue requirement must become an open lapse: %+v", silo.Protocol.Deviations)
	}
	if dev.MissedCount != 1 {
		t.Fatalf("missed months = %d, want 1", dev.MissedCount)
	}

	advanceMonths(e, silo, 1) // one more overdue month
	if dev.MissedCount != 2 {
		t.Fatalf("missed months = %d, want 2", dev.MissedCount)
	}

	CompleteProtocolRequirement(silo, clauses, "anomaly_report")
	if len(silo.Protocol.Pending) != 0 {
		t.Fatalf("completion must clear the pending requirement: %+v", silo.Protocol.Pending)
	}
	if dev := findDeviation(&silo.Protocol, "R-01", DeviationLapse); dev == nil || dev.Open {
		t.Fatalf("completion must close the lapse: %+v", silo.Protocol.Deviations)
	}
	if silo.Protocol.J != 0 {
		t.Fatalf("J must drop after closure: %v", silo.Protocol.J)
	}
}

// TestOutcomeReviewReadsTrueResults locks §6.2: outcome targets are read at
// review time from the true metrics; recovery closes the deviation.
func TestOutcomeReviewReadsTrueResults(t *testing.T) {
	e, silo, _ := protocolFixture(t)

	silo.Rebellion = 0.8           // above O-01's 0.6 cap
	e.AdvanceProtocolMonthly(silo) // anchor set
	advanceMonths(e, silo, 3)      // first review at +3 months

	dev := findDeviation(&silo.Protocol, "O-01", DeviationUnmet)
	if dev == nil || !dev.Open {
		t.Fatalf("out-of-range review must record unmet: %+v", silo.Protocol.Deviations)
	}
	if silo.Protocol.FailedReviews != 1 {
		t.Fatalf("failed reviews = %d, want 1", silo.Protocol.FailedReviews)
	}

	silo.Rebellion = 0.2 // recovered
	advanceMonths(e, silo, 3)
	if dev := findDeviation(&silo.Protocol, "O-01", DeviationUnmet); dev == nil || dev.Open {
		t.Fatalf("recovery must close the unmet deviation: %+v", silo.Protocol.Deviations)
	}
	if silo.Protocol.FailedReviews != 0 {
		t.Fatalf("failed reviews must reset on recovery: %d", silo.Protocol.FailedReviews)
	}
}

// TestInterventionChainIsDeterministicAndWarnable locks §6.3: warning at
// J>=1.0, correction at J>=1.5 with two failed major reviews, then override
// and assessment in fixed order; the control channel opens only inside the
// correction/override/assessment stages.
func TestInterventionChainIsDeterministicAndWarnable(t *testing.T) {
	e, silo, clauses := protocolFixture(t)

	// Stack deviations: one major violation (0.6), then let the trigger metric
	// and bad outcomes do the rest.
	RecordProtocolAction(silo, clauses, "unauthorized_cross_contact", true)
	silo.Rebellion = 0.9    // O-01 cap 0.6
	silo.Cohesion = 0.1     // O-02 floor 0.3
	silo.EventTrigger = 1.5 // R-01 trigger

	e.AdvanceProtocolMonthly(silo) // anchors set, R-01 activated; J=0.6
	if silo.Protocol.InterventionStage != InterventionNone {
		t.Fatalf("J=0.6 must stay without contact: %+v", silo.Protocol)
	}
	if len(silo.Protocol.Pending) != 1 {
		t.Fatalf("R-01 must be pending: %+v", silo.Protocol.Pending)
	}

	advanceMonths(e, silo, 2) // R-01 lapses: J = 0.75
	if findDeviation(&silo.Protocol, "R-01", DeviationLapse) == nil {
		t.Fatal("R-01 must lapse after its deadline")
	}
	if silo.Protocol.InterventionStage != InterventionNone {
		t.Fatalf("J=0.75 must stay without contact: %+v", silo.Protocol)
	}

	advanceMonths(e, silo, 1) // first reviews fail: J = 0.6+0.15+0.6+0.25 = 1.6
	if silo.Protocol.InterventionStage != InterventionWarning || !silo.Protocol.WarningIssued {
		t.Fatalf("J>=1.0 must issue the warning stage: %+v", silo.Protocol)
	}
	if silo.Protocol.ChannelOpen {
		t.Fatal("warning alone must not open the control channel")
	}

	advanceMonths(e, silo, 3) // second failed major review -> correction
	if silo.Protocol.FailedReviews < 2 {
		t.Fatalf("failed reviews = %d, want >= 2", silo.Protocol.FailedReviews)
	}
	if silo.Protocol.InterventionStage != InterventionCorrection {
		t.Fatalf("J>=1.5 with two failed reviews must enter correction: %+v", silo.Protocol)
	}
	if !silo.Protocol.ChannelOpen {
		t.Fatal("correction must open the control channel")
	}

	advanceMonths(e, silo, 2) // correction window expires -> override
	if silo.Protocol.InterventionStage != InterventionOverride {
		t.Fatalf("unrecovered correction must escalate to override: %+v", silo.Protocol)
	}
	if !silo.Protocol.ChannelOpen {
		t.Fatal("override must keep the control channel open")
	}

	advanceMonths(e, silo, 1) // override -> assessment
	if silo.Protocol.InterventionStage != InterventionAssessment {
		t.Fatalf("override must proceed to assessment: %+v", silo.Protocol)
	}

	advanceMonths(e, silo, 1) // assessment concludes -> no contact
	if silo.Protocol.InterventionStage != InterventionNone {
		t.Fatalf("assessment must conclude: %+v", silo.Protocol)
	}
	if silo.Protocol.ChannelOpen {
		t.Fatal("channel must close when the chain ends")
	}
}

// TestAuthorityShocksConsumeOnceWithCap locks §3.2: shocks apply once in the
// next monthly authority update, capped at ±12 per month.
func TestAuthorityShocksConsumeOnceWithCap(t *testing.T) {
	silo := CreateInitialSilo("Shock Fixture", 122, nil)
	InitializeAuthorityState(silo, false)

	QueueAuthorityShock(silo, -20) // capped at -12
	QueueAuthorityShock(silo, 3)   // total -17 -> capped -12
	before := silo.Authority.W
	AdvanceAuthority(silo, AuthorityInputs{BailiffLoyalty: 0.8, ITInternalLoyalty: 0.8})
	alpha := 1 - math.Pow(2, -1.0/8)
	wEq := 50 + 25*silo.Authority.K + 12*silo.Authority.D + 10*silo.Authority.P + 12*silo.Authority.X - 40*silo.Authority.E
	want := clamp(before+alpha*(wEq-before), 0, 100) - 12
	if math.Abs(silo.Authority.W-want) > 1e-9 {
		t.Fatalf("W after capped shocks = %v, want %v", silo.Authority.W, want)
	}
	if len(silo.Authority.Shocks) != 0 {
		t.Fatalf("shock queue must be consumed once: %+v", silo.Authority.Shocks)
	}
}

// TestGeneratorsDeduplicateAndRespectChannelLocks locks §6.5: each relic/
// reflection/utterance yields one observation; the control channel only
// produces utterances inside the intervention chain; G aggregates and tiers.
func TestGeneratorsDeduplicateAndRespectChannelLocks(t *testing.T) {
	_, silo, _ := protocolFixture(t)

	if silo.Protocol.G != 0 || GCognitionTier(silo.Protocol.G) != "apprentice_unaware" {
		t.Fatalf("director must start unaware: G=%v", silo.Protocol.G)
	}

	if !InspectRelicForDirector(silo, "relic_alpha", "decode") {
		t.Fatal("first relic inspection must absorb")
	}
	gAfterRelic := silo.Protocol.G
	if InspectRelicForDirector(silo, "relic_alpha", "decode") {
		t.Fatal("same relic x method must not produce twice")
	}
	if silo.Protocol.G != gAfterRelic {
		t.Fatal("duplicate relic inspection must not move G")
	}
	if !InspectRelicForDirector(silo, "relic_alpha", "catalog") {
		t.Fatal("a different inspection method must still produce")
	}

	if !ReflectOnProtocol(silo, "incident_122_03") {
		t.Fatal("first reflection must absorb")
	}
	if ReflectOnProtocol(silo, "incident_122_03") {
		t.Fatal("same reflection_key must not produce twice")
	}

	// Channel closed: no utterance.
	if ReceiveChannelUtterance(silo, "utterance_1") {
		t.Fatal("control channel must stay silent outside the intervention chain")
	}
	silo.Protocol.ChannelOpen = true
	if !ReceiveChannelUtterance(silo, "utterance_1") {
		t.Fatal("open channel must deliver an utterance")
	}
	if ReceiveChannelUtterance(silo, "utterance_1") {
		t.Fatal("same utterance_id must not produce twice")
	}

	if silo.Protocol.G <= gAfterRelic {
		t.Fatalf("G must grow with distinct observations: %v -> %v", gAfterRelic, silo.Protocol.G)
	}
	if GCognitionTier(silo.Protocol.G) == "apprentice_unaware" {
		t.Fatalf("G tier must advance past unaware: %v", silo.Protocol.G)
	}
}

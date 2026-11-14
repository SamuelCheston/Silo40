package service

import (
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestInformationEndingNarrativeReplacesFormula locks 25-information-endgame
// §2: the INFORMATION settlement shows the story, not the P2 score formula,
// and the spillover settlement line reports sent/reached counts (§3).
func TestInformationEndingNarrativeReplacesFormula(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		station.Professions = append(station.Professions, prof)
	}
	// Two outbound signals reaching one other silo.
	svc.world.OutboundSignals = []model.OutboundSignal{
		{Seq: 1, SourceSilo: 1, TargetSilo: 2, EventID: "x", Year: 128, Month: 1},
		{Seq: 2, SourceSilo: 1, TargetSilo: 2, EventID: "y", Year: 128, Month: 2},
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingInformation {
		t.Fatalf("expected INFORMATION selection, got %+v", decision)
	}
	if strings.Contains(decision.Reason, "P2 加权") {
		t.Fatalf("score formula still on the ending screen: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "《共证》成立") {
		t.Fatalf("narrative missing: %q", decision.Reason)
	}
	if !strings.Contains(decision.Reason, "共有 2 条信号送出驻地，到达 1 座他堡") {
		t.Fatalf("spillover settlement line missing: %q", decision.Reason)
	}
}

// TestInformationEndingNarrativeZeroSpillover locks the zero-spillover line:
// no signals sent is stated plainly and never reads as a failure.
func TestInformationEndingNarrativeZeroSpillover(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		station.Professions = append(station.Professions, prof)
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingInformation {
		t.Fatalf("expected INFORMATION selection, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "外溢为零，不影响本次裁决") {
		t.Fatalf("zero-spillover settlement line missing: %q", decision.Reason)
	}
}

// TestInformationVigilEveFires drives the convergence beat end to end on the
// real bus wiring: P-1 lit earlier, final year reached, rebellion under the
// reconnaissance feed threshold -> the vigil node lights once.
func TestInformationVigilEveFires(t *testing.T) {
	boot := bootstrapRealContent(t)

	var vigilDef engine.ContentEventDefinition
	var found bool
	for _, def := range boot.contentDefinitions {
		if def.Key == "special:info_vigil_eve" {
			vigilDef = def
			found = true
		}
	}
	if !found {
		t.Fatal("special:info_vigil_eve did not load from events/special/")
	}

	svc := chainTestService(vigilDef)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 129, 2
	svc.silo.Rebellion = 0.2
	svc.statesMapFor(svc.silo.ID)["special:public_fact_cross_dept"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:public_fact_cross_dept", Triggered: true,
	}

	if svc.queueSystemContentEvents("TEST") == 0 {
		t.Fatal("vigil node was not queued in the final year")
	}
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:info_vigil_eve") {
		t.Fatal("info_vigil_eve never fired")
	}
}

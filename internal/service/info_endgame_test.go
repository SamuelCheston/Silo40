package service

import (
	"path/filepath"
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
		// Re-anchored gate (belief layer): seed facts above the sharing floor.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
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
		// Re-anchored gate (belief layer): seed facts above the sharing floor.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
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

// vigilFixtureService builds the vigil scenario: P-1 lit, rebellion calm,
// clock set before the final year so only the foreshadow window can fire.
func vigilFixtureService(t *testing.T) *GameService {
	t.Helper()
	boot := bootstrapRealContent(t)
	var vigilDef engine.ContentEventDefinition
	for _, def := range boot.contentDefinitions {
		if def.Key == "special:info_vigil_eve" {
			vigilDef = def
		}
	}
	if vigilDef.Key == "" {
		t.Fatal("special:info_vigil_eve did not load from events/special/")
	}
	svc := chainTestService(vigilDef)
	svc.silo.CurrentYear, svc.silo.CurrentMonth = 127, 1
	svc.silo.Rebellion = 0.2
	svc.statesMapFor(svc.silo.ID)["special:public_fact_cross_dept"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "special:public_fact_cross_dept", Triggered: true,
	}
	return svc
}

// TestInformationVigilForeshadowsNearGate locks 修复4: the ending settles the
// month the belief gate is met, so the old final-year-only vigil was pure
// foreshadow that fast playthroughs never saw. With coverage within one
// department of the gate the vigil lights early; far from the gate it stays
// dark until the final year.
func TestInformationVigilForeshadowsNearGate(t *testing.T) {
	svc := vigilFixtureService(t)
	// One department complete, one missing two facts: within the 3-fact gap.
	svc.silo.Professions = []model.Profession{
		{Name: "IT", Beliefs: []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}},
		{Name: "Judicial", Beliefs: []model.FactBelief{{FactID: "d", B: 0.5}}},
	}
	if svc.queueSystemContentEvents("TEST") == 0 {
		t.Fatal("vigil did not foreshadow while coverage was within one department of the gate")
	}
	svc.drainQueuedEvents()
	if !contentTriggered(svc, "special:info_vigil_eve") {
		t.Fatal("info_vigil_eve never fired in the foreshadow window")
	}
}

func TestInformationVigilStaysDarkFarFromGate(t *testing.T) {
	svc := vigilFixtureService(t)
	// Three departments with nothing above the floor: 9 facts missing > gap 3.
	svc.silo.Professions = []model.Profession{
		{Name: "IT", Beliefs: []model.FactBelief{{FactID: "a", B: 0.1}}},
		{Name: "Judicial"},
		{Name: "Medical"},
	}
	if svc.queueSystemContentEvents("TEST") != 0 {
		t.Fatal("vigil fired long before the gate was in sight")
	}
	svc.drainQueuedEvents()
	if contentTriggered(svc, "special:info_vigil_eve") {
		t.Fatal("info_vigil_eve fired far from the gate")
	}
}

// TestInformationProgressExposedInState locks 修复3: the 《共证》 chain state
// (node lights, replication count, department coverage, gate constants) is a
// first-class part of every state snapshot, so the player can see 复制进展 /
// 活链成立 instead of guessing from event history.
func TestInformationProgressExposedInState(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "information-progress.db"))
	req := threeSiloRequest(2402)
	req.Profession = "IT"
	state, err := svc.CreateGame(req)
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	progress := state.InformationProgress
	if progress == nil {
		t.Fatal("game state lacks information_progress")
	}
	if len(progress.Nodes) != 6 {
		t.Fatalf("expected the 6 《共证》 chain nodes, got %d", len(progress.Nodes))
	}
	if progress.Nodes[0].Title != "陪同回查" || progress.Nodes[1].Title != "复制推进" {
		t.Fatalf("chain node titles out of order: %+v", progress.Nodes)
	}
	for _, node := range progress.Nodes {
		if node.Lit {
			t.Fatalf("node %s lit on a fresh game", node.Key)
		}
	}
	if progress.GateFactCount != informationGateFactCount {
		t.Fatalf("gate fact count mismatch: %d vs %d", progress.GateFactCount, informationGateFactCount)
	}
	if progress.BeliefFloor != engine.DefaultInformationConfig().QualRumor {
		t.Fatalf("belief floor mismatch: %v", progress.BeliefFloor)
	}

	// Open the line: one gather + one share lights the replication heartbeat.
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil || !outcome.Result.Executed {
			t.Fatalf("execute %+v: %+v (%v)", action, outcome.Result, err)
		}
	}
	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Judicial"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Judicial",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	after, err := svc.GetState()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	progress = after.InformationProgress
	if progress == nil {
		t.Fatal("state after actions lacks information_progress")
	}
	var replication *model.InformationChainNode
	for i := range progress.Nodes {
		if progress.Nodes[i].Key == "special:replication_progress" {
			replication = &progress.Nodes[i]
		}
	}
	if replication == nil || !replication.Lit || replication.Count < 1 {
		t.Fatalf("replication heartbeat not visible after the first share: %+v", replication)
	}
	judicialCovered := false
	for _, dept := range progress.Departments {
		if dept.Name == "Judicial" && dept.AboveFloor >= 1 {
			judicialCovered = true
		}
	}
	if !judicialCovered {
		t.Fatalf("department coverage does not reflect the shared belief: %+v", progress.Departments)
	}
}

func TestInformationEndingPlayableFromNewGame(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "information-playable.db"))
	req := threeSiloRequest(2401)
	req.Profession = "Judicial"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v, known=%v, beliefs=%+v)", action, err, svc.agent.ActionPoints, svc.agent.KnownFragments, svc.agent.Beliefs)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v, known=%v, beliefs=%+v)", action, outcome.Result, svc.agent.ActionPoints, svc.agent.KnownFragments, svc.agent.Beliefs)
		}
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("real Gather action did not open the line: %s", svc.world.ProgressStage)
	}
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// Let the real bailiff observation/publication chain run, then let the next
	// natural monthly authority update compute Q from the registered evidence.
	for month := 0; month < 12 && svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bailiff public evidence at month %d: %v", month, err)
		}
	}
	publishFloor := engine.DefaultInformationConfig().QualPublish
	judicialBelief := engine.BeliefFor(svc.agent, "Judicial_2")
	if judicialBelief == nil || judicialBelief.B < publishFloor {
		t.Fatalf("real actions did not reach the Judicial_2 publish floor %.2f: belief=%+v, known=%v, AP=%v", publishFloor, judicialBelief, svc.agent.KnownFragments, svc.agent.ActionPoints)
	}
	for _, key := range []string{"special:bailiff_dispatch_evidence_b", "special:bailiff_evidence_public_b"} {
		if !contentTriggered(svc, key) {
			t.Fatalf("%s did not fire through real monthly progression (belief=%+v, evidence=%+v)", key, judicialBelief, svc.silo.PublicEvidence)
		}
	}
	if svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 {
		t.Fatalf("natural authority updates did not produce Q from bailiff evidence: Q=%v, evidence=%+v", svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
	}

	var receiver *model.PlayerActionMeta
	for i := 0; i < 8; i++ {
		receiver = actionByID(svc.availablePlayerActions(), "RECEIVER_VERIFY_ACCOMPANY")
		if receiver != nil && receiver.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("recover AP for receiver verification: %v", err)
		}
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	// 修复1 回归：真实内容下执行回执必须复述事件故事，不能退回系统黑话。
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("receiver verification failed: %+v (%v)", outcome.Result, err)
	}
	if !strings.Contains(outcome.Result.Message, "陪同回查") {
		t.Fatalf("receiver verification receipt lost its story: %q", outcome.Result.Message)
	}

	circulationActions := 0
	attemptCirculation := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("circulate with %+v: %v", action, err)
		}
		if outcome.Result.Executed {
			circulationActions++
		}
	}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}

		// INFORMATION now requires circulation to outrun belief forgetting.
		// Keep sharing the strongest missing fact with the neediest department,
		// gathering across departments when no useful fact can be shared.
		ap := svc.agent.ActionPoints
		shareCost := model.ACTION_COSTS[model.ActionShareInfo]
		gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
		buildCost := model.ACTION_COSTS[model.ActionBuildConnection]
		gatherFrom := rotation[month%len(rotation)]
		switch {
		case svc.silo.Rebellion >= 0.7:
			// Lie low while rebellion runs hot.
		case ap >= shareCost && month >= 4:
			dept := neediestDepartment(svc.silo)
			fragment, ok := strongestShareableFactFor(svc.agent, departmentByAddress(svc.silo, dept))
			if ok {
				attemptCirculation(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept,
					FragmentIds: []string{fragment},
				})
			} else if ap >= gatherCost {
				attemptCirculation(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
			}
		case ap >= buildCost && month < 12:
			attemptCirculation(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
		case ap >= gatherCost:
			attemptCirculation(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
		}
	}
	if circulationActions == 0 {
		t.Fatal("playable line reached its ending without any sustained circulation action")
	}
	for _, key := range []string{
		"player_actions:receiver_verify_accompany",
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the playable line (Q=%v, evidence=%+v)", key, svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:replication_progress"].TriggerCount; count < 4 {
		t.Errorf("replication heartbeat fired %d times, want at least 4", count)
	}
	for _, factID := range []string{
		"Judicial_2",
		"fact:info_chain_replication",
		"fact:receiver_verification",
		"fact:independent_receivers",
	} {
		found := false
		for _, record := range svc.silo.PublicEvidence {
			if record.FactID == factID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("public evidence %s was not produced by the playable line", factID)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingInformation {
		t.Fatalf("playable line did not settle as INFORMATION at %d-%02d: status=%+v fragments=%+v evidence=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus, svc.silo.Professions, svc.silo.PublicEvidence)
	}

	t.Logf("real INFORMATION route: Judicial_2 B=%.6f observations=%d Q=%.6f at %d-%02d", judicialBelief.B, judicialBelief.ObsCount, svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.CurrentYear, svc.silo.CurrentMonth)

	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal INFORMATION state accepted another action: %+v", outcome.Result)
	}
}

package service

import (
	"path/filepath"
	"sort"
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
		// Gate reads permeation maturity: seed strong beliefs, then let the
		// real monthly dispatch (forgetting + permeation) mature them.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: permeationSeedB}, {FactID: "b", B: permeationSeedB}, {FactID: "c", B: permeationSeedB},
		}
		station.Professions = append(station.Professions, prof)
	}
	matureStationBeliefsThroughRealMonths(station)
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
		// Gate reads permeation maturity: seed strong beliefs, then let the
		// real monthly dispatch (forgetting + permeation) mature them.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: permeationSeedB}, {FactID: "b", B: permeationSeedB}, {FactID: "c", B: permeationSeedB},
		}
		station.Professions = append(station.Professions, prof)
	}
	matureStationBeliefsThroughRealMonths(station)

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
	// The gate reads maturity, so the fixture matures the seeded beliefs
	// through the real monthly dispatch (forgetting + permeation).
	svc.silo.Professions = []model.Profession{
		{Name: "IT", Beliefs: []model.FactBelief{
			{FactID: "a", B: permeationSeedB}, {FactID: "b", B: permeationSeedB}, {FactID: "c", B: permeationSeedB},
		}},
		{Name: "Judicial", Beliefs: []model.FactBelief{{FactID: "d", B: permeationSeedB}}},
	}
	matureStationBeliefsThroughRealMonths(svc.silo)
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
			t.Fatalf("execute %+v rejected: %+v (AP=%v, verified=%v, evidence=%+v, beliefs=%+v)", action, outcome.Result, svc.agent.ActionPoints, svc.silo.VerifiedEvidence, svc.silo.PublicEvidence, svc.agent.Beliefs)
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

	// Let the real bailiff observation chain run until the player's belief in
	// Judicial_2 reaches the publish floor.
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
	if !contentTriggered(svc, "special:bailiff_dispatch_evidence_b") {
		t.Fatalf("special:bailiff_dispatch_evidence_b did not fire through real monthly progression (belief=%+v)", judicialBelief)
	}

	// Publication is a player decision (sub-item 6 replaced the automatic
	// scaffold): verify then publish until BOTH core facts are registered,
	// letting the next natural monthly authority update compute Q from the
	// evidence. Permeation retune (18-time-permeation §7): a publication is the
	// only absorption strong enough to seed every department at once, so both
	// publishable core facts go out before the circulation window — this is a
	// re-ordering of actions the route already uses, not a new action.
	verifyReportFloor := engine.DefaultInformationConfig().QualReport
	publishedFacts := func() map[string]bool {
		out := map[string]bool{}
		for _, record := range svc.silo.PublicEvidence {
			out[record.FactID] = true
		}
		return out
	}
	for i := 0; i < 14; i++ {
		published := publishedFacts()
		if published["Judicial_1"] && published["Judicial_2"] {
			break
		}
		verifiable, publishable := false, false
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			belief := engine.BeliefFor(svc.agent, fact)
			if belief == nil {
				continue
			}
			verified := false
			for _, v := range svc.silo.VerifiedEvidence {
				if v == fact {
					verified = true
				}
			}
			if belief.B >= verifyReportFloor && !verified {
				verifiable = true
			}
			if belief.B >= publishFloor && verified && !published[fact] {
				publishable = true
			}
		}
		if verifiable && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence"})
		}
		if publishable && svc.agent.ActionPoints >= 20 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence"})
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance publication at step %d: %v", i, err)
		}
		// The longer publication window can outlast the reset chain's fuse on
		// its own; answer the termination-chain omen here exactly as the
		// circulation loop does (same helper, no new action).
		progressResetCountermeasureForRoute(t, svc, execute)
	}
	if !contentTriggered(svc, "player_action:publish_evidence") {
		t.Fatalf("player publication never completed (belief=%+v, verified=%v, evidence=%+v)", judicialBelief, svc.silo.VerifiedEvidence, svc.silo.PublicEvidence)
	}
	if svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 {
		t.Fatalf("natural authority updates did not produce Q from bailiff evidence: Q=%v, evidence=%+v", svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
	}

	if svc.gameOver() {
		t.Fatalf("game ended early at %d-%02d during publication: %+v reason=%q",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus.Type, decisionReasonOf(svc.silo.VictoryStatus))
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
	if err != nil {
		t.Fatalf("receiver verification errored at %d-%02d: %v (AP=%v)", svc.silo.CurrentYear, svc.silo.CurrentMonth, err, svc.agent.ActionPoints)
	}
	if !outcome.Result.Executed {
		t.Fatalf("receiver verification failed: %+v", outcome.Result)
	}
	if !strings.Contains(outcome.Result.Message, "陪同回查") {
		t.Fatalf("receiver verification receipt lost its story: %q", outcome.Result.Message)
	}

	circulationActions := 0
	attemptCirculation := func(action model.AgentAction) bool {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			// 流通摩擦 (T1-2) 会让目标部门更贵；AP 不够时让路线等到下个月
			// 的自然回复，而不是当作路线断裂。
			if strings.Contains(err.Error(), "Action Points") {
				return false
			}
			t.Fatalf("circulate with %+v: %v", action, err)
		}
		if outcome.Result.Executed {
			circulationActions++
		}
		return outcome.Result.Executed
	}
	// Per-pair share cap (same anti-deadlock as the §7.4 calibration): a fact
	// that does not absorb past the floor for one department (repeat-penalty
	// stall) must not be re-shared forever while other departments wait for a
	// third fact — the loop gathers instead and grows the portfolio.
	shareAttempts := map[string]int{}
	// Permeation upkeep cap: a running streak only needs re-circulation when
	// forgetting threatens to break it before maturity (18-time-permeation
	// §3.3/§5). The cap stays an anti-deadlock bound, never a play target.
	upkeepAttempts := map[string]int{}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)

		// Permeation-aware circulation (18-time-permeation §7 fair retune):
		// the gate now reads matured facts, so the strategy front-loads the
		// opening AP into circulation (every share needs a full permeation
		// window before it counts) and spends upkeep shares to keep running
		// streaks alive against forgetting until they mature. No new actions,
		// no cost changes, loop cap unchanged (criterion 3/4).
		shareCost := model.ACTION_COSTS[model.ActionShareInfo]
		gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
		// 流通摩擦 (T1-2) 让戒备 ≥ 0.30 的部门更贵；路线按服务端权威成本决定
		// 此刻付不付得起，避免把一整个月浪费在必被拒的尝试上。
		shareCostFor := func(dept string) float64 {
			action := model.AgentAction{Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{"probe"}}
			if err := svc.resolveActionCost(&action); err != nil {
				t.Fatalf("resolve share cost: %v", err)
			}
			return action.Cost
		}
		buildCostFor := func(dept string) float64 {
			action := model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept}
			if err := svc.resolveActionCost(&action); err != nil {
				t.Fatalf("resolve build cost: %v", err)
			}
			return action.Cost
		}
		gatherFrom := rotation[month%len(rotation)]
		if svc.silo.Rebellion >= 0.7 {
			// Lie low while rebellion runs hot.
			continue
		}
		circulated := false
		for svc.agent.ActionPoints >= shareCost {
			// Neediness reads the gate: fewest matured first, then fewest
			// healthy slots (a department mid-permeation is needier than one
			// that already holds three running streaks).
			order := append([]model.Profession(nil), svc.silo.Professions...)
			sort.SliceStable(order, func(i, j int) bool {
				mi, mj := informationMaturedFacts(&order[i]), informationMaturedFacts(&order[j])
				if mi != mj {
					return mi < mj
				}
				return informationActiveSlots(&order[i]) < informationActiveSlots(&order[j])
			})
			shared := false
			for i := range order {
				dept := &order[i]
				if informationMaturedFacts(dept) >= informationGateFactCount {
					continue
				}
				// Upkeep FIRST: a running streak about to break is months of
				// invested calendar — protecting it outranks planting a new
				// slot (18-time-permeation §3.3 第 2 条: 淡忘是渗透的天敌).
				if fragment, ok := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts); ok &&
					svc.agent.ActionPoints >= shareCostFor(dept.Name) {
					upkeepAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						break
					}
				}
				// Three healthy slots (matured or mid-permeation): stop
				// investing here — extra facts would only dilute the AP the
				// needier departments still need for their first seeds.
				if informationActiveSlots(dept) >= informationGateFactCount {
					continue
				}
				// Build the pipe before seeding: a zero-connection share
				// absorbs ≈ the floor itself and dies within months, so trust
				// is invested first while the seeding window is still open
				// (the share action's own contract: 没有信任的部门接不住事实).
				if month <= 30 && svc.pendingOperation == nil &&
					agentConnectionTo(svc, dept) < 0.3 && svc.agent.ActionPoints >= buildCostFor(dept.Name) {
					if attemptCirculation(model.AgentAction{
						Type: model.ActionBuildConnection, TargetDept: dept.Name,
					}) {
						shared = true
						break
					}
				}
				// New coverage: the strongest fact the department does not yet
				// hold above the floor, skipping capped pairs. The cap stays an
				// anti-deadlock bound (raised so a broken streak can be
				// re-seeded once), never a play target.
				if fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					return shareAttempts[dept.Name+":"+fact] >= 4
				}); ok {
					// A seed only takes if the agent's own conviction is strong
					// enough to absorb past the floor at this connection —
					// otherwise gather the source department first to deepen it
					// (a sub-floor seed just dies and wastes the slot).
					if agentBelief := engine.BeliefFor(svc.agent, fragment); agentBelief != nil && agentBelief.B < 0.80 &&
						svc.agent.ActionPoints >= gatherCost {
						if attemptCirculation(model.AgentAction{
							Type: model.ActionGatherInfo, TargetDept: fragmentDepartment(fragment),
						}) {
							shared = true
							break
						}
					}
					shareAction := model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}
					if err := svc.resolveActionCost(&shareAction); err != nil {
						t.Fatalf("resolve share cost: %v", err)
					}
					if svc.agent.ActionPoints < shareAction.Cost {
						// 戒备上浮后暂时付不起这一格：换下一个部门试，
						// 别把整个月的剩余 AP 一起放弃。
						continue
					}
					shareAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(shareAction) {
						shared = true
						break
					}
				}
			}
			if !shared {
				break
			}
			circulated = true
		}
		if circulated {
			continue
		}
		switch {
		case svc.agent.ActionPoints >= buildCostFor("Judicial") && month < 12:
			attemptCirculation(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
		case svc.agent.ActionPoints >= gatherCost:
			// Nothing shareable: gather — the portfolio must keep growing, a
			// stronger fourth fact is what the hardest department (zero-
			// population Mayor) needs for its third matured belief.
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

// informationHeldFacts counts a department's facts at or above the sharing
// floor (QualRumor) — the secondary neediness key for the circulation loop.
func informationHeldFacts(prof *model.Profession) int {
	count := 0
	for _, belief := range prof.Beliefs {
		if belief.B >= engine.DefaultInformationConfig().QualRumor {
			count++
		}
	}
	return count
}

// informationMaturedFacts counts a department's matured facts — the quantity
// the permeation gate reads, and the primary neediness ordering key.
func informationMaturedFacts(prof *model.Profession) int {
	return engine.MaturedFactCount(prof)
}

// endangeredStreakCandidate picks the department's running streak that is
// about to break: B within the danger margin above the floor (≈6 months of
// forgetting left). Among endangered streaks the highest counter wins — the
// most invested calendar is protected first (18-time-permeation §3.3 第 2 条).
// Only facts with a running counter qualify; a reset streak restarts through a
// fresh share instead. The attempts map caps re-shares per pair.
func endangeredStreakCandidate(agent *model.Agent, dept *model.Profession, attempts map[string]int) (string, bool) {
	cfg := engine.DefaultInformationConfig()
	// Re-circulation carries the same-source repeat penalty (independence 0.25),
	// so one upkeep share only lifts B ≈0.05 — upkeep must start while the
	// streak still has a real buffer, and the cap must cover a full window.
	const maxUpkeepPerPair = 12
	dangerFloor := cfg.QualRumor + 0.12
	best := ""
	bestCounter := -1
	bestB := 0.0
	for _, belief := range dept.Beliefs {
		if belief.Matured || belief.PermeationCounter <= 0 || belief.B < cfg.QualRumor || belief.B >= dangerFloor {
			continue
		}
		if attempts[dept.Name+":"+belief.FactID] >= maxUpkeepPerPair {
			continue
		}
		agentBelief := engine.BeliefFor(agent, belief.FactID)
		if agentBelief == nil || agentBelief.B < cfg.QualRumor {
			continue
		}
		if best == "" || belief.PermeationCounter > bestCounter ||
			(belief.PermeationCounter == bestCounter && belief.B < bestB) {
			best, bestCounter, bestB = belief.FactID, belief.PermeationCounter, belief.B
		}
	}
	return best, best != ""
}

// decisionReasonOf extracts the arbitration reason from a victory status for
// diagnostics (empty when no decision was recorded).
func decisionReasonOf(status *model.VictoryStatus) string {
	if status == nil || status.Decision == nil {
		return ""
	}
	return status.Decision.Reason
}

// fragmentDepartment derives the owning department from a fragment ID
// ("Mayor_1" -> "Mayor"), matching the catalog prefix convention.
func fragmentDepartment(fragment string) string {
	if idx := strings.Index(fragment, "_"); idx > 0 {
		return fragment[:idx]
	}
	return fragment
}

// agentConnectionTo reads the agent's connection value for a department —
// the transmit side of share absorption.
func agentConnectionTo(svc *GameService, dept *model.Profession) float64 {
	for _, connection := range svc.agent.Connections {
		if connection.ProfessionID == dept.ID {
			return connection.Value
		}
	}
	return 0
}

// informationActiveSlots counts a department's healthy gate slots: matured
// facts plus running streaks still above the floor. Three healthy slots mean
// the department needs protection, not more seeds.
func informationActiveSlots(dept *model.Profession) int {
	cfg := engine.DefaultInformationConfig()
	count := 0
	for _, belief := range dept.Beliefs {
		if belief.Matured || (belief.PermeationCounter > 0 && belief.B >= cfg.QualRumor) {
			count++
		}
	}
	return count
}

// strongestShareableFactExcluding is strongestShareableFactFor with an
// exclusion predicate on the fact ID (used to skip capped pairs).
func strongestShareableFactExcluding(agent *model.Agent, dept *model.Profession, skip func(string) bool) (string, bool) {
	held := map[string]bool{}
	if dept != nil {
		for _, belief := range dept.Beliefs {
			if belief.B >= engine.DefaultInformationConfig().QualRumor {
				held[belief.FactID] = true
			}
		}
	}
	best := ""
	bestB := 0.0
	for _, belief := range agent.Beliefs {
		if belief.B < engine.DefaultInformationConfig().QualRumor || held[belief.FactID] || (skip != nil && skip(belief.FactID)) {
			continue
		}
		if belief.B > bestB || (belief.B == bestB && belief.FactID < best) {
			best = belief.FactID
			bestB = belief.B
		}
	}
	return best, best != ""
}

// TestInformationProgressExposesTerminationChain locks the 开局自解释目标卡
// 判据三读数: the termination-chain omen and the established countermeasure
// are facts already present in content state; informationProgress only reads
// them through (no RNG), so the mission card can answer "终止链现在怎样".
func TestInformationProgressExposesTerminationChain(t *testing.T) {
	svc := chainTestService()
	progress := svc.informationProgress()
	if progress == nil {
		t.Fatal("informationProgress returned nil")
	}
	if progress.ResetOmenSeen || progress.ResetCountermeasureDone {
		t.Fatalf("fresh game must not report termination-chain progress: %+v", progress)
	}
	svc.statesMapFor(svc.silo.ID)["crisis:reset_omen_requisition"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "crisis:reset_omen_requisition", Triggered: true,
	}
	progress = svc.informationProgress()
	if !progress.ResetOmenSeen || progress.ResetCountermeasureDone {
		t.Fatalf("omen must read before the countermeasure: %+v", progress)
	}
	svc.statesMapFor(svc.silo.ID)["player_action:reset_countermeasure_done"] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: "player_action:reset_countermeasure_done", Triggered: true,
	}
	progress = svc.informationProgress()
	if !progress.ResetOmenSeen || !progress.ResetCountermeasureDone {
		t.Fatalf("countermeasure completion must read: %+v", progress)
	}
}

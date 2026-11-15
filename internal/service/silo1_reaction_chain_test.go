package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// reactionChainKeys lists the Silo 1 post-ultimatum reaction chain, grouped by
// independent enforcement channel (本地执行 / 远程终止 / 远程物理打击).
var reactionChainKeys = []string{
	// Local execution channel.
	"special:silo1_purge_omen",
	"special:silo1_purge_exposed",
	"crisis:silo1_purge_execution",
	// Remote termination channel.
	"special:termination_channel_test",
	"special:termination_channel_disrupted",
	"crisis:termination_release",
	// Remote physical strike channel.
	"special:strike_reconnaissance",
	"crisis:bombing_terminal",
}

func findContentDefinitionByKey(t *testing.T, svc *GameService, key string) engine.ContentEventDefinition {
	t.Helper()
	for _, def := range svc.contentDefinitions {
		if def.Key == key {
			return def
		}
	}
	t.Fatalf("content event %q was not loaded", key)
	return engine.ContentEventDefinition{}
}

// TestSilo1ReactionChainEventsLoad locks that every reaction-chain node loads
// and passes content validation (a malformed trigger/effect would fail boot).
func TestSilo1ReactionChainEventsLoad(t *testing.T) {
	svc := bootstrapRealContent(t)
	have := map[string]bool{}
	for _, def := range svc.contentDefinitions {
		have[def.Key] = true
	}
	for _, key := range reactionChainKeys {
		if !have[key] {
			t.Fatalf("reaction chain event %q not loaded", key)
		}
	}
}

// TestReactionSettlementsAreNonNumericalHooks locks the ruling that settlements
// write no population figures and no safeguard-risk deltas: all three settlement
// nodes are apply-hooks with no population/risk effects. The remote-termination
// and remote-strike hooks connect to the ending purely through their completion
// keys; the local purge is a no-op TODO hook.
func TestReactionSettlementsAreNonNumericalHooks(t *testing.T) {
	svc := bootstrapRealContent(t)
	for _, key := range []string{
		"crisis:bombing_terminal",
		"crisis:termination_release",
		"crisis:silo1_purge_execution",
	} {
		def := findContentDefinitionByKey(t, svc, key)
		if !def.ScriptHooks.Apply {
			t.Fatalf("%s must be an apply settlement hook", key)
		}
		for _, effect := range def.Effects {
			if effect.Type == "silo_population_delta" {
				t.Fatalf("%s must not reduce population", key)
			}
			if effect.Type == "silo_metric_delta" && effect.Metric == "safeguard_risk" {
				t.Fatalf("%s must not touch safeguard_risk", key)
			}
		}
	}
}

// TestReactionPurgeHookBindsInformationCountermeasure locks the information-side
// binding: PURGE_PLOT_LEAK (the everyday "spread the count times" action) schedules
// silo1_purge_exposed, and the node stays schedule-only so it never fires from the
// monthly scan. Propaganda is still NOT a countermeasure (old ruling unchanged).
func TestReactionPurgeHookBindsInformationCountermeasure(t *testing.T) {
	svc := bootstrapRealContent(t)
	def := findContentDefinitionByKey(t, svc, "special:silo1_purge_exposed")
	if !def.ScriptHooks.CanTrigger {
		t.Fatalf("special:silo1_purge_exposed must declare a canTrigger hook (schedule-only guard)")
	}
	silo := &model.Silo{ID: 1, CurrentYear: 127, CurrentMonth: 1}
	ok, err := engine.RunContentEventCanTriggerScript(def, engine.ContentEvaluationContext{
		Silo: silo,
		States: map[string]engine.ContentEventRuntime{
			"special:silo1_purge_omen":        {Triggered: true},
			"player_action:action_share_info": {Triggered: true},
		},
	}, false)
	if err != nil {
		t.Fatalf("silo1_purge_exposed canTrigger: %v", err)
	}
	if ok {
		t.Fatal("silo1_purge_exposed must stay schedule-only (canTrigger false)")
	}

	leak := findContentDefinitionByKey(t, svc, "player_actions:silo1_purge_plot_leak")
	if leak.PlayerAction == nil || leak.PlayerAction.ID != "PURGE_PLOT_LEAK" {
		t.Fatalf("plot leak is not declared as a player action: %+v", leak.PlayerAction)
	}
	scheduled := false
	for _, effect := range leak.Effects {
		if effect.Type == "schedule_event" && effect.EventID == "silo1_purge_exposed" {
			scheduled = true
		}
	}
	if !scheduled {
		t.Fatal("PURGE_PLOT_LEAK must schedule silo1_purge_exposed")
	}
}

// TestReactionSettlementFactKeysArbitrateDeath locks the ending wiring: a fired
// remote-termination or remote-strike settlement is a DEATH fact in arbitration,
// and a surviving escape group still converts it into the REBELLION pyrrhic
// epilogue (P0 escape exception).
func TestReactionSettlementFactKeysArbitrateDeath(t *testing.T) {
	for _, factKey := range []string{terminationFactKey, strikeFactKey} {
		svc, station, _ := weightedEndingFixture("TIME")
		svc.contentStates = map[uint]map[string]model.ContentEventState{
			station.ID: {factKey: {Triggered: true}},
		}
		decision := svc.resolveWeightedEnding(station, svc.agent)
		if decision == nil || decision.Selected != "DEATH" {
			t.Fatalf("settlement fact %s must arbitrate DEATH, got %+v", factKey, decision)
		}

		// With a surviving escape group the same fact becomes a pyrrhic REBELLION.
		svc2, station2, _ := weightedEndingFixture("TIME")
		station2.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 120, Surviving: true}
		svc2.contentStates = map[uint]map[string]model.ContentEventState{
			station2.ID: {factKey: {Triggered: true}},
		}
		pyrrhic := svc2.resolveWeightedEnding(station2, svc2.agent)
		if pyrrhic == nil || pyrrhic.Selected != "REBELLION" {
			t.Fatalf("settlement fact %s with escape group must avert DEATH into REBELLION, got %+v", factKey, pyrrhic)
		}
	}
}

// TestLocalPurgeSettlementIsNotADeathFact locks that the local purge settlement
// is deliberately NOT wired into the DEATH facts this round (no modeled
// consequence yet — it must not end the game on its own).
func TestLocalPurgeSettlementIsNotADeathFact(t *testing.T) {
	svc, station, _ := weightedEndingFixture("TIME")
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {"crisis:silo1_purge_execution": {Triggered: true}},
	}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision != nil && decision.Selected == "DEATH" {
		t.Fatalf("local purge settlement must not arbitrate DEATH this round, got %+v", decision)
	}
}

// TestReactionScheduleKeepsPhysicalStrikeReachable locks the minimal scheduling
// invariant: the unopposed termination release cannot land before the
// reconnaissance window plus its three-month physical delivery has elapsed, so
// the ordinary path always opens the physical channel first. The release reads
// its own relative clock (and the jamming guard) instead of being scheduled.
func TestReactionScheduleKeepsPhysicalStrikeReachable(t *testing.T) {
	svc := bootstrapRealContent(t)
	termination := findContentDefinitionByKey(t, svc, "special:termination_channel_test")
	release := findContentDefinitionByKey(t, svc, "crisis:termination_release")
	strike := findContentDefinitionByKey(t, svc, "special:strike_reconnaissance")
	for _, effect := range termination.Effects {
		if effect.Type == "schedule_event" && effect.EventID == "termination_release" {
			t.Fatal("termination_channel_test must not schedule the release: the release owns its clock and guard")
		}
	}
	releaseDelay, reconDelay, bombingDelay := 0, 0, 0
	guarded := false
	for _, condition := range release.Trigger.Conditions {
		if condition.Type == "months_since_event_gte" && condition.EventID == "termination_channel_test" {
			releaseDelay = condition.Months
		}
		if condition.Type == "world_flag_not_set" && condition.Flag == model.WorldFlagNetworkJammed {
			guarded = true
		}
	}
	if !guarded {
		t.Fatal("termination_release must read the network_jammed guard (分渠道纪律)")
	}
	for _, condition := range strike.Trigger.Conditions {
		if condition.Type == "months_since_event_gte" && condition.EventID == "termination_channel_test" {
			reconDelay = condition.Months
		}
	}
	for _, effect := range strike.Effects {
		if effect.Type == "schedule_event" && effect.EventID == "bombing_terminal" {
			bombingDelay = effect.DelayMonths
		}
	}
	if releaseDelay == 0 || reconDelay == 0 || bombingDelay == 0 {
		t.Fatalf("reaction schedule is incomplete: release=%d recon=%d bombing=%d", releaseDelay, reconDelay, bombingDelay)
	}
	if releaseDelay <= reconDelay+bombingDelay {
		t.Fatalf("termination release at +%d months must follow physical delivery at +%d months",
			releaseDelay, reconDelay+bombingDelay)
	}
}

// TestTerminationChannelCountermeasureBinding locks the everyday-action binding:
// the jamming action schedules the disrupted-channel settlement, the release
// guards on that world fact, and a jammed channel never reaches DEATH through the
// remote channel even when the completion key is present.
func TestTerminationChannelCountermeasureBinding(t *testing.T) {
	svc := bootstrapRealContent(t)
	jam := findContentDefinitionByKey(t, svc, "player_actions:termination_channel_jam")
	if jam.PlayerAction == nil || jam.PlayerAction.ID != "TERMINATION_CHANNEL_JAM" {
		t.Fatalf("jamming action is not declared as a player action: %+v", jam.PlayerAction)
	}
	scheduled := false
	for _, effect := range jam.Effects {
		if effect.Type == "schedule_event" && effect.EventID == "termination_channel_disrupted" {
			scheduled = true
		}
	}
	if !scheduled {
		t.Fatal("jamming action must schedule termination_channel_disrupted")
	}
	release := findContentDefinitionByKey(t, svc, "crisis:termination_release")
	if !selectsFlagGuard(release, model.WorldFlagNetworkJammed) {
		t.Fatal("termination_release must guard on network_jammed")
	}

	// Trigger evaluation: with the self-test recorded and 10 months elapsed the
	// release is due, but a jammed channel must keep it from firing.
	dueStates := map[string]engine.ContentEventRuntime{
		"special:termination_channel_test": {Triggered: true, LastTriggeredTimestamp: 0},
	}
	silo := &model.Silo{ID: 1, CurrentYear: 131, CurrentMonth: 1}
	world := &model.World{}
	if !engine.CanTriggerContentEvent(release, engine.ContentEvaluationContext{
		Silo: silo, States: dueStates, World: world,
	}) {
		// The months gate is anchored to the recorded timestamp; set it explicitly
		// so the check is about the flag, not about fixture arithmetic.
		t.Fatal("unopposed release must fire once the window elapses")
	}
	world.Flags = map[string]bool{model.WorldFlagNetworkJammed: true}
	if engine.CanTriggerContentEvent(release, engine.ContentEvaluationContext{
		Silo: silo, States: dueStates, World: world,
	}) {
		t.Fatal("jammed release must be a no-op (never fires)")
	}
	// Single-silo context (no world) still allows the release: the guard only
	// closes the channel when the world fact is actually recorded.
	if !engine.CanTriggerContentEvent(release, engine.ContentEvaluationContext{
		Silo: silo, States: dueStates,
	}) {
		t.Fatal("release must not be blocked by the missing world context")
	}

	// Arbitration still honours the fact when the remote channel was NOT jammed.
	svcJammed, station, _ := weightedEndingFixture("TIME")
	svcJammed.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {terminationFactKey: {Triggered: true}},
	}
	if decision := svcJammed.resolveWeightedEnding(station, svcJammed.agent); decision == nil || decision.Selected != "DEATH" {
		t.Fatalf("termination fact must arbitrate DEATH when present, got %+v", decision)
	}
}

func selectsFlagGuard(def engine.ContentEventDefinition, flag string) bool {
	for _, condition := range def.Trigger.Conditions {
		if condition.Type == "world_flag_not_set" && condition.Flag == flag {
			return true
		}
	}
	return false
}

// TestDeathPhysicalStrikePlayableFromNewGame drives the complete Silo 1 physical
// termination channel through public game operations. Deception raises real
// rebellion; Supply rationing keeps FRACTURE from winning the race before the
// scheduled strike lands. No metric, content fact, escape group, ending status,
// or delay task is seeded by the test.
func TestDeathPhysicalStrikePlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testDeathPhysicalStrikePlayableFromNewGame)
}

func testDeathPhysicalStrikePlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "death-physical-strike.db"))
	req := threeSiloArchetypeRequest(2101, archetype)
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
	}
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionConductPropaganda})
	}

	observed := map[string]bool{}
	observedAt := map[string]int{}
	observe := func() {
		t.Helper()
		for _, key := range []string{
			"histories:silo1_ultimatum",
			"special:silo1_ultimatum_cross",
			"special:silo1_purge_omen",
			"special:termination_channel_test",
			"special:strike_reconnaissance",
			"crisis:bombing_terminal",
			"crisis:termination_release",
		} {
			if observed[key] || !contentTriggered(svc, key) {
				continue
			}
			observed[key] = true
			observedAt[key] = gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth)
			if key == "crisis:bombing_terminal" && svc.gameOver() {
				t.Fatalf("bombing was first observed after game over: status=%+v", svc.silo.VictoryStatus)
			}
			t.Logf("%s fired at %d-%02d: rebellion=%.3f energy=%.0f supplies=%.0f population=%d suspicion=%.3f game_over=%t",
				key, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), resourceAmount(svc.silo, "Supplies"),
				svc.silo.TotalPopulation, svc.agent.SuspicionLevel, svc.gameOver())
		}
	}

	deceivePending := func() bool {
		for _, op := range svc.world.StationOps.Operations {
			if op.ActionID == StationOpDeceive && op.Status == "PENDING" {
				return true
			}
		}
		return false
	}
	deceives, rations := 0, 0
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		observe()
		if svc.gameOver() {
			break
		}

		buildupOpen := svc.silo.CurrentYear >= 128
		buildingRebellion := buildupOpen && svc.silo.Rebellion < 0.82
		pending := deceivePending()
		deceive := actionByID(svc.availablePlayerActions(), StationOpDeceive)
		// Famine retune (19 §6.3 荒年局多配给): the physical channel only opens
		// to a station that is still running (Energy ≥ 1), and reconnaissance
		// must land within one month of its first eligible slot or the physical
		// delivery overshoots the scheduled remote termination. Famine drains
		// energy to zero earlier and more often, so once the channel test is
		// lit the energy floor is defended regardless of rebellion — the old
		// rebellion-gated guard alone woke up too late on the famine draw.
		strikeChannelLive := contentTriggered(svc, "special:termination_channel_test") &&
			!contentTriggered(svc, "special:strike_reconnaissance")
		energyGuard := (svc.silo.Rebellion >= 0.6 && resourceAmount(svc.silo, "Energy") < 1500) ||
			(strikeChannelLive && resourceAmount(svc.silo, "Energy") < 500)
		if buildingRebellion && deceives < 2 && !pending && !energyGuard && deceive != nil && deceive.Enabled &&
			svc.agent.ActionPoints >= float64(deceive.APCost+15) {
			execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive})
			deceives++
			t.Logf("deceive %d at %d-%02d: rebellion=%.3f energy=%.0f population=%d AP=%.1f suspicion=%.3f",
				deceives, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.ActionPoints, svc.agent.SuspicionLevel)
			observe()
			continue
		}

		// Once the repeatable underground-coordination node can snowball
		// rebellion, keep Energy positive so it cannot confirm FRACTURE.
		if energyGuard && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{
				Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Energy",
			})
			rations++
			t.Logf("ration %d at %d-%02d: rebellion=%.3f energy=%.0f supplies=%.0f materials=%.0f population=%d AP=%.1f",
				rations, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Rebellion,
				resourceAmount(svc.silo, "Energy"), resourceAmount(svc.silo, "Supplies"),
				resourceAmount(svc.silo, "Materials"), svc.silo.TotalPopulation, svc.agent.ActionPoints)
			observe()
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance DEATH physical strike at month %d: %v", month, err)
		}
		observe()
	}

	if deceives == 0 || rations == 0 {
		t.Fatalf("route did not use its real rebellion and survival levers: deceives=%d rations=%d", deceives, rations)
	}
	states := svc.contentStates[svc.silo.ID]
	for _, key := range []string{
		"histories:silo1_ultimatum",
		"special:silo1_ultimatum_cross",
		"special:silo1_purge_omen",
		"special:termination_channel_test",
		"special:strike_reconnaissance",
		"crisis:bombing_terminal",
	} {
		if !states[key].Triggered || states[key].TriggerCount != 1 {
			t.Fatalf("%s did not fire exactly once through real play: state=%+v status=%+v", key, states[key], svc.silo.VictoryStatus)
		}
	}
	ultimatum := states["histories:silo1_ultimatum"]
	strike := states["special:strike_reconnaissance"]
	bombing := states["crisis:bombing_terminal"]
	if ultimatum.LastTriggeredTimestamp >= strike.LastTriggeredTimestamp {
		t.Fatalf("physical strike did not follow the ultimatum: ultimatum=%+v strike=%+v", ultimatum, strike)
	}
	if bombing.LastTriggeredTimestamp <= strike.LastTriggeredTimestamp {
		t.Fatalf("bombing did not follow reconnaissance: strike=%+v bombing=%+v", strike, bombing)
	}
	// Reconnaissance is recorded by the post-clock content sweep, while its
	// scheduled target is dispatched before the third subsequent local phase.
	// Measure at completed public operations so both ends use the same clock
	// boundary and exactly three real monthly advances separate their observation.
	if got, want := observedAt["crisis:bombing_terminal"]-observedAt["special:strike_reconnaissance"], 3*daysPerMonth; got != want {
		t.Fatalf("publicly observed bombing delay = %d days, want %d: observed_at=%v strike=%+v bombing=%+v",
			got, want, observedAt, strike, bombing)
	}
	if contentTriggered(svc, fractureFactKey) {
		t.Fatal("FRACTURE settled before the physical strike could land")
	}
	if termination := states[terminationFactKey]; termination.Triggered {
		if termination.LastTriggeredTimestamp <= bombing.LastTriggeredTimestamp {
			t.Fatalf("remote termination did not land after the physical strike: bombing=%+v termination=%+v", bombing, termination)
		}
		if !observed[terminationFactKey] || observedAt[terminationFactKey] <= observedAt["crisis:bombing_terminal"] {
			t.Fatalf("publicly observed release must strictly follow bombing: observed_at=%v", observedAt)
		}
	}
	if svc.silo.EscapedGroup.SurvivingEscapeGroup() {
		t.Fatalf("route unexpectedly produced a surviving escape group: %+v", svc.silo.EscapedGroup)
	}
	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil || status.Decision.Selected != endingDeath {
		t.Fatalf("physical termination chain did not select DEATH at %d-%02d: status=%+v rebellion=%.3f energy=%.0f population=%d suspicion=%.3f",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, status, svc.silo.Rebellion,
			resourceAmount(svc.silo, "Energy"), svc.silo.TotalPopulation, svc.agent.SuspicionLevel)
	}
	t.Logf("DEATH settled at %d-%02d after %d deceives and %d rations: decision=%s escaped_group=%+v",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, deceives, rations, status.Decision.Selected, svc.silo.EscapedGroup)

	terminalAction := model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}
	if outcome, err := svc.ExecuteAction(terminalAction); err == nil {
		t.Fatalf("terminal DEATH state accepted ExecuteAction: %+v", outcome.Result)
	}
	if _, err := svc.BeginAction(terminalAction); err == nil {
		t.Fatal("terminal DEATH state accepted BeginAction")
	}
}

// TestTerminationChannelCountermeasurePlayableFromNewGame drives the everyday
// jamming action through public operations: build real trust with Mechanical,
// let the Silo 1 ladder reach the channel self-test, then spend AP on the
// maintenance-night action. The jam lands a month later; the physical channel
// still kills the silo, which is the point of the per-channel discipline.
func TestTerminationChannelCountermeasurePlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testTerminationChannelCountermeasurePlayableFromNewGame)
}

func testTerminationChannelCountermeasurePlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "termination-jam.db"))
	req := threeSiloArchetypeRequest(2101, archetype)
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}
	tryExecute := func(action model.AgentAction) bool {
		outcome, err := svc.ExecuteAction(action)
		return err == nil && outcome.Result.Executed
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if len(svc.agent.KnownFragments) > 0 {
		execute(model.AgentAction{
			Type: model.ActionShareInfo, TargetDept: "Mayor",
			FragmentIds: []string{svc.agent.KnownFragments[0]},
		})
	}
	// The everyday action needs a real Mechanical (or IT) contact; build it the
	// ordinary way instead of seeding the relation. One Mayor contact keeps the
	// propaganda lever unlocked (relation ≥ 0.15).
	execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
	execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mechanical"})
	}
	execute(model.AgentAction{Type: model.ActionConductPropaganda})
	execute(model.AgentAction{Type: model.ActionConductPropaganda})

	deceivePending := func() bool {
		for _, op := range svc.world.StationOps.Operations {
			if op.ActionID == StationOpDeceive && op.Status == "PENDING" {
				return true
			}
		}
		return false
	}
	jamMonth, deceives, rations := 0, 0, 0
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		if contentTriggered(svc, "special:termination_channel_test") {
			if jam := actionByID(svc.availablePlayerActions(), "TERMINATION_CHANNEL_JAM"); jam != nil && jam.Enabled {
				if tryExecute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "TERMINATION_CHANNEL_JAM"}) {
					jamMonth = month
					t.Logf("jammed the termination channel at %d-%02d (AP left %.1f)",
						svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.agent.ActionPoints)
				}
			}
		}
		if jamMonth > 0 && contentTriggered(svc, "special:termination_channel_disrupted") {
			break
		}

		buildupOpen := svc.silo.CurrentYear >= 128
		switch svc.silo.Archetype {
		case engine.ArchetypeFamine:
			// 荒年的危机驱动恐慌把一号堡阶梯整体提前，信道自检早于基线约一年
			// 到达；积累窗口相应前移，保证自检仍由真实叛乱 (玩家策反) 达到，
			// 而不是被动等来自饥荒压力的阶梯 (19 §6.3 策略时机迁移)。
			buildupOpen = svc.silo.CurrentYear >= 126
		case engine.ArchetypeRift:
			// 裂隙的波动放大同样提前自检；在真实自检窗口前先使用既有策反。
			buildupOpen = svc.silo.CurrentYear >= 127
		case engine.ArchetypeBedrock:
			// 磐石档的阶梯轨迹实测把自检推到 126-06 (高 W 改变早期事件时序)，
			// 积累窗口同荒年前移，保证自检经真实叛乱达到。
			buildupOpen = svc.silo.CurrentYear >= 126
		}
		buildingRebellion := buildupOpen && svc.silo.Rebellion < 0.82
		deceive := actionByID(svc.availablePlayerActions(), StationOpDeceive)
		energyGuard := svc.silo.Rebellion >= 0.6 && resourceAmount(svc.silo, "Energy") < 1500
		if buildingRebellion && deceives < 2 && !deceivePending() && !energyGuard &&
			deceive != nil && deceive.Enabled && svc.agent.ActionPoints >= float64(deceive.APCost+15) {
			execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpDeceive})
			deceives++
			continue
		}
		if energyGuard && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{
				Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Energy",
			})
			rations++
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance termination-jam route at month %d: %v", month, err)
		}
	}

	if jamMonth == 0 {
		t.Fatalf("the everyday jamming action never became available: states=%+v", svc.contentStates[svc.silo.ID]["special:termination_channel_test"])
	}
	if deceives == 0 {
		t.Fatalf("route did not reach the channel self-test through real rebellion: deceives=%d", deceives)
	}
	states := svc.contentStates[svc.silo.ID]
	if !states["special:termination_channel_disrupted"].Triggered {
		t.Fatalf("jamming did not land: states=%+v", states["special:termination_channel_disrupted"])
	}
	if !svc.world.Flags[model.WorldFlagNetworkJammed] {
		t.Fatal("network_jammed was not written by the countermeasure")
	}
	if states[terminationFactKey].Triggered {
		t.Fatalf("the jammed remote channel still released: %+v", states[terminationFactKey])
	}
}

// TestPurgeExecutionGradesOnCountermeasure locks the purge settlement: the same
// execution node hits harder when nobody touched the roster, and records the
// grades in population/history/cohesion/panic rather than a single constant.
func TestPurgeExecutionGradesOnCountermeasure(t *testing.T) {
	svc := bootstrapRealContent(t)
	def := findContentDefinitionByKey(t, svc, "crisis:silo1_purge_execution")
	silo := &model.Silo{ID: 1, TotalPopulation: 8000, CurrentYear: 128, CurrentMonth: 1}
	grade := func(states map[string]engine.ContentEventRuntime) engine.ContentScriptApplyResult {
		t.Helper()
		result, err := engine.RunContentEventApplyScript(def, engine.ContentEvaluationContext{
			Silo: silo, States: states,
		})
		if err != nil {
			t.Fatalf("purge apply: %v", err)
		}
		return result
	}
	effectValue := func(result engine.ContentScriptApplyResult, effectType, metric string) float64 {
		for _, effect := range result.Effects {
			if effect.Type == effectType && (metric == "" || effect.Metric == metric) {
				return effect.Value
			}
		}
		return 0
	}

	heavy := grade(map[string]engine.ContentEventRuntime{})
	soft := grade(map[string]engine.ContentEventRuntime{
		"special:silo1_purge_softened": {Triggered: true},
	})
	exposed := grade(map[string]engine.ContentEventRuntime{
		"special:silo1_purge_exposed": {Triggered: true},
	})
	both := grade(map[string]engine.ContentEventRuntime{
		"special:silo1_purge_softened": {Triggered: true},
		"special:silo1_purge_exposed":  {Triggered: true},
	})
	heavyPop := effectValue(heavy, "silo_population_delta", "")
	softPop := effectValue(soft, "silo_population_delta", "")
	exposedPop := effectValue(exposed, "silo_population_delta", "")
	if !(heavyPop < exposedPop && exposedPop < softPop && softPop < 0) {
		t.Fatalf("countermeasures must grade the purge toll: heavy=%v exposed=%v soft=%v", heavyPop, exposedPop, softPop)
	}
	if got := effectValue(both, "silo_population_delta", ""); got != softPop {
		t.Fatalf("both countermeasures must settle at the soft grade: got=%v want=%v", got, softPop)
	}
	if effectValue(exposed, "silo_metric_delta", "history_burden") >= effectValue(heavy, "silo_metric_delta", "history_burden") {
		t.Fatal("exposed purge must leave less history burden than the heavy one")
	}
	if !(effectValue(exposed, "silo_metric_delta", "cohesion") > effectValue(heavy, "silo_metric_delta", "cohesion")) {
		t.Fatal("exposed purge must cost less cohesion than the heavy one")
	}
	if heavyPop > -0.06*8000 {
		t.Fatalf("heavy grade unexpectedly light: %v", heavyPop)
	}
	if softPop < -0.05*8000 {
		t.Fatalf("soft grade unexpectedly heavy: %v", softPop)
	}
	if effectValue(heavy, "silo_metric_delta", "history_burden") <= effectValue(soft, "silo_metric_delta", "history_burden") {
		t.Fatal("heavy purge must leave more history burden than the softened one")
	}
	if effectValue(heavy, "silo_metric_delta", "cohesion") >= effectValue(soft, "silo_metric_delta", "cohesion") {
		t.Fatal("heavy purge must cost more cohesion than the softened one")
	}
}

// TestPurgeCountermeasurePlayableFromNewGame drives the roster tampering through
// public operations: build IT trust, land the purge omen, spend AP on the
// rebinding night, and verify the softened fact lands on the scheduled path.
// The execution's graded toll is locked separately in
// TestPurgeExecutionGradesOnCountermeasure.
func TestPurgeCountermeasurePlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testPurgeCountermeasurePlayableFromNewGame)
}

func testPurgeCountermeasurePlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "purge-countermeasure.db"))
	req := threeSiloArchetypeRequest(2101, archetype)
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}
	itRelation := func() float64 {
		for _, profession := range svc.silo.Professions {
			if profession.Name != "IT" {
				continue
			}
			for _, connection := range svc.agent.Connections {
				if connection.ProfessionID == profession.ID {
					return connection.Value
				}
			}
		}
		return 0
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if len(svc.agent.KnownFragments) > 0 {
		execute(model.AgentAction{
			Type: model.ActionShareInfo, TargetDept: "Mayor",
			FragmentIds: []string{svc.agent.KnownFragments[0]},
		})
	}
	// Build the IT contact the ordinary way until the everyday action unlocks.
	for month := 0; month < 6 && itRelation() < 0.2; month++ {
		if build := actionByID(svc.availablePlayerActions(), "action_build_connection"); build != nil &&
			build.Enabled && svc.agent.ActionPoints >= float64(build.APCost) {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "IT"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance relation build: %v", err)
		}
	}
	if itRelation() < 0.2 {
		t.Fatalf("could not build the IT contact: relation=%.2f", itRelation())
	}

	// Seed the landed omen fact (its own trigger reachability is covered by the
	// termination route and the monthly queue). From here the chain runs through
	// public actions and the scheduled-event path only.
	siloID := svc.silo.ID
	states := svc.contentStates[siloID]
	if states == nil {
		states = map[string]model.ContentEventState{}
		svc.contentStates[siloID] = states
	}
	states["special:silo1_purge_omen"] = model.ContentEventState{
		SiloID: siloID, EventKey: "special:silo1_purge_omen", Triggered: true, TriggerCount: 1,
		LastTriggeredTimestamp: gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth),
	}
	tamper := actionByID(svc.availablePlayerActions(), "PURGE_ROSTER_TAMPER")
	if tamper == nil || !tamper.Enabled {
		t.Fatalf("roster tampering is not available after the omen: %+v (relation=%.2f)", tamper, itRelation())
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "PURGE_ROSTER_TAMPER"})
	if !contentTriggered(svc, "player_action:PURGE_ROSTER_TAMPER") {
		t.Fatal("roster tampering action was not recorded")
	}
	for month := 0; month < 3 && !contentTriggered(svc, "special:silo1_purge_softened"); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance purge countermeasure: %v", err)
		}
	}
	if !contentTriggered(svc, "special:silo1_purge_softened") {
		t.Fatal("roster tampering did not record the softened fact")
	}
}

// The information-side countermeasure must be playable through public actions:
// build the Judicial contact the ordinary way, then use the everyday "spread the
// count times" action after the omen. It records silo1_purge_exposed, which grades
// the execution at the middle tier (see TestPurgeExecutionGradesOnCountermeasure).
func TestPurgePlotLeakPlayableFromNewGame(t *testing.T) {
	runPlayableArchetypeMatrix(t, testPurgePlotLeakPlayableFromNewGame)
}

func testPurgePlotLeakPlayableFromNewGame(t *testing.T, archetype string) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "purge-plot-leak.db"))
	req := threeSiloArchetypeRequest(2103, archetype)
	req.Profession = "Supply"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}
	judicialRelation := func() float64 {
		for _, profession := range svc.silo.Professions {
			if profession.Name != "Judicial" {
				continue
			}
			for _, connection := range svc.agent.Connections {
				if connection.ProfessionID == profession.ID {
					return connection.Value
				}
			}
		}
		return 0
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	for month := 0; month < 10 && judicialRelation() < 0.2; month++ {
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled && svc.agent.ActionPoints >= float64(build.APCost) {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance relation build: %v", err)
		}
	}
	if judicialRelation() < 0.2 {
		t.Fatalf("could not build the Judicial contact: relation=%.2f", judicialRelation())
	}

	// Seed the landed omen fact; the binding under test starts from here.
	siloID := svc.silo.ID
	states := svc.contentStates[siloID]
	if states == nil {
		states = map[string]model.ContentEventState{}
		svc.contentStates[siloID] = states
	}
	states["special:silo1_purge_omen"] = model.ContentEventState{
		SiloID: siloID, EventKey: "special:silo1_purge_omen", Triggered: true, TriggerCount: 1,
		LastTriggeredTimestamp: gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth),
	}
	for svc.agent.ActionPoints < 2 {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance to afford the leak: %v", err)
		}
	}
	leak := actionByID(svc.availablePlayerActions(), "PURGE_PLOT_LEAK")
	if leak == nil || !leak.Enabled {
		t.Fatalf("plot leak is not available after the omen: %+v (relation=%.2f)", leak, judicialRelation())
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "PURGE_PLOT_LEAK"})
	if !contentTriggered(svc, "player_action:PURGE_PLOT_LEAK") {
		t.Fatal("plot leak action was not recorded")
	}
	for month := 0; month < 3 && !contentTriggered(svc, "special:silo1_purge_exposed"); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance purge exposure: %v", err)
		}
	}
	if !contentTriggered(svc, "special:silo1_purge_exposed") {
		t.Fatal("plot leak did not record the exposed fact")
	}
}

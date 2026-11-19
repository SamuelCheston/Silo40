package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"silo40/internal/model"
)

func newGatedService(t *testing.T, dbPath string) *GameService {
	t.Helper()
	svc := newTwoSiloService(t, dbPath)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.loadActionGates(dirs); err != nil {
		t.Fatalf("load action gates: %v", err)
	}
	return svc
}

func completeGateEvent(svc *GameService, eventKey string) {
	svc.statesMapFor(svc.silo.ID)[eventKey] = model.ContentEventState{
		SiloID: svc.silo.ID, EventKey: eventKey, Triggered: true, TriggerCount: 1,
	}
	svc.refreshActionProgress()
}

func setGateConnection(svc *GameService, professionName string, value float64) {
	for _, profession := range svc.silo.Professions {
		if profession.Name != professionName {
			continue
		}
		for i := range svc.agent.Connections {
			if svc.agent.Connections[i].ProfessionID == profession.ID {
				svc.agent.Connections[i].Value = value
				return
			}
		}
		svc.agent.Connections = append(svc.agent.Connections, model.Connection{AgentID: svc.agent.ID, ProfessionID: profession.ID, Value: value})
		return
	}
}

func actionByID(actions []model.PlayerActionMeta, id string) *model.PlayerActionMeta {
	for i := range actions {
		if actions[i].ID == id {
			return &actions[i]
		}
	}
	return nil
}

// TestActionChoiceGroupsAreExclusive is a gate unit test: it seeds completion
// facts instead of claiming to drive each story chain as a playable run.
func TestActionChoiceGroupsAreExclusive(t *testing.T) {
	type choice struct {
		id       string
		eventKey string
	}
	groups := []struct {
		name        string
		unlockEvent string
		choices     []choice
	}{
		{
			name: "charter_contact", unlockEvent: "player_actions:charter_contact_offer",
			choices: []choice{
				{id: "CHARTER_CONTACT_ACCEPT", eventKey: "player_actions:charter_contact_accept"},
				{id: "CHARTER_CONTACT_DECLINE", eventKey: "player_actions:charter_contact_decline"},
				{id: "CHARTER_CONTACT_DEFER", eventKey: "player_actions:charter_contact_defer"},
			},
		},
		{
			name: "mayor_ally", unlockEvent: "player_actions:legitimacy_survey",
			choices: []choice{
				{id: "MAYOR_ALLY_ACCEPT", eventKey: "player_actions:mayor_ally_accept"},
				{id: "MAYOR_ALLY_BYPASS", eventKey: "player_actions:mayor_ally_bypass"},
			},
		},
		{
			name: "succession", unlockEvent: "special:mayor_public_stand",
			choices: []choice{
				{id: "SUCCESSION_PURE_RULE", eventKey: "player_actions:succession_pure_rule"},
				{id: "SUCCESSION_SEED_PROCEDURE", eventKey: "player_actions:succession_seed_procedure"},
			},
		},
		{
			name: "pre_fall", unlockEvent: "special:pre_fall_choice",
			choices: []choice{
				{id: "PRE_FALL_LOYALTY_MOBILIZATION", eventKey: "player_actions:pre_fall_loyalty_mobilization"},
				{id: "PRE_FALL_TOTAL_RIOT", eventKey: "player_actions:pre_fall_total_riot"},
				{id: "PRE_FALL_OPEN_INQUIRY", eventKey: "player_actions:pre_fall_open_inquiry"},
			},
		},
		{
			name: "route_survey", unlockEvent: "player_actions:route_survey_offer",
			choices: []choice{
				{id: "ROUTE_SURVEY_ACCEPT", eventKey: "player_actions:route_survey_accept"},
				{id: "ROUTE_SURVEY_DECLINE", eventKey: "player_actions:route_survey_decline"},
				{id: "ROUTE_SURVEY_DEFER", eventKey: "player_actions:route_survey_defer"},
			},
		},
		{
			name: "escape_window", unlockEvent: "special:escape_window_open",
			choices: []choice{
				{id: "ESCAPE_WINDOW_ORDERLY", eventKey: "player_actions:escape_window_orderly"},
				{id: "ESCAPE_WINDOW_RIOT", eventKey: "player_actions:escape_window_riot"},
			},
		},
	}

	for _, group := range groups {
		for _, selected := range group.choices {
			t.Run(group.name+"/"+selected.id, func(t *testing.T) {
				svc := newGatedService(t, filepath.Join(t.TempDir(), "choice-exclusive.db"))
				if _, err := svc.CreateGame(threeSiloRequest(7001)); err != nil {
					t.Fatalf("create game: %v", err)
				}
				completeGateEvent(svc, group.unlockEvent)
				for _, option := range group.choices {
					meta := actionByID(svc.availablePlayerActions(), option.id)
					if meta == nil || !meta.Enabled || meta.Visibility != model.ActionAvailable {
						t.Fatalf("%s must initially be available: %+v", option.id, meta)
					}
				}

				completeGateEvent(svc, selected.eventKey)
				actions := svc.availablePlayerActions()
				for _, option := range group.choices {
					if !svc.actionUnlocked(option.id) {
						t.Fatalf("%s lost its permanent unlock after %s", option.id, selected.id)
					}
					if option.id == selected.id {
						continue
					}
					if meta := actionByID(actions, option.id); meta != nil {
						t.Fatalf("%s remains visible after %s was selected: %+v", option.id, selected.id, meta)
					}
					_, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: option.id})
					if err == nil || !strings.Contains(err.Error(), "同一组抉择已选定其他项") {
						t.Fatalf("%s bypass after %s was not rejected by its gate: %v", option.id, selected.id, err)
					}
				}
			})
		}
	}
}

func TestActionGatesStartNarrowAndUnlockFromStoryStage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gates-story.db")
	svc := newGatedService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(101)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	initial := svc.availablePlayerActions()
	if len(initial) != 1 || initial[0].ID != "action_gather_info" || initial[0].Visibility != model.ActionAvailable {
		t.Fatalf("opening must expose only Gather Info: %+v", initial)
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionSilo1Operation, ActionID: Silo1OpDeceive}); err == nil || !strings.Contains(err.Error(), "not been unlocked") {
		t.Fatalf("locked operation bypass was not rejected: %v", err)
	}

	svc.agent.KnownFragments = nil
	completeGateEvent(svc, observationFirstLocalKey)
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("story node did not advance stage: %s", svc.world.ProgressStage)
	}
	share := actionByID(svc.availablePlayerActions(), "action_share_info")
	if share == nil || share.Visibility != model.ActionUnlocked || share.Enabled || !strings.Contains(share.DisabledReason, "至少 1 个") {
		t.Fatalf("share should be visible with a concrete disabled reason: %+v", share)
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionShareInfo, TargetDept: "IT"}); err == nil || !strings.Contains(err.Error(), "至少 1 个") {
		t.Fatalf("unavailable action bypass was not rejected with its reason: %v", err)
	}
	svc.agent.KnownFragments = []string{"IT_1"}
	share = actionByID(svc.availablePlayerActions(), "action_share_info")
	if share == nil || share.Visibility != model.ActionAvailable || !share.Enabled || share.DisabledReason != "" {
		t.Fatalf("share did not become available after meeting its attribute threshold: %+v", share)
	}

	completeGateEvent(svc, "player_action:action_share_info")
	if svc.world.ProgressStage != "archive_awakened" {
		t.Fatalf("second story node did not advance the ordered stage: %s", svc.world.ProgressStage)
	}
	if err := svc.SaveGame(); err != nil {
		t.Fatalf("save: %v", err)
	}
	fresh := newGatedService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if fresh.world.ProgressStage != "archive_awakened" || !fresh.actionUnlocked("action_share_info") {
		t.Fatalf("stage/unlocks were not restored: stage=%s unlocks=%v", fresh.world.ProgressStage, fresh.world.UnlockedActions)
	}
}

// TestActionGatesKeepAdvancingThroughLateChain locks 修复5: the stage track
// must not go silent after archive_awakened (month 1) — the late 《共证》
// milestones (living chain, public fact) advance it, so the stage line in the
// world panel keeps reflecting progress through the endgame.
func TestActionGatesKeepAdvancingThroughLateChain(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "gates-late-chain.db"))
	if _, err := svc.CreateGame(threeSiloRequest(103)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	completeGateEvent(svc, observationFirstLocalKey)
	completeGateEvent(svc, "player_action:action_share_info")
	if svc.world.ProgressStage != "archive_awakened" {
		t.Fatalf("opening chain did not reach archive_awakened: %s", svc.world.ProgressStage)
	}

	completeGateEvent(svc, "special:living_chain_confirmed")
	if svc.world.ProgressStage != "chain_living" {
		t.Fatalf("living chain milestone did not advance the stage: %s", svc.world.ProgressStage)
	}
	completeGateEvent(svc, "special:public_fact_cross_dept")
	if svc.world.ProgressStage != "chain_public_fact" {
		t.Fatalf("public fact milestone did not advance the stage: %s", svc.world.ProgressStage)
	}
}

// TestReceiverVerifyGateReasonStaysReadable locks the gate-reason wording: the
// player-facing reason states the concrete Judicial bar and never leaks
// balance jargon like 初始校准值.
func TestReceiverVerifyGateReasonStaysReadable(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "gates-receiver-reason.db"))
	if _, err := svc.CreateGame(threeSiloRequest(104)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	completeGateEvent(svc, observationFirstLocalKey)
	completeGateEvent(svc, "player_action:action_share_info")
	setGateConnection(svc, "Judicial", 0)

	receiver := actionByID(svc.availablePlayerActions(), "RECEIVER_VERIFY_ACCOMPANY")
	if receiver == nil {
		t.Fatal("receiver verification missing once unlocked")
	}
	if receiver.Enabled {
		t.Fatal("receiver verification must be disabled without Judicial trust")
	}
	if !strings.Contains(receiver.DisabledReason, "0.10") || strings.Contains(receiver.DisabledReason, "校准") {
		t.Fatalf("gate reason is not player-readable: %q", receiver.DisabledReason)
	}
}

func TestActionGatesUseAttributesRelationsAndAPReasons(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "gates-thresholds.db"))
	if _, err := svc.CreateGame(threeSiloRequest(102)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	setGateConnection(svc, "Mayor", 0)
	setGateConnection(svc, "Mechanical", 0)
	svc.agent.Ambition = 0
	svc.agent.Influence = 0
	svc.world.UnlockedActions = nil
	svc.refreshActionProgress()
	if actionByID(svc.availablePlayerActions(), "action_conduct_propaganda") != nil {
		t.Fatal("relation-gated propaganda should remain hidden below its unlock threshold")
	}

	completeGateEvent(svc, observationFirstLocalKey)
	setGateConnection(svc, "Mayor", 0.2)
	svc.refreshActionProgress()
	if action := actionByID(svc.availablePlayerActions(), "action_conduct_propaganda"); action == nil || !action.Enabled {
		t.Fatalf("relation threshold did not unlock propaganda: %+v", action)
	}

	svc.agent.Ambition = 0.6
	svc.agent.Influence = 0.6
	setGateConnection(svc, "Mayor", 0)
	svc.refreshActionProgress()
	incite := actionByID(svc.availablePlayerActions(), "action_incite_rebellion")
	if incite == nil || incite.Enabled || !strings.Contains(incite.DisabledReason, "0.25") {
		t.Fatalf("attribute unlock and relation availability should be independent: %+v", incite)
	}

	svc.agent.ActionPoints = 0
	gather := actionByID(svc.availablePlayerActions(), "action_gather_info")
	if gather == nil || gather.Enabled || gather.Visibility != model.ActionUnlocked || !strings.Contains(gather.DisabledReason, "AP") {
		t.Fatalf("AP shortage must produce an unlocked disabled state: %+v", gather)
	}
	if _, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "IT"}); err == nil || !strings.Contains(err.Error(), "Action Points") {
		t.Fatalf("execution should reject AP shortage before effects: %v", err)
	}
}

func TestOpeningRequiresSuccessfulActionsNotIdleMonths(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "opening-actions.db"))
	if _, err := svc.CreateGame(threeSiloRequest(104)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if got := len(svc.agent.KnownFragments); got != initialPlayerFragmentsPerProfession {
		t.Fatalf("opening knowledge is not limited: %d fragments", got)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("idle month: %v", err)
	}
	if svc.world.ProgressStage != "arrival" {
		t.Fatalf("idle month advanced the stage without an action: %s", svc.world.ProgressStage)
	}
	if actions := svc.availablePlayerActions(); len(actions) != 1 || actions[0].ID != "action_gather_info" {
		t.Fatalf("idle month unlocked actions without completion: %+v", actions)
	}

	gathered, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if err != nil || !gathered.Result.Executed {
		t.Fatalf("opening Gather Info must have real work: result=%+v err=%v", gathered, err)
	}
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("successful Gather did not advance stage: %s", svc.world.ProgressStage)
	}
	shared, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionShareInfo, TargetDept: "Mayor", FragmentIds: []string{svc.agent.KnownFragments[0]}})
	if err != nil || !shared.Result.Executed {
		t.Fatalf("second action must complete: result=%+v err=%v", shared, err)
	}
	if svc.world.ProgressStage != "archive_awakened" {
		t.Fatalf("successful Share did not advance stage: %s", svc.world.ProgressStage)
	}
}

func TestOpeningCompletionWorksThroughQueuedAPI(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "opening-queued.db"))
	if _, err := svc.CreateGame(threeSiloRequest(105)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.BeginAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("begin Gather: %v", err)
	}
	drainOperationForTest(t, svc)
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("queued action completion did not advance stage: %s", svc.world.ProgressStage)
	}
}

// C6：FOUND_FACTION 无阶段前置 (unlock {})，可用性由派生威望 + 意识形态
// 强度双闸决定；执行走统一 mutation 管线创立零成员派系并扣 AP。
func TestFoundFactionGateAndExecution(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "gate-found-faction.db"))
	if _, err := svc.CreateGame(threeSiloRequest(107)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	setPlayerIdeology := func(pf float64) {
		for i := range svc.silo.Professions {
			if svc.silo.Professions[i].Name == svc.agent.Profession {
				if svc.silo.Professions[i].Ideologies == nil {
					svc.silo.Professions[i].Ideologies = map[string]float64{}
				}
				svc.silo.Professions[i].Ideologies[model.IdeologyProForeign] = pf
				return
			}
		}
		t.Fatal("player profession not found")
	}
	svc.agent.Ambition = 0.6
	svc.agent.Influence = 0.5 // prestige 55
	setPlayerIdeology(0.9)    // intensity 0.4
	svc.refreshActionProgress()
	if action := actionByID(svc.availablePlayerActions(), "action_found_faction"); action == nil || !action.Enabled {
		t.Fatalf("prestige 55 + intensity 0.4 must enable founding: %+v", action)
	}

	setPlayerIdeology(0.6) // intensity 0.1
	svc.refreshActionProgress()
	if action := actionByID(svc.availablePlayerActions(), "action_found_faction"); action != nil && action.Enabled {
		t.Fatalf("low intensity must hide or disable founding: %+v", action)
	}
	setPlayerIdeology(0.9)
	svc.agent.Ambition = 0.4 // prestige 45
	svc.refreshActionProgress()
	if action := actionByID(svc.availablePlayerActions(), "action_found_faction"); action != nil && action.Enabled {
		t.Fatalf("low prestige must hide or disable founding: %+v", action)
	}

	svc.agent.Ambition = 0.6
	svc.agent.ActionPoints = 25
	svc.refreshActionProgress()
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionFoundFaction})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("found faction execution failed: %+v err=%v", outcome, err)
	}
	var founded *model.Faction
	for i := range svc.silo.Factions {
		f := &svc.silo.Factions[i]
		if f.RepresentativeKind == "PLAYER" && f.RepresentativeID == svc.agent.ID {
			founded = f
		}
	}
	if founded == nil || founded.MemberCount != 0 || founded.FounderKind != "PLAYER" {
		t.Fatalf("execution must create a zero-member player-founded faction: %+v", founded)
	}
	if svc.agent.ActionPoints != 5 {
		t.Fatalf("AP = %v, want 25-20=5", svc.agent.ActionPoints)
	}
}

func TestActionGateAPReasonKeepsFraction(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "gate-ap-fraction.db"))
	if _, err := svc.CreateGame(threeSiloRequest(106)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	svc.world.UnlockedActions = append(svc.world.UnlockedActions, "action_share_info")
	svc.agent.ActionPoints = 19.925
	meta, ok := svc.applyActionGate(model.PlayerActionMeta{ID: "action_share_info", ActionType: model.ActionShareInfo, APCost: 20, Enabled: true})
	if !ok || meta.Enabled || !strings.Contains(meta.DisabledReason, "19.93") || strings.Contains(meta.DisabledReason, "当前只有 20 AP") {
		t.Fatalf("AP reason hides the real shortage: %+v", meta)
	}
}

func TestSyncRootContentConfigsRefreshesExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"action_gates.json", "relics.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("stale"), 0o644); err != nil {
			t.Fatalf("seed stale %s: %v", name, err)
		}
	}
	gateContent := []byte(`{"version":1}`)
	relicContent := []byte(`{"version":1,"spawn":{"min":0,"max":0},"locations":[],"relics":[]}`)
	embedded := fstest.MapFS{
		"events/action_gates.json": {Data: gateContent},
		"events/relics.json":       {Data: relicContent},
	}
	if err := syncRootContentConfigs(embedded, dir); err != nil {
		t.Fatalf("sync configs: %v", err)
	}
	for name, content := range map[string][]byte{"action_gates.json": gateContent, "relics.json": relicContent} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != string(content) {
			t.Fatalf("packaged %s was not refreshed: %q err=%v", name, got, err)
		}
	}
}

func TestSyncRootContentConfigsFailsClosedOnMissingEmbeddedFile(t *testing.T) {
	dir := t.TempDir()
	embedded := fstest.MapFS{"events/action_gates.json": {Data: []byte(`{"version":1}`)}}
	if err := syncRootContentConfigs(embedded, dir); err == nil || !strings.Contains(err.Error(), "relics.json") {
		t.Fatalf("missing embedded relic catalog must fail the sync, got: %v", err)
	}
}

func TestMissingActionGateConfigFailsClosed(t *testing.T) {
	svc := &GameService{}
	root := t.TempDir()
	err := svc.loadActionGates(map[string]string{"mechanics": filepath.Join(root, "events", "mechanics")})
	if err == nil || !svc.actionGates.Enabled || svc.actionGates.DefaultVisibility != model.ActionLocked {
		t.Fatalf("missing config must fail closed with an explicit error: runtime=%+v err=%v", svc.actionGates, err)
	}
}

func TestActionGateCompilerRejectsUnknownMetric(t *testing.T) {
	_, err := compileActionGates(actionGateFile{
		Version: 1, InitialStage: "arrival", Stages: []actionStage{{ID: "arrival"}},
		Actions: []actionGate{{ID: "bad", Available: actionCondition{Metric: "invented"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported metric") {
		t.Fatalf("unknown condition metric should fail content validation: %v", err)
	}
}

package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

func threeSiloRequest(seed int64) model.CreateGameRequest {
	return model.CreateGameRequest{
		SiloName:     "ASSIGNED_STATION",
		AgentName:    "Player",
		Profession:   "IT",
		Seed:         seed,
		IncludeSilo1: true,
		Neighbor:     &model.NeighborSiloConfig{},
	}
}

// TestPeerSilosStartWithDivergentResources locks 每堡自己的节奏：三座堡的
// 初始资源库存互不相同（驻地标准、1 号充足、18 号老化），资源阈值类危机
// 因此不会三堡同月齐发。差异是确定性常量，不消耗 RNG。
func TestPeerSilosStartWithDivergentResources(t *testing.T) {
	amount := func(silo *model.Silo, resourceType string) float64 {
		for _, resource := range silo.Resources {
			if resource.Type == resourceType {
				return resource.Amount
			}
		}
		return -1
	}
	station := createAssignedStation(99, 122, nil)
	silo1 := createSilo1(122)
	silo18 := createSilo18(122)
	for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
		a, b, c := amount(station, resourceType), amount(silo1, resourceType), amount(silo18, resourceType)
		if a == b || b == c || a == c {
			t.Errorf("%s: initial amounts must diverge across silos (station=%v silo1=%v silo18=%v)", resourceType, a, b, c)
		}
	}
	if amount(silo18, "Materials") >= amount(station, "Materials") {
		t.Errorf("aging silo 18 must start with less material than the station")
	}
	if amount(silo1, "Materials") <= amount(station, "Materials") {
		t.Errorf("central silo 1 must start with more material than the station")
	}
}

func TestFullSilo1RunsAsPeerAndEmitsSignals(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "peer.db"))
	if _, err := svc.CreateGame(threeSiloRequest(51)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if len(svc.silos) != 3 {
		t.Fatalf("expected three full silos, got %d", len(svc.silos))
	}
	peer := svc.fullSilo1()
	if peer == nil || peer.ID != 2 {
		t.Fatalf("expected full Silo 1 at internal ID 2, got %+v", peer)
	}
	if svc.siloByID(3) == nil || svc.silo18Canon() == nil {
		t.Fatal("expected Silo 18 to remain a separate full silo")
	}
	foundArchives := false
	for _, profession := range peer.Professions {
		if profession.Name == "Archives" {
			foundArchives = true
		}
	}
	if !foundArchives {
		t.Fatal("Silo 1 does not have its distinct Archives department")
	}

	// 错线修复后：peer 模式下带压力条件的通牒必须签发 (反应链读的就是它)，
	// 而压力回落时不签发——阈值脚本两侧都验。旧口径"peer 模式不跑通牒脚本"
	// 正是被修复的错线 (通牒无人签发、反应链永远读不到)。
	svc.silo.Rebellion = 0.95
	svc.agent.PropagandaLevel = 3 // 有组织煽动条件满足
	var logs []string
	var stories []model.StoryEvent
	svc.runSilo1StateMachine(&logs, &stories)
	if len(svc.world.CrossQueue) != 1 || svc.world.CrossQueue[0].EventID != "silo1_ultimatum_cross" ||
		svc.world.Silo1.OrderPhase != "ULTIMATUM" {
		t.Fatalf("conditional ultimatum must fire in peer mode under pressure: queue=%+v phase=%s",
			svc.world.CrossQueue, svc.world.Silo1.OrderPhase)
	}
	svcLow, _, _ := weightedEndingFixture("TIME")
	svcLow.silo.Rebellion = 0.1
	svcLow.world.Silo1.OrderPhase = "LOCAL_SIM"
	logsLow, storiesLow := []string{}, []model.StoryEvent{}
	svcLow.runSilo1StateMachine(&logsLow, &storiesLow)
	if len(svcLow.world.CrossQueue) != 0 || svcLow.world.Silo1.OrderPhase != "LOCAL_SIM" {
		t.Fatalf("low pressure must not issue an ultimatum in peer mode: queue=%+v phase=%s",
			svcLow.world.CrossQueue, svcLow.world.Silo1.OrderPhase)
	}

	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if peer.CurrentYear != 122 || peer.CurrentMonth != 2 {
		t.Fatalf("Silo 1 did not advance its local clock: Y%d M%d", peer.CurrentYear, peer.CurrentMonth)
	}
	if svc.agents[peer.ID].ActionPoints == 50 {
		t.Fatal("Silo 1 AI operator did not use the common action pipeline")
	}
	if svc.world.Silo1.LastReport == "" {
		t.Fatal("expected an intelligence report derived from Silo 1 departments")
	}
	foundSignal := false
	for _, entry := range svc.eventLogs {
		if entry.SiloID == svc.world.PlayerSiloID && entry.EventID == silo1SignalID {
			foundSignal = true
		}
	}
	if !foundSignal {
		t.Fatal("assigned station did not receive the Silo 1 signal")
	}
}

func TestFullSilo1SaveResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "peer-resume.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(52)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(2); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if err := svc.SaveGame(); err != nil {
		t.Fatalf("save game: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(fresh.silos) != 3 || fresh.fullSilo1() == nil {
		t.Fatalf("full Silo 1 was not restored: %d silos", len(fresh.silos))
	}
	if fresh.siloByID(3) == nil || fresh.fullSilo1().CurrentMonth != 3 {
		t.Fatalf("unexpected restored world: %+v", fresh.silos)
	}
}

func TestThreeSiloDoubleRunByteIdentical(t *testing.T) {
	run := func() string {
		svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "determinism.db"))
		if _, err := svc.CreateGame(threeSiloRequest(53)); err != nil {
			t.Fatalf("create game: %v", err)
		}
		if _, err := svc.PassTime(12); err != nil {
			t.Fatalf("pass time: %v", err)
		}
		data, err := json.Marshal(svc.snapshotTwoSilo(53, 12))
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}
		return string(data)
	}
	a, b := run(), run()
	if a != b {
		t.Fatalf("three-silo runs diverged\nrun1=%s\nrun2=%s", a, b)
	}
}

func TestSyncSilo1PeerCanonDestructionOverridesOtherEnding(t *testing.T) {
	peer := &model.Silo{VictoryStatus: &model.VictoryStatus{IsWon: true, Type: "information"}}
	svc := &GameService{
		world: &model.World{Silo1: model.Silo1State{Alive: false}, Flags: map[string]bool{}},
		silos: []*model.Silo{{Traits: []string{silo1FullTrait}, VictoryStatus: peer.VictoryStatus}},
	}
	svc.syncSilo1Peer()
	peer = svc.fullSilo1()
	if svc.world.Silo1.Alive || peer.VictoryStatus == nil || peer.VictoryStatus.Type != "DEATH" {
		t.Fatalf("canon destruction was overwritten by a local ending: alive=%v status=%+v", svc.world.Silo1.Alive, peer.VictoryStatus)
	}
}

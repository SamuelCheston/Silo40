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

	// The legacy omniscient ultimatum script must not run for the peer model.
	svc.silo.Rebellion = 0.95
	var logs []string
	var stories []model.StoryEvent
	svc.runSilo1StateMachine(&logs, &stories)
	if len(svc.world.CrossQueue) != 0 || svc.world.Silo1.OrderPhase != "LOCAL_SIM" {
		t.Fatalf("legacy Silo 1 privilege script fired in peer mode: queue=%d phase=%s", len(svc.world.CrossQueue), svc.world.Silo1.OrderPhase)
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

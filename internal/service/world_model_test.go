package service

import (
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// World skeleton: CreateGame produces a world, persist/Resume round-trips it,
// the world clock mirrors the player silo clock, and GameState exposes the
// world view.
func TestWorldCreatedAndRoundTrips(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "world.db")
	svc := newVersionTestService(t, dbPath)
	createVersionTestGame(t, svc)

	if svc.world == nil {
		t.Fatal("expected CreateGame to produce a world")
	}
	if svc.world.Seed != svc.silo.RNGSeed {
		t.Errorf("world seed %d != silo rng seed %d", svc.world.Seed, svc.silo.RNGSeed)
	}
	if svc.world.PlayerSiloID != ActiveSiloID {
		t.Errorf("world player silo = %d, want %d", svc.world.PlayerSiloID, ActiveSiloID)
	}
	if svc.world.Pacing != model.DefaultWorldPacing() {
		t.Errorf("world pacing = %+v, want defaults", svc.world.Pacing)
	}
	if !svc.world.Silo1.Alive || svc.world.Silo1.OrderPhase != "DORMANT" {
		t.Errorf("world silo1 state = %+v, want alive DORMANT", svc.world.Silo1)
	}
	if len(svc.world.Topology) != 0 {
		t.Errorf("world topology must start empty, got %v", svc.world.Topology)
	}
	if svc.silo.WorldID != ActiveWorldID {
		t.Errorf("silo.WorldID = %d, want %d", svc.silo.WorldID, ActiveWorldID)
	}

	// Advance time: the world clock mirrors the player silo clock.
	if _, err := svc.PassTime(3); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if svc.world.CurrentYear != svc.silo.CurrentYear || svc.world.CurrentMonth != svc.silo.CurrentMonth {
		t.Errorf("world clock %d-%d != silo clock %d-%d after PassTime",
			svc.world.CurrentYear, svc.world.CurrentMonth, svc.silo.CurrentYear, svc.silo.CurrentMonth)
	}

	// Round-trip: persist, resume in a fresh service, world intact.
	if err := svc.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	fresh := newVersionTestService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if fresh.world == nil {
		t.Fatal("resume lost the world")
	}
	if fresh.world.Seed != svc.world.Seed ||
		fresh.world.CurrentYear != svc.world.CurrentYear ||
		fresh.world.CurrentMonth != svc.world.CurrentMonth ||
		fresh.world.PlayerSiloID != svc.world.PlayerSiloID {
		t.Errorf("resumed world %+v != persisted world %+v", fresh.world, svc.world)
	}
	if fresh.world.Pacing != svc.world.Pacing {
		t.Errorf("resumed pacing %+v != persisted pacing %+v", fresh.world.Pacing, svc.world.Pacing)
	}

	state, err := fresh.GetState()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.World == nil {
		t.Fatal("game state must expose the world view")
	}
	if state.World.PlayerSiloID != ActiveSiloID || len(state.World.Silos) != 1 || !state.World.Silos[0].IsPlayer {
		t.Errorf("unexpected world view: %+v", state.World)
	}
}

// Pacing customization: requested values are clamped into range, zero values
// fall back to the 1.0 defaults (legacy-compatible behavior).
func TestWorldPacingNormalized(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pacing.db")
	svc := newVersionTestService(t, dbPath)
	if _, err := svc.CreateGame(model.CreateGameRequest{
		SiloName:   "Pacing",
		StartYear:  122,
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       11,
		Pacing:     &model.WorldPacing{ResourceRate: 9.9, IdeologyRate: 0, PoliticalRate: 0.1},
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	want := model.WorldPacing{
		ResourceRate:   2.0, // clamped from 9.9
		IdeologyRate:   1.0, // zero -> default
		PoliticalRate:  0.5, // clamped from 0.1
		ActionRate:     1.0,
		DemographyRate: 1.0,
	}
	if svc.world.Pacing != want {
		t.Fatalf("world pacing = %+v, want %+v", svc.world.Pacing, want)
	}
}

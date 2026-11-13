package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

func TestFloorStabilityContentEffect(t *testing.T) {
	silo := CreateInitialSilo("Floors", 122, nil)

	// Zone-scoped delta touches only Lower floors.
	ApplyContentEffects([]ContentEffect{
		{Type: "floor_stability_delta", Zone: "Lower", Value: -0.3},
	}, silo)
	for _, floor := range silo.Floors {
		want := 1.0
		if floor.Zone == "Lower" {
			want = 0.7
		}
		if floor.Stability != want {
			t.Fatalf("floor %d zone %s stability = %v, want %v", floor.Level, floor.Zone, floor.Stability, want)
		}
	}

	// Empty zone touches all floors; clamped at 0.
	ApplyContentEffects([]ContentEffect{
		{Type: "floor_stability_delta", Value: -0.9},
	}, silo)
	for _, floor := range silo.Floors {
		want := 0.1 // 1.0 - 0.9 for non-Lower floors
		if floor.Zone == "Lower" {
			want = 0 // 0.7 - 0.9 clamped
		}
		if floor.Stability < want-1e-9 || floor.Stability > want+1e-9 {
			t.Fatalf("floor %d zone %s stability = %v, want %v", floor.Level, floor.Zone, floor.Stability, want)
		}
	}
}

func setZoneStability(silo *model.Silo, zone string, value float64) {
	for i := range silo.Floors {
		if silo.Floors[i].Zone == zone {
			silo.Floors[i].Stability = value
		}
	}
}

func runResourceUpdateOnce(t *testing.T, silo *model.Silo) {
	t.Helper()
	SetRNG(rand.New(rand.NewSource(5)))
	e := NewGameEngine()
	e.runEventMechanic(EVENT_RESOURCE_UPDATE, e.CreateResourceUpdateEvent(silo, 1.0), NewEventContext())
}

// TestFloorStabilityAffectsProduction locks the production consumption layer:
// degraded Lower-zone stability must reduce output. Compared via the Supplies
// net rate (food chain), because Energy net balance is structurally pinned to
// -stockpile while agriculture is energy-limited, and Materials stock drains
// to zero either way.
func TestFloorStabilityAffectsProduction(t *testing.T) {
	pristine := CreateInitialSilo("Pristine", 122, nil)
	runResourceUpdateOnce(t, pristine)
	pristineRate := resourceNet(pristine, "Supplies")

	degraded := CreateInitialSilo("Degraded", 122, nil)
	setZoneStability(degraded, "Lower", 0.2)
	runResourceUpdateOnce(t, degraded)
	degradedRate := resourceNet(degraded, "Supplies")

	if degradedRate >= pristineRate {
		t.Fatalf("degraded Lower stability must reduce Supplies net rate: pristine=%v degraded=%v", pristineRate, degradedRate)
	}
}

// TestFloorStabilityAffectsDeaths locks the mortality consumption layer:
// average stability below 0.8 adds deaths; a pristine silo keeps the base
// rate (neutral baseline).
func TestFloorStabilityAffectsDeaths(t *testing.T) {
	pristine := CreateInitialSilo("Pristine", 122, nil)
	pristine.Resources = []model.Resource{{Type: "Energy", Amount: 5000}, {Type: "Materials", Amount: 2000}, {Type: "Supplies", Amount: 5000}}
	e := NewGameEngine()
	e.runEventMechanic(EVENT_METRICS_UPDATE, e.CreateMetricsUpdateEvent(pristine, 1.0), NewEventContext())
	pristineDeaths := 10000 - pristine.TotalPopulation
	if pristineDeaths != 10 { // 10000 * 0.001 * 1 year
		t.Fatalf("pristine silo should lose exactly the base-rate 10 people, lost %d", pristineDeaths)
	}

	degraded := CreateInitialSilo("Degraded", 122, nil)
	degraded.Resources = []model.Resource{{Type: "Energy", Amount: 5000}, {Type: "Materials", Amount: 2000}, {Type: "Supplies", Amount: 5000}}
	for i := range degraded.Floors {
		degraded.Floors[i].Stability = 0.4
	}
	e.runEventMechanic(EVENT_METRICS_UPDATE, e.CreateMetricsUpdateEvent(degraded, 1.0), NewEventContext())
	degradedDeaths := 10000 - degraded.TotalPopulation
	// 0.001 + (0.8-0.4)*0.25 = 0.101 -> 1010 deaths/year.
	if degradedDeaths <= pristineDeaths {
		t.Fatalf("degraded stability must increase deaths: pristine=%d degraded=%d", pristineDeaths, degradedDeaths)
	}
	if degradedDeaths != 1010 {
		t.Fatalf("expected 1010 deaths at stability 0.4, got %d", degradedDeaths)
	}
}

func resourceNet(silo *model.Silo, resType string) float64 {
	for _, r := range silo.Resources {
		if r.Type == resType {
			return r.NetBalance
		}
	}
	return 0
}

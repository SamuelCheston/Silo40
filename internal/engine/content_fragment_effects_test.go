package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

func fragmentTestSilo() *model.Silo {
	return &model.Silo{
		Professions: []model.Profession{
			{Name: "IT", KnownFragments: []string{"IT_1", "IT_2", "IT_3", "IT_4", "IT_5"}},
			{Name: "Police", KnownFragments: []string{"Police_1", "Police_2"}},
			{Name: "Agricultural", KnownFragments: []string{"Agricultural_1"}},
		},
	}
}

func TestContentFragmentAddExplicit(t *testing.T) {
	SetRNG(nil)
	silo := fragmentTestSilo()

	ApplyContentEffects([]ContentEffect{
		{Type: "profession_fragment_add", Profession: "Police", Fragment: "Mayor_1"},
	}, silo)

	police := findDept(silo, "Police")
	if police == nil || len(police.KnownFragments) != 3 || police.KnownFragments[2] != "Mayor_1" {
		t.Fatalf("expected Mayor_1 appended to Police, got %+v", police)
	}

	// Duplicate add is a no-op.
	ApplyContentEffects([]ContentEffect{
		{Type: "profession_fragment_add", Profession: "Police", Fragment: "Mayor_1"},
	}, silo)
	if len(findDept(silo, "Police").KnownFragments) != 3 {
		t.Fatalf("duplicate fragment add must be a no-op")
	}
}

func TestContentFragmentAddLaggardAndRandomPick(t *testing.T) {
	SetRNG(rand.New(rand.NewSource(99)))
	silo := fragmentTestSilo()

	// Empty profession targets the laggard department (Agricultural, 1 fragment);
	// empty fragment draws a previously unknown one from the engine RNG.
	ApplyContentEffects([]ContentEffect{
		{Type: "profession_fragment_add"},
	}, silo)

	agri := findDept(silo, "Agricultural")
	if agri == nil || len(agri.KnownFragments) != 2 {
		t.Fatalf("expected laggard Agricultural to gain one fragment, got %+v", agri)
	}
	added := agri.KnownFragments[1]
	if added == "" || added == "Agricultural_1" {
		t.Fatalf("expected a new fragment, got %q", added)
	}
	known := false
	for _, f := range model.ALL_FRAGMENTS {
		if f == added {
			known = true
		}
	}
	if !known {
		t.Fatalf("added fragment %q is not in ALL_FRAGMENTS", added)
	}
}

func TestContentFragmentRemove(t *testing.T) {
	silo := fragmentTestSilo()

	ApplyContentEffects([]ContentEffect{
		{Type: "profession_fragment_remove", Profession: "Police", Fragment: "Police_2"},
	}, silo)
	police := findDept(silo, "Police")
	if len(police.KnownFragments) != 1 || police.KnownFragments[0] != "Police_1" {
		t.Fatalf("expected Police_2 removed, got %+v", police.KnownFragments)
	}

	// Removing an unknown fragment is a no-op.
	ApplyContentEffects([]ContentEffect{
		{Type: "profession_fragment_remove", Profession: "Police", Fragment: "IT_5"},
	}, silo)
	if len(findDept(silo, "Police").KnownFragments) != 1 {
		t.Fatalf("removing an unknown fragment must be a no-op")
	}
}

func TestContentFragmentEffectValidation(t *testing.T) {
	if err := validateEffect(ContentEffect{Type: "profession_fragment_add"}); err != nil {
		t.Fatalf("profession_fragment_add with empty fields must validate: %v", err)
	}
	if err := validateEffect(ContentEffect{Type: "profession_fragment_remove", Profession: "Police"}); err == nil {
		t.Fatalf("profession_fragment_remove without fragment must fail validation")
	}
	if err := validateEffect(ContentEffect{Type: "profession_fragment_remove", Profession: "Police", Fragment: "Police_1"}); err != nil {
		t.Fatalf("profession_fragment_remove with both fields must validate: %v", err)
	}
}

// TestFragmentBridgeMakesInformationEndingReachable proves the end-to-end
// path: repeated laggard-targeted fragment grants (the content effect
// semantics) lift every department to >= 5 known fragments, which the
// victory check then resolves as INFORMATION.
func TestFragmentBridgeMakesInformationEndingReachable(t *testing.T) {
	SetRNG(rand.New(rand.NewSource(7)))
	silo := CreateInitialSilo("FragmentReach", 122, nil)
	agent := CreateInitialAgent("Agent", "IT", nil, silo)

	for i := 0; i < 60; i++ {
		allReady := true
		for _, p := range silo.Professions {
			if len(p.KnownFragments) < 5 {
				allReady = false
				break
			}
		}
		if allReady {
			break
		}
		ApplyContentEffects([]ContentEffect{
			{Type: "profession_fragment_add"},
		}, silo)
	}
	for _, p := range silo.Professions {
		if len(p.KnownFragments) < 5 {
			t.Fatalf("department %s still below 5 fragments after grants: %v", p.Name, p.KnownFragments)
		}
	}

	e := NewGameEngine()
	e.checkVictoryConditions(silo, agent)
	if silo.VictoryStatus == nil || silo.VictoryStatus.Type != "INFORMATION" {
		t.Fatalf("expected INFORMATION ending once all departments hold >= 5 fragments, got %+v", silo.VictoryStatus)
	}
}

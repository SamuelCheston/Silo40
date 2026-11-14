package engine

import (
	"strings"
	"testing"
	"unicode/utf8"

	"silo40/internal/model"
)

// Contract: the JS mechanic definitions are the single source of truth for
// built-in actions. At engine construction applyMechanicMetadata overrides
// the Go-side tables (ACTION_COSTS / ACTION_DURATIONS / PROFESSION_ACTIONS)
// from the JS definitions; these tests fail loudly if the override did not
// happen or the two sides drift apart.

func TestContractCommonActionsJSOverridesGoTables(t *testing.T) {
	e := NewGameEngine()
	if e.MechanicEngine == nil {
		t.Fatal("mechanic engine unavailable")
	}
	defs := e.MechanicEngine.CommonActionDefinitions()
	if len(defs) == 0 {
		t.Fatal("no JS common action definitions loaded")
	}
	byType := map[model.AgentActionType]MechanicDefinition{}
	for _, def := range defs {
		byType[model.AgentActionType(def.ActionType)] = def
	}
	for actionType, def := range byType {
		if cost := model.ACTION_COSTS[actionType]; def.APCost > 0 && cost != float64(def.APCost) {
			t.Errorf("%s: ACTION_COSTS=%v but JS ap_cost=%d (override drift)", actionType, cost, def.APCost)
		}
		if dur := model.ACTION_DURATIONS[actionType]; dur != def.DurationMonths {
			t.Errorf("%s: ACTION_DURATIONS=%v but JS duration_months=%d (override drift)", actionType, dur, def.DurationMonths)
		}
	}
	// Every built-in executable action type must be backed by a JS definition.
	for _, actionType := range []model.AgentActionType{
		model.ActionGatherInfo, model.ActionShareInfo, model.ActionBuildConnection,
		model.ActionInciteRebellion, model.ActionConductPropaganda, model.ActionPublicizeFaction,
	} {
		if _, ok := byType[actionType]; !ok {
			t.Errorf("built-in action %s has no JS definition", actionType)
		}
	}
}

// 开局自解释契约：通用动作描述必须承载"这个动作在目标里意味着什么"，
// 不能退回到纯机制速记（"Probe a department" 时代）。描述是玩家理解
// "我为什么要做这个"的唯一界面通道之一。
func TestContractCommonActionDescriptionsCarryPurpose(t *testing.T) {
	e := NewGameEngine()
	if e.MechanicEngine == nil {
		t.Fatal("mechanic engine unavailable")
	}
	defs := e.MechanicEngine.CommonActionDefinitions()
	if len(defs) == 0 {
		t.Fatal("no JS common action definitions loaded")
	}
	for _, def := range defs {
		if strings.TrimSpace(def.Description) == "" {
			t.Errorf("%s: description is empty", def.ID)
			continue
		}
		if n := utf8.RuneCountInString(def.Description); n < 24 {
			t.Errorf("%s: description too short to carry purpose (%d runes): %q", def.ID, n, def.Description)
		}
	}
}

func TestContractProfessionActionsJSOverridesGoTable(t *testing.T) {
	e := NewGameEngine()
	if e.MechanicEngine == nil {
		t.Fatal("mechanic engine unavailable")
	}
	defs := e.MechanicEngine.ProfessionActionDefinitions()
	if len(defs) == 0 {
		t.Fatal("no JS profession action definitions loaded")
	}
	if len(PROFESSION_ACTIONS) != len(defs) {
		t.Fatalf("PROFESSION_ACTIONS has %d entries, JS definitions have %d", len(PROFESSION_ACTIONS), len(defs))
	}
	byID := map[string]MechanicDefinition{}
	for _, def := range defs {
		byID[def.ProfessionAction] = def
	}
	for _, action := range PROFESSION_ACTIONS {
		def, ok := byID[action.ID]
		if !ok {
			t.Errorf("PROFESSION_ACTIONS entry %s missing from JS definitions", action.ID)
			continue
		}
		if action.APCost != float64(def.APCost) {
			t.Errorf("%s: meta APCost=%v but JS ap_cost=%d", action.ID, action.APCost, def.APCost)
		}
		if string(action.TargetType) != def.TargetType {
			t.Errorf("%s: meta TargetType=%v but JS target_type=%q", action.ID, action.TargetType, def.TargetType)
		}
		if action.Profession != def.Profession {
			t.Errorf("%s: meta Profession=%v but JS profession=%q", action.ID, action.Profession, def.Profession)
		}
		if action.SuspicionPenalty != def.SuspicionPenalty {
			t.Errorf("%s: meta SuspicionPenalty=%v but JS suspicion_penalty=%v", action.ID, action.SuspicionPenalty, def.SuspicionPenalty)
		}
	}
}

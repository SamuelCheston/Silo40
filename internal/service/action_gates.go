package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"silo40/internal/model"
)

// actionCondition is a recursive content-declared predicate. An empty node is true.
type actionCondition struct {
	All            []actionCondition `json:"all,omitempty"`
	Any            []actionCondition `json:"any,omitempty"`
	EventCompleted string            `json:"event_completed,omitempty"`
	StageAtLeast   string            `json:"stage_at_least,omitempty"`
	Metric         string            `json:"metric,omitempty"`
	Relation       string            `json:"relation,omitempty"`
	Min            *float64          `json:"min,omitempty"`
	Max            *float64          `json:"max,omitempty"`
}

type actionStage struct {
	ID   string          `json:"id"`
	When actionCondition `json:"when"`
}

type actionGate struct {
	ID         string          `json:"id,omitempty"`
	ActionType string          `json:"action_type,omitempty"`
	Unlock     actionCondition `json:"unlock"`
	Available  actionCondition `json:"available"`
	Reason     string          `json:"reason,omitempty"`
}

type actionGateFile struct {
	Version           int                    `json:"version"`
	InitialStage      string                 `json:"initial_stage"`
	DefaultVisibility model.ActionVisibility `json:"default_visibility"`
	Stages            []actionStage          `json:"stages"`
	Actions           []actionGate           `json:"actions"`
}

type actionGateRuntime struct {
	Enabled           bool
	InitialStage      string
	DefaultVisibility model.ActionVisibility
	Stages            []actionStage
	Actions           []actionGate
	byID              map[string]actionGate
	byType            map[string]actionGate
}

func actionGateConfigPath(dirs map[string]string) (string, error) {
	if dir := dirs["mechanics"]; dir != "" {
		return filepath.Join(filepath.Dir(dir), "action_gates.json"), nil
	}
	groups := make([]string, 0, len(dirs))
	for group := range dirs {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	if len(groups) == 0 {
		return "", fmt.Errorf("no content directories for action gates")
	}
	return filepath.Join(filepath.Dir(dirs[groups[0]]), "action_gates.json"), nil
}

// loadActionGates is production-only content bootstrapping. Legacy mechanism
// tests that do not call it retain their original ungated fixtures.
func (s *GameService) loadActionGates(dirs map[string]string) error {
	path, err := actionGateConfigPath(dirs)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Fail closed in production: a missing registry must never expose every action.
		s.actionGates = actionGateRuntime{Enabled: true, DefaultVisibility: model.ActionLocked}
		return fmt.Errorf("required action gate config is missing: %s", path)
	}
	if err != nil {
		return fmt.Errorf("read action gates: %w", err)
	}
	var file actionGateFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return fmt.Errorf("parse action gates %s: %w", path, err)
	}
	runtime, err := compileActionGates(file)
	if err != nil {
		return fmt.Errorf("validate action gates %s: %w", path, err)
	}
	s.actionGates = runtime
	return nil
}

func compileActionGates(file actionGateFile) (actionGateRuntime, error) {
	if file.Version != 1 {
		return actionGateRuntime{}, fmt.Errorf("unsupported version %d", file.Version)
	}
	if file.DefaultVisibility == "" {
		file.DefaultVisibility = model.ActionLocked
	}
	if file.DefaultVisibility != model.ActionLocked && file.DefaultVisibility != model.ActionAvailable {
		return actionGateRuntime{}, fmt.Errorf("invalid default visibility %q", file.DefaultVisibility)
	}
	runtime := actionGateRuntime{
		Enabled:           true,
		InitialStage:      strings.TrimSpace(file.InitialStage),
		DefaultVisibility: file.DefaultVisibility,
		Stages:            file.Stages,
		Actions:           file.Actions,
		byID:              map[string]actionGate{},
		byType:            map[string]actionGate{},
	}
	stageIDs := map[string]bool{}
	for _, stage := range file.Stages {
		if strings.TrimSpace(stage.ID) == "" || stageIDs[stage.ID] {
			return actionGateRuntime{}, fmt.Errorf("duplicate or empty stage %q", stage.ID)
		}
		stageIDs[stage.ID] = true
		if err := validateActionCondition(stage.When); err != nil {
			return actionGateRuntime{}, fmt.Errorf("stage %s: %w", stage.ID, err)
		}
	}
	if runtime.InitialStage == "" || !stageIDs[runtime.InitialStage] {
		return actionGateRuntime{}, fmt.Errorf("initial stage %q is not declared", runtime.InitialStage)
	}
	for _, gate := range file.Actions {
		if gate.ID == "" && gate.ActionType == "" {
			return actionGateRuntime{}, fmt.Errorf("action gate requires id or action_type")
		}
		if gate.ID != "" {
			if _, exists := runtime.byID[gate.ID]; exists {
				return actionGateRuntime{}, fmt.Errorf("duplicate action gate %q", gate.ID)
			}
			runtime.byID[gate.ID] = gate
		}
		if gate.ActionType != "" {
			if _, exists := runtime.byType[gate.ActionType]; exists {
				return actionGateRuntime{}, fmt.Errorf("duplicate action type gate %q", gate.ActionType)
			}
			runtime.byType[gate.ActionType] = gate
		}
		if err := validateActionCondition(gate.Unlock); err != nil {
			return actionGateRuntime{}, fmt.Errorf("action %s unlock: %w", gate.ID, err)
		}
		if err := validateActionCondition(gate.Available); err != nil {
			return actionGateRuntime{}, fmt.Errorf("action %s available: %w", gate.ID, err)
		}
	}
	return runtime, nil
}

func validateActionCondition(condition actionCondition) error {
	for _, child := range append(append([]actionCondition(nil), condition.All...), condition.Any...) {
		if err := validateActionCondition(child); err != nil {
			return err
		}
	}
	if condition.Metric != "" && !supportedActionMetric(condition.Metric) {
		return fmt.Errorf("unsupported metric %q", condition.Metric)
	}
	return nil
}

func supportedActionMetric(metric string) bool {
	switch metric {
	case "action_points", "political_prestige", "propaganda_level", "suspicion_level", "known_fragments",
		"legitimacy", "cohesion", "rebellion", "history_burden":
		return true
	default:
		return false
	}
}

func (s *GameService) initialProgressStage() string {
	if s.actionGates.Enabled {
		return s.actionGates.InitialStage
	}
	return ""
}

func (s *GameService) stageIndex(stage string) int {
	for i, candidate := range s.actionGates.Stages {
		if candidate.ID == stage {
			return i
		}
	}
	return -1
}

// refreshActionProgress advances stages and permanently records newly unlocked
// action IDs. It consumes no RNG and never iterates a map to affect outcomes.
func (s *GameService) refreshActionProgress() {
	if !s.actionGates.Enabled || s.world == nil || s.agent == nil {
		return
	}
	if s.stageIndex(s.world.ProgressStage) < 0 {
		s.world.ProgressStage = s.actionGates.InitialStage
	}
	for next := s.stageIndex(s.world.ProgressStage) + 1; next < len(s.actionGates.Stages); next++ {
		if !s.evaluateActionCondition(s.actionGates.Stages[next].When) {
			break
		}
		s.world.ProgressStage = s.actionGates.Stages[next].ID
	}
	unlocked := make(map[string]bool, len(s.world.UnlockedActions))
	for _, id := range s.world.UnlockedActions {
		unlocked[id] = true
	}
	for _, gate := range s.actionGates.Actions {
		key := gate.ID
		if key == "" {
			key = "type:" + gate.ActionType
		}
		if unlocked[key] || !s.evaluateActionCondition(gate.Unlock) {
			continue
		}
		s.world.UnlockedActions = append(s.world.UnlockedActions, key)
		unlocked[key] = true
	}
}

func (s *GameService) evaluateActionCondition(condition actionCondition) bool {
	for _, child := range condition.All {
		if !s.evaluateActionCondition(child) {
			return false
		}
	}
	if len(condition.Any) > 0 {
		matched := false
		for _, child := range condition.Any {
			if s.evaluateActionCondition(child) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if condition.EventCompleted != "" && !s.actionEventCompleted(condition.EventCompleted) {
		return false
	}
	if condition.StageAtLeast != "" {
		current := s.stageIndex(s.world.ProgressStage)
		required := s.stageIndex(condition.StageAtLeast)
		if current < 0 || required < 0 || current < required {
			return false
		}
	}
	value, hasValue := s.actionConditionValue(condition)
	if (condition.Metric != "" || condition.Relation != "") && !hasValue {
		return false
	}
	if condition.Min != nil && (!hasValue || value < *condition.Min) {
		return false
	}
	if condition.Max != nil && (!hasValue || value > *condition.Max) {
		return false
	}
	return true
}

func (s *GameService) actionEventCompleted(eventKey string) bool {
	if s.silo == nil {
		return false
	}
	states := s.contentStates[s.silo.ID]
	if state, ok := states[eventKey]; ok && state.Triggered {
		return true
	}
	for key, state := range states {
		if state.Triggered && strings.HasSuffix(key, ":"+eventKey) {
			return true
		}
	}
	return false
}

func (s *GameService) actionConditionValue(condition actionCondition) (float64, bool) {
	if condition.Relation != "" {
		name := strings.TrimPrefix(condition.Relation, "relation:")
		for _, profession := range s.silo.Professions {
			if profession.Name != name {
				continue
			}
			for _, connection := range s.agent.Connections {
				if connection.ProfessionID == profession.ID {
					return connection.Value, true
				}
			}
			return 0, true
		}
		return 0, false
	}
	switch condition.Metric {
	case "action_points":
		return s.agent.ActionPoints, true
	case "political_prestige":
		return s.agent.PoliticalPrestige, true
	case "propaganda_level":
		return s.agent.PropagandaLevel, true
	case "suspicion_level":
		return s.agent.SuspicionLevel, true
	case "known_fragments":
		return float64(len(s.agent.KnownFragments)), true
	case "legitimacy":
		return s.silo.MayorLegitimacy(), true
	case "cohesion":
		return s.silo.Cohesion, true
	case "rebellion":
		return s.silo.Rebellion, true
	case "history_burden":
		return s.silo.HistoryBurden, true
	default:
		return 0, false
	}
}

func (s *GameService) gateForAction(id string, actionType model.AgentActionType) (actionGate, string, bool) {
	if gate, ok := s.actionGates.byID[id]; ok {
		return gate, id, true
	}
	if gate, ok := s.actionGates.byType[string(actionType)]; ok {
		return gate, "type:" + string(actionType), true
	}
	return actionGate{}, "", false
}

func (s *GameService) actionUnlocked(key string) bool {
	for _, id := range s.world.UnlockedActions {
		if id == key {
			return true
		}
	}
	return false
}

func (s *GameService) applyActionGate(meta model.PlayerActionMeta) (model.PlayerActionMeta, bool) {
	if !s.actionGates.Enabled {
		meta.Visibility = model.ActionAvailable
		if !meta.Enabled {
			meta.Visibility = model.ActionUnlocked
		}
		return meta, true
	}
	s.refreshActionProgress()
	gate, key, declared := s.gateForAction(meta.ID, meta.ActionType)
	if !declared {
		if s.actionGates.DefaultVisibility == model.ActionLocked {
			return meta, false
		}
		if meta.Enabled {
			meta.Visibility = model.ActionAvailable
		} else {
			meta.Visibility = model.ActionUnlocked
		}
		return meta, true
	}
	if !s.actionUnlocked(key) {
		return meta, false
	}
	if !s.evaluateActionCondition(gate.Available) {
		meta.Enabled = false
		meta.Visibility = model.ActionUnlocked
		meta.DisabledReason = gate.Reason
		if meta.DisabledReason == "" {
			meta.DisabledReason = "当前可用条件未满足"
		}
		return meta, true
	}
	if !meta.Enabled {
		meta.Visibility = model.ActionUnlocked
		if meta.DisabledReason == "" {
			meta.DisabledReason = "当前目标或剧情条件未满足"
		}
		return meta, true
	}
	if s.agent.ActionPoints < float64(meta.APCost) {
		meta.Enabled = false
		meta.Visibility = model.ActionUnlocked
		meta.DisabledReason = fmt.Sprintf("需要 %d AP，当前只有 %.2f AP", meta.APCost, s.agent.ActionPoints)
		return meta, true
	}
	meta.Enabled = true
	meta.Visibility = model.ActionAvailable
	meta.DisabledReason = ""
	return meta, true
}

func (s *GameService) actionIDForGate(action model.AgentAction) string {
	switch action.Type {
	case model.ActionPlayerEvent, model.ActionStationOperation, model.ActionSilo1Operation:
		return action.ActionID
	case model.ActionProfession:
		return action.ProfessionAction
	default:
		if s.engine != nil && s.engine.MechanicEngine != nil {
			for _, def := range s.engine.MechanicEngine.CommonActionDefinitions() {
				if def.ActionType == string(action.Type) {
					return def.ID
				}
			}
		}
		return string(action.Type)
	}
}

// validateActionGate is the server-side bypass guard for both execution APIs.
func (s *GameService) validateActionGate(action model.AgentAction) error {
	if !s.actionGates.Enabled {
		return nil
	}
	s.refreshActionProgress()
	id := s.actionIDForGate(action)
	gate, key, declared := s.gateForAction(id, action.Type)
	if !declared {
		if s.actionGates.DefaultVisibility == model.ActionLocked {
			return fmt.Errorf("action %q has not been unlocked", id)
		}
		return nil
	}
	if !s.actionUnlocked(key) {
		return fmt.Errorf("action %q has not been unlocked", id)
	}
	if !s.evaluateActionCondition(gate.Available) {
		if gate.Reason != "" {
			return fmt.Errorf("%s", gate.Reason)
		}
		return fmt.Errorf("action %q is unlocked but currently unavailable", id)
	}
	if s.agent.ActionPoints < action.Cost {
		return fmt.Errorf("not enough Action Points (AP): need %.2f, have %.2f", action.Cost, s.agent.ActionPoints)
	}
	return nil
}

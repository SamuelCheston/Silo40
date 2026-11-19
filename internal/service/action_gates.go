package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// actionCondition is a recursive content-declared predicate. An empty node is true.
type actionCondition struct {
	All            []actionCondition `json:"all,omitempty"`
	Any            []actionCondition `json:"any,omitempty"`
	Not            *actionCondition  `json:"not,omitempty"`
	EventCompleted string            `json:"event_completed,omitempty"`
	StageAtLeast   string            `json:"stage_at_least,omitempty"`
	Metric         string            `json:"metric,omitempty"`
	Relation       string            `json:"relation,omitempty"`
	// Trait 门控型特质条件 (19 §4)：玩家特工持有该特质时成立。只扩大可达
	// 集 (解锁专属动作)，不缩小任何既有路径。
	Trait string   `json:"trait,omitempty"`
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
}

// knownAgentTraits 是门控条件可引用的特工特质封闭集 (与
// engine.CreateInitialAgent 的特质分支一致)；配置里出现集合外的特质名直接
// 校验失败，防止静默写错字让动作永不可解锁。
var knownAgentTraits = map[string]bool{
	"隐秘行事": true,
	"地堡土著": true,
	"魅力非凡": true,
}

type actionStage struct {
	ID   string          `json:"id"`
	When actionCondition `json:"when"`
}

type actionGate struct {
	ID                  string          `json:"id,omitempty"`
	ActionType          string          `json:"action_type,omitempty"`
	Unlock              actionCondition `json:"unlock"`
	Available           actionCondition `json:"available"`
	HideWhenUnavailable bool            `json:"hide_when_unavailable,omitempty"`
	Reason              string          `json:"reason,omitempty"`
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

// actionGateConfigPath lives in relic_catalog.go (shared rootConfigPath).

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
	if condition.Not != nil {
		if err := validateActionCondition(*condition.Not); err != nil {
			return err
		}
	}
	if condition.Metric != "" && !supportedActionMetric(condition.Metric) {
		return fmt.Errorf("unsupported metric %q", condition.Metric)
	}
	if condition.Trait != "" && !knownAgentTraits[condition.Trait] {
		return fmt.Errorf("unsupported trait %q", condition.Trait)
	}
	return nil
}

func supportedActionMetric(metric string) bool {
	switch metric {
	case "action_points", "political_prestige", "propaganda_level", "suspicion_level", "known_fragments",
		"legitimacy", "cohesion", "rebellion", "history_burden",
		// 21 设计卷：谈判层 0/1 门控指标。
		"negotiation_channel", "negotiation_compromise", "negotiation_takeover_decided",
		// 16 §3.5 (Layer 3)：外溢投递侧门控指标——包裹质量与各信道可用性。
		"package_quality", "spillover_channel_it_line", "spillover_channel_cross_signal", "spillover_channel_physical",
		// 20 分册 T3-1b：身份线日常绑定——地点/职务动作的剩余灰置月数，
		// 以及抹除准备期标记 (suspicion ≥ 0.7)。
		"identity_location_lock", "identity_profession_lock", "identity_erasure_stage":
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
	if condition.Not != nil && s.evaluateActionCondition(*condition.Not) {
		return false
	}
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
	if condition.Trait != "" && !s.agentHasTrait(condition.Trait) {
		return false
	}
	// 同阶层免引荐 (19 §4.3)：地堡土著玩家对同阶层目标部门的关系门视为已
	// 满足——抹掉该关系条件与其数值界，其余条件 (阶段/事件/指标) 照常评估。
	if condition.Relation != "" && s.relationGateWaived(condition.Relation) {
		condition.Relation = ""
		condition.Min = nil
		condition.Max = nil
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

// agentHasTrait 玩家特工是否持有指定特质 (中文名，与 CreateInitialAgent 的
// 登记口径一致)。
func (s *GameService) agentHasTrait(trait string) bool {
	if s.agent == nil {
		return false
	}
	for _, item := range s.agent.Traits {
		if item == trait {
			return true
		}
	}
	return false
}

// relationGateWaived 实现「同阶层免引荐」(19 §4.3)：仅当玩家持有地堡土著特
// 质、且关系门目标部门与玩家职业同阶层 (ClassType 相同) 时，该关系门视为
// 已满足。只放宽关系门：阶段门 (first_signal/archive_awakened 等)、部门准
// 入门与宣传门不受影响；口径二元 (视为满足)，无中间态。
func (s *GameService) relationGateWaived(relation string) bool {
	if relation == "" || s.agent == nil || s.silo == nil {
		return false
	}
	if !s.agentHasTrait("地堡土著") {
		return false
	}
	name := strings.TrimPrefix(relation, "relation:")
	var targetClass, agentClass string
	for i := range s.silo.Professions {
		prof := &s.silo.Professions[i]
		if prof.Name == name {
			targetClass = prof.ClassType
		}
		if prof.Name == s.agent.Profession {
			agentClass = prof.ClassType
		}
	}
	return targetClass != "" && targetClass == agentClass
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
		return engine.DerivedPrestige(s.agent.Ambition, s.agent.Influence), true
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
	case "negotiation_channel":
		// 21 设计卷：谈判信道畅通 (开信且未崩塌) 为 1，否则 0。
		if s.silo.Negotiation.ChannelOpen && !s.silo.Negotiation.Closed {
			return 1, true
		}
		return 0, true
	case "negotiation_compromise":
		// 21 设计卷：妥协成立为 1，否则 0。
		if s.silo.Negotiation.CompromiseEstablished {
			return 1, true
		}
		return 0, true
	case "negotiation_takeover_decided":
		// 21 设计卷：妥协后二选一已定为 1，否则 0。
		if s.silo.Negotiation.Takeover != "" {
			return 1, true
		}
		return 0, true
	case "package_quality":
		// 16 §3.5 (Layer 3)：外溢包裹五种能力完备度 (0-1)。
		return s.silo.SpilloverPrep.PackageQuality, true
	case "spillover_channel_it_line":
		return spilloverChannelGateValue(s.silo, model.SpilloverChannelITLine), true
	case "spillover_channel_cross_signal":
		return spilloverChannelGateValue(s.silo, model.SpilloverChannelCrossSignal), true
	case "spillover_channel_physical":
		return spilloverChannelGateValue(s.silo, model.SpilloverChannelPhysicalHandoff), true
	case "identity_location_lock":
		// 20 分册 T3-1b：复核期剩余灰置月数 (>0 = 不可用)。
		return float64(s.silo.Countermeasure.LocationLockMonths), true
	case "identity_profession_lock":
		// 20 分册 T3-1b：复查期剩余灰置月数 (>0 = 不可用)。
		return float64(s.silo.Countermeasure.ProfessionLockMonths), true
	case "identity_erasure_stage":
		// 20 分册 T3-1b：抹除准备期 = 怀疑度 ≥ 0.7 (既有身份线节点口径)。
		if s.agent.SuspicionLevel >= 0.7 {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// spilloverChannelGateValue 该种类未中断信道存在为 1，否则 0 (16 §3.5)。
func spilloverChannelGateValue(silo *model.Silo, kind string) float64 {
	if spilloverChannelForSend(silo, kind) >= 0 {
		return 1
	}
	return 0
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
		if gate.HideWhenUnavailable {
			return meta, false
		}
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

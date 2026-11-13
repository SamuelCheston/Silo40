package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"silo40/internal/model"
)

const (
	ContentFireModeOnce       = "ONCE"
	ContentFireModeRepeatable = "REPEATABLE"
	contentTimestampBaseYear  = 100
	contentDaysPerMonth       = 30
	contentMonthsPerYear      = 12
)

// ContentEventDefinition 文件驱动的剧情事件定义。
type ContentEventDefinition struct {
	ID             uint
	Key            string
	SourceGroup    string
	SourceFile     string
	SourceFormat   string
	EventID        string                   `json:"id"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description"`
	Type           string                   `json:"type"`
	FireMode       string                   `json:"fire_mode"`
	CooldownMonths int                      `json:"cooldown_months,omitempty"`
	Trigger        ContentTrigger           `json:"trigger"`
	Effects        []ContentEffect          `json:"effects"`
	PlayerAction   *ContentPlayerActionSpec `json:"player_action,omitempty"`
	ScriptHooks    ContentScriptHooks       `json:"-"`
	ScriptSource   string                   `json:"-"`
}

// ContentScriptHooks lists which restricted JavaScript hooks are present.
type ContentScriptHooks struct {
	CanTrigger bool
	Apply      bool
}

// ContentTrigger 结构化触发条件。
type ContentTrigger struct {
	Type       string           `json:"type"`
	Metric     string           `json:"metric,omitempty"`
	Flag       string           `json:"flag,omitempty"`
	Ideology   string           `json:"ideology,omitempty"`
	Profession string           `json:"profession,omitempty"`
	Category   string           `json:"category,omitempty"`
	EventID    string           `json:"event_id,omitempty"`
	Action     string           `json:"action,omitempty"`
	Value      float64          `json:"value,omitempty"`
	Count      int              `json:"count,omitempty"`
	Year       int              `json:"year,omitempty"`
	Month      int              `json:"month,omitempty"`
	Timestamp  int              `json:"timestamp,omitempty"`
	Conditions []ContentTrigger `json:"conditions,omitempty"`
}

// ContentEffect 结构化事件效果。
//
// Scope (P0.5 多堡) 决定效果作用的目标域，缺省 "self" = 现状逐位不变：
//   - "self"                 作用于触发事件的那个堡 (向后兼容根基)
//   - "world"                作用于世界状态 (仅 world_flag_set 等世界效果)
//   - "silo:<ref>"           作用于具体堡；ref 可为编号 / "player" / 标签
//   - "neighbors"            按 World.Topology 解析为触发堡的相邻堡集合
//   - "all"                  作用于世界内全部堡
//
// 目标解析在施加期完成 (见 resolveScope)，因为目标堡的存活状态是运行时事实。
type ContentEffect struct {
	Type        string  `json:"type"`
	Scope       string  `json:"scope,omitempty"`
	Metric      string  `json:"metric,omitempty"`
	Flag        string  `json:"flag,omitempty"`
	Ideology    string  `json:"ideology,omitempty"`
	Profession  string  `json:"profession,omitempty"`
	Fragment    string  `json:"fragment,omitempty"`
	Zone        string  `json:"zone,omitempty"`
	Value       float64 `json:"value,omitempty"`
	BoolValue   bool    `json:"bool_value,omitempty"`
	DelayMonths int     `json:"delay_months,omitempty"`
	EventID     string  `json:"event_id,omitempty"`
}

// ContentPlayerActionSpec describes how a player action event is shown in UI.
type ContentPlayerActionSpec struct {
	ID                  string `json:"id,omitempty"`
	Label               string `json:"label,omitempty"`
	Description         string `json:"description,omitempty"`
	Scope               string `json:"scope,omitempty"`
	Profession          string `json:"profession,omitempty"`
	ProfessionGroup     string `json:"profession_group,omitempty"`
	ActionType          string `json:"action_type,omitempty"`
	TargetType          string `json:"target_type,omitempty"`
	APCost              int    `json:"ap_cost,omitempty"`
	DurationMonths      int    `json:"duration_months,omitempty"`
	UnavailableBehavior string `json:"unavailable_behavior,omitempty"`
}

// ContentEventRuntime 单局游戏内的运行时触发状态。
type ContentEventRuntime struct {
	Triggered              bool
	TriggerCount           int
	LastTriggeredTimestamp int
}

// ContentEvaluationContext provides runtime state for trigger evaluation.
//
// World / SignalsReceived (P0.5 多堡)：世界事实与"本月收到的跨堡信号"只在
// 多堡编排时注入；nil/空时所有世界化触发器按不可触发处理，单堡行为不变。
type ContentEvaluationContext struct {
	Silo            *model.Silo
	World           *model.World
	SignalsReceived []string
	Runtime         ContentEventRuntime
	States          map[string]ContentEventRuntime
	Action          *model.AgentAction
}

// LoadContentEventDefinitions 从多个目录加载 JSON 事件定义。
func LoadContentEventDefinitions(dirs map[string]string) ([]ContentEventDefinition, error) {
	var defs []ContentEventDefinition
	for sourceGroup, dir := range dirs {
		entries, err := loadContentEventDir(sourceGroup, dir)
		if err != nil {
			return nil, err
		}
		defs = append(defs, entries...)
	}

	sort.Slice(defs, func(i, j int) bool {
		if defs[i].SourceGroup != defs[j].SourceGroup {
			return defs[i].SourceGroup < defs[j].SourceGroup
		}
		if defs[i].SourceFile != defs[j].SourceFile {
			return defs[i].SourceFile < defs[j].SourceFile
		}
		return defs[i].EventID < defs[j].EventID
	})
	return defs, nil
}

func loadContentEventDir(sourceGroup, dir string) ([]ContentEventDefinition, error) {
	var defs []ContentEventDefinition
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// The action gate registry is content configuration, not an event definition.
		if filepath.Base(path) == "action_gates.json" {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".js":
		default:
			return nil
		}

		fileDefs, err := loadContentEventFile(sourceGroup, dir, path)
		if err != nil {
			return err
		}
		defs = append(defs, fileDefs...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %w", sourceGroup, err)
	}
	return defs, nil
}

func loadContentEventFile(sourceGroup, baseDir, path string) ([]ContentEventDefinition, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js":
		return loadContentEventJSFile(sourceGroup, baseDir, path)
	case ".json":
	default:
		return nil, fmt.Errorf("unsupported content event file extension %q", filepath.Ext(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read content event file %s: %w", path, err)
	}

	var defs []ContentEventDefinition
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		parseErr := json.Unmarshal(data, &defs)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse content event file %s: %w", path, parseErr)
		}
	} else {
		var def ContentEventDefinition
		parseErr := json.Unmarshal(data, &def)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse content event file %s: %w", path, parseErr)
		}
		defs = []ContentEventDefinition{def}
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		relPath = filepath.Base(path)
	}
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for i := range defs {
		if defs[i].EventID == "" {
			defs[i].EventID = baseName
		}
		defs[i].SourceGroup = sourceGroup
		defs[i].SourceFile = filepath.ToSlash(relPath)
		defs[i].SourceFormat = "json"
		defs[i].Key = sourceGroup + ":" + defs[i].EventID
		normalizeContentEventDefinition(&defs[i])
		if err := ValidateContentEventDefinition(defs[i]); err != nil {
			return nil, fmt.Errorf("invalid content event in %s: %w", path, err)
		}
	}
	return defs, nil
}

func normalizeContentEventDefinition(def *ContentEventDefinition) {
	def.FireMode = strings.ToUpper(strings.TrimSpace(def.FireMode))
	if def.FireMode == "" {
		def.FireMode = ContentFireModeOnce
	}
	def.Type = strings.ToUpper(strings.TrimSpace(def.Type))
	normalizeTrigger(&def.Trigger)
	for i := range def.Effects {
		def.Effects[i].Type = strings.ToLower(strings.TrimSpace(def.Effects[i].Type))
		def.Effects[i].Scope = strings.ToLower(strings.TrimSpace(def.Effects[i].Scope))
		def.Effects[i].Metric = strings.ToLower(strings.TrimSpace(def.Effects[i].Metric))
		def.Effects[i].Flag = strings.ToLower(strings.TrimSpace(def.Effects[i].Flag))
		def.Effects[i].Profession = strings.TrimSpace(def.Effects[i].Profession)
		def.Effects[i].Fragment = strings.TrimSpace(def.Effects[i].Fragment)
		def.Effects[i].Zone = strings.TrimSpace(def.Effects[i].Zone)
		def.Effects[i].EventID = strings.TrimSpace(def.Effects[i].EventID)
	}
	normalizePlayerActionSpec(def)
}

func normalizeTrigger(trigger *ContentTrigger) {
	trigger.Type = strings.ToLower(strings.TrimSpace(trigger.Type))
	trigger.Metric = strings.ToLower(strings.TrimSpace(trigger.Metric))
	trigger.Flag = strings.ToLower(strings.TrimSpace(trigger.Flag))
	trigger.Ideology = strings.TrimSpace(trigger.Ideology)
	trigger.Profession = strings.TrimSpace(trigger.Profession)
	trigger.Category = normalizeContentCategory(trigger.Category)
	trigger.EventID = strings.TrimSpace(trigger.EventID)
	trigger.Action = strings.ToUpper(strings.TrimSpace(trigger.Action))
	if trigger.Month == 0 {
		trigger.Month = 1
	}
	for i := range trigger.Conditions {
		normalizeTrigger(&trigger.Conditions[i])
	}
}

func normalizePlayerActionSpec(def *ContentEventDefinition) {
	if def.PlayerAction == nil {
		return
	}
	def.PlayerAction.ID = strings.TrimSpace(def.PlayerAction.ID)
	if def.PlayerAction.ID == "" {
		def.PlayerAction.ID = def.EventID
	}
	def.PlayerAction.Label = strings.TrimSpace(def.PlayerAction.Label)
	if def.PlayerAction.Label == "" {
		def.PlayerAction.Label = def.Title
	}
	def.PlayerAction.Description = strings.TrimSpace(def.PlayerAction.Description)
	if def.PlayerAction.Description == "" {
		def.PlayerAction.Description = def.Description
	}
	def.PlayerAction.Scope = strings.ToLower(strings.TrimSpace(def.PlayerAction.Scope))
	if def.PlayerAction.Scope == "" {
		def.PlayerAction.Scope = "common"
	}
	def.PlayerAction.Profession = strings.TrimSpace(def.PlayerAction.Profession)
	def.PlayerAction.ProfessionGroup = strings.ToUpper(strings.TrimSpace(def.PlayerAction.ProfessionGroup))
	def.PlayerAction.ActionType = strings.ToUpper(strings.TrimSpace(def.PlayerAction.ActionType))
	if def.PlayerAction.ActionType == "" {
		def.PlayerAction.ActionType = string(model.ActionPlayerEvent)
	}
	def.PlayerAction.TargetType = strings.ToUpper(strings.TrimSpace(def.PlayerAction.TargetType))
	if def.PlayerAction.TargetType == "" {
		def.PlayerAction.TargetType = "NONE"
	}
	def.PlayerAction.UnavailableBehavior = strings.ToLower(strings.TrimSpace(def.PlayerAction.UnavailableBehavior))
	if def.PlayerAction.UnavailableBehavior == "" {
		def.PlayerAction.UnavailableBehavior = "hide"
	}
	if def.PlayerAction.APCost < 0 {
		def.PlayerAction.APCost = 0
	}
	if def.PlayerAction.DurationMonths < 0 {
		def.PlayerAction.DurationMonths = 0
	}
}

// ValidateContentEventDefinition 对文件驱动事件做基础校验。
func ValidateContentEventDefinition(def ContentEventDefinition) error {
	if def.EventID == "" {
		return fmt.Errorf("missing id")
	}
	if def.Title == "" {
		return fmt.Errorf("missing title")
	}
	switch def.FireMode {
	case ContentFireModeOnce, ContentFireModeRepeatable:
	default:
		return fmt.Errorf("unsupported fire_mode %q", def.FireMode)
	}
	if def.Trigger.Type != "" {
		if err := validateTrigger(def.Trigger); err != nil {
			return err
		}
	} else if !def.ScriptHooks.CanTrigger {
		return fmt.Errorf("missing trigger or script.canTrigger")
	}
	for _, effect := range def.Effects {
		if err := validateEffect(effect); err != nil {
			return err
		}
	}
	if len(def.Effects) == 0 && !def.ScriptHooks.Apply {
		return fmt.Errorf("missing effects or script.apply")
	}
	if err := validatePlayerActionSpec(def); err != nil {
		return err
	}
	return nil
}

func validatePlayerActionSpec(def ContentEventDefinition) error {
	if def.PlayerAction == nil {
		return nil
	}
	switch def.PlayerAction.Scope {
	case "common", "profession", "profession_group", "faction_member", "faction_leader":
	default:
		return fmt.Errorf("unsupported player_action.scope %q", def.PlayerAction.Scope)
	}
	switch def.PlayerAction.TargetType {
	case "NONE", "DEPT", "RESOURCE":
	default:
		return fmt.Errorf("unsupported player_action.target_type %q", def.PlayerAction.TargetType)
	}
	switch def.PlayerAction.UnavailableBehavior {
	case "hide", "disable":
	default:
		return fmt.Errorf("unsupported player_action.unavailable_behavior %q", def.PlayerAction.UnavailableBehavior)
	}
	if def.PlayerAction.ActionType == "" {
		return fmt.Errorf("player_action.action_type is required")
	}
	if def.PlayerAction.Scope == "profession" && def.PlayerAction.Profession == "" {
		return fmt.Errorf("profession scope requires player_action.profession")
	}
	if def.PlayerAction.Scope == "profession_group" && def.PlayerAction.ProfessionGroup == "" {
		return fmt.Errorf("profession_group scope requires player_action.profession_group")
	}
	return nil
}

func validateTrigger(trigger ContentTrigger) error {
	switch trigger.Type {
	case "all", "any":
		if len(trigger.Conditions) == 0 {
			return fmt.Errorf("%s trigger requires conditions", trigger.Type)
		}
		for _, child := range trigger.Conditions {
			if err := validateTrigger(child); err != nil {
				return err
			}
		}
		return nil
	case "timestamp_at_or_after":
		if trigger.Timestamp == 0 && trigger.Year == 0 {
			return fmt.Errorf("timestamp_at_or_after requires timestamp or year")
		}
		return nil
	case "silo_metric_gte", "silo_metric_lte", "silo_resource_gte", "silo_resource_lte":
		if trigger.Metric == "" {
			return fmt.Errorf("%s requires metric", trigger.Type)
		}
		return nil
	case "silo_population_gte", "silo_population_lte":
		return nil
	case "silo_flag_true", "silo_flag_false":
		if trigger.Flag == "" {
			return fmt.Errorf("%s requires flag", trigger.Type)
		}
		return nil
	case "world_flag_true", "world_flag_false":
		if trigger.Flag == "" {
			return fmt.Errorf("%s requires flag", trigger.Type)
		}
		if !model.ValidWorldFlag(trigger.Flag) {
			return fmt.Errorf("%s flag %q is not in the world flag whitelist", trigger.Type, trigger.Flag)
		}
		return nil
	case "signal_received":
		// 本堡在本月收到了跨堡信号；event_id 可选 (限定具体信号)。
		return nil
	case "event_triggered":
		if trigger.EventID == "" {
			return fmt.Errorf("event_triggered requires event_id")
		}
		return nil
	case "player_action_is":
		if trigger.Action == "" {
			return fmt.Errorf("player_action_is requires action")
		}
		return nil
	case "profession_metric_gte_count", "profession_ideology_gte_count":
		if trigger.Count <= 0 {
			return fmt.Errorf("%s requires positive count", trigger.Type)
		}
		return nil
	default:
		return fmt.Errorf("unsupported trigger type %q", trigger.Type)
	}
}

func validateEffect(effect ContentEffect) error {
	if err := validateEffectScope(effect); err != nil {
		return err
	}
	switch effect.Type {
	case "silo_metric_delta", "silo_resource_delta":
		if effect.Metric == "" {
			return fmt.Errorf("%s requires metric", effect.Type)
		}
	case "silo_population_delta":
		// no extra fields required
	case "profession_metric_delta_all":
		if effect.Metric == "" {
			return fmt.Errorf("profession_metric_delta_all requires metric")
		}
	case "profession_ideology_delta_all":
		if effect.Ideology == "" {
			return fmt.Errorf("profession_ideology_delta_all requires ideology")
		}
	case "silo_flag_set":
		if effect.Flag == "" {
			return fmt.Errorf("silo_flag_set requires flag")
		}
	case "profession_fragment_add":
		// profession 与 fragment 均可省略：profession 空 = 补给碎片最少的部门，
		// fragment 空 = 随机发一个该部门尚不知道的碎片。
	case "profession_fragment_remove":
		if effect.Profession == "" || effect.Fragment == "" {
			return fmt.Errorf("profession_fragment_remove requires profession and fragment")
		}
	case "floor_stability_delta":
		// zone 空 = 全部楼层；否则按 Upper/Mid/Lower 过滤。
	case "schedule_event":
		if effect.EventID == "" {
			return fmt.Errorf("schedule_event requires event_id")
		}
		if effect.DelayMonths <= 0 {
			return fmt.Errorf("schedule_event requires positive delay_months")
		}
	case "cross_signal":
		// 跨堡事件投递 (多堡研究 §4.2)：向目标域投递延迟跨堡事件。
		// self 域请用现有 schedule_event。
		if effect.EventID == "" {
			return fmt.Errorf("cross_signal requires event_id")
		}
		if effect.DelayMonths < 0 {
			return fmt.Errorf("cross_signal requires delay_months >= 0")
		}
		if effect.Scope == "" || effect.Scope == "self" {
			return fmt.Errorf("cross_signal requires a non-self scope")
		}
	case "world_flag_set":
		if effect.Flag == "" {
			return fmt.Errorf("world_flag_set requires flag")
		}
		if !model.ValidWorldFlag(effect.Flag) {
			return fmt.Errorf("world_flag_set flag %q is not in the world flag whitelist", effect.Flag)
		}
	default:
		return fmt.Errorf("unsupported effect type %q", effect.Type)
	}
	return nil
}

// validateEffectScope 校验 scope 枚举 (未知 scope = 加载失败，维持显式失败原则)。
// schedule_event 是本地调度语义，不接受跨堡 scope。
func validateEffectScope(effect ContentEffect) error {
	scope := effect.Scope
	if scope == "" || scope == "self" {
		return nil
	}
	if effect.Type == "schedule_event" {
		return fmt.Errorf("schedule_event does not accept scope %q (use cross_signal for cross-silo delivery)", scope)
	}
	switch {
	case scope == "world", scope == "neighbors", scope == "all":
		return nil
	case strings.HasPrefix(scope, "silo:"):
		if strings.TrimSpace(strings.TrimPrefix(scope, "silo:")) == "" {
			return fmt.Errorf("silo scope requires a reference (id / player / tag)")
		}
		return nil
	default:
		return fmt.Errorf("unsupported effect scope %q", scope)
	}
}

// CanTriggerContentEvent 判断当前事件是否满足触发条件以及运行时限制。
func CanTriggerContentEvent(def ContentEventDefinition, ctx ContentEvaluationContext) bool {
	return canTriggerContentEvent(def, ctx, false)
}

// CanDisplayPlayerAction checks current availability while ignoring player_action_is.
func CanDisplayPlayerAction(def ContentEventDefinition, ctx ContentEvaluationContext) bool {
	return canTriggerContentEvent(def, ctx, true)
}

func canTriggerContentEvent(def ContentEventDefinition, ctx ContentEvaluationContext, ignoreActionMatch bool) bool {
	if ctx.Silo == nil {
		return false
	}
	if def.FireMode == ContentFireModeOnce && ctx.Runtime.Triggered {
		return false
	}
	now := contentGameTimestamp(ctx.Silo.CurrentYear, ctx.Silo.CurrentMonth)
	if def.FireMode == ContentFireModeRepeatable && def.CooldownMonths > 0 && ctx.Runtime.TriggerCount > 0 {
		cooldown := def.CooldownMonths * contentDaysPerMonth
		if now-ctx.Runtime.LastTriggeredTimestamp < cooldown {
			return false
		}
	}
	if def.Trigger.Type != "" && !evaluateContentTrigger(def.Trigger, ctx, ignoreActionMatch) {
		return false
	}
	if def.ScriptHooks.CanTrigger {
		ok, err := RunContentEventCanTriggerScript(def, ctx, ignoreActionMatch)
		if err != nil {
			return false
		}
		return ok
	}
	return true
}

func evaluateContentTrigger(trigger ContentTrigger, ctx ContentEvaluationContext, ignoreActionMatch bool) bool {
	switch trigger.Type {
	case "all":
		for _, child := range trigger.Conditions {
			if !evaluateContentTrigger(child, ctx, ignoreActionMatch) {
				return false
			}
		}
		return true
	case "any":
		for _, child := range trigger.Conditions {
			if evaluateContentTrigger(child, ctx, ignoreActionMatch) {
				return true
			}
		}
		return false
	case "timestamp_at_or_after":
		target := trigger.Timestamp
		if target == 0 {
			target = contentGameTimestamp(trigger.Year, trigger.Month)
		}
		return contentGameTimestamp(ctx.Silo.CurrentYear, ctx.Silo.CurrentMonth) >= target
	case "silo_metric_gte":
		value, ok := readSiloMetric(ctx.Silo, trigger.Metric)
		return ok && value >= trigger.Value
	case "silo_metric_lte":
		value, ok := readSiloMetric(ctx.Silo, trigger.Metric)
		return ok && value <= trigger.Value
	case "silo_resource_gte":
		value, ok := readSiloResource(ctx.Silo, trigger.Metric)
		return ok && value >= trigger.Value
	case "silo_resource_lte":
		value, ok := readSiloResource(ctx.Silo, trigger.Metric)
		return ok && value <= trigger.Value
	case "silo_population_gte":
		return float64(ctx.Silo.TotalPopulation) >= trigger.Value
	case "silo_population_lte":
		return float64(ctx.Silo.TotalPopulation) <= trigger.Value
	case "silo_flag_true":
		value, ok := readSiloFlagWithWorld(ctx, trigger.Flag)
		return ok && value
	case "silo_flag_false":
		value, ok := readSiloFlagWithWorld(ctx, trigger.Flag)
		return ok && !value
	case "world_flag_true":
		value, ok := readWorldFlag(ctx.World, trigger.Flag)
		return ok && value
	case "world_flag_false":
		value, ok := readWorldFlag(ctx.World, trigger.Flag)
		return ok && !value
	case "signal_received":
		if len(ctx.SignalsReceived) == 0 {
			return false
		}
		if trigger.EventID == "" {
			return true
		}
		for _, id := range ctx.SignalsReceived {
			if id == trigger.EventID {
				return true
			}
		}
		return false
	case "event_triggered":
		for key, runtime := range ctx.States {
			if !runtime.Triggered {
				continue
			}
			if contentStateMatchesTrigger(key, trigger) {
				return true
			}
		}
		return false
	case "player_action_is":
		if ignoreActionMatch {
			return true
		}
		return ctx.Action != nil && contentActionMatches(ctx.Action, trigger.Action)
	case "profession_metric_gte_count":
		count := 0
		for _, profession := range ctx.Silo.Professions {
			if trigger.Profession != "" && profession.Name != trigger.Profession {
				continue
			}
			value, ok := readProfessionMetric(profession, trigger.Metric)
			if ok && value >= trigger.Value {
				count++
			}
		}
		return count >= trigger.Count
	case "profession_ideology_gte_count":
		count := 0
		for _, profession := range ctx.Silo.Professions {
			if trigger.Profession != "" && profession.Name != trigger.Profession {
				continue
			}
			if profession.Ideologies[trigger.Ideology] >= trigger.Value {
				count++
			}
		}
		return count >= trigger.Count
	default:
		return false
	}
}

func contentActionMatches(action *model.AgentAction, expected string) bool {
	if action == nil {
		return false
	}
	target := normalizeContentActionKey(expected)
	if target == "" {
		return false
	}
	if normalizeContentActionKey(action.ActionID) == target {
		return true
	}
	if normalizeContentActionKey(string(action.Type)) == target {
		return true
	}
	if normalizeContentActionKey(action.ProfessionAction) == target {
		return true
	}
	return false
}

func normalizeContentActionKey(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func ContentPlayerActionID(def ContentEventDefinition) string {
	if def.PlayerAction != nil && def.PlayerAction.ID != "" {
		return def.PlayerAction.ID
	}
	return def.EventID
}

// ApplyContentEvent 把事件效果实际施加到地堡状态上 (单堡兼容入口：
// 语义等价于 ScopeContext{Self: silo}，与历史行为逐位一致)。
func ApplyContentEvent(def ContentEventDefinition, silo *model.Silo) (model.StoryEvent, []ContentScheduledEvent) {
	return ApplyContentEventScoped(def, &ScopeContext{Self: silo})
}

// ApplyContentEventScoped 把事件效果施加到 scope 解析出的目标域上 (P0.5)。
func ApplyContentEventScoped(def ContentEventDefinition, sc *ScopeContext) (model.StoryEvent, []ContentScheduledEvent) {
	scheduled := ApplyContentEffectsScoped(def.Effects, sc)
	return model.StoryEvent{
		ID:          def.EventID,
		Category:    normalizeContentCategory(def.SourceGroup),
		Title:       def.Title,
		Description: def.Description,
		Type:        def.Type,
	}, scheduled
}

func ContentEventBusName(def ContentEventDefinition) string {
	category := normalizeContentCategory(def.SourceGroup)
	if category == "" {
		return def.EventID
	}
	return category + ":" + def.EventID
}

func ContentTriggerEventNames(def ContentEventDefinition) []string {
	names := map[string]bool{}
	collectContentTriggerEventNames(def, def.Trigger, names)
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func HasEventTriggeredTrigger(trigger ContentTrigger) bool {
	if trigger.Type == "event_triggered" {
		return true
	}
	for _, child := range trigger.Conditions {
		if HasEventTriggeredTrigger(child) {
			return true
		}
	}
	return false
}

func collectContentTriggerEventNames(def ContentEventDefinition, trigger ContentTrigger, names map[string]bool) {
	if trigger.Type == "event_triggered" && trigger.EventID != "" {
		category := trigger.Category
		if category == "" && normalizeContentCategory(def.SourceGroup) == "crisis" {
			category = "special"
		}
		name := trigger.EventID
		if category != "" {
			name = category + ":" + trigger.EventID
		}
		names[name] = true
	}
	for _, child := range trigger.Conditions {
		collectContentTriggerEventNames(def, child, names)
	}
}

func normalizeContentCategory(group string) string {
	switch strings.ToLower(strings.TrimSpace(group)) {
	case "histories", "history":
		return "history"
	case "special":
		return "special"
	case "crisis":
		return "crisis"
	case "player_actions", "player_action":
		return "player_action"
	case "events":
		return "special"
	default:
		return strings.ToLower(strings.TrimSpace(group))
	}
}

func contentStateMatchesTrigger(key string, trigger ContentTrigger) bool {
	stateCategory := ""
	stateEventID := key
	if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
		stateCategory = normalizeContentCategory(parts[0])
		stateEventID = parts[1]
	}
	if trigger.EventID != "" && stateEventID != trigger.EventID {
		return false
	}
	if trigger.Category != "" && stateCategory != "" && stateCategory != trigger.Category {
		return false
	}
	if trigger.Category != "" && stateCategory == "" {
		return false
	}
	return true
}

// ContentScheduledEvent represents an event to be scheduled.
type ContentScheduledEvent struct {
	EventID     string
	DelayMonths int
}

// ScopeContext 施加期解析 effect scope 所需的世界级事实 (多堡研究 §4.2)。
// World == nil (引擎内测试 / 单堡调用方) 时只能解析 self 域，跨堡效果会
// 记入 Errors 而不是静默丢弃；单堡调用路径与历史行为逐位一致。
type ScopeContext struct {
	Self  *model.Silo
	World *model.World
	Silos []*model.Silo // 世界内全部堡，按 SiloID 升序
	// CrossSignals 收集 cross_signal 效果 (服务层提供入队汇点)；nil = 无处投递。
	CrossSignals *[]CrossSignalRequest
	// Errors 收集目标解析失败等显式错误 (对齐 07 §12：不静默丢弃)。
	Errors *[]string
}

// CrossSignalRequest 一次 cross_signal 效果产生的跨堡投递请求。
// 由服务层转换为 model.CrossEvent 进入 World.CrossQueue。
type CrossSignalRequest struct {
	SourceSilo  uint
	TargetScope string
	EventID     string
	DelayMonths int
}

// ApplyContentEffects 单堡兼容入口：语义等价于 ScopeContext{Self: silo}。
func ApplyContentEffects(effects []ContentEffect, silo *model.Silo) []ContentScheduledEvent {
	return ApplyContentEffectsScoped(effects, &ScopeContext{Self: silo})
}

// ApplyContentEffectsScoped 按 scope 施加效果。schedule_event 仍按本地调度
// 语义收集返回 (只作用于 self)；cross_signal 进入 ScopeContext.CrossSignals。
func ApplyContentEffectsScoped(effects []ContentEffect, sc *ScopeContext) []ContentScheduledEvent {
	var scheduled []ContentScheduledEvent
	for _, effect := range effects {
		if effect.Type == "schedule_event" {
			scheduled = append(scheduled, ContentScheduledEvent{
				EventID:     effect.EventID,
				DelayMonths: effect.DelayMonths,
			})
			continue
		}
		applyContentEffectScoped(effect, sc)
	}
	return scheduled
}

// applyContentEffectScoped 按 scope 解析目标域并逐堡施加 (按 SiloID 升序，
// 确定性)。世界类效果 (world_flag_set) 直接作用于世界状态。
func applyContentEffectScoped(effect ContentEffect, sc *ScopeContext) {
	if sc == nil {
		return
	}
	switch effect.Type {
	case "cross_signal":
		if sc.CrossSignals == nil {
			sc.reportError("cross_signal " + effect.EventID + " dropped: no world context")
			return
		}
		source := uint(0)
		if sc.Self != nil {
			source = sc.Self.ID
		}
		*sc.CrossSignals = append(*sc.CrossSignals, CrossSignalRequest{
			SourceSilo:  source,
			TargetScope: effect.Scope,
			EventID:     effect.EventID,
			DelayMonths: effect.DelayMonths,
		})
		return
	case "world_flag_set":
		if sc.World == nil {
			sc.reportError("world_flag_set " + effect.Flag + " dropped: no world context")
			return
		}
		writeWorldFlag(sc, effect.Flag, effect.BoolValue)
		return
	case "silo_flag_set":
		// silo1_destroyed 本质是世界事实 (设计 v2 §一)：有世界上下文时经世界
		// 别名写入 (World.Silo1.Alive) 并镜像到玩家堡，保证触发器读取一致；
		// 无世界上下文 (单堡/引擎测试) 时落到堡字段，行为与历史逐位一致。
		if effect.Flag == model.WorldFlagSilo1Destroyed && sc.World != nil {
			writeWorldFlag(sc, effect.Flag, effect.BoolValue)
			return
		}
	}

	targets := resolveScope(sc, effect.Scope)
	for _, target := range targets {
		applyContentEffect(effect, target)
	}
}

// resolveScope 把 scope 字符串解析为目标堡集合 (施加期解析；目标存活状态
// 是运行时事实)。解析失败记入 Errors 并返回空集——不静默丢弃。
// 结果恒按 SiloID 升序 (确定性纪律，多堡研究 §2.3)。
func resolveScope(sc *ScopeContext, scope string) []*model.Silo {
	switch {
	case scope == "" || scope == "self":
		if sc.Self == nil {
			return nil
		}
		return []*model.Silo{sc.Self}
	case scope == "all":
		return append([]*model.Silo(nil), sc.Silos...)
	case scope == "neighbors":
		if sc.World == nil || sc.Self == nil {
			sc.reportError("scope neighbors dropped: no world context")
			return nil
		}
		return silosByIDs(sc, sc.World.Topology[sc.Self.ID])
	case strings.HasPrefix(scope, "silo:"):
		ref := strings.TrimSpace(strings.TrimPrefix(scope, "silo:"))
		if sc.World == nil {
			sc.reportError("scope " + scope + " dropped: no world context")
			return nil
		}
		switch ref {
		case "player":
			return silosByIDs(sc, []uint{sc.World.PlayerSiloID})
		case "nearest_alive_neighbor":
			if sc.Self == nil {
				return nil
			}
			for _, id := range sc.World.Topology[sc.Self.ID] { // silosByIDs 已按升序
				if silo := siloByID(sc, id); silo != nil && silo.VictoryStatus == nil {
					return []*model.Silo{silo}
				}
			}
			sc.reportError("scope " + scope + ": no alive neighbor found")
			return nil
		default:
			id, err := strconv.ParseUint(ref, 10, 64)
			if err != nil {
				sc.reportError("scope " + scope + ": unknown silo reference")
				return nil
			}
			return silosByIDs(sc, []uint{uint(id)})
		}
	default:
		// validateEffectScope 已在加载期拦截；此处为运行时兜底。
		sc.reportError("scope " + scope + ": unsupported at apply time")
		return nil
	}
}

// silosByIDs 按给定 ID 集合解析堡指针，结果按 SiloID 升序；未知 ID 记错误。
func silosByIDs(sc *ScopeContext, ids []uint) []*model.Silo {
	sorted := append([]uint(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var result []*model.Silo
	for _, id := range sorted {
		if silo := siloByID(sc, id); silo != nil {
			result = append(result, silo)
		} else {
			sc.reportError("scope target silo:" + strconv.FormatUint(uint64(id), 10) + " not found")
		}
	}
	return result
}

func siloByID(sc *ScopeContext, id uint) *model.Silo {
	for _, silo := range sc.Silos {
		if silo != nil && silo.ID == id {
			return silo
		}
	}
	return nil
}

func (sc *ScopeContext) reportError(message string) {
	if sc != nil && sc.Errors != nil && message != "" {
		*sc.Errors = append(*sc.Errors, message)
	}
}

func applyContentEffect(effect ContentEffect, silo *model.Silo) {
	switch effect.Type {
	case "silo_metric_delta":
		applySiloMetricDelta(silo, effect.Metric, effect.Value)
	case "silo_resource_delta":
		applySiloResourceDelta(silo, effect.Metric, effect.Value)
	case "silo_population_delta":
		applySiloPopulationDelta(silo, effect.Value)
	case "profession_metric_delta_all":
		for i := range silo.Professions {
			applyProfessionMetricDelta(&silo.Professions[i], effect.Metric, effect.Value)
		}
	case "profession_ideology_delta_all":
		for i := range silo.Professions {
			if silo.Professions[i].Ideologies == nil {
				silo.Professions[i].Ideologies = map[string]float64{}
			}
			silo.Professions[i].Ideologies[effect.Ideology] = clampUnit(silo.Professions[i].Ideologies[effect.Ideology] + effect.Value)
		}
	case "silo_flag_set":
		writeSiloFlag(silo, effect.Flag, effect.BoolValue)
	case "profession_fragment_add":
		applyContentFragmentAdd(silo, effect.Profession, effect.Fragment)
	case "profession_fragment_remove":
		applyContentFragmentRemove(silo, effect.Profession, effect.Fragment)
	case "floor_stability_delta":
		applyContentFloorStabilityDelta(silo, effect.Zone, effect.Value)
	case "schedule_event":
		// Handled at the service level where the runtime can rewrite the main event queue.
	}
}

// applyContentFragmentAdd 把一个信息碎片持久写入部门 KnownFragments。
// profession 为空时补给当前碎片最少的部门（并列取 ID 更小者，确定性）；
// fragment 为空时从该部门未知碎片中按引擎随机源抽取（种子可复现）。
func applyContentFragmentAdd(silo *model.Silo, profession, fragment string) {
	if silo == nil || len(silo.Professions) == 0 {
		return
	}
	target := -1
	if strings.TrimSpace(profession) != "" {
		for i := range silo.Professions {
			if silo.Professions[i].Name == profession {
				target = i
				break
			}
		}
		if target < 0 {
			return
		}
	} else {
		for i := range silo.Professions {
			if target < 0 || len(silo.Professions[i].KnownFragments) < len(silo.Professions[target].KnownFragments) {
				target = i
			}
		}
	}
	prof := &silo.Professions[target]

	pick := strings.TrimSpace(fragment)
	if pick == "" {
		known := map[string]bool{}
		for _, f := range prof.KnownFragments {
			known[f] = true
		}
		var candidates []string
		for _, f := range model.ALL_FRAGMENTS {
			if !known[f] {
				candidates = append(candidates, f)
			}
		}
		if len(candidates) == 0 {
			return
		}
		pick = candidates[rng.Intn(len(candidates))]
	} else {
		for _, f := range prof.KnownFragments {
			if f == pick {
				return
			}
		}
	}
	prof.KnownFragments = append(prof.KnownFragments, pick)
}

// applyContentFragmentRemove 从部门 KnownFragments 移除指定碎片；不存在则无操作。
func applyContentFragmentRemove(silo *model.Silo, profession, fragment string) {
	prof := findDept(silo, profession)
	if prof == nil {
		return
	}
	prof.KnownFragments = removeString(prof.KnownFragments, fragment)
}

// applyContentFloorStabilityDelta 调整楼层结构稳定度（0–1 截断）。
// zone 为空作用于全部楼层，否则只作用于匹配 Upper/Mid/Lower 的楼层。
func applyContentFloorStabilityDelta(silo *model.Silo, zone string, delta float64) {
	if silo == nil {
		return
	}
	for i := range silo.Floors {
		if zone != "" && !strings.EqualFold(silo.Floors[i].Zone, zone) {
			continue
		}
		silo.Floors[i].Stability = clampUnit(silo.Floors[i].Stability + delta)
	}
}

func applySiloResourceDelta(silo *model.Silo, resourceType string, delta float64) {
	for i := range silo.Resources {
		if strings.EqualFold(silo.Resources[i].Type, resourceType) {
			silo.Resources[i].Amount = math.Max(0, silo.Resources[i].Amount+delta)
			return
		}
	}
}

func applySiloPopulationDelta(silo *model.Silo, delta float64) {
	silo.TotalPopulation = int(math.Max(0, float64(silo.TotalPopulation)+delta))
}

func applySiloMetricDelta(silo *model.Silo, metric string, delta float64) {
	switch metric {
	case "legitimacy":
		silo.SetMayorLegitimacy(silo.MayorLegitimacy() + delta)
	case "cohesion":
		silo.Cohesion = clampUnit(silo.Cohesion + delta)
	case "rebellion":
		silo.Rebellion = clampUnit(silo.Rebellion + delta)
	case "dept_tension":
		silo.DeptTension = clampUnit(silo.DeptTension + delta)
	case "class_fragmentation":
		silo.ClassFragmentation = clampUnit(silo.ClassFragmentation + delta)
	case "history_burden":
		silo.HistoryBurden = clampUnit(silo.HistoryBurden + delta)
	case "event_trigger":
		silo.EventTrigger = math.Max(0, silo.EventTrigger+delta)
	case "countdown":
		silo.Countdown = math.Max(0, silo.Countdown+delta)
	}
}

func applyProfessionMetricDelta(profession *model.Profession, metric string, delta float64) {
	switch metric {
	case "panic_value":
		profession.PanicValue = clampUnit(profession.PanicValue + delta)
	case "productivity":
		profession.Productivity = clampUnit(profession.Productivity + delta)
	}
}

func readSiloResource(silo *model.Silo, resourceType string) (float64, bool) {
	for _, r := range silo.Resources {
		if strings.EqualFold(r.Type, resourceType) {
			return r.Amount, true
		}
	}
	return 0, false
}

func readSiloMetric(silo *model.Silo, metric string) (float64, bool) {
	switch metric {
	case "legitimacy":
		return silo.MayorLegitimacy(), true
	case "cohesion":
		return silo.Cohesion, true
	case "rebellion":
		return silo.Rebellion, true
	case "dept_tension":
		return silo.DeptTension, true
	case "class_fragmentation":
		return silo.ClassFragmentation, true
	case "history_burden":
		return silo.HistoryBurden, true
	case "event_trigger":
		return silo.EventTrigger, true
	case "countdown":
		return silo.Countdown, true
	default:
		return 0, false
	}
}

func readProfessionMetric(profession model.Profession, metric string) (float64, bool) {
	switch metric {
	case "panic_value":
		return profession.PanicValue, true
	case "productivity":
		return profession.Productivity, true
	default:
		return 0, false
	}
}

// readSiloFlagWithWorld 堡 flag 读取的世界化别名 (多堡研究 §2.2)：
// silo1_destroyed 本质是世界事实，存在世界层后优先读 World.Silo1.Alive；
// 无世界上下文 (引擎内测试) 时回退到堡字段，与历史行为一致。
func readSiloFlagWithWorld(ctx ContentEvaluationContext, flag string) (bool, bool) {
	if flag == model.WorldFlagSilo1Destroyed && ctx.World != nil {
		return readWorldFlag(ctx.World, flag)
	}
	return readSiloFlag(ctx.Silo, flag)
}

// readWorldFlag 读世界 flag。silo1_destroyed 是 World.Silo1.Alive 的别名。
func readWorldFlag(world *model.World, flag string) (bool, bool) {
	if world == nil || !model.ValidWorldFlag(flag) {
		return false, false
	}
	if flag == model.WorldFlagSilo1Destroyed {
		return !world.Silo1.Alive, true
	}
	return world.Flags[flag], true
}

// writeWorldFlag 写世界 flag。silo1_destroyed 同步镜像到玩家堡的
// Silo1Destroyed 字段 (旧触发器与胜负判定的兼容别名，设计 v2 §一)。
func writeWorldFlag(sc *ScopeContext, flag string, value bool) {
	world := sc.World
	if world.Flags == nil {
		world.Flags = map[string]bool{}
	}
	if flag == model.WorldFlagSilo1Destroyed {
		world.Silo1.Alive = !value
		if player := siloByID(sc, world.PlayerSiloID); player != nil {
			player.Silo1Destroyed = value
		}
		return
	}
	world.Flags[flag] = value
}

func readSiloFlag(silo *model.Silo, flag string) (bool, bool) {
	switch flag {
	case "silo1_destroyed":
		return silo.Silo1Destroyed, true
	default:
		return false, false
	}
}

func writeSiloFlag(silo *model.Silo, flag string, value bool) {
	switch flag {
	case "silo1_destroyed":
		silo.Silo1Destroyed = value
	}
}

func clampUnit(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func contentGameTimestamp(year, month int) int {
	return ((year-contentTimestampBaseYear)*contentMonthsPerYear + (month - 1)) * contentDaysPerMonth
}

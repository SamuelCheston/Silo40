package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
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
type ContentEffect struct {
	Type        string  `json:"type"`
	Metric      string  `json:"metric,omitempty"`
	Flag        string  `json:"flag,omitempty"`
	Ideology    string  `json:"ideology,omitempty"`
	Profession  string  `json:"profession,omitempty"`
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
type ContentEvaluationContext struct {
	Silo    *model.Silo
	Runtime ContentEventRuntime
	States  map[string]ContentEventRuntime
	Action  *model.AgentAction
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
		def.Effects[i].Metric = strings.ToLower(strings.TrimSpace(def.Effects[i].Metric))
		def.Effects[i].Flag = strings.ToLower(strings.TrimSpace(def.Effects[i].Flag))
		def.Effects[i].Profession = strings.TrimSpace(def.Effects[i].Profession)
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
	case "schedule_event":
		if effect.EventID == "" {
			return fmt.Errorf("schedule_event requires event_id")
		}
		if effect.DelayMonths <= 0 {
			return fmt.Errorf("schedule_event requires positive delay_months")
		}
	default:
		return fmt.Errorf("unsupported effect type %q", effect.Type)
	}
	return nil
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
		value, ok := readSiloFlag(ctx.Silo, trigger.Flag)
		return ok && value
	case "silo_flag_false":
		value, ok := readSiloFlag(ctx.Silo, trigger.Flag)
		return ok && !value
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

// ApplyContentEvent 把事件效果实际施加到地堡状态上。
func ApplyContentEvent(def ContentEventDefinition, silo *model.Silo) (model.StoryEvent, []ContentScheduledEvent) {
	scheduled := ApplyContentEffects(def.Effects, silo)
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

func ApplyContentEffects(effects []ContentEffect, silo *model.Silo) []ContentScheduledEvent {
	var scheduled []ContentScheduledEvent
	for _, effect := range effects {
		if effect.Type == "schedule_event" {
			scheduled = append(scheduled, ContentScheduledEvent{
				EventID:     effect.EventID,
				DelayMonths: effect.DelayMonths,
			})
			continue
		}
		applyContentEffect(effect, silo)
	}
	return scheduled
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
	case "schedule_event":
		// Handled at the service level where the runtime can rewrite the main event queue.
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
		silo.Legitimacy = clampUnit(silo.Legitimacy + delta)
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
		profession.Productivity = math.Max(0, profession.Productivity+delta)
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
		return silo.Legitimacy, true
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

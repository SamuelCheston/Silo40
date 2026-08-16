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
	EventID        string          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Type           string          `json:"type"`
	FireMode       string          `json:"fire_mode"`
	CooldownMonths int             `json:"cooldown_months,omitempty"`
	Trigger        ContentTrigger  `json:"trigger"`
	Effects        []ContentEffect `json:"effects"`
}

// ContentTrigger 结构化触发条件。
type ContentTrigger struct {
	Type       string           `json:"type"`
	Metric     string           `json:"metric,omitempty"`
	Flag       string           `json:"flag,omitempty"`
	Ideology   string           `json:"ideology,omitempty"`
	Profession string           `json:"profession,omitempty"`
	Value      float64          `json:"value,omitempty"`
	Count      int              `json:"count,omitempty"`
	Year       int              `json:"year,omitempty"`
	Month      int              `json:"month,omitempty"`
	Timestamp  int              `json:"timestamp,omitempty"`
	Conditions []ContentTrigger `json:"conditions,omitempty"`
}

// ContentEffect 结构化事件效果。
type ContentEffect struct {
	Type       string  `json:"type"`
	Metric     string  `json:"metric,omitempty"`
	Flag       string  `json:"flag,omitempty"`
	Ideology   string  `json:"ideology,omitempty"`
	Profession string  `json:"profession,omitempty"`
	Value      float64 `json:"value,omitempty"`
	BoolValue  bool    `json:"bool_value,omitempty"`
}

// ContentEventRuntime 单局游戏内的运行时触发状态。
type ContentEventRuntime struct {
	Triggered              bool
	TriggerCount           int
	LastTriggeredTimestamp int
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
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".json" {
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
}

func normalizeTrigger(trigger *ContentTrigger) {
	trigger.Type = strings.ToLower(strings.TrimSpace(trigger.Type))
	trigger.Metric = strings.ToLower(strings.TrimSpace(trigger.Metric))
	trigger.Flag = strings.ToLower(strings.TrimSpace(trigger.Flag))
	trigger.Ideology = strings.TrimSpace(trigger.Ideology)
	trigger.Profession = strings.TrimSpace(trigger.Profession)
	if trigger.Month == 0 {
		trigger.Month = 1
	}
	for i := range trigger.Conditions {
		normalizeTrigger(&trigger.Conditions[i])
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
	if err := validateTrigger(def.Trigger); err != nil {
		return err
	}
	for _, effect := range def.Effects {
		if err := validateEffect(effect); err != nil {
			return err
		}
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
	case "silo_metric_gte", "silo_metric_lte":
		if trigger.Metric == "" {
			return fmt.Errorf("%s requires metric", trigger.Type)
		}
		return nil
	case "silo_flag_true", "silo_flag_false":
		if trigger.Flag == "" {
			return fmt.Errorf("%s requires flag", trigger.Type)
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
	case "silo_metric_delta":
		if effect.Metric == "" {
			return fmt.Errorf("silo_metric_delta requires metric")
		}
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
	default:
		return fmt.Errorf("unsupported effect type %q", effect.Type)
	}
	return nil
}

// CanTriggerContentEvent 判断当前事件是否满足触发条件以及运行时限制。
func CanTriggerContentEvent(def ContentEventDefinition, silo *model.Silo, runtime ContentEventRuntime) bool {
	if silo == nil {
		return false
	}
	if def.FireMode == ContentFireModeOnce && runtime.Triggered {
		return false
	}
	now := contentGameTimestamp(silo.CurrentYear, silo.CurrentMonth)
	if def.FireMode == ContentFireModeRepeatable && def.CooldownMonths > 0 && runtime.TriggerCount > 0 {
		cooldown := def.CooldownMonths * contentDaysPerMonth
		if now-runtime.LastTriggeredTimestamp < cooldown {
			return false
		}
	}
	return evaluateContentTrigger(def.Trigger, silo)
}

func evaluateContentTrigger(trigger ContentTrigger, silo *model.Silo) bool {
	switch trigger.Type {
	case "all":
		for _, child := range trigger.Conditions {
			if !evaluateContentTrigger(child, silo) {
				return false
			}
		}
		return true
	case "any":
		for _, child := range trigger.Conditions {
			if evaluateContentTrigger(child, silo) {
				return true
			}
		}
		return false
	case "timestamp_at_or_after":
		target := trigger.Timestamp
		if target == 0 {
			target = contentGameTimestamp(trigger.Year, trigger.Month)
		}
		return contentGameTimestamp(silo.CurrentYear, silo.CurrentMonth) >= target
	case "silo_metric_gte":
		value, ok := readSiloMetric(silo, trigger.Metric)
		return ok && value >= trigger.Value
	case "silo_metric_lte":
		value, ok := readSiloMetric(silo, trigger.Metric)
		return ok && value <= trigger.Value
	case "silo_flag_true":
		value, ok := readSiloFlag(silo, trigger.Flag)
		return ok && value
	case "silo_flag_false":
		value, ok := readSiloFlag(silo, trigger.Flag)
		return ok && !value
	case "profession_metric_gte_count":
		count := 0
		for _, profession := range silo.Professions {
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
		for _, profession := range silo.Professions {
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

// ApplyContentEvent 把事件效果实际施加到地堡状态上。
func ApplyContentEvent(def ContentEventDefinition, silo *model.Silo) model.StoryEvent {
	for _, effect := range def.Effects {
		applyContentEffect(effect, silo)
	}
	return model.StoryEvent{
		ID:          def.EventID,
		Title:       def.Title,
		Description: def.Description,
		Type:        def.Type,
	}
}

func applyContentEffect(effect ContentEffect, silo *model.Silo) {
	switch effect.Type {
	case "silo_metric_delta":
		applySiloMetricDelta(silo, effect.Metric, effect.Value)
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
	}
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

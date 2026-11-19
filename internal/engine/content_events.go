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
//
// DeadlineYear / DeadlineMonth 是"到点强制拉起"兜底 (15/22 分册约定)：到该日历
// 时点仍未触发过的事件在月度评估里无条件点火一次，绕过 trigger 与
// script.canTrigger。用于剧情上必须发生的窗口内事件 (如 charter_lock_day)，
// 防止前置早于时点完成时一次性评估永不点火、反被窗口超时先杀线。
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
	DeadlineYear   int                      `json:"deadline_year,omitempty"`
	DeadlineMonth  int                      `json:"deadline_month,omitempty"`
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
	Months     int              `json:"months,omitempty"` // months_since_event_gte：距事件点亮的月数下限
	Conditions []ContentTrigger `json:"conditions,omitempty"`
}

// 秩序压力复合度量 (order_pressure)：单读叛乱在现有模型下不可达 (实测玩家
// 上限 ~0.45)，1 号反应链改读叛乱与部门恐慌的加权和。权重 TODO(balance)，
// 由 sim 校准：判据是"专门推叛乱的局能触发、其他玩法不触发"。
const (
	OrderPressureRebellionWeight = 0.6 // TODO(balance)
	OrderPressurePanicWeight     = 0.4 // TODO(balance)
	// Silo1UltimatumPressure 是 1 号签发通牒与三条反应链前兆共用的复合门槛
	// (TODO(balance))：sim 判据 = 专门推叛乱的局能在结算前触发至少一条链，
	// 信息路/撤离路/躺平局不触发。
	Silo1UltimatumPressure = 0.45 // TODO(balance)
	// Silo1AgitationPropaganda 是"有组织煽动"的宣传等级门槛 (TODO(balance))：
	// 复合压力只衡量街头热度，背景漂移也能把它推过线；宣传机器是玩家把
	// 不满组织化的可归属事实，反应链只对"压力＋组织化"回应，避免把信息路/
	// 撤离路的背景恐慌误读成对秩序的推动。
	Silo1AgitationPropaganda = 2.0 // TODO(balance)
)

// OrderPressure 叛乱与平均部门恐慌的加权和。恐慌取全部部门 PanicValue 均值
// (部门升序累加，确定性)；无部门时恐慌项为 0。
func OrderPressure(silo *model.Silo) float64 {
	panicMean := 0.0
	if len(silo.Professions) > 0 {
		total := 0.0
		for _, profession := range silo.Professions {
			total += profession.PanicValue
		}
		panicMean = total / float64(len(silo.Professions))
	}
	return OrderPressureRebellionWeight*silo.Rebellion + OrderPressurePanicWeight*panicMean
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
	// Count 整数载荷：spillover_channel_add 的目标堡 ID。
	Count   int    `json:"count,omitempty"`
	EventID string `json:"event_id,omitempty"`
	// Action 仅用于 player_action_complete：写入的玩家完成键动作 ID
	// (服务层加 "player_action:" 前缀，如 reset_countermeasure_done)。
	Action string `json:"action,omitempty"`
	// Observer 仅用于 belief_observe："player" / "residents" / 部门名。
	Observer string `json:"observer,omitempty"`
	// Reliability/Provenance/Coverage 服务于 belief_observe 与
	// public_evidence_register (§3.5/§3.7)。
	Reliability float64 `json:"reliability,omitempty"`
	Provenance  float64 `json:"provenance,omitempty"`
	Coverage    float64 `json:"coverage,omitempty"`
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
	// Agent 当前模拟堡的操作者 (G3：内容脚本可读怀疑度/威望等 actor 状态)。
	Agent *model.Agent
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
	if def.DeadlineYear > 0 && def.DeadlineMonth == 0 {
		def.DeadlineMonth = 1
	}
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
		def.Effects[i].Action = strings.TrimSpace(def.Effects[i].Action)
		def.Effects[i].Observer = strings.TrimSpace(def.Effects[i].Observer)
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
	if def.DeadlineYear > 0 || def.DeadlineMonth > 0 {
		if def.DeadlineYear <= 0 || def.DeadlineMonth < 1 || def.DeadlineMonth > 12 {
			return fmt.Errorf("deadline requires a positive year and a month in 1-12")
		}
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
	case "preparation_gte":
		// 16 §6.2 (Layer 1): ledger field >= threshold, for downstream gating.
		if trigger.Metric == "" {
			return fmt.Errorf("preparation_gte requires metric")
		}
		if _, ok := readPreparationMetric(&model.Silo{}, trigger.Metric); !ok {
			return fmt.Errorf("preparation_gte metric %q is not a readable ledger field", trigger.Metric)
		}
		return nil
	case "silo_population_gte", "silo_population_lte":
		return nil
	case "silo_flag_true", "silo_flag_false":
		if trigger.Flag == "" {
			return fmt.Errorf("%s requires flag", trigger.Type)
		}
		return nil
	case "world_flag_true", "world_flag_false", "world_flag_not_set":
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
	case "months_since_event_gte":
		// 相对时间门：距某事件点亮至少 N 月。反应链用它替代绝对年份门，
		// 使升级阶梯锚定通牒而非日历。
		if trigger.EventID == "" || trigger.Category == "" {
			return fmt.Errorf("months_since_event_gte requires category and event_id")
		}
		if trigger.Months < 0 {
			return fmt.Errorf("months_since_event_gte requires non-negative months")
		}
		return nil
	case "actor_metric_gte":
		switch trigger.Metric {
		case "propaganda_level", "suspicion_level", "political_prestige", "ambition", "influence":
			return nil
		default:
			return fmt.Errorf("actor_metric_gte metric %q is not readable", trigger.Metric)
		}
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
	case "profession_metric_delta":
		if effect.Metric == "" {
			return fmt.Errorf("profession_metric_delta requires metric")
		}
		if strings.TrimSpace(effect.Profession) == "" {
			return fmt.Errorf("profession_metric_delta requires profession")
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
	case "cohort_ideology_delta":
		// G4: persistent cohort-level ideology write. profession 空 = 全部 cohort。
		if effect.Ideology == "" {
			return fmt.Errorf("cohort_ideology_delta requires ideology")
		}
	case "authority_evidence_delta":
		// G2: PublicEvidenceQ supply for the public-charter qualification.
	case "authority_public_fact":
		// G2: independent public fact registration; flag is the dedup key.
		if effect.Flag == "" {
			return fmt.Errorf("authority_public_fact requires flag as the fact key")
		}
	case "authority_pact_qualify":
		// G2: charter qualification flag (bool_value).
	case "escape_group_depart":
		// G5: move population out of the silo into a surviving escape group.
	case "escape_group_status":
		// G5: mark the escape group surviving or destroyed (bool_value).
	case "top_stance_reeval":
		// 16 §3.2 (ADR-0008): weighted-random re-evaluation of the top stance
		// of mayor/it/both (flag). Internal state; never a player option.
		switch effect.Flag {
		case "mayor", "it", "both":
		default:
			return fmt.Errorf("top_stance_reeval requires flag mayor|it|both")
		}
	case "top_stance_guarantee":
		// 16 §3.2: raise (or lower) the credibility of the rebels' guarantees;
		// weight input of the stance decision, clamped to [0,1] at apply time.
	case "preparation_delta":
		// 16 §6.2 (Layer 1): whitelisted ledger writes; unknown field = failure.
		if err := validatePreparationDelta(effect); err != nil {
			return err
		}
	case "preparation_transfer":
		// 16 §6.2: resource transfer into the sealed ledger; the deduction is
		// immediate and insufficient balance fails explicitly at apply time.
		// Metric 比较大小写不敏感：脚本效果归一化会把 metric 小写化。
		switch strings.ToLower(effect.Metric) {
		case "supplies", "materials":
		default:
			return fmt.Errorf("preparation_transfer metric must be Supplies or Materials")
		}
		if effect.Value <= 0 {
			return fmt.Errorf("preparation_transfer requires positive value")
		}
	case "spillover_channel_add":
		// 16 §6.2: register one restricted channel (establishment side only;
		// delivery/reception settlement is Layer 3).
		if _, ok := spilloverChannelKinds[effect.Flag]; !ok {
			return fmt.Errorf("spillover_channel_add flag must be it_line|cross_signal|physical_handoff")
		}
		if effect.Count <= 0 {
			return fmt.Errorf("spillover_channel_add requires positive count (target silo id)")
		}
	case "spillover_send":
		// 16 §3.5 (Layer 3 投递侧)：请求经一条已建立的受限信道投递外溢包裹。
		// flag = 信道种类；目标堡与可靠度读建立侧账本。投递结算 (可靠度抽取、
		// 暴露记账、跨堡入队) 由服务层在全部效果施加后执行。
		if _, ok := spilloverChannelKinds[effect.Flag]; !ok {
			return fmt.Errorf("spillover_send flag must be it_line|cross_signal|physical_handoff")
		}
	case "escape_org_suspicion_delta":
		// 20 修订/ADR-0014：组织行为累积组织怀疑度。只增不减 (衰减走月度
		// 回合)；囤积类动作永不声明本效果。
		if effect.Value <= 0 {
			return fmt.Errorf("escape_org_suspicion_delta requires positive value")
		}
	case "escape_departure_settle":
		// 16 §3.4 (departure settlement point, minimal form): the service layer
		// settles after all effects apply. No fields; the settlement reads the
		// stance + ledger + route facts and writes the escape group.
	case "negotiation_open":
		// 21 设计卷：开信道请求，服务层执行 (性情抽取 + 叙事)。无字段。
	case "negotiation_round":
		// 21 设计卷：高强度回合请求。flag = probe 试探 | terms 递条款。
		switch effect.Flag {
		case "probe", "terms":
		default:
			return fmt.Errorf("negotiation_round requires flag probe|terms")
		}
	case "negotiation_resolve":
		// 21 设计卷：妥协后二选一。flag = takeover 接管 | hands_off 不接管。
		switch effect.Flag {
		case "takeover", "hands_off":
		default:
			return fmt.Errorf("negotiation_resolve requires flag takeover|hands_off")
		}
	case "player_action_complete":
		// 玩家完成键写入路径 (18 分册 copies_distributed 一类节点)：内容事件
		// 由此写入 player_action:<action> 完成键，服务层负责落状态并发总线事件。
		if strings.TrimSpace(effect.Action) == "" {
			return fmt.Errorf("player_action_complete requires action")
		}
	case "belief_observe":
		// §3.3/§3.5：向观察者层注入一次事实观察 (玩家/部门/具名人物)。
		if strings.TrimSpace(effect.Fragment) == "" {
			return fmt.Errorf("belief_observe requires fragment")
		}
		if strings.TrimSpace(effect.Observer) == "" {
			return fmt.Errorf("belief_observe requires observer")
		}
	case "public_evidence_register":
		// §3.7：登记公开证据 (provenance/coverage)；Q 与独立事实资格只读登记。
		if strings.TrimSpace(effect.Fragment) == "" {
			return fmt.Errorf("public_evidence_register requires fragment")
		}
		if effect.Provenance <= 0 || effect.Coverage <= 0 {
			return fmt.Errorf("public_evidence_register requires positive provenance and coverage")
		}
	case "evidence_verify":
		// 子项 6：核验把事实提升为"已核验档案"级 provenance 的前置。
		if strings.TrimSpace(effect.Fragment) == "" {
			return fmt.Errorf("evidence_verify requires fragment")
		}
	case "authority_w_shock":
		// 威势冲击 (§3.2 δe)：内容事件经冲击队列影响 W，下次月度更新消费一次。
		if effect.Value == 0 {
			return fmt.Errorf("authority_w_shock requires non-zero value")
		}
	case "decisive_shock":
		// 裁决的冲击通道：结构性事件的离散负冲击 (三类触发源)。value 缺省 =
		// DecisiveShockMagnitude；flag 可选，覆盖冲击源键 (缺省由服务层用事件键)。
		if effect.Value > 0 {
			return fmt.Errorf("decisive_shock value must be negative (got %v)", effect.Value)
		}
	case "alliance_loyalty_delta":
		// §5.1：二元联盟忠诚冲击 (公开证据动摇法警/IT 内部)。
		// observer = "bailiff" / "it_internal"。
		if effect.Observer != "bailiff" && effect.Observer != "it_internal" {
			return fmt.Errorf("alliance_loyalty_delta requires observer bailiff or it_internal")
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

// HasPlayerActionTrigger reports whether the trigger tree contains a
// player_action_is node. Player-action-group definitions without one must not
// ride along on unrelated actions (they fire only when explicitly submitted).
func HasPlayerActionTrigger(def ContentEventDefinition) bool {
	return hasPlayerActionTrigger(def.Trigger)
}

func hasPlayerActionTrigger(trigger ContentTrigger) bool {
	if trigger.Type == "player_action_is" {
		return true
	}
	for _, child := range trigger.Conditions {
		if hasPlayerActionTrigger(child) {
			return true
		}
	}
	return false
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
		// 荒年档危机提前且加密 (19 §3.2)：仅 crisis 组冷却乘档内倍率，其余
		// 事件与空原型逐位不变。
		if def.SourceGroup == "crisis" {
			cooldown = int(math.Round(float64(cooldown) * CrisisCooldownScale(ctx.Silo)))
		}
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
	case "preparation_gte":
		value, ok := readPreparationMetric(ctx.Silo, trigger.Metric)
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
	case "world_flag_not_set":
		// 守卫型条件：世界事实未成立即通过。与 world_flag_false 的区别是
		// 没有世界上下文 (单堡模式) 时按"未设立"处理，而不是一律不成立——
		// 反制守卫不能因为运行模式把渠道永久封死。
		value, ok := readWorldFlag(ctx.World, trigger.Flag)
		return !ok || !value
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
	case "months_since_event_gte":
		// 相对时间门：锚定事件点亮月而非绝对日历。未点亮则不成立。
		state, ok := ctx.States[trigger.Category+":"+trigger.EventID]
		if !ok || !state.Triggered {
			return false
		}
		elapsed := contentGameTimestamp(ctx.Silo.CurrentYear, ctx.Silo.CurrentMonth) - state.LastTriggeredTimestamp
		return elapsed/contentDaysPerMonth >= trigger.Months
	case "actor_metric_gte":
		if ctx.Agent == nil {
			return false
		}
		var value float64
		switch trigger.Metric {
		case "propaganda_level":
			value = ctx.Agent.PropagandaLevel
		case "suspicion_level":
			value = ctx.Agent.SuspicionLevel
		case "political_prestige":
			value = DerivedPrestige(ctx.Agent.Ambition, ctx.Agent.Influence)
		case "ambition":
			value = ctx.Agent.Ambition
		case "influence":
			value = ctx.Agent.Influence
		default:
			return false
		}
		return value >= trigger.Value
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
		FireMode:    string(def.FireMode),
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
	// CompletionKeys 收集 player_action_complete 效果的动作 ID (服务层写入
	// 完成键并发总线事件)；nil = 无处投递 (引擎内直调时静默跳过)。
	CompletionKeys *[]string
	// BeliefObservations 收集 belief_observe 效果 (服务层落信念，因为玩家
	// agent 不在施加期的 silo 域内)；nil = 无处投递。
	BeliefObservations *[]BeliefObservationRequest
	// PublicEvidenceWrites 收集 public_evidence_register 效果；nil = 无处投递。
	PublicEvidenceWrites *[]model.PublicEvidenceRecord
	// VerifiedFacts 收集 evidence_verify 效果；nil = 无处投递。
	VerifiedFacts *[]string
	// WShocks 收集 authority_w_shock 效果；nil = 无处投递。
	WShocks *[]float64
	// DecisiveShocks 收集 decisive_shock 效果 (裁决的冲击通道)；nil = 无处投递。
	DecisiveShocks *[]DecisiveShockRequest
	// AllianceDeltas 收集 alliance_loyalty_delta 效果 (组名 -> 增量列表)；
	// nil = 无处投递。
	AllianceDeltas *[]AllianceLoyaltyDelta
	// EscapeDepartureSettle 收集 escape_departure_settle 效果：出发结算点
	// 请求 (16 §3.4 第一段结算的最小形态)。服务层在全部效果施加后执行结算
	// (结算读立场与账本，产出逃出群体；仲裁只读群体事实)。nil = 无处投递。
	EscapeDepartureSettle *bool
	// SpilloverSendKinds 收集 spillover_send 效果 (16 §3.5 Layer 3)：请求投递
	// 的信道种类按效果施加顺序排列。服务层在全部效果施加后逐条结算 (读信道
	// 账本、可靠度抽取、暴露记账、跨堡入队)。nil = 无处投递。
	SpilloverSendKinds *[]string
	// NegotiationOpenRequested 收集 negotiation_open 效果 (21 设计卷)：
	// 开信道请求，服务层在全部效果施加后执行 (含性情抽取与叙事)。
	// nil = 无处投递。
	NegotiationOpenRequested *bool
	// NegotiationRoundKind 收集 negotiation_round 效果 (21 设计卷)：
	// 高强度回合请求 (probe 试探 / terms 递条款)，服务层在全部效果施加后执行。
	// nil = 无处投递。
	NegotiationRoundKind *string
	// NegotiationResolveKind 收集 negotiation_resolve 效果 (21 设计卷)：
	// 妥协后二选一请求 (takeover 接管 / hands_off 不接管)，服务层执行。
	// nil = 无处投递。
	NegotiationResolveKind *string
	// Errors 收集目标解析失败等显式错误 (对齐 07 §12：不静默丢弃)。
	Errors *[]string
}

// AllianceLoyaltyDelta 一次联盟忠诚冲击。
type AllianceLoyaltyDelta struct {
	Group string // bailiff / it_internal
	Value float64
}

// DecisiveShockRequest 一次 decisive_shock 效果的落地请求 (裁决的冲击通道)。
// Source 为空时由服务层以事件键填充；Value 为 0 时取 DecisiveShockMagnitude。
type DecisiveShockRequest struct {
	Source string
	Value  float64
}

// BeliefObservationRequest 一次 belief_observe 效果的落库请求。ObservationID
// 由服务层按事件键生成 (内容事件不重放，同一事件同一事实只记一次)。
type BeliefObservationRequest struct {
	Observer      string // "player" / "residents" / 部门名
	FactID        string
	ObservationID string
	Reliability   float64
	Independence  float64
}

// CrossSignalRequest 一次 cross_signal 效果产生的跨堡投递请求。
// 由服务层转换为 model.CrossEvent 进入 World.CrossQueue。
type CrossSignalRequest struct {
	SourceSilo  uint
	TargetScope string
	EventID     string
	DelayMonths int
}

// SpilloverPackageEvent 外溢包裹投递的保留跨堡事件 ID (16 §3.5 Layer 3)。
// 它不是内容事件定义：派发阶段识别该 ID 后转交服务层做接收结算
// (独立核验与再复制)，不走内容效果路径。
const SpilloverPackageEvent = "spillover_package"

// ApplyContentEffects 单堡兼容入口：语义等价于 ScopeContext{Self: silo}。
func ApplyContentEffects(effects []ContentEffect, silo *model.Silo) []ContentScheduledEvent {
	return ApplyContentEffectsScoped(effects, &ScopeContext{Self: silo})
}

// ApplyContentEffectsScoped 按 scope 施加效果。schedule_event 仍按本地调度
// 语义收集返回 (只作用于 self)；cross_signal 进入 ScopeContext.CrossSignals；
// player_action_complete 的动作 ID 进入 ScopeContext.CompletionKeys，由服务层
// 写入完成键并发总线事件 (效果施加顺序即写入顺序，确定性)。
func ApplyContentEffectsScoped(effects []ContentEffect, sc *ScopeContext) []ContentScheduledEvent {
	var scheduled []ContentScheduledEvent
	for _, effect := range effects {
		switch effect.Type {
		case "schedule_event":
			scheduled = append(scheduled, ContentScheduledEvent{
				EventID:     effect.EventID,
				DelayMonths: effect.DelayMonths,
			})
			continue
		case "player_action_complete":
			action := strings.TrimSpace(effect.Action)
			if action != "" && sc != nil && sc.CompletionKeys != nil {
				*sc.CompletionKeys = append(*sc.CompletionKeys, action)
			}
			continue
		case "belief_observe":
			if sc != nil && sc.BeliefObservations != nil && strings.TrimSpace(effect.Fragment) != "" {
				*sc.BeliefObservations = append(*sc.BeliefObservations, BeliefObservationRequest{
					Observer:     strings.TrimSpace(effect.Observer),
					FactID:       strings.TrimSpace(effect.Fragment),
					Reliability:  effect.Reliability,
					Independence: 1,
				})
			}
			continue
		case "public_evidence_register":
			if sc != nil && sc.PublicEvidenceWrites != nil && strings.TrimSpace(effect.Fragment) != "" {
				*sc.PublicEvidenceWrites = append(*sc.PublicEvidenceWrites, model.PublicEvidenceRecord{
					FactID:     strings.TrimSpace(effect.Fragment),
					Provenance: effect.Provenance,
					Coverage:   effect.Coverage,
				})
			}
			continue
		case "evidence_verify":
			if sc != nil && sc.VerifiedFacts != nil && strings.TrimSpace(effect.Fragment) != "" {
				*sc.VerifiedFacts = append(*sc.VerifiedFacts, strings.TrimSpace(effect.Fragment))
			}
			continue
		case "authority_w_shock":
			if sc != nil && sc.WShocks != nil && effect.Value != 0 {
				*sc.WShocks = append(*sc.WShocks, effect.Value)
			}
			continue
		case "decisive_shock":
			if sc != nil && sc.DecisiveShocks != nil {
				*sc.DecisiveShocks = append(*sc.DecisiveShocks, DecisiveShockRequest{
					Source: strings.TrimSpace(effect.Flag),
					Value:  effect.Value,
				})
			}
			continue
		case "alliance_loyalty_delta":
			if sc != nil && sc.AllianceDeltas != nil && effect.Value != 0 {
				*sc.AllianceDeltas = append(*sc.AllianceDeltas, AllianceLoyaltyDelta{
					Group: effect.Observer,
					Value: effect.Value,
				})
			}
			continue
		case "escape_departure_settle":
			// 16 §3.4：出发结算点请求。结算在所有效果 (含立场终局重估)
			// 施加完毕后由服务层执行一次。
			if sc != nil && sc.EscapeDepartureSettle != nil {
				*sc.EscapeDepartureSettle = true
			}
			continue
		case "spillover_send":
			// 16 §3.5 (Layer 3)：投递请求按效果顺序收集，服务层结算。
			if sc != nil && sc.SpilloverSendKinds != nil {
				*sc.SpilloverSendKinds = append(*sc.SpilloverSendKinds, effect.Flag)
			}
			continue
		case "negotiation_open":
			// 21 设计卷：开信道请求 (性情抽取与叙事由服务层执行)。
			if sc != nil && sc.NegotiationOpenRequested != nil {
				*sc.NegotiationOpenRequested = true
			}
			continue
		case "negotiation_round":
			// 21 设计卷：高强度回合请求 (flag=probe|terms)。
			if sc != nil && sc.NegotiationRoundKind != nil {
				*sc.NegotiationRoundKind = effect.Flag
			}
			continue
		case "negotiation_resolve":
			// 21 设计卷：妥协后二选一请求 (flag=takeover|hands_off)。
			if sc != nil && sc.NegotiationResolveKind != nil {
				*sc.NegotiationResolveKind = effect.Flag
			}
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
	case "top_stance_reeval":
		// 16 §3.2：立场重估只作用于驻地 (self)；权重输入含世界事实
		// (Silo 1 命令源)，因此在 scoped 层施加。随机源走引擎单一 RNG 流。
		if sc.Self != nil {
			ReevalTopStance(sc.Self, sc.World, effect.Flag)
		}
		return
	case "top_stance_guarantee":
		if sc.Self != nil {
			addTopStanceGuarantee(sc.Self, effect.Value)
		}
		return
	case "preparation_delta":
		// 16 §6.2 (Layer 1)：账本写入只作用于驻地；非法写入显式失败。
		if err := applyPreparationDelta(sc.Self, effect); err != nil {
			sc.reportError(err.Error())
		}
		return
	case "preparation_transfer":
		// 划拨即扣减；余额不足显式失败 (不部分划拨、不静默跳过)。
		if err := applyPreparationTransfer(sc.Self, effect.Metric, effect.Value); err != nil {
			sc.reportError(err.Error())
		}
		return
	case "spillover_channel_add":
		if err := applySpilloverChannelAdd(sc.Self, effect.Flag, effect.Count, effect.Value); err != nil {
			sc.reportError(err.Error())
		}
		return
	case "escape_org_suspicion_delta":
		// 20 修订/ADR-0014：组织行为累积组织怀疑度 (囤积不驱动)。
		AddEscapeOrgSuspicion(sc.Self, effect.Value)
		return
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
			applyProfessionMetricDelta(silo, &silo.Professions[i], effect.Metric, effect.Value)
		}
	case "profession_metric_delta":
		if prof := findDept(silo, effect.Profession); prof != nil {
			applyProfessionMetricDelta(silo, prof, effect.Metric, effect.Value)
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
	case "cohort_ideology_delta":
		applyContentCohortIdeologyDelta(silo, effect.Profession, effect.Ideology, effect.Value)
	case "authority_evidence_delta":
		applyContentAuthorityEvidenceDelta(silo, effect.Value)
	case "authority_public_fact":
		applyContentAuthorityPublicFact(silo, effect.Flag)
	case "authority_pact_qualify":
		if silo != nil {
			silo.Authority.Judicial.PactQualified = effect.BoolValue
		}
	case "escape_group_depart":
		applyContentEscapeGroupDepart(silo, effect.Value)
	case "escape_group_status":
		if silo != nil && silo.EscapedGroup.Initialized {
			silo.EscapedGroup.Surviving = effect.BoolValue
		}
	case "schedule_event":
		// Handled at the service level where the runtime can rewrite the main event queue.
	}
}

// applyContentCohortIdeologyDelta G4：cohort 思潮直写。与 profession_ideology_delta_all
// 不同，它写入人口真源 (Cohort.Ideologies)，不会被每 tick 的派生视图覆写。
// profession 非空时只作用于该部门的 cohort；cohort 按切片既有 (持久化) 顺序处理。
func applyContentCohortIdeologyDelta(silo *model.Silo, profession, ideology string, delta float64) {
	if silo == nil || ideology == "" {
		return
	}
	professionID := uint(0)
	filter := strings.TrimSpace(profession) != ""
	if filter {
		found := false
		for i := range silo.Professions {
			if silo.Professions[i].Name == profession {
				professionID = silo.Professions[i].ID
				found = true
				break
			}
		}
		if !found {
			return
		}
	}
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if filter && cohort.ProfessionID != professionID {
			continue
		}
		if cohort.Ideologies == nil {
			cohort.Ideologies = map[string]float64{}
		}
		cohort.Ideologies[ideology] = clampUnit(cohort.Ideologies[ideology] + delta)
	}
}

// applyContentAuthorityEvidenceDelta G2：公开证据强度 Q 的内容供给 (0–1 截断)。
func applyContentAuthorityEvidenceDelta(silo *model.Silo, delta float64) {
	if silo == nil {
		return
	}
	j := &silo.Authority.Judicial
	j.PublicEvidenceQ = clampUnit(j.PublicEvidenceQ + delta)
}

// applyContentAuthorityPublicFact G2：登记一条独立公共事实。按键去重、按登记
// 顺序追加 (确定性)；计数与键表长度保持一致，供 pact 锁定动力学读取。
func applyContentAuthorityPublicFact(silo *model.Silo, key string) {
	if silo == nil || strings.TrimSpace(key) == "" {
		return
	}
	j := &silo.Authority.Judicial
	for _, existing := range j.PublicFactKeys {
		if existing == key {
			return
		}
	}
	j.PublicFactKeys = append(j.PublicFactKeys, key)
	j.IndependentPublicFacts = len(j.PublicFactKeys)
}

// applyContentEscapeGroupDepart G5：把 n 人从堡内移入逃出群体。移出即堡外
// 存续事实：群体记为存续，首次离开记录时间戳。人数不得超过当前堡内人口。
func applyContentEscapeGroupDepart(silo *model.Silo, count float64) {
	if silo == nil {
		return
	}
	n := int(count)
	if n <= 0 || silo.TotalPopulation <= 0 {
		return
	}
	if n > silo.TotalPopulation {
		n = silo.TotalPopulation
	}
	silo.TotalPopulation -= n
	group := &silo.EscapedGroup
	group.Population += n
	group.Surviving = true
	if !group.Initialized {
		group.Initialized = true
		group.DepartedYear = silo.CurrentYear
		group.DepartedMonth = silo.CurrentMonth
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
		// 裂隙档放大叛乱累积的正向增量 (19 §3.2)；空原型恒等。
		silo.Rebellion = clampUnit(silo.Rebellion + AmplifyOrderDelta(silo, metric, delta))
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
	case "safeguard_risk":
		// G6: content may now raise or lower intervention risk. The lower bound
		// matches the mechanic layer; the death threshold stays a victory_check
		// readout, not an effect-side cap.
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+delta)
	}
}

func applyProfessionMetricDelta(silo *model.Silo, profession *model.Profession, metric string, delta float64) {
	switch metric {
	case "panic_value":
		// 裂隙档放大恐慌抬升的正向增量 (19 §3.2)；回落与空原型恒等。
		profession.PanicValue = clampUnit(profession.PanicValue + AmplifyOrderDelta(silo, metric, delta))
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
	case "safeguard_risk":
		return silo.SafeguardRisk, true
	case "order_pressure":
		return OrderPressure(silo), true
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
		if value {
			// G4: snapshot the station's cohort ideologies at the exact moment
			// the destruction fact lands, before aftermath panic propagates.
			if player := siloByID(sc, world.PlayerSiloID); player != nil && !player.Silo1Destroyed {
				takePreFallIdeologySnapshot(player)
			}
		}
		world.Silo1.Alive = !value
		if player := siloByID(sc, world.PlayerSiloID); player != nil {
			player.Silo1Destroyed = value
		}
		return
	}
	world.Flags[flag] = value
}

// takePreFallIdeologySnapshot G4：覆灭前夕思潮快照。读取 Cohort 的人口加权
// 思潮 (真源，不是部门派生视图)；cohort 按 ID 升序、思潮轴按名称升序累加，
// 保证跨进程逐位一致。只在首次置真时截取一次。
//
// 资格口径 (结局体系 12 §3.3)：亲外思潮为主导或复合主导一极即合格；纯种
// 地堡思潮 (忠诚主导) 一律不合格。亲外权重必须为正——资格只能来自覆灭前
// 建成的可核验交往经历，不能由覆灭后的恐慌产生。
func takePreFallIdeologySnapshot(silo *model.Silo) {
	if silo == nil || silo.PreFallSnapshot.Taken {
		return
	}
	cohorts := append([]model.PopulationCohort(nil), silo.Cohorts...)
	sort.SliceStable(cohorts, func(i, j int) bool { return cohorts[i].ID < cohorts[j].ID })

	axisSet := map[string]bool{}
	for i := range cohorts {
		for axis := range cohorts[i].Ideologies {
			axisSet[axis] = true
		}
	}
	axes := make([]string, 0, len(axisSet))
	for axis := range axisSet {
		axes = append(axes, axis)
	}
	sort.Strings(axes)

	weighted := map[string]float64{}
	totalWeight := 0.0
	for i := range cohorts {
		cohort := &cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		w := float64(cohort.Count)
		totalWeight += w
		for _, axis := range axes {
			weighted[axis] += clampUnit(cohort.Ideologies[axis]) * w
		}
	}
	snapshot := model.PreFallIdeologySnapshot{
		Taken:      true,
		Year:       silo.CurrentYear,
		Month:      silo.CurrentMonth,
		Population: totalPopulationOf(cohorts),
		Ideologies: map[string]float64{},
	}
	if totalWeight > 0 {
		dominant := ""
		best := -1.0
		for _, axis := range axes {
			value := weighted[axis] / totalWeight
			snapshot.Ideologies[axis] = value
			if value > best {
				best = value
				dominant = axis
			}
		}
		snapshot.DominantAxis = dominant
		snapshot.OpenQualified = snapshot.Ideologies[model.IdeologyProForeign] > 0 &&
			snapshot.Ideologies[model.IdeologyProForeign] >= snapshot.Ideologies[model.IdeologyLoyalty]
	}
	silo.PreFallSnapshot = snapshot
}

func totalPopulationOf(cohorts []model.PopulationCohort) int {
	total := 0
	for i := range cohorts {
		if cohorts[i].Count > 0 {
			total += cohorts[i].Count
		}
	}
	return total
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
		if value && silo != nil && !silo.Silo1Destroyed {
			takePreFallIdeologySnapshot(silo)
		}
		silo.Silo1Destroyed = value
	}
}

func clampUnit(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func contentGameTimestamp(year, month int) int {
	return ((year-contentTimestampBaseYear)*contentMonthsPerYear + (month - 1)) * contentDaysPerMonth
}

// ContentGameTimestamp exposes the content timestamp scale (days since the
// base year) for service-side deadline evaluation.
func ContentGameTimestamp(year, month int) int {
	return contentGameTimestamp(year, month)
}

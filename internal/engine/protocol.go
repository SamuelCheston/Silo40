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

// Protocol 条款类别 (10-it-control-credibility §6.1)。
const (
	ProtocolProhibition      = "prohibition"
	ProtocolRequiredResponse = "required_response"
	ProtocolOutcomeTarget    = "outcome_target"
	ProtocolReporting        = "reporting"
	ProtocolGap              = "gap"
)

// 偏离严重度。
const (
	SeverityMinor       = "minor"
	SeverityMajor       = "major"
	SeverityExistential = "existential" // 仅违反
	SeverityCritical    = "critical"    // 未贯彻/未达成的最高档
)

// 三类偏离 (§6.2)。
const (
	DeviationViolation = "violation" // 违反：行为命中禁止条款
	DeviationLapse     = "lapse"     // 未贯彻：必需动作/报告超过 deadline
	DeviationUnmet     = "unmet"     // 未达成：review 日真实目标越界
)

// 介入链阶段 (§6.3)。
const (
	InterventionNone       = ""
	InterventionWarning    = "warning"    // J≥1.0：带条款原因的异常预警
	InterventionCorrection = "correction" // J≥1.5 且连续两次 review 未恢复：限期纠正
	InterventionOverride   = "override"   // 纠正失败：权威覆写/重置
	InterventionAssessment = "assessment" // Safeguard 评估 (存在级违反直达)
)

// ProtocolFile events/protocol/*.json 的文件结构。
type ProtocolFile struct {
	Version int                    `json:"version"`
	Clauses []model.ProtocolClause `json:"clauses"`
}

// LoadProtocolFile 读取并校验一个 Protocol 定义文件。
func LoadProtocolFile(path string) (ProtocolFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProtocolFile{}, err
	}
	var file ProtocolFile
	if err := json.Unmarshal(data, &file); err != nil {
		return ProtocolFile{}, fmt.Errorf("parse protocol %s: %w", path, err)
	}
	if file.Version <= 0 {
		return ProtocolFile{}, fmt.Errorf("protocol %s: missing version", path)
	}
	ids := map[string]bool{}
	for i := range file.Clauses {
		clause := &file.Clauses[i]
		clause.ID = strings.TrimSpace(clause.ID)
		clause.Category = strings.ToLower(strings.TrimSpace(clause.Category))
		clause.Severity = strings.ToLower(strings.TrimSpace(clause.Severity))
		if clause.ID == "" || ids[clause.ID] {
			return ProtocolFile{}, fmt.Errorf("protocol %s: duplicate or empty clause id", path)
		}
		ids[clause.ID] = true
		switch clause.Category {
		case ProtocolProhibition, ProtocolRequiredResponse, ProtocolOutcomeTarget, ProtocolReporting, ProtocolGap:
		default:
			return ProtocolFile{}, fmt.Errorf("protocol clause %s: unsupported category %q", clause.ID, clause.Category)
		}
		switch clause.Severity {
		case SeverityMinor, SeverityMajor, SeverityExistential, SeverityCritical:
		default:
			return ProtocolFile{}, fmt.Errorf("protocol clause %s: unsupported severity %q", clause.ID, clause.Severity)
		}
	}
	sort.Slice(file.Clauses, func(i, j int) bool { return file.Clauses[i].ID < file.Clauses[j].ID })
	return file, nil
}

// LoadProtocolDir 读取 events/protocol 目录下的全部条款文件并按条款 ID 合并。
// 返回真实条款集与配置版本号。
func LoadProtocolDir(dir string) ([]model.ProtocolClause, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	var clauses []model.ProtocolClause
	version := 0
	for _, name := range names {
		file, err := LoadProtocolFile(filepath.Join(dir, name))
		if err != nil {
			return nil, 0, err
		}
		if file.Version > version {
			version = file.Version
		}
		clauses = append(clauses, file.Clauses...)
	}
	sort.Slice(clauses, func(i, j int) bool { return clauses[i].ID < clauses[j].ID })
	return clauses, version, nil
}

// EnsureProtocolInitialized 把真实条款拷为 believed 副本 (初始无伪造)。
// 只在 believed 副本尚不存在时写入；条款集之后的加载会补齐空副本。
func EnsureProtocolInitialized(silo *model.Silo, trueClauses []model.ProtocolClause, configVersion int) {
	if silo == nil {
		return
	}
	state := &silo.Protocol
	state.ConfigVersion = configVersion
	if len(state.BelievedClauses) == 0 && len(trueClauses) > 0 {
		state.BelievedClauses = append([]model.ProtocolClause(nil), trueClauses...)
		state.BelievedVersion = configVersion
	}
	if state.ReviewAnchors == nil {
		state.ReviewAnchors = map[string]int{}
	}
}

// deviationPenalties §6.2 分级表：R_s 增量 (每周期)、W 惩罚 (需联盟知悉)、
// 介入分。周期含义按类别：违反=一次，未贯彻=每逾期月，未达成=每次 review。
func deviationPenalties(kind, severity string) (rS, wPenalty, intervention float64) {
	switch kind {
	case DeviationViolation:
		switch severity {
		case SeverityMinor:
			return 0.02, 2, 0.2
		case SeverityMajor:
			return 0.08, 5, 0.6
		case SeverityExistential:
			return 0.25, 10, 1.2
		}
	case DeviationLapse:
		switch severity {
		case SeverityMinor:
			return 0.01, 1, 0.15
		case SeverityMajor:
			return 0.03, 3, 0.40
		case SeverityCritical:
			return 0.06, 5, 0.80
		}
	case DeviationUnmet:
		switch severity {
		case SeverityMinor:
			return 0.02, 2, 0.25
		case SeverityMajor:
			return 0.05, 4, 0.60
		case SeverityCritical:
			return 0.10, 6, 1.00
		}
	}
	return 0, 0, 0
}

func protocolTimestamp(year, month int) int {
	return contentGameTimestamp(year, month)
}

func findDeviation(state *model.ProtocolState, clauseID, kind string) *model.ProtocolDeviation {
	for i := range state.Deviations {
		if state.Deviations[i].ClauseID == clauseID && state.Deviations[i].Kind == kind {
			return &state.Deviations[i]
		}
	}
	return nil
}

// recordDeviation 记录或延长一条偏离 (§6.2：每条款每类只记一次)。
// W 惩罚只在关键联盟知悉时发生；J 是权威系统结果，不依赖舆论。偏离不再
// 直接累加风险度，改按期数登记事件 (ADR-0019)，经观测管线进入 1 号信念。
func recordDeviation(silo *model.Silo, clause model.ProtocolClause, kind string, acknowledged bool, periods int) {
	state := &silo.Protocol
	if periods <= 0 {
		periods = 1
	}
	rS, wPenalty, _ := deviationPenalties(kind, clause.Severity)
	dev := findDeviation(state, clause.ID, kind)
	if dev == nil {
		state.Deviations = append(state.Deviations, model.ProtocolDeviation{
			ClauseID:     clause.ID,
			Kind:         kind,
			Severity:     clause.Severity,
			Year:         silo.CurrentYear,
			Month:        silo.CurrentMonth,
			Open:         true,
			MissedCount:  periods,
			Acknowledged: acknowledged,
		})
	} else {
		dev.MissedCount += periods
		dev.Open = true
		if acknowledged {
			dev.Acknowledged = true
		}
	}
	applySafeguardConsequence(silo, SafeguardIncidentDeviation, rS, periods)
	if acknowledged && wPenalty > 0 {
		QueueAuthorityShock(silo, -wPenalty*float64(periods))
	}
}

// QueueAuthorityShock 把一次 W 冲击放入队列，只在下一次月度权威更新消费一次
// (§3.2 的 Σδe；单月合计 cap 12 在消费处执行)。
func QueueAuthorityShock(silo *model.Silo, delta float64) {
	if silo == nil || delta == 0 {
		return
	}
	silo.Authority.Shocks = append(silo.Authority.Shocks, delta)
}

// recomputeJ §6.3：J = Σ 未关闭偏离的 intervention_points。
func recomputeJ(silo *model.Silo, clauses []model.ProtocolClause) {
	byID := map[string]model.ProtocolClause{}
	for _, clause := range clauses {
		byID[clause.ID] = clause
	}
	state := &silo.Protocol
	j := 0.0
	for _, dev := range state.Deviations {
		if !dev.Open {
			continue
		}
		clause, ok := byID[dev.ClauseID]
		if !ok {
			continue
		}
		j += clause.InterventionPoints
	}
	state.J = j
}

// RecordProtocolAction 行为命中禁止条款即构成违反 (真实判定，不受 IT 认知
// 影响)。actionKey 与条款的 forbidden_action 精确匹配；每条款只记一次。
// acknowledged 表示关键联盟当场知悉 (决定 W 惩罚)。
func RecordProtocolAction(silo *model.Silo, clauses []model.ProtocolClause, actionKey string, acknowledged bool) {
	if silo == nil || strings.TrimSpace(actionKey) == "" {
		return
	}
	for _, clause := range clauses {
		if clause.Category != ProtocolProhibition || clause.ForbiddenAction != actionKey {
			continue
		}
		if findDeviation(&silo.Protocol, clause.ID, DeviationViolation) != nil {
			continue
		}
		recordDeviation(silo, clause, DeviationViolation, acknowledged, 1)
		if clause.Severity == SeverityExistential {
			// 存在级违反立即进入一次权威审查 (§6.3)。
			silo.Protocol.InterventionStage = InterventionAssessment
			silo.Protocol.StageMonths = 0
		}
	}
	recomputeJ(silo, clauses)
}

// ActivateProtocolRequirement 当触发条件成立时激活必需动作限期。
func ActivateProtocolRequirement(silo *model.Silo, clauses []model.ProtocolClause, triggerMetric string) {
	if silo == nil {
		return
	}
	state := &silo.Protocol
	now := protocolTimestamp(silo.CurrentYear, silo.CurrentMonth)
	for _, clause := range clauses {
		if (clause.Category != ProtocolRequiredResponse && clause.Category != ProtocolReporting) ||
			clause.TriggerMetric != triggerMetric || clause.DeadlineMonths <= 0 {
			continue
		}
		alreadyPending := false
		for _, pending := range state.Pending {
			if pending.ClauseID == clause.ID {
				alreadyPending = true
				break
			}
		}
		if alreadyPending || findDeviation(state, clause.ID, DeviationLapse) != nil {
			continue
		}
		state.Pending = append(state.Pending, model.PendingRequirement{
			ClauseID:       clause.ID,
			DueTimestamp:   now + clause.DeadlineMonths*contentDaysPerMonth,
			ActivatedYear:  silo.CurrentYear,
			ActivatedMonth: silo.CurrentMonth,
		})
	}
}

// CompleteProtocolRequirement 完成必需动作：撤销未逾期的限期；已构成未贯彻的
// 偏离被关闭 (真实结果不可伪造消除，但停止继续恶化)。
func CompleteProtocolRequirement(silo *model.Silo, clauses []model.ProtocolClause, actionKey string) {
	if silo == nil || strings.TrimSpace(actionKey) == "" {
		return
	}
	state := &silo.Protocol
	byID := map[string]model.ProtocolClause{}
	for _, clause := range clauses {
		byID[clause.ID] = clause
	}
	kept := state.Pending[:0]
	for _, pending := range state.Pending {
		clause, ok := byID[pending.ClauseID]
		if ok && clause.RequiredAction == actionKey {
			continue // 完成：撤销限期
		}
		kept = append(kept, pending)
	}
	state.Pending = kept
	for i := range state.Deviations {
		dev := &state.Deviations[i]
		if dev.Kind != DeviationLapse || !dev.Open {
			continue
		}
		if clause, ok := byID[dev.ClauseID]; ok && clause.RequiredAction == actionKey {
			dev.Open = false
		}
	}
	recomputeJ(silo, clauses)
}

// AdvanceProtocolMonthly 月度 Protocol 结算：必需动作限期、目标 review、
// J 重算与介入链推进。确定性，无 RNG。在权威月度更新之后、时钟推进之前调用。
// 真实条款判定，与 IT 的 believed 副本无关 (无知不改变真实判定)。
func (e *GameEngine) AdvanceProtocolMonthly(silo *model.Silo) {
	if silo == nil {
		return
	}
	clauses := e.ProtocolClauses
	state := &silo.Protocol
	if len(state.BelievedClauses) == 0 {
		EnsureProtocolInitialized(silo, clauses, e.ProtocolVersion)
	}
	if state.ReviewAnchors == nil {
		state.ReviewAnchors = map[string]int{}
	}
	now := protocolTimestamp(silo.CurrentYear, silo.CurrentMonth)

	byID := map[string]model.ProtocolClause{}
	for _, clause := range clauses {
		byID[clause.ID] = clause
	}

	// 0. 触发条件成立的必需动作/报告条款激活限期。
	for _, clause := range clauses {
		if (clause.Category != ProtocolRequiredResponse && clause.Category != ProtocolReporting) ||
			clause.TriggerMetric == "" || clause.DeadlineMonths <= 0 {
			continue
		}
		value, readable := readSiloMetric(silo, clause.TriggerMetric)
		if !readable || value < clause.TriggerAtLeast {
			continue
		}
		ActivateProtocolRequirement(silo, clauses, clause.TriggerMetric)
	}

	// 1. 必需动作/报告限期：逾期即未贯彻并每月继续恶化。
	kept := state.Pending[:0]
	for _, pending := range state.Pending {
		clause, ok := byID[pending.ClauseID]
		if !ok {
			continue
		}
		if now >= pending.DueTimestamp {
			recordDeviation(silo, clause, DeviationLapse, true, 1)
			kept = append(kept, pending) // 保留直到动作完成
			continue
		}
		kept = append(kept, pending)
	}
	state.Pending = kept

	// 2. 目标条款 review：到点读真实结果，不看 IT 是否努力。
	for _, clause := range clauses {
		if clause.Category != ProtocolOutcomeTarget || clause.ReviewIntervalMonths <= 0 || clause.TargetMetric == "" {
			continue
		}
		anchor, ok := state.ReviewAnchors[clause.ID]
		if !ok {
			state.ReviewAnchors[clause.ID] = now
			continue
		}
		if now-anchor < clause.ReviewIntervalMonths*contentDaysPerMonth {
			continue
		}
		state.ReviewAnchors[clause.ID] = now
		value, readable := readSiloMetric(silo, clause.TargetMetric)
		if !readable {
			continue
		}
		outOfRange := (clause.TargetMin != nil && value < *clause.TargetMin) ||
			(clause.TargetMax != nil && value > *clause.TargetMax)
		dev := findDeviation(state, clause.ID, DeviationUnmet)
		if outOfRange {
			recordDeviation(silo, clause, DeviationUnmet, true, 1)
			if clause.Severity != SeverityMinor {
				state.FailedReviews++
			}
		} else if dev != nil && dev.Open {
			dev.Open = false
			state.FailedReviews = 0
		}
	}

	// 3. J 重算与介入链 (§6.3：罕见、确定、可预警，无随机)。
	recomputeJ(silo, clauses)
	switch state.InterventionStage {
	case InterventionNone:
		if state.J >= 1.0 && !state.WarningIssued {
			state.InterventionStage = InterventionWarning
			state.WarningIssued = true
			state.StageMonths = 0
		}
		if state.J >= 1.5 && state.FailedReviews >= 2 {
			state.InterventionStage = InterventionCorrection
			state.StageMonths = 0
		}
	case InterventionWarning:
		if state.J >= 1.5 && state.FailedReviews >= 2 {
			state.InterventionStage = InterventionCorrection
			state.StageMonths = 0
		} else if state.J < 1.0 {
			state.InterventionStage = InterventionNone
		}
	case InterventionCorrection:
		state.StageMonths++
		if state.J < 1.0 {
			state.InterventionStage = InterventionNone
			state.FailedReviews = 0
			break
		}
		// TODO(balance): 限期纠正 2 月为 sim 起点。
		if state.StageMonths >= 2 {
			state.InterventionStage = InterventionOverride
			state.StageMonths = 0
			// 权威覆写/重置：一次性冲击 (TODO(balance))。
			silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+0.25)
			QueueAuthorityShock(silo, -10)
		}
	case InterventionOverride:
		state.StageMonths++
		if state.StageMonths >= 1 {
			state.InterventionStage = InterventionAssessment
			state.StageMonths = 0
		}
	case InterventionAssessment:
		state.StageMonths++
		if state.StageMonths >= 1 {
			// R_s ≥ 1 的硬失败由胜负判定独立读取；评估结束回到无接触。
			state.InterventionStage = InterventionNone
			state.StageMonths = 0
			state.FailedReviews = 0
		}
	}
	// 控制通道仅在介入链开启 (§6.3/§6.5)。
	state.ChannelOpen = state.InterventionStage == InterventionCorrection ||
		state.InterventionStage == InterventionOverride ||
		state.InterventionStage == InterventionAssessment
}

// GCognitionTier §6.4 认知梯度。
func GCognitionTier(g float64) string {
	switch {
	case g < 0.10:
		return "apprentice_unaware"
	case g < 0.35:
		return "senses_hidden_authors"
	case g < 0.65:
		return "infers_cross_silo_network"
	case g < 0.85:
		return "understands_origin"
	default:
		return "understands_limits"
	}
}

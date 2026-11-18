package service

// 黑箱缺口通道 (22 黑箱学习系列 02 卷 §6 缺口清单)。属缺口通道格实现。
//
// 范围：G2 举报管道、G3 Order 例行指令账本、G5 人口报告通道、G7 注意力预算
// 节流、G9 异常观察计数器、G1 稽查机制最小骨架。
//
// 明确不做 (02 卷 §6 处置)：
//   - G4 部门产出缺口汇总上报：并入 (c) 执行反馈，不单独成观测量，无代码。
//   - G6 专线信号质量/逐堡心跳：首版放弃，仅 02 卷 §3.6 留设计占位。
//   - G8 Protocol 真值审判与黑箱并存：已裁决双轨分离 (00 卷裁决 4)，无代码。
//
// 纪律：门控同调度器 (功能关闭/未启用早退)；确定性优先，本文件零 RNG 消耗；
// 全部数值参数 TODO(balance)。扫描对象为各目标堡——1 号自身 (观察者) 不在
// 自己的观测通道里，与指标一排除 1 号同口径 (stewardshipExperimentIsSilo1)。

import (
	"fmt"
	"sort"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 缺口通道数值参数。全部 TODO(balance)，起点值见 22-02 §3/§6。
const (
	// bbChannelCensusDropThreshold G5 人口环比跌幅异常阈值 (22-02 §3.4 的
	// 最小定量替代：「某堡可能出事了」)。
	bbChannelCensusDropThreshold = 0.02 // TODO(balance)
	// bbChannelAuditPDangerThreshold G1 被动专项核查的 P_danger 门槛：1 号自己
	// 的危险区概率估计越过该值才值得额外花一次主动观测。
	bbChannelAuditPDangerThreshold = 0.5 // TODO(balance)
	// bbChannelAuditExecutionFloor G1 执行门槛：稽查由本地执行层落实，执行强度
	// E 低于该值即「稽查空转」——高指派把 Θ 推高后 E→0，1 号再怎么起疑也查
	// 不动 (22 系列 §8 A2「该堡 IT 服从/稽查显著退化」)。TODO(balance)
	bbChannelAuditExecutionFloor = 0.15
	// bbChannelAuditRoutineMinRisk 例行稽查的最低指派档：稽查频率随 r 分配
	// (22-02 §3.8)，低 r 无稽查动机。TODO(balance)
	bbChannelAuditRoutineMinRisk   = 0.20
	bbChannelAuditRoutinePeriodMid = 4 // r ∈ [0.20, 0.60)：每 4 月一次
	bbChannelAuditRoutinePeriodHi  = 2 // r ≥ 0.60：每 2 月一次
)

// SafeguardIncidents 台账键约定 (登记簿键契约，与 W5 Θ 行为化兼容)：
//
//   - 四类旧累加键 (engine.SafeguardIncident*：ignore / cover_collapse /
//     deviation / it_overreach) 由引擎侧写入，G9 月度汇总后清零 (消费)。
//   - 下列四个新键由通道回合 / W5 写入；G9 汇总时计入但不清零——保留给 W5
//     (blackbox_behavior.go) 读取一个周期，其生命周期由写入方/消费方自行管理。
//   - "obs" 前缀键预留给观测管线的簿记键，G9 汇总时排除 (非异常语义)。
const (
	bbChannelIncidentReportsReceived = "reports_received" // G2 写入；观测管线与 1 号因子消费
	bbChannelIncidentCensusAnomaly   = "census_anomaly"   // G5 写入
	bbChannelIncidentAuditFact       = "audit_fact"       // G1 写入；W5 的 f_risk 消费
	bbChannelIncidentEquipmentUse    = "equipment_use"    // W5 写入并管理 (此处仅登记契约)
)

// Experiment.Ledger 台账键约定：
//
//   - "census_prev:<siloID>"      G5 各堡上月人口水位 (int 人口数)。
//   - "attention_used:<月>:<siloID>" G7 主动观测槽占用标记 (值 1)；
//     月 = year*12+month 线性月份索引 (与指标一同口径)。
const (
	bbChannelLedgerCensusPrevPrefix = "census_prev:"
	bbChannelLedgerAttentionPrefix  = "attention_used:"
)

// runBlackBoxChannelsMonth 缺口通道月度回合 (2b' 固定顺序第三位)。
// 门控同调度器 (功能关闭由调度器早退；此处再防未启用零值)。逐堡按 SiloID
// 升序 (s.silos 序)，堡内固定顺序：G2 举报 → G5 人口 → G1 被动核查 →
// G9 汇总消费。全程零 RNG。
func (s *GameService) runBlackBoxChannelsMonth(logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return
	}
	for _, silo := range s.silos {
		if siloEnded(silo) || !silo.BlackBox.Initialized || s.stewardshipExperimentIsSilo1(silo) {
			continue
		}
		s.bbChannelReportsMonth(silo, logs)
		s.bbChannelCensusMonth(silo)
		s.bbChannelPassiveAuditMonth(silo, logs)
		s.bbChannelConsumeIncidentsMonth(silo, logs)
	}
}

// ===== G2 举报管道 =====

// bbChannelReportsMonth G2：扫描该堡部门信念层，任一事实浓度 ≥ QualReport
// (engine 配置只读引用，不硬编码) → 当月「报告送达」+1。每堡每月至多 +1
// (任一事实达标即送达一份报告)。写入 SafeguardIncidents 并记审计日志行。
// 确定性：部门/信念按切片序扫描，零 RNG。
func (s *GameService) bbChannelReportsMonth(silo *model.Silo, logs *[]string) {
	qualReport := engine.DefaultInformationConfig().QualReport
	qualified := false
	for pi := range silo.Professions {
		for _, belief := range silo.Professions[pi].Beliefs {
			if belief.B >= qualReport {
				qualified = true
				break
			}
		}
		if qualified {
			break
		}
	}
	if !qualified {
		return
	}
	bbChannelRegisterIncident(silo, bbChannelIncidentReportsReceived, 1)
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"[Silo1Stewardship] silo %d report received (department belief >= %.2f), reports_received=%d",
			silo.ID, qualReport, silo.SafeguardIncidents[bbChannelIncidentReportsReceived]))
	}
}

// ===== G3 Order 例行指令账本 =====

// silo1DirectiveEntry Order 例行指令账本条目 (从 BlackBox.Log 派生，不新增
// 存档字段)。
type silo1DirectiveEntry struct {
	Year  int
	Month int
	R     float64 // 指令内容 = 黑箱指派的 r
	Band  string  // 风险档位 R0–R5 (22-00 §2.2)
}

// silo1DirectiveLedger G3：近 12 月 (含当月) 的指令条目，时间升序。
// 指令 = 黑箱指派：BlackBox.Log 的 ChosenR 序列即指令到达事实，故账本完全
// 从 Log 派生，不新增存档字段。供 f_attn 流量比与玩家侧叙事复用。
// 未启用/无日志 → 空账本。
func (s *GameService) silo1DirectiveLedger(silo *model.Silo) []silo1DirectiveEntry {
	if silo == nil || s.world == nil {
		return nil
	}
	now := stewardshipExperimentMonthIndex(s.world.CurrentYear, s.world.CurrentMonth)
	var out []silo1DirectiveEntry
	for _, entry := range silo.BlackBox.Log {
		idx := stewardshipExperimentMonthIndex(entry.Year, entry.Month)
		if idx > now || idx < now-11 {
			continue
		}
		out = append(out, silo1DirectiveEntry{
			Year:  entry.Year,
			Month: entry.Month,
			R:     entry.ChosenR,
			Band:  silo1DirectiveBand(entry.ChosenR),
		})
	}
	return out
}

// silo1DirectiveBand r → 风险档位 (22-00 §2.2 映射，边界锚定动力学参数)。
// 菜单值 k/20 与边界常量为同一 float64 最近值，边界比较精确。
func silo1DirectiveBand(r float64) string {
	switch {
	case r < 0.10:
		return "R0" // 静默
	case r <= 0.35:
		return "R1" // 例行
	case r <= 0.50:
		return "R2" // 关注
	case r <= 0.65:
		return "R3" // 施压
	case r <= 0.85:
		return "R4" // 高压
	default:
		return "R5" // 满压
	}
}

// ===== G5 人口报告通道 =====

// bbChannelCensusMonth G5：记录该堡人口水位到台账，与上月记录环比；跌幅 ≥
// 阈值 → 登记 census_anomaly (「某堡可能出事了」的最小定量)。首月无上月
// 记录只落数。零 RNG。
func (s *GameService) bbChannelCensusMonth(silo *model.Silo) {
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	key := fmt.Sprintf("%s%d", bbChannelLedgerCensusPrevPrefix, silo.ID)
	prev, hasPrev := exp.Ledger[key]
	pop := silo.TotalPopulation
	if hasPrev && prev > 0 && pop < prev {
		drop := float64(prev-pop) / float64(prev)
		if drop >= bbChannelCensusDropThreshold {
			bbChannelRegisterIncident(silo, bbChannelIncidentCensusAnomaly, 1)
		}
	}
	exp.Ledger[key] = pop
}

// ===== G7 注意力预算 =====

// silo1ActiveObservationAllowed G7 节流纪律：每堡每月最多一次主动观测。
// 用 Experiment.Ledger["attention_used:<月>:<siloID>"] 标记占用。
// 待 22-03 探索层落地后接入主动稽查/核查触发点 (本轮调用方为 G1 被动核查)。
// 未启用 (零值) → false (保守：无注意力预算)。
func (s *GameService) silo1ActiveObservationAllowed(siloID uint) bool {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return false
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	return exp.Ledger[bbChannelAttentionKey(s.world.CurrentYear, s.world.CurrentMonth, siloID)] == 0
}

// silo1ConsumeActiveObservation 占用该堡当月的主动观测槽 (调用方须先经
// silo1ActiveObservationAllowed 确认)。
func (s *GameService) silo1ConsumeActiveObservation(siloID uint) {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	exp.Ledger[bbChannelAttentionKey(s.world.CurrentYear, s.world.CurrentMonth, siloID)] = 1
}

// bbChannelAttentionKey G7 台账键：月 = year*12+month 线性索引。
func bbChannelAttentionKey(year, month int, siloID uint) string {
	return fmt.Sprintf("%s%d:%d", bbChannelLedgerAttentionPrefix, stewardshipExperimentMonthIndex(year, month), siloID)
}

// ===== G1 稽查机制 (最小骨架) =====

// bbChannelPassiveAuditMonth G1 稽查：触发来源有二——
//  1. 被动专项核查：该堡黑箱日志最新一条 (当月 2b 决策留痕) 的 P_danger ≥ 门槛；
//  2. 例行稽查：稽查频率随指派 r 分配 (22-02 §3.8)，r 越高排期越密。
//
// 触发只是「1 号要查」；能否查出东西取决于本地执行层——执行强度 E 低于门槛
// 即稽查空转 (22 系列 §8 A2：极高指派 → 服从/稽查显著退化)，不登记事实、
// 不占用观测槽。实际执行的稽查记一条 audit_fact (供 W5 的 f_risk 消费) + 审计
// 日志行，并占用当月主动观测槽。主动稽查操作 (注意力成本/暴露/延迟结算) 待
// 22-03 探索层落地，不新增玩家动作、不碰 action_gates。零 RNG。
func (s *GameService) bbChannelPassiveAuditMonth(silo *model.Silo, logs *[]string) {
	bb := &silo.BlackBox
	if len(bb.Log) == 0 {
		return
	}
	last := bb.Log[len(bb.Log)-1]
	emergency := last.PDanger >= bbChannelAuditPDangerThreshold
	routine := bbChannelRoutineAuditDue(bb.LastAssignedRisk, s.world.CurrentYear, s.world.CurrentMonth)
	if !emergency && !routine {
		return
	}
	if !s.silo1ActiveObservationAllowed(silo.ID) {
		return
	}
	if bb.CurrentExecution < bbChannelAuditExecutionFloor {
		if logs != nil {
			*logs = append(*logs, fmt.Sprintf(
				"[Silo1Stewardship] silo %d inspection postponed (execution %.2f < %.2f)",
				silo.ID, bb.CurrentExecution, bbChannelAuditExecutionFloor))
		}
		return
	}
	s.silo1ConsumeActiveObservation(silo.ID)
	bbChannelRegisterIncident(silo, bbChannelIncidentAuditFact, 1)
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"[Silo1Stewardship] silo %d audit carried out (P_danger %.2f, r %.2f, execution %.2f), audit_fact=%d",
			silo.ID, last.PDanger, bb.LastAssignedRisk, bb.CurrentExecution,
			silo.SafeguardIncidents[bbChannelIncidentAuditFact]))
	}
}

// bbChannelRoutineAuditDue 例行稽查排期：低 r 无稽查动机；r 越高周期越短。
// 纯月索引取模，零 RNG。
func bbChannelRoutineAuditDue(r float64, year, month int) bool {
	if r < bbChannelAuditRoutineMinRisk {
		return false
	}
	period := bbChannelAuditRoutinePeriodMid
	if r >= bbChannelAuditRoutineMinRisk+0.40 { // r ≥ 0.60
		period = bbChannelAuditRoutinePeriodHi
	}
	return stewardshipExperimentMonthIndex(year, month)%period == 0
}

// ===== G9 异常观察计数器 =====

// bbChannelConsumeIncidentsMonth G9：汇总 SafeguardIncidents 全部非 "obs"
// 前缀键的计数为「1 号可见异常数」，记审计日志行后按登记簿月度消费语义
// (ADR-0019) 清零已消费键。
//
// 清零范围只限四类旧累加键 (ignore / cover_collapse / deviation /
// it_overreach)；reports_received / census_anomaly / audit_fact /
// equipment_use 四个新键保留给 W5 读取一个周期，由写入方/消费方自行管理
// 生命周期 (键契约见文件头常量块)。"obs" 前缀键为观测管线簿记预留，
// 不计入异常数。确定性：键升序汇总，零 RNG。
func (s *GameService) bbChannelConsumeIncidentsMonth(silo *model.Silo, logs *[]string) {
	if len(silo.SafeguardIncidents) == 0 {
		return
	}
	keys := make([]string, 0, len(silo.SafeguardIncidents))
	for kind := range silo.SafeguardIncidents {
		keys = append(keys, kind)
	}
	sort.Strings(keys)
	total := 0
	for _, kind := range keys {
		if strings.HasPrefix(kind, "obs") {
			continue
		}
		total += silo.SafeguardIncidents[kind]
	}
	if total > 0 && logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"[Silo1Stewardship] silo %d visible anomalies: %d", silo.ID, total))
	}
	for _, kind := range []string{
		engine.SafeguardIncidentIgnore,
		engine.SafeguardIncidentCoverCollapse,
		engine.SafeguardIncidentDeviation,
		engine.SafeguardIncidentITOverreach,
	} {
		delete(silo.SafeguardIncidents, kind)
	}
}

// bbChannelRegisterIncident 服务侧登记 (同引擎侧语义：惰性初始化，n ≤ 0 空转)。
func bbChannelRegisterIncident(silo *model.Silo, kind string, n int) {
	if silo == nil || n <= 0 {
		return
	}
	if silo.SafeguardIncidents == nil {
		silo.SafeguardIncidents = map[string]int{}
	}
	silo.SafeguardIncidents[kind] += n
}

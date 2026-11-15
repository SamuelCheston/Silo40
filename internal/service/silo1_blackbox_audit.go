package service

// 1 号黑箱审计导出 (22 黑箱学习系列 00 卷裁决 12：1 号信念决策日志是审计渠道，
// 不进玩家界面；00 卷裁决 7：1 号内部量不进 UI)。结构先例：GetLlmDecisions
// (服务层导出 + 调试 HTTP 载体)。本文件只提供审计 JSON 的组装；HTTP 端点见
// internal/api/debug_http.go 的 /debug/silo1/blackbox (可选挂载，未挂时 503)。
//
// 纪律：wails 不绑定本方法、前端不呈现 (04 卷 §3.d 禁止项)；玩家经 Order 渠道
// 故事事件感知行为后果，不经本导出。

import (
	"encoding/json"
	"fmt"
	"sort"
)

// silo1BlackBoxAuditSilo 单堡审计切片：黑箱决策日志 + 盲区经营日志 + 必要的
// 世界侧读数 (审计回放用)。
type silo1BlackBoxAuditSilo struct {
	SiloID           uint                     `json:"silo_id"`
	SiloName         string                   `json:"silo_name"`
	Theta            float64                  `json:"theta"` // 世界侧真值 (审计可读，玩家不可见)
	CurrentExecution float64                  `json:"current_execution"`
	LastAssignedRisk float64                  `json:"last_assigned_risk"`
	SafeguardRisk    float64                  `json:"safeguard_risk"`
	ThetaHat         float64                  `json:"theta_hat"` // 1 号最新后验均值 (信念侧)
	PDanger          float64                  `json:"p_danger"`
	Lo               float64                  `json:"lo"`
	Hi               float64                  `json:"hi"`
	PendingObs       int                      `json:"pending_observations"`
	ProbeCooldown    int                      `json:"probe_cooldown,omitempty"`
	ObediencePulse   int                      `json:"obedience_pulse,omitempty"`
	Log              []blackBoxAuditLogEntry  `json:"log"`
	BlindSpot        *silo1BlindSpotAuditPart `json:"blind_spot,omitempty"`
}

// blackBoxAuditLogEntry 决策日志条目 (原样导出，字段对齐 model.BlackBoxLogEntry)。
type blackBoxAuditLogEntry struct {
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	ThetaHat float64 `json:"theta_hat"`
	Lo       float64 `json:"lo"`
	Hi       float64 `json:"hi"`
	PDanger  float64 `json:"p_danger"`
	ChosenR  float64 `json:"chosen_r"`
	Utility  float64 `json:"utility"`
	Reason   string  `json:"reason,omitempty"`
	Factors  string  `json:"factors"`
}

// silo1BlindSpotAuditPart 盲区经营审计切片：阶段读数 + 决策日志。
type silo1BlindSpotAuditPart struct {
	Stage       string                        `json:"stage,omitempty"`
	StageMonths int                           `json:"stage_months,omitempty"`
	Suspicion   float64                       `json:"suspicion"`
	Log         []blackBoxAuditBlindSpotEntry `json:"log"`
}

// blackBoxAuditBlindSpotEntry 盲区决策日志条目 (字段对齐
// model.BlindSpotDecisionLogEntry)。
type blackBoxAuditBlindSpotEntry struct {
	Year      int    `json:"year"`
	Month     int    `json:"month"`
	Acted     bool   `json:"acted"`
	Choice    string `json:"choice,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Intensity int    `json:"intensity,omitempty"`
	Weights   string `json:"weights"`
	Factors   string `json:"factors"`
}

// silo1BlackBoxAuditPayload 审计导出顶层。
type silo1BlackBoxAuditPayload struct {
	Enabled   bool                     `json:"enabled"`
	Year      int                      `json:"year"`
	Month     int                      `json:"month"`
	Silos     []silo1BlackBoxAuditSilo `json:"silos"`
	MetricOne silo1AuditMetricOne      `json:"metric_one"`
	MetricTwo silo1AuditMetricTwo      `json:"metric_two"`
}

// silo1AuditMetricOne 指标一读数 (实验履约与保留多样性)。
type silo1AuditMetricOne struct {
	Initialized        bool    `json:"initialized"`
	FulfillmentDeficit float64 `json:"fulfillment_deficit"`
	DiversityLoss      float64 `json:"diversity_loss"`
	RetentionDiversity float64 `json:"retention_diversity"`
	Scheduled          []uint  `json:"scheduled,omitempty"`
	Cleared            []uint  `json:"cleared,omitempty"`
	Missed             []uint  `json:"missed,omitempty"`
}

// silo1AuditMetricTwo 指标二读数 (剩余人员平均寿命)。
type silo1AuditMetricTwo struct {
	Initialized bool            `json:"initialized"`
	AvgLifespan float64         `json:"avg_lifespan"`
	Exhausted   bool            `json:"exhausted"`
	Postures    map[uint]string `json:"postures,omitempty"`
}

// GetSilo1BlackBoxAudit 导出各堡黑箱日志 + 盲区日志 + 指标一/二读数的 JSON
// (审计渠道，00 卷裁决 12)。功能未开启或无会话时返回错误。
func (s *GameService) GetSilo1BlackBoxAudit() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return "", fmt.Errorf("no active game session")
	}
	if !s.silo1BlackBoxOn() {
		return "", fmt.Errorf("silo1 black box is disabled (create the game with silo1_black_box=true)")
	}

	payload := silo1BlackBoxAuditPayload{
		Enabled: true,
		Year:    s.world.CurrentYear,
		Month:   s.world.CurrentMonth,
		Silos:   []silo1BlackBoxAuditSilo{},
	}
	for _, silo := range s.silos {
		if silo == nil {
			continue
		}
		entry := silo1BlackBoxAuditSilo{
			SiloID:           silo.ID,
			SiloName:         silo.Name,
			Theta:            silo.BlackBox.Theta,
			CurrentExecution: silo.BlackBox.CurrentExecution,
			LastAssignedRisk: silo.BlackBox.LastAssignedRisk,
			SafeguardRisk:    silo.SafeguardRisk,
			PendingObs:       len(silo.BlackBox.PendingObservations),
			ProbeCooldown:    silo.BlackBox.ProbeCooldown,
			ObediencePulse:   silo.BlackBox.ObediencePulse,
			Log:              []blackBoxAuditLogEntry{},
		}
		if len(silo.BlackBox.Belief) > 0 {
			entry.ThetaHat, entry.PDanger, entry.Lo, entry.Hi = blackBoxSummarize(silo.BlackBox.Belief)
		}
		for _, e := range silo.BlackBox.Log {
			entry.Log = append(entry.Log, blackBoxAuditLogEntry{
				Year: e.Year, Month: e.Month, ThetaHat: e.ThetaHat, Lo: e.Lo, Hi: e.Hi,
				PDanger: e.PDanger, ChosenR: e.ChosenR, Utility: e.Utility,
				Reason: e.Reason, Factors: e.Factors,
			})
		}
		if silo.BlindSpot.Initialized {
			part := &silo1BlindSpotAuditPart{
				Stage:       silo.BlindSpot.Stage,
				StageMonths: silo.BlindSpot.StageMonths,
				Suspicion:   silo.BlindSpot.Suspicion,
				Log:         []blackBoxAuditBlindSpotEntry{},
			}
			for _, e := range silo.BlindSpot.Log {
				part.Log = append(part.Log, blackBoxAuditBlindSpotEntry{
					Year: e.Year, Month: e.Month, Acted: e.Acted, Choice: e.Choice,
					Channel: e.Channel, Intensity: e.Intensity, Weights: e.Weights, Factors: e.Factors,
				})
			}
			entry.BlindSpot = part
		}
		payload.Silos = append(payload.Silos, entry)
	}

	st := &s.world.Silo1Stewardship
	payload.MetricOne = silo1AuditMetricOne{
		Initialized:        st.Initialized,
		FulfillmentDeficit: st.Experiment.FulfillmentDeficit,
		DiversityLoss:      st.Experiment.DiversityLoss,
		RetentionDiversity: s.stewardshipRetentionDiversity(),
		Scheduled:          sortedUintKeys(st.Experiment.Scheduled),
		Cleared:            sortedUintKeys(st.Experiment.Cleared),
	}
	for id, missed := range st.Experiment.Missed {
		if missed {
			payload.MetricOne.Missed = append(payload.MetricOne.Missed, id)
		}
	}
	payload.MetricOne.Missed = sortedUintSlice(payload.MetricOne.Missed)
	payload.MetricTwo = silo1AuditMetricTwo{
		Initialized: st.Lifespan.Initialized,
		AvgLifespan: st.Lifespan.AvgLifespan,
		Exhausted:   s.stewardshipLifespanExhausted(),
		Postures:    st.Lifespan.Postures,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// sortedUintKeys map 键升序切片 (确定性输出：map 遍历序随机)。
func sortedUintKeys(m map[uint]int) []uint {
	ids := make([]uint, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// sortedUintSlice uint 切片升序 (确定性输出)。
func sortedUintSlice(ids []uint) []uint {
	out := append([]uint(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

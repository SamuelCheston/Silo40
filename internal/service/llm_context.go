package service

// LLM 上下文档位的服务层装配：档位与独立开关的数据源、切换 API。
//
// 路径与参与度档位完全一致 (不另造配置通道)：
//   - 环境变量 → llm.LoadEnv → setupLlmAdvisor → Advisor；
//   - 中途切换 → SetLlmContext → Advisor.SetContext (记 kind=context_config
//     日志，Resume 恢复)；
//   - 富上下文段落由 llmContextData 在 prompt 组装时按需提供——它只在游戏
//     线程持有 s.mu 时被调用 (Advise/Committed/queuePrefetch 全在锁内)，
//     因此只做纯状态读取：不拿锁、不消耗 RNG、不写状态。

import (
	"fmt"
	"strings"

	"silo40/internal/llm"
)

// LlmContextConfig 用户可调的上下文配置 (前后端通信)：档位 + 独立开关。
type LlmContextConfig struct {
	Level             string `json:"level"`
	PersonalMemory    bool   `json:"personal_memory"`
	ProtocolClauses   bool   `json:"protocol_clauses"`
	PeerSiloIntel     bool   `json:"peer_silo_intel"`
	EventSummaryLimit int    `json:"event_summary_limit"`
}

// SetLlmContext 中途切换上下文档位与独立开关：切换记入决策日志，下一决策
// 点即按新配置组装 prompt。参与度档位不受影响 (两者正交)。
func (s *GameService) SetLlmContext(cfg LlmContextConfig) (LlmContextConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.llmAdvisor == nil {
		return LlmContextConfig{}, fmt.Errorf("LLM is disabled (set SILO40_LLM_MODE=shadow|on)")
	}
	level, err := llm.ParseContextLevel(cfg.Level)
	if err != nil {
		return LlmContextConfig{}, err
	}
	limit := cfg.EventSummaryLimit
	if limit < 0 {
		limit = 0
	}
	opts := llm.ContextOptions{
		PersonalMemory:    cfg.PersonalMemory,
		ProtocolClauses:   cfg.ProtocolClauses,
		EventSummaryLimit: limit,
		PeerSiloIntel:     cfg.PeerSiloIntel,
	}
	if err := s.llmAdvisor.SetContext(level, opts, "api"); err != nil {
		return LlmContextConfig{}, err
	}
	return s.llmContextConfigLocked(), nil
}

// llmContextConfigLocked 读回当前生效的上下文配置 (调用方持有 s.mu)。
func (s *GameService) llmContextConfigLocked() LlmContextConfig {
	if s.llmAdvisor == nil {
		return LlmContextConfig{Level: string(llm.DefaultContextLevel)}
	}
	opts := s.llmAdvisor.GetContextOptions()
	return LlmContextConfig{
		Level:             string(s.llmAdvisor.GetContextLevel()),
		PersonalMemory:    opts.PersonalMemory,
		ProtocolClauses:   opts.ProtocolClauses,
		PeerSiloIntel:     opts.PeerSiloIntel,
		EventSummaryLimit: opts.EventSummaryLimit,
	}
}

// peerStatusCN 与前端 WorldPanel 的 statusText 同口径。
var peerStatusCN = map[string]string{
	"normal":    "运转中",
	"unrest":    "动荡",
	"silent":    "失联",
	"destroyed": "覆灭",
}

// llmContextData 为 prompt 组装提供富上下文段落。档位决定取用哪些段落，
// 开关决定附加段落；空段落不下发 (低档真省 token)。
func (s *GameService) llmContextData(siloID uint) llm.ContextData {
	var data llm.ContextData
	if s.world == nil {
		return data
	}
	silo := s.siloByID(siloID)
	if silo == nil {
		return data
	}

	// 近期事件摘要 (新→旧，条数上限由档外开关决定)。
	for i := len(s.eventLogs) - 1; i >= 0; i-- {
		l := &s.eventLogs[i]
		if l.SiloID != siloID {
			continue
		}
		data.RecentEvents = append(data.RecentEvents, llm.EventSummary{
			Year:        l.Year,
			Month:       l.Month,
			Category:    l.Category,
			Title:       l.Title,
			Description: l.Description,
		})
	}

	// 本堡主线经过：history 类事件即主线节点 (旧→新)。
	for i := range s.eventLogs {
		l := &s.eventLogs[i]
		if l.SiloID != siloID || l.Category != "history" {
			continue
		}
		data.MainlineEvents = append(data.MainlineEvents, llm.EventSummary{
			Year:        l.Year,
			Month:       l.Month,
			Category:    l.Category,
			Title:       l.Title,
			Description: l.Description,
		})
	}

	// 条文法规：真实 Order 条款 (加载器持有)。
	for _, c := range s.protocolClauses {
		desc := c.Description
		if desc == "" {
			switch {
			case c.ForbiddenAction != "":
				desc = "禁止 " + c.ForbiddenAction
			case c.RequiredAction != "":
				desc = "必须 " + c.RequiredAction
			case c.TargetMetric != "":
				desc = "目标 " + c.TargetMetric
			}
		}
		data.ProtocolClauses = append(data.ProtocolClauses, llm.ClauseSummary{
			ID:          c.ID,
			Category:    c.Category,
			Severity:    c.Severity,
			Description: desc,
		})
	}

	// silo 设定 (世界观档)：只陈述代码与知识文档已确立的事实。
	setting := fmt.Sprintf("当前地堡：%s（%d 人，%d 个部门）。",
		silo.Name, silo.TotalPopulation, len(silo.Professions))
	if len(silo.Traits) > 0 {
		setting += "地堡特质：" + strings.Join(silo.Traits, "、") + "。"
	}
	if s.world.ProgressStage != "" {
		setting += "当前阶段：" + s.world.ProgressStage + "。"
	}
	setting += "本地堡是地堡网络中的一站：网络按 Order 条款对各站施行报告与管制，" +
		"未经授权的跨堡接触被条款禁止；站内由市长、部门与派系构成。"
	data.WorldSetting = setting

	// 其他地堡关系 (世界观档)：拓扑相邻与否 + 谣言档状态。
	for _, peer := range s.silos {
		if peer.ID == siloID {
			continue
		}
		relation := "peer"
		for _, n := range s.world.Topology[siloID] {
			if n == peer.ID {
				relation = "neighbor"
				break
			}
		}
		data.PeerSilos = append(data.PeerSilos, llm.PeerSiloBrief{
			ID:       peer.ID,
			Name:     peer.Name,
			Relation: relation,
			Status:   rumorStatus(peer),
		})
	}

	// 长期剧情线 (世界观档)：各结局线已推进到的事实 (不预测、不预支)。
	for _, def := range lineDefs {
		p := s.buildLineProgress(def, silo)
		if p.Depth == 0 && !p.Closed {
			continue
		}
		line := fmt.Sprintf("%s：里程碑 %d/%d", p.Title, p.Depth, len(p.Milestones))
		if p.Regressed {
			line += "（已回摆）"
		}
		if p.Closed {
			line += "（已关闭）"
		}
		data.LongTermLines = append(data.LongTermLines, line)
	}
	if s.world.MainLine != "" {
		for _, def := range lineDefs {
			if def.Line == s.world.MainLine {
				data.LongTermLines = append([]string{"当前主线：" + def.Title}, data.LongTermLines...)
				break
			}
		}
	}

	// 其他地堡情报 (独立开关)：只给谣言档视图，不给精确值 (§3.4 的唯一
	// 用户自选例外)。
	for _, v := range s.buildRumorViews() {
		line := fmt.Sprintf("%s：%s（消息来源 %s）", v.Name, peerStatusCN[v.Status], v.SignalSource)
		if v.RebellionBand != "" {
			line += "，叛乱档 " + v.RebellionBand
		}
		if v.PopulationBand != "" {
			line += "，人口档 " + v.PopulationBand
		}
		data.PeerIntel = append(data.PeerIntel, line)
	}
	if s.world.Silo1.LastReport != "" {
		data.PeerIntel = append(data.PeerIntel, "1 号最近通报："+s.world.Silo1.LastReport)
	}

	return data
}

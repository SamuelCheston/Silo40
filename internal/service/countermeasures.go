package service

import (
	"silo40/internal/engine"
	"silo40/internal/model"
)

// runCountermeasureMonth 20 分册第 5 批反制的月度回合：T1-3 关系冷却、
// T2-4 谈判可信度侵蚀、T3-1b 身份线日常绑定、T3-2 联系人监视。
//
// 位置与筹备反制层相同：各堡本地结算后、时钟推进前 (单堡路径同)。
// 全部为确定性状态反应，不消耗 RNG；未越线的运行不写任何账本字段。
func (s *GameService) runCountermeasureMonth(silo *model.Silo) ([]string, []model.StoryEvent) {
	if silo == nil {
		return nil, nil
	}
	agent := s.agents[silo.ID]
	if agent == nil {
		agent = s.agent
	}
	engine.AdvanceCountermeasures(silo, agent)
	return nil, nil
}

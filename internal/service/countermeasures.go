package service

import (
	"fmt"

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
	signals := engine.AdvanceCountermeasures(silo, agent)
	if len(signals) == 0 {
		return nil, nil
	}

	logs := make([]string, 0, len(signals))
	stories := make([]model.StoryEvent, 0, len(signals))
	for _, signal := range signals {
		switch signal.Kind {
		case engine.CounterSignalGuarantee:
			logs = append(logs, fmt.Sprintf("[Countermeasure] %d-%02d T2-4 prep_exposure=%.2f 保障可信度被侵蚀",
				silo.CurrentYear, silo.CurrentMonth, silo.EscapePrep.PrepExposure))
			stories = append(stories, s.countermeasureStory(silo, "guarantee", "中间人的措辞",
				"中间人回来时话少了很多。他转述的条款听上去比你们递过去的更含糊。"))
		}
	}
	return logs, stories
}

// countermeasureStory 反制叙事：只描述发生的事，不指认玩家 (20 分册 §5.2)。
func (s *GameService) countermeasureStory(silo *model.Silo, key, title, description string) model.StoryEvent {
	return model.StoryEvent{
		ID:          fmt.Sprintf("countermeasure:%d:%d:%d:%s", silo.CurrentYear, silo.CurrentMonth, silo.ID, key),
		Category:    "SOCIAL",
		Title:       title,
		Description: description,
		Type:        "SOCIAL",
		SiloID:      silo.ID,
		SiloName:    silo.Name,
	}
}

package service

// 玩家对邻堡事件的感知 (事件墙治理 §每堡自己的主题)：玩家对邻堡的认知
// 本来就是间接的——谣言、信号、道听途说。邻堡的日常琐碎不进入玩家的情报
// 记录；只有灾难、制度变动与一次性转折会经谣言通道传出，并被改写为
// "听到的版本"。规则全部确定性，不消耗 RNG。

import (
	"silo40/internal/engine"
	"silo40/internal/model"
)

// rumorChannel 玩家听到邻堡消息的渠道，与迷雾视图 (buildRumorViews) 一致：
// IT 特工听的是线路里的杂音，其他人听的是谣言。
func (s *GameService) rumorChannel() string {
	if s.agent != nil && s.agent.Profession == "IT" {
		return "线路杂音"
	}
	return "谣言"
}

// neighborNewsworthy 邻堡事件是否值得传出去：危机与制度变动传得出；
// 一次性转折传得出；可重复的日常琐碎传不出。
func neighborNewsworthy(story model.StoryEvent) bool {
	switch story.Category {
	case "crisis", "protocol":
		return true
	}
	return story.FireMode == engine.ContentFireModeOnce
}

// perceiveNeighborStory 返回玩家能听到的邻堡事件版本。ok=false 表示该事件
// 留在邻堡内部，不进入玩家的情报记录。正典链条 (category history) 与
// 外部框架事件 (截获信号等) 本就是玩家的间接感知，原样放行。
func (s *GameService) perceiveNeighborStory(story model.StoryEvent, silo *model.Silo) (model.StoryEvent, bool) {
	if silo == nil || s.world == nil || silo.ID == s.world.PlayerSiloID {
		return story, true
	}
	if story.Category == "history" || story.Type == "EXTERNAL" {
		return story, true
	}
	if !neighborNewsworthy(story) {
		return story, false
	}
	rumor := story
	rumor.Title = "传言：" + story.Title
	rumor.Description = "经「" + s.rumorChannel() + "」听到：" + story.Description
	return rumor, true
}

// perceiveMonthStories 对世界 tick 收集到的整月故事应用邻堡感知过滤。
// 故事已带来源堡归属 (stampSilo)；无归属或归属玩家堡的原样保留。
func (s *GameService) perceiveMonthStories(stories []model.StoryEvent) []model.StoryEvent {
	if !s.multiSilo() {
		return stories
	}
	out := make([]model.StoryEvent, 0, len(stories))
	for _, story := range stories {
		origin := s.siloByID(story.SiloID)
		perceived, ok := s.perceiveNeighborStory(story, origin)
		if !ok {
			continue
		}
		out = append(out, perceived)
	}
	return out
}

// isContentFiringSource 判断 recordStories 的来源是否为内容事件点火
// (月度推进/动作结算/跨堡信号/延迟派发)。只有这类来源的邻堡故事需要走
// 谣言过滤；情报产品、行动回执、正典链条等本就是玩家视角的记录，直接放行。
func isContentFiringSource(source string) bool {
	switch source {
	case "CROSS_SIGNAL", "SCHEDULED":
		return true
	}
	return len(source) > 8 && (source[len(source)-8:] == ":CONTENT")
}

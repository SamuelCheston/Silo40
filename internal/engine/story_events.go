package engine

import (
	"math/rand"

	"silo40/internal/model"
)

// STORY_EVENT_TYPE 剧情随机事件类型
const STORY_EVENT_TYPE = "STORY_EVENT"

// StoryEvent 剧情随机事件 (含执行效果)
type StoryEvent struct {
	ID          string
	Category    string
	Title       string
	Description string
	Type        string // SOCIAL / TECHNICAL / EXTERNAL
	Effects     func(silo *model.Silo)
}

// EventEngine 剧情随机事件引擎
type EventEngine struct {
	events []StoryEvent
}

func NewEventEngine() *EventEngine {
	// 随机剧情事件已完全外挂到 events/ 目录中，
	// 由 Content Loader 动态加载。
	return &EventEngine{events: []StoryEvent{}}
}

// TriggerRandomEvent 触发随机剧情事件 (在 EventContext 内发布)
// @returns 被选中的剧情事件，未触发返回 nil
func (e *EventEngine) TriggerRandomEvent(silo *model.Silo, bus *EventBus, ctx *EventContext) *model.StoryEvent {
	if silo.EventTrigger < 1.0 {
		return nil
	}

	// 重置触发器
	silo.EventTrigger = 0

	idx := rand.Intn(len(e.events))
	story := e.events[idx]

	bus.Emit(CreateEvent("story_"+story.ID, STORY_EVENT_TYPE, map[string]interface{}{
		"silo":  silo,
		"story": &story,
	}), ctx)

	return &model.StoryEvent{
		ID:          story.ID,
		Category:    story.Category,
		Title:       story.Title,
		Description: story.Description,
		Type:        story.Type,
	}
}

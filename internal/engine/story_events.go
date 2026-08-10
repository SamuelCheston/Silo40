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
	return &EventEngine{events: []StoryEvent{
		{
			ID: "water_leak", Title: "供水管线泄漏",
			Description: "底层机械部报告发生严重水管泄漏，部分楼层供水中断。",
			Type:        "TECHNICAL",
			Effects: func(silo *model.Silo) {
				for i := range silo.Resources {
					if silo.Resources[i].Type == "Supplies" {
						silo.Resources[i].Amount -= 500
					}
				}
				silo.Cohesion -= 0.05
			},
		},
		{
			ID: "food_poisoning", Title: "群体性食物中毒",
			Description: "水培区的一批农产品受到污染，引发大规模恐慌。",
			Type:        "SOCIAL",
			Effects: func(silo *model.Silo) {
				for i := range silo.Professions {
					silo.Professions[i].PanicValue += 0.1
				}
				silo.Legitimacy -= 0.05
			},
		},
		{
			ID: "outside_signal", Title: "接收到外部信号",
			Description: "IT部门截获了一段模糊的无线电信号，似乎来自地表或其他地堡。",
			Type:        "EXTERNAL",
			Effects: func(silo *model.Silo) {
				for i := range silo.Professions {
					silo.Professions[i].Ideologies[model.IdeologyProForeign] += 0.05
				}
			},
		},
		{
			ID: "silo1_destroyed_signal", Title: "一号地堡失去联系",
			Description: "所有与一号地堡的通信协议均已超时，服务器不再响应。",
			Type:        "EXTERNAL",
			Effects: func(silo *model.Silo) {
				silo.Silo1Destroyed = true
			},
		},
	}}
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
		Title:       story.Title,
		Description: story.Description,
		Type:        story.Type,
	}
}

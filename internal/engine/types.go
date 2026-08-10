package engine

import "silo40/internal/model"

// ============ 统一游戏时间 ============
type GameTime struct {
	Year  int
	Month int // 1-12
}

// CompareTime 比较两个游戏时间：返回负/零/正
func CompareTime(a, b GameTime) int {
	if a.Year != b.Year {
		return a.Year - b.Year
	}
	return a.Month - b.Month
}

// AdvanceTime 时间推移 months 个月
func AdvanceTime(t GameTime, months int) GameTime {
	total := t.Year*12 + (t.Month - 1) + months
	year := total / 12
	month := (total % 12) + 1
	return GameTime{Year: year, Month: month}
}

// ============ 统一事件对象 ============
// GameEvent 游戏内一切行为统一抽象为事件
type GameEvent struct {
	ID          string
	Type        string
	Source      string
	Target      string
	TriggerTime *GameTime
	Data        map[string]interface{}
}

// CreateEvent 事件工厂
func CreateEvent(id, typ string, data map[string]interface{}) *GameEvent {
	if data == nil {
		data = map[string]interface{}{}
	}
	return &GameEvent{ID: id, Type: typ, Data: data}
}

// ============ 全局游戏状态视图 ============
// State 事件处理器读取/写入的全局状态
type State struct {
	Silo  *model.Silo
	Agent *model.Agent
	Logs  []string
}

// ============ 事件上下文 (避免事件风暴) ============
// EventContext fired: 本链内已触发的事件 id 集合，防止无限循环; depth: 链深度
type EventContext struct {
	fired    map[string]bool
	depth    int
	maxDepth int
}

func NewEventContext() *EventContext {
	return &EventContext{fired: map[string]bool{}, maxDepth: 20}
}

func (c *EventContext) CanFire(id string) bool {
	return c.depth < c.maxDepth && !c.fired[id]
}

func (c *EventContext) MarkFired(id string) {
	c.fired[id] = true
}

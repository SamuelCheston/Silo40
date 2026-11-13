package engine

// Condition 命名条件函数
type Condition func(state *State, event *GameEvent) bool

// ConditionEngine 条件引擎：集中注册、复用命名条件，供 RuleEngine / TriggerEngine 引用
type ConditionEngine struct {
	conditions map[string]Condition
}

func NewConditionEngine() *ConditionEngine {
	return &ConditionEngine{conditions: map[string]Condition{}}
}

func (c *ConditionEngine) Register(id string, fn Condition) {
	c.conditions[id] = fn
}

func (c *ConditionEngine) RegisterMany(defs map[string]Condition) {
	for id, fn := range defs {
		c.Register(id, fn)
	}
}

func (c *ConditionEngine) Check(id string, state *State, event *GameEvent) bool {
	fn, ok := c.conditions[id]
	if !ok {
		return false
	}
	return fn(state, event)
}

func (c *ConditionEngine) Has(id string) bool {
	_, ok := c.conditions[id]
	return ok
}

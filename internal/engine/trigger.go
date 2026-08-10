package engine

// Trigger 剧情触发器
type Trigger struct {
	ID          string
	Description string
	Condition   func(state *State) bool
	Effect      func(bus *EventBus, ctx *EventContext, state *State)
}

// TriggerEngine 剧情触发器：游戏持续检查 trigger 条件，满足即 fire
type TriggerEngine struct {
	triggers []Trigger
}

func NewTriggerEngine() *TriggerEngine {
	return &TriggerEngine{}
}

func (t *TriggerEngine) Register(trigger Trigger) {
	t.triggers = append(t.triggers, trigger)
}

func (t *TriggerEngine) RegisterMany(triggers []Trigger) {
	for _, tr := range triggers {
		t.Register(tr)
	}
}

// Evaluate 每次 Tick 后评估所有触发器
func (t *TriggerEngine) Evaluate(bus *EventBus, ctx *EventContext, state *State) {
	for _, tr := range t.triggers {
		if tr.Condition(state) {
			tr.Effect(bus, ctx, state)
		}
	}
}

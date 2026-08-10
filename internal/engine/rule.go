package engine

import "silo40/internal/model"

// RuleEffect 规则效果：数据驱动的描述，运行时由 RuleEngine 解释执行
type RuleEffect struct {
	Type   string // script / fire_event / schedule_event
	Script string
	Event  *GameEvent
	DelayMonths int
}

// GameRule 游戏规则: event + trigger + effects
type GameRule struct {
	ID      string
	Trigger struct {
		EventType string // 监听的源事件类型；缺省/ANY 表示响应所有事件
		Condition string // ConditionEngine 中的条件 ID
	}
	Effects []RuleEffect
}

// RuleEngine 规则引擎：订阅总线所有事件，逐条匹配规则并执行效果
type RuleEngine struct {
	bus    *EventBus
	rules  []GameRule
	cond   *ConditionEngine
	scripts *ScriptEngine
	// OnSchedule 延时效果回调：由 GameEngine 注入，将延时事件送入 Scheduler
	OnSchedule func(event *GameEvent, delayMonths int, source *GameEvent)
	// LastState 兜底状态引用：由 GameEngine 在每次 tick 前刷新
	LastState *State
}

func NewRuleEngine(cond *ConditionEngine, scripts *ScriptEngine, bus *EventBus) *RuleEngine {
	r := &RuleEngine{cond: cond, scripts: scripts, bus: bus}
	bus.SubscribeAny(func(event *GameEvent, ctx *EventContext) { r.OnEvent(event, ctx) })
	return r
}

func (r *RuleEngine) Register(rule GameRule) {
	r.rules = append(r.rules, rule)
}

func (r *RuleEngine) RegisterMany(rules []GameRule) {
	for _, rule := range rules {
		r.Register(rule)
	}
}

func (r *RuleEngine) OnEvent(event *GameEvent, ctx *EventContext) {
	state := r.ResolveState(event)
	if state.Silo == nil {
		return // 无状态上下文 (纯信号事件)，跳过规则
	}
	for _, rule := range r.rules {
		if rule.ID == event.ID {
			continue // 防止规则自我触发
		}
		t := rule.Trigger
		if t.EventType != "" && t.EventType != event.Type && t.EventType != ANY {
			continue
		}
		if t.Condition != "" && !r.cond.Check(t.Condition, state, event) {
			continue
		}
		for _, effect := range rule.Effects {
			r.applyEffect(effect, event, state, ctx)
		}
	}
}

func (r *RuleEngine) applyEffect(effect RuleEffect, source *GameEvent, state *State, ctx *EventContext) {
	switch effect.Type {
	case "script":
		if effect.Script != "" {
			r.scripts.Run(effect.Script, source, state, r.bus, ctx)
		}
	case "fire_event":
		if effect.Event == nil {
			return
		}
		child := r.buildChild(effect.Event, source)
		r.bus.Emit(child, ctx)
	case "schedule_event":
		if effect.Event == nil || effect.DelayMonths <= 0 || r.OnSchedule == nil {
			return
		}
		child := r.buildChild(effect.Event, source)
		r.OnSchedule(child, effect.DelayMonths, source)
	}
}

func (r *RuleEngine) buildChild(tmpl *GameEvent, source *GameEvent) *GameEvent {
	src := tmpl.Source
	if src == "" {
		src = source.Source
	}
	tgt := tmpl.Target
	if tgt == "" {
		tgt = source.Target
	}
	return &GameEvent{
		ID:     tmpl.ID,
		Type:   tmpl.Type,
		Source: src,
		Target: tgt,
		Data:   tmpl.Data,
	}
}

// ResolveState 从事件 data 中恢复 State 视图
func (r *RuleEngine) ResolveState(event *GameEvent) *State {
	st := &State{}
	if v, ok := event.Data["silo"].(*model.Silo); ok && v != nil {
		st.Silo = v
	} else if r.LastState != nil {
		st.Silo = r.LastState.Silo
	}
	if v, ok := event.Data["agent"].(*model.Agent); ok && v != nil {
		st.Agent = v
	} else if r.LastState != nil {
		st.Agent = r.LastState.Agent
	}
	if v, ok := event.Data["logs"].([]string); ok {
		st.Logs = v
	} else if r.LastState != nil {
		st.Logs = r.LastState.Logs
	}
	return st
}

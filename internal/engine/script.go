package engine

// Script 命名脚本：把 "效果" 注册为命名脚本，供 RuleEngine 的 effects 按名调用
type Script func(event *GameEvent, state *State, bus *EventBus, ctx *EventContext)

// ScriptEngine 脚本引擎
type ScriptEngine struct {
	scripts map[string]Script
}

func NewScriptEngine() *ScriptEngine {
	return &ScriptEngine{scripts: map[string]Script{}}
}

func (s *ScriptEngine) Register(name string, script Script) {
	s.scripts[name] = script
}

func (s *ScriptEngine) RegisterMany(defs map[string]Script) {
	for name, fn := range defs {
		s.Register(name, fn)
	}
}

func (s *ScriptEngine) Run(name string, event *GameEvent, state *State, bus *EventBus, ctx *EventContext) {
	if script, ok := s.scripts[name]; ok {
		script(event, state, bus, ctx)
	}
}

func (s *ScriptEngine) Has(name string) bool {
	_, ok := s.scripts[name]
	return ok
}

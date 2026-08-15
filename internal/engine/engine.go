package engine

import (
	"fmt"
	"math"
	"math/rand"

	"silo40/internal/model"
)

// ============ 统一事件类型 (一切行为都是事件) ============
const (
	EVENT_TIME_TICK       = "TIME_TICK"       // 时间推进基准
	EVENT_AGENT_UPDATE    = "AGENT_UPDATE"    // 特工状态更新
	EVENT_RESOURCE_UPDATE = "RESOURCE_UPDATE" // 资源结算
	EVENT_METRICS_UPDATE  = "METRICS_UPDATE"  // 地堡指标更新
	EVENT_IDEOLOGY_UPDATE = "IDEOLOGY_UPDATE" // 思潮演化
	EVENT_NPC_ACTIONS     = "NPC_ACTIONS"     // NPC 自主行为
	EVENT_VICTORY_CHECK   = "VICTORY_CHECK"   // 胜利判定
	EVENT_STORY_EVENT     = STORY_EVENT_TYPE  // 剧情随机事件
	EVENT_ACTOR_ACTION    = "ACTOR_ACTION"    // 任意 Actor (玩家/NPC) 动作
	EVENT_GAME_OVER       = "GAME_OVER"       // 游戏结束
)

// GameEngine 事件驱动版游戏引擎
// 所有系统 (特工/资源/思潮/指标/NPC/胜利/剧情) 全部注册为 EventBus 的订阅者，
// 由 TIME_TICK 编排触发；延时事件进入 Scheduler (MinHeap)；
// 游戏规则由 RuleEngine 数据驱动；剧情触发由 TriggerEngine 条件驱动；
// 防事件风暴由 EventContext (fired set + maxDepth) 保证。
type GameEngine struct {
	Bus             *EventBus
	Scheduler       *Scheduler
	ConditionEngine *ConditionEngine
	ScriptEngine    *ScriptEngine
	RuleEngine      *RuleEngine
	TriggerEngine   *TriggerEngine
	EventEngine     *EventEngine

	// 内部结果收集
	idCounter    int
	tickStories  []*model.StoryEvent
	actionResult *model.ActionResult
	logs         []string

	// 配置常量
	perCapitaConsumption map[string]float64
	resourceProducers    map[string][]string
	professionFactors    map[string]float64
	traitFactors         map[string]float64
}

func NewGameEngine() *GameEngine {
	e := &GameEngine{
		Bus:             NewEventBus(),
		Scheduler:       NewScheduler(),
		ConditionEngine: NewConditionEngine(),
		ScriptEngine:    NewScriptEngine(),
		TriggerEngine:   NewTriggerEngine(),
		EventEngine:     NewEventEngine(),
		perCapitaConsumption: map[string]float64{
			"Supplies":  1.0,
			"Energy":    1.0,
			"Materials": 1.0,
		},
		resourceProducers: map[string][]string{
			"Energy":    {"Mechanical"},
			"Materials": {"Mines"},
			"Supplies":  {"Supply", "Agricultural"},
		},
		professionFactors: map[string]float64{
			"Mayor":      0.5,
			"Judicial":   0.4,
			"IT":         0.3,
			"Police":     0.3,
			"Mechanical": 0.2,
			"Medical":    0.2,
		},
		traitFactors: map[string]float64{
			"地堡土著":   0.1,
			"一号地堡密使": 0.5,
			"煽动者":    0.2,
			"守旧派":    -0.1,
		},
	}
	e.RuleEngine = NewRuleEngine(e.ConditionEngine, e.ScriptEngine, e.Bus)

	// 规则引擎的延时效果 → 进入 Scheduler
	e.RuleEngine.OnSchedule = func(event *GameEvent, delayMonths int, source *GameEvent) {
		if silo, ok := source.Data["silo"].(*model.Silo); ok && silo != nil {
			now := GameTime{Year: silo.CurrentYear, Month: silo.CurrentMonth}
			e.Scheduler.Schedule(event, AdvanceTime(now, delayMonths))
		}
	}

	e.registerSystems()
	e.registerConditions()
	e.registerScripts()
	e.registerRules()
	e.registerTriggers()
	return e
}

func (e *GameEngine) nextID() string {
	e.idCounter++
	return itoa(e.idCounter)
}

func (e *GameEngine) logf(msg string) {
	e.logs = append(e.logs, msg)
}

// ============ 系统注册：一切通过事件总线 ============
func (e *GameEngine) registerSystems() {
	// --- 时间推进编排：依次触发各子系统 ---
	e.Bus.Subscribe(EVENT_TIME_TICK, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		agent, _ := event.Data["agent"].(*model.Agent)
		deltaYears := event.Data["deltaYears"].(float64)

		e.Bus.Emit(CreateEvent("agent_update#"+e.nextID(), EVENT_AGENT_UPDATE, map[string]interface{}{
			"silo": silo, "agent": agent, "deltaYears": deltaYears,
		}), ctx)

		e.Bus.Emit(CreateEvent("resource_update#"+e.nextID(), EVENT_RESOURCE_UPDATE, map[string]interface{}{
			"silo": silo, "deltaYears": deltaYears,
		}), ctx)

		e.Bus.Emit(CreateEvent("ideology_update#"+e.nextID(), EVENT_IDEOLOGY_UPDATE, map[string]interface{}{
			"silo": silo, "deltaYears": deltaYears,
		}), ctx)

		e.Bus.Emit(CreateEvent("npc_actions#"+e.nextID(), EVENT_NPC_ACTIONS, map[string]interface{}{
			"silo": silo, "agent": agent, "deltaYears": deltaYears,
		}), ctx)

		e.Bus.Emit(CreateEvent("metrics_update#"+e.nextID(), EVENT_METRICS_UPDATE, map[string]interface{}{
			"silo": silo, "deltaYears": deltaYears,
		}), ctx)

		e.Bus.Emit(CreateEvent("victory_check#"+e.nextID(), EVENT_VICTORY_CHECK, map[string]interface{}{
			"silo": silo, "agent": agent,
		}), ctx)
	})

	// --- 特工状态更新 ---
	e.Bus.Subscribe(EVENT_AGENT_UPDATE, func(event *GameEvent, ctx *EventContext) {
		silo, _ := event.Data["silo"].(*model.Silo)
		agent := event.Data["agent"].(*model.Agent)
		deltaYears := event.Data["deltaYears"].(float64)
		e.UpdateAgentState(agent, deltaYears, silo, e.logf)
	})

	// --- 资源结算 (含运作条件校验) ---
	e.Bus.Subscribe(EVENT_RESOURCE_UPDATE, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		deltaYears := event.Data["deltaYears"].(float64)
		e.updateResources(silo, deltaYears)
	})

	// --- 地堡指标更新 (倒计时/叛乱/人口) ---
	e.Bus.Subscribe(EVENT_METRICS_UPDATE, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		deltaYears := event.Data["deltaYears"].(float64)
		e.updateSiloMetrics(silo, deltaYears)
	})

	// --- 思潮演化 ---
	e.Bus.Subscribe(EVENT_IDEOLOGY_UPDATE, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		deltaYears := event.Data["deltaYears"].(float64)
		e.updateIdeology(silo, deltaYears)
	})

	// --- NPC 自主行为 (统一 Actor 管线) ---
	e.Bus.Subscribe(EVENT_NPC_ACTIONS, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		agent, _ := event.Data["agent"].(*model.Agent)
		deltaYears := event.Data["deltaYears"].(float64)
		e.RunNpcTurn(silo, agent, deltaYears, e.logf)
	})

	// --- 胜利判定 + 分数结算 ---
	e.Bus.Subscribe(EVENT_VICTORY_CHECK, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		agent, _ := event.Data["agent"].(*model.Agent)
		e.checkVictoryConditions(silo, agent)

		// 游戏结束 → 计算最终评分并发布 GAME_OVER
		if silo.VictoryStatus != nil && silo.VictoryStatus.Score == nil {
			silo.VictoryStatus.Score = e.CalculateScore(silo)
			e.Bus.Emit(CreateEvent("game_over#"+e.nextID(), EVENT_GAME_OVER, map[string]interface{}{
				"silo": silo,
			}), NewEventContext())
		}
	})

	// --- 任意 Actor 动作执行 (玩家/NPC 共用) ---
	e.Bus.Subscribe(EVENT_ACTOR_ACTION, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		actor := event.Data["actor"].(ActorRef)
		action := event.Data["action"].(model.AgentAction)
		agent, _ := event.Data["agent"].(*model.Agent)
		res := e.ExecuteActionInternal(silo, actor, action, agent)
		e.actionResult = &res
	})

	// --- 剧情随机事件效果应用 ---
	e.Bus.Subscribe(EVENT_STORY_EVENT, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		story := event.Data["story"].(*StoryEvent)
		story.Effects(silo)
	})
}

// ============ 规则引擎：条件 / 脚本 / 规则 ============
func (e *GameEngine) registerConditions() {
	e.ConditionEngine.RegisterMany(map[string]Condition{
		"supplies_low": func(state *State, event *GameEvent) bool {
			for _, r := range state.Silo.Resources {
				if r.Type == "Supplies" && r.Amount < 200 {
					return true
				}
			}
			return false
		},
		"panic_high": func(state *State, event *GameEvent) bool {
			for _, p := range state.Silo.Professions {
				if p.PanicValue > 0.7 {
					return true
				}
			}
			return false
		},
		"rebellion_high": func(state *State, event *GameEvent) bool {
			return state.Silo.Rebellion > 0.6
		},
	})
}

func (e *GameEngine) registerScripts() {
	e.ScriptEngine.RegisterMany(map[string]Script{
		// 恐慌蔓延 → 亲外度小幅上升
		"panic_to_ideology": func(event *GameEvent, state *State, bus *EventBus, ctx *EventContext) {
			for i := range state.Silo.Professions {
				p := &state.Silo.Professions[i]
				// 恐慌转化为思潮
				p.Ideologies[model.IdeologyProForeign] = math.Min(1.0, p.Ideologies[model.IdeologyProForeign]+0.02)
			}
			state.Logs = append(state.Logs, "[Rule] 高恐慌情绪转化为对外部世界的好奇。")
		},
		// 高叛乱 → 增加额外死亡
		"rebellion_deaths": func(event *GameEvent, state *State, bus *EventBus, ctx *EventContext) {
			extra := int(float64(state.Silo.TotalPopulation) * 0.005)
			state.Silo.TotalPopulation = int(math.Max(0, float64(state.Silo.TotalPopulation-extra)))
			state.Logs = append(state.Logs, "[Rule] 叛乱冲突造成约 "+itoa(extra)+" 人伤亡。")
		},
		// 物资短缺 → 凝聚力缓慢下降
		"supplies_shortage_cohesion": func(event *GameEvent, state *State, bus *EventBus, ctx *EventContext) {
			state.Silo.Cohesion = math.Max(0, state.Silo.Cohesion-0.02)
			state.Logs = append(state.Logs, "[Rule] 物资短缺削弱了地堡的凝聚力。")
		},
	})
}

func (e *GameEngine) registerRules() {
	rules := []GameRule{
		{
			ID: "supplies_shortage_rule",
			Trigger: struct {
				EventType string
				Condition string
			}{EventType: EVENT_METRICS_UPDATE, Condition: "supplies_low"},
			Effects: []RuleEffect{
				{Type: "script", Script: "supplies_shortage_cohesion"},
				{Type: "script", Script: "panic_to_ideology"},
			},
		},
		{
			ID: "rebellion_escalation_rule",
			Trigger: struct {
				EventType string
				Condition string
			}{EventType: EVENT_VICTORY_CHECK, Condition: "rebellion_high"},
			Effects: []RuleEffect{
				{Type: "script", Script: "rebellion_deaths"},
			},
		},
		{
			// 延时事件示例：恐慌过高时，3 个月后触发 "暴乱爆发" 延时事件
			ID: "panic_storm_delay_rule",
			Trigger: struct {
				EventType string
				Condition string
			}{EventType: EVENT_VICTORY_CHECK, Condition: "panic_high"},
			Effects: []RuleEffect{
				{
					Type: "schedule_event", DelayMonths: 3,
					Event: CreateEvent("delayed_riot", "DELAYED_RIOT", map[string]interface{}{}),
				},
			},
		},
	}
	e.RuleEngine.RegisterMany(rules)

	// 延时事件订阅：暴乱爆发 → 恐慌加剧
	e.Bus.Subscribe("DELAYED_RIOT", func(event *GameEvent, ctx *EventContext) {
		silo, _ := event.Data["silo"].(*model.Silo)
		if silo == nil {
			return
		}
		for i := range silo.Professions {
			silo.Professions[i].PanicValue = math.Min(1.0, silo.Professions[i].PanicValue+0.15)
		}
		silo.Legitimacy = math.Max(0, silo.Legitimacy-0.1)
	})
}

// ============ 剧情触发器 ============
func (e *GameEngine) registerTriggers() {
	e.TriggerEngine.RegisterMany([]Trigger{
		{
			ID:          "silo1_fallout",
			Description: "一号地堡失联后，凝聚力下降、恐慌上升 (剧情链条起点)",
			Condition: func(state *State) bool {
				return state.Silo.Silo1Destroyed && state.Silo.Cohesion > 0.3
			},
			Effect: func(bus *EventBus, ctx *EventContext, state *State) {
				state.Silo.Cohesion = math.Max(0, state.Silo.Cohesion-0.05)
				for i := range state.Silo.Professions {
					state.Silo.Professions[i].PanicValue = math.Min(1.0, state.Silo.Professions[i].PanicValue+0.05)
				}
				state.Logs = append(state.Logs, "[Trigger] 一号地堡失联的传闻在居民中蔓延。")
			},
		},
		{
			ID:          "pro_foreign_awakening",
			Description: "多数部门亲外度超过 50% 时，社会进入觉醒阶段",
			Condition: func(state *State) bool {
				count := 0
				for _, p := range state.Silo.Professions {
					if p.Ideologies[model.IdeologyProForeign] > 0.5 {
						count++
					}
				}
				return count >= 4
			},
			Effect: func(bus *EventBus, ctx *EventContext, state *State) {
				state.Silo.Legitimacy = math.Max(0, state.Silo.Legitimacy-0.05)
				state.Logs = append(state.Logs, "[Trigger] 社会思潮觉醒，旧秩序受到挑战。")
			},
		},
	})
}

// ============ 对外接口 ============

// SubmitAction 统一动作入口：玩家与 NPC 共用同一执行管线
func (e *GameEngine) SubmitAction(actor ActorRef, silo *model.Silo, action model.AgentAction, agent *model.Agent) model.ActionResult {
	e.actionResult = nil
	ctx := NewEventContext()
	e.RuleEngine.LastState = &State{Silo: silo, Agent: agent, Logs: e.logs}

	e.Bus.Emit(CreateEvent("action#"+e.nextID(), EVENT_ACTOR_ACTION, map[string]interface{}{
		"silo": silo, "actor": actor, "action": action, "agent": agent,
	}), ctx)

	if e.actionResult != nil {
		return *e.actionResult
	}
	return model.ActionResult{Executed: false, Message: "Unknown error."}
}

// ExecuteAgentAction 玩家动作入口
func (e *GameEngine) ExecuteAgentAction(silo *model.Silo, agent *model.Agent, action model.AgentAction) model.ActionResult {
	actor := CreateActorRefForAgent(agent, silo)
	return e.SubmitAction(actor, silo, action, agent)
}

// UpdateSiloState 推进一个游戏时间片：发布 TIME_TICK 并结算延时/剧情/随机事件
// 返回收集的日志与触发的剧情事件
func (e *GameEngine) UpdateSiloState(silo *model.Silo, deltaYears float64, agent *model.Agent) ([]string, []*model.StoryEvent) {
	e.tickStories = nil
	e.logs = []string{}
	ctx := NewEventContext()

	// 刷新规则引擎状态视图 (规则/脚本将日志写入 state.logs)
	state := &State{Silo: silo, Agent: agent, Logs: e.logs}
	e.RuleEngine.LastState = state

	// 1. 时间推进 (各系统订阅响应)
	e.Bus.Emit(CreateEvent("tick#"+e.nextID(), EVENT_TIME_TICK, map[string]interface{}{
		"silo": silo, "agent": agent, "deltaYears": deltaYears,
	}), ctx)

	// 2. 调度器：触发到期延时事件
	e.Scheduler.Tick(GameTime{Year: silo.CurrentYear, Month: silo.CurrentMonth}, e.Bus)

	// 3. 剧情触发器：条件检查
	e.TriggerEngine.Evaluate(e.Bus, ctx, state)

	// 4. 随机剧情事件
	story := e.EventEngine.TriggerRandomEvent(silo, e.Bus, ctx)
	if story != nil {
		e.tickStories = append(e.tickStories, story)
	}

	// 脚本/触发器可能对 state.Logs 触发 append 扩容，返回最新引用
	return e.RuleEngine.LastState.Logs, e.tickStories
}

// ScheduleEvent 注册延时事件到调度器
func (e *GameEngine) ScheduleEvent(event *GameEvent, at GameTime) {
	e.Scheduler.Schedule(event, at)
}

// UpdateAgentState 特工状态更新：包装为统一 Actor 状态结算
func (e *GameEngine) UpdateAgentState(agent *model.Agent, deltaYears float64, silo *model.Silo, addLog func(string)) {
	if silo == nil {
		return
	}
	view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		return
	}
	e.UpdateActorState(view, silo, deltaYears, addLog)
}

// UpdateActorState 统一 Actor 状态结算：被动特质 / 威望 / AP 恢复 / 怀疑度衰减
func (e *GameEngine) UpdateActorState(view *ActorView, silo *model.Silo, deltaYears float64, addLog func(string)) {
	if deltaYears <= 0 {
		return
	}

	// Medical 被动特质：随时间随机获得信息碎片 (对任意 Actor 生效)
	if view.Profession() == "Medical" {
		if rand.Float64() < 0.2 { // 20% chance per year
			known := map[string]bool{}
			for _, f := range view.KnownFragments() {
				known[f] = true
			}
			var available []string
			for _, f := range model.ALL_FRAGMENTS {
				if !known[f] {
					available = append(available, f)
				}
			}
			if len(available) > 0 {
				randomFragment := available[rand.Intn(len(available))]
				at := view.agentOrProf()
				if at.agent != nil {
					at.agent.KnownFragments = append(at.agent.KnownFragments, randomFragment)
				} else if at.resident != nil {
					at.resident.KnownFragments = append(at.resident.KnownFragments, randomFragment)
				} else if at.prof != nil {
					at.prof.KnownFragments = append(at.prof.KnownFragments, randomFragment)
				}
				if addLog != nil {
					if view.IsPlayer() {
						addLog("[Medical Passive] Your medical duties allowed you to overhear rumors, gaining information about " + randomFragment + ".")
					} else {
						addLog("[Medical Passive] " + view.Label() + " overheard rumors and gained intel on " + randomFragment + ".")
					}
				}
			}
		}
	}

	// 1. 计算平均人脉值 (0.0 - 1.0)
	connValues := view.ConnectionValues()
	totalConnection := 0.0
	if len(connValues) > 0 {
		for _, v := range connValues {
			totalConnection += v
		}
		totalConnection /= float64(len(connValues))
	}

	// 2. 计算职业修正系数
	profFactor := e.professionFactors[view.Profession()]

	// 3. 计算特质修正系数
	traitFactor := 0.0
	for _, trait := range view.Traits() {
		traitFactor += e.traitFactors[trait]
	}

	// 4. 计算政治威望：整合人脉、职业背景、权力等级和阶层偏见
	// 获取职业对象以提取权力等级和阶层信息
	var powerLevel int
	var classType string
	for _, p := range silo.Professions {
		if p.Name == view.Profession() {
			powerLevel = p.PowerLevel
			classType = p.ClassType
			break
		}
	}

	// 基础威望来源于人脉积累
	basePrestige := totalConnection * 100
	// 职业背景加成
	careerBonus := (1 + profFactor) * (1 + traitFactor)
	// 权力等级和阶层偏见对威望的直接贡献 (取代独立的影响力参数)
	structuralPrestige := float64(powerLevel)*1.5 + classBias(classType, 2.0)

	prestige := basePrestige*careerBonus + structuralPrestige
	if prestige < 0 {
		prestige = 0
	}
	view.SetPoliticalPrestige(prestige)

	// 5. 给予政治点数 (仅玩家特工) 和行动点数 (AP)
	const pointGainRate = 0.1
	if view.IsPlayer() {
		view.SetPoliticalPoints(view.PoliticalPoints() + prestige*pointGainRate*deltaYears)
	}

	// 行动点数恢复：基础恢复 10 点/年，受威望加成
	apGainRate := 10 + (prestige * 0.05)
	view.SetActionPoints(view.ActionPoints() + apGainRate*deltaYears)
	// 设置 AP 上限
	maxAP := 100.0
	if view.ActionPoints() > maxAP {
		view.SetActionPoints(maxAP)
	}

	// 6. 怀疑度随时间衰减
	const suspicionDecayRate = 0.05 // 每年降低5%
	if view.SuspicionLevel() > 0 {
		s := view.SuspicionLevel() - suspicionDecayRate*deltaYears
		if s < 0 {
			s = 0
		}
		view.SetSuspicionLevel(s)
	}
}

// ExecuteActionInternal Actor 执行动作内部实现 (玩家/NPC 共用)
func (e *GameEngine) ExecuteActionInternal(silo *model.Silo, actor ActorRef, action model.AgentAction, agent *model.Agent) model.ActionResult {
	view, err := CreateActorView(actor, silo, agent)
	if err != nil {
		return model.ActionResult{Executed: false, Message: err.Error()}
	}
	if view.ActionPoints() < action.Cost {
		return model.ActionResult{Executed: false, Message: "Not enough Action Points (AP)."}
	}

	preSuspicion := view.SuspicionLevel()
	var result model.ActionResult

	switch action.Type {
	case model.ActionGatherInfo:
		result = e.gatherInformation(silo, view, action)
	case model.ActionShareInfo:
		result = e.shareInformation(silo, view, action)
	case model.ActionBuildConnection:
		result = e.buildConnection(silo, view, action)
	case model.ActionInciteRebellion:
		result = e.inciteRebellion(silo, view, action)
	case model.ActionConductPropaganda:
		result = e.conductPropaganda(silo, view, action)
	case model.ActionPublicizeFaction:
		result = e.publicizeFaction(silo, view, action)
	case model.ActionProfession:
		result = e.executeProfessionAction(silo, view, action)
	}

	if result.Executed {
		gained := view.SuspicionLevel() - preSuspicion

		// 基础行为怀疑度惩罚 (兜底产生)
		switch action.Type {
		case model.ActionInciteRebellion:
			gained += 0.05
		case model.ActionShareInfo:
			gained += 0.01
		case model.ActionBuildConnection:
			gained += 0.01
		case model.ActionGatherInfo:
			gained += 0.005
		case model.ActionConductPropaganda:
			gained += 0.02
		case model.ActionProfession:
			def := GetProfessionAction(action.ProfessionAction)
			if def != nil {
				gained += def.SuspicionPenalty
			} else {
				gained += 0.02
			}
		}

		// 职业修正
		switch view.Profession() {
		case "Mayor":
			gained *= 2.0
		case "IT":
			gained *= 0.3
		case "Police":
			discount := 0.5 + rand.Float64()*0.4
			gained *= discount
		case "Mines":
			gained *= 0.3
		}

		// 特质修正
		for _, t := range view.Traits() {
			if t == "隐秘行事" {
				gained *= 0.8
				break
			}
		}

		view.SetSuspicionLevel(preSuspicion + gained)

		// IT 专属机制：恶化 safeguard 风险系数
		if view.Profession() == "IT" {
			silo.SafeguardRisk += action.Cost * 0.002
		}
	}

	return result
}

// BuildConnection 建立或强化与目标部门的人脉
func (e *GameEngine) buildConnection(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	if action.TargetDept == "" {
		return model.ActionResult{Executed: false, Message: "Invalid target department."}
	}
	targetProf := findDept(silo, action.TargetDept)
	if targetProf == nil {
		return model.ActionResult{Executed: false, Message: "Target department not found."}
	}

	increaseValue := 0.05 + (view.PoliticalPrestige() * 0.005)
	for _, t := range view.Traits() {
		if t == "魅力非凡" {
			increaseValue *= 1.5
			break
		}
	}
	view.AddConnection(targetProf.ID, increaseValue)

	view.SetActionPoints(view.ActionPoints() - action.Cost)
	return model.ActionResult{Executed: true, Message: "Successfully built connections with " + targetProf.Name + "."}
}

// InciteRebellion 煽动底层叛乱
func (e *GameEngine) inciteRebellion(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	commoners := []*model.Profession{}
	for i := range silo.Professions {
		if silo.Professions[i].ClassType == "COMMONER" {
			commoners = append(commoners, &silo.Professions[i])
		}
	}
	if len(commoners) == 0 {
		return model.ActionResult{Executed: false, Message: "No commoner departments found to incite."}
	}

	for _, prof := range commoners {
		connectionValue := view.GetConnection(prof.ID)

		baseEffect := 0.05 + (view.PoliticalPrestige() * 0.002)
		propagandaMultiplier := 1 + (view.PropagandaLevel() * 0.2)
		multiplier := (1 + connectionValue) * propagandaMultiplier
		finalEffect := baseEffect * multiplier

		prof.PanicValue += finalEffect
		prof.Ideologies[model.IdeologyProForeign] += finalEffect * 0.5
	}

	view.SetActionPoints(view.ActionPoints() - action.Cost)
	return model.ActionResult{Executed: true, Message: "Incited unrest among all commoner departments."}
}

// ConductPropaganda 主动进行宣传
func (e *GameEngine) conductPropaganda(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	view.SetPropagandaLevel(view.PropagandaLevel() + 1.0)
	view.SetActionPoints(view.ActionPoints() - action.Cost)
	return model.ActionResult{Executed: true, Message: "Conducted propaganda. Propaganda Level increased by 1.0."}
}

// PublicizeFaction 昭告天下：公开阵营存在并增加威望
func (e *GameEngine) publicizeFaction(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	resident := view.resident
	if resident == nil {
		// 如果是特工，尝试找到对应的居民
		for i := range silo.Residents {
			if silo.Residents[i].Alive && silo.Residents[i].Profession == view.agent.Profession && silo.Residents[i].IsRepresentative {
				resident = &silo.Residents[i]
				break
			}
		}
	}

	if resident == nil || !resident.IsRepresentative || resident.FactionID == nil {
		return model.ActionResult{Executed: false, Message: "Only faction representatives can publicize their faction."}
	}

	var targetFaction *model.Faction
	for i := range silo.Factions {
		if silo.Factions[i].ID == *resident.FactionID {
			targetFaction = &silo.Factions[i]
			break
		}
	}

	if targetFaction == nil {
		return model.ActionResult{Executed: false, Message: "Faction not found."}
	}

	if targetFaction.IsPublic {
		return model.ActionResult{Executed: false, Message: "Faction is already public."}
	}

	targetFaction.IsPublic = true
	targetFaction.Prestige += 15.0 // 昭告天下增加威望

	view.SetActionPoints(view.ActionPoints() - action.Cost)
	return model.ActionResult{Executed: true, Message: "Successfully publicized " + targetFaction.Name + ". The silo is now aware of your presence."}
}

// ExecuteProfessionAction 职业专属行动
func (e *GameEngine) executeProfessionAction(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	def := GetProfessionAction(action.ProfessionAction)
	if def == nil {
		return model.ActionResult{Executed: false, Message: "Unknown profession action."}
	}
	if def.Profession != view.Profession() {
		return model.ActionResult{Executed: false, Message: def.Label + " is not available to " + view.Profession() + "."}
	}
	if def.TargetType == TargetDept && action.TargetDept == "" {
		return model.ActionResult{Executed: false, Message: "Please select a target department."}
	}
	if def.TargetType == TargetResource && action.ResourceTarget == "" {
		return model.ActionResult{Executed: false, Message: "Please select a target resource."}
	}

	target := ""
	switch def.TargetType {
	case TargetDept:
		target = action.TargetDept
	case TargetResource:
		target = action.ResourceTarget
	}

	result := def.Effect(silo, view, target)
	if result.Executed {
		view.SetActionPoints(view.ActionPoints() - def.APCost)
	}
	return result
}

// GatherInformation 搜集其他部门的信息碎片
func (e *GameEngine) gatherInformation(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	if action.TargetDept == "" {
		return model.ActionResult{Executed: false, Message: "Invalid target department."}
	}

	known := map[string]bool{}
	for _, f := range view.KnownFragments() {
		known[f] = true
	}
	prefix := action.TargetDept + "_"
	var unknown []string
	for _, f := range model.ALL_FRAGMENTS {
		if len(f) > len(prefix) && f[:len(prefix)] == prefix && !known[f] {
			unknown = append(unknown, f)
		}
	}

	if len(unknown) > 0 {
		fragmentToGather := unknown[rand.Intn(len(unknown))]
		at := view.agentOrProf()
		if at.agent != nil {
			at.agent.KnownFragments = append(at.agent.KnownFragments, fragmentToGather)
		} else if at.resident != nil {
			at.resident.KnownFragments = append(at.resident.KnownFragments, fragmentToGather)
		} else if at.prof != nil {
			at.prof.KnownFragments = append(at.prof.KnownFragments, fragmentToGather)
		}
		view.SetActionPoints(view.ActionPoints() - action.Cost)
		return model.ActionResult{Executed: true, Message: "Gathered intel on " + fragmentToGather + "."}
	}

	return model.ActionResult{Executed: false, Message: "Your department already knows everything about " + action.TargetDept + "."}
}

// ShareInformation 将自己掌握的信息碎片分享给目标部门
func (e *GameEngine) shareInformation(silo *model.Silo, view *ActorView, action model.AgentAction) model.ActionResult {
	if action.TargetDept == "" || len(action.FragmentIds) == 0 {
		return model.ActionResult{Executed: false, Message: "Invalid target or no fragments selected."}
	}

	targetProf := findDept(silo, action.TargetDept)
	if targetProf == nil {
		return model.ActionResult{Executed: false, Message: "Target department not found."}
	}

	connectionValue := view.GetConnection(targetProf.ID)

	// AP 即使被拒绝也会消耗
	view.SetActionPoints(view.ActionPoints() - action.Cost)

	known := map[string]bool{}
	for _, f := range view.KnownFragments() {
		known[f] = true
	}
	unexplainedCount := 0
	for _, id := range action.FragmentIds {
		if !known[id] {
			unexplainedCount++
		}
	}

	if unexplainedCount > 0 {
		suspicionPenalty := (float64(unexplainedCount) * 0.1) + (math.Pow(float64(unexplainedCount), 1.5) * 0.05)
		view.SetSuspicionLevel(view.SuspicionLevel() + suspicionPenalty)
	}

	acceptanceRate := 0.1 + targetProf.Ideologies[model.IdeologyProForeign] + connectionValue
	acceptanceRate -= float64(unexplainedCount) * 0.1
	if acceptanceRate < 0.05 {
		acceptanceRate = 0.05
	}
	if acceptanceRate > 1.0 {
		acceptanceRate = 1.0
	}

	roll := rand.Float64()
	if roll > acceptanceRate {
		return model.ActionResult{
			Executed: true,
			Message:  "Attempted to share info with " + targetProf.Name + ", but they rejected it! (Acceptance rate was " + formatPct(acceptanceRate) + "%)",
		}
	}

	if targetProf.KnownFragments == nil {
		targetProf.KnownFragments = []string{}
	}
	knownTarget := map[string]bool{}
	for _, f := range targetProf.KnownFragments {
		knownTarget[f] = true
	}
	for _, f := range action.FragmentIds {
		if !knownTarget[f] {
			targetProf.KnownFragments = append(targetProf.KnownFragments, f)
			knownTarget[f] = true
		}
	}

	panicImpact := 0.05 + float64(unexplainedCount)*0.05
	targetProf.PanicValue = math.Min(1.0, targetProf.PanicValue+panicImpact)

	if connectionValue >= 0.3 {
		ideologyImpact := 0.02 + float64(unexplainedCount)*0.02
		targetProf.Ideologies[model.IdeologyProForeign] = math.Min(1.0, targetProf.Ideologies[model.IdeologyProForeign]+ideologyImpact)
	}

	return model.ActionResult{
		Executed: true,
		Message:  "Successfully shared " + itoa(len(action.FragmentIds)) + " fragments with " + targetProf.Name + ". (Included " + itoa(unexplainedCount) + " pieces of unexplained knowledge)",
	}
}

// RunNpcTurn NPC 回合 (统一 Actor 管线)：经济结算 + 决策层提交动作
func (e *GameEngine) RunNpcTurn(silo *model.Silo, agent *model.Agent, deltaYears float64, addLog func(string)) {
	if silo == nil {
		return
	}

	// 1. Keep aggregated population / key resident / faction state in sync,
	// but do not drive political decisions from factions yet. Profession
	// decisions remain the active NPC production-side behavior.
	updatePopulationCohorts(silo, deltaYears)
	updateKeyResidents(silo, deltaYears)
	RebuildImplicitFactions(silo)

	var npcProfs []*model.Profession
	for i := range silo.Professions {
		p := &silo.Professions[i]
		if agent == nil || p.Name != agent.Profession {
			npcProfs = append(npcProfs, p)
		}
	}

	// 2. Profession-side economic settlement
	for _, prof := range npcProfs {
		view, err := CreateActorView(CreateActorRefForProfession(prof), silo, nil)
		if err != nil {
			continue
		}
		e.UpdateActorState(view, silo, deltaYears, addLog)
	}

	// 3. NPC-side decisions (Professions and Resident Representatives)
	var brain NpcBrain
	
	// 3a. Profession decisions
	for _, prof := range npcProfs {
		view, err := CreateActorView(CreateActorRefForProfession(prof), silo, nil)
		if err != nil {
			continue
		}
		decision := brain.Decide(view, silo, deltaYears)
		if decision == nil {
			continue
		}
		result := e.SubmitAction(view.Ref, silo, decision.Action, nil)
		if result.Executed && addLog != nil && rand.Float64() < decision.LogChance {
			addLog(view.Label() + " " + decision.Message)
		}
	}

	// 3b. Resident Representative decisions (e.g., Faction Leaders)
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if !res.Alive || !res.IsRepresentative {
			continue
		}
		view, err := CreateActorView(CreateActorRefForResident(res), silo, nil)
		if err != nil {
			continue
		}
		decision := brain.Decide(view, silo, deltaYears)
		if decision == nil {
			continue
		}
		result := e.SubmitAction(view.Ref, silo, decision.Action, nil)
		if result.Executed && addLog != nil && rand.Float64() < decision.LogChance {
			addLog(view.Label() + " " + decision.Message)
		}
	}
}

// CalculateScore 最终评分结算
func (e *GameEngine) CalculateScore(silo *model.Silo) *model.GameScore {
	survival := silo.TotalPopulation * 1

	diversity := 0
	for _, p := range silo.Professions {
		if p.Productivity > 0.5 {
			diversity += 100
		}
	}

	heritage := int(math.Floor((1.0 - silo.HistoryBurden) * 500))

	avgIdeology := 0.0
	if len(silo.Professions) > 0 {
		for _, p := range silo.Professions {
			avgIdeology += p.Ideologies[model.IdeologyProForeign]
		}
		avgIdeology /= float64(len(silo.Professions))
	}
	ideology := int(math.Floor(avgIdeology * 200))

	multiplier := 1.0
	if silo.VictoryStatus != nil {
		switch silo.VictoryStatus.Type {
		case "INFORMATION":
			multiplier = 2.0
		case "TIME":
			multiplier = 1.5
		case "REBELLION":
			multiplier = 1.2
		case "EXCLUSIONIST":
			multiplier = 0.5
		case "DEATH":
			multiplier = 0
		case "AGENT_COMPROMISED":
			multiplier = 0
		}
	}

	total := int(math.Floor(float64(survival+diversity+heritage+ideology) * multiplier))

	return &model.GameScore{
		Total:           total,
		SurvivalPoints:  survival,
		DiversityPoints: diversity,
		HeritagePoints:  heritage,
		IdeologyPoints:  ideology,
		Multiplier:      multiplier,
	}
}

// GetEndingNarrative 结局叙事文案
func (e *GameEngine) GetEndingNarrative(silo *model.Silo) string {
	if silo.VictoryStatus == nil {
		return "地堡的故事仍在继续..."
	}

	narrative := silo.VictoryStatus.Description + "\n\n"

	proForeign := 0
	for _, p := range silo.Professions {
		if p.Ideologies[model.IdeologyProForeign] > 0.5 {
			proForeign++
		}
	}
	proForeignRatio := float64(proForeign) / float64(len(silo.Professions))

	if proForeignRatio > 0.7 {
		narrative += "地堡社会展现出了前所未有的开放性，人们渴望与外界建立联系。"
	} else if proForeignRatio < 0.2 {
		narrative += "地堡社会深陷排外情绪，人们对任何来自外部的事物都充满敌意。"
	} else {
		narrative += "地堡社会在保守与开放之间艰难地维持着平衡。"
	}

	if silo.HistoryBurden > 0.5 {
		narrative += " 沉重的历史包袱如阴影般笼罩着每一个人，文明的进步举步维艰。"
	} else {
		narrative += " 过去的一页已被翻开，新的一代正以轻松的姿态面对未来。"
	}

	return narrative
}

// CheckVictoryConditions 胜利判定 (公开包装)
func (e *GameEngine) CheckVictoryConditions(silo *model.Silo, agent *model.Agent) {
	e.checkVictoryConditions(silo, agent)
}

func (e *GameEngine) checkVictoryConditions(silo *model.Silo, agent *model.Agent) {
	if silo.VictoryStatus != nil && silo.VictoryStatus.IsWon {
		return
	}

	// 0. Safeguard 危机 (IT专属)
	if silo.SafeguardRisk >= 1.0 {
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       false,
			Type:        "DEATH",
			Description: "Safeguard 协议被激活。IT部门的过度干预触发了底层核心逻辑，清理程序启动，40号地堡被彻底清洗。",
		}
		return
	}

	// 1. 信息胜利：每个部门至少获得5个其他部门的信息碎片
	allDeptsHaveFragments := false
	if len(silo.Professions) > 0 {
		allDeptsHaveFragments = true
		for _, prof := range silo.Professions {
			unique := map[string]bool{}
			for _, f := range prof.KnownFragments {
				unique[f] = true
			}
			if len(unique) < 5 {
				allDeptsHaveFragments = false
				break
			}
		}
	}

	if allDeptsHaveFragments {
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       true,
			Type:        "INFORMATION",
			Description: "你成功让真相在所有部门间流传。全知视角的拼图终于拼凑完整，地堡的居民迎来了觉醒的黎明。",
		}
		return
	}

	// 2. 时间胜利判定：由"1号地堡覆灭"事件触发后结算
	if silo.Silo1Destroyed {
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       true,
			Type:        "TIME",
			Description: "一号地堡已经覆灭，控制网络断开。40号地堡迎来了属于自己的时间。",
		}
		return
	}

	// 3. 叛乱胜利
	if agent != nil && silo.TotalPopulation > 0 {
		const suspicionThreshold = 1.0
		if agent.SuspicionLevel >= suspicionThreshold {
			silo.VictoryStatus = &model.VictoryStatus{
				IsWon:       false,
				Type:        "AGENT_COMPROMISED",
				Description: "由于传播过多掺杂了个人意图的虚假信息，你的特工身份彻底暴露。司法部已经下达了逮捕令。",
			}
			return
		}

		// TODO: 叛乱胜利逻辑将在此处重写。目前使用地堡叛乱值作为临时判定。
		if silo.Rebellion >= 1.0 {
			silo.VictoryStatus = &model.VictoryStatus{
				IsWon:       true,
				Type:        "REBELLION",
				Description: "你成功组织了反抗力量并发动了叛乱。旧的统治被推翻，幸存者们冲破了封闭的牢笼。",
			}
			return
		}
	}

	// 4. 失败判定 (人口灭绝)
	if silo.TotalPopulation <= 0 {
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       false,
			Type:        "DEATH",
			Description: "地堡内已无生命迹象。人类最后的堡垒沦为了一座寂静的坟墓。",
		}
	}
}

func getProfessionByID(silo *model.Silo, id uint) *model.Profession {
	for i := range silo.Professions {
		if silo.Professions[i].ID == id {
			return &silo.Professions[i]
		}
	}
	return nil
}

// updateIdeology 思潮演化：核心逻辑作用于人口单元 (Cohort)，生产部门同步宏观分布
func (e *GameEngine) updateIdeology(silo *model.Silo, deltaYears float64) {
	stability := silo.Cohesion

	// 1. 人口单元 (Cohort) 级别的思潮演化
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		prof := getProfessionByID(silo, c.ProfessionID)
		if prof == nil {
			continue
		}

		// 恐慌值与社会不稳定性导致思潮偏移
		if prof.PanicValue > 0.3 && stability < 0.5 {
			drift := prof.PanicValue * (1.0 - stability) * deltaYears * 0.01
			c.Ideologies[model.IdeologyProForeign] += drift
		}

		// 恐慌转化为激进意识形态
		if prof.PanicValue > 0 {
			conversionRate := 0.10
			convertedAmount := prof.PanicValue * conversionRate * deltaYears
			c.Ideologies[model.IdeologyProForeign] += convertedAmount
			// 注意：PanicValue 属于部门属性，在 sync 阶段处理，此处仅消耗
		}

		// 限制范围
		for key, val := range c.Ideologies {
			if val > 1.0 {
				c.Ideologies[key] = 1.0
			} else if val < 0 {
				c.Ideologies[key] = 0
			}
		}
	}

	// 2. 特殊地堡特质影响 (如：精神药物)
	for _, trait := range silo.Traits {
		if trait == "psychoactive_meds" {
			itDept := findDept(silo, "IT")
			if itDept != nil {
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					if prof := getProfessionByID(silo, c.ProfessionID); prof != nil && prof.Name != "IT" {
						for key, targetIdeology := range itDept.Ideologies {
							diff := targetIdeology - c.Ideologies[key]
							c.Ideologies[key] += diff * 0.05 * deltaYears
						}
					}
				}
			}
			break
		}
	}

	// 3. 消耗部门恐慌值
	for i := range silo.Professions {
		p := &silo.Professions[i]
		if p.PanicValue > 0 {
			p.PanicValue -= p.PanicValue * 0.10 * deltaYears
			if p.PanicValue < 0 {
				p.PanicValue = 0
			}
		}
	}

	// 4. 同步生产部门的宏观思潮数据 (符合用户设计：公民宏观数据挂在部门下)
	for i := range silo.Professions {
		p := &silo.Professions[i]
		// 重置部门意识形态分布
		newIdeologies := make(map[string]float64)
		totalPop := 0.0
		for _, c := range silo.Cohorts {
			if c.ProfessionID == p.ID {
				weight := float64(c.Count)
				for k, v := range c.Ideologies {
					newIdeologies[k] += v * weight
				}
				totalPop += weight
			}
		}
		if totalPop > 0 {
			for k := range newIdeologies {
				newIdeologies[k] /= totalPop
			}
			p.Ideologies = newIdeologies
		}
	}
}

// updateResources 资源结算：按人头计算，包含生产与转化逻辑
func (e *GameEngine) updateResources(silo *model.Silo, deltaYears float64) {
	if deltaYears <= 0 {
		return
	}

	totalPop := float64(silo.TotalPopulation)
	isRebelling := silo.Rebellion > 0.7

	// 1. 获取各部门生产力与效率
	getEfficiency := func(profName string) float64 {
		prof := findDept(silo, profName)
		if prof == nil {
			return 0
		}
		// 效率默认 1.0
		eff := 1.0
		if isRebelling {
			eff = 0.3
		}
		return eff * float64(prof.Population)
	}

	// 2. 计算基础产率 (2.1 人份 / 人)
	const prodRate = 2.1

	// 3. 计算 Energy 生产 (Mechanical)
	energyProdRate := prodRate * e.perCapitaConsumption["Energy"] * getEfficiency("Mechanical")
	energyConsPopRate := e.perCapitaConsumption["Energy"] * totalPop

	// 4. 计算 Materials 生产 (Mines)
	materialsProdRate := prodRate * e.perCapitaConsumption["Materials"] * getEfficiency("Mines")
	materialsConsPopRate := e.perCapitaConsumption["Materials"] * totalPop

	// 5. 计算 Supplies 生产容量 (Agricultural & Supply)
	agriCapacityRate := prodRate * e.perCapitaConsumption["Supplies"] * getEfficiency("Agricultural")
	supplyCapacityRate := prodRate * e.perCapitaConsumption["Supplies"] * getEfficiency("Supply")
	suppliesConsPopRate := e.perCapitaConsumption["Supplies"] * totalPop

	// 6. 执行转化逻辑
	// 农业部：Energy -> Supplies (1:1)
	// 供给部：Materials -> Supplies (1:1)

	// 获取当前库存量
	var energyRes, materialsRes, suppliesRes *model.Resource
	for i := range silo.Resources {
		switch silo.Resources[i].Type {
		case "Energy":
			energyRes = &silo.Resources[i]
		case "Materials":
			materialsRes = &silo.Resources[i]
		case "Supplies":
			suppliesRes = &silo.Resources[i]
		}
	}

	// 能量转化上限 (库存 + 本期产出 - 人口消耗)
	energyAvailable := math.Max(0, energyRes.Amount/deltaYears+energyProdRate-energyConsPopRate)
	actualAgriSuppliesProd := math.Min(agriCapacityRate, energyAvailable)

	// 矿物转化上限 (库存 + 本期产出 - 人口消耗)
	materialsAvailable := math.Max(0, materialsRes.Amount/deltaYears+materialsProdRate-materialsConsPopRate)
	actualSupplySuppliesProd := math.Min(supplyCapacityRate, materialsAvailable)

	// 7. 更新结算数据
	energyRes.NetBalance = energyProdRate - energyConsPopRate - actualAgriSuppliesProd
	materialsRes.NetBalance = materialsProdRate - materialsConsPopRate - actualSupplySuppliesProd
	suppliesRes.NetBalance = actualAgriSuppliesProd + actualSupplySuppliesProd - suppliesConsPopRate

	// 8. 应用变动
	for i := range silo.Resources {
		r := &silo.Resources[i]
		r.Amount += r.NetBalance * deltaYears
		if r.Amount < 0 {
			r.Amount = 0
		}
	}
}

// updateSiloMetrics 地堡指标更新
func (e *GameEngine) updateSiloMetrics(silo *model.Silo, deltaYears float64) {
	silo.Countdown -= deltaYears
	if silo.Countdown < 0 {
		silo.Countdown = 0
	}

	silo.EventTrigger += (1.0 - silo.Cohesion) * deltaYears * 0.1

	// --- 政治系统：由阵营 (Factions) 驱动 ---
	totalInf := 0.0
	weightedCohesion := 0.0
	radicalInf := 0.0

	for _, f := range silo.Factions {
		totalInf += f.Influence
		weightedCohesion += f.Cohesion * f.Influence
		// 凝聚力（忠诚度）低于 0.4 的阵营视为激进/不满阵营
		if f.Cohesion < 0.4 {
			radicalInf += f.Influence * (0.4 - f.Cohesion)
		}
	}

	if totalInf > 0 {
		// 全局凝聚力与合法性由阵营状态派生
		silo.Cohesion = weightedCohesion / totalInf
		silo.Legitimacy = silo.Cohesion // 基础合法性等同于全局忠诚度
	}

	// 叛乱值受激进阵营影响
	const rebellionThreshold = 0.05
	if totalInf > 0 {
		stressFactor := radicalInf / totalInf
		if stressFactor > rebellionThreshold {
			silo.Rebellion += (stressFactor - rebellionThreshold) * deltaYears * 0.1
		} else {
			silo.Rebellion -= 0.02 * deltaYears
		}
	}

	if silo.Rebellion > 1.0 {
		silo.Rebellion = 1.0
	} else if silo.Rebellion < 0 {
		silo.Rebellion = 0
	}

	// 计算部门间紧张程度 (DeptTension) - 仅作为经济/社会统计，不再驱动政治系统
	totalRel := 0.0
	relCount := 0
	for _, p := range silo.Professions {
		for _, rel := range p.Relations {
			totalRel += rel
			relCount++
		}
	}
	if relCount > 0 {
		silo.DeptTension = 1.0 - (totalRel / float64(relCount))
	}

	// 计算精英与平民割裂程度 (ClassFragmentation) - 宏观数据挂在部门之下
	eliteLoyalty := 0.0
	commonerLoyalty := 0.0
	eliteCount := 0
	commonerCount := 0

	for _, p := range silo.Professions {
		// 从所属的 cohorts 中计算平均忠诚度 (符合用户“宏观数据挂在部门下”的设计)
		avgL := 0.0
		cCount := 0
		for _, c := range silo.Cohorts {
			if c.ProfessionID == p.ID {
				avgL += c.Ideologies[model.IdeologyLoyalty]
				cCount++
			}
		}
		if cCount > 0 {
			avgL /= float64(cCount)
		}

		if p.ClassType == "ELITE" {
			eliteLoyalty += avgL
			eliteCount++
		} else {
			commonerLoyalty += avgL
			commonerCount++
		}
	}

	if eliteCount > 0 && commonerCount > 0 {
		silo.ClassFragmentation = math.Abs((eliteLoyalty / float64(eliteCount)) - (commonerLoyalty / float64(commonerCount)))
	}

	// 每隔一段时间（如每季度）在日志中反馈宏观指标变化
	if silo.CurrentMonth%3 == 0 && e.logs != nil {
		e.logf(fmt.Sprintf("[Political] 凝聚力: %.2f, 叛乱风险: %.2f, 部门紧张度: %.2f", silo.Cohesion, silo.Rebellion, silo.DeptTension))
	}

	e.updatePopulation(silo, deltaYears)
}

// updatePopulation 人口更新
func (e *GameEngine) updatePopulation(silo *model.Silo, deltaYears float64) {
	deathRate := 0.001

	for _, r := range silo.Resources {
		if r.Amount <= 0 {
			deathRate += 0.05
		}
	}

	if silo.Rebellion > 0.8 {
		deathRate += (silo.Rebellion - 0.8) * 0.2
	}

	deaths := float64(silo.TotalPopulation) * deathRate * deltaYears
	deathCount := int(math.Floor(deaths))
	applyPopulationDeathsToCohorts(silo, deathCount)
	if silo.TotalPopulation < 0 {
		silo.TotalPopulation = 0
	}
	refreshProfessionPopulationFromCohorts(silo)
}

func formatPct(v float64) string {
	return itoa(int(v * 100))
}

package engine

import (
	"fmt"
	"math"
	"sort"

	"silo40/internal/model"
)

// ============ 核心流水线事件 (程序流，不再由总线驱动) ============
const (
	EVENT_AGENT_UPDATE    = "AGENT_UPDATE"    // 特工状态更新
	EVENT_RESOURCE_UPDATE = "RESOURCE_UPDATE" // 资源结算
	EVENT_METRICS_UPDATE  = "METRICS_UPDATE"  // 地堡指标更新
	EVENT_IDEOLOGY_UPDATE = "IDEOLOGY_UPDATE" // 思潮演化
	EVENT_VICTORY_CHECK   = "VICTORY_CHECK"   // 胜利判定
)

// ============ 领域事件 (由总线驱动，解耦业务逻辑) ============
const (
	EVENT_TIME_CONTENT = "TIME_CONTENT" // 内容系统触发信号
	EVENT_ACTOR_ACTION = "ACTOR_ACTION" // Actor 动作执行
	EVENT_STORY_EVENT  = STORY_EVENT_TYPE
	EVENT_GAME_OVER    = "GAME_OVER"
)

// GameEngine 事件驱动版游戏引擎
// 所有系统 (特工/资源/思潮/指标/NPC/胜利/剧情) 全部注册为 EventBus 的订阅者，
// 由 TIME_CONTENT/ACTOR_ACTION/STORY_EVENT/GAME_OVER 等领域事件编排触发；
// 游戏规则由 RuleEngine 数据驱动；剧情触发由 TriggerEngine 条件驱动。
type GameEngine struct {
	Bus             *EventBus
	ConditionEngine *ConditionEngine
	ScriptEngine    *ScriptEngine
	MechanicEngine  *MechanicEngine
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

	// NpcPolicy optionally advises NPC decisions (LLM layer). nil keeps the
	// original algorithm bit-for-bit; the service layer injects it.
	NpcPolicy NpcPolicy

	// WeightedEndingResolver optionally replaces non-catastrophic ending triggers
	// with a multi-entity decision. Nil preserves standalone engine behavior.
	WeightedEndingResolver func(*model.Silo, *model.Agent) *model.EndingDecision
}

func NewGameEngine() *GameEngine {
	e := &GameEngine{
		Bus:             NewEventBus(),
		ConditionEngine: NewConditionEngine(),
		ScriptEngine:    NewScriptEngine(),
		MechanicEngine:  NewMechanicEngine(),
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
	e.applyMechanicMetadata()

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

func (e *GameEngine) prepareStepState(silo *model.Silo, agent *model.Agent) {
	e.logs = []string{}
	e.tickStories = nil
	e.RuleEngine.LastState = &State{Silo: silo, Agent: agent, Logs: e.logs}
}

// ============ 系统注册：领域事件 ============
func (e *GameEngine) registerSystems() {
	// --- 领域事件：任意 Actor 动作执行 (玩家/NPC 共用) ---
	e.Bus.Subscribe(EVENT_ACTOR_ACTION, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		actorRef := event.Data["actor"].(ActorRef)
		action := event.Data["action"].(model.AgentAction)
		agent, _ := event.Data["agent"].(*model.Agent)
		actor, err := CreateActorView(actorRef, silo, agent)
		if err != nil {
			failed := model.ActionResult{Executed: false, Message: err.Error()}
			e.actionResult = &failed
			return
		}
		// 职业专属行动必须在服务端校验归属：路由只按行动 ID 查表，
		// 不校验会让任意职业执行任意职业行动（如 Mayor 执行 IT_COVER_UP）。
		if action.Type == model.ActionProfession {
			def := GetProfessionAction(action.ProfessionAction)
			if def == nil {
				failed := model.ActionResult{Executed: false, Message: "Unknown profession action: " + action.ProfessionAction}
				e.actionResult = &failed
				return
			}
			if def.Profession != actor.Profession() {
				failed := model.ActionResult{Executed: false, Message: def.Label + " is not available to " + actor.Profession() + "."}
				e.actionResult = &failed
				return
			}
		}
		result, err := e.runMechanicForEvent(event, ctx, actor)
		if err != nil {
			// 显式失败，不再静默回退 Go 硬编码实现：行动语义的唯一权威来源是
			// events/mechanics/*.js。JS 缺失/报错必须被看见，而不是降级成
			// 另一套语义 (项目原则：校验必须显式失败，禁止假绿灯)。
			failed := model.ActionResult{Executed: false, Message: "Action execution failed: " + err.Error()}
			e.actionResult = &failed
			return
		}
		// IT 被动机制：任何成功的 IT 行动都会恶化 safeguard 风险 (与成本成正比)。
		if result.ActionResult != nil && result.ActionResult.Executed && actor.Profession() == "IT" {
			silo.SafeguardRisk += action.Cost * 0.002
		}
		e.actionResult = result.ActionResult
	})

	// --- 领域事件：游戏结束通告 ---
	e.Bus.Subscribe(EVENT_GAME_OVER, func(event *GameEvent, ctx *EventContext) {
		// 这里可以添加全局游戏结束逻辑，如记录存档或发送通知
	})

	// --- 剧情随机事件效果应用 ---
	e.Bus.Subscribe(EVENT_STORY_EVENT, func(event *GameEvent, ctx *EventContext) {
		silo := event.Data["silo"].(*model.Silo)
		story := event.Data["story"].(*StoryEvent)
		// 剧情效果现在也推荐通过 Mechanic 脚本实现，但这里保留直接执行以兼容旧定义
		if story.Effects != nil {
			story.Effects(silo)
		}
	})
}

// ============ 规则引擎：条件 / 脚本 / 规则 ============
func (e *GameEngine) registerConditions() {
	// 条件逻辑已外挂到外部事件定义中。
	e.ConditionEngine.RegisterMany(map[string]Condition{})
}

func (e *GameEngine) registerScripts() {
	// 规则脚本已外挂到外部事件定义中。
	e.ScriptEngine.RegisterMany(map[string]Script{})
}

func (e *GameEngine) registerRules() {
	// 游戏规则已外挂到 events/rules/ 目录中，
	// 由 service 层统一加载并响应。
	e.RuleEngine.RegisterMany([]GameRule{})
}

// ============ 剧情触发器 ============
func (e *GameEngine) registerTriggers() {
	// 条件驱动剧情事件已迁移为 events/ 与 histories/ 目录中的 JSON 定义，
	// 由 service 层统一加载、入库并在每次推进后评估。
}

// ============ 对外接口 ============

// EnqueueAction 统一动作入口：玩家与 NPC 共用同一执行管线。
func (e *GameEngine) EnqueueAction(actor ActorRef, silo *model.Silo, action model.AgentAction, agent *model.Agent) {
	e.actionResult = nil
	e.Bus.Publish(CreateEvent("action#"+e.nextID(), EVENT_ACTOR_ACTION, map[string]interface{}{
		"silo": silo, "actor": actor, "action": action, "agent": agent,
	}))
}

func (e *GameEngine) CreateTimeContentEvent(silo *model.Silo, agent *model.Agent, source string) *GameEvent {
	return CreateEvent("time_content#"+e.nextID(), EVENT_TIME_CONTENT, map[string]interface{}{
		"silo": silo, "agent": agent, "content_source": source,
	})
}

// EnqueueInstantResolution 结算一次回合产生的领域事件 (如 NPC 行为、胜利判定)。
// 注意：这不推进游戏时间。
func (e *GameEngine) EnqueueInstantResolution(silo *model.Silo, agent *model.Agent) {
	ctx := NewEventContext()
	// NPC 回合
	e.RunNpcTurn(silo, agent, 0, e.logf)
	// 胜负判定
	e.runEventMechanic(EVENT_VICTORY_CHECK, e.CreateVictoryCheckEvent(silo, agent), ctx)
	if silo.VictoryStatus != nil && silo.VictoryStatus.Score == nil {
		silo.VictoryStatus.Score = e.CalculateScore(silo)
		e.Bus.Publish(CreateEvent("game_over#"+e.nextID(), EVENT_GAME_OVER, map[string]interface{}{
			"silo": silo,
		}))
	}
}

// Step 处理队列中的下一个事件，并返回该事件与本步日志。
func (e *GameEngine) Step(silo *model.Silo, agent *model.Agent) (*GameEvent, []string, []*model.StoryEvent) {
	e.prepareStepState(silo, agent)
	event := e.Bus.Step()
	if event == nil {
		return nil, nil, nil
	}
	return event, e.RuleEngine.LastState.Logs, e.tickStories
}

// Drain 处理当前队列中的所有事件。
// 与 service 层 drainQueuedEvents 对齐：除 tickStories 外，也回收事件自带的
// step_logs / step_stories（内容事件把自己的故事写在这里），否则 PassTime
// 路径下内容事件的故事不会进入 TickResult.Stories。
func (e *GameEngine) Drain(silo *model.Silo, agent *model.Agent) ([]string, []*model.StoryEvent) {
	var logs []string
	var stories []*model.StoryEvent
	for e.Bus.Pending() > 0 {
		processed, stepLogs, stepStories := e.Step(silo, agent)
		logs = append(logs, stepLogs...)
		stories = append(stories, stepStories...)
		if processed != nil && processed.Data != nil {
			if moreLogs, ok := processed.Data["step_logs"].(*[]string); ok && moreLogs != nil {
				logs = append(logs, (*moreLogs)...)
			}
			if moreStories, ok := processed.Data["step_stories"].(*[]model.StoryEvent); ok && moreStories != nil {
				for i := range *moreStories {
					stories = append(stories, &(*moreStories)[i])
				}
			}
		}
	}
	return logs, stories
}

// ConsumeActionResult 读取并清空最近一次动作结果。
func (e *GameEngine) ConsumeActionResult() *model.ActionResult {
	if e.actionResult == nil {
		return nil
	}
	res := *e.actionResult
	e.actionResult = nil
	return &res
}

// SubmitAction 兼容旧的同步动作入口。
func (e *GameEngine) SubmitAction(actor ActorRef, silo *model.Silo, action model.AgentAction, agent *model.Agent) model.ActionResult {
	e.EnqueueAction(actor, silo, action, agent)
	e.Drain(silo, agent)
	if e.actionResult != nil {
		res := *e.actionResult
		e.actionResult = nil
		return res
	}
	return model.ActionResult{Executed: false, Message: "Unknown error."}
}

// ExecuteAgentAction 玩家动作入口
func (e *GameEngine) ExecuteAgentAction(silo *model.Silo, agent *model.Agent, action model.AgentAction) model.ActionResult {
	actor := CreateActorRefForAgent(agent, silo)
	return e.SubmitAction(actor, silo, action, agent)
}

// UpdateSiloState 推进一个游戏时间片：执行结算流水线并处理产生的领域事件。
// 语义 = PrepareTick → RunLocalPhase → AdvanceSiloClock → RunPostPhase → Drain。
// 世界 tick (P0.5+) 在多堡之间插入跨堡阶段时按相同顺序拆开调用这三相，
// 单堡路径保持整体调用，随机流消耗顺序逐位不变。
func (e *GameEngine) UpdateSiloState(silo *model.Silo, deltaYears float64, agent *model.Agent, source string) ([]string, []*model.StoryEvent) {
	e.PrepareTick(silo, agent)
	e.RunLocalPhase(silo, agent, deltaYears)
	AdvanceSiloClock(silo)
	e.RunPostPhase(silo, agent, source)

	// 处理产生的领域事件 (如随机剧情、规则引擎触发)
	logs, stories := e.Drain(silo, agent)

	return logs, stories
}

// PrepareTick 重置一次月度结算的内部收集状态并把状态视图指向当前堡。
// 世界 tick 对每个堡各调用一次 (按 SiloID 升序)。
func (e *GameEngine) PrepareTick(silo *model.Silo, agent *model.Agent) {
	e.prepareStepState(silo, agent)
}

// RunLocalPhase 世界 tick 阶段 1 的单堡本地结算：特工/资源/思潮/NPC/指标/
// 胜负 (流水线步骤 1-6)。不推进时钟、不评估触发器——跨堡阶段 (阶段 2)
// 读取的是本月本地结算后的状态，时钟推进 (阶段 3) 在跨堡结算之后。
func (e *GameEngine) RunLocalPhase(silo *model.Silo, agent *model.Agent, deltaYears float64) {
	ctx := NewEventContext()

	// 按照确定性的顺序执行各个子系统 (程序逻辑)
	// 1. 特工状态更新
	e.runEventMechanic(EVENT_AGENT_UPDATE, e.CreateAgentUpdateEvent(silo, agent, deltaYears), ctx)

	// 2. 资源结算
	e.runEventMechanic(EVENT_RESOURCE_UPDATE, e.CreateResourceUpdateEvent(silo, deltaYears), ctx)

	// 3. 思潮演化
	e.runEventMechanic(EVENT_IDEOLOGY_UPDATE, e.CreateIdeologyUpdateEvent(silo, deltaYears), ctx)

	// 4. 权威状态：月度更新后才允许影响本月 NPC 决策；零月路径不进入这里。
	AdvanceAuthorityMonthly(silo)

	// 5. NPC 自主行为
	e.RunNpcTurn(silo, agent, deltaYears, e.logf)

	// 6. 地堡指标更新
	e.runEventMechanic(EVENT_METRICS_UPDATE, e.CreateMetricsUpdateEvent(silo, deltaYears), ctx)

	// 7. 胜负判定
	e.runEventMechanic(EVENT_VICTORY_CHECK, e.CreateVictoryCheckEvent(silo, agent), ctx)
	if silo.VictoryStatus != nil && silo.VictoryStatus.Score == nil {
		silo.VictoryStatus.Score = e.CalculateScore(silo)
		e.Bus.Publish(CreateEvent("game_over#"+e.nextID(), EVENT_GAME_OVER, map[string]interface{}{
			"silo": silo,
		}))
	}
}

// AdvanceSiloClock 推进单个堡的日历一个月。世界 tick 阶段 3 在世界时钟
// 推进后逐堡镜像；单堡路径与原流水线步骤 7 逐位一致。
func AdvanceSiloClock(silo *model.Silo) {
	if silo.CurrentMonth == 12 {
		silo.CurrentMonth = 1
		silo.CurrentYear++
	} else {
		silo.CurrentMonth++
	}
}

// RunPostPhase 世界 tick 阶段 4 的单堡后置处理：规则引擎/触发器评估、
// 随机事件与内容事件触发信号 (流水线步骤 8-9)。必须在时钟推进之后，
// 与历史行为一致。
func (e *GameEngine) RunPostPhase(silo *model.Silo, agent *model.Agent, source string) {
	ctx := NewEventContext()

	state := &State{Silo: silo, Agent: agent, Logs: e.logs}
	e.RuleEngine.LastState = state
	e.TriggerEngine.Evaluate(e.Bus, ctx, state)
	story := e.EventEngine.TriggerRandomEvent(silo, e.Bus, ctx)
	if story != nil {
		e.tickStories = append(e.tickStories, story)
	}

	// 触发内容事件 (由 GameService 监听并处理)
	e.Bus.Publish(e.CreateTimeContentEvent(silo, agent, source))
}

func (e *GameEngine) CreateAgentUpdateEvent(silo *model.Silo, agent *model.Agent, deltaYears float64) *GameEvent {
	return CreateEvent("agent_update#"+e.nextID(), EVENT_AGENT_UPDATE, map[string]interface{}{
		"silo": silo, "agent": agent, "deltaYears": deltaYears,
	})
}

func (e *GameEngine) CreateResourceUpdateEvent(silo *model.Silo, deltaYears float64) *GameEvent {
	return CreateEvent("resource_update#"+e.nextID(), EVENT_RESOURCE_UPDATE, map[string]interface{}{
		"silo": silo, "deltaYears": deltaYears,
	})
}

func (e *GameEngine) CreateIdeologyUpdateEvent(silo *model.Silo, deltaYears float64) *GameEvent {
	return CreateEvent("ideology_update#"+e.nextID(), EVENT_IDEOLOGY_UPDATE, map[string]interface{}{
		"silo": silo, "deltaYears": deltaYears,
	})
}

func (e *GameEngine) CreateMetricsUpdateEvent(silo *model.Silo, deltaYears float64) *GameEvent {
	return CreateEvent("metrics_update#"+e.nextID(), EVENT_METRICS_UPDATE, map[string]interface{}{
		"silo": silo, "deltaYears": deltaYears,
	})
}

func (e *GameEngine) CreateVictoryCheckEvent(silo *model.Silo, agent *model.Agent) *GameEvent {
	return CreateEvent("victory_check#"+e.nextID(), EVENT_VICTORY_CHECK, map[string]interface{}{
		"silo": silo, "agent": agent,
	})
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
	if e.MechanicEngine != nil {
		if def, ok := e.MechanicEngine.EventDefinition(EVENT_AGENT_UPDATE); ok {
			result, err := e.MechanicEngine.Run(def, MechanicContext{
				Event: CreateEvent("actor_state#"+e.nextID(), EVENT_AGENT_UPDATE, map[string]interface{}{
					"silo":       silo,
					"deltaYears": deltaYears,
				}),
				Silo:  silo,
				Actor: view,
			})
			if err == nil {
				for _, mutation := range result.Mutations {
					// 直接应用物理变更。PLAYER 的 ActorRef 必须带上 agent 才能
					// 重新解析 (nil 时 CreateActorView 报错，mutations 会被静默丢弃)。
					e.ApplyMutation(silo, view.Agent(), view.Ref, mutation)
				}
				if addLog != nil {
					for _, entry := range result.Logs {
						addLog(entry)
					}
				}
				return
			}
		}
	}

	// Medical 被动特质：随时间随机获得信息碎片 (对任意 Actor 生效)
	if view.Profession() == "Medical" {
		if rng.Float64() < 0.2 { // 20% chance per year
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
				randomFragment := available[rng.Intn(len(available))]
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
	// Faction-driven loyalty evolution is an explicit, time-scaled step: it
	// advances on monthly ticks and is a no-op for zero-duration instant
	// resolutions (deltaYears == 0), so instant actions cannot accelerate it.
	ApplyFactionLoyaltyDynamics(silo, deltaYears)

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
	brain := NpcBrain{policy: e.NpcPolicy}

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
		if obs, ok := brain.policy.(ExecutionObserver); ok {
			obs.NpcExecuted(view, silo, decision.Action, result)
		}
		if result.Executed && addLog != nil && rng.Float64() < decision.LogChance {
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
		if obs, ok := brain.policy.(ExecutionObserver); ok {
			obs.NpcExecuted(view, silo, decision.Action, result)
		}
		if result.Executed && addLog != nil && rng.Float64() < decision.LogChance {
			addLog(view.Label() + " " + decision.Message)
		}
	}
}

// CalculateScore 最终评分结算
func (e *GameEngine) CalculateScore(silo *model.Silo) *model.GameScore {
	if e.MechanicEngine != nil {
		if def, ok := e.MechanicEngine.FormulaDefinition("score"); ok {
			result, err := e.MechanicEngine.Run(def, MechanicContext{Silo: silo})
			if err == nil && result.Score != nil {
				return result.Score
			}
		}
	}
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

// checkVictoryConditions 是结局判定的 Go 降级副本：主路径是 events/mechanics/
// core_systems.js 的 VICTORY_CHECK mechanic（runEventMechanic 分发），只有当
// JS mechanic 缺失或执行报错时才走到这里（见 runEventMechanic 的 fallback）。
// 两份实现必须保持语义一致：修改任一侧的结局规则时，另一侧必须同步修改，
// 并由 TestCheckVictoryConditionsFallbackSemantics 锁住本侧语义。
func (e *GameEngine) checkVictoryConditions(silo *model.Silo, agent *model.Agent) {
	if silo.VictoryStatus != nil && silo.VictoryStatus.IsWon {
		return
	}

	// 0. Safeguard 危机 (IT专属)
	if silo.SafeguardRisk >= 1.0 {
		silo.VictoryStatus = &model.VictoryStatus{
			IsWon:       false,
			Type:        "DEATH",
			Description: "Safeguard 协议被激活。IT部门的过度干预触发了底层核心逻辑，清理程序启动，驻地地堡被彻底清洗。",
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
			Description: "一号地堡已经覆灭，控制网络断开。40 号派出的特工迎来了属于自己的时间。",
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

		// 叛乱胜利当前以地堡叛乱值 >= 1.0 判定。注意：这是 JS 主路径
		// (core_systems.js VICTORY_CHECK) 的降级副本，两侧规则必须同步修改。
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

	// 2. 计算基础产率 (4.5 人份 / 人)
	const prodRate = 4.5

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
		// 全局凝聚力由阵营状态派生；Mayor 个人正统性不再由地堡指标覆写。
		silo.Cohesion = weightedCohesion / totalInf
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

	// DeptTension must not depend on Go map iteration or DB preload order.
	// Sort a copy by persistent profession ID, then sort each Relations map by key.
	professions := append([]model.Profession(nil), silo.Professions...)
	sort.SliceStable(professions, func(i, j int) bool {
		if professions[i].ID != professions[j].ID {
			return professions[i].ID < professions[j].ID
		}
		return professions[i].Name < professions[j].Name
	})
	totalRel := 0.0
	relCount := 0
	for _, p := range professions {
		keys := make([]string, 0, len(p.Relations))
		for k := range p.Relations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			totalRel += p.Relations[k]
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

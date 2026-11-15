package service

// P0.5 双地堡原型：世界 tick 编排、跨堡事件队列、一号地堡脚本化状态机、
// 第二堡确定性 AI 代操作、迷雾视图。全部多堡机制仅在 multiSilo() 为真时
// 激活；单堡 (空邻堡) 配置不进入本文件的任何路径，行为逐位不变。
//
// 世界月度 tick 的阶段顺序 (多堡研究 §3.2)：
//
//	阶段 0  AI 代操作        —— 非玩家堡的确定性动作 (玩家指令的等价物)
//	阶段 0c 18 号正典链条     —— 时间戳强制触发，权威覆写 18 号状态 (见 silo18_chain.go)
//	阶段 1  各堡本地结算      —— 按 SiloID 升序，流水线步骤 1-6
//	阶段 2a 跨堡队列派发      —— 本地结算后、时钟推进前
//	阶段 2b 一号地堡裁决      —— 读取全部堡阶段 1 后状态
//	阶段 3  时钟推进          —— 世界时钟为权威，各堡镜像
//	阶段 4  各堡后置          —— 按 SiloID 升序，流水线步骤 8-9
//
// 确定性纪律 (多堡研究 §2.3)：所有逐堡循环按 SiloID 升序；跨堡队列按
// (到期月, Seq) 排序；单一 RNG 流不分片 (方案 A)。

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// runOneMonthWorld 多堡月度世界 tick。返回本月全部日志与故事 (各堡合并)。
func (s *GameService) runOneMonthWorld(source string) ([]string, []model.StoryEvent) {
	var logs []string
	var stories []model.StoryEvent
	s.monthPlayerLogs = nil

	// 阶段 0a：延迟任务递减一次，按目标堡分组 (到期者在该堡本地阶段前发布)。
	staged := s.checkDelayedEventsStaged()

	// 阶段 0b：AI 代操作非玩家堡。
	s.runAIPhase(&logs, &stories)

	// 阶段 0c：18 号正典链条 (时间戳强制触发；节点生效时权威覆写 18 号状态)。
	// 单堡 / 无 18 号时为空操作。
	s.runSilo18Chain(&logs, &stories)

	// 阶段 1：各堡本地结算 (按 SiloID 升序)。
	for _, silo := range s.silos {
		if siloEnded(silo) {
			continue
		}
		s.currentSilo = silo
		s.currentAgent = s.agents[silo.ID]
		s.publishStagedDelayedEvents(staged, silo.ID)
		// 到期的延迟 (schedule) 事件必须在这里一次性排空并回收故事，不能让它们
		// 滞留在总线上等 RunLocalPhase 的 NPC SubmitAction 顺带 Drain——那条路径
		// 会把故事丢弃，导致通牒余波/轮班怨气/整肃执行/终止投放等延迟链是否可见
		// 取决于当月该堡 NPC 是否行动。固定在本地阶段前排空，保证故事必然进入
		// 当月反馈，且派发时点确定 (不依赖 RNG 决定的 NPC 行为)。
		delayedLogs, delayedStories := s.engine.Drain(silo, s.currentAgent)
		logs = append(logs, delayedLogs...)
		delayedDeref := derefStories(delayedStories)
		stampSilo(delayedDeref, silo)
		stories = append(stories, delayedDeref...)
		s.engine.PrepareTick(silo, s.currentAgent)
		s.engine.RunLocalPhase(silo, s.currentAgent, 1.0/12.0)
		phaseLogs, phaseStories := s.engine.Drain(silo, s.currentAgent)
		logs = append(logs, phaseLogs...)
		phaseDeref := derefStories(phaseStories)
		stampSilo(phaseDeref, silo)
		stories = append(stories, phaseDeref...)
		// 21 设计卷：谈判层月度低强度通气 (各堡本地结算后、时钟推进前)。
		// 仅当该堡信道畅通时执行回合并消耗 RNG；未开信道的堡逐位不变。
		negotiationLogs, negotiationStories := s.runNegotiationMonth(silo)
		logs = append(logs, negotiationLogs...)
		stampSilo(negotiationStories, silo)
		stories = append(stories, negotiationStories...)
		// 20 修订/ADR-0014：筹备反制层月度回合 (组织怀疑度衰减与 IT 响应)。
		// 仅当该堡已产生组织怀疑度时结算；未做组织动作的堡逐位不变。
		responseLogs, responseStories := s.runEscapeResponseMonth(silo)
		logs = append(logs, responseLogs...)
		stampSilo(responseStories, silo)
		stories = append(stories, responseStories...)
	}

	// 阶段 2a：跨堡队列派发 (各堡本地结算后、时钟推进前)。
	s.dispatchCrossQueue(&logs, &stories)
	// 阶段 2b：一号地堡裁决 (跨堡派发后，读取全部堡阶段 1 后状态)。
	s.runSilo1StateMachine(&logs, &stories)
	// 阶段 2b 黑箱回合 (22 黑箱学习系列 01 卷 §2.2)：观测到达→信念更新→零 RNG
	// 决策→指派 r。功能关闭时早退，逐位不变。
	s.runSilo1BlackBoxRound(&logs, &stories)
	// 阶段 2b'：一号地堡权衡层指标一/二 (22 设计卷)。功能关闭时早退。
	s.runSilo1StewardshipMonth(&logs, &stories)
	// 阶段 2c：堡外逃出群体月度维持 (16 §6.4，Layer 2)。读 1 号注意力现值，
	// 用当月日历算天气；只对激活的存续群体生效，零值状态逐位不变。
	s.runOutsideSustenance(&logs, &stories)

	// 阶段 3：时钟推进。世界时钟为权威，各堡时钟镜像。
	for _, silo := range s.silos {
		engine.AdvanceSiloClock(silo)
	}
	s.world.CurrentYear = s.silo.CurrentYear
	s.world.CurrentMonth = s.silo.CurrentMonth

	// Player operations resolve against live targets in stable SiloID order:
	// the assigned station occupies ActiveSiloID, followed by live Silo 1.
	s.resolvePlayerOperations(&logs, &stories)

	// 阶段 4：各堡后置 (按 SiloID 升序)。
	for _, silo := range s.silos {
		if siloEnded(silo) {
			continue
		}
		s.currentSilo = silo
		s.currentAgent = s.agents[silo.ID]
		s.engine.RunPostPhase(silo, s.currentAgent, source)
		phaseLogs, phaseStories := s.engine.Drain(silo, s.currentAgent)
		logs = append(logs, phaseLogs...)
		postDeref := derefStories(phaseStories)
		stampSilo(postDeref, silo)
		stories = append(stories, postDeref...)
	}

	// 收尾：清空本月跨堡信号、复位结算上下文。
	s.receivedSignals = map[uint][]string{}
	s.currentSilo = nil
	s.currentAgent = nil

	// LLM gating 仅消费玩家堡本月日志 (设计 §9)。
	if s.llmAdvisor != nil {
		s.llmAdvisor.ObserveMonth(s.silo, s.agent, s.monthPlayerLogs)
	}
	s.monthPlayerLogs = nil

	// 邻堡感知过滤：邻堡日常琐碎不进入玩家视野，值得传出的改写成谣言版本；
	// 同月重复条目 (多堡副本/跨堡同名信号) 再合并成一条汇总，不靠堡名前缀
	// 堆文本墙。
	stories = s.perceiveMonthStories(stories)
	stories = mergeNeighborStoryCopies(stories)

	return logs, stories
}

// resolvePlayerOperations keeps operation settlement ordered by target SiloID.
// The assigned station is ActiveSiloID; live Silo 1 is created immediately after it.
func (s *GameService) resolvePlayerOperations(logs *[]string, stories *[]model.StoryEvent) {
	s.resolveStationOperations(logs, stories)
	s.resolveSilo1Operations(logs, stories)
}

// derefStories 把故事指针切片转为值切片 (跳过 nil)。
func derefStories(stories []*model.StoryEvent) []model.StoryEvent {
	out := make([]model.StoryEvent, 0, len(stories))
	for _, st := range stories {
		if st != nil {
			out = append(out, *st)
		}
	}
	return out
}

// stampSilo 就地给一批故事打上来源堡归属。多堡同月会触发相同内容 (初始状态相同、
// 触发条件多为全局时间戳)，月度反馈若不标注来源堡，玩家看到的就是一堵分不清彼此的
// 重复文本墙。
func stampSilo(stories []model.StoryEvent, silo *model.Silo) {
	if silo == nil {
		return
	}
	for i := range stories {
		stories[i].SiloID = silo.ID
		stories[i].SiloName = silo.Name
	}
}

// ============ 阶段 0b：第二堡 AI 代操作 ============

// runAIPhase 为非玩家堡执行确定性 AI 动作 (零时长，避免在阶段 0 内推进时间)。
// 决策为纯规则 (不消耗 RNG)；动作执行本身走与玩家相同的机制管线，其随机性
// 来自单一共享 RNG 流，因阶段顺序固定而可复现。
func (s *GameService) runAIPhase(logs *[]string, stories *[]model.StoryEvent) {
	for _, silo := range s.silos {
		if silo.ID == s.world.PlayerSiloID || siloEnded(silo) {
			continue
		}
		agent := s.agents[silo.ID]
		if agent == nil {
			continue
		}
		action := aiChooseAction(silo, agent)
		if action == nil {
			continue
		}
		if err := s.resolveActionCost(action); err != nil {
			continue
		}
		if agent.ActionPoints < action.Cost {
			continue
		}
		s.currentSilo = silo
		s.currentAgent = agent
		actor := engine.CreateActorRefForAgent(agent, silo)
		s.engine.EnqueueAction(actor, silo, *action, agent)
		actionLogs, actionStories := s.drainQueuedEvents()
		*logs = append(*logs, actionLogs...)
		stampSilo(actionStories, silo)
		*stories = append(*stories, actionStories...)
		_ = s.engine.ConsumeActionResult()
		s.currentSilo = nil
		s.currentAgent = nil
	}
}

// aiChooseAction 确定性 AI 策略 (无 RNG)：对叛乱最高的部门收集情报，或在
// 凝聚力低时进行宣传。只选零时长动作。返回 nil = 本月不行动。
func aiChooseAction(silo *model.Silo, agent *model.Agent) *model.AgentAction {
	if agent.ActionPoints < 10 {
		return nil
	}
	// 找叛乱/恐慌最高的部门 (并列取更靠前者，确定性)。
	best := -1
	bestPanic := -1.0
	for i := range silo.Professions {
		if silo.Professions[i].PanicValue > bestPanic {
			bestPanic = silo.Professions[i].PanicValue
			best = i
		}
	}
	if best >= 0 {
		return &model.AgentAction{
			Type:       model.ActionGatherInfo,
			TargetDept: silo.Professions[best].Name,
		}
	}
	return nil
}

// ============ 阶段 2a：跨堡队列派发 ============

// enqueueCrossSignal 把一个 cross_signal 效果转换为跨堡事件入队。到期月 =
// 当前世界时钟 + delay。Seq 由世界单调计数器分配 (确定性排序用)。
func (s *GameService) enqueueCrossSignal(req engine.CrossSignalRequest) {
	if s.world == nil {
		return
	}
	s.world.CrossSeq++
	due := engine.AdvanceTime(engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}, req.DelayMonths)
	s.world.CrossQueue = append(s.world.CrossQueue, model.CrossEvent{
		Seq:         s.world.CrossSeq,
		SourceSilo:  req.SourceSilo,
		TargetScope: req.TargetScope,
		EventID:     req.EventID,
		DueYear:     due.Year,
		DueMonth:    due.Month,
	})
}

// dispatchCrossQueue 派发所有已到期的跨堡事件，按 (到期月, Seq) 升序。到期
// 判定以当前世界时钟 (时钟推进前) 为基准。派发过程中新入队且已到期的事件会在
// 同一阶段内继续派发 (链式)，直到无到期项。
func (s *GameService) dispatchCrossQueue(logs *[]string, stories *[]model.StoryEvent) {
	for {
		idx := s.nextDueCrossEvent()
		if idx < 0 {
			return
		}
		entry := s.world.CrossQueue[idx]
		s.world.CrossQueue = append(s.world.CrossQueue[:idx], s.world.CrossQueue[idx+1:]...)
		s.fireCrossEvent(entry, logs, stories)
	}
}

// nextDueCrossEvent 返回到期 (DueTime <= 世界当前时钟) 且 (到期月, Seq) 最小的
// 跨堡事件下标；无则 -1。
func (s *GameService) nextDueCrossEvent() int {
	best := -1
	now := engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}
	for i, entry := range s.world.CrossQueue {
		due := engine.GameTime{Year: entry.DueYear, Month: entry.DueMonth}
		if engine.CompareTime(due, now) > 0 {
			continue // 未到期
		}
		if best < 0 || crossEventLess(entry, s.world.CrossQueue[best]) {
			best = i
		}
	}
	return best
}

// crossEventLess 按 (到期月, Seq) 排序。
func crossEventLess(a, b model.CrossEvent) bool {
	ta := engine.GameTime{Year: a.DueYear, Month: a.DueMonth}
	tb := engine.GameTime{Year: b.DueYear, Month: b.DueMonth}
	if c := engine.CompareTime(ta, tb); c != 0 {
		return c < 0
	}
	return a.Seq < b.Seq
}

// fireCrossEvent 在当前世界时钟下把一个跨堡事件施加到其目标域的所有堡。
// 目标堡按 SiloID 升序处理；每个目标堡记录"收到信号"并执行事件定义的内容效果。
func (s *GameService) fireCrossEvent(entry model.CrossEvent, logs *[]string, stories *[]model.StoryEvent) {
	targets := s.resolveCrossTargets(entry)
	for _, target := range targets {
		if siloEnded(target) {
			continue
		}
		def, ok := s.findContentDefinition(entry.EventID)
		if !ok {
			*logs = append(*logs, fmt.Sprintf("[CrossSignal] target event not found: %s", entry.EventID))
			continue
		}
		// ONCE 事件对已触发的目标堡不再重复投递 (防止跨堡链条无限往返)。
		if def.FireMode == engine.ContentFireModeOnce {
			if state, _ := s.contentStateForDefinitionIn(target.ID, def); state.Triggered {
				continue
			}
		}
		s.receivedSignals[target.ID] = append(s.receivedSignals[target.ID], entry.EventID)
		// G7: outbound delivery log for spillover settlement. dispatchCrossQueue
		// pops in (due month, Seq) order and targets iterate in ascending ID
		// order, so this append sequence is fully deterministic.
		s.world.OutboundSignals = append(s.world.OutboundSignals, model.OutboundSignal{
			Seq:        entry.Seq,
			SourceSilo: entry.SourceSilo,
			TargetSilo: target.ID,
			EventID:    entry.EventID,
			Year:       s.world.CurrentYear,
			Month:      s.world.CurrentMonth,
		})
		s.currentSilo = target
		s.currentAgent = s.agents[target.ID]
		// 传 nil 作为 story/log 汇点：handleContentEvent 会同时把故事写进
		// story_sink 与 step_story_sink，若这里再提供 story_sink，同一故事会被
		// drainQueuedEvents 经 step_stories 再收一次，月度反馈里跨堡故事因此翻倍
		// (持久化历史走 recordStories 单份，本就正确)。只经 drain 收一次，与月度
		// 本地通道的取值方式对齐。
		if s.emitContentEvent(def, "CROSS_SIGNAL", engine.NewEventContext(), nil, nil, nil) {
			moreLogs, moreStories := s.drainQueuedEvents()
			*logs = append(*logs, moreLogs...)
			stampSilo(moreStories, target)
			*stories = append(*stories, moreStories...)
		}
		s.currentSilo = nil
		s.currentAgent = nil
	}
}

// resolveCrossTargets 解析跨堡事件的目标域为堡集合 (按 SiloID 升序)。
// 支持的 scope：all / neighbors / neighbors_of:<id> / silo:<id|player>。
func (s *GameService) resolveCrossTargets(entry model.CrossEvent) []*model.Silo {
	scope := entry.TargetScope
	switch {
	case scope == "all":
		return append([]*model.Silo(nil), s.silos...)
	case scope == "neighbors":
		return s.neighborsOf(entry.SourceSilo)
	case strings.HasPrefix(scope, "neighbors_of:"):
		id := parseSiloRef(strings.TrimPrefix(scope, "neighbors_of:"))
		return s.neighborsOf(id)
	case strings.HasPrefix(scope, "silo:"):
		ref := strings.TrimSpace(strings.TrimPrefix(scope, "silo:"))
		var id uint
		if ref == "player" {
			id = s.world.PlayerSiloID
		} else {
			id = parseSiloRef(ref)
		}
		if silo := s.siloByID(id); silo != nil {
			return []*model.Silo{silo}
		}
		return nil
	default:
		return nil
	}
}

func (s *GameService) neighborsOf(siloID uint) []*model.Silo {
	var result []*model.Silo
	for _, id := range s.world.Topology[siloID] {
		if silo := s.siloByID(id); silo != nil {
			result = append(result, silo)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func parseSiloRef(ref string) uint {
	id, err := strconv.ParseUint(strings.TrimSpace(ref), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// ============ 阶段 2b：一号地堡脚本化状态机 ============

// runSilo1StateMachine 一号地堡最小状态机 (多堡研究 §5 方案 A)：观测各堡信号
// (叛乱值) 更新注意力，按 Order 规则发出一次性通牒。全部经跨堡事件表达，确定性。
func (s *GameService) runSilo1StateMachine(logs *[]string, stories *[]model.StoryEvent) {
	if s.fullSilo1() != nil {
		s.syncSilo1Peer()
		s.captureSilo1Signals(logs, stories)
		// 全模拟模式下继续走通牒判定：反应链读的是这份带压力条件的通牒
		// (special:silo1_ultimatum_cross)，早退会让它永远无人签发 (错线修复)。
	}
	w := s.world
	if w == nil || !w.Silo1.Alive {
		return
	}
	if w.Silo1.Attention == nil {
		w.Silo1.Attention = map[uint]float64{}
	}
	// 观测信号：各堡叛乱值 (阶段 1 后状态) 作为威胁评估。
	for _, silo := range s.silos {
		w.Silo1.Attention[silo.ID] = silo.Rebellion
	}
	// 一次性通牒：首个秩序压力越线的堡 (按 SiloID 升序)。仅在观察相位触发
	// (WATCHING = 遗留脚本模式；LOCAL_SIM = 全模拟模式——两种模式下反应链都
	// 读这份带压力条件的通牒)。
	// 压力 = 叛乱与平均恐慌的复合 (engine.OrderPressure)——单读叛乱 ≥0.8 在
	// 现有模型下不可达 (实测玩家上限 ~0.45)，复合判据由 sim 校准。
	if w.Silo1.OrderPhase != "WATCHING" && w.Silo1.OrderPhase != "LOCAL_SIM" {
		return
	}
	// G11: network_jammed 是终止信号信道被干扰的世界事实 (21 分册
	// termination_channel_disrupted 的消费方)。信道被干扰期间 Silo 1 无法签发
	// 通牒；物理渠道 (bombing_terminal 等) 不受影响，仍由内容调度。
	// TODO(adjudication): 是否同时暂停 StationOps 反间谍轨道递增，待设计裁决。
	if w.Flags[model.WorldFlagNetworkJammed] {
		return
	}
	// 22 §4.3 耗尽守卫：权衡层启用且寿命耗尽时反应链入口关闭——不再签发新通牒，
	// 已点火阶梯不受影响。未启用 (零值状态) 恒 false，既有行为逐位不变。
	if s.stewardshipLifespanExhausted() {
		return
	}
	for _, silo := range s.silos {
		// 复合压力 + 有组织煽动 (宣传等级) 双条件：背景漂移推起的街头热度
		// 不足以招致通牒，玩家把不满组织化才算"对秩序的推动"。
		agent := s.agents[silo.ID]
		if siloEnded(silo) || engine.OrderPressure(silo) < engine.Silo1UltimatumPressure ||
			agent == nil || agent.PropagandaLevel < engine.Silo1AgitationPropaganda {
			continue
		}
		w.Silo1.OrderPhase = "ULTIMATUM"
		s.enqueueCrossSignal(engine.CrossSignalRequest{
			SourceSilo:  0, // Silo 1 / 世界
			TargetScope: "silo:" + strconv.FormatUint(uint64(silo.ID), 10),
			EventID:     "silo1_ultimatum_cross",
			DelayMonths: 0,
		})
		*logs = append(*logs, fmt.Sprintf("[Silo1] Ultimatum issued to silo %d", silo.ID))
		s.stewardshipRegisterClearingObligation(silo.ID)
		s.stewardshipChargeLifespan("ultimatum")
		// G10 表面服从脉冲 (22-02 §3.5)：通牒后的表演性服从窗口，
		// 初值月数见引擎常量总表 (TODO(balance))。
		if silo.BlackBox.Initialized {
			silo.BlackBox.ObediencePulse = engine.BlackBoxObediencePulseMonths
		}
		break
	}
}

// ============ scope 上下文 (供内容事件施加期使用) ============

// scopeContextFor 为指定堡构建施加期 scope 上下文。多堡时注入世界/堡列表与
// 跨堡信号、错误汇点；单堡时仅 Self，行为与历史一致。
func (s *GameService) scopeContextFor(self *model.Silo) *engine.ScopeContext {
	sc := &engine.ScopeContext{Self: self}
	if s.multiSilo() {
		sc.World = s.world
		sc.Silos = s.silos
		sc.CrossSignals = &[]engine.CrossSignalRequest{}
		sc.Errors = &[]string{}
	}
	return sc
}

// flushScopeResults 回收施加期产生的跨堡信号 (入队) 与解析错误 (记日志)。
func (s *GameService) flushScopeResults(sc *engine.ScopeContext, event *engine.GameEvent) {
	if sc == nil || !s.multiSilo() {
		return
	}
	if sc.CrossSignals != nil {
		for _, req := range *sc.CrossSignals {
			s.enqueueCrossSignal(req)
		}
	}
	if sc.Errors != nil {
		for _, msg := range *sc.Errors {
			s.appendContentLog(event, "[Scope] "+msg)
		}
	}
}

// ============ 迷雾视图 (RumorView) ============

// buildRumorViews 生成玩家对非玩家堡的模糊认知 (只给量化档，不给精确值)。
// 玩家堡自身真实值照常下发 (不在本视图内)。
func (s *GameService) buildRumorViews() []model.RumorView {
	if !s.multiSilo() {
		return nil
	}
	var views []model.RumorView
	for _, silo := range s.silos {
		if silo.ID == s.world.PlayerSiloID {
			continue
		}
		status := "normal"
		if siloEnded(silo) {
			// 结局类型大小写不敏感：JS mechanic 主路径落小写，Go 降级路径落大写。
			if silo.VictoryStatus != nil && strings.EqualFold(silo.VictoryStatus.Type, "death") {
				status = "destroyed"
			} else {
				status = "silent"
			}
		} else if silo.Rebellion >= 0.5 {
			status = "unrest"
		}
		source := "rumor"
		if s.agent != nil && s.agent.Profession == "IT" {
			source = "it_line"
		}
		view := model.RumorView{
			SiloID:        silo.ID,
			Name:          silo.Name,
			Status:        status,
			LastConfirmed: gameTimestamp(s.world.CurrentYear, s.world.CurrentMonth),
			SignalSource:  source,
		}
		if source == "it_line" {
			view.RebellionBand = rebellionBand(silo.Rebellion)
			view.PopulationBand = populationBand(silo.TotalPopulation)
		}
		views = append(views, view)
	}
	return views
}

func rebellionBand(v float64) string {
	switch {
	case v >= 0.6:
		return "high"
	case v >= 0.3:
		return "mid"
	default:
		return "low"
	}
}

func populationBand(pop int) string {
	switch {
	case pop >= 8000:
		return "high"
	case pop >= 4000:
		return "mid"
	default:
		return "low"
	}
}

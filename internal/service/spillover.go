package service

import (
	"fmt"
	"sort"
	"strconv"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 外溢投递侧 (16 §3.5，Layer 3)。
//
// 纪律：
//   - 外溢不进胜负资格：本文件不写任何资格判定，只产出投递事实
//     (World.OutboundSignals 与堡级投递账本)，计数只进《共证》加权与结算行。
//   - 单一 RNG 流：取样点只有两处——发送侧可靠度抽取与接收侧核验裁量。
//     无投递动作、无包裹到达的运行不进入任何取样，RNG 消耗逐位不变。
//   - 风险落在成本与选择空间：使用暴露累积进组织怀疑度 (ADR-0014) 与
//     1 号反谍轨道 (异常流量)；T2-2 信道审计只降档/中断信道。不没收、
//     不判负——暴露的代价是"日后更贵、更窄"，不是一暴露就完。
//   - 接收侧不碰正典状态：接收结算只读目标堡自身模拟状态 (碎片基础、
//     开放程度、IT 控制强度)，只写世界级投递日志与跨堡队列，不修改目标堡
//     的任何具名人物或模拟状态。
//   - 跨堡编排复用既有世界 tick：包裹走 World.CrossQueue (到期月, Seq)
//     派发纪律，到达结算在跨堡派发阶段执行，不另造影子状态。

// 全部数值为初始校准值。TODO(balance)
const (
	// 信道容量：各信道一次投递能携带的包裹质量份额。
	spilloverCapacityITLine      = 1.00 // 专线：整包夹带
	spilloverCapacityCrossSignal = 0.60 // 信号：一次只够传几行字
	spilloverCapacityPhysical    = 1.00 // 实体：整包携带 (慢)

	// 在途月数。
	spilloverTransitITLine      = 1
	spilloverTransitCrossSignal = 2
	spilloverTransitPhysical    = 6

	// 路上损耗：到达质量乘数 (信号衰减、实体磨损)。
	spilloverTransitSurvivalITLine      = 1.00
	spilloverTransitSurvivalCrossSignal = 0.90
	spilloverTransitSurvivalPhysical    = 0.95

	// 单次使用暴露：组织怀疑度增量 (通信规模驱动组织怀疑度，20 修订/ADR-0014)。
	spilloverUseSuspicionITLine      = 0.08 // 专线夹带是最显眼的通信
	spilloverUseSuspicionCrossSignal = 0.05
	spilloverUseSuspicionPhysical    = 0.03 // 实体交接较信号更不显眼

	// 单次使用暴露：1 号反谍轨道 (异常流量)。专线借的是一号自己的网络，
	// 流量形状最容易被注意到；实体交接不产生网络流量。
	spilloverUseSilo1ExposureITLine      = 0.06
	spilloverUseSilo1ExposureCrossSignal = 0.03
	spilloverUseSilo1ExposurePhysical    = 0.00

	// T2-2 信道审计 (roadmap 反制档，外溢暴露结算的首个落点)：使用次数
	// 到达档位门槛 → 可靠度降一档；降档后跌破地板 → 信道中断。
	spilloverAuditUsesITLine       = 3
	spilloverAuditUsesCrossSignal  = 5
	spilloverAuditUsesPhysical     = 8
	spilloverAuditDowngradeFactor  = 0.5
	spilloverAuditBreakReliability = 0.15

	// 接收结算 (目标堡独立核验)：带权裁量的权重因子。读目标堡自身模拟
	// 状态，不读发送方状态；自由裁量地板保证任何状态下两档都压不没。
	spilloverReceptionBase      = 0.20 // 接受基础权重
	spilloverReceptionQualityW  = 0.55 // 包裹质量 (五种能力完备度)
	spilloverReceptionOpennessW = 0.35 // 开放程度 (亲外思潮人口加权)
	spilloverReceptionFragmentW = 0.20 // 碎片基础 (过地板信念数归一)
	spilloverReceptionControlW  = 0.50 // IT 控制强度 (W 归一)：控制越紧越难接收 (减项)
	spilloverReceptionRejectW   = 1.00 // 拒绝基准权重
	spilloverReceptionFloor     = 0.05 // 自由裁量地板 (同谈判层/响应层口径)

	// 再复制：接收方独立长出下一代。
	spilloverReplicationDelay    = 3    // 接收方需要数月核验并复制一代
	spilloverReplicationFidelity = 0.70 // 每代质量保留 (复制不是无损的)
	spilloverMinRelayQuality     = 0.15 // 再复制质量地板：低于它链条自然终止
	spilloverMaxGeneration       = 5    // 链代次硬上限 (安全边界)

	// 碎片基础归一分母。
	spilloverFragmentNormCount = 9
)

// spilloverChannelForSend 返回该种类第一条未中断信道的账本下标 (建立序，
// 确定性)；无可用信道返回 -1。
func spilloverChannelForSend(silo *model.Silo, kind string) int {
	if silo == nil {
		return -1
	}
	for i := range silo.SpilloverPrep.Channels {
		channel := &silo.SpilloverPrep.Channels[i]
		if channel.Kind == kind && !channel.Broken {
			return i
		}
	}
	return -1
}

// settleSpilloverSends 按效果施加顺序逐条结算外溢投递请求，并把回执叙事
// 并入当前事件的故事汇点。空请求列表不产生任何结算与 RNG 消耗。
func (s *GameService) settleSpilloverSends(silo *model.Silo, event *engine.GameEvent, kinds []string) {
	if silo == nil || len(kinds) == 0 {
		return
	}
	stories := []model.StoryEvent{}
	for _, kind := range kinds {
		story := s.settleSpilloverSend(silo, kind)
		if story != nil {
			stories = append(stories, *story)
		}
	}
	if len(stories) == 0 {
		return
	}
	if event != nil {
		if sink, ok := event.Data["story_sink"].(*[]model.StoryEvent); ok && sink != nil {
			*sink = append(*sink, stories...)
		}
		if sink, ok := event.Data["step_story_sink"].(*[]model.StoryEvent); ok && sink != nil {
			*sink = append(*sink, stories...)
		}
		if source, _ := event.Data["content_source"].(string); source != "" {
			pointers := make([]*model.StoryEvent, 0, len(stories))
			for i := range stories {
				storyCopy := stories[i]
				pointers = append(pointers, &storyCopy)
			}
			s.recordStories(pointers, source)
		}
	}
	for i := range stories {
		s.appendContentLog(event, "[Spillover] "+stories[i].Description)
	}
}

// settleSpilloverSend 结算一次外溢投递 (16 §3.5 触达 R＋暴露 E)：
// 信道查找 → 邻接校验 → 暴露记账 (尝试即付) → 可靠度抽取 → 入队/丢失。
// 返回玩家可见的回执叙事；非法请求显式失败 (记日志并告知原因)。
func (s *GameService) settleSpilloverSend(silo *model.Silo, kind string) *model.StoryEvent {
	if silo == nil {
		return nil
	}
	story := func(title, description string) *model.StoryEvent {
		return &model.StoryEvent{
			ID: "spillover_send_" + kind, Category: "player_action",
			Title: title, Description: description, Type: "TECHNICAL",
		}
	}

	idx := spilloverChannelForSend(silo, kind)
	if idx < 0 {
		return story("外溢投递未执行", "没有可用的"+spilloverChannelLabel(kind)+"信道——先建立信道，再谈投递。")
	}
	quality := silo.SpilloverPrep.PackageQuality
	if quality <= 0 {
		return story("外溢投递未执行", "包裹还没有任何内容——五种能力一份都没装进去，发出去的只会是空壳。")
	}
	channel := &silo.SpilloverPrep.Channels[idx]
	if kind == model.SpilloverChannelCrossSignal && s.world != nil && !s.silosAdjacent(silo.ID, channel.TargetSilo) {
		return story("外溢投递未执行", "跨堡信号够不到目标：定向波束受邻接约束，只能投向相邻的堡。")
	}

	// 能带多少：信道容量 × 包裹质量；路上损耗什么：在途损耗乘数。
	carried := clampUnit(quality * spilloverChannelCapacity(kind))
	arriving := clampUnit(carried * spilloverChannelTransitSurvival(kind))

	// 暴露记账 (尝试即付：失败也会留下痕迹)。
	engine.AddEscapeOrgSuspicion(silo, spilloverChannelUseSuspicion(kind))
	if s.world != nil && s.world.Silo1.Alive {
		s.world.Silo1.Exposure = clampSilo1Unit(s.world.Silo1.Exposure + spilloverChannelUseSilo1Exposure(kind))
	}

	// T2-2 信道审计：使用次数到档 → 可靠度降档 → 跌破地板中断。
	channel.Uses++
	auditNote := ""
	if threshold := spilloverAuditThreshold(kind); threshold > 0 && channel.Uses%threshold == 0 {
		channel.Reliability = clampUnit(channel.Reliability * spilloverAuditDowngradeFactor)
		if channel.Reliability < spilloverAuditBreakReliability {
			channel.Broken = true
			auditNote = "这条信道被对向审计盯死，已经中断。"
		} else {
			auditNote = fmt.Sprintf("信道被对向审计盯上，可靠度降到 %.2f。", channel.Reliability)
		}
	}

	// 可靠度结算：单一 RNG 流取样 (仅此一处发送侧取样)。
	transit := spilloverChannelTransit(kind)
	due := engine.AdvanceTime(engine.GameTime{Year: silo.CurrentYear, Month: silo.CurrentMonth}, transit)
	delivered := engine.DrawFloat64() < channel.Reliability
	silo.SpilloverPrep.Deliveries = append(silo.SpilloverPrep.Deliveries, model.SpilloverDelivery{
		Kind: kind, TargetSilo: channel.TargetSilo,
		Year: silo.CurrentYear, Month: silo.CurrentMonth,
		ArriveYear: due.Year, ArriveMonth: due.Month,
		Quality: arriving, Delivered: delivered,
	})

	if !delivered {
		text := fmt.Sprintf("包裹经%s送出，但在路上丢了——可靠度 %.2f 没有兜住这一次。暴露已经记账，只能重来。",
			spilloverChannelLabel(kind), channel.Reliability)
		if auditNote != "" {
			text += auditNote
		}
		return story("外溢投递：路上丢失", text)
	}

	if s.world != nil {
		s.enqueueSpilloverPackage(silo.ID, channel.TargetSilo, transit, arriving, 1, silo.ID)
	}
	text := fmt.Sprintf("包裹经%s送出：五种能力完备度 %.2f，带出去 %.2f，预计 Y%d M%d 抵达。",
		spilloverChannelLabel(kind), quality, arriving, due.Year, due.Month)
	if auditNote != "" {
		text += auditNote
	}
	return story("外溢投递：已上路", text)
}

// enqueueSpilloverPackage 把一个外溢包裹投递为跨堡事件 (发送侧与再复制共用)。
// 走既有 CrossQueue (到期月, Seq) 派发纪律；Payload 只含标量 (可复现)。
func (s *GameService) enqueueSpilloverPackage(sourceSilo, targetSilo uint, delayMonths int, quality float64, generation int, rootSilo uint) {
	if s.world == nil {
		return
	}
	s.world.CrossSeq++
	due := engine.AdvanceTime(engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}, delayMonths)
	s.world.CrossQueue = append(s.world.CrossQueue, model.CrossEvent{
		Seq:         s.world.CrossSeq,
		SourceSilo:  sourceSilo,
		TargetScope: "silo:" + strconv.FormatUint(uint64(targetSilo), 10),
		EventID:     engine.SpilloverPackageEvent,
		DueYear:     due.Year,
		DueMonth:    due.Month,
		Payload: map[string]string{
			"quality":    strconv.FormatFloat(quality, 'f', 6, 64),
			"generation": strconv.Itoa(generation),
			"root":       strconv.FormatUint(uint64(rootSilo), 10),
		},
	})
}

// settleSpilloverArrival 跨堡派发阶段的外溢包裹到达结算 (16 §3.5 接收 P)：
// 目标堡按自身模拟状态独立核验，决定是否接收并再复制一代。只读目标堡状态，
// 只写世界级日志与跨堡队列。
func (s *GameService) settleSpilloverArrival(entry model.CrossEvent, logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil {
		return
	}
	quality, generation, root, ok := parseSpilloverPayload(entry.Payload)
	if !ok {
		*logs = append(*logs, fmt.Sprintf("[Spillover] malformed package payload seq=%d, dropped", entry.Seq))
		return
	}
	// 再复制代由接收堡发出：发出方若在途中覆灭，这一代不再投递
	// (第一代由玩家动作直接入队，不做发出方存活检查——宿主被毁不追溯
	// 抹掉已在途的包裹，01-information §1 口径)。
	if generation >= 2 {
		if source := s.siloByID(entry.SourceSilo); source == nil || siloEnded(source) {
			*logs = append(*logs, fmt.Sprintf("[Spillover] gen%d package seq=%d dropped: relay silo %d gone", generation, entry.Seq, entry.SourceSilo))
			return
		}
	}
	targets := s.resolveCrossTargets(entry)
	if len(targets) == 0 {
		*logs = append(*logs, fmt.Sprintf("[Spillover] package seq=%d has no live target, dropped", entry.Seq))
		return
	}
	target := targets[0]
	if siloEnded(target) {
		*logs = append(*logs, fmt.Sprintf("[Spillover] package seq=%d arrived after target %d ended, dropped", entry.Seq, target.ID))
		return
	}

	// 独立核验：带权裁量读目标堡自身状态 (单一 RNG 流取样，仅到达时发生)。
	acceptW, rejectW := spilloverReceptionWeights(target, quality)
	total := acceptW + rejectW
	accepted := total > 0 && engine.DrawFloat64()*total < acceptW

	s.world.OutboundSignals = append(s.world.OutboundSignals, model.OutboundSignal{
		Seq:        entry.Seq,
		SourceSilo: entry.SourceSilo,
		TargetSilo: target.ID,
		EventID:    entry.EventID,
		Year:       s.world.CurrentYear,
		Month:      s.world.CurrentMonth,
		Spillover:  true,
		Received:   accepted,
		Generation: generation,
		Quality:    quality,
		RootSilo:   root,
	})

	story := model.StoryEvent{Category: "history", Type: "EXTERNAL"}
	if accepted {
		story.ID = fmt.Sprintf("spillover_received_%d_%d", entry.Seq, target.ID)
		story.Title = "远处的回执"
		story.Description = fmt.Sprintf("来自 %s 方向的消息：有人用自己的材料完成了核验——包裹在那里独立站住了 (质量 %.2f，第 %d 代)。",
			target.Name, quality, generation)
		*logs = append(*logs, fmt.Sprintf("[Spillover] package seq=%d independently received by silo %d (gen %d, quality %.2f)",
			entry.Seq, target.ID, generation, quality))
		s.scheduleSpilloverReplication(entry, target, quality, generation, root, logs)
	} else {
		story.ID = fmt.Sprintf("spillover_rejected_%d_%d", entry.Seq, target.ID)
		story.Title = "远处没有回音"
		story.Description = fmt.Sprintf("%s 方向始终没有回音——包裹送到了，但那里没能独立完成核验 (质量 %.2f，第 %d 代)。",
			target.Name, quality, generation)
		*logs = append(*logs, fmt.Sprintf("[Spillover] package seq=%d reached silo %d but was not independently received (gen %d)",
			entry.Seq, target.ID, generation))
	}
	story.SiloID = target.ID
	story.SiloName = target.Name
	*stories = append(*stories, story)
}

// scheduleSpilloverReplication 接收方再复制一代：向尚未到达过的邻堡投递
// 下一代包裹。不回传来源与已接收者 (无回声)；质量逐代衰减，低于地板或
// 到达代次上限时链条自然终止。
func (s *GameService) scheduleSpilloverReplication(entry model.CrossEvent, target *model.Silo, quality float64, generation int, root uint, logs *[]string) {
	if generation >= spilloverMaxGeneration {
		return
	}
	nextQuality := clampUnit(quality * spilloverReplicationFidelity)
	if nextQuality < spilloverMinRelayQuality {
		*logs = append(*logs, fmt.Sprintf("[Spillover] silo %d received but quality %.2f below relay floor %.2f: chain stops",
			target.ID, nextQuality, spilloverMinRelayQuality))
		return
	}
	reached := s.spilloverChainReached(root)
	neighbors := s.neighborsOf(target.ID)
	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].ID < neighbors[j].ID })
	scheduled := 0
	for _, neighbor := range neighbors {
		if neighbor == nil || neighbor.ID == root || reached[neighbor.ID] || siloEnded(neighbor) {
			continue
		}
		s.enqueueSpilloverPackage(target.ID, neighbor.ID, spilloverReplicationDelay, nextQuality, generation+1, root)
		scheduled++
	}
	if scheduled > 0 {
		*logs = append(*logs, fmt.Sprintf("[Spillover] silo %d replicates gen %d to %d neighbor(s) (quality %.2f)",
			target.ID, generation+1, scheduled, nextQuality))
	}
}

// spilloverChainReached 以 root 为起源的链已到达过的堡集合 (任意代次、
// 任意接收结果的到达记录；用于再复制的去重回声)。
func (s *GameService) spilloverChainReached(root uint) map[uint]bool {
	reached := map[uint]bool{}
	if s.world == nil {
		return reached
	}
	for _, signal := range s.world.OutboundSignals {
		if signal.Spillover && signal.RootSilo == root {
			reached[signal.TargetSilo] = true
		}
	}
	return reached
}

// spilloverReceptionWeights 接收裁量的两档权重 (未归一)。包裹质量、开放
// 程度与碎片基础推高接受；IT 控制强度压低接受。两档都保留自由裁量地板。
func spilloverReceptionWeights(target *model.Silo, quality float64) (accept, reject float64) {
	accept = spilloverReceptionBase +
		spilloverReceptionQualityW*clampUnit(quality) +
		spilloverReceptionOpennessW*spilloverOpenness(target) +
		spilloverReceptionFragmentW*spilloverFragmentBasis(target) -
		spilloverReceptionControlW*spilloverControl(target)
	reject = spilloverReceptionRejectW
	if accept < spilloverReceptionFloor {
		accept = spilloverReceptionFloor
	}
	if reject < spilloverReceptionFloor {
		reject = spilloverReceptionFloor
	}
	return accept, reject
}

// spilloverOpenness 目标堡开放程度：人口加权亲外思潮 (0-1)。读法与
// exclusionistEligible 的 cohort 口径一致 (堡序确定性由调用方保证)。
func spilloverOpenness(silo *model.Silo) float64 {
	if silo == nil {
		return 0
	}
	weight, proForeign := 0.0, 0.0
	for i := range silo.Cohorts {
		cohort := &silo.Cohorts[i]
		if cohort.Count <= 0 {
			continue
		}
		w := float64(cohort.Count)
		proForeign += clampUnitIdeology(cohort.Ideologies[model.IdeologyProForeign]) * w
		weight += w
	}
	if weight == 0 {
		return 0
	}
	return proForeign / weight
}

// spilloverFragmentBasis 目标堡碎片基础：过地板信念数归一 (0-1)。
func spilloverFragmentBasis(silo *model.Silo) float64 {
	if silo == nil {
		return 0
	}
	floor := engine.DefaultInformationConfig().QualRumor
	count := 0
	for i := range silo.Professions {
		for _, belief := range silo.Professions[i].Beliefs {
			if belief.B >= floor {
				count++
			}
		}
	}
	return clampUnit(float64(count) / spilloverFragmentNormCount)
}

// spilloverControl 目标堡 IT 控制强度：残余力量 W 归一 (0-1)。
func spilloverControl(silo *model.Silo) float64 {
	if silo == nil {
		return 0
	}
	return clampUnit(silo.Authority.W / 100)
}

// parseSpilloverPayload 解析包裹载荷；字段缺失或非法显式失败。
func parseSpilloverPayload(payload map[string]string) (quality float64, generation int, root uint, ok bool) {
	qualityText, hasQuality := payload["quality"]
	generationText, hasGeneration := payload["generation"]
	rootText, hasRoot := payload["root"]
	if !hasQuality || !hasGeneration || !hasRoot {
		return 0, 0, 0, false
	}
	qualityValue, err := strconv.ParseFloat(qualityText, 64)
	if err != nil || qualityValue < 0 || qualityValue > 1 {
		return 0, 0, 0, false
	}
	generationValue, err := strconv.Atoi(generationText)
	if err != nil || generationValue < 1 {
		return 0, 0, 0, false
	}
	rootValue, err := strconv.ParseUint(rootText, 10, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return qualityValue, generationValue, uint(rootValue), true
}

// silosAdjacent 两堡是否邻接 (读世界拓扑)。
func (s *GameService) silosAdjacent(a, b uint) bool {
	if s.world == nil {
		return false
	}
	for _, id := range s.world.Topology[a] {
		if id == b {
			return true
		}
	}
	return false
}

// ---- 信道种类参数表 (发送侧) ----

func spilloverChannelCapacity(kind string) float64 {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverCapacityITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverCapacityCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverCapacityPhysical
	}
	return 0
}

func spilloverChannelTransit(kind string) int {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverTransitITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverTransitCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverTransitPhysical
	}
	return 0
}

func spilloverChannelTransitSurvival(kind string) float64 {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverTransitSurvivalITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverTransitSurvivalCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverTransitSurvivalPhysical
	}
	return 0
}

func spilloverChannelUseSuspicion(kind string) float64 {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverUseSuspicionITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverUseSuspicionCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverUseSuspicionPhysical
	}
	return 0
}

func spilloverChannelUseSilo1Exposure(kind string) float64 {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverUseSilo1ExposureITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverUseSilo1ExposureCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverUseSilo1ExposurePhysical
	}
	return 0
}

func spilloverAuditThreshold(kind string) int {
	switch kind {
	case model.SpilloverChannelITLine:
		return spilloverAuditUsesITLine
	case model.SpilloverChannelCrossSignal:
		return spilloverAuditUsesCrossSignal
	case model.SpilloverChannelPhysicalHandoff:
		return spilloverAuditUsesPhysical
	}
	return 0
}

func spilloverChannelLabel(kind string) string {
	switch kind {
	case model.SpilloverChannelITLine:
		return "IT 专线"
	case model.SpilloverChannelCrossSignal:
		return "跨堡信号"
	case model.SpilloverChannelPhysicalHandoff:
		return "实体交接"
	}
	return kind
}

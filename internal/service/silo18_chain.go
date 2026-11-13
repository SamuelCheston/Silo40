package service

// Silo 18 原著事件链：链条节点 N1–N9 (silo18-event-chain-design §2.2/§4)。
//
// 机制 (作者裁决)：
//   - 链条按时间戳强制触发：世界 tick 阶段 0c (AI 群演之后、各堡本地结算前)，
//     到期 (Due <= 世界时钟) 的节点必然生效，不依赖 AI/模拟制造的任何条件。
//   - 节点生效时权威覆写 18 号状态 (计量/人物生死/结局)，覆盖此前 AI/模拟
//     累积的漂移；AI 只在空档当群演，不掌具名人物生死。
//   - 节点生效记录写入 contentStates (key "silo18_chain:<节点>")，随 v3 存档
//     持久化，读档后不重复生效——存档格式因此无需变更。
//   - 节点生效不消耗 RNG；对 40 号的可观测耦合经跨堡队列投递 (按 (到期月, Seq)
//     排序)。确定性直接继承 P0.5 契约。
//
// 跨堡耦合遵循"少数关键、多数独立"：只有世界级动作 (拒洁者翻山、绿地广播外泄、
// 18 号终止断联) 才跨界；每条都有明确因果来源。
// N8 难民按原著流向 17 号，40 号只感知过程信号；N9 复用既有
// silo1_destroyed_signal (Y130) 作为终局信号，不另开新信号。

import (
	"fmt"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// silo18ChainNode 一个正典链条节点。节点按 Due 升序排列。
type silo18ChainNode struct {
	ID          string // N1..N9
	DueYear     int
	DueMonth    int
	Title       string
	Description string
	Apply       func(s *GameService, silo18 *model.Silo, logs *[]string)
}

// silo18ChainNodes N1–N9 节点表 (设计 §2.2 映射表，错峰 >=3 月)。
// 效果只投持久杠杆；对 40 号的耦合信号经跨堡队列投递 (事件见 events/special/)。
var silo18ChainNodes = []silo18ChainNode{
	{
		ID: "N1", DueYear: 122, DueMonth: 2,
		Title:       "治安官 Holston 出堡清洁",
		Description: "18 号治安官 Holston 要求出去，执行清洁后死亡。治安官空缺，禁忌的第一道裂缝出现了。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			killSilo18Named(silo18, "Holston", logs)
			addProfessionPanic(silo18, "Judicial", 0.10)
			addProfessionPanic(silo18, "Mechanical", 0.06)
			addProfessionIdeology(silo18, "Mechanical", model.IdeologyProForeign, 0.03)
			silo18.HistoryBurden = clampUnitChain(silo18.HistoryBurden + 0.02)
			// 耦合：低档谣言流入 40 号 (滞后 1 月)。
			s.enqueueSilo18Signal(silo18.ID, "silo18_rumor_sheriff", 1)
		},
	},
	{
		ID: "N2", DueYear: 122, DueMonth: 5,
		Title:       "市长 Jahns 暴毙，Bernard 夺权",
		Description: "市长 Jahns 下行招募机械师 Juliette；Bernard 毒杀 Jahns 与副手 Marnes，依 Pact 自任代理市长。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			killSilo18Named(silo18, "Jahns", logs)
			killSilo18Named(silo18, "Marnes", logs)
			tagSilo18Named(silo18, "Bernard", "acting_mayor")
			engine.AssignMayorHolder(silo18, "Bernard")
			silo18.SetMayorLegitimacy(silo18.MayorLegitimacy() - 0.15)
			silo18.HistoryBurden = clampUnitChain(silo18.HistoryBurden + 0.08)
			silo18.DeptTension = clampUnitChain(silo18.DeptTension + 0.05)
			addProfessionPanic(silo18, "Mayor", 0.15)
			addProfessionPanic(silo18, "IT", 0.06)
			// 耦合：市长暴毙 / IT 接管谣言 (滞后 1 月)。
			s.enqueueSilo18Signal(silo18.ID, "silo18_rumor_it_takeover", 1)
		},
	},
	{
		ID: "N3", DueYear: 123, DueMonth: 3,
		Title:       "Juliette 拒洁翻山",
		Description: "Juliette 拒绝清洁，走出视野、翻山抵达 17 号——\"有人走出去却没死\"。信条被击穿。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			tagSilo18Named(silo18, "Juliette", "crossed_outside")
			silo18.Rebellion = clampUnitChain(silo18.Rebellion + 0.10)
			silo18.EventTrigger += 0.05
			addAllProfessionsIdeology(silo18, model.IdeologyProForeign, 0.08)
			addProfessionPanic(silo18, "Mechanical", 0.10)
			// 耦合 (强信号，当月即达)：拒洁者走出视野，击穿信条。
			s.enqueueSilo18Signal(silo18.ID, "silo18_refuser_crossed", 0)
		},
	},
	{
		ID: "N4", DueYear: 124, DueMonth: 2,
		Title:       "Mechanical 起义被镇压",
		Description: "机械部门起义，Bernard 血腥镇压，并承认杀害了机械师 George。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			killSilo18Named(silo18, "George", logs)
			silo18.Rebellion = clampUnitChain(silo18.Rebellion + 0.12)
			silo18.Cohesion = clampUnitChain(silo18.Cohesion - 0.08)
			silo18.HistoryBurden = clampUnitChain(silo18.HistoryBurden + 0.06)
			silo18.TotalPopulation -= 350
			if silo18.TotalPopulation < 0 {
				silo18.TotalPopulation = 0
			}
			addProfessionPanic(silo18, "Mechanical", 0.15)
			addProfessionPanic(silo18, "Police", 0.08)
			// 耦合：起义被镇压的谣言 (混乱中消息慢，滞后 2 月；外逃者去了 17 号)。
			s.enqueueSilo18Signal(silo18.ID, "silo18_rumor_uprising", 2)
		},
	},
	{
		ID: "N5", DueYear: 125, DueMonth: 1,
		Title:       "Lukas 任 IT 主管，Juliette 当选市长",
		Description: "Lukas Kyle 经远程授职接任 IT 主管；Juliette 返堡洗冤，当选市长。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			tagSilo18Named(silo18, "Lukas Kyle", "it_head")
			tagSilo18Named(silo18, "Juliette", "mayor")
			engine.AssignMayorHolder(silo18, "Juliette")
			// 权威覆写：政权更迭后的合法性/凝聚力区间。
			silo18.SetMayorLegitimacy(0.75)
			silo18.Cohesion = 0.62
			silo18.Rebellion = clampUnitChain(silo18.Rebellion - 0.10)
			addAllProfessionsPanic(silo18, -0.05)
			// 耦合：政权更迭谣言 (IT 线可察"远程授职"异常，滞后 1 月)。
			s.enqueueSilo18Signal(silo18.ID, "silo18_rumor_regime", 1)
		},
	},
	{
		ID: "N6", DueYear: 126, DueMonth: 1,
		Title:       "掘进机横向挖向 17 号",
		Description: "Juliette 启动掘进机横向挖向 17 号——招致毁灭的禁忌之举。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			silo18.HistoryBurden = clampUnitChain(silo18.HistoryBurden + 0.10)
			silo18.DeptTension = clampUnitChain(silo18.DeptTension + 0.08)
			silo18.SafeguardRisk = clampUnitChain(silo18.SafeguardRisk + 0.05)
			// 耦合 (IT 信道，当月即达)：40 号 IT 侦测到地层掘进震动 / 线路异动。
			s.enqueueSilo18Signal(silo18.ID, "silo18_digging_signal", 0)
		},
	},
	{
		ID: "N7", DueYear: 127, DueMonth: 3,
		Title:       "Silo 1 内乱，绿地情报外泄",
		Description: "Silo 1 内乱；Donald/Charlotte 的无人机发现毒区外有绿地。消息开始往外漏。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			addAllProfessionsIdeology(silo18, model.IdeologyProForeign, 0.10)
			addAllProfessionsPanic(silo18, -0.05)
			silo18.Cohesion = clampUnitChain(silo18.Cohesion + 0.03)
			// 耦合：无人机绿地侦察记录外泄，可能传入 40 号 (滞后 1 月)。
			s.enqueueSilo18Signal(silo18.ID, "silo18_green_reconnaissance", 1)
		},
	},
	{
		ID: "N8", DueYear: 128, DueMonth: 6,
		Title:       "18 号被终止",
		Description: "暴乱之后，Thurman 以纳米虫终止 18 号；Lukas 死。约 200 人经隧道出逃 17 号，Shirly 封死了身后的隧道。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			killSilo18Named(silo18, "Lukas Kyle", logs)
			silo18.TotalPopulation = 0
			silo18.VictoryStatus = &model.VictoryStatus{
				IsWon: false,
				Type:  "DEATH",
				Description: "18 号地堡被一号地堡以纳米虫终止。约 200 人经隧道出逃 17 号——" +
					"他们最终会从那里离开地表。终局收在\"离开\"上。",
			}
			// 耦合 (当月即达)：邻堡静默 / 断联。难民按原著流向 17 号，不入 40 号。
			s.enqueueSilo18Signal(silo18.ID, "silo18_silence", 0)
		},
	},
	{
		ID: "N9", DueYear: 130, DueMonth: 1,
		Title:       "Donald 引爆：一号地堡覆灭",
		Description: "Donald 引爆，与 Silo 1、Thurman 同归于尽。离开 18 号的人们正走在 17 号的地表出口——蓝天白云。",
		Apply: func(s *GameService, silo18 *model.Silo, logs *[]string) {
			// 世界事实：一号地堡覆灭 (与既有 silo1_destroyed_signal 同月合流)。
			// 终局信号复用 silo1_destroyed_signal，不另开新信号 (作者裁决)。
			s.world.Silo1.Alive = false
		},
	},
}

// runSilo18Chain 世界 tick 阶段 0c：18 号正典链条结算。到期节点按序强制生效；
// 生效记录写入 contentStates (随存档持久化)。单堡 / 无 18 号时直接返回，
// 单堡行为逐位不变。
func (s *GameService) runSilo18Chain(logs *[]string, stories *[]model.StoryEvent) {
	silo18 := s.silo18Canon()
	if silo18 == nil {
		return
	}
	now := engine.GameTime{Year: s.world.CurrentYear, Month: s.world.CurrentMonth}

	prevSilo, prevAgent := s.currentSilo, s.currentAgent
	s.currentSilo = silo18
	s.currentAgent = s.agents[silo18.ID]
	defer func() {
		s.currentSilo = prevSilo
		s.currentAgent = prevAgent
	}()

	for _, node := range silo18ChainNodes {
		due := engine.GameTime{Year: node.DueYear, Month: node.DueMonth}
		if engine.CompareTime(due, now) > 0 {
			break // 节点按到期月升序；后面的都未到期
		}
		if s.silo18NodeFired(silo18.ID, node.ID) {
			continue
		}
		node.Apply(s, silo18, logs)
		s.markSilo18NodeFired(silo18.ID, node.ID, now)
		s.emitSilo18ChainStory(node, stories)
		*logs = append(*logs, fmt.Sprintf("[Silo18Chain] node %s (%s) fired at Y%dM%d",
			node.ID, node.Title, now.Year, now.Month))
	}
}

// enqueueSilo18Signal 以 18 号为源向其邻堡 (= 玩家堡) 投递耦合信号。
func (s *GameService) enqueueSilo18Signal(sourceSilo uint, eventID string, delayMonths int) {
	s.enqueueCrossSignal(engine.CrossSignalRequest{
		SourceSilo:  sourceSilo,
		TargetScope: "neighbors",
		EventID:     eventID,
		DelayMonths: delayMonths,
	})
}

// emitSilo18ChainStory 记录链条节点故事到 18 号事件日志与本月输出汇点。
func (s *GameService) emitSilo18ChainStory(node silo18ChainNode, stories *[]model.StoryEvent) {
	story := model.StoryEvent{
		ID:          "silo18_chain_" + node.ID,
		Category:    "history",
		Title:       node.Title,
		Description: node.Description,
		Type:        "EXTERNAL",
	}
	*stories = append(*stories, story)
	storyCopy := story
	s.recordStories([]*model.StoryEvent{&storyCopy}, "SILO18_CHAIN:"+node.ID)
}

func silo18ChainStateKey(nodeID string) string { return "silo18_chain:" + nodeID }

// silo18NodeFired 节点是否已生效 (生效记录随 v3 存档持久化)。
func (s *GameService) silo18NodeFired(siloID uint, nodeID string) bool {
	states := s.contentStates[siloID]
	if states == nil {
		return false
	}
	return states[silo18ChainStateKey(nodeID)].Triggered
}

// markSilo18NodeFired 写入节点生效记录 (持久化键，防重复生效)。
func (s *GameService) markSilo18NodeFired(siloID uint, nodeID string, now engine.GameTime) {
	states := s.statesMapFor(siloID)
	key := silo18ChainStateKey(nodeID)
	state := states[key]
	state.SiloID = siloID
	state.EventKey = key
	state.Triggered = true
	state.TriggerCount++
	state.LastTriggeredTimestamp = gameTimestamp(now.Year, now.Month)
	states[key] = state
}

// ============ 18 号侧权威覆写辅助 ============

// killSilo18Named 链条节点权威设定具名人物死亡。
func killSilo18Named(silo18 *model.Silo, name string, logs *[]string) {
	if r := namedResident(silo18, name); r != nil && r.Alive {
		r.Alive = false
		*logs = append(*logs, fmt.Sprintf("[Silo18Chain] %s died", name))
	}
}

// tagSilo18Named 链条节点权威追加具名人物角色标签 (幂等)。
func tagSilo18Named(silo18 *model.Silo, name, tag string) {
	if r := namedResident(silo18, name); r != nil && !residentHasTag(r, tag) {
		r.Tags = append(r.Tags, tag)
	}
}

func addProfessionPanic(silo18 *model.Silo, profession string, delta float64) {
	for i := range silo18.Professions {
		if silo18.Professions[i].Name == profession {
			silo18.Professions[i].PanicValue = clampUnitChain(silo18.Professions[i].PanicValue + delta)
			return
		}
	}
}

func addAllProfessionsPanic(silo18 *model.Silo, delta float64) {
	for i := range silo18.Professions {
		silo18.Professions[i].PanicValue = clampUnitChain(silo18.Professions[i].PanicValue + delta)
	}
}

func addProfessionIdeology(silo18 *model.Silo, profession, ideology string, delta float64) {
	for i := range silo18.Professions {
		if silo18.Professions[i].Name != profession {
			continue
		}
		if silo18.Professions[i].Ideologies == nil {
			silo18.Professions[i].Ideologies = map[string]float64{}
		}
		silo18.Professions[i].Ideologies[ideology] = clampUnitChain(silo18.Professions[i].Ideologies[ideology] + delta)
		return
	}
}

func addAllProfessionsIdeology(silo18 *model.Silo, ideology string, delta float64) {
	for i := range silo18.Professions {
		if silo18.Professions[i].Ideologies == nil {
			silo18.Professions[i].Ideologies = map[string]float64{}
		}
		silo18.Professions[i].Ideologies[ideology] = clampUnitChain(silo18.Professions[i].Ideologies[ideology] + delta)
	}
}

func clampUnitChain(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

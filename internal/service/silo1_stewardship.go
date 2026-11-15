package service

// 1 号权衡层调度器 (22 设计卷、ADR-0015/0016)。
//
// 纪律：
//   - 仅多堡且 1 号存活时运行；单堡 (空邻堡) 配置与零值状态早退，行为逐位不变。
//   - 月度挂载点在 runOneMonthWorld 阶段 2b (runSilo1StateMachine 之后、阶段 2c
//     之前)：此时各堡阶段 1 后状态已定、跨堡队列已派发，1 号能读到当月全貌。
//   - 固定调用顺序 (确定性)：指标一 → 指标二 → 指标三；RNG 取样全部经引擎
//     单一 RNG 流，取样顺序按 SiloID 升序。
//   - 黑箱壁：决策层 (本包) 不得读物理层曲线参数与 Silo.SelfPreservation 真值，
//     只能消费 engine 包生成的观测包 (22 §6.3)。

import "silo40/internal/model"

// stewardshipDecisionLogCap 决策日志环形保留条数 (呈现与审计共用)。
const stewardshipDecisionLogCap = 24 // TODO(balance)

// runSilo1StewardshipMonth 1 号权衡层月度回合。
func (s *GameService) runSilo1StewardshipMonth(logs *[]string, stories *[]model.StoryEvent) {
	if !s.multiSilo() || s.world == nil || !s.world.Silo1.Alive {
		return
	}
	st := &s.world.Silo1Stewardship
	if !st.Initialized {
		s.initSilo1Stewardship()
	}
	// 固定顺序：指标一 (履约与多样性) → 指标二 (寿命) → 指标三 (指派决策)。
	s.stewardshipExperimentMonth(logs, stories)
	s.stewardshipLifespanMonth(logs, stories)
	s.stewardshipAssignmentMonth(logs, stories)
}

// initSilo1Stewardship 首次启用：写入初始状态 (确定性常量，不消耗 RNG)。
func (s *GameService) initSilo1Stewardship() {
	st := &s.world.Silo1Stewardship
	st.Initialized = true
	st.Experiment.Scheduled = map[uint]int{}
	st.Experiment.Cleared = map[uint]int{}
	st.Experiment.Missed = map[uint]bool{}
	st.Lifespan.Initialized = true
	st.Lifespan.AvgLifespan = stewardshipLifespanStart
	st.Lifespan.Postures = map[uint]string{}
	st.Assignments = map[uint]model.Silo1AssignmentState{}
	for _, silo := range s.silos {
		if siloEnded(silo) {
			continue
		}
		st.Lifespan.Postures[silo.ID] = model.Silo1PostureNormal
		st.Assignments[silo.ID] = model.Silo1AssignmentState{
			Posture:               model.Silo1PostureNormal,
			BeliefEffectUnc:       1.0, // 开局完全不确定："它在猜"的起点
			BeliefDisaffectionUnc: 1.0,
			Incidents:             map[string]int{},
		}
	}
}

// recordSilo1Decision 追加一条决策留痕 (环形截断)。
func (s *GameService) recordSilo1Decision(rec model.Silo1DecisionRecord) {
	st := &s.world.Silo1Stewardship
	st.Decisions = append(st.Decisions, rec)
	if len(st.Decisions) > stewardshipDecisionLogCap {
		st.Decisions = st.Decisions[len(st.Decisions)-stewardshipDecisionLogCap:]
	}
}

// populateStewardshipView 把 1 号权衡层的最小可读呈现挂到世界视图上：两个
// 指标 + 指派决策依据 (22 §8)。未启用 (单堡/旧档/零值状态) 时不挂任何字段，
// 消费方行为逐位不变。
func (s *GameService) populateStewardshipView(view *model.WorldView) {
	if view == nil || s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return
	}
	st := &s.world.Silo1Stewardship
	view.StewardshipEnabled = true
	exp := st.Experiment
	view.Silo1Experiment = &model.ExperimentStewardshipView{
		FulfillmentDeficit: exp.FulfillmentDeficit,
		DiversityLoss:      exp.DiversityLoss,
		ScheduledCount:     len(exp.Scheduled),
		ClearedCount:       len(exp.Cleared),
		MissedCount:        len(exp.Misses),
	}
	view.Silo1Lifespan = &model.LifespanStewardshipView{
		AvgLifespan: st.Lifespan.AvgLifespan,
		Postures:    st.Lifespan.Postures,
	}
	if n := len(st.Decisions); n > 0 {
		view.Silo1RecentDecisions = append([]model.Silo1DecisionRecord(nil), st.Decisions...)
	}
}

package service

// 1 号权衡层调度器 (22 设计卷、22 黑箱学习系列、ADR-0018)。
//
// 纪律：
//   - 功能开关门控：World.Silo1BlackBoxEnabled 为 false (默认/旧档) 时全步骤
//     跳过，零 RNG 消耗，与既有 parity 基线逐位一致。
//   - 指标一/二月度回合挂阶段 2b'（黑箱回合之后）；指标三的黑箱回合挂阶段
//     2b 既有内容之后 (22 黑箱学习系列 01 卷 §2)，Θ/E 结算与观测产生挂
//     RunLocalPhase 4.55/4.65 (engine 包)。
//   - 确定性：信念更新与决策全程零 RNG；唯一 RNG 消耗点是观测产生 (门控、
//     SiloID 升序、信号类型升序固定取样)。

import "silo40/internal/model"

// stewardshipLifespanStart 平均寿命初始值 (年)。TODO(balance)
const stewardshipLifespanStart = 20.0

// silo1BlackBoxOn 权衡层功能开关读数 (建档请求持久化到世界)。
func (s *GameService) silo1BlackBoxOn() bool {
	return s.world != nil && s.world.Silo1BlackBoxEnabled
}

// runSilo1StewardshipMonth 指标一/二月度回合 (阶段 2b')。
func (s *GameService) runSilo1StewardshipMonth(logs *[]string, stories *[]model.StoryEvent) {
	if !s.silo1BlackBoxOn() || !s.world.Silo1.Alive {
		return
	}
	if !s.world.Silo1Stewardship.Initialized {
		s.initSilo1Stewardship()
	}
	// 固定顺序：指标一 (履约与多样性) → 指标二 (寿命) → 缺口通道 → Θ 行为化。
	s.stewardshipExperimentMonth(logs, stories)
	s.stewardshipLifespanMonth(logs, stories)
	s.runBlackBoxChannelsMonth(logs, stories)
	s.runBlackBoxBehaviorMonth(logs, stories)
}

// initSilo1Stewardship 首次启用：写入指标一/二初始状态 (确定性常量，不消耗 RNG)。
func (s *GameService) initSilo1Stewardship() {
	st := &s.world.Silo1Stewardship
	st.Initialized = true
	st.Experiment.Scheduled = map[uint]int{}
	st.Experiment.Cleared = map[uint]int{}
	st.Experiment.Missed = map[uint]bool{}
	st.Experiment.Ledger = map[string]int{}
	st.Lifespan.Initialized = true
	st.Lifespan.AvgLifespan = stewardshipLifespanStart
	st.Lifespan.Postures = map[uint]string{}
	st.Lifespan.Ledger = map[string]int{}
	for _, silo := range s.silos {
		if siloEnded(silo) {
			continue
		}
		st.Lifespan.Postures[silo.ID] = model.Silo1PostureNormal
	}
}

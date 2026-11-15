package service

// 指标三：safeguard 风险度指派与 selfPreservation 动力学 (22 设计卷 §5/§6)。
// 属格三实现。
//
// 黑箱纪律 (ADR-0015)：本文件的决策代码不得读取物理层曲线参数、不得读取
// Silo.SelfPreservation 真值，只能消费 engine 包生成的观测包；决策基于 1 号
// 自己的信念估计 (估计数值 + 不确定性)，日志体现"它在猜"。

import "silo40/internal/model"

// stewardshipAssignmentMonth 指标三月度回合 (格三实现)。
// 职责：物理层月度推进 (S 动力学与观测生成，经 engine 包)、信念更新、冷却到
// 期时的带权随机指派 (风险度 + 监控档位)、决策日志。
func (s *GameService) stewardshipAssignmentMonth(logs *[]string, stories *[]model.StoryEvent) {
	// TODO(格三): 按 22 设计卷 §6 实现。
}

// stewardshipRegisterIncident 后果事件登记 (原累加点改为决策输入，ADR-0016)。
// kind ∈ {ignore, cover_collapse, deviation, it_overreach}。由各原累加点调用；
// 格三实现写入 Assignments[siloID].Incidents。
func (s *GameService) stewardshipRegisterIncident(siloID uint, kind string) {
	// TODO(格三): 登记 incident。
}

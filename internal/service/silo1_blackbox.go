package service

import "silo40/internal/model"

// runSilo1BlackBoxRound 阶段 2b 黑箱回合 (22 黑箱学习系列 01 卷 §2.2)：
// 观测到达 → 信念更新 → 摘要 → 零 RNG 决策 → 指派 r。格三 (黑箱回合) 实现。
func (s *GameService) runSilo1BlackBoxRound(logs *[]string, stories *[]model.StoryEvent) {
	// TODO(黑箱回合格): 按 22 黑箱学习系列 01 卷 §7.1 伪代码实现。
}

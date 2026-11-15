package service

// 指标一：实验履约与保留多样性 (22 设计卷 §3)。属格一实现。
//
// 语义：该按要求清除的叛乱地堡没被清除 → 履约缺口恶化；每清除一个地堡 →
// 多样性损耗恶化 (随时间衰减，越接近实验结束 Y130M1 清除带来的损耗越轻)。
// 该指标只影响"最终可保留地堡的多样性"，不判负、不处刑。

import "silo40/internal/model"

// stewardshipExperimentMonth 指标一月度回合 (格一实现)。
// 职责：清除义务登记 (通牒签发)、清除成立检测 (DEATH)、miss 结算、双指标衰减。
func (s *GameService) stewardshipExperimentMonth(logs *[]string, stories *[]model.StoryEvent) {
	// TODO(格一): 按 22 设计卷 §3 实现。
}

// stewardshipRegisterClearingObligation 通牒签发时登记清除义务 (由反应链签发
// 路径调用；格一实现登记逻辑)。
func (s *GameService) stewardshipRegisterClearingObligation(siloID uint) {
	// TODO(格一): 登记 Scheduled。
}

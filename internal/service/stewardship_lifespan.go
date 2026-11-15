package service

// 指标二：剩余人员平均寿命 (22 设计卷 §4)。属格二实现。
//
// 语义：1 号人员轮流休眠；时间本身必然使平均寿命下降 (什么都不做也保不住)，
// 行动按监控档位额外扣减 (HIGH > NORMAL > REPORT_ONLY)。耗尽导致能力退化
// (档位强制下调、不再签发新通牒)，不是判负；18 号正典链不读本指标。

import "silo40/internal/model"

// stewardshipLifespanMonth 指标二月度回合 (格二实现)。
// 职责：自然衰减、档位成本扣减、耗尽退化、一次性行动成本由
// stewardshipChargeLifespan 记账。
func (s *GameService) stewardshipLifespanMonth(logs *[]string, stories *[]model.StoryEvent) {
	// TODO(格二): 按 22 设计卷 §4 实现。
}

// stewardshipChargeLifespan 一次性行动成本 (签发通牒/整肃链/终止链点火时由
// 对应路径调用；格二实现扣减与下限)。
func (s *GameService) stewardshipChargeLifespan(kind string) {
	// TODO(格二): opCost* 扣减。
}

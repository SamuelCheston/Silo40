package service

import (
	"fmt"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// runEscapeResponseMonth 逃跑筹备反制层的月度回合 (20 分册修订、ADR-0014)。
// 位置与谈判层月度通气相同：各堡本地结算后、时钟推进前 (单堡路径同)。
//
// 纪律：
//   - 未产生组织怀疑度的运行不进入任何结算与 RNG 取样，逐位不变。
//   - 开窗 (escape_window_open) 或出发后层结束：叛乱已成公开事实，
//     秘密筹备期的响应失去对象；进行中的蒙蔽就此收尾，不再计费。
//   - 怀疑度按月衰减：组织行为停了，风声也会淡 (同淡忘口径)。
func (s *GameService) runEscapeResponseMonth(silo *model.Silo) ([]string, []model.StoryEvent) {
	if silo == nil || silo.VictoryStatus != nil || !silo.EscapeResponse.Initialized {
		return nil, nil
	}
	if s.escapeResponseLayerOver(silo) {
		s.escapeResponseEndCover(silo, "开窗/出发后筹备响应层结束")
		return nil, nil
	}
	engine.DecayEscapeOrgSuspicion(silo)
	return s.escapeResponseDecide(silo)
}

// escapeResponseLayerOver 报告筹备响应层是否应当停摆：撤离窗口已开
// (筹备期结束，叛乱成为公开事实) 或逃出群体已成立 (动向已落地)。
func (s *GameService) escapeResponseLayerOver(silo *model.Silo) bool {
	if silo.EscapedGroup.Initialized {
		return true
	}
	states := s.contentStates[silo.ID]
	if states == nil {
		return false
	}
	return states["special:escape_window_open"].Triggered
}

// escapeResponseEndCover 收尾进行中的蒙蔽 (层结束时不计费、不判崩塌)。
func (s *GameService) escapeResponseEndCover(silo *model.Silo, outcome string) {
	state := &silo.EscapeResponse
	if !state.CoverActive {
		return
	}
	state.CoverActive = false
	state.CoverMonths = 0
	if len(state.Decisions) > 0 {
		state.Decisions[len(state.Decisions)-1].Outcome = outcome
	}
}

// escapeResponseDecide 月度决策与蒙蔽维持：
//  1. 蒙蔽进行中：按月支付 R_op 并抽取暴露；崩塌时 safeguard 风险上浮。
//     蒙蔽期间不做新决策 (IT 的裁量已被占用)。
//  2. 怀疑度到门槛且冷却已过：IT 带权随机三选一 (镇压/蒙蔽/不作为)，
//     归一化权重与因子摘要写入决策日志，叙事走月度故事通道。
func (s *GameService) escapeResponseDecide(silo *model.Silo) ([]string, []model.StoryEvent) {
	state := &silo.EscapeResponse
	year, month := silo.CurrentYear, silo.CurrentMonth

	if state.CoverActive {
		if !engine.EscapeResponseCoverMonth(silo) {
			return nil, nil // 蒙蔽维持住了：本月无事发生 (这本身就是掩护)
		}
		if len(state.Decisions) > 0 {
			state.Decisions[len(state.Decisions)-1].Outcome = "蒙蔽崩塌"
		}
		log := fmt.Sprintf("[EscapeResponse] %d-%02d 蒙蔽第 %d 个月崩塌：1 号察觉异常，safeguard 风险上浮",
			year, month, state.CoverMonths)
		story := s.escapeResponseStory(silo, "cover_collapse", "上级的例行问询",
			"上级来了一份例行问询，问起近几个月的流量形状异常。IT 的回复报告比往常长。")
		return []string{log}, []model.StoryEvent{story}
	}

	result := engine.RunEscapeResponseDecision(silo)
	if result == nil {
		return nil, nil
	}
	state.AppendDecision(model.EscapeResponseLogEntry{
		Year:     year,
		Month:    month,
		Decision: result.Decision,
		Weights:  result.Weights,
		Factors:  result.Factors,
	})

	var title, description string
	switch result.Decision {
	case model.EscapeResponseSuppress:
		title = "突击清查"
		description = "一轮突击清查扫过维修夹层与集结点，批次暂时安静地散开了。部门戒备比往常更紧。"
	case model.EscapeResponseConceal:
		title = "流量排班调整"
		description = "IT 调整了一批例行流量排班，没有解释。管道里的形状与往常不太一样了。"
	case model.EscapeResponseIgnore:
		title = "记录归档"
		description = "一些异常记录被归档了。没有人追查，但它们还留在档案里。"
	}
	log := fmt.Sprintf("[EscapeResponse] %d-%02d 怀疑度=%.2f 决策=%s 因子=%s",
		year, month, state.OrgSuspicion, result.Decision, result.Factors)
	story := s.escapeResponseStory(silo, result.Decision, title, description)
	return []string{log}, []model.StoryEvent{story}
}

// escapeResponseStory 响应层叙事：只描述发生的事，不指认玩家为对象
// (呈现纪律同谈判层)。
func (s *GameService) escapeResponseStory(silo *model.Silo, key, title, description string) model.StoryEvent {
	return model.StoryEvent{
		ID:          fmt.Sprintf("escape_response:%d:%d:%d:%s", silo.CurrentYear, silo.CurrentMonth, silo.ID, key),
		Category:    "SOCIAL",
		Title:       title,
		Description: description,
		Type:        "SOCIAL",
		SiloID:      silo.ID,
		SiloName:    silo.Name,
	}
}

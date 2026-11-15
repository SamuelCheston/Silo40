package service

import (
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

// escapeResponseDecide 月度决策与蒙蔽维持。提交 2 仅落怀疑度衰减；
// IT 带权三选一与蒙蔽结算由后续提交接入。
func (s *GameService) escapeResponseDecide(silo *model.Silo) ([]string, []model.StoryEvent) {
	return nil, nil
}

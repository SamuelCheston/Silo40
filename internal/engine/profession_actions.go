package engine

import (
	"silo40/internal/model"
)

// ============ 职业专属行动注册表 (元数据层) ============
// 每个职业拥有 2 个专属行动。行动定义的权威来源是
// events/mechanics/profession_actions.js：引擎构造时 applyMechanicMetadata
// 用 JS 定义整体替换本注册表 (成本 / 目标类型 / 怀疑度惩罚等元数据)。
// 这里曾存在的 Go Effect 实现已全部移除——行动效果只由 JS 机制脚本执行；
// JS 不可用时行动显式失败，而不是降级到另一套语义 (禁止假绿灯)。

type ProfessionActionTargetType string

const (
	TargetNone     ProfessionActionTargetType = "NONE"
	TargetDept     ProfessionActionTargetType = "DEPT"
	TargetResource ProfessionActionTargetType = "RESOURCE"
)

type ProfessionAction struct {
	ID               string
	Profession       string
	Label            string
	Description      string
	APCost           float64
	TargetType       ProfessionActionTargetType
	SuspicionPenalty float64
}

var PROFESSION_ACTIONS = []*ProfessionAction{}

// ---- 通用辅助 ----

func findDept(silo *model.Silo, name string) *model.Profession {
	for i := range silo.Professions {
		if silo.Professions[i].Name == name {
			return &silo.Professions[i]
		}
	}
	return nil
}

type actorTarget struct {
	agent    *model.Agent
	prof     *model.Profession
	resident *model.Resident
}

func (v *ActorView) agentOrProf() actorTarget {
	if v.agent != nil {
		return actorTarget{agent: v.agent}
	}
	if v.resident != nil {
		return actorTarget{resident: v.resident}
	}
	return actorTarget{prof: v.prof}
}

// GetProfessionActions 按职业筛选行动定义
func GetProfessionActions(profession string) []*ProfessionAction {
	var out []*ProfessionAction
	for _, a := range PROFESSION_ACTIONS {
		if a.Profession == profession {
			out = append(out, a)
		}
	}
	return out
}

// GetProfessionAction 按 id 查找行动定义
func GetProfessionAction(id string) *ProfessionAction {
	for _, a := range PROFESSION_ACTIONS {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// GetProfessionActionMeta 返回职业行动元数据 (前端渲染用，不含执行逻辑)
func GetProfessionActionMeta(profession string) []model.ProfessionActionMeta {
	var out []model.ProfessionActionMeta
	for _, a := range GetProfessionActions(profession) {
		out = append(out, model.ProfessionActionMeta{
			ID:               a.ID,
			Profession:       a.Profession,
			Label:            a.Label,
			Description:      a.Description,
			APCost:           int(a.APCost),
			TargetType:       string(a.TargetType),
			SuspicionPenalty: a.SuspicionPenalty,
		})
	}
	return out
}

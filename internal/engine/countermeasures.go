package engine

import (
	"silo40/internal/model"
)

// 20 分册第 5 批：世界反制剩余四档的确定性月度结算。
//
// 纪律：
//   - 全部为状态反应，不消耗 RNG；未触发任何门槛的运行逐位不变。
//   - 只抬成本/收窄路径，不直接判负；所有窗口到期自动解除 (6.3 上界纪律)。
//   - 同源在冷却周期内只生效一次，不叠加 (6.4)。
//   - 新增持久状态零值 = 旧档语义。
//
// 数值为设计给定的起点值，TODO(balance)。

const (
	// T1-3 关系冷却：部门对泄漏事实 B ≥ 0.45 → 关系增长减半 12 月。
	counterRelationThreshold = 0.45 // TODO(balance)
	counterRelationMonths    = 12   // TODO(balance)

	// T2-4 谈判可信度侵蚀：prep_exposure ≥ 0.4 → 保障可信度 −0.15。
	counterGuaranteeThreshold = 0.40  // TODO(balance)
	counterGuaranteeDelta     = -0.15 // TODO(balance)

	// T3-1b 身份线日常绑定：每级节点附带一个月的动作灰置。
	counterIdentityLockMonths = 1 // TODO(balance)

	// T3-2 联系人监视：部门对泄漏事实 B ≥ 0.60 → 关系 −0.10、
	// 接触类动作 3 月不可用、怀疑度 +0.03 (6.2 预算)。
	counterContactThreshold = 0.60 // TODO(balance)
	counterContactMonths    = 3    // TODO(balance)
	counterContactRelation  = 0.10 // TODO(balance)
	counterContactSuspicion = 0.03 // TODO(balance)
)

// 反制信号：服务层据此生成月度故事 (内容层，不指认玩家)。
const (
	CounterSignalContact   = "contact_surveillance"
	CounterSignalGuarantee = "guarantee_erosion"
)

// CountermeasureSignal 一次月度结算产生的、需要叙事的反制事实。
type CountermeasureSignal struct {
	Kind string
	Dept string
}

// CountermeasureTriggers 身份线节点的完成状态 (由服务层从内容账本读取)。
type CountermeasureTriggers struct {
	IdentityQuery bool
	ArchiveReview bool
}

// AdvanceCountermeasures 反制账本的月度回合：先让既有窗口递减一格，再结算
// 本月新触发。返回需要叙事的新信号 (可能为空)。
func AdvanceCountermeasures(silo *model.Silo, agent *model.Agent, triggers CountermeasureTriggers) []CountermeasureSignal {
	if silo == nil {
		return nil
	}
	state := &silo.Countermeasure
	tickCountermeasureWindows(state)

	var signals []CountermeasureSignal

	// T1-3：部门知道了泄漏事实，关系还维持，但推进变慢。
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if departmentLeakBelief(prof) < counterRelationThreshold {
			continue
		}
		if state.RelationCooldown[prof.Name] > 0 {
			continue // 冷却周期内不刷新、不叠加
		}
		if state.RelationCooldown == nil {
			state.RelationCooldown = map[string]int{}
		}
		state.RelationCooldown[prof.Name] = counterRelationMonths
	}

	// T3-2：联系人被调离/问话——同源每部门只发生一次。
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		if departmentLeakBelief(prof) < counterContactThreshold {
			continue
		}
		if _, seen := state.ContactWatch[prof.Name]; seen {
			continue
		}
		if state.ContactWatch == nil {
			state.ContactWatch = map[string]int{}
		}
		state.ContactWatch[prof.Name] = counterContactMonths
		if agent != nil {
			drainAgentConnection(agent, prof.ID, counterContactRelation)
			agent.SuspicionLevel = clampUnit(agent.SuspicionLevel + counterContactSuspicion)
		}
		signals = append(signals, CountermeasureSignal{Kind: CounterSignalContact, Dept: prof.Name})
	}

	// T2-4：筹备活动可见度越线，承诺听上去不再可信 (只侵蚀一次)。
	if !state.GuaranteeEroded && silo.EscapePrep.PrepExposure >= counterGuaranteeThreshold {
		addTopStanceGuarantee(silo, counterGuaranteeDelta)
		state.GuaranteeEroded = true
		signals = append(signals, CountermeasureSignal{Kind: CounterSignalGuarantee})
	}

	// T3-1b：复核期 → 地点类动作下一月灰置；复查期 → 职务动作下一月灰置。
	if triggers.IdentityQuery && !state.IdentityQueryBound {
		state.IdentityQueryBound = true
		state.LocationLockMonths = counterIdentityLockMonths
	}
	if triggers.ArchiveReview && !state.ArchiveReviewBound {
		state.ArchiveReviewBound = true
		state.ProfessionLockMonths = counterIdentityLockMonths
	}

	return signals
}

// tickCountermeasureWindows 每个月初把剩余月数递减一格 (夹到 0)。
func tickCountermeasureWindows(state *model.CountermeasureState) {
	for name, months := range state.RelationCooldown {
		if months > 0 {
			state.RelationCooldown[name] = months - 1
		}
	}
	for name, months := range state.ContactWatch {
		if months > 0 {
			state.ContactWatch[name] = months - 1
		}
	}
	if state.LocationLockMonths > 0 {
		state.LocationLockMonths--
	}
	if state.ProfessionLockMonths > 0 {
		state.ProfessionLockMonths--
	}
}

// departmentLeakBelief 部门信念层中对泄漏事实的最高确信值 (B)。只读
// leak:<fact_id> 观察，避免把部门原有的正当知识误当反制输入。
func departmentLeakBelief(prof *model.Profession) float64 {
	max := 0.0
	for _, belief := range prof.Beliefs {
		if !IsLeakFact(belief.FactID) {
			continue
		}
		if belief.B > max {
			max = belief.B
		}
	}
	return max
}

// drainAgentConnection 关系下降 (夹到 0)；没有该部门的连接时视为 0。
func drainAgentConnection(agent *model.Agent, professionID uint, delta float64) {
	for i := range agent.Connections {
		if agent.Connections[i].ProfessionID != professionID {
			continue
		}
		v := agent.Connections[i].Value - delta
		if v < 0 {
			v = 0
		}
		agent.Connections[i].Value = v
		return
	}
}

// ContactUnderWatch 报告该部门的联系人当前是否不可接触 (T3-2)。
func ContactUnderWatch(silo *model.Silo, dept string) bool {
	if silo == nil || dept == "" {
		return false
	}
	return silo.Countermeasure.ContactWatch[dept] > 0
}

// RelationCooling 报告该部门的关系增长是否仍在 T1-3 冷却期。
func RelationCooling(silo *model.Silo, dept string) bool {
	if silo == nil || dept == "" {
		return false
	}
	return silo.Countermeasure.RelationCooldown[dept] > 0
}

package service

// 指标二：剩余人员平均寿命 (22 设计卷 §4)。属格二实现。
//
// 语义：1 号人员轮流休眠；时间本身必然使平均寿命下降 (什么都不做也保不住)，
// 行动按监控档位额外扣减 (HIGH > NORMAL > REPORT_ONLY)。耗尽导致能力退化
// (档位强制下调、不再签发新通牒)，不是判负。
//
// 正典豁免：18 号链与一切时间戳强制节点不读本指标；寿命耗尽不阻塞正典。
// 本层不写任何正典代码、不被正典读取。

import (
	"fmt"
	"sort"
	"strconv"

	"silo40/internal/model"
)

// 指标二数值 (22 §4.2/§4.3)。全部 TODO(balance)。
const (
	lifespanNaturalDecay      = 0.12  // 每月自然衰减 (年/月)
	postureCostHigh           = 0.10  // HIGH 档月度成本 (年/月)
	postureCostNormal         = 0.04  // NORMAL 档月度成本 (年/月)
	postureCostMin            = 0.005 // REPORT_ONLY 档月度成本 (年/月)
	opCostUltimatum           = 0.3   // 一次性：签发通牒
	opCostPurge               = 0.5   // 一次性：整肃链点火
	opCostTermination         = 0.8   // 一次性：终止链点火
	postureDowngradeThreshold = 3.0   // ≤ 该值：所有 HIGH 档强制降 NORMAL
)

// 台账键 (Ledger 防重复计费)。状态机每局至多签发一枚通牒 (OrderPhase 转
// ULTIMATUM 后不回退)，故 "ultimatum" 用全局键；purge/termination 点火在
// 月度回合里按堡检测，键带 siloID (22 §4.2)。
const (
	lifespanLedgerUltimatum = "ultimatum"
	lifespanLedgerExhausted = "exhausted"
)

// 一次性成本点火事实键 (反应链内容状态，各堡状态图读取)。
const (
	lifespanPurgeIgnitionKey       = "special:silo1_purge_omen"
	lifespanTerminationIgnitionKey = "special:termination_channel_test"
)

// stewardshipLifespanMonth 指标二月度回合 (格二实现)。
// 顺序 (确定性)：自然衰减 → 档位成本 (siloID 升序) → 一次性点火检测与计费
// (siloID 升序) → 耗尽退化。未启用 (零值状态) 早退，行为逐位不变。
func (s *GameService) stewardshipLifespanMonth(logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil {
		return
	}
	st := &s.world.Silo1Stewardship
	if !st.Initialized || !st.Lifespan.Initialized {
		return
	}
	ls := &st.Lifespan

	// 1) 自然衰减：时间本身必然消耗寿命 (不操作也下降)。
	drainLifespan(ls, lifespanNaturalDecay)

	// 2) 档位成本：按 Postures 每堡档位扣减 (升序遍历，浮点路径确定)。
	for _, siloID := range sortedPostureIDs(ls.Postures) {
		drainLifespan(ls, postureMonthlyCost(ls.Postures[siloID]))
	}

	// 3) 一次性行动成本的点火检测：整肃链/终止链首次 Triggered 且台账未计费
	//    → 计费 (通牒由签发路径直接调用 stewardshipChargeLifespan)。
	s.chargeLifespanIgnitions(ls, logs)

	// 4) 耗尽退化 (能力退化，不是判负)。
	degradeLifespanPostures(ls, logs)
}

// stewardshipChargeLifespan 一次性行动成本 (签发通牒/整肃链/终止链点火时由
// 对应路径调用)。kind ∈ {ultimatum, purge, termination}；台账同键只计一次。
// 未启用 (零值状态) 为空操作。
func (s *GameService) stewardshipChargeLifespan(kind string) {
	if s.world == nil {
		return
	}
	st := &s.world.Silo1Stewardship
	if !st.Initialized || !st.Lifespan.Initialized {
		return
	}
	cost, ok := lifespanOpCost(kind)
	if !ok {
		return
	}
	chargeLifespanEntry(&st.Lifespan, kind, cost, nil)
}

// stewardshipLifespanExhausted 寿命是否已耗尽 (读台账 exhausted 标记)。
// 耗尽时反应链入口关闭：不再签发新通牒，已点火阶梯不受影响 (22 §4.3)。
// 未启用 (零值状态) 恒 false——既有行为逐位不变。
func (s *GameService) stewardshipLifespanExhausted() bool {
	if s.world == nil {
		return false
	}
	st := &s.world.Silo1Stewardship
	if !st.Initialized || !st.Lifespan.Initialized {
		return false
	}
	return st.Lifespan.Ledger[lifespanLedgerExhausted] == 1
}

// chargeLifespanIgnitions 扫描各堡内容状态 (siloID 升序)：整肃链
// (special:silo1_purge_omen) / 终止链 (special:termination_channel_test)
// 首次 Triggered 且台账未计费 → 按台账键 (kind:siloID) 计费一次。
func (s *GameService) chargeLifespanIgnitions(ls *model.LifespanStewardship, logs *[]string) {
	for _, siloID := range sortedSiloIDs(s.silos) {
		states := s.statesMapFor(siloID)
		if state, ok := states[lifespanPurgeIgnitionKey]; ok && state.Triggered {
			key := "purge:" + strconv.FormatUint(uint64(siloID), 10)
			chargeLifespanEntry(ls, key, opCostPurge, logs)
		}
		if state, ok := states[lifespanTerminationIgnitionKey]; ok && state.Triggered {
			key := "termination:" + strconv.FormatUint(uint64(siloID), 10)
			chargeLifespanEntry(ls, key, opCostTermination, logs)
		}
	}
}

// chargeLifespanEntry 台账记账：同键只计一次 (防重复计费)，夹下限 ≥0。
func chargeLifespanEntry(ls *model.LifespanStewardship, key string, cost float64, logs *[]string) {
	if ls.Ledger == nil {
		ls.Ledger = map[string]int{}
	}
	if ls.Ledger[key] == 1 {
		return
	}
	ls.Ledger[key] = 1
	drainLifespan(ls, cost)
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf("[Silo1Stewardship] lifespan cost %.3f charged (%s)", cost, key))
	}
}

// degradeLifespanPostures 耗尽退化 (22 §4.3)：
//   - AvgLifespan ≤ 0：夹 0、全部档位强制 REPORT_ONLY、台账记 exhausted=1
//     (此后每月维持强制，防止指派层回升档位)；
//   - AvgLifespan ≤ postureDowngradeThreshold：所有 HIGH 档强制降 NORMAL。
//
// 退化只在状态实际变化时写日志。
func degradeLifespanPostures(ls *model.LifespanStewardship, logs *[]string) {
	if ls.Ledger == nil {
		ls.Ledger = map[string]int{}
	}
	appendLog := func(msg string) {
		if logs != nil {
			*logs = append(*logs, msg)
		}
	}
	exhausted := ls.Ledger[lifespanLedgerExhausted] == 1
	if ls.AvgLifespan <= 0 {
		ls.AvgLifespan = 0
		if !exhausted {
			ls.Ledger[lifespanLedgerExhausted] = 1
			exhausted = true
			appendLog("[Silo1Stewardship] lifespan exhausted: all postures forced to REPORT_ONLY, no new ultimatums")
		}
	}
	if exhausted {
		for siloID, posture := range ls.Postures {
			if posture != model.Silo1PostureReportOnly {
				ls.Postures[siloID] = model.Silo1PostureReportOnly
				appendLog(fmt.Sprintf("[Silo1Stewardship] posture of silo %d forced back to REPORT_ONLY while exhausted", siloID))
			}
		}
		return
	}
	if ls.AvgLifespan <= postureDowngradeThreshold {
		for _, siloID := range sortedPostureIDs(ls.Postures) {
			if ls.Postures[siloID] == model.Silo1PostureHigh {
				ls.Postures[siloID] = model.Silo1PostureNormal
				appendLog(fmt.Sprintf("[Silo1Stewardship] low lifespan: posture of silo %d downgraded HIGH -> NORMAL", siloID))
			}
		}
	}
}

// drainLifespan 扣减寿命并夹下限 ≥0。
func drainLifespan(ls *model.LifespanStewardship, amount float64) {
	ls.AvgLifespan -= amount
	if ls.AvgLifespan < 0 {
		ls.AvgLifespan = 0
	}
}

// postureMonthlyCost 档位月度成本；未知档位不计费。
func postureMonthlyCost(posture string) float64 {
	switch posture {
	case model.Silo1PostureHigh:
		return postureCostHigh
	case model.Silo1PostureNormal:
		return postureCostNormal
	case model.Silo1PostureReportOnly:
		return postureCostMin
	}
	return 0
}

// lifespanOpCost 一次性行动成本；未知类型报 ok=false。
func lifespanOpCost(kind string) (float64, bool) {
	switch kind {
	case "ultimatum":
		return opCostUltimatum, true
	case "purge":
		return opCostPurge, true
	case "termination":
		return opCostTermination, true
	}
	return 0, false
}

// sortedPostureIDs Postures 键升序 (确定性：map 遍历顺序随机)。
func sortedPostureIDs(postures map[uint]string) []uint {
	ids := make([]uint, 0, len(postures))
	for id := range postures {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

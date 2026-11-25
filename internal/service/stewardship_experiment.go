package service

// 指标一：实验履约与保留多样性 (22 设计卷 §3)。属格一实现。
//
// 语义：该按要求清除的叛乱地堡没被清除 → 履约缺口恶化；每清除一个地堡 →
// 多样性损耗恶化 (随时间衰减，越接近实验结束 Y130M1 清除带来的损耗越轻)。
// 该指标只影响"最终可保留地堡的多样性"，不判负、不处刑。
//
// 硬约束 (22 设计卷 §1)：本层不进入任何玩家结局资格/仲裁读取路径，不直接
// 判负、不直接决定处刑；单堡 (空邻堡) 与未启用 (零值) 状态全部早退，状态零值
// 不变。

import (
	"fmt"
	"strings"

	"silo40/internal/model"
)

// 指标一数值参数 (22 设计卷 §3.3)。全部 TODO(balance)，初始值取自设计卷。
const (
	// stewardshipClearPenalty 每清除一个被登记堡带来的多样性损耗基数。
	stewardshipClearPenalty = 0.15 // TODO(balance)
	// stewardshipMissPenalty 一次"该清除而未清除"的履约缺口增量。
	stewardshipMissPenalty = 0.10 // TODO(balance)
	// stewardshipClearDeadlineMonths 清除义务窗口 (通牒 U+0 到打击 U+19 再加缓冲)。
	stewardshipClearDeadlineMonths = 24 // TODO(balance)
	// stewardshipDecayRate 每月两项指标的衰减率。
	stewardshipDecayRate = 0.02 // TODO(balance)
	// stewardshipDiversityWeight / stewardshipFulfillmentWeight 保留多样性读数
	// 对两项指标的扣减权重 (wD / wF)。
	stewardshipDiversityWeight   = 0.5 // TODO(balance)
	stewardshipFulfillmentWeight = 0.5 // TODO(balance)
	// stewardshipEndReferenceYear/Month 实验结束参照 (18 号正典终局信号，
	// 18-chain N9)：Y130M1。它是倒计时锚点，不是"游戏窗口长度"的声明。
	stewardshipEndReferenceYear  = 130
	stewardshipEndReferenceMonth = 1
)

// stewardshipExperimentLedgerInitialCandidates Ledger 键：启用时候选堡总数。
const stewardshipExperimentLedgerInitialCandidates = "initial_candidates"

// stewardshipExperimentMonthIndex 线性月份索引 (year*12+month)，用于登记/清除/
// miss 的月份差计算。与 22 设计卷 §3.3 的 monthsRemaining 口径一致。
func stewardshipExperimentMonthIndex(year, month int) int {
	return year*12 + month
}

// stewardshipExperimentMonthsRemaining 距实验结束参照 (Y130M1) 的剩余月数，
// 负值取 0。
func stewardshipExperimentMonthsRemaining(year, month int) int {
	rem := stewardshipExperimentMonthIndex(stewardshipEndReferenceYear, stewardshipEndReferenceMonth) -
		stewardshipExperimentMonthIndex(year, month)
	if rem < 0 {
		return 0
	}
	return rem
}

// stewardshipExperimentProximity 清除时点距实验结束的接近度：越接近 Y130M1
// 越小 (候选本来就要收拢到一个，此时清除的多样性损耗越轻)。归一跨度取本局
// 实际倒计时 (windowMonths)，跨度未知或已尽 (≤0) 时取 0。
func stewardshipExperimentProximity(year, month, windowMonths int) float64 {
	if windowMonths <= 0 {
		return 0
	}
	return clampUnit01(float64(stewardshipExperimentMonthsRemaining(year, month)) / float64(windowMonths))
}

// stewardshipExperimentIsSilo1 判断某堡是否为 1 号地堡 (全模拟 peer)。
func (s *GameService) stewardshipExperimentIsSilo1(silo *model.Silo) bool {
	silo1 := s.fullSilo1()
	return silo != nil && silo1 != nil && silo.ID == silo1.ID
}

// stewardshipExperimentSiloDestroyed 堡是否覆灭 (DEATH，大小写不敏感)。
func stewardshipExperimentSiloDestroyed(silo *model.Silo) bool {
	return silo != nil && silo.VictoryStatus != nil && strings.EqualFold(silo.VictoryStatus.Type, "death")
}

// stewardshipExperimentEnsureMaps 防御性初始化 nil map (存档 omitempty 回读后
// 空表为 nil)。
func stewardshipExperimentEnsureMaps(exp *model.ExperimentStewardship) {
	if exp.Scheduled == nil {
		exp.Scheduled = map[uint]int{}
	}
	if exp.Cleared == nil {
		exp.Cleared = map[uint]int{}
	}
	if exp.Missed == nil {
		exp.Missed = map[uint]bool{}
	}
	if exp.Ledger == nil {
		exp.Ledger = map[string]int{}
	}
}

// stewardshipExperimentEnsureInitialCandidates 惰性写入启用时候选堡总数
// (非 1 号堡数)。调度器的 initSilo1Stewardship 只建空表，本层首次使用时落数；
// 堡列表在开局后固定，故何时落数数值都等于启用时口径。
func (s *GameService) stewardshipExperimentEnsureInitialCandidates(exp *model.ExperimentStewardship) {
	stewardshipExperimentEnsureMaps(exp)
	if exp.Ledger[stewardshipExperimentLedgerInitialCandidates] > 0 {
		return
	}
	count := 0
	for _, silo := range s.silos {
		if silo == nil || s.stewardshipExperimentIsSilo1(silo) {
			continue
		}
		count++
	}
	exp.Ledger[stewardshipExperimentLedgerInitialCandidates] = count
}

// stewardshipExperimentEnsureWindowMonths 惰性落数 proximity 归一跨度：权衡层
// 首次运行时取当月 (开局月) 距 Y130M1 参照月的剩余月数——Y122M1 开局即 96、
// StartYear=120 开局即 120，跨度由本局运行时日历决定，不再有写死的"总窗口"常数。
// 无跨度的旧档回读时以登记簿里最早的登记月兜底 (跨度不因续玩被压缩)，落数后
// 不再改动，读档复现。参照月已过 (跨度 ≤0) 时不落数，接近度按 0 处理。
func (s *GameService) stewardshipExperimentEnsureWindowMonths(exp *model.ExperimentStewardship) {
	stewardshipExperimentEnsureMaps(exp)
	if exp.WindowMonths > 0 {
		return
	}
	end := stewardshipExperimentMonthIndex(stewardshipEndReferenceYear, stewardshipEndReferenceMonth)
	span := end - stewardshipExperimentMonthIndex(s.world.CurrentYear, s.world.CurrentMonth)
	earliest := 0
	for _, registered := range exp.Scheduled {
		if earliest == 0 || registered < earliest {
			earliest = registered
		}
	}
	if earliest > 0 && end-earliest > span {
		span = end - earliest
	}
	if span > 0 {
		exp.WindowMonths = span
	}
}

// stewardshipExperimentMonth 指标一月度回合 (格一实现)。
// 职责：清除成立检测 (DEATH)、miss 结算、双指标衰减。按 SiloID 升序扫描
// (确定性，22 设计卷 §3.2)。未启用 (零值) 早退，状态零值不变。
func (s *GameService) stewardshipExperimentMonth(logs *[]string, stories *[]model.StoryEvent) {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	s.stewardshipExperimentEnsureInitialCandidates(exp)
	s.stewardshipExperimentEnsureWindowMonths(exp)

	now := stewardshipExperimentMonthIndex(s.world.CurrentYear, s.world.CurrentMonth)

	// 清除成立检测 + miss 结算：仅针对已登记堡，按 SiloID 升序。
	for _, silo := range s.silos {
		if silo == nil {
			continue
		}
		registeredMonth, registered := exp.Scheduled[silo.ID]
		if !registered {
			continue
		}
		if _, cleared := exp.Cleared[silo.ID]; cleared {
			continue
		}
		// 清除成立：被登记堡覆灭 (任意来路都算候选减少)。
		if stewardshipExperimentSiloDestroyed(silo) {
			exp.Cleared[silo.ID] = now
			proximity := stewardshipExperimentProximity(s.world.CurrentYear, s.world.CurrentMonth, exp.WindowMonths)
			loss := stewardshipClearPenalty * proximity
			exp.DiversityLoss += loss
			if logs != nil {
				*logs = append(*logs, fmt.Sprintf("[Silo1Stewardship] silo %d cleared, diversity loss +%.4f (proximity %.3f)", silo.ID, loss, proximity))
			}
			continue
		}
		// miss 结算：登记超过窗口仍未覆灭，每个登记只结算一次。
		if exp.Missed[silo.ID] || now-registeredMonth < stewardshipClearDeadlineMonths {
			continue
		}
		exp.FulfillmentDeficit += stewardshipMissPenalty
		exp.Missed[silo.ID] = true
		exp.Misses = append(exp.Misses, model.ExperimentMissRecord{
			SiloID: silo.ID,
			Year:   s.world.CurrentYear,
			Month:  s.world.CurrentMonth,
		})
		if logs != nil {
			*logs = append(*logs, fmt.Sprintf("[Silo1Stewardship] silo %d clearing missed, fulfillment deficit +%.4f", silo.ID, stewardshipMissPenalty))
		}
	}

	// 每月衰减：两项指标各乘 (1 − decayRate)。
	exp.FulfillmentDeficit *= 1 - stewardshipDecayRate
	exp.DiversityLoss *= 1 - stewardshipDecayRate
}

// stewardshipRegisterClearingObligation 通牒签发时登记清除义务 (由反应链签发
// 路径调用；格一实现登记逻辑)。仅当权衡层已启用时生效；重复签发不重复登记。
func (s *GameService) stewardshipRegisterClearingObligation(siloID uint) {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	if _, ok := exp.Scheduled[siloID]; ok {
		return
	}
	exp.Scheduled[siloID] = stewardshipExperimentMonthIndex(s.world.CurrentYear, s.world.CurrentMonth)
}

// stewardshipRetentionDiversity 保留多样性读数 (供格三决策因子与呈现)：
// clamp01(candidatesAlive/initialCandidates − wD·DiversityLoss − wF·FulfillmentDeficit)。
// 未启用 / 无候选时返回 1 (满多样性，中性安全值)。本读数不改写任何状态。
func (s *GameService) stewardshipRetentionDiversity() float64 {
	if s.world == nil || !s.world.Silo1Stewardship.Initialized {
		return 1
	}
	exp := &s.world.Silo1Stewardship.Experiment
	stewardshipExperimentEnsureMaps(exp)
	s.stewardshipExperimentEnsureInitialCandidates(exp)
	initial := exp.Ledger[stewardshipExperimentLedgerInitialCandidates]
	if initial <= 0 {
		return 1
	}
	alive := 0
	for _, silo := range s.silos {
		if silo == nil || s.stewardshipExperimentIsSilo1(silo) || stewardshipExperimentSiloDestroyed(silo) {
			continue
		}
		alive++
	}
	ratio := float64(alive) / float64(initial)
	return clampUnit01(ratio - stewardshipDiversityWeight*exp.DiversityLoss - stewardshipFulfillmentWeight*exp.FulfillmentDeficit)
}

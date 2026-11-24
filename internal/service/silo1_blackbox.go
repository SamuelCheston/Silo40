package service

// 1 号黑箱回合 (22 黑箱学习系列 01 卷 §2.2/§5/§6/§7.1)。阶段 2b 既有内容
// (runSilo1StateMachine) 之后追加：按 SiloID 升序逐堡抽干到达观测 → N 箱信念
// 更新 → 摘要 → 零 RNG 期望效用决策 → 指派 r。
//
// 黑箱壁纪律 (验收级，AST 守卫测试锁定)：
//   - 本文件决策输入只有 Belief、PendingObservations 与 1 号内部参数；禁止引用
//     BlackBox.Theta 真值、UpdateBlackBoxMonthly、ExecutionIntensity/
//     CurrentExecution 及一切物理层曲线标识符 (不 import engine 包)。
//   - 信念更新与决策全程零 RNG (01 卷 §7.3)；唯一 RNG 消耗点是观测产生
//     (4.65，engine 包，不归本文件)。
//   - 决策日志只进存档审计 (BlackBox.Log 环形上限 24)，不进故事事件、不进
//     玩家视图 (00 卷裁决 12)。

import (
	"fmt"
	"math"
	"sort"

	"silo40/internal/model"
)

// 1 号内部参数 (信念/决策，与 01 §9 一致) 集中在 silo1_blackbox_constants.go；
// 黑箱壁禁止 import engine，与世界侧参数表 engine/blackbox_constants.go
// 保持独立同值基线 (01 卷 §6.2)。

// blackBoxMenu 决策菜单 R = {0.00, 0.05, ..., 0.95} 升序 (01 卷 §6.1)。
// 上界 0.95 < 1.0：决策层不自愿触碰 DEATH 线 (SafeguardRisk ≥ 1.0 是内容事件
// 推动的硬失败渠道，不是最优解候选)。用 k/20 而非 k*0.05 构造：除法保证端点
// 恰为字面 0.95 (乘法有 ULP 漂移)。
var blackBoxMenu = func() []float64 {
	menu := make([]float64, blackBoxMenuPoints)
	for k := range menu {
		menu[k] = float64(k) / float64(blackBoxMenuPoints)
	}
	return menu
}()

// ===== 探索接口 (01 卷 §6.4，留给轨道 C) =====

// BlackBoxProbeContext 传给探索提议者的决策时刻上下文。
type BlackBoxProbeContext struct {
	Year     int
	Month    int
	ThetaHat float64
	Lo       float64
	Hi       float64
	PDanger  float64
}

// BlackBoxProbeProposer 探索提议接口 (01 卷 §6.4 签名)。ok=true 时以 rProbe
// 覆盖 argmax 结果 (00 卷裁决 10：探索建议优先于利用)；实现必须是纯确定性的，
// 或经引擎唯一 RNG 流、按 SiloID 序消耗 (禁止私有随机源)。
type BlackBoxProbeProposer interface {
	ProbeProposer(silo *model.Silo, belief []float64, ctx BlackBoxProbeContext) (rProbe float64, ok bool)
}

// blackBoxNoProbe 本轮默认探索提议者：不探索 (恒 ok=false)。
type blackBoxNoProbe struct{}

func (blackBoxNoProbe) ProbeProposer(*model.Silo, []float64, BlackBoxProbeContext) (float64, bool) {
	return 0, false
}

// silo1BlackBoxProber 现行探索提议者 (轨道 C 替换；默认不探索)。
var silo1BlackBoxProber BlackBoxProbeProposer = blackBoxNoProbe{}

// ===== 回合入口 =====

// runSilo1BlackBoxRound 阶段 2b 黑箱回合。功能关闭或 1 号覆灭 → 直接返回
// (逐位不变)；专线干扰 → 早退并丢弃在途观测 (00 卷裁决 2)。
// 不写 logs/stories：决策留痕只进 BlackBox.Log 存档审计 (00 卷裁决 12)。
func (s *GameService) runSilo1BlackBoxRound(logs *[]string, stories *[]model.StoryEvent) {
	if !s.silo1BlackBoxOn() || !s.world.Silo1.Alive {
		return
	}
	if s.world.Flags[model.WorldFlagNetworkJammed] {
		// 专线干扰：不更新信念、不签发新指派 (LastAssignedRisk 沿用，断联期间
		// 最后一条指令继续生效)；产生与在途的观测丢弃 (断联 = 信息丢失)。
		for _, silo := range s.silos {
			if siloEnded(silo) {
				continue
			}
			silo.BlackBox.PendingObservations = nil
		}
		return
	}
	year, month := s.world.CurrentYear, s.world.CurrentMonth
	for _, silo := range s.silos {
		if siloEnded(silo) || !silo.BlackBox.Initialized {
			continue
		}
		s.runSilo1BlackBoxSiloMonth(silo, year, month)
	}
}

// runSilo1BlackBoxSiloMonth 单堡黑箱月 (01 卷 §7.1)：抽干到达观测 → 信念更新
// → 摘要 → 零 RNG 决策 → 写回与审计日志。
func (s *GameService) runSilo1BlackBoxSiloMonth(silo *model.Silo, year, month int) {
	bb := &silo.BlackBox
	if bb.Belief == nil {
		bb.Belief = blackBoxPrior()
	}

	// (1) 观测到达：抽干到达月 ≤ 当月的条目，按 (到达月, 信号类型, seq) 升序。
	arrived := blackBoxDrainArrived(&bb.PendingObservations, year, month)
	for _, o := range arrived {
		blackBoxUpdateBelief(bb.Belief, o)
	}

	// (2) 信念摘要。
	thetaHat, pDanger, lo, hi := blackBoxSummarize(bb.Belief)

	// (3) 决策 (零 RNG)：冷却 >0 递减；归零后探索提议者可覆盖 argmax
	// (00 卷裁决 10)；本轮默认提议者恒 ok=false。
	ctx := BlackBoxProbeContext{Year: year, Month: month, ThetaHat: thetaHat, Lo: lo, Hi: hi, PDanger: pDanger}
	var chosenR, utility float64
	reason := "exploit"
	if bb.ProbeCooldown > 0 {
		bb.ProbeCooldown--
		chosenR, utility = blackBoxArgmax(bb.Belief)
	} else if rProbe, ok := silo1BlackBoxProber.ProbeProposer(silo, bb.Belief, ctx); ok {
		chosenR, utility, reason = rProbe, blackBoxUtility(bb.Belief, rProbe), "explore"
		bb.ProbeCooldown = blackBoxProbeCooldownStart // 使用后设置 (01 卷 §6.4)
	} else {
		chosenR, utility = blackBoxArgmax(bb.Belief)
	}

	// (4) 写回：基线 r (内容事件继续按既有增量通道在基线上叠加)。
	silo.SafeguardRisk = chosenR
	bb.LastAssignedRisk = chosenR

	// (5) 审计日志：只进存档，不进故事事件、不进玩家视图 (00 卷裁决 12)。
	bb.Log = append(bb.Log, model.BlackBoxLogEntry{
		Year:     year,
		Month:    month,
		ThetaHat: thetaHat,
		Lo:       lo,
		Hi:       hi,
		PDanger:  pDanger,
		ChosenR:  chosenR,
		Utility:  utility,
		Reason:   reason,
		Factors:  blackBoxFactorSummary(bb.Log, arrived, thetaHat, year, month),
	})
	if len(bb.Log) > blackBoxLogCap {
		bb.Log = bb.Log[len(bb.Log)-blackBoxLogCap:]
	}
}

// ===== 观测抽干 =====

// blackBoxDrainArrived 从 pending 抽干到达月 ≤ 当月的观测，按
// (到达月, 信号类型, seq) 升序返回 (01 卷 §7.1)；其余保持原顺序留待下月。
func blackBoxDrainArrived(pending *[]model.BlackBoxPendingObservation, year, month int) []model.BlackBoxPendingObservation {
	var arrived, remaining []model.BlackBoxPendingObservation
	for _, o := range *pending {
		if o.ArriveYear < year || (o.ArriveYear == year && o.ArriveMonth <= month) {
			arrived = append(arrived, o)
		} else {
			remaining = append(remaining, o)
		}
	}
	sort.Slice(arrived, func(i, j int) bool {
		a, b := arrived[i], arrived[j]
		if a.ArriveYear != b.ArriveYear {
			return a.ArriveYear < b.ArriveYear
		}
		if a.ArriveMonth != b.ArriveMonth {
			return a.ArriveMonth < b.ArriveMonth
		}
		if a.SignalType != b.SignalType {
			return a.SignalType < b.SignalType
		}
		return a.Seq < b.Seq
	})
	*pending = remaining
	return arrived
}

// ===== 信念：先验、更新、摘要 =====

// blackBoxPrior 先验信念 (01 卷 §5.1)：b_0(i) ∝ (1−ε)·exp(−λ·θ_i) + ε。
// 低箱重 (表面服从是常态) + 均匀地板 (任何箱永不归零锁死)。
func blackBoxPrior() []float64 {
	belief := make([]float64, blackBoxBins)
	sum := 0.0
	for i := range belief {
		v := (1-blackBoxPriorFloor)*math.Exp(-blackBoxPriorLambda*blackBoxBinCenter(i)) + blackBoxPriorFloor
		belief[i] = v
		sum += v
	}
	for i := range belief {
		belief[i] /= sum
	}
	return belief
}

// blackBoxBinCenter 箱中心 θ_i = (i+0.5)/N。
func blackBoxBinCenter(i int) float64 {
	return (float64(i) + 0.5) / float64(blackBoxBins)
}

// blackBoxUpdateBelief 单条观测的后验更新 (01 卷 §5.3 噪声膨胀归因 + §5.4
// 伪装混合似然)：σ_eff² = σ_k² + (ρ̂·δ_k)²；
// L_i = (1−π̂(θ_i))·φ(y;θ_i,σ_eff) + π̂(θ_i)·φ(y;θ_i−d_k,σ_eff)；
// b ← normalize(b⊙L)。零 RNG。
func blackBoxUpdateBelief(belief []float64, o model.BlackBoxPendingObservation) {
	params, ok := blackBoxSignalTable[o.SignalType]
	if !ok {
		return // 未知信号类型：不参与信念更新
	}
	drift := blackBoxDriftHat * float64(params.delay)
	sigmaEff := math.Sqrt(params.noise*params.noise + drift*drift)
	sum := 0.0
	for i := range belief {
		theta := blackBoxBinCenter(i)
		pi := blackBoxDisguiseRate(theta)
		likelihood := (1-pi)*blackBoxGaussPDF(o.Value, theta, sigmaEff) +
			pi*blackBoxGaussPDF(o.Value, theta-params.disguiseShift, sigmaEff)
		belief[i] *= likelihood
		sum += belief[i]
	}
	if sum <= 0 {
		return // ε 地板下实际不可达的数值退化：保留原信念
	}
	for i := range belief {
		belief[i] /= sum
	}
}

// blackBoxDisguiseRate 1 号内部伪装率假设 π̂(θ) = clamp01((θ−0.30)/0.40)
// (与产生侧同形式，但这是 1 号自己的模型，非读取世界参数；01 卷 §5.4)。
func blackBoxDisguiseRate(theta float64) float64 {
	return clampUnit01((theta - blackBoxDisguiseOnset) / blackBoxDisguiseSpan)
}

// blackBoxGaussPDF 高斯密度 φ(y; mean, sigma)。
func blackBoxGaussPDF(y, mean, sigma float64) float64 {
	z := (y - mean) / sigma
	return math.Exp(-0.5*z*z) / (sigma * math.Sqrt(2*math.Pi))
}

// blackBoxSummarize 信念摘要 (01 卷 §5.5)：后验均值 Θ̂、危险区概率
// P_danger = Σ_{θ_i ≥ θ_danger} b_i、包含 90% 质量的最短箱区间。
func blackBoxSummarize(belief []float64) (thetaHat, pDanger, lo, hi float64) {
	for i, p := range belief {
		theta := blackBoxBinCenter(i)
		thetaHat += p * theta
		if theta >= blackBoxDangerTheta {
			pDanger += p
		}
	}
	lo, hi = blackBoxCredibleInterval(belief)
	return thetaHat, pDanger, lo, hi
}

// blackBoxCredibleInterval 包含 90% 质量的最短箱区间 (升序扫描；同长保留更小
// 下界，与「只有严格更高分才替换」同口径)。返回区间端点 (箱边界)。
func blackBoxCredibleInterval(belief []float64) (float64, float64) {
	n := len(belief)
	bestLen := n + 1
	bestI, bestJ := 0, n-1
	for i := 0; i < n; i++ {
		mass := 0.0
		for j := i; j < n; j++ {
			mass += belief[j]
			if mass >= blackBoxCredibleMass {
				if j-i+1 < bestLen {
					bestLen = j - i + 1
					bestI, bestJ = i, j
				}
				break
			}
		}
	}
	return float64(bestI) / float64(n), float64(bestJ+1) / float64(n)
}

// ===== 决策：期望效用 argmax =====

// blackBoxArgmax 决策菜单上的期望效用 argmax (01 卷 §6.3)：
// U(r) = Σ_i b_i·Ê(r;θ_i) − κ·r。升序扫描、只有严格更高分才替换 (平局保留
// 更小 r)。零 RNG。
func blackBoxArgmax(belief []float64) (float64, float64) {
	return blackBoxMenuArgmax(func(r float64) float64 {
		return blackBoxUtility(belief, r)
	})
}

// blackBoxMenuArgmax 菜单升序扫描；只有严格更高分才替换当前选择——平局保留
// 更小的 r (knowledge/08 §2 同口径)。
func blackBoxMenuArgmax(score func(float64) float64) (float64, float64) {
	best := math.Inf(-1)
	chosen := blackBoxMenu[0]
	for _, r := range blackBoxMenu {
		if u := score(r); u > best {
			best, chosen = u, r
		}
	}
	return chosen, best
}

// blackBoxUtility U(r) = Σ_i b_i·Ê(r;θ_i) − κ·r (期望执行强度 − 压力代价)。
func blackBoxUtility(belief []float64, r float64) float64 {
	expected := 0.0
	for i, p := range belief {
		expected += p * blackBoxInternalExecution(r, blackBoxBinCenter(i))
	}
	return expected - blackBoxPressureCost*r
}

// blackBoxInternalExecution Ê(r;θ)：1 号自己的高斯内部执行模型 (01 卷 §6.2)。
// 峰位 r̂*(θ) = r_0 − k_shift·θ、峰高 Ê_max(θ) = e_0 − (e_0−e_1)·θ、
// 峰宽 σ̂(θ) = σ_0·(1 − k_width·θ)。
func blackBoxInternalExecution(r, theta float64) float64 {
	peakR := blackBoxInternalPeakR - blackBoxInternalPeakShift*theta
	peakE := blackBoxInternalPeakHigh - (blackBoxInternalPeakHigh-blackBoxInternalPeakLow)*theta
	width := blackBoxInternalPeakWidth * (1 - blackBoxInternalWidthShrink*theta)
	d := r - peakR
	return peakE * math.Exp(-d*d/(2*width*width))
}

// ===== 因子摘要 =====

// blackBoxFactorSummary 因子读数摘要 (04 卷 §2.2)：观测条数、最大延迟月数、
// 伪装嫌疑 π̂(Θ̂)、距上次指派月数 (审计日志推导；首次指派为 0)。
func blackBoxFactorSummary(log []model.BlackBoxLogEntry, arrived []model.BlackBoxPendingObservation, thetaHat float64, year, month int) string {
	maxDelay := 0
	for _, o := range arrived {
		delay := (o.ArriveYear*12 + o.ArriveMonth) - (o.ReflectedYear*12 + o.ReflectedMonth)
		if delay > maxDelay {
			maxDelay = delay
		}
	}
	gap := 0
	if n := len(log); n > 0 {
		last := log[n-1]
		gap = (year*12 + month) - (last.Year*12 + last.Month)
	}
	return fmt.Sprintf("观测 %d 条、延迟 %d 月、伪装嫌疑 %.2f、距上次指派 %d 月",
		len(arrived), maxDelay, blackBoxDisguiseRate(thetaHat), gap)
}

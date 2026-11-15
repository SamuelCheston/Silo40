package engine

// 黑箱观测产生 (22 黑箱学习系列 01 卷 §2.2/§5.2/§7.1)。属观测核心格实现。
//
// RNG 纪律 (22-01 §7.3、00 卷裁决 3)：唯一消耗点。每产生信号恰好 3 次
// DrawFloat64——1 次伪装判定 + 2 次噪声取样 (Box-Muller)；堡按 SiloID 升序
// (调用方保证)、信号按类型升序固定取样；功能关闭或信号当月不产生 → 零消耗，
// 与既有 parity 基线逐位一致。
//
// 结构不变性 (23-05 §3.3)：盲区经营注入只改三个标量参数
// (p_disg_eff/d_k_eff/σ_k_eff)，DrawFloat64 调用次数/顺序与队列结构逐位不变。
//
// 代理口径 (22-02 §6 缺口清单短期替代)：三信号载体侧的代理读数定义在
// blackBoxSignalBase；除物理层 Θ 外不引用任何黑箱真值 (Θ' 是 4.65 的合法输入，
// 22-01 §5.2)。

import (
	"math"

	"silo40/internal/model"
)

// 22 基线信号参数 (22-01 §5.2)：延迟 δ_k (月)、噪声 σ_k、伪装偏移 d_k。
// 均 TODO(balance)。
const (
	bbReceiptDelayDelayMonths = 1 // δ receipt_delay：专线例行流量
	bbExecutionGapDelayMonths = 2 // δ execution_gap：复核耗时
	bbLineToneDelayMonths     = 2 // δ line_tone：例行汇报周期
	bbReceiptDelaySigma       = 0.10
	bbExecutionGapSigma       = 0.12
	bbLineToneSigma           = 0.15
	bbReceiptDelayDisguise    = 0.05
	bbExecutionGapDisguise    = 0.15
	bbLineToneDisguise        = 0.25
	bbDisguiseProbThetaLo     = 0.30 // p_disg(Θ') = clamp01((Θ'−0.30)/0.40)
	bbDisguiseProbThetaSpan   = 0.40 // 戒心过半才开始系统性粉饰太平
	bbReceiptDelayBaseline    = 0.10 // receipt_delay 常量基线 (见 blackBoxSignalBase)
)

// bbSignalSpec 单个观测信号的产生参数。
type bbSignalSpec struct {
	signalType  string
	delayMonths int
	sigma       float64                 // σ_k 基线噪声
	disguise    float64                 // d_k 基线伪装偏移
	dCap        float64                 // d_k^cap 注入上限 (23-05 §3.1)
	weights     [bbChannelCount]float64 // 通道权重 w_{k,c} (23-05 §3.2)
}

// bbSignalSpecs 固定产生顺序：22-01 §5.2 表序 / §7.1 伪代码
// 「for k in [receipt_delay, execution_gap, line_tone] 升序」。顺序写死为切片，
// 不依赖 map 遍历 (确定性纪律)。
var bbSignalSpecs = [...]bbSignalSpec{
	{model.BlackBoxSignalReceiptDelay, bbReceiptDelayDelayMonths, bbReceiptDelaySigma, bbReceiptDelayDisguise, bbInjectDCapReceiptDelay, bbWeightsReceiptDelay},
	{model.BlackBoxSignalExecutionGap, bbExecutionGapDelayMonths, bbExecutionGapSigma, bbExecutionGapDisguise, bbInjectDCapExecutionGap, bbWeightsExecutionGap},
	{model.BlackBoxSignalLineTone, bbLineToneDelayMonths, bbLineToneSigma, bbLineToneDisguise, bbInjectDCapLineTone, bbWeightsLineTone},
}

// GenerateBlackBoxObservations RunLocalPhase 步骤 4.65：反映结算后的 Θ'，
// 按信号类型升序产生观测并按延迟 δ_k 入队 PendingObservations。
// 门控：BlackBox.Initialized 为 false (功能关闭/旧档/单堡) 时直接返回，零 RNG。
//
// network_jammed (00 卷裁决 2：干扰期观测丢弃) 是世界级事实；本函数签名只有堡级
// 输入、读不到世界 flag，丢弃由服务层黑箱回合抽干侧处理，此处不重复判定。
func GenerateBlackBoxObservations(silo *model.Silo) {
	if silo == nil || !silo.BlackBox.Initialized {
		return
	}
	theta := silo.BlackBox.Theta // Θ' = 4.55 结算后的真值 (4.65 合法输入，22-01 §5.2)
	now := GameTime{Year: silo.CurrentYear, Month: silo.CurrentMonth}
	for _, spec := range bbSignalSpecs {
		base, produced := blackBoxSignalBase(silo, spec.signalType)
		if !produced {
			continue // 当月不产生 → 零 RNG (22-01 §7.3 双重门控)
		}
		pDisgEff, dEff, sigmaEff := blackboxInjectionParams(silo, spec, theta)
		disguised := DrawFloat64() < pDisgEff // RNG 消耗 1：伪装判定
		bias := 0.0
		if disguised {
			bias = -dEff // 把报告粉饰得更平静 (22-01 §5.2)
		}
		eta := boxMuller(DrawFloat64(), DrawFloat64()) // RNG 消耗 2+3：噪声取样
		y := clamp01(base + bias + sigmaEff*eta)
		arrive := AdvanceTime(now, spec.delayMonths)
		silo.BlackBox.PendingObservations = append(silo.BlackBox.PendingObservations, model.BlackBoxPendingObservation{
			ArriveYear:     arrive.Year,
			ArriveMonth:    arrive.Month,
			ReflectedYear:  now.Year,
			ReflectedMonth: now.Month,
			SignalType:     spec.signalType,
			Value:          y,
			Seq:            silo.BlackBox.ObsSeq,
		})
		silo.BlackBox.ObsSeq++
	}
}

// blackBoxSignalBase 信号 k 的载体侧代理读数 (base) 与当月是否产生
// (22-02 §6 缺口清单的短期替代口径)。除物理层 Θ 外不引用任何黑箱真值；
// 代理只读堡面既有量 (OrderPressure、部门恐慌) 与常量。
//
// W2 口径：三信号均每月产生。
//
//   - receipt_delay：回执常态 → 每月产生。载体 (Order 例行指令账本) 缺失
//     (22-02 §6 G3)：读数为常量基线 + 噪声——G3 落地前为基线代理。
//     TODO(balance)：G3 落地后改读真实回执时延归一化。
//   - execution_gap：代理 = OrderPressure 的归一化读数 (22-02 §6 G3 短期替代：
//     以 OrderPressure 反向变化作执行质量代理)。信号极性是「差异/坏度」
//     (越高越坏)，故直接取 OrderPressure 本身 (叛乱与平均恐慌的加权和，
//     已在 [0,1])：压力越高 = 执行缺口越大。首版以当月水位代替「变化」
//     (无上月水位存档字段)。TODO(balance)：G3 落地后改读真实复核差异。
//   - line_tone：代理 = IT 部门恐慌的归一化读数——IT 主任部门恐慌越高，
//     例行汇报的态度信号越差。TODO(balance)：例行汇报载体落地后改读真实
//     态度观测。
func blackBoxSignalBase(silo *model.Silo, signalType string) (base float64, produced bool) {
	switch signalType {
	case model.BlackBoxSignalReceiptDelay:
		return bbReceiptDelayBaseline, true
	case model.BlackBoxSignalExecutionGap:
		return blackBoxExecutionGapProxy(silo), true
	case model.BlackBoxSignalLineTone:
		return blackBoxLineToneProxy(silo), true
	}
	return 0, false
}

// blackBoxExecutionGapProxy execution_gap 载体代理：OrderPressure 归一化读数。
// 只读堡面既有量，不引用黑箱真值。
func blackBoxExecutionGapProxy(silo *model.Silo) float64 {
	return clamp01(OrderPressure(silo))
}

// blackBoxLineToneProxy line_tone 载体代理：IT 部门恐慌归一化读数。
// 无 IT 部门时退化为全部门恐慌均值 (切片序累加，确定性)；无部门时为 0。
func blackBoxLineToneProxy(silo *model.Silo) float64 {
	for _, profession := range silo.Professions {
		if profession.Name == "IT" {
			return clamp01(profession.PanicValue)
		}
	}
	if len(silo.Professions) == 0 {
		return 0
	}
	total := 0.0
	for _, profession := range silo.Professions {
		total += profession.PanicValue
	}
	return clamp01(total / float64(len(silo.Professions)))
}

// blackboxInjectionParams 23-05 §3.1 注入映射 (零 RNG，确定性)：由堡盲区经营
// 状态计算信号 k 的产生侧有效参数 (p_disg_eff, d_k_eff, σ_k_eff)。
//
//	p_disg_eff(Θ') = clamp01( p_disg(Θ') · (1 + b_p·ῑ_k) )
//	d_k_eff        = min( d_k + b_d·ῑ_k , d_k^cap )
//	σ_k_eff        = min( σ_k·(1 + b_σ·Σ_c w_{k,c}·Cov_c) , σ_k·(1+s_max) )
//	ῑ_k            = Σ_c w_{k,c}·Cov_c·Bias_c        // 固定通道序累加
//
// 门控 (23-05 §8.2)：BlindSpot 未启用 (含旧档零值) → 22 基线参数 (恒等)；
// 已启用但全零时各式自然退化为恒等。只改三个标量参数，DrawFloat64 调用
// 次数/顺序逐位不变 (§3.3 结构不变性)。
func blackboxInjectionParams(silo *model.Silo, spec bbSignalSpec, theta float64) (pDisgEff, dEff, sigmaEff float64) {
	pDisg := blackBoxDisguiseProb(theta)
	blind := &silo.BlindSpot
	if !blind.Initialized {
		return pDisg, spec.disguise, spec.sigma
	}
	// 固定通道序累加 (equipment→schedule→reports→ledger→census→timing)，
	// 禁止 map 序 (23-05 §2.1 纪律 4)。
	iotaBar := 0.0 // ῑ_k：有效经营量
	covBar := 0.0  // Σ w_{k,c}·Cov_c：噪声膨胀只挂 Cov_c (§2.1 纪律 6)
	for c := 0; c < bbChannelCount; c++ {
		w := spec.weights[c]
		if w == 0 {
			continue
		}
		iotaBar += w * blind.Coverage[c] * blind.Bias[c]
		covBar += w * blind.Coverage[c]
	}
	pDisgEff = clamp01(pDisg * (1 + bbInjectBP*iotaBar))
	dEff = math.Min(spec.disguise+bbInjectBD*iotaBar, spec.dCap)
	sigmaEff = math.Min(spec.sigma*(1+bbInjectBSigma*covBar), spec.sigma*(1+bbInjectSMax))
	return pDisgEff, dEff, sigmaEff
}

// blackBoxDisguiseProb 22 基线伪装概率 p_disg(Θ') = clamp01((Θ'−0.30)/0.40)
// (22-01 §5.2)。TODO(balance)。
func blackBoxDisguiseProb(theta float64) float64 {
	return clamp01((theta - bbDisguiseProbThetaLo) / bbDisguiseProbThetaSpan)
}

// boxMuller Box-Muller 变换：两个 [0,1) 均匀样本 → 一个标准正态样本。
// 取 1−u1 使 log 宗量落在 (0,1]，避免 log(0)。零 RNG：只消费调用方传入的
// 两次 DrawFloat64 结果，不新增抽样。
func boxMuller(u1, u2 float64) float64 {
	return math.Sqrt(-2*math.Log(1-u1)) * math.Cos(2*math.Pi*u2)
}

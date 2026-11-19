package engine

// 黑箱物理层 (22 黑箱学习系列 01 卷 §3/§4)：Θ 动力学与执行强度单峰 E(r, Θ)。
//
// 黑箱壁纪律：本文件的 Θ 真值、E 曲线参数只允许被本地堡机制 (4.55 结算、
// 4.6 消费、4.65 观测产生) 读取；服务包的黑箱回合 (信念与决策) 禁止引用本
// 文件的任何参数与真值标识符 (AST 守卫测试锁定)。

import (
	"math"

	"silo40/internal/model"
)

// UpdateBlackBoxMonthly RunLocalPhase 步骤 4.55 (22 黑箱学习系列 01 卷 §2.2)：
// 读上月指派 LastAssignedRisk，结算 Θ(t+1) = clamp01(Θ + growth(r) − decay(Θ, r))，
// 计算 E(r, Θ(t+1)) 写入 CurrentExecution 生效槽。世界侧簿记，零 RNG。
// 门控：BlackBox.Initialized 为 false (功能关闭/旧档/单堡) 时直接返回，
// 状态逐位不变。
func UpdateBlackBoxMonthly(silo *model.Silo) {
	if silo == nil || !silo.BlackBox.Initialized {
		return
	}
	bb := &silo.BlackBox

	// 本月生效 r：上月 2b 指派；零值 = 首次启用月，以当前 SafeguardRisk 为
	// 生效 r (连续性，01 卷 §8)。
	rEff := bb.LastAssignedRisk
	if rEff == 0 {
		rEff = silo.SafeguardRisk
	}

	// Θ 结算 (01 卷 §3.2 候选 A 分段线性加性)：
	// x = 超压归一化；growth = g_max·x；decay = Θ·(1/τ)·(1−x) (随压力抑制)。
	x := clamp01((rEff - BlackBoxRCalm) / (BlackBoxRFull - BlackBoxRCalm))
	theta := clamp01(bb.Theta + BlackBoxGrowthMax*x - bb.Theta*(1/BlackBoxTauDecay)*(1-x))

	// E 生效 (01 卷 §4.2 候选 A 高斯型)：峰位左移、峰高下降、峰宽收窄。
	execution := blackBoxExecution(rEff, theta)

	// G10 表面服从脉冲 (22-02 §3.5)：通牒后短期表面服从——Θ 不变、E 暂时抬高。
	// 临时项随脉冲剩余月递减，结算后脉冲减一。
	if bb.ObediencePulse > 0 {
		frac := float64(bb.ObediencePulse) / float64(BlackBoxObediencePulseMonths)
		if frac > 1 {
			frac = 1
		}
		execution = clamp01(execution + BlackBoxObediencePulseDeltaE*frac)
		bb.ObediencePulse--
	}

	bb.Theta = theta
	bb.CurrentExecution = execution
}

// blackBoxExecution E(r, Θ) = E_max(Θ)·exp(−(r − r*(Θ))² / (2σ(Θ)²))
// (01 卷 §4.2 候选 A)。r*(Θ) = r_0 − k_shift·Θ；E_max(Θ) = e_0 − (e_0−e_1)·Θ；
// σ(Θ) = σ_0·(1 − k_width·Θ)。
func blackBoxExecution(r, theta float64) float64 {
	peak := BlackBoxPeakR0 - BlackBoxPeakShiftK*theta
	height := BlackBoxPeakE0 - (BlackBoxPeakE0-BlackBoxPeakE1)*theta
	width := BlackBoxPeakSigma0 * (1 - BlackBoxPeakWidthK*theta)
	d := r - peak
	return height * math.Exp(-d*d/(2*width*width))
}

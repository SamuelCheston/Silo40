package service

// 1 号内部参数集中于此 (22 黑箱学习系列 01 卷 §5/§6/§9)。
// 黑箱壁禁止本文件所在决策侧 import engine (silo1_blackbox_test.go AST 守卫)，
// 因此属 1 号自己理论的参数不放进 engine/blackbox_constants.go，而是与世界侧
// 参数表保持独立同值基线 (01 卷 §6.2)：基线初值与真实参数同值，但属 1 号
// 自己的理论——它读不到真实 Θ，只读自己的信念。
// 所有数值为推荐初值，等待 sim 校准覆盖 (TODO(balance))。

import "silo40/internal/model"

// ===== 1 号内部参数 (与 01 §9 一致) =====

const (
	blackBoxBins = 20 // N：信念箱数 (箱宽 0.05，θ_danger 恰落箱边界 12/20)

	blackBoxPriorLambda  = 4.0  // λ：先验衰减率 (先验均值 ≈ 0.24)
	blackBoxPriorFloor   = 0.05 // ε：先验均匀地板 (防零锁死)
	blackBoxDriftHat     = 0.15 // ρ̂：1 号对主任立场月漂移上界的内部假设
	blackBoxDangerTheta  = 0.60 // θ_danger：危险区阈值
	blackBoxPressureCost = 0.10 // κ：高压指派的压力代价系数
	blackBoxCredibleMass = 0.90 // 不确定性输出：最短区间包含质量

	blackBoxLogCap             = 24 // 决策日志环形上限 (与 NegotiationLogCap 同口径)
	blackBoxProbeCooldownStart = 6  // ProbeCooldown 初值 (月，01 卷 §9，接口预留)

	blackBoxMenuPoints = 20 // 决策菜单点数：{0.00, 0.05, ..., 0.95}，步长 0.05 = 1/20
)

// 1 号自己的高斯内部执行模型 Ê(r;θ) 的参数 (01 卷 §6.2)：基线初值与真实参数
// 同值，但属 1 号自己的理论——它读不到真实 Θ，只读自己的信念。
const (
	blackBoxInternalPeakR       = 0.45 // r_0：Θ=0 时峰位
	blackBoxInternalPeakShift   = 0.30 // k_shift：峰移斜率
	blackBoxInternalPeakHigh    = 0.95 // e_0：Θ=0 时峰高
	blackBoxInternalPeakLow     = 0.40 // e_1：Θ=1 时峰高
	blackBoxInternalPeakWidth   = 0.20 // σ_0：Θ=0 时峰宽
	blackBoxInternalWidthShrink = 0.50 // k_width：峰宽收窄斜率
)

// 内部伪装率假设的线性形式参数 (π̂(θ) = clamp01((θ−0.30)/0.40)，
// 与产生侧同形式但属 1 号自己的模型)。
const (
	blackBoxDisguiseOnset = 0.30
	blackBoxDisguiseSpan  = 0.40
)

// blackBoxSignalParams 1 号对一种观测信号的内部似然参数化。
type blackBoxSignalParams struct {
	delay         int     // δ_k：延迟 (月)
	noise         float64 // σ_k：观测噪声
	disguiseShift float64 // d_k：伪装偏移
}

// blackBoxSignalTable 信号参数表 (键为 model.BlackBoxSignal*)。
var blackBoxSignalTable = map[string]blackBoxSignalParams{
	model.BlackBoxSignalReceiptDelay: {delay: 1, noise: 0.10, disguiseShift: 0.05},
	model.BlackBoxSignalExecutionGap: {delay: 2, noise: 0.12, disguiseShift: 0.15},
	model.BlackBoxSignalLineTone:     {delay: 2, noise: 0.15, disguiseShift: 0.25},
}

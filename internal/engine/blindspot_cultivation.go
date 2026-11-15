package engine

// 盲区经营决策桩 (23 系列 03 卷)。属盲区经营格实现。
//
// 纪律：手段选择是六因子连乘权重 + ε 地板的三级抽取 (是否动手 → 手段类 →
// 力度)，不得退化为确定性 argmax；每堡每月 RNG 消耗 1–3 次 (DrawFloat64)，
// 堡按 SiloID 升序；功能关闭 (BlindSpot.Initialized 为 false) → 零消耗。

import "silo40/internal/model"

// RunBlindSpotCultivationMonthly RunLocalPhase 步骤 4.62 (23 系列 03 卷)：
// 每堡每月一次带权重的随机伪装决策；抽中的 (手段类 + 力度) 写入 04 卷累积
// 状态 (盲区厚度月度增量) 并留决策审计日志。
func RunBlindSpotCultivationMonthly(silo *model.Silo) {
	// TODO(盲区经营格): 按 23 系列 03 卷 §2/§3 实现。
}

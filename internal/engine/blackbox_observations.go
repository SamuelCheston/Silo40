package engine

// 黑箱观测产生桩 (22 黑箱学习系列 01 卷 §2.2/§5.2)。属观测核心格实现。
//
// RNG 纪律：唯一消耗点。每产生信号 1 次伪装判定 + 2 次噪声取样 (Box-Muller)，
// 信号类型升序、堡 SiloID 升序；功能关闭或信号当月不产生 → 零消耗。

import "silo40/internal/model"

// GenerateBlackBoxObservations RunLocalPhase 步骤 4.65：反映结算后的 Θ，
// 按信号类型升序产生观测并按延迟入队 PendingObservations。
func GenerateBlackBoxObservations(silo *model.Silo) {
	// TODO(观测核心格): 按 01 卷 §5.2/§7.1、02 卷似然/伪装模型与
	// 23 系列 05 卷注入映射实现。
}

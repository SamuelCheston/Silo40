package engine

// 黑箱物理层桩 (22 黑箱学习系列 01 卷)。属物理层格实现。
//
// 黑箱壁纪律：本文件的 Θ 真值、E 曲线参数只允许被本地堡机制 (4.55 结算、
// 4.6 消费、4.65 观测产生) 读取；服务包的黑箱回合 (信念与决策) 禁止引用本
// 文件的任何参数与真值标识符 (AST 守卫测试锁定)。

import "silo40/internal/model"

// UpdateBlackBoxMonthly RunLocalPhase 步骤 4.55 (22 黑箱学习系列 01 卷 §2.2)：
// 读上月指派 LastAssignedRisk，结算 Θ(t+1) = clamp01(Θ + growth(r) − decay(Θ, r))，
// 计算 E(r, Θ(t+1)) 写入 CurrentExecution 生效槽。世界侧簿记，零 RNG。
// 门控：BlackBox.Initialized 为 false (功能关闭/旧档/单堡) 时直接返回。
func UpdateBlackBoxMonthly(silo *model.Silo) {
	// TODO(物理层格): 按 01 卷 §3/§4 与 §9 参数总表实现。
}

// GenerateBlackBoxObservations RunLocalPhase 步骤 4.65 (22 黑箱学习系列 01 卷
// §2.2)：反映结算后的 Θ，按信号类型升序产生观测 (每信号 1 次伪装判定 + 2 次
// 噪声取样，engine.DrawFloat64)，按延迟入队 PendingObservations。
// 唯一 RNG 消耗点：门控关闭或信号当月不产生时零消耗。
func GenerateBlackBoxObservations(silo *model.Silo) {
	// TODO(观测核心格): 按 01 卷 §5.2/§7.1 与 02 卷似然/伪装模型实现。
}

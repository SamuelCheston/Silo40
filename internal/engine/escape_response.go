package engine

import (
	"silo40/internal/model"
)

// 逃跑筹备的反制层 (20 分册修订、ADR-0014)。
//
// 设计纪律：
//   - 审计没收已整体移除：它造成反向梯度 (准备越重、暴露越高、被没收越多，
//     重准备反而更吃亏)。囤积账本 (防护服/补给/材料) 永不被反制扣减。
//   - 组织怀疑度只由组织行为驱动：动员范围 (演练/预案功能位/照护与生产
//     组织) 与通信频次规模 (外溢信道/保障磋商)。囤积不抬高怀疑度——
//     准备本身不惹眼，惹眼的是组织行为。
//   - 怀疑度到门槛后 IT 必须做选择：镇压 / 蒙住 1 号 / 不作为。选择是
//     带权重的随机裁量 (ADR-0007 同构)，不是阈值 if-else；权重因子可归因，
//     决策日志留痕。
//   - 任何机制都不把玩家困死：全部后果是成本与风险的上浮 (戒备/恐慌/
//     safeguard 风险)，不直接判负、不没收、不永久锁定；玩家永远还有路走。
//   - 随机源走引擎单一 RNG 流：未产生组织怀疑度的运行不进入任何取样，
//     RNG 消耗逐位不变 (同谈判层纪律)。

// 怀疑度累积与衰减的全部数值为初始校准值。TODO(balance)
const (
	// escapeResponseSuspicionDecay 月度衰减：组织行为停了，风声也会淡
	// (与信念淡忘 0.01/月同构，17 §3 修复 A)。
	escapeResponseSuspicionDecay = 0.01
)

// AddEscapeOrgSuspicion 组织行为累积怀疑度 (内容效果 escape_org_suspicion_delta
// 的唯一写入口)。夹到 [0,1]；首次累积初始化响应层。
func AddEscapeOrgSuspicion(silo *model.Silo, delta float64) {
	if silo == nil || delta <= 0 {
		return
	}
	state := &silo.EscapeResponse
	state.Initialized = true
	state.OrgSuspicion = clampUnit(state.OrgSuspicion + delta)
}

// DecayEscapeOrgSuspicion 月度衰减 (月度响应回合调用)。未初始化的运行
// 逐位不变。
func DecayEscapeOrgSuspicion(silo *model.Silo) {
	if silo == nil || !silo.EscapeResponse.Initialized {
		return
	}
	silo.EscapeResponse.OrgSuspicion = clampUnit(silo.EscapeResponse.OrgSuspicion - escapeResponseSuspicionDecay)
}

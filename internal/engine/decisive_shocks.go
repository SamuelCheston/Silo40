package engine

import "silo40/internal/model"

// 裁决 (W 公式与约章目标不兼容的岔口)：日常平衡点维持现状——极限压力下
// W_eq 底部落在 25–32，表达"平常的压力磨不垮威势"；约章门 W≤18 只能被
// 结构性事件的离散负冲击击穿。冲击通道不复用 E、不走日常累积，量级推导
// 见 docs/research/10-it-control-credibility/14-parameters-and-audit.md。

// 触发源只给三类 (裁决)。
const (
	// DecisiveSourceSilo1Destroyed 1 号地堡覆灭。
	DecisiveSourceSilo1Destroyed = "silo1_destroyed"
	// DecisiveSourceTerminationFailed 终止链失效。
	DecisiveSourceTerminationFailed = "termination_channel_disrupted"
	// DecisiveSourceMassEvidence 大规模公开证据 (合格公开事实数首次达到约章门槛)。
	DecisiveSourceMassEvidence = "large_scale_evidence"
)

// DecisiveShockMagnitude 单次决定性冲击的默认量级。
// TODO(balance): −24 是推导起点 (见 14 分册 §冲击量级推导)：W_eq 谷底
// 25–32、α≈0.083 时，单次 −24 从谷底可维持约 6 个月 W≤18 窗口，足够
// PactEntryMonths=3 的锁定；需跑验与对局测量后定稿。
const DecisiveShockMagnitude = -24.0

// QueueDecisiveShock 为下一次月度权威更新排队一次结构性离散冲击。同源只发
// 一次 (不叠加)：已登记或已消费的同源冲击会使本次入队返回 false。记录永久
// 保留在 DecisiveShocks 中作审计，消费只推进 DecisiveApplied 游标。
func QueueDecisiveShock(silo *model.Silo, source string, value float64) bool {
	if silo == nil || source == "" || value == 0 {
		return false
	}
	for _, shock := range silo.Authority.DecisiveShocks {
		if shock.Source == source {
			return false
		}
	}
	silo.Authority.DecisiveShocks = append(silo.Authority.DecisiveShocks, model.DecisiveShock{
		Value:  value,
		Source: source,
		Year:   silo.CurrentYear,
		Month:  silo.CurrentMonth,
	})
	return true
}

// consumeDecisiveShocks 在月度权威更新中消费待处理的决定性冲击。不受普通
// 冲击队列的 ±12 上限约束 (裁决：可击穿日常平衡点下限)；记录保留，只推进
// 消费游标。
func consumeDecisiveShocks(a *model.ShadowAuthority) {
	for a.DecisiveApplied < len(a.DecisiveShocks) {
		shock := a.DecisiveShocks[a.DecisiveApplied]
		a.DecisiveApplied++
		a.W = clamp(a.W+shock.Value, 0, 100)
	}
}

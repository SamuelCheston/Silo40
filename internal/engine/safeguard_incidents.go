package engine

// ADR-0018：safeguard 由后果累加改为 1 号指派基线——Go 侧后果累加点改为事件
// 登记。原直接改写 Silo.SafeguardRisk 的四类累加点 (筹备反制不作为、蒙蔽崩塌、
// 协议偏离、IT 过度干预) 照旧发生、照旧记录，但只登记到 Silo.SafeguardIncidents，
// 经观测管线 (举报率代理输入) 进入 1 号信念，即「改为进入 1 号决策的输入因子」。
//
// ≥1.0 → DEATH 硬线保留 (engine.go checkVictoryConditions 与
// events/mechanics/core_systems.js VICTORY_CHECK 两侧，22 系列 00 卷 §5.1)：
// 该线是内容事件推动的渠道 (《寂堡》)，1 号决策层永不自愿触碰 (菜单上界 0.95)。

import (
	"math"

	"silo40/internal/model"
)

// 登记类型 (SafeguardIncidents 的 kind 键)。
const (
	// SafeguardIncidentIgnore 筹备反制不作为：异常悬而未决。
	SafeguardIncidentIgnore = "ignore"
	// SafeguardIncidentCoverCollapse 蒙蔽崩塌：付不起月费或暴露。
	SafeguardIncidentCoverCollapse = "cover_collapse"
	// SafeguardIncidentDeviation 协议偏离：按期数登记 (违反=一次，
	// 未贯彻=每逾期月，未达成=每次 review)。
	SafeguardIncidentDeviation = "deviation"
	// SafeguardIncidentITOverreach IT 过度干预：对风险度的保护性篡改。
	SafeguardIncidentITOverreach = "it_overreach"
)

// registerSafeguardIncident 向堡的事件登记簿登记 n 次 kind 类事件
// (map 惰性初始化；旧档零值 = 空登记簿)。
func registerSafeguardIncident(silo *model.Silo, kind string, n int) {
	if silo == nil || n <= 0 {
		return
	}
	if silo.SafeguardIncidents == nil {
		silo.SafeguardIncidents = map[string]int{}
	}
	silo.SafeguardIncidents[kind] += n
}

// applySafeguardConsequence 承接原「直接累加风险度」的各后果点：功能开启时
// 只登记事件 (经观测管线进入 1 号信念)；功能关闭时按旧语义直接累加
// SafeguardRisk，保证旧档零值行为逐位不变 (ADR-0018)。
func applySafeguardConsequence(silo *model.Silo, kind string, riskDelta float64, n int) {
	if silo == nil {
		return
	}
	if silo.BlackBox.Initialized {
		registerSafeguardIncident(silo, kind, n)
		return
	}
	if riskDelta != 0 && n > 0 {
		silo.SafeguardRisk = math.Max(0, silo.SafeguardRisk+riskDelta*float64(n))
	}
}

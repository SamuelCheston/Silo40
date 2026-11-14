package engine

import (
	"strings"

	"silo40/internal/model"
)

// 上层控制方相关事实 (§6.4/§6.5)：三类生成器各自的 Fact。Order 事实不在
// ALL_FRAGMENTS 公共目录里——它们不属于堡内公共图书馆。
const (
	OrderFactRelicNetwork  = "Order_1" // Relic 指向的多堡网络
	OrderFactHiddenAuthors = "Order_2" // Protocol 存在隐藏编写者
	OrderFactChannelNature = "Order_3" // 控制通道背后是什么
)

// 生成器观察的渠道可靠度 (TODO(balance): sim 起点)。
const (
	relicReliability      = 0.7
	reflectionReliability = 0.6
	channelReliability    = 0.9
)

// absorbDirectorObservation 三类生成器的统一落点 (§6.5：统一输出
// Fact/Version/Observation)：只更新主任信念层，再重算 G；不直接改
// G/W/E/Q。observation_id 去重——同一实物/参悟/话语不重复产出。
func absorbDirectorObservation(silo *model.Silo, factID, obsID string, reliability float64) bool {
	if silo == nil || factID == "" || strings.TrimSpace(obsID) == "" {
		return false
	}
	state := &silo.Protocol
	belief := EnsureBeliefIn(&state.DirectorBeliefs, factID)
	if belief == nil {
		return false
	}
	cfg := DefaultInformationConfig()
	strength := AbsorptionStrength(cfg, reliability, 0, cfg.IndependenceNew)
	if !ObserveFact(cfg, belief, FragmentSpecFor(factID), obsID, strength) {
		return false
	}
	RecomputeG(state)
	return true
}

// InspectRelicForDirector Relic 生成器 (§6.5 A)：每件实物×检查方式一次。
// 实物是否存在由调用方 (内容层) 负责；没收链也走同一入口 (没收=检查方式之一)。
func InspectRelicForDirector(silo *model.Silo, relicKey, method string) bool {
	relicKey = strings.TrimSpace(relicKey)
	method = strings.TrimSpace(method)
	if relicKey == "" || method == "" {
		return false
	}
	return absorbDirectorObservation(silo, OrderFactRelicNetwork, "relic:"+relicKey+":"+method, relicReliability)
}

// ReflectOnProtocol Protocol 事后参悟生成器 (§6.5 B)：每个不同
// reflection_key 一次 (一次具体处置与 Protocol 不符/缺条/目标失败)。
func ReflectOnProtocol(silo *model.Silo, reflectionKey string) bool {
	reflectionKey = strings.TrimSpace(reflectionKey)
	if reflectionKey == "" {
		return false
	}
	return absorbDirectorObservation(silo, OrderFactHiddenAuthors, "reflection:"+reflectionKey, reflectionReliability)
}

// ReceiveChannelUtterance 控制通道对话生成器 (§6.5 C)：仅在介入链开启时
// 可接收 (常态无接触)；每个不同 utterance_id 一次。
func ReceiveChannelUtterance(silo *model.Silo, utteranceID string) bool {
	utteranceID = strings.TrimSpace(utteranceID)
	if utteranceID == "" || silo == nil || !silo.Protocol.ChannelOpen {
		return false
	}
	return absorbDirectorObservation(silo, OrderFactChannelNature, "utterance:"+utteranceID, channelReliability)
}

// orderFactCount 上层控制方相关事实的固定集合大小：G 按固定集合取均值，
// 未知事实计 0——新观察只推进 G，不因新事实加入而稀释 (单调)。
const orderFactCount = 3

// RecomputeG §6.4：G 由上层控制方相关事实 (Order_*) 的主任信念聚合
// (固定集合等权均值)。确定性；无观察时 G=0 (学徒阶段)。
func RecomputeG(state *model.ProtocolState) {
	if state == nil {
		return
	}
	total := 0.0
	for i := range state.DirectorBeliefs {
		belief := &state.DirectorBeliefs[i]
		if !strings.HasPrefix(belief.FactID, "Order_") {
			continue
		}
		total += belief.B
	}
	state.G = total / orderFactCount
}

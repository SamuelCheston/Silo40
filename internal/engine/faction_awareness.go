package engine

import (
	"fmt"
	"math"

	"silo40/internal/model"
)

// ============ 派系感知 (24 分册 §1.2/§1.3, C8) ============
// 发起派系时，全堡观察者 (部门 + 挂 cohort 的居民代表 + 玩家) 按三通道
// 感知度写入 faction:<id>:founded 信念事实；此后月度口碑沿关系网络扩散，
// 信息动作成功时搭载传闻。感知事实只淡忘、永不成熟、不计共证门
// (information.go 豁免)。C9 加入掷骰读 belief.B ≥ AwarenessJoinFloor。
//
// 确定性 (§6)：感知度是纯函数零 RNG；口碑掷骰按 (派系 ID, 源, 目标) 固定
// 升序；玩家搭载不掷骰；无政治派系时搭载零掷骰 (不移动早期共享流)。

// FactionFoundedFactID 派系成立感知事实键：faction:<派系ID>:founded。
func FactionFoundedFactID(factionID uint) string {
	return fmt.Sprintf("%s%d:founded", FactionFactPrefix, factionID)
}

// factionAwareness 三通道感知度 ∈ [0,1] (24 §1.2)，各通道单调：
//   - 关系通道：观察者对发起者所属部门的人脉 (GetConnection 统一口径)。
//     居民对领袖的个人关系子通道待 C9 的 LeaderRelations 字段，暂不启用。
//   - 思潮通道：(pf, loyalty) 平面加权欧氏距离取补，复用聚类特征权重。
//   - 阶层通道：权力级差取补 (级差 10 为满幅)，同 ClassType 加成。
//
// 加权平均后 clamp01；权重全零时返回 0。纯函数，零 RNG。
func factionAwareness(silo *model.Silo, observer, leader *ActorView) float64 {
	if silo == nil || observer == nil || leader == nil {
		return 0
	}
	leaderProf := professionByActor(silo, leader)

	rel := 0.0
	if leaderProf != nil {
		rel = clamp01(observer.GetConnection(leaderProf.ID))
	}

	oi, li := observer.Ideologies(), leader.Ideologies()
	wPF := FactionConfig.FeatureWeightProForeign
	wLoy := FactionConfig.FeatureWeightLoyalty
	a := []float64{oi[model.IdeologyProForeign] * wPF, oi[model.IdeologyLoyalty] * wLoy}
	b := []float64{li[model.IdeologyProForeign] * wPF, li[model.IdeologyLoyalty] * wLoy}
	dMax := featureDistance([]float64{wPF, wLoy}, []float64{0, 0})
	ideo := 0.5
	if dMax > 0 {
		ideo = clamp01(1 - featureDistance(a, b)/dMax)
	}

	tier := 0.0
	observerProf := professionByActor(silo, observer)
	if observerProf != nil && leaderProf != nil {
		gap := math.Abs(float64(observerProf.PowerLevel) - float64(leaderProf.PowerLevel))
		tier = clamp01(1 - gap/10)
		if observerProf.ClassType == leaderProf.ClassType {
			tier = clamp01(tier + FactionConfig.AwarenessSameClassBonus)
		}
	}

	wR := FactionConfig.AwarenessRelationWeight
	wI := FactionConfig.AwarenessIdeologyWeight
	wT := FactionConfig.AwarenessTierWeight
	sum := wR + wI + wT
	if sum <= 0 {
		return 0
	}
	return clamp01((wR*rel + wI*ideo + wT*tier) / sum)
}

func sameActorRef(a, b ActorRef) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case ActorPlayer:
		return a.AgentID == b.AgentID
	case ActorNPC:
		return a.ProfessionID == b.ProfessionID
	default:
		return a.ResidentID == b.ResidentID
	}
}

// writeFoundedBelief 把发起感知度直接写入信念 (先验而非观察吸收)：
// B 取感知度与既有值的较大者，观察 ID 去重。零 RNG。
func writeFoundedBelief(beliefs *[]model.FactBelief, factID string, awareness float64, obsID string) {
	belief := EnsureBeliefIn(beliefs, factID)
	if belief == nil {
		return
	}
	if awareness > belief.B {
		belief.B = clamp01(awareness)
	}
	for _, obs := range belief.Observations {
		if obs == obsID {
			return
		}
	}
	belief.Observations = append(belief.Observations, obsID)
	belief.ObsCount++
}

// seedFactionAwareness 发起钩子：为每个观察者写入成立感知事实。
// 遍历顺序固定：部门 (切片序 = ID 升序) → 挂 cohort 的存活居民代表
// (切片序) → 玩家最后 (ADR-0018 同构)。创始人自身 B=1。不挂 cohort 的
// 居民 (18 号正典具名人物) 不是代表观察者，跳过。零 RNG。
func (e *GameEngine) seedFactionAwareness(silo *model.Silo, agent *model.Agent, faction *model.Faction, founder *ActorView) {
	if silo == nil || faction == nil || founder == nil {
		return
	}
	factID := FactionFoundedFactID(faction.ID)
	obsID := fmt.Sprintf("found:%s:Y%dM%d", factID, faction.FoundedYear, faction.FoundedMonth)

	for i := range silo.Professions {
		prof := &silo.Professions[i]
		view, err := CreateActorView(CreateActorRefForProfession(prof), silo, nil)
		if err != nil {
			continue
		}
		awareness := factionAwareness(silo, view, founder)
		if sameActorRef(view.Ref, founder.Ref) {
			awareness = 1
		}
		writeFoundedBelief(&prof.Beliefs, factID, awareness, obsID)
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if !r.Alive || r.CohortID == nil {
			continue
		}
		view, err := CreateActorView(CreateActorRefForResident(r), silo, nil)
		if err != nil {
			continue
		}
		awareness := factionAwareness(silo, view, founder)
		if sameActorRef(view.Ref, founder.Ref) {
			awareness = 1
		}
		writeFoundedBelief(&r.Beliefs, factID, awareness, obsID)
	}
	if agent != nil {
		view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
		if err == nil {
			awareness := factionAwareness(silo, view, founder)
			if sameActorRef(view.Ref, founder.Ref) {
				awareness = 1
			}
			writeFoundedBelief(&agent.Beliefs, factID, awareness, obsID)
		}
	}
}

// gossipSource 口碑传播源：观察者视图 + 信念层指针 + 同源去重标签。
type gossipSource struct {
	view    *ActorView
	beliefs *[]model.FactBelief
	tag     string
}

// ApplyFactionGossip 月度口碑扩散 (24 §1.3)：置信度 ≥ GossipBeliefThreshold
// 的观察者把派系事实沿关系网络传给关系 ≥ GossipRelationMin 的部门，
// 每跳接收强度 = 源 B × GossipReliabilityDecay，每 (源, 目标) 对按
// GossipSpreadChance 掷骰一次。多跳衰减自然涌现：接收方达到阈值后在
// 后续月份以自己的 (更低) B 继续传。
//
// 掷骰顺序固定：派系 ID 升序 → 源 (部门 ID 升序、居民 ID 升序、玩家最后)
// → 目标部门 ID 升序。切片序即 ID 升序 (初始化与追加均按 max+1)。
// 挂在 RunNpcTurn 月度演化步 (deltaYears > 0)，不进 MaintainFactions。
func ApplyFactionGossip(silo *model.Silo, agent *model.Agent) {
	if silo == nil {
		return
	}
	var sources []gossipSource
	for i := range silo.Professions {
		prof := &silo.Professions[i]
		view, err := CreateActorView(CreateActorRefForProfession(prof), silo, nil)
		if err != nil {
			continue
		}
		sources = append(sources, gossipSource{
			view: view, beliefs: &prof.Beliefs,
			tag: fmt.Sprintf("gossip:%s:%d", ActorNPC, prof.ID),
		})
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if !r.Alive || r.CohortID == nil {
			continue
		}
		view, err := CreateActorView(CreateActorRefForResident(r), silo, nil)
		if err != nil {
			continue
		}
		sources = append(sources, gossipSource{
			view: view, beliefs: &r.Beliefs,
			tag: fmt.Sprintf("gossip:%s:%d", ActorResident, r.ID),
		})
	}
	if agent != nil {
		view, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
		if err == nil {
			sources = append(sources, gossipSource{
				view: view, beliefs: &agent.Beliefs,
				tag: fmt.Sprintf("gossip:%s:%d", ActorPlayer, agent.ID),
			})
		}
	}
	if len(sources) == 0 {
		return
	}

	cfg := DefaultInformationConfig()
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == unaffiliatedSignature {
			continue
		}
		factID := FactionFoundedFactID(f.ID)
		for _, src := range sources {
			belief := BeliefIn(src.beliefs, factID)
			if belief == nil || belief.B < FactionConfig.GossipBeliefThreshold {
				continue
			}
			strength := belief.B * FactionConfig.GossipReliabilityDecay
			for j := range silo.Professions {
				target := &silo.Professions[j]
				if src.view.Ref.Kind == ActorNPC && target.ID == src.view.Ref.ProfessionID {
					continue
				}
				if src.view.GetConnection(target.ID) < FactionConfig.GossipRelationMin {
					continue
				}
				if rng.Float64() >= FactionConfig.GossipSpreadChance {
					continue
				}
				AbsorbSharedFact(cfg, &target.Beliefs, factID, src.tag, strength)
			}
		}
	}
}

// actorBeliefs 该 Actor 的信念层指针 (Agent / Resident / Profession 同构)。
func actorBeliefs(v *ActorView) *[]model.FactBelief {
	if v == nil {
		return nil
	}
	if v.agent != nil {
		return &v.agent.Beliefs
	}
	if v.resident != nil {
		return &v.resident.Beliefs
	}
	if v.prof != nil {
		return &v.prof.Beliefs
	}
	return nil
}

// applyFactionRumorPiggyback 信息动作搭载传闻 (24 §1.3)：GATHER_INFO /
// SHARE_INFO / 田间闲聊 / Medical 被动成功时调用。无政治派系零掷骰返回
// (早期共享流不移动)；玩家确定性搭载不消耗 RNG (§6)，NPC/居民按
// RumorPiggybackChance 掷骰一次。命中后选当前信念最低的派系 (并列取
// ID 最小，确定性无额外掷骰)，按 ChannelRumorRel 强度吸收一次 "rumor" 观察。
func (e *GameEngine) applyFactionRumorPiggyback(silo *model.Silo, actor *ActorView) {
	if silo == nil || actor == nil {
		return
	}
	var political []*model.Faction
	for i := range silo.Factions {
		if silo.Factions[i].Signature != unaffiliatedSignature {
			political = append(political, &silo.Factions[i])
		}
	}
	if len(political) == 0 {
		return
	}
	beliefs := actorBeliefs(actor)
	if beliefs == nil {
		return
	}
	if !actor.IsPlayer() && rng.Float64() >= FactionConfig.RumorPiggybackChance {
		return
	}

	var pick *model.Faction
	pickB := math.Inf(1)
	for _, f := range political {
		b := 0.0
		if belief := BeliefIn(beliefs, FactionFoundedFactID(f.ID)); belief != nil {
			b = belief.B
		}
		if b < pickB {
			pick, pickB = f, b
		}
	}
	if pick == nil {
		return
	}
	cfg := DefaultInformationConfig()
	AbsorbSharedFact(cfg, beliefs, FactionFoundedFactID(pick.ID), "rumor", cfg.ChannelRumorRel)
}
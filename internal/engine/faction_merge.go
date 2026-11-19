package engine

// 合并 (24 篇 §1.6, C10)：领袖发起 MERGE_FACTION，把目标派系并入自己
// 领导的派系。存续方 (发起者派系) 保留 ID，被并方成员 cohort/居民转移、
// Memory 追加合并记录、被并方除名。三闸：双方领袖个人关系 ≥ MergeRelationMin
// (LeaderRelations 子矩阵，玩家领袖走 agent:<id> 键)、思潮距离 ≤ 亲缘半径、
// AP ≥ MergeAPCost。零成员派系可发起/被合并 (亲缘回退领袖特征)。
//
// 与加入不同，合并的关系闸对玩家同样生效：合并是两位领袖的合意，不能
// 凭干预者身份强迫对方领袖 (ADR-0010 的直选豁免只适用于加入自己的归属)。
//
// 确定性契约 (§6)：玩家动作不消耗 RNG；NPC 加权候选不掷骰 (落点由 pick
// 在 scoreCandidates 统一消耗)；成员转移按切片序，浮点统计走 recomputeFactionStats。

import (
	"fmt"

	"silo40/internal/model"
)

// actorLedFaction 该 Actor 任领袖 (代表) 的派系，按 RepresentativeKind/ID
// 精确匹配 (区别于 FactionID 的成员资格回退)。无领导派系返回 nil。
func actorLedFaction(silo *model.Silo, view *ActorView) *model.Faction {
	if silo == nil || view == nil {
		return nil
	}
	var kind string
	var id uint
	switch view.Ref.Kind {
	case ActorPlayer:
		kind, id = string(ActorPlayer), view.Ref.AgentID
	case ActorNPC:
		kind, id = string(ActorNPC), view.Ref.ProfessionID
	case ActorResident:
		kind, id = string(ActorResident), view.Ref.ResidentID
	default:
		return nil
	}
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.RepresentativeKind == kind && f.RepresentativeID == id {
			return f
		}
	}
	return nil
}

// factionLeaderPersonalRelation 两位领袖的个人关系 (LeaderRelations 子矩阵)，
// 取双向读数的最小值 (合意需双方都达标)。居民领袖读自身子矩阵中对方键；
// 玩家领袖无独立子矩阵，其方向由对方居民子矩阵的 agent:<id> 键承载。
// 返回 (关系值, 是否有可读方向)；双方都不可解析时 ok=false。
func factionLeaderPersonalRelation(silo *model.Silo, fA, fB *model.Faction) (float64, bool) {
	if silo == nil || fA == nil || fB == nil {
		return 0, false
	}
	var vals []float64
	if fA.RepresentativeKind == string(ActorResident) {
		if rA := residentByID(silo, fA.RepresentativeID); rA != nil {
			vals = append(vals, rA.LeaderRelations[leaderRelationKey(fB.RepresentativeKind, fB.RepresentativeID)])
		}
	}
	if fB.RepresentativeKind == string(ActorResident) {
		if rB := residentByID(silo, fB.RepresentativeID); rB != nil {
			vals = append(vals, rB.LeaderRelations[leaderRelationKey(fA.RepresentativeKind, fA.RepresentativeID)])
		}
	}
	if len(vals) == 0 {
		return 0, false
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// mergeFactionEligible MERGE_FACTION 资格三闸 + 领袖身份 (24 篇 §1.6)。
// 发起者必须领导一个政治派系 (存续方)，目标须为另一政治派系。关系/思潮/AP
// 三闸对玩家与 NPC 同口径生效。
func mergeFactionEligible(view *ActorView, silo *model.Silo, target *model.Faction, agent *model.Agent) (bool, string) {
	if view == nil || silo == nil || target == nil {
		return false, "actor unavailable"
	}
	if target.Signature == unaffiliatedSignature {
		return false, "无阵营桶不是政治派系"
	}
	own := actorLedFaction(silo, view)
	if own == nil {
		return false, "只有派系领袖才能发起合并"
	}
	if own.ID == target.ID {
		return false, "不能与自身合并"
	}
	if view.ActionPoints() < FactionConfig.MergeAPCost {
		return false, "AP 不足"
	}
	if rel, ok := factionLeaderPersonalRelation(silo, own, target); !ok || rel < FactionConfig.MergeRelationMin {
		return false, "领袖个人关系不足"
	}
	cA, okA := factionFeatureCentroid(silo, own, agent)
	cB, okB := factionFeatureCentroid(silo, target, agent)
	if !okA || !okB {
		return false, "无法计算亲缘"
	}
	if featureDistance(cA, cB) > FactionConfig.AffinityRadius {
		return false, "思潮距离超出亲缘半径"
	}
	return true, ""
}

// mergeableFactionsForLeader 领袖当前可合并的目标派系列表 (三闸通过)。
// NPC 加权候选用；不消耗 RNG。无 agent 上下文，玩家领袖的零成员目标因
// 亲缘不可解析而排除 (与 joinableFactionsForRepresentative 同限制)。
func mergeableFactionsForLeader(silo *model.Silo, view *ActorView) []*model.Faction {
	if silo == nil || view == nil {
		return nil
	}
	own := actorLedFaction(silo, view)
	if own == nil {
		return nil
	}
	var out []*model.Faction
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == unaffiliatedSignature || f.ID == own.ID {
			continue
		}
		if ok, _ := mergeFactionEligible(view, silo, f, nil); ok {
			out = append(out, f)
		}
	}
	return out
}

// applyFactionMergeActor 处理 faction_merge_actor mutation：权威资格判定、
// 成员转移、Memory 追加、被并方除名与 AP 扣除原子完成 (JS 侧只做同口径
// 预检给出失败文案)。存续方保留 ID；目标派系 ID 由 mutation.IntValue 携带。
func (e *GameEngine) applyFactionMergeActor(silo *model.Silo, actor *ActorView, agent *model.Agent, targetID uint) {
	if actor == nil || silo == nil {
		return
	}
	target := factionByID(silo, targetID)
	if ok, reason := mergeFactionEligible(actor, silo, target, agent); !ok {
		e.logf(fmt.Sprintf("%s 的派系合并被拒绝：%s。", actor.Label(), reason))
		return
	}
	own := actorLedFaction(silo, actor)
	ownID := own.ID
	absorbedName := target.Name

	// 成员转移：被并方 cohort 与居民归属改挂存续方；被并方领袖降为成员。
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		if c.FactionID != nil && *c.FactionID == target.ID {
			c.FactionID = uintPtr(ownID)
		}
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if r.FactionID != nil && *r.FactionID == target.ID {
			r.FactionID = uintPtr(ownID)
			r.IsRepresentative = false
		}
	}
	if agent != nil && agent.FactionID != nil && *agent.FactionID == target.ID {
		agent.FactionID = uintPtr(ownID)
	}

	// Memory 追加合并记录 (存续方身份字段持久不动)。
	own.Memory = append(own.Memory, fmt.Sprintf("merged:%d:Y%dM%d", target.ID, silo.CurrentYear, silo.CurrentMonth))

	// 被并方除名 (沿用 dissolveFaction 的原地过滤；此后 own 指针可能位移，
	// 须按 ID 重取再重算统计)。
	kept := silo.Factions[:0]
	for _, f := range silo.Factions {
		if f.ID != target.ID {
			kept = append(kept, f)
		}
	}
	silo.Factions = kept

	if survivor := factionByID(silo, ownID); survivor != nil {
		recomputeFactionStats(silo, survivor, factionMemberCohorts(silo, ownID))
	}
	actor.SetActionPoints(actor.ActionPoints() - FactionConfig.MergeAPCost)
	e.logf(fmt.Sprintf("%s 把「%s」并入自己的派系 (存续 ID %d)。", actor.Label(), absorbedName, ownID))
}
package engine

// 加入/退出 (24 篇 §1.4, C9)：成员资格持久 (C7)，本文件实现其演化——
// 已感知的无阵营 cohort 每月加权随机掷骰加入，意识形态相近度主导
// (相近度^JoinAffinityExponent)；关系三档闸 + 部门敌意硬否决；漂移出
// 亲缘半径的成员 cohort 以惯性保护的对称掷骰退出。玩家通过 JOIN_FACTION
// 直选加入，不掷骰 (ADR-0018/0010)。
//
// 信念读取接缝：C8 把感知度以 `faction:<id>:founded` 事实写入观测者信念层
// (model.FactBelief，挂在 Resident/Profession/Agent 上)。引擎侧直接读 model
// 信念 (BeliefIn)，无需 service 回调——这是侵入最小的路径。运行时若无该
// 事实则 belief=0，NPC 加入不触发 (零感知=零加入，安全)。
//
// 确定性契约 (§6)：加入/退出掷骰按 cohort ID 升序；玩家动作不消耗 RNG；
// 月度扫描只在 deltaYears>0 时运行 (零月即时结算不加速成员流动)。

import (
	"fmt"
	"math"
	"sort"

	"silo40/internal/model"
)

// leaderRelationKey 领袖↔代表个人关系子矩阵的键 (24 篇 §3)：居民 res:<id>、
// 玩家 agent:<id>、部门 npc:<id>，与部门关系键 (按部门名) 隔离。
func leaderRelationKey(kind string, id uint) string {
	switch kind {
	case string(ActorResident):
		return fmt.Sprintf("res:%d", id)
	case string(ActorPlayer):
		return fmt.Sprintf("agent:%d", id)
	case string(ActorNPC):
		return fmt.Sprintf("npc:%d", id)
	}
	return ""
}

// ideologyFeature 思潮+阶层加权特征向量 (与 cohortFeature 同口径)。
func ideologyFeature(ideo map[string]float64, prof *model.Profession) []float64 {
	classCoord := 0.0
	if prof != nil && prof.ClassType == "ELITE" {
		classCoord = 1.0
	}
	return []float64{
		ideo[model.IdeologyProForeign] * FactionConfig.FeatureWeightProForeign,
		ideo[model.IdeologyLoyalty] * FactionConfig.FeatureWeightLoyalty,
		classCoord * FactionConfig.FeatureWeightClass,
	}
}

// factionLeaderProfession 派系现任领袖所属部门 (关系闸/亲缘 fallback 用)。
// PLAYER 领袖需要 agent 解析其职业；agent 缺失或不匹配时返回 nil。
func factionLeaderProfession(silo *model.Silo, f *model.Faction, agent *model.Agent) *model.Profession {
	if silo == nil || f == nil {
		return nil
	}
	switch f.RepresentativeKind {
	case string(ActorResident):
		if r := residentByID(silo, f.RepresentativeID); r != nil {
			return getProfessionByID(silo, r.ProfessionID)
		}
	case string(ActorNPC):
		return getProfessionByID(silo, f.RepresentativeID)
	case string(ActorPlayer):
		if agent != nil && agent.ID == f.RepresentativeID {
			for i := range silo.Professions {
				if silo.Professions[i].Name == agent.Profession {
					return &silo.Professions[i]
				}
			}
		}
	}
	return nil
}

// factionLeaderFeature 零成员派系的亲缘 fallback：取领袖本人的特征向量。
func factionLeaderFeature(silo *model.Silo, f *model.Faction, agent *model.Agent) ([]float64, bool) {
	if f == nil {
		return nil, false
	}
	switch f.RepresentativeKind {
	case string(ActorResident):
		if r := residentByID(silo, f.RepresentativeID); r != nil {
			return ideologyFeature(r.Ideologies, getProfessionByID(silo, r.ProfessionID)), true
		}
	case string(ActorNPC):
		if prof := getProfessionByID(silo, f.RepresentativeID); prof != nil {
			return ideologyFeature(prof.Ideologies, prof), true
		}
	case string(ActorPlayer):
		if agent != nil && agent.ID == f.RepresentativeID {
			prof := factionLeaderProfession(silo, f, agent)
			if prof != nil {
				return ideologyFeature(prof.Ideologies, prof), true
			}
		}
	}
	return nil, false
}

// factionFeatureCentroid 派系成员人数加权的特征质心；零成员时回退领袖特征。
// 返回 false = 无法计算亲缘 (无成员且领袖不可解析)，加入/合并跳过该目标。
func factionFeatureCentroid(silo *model.Silo, f *model.Faction, agent *model.Agent) ([]float64, bool) {
	if silo == nil || f == nil {
		return nil, false
	}
	members := factionMemberCohorts(silo, f.ID)
	total := 0
	sum := []float64{0, 0, 0}
	for _, c := range members {
		feat := cohortFeature(silo, c, getProfessionByID(silo, c.ProfessionID))
		for i := range sum {
			sum[i] += feat[i] * float64(c.Count)
		}
		total += c.Count
	}
	if total > 0 {
		for i := range sum {
			sum[i] /= float64(total)
		}
		return sum, true
	}
	return factionLeaderFeature(silo, f, agent)
}

// cohortRepresentative 查 cohort 的现任代表居民 (不创建——创建会消耗 RNG
// 并引入副作用，维护管线已保证代表存在)。无代表返回 nil。
func cohortRepresentative(silo *model.Silo, cohort *model.PopulationCohort) *model.Resident {
	if silo == nil || cohort == nil {
		return nil
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if r.Alive && r.CohortID != nil && *r.CohortID == cohort.ID {
			return r
		}
	}
	return nil
}

// joinCloseness 意识形态相近度 ∈ [0,1]：亲缘半径内线性衰减，圈外为 0。
func joinCloseness(dist float64) float64 {
	if FactionConfig.AffinityRadius <= 0 {
		return 0
	}
	return clamp01(1 - dist/FactionConfig.AffinityRadius)
}

// joinProbability 加入概率 = 相近度^k (意识形态主导，单调)。
func joinProbability(dist float64) float64 {
	return math.Pow(joinCloseness(dist), FactionConfig.JoinAffinityExponent)
}

// exitProbability 退出概率：漂移出亲缘半径后的对称掷骰，惯性保护使其
// 低于加入概率 (×(1−ExitInertiaFactor))。半径内为 0；刚出圈时极小 (需多
// 月累积)，远超出圈时封顶 (1−惯性)。
func exitProbability(dist float64) float64 {
	if FactionConfig.AffinityRadius <= 0 {
		return 0
	}
	excess := dist/FactionConfig.AffinityRadius - 1
	if excess <= 0 {
		return 0
	}
	if excess > 1 {
		excess = 1
	}
	return math.Pow(excess, FactionConfig.JoinAffinityExponent) * (1 - FactionConfig.ExitInertiaFactor)
}

// joinRelationGate 加入的关系三档闸 + 部门敌意硬否决 (24 篇 §1.4)。
// 关系取「领袖↔代表个人关系」与「代表对领袖部门关系」的高者。返回
// (关系值, 是否通过)。仅约束 NPC；玩家直选不走此闸 (ADR-0010)。
func joinRelationGate(silo *model.Silo, rep *model.Resident, target *model.Faction, agent *model.Agent) (float64, bool) {
	if rep == nil || target == nil {
		return 0, false
	}
	leaderProf := factionLeaderProfession(silo, target, agent)
	cohortProf := getProfessionByID(silo, rep.ProfessionID)
	if leaderProf == nil || cohortProf == nil {
		return 0, false
	}
	// 部门双向敌意硬否决 (同部门豁免)。
	if cohortProf.ID != leaderProf.ID {
		if cohortProf.Relations[leaderProf.Name] < FactionConfig.BadRelationThreshold ||
			leaderProf.Relations[cohortProf.Name] < FactionConfig.BadRelationThreshold {
			return 0, false
		}
	}
	personal := rep.LeaderRelations[leaderRelationKey(target.RepresentativeKind, target.RepresentativeID)]
	dept := rep.Relations[leaderProf.Name]
	relation := math.Max(personal, dept)
	if relation < FactionConfig.JoinRelationVeto {
		return 0, false
	}
	gap := cohortProf.PowerLevel - leaderProf.PowerLevel
	if gap < 0 {
		gap = -gap
	}
	if gap >= FactionConfig.CrossTierPowerGap && relation < FactionConfig.CrossTierRelationMin {
		return 0, false
	}
	if cohortProf.ClassType != leaderProf.ClassType && relation < FactionConfig.CrossClassRelationMin {
		return 0, false
	}
	return relation, true
}

// cohortBeliefAboutFaction cohort 对某派系发起事实的感知度：取代表居民与
// 所属部门两层信念的高者 (C8 可写任一观测者层)。无事实 = 0。
func cohortBeliefAboutFaction(silo *model.Silo, cohort *model.PopulationCohort, rep *model.Resident, factionID uint) float64 {
	factID := FactionFoundedFactID(factionID)
	best := 0.0
	if rep != nil {
		if b := BeliefIn(&rep.Beliefs, factID); b != nil && b.B > best {
			best = b.B
		}
	}
	if cohort != nil {
		if prof := getProfessionByID(silo, cohort.ProfessionID); prof != nil {
			if b := BeliefIn(&prof.Beliefs, factID); b != nil && b.B > best {
				best = b.B
			}
		}
	}
	return best
}

// sortedCohortsByID 按 cohort ID 升序返回切片 (掷骰顺序契约 §6)。
func sortedCohortsByID(silo *model.Silo) []*model.PopulationCohort {
	out := make([]*model.PopulationCohort, 0, len(silo.Cohorts))
	for i := range silo.Cohorts {
		out = append(out, &silo.Cohorts[i])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// politicalFactionByID 查政治派系 (无阵营桶返回 nil)。
func politicalFactionByID(silo *model.Silo, id uint) *model.Faction {
	f := factionByID(silo, id)
	if f == nil || f.Signature == unaffiliatedSignature {
		return nil
	}
	return f
}

// cohortJoinFaction cohort 加入派系：cohort 与代表居民归属同步转移。
func cohortJoinFaction(silo *model.Silo, cohort *model.PopulationCohort, factionID uint) {
	cohort.FactionID = uintPtr(factionID)
	if rep := cohortRepresentative(silo, cohort); rep != nil {
		rep.FactionID = uintPtr(factionID)
	}
}

// ApplyFactionJoinDynamics 月度加入/退出演化 (挂 RunNpcTurn，与
// ApplyFactionLoyaltyDynamics 同层，不是 MaintainFactions)。先退后加：
// 退出扫描只作用于既有成员，加入扫描只作用于无阵营者，且跳过本月退出者，
// 杜绝单月归属往返。零月 (deltaYears<=0) 无操作。
func ApplyFactionJoinDynamics(silo *model.Silo, agent *model.Agent, deltaYears float64) {
	if silo == nil || deltaYears <= 0 {
		return
	}
	repairPlayerFactionMembership(silo, agent)
	unaffiliatedID := ensureUnaffiliatedFaction(silo)
	cohorts := sortedCohortsByID(silo)

	// 相位 1：退出。成员 cohort 漂移出亲缘半径 → 惯性保护的对称掷骰。
	exited := make(map[uint]bool)
	for _, cohort := range cohorts {
		if cohort.Count <= 0 || cohort.FactionID == nil {
			continue
		}
		f := politicalFactionByID(silo, *cohort.FactionID)
		if f == nil {
			continue
		}
		centroid, ok := factionFeatureCentroid(silo, f, agent)
		if !ok {
			continue
		}
		feat := cohortFeature(silo, cohort, getProfessionByID(silo, cohort.ProfessionID))
		exitProb := exitProbability(featureDistance(feat, centroid))
		if exitProb <= 0 {
			continue // 仍在亲缘半径内，无退出压力
		}
		if rng.Float64() < exitProb {
			cohort.FactionID = uintPtr(unaffiliatedID)
			if rep := cohortRepresentative(silo, cohort); rep != nil {
				rep.FactionID = uintPtr(unaffiliatedID)
			}
			exited[cohort.ID] = true
		}
	}

	// 相位 2：加入。无阵营 cohort 按感知地板筛选目标，关系闸过滤，
	// 相近度^k 主导概率；每个已感知 cohort 消耗一次掷骰 (ID 升序)。
	for _, cohort := range cohorts {
		if cohort.Count <= 0 || exited[cohort.ID] {
			continue
		}
		if cohort.FactionID != nil && politicalFactionByID(silo, *cohort.FactionID) != nil {
			continue // 已有政治归属
		}
		rep := cohortRepresentative(silo, cohort)
		if rep == nil {
			continue
		}
		feat := cohortFeature(silo, cohort, getProfessionByID(silo, cohort.ProfessionID))
		bestProb := -1.0
		var bestFaction *model.Faction
		perceived := false
		for i := range silo.Factions {
			f := &silo.Factions[i]
			if f.Signature == unaffiliatedSignature {
				continue
			}
			if cohortBeliefAboutFaction(silo, cohort, rep, f.ID) < FactionConfig.AwarenessJoinFloor {
				continue
			}
			perceived = true
			if _, ok := joinRelationGate(silo, rep, f, agent); !ok {
				continue
			}
			centroid, ok := factionFeatureCentroid(silo, f, agent)
			if !ok {
				continue
			}
			prob := joinProbability(featureDistance(feat, centroid))
			if prob > bestProb { // 平局取先遇到者 (Factions 切片序)，ID 小者优先
				bestProb = prob
				bestFaction = f
			}
		}
		if !perceived {
			continue // 零感知 = 零掷骰 = 零加入
		}
		if rng.Float64() < bestProb && bestFaction != nil {
			cohortJoinFaction(silo, cohort, bestFaction.ID)
		}
	}
}

// repairPlayerFactionMembership 清理玩家悬空成员资格 (加入的派系解散/除名)。
func repairPlayerFactionMembership(silo *model.Silo, agent *model.Agent) {
	if silo == nil || agent == nil || agent.FactionID == nil {
		return
	}
	if politicalFactionByID(silo, *agent.FactionID) == nil {
		agent.FactionID = nil
	}
}

// joinFactionEligible JOIN_FACTION 动作资格 (玩家/NPC 同构的「无政治派系」闸
// + AP；NPC 另受感知地板与关系闸约束，玩家直选豁免 ADR-0010)。
func joinFactionEligible(view *ActorView, silo *model.Silo, target *model.Faction, agent *model.Agent) (bool, string) {
	if view == nil || silo == nil || target == nil {
		return false, "actor unavailable"
	}
	if target.Signature == unaffiliatedSignature {
		return false, "无阵营桶不是政治派系"
	}
	if fid := view.FactionID(); fid != nil {
		if f := factionByID(silo, *fid); f != nil && f.Signature != unaffiliatedSignature {
			if f.ID == target.ID {
				return false, "已是该派系成员"
			}
			return false, "已身处政治派系"
		}
	}
	if view.ActionPoints() < FactionConfig.JoinAPCost {
		return false, "AP 不足"
	}
	if !view.IsPlayer() {
		rep := residentActorRepresentative(view, silo)
		if rep == nil {
			return false, "无代表居民"
		}
		if cohortBeliefAboutFaction(silo, cohortOfRepresentative(silo, rep), rep, target.ID) < FactionConfig.AwarenessJoinFloor {
			return false, "尚未感知该派系"
		}
		if _, ok := joinRelationGate(silo, rep, target, agent); !ok {
			return false, "关系门槛未达"
		}
	}
	return true, ""
}

// residentActorRepresentative 取 RESIDENT actor 的居民指针 (关系闸输入)。
func residentActorRepresentative(view *ActorView, silo *model.Silo) *model.Resident {
	if view == nil || view.Ref.Kind != ActorResident {
		return nil
	}
	return residentByID(silo, view.Ref.ResidentID)
}

// joinableFactionsForRepresentative 居民代表当前可加入的派系列表 (无政治
// 归属 + 已感知 + 关系闸通过 + 亲缘可算)。NPC 加权候选用；不消耗 RNG。
// 无 agent 上下文，玩家领袖的目标不可解析部门而排除 (月度扫描带 agent 覆盖)。
func joinableFactionsForRepresentative(silo *model.Silo, view *ActorView) []*model.Faction {
	if silo == nil || view == nil || view.Ref.Kind != ActorResident {
		return nil
	}
	if fid := view.FactionID(); fid != nil && politicalFactionByID(silo, *fid) != nil {
		return nil
	}
	rep := residentActorRepresentative(view, silo)
	cohort := cohortOfRepresentative(silo, rep)
	if rep == nil || cohort == nil {
		return nil
	}
	var out []*model.Faction
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.Signature == unaffiliatedSignature {
			continue
		}
		if cohortBeliefAboutFaction(silo, cohort, rep, f.ID) < FactionConfig.AwarenessJoinFloor {
			continue
		}
		if _, ok := joinRelationGate(silo, rep, f, nil); !ok {
			continue
		}
		if _, ok := factionFeatureCentroid(silo, f, nil); !ok {
			continue
		}
		out = append(out, f)
	}
	return out
}

// cohortOfRepresentative 取代表居民所属 cohort (感知读取用)。
func cohortOfRepresentative(silo *model.Silo, rep *model.Resident) *model.PopulationCohort {
	if silo == nil || rep == nil || rep.CohortID == nil {
		return nil
	}
	for i := range silo.Cohorts {
		if silo.Cohorts[i].ID == *rep.CohortID {
			return &silo.Cohorts[i]
		}
	}
	return nil
}

// applyFactionJoinActor 处理 faction_join_actor mutation：权威资格判定与
// AP 扣除原子完成 (JS 侧只做同口径预检给出失败文案)。玩家直选不掷骰；
// 居民代表加入会带上其 cohort。目标派系 ID 由 mutation.IntValue 携带。
func (e *GameEngine) applyFactionJoinActor(silo *model.Silo, actor *ActorView, agent *model.Agent, targetID uint) {
	if actor == nil || silo == nil {
		return
	}
	target := factionByID(silo, targetID)
	if ok, reason := joinFactionEligible(actor, silo, target, agent); !ok {
		e.logf(fmt.Sprintf("%s 加入派系被拒绝：%s。", actor.Label(), reason))
		return
	}
	if actor.Ref.Kind != ActorPlayer && actor.Ref.Kind != ActorResident {
		// 部门 NPC 的成员流动走月度信念扫描，不走显式 JOIN 动作。
		e.logf(fmt.Sprintf("%s 无法以部门身份显式加入派系。", actor.Label()))
		return
	}
	actor.SetActionPoints(actor.ActionPoints() - FactionConfig.JoinAPCost)
	switch actor.Ref.Kind {
	case ActorPlayer:
		if a := actor.Agent(); a != nil {
			a.FactionID = uintPtr(target.ID)
		}
	case ActorResident:
		if r := residentByID(silo, actor.Ref.ResidentID); r != nil {
			r.FactionID = uintPtr(target.ID)
			if r.CohortID != nil {
				for i := range silo.Cohorts {
					if silo.Cohorts[i].ID == *r.CohortID {
						silo.Cohorts[i].FactionID = uintPtr(target.ID)
						break
					}
				}
			}
		}
	}
	e.logf(fmt.Sprintf("%s 加入「%s」。", actor.Label(), target.Name))
}
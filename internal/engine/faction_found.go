package engine

import (
	"fmt"

	"silo40/internal/model"
)

// foundedSignaturePrefix 事件发起派系的签名前缀 (24 篇 C6)。C7 起
// 发起是派系的唯一来源，MaintainFactions 只做生命周期维护。
const foundedSignaturePrefix = "founded:"

// foundFactionEligible 发起资格三闸 + 无政治派系 (24 篇 §1)：
// 派生威望 ≥ 50、意识形态强度 ≥ 0.25、AP ≥ 20；已代表/隶属政治派系者
// 不能再发起 (无阵营派系不算政治派系)。玩家与 NPC 同构，无冷却、
// 无野心硬门槛——野心只进 NPC 加权决策的权重。
func foundFactionEligible(view *ActorView, silo *model.Silo) (bool, string) {
	if view == nil || silo == nil {
		return false, "actor unavailable"
	}
	if view.PoliticalPrestige() < FactionConfig.FoundPrestigeThreshold {
		return false, "政治威望不足"
	}
	if view.IdeologicalIntensity() < FactionConfig.FoundIntensityThreshold {
		return false, "意识形态立场不够鲜明"
	}
	if view.ActionPoints() < FactionConfig.FoundAPCost {
		return false, "AP 不足"
	}
	if fid := view.FactionID(); fid != nil {
		for i := range silo.Factions {
			f := &silo.Factions[i]
			if f.ID == *fid && f.Signature != unaffiliatedSignature {
				return false, "已身处政治派系"
			}
		}
	}
	return true, ""
}

// applyFactionFoundActor 处理 faction_found_actor mutation：权威资格判定
// 与 AP 扣除在此原子完成 (JS 侧只做同口径预检给出失败文案)。创立零成员
// 派系 (ADR-0018)：代表=创始人=发起 Actor，立场取其思潮，筹备期人数
// 规则由 C7 的消亡管线执行。
func (e *GameEngine) applyFactionFoundActor(silo *model.Silo, actor *ActorView) {
	if actor == nil {
		return
	}
	if ok, reason := foundFactionEligible(actor, silo); !ok {
		e.logf(fmt.Sprintf("%s 的派系发起被拒绝：%s。", actor.Label(), reason))
		return
	}

	kind := string(actor.Ref.Kind)
	var actorID uint
	switch actor.Ref.Kind {
	case ActorPlayer:
		actorID = actor.Ref.AgentID
	case ActorNPC:
		actorID = actor.Ref.ProfessionID
	default:
		actorID = actor.Ref.ResidentID
	}

	nextID := uint(1)
	for i := range silo.Factions {
		if silo.Factions[i].ID >= nextID {
			nextID = silo.Factions[i].ID + 1
		}
	}

	ideo := actor.Ideologies()
	pf := ideo[model.IdeologyProForeign]
	loyalty := ideo[model.IdeologyLoyalty]
	stance := stanceTag(loyalty, pf)

	// 签名携带 pro_foreign 桶，与聚类派系同词表，结局仲裁按同一口径
	// 识别开放派系 (weighted_endings 读 Signature)。
	signature := fmt.Sprintf("%s%s:%d", foundedSignaturePrefix, kind, actorID)
	tags := []string{stance}
	switch {
	case pf >= 0.7:
		signature += "|" + model.IdeologyProForeign + ":High"
		tags = append(tags, model.IdeologyProForeign+":High")
	case pf >= 0.3:
		signature += "|" + model.IdeologyProForeign + ":Medium"
		tags = append(tags, model.IdeologyProForeign+":Medium")
	default:
		tags = append(tags, "ideology:status_quo")
	}

	name := actor.Label() + "的派系"
	faction := model.Faction{
		ID:                 nextID,
		SiloID:             silo.ID,
		Name:               name,
		Signature:          signature,
		Tags:               tags,
		TagStats:           map[string]int{},
		MemberCount:        0,
		RepresentativeKind: kind,
		RepresentativeID:   actorID,
		RepresentativeName: actor.Label(),
		FounderKind:        kind,
		FounderID:          actorID,
		FoundedYear:        silo.CurrentYear,
		FoundedMonth:       silo.CurrentMonth,
		Stance:             stance,
		Memory:             []string{fmt.Sprintf("founded:Y%dM%d", silo.CurrentYear, silo.CurrentMonth)},
	}
	silo.Factions = append(silo.Factions, faction)
	actor.SetActionPoints(actor.ActionPoints() - FactionConfig.FoundAPCost)

	if actor.Ref.Kind == ActorResident {
		for i := range silo.Residents {
			r := &silo.Residents[i]
			if r.ID == actorID {
				r.IsRepresentative = true
				r.FactionID = uintPtr(faction.ID)
				break
			}
		}
	}

	e.logf(fmt.Sprintf("%s 发起派系聚类事件，创立「%s」(零成员筹备期)。", actor.Label(), name))
}

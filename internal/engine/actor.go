package engine

import (
	"fmt"
	"sort"

	"silo40/internal/model"
)

// ============ Actor 抽象 (状态归一) ============
// 统一 Actor 模型：
// - PLAYER：特工 Agent (由 UI 控制)
// - NPC：部门 Profession (由 NpcBrain 决策层控制)
// 执行引擎只认识 ActorView (统一状态视图)，不区分来源。

type ActorKind string

const (
	ActorPlayer   ActorKind = "PLAYER"
	ActorNPC      ActorKind = "NPC"
	ActorResident ActorKind = "RESIDENT"
)

type ActorRef struct {
	Kind         ActorKind
	AgentID      uint // PLAYER：特工 id
	ProfessionID uint // 被控制的部门 id (NPC 必填；PLAYER 由 profession 推导)
	ResidentID   uint // RESIDENT：居民 id
}

func CreateActorRefForAgent(agent *model.Agent, silo *model.Silo) ActorRef {
	ref := ActorRef{Kind: ActorPlayer, AgentID: agent.ID}
	for _, p := range silo.Professions {
		if p.Name == agent.Profession {
			ref.ProfessionID = p.ID
			break
		}
	}
	return ref
}

func CreateActorRefForProfession(prof *model.Profession) ActorRef {
	return ActorRef{Kind: ActorNPC, ProfessionID: prof.ID}
}

func CreateActorRefForResident(resident *model.Resident) ActorRef {
	return ActorRef{Kind: ActorResident, ResidentID: resident.ID, ProfessionID: resident.ProfessionID}
}

// ActorView 统一状态视图：对 Agent / Profession 提供同构读写接口
type ActorView struct {
	Ref      ActorRef
	agent    *model.Agent
	prof     *model.Profession
	resident *model.Resident
	silo     *model.Silo
}

func CreateActorView(ref ActorRef, silo *model.Silo, agent *model.Agent) (*ActorView, error) {
	if ref.Kind == ActorPlayer {
		if agent == nil {
			return nil, fmt.Errorf("PLAYER actor requires agent reference")
		}
		return &ActorView{Ref: ref, agent: agent, silo: silo}, nil
	}
	if ref.Kind == ActorResident {
		for i := range silo.Residents {
			if silo.Residents[i].ID == ref.ResidentID {
				return &ActorView{Ref: ref, resident: &silo.Residents[i], silo: silo}, nil
			}
		}
		return nil, fmt.Errorf("resident actor not found: %d", ref.ResidentID)
	}
	for i := range silo.Professions {
		if silo.Professions[i].ID == ref.ProfessionID {
			return &ActorView{Ref: ref, prof: &silo.Professions[i], silo: silo}, nil
		}
	}
	return nil, fmt.Errorf("NPC actor profession not found: %d", ref.ProfessionID)
}

func (v *ActorView) IsPlayer() bool { return v.Ref.Kind == ActorPlayer }

// Agent exposes the underlying player agent (nil for non-player actors).
// Callers that re-resolve this view's Ref (e.g. ApplyMutation) need it:
// PLAYER refs cannot be reconstructed from the silo alone.
func (v *ActorView) Agent() *model.Agent { return v.agent }

// Profession 所属部门名称
func (v *ActorView) Profession() string {
	if v.agent != nil {
		return v.agent.Profession
	}
	if v.resident != nil {
		return v.resident.Profession
	}
	return v.prof.Name
}

// Label 日志称谓：玩家显示特工名，NPC 显示部门名
func (v *ActorView) Label() string {
	if v.agent != nil {
		if v.agent.Name != "" {
			return v.agent.Name
		}
		return v.agent.Profession
	}
	if v.resident != nil {
		if v.resident.Name != "" {
			return v.resident.Name
		}
		return v.resident.Profession
	}
	return v.prof.Name
}

// ============ 统一经济字段 (读写穿透到 Agent / Profession) ============

func (v *ActorView) ActionPoints() float64 {
	if v.agent != nil {
		return v.agent.ActionPoints
	}
	if v.resident != nil {
		return v.resident.ActionPoints
	}
	return v.prof.ActionPoints
}

func (v *ActorView) SetActionPoints(val float64) {
	if v.agent != nil {
		v.agent.ActionPoints = val
	} else if v.resident != nil {
		v.resident.ActionPoints = val
	} else {
		v.prof.ActionPoints = val
	}
}

func (v *ActorView) SuspicionLevel() float64 {
	if v.agent != nil {
		return v.agent.SuspicionLevel
	}
	if v.resident != nil {
		return v.resident.SuspicionLevel
	}
	return v.prof.SuspicionLevel
}

func (v *ActorView) SetSuspicionLevel(val float64) {
	if v.agent != nil {
		v.agent.SuspicionLevel = val
	} else if v.resident != nil {
		v.resident.SuspicionLevel = val
	} else {
		v.prof.SuspicionLevel = val
	}
}

// DerivedPrestige 政治威望全层派生公式 (24 篇 §2)：prestige = 50×(A+I)，
// 值域 0–100。Agent / Profession / Resident 同构，不再持有独立威望字段。
func DerivedPrestige(ambition, influence float64) float64 {
	return 50 * (ambition + influence)
}

func (v *ActorView) PoliticalPrestige() float64 {
	return DerivedPrestige(v.Ambition(), v.Influence())
}

// Ambition 野心：开局定死，仅事件可改 (读写穿透到 Agent / Profession / Resident)
func (v *ActorView) Ambition() float64 {
	if v.agent != nil {
		return v.agent.Ambition
	}
	if v.resident != nil {
		return v.resident.Ambition
	}
	return v.prof.Ambition
}

func (v *ActorView) SetAmbition(val float64) {
	if v.agent != nil {
		v.agent.Ambition = val
	} else if v.resident != nil {
		v.resident.Ambition = val
	} else {
		v.prof.Ambition = val
	}
}

// Influence 个人/部门影响力
func (v *ActorView) Influence() float64 {
	if v.agent != nil {
		return v.agent.Influence
	}
	if v.resident != nil {
		return v.resident.Influence
	}
	return v.prof.Influence
}

func (v *ActorView) SetInfluence(val float64) {
	if v.agent != nil {
		v.agent.Influence = val
	} else if v.resident != nil {
		v.resident.Influence = val
	} else {
		v.prof.Influence = val
	}
}

func (v *ActorView) PropagandaLevel() float64 {
	if v.agent != nil {
		return v.agent.PropagandaLevel
	}
	if v.resident != nil {
		return v.resident.PropagandaLevel
	}
	return v.prof.PropagandaLevel
}

func (v *ActorView) SetPropagandaLevel(val float64) {
	if v.agent != nil {
		v.agent.PropagandaLevel = val
	} else if v.resident != nil {
		v.resident.PropagandaLevel = val
	} else {
		v.prof.PropagandaLevel = val
	}
}

// Productivity 生产力：读写 Actor 所属部门的生产力
func (v *ActorView) Productivity() float64 {
	if v.prof != nil {
		return v.prof.Productivity
	}
	if v.resident != nil {
		for i := range v.silo.Professions {
			if v.silo.Professions[i].ID == v.resident.ProfessionID {
				return v.silo.Professions[i].Productivity
			}
		}
	}
	for i := range v.silo.Professions {
		if v.silo.Professions[i].Name == v.agent.Profession {
			return v.silo.Professions[i].Productivity
		}
	}
	return 1
}

func (v *ActorView) SetProductivity(val float64) {
	if v.prof != nil {
		v.prof.Productivity = val
		return
	}
	if v.resident != nil {
		for i := range v.silo.Professions {
			if v.silo.Professions[i].ID == v.resident.ProfessionID {
				v.silo.Professions[i].Productivity = val
				return
			}
		}
	}
	for i := range v.silo.Professions {
		if v.silo.Professions[i].Name == v.agent.Profession {
			v.silo.Professions[i].Productivity = val
			return
		}
	}
}

// PoliticalPoints 政治点数：仅玩家特工拥有 (NPC 恒为 0)
func (v *ActorView) PoliticalPoints() float64 {
	if v.agent != nil {
		return v.agent.PoliticalPoints
	}
	return 0
}

func (v *ActorView) SetPoliticalPoints(val float64) {
	if v.agent != nil {
		v.agent.PoliticalPoints = val
	}
}

func (v *ActorView) Traits() []string {
	if v.agent != nil {
		if v.agent.Traits == nil {
			v.agent.Traits = []string{}
		}
		return v.agent.Traits
	}
	if v.resident != nil {
		if v.resident.Tags == nil {
			v.resident.Tags = []string{}
		}
		return v.resident.Tags
	}
	if v.prof.Traits == nil {
		v.prof.Traits = []string{}
	}
	return v.prof.Traits
}

func (v *ActorView) KnownFragments() []string {
	if v.agent != nil {
		if v.agent.KnownFragments == nil {
			v.agent.KnownFragments = []string{}
		}
		return v.agent.KnownFragments
	}
	if v.resident != nil {
		if v.resident.KnownFragments == nil {
			v.resident.KnownFragments = []string{}
		}
		return v.resident.KnownFragments
	}
	if v.prof.KnownFragments == nil {
		v.prof.KnownFragments = []string{}
	}
	return v.prof.KnownFragments
}

func (v *ActorView) IsRepresentative() bool {
	if v.resident != nil {
		return v.resident.IsRepresentative
	}
	// ADR-0018：玩家与 NPC 同构——任代表的派系 (含零成员派系) 通过
	// RepresentativeKind=PLAYER/NPC 引用判定，不另设特殊分支。
	return v.representedFaction() != nil
}

func (v *ActorView) FactionID() *uint {
	if v.resident != nil {
		return v.resident.FactionID
	}
	// 玩家成员资格 (JOIN_FACTION) 优先；未加入时回退到任代表的派系 (发起/领导)。
	if v.agent != nil && v.agent.FactionID != nil {
		return v.agent.FactionID
	}
	if f := v.representedFaction(); f != nil {
		id := f.ID
		return &id
	}
	return nil
}

// representedFaction 该 Actor 任代表的派系 (PLAYER 按特工 id、NPC 按部门 id)。
func (v *ActorView) representedFaction() *model.Faction {
	if v.silo == nil {
		return nil
	}
	var kind string
	var id uint
	switch {
	case v.agent != nil:
		kind, id = string(ActorPlayer), v.agent.ID
	case v.prof != nil:
		kind, id = string(ActorNPC), v.prof.ID
	default:
		return nil
	}
	for i := range v.silo.Factions {
		f := &v.silo.Factions[i]
		if f.RepresentativeKind == kind && f.RepresentativeID == id {
			return f
		}
	}
	return nil
}

// Ideologies 意识形态向量：居民/部门读自身，玩家读所属部门的思潮
// (Agent 不持有独立 Ideologies，ADR-0018 同构口径)。
func (v *ActorView) Ideologies() map[string]float64 {
	if v.resident != nil {
		return v.resident.Ideologies
	}
	if v.prof != nil {
		return v.prof.Ideologies
	}
	if v.agent != nil && v.silo != nil {
		for i := range v.silo.Professions {
			if v.silo.Professions[i].Name == v.agent.Profession {
				return v.silo.Professions[i].Ideologies
			}
		}
	}
	return nil
}

// IdeologicalIntensity 意识形态强度 |pro_foreign−0.5| (发起资格输入)。
func (v *ActorView) IdeologicalIntensity() float64 {
	return ideologicalIntensity(v.Ideologies()[model.IdeologyProForeign])
}

func (v *ActorView) Relations() map[string]float64 {
	if v.resident != nil {
		return v.resident.Relations
	}
	if v.prof != nil {
		return v.prof.Relations
	}
	// Agent 的关系是 slice，需要特殊转换
	if v.agent != nil {
		rels := make(map[string]float64)
		for _, conn := range v.agent.Connections {
			for _, p := range v.silo.Professions {
				if p.ID == conn.ProfessionID {
					rels[p.Name] = conn.Value
					break
				}
			}
		}
		return rels
	}
	return nil
}

// ============ 人脉 (统一按部门 id 访问) ============

// ConnectionValues 该 Actor 对各地堡部门的人脉值数组 (用于平均威望计算)
func (v *ActorView) ConnectionValues() []float64 {
	if v.agent != nil {
		vals := make([]float64, 0, len(v.agent.Connections))
		for _, c := range v.agent.Connections {
			vals = append(vals, c.Value)
		}
		return vals
	}
	if v.resident != nil {
		return sortedRelationValues(v.resident.Relations)
	}
	return sortedRelationValues(v.prof.Relations)
}

// sortedRelationValues returns relation values in key order: callers sum the
// slice, and map iteration order would make the float sum nondeterministic.
func sortedRelationValues(rels map[string]float64) []float64 {
	keys := make([]string, 0, len(rels))
	for k := range rels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]float64, 0, len(keys))
	for _, k := range keys {
		vals = append(vals, rels[k])
	}
	return vals
}

func (v *ActorView) GetConnection(professionID uint) float64 {
	if v.agent != nil {
		for _, c := range v.agent.Connections {
			if c.ProfessionID == professionID {
				return c.Value
			}
		}
		return 0
	}
	if v.resident != nil {
		for i := range v.silo.Professions {
			if v.silo.Professions[i].ID == professionID {
				return v.resident.Relations[v.silo.Professions[i].Name]
			}
		}
		return 0
	}
	for i := range v.silo.Professions {
		if v.silo.Professions[i].ID == professionID {
			return v.prof.Relations[v.silo.Professions[i].Name]
		}
	}
	return 0
}

func (v *ActorView) SetConnection(professionID uint, value float64) {
	if v.agent != nil {
		if v.agent.Connections == nil {
			v.agent.Connections = []model.Connection{}
		}
		for i := range v.agent.Connections {
			if v.agent.Connections[i].ProfessionID == professionID {
				v.agent.Connections[i].Value = value
				return
			}
		}
		v.agent.Connections = append(v.agent.Connections, model.Connection{
			AgentID:      v.agent.ID,
			ProfessionID: professionID,
			Value:        value,
		})
		return
	}
	if v.resident != nil {
		for i := range v.silo.Professions {
			if v.silo.Professions[i].ID == professionID {
				if v.resident.Relations == nil {
					v.resident.Relations = map[string]float64{}
				}
				v.resident.Relations[v.silo.Professions[i].Name] = value
				return
			}
		}
		return
	}
	for i := range v.silo.Professions {
		if v.silo.Professions[i].ID == professionID {
			if v.prof.Relations == nil {
				v.prof.Relations = map[string]float64{}
			}
			v.prof.Relations[v.silo.Professions[i].Name] = value
			return
		}
	}
}

func (v *ActorView) AddConnection(professionID uint, delta float64) {
	cur := v.GetConnection(professionID) + delta
	if cur < 0 {
		cur = 0
	}
	if cur > 1.0 {
		cur = 1.0
	}
	v.SetConnection(professionID, cur)
}

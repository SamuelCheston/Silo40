package engine

import (
	"fmt"

	"silo40/internal/model"
)

// ============ Actor 抽象 (状态归一) ============
// 统一 Actor 模型：
// - PLAYER：特工 Agent (由 UI 控制)
// - NPC：部门 Profession (由 NpcBrain 决策层控制)
// 执行引擎只认识 ActorView (统一状态视图)，不区分来源。

type ActorKind string

const (
	ActorPlayer ActorKind = "PLAYER"
	ActorNPC    ActorKind = "NPC"
)

type ActorRef struct {
	Kind          ActorKind
	AgentID       uint // PLAYER：特工 id
	ProfessionID  uint // 被控制的部门 id (NPC 必填；PLAYER 由 profession 推导)
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

// ActorView 统一状态视图：对 Agent / Profession 提供同构读写接口
type ActorView struct {
	Ref   ActorRef
	agent *model.Agent
	prof  *model.Profession
	silo  *model.Silo
}

func CreateActorView(ref ActorRef, silo *model.Silo, agent *model.Agent) (*ActorView, error) {
	if ref.Kind == ActorPlayer {
		if agent == nil {
			return nil, fmt.Errorf("PLAYER actor requires agent reference")
		}
		return &ActorView{Ref: ref, agent: agent, silo: silo}, nil
	}
	for i := range silo.Professions {
		if silo.Professions[i].ID == ref.ProfessionID {
			return &ActorView{Ref: ref, prof: &silo.Professions[i], silo: silo}, nil
		}
	}
	return nil, fmt.Errorf("NPC actor profession not found: %d", ref.ProfessionID)
}

func (v *ActorView) IsPlayer() bool { return v.Ref.Kind == ActorPlayer }

// Profession 所属部门名称
func (v *ActorView) Profession() string {
	if v.agent != nil {
		return v.agent.Profession
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
	return v.prof.Name
}

// ============ 统一经济字段 (读写穿透到 Agent / Profession) ============

func (v *ActorView) ActionPoints() float64 {
	if v.agent != nil {
		return v.agent.ActionPoints
	}
	return v.prof.ActionPoints
}

func (v *ActorView) SetActionPoints(val float64) {
	if v.agent != nil {
		v.agent.ActionPoints = val
	} else {
		v.prof.ActionPoints = val
	}
}

func (v *ActorView) SuspicionLevel() float64 {
	if v.agent != nil {
		return v.agent.SuspicionLevel
	}
	return v.prof.SuspicionLevel
}

func (v *ActorView) SetSuspicionLevel(val float64) {
	if v.agent != nil {
		v.agent.SuspicionLevel = val
	} else {
		v.prof.SuspicionLevel = val
	}
}

func (v *ActorView) PoliticalPrestige() float64 {
	if v.agent != nil {
		return v.agent.PoliticalPrestige
	}
	return v.prof.PoliticalPrestige
}

func (v *ActorView) SetPoliticalPrestige(val float64) {
	if v.agent != nil {
		v.agent.PoliticalPrestige = val
	} else {
		v.prof.PoliticalPrestige = val
	}
}

func (v *ActorView) PropagandaLevel() float64 {
	if v.agent != nil {
		return v.agent.PropagandaLevel
	}
	return v.prof.PropagandaLevel
}

func (v *ActorView) SetPropagandaLevel(val float64) {
	if v.agent != nil {
		v.agent.PropagandaLevel = val
	} else {
		v.prof.PropagandaLevel = val
	}
}

func (v *ActorView) OrganizationFactor() float64 {
	if v.agent != nil {
		return v.agent.OrganizationFactor
	}
	if v.prof.OrganizationFactor == 0 {
		return 1.0
	}
	return v.prof.OrganizationFactor
}

func (v *ActorView) SetOrganizationFactor(val float64) {
	if v.agent != nil {
		v.agent.OrganizationFactor = val
	} else {
		v.prof.OrganizationFactor = val
	}
}

// Productivity 生产力：读写 Actor 所属部门的生产力
func (v *ActorView) Productivity() float64 {
	if v.prof != nil {
		return v.prof.Productivity
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
	if v.prof.KnownFragments == nil {
		v.prof.KnownFragments = []string{}
	}
	return v.prof.KnownFragments
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
	vals := make([]float64, 0, len(v.prof.Relations))
	for _, val := range v.prof.Relations {
		vals = append(vals, val)
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

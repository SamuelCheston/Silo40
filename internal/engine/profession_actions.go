package engine

import (
	"math"
	"math/rand"

	"silo40/internal/model"
)

// ============ 职业专属行动注册表 (数据驱动) ============
// 每个职业拥有 2 个专属行动。行动定义独立于执行管线，
// 玩家与 NPC 通过统一 Actor 管线 (executeActionInternal) 提交执行，
// AP 成本 / 怀疑度惩罚由 engine 统一结算，effect 只负责世界状态变更。

type ProfessionActionTargetType string

const (
	TargetNone     ProfessionActionTargetType = "NONE"
	TargetDept     ProfessionActionTargetType = "DEPT"
	TargetResource ProfessionActionTargetType = "RESOURCE"
)

type ProfessionAction struct {
	ID               string
	Profession       string
	Label            string
	Description      string
	APCost           float64
	TargetType       ProfessionActionTargetType
	SuspicionPenalty float64
	Effect           func(silo *model.Silo, view *ActorView, target string) model.ActionResult
}

var PROFESSION_ACTIONS = []*ProfessionAction{}

// ---- 通用数值辅助 ----

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func findDept(silo *model.Silo, name string) *model.Profession {
	for i := range silo.Professions {
		if silo.Professions[i].Name == name {
			return &silo.Professions[i]
		}
	}
	return nil
}

func addResource(silo *model.Silo, typ string, amount float64) {
	for i := range silo.Resources {
		if silo.Resources[i].Type == typ {
			silo.Resources[i].Amount += amount
			return
		}
	}
}

// gainFragment 从指定部门的信息碎片池中随机获取 count 个未知碎片给 Actor
func gainFragment(view *ActorView, deptName string, count int) []string {
	var gained []string
	var pool []string
	for _, f := range model.ALL_FRAGMENTS {
		if len(f) > len(deptName) && f[:len(deptName)] == deptName && f[len(deptName)] == '_' {
			pool = append(pool, f)
		}
	}
	known := map[string]bool{}
	for _, f := range view.KnownFragments() {
		known[f] = true
	}
	unknown := []string{}
	for _, f := range pool {
		if !known[f] {
			unknown = append(unknown, f)
		}
	}
	for i := 0; i < count && len(unknown) > 0; i++ {
		idx := rand.Intn(len(unknown))
		frag := unknown[idx]
		unknown = append(unknown[:idx], unknown[idx+1:]...)
		view.KnownFragments()
		// append to underlying slice
		switch v := view.agentOrProf(); {
		case v.agent != nil:
			v.agent.KnownFragments = append(v.agent.KnownFragments, frag)
		case v.resident != nil:
			v.resident.KnownFragments = append(v.resident.KnownFragments, frag)
		case v.prof != nil:
			v.prof.KnownFragments = append(v.prof.KnownFragments, frag)
		}
		gained = append(gained, frag)
	}
	return gained
}

// removeFragments 随机移除目标部门 count 个已掌握的碎片 (阻碍信息胜利)
func removeFragments(silo *model.Silo, deptName string, count int) []string {
	dept := findDept(silo, deptName)
	if dept == nil {
		return nil
	}
	pool := make([]string, len(dept.KnownFragments))
	copy(pool, dept.KnownFragments)
	var removed []string
	for i := 0; i < count && len(pool) > 0; i++ {
		idx := rand.Intn(len(pool))
		frag := pool[idx]
		pool = append(pool[:idx], pool[idx+1:]...)
		var kept []string
		for _, f := range dept.KnownFragments {
			if f != frag {
				kept = append(kept, f)
			}
		}
		dept.KnownFragments = kept
		removed = append(removed, frag)
	}
	return removed
}

type actorTarget struct {
	agent    *model.Agent
	prof     *model.Profession
	resident *model.Resident
}

func (v *ActorView) agentOrProf() actorTarget {
	if v.agent != nil {
		return actorTarget{agent: v.agent}
	}
	if v.resident != nil {
		return actorTarget{resident: v.resident}
	}
	return actorTarget{prof: v.prof}
}

func init() {
	PROFESSION_ACTIONS = []*ProfessionAction{
		// ================= Mayor (政治领袖) =================
		{
			ID: "MAYOR_PUBLIC_ADDRESS", Profession: "Mayor",
			Label:       "Public Address",
			Description: "Deliver a rousing speech to the silo. Legitimacy +6%, all departments panic -4%, cohesion +3%.",
			APCost:      15, TargetType: TargetNone, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				// 市长演讲通过提高全体人口忠诚度来间接提升合法性与凝聚力
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					c.Ideologies[model.IdeologyLoyalty] = clamp(c.Ideologies[model.IdeologyLoyalty]+0.06, 0, 1)
				}
				for i := range silo.Professions {
					silo.Professions[i].PanicValue = clamp(silo.Professions[i].PanicValue-0.04, 0, 1)
				}
				return model.ActionResult{Executed: true, Message: "Public address delivered. Population loyalty increased and panic eased."}
			},
		},
		{
			ID: "MAYOR_DIRECT_ORDER", Profession: "Mayor",
			Label:       "Direct Order",
			Description: "Issue an executive order to a department. Target productivity +10%, panic -5%.",
			APCost:      20, TargetType: TargetDept, SuspicionPenalty: 0.02,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.Productivity = clamp(dept.Productivity+0.10, 0, 1)
				dept.PanicValue = clamp(dept.PanicValue-0.05, 0, 1)
				return model.ActionResult{Executed: true, Message: "Executive order issued. " + dept.Name + " productivity improved."}
			},
		},

		// ================= Judicial (司法部) =================
		{
			ID: "JUDICIAL_SEARCH_WARRANT", Profession: "Judicial",
			Label:       "Search Warrant",
			Description: "Serve a search warrant on a department to seize intel. Gain 1 fragment; target panic +3%.",
			APCost:      15, TargetType: TargetDept, SuspicionPenalty: 0.02,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				gained := gainFragment(view, dept.Name, 1)
				dept.PanicValue = clamp(dept.PanicValue+0.03, 0, 1)
				if len(gained) == 0 {
					return model.ActionResult{Executed: false, Message: "Nothing new was found while searching " + dept.Name + "."}
				}
				return model.ActionResult{Executed: true, Message: "Search warrant executed. Seized intel on " + gained[0] + " from " + dept.Name + "."}
			},
		},
		{
			ID: "JUDICIAL_ARREST", Profession: "Judicial",
			Label:       "Arrest",
			Description: "Arrest key figures in a department. Target action points -20, panic +10%, legitimacy +4%.",
			APCost:      25, TargetType: TargetDept, SuspicionPenalty: 0.03,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.ActionPoints = math.Max(0, dept.ActionPoints-20)
				dept.PanicValue = clamp(dept.PanicValue+0.10, 0, 1)
				// 逮捕行动通过震慑提升整体忠诚度
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					c.Ideologies[model.IdeologyLoyalty] = clamp(c.Ideologies[model.IdeologyLoyalty]+0.04, 0, 1)
				}
				return model.ActionResult{Executed: true, Message: "Arrests carried out in " + dept.Name + ". The law is reaffirmed."}
			},
		},

		// ================= IT (IT部门) =================
		{
			ID: "IT_SURVEILLANCE", Profession: "IT",
			Label:       "Surveillance",
			Description: "Place a department under full surveillance. Connection +15% with the target (leverage of fear).",
			APCost:      15, TargetType: TargetDept, SuspicionPenalty: 0,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				view.AddConnection(dept.ID, 0.15)
				return model.ActionResult{Executed: true, Message: dept.Name + " is now under surveillance. You hold leverage over them."}
			},
		},
		{
			ID: "IT_COVER_UP", Profession: "IT",
			Label:       "Cover-Up",
			Description: "Erase sensitive records from a department. Remove 1-2 fragments (blocks the information victory); safeguard risk +3%.",
			APCost:      25, TargetType: TargetDept, SuspicionPenalty: 0,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				count := 1
				if rand.Float64() < 0.5 {
					count = 2
				}
				removed := removeFragments(silo, dept.Name, count)
				dept.PanicValue = clamp(dept.PanicValue+0.05, 0, 1)
				silo.SafeguardRisk = silo.SafeguardRisk + 0.03
				if len(removed) == 0 {
					return model.ActionResult{Executed: false, Message: dept.Name + " holds no records worth erasing."}
				}
				return model.ActionResult{Executed: true, Message: "Records erased from " + dept.Name + ": " + joinStr(removed, ", ") + ". Safeguard risk grows..."}
			},
		},

		// ================= Police (警察) =================
		{
			ID: "POLICE_INTERROGATE", Profession: "Police",
			Label:       "Interrogate",
			Description: "Interrogate detainees from a department. Gain 1 fragment; target panic +5%.",
			APCost:      15, TargetType: TargetDept, SuspicionPenalty: 0.02,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				gained := gainFragment(view, dept.Name, 1)
				dept.PanicValue = clamp(dept.PanicValue+0.05, 0, 1)
				if len(gained) == 0 {
					return model.ActionResult{Executed: false, Message: "Interrogation yielded nothing new from " + dept.Name + "."}
				}
				return model.ActionResult{Executed: true, Message: "Interrogation extracted intel on " + gained[0] + " from " + dept.Name + "."}
			},
		},
		{
			ID: "POLICE_CRACKDOWN", Profession: "Police",
			Label:       "Crackdown",
			Description: "Suppress a department by force. Target panic -15%, ideology -5%, productivity -3%.",
			APCost:      25, TargetType: TargetDept, SuspicionPenalty: 0.03,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.PanicValue = clamp(dept.PanicValue-0.15, 0, 1)
				// 镇压行动直接降低该部门下所有人口单元的激进思潮
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					if c.ProfessionID == dept.ID {
						c.Ideologies[model.IdeologyProForeign] = clamp(c.Ideologies[model.IdeologyProForeign]-0.05, 0, 1)
					}
				}
				dept.Productivity = clamp(dept.Productivity-0.03, 0, 1)
				return model.ActionResult{Executed: true, Message: "Crackdown executed in " + dept.Name + ". Order restored, at a cost."}
			},
		},

		// ================= Medical (医疗部) =================
		{
			ID: "MEDICAL_TREAT", Profession: "Medical",
			Label:       "Community Treatment",
			Description: "Deploy medics to a department. Target panic -12%, productivity +5%.",
			APCost:      15, TargetType: TargetDept, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.PanicValue = clamp(dept.PanicValue-0.12, 0, 1)
				dept.Productivity = clamp(dept.Productivity+0.05, 0, 1)
				return model.ActionResult{Executed: true, Message: "Medics deployed to " + dept.Name + ". Panic eased and health improved."}
			},
		},
		{
			ID: "MEDICAL_QUARANTINE", Profession: "Medical",
			Label:       "Quarantine",
			Description: "Quarantine a department \"for its own safety\". Target panic -20%, but productivity -12% and ideology -4%.",
			APCost:      20, TargetType: TargetDept, SuspicionPenalty: 0.02,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.PanicValue = clamp(dept.PanicValue-0.20, 0, 1)
				dept.Productivity = clamp(dept.Productivity-0.12, 0, 1)
				// 隔离行动降低该部门的激进思潮偏移
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					if c.ProfessionID == dept.ID {
						c.Ideologies[model.IdeologyProForeign] = clamp(c.Ideologies[model.IdeologyProForeign]-0.04, 0, 1)
					}
				}
				return model.ActionResult{Executed: true, Message: dept.Name + " placed under quarantine. The silence is oppressive."}
			},
		},

		// ================= Supply (供给部) =================
		{
			ID: "SUPPLY_RATION", Profession: "Supply",
			Label:       "Ration Allocation",
			Description: "Reallocate stockpiles. Add +1000 to a chosen resource (Energy / Materials / Supplies).",
			APCost:      15, TargetType: TargetResource, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				if target == "" {
					return model.ActionResult{Executed: false, Message: "Invalid resource target."}
				}
				addResource(silo, target, 1000)
				return model.ActionResult{Executed: true, Message: "Reallocated stockpiles. +1000 " + target + "."}
			},
		},
		{
			ID: "SUPPLY_SHELTER", Profession: "Supply",
			Label:       "Shelter",
			Description: "Smuggle a department under your protection. Target panic -10%, connection +15%, productivity +5%.",
			APCost:      20, TargetType: TargetDept, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				dept.PanicValue = clamp(dept.PanicValue-0.10, 0, 1)
				dept.Productivity = clamp(dept.Productivity+0.05, 0, 1)
				view.AddConnection(dept.ID, 0.15)
				return model.ActionResult{Executed: true, Message: dept.Name + " is now sheltered by the Supply network."}
			},
		},

		// ================= Mechanical (机械部) =================
		{
			ID: "MECHANICAL_OVERHAUL", Profession: "Mechanical",
			Label:       "Overhaul",
			Description: "Overhaul the silo machinery. Energy +500, Materials +200, own productivity +5%.",
			APCost:      15, TargetType: TargetNone, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				addResource(silo, "Energy", 500)
				addResource(silo, "Materials", 200)
				view.SetProductivity(clamp(view.Productivity()+0.05, 0, 1))
				return model.ActionResult{Executed: true, Message: "Machinery overhauled. Energy and materials production improved."}
			},
		},
		{
			ID: "MECHANICAL_PIPE_TAP", Profession: "Mechanical",
			Label:       "Pipe Tap",
			Description: "Eavesdrop through the pipes that carry every whisper. Gain 1 fragment from a department.",
			APCost:      15, TargetType: TargetDept, SuspicionPenalty: 0.02,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				gained := gainFragment(view, dept.Name, 1)
				if len(gained) == 0 {
					return model.ActionResult{Executed: false, Message: "The pipes carried nothing new about " + dept.Name + "."}
				}
				return model.ActionResult{Executed: true, Message: "Eavesdropped through the pipes. Learned about " + gained[0] + " from " + dept.Name + "."}
			},
		},

		// ================= Mines (矿工) =================
		{
			ID: "MINES_DEEP_EXCAVATION", Profession: "Mines",
			Label:       "Deep Excavation",
			Description: "Push the mines deeper. Materials +800, own productivity +5%.",
			APCost:      15, TargetType: TargetNone, SuspicionPenalty: 0.005,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				addResource(silo, "Materials", 800)
				view.SetProductivity(clamp(view.Productivity()+0.05, 0, 1))
				return model.ActionResult{Executed: true, Message: "Deep excavation completed. Materials reserves increased."}
			},
		},
		{
			ID: "MINES_TUNNEL_NETWORK", Profession: "Mines",
			Label:       "Tunnel Network",
			Description: "Spin a network through the lower tunnels. All commoner departments connection +10%, ideology +3%.",
			APCost:      20, TargetType: TargetNone, SuspicionPenalty: 0.005,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				for i := range silo.Professions {
					p := &silo.Professions[i]
					if p.ClassType == "COMMONER" {
						view.AddConnection(p.ID, 0.10)
						// 隧道网络在平民部门中传播激进思潮
						for j := range silo.Cohorts {
							c := &silo.Cohorts[j]
							if c.ProfessionID == p.ID {
								c.Ideologies[model.IdeologyProForeign] = clamp(c.Ideologies[model.IdeologyProForeign]+0.03, 0, 1)
							}
						}
					}
				}
				return model.ActionResult{Executed: true, Message: "The tunnel network hums with new alliances and whispered hopes."}
			},
		},

		// ================= Agricultural (农业) =================
		{
			ID: "AGRICULTURAL_INTENSIVE_HARVEST", Profession: "Agricultural",
			Label:       "Intensive Harvest",
			Description: "Work the hydroponics around the clock. Supplies +1500, own productivity +8%.",
			APCost:      15, TargetType: TargetNone, SuspicionPenalty: 0.01,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				addResource(silo, "Supplies", 1500)
				view.SetProductivity(clamp(view.Productivity()+0.08, 0, 1))
				return model.ActionResult{Executed: true, Message: "Intensive harvest completed. Supplies increased."}
			},
		},
		{
			ID: "AGRICULTURAL_FIELD_GOSSIP", Profession: "Agricultural",
			Label:       "Field Gossip",
			Description: "Let rumors ripen in the fields. Gain 1 fragment from a department; target ideology +3%.",
			APCost:      10, TargetType: TargetDept, SuspicionPenalty: 0.015,
			Effect: func(silo *model.Silo, view *ActorView, target string) model.ActionResult {
				dept := findDept(silo, target)
				if dept == nil {
					return model.ActionResult{Executed: false, Message: "Target department not found."}
				}
				gained := gainFragment(view, dept.Name, 1)
				// 田间八卦在该部门中增加激进思潮
				for i := range silo.Cohorts {
					c := &silo.Cohorts[i]
					if c.ProfessionID == dept.ID {
						c.Ideologies[model.IdeologyProForeign] = clamp(c.Ideologies[model.IdeologyProForeign]+0.03, 0, 1)
					}
				}
				if len(gained) == 0 {
					return model.ActionResult{Executed: false, Message: "The fields whispered nothing new about " + dept.Name + "."}
				}
				return model.ActionResult{Executed: true, Message: "Rumors spread from the fields. Heard about " + gained[0] + " from " + dept.Name + "."}
			},
		},
	}
}

// GetProfessionActions 按职业筛选行动定义
func GetProfessionActions(profession string) []*ProfessionAction {
	var out []*ProfessionAction
	for _, a := range PROFESSION_ACTIONS {
		if a.Profession == profession {
			out = append(out, a)
		}
	}
	return out
}

// GetProfessionAction 按 id 查找行动定义
func GetProfessionAction(id string) *ProfessionAction {
	for _, a := range PROFESSION_ACTIONS {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// GetProfessionActionMeta 返回职业行动元数据 (前端渲染用，不含执行逻辑)
func GetProfessionActionMeta(profession string) []model.ProfessionActionMeta {
	var out []model.ProfessionActionMeta
	for _, a := range GetProfessionActions(profession) {
		out = append(out, model.ProfessionActionMeta{
			ID:               a.ID,
			Profession:       a.Profession,
			Label:            a.Label,
			Description:      a.Description,
			APCost:           int(a.APCost),
			TargetType:       string(a.TargetType),
			SuspicionPenalty: a.SuspicionPenalty,
		})
	}
	return out
}

func joinStr(items []string, sep string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

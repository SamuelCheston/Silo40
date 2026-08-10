package engine

import (
	"math"
	"math/rand"

	"silo40/internal/model"
)

// ============ NPC 决策层 (效用函数先行) ============
// NpcBrain 只负责"决策"：根据 ActorView (统一状态视图) 与地堡世界状态，
// 在 AP 预算内挑选一个动作。决策结果通过 SubmitAction 进入统一执行管线。

type NpcDecision struct {
	Action    model.AgentAction
	Message   string
	LogChance float64
}

type WeightedDecision struct {
	decision NpcDecision
	weight   float64
}

type NpcBrain struct{}

// FREQUENCY_PER_YEAR 每年尝试行动的平均次数 (按 deltaYears 缩放)
const FREQUENCY_PER_YEAR = 1.2

// Decide 决策入口：返回待提交动作；无可行动作或概率未命中时返回 nil
func (NpcBrain) Decide(view *ActorView, silo *model.Silo, deltaYears float64) *NpcDecision {
	attemptChance := math.Min(1.0, FREQUENCY_PER_YEAR*deltaYears)
	if rand.Float64() > attemptChance {
		return nil
	}

	candidates := NpcBrain{}.scoreCandidates(view, silo)
	if len(candidates) == 0 {
		return nil
	}

	total := 0.0
	for _, c := range candidates {
		total += c.weight
	}
	roll := rand.Float64() * total
	for _, c := range candidates {
		roll -= c.weight
		if roll <= 0 {
			d := c.decision
			return &d
		}
	}
	d := candidates[len(candidates)-1].decision
	return &d
}

func (NpcBrain) scoreCandidates(view *ActorView, silo *model.Silo) []WeightedDecision {
	var candidates []WeightedDecision
	ap := view.ActionPoints()
	ownID := view.Ref.ProfessionID
	depts := silo.Professions

	add := func(typ model.AgentActionType, weight float64, message string, logChance float64) {
		cost := model.ACTION_COSTS[typ]
		if cost > ap {
			return // 决策前预算检查：只提交可执行动作
		}
		if weight < 0 {
			weight = 0
		}
		candidates = append(candidates, WeightedDecision{
			decision: NpcDecision{
				Action:    model.AgentAction{Type: typ, Cost: cost},
				Message:   message,
				LogChance: logChance,
			},
			weight: weight,
		})
	}

	// --- GATHER_INFO：还有未知目标碎片时搜集 ---
	var gatherable []*model.Profession
	known := map[string]bool{}
	for _, f := range view.KnownFragments() {
		known[f] = true
	}
	for i := range depts {
		d := &depts[i]
		prefix := d.Name + "_"
		for _, f := range model.ALL_FRAGMENTS {
			if len(f) > len(prefix) && f[:len(prefix)] == prefix && !known[f] {
				gatherable = append(gatherable, d)
				break
			}
		}
	}
	if len(gatherable) > 0 {
		target := gatherable[rand.Intn(len(gatherable))]
		knowRatio := float64(len(view.KnownFragments())) / float64(len(model.ALL_FRAGMENTS))
		add(model.ActionGatherInfo, 1.5+(1-knowRatio)*1.5, "is secretly gathering intel on "+target.Name+".", 0.4)
	}

	// --- SHARE_INFO：有高信任盟友且持有碎片时共享 ---
	var allies []*model.Profession
	for i := range depts {
		d := &depts[i]
		if d.ID != ownID && view.GetConnection(d.ID) >= 0.8 {
			allies = append(allies, d)
		}
	}
	if len(view.KnownFragments()) > 0 && len(allies) > 0 {
		target := allies[rand.Intn(len(allies))]
		frag := view.KnownFragments()[rand.Intn(len(view.KnownFragments()))]
		add(model.ActionShareInfo, 1.2, "confidentially shared "+frag+" secrets with "+target.Name+".", 0.8)
	}

	// --- BUILD_CONNECTION：人脉薄弱时结交 ---
	var weakTargets []*model.Profession
	for i := range depts {
		d := &depts[i]
		if d.ID != ownID && view.GetConnection(d.ID) < 0.8 {
			weakTargets = append(weakTargets, d)
		}
	}
	if len(weakTargets) > 0 {
		weakest := weakTargets[0]
		for _, d := range weakTargets[1:] {
			if view.GetConnection(d.ID) < view.GetConnection(weakest.ID) {
				weakest = d
			}
		}
		deficit := 0.8 - view.GetConnection(weakest.ID)
		add(model.ActionBuildConnection, 1.0+deficit*2.0, "strengthened ties with "+weakest.Name+".", 0.5)
	}

	// --- CONDUCT_PROPAGANDA：宣传力度不足时宣传 ---
	if view.PropagandaLevel() < 3.0 {
		add(model.ActionConductPropaganda, 0.8, "conducted propaganda to boost influence.", 0.4)
	}

	// --- INCITE_REBELLION：平民恐慌严重且自身怀疑度低时煽动 ---
	avgPanic := 0.0
	hasCommoners := false
	if len(depts) > 0 {
		for i := range depts {
			avgPanic += depts[i].PanicValue
			if depts[i].ClassType == "COMMONER" {
				hasCommoners = true
			}
		}
		avgPanic /= float64(len(depts))
	}
	if hasCommoners && avgPanic > 0.4 && view.SuspicionLevel() < 0.5 {
		add(model.ActionInciteRebellion, 0.8, "incited unrest among the commoners.", 0.3)
	}

	// --- 职业专属行动：按职业注册表挑选，走统一执行管线 ---
	resTypes := []string{"Energy", "Materials", "Supplies"}
	for _, def := range GetProfessionActions(view.Profession()) {
		if def.APCost > ap {
			continue
		}
		action := model.AgentAction{Type: model.ActionProfession, ProfessionAction: def.ID, Cost: def.APCost}
		switch def.TargetType {
		case TargetDept:
			var others []*model.Profession
			for i := range depts {
				if depts[i].ID != ownID {
					others = append(others, &depts[i])
				}
			}
			if len(others) == 0 {
				continue
			}
			action.TargetDept = others[rand.Intn(len(others))].Name
		case TargetResource:
			action.ResourceTarget = resTypes[rand.Intn(len(resTypes))]
		}
		candidates = append(candidates, WeightedDecision{
			decision: NpcDecision{
				Action:    action,
				Message:   "performed " + def.Label + ".",
				LogChance: 0.35,
			},
			weight: 0.8,
		})
	}

	return candidates
}

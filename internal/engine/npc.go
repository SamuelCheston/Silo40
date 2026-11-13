package engine

import (
	"math"

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

type NpcBrain struct {
	// policy optionally advises decisions (LLM layer). nil = original
	// algorithm bit-for-bit. Set via GameEngine.NpcPolicy.
	policy NpcPolicy
}

// FREQUENCY_PER_YEAR 每年尝试行动的平均次数 (按 deltaYears 缩放)
const FREQUENCY_PER_YEAR = 1.2

// CandidateInfo is the exported description of one weighted candidate,
// handed to NpcPolicy without exposing internal structs.
type CandidateInfo struct {
	Index            int
	Type             model.AgentActionType
	ProfessionAction string // set when Type == ActionProfession
	TargetDept       string
	ResourceTarget   string
	FragmentIds      []string // set for SHARE_INFO: the fragments to share
	APCost           float64
	BaseWeight       float64
}

// NpcAdvice carries optional advice for one decision point. Zero value
// (nil maps / nil Force) leaves the original weights and sampling intact.
type NpcAdvice struct {
	// Channel B: willingness factor per action type. Values are clamped to
	// [AdviceFactorMin, AdviceFactorMax] and multiply the base weight.
	Factors map[model.AgentActionType]float64
	// Channel A: weight multiplier per candidate index. Values are clamped
	// to [AdviceFactorMin, AdviceFactorMax].
	Scores map[int]float64
	// Force pins the chosen candidate index (deterministic replay). The
	// weighted rng draw is still consumed so the rng stream stays intact.
	Force *int
}

// Advice clamp bounds: per-channel multipliers and their combined product.
const (
	AdviceFactorMin   = 0.5
	AdviceFactorMax   = 2.0
	AdviceCombinedMin = 0.25
	AdviceCombinedMax = 4.0
)

// NpcPolicy advises NPC decisions. Implementations must not block, must not
// consume the engine rng, and must not mutate view/silo/candidates.
type NpcPolicy interface {
	// Advise returns advice for this decision point, or nil for "original
	// algorithm". Called after candidates are scored, before sampling.
	Advise(view *ActorView, silo *model.Silo, candidates []CandidateInfo) *NpcAdvice
	// Committed reports the final chosen candidate index (after any Force
	// override) so the policy can record the decision.
	Committed(view *ActorView, silo *model.Silo, candidates []CandidateInfo, advice *NpcAdvice, chosenIndex int)
}

// ExecutionObserver is an optional NpcPolicy extension: implementations learn
// whether each committed NPC action actually executed (SubmitAction can still
// fail after the pick). The agent-operator recorder uses it to label training
// records with the true execution outcome. Called right after SubmitAction,
// in decision order; implementations must not block or touch the rng.
type ExecutionObserver interface {
	NpcExecuted(view *ActorView, silo *model.Silo, action model.AgentAction, result model.ActionResult)
}

// Decide 决策入口：返回待提交动作；无可行动作或概率未命中时返回 nil
func (b NpcBrain) Decide(view *ActorView, silo *model.Silo, deltaYears float64) *NpcDecision {
	attemptChance := math.Min(1.0, FREQUENCY_PER_YEAR*deltaYears)
	if rng.Float64() > attemptChance {
		return nil
	}

	candidates := b.scoreCandidates(view, silo, func(n int) int { return rng.Intn(n) })
	if len(candidates) == 0 {
		return nil
	}

	// Optional advice hook: never blocks, never consumes the engine rng.
	var advice *NpcAdvice
	var infos []CandidateInfo
	if b.policy != nil {
		infos = make([]CandidateInfo, len(candidates))
		for i, c := range candidates {
			infos[i] = CandidateInfo{
				Index:            i,
				Type:             c.decision.Action.Type,
				ProfessionAction: c.decision.Action.ProfessionAction,
				TargetDept:       c.decision.Action.TargetDept,
				ResourceTarget:   c.decision.Action.ResourceTarget,
				FragmentIds:      c.decision.Action.FragmentIds,
				APCost:           c.decision.Action.Cost,
				BaseWeight:       c.weight,
			}
		}
		advice = b.policy.Advise(view, silo, infos)
		applyAdvice(candidates, advice)
	}

	total := 0.0
	for _, c := range candidates {
		total += c.weight
	}
	// The draw is consumed even when Force overrides the pick, keeping the
	// rng stream identical with and without advice (determinism contract).
	roll := rng.Float64() * total
	chosen := len(candidates) - 1
	for i, c := range candidates {
		roll -= c.weight
		if roll <= 0 {
			chosen = i
			break
		}
	}
	if advice != nil && advice.Force != nil && *advice.Force >= 0 && *advice.Force < len(candidates) {
		chosen = *advice.Force
	}
	if b.policy != nil {
		b.policy.Committed(view, silo, infos, advice, chosen)
	}
	d := candidates[chosen].decision
	return &d
}

// PredictCandidates computes the candidate set an actor would face on a
// decision attempt, purely from state: no rng is consumed and targets are
// picked deterministically (first eligible), so predicted targets may differ
// from the ones the live draw eventually chooses. Used by the LLM prefetch
// layer, which fingerprints candidates by type sequence only.
func PredictCandidates(view *ActorView, silo *model.Silo) []CandidateInfo {
	cands := NpcBrain{}.scoreCandidates(view, silo, func(n int) int { return 0 })
	infos := make([]CandidateInfo, len(cands))
	for i, c := range cands {
		infos[i] = CandidateInfo{
			Index:            i,
			Type:             c.decision.Action.Type,
			ProfessionAction: c.decision.Action.ProfessionAction,
			TargetDept:       c.decision.Action.TargetDept,
			ResourceTarget:   c.decision.Action.ResourceTarget,
			FragmentIds:      c.decision.Action.FragmentIds,
			APCost:           c.decision.Action.Cost,
			BaseWeight:       c.weight,
		}
	}
	return infos
}

// DecisionActors enumerates every actor that may reach a decision point in
// RunNpcTurn (profession NPCs except the player's, plus alive resident
// representatives), in RunNpcTurn order. It does not consume rng. Used by
// the LLM prefetch layer to predict next month's decision points.
func DecisionActors(silo *model.Silo, agent *model.Agent) []*ActorView {
	if silo == nil {
		return nil
	}
	var views []*ActorView
	for i := range silo.Professions {
		p := &silo.Professions[i]
		if agent != nil && p.Name == agent.Profession {
			continue
		}
		if view, err := CreateActorView(CreateActorRefForProfession(p), silo, nil); err == nil {
			views = append(views, view)
		}
	}
	for i := range silo.Residents {
		res := &silo.Residents[i]
		if !res.Alive || !res.IsRepresentative {
			continue
		}
		if view, err := CreateActorView(CreateActorRefForResident(res), silo, nil); err == nil {
			views = append(views, view)
		}
	}
	return views
}

// applyAdvice multiplies candidate weights by the clamped advice channels.
// nil advice (or empty channels) leaves weights untouched.
func applyAdvice(candidates []WeightedDecision, advice *NpcAdvice) {
	if advice == nil {
		return
	}
	for i := range candidates {
		mult := 1.0
		if f, ok := advice.Factors[candidates[i].decision.Action.Type]; ok {
			mult *= clampAdviceFactor(f)
		}
		if s, ok := advice.Scores[i]; ok {
			mult *= clampAdviceFactor(s)
		}
		if mult == 1.0 {
			continue
		}
		if mult < AdviceCombinedMin {
			mult = AdviceCombinedMin
		}
		if mult > AdviceCombinedMax {
			mult = AdviceCombinedMax
		}
		candidates[i].weight *= mult
	}
}

func clampAdviceFactor(v float64) float64 {
	if v < AdviceFactorMin {
		return AdviceFactorMin
	}
	if v > AdviceFactorMax {
		return AdviceFactorMax
	}
	return v
}

// scoreCandidates builds the weighted candidate list. pick selects random
// targets: the live path passes the engine rng (consuming it in the contract
// order); PredictCandidates passes a deterministic picker so prefetching
// never touches the rng stream (R1).
func (NpcBrain) scoreCandidates(view *ActorView, silo *model.Silo, pick func(n int) int) []WeightedDecision {
	var candidates []WeightedDecision
	ap := view.ActionPoints()
	ownID := view.Ref.ProfessionID
	depts := silo.Professions

	// add builds a candidate whose action carries targetDept: the execution
	// handlers (events/mechanics/common_actions.js) require the target on
	// the action itself. Candidates whose target was picked by rng here must
	// pass it through, otherwise the chosen action fails at execution time.
	add := func(typ model.AgentActionType, weight float64, message string, logChance float64, targetDept string) {
		cost := model.ACTION_COSTS[typ]
		if cost > ap {
			return // 决策前预算检查：只提交可执行动作
		}
		if weight < 0 {
			weight = 0
		}
		candidates = append(candidates, WeightedDecision{
			decision: NpcDecision{
				Action:    model.AgentAction{Type: typ, Cost: cost, TargetDept: targetDept},
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
		target := gatherable[pick(len(gatherable))]
		knowRatio := float64(len(view.KnownFragments())) / float64(len(model.ALL_FRAGMENTS))
		add(model.ActionGatherInfo, 1.5+(1-knowRatio)*1.5, "is secretly gathering intel on "+target.Name+".", 0.4, target.Name)
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
		target := allies[pick(len(allies))]
		frag := view.KnownFragments()[pick(len(view.KnownFragments()))]
		cost := model.ACTION_COSTS[model.ActionShareInfo]
		if cost <= ap {
			candidates = append(candidates, WeightedDecision{
				decision: NpcDecision{
					// The handler requires both the target and the shared
					// fragment ids on the action.
					Action: model.AgentAction{
						Type: model.ActionShareInfo, Cost: cost,
						TargetDept: target.Name, FragmentIds: []string{frag},
					},
					Message:   "confidentially shared " + frag + " secrets with " + target.Name + ".",
					LogChance: 0.8,
				},
				weight: 1.2,
			})
		}
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
		add(model.ActionBuildConnection, 1.0+deficit*2.0, "strengthened ties with "+weakest.Name+".", 0.5, weakest.Name)
	}

	// --- CONDUCT_PROPAGANDA：宣传力度不足时宣传 ---
	if view.PropagandaLevel() < 3.0 {
		add(model.ActionConductPropaganda, 0.8, "conducted propaganda to boost influence.", 0.4, "")
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
		add(model.ActionInciteRebellion, 0.8, "incited unrest among the commoners.", 0.3, "")
	}

	// --- PUBLICIZE_FACTION：身为领袖且阵营未公开时 ---
	if view.IsRepresentative() && view.FactionID() != nil {
		var myFaction *model.Faction
		fid := *view.FactionID()
		for i := range silo.Factions {
			if silo.Factions[i].ID == fid {
				myFaction = &silo.Factions[i]
				break
			}
		}
		if myFaction != nil && !myFaction.IsPublic {
			// 领袖非常有动力公开自己的阵营
			add(model.ActionPublicizeFaction, 2.5, "solemnly publicized the existence of "+myFaction.Name+" to the entire silo.", 1.0, "")
		}
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
			action.TargetDept = others[pick(len(others))].Name
		case TargetResource:
			action.ResourceTarget = resTypes[pick(len(resTypes))]
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

package service

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 19-opening-variability §4.2/§6.4 的隐秘行事实测矩阵：
//   - 三档驻地原型下无声探查的可见性/门控；
//   - UNSAMPLED 实测（零怀疑、2× AP 口径，与普通探查同种子对照）；
//   - 痕迹不豁免（失败探查照旧抬升目标戒备）；
//   - §6.4 附加项：隐秘行事 × 每原型 ≥1 个《共证》通过组合（真实建档打到
//     INFORMATION 结局，策略实际使用无声探查）。
//
// 全部状态经真实动作与真实月度推进产生；固定种子可复现，不引入新随机源。

// TestSilentProbeVisibleAcrossArchetypes locks §6.4 三档可见性：三个原型覆写
// 下，隐秘行事特工开局均可见可用无声探查；无特质均不可见、执行被拒。
func TestSilentProbeVisibleAcrossArchetypes(t *testing.T) {
	for _, archetype := range []string{engine.ArchetypeBedrock, engine.ArchetypeRift, engine.ArchetypeFamine} {
		t.Run(archetype, func(t *testing.T) {
			svc := newGatedService(t, filepath.Join(t.TempDir(), "silent-matrix-shadowy.db"))
			if _, err := svc.CreateGame(archetypeRequest(201, archetype, "shadowy")); err != nil {
				t.Fatalf("create game: %v", err)
			}
			if svc.silo.Archetype != archetype {
				t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, archetype)
			}
			meta := actionByID(svc.availablePlayerActions(), "action_silent_probe")
			if meta == nil || meta.Visibility != model.ActionAvailable || !meta.Enabled {
				t.Fatalf("silent probe must be visible and available for shadowy agents on %s: %+v", archetype, meta)
			}
			outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"})
			if err != nil || !outcome.Result.Executed {
				t.Fatalf("silent probe execution rejected on %s: %v %+v", archetype, err, outcome.Result)
			}

			plain := newGatedService(t, filepath.Join(t.TempDir(), "silent-matrix-plain.db"))
			if _, err := plain.CreateGame(archetypeRequest(201, archetype)); err != nil {
				t.Fatalf("create game: %v", err)
			}
			if meta := actionByID(plain.availablePlayerActions(), "action_silent_probe"); meta != nil {
				t.Fatalf("silent probe must stay hidden without the trait on %s: %+v", archetype, meta)
			}
			if _, err := plain.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"}); err == nil || !strings.Contains(err.Error(), "not been unlocked") {
				t.Fatalf("silent probe bypass without trait must be rejected on %s: %v", archetype, err)
			}
		})
	}
}

// TestSilentProbeUnsampledAndDoubleCost locks §4.2 UNSAMPLED 实测：同一固定
// 种子下，司法特工对非 Mines 部门（SAMPLED 域）各做 6 次跨域探查——无声探查
// 每次结算后怀疑度恒为 0，普通探查每次结算后怀疑度 > 0；两路 AP 扣除分别按
// resolveActionCost 权威口径的 2× 与 1× 结算。同种子建档使两路 RNG 流逐位一致
// （成败序列相同），差异只来自检测分类。
func TestSilentProbeUnsampledAndDoubleCost(t *testing.T) {
	const probeCount = 6
	// Judicial（SAMPLED 职业）探查的非 Mines 目标：检测分类结构上为 SAMPLED。
	targets := []string{"Mayor", "IT", "Mayor", "IT", "Mayor", "IT"}
	for _, archetype := range []string{engine.ArchetypeBedrock, engine.ArchetypeRift, engine.ArchetypeFamine} {
		t.Run(archetype+"/silent", func(t *testing.T) {
			svc := newGatedService(t, filepath.Join(t.TempDir(), "silent-unsampled.db"))
			if _, err := svc.CreateGame(archetypeRequest(201, archetype, "shadowy")); err != nil {
				t.Fatalf("create game: %v", err)
			}
			for i := 0; i < probeCount; i++ {
				target := targets[i]
				silentCost := resolveProbeCost(t, svc, model.ActionSilentProbe, target)
				gatherCost := resolveProbeCost(t, svc, model.ActionGatherInfo, target)
				if silentCost != 2*gatherCost {
					t.Fatalf("silent probe cost = %v, want 2× gather cost %v (%s)", silentCost, gatherCost, target)
				}
				apBefore := svc.agent.ActionPoints
				outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: target})
				if err != nil || !outcome.Result.Executed {
					t.Fatalf("silent probe %d on %s failed: %v %+v", i+1, target, err, outcome.Result)
				}
				if got := apBefore - svc.agent.ActionPoints; got != silentCost {
					t.Fatalf("silent probe %d charged %v AP, want %v", i+1, got, silentCost)
				}
				if svc.agent.SuspicionLevel != 0 {
					t.Fatalf("silent probe %d must stay UNSAMPLED (zero suspicion), got %v", i+1, svc.agent.SuspicionLevel)
				}
			}
		})
		t.Run(archetype+"/sampled", func(t *testing.T) {
			compare := newGatedService(t, filepath.Join(t.TempDir(), "silent-sampled.db"))
			if _, err := compare.CreateGame(archetypeRequest(201, archetype)); err != nil {
				t.Fatalf("create game: %v", err)
			}
			for i := 0; i < probeCount; i++ {
				target := targets[i]
				gatherCost := resolveProbeCost(t, compare, model.ActionGatherInfo, target)
				apBefore := compare.agent.ActionPoints
				outcome, err := compare.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: target})
				if err != nil || !outcome.Result.Executed {
					t.Fatalf("gather %d on %s failed: %v %+v", i+1, target, err, outcome.Result)
				}
				if got := apBefore - compare.agent.ActionPoints; got != gatherCost {
					t.Fatalf("gather %d charged %v AP, want %v", i+1, got, gatherCost)
				}
				if compare.agent.SuspicionLevel <= 0 {
					t.Fatalf("ordinary gather %d by Judicial must be SAMPLED and raise suspicion, got %v", i+1, compare.agent.SuspicionLevel)
				}
			}
		})
	}
}

// TestSilentProbeFailureStillRaisesAlertness locks §4.2 边界「你不被怀疑，但
// 动作仍有痕迹」：无声探查失败时目标部门戒备照旧抬升（不豁免），且失败本身
// 也不产生怀疑。失败不是白盒构造——逐种子逐目标真实执行，取第一次真实发生
// 的失败断言其后果；窗口内若无失败则扩窗而不是放宽断言。
func TestSilentProbeFailureStillRaisesAlertness(t *testing.T) {
	cfg := engine.DefaultInformationConfig()
	// 非 Mines 跨域目标（Mines 结构性 UNSAMPLED，不作为本断言对象）。
	probeTargets := []string{"Mayor", "IT", "Supply", "Mechanical", "Medical", "Police", "Agricultural"}
	for seed := int64(201); seed < 221; seed++ {
		svc := newGatedService(t, filepath.Join(t.TempDir(), fmt.Sprintf("silent-alertness-%d.db", seed)))
		if _, err := svc.CreateGame(archetypeRequest(seed, engine.ArchetypeBedrock, "shadowy")); err != nil {
			t.Fatalf("create game (seed %d): %v", seed, err)
		}
		for _, target := range probeTargets {
			before := departmentAlertness(t, svc, target)
			outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: target})
			if err != nil || !outcome.Result.Executed {
				t.Fatalf("silent probe on %s (seed %d): %v %+v", target, seed, err, outcome.Result)
			}
			after := departmentAlertness(t, svc, target)
			if after <= before {
				continue // 该次探查成功（真实结果），换下一个目标
			}
			if delta := after - before; delta != cfg.AlertOnFailedProbe {
				t.Fatalf("failed silent probe raised %s alertness by %v, want exactly %v", target, delta, cfg.AlertOnFailedProbe)
			}
			if svc.agent.SuspicionLevel != 0 {
				t.Fatalf("failed silent probe must stay UNSAMPLED, suspicion = %v", svc.agent.SuspicionLevel)
			}
			if !strings.Contains(outcome.Result.Message, "戒备上升") {
				t.Fatalf("failed silent probe receipt must report the alertness trace: %q", outcome.Result.Message)
			}
			return
		}
	}
	t.Fatal("no silent probe failure observed in the seed window; widen the window instead of weakening the assertion")
}

func departmentAlertness(t *testing.T, svc *GameService, name string) float64 {
	t.Helper()
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == name {
			return svc.silo.Professions[i].Alertness
		}
	}
	t.Fatalf("department %s not found", name)
	return 0
}

// TestSilentProbeInformationEndingMatrix locks §6.4 附加项：隐秘行事 × 每原型
// ≥1 个《共证》通过组合。原型覆写钉档，从建档真实打到 INFORMATION 结局；策略
// 是参考路线（TestInformationEndingPlayableFromNewGame）的原型适配版：
//   - 开局与加深确信用无声探查（免怀疑的跨域取材），流通/核验/发布照旧；
//   - 发布尊重 2 个月冷却（内容事件 canTrigger 的真实守卫）；
//   - 全部门各 3 个健康槽位落齐后停手等待渗透成熟（淡忘未启用，渗透是纯
//     时间门），把动作痕迹压到最低。
//
// 组合实测（种子/职业/原型/结算月，第 5 批反制落地重调后）：
//   - 2401 / Judicial / bedrock（磐石）→ 129-05（IT 反击怀疑压力最大，靠提速跑赢）
//   - 2404 / Judicial / famine（荒年）→ 129-09
//   - 2424 / Judicial / rift（裂隙）→ 129-09（裂隙叛乱放大轴下只有部分种子可赢，
//     见报告：秩序波动轴是裂隙档的卡点）
//
// 流通循环已移植参考路线同套门（成本门/维护优先/半成品续播/日历播种截止），
// 结算月较反制前基线整体后移 ≈1-2 年，全部仍在 130-01 地平线内。
func TestSilentProbeInformationEndingMatrix(t *testing.T) {
	cases := []struct {
		archetype string
		seed      int64
	}{
		{engine.ArchetypeBedrock, 2401},
		{engine.ArchetypeFamine, 2404},
		{engine.ArchetypeRift, 2424},
	}
	for _, tc := range cases {
		t.Run(tc.archetype, func(t *testing.T) {
			runSilentProbeInformationRoute(t, tc.archetype, tc.seed)
		})
	}
}

func runSilentProbeInformationRoute(t *testing.T, archetype string, seed int64) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "silent-information.db"))
	req := threeSiloRequest(seed)
	req.Profession = "Judicial"
	req.Archetype = archetype
	req.TraitIds = []string{"shadowy"}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	silentUses := 0
	circulationActions := 0
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v, known=%v)", action, err, svc.agent.ActionPoints, svc.agent.KnownFragments)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
		if action.Type == model.ActionSilentProbe {
			silentUses++
		}
		circulationActions++
	}
	tryExecute := func(action model.AgentAction) bool {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil || !outcome.Result.Executed {
			return false
		}
		if action.Type == model.ActionSilentProbe {
			silentUses++
		}
		circulationActions++
		return true
	}
	// 服务端权威成本口径（含规范冲突上浮；无声探查 = 2×）。
	resolvedCost := func(actionType model.AgentActionType, target string) float64 {
		t.Helper()
		action := model.AgentAction{Type: actionType, TargetDept: target}
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolve cost for %s: %v", actionType, err)
		}
		return action.Cost
	}

	// 开局：无声探查越域取材（计入首观察键，打开 first_signal）。
	execute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"})
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("silent probe did not open the line: %s", svc.world.ProgressStage)
	}
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// 等真实法警观察链把 Judicial_2 抬到发布地板（与参考路线同口径）。
	for month := 0; month < 12 && svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bailiff at month %d: %v", month, err)
		}
	}
	publishFloor := engine.DefaultInformationConfig().QualPublish
	judicialBelief := engine.BeliefFor(svc.agent, "Judicial_2")
	if judicialBelief == nil || judicialBelief.B < publishFloor {
		t.Fatalf("real months did not reach the Judicial_2 publish floor %.2f: belief=%+v", publishFloor, judicialBelief)
	}
	if !contentTriggered(svc, "special:bailiff_dispatch_evidence_b") {
		t.Fatalf("special:bailiff_dispatch_evidence_b did not fire through real monthly progression")
	}

	// 核验与发布 (参考路线重调同口径)：双事实进公开登记；发布段扩到 40 步并
	// 加待发布存蓄纪律——investigate_dispatch (15 AP) 是抬升 Judicial_1 的
	// 确定性手段，但一旦有事实已核验只差 20 AP 发布，就必须存 AP 而不是继续
	// 调查，否则月收入被吃干、第二条事实发布带来的免费粘性槽整段丢失。
	publishedFacts := func() map[string]bool {
		out := map[string]bool{}
		for _, record := range svc.silo.PublicEvidence {
			out[record.FactID] = true
		}
		return out
	}
	eventReady := func(id string) bool {
		meta := actionByID(svc.availablePlayerActions(), id)
		return meta != nil && meta.Enabled
	}
	publishPending := func() bool {
		verified := map[string]bool{}
		for _, fact := range svc.silo.VerifiedEvidence {
			verified[fact] = true
		}
		published := publishedFacts()
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if !verified[fact] || published[fact] {
				continue
			}
			if belief := engine.BeliefFor(svc.agent, fact); belief != nil && belief.B >= publishFloor {
				return true
			}
		}
		return false
	}
	for i := 0; i < 40; i++ {
		published := publishedFacts()
		if published["Judicial_1"] && published["Judicial_2"] {
			break
		}
		if svc.gameOver() {
			break
		}
		acted := false
		if eventReady("verify_evidence") && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence"})
			acted = true
		}
		if eventReady("publish_evidence") && svc.agent.ActionPoints >= 20 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence"})
			acted = true
		}
		if !acted && !publishPending() && eventReady("investigate_dispatch") && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "investigate_dispatch"})
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance publication at step %d: %v", i, err)
		}
		progressResetCountermeasureForRoute(t, svc, execute)
	}
	if !contentTriggered(svc, "player_action:publish_evidence") {
		t.Fatalf("player publication never completed (verified=%v, evidence=%+v)", svc.silo.VerifiedEvidence, svc.silo.PublicEvidence)
	}
	if svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 {
		t.Fatalf("natural authority updates did not produce Q from bailiff evidence: Q=%v", svc.silo.Authority.Judicial.PublicEvidenceQ)
	}
	if svc.gameOver() {
		t.Fatalf("game ended early at %d-%02d during publication: reason=%q",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, decisionReasonOf(svc.silo.VictoryStatus))
	}

	// 陪同回查（Judicial 特工开局关系即满足 0.10 门）。
	var receiver *model.PlayerActionMeta
	for i := 0; i < 8; i++ {
		receiver = actionByID(svc.availablePlayerActions(), "RECEIVER_VERIFY_ACCOMPANY")
		if receiver != nil && receiver.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("recover AP for receiver verification: %v", err)
		}
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})

	// 流通循环 (info_endgame/rift 重调验证过的同一套机制, 无声探查变体)：
	//   - 服务端权威成本门 shareCostFor/buildCostFor：戒备上浮更贵、T3-2 监视
	//     期接触不可用 (resolveActionCost 报错→+Inf)，尝试不打在必拒动作上；
	//   - 断流维护 (upkeep) 优先，优先部门 (IT/Judicial) 买不起时整月存 AP；
	//   - 半成品种子续播不换种、陈源不播、深填到 B ≥ 0.45 滑行 30 月窗；
	//   - 播种截止用日历 (结算地平线 130-01 前留足 30 月渗透窗)；部门循环内
	//     建联预算全部换成分享 (建联自带 1 月时长，吸收差仅 ≈0.1)。
	shareCost := model.ACTION_COSTS[model.ActionShareInfo]
	gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
	shareAttempts := map[string]int{}
	upkeepAttempts := map[string]int{}
	pendingSeed := map[string]string{}
	shareCostFor := func(dept string) float64 {
		action := model.AgentAction{Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{"probe"}}
		if err := svc.resolveActionCost(&action); err != nil {
			return math.Inf(1)
		}
		return action.Cost
	}
	buildCostFor := func(dept string) float64 {
		action := model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept}
		if err := svc.resolveActionCost(&action); err != nil {
			return math.Inf(1)
		}
		return action.Cost
	}
	priority := func(name string) int {
		switch name {
		case "IT", "Judicial":
			return 0
		case "Mayor":
			return 1
		}
		return 2
	}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.silo.Rebellion >= 0.7 {
			continue
		}
		circulated := false
		saveForPriority := false
		for svc.agent.ActionPoints >= shareCost {
			order := append([]model.Profession(nil), svc.silo.Professions...)
			sort.SliceStable(order, func(i, j int) bool {
				pi, pj := priority(order[i].Name), priority(order[j].Name)
				if pi != pj {
					return pi < pj
				}
				mi, mj := informationMaturedFacts(&order[i]), informationMaturedFacts(&order[j])
				if mi != mj {
					return mi < mj
				}
				return informationActiveSlots(&order[i]) < informationActiveSlots(&order[j])
			})
			shared := false
			for i := range order {
				dept := &order[i]
				if informationMaturedFacts(dept) >= informationGateFactCount {
					continue
				}
				_, endangered := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts)
				pendingFact, hasPending := pendingSeed[dept.Name]
				pendingBelowFloor := false
				if hasPending {
					if belief := professionBelief(dept, pendingFact); belief == nil ||
						belief.B < engine.DefaultInformationConfig().QualRumor {
						pendingBelowFloor = true
					}
				}
				if priority(dept.Name) == 0 &&
					(informationActiveSlots(dept) < informationGateFactCount || endangered || pendingBelowFloor) &&
					shareCostFor(dept.Name) > svc.agent.ActionPoints {
					saveForPriority = true
					break
				}
				if fragment, ok := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts); ok &&
					svc.agent.ActionPoints >= shareCostFor(dept.Name) {
					upkeepAttempts[dept.Name+":"+fragment]++
					if tryExecute(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						break
					}
				}
				if informationActiveSlots(dept) >= informationGateFactCount {
					if _, waiting := pendingSeed[dept.Name]; !waiting {
						continue
					}
				}
				if 130*12+1-(svc.silo.CurrentYear*12+svc.silo.CurrentMonth) <= 30 {
					continue
				}
				fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					deadStreak := func(belief *model.FactBelief) bool {
						return belief != nil && belief.B < engine.DefaultInformationConfig().QualRumor &&
							belief.PermeationCounter == 0
					}
					if shareAttempts[dept.Name+":"+fact] < 4 {
						// 陈源不播：部门对同源仍有活观察时，同源递减把再吸收
						// 压在地板附近，30 月窗内必断流。豁免只给"玩家播过的
						// 流已死"——看观察记录 (share:player) 而非 shareAttempts。
						for j := range dept.Beliefs {
							if dept.Beliefs[j].FactID == fact {
								attempted := false
								for _, obs := range dept.Beliefs[j].Observations {
									if strings.HasPrefix(obs, "share:player:") {
										attempted = true
										break
									}
								}
								return !(attempted && deadStreak(&dept.Beliefs[j]))
							}
						}
						return false
					}
					return !deadStreak(professionBelief(dept, fact))
				})
				if pending, waiting := pendingSeed[dept.Name]; waiting {
					belief := professionBelief(dept, pending)
					if (belief == nil || (belief.B < engine.DefaultInformationConfig().QualRumor && belief.PermeationCounter == 0)) &&
						shareAttempts[dept.Name+":"+pending] < 8 {
						fragment, ok = pending, true
					} else {
						delete(pendingSeed, dept.Name)
					}
				}
				if ok {
					// 加深确信用无声探查 (免怀疑)，付不起时回退普通 gather。
					if agentBelief := engine.BeliefFor(svc.agent, fragment); agentBelief != nil && agentBelief.B < 0.80 {
						probeCost := resolvedCost(model.ActionSilentProbe, fragmentDepartment(fragment))
						if svc.agent.ActionPoints >= probeCost+shareCost {
							if tryExecute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: fragmentDepartment(fragment)}) {
								shared = true
								break
							}
						} else if svc.agent.ActionPoints >= gatherCost+shareCost {
							if tryExecute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: fragmentDepartment(fragment)}) {
								shared = true
								break
							}
						}
					}
					if math.IsInf(shareCostFor(dept.Name), 1) || svc.agent.ActionPoints < shareCostFor(dept.Name) {
						continue
					}
					shareAction := model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}
					shareAttempts[dept.Name+":"+fragment]++
					if tryExecute(shareAction) {
						shared = true
						for shareAttempts[dept.Name+":"+fragment] < 6 &&
							svc.agent.ActionPoints >= shareCostFor(dept.Name) {
							belief := professionBelief(dept, fragment)
							if belief == nil || belief.B >= 0.45 {
								break
							}
							shareAttempts[dept.Name+":"+fragment]++
							if !tryExecute(shareAction) {
								break
							}
						}
						if belief := professionBelief(dept, fragment); belief == nil ||
							belief.B < engine.DefaultInformationConfig().QualRumor {
							pendingSeed[dept.Name] = fragment
						} else {
							delete(pendingSeed, dept.Name)
						}
						break
					}
				}
			}
			if !shared {
				break
			}
			circulated = true
		}
		if circulated {
			continue
		}
		if saveForPriority {
			continue
		}
		if svc.agent.ActionPoints >= buildCostFor("Judicial") && month < 12 {
			tryExecute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		gatherFrom := svc.agent.Profession
		probeCost := resolvedCost(model.ActionSilentProbe, gatherFrom)
		if svc.agent.ActionPoints >= probeCost {
			tryExecute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: gatherFrom})
		} else if svc.agent.ActionPoints >= gatherCost {
			tryExecute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
		}
	}

	if circulationActions == 0 {
		t.Fatal("route reached its ending without any circulation action")
	}
	if silentUses < 3 {
		t.Fatalf("route must actually use the silent probe (若干次), used %d times", silentUses)
	}
	for _, key := range []string{
		"player_actions:receiver_verify_accompany",
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the silent route (Q=%v, evidence=%+v)", key, svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:replication_progress"].TriggerCount; count < 4 {
		t.Errorf("replication heartbeat fired %d times, want at least 4", count)
	}
	for _, factID := range []string{
		"Judicial_2",
		"fact:info_chain_replication",
		"fact:receiver_verification",
		"fact:independent_receivers",
	} {
		found := false
		for _, record := range svc.silo.PublicEvidence {
			if record.FactID == factID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("public evidence %s was not produced by the silent route", factID)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingInformation {
		excluded := ""
		if svc.silo.VictoryStatus != nil && svc.silo.VictoryStatus.Decision != nil {
			excluded = svc.silo.VictoryStatus.Decision.Excluded[endingInformation]
		}
		t.Fatalf("silent route did not settle as INFORMATION at %d-%02d: excluded=%q status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, excluded, svc.silo.VictoryStatus)
	}
	t.Logf("silent INFORMATION route on %s (seed %d): settled %d-%02d silentUses=%d suspicion=%.3f rebellion=%.3f",
		archetype, seed, svc.silo.CurrentYear, svc.silo.CurrentMonth, silentUses, svc.agent.SuspicionLevel, svc.silo.Rebellion)

	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal INFORMATION state accepted another action: %+v", outcome.Result)
	}
}

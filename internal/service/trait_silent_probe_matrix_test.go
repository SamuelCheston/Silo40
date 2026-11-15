package service

import (
	"fmt"
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
// 组合实测（种子/职业/原型/结算月）：
//   - 2401 / Judicial / bedrock（磐石）→ 128-11（IT 反击怀疑压力最大，靠提速跑赢）
//   - 2404 / Judicial / famine（荒年）→ 127-07
//   - 2424 / Judicial / rift（裂隙）→ 127-10（裂隙叛乱放大轴下只有部分种子可赢，
//     见报告：秩序波动轴是裂隙档的卡点）
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

	// 核验与发布：两份核心事实都进公开登记；发布尊重 2 个月冷却。
	verifyReportFloor := engine.DefaultInformationConfig().QualReport
	publishedFacts := func() map[string]bool {
		out := map[string]bool{}
		for _, record := range svc.silo.PublicEvidence {
			out[record.FactID] = true
		}
		return out
	}
	publishCooldownReady := func() bool {
		state := svc.contentStates[svc.silo.ID]["player_action:publish_evidence"]
		if state.TriggerCount == 0 {
			return true
		}
		now := gameTimestamp(svc.silo.CurrentYear, svc.silo.CurrentMonth)
		return now-state.LastTriggeredTimestamp >= 2*daysPerMonth
	}
	for i := 0; i < 20; i++ {
		published := publishedFacts()
		if published["Judicial_1"] && published["Judicial_2"] {
			break
		}
		verifiable, publishable := false, false
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			belief := engine.BeliefFor(svc.agent, fact)
			if belief == nil {
				continue
			}
			verified := false
			for _, v := range svc.silo.VerifiedEvidence {
				if v == fact {
					verified = true
				}
			}
			if belief.B >= verifyReportFloor && !verified {
				verifiable = true
			}
			if belief.B >= publishFloor && verified && !published[fact] {
				publishable = true
			}
		}
		if verifiable && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence"})
		}
		if publishable && publishCooldownReady() && svc.agent.ActionPoints >= 20 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence"})
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

	// 流通循环：把 9 个部门各 3 个槽位尽快落齐（渗透是纯时间门，落齐即停手
	// 等待成熟——动作越少，磐石档的 IT 反击怀疑越追不上结算）。加深确信优先
	// 无声探查；叛乱过热时蛰伏。
	shareCost := model.ACTION_COSTS[model.ActionShareInfo]
	buildCost := model.ACTION_COSTS[model.ActionBuildConnection]
	shareAttempts := map[string]int{}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	allSeeded := func() bool {
		for i := range svc.silo.Professions {
			if informationActiveSlots(&svc.silo.Professions[i]) < informationGateFactCount {
				return false
			}
		}
		return true
	}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if allSeeded() || svc.silo.Rebellion >= 0.7 {
			continue
		}
		gatherFrom := rotation[month%len(rotation)]
		circulated := false
		for svc.agent.ActionPoints >= shareCost {
			order := append([]model.Profession(nil), svc.silo.Professions...)
			sort.SliceStable(order, func(i, j int) bool {
				mi, mj := informationMaturedFacts(&order[i]), informationMaturedFacts(&order[j])
				if mi != mj {
					return mi < mj
				}
				return informationActiveSlots(&order[i]) < informationActiveSlots(&order[j])
			})
			shared := false
			for i := range order {
				dept := &order[i]
				if informationActiveSlots(dept) >= informationGateFactCount {
					continue
				}
				if month <= 30 && svc.pendingOperation == nil &&
					agentConnectionTo(svc, dept) < 0.3 && svc.agent.ActionPoints >= buildCost+shareCost {
					if tryExecute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept.Name}) {
						shared = true
						break
					}
				}
				if fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					return shareAttempts[dept.Name+":"+fact] >= 4
				}); ok {
					if agentBelief := engine.BeliefFor(svc.agent, fragment); agentBelief != nil && agentBelief.B < 0.80 {
						probeCost := resolvedCost(model.ActionSilentProbe, fragmentDepartment(fragment))
						if svc.agent.ActionPoints >= probeCost+shareCost {
							if tryExecute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: fragmentDepartment(fragment)}) {
								shared = true
								break
							}
						} else if gatherCost := resolvedCost(model.ActionGatherInfo, fragmentDepartment(fragment)); svc.agent.ActionPoints >= gatherCost+shareCost {
							if tryExecute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: fragmentDepartment(fragment)}) {
								shared = true
								break
							}
						}
					}
					shareAttempts[dept.Name+":"+fragment]++
					if tryExecute(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
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
		probeCost := resolvedCost(model.ActionSilentProbe, gatherFrom)
		if svc.agent.ActionPoints >= probeCost {
			tryExecute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: gatherFrom})
		} else if gatherCost := resolvedCost(model.ActionGatherInfo, gatherFrom); svc.agent.ActionPoints >= gatherCost {
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
		t.Fatalf("silent route did not settle as INFORMATION at %d-%02d: status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus)
	}
	t.Logf("silent INFORMATION route on %s (seed %d): settled %d-%02d silentUses=%d suspicion=%.3f rebellion=%.3f",
		archetype, seed, svc.silo.CurrentYear, svc.silo.CurrentMonth, silentUses, svc.agent.SuspicionLevel, svc.silo.Rebellion)

	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal INFORMATION state accepted another action: %+v", outcome.Result)
	}
}

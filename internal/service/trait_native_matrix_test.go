package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// trait_native_matrix_test.go 锁 19 §4.3/§6.4 的地堡土著矩阵附加项：
// 精英视角的同阶层免引荐 (关系门豁免、阶段门/事件门保留)，以及
// 土著 × 三原型的《公议》真实可玩组合。

// connectionValue 只读玩家特工对某部门的首条真实连接值。
func connectionValue(t *testing.T, svc *GameService, dept string) float64 {
	t.Helper()
	for _, profession := range svc.silo.Professions {
		if profession.Name != dept {
			continue
		}
		for _, connection := range svc.agent.Connections {
			if connection.ProfessionID == profession.ID {
				return connection.Value
			}
		}
		return 0
	}
	t.Fatalf("department %s missing", dept)
	return 0
}

// nativeRequest 精英职业土著建档请求 (关系门豁免的精英视角)。
func nativeRequest(seed int64, profession string) model.CreateGameRequest {
	return model.CreateGameRequest{
		SiloName:   "NativeEliteStation",
		AgentName:  "Player",
		Profession: profession,
		Seed:       seed,
		TraitIds:   []string{"native"},
	}
}

// TestNativeEliteWaivesSameClassRelationGates 锁 19 §4.3 的精英视角：精英职业
// 土著对同阶层 (ELITE) 部门的关系门视为已满足，且豁免必须由真实连接值低于
// 阈值来证明 (不是初始值本来就过关)；跨阶层 (COMMONER) 关系门不豁免。
func TestNativeEliteWaivesSameClassRelationGates(t *testing.T) {
	min02 := 0.2
	min015 := 0.15

	// Police 土著：开局对 Judicial (ELITE) 的连接是确定性的 0.15 (<0.20)，
	// 关系门天然不过——豁免成立时 evaluateActionCondition 必须翻绿。
	police := newGatedService(t, filepath.Join(t.TempDir(), "native-police.db"))
	if _, err := police.CreateGame(nativeRequest(302, "Police")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if got := connectionValue(t, police, "Judicial"); got >= 0.2 {
		t.Fatalf("police opening Judicial connection = %v, want < 0.20 so the waiver is observable", got)
	}
	if !police.evaluateActionCondition(actionCondition{Relation: "relation:Judicial", Min: &min02}) {
		t.Fatalf("native elite must waive the same-class Judicial relation gate (connection %v < 0.20)",
			connectionValue(t, police, "Judicial"))
	}
	// 跨阶层：Police (ELITE) 对 Mechanical (COMMONER) 的关系门不豁免。
	if got := connectionValue(t, police, "Mechanical"); got >= 0.2 {
		t.Fatalf("police opening Mechanical connection = %v, want < 0.20", got)
	}
	if police.evaluateActionCondition(actionCondition{Relation: "relation:Mechanical", Min: &min02}) {
		t.Fatal("cross-class relation gate must not be waived for a native elite")
	}

	// 对照：同种子无特质的 Police，Judicial 关系门保持 (0.15 < 0.20)。
	plainPolice := newGatedService(t, filepath.Join(t.TempDir(), "native-police-plain.db"))
	if _, err := plainPolice.CreateGame(model.CreateGameRequest{
		SiloName: "PlainPoliceStation", AgentName: "Player", Profession: "Police", Seed: 302,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if plainPolice.evaluateActionCondition(actionCondition{Relation: "relation:Judicial", Min: &min02}) {
		t.Fatal("without the native trait the Judicial relation gate must hold for Police (0.15 < 0.20)")
	}

	// IT 土著：开局唯一对 Mayor (ELITE) 的连接来自特质的同阶层加成
	// (0.10–0.15，严格 <0.15)，LEGITIMACY_SURVEY 的 Mayor 关系门天然不过。
	it := newGatedService(t, filepath.Join(t.TempDir(), "native-it.db"))
	if _, err := it.CreateGame(nativeRequest(303, "IT")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if got := connectionValue(t, it, "Mayor"); got >= 0.15 {
		t.Fatalf("native IT opening Mayor connection = %v, want < 0.15 so the waiver is observable", got)
	}
	if !it.evaluateActionCondition(actionCondition{Relation: "relation:Mayor", Min: &min015}) {
		t.Fatalf("native elite must waive the same-class Mayor relation gate (connection %v < 0.15)",
			connectionValue(t, it, "Mayor"))
	}
	plainIT := newGatedService(t, filepath.Join(t.TempDir(), "native-it-plain.db"))
	if _, err := plainIT.CreateGame(model.CreateGameRequest{
		SiloName: "PlainITStation", AgentName: "Player", Profession: "IT", Seed: 303,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if plainIT.evaluateActionCondition(actionCondition{Relation: "relation:Mayor", Min: &min015}) {
		t.Fatalf("without the native trait the Mayor relation gate must hold for IT (connection %v < 0.15)",
			connectionValue(t, plainIT, "Mayor"))
	}
}

// TestNativeEliteUnlocksGatedActionsWithoutBuildNetwork 走真实
// refreshActionProgress 路径：一次真实探查推进 first_signal 阶段后，
// 精英土著的关系门控动作无需任何 Build Network 即解锁；同种子非土著保持锁定。
// 事件门 (规则断点) 与阶段门照常生效——豁免只放宽关系门。
func TestNativeEliteUnlocksGatedActionsWithoutBuildNetwork(t *testing.T) {
	// IT 土著：LEGITIMACY_SURVEY = first_signal + Mayor 关系 0.15。
	it := newGatedService(t, filepath.Join(t.TempDir(), "native-it-unlock.db"))
	if _, err := it.CreateGame(nativeRequest(304, "IT")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if it.actionUnlocked("LEGITIMACY_SURVEY") {
		t.Fatal("stage gate must keep LEGITIMACY_SURVEY locked at arrival even for a native")
	}
	outcome, err := it.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("opening gather failed: %v %+v", err, outcome)
	}
	if it.world.ProgressStage != "first_signal" {
		t.Fatalf("gather did not advance the stage: %s", it.world.ProgressStage)
	}
	if !it.actionUnlocked("LEGITIMACY_SURVEY") {
		t.Fatalf("LEGITIMACY_SURVEY must unlock for native IT without Build Network (Mayor connection %v < 0.15)",
			connectionValue(t, it, "Mayor"))
	}
	if meta := actionByID(it.availablePlayerActions(), "LEGITIMACY_SURVEY"); meta == nil || !meta.Enabled {
		t.Fatalf("LEGITIMACY_SURVEY must be available once the relation gate is waived: %+v", meta)
	}

	plainIT := newGatedService(t, filepath.Join(t.TempDir(), "native-it-unlock-plain.db"))
	if _, err := plainIT.CreateGame(model.CreateGameRequest{
		SiloName: "PlainITStation", AgentName: "Player", Profession: "IT", Seed: 304,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := plainIT.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("plain gather failed: %v", err)
	}
	if plainIT.actionUnlocked("LEGITIMACY_SURVEY") {
		t.Fatal("LEGITIMACY_SURVEY must stay locked for non-native IT at game start")
	}

	// Police 土著：investigate_dispatch = first_signal + Judicial 关系 0.20。
	police := newGatedService(t, filepath.Join(t.TempDir(), "native-police-unlock.db"))
	if _, err := police.CreateGame(nativeRequest(305, "Police")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := police.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	if !police.actionUnlocked("investigate_dispatch") {
		t.Fatalf("investigate_dispatch must unlock for native Police without Build Network (Judicial connection %v < 0.20)",
			connectionValue(t, police, "Judicial"))
	}
	plainPolice := newGatedService(t, filepath.Join(t.TempDir(), "native-police-unlock-plain.db"))
	if _, err := plainPolice.CreateGame(model.CreateGameRequest{
		SiloName: "PlainPoliceStation", AgentName: "Player", Profession: "Police", Seed: 305,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := plainPolice.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("plain gather failed: %v", err)
	}
	if plainPolice.actionUnlocked("investigate_dispatch") {
		t.Fatal("investigate_dispatch must stay locked for non-native Police (Judicial 0.15 < 0.20)")
	}

	// Judicial 土著：CHARTER_CONTACT_OFFER 的关系门一侧豁免，但事件门保留——
	// 规则断点未真实发生前，接触卡不得解锁 (见 §6.4 组合测试里的断点后可解锁)。
	judicial := newGatedService(t, filepath.Join(t.TempDir(), "native-judicial-unlock.db"))
	if _, err := judicial.CreateGame(nativeRequest(306, "Judicial")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := judicial.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	if judicial.actionUnlocked("CHARTER_CONTACT_OFFER") {
		t.Fatal("event gate must keep CHARTER_CONTACT_OFFER locked before the breach, even with the relation gate waived")
	}
}

// TestNativeTraitDoesNotWaiveStageGates 锁 19 §4.3 边界：免引荐只放宽关系门。
// 土著跳过阶段直接分享仍被拒；阶段推进仍必须由真实动作产生。
func TestNativeTraitDoesNotWaiveStageGates(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "native-stage.db"))
	if _, err := svc.CreateGame(nativeRequest(307, "Judicial")); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.world.ProgressStage != "arrival" {
		t.Fatalf("opening stage = %s, want arrival", svc.world.ProgressStage)
	}
	if svc.evaluateActionCondition(actionCondition{StageAtLeast: "first_signal"}) {
		t.Fatal("stage condition must not be waived by the native trait")
	}
	if svc.actionUnlocked("action_share_info") {
		t.Fatal("action_share_info must stay locked before first_signal for a native")
	}
	// 白盒绕过尝试：直接执行分享被服务端门控拒绝。
	if _, err := svc.ExecuteAction(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	}); err == nil || !strings.Contains(err.Error(), "not been unlocked") {
		t.Fatalf("native must not bypass the share stage gate: %v", err)
	}
	// 一次真实探查产生 observation:first_local，阶段真实推进后分享解锁。
	outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if err != nil || !outcome.Result.Executed {
		t.Fatalf("opening gather failed: %v %+v", err, outcome)
	}
	if !svc.statesMapFor(svc.silo.ID)[observationFirstLocalKey].Triggered {
		t.Fatal("gather did not produce the first local observation")
	}
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("stage did not advance after the real observation: %s", svc.world.ProgressStage)
	}
	if !svc.actionUnlocked("action_share_info") {
		t.Fatal("action_share_info must unlock once the real stage advanced")
	}
}

// nativeRuleOfLawRequest §6.4 组合建档：原型覆写钉档 + 精英职业土著。
func nativeRuleOfLawRequest(seed int64, archetype, profession string) model.CreateGameRequest {
	req := threeSiloRequest(seed)
	req.Profession = profession
	req.TraitIds = []string{"native"}
	req.Archetype = archetype
	// 与参考路线同口径：赶在固定档案广播 (122-01) 之前开局，给真实流通留出
	// history_burden 累积时间。
	req.StartYear = 120
	return req
}

// driveNativeRuleOfLawRoute 从真实建档把「土著 × archetype × 精英职业」组合
// 打到 RULE_OF_LAW 结局：关系门豁免跳过攒关系期，其余条件 (断点、起草、锁定
// 窗、公开证据、加权裁决) 全部经真实动作与月度推进达成。返回月数供报告。
func driveNativeRuleOfLawRoute(t *testing.T, archetype, profession string, seed int64) int {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "native-rol-"+archetype+"-"+profession+".db"))
	if _, err := svc.CreateGame(nativeRuleOfLawRequest(seed, archetype, profession)); err != nil {
		t.Fatalf("create game (%s/%s): %v", archetype, profession, err)
	}
	if svc.silo.Archetype != archetype {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, archetype)
	}
	// 豁免是否「承重」：开局 Judicial 关系低于 0.20 时，接触卡的关系门只能靠
	// 同阶层免引荐通过 (Police 档恒为 0.15；Judicial 档初始即 ≥0.20，豁免语义
	// 生效但非承重，承重口径由 focused 测试锁定)。
	if judicialConn := connectionValue(t, svc, "Judicial"); judicialConn < 0.2 {
		t.Logf("%s/%s: relation gate is load-bearing (opening Judicial connection %v < 0.20)",
			archetype, profession, judicialConn)
	}
	buildConnectionCount := 0
	execute := func(action model.AgentAction) {
		t.Helper()
		if action.Type == model.ActionBuildConnection {
			buildConnectionCount++
		}
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v, known=%v)", action, err, svc.agent.ActionPoints, svc.agent.KnownFragments)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v, known=%v)", action, outcome.Result, svc.agent.ActionPoints, svc.agent.KnownFragments)
		}
	}

	// 开局：一次真实探查 + 首次分享——断点需要分享完成键 + history_burden ≥ 0.10。
	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// 接触阶段：精英土著免引荐——不做任何 Build Network，只等真实断点点火，
	// 接触卡必须凭关系门豁免直接解锁。
	var offer *model.PlayerActionMeta
	for i := 0; i < 48 && !svc.gameOver(); i++ {
		offer = actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_OFFER")
		if offer != nil && offer.Enabled {
			break
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance for contact card (%s): %v", archetype, err)
		}
	}
	if offer == nil || !offer.Enabled {
		t.Fatalf("%s: charter contact card never unlocked without relation building (breach=%v burden=%.3f W=%.2f stage=%s)",
			archetype, contentTriggered(svc, "special:order_chain_breach"), svc.silo.HistoryBurden,
			svc.silo.Authority.W, svc.world.ProgressStage)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_OFFER"})
	accept := actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_ACCEPT")
	if accept == nil || !accept.Enabled {
		t.Fatalf("%s: accept choice not available after the contact offer: %+v", archetype, accept)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_ACCEPT"})

	windowOpenAt := 0
	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	decisiveFired := false
	decisivePublishW, decisiveImpactW := 0.0, 0.0
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		return err == nil && outcome.Result.Executed
	}
	// 决定性发布时机 (磐石教训)：第二次合格事实发布是 −24 决定性冲击的载体，
	// 必须等日常压力把 W 磨到谷底附近 (≤42：−24−5 后落进 ≤18 窗口) 再出手；
	// 过早发布时冲击被高 W 吞掉，窗口打不开，而高 W 下 IT 反扑 (每次怀疑
	// +0.08) 会先拖垮特工。阈值对裂隙/荒年是宽松上限 (它们发布时 W 已更低)。
	const decisiveWGate = 42.0
	for month := 1; month <= 140 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance %s line at month %d: %v", archetype, month, err)
		}
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		// 与参考路线同口径：反制只需要一次真实二次分享，复用任一真实持有的事实。
		if contentTriggered(svc, "crisis:reset_omen_requisition") &&
			contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") &&
			!contentTriggered(svc, "player_action:reset_countermeasure_done") &&
			svc.agent.ActionPoints >= model.ACTION_COSTS[model.ActionShareInfo] {
			for _, belief := range svc.agent.Beliefs {
				if belief.B < 0.3 {
					continue
				}
				execute(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: "Mayor", FragmentIds: []string{belief.FactID},
				})
				break
			}
		}
		gameMonth := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
		if windowOpenAt == 0 && svc.silo.Authority.W <= 18 &&
			svc.silo.Authority.Judicial.PublicEvidenceQ >= 0.85 &&
			svc.silo.Authority.Judicial.IndependentPublicFacts >= 2 {
			windowOpenAt = month
		}

		// 证据链：先核验发布第一条，再集中资源做第二条的决定性发布。
		secondPublishReady := exposes >= 3 && gameMonth-lastExposeMonth >= 3 &&
			svc.agent.ActionPoints >= 40 && svc.silo.Authority.W <= decisiveWGate
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(svc.silo, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		if factPublished(svc.silo, "Judicial_1") && !factVerified(svc.silo, "Judicial_2") &&
			playerCoreBelief(svc, "Judicial_2") >= 0.85 && svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if fact == "Judicial_2" && !secondPublishReady {
				continue
			}
			if playerCoreBelief(svc, fact) >= 0.9 && factVerified(svc.silo, fact) &&
				!factPublished(svc.silo, fact) && svc.agent.ActionPoints >= 20 {
				publishW := svc.silo.Authority.W
				if attempt("publish_evidence", 20) && fact == "Judicial_2" {
					decisiveFired = true
					decisivePublishW = publishW
				}
				break
			}
		}
		if svc.agent.ActionPoints >= 15 &&
			(playerCoreBelief(svc, "Judicial_1") < 0.9 || playerCoreBelief(svc, "Judicial_2") < 0.9) &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			attempt("investigate_dispatch", 15)
		}

		// 公开执行失败是日常磨 W 的主通道 (−8 冲击 + 公开观察抬 E + 法警忠诚
		// 受挫)：决定性发布前允许更多次，把 W 压进决定性格。
		exposeForPressure := !decisiveFired && exposes < 6 && svc.agent.ActionPoints >= 25
		exposeForDecisive := decisiveFired && decisiveImpactW == 0 && svc.agent.ActionPoints >= 20
		if (exposeForPressure || exposeForDecisive) && gameMonth-lastExposeMonth >= 3 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			if attempt("expose_execution_failure", 20) {
				exposes++
				lastExposeMonth = gameMonth
				if decisiveFired && decisiveImpactW == 0 {
					decisiveImpactW = svc.silo.Authority.W
				}
			}
		}
		// 共享供给：把核心调度事实喂给平民部门，产生 Q 与独立公开事实；更宽的
		// 流通面同时抬高暴露水位 E，压低 W 的日常平衡点。
		for _, dept := range []string{"Mechanical", "Supply", "Mines", "Agricultural"} {
			for _, fact := range []string{"Judicial_1", "Judicial_2"} {
				if playerCoreBelief(svc, fact) < 0.6 || svc.agent.ActionPoints < 25 ||
					shares[dept+":"+fact] >= 2 || (exposes >= 6 && !decisiveFired) {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{fact},
				})
				if err == nil && outcome.Result.Executed {
					shares[dept+":"+fact]++
				}
			}
		}
	}

	for _, key := range []string{
		"special:order_chain_breach",
		"player_actions:charter_contact_offer",
		"player_actions:charter_contact_accept",
		"special:charter_draft_step",
		"special:charter_draft_convened",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s: %s did not fire in the native route (W=%v Q=%v facts=%v)", archetype, key,
				svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
				svc.silo.Authority.Judicial.IndependentPublicFacts)
		}
	}
	// 锁定日 (charter_lock_day) 有 127-01 的日历闸：免引荐让本路线可能早于
	// 127 年就凭引擎约章动力学 (PactLocked) 结算。日历闸到达时锁定日必须真
	// 实点火；未到达时以引擎锁定事实为准。
	if svc.silo.CurrentYear >= 127 {
		if !contentTriggered(svc, "special:charter_lock_day") {
			t.Errorf("%s: charter_lock_day did not fire inside the open lock window (W=%v Q=%v facts=%v)",
				archetype, svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
				svc.silo.Authority.Judicial.IndependentPublicFacts)
		}
	}
	if count := svc.contentStates[svc.silo.ID]["special:charter_draft_step"].TriggerCount; count < 2 {
		t.Errorf("%s: drafting heartbeat fired %d times, want at least 2", archetype, count)
	}
	for _, factID := range []string{"Judicial_1", "Judicial_2"} {
		found := false
		for _, record := range svc.silo.PublicEvidence {
			if record.FactID == factID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: shared supply did not register %s (evidence=%+v)", archetype, factID, svc.silo.PublicEvidence)
		}
	}
	if buildConnectionCount != 0 {
		t.Fatalf("%s: the native route spent %d Build Network actions; the exemption must skip the relation-building phase",
			archetype, buildConnectionCount)
	}
	if !decisiveFired {
		t.Fatalf("%s: the second qualifying fact was never published through the real action", archetype)
	}
	if windowOpenAt == 0 {
		status := svc.silo.VictoryStatus
		if status != nil && status.Decision != nil {
			t.Logf("%s window-never-opened settlement: ending=%s selected=%s date=%d-%02d pool=%v scores=%v excluded=%v",
				archetype, status.Type, status.Decision.Selected, svc.silo.CurrentYear, svc.silo.CurrentMonth,
				status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
		}
		t.Logf("%s window-never-opened state: E=%.3f K=%.3f rebellion=%.3f suspicion=%.3f shocks=%+v decisive=%+v",
			archetype, svc.silo.Authority.E, svc.silo.Authority.K, svc.silo.Rebellion,
			svc.agent.SuspicionLevel, svc.silo.Authority.Shocks, svc.silo.Authority.DecisiveShocks)
		t.Fatalf("%s: the real decisive event never opened the covenant window (W=%v Q=%v facts=%v)",
			archetype, svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
			svc.silo.Authority.Judicial.IndependentPublicFacts)
	}
	if !contentTriggered(svc, "player_action:reset_countermeasure_done") {
		t.Fatalf("%s: the route did not complete the RESET countermeasure through real actions", archetype)
	}
	if contentTriggered(svc, resetFactKey) {
		t.Fatalf("%s: the completed RESET countermeasure still emitted reset_completed", archetype)
	}
	if !svc.silo.Authority.Judicial.PactLocked {
		t.Fatalf("%s: the native route ended without a locked pact", archetype)
	}
	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil {
		t.Fatalf("%s: native route did not produce a real arbitration decision: status=%+v", archetype, status)
	}
	ending := status.Type
	selected := status.Decision.Selected
	t.Logf("%s native RULE_OF_LAW route settled: ending=%s selected=%s date=%d-%02d W=%.2f Q=%.4f facts=%d pact_locked=%v rebellion=%.3f suspicion=%.3f window_open_month=%d exposes=%d publishW=%.2f impactW=%.2f pool=%v scores=%v",
		archetype, ending, selected, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Authority.W,
		svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.Authority.Judicial.IndependentPublicFacts,
		svc.silo.Authority.Judicial.PactLocked, svc.silo.Rebellion, svc.agent.SuspicionLevel,
		windowOpenAt, exposes, decisivePublishW, decisiveImpactW, status.Decision.Pool, status.Decision.Scores)
	if selected != endingRuleOfLaw {
		for _, contribution := range status.Decision.Contributions {
			t.Logf("%s contribution: kind=%s entity=%s name=%s scores=%v", archetype, contribution.Kind, contribution.EntityID, contribution.Name, contribution.Scores)
		}
		t.Fatalf("%s: native route selected %q instead of RULE_OF_LAW: pool=%v scores=%v excluded=%v",
			archetype, selected, status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("%s: terminal RULE_OF_LAW state accepted another action: %+v", archetype, outcome.Result)
	}
	return svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
}

// TestNativeRuleOfLawPlayablePerArchetype 锁 19 §6.4 附加项：地堡土著 ×
// 每原型 ≥1 个《公议》通过组合。原型覆写钉档，精英职业土著，免引荐跳过
// CHARTER_CONTACT_OFFER 前的攒关系期，全程真实动作。每原型两个组合：
// Judicial (任务建议职业) 与 Police (开局 Judicial 关系 0.15 < 0.20，接触卡
// 的关系门只能靠豁免通过——豁免承重的组合)。
func TestNativeRuleOfLawPlayablePerArchetype(t *testing.T) {
	cases := []struct {
		archetype  string
		profession string
		seed       int64
	}{
		{engine.ArchetypeRift, "Judicial", 4201},
		{engine.ArchetypeRift, "Police", 4202},
		{engine.ArchetypeFamine, "Judicial", 4301},
		{engine.ArchetypeFamine, "Police", 4302},
		{engine.ArchetypeBedrock, "Judicial", 4101},
		{engine.ArchetypeBedrock, "Police", 4102},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.archetype+"/"+tc.profession, func(t *testing.T) {
			months := driveNativeRuleOfLawRoute(t, tc.archetype, tc.profession, tc.seed)
			t.Logf("%s/%s route settled in %d game months", tc.archetype, tc.profession, months)
		})
	}
}

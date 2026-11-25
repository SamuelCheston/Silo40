package service

// archetype_bedrock_test.go 磐石档 (IT 控制强化) 的标定证据 (19 §6.2/§8)：
//
//   - 开局可感知性：建档首月玩家可见量的档间差异 (权威面板 W、SAMPLED 探查
//     怀疑增量、资源不变)；
//   - 三线可达性：原型覆写钉档、固定种子、无头真实跑通的《公议》《人治》
//     《共证》通关实例——策略复用既有可玩线参考路线的原型适配形态，全程只经
//     CreateGame/ExecuteAction/PassTime 改变模拟，不白盒设状态。
//
// 可达性约束 (19 §6.2)：(a)《公议》W≤18+PactLocked、(b)《人治》W≤30 必须在
// 地平线内真实可达；(c) W0 上调幅度不得使磨 W 动作数超出地平线 AP 预算。
//
// 定稿旋钮 (W0Delta=+10, WHalfLifeMonths=12, DetectionFactor=1.5) 下的实测：
//
//	《公议》  种子 4101/Judicial/120 开局：126-02 结算 (73/120 月)，W 谷值
//	         4.27，决定性发布 W=39.21，怀疑峰 0.365，磨 W 曝光 7 次 (与 +8
//	         时相同，(c) 成立)；
//	《人治》  种子 2401/Judicial/122 开局：130-02 地平线结算 (97 月)，结算
//	         W=12.71 (门槛 30)，W 谷值 11.73，怀疑峰 0.390，顶压曝光 13 次；
//	《共证》  种子 2401/Judicial+隐秘行事/122 开局：128-11 结算 (82 月)，
//	         怀疑峰 0.735 (硬线 1.0)，无声探查 12 次。
//
// 实测边界 (回退判据)：W0Delta ≥ +11 或 WHalfLifeMonths ≥ 13 时，高 W 使
// ItCostCurve 反击成本更低、IT 反击更密，《共证》种子 2401 怀疑峰跳至
// 0.95+ (距《除名》硬线 0.03–0.05)；检测系数至 2.0 钉档各线仍绿 (策略结构性
// 规避 SAMPLED 探查)，1.5 为探查密集型矩阵组合保留余量。推导细节在
// internal/engine/archetype.go 的 bedrock 块注释。

import (
	"math"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// bedrockRequest 磐石档建档请求：覆写钉档 (19 §3.3 测试与选档同一口径)。
func bedrockRequest(seed int64, profession string, startYear int) model.CreateGameRequest {
	req := threeSiloRequest(seed)
	req.Profession = profession
	req.Archetype = engine.ArchetypeBedrock
	if startYear != 0 {
		req.StartYear = startYear
	}
	return req
}

// bedrockTracker 逐月采样 W 谷值与怀疑峰值 (可达性证据的关键量)。
type bedrockTracker struct {
	wMin     float64
	susMax   float64
	startAbs int
}

func newBedrockTracker(svc *GameService) *bedrockTracker {
	return &bedrockTracker{
		wMin:     svc.silo.Authority.W,
		startAbs: svc.silo.CurrentYear*12 + svc.silo.CurrentMonth,
	}
}

func (tr *bedrockTracker) sample(svc *GameService) {
	if w := svc.silo.Authority.W; w < tr.wMin {
		tr.wMin = w
	}
	if s := svc.agent.SuspicionLevel; s > tr.susMax {
		tr.susMax = s
	}
}

func (tr *bedrockTracker) months(svc *GameService) int {
	return svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - tr.startAbs
}

// TestBedrockOpeningPerceptible 锁 19 §3.2 的开局可感知面：建档首月，档间差异
// 全部落在玩家可见量上——权威面板 W (磐石最高、裂隙最低)、SAMPLED 探查回执的
// 怀疑增量 (磐石 > 裂隙，比值 = 检测系数比值)、资源水位不变 (磐石不动资源)。
// 同种子建档使两路 RNG 流逐位一致，差异只来自原型旋钮。
func TestBedrockOpeningPerceptible(t *testing.T) {
	// 种子 2401 号数派生为裂隙档：无覆写建档即基线参照，覆写磐石钉档对照。
	const seed = int64(2401)
	if got := engine.ArchetypeForStation(assignedStationNumber(seed)); got != engine.ArchetypeRift {
		t.Fatalf("seed %d must derive rift for the plain baseline, got %s", seed, got)
	}
	base := engine.DefaultAuthorityConfig()
	bedrockArch, _ := engine.ArchetypeByID(engine.ArchetypeBedrock)
	riftArch, _ := engine.ArchetypeByID(engine.ArchetypeRift)

	create := func(archetype string) *GameService {
		svc := newGatedService(t, filepath.Join(t.TempDir(), "bedrock-opening-"+archetype+".db"))
		req := archetypeRequest(seed, archetype)
		req.Profession = "Judicial"
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create game (%s): %v", archetype, err)
		}
		return svc
	}

	bedrock := create(engine.ArchetypeBedrock)
	rift := create(engine.ArchetypeRift)
	plain := create("") // 号数派生 (本种子 = 裂隙)：覆写与派生同口径的基线参照

	// (1) 权威面板 W：磐石 = NewGameW+Δ 上调、裂隙下调、磐石严格更高。
	if got, want := bedrock.silo.Authority.W, base.NewGameW+bedrockArch.W0Delta; got != want {
		t.Fatalf("bedrock opening W = %v, want %v", got, want)
	}
	if got, want := rift.silo.Authority.W, base.NewGameW+riftArch.W0Delta; got != want {
		t.Fatalf("rift opening W = %v, want %v", got, want)
	}
	if plain.silo.Authority.W != rift.silo.Authority.W {
		t.Fatalf("derived (plain) opening W = %v must match the rift override %v",
			plain.silo.Authority.W, rift.silo.Authority.W)
	}
	if bedrock.silo.Authority.W <= rift.silo.Authority.W {
		t.Fatalf("opening W must be perceptibly higher on bedrock: bedrock=%v rift=%v",
			bedrock.silo.Authority.W, rift.silo.Authority.W)
	}
	// CommandIT 随 W 同向 (权威面板的 IT 指挥链读数)。
	if bedrock.silo.Authority.Judicial.CommandIT <= rift.silo.Authority.Judicial.CommandIT {
		t.Fatalf("bedrock CommandIT = %v must exceed rift %v",
			bedrock.silo.Authority.Judicial.CommandIT, rift.silo.Authority.Judicial.CommandIT)
	}

	// (2) 资源不变：磐石不动初始资源水位——与同种子裂隙/派生档逐位一致。
	for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
		b := siloResourceAmount(bedrock.silo, resourceType)
		r := siloResourceAmount(rift.silo, resourceType)
		p := siloResourceAmount(plain.silo, resourceType)
		if b != r || b != p {
			t.Fatalf("%s must be identical across bedrock/rift/plain openings: %v / %v / %v",
				resourceType, b, r, p)
		}
	}

	// (3) SAMPLED 探查回执怀疑增量：同种子同动作序列下，磐石增量 > 裂隙增量，
	// 且比值恰为检测系数比值 (其余因子逐位一致)。司法特工跨域探查市长为
	// SAMPLED 分类 (非 Mines 域)。
	probe := func(svc *GameService) float64 {
		before := svc.agent.SuspicionLevel
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
		if err != nil || !outcome.Result.Executed {
			t.Fatalf("opening probe failed: %v %+v", err, outcome.Result)
		}
		return svc.agent.SuspicionLevel - before
	}
	bedrockDelta := probe(bedrock)
	riftDelta := probe(rift)
	if bedrockDelta <= 0 || riftDelta <= 0 {
		t.Fatalf("SAMPLED opening probe must raise suspicion: bedrock=%v rift=%v", bedrockDelta, riftDelta)
	}
	if bedrockDelta <= riftDelta {
		t.Fatalf("bedrock probe suspicion increment %v must exceed rift %v", bedrockDelta, riftDelta)
	}
	wantRatio := bedrockArch.DetectionFactor / riftArch.DetectionFactor
	if ratio := bedrockDelta / riftDelta; math.Abs(ratio-wantRatio) > 1e-9 {
		t.Fatalf("suspicion increment ratio = %v, want detection-factor ratio %v", ratio, wantRatio)
	}
	t.Logf("bedrock opening perceptibility: W=%v (rift %v), probe suspicion +%v (rift +%v, ratio %.3f), resources identical",
		bedrock.silo.Authority.W, rift.silo.Authority.W, bedrockDelta, riftDelta, bedrockDelta/riftDelta)
}

// TestBedrockRuleOfLawLinePlayable 磐石档《公议》可达性证据 (19 §6.2 (a)(c))：
// 覆写钉档、种子 4101、司法特工、120 年开局 (地平线 120 个月至 130-01)。策略
// 是参考路线 (TestRuleOfLawLinePlayableFromNewGame / 土著矩阵磐石组合) 的无
// 特质形态：司法职业开局对司法部关系即 ≥0.20，接触卡关系门天然通过；决定性
// 发布 (−24 冲击) 等日常压力把 W 磨进谷底带 (≤42) 再出手。
func TestBedrockRuleOfLawLinePlayable(t *testing.T) {
	const seed = int64(4101)
	svc := newGatedService(t, filepath.Join(t.TempDir(), "bedrock-ruleoflaw.db"))
	if _, err := svc.CreateGame(bedrockRequest(seed, "Judicial", 120)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeBedrock {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeBedrock)
	}
	tr := newBedrockTracker(svc)
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
	}
	passTime := func() {
		t.Helper()
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bedrock RULE_OF_LAW line: %v", err)
		}
		tr.sample(svc)
	}

	// 开局：一次真实探查 + 首次分享——断点需要分享完成键 + history_burden ≥ 0.10。
	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// 等真实断点点火：司法职业关系门天然满足，接触卡只等事件门。
	var offer *model.PlayerActionMeta
	for i := 0; i < 48 && !svc.gameOver(); i++ {
		offer = actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_OFFER")
		if offer != nil && offer.Enabled {
			break
		}
		passTime()
	}
	if offer == nil || !offer.Enabled {
		t.Fatalf("charter contact never unlocked (breach=%v burden=%.3f W=%.2f stage=%s)",
			contentTriggered(svc, "special:order_chain_breach"), svc.silo.HistoryBurden,
			svc.silo.Authority.W, svc.world.ProgressStage)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_OFFER"})
	accept := actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_ACCEPT")
	if accept == nil || !accept.Enabled {
		t.Fatalf("accept choice not available after the contact offer: %+v", accept)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_ACCEPT"})
	tr.startAbs = svc.silo.CurrentYear*12 + svc.silo.CurrentMonth // 结算月数自约章接触起算

	windowOpenAt := 0
	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	decisiveFired := false
	decisivePublishW := 0.0
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		return err == nil && outcome.Result.Executed
	}
	// 决定性发布时机 (磐石教训)：−24 冲击必须落在日常磨到谷底带的 W 上
	// (≤42：−24−5 后落进 ≤18 锁定窗)；过早发布被高 W 吞掉，窗口打不开。
	const decisiveWGate = 42.0
	for month := 1; month <= 140 && !svc.gameOver(); month++ {
		passTime()
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
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
		gameMonth := tr.months(svc)
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

		// 公开执行失败是日常磨 W 的主通道 (−8 冲击)：决定性发布前允许更多次。
		exposeForPressure := !decisiveFired && exposes < 6 && svc.agent.ActionPoints >= 25
		exposeForDecisive := decisiveFired && windowOpenAt == 0 && svc.agent.ActionPoints >= 20
		if (exposeForPressure || exposeForDecisive) && gameMonth-lastExposeMonth >= 3 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			if attempt("expose_execution_failure", 20) {
				exposes++
				lastExposeMonth = gameMonth
			}
		}
		// 共享供给：核心调度事实喂给平民部门，产生 Q 与独立公开事实，扩大流通面。
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

	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil {
		t.Fatalf("bedrock RULE_OF_LAW route did not settle: W=%.2f Q=%.4f facts=%d pact_locked=%v window_open=%d exposes=%d",
			svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
			svc.silo.Authority.Judicial.IndependentPublicFacts, svc.silo.Authority.Judicial.PactLocked,
			windowOpenAt, exposes)
	}
	if !svc.silo.Authority.Judicial.PactLocked {
		t.Fatalf("bedrock route ended without a locked pact: W=%.2f Q=%.4f facts=%d",
			svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
			svc.silo.Authority.Judicial.IndependentPublicFacts)
	}
	if windowOpenAt == 0 {
		t.Fatalf("the decisive event never opened the covenant window (W≤18+Q≥0.85+facts≥2)")
	}
	if !decisiveFired {
		t.Fatal("the second qualifying fact was never published through the real action")
	}
	if status.Decision.Selected != endingRuleOfLaw {
		t.Fatalf("bedrock route selected %q instead of RULE_OF_LAW: pool=%v scores=%v excluded=%v",
			status.Decision.Selected, status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
	}
	// (a) 的实测边界：锁定窗在地平线 (120 年开局 → 130-01 一号堡覆灭，120 个月)
	// 内真实打开。
	if elapsed := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - (120*12 + 1); elapsed > 120 {
		t.Fatalf("settlement %d-%02d (%d months) beyond the 120-month horizon",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, elapsed)
	}
	t.Logf("bedrock RULE_OF_LAW (seed %d): settled %d-%02d in %d months, W_min=%.2f publishW=%.2f suspicion_peak=%.3f exposes=%d pact_locked=%v pool=%v",
		seed, svc.silo.CurrentYear, svc.silo.CurrentMonth, tr.months(svc), tr.wMin, decisivePublishW,
		tr.susMax, exposes, svc.silo.Authority.Judicial.PactLocked, status.Decision.Pool)
}

// TestBedrockPersonalRuleLinePlayable 磐石档《人治》可达性证据 (19 §6.2 (b)(c))：
// 覆写钉档、种子 2401、司法特工、122 年开局 (地平线 = 130-01 一号堡覆灭强制
// 结算，96 个月)。策略是验收宿主 (playPersonalRuleAcceptance) 的原型适配形态：
// 只发布一条核心事实 (无决定性冲击)，W 必须靠日常压力与定时顶压退到 ≤30；
// 磐石档 W 高且退得慢，掩护周期与地平线顶压承担全部磨 W 职责。
func TestBedrockPersonalRuleLinePlayable(t *testing.T) {
	const seed = int64(2401)
	svc := newGatedService(t, filepath.Join(t.TempDir(), "bedrock-personalrule.db"))
	if _, err := svc.CreateGame(bedrockRequest(seed, "Judicial", 0)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeBedrock {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeBedrock)
	}
	tr := newBedrockTracker(svc)
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
	}
	passTime := func() {
		t.Helper()
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bedrock PERSONAL_RULE line: %v", err)
		}
		tr.sample(svc)
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// 市长盟友路线走真实门控 (Mayor 关系)。
	offer := func(cardID, buildDept string, budget int) {
		t.Helper()
		for i := 0; i < budget; i++ {
			card := actionByID(svc.availablePlayerActions(), cardID)
			if card != nil && card.Enabled {
				execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: cardID})
				return
			}
			build := actionByID(svc.availablePlayerActions(), "action_build_connection")
			if build != nil && build.Enabled {
				execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: buildDept})
				continue
			}
			passTime()
			if svc.gameOver() {
				t.Fatalf("game ended while waiting for %s", cardID)
			}
		}
		t.Fatalf("%s never unlocked within budget", cardID)
	}
	offer("LEGITIMACY_SURVEY", "Mayor", 12)
	offer("MAYOR_ALLY_ACCEPT", "Mayor", 1)

	// 证据路线的调查门需要司法部连接 ≥0.2 (司法职业开局即满足，真实建连兜底)。
	for i := 0; i < 12 && connectionValue(t, svc, "Judicial") < 0.2; i++ {
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		passTime()
	}

	successionTaken := false
	exposes, lastExpose := 0, -3
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		return err == nil && outcome.Result.Executed
	}
	exposeBurst := func(m int, cap int, maxSuspicion float64) {
		if svc.agent.SuspicionLevel >= maxSuspicion || exposes >= cap || m-lastExpose < 3 ||
			svc.agent.ActionPoints < 20 {
			return
		}
		if playerCoreBelief(svc, "Judicial_1") < 0.6 && playerCoreBelief(svc, "Judicial_2") < 0.6 {
			return
		}
		if attempt("expose_execution_failure", 20) {
			exposes++
			lastExpose = m
		}
	}
	// 配给是 Supply 职业动作：司法特工尝试会被拒 (与验收宿主同口径，静默跳过)。
	ration := func() {
		if svc.agent.ActionPoints < 15 {
			return
		}
		_, _ = svc.ExecuteAction(model.AgentAction{
			Type: model.ActionProfession, ProfessionAction: "SUPPLY_RATION", ResourceTarget: "Supplies",
		})
	}
	// RESET 反制 (与验收宿主 r.countermeasures 同口径)：完成键计任意分享
	// (trigger_count ≥ 2)，即使目标部门已持有也重分享最强事实——本线发布早，
	// 核心事实可能已广播全堡，"新部门" 变体会因无新事实而饿死反制。
	countermeasures := func() {
		if svc.gameOver() || contentTriggered(svc, "player_action:reset_countermeasure_done") ||
			!contentTriggered(svc, "crisis:reset_omen_requisition") {
			return
		}
		if !contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") {
			if card := actionByID(svc.availablePlayerActions(), "RESET_COUNTERMEASURE_OFFER"); card != nil && card.Enabled {
				execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RESET_COUNTERMEASURE_OFFER"})
			}
			return
		}
		if svc.agent.ActionPoints < model.ACTION_COSTS[model.ActionShareInfo] {
			return
		}
		best, bestB := "", 0.0
		for _, belief := range svc.agent.Beliefs {
			if belief.B >= 0.30 && belief.B > bestB {
				best, bestB = belief.FactID, belief.B
			}
		}
		if best == "" {
			rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
			execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: rotation[tr.months(svc)%len(rotation)]})
			return
		}
		execute(model.AgentAction{
			Type: model.ActionShareInfo, TargetDept: neediestDepartment(svc.silo), FragmentIds: []string{best},
		})
	}

	for m := 1; m <= acceptanceMaxMonths && !svc.gameOver(); m++ {
		passTime()
		if svc.gameOver() {
			break
		}
		countermeasures()
		if svc.gameOver() {
			break
		}
		if !successionTaken && contentTriggered(svc, "special:mayor_public_stand") {
			if card := actionByID(svc.availablePlayerActions(), "SUCCESSION_PURE_RULE"); card != nil && card.Enabled {
				if attempt("SUCCESSION_PURE_RULE", 0) {
					successionTaken = true
				}
			}
		}
		// 只发布 Judicial_1：一条合格事实、无决定性冲击——W 必须靠日常压力退。
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(svc.silo, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		if playerCoreBelief(svc, "Judicial_1") >= 0.9 && factVerified(svc.silo, "Judicial_1") &&
			!factPublished(svc.silo, "Judicial_1") && svc.agent.ActionPoints >= 20 {
			attempt("publish_evidence", 20)
		}
		if svc.agent.ActionPoints >= 15 && playerCoreBelief(svc, "Judicial_1") < 0.9 &&
			playerCoreBelief(svc, "Judicial_1") >= 0.6 {
			attempt("investigate_dispatch", 15)
		}

		windowOpen := svc.silo.Authority.W <= 33 || exposes >= 6
		horizon := svc.silo.CurrentYear >= 129
		if svc.silo.Rebellion >= 0.7 {
			continue
		}
		switch {
		case horizon:
			// 地平线顶压：一号堡覆灭 (130-01) 是强制结算点，顶压必须把 W 压进
			// ≤30 并撑过末月漂移。磐石档衰减慢，冲击余效更长，顶压可以略提前。
			if svc.agent.ActionPoints >= 20 {
				cal := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
				if 130*12+1-cal <= 12 || (m-lastExpose >= 8 && svc.agent.SuspicionLevel < 0.88) {
					exposeBurst(m, 15, 0.88)
				}
			}
			if resourceAmount(svc.silo, "Supplies") < 500 {
				ration()
			}
		case windowOpen:
			// 掩护周期：定期曝光维持联盟侵蚀，让 IT 忙于维护而不是反击玩家
			// (counter_investigate_player +0.08/次是主要怀疑来源)。
			exposeDueWindow := m-lastExpose >= 8 && svc.agent.SuspicionLevel < 0.88 &&
				svc.silo.CurrentYear < 129
			supplies := resourceAmount(svc.silo, "Supplies")
			switch {
			case exposeDueWindow && svc.agent.ActionPoints >= 25:
				exposeBurst(m, 12, 0.88)
			case supplies < 1000:
				ration()
			case supplies < 2500 && !exposeDueWindow:
				ration()
			}
		default:
			// 紧缩施压期：先曝光 (抬 E + 侵蚀联盟 + W 冲击)，再维生，再调查。
			exposeDue := exposes < 6 && m-lastExpose >= 3 && svc.agent.SuspicionLevel < 0.6
			switch {
			case exposeDue && svc.agent.ActionPoints >= 20:
				exposeBurst(m, 6, 0.6)
			case resourceAmount(svc.silo, "Supplies") < 2500:
				ration()
			case !exposeDue || svc.agent.ActionPoints >= 35:
				if svc.agent.ActionPoints >= 15 && playerCoreBelief(svc, "Judicial_1") < 0.9 &&
					playerCoreBelief(svc, "Judicial_1") >= 0.6 {
					attempt("investigate_dispatch", 15)
				}
			}
		}
	}

	status := svc.silo.VictoryStatus
	if status == nil {
		t.Fatalf("bedrock PERSONAL_RULE line did not settle within the horizon: W=%.2f suspicion=%.3f exposes=%d",
			svc.silo.Authority.W, svc.agent.SuspicionLevel, exposes)
	}
	if status.Decision == nil {
		t.Fatalf("bedrock PERSONAL_RULE settlement has no arbitration decision: %+v", status)
	}
	if status.Decision.Selected != endingPersonalRule {
		t.Fatalf("bedrock route selected %q instead of PERSONAL_RULE at %d-%02d: W=%.2f suspicion=%.3f exposes=%d omen=%v offer=%v done=%v execution=%v pool=%v scores=%v excluded=%v",
			status.Decision.Selected, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Authority.W,
			svc.agent.SuspicionLevel, exposes,
			contentTriggered(svc, "crisis:reset_omen_requisition"),
			contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER"),
			contentTriggered(svc, "player_action:reset_countermeasure_done"),
			contentTriggered(svc, "crisis:reset_execution"),
			status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
	}
	if svc.silo.Authority.W > 30 {
		t.Fatalf("PERSONAL_RULE settled with W=%.2f above the 30 bar", svc.silo.Authority.W)
	}
	t.Logf("bedrock PERSONAL_RULE (seed %d): settled %d-%02d in %d months, W_min=%.2f W_settle=%.2f suspicion_peak=%.3f exposes=%d succession=%v pool=%v",
		seed, svc.silo.CurrentYear, svc.silo.CurrentMonth, tr.months(svc), tr.wMin, svc.silo.Authority.W,
		tr.susMax, exposes, successionTaken, status.Decision.Pool)
}

// TestBedrockInformationLinePlayable 磐石档《共证》可达性证据：覆写钉档、种子
// 2401、司法特工 (隐秘行事)、122 年开局。策略是无声探查矩阵磐石组合
// (runSilentProbeInformationRoute) 的原型适配形态，并记录标定关键量。
//
// 为什么磐石档的信息线要走无声探查形态 (实测)：无特质参考路线在磐石档两次
// 塌进《除名》——(i) 发布完成后联盟侵蚀归零，IT 资源全转反击 (约 7 个月一次
// +0.08)；(ii) Mayor 零人口部门的第 3 份成熟事实必须靠后期扩池探查补给，
// SAMPLED 探查 ×1.25 的怀疑叠加反击在渗透成熟前越过 1.0 硬线。用《人治》
// 宿主的曝光掩护杠杆可以压住反击，但压不住扩池探查自身的怀疑。无声探查把
// 扩池与加深确信移出抽样路径，正是 19 §3.2 "《共证》的隐匿簇从开局就是紧约
// 束" 的玩法投影：磐石档信息线变贵，贵在该花 2×AP 买免怀疑。
func TestBedrockInformationLinePlayable(t *testing.T) {
	const seed = int64(2401)
	svc := newGatedService(t, filepath.Join(t.TempDir(), "bedrock-information.db"))
	req := bedrockRequest(seed, "Judicial", 0)
	req.TraitIds = []string{"shadowy"}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeBedrock {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeBedrock)
	}
	tr := newBedrockTracker(svc)
	circulationActions := 0
	silentUses := 0
	count := func(action model.AgentAction) {
		circulationActions++
		if action.Type == model.ActionSilentProbe {
			silentUses++
		}
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
		count(action)
	}
	tryExecute := func(action model.AgentAction) bool {
		outcome, err := svc.ExecuteAction(action)
		if err != nil || !outcome.Result.Executed {
			return false
		}
		count(action)
		return true
	}
	passTime := func() {
		t.Helper()
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bedrock INFORMATION line: %v", err)
		}
		tr.sample(svc)
	}
	// 服务端权威成本口径 (含规范冲突上浮；无声探查 = 2×)。
	resolvedCost := func(actionType model.AgentActionType, target string) float64 {
		t.Helper()
		action := model.AgentAction{Type: actionType, TargetDept: target}
		if err := svc.resolveActionCost(&action); err != nil {
			t.Fatalf("resolve cost for %s: %v", actionType, err)
		}
		return action.Cost
	}

	// 开局：无声探查越域取材 (计入首观察键，打开 first_signal)。
	execute(model.AgentAction{Type: model.ActionSilentProbe, TargetDept: "Mayor"})
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("silent probe did not open the line: %s", svc.world.ProgressStage)
	}
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	// 等真实法警观察链把 Judicial_2 抬到发布地板 (与参考路线同口径)。
	for month := 0; month < 12 && svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 && !svc.gameOver(); month++ {
		passTime()
	}
	publishFloor := engine.DefaultInformationConfig().QualPublish
	judicialBelief := engine.BeliefFor(svc.agent, "Judicial_2")
	if judicialBelief == nil || judicialBelief.B < publishFloor {
		t.Fatalf("real months did not reach the Judicial_2 publish floor %.2f: belief=%+v", publishFloor, judicialBelief)
	}
	if !contentTriggered(svc, "special:bailiff_dispatch_evidence_b") {
		t.Fatal("special:bailiff_dispatch_evidence_b did not fire through real monthly progression")
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
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		return err == nil && outcome.Result.Executed
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
		acted := false
		if verifiable && svc.agent.ActionPoints >= 15 {
			acted = attempt("verify_evidence", 15)
		}
		if publishable && publishCooldownReady() && svc.agent.ActionPoints >= 20 {
			acted = attempt("publish_evidence", 20) || acted
		}
		// 双事实发布 (26 §四 B)：无核验/发布可做时，用官方调查把较弱链
		// (Judicial_1) 抬到发布地板，而不是让路线只发一条事实、再靠流通
		// 补 18 个槽。同时必须存蓄：待发布事实存在时不要再花 15 AP 调查，
		// 否则 AP 被月收入钉在 20 以下，第二条事实永远发不出去 (裂隙档
		// 实测 40 个月只发出一条)。
		pendingPublish := false
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if published[fact] {
				continue
			}
			verified := false
			for _, v := range svc.silo.VerifiedEvidence {
				if v == fact {
					verified = true
				}
			}
			belief := engine.BeliefFor(svc.agent, fact)
			if verified && belief != nil && belief.B >= publishFloor {
				pendingPublish = true
			}
		}
		if !acted && !pendingPublish && !verifiable && svc.agent.ActionPoints >= 15 {
			attempt("investigate_dispatch", 15)
		}
		passTime()
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

	// 陪同回查 (Judicial 特工开局关系即满足 0.10 门)。
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
		passTime()
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})

	// 流通循环：9 部门各 3 个槽位成熟前持续推进 (渗透是纯时间门)；两条核心
	// 事实发布后每个部门只差一个第三槽。加深确信与扩池探查优先无声探查；
	// 叛乱过热时蛰伏。
	shareCost := model.ACTION_COSTS[model.ActionShareInfo]
	gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
	shareAttempts := map[string]int{}
	upkeepAttempts := map[string]int{}
	// 半成品种子续播：同一事实没补过地板前不换种 (见 info_endgame 同名字段)。
	pendingSeed := map[string]string{}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	// 分享成本按服务端权威口径；监视期接触不可用视作无穷贵。
	shareCostFor := func(name string) float64 {
		if engine.ContactUnderWatch(svc.silo, name) {
			return math.Inf(1)
		}
		return resolvedCost(model.ActionShareInfo, name)
	}
	// 监视者排序 (26 §四 E)：IT/Judicial 优先，Mayor 次之。
	priority := func(name string) int {
		switch name {
		case "IT", "Judicial":
			return 0
		case "Mayor":
			return 1
		}
		return 2
	}
	allMatured := func() bool {
		for i := range svc.silo.Professions {
			if informationMaturedFacts(&svc.silo.Professions[i]) < informationGateFactCount {
				return false
			}
		}
		return true
	}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		passTime()
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if allMatured() || svc.silo.Rebellion >= 0.7 {
			continue
		}
		gatherFrom := rotation[month%len(rotation)]
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
				_, endangered := endangeredStreakCandidateWithin(svc.agent, dept, upkeepAttempts, 0.06)
				pendingFact, hasPending := pendingSeed[dept.Name]
				pendingBelowFloor := false
				if hasPending {
					if belief := professionBelief(dept, pendingFact); belief == nil ||
						belief.B < engine.DefaultInformationConfig().QualRumor {
						pendingBelowFloor = true
					}
				}
				// AP 存蓄 (同 info_endgame)：IT/Judicial 的分享成本翻倍，
				// 月收入却只有 ≈3；买不起时整月不花，留给优先部门，否则
				// 刚过地板的槽会在维护前断流。
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
				// 播种截止：以日历为准——动作自带时长，循环序号会漂移；
				// 结算 Y130M1 前不足 30 月的新种已无法成熟。
				if 130*12+1-(svc.silo.CurrentYear*12+svc.silo.CurrentMonth) <= 30 {
					continue
				}
				// 建联已整体移除 (26 §九 偏差 1)：1 月时长 × 36 次会把播种
				// 窗口吃光，而吸收公式由 B_player 主导，关系 0.2→0.5 只差
				// ≈0.1；把 AP/日历直接换成分享与深填。
				fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					return shareAttempts[dept.Name+":"+fact] >= 4
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
					shareAttempts[dept.Name+":"+fragment]++
					if tryExecute(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						// 深填 (26 §四 D)：新槽连续分享到 B ≥ 0.45，正好滑行
						// 完 30 月渗透窗；浅播会在 10 月内断流。
						for shareAttempts[dept.Name+":"+fragment] < 6 &&
							svc.agent.ActionPoints >= shareCostFor(dept.Name) &&
							!engine.ContactUnderWatch(svc.silo, dept.Name) {
							belief := professionBelief(dept, fragment)
							if belief == nil || belief.B >= 0.45 {
								break
							}
							shareAttempts[dept.Name+":"+fragment]++
							if !tryExecute(model.AgentAction{
								Type: model.ActionShareInfo, TargetDept: dept.Name,
								FragmentIds: []string{fragment},
							}) {
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
		t.Fatalf("route must actually use the silent probe on bedrock, used %d times", silentUses)
	}
	for _, key := range []string{
		"player_actions:receiver_verify_accompany",
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the bedrock route (Q=%v, evidence=%+v)", key,
				svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingInformation {
		excluded := ""
		if svc.silo.VictoryStatus != nil && svc.silo.VictoryStatus.Decision != nil {
			excluded = svc.silo.VictoryStatus.Decision.Excluded[endingInformation]
		}
		t.Fatalf("bedrock route did not settle as INFORMATION at %d-%02d: excluded=%q status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, excluded, svc.silo.VictoryStatus)
	}
	t.Logf("bedrock INFORMATION (seed %d): settled %d-%02d in %d months, W_min=%.2f suspicion_peak=%.3f rebellion=%.3f silent_uses=%d",
		seed, svc.silo.CurrentYear, svc.silo.CurrentMonth, tr.months(svc), tr.wMin, tr.susMax, svc.silo.Rebellion, silentUses)
}

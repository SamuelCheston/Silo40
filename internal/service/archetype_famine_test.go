package service

// archetype_famine_test.go —— 19-opening-variability §3.2/§6.2/§6.3/§8 荒年档
// (资源匮乏) 的标定验收，只钉荒年档：
//
//  1. 旋钮终值锁：ResourceFactor / CrisisCooldownFactor 的标定值与方向
//     (边界标定过程与各方向打破组合见 internal/engine/archetype.go 荒年条目
//     注释与 feat/ov-arch-famine 提交记录)。
//  2. 危机时点：同种子对照，荒年档首个资源危机严格早于无原型基线，且不早于
//     第 1 月玩家行动窗口 (19 §8：危机提前量以"首月可感知、不早于玩家首次
//     行动窗口"为界)。
//  3. 可达性：Archetype="famine" 覆写钉档，固定种子无头真实打到结局结算
//     (不白盒)：《越岭》(荒年重压线——维生挤占日历)、《公议》(土著组合)、
//     《共证》(无声探查组合) 各一个通关实例，记录结算月数与资源轨迹最低点。
//
// 全部状态经真实动作与真实月度推进产生；固定种子可复现，不引入新随机源。

import (
	"math"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 资源阈值类危机是荒年档"危机提前"的可感知面。判别事件取
// energy_rolling_blackout (Energy≤300，19 §8 的例示事件) 与 food_poisoning
// (Supplies≤2000)。water_leak (Materials≤800) 不作判别：材料经济在两种配置下
// 都在第 1 月内击穿阈值 (开局 2000/1200 对 ~数千/月的净消耗)，触发月无档间
// 差异，只在报告里记录。
const (
	famineTimingBlackoutKey = "special:energy_rolling_blackout"
	famineTimingFoodKey     = "crisis:food_poisoning"
)

// famineResourceTrack 逐月采样的资源轨迹：三线可达性证据记录的"资源轨迹
// 最低点"。采样只读真实月度推进后的库存，不写任何状态。
type famineResourceTrack struct {
	MinEnergy         float64
	MinMaterials      float64
	MinSupplies       float64
	MinEnergyMonth    int
	MinMaterialsMonth int
	MinSuppliesMonth  int
	startAbs          int
}

func newFamineResourceTrack(silo *model.Silo) *famineResourceTrack {
	return &famineResourceTrack{
		MinEnergy:    math.Inf(1),
		MinMaterials: math.Inf(1),
		MinSupplies:  math.Inf(1),
		startAbs:     silo.CurrentYear*12 + silo.CurrentMonth,
	}
}

func (tr *famineResourceTrack) sample(silo *model.Silo) {
	month := silo.CurrentYear*12 + silo.CurrentMonth - tr.startAbs
	if e := resourceAmount(silo, "Energy"); e < tr.MinEnergy {
		tr.MinEnergy = e
		tr.MinEnergyMonth = month
	}
	if m := resourceAmount(silo, "Materials"); m < tr.MinMaterials {
		tr.MinMaterials = m
		tr.MinMaterialsMonth = month
	}
	if s := resourceAmount(silo, "Supplies"); s < tr.MinSupplies {
		tr.MinSupplies = s
		tr.MinSuppliesMonth = month
	}
}

// TestFamineKnobsLockedAtCalibratedValues 锁荒年档两旋钮的标定终值与方向
// (19 §3.2)：荒年=资源匮乏——水位更低 (<1)、危机更早更密 (<1)；其余轴保持
// 基线 (零值=不动)。标定边界证据见 internal/engine/archetype.go 荒年注释。
func TestFamineKnobsLockedAtCalibratedValues(t *testing.T) {
	cfg, ok := engine.ArchetypeByID(engine.ArchetypeFamine)
	if !ok {
		t.Fatal("famine archetype must stay registered")
	}
	if cfg.ResourceFactor != 0.93 {
		t.Fatalf("famine ResourceFactor = %v, want calibrated 0.93 (边界标定见 archetype.go 注释)", cfg.ResourceFactor)
	}
	if cfg.CrisisCooldownFactor != 0.9 {
		t.Fatalf("famine CrisisCooldownFactor = %v, want calibrated 0.9 (边界标定见 archetype.go 注释)", cfg.CrisisCooldownFactor)
	}
	// 方向锁：荒年=资源匮乏——水位更低、危机更早更密。
	if cfg.ResourceFactor >= 1 || cfg.CrisisCooldownFactor >= 1 {
		t.Fatalf("famine knobs must press below baseline: resource=%v cooldown=%v",
			cfg.ResourceFactor, cfg.CrisisCooldownFactor)
	}
	// 单一压力源锁 (19 §3.1)：荒年只动资源轴——W/检测/波动轴保持基线零值。
	if cfg.W0Delta != 0 || cfg.WHalfLifeMonths != 0 || cfg.DetectionFactor != 0 || cfg.PanicRebellionAmp != 0 {
		t.Fatalf("famine must only press the resource axis: %+v", cfg)
	}
}

// resourceCrisisFireMonths 零输入推进，逐月观测判别危机的首次触发月
// (1 = 首次 PassTime 结算)。推进前 (建档时) 不得已有危机触发——那是"早于
// 玩家首次行动窗口"的显式失败。返回 key→首次触发月 (0 = 上限内未触发)。
func resourceCrisisFireMonths(t *testing.T, svc *GameService, capMonths int) map[string]int {
	t.Helper()
	for _, key := range []string{famineTimingBlackoutKey, famineTimingFoodKey} {
		if contentTriggered(svc, key) {
			t.Fatalf("resource crisis %s already triggered at game creation — before the player's first action window", key)
		}
	}
	fires := map[string]int{}
	for m := 1; m <= capMonths; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance crisis-timing run at month %d: %v", m, err)
		}
		for _, key := range []string{famineTimingBlackoutKey, famineTimingFoodKey} {
			if fires[key] == 0 && contentTriggered(svc, key) {
				fires[key] = m
			}
		}
		if svc.gameOver() {
			break
		}
	}
	return fires
}

// TestFamineFirstResourceCrisisTiming 锁 19 §8 危机提前量边界：同种子对照，
// 荒年档首个资源危机 (如 energy_rolling_blackout，Energy≤300 触发) 严格早于
// 无原型基线，且不早于第 1 月玩家行动窗口；首月资源条即可感知差异。
//
// 基线臂构造：无原型 = 原型字段零值 = 现行行为逐位不变 (19 §3.3)。建档口
// 径里覆写字段必填且只认已注册档位，故基线臂用"荒年档两旋钮置于基线值
// (1.0/1.0)"的覆写实现——两钮 1.0 时 ApplyArchetypeOpening 与
// CrisisCooldownScale 全部走"不动"分支，行为与无原型逐位一致。
func TestFamineFirstResourceCrisisTiming(t *testing.T) {
	// 种子 2401 号数派生为裂隙档 (16 号)——两臂都靠覆写钉档，排除号数派生
	// 干扰；同种子同号数，差异只来自荒年旋钮。
	const seed = int64(2401)
	const capMonths = 120
	newTimingGame := func(t *testing.T, name string) *GameService {
		t.Helper()
		svc := newGatedService(t, filepath.Join(t.TempDir(), name))
		req := threeSiloRequest(seed)
		req.Archetype = engine.ArchetypeFamine
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create game (%s): %v", name, err)
		}
		return svc
	}

	// 基线臂：荒年两钮置于基线值 1.0 = 无原型逐位行为。
	restore := engine.OverrideArchetypeConfigForTest(engine.ArchetypeFamine, engine.ArchetypeConfig{
		DisplayName:          "荒年",
		ResourceFactor:       1.0,
		CrisisCooldownFactor: 1.0,
	})
	baselineSvc := newTimingGame(t, "famine-timing-baseline.db")
	baselineEnergy := resourceAmount(baselineSvc.silo, "Energy")
	baselineMaterials := resourceAmount(baselineSvc.silo, "Materials")
	baselineSupplies := resourceAmount(baselineSvc.silo, "Supplies")
	baselineFires := resourceCrisisFireMonths(t, baselineSvc, capMonths)
	restore()
	if baselineFires[famineTimingBlackoutKey] == 0 {
		t.Fatalf("baseline energy blackout did not fire within %d months (seed %d): pick a seed where both arms are observable", capMonths, seed)
	}

	// 荒年臂：真实注册配置 (标定终值)。
	famineCfg, _ := engine.ArchetypeByID(engine.ArchetypeFamine)
	famineSvc := newTimingGame(t, "famine-timing-famine.db")
	if got := famineSvc.silo.Archetype; got != engine.ArchetypeFamine {
		t.Fatalf("override archetype = %q, want %q", got, engine.ArchetypeFamine)
	}
	// 首月可感知 (19 §8)：建档当月资源条即被压低，压幅=注册旋钮。
	for res, base := range map[string]float64{
		"Energy": baselineEnergy, "Materials": baselineMaterials, "Supplies": baselineSupplies,
	} {
		if got, want := resourceAmount(famineSvc.silo, res), base*famineCfg.ResourceFactor; got != want {
			t.Fatalf("famine opening %s = %v, want %v (baseline %v × %v)", res, got, want, base, famineCfg.ResourceFactor)
		}
		if resourceAmount(famineSvc.silo, res) >= base {
			t.Fatalf("famine opening %s must be visibly below baseline", res)
		}
	}
	famineOpeningEnergy := resourceAmount(famineSvc.silo, "Energy")
	famineFires := resourceCrisisFireMonths(t, famineSvc, capMonths)

	t.Logf("first resource crisis (seed %d, cap %d): blackout famine=m%d baseline=m%d; food_poisoning famine=m%d baseline=m%d; opening energy %.0f vs %.0f",
		seed, capMonths, famineFires[famineTimingBlackoutKey], baselineFires[famineTimingBlackoutKey],
		famineFires[famineTimingFoodKey], baselineFires[famineTimingFoodKey],
		famineOpeningEnergy, baselineEnergy)

	famineBlackout, baselineBlackout := famineFires[famineTimingBlackoutKey], baselineFires[famineTimingBlackoutKey]
	if famineBlackout == 0 {
		t.Fatalf("famine energy blackout did not fire within %d months", capMonths)
	}
	// 不早于第 1 月玩家行动窗口：首危机只能落在首次月度结算 (月 1) 或之后；
	// 建档时点 (月 0) 已有危机在 resourceCrisisFireMonths 里已显式失败。
	if famineBlackout < 1 {
		t.Fatalf("famine first crisis month %d precedes the month-1 player action window", famineBlackout)
	}
	if famineBlackout >= baselineBlackout {
		t.Fatalf("famine energy blackout (month %d) must precede the no-prototype baseline (month %d)",
			famineBlackout, baselineBlackout)
	}
	// 次判别事件同向：补给危机也必须提前 (两事件都测不到说明种子无判别力)。
	if famineFires[famineTimingFoodKey] == 0 || baselineFires[famineTimingFoodKey] == 0 ||
		famineFires[famineTimingFoodKey] >= baselineFires[famineTimingFoodKey] {
		t.Fatalf("famine food_poisoning (month %d) must precede the baseline (month %d)",
			famineFires[famineTimingFoodKey], baselineFires[famineTimingFoodKey])
	}
}

// ---------------------------------------------------------------------------
// 可达性证据：三线荒年钉档真实通关 (结算月数 + 资源轨迹最低点)。
// 策略直接复用各自已实测的荒年/原型适配跑法 (《越岭》走 playLine 共享)，
// 仅增量：建档覆写钉荒年档、逐月资源采样、结算月数与轨迹最低点回报。
// ---------------------------------------------------------------------------

// newFamineEscapeRunner 与 newEscapeRunner 同口径，唯建档以覆写钉荒年档。
func newFamineEscapeRunner(t *testing.T, line string, seed int64, prepLevel string) *escapeRunner {
	t.Helper()
	r := newEscapeRunner(t, line, seed, prepLevel, engine.ArchetypeFamine)
	if r.svc.silo.Archetype != engine.ArchetypeFamine {
		t.Fatalf("archetype override = %q, want %q", r.svc.silo.Archetype, engine.ArchetypeFamine)
	}
	return r
}

// TestFamineReachabilityRebellionLine 可达性证据·《越岭》(REBELLION)：荒年
// 重压线——维生挤占日历。策略直接复用 runEscapeLine 的荒年重调跑法
// (playLine 共享：渗透筹备＋确认接应＋备齐 20 套防护服即出发)，仅增量：
// 建档覆写钉档与逐月资源采样。种子 2402 的号数派生亦为荒年 (17 号)，
// 覆写与派生在此钉同一档。
func TestFamineReachabilityRebellionLine(t *testing.T) {
	const seed = int64(2402)
	r := newFamineEscapeRunner(t, "famine-reachability-rebellion", seed, "full")
	svc := r.svc
	track := newFamineResourceTrack(svc.silo)
	track.sample(svc.silo)
	r.monthHook = func() { track.sample(svc.silo) }
	r.playLine("full")
	rec := r.rec
	track.sample(svc.silo)

	t.Logf("famine 《越岭》 reachability: seed=%d settled_month=%d ending=%s won=%v departed=%d survivors=%d months_out=%d stance_at_launch=%s; resource mins: energy=%.0f@m%d materials=%.0f@m%d supplies=%.0f@m%d",
		seed, rec.SettledMonth, rec.Ending, rec.Won, rec.Departed, rec.Survivors, rec.MonthsOut,
		rec.StanceAtLaunch, track.MinEnergy, track.MinEnergyMonth, track.MinMaterials,
		track.MinMaterialsMonth, track.MinSupplies, track.MinSuppliesMonth)

	if rec.LaunchMonth == 0 || rec.Departed <= 0 {
		t.Fatalf("famine 《越岭》 did not reach a real departure: launchMonth=%d departed=%d excluded=%v",
			rec.LaunchMonth, rec.Departed, rec.Excluded)
	}
	if rec.Ending != endingRebellion || !rec.Won {
		t.Fatalf("famine 《越岭》 settled %q (won=%v), want REBELLION; pool=%v excluded=%v reason=%q",
			rec.Ending, rec.Won, rec.Pool, rec.Excluded, rec.Reason)
	}
	if rec.MonthsOut < 1 {
		t.Fatalf("ending settled before the group lived one outside month: monthsOut=%d demise=%q",
			rec.MonthsOut, rec.Demise)
	}
	if !rec.GroupSurvivingAtSettlement || rec.Demise != "" {
		t.Fatalf("famine 《越岭》 settled on a dead escaped group: surviving=%v demise=%q",
			rec.GroupSurvivingAtSettlement, rec.Demise)
	}
	if rec.GroupPopulationAtSettlement < outsideMinGroupSize {
		t.Fatalf("settled group never cleared the %d-person disband judgment: population=%d",
			outsideMinGroupSize, rec.GroupPopulationAtSettlement)
	}
	if rec.SettledMonth <= 0 {
		t.Fatalf("settled month must be recorded: %+v", rec)
	}
}

// driveFamineNativeRuleOfLawRouteWithTrack 《公议》可达性证据跑法：逐字复刻
// driveNativeRuleOfLawRoute 的土著 × 荒年实测路线 (关系门豁免跳过攒关系期，
// 其余条件全部经真实动作与月度推进)，仅增量：逐月资源采样与轨迹回报。
// 建档口径不变：原型覆写钉荒年档 + 精英职业土著 + StartYear 120。
func driveFamineNativeRuleOfLawRouteWithTrack(t *testing.T, profession string, seed int64) (int, *famineResourceTrack) {
	t.Helper()
	archetype := engine.ArchetypeFamine
	svc := newGatedService(t, filepath.Join(t.TempDir(), "famine-reachability-rol.db"))
	if _, err := svc.CreateGame(nativeRuleOfLawRequest(seed, archetype, profession)); err != nil {
		t.Fatalf("create game (%s/%s): %v", archetype, profession, err)
	}
	if svc.silo.Archetype != archetype {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, archetype)
	}
	track := newFamineResourceTrack(svc.silo)
	track.sample(svc.silo)
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

	// 接触阶段：精英土著免引荐——不做任何 Build Network，只等真实断点点火。
	var offer *model.PlayerActionMeta
	for i := 0; i < 48 && !svc.gameOver(); i++ {
		offer = actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_OFFER")
		if offer != nil && offer.Enabled {
			break
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance for contact card (%s): %v", archetype, err)
		}
		track.sample(svc.silo)
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
	const decisiveWGate = 42.0
	for month := 1; month <= 140 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance %s line at month %d: %v", archetype, month, err)
		}
		track.sample(svc.silo)
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
		gameMonth := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
		if windowOpenAt == 0 && svc.silo.Authority.W <= 18 &&
			svc.silo.Authority.Judicial.PublicEvidenceQ >= 0.85 &&
			svc.silo.Authority.Judicial.IndependentPublicFacts >= 2 {
			windowOpenAt = month
		}

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
		track.sample(svc.silo)
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
	if status.Decision.Selected != endingRuleOfLaw {
		t.Fatalf("%s: native route selected %q instead of RULE_OF_LAW: pool=%v scores=%v excluded=%v",
			archetype, status.Decision.Selected, status.Decision.Pool, status.Decision.Scores, status.Decision.Excluded)
	}
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("%s: terminal RULE_OF_LAW state accepted another action: %+v", archetype, outcome.Result)
	}
	t.Logf("%s native RULE_OF_LAW reachability route settled: date=%d-%02d W=%.2f Q=%.4f facts=%d pact_locked=%v window_open_month=%d exposes=%d publishW=%.2f impactW=%.2f pool=%v scores=%v",
		archetype, svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.Authority.W,
		svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.Authority.Judicial.IndependentPublicFacts,
		svc.silo.Authority.Judicial.PactLocked, windowOpenAt, exposes, decisivePublishW, decisiveImpactW,
		status.Decision.Pool, status.Decision.Scores)
	return svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs, track
}

// TestFamineReachabilityRuleOfLawLine 可达性证据·《公议》(RULE_OF_LAW)：
// 土著 × 荒年已实测组合 (种子 4301 / Judicial，覆写钉档)。记录结算月数与
// 资源轨迹最低点。
func TestFamineReachabilityRuleOfLawLine(t *testing.T) {
	const seed = int64(4301)
	months, track := driveFamineNativeRuleOfLawRouteWithTrack(t, "Judicial", seed)
	t.Logf("famine 《公议》 reachability: seed=%d profession=Judicial settled_months=%d; resource mins: energy=%.0f@m%d materials=%.0f@m%d supplies=%.0f@m%d",
		seed, months, track.MinEnergy, track.MinEnergyMonth, track.MinMaterials,
		track.MinMaterialsMonth, track.MinSupplies, track.MinSuppliesMonth)
	if months <= 0 {
		t.Fatalf("settled months must be recorded, got %d", months)
	}
}

// runFamineSilentInformationRouteWithTrack 《共证》可达性证据跑法：逐字复刻
// runSilentProbeInformationRoute 的无声探查 × 荒年实测路线 (开局与加深确信
// 用无声探查，流通/核验/发布照旧，渗透成熟后停手)，仅增量：逐月资源采样与
// 结算月数/轨迹回报。建档口径不变：原型覆写钉荒年档 + shadowy 特质。
func runFamineSilentInformationRouteWithTrack(t *testing.T, seed int64) (int, *famineResourceTrack) {
	t.Helper()
	archetype := engine.ArchetypeFamine
	svc := newGatedService(t, filepath.Join(t.TempDir(), "famine-reachability-info.db"))
	req := threeSiloRequest(seed)
	req.Profession = "Judicial"
	req.Archetype = archetype
	req.TraitIds = []string{"shadowy"}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != archetype {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, archetype)
	}
	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	track := newFamineResourceTrack(svc.silo)
	track.sample(svc.silo)
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
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bailiff at month %d: %v", month, err)
		}
		track.sample(svc.silo)
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
		track.sample(svc.silo)
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
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("recover AP for receiver verification: %v", err)
		}
		track.sample(svc.silo)
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})

	// 流通循环：9 部门各 3 槽落齐后停手等渗透成熟。
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
		track.sample(svc.silo)
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
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"}); err == nil {
		t.Fatalf("terminal INFORMATION state accepted another action: %+v", outcome.Result)
	}
	return svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs, track
}

// TestFamineReachabilityInformationLine 可达性证据·《共证》(INFORMATION)：
// 无声探查 × 荒年已实测组合 (种子 2404 / Judicial，覆写钉档)。记录结算月数
// 与资源轨迹最低点。
func TestFamineReachabilityInformationLine(t *testing.T) {
	const seed = int64(2404)
	months, track := runFamineSilentInformationRouteWithTrack(t, seed)
	t.Logf("famine 《共证》 reachability: seed=%d profession=Judicial settled_months=%d; resource mins: energy=%.0f@m%d materials=%.0f@m%d supplies=%.0f@m%d",
		seed, months, track.MinEnergy, track.MinEnergyMonth, track.MinMaterials,
		track.MinMaterialsMonth, track.MinSupplies, track.MinSuppliesMonth)
	if months <= 0 {
		t.Fatalf("settled months must be recorded, got %d", months)
	}
}

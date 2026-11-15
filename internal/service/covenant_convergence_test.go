package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 裁决 (W 公式与约章目标不兼容的岔口) 的收束跑验：
//
//  1. 只有日常压力 (持续施压但无决定性事件) → 约章不成；
//  2. 日常压力 + 一次决定性事件 (大规模公开证据冲击) → 约章成。
//
// 两个场景共用 §7.4 校准的开局 (calibrationService) 与同一套技巧策略，
// 唯一变量是是否发布第二份核心证据——第二份合格公开事实登记本身就是
// "大规模公开证据" 冲击的触发源。

func factVerified(silo *model.Silo, factID string) bool {
	for _, fact := range silo.VerifiedEvidence {
		if fact == factID {
			return true
		}
	}
	return false
}

func factPublished(silo *model.Silo, factID string) bool {
	for _, record := range silo.PublicEvidence {
		if record.FactID == factID {
			return true
		}
	}
	return false
}

// TestCovenantConvergenceDailyPressureAloneFails 锁定"日常压力不成"：只发布
// 第一份核心证据 (合格事实数始终 = 1)，三类决定性冲击源一个都不出现，
// 技巧策略持续施压 20 个月也到不了 W≤18。
func TestCovenantConvergenceDailyPressureAloneFails(t *testing.T) {
	svc := calibrationService(t, 421)
	station := svc.silo

	station.Rebellion = 0.2
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if playerCoreBelief(svc, "Judicial_1") <= 0 {
		t.Fatal("evidence chain A did not seed player belief")
	}
	supportBelief := engine.EnsureBelief(svc.agent, "Judicial_3")
	engine.ObserveFact(engine.DefaultInformationConfig(), supportBelief,
		engine.FragmentSpecFor("Judicial_3"), "convergence:Judicial_3", 0.8)

	minW := station.Authority.W
	lowStreak, maxLowStreak := 0, 0
	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	startAbs := station.CurrentYear*12 + station.CurrentMonth

	// 32 months at the merged 36/yr AP economy (same stretch as the §7.4
	// calibration window).
	for gameMonth := 0; gameMonth <= 32; gameMonth = station.CurrentYear*12 + station.CurrentMonth - startAbs {
		// 叛乱通道硬钉在 0.2：本测试度量的是约章/W 动力学。煽动效应量落地后
		// NPC 月度煽动会真实推高叛乱并反噬 W 与信念接收，把外部通道钉住才能
		// 让断言继续只测它要测的东西 (判据未变：第二份发布仍须真实发生)。
		station.Rebellion = 0.2
		if station.DeptTension < 0.3 {
			station.DeptTension = 0.3
		}

		// Chain: investigate -> verify -> publish, Judicial_1 only. The second
		// core fact is deliberately never published, so the qualifying-fact
		// count stays at 1 and the large-scale-evidence shock never fires.
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(station, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			tryAct(t, svc, "verify_evidence", 15)
		}
		if playerCoreBelief(svc, "Judicial_1") >= 0.9 && factVerified(station, "Judicial_1") &&
			!factPublished(station, "Judicial_1") && svc.agent.ActionPoints >= 20 {
			tryAct(t, svc, "publish_evidence", 20)
		}
		if svc.agent.ActionPoints >= 15 && playerCoreBelief(svc, "Judicial_1") < 0.9 &&
			playerCoreBelief(svc, "Judicial_1") >= 0.6 {
			mustAct(t, svc, "investigate_dispatch", 15)
		}

		// Same everyday pressure as the full calibration: capability checks and
		// fact circulation with spare AP.
		if exposes < 8 && gameMonth-lastExposeMonth >= 3 && svc.agent.ActionPoints >= 25 &&
			playerCoreBelief(svc, "Judicial_1") >= 0.6 {
			mustAct(t, svc, "expose_execution_failure", 20)
			exposes++
			lastExposeMonth = gameMonth
		}
		for _, dept := range []string{"Mechanical", "Supply"} {
			for _, fact := range []string{"Judicial_1", "Judicial_3"} {
				if playerCoreBelief(svc, fact) < 0.6 || svc.agent.ActionPoints < 25 || shares[dept+":"+fact] >= 2 {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept,
					FragmentIds: []string{fact}, Cost: 20,
				})
				if err == nil && outcome.Result.Executed {
					shares[dept+":"+fact]++
				}
			}
		}

		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", gameMonth, err)
		}
		w := station.Authority.W
		if w < minW {
			minW = w
		}
		if w <= 18 {
			lowStreak++
			if lowStreak > maxLowStreak {
				maxLowStreak = lowStreak
			}
		} else {
			lowStreak = 0
		}
	}

	env := engine.ComputeInformationEnvironment(station)
	t.Logf("daily-only: minW=%.2f finalW=%.2f E=%.4f Q=%.4f qualifying=%d maxLowStreak=%d exposes=%d shocks=%+v",
		minW, station.Authority.W, station.Authority.E, env.PublicEvidenceQ, env.QualifyingFacts,
		maxLowStreak, exposes, station.Authority.DecisiveShocks)

	if len(station.Authority.DecisiveShocks) != 0 {
		t.Fatalf("daily pressure must not produce decisive shocks: %+v", station.Authority.DecisiveShocks)
	}
	if env.QualifyingFacts >= engine.DefaultAuthorityConfig().PactMinFacts {
		t.Fatalf("daily-only scenario unexpectedly reached %d qualifying facts", env.QualifyingFacts)
	}
	if minW <= 18 {
		t.Fatalf("ruling violated: daily pressure alone reached W=%.2f (covenant gate 18)", minW)
	}
	if maxLowStreak >= engine.DefaultAuthorityConfig().PactEntryMonths {
		t.Fatalf("daily pressure sustained the covenant gate for %d months", maxLowStreak)
	}
	if station.Authority.Judicial.PactLocked {
		t.Fatal("pact locked without a decisive event")
	}
}

// TestCovenantConvergenceDecisiveEventSucceeds 锁定"决定性事件促成约章"：
// 裁决的情形是"日常压力＋一次决定性事件"——先只发布第一份核心证据，
// 以核查、曝光和流通建立日常压力；随后储备同一结算月所需的 AP，发布
// 第二份核心证据并立即曝光。第二份登记触发的大规模公开证据冲击与普通
// 负冲击共同击穿日常谷底，持续低于约章门后 PactLocked 成立。发布门只读
// 已完成的动作与资源，不再等待一个必须由本次发布自身才能产生的 W 谷底。
func TestCovenantConvergenceDecisiveEventSucceeds(t *testing.T) {
	svc := calibrationService(t, 421)
	station := svc.silo
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v", action, err)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v", action, outcome.Result)
		}
	}

	station.Rebellion = 0.2
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	if playerCoreBelief(svc, "Judicial_1") <= 0 {
		t.Fatal("evidence chain A did not seed player belief")
	}
	supportBelief := engine.EnsureBelief(svc.agent, "Judicial_3")
	engine.ObserveFact(engine.DefaultInformationConfig(), supportBelief,
		engine.FragmentSpecFor("Judicial_3"), "convergence:Judicial_3", 0.8)

	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	startAbs := station.CurrentYear*12 + station.CurrentMonth
	decisiveFired := false
	minW := station.Authority.W
	decisivePublishW, decisiveImpactW := 0.0, 0.0
	judicialBelief := func(factID string) float64 {
		for i := range station.Professions {
			if station.Professions[i].Name == "Judicial" {
				if belief := engine.BeliefIn(&station.Professions[i].Beliefs, factID); belief != nil {
					return belief.B
				}
			}
		}
		return 0
	}

	// Establish daily pressure first, then combine the second publication with
	// one final exposure so the ordinary and decisive shocks settle together.
	// Keep the existing 96-month harness cap for the divergent peer-silo rhythm.
	// 96 is a test bound, not the game length or a standard window (ADR-0018).
	for gameMonth := 0; gameMonth <= 96 && !station.Authority.Judicial.PactLocked && !svc.gameOver(); gameMonth = station.CurrentYear*12 + station.CurrentMonth - startAbs {
		if station.Rebellion < 0.2 {
			station.Rebellion = 0.2
		}

		if station.DeptTension < 0.3 {
			station.DeptTension = 0.3
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}

		// Establish everyday pressure first, then publish the second core fact as
		// the decisive blow. The release gate is the pressure campaign itself, not
		// a W trough that only the second publication's own shock can produce.
		beliefFloor := engine.DefaultInformationConfig().PactFactBelief
		// Circulate before evaluating the release gate. Otherwise a share at the
		// end of the month merely crosses 0.85, next month's forgetting pulls it
		// back below the floor, and the reserved publication AP is never spent.
		if exposes >= 3 && !decisiveFired && factVerified(station, "Judicial_2") {
			for _, fact := range []string{"Judicial_1", "Judicial_2"} {
				shareCost := model.ACTION_COSTS[model.ActionShareInfo]
				if judicialBelief(fact) >= beliefFloor || svc.agent.ActionPoints < 40+shareCost || shares["Judicial:"+fact] >= 12 {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: "Judicial", FragmentIds: []string{fact},
				})
				if err == nil && outcome.Result.Executed {
					shares["Judicial:"+fact]++
				}
			}
		}
		secondPublishReady := exposes >= 3 && svc.agent.ActionPoints >= 40 && gameMonth-lastExposeMonth >= 3 &&
			judicialBelief("Judicial_1") >= beliefFloor && judicialBelief("Judicial_2") >= beliefFloor

		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(station, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			tryAct(t, svc, "verify_evidence", 15)
		}
		if factPublished(station, "Judicial_1") && !factVerified(station, "Judicial_2") &&
			playerCoreBelief(svc, "Judicial_2") >= 0.85 && svc.agent.ActionPoints >= 15 {
			tryAct(t, svc, "verify_evidence", 15)
		}
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if fact == "Judicial_2" && !secondPublishReady {
				continue
			}
			if playerCoreBelief(svc, fact) >= 0.9 && factVerified(station, fact) &&
				!factPublished(station, fact) && svc.agent.ActionPoints >= 20 {
				if tryAct(t, svc, "publish_evidence", 20) && fact == "Judicial_2" {
					decisiveFired = true
					decisivePublishW = station.Authority.W
				}
				break
			}
		}

		// Investigate whichever chain is behind; both must reach the publish floor.
		if svc.agent.ActionPoints >= 15 &&
			(playerCoreBelief(svc, "Judicial_1") < 0.9 || playerCoreBelief(svc, "Judicial_2") < 0.9) &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			mustAct(t, svc, "investigate_dispatch", 15)
		}

		exposeForPressure := !decisiveFired && exposes < 3 && svc.agent.ActionPoints >= 25
		exposeForDecisive := decisiveFired && svc.agent.ActionPoints >= 20
		if (exposeForPressure || exposeForDecisive) && gameMonth-lastExposeMonth >= 3 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			mustAct(t, svc, "expose_execution_failure", 20)
			exposes++
			lastExposeMonth = gameMonth
		}
		// Keep Judicial's live beliefs above the covenant evidence floor as well
		// as spreading to the large departments; publication alone now decays.
		for _, dept := range []string{"Judicial", "Mechanical", "Supply"} {
			for _, fact := range []string{"Judicial_1", "Judicial_2", "Judicial_3"} {
				shareCost := model.ACTION_COSTS[model.ActionShareInfo]
				maintainJudicial := dept == "Judicial" && fact != "Judicial_3" && exposes >= 3
				shareCap := 2
				if maintainJudicial {
					shareCap = 12
				}
				if playerCoreBelief(svc, fact) < 0.6 || svc.agent.ActionPoints < shareCost || shares[dept+":"+fact] >= shareCap {
					continue
				}
				if exposes >= 3 && !decisiveFired && (!maintainJudicial || svc.agent.ActionPoints < 40+shareCost) {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept,
					FragmentIds: []string{fact}, Cost: 20,
				})
				if err == nil && outcome.Result.Executed {
					shares[dept+":"+fact]++
				}
			}
		}

		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", gameMonth, err)
		}
		if station.Authority.W < minW {
			minW = station.Authority.W
		}
		if decisiveFired && decisiveImpactW == 0 {
			decisiveImpactW = station.Authority.W
		}
	}

	j := station.Authority.Judicial
	t.Logf("decisive: finalW=%.2f minW=%.2f Q=%.4f qualifying=%d pact=%.3f regime=%s exposes=%d shocks=%+v applied=%d decisiveFired=%v publishW=%.2f impactW=%.2f",
		station.Authority.W, minW, j.PublicEvidenceQ, j.IndependentPublicFacts, j.Pact, j.Regime,
		exposes, station.Authority.DecisiveShocks, station.Authority.DecisiveApplied, decisiveFired, decisivePublishW, decisiveImpactW)

	if !decisiveFired {
		t.Fatalf("the second publication never happened: W=%.2f beliefs=(%.2f, %.2f) evidence=%+v",
			station.Authority.W, playerCoreBelief(svc, "Judicial_1"), playerCoreBelief(svc, "Judicial_2"),
			station.PublicEvidence)
	}
	if !j.PactLocked {
		t.Fatalf("decisive event did not converge the covenant: W=%.2f Q=%.4f facts=%d pact=%.3f shocks=%+v",
			station.Authority.W, j.PublicEvidenceQ, j.IndependentPublicFacts, j.Pact, station.Authority.DecisiveShocks)
	}
	if j.Regime != "PACT" {
		t.Fatalf("regime after lock = %q, want PACT", j.Regime)
	}
	found := false
	for _, shock := range station.Authority.DecisiveShocks {
		if shock.Source == engine.DecisiveSourceMassEvidence {
			found = true
		}
	}
	if !found {
		t.Fatalf("large-scale-evidence shock missing from the audit trail: %+v", station.Authority.DecisiveShocks)
	}
	if station.Authority.DecisiveApplied < 1 {
		t.Fatal("decisive shock was recorded but never consumed")
	}
}

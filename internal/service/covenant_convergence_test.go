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
		if station.Rebellion < 0.2 {
			station.Rebellion = 0.2
		}
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
	t.Logf("daily-only: minW=%.2f finalW=%.2f E=%.4f Q=%.4f qualifying=%d maxLowStreak=%d shocks=%+v",
		minW, station.Authority.W, station.Authority.E, env.PublicEvidenceQ, env.QualifyingFacts,
		maxLowStreak, station.Authority.DecisiveShocks)

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
// 裁决的情形是"日常压力＋一次决定性事件"——先用同一技巧策略持续施压
// (只发布第一份核心证据、核查/曝光/流通把 W 压向日常谷底)，等 W 落到谷底
// 带后再发布第二份核心证据；第二份登记触发大规模公开证据冲击，击穿谷底
// 并持续低于约章门，PactLocked 成立。(冲击量级按谷底带推导，见
// 14-parameters-and-audit.md §9.1：冲击必须落在已被日常压力磨低的威势上。)
func TestCovenantConvergenceDecisiveEventSucceeds(t *testing.T) {
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

	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	startAbs := station.CurrentYear*12 + station.CurrentMonth
	decisiveFired := false

	// Phase 1 (daily pressure) runs until W reaches the trough band; phase 2
	// then publishes the second core fact as the decisive event. 96 个月上限：
	// 邻堡资源曲线分化后 (每堡自己的节奏)，单一 RNG 流的交织改变，脚本化
	// 策略到达谷底带与第二份发布的时点整体后移，收敛变慢但路径不变。
	for gameMonth := 0; gameMonth <= 96 && !station.Authority.Judicial.PactLocked; gameMonth = station.CurrentYear*12 + station.CurrentMonth - startAbs {
		if station.Rebellion < 0.2 {
			station.Rebellion = 0.2
		}
		if station.DeptTension < 0.3 {
			station.DeptTension = 0.3
		}
		// Skilled play answers the RESET omens (census -> requisition ->
		// execution) with the countermeasure route; without it the long
		// pressure campaign dies to 《遗忘》 before the covenant window.
		progressResetCountermeasureForRoute(t, svc, func(action model.AgentAction) {
			if _, err := svc.ExecuteAction(action); err != nil {
				t.Logf("countermeasure step %+v deferred: %v", action, err)
			}
		})
		if svc.gameOver() {
			break
		}

		troughReached := station.Authority.W <= 34

		// Verify/publish: Judicial_1 as soon as ready; Judicial_2 only once the
		// daily pressure has reached the trough (the decisive blow).
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(station, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			tryAct(t, svc, "verify_evidence", 15)
		}
		if troughReached && decisiveFired == false && playerCoreBelief(svc, "Judicial_2") >= 0.85 &&
			!factVerified(station, "Judicial_2") && svc.agent.ActionPoints >= 15 {
			tryAct(t, svc, "verify_evidence", 15)
		}
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if fact == "Judicial_2" && !troughReached {
				continue
			}
			if playerCoreBelief(svc, fact) >= 0.9 && factVerified(station, fact) &&
				!factPublished(station, fact) && svc.agent.ActionPoints >= 20 {
				if tryAct(t, svc, "publish_evidence", 20) && fact == "Judicial_2" {
					decisiveFired = true
				}
				break
			}
		}

		// Investigate whichever chain is behind (both must eventually reach the
		// publish floor; the second one waits for the trough to be published).
		if svc.agent.ActionPoints >= 15 &&
			(playerCoreBelief(svc, "Judicial_1") < 0.9 || playerCoreBelief(svc, "Judicial_2") < 0.9) &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			mustAct(t, svc, "investigate_dispatch", 15)
		}

		if exposes < 8 && gameMonth-lastExposeMonth >= 3 && svc.agent.ActionPoints >= 25 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			mustAct(t, svc, "expose_execution_failure", 20)
			exposes++
			lastExposeMonth = gameMonth
		}
		for _, dept := range []string{"Mechanical", "Supply"} {
			for _, fact := range []string{"Judicial_1", "Judicial_2", "Judicial_3"} {
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
	}

	j := station.Authority.Judicial
	t.Logf("decisive: finalW=%.2f Q=%.4f qualifying=%d pact=%.3f regime=%s shocks=%+v applied=%d decisiveFired=%v",
		station.Authority.W, j.PublicEvidenceQ, j.IndependentPublicFacts, j.Pact, j.Regime,
		station.Authority.DecisiveShocks, station.Authority.DecisiveApplied, decisiveFired)

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

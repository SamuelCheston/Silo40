package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
)

// 抉择即时后果验收（继承批次）：两个继承选项不再只是空转的完成键——
// 接受纯粹人治兑现"被保护的睡眠"（凝聚 +0.02）并松绑被搁置的历史追问
// （历史包袱 −0.02）；埋程序种子重新打开问题（历史包袱 +0.02）并机械
// 兑现"司法侧的程序派会注意到这些种子"（Judicial 收到 Judicial_1 观察）。
// 资格口径不变：人治收束加分仍读完成键，种子改道仍走 action_gates 边。

func gatedSuccessionService(t *testing.T) *GameService {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "succession-revisit.db"))
	if _, err := svc.CreateGame(threeSiloRequest(8021)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := svc.PassTime(1); err != nil {
		t.Fatalf("pass time: %v", err)
	}
	completeGateEvent(svc, "special:mayor_public_stand")
	return svc
}

// 接受纯粹人治：凝聚 +0.02、历史包袱 −0.02；收束资格事实不受影响。
func TestSuccessionPureRuleConsequences(t *testing.T) {
	svc := gatedSuccessionService(t)
	svc.silo.Cohesion = 0.5
	// 基线压在 0.08/0.12 阈值之下：ExecuteAction 内部会 drain 事件队列，
	// history_burden_awakened / archive_purge_order 一旦达标会混入额外增量。
	svc.silo.HistoryBurden = 0.05

	executePlayerEventAction(t, svc, "SUCCESSION_PURE_RULE")

	if !charterRevisitAlmostEqual(svc.silo.Cohesion, 0.52) {
		t.Fatalf("cohesion = %v, want 0.52", svc.silo.Cohesion)
	}
	if !charterRevisitAlmostEqual(svc.silo.HistoryBurden, 0.03) {
		t.Fatalf("history_burden = %v, want 0.03", svc.silo.HistoryBurden)
	}
}

// 埋程序种子：历史包袱 +0.02，司法职业层收到一次来自本动作的 Judicial_1 观察。
func TestSuccessionSeedProcedureConsequences(t *testing.T) {
	svc := gatedSuccessionService(t)
	svc.silo.HistoryBurden = 0.05 // 同上：避开 drain 期间的包袱阈值事件
	judicial := -1
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == "Judicial" {
			judicial = i
			break
		}
	}
	if judicial < 0 {
		t.Fatal("no Judicial profession in created silo")
	}
	beforeB := 0.0
	if before := engine.BeliefIn(&svc.silo.Professions[judicial].Beliefs, "Judicial_1"); before != nil {
		beforeB = before.B
	}

	executePlayerEventAction(t, svc, "SUCCESSION_SEED_PROCEDURE")

	if !charterRevisitAlmostEqual(svc.silo.HistoryBurden, 0.07) {
		t.Fatalf("history_burden = %v, want 0.07", svc.silo.HistoryBurden)
	}
	// beliefObserve 是加权更新而非直接赋值：只断言可信度被本动作的观察
	// 推高，且观察来源确实是 succession_seed_procedure。
	belief := engine.BeliefIn(&svc.silo.Professions[judicial].Beliefs, "Judicial_1")
	if belief == nil || belief.B <= beforeB {
		t.Fatalf("Judicial_1 belief = %+v, want B > %v", belief, beforeB)
	}
	observed := false
	for _, obs := range belief.Observations {
		if strings.Contains(obs, "succession_seed_procedure") {
			observed = true
			break
		}
	}
	if !observed {
		t.Fatalf("Judicial_1 observations %v missing succession_seed_procedure source", belief.Observations)
	}
}

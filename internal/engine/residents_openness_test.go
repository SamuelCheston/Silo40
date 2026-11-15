package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestOpennessErodesLoyaltyCeiling locks 《无人裁决》可达性的核心耦合：
// 亲外思潮越高的 cohort，忠诚的漂移天花板越低——已扎根的可核验事实溶解
// 无条件服从。不投入时亲外停在低位，忠诚天花板几乎不受影响。
func TestOpennessErodesLoyaltyCeiling(t *testing.T) {
	silo := CreateInitialSilo("测试堡", 122, nil)
	if len(silo.Cohorts) < 2 {
		t.Fatalf("expected at least 2 cohorts, got %d", len(silo.Cohorts))
	}
	open := &silo.Cohorts[0]
	closed := &silo.Cohorts[1]
	open.Ideologies[model.IdeologyProForeign] = 0.7
	closed.Ideologies[model.IdeologyProForeign] = 0.1
	open.Ideologies[model.IdeologyLoyalty] = 0.9
	closed.Ideologies[model.IdeologyLoyalty] = 0.9

	// 短窗口断言：同样的正统性下，高亲外 cohort 的忠诚向下漂移，低亲外的
	// 向上漂移。长窗口里亲外自身会向低目标回落、侵蚀随之减弱——这正是
	// "只有持续投入才守得住"的机制，不在此断言。
	startOpen := open.Ideologies[model.IdeologyLoyalty]
	for i := 0; i < 3; i++ {
		updatePopulationCohorts(silo, 1.0/12.0)
	}
	openLoyalty := open.Ideologies[model.IdeologyLoyalty]
	closedLoyalty := closed.Ideologies[model.IdeologyLoyalty]
	if openLoyalty >= startOpen {
		t.Fatalf("open cohort loyalty must drift down from %.3f, got %.3f", startOpen, openLoyalty)
	}
	if openLoyalty >= closedLoyalty {
		t.Fatalf("open cohort loyalty must erode below closed cohort: open=%.4f closed=%.4f", openLoyalty, closedLoyalty)
	}
}

// TestNudgeAllCohortIdeology locks 公共事实推动全堡思潮的落地形状：只推
// 存活 cohort，夹在 [0,1]，空堡/零增量是安全的空操作。
func TestNudgeAllCohortIdeology(t *testing.T) {
	silo := CreateInitialSilo("测试堡", 122, nil)
	before := make([]float64, len(silo.Cohorts))
	for i := range silo.Cohorts {
		before[i] = silo.Cohorts[i].Ideologies[model.IdeologyProForeign]
	}
	NudgeAllCohortIdeology(silo, model.IdeologyProForeign, 0.02)
	moved := 0
	for i := range silo.Cohorts {
		if silo.Cohorts[i].Count <= 0 {
			continue
		}
		got := silo.Cohorts[i].Ideologies[model.IdeologyProForeign]
		if got > before[i] {
			moved++
		}
		if got > 1.0 || got < 0 {
			t.Fatalf("cohort %d ideology out of unit range: %v", silo.Cohorts[i].ID, got)
		}
	}
	if moved == 0 {
		t.Fatal("nudge moved no live cohort")
	}
	NudgeAllCohortIdeology(nil, model.IdeologyProForeign, 0.02) // 空堡安全
	NudgeAllCohortIdeology(silo, model.IdeologyProForeign, 0)    // 零增量安全
}

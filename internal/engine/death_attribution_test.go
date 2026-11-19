package engine

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// TestDeathAttributionSplitsCauses 锁定 apply_population_deaths 突变的归因
// 日志：死因按 metrics_update 死亡率公式逐项拆分、各项之和等于总数，
// 且归因本身不改动人口以外的任何数值状态。
func TestDeathAttributionSplitsCauses(t *testing.T) {
	silo := CreateInitialSilo("Attr", 122, nil)
	silo.Resources = []model.Resource{
		{Type: "Energy", Amount: 0},
		{Type: "Materials", Amount: 100},
		{Type: "Supplies", Amount: 0},
	}
	for i := range silo.Floors {
		silo.Floors[i].Stability = 0.4
	}
	silo.Rebellion = 0.9

	e := NewGameEngine()
	e.PrepareTick(silo, nil)
	before := silo.TotalPopulation
	e.ApplyMutation(silo, nil, ActorRef{}, MechanicMutation{Type: "apply_population_deaths", IntValue: 124})

	if silo.TotalPopulation != before-124 {
		t.Fatalf("deaths must still be applied in full: before=%d after=%d", before, silo.TotalPopulation)
	}
	if silo.Rebellion != 0.9 {
		t.Fatalf("attribution must not modify rebellion, got %v", silo.Rebellion)
	}

	if len(e.logs) != 1 {
		t.Fatalf("expected exactly one attribution log, got %v", e.logs)
	}
	// 死亡率 = 基础 0.001 + 能源耗尽 0.05 + 补给耗尽 0.05
	//        + 结构失稳 (0.8-0.4)*0.25 = 0.10 + 叛乱冲突 (0.9-0.8)*0.2 = 0.02
	// 124 人按份额拆分，余数 1 归小数部分最大的基础死亡。
	want := "[人口] 122年1月死亡 124 人：基础死亡 1、能源耗尽 28、补给耗尽 28、结构失稳 56、叛乱冲突 11。"
	if e.logs[0] != want {
		t.Fatalf("attribution log mismatch:\n got %s\nwant %s", e.logs[0], want)
	}
}

// TestDeathAttributionUsesPreBatchState 锁定批前快照口径：同一批突变里
// silo_metric_set rebellion 先于 apply_population_deaths 应用，归因必须按
// 批前状态读取（与 JS metrics_update 读 metrics.rebellion 的口径一致），
// 不能把批内覆写后的高叛乱算进死因。
func TestDeathAttributionUsesPreBatchState(t *testing.T) {
	silo := CreateInitialSilo("PreBatch", 122, nil)
	silo.Rebellion = 0.2

	e := NewGameEngine()
	e.PrepareTick(silo, nil)
	event := CreateEvent("death_attr#1", EVENT_METRICS_UPDATE, map[string]interface{}{
		"silo": silo,
	})
	mutations := []MechanicMutation{
		{Type: "silo_metric_set", Metric: "rebellion", Value: 0.95},
		{Type: "apply_population_deaths", IntValue: 10},
	}
	e.applyMechanicMutations(event, mutations, NewEventContext())

	if silo.Rebellion != 0.95 {
		t.Fatalf("rebellion mutation must still apply, got %v", silo.Rebellion)
	}
	if len(e.logs) != 1 {
		t.Fatalf("expected exactly one attribution log, got %v", e.logs)
	}
	if strings.Contains(e.logs[0], "叛乱冲突") {
		t.Fatalf("pre-batch rebellion 0.2 must not attribute deaths to rebellion: %s", e.logs[0])
	}
	want := "[人口] 122年1月死亡 10 人：基础死亡 10。"
	if e.logs[0] != want {
		t.Fatalf("attribution log mismatch:\n got %s\nwant %s", e.logs[0], want)
	}
}

// TestDeathAttributionDeterministic 同状态双跑归因必须逐字节一致，
// 且不产生人口数字以外的状态差异。
func TestDeathAttributionDeterministic(t *testing.T) {
	run := func() (string, int) {
		silo := CreateInitialSilo("Det", 122, nil)
		silo.Resources = []model.Resource{
			{Type: "Energy", Amount: 0},
			{Type: "Materials", Amount: 30},
			{Type: "Supplies", Amount: 0},
		}
		for i := range silo.Floors {
			silo.Floors[i].Stability = 0.6
		}
		e := NewGameEngine()
		e.PrepareTick(silo, nil)
		e.ApplyMutation(silo, nil, ActorRef{}, MechanicMutation{Type: "apply_population_deaths", IntValue: 57})
		if len(e.logs) != 1 {
			t.Fatalf("expected exactly one attribution log, got %v", e.logs)
		}
		return e.logs[0], silo.TotalPopulation
	}
	firstLog, firstPop := run()
	secondLog, secondPop := run()
	if firstLog != secondLog || firstPop != secondPop {
		t.Fatalf("attribution not deterministic: %q/%d vs %q/%d", firstLog, firstPop, secondLog, secondPop)
	}
	// 份额拆分总和必须等于死亡总数。
	parts := []deathCausePart{
		{Label: "基础死亡", Rate: 0.001},
		{Label: "能源耗尽", Rate: 0.05},
		{Label: "补给耗尽", Rate: 0.05},
		{Label: "结构失稳", Rate: (0.8 - 0.6) * 0.25},
	}
	sum := 0
	for _, c := range splitDeathsByCauses(57, parts) {
		sum += c
	}
	if sum != 57 {
		t.Fatalf("split must sum to total deaths: got %d want 57", sum)
	}
}

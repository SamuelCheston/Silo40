package engine

import (
	"testing"

	"silo40/internal/model"
)

// smoke: 初始化 + 推进 10 年 + 执行通用动作 + 职业动作 + 验证胜利判定不 panic
func TestEngineSmoke(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Silo 40", 122, []string{"shadowy"})
	agent := CreateInitialAgent("Juliette", "Mechanical", []string{"shadowy"}, silo)

	if len(silo.Cohorts) == 0 {
		t.Fatalf("expected aggregated population cohorts to be generated")
	}
	if len(silo.Residents) == 0 {
		t.Fatalf("expected key residents to be generated")
	}
	if len(silo.Factions) == 0 {
		t.Fatalf("expected implicit factions to be generated")
	}
	for _, faction := range silo.Factions {
		if faction.RepresentativeResidentID == 0 {
			t.Fatalf("faction %s has no representative", faction.Name)
		}
		if faction.RepresentativeCohortID == nil {
			t.Fatalf("faction %s has no representative cohort", faction.Name)
		}
	}

	// 推进 120 个月 (10 年)
	for i := 0; i < 120; i++ {
		logs, stories := e.UpdateSiloState(silo, 1.0/12.0, agent, "TEST")
		_ = logs
		_ = stories
	}

	if silo.TotalPopulation <= 0 {
		t.Fatalf("population died too fast: %d", silo.TotalPopulation)
	}
	if silo.CurrentYear != 132 {
		t.Fatalf("expected year 132, got %d", silo.CurrentYear)
	}

	// 通用动作
	res := e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 10,
	})
	t.Logf("GATHER_INFO: %v %s", res.Executed, res.Message)

	res = e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionBuildConnection, TargetDept: "Mayor", Cost: 15,
	})
	t.Logf("BUILD_CONNECTION: %v %s", res.Executed, res.Message)

	// 职业行动
	res = e.ExecuteAgentAction(silo, agent, model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MECHANICAL_OVERHAUL", Cost: 15,
	})
	t.Logf("PROFESSION(MECHANICAL_OVERHAUL): %v %s", res.Executed, res.Message)

	if !res.Executed {
		t.Fatalf("profession action should have executed: %s", res.Message)
	}

	// 评分与叙事
	score := e.CalculateScore(silo)
	if score == nil || score.Total <= 0 {
		t.Fatalf("expected positive score, got %+v", score)
	}
	t.Logf("score: %+v", score)
}

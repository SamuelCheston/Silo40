package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestCheckVictoryConditionsFallbackSemantics 锁住 Go 兜底结局镜像的语义。
// 该镜像是 events/mechanics/core_systems.js VICTORY_CHECK mechanic 的降级
// 副本（JS 缺失/报错时生效）；修改任一侧结局规则必须同步另一侧并更新本测试。
func TestCheckVictoryConditionsFallbackSemantics(t *testing.T) {
	e := &GameEngine{}

	baseSilo := func() *model.Silo {
		return &model.Silo{
			TotalPopulation: 1000,
			Professions: []model.Profession{
				{Name: "IT", KnownFragments: []string{"Mayor_1", "Mayor_2"}},
			},
		}
	}
	agent := &model.Agent{}

	t.Run("safeguard risk triggers DEATH loss first", func(t *testing.T) {
		silo := baseSilo()
		silo.SafeguardRisk = 1.0
		silo.Silo1Destroyed = true // 优先级低于 safeguard 的判定必须被压在后面
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus == nil || silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "DEATH" {
			t.Fatalf("expected DEATH loss, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("all departments holding 5+ fragments wins INFORMATION", func(t *testing.T) {
		silo := baseSilo()
		frags := []string{"Mayor_1", "Mayor_2", "IT_1", "IT_2", "Police_1"}
		silo.Professions = []model.Profession{
			{Name: "IT", KnownFragments: frags},
			{Name: "Mayor", KnownFragments: frags},
		}
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus == nil || !silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "INFORMATION" {
			t.Fatalf("expected INFORMATION win, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("silo1 destroyed wins TIME", func(t *testing.T) {
		silo := baseSilo()
		silo.Silo1Destroyed = true
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus == nil || !silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "TIME" {
			t.Fatalf("expected TIME win, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("agent suspicion at cap loses AGENT_COMPROMISED before rebellion", func(t *testing.T) {
		silo := baseSilo()
		silo.Rebellion = 1.0 // 怀疑度优先于叛乱判定
		compromised := &model.Agent{SuspicionLevel: 1.0}
		e.checkVictoryConditions(silo, compromised)
		if silo.VictoryStatus == nil || silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "AGENT_COMPROMISED" {
			t.Fatalf("expected AGENT_COMPROMISED loss, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("rebellion at cap wins REBELLION", func(t *testing.T) {
		silo := baseSilo()
		silo.Rebellion = 1.0
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus == nil || !silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "REBELLION" {
			t.Fatalf("expected REBELLION win, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("extinction loses DEATH", func(t *testing.T) {
		silo := baseSilo()
		silo.TotalPopulation = 0
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus == nil || silo.VictoryStatus.IsWon || silo.VictoryStatus.Type != "DEATH" {
			t.Fatalf("expected DEATH loss, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("no ending leaves status nil", func(t *testing.T) {
		silo := baseSilo()
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus != nil {
			t.Fatalf("expected nil victory status, got %+v", silo.VictoryStatus)
		}
	})

	t.Run("already-won status is never overwritten", func(t *testing.T) {
		silo := baseSilo()
		silo.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "INFORMATION"}
		silo.TotalPopulation = 0 // 若被覆盖会变成 DEATH
		e.checkVictoryConditions(silo, agent)
		if silo.VictoryStatus.Type != "INFORMATION" || !silo.VictoryStatus.IsWon {
			t.Fatalf("won status must be preserved, got %+v", silo.VictoryStatus)
		}
	})
}

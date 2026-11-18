package service

import (
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// C8 端到端：玩家经 ExecuteAgentAction 发起 FOUND_FACTION (JS 预检 →
// faction_found_actor) → 三通道感知写入 faction:<id>:founded。创始人 B=1，
// 各部门持有感知信念；渗透后仍不成熟、共证资格门不受污染。
func TestFoundFactionSeedsAwarenessEndToEnd(t *testing.T) {
	svc := counterTestService(t)
	svc.agent.Ambition = 0.8
	svc.agent.Influence = 0.6
	svc.agent.ActionPoints = 60
	it := counterProfession(t, svc, "IT")
	it.Ideologies[model.IdeologyProForeign] = 0.85

	result := svc.engine.ExecuteAgentAction(svc.silo, svc.agent, model.AgentAction{
		Type: model.ActionFoundFaction, Cost: 20,
	})
	if !result.Executed {
		t.Fatalf("FOUND_FACTION not executed: %s", result.Message)
	}
	var founded *model.Faction
	for i := range svc.silo.Factions {
		if strings.HasPrefix(svc.silo.Factions[i].Signature, "founded:") {
			founded = &svc.silo.Factions[i]
		}
	}
	if founded == nil {
		t.Fatalf("no founded faction: %+v", svc.silo.Factions)
	}
	factID := engine.FactionFoundedFactID(founded.ID)
	if b := engine.BeliefIn(&svc.agent.Beliefs, factID); b == nil || b.B != 1 {
		t.Fatalf("founder belief = %+v, want B=1", b)
	}
	for i := range svc.silo.Professions {
		prof := &svc.silo.Professions[i]
		if b := engine.BeliefIn(&prof.Beliefs, factID); b == nil || b.B < 0 || b.B > 1 {
			t.Fatalf("%s awareness belief missing or out of range: %+v", prof.Name, b)
		}
	}

	engine.ApplyPermeation(svc.silo, engine.InformationConfig{PermeationMonths: 1, QualRumor: 0})
	for i := range svc.silo.Professions {
		prof := &svc.silo.Professions[i]
		if b := engine.BeliefIn(&prof.Beliefs, factID); b.Matured {
			t.Fatalf("%s faction awareness matured", prof.Name)
		}
		if got := engine.MaturedFactCount(prof); got != 0 {
			t.Errorf("%s matured fact count = %d, want 0 (faction fact excluded)", prof.Name, got)
		}
	}
}
package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 20 分册第 5 批（T1-3/T2-4/T3-1b/T3-2）的固定输入复现测试。
// 数值为设计起点值：B 0.45 / 12 月、prep_exposure 0.4 / -0.15、B 0.60 /
// 关系 -0.10 / 3 月 / 怀疑度 +0.03、身份线灰置 1 月。

// 20 分册 §3.2：玩家一次流通会在 IT/Judicial 留下 leak: 观察；泄漏事实
// 只淡忘、永不成熟，也不计入部门成熟事实数 (23 共证资格门不受污染)。
func TestLeakObservationFromShareStaysImmature(t *testing.T) {
	svc := counterTestService(t)
	svc.agent.Beliefs = []model.FactBelief{{FactID: "Mayor_archive", B: 0.9}}
	svc.agent.ActionPoints = 50

	result := svc.engine.ExecuteAgentAction(svc.silo, svc.agent, model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Supply", FragmentIds: []string{"Mayor_archive"}, Cost: 2,
	})
	if !result.Executed {
		t.Fatalf("share not executed: %s", result.Message)
	}
	for _, name := range []string{"IT", "Judicial"} {
		prof := counterProfession(t, svc, name)
		leakB := 0.0
		for _, belief := range prof.Beliefs {
			if belief.FactID == engine.LeakFactPrefix+"Mayor_archive" {
				leakB = belief.B
			}
		}
		if leakB <= 0 {
			t.Fatalf("%s did not observe the leak, beliefs=%+v", name, prof.Beliefs)
		}
	}

	engine.ApplyPermeation(svc.silo, engine.InformationConfig{PermeationMonths: 1, QualRumor: 0})
	for _, name := range []string{"IT", "Judicial"} {
		prof := counterProfession(t, svc, name)
		for _, belief := range prof.Beliefs {
			if belief.FactID == engine.LeakFactPrefix+"Mayor_archive" && belief.Matured {
				t.Fatalf("%s leaked observation matured", name)
			}
		}
		if got := engine.MaturedFactCount(prof); got != 0 {
			t.Errorf("%s matured fact count = %d, want 0 (leak excluded)", name, got)
		}
	}
}

func counterTestService(t *testing.T) *GameService {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "countermeasures.db"))
	if _, err := svc.CreateGame(twoSiloRequest(707)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

func counterProfession(t *testing.T, svc *GameService, name string) *model.Profession {
	t.Helper()
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].Name == name {
			return &svc.silo.Professions[i]
		}
	}
	t.Fatalf("profession %q not found", name)
	return nil
}

func counterConnection(t *testing.T, svc *GameService, professionID uint) float64 {
	t.Helper()
	for _, connection := range svc.agent.Connections {
		if connection.ProfessionID == professionID {
			return connection.Value
		}
	}
	return 0
}

func setConnection(t *testing.T, svc *GameService, professionID uint, value float64) {
	t.Helper()
	for i := range svc.agent.Connections {
		if svc.agent.Connections[i].ProfessionID == professionID {
			svc.agent.Connections[i].Value = value
			return
		}
	}
	svc.agent.Connections = append(svc.agent.Connections, model.Connection{ProfessionID: professionID, Value: value})
}

func setBelief(svc *GameService, profession *model.Profession, factID string, b float64) {
	for i := range profession.Beliefs {
		if profession.Beliefs[i].FactID == factID {
			profession.Beliefs[i].B = b
			return
		}
	}
	profession.Beliefs = append(profession.Beliefs, model.FactBelief{FactID: factID, B: b})
}

// T1-3：部门对泄漏事实 B ≥ 0.45 → 关系冷却 12 月，冷却期内 Build 关系
// 增长减半；回执措辞变化由同一账本驱动 (JS 读 relation_cooldown)。
func TestT13RelationCoolingHalvesBuild(t *testing.T) {
	svc := counterTestService(t)
	mayor := counterProfession(t, svc, "IT")
	setBelief(svc, mayor, "leak:Mayor_archive", 0.45)

	// 基线：无冷却时的单次建联增量。
	setConnection(t, svc, mayor.ID, 0.2)
	base := svc.engine.ExecuteAgentAction(svc.silo, svc.agent, model.AgentAction{
		Type: model.ActionBuildConnection, TargetDept: "IT", Cost: 4,
	})
	if !base.Executed {
		t.Fatalf("baseline build not executed: %s", base.Message)
	}
	full := counterConnection(t, svc, mayor.ID) - 0.2
	if full <= 0 {
		t.Fatalf("baseline build did not raise connection: %+v", base)
	}

	// 触发 T1-3。
	svc.runCountermeasureMonth(svc.silo)
	if got := svc.silo.Countermeasure.RelationCooldown["IT"]; got != 12 {
		t.Fatalf("relation cooldown = %d months, want 12", got)
	}

	setConnection(t, svc, mayor.ID, 0.2)
	cooled := svc.engine.ExecuteAgentAction(svc.silo, svc.agent, model.AgentAction{
		Type: model.ActionBuildConnection, TargetDept: "IT", Cost: 4,
	})
	if !cooled.Executed {
		t.Fatalf("cooled build not executed: %s", cooled.Message)
	}
	half := counterConnection(t, svc, mayor.ID) - 0.2
	if diff := half - full/2; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cooled build delta = %.6f, want half of %.6f", half, full)
	}
	if !strings.Contains(cooled.Message, "比上次谨慎") {
		t.Errorf("cooled build message = %q, want cautious wording", cooled.Message)
	}

	// 冷却期内重复越线不刷新、不叠加。
	svc.runCountermeasureMonth(svc.silo)
	if got := svc.silo.Countermeasure.RelationCooldown["IT"]; got != 11 {
		t.Errorf("relation cooldown after next month = %d, want 11 (no refresh)", got)
	}
}

// T2-4：prep_exposure ≥ 0.4 → Guarantee −0.15，且同一来源只侵蚀一次。
func TestT24GuaranteeErosionOnce(t *testing.T) {
	svc := counterTestService(t)
	svc.silo.TopStance.Guarantee = 0.5
	svc.silo.EscapePrep.PrepExposure = 0.39

	if logs, stories := svc.runCountermeasureMonth(svc.silo); len(logs) != 0 || len(stories) != 0 {
		t.Fatalf("below threshold produced signals: %v %v", logs, stories)
	}
	if svc.silo.TopStance.Guarantee != 0.5 {
		t.Fatalf("guarantee changed below threshold: %.3f", svc.silo.TopStance.Guarantee)
	}

	svc.silo.EscapePrep.PrepExposure = 0.4
	logs, stories := svc.runCountermeasureMonth(svc.silo)
	if got := svc.silo.TopStance.Guarantee; got < 0.3499 || got > 0.3501 {
		t.Fatalf("guarantee = %.4f, want 0.35 after T2-4", got)
	}
	if !svc.silo.Countermeasure.GuaranteeEroded {
		t.Fatal("T2-4 flag not set")
	}
	if len(logs) != 1 || len(stories) != 1 {
		t.Fatalf("T2-4 signals = %d logs / %d stories, want 1/1", len(logs), len(stories))
	}
	if stories[0].Title != "中间人的措辞" {
		t.Errorf("T2-4 story title = %q", stories[0].Title)
	}

	// 同源去重：暴露继续上升也不再次侵蚀。
	svc.silo.EscapePrep.PrepExposure = 0.8
	if logs, stories := svc.runCountermeasureMonth(svc.silo); len(logs) != 0 || len(stories) != 0 {
		t.Fatalf("T2-4 re-fired: %v %v", logs, stories)
	}
	if got := svc.silo.TopStance.Guarantee; got < 0.3499 || got > 0.3501 {
		t.Errorf("guarantee after re-check = %.4f, want 0.35", got)
	}
}


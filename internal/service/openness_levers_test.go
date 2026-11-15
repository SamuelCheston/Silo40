package service

import (
	"math"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// professionProForeign 按人口加权统计某部门的亲外思潮 (cohort 层是真源)。
func professionProForeign(silo *model.Silo, professionName string) float64 {
	profID := uint(0)
	for _, prof := range silo.Professions {
		if prof.Name == professionName {
			profID = prof.ID
			break
		}
	}
	if profID == 0 {
		return -1
	}
	weight, sum := 0.0, 0.0
	for _, cohort := range silo.Cohorts {
		if cohort.ProfessionID != profID || cohort.Count <= 0 {
			continue
		}
		sum += cohort.Ideologies[model.IdeologyProForeign] * float64(cohort.Count)
		weight += float64(cohort.Count)
	}
	if weight == 0 {
		return -1
	}
	return sum / weight
}

// populationProForeign 全堡人口加权亲外——与覆灭前夕快照同一口径。
func populationProForeign(silo *model.Silo) float64 {
	weight, sum := 0.0, 0.0
	for _, cohort := range silo.Cohorts {
		if cohort.Count <= 0 {
			continue
		}
		sum += cohort.Ideologies[model.IdeologyProForeign] * float64(cohort.Count)
		weight += float64(cohort.Count)
	}
	if weight == 0 {
		return 0
	}
	return sum / weight
}

// TestShareAbsorptionRaisesReceiverOpenness locks 开放痕迹·分享侧：被吸收的
// 事实抬升接收部门 cohort 的亲外思潮，其他部门不受影响——开放与《共证》
// 传播是同一条路。
func TestShareAbsorptionRaisesReceiverOpenness(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "openness-share.db"))
	req := threeSiloRequest(2403)
	req.Profession = "IT"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	receiverBefore := professionProForeign(svc.silo, "Judicial")
	bystanderBefore := professionProForeign(svc.silo, "Supply")
	if receiverBefore < 0 || bystanderBefore < 0 {
		t.Fatalf("failed to read cohort ideologies: receiver=%v bystander=%v", receiverBefore, bystanderBefore)
	}

	// 先开线：分享门控要求 first_signal 阶段 (一次 Gather 即达)。
	if outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Judicial"}); err != nil || outcome == nil || !outcome.Result.Executed {
		t.Fatalf("opening gather failed: %+v (%v)", outcome, err)
	}
	if len(svc.agent.KnownFragments) == 0 {
		t.Fatal("no fragment available to share")
	}

	outcome, err := svc.ExecuteAction(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Judicial",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	if err != nil || outcome == nil || !outcome.Result.Executed {
		t.Fatalf("share failed: %+v (%v)", outcome, err)
	}

	receiverAfter := professionProForeign(svc.silo, "Judicial")
	bystanderAfter := professionProForeign(svc.silo, "Supply")
	if receiverAfter <= receiverBefore {
		t.Fatalf("receiver openness must rise after absorption: before=%.4f after=%.4f", receiverBefore, receiverAfter)
	}
	// 旁观部门也可能被同月触发的公共证据登记全堡推动；但接收部门必须
	// 严格涨得更多 (定向吸收痕迹 + 全堡推动)。
	if receiverAfter-receiverBefore <= bystanderAfter-bystanderBefore {
		t.Fatalf("receiver must gain strictly more than bystanders: receiver Δ=%.4f bystander Δ=%.4f",
			receiverAfter-receiverBefore, bystanderAfter-bystanderBefore)
	}
}

// TestPublicEvidenceRaisesSiloOpenness locks 开放痕迹·公开侧：新事实成为公共
// 真相时全堡亲外被推动一次；重复登记同一事实不再推动。
func TestPublicEvidenceRaisesSiloOpenness(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "openness-public.db"))
	req := threeSiloRequest(2404)
	req.Profession = "IT"
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	before := populationProForeign(svc.silo)

	svc.applyPublicEvidenceWrites(svc.silo.ID, []model.PublicEvidenceRecord{
		{FactID: "openness_test_fact", Provenance: 0.6, Coverage: 0.5},
	})
	afterFirst := populationProForeign(svc.silo)
	if afterFirst < before+publicFactOpennessNudge-1e-9 {
		t.Fatalf("public fact must nudge population openness: before=%.4f after=%.4f", before, afterFirst)
	}

	svc.applyPublicEvidenceWrites(svc.silo.ID, []model.PublicEvidenceRecord{
		{FactID: "openness_test_fact", Provenance: 0.7, Coverage: 0.6},
	})
	afterRepeat := populationProForeign(svc.silo)
	if math.Abs(afterRepeat-afterFirst) > 1e-9 {
		t.Fatalf("re-registering the same fact must not nudge again: %.4f -> %.4f", afterFirst, afterRepeat)
	}
}

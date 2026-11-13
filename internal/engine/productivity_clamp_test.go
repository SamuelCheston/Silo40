package engine

import (
	"testing"
)

// Productivity is a [0,1] ratio (the Go action fallback always clamped it).
// The mutation/content effect paths must clamp too — the mechanic mutation
// path used to grow it without bound.
func TestProductivityClampedOnMutationPath(t *testing.T) {
	e := NewGameEngine()
	silo := CreateInitialSilo("Clamp Silo", 122, nil)
	dept := findDept(silo, "Mines")

	grow := MechanicMutation{Type: "profession_metric_delta", Profession: "Mines", Field: "productivity", Value: 0.5}
	for i := 0; i < 3; i++ {
		e.ApplyMutation(silo, nil, ActorRef{}, grow)
	}
	if dept.Productivity != 1.0 {
		t.Errorf("expected productivity clamped to 1.0, got %v", dept.Productivity)
	}

	e.ApplyMutation(silo, nil, ActorRef{}, MechanicMutation{Type: "profession_metric_delta", Profession: "Mines", Field: "productivity", Value: -1.5})
	if dept.Productivity != 0 {
		t.Errorf("expected productivity clamped to 0, got %v", dept.Productivity)
	}

	e.ApplyMutation(silo, nil, ActorRef{}, MechanicMutation{Type: "profession_metric_set", Profession: "Mines", Field: "productivity", Value: 3.0})
	if dept.Productivity != 1.0 {
		t.Errorf("expected productivity set clamped to 1.0, got %v", dept.Productivity)
	}
}

// The content-effect applier shares the same semantics.
func TestProductivityClampedOnContentEffectPath(t *testing.T) {
	silo := CreateInitialSilo("Clamp Silo", 122, nil)
	dept := findDept(silo, "Mines")
	dept.Productivity = 0.9

	applyProfessionMetricDelta(dept, "productivity", 0.5)
	if dept.Productivity != 1.0 {
		t.Errorf("expected content effect productivity clamped to 1.0, got %v", dept.Productivity)
	}
	applyProfessionMetricDelta(dept, "productivity", -1.5)
	if dept.Productivity != 0 {
		t.Errorf("expected content effect productivity clamped to 0, got %v", dept.Productivity)
	}
}

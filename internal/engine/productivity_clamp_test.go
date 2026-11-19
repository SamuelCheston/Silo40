package engine

import (
	"math"
	"strings"
	"testing"

	"silo40/internal/model"
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

	applyProfessionMetricDelta(silo, dept, "productivity", 0.5)
	if dept.Productivity != 1.0 {
		t.Errorf("expected content effect productivity clamped to 1.0, got %v", dept.Productivity)
	}
	applyProfessionMetricDelta(silo, dept, "productivity", -1.5)
	if dept.Productivity != 0 {
		t.Errorf("expected content effect productivity clamped to 0, got %v", dept.Productivity)
	}
}

func TestProductivityScalesResourceProduction(t *testing.T) {
	for _, tc := range []struct {
		department, resource string
	}{
		{"Mechanical", "Energy"},
		{"Mines", "Materials"},
		{"Agricultural", "Supplies"},
		{"Supply", "Supplies"},
	} {
		t.Run(tc.department, func(t *testing.T) {
			net := func(productivity float64) (float64, int) {
				silo := CreateInitialSilo("Production", 122, nil)
				department := findDept(silo, tc.department)
				department.Productivity = productivity
				for i := range silo.Resources {
					silo.Resources[i].Amount = 100_000 // Inputs do not limit conversion.
				}
				runResourceUpdateOnce(t, silo)
				return resourceNet(silo, tc.resource), department.Population
			}
			full, population := net(1)
			for _, productivity := range []float64{0.5, 0} {
				got, _ := net(productivity)
				want := full - (1-productivity)*4.5*float64(population)
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("productivity %v: %s net = %v, want %v", productivity, tc.resource, got, want)
				}
			}
		})
	}
}

func TestProductivityReceiptReflectsDirectOrder(t *testing.T) {
	silo := CreateInitialSilo("Production", 122, nil)
	agent := CreateInitialAgent("Mayor", "Mayor", nil, silo)
	agent.ActionPoints = 100
	engine := NewGameEngine()
	order := model.AgentAction{
		Type: model.ActionProfession, ProfessionAction: "MAYOR_DIRECT_ORDER", TargetDept: "Mechanical", Cost: 20,
	}
	dept := findDept(silo, "Mechanical")
	for _, tc := range []struct {
		before  float64
		message string
	}{
		{1, "Production capacity factor is already at its maximum"},
		{0.95, "Production capacity factor rose to 100.0%"},
	} {
		dept.Productivity = tc.before
		result := engine.ExecuteAgentAction(silo, agent, order)
		if !result.Executed || dept.Productivity != 1 || !strings.Contains(result.Message, tc.message) {
			t.Errorf("productivity %v -> %v, result %+v, want %q", tc.before, dept.Productivity, result, tc.message)
		}
	}
}

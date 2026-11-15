package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestEscapeOrgSuspicionAccumulatesClamps 累积夹 [0,1] 且首次累积初始化层。
func TestEscapeOrgSuspicionAccumulatesClamps(t *testing.T) {
	silo := &model.Silo{}
	AddEscapeOrgSuspicion(silo, 0.4)
	if !silo.EscapeResponse.Initialized {
		t.Fatal("first accumulation must initialize the response layer")
	}
	if silo.EscapeResponse.OrgSuspicion != 0.4 {
		t.Fatalf("suspicion = %v, want 0.4", silo.EscapeResponse.OrgSuspicion)
	}
	AddEscapeOrgSuspicion(silo, 0.9)
	if silo.EscapeResponse.OrgSuspicion != 1.0 {
		t.Fatalf("suspicion = %v, want clamped 1.0", silo.EscapeResponse.OrgSuspicion)
	}
	// 非正增量是无效写入 (验证层应拦截)，这里保持状态不变。
	AddEscapeOrgSuspicion(silo, -0.5)
	AddEscapeOrgSuspicion(silo, 0)
	if silo.EscapeResponse.OrgSuspicion != 1.0 {
		t.Fatalf("suspicion = %v, want unchanged 1.0", silo.EscapeResponse.OrgSuspicion)
	}
}

// TestEscapeOrgSuspicionDecay 月度衰减夹到 0；未初始化的堡逐位不变。
func TestEscapeOrgSuspicionDecay(t *testing.T) {
	silo := &model.Silo{}
	silo.EscapeResponse.Initialized = true
	silo.EscapeResponse.OrgSuspicion = 0.005
	DecayEscapeOrgSuspicion(silo)
	if silo.EscapeResponse.OrgSuspicion != 0 {
		t.Fatalf("suspicion = %v, want floored 0", silo.EscapeResponse.OrgSuspicion)
	}

	fresh := &model.Silo{}
	DecayEscapeOrgSuspicion(fresh)
	if fresh.EscapeResponse.Initialized || fresh.EscapeResponse.OrgSuspicion != 0 {
		t.Fatalf("uninitialized silo must stay untouched: %+v", fresh.EscapeResponse)
	}
}

// TestEscapeOrgSuspicionEffectValidation 效果只接受正增量 (只增不减，
// 衰减走月度回合)。
func TestEscapeOrgSuspicionEffectValidation(t *testing.T) {
	if err := validateEffect(ContentEffect{Type: "escape_org_suspicion_delta", Value: 0.05}); err != nil {
		t.Fatalf("positive delta must validate: %v", err)
	}
	if err := validateEffect(ContentEffect{Type: "escape_org_suspicion_delta", Value: -0.05}); err == nil {
		t.Fatal("negative delta must fail validation")
	}
	if err := validateEffect(ContentEffect{Type: "escape_org_suspicion_delta"}); err == nil {
		t.Fatal("zero delta must fail validation")
	}
}

// TestEscapeOrgSuspicionEffectApplies 经施加管线写入响应层。
func TestEscapeOrgSuspicionEffectApplies(t *testing.T) {
	silo := &model.Silo{}
	sc := &ScopeContext{Self: silo, Errors: &[]string{}}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "escape_org_suspicion_delta", Value: 0.06},
		{Type: "escape_org_suspicion_delta", Value: 0.03},
	}, sc)
	if len(*sc.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", *sc.Errors)
	}
	if silo.EscapeResponse.OrgSuspicion != 0.09 {
		t.Fatalf("suspicion = %v, want 0.09", silo.EscapeResponse.OrgSuspicion)
	}
}

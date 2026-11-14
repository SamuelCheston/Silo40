package engine

import (
	"testing"

	"silo40/internal/model"
)

func prepTestSilo() *model.Silo {
	return &model.Silo{
		Resources: []model.Resource{
			{Type: "Supplies", Amount: 1000},
			{Type: "Materials", Amount: 500},
		},
	}
}

// 白名单：未知字段在加载期即失败 (显式失败原则)。
func TestPreparationDeltaWhitelist(t *testing.T) {
	if err := validateEffect(ContentEffect{Type: "preparation_delta", Metric: "drill_level", Value: 0.1}); err != nil {
		t.Fatalf("whitelisted field should validate: %v", err)
	}
	if err := validateEffect(ContentEffect{Type: "preparation_delta", Metric: "success_rate", Value: 0.1}); err == nil {
		t.Fatal("unknown ledger field must fail validation")
	}
	if err := validateEffect(ContentEffect{Type: "preparation_delta", Metric: "plan_role", Profession: "mechanical", Value: 1}); err != nil {
		t.Fatalf("plan role should validate: %v", err)
	}
	if err := validateEffect(ContentEffect{Type: "preparation_delta", Metric: "plan_role", Profession: "shaman", Value: 1}); err == nil {
		t.Fatal("unknown plan role must fail validation")
	}
}

// 整数量与程度量写入：夹取语义。
func TestPreparationDeltaClamps(t *testing.T) {
	silo := prepTestSilo()
	sc := &ScopeContext{Self: silo, Errors: &[]string{}}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_delta", Metric: "suits_functional", Value: 3},
		{Type: "preparation_delta", Metric: "suits_functional", Value: -10}, // 审计没收不可为负
		{Type: "preparation_delta", Metric: "drill_level", Value: 1.7},
		{Type: "preparation_delta", Metric: "prep_exposure", Value: -2},
	}, sc)
	if len(*sc.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", *sc.Errors)
	}
	prep := silo.EscapePrep
	if prep.SuitsFunctional != 0 {
		t.Fatalf("suits = %d, want clamped 0", prep.SuitsFunctional)
	}
	if prep.DrillLevel != 1 {
		t.Fatalf("drill = %v, want clamped 1", prep.DrillLevel)
	}
	if prep.PrepExposure != 0 {
		t.Fatalf("exposure = %v, want clamped 0", prep.PrepExposure)
	}
	if !prep.Initialized {
		t.Fatal("ledger should be initialized after first write")
	}
}

// 预案完备度 = 功能位覆盖率：去重、升序、可撤销。
func TestPreparationPlanRoles(t *testing.T) {
	silo := prepTestSilo()
	sc := &ScopeContext{Self: silo, Errors: &[]string{}}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_delta", Metric: "plan_role", Profession: "medical", Value: 1},
		{Type: "preparation_delta", Metric: "plan_role", Profession: "mechanical", Value: 1},
		{Type: "preparation_delta", Metric: "plan_role", Profession: "mechanical", Value: 1}, // 重复
	}, sc)
	prep := silo.EscapePrep
	if len(prep.PlanRoles) != 2 || prep.PlanRoles[0] != "mechanical" || prep.PlanRoles[1] != "medical" {
		t.Fatalf("plan roles = %v, want deduped sorted [mechanical medical]", prep.PlanRoles)
	}
	if want := 2.0 / model.PlanRoleCount; prep.PlanCompleteness != want {
		t.Fatalf("completeness = %v, want %v", prep.PlanCompleteness, want)
	}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_delta", Metric: "plan_role", Profession: "mechanical", Value: -1},
	}, sc)
	if len(silo.EscapePrep.PlanRoles) != 1 || silo.EscapePrep.PlanCompleteness != 1.0/model.PlanRoleCount {
		t.Fatalf("role removal failed: %v %v", silo.EscapePrep.PlanRoles, silo.EscapePrep.PlanCompleteness)
	}
}

// 接应档位只前进；非法移动显式失败。
func TestPreparationReceptionStage(t *testing.T) {
	silo := prepTestSilo()
	errs := &[]string{}
	sc := &ScopeContext{Self: silo, Errors: errs}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_delta", Metric: "reception_stage", Flag: "confirmed"},
	}, sc)
	if silo.EscapePrep.ReceptionStage != model.ReceptionConfirmed {
		t.Fatalf("stage = %q, want confirmed", silo.EscapePrep.ReceptionStage)
	}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_delta", Metric: "reception_stage", Flag: "surveyed"}, // 回退
	}, sc)
	if len(*errs) == 0 {
		t.Fatal("stage regression must fail explicitly")
	}
	if silo.EscapePrep.ReceptionStage != model.ReceptionConfirmed {
		t.Fatalf("stage changed on failed write: %q", silo.EscapePrep.ReceptionStage)
	}
}

// 划拨即扣减：堡内资源立即减少；余额不足显式失败、不部分划拨。
func TestPreparationTransfer(t *testing.T) {
	silo := prepTestSilo()
	errs := &[]string{}
	sc := &ScopeContext{Self: silo, Errors: errs}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_transfer", Metric: "Supplies", Value: 300},
	}, sc)
	if len(*errs) != 0 {
		t.Fatalf("unexpected errors: %v", *errs)
	}
	if got := silo.Resources[0].Amount; got != 700 {
		t.Fatalf("supplies after transfer = %v, want 700", got)
	}
	if silo.EscapePrep.SuppliesPacked != 300 {
		t.Fatalf("packed = %v, want 300", silo.EscapePrep.SuppliesPacked)
	}

	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "preparation_transfer", Metric: "Supplies", Value: 5000}, // 超出余额
	}, sc)
	if len(*errs) == 0 {
		t.Fatal("insufficient balance must fail explicitly")
	}
	if got := silo.Resources[0].Amount; got != 700 {
		t.Fatalf("failed transfer must not deduct, got %v", got)
	}
	if silo.EscapePrep.SuppliesPacked != 300 {
		t.Fatalf("failed transfer must not pack, got %v", silo.EscapePrep.SuppliesPacked)
	}
}

// 信道登记：种类白名单、目标堡校验、建立即记暴露成本。
func TestSpilloverChannelAdd(t *testing.T) {
	silo := prepTestSilo()
	silo.CurrentYear, silo.CurrentMonth = 124, 5
	errs := &[]string{}
	sc := &ScopeContext{Self: silo, Errors: errs}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "spillover_channel_add", Flag: "cross_signal", Count: 2, Value: 0.5},
	}, sc)
	if len(*errs) != 0 {
		t.Fatalf("unexpected errors: %v", *errs)
	}
	if len(silo.SpilloverPrep.Channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(silo.SpilloverPrep.Channels))
	}
	ch := silo.SpilloverPrep.Channels[0]
	if ch.Kind != model.SpilloverChannelCrossSignal || ch.TargetSilo != 2 || ch.Reliability != 0.5 || ch.ExposureCost <= 0 {
		t.Fatalf("channel fields wrong: %+v", ch)
	}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "spillover_channel_add", Flag: "pigeon", Count: 2, Value: 0.5},
	}, sc)
	if len(*errs) == 0 {
		t.Fatal("unknown channel kind must fail")
	}
}

// preparation_gte 触发器：加载期字段白名单 + 运行期读数。
func TestPreparationGteTrigger(t *testing.T) {
	if err := validateTrigger(ContentTrigger{Type: "preparation_gte", Metric: "suits_functional", Value: 2}); err != nil {
		t.Fatalf("valid trigger rejected: %v", err)
	}
	if err := validateTrigger(ContentTrigger{Type: "preparation_gte", Metric: "vibes", Value: 2}); err == nil {
		t.Fatal("unknown ledger metric must fail trigger validation")
	}
	silo := prepTestSilo()
	silo.EscapePrep.SuitsFunctional = 3
	ctx := ContentEvaluationContext{Silo: silo}
	if !evaluateContentTrigger(ContentTrigger{Type: "preparation_gte", Metric: "suits_functional", Value: 3}, ctx, false) {
		t.Fatal("3 suits should satisfy >= 3")
	}
	if evaluateContentTrigger(ContentTrigger{Type: "preparation_gte", Metric: "suits_functional", Value: 4}, ctx, false) {
		t.Fatal("3 suits must not satisfy >= 4")
	}
}

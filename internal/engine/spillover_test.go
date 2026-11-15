package engine

import "testing"

// TestSpilloverSendEffectValidation 锁 spillover_send 效果的封闭集校验：
// 三种信道种类合法，未知种类显式失败 (16 §3.5 Layer 3)。
func TestSpilloverSendEffectValidation(t *testing.T) {
	for _, kind := range []string{"it_line", "cross_signal", "physical_handoff"} {
		if err := validateEffect(ContentEffect{Type: "spillover_send", Flag: kind}); err != nil {
			t.Fatalf("spillover_send %s must validate: %v", kind, err)
		}
	}
	if err := validateEffect(ContentEffect{Type: "spillover_send", Flag: "pigeon"}); err == nil {
		t.Fatal("unknown channel kind must fail validation")
	}
	if err := validateEffect(ContentEffect{Type: "spillover_send"}); err == nil {
		t.Fatal("missing channel kind must fail validation")
	}
}

// TestSpilloverSendEffectCollected 锁效果收集：spillover_send 进入
// ScopeContext.SpilloverSendKinds，按施加顺序排列，不直接改动堡状态。
func TestSpilloverSendEffectCollected(t *testing.T) {
	kinds := []string{}
	sc := &ScopeContext{SpilloverSendKinds: &kinds}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "spillover_send", Flag: "it_line"},
		{Type: "spillover_send", Flag: "physical_handoff"},
	}, sc)
	if len(kinds) != 2 || kinds[0] != "it_line" || kinds[1] != "physical_handoff" {
		t.Fatalf("spillover_send collection mismatch: %+v", kinds)
	}
}

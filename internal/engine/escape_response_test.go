package engine

import (
	"math/rand"
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

// escapeResponseTestSilo 一个带部门与 IT 权威状态的决策测试堡。
func escapeResponseTestSilo() *model.Silo {
	silo := &model.Silo{CurrentYear: 125, CurrentMonth: 6}
	silo.Professions = []model.Profession{
		{Name: "Judicial"}, {Name: "IT"}, {Name: "Supply"},
	}
	silo.ItAuthority = model.ItAuthorityState{Initialized: true, ROp: 60}
	silo.Authority.W = 50
	silo.EscapeResponse.Initialized = true
	silo.EscapeResponse.OrgSuspicion = 0.5
	return silo
}

// TestEscapeResponseWeightsFloorAndNormalize 自由裁量地板在极端状态下
// 不被压没；归一化权重之和为 1。
func TestEscapeResponseWeightsFloorAndNormalize(t *testing.T) {
	cases := []func(*model.Silo){
		func(s *model.Silo) {}, // 基线
		func(s *model.Silo) { s.Authority.W = 100; s.SafeguardRisk = 1.0; s.TopStance.IT = model.TopStanceBlock },
		func(s *model.Silo) {
			s.Authority.W = 0
			s.ItAuthority.ROp = 0
			s.TopStance.IT = model.TopStanceCooperate
			s.TopStance.Guarantee = 1
			s.EscapeResponse.OrgSuspicion = 1
		},
		func(s *model.Silo) { s.EscapeResponse.OrgSuspicion = 0; s.SafeguardRisk = 0 },
	}
	for i, mutate := range cases {
		silo := escapeResponseTestSilo()
		mutate(silo)
		w := EscapeResponseDecisionWeights(silo)
		if w.Suppress < escapeResponseDiscretionFloor || w.Conceal < escapeResponseDiscretionFloor || w.Ignore < escapeResponseDiscretionFloor {
			t.Fatalf("case %d: discretion floor violated: %+v", i, w)
		}
		norm := NormalizeEscapeResponseWeights(w)
		sum := norm[model.EscapeResponseSuppress] + norm[model.EscapeResponseConceal] + norm[model.EscapeResponseIgnore]
		if sum < 0.999 || sum > 1.001 {
			t.Fatalf("case %d: normalized weights sum = %v", i, sum)
		}
	}
}

// TestEscapeResponseDecisionNeverTouchesHoardedLedger 核心不变式：任何
// 决策档位都不扣减囤积账本与外溢信道 (审计没收永不回归)，且同种子
// 决策可复现。
func TestEscapeResponseDecisionNeverTouchesHoardedLedger(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(1); seed <= 40; seed++ {
		SetRNG(rand.New(rand.NewSource(seed)))
		silo := escapeResponseTestSilo()
		silo.EscapePrep = model.EscapePreparation{
			Initialized: true, SuitsFunctional: 12, SuppliesPacked: 300, MaterialsPacked: 80,
			DrillLevel: 0.4, PrepExposure: 0.5,
		}
		silo.SpilloverPrep.Channels = []model.SpilloverChannel{{Kind: "it_line", TargetSilo: 3, Reliability: 0.75}}
		hoardSuits, hoardSupplies, hoardMaterials := silo.EscapePrep.SuitsFunctional, silo.EscapePrep.SuppliesPacked, silo.EscapePrep.MaterialsPacked
		channel := silo.SpilloverPrep.Channels[0]

		result := RunEscapeResponseDecision(silo)
		if result == nil {
			t.Fatalf("seed %d: decision must fire at suspicion 0.5", seed)
		}
		seen[result.Decision] = true
		if silo.EscapePrep.SuitsFunctional != hoardSuits ||
			silo.EscapePrep.SuppliesPacked != hoardSupplies ||
			silo.EscapePrep.MaterialsPacked != hoardMaterials {
			t.Fatalf("seed %d: decision %s mutated the hoarded ledger: suits %d->%d supplies %v->%v materials %v->%v",
				seed, result.Decision, hoardSuits, silo.EscapePrep.SuitsFunctional,
				hoardSupplies, silo.EscapePrep.SuppliesPacked, hoardMaterials, silo.EscapePrep.MaterialsPacked)
		}
		if len(silo.SpilloverPrep.Channels) != 1 || silo.SpilloverPrep.Channels[0] != channel {
			t.Fatalf("seed %d: decision %s mutated spillover channels", seed, result.Decision)
		}
		if len(result.Weights) != 3 || result.Factors == "" {
			t.Fatalf("seed %d: decision log must carry weights and factors: %+v", seed, result)
		}
	}
	for _, decision := range []string{model.EscapeResponseSuppress, model.EscapeResponseConceal, model.EscapeResponseIgnore} {
		if !seen[decision] {
			t.Fatalf("decision %s never drawn across seeds; weights mis-calibrated", decision)
		}
	}
}

// TestEscapeResponseDecisionDeterministic 同种子 + 同状态 → 同一决策。
func TestEscapeResponseDecisionDeterministic(t *testing.T) {
	draw := func(seed int64) string {
		SetRNG(rand.New(rand.NewSource(seed)))
		result := RunEscapeResponseDecision(escapeResponseTestSilo())
		if result == nil {
			t.Fatalf("seed %d: decision must fire", seed)
		}
		return result.Decision
	}
	for _, seed := range []int64{7, 8, 9} {
		if a, b := draw(seed), draw(seed); a != b {
			t.Fatalf("seed %d diverged: %s vs %s", seed, a, b)
		}
	}
}

// TestEscapeResponseSuppressEffects 镇压只压组织：怀疑度被消费、戒备与
// 恐慌上浮，囤积不动。
func TestEscapeResponseSuppressEffects(t *testing.T) {
	SetRNG(rand.New(rand.NewSource(1)))
	silo := escapeResponseTestSilo()
	suspicionBefore := silo.EscapeResponse.OrgSuspicion

	var result *EscapeResponseDecisionResult
	for seed := int64(1); seed <= 40 && result == nil; seed++ {
		SetRNG(rand.New(rand.NewSource(seed)))
		trial := escapeResponseTestSilo()
		r := RunEscapeResponseDecision(trial)
		if r != nil && r.Decision == model.EscapeResponseSuppress {
			silo, result = trial, r
		}
	}
	if result == nil {
		t.Fatal("no seed produced suppress; weights mis-calibrated")
	}
	if silo.EscapeResponse.OrgSuspicion >= suspicionBefore {
		t.Fatalf("suppress must consume suspicion: %v -> %v",
			suspicionBefore, silo.EscapeResponse.OrgSuspicion)
	}
	for _, prof := range silo.Professions {
		if prof.PanicValue <= 0 {
			t.Fatalf("suppress must raise panic for %s", prof.Name)
		}
		if (prof.Name == "Judicial" || prof.Name == "IT") && prof.Alertness <= 0 {
			t.Fatalf("suppress must raise alertness for %s", prof.Name)
		}
		if prof.Name == "Supply" && prof.Alertness != 0 {
			t.Fatal("suppress must not raise alertness for unrelated departments")
		}
	}
}

// TestEscapeResponseIgnoreRegistersIncident 不作为的后果 (ADR-0017)：
// 不再直接累加风险度，改登记 ignore 事件，经观测管线进入 1 号信念
// (异常悬而未决，1 号可能自己注意到)。
func TestEscapeResponseIgnoreRegistersIncident(t *testing.T) {
	var silo *model.Silo
	for seed := int64(1); ; seed++ {
		SetRNG(rand.New(rand.NewSource(seed)))
		trial := escapeResponseTestSilo()
		r := RunEscapeResponseDecision(trial)
		if r != nil && r.Decision == model.EscapeResponseIgnore {
			silo = trial
			break
		}
		if seed > 100 {
			t.Fatal("no seed produces ignore; weights mis-calibrated")
		}
	}
	if silo.SafeguardRisk != 0 {
		t.Fatalf("ignore must not write safeguard risk directly: %v", silo.SafeguardRisk)
	}
	if silo.SafeguardIncidents[SafeguardIncidentIgnore] != 1 {
		t.Fatalf("ignore must register one incident: %+v", silo.SafeguardIncidents)
	}
}

// TestEscapeResponseCoverMonth 蒙蔽维持：按月付 R_op；付不起时崩塌并
// 登记蒙蔽崩塌事件 (ADR-0017)，不再直接累加风险度。
func TestEscapeResponseCoverMonth(t *testing.T) {
	// 付得起：扣费，维持 (低暴露概率下取一个不暴露的种子)。
	silo := escapeResponseTestSilo()
	silo.EscapeResponse.CoverActive = true
	silo.ItAuthority.ROp = 60
	SetRNG(rand.New(rand.NewSource(1234)))
	collapsed := EscapeResponseCoverMonth(silo)
	if collapsed {
		t.Fatalf("cover with full budget must hold at base leak chance")
	}
	if silo.ItAuthority.ROp >= 60 {
		t.Fatalf("cover must drain R_op: %v", silo.ItAuthority.ROp)
	}
	if silo.EscapeResponse.CoverMonths != 1 || !silo.EscapeResponse.CoverActive {
		t.Fatalf("cover state wrong: %+v", silo.EscapeResponse)
	}

	// 付不起：崩塌，登记蒙蔽崩塌事件，蒙蔽结束；风险度不被直接改写。
	poor := escapeResponseTestSilo()
	poor.EscapeResponse.CoverActive = true
	poor.ItAuthority.ROp = 1
	rsBefore := poor.SafeguardRisk
	if !EscapeResponseCoverMonth(poor) {
		t.Fatal("unaffordable cover must collapse")
	}
	if poor.EscapeResponse.CoverActive {
		t.Fatal("collapsed cover must end")
	}
	if poor.SafeguardRisk != rsBefore {
		t.Fatalf("collapse must not write safeguard risk directly: %v -> %v", rsBefore, poor.SafeguardRisk)
	}
	if poor.SafeguardIncidents[SafeguardIncidentCoverCollapse] != 1 {
		t.Fatalf("collapse must register one cover_collapse incident: %+v", poor.SafeguardIncidents)
	}
}

// TestEscapeResponseCooldownBlocksRefire 决策冷却内不重复点火；蒙蔽
// 进行中不做新决策。
func TestEscapeResponseCooldownBlocksRefire(t *testing.T) {
	SetRNG(rand.New(rand.NewSource(9)))
	silo := escapeResponseTestSilo()
	first := RunEscapeResponseDecision(silo)
	if first == nil {
		t.Fatal("first decision must fire")
	}
	// 同月再点火：冷却未过。
	if again := RunEscapeResponseDecision(silo); again != nil {
		t.Fatal("decision must not refire within cooldown")
	}
	// 冷却已过 (4 个月后)：怀疑度重新到门槛时可再次点火。
	silo.CurrentMonth += 4
	silo.EscapeResponse.OrgSuspicion = 0.5
	silo.EscapeResponse.CoverActive = false
	if again := RunEscapeResponseDecision(silo); again == nil {
		t.Fatal("decision must refire after cooldown with suspicion above threshold")
	}
}

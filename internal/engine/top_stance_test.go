package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

func stanceTestSilo() *model.Silo {
	silo := &model.Silo{}
	silo.SetMayorLegitimacy(0.6)
	silo.Rebellion = 0.3
	silo.Authority.W = 0.5
	silo.Authority.PoliceObedience = 0.7
	return silo
}

// 零值 = 阻挠：未初始化的立场读出来就是阻挠 (16 §6.1 升级语义)。
func TestTopStanceZeroValueIsBlock(t *testing.T) {
	var s model.TopStanceState
	if got := s.MayorStance(); got != model.TopStanceBlock {
		t.Fatalf("mayor zero stance = %q, want block", got)
	}
	if got := s.ITStance(); got != model.TopStanceBlock {
		t.Fatalf("it zero stance = %q, want block", got)
	}
}

// 确定性：同种子 + 同状态序列 → 同样的立场轨迹 (裁量来自权重随机，
// 确定性来自种子)。
func TestTopStanceReevalDeterministic(t *testing.T) {
	run := func(seed int64) string {
		SetRNG(rand.New(rand.NewSource(seed)))
		silo := stanceTestSilo()
		world := &model.World{Flags: map[string]bool{}}
		ReevalTopStance(silo, world, "both")
		traj := silo.TopStance.Mayor + "/" + silo.TopStance.IT
		// 状态漂移后第二次重估 (磋商持续、立场可反复重估)。
		silo.Rebellion = 0.7
		silo.TopStance.Guarantee = 0.5
		ReevalTopStance(silo, world, "both")
		return traj + "|" + silo.TopStance.Mayor + "/" + silo.TopStance.IT
	}
	a := run(4001)
	b := run(4001)
	if a != b {
		t.Fatalf("same seed diverged: %v vs %v", a, b)
	}
	if c := run(4002); c == a {
		// 不同种子允许巧合相同，但两组权重下四档全同的概率极低；
		// 若相同说明随机源没有真正参与决策。
		t.Logf("warning: different seed produced identical trajectory %v", c)
	}
}

// 权重塑形：高叛乱 + 命令源枯竭 + 可信保障 + 警察服从转移把立场分布
// 推向不阻挠/配合；低叛乱 + 高 W 保持阻挠主导。权重是输入不是阈值：
// 任何单条状态都不锁死结果。
func TestTopStanceWeightShaping(t *testing.T) {
	counts := func(mutate func(*model.Silo, *model.World)) map[string]int {
		out := map[string]int{}
		for seed := int64(0); seed < 400; seed++ {
			SetRNG(rand.New(rand.NewSource(seed)))
			silo := stanceTestSilo()
			world := &model.World{Flags: map[string]bool{}}
			mutate(silo, world)
			ReevalTopStance(silo, world, "it")
			out[silo.TopStance.IT]++
		}
		return out
	}

	hostile := counts(func(s *model.Silo, w *model.World) {
		s.Rebellion = 0.1
		s.Authority.W = 0.9
	})
	if hostile[model.TopStanceBlock] <= hostile[model.TopStanceNonBlock]+hostile[model.TopStanceCooperate] {
		t.Fatalf("high W + low rebellion should stay block-dominant, got %v", hostile)
	}

	drifted := counts(func(s *model.Silo, w *model.World) {
		s.Rebellion = 0.85
		s.Authority.W = 0.15
		s.Authority.PoliceObedience = 0.3
		s.TopStance.Guarantee = 0.8
		s.Silo1Destroyed = true
	})
	if drifted[model.TopStanceBlock] >= drifted[model.TopStanceNonBlock]+drifted[model.TopStanceCooperate] {
		t.Fatalf("collapsed pillars should drift away from block, got %v", drifted)
	}
}

// 重估不单调：同一局内立场可以回摆 (玉石俱焚保留为概率，不被禁止)。
func TestTopStanceReevalCanSwingBack(t *testing.T) {
	silo := stanceTestSilo()
	world := &model.World{Flags: map[string]bool{}}
	saw := map[string]bool{}
	for seed := int64(0); seed < 200 && len(saw) < 3; seed++ {
		SetRNG(rand.New(rand.NewSource(seed)))
		s := *silo
		ReevalTopStance(&s, world, "mayor")
		saw[s.TopStance.Mayor] = true
	}
	if len(saw) < 2 {
		t.Fatalf("expected the weighted draw to produce more than one stance across seeds, got %v", saw)
	}
}

// 保障可信度夹在 [0,1]。
func TestTopStanceGuaranteeClamp(t *testing.T) {
	silo := stanceTestSilo()
	addTopStanceGuarantee(silo, 1.7)
	if silo.TopStance.Guarantee != 1 {
		t.Fatalf("guarantee = %v, want clamped 1", silo.TopStance.Guarantee)
	}
	addTopStanceGuarantee(silo, -3)
	if silo.TopStance.Guarantee != 0 {
		t.Fatalf("guarantee = %v, want clamped 0", silo.TopStance.Guarantee)
	}
}

// 封闭集校验：flag 白名单；施加路径经 ScopeContext 生效。
func TestTopStanceEffects(t *testing.T) {
	if err := validateEffect(ContentEffect{Type: "top_stance_reeval", Flag: "mayor"}); err != nil {
		t.Fatalf("reeval mayor should validate: %v", err)
	}
	if err := validateEffect(ContentEffect{Type: "top_stance_reeval", Flag: "silo1"}); err == nil {
		t.Fatal("reeval with unknown actor must fail validation")
	}
	if err := validateEffect(ContentEffect{Type: "top_stance_guarantee", Value: 0.2}); err != nil {
		t.Fatalf("guarantee should validate: %v", err)
	}

	SetRNG(rand.New(rand.NewSource(4003)))
	silo := stanceTestSilo()
	sc := &ScopeContext{Self: silo, World: &model.World{Flags: map[string]bool{}}}
	ApplyContentEffectsScoped([]ContentEffect{
		{Type: "top_stance_guarantee", Value: 0.4},
		{Type: "top_stance_reeval", Flag: "both"},
	}, sc)
	if silo.TopStance.Guarantee != 0.4 {
		t.Fatalf("guarantee = %v, want 0.4", silo.TopStance.Guarantee)
	}
	if !silo.TopStance.Initialized || silo.TopStance.Reevaluations != 1 {
		t.Fatalf("reeval not recorded: %+v", silo.TopStance)
	}
	switch silo.TopStance.MayorStance() {
	case model.TopStanceBlock, model.TopStanceNonBlock, model.TopStanceCooperate:
	default:
		t.Fatalf("mayor stance %q out of enum", silo.TopStance.Mayor)
	}
}

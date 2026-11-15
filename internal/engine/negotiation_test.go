package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

func negotiationTestSilo() *model.Silo {
	silo := &model.Silo{}
	silo.SetMayorLegitimacy(0.6)
	silo.Rebellion = 0.4
	silo.Authority.W = 55
	silo.Authority.PoliceObedience = 0.7
	silo.Negotiation.Initialized = true
	silo.Negotiation.ChannelOpen = true
	silo.Negotiation.MayorDisposition = model.MayorDispositionPragmatist
	return silo
}

// 权重不被状态锁死：极端输入下三条路仍全部有非零概率——不存在
// "叛乱 ≥ X → 必须妥协"的阈值 (18 §3.1)。
func TestMayorPathNoThresholdLock(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*model.Silo)
	}{
		{"满叛乱高保障", func(s *model.Silo) {
			s.Rebellion = 1.0
			s.TopStance.Guarantee = 1.0
			s.SetMayorLegitimacy(0)
			s.Authority.W = 0
			s.Authority.PoliceObedience = 0
		}},
		{"零叛乱高镇压力", func(s *model.Silo) {
			s.Rebellion = 0
			s.SetMayorLegitimacy(1)
			s.Authority.W = 100
			s.Authority.PoliceObedience = 1
		}},
	}
	for _, tc := range cases {
		silo := negotiationTestSilo()
		tc.mutate(silo)
		w := MayorPathWeights(silo, &model.World{Flags: map[string]bool{}}, NegotiationKindMonth)
		if w.suppress <= 0 || w.talk <= 0 || w.concede <= 0 {
			t.Fatalf("%s: 三条路必须全部保留非零权重 (裁量不被锁死)，got %+v", tc.name, w)
		}
	}
}

// 权重塑形：高叛乱＋保障可信把分布推向妥协；低叛乱＋高镇压力保持镇压主导。
func TestMayorPathWeightShaping(t *testing.T) {
	counts := func(mutate func(*model.Silo)) map[string]int {
		out := map[string]int{}
		for seed := int64(0); seed < 400; seed++ {
			SetRNG(rand.New(rand.NewSource(seed)))
			silo := negotiationTestSilo()
			silo.Negotiation.MayorDisposition = model.MayorDispositionPragmatist
			mutate(silo)
			w := MayorPathWeights(silo, &model.World{Flags: map[string]bool{}}, NegotiationKindMonth)
			out[DrawMayorPath(w)]++
		}
		return out
	}

	entrenched := counts(func(s *model.Silo) {
		s.Rebellion = 0.1
		s.SetMayorLegitimacy(0.9)
		s.Authority.W = 90
		s.Authority.PoliceObedience = 0.9
		s.Negotiation.MayorDisposition = model.MayorDispositionHardliner
	})
	if entrenched[model.NegotiationPathSuppress] <= entrenched[model.NegotiationPathConcede] {
		t.Fatalf("high legitimacy + high force + hardliner should stay suppress-dominant, got %v", entrenched)
	}

	cornered := counts(func(s *model.Silo) {
		s.Rebellion = 0.9
		s.SetMayorLegitimacy(0.15)
		s.Authority.W = 5
		s.Authority.PoliceObedience = 0.2
		s.TopStance.Guarantee = 0.8
		s.Negotiation.MayorDisposition = model.MayorDispositionSurvivor
	})
	if cornered[model.NegotiationPathConcede] <= cornered[model.NegotiationPathSuppress] {
		t.Fatalf("collapsed pillars + credible guarantee should drift toward concede, got %v", cornered)
	}
}

// 拖延疲劳：连续无进展回合压低谈判权重 (软衰减，不是阈值闸)。
func TestMayorPathStallFatigue(t *testing.T) {
	silo := negotiationTestSilo()
	world := &model.World{Flags: map[string]bool{}}
	fresh := MayorPathWeights(silo, world, NegotiationKindMonth)
	silo.Negotiation.StalledRounds = 8
	stalled := MayorPathWeights(silo, world, NegotiationKindMonth)
	if stalled.talk >= fresh.talk {
		t.Fatalf("stall fatigue must lower talk weight: fresh=%+v stalled=%+v", fresh, stalled)
	}
	if stalled.suppress <= 0 || stalled.concede <= 0 {
		t.Fatalf("stall fatigue must not lock the path: %+v", stalled)
	}
}

// 确定性：同种子 + 同状态序列 → 同样的谈判决策轨迹。
func TestNegotiationDrawsDeterministic(t *testing.T) {
	run := func(seed int64) string {
		SetRNG(rand.New(rand.NewSource(seed)))
		silo := negotiationTestSilo()
		world := &model.World{Flags: map[string]bool{}}
		traj := ""
		for i := 0; i < 6; i++ {
			traj += DrawMayorPath(MayorPathWeights(silo, world, NegotiationKindMonth))
			traj += "/"
			traj += DrawRebelPosture(RebelPostureWeights(silo))
			traj += "/"
			traj += DrawITReaction(ITReactionWeights(silo, world))
			if DrawRebelAccept(RebelAcceptWeights(silo)) {
				traj += "/A"
			} else {
				traj += "/R"
			}
			traj += "|"
		}
		return traj + DrawMayorDisposition()
	}
	a := run(4101)
	b := run(4101)
	if a != b {
		t.Fatalf("same seed diverged: %v vs %v", a, b)
	}
}

// 市长性情抽取：三档都能出现 (近等权)。
func TestMayorDispositionDistribution(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(0); seed < 300; seed++ {
		SetRNG(rand.New(rand.NewSource(seed)))
		seen[DrawMayorDisposition()]++
	}
	for _, disposition := range []string{model.MayorDispositionHardliner, model.MayorDispositionPragmatist, model.MayorDispositionSurvivor} {
		if seen[disposition] == 0 {
			t.Fatalf("disposition %s never drawn across 300 seeds: %v", disposition, seen)
		}
	}
}

// 叛乱方接受决策：背信记忆与高叛乱推高拒绝；可信保障推高接受。
// 失败 (拒绝) 是合法结果：总权重为零时兜底拒绝，不默认放行。
func TestRebelAcceptShaping(t *testing.T) {
	rate := func(mutate func(*model.Silo)) int {
		accepted := 0
		for seed := int64(0); seed < 400; seed++ {
			SetRNG(rand.New(rand.NewSource(seed)))
			silo := negotiationTestSilo()
			mutate(silo)
			if DrawRebelAccept(RebelAcceptWeights(silo)) {
				accepted++
			}
		}
		return accepted
	}
	trusting := rate(func(s *model.Silo) {
		s.TopStance.Guarantee = 0.9
		s.Rebellion = 0.3
	})
	burned := rate(func(s *model.Silo) {
		s.TopStance.Guarantee = 0.05
		s.Rebellion = 0.9
		s.Negotiation.SuppressRounds = 5
	})
	if trusting <= burned {
		t.Fatalf("credible guarantee should raise acceptance: trusting=%d burned=%d", trusting, burned)
	}
	var zero rebelAcceptWeights
	if DrawRebelAccept(zero.clamp()) {
		t.Fatal("zero weights must fall back to reject (failure is a legal outcome)")
	}
}

// IT 反应：命令源枯竭 + 可信保障推向不干涉；专线仍在下令 + 高 W 保留破坏。
func TestITReactionShaping(t *testing.T) {
	counts := func(mutate func(*model.Silo, *model.World)) map[string]int {
		out := map[string]int{}
		for seed := int64(0); seed < 400; seed++ {
			SetRNG(rand.New(rand.NewSource(seed)))
			silo := negotiationTestSilo()
			world := &model.World{Flags: map[string]bool{}}
			mutate(silo, world)
			out[DrawITReaction(ITReactionWeights(silo, world))]++
		}
		return out
	}
	loyal := counts(func(s *model.Silo, w *model.World) {
		s.Authority.W = 90
		s.Authority.PoliceObedience = 0.9
	})
	if loyal[model.NegotiationITUndermine] == 0 {
		t.Fatalf("high W with live command source must keep undermine possible: %v", loyal)
	}
	collapsed := counts(func(s *model.Silo, w *model.World) {
		s.Authority.W = 5
		s.Authority.PoliceObedience = 0.3
		s.TopStance.Guarantee = 0.8
		s.Silo1Destroyed = true
	})
	if collapsed[model.NegotiationITStandAside] <= collapsed[model.NegotiationITUndermine] {
		t.Fatalf("collapsed pillars + guarantee should drift to stand-aside, got %v", collapsed)
	}
}

// 叛乱侵蚀正统性：与叛乱水平成正比；叛乱为 0 时逐位不变 (18 §4)。
func TestRebellionLegitimacyErosion(t *testing.T) {
	silo := negotiationTestSilo()
	silo.Rebellion = 0
	before := silo.MayorLegitimacy()
	ApplyRebellionLegitimacyErosion(silo, 1.0/12.0)
	if silo.MayorLegitimacy() != before {
		t.Fatalf("zero rebellion must not erode legitimacy: %v -> %v", before, silo.MayorLegitimacy())
	}

	silo.Rebellion = 0.8
	before = silo.MayorLegitimacy()
	ApplyRebellionLegitimacyErosion(silo, 1.0/12.0)
	if silo.MayorLegitimacy() >= before {
		t.Fatalf("active rebellion must erode legitimacy: %v -> %v", before, silo.MayorLegitimacy())
	}

	high := negotiationTestSilo()
	high.Rebellion = 0.8
	low := negotiationTestSilo()
	low.Rebellion = 0.2
	highBefore, lowBefore := high.MayorLegitimacy(), low.MayorLegitimacy()
	ApplyRebellionLegitimacyErosion(high, 1.0/12.0)
	ApplyRebellionLegitimacyErosion(low, 1.0/12.0)
	if highBefore-high.MayorLegitimacy() <= lowBefore-low.MayorLegitimacy() {
		t.Fatalf("erosion must scale with rebellion: high=%v low=%v", highBefore-high.MayorLegitimacy(), lowBefore-low.MayorLegitimacy())
	}
}

// 决策日志归一化权重：三档之和为 1 (三位小数)。
func TestNegotiationNormalize(t *testing.T) {
	silo := negotiationTestSilo()
	world := &model.World{Flags: map[string]bool{}}
	sum := 0.0
	for _, v := range NormalizeMayorPath(MayorPathWeights(silo, world, NegotiationKindMonth)) {
		sum += v
	}
	if sum < 0.998 || sum > 1.002 {
		t.Fatalf("normalized mayor path weights must sum to 1, got %v", sum)
	}
	sum = 0
	for _, v := range NormalizeITReaction(ITReactionWeights(silo, world)) {
		sum += v
	}
	if sum < 0.998 || sum > 1.002 {
		t.Fatalf("normalized IT weights must sum to 1, got %v", sum)
	}
}

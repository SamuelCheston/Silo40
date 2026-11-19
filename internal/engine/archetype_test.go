package engine

import (
	"testing"

	"silo40/internal/model"
)

// TestStationArchetypeTable locks 19 §3.3：号数 2–17 全覆盖、每档 5–6 个号、
// 相邻号数不同档 (重开手感)、2/9/17 分属三档 (实测对比可见)，且派生纯查表
// 不消耗 RNG。
func TestStationArchetypeTable(t *testing.T) {
	counts := map[string]int{}
	prev := ""
	for number := 2; number <= 17; number++ {
		id := ArchetypeForStation(number)
		if _, ok := ArchetypeByID(id); !ok {
			t.Fatalf("station %d maps to unregistered archetype %q", number, id)
		}
		counts[id]++
		if id == prev {
			t.Fatalf("stations %d and %d share archetype %q — adjacent numbers must differ", number-1, number, id)
		}
		prev = id
	}
	if len(counts) != 3 {
		t.Fatalf("want 3 archetypes, got %v", counts)
	}
	for id, count := range counts {
		if count < 5 || count > 6 {
			t.Fatalf("archetype %s covers %d stations, want 5–6", id, count)
		}
	}
	if ArchetypeForStation(2) == ArchetypeForStation(9) ||
		ArchetypeForStation(9) == ArchetypeForStation(17) ||
		ArchetypeForStation(2) == ArchetypeForStation(17) {
		t.Fatalf("stations 2/9/17 must land on three different archetypes for the empirical comparison: %s/%s/%s",
			ArchetypeForStation(2), ArchetypeForStation(9), ArchetypeForStation(17))
	}
	if got := ArchetypeForStation(1); got != "" {
		t.Fatalf("station 1 is protected and must not map to an archetype, got %q", got)
	}
	if got := ArchetypeForStation(18); got != "" {
		t.Fatalf("station 18 is protected and must not map to an archetype, got %q", got)
	}
}

func resourceAmount(silo *model.Silo, resourceType string) float64 {
	for i := range silo.Resources {
		if silo.Resources[i].Type == resourceType {
			return silo.Resources[i].Amount
		}
	}
	return -1
}

// TestApplyArchetypeOpening locks 各档开局旋钮与"空原型逐位不变"。
func TestApplyArchetypeOpening(t *testing.T) {
	cfg := DefaultAuthorityConfig()

	base := CreateInitialSilo("Base", 122, nil)
	if base.Authority.W != cfg.NewGameW {
		t.Fatalf("baseline W = %v, want %v", base.Authority.W, cfg.NewGameW)
	}

	bedrock := CreateInitialSilo("Bedrock", 122, nil)
	bedrock.Archetype = ArchetypeBedrock
	ApplyArchetypeOpening(bedrock)
	bedrockCfg, _ := ArchetypeByID(ArchetypeBedrock)
	if want := cfg.NewGameW + bedrockCfg.W0Delta; bedrock.Authority.W != want {
		t.Fatalf("bedrock W = %v, want %v", bedrock.Authority.W, want)
	}
	if got := resourceAmount(bedrock, "Energy"); got != 5000 {
		t.Fatalf("bedrock must not touch resources, Energy = %v", got)
	}

	rift := CreateInitialSilo("Rift", 122, nil)
	rift.Archetype = ArchetypeRift
	ApplyArchetypeOpening(rift)
	riftCfg, _ := ArchetypeByID(ArchetypeRift)
	if want := cfg.NewGameW + riftCfg.W0Delta; rift.Authority.W != want {
		t.Fatalf("rift W = %v, want %v", rift.Authority.W, want)
	}

	famine := CreateInitialSilo("Famine", 122, nil)
	famine.Archetype = ArchetypeFamine
	ApplyArchetypeOpening(famine)
	famineCfg, _ := ArchetypeByID(ArchetypeFamine)
	if famine.Authority.W != cfg.NewGameW {
		t.Fatalf("famine must not touch W, got %v", famine.Authority.W)
	}
	for _, resourceType := range []string{"Energy", "Materials", "Supplies"} {
		base := 5000.0
		if resourceType == "Materials" {
			base = 2000.0
		}
		if got, want := resourceAmount(famine, resourceType), base*famineCfg.ResourceFactor; got != want {
			t.Fatalf("famine %s = %v, want %v", resourceType, got, want)
		}
	}

	// 空原型：ApplyArchetypeOpening 必须逐位不改。
	empty := CreateInitialSilo("Empty", 122, nil)
	beforeW, beforeEnergy := empty.Authority.W, resourceAmount(empty, "Energy")
	ApplyArchetypeOpening(empty)
	if empty.Authority.W != beforeW || resourceAmount(empty, "Energy") != beforeEnergy {
		t.Fatalf("empty archetype must stay bit-identical")
	}
}

// TestAuthorityConfigForArchetype locks 磐石档 W 衰减放缓、其余档与空原型读
// 默认配置。
func TestAuthorityConfigForArchetype(t *testing.T) {
	defaults := DefaultAuthorityConfig()
	if got := AuthorityConfigFor(nil); got != defaults {
		t.Fatalf("nil silo must read default config")
	}
	plain := &model.Silo{}
	if got := AuthorityConfigFor(plain); got != defaults {
		t.Fatalf("empty archetype must read default config")
	}
	bedrock := &model.Silo{Archetype: ArchetypeBedrock}
	cfg := AuthorityConfigFor(bedrock)
	bedrockArch, _ := ArchetypeByID(ArchetypeBedrock)
	if cfg.WHalfLifeMonths != bedrockArch.WHalfLifeMonths || cfg.WHalfLifeMonths <= defaults.WHalfLifeMonths {
		t.Fatalf("bedrock half-life = %v, want slower than baseline %v", cfg.WHalfLifeMonths, defaults.WHalfLifeMonths)
	}
	rift := &model.Silo{Archetype: ArchetypeRift}
	if got := AuthorityConfigFor(rift); got != defaults {
		t.Fatalf("rift must keep baseline W dynamics, got %+v", got)
	}
}

// TestAmplifyOrderDelta locks 裂隙档只放大恐慌/叛乱的正向增量。
func TestAmplifyOrderDelta(t *testing.T) {
	rift := &model.Silo{Archetype: ArchetypeRift}
	arch, _ := ArchetypeByID(ArchetypeRift)
	if got := AmplifyOrderDelta(rift, "panic_value", 0.1); got != 0.1*arch.PanicRebellionAmp {
		t.Fatalf("rift panic amplification = %v, want %v", got, 0.1*arch.PanicRebellionAmp)
	}
	if got := AmplifyOrderDelta(rift, "rebellion", 0.05); got != 0.05*arch.PanicRebellionAmp {
		t.Fatalf("rift rebellion amplification = %v, want %v", got, 0.05*arch.PanicRebellionAmp)
	}
	if got := AmplifyOrderDelta(rift, "panic_value", -0.1); got != -0.1 {
		t.Fatalf("panic decay must not be amplified, got %v", got)
	}
	if got := AmplifyOrderDelta(rift, "cohesion", 0.1); got != 0.1 {
		t.Fatalf("non-order metrics must pass through, got %v", got)
	}
	plain := &model.Silo{}
	if got := AmplifyOrderDelta(plain, "panic_value", 0.1); got != 0.1 {
		t.Fatalf("empty archetype must pass through, got %v", got)
	}
}

// TestArchetypeDetectionFactor locks 检测系数档间取值与空原型恒等。
func TestArchetypeDetectionFactor(t *testing.T) {
	if got := ArchetypeDetectionFactor(nil); got != 1 {
		t.Fatalf("nil silo factor = %v, want 1", got)
	}
	if got := ArchetypeDetectionFactor(&model.Silo{}); got != 1 {
		t.Fatalf("empty archetype factor = %v, want 1", got)
	}
	bedrock, _ := ArchetypeByID(ArchetypeBedrock)
	if got := ArchetypeDetectionFactor(&model.Silo{Archetype: ArchetypeBedrock}); got != bedrock.DetectionFactor || got <= 1 {
		t.Fatalf("bedrock factor = %v, want >1 (%v)", got, bedrock.DetectionFactor)
	}
	rift, _ := ArchetypeByID(ArchetypeRift)
	if got := ArchetypeDetectionFactor(&model.Silo{Archetype: ArchetypeRift}); got != rift.DetectionFactor || got >= 1 {
		t.Fatalf("rift factor = %v, want <1 (%v)", got, rift.DetectionFactor)
	}
}

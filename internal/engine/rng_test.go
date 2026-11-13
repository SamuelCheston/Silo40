package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// TestSeededWorldGenerationIsReproducible 锁定 RNG 注入契约：同一种子必须
// 生成逐字段相同的世界（居民/队列/思潮/特质随机全部走注入源）。
func TestSeededWorldGenerationIsReproducible(t *testing.T) {
	SetRNG(rand.New(rand.NewSource(42)))
	siloA := CreateInitialSilo("Repro", 122, []string{"leak"})
	agentA := CreateInitialAgent("Agent", "IT", []string{}, siloA)

	SetRNG(rand.New(rand.NewSource(42)))
	siloB := CreateInitialSilo("Repro", 122, []string{"leak"})
	agentB := CreateInitialAgent("Agent", "IT", []string{}, siloB)

	if !reflect.DeepEqual(siloA, siloB) {
		t.Fatalf("same seed produced different silos")
	}
	if !reflect.DeepEqual(agentA, agentB) {
		t.Fatalf("same seed produced different agents")
	}

	// 不同种子应当生成不同的世界（随机字段足够多，碰撞概率可忽略）。
	SetRNG(rand.New(rand.NewSource(43)))
	siloC := CreateInitialSilo("Repro", 122, []string{"leak"})
	if reflect.DeepEqual(siloA.Residents, siloC.Residents) &&
		reflect.DeepEqual(siloA.Cohorts, siloC.Cohorts) {
		t.Fatalf("different seeds produced identical worlds")
	}
}

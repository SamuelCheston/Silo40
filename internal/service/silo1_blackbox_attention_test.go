package service

import (
	"math/rand"
	"path/filepath"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// TestAttentionDirectReadWhenBlackBoxDisabled 00 卷裁决 6 关闭侧：功能关闭时
// Attention=Rebellion 直读逐位不变。
func TestAttentionDirectReadWhenBlackBoxDisabled(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "attention-off.db"))
	if _, err := svc.CreateGame(threeSiloRequest(81)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.world.Silo1BlackBoxEnabled {
		t.Fatal("threeSiloRequest 默认不应启用黑箱功能")
	}
	for i, silo := range svc.silos {
		silo.Rebellion = 0.11 + 0.23*float64(i) // 各不相同的精确值
	}
	engine.SetRNG(rand.New(rand.NewSource(81)))
	var logs []string
	var stories []model.StoryEvent
	svc.runSilo1StateMachine(&logs, &stories)
	for _, silo := range svc.silos {
		if got := svc.world.Silo1.Attention[silo.ID]; got != silo.Rebellion {
			t.Fatalf("功能关闭必须保持直读：silo %d attention=%v rebellion=%v",
				silo.ID, got, silo.Rebellion)
		}
	}
}

// TestAttentionBandedWhenBlackBoxEnabled 00 卷裁决 6 开启侧：功能开启时
// Attention 为 22-02 §3.7 分档翻转噪声观测，输出只落在合法档位值上。
func TestAttentionBandedWhenBlackBoxEnabled(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "attention-on.db"))
	req := threeSiloRequest(82)
	req.Silo1BlackBox = true
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	// 覆盖 low/mid/high 三个真档。
	rebellions := []float64{0.10, 0.45, 0.90}
	legal := map[float64]bool{0.15: true, 0.45: true, 0.80: true}
	for seed := int64(0); seed < 12; seed++ {
		for i, silo := range svc.silos {
			silo.Rebellion = rebellions[i%len(rebellions)]
		}
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		var logs []string
		var stories []model.StoryEvent
		svc.runSilo1StateMachine(&logs, &stories)
		for _, silo := range svc.silos {
			got := svc.world.Silo1.Attention[silo.ID]
			if !legal[got] {
				t.Fatalf("seed %d silo %d：Attention=%v 不是合法档位值 (真档叛乱=%v)",
					seed, silo.ID, got, silo.Rebellion)
			}
		}
	}
}

// TestAttentionBandFlipDeterministic 分档翻转噪声的确定性：同一种子逐位复现，
// 且每堡恰好消耗 1 次 DrawFloat64 (下一抽等于参考流第 N+1 抽)。
func TestAttentionBandFlipDeterministic(t *testing.T) {
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "attention-flip.db"))
	req := threeSiloRequest(84)
	req.Silo1BlackBox = true
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	for _, silo := range svc.silos {
		silo.Rebellion = 0.45 // mid 档：翻转可去 low 或 high
	}
	run := func(seed int64) map[uint]float64 {
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		var logs []string
		var stories []model.StoryEvent
		svc.runSilo1StateMachine(&logs, &stories)
		out := map[uint]float64{}
		for id, v := range svc.world.Silo1.Attention {
			out[id] = v
		}
		return out
	}
	first, second := run(7), run(7)
	for id, v := range first {
		if second[id] != v {
			t.Fatalf("同种子两次运行 Attention 不一致：silo %d %v != %v", id, v, second[id])
		}
	}
	// 消耗计数：N 堡各 1 次 → 下一抽等于参考流第 N+1 抽。
	engine.SetRNG(rand.New(rand.NewSource(9)))
	var logs []string
	var stories []model.StoryEvent
	svc.runSilo1StateMachine(&logs, &stories)
	ref := rand.New(rand.NewSource(9))
	for range svc.silos {
		ref.Float64()
	}
	if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("Attention 替换消耗次数不是每堡 1 次：%v != %v", got, want)
	}
}

package service

// P0.5 验收 #6 与存档体积：两堡 vs 单堡 sim 对照 + 实际存档尺寸。
// 对照数据同时打印到测试输出，供研究报告引用。

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// TestSaveSizeSingleVsTwoSilo 实测单堡与双堡存档体积 (验收 #5 的数字)。
func TestSaveSizeSingleVsTwoSilo(t *testing.T) {
	measure := func(t *testing.T, name string, req model.CreateGameRequest, months int) int64 {
		t.Helper()
		dbPath := filepath.Join(t.TempDir(), name+".db")
		svc := newTwoSiloService(t, dbPath)
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		for m := 1; m <= months; m++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("%s pass time %d: %v", name, m, err)
			}
			if svc.gameOver() {
				break
			}
		}
		if err := svc.persist(); err != nil {
			t.Fatalf("%s persist: %v", name, err)
		}
		info, err := os.Stat(dbPath)
		if err != nil {
			t.Fatalf("%s stat: %v", name, err)
		}
		return info.Size()
	}

	const months = 120
	single := measure(t, "single", model.CreateGameRequest{
		SiloName: "SizeSingle", AgentName: "A", Profession: "IT", Seed: 7,
	}, months)
	two := measure(t, "twosilo", twoSiloRequest(7), months)

	t.Logf("save size after %d months: single-silo=%d bytes (%.1f KB), two-silo=%d bytes (%.1f KB), ratio=%.2fx",
		months, single, float64(single)/1024, two, float64(two)/1024, float64(two)/float64(single))
	if two <= single {
		t.Fatalf("expected two-silo save larger than single-silo: %d vs %d", two, single)
	}
}

// simSummary 一种模式的聚合对照数据。
type simSummary struct {
	Games        int
	Endings      map[string]int
	TotalMonths  int
	TotalPop     int
	TotalReb     float64
	CrossSignals int
}

// TestTwoSiloVsSingleSimComparison 验收 #6：同种子下单堡与双堡的 sim 对照。
// 打印结局分布 / 平均时长 / 平均终值人口与叛乱 / 跨堡信号数。
func TestTwoSiloVsSingleSimComparison(t *testing.T) {
	seeds := []int64{7, 8, 9, 10, 11, 12}
	const maxMonths = 200

	runMode := func(t *testing.T, twoSilo bool) simSummary {
		t.Helper()
		sum := simSummary{Endings: map[string]int{}}
		for _, seed := range seeds {
			dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("sim-%d.db", seed))
			svc := newTwoSiloService(t, dbPath)
			req := model.CreateGameRequest{
				SiloName:   fmt.Sprintf("Sim-%d", seed),
				AgentName:  "Agent",
				Profession: "IT",
				Seed:       seed,
			}
			if twoSilo {
				req.Neighbor = &model.NeighborSiloConfig{}
			}
			if _, err := svc.CreateGame(req); err != nil {
				t.Fatalf("create seed %d two=%v: %v", seed, twoSilo, err)
			}
			months := 0
			for m := 1; m <= maxMonths; m++ {
				if _, err := svc.PassTime(1); err != nil {
					t.Fatalf("seed %d month %d: %v", seed, m, err)
				}
				months = m
				if svc.gameOver() {
					break
				}
			}
			sum.Games++
			sum.TotalMonths += months
			ending := "NO_ENDING(max_months)"
			if svc.silo.VictoryStatus != nil {
				ending = svc.silo.VictoryStatus.Type
			}
			sum.Endings[ending]++
			sum.TotalPop += svc.silo.TotalPopulation
			sum.TotalReb += svc.silo.Rebellion
			if svc.world != nil {
				sum.CrossSignals += svc.world.CrossSeq
			}
		}
		return sum
	}

	single := runMode(t, false)
	two := runMode(t, true)

	report := func(name string, s simSummary) {
		t.Logf("%s: games=%d avg_months=%.1f avg_final_pop=%.0f avg_final_rebellion=%.3f cross_signals=%d endings=%v",
			name, s.Games, float64(s.TotalMonths)/float64(s.Games),
			float64(s.TotalPop)/float64(s.Games), s.TotalReb/float64(s.Games),
			s.CrossSignals, s.Endings)
	}
	report("single-silo", single)
	report("two-silo   ", two)

	if two.CrossSignals == 0 {
		t.Fatalf("two-silo mode produced no cross-silo signals; the comparison would be vacuous")
	}
}

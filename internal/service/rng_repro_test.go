package service

import (
	"path/filepath"
	"reflect"
	"testing"

	"silo40/internal/model"
	"silo40/internal/repository"
)

// reproSnapshot 只取确定性数值字段，避开 GORM 时间戳。
type reproSnapshot struct {
	Year, Month        int
	Population         int
	Legitimacy         float64
	Cohesion           float64
	Rebellion          float64
	DeptTension        float64
	ClassFragmentation float64
	HistoryBurden      float64
	Countdown          float64
	SafeguardRisk      float64
	Resources          map[string]float64
	Ideologies         map[string]map[string]float64
	AgentAP            float64
	AgentSuspicion     float64
	AgentFragments     []string
	GameOver           bool
	EndingType         string
}

func snapshot(svc *GameService) reproSnapshot {
	silo := svc.silo
	snap := reproSnapshot{
		Year:               silo.CurrentYear,
		Month:              silo.CurrentMonth,
		Population:         silo.TotalPopulation,
		Legitimacy:         silo.Legitimacy,
		Cohesion:           silo.Cohesion,
		Rebellion:          silo.Rebellion,
		DeptTension:        silo.DeptTension,
		ClassFragmentation: silo.ClassFragmentation,
		HistoryBurden:      silo.HistoryBurden,
		Countdown:          silo.Countdown,
		SafeguardRisk:      silo.SafeguardRisk,
		Resources:          map[string]float64{},
		Ideologies:         map[string]map[string]float64{},
		AgentAP:            svc.agent.ActionPoints,
		AgentSuspicion:     svc.agent.SuspicionLevel,
		AgentFragments:     svc.agent.KnownFragments,
		GameOver:           silo.VictoryStatus != nil,
	}
	for _, r := range silo.Resources {
		snap.Resources[r.Type] = r.Amount
	}
	for _, p := range silo.Professions {
		snap.Ideologies[p.Name] = p.Ideologies
	}
	if silo.VictoryStatus != nil {
		snap.EndingType = silo.VictoryStatus.Type
	}
	return snap
}

func newSeededService(t *testing.T, seed int64) *GameService {
	t.Helper()
	db, err := repository.InitDB(filepath.Join(t.TempDir(), "repro.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := NewGameService(db)
	req := model.CreateGameRequest{
		SiloName:   "Repro",
		StartYear:  122,
		AgentName:  "Agent",
		Profession: "IT",
		Seed:       seed,
	}
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

// TestSameSeedSameOpsReproduceSameState 锁定核心复现契约：
// 同一种子 + 同一操作序列 => 完全相同的状态轨迹与结局。
func TestSameSeedSameOpsReproduceSameState(t *testing.T) {
	run := func(seed int64) reproSnapshot {
		svc := newSeededService(t, seed)
		for i := 0; i < 18; i++ {
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("pass time month %d: %v", i+1, err)
			}
		}
		return snapshot(svc)
	}

	a := run(42)
	b := run(42)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same seed + same ops diverged:\n A=%+v\n B=%+v", a, b)
	}

	// 不同种子应当走出不同轨迹（18 个月内随机消耗足够多，碰撞概率可忽略）。
	c := run(43)
	if reflect.DeepEqual(a, c) {
		t.Fatalf("different seeds produced identical trajectories")
	}
}

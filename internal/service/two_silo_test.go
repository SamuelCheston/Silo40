package service

// P0.5 双地堡原型验收测试：
//   #2 同种子双跑逐字节一致 (含跨堡投递顺序)
//   #3 跨堡事件在"本地结算后、时钟推进前"的独立阶段结算，顺序稳定
//   #4 计量隔离：一堡指标变化不渗到另一堡
//   #5 存档落两堡 + 跨堡队列并正确 Resume
// 单堡逐位不变 (#1) 由 parity_harness_test.go 前后对照保证。

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"
)

// newTwoSiloService 构建绑定固定 DB 的双堡服务 (含真实内容)。
func newTwoSiloService(t *testing.T, dbPath string) *GameService {
	t.Helper()
	db, err := repository.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	svc := NewGameService(db)
	dirs, err := resolveContentDirectories()
	if err != nil {
		t.Fatalf("resolve content directories: %v", err)
	}
	if err := svc.syncContentDefinitions(dirs); err != nil {
		t.Fatalf("sync content definitions: %v", err)
	}
	if err := svc.loadContentDefinitions(); err != nil {
		t.Fatalf("load content definitions: %v", err)
	}
	return svc
}

// twoSiloRequest 标准双堡开局。
func twoSiloRequest(seed int64) model.CreateGameRequest {
	return model.CreateGameRequest{
		SiloName:   fmt.Sprintf("TwoSilo-%d", seed),
		AgentName:  "PlayerAgent",
		Profession: "IT",
		Seed:       seed,
		Neighbor:   &model.NeighborSiloConfig{},
	}
}

// siloSnapshot 单堡可比快照 (确定性字段)。
type siloSnapshot struct {
	ID            uint             `json:"id"`
	Population    int              `json:"population"`
	Rebellion     float64          `json:"rebellion"`
	Cohesion      float64          `json:"cohesion"`
	Legitimacy    float64          `json:"legitimacy"`
	HistoryBurden float64          `json:"history_burden"`
	SafeguardRisk float64          `json:"safeguard_risk"`
	Year          int              `json:"year"`
	Month         int              `json:"month"`
	Ending        string           `json:"ending"`
	Resources     map[string]int64 `json:"resources"`
	Fires         map[string]int   `json:"fires"`
	NamedAlive    map[string]bool  `json:"named_alive,omitempty"` // 具名人物 (18 号) 生死态
}

// twoSiloRecord 世界级可比快照。
type twoSiloRecord struct {
	Seed         int64              `json:"seed"`
	Months       int                `json:"months"`
	WorldYear    int                `json:"world_year"`
	WorldMonth   int                `json:"world_month"`
	CrossSeq     int                `json:"cross_seq"`
	CrossPending int                `json:"cross_pending"`
	Silo1Phase   string             `json:"silo1_phase"`
	Silo1Alive   bool               `json:"silo1_alive"`
	Flags        map[string]bool    `json:"flags"`
	Attention    map[string]float64 `json:"attention"`
	Silos        []siloSnapshot     `json:"silos"`
	AgentAP      map[string]float64 `json:"agent_ap"`
}

// snapshotTwoSilo 采集当前世界快照 (用于双跑对比)。
func (s *GameService) snapshotTwoSilo(seed int64, months int) twoSiloRecord {
	rec := twoSiloRecord{
		Seed:         seed,
		Months:       months,
		WorldYear:    s.world.CurrentYear,
		WorldMonth:   s.world.CurrentMonth,
		CrossSeq:     s.world.CrossSeq,
		CrossPending: len(s.world.CrossQueue),
		Silo1Phase:   s.world.Silo1.OrderPhase,
		Silo1Alive:   s.world.Silo1.Alive,
		Flags:        map[string]bool{},
		Attention:    map[string]float64{},
		AgentAP:      map[string]float64{},
	}
	for k, v := range s.world.Flags {
		rec.Flags[k] = v
	}
	for id, v := range s.world.Silo1.Attention {
		rec.Attention[fmt.Sprintf("%d", id)] = v
	}
	for _, silo := range s.silos {
		snap := siloSnapshot{
			ID:            silo.ID,
			Population:    silo.TotalPopulation,
			Rebellion:     silo.Rebellion,
			Cohesion:      silo.Cohesion,
			Legitimacy:    silo.Legitimacy,
			HistoryBurden: silo.HistoryBurden,
			SafeguardRisk: silo.SafeguardRisk,
			Year:          silo.CurrentYear,
			Month:         silo.CurrentMonth,
			Ending:        "NO_ENDING",
			Resources:     map[string]int64{},
			Fires:         map[string]int{},
		}
		if silo.VictoryStatus != nil {
			snap.Ending = silo.VictoryStatus.Type
		}
		for _, r := range silo.Resources {
			snap.Resources[r.Type] = int64(r.Amount)
		}
		for _, entry := range s.eventLogs {
			if entry.SiloID != silo.ID {
				continue
			}
			key := entry.EventID
			if entry.Category != "" {
				key = entry.Category + ":" + entry.EventID
			}
			snap.Fires[key]++
		}
		for _, resident := range silo.Residents {
			if !residentHasTag(&resident, silo18NamedTag) {
				continue
			}
			if snap.NamedAlive == nil {
				snap.NamedAlive = map[string]bool{}
			}
			snap.NamedAlive[resident.Name] = resident.Alive
		}
		rec.Silos = append(rec.Silos, snap)
		if agent, ok := s.agents[silo.ID]; ok {
			rec.AgentAP[fmt.Sprintf("%d", silo.ID)] = agent.ActionPoints
		}
	}
	sort.Slice(rec.Silos, func(i, j int) bool { return rec.Silos[i].ID < rec.Silos[j].ID })
	return rec
}

// runTwoSiloGame 跑一局双堡游戏 (可选在 persistAt 月落库并 Resume)。
func runTwoSiloGame(t *testing.T, seed int64, maxMonths, persistAt int) twoSiloRecord {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "twosilo.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(twoSiloRequest(seed)); err != nil {
		t.Fatalf("create two-silo game seed %d: %v", seed, err)
	}
	months := 0
	for m := 1; m <= maxMonths; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("seed %d pass time month %d: %v", seed, m, err)
		}
		months = m
		if persistAt > 0 && m == persistAt {
			if err := svc.persist(); err != nil {
				t.Fatalf("seed %d persist: %v", seed, err)
			}
			fresh := newTwoSiloService(t, dbPath)
			if err := fresh.Resume(); err != nil {
				t.Fatalf("seed %d resume: %v", seed, err)
			}
			svc = fresh
		}
		if svc.gameOver() {
			break
		}
	}
	return svc.snapshotTwoSilo(seed, months)
}

// TestTwoSiloDoubleRunByteIdentical 验收 #2：同种子双跑逐字节一致 (含跨堡投递)。
func TestTwoSiloDoubleRunByteIdentical(t *testing.T) {
	for _, seed := range []int64{7, 8} {
		a := runTwoSiloGame(t, seed, 120, 0)
		b := runTwoSiloGame(t, seed, 120, 0)
		ja, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("marshal a: %v", err)
		}
		jb, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("marshal b: %v", err)
		}
		if string(ja) != string(jb) {
			t.Fatalf("seed %d: two runs diverged\nrun1=%s\nrun2=%s", seed, ja, jb)
		}
		// 跨堡机制必须真的发生过：否则本测试退化为单堡对照。
		if a.CrossSeq == 0 {
			t.Fatalf("seed %d: expected cross-silo events to be enqueued, got CrossSeq=0", seed)
		}
	}
}

// TestTwoSiloCrossPhaseOrdering 验收 #3：跨堡事件在本地结算后、时钟推进前结算，
// 且顺序稳定。观测点：18 号链条 N1 (Y122M2 生效) 投递给玩家堡的首个耦合信号
// silo18_rumor_sheriff (延迟 1 月，due Y122M3)。
func TestTwoSiloCrossPhaseOrdering(t *testing.T) {
	deliveredAt, deliveredTimestamp, clockAfterDelivery := runCrossDeliveryProbe(t, "ordering.db")
	if deliveredAt < 0 {
		t.Fatalf("silo18_rumor_sheriff was never delivered within 12 months")
	}
	// 时钟推进在跨堡结算之后：投递时记录的时间戳必须严格早于当前世界时钟。
	nowTimestamp := gameTimestamp(clockAfterDelivery.CurrentYear, clockAfterDelivery.CurrentMonth)
	if deliveredTimestamp >= nowTimestamp {
		t.Fatalf("cross event timestamp %d not before advanced clock %d — cross phase must run before clock advance",
			deliveredTimestamp, nowTimestamp)
	}
	// 顺序稳定：再跑一局同种子，投递月与时间戳必须一致。
	deliveredAt2, deliveredTimestamp2, _ := runCrossDeliveryProbe(t, "ordering2.db")
	if deliveredAt != deliveredAt2 || deliveredTimestamp != deliveredTimestamp2 {
		t.Fatalf("cross delivery order unstable: run1(month=%d ts=%d) run2(month=%d ts=%d)",
			deliveredAt, deliveredTimestamp, deliveredAt2, deliveredTimestamp2)
	}
}

// runCrossDeliveryProbe 跑一局并记录首个链条耦合信号在玩家堡的投递时刻。
func runCrossDeliveryProbe(t *testing.T, dbFile string) (int, int, model.World) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), dbFile)
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(twoSiloRequest(7)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	player := svc.siloByID(ActiveSiloID)
	for m := 1; m <= 12; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", m, err)
		}
		if state, ok := svc.contentStates[player.ID]["special:silo18_rumor_sheriff"]; ok && state.Triggered {
			return m, state.LastTriggeredTimestamp, *svc.world
		}
	}
	return -1, -1, model.World{}
}

// TestTwoSiloMetricIsolation 验收 #4：一堡指标变化不渗到另一堡。
func TestTwoSiloMetricIsolation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "isolation.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(twoSiloRequest(7)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	player := svc.siloByID(ActiveSiloID)
	neighbor := svc.siloByID(ActiveSiloID + 1)

	// 直接对邻堡施加一个指标变化 (模拟跨堡效果落到邻堡)。
	before := player.Rebellion
	sc := svc.scopeContextFor(neighbor)
	engine.ApplyContentEffectsScoped([]engine.ContentEffect{
		{Type: "silo_metric_delta", Metric: "rebellion", Value: 0.3},
	}, sc)
	svc.flushScopeResults(sc, nil)

	if neighbor.Rebellion < 0.3-1e-9 {
		t.Fatalf("expected neighbor rebellion to rise to >=0.3, got %v", neighbor.Rebellion)
	}
	if player.Rebellion != before {
		t.Fatalf("player rebellion leaked: before=%v after=%v", before, player.Rebellion)
	}

	// 反向：对玩家堡施加效果，邻堡不受影响。
	neighborBefore := neighbor.Cohesion
	sc2 := svc.scopeContextFor(player)
	engine.ApplyContentEffectsScoped([]engine.ContentEffect{
		{Type: "silo_metric_delta", Metric: "cohesion", Value: -0.2},
	}, sc2)
	svc.flushScopeResults(sc2, nil)
	if player.Cohesion >= 1.0 {
		t.Fatalf("expected player cohesion to drop, got %v", player.Cohesion)
	}
	if neighbor.Cohesion != neighborBefore {
		t.Fatalf("neighbor cohesion leaked: before=%v after=%v", neighborBefore, neighbor.Cohesion)
	}
}

// TestTwoSiloSaveResumeWithCrossQueue 验收 #5：两堡 + 跨堡队列落库并正确 Resume。
func TestTwoSiloSaveResumeWithCrossQueue(t *testing.T) {
	// 第 2 月末落库并 Resume：N1 刚生效，其耦合信号 (延迟 1 月) 正在队列在途，
	// 验证跨堡队列的落库/读档与链条在恢复后继续成立。
	rec := runTwoSiloGame(t, 7, 60, 2)
	if rec.CrossSeq == 0 {
		t.Fatalf("expected cross activity after resume, got CrossSeq=0")
	}
	playerGotRumor := false
	chainFiredOnSilo18 := false
	for _, silo := range rec.Silos {
		if silo.ID == ActiveSiloID && silo.Fires["special:silo18_rumor_sheriff"] > 0 {
			playerGotRumor = true
		}
		if silo.ID == ActiveSiloID+1 && silo.Fires["history:silo18_chain_N1"] > 0 {
			chainFiredOnSilo18 = true
		}
	}
	if !playerGotRumor {
		t.Fatalf("expected silo18_rumor_sheriff to be delivered after resume; silos=%+v", rec.Silos)
	}
	if !chainFiredOnSilo18 {
		t.Fatalf("expected chain node N1 recorded on silo 18 after resume; silos=%+v", rec.Silos)
	}
}

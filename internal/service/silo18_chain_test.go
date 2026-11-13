package service

// Silo 18 原著事件链验收测试：
//   - 逐节点实测触发表 (预期窗口 vs 实测触发月)
//   - 40 号侧耦合信号实际发生且有因果来源
//   - 具名人物生死由链条节点权威设定
//   - 18 号身份与开局状态
// 单堡逐位不变由 parity harness 前后对照保证；双跑逐字节由
// TestTwoSiloDoubleRunByteIdentical (含具名人物生死态) 保证。

import (
	"path/filepath"
	"testing"

	"silo40/internal/model"
)

// silo18ExpectedFire 节点预期触发时钟 (设计 §2.2 映射表)。
type silo18ExpectedFire struct {
	NodeID string
	Year   int
	Month  int
}

var silo18ExpectedFires = []silo18ExpectedFire{
	{"N1", 122, 2},
	{"N2", 122, 5},
	{"N3", 123, 3},
	{"N4", 124, 2},
	{"N5", 125, 1},
	{"N6", 126, 1},
	{"N7", 127, 3},
	{"N8", 128, 6},
	{"N9", 130, 1},
}

// runSilo18FullGame 跑一局双堡到 Y130M1 之后 (96 月窗口 + 余量)。
// 玩家堡中途出结局也不停：世界时钟与链条继续推进 (链条不依赖玩家堡存活)。
func runSilo18FullGame(t *testing.T, seed int64, months int) *GameService {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "silo18chain.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(seed)); err != nil {
		t.Fatalf("create game seed %d: %v", seed, err)
	}
	for m := 1; m <= months; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("seed %d pass time month %d: %v", seed, m, err)
		}
	}
	return svc
}

// TestSilo18ChainTriggerTable 验收：逐节点实测触发表。全部节点必须恰好在其
// 预期月份生效 (时间戳强制触发)，不提前、不滞后、不重复。
func TestSilo18ChainTriggerTable(t *testing.T) {
	svc := runSilo18FullGame(t, 7, 100)
	silo18 := svc.silo18Canon()
	if silo18 == nil {
		t.Fatalf("expected silo 18 in two-silo game")
	}

	t.Logf("node | expected | actual timestamp | status")
	for _, expected := range silo18ExpectedFires {
		wantTS := gameTimestamp(expected.Year, expected.Month)
		state, ok := svc.contentStates[silo18.ID][silo18ChainStateKey(expected.NodeID)]
		if !ok || !state.Triggered {
			t.Fatalf("node %s never fired (expected at Y%dM%d)", expected.NodeID, expected.Year, expected.Month)
		}
		if state.TriggerCount != 1 {
			t.Fatalf("node %s fired %d times, want exactly 1", expected.NodeID, state.TriggerCount)
		}
		status := "OK"
		if state.LastTriggeredTimestamp != wantTS {
			status = "MISMATCH"
		}
		t.Logf("%-4s | Y%dM%-2d | %-16d | %s", expected.NodeID, expected.Year, expected.Month,
			state.LastTriggeredTimestamp, status)
		if state.LastTriggeredTimestamp != wantTS {
			t.Fatalf("node %s fired at timestamp %d, want %d (Y%dM%d)",
				expected.NodeID, state.LastTriggeredTimestamp, wantTS, expected.Year, expected.Month)
		}
	}
	// N9 世界事实：一号地堡覆灭。
	if svc.world.Silo1.Alive {
		t.Fatalf("expected Silo 1 destroyed after N9 (Y130M1)")
	}
}

// runSilo18ObserverGame keeps the station's signal probe active after local
// endings. It is test-only and does not alter production ending behavior.
func runSilo18ObserverGame(t *testing.T, seed int64, months int) *GameService {
	t.Helper()
	svc := newTwoSiloService(t, filepath.Join(t.TempDir(), "silo18observer.db"))
	if _, err := svc.CreateGame(threeSiloRequest(seed)); err != nil {
		t.Fatalf("create observer game: %v", err)
	}
	for m := 1; m <= months; m++ {
		svc.silo.VictoryStatus = nil
		svc.silo.Rebellion = 0
		svc.silo.SafeguardRisk = 0
		svc.agent.SuspicionLevel = 0
		for i := range svc.silo.Professions {
			svc.silo.Professions[i].KnownFragments = nil
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("observer pass time month %d: %v", m, err)
		}
	}
	return svc
}

// TestSilo18CouplingSignalsObserved verifies that every coupling signal reaches
// the assigned station with its canon-node cause, independent of weighted endings.
func TestSilo18CouplingSignalsObserved(t *testing.T) {
	svc := runSilo18ObserverGame(t, 10, 100)
	player := svc.siloByID(ActiveSiloID)

	// 因果表：耦合事件 -> 来源节点 (设计 §2.2 第 6 列)。
	couplings := []struct {
		EventKey string
		Source   string
	}{
		{"special:silo18_rumor_sheriff", "N1"},
		{"special:silo18_rumor_it_takeover", "N2"},
		{"special:silo18_refuser_crossed", "N3"},
		{"special:silo18_rumor_uprising", "N4"},
		{"special:silo18_rumor_regime", "N5"},
		{"special:silo18_digging_signal", "N6"},
		{"special:silo18_green_reconnaissance", "N7"},
		{"special:silo18_silence", "N8"},
	}
	for _, c := range couplings {
		state, ok := svc.contentStates[player.ID][c.EventKey]
		if !ok || !state.Triggered {
			t.Fatalf("coupling signal %s (source: %s) was never observed at the assigned station",
				c.EventKey, c.Source)
		}
		t.Logf("observed %-38s source=%s timestamp=%d", c.EventKey, c.Source, state.LastTriggeredTimestamp)
	}

	// N8 后迷雾视图必须显示 18 号覆灭。
	views := svc.buildRumorViews()
	var silo18View *model.RumorView
	for i := range views {
		if views[i].SiloID == svc.silo18Canon().ID {
			silo18View = &views[i]
		}
	}
	if silo18View == nil || silo18View.Status != "destroyed" {
		t.Fatalf("expected silo 18 rumor status destroyed after N8, got %+v", silo18View)
	}
	// 终局信号复用既有 silo1_destroyed_signal (Y130M1)，不另开新信号。
	if state, ok := svc.contentStates[player.ID]["histories:silo1_destroyed_signal"]; !ok || !state.Triggered {
		t.Fatalf("expected existing silo1_destroyed_signal to fire on player silo at Y130M1")
	}
}

// TestSilo18NamedCharactersFate 验收：具名人物是真实个体，生死只由链条节点
// 权威设定 (Holston=N1, Jahns/Marnes=N2, George=N4, Lukas=N8)；AI 群演与
// 日常模拟不掌其生死。
func TestSilo18NamedCharactersFate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "namedfate.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(7)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	silo18 := svc.silo18Canon()
	if silo18 == nil {
		t.Fatalf("expected silo 18")
	}

	// 开局：全部具名人物存活，且为带标签的真实 Resident。
	wantCast := map[string]bool{
		"Jahns": true, "Marnes": true, "Bernard": true, "Lukas Kyle": true,
		"Holston": true, "Juliette": true, "Walker": true, "Knox": true,
		"Shirly": true, "George": true,
	}
	for name := range wantCast {
		if r := namedResident(silo18, name); r == nil || !r.Alive {
			t.Fatalf("named character %s missing or dead at game start", name)
		}
	}

	// 逐月推进，记录每个具名人物的死亡月份：必须恰好等于其链条节点月份。
	// 节点在阶段 0c 生效 (时钟尚未推进)，因此检测到时钟已 +1，需减回。
	deathMonth := map[string]int{}
	for m := 1; m <= 100; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", m, err)
		}
		nowIndex := (svc.world.CurrentYear-122)*12 + svc.world.CurrentMonth
		for name := range wantCast {
			if _, recorded := deathMonth[name]; recorded {
				continue
			}
			if r := namedResident(silo18, name); r != nil && !r.Alive {
				deathMonth[name] = nowIndex - 1
			}
		}
	}

	// 死亡月份 = 节点生效月 (生效发生在阶段 0c，时钟尚未推进)。
	wantDeaths := map[string]int{
		"Holston":    monthIndex(122, 2), // N1
		"Jahns":      monthIndex(122, 5), // N2
		"Marnes":     monthIndex(122, 5), // N2
		"George":     monthIndex(124, 2), // N4
		"Lukas Kyle": monthIndex(128, 6), // N8
	}
	for name, want := range wantDeaths {
		got, ok := deathMonth[name]
		if !ok {
			t.Fatalf("named character %s never died, want death at node month", name)
		}
		if got != want {
			t.Fatalf("named character %s died at month index %d, want %d", name, got, want)
		}
	}
	// 其余具名人物活到终局 (生死不受 AI/模拟支配)。
	for name := range wantCast {
		if _, isDeath := wantDeaths[name]; isDeath {
			continue
		}
		if _, died := deathMonth[name]; died {
			t.Fatalf("named character %s died outside the chain (AI/sim must not kill named characters)", name)
		}
	}
	// 链条角色标签权威覆写。
	if r := namedResident(silo18, "Bernard"); !residentHasTag(r, "acting_mayor") {
		t.Fatalf("Bernard missing acting_mayor tag after N2")
	}
	if r := namedResident(silo18, "Juliette"); !residentHasTag(r, "mayor") || !residentHasTag(r, "crossed_outside") {
		t.Fatalf("Juliette missing chain tags after N3/N5")
	}
	if r := namedResident(silo18, "Lukas Kyle"); !residentHasTag(r, "it_head") {
		t.Fatalf("Lukas Kyle missing it_head tag after N5")
	}
}

// monthIndex 绝对月序号 (Y122M1 = 1)。
func monthIndex(year, month int) int {
	return (year-122)*12 + month
}

// TestSilo18IdentityAtCreate 验收：第二堡固定为 18 号，带正典标记、前史计量
// 与具名人物；单堡配置不产生 18 号。
func TestSilo18IdentityAtCreate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(7)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	silo18 := svc.silo18Canon()
	if silo18 == nil {
		t.Fatalf("expected Silo 18 in the three-silo world")
	}
	if silo18.Name != "18 号地堡" {
		t.Fatalf("second silo name = %q, want 18 号地堡", silo18.Name)
	}
	if !residentSiloHasTrait(silo18, silo18CanonTrait) {
		t.Fatalf("silo 18 missing canon trait")
	}
	// 前史承载：大革命传说带来的历史包袱。
	if silo18.HistoryBurden < 0.5 {
		t.Fatalf("silo 18 history burden %v too low for pre-history load", silo18.HistoryBurden)
	}
	if namedResident(silo18, "Juliette") == nil || namedResident(silo18, "Bernard") == nil {
		t.Fatalf("silo 18 missing named cast")
	}
	// 玩家堡不受影响：无具名人物、无正典标记。
	player := svc.siloByID(ActiveSiloID)
	if residentSiloHasTrait(player, silo18CanonTrait) {
		t.Fatalf("player silo must not carry canon trait")
	}
	for i := range player.Residents {
		if residentHasTag(&player.Residents[i], silo18NamedTag) {
			t.Fatalf("player silo must not have named canon residents")
		}
	}

	// 单堡配置：无 18 号，链条机制完全休眠。
	dbPath2 := filepath.Join(t.TempDir(), "identity-single.db")
	single := newTwoSiloService(t, dbPath2)
	if _, err := single.CreateGame(model.CreateGameRequest{
		SiloName: "Solo", AgentName: "A", Profession: "IT", Seed: 7,
	}); err != nil {
		t.Fatalf("create single game: %v", err)
	}
	if single.silo18Canon() != nil {
		t.Fatalf("single-silo game must not have silo 18")
	}
}

// TestSilo18ChainSurvivesResume 验收：链条生效记录随 v3 存档持久化，读档后
// 不重复生效、未到期节点继续按时生效 (存档格式不变)。
func TestSilo18ChainSurvivesResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "chainresume.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(7)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	// 跑到 N3 生效后落库 (N3 = Y123M3 = 第 15 月；落库时时钟 Y123M4)。
	for m := 1; m <= 15; m++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", m, err)
		}
	}
	silo18 := svc.silo18Canon()
	for _, node := range []string{"N1", "N2", "N3"} {
		if !svc.silo18NodeFired(silo18.ID, node) {
			t.Fatalf("node %s should have fired before persist", node)
		}
	}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	silo18Fresh := fresh.silo18Canon()
	if silo18Fresh == nil {
		t.Fatalf("silo 18 missing after resume")
	}
	// 已生效节点不重复：再跑 90 月 (越过 N9)，TriggerCount 保持 1。
	for m := 1; m <= 90; m++ {
		if _, err := fresh.PassTime(1); err != nil {
			t.Fatalf("post-resume pass time month %d: %v", m, err)
		}
	}
	for _, node := range silo18ExpectedFires {
		state, ok := fresh.contentStates[silo18Fresh.ID][silo18ChainStateKey(node.NodeID)]
		if !ok || !state.Triggered {
			t.Fatalf("node %s not fired after resume run", node.NodeID)
		}
		if state.TriggerCount != 1 {
			t.Fatalf("node %s fired %d times after resume, want 1 (no double-apply)", node.NodeID, state.TriggerCount)
		}
	}
	// 具名人物生死在读档后保持一致。
	if r := namedResident(silo18Fresh, "Holston"); r == nil || r.Alive {
		t.Fatalf("Holston must stay dead after resume")
	}
	if r := namedResident(silo18Fresh, "Juliette"); r == nil || !r.Alive {
		t.Fatalf("Juliette must stay alive after resume")
	}
}

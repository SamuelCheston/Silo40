package service

import (
	"path/filepath"
	"strings"
	"testing"

	"silo40/internal/model"
)

// TestNeighborStoryPerceptionRules locks 邻堡感知过滤：日常琐碎传不出去，
// 危机与一次性转折以谣言版本传出，正典链条与外部信号原样放行。
func TestNeighborStoryPerceptionRules(t *testing.T) {
	svc := chainTestService()
	svc.world = &model.World{PlayerSiloID: 1}
	svc.agent = &model.Agent{Profession: "IT"}
	neighbor := &model.Silo{ID: 3, Name: "18 号地堡"}

	routine := model.StoryEvent{ID: "shift_grumble", Category: "special", FireMode: "REPEATABLE", Title: "轮班怨气"}
	if _, ok := svc.perceiveNeighborStory(routine, neighbor); ok {
		t.Fatal("repeatable neighbor routine must not reach the player")
	}

	crisis := model.StoryEvent{ID: "water_leak", Category: "crisis", FireMode: "REPEATABLE", Title: "供水管线泄漏", Description: "底层水管泄漏。"}
	perceived, ok := svc.perceiveNeighborStory(crisis, neighbor)
	if !ok {
		t.Fatal("neighbor crisis must reach the player")
	}
	if !strings.HasPrefix(perceived.Title, "传言：") || !strings.Contains(perceived.Description, "线路杂音") {
		t.Fatalf("crisis must be reframed as a rumor: %+v", perceived)
	}

	turning := model.StoryEvent{ID: "archive_purge_order", Category: "special", FireMode: "ONCE", Title: "档案清洗令"}
	if _, ok := svc.perceiveNeighborStory(turning, neighbor); !ok {
		t.Fatal("one-off neighbor turning point must reach the player")
	}

	signal := model.StoryEvent{ID: "silo1_department_signal", Category: "special", Type: "EXTERNAL", Title: "截获：1 号地堡部门信号"}
	perceived, ok = svc.perceiveNeighborStory(signal, neighbor)
	if !ok || perceived.Title != signal.Title {
		t.Fatalf("intercepted signal must pass through unreframed: %+v", perceived)
	}

	playerSilo := &model.Silo{ID: 1, Name: "16 号地堡"}
	if perceived, ok = svc.perceiveNeighborStory(routine, playerSilo); !ok || perceived.Title != routine.Title {
		t.Fatalf("player silo stories must pass through untouched: %+v", perceived)
	}
}

// TestPlayerFeedCarriesNoNeighborRoutine locks 每堡自己的主题：跑一段真实
// 三堡月份后，玩家情报记录里的邻堡条目只能是谣言版本或玩家视角产品
// (截获信号/正典链条/行动回执)，不能有邻堡日常原文。
func TestPlayerFeedCarriesNoNeighborRoutine(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "perception.db"))
	if _, err := svc.CreateGame(threeSiloRequest(2026)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	for month := 0; month < 18; month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("pass time month %d: %v", month, err)
		}
	}
	playerID := svc.world.PlayerSiloID
	checked := 0
	for _, entry := range svc.eventLogs {
		if entry.SiloID == playerID {
			continue
		}
		checked++
		playerView := strings.HasPrefix(entry.Title, "传言：") ||
			entry.Category == "history" || entry.Type == "EXTERNAL" ||
			strings.HasPrefix(entry.Source, "INTEL:") || strings.HasPrefix(entry.Source, "SILO18_CHAIN:") ||
			strings.HasPrefix(entry.Source, "OPERATION_REPORT:") || strings.HasPrefix(entry.Source, "CROSS_EFFECT:")
		if !playerView {
			t.Fatalf("neighbor routine leaked into the player feed: Y%dM%d %s (%s/%s)",
				entry.Year, entry.Month, entry.Title, entry.Category, entry.Source)
		}
	}
	if checked == 0 {
		t.Fatal("expected at least one neighbor entry in the player feed over 18 months")
	}
}

// TestSameMonthNeighborCopiesMerge locks 削减同月堆叠：同月同一事件同一
// 标题的多条合并为一条带来源堡列表的汇总条目；亲身经历与谣言版本标题
// 不同，分开保留；跨堡同名信号 (玩家与邻堡同收) 也合并。
func TestSameMonthNeighborCopiesMerge(t *testing.T) {
	stories := []model.StoryEvent{
		{ID: "water_leak", Title: "供水管线泄漏", SiloID: 1, SiloName: "16 号地堡"},
		{ID: "water_leak", Title: "传言：供水管线泄漏", SiloID: 2, SiloName: "1 号地堡"},
		{ID: "water_leak", Title: "传言：供水管线泄漏", SiloID: 3, SiloName: "18 号地堡"},
		{ID: "food_poisoning", Title: "传言：群体性食物中毒", SiloID: 3, SiloName: "18 号地堡"},
		{ID: "silo18_refuser_crossed", Title: "强信号：拒洁者走出了视野", SiloID: 1, SiloName: "16 号地堡"},
		{ID: "silo18_refuser_crossed", Title: "强信号：拒洁者走出了视野", SiloID: 2, SiloName: "1 号地堡"},
	}
	merged := mergeNeighborStoryCopies(stories)
	if len(merged) != 4 {
		t.Fatalf("expected 4 entries after merge, got %d: %+v", len(merged), merged)
	}
	if merged[0].Title != "供水管线泄漏" || merged[0].SiloName != "16 号地堡" {
		t.Fatalf("player silo entry must stay untouched: %+v", merged[0])
	}
	if merged[1].Title != "传言：供水管线泄漏（1 号地堡、18 号地堡）" || merged[1].SiloName != "1 号地堡、18 号地堡" {
		t.Fatalf("neighbor copies must merge into one summary entry: %+v", merged[1])
	}
	if merged[2].Title != "传言：群体性食物中毒" {
		t.Fatalf("single neighbor entry must not gain a suffix: %+v", merged[2])
	}
	if merged[3].Title != "强信号：拒洁者走出了视野（1 号地堡、16 号地堡）" {
		t.Fatalf("identical cross-silo signals must merge: %+v", merged[3])
	}
}

// TestEventHistoryMergesNeighborCopies locks 读取侧合并：持久化历史里同年
// 同月同一事件的邻堡条目并入首条。
func TestEventHistoryMergesNeighborCopies(t *testing.T) {
	svc := chainTestService()
	svc.world = &model.World{PlayerSiloID: 1}
	svc.silos = append(svc.silos,
		&model.Silo{ID: 2, Name: "1 号地堡"},
		&model.Silo{ID: 3, Name: "18 号地堡"},
	)
	mk := func(siloID uint, title string) model.StoryEventLog {
		return model.StoryEventLog{SiloID: siloID, Year: 123, Month: 3, EventID: "archive_purge_order", Title: title}
	}
	svc.eventLogs = []model.StoryEventLog{
		mk(1, "档案清洗令"),
		mk(2, "传言：档案清洗令"),
		mk(3, "传言：档案清洗令"),
		{SiloID: 3, Year: 123, Month: 4, EventID: "other", Title: "别的传闻"},
	}
	history := svc.eventHistory(0)
	if len(history) != 3 {
		t.Fatalf("expected 3 history entries after merge, got %d: %+v", len(history), history)
	}
	// 最新在前：M4 条目最先；M3 的三条 (玩家＋两邻堡) 合并后剩两条。
	if history[0].Title != "别的传闻" {
		t.Fatalf("unrelated entry must pass through: %+v", history[0])
	}
	if history[1].Title != "传言：档案清洗令（1 号地堡、18 号地堡）" {
		t.Fatalf("neighbor history copies must merge: %+v", history[1])
	}
	if history[2].Title != "档案清洗令" || history[2].SiloID != 1 {
		t.Fatalf("player entry must stay untouched: %+v", history[2])
	}
}

package service

import (
	"strings"
	"testing"

	"silo40/internal/model"
)

// outsideFixture 一个未终局的驻地 + 已出发存续群体 (堡外维持激活)。
func outsideFixture() (*GameService, *model.Silo) {
	station := &model.Silo{ID: 1, Name: "驻地", TotalPopulation: 5000, CurrentYear: 126, CurrentMonth: 3}
	svc := &GameService{
		world: &model.World{PlayerSiloID: 1, Silo1: model.Silo1State{Alive: true, Attention: map[uint]float64{}}},
		silo:  station,
		silos: []*model.Silo{station},
	}
	station.EscapedGroup = model.EscapeGroup{
		Initialized: true, Population: 120, Departed: 150, Surviving: true,
		DepartedYear: 126, DepartedMonth: 3,
		Outside: model.OutsideState{
			Active: true, SuppliesLeft: 300, SuitIntegrity: 0.9,
			RouteSafety: 0.8, Tracking: model.TrackingNone,
			Leaders: 3, Artisans: 6, Elders: 14,
		},
	}
	return svc, station
}

// 三个维持侧面逐月消耗：补给、防护服完整性、路线安全各自下降，月数累积，
// 月度日志行可读 (玩家逐月在读什么)。
func TestOutsideSustenanceConsumes(t *testing.T) {
	svc, station := outsideFixture()
	g := &station.EscapedGroup
	before := *g

	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runOutsideSustenance(&logs, &stories)

	out := g.Outside
	if out.MonthsOut != 1 {
		t.Fatalf("months out = %d, want 1", out.MonthsOut)
	}
	if out.SuppliesLeft >= before.Outside.SuppliesLeft {
		t.Fatalf("supplies must deplete: %v -> %v", before.Outside.SuppliesLeft, out.SuppliesLeft)
	}
	if out.SuitIntegrity >= before.Outside.SuitIntegrity {
		t.Fatalf("suit integrity must wear: %v -> %v", before.Outside.SuitIntegrity, out.SuitIntegrity)
	}
	if out.RouteSafety >= before.Outside.RouteSafety {
		t.Fatalf("route safety must degrade: %v -> %v", before.Outside.RouteSafety, out.RouteSafety)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "[Outside]") || !strings.Contains(logs[0], "补给约可支撑") {
		t.Fatalf("monthly outside line missing or malformed: %v", logs)
	}
	if !g.Surviving || !out.Active {
		t.Fatalf("healthy first month must keep the group alive: %+v", g)
	}
}

// 三种死法各自成立：耗尽、冻死、被地形困住。覆没原因写事实，群体置不存续。
func TestOutsideFailureModes(t *testing.T) {
	runToDemise := func(mutate func(*model.Silo), maxMonths int) *model.EscapeGroup {
		svc, station := outsideFixture()
		station.CurrentMonth = 6 // 夏季起步，避免冻死抢跑
		mutate(station)
		logs, stories := []string{}, []model.StoryEvent{}
		for i := 0; i < maxMonths && station.EscapedGroup.Surviving; i++ {
			svc.runOutsideSustenance(&logs, &stories)
		}
		return &station.EscapedGroup
	}

	// 耗尽：补给极少，路线安全不足以觅食补差。
	exhausted := runToDemise(func(s *model.Silo) {
		s.EscapedGroup.Outside.SuppliesLeft = 30
		s.EscapedGroup.Outside.RouteSafety = 0.2
	}, 60)
	if exhausted.Surviving || exhausted.Outside.DemiseCause != model.DemiseExhaust {
		t.Fatalf("starvation path must end in exhaust, got %+v", exhausted.Outside)
	}

	// 冻死：防护服完整性低 + 深冬出发。
	frozen := runToDemise(func(s *model.Silo) {
		s.CurrentMonth = 1 // 深冬：严寒 0.9
		s.EscapedGroup.Outside.SuitIntegrity = 0.3
		s.EscapedGroup.Outside.SuppliesLeft = 5000
	}, 60)
	if frozen.Surviving || frozen.Outside.DemiseCause != model.DemiseFreeze {
		t.Fatalf("cold path must end in freeze, got %+v", frozen.Outside)
	}

	// 被地形困住：路线安全坍塌，补给充足也走不出去。
	trapped := runToDemise(func(s *model.Silo) {
		s.EscapedGroup.Outside.RouteSafety = 0.1
		s.EscapedGroup.Outside.SuppliesLeft = 5000
	}, 60)
	if trapped.Surviving || trapped.Outside.DemiseCause != model.DemiseTrap {
		t.Fatalf("terrain path must end in trap, got %+v", trapped.Outside)
	}
}

// 追踪四档：热度定档、档位改变选择空间 (弃置补给/损耗/觅食封锁)、
// 距离增长让热度衰减、档位可回落。追踪不是必然失败。
func TestOutsideTrackingTiers(t *testing.T) {
	// 热度定档。
	for _, tc := range []struct {
		heat float64
		want string
	}{
		{0.1, model.TrackingNone},
		{0.3, model.TrackingAware},
		{0.6, model.TrackingTrailing},
		{0.9, model.TrackingPursuit},
	} {
		if got := outsideTrackingTier(tc.heat); got != tc.want {
			t.Fatalf("tier(%.1f) = %q, want %q", tc.heat, got, tc.want)
		}
	}

	// 1 号注意力抬升热度：察觉 → 咬住痕迹 (弃置补给) → 追捕 (逐月损耗)。
	svc, station := outsideFixture()
	station.EscapedGroup.Outside.TrackingBase = 0.3
	station.EscapedGroup.Outside.SuppliesLeft = 5000
	station.EscapedGroup.Outside.RouteSafety = 0.9
	svc.world.Silo1.Attention[station.ID] = 0.4 // heat = 0.3+0.24-0.02 = 0.52 → trailing
	suppliesBefore := station.EscapedGroup.Outside.SuppliesLeft
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runOutsideSustenance(&logs, &stories)
	out := station.EscapedGroup.Outside
	if out.Tracking != model.TrackingTrailing {
		t.Fatalf("tracking = %q, want trailing under attention 0.4", out.Tracking)
	}
	// 弃置补给发生在进入 trailing 的瞬间 (先于当月消耗)：余量必须明显下降。
	if out.SuppliesLeft >= suppliesBefore {
		t.Fatalf("entering trailing must drop supplies: %v -> %v", suppliesBefore, out.SuppliesLeft)
	}

	// 追捕：注意力继续抬升 → 逐月损耗，但追踪不是必然失败 (群体仍在存续)。
	svc.world.Silo1.Attention[station.ID] = 0.9 // heat = 0.3+0.54-0.02 = 0.82 → pursuit
	popBefore := station.EscapedGroup.Population
	svc.runOutsideSustenance(&logs, &stories)
	out = station.EscapedGroup.Outside
	if out.Tracking != model.TrackingPursuit {
		t.Fatalf("tracking = %q, want pursuit under attention 0.9", out.Tracking)
	}
	if station.EscapedGroup.Population >= popBefore {
		t.Fatalf("pursuit must cost people: %d -> %d", popBefore, station.EscapedGroup.Population)
	}
	if !station.EscapedGroup.Surviving {
		t.Fatal("one pursuit month must not annihilate a healthy group — tracking changes choice space, not fate")
	}

	// 距离增长：注意力归零后热度随月数衰减，档位回落到没人注意。
	svc.world.Silo1.Attention[station.ID] = 0
	for i := 0; i < 30 && station.EscapedGroup.Surviving; i++ {
		svc.runOutsideSustenance(&logs, &stories)
	}
	if station.EscapedGroup.Surviving && station.EscapedGroup.Outside.Tracking != model.TrackingNone {
		t.Fatalf("heat must decay with distance: still %q", station.EscapedGroup.Outside.Tracking)
	}
}

// 群体质量：组合事实派生 0-1 质量；进 P2 加权只作玩家贡献项——
// 资格仍只读群体存续，质量不是第二道资格门。
func TestOutsideGroupQualityWeightsNotGates(t *testing.T) {
	// 组合派生：有人有组织 > 乌合之众 > 无人。
	organized := &model.EscapeGroup{Population: 120, Outside: model.OutsideState{Leaders: 4, Artisans: 8, Elders: 10, Wounded: 5}}
	crowd := &model.EscapeGroup{Population: 120}
	if q := outsideGroupQuality(organized); q <= 0 || q > 1 {
		t.Fatalf("quality out of range: %v", q)
	}
	if outsideGroupQuality(organized) <= outsideGroupQuality(crowd) {
		t.Fatalf("organized group must outquality a crowd: %v vs %v",
			outsideGroupQuality(organized), outsideGroupQuality(crowd))
	}
	if outsideGroupQuality(&model.EscapeGroup{}) != 0 {
		t.Fatal("empty group quality must be 0")
	}

	// 加权：质量进 REBELLION 的 P2 分数，但只影响分数。
	newDecision := func(qualityGroup model.EscapeGroup) *model.EndingDecision {
		svc, station, _ := weightedEndingFixture("REBELLION")
		station.TotalPopulation = 5000
		station.EscapedGroup = qualityGroup
		svc.agent.Ambition = 1
		svc.agent.Influence = 1
		return svc.resolveWeightedEnding(station, svc.agent)
	}
	group := model.EscapeGroup{
		Initialized: true, Population: 120, Departed: 150, Surviving: true,
		Outside: model.OutsideState{Active: true, Leaders: 4, Artisans: 8, Elders: 10, MonthsOut: 6},
	}
	withQuality := newDecision(group)
	if withQuality.Selected != endingRebellion {
		t.Fatalf("expected REBELLION, got %+v", withQuality)
	}
	crowdGroup := group
	crowdGroup.Outside = model.OutsideState{Active: true, MonthsOut: 6}
	withoutQuality := newDecision(crowdGroup)
	if withQuality.Scores[endingRebellion] <= withoutQuality.Scores[endingRebellion] {
		t.Fatalf("quality must raise the REBELLION score: %v vs %v",
			withQuality.Scores[endingRebellion], withoutQuality.Scores[endingRebellion])
	}
	// 资格不读质量：无群体时质量再高也不进 pool。
	noGroup := model.EscapeGroup{Outside: model.OutsideState{Leaders: 99, Artisans: 99}}
	decision := newDecision(noGroup)
	if _, excluded := decision.Excluded[endingRebellion]; !excluded {
		t.Fatalf("quality without a surviving group must not qualify REBELLION: %+v", decision.Pool)
	}
}

// 确定性：堡外维持不新增 RNG 取样——同一状态跑两次逐月一致。
func TestOutsideSustenanceDeterministic(t *testing.T) {
	run := func() []int {
		svc, station := outsideFixture()
		station.EscapedGroup.Outside.SuppliesLeft = 150
		station.EscapedGroup.Outside.RouteSafety = 0.35
		pops := []int{}
		logs, stories := []string{}, []model.StoryEvent{}
		for i := 0; i < 24 && station.EscapedGroup.Surviving; i++ {
			svc.runOutsideSustenance(&logs, &stories)
			pops = append(pops, station.EscapedGroup.Population)
		}
		return pops
	}
	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("runs diverged in length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("month %d diverged: %d vs %d", i+1, a[i], b[i])
		}
	}
}

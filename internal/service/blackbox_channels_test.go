package service

// 缺口通道格 (W6) 测试：22 黑箱学习系列 02 卷 §6 缺口清单的验收面——
// 各通道门控关闭零变化、开启时计数/日志正确、G9 台账键消费与清零语义
// (四类旧键清零、新键保留)、G3 派生账本与 Log 一致、G5 环比阈值、
// G1 的 P_danger 门槛与节流、全程零 RNG。

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// blackBoxChannelsFixture 已启用权衡层与黑箱状态的最小三堡服务：驻地 (ID 1)、
// 1 号 (ID 2, 带 silo1FullTrait)、18 号 (ID 3)。时钟默认 Y122M1。Ledger 空表
// 由被测代码防御性初始化，覆盖存档回读 nil 表路径。
func blackBoxChannelsFixture() (*GameService, *model.Silo, *model.Silo, *model.Silo) {
	station := &model.Silo{ID: 1, Name: "驻地", TotalPopulation: 10000,
		BlackBox: model.Silo1BlackBoxState{Initialized: true}}
	silo1 := &model.Silo{ID: 2, Name: "1 号地堡", Traits: []string{silo1FullTrait}, TotalPopulation: 10000,
		BlackBox: model.Silo1BlackBoxState{Initialized: true}}
	silo18 := &model.Silo{ID: 3, Name: "18 号地堡", TotalPopulation: 10000,
		BlackBox: model.Silo1BlackBoxState{Initialized: true}}
	world := &model.World{
		CurrentYear:  122,
		CurrentMonth: 1,
		PlayerSiloID: 1,
		Silo1:        model.Silo1State{Alive: true, OrderPhase: "LOCAL_SIM"},
		Silo1Stewardship: model.Silo1StewardshipState{
			Initialized: true,
		},
		Flags: map[string]bool{},
	}
	svc := &GameService{
		world: world,
		silos: []*model.Silo{station, silo1, silo18},
		silo:  station,
	}
	return svc, station, silo1, silo18
}

// ===== 门控 =====

// TestBlackBoxChannelsGatedOffZeroChangeZeroRNG 调度器门控 (功能关 / 1 号覆灭)：
// 整个 2b' 早退——即使各通道输入全部「上膛」也零变化、零 RNG。
func TestBlackBoxChannelsGatedOffZeroChangeZeroRNG(t *testing.T) {
	check := func(name string, svc *GameService) {
		t.Helper()
		target := svc.siloByID(3)
		if target == nil {
			t.Fatalf("%s: three-silo game must contain silo 3", name)
		}
		// 上膛：举报、稽查、人口、旧累加键全部处于「一跑就响」的状态。
		target.Professions[0].Beliefs = append(target.Professions[0].Beliefs,
			model.FactBelief{FactID: "Order_Bait", B: 0.95})
		target.BlackBox.Log = append(target.BlackBox.Log, model.BlackBoxLogEntry{
			Year: 122, Month: 1, PDanger: 0.9, ChosenR: 0.4,
		})
		target.SafeguardIncidents = map[string]int{engine.SafeguardIncidentDeviation: 3}

		const seed = 4321
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		logs, stories := []string{}, []model.StoryEvent{}
		svc.runSilo1StewardshipMonth(&logs, &stories)

		if len(logs) != 0 || len(stories) != 0 {
			t.Fatalf("%s: gated-off stewardship must stay silent: logs=%v stories=%v", name, logs, stories)
		}
		if svc.world.Silo1Stewardship.Initialized {
			t.Fatalf("%s: gated-off stewardship must not initialize", name)
		}
		if got := target.SafeguardIncidents; got[bbChannelIncidentReportsReceived] != 0 ||
			got[bbChannelIncidentAuditFact] != 0 || got[bbChannelIncidentCensusAnomaly] != 0 {
			t.Fatalf("%s: gated-off round must not register channel incidents: %+v", name, got)
		}
		if target.SafeguardIncidents[engine.SafeguardIncidentDeviation] != 3 {
			t.Fatalf("%s: gated-off round must not consume legacy incidents: %+v",
				name, target.SafeguardIncidents)
		}
		if ledger := svc.world.Silo1Stewardship.Experiment.Ledger; len(ledger) != 0 {
			t.Fatalf("%s: gated-off round must not write ledger keys: %+v", name, ledger)
		}
		ref := rand.New(rand.NewSource(seed))
		for i := 0; i < 64; i++ {
			if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
				t.Fatalf("%s: gated-off round consumed RNG (draw %d: %v != %v)", name, i, got, want)
			}
		}
	}

	off := newBlackBoxGame(t, 601)
	off.world.Silo1BlackBoxEnabled = false
	check("feature-off", off)

	dead := newBlackBoxGame(t, 602)
	dead.world.Silo1.Alive = false
	check("silo1-dead", dead)
}

// TestBlackBoxChannelsRequiresInitialized 直接调用门控：权衡层未启用 (零值)
// 早退，输入上膛也零变化。
func TestBlackBoxChannelsRequiresInitialized(t *testing.T) {
	svc, _, silo1, silo18 := blackBoxChannelsFixture()
	svc.world.Silo1Stewardship.Initialized = false
	silo18.Professions = []model.Profession{{Name: "Security",
		Beliefs: []model.FactBelief{{FactID: "Order_Bait", B: 0.95}}}}
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{Year: 122, Month: 1, PDanger: 0.9}}
	silo18.SafeguardIncidents = map[string]int{engine.SafeguardIncidentDeviation: 3}

	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)

	if len(logs) != 0 || len(stories) != 0 {
		t.Fatalf("uninitialized channels must stay silent: logs=%v stories=%v", logs, stories)
	}
	if len(silo18.SafeguardIncidents) != 1 || silo18.SafeguardIncidents[engine.SafeguardIncidentDeviation] != 3 {
		t.Fatalf("uninitialized channels must not touch incidents: %+v", silo18.SafeguardIncidents)
	}
	if silo1.SafeguardIncidents != nil || svc.world.Silo1Stewardship.Experiment.Ledger != nil {
		t.Fatal("uninitialized channels must not write any state")
	}
}

// ===== G2 举报管道 =====

// TestBlackBoxChannelsReportsPipeline 开启时：任一部门事实浓度 ≥ QualReport →
// 该堡当月报告送达 +1 + 审计日志行；1 号自身不被扫描；未达标堡不计数。
func TestBlackBoxChannelsReportsPipeline(t *testing.T) {
	svc, station, silo1, silo18 := blackBoxChannelsFixture()
	qualReport := engine.DefaultInformationConfig().QualReport
	silo18.Professions = []model.Profession{{Name: "Security",
		Beliefs: []model.FactBelief{
			{FactID: "Fact_Below", B: qualReport - 0.001},
			{FactID: "Fact_At", B: qualReport}, // 恰达阈值 (≥)
		}}}
	// 1 号自身高信念：观察者不在自己的观测通道里。
	silo1.Professions = []model.Profession{{Name: "Archives",
		Beliefs: []model.FactBelief{{FactID: "Fact_High", B: 0.99}}}}
	// 驻地未达标。
	station.Professions = []model.Profession{{Name: "Supply",
		Beliefs: []model.FactBelief{{FactID: "Fact_Low", B: qualReport - 0.001}}}}

	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)

	if got := silo18.SafeguardIncidents[bbChannelIncidentReportsReceived]; got != 1 {
		t.Fatalf("qualified silo must receive exactly one report, got %d", got)
	}
	if len(stories) != 0 {
		t.Fatalf("channels must not emit story events: %+v", stories)
	}
	found := false
	for _, line := range logs {
		if strings.Contains(line, "silo 3 report received") {
			found = true
		}
	}
	if !found {
		t.Fatalf("report delivery must leave an audit log line: %v", logs)
	}
	if got := silo1.SafeguardIncidents[bbChannelIncidentReportsReceived]; got != 0 {
		t.Fatalf("silo 1 itself must not be scanned, got %d", got)
	}
	if got := station.SafeguardIncidents[bbChannelIncidentReportsReceived]; got != 0 {
		t.Fatalf("below-threshold silo must not receive reports, got %d", got)
	}
}

// TestBlackBoxChannelsReportsOnePerMonth 多部门多事实达标仍只送达一份报告。
func TestBlackBoxChannelsReportsOnePerMonth(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.Professions = []model.Profession{
		{Name: "Security", Beliefs: []model.FactBelief{{FactID: "Fact_A", B: 0.9}}},
		{Name: "Archives", Beliefs: []model.FactBelief{{FactID: "Fact_B", B: 0.95}, {FactID: "Fact_C", B: 0.99}}},
	}
	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)
	if got := silo18.SafeguardIncidents[bbChannelIncidentReportsReceived]; got != 1 {
		t.Fatalf("multiple qualified facts must still deliver one report, got %d", got)
	}
}

// ===== G3 Order 例行指令账本 =====

// TestBlackBoxChannelsDirectiveLedger 派生账本与 BlackBox.Log 一致：只含近 12 月
// (含当月)，时间升序，r 与档位逐条正确；未来条目不出现。
func TestBlackBoxChannelsDirectiveLedger(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	// 15 个月指派 Y122M1..Y123M3，覆盖全部档位边界。
	rs := []float64{0.00, 0.05, 0.10, 0.20, 0.35, 0.40, 0.50, 0.55, 0.65, 0.70, 0.85, 0.90, 0.95, 0.30, 0.25}
	for i, r := range rs {
		idx := stewardshipExperimentMonthIndex(122, 1) + i
		silo18.BlackBox.Log = append(silo18.BlackBox.Log, model.BlackBoxLogEntry{
			Year: idx / 12, Month: idx % 12, ChosenR: r,
		})
	}
	// 未来条目 (超出窗口的上侧) 必须被排除。
	silo18.BlackBox.Log = append(silo18.BlackBox.Log, model.BlackBoxLogEntry{
		Year: 123, Month: 4, ChosenR: 0.4,
	})

	svc.world.CurrentYear, svc.world.CurrentMonth = 123, 3
	entries := svc.silo1DirectiveLedger(silo18)
	if len(entries) != 12 {
		t.Fatalf("directive ledger must cover the last 12 months, got %d entries", len(entries))
	}
	wantFirst := silo1DirectiveEntry{Year: 122, Month: 4, R: 0.20, Band: "R1"}
	if entries[0] != wantFirst {
		t.Fatalf("oldest entry = %+v, want %+v", entries[0], wantFirst)
	}
	wantLast := silo1DirectiveEntry{Year: 123, Month: 3, R: 0.25, Band: "R1"}
	if entries[len(entries)-1] != wantLast {
		t.Fatalf("newest entry = %+v, want %+v", entries[len(entries)-1], wantLast)
	}
	for i := 1; i < len(entries); i++ {
		prev, cur := entries[i-1], entries[i]
		if stewardshipExperimentMonthIndex(prev.Year, prev.Month) >=
			stewardshipExperimentMonthIndex(cur.Year, cur.Month) {
			t.Fatalf("ledger must ascend by month: %+v then %+v", prev, cur)
		}
	}
	// 逐条与 Log 对照 (窗口内 Log 下标 3..14)。
	for i, entry := range entries {
		src := silo18.BlackBox.Log[i+3]
		if entry.Year != src.Year || entry.Month != src.Month || entry.R != src.ChosenR {
			t.Fatalf("entry %d diverges from Log: %+v vs %+v", i, entry, src)
		}
		if entry.Band != silo1DirectiveBand(src.ChosenR) {
			t.Fatalf("entry %d band = %q, want %q", i, entry.Band, silo1DirectiveBand(src.ChosenR))
		}
	}
}

// TestBlackBoxChannelsDirectiveBand 档位映射边界 (22-00 §2.2)。
func TestBlackBoxChannelsDirectiveBand(t *testing.T) {
	cases := []struct {
		r    float64
		want string
	}{
		{0.00, "R0"}, {0.05, "R0"},
		{0.10, "R1"}, {0.35, "R1"},
		{0.40, "R2"}, {0.50, "R2"},
		{0.55, "R3"}, {0.65, "R3"},
		{0.70, "R4"}, {0.85, "R4"},
		{0.90, "R5"}, {0.95, "R5"},
	}
	for _, tc := range cases {
		if got := silo1DirectiveBand(tc.r); got != tc.want {
			t.Fatalf("band(%.2f) = %q, want %q", tc.r, got, tc.want)
		}
	}
}

// TestBlackBoxChannelsDirectiveLedgerEmpty 无日志/未启用 → 空账本。
func TestBlackBoxChannelsDirectiveLedgerEmpty(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	if entries := svc.silo1DirectiveLedger(silo18); entries != nil {
		t.Fatalf("empty log must give nil ledger, got %+v", entries)
	}
	if entries := svc.silo1DirectiveLedger(nil); entries != nil {
		t.Fatalf("nil silo must give nil ledger, got %+v", entries)
	}
}

// ===== G5 人口报告通道 =====

// TestBlackBoxChannelsCensusAnomaly 环比阈值：首月只落数；跌幅恰达阈值记异常；
// 低于阈值与人口增长不记；水位逐月更新。
func TestBlackBoxChannelsCensusAnomaly(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	ledgerKey := fmt.Sprintf("%s%d", bbChannelLedgerCensusPrevPrefix, silo18.ID)
	ledger := func() map[string]int {
		return svc.world.Silo1Stewardship.Experiment.Ledger
	}
	anomalies := func() int { return silo18.SafeguardIncidents[bbChannelIncidentCensusAnomaly] }

	// 首月：无上月记录，只落数。
	silo18.TotalPopulation = 10000
	svc.bbChannelCensusMonth(silo18)
	if ledger()[ledgerKey] != 10000 || anomalies() != 0 {
		t.Fatalf("first month must only record: ledger=%d anomalies=%d", ledger()[ledgerKey], anomalies())
	}

	// 环比恰跌 2% (≥ 阈值) → 异常。
	svc.world.CurrentMonth = 2
	silo18.TotalPopulation = 9800
	svc.bbChannelCensusMonth(silo18)
	if anomalies() != 1 || ledger()[ledgerKey] != 9800 {
		t.Fatalf("exact-threshold drop must register: anomalies=%d ledger=%d", anomalies(), ledger()[ledgerKey])
	}

	// 跌幅 100/9800 ≈ 1.02% < 阈值 → 不记。
	svc.world.CurrentMonth = 3
	silo18.TotalPopulation = 9700
	svc.bbChannelCensusMonth(silo18)
	if anomalies() != 1 || ledger()[ledgerKey] != 9700 {
		t.Fatalf("below-threshold drop must not register: anomalies=%d ledger=%d", anomalies(), ledger()[ledgerKey])
	}

	// 人口增长 → 不记。
	svc.world.CurrentMonth = 4
	silo18.TotalPopulation = 9900
	svc.bbChannelCensusMonth(silo18)
	if anomalies() != 1 || ledger()[ledgerKey] != 9900 {
		t.Fatalf("growth must not register: anomalies=%d ledger=%d", anomalies(), ledger()[ledgerKey])
	}

	// 198/9900 = 恰 2% → 再记一次。
	svc.world.CurrentMonth = 5
	silo18.TotalPopulation = 9702
	svc.bbChannelCensusMonth(silo18)
	if anomalies() != 2 || ledger()[ledgerKey] != 9702 {
		t.Fatalf("second threshold drop must register: anomalies=%d ledger=%d", anomalies(), ledger()[ledgerKey])
	}
}

// ===== G7 注意力预算 =====

// TestBlackBoxChannelsAttentionThrottle 每堡每月最多一次主动观测：占用后同月
// 拒绝、他堡不受影响、跨月释放；未启用恒拒绝且消费为空操作。
func TestBlackBoxChannelsAttentionThrottle(t *testing.T) {
	svc, station, _, silo18 := blackBoxChannelsFixture()
	if !svc.silo1ActiveObservationAllowed(station.ID) {
		t.Fatal("fresh month must allow active observation")
	}
	svc.silo1ConsumeActiveObservation(station.ID)
	if svc.silo1ActiveObservationAllowed(station.ID) {
		t.Fatal("consumed slot must be denied within the same month")
	}
	if !svc.silo1ActiveObservationAllowed(silo18.ID) {
		t.Fatal("other silo's slot must be unaffected")
	}
	svc.world.CurrentMonth = 2
	if !svc.silo1ActiveObservationAllowed(station.ID) {
		t.Fatal("new month must free the slot")
	}

	disabled, _, _, _ := blackBoxChannelsFixture()
	disabled.world.Silo1Stewardship.Initialized = false
	if disabled.silo1ActiveObservationAllowed(3) {
		t.Fatal("disabled stewardship must deny active observation")
	}
	disabled.silo1ConsumeActiveObservation(3)
	if disabled.world.Silo1Stewardship.Experiment.Ledger != nil {
		t.Fatal("disabled consume must stay a no-op")
	}
}

// ===== G1 稽查机制 (最小骨架) =====

// TestBlackBoxChannelsPassiveAudit P_danger 恰达门槛、执行层在线且节流允许 →
// 专项核查事实 +1 + 审计日志行 + 占用当月观测槽。
func TestBlackBoxChannelsPassiveAudit(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{
		Year: 122, Month: 1, PDanger: bbChannelAuditPDangerThreshold, ChosenR: 0.4,
	}}
	silo18.BlackBox.CurrentExecution = 1.0
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)

	if got := silo18.SafeguardIncidents[bbChannelIncidentAuditFact]; got != 1 {
		t.Fatalf("passive audit must register one audit_fact, got %d", got)
	}
	if svc.silo1ActiveObservationAllowed(silo18.ID) {
		t.Fatal("passive audit must consume the monthly observation slot")
	}
	key := bbChannelAttentionKey(122, 1, silo18.ID)
	if svc.world.Silo1Stewardship.Experiment.Ledger[key] != 1 {
		t.Fatal("attention ledger must record the consumed slot")
	}
	found := false
	for _, line := range logs {
		if strings.Contains(line, "silo 3 audit carried out") {
			found = true
		}
	}
	if !found {
		t.Fatalf("passive audit must leave an audit log line: %v", logs)
	}
	if len(stories) != 0 {
		t.Fatalf("passive audit must not emit story events: %+v", stories)
	}
}

// TestBlackBoxChannelsPassiveAuditIdleAtExtremeRisk 执行强度低于门槛 (极高指派
// 把 Θ 推高、E→0)：稽查空转，不登记事实、不占用观测槽 (22 系列 §8 A2)。
func TestBlackBoxChannelsPassiveAuditIdleAtExtremeRisk(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{
		Year: 122, Month: 1, PDanger: 0.9, ChosenR: 0.9,
	}}
	silo18.BlackBox.LastAssignedRisk = 0.9
	silo18.BlackBox.CurrentExecution = bbChannelAuditExecutionFloor - 0.001
	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)
	if got := silo18.SafeguardIncidents[bbChannelIncidentAuditFact]; got != 0 {
		t.Fatalf("idle execution must not register audit facts: %+v", silo18.SafeguardIncidents)
	}
	if !svc.silo1ActiveObservationAllowed(silo18.ID) {
		t.Fatal("idle execution must not consume the observation slot")
	}
	found := false
	for _, line := range logs {
		if strings.Contains(line, "inspection postponed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("idle execution must leave a postponement log line: %v", logs)
	}
}

// TestBlackBoxChannelsPassiveAuditBelowThreshold P_danger 低于门槛 → 不触发、
// 不占用观测槽。
func TestBlackBoxChannelsPassiveAuditBelowThreshold(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{
		Year: 122, Month: 1, PDanger: bbChannelAuditPDangerThreshold - 0.001, ChosenR: 0.4,
	}}
	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)
	if silo18.SafeguardIncidents[bbChannelIncidentAuditFact] != 0 {
		t.Fatalf("below-threshold P_danger must not trigger audit: %+v", silo18.SafeguardIncidents)
	}
	if !svc.silo1ActiveObservationAllowed(silo18.ID) {
		t.Fatal("below-threshold trigger must not consume the observation slot")
	}
}

// TestBlackBoxChannelsPassiveAuditThrottled 节流已占用 → 即使 P_danger 达标也
// 不触发 (每堡每月一次)。
func TestBlackBoxChannelsPassiveAuditThrottled(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{
		Year: 122, Month: 1, PDanger: 0.9, ChosenR: 0.4,
	}}
	svc.silo1ConsumeActiveObservation(silo18.ID) // 当月槽已被其他主动观测占用
	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)
	if silo18.SafeguardIncidents[bbChannelIncidentAuditFact] != 0 {
		t.Fatalf("throttled silo must not be audited: %+v", silo18.SafeguardIncidents)
	}
}

// TestBlackBoxChannelsPassiveAuditNeedsLog 无黑箱日志 (从未指派) → 不触发。
func TestBlackBoxChannelsPassiveAuditNeedsLog(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	logs := []string{}
	stories := []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)
	if silo18.SafeguardIncidents[bbChannelIncidentAuditFact] != 0 {
		t.Fatal("empty black-box log must not trigger audit")
	}
}

// ===== G9 异常观察计数器 =====

// TestBlackBoxChannelsIncidentConsumption 汇总全部非 obs 键；四类旧累加键清零，
// W5 契约新键与 obs 前缀键保留。
func TestBlackBoxChannelsIncidentConsumption(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.SafeguardIncidents = map[string]int{
		engine.SafeguardIncidentIgnore:        2,
		engine.SafeguardIncidentCoverCollapse: 1,
		engine.SafeguardIncidentDeviation:     3,
		engine.SafeguardIncidentITOverreach:   1,
		bbChannelIncidentReportsReceived:      4,
		bbChannelIncidentCensusAnomaly:        5,
		bbChannelIncidentAuditFact:            6,
		bbChannelIncidentEquipmentUse:         7,
		"obs_bookkeeping":                     100, // obs 前缀：不计入异常数
	}
	logs := []string{}
	svc.bbChannelConsumeIncidentsMonth(silo18, &logs)

	// 2+1+3+1+4+5+6+7 = 29 (obs_bookkeeping 不计)。
	if len(logs) != 1 || !strings.Contains(logs[0], "silo 3 visible anomalies: 29") {
		t.Fatalf("consumption must log the visible anomaly count, got %v", logs)
	}
	for _, kind := range []string{
		engine.SafeguardIncidentIgnore,
		engine.SafeguardIncidentCoverCollapse,
		engine.SafeguardIncidentDeviation,
		engine.SafeguardIncidentITOverreach,
	} {
		if _, ok := silo18.SafeguardIncidents[kind]; ok {
			t.Fatalf("legacy key %q must be cleared after consumption", kind)
		}
	}
	// W5 契约键保留一个周期 (值不变)。
	if silo18.SafeguardIncidents[bbChannelIncidentReportsReceived] != 4 ||
		silo18.SafeguardIncidents[bbChannelIncidentCensusAnomaly] != 5 ||
		silo18.SafeguardIncidents[bbChannelIncidentAuditFact] != 6 ||
		silo18.SafeguardIncidents[bbChannelIncidentEquipmentUse] != 7 {
		t.Fatalf("W5 contract keys must survive consumption: %+v", silo18.SafeguardIncidents)
	}
	if silo18.SafeguardIncidents["obs_bookkeeping"] != 100 {
		t.Fatalf("obs-prefixed keys must survive consumption: %+v", silo18.SafeguardIncidents)
	}
}

// TestBlackBoxChannelsIncidentConsumptionEdge 空登记簿零日志；仅旧键时清空后
// 登记簿为空表。
func TestBlackBoxChannelsIncidentConsumptionEdge(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	logs := []string{}
	svc.bbChannelConsumeIncidentsMonth(silo18, &logs) // nil 登记簿
	if len(logs) != 0 {
		t.Fatalf("empty registry must stay silent: %v", logs)
	}

	silo18.SafeguardIncidents = map[string]int{engine.SafeguardIncidentDeviation: 2}
	svc.bbChannelConsumeIncidentsMonth(silo18, &logs)
	if len(logs) != 1 || !strings.Contains(logs[0], "visible anomalies: 2") {
		t.Fatalf("legacy-only registry must log its count, got %v", logs)
	}
	if len(silo18.SafeguardIncidents) != 0 {
		t.Fatalf("legacy-only registry must end empty: %+v", silo18.SafeguardIncidents)
	}
}

// ===== 零 RNG 与管线集成 =====

// TestBlackBoxChannelsRoundConsumesNoRNG 各通道真实工作 (举报、人口、核查、
// 消费全部触发) 也零 RNG：回合前后引擎 RNG 流逐位不变。
func TestBlackBoxChannelsRoundConsumesNoRNG(t *testing.T) {
	svc, _, _, silo18 := blackBoxChannelsFixture()
	silo18.Professions = []model.Profession{{Name: "Security",
		Beliefs: []model.FactBelief{{FactID: "Fact_A", B: 0.9}}}}
	silo18.BlackBox.Log = []model.BlackBoxLogEntry{{Year: 122, Month: 1, PDanger: 0.8}}
	silo18.BlackBox.CurrentExecution = 1.0
	silo18.SafeguardIncidents = map[string]int{engine.SafeguardIncidentDeviation: 2}

	const seed = 555
	engine.SetRNG(rand.New(rand.NewSource(seed)))
	logs, stories := []string{}, []model.StoryEvent{}
	svc.runBlackBoxChannelsMonth(&logs, &stories)

	if len(logs) == 0 || len(stories) != 0 {
		t.Fatalf("round must actually work (and stay out of stories): logs=%v stories=%v", logs, stories)
	}
	if silo18.SafeguardIncidents[bbChannelIncidentReportsReceived] != 1 ||
		silo18.SafeguardIncidents[bbChannelIncidentAuditFact] != 1 {
		t.Fatalf("round must register channel incidents: %+v", silo18.SafeguardIncidents)
	}
	ref := rand.New(rand.NewSource(seed))
	for i := 0; i < 64; i++ {
		if got, want := engine.DrawFloat64(), ref.Float64(); got != want {
			t.Fatalf("channels round consumed RNG (draw %d: %v != %v)", i, got, want)
		}
	}
}

// TestBlackBoxChannelsRunThroughWorldPipeline 真实多堡管线驱动：通道回合随月度
// tick 挂载运行——每个非 1 号堡落下人口水位台账，1 号自身不落。
func TestBlackBoxChannelsRunThroughWorldPipeline(t *testing.T) {
	svc := newBlackBoxGame(t, 611)
	advanceMonthForTest(t, svc)
	if !svc.world.Silo1Stewardship.Initialized {
		t.Fatal("stewardship must initialize on the first multi-silo month")
	}
	ledger := svc.world.Silo1Stewardship.Experiment.Ledger
	for _, silo := range svc.silos {
		key := fmt.Sprintf("%s%d", bbChannelLedgerCensusPrevPrefix, silo.ID)
		_, ok := ledger[key]
		if svc.stewardshipExperimentIsSilo1(silo) {
			if ok {
				t.Fatalf("silo 1 itself must not get a census ledger entry: %+v", ledger)
			}
			continue
		}
		if !ok {
			t.Fatalf("silo %d must get a census ledger entry: %+v", silo.ID, ledger)
		}
		if ledger[key] <= 0 {
			t.Fatalf("silo %d census water level must be positive, got %d", silo.ID, ledger[key])
		}
	}
}

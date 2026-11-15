package service

import (
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// 外溢投递侧 (16 §3.5 Layer 3) 定点测试。
//
// 纪律：接收裁量走单一 RNG 流，测试用"同种子孪生源"预读期望抽取值，
// 同时锁住结算结果与"取样点恰好一处"两件事；无效请求必须零 RNG 消耗。

// spilloverTestService 三堡开局 (驻地=1, 1 号=2, 18 号=3，拓扑全连接)。
func spilloverTestService(t *testing.T) *GameService {
	t.Helper()
	svc := newGatedService(t, filepath.Join(t.TempDir(), "spillover.db"))
	if _, err := svc.CreateGame(threeSiloRequest(7101)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	return svc
}

// drawSeedFor 返回使首次抽取落在接受侧 (或拒绝侧) 的最小种子，供接收
// 结算测试做确定性断言。
func drawSeedFor(t *testing.T, acceptW, rejectW float64, accept bool) int64 {
	t.Helper()
	total := acceptW + rejectW
	for seed := int64(1); seed < 4096; seed++ {
		draw := rand.New(rand.NewSource(seed)).Float64()
		if (draw*total < acceptW) == accept {
			return seed
		}
	}
	t.Fatalf("no seed produces accept=%v for weights %v/%v", accept, acceptW, rejectW)
	return 0
}

// TestSpilloverSendRequiresChannelAndPackage 锁显式失败与零 RNG 消耗：
// 无信道或空包裹的投递请求不取样、不入队、不记账。
func TestSpilloverSendRequiresChannelAndPackage(t *testing.T) {
	svc := spilloverTestService(t)
	engine.SetRNG(rand.New(rand.NewSource(7)))
	twin := rand.New(rand.NewSource(7))

	story := svc.settleSpilloverSend(svc.silo, model.SpilloverChannelITLine)
	if story == nil || !strings.Contains(story.Description, "信道") {
		t.Fatalf("missing channel must fail explicitly, got %+v", story)
	}
	if len(svc.world.CrossQueue) != 0 || len(svc.silo.SpilloverPrep.Deliveries) != 0 {
		t.Fatalf("invalid send must not enqueue or ledger: queue=%d deliveries=%d",
			len(svc.world.CrossQueue), len(svc.silo.SpilloverPrep.Deliveries))
	}
	if got := engine.DrawFloat64(); got != twin.Float64() {
		t.Fatalf("invalid send consumed RNG: %v vs %v", got, twin.Float64())
	}

	svc.silo.SpilloverPrep.Channels = append(svc.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: model.SpilloverChannelITLine, TargetSilo: 3, Reliability: 1,
	})
	story = svc.settleSpilloverSend(svc.silo, model.SpilloverChannelITLine)
	if story == nil || !strings.Contains(story.Description, "空壳") {
		t.Fatalf("empty package must fail explicitly, got %+v", story)
	}
	if len(svc.world.CrossQueue) != 0 {
		t.Fatalf("empty package must not enqueue: %d", len(svc.world.CrossQueue))
	}
	if got := engine.DrawFloat64(); got != twin.Float64() {
		t.Fatalf("empty package send consumed RNG: %v vs %v", got, twin.Float64())
	}
}

// TestSpilloverSendDeliversAndExposes 锁发送侧因果：整包经专线容量与损耗
// 结算为到达质量，入队携带代次与链起源，暴露立即记账 (组织怀疑度 + 1 号
// 反谍轨道)，投递账本留下可审计记录。
func TestSpilloverSendDeliversAndExposes(t *testing.T) {
	svc := spilloverTestService(t)
	svc.silo.SpilloverPrep.Channels = append(svc.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: model.SpilloverChannelITLine, TargetSilo: 3, Reliability: 1,
	})
	svc.silo.SpilloverPrep.PackageQuality = 0.8
	suspicionBefore := svc.silo.EscapeResponse.OrgSuspicion
	exposureBefore := svc.world.Silo1.Exposure

	engine.SetRNG(rand.New(rand.NewSource(9)))
	story := svc.settleSpilloverSend(svc.silo, model.SpilloverChannelITLine)
	if story == nil || !strings.Contains(story.Title, "已上路") {
		t.Fatalf("reliable send must depart, got %+v", story)
	}
	if len(svc.world.CrossQueue) != 1 {
		t.Fatalf("send must enqueue exactly one package: %d", len(svc.world.CrossQueue))
	}
	entry := svc.world.CrossQueue[0]
	if entry.EventID != engine.SpilloverPackageEvent || entry.SourceSilo != svc.silo.ID {
		t.Fatalf("unexpected cross event: %+v", entry)
	}
	quality, generation, root, ok := parseSpilloverPayload(entry.Payload)
	if !ok || quality != 0.8 || generation != 1 || root != svc.silo.ID {
		t.Fatalf("payload mismatch: quality=%v gen=%d root=%d ok=%v", quality, generation, root, ok)
	}
	deliveries := svc.silo.SpilloverPrep.Deliveries
	if len(deliveries) != 1 || !deliveries[0].Delivered || deliveries[0].Quality != 0.8 {
		t.Fatalf("delivery ledger mismatch: %+v", deliveries)
	}
	if svc.silo.EscapeResponse.OrgSuspicion <= suspicionBefore {
		t.Fatalf("send must accumulate org suspicion: %v -> %v", suspicionBefore, svc.silo.EscapeResponse.OrgSuspicion)
	}
	if svc.world.Silo1.Exposure <= exposureBefore {
		t.Fatalf("it_line send must raise Silo 1 counterintel exposure: %v -> %v", exposureBefore, svc.world.Silo1.Exposure)
	}
	if svc.silo.SpilloverPrep.Channels[0].Uses != 1 {
		t.Fatalf("channel use must be audited: %+v", svc.silo.SpilloverPrep.Channels[0])
	}
}

// TestSpilloverSendLostInTransit 锁路上丢失：可靠度为 0 时包裹不入队，
// 但暴露照样记账 (失败也留痕迹)，账本记录丢失事实。
func TestSpilloverSendLostInTransit(t *testing.T) {
	svc := spilloverTestService(t)
	svc.silo.SpilloverPrep.Channels = append(svc.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: model.SpilloverChannelCrossSignal, TargetSilo: 3, Reliability: 0,
	})
	svc.silo.SpilloverPrep.PackageQuality = 0.6
	suspicionBefore := svc.silo.EscapeResponse.OrgSuspicion

	engine.SetRNG(rand.New(rand.NewSource(13)))
	story := svc.settleSpilloverSend(svc.silo, model.SpilloverChannelCrossSignal)
	if story == nil || !strings.Contains(story.Title, "路上丢失") {
		t.Fatalf("zero reliability must lose the package, got %+v", story)
	}
	if len(svc.world.CrossQueue) != 0 {
		t.Fatalf("lost package must not enqueue: %d", len(svc.world.CrossQueue))
	}
	deliveries := svc.silo.SpilloverPrep.Deliveries
	if len(deliveries) != 1 || deliveries[0].Delivered {
		t.Fatalf("lost delivery must ledger as not delivered: %+v", deliveries)
	}
	// 跨堡信号容量 0.6 × 包裹 0.6 = 0.36，再乘在途存活 0.9。
	if want := 0.6 * 0.6 * 0.9; deliveries[0].Quality != want {
		t.Fatalf("carried quality mismatch: got %v want %v", deliveries[0].Quality, want)
	}
	if svc.silo.EscapeResponse.OrgSuspicion <= suspicionBefore {
		t.Fatalf("failed send must still accumulate exposure")
	}
}

// TestSpilloverArrivalReception 锁接收结算：到达记录携带独立接收判定，
// 判定读目标堡自身状态；接受时向"尚未到达过的邻堡"排定再复制代，
// 不接受时链就地终止。
func TestSpilloverArrivalReception(t *testing.T) {
	run := func(t *testing.T, accept bool) *GameService {
		t.Helper()
		svc := spilloverTestService(t)
		target := svc.siloByID(3)
		if target == nil {
			t.Fatal("silo 18 missing")
		}
		quality := 0.8
		acceptW, rejectW := spilloverReceptionWeights(target, quality)
		seed := drawSeedFor(t, acceptW, rejectW, accept)
		svc.enqueueSpilloverPackage(svc.silo.ID, target.ID, 0, quality, 1, svc.silo.ID)
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		logs, stories := []string{}, []model.StoryEvent{}
		svc.dispatchCrossQueue(&logs, &stories)

		var packageSignal *model.OutboundSignal
		for i := range svc.world.OutboundSignals {
			signal := &svc.world.OutboundSignals[i]
			if signal.Spillover {
				packageSignal = signal
			}
		}
		if packageSignal == nil {
			t.Fatalf("arrival must append a spillover outbound record (accept=%v)", accept)
		}
		if packageSignal.Received != accept || packageSignal.RootSilo != svc.silo.ID || packageSignal.Generation != 1 {
			t.Fatalf("arrival record mismatch (accept=%v): %+v", accept, *packageSignal)
		}
		if len(stories) == 0 {
			t.Fatalf("arrival must surface a story (accept=%v)", accept)
		}
		return svc
	}

	// 接受：向 18 号的邻堡中尚未到达者 (只剩 1 号地堡) 排定第 2 代。
	accepted := run(t, true)
	relays := 0
	for _, entry := range accepted.world.CrossQueue {
		if entry.EventID != engine.SpilloverPackageEvent {
			continue
		}
		quality, generation, root, ok := parseSpilloverPayload(entry.Payload)
		if !ok || generation != 2 || root != accepted.silo.ID {
			t.Fatalf("relay payload mismatch: %+v ok=%v", entry.Payload, ok)
		}
		if want := 0.8 * spilloverReplicationFidelity; quality != want {
			t.Fatalf("relay quality mismatch: got %v want %v", quality, want)
		}
		if entry.TargetScope != "silo:2" || entry.SourceSilo != 3 {
			t.Fatalf("relay must go from silo 18 to the unreached neighbor silo 2: %+v", entry)
		}
		relays++
	}
	if relays != 1 {
		t.Fatalf("accepted package must schedule exactly one relay, got %d", relays)
	}

	// 拒绝：无再复制。
	rejected := run(t, false)
	for _, entry := range rejected.world.CrossQueue {
		if entry.EventID == engine.SpilloverPackageEvent {
			t.Fatalf("rejected package must not replicate: %+v", entry)
		}
	}
}

// TestSpilloverReplicationChainEnds 锁链的自然边界：第 2 代到达 1 号地堡后，
// 其邻堡 (驻地与 18 号) 都已在链内，第 3 代无处可去，链条终止。
func TestSpilloverReplicationChainEnds(t *testing.T) {
	svc := spilloverTestService(t)
	arrive := func(source, target uint, quality float64, generation int, root uint) {
		t.Helper()
		acceptW, rejectW := spilloverReceptionWeights(svc.siloByID(target), quality)
		seed := drawSeedFor(t, acceptW, rejectW, true)
		entry := model.CrossEvent{
			Seq: 900 + generation, SourceSilo: source,
			TargetScope: "silo:" + itoaU(target), EventID: engine.SpilloverPackageEvent,
			DueYear: svc.world.CurrentYear, DueMonth: svc.world.CurrentMonth,
			Payload: map[string]string{
				"quality": ftoa(quality), "generation": strconv.Itoa(generation), "root": itoaU(root),
			},
		}
		engine.SetRNG(rand.New(rand.NewSource(seed)))
		logs, stories := []string{}, []model.StoryEvent{}
		svc.settleSpilloverArrival(entry, &logs, &stories)
	}

	arrive(1, 3, 0.8, 1, 1) // 驻地 → 18 号：接受，排定 2 代到 1 号地堡
	arrive(3, 2, 0.56, 2, 1) // 18 号 → 1 号地堡：接受，但无处可再传

	for _, entry := range svc.world.CrossQueue {
		if entry.EventID == engine.SpilloverPackageEvent {
			_, generation, _, _ := parseSpilloverPayload(entry.Payload)
			if generation >= 3 {
				t.Fatalf("chain must end when every neighbor is already reached: %+v", entry)
			}
		}
	}
	reached := svc.spilloverChainReached(1)
	if !reached[2] || !reached[3] {
		t.Fatalf("chain reach set mismatch: %+v", reached)
	}
}

// TestSpilloverCountsTightened 锁计数口径：通用信号维持 G7 到达口径，
// 外溢包裹只计"到达且独立接收"，链按 RootSilo 归属。
func TestSpilloverCountsTightened(t *testing.T) {
	svc, _, _ := weightedEndingFixture("INFORMATION")
	svc.world.OutboundSignals = []model.OutboundSignal{
		{Seq: 1, SourceSilo: 1, TargetSilo: 2, EventID: "x", Year: 128, Month: 1},
		{Seq: 2, SourceSilo: 1, TargetSilo: 3, EventID: engine.SpilloverPackageEvent, Spillover: true, Received: true, Generation: 1, RootSilo: 1},
		{Seq: 3, SourceSilo: 1, TargetSilo: 4, EventID: engine.SpilloverPackageEvent, Spillover: true, Received: false, Generation: 1, RootSilo: 1},
		{Seq: 4, SourceSilo: 3, TargetSilo: 2, EventID: engine.SpilloverPackageEvent, Spillover: true, Received: true, Generation: 2, RootSilo: 1},
		{Seq: 5, SourceSilo: 2, TargetSilo: 3, EventID: "y", Year: 128, Month: 3},
	}
	// 触达：2 号 (通用信号到达 + 2 代独立接收) 与 3 号 (1 代独立接收)；
	// 4 号到达但未被独立接收，不计。
	if got := svc.spilloverSiloCount(1); got != 2 {
		t.Fatalf("tightened reach count mismatch: got %d want 2", got)
	}
	// 送出：通用信号 1 条 + 第一代包裹到达 2 条 (含被拒的 4 号)。
	if got := svc.spilloverSignalCount(1); got != 3 {
		t.Fatalf("sent count mismatch: got %d want 3", got)
	}
}

// TestSpilloverSettlementLine 锁结算行：确有投递尝试时追加包裹明细
// (尝试/丢失/独立接收/代次)；外溢仍只进叙事，不进资格。
func TestSpilloverSettlementLine(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: permeationSeedB}, {FactID: "b", B: permeationSeedB}, {FactID: "c", B: permeationSeedB},
		}
		station.Professions = append(station.Professions, prof)
	}
	matureStationBeliefsThroughRealMonths(station)
	station.SpilloverPrep.Deliveries = []model.SpilloverDelivery{
		{Kind: model.SpilloverChannelITLine, TargetSilo: 3, Delivered: true, Quality: 0.8},
		{Kind: model.SpilloverChannelITLine, TargetSilo: 3, Delivered: false, Quality: 0.8},
	}
	svc.world.OutboundSignals = []model.OutboundSignal{
		{Seq: 1, SourceSilo: 1, TargetSilo: 3, EventID: engine.SpilloverPackageEvent, Spillover: true, Received: true, Generation: 1, RootSilo: 1, Quality: 0.8},
		{Seq: 2, SourceSilo: 3, TargetSilo: 2, EventID: engine.SpilloverPackageEvent, Spillover: true, Received: true, Generation: 2, RootSilo: 1, Quality: 0.56},
	}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingInformation {
		t.Fatalf("expected INFORMATION selection, got %+v", decision)
	}
	for _, want := range []string{
		"外溢包裹：共 2 次投递",
		"1 次路上丢失",
		"2 座他堡独立完成核验",
		"链在宿主之外再复制到第 2 代",
	} {
		if !strings.Contains(decision.Reason, want) {
			t.Fatalf("settlement line missing %q: %q", want, decision.Reason)
		}
	}
}

// TestSpilloverChannelAudit 锁 T2-2：使用次数到档可靠度降档，跌破地板
// 信道中断；中断的信道不再参与投递选择。
func TestSpilloverChannelAudit(t *testing.T) {
	svc := spilloverTestService(t)
	svc.silo.SpilloverPrep.Channels = append(svc.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: model.SpilloverChannelITLine, TargetSilo: 3, Reliability: 1,
	})
	svc.silo.SpilloverPrep.PackageQuality = 0.8
	engine.SetRNG(rand.New(rand.NewSource(17)))

	for i := 0; i < spilloverAuditUsesITLine; i++ {
		svc.settleSpilloverSend(svc.silo, model.SpilloverChannelITLine)
	}
	channel := svc.silo.SpilloverPrep.Channels[0]
	if want := spilloverAuditDowngradeFactor; channel.Reliability != want {
		t.Fatalf("audit must downgrade reliability after %d uses: got %v want %v",
			spilloverAuditUsesITLine, channel.Reliability, want)
	}
	if channel.Broken {
		t.Fatalf("first downgrade must not break the channel: %+v", channel)
	}

	// 继续用到中断：1.0 → 0.5 → 0.25 → 0.125 (< 0.15 地板 → 中断)。
	for i := 0; i < spilloverAuditUsesITLine*2 && !svc.silo.SpilloverPrep.Channels[0].Broken; i++ {
		svc.settleSpilloverSend(svc.silo, model.SpilloverChannelITLine)
	}
	channel = svc.silo.SpilloverPrep.Channels[0]
	if !channel.Broken {
		t.Fatalf("sustained use must break the channel: %+v", channel)
	}
	if spilloverChannelForSend(svc.silo, model.SpilloverChannelITLine) != -1 {
		t.Fatalf("broken channel must not be selectable")
	}
}

// TestSpilloverSendGateMetrics 锁门控指标：包裹质量与各类信道可用性。
func TestSpilloverSendGateMetrics(t *testing.T) {
	svc := spilloverTestService(t)
	if value, ok := svc.actionConditionValue(actionCondition{Metric: "spillover_channel_it_line"}); !ok || value != 0 {
		t.Fatalf("channel gate must read 0 without channels: %v %v", value, ok)
	}
	svc.silo.SpilloverPrep.Channels = append(svc.silo.SpilloverPrep.Channels, model.SpilloverChannel{
		Kind: model.SpilloverChannelITLine, TargetSilo: 3, Reliability: 0.75,
	})
	svc.silo.SpilloverPrep.PackageQuality = 0.4
	if value, ok := svc.actionConditionValue(actionCondition{Metric: "spillover_channel_it_line"}); !ok || value != 1 {
		t.Fatalf("channel gate must read 1 with a live channel: %v %v", value, ok)
	}
	if value, ok := svc.actionConditionValue(actionCondition{Metric: "spillover_channel_cross_signal"}); !ok || value != 0 {
		t.Fatalf("other channel kinds must stay unavailable: %v %v", value, ok)
	}
	if value, ok := svc.actionConditionValue(actionCondition{Metric: "package_quality"}); !ok || value != 0.4 {
		t.Fatalf("package quality gate mismatch: %v %v", value, ok)
	}
}

// itoaU / ftoa 测试载荷辅助。
func itoaU(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

func ftoa(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}

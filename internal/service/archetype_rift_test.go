package service

import (
	"math"
	"path/filepath"
	"sort"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// archetype_rift_test.go 裂隙档调优格 (19 §3.2/§6.2/§6.3/§8)。
//
// 本文件只做三件事，全部只动裂隙档的验证面：
//
//  1. 收敛点钉档：裂隙旋钮收敛于 W0 −8 / 检测 0.8 / 放大 1.04。边界回炉
//     证据 (amp 1.2 共证/无人裁决塌进《断层》、amp 1.05/1.045 深闭输给重置
//     链同月竞速、检测 0.75/0.78 除名怀疑到不了 1.0) 由既有可玩宿主复现；
//     本格复探发现 1.03 与竞速塌缩点之间还有 1.04 一格安全空间 (裕度机制
//     见 TestRiftKnobsAtConvergedValues 注释)，放大微调至 1.04 并以全量
//     证据钉档；检测无下探空间，保持 0.8。
//  2. 波动放大可感知性：同种子同动作序列下，裂隙档恐慌/叛乱正向增量严格
//     大于无原型基线 (AmplifyOrderDelta 口径 + 真实事件双路)，回落不受影响。
//  3. 可达性证据：Archetype="rift" 覆写钉档，固定种子无头真实通关公议/共证/
//     除名三线，记录结算月数与叛乱/恐慌峰值，量化断层线 (rebellion≥0.9) 与
//     活链 (rebellion≤0.8) 的挤压。
//
// 纪律：不改既有测试、不改磐石/荒年块、不改资格门、不引入新随机源、不白盒。

// TestRiftKnobsAtConvergedValues 钉住裂隙档回炉收敛后的旋钮值与方向。
//
// 边界证据 (既有可玩宿主复现 + 本格复探，2026-11-15)：
//   - amp 1.2：TestInformationEndingPlayableFromNewGame (共证 2401) 塌成
//     FRACTURE @128-02 (全部门恐慌 0.93)；TestTimeLinePlayableFromNewGame
//     (无人裁决 3401) 塌成 FRACTURE @130-01——恐慌→维生螺旋成立。
//   - amp 1.05/1.045：TestExclusionistEndingPlayableFromNewGame (深闭 2801
//     零输入) 以 RESET @125-02 结算，输掉与重置链的同月竞速。机制：放大把
//     征用前兆 (crisis:reset_omen_requisition) 从 124-06 提前到 124-05，
//     清洁判决链由此赶在《深闭》结算同月跑完。
//   - amp 1.04：征用前兆仍在 124-06，重置执行晚于《深闭》结算 (125-02)
//     一个月才到，裕度与 1.03 相同；裂隙九宿主 (共证/公议/无人裁决/除名/
//     深闭/人治/重置反制/无声探查×裂隙/土著×裂隙) 全绿——1.04 是全线可达
//     的最大保留放大，由本钉档值固定。
//   - 检测 0.75：TestCompromisedEndingPlayableFromNewGame (除名 2801/Mayor)
//     地平线 130-02 怀疑 0.935 < 1.0，塌成 EXCLUSIONIST；复探 0.78 怀疑
//     0.983 仍差一线——0.8 即最低安全值。
func TestRiftKnobsAtConvergedValues(t *testing.T) {
	arch, ok := engine.ArchetypeByID(engine.ArchetypeRift)
	if !ok {
		t.Fatal("rift archetype not registered")
	}
	// 方向固定 (19 §3.2)：裂隙 = 控制松弛 + 波动放大——W 更低、检测更弱、
	// 波动更大。只钉方向与收敛量级，不改任何既有测试。
	if arch.W0Delta >= 0 {
		t.Fatalf("rift must lower initial W, got W0Delta = %v", arch.W0Delta)
	}
	if arch.DetectionFactor >= 1 {
		t.Fatalf("rift must weaken SAMPLED detection, got factor = %v", arch.DetectionFactor)
	}
	if arch.PanicRebellionAmp <= 1 {
		t.Fatalf("rift must amplify order volatility, got amp = %v", arch.PanicRebellionAmp)
	}
	// 收敛量级 (回炉账)：
	if arch.W0Delta != -8 {
		t.Fatalf("rift W0Delta = %v, want converged value -8", arch.W0Delta)
	}
	if arch.DetectionFactor != 0.8 {
		t.Fatalf("rift DetectionFactor = %v, want converged value 0.8 (0.75/0.78 make 《除名》 unreachable)", arch.DetectionFactor)
	}
	if arch.PanicRebellionAmp != 1.04 {
		t.Fatalf("rift PanicRebellionAmp = %v, want converged value 1.04 (1.045 loses the 《深闭》 race, 1.2 collapses 《共证》/《无人裁决》)", arch.PanicRebellionAmp)
	}
	// 裂隙档不触碰资源与危机节奏 (那是荒年轴)：
	if arch.ResourceFactor != 0 || arch.CrisisCooldownFactor != 0 {
		t.Fatalf("rift must not touch famine-axis knobs: resource=%v cooldown=%v",
			arch.ResourceFactor, arch.CrisisCooldownFactor)
	}
}

// TestRiftOrderAmplificationPerceptible locks 19 §3.2 波动放大的可感知性：
// 同种子同动作序列下，裂隙档的恐慌/叛乱正向增量严格大于无原型基线，回落
// 不受影响。双路证据：
//
//	路一 (AmplifyOrderDelta 口径)：放大函数对正向增量按档内系数放大、对
//	    回落恒等——无原型基线即恒等函数，逐增量直接对比。
//	路二 (真实事件)：真实建档、真实月度推进，同种子零输入双跑 (裂隙 vs
//	    磐石对照)。对照档的旋钮在零输入早期局全部不作用于秩序波动轴 (检测
//	    只乘玩家动作怀疑、放大恒等、W 不进入月度叛乱/恐慌动力学、无 W 门控
//	    事件)，两档共享同一事件点火序列；在点火集合一致的匹配月上对比净增
//	    量：含正向叛乱效果的月严格更大，无事件月回落相等。
//
// 结论口径：若档内放大在真实事件路上仍严格可测，则可感知性成立；若弱到
// 不可测，则如实记录 (见测试日志)，裂隙档差异主轴改为检测 0.8 + 低 W。
func TestRiftOrderAmplificationPerceptible(t *testing.T) {
	riftArch, ok := engine.ArchetypeByID(engine.ArchetypeRift)
	if !ok {
		t.Fatal("rift archetype not registered")
	}
	amp := riftArch.PanicRebellionAmp
	if amp <= 1 {
		t.Fatalf("rift amplification must point upward for perceptibility, got %v", amp)
	}

	// 路一：AmplifyOrderDelta 口径。无原型基线 = 恒等；裂隙 = 正向 ×amp。
	riftSilo := &model.Silo{Archetype: engine.ArchetypeRift}
	plainSilo := &model.Silo{} // 无原型基线：行为逐位不变
	for _, metric := range []string{"panic_value", "rebellion"} {
		for _, delta := range []float64{0.02, 0.05, 0.1} {
			got := engine.AmplifyOrderDelta(riftSilo, metric, delta)
			base := engine.AmplifyOrderDelta(plainSilo, metric, delta)
			if base != delta {
				t.Fatalf("no-archetype baseline must be identity, got %v for %s %+v", base, metric, delta)
			}
			if got <= base {
				t.Fatalf("rift %s increment %v must be strictly greater than baseline %v", metric, got, base)
			}
			if want := delta * amp; math.Abs(got-want) > 1e-12 {
				t.Fatalf("rift %s increment = %v, want %v×%v = %v", metric, got, delta, amp, want)
			}
		}
		// 回落 (负增量) 不受放大影响：
		for _, delta := range []float64{-0.02, -0.1} {
			if got, base := engine.AmplifyOrderDelta(riftSilo, metric, delta), engine.AmplifyOrderDelta(plainSilo, metric, delta); got != base {
				t.Fatalf("rift %s decay must stay unamplified: %v vs baseline %v", metric, got, base)
			}
		}
	}

	// 路二：真实事件双跑。同种子 2801、零输入 (同动作序列 = 无动作)，裂隙
	// vs 磐石对照：磐石档在秩序波动轴上的旋钮全部不作用于零输入早期局——
	// 检测系数只乘玩家动作的抽样怀疑 (零输入无怀疑来源)，放大系数恒等，
	// W 偏移与半衰期不进入月度叛乱/恐慌动力学 (政治结算与权威结算都不读
	// 对方)，早期也没有 W 门控事件点火；两档因此共享同一事件点火序列，
	// 差异只剩放大本身。匹配月过滤仍保留，兜底任何意外发散。
	const months = 36
	type monthRecord struct {
		rebDelta float64
		fired    map[string]int // 当月点火事件键 → 次数
	}
	run := func(archetype string) []monthRecord {
		svc := newGatedService(t, filepath.Join(t.TempDir(), "rift-percept-"+archetype+".db"))
		req := threeSiloRequest(2801)
		req.Archetype = archetype
		if _, err := svc.CreateGame(req); err != nil {
			t.Fatalf("create game (%s): %v", archetype, err)
		}
		if svc.silo.Archetype != archetype {
			t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, archetype)
		}
		prevCounts := map[string]int{}
		records := make([]monthRecord, 0, months)
		for month := 1; month <= months; month++ {
			rebBefore := svc.silo.Rebellion
			if _, err := svc.PassTime(1); err != nil {
				t.Fatalf("advance %s at month %d: %v", archetype, month, err)
			}
			if svc.gameOver() {
				break
			}
			fired := map[string]int{}
			for key, state := range svc.contentStates[svc.silo.ID] {
				if state.TriggerCount > prevCounts[key] {
					fired[key] = state.TriggerCount - prevCounts[key]
					prevCounts[key] = state.TriggerCount
				}
			}
			records = append(records, monthRecord{rebDelta: svc.silo.Rebellion - rebBefore, fired: fired})
		}
		return records
	}
	riftMonths := run(engine.ArchetypeRift)
	baseMonths := run(engine.ArchetypeBedrock)

	// 匹配月：两档当月点火集合一致 (兜底过滤任何意外发散月)。
	sameFiring := func(a, b map[string]int) bool {
		if len(a) != len(b) {
			return false
		}
		for key, count := range a {
			if b[key] != count {
				return false
			}
		}
		return true
	}
	// 当月点火事件的声明式正向叛乱效果之和；含脚本 apply 的事件效果不可见，
	// 该月标记为不可分解 (只参与点火集合对比，不参与增量断言)。
	defs := contentDefinitionMap(t)
	positiveRebellionEffect := func(fired map[string]int) (sum float64, decomposable bool) {
		decomposable = true
		for key, count := range fired {
			def, ok := defs[key]
			if !ok {
				decomposable = false
				continue
			}
			if def.ScriptHooks.Apply {
				decomposable = false
			}
			for _, effect := range def.Effects {
				if effect.Type == "silo_metric_delta" && effect.Metric == "rebellion" && effect.Value > 0 {
					sum += effect.Value * float64(count)
				}
			}
		}
		return sum, decomposable
	}

	n := len(riftMonths)
	if len(baseMonths) < n {
		n = len(baseMonths)
	}
	if n < 12 {
		t.Fatalf("paired run too short to compare: rift=%d base=%d months", len(riftMonths), len(baseMonths))
	}
	strictGreaterMonths, decayEqualMonths := 0, 0
	for i := 0; i < n; i++ {
		riftMonth, baseMonth := riftMonths[i], baseMonths[i]
		if !sameFiring(riftMonth.fired, baseMonth.fired) {
			continue
		}
		sum, decomposable := positiveRebellionEffect(riftMonth.fired)
		if !decomposable {
			continue
		}
		if sum > 0 {
			// 正向增量月：裂隙严格大于基线，超出量 ≈ (amp−1)×Σ效果。
			if riftMonth.rebDelta <= baseMonth.rebDelta {
				t.Fatalf("month %d: rift rebellion delta %+.9f must be strictly greater than baseline %+.9f (fired=%v)",
					i+1, riftMonth.rebDelta, baseMonth.rebDelta, riftMonth.fired)
			}
			excess := riftMonth.rebDelta - baseMonth.rebDelta
			wantExcess := (amp - 1) * sum
			if excess < wantExcess/2 || excess > wantExcess*1.5+1e-4 {
				t.Fatalf("month %d: amplification excess %+.9f outside band around (amp-1)×Σ=%+.9f (fired=%v)",
					i+1, excess, wantExcess, riftMonth.fired)
			}
			strictGreaterMonths++
			t.Logf("perceptible month %d: baseline %+.9f → rift %+.9f (excess %+.9f, Σeffects=%.3f, fired=%v)",
				i+1, baseMonth.rebDelta, riftMonth.rebDelta, excess, sum, riftMonth.fired)
		} else if riftMonth.rebDelta < 0 && baseMonth.rebDelta < 0 {
			// 无正向事件月：回落必须不受放大影响 (自然衰减两档同值)。
			if math.Abs(riftMonth.rebDelta-baseMonth.rebDelta) > 1e-4 {
				t.Fatalf("month %d: decay must stay unamplified: rift %+.9f vs baseline %+.9f",
					i+1, riftMonth.rebDelta, baseMonth.rebDelta)
			}
			decayEqualMonths++
		}
	}
	if strictGreaterMonths == 0 {
		t.Fatalf("no matched month with positive rebellion effects observed in %d months — widen the window instead of weakening the claim", n)
	}
	if decayEqualMonths == 0 {
		t.Fatalf("no matched decay month observed in %d months — widen the window instead of weakening the claim", n)
	}
	t.Logf("rift order amplification perceptible: %d strict-greater months, %d unamplified decay months (amp=%v)",
		strictGreaterMonths, decayEqualMonths, amp)
}

// contentDefinitionMap 真实内容定义按事件键索引 (只读)。
func contentDefinitionMap(t *testing.T) map[string]engine.ContentEventDefinition {
	t.Helper()
	boot := bootstrapRealContent(t)
	defs := map[string]engine.ContentEventDefinition{}
	for _, def := range boot.contentDefinitions {
		defs[def.Key] = def
	}
	return defs
}

// riftPeaks 裂隙档可达性跑测的叛乱/恐慌峰值记录 (19 §8 失败线挤压量化)。
type riftPeaks struct {
	rebellionPeak   float64
	panicPeak       float64
	rebellionAtEnd  float64
	monthsRebGe08   int // 叛乱 ≥0.8 (断层喂大阈值) 的月数
	monthsRebGe07   int // 叛乱 ≥0.7 (活链路线蛰伏线) 的月数
}

func (p *riftPeaks) observe(svc *GameService) {
	if svc.silo.Rebellion > p.rebellionPeak {
		p.rebellionPeak = svc.silo.Rebellion
	}
	if svc.silo.Rebellion >= 0.8 {
		p.monthsRebGe08++
	}
	if svc.silo.Rebellion >= 0.7 {
		p.monthsRebGe07++
	}
	for i := range svc.silo.Professions {
		if svc.silo.Professions[i].PanicValue > p.panicPeak {
			p.panicPeak = svc.silo.Professions[i].PanicValue
		}
	}
}

// TestRiftRuleOfLawReachable 可达性证据·《公议》：Archetype="rift" 覆写钉档，
// 种子 2401 / Mines / 起始 120 (与参考路线 TestRuleOfLawLinePlayableFromNewGame
// 同口径)，无头真实打到 RULE_OF_LAW 结算；全程真实动作与月度推进，不白盒。
// 策略即参考路线的裂隙适配版 (隧道前置 5 条、曝光基金优先于填充)。
func TestRiftRuleOfLawReachable(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "rift-rol.db"))
	req := threeSiloRequest(2401)
	req.Profession = "Mines"
	req.StartYear = 120
	req.Archetype = engine.ArchetypeRift
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeRift {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeRift)
	}
	peaks := &riftPeaks{}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
		peaks.observe(svc)
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	var offer *model.PlayerActionMeta
	for i := 0; i < 36; i++ {
		offer = actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_OFFER")
		if offer != nil && offer.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled && svc.agent.ActionPoints >= model.ACTION_COSTS[model.ActionBuildConnection] {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance for contact card: %v", err)
		}
		peaks.observe(svc)
	}
	if offer == nil || !offer.Enabled {
		t.Fatalf("charter contact card never unlocked through real gates: %+v", offer)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_OFFER"})
	accept := actionByID(svc.availablePlayerActions(), "CHARTER_CONTACT_ACCEPT")
	if accept == nil || !accept.Enabled {
		t.Fatalf("accept choice not available after the contact offer: %+v", accept)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "CHARTER_CONTACT_ACCEPT"})

	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	exposes, lastExposeMonth := 0, -3
	shares := map[string]int{}
	tunnels := 0
	decisiveFired := false
	decisiveImpactDone := false
	attempt := func(actionID string, cost float64) bool {
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		ok := err == nil && outcome.Result.Executed
		if ok {
			peaks.observe(svc)
		}
		return ok
	}
	for month := 1; month <= 140 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance RULE_OF_LAW line at month %d: %v", month, err)
		}
		peaks.observe(svc)
		if svc.gameOver() {
			break
		}
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		if contentTriggered(svc, "crisis:reset_omen_requisition") &&
			contentTriggered(svc, "player_action:RESET_COUNTERMEASURE_OFFER") &&
			!contentTriggered(svc, "player_action:reset_countermeasure_done") &&
			svc.agent.ActionPoints >= model.ACTION_COSTS[model.ActionShareInfo] {
			for _, belief := range svc.agent.Beliefs {
				if belief.B < 0.3 {
					continue
				}
				execute(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: "Mayor", FragmentIds: []string{belief.FactID},
				})
				break
			}
		}
		// 裂隙适配 (与参考路线同策略)：放大让叛乱螺旋早于约章锁定成熟，
		// 锁定前隧道网络停在 5 条，把 AP 让给曝光基金；锁定后继续铺。
		tunnelTarget := 5
		if svc.silo.Authority.Judicial.PactLocked {
			tunnelTarget = 14
		}
		if tunnels < tunnelTarget && svc.agent.ActionPoints >= 20 {
			tunnel := actionByID(svc.availablePlayerActions(), "MINES_TUNNEL_NETWORK")
			if tunnel != nil && tunnel.Enabled {
				execute(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MINES_TUNNEL_NETWORK"})
				tunnels++
				continue
			}
		}
		exposeReserve := 0.0
		if !decisiveFired && exposes < 3 {
			exposeReserve = 25
		}
		gameMonth := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
		secondPublishReady := exposes >= 3 && gameMonth-lastExposeMonth >= 3 &&
			svc.agent.ActionPoints >= 20
		if playerCoreBelief(svc, "Judicial_1") >= 0.85 && !factVerified(svc.silo, "Judicial_1") &&
			svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		if factPublished(svc.silo, "Judicial_1") && !factVerified(svc.silo, "Judicial_2") &&
			playerCoreBelief(svc, "Judicial_2") >= 0.85 && svc.agent.ActionPoints >= 15 {
			attempt("verify_evidence", 15)
		}
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if fact == "Judicial_2" && !secondPublishReady {
				continue
			}
			if playerCoreBelief(svc, fact) >= 0.9 && factVerified(svc.silo, fact) &&
				!factPublished(svc.silo, fact) && svc.agent.ActionPoints >= 20 {
				if attempt("publish_evidence", 20) && fact == "Judicial_2" {
					decisiveFired = true
				}
				break
			}
		}
		if svc.agent.ActionPoints >= 15+exposeReserve &&
			(playerCoreBelief(svc, "Judicial_1") < 0.9 || playerCoreBelief(svc, "Judicial_2") < 0.9) &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			attempt("investigate_dispatch", 15)
		}
		exposeForPressure := !decisiveFired && exposes < 3 && svc.agent.ActionPoints >= 25
		exposeForDecisive := decisiveFired && !decisiveImpactDone && svc.agent.ActionPoints >= 20
		if (exposeForPressure || exposeForDecisive) && gameMonth-lastExposeMonth >= 3 &&
			(playerCoreBelief(svc, "Judicial_1") >= 0.6 || playerCoreBelief(svc, "Judicial_2") >= 0.6) {
			if attempt("expose_execution_failure", 20) {
				exposes++
				lastExposeMonth = gameMonth
				if decisiveFired {
					decisiveImpactDone = true
				}
			}
		}
		for _, dept := range []string{"Mechanical", "Supply"} {
			for _, fact := range []string{"Judicial_1", "Judicial_2"} {
				if playerCoreBelief(svc, fact) < 0.6 || svc.agent.ActionPoints < 25+exposeReserve ||
					shares[dept+":"+fact] >= 2 || (exposes >= 3 && !decisiveFired) {
					continue
				}
				outcome, err := svc.ExecuteAction(model.AgentAction{
					Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{fact},
				})
				if err == nil && outcome.Result.Executed {
					shares[dept+":"+fact]++
					peaks.observe(svc)
				}
			}
		}
	}

	if !svc.silo.Authority.Judicial.PactLocked {
		t.Fatalf("rift RULE_OF_LAW route ended without a locked pact (W=%v Q=%v facts=%v)",
			svc.silo.Authority.W, svc.silo.Authority.Judicial.PublicEvidenceQ,
			svc.silo.Authority.Judicial.IndependentPublicFacts)
	}
	status := svc.silo.VictoryStatus
	if status == nil || status.Decision == nil || status.Decision.Selected != endingRuleOfLaw {
		t.Fatalf("rift route did not settle as RULE_OF_LAW at %d-%02d: status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, status)
	}
	peaks.rebellionAtEnd = svc.silo.Rebellion
	settledMonths := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
	t.Logf("rift 《公议》 reachable: seed=2401 Mines start=120 settled=%d-%02d (%d months, %d tunnels, %d exposes) rebellion peak=%.3f (≥0.7: %d months, ≥0.8: %d months) panic peak=%.3f end rebellion=%.3f",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, settledMonths, tunnels, exposes,
		peaks.rebellionPeak, peaks.monthsRebGe07, peaks.monthsRebGe08, peaks.panicPeak, peaks.rebellionAtEnd)
}

// TestRiftInformationReachable 可达性证据·《共证》：Archetype="rift" 覆写钉档，
// 种子 2401 / Judicial (与参考路线 TestInformationEndingPlayableFromNewGame
// 同口径)，无头真实打到 INFORMATION 结算。策略即参考路线的渗透适配循环
// (维护优先、蛰伏线 rebellion≥0.7)。本线同时量化裂隙档的核心挤压：活链确认
// 要求 rebellion ≤ 0.8 (living_chain_confirmed 触发)，而放大把正向叛乱增量
// 抬向断层喂大阈值 (≥0.9)——记录活链点火月的叛乱值与全程峰值。
func TestRiftInformationReachable(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "rift-information.db"))
	req := threeSiloRequest(2401)
	req.Profession = "Judicial"
	req.Archetype = engine.ArchetypeRift
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeRift {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeRift)
	}
	peaks := &riftPeaks{}
	livingChainRebellion := -1.0
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
		peaks.observe(svc)
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	if svc.world.ProgressStage != "first_signal" {
		t.Fatalf("real Gather action did not open the line: %s", svc.world.ProgressStage)
	}
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})

	for month := 0; month < 12 && svc.silo.Authority.Judicial.PublicEvidenceQ <= 0 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance bailiff evidence at month %d: %v", month, err)
		}
		peaks.observe(svc)
	}
	publishFloor := engine.DefaultInformationConfig().QualPublish
	judicialBelief := engine.BeliefFor(svc.agent, "Judicial_2")
	if judicialBelief == nil || judicialBelief.B < publishFloor {
		t.Fatalf("real months did not reach the Judicial_2 publish floor %.2f: belief=%+v", publishFloor, judicialBelief)
	}

	verifyReportFloor := engine.DefaultInformationConfig().QualReport
	publishedFacts := func() map[string]bool {
		out := map[string]bool{}
		for _, record := range svc.silo.PublicEvidence {
			out[record.FactID] = true
		}
		return out
	}
	attempt := func(actionID string, cost float64) bool {
		t.Helper()
		if svc.agent.ActionPoints < cost {
			return false
		}
		outcome, err := svc.ExecuteAction(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: actionID})
		ok := err == nil && outcome.Result.Executed
		if ok {
			peaks.observe(svc)
		}
		return ok
	}
	for i := 0; i < 14; i++ {
		published := publishedFacts()
		if published["Judicial_1"] && published["Judicial_2"] {
			break
		}
		verifiable, publishable := false, false
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			belief := engine.BeliefFor(svc.agent, fact)
			if belief == nil {
				continue
			}
			verified := false
			for _, v := range svc.silo.VerifiedEvidence {
				if v == fact {
					verified = true
				}
			}
			if belief.B >= verifyReportFloor && !verified {
				verifiable = true
			}
			if belief.B >= publishFloor && verified && !published[fact] {
				publishable = true
			}
		}
		if verifiable {
			attempt("verify_evidence", 15)
		}
		if publishable {
			attempt("publish_evidence", 20)
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance publication at step %d: %v", i, err)
		}
		peaks.observe(svc)
		progressResetCountermeasureForRoute(t, svc, execute)
	}
	if !contentTriggered(svc, "player_action:publish_evidence") {
		t.Fatalf("player publication never completed (verified=%v, evidence=%+v)", svc.silo.VerifiedEvidence, svc.silo.PublicEvidence)
	}
	if svc.gameOver() {
		t.Fatalf("game ended early at %d-%02d during publication: reason=%q",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, decisionReasonOf(svc.silo.VictoryStatus))
	}

	var receiver *model.PlayerActionMeta
	for i := 0; i < 8; i++ {
		receiver = actionByID(svc.availablePlayerActions(), "RECEIVER_VERIFY_ACCOMPANY")
		if receiver != nil && receiver.Enabled {
			break
		}
		build := actionByID(svc.availablePlayerActions(), "action_build_connection")
		if build != nil && build.Enabled {
			execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
			continue
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("recover AP for receiver verification: %v", err)
		}
		peaks.observe(svc)
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})

	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	circulationActions := 0
	attemptCirculation := func(action model.AgentAction) bool {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("circulate with %+v: %v", action, err)
		}
		if outcome.Result.Executed {
			circulationActions++
			peaks.observe(svc)
		}
		return outcome.Result.Executed
	}
	shareAttempts := map[string]int{}
	upkeepAttempts := map[string]int{}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		peaks.observe(svc)
		if svc.gameOver() {
			break
		}
		if livingChainRebellion < 0 && contentTriggered(svc, "special:living_chain_confirmed") {
			livingChainRebellion = svc.silo.Rebellion
		}
		progressResetCountermeasureForRoute(t, svc, execute)

		shareCost := model.ACTION_COSTS[model.ActionShareInfo]
		gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
		buildCost := model.ACTION_COSTS[model.ActionBuildConnection]
		gatherFrom := rotation[month%len(rotation)]
		if svc.silo.Rebellion >= 0.7 {
			// 裂隙适配：叛乱过热时蛰伏 (与参考路线同策略)。
			continue
		}
		circulated := false
		for svc.agent.ActionPoints >= shareCost {
			order := append([]model.Profession(nil), svc.silo.Professions...)
			sort.SliceStable(order, func(i, j int) bool {
				mi, mj := informationMaturedFacts(&order[i]), informationMaturedFacts(&order[j])
				if mi != mj {
					return mi < mj
				}
				return informationActiveSlots(&order[i]) < informationActiveSlots(&order[j])
			})
			shared := false
			for i := range order {
				dept := &order[i]
				if informationMaturedFacts(dept) >= informationGateFactCount {
					continue
				}
				if fragment, ok := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts); ok {
					upkeepAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						break
					}
				}
				if informationActiveSlots(dept) >= informationGateFactCount {
					continue
				}
				if month <= 30 && svc.pendingOperation == nil &&
					agentConnectionTo(svc, dept) < 0.3 && svc.agent.ActionPoints >= buildCost {
					if attemptCirculation(model.AgentAction{
						Type: model.ActionBuildConnection, TargetDept: dept.Name,
					}) {
						shared = true
						break
					}
				}
				if fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					return shareAttempts[dept.Name+":"+fact] >= 4
				}); ok {
					if agentBelief := engine.BeliefFor(svc.agent, fragment); agentBelief != nil && agentBelief.B < 0.80 &&
						svc.agent.ActionPoints >= gatherCost {
						if attemptCirculation(model.AgentAction{
							Type: model.ActionGatherInfo, TargetDept: fragmentDepartment(fragment),
						}) {
							shared = true
							break
						}
					}
					shareAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						break
					}
				}
			}
			if !shared {
				break
			}
			circulated = true
		}
		if circulated {
			continue
		}
		switch {
		case svc.agent.ActionPoints >= buildCost && month < 12:
			attemptCirculation(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Judicial"})
		case svc.agent.ActionPoints >= gatherCost:
			attemptCirculation(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: gatherFrom})
		}
	}

	if circulationActions == 0 {
		t.Fatal("rift route reached its ending without any sustained circulation action")
	}
	for _, key := range []string{
		"player_actions:receiver_verify_accompany",
		"special:replication_progress",
		"special:independent_receivers_met",
		"special:living_chain_confirmed",
		"special:public_fact_cross_dept",
	} {
		if !contentTriggered(svc, key) {
			t.Errorf("%s did not fire in the rift route (Q=%v, evidence=%+v)", key, svc.silo.Authority.Judicial.PublicEvidenceQ, svc.silo.PublicEvidence)
		}
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingInformation {
		t.Fatalf("rift route did not settle as INFORMATION at %d-%02d: status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus)
	}
	settledMonths := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
	t.Logf("rift 《共证》 reachable: seed=2401 Judicial settled=%d-%02d (%d circulation months, %d actions) rebellion peak=%.3f (≥0.7: %d months, ≥0.8: %d months) living-chain rebellion=%.3f (cap 0.8, fracture feed 0.9) panic peak=%.3f",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, settledMonths, circulationActions,
		peaks.rebellionPeak, peaks.monthsRebGe07, peaks.monthsRebGe08, livingChainRebellion, peaks.panicPeak)
}

// TestRiftCompromisedReachable 可达性证据·《除名》：Archetype="rift" 覆写钉档，
// 种子 2801 / Mayor (与参考路线 TestCompromisedEndingPlayableFromNewGame 同口径)，
// 无头真实打到 AGENT_COMPROMISED 结算。本线是检测系数 0.8 的承重宿主：抽样
// 怀疑积累 (建连 ×0.8) 必须在地平线内到 1.0。裂隙适配：叛乱过热时市长演说压恐慌。
func TestRiftCompromisedReachable(t *testing.T) {
	svc := newGatedService(t, filepath.Join(t.TempDir(), "rift-compromised.db"))
	req := threeSiloRequest(2801)
	req.Profession = "Mayor"
	req.Archetype = engine.ArchetypeRift
	if _, err := svc.CreateGame(req); err != nil {
		t.Fatalf("create game: %v", err)
	}
	if svc.silo.Archetype != engine.ArchetypeRift {
		t.Fatalf("archetype override = %q, want %q", svc.silo.Archetype, engine.ArchetypeRift)
	}
	peaks := &riftPeaks{}
	execute := func(action model.AgentAction) {
		t.Helper()
		outcome, err := svc.ExecuteAction(action)
		if err != nil {
			t.Fatalf("execute %+v: %v (AP=%v)", action, err, svc.agent.ActionPoints)
		}
		if !outcome.Result.Executed {
			t.Fatalf("execute %+v rejected: %+v (AP=%v)", action, outcome.Result, svc.agent.ActionPoints)
		}
		peaks.observe(svc)
	}

	execute(model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor"})
	execute(model.AgentAction{
		Type: model.ActionShareInfo, TargetDept: "Mayor",
		FragmentIds: []string{svc.agent.KnownFragments[0]},
	})
	for i := 0; i < 2; i++ {
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
		execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mechanical"})
	}
	execute(model.AgentAction{Type: model.ActionConductPropaganda})

	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	infiltrations, suspicionBuilds := 0, 0
	for month := 1; month <= 120 && !svc.gameOver(); month++ {
		progressResetCountermeasureForRoute(t, svc, execute)
		if svc.gameOver() {
			break
		}
		actions := svc.availablePlayerActions()
		// 裂隙适配 (与参考路线同策略)：放大让叛乱更热，过热时市长演说压恐慌。
		if svc.silo.Rebellion >= 0.7 && svc.agent.ActionPoints >= 15 {
			if address := actionByID(actions, "MAYOR_PUBLIC_ADDRESS"); address != nil && address.Enabled {
				execute(model.AgentAction{Type: model.ActionProfession, ProfessionAction: "MAYOR_PUBLIC_ADDRESS"})
				continue
			}
		}
		if infiltrations == 0 {
			infiltrate := actionByID(actions, StationOpInfiltrate)
			if infiltrate != nil && infiltrate.Enabled && svc.agent.ActionPoints >= float64(infiltrate.APCost) {
				execute(model.AgentAction{Type: model.ActionStationOperation, ActionID: StationOpInfiltrate})
				infiltrations++
				continue
			}
		} else {
			build := actionByID(actions, "action_build_connection")
			if build != nil && build.Enabled && svc.agent.ActionPoints >= float64(build.APCost) {
				execute(model.AgentAction{Type: model.ActionBuildConnection, TargetDept: "Mayor"})
				suspicionBuilds++
				continue
			}
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance AGENT_COMPROMISED line at month %d: %v", month, err)
		}
		peaks.observe(svc)
	}

	if infiltrations == 0 || suspicionBuilds == 0 {
		t.Fatalf("real gates did not execute the covert probe and sustained relation building: infiltrations=%d builds=%d status=%+v",
			infiltrations, suspicionBuilds, svc.silo.VictoryStatus)
	}
	if svc.silo.VictoryStatus == nil || svc.silo.VictoryStatus.Decision == nil ||
		svc.silo.VictoryStatus.Decision.Selected != endingCompromised {
		t.Fatalf("rift route did not settle as AGENT_COMPROMISED at %d-%02d: status=%+v suspicion=%.3f",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, svc.silo.VictoryStatus, svc.agent.SuspicionLevel)
	}
	if svc.agent.SuspicionLevel < 1.0 {
		t.Fatalf("AGENT_COMPROMISED settled below the suspicion cap: %.3f", svc.agent.SuspicionLevel)
	}
	settledMonths := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
	t.Logf("rift 《除名》 reachable: seed=2801 Mayor settled=%d-%02d (%d months, %d infiltration, %d builds) suspicion=%.3f rebellion peak=%.3f (≥0.7: %d months) panic peak=%.3f",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, settledMonths, infiltrations, suspicionBuilds,
		svc.agent.SuspicionLevel, peaks.rebellionPeak, peaks.monthsRebGe07, peaks.panicPeak)
}

package service

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
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
	rebellionPeak  float64
	panicPeak      float64
	rebellionAtEnd float64
	monthsRebGe08  int // 叛乱 ≥0.8 (断层喂大阈值) 的月数
	monthsRebGe07  int // 叛乱 ≥0.7 (活链路线蛰伏线) 的月数
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
	// 20 §7.3 观测项随批并入 (裂隙信息线复跑)：
	//  ① 恶化档触发分布——T1-3 关系冷却与 T3-2 联系人监视按月末采样
	//    (反制 tick 走月度阶段)：窗口 = 部门由无到有的进入次数，月数 =
	//    部门·月累计，最长单窗受 §6.3 上界纪律 (≤24 月) 约束；任一档
	//    全程零触发 → 显式失败 (§7.3 第 1 条：阈值脱离真实玩法，回 §6
	//    重调，而不是放任不观测)。
	//  ③ 摩擦 AP——T1-2 戒备上浮建联 / T2-3 收紧交接 / T3-2 关系跌落重建
	//    在执行时点按 resolveActionCost 权威成本记账超出基准的部分，全程
	//    总额受 §6.1 预算 ≤36 AP 约束。
	//  ② 结算月漂移随结算行记录，对照账落在 roadmap 第 5 批状态行。
	// 窗口口径 (20 §4.1/§6.3)：T1-3 每次触发授窗 counterRelationMonths 月，
	// 到期后部门仍藏泄漏则"再触发一次"另起新窗 (刷新不叠加)。因此按授窗
	// 计数：剩余月数从 0 转正或回跳 (到期同月重授) = 一个新窗；§6.3 的
	// "超过 24 月即判过界"上界纪律约束单授窗连续时长。连续受冷却的链长与
	// 同月受冷却部门数只入观测账 (链长是 §4.1 明示的设计行为)。
	type cooldownStat struct {
		grants, months, running, maxGrant int
	}
	cooldownStats := map[string]*cooldownStat{}
	prevCooldown := map[string]int{}
	cooldownChainMax := 0
	cooldownChainRunning := map[string]int{}
	maxSimulCooldown := 0
	contactWatches := map[string]int{}
	contactMonths := 0
	contactRunning := map[string]int{}
	observeCounters := func() {
		simul := 0
		for i := range svc.silo.Professions {
			dept := &svc.silo.Professions[i]
			rem := svc.silo.Countermeasure.RelationCooldown[dept.Name]
			st := cooldownStats[dept.Name]
			if st == nil {
				st = &cooldownStat{}
				cooldownStats[dept.Name] = st
			}
			if rem > 0 {
				simul++
				if prevCooldown[dept.Name] == 0 || rem >= prevCooldown[dept.Name] {
					st.grants++
					st.running = 1
					cooldownChainRunning[dept.Name] = 1
				} else {
					st.running++
					cooldownChainRunning[dept.Name]++
				}
				st.months++
				if st.running > st.maxGrant {
					st.maxGrant = st.running
				}
				if cooldownChainRunning[dept.Name] > cooldownChainMax {
					cooldownChainMax = cooldownChainRunning[dept.Name]
				}
			} else {
				st.running = 0
				cooldownChainRunning[dept.Name] = 0
			}
			prevCooldown[dept.Name] = rem
			if svc.silo.Countermeasure.ContactWatch[dept.Name] > 0 {
				if contactRunning[dept.Name] == 0 {
					contactWatches[dept.Name]++
				}
				contactRunning[dept.Name]++
				contactMonths++
			} else {
				contactRunning[dept.Name] = 0
			}
		}
		if simul > maxSimulCooldown {
			maxSimulCooldown = simul
		}
	}
	var frictionBuildAP, frictionShareAP float64
	accountFriction := func(action model.AgentAction) {
		var probe model.AgentAction
		var base float64
		switch action.Type {
		case model.ActionBuildConnection:
			probe = model.AgentAction{Type: model.ActionBuildConnection, TargetDept: action.TargetDept}
			base = model.ACTION_COSTS[model.ActionBuildConnection]
		case model.ActionShareInfo:
			probe = model.AgentAction{Type: model.ActionShareInfo, TargetDept: action.TargetDept, FragmentIds: []string{"probe"}}
			base = model.ACTION_COSTS[model.ActionShareInfo]
		default:
			return
		}
		if err := svc.resolveActionCost(&probe); err != nil {
			return
		}
		if over := probe.Cost - base; over > 0 {
			if action.Type == model.ActionBuildConnection {
				frictionBuildAP += over
			} else {
				frictionShareAP += over
			}
		}
	}
	execute := func(action model.AgentAction) {
		t.Helper()
		accountFriction(action)
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
		observeCounters()
	}
	publishFloor := engine.DefaultInformationConfig().QualPublish
	judicialBelief := engine.BeliefFor(svc.agent, "Judicial_2")
	if judicialBelief == nil || judicialBelief.B < publishFloor {
		t.Fatalf("real months did not reach the Judicial_2 publish floor %.2f: belief=%+v", publishFloor, judicialBelief)
	}

	publishedFacts := func() map[string]bool {
		out := map[string]bool{}
		for _, record := range svc.silo.PublicEvidence {
			out[record.FactID] = true
		}
		return out
	}
	// 双事实发布与待发布存蓄 (与参考路线 26 §四 B 同口径)：Judicial_1 开局
	// 通常没有信念，investigate_dispatch (15 AP，确定性深挖较弱链) 是把它抬
	// 到发布地板的现成手段；一旦有事实已核验待发布，AP 必须存给 20 的发布
	// 而不是继续 15 的调查——否则月收入被调查吃干，第二条事实发布带来的
	// 全部门免费粘性槽整段丢失 (早期 AP 富余 + Judicial_1 未公开，正是
	// 124-07 自发 Judicial_1 大海啸、把尾部门预算吃光的来源)。
	eventReady := func(id string) bool {
		meta := actionByID(svc.availablePlayerActions(), id)
		return meta != nil && meta.Enabled
	}
	publishPending := func() bool {
		verified := map[string]bool{}
		for _, fact := range svc.silo.VerifiedEvidence {
			verified[fact] = true
		}
		published := publishedFacts()
		for _, fact := range []string{"Judicial_1", "Judicial_2"} {
			if !verified[fact] || published[fact] {
				continue
			}
			if belief := engine.BeliefFor(svc.agent, fact); belief != nil && belief.B >= publishFloor {
				return true
			}
		}
		return false
	}
	for i := 0; i < 40; i++ {
		published := publishedFacts()
		if published["Judicial_1"] && published["Judicial_2"] {
			break
		}
		if svc.gameOver() {
			break
		}
		acted := false
		if eventReady("verify_evidence") && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "verify_evidence"})
			acted = true
		}
		if eventReady("publish_evidence") && svc.agent.ActionPoints >= 20 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "publish_evidence"})
			acted = true
		}
		if !acted && !publishPending() && eventReady("investigate_dispatch") && svc.agent.ActionPoints >= 15 {
			execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "investigate_dispatch"})
		}
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance publication at step %d: %v", i, err)
		}
		peaks.observe(svc)
		observeCounters()
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
		observeCounters()
	}
	if receiver == nil || !receiver.Enabled {
		t.Fatalf("receiver verification never became available through real gates: %+v", receiver)
	}
	execute(model.AgentAction{Type: model.ActionPlayerEvent, ActionID: "RECEIVER_VERIFY_ACCOMPANY"})

	startAbs := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth
	circulationActions := 0
	attemptCirculation := func(action model.AgentAction) bool {
		t.Helper()
		accountFriction(action)
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
	// 半成品种子续播 (与参考路线 info_endgame 同一套机制)：同一次分享若因目标
	// 部门的低 receptivity/同源递减只吸收到地板下方，下个月必须优先把同一事实
	// 补过地板，而不是换种重来——换种等于把已投入的渗透窗清零。
	pendingSeed := map[string]string{}
	rotation := []string{"IT", "Supply", "Mechanical", "Medical", "Judicial", "Mayor", "Police", "Mines", "Agricultural"}
	for month := 1; month <= 96 && !svc.gameOver(); month++ {
		if _, err := svc.PassTime(1); err != nil {
			t.Fatalf("advance INFORMATION line at month %d: %v", month, err)
		}
		peaks.observe(svc)
		observeCounters()
		if svc.gameOver() {
			break
		}
		if livingChainRebellion < 0 && contentTriggered(svc, "special:living_chain_confirmed") {
			livingChainRebellion = svc.silo.Rebellion
		}
		progressResetCountermeasureForRoute(t, svc, execute)

		shareCost := model.ACTION_COSTS[model.ActionShareInfo]
		gatherCost := model.ACTION_COSTS[model.ActionGatherInfo]
		// 本部门 gather (参考路线同口径，26 §四 E)：跨部门轮转的加深吃
		// 同源递减与规范冲突双重折扣，是把新事实抬到可分享强度 (B≥0.80)
		// 的唯一稳定通道；轮转探查只作空 profession 的兜底。
		gatherFrom := svc.agent.Profession
		if gatherFrom == "" {
			gatherFrom = rotation[month%len(rotation)]
		}
		// 服务端权威成本门 (20 分册 T1-2/T2-3/T3-2)：流通摩擦让戒备部门更贵、
		// 监视期接触直接不可用 (resolveActionCost 报错)。路线按真实成本决定
		// 此刻付不付得起，避免把尝试打在必被拒的动作上 (attemptCirculation
		// 对 ExecuteAction 报错是零容忍 fatal)。Inf = 本月该部门接触不可用。
		shareCostFor := func(dept string) float64 {
			action := model.AgentAction{Type: model.ActionShareInfo, TargetDept: dept, FragmentIds: []string{"probe"}}
			if err := svc.resolveActionCost(&action); err != nil {
				return math.Inf(1)
			}
			return action.Cost
		}
		buildCostFor := func(dept string) float64 {
			action := model.AgentAction{Type: model.ActionBuildConnection, TargetDept: dept}
			if err := svc.resolveActionCost(&action); err != nil {
				return math.Inf(1)
			}
			return action.Cost
		}
		if svc.silo.Rebellion >= 0.7 {
			// 裂隙适配：叛乱过热时蛰伏 (与参考路线同策略)。
			continue
		}
		// 监视者排序 (参考路线同口径)：IT/Judicial 既是目标又是泄漏观察者，
		// 戒备上浮时最先付不起，优先保它们的槽。
		priority := func(name string) int {
			switch name {
			case "IT", "Judicial":
				return 0
			case "Mayor":
				return 1
			}
			return 2
		}
		circulated := false
		saveForPriority := false
		for svc.agent.ActionPoints >= shareCost {
			order := append([]model.Profession(nil), svc.silo.Professions...)
			sort.SliceStable(order, func(i, j int) bool {
				pi, pj := priority(order[i].Name), priority(order[j].Name)
				if pi != pj {
					return pi < pj
				}
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
				_, endangered := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts)
				pendingFact, hasPending := pendingSeed[dept.Name]
				pendingBelowFloor := false
				if hasPending {
					if belief := professionBelief(dept, pendingFact); belief == nil ||
						belief.B < engine.DefaultInformationConfig().QualRumor {
						pendingBelowFloor = true
					}
				}
				// AP 存蓄给优先部门 (参考路线同机制)：IT/Judicial 买不起时整月
				// 存蓄，不再把零散 AP 花给便宜部门——否则戒备上浮期的维护
				// share 永远差 1–2 AP，已过地板的槽就在窗口内断流。
				if priority(dept.Name) == 0 &&
					(informationActiveSlots(dept) < informationGateFactCount || endangered || pendingBelowFloor) &&
					shareCostFor(dept.Name) > svc.agent.ActionPoints {
					saveForPriority = true
					break
				}
				// Upkeep FIRST：断流风险中的连续渗透是已投入的日历，保护它
				// 优先于播新槽 (18-time-permeation §3.3 第 2 条)。
				if fragment, ok := endangeredStreakCandidate(svc.agent, dept, upkeepAttempts); ok &&
					svc.agent.ActionPoints >= shareCostFor(dept.Name) {
					upkeepAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}) {
						shared = true
						break
					}
				}
				// 三槽健康即停手；半成品种子未补过地板的除外——补完它是回收
				// 已花掉的日历。
				if informationActiveSlots(dept) >= informationGateFactCount {
					if _, waiting := pendingSeed[dept.Name]; !waiting {
						continue
					}
				}
				// 播种截止用日历而非循环序号：结算地平线 130-01 前必须走完
				// 30 月渗透窗，一次 share/gather 自带 1 月时长。
				if 130*12+1-(svc.silo.CurrentYear*12+svc.silo.CurrentMonth) <= 30 {
					continue
				}
				// 建联是 1 月时长的动作 (参考路线同结论，26 §五)：4 AP + 整月，
				// 而吸收公式里 B_player(≈0.8) 主导，起始关系 0.2 与 0.5 的单次
				// 吸收只差 ≈0.1——部门循环内建联预算全部换成分享，否则种子窗口
				// 被日历吃光；早期建联只保留月末兜底一次。
				fragment, ok := strongestShareableFactExcluding(svc.agent, dept, func(fact string) bool {
					deadStreak := func(belief *model.FactBelief) bool {
						return belief != nil && belief.B < engine.DefaultInformationConfig().QualRumor &&
							belief.PermeationCounter == 0
					}
					if shareAttempts[dept.Name+":"+fact] < 4 {
						// 陈源不播 (参考路线同策略)：部门对同源仍有活观察
						// (公告/前轮未跌破地板) 时，同源递减把再吸收压在
						// ≈0.45——30 月窗内必断流 (实测尾部门 125 波
						// Judicial_1 全部夭折)。等兜底 gather 把新事实抬到
						// 可分享强度后播新源 (参考路线建联序幕 + Judicial_5
						// 波浪同一节奏)。豁免只给"玩家播过的流死了"这一格：
						// "播过"看观察记录而非 shareAttempts——upkeep 通道
						// 走的是同一分享动作但计数在 upkeepAttempts，只看后者
						// 会让维护过的死亡槽永不获豁免 (Mayor 实测死锁)。
						// 公告夭折条目 (无 share:player 观察) 不豁免——那是
						// 抢波旧源的入口。
						for j := range dept.Beliefs {
							if dept.Beliefs[j].FactID == fact {
								attempted := false
								for _, obs := range dept.Beliefs[j].Observations {
									if strings.HasPrefix(obs, "share:player:") {
										attempted = true
										break
									}
								}
								return !(attempted && deadStreak(&dept.Beliefs[j]))
							}
						}
						return false
					}
					// 断流重置：槽已跌破地板且计数归零 = 流已死，允许重播。
					return !deadStreak(professionBelief(dept, fact))
				})
				// 半成品优先：同一事实没补过地板前不换种子。
				if pending, waiting := pendingSeed[dept.Name]; waiting {
					belief := professionBelief(dept, pending)
					if (belief == nil || (belief.B < engine.DefaultInformationConfig().QualRumor && belief.PermeationCounter == 0)) &&
						shareAttempts[dept.Name+":"+pending] < 8 {
						fragment, ok = pending, true
					} else {
						delete(pendingSeed, dept.Name)
					}
				}
				if ok {
					if agentBelief := engine.BeliefFor(svc.agent, fragment); agentBelief != nil && agentBelief.B < 0.80 &&
						svc.agent.ActionPoints >= gatherCost {
						if attemptCirculation(model.AgentAction{
							Type: model.ActionGatherInfo, TargetDept: fragmentDepartment(fragment),
						}) {
							shared = true
							break
						}
					}
					if math.IsInf(shareCostFor(dept.Name), 1) || svc.agent.ActionPoints < shareCostFor(dept.Name) {
						// 监视中/付不起这一格：换下一个部门试，别放弃整月。
						continue
					}
					shareAction := model.AgentAction{
						Type: model.ActionShareInfo, TargetDept: dept.Name,
						FragmentIds: []string{fragment},
					}
					shareAttempts[dept.Name+":"+fragment]++
					if attemptCirculation(shareAction) {
						shared = true
						// 深填 (参考路线同机制)：同月连发到 B ≥ 0.45，正好滑行
						// 整个 30 月渗透窗；浅播会在窗口内断流反而更贵。
						for shareAttempts[dept.Name+":"+fragment] < 6 &&
							svc.agent.ActionPoints >= shareCostFor(dept.Name) {
							belief := professionBelief(dept, fragment)
							if belief == nil || belief.B >= 0.45 {
								break
							}
							shareAttempts[dept.Name+":"+fragment]++
							if !attemptCirculation(shareAction) {
								break
							}
						}
						if belief := professionBelief(dept, fragment); belief == nil ||
							belief.B < engine.DefaultInformationConfig().QualRumor {
							pendingSeed[dept.Name] = fragment
						} else {
							delete(pendingSeed, dept.Name)
						}
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
		if saveForPriority {
			// 本月有意不花 AP：存给买不起的优先部门，连兜底动作也不做。
			continue
		}
		switch {
		case svc.agent.ActionPoints >= buildCostFor("Judicial") && month < 12:
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
		excluded := ""
		if svc.silo.VictoryStatus != nil && svc.silo.VictoryStatus.Decision != nil {
			excluded = svc.silo.VictoryStatus.Decision.Excluded[endingInformation]
		}
		t.Fatalf("rift route did not settle as INFORMATION at %d-%02d: excluded=%q status=%+v",
			svc.silo.CurrentYear, svc.silo.CurrentMonth, excluded, svc.silo.VictoryStatus)
	}
	settledMonths := svc.silo.CurrentYear*12 + svc.silo.CurrentMonth - startAbs
	t.Logf("rift 《共证》 reachable: seed=2401 Judicial settled=%d-%02d (%d circulation months, %d actions) rebellion peak=%.3f (≥0.7: %d months, ≥0.8: %d months) living-chain rebellion=%.3f (cap 0.8, fracture feed 0.9) panic peak=%.3f",
		svc.silo.CurrentYear, svc.silo.CurrentMonth, settledMonths, circulationActions,
		peaks.rebellionPeak, peaks.monthsRebGe07, peaks.monthsRebGe08, livingChainRebellion, peaks.panicPeak)

	// §7.3 观测账 (20 §7.3①③)：本线真实玩法必须触发到恶化两档——任一档
	// 全程零触发即显式失败 (阈值脱离玩法，回 20 §6 重调)；单授窗连续时长
	// 受 §6.3 上界纪律 (≤24 月) 约束；摩擦 AP 总额受 §6.1 预算 (≤36) 约束。
	totalCooldownGrants, totalCooldownMonths, maxGrantWindow := 0, 0, 0
	for _, st := range cooldownStats {
		totalCooldownGrants += st.grants
		totalCooldownMonths += st.months
		if st.maxGrant > maxGrantWindow {
			maxGrantWindow = st.maxGrant
		}
	}
	totalContactWatches := 0
	for _, n := range contactWatches {
		totalContactWatches += n
	}
	frictionAP := frictionBuildAP + frictionShareAP
	t.Logf("[Countermeasure] rift-info 观测账: T1-3 授窗 %d 次 / %d 部门·月 (最长单授窗 %d 月, 上界 24; 最长连续受冷 %d 月; 同月最多 %d 部门在冷却); T3-2 盯防 %d 窗 / %d 部门·月; 摩擦 AP=%.1f (建联 %.1f + 流通 %.1f, 预算 ≤36)",
		totalCooldownGrants, totalCooldownMonths, maxGrantWindow, cooldownChainMax,
		maxSimulCooldown, totalContactWatches, contactMonths,
		frictionAP, frictionBuildAP, frictionShareAP)
	if totalCooldownGrants == 0 {
		t.Fatalf("§7.3① T1-3 relation cooldown never fired in the rift information run (thresholds detached from real play)")
	}
	if totalContactWatches == 0 {
		t.Fatalf("§7.3① T3-2 contact watch never fired in the rift information run (thresholds detached from real play)")
	}
	if maxGrantWindow > 24 {
		t.Fatalf("§6.3 upper-bound discipline: T1-3 longest granted window %d months > 24", maxGrantWindow)
	}
	if frictionAP > 36 {
		t.Fatalf("§6.1 friction budget: total friction AP %.1f > 36", frictionAP)
	}
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

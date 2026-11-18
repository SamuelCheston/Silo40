package engine

import (
	"encoding/json"
	"math"
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// C8 (24 分册 §1.2/§1.3)：三通道感知 + faction: 信念事实 + 月度口碑 +
// 信息动作搭载。确定性契约：感知/发起零 RNG，玩家搭载零 RNG，无政治
// 派系零掷骰，同种子双跑逐字节一致。

// awarenessPair 构造领袖部门 (Mechanical) 与观察者部门 (Supply)，
// 默认各通道输入相同，子测试只改动一个通道验证单调性。
func awarenessPair() (*model.Silo, *ActorView, *ActorView) {
	silo := &model.Silo{
		ID: 1,
		Professions: []model.Profession{
			{
				ID: 1, Name: "Mechanical", ClassType: "COMMONER", PowerLevel: 4,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
				Relations:  map[string]float64{"Mechanical": 1.0},
			},
			{
				ID: 2, Name: "Supply", ClassType: "COMMONER", PowerLevel: 4,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
				Relations:  map[string]float64{"Supply": 1.0, "Mechanical": 0.5},
			},
		},
	}
	leader, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[0]), silo, nil)
	if err != nil {
		panic(err)
	}
	observer, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[1]), silo, nil)
	if err != nil {
		panic(err)
	}
	return silo, observer, leader
}

func TestFactionAwarenessChannelMonotonicity(t *testing.T) {
	// 关系通道：对领袖部门人脉越高，感知度越高。
	silo, observer, leader := awarenessPair()
	low := factionAwareness(silo, observer, leader)
	silo.Professions[1].Relations["Mechanical"] = 0.9
	if high := factionAwareness(silo, observer, leader); high <= low {
		t.Fatalf("关系通道不单调：0.5→%v, 0.9→%v", low, high)
	}

	// 思潮通道：(pf, loyalty) 平面越接近领袖，感知度越高。
	silo, observer, leader = awarenessPair()
	near := factionAwareness(silo, observer, leader)
	silo.Professions[1].Ideologies[model.IdeologyProForeign] = 0.1
	if far := factionAwareness(silo, observer, leader); far >= near {
		t.Fatalf("思潮通道不单调：同思潮→%v, 远思潮→%v", near, far)
	}

	// 阶层通道：权力级差越小感知度越高；同阶层有加成。
	silo, observer, leader = awarenessPair()
	silo.Professions[1].PowerLevel = 5
	closeTier := factionAwareness(silo, observer, leader)
	silo.Professions[1].PowerLevel = 9
	if farTier := factionAwareness(silo, observer, leader); farTier >= closeTier {
		t.Fatalf("阶层通道不单调：级差1→%v, 级差5→%v", closeTier, farTier)
	}
	silo.Professions[1].PowerLevel = 6 // 级差 2：tier=0.8，同阶层加成后 0.95 (未触顶)
	silo.Professions[1].ClassType = "ELITE"
	crossClass := factionAwareness(silo, observer, leader)
	silo.Professions[1].ClassType = "COMMONER"
	if sameClass := factionAwareness(silo, observer, leader); sameClass <= crossClass {
		t.Fatalf("同阶层加成缺失：同阶层→%v, 跨阶层→%v", sameClass, crossClass)
	}

	// 值域与纯函数性：任何输入 ∈ [0,1]，重复调用同值 (零 RNG)。
	silo, observer, leader = awarenessPair()
	SetRNG(rand.New(rand.NewSource(8800)))
	a := factionAwareness(silo, observer, leader)
	if a < 0 || a > 1 || math.IsNaN(a) {
		t.Fatalf("感知度越界：%v", a)
	}
	if b := factionAwareness(silo, observer, leader); a != b {
		t.Fatalf("感知度非纯函数：%v vs %v", a, b)
	}
	ref := rand.New(rand.NewSource(8800))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("感知度计算消耗了 RNG：%v != %v", got, want)
	}
}

// 发起钩子：全堡观察者写入 faction:<id>:founded；创始人 B=1；不挂 cohort
// 的正典居民与死亡居民不是代表观察者；发起零 RNG。
func TestFoundFactionSeedsAwarenessBeliefs(t *testing.T) {
	silo := foundTestSilo()
	silo.Professions = append(silo.Professions,
		model.Profession{
			ID: 2, Name: "Supply", ClassType: "COMMONER", PowerLevel: 6,
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.6, model.IdeologyLoyalty: 0.5},
			Relations:  map[string]float64{"Supply": 1.0, "Mechanical": 0.4},
		},
		model.Profession{
			ID: 3, Name: "IT", ClassType: "ELITE", PowerLevel: 9,
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.3, model.IdeologyLoyalty: 0.8},
			Relations:  map[string]float64{"IT": 1.0},
		},
	)
	silo.Cohorts = []model.PopulationCohort{{ID: 1, ProfessionID: 1, Name: "Mechanical Bloc", Count: 300}}
	silo.Residents = []model.Resident{
		{
			ID: 9, Alive: true, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical",
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.85, model.IdeologyLoyalty: 0.25},
			Relations:  map[string]float64{"Mechanical": 0.8},
		},
		{ID: 10, Alive: true, ProfessionID: 1, Profession: "Mechanical"}, // 正典具名：不挂 cohort
		{ID: 11, Alive: false, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical"},
	}
	agent := &model.Agent{ID: 42, Name: "Founder", Profession: "Mechanical", Ambition: 0.6, Influence: 0.5, ActionPoints: 25}

	const seed = int64(8801)
	SetRNG(rand.New(rand.NewSource(seed)))
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, CreateActorRefForAgent(agent, silo), MechanicMutation{Type: "faction_found_actor"})
	if len(silo.Factions) != 1 {
		t.Fatalf("founding failed: %d factions", len(silo.Factions))
	}
	ref := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("发起感知写入消耗了 RNG：%v != %v", got, want)
	}

	// C9 接口契约：事实键 faction:<派系ID>:founded。
	factID := FactionFoundedFactID(silo.Factions[0].ID)
	if factID != "faction:1:founded" {
		t.Fatalf("事实键 = %q，想要 faction:1:founded", factID)
	}
	if b := BeliefIn(&agent.Beliefs, factID); b == nil || b.B != 1 {
		t.Fatalf("创始人信念 = %+v，想要 B=1", b)
	}
	for i := range silo.Professions {
		b := BeliefIn(&silo.Professions[i].Beliefs, factID)
		if b == nil || b.B < 0 || b.B > 1 {
			t.Fatalf("部门 %s 感知信念缺失或越界：%+v", silo.Professions[i].Name, b)
		}
	}
	if b := BeliefIn(&silo.Residents[0].Beliefs, factID); b == nil || b.B <= 0 {
		t.Fatalf("居民代表感知信念缺失：%+v", b)
	}
	if b := BeliefIn(&silo.Residents[1].Beliefs, factID); b != nil {
		t.Fatalf("不挂 cohort 的正典居民不应有代表感知：%+v", b)
	}
	if b := BeliefIn(&silo.Residents[2].Beliefs, factID); b != nil {
		t.Fatalf("死亡居民不应有感知：%+v", b)
	}
}

// faction: 事实与 leak: 同豁免：永不成熟、不计共证门；普通事实不受影响。
func TestFactionFactNeverMaturesNorCounts(t *testing.T) {
	silo := &model.Silo{Professions: []model.Profession{{
		ID: 1, Name: "IT",
		Beliefs: []model.FactBelief{
			{FactID: "faction:1:founded", B: 0.9},
			{FactID: "leak:Mayor_archive", B: 0.9},
			{FactID: "Mayor_archive", B: 0.9},
		},
	}}}
	cfg := InformationConfig{PermeationMonths: 2, QualRumor: 0.3}
	for i := 0; i < 5; i++ {
		ApplyPermeation(silo, cfg)
	}
	prof := &silo.Professions[0]
	fb := BeliefIn(&prof.Beliefs, "faction:1:founded")
	if fb.Matured || fb.PermeationCounter != 0 {
		t.Fatalf("派系感知事实参与了渗透：%+v", fb)
	}
	lb := BeliefIn(&prof.Beliefs, "leak:Mayor_archive")
	if lb.Matured {
		t.Fatalf("泄漏事实成熟回归：%+v", lb)
	}
	if nb := BeliefIn(&prof.Beliefs, "Mayor_archive"); !nb.Matured {
		t.Fatalf("普通事实应照常成熟：%+v", nb)
	}
	if got := MaturedFactCount(prof); got != 1 {
		t.Fatalf("成熟事实数 = %d，想要 1 (仅普通事实)", got)
	}
	fb.Matured = true // 即使被外力置成熟也不计入共证门
	if got := MaturedFactCount(prof); got != 1 {
		t.Fatalf("强制成熟后计数 = %d，想要 1", got)
	}
}

// 月度口碑：高置信源沿关系网络传播，首跳强度 = 源 B × 0.6；多跳衰减
// 自然涌现 (Supply 达到阈值后转告 IT)。
func TestFactionGossipSpreadsWithDecay(t *testing.T) {
	silo := &model.Silo{
		ID:       1,
		Factions: []model.Faction{{ID: 1, Signature: "founded:NPC:1"}},
		Professions: []model.Profession{
			{ID: 1, Name: "Mechanical", Relations: map[string]float64{"Mechanical": 1.0, "Supply": 0.6}},
			{ID: 2, Name: "Supply", Relations: map[string]float64{"Supply": 1.0, "IT": 0.6}},
			{ID: 3, Name: "IT", Relations: map[string]float64{"IT": 1.0}},
		},
	}
	factID := FactionFoundedFactID(1)
	EnsureBeliefIn(&silo.Professions[0].Beliefs, factID).B = 0.8

	SetRNG(rand.New(rand.NewSource(8802)))
	firstHop := -1.0
	for m := 0; m < 60; m++ {
		ApplyFactionGossip(silo, nil)
		if b := BeliefIn(&silo.Professions[1].Beliefs, factID); b != nil && b.B > 0 && firstHop < 0 {
			firstHop = b.B
		}
	}
	if firstHop <= 0 {
		t.Fatal("60 月后 Supply 从未听到传闻")
	}
	if math.Abs(firstHop-0.48) > 1e-9 {
		t.Fatalf("首跳 B = %v，想要 0.8×0.6=0.48 (每跳折减)", firstHop)
	}
	if b := BeliefIn(&silo.Professions[2].Beliefs, factID); b == nil || b.B <= 0 {
		t.Fatal("IT 从未通过 Supply 听到传闻 (多跳传播缺失)")
	}
}

// 无政治派系：口碑零掷骰 (早期共享流不移动)；nil 堡不 panic。
func TestFactionGossipZeroFactionZeroRNG(t *testing.T) {
	silo := &model.Silo{
		ID:       1,
		Factions: []model.Faction{{ID: 1, Signature: unaffiliatedSignature}},
		Professions: []model.Profession{
			{ID: 1, Name: "Mechanical", Relations: map[string]float64{"Supply": 0.9}},
			{ID: 2, Name: "Supply", Relations: map[string]float64{"Mechanical": 0.9}},
		},
	}
	EnsureBeliefIn(&silo.Professions[0].Beliefs, FactionFoundedFactID(1)).B = 0.9
	const seed = int64(8805)
	SetRNG(rand.New(rand.NewSource(seed)))
	ApplyFactionGossip(silo, nil)
	ref := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("无政治派系时口碑消耗了 RNG：%v != %v", got, want)
	}
	ApplyFactionGossip(nil, nil)
}

// 搭载：玩家确定性命中且零 RNG；NPC 掷骰最终命中；无政治派系零掷骰。
func TestRumorPiggybackPlayerDeterministicNPCRolls(t *testing.T) {
	silo := &model.Silo{
		ID:       1,
		Factions: []model.Faction{{ID: 1, Signature: "founded:NPC:1"}},
		Professions: []model.Profession{
			{ID: 1, Name: "IT"},
		},
	}
	agent := &model.Agent{ID: 42, Profession: "IT"}
	e := NewGameEngine()
	playerView, err := CreateActorView(CreateActorRefForAgent(agent, silo), silo, agent)
	if err != nil {
		t.Fatal(err)
	}
	factID := FactionFoundedFactID(1)

	const seed = int64(8803)
	SetRNG(rand.New(rand.NewSource(seed)))
	for i := 0; i < 10; i++ {
		e.applyFactionRumorPiggyback(silo, playerView)
	}
	ref := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref.Float64(); got != want {
		t.Fatalf("玩家搭载消耗了 RNG：%v != %v", got, want)
	}
	if b := BeliefIn(&agent.Beliefs, factID); b == nil || b.B <= 0 {
		t.Fatalf("玩家确定性搭载未命中：%+v", b)
	}

	npcView, err := CreateActorView(CreateActorRefForProfession(&silo.Professions[0]), silo, nil)
	if err != nil {
		t.Fatal(err)
	}
	SetRNG(rand.New(rand.NewSource(seed)))
	for i := 0; i < 100; i++ {
		e.applyFactionRumorPiggyback(silo, npcView)
	}
	if b := BeliefIn(&silo.Professions[0].Beliefs, factID); b == nil || b.B <= 0 {
		t.Fatalf("NPC 搭载 100 次掷骰从未命中：%+v", b)
	}

	// 无政治派系 → NPC 也零掷骰。
	silo.Factions[0].Signature = unaffiliatedSignature
	SetRNG(rand.New(rand.NewSource(seed)))
	e.applyFactionRumorPiggyback(silo, npcView)
	ref2 := rand.New(rand.NewSource(seed))
	if got, want := DrawFloat64(), ref2.Float64(); got != want {
		t.Fatalf("无政治派系时搭载消耗了 RNG：%v != %v", got, want)
	}
}

// JS 搭载链路：GATHER_INFO 成功路径发 faction_rumor_piggyback marker →
// ApplyMutation → 玩家确定性吸收。手工政治派系 (非发起) 让玩家初始无
// 感知信念；即时结算 deltaYears=0 不跑口碑，信念增长只能来自搭载。
func TestInfoActionPiggybackViaJS(t *testing.T) {
	const seed = int64(8806)
	SetRNG(rand.New(rand.NewSource(seed)))
	e := NewGameEngine()
	silo := CreateInitialSilo("C8 Piggyback", 122, nil)
	agent := CreateInitialAgent("C8 Agent", "IT", nil, silo)
	agent.ActionPoints = 100
	silo.Factions = append(silo.Factions, model.Faction{
		ID: 7, SiloID: silo.ID, Name: "手工派系", Signature: "founded:NPC:1",
		RepresentativeKind: string(ActorNPC), RepresentativeID: 1,
	})
	factID := FactionFoundedFactID(7)
	if b := BeliefIn(&agent.Beliefs, factID); b != nil {
		t.Fatalf("player must start unaware: %+v", b)
	}
	for i := 0; i < 20; i++ {
		e.ExecuteAgentAction(silo, agent, model.AgentAction{Type: model.ActionGatherInfo, TargetDept: "Mayor", Cost: 2})
		if b := BeliefIn(&agent.Beliefs, factID); b != nil && b.B > 0 {
			return // 搭载命中
		}
	}
	t.Fatal("20 次 GATHER_INFO 后玩家仍未通过 JS 搭载听到派系传闻")
}

// 同种子双跑逐字节一致：初始化 + 发起 + 24 月演化 (口碑/搭载/NPC 决策
// 全部在共享流上)。手工成员资格让派系越过筹备期，口碑持续运行。
func TestFactionAwarenessSameSeedByteIdentical(t *testing.T) {
	const seed = int64(8804)
	run := func() []byte {
		SetRNG(rand.New(rand.NewSource(seed)))
		e := NewGameEngine()
		silo := CreateInitialSilo("C8 Awareness", 122, nil)
		agent := CreateInitialAgent("C8 Agent", "IT", nil, silo)
		agent.Ambition = 0.8
		agent.Influence = 0.6
		agent.ActionPoints = 60
		for i := range silo.Professions {
			if silo.Professions[i].Name == "IT" {
				silo.Professions[i].Ideologies[model.IdeologyProForeign] = 0.85
			}
		}
		res := e.ExecuteAgentAction(silo, agent, model.AgentAction{Type: model.ActionFoundFaction, Cost: 20})
		if !res.Executed {
			t.Fatalf("founding not executed: %s", res.Message)
		}
		foundedID := silo.Factions[len(silo.Factions)-1].ID
		silo.Cohorts[0].FactionID = uintPtr(foundedID) // C9 前的手工成员资格
		for m := 0; m < 24; m++ {
			e.UpdateSiloState(silo, 1.0/12.0, agent, "C8TEST")
		}
		out, err := json.Marshal(struct {
			Silo        *model.Silo         `json:"silo"`
			Beliefs     []model.FactBelief  `json:"agent_beliefs"`
			Connections []model.Connection  `json:"agent_connections"`
		}{silo, agent.Beliefs, agent.Connections})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return out
	}
	a, b := run(), run()
	if string(a) != string(b) {
		t.Fatalf("同种子双跑不逐字节一致 (长度 %d vs %d)", len(a), len(b))
	}
}
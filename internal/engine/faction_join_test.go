package engine

import (
	"math/rand"
	"testing"

	"silo40/internal/model"
)

// joinTestSilo 两部门 (Mechanical COMMONER PL4 / Judicial ELITE PL8) + 一个
// 居民领袖派系。思潮同轴 (pf0.9/loy0.2)，便于构造 dist=0 的亲缘圈内情形。
func joinTestSilo() *model.Silo {
	return &model.Silo{
		ID:           1,
		CurrentYear:  122,
		CurrentMonth: 5,
		Professions: []model.Profession{
			{
				ID: 1, Name: "Mechanical", ClassType: "COMMONER", PowerLevel: 4, Zone: "Lower",
				Relations:  map[string]float64{"Mechanical": 1.0, "Judicial": 0.5},
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			},
			{
				ID: 2, Name: "Judicial", ClassType: "ELITE", PowerLevel: 8, Zone: "Upper",
				Relations:  map[string]float64{"Judicial": 1.0, "Mechanical": 0.5},
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			},
		},
	}
}

// 加入概率意识形态主导：相近度^k 单调，圈外为 0，指数放大差异。
func TestJoinProbabilityIdeologyMonotonic(t *testing.T) {
	if got := joinProbability(0); got != 1 {
		t.Fatalf("dist 0 → closeness 1 → prob 1, got %v", got)
	}
	if got := joinProbability(0.15); got != 0.25 { // closeness 0.5 ^2
		t.Fatalf("dist 0.15 → prob 0.25, got %v", got)
	}
	if got := joinProbability(FactionConfig.AffinityRadius); got != 0 {
		t.Fatalf("at radius prob must be 0, got %v", got)
	}
	if got := joinProbability(0.45); got != 0 {
		t.Fatalf("beyond radius prob must be 0, got %v", got)
	}
	prev := -1.0
	for _, d := range []float64{0, 0.05, 0.1, 0.2, 0.29} {
		p := joinProbability(d)
		if prev >= 0 && p >= prev {
			t.Fatalf("join probability must strictly decrease with distance: d=%v p=%v prev=%v", d, p, prev)
		}
		prev = p
	}
	// 指数主导：k=1 时同一距离概率更高 (相近度^1 > 相近度^2)。
	saved := FactionConfig.JoinAffinityExponent
	FactionConfig.JoinAffinityExponent = 1
	linear := joinProbability(0.15)
	FactionConfig.JoinAffinityExponent = saved
	if linear <= 0.25 {
		t.Fatalf("lower exponent must raise probability: k=1 → %v, want > 0.25", linear)
	}
}

// 退出惯性：半径内无压力，刚出圈极小 (需多月)，封顶低于加入概率。
func TestExitProbabilityInertiaLowerThanJoin(t *testing.T) {
	if got := exitProbability(0.29); got != 0 {
		t.Fatalf("within radius exit prob must be 0, got %v", got)
	}
	if got := exitProbability(FactionConfig.AffinityRadius); got != 0 {
		t.Fatalf("at radius exit prob must be 0, got %v", got)
	}
	near := exitProbability(0.33) // excess 0.1 ^2 ×0.5 = 0.005
	if near <= 0 || near >= 0.01 {
		t.Fatalf("just beyond radius must be tiny (inertia), got %v", near)
	}
	far := exitProbability(0.60) // excess capped 1 ×0.5
	if far != 0.5 {
		t.Fatalf("far drift exit prob = 0.5, got %v", far)
	}
	if far >= joinProbability(0) {
		t.Fatalf("exit prob must be lower than join prob: %v vs %v", far, joinProbability(0))
	}
	saved := FactionConfig.ExitInertiaFactor
	FactionConfig.ExitInertiaFactor = 0
	if got := exitProbability(0.60); got != 1 {
		t.Fatalf("zero inertia → far exit prob 1, got %v", got)
	}
	FactionConfig.ExitInertiaFactor = saved
}

// 关系三档闸 + 跨层级/跨阶级 + 部门敌意硬否决。
func TestJoinRelationGates(t *testing.T) {
	newCase := func(leaderProfID uint, personal, dept float64) (*model.Silo, *model.Resident, *model.Faction) {
		silo := joinTestSilo()
		leader := &model.Resident{
			ID: 100, Alive: true, ProfessionID: leaderProfID, Profession: silo.Professions[leaderProfID-1].Name,
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
		}
		silo.Residents = append(silo.Residents, *leader)
		f := &model.Faction{
			ID: 7, Signature: "founded:RESIDENT:100", RepresentativeKind: string(ActorResident), RepresentativeID: 100,
		}
		silo.Factions = append(silo.Factions, *f)
		rep := &model.Resident{
			ID: 9, Alive: true, ProfessionID: 1, Profession: "Mechanical",
			Relations:       map[string]float64{"Mechanical": 1.0, "Judicial": dept},
			LeaderRelations: map[string]float64{"res:100": personal},
			Ideologies:      map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
		}
		silo.Residents = append(silo.Residents, *rep)
		return silo, &silo.Residents[len(silo.Residents)-1], &silo.Factions[0]
	}

	// 一票否决：关系 < 0.10。
	silo, rep, f := newCase(2, 0.05, 0.05)
	if _, ok := joinRelationGate(silo, rep, f, nil); ok {
		t.Fatal("relation below JoinRelationVeto must be vetoed")
	}
	// 跨阶级 (COMMONER→ELITE)：关系 0.5 < 0.60 → 拒。
	silo, rep, f = newCase(2, 0.5, 0.5)
	if _, ok := joinRelationGate(silo, rep, f, nil); ok {
		t.Fatal("cross-class relation 0.5 < 0.60 must be rejected")
	}
	// 跨阶级通过：关系 0.7 ≥ 0.60 (同时满足跨层级 0.40)。
	silo, rep, f = newCase(2, 0.7, 0.7)
	if _, ok := joinRelationGate(silo, rep, f, nil); !ok {
		t.Fatal("cross-class relation 0.7 must pass")
	}
	// 部门双向敌意硬否决：即使个人关系 0.9，部门关系 < 0.05 → 拒。
	silo, rep, f = newCase(2, 0.9, 0.9)
	silo.Professions[0].Relations["Judicial"] = 0.01
	if _, ok := joinRelationGate(silo, rep, f, nil); ok {
		t.Fatal("department hostility below BadRelationThreshold must hard-veto")
	}
	// 跨层级独立闸：同阶级 (改领袖部门为 COMMONER PL8)，关系 0.3 < 0.40 → 拒。
	silo, rep, f = newCase(2, 0.3, 0.3)
	silo.Professions[1].ClassType = "COMMONER"
	silo.Professions[1].PowerLevel = 8
	if _, ok := joinRelationGate(silo, rep, f, nil); ok {
		t.Fatal("cross-tier relation 0.3 < 0.40 must be rejected")
	}
	// 跨层级通过：关系 0.5 ≥ 0.40。
	silo, rep, f = newCase(2, 0.5, 0.5)
	silo.Professions[1].ClassType = "COMMONER"
	silo.Professions[1].PowerLevel = 8
	if _, ok := joinRelationGate(silo, rep, f, nil); !ok {
		t.Fatal("cross-tier relation 0.5 must pass")
	}
}

// 玩家直选不掷骰：RNG 流不变断言 + 成员资格/AP 结算。
func TestPlayerJoinConsumesNoRNG(t *testing.T) {
	silo := joinTestSilo()
	silo.Residents = []model.Resident{{
		ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical",
		Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
	}}
	silo.Factions = []model.Faction{{
		ID: 7, Name: "Mechanical Bloc", Signature: "founded:RESIDENT:100",
		RepresentativeKind: string(ActorResident), RepresentativeID: 100,
	}}
	agent := &model.Agent{ID: 42, Profession: "Mechanical", ActionPoints: 15}
	ref := CreateActorRefForAgent(agent, silo)

	const seed = 987654321
	SetRNG(rand.New(rand.NewSource(seed)))
	control := rng.Float64()

	SetRNG(rand.New(rand.NewSource(seed)))
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_join_actor", IntValue: 7})
	after := rng.Float64()
	if after != control {
		t.Fatalf("player direct join must not consume RNG: after=%v control=%v", after, control)
	}
	if agent.FactionID == nil || *agent.FactionID != 7 {
		t.Fatalf("player membership not set: %+v", agent.FactionID)
	}
	if agent.ActionPoints != 5 {
		t.Fatalf("AP = %v, want 15-10=5", agent.ActionPoints)
	}
	view, _ := CreateActorView(ref, silo, agent)
	if view.FactionID() == nil || *view.FactionID() != 7 {
		t.Fatal("player view must resolve joined faction")
	}
}

// 玩家「无政治派系」闸：已在其他政治派系不可重复加入；已是目标成员亦拒。
func TestPlayerJoinPoliticalFactionGate(t *testing.T) {
	silo := joinTestSilo()
	silo.Factions = []model.Faction{
		{ID: 7, Signature: "founded:RESIDENT:100", RepresentativeKind: string(ActorResident), RepresentativeID: 100},
		{ID: 8, Signature: "founded:RESIDENT:101", RepresentativeKind: string(ActorResident), RepresentativeID: 101},
	}
	agent := &model.Agent{ID: 42, Profession: "Mechanical", ActionPoints: 30, FactionID: uintPtr(7)}
	ref := CreateActorRefForAgent(agent, silo)
	e := NewGameEngine()
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_join_actor", IntValue: 8})
	if agent.FactionID == nil || *agent.FactionID != 7 || agent.ActionPoints != 30 {
		t.Fatalf("join while in another political faction must be rejected: fid=%v AP=%v", agent.FactionID, agent.ActionPoints)
	}
	// 已是目标成员 → 拒。
	e.ApplyMutation(silo, agent, ref, MechanicMutation{Type: "faction_join_actor", IntValue: 7})
	if agent.ActionPoints != 30 {
		t.Fatalf("join own faction must be rejected without AP cost: AP=%v", agent.ActionPoints)
	}
}

// 居民代表加入带上其 cohort；AP 扣除；信念/关系闸 NPC 侧生效。
func TestResidentJoinBringsCohort(t *testing.T) {
	silo := joinTestSilo()
	silo.Residents = []model.Resident{
		{ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical",
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}},
		{ID: 9, Alive: true, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical",
			ActionPoints: 20,
			Relations:    map[string]float64{"Mechanical": 0.6},
			Ideologies:   map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
			Beliefs:      []model.FactBelief{{FactID: factionFoundedFactID(7), B: 0.6}}},
	}
	silo.Cohorts = []model.PopulationCohort{{
		ID: 1, ProfessionID: 1, Count: 300, FactionID: nil,
		Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
	}}
	silo.Factions = []model.Faction{{
		ID: 7, Name: "Mechanical Bloc", Signature: "founded:RESIDENT:100",
		RepresentativeKind: string(ActorResident), RepresentativeID: 100,
	}}
	ref := CreateActorRefForResident(&silo.Residents[1])
	e := NewGameEngine()
	e.ApplyMutation(silo, nil, ref, MechanicMutation{Type: "faction_join_actor", IntValue: 7})
	if silo.Residents[1].FactionID == nil || *silo.Residents[1].FactionID != 7 {
		t.Fatalf("resident rep must join target: %+v", silo.Residents[1].FactionID)
	}
	if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != 7 {
		t.Fatalf("cohort must follow representative: %+v", silo.Cohorts[0].FactionID)
	}
	if silo.Residents[1].ActionPoints != 10 {
		t.Fatalf("AP = %v, want 20-10=10", silo.Residents[1].ActionPoints)
	}
}

// 月度加入扫描：零感知=零加入；已感知且亲缘圈内 (dist=0 → prob 1) 必加入。
func TestMonthlyJoinBeliefGated(t *testing.T) {
	build := func(belief float64) *model.Silo {
		silo := joinTestSilo()
		silo.Residents = []model.Resident{
			{ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical",
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}},
			{ID: 9, Alive: true, CohortID: uintPtr(1), ProfessionID: 1, Profession: "Mechanical",
				Relations:  map[string]float64{"Mechanical": 0.6},
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
				Beliefs:    []model.FactBelief{{FactID: factionFoundedFactID(7), B: belief}}},
		}
		silo.Cohorts = []model.PopulationCohort{{
			ID: 1, ProfessionID: 1, Count: 300,
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2},
		}}
		silo.Factions = []model.Faction{{
			ID: 7, Signature: "founded:RESIDENT:100",
			RepresentativeKind: string(ActorResident), RepresentativeID: 100,
		}}
		return silo
	}
	SetRNG(rand.New(rand.NewSource(1)))
	// 零感知 → 不加入。
	silo := build(0)
	ApplyFactionJoinDynamics(silo, nil, 1.0/12)
	if silo.Cohorts[0].FactionID != nil && politicalFactionByID(silo, *silo.Cohorts[0].FactionID) != nil {
		t.Fatal("zero awareness must not trigger join")
	}
	// 低于地板 (0.29 < 0.30) → 不加入。
	silo = build(FactionConfig.AwarenessJoinFloor - 0.01)
	ApplyFactionJoinDynamics(silo, nil, 1.0/12)
	if silo.Cohorts[0].FactionID != nil && politicalFactionByID(silo, *silo.Cohorts[0].FactionID) != nil {
		t.Fatal("belief below floor must not trigger join")
	}
	// 已感知 + dist=0 (领袖特征==cohort 特征) → prob 1 必加入；加入当月不退出。
	silo = build(0.6)
	ApplyFactionJoinDynamics(silo, nil, 1.0/12)
	if silo.Cohorts[0].FactionID == nil || *silo.Cohorts[0].FactionID != 7 {
		t.Fatalf("perceived cohort at dist 0 must join (prob 1): %+v", silo.Cohorts[0].FactionID)
	}
	if silo.Residents[1].FactionID == nil || *silo.Residents[1].FactionID != 7 {
		t.Fatal("representative must follow cohort join")
	}
	// 零月路径无操作。
	silo = build(0.6)
	ApplyFactionJoinDynamics(silo, nil, 0)
	if silo.Cohorts[0].FactionID != nil && politicalFactionByID(silo, *silo.Cohorts[0].FactionID) != nil {
		t.Fatal("zero-duration step must not advance membership")
	}
}

// 退出后当月不可重新加入 (无单月归属往返)：强制退出 (惯性=0) + 持有信念，
// 退出扫描先于加入扫描且退出者被加入扫描跳过。
func TestMonthlyExitNoSameMonthRejoin(t *testing.T) {
	silo := joinTestSilo()
	silo.Residents = []model.Resident{
		{ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical",
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}},
		{ID: 9, Alive: true, CohortID: uintPtr(2), ProfessionID: 1, Profession: "Mechanical",
			Relations:  map[string]float64{"Mechanical": 0.6},
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.1, model.IdeologyLoyalty: 0.2},
			Beliefs:    []model.FactBelief{{FactID: factionFoundedFactID(7), B: 0.9}}},
	}
	// cohort 1 锚定质心 (pf0.9 大数量)，cohort 2 漂移远 (pf0.1)。
	silo.Cohorts = []model.PopulationCohort{
		{ID: 1, ProfessionID: 1, Count: 1000, FactionID: uintPtr(7),
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}},
		{ID: 2, ProfessionID: 1, Count: 10, FactionID: uintPtr(7),
			Ideologies: map[string]float64{model.IdeologyProForeign: 0.1, model.IdeologyLoyalty: 0.2}},
	}
	silo.Factions = []model.Faction{{
		ID: 7, Signature: "founded:RESIDENT:100",
		RepresentativeKind: string(ActorResident), RepresentativeID: 100,
	}}
	saved := FactionConfig.ExitInertiaFactor
	FactionConfig.ExitInertiaFactor = 0 // 远漂移 → exitProb 1，强制退出
	defer func() { FactionConfig.ExitInertiaFactor = saved }()
	SetRNG(rand.New(rand.NewSource(3)))
	ApplyFactionJoinDynamics(silo, nil, 1.0/12)
	unaff := ensureUnaffiliatedFaction(silo)
	if silo.Cohorts[1].FactionID == nil || *silo.Cohorts[1].FactionID != unaff {
		t.Fatalf("drifted cohort must exit to unaffiliated and not rejoin same month: %+v", silo.Cohorts[1].FactionID)
	}
}

// 加入扫描按 cohort ID 升序、同种子双跑一致 (确定性契约 §6)。
func TestJoinScanDeterministicDoubleRun(t *testing.T) {
	build := func() *model.Silo {
		silo := joinTestSilo()
		silo.Residents = []model.Resident{
			{ID: 100, Alive: true, ProfessionID: 1, Profession: "Mechanical",
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.9, model.IdeologyLoyalty: 0.2}},
		}
		for _, id := range []uint{1, 2, 3} {
			silo.Cohorts = append(silo.Cohorts, model.PopulationCohort{
				ID: id, ProfessionID: 1, Count: 250,
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.85, model.IdeologyLoyalty: 0.2},
			})
			silo.Residents = append(silo.Residents, model.Resident{
				ID: 10 + id, Alive: true, CohortID: uintPtr(id), ProfessionID: 1, Profession: "Mechanical",
				Relations:  map[string]float64{"Mechanical": 0.6},
				Ideologies: map[string]float64{model.IdeologyProForeign: 0.85, model.IdeologyLoyalty: 0.2},
				Beliefs:    []model.FactBelief{{FactID: factionFoundedFactID(7), B: 0.7}},
			})
		}
		silo.Factions = []model.Faction{{
			ID: 7, Signature: "founded:RESIDENT:100",
			RepresentativeKind: string(ActorResident), RepresentativeID: 100,
		}}
		return silo
	}
	run := func() []uint {
		silo := build()
		SetRNG(rand.New(rand.NewSource(42)))
		ApplyFactionJoinDynamics(silo, nil, 1.0/12)
		var got []uint
		for _, c := range silo.Cohorts {
			if c.FactionID != nil {
				got = append(got, *c.FactionID)
			} else {
				got = append(got, 0)
			}
		}
		return got
	}
	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("double run length mismatch: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("double run diverged at cohort %d: %v vs %v", i, a, b)
		}
	}
}
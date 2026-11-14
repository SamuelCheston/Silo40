package service

import (
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"silo40/internal/model"
)

func weightedEndingFixture(trigger string) (*GameService, *model.Silo, *model.Agent) {
	station := &model.Silo{ID: 1, Name: "10 号地堡", VictoryStatus: &model.VictoryStatus{IsWon: true, Type: trigger}}
	peer := &model.Silo{ID: 2, Name: "1 号地堡", Cohesion: 0.8, Rebellion: 0.1, Traits: []string{silo1FullTrait}}
	agent := &model.Agent{ID: 1, Name: "Operative", SiloID: 1}
	svc := &GameService{
		world: &model.World{PlayerSiloID: 1, Silo1: model.Silo1State{Alive: true}},
		silo:  station, silos: []*model.Silo{station, peer}, agent: agent,
	}
	return svc, station, agent
}

func TestWeightedEndingRecordsAllEntityKinds(t *testing.T) {
	svc, station, _ := weightedEndingFixture("TIME")
	// Qualify several candidates so every entity kind can score: destruction
	// with an open snapshot (TIME + EXCLUSIONIST convergence), an escape group
	// (REBELLION), and department fragments (INFORMATION).
	svc.world.Silo1.Alive = false
	station.Silo1Destroyed = true
	station.PreFallSnapshot = model.PreFallIdeologySnapshot{Taken: true, OpenQualified: true, DominantAxis: model.IdeologyProForeign}
	station.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 120, Surviving: true}
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		// Re-anchored gate (belief layer): seed facts above the sharing floor.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
		station.Professions = append(station.Professions, prof)
	}
	station.Factions = []model.Faction{{
		ID: 2, Name: "Open Circle", Signature: "pro_foreign:High", Influence: 1, Cohesion: 1,
	}}
	station.Residents = []model.Resident{{
		ID: 3, Name: "Named Resident", Alive: true, Influence: 1,
		Ideologies: map[string]float64{model.IdeologyProForeign: 1},
	}}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil {
		t.Fatal("expected a weighted decision")
	}
	kinds := map[string]bool{}
	var total float64
	for _, contribution := range decision.Contributions {
		kinds[contribution.Kind] = true
		for _, outcome := range endingOutcomeOrder {
			total += contribution.Scores[outcome]
		}
	}
	for _, kind := range []string{"player", "faction", "resident", "silo1"} {
		if !kinds[kind] {
			t.Fatalf("missing %s contribution: %+v", kind, decision.Contributions)
		}
	}
	for _, outcome := range endingOutcomeOrder {
		var outcomeTotal float64
		for _, contribution := range decision.Contributions {
			outcomeTotal += contribution.Scores[outcome]
		}
		if math.Abs(outcomeTotal-decision.Scores[outcome]) > 1e-12 {
			t.Fatalf("score audit mismatch for %s: contributions=%v total=%v", outcome, outcomeTotal, decision.Scores[outcome])
		}
	}
	if total == 0 || decision.Selected == "" {
		t.Fatalf("empty decision: %+v", decision)
	}
}

func TestWeightedEndingFactionsAndNamedResidentsCanOverridePlayer(t *testing.T) {
	// REBELLION qualification needs a surviving escape group (G5); with it,
	// faction and resident weights can still override the player's direction.
	svc, station, _ := weightedEndingFixture("INFORMATION")
	station.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 200, Surviving: true}
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		// Re-anchored gate (belief layer): seed facts above the sharing floor.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
		station.Professions = append(station.Professions, prof)
	}
	baseline := svc.resolveWeightedEnding(station, svc.agent)
	if baseline.Selected != "INFORMATION" {
		t.Fatalf("baseline with qualified public facts should stay INFORMATION: %+v", baseline)
	}
	station.Factions = []model.Faction{{ID: 4, Name: "Revolutionaries", Influence: 20, Cohesion: 0}}
	withFaction := svc.resolveWeightedEnding(station, svc.agent)
	if withFaction.Selected != "REBELLION" || withFaction.Selected == baseline.Selected {
		t.Fatalf("faction weight did not change the outcome: baseline=%s faction=%s", baseline.Selected, withFaction.Selected)
	}

	svc2, station2, _ := weightedEndingFixture("INFORMATION")
	station2.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 200, Surviving: true}
	station2.Residents = []model.Resident{{
		ID: 5, Name: "Ambitious Resident", Alive: true, Influence: 50, Ambition: 1,
		Ideologies: map[string]float64{},
	}}
	withResident := svc2.resolveWeightedEnding(station2, svc2.agent)
	if withResident.Selected != "REBELLION" {
		t.Fatalf("named resident weight did not affect the outcome: %+v", withResident)
	}
}

func TestWeightedEndingUsesSilo1AndDeterministicTieOrder(t *testing.T) {
	// Silo 1 destruction only feeds TIME after the pre-fall ideology gate passes.
	svc, station, _ := weightedEndingFixture("TIME")
	svc.world.Silo1.Alive = false
	station.Silo1Destroyed = true
	station.PreFallSnapshot = model.PreFallIdeologySnapshot{Taken: true, OpenQualified: true, DominantAxis: model.IdeologyProForeign}
	first := svc.resolveWeightedEnding(station, svc.agent)
	second := svc.resolveWeightedEnding(station, svc.agent)
	if first.Selected != "TIME" {
		t.Fatalf("Silo 1 destruction with a qualified snapshot did not contribute to TIME: %+v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical entities produced different decisions: %+v vs %+v", first, second)
	}

	// Same destruction, failed ideology gate: TIME is excluded and Silo 1's
	// destruction contribution must not leak into scoring (结局体系 12 §6.2).
	svc2, station2, _ := weightedEndingFixture("TIME")
	svc2.world.Silo1.Alive = false
	station2.Silo1Destroyed = true
	station2.PreFallSnapshot = model.PreFallIdeologySnapshot{Taken: true, OpenQualified: false, DominantAxis: model.IdeologyLoyalty}
	gated := svc2.resolveWeightedEnding(station2, svc2.agent)
	if gated.Selected == "TIME" {
		t.Fatalf("pure-silo ideology snapshot must not qualify TIME: %+v", gated)
	}
	if gated.Scores["TIME"] != 0 {
		t.Fatalf("excluded TIME still received score: %+v", gated)
	}
}

func TestWeightedEndingTieBreaksInFixedOrder(t *testing.T) {
	// INFORMATION and REBELLION receive identical player base and faction
	// influence weights; with a surviving escape group and public facts both
	// qualify and tie, so the fixed order must pick INFORMATION. 修复7 后玩家
	// 碎片贡献读信念覆盖 (2 部门 × 3 事实 = +0.3)，用一个具名居民的野心项
	// (0.8 × 0.75 × 0.5 = +0.3) 给 REBELLION 补回同分。
	svc, station, _ := weightedEndingFixture("REBELLION")
	station.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 150, Surviving: true}
	for _, prof := range []model.Profession{{ID: 1, Name: "IT"}, {ID: 2, Name: "Judicial"}} {
		prof.KnownFragments = []string{"a", "b", "c", "d", "e"}
		// Re-anchored gate (belief layer): seed facts above the sharing floor.
		prof.Beliefs = []model.FactBelief{
			{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
		}
		station.Professions = append(station.Professions, prof)
	}
	station.Factions = []model.Faction{{ID: 7, Name: "Charter Circle", Influence: 20, Cohesion: 1}}
	station.Residents = []model.Resident{{
		ID: 8, Name: "Ambitious Balancer", Alive: true, Influence: 0.75, Ambition: 0.5,
		Ideologies: map[string]float64{},
	}}
	svc.world.Silo1.Alive = false
	if peer := svc.silos[1]; peer != nil {
		peer.Cohesion = 0
		peer.Rebellion = 0
	}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision.Scores["INFORMATION"] != decision.Scores["REBELLION"] || decision.Scores["INFORMATION"] == 0 {
		t.Fatalf("expected a non-zero tie, got %+v", decision.Scores)
	}
	if decision.Selected != "INFORMATION" {
		t.Fatalf("tie must resolve to the earlier candidate in fixed order: %+v", decision)
	}
	wantOrder := []string{"INFORMATION", "REBELLION", "RULE_OF_LAW", "TIME", "PERSONAL_RULE", "EXCLUSIONIST"}
	if !reflect.DeepEqual(endingOutcomeOrder, wantOrder) {
		t.Fatalf("tie-break order changed: %v", wantOrder)
	}
}

// TestCompletedBeliefGateOutweighsLivingSilo1Exclusionist locks 修复7: an agent
// who completes the belief gate must win on the INFORMATION weighting even with
// zero prestige and no legacy fragments — the old KnownFragments-based player
// term scored ~0 for commoner professions and let Silo 1's standing
// EXCLUSIONIST contribution (base 2 + cohesion) steal the settlement by a hair
// (职业横评 Mechanical-40: 3.985 vs 3.996).
func TestCompletedBeliefGateOutweighsLivingSilo1Exclusionist(t *testing.T) {
	svc, station, _ := weightedEndingFixture("INFORMATION")
	svc.agent.PoliticalPrestige = 0
	svc.agent.KnownFragments = nil
	names := []string{"Mayor", "Judicial", "IT", "Police", "Medical", "Supply", "Mechanical", "Mines", "Agricultural"}
	for i, name := range names {
		station.Professions = append(station.Professions, model.Profession{
			ID:   uint(i + 1),
			Name: name,
			Beliefs: []model.FactBelief{
				{FactID: "a", B: 0.5}, {FactID: "b", B: 0.5}, {FactID: "c", B: 0.5},
			},
		})
	}
	// Closed order still dominates: EXCLUSIONIST qualifies, and Silo 1 alive
	// gives it its standing base+cohesion contribution.
	station.Cohorts = []model.PopulationCohort{{
		ID: 1, Count: 100,
		Ideologies: map[string]float64{model.IdeologyLoyalty: 0.8, model.IdeologyProForeign: 0.1},
	}}

	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != endingInformation {
		t.Fatalf("completed belief gate must settle INFORMATION, got %+v", decision)
	}
	if decision.Scores[endingInformation] <= decision.Scores[endingExclusionist] {
		t.Fatalf("INFORMATION must outscore the living-Silo1 EXCLUSIONIST base: %+v", decision.Scores)
	}
}

func TestWeightedEndingAcceptsLowercaseEngineTrigger(t *testing.T) {
	svc, station, _ := weightedEndingFixture("time")
	if decision := svc.resolveWeightedEnding(station, svc.agent); decision == nil {
		t.Fatal("lowercase JS victory trigger was not arbitrated")
	}
}

func TestEndingDecisionSurvivesSaveResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "weighted-ending.db")
	svc := newTwoSiloService(t, dbPath)
	if _, err := svc.CreateGame(threeSiloRequest(103)); err != nil {
		t.Fatalf("create game: %v", err)
	}
	decision := &model.EndingDecision{
		ConfigVersion: 1, Selected: "TIME", MinimumScore: 1,
		Scores:        map[string]float64{"TIME": 4},
		Contributions: []model.EndingEntityContribution{{EntityID: "silo:1", Kind: "silo1", Name: "1 号地堡", Scores: map[string]float64{"TIME": 4}}},
	}
	svc.silo.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "TIME", Decision: decision}
	if err := svc.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	fresh := newTwoSiloService(t, dbPath)
	if err := fresh.Resume(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if fresh.silo.VictoryStatus == nil || !reflect.DeepEqual(fresh.silo.VictoryStatus.Decision, decision) {
		t.Fatalf("ending decision was not restored: %+v", fresh.silo.VictoryStatus)
	}
}

func TestWeightedEndingPreservesCatastrophicFailure(t *testing.T) {
	// P0 guardrails settle before any weighting and are never overridden.
	svc, station, _ := weightedEndingFixture("DEATH")
	station.VictoryStatus.IsWon = false
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != "DEATH" {
		t.Fatalf("death fact must settle as P0.1: %+v", decision)
	}
	for _, score := range decision.Scores {
		if score != 0 {
			t.Fatalf("hard failure must not enter weighted scoring: %+v", decision.Scores)
		}
	}

	// A surviving escape group turns the local destruction into a REBELLION
	// pyrrhic victory instead of DEATH (结局体系 12 §8).
	svc2, station2, _ := weightedEndingFixture("DEATH")
	station2.VictoryStatus.IsWon = false
	station2.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 300, Surviving: true}
	pyrrhic := svc2.resolveWeightedEnding(station2, svc2.agent)
	if pyrrhic == nil || pyrrhic.Selected != "REBELLION" {
		t.Fatalf("surviving escape group must avert DEATH into REBELLION: %+v", pyrrhic)
	}

	// AGENT_COMPROMISED still outranks the pyrrhic conversion (P0.4 precedes P1).
	svc3, station3, _ := weightedEndingFixture("AGENT_COMPROMISED")
	station3.VictoryStatus.IsWon = false
	station3.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 300, Surviving: true}
	compromised := svc3.resolveWeightedEnding(station3, svc3.agent)
	if compromised == nil || compromised.Selected != "AGENT_COMPROMISED" {
		t.Fatalf("identity exposure must outrank the pyrrhic conversion: %+v", compromised)
	}
}

func TestWeightedEndingP0FactKeyOrder(t *testing.T) {
	// reset_completed and fracture_confirmed are content fact keys; RESET
	// precedes FRACTURE, both precede AGENT_COMPROMISED and all weighting.
	svc, station, _ := weightedEndingFixture("AGENT_COMPROMISED")
	station.VictoryStatus.IsWon = false
	svc.contentStates = map[uint]map[string]model.ContentEventState{
		station.ID: {
			resetFactKey:    {Triggered: true},
			fractureFactKey: {Triggered: true},
		},
	}
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil || decision.Selected != "RESET" {
		t.Fatalf("RESET must precede FRACTURE and AGENT_COMPROMISED: %+v", decision)
	}

	// With a surviving escape group the reset demotes to a REBELLION pyrrhic
	// epilogue — unless the agent is already exposed: identity exposure (P0.4)
	// outranks every station-side convergence (结局体系 12 §8).
	station.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "TIME"}
	station.EscapedGroup = model.EscapeGroup{Initialized: true, Population: 100, Surviving: true}
	averted := svc.resolveWeightedEnding(station, svc.agent)
	if averted == nil || averted.Selected != "REBELLION" {
		t.Fatalf("escape group must avert RESET into REBELLION: %+v", averted)
	}
	station.VictoryStatus = &model.VictoryStatus{IsWon: false, Type: "AGENT_COMPROMISED"}
	compromised := svc.resolveWeightedEnding(station, svc.agent)
	if compromised == nil || compromised.Selected != "AGENT_COMPROMISED" {
		t.Fatalf("exposure must outrank the reset pyrrhic conversion: %+v", compromised)
	}
}

// personalRuleStation builds a station standing on the G12 ruling: a named
// Mayor with personal legitimacy, high police obedience, low W, and a Judicial
// in the contested state (neutral — neither IT's nor the Mayor's).
func personalRuleStation() *model.Silo {
	station := &model.Silo{ID: 1, Name: "10 号地堡"}
	station.Mayor = model.MayorOffice{Initialized: true, HolderResidentID: 7, HolderName: "Named Mayor", Legitimacy: 0.6}
	station.Authority = model.ShadowAuthority{
		Initialized: true, W: 20, PoliceObedience: 0.7,
		Judicial: model.JudicialAttachment{IT: 1, Regime: "SWING"},
	}
	return station
}

// TestPersonalRuleEligibilityFollowsRuling locks the G12 qualification: police
// obedience + Mayor legitimacy, with the Judicial's neutrality no obstacle —
// only its return to IT control blocks. No judgment reads "Judicial belongs to
// the Mayor" (02-personal-rule §4, 16-chain-personal-rule §4).
func TestPersonalRuleEligibilityFollowsRuling(t *testing.T) {
	if ok, reason := personalRuleEligible(personalRuleStation()); !ok {
		t.Fatalf("obedience + legitimacy + neutral Judicial must qualify: %s", reason)
	}

	cases := []struct {
		name   string
		mutate func(*model.Silo)
	}{
		{"Judicial back in IT hands", func(s *model.Silo) { s.Authority.Judicial.Regime = "IT" }},
		{"W still high", func(s *model.Silo) { s.Authority.W = 40 }},
		{"obedience too low", func(s *model.Silo) { s.Authority.PoliceObedience = 0.4 }},
		{"legitimacy too low", func(s *model.Silo) { s.Mayor.Legitimacy = 0.2 }},
		{"no named holder", func(s *model.Silo) { s.Mayor.HolderResidentID = 0 }},
	}
	for _, tc := range cases {
		station := personalRuleStation()
		tc.mutate(station)
		if ok, reason := personalRuleEligible(station); ok {
			t.Fatalf("%s must disqualify PERSONAL_RULE", tc.name)
		} else if reason == "" {
			t.Fatalf("%s must carry an exclusion reason", tc.name)
		}
	}

	// A locked pact does not block PERSONAL_RULE; the two convergences may
	// coexist and the fixed tie order prefers RULE_OF_LAW (02 §4 last row).
	pacted := personalRuleStation()
	pacted.Authority.Judicial.Regime = "PACT"
	pacted.Authority.Judicial.PactLocked = true
	if ok, reason := personalRuleEligible(pacted); !ok {
		t.Fatalf("pact lock must not disqualify PERSONAL_RULE: %s", reason)
	}
}

// TestWeightedEndingPersonalRuleArbitratesWithoutJudicialAttachment locks the
// end-to-end path: the ruling-state station enters the pool and wins on its
// own support, and the same station with the Judicial back in IT hands no
// longer qualifies.
func TestWeightedEndingPersonalRuleArbitratesWithoutJudicialAttachment(t *testing.T) {
	svc, station, agent := weightedEndingFixture("PERSONAL_RULE")
	*station = *personalRuleStation()
	station.VictoryStatus = &model.VictoryStatus{IsWon: true, Type: "PERSONAL_RULE"}
	agent.PoliticalPrestige = 100 // pushes the personal-rule score past the minimum

	decision := svc.resolveWeightedEnding(station, agent)
	if decision == nil || decision.Selected != "PERSONAL_RULE" {
		t.Fatalf("ruling-state station must arbitrate PERSONAL_RULE: %+v", decision)
	}

	station.Authority.Judicial.Regime = "IT"
	blocked := svc.resolveWeightedEnding(station, agent)
	if blocked != nil && blocked.Selected == "PERSONAL_RULE" {
		t.Fatalf("Judicial back in IT hands must block PERSONAL_RULE: %+v", blocked)
	}
}

// TestRuleOfLawExplanationCoversFourDimensions locks 子项 7: the RULE_OF_LAW
// arbitration carries the four-dimension audit explanation (motive /
// intervention / evidence / adhesion, 10 §9), whether or not it qualifies.
func TestRuleOfLawExplanationCoversFourDimensions(t *testing.T) {
	svc, station, agent := weightedEndingFixture("RULE_OF_LAW")
	decision := svc.resolveWeightedEnding(station, agent)
	if decision == nil {
		t.Fatal("expected a decision")
	}
	explanation := decision.Explanations["RULE_OF_LAW"]
	for _, dimension := range []string{"动机:", "介入:", "证据:", "附着:"} {
		if !strings.Contains(explanation, dimension) {
			t.Fatalf("RULE_OF_LAW explanation missing %s: %q", dimension, explanation)
		}
	}
}

func TestWeightedEndingEmptyPoolContinuesSimulation(t *testing.T) {
	// A rebellion peak without escape, fracture, or other qualified candidates
	// must not force an ending: the request is rejected and simulation goes on.
	svc, station, _ := weightedEndingFixture("REBELLION")
	station.Rebellion = 1.0
	decision := svc.resolveWeightedEnding(station, svc.agent)
	if decision == nil {
		t.Fatal("expected an audit decision even for a rejected request")
	}
	if decision.Selected != "" {
		t.Fatalf("unqualified rebellion must continue simulating, got %+v", decision)
	}
}

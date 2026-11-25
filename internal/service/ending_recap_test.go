package service

import (
	"strings"
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// recapLit 点亮一个键并指定时刻。
func recapLit(svc *GameService, key string, year, month int) {
	svc.statesMapFor(svc.silo.ID)[key] = litState(year, month)
}

// TestEndingRecapProjection locks the projection shape: 3–6 entries, sorted
// by in-game time, each carrying the date, the line, the existing event copy
// and a fact-based consequence sentence.
func TestEndingRecapProjection(t *testing.T) {
	svc := linesFixture()
	svc.contentDefinitions = []engine.ContentEventDefinition{
		{Key: "player_actions:receiver_verify_accompany", Title: "接收者当面核验", Description: "你陪接收者走完了核验全程。"},
		{Key: "special:replication_progress", Title: "副本复制推进", Description: "副本开始脱离单一载体。"},
	}
	recapLit(svc, "special:replication_progress", 123, 5)
	recapLit(svc, "player_actions:receiver_verify_accompany", 122, 3)
	recapLit(svc, "player_actions:publish_evidence", 124, 1)

	recap := svc.buildEndingRecap(svc.silo, endingInformation)
	if len(recap) != 3 {
		t.Fatalf("recap entries = %d, want 3: %+v", len(recap), recap)
	}
	first := recap[0]
	if first.Year != 122 || first.Month != 3 || first.Key != "player_actions:receiver_verify_accompany" {
		t.Fatalf("entries must be chronological, first = %+v", first)
	}
	if first.Line != model.LineInformation {
		t.Fatalf("line = %q, want information", first.Line)
	}
	if first.Title != "接收者当面核验" || first.Description != "你陪接收者走完了核验全程。" {
		t.Fatalf("must reuse existing event copy, got %+v", first)
	}
	for _, entry := range recap {
		if entry.Consequence == "" {
			t.Fatalf("every entry needs a causal sentence: %+v", entry)
		}
	}
	if recap[2].Key != "player_actions:publish_evidence" || recap[2].Year != 124 {
		t.Fatalf("last entry = %+v, want publish_evidence at Y124", recap[2])
	}
}

// TestEndingRecapAllTenEndings locks that every ending can eat the projection:
// with its causal facts lit, each produces 3–6 chronological entries.
func TestEndingRecapAllTenEndings(t *testing.T) {
	cases := []struct {
		ending string
		keys   []string
	}{
		{endingInformation, []string{"player_actions:receiver_verify_accompany", "special:replication_progress", "special:independent_receivers_met"}},
		{endingRebellion, []string{"special:departure_willingness", "special:route_confirmed", "player_actions:escape_window_orderly"}},
		{endingRuleOfLaw, []string{"special:order_chain_breach", "player_actions:charter_contact_accept", "special:charter_lock_day"}},
		{endingTime, []string{"special:open_discourse_milestone", "special:pre_fall_choice", "player_actions:pre_fall_open_inquiry"}},
		{endingPersonalRule, []string{"special:mayor_evidence_step", "special:mayor_crisis_stage", "player_actions:succession_pure_rule"}},
		{endingExclusionist, []string{normalcyFactKey, "player_actions:receiver_verify_accompany", "special:replication_progress"}},
		{endingReset, []string{resetFactKey, "special:reset_omen_census", "player_actions:reset_countermeasure_offer"}},
		{endingFracture, []string{fractureFactKey, "crisis:fracture_infrastructure", "player_actions:escape_window_riot"}},
		{endingDeath, []string{terminationFactKey, "special:last_interlock_warming", "player_actions:termination_channel_jam"}},
		{endingCompromised, []string{"special:identity_trace_accumulates", "special:archive_review_opened", "special:agent_erasure_staged"}},
	}
	for _, tc := range cases {
		svc := linesFixture()
		for i, key := range tc.keys {
			recapLit(svc, key, 122+i, 1+i)
		}
		recap := svc.buildEndingRecap(svc.silo, tc.ending)
		if len(recap) < 3 || len(recap) > 6 {
			t.Fatalf("%s: recap entries = %d, want 3–6: %+v", tc.ending, len(recap), recap)
		}
		for i := 1; i < len(recap); i++ {
			prev, cur := recap[i-1], recap[i]
			if prev.Year > cur.Year || (prev.Year == cur.Year && prev.Month > cur.Month) {
				t.Fatalf("%s: entries not chronological: %+v", tc.ending, recap)
			}
		}
		for _, entry := range recap {
			if entry.Title == "" || entry.Consequence == "" || entry.Year == 0 {
				t.Fatalf("%s: entry missing date/title/consequence: %+v", tc.ending, entry)
			}
		}
	}
}

// TestEndingRecapCapAndDegrade locks the two guardrails: never dump the whole
// log (cap 6), and sparse material degrades to nil — the settlement screen
// falls back to its existing lines instead of an empty panel.
func TestEndingRecapCapAndDegrade(t *testing.T) {
	svc := linesFixture()
	for i, def := range lineDefs {
		for j, m := range def.Milestones {
			recapLit(svc, m.Key, 122+i, 1+j)
		}
	}
	recap := svc.buildEndingRecap(svc.silo, endingInformation)
	if len(recap) != 6 {
		t.Fatalf("recap must cap at 6, got %d", len(recap))
	}

	sparse := linesFixture()
	recapLit(sparse, "special:replication_progress", 123, 5)
	if got := sparse.buildEndingRecap(sparse.silo, endingInformation); got != nil {
		t.Fatalf("sparse material must degrade to nil, got %+v", got)
	}
	empty := linesFixture()
	if got := empty.buildEndingRecap(empty.silo, endingInformation); got != nil {
		t.Fatalf("no facts must degrade to nil, got %+v", got)
	}
}

// TestEndingRecapBranchReading locks that the branch actually taken is the
// one read back: the riot window lights the rebellion milestone slot and the
// consequence says so.
func TestEndingRecapBranchReading(t *testing.T) {
	svc := linesFixture()
	recapLit(svc, "special:departure_willingness", 122, 2)
	recapLit(svc, "special:route_confirmed", 122, 6)
	recapLit(svc, "player_actions:escape_window_riot", 123, 1)
	svc.silo.EscapedGroup = model.EscapeGroup{Initialized: true, Departed: 40, Population: 31}

	recap := svc.buildEndingRecap(svc.silo, endingRebellion)
	var branch *model.EndingRecapEntry
	for i := range recap {
		if recap[i].Key == "player_actions:escape_window_riot" {
			branch = &recap[i]
		}
		if recap[i].Key == "player_actions:escape_window_orderly" {
			t.Fatal("untaken orderly branch must not be read back")
		}
	}
	if branch == nil {
		t.Fatalf("riot branch missing from recap: %+v", recap)
	}
	if branch.Line != model.LineRebellion || !strings.Contains(branch.Consequence, "暴乱") {
		t.Fatalf("branch entry wrong: %+v", branch)
	}
	if !strings.Contains(branch.Consequence, "40 人出发") || !strings.Contains(branch.Consequence, "31 人幸存") {
		t.Fatalf("consequence must read the departure facts: %+v", branch)
	}
}

// TestEndingRecapEventLogFallback locks the StoryEventLog fallback: a lit key
// without timestamp or content definition still gets its date and copy from
// the actual firing record.
func TestEndingRecapEventLogFallback(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["player_action:reset_countermeasure_done"] = model.ContentEventState{Triggered: true, TriggerCount: 1}
	recapLit(svc, resetFactKey, 125, 2)
	recapLit(svc, "special:reset_omen_census", 124, 7)
	svc.eventLogs = []model.StoryEventLog{
		{SiloID: 1, EventID: "reset_countermeasure_done", Year: 124, Month: 9, Title: "反制完成", Description: "你把副本送出了清洗半径。"},
	}

	recap := svc.buildEndingRecap(svc.silo, endingReset)
	var done *model.EndingRecapEntry
	for i := range recap {
		if recap[i].Key == "player_action:reset_countermeasure_done" {
			done = &recap[i]
		}
	}
	if done == nil {
		t.Fatalf("countermeasure completion missing from recap: %+v", recap)
	}
	if done.Year != 124 || done.Month != 9 || done.Title != "反制完成" || done.Description == "" {
		t.Fatalf("event log fallback failed: %+v", done)
	}
}

// TestFinishArbitrationRecapWiring locks the hook: a settled decision carries
// the recap and the world audit copy; the empty-pool "continue simulating"
// decision never grows one.
func TestFinishArbitrationRecapWiring(t *testing.T) {
	svc := linesFixture()
	recapLit(svc, "player_actions:receiver_verify_accompany", 122, 3)
	recapLit(svc, "special:replication_progress", 123, 5)
	recapLit(svc, "special:independent_receivers_met", 124, 1)

	settled := &model.EndingDecision{Selected: endingInformation}
	got := svc.finishArbitration(settled, svc.silo)
	if len(got.Recap) != 3 {
		t.Fatalf("settled decision must carry recap, got %+v", got.Recap)
	}
	if svc.world.LastEndingAudit != settled {
		t.Fatal("world audit copy must still be recorded")
	}

	continueSim := &model.EndingDecision{Selected: ""}
	got = svc.finishArbitration(continueSim, svc.silo)
	if got.Recap != nil {
		t.Fatalf("continue-simulating decision must not grow recap: %+v", got.Recap)
	}
}

// TestEndingRecapResourceSpending locks the resource-investment readback:
// transfer counts and the sealed-ledger settlement facts are read back per
// action; plan-role and reception entries carry their ledger facts.
func TestEndingRecapResourceSpending(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["player_actions:escape_prep_transfer_supplies"] = model.ContentEventState{
		Triggered: true, TriggerCount: 3, LastTriggeredTimestamp: gameTimestamp(123, 2),
	}
	recapLit(svc, "player_actions:escape_plan_supply", 123, 4)
	recapLit(svc, "player_actions:escape_reception_stocking", 123, 6)
	svc.silo.EscapePrep = model.EscapePreparation{
		Initialized: true, SuppliesPacked: 750, PlanRoles: []string{"supply"},
		PlanCompleteness: 1.0 / 6, ReceptionStage: "stocked",
	}

	recap := svc.buildEndingRecap(svc.silo, endingRebellion)
	byKey := map[string]model.EndingRecapEntry{}
	for _, e := range recap {
		byKey[e.Key] = e
	}
	transfer, ok := byKey["player_actions:escape_prep_transfer_supplies"]
	if !ok {
		t.Fatalf("resource spending must be read back: %+v", recap)
	}
	if !strings.Contains(transfer.Consequence, "3 次") || !strings.Contains(transfer.Consequence, "750 人·月") {
		t.Fatalf("transfer consequence must read count and ledger: %+v", transfer)
	}
	plan := byKey["player_actions:escape_plan_supply"]
	if !strings.Contains(plan.Consequence, "补给") || !strings.Contains(plan.Consequence, "功能位") {
		t.Fatalf("plan-role consequence must read the role and completeness: %+v", plan)
	}
	reception := byKey["player_actions:escape_reception_stocking"]
	if !strings.Contains(reception.Consequence, "已囤备") {
		t.Fatalf("reception consequence must read the stage: %+v", reception)
	}
}

// TestEndingRecapCrossLineJunctions locks the cross-line junction readback:
// even when the settled ending is INFORMATION, branches actually taken on
// other lines are read back, and untaken sides of each mutex group stay out.
func TestEndingRecapCrossLineJunctions(t *testing.T) {
	svc := linesFixture()
	recapLit(svc, "player_actions:charter_contact_offer", 122, 4)
	recapLit(svc, "player_actions:charter_contact_decline", 122, 5)
	recapLit(svc, "player_actions:route_survey_defer", 123, 1)
	recapLit(svc, "player_actions:pre_fall_total_riot", 124, 2)

	recap := svc.buildEndingRecap(svc.silo, endingInformation)
	byKey := map[string]model.EndingRecapEntry{}
	for _, e := range recap {
		byKey[e.Key] = e
	}
	for _, key := range []string{
		"player_actions:charter_contact_offer", "player_actions:charter_contact_decline",
		"player_actions:route_survey_defer", "player_actions:pre_fall_total_riot",
	} {
		if _, ok := byKey[key]; !ok {
			t.Fatalf("cross-line decision %s missing from recap: %+v", key, recap)
		}
	}
	for _, key := range []string{
		"player_actions:charter_contact_accept", "player_actions:charter_contact_defer",
		"player_actions:route_survey_accept", "player_actions:pre_fall_open_inquiry",
	} {
		if _, ok := byKey[key]; ok {
			t.Fatalf("untaken branch must not be read back: %s", key)
		}
	}
	if !strings.Contains(byKey["player_actions:charter_contact_decline"].Consequence, "拒绝") {
		t.Fatalf("decline branch consequence wrong: %+v", byKey["player_actions:charter_contact_decline"])
	}
	if !strings.Contains(byKey["player_actions:route_survey_defer"].Consequence, "推迟") {
		t.Fatalf("defer branch consequence wrong: %+v", byKey["player_actions:route_survey_defer"])
	}
	if !strings.Contains(byKey["player_actions:pre_fall_total_riot"].Consequence, "全面暴乱") {
		t.Fatalf("riot branch consequence wrong: %+v", byKey["player_actions:pre_fall_total_riot"])
	}
}

// TestEndingRecapCountermeasureCounts locks the countermeasure readback: each
// countermeasure reports its trigger count and whether its effect fact landed.
func TestEndingRecapCountermeasureCounts(t *testing.T) {
	svc := linesFixture()
	recapLit(svc, "player_actions:silo1_purge_countermeasure", 124, 3)
	recapLit(svc, "special:silo1_purge_softened", 124, 4)
	recapLit(svc, "player_actions:silo1_purge_plot_leak", 124, 5)
	recapLit(svc, "player_actions:termination_channel_jam", 124, 6)

	recap := svc.buildEndingRecap(svc.silo, endingReset)
	byKey := map[string]model.EndingRecapEntry{}
	for _, e := range recap {
		byKey[e.Key] = e
	}
	roster, ok := byKey["player_actions:silo1_purge_countermeasure"]
	if !ok {
		t.Fatalf("roster countermeasure missing from recap: %+v", recap)
	}
	if !strings.Contains(roster.Consequence, "触发 1 次") || !strings.Contains(roster.Consequence, "已生效") {
		t.Fatalf("roster consequence must read count and effect fact: %+v", roster)
	}
	leak := byKey["player_actions:silo1_purge_plot_leak"]
	if !strings.Contains(leak.Consequence, "触发 1 次") || !strings.Contains(leak.Consequence, "尚未被公开") {
		t.Fatalf("leak consequence must read count and pending fact: %+v", leak)
	}
	jam := byKey["player_actions:termination_channel_jam"]
	if !strings.Contains(jam.Consequence, "触发 1 次") || !strings.Contains(jam.Consequence, "未生效") {
		t.Fatalf("jam consequence must read count and pending fact: %+v", jam)
	}
}

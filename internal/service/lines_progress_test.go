package service

import (
	"testing"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// linesFixture builds a minimal service with a world clock and empty content
// states — enough for linesProgress/syncMainLine, which read no RNG.
func linesFixture() *GameService {
	svc := &GameService{
		engine:        engine.NewGameEngine(),
		contentStates: map[uint]map[string]model.ContentEventState{},
	}
	svc.silo = &model.Silo{ID: 1, Name: "Line Fixture", CurrentYear: 122, CurrentMonth: 3}
	svc.agent = &model.Agent{ID: 1, Name: "Operative", SiloID: 1}
	svc.silos = []*model.Silo{svc.silo}
	svc.world = &model.World{CurrentYear: 122, CurrentMonth: 3, Silo1: model.Silo1State{Alive: true}}
	return svc
}

func litState(year, month int) model.ContentEventState {
	return model.ContentEventState{
		Triggered:              true,
		TriggerCount:           1,
		LastTriggeredTimestamp: gameTimestamp(year, month),
	}
}

// TestDeriveMainLine locks the main-line rule: deepest milestone layer wins;
// ties resolve by recency, then by the fixed line order; no milestones means
// the main line stays undetermined.
func TestDeriveMainLine(t *testing.T) {
	line := func(id string, depth, ts int) model.LineProgress {
		return model.LineProgress{Line: id, Depth: depth, LastAdvanced: ts}
	}
	cases := []struct {
		name  string
		lines []model.LineProgress
		want  string
	}{
		{"all empty", []model.LineProgress{line(model.LineInformation, 0, 0), line(model.LineTime, 0, 0)}, ""},
		{"single lit", []model.LineProgress{line(model.LineInformation, 1, 10)}, model.LineInformation},
		{"deeper wins", []model.LineProgress{line(model.LineInformation, 1, 90), line(model.LineRuleOfLaw, 2, 10)}, model.LineRuleOfLaw},
		{"depth tie takes recency", []model.LineProgress{line(model.LineInformation, 2, 10), line(model.LineRebellion, 2, 40)}, model.LineRebellion},
		{"full tie takes fixed order", []model.LineProgress{line(model.LinePersonalRule, 1, 10), line(model.LineInformation, 1, 10)}, model.LineInformation},
	}
	for _, tc := range cases {
		if got := deriveMainLine(tc.lines); got != tc.want {
			t.Fatalf("%s: deriveMainLine = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestSyncMainLineRecordsShifts locks that the main line is persisted and
// every change appends one reroute record — including the first
// establishment (From empty) — and that the sync is idempotent.
func TestSyncMainLineRecordsShifts(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)

	// One information milestone: the main line is established.
	states["player_actions:receiver_verify_accompany"] = litState(122, 3)
	progress := svc.linesProgress()
	if progress.MainLine != model.LineInformation {
		t.Fatalf("main line = %q, want information", progress.MainLine)
	}
	if len(svc.world.LineShifts) != 1 || svc.world.LineShifts[0].From != "" ||
		svc.world.LineShifts[0].To != model.LineInformation {
		t.Fatalf("first establishment must record one shift from empty, got %+v", svc.world.LineShifts)
	}
	if shift := svc.world.LineShifts[0]; shift.Year != 122 || shift.Month != 3 {
		t.Fatalf("shift timestamp = Y%d M%d, want Y122 M3", shift.Year, shift.Month)
	}

	// Idempotent: reading again changes nothing.
	svc.linesProgress()
	if len(svc.world.LineShifts) != 1 {
		t.Fatalf("idempotent sync appended extra shifts: %+v", svc.world.LineShifts)
	}

	// Two rule-of-law milestones outrank depth 1: a visible reroute.
	states["special:order_chain_breach"] = litState(123, 5)
	states["player_actions:charter_contact_accept"] = litState(123, 6)
	progress = svc.linesProgress()
	if progress.MainLine != model.LineRuleOfLaw {
		t.Fatalf("main line after reroute = %q, want rule_of_law", progress.MainLine)
	}
	if len(svc.world.LineShifts) != 2 {
		t.Fatalf("expected 2 shifts, got %+v", svc.world.LineShifts)
	}
	shift := svc.world.LineShifts[1]
	if shift.From != model.LineInformation || shift.To != model.LineRuleOfLaw ||
		shift.Year != 122 || shift.Month != 3 {
		t.Fatalf("reroute record wrong: %+v", shift)
	}
	// The snapshot carries the persisted history for the panel's reroute log.
	if len(progress.Shifts) != 2 {
		t.Fatalf("snapshot shifts = %d, want 2", len(progress.Shifts))
	}
}

// TestSyncMainLineStopsAfterGameOver locks that the ending screen freezes the
// line picture: no reroute is recorded once the outcome is settled.
func TestSyncMainLineStopsAfterGameOver(t *testing.T) {
	svc := linesFixture()
	svc.silo.VictoryStatus = &model.VictoryStatus{Type: "INFORMATION", IsWon: true}
	svc.statesMapFor(svc.silo.ID)["player_actions:receiver_verify_accompany"] = litState(122, 3)
	progress := svc.linesProgress()
	if progress.MainLine != "" || len(svc.world.LineShifts) != 0 {
		t.Fatalf("game over must not move the main line: main=%q shifts=%+v",
			progress.MainLine, svc.world.LineShifts)
	}
}

// TestPersonalRuleRegression locks the swing-back reading: once the mayor has
// taken a public stand (milestone stays lit), losing the qualification is
// reported as regression — the road did not disappear, the order swung back.
func TestPersonalRuleRegression(t *testing.T) {
	svc := linesFixture()
	station := svc.silo
	station.Mayor.HolderResidentID = 7
	station.Mayor.HolderName = "Named Mayor"
	station.Authority.W = 20
	station.Authority.PoliceObedience = 0.7
	station.Authority.Judicial.Regime = "SWING"
	station.SetMayorLegitimacy(0.6)
	states := svc.statesMapFor(station.ID)
	states["special:mayor_public_stand"] = litState(124, 1)

	progress := svc.linesProgress()
	var line *model.LineProgress
	for i := range progress.Lines {
		if progress.Lines[i].Line == model.LinePersonalRule {
			line = &progress.Lines[i]
		}
	}
	if line == nil {
		t.Fatal("personal rule line missing from snapshot")
	}
	if !line.Qualified || line.Regressed {
		t.Fatalf("order intact: qualified=%v regressed=%v (note %q)", line.Qualified, line.Regressed, line.QualNote)
	}

	// W rises again: the stand stays lit, the qualification does not.
	station.Authority.W = 80
	progress = svc.linesProgress()
	for i := range progress.Lines {
		if progress.Lines[i].Line == model.LinePersonalRule {
			line = &progress.Lines[i]
		}
	}
	if line.Qualified {
		t.Fatal("high W must break personal-rule qualification")
	}
	if !line.Regressed {
		t.Fatalf("lit stand + lost qualification must read as regression: %+v", line)
	}
	if line.QualNote == "" {
		t.Fatal("regression must carry the concrete reason")
	}
}

// TestRuleOfLawClosedAfterWindowExpiry locks that an expired charter window
// without a lock day closes the line as a fact — not a prediction.
func TestRuleOfLawClosedAfterWindowExpiry(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:order_chain_breach"] = litState(123, 2)
	states["crisis:charter_window_expiry"] = litState(125, 9)

	progress := svc.linesProgress()
	for i := range progress.Lines {
		line := progress.Lines[i]
		if line.Line != model.LineRuleOfLaw {
			continue
		}
		if !line.Closed || line.CloseNote == "" {
			t.Fatalf("expired window without lock must close the line: %+v", line)
		}
		return
	}
	t.Fatal("rule of law line missing from snapshot")
}

// TestTimeLineClosedWhenGateFails locks that a failed pre-fall ideology gate
// closes 《No One Adjudicates》: the hard condition is fixed at the fall.
func TestTimeLineClosedWhenGateFails(t *testing.T) {
	svc := linesFixture()
	svc.silo.Silo1Destroyed = true
	svc.silo.PreFallSnapshot = model.PreFallIdeologySnapshot{Taken: true, OpenQualified: false}

	progress := svc.linesProgress()
	for i := range progress.Lines {
		line := progress.Lines[i]
		if line.Line != model.LineTime {
			continue
		}
		if !line.Closed || line.QualNote == "" {
			t.Fatalf("failed ideology gate must close the line with the reason: %+v", line)
		}
		return
	}
	t.Fatal("time line missing from snapshot")
}

// TestFailureOmensOnlyFired locks the failure-hint discipline: the snapshot
// lists only omens that already happened, with their in-game date, and never
// invents warnings for risks that have not materialized.
func TestFailureOmensOnlyFired(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:reset_omen_census"] = model.ContentEventState{
		Triggered: true, TriggerCount: 1, LastTriggeredTimestamp: gameTimestamp(124, 7),
	}
	states["special:normalcy_hardens"] = model.ContentEventState{
		Triggered: true, TriggerCount: 3, LastTriggeredTimestamp: gameTimestamp(126, 1),
	}

	progress := svc.linesProgress()
	if len(progress.Omens) != 2 {
		t.Fatalf("omens = %+v, want exactly the two fired keys", progress.Omens)
	}
	byKey := map[string]model.OmenFact{}
	for _, omen := range progress.Omens {
		byKey[omen.Key] = omen
	}
	census := byKey["special:reset_omen_census"]
	if census.Year != 124 || census.Month != 7 {
		t.Fatalf("census omen date = Y%d M%d, want Y124 M7", census.Year, census.Month)
	}
	if normalcy := byKey["special:normalcy_hardens"]; normalcy.Count != 3 {
		t.Fatalf("normalcy count = %d, want 3 (counting is the narrative)", normalcy.Count)
	}
	for key := range byKey {
		if key != "special:reset_omen_census" && key != "special:normalcy_hardens" {
			t.Fatalf("unexpected omen %q — unfired keys must stay invisible", key)
		}
	}
}

// TestDecodeGameTimestamp locks the inverse of gameTimestamp for display.
func TestDecodeGameTimestamp(t *testing.T) {
	for _, pair := range []struct{ year, month int }{{100, 1}, {122, 3}, {131, 12}} {
		year, month := decodeGameTimestamp(gameTimestamp(pair.year, pair.month))
		if year != pair.year || month != pair.month {
			t.Fatalf("decode(encode(Y%d M%d)) = Y%d M%d", pair.year, pair.month, year, month)
		}
	}
	if year, month := decodeGameTimestamp(-1); year != 0 || month != 0 {
		t.Fatalf("negative timestamp must decode to zero date, got Y%d M%d", year, month)
	}
}

// TestLineMilestoneAlternateKeys locks that mutually exclusive branches
// (orderly vs riot window) count as one milestone step: lighting either one
// advances the rebellion line by exactly one.
func TestLineMilestoneAlternateKeys(t *testing.T) {
	svc := linesFixture()
	states := svc.statesMapFor(svc.silo.ID)
	states["special:departure_willingness"] = litState(125, 2)
	states["special:route_confirmed"] = litState(125, 3)
	states["special:escape_window_open"] = litState(125, 4)
	states["player_actions:escape_window_riot"] = litState(125, 5)

	progress := svc.linesProgress()
	for i := range progress.Lines {
		line := progress.Lines[i]
		if line.Line != model.LineRebellion {
			continue
		}
		if line.Depth != 4 {
			t.Fatalf("rebellion depth = %d, want 4 (riot branch counts as the window-method step)", line.Depth)
		}
		step := line.Milestones[3]
		if !step.Lit || step.Count != 1 {
			t.Fatalf("window-method milestone must be lit by the riot branch: %+v", step)
		}
		return
	}
	t.Fatal("rebellion line missing from snapshot")
}

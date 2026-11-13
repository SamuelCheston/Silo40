package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"silo40/internal/engine"
	"silo40/internal/model"

	"gorm.io/gorm"
)

// DefaultEventWindow is the number of months after an ONCE special event
// during which decision points stay LLM-eligible at the events level.
const DefaultEventWindow = 1

// Advisor implements engine.NpcPolicy. It gates decision points by the
// player's participation level, applies/records LLM advice per mode, and
// replays recorded decisions so LLM-assisted games stay reproducible.
//
// Determinism rules enforced here:
//   - never touches the engine rng;
//   - replay forces recorded choices instead of re-consulting the model;
//   - replay divergence is a sticky explicit error, never silent fallback.
type Advisor struct {
	mu     sync.Mutex
	db     *gorm.DB
	client Client
	mode   Mode
	level  Level

	eventWindow int
	timeout     time.Duration

	// isOnceEvent reports whether a fired event is an ONCE special/crisis/
	// history event. Supplied by the service layer from content definitions.
	isOnceEvent func(category, eventID string) bool

	// Replay state: ordered log loaded at Resume; cursor advances as points
	// are consumed. Past the last entry the game continues live.
	replayEntries []DecisionLog
	replayCursor  int
	pendingReplay *DecisionLog // matched entry awaiting Committed

	// Gating state, refreshed by ObserveMonth for the following month.
	gateReason        string
	gateReasonPrevSig string
	lastOnceFireMonth int
	currentAbsMonth   int
	bands             keyState

	// Per-decision scratch state shared between Advise and Committed.
	lastRequest    string
	lastValidation string
	lastLatency    int
	lastResponse   *RawResponse
	lastUsage      Usage
	lastPrefetched bool // advice came from the prefetch cache

	shadowCh chan shadowJob
	shadowWG sync.WaitGroup

	// Prefetch (mode=on only): after each monthly tick the advisor predicts
	// next month's gated decision points and evaluates them in the background;
	// Advise only reads the cache and never blocks. A miss falls back to the
	// original weights.
	prefetch      bool
	prefetchCh    chan prefetchJob
	prefetchQuit  chan struct{} // closed on Close: workers stop picking up jobs
	prefetchWG    sync.WaitGroup
	prefetchCache map[prefetchKey]*prefetchEntry

	// Warmup: one cheap call at startup loads the model so the first gated
	// decision does not pay the cold-load cost. Outcome recorded for audit.
	warmupDone bool
	warmupErr  error

	err error // sticky fatal error (replay divergence); surfaced by Error()
}

// prefetchKey identifies one predicted decision point. The fingerprint is the
// action-type sequence only: targets are chosen by the engine rng at decision
// time and legitimately differ from the prediction.
type prefetchKey struct {
	absMonth  int
	actorKind string
	actorID   uint
	typeSeq   string
}

type prefetchEntry struct {
	advice     *engine.NpcAdvice
	resp       *RawResponse
	validation string // ok | ok_repaired
	user       string // prompt actually sent (audit hash)
	usage      Usage
}

type prefetchJob struct {
	key    prefetchKey
	system string
	user   string
	nCand  int
}

func typeSeq(infos []engine.CandidateInfo) string {
	s := ""
	for i := range infos {
		if i > 0 {
			s += "|"
		}
		s += string(infos[i].Type)
		if infos[i].ProfessionAction != "" {
			s += ":" + infos[i].ProfessionAction
		}
	}
	return s
}

// keyState holds the §9.1 key-node clauses for the upcoming month.
type keyState struct {
	RebellionHot     bool // rebellion > 0.7 (core_systems.js isRebelling)
	SafeguardHalf    bool // safeguard_risk >= 0.5 (half of DEATH trigger)
	CohesionRadical  bool // cohesion < 0.4 (radicalization threshold)
	SuspicionHalf    bool // agent suspicion >= 0.5 (half of compromise)
	PublicizePending bool // non-public faction with a representative exists
	FactionChanged   bool // faction signature/public set changed last month
}

func (k keyState) any() bool {
	return k.RebellionHot || k.SafeguardHalf || k.CohesionRadical ||
		k.SuspicionHalf || k.PublicizePending || k.FactionChanged
}

// DefaultTimeoutMS bounds the synchronous mode=on call. Local models
// (qwen3.5:4b on RTX 4060) need ~2-3s per call, so the default is sized
// for local endpoints; hosted mini-class models answer well within it.
const DefaultTimeoutMS = 8000

// Config carries advisor construction options.
type Config struct {
	DB          *gorm.DB
	Client      Client
	Mode        Mode
	Level       Level
	EventWindow int
	// TimeoutMS bounds the synchronous mode=on call. Default DefaultTimeoutMS.
	TimeoutMS int
	// Prefetch enables background evaluation of predicted next-month decision
	// points in mode=on (zero tick latency; misses fall back to the original
	// weights). Warmup issues one cheap call at startup so the first gated
	// decision does not pay the model cold-load cost.
	Prefetch bool
	Warmup   bool
}

// NewAdvisor builds an advisor. Mode off returns nil (zero footprint).
func NewAdvisor(cfg Config) (*Advisor, error) {
	if cfg.Mode == ModeOff {
		return nil, nil
	}
	if cfg.DB == nil {
		return nil, fmt.Errorf("llm: advisor requires a db for the decision log")
	}
	if err := Migrate(cfg.DB); err != nil {
		return nil, err
	}
	if cfg.EventWindow < 0 {
		cfg.EventWindow = DefaultEventWindow
	}
	if cfg.TimeoutMS <= 0 {
		cfg.TimeoutMS = DefaultTimeoutMS
	}
	a := &Advisor{
		db:                cfg.DB,
		client:            cfg.Client,
		mode:              cfg.Mode,
		level:             cfg.Level,
		eventWindow:       cfg.EventWindow,
		timeout:           time.Duration(cfg.TimeoutMS) * time.Millisecond,
		lastOnceFireMonth: -1,
		shadowCh:          make(chan shadowJob, 256),
		prefetch:          cfg.Prefetch && cfg.Mode == ModeOn,
	}
	for i := 0; i < 2; i++ {
		a.shadowWG.Add(1)
		go a.shadowWorker()
	}
	if a.prefetch {
		a.prefetchCh = make(chan prefetchJob, 256)
		a.prefetchQuit = make(chan struct{})
		a.prefetchCache = map[prefetchKey]*prefetchEntry{}
		for i := 0; i < 2; i++ {
			a.prefetchWG.Add(1)
			go a.prefetchWorker()
		}
	}
	if cfg.Warmup {
		a.Warmup()
	}
	return a, nil
}

// Close drains the shadow and prefetch workers. Pending prefetch jobs are
// discarded: they are best-effort and the game never waits on them.
func (a *Advisor) Close() {
	if a == nil {
		return
	}
	close(a.shadowCh)
	a.shadowWG.Wait()
	if a.prefetchCh != nil {
		close(a.prefetchQuit) // queued jobs are best-effort: drop them
		close(a.prefetchCh)
		a.prefetchWG.Wait()
	}
}

// Warmup loads the endpoint's model with one cheap call so the first gated
// decision does not pay the cold-load cost (which exceeds the on-call
// timeout and used to cause a guaranteed fallback per cold start). Best
// effort and async; the outcome is recorded and logged.
func (a *Advisor) Warmup() {
	if a == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		start := time.Now()
		_, _, err := a.client.Complete(ctx,
			"Connectivity probe for the Silo 40 NPC advisor. Reply with ONLY: {\"ok\":true}",
			"{}")
		a.mu.Lock()
		a.warmupDone = true
		a.warmupErr = err
		a.mu.Unlock()
		if err != nil {
			log.Printf("LLM warmup failed after %v: %v", time.Since(start).Round(time.Millisecond), err)
		} else {
			log.Printf("LLM warmup ok in %v", time.Since(start).Round(time.Millisecond))
		}
	}()
}

// WarmupStatus reports whether the startup warmup finished and its error, if
// any (debug surface).
func (a *Advisor) WarmupStatus() (done bool, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.warmupDone, a.warmupErr
}

// Error returns the sticky fatal error, if any (replay divergence). The
// service layer checks this after every month and fails explicitly.
func (a *Advisor) Error() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

// Mode/Level accessors (audit endpoints).
func (a *Advisor) GetMode() Mode {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mode
}

func (a *Advisor) GetLevel() Level {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.level
}

// SetEventFilter wires the ONCE-event predicate from content definitions.
func (a *Advisor) SetEventFilter(f func(category, eventID string) bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isOnceEvent = f
}

// SetLevel switches participation mid-game (design §11): appends a config
// entry so replay restores the switch point, then applies it.
func (a *Advisor) SetLevel(level Level, source string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return a.err
	}
	entry := &DecisionLog{
		Kind:         KindConfig,
		Year:         0,
		Month:        0,
		ConfigLevel:  string(level),
		ConfigSource: source,
		Model:        a.modelID(),
	}
	a.stampClock(entry, nil)
	if err := AppendEntry(a.db, entry); err != nil {
		return err
	}
	a.level = level
	return nil
}

// ResumeFrom loads the existing log for replay. Call after the service
// restores the game state; entries are consumed in order as play continues.
// A gapped sequence is a sticky explicit error (R2: no silent divergence).
func (a *Advisor) ResumeFrom(entries []DecisionLog) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			a.err = fmt.Errorf("llm decision log gap: entry %d has seq %d", i, e.Seq)
			return
		}
	}
	a.replayEntries = entries
	a.replayCursor = 0
	a.pendingReplay = nil
	// Apply config entries up to the current clock so the resumed level
	// matches the last recorded switch.
	for a.replayCursor < len(a.replayEntries) {
		e := a.replayEntries[a.replayCursor]
		if e.Kind != KindConfig {
			break
		}
		a.level = Level(e.ConfigLevel)
		a.replayCursor++
	}
}

// ResetForNewGame wipes replay state and the log table for a fresh game.
// The single-active-game design replaces the whole save graph on create, so
// the previous game's log belongs to a dead timeline and must not leak
// into the new game's replay.
func (a *Advisor) ResetForNewGame() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.db.Where("1 = 1").Delete(&DecisionLog{}).Error; err != nil {
		return fmt.Errorf("llm log reset: %w", err)
	}
	a.replayEntries = nil
	a.replayCursor = 0
	a.pendingReplay = nil
	a.err = nil
	a.lastOnceFireMonth = -1
	a.currentAbsMonth = 0
	a.gateReason = ""
	a.gateReasonPrevSig = ""
	a.bands = keyState{}
	a.prefetchCache = map[prefetchKey]*prefetchEntry{}
	return nil
}

// stampClock fills year/month with the decision month (the silo clock as it
// reads at decision time, before the tick advances it). Caller holds mu.
func (a *Advisor) stampClock(entry *DecisionLog, silo *model.Silo) {
	abs := a.currentAbsMonth
	if silo != nil {
		abs = decisionAbsMonth(silo)
	}
	if abs <= 0 {
		return
	}
	entry.Year = 100 + (abs / 12)
	entry.Month = abs%12 + 1
}

func decisionAbsMonth(silo *model.Silo) int {
	return (silo.CurrentYear-100)*12 + silo.CurrentMonth - 1
}

// ObserveMonth updates gating state after a monthly tick. newLogs are the
// event log entries appended during that tick. The gate computed here
// applies to the next month's decision points.
func (a *Advisor) ObserveMonth(silo *model.Silo, agent *model.Agent, newLogs []model.StoryEventLog) {
	if a == nil || silo == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return
	}
	a.currentAbsMonth = (silo.CurrentYear-100)*12 + silo.CurrentMonth - 1

	// Consume config entries that are due (replay or live switches are both
	// ordered through the log).
	for a.replayCursor < len(a.replayEntries) {
		e := a.replayEntries[a.replayCursor]
		if e.Kind != KindConfig {
			break
		}
		a.level = Level(e.ConfigLevel)
		a.replayCursor++
	}

	// ONCE special/crisis/history fires extend the event window.
	if a.isOnceEvent != nil {
		for i := range newLogs {
			l := &newLogs[i]
			if a.isOnceEvent(l.Category, l.EventID) {
				// The tick already advanced the clock; the fire belongs to
				// the month that just ended.
				a.lastOnceFireMonth = a.currentAbsMonth - 1
			}
		}
	}

	// Key-node clauses for the upcoming month (state at month end is the
	// predictor; metrics drift slowly between ticks).
	prevSig := factionSignature(silo)
	a.bands = keyState{
		RebellionHot:    silo.Rebellion > 0.7,
		SafeguardHalf:   silo.SafeguardRisk >= 0.5,
		CohesionRadical: silo.Cohesion < 0.4,
		PublicizePending: func() bool {
			for i := range silo.Factions {
				f := &silo.Factions[i]
				if !f.IsPublic && f.RepresentativeResidentID > 0 && f.Signature != "special:unaffiliated" {
					return true
				}
			}
			return false
		}(),
		FactionChanged: a.prevSig() != "" && a.prevSig() != prevSig,
	}
	if agent != nil {
		a.bands.SuspicionHalf = agent.SuspicionLevel >= 0.5
	}
	a.setPrevSig(prevSig)

	switch {
	case a.level.AtLeast(LevelFull):
		a.gateReason = "full"
	case a.level.AtLeast(LevelKey) && a.bands.any():
		a.gateReason = "key"
	case a.level.AtLeast(LevelEvent) && a.withinEventWindow():
		a.gateReason = "events"
	default:
		a.gateReason = ""
	}

	a.queuePrefetch(silo, agent)
}

// queuePrefetch (caller holds mu) predicts next month's decision points when
// the gate will be open and enqueues one background evaluation per actor.
// Pure state reads + prompt rendering happen here on the game thread; only
// the HTTP calls run in workers. Best effort: a full queue drops jobs.
func (a *Advisor) queuePrefetch(silo *model.Silo, agent *model.Agent) {
	if !a.prefetch || a.err != nil || a.gateReason == "" {
		return
	}
	nextAbs := a.currentAbsMonth + 1
	// Drop cache entries for months that are already past.
	for k := range a.prefetchCache {
		if k.absMonth < nextAbs {
			delete(a.prefetchCache, k)
		}
	}
	for _, view := range engine.DecisionActors(silo, agent) {
		infos := engine.PredictCandidates(view, silo)
		if len(infos) == 0 {
			continue
		}
		key := prefetchKey{
			absMonth:  nextAbs,
			actorKind: string(view.Ref.Kind),
			actorID:   actorID(view),
			typeSeq:   typeSeq(infos),
		}
		if _, exists := a.prefetchCache[key]; exists {
			continue
		}
		system, user := a.buildRequest(view, silo, infos)
		job := prefetchJob{key: key, system: system, user: user, nCand: len(infos)}
		select {
		case a.prefetchCh <- job:
		default: // queue full: prefetch is best-effort
		}
	}
}

// prefetchWorker evaluates predicted decision points in the background.
func (a *Advisor) prefetchWorker() {
	defer a.prefetchWG.Done()
	for job := range a.prefetchCh {
		select {
		case <-a.prefetchQuit:
			return
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		raw, usage, err := a.client.Complete(ctx, job.system, job.user)
		cancel()
		if err != nil {
			continue // decision time will miss and use the original weights
		}
		resp, perr := ParseResponse(raw, job.nCand)
		if perr != nil {
			continue
		}
		val := "ok"
		if resp.Repaired {
			val = "ok_repaired"
		}
		entry := &prefetchEntry{
			advice:     ToAdvice(resp),
			resp:       resp,
			validation: val,
			user:       job.user,
			usage:      usage,
		}
		a.mu.Lock()
		if a.prefetchCache != nil {
			a.prefetchCache[job.key] = entry
		}
		a.mu.Unlock()
	}
}

// prevSig/setPrevSig keep the faction-change detector in the struct without
// adding exported surface.
func (a *Advisor) prevSig() string     { return a.gateReasonPrevSig }
func (a *Advisor) setPrevSig(s string) { a.gateReasonPrevSig = s }

func (a *Advisor) withinEventWindow() bool {
	return a.lastOnceFireMonth >= 0 &&
		a.currentAbsMonth-a.lastOnceFireMonth <= a.eventWindow
}

func factionSignature(silo *model.Silo) string {
	sig := ""
	for i := range silo.Factions {
		sig += fmt.Sprintf("%s:%v;", silo.Factions[i].Signature, silo.Factions[i].IsPublic)
	}
	return sig
}

// Advise implements engine.NpcPolicy.
func (a *Advisor) Advise(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) *engine.NpcAdvice {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return nil
	}

	// Replay takes precedence over the current mode: recorded history must
	// reproduce even if the player (or env) now runs a different mode.
	// Entries before the current point are stale (covered by the resumed
	// checkpoint) and skipped; an entry at the current point with a
	// mismatched candidate set is an explicit divergence (R2).
	curAbs := decisionAbsMonth(silo)
	curOrder := actorOrder(string(view.Ref.Kind), actorID(view))
	for a.replayCursor < len(a.replayEntries) {
		e := a.replayEntries[a.replayCursor]
		if e.Kind == KindConfig {
			a.level = Level(e.ConfigLevel)
			a.replayCursor++
			continue
		}
		eAbs := (e.Year-100)*12 + e.Month - 1
		eOrder := actorOrder(e.ActorKind, e.ActorID)
		if eAbs < curAbs || (eAbs == curAbs && eOrder < curOrder) {
			a.replayCursor++ // past point, already covered by the checkpoint
			continue
		}
		if eAbs == curAbs && eOrder == curOrder {
			candJSON, _ := json.Marshal(ContextBuilder{}.BuildCandidateList(candidates))
			if string(candJSON) != e.CandidatesJSON {
				a.err = fmt.Errorf("llm replay divergence at %d-%02d actor %d: candidate set changed",
					e.Year, e.Month, e.ActorID)
				return nil
			}
			idx := e.FinalChoice
			a.pendingReplay = &e
			return &engine.NpcAdvice{Force: &idx}
		}
		break // entry belongs to a later point: this one was not recorded
	}

	if a.mode == ModeOff || a.level == LevelOff || a.gateReason == "" {
		return nil
	}
	if a.mode == ModeShadow {
		return nil // shadow never influences the pick
	}

	// Prefetch mode: the decision only reads the cache and never blocks the
	// tick. A miss falls back to the original weights (audit: prefetch_miss).
	if a.prefetch {
		key := prefetchKey{
			absMonth:  curAbs,
			actorKind: string(view.Ref.Kind),
			actorID:   actorID(view),
			typeSeq:   typeSeq(candidates),
		}
		if pe, ok := a.prefetchCache[key]; ok {
			delete(a.prefetchCache, key)
			a.lastRequest = pe.user
			a.lastValidation = pe.validation
			a.lastLatency = 0
			a.lastResponse = pe.resp
			a.lastUsage = pe.usage
			a.lastPrefetched = true
			return pe.advice
		}
		a.lastValidation = "prefetch_miss"
		a.lastPrefetched = false
		return nil
	}

	// mode=on: consult the client synchronously with a hard timeout. The
	// tick waits at most this budget; prefetching (§4) removes even this.
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()
	system, user := a.buildRequest(view, silo, candidates)
	start := time.Now()
	raw, usage, err := a.client.Complete(ctx, system, user)
	a.lastLatency = int(time.Since(start) / time.Millisecond)
	a.lastRequest = user
	a.lastUsage = usage
	if err != nil {
		a.lastValidation = "request_error"
		return nil
	}
	resp, err := ParseResponse(raw, len(candidates))
	if err != nil {
		a.lastValidation = "schema_invalid"
		return nil
	}
	if resp.Repaired {
		a.lastValidation = "ok_repaired"
	} else {
		a.lastValidation = "ok"
	}
	a.lastResponse = resp
	return ToAdvice(resp)
}

// Committed implements engine.NpcPolicy: records the decision point.
func (a *Advisor) Committed(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo, advice *engine.NpcAdvice, chosenIndex int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return
	}

	// Replay consumption runs before any mode check: history entries must be
	// consumed even if the current mode is off.
	if a.pendingReplay != nil {
		if chosenIndex != a.pendingReplay.FinalChoice {
			a.err = fmt.Errorf("llm replay divergence: forced choice %d but engine picked %d",
				a.pendingReplay.FinalChoice, chosenIndex)
			a.pendingReplay = nil
			return
		}
		a.pendingReplay = nil
		a.replayCursor++
		return
	}

	if a.mode == ModeOff || a.level == LevelOff || a.gateReason == "" {
		return // non-gated points reproduce naturally; no log needed
	}

	candJSON, _ := json.Marshal(ContextBuilder{}.BuildCandidateList(candidates))
	entry := &DecisionLog{
		Kind:           KindDecision,
		ActorKind:      string(view.Ref.Kind),
		ActorID:        actorID(view),
		ActorName:      view.Label(),
		CandidatesJSON: string(candJSON),
		FinalChoice:    chosenIndex,
		Level:          string(a.level),
		Model:          a.modelID(),
	}
	a.stampClock(entry, silo)

	switch a.mode {
	case ModeShadow:
		entry.Source = SourceShadow
		system, user := a.buildRequest(view, silo, candidates)
		entry.PromptHash = hashString(user)
		if err := AppendEntry(a.db, entry); err != nil {
			a.err = fmt.Errorf("llm shadow log append: %w", err)
			return
		}
		// The system prompt carries the task + output schema; dropping it
		// makes the model emit schema-invalid responses (seen as 94%
		// schema_invalid in a real gate run).
		job := shadowJob{entryID: entry.ID, system: system, user: user}
		select {
		case a.shadowCh <- job:
		default:
			// Queue full: recording the LLM side is best-effort in shadow;
			// the game-facing entry above is already persisted.
		}
	case ModeOn:
		entry.PromptHash = hashString(a.lastRequest)
		// ok_repaired advice is applied by Advise just like ok advice, so it
		// must be attributed to the LLM in the log (release audit truth).
		if (a.lastValidation == "ok" || a.lastValidation == "ok_repaired") && a.lastResponse != nil {
			if a.lastPrefetched {
				entry.Source = SourceLLMPrefetch
			} else {
				entry.Source = SourceLLM
			}
			if b, err := json.Marshal(a.lastResponse.Scores); err == nil {
				entry.ScoresJSON = string(b)
			}
			if b, err := json.Marshal(a.lastResponse.Factors); err == nil {
				entry.FactorsJSON = string(b)
			}
		} else {
			entry.Source = SourceOriginal
		}
		entry.Validation = a.lastValidation
		entry.LatencyMs = a.lastLatency
		entry.PromptTokens = a.lastUsage.PromptTokens
		entry.CompletionTokens = a.lastUsage.CompletionTokens
		if err := AppendEntry(a.db, entry); err != nil {
			a.err = fmt.Errorf("llm decision log append: %w", err)
			return
		}
	}
	a.lastValidation = ""
	a.lastResponse = nil
	a.lastRequest = ""
	a.lastLatency = 0
	a.lastUsage = Usage{}
	a.lastPrefetched = false
}

func actorID(view *engine.ActorView) uint {
	switch view.Ref.Kind {
	case engine.ActorNPC:
		return view.Ref.ProfessionID
	case engine.ActorResident:
		return view.Ref.ResidentID
	}
	return view.Ref.AgentID
}

// actorOrder reproduces RunNpcTurn's decision order within a month:
// professions first (by ID), then resident representatives.
func actorOrder(kind string, id uint) int {
	if kind == string(engine.ActorResident) {
		return 1000 + int(id)
	}
	return int(id)
}

// buildRequest renders system+user prompts. The user payload is a strict
// whitelist (design §3.3) and never includes fog-of-war data (§3.4): only
// the actor's own slice, public silo metrics, per-department public stats
// and the candidate list. Base weights are included as the alignment anchor
// (author directive after the first gate run; see BuildCandidateList).
func (a *Advisor) buildRequest(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) (string, string) {
	system := "You are the decision advisor for NPC departments in the bunker " +
		"simulation game Silo 40. You receive one NPC's situation and its " +
		"feasible candidate actions. Each candidate carries base_weight and " +
		"anchor_score: the engine's heuristic baseline (already mapped to the " +
		"0-10 scale), computed from coarse game numbers. The engine cannot read " +
		"the situation the way you can — your job is situational judgment on top " +
		"of that baseline. Score each candidate 0-10: start from anchor_score, " +
		"then move away from it whenever this NPC's specific situation (its " +
		"fragments, suspicion, prestige, propaganda level, the silo's tensions, " +
		"department panic) supports doing so. Deviating by several points is " +
		"expected when the situation is informative; copying the anchor is only " +
		"right when the situation gives no signal at all. Do NOT score by " +
		"action-type habit: gather_info, share_info, build_connection, " +
		"conduct_propaganda, incite_rebellion, publicize_faction and " +
		"profession_action are all legitimate — in particular conduct_propaganda, " +
		"share_info and publicize_faction must be judged on the situation like " +
		"any other candidate, not discounted by category. Optionally adjust " +
		"willingness factors per action type (multipliers 0.5-2); use a factor " +
		"other than 1.0 only when the situation justifies a type-level shift. " +
		"Reply with ONLY a JSON object: " +
		"{\"scores\":[{\"candidate_id\":<int>,\"score\":<0-10>}]," +
		"\"factors\":[{\"action_type\":<gather_info|share_info|build_connection|" +
		"conduct_propaganda|incite_rebellion|publicize_faction|profession_action>," +
		"\"factor\":<0.5-2>}]} covering every candidate exactly once. No prose."

	depts := make([]map[string]any, 0, len(silo.Professions))
	for i := range silo.Professions {
		p := &silo.Professions[i]
		depts = append(depts, map[string]any{
			"name": p.Name, "class": p.ClassType, "panic": round2(p.PanicValue),
			"productivity": round2(p.Productivity), "power": p.PowerLevel,
		})
	}
	factions := make([]map[string]any, 0)
	for i := range silo.Factions {
		f := &silo.Factions[i]
		// Fog filter: only public factions are visible to arbitrary NPC
		// actors here; non-public faction internals never enter prompts.
		if !f.IsPublic {
			continue
		}
		factions = append(factions, map[string]any{
			"name": f.Name, "members": f.MemberCount, "influence": round2(f.Influence),
		})
	}
	resources := map[string]float64{}
	for _, r := range silo.Resources {
		resources[r.Type] = r.Amount
	}
	payload := map[string]any{
		"actor": map[string]any{
			"profession":        view.Profession(),
			"action_points":     round2(view.ActionPoints()),
			"suspicion":         round2(view.SuspicionLevel()),
			"prestige":          round2(view.PoliticalPrestige()),
			"propaganda":        round2(view.PropagandaLevel()),
			"known_fragments":   view.KnownFragments(),
			"is_representative": view.IsRepresentative(),
		},
		"silo": map[string]any{
			"year": silo.CurrentYear, "month": silo.CurrentMonth,
			"population": silo.TotalPopulation,
			"legitimacy": round2(silo.MayorLegitimacy()), "cohesion": round2(silo.Cohesion),
			"rebellion": round2(silo.Rebellion), "resources": resources,
		},
		"departments": depts,
		"factions":    factions,
		"candidates":  ContextBuilder{}.BuildCandidateList(candidates),
	}
	user, _ := json.Marshal(payload)
	return system, string(user)
}

func (a *Advisor) modelID() string {
	if a.client == nil {
		return ""
	}
	return a.client.ModelID()
}

// shadowJob carries one async shadow evaluation.
type shadowJob struct {
	entryID uint
	system  string
	user    string
}

func (a *Advisor) shadowWorker() {
	defer a.shadowWG.Done()
	for job := range a.shadowCh {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		start := time.Now()
		raw, usage, err := a.client.Complete(ctx, job.system, job.user)
		latency := int(time.Since(start) / time.Millisecond)
		cancel()

		var scoresJSON, factorsJSON, validation string
		var shadowChoice *int
		modelID := a.modelID()
		if err != nil {
			validation = "request_error"
		} else {
			nCandidates := countCandidates(job.user)
			resp, perr := ParseResponse(raw, nCandidates)
			if perr != nil {
				validation = "schema_invalid"
				// Diagnose gate failures: keep a truncated sample of what the
				// model actually returned.
				log.Printf("llm shadow schema_invalid (%v, %d candidates): %s",
					perr, nCandidates, truncate(raw, 300))
			} else {
				if resp.Repaired {
					validation = "ok_repaired"
				} else {
					validation = "ok"
				}
				if b, merr := json.Marshal(resp.Scores); merr == nil {
					scoresJSON = string(b)
				}
				if b, merr := json.Marshal(resp.Factors); merr == nil {
					factorsJSON = string(b)
				}
				if idx, ok := topChoice(resp); ok {
					shadowChoice = &idx
				}
			}
		}
		a.mu.Lock()
		db := a.db
		a.mu.Unlock()
		// Retry transient SQLite contention (game thread also writes); a lost
		// shadow evaluation would stall any pacing that waits on it.
		var werr error
		for attempt := 0; attempt < 3; attempt++ {
			werr = UpdateShadowFields(db, job.entryID, scoresJSON, factorsJSON, modelID, validation,
				shadowChoice, latency, usage.PromptTokens, usage.CompletionTokens)
			if werr == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if werr != nil {
			log.Printf("llm shadow log update failed for entry %d: %v", job.entryID, werr)
		}
	}
}

func countCandidates(user string) int {
	var payload struct {
		Candidates []json.RawMessage `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		return 0
	}
	return len(payload.Candidates)
}

func topChoice(resp *RawResponse) (int, bool) {
	best, bestScore, ok := 0, -1.0, false
	for _, s := range resp.Scores {
		if s.Score > bestScore {
			best, bestScore, ok = s.CandidateID, s.Score, true
		}
	}
	return best, ok
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func hashString(s string) string {
	if s == "" {
		return ""
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

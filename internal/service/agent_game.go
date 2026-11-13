package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// ============ Agent operator mode ============
//
// An external AI agent (with tool-calling ability) plays the game over the
// debug HTTP surface instead of an OpenAI-compatible completion endpoint.
// The monthly tick pauses at every NPC decision point; the agent receives
// the actor's situation plus the executable candidates and submits one
// strict choice (candidate_id + per-candidate scores + optional rationale).
// The choice enters the engine through NpcAdvice.Force, so the picked
// candidate is exactly what executes (the weighted rng draw is still
// consumed, keeping the rng stream order intact).
//
// Every decision becomes one agentRecord in the exact shape the
// silo40-npc-ml data pipeline consumes (pipeline.py validate_raw contract):
// prompt (state snapshot) + candidates (action_id/args/executable and the
// four regression numbers) + executed_action + response. Records are kept
// in memory and exported as JSONL over HTTP; the SQLite llm_decision_logs
// table is intentionally NOT reused because its schema is the LLM-advisor
// scores/factors shape, not the pipeline's executed-action shape.

const agentDecisionVersion = "agent-operator-v1"

// Execution statuses recorded on agent records. Only "executed" passes the
// pipeline; every other value lands in discarded.jsonl with a reason.
const (
	agentStatusExecuted = "executed"
	agentStatusFailed   = "execution_failed"
	agentStatusTimeout  = "timeout_fallback"
	agentStatusAborted  = "aborted"
)

// EnvAgentTimeoutMS bounds how long the tick waits at a decision point for
// the agent's choice before the session fails (records stay exportable).
const EnvAgentTimeoutMS = "SILO40_AGENT_TIMEOUT_MS"

// DefaultAgentChooseTimeout is generous: the agent is an AI operator that
// reads state and reasons, not a low-latency completion endpoint.
const DefaultAgentChooseTimeout = 30 * time.Minute

// AgentStartRequest configures a new agent-played game.
type AgentStartRequest struct {
	Seed         int64  `json:"seed"`
	SiloName     string `json:"silo_name"`
	AgentName    string `json:"agent_name"`
	Profession   string `json:"profession"`
	IncludeSilo1 bool   `json:"include_silo1"`
	Neighbor     bool   `json:"neighbor"`
	// TimeoutMS overrides the per-decision wait budget (0 = env/default).
	TimeoutMS int `json:"timeout_ms"`
}

// AgentCandidate is one executable candidate as the agent sees it.
type AgentCandidate struct {
	CandidateID int               `json:"candidate_id"`
	ActionID    string            `json:"action_id"`
	Args        map[string]string `json:"args"`
	Description string            `json:"description"`
	APCost      float64           `json:"ap_cost"`
	BaseWeight  float64           `json:"base_weight"`
	AnchorScore float64           `json:"anchor_score"`
}

// AgentDecisionContext is delivered at a decision point. The state slices
// use the same fog-filter whitelist as the LLM advisor prompts: only the
// actor's own slice, public silo metrics, per-department public stats and
// public factions.
type AgentDecisionContext struct {
	DecisionID  string           `json:"decision_id"`
	Year        int              `json:"year"`
	Month       int              `json:"month"`
	Actor       map[string]any   `json:"actor"`
	Silo        map[string]any   `json:"silo"`
	Departments []map[string]any `json:"departments"`
	Factions    []map[string]any `json:"factions"`
	Candidates  []AgentCandidate `json:"candidates"`
}

// AgentChooseRequest is the agent's strict decision payload.
type AgentChooseRequest struct {
	DecisionID string       `json:"decision_id"`
	Candidate  int          `json:"candidate_id"`
	Scores     []AgentScore `json:"scores"`
	Rationale  string       `json:"rationale"`
}

// AgentScore is one candidate's 0-10 score; the full set must cover every
// candidate exactly once (same coverage rule as the LLM advisor schema).
type AgentScore struct {
	CandidateID int     `json:"candidate_id"`
	Score       float64 `json:"score"`
}

// AgentMonthSummary is returned whenever a month completes (possibly after
// several decisions inside it).
type AgentMonthSummary struct {
	Status     string   `json:"status"` // month_end | game_over
	Year       int      `json:"year"`
	Month      int      `json:"month"`
	GameOver   bool     `json:"game_over"`
	Ending     string   `json:"ending,omitempty"`
	Population int      `json:"population"`
	Rebellion  float64  `json:"rebellion"`
	Cohesion   float64  `json:"cohesion"`
	Legitimacy float64  `json:"legitimacy"`
	Logs       []string `json:"logs"`
	// Counters for the operator loop.
	DecisionsRecorded int `json:"decisions_recorded"`
	MonthsElapsed     int `json:"months_elapsed"`
}

// agentAction is the canonical strict action shape ({"action_id","args"})
// used for both executed_action and response in records.
type agentAction struct {
	ActionID string            `json:"action_id"`
	Args     map[string]string `json:"args"`
}

// agentCandidateRecord is one candidate inside a record: the public fields
// plus the four regression numbers the pipeline requires.
type agentCandidateRecord struct {
	ActionID        string            `json:"action_id"`
	Args            map[string]string `json:"args"`
	Executable      bool              `json:"executable"`
	OriginalWeight  float64           `json:"original_weight"`
	ModelScore      float64           `json:"model_score"`
	Willingness     float64           `json:"willingness"`
	FinalMultiplier float64           `json:"final_multiplier"`
}

// agentRecord is one decision record in the pipeline's validate_raw shape.
// JSON keys are the pipeline's normalized record keys (the same ones the
// SQLite export path produces from *_json columns).
type agentRecord struct {
	DecisionID      string                 `json:"decision_id"`
	GameID          string                 `json:"game_id"`
	OccurredAt      string                 `json:"occurred_at"`
	NpcID           string                 `json:"npc_id"`
	DecisionVersion string                 `json:"decision_version"`
	Prompt          string                 `json:"prompt"`
	Candidates      []agentCandidateRecord `json:"candidates"`
	ExecutedAction  *agentAction           `json:"executed_action"`
	ExecutionStatus string                 `json:"execution_status"`
	ExecutionReason string                 `json:"execution_reason,omitempty"`
	Response        *agentAction           `json:"response"`
}

type agentPhase int

const (
	agentPhaseNone           agentPhase = iota
	agentPhaseIdle                      // between months: waiting for step
	agentPhaseRunning                   // tick inside the current step/choose call
	agentPhaseAwaitingChoice            // parked at a decision point
	agentPhaseGameOver
	agentPhaseStopped
	agentPhaseError
)

func (p agentPhase) String() string {
	switch p {
	case agentPhaseIdle:
		return "idle"
	case agentPhaseRunning:
		return "running"
	case agentPhaseAwaitingChoice:
		return "awaiting_choice"
	case agentPhaseGameOver:
		return "game_over"
	case agentPhaseStopped:
		return "stopped"
	case agentPhaseError:
		return "error"
	}
	return "none"
}

type agentParkKind int

const (
	parkDecision agentParkKind = iota
	parkMonth
	parkError
)

type agentPark struct {
	kind    agentParkKind
	ctx     *AgentDecisionContext
	summary *AgentMonthSummary
	err     error
}

type agentDelivery struct {
	chosenIdx int
	scores    []AgentScore // ordered by candidate_id
	rationale string
}

type pendingAgentDecision struct {
	seq       int
	ctx       *AgentDecisionContext
	infos     []engine.CandidateInfo
	record    *agentRecord
	delivered bool
	delivery  agentDelivery
	doneCh    chan struct{}
}

// AgentSession drives one agent-played game. The engine tick runs in a
// dedicated goroutine; at each decision point it parks until the agent
// submits a choice (or the wait budget elapses, which ends the session).
// All GameService calls happen on the runner goroutine, so the service
// mutex contract is untouched by the HTTP side.
type AgentSession struct {
	svc *GameService

	mu      sync.Mutex
	phase   agentPhase
	gameID  string
	seed    int64
	timeout time.Duration

	seq            int
	months         int
	records        []agentRecord
	pending        *pendingAgentDecision
	awaitingCommit *pendingAgentDecision // delivered, engine has not committed yet
	lastCommitted  *agentRecord          // committed, awaiting execution result
	errMsg         string

	// parkCh is buffered so the final month park of a stopped/errored
	// session cannot wedge the runner when no controller consumes it.
	parkCh chan agentPark
	stepCh chan struct{}
	quitCh chan struct{}
	doneCh chan struct{}
}

// agentPolicy adapts the session to the engine hooks.
type agentPolicy struct{ s *AgentSession }

func (p *agentPolicy) Advise(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) *engine.NpcAdvice {
	return p.s.advise(view, silo, candidates)
}

func (p *agentPolicy) Committed(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo, advice *engine.NpcAdvice, chosenIndex int) {
	p.s.committed(advice, chosenIndex)
}

func (p *agentPolicy) NpcExecuted(view *engine.ActorView, silo *model.Silo, action model.AgentAction, result model.ActionResult) {
	p.s.npcExecuted(action, result)
}

// StartAgentGame creates a fresh game and hands every NPC decision point to
// the agent. Fails while a previous session is still unfinished.
func (s *GameService) StartAgentGame(req AgentStartRequest) (any, error) {
	if prev := s.getAgentSession(); prev != nil && !prev.terminated() {
		return nil, fmt.Errorf("an agent session is still active; call /debug/agent/stop first")
	}

	createReq := model.CreateGameRequest{
		SiloName:     req.SiloName,
		AgentName:    req.AgentName,
		Profession:   req.Profession,
		Seed:         req.Seed,
		IncludeSilo1: req.IncludeSilo1,
	}
	if req.Neighbor {
		createReq.Neighbor = &model.NeighborSiloConfig{}
	}
	state, err := s.CreateGame(createReq)
	if err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	timeout := DefaultAgentChooseTimeout
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
	} else if v := os.Getenv(EnvAgentTimeoutMS); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}

	seed := req.Seed
	if state != nil && state.Silo.RNGSeed != 0 {
		seed = state.Silo.RNGSeed
	}
	sess := &AgentSession{
		svc:     s,
		phase:   agentPhaseIdle,
		seed:    seed,
		gameID:  fmt.Sprintf("agent-%d-%s", seed, time.Now().Format("20060102T150405")),
		timeout: timeout,
		parkCh:  make(chan agentPark, 2),
		stepCh:  make(chan struct{}),
		quitCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	s.mu.Lock()
	// Replace whatever bindEngineRuntime installed (LLM advisor or nil):
	// for this game the agent IS the NPC policy.
	s.engine.NpcPolicy = &agentPolicy{s: sess}
	s.mu.Unlock()

	s.agentMu.Lock()
	s.agentSession = sess
	s.agentMu.Unlock()

	go sess.runner()
	return sess.status(), nil
}

// AgentStep advances the simulation one month, returning at the first
// decision point (status "decision") or at month end (status "month_end" /
// "game_over").
func (s *GameService) AgentStep() (any, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return nil, fmt.Errorf("no agent session; call /debug/agent/start first")
	}
	return sess.step()
}

// AgentChoose delivers the agent's decision for the pending point and then
// behaves like step: returns at the next decision point or month end.
func (s *GameService) AgentChoose(req AgentChooseRequest) (any, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return nil, fmt.Errorf("no agent session; call /debug/agent/start first")
	}
	return sess.choose(req)
}

// AgentStatus reports the session state for the operator loop.
func (s *GameService) AgentStatus() (any, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return map[string]any{"active": false}, nil
	}
	return sess.status(), nil
}

// AgentExport returns the game's decision records.
func (s *GameService) AgentExport() (any, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return nil, fmt.Errorf("no agent session")
	}
	return sess.export(), nil
}

// AgentExportJSONL returns the records as pipeline-ready JSONL text (one
// validate_raw-shaped record per line).
func (s *GameService) AgentExportJSONL() (string, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return "", fmt.Errorf("no agent session")
	}
	return sess.exportJSONL(), nil
}

// AgentStop ends the session. A pending decision is recorded as aborted and
// the current month finishes on the original weights; records stay
// exportable afterwards.
func (s *GameService) AgentStop() (any, error) {
	sess := s.getAgentSession()
	if sess == nil {
		return nil, fmt.Errorf("no agent session")
	}
	sess.stop()
	return sess.status(), nil
}

func (s *GameService) getAgentSession() *AgentSession {
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	return s.agentSession
}

// terminated reports whether the session reached a terminal phase.
func (a *AgentSession) terminated() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.phase == agentPhaseGameOver || a.phase == agentPhaseStopped || a.phase == agentPhaseError
}

// runner owns all GameService calls (PassTime). It waits for step signals,
// runs one month, and reports parks; decision parks are emitted from inside
// advise() on this same goroutine.
func (a *AgentSession) runner() {
	defer close(a.doneCh)
	defer func() {
		if r := recover(); r != nil {
			a.mu.Lock()
			a.phase = agentPhaseError
			a.errMsg = fmt.Sprintf("runner panic: %v", r)
			a.mu.Unlock()
			select {
			case a.parkCh <- agentPark{kind: parkError, err: fmt.Errorf("runner panic: %v", r)}:
			default:
			}
		}
	}()
	for {
		select {
		case <-a.quitCh:
			return
		case <-a.stepCh:
		}

		a.mu.Lock()
		a.phase = agentPhaseRunning
		a.mu.Unlock()

		result, err := a.svc.PassTime(1)
		if err != nil {
			a.mu.Lock()
			a.phase = agentPhaseError
			a.errMsg = err.Error()
			a.mu.Unlock()
			a.parkCh <- agentPark{kind: parkError, err: err}
			return
		}

		a.mu.Lock()
		a.months++
		summary := a.buildSummaryLocked(result)
		// A timeout may have flipped the phase to error mid-month; do not
		// paper over it with idle.
		if a.phase == agentPhaseRunning {
			if result.GameOver {
				a.phase = agentPhaseGameOver
			} else {
				a.phase = agentPhaseIdle
			}
		}
		a.mu.Unlock()
		a.parkCh <- agentPark{kind: parkMonth, summary: summary}
	}
}

// advise implements the parking side of engine.NpcPolicy.Advise. Called on
// the runner goroutine inside PassTime.
func (a *AgentSession) advise(view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) *engine.NpcAdvice {
	a.mu.Lock()
	switch a.phase {
	case agentPhaseStopped, agentPhaseError, agentPhaseGameOver:
		a.mu.Unlock()
		return nil
	}
	a.seq++
	seq := a.seq
	pending := &pendingAgentDecision{
		seq:    seq,
		ctx:    a.buildContextLocked(seq, view, silo, candidates),
		infos:  candidates,
		record: a.buildRecordScaffoldLocked(seq, view, silo, candidates),
		doneCh: make(chan struct{}),
	}
	a.pending = pending
	a.phase = agentPhaseAwaitingChoice
	a.mu.Unlock()

	// Hand the decision to the operator and park the tick.
	a.parkCh <- agentPark{kind: parkDecision, ctx: pending.ctx}

	timer := time.NewTimer(a.timeout)
	defer timer.Stop()
	select {
	case <-pending.doneCh:
	case <-timer.C:
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending != pending {
		// stop() already recorded this decision as aborted.
		return nil
	}
	if pending.delivered {
		a.pending = nil
		a.awaitingCommit = pending
		a.phase = agentPhaseRunning
		a.applyDeliveryLocked(pending)
		idx := pending.delivery.chosenIdx
		return &engine.NpcAdvice{Force: &idx}
	}
	// Timeout: the operator is gone. Keep the record (status marks it),
	// end the session explicitly, and let the original weights finish the
	// month; the pipeline discards the sample.
	a.pending = nil
	a.phase = agentPhaseError
	a.errMsg = fmt.Sprintf("agent choose timeout after %v at decision %s; session ended, records preserved",
		a.timeout, pending.record.DecisionID)
	pending.record.ExecutionStatus = agentStatusTimeout
	a.records = append(a.records, *pending.record)
	return nil
}

// committed implements engine.NpcPolicy.Committed: appends the record for
// agent-forced decisions at the engine's official commit point.
func (a *AgentSession) committed(advice *engine.NpcAdvice, chosenIndex int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p := a.awaitingCommit
	if advice == nil || advice.Force == nil || p == nil {
		// Original-weights decisions (timeout/abort/stop) were recorded at
		// the point they happened; nothing to append here.
		return
	}
	a.awaitingCommit = nil
	if chosenIndex != p.delivery.chosenIdx {
		p.record.ExecutionStatus = agentStatusFailed
		p.record.ExecutionReason = fmt.Sprintf("engine picked %d, agent forced %d", chosenIndex, p.delivery.chosenIdx)
	}
	a.records = append(a.records, *p.record)
	a.lastCommitted = &a.records[len(a.records)-1]
}

// npcExecuted implements engine.ExecutionObserver: patches the last
// agent-decision record with the true execution outcome.
func (a *AgentSession) npcExecuted(action model.AgentAction, result model.ActionResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rec := a.lastCommitted
	a.lastCommitted = nil
	if rec == nil || rec.ExecutionStatus != agentStatusExecuted {
		return
	}
	if rec.ExecutedAction == nil || rec.ExecutedAction.ActionID != agentActionIDForEngineAction(action) {
		return
	}
	if !result.Executed {
		rec.ExecutionStatus = agentStatusFailed
		rec.ExecutionReason = "execution failed: " + strings.TrimSpace(result.Message)
	}
}

// step signals one month and waits for the first park.
func (a *AgentSession) step() (any, error) {
	a.mu.Lock()
	switch a.phase {
	case agentPhaseIdle:
		// ok
	case agentPhaseAwaitingChoice:
		id := ""
		if a.pending != nil {
			id = a.pending.ctx.DecisionID
		}
		a.mu.Unlock()
		return nil, fmt.Errorf("decision %s is pending; submit /debug/agent/choose first", id)
	case agentPhaseGameOver:
		a.mu.Unlock()
		return nil, fmt.Errorf("game over; export records and start a new game")
	case agentPhaseStopped:
		a.mu.Unlock()
		return nil, fmt.Errorf("session stopped; export records and start a new game")
	case agentPhaseError:
		err := a.errMsg
		a.mu.Unlock()
		return nil, fmt.Errorf("session error: %s", err)
	default:
		a.mu.Unlock()
		return nil, fmt.Errorf("session busy")
	}
	a.mu.Unlock()

	a.stepCh <- struct{}{}
	return a.waitPark()
}

// choose validates and delivers the decision, then waits for the next park.
func (a *AgentSession) choose(req AgentChooseRequest) (any, error) {
	a.mu.Lock()
	if a.phase != agentPhaseAwaitingChoice || a.pending == nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("no pending decision")
	}
	p := a.pending
	if req.DecisionID != p.ctx.DecisionID {
		a.mu.Unlock()
		return nil, fmt.Errorf("decision_id mismatch: pending %q", p.ctx.DecisionID)
	}
	delivery, err := validateChoiceLocked(p, req)
	if err != nil {
		a.mu.Unlock()
		return nil, err
	}
	p.delivery = delivery
	p.delivered = true
	a.phase = agentPhaseRunning
	close(p.doneCh)
	a.mu.Unlock()

	return a.waitPark()
}

// stop ends the session, aborting a pending decision if there is one.
func (a *AgentSession) stop() {
	a.mu.Lock()
	if a.phase == agentPhaseAwaitingChoice && a.pending != nil {
		p := a.pending
		p.record.ExecutionStatus = agentStatusAborted
		a.records = append(a.records, *p.record)
		a.pending = nil
		close(p.doneCh)
	}
	if a.phase != agentPhaseError && a.phase != agentPhaseGameOver {
		a.phase = agentPhaseStopped
	}
	a.mu.Unlock()

	close(a.quitCh)
	<-a.doneCh // wait for the runner to finish the month and exit
}

// waitPark converts the next park event into the HTTP response shape. The
// payload is normalized through JSON so in-process callers see exactly the
// same map[string]any structure the HTTP surface serializes.
func (a *AgentSession) waitPark() (any, error) {
	park := <-a.parkCh
	switch park.kind {
	case parkDecision:
		return map[string]any{"status": "decision", "decision": agentToJSONMap(park.ctx)}, nil
	case parkMonth:
		return agentToJSONMap(park.summary), nil
	case parkError:
		return nil, park.err
	}
	return nil, fmt.Errorf("internal: unknown park kind")
}

func agentToJSONMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	return m
}

// validateChoiceLocked checks the strict decision payload. Caller holds mu.
func validateChoiceLocked(p *pendingAgentDecision, req AgentChooseRequest) (agentDelivery, error) {
	n := len(p.infos)
	if req.Candidate < 0 || req.Candidate >= n {
		return agentDelivery{}, fmt.Errorf("candidate_id %d out of range [0,%d)", req.Candidate, n)
	}
	if len(req.Scores) != n {
		return agentDelivery{}, fmt.Errorf("scores cover %d of %d candidates", len(req.Scores), n)
	}
	ordered := make([]AgentScore, n)
	seen := make([]bool, n)
	for _, s := range req.Scores {
		if s.CandidateID < 0 || s.CandidateID >= n {
			return agentDelivery{}, fmt.Errorf("score candidate_id %d out of range [0,%d)", s.CandidateID, n)
		}
		if seen[s.CandidateID] {
			return agentDelivery{}, fmt.Errorf("duplicate score for candidate_id %d", s.CandidateID)
		}
		if s.Score < 0 || s.Score > 10 {
			return agentDelivery{}, fmt.Errorf("score %v out of range [0,10]", s.Score)
		}
		seen[s.CandidateID] = true
		ordered[s.CandidateID] = s
	}
	return agentDelivery{
		chosenIdx: req.Candidate,
		scores:    ordered,
		rationale: strings.TrimSpace(req.Rationale),
	}, nil
}

// applyDeliveryLocked fills the record with the agent's delivered choice.
// Caller holds mu.
func (a *AgentSession) applyDeliveryLocked(p *pendingAgentDecision) {
	rec := p.record
	for i := range rec.Candidates {
		score := p.delivery.scores[i].Score
		rec.Candidates[i].ModelScore = score
		rec.Candidates[i].Willingness = 1.0
		rec.Candidates[i].FinalMultiplier = agentScoreToMultiplier(score)
	}
	chosen := p.infos[p.delivery.chosenIdx]
	act := &agentAction{ActionID: agentActionID(chosen), Args: agentCandidateArgs(chosen)}
	rec.ExecutedAction = act
	rec.Response = &agentAction{ActionID: act.ActionID, Args: act.Args}
	rec.ExecutionStatus = agentStatusExecuted
	rec.ExecutionReason = p.delivery.rationale
}

// agentScoreToMultiplier mirrors llm.scoreToMultiplier: a 0-10 score maps
// onto the engine advice multiplier range [0.5, 2.0].
func agentScoreToMultiplier(score float64) float64 {
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	return engine.AdviceFactorMin + score/10*(engine.AdviceFactorMax-engine.AdviceFactorMin)
}

// status is the operator-facing session snapshot.
func (a *AgentSession) status() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := map[string]any{
		"active":             a.phase != agentPhaseNone,
		"phase":              a.phase.String(),
		"game_id":            a.gameID,
		"seed":               a.seed,
		"months_elapsed":     a.months,
		"decisions_recorded": len(a.records),
		"timeout_ms":         int(a.timeout / time.Millisecond),
	}
	if a.pending != nil {
		st["pending_decision_id"] = a.pending.ctx.DecisionID
	}
	if a.errMsg != "" {
		st["error"] = a.errMsg
	}
	return st
}

// export returns the records plus session metadata.
func (a *AgentSession) export() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	records := make([]agentRecord, len(a.records))
	copy(records, a.records)
	return map[string]any{
		"game_id":            a.gameID,
		"seed":               a.seed,
		"phase":              a.phase.String(),
		"months_elapsed":     a.months,
		"decisions_recorded": len(records),
		"records":            records,
	}
}

// exportJSONL renders one canonical JSON record per line.
func (a *AgentSession) exportJSONL() string {
	a.mu.Lock()
	records := make([]agentRecord, len(a.records))
	copy(records, a.records)
	a.mu.Unlock()

	var sb strings.Builder
	for i := range records {
		b, err := json.Marshal(records[i])
		if err != nil {
			continue
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// buildSummaryLocked renders the month-end summary. Caller holds mu.
func (a *AgentSession) buildSummaryLocked(result *model.TickResult) *AgentMonthSummary {
	sum := &AgentMonthSummary{
		Status:            "month_end",
		Year:              result.Silo.CurrentYear,
		Month:             result.Silo.CurrentMonth,
		GameOver:          result.GameOver,
		Population:        result.Silo.TotalPopulation,
		Rebellion:         agentRound2(result.Silo.Rebellion),
		Cohesion:          agentRound2(result.Silo.Cohesion),
		Legitimacy:        agentRound2(result.Silo.MayorLegitimacy()),
		Logs:              result.Logs,
		DecisionsRecorded: len(a.records),
		MonthsElapsed:     a.months,
	}
	if result.GameOver {
		sum.Status = "game_over"
		if result.Silo.VictoryStatus != nil {
			sum.Ending = result.Silo.VictoryStatus.Type
		}
	}
	return sum
}

// buildContextLocked renders what the agent sees at a decision point.
// Caller holds mu.
func (a *AgentSession) buildContextLocked(seq int, view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) *AgentDecisionContext {
	state := a.buildStatePayloadLocked(view, silo)
	cands := make([]AgentCandidate, len(candidates))
	wMin, wMax := 0.0, 0.0
	for i, c := range candidates {
		if i == 0 || c.BaseWeight < wMin {
			wMin = c.BaseWeight
		}
		if i == 0 || c.BaseWeight > wMax {
			wMax = c.BaseWeight
		}
	}
	for i, c := range candidates {
		cands[i] = AgentCandidate{
			CandidateID: i,
			ActionID:    agentActionID(c),
			Args:        agentCandidateArgs(c),
			Description: agentCandidateDescription(c),
			APCost:      c.APCost,
			BaseWeight:  agentRound2(c.BaseWeight),
			AnchorScore: agentAnchorScore(c.BaseWeight, wMin, wMax),
		}
	}
	return &AgentDecisionContext{
		DecisionID:  fmt.Sprintf("%s-%04d", a.gameID, seq),
		Year:        silo.CurrentYear,
		Month:       silo.CurrentMonth,
		Actor:       state["actor"].(map[string]any),
		Silo:        state["silo"].(map[string]any),
		Departments: state["departments"].([]map[string]any),
		Factions:    state["factions"].([]map[string]any),
		Candidates:  cands,
	}
}

// buildRecordScaffoldLocked renders the pipeline record skeleton (prompt +
// candidates with original weights). Scores/choice fields are filled when
// the agent delivers. Caller holds mu.
func (a *AgentSession) buildRecordScaffoldLocked(seq int, view *engine.ActorView, silo *model.Silo, candidates []engine.CandidateInfo) *agentRecord {
	state := a.buildStatePayloadLocked(view, silo)
	promptPayload := map[string]any{
		"actor":       state["actor"],
		"silo":        state["silo"],
		"departments": state["departments"],
		"factions":    state["factions"],
	}
	promptBytes, _ := json.Marshal(promptPayload)

	recs := make([]agentCandidateRecord, len(candidates))
	for i, c := range candidates {
		recs[i] = agentCandidateRecord{
			ActionID:       agentActionID(c),
			Args:           agentCandidateArgs(c),
			Executable:     true,
			OriginalWeight: agentRound2(c.BaseWeight),
		}
	}
	return &agentRecord{
		DecisionID:      fmt.Sprintf("%s-%04d", a.gameID, seq),
		GameID:          a.gameID,
		OccurredAt:      fmt.Sprintf("Y%d-M%02d", silo.CurrentYear, silo.CurrentMonth),
		NpcID:           agentNpcID(view),
		DecisionVersion: agentDecisionVersion,
		Prompt:          string(promptBytes),
		Candidates:      recs,
		ExecutionStatus: agentStatusExecuted, // provisional; finalized on commit/execute
	}
}

// buildStatePayloadLocked renders the fog-filtered state whitelist (same
// fields as the LLM advisor's buildRequest, plus silo identity so
// multi-silo games stay attributable). Caller holds mu.
func (a *AgentSession) buildStatePayloadLocked(view *engine.ActorView, silo *model.Silo) map[string]any {
	depts := make([]map[string]any, 0, len(silo.Professions))
	for i := range silo.Professions {
		p := &silo.Professions[i]
		depts = append(depts, map[string]any{
			"name": p.Name, "class": p.ClassType, "panic": agentRound2(p.PanicValue),
			"productivity": agentRound2(p.Productivity), "power": p.PowerLevel,
		})
	}
	factions := make([]map[string]any, 0)
	for i := range silo.Factions {
		f := &silo.Factions[i]
		// Fog filter: only public factions are visible to NPC actors.
		if !f.IsPublic {
			continue
		}
		factions = append(factions, map[string]any{
			"name": f.Name, "members": f.MemberCount, "influence": agentRound2(f.Influence),
		})
	}
	resources := map[string]float64{}
	for _, r := range silo.Resources {
		resources[r.Type] = r.Amount
	}
	return map[string]any{
		"actor": map[string]any{
			"name":              view.Label(),
			"profession":        view.Profession(),
			"action_points":     agentRound2(view.ActionPoints()),
			"suspicion":         agentRound2(view.SuspicionLevel()),
			"prestige":          agentRound2(view.PoliticalPrestige()),
			"propaganda":        agentRound2(view.PropagandaLevel()),
			"known_fragments":   view.KnownFragments(),
			"is_representative": view.IsRepresentative(),
		},
		"silo": map[string]any{
			"id": silo.ID, "name": silo.Name,
			"year": silo.CurrentYear, "month": silo.CurrentMonth,
			"population": silo.TotalPopulation,
			"legitimacy": agentRound2(silo.MayorLegitimacy()), "cohesion": agentRound2(silo.Cohesion),
			"rebellion": agentRound2(silo.Rebellion), "resources": resources,
		},
		"departments": depts,
		"factions":    factions,
	}
}

// agentActionID synthesizes the strict action_id for one candidate. The
// pipeline requires action_id to be unique per record, so profession
// actions use their (globally unique) definition ID instead of the shared
// PROFESSION_ACTION type string; the six base types appear at most once
// per candidate set and keep their type string. The mapping back to an
// engine AgentAction is documented in references/agent-operator.md.
func agentActionID(c engine.CandidateInfo) string {
	if c.Type == model.ActionProfession && c.ProfessionAction != "" {
		return c.ProfessionAction
	}
	return string(c.Type)
}

// agentCandidateArgs synthesizes the strict args dict for one candidate
// (targets only; the profession action identity lives in action_id).
func agentCandidateArgs(c engine.CandidateInfo) map[string]string {
	args := map[string]string{}
	if c.TargetDept != "" {
		args["target_dept"] = c.TargetDept
	}
	if c.ResourceTarget != "" {
		args["resource_target"] = c.ResourceTarget
	}
	if len(c.FragmentIds) > 0 {
		args["fragments"] = strings.Join(c.FragmentIds, ",")
	}
	return args
}

// agentActionIDForEngineAction maps an executed engine action back to the
// record action_id vocabulary (mirror of agentActionID).
func agentActionIDForEngineAction(action model.AgentAction) string {
	if action.Type == model.ActionProfession && action.ProfessionAction != "" {
		return action.ProfessionAction
	}
	return string(action.Type)
}

// agentCandidateDescription gives the agent a one-line human reading of the
// candidate (not part of the training label).
func agentCandidateDescription(c engine.CandidateInfo) string {
	switch c.Type {
	case model.ActionGatherInfo:
		return "secretly gather intel on " + c.TargetDept
	case model.ActionShareInfo:
		return "share known fragments with " + c.TargetDept
	case model.ActionBuildConnection:
		return "build a connection with " + c.TargetDept
	case model.ActionConductPropaganda:
		return "conduct propaganda to boost influence"
	case model.ActionInciteRebellion:
		return "incite unrest among the commoners"
	case model.ActionPublicizeFaction:
		return "publicize the faction to the whole silo"
	case model.ActionProfession:
		desc := "profession action " + c.ProfessionAction
		if c.TargetDept != "" {
			desc += " on " + c.TargetDept
		}
		if c.ResourceTarget != "" {
			desc += " on " + c.ResourceTarget
		}
		return desc
	}
	return string(c.Type)
}

func agentNpcID(view *engine.ActorView) string {
	switch view.Ref.Kind {
	case engine.ActorNPC:
		return fmt.Sprintf("NPC:%d", view.Ref.ProfessionID)
	case engine.ActorResident:
		return fmt.Sprintf("RESIDENT:%d", view.Ref.ResidentID)
	}
	return fmt.Sprintf("AGENT:%d", view.Ref.AgentID)
}

// agentAnchorScore mirrors llm.anchorScore: normalize a base weight into
// the 0-10 scale over [2,8].
func agentAnchorScore(w, wMin, wMax float64) float64 {
	if wMax <= wMin {
		return 5
	}
	return agentRound2(2 + 6*(w-wMin)/(wMax-wMin))
}

func agentRound2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

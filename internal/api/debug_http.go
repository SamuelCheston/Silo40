package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"silo40/internal/model"
)

const defaultDebugHTTPAddr = "127.0.0.1:18080"

// GameAPI mirrors the methods exposed through Wails so the same operations
// can be exercised over HTTP during debugging.
type GameAPI interface {
	CreateGame(req model.CreateGameRequest) (*model.GameState, error)
	GetGameState() (*model.GameState, error)
	GetEventHistory(limit int) (*model.EventHistoryResult, error)
	PassTime(months int) (*model.TickResult, error)
	ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error)
	GetEndingNarrative() (string, error)
	HasActiveGame() (bool, error)
	GetProfessionActions(profession string) ([]model.ProfessionActionMeta, error)
	GetProfessionAction(id string) (*model.ProfessionActionMeta, error)
}

type DebugHTTPServer struct {
	addr   string
	api    GameAPI
	llm    LlmAPI
	agent  AgentAPI
	server *http.Server
}

// LlmAPI is the optional LLM advisor surface (design §6.3/§11). It is kept
// separate from GameAPI so existing implementors are unaffected; when not
// set, the /debug/game/llm/* routes answer 503.
type LlmAPI interface {
	LlmStatus() (any, error)
	LlmDecisions(limit int) (any, error)
	SetLlmParticipation(level string) (string, error)
}

// AgentAPI is the agent-operator surface: an external AI agent with tool
// access plays the game itself (the tick parks at each NPC decision point
// until the agent submits a choice). Request bodies arrive as raw JSON so
// this package stays decoupled from the service-layer types; when not set,
// the /debug/agent/* routes answer 503.
type AgentAPI interface {
	AgentStart(body []byte) (any, error)
	AgentStep() (any, error)
	AgentChoose(body []byte) (any, error)
	AgentStatus() (any, error)
	AgentExport() (any, error)
	AgentExportJSONL() (string, error)
	AgentStop() (any, error)
}

// SetLlmAPI attaches the LLM surface before Start.
func (s *DebugHTTPServer) SetLlmAPI(l LlmAPI) { s.llm = l }

// SetAgentAPI attaches the agent-operator surface before Start.
func (s *DebugHTTPServer) SetAgentAPI(a AgentAPI) { s.agent = a }

func NewDebugHTTPServer(addr string, api GameAPI) *DebugHTTPServer {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = defaultDebugHTTPAddr
	}

	s := &DebugHTTPServer{
		addr: addr,
		api:  api,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/health", s.handleHealth)
	mux.HandleFunc("/debug/game/active", s.handleHasActiveGame)
	mux.HandleFunc("/debug/game/state", s.handleGetGameState)
	mux.HandleFunc("/debug/game/create", s.handleCreateGame)
	mux.HandleFunc("/debug/game/pass-time", s.handlePassTime)
	mux.HandleFunc("/debug/game/action", s.handleExecuteAction)
	mux.HandleFunc("/debug/game/ending", s.handleGetEndingNarrative)
	mux.HandleFunc("/debug/game/events", s.handleGetEventHistory)
	mux.HandleFunc("/debug/game/profession-actions", s.handleGetProfessionActions)
	mux.HandleFunc("/debug/game/profession-action", s.handleGetProfessionAction)
	mux.HandleFunc("/debug/game/llm/status", s.handleLlmStatus)
	mux.HandleFunc("/debug/game/llm/decisions", s.handleLlmDecisions)
	mux.HandleFunc("/debug/game/llm/participation", s.handleLlmParticipation)
	mux.HandleFunc("/debug/agent/start", s.handleAgentStart)
	mux.HandleFunc("/debug/agent/step", s.handleAgentStep)
	mux.HandleFunc("/debug/agent/choose", s.handleAgentChoose)
	mux.HandleFunc("/debug/agent/status", s.handleAgentStatus)
	mux.HandleFunc("/debug/agent/export", s.handleAgentExport)
	mux.HandleFunc("/debug/agent/stop", s.handleAgentStop)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

func (s *DebugHTTPServer) Addr() string {
	return s.addr
}

func (s *DebugHTTPServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		log.Printf("Debug HTTP server listening on http://%s", listener.Addr().String())
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Debug HTTP server stopped with error: %v", err)
		}
	}()

	return nil
}

func (s *DebugHTTPServer) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *DebugHTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"addr": s.addr,
	})
}

func (s *DebugHTTPServer) handleHasActiveGame(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	active, err := s.api.HasActiveGame()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"active": active})
}

func (s *DebugHTTPServer) handleGetGameState(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	state, err := s.api.GetGameState()
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *DebugHTTPServer) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var req model.CreateGameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state, err := s.api.CreateGame(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *DebugHTTPServer) handlePassTime(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Months int `json:"months"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Months <= 0 {
		req.Months = 1
	}
	result, err := s.api.PassTime(req.Months)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var action model.AgentAction
	if err := decodeJSON(r, &action); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.api.ExecuteAction(action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleGetEndingNarrative(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	narrative, err := s.api.GetEndingNarrative()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ending_narrative": narrative})
}

func (s *DebugHTTPServer) handleGetEventHistory(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	limit, err := intQuery(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.api.GetEventHistory(limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleGetProfessionActions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	profession := strings.TrimSpace(r.URL.Query().Get("profession"))
	if profession == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing query parameter: profession"))
		return
	}
	result, err := s.api.GetProfessionActions(profession)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleGetProfessionAction(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing query parameter: id"))
		return
	}
	result, err := s.api.GetProfessionAction(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if result == nil {
		writeError(w, http.StatusNotFound, errors.New("profession action not found"))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleLlmStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if s.llm == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("LLM advisor not attached"))
		return
	}
	status, err := s.llm.LlmStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *DebugHTTPServer) handleLlmDecisions(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if s.llm == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("LLM advisor not attached"))
		return
	}
	limit, err := intQuery(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entries, err := s.llm.LlmDecisions(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *DebugHTTPServer) handleLlmParticipation(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if s.llm == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("LLM advisor not attached"))
		return
	}
	var req struct {
		Level string `json:"level"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	level, err := s.llm.SetLlmParticipation(req.Level)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"level": level})
}

// agentBody reads a bounded raw JSON request body for the agent surface.
func agentBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *DebugHTTPServer) handleAgentStart(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	body, err := agentBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.agent.AgentStart(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleAgentStep(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	result, err := s.agent.AgentStep()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleAgentChoose(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	body, err := agentBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.agent.AgentChoose(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	result, err := s.agent.AgentStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleAgentExport(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("format")) == "jsonl" {
		text, err := s.agent.AgentExportJSONL()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, text); err != nil {
			log.Printf("failed to write agent export: %v", err)
		}
		return
	}
	result, err := s.agent.AgentExport()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *DebugHTTPServer) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent surface not attached"))
		return
	}
	result, err := s.agent.AgentStop()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.Header().Set("Allow", method+", OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return false
	}
	return true
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func intQuery(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("invalid integer query parameter: " + key)
	}
	return value, nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

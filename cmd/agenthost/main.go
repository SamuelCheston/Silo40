// Command agenthost runs the Silo40 game service headless (no GUI) with the
// debug HTTP surface, so an external AI agent can play games over HTTP and
// produce real NPC decision records for post-training.
//
// Usage (run from a checkout that contains the events/ tree — the host
// loads content from disk, including the action-gate registry, with the
// same fail-closed guarantee as the production embed path):
//
//	go run ./cmd/agenthost -addr 127.0.0.1:18080
//
// The agent then drives the game through the /debug/agent/* endpoints
// (see references/agent-operator.md). One game at a time; records are
// exported per game via GET /debug/agent/export?format=jsonl.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"silo40/internal/api"
	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"
	"silo40/internal/service"

	"gorm.io/gorm"
)

// serviceAPI adapts GameService to api.GameAPI (same shape as the Wails
// binding layer in app.go).
type serviceAPI struct{ svc *service.GameService }

func (a serviceAPI) CreateGame(req model.CreateGameRequest) (*model.GameState, error) {
	return a.svc.CreateGame(req)
}
func (a serviceAPI) GetGameState() (*model.GameState, error) { return a.svc.GetState() }
func (a serviceAPI) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	return a.svc.GetEventHistory(limit)
}
func (a serviceAPI) PassTime(months int) (*model.TickResult, error) { return a.svc.PassTime(months) }
func (a serviceAPI) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	return a.svc.ExecuteAction(action)
}
func (a serviceAPI) GetEndingNarrative() (string, error) { return a.svc.GetEndingNarrative(), nil }
func (a serviceAPI) HasActiveGame() (bool, error)        { return a.svc.HasActiveGame(), nil }
func (a serviceAPI) GetProfessionActions(profession string) ([]model.ProfessionActionMeta, error) {
	return engine.GetProfessionActionMeta(profession), nil
}
func (a serviceAPI) GetProfessionAction(id string) (*model.ProfessionActionMeta, error) {
	def := engine.GetProfessionAction(id)
	if def == nil {
		return nil, nil
	}
	return &model.ProfessionActionMeta{
		ID: def.ID, Profession: def.Profession, Label: def.Label,
		Description: def.Description, APCost: int(def.APCost), TargetType: string(def.TargetType),
		SuspicionPenalty: def.SuspicionPenalty,
	}, nil
}

// agentAPI adapts GameService to api.AgentAPI.
type agentAPI struct{ svc *service.GameService }

func (a agentAPI) AgentStart(body []byte) (any, error) {
	var req service.AgentStartRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, fmt.Errorf("invalid start request: %w", err)
		}
	}
	return a.svc.StartAgentGame(req)
}
func (a agentAPI) AgentStep() (any, error) { return a.svc.AgentStep() }
func (a agentAPI) AgentChoose(body []byte) (any, error) {
	var req service.AgentChooseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid choose request: %w", err)
	}
	return a.svc.AgentChoose(req)
}
func (a agentAPI) AgentStatus() (any, error)         { return a.svc.AgentStatus() }
func (a agentAPI) AgentExport() (any, error)         { return a.svc.AgentExport() }
func (a agentAPI) AgentExportJSONL() (string, error) { return a.svc.AgentExportJSONL() }
func (a agentAPI) AgentStop() (any, error)           { return a.svc.AgentStop() }

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "debug HTTP listen address")
	dbPath := flag.String("db", "silo40-agenthost.db", "SQLite database file")
	flag.Parse()

	db, err := repository.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}

	svc := service.NewGameService(db)
	if err := svc.BootstrapContentFromDisk(); err != nil {
		log.Fatalf("bootstrap content from disk: %v", err)
	}
	if err := svc.Resume(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("resume: %v (starting fresh is normal)", err)
		}
	}

	server := api.NewDebugHTTPServer(*addr, serviceAPI{svc: svc})
	server.SetAgentAPI(agentAPI{svc: svc})
	if err := server.Start(); err != nil {
		log.Fatalf("start debug http server: %v", err)
	}
	log.Printf("agenthost ready: http://%s (agent endpoints under /debug/agent/*)", *addr)

	// Wait for SIGINT/SIGTERM, then shut the HTTP server down cleanly.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("shutdown: %v", err)
	}
	log.Printf("agenthost stopped")
}

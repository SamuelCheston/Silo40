package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"silo40/internal/model"
)

type stubGameAPI struct{}

func (stubGameAPI) CreateGame(req model.CreateGameRequest) (*model.GameState, error) {
	return &model.GameState{
		Silo:  model.Silo{Name: req.SiloName},
		Agent: model.Agent{Name: req.AgentName, Profession: req.Profession},
	}, nil
}

func (stubGameAPI) GetGameState() (*model.GameState, error) {
	return &model.GameState{
		Silo:  model.Silo{Name: "Silo 40"},
		Agent: model.Agent{Name: "Juliette"},
	}, nil
}

func (stubGameAPI) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	return &model.EventHistoryResult{
		Events: []model.StoryEventLog{{Title: "Story", Month: 1, Year: 122}},
	}, nil
}

func (stubGameAPI) PassTime(months int) (*model.TickResult, error) {
	return &model.TickResult{
		Silo:  model.Silo{CurrentMonth: months},
		Agent: model.Agent{Name: "Juliette"},
	}, nil
}

func (stubGameAPI) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	return &model.ActionOutcome{
		Result: model.ActionResult{Executed: true, Message: string(action.Type)},
	}, nil
}

func (stubGameAPI) GetEndingNarrative() (string, error) {
	return "ending", nil
}

func (stubGameAPI) HasActiveGame() (bool, error) {
	return true, nil
}

func (stubGameAPI) GetProfessionActions(profession string) ([]model.ProfessionActionMeta, error) {
	return []model.ProfessionActionMeta{{ID: "inspect", Profession: profession, Label: "Inspect"}}, nil
}

func (stubGameAPI) GetProfessionAction(id string) (*model.ProfessionActionMeta, error) {
	return &model.ProfessionActionMeta{ID: id, Profession: "Mechanical", Label: "Inspect"}, nil
}

func TestDebugHTTPCreateGame(t *testing.T) {
	server := NewDebugHTTPServer("", stubGameAPI{})

	req := httptest.NewRequest(http.MethodPost, "/debug/game/create", strings.NewReader(`{
		"silo_name":"Silo 17",
		"agent_name":"Walker",
		"profession":"IT"
	}`))
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got model.GameState
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got.Silo.Name != "Silo 17" {
		t.Fatalf("expected silo name to round-trip, got %q", got.Silo.Name)
	}
	if got.Agent.Name != "Walker" {
		t.Fatalf("expected agent name to round-trip, got %q", got.Agent.Name)
	}
}

func TestDebugHTTPPassTimeDefaultsToOneMonth(t *testing.T) {
	server := NewDebugHTTPServer("", stubGameAPI{})

	req := httptest.NewRequest(http.MethodPost, "/debug/game/pass-time", strings.NewReader(`{"months":0}`))
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got model.TickResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got.Silo.CurrentMonth != 1 {
		t.Fatalf("expected default month to be 1, got %d", got.Silo.CurrentMonth)
	}
}

func TestDebugHTTPProfessionActionsRequiresProfession(t *testing.T) {
	server := NewDebugHTTPServer("", stubGameAPI{})

	req := httptest.NewRequest(http.MethodGet, "/debug/game/profession-actions", nil)
	rec := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

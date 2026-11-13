package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubLlmAPI struct {
	level string
}

func (s *stubLlmAPI) LlmStatus() (any, error) {
	return map[string]any{"enabled": true, "mode": "shadow", "level": s.level}, nil
}

func (s *stubLlmAPI) LlmDecisions(limit int) (any, error) {
	return []map[string]any{{"seq": 1, "kind": "decision"}}, nil
}

func (s *stubLlmAPI) SetLlmParticipation(level string) (string, error) {
	s.level = level
	return level, nil
}

func newLlmTestServer(t *testing.T, llm LlmAPI) *httptest.Server {
	t.Helper()
	srv := NewDebugHTTPServer("127.0.0.1:0", stubGameAPI{})
	srv.SetLlmAPI(llm)
	ts := httptest.NewServer(srv.server.Handler)
	t.Cleanup(ts.Close)
	return ts
}

func TestLlmRoutesRequireAttachedAdvisor(t *testing.T) {
	ts := newLlmTestServer(t, nil)
	for _, path := range []string{"/debug/game/llm/status", "/debug/game/llm/decisions"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("GET %s without advisor: want 503, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestLlmRoutesServeStatusDecisionsAndSwitch(t *testing.T) {
	stub := &stubLlmAPI{level: "off"}
	ts := newLlmTestServer(t, stub)

	resp, err := http.Get(ts.URL + "/debug/game/llm/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status route: %v / %v", err, resp)
	}
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/debug/game/llm/decisions?limit=10")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("decisions route: %v / %v", err, resp)
	}
	var body struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || len(body.Entries) != 1 {
		t.Fatalf("decisions body: %v / %+v", err, body)
	}
	resp.Body.Close()

	resp, err = http.Post(ts.URL+"/debug/game/llm/participation", "application/json",
		strings.NewReader(`{"level":"key"}`))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("participation route: %v / %v", err, resp)
	}
	var switched struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&switched); err != nil || switched.Level != "key" {
		t.Fatalf("participation body: %v / %+v", err, switched)
	}
	resp.Body.Close()
	if stub.level != "key" {
		t.Fatalf("switch not applied: %q", stub.level)
	}
}

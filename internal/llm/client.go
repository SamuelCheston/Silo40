package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"time"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// Usage reports token accounting for one completion (zero when the
// endpoint does not provide it).
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Client abstracts the model endpoint so the chain can be validated without
// a real API key (FakeClient) and any OpenAI-compatible endpoint can be
// swapped via base URL.
type Client interface {
	// Complete sends system+user messages and returns the raw text reply
	// plus token usage.
	Complete(ctx context.Context, system, user string) (string, Usage, error)
	// ModelID identifies the model for the decision log (replay audit).
	ModelID() string
}

// HTTPClient talks to an OpenAI-compatible /chat/completions endpoint.
type HTTPClient struct {
	BaseURL string // e.g. https://api.openai.com/v1 or http://127.0.0.1:11434/v1
	APIKey  string
	Model   string
	HTTP    *http.Client
	// ReasoningEffort, when non-empty, is sent as reasoning_effort. Thinking
	// models (qwen3.x) need "none" for this task: with thinking on they burn
	// thousands of tokens before emitting JSON. Strict providers reject the
	// field on non-reasoning models, so it stays opt-in.
	ReasoningEffort string
}

// NewHTTPClient fills defaults: OpenAI base URL. The client-level timeout is
// a generous backstop; the effective budget is the per-request context the
// advisor passes (SILO40_LLM_TIMEOUT_MS for mode=on, 15s for shadow).
func NewHTTPClient(baseURL, apiKey, modelName, reasoningEffort string) *HTTPClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &HTTPClient{
		BaseURL:         strings.TrimRight(baseURL, "/"),
		APIKey:          apiKey,
		Model:           modelName,
		HTTP:            &http.Client{Timeout: 30 * time.Second},
		ReasoningEffort: reasoningEffort,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	Temperature     float64       `json:"temperature"`
	MaxTokens       int           `json:"max_tokens"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *HTTPClient) Complete(ctx context.Context, system, user string) (string, Usage, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature:     0,
		MaxTokens:       400,
		ReasoningEffort: c.ReasoningEffort,
	})
	if err != nil {
		return "", Usage{}, err
	}
	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", Usage{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", Usage{}, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, truncate(string(data), 200))
	}
	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", Usage{}, fmt.Errorf("bad response from %s: %w", url, err)
	}
	if parsed.Error != nil {
		return "", Usage{}, fmt.Errorf("endpoint error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("endpoint returned no choices")
	}
	usage := Usage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
	}
	return parsed.Choices[0].Message.Content, usage, nil
}

func (c *HTTPClient) ModelID() string {
	return c.BaseURL + "#" + c.Model
}

// FakeClient is a deterministic, dependency-free Client used to validate the
// shadow/replay chain without a real key. Scores are a pure function of the
// request text, so identical contexts always produce identical advice.
type FakeClient struct {
	Name string
}

func (f *FakeClient) Complete(_ context.Context, _, user string) (string, Usage, error) {
	var ctx struct {
		Candidates []struct {
			CandidateID int    `json:"candidate_id"`
			Type        string `json:"type"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(user), &ctx); err != nil {
		return "", Usage{}, fmt.Errorf("fake client: bad context: %w", err)
	}
	type outScore struct {
		CandidateID int     `json:"candidate_id"`
		Score       float64 `json:"score"`
	}
	out := struct {
		Scores  []outScore `json:"scores"`
		Factors []struct {
			ActionType string  `json:"action_type"`
			Factor     float64 `json:"factor"`
		} `json:"factors,omitempty"`
	}{}
	for _, c := range ctx.Candidates {
		base := 5.0
		switch model.AgentActionType(c.Type) {
		case model.ActionGatherInfo:
			base = 6.0
		case model.ActionBuildConnection:
			base = 5.5
		case model.ActionProfession:
			base = 7.0
		case model.ActionShareInfo:
			base = 4.5
		case model.ActionConductPropaganda:
			base = 4.0
		case model.ActionInciteRebellion:
			base = 3.0
		case model.ActionPublicizeFaction:
			base = 8.0
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(fmt.Sprintf("%s:%d", c.Type, c.CandidateID)))
		jitter := float64(h.Sum32()%5) * 0.25 // deterministic 0..1.0
		out.Scores = append(out.Scores, outScore{CandidateID: c.CandidateID, Score: base + jitter})
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", Usage{}, err
	}
	return string(data), Usage{}, nil
}

func (f *FakeClient) ModelID() string {
	if f.Name == "" {
		return "fake#deterministic"
	}
	return "fake#" + f.Name
}

// ContextBuilder renders the per-NPC decision context (design §3.3/§3.4).
// Kept in the llm package so the fog-filter whitelist is testable in
// isolation; the engine supplies visibility levels.
type ContextBuilder struct{}

// CandidateContext is the wire shape of one candidate inside a request.
type CandidateContext struct {
	CandidateID int     `json:"candidate_id"`
	Type        string  `json:"type"`
	Target      string  `json:"target,omitempty"`
	APCost      float64 `json:"ap_cost"`
	BaseWeight  float64 `json:"base_weight"`
	AnchorScore float64 `json:"anchor_score"`
}

// BuildCandidateList maps engine candidate infos to the wire shape. Targets
// are included (they are the NPC's own intended target, not fog info).
// base_weight and anchor_score are included as the alignment anchor: the
// first gate run (no anchor, weights hidden) showed the model scoring by
// action-type labels instead of the situation (mode agreement 12.9%).
func (ContextBuilder) BuildCandidateList(infos []engine.CandidateInfo) []CandidateContext {
	out := make([]CandidateContext, 0, len(infos))
	wMin, wMax := 0.0, 0.0
	for i, c := range infos {
		if i == 0 || c.BaseWeight < wMin {
			wMin = c.BaseWeight
		}
		if i == 0 || c.BaseWeight > wMax {
			wMax = c.BaseWeight
		}
	}
	for _, c := range infos {
		cc := CandidateContext{
			CandidateID: c.Index,
			Type:        string(c.Type),
			APCost:      c.APCost,
			BaseWeight:  round2(c.BaseWeight),
			AnchorScore: anchorScore(c.BaseWeight, wMin, wMax),
		}
		switch {
		case c.Type == model.ActionProfession:
			cc.Target = c.ProfessionAction
		case c.TargetDept != "":
			cc.Target = c.TargetDept
		case c.ResourceTarget != "":
			cc.Target = c.ResourceTarget
		}
		out = append(out, cc)
	}
	return out
}

// anchorScore normalizes a base weight into the 0-10 score scale, spread
// over [2,8] so the model has room to deviate in either direction. Equal
// weights (or a single candidate) map to the neutral middle 5.
func anchorScore(w, wMin, wMax float64) float64 {
	if wMax <= wMin {
		return 5
	}
	return round2(2 + 6*(w-wMin)/(wMax-wMin))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

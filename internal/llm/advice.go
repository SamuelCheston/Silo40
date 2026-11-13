package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"silo40/internal/engine"
	"silo40/internal/model"
)

// factorKeyFor maps channel-B factor keys to engine action types. The keys
// are the design-doc willingness coefficients (§10.2); profession actions
// share one coefficient by design.
var factorKeyFor = map[string]model.AgentActionType{
	"gather_info":        model.ActionGatherInfo,
	"share_info":         model.ActionShareInfo,
	"build_connection":   model.ActionBuildConnection,
	"conduct_propaganda": model.ActionConductPropaganda,
	"incite_rebellion":   model.ActionInciteRebellion,
	"publicize_faction":  model.ActionPublicizeFaction,
	"profession_action":  model.ActionProfession,
}

// FactorKeyForType is the inverse of factorKeyFor (audit/logging).
func FactorKeyForType(t model.AgentActionType) string {
	for k, v := range factorKeyFor {
		if v == t {
			return k
		}
	}
	return string(t)
}

// RawResponse is the validated shape of a model reply (design §3.2/§10).
type RawResponse struct {
	Scores  []RawScore  `json:"scores"`
	Factors []RawFactor `json:"factors,omitempty"`
	// Repaired is true when deterministic bracket repair was needed to make
	// the reply parse (qwen3.x non-thinking mode empirically drops the final
	// closing brace). Repaired replies pass the same full validation.
	Repaired bool `json:"-"`
}

// RawScore is channel A: one candidate's 0-10 score.
type RawScore struct {
	CandidateID int     `json:"candidate_id"`
	Score       float64 `json:"score"`
}

// RawFactor is channel B: one action-type willingness multiplier.
type RawFactor struct {
	ActionType string  `json:"action_type"`
	Factor     float64 `json:"factor"`
}

// ParseResponse extracts and validates a RawResponse from model output.
// nCandidates is the exact candidate set that was sent; the response must
// cover it exactly. Prose-wrapped JSON (``` fences, leading text) is
// salvaged by taking the outermost braces; replies truncated of their final
// closing brackets (a qwen3.x non-thinking quirk) are deterministically
// repaired and then pass the same full validation.
func ParseResponse(raw string, nCandidates int) (*RawResponse, error) {
	text := strings.TrimSpace(raw)
	if a, b := strings.Index(text, "{"), strings.LastIndex(text, "}"); a >= 0 && b > a {
		text = text[a : b+1]
	}
	var resp RawResponse
	err := json.Unmarshal([]byte(text), &resp)
	if err != nil {
		repaired := repairTruncatedJSON(text)
		if rerr := json.Unmarshal([]byte(repaired), &resp); rerr != nil {
			return nil, fmt.Errorf("unparseable response: %w", err)
		}
		resp.Repaired = true
	}
	if len(resp.Scores) == 0 {
		return nil, fmt.Errorf("response has no scores")
	}
	seen := map[int]bool{}
	for _, s := range resp.Scores {
		if s.CandidateID < 0 || s.CandidateID >= nCandidates {
			return nil, fmt.Errorf("candidate_id %d out of range [0,%d)", s.CandidateID, nCandidates)
		}
		if seen[s.CandidateID] {
			return nil, fmt.Errorf("duplicate candidate_id %d", s.CandidateID)
		}
		seen[s.CandidateID] = true
		if s.Score < 0 || s.Score > 10 {
			return nil, fmt.Errorf("score %v out of range [0,10]", s.Score)
		}
	}
	if len(seen) != nCandidates {
		return nil, fmt.Errorf("scores cover %d of %d candidates", len(seen), nCandidates)
	}
	for _, f := range resp.Factors {
		if _, ok := factorKeyFor[f.ActionType]; !ok {
			return nil, fmt.Errorf("unknown factor action_type %q", f.ActionType)
		}
	}
	return &resp, nil
}

// ToAdvice converts a validated response into engine advice. Channel-A
// scores map 0-10 onto multipliers [0.5, 2.0]; channel-B factors are used
// as multipliers directly. The engine clamps again at application time.
func ToAdvice(resp *RawResponse) *engine.NpcAdvice {
	if resp == nil {
		return nil
	}
	adv := &engine.NpcAdvice{}
	if len(resp.Scores) > 0 {
		adv.Scores = make(map[int]float64, len(resp.Scores))
		for _, s := range resp.Scores {
			adv.Scores[s.CandidateID] = scoreToMultiplier(s.Score)
		}
	}
	if len(resp.Factors) > 0 {
		adv.Factors = make(map[model.AgentActionType]float64, len(resp.Factors))
		for _, f := range resp.Factors {
			if t, ok := factorKeyFor[f.ActionType]; ok {
				adv.Factors[t] = f.Factor
			}
		}
	}
	return adv
}

func scoreToMultiplier(score float64) float64 {
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	return engine.AdviceFactorMin + score/10*(engine.AdviceFactorMax-engine.AdviceFactorMin)
}

// repairTruncatedJSON appends the closing brackets/quote missing from a
// truncated JSON value, based on a string-aware bracket balance scan.
// Deterministic; the result still has to pass full schema validation.
func repairTruncatedJSON(text string) string {
	var stack []byte
	inStr := false
	esc := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			switch c {
			case '\\':
				esc = true
			case '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			stack = append(stack, c)
		case '}':
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
		case ']':
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if inStr {
		text += `"`
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			text += "}"
		} else {
			text += "]"
		}
	}
	return text
}

package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContextLevel is the player-adjustable context level: how much situation is
// packed into the prompt when a decision point is referred to the model. It
// is orthogonal to the participation Level — participation decides "when to
// ask", the context level decides "how much to include when asking".
// Levels are nested: minimal ⊂ standard ⊂ rich ⊂ world.
type ContextLevel string

const (
	ContextMinimal  ContextLevel = "minimal"  // 仅当前 NPC、候选动作、输出契约
	ContextStandard ContextLevel = "standard" // + 地堡大局指标、相关派系人物、近期事件摘要
	ContextRich     ContextLevel = "rich"     // + 本堡主线经过
	ContextWorld    ContextLevel = "world"    // + silo 设定、其他地堡关系、长期剧情线
)

// DefaultContextLevel is the release default: standard keeps the historical
// payload shape (silo metrics + departments + factions) and adds the recent
// event summary per the roadmap table.
const DefaultContextLevel = ContextStandard

// ParseContextLevel normalizes a context level string; empty means default.
func ParseContextLevel(s string) (ContextLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return DefaultContextLevel, nil
	case "minimal", "min":
		return ContextMinimal, nil
	case "standard", "std":
		return ContextStandard, nil
	case "rich":
		return ContextRich, nil
	case "world":
		return ContextWorld, nil
	}
	return DefaultContextLevel, fmt.Errorf("invalid context level %q (want minimal|standard|rich|world)", s)
}

// AtLeast reports whether l carries at least the content of other (nesting
// order minimal < standard < rich < world).
func (l ContextLevel) AtLeast(other ContextLevel) bool {
	return contextRank(l) >= contextRank(other)
}

func contextRank(l ContextLevel) int {
	switch l {
	case ContextStandard:
		return 1
	case ContextRich:
		return 2
	case ContextWorld:
		return 3
	}
	return 0
}

// DefaultEventSummaryLimit caps the recent event summary entries at the
// standard+ levels. TODO(balance): 5 is a first guess for the cost/awareness
// trade-off; retune after real token accounting.
const DefaultEventSummaryLimit = 5

// ContextOptions carries the independent switches that stay orthogonal to
// the level (roadmap: 档位之外保留独立开关).
type ContextOptions struct {
	// PersonalMemory includes the actor's traits, known fragments and
	// relations (个人记忆). Default on: fragments were always part of the
	// historical prompt.
	PersonalMemory bool `json:"memory"`
	// ProtocolClauses includes the Order clause list (条文法规). Default
	// off: the historical prompt never carried clauses.
	ProtocolClauses bool `json:"protocol"`
	// EventSummaryLimit caps the recent event summary entries (事件摘要条数).
	// 0 disables the section entirely. Default DefaultEventSummaryLimit.
	EventSummaryLimit int `json:"event_limit"`
	// PeerSiloIntel includes rumor-grade intel about other silos (其他地堡
	// 情报). Default off: the prompt whitelist (§3.4) stays fog-free unless
	// the user explicitly opts in.
	PeerSiloIntel bool `json:"peer"`
}

// DefaultContextOptions is the release default: memory on (historical
// shape), everything else off / roadmap default.
func DefaultContextOptions() ContextOptions {
	return ContextOptions{
		PersonalMemory:    true,
		ProtocolClauses:   false,
		EventSummaryLimit: DefaultEventSummaryLimit,
		PeerSiloIntel:     false,
	}
}

// contextConfigRecord is the JSON payload of a kind=context log entry.
type contextConfigRecord struct {
	Level      string `json:"level"`
	Memory     bool   `json:"memory"`
	Protocol   bool   `json:"protocol"`
	EventLimit int    `json:"event_limit"`
	Peer       bool   `json:"peer"`
}

func marshalContextConfig(level ContextLevel, opts ContextOptions) (string, error) {
	b, err := json.Marshal(contextConfigRecord{
		Level:      string(level),
		Memory:     opts.PersonalMemory,
		Protocol:   opts.ProtocolClauses,
		EventLimit: opts.EventSummaryLimit,
		Peer:       opts.PeerSiloIntel,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseContextConfig restores a context config from its log payload. An
// unparsable payload is rejected so replay never applies a half-read config.
func parseContextConfig(s string) (ContextLevel, ContextOptions, error) {
	var rec contextConfigRecord
	if err := json.Unmarshal([]byte(s), &rec); err != nil {
		return DefaultContextLevel, DefaultContextOptions(), fmt.Errorf("llm: bad context config %q: %w", s, err)
	}
	level, err := ParseContextLevel(rec.Level)
	if err != nil {
		return DefaultContextLevel, DefaultContextOptions(), err
	}
	opts := ContextOptions{
		PersonalMemory:    rec.Memory,
		ProtocolClauses:   rec.Protocol,
		EventSummaryLimit: rec.EventLimit,
		PeerSiloIntel:     rec.Peer,
	}
	return level, opts, nil
}

// EventSummary is one entry of the event-summary prompt sections.
type EventSummary struct {
	Year        int    `json:"year"`
	Month       int    `json:"month"`
	Category    string `json:"category,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ClauseSummary is one Order clause rendered for the prompt (条文法规).
type ClauseSummary struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// PeerSiloBrief describes one other silo at the world level (其他地堡关系).
type PeerSiloBrief struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Relation string `json:"relation"`          // player | neighbor | silo1 | peer
	Status   string `json:"status,omitempty"` // normal / unrest / silent / destroyed
}

// ContextData carries the optional rich-context sections supplied by the
// service layer. Every field is optional; empty sections are omitted from
// the prompt so lower levels truly cost fewer tokens.
type ContextData struct {
	RecentEvents    []EventSummary  // 近期事件摘要 (standard+, capped by EventSummaryLimit)
	MainlineEvents  []EventSummary  // 本堡主线经过 (rich+)
	ProtocolClauses []ClauseSummary // 条文法规 (switch)
	WorldSetting    string          // silo 设定 (world)
	PeerSilos       []PeerSiloBrief // 其他地堡关系 (world)
	LongTermLines   []string        // 长期剧情线 (world)
	PeerIntel       []string        // 其他地堡情报 (switch, rumor-grade)
}

// ContextProvider supplies ContextData for one silo at prompt build time.
// Called on the game thread only (Advise/Committed/queuePrefetch all run
// under the service lock); implementations must not block, must not consume
// the engine rng, and must not mutate game state.
type ContextProvider func(siloID uint) ContextData

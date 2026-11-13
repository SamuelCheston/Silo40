package llm

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Log entry kinds.
const (
	KindDecision = "decision"
	KindConfig   = "config"
)

// Decision sources recorded per entry.
const (
	SourceOriginal    = "original"     // original algorithm decided (level gate off)
	SourceLLM         = "llm"          // LLM advice applied (synchronous call)
	SourceLLMPrefetch = "llm_prefetch" // LLM advice applied from the prefetch cache
	SourceShadow      = "shadow"       // original applied; LLM choice recorded alongside
	SourceReplay      = "replay"       // choice restored from this log during replay
)

// DecisionLog is the append-only record that makes LLM-assisted games
// replayable (design §2.2/§10.3/§11). Both decision points and participation
// config switches live in this table, ordered by Seq. It is intentionally
// NOT part of the persist() full-graph replacement: entries survive across
// save checkpoints and must never be rewritten.
type DecisionLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Seq       int64     `gorm:"uniqueIndex" json:"seq"` // per-game monotonic sequence
	Kind      string    `gorm:"size:16;index" json:"kind"`
	Year      int       `json:"year"`
	Month     int       `json:"month"`
	ActorKind string    `gorm:"size:16" json:"actor_kind,omitempty"` // PROFESSION | RESIDENT
	ActorID   uint      `json:"actor_id,omitempty"`
	ActorName string    `gorm:"size:80" json:"actor_name,omitempty"`
	// Decision payload (Kind=decision).
	CandidatesJSON string `json:"candidates_json,omitempty"`
	FactorsJSON    string `json:"factors_json,omitempty"` // channel B as returned
	ScoresJSON     string `json:"scores_json,omitempty"`  // channel A as returned
	FinalChoice    int    `json:"final_choice"`
	Source         string `gorm:"size:16" json:"source,omitempty"`
	ShadowChoice   *int   `json:"shadow_choice,omitempty"` // LLM's pick in shadow mode
	// Config payload (Kind=config).
	ConfigLevel  string `gorm:"size:16" json:"config_level,omitempty"`
	ConfigSource string `gorm:"size:16" json:"config_source,omitempty"` // env | create | api
	// Audit.
	Level      string    `gorm:"size:16" json:"level,omitempty"` // participation level in force
	Model      string    `gorm:"size:160" json:"model,omitempty"`
	PromptHash string    `gorm:"size:32" json:"prompt_hash,omitempty"`
	Validation string    `gorm:"size:32" json:"validation,omitempty"` // ok | <failure reason>
	LatencyMs  int       `json:"latency_ms,omitempty"`
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName pins the table name.
func (DecisionLog) TableName() string { return "llm_decision_logs" }

// Migrate creates the log table if needed.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("llm: nil db")
	}
	return db.AutoMigrate(&DecisionLog{})
}

// AppendEntry writes one entry with the next sequence number. Callers must
// hold their own ordering guarantee (the advisor serializes via its mutex).
func AppendEntry(db *gorm.DB, entry *DecisionLog) error {
	if db == nil {
		return fmt.Errorf("llm: nil db")
	}
	var last DecisionLog
	if err := db.Order("seq DESC").First(&last).Error; err == nil {
		entry.Seq = last.Seq + 1
	} else {
		entry.Seq = 1
	}
	return db.Create(entry).Error
}

// LoadEntries returns the full ordered log for audit/replay.
func LoadEntries(db *gorm.DB) ([]DecisionLog, error) {
	if db == nil {
		return nil, fmt.Errorf("llm: nil db")
	}
	var entries []DecisionLog
	if err := db.Order("seq ASC").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// UpdateShadowFields patches the async LLM result into a shadow entry that
// was written at commit time (seq already assigned).
func UpdateShadowFields(db *gorm.DB, id uint, scoresJSON, factorsJSON, model, validation string, shadowChoice *int, latencyMs, promptTokens, completionTokens int) error {
	if db == nil {
		return fmt.Errorf("llm: nil db")
	}
	return db.Model(&DecisionLog{}).Where("id = ?", id).Updates(map[string]any{
		"scores_json":       scoresJSON,
		"factors_json":      factorsJSON,
		"model":             model,
		"validation":        validation,
		"shadow_choice":     shadowChoice,
		"latency_ms":        latencyMs,
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
	}).Error
}

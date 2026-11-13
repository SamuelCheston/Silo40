package llm

import (
	"os"
	"strconv"
	"strings"
)

// Environment variable names (author-visible configuration, documented in
// references/llm-npc.md). The project has no config-file system; env vars
// follow the SILO40_SEED / SILO40_DEBUG_HTTP_ADDR precedent.
const (
	EnvMode          = "SILO40_LLM_MODE"          // off | shadow | on
	EnvParticipation = "SILO40_LLM_PARTICIPATION" // off | events | key | full
	EnvBaseURL       = "SILO40_LLM_BASE_URL"      // OpenAI-compatible base URL
	EnvAPIKey        = "SILO40_LLM_API_KEY"       // never persisted, never logged
	EnvAPIKeyFile    = "SILO40_LLM_KEY_FILE"      // optional key file (0600)
	EnvModel         = "SILO40_LLM_MODEL"         // model name for the log
	EnvEventWindow   = "SILO40_LLM_EVENT_WINDOW"  // months after ONCE events
	EnvTimeoutMS     = "SILO40_LLM_TIMEOUT_MS"    // mode=on per-call budget
	// EnvReasoningEffort: sent as reasoning_effort when non-empty. Thinking
	// models (qwen3.x) must use "none" for this task or they burn thousands
	// of tokens on reasoning before emitting JSON.
	EnvReasoningEffort = "SILO40_LLM_REASONING_EFFORT"
	// EnvWarmup defaults on; EnvPrefetch defaults off to avoid background
	// model spend when predicted work cannot finish before the next tick.
	EnvWarmup   = "SILO40_LLM_WARMUP"
	EnvPrefetch = "SILO40_LLM_PREFETCH"
)

// EnvConfig is the parsed environment configuration.
type EnvConfig struct {
	Mode            Mode
	Level           Level
	BaseURL         string
	APIKey          string
	Model           string
	EventWindow     int
	TimeoutMS       int
	ReasoningEffort string
	Warmup          bool
	Prefetch        bool
	ModeParseErr    error
	LevelParseErr   error
}

// envFlagDefault parses an explicit boolean switch and applies defaultValue
// only when the variable is empty. False values always win over defaults.
func envFlagDefault(v string, defaultValue bool) bool {
	if strings.TrimSpace(v) == "" {
		return defaultValue
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

// LoadEnv reads the SILO40_LLM_* variables. Parse errors are returned in
// the struct so the service layer can log them explicitly and fall back to
// off (never a silent half-enabled state).
func LoadEnv() EnvConfig {
	cfg := EnvConfig{EventWindow: DefaultEventWindow}

	cfg.Mode, cfg.ModeParseErr = ParseMode(os.Getenv(EnvMode))
	if cfg.ModeParseErr != nil {
		cfg.Mode = ModeOff
	}
	if p := os.Getenv(EnvParticipation); p != "" {
		cfg.Level, cfg.LevelParseErr = ParseLevel(p)
		if cfg.LevelParseErr != nil {
			cfg.Level = LevelOff
		}
	} else {
		// Release default (author-approved 2026-11-13): when LLM is enabled
		// and no level is set, intervene only at key nodes. off/shadow modes
		// stay contract-safe; explicit SILO40_LLM_PARTICIPATION=off restores
		// the old "enabled but never intervenes" behavior.
		cfg.Level = LevelKey
	}
	cfg.BaseURL = os.Getenv(EnvBaseURL)
	cfg.Model = os.Getenv(EnvModel)
	cfg.ReasoningEffort = os.Getenv(EnvReasoningEffort)
	cfg.Warmup = envFlagDefault(os.Getenv(EnvWarmup), true)
	cfg.Prefetch = envFlagDefault(os.Getenv(EnvPrefetch), false)
	cfg.APIKey = os.Getenv(EnvAPIKey)
	if cfg.APIKey == "" {
		if file := os.Getenv(EnvAPIKeyFile); file != "" {
			if data, err := os.ReadFile(file); err == nil {
				cfg.APIKey = string(data)
			}
		}
	}
	if w := os.Getenv(EnvEventWindow); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n >= 0 {
			cfg.EventWindow = n
		}
	}
	if w := os.Getenv(EnvTimeoutMS); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			cfg.TimeoutMS = n
		}
	}
	return cfg
}

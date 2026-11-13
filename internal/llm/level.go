// Package llm implements the optional LLM advisor layer for NPC decisions:
// participation levels, advice schema validation, OpenAI-compatible clients
// (HTTP + deterministic fake), and the append-only decision/config log that
// makes LLM-assisted games replayable under the determinism contract.
//
// The package never touches the engine rng; all advice flows through
// engine.NpcPolicy injected by the service layer.
package llm

import (
	"fmt"
	"strings"
)

// Mode controls whether LLM advice is applied to the game.
type Mode string

const (
	ModeOff    Mode = "off"    // LLM disabled (default, contract-safe)
	ModeShadow Mode = "shadow" // LLM decisions recorded but never applied
	ModeOn     Mode = "on"     // LLM advice applied per participation level
)

// ParseMode normalizes a mode string; empty means off.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return ModeOff, nil
	case "shadow":
		return ModeShadow, nil
	case "on":
		return ModeOn, nil
	}
	return ModeOff, fmt.Errorf("invalid LLM mode %q (want off|shadow|on)", s)
}

// Level is the player-adjustable participation level: which decision points
// the LLM may intervene on. Levels are nested: off ⊂ events ⊂ key ⊂ full.
type Level string

const (
	LevelOff   Level = "off"   // never intervene
	LevelEvent Level = "events" // only around ONCE special/crisis/history events
	LevelKey   Level = "key"   // events ∪ key-node metric/structure clauses
	LevelFull  Level = "full"  // every decision point
)

// ParseLevel normalizes a participation level string; empty means off.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return LevelOff, nil
	case "events", "event":
		return LevelEvent, nil
	case "key":
		return LevelKey, nil
	case "full":
		return LevelFull, nil
	}
	return LevelOff, fmt.Errorf("invalid participation level %q (want off|events|key|full)", s)
}

// AtLeast reports whether l covers at least the coverage of other (nesting
// order off < events < key < full).
func (l Level) AtLeast(other Level) bool {
	return rank(l) >= rank(other)
}

func rank(l Level) int {
	switch l {
	case LevelEvent:
		return 1
	case LevelKey:
		return 2
	case LevelFull:
		return 3
	}
	return 0
}

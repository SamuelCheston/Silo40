#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -n "${SILO40_BINARY:-}" ]]; then
  BINARY="$SILO40_BINARY"
elif [[ -x "$ROOT/build/bin/silo40" ]]; then
  BINARY="$ROOT/build/bin/silo40"
else
  BINARY="$ROOT/silo40"
fi

if [[ ! -x "$BINARY" ]]; then
  printf 'Silo 40 executable not found or not executable: %s\n' "$BINARY" >&2
  exit 1
fi

EVENTS_DIR="$(dirname "$BINARY")/events"
if [[ -d "$EVENTS_DIR" && ! -f "$EVENTS_DIR/action_gates.json" ]]; then
  printf 'Silo 40 package is incomplete: missing %s/action_gates.json\n' "$EVENTS_DIR" >&2
  exit 1
fi

DATA_DIR="${SILO40_DATA_DIR:-$ROOT/.playable-data}"
mkdir -p "$DATA_DIR"
export SILO40_LLM_MODE=off
export SILO40_DEBUG_HTTP_ADDR="${SILO40_DEBUG_HTTP_ADDR:-127.0.0.1:18092}"

cd "$DATA_DIR"
exec "$BINARY"

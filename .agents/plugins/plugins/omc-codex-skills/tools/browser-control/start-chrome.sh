#!/usr/bin/env bash
set -euo pipefail

PORT="${BROWSER_CONTROL_PORT:-9222}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
PROFILE="${BROWSER_CONTROL_PROFILE:-${WORKSPACE_ROOT}/.codex-tools/chrome-profile}"
URL="${1:-http://127.0.0.1:8081}"

if lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Chrome CDP already listening on 127.0.0.1:${PORT}; not opening a new window or tab." >&2
  exit 0
fi

mkdir -p "${PROFILE}"

open -na "Google Chrome" --args \
  --remote-debugging-port="${PORT}" \
  --user-data-dir="${PROFILE}" \
  --no-first-run \
  --no-default-browser-check \
  "${URL}"

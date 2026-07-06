#!/usr/bin/env bash
set -euo pipefail

PORT="${BROWSER_CONTROL_PORT:-9222}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
PROFILE="${BROWSER_CONTROL_PROFILE:-${WORKSPACE_ROOT}/.codex-tools/chrome-profile}"

pids="$(lsof -tiTCP:"${PORT}" -sTCP:LISTEN || true)"
if [[ -z "${pids}" ]]; then
  echo "No Chrome CDP listener on 127.0.0.1:${PORT}."
  exit 0
fi

for pid in ${pids}; do
  command_line="$(ps -p "${pid}" -o command= || true)"
  if [[ "${command_line}" == *"Google Chrome"* && "${command_line}" == *"--remote-debugging-port=${PORT}"* && "${command_line}" == *"--user-data-dir=${PROFILE}"* ]]; then
    kill "${pid}"
    echo "Stopped managed Chrome PID ${pid}."
  else
    echo "Refusing to stop PID ${pid}; it does not match the managed Chrome profile/port." >&2
    exit 1
  fi
done

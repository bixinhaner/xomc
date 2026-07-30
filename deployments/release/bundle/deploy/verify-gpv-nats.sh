#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
NATS_SERVER_BIN="${NATS_SERVER_BIN:-$(command -v nats-server || true)}"
[ -n "$NATS_SERVER_BIN" ] || {
  echo "nats-server is required for GPV JetStream verification" >&2
  exit 1
}

PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
STORE="$(mktemp -d)"
LOG="$STORE/nats.log"
NATS_PID=""

cleanup() {
  if [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
    kill "$NATS_PID" 2>/dev/null || true
    wait "$NATS_PID" 2>/dev/null || true
  fi
  rm -rf "$STORE"
}
trap cleanup EXIT

"$NATS_SERVER_BIN" -js -a 127.0.0.1 -p "$PORT" -sd "$STORE/data" >"$LOG" 2>&1 &
NATS_PID=$!
for _ in $(seq 1 100); do
  grep -Fq 'Server is ready' "$LOG" && break
  kill -0 "$NATS_PID" 2>/dev/null || {
    cat "$LOG" >&2
    exit 1
  }
  sleep 0.05
done
grep -Fq 'Server is ready' "$LOG" || {
  cat "$LOG" >&2
  exit 1
}

cd "$REPO_ROOT/omcgo"
GPV_NATS_TEST_URL="nats://127.0.0.1:$PORT" \
  go test ./internal/core/event ./cmd/gpv-handoff \
    -run 'TestKeyedQueue|TestKeyedPull|TestPrepareGPVHandoff|TestRunFreshInstall' \
    -count=1 -v

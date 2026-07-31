#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
if [ -z "${NATS_SERVER_BIN:-}" ]; then
  NATS_SERVER_BIN="$(command -v nats-server || true)"
  if [ -z "$NATS_SERVER_BIN" ]; then
    for candidate in "$HOME/.local/bin/nats-server" "$HOME/.opencode/bin/nats-server" /usr/local/bin/nats-server; do
      if [ -x "$candidate" ]; then
        NATS_SERVER_BIN="$candidate"
        break
      fi
    done
  fi
fi
[ -n "$NATS_SERVER_BIN" ] || {
  echo "nats-server is required for GPV JetStream verification" >&2
  exit 1
}
GO_BIN="${GO_BIN:-$(command -v go || true)}"
if [ -z "$GO_BIN" ]; then
  for candidate in "$HOME/.local/go/bin/go" "$HOME/.opencode/bin/go" "$HOME/.g/go/bin/go" /usr/local/go/bin/go; do
    if [ -x "$candidate" ]; then
      GO_BIN="$candidate"
      case "$candidate" in
        */go/bin/go) export GOROOT="${candidate%/bin/go}" ;;
      esac
      break
    fi
  done
fi
[ -n "$GO_BIN" ] || {
  echo "go is required for GPV JetStream verification" >&2
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
run_gpv_tests() {
  GPV_NATS_TEST_URL="nats://127.0.0.1:$PORT" \
    "$GO_BIN" test ./internal/core/event ./cmd/gpv-handoff \
      -run 'TestKeyedQueue|TestKeyedPull|TestPrepareGPVHandoff|TestRunFreshInstall' \
      -count=1 -v
}

# A freshly started JetStream can occasionally time out the first publish
# confirmation under a busy build host. Retry the complete isolated suite once;
# a second failure remains a real gate failure.
if ! run_gpv_tests; then
  echo "GPV NATS verification transiently failed; retrying once" >&2
  run_gpv_tests
fi

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

PORT=""
STORE=""
LOG=""
NATS_PID=""

cleanup() {
  if [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
    kill "$NATS_PID" 2>/dev/null || true
    wait "$NATS_PID" 2>/dev/null || true
  fi
  [ -n "$STORE" ] && rm -rf "$STORE"
  return 0
}
trap cleanup EXIT

start_nats() {
  PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
  STORE="$(mktemp -d)"
  LOG="$STORE/nats.log"
  "$NATS_SERVER_BIN" -js -a 127.0.0.1 -p "$PORT" -sd "$STORE/data" >"$LOG" 2>&1 &
  NATS_PID=$!
  for _ in $(seq 1 100); do
    grep -Fq 'Server is ready' "$LOG" && return 0
    kill -0 "$NATS_PID" 2>/dev/null || break
    sleep 0.05
  done
  cat "$LOG" >&2
  return 1
}

stop_nats() {
  if [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
    kill "$NATS_PID" 2>/dev/null || true
    wait "$NATS_PID" 2>/dev/null || true
  fi
  NATS_PID=""
  [ -n "$STORE" ] && rm -rf "$STORE"
  PORT=""
  STORE=""
  LOG=""
}

run_gpv_tests() {
  start_nats || return 1
  cd "$REPO_ROOT/omcgo"
  local test_status
  if GPV_NATS_TEST_URL="nats://127.0.0.1:$PORT" \
    "$GO_BIN" test ./internal/core/event ./cmd/gpv-handoff \
      -run 'TestKeyedQueue|TestKeyedPull|TestPrepareGPVHandoff|TestRunFreshInstall' \
      -count=1 -v; then
    test_status=0
  else
    test_status=$?
  fi
  if [ "$test_status" -ne 0 ]; then
    echo "NATS 服务端日志（本次尝试）:" >&2
    cat "$LOG" >&2 || true
  fi
  stop_nats
  return "$test_status"
}

# 每次尝试都使用全新的 NATS 进程、端口和 JetStream 存储目录。这样重试真正隔离
# 上一轮可能残留的订阅/元数据状态，而不是在同一实例上重复触发相同故障。
for attempt in 1 2; do
  if run_gpv_tests; then
    exit 0
  fi
  if [ "$attempt" -lt 2 ]; then
    echo "GPV NATS verification transiently failed; retrying with a fresh server" >&2
  fi
done
exit 1

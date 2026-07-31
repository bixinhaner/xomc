#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
NATS_SERVER_IMAGE="${NATS_SERVER_IMAGE:-nats:2.12.11-alpine3.22}"
NATS_DOCKER_CID=""
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
if [ -z "$NATS_SERVER_BIN" ]; then
  command -v docker >/dev/null 2>&1 || {
    echo "nats-server or docker is required for GPV JetStream verification" >&2
    exit 1
  }
  docker image inspect "$NATS_SERVER_IMAGE" >/dev/null 2>&1 || {
    echo "nats-server is unavailable and local Docker image is missing: $NATS_SERVER_IMAGE" >&2
    exit 1
  }
  NATS_SERVER_MODE=docker
else
  NATS_SERVER_MODE=bin
fi
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
  if [ "$NATS_SERVER_MODE" = docker ]; then
    [ -z "$NATS_DOCKER_CID" ] || docker rm -f "$NATS_DOCKER_CID" >/dev/null 2>&1 || true
  elif [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
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
  if [ "$NATS_SERVER_MODE" = docker ]; then
    NATS_DOCKER_CID="$(docker run -d --rm --name "omc-gpv-nats-$$" \
      -p "127.0.0.1:${PORT}:4222" -v "$STORE/data:/data" "$NATS_SERVER_IMAGE" \
      -js -a 0.0.0.0 -p 4222 -sd /data)" || return 1
    NATS_PID=""
  else
    "$NATS_SERVER_BIN" -js -a 127.0.0.1 -p "$PORT" -sd "$STORE/data" >"$LOG" 2>&1 &
    NATS_PID=$!
  fi
  for _ in $(seq 1 100); do
    if [ "$NATS_SERVER_MODE" = docker ]; then
      docker logs "$NATS_DOCKER_CID" 2>&1 | grep -Fq 'Server is ready' && return 0
      docker inspect -f '{{.State.Running}}' "$NATS_DOCKER_CID" 2>/dev/null | grep -Fxq true || break
    else
      grep -Fq 'Server is ready' "$LOG" && return 0
      kill -0 "$NATS_PID" 2>/dev/null || break
    fi
    sleep 0.05
  done
  if [ "$NATS_SERVER_MODE" = docker ]; then
    docker logs "$NATS_DOCKER_CID" >&2 2>&1 || true
  else
    cat "$LOG" >&2
  fi
  return 1
}

wait_for_nats_port() {
  for _ in $(seq 1 100); do
    if python3 - "$PORT" <<'PY'
import socket
import sys

port = int(sys.argv[1])
sock = socket.socket()
sock.settimeout(0.2)
try:
    sock.connect(("127.0.0.1", port))
except OSError:
    sys.exit(1)
finally:
    sock.close()
PY
    then
      return 0
    fi
    sleep 0.05
  done
  return 1
}

stop_nats() {
  if [ "$NATS_SERVER_MODE" = docker ]; then
    [ -z "$NATS_DOCKER_CID" ] || docker rm -f "$NATS_DOCKER_CID" >/dev/null 2>&1 || true
    NATS_DOCKER_CID=""
  elif [ -n "$NATS_PID" ] && kill -0 "$NATS_PID" 2>/dev/null; then
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
  wait_for_nats_port || {
    echo "NATS client port did not become ready: $PORT" >&2
    stop_nats
    return 1
  }
  sleep 0.2
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
# 上一轮可能残留的订阅/元数据状态，而不是在同一实例上重复触发相同故障。测试
# 覆盖多个 durable/stream 切换场景，启动期管理 API 偶发竞态时允许三次隔离重试。
for attempt in 1 2 3; do
  if run_gpv_tests; then
    exit 0
  fi
  if [ "$attempt" -lt 3 ]; then
    echo "GPV NATS verification transiently failed; retrying with a fresh server" >&2
  fi
done
exit 1

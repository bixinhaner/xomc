#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
NATS_SERVER_IMAGE="${NATS_SERVER_IMAGE:-nats:2.12.11-alpine3.22}"
NATS_DOCKER_CID=""

resolve_nats_docker_network() {
  if [ -n "${NATS_DOCKER_NETWORK:-}" ]; then
    printf '%s\n' "$NATS_DOCKER_NETWORK"
    return 0
  fi

  local host_os docker_os
  host_os="$(uname -s 2>/dev/null || printf 'unknown')"
  docker_os="$(docker info --format '{{.OperatingSystem}}' 2>/dev/null || true)"
  if [ "$host_os" = "Darwin" ]; then
    printf 'bridge\n'
    return 0
  fi
  case "$docker_os" in
    *Docker\ Desktop*) printf 'bridge\n' ;;
    *) printf 'host\n' ;;
  esac
}

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
  NATS_DOCKER_NETWORK="$(resolve_nats_docker_network)"
  case "$NATS_DOCKER_NETWORK" in
    host|bridge) ;;
    *)
      echo "NATS_DOCKER_NETWORK must be host or bridge, got: $NATS_DOCKER_NETWORK" >&2
      exit 1
      ;;
  esac
  echo "GPV NATS Docker network: $NATS_DOCKER_NETWORK" >&2
else
  NATS_SERVER_MODE=bin
fi
GO_BIN="${GO_BIN:-$(command -v go || true)}"
if [ -z "$GO_BIN" ]; then
  for candidate in \
    "$HOME/.local/go/bin/go" "$HOME/.opencode/bin/go" "$HOME/.opencode/go/bin/go" "$HOME/.g/go/bin/go" \
    /root/.local/go/bin/go /root/.opencode/bin/go /root/.g/go/bin/go \
    /usr/local/go/bin/go /opt/go/bin/go; do
    if [ -x "$candidate" ]; then
      GO_BIN="$candidate"
      break
    fi
  done
fi

if [ -n "$GO_BIN" ]; then
  case "$GO_BIN" in
    */go/bin/go) export GOROOT="${GO_BIN%/bin/go}" ;;
  esac
fi

[ -n "$GO_BIN" ] || {
  echo "go is required for GPV JetStream verification (set GO_BIN=/path/to/go to override)" >&2
  exit 1
}
"$GO_BIN" version >/dev/null 2>&1 || {
  echo "Go executable is not usable: $GO_BIN (check GOROOT or set GO_BIN)" >&2
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
    local nats_bind_address=127.0.0.1
    local docker_run_args=(--network "$NATS_DOCKER_NETWORK" -d --rm --name "omc-gpv-nats-$$")
    if [ "$NATS_DOCKER_NETWORK" = bridge ]; then
      nats_bind_address=0.0.0.0
      docker_run_args+=( -p "127.0.0.1:$PORT:$PORT" )
    fi
    NATS_DOCKER_CID="$(docker run "${docker_run_args[@]}" \
      -v "$STORE/data:/data" "$NATS_SERVER_IMAGE" \
      -js -a "$nats_bind_address" -p "$PORT" -sd /data)" || return 1
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
  # cmd/gpv-handoff 的 fresh-install 用例会先删除 JetStream 中的全部 stream。
  # Go 默认并行执行命令行中的多个 package；合并成一个 go test 命令会让它删掉
  # internal/core/event 正在验证的临时 stream，造成随机 stream-not-found 假失败。
  if GPV_NATS_TEST_URL="nats://127.0.0.1:$PORT" \
      "$GO_BIN" test ./internal/core/event \
        -run 'TestKeyedQueue|TestKeyedPull|TestPrepareGPVHandoff' -count=1 -v &&
    GPV_NATS_TEST_URL="nats://127.0.0.1:$PORT" \
      "$GO_BIN" test ./cmd/gpv-handoff -run 'TestRunFreshInstall' -count=1 -v; then
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

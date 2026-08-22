#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.test.yml"
PROJECT="${OMC_TEST_PROJECT:-omcgo-test}"

. "$SCRIPT_DIR/docker-network-lib.sh"
docker_network_resolve_bip "${DOCKER_NETWORK_ENV_FILE:-$REPO_ROOT/.env}" || {
  echo "[dc-test] 无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）" >&2
  exit 1
}
docker_network_plan || {
  echo "[dc-test] DOCKER_BIP 无效或无法派生 Docker 网段：${DOCKER_BIP:-<空>}" >&2
  exit 1
}

exec docker compose -p "$PROJECT" -f "$COMPOSE_FILE" "$@"
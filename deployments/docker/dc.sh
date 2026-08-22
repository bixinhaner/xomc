#!/usr/bin/env bash
# =============================================================================
# dc.sh —— OMC 本地 dev 栈 docker compose 包装器（自动带 resources.env）
# =============================================================================
# 解决「每次换机器都要手改资源限额」：本 wrapper 始终把 plan-resources.sh 生成的
# deployments/docker/resources.env 通过 --env-file 喂给 compose，且无论你在哪个目录
# 执行都自动定位到仓库根，省去 -f 长路径与 --env-file 的记忆负担。
#
# 行为：
#   · 若 resources.env 不存在 → 自动先跑一次 plan-resources.sh 按本机硬件生成。
#   · 透传所有参数给 docker compose（up/down/ps/logs/restart/build…）。
#   · COMPOSE_PROJECT 默认 omc（与现网容器名 omc-* 一致）；可用 OMC_PROJECT 覆盖。
#
# 用法：
#   bash deployments/docker/dc.sh up -d --build app worker
#   bash deployments/docker/dc.sh ps
#   bash deployments/docker/dc.sh logs -f --tail=100 acs
#   bash deployments/docker/dc.sh down
#   OMC_SKIP_AUTOPLAN=1 bash deployments/docker/dc.sh up -d   # 跳过自动规划（用 compose 默认值）
#   bash deployments/docker/dc.sh --replan up -d              # 强制重新探测硬件再起
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
RES_ENV="$SCRIPT_DIR/resources.env"
PROJECT="${OMC_PROJECT:-omc}"

if [ -f "$SCRIPT_DIR/docker-network-lib.sh" ]; then
  . "$SCRIPT_DIR/docker-network-lib.sh"
else
  echo "[dc] 缺少 Docker 网段规划库：$SCRIPT_DIR/docker-network-lib.sh" >&2
  exit 1
fi
docker_network_resolve_bip "${DOCKER_NETWORK_ENV_FILE:-$REPO_ROOT/.env}" || {
  echo "[dc] 无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）" >&2
  exit 1
}
docker_network_plan || {
  echo "[dc] DOCKER_BIP 无效或无法派生 Docker 网段：${DOCKER_BIP:-<空>}" >&2
  exit 1
}

REPLAN=0
if [ "${1:-}" = "--replan" ]; then REPLAN=1; shift; fi

# 生成 / 重生成 resources.env（除非显式跳过）
if [ "${OMC_SKIP_AUTOPLAN:-0}" != 1 ]; then
  if [ "$REPLAN" = 1 ] || [ ! -f "$RES_ENV" ]; then
    echo "[dc] 按本机硬件生成资源规划 → $RES_ENV"
    bash "$SCRIPT_DIR/plan-resources.sh" -o "$RES_ENV"
    echo ""
  fi
fi

# compose 一旦显式传任一 --env-file 就停止自动加载根 .env，故两个都须显式传。
# 顺序：先 .env（OMCGO_ENV 等基础变量），后 resources.env —— 后者覆盖前者，保证资源旋钮以规划值为准。
ENV_ARGS=()
[ -f "$REPO_ROOT/.env" ] && ENV_ARGS+=( --env-file "$REPO_ROOT/.env" )
[ -f "$RES_ENV" ] && [ "${OMC_SKIP_AUTOPLAN:-0}" != 1 ] && ENV_ARGS+=( --env-file "$RES_ENV" )

# 注意：macOS 默认 bash 3.2 下，set -u 时展开空数组 "${arr[@]}" 会报 unbound variable，
# 故用 "${arr[@]+"${arr[@]}"}" 惯用法 —— 空数组时安全展开为「无」。
exec docker compose -p "$PROJECT" "${ENV_ARGS[@]+"${ENV_ARGS[@]}"}" -f "$COMPOSE_FILE" "$@"

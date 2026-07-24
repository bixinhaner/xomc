#!/usr/bin/env bash
# =============================================================================
# OMC 启动健康校验 — 部署完成后执行（全 docker compose 部署版）
#
# 检查内容：
#   · docker compose 容器状态（business + infra + monitoring）
#   · 5 个核心健康端点（端口与 docker-compose port mapping 对齐）：
#     app /healthz（:9091）/ acs /healthz（:9095，容器 9090→宿主 9095）/
#     worker /healthz（:9092）/ app /metrics（:9091）/ 前端 SPA（:8081，
#     web 容器 nginx；:8080 是 ACS CWMP 反代不检）
#
# 用法：
#   bash healthcheck.sh                # 默认完整检查
#   bash healthcheck.sh -h | --help    # 本帮助
#
# 参数：
#   -h, --help    本帮助
#
# 退出码：0 全部通过 / 1 存在失败项
# =============================================================================
case "${1:-}" in
  -h|--help) sed -n '3,17p' "$0"; exit 0 ;;
esac

set -u

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-omcgo}"
SKIP_MONITORING=0
if [ -f "$DEPLOY_DIR/monitoring-profile-lib.sh" ]; then
  . "$DEPLOY_DIR/monitoring-profile-lib.sh"
else
  echo "  [FAIL] 缺 $DEPLOY_DIR/monitoring-profile-lib.sh"
  exit 1
fi
monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING" || {
  echo "  [FAIL] 无法读取 monitoring profile"
  exit 1
}

# docker compose 命令
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "  [FAIL] docker compose 未安装"
  exit 1
fi

# 组装 -f 参数（按文件存在情况）
COMPOSE_FILES=()
for f in docker-compose.infra.yml docker-compose.app.yml docker-compose.web.yml; do
  [ -f "$DEPLOY_DIR/$f" ] && COMPOSE_FILES+=( -f "$DEPLOY_DIR/$f" )
done
if [ "$SKIP_MONITORING" = 0 ] && [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ]; then
  COMPOSE_FILES+=( -f "$DEPLOY_DIR/docker-compose.monitoring.yml" )
fi
DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${COMPOSE_FILES[@]}" )

ok=0; fail=0
check() {  # check <描述> <命令...>
  local desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  [OK]   $desc"; ok=$((ok+1))
  else
    echo "  [FAIL] $desc"; fail=$((fail+1))
  fi
}

# container_running <service> —— 通过 docker compose ps 拿容器 ID 并检查 State=running
container_running() {
  local svc="$1"
  local cid
  cid="$("${DC[@]}" ps -q "$svc" 2>/dev/null | head -n1)"
  [ -n "$cid" ] || return 1
  [ "$(docker inspect -f '{{.State.Running}}' "$cid" 2>/dev/null)" = "true" ]
}

echo "== docker compose 业务容器 =="
for svc in app acs worker; do
  check "$svc 容器 running" container_running "$svc"
done

echo "== docker compose 基础设施容器 =="
for svc in postgres postgres-tsdb redis nats minio; do
  check "$svc 容器 running" container_running "$svc"
done

if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
  echo "== docker compose web 容器 =="
  check "web 容器 running" container_running web
fi

if [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ] && [ "$SKIP_MONITORING" = 0 ]; then
  echo "== docker compose 监控容器 =="
  for svc in prometheus alertmanager grafana loki tempo otelcol; do
    check "$svc 容器 running" container_running "$svc"
  done
  # otelcol-contrib 是 distroless 镜像，不能假设容器内有 shell/curl/wget。
  # monitoring compose 将 health_check extension 仅映射到宿主回环供外部探测。
  check "otelcol health extension (:13133)" curl -fsS http://127.0.0.1:13133/
fi

echo "== 服务健康端点 =="
# /healthz + /metrics 都在 metrics 端口上注册（internal/core/components/monitor/metrics.go）。
# 业务进程主 HTTP（app:8081 / acs SOAP:7547）不直接暴露 /healthz —— 用 metrics 端口检健康。
# 端口与 compose port mapping 对齐：app/worker 容器 == 宿主；acs 容器 9090 → 宿主 9095。
check "app    /healthz (:9091)"  curl -fsS http://127.0.0.1:9091/healthz
check "acs    /healthz (:9095)"  curl -fsS http://127.0.0.1:9095/healthz
check "worker /healthz (:9092)"  curl -fsS http://127.0.0.1:9092/healthz
check "app    /metrics (:9091)"  curl -fsS http://127.0.0.1:9091/metrics
# 前端 SPA：web 容器 nginx :8081 served（:8080 是 ACS CWMP 反代，GET / 不响应，不检）。
check "前端 SPA (:8081)"          curl -fsS http://127.0.0.1:8081/ -o /dev/null

echo
echo "compose ps 详情："
"${DC[@]}" ps 2>/dev/null || echo "  (无法读取 compose 状态)"

echo
echo "结果：通过 $ok 项，失败 $fail 项"
[ "$fail" -eq 0 ] || { echo "存在失败项，参见部署方案故障排查章节。"; exit 1; }
echo "校验通过。"

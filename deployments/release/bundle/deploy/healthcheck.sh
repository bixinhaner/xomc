#!/usr/bin/env bash
# =============================================================================
# OMC 启动健康校验 — 部署完成后执行（全 docker compose 部署版）
#
# 检查内容：
#   · docker compose 容器状态（business + infra + monitoring）
#   · 4 个核心健康端点：app /health（:8081）/ acs /healthz（:9090）/
#     app /metrics（:9091）/ 前端（:8080）
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
for f in docker-compose.infra.yml docker-compose.app.yml docker-compose.web.yml docker-compose.monitoring.yml; do
  [ -f "$DEPLOY_DIR/$f" ] && COMPOSE_FILES+=( -f "$DEPLOY_DIR/$f" )
done
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
for svc in postgres redis nats minio; do
  check "$svc 容器 running" container_running "$svc"
done

if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
  echo "== docker compose web 容器 =="
  check "web 容器 running" container_running web
fi

if [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ]; then
  echo "== docker compose 监控容器 =="
  for svc in prometheus alertmanager grafana loki tempo otelcol; do
    check "$svc 容器 running" container_running "$svc"
  done
fi

echo "== 服务健康端点 =="
check "app  /health  (:8081)"   curl -fsS http://127.0.0.1:8081/health
check "acs  /healthz (:9090)"   curl -fsS http://127.0.0.1:9090/healthz
check "app  metrics  (:9091)"   curl -fsS http://127.0.0.1:9091/metrics
check "前端 (:8080)"             curl -fsS http://127.0.0.1:8080/

echo
echo "compose ps 详情："
"${DC[@]}" ps 2>/dev/null || echo "  (无法读取 compose 状态)"

echo
echo "结果：通过 $ok 项，失败 $fail 项"
[ "$fail" -eq 0 ] || { echo "存在失败项，参见部署方案故障排查章节。"; exit 1; }
echo "校验通过。"

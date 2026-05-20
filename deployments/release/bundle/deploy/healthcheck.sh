#!/usr/bin/env bash
# =============================================================================
# OMC 启动健康校验 — 部署完成后执行
#
# 检查内容：
#   · omcgo-app / omcgo-acs / omcgo-worker 三个 systemd 服务是否 active
#   · 基础设施 docker compose 容器状态（postgres / redis / nats / minio）
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
  -h|--help) sed -n '3,19p' "$0"; exit 0 ;;
esac

set -u

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
ok=0; fail=0
check() {  # check <描述> <命令...>
  local desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  [OK]   $desc"; ok=$((ok+1))
  else
    echo "  [FAIL] $desc"; fail=$((fail+1))
  fi
}

echo "== OMC 三进程（systemd）=="
for svc in omcgo-app omcgo-acs omcgo-worker; do
  check "$svc active" systemctl is-active --quiet "$svc"
done

echo "== 基础设施容器 =="
docker compose -f "$DEPLOY_DIR/docker-compose.infra.yml" ps 2>/dev/null || \
  echo "  (无法读取 compose 状态，请手动检查)"

echo "== 服务健康端点 =="
check "app  /health  (:8081)"   curl -fsS http://127.0.0.1:8081/health
check "acs  /healthz (:9090)"   curl -fsS http://127.0.0.1:9090/healthz
check "app  metrics  (:9091)"   curl -fsS http://127.0.0.1:9091/metrics
check "前端 (:8080)"             curl -fsS http://127.0.0.1:8080/

echo
echo "结果：通过 $ok 项，失败 $fail 项"
[ "$fail" -eq 0 ] || { echo "存在失败项，参见部署方案 §10 故障排查。"; exit 1; }
echo "校验通过。"

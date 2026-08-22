#!/usr/bin/env bash
# =============================================================================
# OMC 服务控制脚本 — 启动 / 停止 / 重启 / 状态 / 日志
#
# 用法：
#   bash svc.sh status                  # 查看所有服务状态（默认）
#   bash svc.sh start [svc...]          # 启动全栈，或指定服务
#   bash svc.sh stop [svc...]           # 停止全栈，或指定服务
#   bash svc.sh restart [svc...]        # 重启全栈，或指定服务
#   bash svc.sh logs <svc> [--tail N]   # 查看指定服务日志（默认 tail 50）
#   bash svc.sh logs <svc> -f           # 跟随日志（Ctrl-C 退出）
#   bash svc.sh ps                      # 等价 status
#   bash svc.sh down                    # 关栈并清容器（保留 volume）
#   bash svc.sh up                      # 同 start，up -d 全栈
#   bash svc.sh -h                      # 帮助
#
# 选项（任意子命令前后皆可）：
#   --skip-web                          # 不操作 web compose
#   --skip-monitoring                   # 不操作 monitoring compose
#
# 工作目录：脚本须从「含 docker-compose.*.yml 的 deploy/ 目录」运行（与 install.sh
# 同目录）；自动按存在性把 4 个 compose file 拼入命令。compose 项目名固定 omcgo
# （与 install.sh 一致）。
#
# 不需要 root 权限（除非 docker daemon 需要 sudo）。
# =============================================================================

set -uo pipefail

log()  { echo -e "\033[1;32m[svc]\033[0m $*"; }
warn() { echo -e "\033[1;33m[svc][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[svc][错误]\033[0m $*" >&2; exit "${2:-1}"; }

usage() {
  sed -n '2,28p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

COMPOSE_PROJECT="omcgo"
SKIP_WEB=0
SKIP_MONITORING=0
ACTION=""
TARGETS=()
EXTRA_ARGS=()

# 解析参数：支持选项 + 子命令 + 自由位置参数任意顺序
while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help)         usage 0 ;;
    --skip-web)        SKIP_WEB=1; shift ;;
    --skip-monitoring) SKIP_MONITORING=1; shift ;;
    status|ps|start|stop|restart|logs|down|up|pull)
      if [ -z "$ACTION" ]; then
        ACTION="$1"
      else
        # 子命令已定，剩下进 TARGETS（或 logs 的 -f / --tail 等）
        TARGETS+=("$1")
      fi
      shift ;;
    -f|--follow|--tail|-n)
      EXTRA_ARGS+=("$1"); shift ;;
    *)
      if [ -z "$ACTION" ]; then
        die "未知子命令：$1（-h 查看用法）"
      fi
      TARGETS+=("$1"); shift ;;
  esac
done

[ -z "$ACTION" ] && ACTION="status"

# docker compose v2 / v1 兼容
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  die "未检测到 docker compose v2 / docker-compose v1"
fi

# compose 文件按存在性拼接（infra/app 必有；web/monitoring 看 skip flag 与存在性）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || die "无法进入脚本目录：$SCRIPT_DIR"
if [ -f "$SCRIPT_DIR/docker-network-lib.sh" ]; then
  . "$SCRIPT_DIR/docker-network-lib.sh"
elif [ -f "$SCRIPT_DIR/../../../docker/docker-network-lib.sh" ]; then
  . "$SCRIPT_DIR/../../../docker/docker-network-lib.sh"
else
  die "缺 docker-network-lib.sh；无法按 DOCKER_BIP 规划 Docker 网段"
fi
docker_network_resolve_bip "$SCRIPT_DIR/.env" ||
  die "无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）"
docker_network_plan || die "DOCKER_BIP 无效或无法派生 Docker 网段：${DOCKER_BIP:-<空>}"
if [ -f "$SCRIPT_DIR/storage-paths-lib.sh" ]; then
  . "$SCRIPT_DIR/storage-paths-lib.sh"
else
  die "缺 $SCRIPT_DIR/storage-paths-lib.sh"
fi
if [ -f "$SCRIPT_DIR/resource-env-lib.sh" ]; then
  . "$SCRIPT_DIR/resource-env-lib.sh"
else
  die "缺 $SCRIPT_DIR/resource-env-lib.sh（完整资源规划契约库）"
fi
if [ -f "$SCRIPT_DIR/resource-plan-metrics.sh" ]; then
  . "$SCRIPT_DIR/resource-plan-metrics.sh"
else
  die "缺 $SCRIPT_DIR/resource-plan-metrics.sh（资源计划 Prometheus 指标生成器）"
fi
if [ -f "$SCRIPT_DIR/monitoring-profile-lib.sh" ]; then
  . "$SCRIPT_DIR/monitoring-profile-lib.sh"
else
  die "缺 $SCRIPT_DIR/monitoring-profile-lib.sh"
fi
if [ -f "$SCRIPT_DIR/gpv-handoff-lib.sh" ]; then
  . "$SCRIPT_DIR/gpv-handoff-lib.sh"
else
  die "缺 $SCRIPT_DIR/gpv-handoff-lib.sh"
fi

# 显式 flag 优先；无 flag 时读取 install.sh 持久化在 .env 的部署模式。
monitoring_profile_apply_runtime ".env" "$SKIP_MONITORING" ||
  die "无法读取 monitoring profile"

COMPOSE_FILES=()
[ -f docker-compose.infra.yml ] && COMPOSE_FILES+=( -f docker-compose.infra.yml )
[ -f docker-compose.app.yml ]   && COMPOSE_FILES+=( -f docker-compose.app.yml )
if [ "$SKIP_WEB" = 0 ] && [ -f docker-compose.web.yml ]; then
  COMPOSE_FILES+=( -f docker-compose.web.yml )
fi
if [ "$SKIP_MONITORING" = 0 ] && [ -f docker-compose.monitoring.yml ]; then
  COMPOSE_FILES+=( -f docker-compose.monitoring.yml )
fi

[ ${#COMPOSE_FILES[@]} -eq 0 ] && die "当前目录未发现 docker-compose.*.yml（应在 deploy/ 目录运行）"

# 资源限额：compose 经 --env-file 读取 resources.env(plan-resources.sh 生成)。改完
# resources.env 后 svc.sh restart 即按新限额重建。一旦显式传任一 --env-file，compose
# 不再自动加载 ./.env，故 .env 也必须显式传；resources.env 缺失则拒绝执行。
ENV_FILES=()
[ -f .env ]          && ENV_FILES+=( --env-file .env )
[ -f resources.env ] ||
  die "缺少 resources.env；请先运行 plan-resources.sh，禁止静默回退 Compose 默认限额"
resource_env_validate resources.env ||
  die "resources.env 不是完整资源规划；请重新运行 plan-resources.sh，禁止缺失项静默回退 Compose 默认值"
ENV_FILES+=( --env-file resources.env )

refresh_resource_plan_metrics() {
  resource_plan_metrics_write resources.env ||
    die "无法生成 resources.env 对应的 Prometheus 资源计划指标"
}

DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${ENV_FILES[@]}" "${COMPOSE_FILES[@]}" )

app_exists() {
  local cid
  cid="$("${DC[@]}" ps -a -q app 2>/dev/null || true)"
  [ -n "$cid" ]
}

if gpv_handoff_action_touches_app "$ACTION" "${TARGETS[@]}" && app_exists; then
  log "在启动、停止或重建现有 app 前预创建 GPV RPC 固定 durable ..."
  gpv_handoff_prepare ||
    die "GPV consumer handoff 失败；未改变现有 app，未执行 $ACTION"
fi

# 行为分派
case "$ACTION" in
  status|ps)
    log "compose ps（项目 $COMPOSE_PROJECT）"
    "${DC[@]}" ps "${TARGETS[@]}"
    ;;

  start|up)
    storage_prepare_configured_env_paths ".env" ||
      die "有状态服务数据路径校验/创建失败；请检查 .env 中五个 *_DATA_PATH"
    refresh_resource_plan_metrics
    if [ ${#TARGETS[@]} -gt 0 ]; then
      log "启动服务：${TARGETS[*]}"
      "${DC[@]}" up -d "${TARGETS[@]}"
    else
      log "启动全栈（infra + app + web + monitoring，按 skip flag 过滤）"
      "${DC[@]}" up -d
    fi
    ;;

  stop)
    if [ ${#TARGETS[@]} -gt 0 ]; then
      log "停止服务：${TARGETS[*]}"
      "${DC[@]}" stop "${TARGETS[@]}"
    else
      log "停止全栈（容器保留，volume 保留）"
      "${DC[@]}" stop
    fi
    ;;

  restart)
    storage_prepare_configured_env_paths ".env" ||
      die "有状态服务数据路径校验/创建失败；请检查 .env 中五个 *_DATA_PATH"
    refresh_resource_plan_metrics
    # 一次性迁移 job（run-once，跑完即 Exited）。对已退出容器执行 docker compose
    # restart 语义不对，且会脱离 depends_on 健康门控在错误时机被强行拉起。
    ONESHOT_RE='^(migrate-schema|migrate-seed-sql|migrate-seed)$'
    if [ ${#TARGETS[@]} -gt 0 ]; then
      # 指定服务：过滤掉一次性 job
      FILTERED=()
      for s in "${TARGETS[@]}"; do
        if printf '%s' "$s" | grep -qE "$ONESHOT_RE"; then
          warn "跳过一次性迁移 job：$s（如需重跑用 bash svc.sh start $s）"
        else
          FILTERED+=("$s")
        fi
      done
      [ ${#FILTERED[@]} -eq 0 ] && die "重启目标全是一次性 job，已跳过；重跑迁移请用：bash svc.sh start <job>"
      log "重启服务：${FILTERED[*]}"
      "${DC[@]}" restart "${FILTERED[@]}"
    else
      # 整栈：不用 docker compose restart（它并发重启所有容器、忽略 depends_on、
      # 对 one-shot job 语义错误——2026-06-10 整栈 restart 即因此撞 OCI fork EOF +
      # 容器 IP 重排 + nginx 上游失效）。改用 up -d --force-recreate：按 depends_on +
      # 健康门控有序重建（基础设施 healthy → migrate 幂等重跑 → app/acs/worker → web），
      # 天然错峰、避免 fork 风暴；nginx 经 resolver 自动重新解析后端 IP。
      log "重启全栈（按 depends_on 有序重建，经健康门控；迁移 job 幂等重跑）"
      "${DC[@]}" up -d --force-recreate
    fi
    ;;

  logs)
    [ ${#TARGETS[@]} -eq 0 ] && die "logs 需指定服务名（如：bash svc.sh logs app）"
    # 默认 tail 50 — 但若 EXTRA_ARGS 已含 --tail 则不重复
    if ! printf '%s\n' "${EXTRA_ARGS[@]}" | grep -q -- '--tail\|-n'; then
      EXTRA_ARGS+=( --tail 50 )
    fi
    "${DC[@]}" logs "${EXTRA_ARGS[@]}" "${TARGETS[@]}"
    ;;

  pull)
    log "拉取最新镜像"
    "${DC[@]}" pull
    ;;

  down)
    warn "compose down 将移除容器（volume 保留）"
    "${DC[@]}" down
    ;;

  *)
    die "未实现的子命令：$ACTION（-h 查看用法）"
    ;;
esac

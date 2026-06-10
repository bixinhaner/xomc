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

DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${COMPOSE_FILES[@]}" )

# 行为分派
case "$ACTION" in
  status|ps)
    log "compose ps（项目 $COMPOSE_PROJECT）"
    "${DC[@]}" ps "${TARGETS[@]}"
    ;;

  start|up)
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
    if [ ${#TARGETS[@]} -gt 0 ]; then
      log "重启服务：${TARGETS[*]}"
      "${DC[@]}" restart "${TARGETS[@]}"
    else
      log "重启全栈"
      "${DC[@]}" restart
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

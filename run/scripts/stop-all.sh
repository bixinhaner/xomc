#!/bin/bash
# OMC 一键停止所有服务

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_ok()   { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }

# 通过进程名查找PID
find_running_pid() {
    pgrep -x "$1" 2>/dev/null | head -1
}

# 停止进程：先查PID文件，再查进程名，都找不到才认为未运行
stop_process() {
    local name="$1"
    local pid_file="$2"
    local proc_name="$3"  # 用于 fallback 检测的进程名（可选）

    local pid=""

    # 1. 尝试从 PID 文件获取
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if ! kill -0 "$pid" 2>/dev/null; then
            pid=""  # PID 文件里的进程已不存在
        fi
        rm -f "$pid_file"
    fi

    # 2. PID 文件没找到，通过进程名 fallback
    if [ -z "$pid" ] && [ -n "$proc_name" ]; then
        pid=$(find_running_pid "$proc_name")
    fi

    # 3. 执行停止
    if [ -n "$pid" ]; then
        kill "$pid" 2>/dev/null
        # 等待最多 3 秒优雅退出
        for i in 1 2 3; do
            if ! kill -0 "$pid" 2>/dev/null; then
                break
            fi
            sleep 1
        done
        # 还没退出则强制终止
        if kill -0 "$pid" 2>/dev/null; then
            kill -9 "$pid" 2>/dev/null
        fi
        log_ok "$name 已停止 (PID: $pid)"
    else
        log_warn "$name 未在运行"
    fi
}

echo "========== 停止 OMC 服务 =========="

# 前端：vite 启动后进程名是 node，用 PID 文件为主
stop_process "前端"         "$RUN_DIR/frontend.pid" ""
stop_process "omcgo-app"    "$RUN_DIR/app.pid"      "omcgo-app"
stop_process "omcgo-acs"    "$RUN_DIR/acs.pid"       "omcgo-acs"
stop_process "omcgo-worker" "$RUN_DIR/worker.pid"    "omcgo-worker"
stop_process "NATS"         "$RUN_DIR/nats.pid"      "nats-server"
stop_process "MinIO"        "$RUN_DIR/minio.pid"     "minio"

# PostgreSQL 和 Redis 为有状态依赖，默认跨 restart 保持运行（restart-all 依赖它们在 migrate 阶段可用）。
# 如需完全停止：bash run/scripts/stop-all.sh --with-db
if [ "${1:-}" = "--with-db" ]; then
    stop_process "Redis" "$RUN_DIR/redis.pid" "redis-server"
    echo -n "PostgreSQL ... "
    PGDATA="$(brew --prefix postgresql@16 2>/dev/null | sed 's|/opt/postgresql@16|/var/postgresql@16|')"
    [ -d "$PGDATA" ] || PGDATA="/opt/homebrew/var/postgresql@16"
    if pg_isready -h localhost -q 2>/dev/null; then
        pg_ctl -D "$PGDATA" -m fast stop >/dev/null 2>&1
        rm -f "$RUN_DIR/postgres.pid"
        log_ok "PostgreSQL 已停止"
    else
        rm -f "$RUN_DIR/postgres.pid"
        log_warn "PostgreSQL 未在运行"
    fi
else
    echo ""
    echo "注意: PostgreSQL 和 Redis 保持运行以便 restart-all 迁移阶段使用。"
    echo "如需完全停止: bash run/scripts/stop-all.sh --with-db"
fi

echo "========== 完成 =========="

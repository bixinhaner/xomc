#!/bin/bash
# OMC 服务状态检查

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}● 运行中${NC}  $1"; }
fail() { echo -e "  ${RED}○ 未运行${NC}  $1"; }
warn() { echo -e "  ${YELLOW}● 运行中${NC}  $1"; }

PG_READY="/usr/local/Cellar/postgresql@16/16.13/bin/pg_isready"

echo "╔══════════════════════════════════════╗"
echo "║          OMC 服务状态                ║"
echo "╚══════════════════════════════════════╝"

echo ""
echo "── 基础依赖 ──"

if $PG_READY -h localhost -q 2>/dev/null; then
    ok "PostgreSQL    :5432"
else
    fail "PostgreSQL    :5432"
fi

if redis-cli ping 2>/dev/null | grep -q PONG; then
    ok "Redis         :6379"
else
    fail "Redis         :6379"
fi

if curl -sf http://localhost:8222/healthz >/dev/null 2>&1; then
    ok "NATS          :4222 (监控 :8222)"
else
    fail "NATS          :4222"
fi

if curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
    ok "MinIO         :9000 (控制台 :9001)"
else
    fail "MinIO         :9000"
fi

echo ""
echo "── 应用服务 ──"

# 检查应用进程：先查PID文件，再通过进程名 fallback
check_app() {
    local name="$1"
    local pid_file="$2"
    local proc_name="$3"
    local port="$4"

    local pid=""
    local source=""

    # 1. 尝试 PID 文件
    if [ -f "$pid_file" ]; then
        local file_pid
        file_pid=$(cat "$pid_file")
        if kill -0 "$file_pid" 2>/dev/null; then
            pid="$file_pid"
            source="PID文件"
        fi
    fi

    # 2. PID 文件没有，通过进程名检测
    if [ -z "$pid" ] && [ -n "$proc_name" ]; then
        pid=$(pgrep -x "$proc_name" 2>/dev/null | head -1)
        if [ -n "$pid" ]; then
            source="进程检测"
        fi
    fi

    if [ -n "$pid" ]; then
        if [ "$source" = "进程检测" ]; then
            warn "$name  :$port  (PID: $pid, 无PID文件)"
        else
            ok "$name  :$port  (PID: $pid)"
        fi
    else
        fail "$name  :$port"
    fi
}

check_app "omcgo-app"    "$RUN_DIR/app.pid"      "omcgo-app"    "8081"
check_app "omcgo-acs"    "$RUN_DIR/acs.pid"       "omcgo-acs"    "8080"
check_app "omcgo-worker" "$RUN_DIR/worker.pid"    "omcgo-worker" " -  "
check_app "前端 Vite"     "$RUN_DIR/frontend.pid"  ""             "3000"

echo ""

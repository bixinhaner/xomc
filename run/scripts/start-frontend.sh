#!/bin/bash
# 启动 OMC 前端开发服务器

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$RUN_DIR")"
LOG_DIR="$RUN_DIR/logs"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_ok()   { echo -e "${GREEN}[✓]${NC} $1"; }
log_fail() { echo -e "${RED}[✗]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }

FRONTEND_DIR="$PROJECT_ROOT/omcmb/webcode"
FE_PID_FILE="$RUN_DIR/frontend.pid"

echo "========== 启动前端服务 =========="

# 检查是否已在运行
if [ -f "$FE_PID_FILE" ]; then
    OLD_PID=$(cat "$FE_PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        log_warn "前端已在运行 (PID: $OLD_PID)"
        echo "如需重启，请先执行 stop-all.sh"
        exit 0
    else
        rm -f "$FE_PID_FILE"
    fi
fi

# 检查 node_modules
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
    log_warn "node_modules 不存在，执行 npm install..."
    cd "$FRONTEND_DIR" && npm install
fi

# 启动
cd "$FRONTEND_DIR"
npx vite > "$LOG_DIR/frontend/frontend.log" 2>&1 &
echo $! > "$FE_PID_FILE"
sleep 3

# 验证
if kill -0 "$(cat "$FE_PID_FILE")" 2>/dev/null; then
    log_ok "前端已启动 (PID: $(cat "$FE_PID_FILE"), 端口: 3000)"
    log_ok "日志: $LOG_DIR/frontend/frontend.log"
else
    log_fail "前端启动失败"
    echo "最近日志:"
    tail -10 "$LOG_DIR/frontend/frontend.log" 2>/dev/null
    rm -f "$FE_PID_FILE"
    exit 1
fi

echo "========== 前端服务就绪 =========="

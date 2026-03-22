#!/bin/bash
# 启动 OMC 后端三进程: omcgo-app / omcgo-acs / omcgo-worker

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

OMCGO_DIR="$PROJECT_ROOT/omcgo"
BIN_DIR="$OMCGO_DIR/bin"

# 通过进程名查找已运行的PID（排除 grep 自身）
find_running_pid() {
    local proc_name="$1"
    pgrep -x "$proc_name" 2>/dev/null | head -1
}

# 通过端口查找正在监听的进程PID（仅 LISTEN 状态）
find_pid_by_port() {
    local port="$1"
    lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | awk 'NR>1{print $2}' | head -1
}

echo "========== 启动后端服务 =========="

# 检查二进制是否存在
if [ ! -f "$BIN_DIR/omcgo-app" ] || [ ! -f "$BIN_DIR/omcgo-acs" ] || [ ! -f "$BIN_DIR/omcgo-worker" ]; then
    log_warn "二进制不完整，开始编译..."
    cd "$OMCGO_DIR" && make build
fi

# 通用启动函数
# 参数: name bin pid_file log_file port
start_process() {
    local name="$1"
    local bin="$2"
    local pid_file="$3"
    local log_file="$4"
    local port="$5"

    echo -n "$name ... "

    # 1. 先检查 PID 文件
    if [ -f "$pid_file" ]; then
        local old_pid
        old_pid=$(cat "$pid_file")
        if kill -0 "$old_pid" 2>/dev/null; then
            log_warn "已在运行 (PID: $old_pid, 来源: PID文件)"
            return 0
        else
            rm -f "$pid_file"
        fi
    fi

    # 2. PID 文件不存在，再通过进程名检测
    local existing_pid
    existing_pid=$(find_running_pid "$name")
    if [ -n "$existing_pid" ]; then
        echo "$existing_pid" > "$pid_file"
        log_warn "已在运行 (PID: $existing_pid, 来源: 进程检测，已补录PID文件)"
        return 0
    fi

    # 3. 再通过端口检测（仅对有端口的进程）
    if [ "$port" != "-" ]; then
        local port_pid
        port_pid=$(find_pid_by_port "$port")
        if [ -n "$port_pid" ]; then
            log_fail "端口 $port 被占用 (PID: $port_pid)，请先释放"
            return 1
        fi
    fi

    # 4. 确保日志目录存在
    mkdir -p "$(dirname "$log_file")"

    # 5. 启动（设置开发环境变量）
    cd "$OMCGO_DIR"
    OMCGO_ENV=dev "$bin" > "$log_file" 2>&1 &
    echo $! > "$pid_file"
    sleep 2

    # 6. 验证
    if kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        log_ok "已启动 (PID: $(cat "$pid_file"), 端口: $port)"
    else
        log_fail "启动失败"
        echo "  最近日志:"
        tail -5 "$log_file" 2>/dev/null
        rm -f "$pid_file"
        return 1
    fi
}

start_process "omcgo-app"    "$BIN_DIR/omcgo-app"    "$RUN_DIR/app.pid"    "$LOG_DIR/app/app.log"       "8081"
start_process "omcgo-acs"    "$BIN_DIR/omcgo-acs"    "$RUN_DIR/acs.pid"    "$LOG_DIR/acs/acs.log"       "8080"
start_process "omcgo-worker" "$BIN_DIR/omcgo-worker"  "$RUN_DIR/worker.pid" "$LOG_DIR/worker/worker.log" "-"

echo "========== 后端服务就绪 =========="

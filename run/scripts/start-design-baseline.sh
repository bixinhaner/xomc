#!/bin/bash
# 启动 OMC 设计基线前端（独立 git worktree: ../goomc-design，:3001），用于和当前
# 开发版 (:3000) 并排对比 UI / 设计还原度。

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

DESIGN_WORKTREE="$(dirname "$PROJECT_ROOT")/goomc-design"
DESIGN_WEBCODE="$DESIGN_WORKTREE/omcmb/webcode"
MAIN_WEBCODE="$PROJECT_ROOT/omcmb/webcode"
PID_FILE="$RUN_DIR/design-baseline.pid"
LOG_FILE="$LOG_DIR/frontend/design-baseline.log"

echo "========== 启动设计基线前端 =========="

# 已在运行
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        log_warn "设计基线已在运行 (PID: $OLD_PID)"
        echo "如需重启，请先执行 stop-all.sh"
        exit 0
    else
        rm -f "$PID_FILE"
    fi
fi

# worktree 不存在就提示创建方法，非致命
if [ ! -d "$DESIGN_WEBCODE" ]; then
    log_warn "worktree 不存在: $DESIGN_WORKTREE"
    echo "       创建方法:"
    echo "         cd $PROJECT_ROOT"
    echo "         git branch design-baseline HEAD   # 如分支不存在"
    echo "         git worktree add $DESIGN_WORKTREE design-baseline"
    exit 0
fi

# node_modules 缺失时 symlink 到主仓库，省一次 npm install
if [ ! -e "$DESIGN_WEBCODE/node_modules" ]; then
    ln -s "$MAIN_WEBCODE/node_modules" "$DESIGN_WEBCODE/node_modules"
    log_ok "已 symlink node_modules -> 主仓库"
fi

mkdir -p "$LOG_DIR/frontend"

cd "$DESIGN_WEBCODE"
VITE_USE_MOCK=true npx vite --port 3001 --host 0.0.0.0 > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"
sleep 3

if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    BL_REF=$(git -C "$DESIGN_WORKTREE" log -1 --pretty='%h %s' 2>/dev/null || echo "?")
    log_ok "设计基线已启动 (PID: $(cat "$PID_FILE"), 端口: 3001)"
    log_ok "基线 commit: $BL_REF"
    log_ok "日志: $LOG_FILE"
else
    log_fail "设计基线启动失败"
    echo "最近日志:"
    tail -10 "$LOG_FILE" 2>/dev/null
    rm -f "$PID_FILE"
    exit 1
fi

echo "========== 设计基线前端就绪 =========="

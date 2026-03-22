#!/bin/bash
# OMC 一键重启：停止 → 编译 → 启动

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$RUN_DIR")"
OMCGO_DIR="$PROJECT_ROOT/omcgo"

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "╔══════════════════════════════════════╗"
echo "║       OMC 开发环境一键重启           ║"
echo "╚══════════════════════════════════════╝"
echo ""

# 1. 停止所有服务
bash "$SCRIPT_DIR/stop-all.sh"
echo ""

# 2. 编译后端
echo "========== 编译后端 =========="
cd "$OMCGO_DIR"
if make build; then
    echo -e "${GREEN}[✓]${NC} 编译成功"
else
    echo -e "${RED}[✗]${NC} 编译失败，请修复后重试"
    exit 1
fi
echo ""

# 3. 启动所有服务
bash "$SCRIPT_DIR/start-all.sh"

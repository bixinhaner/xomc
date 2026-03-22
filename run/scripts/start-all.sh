#!/bin/bash
# OMC 一键启动：依赖 → 后端 → 前端

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "╔══════════════════════════════════════╗"
echo "║       OMC 开发环境一键启动           ║"
echo "╚══════════════════════════════════════╝"
echo ""

# 1. 启动依赖
bash "$SCRIPT_DIR/start-deps.sh"
echo ""

# 2. 启动后端
bash "$SCRIPT_DIR/start-backend.sh"
echo ""

# 3. 启动前端
bash "$SCRIPT_DIR/start-frontend.sh"
echo ""

echo "╔══════════════════════════════════════╗"
echo "║           全部服务已启动             ║"
echo "╠══════════════════════════════════════╣"
echo "║  前端:    http://localhost:3000      ║"
echo "║  App:    http://localhost:8081       ║"
echo "║  ACS:    http://localhost:8080       ║"
echo "║  Worker: 后台进程（无端口）          ║"
echo "║  MinIO:  http://localhost:9001       ║"
echo "║  NATS:   http://localhost:8222       ║"
echo "╠══════════════════════════════════════╣"
echo "║  查看状态: bash run/scripts/status.sh║"
echo "║  停止服务: bash run/scripts/stop-all.sh"
echo "╚══════════════════════════════════════╝"

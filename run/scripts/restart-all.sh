#!/bin/bash
# OMC 一键重启：停止 → 清理日志 → 编译 → 迁移 → 启动

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$RUN_DIR")"
OMCGO_DIR="$PROJECT_ROOT/omcgo"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "╔══════════════════════════════════════╗"
echo "║       OMC 开发环境一键重启           ║"
echo "╚══════════════════════════════════════╝"
echo ""

# 1. 停止所有服务（保留 PG/Redis 以便下面 migrate 阶段连接）
bash "$SCRIPT_DIR/stop-all.sh" --keep-db
echo ""

# 2. 清理旧日志和 Redis 临时状态
echo "========== 清理旧日志 =========="
LOG_DIR="$RUN_DIR/logs"
find "$LOG_DIR" -type f \( -name "*.log" -o -name "*.log.gz" \) ! -name ".gitkeep" -delete 2>/dev/null
echo -e "${GREEN}[✓]${NC} 日志已清理"
# 清理 Redis 中的临时状态，避免重启后残留数据干扰新流程
CLEAN_PATTERNS=("provision:discovery:*" "acs:cmdq:*" "acs:session:*")
TOTAL_DEL=0
for pattern in "${CLEAN_PATTERNS[@]}"; do
    DEL_COUNT=$(redis-cli --no-auth-warning keys "$pattern" 2>/dev/null | xargs -r redis-cli --no-auth-warning del 2>/dev/null)
    if [ -n "$DEL_COUNT" ] && [ "$DEL_COUNT" != "0" ]; then
        TOTAL_DEL=$((TOTAL_DEL + DEL_COUNT))
    fi
done
if [ "$TOTAL_DEL" -gt 0 ]; then
    echo -e "${GREEN}[✓]${NC} Redis 临时状态已清理 ($TOTAL_DEL keys)"
fi
echo ""

# 3. 编译后端
echo "========== 编译后端 =========="
cd "$OMCGO_DIR"
if make build; then
    echo -e "${GREEN}[✓]${NC} 编译成功"
else
    echo -e "${RED}[✗]${NC} 编译失败，请修复后重试"
    exit 1
fi
echo ""

# 4. 执行数据库迁移
echo "========== 数据库迁移 =========="
APP_CFG="$OMCGO_DIR/cmd/app/etc/config.local.yaml"
[ ! -f "$APP_CFG" ] && APP_CFG="$OMCGO_DIR/cmd/app/etc/config.dev.yaml"
DB_DSN=$(grep -A1 '^db:' "$APP_CFG" | grep 'dsn:' | sed 's/.*dsn: *"\(.*\)"/\1/')
if [ -n "$DB_DSN" ]; then
    MIGRATE_OUTPUT=$("$OMCGO_DIR/bin/omcgo-migrate" --dsn "$DB_DSN" --path "$OMCGO_DIR/migrations" up 2>&1) && \
        echo -e "${GREEN}[✓]${NC} $MIGRATE_OUTPUT" || \
        echo -e "${YELLOW}[!]${NC} 迁移跳过: $MIGRATE_OUTPUT"
else
    echo -e "${YELLOW}[!]${NC} 未找到数据库 DSN，跳过迁移"
fi
echo ""

# 5. 启动所有服务
bash "$SCRIPT_DIR/start-all.sh"

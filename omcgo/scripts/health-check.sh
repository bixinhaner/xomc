#!/usr/bin/env bash
# OMC Go 服务健康检查脚本
# 用法: ./scripts/health-check.sh

set -euo pipefail

export PATH="/usr/local/opt/postgresql@16/bin:$PATH"
DB_DSN="postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"

echo "=========================================="
echo "  OMC Go 服务健康检查  $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

# 1. PostgreSQL
echo "【1】PostgreSQL"
if pg_isready -q 2>/dev/null; then
    echo "  状态: ✅ 运行中"
    DEVICE_COUNT=$(psql "$DB_DSN" -t -c "SELECT COUNT(*) FROM devices;" 2>/dev/null | tr -d ' ')
    echo "  注册设备数: $DEVICE_COUNT"
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 2. Redis
echo "【2】Redis"
if redis-cli ping 2>/dev/null | grep -q PONG; then
    echo "  状态: ✅ 运行中"
    SESSION_COUNT=$(redis-cli KEYS "acs:session:*" 2>/dev/null | wc -l | tr -d ' ')
    HEARTBEAT_COUNT=$(redis-cli KEYS "acs:heartbeat:*" 2>/dev/null | wc -l | tr -d ' ')
    echo "  活跃会话数: $SESSION_COUNT"
    echo "  心跳设备数: $HEARTBEAT_COUNT"
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 3. NATS
echo "【3】NATS JetStream"
NATS_VER=$(curl -sf http://localhost:8222/varz 2>/dev/null | grep '"version"' | awk -F'"' '{print $4}')
if [ -n "$NATS_VER" ]; then
    echo "  状态: ✅ 运行中 (v$NATS_VER)"
    NATS_CONNS=$(curl -sf http://localhost:8222/varz 2>/dev/null | grep '"connections"' | head -1 | awk -F: '{print $2}' | tr -d ' ,')
    STREAM_INFO=$(curl -sf http://localhost:8222/jsz 2>/dev/null)
    STREAMS=$(echo "$STREAM_INFO" | grep '"streams"' | awk -F: '{print $2}' | tr -d ' ,')
    MSGS=$(echo "$STREAM_INFO" | grep '"messages"' | tail -1 | awk -F: '{print $2}' | tr -d ' ,')
    echo "  客户端连接数: $NATS_CONNS"
    echo "  JetStream 流: $STREAMS"
    echo "  待处理消息数: $MSGS"
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 4. MinIO
echo "【4】MinIO"
if curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
    echo "  状态: ✅ 运行中"
    echo "  控制台: http://localhost:9001"
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 5. ACS Engine
echo "【5】omcgo-acs (TR069 ACS 引擎)"
ACS_PID=$(pgrep -f "omcgo-acs" 2>/dev/null | head -1)
if [ -n "$ACS_PID" ]; then
    echo "  状态: ✅ 运行中 (PID $ACS_PID)"
    echo "  监听: http://localhost:7547/acs"
    METRICS=$(curl -sf http://localhost:9090/metrics 2>/dev/null || echo "")
    if [ -n "$METRICS" ]; then
        INFORM_TOTAL=$(echo "$METRICS" | grep "^acs_inform_total" | awk '{sum+=$2} END {printf "%d", sum}')
        ACTIVE_SESSIONS=$(echo "$METRICS" | grep "^acs_global_active_sessions " | awk '{print $2}')
        echo "  累计 Inform 次数: ${INFORM_TOTAL:-0}"
        echo "  当前活跃会话: ${ACTIVE_SESSIONS:-0}"
    fi
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 6. App
echo "【6】omcgo-app (管理面 REST API)"
APP_PID=$(pgrep -f "omcgo-app" 2>/dev/null | head -1)
if [ -n "$APP_PID" ]; then
    echo "  状态: ✅ 运行中 (PID $APP_PID)"
    echo "  REST API: http://localhost:8080"
else
    echo "  状态: ❌ 未运行"
fi
echo ""

# 7. Worker
echo "【7】omcgo-worker (后台工作进程)"
WORKER_PID=$(pgrep -f "omcgo-worker" 2>/dev/null | head -1)
if [ -n "$WORKER_PID" ]; then
    echo "  状态: ✅ 运行中 (PID $WORKER_PID)"
else
    echo "  状态: ⚠️  未启动（PM/MR/KPI 处理不可用）"
fi
echo ""

# 8. 设备列表
echo "=========================================="
echo "  已注册设备列表"
echo "=========================================="
psql "$DB_DSN" -c \
  "SELECT serial_number AS \"序列号\",
          manufacturer AS \"厂商\",
          oui AS \"OUI\",
          carrier AS \"运营商\",
          status AS \"状态\",
          firmware_version AS \"固件\",
          last_inform_at AS \"最后上报\"
   FROM devices ORDER BY last_inform_at DESC NULLS LAST;" 2>/dev/null || echo "(数据库不可用)"

echo ""
echo "=========================================="
echo "  最近 Inform 活动（Redis 会话）"
echo "=========================================="
for key in $(redis-cli KEYS "acs:session:*" 2>/dev/null); do
    SN=${key#acs:session:}
    TTL=$(redis-cli TTL "$key" 2>/dev/null)
    echo "  设备: $SN (会话剩余 TTL: ${TTL}s)"
done
[ "$(redis-cli KEYS 'acs:session:*' 2>/dev/null | wc -l | tr -d ' ')" -eq 0 ] && echo "  (无活跃会话)"

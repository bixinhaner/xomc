#!/bin/bash
# 启动 OMC 基础依赖服务 (PostgreSQL, Redis, NATS, MinIO)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$RUN_DIR/logs"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_ok()   { echo -e "${GREEN}[✓]${NC} $1"; }
log_fail() { echo -e "${RED}[✗]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }

PG_READY="/usr/local/Cellar/postgresql@16/16.13/bin/pg_isready"

echo "========== 启动基础依赖 =========="

# --- PostgreSQL ---
echo -n "PostgreSQL ... "
if $PG_READY -h localhost -q 2>/dev/null; then
    log_ok "已运行"
else
    # 清理残留 PID 文件
    PID_FILE="/usr/local/var/postgresql@16/postmaster.pid"
    if [ -f "$PID_FILE" ]; then
        PG_PID=$(head -1 "$PID_FILE")
        if ! ps -p "$PG_PID" -o comm= 2>/dev/null | grep -q postgres; then
            rm -f "$PID_FILE"
            log_warn "已清理残留 postmaster.pid"
        fi
    fi
    brew services restart postgresql@16 >/dev/null 2>&1
    sleep 2
    if $PG_READY -h localhost -q 2>/dev/null; then
        log_ok "已启动"
    else
        log_fail "启动失败，请检查: brew services info postgresql@16"
        exit 1
    fi
fi

# --- Redis ---
echo -n "Redis ...... "
if redis-cli ping 2>/dev/null | grep -q PONG; then
    log_ok "已运行"
else
    brew services restart redis >/dev/null 2>&1
    sleep 1
    if redis-cli ping 2>/dev/null | grep -q PONG; then
        log_ok "已启动"
    else
        log_fail "启动失败，请检查: brew services info redis"
        exit 1
    fi
fi

# --- NATS ---
echo -n "NATS ....... "
if curl -sf http://localhost:8222/healthz >/dev/null 2>&1; then
    log_ok "已运行"
else
    NATS_DATA_DIR="$HOME/data/nats"
    mkdir -p "$NATS_DATA_DIR"
    nats-server --jetstream --store_dir "$NATS_DATA_DIR" --http_port 8222 \
        > "$LOG_DIR/nats.log" 2>&1 &
    echo $! > "$RUN_DIR/nats.pid"
    sleep 1
    if curl -sf http://localhost:8222/healthz >/dev/null 2>&1; then
        log_ok "已启动 (PID: $(cat "$RUN_DIR/nats.pid"))"
    else
        log_fail "启动失败，请查看: $LOG_DIR/nats.log"
        exit 1
    fi
fi

# --- MinIO ---
echo -n "MinIO ...... "
if curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
    log_ok "已运行"
else
    MINIO_DATA_DIR="$HOME/data/minio"
    mkdir -p "$MINIO_DATA_DIR"
    MINIO_ROOT_USER=minioadmin MINIO_ROOT_PASSWORD=minioadmin \
        minio server "$MINIO_DATA_DIR" --console-address ":9001" \
        > "$LOG_DIR/minio.log" 2>&1 &
    echo $! > "$RUN_DIR/minio.pid"
    sleep 2
    if curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
        log_ok "已启动 (PID: $(cat "$RUN_DIR/minio.pid"))"
    else
        log_fail "启动失败，请查看: $LOG_DIR/minio.log"
        exit 1
    fi
fi

echo "========== 依赖服务就绪 =========="

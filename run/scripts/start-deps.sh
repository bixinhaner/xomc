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

echo "========== 启动基础依赖 =========="

# --- PostgreSQL ---
echo -n "PostgreSQL ... "
if pg_isready -h localhost -q 2>/dev/null; then
    log_ok "已运行"
else
    PGDATA="$(brew --prefix postgresql@16 2>/dev/null | sed 's|/opt/postgresql@16|/var/postgresql@16|')"
    [ -d "$PGDATA" ] || PGDATA="/opt/homebrew/var/postgresql@16"
    # 清理已崩溃进程留下的 postmaster.pid
    if [ -f "$PGDATA/postmaster.pid" ]; then
        PG_PID=$(head -1 "$PGDATA/postmaster.pid")
        if ! kill -0 "$PG_PID" 2>/dev/null; then
            rm -f "$PGDATA/postmaster.pid"
            log_warn "已清理残留 postmaster.pid"
        fi
    fi
    pg_ctl -D "$PGDATA" -l "$PGDATA/server.log" -o "-p 5432" start -w >/dev/null 2>&1
    sleep 1
    if pg_isready -h localhost -q 2>/dev/null; then
        head -1 "$PGDATA/postmaster.pid" > "$RUN_DIR/postgres.pid"
        log_ok "已启动 (PID: $(cat "$RUN_DIR/postgres.pid"))"
    else
        log_fail "启动失败，请查看: $PGDATA/server.log"
        exit 1
    fi
fi

# --- Redis ---
echo -n "Redis ...... "
if redis-cli ping 2>/dev/null | grep -q PONG; then
    log_ok "已运行"
else
    REDIS_DATA_DIR="$HOME/data/redis"
    mkdir -p "$REDIS_DATA_DIR"
    redis-server --daemonize yes \
        --dir "$REDIS_DATA_DIR" \
        --logfile "$REDIS_DATA_DIR/redis.log" \
        --pidfile "$RUN_DIR/redis.pid" \
        --port 6379 >/dev/null 2>&1
    sleep 1
    if redis-cli ping 2>/dev/null | grep -q PONG; then
        log_ok "已启动 (PID: $(cat "$RUN_DIR/redis.pid"))"
    else
        log_fail "启动失败，请查看: $REDIS_DATA_DIR/redis.log"
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

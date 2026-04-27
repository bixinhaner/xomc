#!/usr/bin/env bash
# =============================================================================
# OMC PostgreSQL 定时备份脚本（T-0042 W1.8）
#
# 用途：
#   定时（cron）调用，输出 pg_dump custom-format 备份到 ./backups/<env>/，
#   并按 retention-days 清理过期备份。失败 retry 3 次（指数退避）。
#
# 用法：
#   bash db_backup.sh --env dev
#   bash db_backup.sh --env prod --db-host pg.prod.internal --db-port 5432 \
#                     --retention-days 30
#
# 必备环境变量（任选其一提供密码，**禁止明文写在脚本里**）：
#   - PGPASSWORD          直接读
#   - 或 ~/.pgpass        Postgres 标准凭据文件
#
# 退出码：
#   0 = 成功
#   非零 = 失败（重试 3 次仍失败 / 参数非法 / pg_dump 不可用）
# =============================================================================
set -euo pipefail
IFS=$'\n\t'

# ---------- 默认参数 ----------
ENV_NAME="dev"
DB_HOST="${OMC_BACKUP_DB_HOST:-localhost}"
DB_PORT="${OMC_BACKUP_DB_PORT:-5432}"
DB_USER="${OMC_BACKUP_DB_USER:-omcgo}"
DB_NAME="${OMC_BACKUP_DB_NAME:-omcgo}"
RETENTION_DAYS=30
COMPRESS_LEVEL=6
MAX_RETRY=3

# 备份根目录（脚本所在目录的上两级 / backups）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
BACKUP_ROOT="${OMC_BACKUP_ROOT:-${REPO_ROOT}/backups}"

# ---------- 参数解析 ----------
usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Options:
  --env <name>             环境名（dev|staging|prod，默认 dev）
  --db-host <host>         DB 主机（默认 localhost）
  --db-port <port>         DB 端口（默认 5432）
  --db-user <user>         DB 用户（默认 omcgo）
  --db-name <name>         DB 库名（默认 omcgo）
  --retention-days <days>  保留天数（默认 30）
  --backup-root <dir>      备份根目录（默认 \${REPO_ROOT}/backups）
  -h | --help              显示此帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env)             ENV_NAME="$2"; shift 2 ;;
        --db-host)         DB_HOST="$2"; shift 2 ;;
        --db-port)         DB_PORT="$2"; shift 2 ;;
        --db-user)         DB_USER="$2"; shift 2 ;;
        --db-name)         DB_NAME="$2"; shift 2 ;;
        --retention-days)  RETENTION_DAYS="$2"; shift 2 ;;
        --backup-root)     BACKUP_ROOT="$2"; shift 2 ;;
        -h|--help)         usage; exit 0 ;;
        *) echo "[ERROR] Unknown arg: $1" >&2; usage; exit 2 ;;
    esac
done

# ---------- 校验依赖 ----------
if ! command -v pg_dump >/dev/null 2>&1; then
    echo "[ERROR] pg_dump 不在 PATH，请安装 postgresql client" >&2
    exit 3
fi

# ---------- 准备目录 ----------
BACKUP_DIR="${BACKUP_ROOT}/${ENV_NAME}"
mkdir -p "${BACKUP_DIR}"
LOG_FILE="${BACKUP_DIR}/backup.log"

ts() { date +'%Y-%m-%d %H:%M:%S%z'; }
log() {
    local msg="[$(ts)] [$$] $*"
    echo "${msg}"
    echo "${msg}" >> "${LOG_FILE}"
}

# ---------- 输出文件名 ----------
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
DUMP_FILE="${BACKUP_DIR}/omc-${TIMESTAMP}.dump"
META_FILE="${BACKUP_DIR}/omc-${TIMESTAMP}.meta"

log "==== backup start env=${ENV_NAME} host=${DB_HOST}:${DB_PORT} db=${DB_NAME} target=${DUMP_FILE} ===="

# ---------- 重试封装 ----------
run_pg_dump() {
    # 注意：不在命令行携带密码；依赖 PGPASSWORD env / .pgpass
    pg_dump \
        --host="${DB_HOST}" \
        --port="${DB_PORT}" \
        --username="${DB_USER}" \
        --dbname="${DB_NAME}" \
        --format=custom \
        --compress="${COMPRESS_LEVEL}" \
        --no-owner \
        --no-privileges \
        --verbose \
        --file="${DUMP_FILE}" \
        2>>"${LOG_FILE}"
}

attempt=1
START_EPOCH=$(date +%s)
while true; do
    log "attempt ${attempt}/${MAX_RETRY}: invoking pg_dump..."
    if run_pg_dump; then
        log "attempt ${attempt} succeeded"
        break
    fi
    rc=$?
    log "attempt ${attempt} failed rc=${rc}"
    if [[ ${attempt} -ge ${MAX_RETRY} ]]; then
        log "[FATAL] pg_dump 失败 ${MAX_RETRY} 次，放弃"
        # 清理半成品
        [[ -f "${DUMP_FILE}" ]] && rm -f "${DUMP_FILE}"
        exit 4
    fi
    sleep_secs=$(( 2 ** attempt ))   # 2, 4, 8 ...
    log "backoff ${sleep_secs}s before retry"
    sleep "${sleep_secs}"
    attempt=$(( attempt + 1 ))
done

END_EPOCH=$(date +%s)
DURATION=$(( END_EPOCH - START_EPOCH ))

# ---------- 计算 size + 校验和 ----------
SIZE_BYTES=$(wc -c < "${DUMP_FILE}" | tr -d ' ')
SIZE_HUMAN=$(du -h "${DUMP_FILE}" | awk '{print $1}')

if command -v sha256sum >/dev/null 2>&1; then
    CHECKSUM=$(sha256sum "${DUMP_FILE}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    CHECKSUM=$(shasum -a 256 "${DUMP_FILE}" | awk '{print $1}')
else
    CHECKSUM="unavailable"
fi

# ---------- 写 meta ----------
cat > "${META_FILE}" <<EOF
file=${DUMP_FILE}
env=${ENV_NAME}
host=${DB_HOST}
port=${DB_PORT}
db=${DB_NAME}
timestamp=${TIMESTAMP}
size_bytes=${SIZE_BYTES}
size_human=${SIZE_HUMAN}
duration_seconds=${DURATION}
sha256=${CHECKSUM}
EOF

log "size=${SIZE_HUMAN} (${SIZE_BYTES} bytes) duration=${DURATION}s sha256=${CHECKSUM}"

# ---------- 清理过期备份 ----------
log "cleanup: 清理 > ${RETENTION_DAYS} 天的旧 dump/meta"
removed=0
# find -mtime +N : 严格大于 N*24h 的文件
while IFS= read -r -d '' old; do
    rm -f "${old}"
    log "  removed ${old}"
    removed=$(( removed + 1 ))
done < <(find "${BACKUP_DIR}" -maxdepth 1 -type f \( -name 'omc-*.dump' -o -name 'omc-*.meta' \) -mtime "+${RETENTION_DAYS}" -print0 2>/dev/null)
log "cleanup: removed=${removed} files"

# ---------- 输出 stdout 摘要（cron / 日志收集器友好） ----------
cat <<EOF
{
  "status": "ok",
  "env": "${ENV_NAME}",
  "file": "${DUMP_FILE}",
  "size_bytes": ${SIZE_BYTES},
  "size_human": "${SIZE_HUMAN}",
  "duration_seconds": ${DURATION},
  "sha256": "${CHECKSUM}",
  "retention_days": ${RETENTION_DAYS},
  "old_files_removed": ${removed}
}
EOF

log "==== backup done ===="
exit 0

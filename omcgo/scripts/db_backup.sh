#!/usr/bin/env bash
# =============================================================================
# OMC PostgreSQL 定时备份脚本（T-0042 W1.8 base + T-0067 W3.H.3 异地扩展）
#
# 用途：
#   定时（cron）调用，输出 pg_dump custom-format 备份到 ./backups/<env>/，
#   并按 retention-days 清理过期备份。失败 retry 3 次（指数退避）。
#   T-0067 扩展：备份成功后，把 dump+meta 上传到异地 S3/MinIO 桶
#   （RPO 目标 < 15min，依赖更高频的 cron / 文件级同步），
#   异地上传失败 **不阻塞** 本地备份成功，仅 warn + metric 计数。
#
# 用法：
#   # 仅本地（W1.8 base 行为）
#   bash db_backup.sh --env dev
#
#   # 本地 + 异地 S3（推荐生产）
#   OFFSITE_S3_ENDPOINT=https://minio.dr.example.com \
#   OFFSITE_S3_BUCKET=omc-backups-offsite \
#   OFFSITE_S3_REGION=cn-east-1 \
#   OFFSITE_S3_ACCESS_KEY=*** OFFSITE_S3_SECRET_KEY=*** \
#   bash db_backup.sh --env prod --db-host pg.prod.internal --retention-days 60
#
# 必备环境变量（任选其一提供密码，**禁止明文写在脚本里**）：
#   - PGPASSWORD          直接读
#   - 或 ~/.pgpass        Postgres 标准凭据文件
#
# 异地上传环境变量（**全部 optional**，未设置则跳过异地段，行为完全等同 W1.8 base）：
#   - OFFSITE_S3_ENDPOINT     S3 endpoint URL（如 https://s3.dr.example.com / minio）
#   - OFFSITE_S3_BUCKET       目标 bucket 名（必需）
#   - OFFSITE_S3_REGION       region（默认 us-east-1，MinIO 任意值即可）
#   - OFFSITE_S3_ACCESS_KEY   access key（必需，aws/mc 二选一会用）
#   - OFFSITE_S3_SECRET_KEY   secret key（必需）
#   - OFFSITE_S3_PREFIX       对象前缀（默认 backups/<env>/）
#   - OFFSITE_S3_CLIENT       上传客户端：auto|mc|aws（默认 auto，优先 mc 再 aws）
#
# Prometheus metric（仅以 stdout JSON 字段输出 + log 行打点，不动 Go 代码）：
#   omc_backup_offsite_total{status="success|failure|skipped"} +1 / 次
#   omc_backup_offsite_duration_seconds                       上传耗时
#   metric 接入待 monitoring agent（mtail / promtail）解析本脚本日志/JSON
#
# 退出码：
#   0 = 成功（异地失败也是 0，只要本地备份成功）
#   非零 = 本地备份失败（重试 3 次仍失败 / 参数非法 / pg_dump 不可用）
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

# ---------- 异地备份（T-0067 W3.H.3） ----------
# 全部走环境变量，**不**接受 CLI flag，避免命令行泄露凭据
OFFSITE_S3_ENDPOINT="${OFFSITE_S3_ENDPOINT:-}"
OFFSITE_S3_BUCKET="${OFFSITE_S3_BUCKET:-}"
OFFSITE_S3_REGION="${OFFSITE_S3_REGION:-us-east-1}"
OFFSITE_S3_ACCESS_KEY="${OFFSITE_S3_ACCESS_KEY:-}"
OFFSITE_S3_SECRET_KEY="${OFFSITE_S3_SECRET_KEY:-}"
OFFSITE_S3_PREFIX="${OFFSITE_S3_PREFIX:-}"
OFFSITE_S3_CLIENT="${OFFSITE_S3_CLIENT:-auto}"
OFFSITE_MAX_RETRY="${OFFSITE_MAX_RETRY:-3}"

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
offsite_endpoint=${OFFSITE_S3_ENDPOINT}
offsite_bucket=${OFFSITE_S3_BUCKET}
EOF

log "size=${SIZE_HUMAN} (${SIZE_BYTES} bytes) duration=${DURATION}s sha256=${CHECKSUM}"

# ---------- 异地 S3/MinIO 上传（T-0067 W3.H.3） ----------
# 默认值
OFFSITE_STATUS="skipped"
OFFSITE_DURATION=0
OFFSITE_TARGET=""
OFFSITE_REASON=""

offsite_pick_client() {
    # 优先 mc（MinIO client，对 MinIO 端点更兼容），次选 aws cli
    case "${OFFSITE_S3_CLIENT}" in
        mc)
            command -v mc >/dev/null 2>&1 && { echo "mc"; return 0; }
            return 1
            ;;
        aws)
            command -v aws >/dev/null 2>&1 && { echo "aws"; return 0; }
            return 1
            ;;
        auto|"")
            command -v mc >/dev/null 2>&1 && { echo "mc"; return 0; }
            command -v aws >/dev/null 2>&1 && { echo "aws"; return 0; }
            return 1
            ;;
        *)
            return 1
            ;;
    esac
}

offsite_upload_mc() {
    # mc 别名注册（每次调用刷新，避免 stale 凭据）
    local alias="omcoffsite"
    local prefix="${OFFSITE_S3_PREFIX:-backups/${ENV_NAME}/}"
    # 去掉前后多余 /
    prefix="${prefix#/}"
    [[ "${prefix}" != */ ]] && prefix="${prefix}/"
    local target="${alias}/${OFFSITE_S3_BUCKET}/${prefix}"

    if ! mc alias set "${alias}" "${OFFSITE_S3_ENDPOINT}" \
            "${OFFSITE_S3_ACCESS_KEY}" "${OFFSITE_S3_SECRET_KEY}" \
            --api S3v4 >>"${LOG_FILE}" 2>&1; then
        log "  offsite[mc]: alias set failed"
        return 1
    fi

    if ! mc cp "${DUMP_FILE}" "${target}" >>"${LOG_FILE}" 2>&1; then
        log "  offsite[mc]: cp dump failed"
        return 2
    fi
    if ! mc cp "${META_FILE}" "${target}" >>"${LOG_FILE}" 2>&1; then
        log "  offsite[mc]: cp meta failed"
        return 3
    fi
    OFFSITE_TARGET="s3://${OFFSITE_S3_BUCKET}/${prefix}$(basename "${DUMP_FILE}")"
    return 0
}

offsite_upload_aws() {
    local prefix="${OFFSITE_S3_PREFIX:-backups/${ENV_NAME}/}"
    prefix="${prefix#/}"
    [[ "${prefix}" != */ ]] && prefix="${prefix}/"
    local target_dump="s3://${OFFSITE_S3_BUCKET}/${prefix}$(basename "${DUMP_FILE}")"
    local target_meta="s3://${OFFSITE_S3_BUCKET}/${prefix}$(basename "${META_FILE}")"

    # AWS CLI 通过 env 接受凭据 + endpoint
    local -a aws_args=(--endpoint-url "${OFFSITE_S3_ENDPOINT}" --region "${OFFSITE_S3_REGION}")
    if ! AWS_ACCESS_KEY_ID="${OFFSITE_S3_ACCESS_KEY}" \
         AWS_SECRET_ACCESS_KEY="${OFFSITE_S3_SECRET_KEY}" \
         aws "${aws_args[@]}" s3 cp "${DUMP_FILE}" "${target_dump}" >>"${LOG_FILE}" 2>&1; then
        log "  offsite[aws]: cp dump failed"
        return 2
    fi
    if ! AWS_ACCESS_KEY_ID="${OFFSITE_S3_ACCESS_KEY}" \
         AWS_SECRET_ACCESS_KEY="${OFFSITE_S3_SECRET_KEY}" \
         aws "${aws_args[@]}" s3 cp "${META_FILE}" "${target_meta}" >>"${LOG_FILE}" 2>&1; then
        log "  offsite[aws]: cp meta failed"
        return 3
    fi
    OFFSITE_TARGET="${target_dump}"
    return 0
}

run_offsite_upload() {
    # 任一关键 env 缺失则视为未启用（与 W1.8 base 行为兼容）
    if [[ -z "${OFFSITE_S3_ENDPOINT}" || -z "${OFFSITE_S3_BUCKET}" \
          || -z "${OFFSITE_S3_ACCESS_KEY}" || -z "${OFFSITE_S3_SECRET_KEY}" ]]; then
        log "offsite: 异地配置未启用（OFFSITE_S3_* 不全），跳过 — metric: omc_backup_offsite_total{status=\"skipped\"}"
        OFFSITE_STATUS="skipped"
        OFFSITE_REASON="not_configured"
        return 0
    fi

    local client
    if ! client=$(offsite_pick_client); then
        log "offsite: 找不到 mc 或 aws CLI（需要 minio mc 或 awscli），跳过"
        OFFSITE_STATUS="failure"
        OFFSITE_REASON="no_client"
        return 0
    fi

    log "offsite: 启动异地上传 client=${client} endpoint=${OFFSITE_S3_ENDPOINT} bucket=${OFFSITE_S3_BUCKET}"
    local off_start off_end attempt rc
    off_start=$(date +%s)
    attempt=1
    while true; do
        if [[ "${client}" == "mc" ]]; then
            offsite_upload_mc; rc=$?
        else
            offsite_upload_aws; rc=$?
        fi
        if [[ ${rc} -eq 0 ]]; then
            off_end=$(date +%s)
            OFFSITE_DURATION=$(( off_end - off_start ))
            OFFSITE_STATUS="success"
            log "offsite: 上传成功 target=${OFFSITE_TARGET} duration=${OFFSITE_DURATION}s — metric: omc_backup_offsite_total{status=\"success\"}"
            return 0
        fi
        log "offsite: attempt ${attempt}/${OFFSITE_MAX_RETRY} 失败 rc=${rc}"
        if [[ ${attempt} -ge ${OFFSITE_MAX_RETRY} ]]; then
            off_end=$(date +%s)
            OFFSITE_DURATION=$(( off_end - off_start ))
            OFFSITE_STATUS="failure"
            OFFSITE_REASON="upload_failed_rc${rc}"
            log "offsite: [WARN] 上传失败 ${OFFSITE_MAX_RETRY} 次（不阻塞本地备份） — metric: omc_backup_offsite_total{status=\"failure\"}"
            return 0
        fi
        sleep $(( 2 ** attempt ))
        attempt=$(( attempt + 1 ))
    done
}

run_offsite_upload || true   # 双保险：永不让异地段失败影响本地备份退出码

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
  "old_files_removed": ${removed},
  "offsite_status": "${OFFSITE_STATUS}",
  "offsite_reason": "${OFFSITE_REASON}",
  "offsite_target": "${OFFSITE_TARGET}",
  "offsite_duration_seconds": ${OFFSITE_DURATION}
}
EOF

log "==== backup done ===="
exit 0

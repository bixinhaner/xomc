#!/usr/bin/env bash
# =============================================================================
# OMC PostgreSQL 恢复演练脚本（T-0042 W1.8）
#
# 用途：
#   把 db_backup.sh 产出的 .dump 文件 restore 到一个**临时数据库**，跑校验，
#   测算 RTO（开始 -> 完成秒数），再销毁临时库。生产数据**不被改动**。
#
# 用法：
#   bash db_restore_drill.sh ./backups/dev/omc-20260427-194500.dump
#   bash db_restore_drill.sh --db-host pg.staging --db-port 5432 \
#                            ./backups/staging/omc-xxx.dump
#
# 必备环境变量（任选其一）：
#   - PGPASSWORD          直接读
#   - 或 ~/.pgpass        Postgres 标准凭据文件
#
# 退出码：
#   0 = 演练全部通过
#   1 = 输入参数非法
#   2 = pg_restore 失败
#   3 = 校验 SQL 失败
#   其他非零 = 临时 DB 创建/销毁失败
# =============================================================================
set -euo pipefail
IFS=$'\n\t'

DB_HOST="${OMC_BACKUP_DB_HOST:-localhost}"
DB_PORT="${OMC_BACKUP_DB_PORT:-5432}"
DB_USER="${OMC_BACKUP_DB_USER:-omcgo}"
SOURCE_DB="${OMC_BACKUP_DB_NAME:-omcgo}"   # 用于行数对比基准
DUMP_FILE=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [options] <dump-file>

Options:
  --db-host <host>     DB 主机（默认 localhost）
  --db-port <port>     DB 端口（默认 5432）
  --db-user <user>     DB 用户（默认 omcgo）
  --source-db <name>   用于行数比对的源库（默认 omcgo）
  -h | --help          帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --db-host)    DB_HOST="$2"; shift 2 ;;
        --db-port)    DB_PORT="$2"; shift 2 ;;
        --db-user)    DB_USER="$2"; shift 2 ;;
        --source-db)  SOURCE_DB="$2"; shift 2 ;;
        -h|--help)    usage; exit 0 ;;
        --) shift; break ;;
        -*) echo "[ERROR] Unknown flag: $1" >&2; usage; exit 1 ;;
        *) DUMP_FILE="$1"; shift ;;
    esac
done

if [[ -z "${DUMP_FILE}" ]]; then
    echo "[ERROR] 缺少 dump 文件路径" >&2
    usage
    exit 1
fi
if [[ ! -f "${DUMP_FILE}" ]]; then
    echo "[ERROR] dump 文件不存在: ${DUMP_FILE}" >&2
    exit 1
fi

if ! command -v pg_restore >/dev/null 2>&1 || ! command -v psql >/dev/null 2>&1; then
    echo "[ERROR] pg_restore / psql 不在 PATH" >&2
    exit 1
fi

# 临时 DB 名（带 PID + 时间戳，避免多实例冲突）
TMP_DB="omc_restore_drill_$(date +%s)_$$"

ts() { date +'%Y-%m-%d %H:%M:%S%z'; }
log() { echo "[$(ts)] $*"; }

PSQL_BASE=(psql --host="${DB_HOST}" --port="${DB_PORT}" --username="${DB_USER}" --no-psqlrc --tuples-only --no-align)

# 预查源库行数（恢复后做对比）
log "==== restore drill begin dump=${DUMP_FILE} tmp_db=${TMP_DB} ===="
log "step 0: 抓取源库 ${SOURCE_DB} 关键表行数作为基准"
SRC_DEVICES=$("${PSQL_BASE[@]}" --dbname="${SOURCE_DB}" -c "SELECT count(*) FROM devices;" 2>/dev/null || echo "n/a")
SRC_USERS=$("${PSQL_BASE[@]}" --dbname="${SOURCE_DB}" -c "SELECT count(*) FROM users;" 2>/dev/null || echo "n/a")
SRC_ALARMLIBS=$("${PSQL_BASE[@]}" --dbname="${SOURCE_DB}" -c "SELECT count(*) FROM alarm_libraries;" 2>/dev/null || echo "n/a")
log "  source baseline: devices=${SRC_DEVICES} users=${SRC_USERS} alarm_libraries=${SRC_ALARMLIBS}"

cleanup() {
    local rc=$?
    log "step 99: 销毁临时库 ${TMP_DB}"
    "${PSQL_BASE[@]}" --dbname=postgres \
        -c "DROP DATABASE IF EXISTS \"${TMP_DB}\";" >/dev/null 2>&1 || \
        log "  WARN: 销毁失败（手工清理：DROP DATABASE \"${TMP_DB}\";）"
    exit "${rc}"
}
trap cleanup EXIT

# ---------- step 1: 创建临时库 ----------
# RTO 计时从这里开始，包含 CREATE DATABASE + pg_restore 全过程
START_EPOCH=$(date +%s)
log "step 1: 创建临时库 ${TMP_DB} (rto_start_epoch=${START_EPOCH})"
"${PSQL_BASE[@]}" --dbname=postgres \
    -c "CREATE DATABASE \"${TMP_DB}\" WITH OWNER \"${DB_USER}\" ENCODING 'UTF8';" >/dev/null

# ---------- step 2: pg_restore ----------
log "step 2: pg_restore -> ${TMP_DB}"

# --no-owner / --no-privileges 与 db_backup.sh 一致
# --exit-on-error 让早期错误立刻报出
# 失败时仍走 trap 销毁临时库
if ! pg_restore \
        --host="${DB_HOST}" \
        --port="${DB_PORT}" \
        --username="${DB_USER}" \
        --dbname="${TMP_DB}" \
        --no-owner \
        --no-privileges \
        --exit-on-error \
        --verbose \
        "${DUMP_FILE}" 2> >(tail -50 >&2); then
    log "[FATAL] pg_restore 失败"
    exit 2
fi

END_EPOCH=$(date +%s)
RTO_SECONDS=$(( END_EPOCH - START_EPOCH ))
log "step 2 done: pg_restore 成功 RTO=${RTO_SECONDS}s"

# ---------- step 3: 校验 ----------
log "step 3: 跑校验 SQL"

PSQL_TMP=(psql --host="${DB_HOST}" --port="${DB_PORT}" --username="${DB_USER}" --dbname="${TMP_DB}" --no-psqlrc --tuples-only --no-align)

verify() {
    local label="$1"
    local sql="$2"
    local expect_relation="${3:-1}"   # 期望关系符："1" 表示期望非零；其他直接做相等比较
    local val
    val=$("${PSQL_TMP[@]}" -c "${sql}" 2>&1 || true)
    if [[ -z "${val}" ]]; then
        log "  CHECK ${label}: FAIL (空结果)"
        return 1
    fi
    if [[ "${expect_relation}" == "1" ]]; then
        if [[ "${val}" =~ ^[0-9]+$ ]] && (( val > 0 )); then
            log "  CHECK ${label}: OK (val=${val})"
            return 0
        fi
        log "  CHECK ${label}: FAIL (val=${val})"
        return 1
    fi
    if [[ "${val}" == "${expect_relation}" ]]; then
        log "  CHECK ${label}: OK (val=${val})"
        return 0
    fi
    log "  CHECK ${label}: FAIL (val=${val} expect=${expect_relation})"
    return 1
}

CHECK_FAILED=0

verify "tables_exist (>0)"           "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"  || CHECK_FAILED=1
verify "devices_nonempty"            "SELECT count(*) FROM devices;"                 || CHECK_FAILED=1
verify "users_nonempty"              "SELECT count(*) FROM users;"                   || CHECK_FAILED=1
verify "alarm_libraries_nonempty"    "SELECT count(*) FROM alarm_libraries;"         || CHECK_FAILED=1

# 行数对比（容忍 < 1% 偏差，因为源库可能在备份后又有写入）
if [[ "${SRC_DEVICES}" =~ ^[0-9]+$ ]]; then
    RESTORED_DEVICES=$("${PSQL_TMP[@]}" -c "SELECT count(*) FROM devices;" 2>&1 || echo "0")
    diff_abs=$(( SRC_DEVICES - RESTORED_DEVICES ))
    diff_abs=${diff_abs#-}
    log "  COMPARE devices: source=${SRC_DEVICES} restored=${RESTORED_DEVICES} diff=${diff_abs}"
    # 允许差异 ≤ 源数量 1%（向上取整 1）
    threshold=$(( SRC_DEVICES / 100 + 1 ))
    if (( diff_abs > threshold )); then
        log "  CHECK devices_consistency: FAIL (diff ${diff_abs} > threshold ${threshold})"
        CHECK_FAILED=1
    else
        log "  CHECK devices_consistency: OK (diff ${diff_abs} ≤ threshold ${threshold})"
    fi
fi

# TimescaleDB 扩展：本地 dev 可能没装，仅在源库有的情况下要求
SRC_HAS_TIMESCALE=$("${PSQL_BASE[@]}" --dbname="${SOURCE_DB}" \
    -c "SELECT count(*) FROM pg_extension WHERE extname='timescaledb';" 2>/dev/null || echo "0")
if [[ "${SRC_HAS_TIMESCALE}" == "1" ]]; then
    verify "timescaledb_extension_alive" \
        "SELECT count(*) FROM pg_extension WHERE extname='timescaledb';" || CHECK_FAILED=1
else
    log "  SKIP timescaledb 检查（源库未启用 TimescaleDB 扩展）"
fi

if (( CHECK_FAILED != 0 )); then
    log "[FATAL] 校验失败，恢复演练判定为 FAIL"
    cat <<EOF
{
  "status": "fail",
  "rto_seconds": ${RTO_SECONDS},
  "tmp_db": "${TMP_DB}",
  "dump_file": "${DUMP_FILE}",
  "reason": "verification_failed"
}
EOF
    exit 3
fi

# ---------- step 4: 输出 RTO 报告 ----------
log "step 4: 演练通过"
cat <<EOF
{
  "status": "ok",
  "rto_seconds": ${RTO_SECONDS},
  "tmp_db": "${TMP_DB}",
  "dump_file": "${DUMP_FILE}",
  "source_db": "${SOURCE_DB}",
  "checks_passed": "tables_exist, devices_nonempty, users_nonempty, alarm_libraries_nonempty, devices_consistency"
}
EOF

# trap cleanup 会收尾销毁临时库
exit 0

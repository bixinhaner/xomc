#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLANNER="$SCRIPT_DIR/plan-resources.sh"
. "$SCRIPT_DIR/storage-paths-lib.sh"

PASS=0
FAIL=0
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
check_eq() {
  local name="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then ok; else bad "$name: got=[$got] want=[$want]"; fi
}

MOUNTS='107374182400|/
536870912000|/data-large
429496729600|/nvme-fast'

run_planner() {
  env \
    OMC_PROBE_CPU=32 \
    OMC_PROBE_MEM_TOTAL_MIB=65536 \
    OMC_PROBE_MEM_AVAIL_MIB=61440 \
    OMC_PROBE_LOAD15=0 \
    OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
    OMC_STORAGE_ENV_FILE="$1" \
    bash "$PLANNER" --assume-dedicated --skip-monitoring -o "$2" "${@:3}"
}

echo "── 最大可用存储写入 .env ──"
ENV_FILE="$TMP/generated.env"
printf 'OMC_PUBLIC_HOST=10.0.0.1\n' > "$ENV_FILE"
if run_planner "$ENV_FILE" "$TMP/resources.env" > "$TMP/output" 2>&1; then
  check_eq "PostgreSQL 默认路径" "$(storage_env_get "$ENV_FILE" POSTGRES_DATA_PATH)" "/data-large/omc-data/postgres"
  check_eq "TimescaleDB 默认路径" "$(storage_env_get "$ENV_FILE" TSDB_DATA_PATH)" "/data-large/omc-data/timescaledb"
  check_eq "Redis 默认路径" "$(storage_env_get "$ENV_FILE" REDIS_DATA_PATH)" "/data-large/omc-data/redis"
  check_eq "NATS 默认路径" "$(storage_env_get "$ENV_FILE" NATS_DATA_PATH)" "/data-large/omc-data/nats"
  check_eq "MinIO 默认路径" "$(storage_env_get "$ENV_FILE" MINIO_DATA_PATH)" "/data-large/omc-data/minio"
  if grep -q '^NATS_MAX_MEMORY_STORE=[0-9][0-9]*$' "$TMP/resources.env"; then
    ok
  else
    bad "resources.env 应输出 NATS_MAX_MEMORY_STORE"
  fi
else
  bad "planner 应成功运行"
fi

echo "── 不覆盖人工路径 ──"
CUSTOM_ENV="$TMP/custom.env"
printf 'MINIO_DATA_PATH=/manual/minio\n' > "$CUSTOM_ENV"
run_planner "$CUSTOM_ENV" "$TMP/custom-resources.env" > "$TMP/custom-output" 2>&1 || bad "自定义路径规划应成功"
check_eq "保留人工 MinIO 路径" "$(storage_env_get "$CUSTOM_ENV" MINIO_DATA_PATH)" "/manual/minio"

echo "── dry-run 不修改并提示人工检查 ──"
DRY_ENV="$TMP/dry.env"
printf 'OMC_PUBLIC_HOST=10.0.0.2\n' > "$DRY_ENV"
BEFORE="$(cat "$DRY_ENV")"
run_planner "$DRY_ENV" "$TMP/dry-resources.env" --dry-run > "$TMP/dry-output" 2>&1 || bad "dry-run 应成功"
check_eq "dry-run 不修改 env" "$(cat "$DRY_ENV")" "$BEFORE"
if grep -q '人工.*\.env' "$TMP/dry-output" && grep -q '不会.*迁移' "$TMP/dry-output"; then
  ok
else
  bad "dry-run 输出应提示人工修改 .env 且不会迁移数据"
fi

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]

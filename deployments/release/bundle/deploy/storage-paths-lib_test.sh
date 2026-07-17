#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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

echo "── 最大可用挂载点选择 ──"
MOUNTS='107374182400|/
536870912000|/data
429496729600|/nvme'
check_eq "选择可用空间最大挂载点" "$(printf '%s\n' "$MOUNTS" | storage_select_largest_mount)" "/data"

echo "── 推荐路径只补空值 ──"
ENV_FILE="$TMP/.env"
printf '%s\n' \
  'POSTGRES_DATA_PATH=/custom/pg' \
  'TSDB_DATA_PATH=' \
  'OMC_PUBLIC_HOST=10.0.0.1' > "$ENV_FILE"
storage_apply_recommended_paths "$ENV_FILE" "/data"
check_eq "保留人工 PostgreSQL 路径" "$(storage_env_get "$ENV_FILE" POSTGRES_DATA_PATH)" "/custom/pg"
check_eq "补 TimescaleDB 路径" "$(storage_env_get "$ENV_FILE" TSDB_DATA_PATH)" "/data/omc-data/timescaledb"
check_eq "补 Redis 路径" "$(storage_env_get "$ENV_FILE" REDIS_DATA_PATH)" "/data/omc-data/redis"
check_eq "补 NATS 路径" "$(storage_env_get "$ENV_FILE" NATS_DATA_PATH)" "/data/omc-data/nats"
check_eq "补 MinIO 路径" "$(storage_env_get "$ENV_FILE" MINIO_DATA_PATH)" "/data/omc-data/minio"
check_eq "保留无关配置" "$(storage_env_get "$ENV_FILE" OMC_PUBLIC_HOST)" "10.0.0.1"

echo "── 拒绝空路径和相对路径 ──"
BAD_ENV="$TMP/bad.env"
printf '%s\n' \
  'POSTGRES_DATA_PATH=/data/pg' \
  'TSDB_DATA_PATH=relative/tsdb' \
  'REDIS_DATA_PATH=/data/redis' \
  'NATS_DATA_PATH=/data/nats' \
  'MINIO_DATA_PATH=/data/minio' > "$BAD_ENV"
if storage_validate_env_paths "$BAD_ENV" >/dev/null 2>&1; then
  bad "相对路径应校验失败"
else
  ok
fi

EMPTY_ENV="$TMP/empty.env"
cp "$BAD_ENV" "$EMPTY_ENV"
sed 's#TSDB_DATA_PATH=relative/tsdb#TSDB_DATA_PATH=#' "$EMPTY_ENV" > "$EMPTY_ENV.next"
mv "$EMPTY_ENV.next" "$EMPTY_ENV"
if storage_validate_env_paths "$EMPTY_ENV" >/dev/null 2>&1; then
  bad "空路径应校验失败"
else
  ok
fi

echo "── 创建五个可写目录 ──"
PREPARE_ENV="$TMP/prepare.env"
storage_apply_recommended_paths "$PREPARE_ENV" "$TMP/disk"
if storage_prepare_env_paths "$PREPARE_ENV"; then
  for key in $STORAGE_PATH_KEYS; do
    path="$(storage_env_get "$PREPARE_ENV" "$key")"
    if [ -d "$path" ] && [ -w "$path" ]; then ok; else bad "$key 目录未创建或不可写"; fi
  done
else
  bad "合法绝对路径应能准备目录"
fi

echo "── 生命周期入口只准备已配置路径 ──"
PARTIAL_ENV="$TMP/partial.env"
printf 'POSTGRES_DATA_PATH=%s\nTSDB_DATA_PATH=\n' "$TMP/partial/postgres" > "$PARTIAL_ENV"
if storage_prepare_configured_env_paths "$PARTIAL_ENV"; then
  [ -d "$TMP/partial/postgres" ] && ok || bad "已配置路径应创建"
else
  bad "部分组件配置路径应被允许"
fi
printf 'POSTGRES_DATA_PATH=relative/postgres\n' > "$PARTIAL_ENV"
if storage_prepare_configured_env_paths "$PARTIAL_ENV" >/dev/null 2>&1; then
  bad "已配置的相对路径应失败"
else
  ok
fi

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]

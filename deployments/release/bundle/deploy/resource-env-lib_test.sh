#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB="$SCRIPT_DIR/resource-env-lib.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }

# 这份夹具是手工核验过的完整资源规划；任意契约遗漏或放宽约束都会令对应测试失效。
write_complete_env() {
  cat >"$1" <<'EOF'
APP_CPUS=2
APP_MEM=2048m
APP_GOMEMLIMIT=1800MiB
APP_GOMAXPROCS=2
ACS_CPUS=2
ACS_MEM=2048m
ACS_GOMEMLIMIT=1800MiB
ACS_GOMAXPROCS=2
WORKER_CPUS=4
WORKER_MEM=4096m
WORKER_GOMEMLIMIT=3600MiB
WORKER_GOMAXPROCS=4
POSTGRES_CPUS=4
POSTGRES_MEM=4096m
PG_SHARED_BUFFERS=1024MB
PG_EFFECTIVE_CACHE_SIZE=3072MB
PG_MAX_CONNECTIONS=180
PG_WORK_MEM=16MB
PG_MAINTENANCE_WORK_MEM=256MB
PG_MAX_WAL_SIZE=2048MB
TSDB_CPUS=4
TSDB_MEM=4096m
TSDB_SHARED_BUFFERS=1024MB
TSDB_EFFECTIVE_CACHE_SIZE=3072MB
TSDB_MAX_CONNECTIONS=180
TSDB_WORK_MEM=16MB
TSDB_MAINTENANCE_WORK_MEM=256MB
TSDB_MAX_WAL_SIZE=2048MB
REDIS_CORE_CPUS=2
REDIS_CORE_MEM=4g
REDIS_CORE_MAXMEMORY=3gb
REDIS_PM_CPUS=2
REDIS_PM_MEM=8g
REDIS_PM_MAXMEMORY=6gb
NATS_CPUS=1
NATS_MEM=512m
NATS_MAX_MEMORY_STORE=134217728
MINIO_CPUS=1
MINIO_MEM=1024m
WEB_CPUS=1
WEB_MEM=512m
OMC_RESOURCE_SCHEMA_VERSION=3
OMC_RESOURCE_PLAN_HOST_CPU=32
OMC_RESOURCE_PLAN_HOST_MEM_MIB=32768
EOF
}

set_key() {
  local file="$1" key="$2" value="$3"
  awk -F= -v key="$key" -v value="$value" 'BEGIN { OFS="=" } $1 == key { $2=value } { print }' "$file" >"$file.next" && mv "$file.next" "$file"
}

expect_invalid() {
  local name="$1" file="$2" expected="$3"
  if (resource_env_validate "$file") >"$TMP/$name.out" 2>&1; then
    bad "$name: 应被拒绝"
  elif grep -Fq -- "$expected" "$TMP/$name.out"; then
    ok
  else
    bad "$name: 错误应包含 [$expected]，实际：$(cat "$TMP/$name.out")"
  fi
}

. "$LIB"

echo "── 完整资源契约 ──"
write_complete_env "$TMP/complete.env"
if resource_env_validate "$TMP/complete.env"; then ok; else bad "完整资源规划应通过验证"; fi

required="$(resource_env_required_keys)"
for key in \
  APP_CPUS APP_MEM APP_GOMEMLIMIT APP_GOMAXPROCS \
  ACS_CPUS ACS_MEM ACS_GOMEMLIMIT ACS_GOMAXPROCS \
  WORKER_CPUS WORKER_MEM WORKER_GOMEMLIMIT WORKER_GOMAXPROCS \
  POSTGRES_CPUS POSTGRES_MEM PG_SHARED_BUFFERS PG_EFFECTIVE_CACHE_SIZE \
  PG_MAX_CONNECTIONS PG_WORK_MEM PG_MAINTENANCE_WORK_MEM PG_MAX_WAL_SIZE \
  TSDB_CPUS TSDB_MEM TSDB_SHARED_BUFFERS TSDB_EFFECTIVE_CACHE_SIZE \
  TSDB_MAX_CONNECTIONS TSDB_WORK_MEM TSDB_MAINTENANCE_WORK_MEM TSDB_MAX_WAL_SIZE \
  REDIS_CORE_CPUS REDIS_CORE_MEM REDIS_CORE_MAXMEMORY \
  REDIS_PM_CPUS REDIS_PM_MEM REDIS_PM_MAXMEMORY \
  NATS_CPUS NATS_MEM NATS_MAX_MEMORY_STORE \
  MINIO_CPUS MINIO_MEM WEB_CPUS WEB_MEM \
  OMC_RESOURCE_SCHEMA_VERSION OMC_RESOURCE_PLAN_HOST_CPU OMC_RESOURCE_PLAN_HOST_MEM_MIB; do
  if printf '%s\n' "$required" | grep -Fxq "$key"; then ok; else bad "required keys 缺少 $key"; fi
done

echo "── 残缺与单位非法必须失败 ──"
cat >"$TMP/partial.env" <<'EOF'
REDIS_CORE_CPUS=2
REDIS_CORE_MEM=4g
REDIS_CORE_MAXMEMORY=3gb
EOF
expect_invalid partial "$TMP/partial.env" APP_CPUS

cp "$TMP/complete.env" "$TMP/duplicate-contract-key.env"
printf 'REDIS_PM_MAXMEMORY=7g\n' >>"$TMP/duplicate-contract-key.env"
expect_invalid duplicate-contract-key "$TMP/duplicate-contract-key.env" REDIS_PM_MAXMEMORY

cp "$TMP/complete.env" "$TMP/bad-cpu.env"
set_key "$TMP/bad-cpu.env" APP_CPUS 0
expect_invalid bad-cpu "$TMP/bad-cpu.env" APP_CPUS

cp "$TMP/complete.env" "$TMP/bad-memory.env"
set_key "$TMP/bad-memory.env" APP_MEM 2bananas
expect_invalid bad-memory "$TMP/bad-memory.env" APP_MEM

cp "$TMP/complete.env" "$TMP/bad-nats-memory.env"
set_key "$TMP/bad-nats-memory.env" NATS_MAX_MEMORY_STORE -1
expect_invalid bad-nats-memory "$TMP/bad-nats-memory.env" NATS_MAX_MEMORY_STORE

echo "── 跨字段安全余量 ──"
cp "$TMP/complete.env" "$TMP/bad-worker-gomem.env"
set_key "$TMP/bad-worker-gomem.env" WORKER_GOMEMLIMIT 4096MiB
expect_invalid bad-worker-gomem "$TMP/bad-worker-gomem.env" WORKER_GOMEMLIMIT

cp "$TMP/complete.env" "$TMP/bad-acs-gomem.env"
set_key "$TMP/bad-acs-gomem.env" ACS_GOMEMLIMIT 2048MiB
expect_invalid bad-acs-gomem "$TMP/bad-acs-gomem.env" ACS_GOMEMLIMIT

cp "$TMP/complete.env" "$TMP/bad-app-gomem.env"
set_key "$TMP/bad-app-gomem.env" APP_GOMEMLIMIT 2048MiB
expect_invalid bad-app-gomem "$TMP/bad-app-gomem.env" APP_GOMEMLIMIT

cp "$TMP/complete.env" "$TMP/bad-redis-core-headroom.env"
set_key "$TMP/bad-redis-core-headroom.env" REDIS_CORE_MAXMEMORY 3073MiB
expect_invalid bad-redis-core-headroom "$TMP/bad-redis-core-headroom.env" REDIS_CORE_MAXMEMORY

cp "$TMP/complete.env" "$TMP/bad-redis-pm-headroom.env"
set_key "$TMP/bad-redis-pm-headroom.env" REDIS_PM_MAXMEMORY 6145MiB
expect_invalid bad-redis-pm-headroom "$TMP/bad-redis-pm-headroom.env" REDIS_PM_MAXMEMORY

cp "$TMP/complete.env" "$TMP/legacy-schema.env"
set_key "$TMP/legacy-schema.env" OMC_RESOURCE_SCHEMA_VERSION 2
expect_invalid legacy-schema "$TMP/legacy-schema.env" "OMC_RESOURCE_SCHEMA_VERSION"

cp "$TMP/complete.env" "$TMP/bad-pg-connections.env"
set_key "$TMP/bad-pg-connections.env" PG_MAX_CONNECTIONS 179
expect_invalid bad-pg-connections "$TMP/bad-pg-connections.env" PG_MAX_CONNECTIONS

cp "$TMP/complete.env" "$TMP/bad-tsdb-connections.env"
set_key "$TMP/bad-tsdb-connections.env" TSDB_MAX_CONNECTIONS 179
expect_invalid bad-tsdb-connections "$TMP/bad-tsdb-connections.env" TSDB_MAX_CONNECTIONS

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]

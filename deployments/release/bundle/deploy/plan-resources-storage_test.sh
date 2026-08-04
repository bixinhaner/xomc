#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLANNER="$SCRIPT_DIR/plan-resources.sh"
. "$SCRIPT_DIR/storage-paths-lib.sh"
. "$SCRIPT_DIR/resource-env-lib.sh"

PASS=0
FAIL=0
export OMC_LANG=en
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
check_eq() {
  local name="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then ok; else bad "$name: got=[$got] want=[$want]"; fi
}
file_inode() {
  stat -c '%i' "$1" 2>/dev/null || stat -f '%i' "$1"
}
file_mtime() {
  stat -c '%Y' "$1" 2>/dev/null || stat -f '%m' "$1"
}

MOUNTS='107374182400|/
536870912000|/data-large
429496729600|/nvme-fast'

run_planner() {
  env \
    OMC_LANG=en \
    OMC_PROBE_CPU=32 \
    OMC_PROBE_MEM_TOTAL_MIB=32768 \
    OMC_PROBE_MEM_AVAIL_MIB=32768 \
    OMC_PROBE_LOAD15=0 \
    OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
    OMC_STORAGE_ENV_FILE="$1" \
    bash "$PLANNER" --assume-dedicated --skip-monitoring -o "$2" "${@:3}"
}

run_large_planner() {
  env \
    OMC_LANG=en \
    OMC_PROBE_CPU=64 \
    OMC_PROBE_MEM_TOTAL_MIB=131072 \
    OMC_PROBE_MEM_AVAIL_MIB=131072 \
    OMC_PROBE_LOAD15=0 \
    OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
    OMC_STORAGE_ENV_FILE="$1" \
    bash "$PLANNER" --assume-dedicated --skip-monitoring -o "$2"
}

echo "── 最大可用存储写入 .env ──"
ENV_FILE="$TMP/generated.env"
printf 'OMC_PUBLIC_HOST=10.0.0.1\n' > "$ENV_FILE"
printf 'LAST_GOOD=must-be-atomically-replaced\n' >"$TMP/resources.env"
BEFORE_INODE="$(file_inode "$TMP/resources.env")"
if run_planner "$ENV_FILE" "$TMP/resources.env" > "$TMP/output" 2>&1; then
  if resource_env_validate "$TMP/resources.env"; then
    ok
  else
    bad "planner 输出必须满足完整资源契约"
  fi
  check_eq "资源契约版本" "$(storage_env_get "$TMP/resources.env" OMC_RESOURCE_SCHEMA_VERSION)" "3"
  check_eq "规划主机 CPU 元数据" "$(storage_env_get "$TMP/resources.env" OMC_RESOURCE_PLAN_HOST_CPU)" "32"
  check_eq "规划主机内存元数据" "$(storage_env_get "$TMP/resources.env" OMC_RESOURCE_PLAN_HOST_MEM_MIB)" "32768"
  check_eq "PostgreSQL 默认路径" "$(storage_env_get "$ENV_FILE" POSTGRES_DATA_PATH)" "/data-large/omc-data/postgres"
  check_eq "TimescaleDB 默认路径" "$(storage_env_get "$ENV_FILE" TSDB_DATA_PATH)" "/data-large/omc-data/timescaledb"
  check_eq "Redis 默认路径" "$(storage_env_get "$ENV_FILE" REDIS_DATA_PATH)" "/data-large/omc-data/redis"
  check_eq "PM Redis 默认路径" "$(storage_env_get "$ENV_FILE" REDIS_PM_DATA_PATH)" "/data-large/omc-data/redis-pm"
  check_eq "NATS 默认路径" "$(storage_env_get "$ENV_FILE" NATS_DATA_PATH)" "/data-large/omc-data/nats"
  check_eq "MinIO 默认路径" "$(storage_env_get "$ENV_FILE" MINIO_DATA_PATH)" "/data-large/omc-data/minio"
  if grep -q '^NATS_MAX_MEMORY_STORE=[0-9][0-9]*$' "$TMP/resources.env"; then
    ok
  else
    bad "resources.env 应输出 NATS_MAX_MEMORY_STORE"
  fi
  check_eq "32核 medium 主库 CPU" "$(storage_env_get "$TMP/resources.env" POSTGRES_CPUS)" "10"
  check_eq "32核 medium TimescaleDB CPU" "$(storage_env_get "$TMP/resources.env" TSDB_CPUS)" "16"
  check_eq "32核 medium worker CPU" "$(storage_env_get "$TMP/resources.env" WORKER_CPUS)" "8"
  check_eq "32GiB medium MinIO 内存余量" "$(storage_env_get "$TMP/resources.env" MINIO_MEM)" "6144m"
  check_eq "web 最低内存避免 nginx OOM" "$(storage_env_get "$TMP/resources.env" WEB_MEM)" "512m"
  if [ "$(file_inode "$TMP/resources.env")" != "$BEFORE_INODE" ]; then
    ok
  else
    bad "planner 成功时应以同目录临时文件原子替换正式 resources.env"
  fi
else
  bad "planner 应成功运行"
fi

echo "── 低配 medium 档按可用预算降级 MinIO，不阻断安装 ──"
FLEX_ENV="$TMP/flexible-medium.env"
printf 'OMC_PUBLIC_HOST=10.0.0.8\n' > "$FLEX_ENV"
if env \
  OMC_PROBE_CPU=32 \
  OMC_PROBE_MEM_TOTAL_MIB=31763 \
  OMC_PROBE_MEM_AVAIL_MIB=29000 \
  OMC_PROBE_LOAD15=0 \
  OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
  OMC_STORAGE_ENV_FILE="$FLEX_ENV" \
  bash "$PLANNER" --assume-dedicated --output "$TMP/flexible-medium-resources.env" \
    >"$TMP/flexible-medium-output" 2>&1; then
  if resource_env_validate "$TMP/flexible-medium-resources.env"; then
    ok
  else
    bad "低配 medium 档资源计划必须满足完整资源契约"
  fi
  flexible_minio_mib="$(resource_env_memory_mib "$(resource_env_get "$TMP/flexible-medium-resources.env" MINIO_MEM)" | awk '{printf "%d", $1}')"
  if [ "$flexible_minio_mib" -ge 512 ] && [ "$flexible_minio_mib" -lt 6144 ] &&
    grep -q 'does not have enough idle budget' "$TMP/flexible-medium-output"; then
    ok
  else
    bad "低配 medium 档应在不削减 floor 的前提下灵活降低 MinIO 内存"
  fi
else
  bad "低配 medium 档不应因 MinIO 6GiB 目标不可达而失败"
fi

echo "── 大机型按主库1/3、时序库1/2、worker1/4分配 CPU ──"
LARGE_ENV="$TMP/large.env"
printf 'OMC_PUBLIC_HOST=10.0.0.3\n' > "$LARGE_ENV"
if run_large_planner "$LARGE_ENV" "$TMP/large-resources.env" > "$TMP/large-output" 2>&1; then
  check_eq "64核 large 主库 CPU" "$(storage_env_get "$TMP/large-resources.env" POSTGRES_CPUS)" "21"
  check_eq "64核 large TimescaleDB CPU" "$(storage_env_get "$TMP/large-resources.env" TSDB_CPUS)" "32"
  check_eq "64核 large worker CPU" "$(storage_env_get "$TMP/large-resources.env" WORKER_CPUS)" "16"
else
  bad "large planner 应成功运行"
fi

echo "── 双 ACS 余量分配不得超出自身预算 ──"
for mem_mib in 40000 45000 50000; do
  sized_env="$TMP/ha-${mem_mib}.env"
  sized_out="$TMP/ha-${mem_mib}-resources.env"
  printf 'OMC_PUBLIC_HOST=10.0.0.5\n' >"$sized_env"
  if env \
    OMC_PROBE_CPU=32 \
    OMC_PROBE_MEM_TOTAL_MIB="$mem_mib" \
    OMC_PROBE_MEM_AVAIL_MIB="$mem_mib" \
    OMC_PROBE_LOAD15=0 \
    OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
    OMC_STORAGE_ENV_FILE="$sized_env" \
    bash "$PLANNER" --assume-dedicated --skip-monitoring \
      -o "$sized_out" >"$TMP/ha-${mem_mib}-output" 2>&1; then
    ok
  else
    bad "${mem_mib}MiB 独占主机的双 ACS 规划不应因内部重复分配超预算"
  fi
done

echo "── 实际内存驱动动态资源分配 ──"
DYNAMIC_ENV="$TMP/dynamic.env"
printf 'OMC_PUBLIC_HOST=10.0.0.6\n' > "$DYNAMIC_ENV"
if env \
  OMC_PROBE_CPU=32 \
  OMC_PROBE_MEM_TOTAL_MIB=31763 \
  OMC_PROBE_MEM_AVAIL_MIB=23756 \
  OMC_PROBE_LOAD15=0 \
  OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
  OMC_STORAGE_ENV_FILE="$DYNAMIC_ENV" \
  bash "$PLANNER" --assume-dedicated --floor-tolerance-pct 50 --output "$TMP/dynamic-resources.env" \
    >"$TMP/dynamic-output" 2>&1; then
  if resource_env_validate "$TMP/dynamic-resources.env"; then
    ok
  else
    bad "实际内存驱动的资源计划必须满足完整资源契约"
  fi
  dynamic_acs_mib="$(resource_env_memory_mib "$(resource_env_get "$TMP/dynamic-resources.env" ACS_MEM)" | awk '{printf "%d", $1}')"
  dynamic_pg_mib="$(resource_env_memory_mib "$(resource_env_get "$TMP/dynamic-resources.env" POSTGRES_MEM)" | awk '{printf "%d", $1}')"
  dynamic_redis_mib="$(resource_env_memory_mib "$(resource_env_get "$TMP/dynamic-resources.env" REDIS_PM_MEM)" | awk '{printf "%d", $1}')"
  if [ "$dynamic_acs_mib" -lt 4096 ] && [ "$dynamic_pg_mib" -lt 7168 ] && [ "$dynamic_redis_mib" -lt 8192 ]; then
    ok
  else
    bad "31 GiB 主机的 ACS/PG/Redis 资源未按实际预算缩放"
  fi
else
  bad "31 GiB 主机应能生成按实际预算缩放的资源计划"
fi

echo "── 过低内存仍必须拒绝部署 ──"
LOW_ENV="$TMP/low-memory.env"
printf 'OMC_PUBLIC_HOST=10.0.0.7\n' > "$LOW_ENV"
if env \
  OMC_PROBE_CPU=16 \
  OMC_PROBE_MEM_TOTAL_MIB=16384 \
  OMC_PROBE_MEM_AVAIL_MIB=16384 \
  OMC_PROBE_LOAD15=0 \
  OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
  OMC_STORAGE_ENV_FILE="$LOW_ENV" \
  bash "$PLANNER" --assume-dedicated --floor-tolerance-pct 50 --output "$TMP/low-resources.env" \
    >"$TMP/low-output" 2>&1; then
  bad "过低内存主机不得生成资源计划"
else
  ok
fi
if [ -e "$TMP/low-resources.env" ]; then
  bad "过低内存主机不得写入 resources.env"
else
  ok
fi
if grep -q -- '--floor-tolerance-pct applies only to later component-floor gaps' "$TMP/low-output" &&
  grep -q -- 'plan-resources.sh --skip-monitoring' "$TMP/low-output"; then
  ok
else
  bad "过低内存提示应说明容忍度边界并给出跳过监控的可执行命令"
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
if grep -q 'Check and adjust.*\.env' "$TMP/dry-output" && grep -q 'does not migrate' "$TMP/dry-output"; then
  ok
else
  bad "dry-run 输出应提示人工修改 .env 且不会迁移数据"
fi

echo "── 生成/验证失败保留 last-good ──"
FAIL_PLANNER_DIR="$TMP/failing-planner"
mkdir -p "$FAIL_PLANNER_DIR"
cp "$PLANNER" "$SCRIPT_DIR/storage-paths-lib.sh" "$SCRIPT_DIR/resource-env-lib.sh" "$FAIL_PLANNER_DIR/"
cat >>"$FAIL_PLANNER_DIR/resource-env-lib.sh" <<'EOF'
resource_env_validate() {
  echo "[resource-env] injected validation failure" >&2
  return 73
}
EOF
FAIL_ENV="$TMP/failing.env"
printf 'OMC_PUBLIC_HOST=10.0.0.4\n' >"$FAIL_ENV"
LAST_GOOD="$TMP/last-good-resources.env"
printf 'LAST_GOOD=preserve-me\n' >"$LAST_GOOD"
BEFORE_SUM="$(cksum <"$LAST_GOOD")"
BEFORE_MTIME="$(file_mtime "$LAST_GOOD")"
if env \
  OMC_PROBE_CPU=32 \
  OMC_PROBE_MEM_TOTAL_MIB=32768 \
  OMC_PROBE_MEM_AVAIL_MIB=32768 \
  OMC_PROBE_LOAD15=0 \
  OMC_PROBE_STORAGE_MOUNTS="$MOUNTS" \
  OMC_STORAGE_ENV_FILE="$FAIL_ENV" \
  bash "$FAIL_PLANNER_DIR/plan-resources.sh" --assume-dedicated --skip-monitoring \
    -o "$LAST_GOOD" >"$TMP/failing-output" 2>&1; then
  bad "注入资源契约验证失败时 planner 必须返回失败"
else
  ok
fi
if [ -f "$LAST_GOOD" ]; then
  check_eq "验证失败保留 last-good 内容" "$(cksum <"$LAST_GOOD")" "$BEFORE_SUM"
  check_eq "验证失败保留 last-good mtime" "$(file_mtime "$LAST_GOOD")" "$BEFORE_MTIME"
else
  bad "验证失败不得删除 last-good resources.env"
fi
if find "$TMP" -maxdepth 1 -name 'last-good-resources.env.tmp.*' | grep -q .; then
  bad "验证失败后不得残留同目录临时资源文件"
else
  ok
fi

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]

#!/usr/bin/env bash
# 资源规划契约：只解析 KEY=VALUE，不执行 operator 可编辑的 resources.env。

resource_env_required_keys() {
  cat <<'EOF'
APP_CPUS
APP_MEM
APP_GOMEMLIMIT
APP_GOMAXPROCS
ACS_CPUS
ACS_MEM
ACS_GOMEMLIMIT
ACS_GOMAXPROCS
WORKER_CPUS
WORKER_MEM
WORKER_GOMEMLIMIT
WORKER_GOMAXPROCS
POSTGRES_CPUS
POSTGRES_MEM
PG_SHARED_BUFFERS
PG_EFFECTIVE_CACHE_SIZE
PG_MAX_CONNECTIONS
PG_WORK_MEM
PG_MAINTENANCE_WORK_MEM
PG_MAX_WAL_SIZE
TSDB_CPUS
TSDB_MEM
TSDB_SHARED_BUFFERS
TSDB_EFFECTIVE_CACHE_SIZE
TSDB_MAX_CONNECTIONS
TSDB_WORK_MEM
TSDB_MAINTENANCE_WORK_MEM
TSDB_MAX_WAL_SIZE
REDIS_CORE_CPUS
REDIS_CORE_MEM
REDIS_CORE_MAXMEMORY
REDIS_PM_CPUS
REDIS_PM_MEM
REDIS_PM_MAXMEMORY
NATS_CPUS
NATS_MEM
NATS_MAX_MEMORY_STORE
MINIO_CPUS
MINIO_MEM
WEB_CPUS
WEB_MEM
OMC_RESOURCE_SCHEMA_VERSION
OMC_RESOURCE_PLAN_HOST_CPU
OMC_RESOURCE_PLAN_HOST_MEM_MIB
EOF
}

resource_env_get() {
  local file="$1" key="$2"
  awk -v key="$key" '
    /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
    {
      pos=index($0, "=")
      if (pos == 0) next
      name=substr($0, 1, pos - 1)
      value=substr($0, pos + 1)
      sub(/\r$/, "", value)
      if (name == key) { print value; exit }
    }
  ' "$file"
}

resource_env_key_count() {
  local file="$1" key="$2"
  awk -v key="$key" '
    /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
    {
      pos=index($0, "=")
      if (pos > 0 && substr($0, 1, pos - 1) == key) count++
    }
    END { print count + 0 }
  ' "$file"
}

resource_env_positive_integer() { [[ "$1" =~ ^[1-9][0-9]*$ ]]; }
resource_env_positive_cpu() {
  [[ "$1" =~ ^[0-9]+([.][0-9]+)?$ ]] && awk -v value="$1" 'BEGIN { exit !(value > 0) }'
}
resource_env_memory_mib() {
  local value="$1" number unit
  if [[ "$value" =~ ^([0-9]+([.][0-9]+)?)([mMgG]([iI])?([bB])?)$ ]]; then
    number="${BASH_REMATCH[1]}"
    unit="${BASH_REMATCH[3],,}"
    awk -v number="$number" -v unit="$unit" 'BEGIN {
      if (number <= 0) exit 1
      if (unit ~ /^g/) printf "%.6f", number * 1024
      else printf "%.6f", number
    }'
  else
    return 1
  fi
}
resource_env_memory_is_positive() { resource_env_memory_mib "$1" >/dev/null; }
resource_env_less_than() {
  local left="$1" right="$2"
  awk -v left="$left" -v right="$right" 'BEGIN { exit !(left < right) }'
}
resource_env_at_most() {
  local left="$1" right="$2"
  awk -v left="$left" -v right="$right" 'BEGIN { exit !(left <= right) }'
}

resource_env_error() {
  local cn="$1" en="${2:-$1}"
  if [ "${OMC_LANG:-en}" = en ]; then
    printf '[resource-env][Error] %s\n' "$en" >&2
  else
    printf '[resource-env][错误] %s\n' "$cn" >&2
  fi
}

resource_env_validate() {
  local file="$1" key value invalid=0
  local app_mem acs_mem worker_mem redis_core_mem redis_core_max redis_pm_mem redis_pm_max
  [ -f "$file" ] || { resource_env_error "文件不存在: $file" "File does not exist: $file"; return 1; }

  while IFS= read -r key; do
    value="$(resource_env_get "$file" "$key")"
    if [ -z "$value" ]; then
      resource_env_error "缺少必填键: $key" "Required key is missing: $key"
      invalid=1
    fi
    if [ "$(resource_env_key_count "$file" "$key")" -gt 1 ]; then
      resource_env_error "契约键不可重复赋值: $key" "Contract key is assigned more than once: $key"
      invalid=1
    fi
  done < <(resource_env_required_keys)
  [ "$invalid" -eq 0 ] || return 1

  for key in APP_CPUS ACS_CPUS WORKER_CPUS POSTGRES_CPUS TSDB_CPUS REDIS_CORE_CPUS REDIS_PM_CPUS NATS_CPUS MINIO_CPUS WEB_CPUS; do
    value="$(resource_env_get "$file" "$key")"
    if ! resource_env_positive_cpu "$value"; then
      resource_env_error "$key 必须是正数 CPU 值: $value" "$key must be a positive CPU value: $value"; invalid=1
    fi
  done
  for key in APP_MEM APP_GOMEMLIMIT ACS_MEM ACS_GOMEMLIMIT WORKER_MEM WORKER_GOMEMLIMIT POSTGRES_MEM PG_SHARED_BUFFERS PG_EFFECTIVE_CACHE_SIZE PG_WORK_MEM PG_MAINTENANCE_WORK_MEM PG_MAX_WAL_SIZE TSDB_MEM TSDB_SHARED_BUFFERS TSDB_EFFECTIVE_CACHE_SIZE TSDB_WORK_MEM TSDB_MAINTENANCE_WORK_MEM TSDB_MAX_WAL_SIZE REDIS_CORE_MEM REDIS_CORE_MAXMEMORY REDIS_PM_MEM REDIS_PM_MAXMEMORY NATS_MEM MINIO_MEM WEB_MEM; do
    value="$(resource_env_get "$file" "$key")"
    if ! resource_env_memory_is_positive "$value"; then
      resource_env_error "$key 必须是带可解析单位的正内存值: $value" "$key must be a positive memory value with a parseable unit: $value"; invalid=1
    fi
  done
  for key in APP_GOMAXPROCS ACS_GOMAXPROCS WORKER_GOMAXPROCS PG_MAX_CONNECTIONS TSDB_MAX_CONNECTIONS NATS_MAX_MEMORY_STORE OMC_RESOURCE_PLAN_HOST_CPU OMC_RESOURCE_PLAN_HOST_MEM_MIB; do
    value="$(resource_env_get "$file" "$key")"
    if ! resource_env_positive_integer "$value"; then
      resource_env_error "$key 必须是正整数: $value" "$key must be a positive integer: $value"; invalid=1
    fi
  done
  value="$(resource_env_get "$file" OMC_RESOURCE_SCHEMA_VERSION)"
  if [ "$value" != "3" ]; then
    resource_env_error "OMC_RESOURCE_SCHEMA_VERSION 必须为 3: ${value}；请重新运行 plan-resources.sh" "OMC_RESOURCE_SCHEMA_VERSION must be 3: ${value}; run plan-resources.sh again"; invalid=1
  fi

  app_mem="$(resource_env_memory_mib "$(resource_env_get "$file" APP_MEM)")"
  acs_mem="$(resource_env_memory_mib "$(resource_env_get "$file" ACS_MEM)")"
  worker_mem="$(resource_env_memory_mib "$(resource_env_get "$file" WORKER_MEM)")"
  redis_core_mem="$(resource_env_memory_mib "$(resource_env_get "$file" REDIS_CORE_MEM)")"
  redis_core_max="$(resource_env_memory_mib "$(resource_env_get "$file" REDIS_CORE_MAXMEMORY)")"
  redis_pm_mem="$(resource_env_memory_mib "$(resource_env_get "$file" REDIS_PM_MEM)")"
  redis_pm_max="$(resource_env_memory_mib "$(resource_env_get "$file" REDIS_PM_MAXMEMORY)")"
  if ! resource_env_less_than "$(resource_env_memory_mib "$(resource_env_get "$file" APP_GOMEMLIMIT)")" "$app_mem"; then
    resource_env_error "APP_GOMEMLIMIT 必须 < APP_MEM" "APP_GOMEMLIMIT must be < APP_MEM"; invalid=1
  fi
  if ! resource_env_less_than "$(resource_env_memory_mib "$(resource_env_get "$file" ACS_GOMEMLIMIT)")" "$acs_mem"; then
    resource_env_error "ACS_GOMEMLIMIT 必须 < ACS_MEM" "ACS_GOMEMLIMIT must be < ACS_MEM"; invalid=1
  fi
  if ! resource_env_less_than "$(resource_env_memory_mib "$(resource_env_get "$file" WORKER_GOMEMLIMIT)")" "$worker_mem"; then
    resource_env_error "WORKER_GOMEMLIMIT 必须 < WORKER_MEM" "WORKER_GOMEMLIMIT must be < WORKER_MEM"; invalid=1
  fi
  if ! resource_env_at_most "$redis_core_max" "$(awk -v value="$redis_core_mem" 'BEGIN { printf "%.6f", value - 1024 }')"; then
    resource_env_error "REDIS_CORE_MAXMEMORY 必须 <= REDIS_CORE_MEM - 1GiB" "REDIS_CORE_MAXMEMORY must be <= REDIS_CORE_MEM - 1GiB"; invalid=1
  fi
  if ! resource_env_at_most "$redis_pm_max" "$(awk -v value="$redis_pm_mem" 'BEGIN { printf "%.6f", value - 2048 }')"; then
    resource_env_error "REDIS_PM_MAXMEMORY 必须 <= REDIS_PM_MEM - 2GiB" "REDIS_PM_MAXMEMORY must be <= REDIS_PM_MEM - 2GiB"; invalid=1
  fi
  if [ "$(resource_env_get "$file" PG_MAX_CONNECTIONS)" -lt 180 ]; then
    resource_env_error "PG_MAX_CONNECTIONS 必须 >= 180" "PG_MAX_CONNECTIONS must be >= 180"; invalid=1
  fi
  if [ "$(resource_env_get "$file" TSDB_MAX_CONNECTIONS)" -lt 180 ]; then
    resource_env_error "TSDB_MAX_CONNECTIONS 必须 >= 180" "TSDB_MAX_CONNECTIONS must be >= 180"; invalid=1
  fi
  [ "$invalid" -eq 0 ]
}

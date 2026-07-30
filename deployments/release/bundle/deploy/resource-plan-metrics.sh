#!/usr/bin/env bash
# 将完整 resources.env 渲染为 node-exporter textfile 指标。该文件不 source 操作员输入。

set -u

RESOURCE_PLAN_METRICS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$RESOURCE_PLAN_METRICS_DIR/resource-env-lib.sh"

resource_plan_metrics_render() { # <resources.env>
  local resource_env="$1" service prefix cpu mem_mib memory_bytes
  resource_env_validate "$resource_env" || return 1

  cat <<'EOF'
# HELP omc_resource_plan_cpu_cores Declared OMC resource plan CPU cores by compose service.
# TYPE omc_resource_plan_cpu_cores gauge
# HELP omc_resource_plan_memory_limit_bytes Declared OMC resource plan memory limit by compose service.
# TYPE omc_resource_plan_memory_limit_bytes gauge
EOF
  for service_prefix in \
    'app APP' 'acs ACS' 'worker WORKER' 'postgres POSTGRES' \
    'postgres-tsdb TSDB' 'redis REDIS' 'nats NATS' 'minio MINIO' 'web WEB'; do
    read -r service prefix <<<"$service_prefix"
    cpu="$(resource_env_get "$resource_env" "${prefix}_CPUS")"
    mem_mib="$(resource_env_memory_mib "$(resource_env_get "$resource_env" "${prefix}_MEM")")" || return 1
    memory_bytes="$(awk -v mib="$mem_mib" 'BEGIN { printf "%.0f", mib * 1024 * 1024 }')"
    printf 'omc_resource_plan_cpu_cores{service="%s"} %s\n' "$service" "$cpu"
    printf 'omc_resource_plan_memory_limit_bytes{service="%s"} %s\n' "$service" "$memory_bytes"
  done
}

resource_plan_metrics_write() { # <resources.env> [output.prom]
  local resource_env="$1" output="${2:-/opt/omc/run/monitoring/resource-plan.prom}" tmp
  mkdir -p "$(dirname "$output")" || return 1
  tmp="$(mktemp "${output}.tmp.XXXXXX")" || return 1
  if ! resource_plan_metrics_render "$resource_env" >"$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  # mktemp 默认是 owner-only；node-exporter 以非 root 用户读取 textfile 时会被拒绝。
  if ! chmod 0644 "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  mv "$tmp" "$output"
}

resource_plan_metrics_remove() { # [output.prom]
  rm -f "${1:-/opt/omc/run/monitoring/resource-plan.prom}"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  [ $# -ge 1 ] || { echo "usage: $0 <resources.env> [output.prom]" >&2; exit 2; }
  resource_plan_metrics_write "$@"
fi

#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/resource-plan-metrics.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cat >"$TMP/resources.env" <<'EOF'
APP_CPUS=2
APP_MEM=1536m
APP_GOMEMLIMIT=1200MiB
APP_GOMAXPROCS=2
ACS_CPUS=5
ACS_MEM=4g
ACS_GOMEMLIMIT=3600MiB
ACS_GOMAXPROCS=5
WORKER_CPUS=8
WORKER_MEM=2g
WORKER_GOMEMLIMIT=1800MiB
WORKER_GOMAXPROCS=8
POSTGRES_CPUS=10
POSTGRES_MEM=7g
PG_SHARED_BUFFERS=1792MB
PG_EFFECTIVE_CACHE_SIZE=5017MB
PG_MAX_CONNECTIONS=180
PG_WORK_MEM=8MB
PG_MAINTENANCE_WORK_MEM=358MB
PG_MAX_WAL_SIZE=4096MB
TSDB_CPUS=16
TSDB_MEM=7g
TSDB_SHARED_BUFFERS=1792MB
TSDB_EFFECTIVE_CACHE_SIZE=5017MB
TSDB_MAX_CONNECTIONS=180
TSDB_WORK_MEM=8MB
TSDB_MAINTENANCE_WORK_MEM=358MB
TSDB_MAX_WAL_SIZE=4096MB
REDIS_CPUS=2
REDIS_MEM=8g
REDIS_MAXMEMORY=7g
NATS_CPUS=1
NATS_MEM=1g
NATS_MAX_MEMORY_STORE=268435456
MINIO_CPUS=4
MINIO_MEM=4g
WEB_CPUS=1
WEB_MEM=512m
OMC_RESOURCE_SCHEMA_VERSION=2
OMC_RESOURCE_PLAN_HOST_CPU=32
OMC_RESOURCE_PLAN_HOST_MEM_MIB=32768
EOF

resource_plan_metrics_write "$TMP/resources.env" "$TMP/resource-plan.prom"
grep -Fx 'omc_resource_plan_cpu_cores{service="worker"} 8' "$TMP/resource-plan.prom"
grep -Fx 'omc_resource_plan_memory_limit_bytes{service="redis"} 8589934592' "$TMP/resource-plan.prom"
grep -Fx 'omc_resource_plan_memory_limit_bytes{service="app"} 1610612736' "$TMP/resource-plan.prom"
[ "$(grep -c '^omc_resource_plan_cpu_cores{service=' "$TMP/resource-plan.prom")" = 9 ]
[ "$(grep -c '^omc_resource_plan_memory_limit_bytes{service=' "$TMP/resource-plan.prom")" = 9 ]

printf 'resource plan metrics: PASS\n'

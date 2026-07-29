#!/usr/bin/env bash
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
RELEASE_DEPLOY="$REPO_ROOT/deployments/release/bundle/deploy"
RELEASE_COMPOSE="$RELEASE_DEPLOY/docker-compose.infra.yml"
RELEASE_APP_COMPOSE="$RELEASE_DEPLOY/docker-compose.app.yml"
RELEASE_MONITORING_COMPOSE="$RELEASE_DEPLOY/docker-compose.monitoring.yml"
RELEASE_HEALTHCHECK="$RELEASE_DEPLOY/healthcheck.sh"
MONITORING_PROFILE_LIB="$RELEASE_DEPLOY/monitoring-profile-lib.sh"
DEV_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.yml"
TEST_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.test.yml"
OTELCOL_CONFIG="$REPO_ROOT/deployments/monitoring/otelcol/config.yaml"
OMC_ALERTS="$REPO_ROOT/deployments/monitoring/alerts/omc-rules.yml"
HOST_ALERTS="$REPO_ROOT/deployments/monitoring/alerts/host-container-alerts.yml"
GRAFANA_DASHBOARD="$REPO_ROOT/deployments/monitoring/grafana-dashboard.json"
GRAFANA_OVERVIEW="$REPO_ROOT/deployments/monitoring/grafana/dashboards/omc-overview.json"
HOST_DASHBOARD="$REPO_ROOT/deployments/monitoring/grafana/dashboards/nginx-host-overview.json"
APP_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/app/etc/config.prod.yaml"
ACS_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/acs/etc/config.prod.yaml"
WORKER_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/worker/etc/config.prod.yaml"
INSTALL="$RELEASE_DEPLOY/install.sh"
SVC="$RELEASE_DEPLOY/svc.sh"
BUILD="$REPO_ROOT/deployments/release/build-release.sh"
DEV_PLANNER="$REPO_ROOT/deployments/docker/plan-resources.sh"
NGINX_DEFAULT="$REPO_ROOT/deployments/docker/default.conf"
NGINX_LOCAL="$REPO_ROOT/deployments/docker/default.local.conf"

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then ok; else bad "$name: $file 未包含 [$pattern]"; fi
}

echo "── release compose 五个 bind mount ──"
contains "PostgreSQL 可配置挂载" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "TimescaleDB 可配置挂载" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "Redis 可配置挂载" '${REDIS_DATA_PATH:-redisdata}:/data' "$RELEASE_COMPOSE"
contains "NATS 可配置挂载" '${NATS_DATA_PATH:-natsdata}:/data' "$RELEASE_COMPOSE"
contains "MinIO 可配置挂载" '${MINIO_DATA_PATH:-miniodata}:/data' "$RELEASE_COMPOSE"

echo "── 32核生产默认 CPU 配额 ──"
contains "PostgreSQL 默认 10 核" 'cpus: "${POSTGRES_CPUS:-10}"' "$RELEASE_COMPOSE"
contains "TimescaleDB 默认 16 核" 'cpus: "${TSDB_CPUS:-16}"' "$RELEASE_COMPOSE"
contains "worker 默认 8 核" 'cpus: "${WORKER_CPUS:-8}"' "$RELEASE_APP_COMPOSE"

echo "── PM 流式聚合生产旋钮 ──"
contains "release worker 启用流式聚合" 'PM_AGGREGATION_ENABLED: "${PM_AGGREGATION_ENABLED:-true}"' "$RELEASE_APP_COMPOSE"
contains "release worker 透传窗口状态 TTL" 'PM_AGGREGATION_WINDOW_TTL: "${PM_AGGREGATION_WINDOW_TTL:-1080h}"' "$RELEASE_APP_COMPOSE"
contains "开发 worker 启用流式聚合" 'PM_AGGREGATION_ENABLED: "${PM_AGGREGATION_ENABLED:-true}"' "$DEV_COMPOSE"

echo "── release .env 模板和升级继承 ──"
for key in POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH; do
  contains "$key 模板" "$key=" "$BUILD"
  contains "$key 升级继承" "$key" "$INSTALL"
done

echo "── install/svc 启动前准备路径 ──"
contains "install 加载存储库" 'storage-paths-lib.sh' "$INSTALL"
contains "install 准备目录" 'storage_prepare_configured_env_paths "$ENV_FILE"' "$INSTALL"
contains "svc 加载存储库" 'storage-paths-lib.sh' "$SVC"
contains "svc 准备目录" 'storage_prepare_configured_env_paths ".env"' "$SVC"

echo "── 开发 compose 保留命名卷默认值 ──"
contains "开发 PostgreSQL 默认命名卷" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 TimescaleDB 默认命名卷" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 Redis 默认命名卷" '${REDIS_DATA_PATH:-redisdata}:/data' "$DEV_COMPOSE"
contains "开发 NATS 默认命名卷" '${NATS_DATA_PATH:-natsdata}:/data' "$DEV_COMPOSE"
contains "开发 MinIO 默认命名卷" '${MINIO_DATA_PATH:-miniodata}:/data' "$DEV_COMPOSE"

echo "── 过载保护配置 ──"
contains "release TimescaleDB 共享内存兜底" 'shm_size: ${TSDB_SHM_SIZE:-512m}' "$RELEASE_COMPOSE"
contains "开发 TimescaleDB 共享内存兜底" 'shm_size: ${TSDB_SHM_SIZE:-512m}' "$DEV_COMPOSE"
contains "release NATS 内存存储上限" 'max_memory_store: ${NATS_MAX_MEMORY_STORE:-134217728}' "$RELEASE_COMPOSE"
contains "开发 NATS 内存存储上限" 'max_memory_store: ${NATS_MAX_MEMORY_STORE:-134217728}' "$DEV_COMPOSE"
contains "release NATS 大积压恢复宽限" 'start_period: 5m' "$RELEASE_COMPOSE"
contains "开发 NATS 大积压恢复宽限" 'start_period: 5m' "$DEV_COMPOSE"
contains "ACS access log 默认关闭" 'access_log off; # ACS 高频请求由应用指标观测，避免与数据盘竞争 IO' "$NGINX_DEFAULT"
contains "本地 ACS access log 默认关闭" 'access_log off; # ACS 高频请求由应用指标观测，避免与数据盘竞争 IO' "$NGINX_LOCAL"
contains "ACS 请求体不落临时文件" 'proxy_request_buffering off;' "$NGINX_DEFAULT"
contains "本地 ACS 请求体不落临时文件" 'proxy_request_buffering off;' "$NGINX_LOCAL"
contains "release MinIO scanner 最低速" 'MINIO_SCANNER_SPEED: "slowest"' "$RELEASE_COMPOSE"
contains "开发 MinIO scanner 最低速" 'MINIO_SCANNER_SPEED: "slowest"' "$DEV_COMPOSE"
contains "release Redis AOF 基线增大" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$RELEASE_COMPOSE"
contains "开发 Redis AOF 基线增大" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$DEV_COMPOSE"
contains "测试 Redis AOF 基线增大" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$TEST_COMPOSE"
contains "开发 TSDB 保留 TimescaleDB 并预载 pg_stat_statements" 'shared_preload_libraries=timescaledb,pg_stat_statements' "$DEV_COMPOSE"
contains "release TSDB 保留 TimescaleDB 并预载 pg_stat_statements" 'shared_preload_libraries=timescaledb,pg_stat_statements' "$RELEASE_COMPOSE"
contains "开发 TSDB 开启 I/O timing" 'track_io_timing=${TSDB_TRACK_IO_TIMING:-on}' "$DEV_COMPOSE"
contains "release TSDB 开启 I/O timing" 'track_io_timing=${TSDB_TRACK_IO_TIMING:-on}' "$RELEASE_COMPOSE"
contains "开发 TSDB 慢 SQL 默认 1 秒" 'log_min_duration_statement=${TSDB_LOG_MIN_DURATION_STATEMENT:-1000}' "$DEV_COMPOSE"
contains "release TSDB 慢 SQL 默认 1 秒" 'log_min_duration_statement=${TSDB_LOG_MIN_DURATION_STATEMENT:-1000}' "$RELEASE_COMPOSE"
contains "开发 TSDB 记录全部临时文件" 'log_temp_files=${TSDB_LOG_TEMP_FILES:-0}' "$DEV_COMPOSE"
contains "release TSDB 记录全部临时文件" 'log_temp_files=${TSDB_LOG_TEMP_FILES:-0}' "$RELEASE_COMPOSE"
contains "collector 配置 TSDB SQL 指标采集" 'sqlquery/tsdb:' "$OTELCOL_CONFIG"
contains "TSDB 指标管道包含 SQL 采集" 'receivers: [postgresql/tsdb, sqlquery/tsdb]' "$OTELCOL_CONFIG"

echo "── 磁盘 I/O 可观测性 ──"
contains "磁盘活动时钟面板不称为利用率" '磁盘 I/O 活动时钟占比' "$HOST_DASHBOARD"
contains "磁盘排队面板" 'node_disk_io_time_weighted_seconds_total' "$HOST_DASHBOARD"
contains "MinIO scanner 面板" 'minio_node_scanner_objects_scanned' "$HOST_DASHBOARD"
contains "磁盘复合饱和告警" 'alert: HostDiskIOSaturated' "$HOST_ALERTS"
contains "磁盘告警要求活动时间" 'node_disk_io_time_seconds_total' "$HOST_ALERTS"
contains "磁盘告警要求等待时延" 'node_disk_read_time_seconds_total' "$HOST_ALERTS"
contains "磁盘告警要求排队" 'node_disk_io_time_weighted_seconds_total' "$HOST_ALERTS"

echo "── production tracing + monitoring profile ──"
if bash "$RELEASE_DEPLOY/monitoring-profile_test.sh"; then
  ok
else
  bad "monitoring profile executable regression"
fi
for config in "$APP_PROD_CONFIG" "$ACS_PROD_CONFIG" "$WORKER_PROD_CONFIG"; do
  contains "生产 tracing 已启用" 'enabled: true' "$config"
  contains "生产 tracing 指向随包 collector" 'endpoint: "otelcol:4317"' "$config"
done
contains "默认安装包含完整监控 compose" '[ "$SKIP_MONITORING" = 0 ] && COMPOSE_FILES+=( -f docker-compose.monitoring.yml )' "$INSTALL"
contains "install 在 source 后应用监控 profile" 'monitoring_profile_apply_install "$ENV_FILE" "$SKIP_MONITORING"' "$INSTALL"
contains "升级强制刷新监控配置 bind mount" '"${DC[@]}" up -d --force-recreate --no-deps prometheus alertmanager loki tempo otelcol grafana' "$INSTALL"
contains "服务控制读取持久化监控 profile" 'monitoring_profile_apply_runtime ".env" "$SKIP_MONITORING"' "$SVC"
contains "监控 profile 关闭 tracing" 'export OMCGO_TRACER_ENABLED=false' "$MONITORING_PROFILE_LIB"
contains "业务容器 tracing 尊重配置与显式覆盖" 'OMCGO_TRACER_ENABLED: "${OMCGO_TRACER_ENABLED:-}"' "$RELEASE_APP_COMPOSE"
contains "健康检查读取持久化监控 profile" 'monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING"' "$RELEASE_HEALTHCHECK"
contains "collector 启用 health_check extension" 'extensions: [health_check, zpages]' "$OTELCOL_CONFIG"
contains "collector health 仅绑定宿主回环" '127.0.0.1:13133:13133' "$RELEASE_MONITORING_COMPOSE"
contains "healthcheck 从 collector 外部探测" 'curl -fsS http://127.0.0.1:13133/' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 NATS exporter" 'nats-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 nginx exporter" 'nginx-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 node exporter" 'node-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 cAdvisor" 'cadvisor' "$RELEASE_HEALTHCHECK"

echo "── discovered counter operational queries ──"
contains "PM 配置外指标告警使用新名称" 'omc_pm_discovered_counters_total' "$OMC_ALERTS"
contains "主 Grafana dashboard 使用新名称" 'omc_pm_discovered_counters_total' "$GRAFANA_DASHBOARD"
contains "overview dashboard 使用新名称" 'omc_pm_discovered_counters_total' "$GRAFANA_OVERVIEW"

echo "── 开发 planner maximize 分支 ──"
TMP_MAX="$(mktemp)"
TMP_PROBE_BIN="$(mktemp -d)"
trap 'rm -f "$TMP_MAX"; rm -rf "$TMP_PROBE_BIN"' EXIT
for command in docker sysctl uname; do
  printf '#!/usr/bin/env bash\nexit 97\n' > "$TMP_PROBE_BIN/$command"
  chmod +x "$TMP_PROBE_BIN/$command"
done
if env PATH="$TMP_PROBE_BIN:$PATH" OMC_PROBE_CPU=32 OMC_PROBE_MEM_TOTAL_MIB=32768 \
   bash "$DEV_PLANNER" --maximize --assume-dedicated --disk-gib 900 -o "$TMP_MAX" >/dev/null 2>&1 &&
   grep -q '^NATS_MAX_MEMORY_STORE=[0-9][0-9]*$' "$TMP_MAX"; then
  ok
else
  bad "开发 planner what-if 模式不应依赖宿主探测且应输出 NATS_MAX_MEMORY_STORE"
fi

echo "════ Results: PASS=$PASS FAIL=$FAIL ════"
[ "$FAIL" -eq 0 ]

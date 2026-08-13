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
INFRA_ALERTS="$REPO_ROOT/deployments/monitoring/alerts/infra-alerts.yml"
HOST_ALERTS="$REPO_ROOT/deployments/monitoring/alerts/host-container-alerts.yml"
RESOURCE_PLAN_METRICS="$RELEASE_DEPLOY/resource-plan-metrics.sh"
GRAFANA_DASHBOARD="$REPO_ROOT/deployments/monitoring/grafana-dashboard.json"
GRAFANA_OVERVIEW="$REPO_ROOT/deployments/monitoring/grafana/dashboards/omc-overview.json"
HOST_DASHBOARD="$REPO_ROOT/deployments/monitoring/grafana/dashboards/nginx-host-overview.json"
APP_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/app/etc/config.prod.yaml"
ACS_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/acs/etc/config.prod.yaml"
WORKER_PROD_CONFIG="$REPO_ROOT/omcgo/cmd/worker/etc/config.prod.yaml"
INSTALL="$RELEASE_DEPLOY/install.sh"
UNINSTALL="$RELEASE_DEPLOY/uninstall.sh"
SVC="$RELEASE_DEPLOY/svc.sh"
BUILD="$REPO_ROOT/deployments/release/build-release.sh"
RELEASE_HANDOFF="$RELEASE_DEPLOY/gpv-handoff-lib.sh"
RELEASE_NATS_VERIFY="$RELEASE_DEPLOY/verify-gpv-nats.sh"
APP_DOCKERFILE="$REPO_ROOT/deployments/docker/Dockerfile.app"
DEV_PLANNER="$REPO_ROOT/deployments/docker/plan-resources.sh"
NGINX_DEFAULT="$REPO_ROOT/deployments/docker/default.conf"
NGINX_LOCAL="$REPO_ROOT/deployments/docker/default.local.conf"
TSDB_BASELINE="$REPO_ROOT/omcgo/migrations/tsdb/000001_tsdb_schema.sql"
SEED_BASELINE="$REPO_ROOT/omcgo/migrations/seed/000001_init_seed.sql"

PASS=0
FAIL=0
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then ok; else bad "$name: $file 未包含 [$pattern]"; fi
}
not_contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then bad "$name: $file 不应包含 [$pattern]"; else ok; fi
}
appears_before() {
  local name="$1" first="$2" second="$3" file="$4"
  local first_line second_line
  first_line="$(grep -nF -- "$first" "$file" | head -1 | cut -d: -f1)"
  second_line="$(grep -nF -- "$second" "$file" | head -1 | cut -d: -f1)"
  if [ -n "$first_line" ] && [ -n "$second_line" ] && [ "$first_line" -lt "$second_line" ]; then
    ok
  else
    bad "$name: [$first] 必须出现在 [$second] 之前"
  fi
}
valid_bash() {
  local name="$1" file="$2"
  if bash -n "$file"; then ok; else bad "$name: $file 存在 Bash 语法错误"; fi
}

echo "── release 运维脚本语法 ──"
valid_bash "healthcheck 可完整解析" "$RELEASE_HEALTHCHECK"

echo "── release compose 六个 bind mount ──"
contains "PostgreSQL 可配置挂载" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "TimescaleDB 可配置挂载" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$RELEASE_COMPOSE"
contains "Redis 可配置挂载" '${REDIS_DATA_PATH:-redisdata}:/data' "$RELEASE_COMPOSE"
contains "PM Redis 可配置挂载" '${REDIS_PM_DATA_PATH:-redispmdata}:/data' "$RELEASE_COMPOSE"
contains "核心 Redis 独立服务" 'redis-core:' "$RELEASE_COMPOSE"
contains "PM Redis 独立服务" 'redis-pm:' "$RELEASE_COMPOSE"
contains "PM Redis 迁移运维入口" 'pm-redis-migrate:' "$RELEASE_APP_COMPOSE"
contains "迁移入口默认 dry-run" 'entrypoint: ["omcctl", "pm-redis", "migrate"]' "$RELEASE_APP_COMPOSE"
contains "核心 Redis 资源键" '${REDIS_CORE_MAXMEMORY:-3gb}' "$RELEASE_COMPOSE"
contains "PM Redis 资源键" '${REDIS_PM_MAXMEMORY:-6gb}' "$RELEASE_COMPOSE"
not_contains "PM Redis 不发布宿主端口" '6381:6379' "$RELEASE_COMPOSE"
contains "NATS 可配置挂载" '${NATS_DATA_PATH:-natsdata}:/data' "$RELEASE_COMPOSE"
contains "MinIO 可配置挂载" '${MINIO_DATA_PATH:-miniodata}:/data' "$RELEASE_COMPOSE"

echo "── 32核生产默认 CPU 配额 ──"
contains "PostgreSQL 默认 10 核" 'cpus: "${POSTGRES_CPUS:-10}"' "$RELEASE_COMPOSE"
contains "TimescaleDB 默认 16 核" 'cpus: "${TSDB_CPUS:-16}"' "$RELEASE_COMPOSE"
contains "worker 默认 8 核" 'cpus: "${WORKER_CPUS:-8}"' "$RELEASE_APP_COMPOSE"

echo "── PM 流式聚合生产旋钮 ──"
contains "release worker 启用流式聚合" 'PM_AGGREGATION_ENABLED: "${PM_AGGREGATION_ENABLED:-true}"' "$RELEASE_APP_COMPOSE"
contains "release worker 聚合并发满足两万设备十二分钟关闭" 'PM_AGGREGATION_CONSUMER_CONCURRENCY: "${PM_AGGREGATION_CONSUMER_CONCURRENCY:-16}"' "$RELEASE_APP_COMPOSE"
contains "release worker 收口并发与生产默认预算一致" 'PM_AGGREGATION_FINALIZE_CONCURRENCY: "${PM_AGGREGATION_FINALIZE_CONCURRENCY:-32}"' "$RELEASE_APP_COMPOSE"
contains "release worker 透传窗口状态 TTL" 'PM_AGGREGATION_WINDOW_TTL: "${PM_AGGREGATION_WINDOW_TTL:-1080h}"' "$RELEASE_APP_COMPOSE"
contains "release worker 默认开启 Redis v2 紧凑写入" 'PM_AGGREGATION_REDIS_V2_WRITE_ENABLED: "${PM_AGGREGATION_REDIS_V2_WRITE_ENABLED:-true}"' "$RELEASE_APP_COMPOSE"
contains "开发 worker 启用流式聚合" 'PM_AGGREGATION_ENABLED: "${PM_AGGREGATION_ENABLED:-true}"' "$DEV_COMPOSE"
contains "开发 worker 默认开启 Redis v2 紧凑写入" 'PM_AGGREGATION_REDIS_V2_WRITE_ENABLED: "${PM_AGGREGATION_REDIS_V2_WRITE_ENABLED:-true}"' "$DEV_COMPOSE"

echo "── release .env 模板和升级继承 ──"
for key in POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH REDIS_PM_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH; do
  contains "$key 模板" "$key=" "$BUILD"
  contains "$key 升级继承" "$key" "$INSTALL"
done

echo "── release 安装严格离线镜像契约 ──"
contains "install 默认英文语言" 'OMC_LANG="${OMC_LANG:-en}"' "$INSTALL"
contains "install 导出语言给 healthcheck" 'export OMC_LANG=' "$INSTALL"
contains "install 支持中英文参数" '--lang|--language' "$INSTALL"
contains "install 依赖库缺失提供英文错误" 'Missing $DEPLOY_DIR/secrets-lib.sh' "$INSTALL"
contains "install 全新安装错误提供英文提示" 'Fresh install requires a valid --public-host' "$INSTALL"
contains "install Compose precheck 错误提供英文提示" 'Docker Compose v2 or docker-compose v1 was not found' "$INSTALL"
contains "install data 合并错误提供英文提示" 'Reverse data merge returned an error' "$INSTALL"
contains "install 配置迁移错误提供英文提示" 'automatic completion failed; current was not switched' "$INSTALL"
contains "install healthcheck 失败摘要支持英文" 'Health-check failure summary (failed items only)' "$INSTALL"
contains "install 基础设施目录缺失提供英文错误" 'Infrastructure image directory is missing:' "$INSTALL"
contains "install 首次 env 提供英文提示" '.env: first deployment (no previous release)' "$INSTALL"
contains "install 基础设施镜像已存在提供英文提示" 'Infrastructure and monitoring images already exist' "$INSTALL"
contains "install TSDB 协调提供英文提示" 'TSDB baseline compatibility reconciliation' "$INSTALL"
contains "install etc 备份提供英文提示" 'Backed up the previous etc directory' "$INSTALL"
contains "install etc 重置提供英文提示" 'etc was reset to the package template' "$INSTALL"
contains "install etc diff 提供英文提示" 'Compare changes with: diff -ru' "$INSTALL"
contains "install 版本目录覆盖提供英文提示" 'Release directory $RELEASE_DIR already exists and will be replaced' "$INSTALL"
contains "uninstall 默认英文语言" 'OMC_LANG="${OMC_LANG:-en}"' "$UNINSTALL"
contains "uninstall 支持中英文参数" '--lang|--language' "$UNINSTALL"
contains "uninstall 接受 cn/en" 'cn|en)' "$UNINSTALL"
contains "uninstall 提供英文帮助" 'OMC uninstall script' "$UNINSTALL"
contains "install 提供英文帮助" 'OMC installation / upgrade script' "$INSTALL"
contains "install 阶段提供英文文案" '1/9 Precheck' "$INSTALL"
contains "install 检查完整监控镜像清单" '"${IMAGE_NGINX_EXPORTER:-}" "${IMAGE_NODE_EXPORTER:-}" "${IMAGE_CADVISOR:-}"' "$INSTALL"
contains "install 缺镜像时禁止隐式联网拉取" '离线安装缺少本地镜像' "$INSTALL"
contains "install 提供显式全新安装模式" '--fresh-install' "$INSTALL"
contains "全新安装要求显式基站地址" '--public-host' "$INSTALL"
contains "全新安装支持自定义资源容忍度" '--floor-tolerance-pct <N>' "$INSTALL"
contains "全新安装传递资源规划语言" 'fresh_plan_args=( --floor-tolerance-pct "$FLOOR_TOLERANCE_PCT" --lang "$OMC_LANG" )' "$INSTALL"
contains "全新安装先规划资源再清理数据" '全新安装资源规划失败；请查看上方资源规划提示' "$INSTALL"
contains "全新安装清理项目 volumes" 'label=com.docker.compose.project="$COMPOSE_PROJECT"' "$INSTALL"
contains "infra 启动禁止 pull" '"${DC[@]}" up --pull never -d postgres postgres-tsdb redis-core redis-pm nats minio' "$INSTALL"
contains "业务启动禁止 pull" '"${DC[@]}" up --pull never -d' "$INSTALL"
contains "候选 ACS 启动禁止 pull" '"${DC[@]}" up --pull never -d --no-deps acs-candidate' "$INSTALL"
contains "一次性迁移成功隐藏 goose 输出" 'run_oneshot_migration()' "$INSTALL"
contains "一次性迁移失败保留容器输出" 'printf '\''%s\n'\'' "$output" >&2' "$INSTALL"
contains "安装先等待 App 就绪" 'app_wait_ready' "$INSTALL"
contains "App 就绪后再启动 Worker" 'remaining_services=(worker)' "$INSTALL"
contains "升级启动 App 前停止旧 Worker" '停止现有 Worker，避免 App 启动期争抢数据库连接' "$INSTALL"
contains "App 启动失败输出诊断" '请根据上方 App 日志排查数据库超时或资源不足' "$INSTALL"
contains "App 启动超时可配置" 'OMC_APP_START_TIMEOUT:-180' "$INSTALL"
appears_before "Worker 启动晚于 App 就绪" 'if ! app_wait_ready; then' 'remaining_services=(worker)' "$INSTALL"
contains "安装在升级写操作前校验基站地址" 'OMC_PUBLIC_HOST 预检通过' "$INSTALL"
contains "安装健康检查支持最终复核" 'HEALTHCHECK_FINAL_GRACE' "$INSTALL"
contains "安装健康检查默认动态等待 90 秒" 'OMC_HEALTHCHECK_TIMEOUT:-90' "$INSTALL"
contains "安装健康检查默认总窗口不额外延长" 'OMC_HEALTHCHECK_FINAL_GRACE:-0' "$INSTALL"
appears_before "安装健康检查在监控重建（cadvisor storm）前执行" '动态等待业务容器启动' '刷新版本目录 bind mount' "$INSTALL"
contains "安装失败只输出健康检查失败摘要" '健康检查失败摘要（仅显示失败项）' "$INSTALL"
contains "安装稳定业务容器跳过正常日志" '业务容器均稳定运行，跳过正常运行日志' "$INSTALL"
for key in GPV_PROVISION_QUEUE GPV_PROVISION_CONCURRENCY GPV_PROVISION_QUEUE_DEPTH \
  GPV_RPC_DURABLE GPV_RPC_SOURCE_CONSUMER GPV_RPC_START_SEQUENCE GPV_RPC_CONCURRENCY GPV_RPC_QUEUE_DEPTH \
  GPV_ACK_WAIT GPV_MAX_DELIVER GPV_MAX_ACK_PENDING; do
  contains "$key release app 透传" "$key:" "$RELEASE_APP_COMPOSE"
  contains "$key 升级继承" "$key" "$INSTALL"
done

echo "── 实例配置安全升级 ──"
contains "install 加载实例配置升级库" 'config-upgrade-lib.sh' "$INSTALL"
contains "普通升级迁移 ACS 历史默认值" 'upgrade_acs_session_limit' "$INSTALL"
contains "普通升级补齐 GPV 消费配置" 'upgrade_app_gpv_response_config' "$INSTALL"
contains "普通升级迁移参数同步恢复上限" 'upgrade_app_param_sync_recovery_limit' "$INSTALL"
contains "普通升级迁移 Worker TSDB 历史默认值" 'upgrade_worker_tsdb_pool' "$INSTALL"
contains "Worker TSDB 自定义值不足时阻断升级" 'WORKER_TSDB_POOL_UPGRADE_RESULT:-invalid' "$INSTALL"
contains "安装预检校验 Worker TSDB 安全预算" 'validate_worker_tsdb_pool_precheck' "$INSTALL"
appears_before "Worker TSDB 门禁先于旧服务停机" \
  'validate_worker_tsdb_pool_precheck' \
  'gpv_handoff_migrate_legacy_systemd' \
  "$INSTALL"
if bash "$RELEASE_DEPLOY/config-upgrade-lib_test.sh"; then
  ok
else
  bad "ACS session limit config migration regression"
fi

echo "── install/svc 启动前准备路径 ──"
contains "install 加载存储库" 'storage-paths-lib.sh' "$INSTALL"
contains "install 准备目录" 'storage_prepare_configured_env_paths "$ENV_FILE"' "$INSTALL"
contains "svc 加载存储库" 'storage-paths-lib.sh' "$SVC"
contains "svc 准备目录" 'storage_prepare_configured_env_paths ".env"' "$SVC"
contains "app 镜像构建 GPV handoff 工具" 'omcgo-gpv-handoff ./cmd/gpv-handoff' "$APP_DOCKERFILE"
contains "release compose 提供 handoff 一次性服务" 'gpv-handoff:' "$RELEASE_APP_COMPOSE"
contains "install 加载 handoff 库" 'gpv-handoff-lib.sh' "$INSTALL"
contains "install 加载 Redis 切换库" 'redis-cutover-lib.sh' "$INSTALL"
contains "旧 Redis 切换先停止写入方" 'prepare_legacy_redis_cutover' "$INSTALL"
contains "旧 Redis 切换校验数据连续性" 'verify_legacy_redis_cutover' "$INSTALL"
contains "旧 Redis 切换迁移 PM 状态" 'migrate_legacy_pm_redis' "$INSTALL"
contains "svc 加载 handoff 库" 'gpv-handoff-lib.sh' "$SVC"
contains "install 在业务 up 前准备 durable" 'gpv_handoff_prepare' "$INSTALL"
contains "install 在 systemd 停服前执行迁移门禁" 'gpv_handoff_migrate_legacy_systemd' "$INSTALL"
contains "svc 在重启前准备 durable" 'gpv_handoff_prepare' "$SVC"
contains "真实 NATS 验证脚本强制注入地址" 'GPV_NATS_TEST_URL=' "$RELEASE_NATS_VERIFY"
contains "真实 NATS 验证支持本地 Docker fallback" 'NATS_SERVER_IMAGE' "$RELEASE_NATS_VERIFY"
appears_before "会清空 stream 的 fresh-install 测试晚于队列测试" \
  '"$GO_BIN" test ./internal/core/event' \
  '"$GO_BIN" test ./cmd/gpv-handoff' \
  "$RELEASE_NATS_VERIFY"
contains "真实 NATS 验证支持固定 Go 路径兜底" '/root/.local/go/bin/go' "$RELEASE_NATS_VERIFY"
contains "真实 NATS 验证支持标准用户态 Go 解压路径" '$HOME/.opencode/go/bin/go' "$RELEASE_NATS_VERIFY"
contains "install 健康等待使用真实截止时间" 'HEALTHCHECK_DEADLINE=' "$INSTALL"
contains "单轮 healthcheck 有独立探针超时" 'HEALTHCHECK_PROBE_TIMEOUT' "$INSTALL"
contains "单轮 healthcheck 超时后继续重试" 'HEALTHCHECK_PROBE_REMAINING' "$INSTALL"
contains "安装记录单轮 healthcheck 超时" 'HEALTHCHECK_PROBE_TIMEOUTS' "$INSTALL"
contains "安装使用轻量启动检查" 'healthcheck.sh" --lang "$OMC_LANG" --startup' "$INSTALL"
contains "安装最终检查传递语言" 'healthcheck.sh" --lang "$OMC_LANG" >' "$INSTALL"
contains "启动检查说明跳过深审计" '完整 healthcheck 将在部署后执行' "$RELEASE_HEALTHCHECK"
contains "启动检查直接使用 Docker 容器标签" 'label=com.docker.compose.service=$svc' "$RELEASE_HEALTHCHECK"
contains "启动检查按 YAML 角色解析" 'redis-routing-check-lib.sh' "$RELEASE_HEALTHCHECK"
contains "启动检查跳过重型审计" 'STARTUP_CHECK=0' "$RELEASE_HEALTHCHECK"
contains "healthcheck 默认英文语言" 'OMC_LANG="${OMC_LANG:-en}"' "$RELEASE_HEALTHCHECK"
contains "healthcheck 支持中英文参数" '--lang|--language' "$RELEASE_HEALTHCHECK"
contains "release planner 默认英文语言" 'OMC_LANG="${OMC_LANG:-en}"' "$RELEASE_DEPLOY/plan-resources.sh"
contains "release planner 支持中英文参数" '--lang|--language' "$RELEASE_DEPLOY/plan-resources.sh"
contains "release planner 英文帮助" 'Output language (default en)' "$RELEASE_DEPLOY/plan-resources.sh"
contains "healthcheck 英文启动结果" 'Startup check result' "$RELEASE_HEALTHCHECK"
contains "启动 HTTP 探针有单次超时" 'curl -fsS --max-time 3' "$RELEASE_HEALTHCHECK"
not_contains "健康等待不得按固定步长伪计时" 'HEALTHCHECK_WAIT=$((HEALTHCHECK_WAIT + HEALTHCHECK_INTERVAL))' "$INSTALL"
if bash "$RELEASE_DEPLOY/gpv-handoff-lib_test.sh"; then
  ok
else
  bad "systemd 到 container 的 GPV handoff 顺序回归"
fi
if bash "$RELEASE_DEPLOY/acs-ha-rollout-test.sh"; then
  ok
else
  bad "ACS 双实例无损发布契约回归"
fi
if bash "$RELEASE_DEPLOY/redis-cutover-lib_test.sh"; then
  ok
else
  bad "旧单 Redis 到双实例切换顺序回归"
fi
if bash "$RELEASE_DEPLOY/redis-routing-check-lib_test.sh"; then
  ok
else
  bad "Redis YAML 角色解析回归"
fi
if bash "$RELEASE_NATS_VERIFY"; then
  ok
else
  bad "真实 NATS GPV handoff / FIFO 回归"
fi
not_contains "业务镜像存在时不得提前重启 app" '业务镜像已存在，跳过 load，重启业务容器' "$INSTALL"
if bash "$RELEASE_DEPLOY/install-resource-preflight_test.sh"; then
  ok
else
  bad "install/svc resources.env preflight regression"
fi

echo "── 开发 compose 保留命名卷默认值 ──"
contains "开发 PostgreSQL 默认命名卷" '${POSTGRES_DATA_PATH:-pgdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 TimescaleDB 默认命名卷" '${TSDB_DATA_PATH:-tsdbdata}:/var/lib/postgresql/data' "$DEV_COMPOSE"
contains "开发 Redis 默认命名卷" '${REDIS_DATA_PATH:-redisdata}:/data' "$DEV_COMPOSE"
contains "开发 PM Redis 默认命名卷" '${REDIS_PM_DATA_PATH:-redispmdata}:/data' "$DEV_COMPOSE"
contains "开发 NATS 默认命名卷" '${NATS_DATA_PATH:-natsdata}:/data' "$DEV_COMPOSE"
contains "开发 MinIO 默认命名卷" '${MINIO_DATA_PATH:-miniodata}:/data' "$DEV_COMPOSE"
contains "开发写入保护默认匹配 compose 项目名前缀" 'OMCGO_DOCKER_VOLUME_PREFIX: "${OMCGO_DOCKER_VOLUME_PREFIX:-docker}"' "$DEV_COMPOSE"
contains "release 写入保护默认匹配生产 compose 项目前缀" 'OMCGO_DOCKER_VOLUME_PREFIX: "${OMCGO_DOCKER_VOLUME_PREFIX:-omcgo}"' "$RELEASE_APP_COMPOSE"

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
contains "release ACS 使用可动态解析的 upstream 池" 'server acs:7557 resolve;' "$NGINX_DEFAULT"
contains "本地 ACS 使用 upstream 池" 'server 127.0.0.1:7557;' "$NGINX_LOCAL"
contains "release ACS upstream 启用长连接复用" 'keepalive 4096;' "$NGINX_DEFAULT"
contains "本地 ACS upstream 启用长连接复用" 'keepalive 4096;' "$NGINX_LOCAL"
contains "release ACS 代理走连接池" 'proxy_pass http://acs_backend;' "$NGINX_DEFAULT"
contains "本地 ACS 代理走连接池" 'proxy_pass http://acs_backend;' "$NGINX_LOCAL"
not_contains "release ACS 不再逐请求变量式建连" 'set $acs_upstream acs:7557;' "$NGINX_DEFAULT"
not_contains "本地 ACS 不再逐请求变量式建连" 'set $acs_upstream 127.0.0.1:7557;' "$NGINX_LOCAL"
contains "release web 扩大上游临时端口范围" 'net.ipv4.ip_local_port_range: "10240 65535"' "$REPO_ROOT/deployments/release/bundle/deploy/docker-compose.web.yml"
contains "开发 web 扩大上游临时端口范围" 'net.ipv4.ip_local_port_range: "10240 65535"' "$DEV_COMPOSE"
contains "release ACS 扩大 accept backlog" 'net.core.somaxconn: "32768"' "$RELEASE_APP_COMPOSE"
contains "release ACS 扩大 SYN backlog" 'net.ipv4.tcp_max_syn_backlog: "32768"' "$RELEASE_APP_COMPOSE"
contains "开发 ACS 扩大 accept backlog" 'net.core.somaxconn: "32768"' "$DEV_COMPOSE"
contains "开发 ACS 扩大 SYN backlog" 'net.ipv4.tcp_max_syn_backlog: "32768"' "$DEV_COMPOSE"
contains "release MinIO scanner 最低速" 'MINIO_SCANNER_SPEED: "slowest"' "$RELEASE_COMPOSE"
contains "开发 MinIO scanner 最低速" 'MINIO_SCANNER_SPEED: "slowest"' "$DEV_COMPOSE"
contains "release 核心 Redis AOF 基线" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$RELEASE_COMPOSE"
contains "release PM Redis AOF 基线" '--auto-aof-rewrite-min-size 2gb --auto-aof-rewrite-percentage 500' "$RELEASE_COMPOSE"
contains "开发核心 Redis AOF 基线" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$DEV_COMPOSE"
contains "开发 PM Redis AOF 基线" '--auto-aof-rewrite-min-size 2gb --auto-aof-rewrite-percentage 500' "$DEV_COMPOSE"
contains "测试 Redis AOF 基线增大" '--auto-aof-rewrite-min-size 1gb --auto-aof-rewrite-percentage 500' "$TEST_COMPOSE"
contains "release Redis 禁止淘汰聚合状态" '--maxmemory-policy noeviction' "$RELEASE_COMPOSE"
contains "开发 Redis 禁止淘汰聚合状态" '--maxmemory-policy noeviction' "$DEV_COMPOSE"
contains "release PM Redis 默认容纳双小时重叠窗口" '--maxmemory ${REDIS_PM_MAXMEMORY:-6gb}' "$RELEASE_COMPOSE"
contains "release PM Redis 默认保留 2GiB COW" 'memory: "${REDIS_PM_MEM:-8g}"' "$RELEASE_COMPOSE"
contains "release 核心 Redis 默认隔离预算" '--maxmemory ${REDIS_CORE_MAXMEMORY:-3gb}' "$RELEASE_COMPOSE"
contains "开发 PM Redis 默认容纳双小时重叠窗口" '--maxmemory ${REDIS_PM_MAXMEMORY:-6gb}' "$DEV_COMPOSE"
contains "开发 PM Redis 默认保留 2GiB COW" 'memory: ${REDIS_PM_MEM:-8g}' "$DEV_COMPOSE"
contains "release 资源规划 PM Redis 双窗口下限" 'redis-pm      8192' "$RELEASE_DEPLOY/plan-resources.sh"
contains "release 资源规划 PM Redis 保留 2GiB COW" 'REDIS_PM_MAXMEM=$(( REDIS_PM_MEM - 2048 ))' "$RELEASE_DEPLOY/plan-resources.sh"
contains "开发资源规划 PM Redis 双窗口下限" 'redis-pm     8192' "$DEV_PLANNER"
contains "开发资源规划 PM Redis 保留 2GiB COW" 'REDIS_PM_MAXMEM=$(( REDIS_PM_MEM - 2048 ))' "$DEV_PLANNER"
contains "release 资源规划禁止淘汰聚合状态" 'REDIS_POLICY=noeviction' "$RELEASE_DEPLOY/plan-resources.sh"
contains "开发资源规划禁止淘汰聚合状态" 'REDIS_POLICY=noeviction' "$DEV_PLANNER"
contains "监控采集核心 Redis" 'endpoint: redis-core:6379' "$OTELCOL_CONFIG"
contains "监控采集 PM Redis" 'endpoint: redis-pm:6379' "$OTELCOL_CONFIG"
contains "Redis 指标区分核心实例" 'set(attributes["instance"], "redis-core")' "$OTELCOL_CONFIG"
contains "Redis 指标区分 PM 实例" 'set(attributes["instance"], "redis-pm")' "$OTELCOL_CONFIG"
contains "核心 Redis 独立断流告警" 'alert: RedisCoreMetricsAbsent' "$INFRA_ALERTS"
contains "PM Redis 独立断流告警" 'alert: RedisPMMetricsAbsent' "$INFRA_ALERTS"
contains "队列样本陈旧只检查 ACS" 'omc_pm_queue_sample_timestamp_seconds{deployment_unit="acs",subject="pm.file.received",durable="pm-workers"}' "$OMC_ALERTS"
contains "Redis 上限告警说明 noeviction" 'noeviction 会拒绝新写入' "$INFRA_ALERTS"
contains "PM 聚合事件失败告警" 'alert: OMCPMStreamingAggregationEventFailures' "$OMC_ALERTS"
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
contains "升级统一刷新完整 compose stack" '"${DC[@]}" up --pull never -d' "$INSTALL"
contains "升级强制刷新版本目录 bind mount" '"${DC[@]}" up --pull never -d --force-recreate --no-deps prometheus alertmanager grafana loki otelcol tempo' "$INSTALL"
contains "服务控制读取持久化监控 profile" 'monitoring_profile_apply_runtime ".env" "$SKIP_MONITORING"' "$SVC"
contains "监控 profile 关闭 tracing" 'export OMCGO_TRACER_ENABLED=false' "$MONITORING_PROFILE_LIB"
contains "业务容器 tracing 尊重配置与显式覆盖" 'OMCGO_TRACER_ENABLED: "${OMCGO_TRACER_ENABLED:-}"' "$RELEASE_APP_COMPOSE"
contains "App API endpoint 同步超时可配置" 'OMCGO_API_ENDPOINT_SYNC_TIMEOUT_SECONDS: "${OMCGO_API_ENDPOINT_SYNC_TIMEOUT_SECONDS:-120}"' "$RELEASE_APP_COMPOSE"
contains "健康检查读取持久化监控 profile" 'monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING"' "$RELEASE_HEALTHCHECK"
contains "collector 启用 health_check extension" 'extensions: [health_check, zpages]' "$OTELCOL_CONFIG"
contains "collector health 仅绑定宿主回环" '127.0.0.1:13133:13133' "$RELEASE_MONITORING_COMPOSE"
contains "healthcheck 从 collector 外部探测" 'curl -fsS --max-time 3 http://127.0.0.1:13133/' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 NATS exporter" 'nats-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 nginx exporter" 'nginx-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 node exporter" 'node-exporter' "$RELEASE_HEALTHCHECK"
contains "healthcheck 覆盖 cAdvisor" 'cadvisor' "$RELEASE_HEALTHCHECK"
contains "healthcheck 核对 ACS upstream 连接池" 'nginx -T' "$RELEASE_HEALTHCHECK"
contains "healthcheck 核对 web 临时端口范围" 'net.ipv4.ip_local_port_range' "$RELEASE_HEALTHCHECK"
contains "healthcheck 核对 ACS accept backlog" 'net.core.somaxconn' "$RELEASE_HEALTHCHECK"
contains "healthcheck 核对 ACS SYN backlog" 'net.ipv4.tcp_max_syn_backlog' "$RELEASE_HEALTHCHECK"
contains "ACS 全局会话拒绝告警" 'alert: ACSGlobalAdmissionRejected' "$REPO_ROOT/deployments/monitoring/alerts/runtime-alerts.yml"

echo "── PM 指标漂移与禁用值监控 ──"
contains "PM 配置外指标告警使用 whitelist miss" 'omc_pm_whitelist_miss_values_total' "$OMC_ALERTS"
not_contains "按配置禁用的 PM 指标不得触发持续业务告警" 'alert: PMKnownIndicatorsDisabled' "$OMC_ALERTS"
contains "BLQ 路由基线修复高优先级接入拼写" "indicator_id = 'C000000014'" "$SEED_BASELINE"
contains "主 Grafana dashboard 展示 whitelist miss" 'omc_pm_whitelist_miss_values_total' "$GRAFANA_DASHBOARD"
contains "主 Grafana dashboard 展示 disabled" 'omc_pm_known_disabled_values_total' "$GRAFANA_DASHBOARD"
contains "overview dashboard 展示 whitelist miss" 'omc_pm_whitelist_miss_values_total' "$GRAFANA_OVERVIEW"
contains "overview dashboard 展示 disabled" 'omc_pm_known_disabled_values_total' "$GRAFANA_OVERVIEW"

echo "── PM 聚合关闭、Redis 与资源漂移监控 ──"
contains "终结领取索引覆盖稳定排序键" 'granularity, window_end, task_version_id, entity_key, window_start' "$TSDB_BASELINE"
contains "小时层级水位索引覆盖上游窗口" 'granularity, window_start, task_version_id' "$TSDB_BASELINE"
contains "资源计划漂移告警" 'alert: OMCResourcePlanDrift' "$HOST_ALERTS"
contains "资源计划 cAdvisor 缺失 critical 告警" 'alert: OMCResourcePlanCAdvisorAbsent' "$HOST_ALERTS"
contains "资源计划 CPU quota 序列缺失告警" 'alert: OMCResourcePlanCPUQuotaSeriesAbsent' "$HOST_ALERTS"
contains "资源计划 CPU period 序列缺失告警" 'alert: OMCResourcePlanCPUPeriodSeriesAbsent' "$HOST_ALERTS"
contains "资源计划内存 limit 序列缺失告警" 'alert: OMCResourcePlanMemoryLimitSeriesAbsent' "$HOST_ALERTS"
contains "资源计划漂移检查 CPU quota" 'container_spec_cpu_quota' "$HOST_ALERTS"
contains "资源计划漂移检查 CPU period" 'container_spec_cpu_period' "$HOST_ALERTS"
contains "资源计划漂移检查内存实际限额" 'container_spec_memory_limit_bytes' "$HOST_ALERTS"
contains "资源漂移规则使用生成的计划 CPU 指标" 'omc_resource_plan_cpu_cores' "$HOST_ALERTS"
contains "资源漂移规则使用生成的计划内存指标" 'omc_resource_plan_memory_limit_bytes' "$HOST_ALERTS"
contains "cAdvisor 当前容器新鲜度记录规则" 'record: omc:container_last_seen:fresh' "$HOST_ALERTS"
fresh_container_filters=$(grep -Fc 'and on (id) omc:container_last_seen:fresh' "$HOST_ALERTS")
if [ "$fresh_container_filters" -ge 10 ]; then
  ok
else
  bad "cAdvisor 资源告警新鲜度过滤不足：got=$fresh_container_filters want>=10"
fi
contains "持久化队列告警去重多进程快照" 'sum by (queue) (max by (queue, status) (omc_persistent_queue_pending{status=~"pending|running"}' "$REPO_ROOT/deployments/monitoring/alerts/storage-queue-alerts.yml"
contains "持久化队列 sent 任务独立按年龄治理" 'alert: PersistentQueueInFlightStale' "$REPO_ROOT/deployments/monitoring/alerts/storage-queue-alerts.yml"
contains "持久化队列 sent 任务按自身到期时间治理" 'omc_persistent_queue_overdue_oldest_age_seconds{queue="device_tasks",status="sent"}' "$REPO_ROOT/deployments/monitoring/alerts/storage-queue-alerts.yml"
contains "存储队列 Dashboard 去重多进程快照" 'max by (queue, status) (omc_persistent_queue_pending)' "$REPO_ROOT/deployments/monitoring/grafana/dashboards/omc-storage-queue-governance.json"
contains "基础设施 Dashboard 去重多进程快照" 'max by (queue, status) (omc_persistent_queue_pending)' "$REPO_ROOT/deployments/monitoring/grafana/dashboards/omc-infra.json"
contains "node exporter 读取资源计划 textfile" '--collector.textfile.directory=/textfile' "$RELEASE_MONITORING_COMPOSE"
contains "node exporter 挂载资源计划 textfile" '/opt/omc/run/monitoring:/textfile:ro' "$RELEASE_MONITORING_COMPOSE"
contains "cAdvisor 使用 Docker 29/overlayfs 支持版本" 'gcr.m.daocloud.io/cadvisor/cadvisor:v0.55.1' "$REPO_ROOT/deployments/release/release.conf"
contains "安装生成资源计划指标" 'resource_plan_metrics_write' "$INSTALL"
contains "服务控制生成资源计划指标" 'resource_plan_metrics_write' "$SVC"
if bash "$RELEASE_DEPLOY/resource-plan-metrics_test.sh"; then
  ok
else
  bad "资源计划 Prometheus textfile 生成回归"
fi
contains "Redis 聚合内存 warning 告警" 'alert: RedisAggregationMemoryHigh' "$INFRA_ALERTS"
contains "Redis 聚合内存 warning 阈值" '>= 0.80' "$INFRA_ALERTS"
contains "Redis 聚合内存 critical 告警" 'alert: RedisAggregationMemoryCritical' "$INFRA_ALERTS"
contains "Redis 聚合内存 critical 阈值" '>= 0.90' "$INFRA_ALERTS"
contains "关闭尾延迟 warning 告警" 'alert: PMFinalizeOldestDueHigh' "$OMC_ALERTS"
contains "关闭尾延迟 warning 阈值" '> 30m' "$OMC_ALERTS"
contains "关闭尾延迟 critical 告警" 'alert: PMFinalizeOldestDueCritical' "$OMC_ALERTS"
contains "关闭尾延迟 critical 阈值" '> 60m' "$OMC_ALERTS"
contains "重算快照扫描成本告警" 'alert: PMRebuildSnapshotScanSlow' "$OMC_ALERTS"
contains "重算快照扫描 P95 指标" 'omc_pm_aggregation_rebuild_snapshot_scan_seconds_bucket' "$OMC_ALERTS"
contains "日聚合版本槽位不完整告警" 'alert: PMDailyVersionExpectedSlotsMismatch' "$OMC_ALERTS"
contains "日聚合版本槽位不一致指标" 'omc_pm_aggregation_daily_version_expected_slots_mismatch_total' "$OMC_ALERTS"
contains "overview 展示资源计划和实际限额" '资源计划与容器实际限额' "$GRAFANA_OVERVIEW"
contains "overview 展示 Redis 聚合状态" 'Redis 聚合状态与内存水位' "$GRAFANA_OVERVIEW"
contains "overview 展示关闭并发" 'PM 窗口关闭 - 并发与领取' "$GRAFANA_OVERVIEW"
contains "overview 展示关闭尾延迟" 'PM 窗口关闭 - 尾延迟' "$GRAFANA_OVERVIEW"
contains "overview 展示重算成本" 'PM 重算快照扫描时延' "$GRAFANA_OVERVIEW"
contains "overview 展示结果替换成本" 'PM 结果替换时延' "$GRAFANA_OVERVIEW"
contains "overview 展示声明 CPU" '声明 CPU 核数' "$GRAFANA_OVERVIEW"
contains "overview 实际值按 Compose service 对齐" 'container_label_com_docker_compose_service' "$GRAFANA_OVERVIEW"
contains "overview 展示 Redis maxmemory" 'redis_memory_max_bytes' "$GRAFANA_OVERVIEW"
contains "overview 将 Redis 活动窗口拆为数量面板" 'Redis 聚合活动窗口数' "$GRAFANA_OVERVIEW"
contains "overview 展示 finalize inflight" 'omc_pm_aggregation_finalize_inflight' "$GRAFANA_OVERVIEW"
contains "overview 展示 rebuild batch" 'omc_pm_aggregation_rebuild_jobs_per_batch' "$GRAFANA_OVERVIEW"
contains "overview 展示 snapshot scan" 'omc_pm_aggregation_rebuild_snapshot_scan_seconds_bucket' "$GRAFANA_OVERVIEW"
contains "overview 展示窗口结果替换" 'omc_pm_aggregation_result_replace_seconds_bucket' "$GRAFANA_OVERVIEW"

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

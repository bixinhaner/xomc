#!/usr/bin/env bash
# =============================================================================
# OMC 启动健康校验 — 部署完成后执行（全 docker compose 部署版）
#
# 检查内容：
#   · docker compose 容器状态（business + infra + monitoring）
#   · 5 个核心健康端点（端口与 docker-compose port mapping 对齐）：
#     app /healthz（:9091）/ acs /healthz（:9095，容器 9090→宿主 9095）/
#     worker /healthz（:9092）/ app /metrics（:9091）/ 前端 SPA（:8081，
#     web 容器 nginx；:8080 是 ACS CWMP 反代不检）
#
# 用法：
#   bash healthcheck.sh                # 默认完整检查
#   bash healthcheck.sh --startup      # 安装阶段轻量启动就绪检查
#   bash healthcheck.sh --lang en      # 指定输出语言（cn 或 en）
#   bash healthcheck.sh --file-entry-smoke  # 真实上传/下载验证（会创建临时测试对象）
#   bash healthcheck.sh -h | --help    # 本帮助
#
# 参数：
#   -h, --help    本帮助
#   --lang <cn|en> 输出语言（默认读取 OMC_LANG，未设置时为 en）
#
# 退出码：0 全部通过 / 1 存在失败项
# =============================================================================
STARTUP_CHECK=0
FILE_ENTRY_SMOKE=0
while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help) sed -n '3,17p' "$0"; exit 0 ;;
    --startup) STARTUP_CHECK=1; shift ;;
    --file-entry-smoke) FILE_ENTRY_SMOKE=1; shift ;;
    --lang|--language) OMC_LANG="${2:?--lang 需要 cn 或 en}"; shift 2 ;;
    --lang=*|--language=*) OMC_LANG="${1#*=}"; shift ;;
    *) echo "  [FAIL] 未知参数：$1"; exit 1 ;;
  esac
done

set -u

OMC_LANG="${OMC_LANG:-en}"
case "$OMC_LANG" in
  cn|en) ;;
  *) echo "  [FAIL] --lang 仅支持 cn 或 en，收到：$OMC_LANG"; exit 1 ;;
esac
health_text() {
  local cn="$1" en="${2:-$1}"
  if [ "$OMC_LANG" = en ] && [ "$en" = "$cn" ]; then
    case "$cn" in
      *"容器 running") en="${cn% 容器 running} container running" ;;
      *"核心 Redis 指向 redis-core") en="${cn%% 核心 Redis*} core Redis points to redis-core" ;;
      *"PM Redis 指向 redis-pm") en="${cn%% PM Redis*} PM Redis points to redis-pm" ;;
      *"容器 OMC_PUBLIC_HOST") en="${cn%% 容器 OMC_PUBLIC_HOST} container OMC_PUBLIC_HOST" ;;
      "redis-core / redis-pm 运行实例身份不同") en="redis-core / redis-pm have distinct runtime identities" ;;
      "web ACS upstream 连接池已加载") en="web ACS upstream connection pool loaded" ;;
      "web 临时端口范围") en="web ephemeral port range" ;;
      *"不就绪"*) en="Service is not ready" ;;
      "前端 SPA"*) en="Frontend SPA${cn#前端 SPA}" ;;
      "ACS candidate"*) en="ACS candidate${cn#ACS candidate}" ;;
    esac
  fi
  if [ "$OMC_LANG" = en ]; then
    printf '%s' "$en"
  else
    printf '%s' "$cn"
  fi
}

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-omcgo}"
SKIP_MONITORING=0
if [ -f "$DEPLOY_DIR/monitoring-profile-lib.sh" ]; then
  . "$DEPLOY_DIR/monitoring-profile-lib.sh"
else
  echo "  [FAIL] 缺 $DEPLOY_DIR/monitoring-profile-lib.sh"
  exit 1
fi
if [ -f "$DEPLOY_DIR/resource-env-lib.sh" ]; then
  . "$DEPLOY_DIR/resource-env-lib.sh"
else
  echo "  [FAIL] 缺 $DEPLOY_DIR/resource-env-lib.sh"
  exit 1
fi
if [ -f "$DEPLOY_DIR/compose-env-lib.sh" ]; then
  . "$DEPLOY_DIR/compose-env-lib.sh"
else
  echo "  [FAIL] 缺 $DEPLOY_DIR/compose-env-lib.sh"
  exit 1
fi
if [ -f "$DEPLOY_DIR/redis-routing-check-lib.sh" ]; then
  . "$DEPLOY_DIR/redis-routing-check-lib.sh"
else
  echo "  [FAIL] 缺 $DEPLOY_DIR/redis-routing-check-lib.sh"
  exit 1
fi
monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING" || {
  echo "  [FAIL] 无法读取 monitoring profile"
  exit 1
}

# docker compose 命令
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "  [FAIL] docker compose 未安装"
  exit 1
fi

# 组装 -f 参数（按文件存在情况）
COMPOSE_FILES=()
for f in docker-compose.infra.yml docker-compose.app.yml docker-compose.web.yml; do
  [ -f "$DEPLOY_DIR/$f" ] && COMPOSE_FILES+=( -f "$DEPLOY_DIR/$f" )
done
if [ "$SKIP_MONITORING" = 0 ] && [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ]; then
  COMPOSE_FILES+=( -f "$DEPLOY_DIR/docker-compose.monitoring.yml" )
fi
ENV_FILES=()
[ -f "$DEPLOY_DIR/.env" ] && ENV_FILES+=( --env-file "$DEPLOY_DIR/.env" )
[ -f "$DEPLOY_DIR/resources.env" ] && ENV_FILES+=( --env-file "$DEPLOY_DIR/resources.env" )
DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${ENV_FILES[@]}" "${COMPOSE_FILES[@]}" )

# 运行时检查以旧版的快速路径为主：一次 docker ps 快照确定全部容器，后续
# 容器内探针直接使用 docker exec。每次调用 docker compose exec 都会重新解析
# 多个 compose 文件；在安装期 daemon 正忙时，逐项 timeout 还会把一轮检查
# 放大到数分钟并产生大量 [SKIP]。安装脚本已经对整轮 --startup 设置了外层
# timeout，健康检查本身保持明确的 [OK]/[FAIL] 语义。
CONTAINER_SNAPSHOT="$(docker ps \
  --filter "label=com.docker.compose.project=$COMPOSE_PROJECT" \
  --format '{{.Label "com.docker.compose.service"}}\t{{.ID}}\t{{.State}}' 2>/dev/null || true)"

ok=0; fail=0
check() {  # check <描述> <命令...>
  local desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  [OK]   $(health_text "$desc")"; ok=$((ok+1))
  else
    echo "  [FAIL] $(health_text "$desc")"; fail=$((fail+1))
  fi
}

check_value() { # check_value <描述> <期望> <实际>
  local desc="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "  [OK]   $(health_text "$desc")"; ok=$((ok+1))
  else
    if [ "$OMC_LANG" = en ]; then
      echo "  [FAIL] $(health_text "$desc") (expected: $expected; actual: $actual)"
    else
      echo "  [FAIL] $(health_text "$desc")（期望: $expected；实际: $actual）"
    fi
    fail=$((fail+1))
  fi
}

container_id() { # container_id <service>
  local svc="$1"
  printf '%s\n' "$CONTAINER_SNAPSHOT" |
    awk -F '\t' -v svc="$svc" '$1 == svc { print $2; exit }'
}

container_exec() { # container_exec <service> <command...>
  local svc="$1" cid
  shift
  cid="$(container_id "$svc")"
  [ -n "$cid" ] || return 1
  docker exec "$cid" "$@"
}

# container_running <service> —— 使用同一份快照，避免每个服务重复 docker ps。
container_running() {
  local svc="$1" state
  state="$(printf '%s\n' "$CONTAINER_SNAPSHOT" |
    awk -F '\t' -v svc="$svc" '$1 == svc { print $3; exit }')"
  [ "$state" = running ]
}

container_sysctl_equals() { # container_sysctl_equals <service> <key> <expected>
  local svc="$1" key="$2" expected="$3" actual rc
  actual="$(container_exec "$svc" sysctl -n "$key" 2>/dev/null)"
  rc=$?
  actual="$(printf '%s' "$actual" | tr -s '[:space:]' ' ' | sed 's/^ //;s/ $//')"
  [ "$actual" = "$expected" ]
}

WEB_NGINX_CONFIG=""
WEB_NGINX_CONFIG_STATE=0
WEB_HTTPS_CERT_STATE=0
web_nginx_config() {
  if [ "$WEB_NGINX_CONFIG_STATE" -eq 1 ]; then
    return 0
  fi
  [ "$WEB_NGINX_CONFIG_STATE" -eq 2 ] && return 1
  WEB_NGINX_CONFIG="$(container_exec web nginx -T 2>&1)" || {
    WEB_NGINX_CONFIG_STATE=2
    return 1
  }
  WEB_NGINX_CONFIG_STATE=1
}

web_acs_upstream_pool_loaded() {
  web_nginx_config || return 1
  printf '%s\n' "$WEB_NGINX_CONFIG" | grep -Fq 'server acs:7557 resolve;'
  printf '%s\n' "$WEB_NGINX_CONFIG" | grep -Fq 'keepalive 4096;'
  printf '%s\n' "$WEB_NGINX_CONFIG" | grep -Fq 'proxy_pass http://acs_backend;'
}

web_https_file_entry_loaded() {
  local rendered_flat
  web_nginx_config || return 1
  rendered_flat="$(printf '%s\n' "$WEB_NGINX_CONFIG" | tr -s '[:space:]' ' ')"
  printf '%s\n' "$rendered_flat" | grep -Fq 'listen 8443 ssl;' &&
    printf '%s\n' "$rendered_flat" | grep -Fq 'ssl_certificate /etc/nginx/cert/cert.pem;' &&
    printf '%s\n' "$rendered_flat" | grep -Fq 'ssl_certificate_key /etc/nginx/cert/key.pem;' &&
    printf '%s\n' "$rendered_flat" | grep -Fq 'proxy_set_header X-Forwarded-Proto https;'
}

web_https_file_entry_has_cert() {
  if [ "$WEB_HTTPS_CERT_STATE" -eq 1 ]; then
    return 0
  fi
  if [ "$WEB_HTTPS_CERT_STATE" -eq 2 ]; then
    return 1
  fi
  if container_exec web sh -c 'test -r /etc/nginx/cert/cert.pem && test -r /etc/nginx/cert/key.pem'; then
    WEB_HTTPS_CERT_STATE=1
    return 0
  fi
  WEB_HTTPS_CERT_STATE=2
  return 1
}

check_web_https_file_entry() {
  local mode="${1:-startup}"
  if web_https_file_entry_has_cert; then
    check "$(health_text 'HTTPS ACS /healthz (:8443)' 'HTTPS ACS /healthz (:8443)')" https_file_entry_status_is /healthz 200
    check "$(health_text 'HTTPS ACS service 路径到达 ACS (:8443)' 'HTTPS ACS service path reaches ACS (:8443)')" https_file_entry_status_is /smallcell/AcsService 405
    if [ "$mode" = full ]; then
      check "$(health_text 'web HTTPS 文件入口已加载' 'web HTTPS file entry loaded')" web_https_file_entry_loaded
    else
      check "$(health_text 'HTTPS 文件上传入口 TLS 可达 ACS (:8443)' 'HTTPS file upload TLS reaches ACS (:8443)')" https_file_entry_status_is /smallcell/FileUploadService 405
      check "$(health_text 'HTTPS 文件下载入口 TLS 可达 ACS (:8443)' 'HTTPS file download TLS reaches ACS (:8443)')" https_file_entry_status_is /smallcell/FileDownloadService/__healthcheck__/missing 404
    fi
  else
    check "$(health_text 'web HTTPS 文件入口证书已安装' 'web HTTPS file-entry certificate installed')" web_https_file_entry_has_cert
  fi
}

https_file_entry_status_is() { # https_file_entry_status_is <path> <expected_status>
  local path="$1" expected="$2" status
  status="$(curl -k -sS --max-time 3 -o /dev/null -w '%{http_code}' "https://127.0.0.1:8443${path}")" || return 1
  [ "$status" = "$expected" ]
}

http_ready_with_retry() {
  local url="$1"
  curl -fsS --connect-timeout 1 --max-time 6 \
    --retry 4 --retry-delay 1 --retry-max-time 20 --retry-connrefused \
    "$url" >/dev/null
}

acs_service_ready() {
  local service="$1" cid ip
  cid="$(container_id "$service")"
  [ -n "$cid" ] || return 1
  ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$cid" 2>/dev/null)"
  [ -n "$ip" ] || return 1
  http_ready_with_retry "http://${ip}:7557/readyz"
}

redis_instance_run_id() {
  local service="$1" out
  out="$(container_exec "$service" redis-cli --raw INFO server 2>/dev/null)" || return 1
  printf '%s' "$out" | awk -F: '$1 == "run_id" { gsub(/\r/, "", $2); print $2; exit }'
}

redis_instances_distinct() {
  local core_id pm_id
  core_id="$(redis_instance_run_id redis-core)" || return 1
  pm_id="$(redis_instance_run_id redis-pm)" || return 1
  [ -n "$core_id" ] && [ -n "$pm_id" ] && [ "$core_id" != "$pm_id" ]
}

check_redis_routing_config() {
  redis_routing_configs_valid /opt/omc/etc
}

redis_config_values() { # redis_config_values <service>
  local service="$1"
  container_exec "$service" redis-cli --raw CONFIG GET maxmemory maxmemory-policy appendonly 2>/dev/null
}

redis_config_value() { # redis_config_value <output> <setting>
  local output="$1" setting="$2"
  printf '%s\n' "$output" | awk -v key="$setting" '$0 == key { if (getline value > 0) { print value; exit } }'
}

pg_setting_values() { # pg_setting_values <service> <user>
  local service="$1" user="$2"
  container_exec "$service" psql -U "$user" -d postgres -Atc \
    'SHOW shared_buffers; SHOW work_mem; SHOW max_connections' 2>/dev/null
}

pg_setting_value() { # pg_setting_value <output> <setting>
  local output="$1" setting="$2" line
  case "$setting" in
    shared_buffers) line=1 ;;
    work_mem) line=2 ;;
    max_connections) line=3 ;;
  esac
  printf '%s\n' "$output" | sed -n "${line}p" | tr -d '[:space:]'
}

# 安装阶段的启动就绪检查必须在完整审计之前结束。完整检查中的 nginx -T、
# 容器 sysctl、Compose config、资源限额和数据库参数可能因初始化负载变慢，
# 不应阻塞“服务是否已经能接收请求”的判定。
if [ "$STARTUP_CHECK" = 1 ]; then
  echo "== $(health_text '启动核心服务检查' 'Core service startup check') =="
  for svc in app acs worker; do
    check "$(health_text "$svc 容器 running" "$svc container running")" container_running "$svc"
  done
  check "$(health_text 'acs-candidate 容器 running' 'acs-candidate container running')" container_running acs-candidate
  for svc in postgres postgres-tsdb redis-core redis-pm nats minio; do
    check "$(health_text "$svc 容器 running" "$svc container running")" container_running "$svc"
  done
  if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
    check "$(health_text 'web 容器 running' 'web container running')" container_running web
  fi
  check "app /healthz (:9091)" curl -fsS --max-time 3 http://127.0.0.1:9091/healthz
  check "acs /healthz (:9095)" curl -fsS --max-time 3 http://127.0.0.1:9095/healthz
  check "worker /healthz (:9092)" curl -fsS --max-time 3 http://127.0.0.1:9092/healthz
  check "app /metrics (:9091)" curl -fsS --max-time 3 http://127.0.0.1:9091/metrics
  check "$(health_text '前端 SPA (:8081)' 'Frontend SPA (:8081)')" curl -fsS --max-time 3 http://127.0.0.1:8081/ -o /dev/null
  if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
    check_web_https_file_entry startup
  fi
  echo
  echo "$(health_text '启动检查已跳过 Redis 路由、实例身份和 ACS candidate /readyz 深审计；完整 healthcheck 将在部署后执行。' 'Startup check skips deep Redis routing, instance identity, and ACS candidate /readyz audits; the full healthcheck runs after deployment.')"
  echo "$(health_text "启动检查结果：通过 $ok 项，失败 $fail 项" "Startup check result: $ok passed, $fail failed")"
  [ "$fail" -eq 0 ] || { echo "$(health_text '启动核心服务尚未就绪。' 'Core services are not ready.')"; exit 1; }
  echo "$(health_text '启动核心服务已就绪。' 'Core services are ready.')"
  exit 0
fi

echo "== $(health_text 'docker compose 业务容器' 'Docker Compose business containers') =="
for svc in app acs worker; do
  check "$svc 容器 running" container_running "$svc"
done
check "acs-candidate 容器 running" container_running acs-candidate
check "acs-candidate /readyz" acs_service_ready acs-candidate

echo "== $(health_text 'docker compose 基础设施容器' 'Docker Compose infrastructure containers') =="
for svc in postgres postgres-tsdb redis-core redis-pm nats minio; do
  check "$svc 容器 running" container_running "$svc"
done

echo "== $(health_text 'Redis 业务路由隔离' 'Redis business routing isolation') =="
for config_file in app.prod.yaml worker.prod.yaml; do
  check "$config_file 核心 Redis 指向 redis-core" yaml_top_level_section_has_address "/opt/omc/etc/$config_file" redis redis-core:6379
  check "$config_file PM Redis 指向 redis-pm" yaml_top_level_section_has_address "/opt/omc/etc/$config_file" pm_redis redis-pm:6379
done
check "redis-core / redis-pm 运行实例身份不同" redis_instances_distinct

if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
  echo "== $(health_text 'docker compose web 容器' 'Docker Compose web container') =="
  check "web 容器 running" container_running web
  check "web ACS upstream 连接池已加载" web_acs_upstream_pool_loaded
  check_web_https_file_entry full
  check "web 临时端口范围" container_sysctl_equals web net.ipv4.ip_local_port_range "10240 65535"
fi

echo "== $(health_text 'ACS 高并发网络参数' 'ACS high-concurrency network parameters') =="
check "ACS accept backlog" container_sysctl_equals acs net.core.somaxconn "32768"
check "ACS SYN backlog" container_sysctl_equals acs net.ipv4.tcp_max_syn_backlog "32768"
check "ACS candidate accept backlog" container_sysctl_equals acs-candidate net.core.somaxconn "32768"
check "ACS candidate SYN backlog" container_sysctl_equals acs-candidate net.ipv4.tcp_max_syn_backlog "32768"

if [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ] && [ "$SKIP_MONITORING" = 0 ]; then
  echo "== $(health_text 'docker compose 监控容器' 'Docker Compose monitoring containers') =="
  for svc in prometheus alertmanager grafana loki tempo otelcol nats-exporter nginx-exporter node-exporter cadvisor; do
    check "$svc 容器 running" container_running "$svc"
  done
  # otelcol-contrib 是 distroless 镜像，不能假设容器内有 shell/curl/wget。
  # monitoring compose 将 health_check extension 仅映射到宿主回环供外部探测。
  check "otelcol health extension (:13133)" http_ready_with_retry http://127.0.0.1:13133/
fi

echo "== $(health_text '服务健康端点' 'Service health endpoints') =="
# /healthz + /metrics 都在 metrics 端口上注册（internal/core/components/monitor/metrics.go）。
# 业务进程主 HTTP（app:8081 / acs SOAP:7547）不直接暴露 /healthz —— 用 metrics 端口检健康。
# 端口与 compose port mapping 对齐：app/worker 容器 == 宿主；acs 容器 9090 → 宿主 9095。
check "app    /healthz (:9091)"  curl -fsS --max-time 3 http://127.0.0.1:9091/healthz
check "acs    /healthz (:9095)"  curl -fsS --max-time 3 http://127.0.0.1:9095/healthz
check "worker /healthz (:9092)"  curl -fsS --max-time 3 http://127.0.0.1:9092/healthz
check "app    /metrics (:9091)"  curl -fsS --max-time 3 http://127.0.0.1:9091/metrics
# 前端 SPA：web 容器 nginx :8081 served（:8080 是 ACS CWMP 反代，GET / 不响应，不检）。
check "前端 SPA (:8081)"          curl -fsS --max-time 3 http://127.0.0.1:8081/ -o /dev/null
if [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]; then
  check_web_https_file_entry startup
fi

if [ "$FILE_ENTRY_SMOKE" = 1 ]; then
  echo "== $(health_text 'HTTPS 文件入口真实上传下载' 'HTTPS file-entry real upload/download smoke') =="
  check "HTTP/HTTPS 文件上传下载闭环" bash "$DEPLOY_DIR/smoke-nginx-https-file-entry.sh"
fi

echo "== $(health_text '基站可达地址实际值核对' 'Verify effective base-station reachable address') =="
effective_public_host="$(deploy_env_effective_value OMC_PUBLIC_HOST "$DEPLOY_DIR/.env" "$DEPLOY_DIR/resources.env" 2>/dev/null || true)"
if deploy_env_public_host_valid "$effective_public_host"; then
  echo "  [OK]   $(health_text 'OMC_PUBLIC_HOST 有效' 'OMC_PUBLIC_HOST is valid')"; ok=$((ok+1))
else
  echo "  [FAIL] $(health_text "OMC_PUBLIC_HOST 必须配置为基站可达主机（当前: ${effective_public_host:-<空>}）" "OMC_PUBLIC_HOST must be reachable by base stations (current: ${effective_public_host:-<empty>})")"; fail=$((fail+1))
fi
container_env_value() { # container_env_value <service> <key>
  local svc="$1" key="$2" cid inspect_output
  cid="$(container_id "$svc")"
  [ -n "$cid" ] || return 1
  inspect_output="$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$cid" 2>/dev/null)" || return 1
  printf '%s\n' "$inspect_output" |
    awk -F= -v key="$key" '$1 == key { print substr($0, length(key) + 2); exit }'
}
for svc in app acs acs-candidate worker; do
  check_value "$svc 容器 OMC_PUBLIC_HOST" "$effective_public_host" "$(container_env_value "$svc" OMC_PUBLIC_HOST)"
done

# resources.env 存在时，必须同时证明「文件 → compose 渲染 → 容器/进程实际值」没有漂移。
# 未使用规划器的历史部署仍允许使用 compose 默认值；但一旦有该文件，残缺或不一致绝不静默通过。
if [ -f "$DEPLOY_DIR/resources.env" ]; then
  echo "== $(health_text 'resources.env 实际值核对' 'Verify effective resources.env values') =="
  if ! resource_env_validate "$DEPLOY_DIR/resources.env"; then
    echo "  [FAIL] $(health_text 'resources.env 完整资源契约' 'Complete resources.env contract')"; fail=$((fail+1))
  else
    echo "  [OK]   $(health_text 'resources.env 完整资源契约' 'Complete resources.env contract')"; ok=$((ok+1))
    COMPOSE_RENDERED="$("${DC[@]}" config 2>/dev/null || true)"

    compose_limit() { # compose_limit <服务> <cpus|memory>
      local svc="$1" field="$2"
      printf '%s\n' "$COMPOSE_RENDERED" | awk -v svc="$svc" -v field="$field" '
        $0 ~ "^  " svc ":" { in_service=1; in_limits=0; next }
        in_service && $0 ~ /^  [A-Za-z0-9_-]+:$/ { exit }
        in_service && $0 ~ /^[[:space:]]+limits:$/ { in_limits=1; next }
        in_service && $0 ~ /^[[:space:]]+reservations:$/ { in_limits=0 }
        in_service && in_limits && $0 ~ "^[[:space:]]+" field ":" {
          sub("^[[:space:]]+" field ":[[:space:]]*", "")
          gsub(/"/, "")
          print
          exit
        }
      '
    }
    resource_bytes() {
      local mib
      mib="$(resource_env_memory_mib "$1")" || return 1
      awk -v mib="$mib" 'BEGIN { printf "%.0f", mib * 1024 * 1024 }'
    }
    check_service_limits() { # service key-prefix
      local svc="$1" prefix="$2" cid expected_cpu expected_mem expected_nano expected_bytes actual_nano actual_bytes rendered_cpu rendered_mem rendered_bytes
      expected_cpu="$(resource_env_get "$DEPLOY_DIR/resources.env" "${prefix}_CPUS")"
      expected_mem="$(resource_env_get "$DEPLOY_DIR/resources.env" "${prefix}_MEM")"
      rendered_cpu="$(compose_limit "$svc" cpus)"
      rendered_mem="$(compose_limit "$svc" memory)"
      expected_bytes="$(resource_bytes "$expected_mem")"
      if [[ "$rendered_mem" =~ ^[0-9]+$ ]]; then
        rendered_bytes="$rendered_mem"
      else
        rendered_bytes="$(resource_bytes "$rendered_mem" 2>/dev/null || true)"
      fi
      check_value "$svc compose cpus" "$expected_cpu" "$rendered_cpu"
      check_value "$svc compose memory (bytes)" "$expected_bytes" "$rendered_bytes"
      cid="$(container_id "$svc")"
      if [ -z "$cid" ]; then
        echo "  [FAIL] $(health_text "$svc docker inspect（未找到容器）" "$svc docker inspect (container not found)")"; fail=$((fail+1)); return
      fi
      read -r actual_nano actual_bytes <<EOF
$(docker inspect -f '{{.HostConfig.NanoCpus}} {{.HostConfig.Memory}}' "$cid" 2>/dev/null)
EOF
      expected_nano="$(awk -v cpu="$expected_cpu" 'BEGIN { printf "%.0f", cpu * 1000000000 }')"
      check_value "$svc docker inspect NanoCpus" "$expected_nano" "$actual_nano"
      check_value "$svc docker inspect Memory" "$expected_bytes" "$actual_bytes"
    }
    for spec in 'app APP' 'acs ACS' 'acs-candidate ACS' 'worker WORKER' 'postgres POSTGRES' 'postgres-tsdb TSDB' 'redis-core REDIS_CORE' 'redis-pm REDIS_PM' 'nats NATS' 'minio MINIO'; do
      check_service_limits ${spec}
    done
    [ -f "$DEPLOY_DIR/docker-compose.web.yml" ] && check_service_limits web WEB

    check_gomaxprocs() { # name port resource key
      local name="$1" port="$2" key="$3" expected actual
      expected="$(resource_env_get "$DEPLOY_DIR/resources.env" "$key")"
      actual="$(curl -fsS --max-time 10 "http://127.0.0.1:$port/metrics" 2>/dev/null | awk '/^go_sched_gomaxprocs_threads / { print $2; exit }')"
      check_value "$name go_sched_gomaxprocs_threads" "$expected" "$actual"
    }
    check_gomaxprocs app 9091 APP_GOMAXPROCS
    check_gomaxprocs acs 9095 ACS_GOMAXPROCS
    check_gomaxprocs worker 9092 WORKER_GOMAXPROCS

    for redis_spec in 'redis-core REDIS_CORE' 'redis-pm REDIS_PM'; do
      read -r redis_svc redis_prefix <<<"$redis_spec"
      redis_expected="$(resource_bytes "$(resource_env_get "$DEPLOY_DIR/resources.env" "${redis_prefix}_MAXMEMORY")")"
      redis_output="$(redis_config_values "$redis_svc" 2>/dev/null || true)"
      for redis_setting in maxmemory maxmemory-policy appendonly; do
        case "$redis_setting" in
          maxmemory) redis_expected_value="$redis_expected" ;;
          maxmemory-policy) redis_expected_value="noeviction" ;;
          appendonly) redis_expected_value="yes" ;;
        esac
        redis_actual="$(redis_config_value "$redis_output" "$redis_setting")"
        check_value "$redis_svc CONFIG GET $redis_setting" "$redis_expected_value" "$redis_actual"
      done
    done

    deploy_env_get() { awk -F= -v key="$2" '$1 == key { print substr($0, length(key)+2); exit }' "$1"; }
    pg_user="$(deploy_env_get "$DEPLOY_DIR/.env" POSTGRES_USER)"
    tsdb_user="$(deploy_env_get "$DEPLOY_DIR/.env" POSTGRES_TSDB_USER)"
    pg_check_settings() { # service user prefix
      local svc="$1" user="$2" prefix="$3" expected actual setting pg_output
      pg_output="$(pg_setting_values "$svc" "$user" 2>/dev/null || true)"
      for setting in shared_buffers work_mem max_connections; do
        case "$setting" in
          shared_buffers) expected="$(resource_env_get "$DEPLOY_DIR/resources.env" "${prefix}_SHARED_BUFFERS")" ;;
          work_mem) expected="$(resource_env_get "$DEPLOY_DIR/resources.env" "${prefix}_WORK_MEM")" ;;
          max_connections) expected="$(resource_env_get "$DEPLOY_DIR/resources.env" "${prefix}_MAX_CONNECTIONS")" ;;
        esac
        actual="$(pg_setting_value "$pg_output" "$setting")"
        if [ "$setting" = "max_connections" ]; then
          check_value "$svc SHOW $setting" "$expected" "$actual"
        else
          expected_mib="$(resource_env_memory_mib "$expected")"
          actual_mib="$(resource_env_memory_mib "$actual" 2>/dev/null || true)"
          check_value "$svc SHOW $setting" "$expected_mib" "$actual_mib"
        fi
      done
    }
    pg_check_settings postgres "$pg_user" PG
    pg_check_settings postgres-tsdb "$tsdb_user" TSDB
  fi
fi

if [ "$fail" -gt 0 ]; then
  echo
  echo "$(health_text 'compose ps 详情：' 'Compose ps details:')"
  "${DC[@]}" ps 2>/dev/null || echo "  (无法读取 compose 状态)"
fi

echo
echo "$(health_text "结果：通过 $ok 项，失败 $fail 项" "Result: $ok passed, $fail failed")"
[ "$fail" -eq 0 ] || { echo "$(health_text '存在失败项，参见部署方案故障排查章节。' 'Failures found; see the deployment troubleshooting section.')"; exit 1; }
echo "$(health_text '校验通过。' 'Healthcheck passed.')"

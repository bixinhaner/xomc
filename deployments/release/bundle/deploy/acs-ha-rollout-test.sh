#!/usr/bin/env bash
set -uo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$DEPLOY_DIR/../../../.." && pwd)"
APP_COMPOSE="$DEPLOY_DIR/docker-compose.app.yml"
INSTALL="$DEPLOY_DIR/install.sh"
HEALTHCHECK="$DEPLOY_DIR/healthcheck.sh"
PROMETHEUS="$REPO_ROOT/deployments/monitoring/prometheus.yml"
ALERTS="$REPO_ROOT/deployments/monitoring/alerts/omc-rules.yml"
RESOURCE_METRICS="$DEPLOY_DIR/resource-plan-metrics.sh"
RESOURCE_PLANNER="$DEPLOY_DIR/plan-resources.sh"
OTELCOL="$REPO_ROOT/deployments/monitoring/otelcol/config.yaml"

pass=0
fail=0

contains() {
  local description="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then
    pass=$((pass + 1))
  else
    echo "FAIL: $description: $file 未包含 [$pattern]" >&2
    fail=$((fail + 1))
  fi
}

echo "── ACS 双实例发布契约 ──"
contains "候选 ACS 服务存在" "acs-candidate:" "$APP_COMPOSE"
contains "候选 ACS 共享正式 DNS 别名" "- acs" "$APP_COMPOSE"
contains "候选 ACS 不发布宿主业务端口" "ACS_CANDIDATE_INTERNAL_ONLY" "$APP_COMPOSE"
contains "候选 ACS 使用独立日志目录" "/opt/omc/run/logs/acs-candidate:/run/logs/acs" "$APP_COMPOSE"
contains "安装创建候选 ACS 日志目录" '"$OMC_ROOT/run/logs/acs-candidate"' "$INSTALL"
contains "安装先预热候选 ACS" "acs_ha_prepare_candidate" "$INSTALL"
contains "首轮升级先校验旧 web 动态 upstream" "web_acs_dynamic_upstream_loaded" "$INSTALL"
contains "动态 ACS server 校验失败必须阻断" "grep -Fq 'server acs:7557 resolve;' || return 1" "$INSTALL"
contains "动态 ACS zone 校验失败必须阻断" "grep -Fq 'zone acs_backend' || return 1" "$INSTALL"
contains "旧 web 不支持动态 upstream 时先刷新入口" "先刷新 web 动态 ACS upstream" "$INSTALL"
contains "候选 ACS 就绪后才允许替换正式实例" "acs_ha_wait_ready acs-candidate" "$INSTALL"
contains "存量发布单独替换正式 ACS" '"${DC[@]}" up -d --no-deps acs' "$INSTALL"
contains "正式 ACS 就绪后才更新其余服务" "acs_ha_wait_ready acs" "$INSTALL"
contains "存量发布的其余服务显式排除 ACS 依赖" '"${DC[@]}" up -d --no-deps "${remaining_services[@]}"' "$INSTALL"
contains "存量发布覆盖全部监控 exporter" "nats-exporter nginx-exporter node-exporter cadvisor" "$INSTALL"
contains "等待 Nginx 动态 DNS 纳入候选实例" "sleep 12" "$INSTALL"
contains "候选探针直达容器 IP 的 CWMP 业务端口" '"http://${service_ip}:7557/readyz"' "$INSTALL"
if [ "$(grep -Fc 'acs_ha_wait_ready acs-candidate' "$INSTALL")" -ge 2 ]; then
  pass=$((pass + 1))
else
  echo "FAIL: DNS 等待后必须再次验证候选 ACS 业务 readiness" >&2
  fail=$((fail + 1))
fi
contains "存量 --skip-web 升级被显式拒绝" "--skip-web 不支持存量 ACS 无损升级" "$INSTALL"
contains "健康检查覆盖候选 ACS" 'check "acs-candidate 容器 running"' "$HEALTHCHECK"
contains "健康检查验证候选 ACS readiness" 'acs_service_ready acs-candidate' "$HEALTHCHECK"
contains "Prometheus 独立抓取候选 ACS" '"acs-candidate:9090"' "$PROMETHEUS"
contains "Prometheus 独立抓取正式 ACS" '"acs-primary:9090"' "$PROMETHEUS"
contains "单 ACS 副本失效仅告警降级" "alert: OMCACSReplicaDown" "$ALERTS"
contains "全部 ACS 副本失效才判主链路中断" 'sum(up{job="omc-acs", instance=~"acs-primary:9090|acs-candidate:9090"}) < 1' "$ALERTS"
contains "副本告警忽略旧抓取目标残留" 'up{job="omc-acs", instance=~"acs-primary:9090|acs-candidate:9090"} == 0' "$ALERTS"
contains "资源计划指标覆盖候选 ACS" "'acs-candidate ACS'" "$RESOURCE_METRICS"
contains "资源规划总额计入候选 ACS" "ALLOC_SUM + ACS_MEM" "$RESOURCE_PLANNER"
contains "OTel 日志标记 primary 实例" 'value: primary' "$OTELCOL"
contains "OTel 日志标记 candidate 实例" 'value: candidate' "$OTELCOL"
contains "OTel 日志实例提升为 Loki 标签" 'resource["loki.resource.labels"]' "$OTELCOL"

echo "结果：通过 $pass 项，失败 $fail 项"
[ "$fail" -eq 0 ]

#!/usr/bin/env bash

# Regression guard for F-01/F-02/F-03/F-04/F-05/F-06/F-07/F-08/F-09/F-10/F-11/F-12/F-13/F-14/F-15/F-16/F-17: cAdvisor must export a bounded Docker
# Compose service label so Prometheus can expose CPU and memory usage by
# app/acs/worker.
set -uo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
prometheus_config="$repo_root/deployments/monitoring/prometheus.yml"
dashboard="$repo_root/deployments/monitoring/grafana/dashboards/omc-overview.json"
resources_dashboard="$repo_root/deployments/monitoring/grafana/dashboards/omc-resources.json"
infra_dashboard="$repo_root/deployments/monitoring/grafana/dashboards/omc-infra.json"
nginx_dashboard="$repo_root/deployments/monitoring/grafana/dashboards/nginx-host-overview.json"
alerts_config="$repo_root/deployments/monitoring/alerts/host-container-alerts.yml"
compose_files=(
  "$repo_root/deployments/docker/docker-compose.yml"
  "$repo_root/deployments/release/bundle/deploy/docker-compose.monitoring.yml"
)

failures=0
fail() {
  printf 'ERROR: %s\n' "$*" >&2
  failures=$((failures + 1))
}

for compose in "${compose_files[@]}"; do
  if [[ ! -f "$compose" ]]; then
    fail "missing compose file: ${compose#$repo_root/}"
    continue
  fi

  cadvisor_block=$(sed -n '/^  cadvisor:/,/^  [^ ]/p' "$compose")
  if ! grep -Fq -- '--store_container_labels=false' <<<"$cadvisor_block"; then
    fail "${compose#$repo_root/}: cAdvisor must disable unbounded container labels"
  fi
  if ! grep -Fq -- '--whitelisted_container_labels=com.docker.compose.service' <<<"$cadvisor_block"; then
    fail "${compose#$repo_root/}: cAdvisor must whitelist com.docker.compose.service"
  fi
done

grep -Fq 'source_labels: [container_label_com_docker_compose_service]' "$prometheus_config" || \
  fail "prometheus.yml: missing Compose service source label relabel"
grep -Fq 'target_label: service' "$prometheus_config" || \
  fail "prometheus.yml: missing service target relabel"

if command -v jq >/dev/null 2>&1; then
  jq -e '.panels[] | select(.title == "CPU Usage by Service") | .targets[] | .expr | contains("service=~\"app|acs|worker\"")' "$dashboard" >/dev/null || \
    fail "omc-overview.json: CPU Usage by Service must keep the bounded service selector"
  jq -e '.panels[] | select(.title == "CPU Usage by Service") | .targets[] | .expr | contains("container_cpu_usage_seconds_total{service=~\"app|acs|worker\"}")' "$dashboard" >/dev/null || \
    fail "omc-overview.json: CPU Usage by Service must use cAdvisor CPU metrics"
  jq -e '.panels[] | select(.title == "Memory Usage by Service") | .targets[] | .expr | contains("sum by (service) (container_memory_working_set_bytes{service=~\"app|acs|worker\"})")' "$dashboard" >/dev/null || \
    fail "omc-overview.json: Memory Usage by Service must aggregate working-set bytes by OMC service"
  jq -e '.panels[] | select(.title == "Memory Usage by Service") | .fieldConfig.defaults.unit == "bytes"' "$dashboard" >/dev/null || \
    fail "omc-overview.json: Memory Usage by Service must use bytes as its unit"
  jq -e '.panels[] | select(.title == "最高 CPU 饱和度（占限额）") | .targets[] | .expr | contains("max(") and contains("sum by (service) ((container_spec_cpu_quota{service=~\"app|acs|worker\"} > 0) / container_spec_cpu_period{service=~\"app|acs|worker\"})")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: highest CPU saturation must aggregate valid per-container quota/period by service"
  jq -e '.panels[] | select(.title == "最高 CPU 饱和度（占限额）") | .fieldConfig.defaults.unit == "percentunit"' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: highest CPU saturation must use percentunit"
  jq -e '.panels[] | select(.title == "最高 内存饱和度（占限额）") | .targets[] | .expr | contains("max(") and contains("(container_spec_memory_limit_bytes{service=~\"app|acs|worker\"} > 0) and (container_spec_memory_limit_bytes{service=~\"app|acs|worker\"} < 64e9)")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: highest memory saturation must filter positive per-container limits below the host sentinel"
  jq -e '.panels[] | select(.title == "最高 内存饱和度（占限额）") | .fieldConfig.defaults.unit == "percentunit"' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: highest memory saturation must use percentunit"
  jq -e '.panels[] | select(.title == "容器重启次数（当前范围）") | .targets[] | .expr | contains("sum(changes(container_start_time_seconds{service=~\"app|acs|worker|postgres|postgres-tsdb|redis|nats|minio|web|prometheus|grafana|loki|tempo|otelcol|alertmanager|cadvisor|node-exporter|nginx-exporter|nats-exporter\"}[$__range]))")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: restart count must cover OMC, dependencies, monitoring, and exporter services"
  jq -e '.panels[] | select(.title == "容器 CPU 饱和度（占限额 %，按容器）") | .targets[] | .expr | contains("sum by (service) (rate(container_cpu_usage_seconds_total{service=~\"app|acs|worker\"}[$__rate_interval]))") and contains("sum by (service) ((container_spec_cpu_quota{service=~\"app|acs|worker\"} > 0) / container_spec_cpu_period{service=~\"app|acs|worker\"})")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: CPU saturation panel must align usage and per-container quota/period by service"
  jq -e '.panels[] | select(.title == "容器 CPU 使用 vs 限额（核）") | .targets[0].expr | contains("sum by (service) (rate(container_cpu_usage_seconds_total{service=~\"app|acs|worker\"}[$__rate_interval]))")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: CPU usage-vs-limit panel must keep the service usage query"
  jq -e '.panels[] | select(.title == "容器 CPU 使用 vs 限额（核）") | .targets[1].expr | contains("sum by (service) ((container_spec_cpu_quota{service=~\"app|acs|worker\"} > 0) / container_spec_cpu_period{service=~\"app|acs|worker\"})")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: CPU usage-vs-limit panel must calculate per-container quota/period before service aggregation"
  jq -e '.panels[] | select(.title == "容器内存饱和度（占限额 %，按容器）") | .targets[] | .expr | contains("sum by (service) (container_memory_working_set_bytes{service=~\"app|acs|worker\"})") and contains("(container_spec_memory_limit_bytes{service=~\"app|acs|worker\"} > 0) and (container_spec_memory_limit_bytes{service=~\"app|acs|worker\"} < 64e9)")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: memory saturation panel must filter valid per-container limits"
  jq -e '.panels[] | select(.title == "容器 CPU CFS 限流比（撞 CPU 限额）") | .targets[] | .expr | contains("container_cpu_cfs_throttled_periods_total{service=~\"app|acs|worker\"}") and contains("container_cpu_cfs_periods_total{service=~\"app|acs|worker\"}")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: CFS throttling panel must use bounded service selectors"
  jq -e '.panels[] | select(.title == "容器网络收发（按容器）") | .targets[] | .expr | contains("interface!~\"lo|veth.*|docker.*|cni.*|flannel.*|br-.*\"")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: network panel must exclude loopback and virtual interfaces"
  jq -e '.panels[] | select(.title == "容器 OOM 事件（当前范围）") | .targets[] | .expr | contains("container_oom_events_total{service=~\"app|acs|worker\"}")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: OOM panel must use the bounded service selector"
  jq -e '.panels[] | select(.title == "容器 OOM 事件（当前范围）") | .fieldConfig.defaults.noValue == "采集器未提供"' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: OOM panel must distinguish missing cAdvisor metric from zero"
  jq -e '.panels[] | select(.title == "容器块 IO 读写（按服务）") | .targets[] | select(.expr | contains("container_fs_reads_bytes_total{service=~\"app|acs|worker\",")) | .expr | contains("device!=\"\"")' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: block-IO panel must filter device samples and use bounded services"
  jq -e '.panels[] | select(.title == "容器块 IO 读写（按服务）") | .fieldConfig.defaults.noValue == "采集器未提供"' "$resources_dashboard" >/dev/null || \
    fail "omc-resources.json: block-IO panel must distinguish missing metrics from zero"
  jq -e '.panels[] | select(.title == "依赖容器 CPU 饱和度（占限额 %，按服务）") | .targets[] | .expr | contains("sum by (service) ((container_spec_cpu_quota{service=~\"postgres|postgres-tsdb|redis|nats|minio|web\"} > 0) / container_spec_cpu_period{service=~\"postgres|postgres-tsdb|redis|nats|minio|web\"})")' "$infra_dashboard" >/dev/null || \
    fail "omc-infra.json: dependency CPU saturation must calculate per-container quota/period"
  jq -e '.panels[] | select(.title == "依赖容器内存饱和度（占限额 %，按服务）") | .targets[] | .expr | contains("sum by (service) ((container_spec_memory_limit_bytes{service=~\"postgres|postgres-tsdb|redis|nats|minio|web\"} > 0) and (container_spec_memory_limit_bytes{service=~\"postgres|postgres-tsdb|redis|nats|minio|web\"} < 64e9))")' "$infra_dashboard" >/dev/null || \
    fail "omc-infra.json: dependency memory saturation must filter valid per-container limits"
  jq -e '.panels[] | select(.title == "监控栈容器内存（按服务）") | .targets[] | .expr | contains("container_label_com_docker_compose_service=~\"prometheus|grafana|loki|tempo|otelcol|alertmanager|cadvisor\"") and contains("container_id=~\".*/docker-[^/]+\\\\.scope\"")' "$infra_dashboard" >/dev/null || \
    fail "omc-infra.json: monitoring memory panel must exclude host cgroups and unlabelled containers"
  jq -e '.panels[] | select(.title == "容器 CPU（按容器 ID）") | .targets[] | .expr | contains("container_id=~\".*/docker-[^/]+\\\\.scope\"")' "$nginx_dashboard" >/dev/null || \
    fail "nginx-host-overview.json: container CPU panel must match systemd Docker scope IDs"
  jq -e '.panels[] | select(.title == "容器内存（按容器 ID）") | .targets[] | .expr | contains("container_id=~\".*/docker-[^/]+\\\\.scope\"")' "$nginx_dashboard" >/dev/null || \
    fail "nginx-host-overview.json: container memory panel must match systemd Docker scope IDs"
else
  grep -Fq 'container_cpu_usage_seconds_total{service=~"app|acs|worker"}' "$dashboard" || \
    fail "omc-overview.json: CPU Usage by Service must keep the bounded service selector"
  grep -Fq 'container_memory_working_set_bytes{service=~"app|acs|worker"}' "$dashboard" || \
    fail "omc-overview.json: Memory Usage by Service must keep the bounded service selector"
  grep -Fq 'container_spec_cpu_quota{service=~"app|acs|worker"} > 0) / container_spec_cpu_period' "$resources_dashboard" || \
    fail "omc-resources.json: highest CPU saturation must aggregate valid per-container quota/period by service"
  grep -Fq 'container_spec_memory_limit_bytes{service=~"app|acs|worker"} < 64e9' "$resources_dashboard" || \
    fail "omc-resources.json: highest memory saturation must filter limits below the host sentinel"
  grep -Fq 'container_start_time_seconds{service=~"app|acs|worker|postgres|postgres-tsdb|redis|nats|minio|web|prometheus|grafana|loki|tempo|otelcol|alertmanager|cadvisor|node-exporter|nginx-exporter|nats-exporter"}' "$resources_dashboard" || \
    fail "omc-resources.json: restart count must cover OMC, dependencies, monitoring, and exporter services"
  grep -Fq 'sum by (service) (rate(container_cpu_usage_seconds_total{service=~"app|acs|worker"}[$__rate_interval]))' "$resources_dashboard" || \
    fail "omc-resources.json: CPU saturation panel must use service aggregation"
  grep -Fq 'sum by (service) ((container_spec_cpu_quota{service=~"app|acs|worker"} > 0) / container_spec_cpu_period{service=~"app|acs|worker"})' "$resources_dashboard" || \
    fail "omc-resources.json: CPU usage-vs-limit panel must calculate per-container quota/period by service"
  grep -Fq 'interface!~"lo|veth.*|docker.*|cni.*|flannel.*|br-.*"' "$resources_dashboard" || \
    fail "omc-resources.json: network panel must exclude loopback and virtual interfaces"
  grep -Fq 'device!=""' "$resources_dashboard" || \
    fail "omc-resources.json: block-IO panel must filter device samples"
  grep -Fq 'container_label_com_docker_compose_service=~"prometheus|grafana|loki|tempo|otelcol|alertmanager|cadvisor"' "$infra_dashboard" || \
    fail "omc-infra.json: monitoring memory panel must use Compose service labels"
  grep -Fq 'container_id=~".*/docker-[^/]+\\\\.scope"' "$nginx_dashboard" || \
    fail "nginx-host-overview.json: container panels must match systemd Docker scope IDs"
fi

if [[ -f "$alerts_config" ]]; then
  cpu_alert_block=$(sed -n '/^      - alert: ContainerCPUSaturation$/,/^      - alert: ContainerCPUThrottling$/p' "$alerts_config")
  grep -Fq 'rate(container_cpu_usage_seconds_total{service=~"app|acs|worker"}[5m])' <<<"$cpu_alert_block" || \
    fail "host-container-alerts.yml: ContainerCPUSaturation must aggregate usage by service"
  grep -Fq '(container_spec_cpu_quota{service=~"app|acs|worker"} > 0)' <<<"$cpu_alert_block" && \
    grep -Fq '/ container_spec_cpu_period{service=~"app|acs|worker"}' <<<"$cpu_alert_block" || \
    fail "host-container-alerts.yml: ContainerCPUSaturation must use per-container quota/period by service"
  grep -Fq 'and on (id) omc:container_last_seen:fresh' <<<"$cpu_alert_block" || \
    fail "host-container-alerts.yml: ContainerCPUSaturation must exclude stale replaced containers"
  if grep -Fq 'name=~"(omcgo|docker)-(app|acs|worker)-.*"' <<<"$cpu_alert_block"; then
    fail "host-container-alerts.yml: ContainerCPUSaturation must not depend on the legacy name selector"
  fi
else
  fail "missing alert config: ${alerts_config#$repo_root/}"
fi

if (( failures > 0 )); then
  printf 'cAdvisor service-label validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'cAdvisor CPU/memory/saturation validation passed.\n'

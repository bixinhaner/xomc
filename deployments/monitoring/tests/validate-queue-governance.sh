#!/usr/bin/env bash

# Keep the Redis governance dashboard aligned with the duplicate-observer
# contract: app and worker scrape the same Redis queues, so panels must expose
# one queue-family value and a separate health signal for every observer.

set -uo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
dashboard="$repo_root/deployments/monitoring/grafana/dashboards/omc-storage-queue-governance.json"
alerts="$repo_root/deployments/monitoring/alerts/storage-queue-alerts.yml"
legacy_alerts="$repo_root/deployments/monitoring/alerts/omc-rules.yml"
failures=0

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  failures=$((failures + 1))
}

if ! command -v jq >/dev/null 2>&1; then
  printf 'ERROR: jq is required to validate the Redis governance dashboard.\n' >&2
  exit 2
fi

if ! jq -e . "$dashboard" >/dev/null 2>&1; then
  fail "dashboard is not valid JSON: ${dashboard#$repo_root/}"
else
  panel_targets() {
    jq -r --arg title "$1" '.panels[] | select(.title == $title) | .targets[]?.expr' "$dashboard"
  }

  require_target() {
    local title=$1
    local expected=$2
    if ! panel_targets "$title" | grep -Fqx "$expected"; then
      fail "dashboard panel '$title' is missing target: $expected"
    fi
  }

  require_target "Redis cmdq/taskq 积压" 'max by (queue_family) (omc_redis_task_queue_length_total)'
  require_target "Redis cmdq/taskq 积压" 'max by (queue_family) (omc_redis_task_queue_active_devices)'
  require_target "Redis cmdq/taskq 积压" 'max by (queue_family) (omc_redis_task_queue_max_length)'
  require_target "Redis 最老任务年龄" 'max by (queue_family) (omc_redis_task_queue_oldest_age_seconds and on (deployment_unit, service, instance, job, queue_family) (omc_redis_task_queue_up == 1) and on (deployment_unit, service, instance, job, queue_family) (time() - omc_redis_task_queue_sample_timestamp_seconds < 120))'
  require_target "Redis 队列观测健康（全量/部署单元）" 'min by (queue_family) (omc_redis_task_queue_up * on (deployment_unit, service, instance, job, queue_family) (time() - omc_redis_task_queue_sample_timestamp_seconds < bool 120))'
  require_target "Redis 队列观测健康（全量/部署单元）" 'min by (deployment_unit, queue_family) (omc_redis_task_queue_up * on (deployment_unit, service, instance, job, queue_family) (time() - omc_redis_task_queue_sample_timestamp_seconds < bool 120))'

  if ! jq -e '.panels[] | select(.title == "Redis 队列观测健康（全量/部署单元）") | .fieldConfig.defaults.thresholds.steps | any(.color == "red" and .value == null) and any(.color == "green" and .value == 1)' "$dashboard" >/dev/null; then
    fail "health panel does not define red=0 and green=1 thresholds"
  fi
fi

if ! grep -Fq 'expr: max by (queue_family) (omc_redis_task_queue_length_total) > 1000' "$alerts"; then
  fail "Redis backlog alert still sums duplicate app/worker observers"
fi

if ! grep -Fq 'expr: (max by (queue_family) (omc_redis_task_queue_oldest_age_seconds' "$alerts" \
  || ! grep -Fq 'omc_redis_task_queue_up == 1' "$alerts" \
  || ! grep -Fq 'time() - omc_redis_task_queue_sample_timestamp_seconds < 120' "$alerts" \
  || ! grep -Fq 'min by (queue_family)' "$alerts" \
  || ! grep -Fq 'time() - omc_redis_task_queue_sample_timestamp_seconds < bool 120' "$alerts"; then
  fail "Redis oldest-age alert is not gated by all-observer health and freshness"
fi

if grep -Fq 'omc_tasks_pending_total' "$legacy_alerts"; then
  fail "legacy process-local task gauge still drives an alert instead of the persistent queue observer"
fi

if ! grep -Fq 'alert: PMRegistrationWaitStale' "$alerts" \
  || ! grep -Fq 'durable="pm-registration-wait"' "$alerts" \
  || ! grep -Fq 'omc_nats_consumer_up == 1' "$alerts" \
  || ! grep -Fq 'time() - omc_nats_consumer_sample_timestamp_seconds < 120' "$alerts"; then
  fail "PM registration wait queue lacks a fresh-observer-gated stale-event alert"
fi

if (( failures > 0 )); then
  printf 'Redis queue governance validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Redis queue governance validation passed.\n'

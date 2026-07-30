#!/usr/bin/env bash

# Lightweight, dependency-free contract check for storage-targets.yml.
# YAML parsing is intentionally left to the deployment toolchain; this check
# catches accidental reintroduction of per-component physical targets.

set -u

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
targets="$repo_root/deployments/monitoring/storage-targets.yml"
recording_rules="$repo_root/deployments/monitoring/alerts/storage-recording.yml"
failures=0

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  failures=$((failures + 1))
}

if [[ ! -f "$targets" ]]; then
  fail "storage target contract is missing"
else
  grep -Eq '^version: 1$' "$targets" || fail "version must be 1"
  grep -Eq '^storage_targets:$' "$targets" || fail "storage_targets section is missing"
  grep -Eq '^minio_buckets:$' "$targets" || fail "minio_buckets section is missing"

  grep -Eq '^[[:space:]]+-[[:space:]]target_type: filesystem$' "$targets" || fail "unified filesystem target is missing"
  grep -Eq '^[[:space:]]+target_id: root$' "$targets" || fail "unified root target is missing"
  grep -Eq '^[[:space:]]+mountpoint: /$' "$targets" || fail "unified root mountpoint is missing"
  grep -Eq '^[[:space:]]+source: node_exporter$' "$targets" || fail "node-exporter source is missing"
  grep -Eq '^logical_components:$' "$targets" || fail "logical component section is missing"
  if [[ $(grep -Ec '^[[:space:]]+-[[:space:]]target_type: filesystem$' "$targets") -ne 1 ]]; then
    fail "exactly one physical filesystem target is required"
  fi
  if grep -Eq '^[[:space:]]+-[[:space:]]target_type: (application|host_filesystem|database|minio|monitoring)$' "$targets"; then
    fail "logical component must not be declared as an independent storage target"
  fi

  for category in pm mr firmware config-backup logs reports exchange ui-assets trace file-bundles config-snapshots device-licenses; do
    grep -Eq "^[[:space:]]+- category: ${category}$" "$targets" || fail "MinIO category is missing: ${category}"
  done

fi

if [[ ! -f "$recording_rules" ]]; then
  fail "storage recording rules are missing"
else
  grep -Eq '^      - record: omc_storage_capacity_bytes$' "$recording_rules" || fail "capacity recording rule is missing"
  grep -Eq '^      - record: omc_storage_used_ratio$' "$recording_rules" || fail "usage ratio recording rule is missing"
  grep -Eq '^      - record: omc_minio_bucket_usage_bytes$' "$recording_rules" || fail "MinIO bucket recording rule is missing"
  grep -Eq '^        expr: max by \(\) \(omc_backup_storage_used_bytes\)$' "$recording_rules" || fail "MinIO bucket recording rule must deduplicate identical deployment-unit gauges"
  grep -Eq '^          target_type: filesystem$' "$recording_rules" || fail "filesystem target label is missing"
  if grep -Eq '^          target_type: (application|minio|database|monitoring)$' "$recording_rules"; then
    fail "logical component target label must not be used for physical capacity rules"
  fi
fi

for dashboard in \
  "$repo_root/deployments/monitoring/grafana/dashboards/omc-infra.json" \
  "$repo_root/deployments/monitoring/grafana/dashboards/omc-storage-queue-governance.json"; do
  if [[ ! -f "$dashboard" ]]; then
    fail "MinIO dashboard is missing: ${dashboard#$repo_root/}"
    continue
  fi
  grep -Eq 'max by \(category\) \(omc_minio_bucket_usage_bytes' "$dashboard" || \
    fail "${dashboard#$repo_root/}: MinIO category panel must collapse duplicate deployment-unit series"
done

if (( failures > 0 )); then
  printf 'Storage target validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Storage target validation passed.\n'

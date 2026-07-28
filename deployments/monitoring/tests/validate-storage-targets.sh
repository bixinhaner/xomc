#!/usr/bin/env bash

# Lightweight, dependency-free contract check for storage-targets.yml.
# YAML parsing is intentionally left to the deployment toolchain; this check
# catches accidental removal of the required logical targets on macOS/Linux.

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

  for target_id in root docker-data postgres-data postgres-tsdb-data minio-data prometheus-data loki-data tempo-data grafana-data alertmanager-data promtail-data; do
    grep -Eq "^[[:space:]]+target_id: ${target_id}$" "$targets" || fail "target_id is missing: ${target_id}"
  done

  for category in pm mr firmware config-backup logs reports exchange ui-assets trace file-bundles config-snapshots device-licenses; do
    grep -Eq "^[[:space:]]+- category: ${category}$" "$targets" || fail "MinIO category is missing: ${category}"
  done

  grep -Eq '^[[:space:]]+mountpoint: /var/lib/docker$' "$targets" || fail "Docker data mountpoint is missing"
  grep -Eq '^[[:space:]]+container_path: /prometheus$' "$targets" || fail "Prometheus data path is missing"
  grep -Eq '^[[:space:]]+container_path: /loki$' "$targets" || fail "Loki data path is missing"
  grep -Eq '^[[:space:]]+container_path: /var/tempo$' "$targets" || fail "Tempo data path is missing"
fi

if [[ ! -f "$recording_rules" ]]; then
  fail "storage recording rules are missing"
else
  grep -Eq '^      - record: omc_storage_capacity_bytes$' "$recording_rules" || fail "capacity recording rule is missing"
  grep -Eq '^      - record: omc_storage_used_ratio$' "$recording_rules" || fail "usage ratio recording rule is missing"
  grep -Eq '^      - record: omc_minio_bucket_usage_bytes$' "$recording_rules" || fail "MinIO bucket recording rule is missing"
  grep -Eq '^          target_type: application$' "$recording_rules" || fail "application target label is missing"
  grep -Eq '^          target_type: minio$' "$recording_rules" || fail "MinIO target label is missing"
fi

if (( failures > 0 )); then
  printf 'Storage target validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Storage target validation passed.\n'

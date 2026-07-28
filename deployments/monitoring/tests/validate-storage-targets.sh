#!/usr/bin/env bash

# Lightweight, dependency-free contract check for storage-targets.yml.
# YAML parsing is intentionally left to the deployment toolchain; this check
# catches accidental removal of the required logical targets on macOS/Linux.

set -u

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
targets="$repo_root/deployments/monitoring/storage-targets.yml"
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

  for target_id in app-root docker-data postgres-data postgres-tsdb-data minio-data prometheus-data loki-data tempo-data grafana-data; do
    grep -Eq "^[[:space:]]+target_id: ${target_id}$" "$targets" || fail "target_id is missing: ${target_id}"
  done

  for category in pm mr firmware config-backup logs reports exchange ui-assets trace; do
    grep -Eq "^[[:space:]]+- category: ${category}$" "$targets" || fail "MinIO category is missing: ${category}"
  done

  grep -Eq '^[[:space:]]+mountpoint: /var/lib/docker$' "$targets" || fail "Docker data mountpoint is missing"
  grep -Eq '^[[:space:]]+container_path: /prometheus$' "$targets" || fail "Prometheus data path is missing"
  grep -Eq '^[[:space:]]+container_path: /loki$' "$targets" || fail "Loki data path is missing"
  grep -Eq '^[[:space:]]+container_path: /var/tempo$' "$targets" || fail "Tempo data path is missing"
fi

if (( failures > 0 )); then
  printf 'Storage target validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Storage target validation passed.\n'

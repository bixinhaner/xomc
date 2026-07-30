#!/usr/bin/env bash

# Validate provisioned Grafana dashboards before they are loaded by Grafana.
# New dashboard JSON files placed in grafana/dashboards/ are discovered
# automatically; the four baseline files are required to remain present.

set -uo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
dashboard_dir="$repo_root/deployments/monitoring/grafana/dashboards"
required_dashboards=(
  "$dashboard_dir/nginx-host-overview.json"
  "$dashboard_dir/omc-infra.json"
  "$dashboard_dir/omc-overview.json"
  "$dashboard_dir/omc-resources.json"
)

failures=0
uid_values=()
uid_paths=()
dashboards=()

fail() {
  printf 'ERROR: %s\n' "$*" >&2
  failures=$((failures + 1))
}

if ! command -v jq >/dev/null 2>&1; then
  printf 'ERROR: jq is required to validate Grafana dashboard JSON.\n' >&2
  exit 2
fi

for dashboard in "${required_dashboards[@]}"; do
  if [[ ! -f "$dashboard" ]]; then
    fail "required dashboard is missing: ${dashboard#$repo_root/}"
    continue
  fi
  dashboards+=("$dashboard")
done

if [[ -d "$dashboard_dir" ]]; then
  while IFS= read -r dashboard; do
    found=false
    for required in "${required_dashboards[@]}"; do
      if [[ "$dashboard" == "$required" ]]; then
        found=true
        break
      fi
    done
    if [[ "$found" == false ]]; then
      dashboards+=("$dashboard")
      printf 'INFO: validating future dashboard: %s\n' "${dashboard#$repo_root/}"
    fi
  done < <(find "$dashboard_dir" -maxdepth 1 -type f -name '*.json' -print | sort)
else
  fail "dashboard directory is missing: ${dashboard_dir#$repo_root/}"
fi

if (( ${#dashboards[@]} == ${#required_dashboards[@]} )); then
  printf 'INFO: no future dashboard JSON is present yet; validating the four baseline dashboards.\n'
fi

for dashboard in "${dashboards[@]}"; do
  relative_path=${dashboard#$repo_root/}
  if ! jq -e . "$dashboard" >/dev/null; then
    fail "$relative_path is not valid JSON"
    continue
  fi

  uid=$(jq -er '.uid | strings | select(length > 0)' "$dashboard" 2>/dev/null) || {
    fail "$relative_path has no non-empty top-level uid"
    continue
  }
  duplicate_uid=false
  for uid_index in "${!uid_values[@]}"; do
    if [[ "${uid_values[$uid_index]}" == "$uid" ]]; then
      fail "duplicate dashboard uid '$uid': ${uid_paths[$uid_index]} and $relative_path"
      duplicate_uid=true
      break
    fi
  done
  if [[ "$duplicate_uid" == false ]]; then
    uid_values+=("$uid")
    uid_paths+=("$relative_path")
  fi

  strings=$(jq -r '.. | strings' "$dashboard")
  if grep -Fq 'namespace="omcgo"' <<<"$strings"; then
    fail "$relative_path contains forbidden selector namespace=\"omcgo\""
  fi
  if grep -Fq 'name=~' <<<"$strings"; then
    fail "$relative_path contains forbidden selector name=~"
  fi
done

if (( failures > 0 )); then
  printf 'Dashboard validation failed: %d issue(s).\n' "$failures" >&2
  exit 1
fi

printf 'Dashboard validation passed: %d dashboard(s), %d unique UID(s).\n' \
  "${#dashboards[@]}" "${#uid_values[@]}"

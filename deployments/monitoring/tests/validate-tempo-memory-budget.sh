#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
compose_file="$repo_root/deployments/docker/docker-compose.yml"
planner_file="$repo_root/deployments/docker/plan-resources.sh"
release_compose_file="$repo_root/deployments/release/bundle/deploy/docker-compose.monitoring.yml"
release_planner_file="$repo_root/deployments/release/bundle/deploy/plan-resources.sh"

tempo_block() {
  awk '
    /^  tempo:$/ { in_tempo = 1 }
    in_tempo && /^  [a-zA-Z0-9_-]+:$/ && $0 != "  tempo:" { exit }
    in_tempo { print }
  ' "$1"
}

validate_compose() {
  local file="$1"
  local block
  block="$(tempo_block "$file")"

  grep -Fq 'GOMEMLIMIT: "768MiB"' <<<"$block" || {
    echo "$file: Tempo must keep its Go runtime below the container hard limit with GOMEMLIMIT=768MiB" >&2
    exit 1
  }

  grep -Fq 'memory: 1g' <<<"$block" || {
    echo "$file: Tempo must have a 1GiB hard limit for peak block compaction" >&2
    exit 1
  }
}

validate_planner() {
  local file="$1"
  grep -Eq 'MON_FIXED_MIB=(0[[:space:]]*;.*&&[[:space:]]*)?4736|MON_FIXED_MIB=4736' "$file" || {
    echo "$file: Resource planner must include Tempo's additional 512MiB" >&2
    exit 1
  }
}

validate_compose "$compose_file"
validate_compose "$release_compose_file"
validate_planner "$planner_file"
validate_planner "$release_planner_file"

echo "Tempo memory budget contract is valid"

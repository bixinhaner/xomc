#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
compose_file="$repo_root/deployments/docker/docker-compose.yml"
planner_file="$repo_root/deployments/docker/plan-resources.sh"

tempo_block="$(
  awk '
    /^  tempo:$/ { in_tempo = 1 }
    in_tempo && /^  [a-zA-Z0-9_-]+:$/ && $0 != "  tempo:" { exit }
    in_tempo { print }
  ' "$compose_file"
)"

grep -Fq 'GOMEMLIMIT: "768MiB"' <<<"$tempo_block" || {
  echo "Tempo must keep its Go runtime below the container hard limit with GOMEMLIMIT=768MiB" >&2
  exit 1
}

grep -Fq 'memory: 1g' <<<"$tempo_block" || {
  echo "Tempo must have a 1GiB hard limit for peak block compaction" >&2
  exit 1
}

grep -Fq 'MON_FIXED_MIB=4736' "$planner_file" || {
  echo "Resource planner must include Tempo's additional 512MiB" >&2
  exit 1
}

echo "Tempo memory budget contract is valid"

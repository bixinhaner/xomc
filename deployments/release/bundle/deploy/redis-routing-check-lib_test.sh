#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=redis-routing-check-lib.sh
source "$SCRIPT_DIR/redis-routing-check-lib.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

write_config() {
  local file="$1" core="$2" pm="$3"
  printf 'redis:\n  addrs:\n    - "%s"\npm_redis:\n  addrs:\n    - "%s"\n' "$core" "$pm" > "$file"
}

write_config "$tmp/app.prod.yaml" redis-core:6379 redis-pm:6379
write_config "$tmp/worker.prod.yaml" redis-core:6379 redis-pm:6379
redis_routing_configs_valid "$tmp"

write_config "$tmp/app.prod.yaml" redis-pm:6379 redis-core:6379
if redis_routing_configs_valid "$tmp"; then
  echo "FAIL: swapped App Redis roles must not pass" >&2
  exit 1
fi

cat > "$tmp/app.prod.yaml" <<'YAML'
# redis-core:6379 and redis-pm:6379 are examples only
redis:
  addrs:
    - "wrong-core:6379"
pm_redis:
  addrs:
    - "wrong-pm:6379"
YAML
if redis_routing_configs_valid "$tmp"; then
  echo "FAIL: Redis addresses present only in comments must not pass" >&2
  exit 1
fi

echo "PASS: Redis routing check parses exact top-level YAML roles"

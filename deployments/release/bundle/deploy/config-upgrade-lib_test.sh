#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=config-upgrade-lib.sh
source "$SCRIPT_DIR/config-upgrade-lib.sh"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/template.yaml" <<'YAML'
session:
  timeout: 5m
  max_concurrent: 30000
  max_rpc_per_session: 50
YAML

cat > "$tmp/legacy.yaml" <<'YAML'
session:
  timeout: 5m
  max_concurrent: 10000
  max_rpc_per_session: 50
software:
  max_concurrent: 5
YAML
upgrade_acs_session_limit "$tmp/legacy.yaml" "$tmp/template.yaml"
grep -q '^  max_concurrent: 30000$' "$tmp/legacy.yaml" || {
  echo "FAIL: legacy ACS default was not upgraded" >&2
  exit 1
}
grep -q '^  max_concurrent: 5$' "$tmp/legacy.yaml" || {
  echo "FAIL: unrelated max_concurrent was changed" >&2
  exit 1
}

cat > "$tmp/custom.yaml" <<'YAML'
session:
  timeout: 5m
  max_concurrent: 15000 # operator override
YAML
before_custom="$(cksum < "$tmp/custom.yaml")"
upgrade_acs_session_limit "$tmp/custom.yaml" "$tmp/template.yaml"
[ "$(cksum < "$tmp/custom.yaml")" = "$before_custom" ] || {
  echo "FAIL: operator override must be preserved" >&2
  exit 1
}

cp "$tmp/template.yaml" "$tmp/current.yaml"
before_current="$(cksum < "$tmp/current.yaml")"
upgrade_acs_session_limit "$tmp/current.yaml" "$tmp/template.yaml"
[ "$(cksum < "$tmp/current.yaml")" = "$before_current" ] || {
  echo "FAIL: current config migration must be idempotent" >&2
  exit 1
}

echo "PASS: ACS session limit upgrade preserves operator configuration"

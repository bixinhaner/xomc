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

cat > "$tmp/nested.yaml" <<'YAML'
session:
  limits:
    max_concurrent: 10000
  max_concurrent: 15000
YAML
before_nested="$(cksum < "$tmp/nested.yaml")"
upgrade_acs_session_limit "$tmp/nested.yaml" "$tmp/template.yaml"
[ "$(cksum < "$tmp/nested.yaml")" = "$before_nested" ] || {
  echo "FAIL: nested max_concurrent must not be treated as the session limit" >&2
  exit 1
}

cat > "$tmp/missing.yaml" <<'YAML'
session:
  timeout: 5m
YAML
before_missing="$(cksum < "$tmp/missing.yaml")"
upgrade_acs_session_limit "$tmp/missing.yaml" "$tmp/template.yaml"
[ "${ACS_SESSION_LIMIT_UPGRADE_RESULT:-}" = "noop" ] || {
  echo "FAIL: missing session limit must not be reported as an operator override" >&2
  exit 1
}
[ "$(cksum < "$tmp/missing.yaml")" = "$before_missing" ] || {
  echo "FAIL: missing session limit config must remain intact" >&2
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

cat > "$tmp/write-failure.yaml" <<'YAML'
session:
  max_concurrent: 10000
YAML
before_failure="$(cksum < "$tmp/write-failure.yaml")"
mv() { return 73; }
if upgrade_acs_session_limit "$tmp/write-failure.yaml" "$tmp/template.yaml"; then
  echo "FAIL: replacement failure must be returned to the installer" >&2
  exit 1
fi
unset -f mv
[ "$(cksum < "$tmp/write-failure.yaml")" = "$before_failure" ] || {
  echo "FAIL: replacement failure must leave the live config intact" >&2
  exit 1
}

echo "PASS: ACS session limit upgrade preserves operator configuration"

cat > "$tmp/app-template.yaml" <<'YAML'
provision:
  auto_configure: true
  gpv_response:
    provision_queue: "${GPV_PROVISION_QUEUE:-provision-gpv}"
    rpc_durable: "${GPV_RPC_DURABLE:-device-rpc-gpv}"
    rpc_start_sequence: ${GPV_RPC_START_SEQUENCE:-0}
    ack_wait: "${GPV_ACK_WAIT:-30s}"
server:
  port: 8081
YAML

cat > "$tmp/app-legacy.yaml" <<'YAML'
provision:
  auto_configure: false
  batch:
    enabled: true
server:
  port: 18081
YAML
upgrade_app_gpv_response_config "$tmp/app-legacy.yaml" "$tmp/app-template.yaml"
grep -q '^  gpv_response:$' "$tmp/app-legacy.yaml" || {
  echo "FAIL: legacy app config did not receive gpv_response" >&2
  exit 1
}
grep -q '^  auto_configure: false$' "$tmp/app-legacy.yaml" || {
  echo "FAIL: GPV config merge changed operator provision settings" >&2
  exit 1
}
grep -q '^  port: 18081$' "$tmp/app-legacy.yaml" || {
  echo "FAIL: GPV config merge changed unrelated settings" >&2
  exit 1
}

before_gpv_current="$(cksum < "$tmp/app-legacy.yaml")"
upgrade_app_gpv_response_config "$tmp/app-legacy.yaml" "$tmp/app-template.yaml"
[ "$(cksum < "$tmp/app-legacy.yaml")" = "$before_gpv_current" ] || {
  echo "FAIL: GPV config merge must be idempotent" >&2
  exit 1
}

cat > "$tmp/app-custom.yaml" <<'YAML'
provision:
  gpv_response:
    rpc_durable: operator-custom
    rpc_start_sequence: 12345
YAML
before_gpv_custom="$(cksum < "$tmp/app-custom.yaml")"
upgrade_app_gpv_response_config "$tmp/app-custom.yaml" "$tmp/app-template.yaml"
[ "$(cksum < "$tmp/app-custom.yaml")" = "$before_gpv_custom" ] || {
  echo "FAIL: existing operator GPV config must be preserved" >&2
  exit 1
}

echo "PASS: app GPV response config upgrade is additive and idempotent"

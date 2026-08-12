#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=config-upgrade-lib.sh
source "$SCRIPT_DIR/config-upgrade-lib.sh"

file_mode() {
  local file="$1"
  stat -c '%a' "$file" 2>/dev/null || stat -f '%Lp' "$file"
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/etc-license-only/license"
: > "$tmp/etc-license-only/license/omcPublicKey.store"
if config_dir_has_instance_config "$tmp/etc-license-only"; then
  echo "FAIL: a license-only etc directory must still be treated as a fresh install" >&2
  exit 1
fi

mkdir -p "$tmp/etc-with-app"
: > "$tmp/etc-with-app/app.prod.yaml"
config_dir_has_instance_config "$tmp/etc-with-app" || {
  echo "FAIL: an etc directory with an instance config must be treated as an upgrade" >&2
  exit 1
}

echo "PASS: instance config detection ignores auxiliary etc files"

cat > "$tmp/pm-template.yaml" <<'YAML'
redis:
  addrs:
    - "redis-core:6379"
pm_redis:
  addrs:
    - "redis-pm:6379"
  pool_size: 200
nats:
  url: "nats://nats:4222"
YAML
cat > "$tmp/pm-legacy.yaml" <<'YAML'
redis:
  addrs:
    - "operator-core:6379"
nats:
  url: "nats://operator-nats:4222"
YAML
upgrade_pm_redis_config "$tmp/pm-legacy.yaml" "$tmp/pm-template.yaml"
grep -Fq '    - "redis-pm:6379"' "$tmp/pm-legacy.yaml" || {
  echo "FAIL: legacy config did not receive dedicated pm_redis" >&2
  exit 1
}
grep -Fq '    - "operator-core:6379"' "$tmp/pm-legacy.yaml" || {
  echo "FAIL: pm_redis upgrade changed operator core Redis" >&2
  exit 1
}
before_pm_current="$(cksum < "$tmp/pm-legacy.yaml")"
upgrade_pm_redis_config "$tmp/pm-legacy.yaml" "$tmp/pm-template.yaml"
[ "$(cksum < "$tmp/pm-legacy.yaml")" = "$before_pm_current" ] || {
  echo "FAIL: pm_redis config upgrade must be idempotent" >&2
  exit 1
}

echo "PASS: production config upgrade adds dedicated PM Redis"

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

cat > "$tmp/dsn-template.yaml" <<'YAML'
db:
  dsn: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
tsdb:
  dsn: "postgres://${POSTGRES_TSDB_USER}:${POSTGRES_TSDB_PASSWORD}@${TSDB_HOST}:5432/${POSTGRES_TSDB_DB}?sslmode=disable"
YAML
cat > "$tmp/dsn-legacy.yaml" <<'YAML'
db:
  dsn: "postgres://omcgo:old-password@postgres:5432/omcgo?sslmode=disable"
  max_connections: 180
tsdb:
  dsn: "postgres://omcgo:old-tsdb-password@postgres-tsdb:5432/omcgo?sslmode=disable"
server:
  port: 7557
YAML
upgrade_prod_database_dsns "$tmp/dsn-legacy.yaml" "$tmp/dsn-template.yaml"
grep -Fq 'postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable' "$tmp/dsn-legacy.yaml" || {
  echo "FAIL: legacy primary database DSN was not synchronized" >&2
  exit 1
}
grep -Fq 'postgres://${POSTGRES_TSDB_USER}:${POSTGRES_TSDB_PASSWORD}@${TSDB_HOST}:5432/${POSTGRES_TSDB_DB}?sslmode=disable' "$tmp/dsn-legacy.yaml" || {
  echo "FAIL: legacy TSDB DSN was not synchronized" >&2
  exit 1
}
grep -q '^  max_connections: 180$' "$tmp/dsn-legacy.yaml" || {
  echo "FAIL: operator database settings were changed" >&2
  exit 1
}

echo "PASS: production database DSN upgrade uses environment credentials"

cat > "$tmp/paramsync-template.yaml" <<'YAML'
param_sync:
  recovery_run_limit: 200
  recovery_task_limit_per_run: 200
  recovery_task_budget: 200
YAML
cat > "$tmp/paramsync-legacy.yaml" <<'YAML'
param_sync:
  recovery_run_limit: 20
  recovery_task_limit_per_run: 200
  recovery_task_budget: 200
server:
  port: 18081
YAML
chmod 0640 "$tmp/paramsync-legacy.yaml"
upgrade_app_param_sync_recovery_limit "$tmp/paramsync-legacy.yaml" "$tmp/paramsync-template.yaml"
grep -q '^  recovery_run_limit: 200$' "$tmp/paramsync-legacy.yaml" || {
  echo "FAIL: legacy parameter sync recovery limit was not upgraded" >&2
  exit 1
}
grep -q '^  port: 18081$' "$tmp/paramsync-legacy.yaml" || {
  echo "FAIL: parameter sync limit upgrade changed unrelated configuration" >&2
  exit 1
}
[ "$(file_mode "$tmp/paramsync-legacy.yaml")" = "640" ] || {
  echo "FAIL: parameter sync limit upgrade changed live config permissions" >&2
  exit 1
}
before_paramsync_current="$(cksum < "$tmp/paramsync-legacy.yaml")"
upgrade_app_param_sync_recovery_limit "$tmp/paramsync-legacy.yaml" "$tmp/paramsync-template.yaml"
[ "$(cksum < "$tmp/paramsync-legacy.yaml")" = "$before_paramsync_current" ] || {
  echo "FAIL: parameter sync recovery limit upgrade must be idempotent" >&2
  exit 1
}
cat > "$tmp/paramsync-custom.yaml" <<'YAML'
param_sync:
  recovery_run_limit: 80 # operator override
YAML
before_paramsync_custom="$(cksum < "$tmp/paramsync-custom.yaml")"
upgrade_app_param_sync_recovery_limit "$tmp/paramsync-custom.yaml" "$tmp/paramsync-template.yaml"
[ "$(cksum < "$tmp/paramsync-custom.yaml")" = "$before_paramsync_custom" ] || {
  echo "FAIL: operator parameter sync recovery limit must be preserved" >&2
  exit 1
}

echo "PASS: parameter sync recovery limit upgrade preserves operator configuration"

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

cat > "$tmp/worker-template.yaml" <<'YAML'
db:
  max_conns: 25
tsdb:
  max_conns: 128
redis:
  pool_size: 100
YAML

cat > "$tmp/worker-legacy.yaml" <<'YAML'
db:
  max_conns: 25
tsdb:
  max_conns: 40 # old project default
redis:
  pool_size: 100
YAML
upgrade_worker_tsdb_pool "$tmp/worker-legacy.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "migrated" ] || {
  echo "FAIL: legacy Worker TSDB pool must be reported as migrated" >&2
  exit 1
}
grep -q '^  max_conns: 128 # old project default$' "$tmp/worker-legacy.yaml" || {
  echo "FAIL: legacy Worker TSDB pool was not upgraded from 40 to 128" >&2
  exit 1
}
grep -q '^  max_conns: 25$' "$tmp/worker-legacy.yaml" || {
  echo "FAIL: Worker DB pool was changed with the TSDB pool" >&2
  exit 1
}

cat > "$tmp/worker-previous-default.yaml" <<'YAML'
tsdb:
  max_conns: 96 # previous production default
YAML
validate_worker_tsdb_pool_precheck "$tmp/worker-previous-default.yaml" "$tmp/worker-template.yaml" || {
  echo "FAIL: precheck must allow the previous 96-connection project default for migration" >&2
  exit 1
}
upgrade_worker_tsdb_pool "$tmp/worker-previous-default.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "migrated" ] || {
  echo "FAIL: previous 96-connection Worker TSDB default must be migrated" >&2
  exit 1
}
grep -q '^  max_conns: 128 # previous production default$' "$tmp/worker-previous-default.yaml" || {
  echo "FAIL: previous Worker TSDB pool was not upgraded from 96 to 128" >&2
  exit 1
}

before_worker_current="$(cksum < "$tmp/worker-legacy.yaml")"
upgrade_worker_tsdb_pool "$tmp/worker-legacy.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "noop" ] || {
  echo "FAIL: current Worker TSDB pool must be a no-op" >&2
  exit 1
}
[ "$(cksum < "$tmp/worker-legacy.yaml")" = "$before_worker_current" ] || {
  echo "FAIL: Worker TSDB pool migration must be idempotent" >&2
  exit 1
}

cat > "$tmp/worker-custom-sufficient.yaml" <<'YAML'
tsdb:
  max_conns: 160
YAML
before_worker_custom="$(cksum < "$tmp/worker-custom-sufficient.yaml")"
upgrade_worker_tsdb_pool "$tmp/worker-custom-sufficient.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "preserved" ] || {
  echo "FAIL: sufficient operator Worker TSDB pool must be preserved" >&2
  exit 1
}
[ "$(cksum < "$tmp/worker-custom-sufficient.yaml")" = "$before_worker_custom" ] || {
  echo "FAIL: sufficient operator Worker TSDB pool was overwritten" >&2
  exit 1
}

cat > "$tmp/worker-custom-insufficient.yaml" <<'YAML'
tsdb:
  max_conns: 64
YAML
before_worker_insufficient="$(cksum < "$tmp/worker-custom-insufficient.yaml")"
upgrade_worker_tsdb_pool "$tmp/worker-custom-insufficient.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "insufficient" ] || {
  echo "FAIL: insufficient operator Worker TSDB pool must block deployment" >&2
  exit 1
}
[ "$(cksum < "$tmp/worker-custom-insufficient.yaml")" = "$before_worker_insufficient" ] || {
  echo "FAIL: insufficient operator Worker TSDB pool must not be overwritten" >&2
  exit 1
}

cat > "$tmp/worker-quoted-legacy.yaml" <<'YAML'
tsdb:
  max_conns: "40" # quoted old project default
YAML
upgrade_worker_tsdb_pool "$tmp/worker-quoted-legacy.yaml" "$tmp/worker-template.yaml"
grep -q '^  max_conns: "128" # quoted old project default$' "$tmp/worker-quoted-legacy.yaml" || {
  echo "FAIL: quoted legacy Worker TSDB pool was not migrated with its scalar style preserved" >&2
  exit 1
}

cat > "$tmp/worker-quoted-sufficient.yaml" <<'YAML'
tsdb:
  max_conns: '160'
YAML
before_worker_quoted_sufficient="$(cksum < "$tmp/worker-quoted-sufficient.yaml")"
upgrade_worker_tsdb_pool "$tmp/worker-quoted-sufficient.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "preserved" ] || {
  echo "FAIL: quoted sufficient Worker TSDB pool must be preserved" >&2
  exit 1
}
[ "$(cksum < "$tmp/worker-quoted-sufficient.yaml")" = "$before_worker_quoted_sufficient" ] || {
  echo "FAIL: quoted sufficient Worker TSDB pool was overwritten" >&2
  exit 1
}

cat > "$tmp/worker-quoted-insufficient.yaml" <<'YAML'
tsdb:
  max_conns: "64"
YAML
upgrade_worker_tsdb_pool "$tmp/worker-quoted-insufficient.yaml" "$tmp/worker-template.yaml"
[ "${WORKER_TSDB_POOL_UPGRADE_RESULT:-}" = "insufficient" ] || {
  echo "FAIL: quoted insufficient Worker TSDB pool must block deployment" >&2
  exit 1
}

cat > "$tmp/worker-bad-template.yaml" <<'YAML'
tsdb:
  max_conns: 64
YAML
cat > "$tmp/worker-template-guard-live.yaml" <<'YAML'
tsdb:
  max_conns: 40
YAML
before_worker_bad_template="$(cksum < "$tmp/worker-template-guard-live.yaml")"
if upgrade_worker_tsdb_pool "$tmp/worker-template-guard-live.yaml" "$tmp/worker-bad-template.yaml"; then
  echo "FAIL: Worker TSDB migration must reject a template whose safe budget is not 128" >&2
  exit 1
fi
[ "$(cksum < "$tmp/worker-template-guard-live.yaml")" = "$before_worker_bad_template" ] || {
  echo "FAIL: invalid Worker template must not modify the live config" >&2
  exit 1
}

validate_worker_tsdb_pool_precheck "$tmp/worker-legacy.yaml" "$tmp/worker-template.yaml"
validate_worker_tsdb_pool_precheck "$tmp/worker-custom-sufficient.yaml" "$tmp/worker-template.yaml"
if validate_worker_tsdb_pool_precheck "$tmp/worker-custom-insufficient.yaml" "$tmp/worker-template.yaml"; then
  echo "FAIL: precheck must reject an insufficient Worker TSDB operator override" >&2
  exit 1
fi
if validate_worker_tsdb_pool_precheck "$tmp/worker-template-guard-live.yaml" "$tmp/worker-bad-template.yaml"; then
  echo "FAIL: precheck must reject an invalid Worker template budget" >&2
  exit 1
fi
validate_worker_tsdb_pool_precheck "$tmp/not-installed-worker.yaml" "$tmp/worker-template.yaml"

for leading_zero_value in 040 '"040"'; do
  cat > "$tmp/worker-leading-zero.yaml" <<YAML
tsdb:
  max_conns: $leading_zero_value
YAML
  before_worker_leading_zero="$(cksum < "$tmp/worker-leading-zero.yaml")"
  if validate_worker_tsdb_pool_precheck "$tmp/worker-leading-zero.yaml" "$tmp/worker-template.yaml"; then
    echo "FAIL: precheck must reject non-canonical Worker TSDB scalar $leading_zero_value" >&2
    exit 1
  fi
  if upgrade_worker_tsdb_pool "$tmp/worker-leading-zero.yaml" "$tmp/worker-template.yaml"; then
    echo "FAIL: migration must reject non-canonical Worker TSDB scalar $leading_zero_value" >&2
    exit 1
  fi
  [ "$(cksum < "$tmp/worker-leading-zero.yaml")" = "$before_worker_leading_zero" ] || {
    echo "FAIL: rejected non-canonical Worker TSDB scalar must not modify live config" >&2
    exit 1
  }
done

echo "PASS: Worker TSDB pool upgrade is safe, idempotent, and detects insufficient overrides"

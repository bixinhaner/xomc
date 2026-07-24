#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$DEPLOY_DIR/monitoring-profile-lib.sh"

tmp_env="$(mktemp)"
trap 'rm -f "$tmp_env"' EXIT
printf '%s\n' 'OMCGO_TRACER_ENABLED=true' > "$tmp_env"

# Reproduce install ordering: package .env is sourced first, then the selected
# deployment profile must win.
set -a
. "$tmp_env"
set +a
monitoring_profile_apply_install "$tmp_env" 1
[ "$OMCGO_TRACER_ENABLED" = false ]
[ "$OMCGO_SKIP_MONITORING" = 1 ]
grep -qx 'OMCGO_SKIP_MONITORING=1' "$tmp_env"

# A later standalone svc/healthcheck process has no inherited shell state and
# must recover the intentional monitoring-free profile from .env.
unset OMCGO_TRACER_ENABLED OMCGO_SKIP_MONITORING
SKIP_MONITORING=0
monitoring_profile_apply_runtime "$tmp_env" "$SKIP_MONITORING"
[ "$SKIP_MONITORING" = 1 ]
[ "$OMCGO_TRACER_ENABLED" = false ]

# A normal install persists the full-monitoring profile and a later process
# must require monitoring/OTEL again.
printf '%s\n' 'OMCGO_TRACER_ENABLED=true' > "$tmp_env"
set -a
. "$tmp_env"
set +a
monitoring_profile_apply_install "$tmp_env" 0
unset OMCGO_SKIP_MONITORING
SKIP_MONITORING=0
monitoring_profile_apply_runtime "$tmp_env" "$SKIP_MONITORING"
[ "$SKIP_MONITORING" = 0 ]
[ "$OMCGO_TRACER_ENABLED" = true ]
grep -qx 'OMCGO_SKIP_MONITORING=0' "$tmp_env"

# The install integration must call the shared profile function after sourcing
# the release .env, otherwise tracer=true can still overwrite the skip flag.
source_line="$(grep -n 'source "$ENV_FILE"' "$DEPLOY_DIR/install.sh" | tail -n1 | cut -d: -f1)"
apply_line="$(grep -n 'monitoring_profile_apply_install "$ENV_FILE" "$SKIP_MONITORING"' "$DEPLOY_DIR/install.sh" | tail -n1 | cut -d: -f1)"
[ -n "$source_line" ] && [ -n "$apply_line" ] && [ "$apply_line" -gt "$source_line" ]

grep -Fq 'monitoring_profile_apply_runtime ".env" "$SKIP_MONITORING"' "$DEPLOY_DIR/svc.sh"
grep -Fq 'monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING"' "$DEPLOY_DIR/healthcheck.sh"

echo "monitoring profile behavior: PASS"

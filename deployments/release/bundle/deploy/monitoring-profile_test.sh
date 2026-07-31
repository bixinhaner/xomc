#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$DEPLOY_DIR/monitoring-profile-lib.sh"

tmp_env="$(mktemp)"
tmp_meta_dir="$(mktemp -d)"
trap 'rm -f "$tmp_env"; rm -rf "$tmp_meta_dir"' EXIT
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
source_line="$(grep -n 'deploy_env_load "$ENV_FILE" "$RESOURCE_ENV_FILE"' "$DEPLOY_DIR/install.sh" | tail -n1 | cut -d: -f1)"
apply_line="$(grep -n 'monitoring_profile_apply_install "$ENV_FILE" "$SKIP_MONITORING"' "$DEPLOY_DIR/install.sh" | tail -n1 | cut -d: -f1)"
[ -n "$source_line" ] && [ -n "$apply_line" ] && [ "$apply_line" -gt "$source_line" ]

grep -Fq 'monitoring_profile_apply_runtime ".env" "$SKIP_MONITORING"' "$DEPLOY_DIR/svc.sh"
grep -Fq 'monitoring_profile_apply_runtime "$DEPLOY_DIR/.env" "$SKIP_MONITORING"' "$DEPLOY_DIR/healthcheck.sh"

portable_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

portable_owner_group() {
  stat -c '%u:%g' "$1" 2>/dev/null || stat -f '%u:%g' "$1"
}

metadata_env="$tmp_meta_dir/.env"
printf '%s\n' 'OMCGO_TRACER_ENABLED=true' > "$metadata_env"
chmod 0640 "$metadata_env"
# Root CI can exercise a real owner/group transition; unprivileged runs still
# verify that the caller's existing ownership is retained.
if [ "$(id -u)" = 0 ]; then
  chown 1:1 "$metadata_env"
fi
mode_before="$(portable_mode "$metadata_env")"
owner_group_before="$(portable_owner_group "$metadata_env")"

monitoring_profile_write_state "$metadata_env" 1

[ "$(portable_mode "$metadata_env")" = "$mode_before" ]
[ "$(portable_owner_group "$metadata_env")" = "$owner_group_before" ]
grep -qx 'OMCGO_TRACER_ENABLED=true' "$metadata_env"
grep -qx 'OMCGO_SKIP_MONITORING=1' "$metadata_env"

new_env="$tmp_meta_dir/new.env"
monitoring_profile_write_state "$new_env" 0
[ "$(portable_mode "$new_env")" = 640 ]
grep -qx 'OMCGO_SKIP_MONITORING=0' "$new_env"

# GNU stat accepts -f but treats it as filesystem statistics. Verify the
# portable helpers prefer GNU's -c form so successful, unrelated -f output is
# never mistaken for file mode or ownership.
fake_bin="$tmp_meta_dir/bin"
mkdir -p "$fake_bin"
cat > "$fake_bin/stat" <<'EOF'
#!/usr/bin/env sh
case "$1:$2" in
  -c:%a) printf '640\n' ;;
  -c:%u:%g) printf '7:8\n' ;;
  -f:*) printf 'GNU filesystem statistics that must not be parsed\n' ;;
  *) exit 1 ;;
esac
EOF
chmod +x "$fake_bin/stat"
[ "$(PATH="$fake_bin:$PATH" monitoring_profile_stat_mode "$metadata_env")" = 640 ]
[ "$(PATH="$fake_bin:$PATH" monitoring_profile_stat_owner_group "$metadata_env")" = 7:8 ]

echo "monitoring profile behavior: PASS"

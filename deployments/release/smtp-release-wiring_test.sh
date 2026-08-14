#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RELEASE_COMPOSE="$SCRIPT_DIR/bundle/deploy/docker-compose.app.yml"
DEV_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.yml"
BUILD_SCRIPT="$SCRIPT_DIR/build-release.sh"
INSTALL_SCRIPT="$SCRIPT_DIR/bundle/deploy/install.sh"

SMTP_KEYS="OMCGO_NOTIFICATION_SMTP_ENABLED OMCGO_NOTIFICATION_SMTP_HOST OMCGO_NOTIFICATION_SMTP_PORT OMCGO_NOTIFICATION_SMTP_USERNAME OMCGO_NOTIFICATION_SMTP_PASSWORD OMCGO_NOTIFICATION_SMTP_FROM OMCGO_NOTIFICATION_SMTP_TLS_MODE OMCGO_NOTIFICATION_SMTP_TIMEOUT OMCGO_NOTIFICATION_SMTP_MAX_ATTACHMENT_BYTES"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

for key in $SMTP_KEYS; do
  release_count="$(grep -c "^[[:space:]]*$key:" "$RELEASE_COMPOSE" || true)"
  [ "$release_count" -eq 2 ] || fail "$key must be injected into release app and worker"

  dev_count="$(grep -c "^[[:space:]]*$key:" "$DEV_COMPOSE" || true)"
  [ "$dev_count" -eq 2 ] || fail "$key must be injected into development app and worker"

  build_count="$(grep -c "^$key=" "$BUILD_SCRIPT" || true)"
  [ "$build_count" -eq 1 ] || fail "$key must have one release .env default"

  grep -F "ENV_PRESERVE_KEYS=" "$INSTALL_SCRIPT" | grep -Fqw "$key" ||
    fail "$key must be inherited across release upgrades"
done

[ -x "$SCRIPT_DIR/bundle/deploy/configure-smtp.sh" ] || fail "configure-smtp.sh must be executable"
[ -f "$SCRIPT_DIR/bundle/deploy/smtp.env.example" ] || fail "smtp.env.example must be shipped"

printf 'PASS: SMTP release wiring covers app, worker, defaults, and upgrade inheritance\n'

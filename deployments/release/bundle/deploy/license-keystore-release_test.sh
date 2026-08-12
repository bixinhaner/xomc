#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.app.yml"
INSTALL_SCRIPT="$SCRIPT_DIR/install.sh"
BUILD_SCRIPT="$REPO_ROOT/deployments/release/build-release.sh"
KEYSTORE="$REPO_ROOT/license-run-time/keystore/omcPublicKey.store"
EXPECTED_SHA256="1650aebbb63f320408ade0bc75128eec45d423d50a6ae61e4e368a30216f3331"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

[ -f "$KEYSTORE" ] || fail "repository TrueLicense keystore is missing"
[ "$(sha256sum "$KEYSTORE" | awk '{print $1}')" = "$EXPECTED_SHA256" ] ||
  fail "repository TrueLicense keystore checksum changed unexpectedly"

grep -Fq '/opt/omc/etc/license/omcPublicKey.store:/etc/omc/license/omcPublicKey.store:ro' "$COMPOSE_FILE" ||
  fail "release app compose does not read-only mount the external keystore"
grep -Fq '/sys:/host/sys:ro' "$COMPOSE_FILE" ||
  fail "release app compose does not mount host sysfs for hardware binding"
grep -Fq 'OMC_LICENSE_STORE_PASSWORD:' "$COMPOSE_FILE" ||
  fail "release app compose does not inject the TrueLicense store password"
grep -Fq 'license/keystore/omcPublicKey.store' "$BUILD_SCRIPT" ||
  fail "release build does not package the TrueLicense keystore"
grep -Fq 'LICENSE_KEYSTORE_SHA256=' "$INSTALL_SCRIPT" ||
  fail "installer does not verify the packaged TrueLicense keystore"
grep -Fq 'install -d -m 0755 "$OMC_ROOT/etc/license"' "$INSTALL_SCRIPT" ||
  fail "installer does not provision the external License directory"

echo "PASS: release package provisions and mounts the external TrueLicense keystore"

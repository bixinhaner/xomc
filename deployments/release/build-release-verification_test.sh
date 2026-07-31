#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if ! grep -Fq 'validate-tempo-memory-budget.sh' "$SCRIPT_DIR/build-release.sh"; then
  echo "FAIL: release verification gate must enforce the Tempo memory budget contract" >&2
  exit 1
fi

if ! grep -Fq -- '--no-xattrs' "$SCRIPT_DIR/build-release.sh"; then
  echo "FAIL: release archives must omit host extended attributes for portable Linux extraction" >&2
  exit 1
fi

if ! bash "$SCRIPT_DIR/build-release.sh" --verify-only >/dev/null 2>&1; then
  echo "FAIL: --verify-only must execute the default release verification gate" >&2
  exit 1
fi

if NATS_SERVER_BIN=/bin/false bash "$SCRIPT_DIR/build-release.sh" --verify-only >/dev/null 2>&1; then
  echo "FAIL: verification failure must fail the release build entry" >&2
  exit 1
fi

echo "PASS: release build executes and enforces the verification gate"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/build-release.sh"

grep -Fq 'git rev-parse HEAD' "$BUILD_SCRIPT"
if grep -Fq 'git rev-parse --short HEAD' "$BUILD_SCRIPT"; then
  echo "release campaign identity must use the full immutable commit" >&2
  exit 1
fi

echo "release commit identity contract: PASS"

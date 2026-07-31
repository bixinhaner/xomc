#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/build-release.sh"
SERVE_SCRIPT="$SCRIPT_DIR/serve.sh"

# macOS /bin/bash 3.2 在 set -u 下展开空数组 "${array[@]}" 会报 unbound variable。
# `${array[@]+"${array[@]}"}` 在 Bash 3.2/4+/5+ 均能保持“空数组不传参数”的语义。
if grep -Fq '${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"}' "$BUILD_SCRIPT"; then
  echo "PASS: build-release empty EXTRA_ARGS is Bash 3.2 safe"
else
  echo "FAIL: build-release expands empty EXTRA_ARGS unsafely under Bash 3.2" >&2
  exit 1
fi

if grep -q 'date -Is' "$BUILD_SCRIPT"; then
  echo "FAIL: build-release uses GNU-only date -Is" >&2
  exit 1
fi
echo "PASS: build-release timestamp is BSD/GNU date compatible"

if grep -Fq "stamp=\"\$(date '+%Y-%m-%dT%H:%M:%S%z')\"" "$BUILD_SCRIPT"; then
  echo "PASS: build-release timestamp uses local timezone"
else
  echo "FAIL: build-release timestamp is not local-timezone based" >&2
  exit 1
fi

if LC_ALL=C grep -Eq '\$[A-Za-z_][A-Za-z0-9_]*[^ -~]' "$BUILD_SCRIPT"; then
  echo "FAIL: unbraced variable directly precedes non-ASCII punctuation" >&2
  exit 1
fi
echo "PASS: build-release variables before Chinese punctuation are braced"

if LC_ALL=C grep -Eq '\$[A-Za-z_][A-Za-z0-9_]*[^ -~]' "$SERVE_SCRIPT"; then
  echo "FAIL: serve.sh has an unbraced variable directly before non-ASCII punctuation" >&2
  exit 1
fi
echo "PASS: serve.sh variables before Chinese punctuation are braced"

if grep -Fq -- '--platform "linux/$ARCH"' "$BUILD_SCRIPT"; then
  echo "PASS: docker build pins the requested target architecture"
else
  echo "FAIL: docker build does not pass the requested --platform" >&2
  exit 1
fi

if grep -Fq 'export COPYFILE_DISABLE=1' "$BUILD_SCRIPT" &&
   grep -Fq "find \"\$STAGE\" -type f -name '._*' -delete" "$BUILD_SCRIPT"; then
  echo "PASS: release package excludes macOS AppleDouble metadata"
else
  echo "FAIL: release package may contain macOS AppleDouble metadata" >&2
  exit 1
fi

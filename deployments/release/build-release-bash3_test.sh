#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/build-release.sh"

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

if LC_ALL=C grep -Eq '\$[A-Za-z_][A-Za-z0-9_]*[^ -~]' "$BUILD_SCRIPT"; then
  echo "FAIL: unbraced variable directly precedes non-ASCII punctuation" >&2
  exit 1
fi
echo "PASS: build-release variables before Chinese punctuation are braced"

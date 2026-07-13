#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=data-upgrade-lib.sh
source "$SCRIPT_DIR/data-upgrade-lib.sh"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

printf '75-default\n' > "$tmp/previous.xml"
printf '167-custom\n' > "$tmp/live.xml"
printf '80-new-default\n' > "$tmp/new.xml"

if should_refresh_builtin "$tmp/new.xml" "$tmp/live.xml" "$tmp/previous.xml"; then
  echo "FAIL: operator-modified builtin must be preserved" >&2
  exit 1
fi

cp "$tmp/previous.xml" "$tmp/live.xml"
should_refresh_builtin "$tmp/new.xml" "$tmp/live.xml" "$tmp/previous.xml" || {
  echo "FAIL: untouched old builtin must refresh" >&2
  exit 1
}

should_refresh_builtin "$tmp/new.xml" "$tmp/live.xml" "$tmp/missing.xml" || {
  echo "FAIL: missing previous baseline must keep legacy refresh behavior" >&2
  exit 1
}

echo "PASS: data upgrade builtin preservation"

#!/usr/bin/env bash
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/gpv-handoff-lib.sh"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
CALLS="$TMP/calls"

cat >"$TMP/compose" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$GPV_HANDOFF_TEST_CALLS"
exit "${GPV_HANDOFF_TEST_EXIT:-0}"
SH
chmod +x "$TMP/compose"

DC=("$TMP/compose")
export GPV_HANDOFF_TEST_CALLS="$CALLS"

GPV_HANDOFF_TEST_EXIT=0
export GPV_HANDOFF_TEST_EXIT
gpv_handoff_prepare
grep -Fq 'run --rm --no-deps gpv-handoff' "$CALLS" || {
  echo "FAIL: handoff must run the one-shot management container" >&2
  exit 1
}

GPV_HANDOFF_TEST_EXIT=17
export GPV_HANDOFF_TEST_EXIT
if gpv_handoff_prepare; then
  echo "FAIL: handoff failure must propagate before app restart" >&2
  exit 1
fi

echo "PASS: GPV handoff gate propagates failure before app restart"

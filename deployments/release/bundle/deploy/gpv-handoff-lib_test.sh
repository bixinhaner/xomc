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
grep -Fxq 'run --rm --no-deps gpv-handoff --config /etc/omcgo/app.prod.yaml' "$CALLS" || {
  echo "FAIL: handoff must run the one-shot management container" >&2
  exit 1
}

gpv_handoff_prepare --fresh-install
grep -Fxq 'run --rm --no-deps gpv-handoff --config /etc/omcgo/app.prod.yaml --fresh-install' "$CALLS" || {
  echo "FAIL: explicit fresh install marker must reach the management container" >&2
  exit 1
}

GPV_HANDOFF_TEST_EXIT=17
export GPV_HANDOFF_TEST_EXIT
if gpv_handoff_prepare; then
  echo "FAIL: handoff failure must propagate before app restart" >&2
  exit 1
fi

FRESH_ROOT="$TMP/fresh-root"
mkdir -p "$FRESH_ROOT"
gpv_handoff_is_fresh_install "$FRESH_ROOT" || {
  echo "FAIL: root without current or saved environment must be explicitly fresh" >&2
  exit 1
}
mkdir -p "$FRESH_ROOT/etc"
: >"$FRESH_ROOT/etc/.env.saved"
if gpv_handoff_is_fresh_install "$FRESH_ROOT"; then
  echo "FAIL: saved environment means reinstall, not fresh install" >&2
  exit 1
fi
rm -f "$FRESH_ROOT/etc/.env.saved"
mkdir -p "$FRESH_ROOT/current/deploy"
: >"$FRESH_ROOT/current/deploy/.env"
if gpv_handoff_is_fresh_install "$FRESH_ROOT"; then
  echo "FAIL: current environment means upgrade, not fresh install" >&2
  exit 1
fi

for action in start up restart stop down; do
  gpv_handoff_action_touches_app "$action" || {
    echo "FAIL: $action without targets must protect app" >&2
    exit 1
  }
done
for action in start up restart stop; do
  gpv_handoff_action_touches_app "$action" worker app || {
    echo "FAIL: $action app must protect app" >&2
    exit 1
  }
  if gpv_handoff_action_touches_app "$action" worker; then
    echo "FAIL: $action worker alone must not run GPV handoff" >&2
    exit 1
  fi
done

echo "PASS: GPV handoff gate propagates failure before app restart"

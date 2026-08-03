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
printf '%s\n' '{"Created":false}'
exit "${GPV_HANDOFF_TEST_EXIT:-0}"
SH
chmod +x "$TMP/compose"

DC=("$TMP/compose")
export GPV_HANDOFF_TEST_CALLS="$CALLS"

GPV_HANDOFF_TEST_EXIT=0
export GPV_HANDOFF_TEST_EXIT
if gpv_handoff_prepare >"$TMP/success.out" 2>"$TMP/success.err"; then
  [ ! -s "$TMP/success.out" ] || { echo "FAIL: successful handoff must hide command output" >&2; exit 1; }
  [ ! -s "$TMP/success.err" ] || { echo "FAIL: successful handoff must not emit stderr" >&2; exit 1; }
else
  echo "FAIL: successful handoff returned failure" >&2
  exit 1
fi
grep -Fxq 'run --rm --no-deps gpv-handoff --config /etc/omcgo/app.prod.yaml' "$CALLS" || {
  echo "FAIL: handoff must run the one-shot management container" >&2
  exit 1
}

gpv_handoff_prepare --fresh-install
grep -Fxq 'run --rm --no-deps gpv-handoff --config /etc/omcgo/app.prod.yaml --fresh-install' "$CALLS" || {
  echo "FAIL: explicit fresh install marker must reach the management container" >&2
  exit 1
}

gpv_handoff_prepare --bootstrap-if-missing
grep -Fxq 'run --rm --no-deps gpv-handoff --config /etc/omcgo/app.prod.yaml --bootstrap-if-missing' "$CALLS" || {
  echo "FAIL: safe bootstrap-if-missing marker must reach the management container" >&2
  exit 1
}

GPV_HANDOFF_TEST_EXIT=17
export GPV_HANDOFF_TEST_EXIT
if gpv_handoff_prepare >"$TMP/failure.out" 2>"$TMP/failure.err"; then
  echo "FAIL: handoff failure must propagate before app restart" >&2
  exit 1
fi
grep -Fxq '{"Created":false}' "$TMP/failure.err" || {
  echo "FAIL: failed handoff must preserve command output" >&2
  exit 1
}

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

SYSTEMD_BIN="$TMP/systemd-bin"
SYSTEMD_UNITS="$TMP/systemd-units"
SYSTEMD_IMAGES="$TMP/systemd-images"
SYSTEMD_CONFIG="$TMP/app.prod.yaml"
SYSTEMD_CALLS="$TMP/systemd-calls"
mkdir -p "$SYSTEMD_BIN" "$SYSTEMD_UNITS" "$SYSTEMD_IMAGES"
: >"$SYSTEMD_CONFIG"
cat >"$SYSTEMD_BIN/docker" <<'SH'
#!/usr/bin/env bash
printf 'docker %s\n' "$*" >>"${GPV_SYSTEMD_TEST_CALLS:?}"
case "${1:-} ${2:-}" in
  "image inspect") exit 0 ;;
  "run --rm") exit "${GPV_SYSTEMD_HANDOFF_EXIT:-0}" ;;
esac
exit 0
SH
cat >"$SYSTEMD_BIN/systemctl" <<'SH'
#!/usr/bin/env bash
printf 'systemctl %s\n' "$*" >>"${GPV_SYSTEMD_TEST_CALLS:?}"
case "${1:-}" in
  list-unit-files) exit 0 ;;
  is-active)
    [ "${3:-}" = "omcgo-app" ] && [ "${GPV_SYSTEMD_APP_ACTIVE:-0}" = "1" ]
    exit
    ;;
esac
exit 0
SH
chmod +x "$SYSTEMD_BIN/docker" "$SYSTEMD_BIN/systemctl"

reset_systemd_units() {
  rm -f "$SYSTEMD_UNITS"/*
  for service in omcgo-app omcgo-acs omcgo-worker; do
    : >"$SYSTEMD_UNITS/$service.service"
  done
  : >"$SYSTEMD_CALLS"
}

export GPV_SYSTEMD_TEST_CALLS="$SYSTEMD_CALLS"
export GPV_SYSTEMD_APP_ACTIVE=1
export GPV_SYSTEMD_UNIT_DIR="$SYSTEMD_UNITS"
export GPV_SYSTEMD_HANDOFF_EXIT=0
reset_systemd_units
PATH="$SYSTEMD_BIN:$PATH" gpv_handoff_migrate_legacy_systemd \
  "omcgo/app:test" "$SYSTEMD_IMAGES" "$SYSTEMD_CONFIG" || {
    echo "FAIL: active legacy systemd app must migrate after successful handoff" >&2
    exit 1
  }
handoff_line="$(grep -n '^docker run --rm ' "$SYSTEMD_CALLS" | cut -d: -f1)"
first_stop_line="$(grep -n '^systemctl stop ' "$SYSTEMD_CALLS" | head -1 | cut -d: -f1)"
[ -n "$handoff_line" ] && [ -n "$first_stop_line" ] && [ "$handoff_line" -lt "$first_stop_line" ] || {
  echo "FAIL: handoff must complete before any legacy service is stopped" >&2
  exit 1
}
grep -Fq -- '--network host' "$SYSTEMD_CALLS" &&
  grep -Fq -- 'OMCGO_NATS_URL=nats://127.0.0.1:4222' "$SYSTEMD_CALLS" &&
  grep -Fq -- '--entrypoint omcgo-gpv-handoff omcgo/app:test' "$SYSTEMD_CALLS" || {
    echo "FAIL: systemd migration must use the release handoff tool against host NATS" >&2
    exit 1
  }
actual_stop_order="$(grep '^systemctl stop ' "$SYSTEMD_CALLS" | sed 's/^systemctl stop //' | tr '\n' ' ')"
[ "$actual_stop_order" = "omcgo-acs omcgo-worker omcgo-app " ] || {
  echo "FAIL: producers and worker must stop before the legacy app: $actual_stop_order" >&2
  exit 1
}

export GPV_SYSTEMD_HANDOFF_EXIT=29
reset_systemd_units
if PATH="$SYSTEMD_BIN:$PATH" gpv_handoff_migrate_legacy_systemd \
  "omcgo/app:test" "$SYSTEMD_IMAGES" "$SYSTEMD_CONFIG"; then
  echo "FAIL: failed systemd handoff must abort migration" >&2
  exit 1
fi
if grep -Eq '^systemctl (stop|disable) ' "$SYSTEMD_CALLS"; then
  echo "FAIL: failed handoff must not stop or disable any legacy service" >&2
  exit 1
fi

echo "PASS: GPV handoff gate propagates failure before app restart"

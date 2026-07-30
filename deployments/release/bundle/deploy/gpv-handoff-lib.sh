#!/usr/bin/env bash

# gpv_handoff_prepare pre-creates the fixed GPV RPC durable while the old app
# is still running. The new durable starts at the old consumer's AckFloor+1,
# so every message published during the stop/start window is retained.
#
# The caller owns DC=(docker compose ...). Any non-zero result is a hard gate:
# callers must return before stopping or recreating the old app.
gpv_handoff_is_fresh_install() {
  local omc_root="$1"
  [ ! -f "$omc_root/current/deploy/.env" ] &&
    [ ! -f "$omc_root/etc/.env.saved" ]
}

gpv_handoff_action_touches_app() {
  local action="$1"
  shift
  case "$action" in
    down) return 0 ;;
    start|up|stop|restart)
      [ $# -eq 0 ] && return 0
      local target
      for target in "$@"; do
        [ "$target" = "app" ] && return 0
      done
      ;;
  esac
  return 1
}

gpv_handoff_prepare() {
  local args=(--config /etc/omcgo/app.prod.yaml)
  if [ "${1:-}" = "--fresh-install" ]; then
    args+=(--fresh-install)
  elif [ $# -gt 0 ]; then
    echo "unsupported gpv_handoff_prepare argument: $1" >&2
    return 2
  fi
  "${DC[@]}" run --rm --no-deps gpv-handoff "${args[@]}"
}

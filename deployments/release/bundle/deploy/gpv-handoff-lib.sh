#!/usr/bin/env bash

# gpv_handoff_prepare pre-creates the fixed GPV RPC durable while the old app
# is still running. The new durable starts at the old consumer's AckFloor+1,
# so every message published during the stop/start window is retained.
#
# The caller owns DC=(docker compose ...). Any non-zero result is a hard gate:
# callers must return before stopping or recreating the old app.
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

gpv_handoff_env_get() {
  local env_file="$1" key="$2"
  [ -f "$env_file" ] || return 1
  awk -v key="$key" '
    index($0, key "=") == 1 {
      print substr($0, length(key) + 2)
      found=1
      exit
    }
    END { if (!found) exit 1 }
  ' "$env_file"
}

gpv_handoff_ensure_image() {
  local image="$1" images_dir="$2" archive loaded=0
  [ -n "$image" ] || {
    echo "GPV systemd handoff image is empty" >&2
    return 1
  }
  docker image inspect "$image" >/dev/null 2>&1 && return 0
  for archive in "$images_dir"/*.tar; do
    [ -f "$archive" ] || continue
    docker load -i "$archive" || return 1
    loaded=1
    docker image inspect "$image" >/dev/null 2>&1 && return 0
  done
  [ "$loaded" = 1 ] &&
    echo "loaded release image archives but GPV handoff image is still missing: $image" >&2
  [ "$loaded" = 0 ] &&
    echo "no release image archive found for GPV systemd handoff: $images_dir" >&2
  return 1
}

gpv_handoff_prepare_systemd() {
  local image="$1" config_path="$2"
  [ -f "$config_path" ] || {
    echo "legacy app config is required before GPV systemd handoff: $config_path" >&2
    return 1
  }
  docker run --rm --network host \
    -e "OMCGO_NATS_URL=${GPV_SYSTEMD_NATS_URL:-nats://127.0.0.1:4222}" \
    -e "GPV_RPC_SOURCE_CONSUMER=${GPV_RPC_SOURCE_CONSUMER:-}" \
    -v "$config_path:/etc/omcgo/app.prod.yaml:ro" \
    --entrypoint omcgo-gpv-handoff \
    "$image" \
    --config /etc/omcgo/app.prod.yaml
}

gpv_handoff_systemd_unit_exists() {
  local service="$1" unit_dir="${GPV_SYSTEMD_UNIT_DIR:-/etc/systemd/system}"
  [ -f "$unit_dir/$service.service" ] ||
    systemctl list-unit-files "$service.service" --no-legend 2>/dev/null |
      awk -v unit="$service.service" '$1 == unit { found=1 } END { exit !found }'
}

gpv_handoff_migrate_legacy_systemd() {
  local image="$1" images_dir="$2" config_path="$3"
  local unit_dir="${GPV_SYSTEMD_UNIT_DIR:-/etc/systemd/system}"
  local service unit backup has_old=0 has_active=0
  local services=(omcgo-acs omcgo-worker omcgo-app)

  command -v systemctl >/dev/null 2>&1 || return 0
  for service in "${services[@]}"; do
    if gpv_handoff_systemd_unit_exists "$service"; then
      has_old=1
      if systemctl is-active --quiet "$service"; then
        has_active=1
      fi
    fi
  done
  [ "$has_old" = 1 ] || return 0

  if [ "$has_active" = 1 ]; then
    gpv_handoff_ensure_image "$image" "$images_dir" || return 1
    gpv_handoff_prepare_systemd "$image" "$config_path" || return 1
    GPV_SYSTEMD_HANDOFF_PREPARED=1
  fi

  for service in "${services[@]}"; do
    gpv_handoff_systemd_unit_exists "$service" || continue
    systemctl stop "$service" 2>/dev/null || true
    systemctl disable "$service" 2>/dev/null || true
    unit="$unit_dir/$service.service"
    if [ -f "$unit" ]; then
      backup="$unit.bak.$(date +%Y%m%d%H%M%S)"
      mv "$unit" "$backup" || return 1
    fi
  done
  systemctl daemon-reload
}

gpv_handoff_prepare() {
  local args=(--config /etc/omcgo/app.prod.yaml)
  local output status
  if [ "${1:-}" = "--fresh-install" ] ||
     [ "${1:-}" = "--bootstrap-if-missing" ]; then
    args+=("$1")
  elif [ $# -gt 0 ]; then
    echo "unsupported gpv_handoff_prepare argument: $1" >&2
    return 2
  fi
  if output="$("${DC[@]}" run --rm --no-deps gpv-handoff "${args[@]}" 2>&1)"; then
    return 0
  else
    status=$?
    printf '%s\n' "$output" >&2
    return "$status"
  fi
}

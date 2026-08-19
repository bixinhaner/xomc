#!/bin/sh
set -eu

LOKI_URL=${LOKI_URL:-http://loki:3100}
LOKI_DATA_DIR=${LOKI_DATA_DIR:-/loki}
LOKI_MAX_BYTES=${LOKI_MAX_BYTES:-10737418240}
LOKI_RETENTION_SECONDS=${LOKI_RETENTION_SECONDS:-604800}
LOKI_CHECK_INTERVAL_SECONDS=${LOKI_CHECK_INTERVAL_SECONDS:-600}
LOKI_DELETE_COOLDOWN_SECONDS=${LOKI_DELETE_COOLDOWN_SECONDS:-7200}
LOKI_STATE_FILE=${LOKI_STATE_FILE:-$LOKI_DATA_DIR/compactor/size-retention.state}
LOKI_DELETE_SELECTOR=${LOKI_DELETE_SELECTOR:-%7Bjob%3D~%22.%2B%22%7D}
LOKI_TIMEZONE_FILE=${LOKI_TIMEZONE_FILE:-/var/lib/omcgo/timezone/system-timezone}

DAY_SECONDS=86400
CURRENT_TIMEZONE=SYSTEM

date_current() {
  if [ "$CURRENT_TIMEZONE" = SYSTEM ]; then
    date "$@"
  else
    TZ="$CURRENT_TIMEZONE" date "$@"
  fi
}

log() {
  printf '%s loki-size-retention: %s\n' "$(date_current +%Y-%m-%dT%H:%M:%S%z)" "$*" >&2
}

is_uint() {
  case "$1" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

for value in "$LOKI_MAX_BYTES" "$LOKI_RETENTION_SECONDS" \
  "$LOKI_CHECK_INTERVAL_SECONDS" "$LOKI_DELETE_COOLDOWN_SECONDS"; do
  is_uint "$value" || {
    log "invalid numeric configuration: $value"
    exit 1
  }
done

mkdir -p "$(dirname "$LOKI_STATE_FILE")"

timezone_is_available() {
  case "$1" in
    ''|/*|*..*|*[!A-Za-z0-9_+./-]*) return 1 ;;
    SYSTEM) return 0 ;;
    UTC) return 0 ;;
    *) [ -f "/usr/share/zoneinfo/$1" ] ;;
  esac
}

load_timezone() {
  candidate=SYSTEM
  if [ -f "$LOKI_TIMEZONE_FILE" ]; then
    candidate=$(sed -n '1p' "$LOKI_TIMEZONE_FILE" 2>/dev/null | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//') || candidate=
    [ -n "$candidate" ] || candidate=SYSTEM
  fi
  if ! timezone_is_available "$candidate"; then
    invalid_timezone=$candidate
    candidate=SYSTEM
  fi
  timezone_changed=0
  [ "$candidate" = "$CURRENT_TIMEZONE" ] || timezone_changed=1
  CURRENT_TIMEZONE=$candidate
  [ -n "${invalid_timezone:-}" ] || invalid_timezone=
  [ -n "$invalid_timezone" ] && log "invalid OMC timezone ${invalid_timezone}; using system timezone"
  [ "$timezone_changed" = 0 ] || log "using timezone=$CURRENT_TIMEZONE"
}

local_midnight_for_epoch() {
  local_epoch=$1
  local_date=$(date_current -d "@$local_epoch" +%Y-%m-%d) || return 1
  date_current -d "$local_date 00:00:00" +%s
}

next_local_midnight() {
  cursor_epoch=$1
  next_date=$(date_current -d "@$((cursor_epoch + DAY_SECONDS))" +%Y-%m-%d) || return 1
  date_current -d "$next_date 00:00:00" +%s
}

load_state() {
  CURSOR=0
  LAST_REQUEST=0
  SAVED_TIMEZONE=

  if [ -r "$LOKI_STATE_FILE" ]; then
    while IFS='=' read -r key value; do
      case "$key" in
        cursor) CURSOR=$value ;;
        last_request) LAST_REQUEST=$value ;;
        timezone) SAVED_TIMEZONE=$value ;;
      esac
    done < "$LOKI_STATE_FILE"
  fi

  is_uint "$CURSOR" || CURSOR=0
  is_uint "$LAST_REQUEST" || LAST_REQUEST=0
  [ "$SAVED_TIMEZONE" = "$CURRENT_TIMEZONE" ] || CURSOR=0
}

save_state() {
  state_tmp="$LOKI_STATE_FILE.tmp"
  printf 'cursor=%s\nlast_request=%s\ntimezone=%s\n' \
    "$CURSOR" "$LAST_REQUEST" "$CURRENT_TIMEZONE" >"$state_tmp"
  mv "$state_tmp" "$LOKI_STATE_FILE"
}

storage_bytes() {
  du -sb "$LOKI_DATA_DIR" 2>/dev/null | awk 'NR == 1 { print $1; exit }'
}

run_once() {
  load_timezone || return 1
  now=$(date_current +%s)
  size=$(storage_bytes || true)
  if ! is_uint "$size"; then
    log "unable to read storage usage under $LOKI_DATA_DIR"
    return 1
  fi

  load_state
  oldest_day=$(local_midnight_for_epoch "$((now - LOKI_RETENTION_SECONDS))") || {
    log "unable to calculate local midnight for timezone=$CURRENT_TIMEZONE"
    return 1
  }
  if [ "$CURSOR" -lt "$oldest_day" ]; then
    CURSOR=$oldest_day
  fi

  if [ "$size" -le "$LOKI_MAX_BYTES" ]; then
    log "storage=${size} bytes, limit=${LOKI_MAX_BYTES}; no capacity cleanup needed"
    return 0
  fi

  if [ $((now - LAST_REQUEST)) -lt "$LOKI_DELETE_COOLDOWN_SECONDS" ]; then
    log "storage=${size} bytes exceeds limit; waiting for the previous deletion request"
    return 0
  fi

  today_start=$(local_midnight_for_epoch "$now") || {
    log "unable to calculate today's local midnight for timezone=$CURRENT_TIMEZONE"
    return 1
  }
  if [ "$CURSOR" -ge "$today_start" ]; then
    log "storage=${size} bytes exceeds limit, but no complete old local date is available"
    return 0
  fi

  delete_end=$(next_local_midnight "$CURSOR") || {
    log "unable to calculate next local midnight for timezone=$CURRENT_TIMEZONE"
    return 1
  }
  [ "$delete_end" -le "$today_start" ] || delete_end=$today_start

  delete_url="$LOKI_URL/loki/api/v1/delete?query=$LOKI_DELETE_SELECTOR&start=$CURSOR&end=$delete_end&max_interval=24h"
  if wget -q -T 15 -O /dev/null --post-data='' "$delete_url"; then
    log "storage=${size} bytes exceeds limit; requested deletion for [$CURSOR,$delete_end)"
    CURSOR=$delete_end
    LAST_REQUEST=$now
    save_state
  else
    log "failed to submit deletion request for [$CURSOR,$delete_end)"
    return 1
  fi
}

while :; do
  run_once || true
  [ "${LOKI_ONCE:-0}" = 1 ] && exit 0
  sleep "$LOKI_CHECK_INTERVAL_SECONDS"
done

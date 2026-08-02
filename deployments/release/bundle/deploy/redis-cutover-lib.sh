#!/usr/bin/env bash

LEGACY_REDIS_CUTOVER=0
LEGACY_REDIS_KEY_COUNT=""
LEGACY_REDIS_DATA_SOURCE=""

redis_cutover_state_path() {
  printf '%s\n' "${REDIS_CUTOVER_STATE_FILE:-${OMC_ROOT}/data/.redis-cutover-state}"
}

persist_redis_cutover_state() {
  local status="$1" state_file state_dir tmp
  state_file="$(redis_cutover_state_path)" || return 1
  state_dir="$(dirname "$state_file")"
  mkdir -p "$state_dir" || return 1
  tmp="$(mktemp "${state_file}.tmp.XXXXXX")" || return 1
  if ! printf 'status=%s\nkey_count=%s\ndata_source=%s\n' \
    "$status" "$LEGACY_REDIS_KEY_COUNT" "$LEGACY_REDIS_DATA_SOURCE" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  chmod 600 "$tmp" || {
    rm -f "$tmp"
    return 1
  }
  if ! mv -f "$tmp" "$state_file"; then
    rm -f "$tmp"
    return 1
  fi
}

recover_pending_redis_cutover() {
  local state_file status
  state_file="$(redis_cutover_state_path)" || return 1
  [ -f "$state_file" ] || return 2
  status="$(awk -F= '$1=="status" { print substr($0, index($0,"=")+1); exit }' "$state_file")"
  case "$status" in
    completed) return 2 ;;
    pending) ;;
    *) return 1 ;;
  esac
  LEGACY_REDIS_KEY_COUNT="$(awk -F= '$1=="key_count" { print substr($0, index($0,"=")+1); exit }' "$state_file")"
  LEGACY_REDIS_DATA_SOURCE="$(awk -F= '$1=="data_source" { print substr($0, index($0,"=")+1); exit }' "$state_file")"
  case "$LEGACY_REDIS_KEY_COUNT" in
    ''|*[!0-9]*) return 1 ;;
  esac
  case "$LEGACY_REDIS_DATA_SOURCE" in
    /*) ;;
    *) return 1 ;;
  esac
  LEGACY_REDIS_CUTOVER=1
}

compose_service_container_ids() {
  local service="$1"
  docker ps -q \
    --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" \
    --filter "label=com.docker.compose.service=${service}"
}

legacy_redis_container_ids() {
  docker ps -aq \
    --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" \
    --filter "label=com.docker.compose.service=redis"
}

redis_data_mount_source() {
  docker inspect -f '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Source}}{{end}}{{end}}' "$1"
}

legacy_redis_is_running() {
  [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null)" = "true" ]
}

# prepare_legacy_redis_cutover stops every service that can write Redis before
# removing the single legacy container. It captures the exact mount and key
# count so the new redis-core can be verified before writers restart.
prepare_legacy_redis_cutover() {
  local legacy_ids legacy_id service writer_id
  legacy_ids="$(legacy_redis_container_ids)" || return 1
  if [ -z "$legacy_ids" ]; then
    if recover_pending_redis_cutover; then
      return 0
    else
      [ "$?" -eq 2 ] && return 0
      return 1
    fi
  fi
  [ "$(printf '%s\n' "$legacy_ids" | awk 'NF { n++ } END { print n+0 }')" -eq 1 ] || return 1
  legacy_id="$legacy_ids"

  # An earlier installer may have persisted pending evidence and stopped the
  # legacy container, then died before docker rm. A stopped Redis cannot answer
  # DBSIZE; recover the already durable evidence and continue the exact remove
  # boundary instead of making the upgrade permanently unretryable.
  if ! legacy_redis_is_running "$legacy_id"; then
    if ! recover_pending_redis_cutover; then
      return 1
    fi
    docker rm "$legacy_id" >/dev/null || return 1
    return 0
  fi

  for service in app worker acs acs-candidate; do
    for writer_id in $(compose_service_container_ids "$service"); do
      docker stop --time 60 "$writer_id" >/dev/null || return 1
    done
  done
  LEGACY_REDIS_KEY_COUNT="$(docker exec "$legacy_id" redis-cli --raw DBSIZE)" || return 1
  LEGACY_REDIS_DATA_SOURCE="$(redis_data_mount_source "$legacy_id")" || return 1
  [ -n "$LEGACY_REDIS_DATA_SOURCE" ] || return 1
  persist_redis_cutover_state pending || return 1
  docker stop --time 30 "$legacy_id" >/dev/null || return 1
  docker rm "$legacy_id" >/dev/null || return 1
  LEGACY_REDIS_CUTOVER=1
}

verify_legacy_redis_cutover() {
  local core_id core_keys core_source
  [ "$LEGACY_REDIS_CUTOVER" = 1 ] || return 0
  core_id="$("${DC[@]}" ps -q redis-core 2>/dev/null | head -n1)"
  [ -n "$core_id" ] || return 1
  core_keys="$(docker exec "$core_id" redis-cli --raw DBSIZE)" || return 1
  core_source="$(redis_data_mount_source "$core_id")" || return 1
  [ "$core_source" = "$LEGACY_REDIS_DATA_SOURCE" ] || return 1
  [ "$core_keys" = "$LEGACY_REDIS_KEY_COUNT" ] || return 1
}

migrate_legacy_pm_redis() {
  [ "$LEGACY_REDIS_CUTOVER" = 1 ] || return 0
  "${DC[@]}" --profile operations run --rm --no-deps pm-redis-migrate \
    --source redis-core:6379 --target redis-pm:6379 \
    --pattern 'pmagg:*' --scan-count 500 --pipeline-size 100 --output json || return 1
  persist_redis_cutover_state completed
}

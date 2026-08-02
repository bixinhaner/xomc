#!/usr/bin/env bash

LEGACY_REDIS_CUTOVER=0
LEGACY_REDIS_KEY_COUNT=""
LEGACY_REDIS_DATA_SOURCE=""

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

# prepare_legacy_redis_cutover stops every service that can write Redis before
# removing the single legacy container. It captures the exact mount and key
# count so the new redis-core can be verified before writers restart.
prepare_legacy_redis_cutover() {
  local legacy_ids legacy_id service writer_id
  legacy_ids="$(legacy_redis_container_ids)" || return 1
  [ -n "$legacy_ids" ] || return 0
  [ "$(printf '%s\n' "$legacy_ids" | awk 'NF { n++ } END { print n+0 }')" -eq 1 ] || return 1
  legacy_id="$legacy_ids"

  LEGACY_REDIS_KEY_COUNT="$(docker exec "$legacy_id" redis-cli --raw DBSIZE)" || return 1
  LEGACY_REDIS_DATA_SOURCE="$(redis_data_mount_source "$legacy_id")" || return 1
  [ -n "$LEGACY_REDIS_DATA_SOURCE" ] || return 1

  for service in app worker acs acs-candidate; do
    for writer_id in $(compose_service_container_ids "$service"); do
      docker stop --time 60 "$writer_id" >/dev/null || return 1
    done
  done
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
    --pattern 'pmagg:*' --scan-count 500 --pipeline-size 100 --output json
}

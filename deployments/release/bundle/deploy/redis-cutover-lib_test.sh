#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
OMC_ROOT="$tmp/omc"
mkdir -p "$OMC_ROOT/data"
COMPOSE_PROJECT=omcgo
DC=(mock-compose)
CALL_LOG="$tmp/docker-calls"
legacy_present=1
legacy_running=1
migration_should_fail=1

docker() {
  printf '%s\n' "$*" >> "$CALL_LOG"
  case "$*" in
    "ps -aq --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=redis")
      if [ "$legacy_present" = 1 ]; then echo legacy-redis; fi
      ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=app") echo old-app ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=worker") echo old-worker ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=acs") echo old-acs ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=acs-candidate") : ;;
    "exec legacy-redis redis-cli --raw DBSIZE") echo 42 ;;
    "inspect -f {{.State.Running}} legacy-redis") [ "$legacy_running" = 1 ] && echo true || echo false ;;
    "inspect -f "*" legacy-redis") echo /var/lib/docker/volumes/omcgo_redisdata/_data ;;
    "exec new-core redis-cli --raw DBSIZE") echo 42 ;;
    "inspect -f "*" new-core") echo /var/lib/docker/volumes/omcgo_redisdata/_data ;;
    "stop --time 60 old-app"|"stop --time 60 old-worker"|"stop --time 60 old-acs") : ;;
    "stop --time 30 legacy-redis") legacy_running=0 ;;
    "rm legacy-redis") legacy_present=0 ;;
    *) echo "unexpected docker call: $*" >&2; return 1 ;;
  esac
}

mock-compose() {
  case "$*" in
    "ps -q redis-core") echo new-core ;;
    "--profile operations run --rm --no-deps pm-redis-migrate --source redis-core:6379 --target redis-pm:6379 --pattern pmagg:* --scan-count 500 --pipeline-size 100 --output json")
      [ "$migration_should_fail" = 0 ] || return 72
      echo '{"failed":0,"conflicts":0,"verified":6}'
      ;;
    *) return 1 ;;
  esac
}

# shellcheck source=redis-cutover-lib.sh
source "$SCRIPT_DIR/redis-cutover-lib.sh"
prepare_legacy_redis_cutover
verify_legacy_redis_cutover
if migrate_legacy_pm_redis >/dev/null; then
  echo "FAIL: injected first migration attempt must fail" >&2
  exit 1
fi

grep -q '^status=pending$' "$OMC_ROOT/data/.redis-cutover-state" || {
  echo "FAIL: interrupted cutover did not persist pending state" >&2
  exit 1
}
stop_line="$(grep -n 'stop --time 60 old-app' "$CALL_LOG" | head -n1 | cut -d: -f1)"
dbsize_line="$(grep -n 'exec legacy-redis redis-cli --raw DBSIZE' "$CALL_LOG" | head -n1 | cut -d: -f1)"
[ "$stop_line" -lt "$dbsize_line" ] || {
  echo "FAIL: legacy DBSIZE was sampled before Redis writers stopped" >&2
  exit 1
}

# Simulate a fresh installer process after the legacy container was already
# removed but before PM migration completed. It must recover pending evidence,
# re-verify core continuity, resume the idempotent copy, and persist completed.
LEGACY_REDIS_CUTOVER=0
LEGACY_REDIS_KEY_COUNT=""
LEGACY_REDIS_DATA_SOURCE=""
migration_should_fail=0
prepare_legacy_redis_cutover
[ "$LEGACY_REDIS_CUTOVER" = 1 ]
[ "$LEGACY_REDIS_KEY_COUNT" = 42 ]
[ "$LEGACY_REDIS_DATA_SOURCE" = /var/lib/docker/volumes/omcgo_redisdata/_data ]
verify_legacy_redis_cutover
migrate_legacy_pm_redis >/dev/null
grep -q '^status=completed$' "$OMC_ROOT/data/.redis-cutover-state"

# A crash after docker stop but before docker rm leaves a stopped legacy
# container plus valid pending evidence. A new process must reuse the evidence,
# remove the stopped container, and continue instead of trying redis-cli.
printf 'status=pending\nkey_count=42\ndata_source=/var/lib/docker/volumes/omcgo_redisdata/_data\n' > "$OMC_ROOT/data/.redis-cutover-state"
legacy_present=1
legacy_running=0
LEGACY_REDIS_CUTOVER=0
LEGACY_REDIS_KEY_COUNT=""
LEGACY_REDIS_DATA_SOURCE=""
prepare_legacy_redis_cutover
[ "$legacy_present" = 0 ]
[ "$LEGACY_REDIS_CUTOVER" = 1 ]
verify_legacy_redis_cutover
migrate_legacy_pm_redis >/dev/null

# A later normal upgrade sees completed evidence and must not rerun migration.
LEGACY_REDIS_CUTOVER=0
LEGACY_REDIS_KEY_COUNT=""
LEGACY_REDIS_DATA_SOURCE=""
prepare_legacy_redis_cutover
[ "$LEGACY_REDIS_CUTOVER" = 0 ]

# Corrupted pending evidence must fail closed rather than silently starting
# App/Worker with an unverified endpoint.
printf 'status=pending\nkey_count=invalid\ndata_source=relative/path\n' > "$OMC_ROOT/data/.redis-cutover-state"
if prepare_legacy_redis_cutover; then
  echo "FAIL: malformed Redis cutover evidence must block installation" >&2
  exit 1
fi

echo "PASS: legacy Redis cutover resumes safely after interrupted PM migration"

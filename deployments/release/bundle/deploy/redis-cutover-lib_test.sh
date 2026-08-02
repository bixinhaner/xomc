#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_PROJECT=omcgo
DC=(mock-compose)
calls=""

docker() {
  calls="${calls}${calls:+|}$*"
  case "$*" in
    "ps -aq --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=redis") echo legacy-redis ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=app") echo old-app ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=worker") echo old-worker ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=acs") echo old-acs ;;
    "ps -q --filter label=com.docker.compose.project=omcgo --filter label=com.docker.compose.service=acs-candidate") : ;;
    "exec legacy-redis redis-cli --raw DBSIZE") echo 42 ;;
    "inspect -f "*" legacy-redis") echo /var/lib/docker/volumes/omcgo_redisdata/_data ;;
    "exec new-core redis-cli --raw DBSIZE") echo 42 ;;
    "inspect -f "*" new-core") echo /var/lib/docker/volumes/omcgo_redisdata/_data ;;
    "stop --time 60 old-app"|"stop --time 60 old-worker"|"stop --time 60 old-acs"|"stop --time 30 legacy-redis"|"rm legacy-redis") : ;;
    *) echo "unexpected docker call: $*" >&2; return 1 ;;
  esac
}

mock-compose() {
  case "$*" in
    "ps -q redis-core") echo new-core ;;
    "--profile operations run --rm --no-deps pm-redis-migrate --source redis-core:6379 --target redis-pm:6379 --pattern pmagg:* --scan-count 500 --pipeline-size 100 --output json") echo '{"failed":0,"conflicts":0,"verified":6}' ;;
    *) return 1 ;;
  esac
}

# shellcheck source=redis-cutover-lib.sh
source "$SCRIPT_DIR/redis-cutover-lib.sh"
prepare_legacy_redis_cutover
verify_legacy_redis_cutover
migrate_legacy_pm_redis >/dev/null

case "$calls" in
  *"stop --time 60 old-app"*"stop --time 60 old-worker"*"stop --time 60 old-acs"*"stop --time 30 legacy-redis"*"rm legacy-redis"*) ;;
  *) echo "FAIL: Redis writers were not stopped before legacy removal: $calls" >&2; exit 1 ;;
esac

[ "$LEGACY_REDIS_KEY_COUNT" = 42 ]
[ "$LEGACY_REDIS_DATA_SOURCE" = /var/lib/docker/volumes/omcgo_redisdata/_data ]
echo "PASS: legacy Redis cutover stops writers and verifies mount/key continuity"

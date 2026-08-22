#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/docker-network-lib.sh"

assert_plan() {
  local bip="$1" compose="$2" pool="$3" test_subnet="$4" migration_subnet="$5"
  unset DOCKER_NETWORK_SUBNET DOCKER_NETWORK_GATEWAY DOCKER_COMPOSE_SUBNET
  unset DOCKER_COMPOSE_GATEWAY DOCKER_ADDR_POOL_BASE DOCKER_ADDR_POOL_SIZE
  unset DOCKER_TEST_SUBNET DOCKER_TEST_GATEWAY DOCKER_MIGRATION_SUBNET DOCKER_MIGRATION_GATEWAY
  DOCKER_BIP="$bip"
  docker_network_plan
  [ "$DOCKER_COMPOSE_SUBNET" = "$compose" ]
  [ "$DOCKER_ADDR_POOL_BASE" = "$pool" ]
  [ "$DOCKER_TEST_SUBNET" = "$test_subnet" ]
  [ "$DOCKER_MIGRATION_SUBNET" = "$migration_subnet" ]
}

assert_rejects() {
  local bip="$1"
  if DOCKER_BIP="$bip" docker_network_plan >/dev/null 2>&1; then
    printf 'expected DOCKER_BIP to be rejected: %s\n' "$bip" >&2
    exit 1
  fi
}

if (unset DOCKER_BIP; docker_network_plan >/dev/null 2>&1); then
  printf 'expected missing DOCKER_BIP to be rejected\n' >&2
  exit 1
fi

assert_plan "173.17.0.1/16" "173.18.0.0/16" "173.19.0.0/16" "173.20.0.0/16" "173.21.0.0/16"
assert_plan "10.240.0.1/16" "10.241.0.0/16" "10.242.0.0/16" "10.243.0.0/16" "10.244.0.0/16"
assert_plan "192.168.50.1/24" "192.168.51.0/24" "192.168.52.0/24" "192.168.53.0/24" "192.168.54.0/24"
assert_rejects "172.24.0.1/16"
assert_rejects "10.0.0.1/15"
assert_rejects "10.0.0.0/16"

TMP_ENV="$(mktemp)"
trap 'rm -f "$TMP_ENV"' EXIT
DOCKER_BIP="10.240.0.1/16"
docker_network_plan
printf 'DOCKER_BIP=old\n' > "$TMP_ENV"
docker_network_write_env_file "$TMP_ENV"
. "$TMP_ENV"
[ "$DOCKER_BIP" = "10.240.0.1/16" ]
[ "$DOCKER_COMPOSE_SUBNET" = "10.241.0.0/16" ]
[ "$DOCKER_ADDR_POOL_BASE" = "10.242.0.0/16" ]
[ "$DOCKER_TEST_SUBNET" = "10.243.0.0/16" ]
[ "$DOCKER_MIGRATION_SUBNET" = "10.244.0.0/16" ]

REMOVED_NETWORK=""
docker() {
  case "$1 ${2:-}" in
    info*) return 0 ;;
    "network ls")
      [ "$REMOVED_NETWORK" = "empty-172" ] || printf 'empty-172\n'
      printf '%s\n' active-172 planned
      ;;
    "network inspect")
      case "${3:-}:${5:-}" in
        empty-172:'{{.Name}}') printf 'legacy-172\n' ;;
        empty-172:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.18.0.0/16\n' ;;
        empty-172:'{{len .Containers}}') printf '0\n' ;;
        active-172:'{{.Name}}') printf 'active-172\n' ;;
        active-172:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.19.0.0/16\n' ;;
        active-172:'{{len .Containers}}') printf '1\n' ;;
        planned:'{{.Name}}') printf 'omcgo\n' ;;
        planned:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '%s\n' "$DOCKER_COMPOSE_SUBNET" ;;
        planned:'{{len .Containers}}') printf '0\n' ;;
        *) return 1 ;;
      esac
      ;;
    "network rm")
      REMOVED_NETWORK="$3"
      return 0
      ;;
    *) return 1 ;;
  esac
}

REMOVED_OUTPUT="$(mktemp)"
trap 'rm -f "$TMP_ENV" "$REMOVED_OUTPUT"' EXIT
docker_network_cleanup_unplanned_networks > "$REMOVED_OUTPUT"
[ "$(cat "$REMOVED_OUTPUT")" = $'legacy-172\t172.18.0.0/16' ]
[ "$REMOVED_NETWORK" = "empty-172" ]
[ "$(docker_network_unplanned_networks)" = $'active-172:172.19.0.0/16' ]

echo "docker network cleanup tests passed"

echo "docker network plan tests passed"

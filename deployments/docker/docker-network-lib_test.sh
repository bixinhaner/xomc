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

assert_plan "10.240.0.1/16" "10.241.0.0/16" "10.242.0.0/16" "10.243.0.0/16" "10.244.0.0/16"
assert_plan "192.168.50.1/24" "192.168.51.0/24" "192.168.52.0/24" "192.168.53.0/24" "192.168.54.0/24"
assert_plan "100.64.0.1/16" "100.65.0.0/16" "100.66.0.0/16" "100.67.0.0/16" "100.68.0.0/16"
assert_rejects "172.24.0.1/16"
assert_rejects "173.17.0.1/16"
assert_rejects "10.255.0.1/16"
assert_rejects "192.168.252.1/24"
assert_rejects "10.0.0.1/15"
assert_rejects "10.0.0.0/16"

TMP_ENV="$(mktemp)"
SOURCE_ENV="$(mktemp)"
DAEMON_JSON_TMP="$(mktemp)"
DEFAULT_DAEMON_JSON="$(mktemp)"
INVALID_DAEMON_JSON="$(mktemp)"
trap 'rm -f "$TMP_ENV" "$SOURCE_ENV" "$DAEMON_JSON_TMP" "$DEFAULT_DAEMON_JSON" "$INVALID_DAEMON_JSON"' EXIT
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

unset DOCKER_BIP DOCKER_BIP_SOURCE
DOCKER_DAEMON_JSON="$DEFAULT_DAEMON_JSON"
docker_network_resolve_bip_from_config "$SOURCE_ENV"
[ "$DOCKER_BIP" = "$DOCKER_BIP_DEFAULT" ]
[ "$DOCKER_BIP_SOURCE" = "default" ]
printf '{"bip":"173.17.0.1/16"}\n' > "$INVALID_DAEMON_JSON"
unset DOCKER_BIP DOCKER_BIP_SOURCE
DOCKER_DAEMON_JSON="$INVALID_DAEMON_JSON"
docker_network_resolve_bip "$SOURCE_ENV"
[ "$DOCKER_BIP" = "$DOCKER_BIP_DEFAULT" ]
[ "$DOCKER_BIP_SOURCE" = "default" ]

printf 'DOCKER_BIP=192.168.50.1/24\n' > "$SOURCE_ENV"
printf '{"bip":"10.250.0.1/16"}\n' > "$DAEMON_JSON_TMP"
DOCKER_BIP="100.64.0.1/16"
DOCKER_BIP_SOURCE=""
docker_network_resolve_bip "$SOURCE_ENV"
[ "$DOCKER_BIP" = "100.64.0.1/16" ]
[ "$DOCKER_BIP_SOURCE" = "environment" ]

unset DOCKER_BIP DOCKER_BIP_SOURCE
DOCKER_DAEMON_JSON="$DAEMON_JSON_TMP"
docker_network_resolve_bip "$SOURCE_ENV"
[ "$DOCKER_BIP" = "192.168.50.1/24" ]
[ "$DOCKER_BIP_SOURCE" = "$SOURCE_ENV" ]

: > "$SOURCE_ENV"
unset DOCKER_BIP DOCKER_BIP_SOURCE
docker_network_resolve_bip_from_config "$SOURCE_ENV"
[ "$DOCKER_BIP" = "$DOCKER_BIP_DEFAULT" ]
[ "$DOCKER_BIP_SOURCE" = "default" ]
unset DOCKER_BIP DOCKER_BIP_SOURCE
docker_network_resolve_bip "$SOURCE_ENV"
[ "$DOCKER_BIP" = "10.250.0.1/16" ]
[ "$DOCKER_BIP_SOURCE" = "$DAEMON_JSON_TMP" ]

REMOVED_NETWORKS=""
docker() {
  case "$1 ${2:-}" in
    info*) return 0 ;;
    "network ls")
      [[ " $REMOVED_NETWORKS " == *" omc-empty "* ]] || printf 'omc-empty\n'
      [[ " $REMOVED_NETWORKS " == *" label-omc "* ]] || printf 'label-omc\n'
      printf '%s\n' omc-active customer-empty planned
      ;;
    "network inspect")
      case "${3:-}:${5:-}" in
        omc-empty:'{{.Name}}') printf 'omcgo-legacy-172\n' ;;
        omc-empty:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.18.0.0/16\n' ;;
        omc-empty:'{{len .Containers}}') printf '0\n' ;;
        label-omc:'{{.Name}}') printf 'legacy-network\n' ;;
        label-omc:'{{index .Labels "com.docker.compose.project"}}') printf 'omc\n' ;;
        label-omc:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.18.1.0/24\n' ;;
        label-omc:'{{len .Containers}}') printf '0\n' ;;
        omc-active:'{{.Name}}') printf 'omcgo-active-172\n' ;;
        omc-active:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.19.0.0/16\n' ;;
        omc-active:'{{len .Containers}}') printf '1\n' ;;
        customer-empty:'{{.Name}}') printf 'customer-172\n' ;;
        customer-empty:'{{index .Labels "com.docker.compose.project"}}') printf 'customer\n' ;;
        customer-empty:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '172.20.0.0/16\n' ;;
        customer-empty:'{{len .Containers}}') printf '0\n' ;;
        planned:'{{.Name}}') printf 'omcgo\n' ;;
        planned:'{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}') printf '%s\n' "$DOCKER_COMPOSE_SUBNET" ;;
        planned:'{{len .Containers}}') printf '0\n' ;;
        *) return 1 ;;
      esac
      ;;
    "network rm")
      REMOVED_NETWORKS="${REMOVED_NETWORKS:+$REMOVED_NETWORKS }$3"
      return 0
      ;;
    *) return 1 ;;
  esac
}

REMOVED_OUTPUT="$(mktemp)"
trap 'rm -f "$TMP_ENV" "$SOURCE_ENV" "$DAEMON_JSON_TMP" "$DEFAULT_DAEMON_JSON" "$INVALID_DAEMON_JSON" "$REMOVED_OUTPUT"' EXIT
docker_network_cleanup_unplanned_networks > "$REMOVED_OUTPUT"
[ "$(cat "$REMOVED_OUTPUT")" = $'omcgo-legacy-172\t172.18.0.0/16\nlegacy-network\t172.18.1.0/24' ]
[ "$REMOVED_NETWORKS" = "omc-empty label-omc" ]
[ "$(docker_network_unplanned_networks)" = $'omcgo-active-172:172.19.0.0/16\ncustomer-172:172.20.0.0/16' ]

echo "docker network cleanup tests passed"

echo "docker network plan tests passed"

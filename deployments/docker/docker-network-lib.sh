#!/usr/bin/env bash

# Resolve and derive every Docker network value from DOCKER_BIP.
# The fixed fallback keeps ordinary deployments configuration-free; operators only
# need to override it when the default range overlaps the business network.
DOCKER_BIP_DEFAULT="10.240.0.1/16"

docker_network_env_value() {
  local file="$1" key="$2"
  [ -f "$file" ] || return 0
  awk -v key="$key" '
    $0 ~ "^[[:space:]]*" key "=" {
      value = $0
      sub("^[[:space:]]*" key "=", "", value)
      print value
      exit
    }
  ' "$file"
}

docker_network_read_daemon_bip() {
  local file="${1:-${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}}"
  command -v python3 >/dev/null 2>&1 || return 0
  python3 - "$file" <<'PYEOF'
import json
import os
import sys

try:
    if not os.path.exists(sys.argv[1]) or os.path.getsize(sys.argv[1]) == 0:
        raise OSError
    with open(sys.argv[1], encoding="utf-8") as handle:
        data = json.load(handle)
    value = data.get("bip", "")
    if isinstance(value, str):
        print(value)
except (OSError, json.JSONDecodeError):
    pass
PYEOF
}

docker_network_require_python3() {
  command -v python3 >/dev/null 2>&1 || {
    echo "python3 is required for Docker network planning and validation" >&2
    return 1
  }
}

docker_network_resolve_bip_from_user_config() {
  local file value
  if [ -n "${DOCKER_BIP:-}" ]; then
    DOCKER_BIP_SOURCE=environment
    return 0
  fi
  for file in "$@"; do
    value="$(docker_network_env_value "$file" DOCKER_BIP)"
    if [ -n "$value" ]; then
      DOCKER_BIP="$value"
      DOCKER_BIP_SOURCE="$file"
      export DOCKER_BIP DOCKER_BIP_SOURCE
      return 0
    fi
  done
  return 1
}

docker_network_resolve_bip_from_config() {
  docker_network_resolve_bip_from_user_config "$@" && return 0
  DOCKER_BIP="$DOCKER_BIP_DEFAULT"
  DOCKER_BIP_SOURCE=default
  export DOCKER_BIP DOCKER_BIP_SOURCE
  return 0
}

docker_network_resolve_bip() {
  local value
  docker_network_resolve_bip_from_user_config "$@" && return 0
  value="$(docker_network_read_daemon_bip "${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}")"
  if [ -n "$value" ] && (DOCKER_BIP="$value" docker_network_plan >/dev/null 2>&1); then
    DOCKER_BIP="$value"
    DOCKER_BIP_SOURCE="${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}"
    export DOCKER_BIP DOCKER_BIP_SOURCE
    return 0
  fi
  DOCKER_BIP="$DOCKER_BIP_DEFAULT"
  DOCKER_BIP_SOURCE=default
  export DOCKER_BIP DOCKER_BIP_SOURCE
  return 0
}

docker_network_plan() {
  [ -n "${DOCKER_BIP:-}" ] || {
    echo "DOCKER_BIP must be planned by the operator" >&2
    return 1
  }
  docker_network_require_python3 || return 1

  local plan key value
  plan="$(python3 - "$DOCKER_BIP" <<'PYEOF'
import ipaddress
import sys

try:
    interface = ipaddress.ip_interface(sys.argv[1])
except ValueError as exc:
    raise SystemExit(f"invalid DOCKER_BIP: {exc}")
if interface.version != 4:
    raise SystemExit("DOCKER_BIP must be an IPv4 interface")
if interface.ip.is_unspecified or interface.ip.is_loopback or interface.ip.is_multicast or interface.ip.is_link_local or interface.ip.is_reserved:
    raise SystemExit(f"DOCKER_BIP must be a usable unicast IPv4 address: {interface.ip}")
if not 16 <= interface.network.prefixlen <= 24:
    raise SystemExit("DOCKER_BIP prefix must be between /16 and /24")
if interface.ip == interface.network.network_address or interface.ip == interface.network.broadcast_address:
    raise SystemExit("DOCKER_BIP must use a host address, not a network or broadcast address")

allowed_ranges = [
  ipaddress.ip_network("10.0.0.0/8"),
  ipaddress.ip_network("192.168.0.0/16"),
  ipaddress.ip_network("100.64.0.0/10"),
]
if not any(interface.network.subnet_of(allowed) for allowed in allowed_ranges):
  raise SystemExit("DOCKER_BIP must be within 10.0.0.0/8, 192.168.0.0/16, or 100.64.0.0/10")

base = int(interface.network.network_address)
block_size = interface.network.num_addresses
networks = []
for offset in range(5):
    start = base + offset * block_size
    if start + block_size > 2**32:
        raise SystemExit("DOCKER_BIP leaves insufficient IPv4 space for the derived Docker networks")
    network = ipaddress.ip_network((start, interface.network.prefixlen))
    if not any(network.subnet_of(allowed) for allowed in allowed_ranges):
      raise SystemExit("DOCKER_BIP derived Docker networks must stay within the allowed range")
    networks.append(network)

pool_size = max(interface.network.prefixlen, 24)
print(f"DOCKER_NETWORK_SUBNET={networks[0]}")
print(f"DOCKER_NETWORK_GATEWAY={interface.ip}")
print(f"DOCKER_COMPOSE_SUBNET={networks[1]}")
print(f"DOCKER_COMPOSE_GATEWAY={networks[1].network_address + 1}")
print(f"DOCKER_ADDR_POOL_BASE={networks[2]}")
print(f"DOCKER_ADDR_POOL_SIZE={pool_size}")
print(f"DOCKER_TEST_SUBNET={networks[3]}")
print(f"DOCKER_TEST_GATEWAY={networks[3].network_address + 1}")
print(f"DOCKER_MIGRATION_SUBNET={networks[4]}")
print(f"DOCKER_MIGRATION_GATEWAY={networks[4].network_address + 1}")
PYEOF
  )" || return 1

  while IFS='=' read -r key value; do
    case "$key" in
      DOCKER_NETWORK_SUBNET|DOCKER_NETWORK_GATEWAY|DOCKER_COMPOSE_SUBNET|DOCKER_COMPOSE_GATEWAY|DOCKER_ADDR_POOL_BASE|DOCKER_ADDR_POOL_SIZE|DOCKER_TEST_SUBNET|DOCKER_TEST_GATEWAY|DOCKER_MIGRATION_SUBNET|DOCKER_MIGRATION_GATEWAY)
        printf -v "$key" '%s' "$value"
        export "$key"
        ;;
    esac
  done <<< "$plan"
  export DOCKER_BIP
}

docker_network_set_env_value() {
  local file="$1" key="$2" value="$3" temporary
  temporary="$(mktemp)" || return 1
  awk -v key="$key" -v value="$value" '
    BEGIN { replaced = 0 }
    $0 ~ "^[[:space:]]*" key "=" {
      if (!replaced) print key "=" value
      replaced = 1
      next
    }
    { print }
    END { if (!replaced) print key "=" value }
  ' "$file" > "$temporary" || {
    rm -f "$temporary"
    return 1
  }
  cat "$temporary" > "$file" || {
    rm -f "$temporary"
    return 1
  }
  rm -f "$temporary"
}

docker_network_write_env_file() {
  local file="$1" key
  [ -f "$file" ] || return 1
  for key in DOCKER_BIP DOCKER_NETWORK_SUBNET DOCKER_NETWORK_GATEWAY \
    DOCKER_COMPOSE_SUBNET DOCKER_COMPOSE_GATEWAY DOCKER_ADDR_POOL_BASE \
    DOCKER_ADDR_POOL_SIZE DOCKER_TEST_SUBNET DOCKER_TEST_GATEWAY \
    DOCKER_MIGRATION_SUBNET DOCKER_MIGRATION_GATEWAY; do
    docker_network_set_env_value "$file" "$key" "${!key:-}" || return 1
  done
}

docker_network_read_daemon_state() {
  local file="${1:-${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}}"
  command -v python3 >/dev/null 2>&1 || return 1
  python3 - "$file" <<'PYEOF'
import json
import os
import sys

try:
    if not os.path.exists(sys.argv[1]) or os.path.getsize(sys.argv[1]) == 0:
        raise OSError
    with open(sys.argv[1], encoding="utf-8") as handle:
        data = json.load(handle)
except (OSError, json.JSONDecodeError):
    data = {}
pools = data.get("default-address-pools")
pool = pools[0] if isinstance(pools, list) and pools and isinstance(pools[0], dict) else {}
print(f"{data.get('bip', '')}\t{pool.get('base', '')}\t{pool.get('size', '')}")
PYEOF
}

docker_network_subnet_is_planned() {
  local subnet="$1"
    python3 - "$subnet" \
      "$DOCKER_NETWORK_SUBNET" "$DOCKER_COMPOSE_SUBNET" "$DOCKER_TEST_SUBNET" \
      "$DOCKER_ADDR_POOL_BASE" "$DOCKER_ADDR_POOL_SIZE" "$DOCKER_MIGRATION_SUBNET" <<'PYEOF'
import ipaddress
import sys

try:
    actual = ipaddress.ip_network(sys.argv[1], strict=False)
    docker_network = ipaddress.ip_network(sys.argv[2], strict=False)
    compose_network = ipaddress.ip_network(sys.argv[3], strict=False)
    test_network = ipaddress.ip_network(sys.argv[4], strict=False)
    pool = ipaddress.ip_network(sys.argv[5], strict=False)
    migration_network = ipaddress.ip_network(sys.argv[7], strict=False)
    pool_size = int(sys.argv[6])
except (ValueError, TypeError):
    raise SystemExit(1)
allowed = actual in (docker_network, compose_network, test_network, migration_network)
allowed = allowed or (actual.subnet_of(pool) and actual.prefixlen >= pool_size)
raise SystemExit(0 if allowed else 1)
PYEOF
}

docker_network_unplanned_networks() {
  command -v docker >/dev/null 2>&1 || return 0
  docker info >/dev/null 2>&1 || return 0

  local network_id network_name subnet
  while IFS= read -r network_id; do
    [ -n "$network_id" ] || continue
    network_name="$(docker network inspect "$network_id" --format '{{.Name}}' 2>/dev/null || echo "$network_id")"
    case "$network_name" in
      host|none) continue ;;
    esac
    while IFS= read -r subnet; do
      [ -n "$subnet" ] || continue
      if ! docker_network_subnet_is_planned "$subnet"; then
        printf '%s:%s\n' "$network_name" "$subnet"
      fi
    done < <(docker network inspect "$network_id" --format '{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}' 2>/dev/null || true)
  done < <(docker network ls -q 2>/dev/null || true)
}

docker_network_is_omc_owned() {
  local network_id="$1" network_name compose_project
  network_name="$(docker network inspect "$network_id" --format '{{.Name}}' 2>/dev/null || true)"
  case "$network_name" in
    omcgo-*|omc-*) return 0 ;;
  esac

  compose_project="$(docker network inspect "$network_id" --format '{{index .Labels "com.docker.compose.project"}}' 2>/dev/null || true)"
  case "$compose_project" in
    omc|omcgo) return 0 ;;
  esac
  return 1
}

docker_network_cleanup_unplanned_networks() {
  command -v docker >/dev/null 2>&1 || return 0
  docker info >/dev/null 2>&1 || return 0

  local network_id network_name subnet unplanned_subnet container_count
  while IFS= read -r network_id; do
    [ -n "$network_id" ] || continue
    network_name="$(docker network inspect "$network_id" --format '{{.Name}}' 2>/dev/null || echo "$network_id")"
    case "$network_name" in
      bridge|host|none) continue ;;
    esac
    docker_network_is_omc_owned "$network_id" || continue

    unplanned_subnet=""
    while IFS= read -r subnet; do
      [ -n "$subnet" ] || continue
      if ! docker_network_subnet_is_planned "$subnet"; then
        unplanned_subnet="$subnet"
        break
      fi
    done < <(docker network inspect "$network_id" --format '{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}' 2>/dev/null || true)
    [ -n "$unplanned_subnet" ] || continue

    container_count="$(docker network inspect "$network_id" --format '{{len .Containers}}' 2>/dev/null || echo 1)"
    [ "$container_count" = 0 ] || continue
    docker network rm "$network_id" >/dev/null 2>&1 || continue
    printf '%s\t%s\n' "$network_name" "$unplanned_subnet"
  done < <(docker network ls -q 2>/dev/null || true)
}

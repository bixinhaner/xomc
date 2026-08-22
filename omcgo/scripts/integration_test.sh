#!/usr/bin/env bash
#
# integration_test.sh — Run integration tests with ephemeral infrastructure
#
# This script:
#   1. Starts test-only Docker containers (PostgreSQL+TimescaleDB, Redis, NATS)
#   2. Waits for all services to become healthy
#   3. Runs database migrations against the test DB
#   4. Executes Go integration tests with required env vars
#   5. Tears down containers regardless of test outcome
#   6. Returns the test exit code
#
# Usage:
#   bash scripts/integration_test.sh
#   bash scripts/integration_test.sh -v          # verbose
#   bash scripts/integration_test.sh -k          # keep containers running after tests
#
# Prerequisites:
#   - Docker and docker compose plugin installed
#   - Go toolchain available
#   - Run from the omcgo/ directory (or script auto-detects)

set -euo pipefail

# --- Configuration -----------------------------------------------------------

NETWORK_LIB="../deployments/docker/docker-network-lib.sh"
COMPOSE_PROJECT="omcgo-test"

DB_DSN="postgres://omcgo_test:omcgo_test@localhost:5433/omcgo_test?sslmode=disable"
REDIS_ADDR="localhost:6380"
NATS_URL="nats://localhost:4223"

MIGRATIONS_DIR="migrations"
MAX_WAIT_SECONDS=60
VERBOSE=""
KEEP_RUNNING=""

# --- Parse arguments ----------------------------------------------------------

while getopts "vk" opt; do
    case $opt in
        v) VERBOSE="-v" ;;
        k) KEEP_RUNNING="true" ;;
        *) echo "Usage: $0 [-v] [-k]"; exit 1 ;;
    esac
done

# --- Helpers ------------------------------------------------------------------

log() {
    echo "[integration-test] $(date '+%H:%M:%S') $*"
}

err() {
    echo "[integration-test] ERROR: $*" >&2
}

# Change to omcgo/ root if not already there
cd_to_project_root() {
    if [[ -f "go.mod" && -d "../deployments" ]]; then
        return 0
    fi
    # Try common locations
    for dir in "." "omcgo" "../omcgo"; do
        if [[ -f "$dir/go.mod" && -d "${dir}/../deployments" ]]; then
            cd "$dir"
            return 0
        fi
    done
    err "Cannot find omcgo project root. Run from omcgo/ directory."
    exit 1
}

# --- Main Flow ----------------------------------------------------------------

cd_to_project_root
log "Working directory: $(pwd)"

[ -f "$NETWORK_LIB" ] || {
    err "Missing Docker network planning library: $NETWORK_LIB"
    exit 1
}
. "$NETWORK_LIB"
if ! docker_network_resolve_bip "${DOCKER_NETWORK_ENV_FILE:-../.env}"; then
    err "DOCKER_BIP is required; configure the customer-planned network in ../.env or pass it in the environment"
    exit 1
fi
docker_network_plan || {
    err "Invalid DOCKER_BIP or unable to derive Docker network plan: ${DOCKER_BIP:-<empty>}"
    exit 1
}
NETWORK_WRAPPER="../deployments/docker/dc-test.sh"
export OMC_TEST_PROJECT="$COMPOSE_PROJECT"

# Track test exit code for cleanup
TEST_EXIT_CODE=0

# Cleanup function: always tear down containers (unless -k)
cleanup() {
    if [[ -n "$KEEP_RUNNING" ]]; then
        log "Keeping containers running (-k flag). Stop with:"
        log "  $NETWORK_WRAPPER down -v"
    else
        log "Tearing down test containers..."
        "$NETWORK_WRAPPER" down -v 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Step 1: Start containers
log "Starting test infrastructure..."
"$NETWORK_WRAPPER" down -v 2>/dev/null || true
"$NETWORK_WRAPPER" up -d --wait --wait-timeout "$MAX_WAIT_SECONDS"
log "All containers healthy."

# Step 2: Enable TimescaleDB extension
log "Enabling TimescaleDB extension..."
PGPASSWORD=omcgo_test psql -h localhost -p 5433 -U omcgo_test -d omcgo_test \
    -c "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;" 2>/dev/null || {
    # If psql is not available locally, use docker exec
    "$NETWORK_WRAPPER" exec -T postgres-test \
        psql -U omcgo_test -d omcgo_test \
        -c "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;"
}

# Step 3: Run database migrations
log "Running database migrations..."
if [[ -f "bin/omcgo-migrate" ]]; then
    ./bin/omcgo-migrate --dsn "$DB_DSN" --path "$MIGRATIONS_DIR" up
elif command -v go &>/dev/null; then
    go run ./cmd/migrate up --dsn "$DB_DSN" --path "$MIGRATIONS_DIR"
else
    err "Neither bin/omcgo-migrate nor go toolchain found."
    exit 1
fi
log "Migrations complete."

# Step 4: Run integration tests
log "Running integration tests..."
export OMCGO_TEST_DB_DSN="$DB_DSN"
export OMCGO_TEST_REDIS_ADDR="$REDIS_ADDR"
export OMCGO_TEST_NATS_URL="$NATS_URL"

set +e
go test $VERBOSE -count=1 -timeout=5m ./test/integration/...
TEST_EXIT_CODE=$?
set -e

# Step 5: Report result
if [[ $TEST_EXIT_CODE -eq 0 ]]; then
    log "All integration tests passed."
else
    log "Integration tests failed (exit code: $TEST_EXIT_CODE)."
fi

exit $TEST_EXIT_CODE

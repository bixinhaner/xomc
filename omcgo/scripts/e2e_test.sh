#!/bin/bash
# scripts/e2e_test.sh — Run E2E tests against a running OMC server
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== OMC E2E Tests ==="

# Check if a server is already running
BASE_URL="${OMCGO_E2E_BASE_URL:-http://localhost:18080}"

if curl -sf "${BASE_URL}/healthz" > /dev/null 2>&1; then
    echo "Server already running at ${BASE_URL}"
else
    echo "No server detected at ${BASE_URL}"
    echo "Start the server first:"
    echo "  make build-app && OMCGO_CONFIG_PATH=cmd/app/etc/config.test.yaml ./bin/omcgo-app"
    echo ""
    echo "Or set OMCGO_E2E_BASE_URL to point to a running instance."
    exit 1
fi

echo "Running E2E tests..."
cd "${PROJECT_DIR}"

export OMCGO_E2E_BASE_URL="${BASE_URL}"
go test -v -count=1 -tags=e2e ./test/e2e/...

echo "=== E2E Tests Complete ==="

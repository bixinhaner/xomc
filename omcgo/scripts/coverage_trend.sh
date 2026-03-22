#!/usr/bin/env bash
# coverage_trend.sh — Run tests, record coverage, and show trend.
#
# Usage:
#   bash scripts/coverage_trend.sh
#
# Outputs:
#   - coverage.out          (go test coverage profile)
#   - coverage_history.csv  (date,commit,coverage append-only log)
#   - Prints coverage change vs previous run (↑ / ↓ / unchanged)

set -euo pipefail

cd "$(dirname "$0")/.."

HISTORY_FILE="coverage_history.csv"
COVER_FILE="coverage.out"

# 1. Run tests with coverage
echo "Running tests with coverage..."
go test -coverprofile="${COVER_FILE}" ./... > /dev/null 2>&1 || true

if [ ! -f "${COVER_FILE}" ]; then
    echo "ERROR: coverage profile not generated"
    exit 1
fi

# 2. Extract total coverage percentage
TOTAL=$(go tool cover -func="${COVER_FILE}" | grep '^total:' | awk '{print $NF}' | tr -d '%')

if [ -z "${TOTAL}" ]; then
    echo "ERROR: could not extract total coverage"
    exit 1
fi

# 3. Get current date and commit hash
DATE=$(date '+%Y-%m-%d')
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 4. Initialize history file if needed
if [ ! -f "${HISTORY_FILE}" ]; then
    echo "date,commit,coverage" > "${HISTORY_FILE}"
fi

# 5. Get previous coverage for comparison
PREV=$(tail -n 1 "${HISTORY_FILE}" | grep -v '^date,' | awk -F',' '{print $3}')

# 6. Append current entry
echo "${DATE},${COMMIT},${TOTAL}" >> "${HISTORY_FILE}"

# 7. Print result with trend
echo "Coverage: ${TOTAL}%  (commit: ${COMMIT}, date: ${DATE})"

if [ -z "${PREV}" ]; then
    echo "First recorded entry — no previous data to compare."
else
    DIFF=$(echo "${TOTAL} - ${PREV}" | bc)
    if [ "$(echo "${DIFF} > 0" | bc)" -eq 1 ]; then
        echo "Trend: ↑ +${DIFF}% (previous: ${PREV}%)"
    elif [ "$(echo "${DIFF} < 0" | bc)" -eq 1 ]; then
        echo "Trend: ↓ ${DIFF}% (previous: ${PREV}%)"
    else
        echo "Trend: unchanged (${PREV}%)"
    fi
fi

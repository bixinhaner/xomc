#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-9222}"

lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN

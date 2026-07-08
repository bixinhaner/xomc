#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="${1:-$(pwd)}"
TARGET_DIR="${WORKSPACE_ROOT}/.codex-tools/browser-control"

mkdir -p "${TARGET_DIR}"

for script in \
  check-cdp.sh \
  start-chrome.sh \
  list-tabs.js \
  navigate-existing-tab.js \
  stop-managed-chrome.sh
do
  cp "${SOURCE_DIR}/${script}" "${TARGET_DIR}/${script}"
  chmod +x "${TARGET_DIR}/${script}"
done

echo "Installed browser-control tools to ${TARGET_DIR}"

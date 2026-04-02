#!/bin/sh
set -e

# Resolve environment-specific config file.
# OMCGO_ENV controls which config variant is used: dev (default), test, prod.
# The binary name is derived from the first argument or OMCGO_SERVICE.
# Config files are stored as /etc/omcgo/{service}.{env}.yaml, e.g. /etc/omcgo/app.prod.yaml

ENV="${OMCGO_ENV:-dev}"
SERVICE="${OMCGO_SERVICE:-app}"

CONFIG="/etc/omcgo/${SERVICE}.${ENV}.yaml"

# Fall back to default config if the env-specific one doesn't exist
if [ ! -f "$CONFIG" ]; then
  CONFIG="/etc/omcgo/${SERVICE}.dev.yaml"
fi

exec "omcgo-${SERVICE}" --config "$CONFIG" "$@"

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

# T-0098 dictloader 用相对路径 xml_base_dir: "data" 加载字典 XML。
# 切到 /etc/omcgo 让相对路径解析为 /etc/omcgo/data（与 Dockerfile.app COPY 一致）。
cd /etc/omcgo

exec "omcgo-${SERVICE}" --config "$CONFIG" "$@"

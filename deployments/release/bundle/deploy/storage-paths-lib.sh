#!/usr/bin/env bash
# OMC 有状态服务宿主机路径：探测、.env 更新、校验与目录准备。

STORAGE_PATH_KEYS="POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH"

storage_env_get() {
  local env_file="$1" key="$2"
  [ -f "$env_file" ] || return 0
  awk -v key="$key" '
    index($0, key "=") == 1 {
      print substr($0, length(key) + 2)
      exit
    }
  ' "$env_file"
}

storage_select_largest_mount() {
  awk -F'|' '
    $1 ~ /^[0-9]+$/ && $2 ~ /^\// {
      if (!found || $1 + 0 > largest + 0) {
        largest=$1
        mount=$2
        found=1
      }
    }
    END {
      if (found) print mount
      else exit 1
    }
  '
}

storage_set_env_if_empty() {
  local env_file="$1" key="$2" value="$3" tmp
  [ -f "$env_file" ] || : > "$env_file"
  [ -n "$(storage_env_get "$env_file" "$key")" ] && return 0

  tmp="$(mktemp)" || return 1
  if awk -v key="$key" -v value="$value" '
      BEGIN { replaced=0 }
      index($0, key "=") == 1 {
        if (!replaced) print key "=" value
        replaced=1
        next
      }
      { print }
      END {
        if (!replaced) print key "=" value
      }
    ' "$env_file" > "$tmp"; then
    cat "$tmp" > "$env_file"
    rm -f "$tmp"
  else
    rm -f "$tmp"
    return 1
  fi
}

storage_apply_recommended_paths() {
  local env_file="$1" mount="${2%/}" key suffix
  [ -n "$mount" ] || mount=""

  for key in $STORAGE_PATH_KEYS; do
    case "$key" in
      POSTGRES_DATA_PATH) suffix="postgres" ;;
      TSDB_DATA_PATH)     suffix="timescaledb" ;;
      REDIS_DATA_PATH)    suffix="redis" ;;
      NATS_DATA_PATH)     suffix="nats" ;;
      MINIO_DATA_PATH)    suffix="minio" ;;
      *) return 1 ;;
    esac
    storage_set_env_if_empty "$env_file" "$key" "$mount/omc-data/$suffix" || return 1
  done
}

storage_validate_env_paths() {
  local env_file="$1" key value failed=0
  for key in $STORAGE_PATH_KEYS; do
    value="$(storage_env_get "$env_file" "$key")"
    case "$value" in
      /*) ;;
      "")
        printf '[storage][错误] %s 未配置；请先运行 plan-resources.sh 或编辑 %s\n' "$key" "$env_file" >&2
        failed=1
        ;;
      *)
        printf '[storage][错误] %s 必须是绝对路径，当前值：%s\n' "$key" "$value" >&2
        failed=1
        ;;
    esac
  done
  [ "$failed" -eq 0 ]
}

storage_prepare_env_paths() {
  local env_file="$1" key value
  storage_validate_env_paths "$env_file" || return 1
  for key in $STORAGE_PATH_KEYS; do
    value="$(storage_env_get "$env_file" "$key")"
    mkdir -p "$value" || {
      printf '[storage][错误] 无法创建 %s=%s\n' "$key" "$value" >&2
      return 1
    }
    if [ ! -d "$value" ] || [ ! -w "$value" ]; then
      printf '[storage][错误] 数据目录不可写：%s=%s\n' "$key" "$value" >&2
      return 1
    fi
  done
}

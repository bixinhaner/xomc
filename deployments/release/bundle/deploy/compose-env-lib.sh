#!/usr/bin/env bash
# Compose 环境文件加载与运行态核对公共函数。
#
# Docker Compose 的优先级是：调用进程环境 > 后声明的 --env-file > 先声明的
# --env-file。install.sh 需要读取 IMAGE_*，因此会把 env 文件导入调用进程；必须按
# Compose 的相同顺序加载全部文件，否则先加载的 .env 会反向覆盖 resources.env。

deploy_env_load() { # deploy_env_load <base-env> [overlay-env ...]
  local file
  for file in "$@"; do
    [ -f "$file" ] || continue
    set -a
    # 文件来自受信任的本地交付包/运维配置，与 install.sh 既有 source .env 语义一致。
    # shellcheck disable=SC1090
    . "$file"
    set +a
  done
}

deploy_env_file_value() { # deploy_env_file_value <key> <file>
  local key="$1" file="$2"
  [ -f "$file" ] || return 1
  awk -F= -v key="$key" '
    $1 == key {
      value = substr($0, length(key) + 2)
      found = 1
    }
    END {
      if (found) print value
      else exit 1
    }
  ' "$file"
}

deploy_env_effective_value() { # deploy_env_effective_value <key> <env-file ...>
  local key="$1" file value="" found=1
  shift
  for file in "$@"; do
    if value="$(deploy_env_file_value "$key" "$file" 2>/dev/null)"; then
      found=0
    fi
  done
  [ "$found" -eq 0 ] || return 1
  printf '%s\n' "$value"
}

deploy_env_public_host_valid() { # deploy_env_public_host_valid <host>
  local host="${1:-}"
  [ -n "$host" ] || return 1
  case "$host" in
    localhost|127.0.0.1|0.0.0.0|::1) return 1 ;;
    *[[:space:]]*|*://*|*/*) return 1 ;;
  esac
}

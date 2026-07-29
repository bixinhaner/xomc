#!/usr/bin/env bash

# 读取 YAML 顶层 session 段内的 max_concurrent，避免误读其他业务段的同名键。
acs_session_max_concurrent() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ {
      in_session = ($0 ~ /^session:[[:space:]]*(#.*)?$/)
    }
    in_session && /^[[:space:]]+max_concurrent:[[:space:]]*[0-9]+/ {
      value = $0
      sub(/^[^:]*:[[:space:]]*/, "", value)
      sub(/[[:space:]#].*$/, "", value)
      print value
      exit
    }
  ' "$config"
}

# upgrade_acs_session_limit <现网配置> <新包模板>
#
# 仅把项目历史默认值 10000 安全迁移到当前模板默认值 30000。任何运维自定义值都保留；
# 已迁移配置不改写，确保升级幂等。结果通过 ACS_SESSION_LIMIT_UPGRADE_RESULT 返回。
upgrade_acs_session_limit() {
  local live_config="$1" template_config="$2"
  local live_limit template_limit tmp

  ACS_SESSION_LIMIT_UPGRADE_RESULT="noop"
  [ -f "$live_config" ] || return 0
  [ -f "$template_config" ] || return 0

  live_limit="$(acs_session_max_concurrent "$live_config")" || return 0
  template_limit="$(acs_session_max_concurrent "$template_config")" || return 0

  [ "$template_limit" = "30000" ] || return 0
  if [ "$live_limit" = "30000" ]; then
    return 0
  fi
  if [ "$live_limit" != "10000" ]; then
    ACS_SESSION_LIMIT_UPGRADE_RESULT="preserved"
    return 0
  fi

  tmp="$(mktemp "${live_config}.tmp.XXXXXX")" || return 1
  if ! awk '
      BEGIN { in_session = 0; changed = 0 }
      /^[^[:space:]#][^:]*:/ {
        in_session = ($0 ~ /^session:[[:space:]]*(#.*)?$/)
      }
      in_session && !changed &&
        /^[[:space:]]+max_concurrent:[[:space:]]*10000([[:space:]]*(#.*)?)?$/ {
        prefix = $0
        sub(/max_concurrent:.*/, "", prefix)
        comment = ""
        if (index($0, "#") > 0) {
          comment = " " substr($0, index($0, "#"))
        }
        print prefix "max_concurrent: 30000" comment
        changed = 1
        next
      }
      { print }
      END { if (changed != 1) exit 42 }
    ' "$live_config" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi

  # 覆写原 inode，保留现网文件的属主与权限。
  if ! cat "$tmp" > "$live_config"; then
    rm -f "$tmp"
    return 1
  fi
  rm -f "$tmp"

  [ "$(acs_session_max_concurrent "$live_config")" = "30000" ] || return 1
  ACS_SESSION_LIMIT_UPGRADE_RESULT="migrated"
}

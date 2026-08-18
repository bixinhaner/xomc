#!/usr/bin/env bash

# 读取 YAML 顶层 session 段内的 max_concurrent，避免误读其他业务段的同名键。
acs_session_max_concurrent() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ {
      in_session = ($0 ~ /^session:[[:space:]]*(#.*)?$/)
      direct_indent = -1
    }
    in_session && $0 ~ /^[[:space:]]+/ && $0 !~ /^[[:space:]]*(#|$)/ {
      match($0, /[^[:space:]]/)
      indent = RSTART - 1
      if (direct_indent < 0) direct_indent = indent
    }
    in_session && indent == direct_indent &&
      /^[[:space:]]+max_concurrent:[[:space:]]*[0-9]+/ {
      value = $0
      sub(/^[^:]*:[[:space:]]*/, "", value)
      sub(/[[:space:]#].*$/, "", value)
      print value
      found = 1
      exit
    }
    END { if (!found) exit 1 }
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
  if ! cp -p "$live_config" "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  if ! awk '
      BEGIN { in_session = 0; direct_indent = -1; changed = 0 }
      /^[^[:space:]#][^:]*:/ {
        in_session = ($0 ~ /^session:[[:space:]]*(#.*)?$/)
        direct_indent = -1
      }
      in_session && $0 ~ /^[[:space:]]+/ && $0 !~ /^[[:space:]]*(#|$)/ {
        match($0, /[^[:space:]]/)
        indent = RSTART - 1
        if (direct_indent < 0) direct_indent = indent
      }
      in_session && indent == direct_indent && !changed &&
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

  # 临时文件与现网配置位于同一目录，cp -p 保留属主/权限，mv 在同一文件系统内原子替换。
  # 替换失败时现网文件完全不动，不能先以重定向方式截断它。
  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi

  [ "$(acs_session_max_concurrent "$live_config")" = "30000" ] || return 1
  ACS_SESSION_LIMIT_UPGRADE_RESULT="migrated"
}

# upgrade_prod_database_dsns <现网配置> <新包模板>
#
# 旧版实例配置可能把数据库密码写成了旧的字面量。生产模板已统一使用
# ${POSTGRES_*} 环境变量，安装升级时只同步 db/tsdb 的 dsn 行，避免 ACS
# 使用旧密码连接存量数据库；其它运维配置保持不变，操作幂等。
upgrade_prod_database_dsns() {
  local live_config="$1" template_config="$2" tmp
  [ -f "$live_config" ] || return 0
  [ -f "$template_config" ] || return 0

  tmp="$(mktemp "${live_config}.dsn.tmp.XXXXXX")" || return 1
  if ! awk -v template_file="$template_config" '
      BEGIN {
        section = ""
        while ((getline line < template_file) > 0) {
          if (line ~ /^[^[:space:]#][^:]*:/) {
            section = line
            sub(/:.*/, "", section)
          }
          if ((section == "db" || section == "tsdb") && line ~ /^[[:space:]]+dsn:[[:space:]]*/) {
            dsn[section] = line
          }
        }
        close(template_file)
        live_section = ""
      }
      /^[^[:space:]#][^:]*:/ {
        live_section = $0
        sub(/:.*/, "", live_section)
      }
      /^[[:space:]]+dsn:[[:space:]]*/ && (live_section == "db" || live_section == "tsdb") {
        if (dsn[live_section] != "") {
          print dsn[live_section]
          changed[live_section] = 1
          next
        }
      }
      { print }
      END {
        if (dsn["db"] != "" && changed["db"] != 1) exit 42
        if (dsn["tsdb"] != "" && changed["tsdb"] != 1) exit 42
      }
    ' "$live_config" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi

  if ! cmp -s "$live_config" "$tmp"; then
    if ! mv -f "$tmp" "$live_config"; then
      rm -f "$tmp"
      return 1
    fi
  else
    rm -f "$tmp"
  fi
}

app_param_sync_recovery_run_limit() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ {
      in_param_sync = ($0 ~ /^param_sync:[[:space:]]*(#.*)?$/)
    }
    in_param_sync && /^[[:space:]]+recovery_run_limit:[[:space:]]*[0-9]+/ {
      value = $0
      sub(/^[^:]*:[[:space:]]*/, "", value)
      sub(/[[:space:]#].*$/, "", value)
      print value
      found = 1
      exit
    }
    END { if (!found) exit 1 }
  ' "$config"
}

# Upgrade only the historical project default. An operator-selected value is a
# capacity decision and must survive package upgrades unchanged.
upgrade_app_param_sync_recovery_limit() {
  local live_config="$1" template_config="$2"
  local live_limit template_limit tmp

  PARAM_SYNC_RECOVERY_LIMIT_UPGRADE_RESULT="noop"
  [ -f "$live_config" ] || return 0
  [ -f "$template_config" ] || return 1
  live_limit="$(app_param_sync_recovery_run_limit "$live_config")" || return 0
  template_limit="$(app_param_sync_recovery_run_limit "$template_config")" || return 1
  [ "$template_limit" = "200" ] || return 0
  [ "$live_limit" = "200" ] && return 0
  if [ "$live_limit" != "20" ]; then
    PARAM_SYNC_RECOVERY_LIMIT_UPGRADE_RESULT="preserved"
    return 0
  fi

  tmp="$(mktemp "${live_config}.param-sync.tmp.XXXXXX")" || return 1
  if ! cp -p "$live_config" "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  if ! awk '
      /^[^[:space:]#][^:]*:/ {
        in_param_sync = ($0 ~ /^param_sync:[[:space:]]*(#.*)?$/)
      }
      in_param_sync && !changed &&
        /^[[:space:]]+recovery_run_limit:[[:space:]]*20([[:space:]]*(#.*)?)?$/ {
        prefix = $0
        sub(/recovery_run_limit:.*/, "", prefix)
        comment = ""
        if (index($0, "#") > 0) comment = " " substr($0, index($0, "#"))
        print prefix "recovery_run_limit: 200" comment
        changed = 1
        next
      }
      { print }
      END { if (changed != 1) exit 42 }
    ' "$live_config" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi
  [ "$(app_param_sync_recovery_run_limit "$live_config")" = "200" ] || return 1
  PARAM_SYNC_RECOVERY_LIMIT_UPGRADE_RESULT="migrated"
}

config_has_top_level_section() {
  local config="$1" section="$2"
  [ -f "$config" ] || return 1
  awk -v wanted="$section" '
    $0 ~ "^" wanted ":[[:space:]]*(#.*)?$" { found=1; exit }
    END { if (!found) exit 1 }
  ' "$config"
}

# config_dir_has_instance_config <实例配置目录>
#
# etc/ may already contain auxiliary state created earlier in the same install
# (notably license/omcPublicKey.store).  Those files do not make a fresh install
# an upgrade.  Conversely, any existing service config means operator settings
# may need preserving, so the caller must use the upgrade path.
config_dir_has_instance_config() {
  local config_dir="$1" service_config
  for service_config in acs.prod.yaml app.prod.yaml worker.prod.yaml; do
    [ -e "$config_dir/$service_config" ] && return 0
  done
  return 1
}

acs_has_trusted_proxy_cidrs() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ { in_server = ($0 ~ /^server:[[:space:]]*(#.*)?$/) }
    in_server && /^[[:space:]]+trusted_proxy_cidrs:[[:space:]]*/ { found=1; exit }
    END { if (!found) exit 1 }
  ' "$config"
}

# Older retained ACS configs predate the trusted Web gateway declaration. Copy
# only this server child from the release template; never rewrite other server
# timeouts, TLS paths, or operator settings.
upgrade_acs_trusted_proxy_cidrs() {
  local live_config="$1" template_config="$2" template_line tmp
  [ -f "$live_config" ] || return 1
  [ -f "$template_config" ] || return 1
  acs_has_trusted_proxy_cidrs "$live_config" && return 0
  template_line="$(awk '
    /^[^[:space:]#][^:]*:/ { in_server = ($0 ~ /^server:[[:space:]]*(#.*)?$/) }
    in_server && /^[[:space:]]+trusted_proxy_cidrs:[[:space:]]*/ { print; found=1; exit }
    END { if (!found) exit 1 }
  ' "$template_config")" || return 1

  tmp="$(mktemp "${live_config}.trusted-proxy.tmp.XXXXXX")" || return 1
  if ! awk -v addition="$template_line" '
      /^[^[:space:]#][^:]*:/ {
        if (in_server && !inserted) { print addition; inserted=1 }
        in_server = ($0 ~ /^server:[[:space:]]*(#.*)?$/)
        if (in_server) saw_server=1
      }
      { print }
      END {
        if (in_server && !inserted) { print addition; inserted=1 }
        if (!saw_server || !inserted) exit 42
      }
    ' "$live_config" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi
  acs_has_trusted_proxy_cidrs "$live_config"
}

# upgrade_pm_redis_config <现网配置> <新包模板>
#
# 旧版本保留的 app/worker 配置没有 pm_redis。只追加新模板中的完整顶层段，
# 不改写现有 redis 或其它运维配置；已有 pm_redis 时保持原样，由应用校验负责
# 拒绝空地址或与核心 Redis 交叉的错误配置。
upgrade_pm_redis_config() {
  local live_config="$1" template_config="$2" block tmp
  [ -f "$live_config" ] || return 1
  [ -f "$template_config" ] || return 1
  config_has_top_level_section "$live_config" pm_redis && return 0

  block="$(mktemp "${live_config}.pm-redis-block.XXXXXX")" || return 1
  if ! awk '
      /^pm_redis:[[:space:]]*(#.*)?$/ { capture=1 }
      capture && printed && /^[^[:space:]#][^:]*:[[:space:]]*/ { exit }
      capture { print; printed=1 }
      END { if (!printed) exit 42 }
    ' "$template_config" > "$block"; then
    rm -f "$block"
    return 1
  fi

  tmp="$(mktemp "${live_config}.tmp.XXXXXX")" || {
    rm -f "$block"
    return 1
  }
  if ! cp -p "$live_config" "$tmp"; then
    rm -f "$block" "$tmp"
    return 1
  fi
  printf '\n' >> "$tmp" || {
    rm -f "$block" "$tmp"
    return 1
  }
  if ! command cat "$block" >> "$tmp"; then
    rm -f "$block" "$tmp"
    return 1
  fi
  rm -f "$block"
  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi
  config_has_top_level_section "$live_config" pm_redis
}

app_has_gpv_response_config() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ {
      in_provision = ($0 ~ /^provision:[[:space:]]*(#.*)?$/)
    }
    in_provision && /^  gpv_response:[[:space:]]*(#.*)?$/ {
      found = 1
      exit
    }
    END { if (!found) exit 1 }
  ' "$config"
}

# upgrade_app_gpv_response_config <现网 app 配置> <新包 app 模板>
#
# 老版本实例配置会被 install.sh 保留，因而不会自然获得新增的
# provision.gpv_response。这里只从新模板复制缺失的完整子段，不改动任何已有
# provision/其它业务配置；同目录临时文件 + 原子替换保证失败时现网文件不被截断。
upgrade_app_gpv_response_config() {
  local live_config="$1" template_config="$2"
  local block tmp

  [ -f "$live_config" ] || return 0
  [ -f "$template_config" ] || return 1
  app_has_gpv_response_config "$live_config" && return 0

  block="$(mktemp "${live_config}.gpv-block.XXXXXX")" || return 1
  if ! awk '
      /^[^[:space:]#][^:]*:/ {
        in_provision = ($0 ~ /^provision:[[:space:]]*(#.*)?$/)
      }
      in_provision && /^  gpv_response:[[:space:]]*(#.*)?$/ {
        capture = 1
      }
      capture && printed && /^[^[:space:]#][^:]*:/ {
        exit
      }
      capture && printed && /^  [^[:space:]#][^:]*:[[:space:]]*/ {
        exit
      }
      capture {
        print
        printed = 1
      }
      END { if (!printed) exit 42 }
    ' "$template_config" > "$block"; then
    rm -f "$block"
    return 1
  fi

  tmp="$(mktemp "${live_config}.tmp.XXXXXX")" || {
    rm -f "$block"
    return 1
  }
  if ! cp -p "$live_config" "$tmp"; then
    rm -f "$block" "$tmp"
    return 1
  fi
  if ! awk -v block_file="$block" '
      BEGIN {
        while ((getline line < block_file) > 0) {
          block[++block_count] = line
        }
        close(block_file)
      }
      function emit_block(    i) {
        for (i = 1; i <= block_count; i++) print block[i]
        inserted = 1
      }
      /^[^[:space:]#][^:]*:/ {
        if (in_provision && !inserted) emit_block()
        in_provision = ($0 ~ /^provision:[[:space:]]*(#.*)?$/)
        if (in_provision) saw_provision = 1
      }
      { print }
      END {
        if (in_provision && !inserted) emit_block()
        if (!saw_provision || !inserted) exit 42
      }
    ' "$live_config" > "$tmp"; then
    rm -f "$block" "$tmp"
    return 1
  fi
  rm -f "$block"

  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi
  app_has_gpv_response_config "$live_config"
}

# 读取 YAML 顶层 tsdb 段内的 max_conns，避免误读 db.max_conns。
worker_tsdb_max_conns() {
  local config="$1"
  [ -f "$config" ] || return 1
  awk '
    /^[^[:space:]#][^:]*:/ {
      in_tsdb = ($0 ~ /^tsdb:[[:space:]]*(#.*)?$/)
      direct_indent = -1
    }
    in_tsdb && $0 ~ /^[[:space:]]+/ && $0 !~ /^[[:space:]]*(#|$)/ {
      match($0, /[^[:space:]]/)
      indent = RSTART - 1
      if (direct_indent < 0) direct_indent = indent
    }
    in_tsdb && indent == direct_indent &&
      /^[[:space:]]+max_conns:[[:space:]]*/ {
      value = $0
      sub(/^[^:]*:[[:space:]]*/, "", value)
      sub(/[[:space:]]*#.*/, "", value)
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      if (value ~ /^"[0-9]+"$/ || value ~ /^\047[0-9]+\047$/) {
        value = substr(value, 2, length(value) - 2)
      }
      # YAML 1.1/Viper 会把 040 解释成八进制 32。只接受规范十进制，
      # 避免 Shell 数值比较、迁移器和应用运行时对同一文本得出不同结果。
      if (value !~ /^(0|[1-9][0-9]*)$/) exit 2
      print value
      found = 1
      exit
    }
    END { if (!found) exit 1 }
  ' "$config"
}

WORKER_TSDB_SAFE_POOL=128

# validate_worker_tsdb_pool_precheck <现网 Worker 配置> <新包 Worker 模板>
#
# 纯只读门禁，供安装器 Step 1 在任何停服、数据迁移或配置改写前调用。
# 首次部署尚无现网配置时放行；存量配置只允许历史默认 40/96（稍后自动迁移）
# 或已经达到安全预算的值。
validate_worker_tsdb_pool_precheck() {
  local live_config="$1" template_config="$2"
  local live_limit template_limit

  [ -f "$template_config" ] || return 1
  template_limit="$(worker_tsdb_max_conns "$template_config")" || return 1
  [ "$template_limit" -eq "$WORKER_TSDB_SAFE_POOL" ] || return 1

  [ -f "$live_config" ] || return 0
  live_limit="$(worker_tsdb_max_conns "$live_config")" || return 1
  [ "$live_limit" -eq 40 ] || [ "$live_limit" -eq 96 ] || [ "$live_limit" -ge "$WORKER_TSDB_SAFE_POOL" ]
}

# upgrade_worker_tsdb_pool <现网 Worker 配置> <新包 Worker 模板>
#
# 仅把项目历史默认值 40/96 迁移到新模板值。足够大的运维自定义值保持不变；
# 小于新模板安全预算的自定义值不自动覆盖，而是返回 insufficient 让安装器在
# 切换 current 和重启服务前明确阻断。结果通过
# WORKER_TSDB_POOL_UPGRADE_RESULT 返回。
upgrade_worker_tsdb_pool() {
  local live_config="$1" template_config="$2"
  local live_limit template_limit tmp

  WORKER_TSDB_POOL_UPGRADE_RESULT="invalid"
  [ -f "$live_config" ] || return 1
  [ -f "$template_config" ] || return 1

  live_limit="$(worker_tsdb_max_conns "$live_config")" || return 1
  template_limit="$(worker_tsdb_max_conns "$template_config")" || return 1
  [ "$template_limit" -eq "$WORKER_TSDB_SAFE_POOL" ] || return 1

  if [ "$live_limit" -eq "$WORKER_TSDB_SAFE_POOL" ]; then
    WORKER_TSDB_POOL_UPGRADE_RESULT="noop"
    return 0
  fi
  if [ "$live_limit" -gt "$WORKER_TSDB_SAFE_POOL" ]; then
    WORKER_TSDB_POOL_UPGRADE_RESULT="preserved"
    return 0
  fi
  if [ "$live_limit" -ne 40 ] && [ "$live_limit" -ne 96 ]; then
    WORKER_TSDB_POOL_UPGRADE_RESULT="insufficient"
    return 0
  fi

  tmp="$(mktemp "${live_config}.tmp.XXXXXX")" || return 1
  if ! cp -p "$live_config" "$tmp"; then
    rm -f "$tmp"
    return 1
  fi
  if ! awk -v target="$WORKER_TSDB_SAFE_POOL" '
      BEGIN { in_tsdb = 0; direct_indent = -1; changed = 0 }
      /^[^[:space:]#][^:]*:/ {
        in_tsdb = ($0 ~ /^tsdb:[[:space:]]*(#.*)?$/)
        direct_indent = -1
      }
      in_tsdb && $0 ~ /^[[:space:]]+/ && $0 !~ /^[[:space:]]*(#|$)/ {
        match($0, /[^[:space:]]/)
        indent = RSTART - 1
        if (direct_indent < 0) direct_indent = indent
      }
      in_tsdb && indent == direct_indent && !changed &&
        /^[[:space:]]+max_conns:[[:space:]]*/ {
        body = $0
        comment = ""
        if (index(body, "#") > 0) {
          comment = " " substr(body, index(body, "#"))
          body = substr(body, 1, index(body, "#") - 1)
        }
        scalar = body
        sub(/^[^:]*:[[:space:]]*/, "", scalar)
        sub(/^[[:space:]]+/, "", scalar)
        sub(/[[:space:]]+$/, "", scalar)
        quote = ""
        if (scalar ~ /^"[0-9]+"$/) {
          quote = "\""
          scalar = substr(scalar, 2, length(scalar) - 2)
        } else if (scalar ~ /^\047[0-9]+\047$/) {
          quote = sprintf("%c", 39)
          scalar = substr(scalar, 2, length(scalar) - 2)
        }
        if (scalar != "40" && scalar != "96") {
          print
          next
        }
        prefix = body
        sub(/max_conns:.*/, "", prefix)
        print prefix "max_conns: " quote target quote comment
        changed = 1
        next
      }
      { print }
      END { if (changed != 1) exit 42 }
    ' "$live_config" > "$tmp"; then
    rm -f "$tmp"
    return 1
  fi

  if ! mv -f "$tmp" "$live_config"; then
    rm -f "$tmp"
    return 1
  fi

  [ "$(worker_tsdb_max_conns "$live_config")" -eq "$WORKER_TSDB_SAFE_POOL" ] || return 1
  WORKER_TSDB_POOL_UPGRADE_RESULT="migrated"
}

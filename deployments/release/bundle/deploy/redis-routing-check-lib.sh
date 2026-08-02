#!/usr/bin/env bash

yaml_top_level_section_has_address() {
  local config="$1" section="$2" expected="$3"
  [ -f "$config" ] || return 1
  awk -v wanted_section="$section" -v expected="$expected" '
    /^[^[:space:]#][^:]*:/ {
      current=$0
      sub(/:.*/, "", current)
      in_section=(current == wanted_section)
      in_addrs=0
      next
    }
    in_section && /^[[:space:]]+addrs:[[:space:]]*(#.*)?$/ {
      match($0, /[^[:space:]]/)
      addrs_indent=RSTART-1
      in_addrs=1
      next
    }
    in_section && in_addrs && $0 !~ /^[[:space:]]*(#|$)/ {
      match($0, /[^[:space:]]/)
      indent=RSTART-1
      if (indent <= addrs_indent) {
        in_addrs=0
        next
      }
      value=$0
      sub(/^[[:space:]]*-[[:space:]]*/, "", value)
      sub(/[[:space:]]+#.*$/, "", value)
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      if ((substr(value,1,1) == "\"" && substr(value,length(value),1) == "\"") ||
          (substr(value,1,1) == "\047" && substr(value,length(value),1) == "\047")) {
        value=substr(value,2,length(value)-2)
      }
      if (value == expected) found=1
    }
    END { if (!found) exit 1 }
  ' "$config"
}

redis_routing_configs_valid() {
  local config_dir="$1" config_file
  for config_file in app.prod.yaml worker.prod.yaml; do
    yaml_top_level_section_has_address "$config_dir/$config_file" redis redis-core:6379 || return 1
    yaml_top_level_section_has_address "$config_dir/$config_file" pm_redis redis-pm:6379 || return 1
  done
}

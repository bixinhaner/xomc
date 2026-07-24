#!/usr/bin/env bash

monitoring_profile_read_state() {
  local env_file="$1"
  [ -f "$env_file" ] || { printf '0\n'; return 0; }
  awk -F= '
    $1 == "OMCGO_SKIP_MONITORING" { value=$2 }
    END { if (value == "1") print "1"; else print "0" }
  ' "$env_file"
}

monitoring_profile_write_state() {
  local env_file="$1" state="$2" tmp
  case "$state" in
    0|1) ;;
    *) return 1 ;;
  esac
  tmp="$(mktemp "${env_file}.monitoring.XXXXXX")" || return 1
  if [ -f "$env_file" ]; then
    awk -F= -v state="$state" '
      $1 == "OMCGO_SKIP_MONITORING" {
        if (!written) print "OMCGO_SKIP_MONITORING=" state
        written=1
        next
      }
      { print }
      END {
        if (!written) print "OMCGO_SKIP_MONITORING=" state
      }
    ' "$env_file" > "$tmp"
  else
    printf 'OMCGO_SKIP_MONITORING=%s\n' "$state" > "$tmp"
  fi
  mv "$tmp" "$env_file"
}

# Called by install.sh only after release .env has been sourced. Persisting a
# separate profile flag keeps operator OMCGO_TRACER_ENABLED preferences intact:
# the monitoring-free profile overrides tracing only at runtime.
monitoring_profile_apply_install() {
  local env_file="$1" requested_skip="$2"
  monitoring_profile_write_state "$env_file" "$requested_skip" || return 1
  export OMCGO_SKIP_MONITORING="$requested_skip"
  if [ "$requested_skip" = 1 ]; then
    export OMCGO_TRACER_ENABLED=false
  fi
}

# Called by standalone svc.sh / healthcheck.sh. Explicit CLI/environment skip
# wins; otherwise the persisted install profile is authoritative.
monitoring_profile_apply_runtime() {
  local env_file="$1" explicit_skip="${2:-0}" inherited_skip="${OMCGO_SKIP_MONITORING:-0}" state
  if [ "$explicit_skip" = 1 ] || [ "$inherited_skip" = 1 ]; then
    state=1
  else
    state="$(monitoring_profile_read_state "$env_file")"
  fi
  SKIP_MONITORING="$state"
  export OMCGO_SKIP_MONITORING="$state"
  if [ "$state" = 1 ]; then
    export OMCGO_TRACER_ENABLED=false
  fi
}

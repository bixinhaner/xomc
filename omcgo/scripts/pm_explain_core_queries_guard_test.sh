#!/usr/bin/env bash
set -euo pipefail

for tool in initdb pg_ctl createdb psql; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "SKIP: $tool is not installed"
    exit 0
  fi
done

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
sql_path="$script_dir/pm_explain_core_queries.sql"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/pm-task8-guard.XXXXXX")"
data_dir="$test_root/data"
socket_dir="$test_root/socket"
database="task8_guard_test"
mkdir -p "$socket_dir"

cleanup() {
  if [[ -f "$data_dir/postmaster.pid" ]]; then
    pg_ctl -D "$data_dir" -m immediate stop >/dev/null 2>&1 || true
  fi
  rm -rf -- "$test_root"
}
trap cleanup EXIT

initdb -D "$data_dir" -A trust -U postgres --no-locale >/dev/null
pg_ctl -D "$data_dir" -o "-h '' -k '$socket_dir'" -w start >/dev/null
createdb -h "$socket_dir" -U postgres "$database"

expect_refusal() {
  local name="$1"
  local expected_message="$2"
  shift 2
  local output="$test_root/$name.out"
  local status

  set +e
  psql -X -h "$socket_dir" -U postgres -d "$database" \
    "$@" -f "$sql_path" >"$output" 2>&1
  status=$?
  set -e

  if [[ "$status" -ne 3 ]]; then
    echo "$name: exit=$status, want 3"
    cat "$output"
    return 1
  fi
  if ! grep -Fq "$expected_message" "$output"; then
    echo "$name: missing refusal message: $expected_message"
    cat "$output"
    return 1
  fi
  if grep -Fq "QUERY " "$output"; then
    echo "$name: reached an EXPLAIN/DELETE query"
    cat "$output"
    return 1
  fi
}

expect_refusal \
  flag_false \
  "isolation flag must be on" \
  -v task8_isolated_safe_environment=off \
  -v task8_expected_database="$database"

expect_refusal \
  wrong_database \
  "database identity mismatch" \
  -v task8_isolated_safe_environment=on \
  -v task8_expected_database=some_other_database

expect_refusal \
  absent_sentinel \
  "clone sentinel missing" \
  -v task8_isolated_safe_environment=on \
  -v task8_expected_database="$database"

echo "PASS: EXPLAIN guard refusal paths exit 3 before dangerous statements"

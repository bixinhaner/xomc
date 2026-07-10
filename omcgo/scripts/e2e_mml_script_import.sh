#!/usr/bin/env bash
# Reproducible, endpoint-backed MML TXT import E2E.
#
# Usage:
#   OMC_TOKEN=<jwt> MML_E2E_SN=<provisioned-sn> \
#     bash omcgo/scripts/e2e_mml_script_import.sh [BASE_URL]
#
# The script intentionally fails (rather than silently skipping) when the
# stack, credentials, or a provisioned device are unavailable. This keeps an
# environment limitation visible in the evidence log.
set -uo pipefail

BASE_URL="${1:-${MML_E2E_BASE_URL:-http://localhost:8081}}"
API="${BASE_URL%/}/api/v1"
TOKEN="${OMC_TOKEN:-${AUTH_TOKEN:-}}"
SN="${MML_E2E_SN:-1202000091177SP0005}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VALID_FIXTURE="$ROOT_DIR/internal/mml/testdata/import-valid.txt"
INVALID_FIXTURE="$ROOT_DIR/internal/mml/testdata/import-invalid.txt"

PASS=0
FAIL=0
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mml-import-e2e.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

pass() { PASS=$((PASS + 1)); echo "[PASS] $*"; }
fail() { FAIL=$((FAIL + 1)); echo "[FAIL] $*" >&2; }

json_value() {
  local path="$1" file="$2"
  python3 - "$path" "$file" <<'PY'
import json, sys
try:
    with open(sys.argv[2], encoding='utf-8') as fh:
        value = json.load(fh)
    for part in sys.argv[1].split('.'):
        if isinstance(value, dict):
            value = value.get(part)
        else:
            value = None
            break
    if value is not None:
        print(value if not isinstance(value, (dict, list)) else json.dumps(value, separators=(',', ':')))
except Exception:
    pass
PY
}

request() {
  local output="$1"; shift
  local status
  status="$(curl --max-time "${MML_E2E_TIMEOUT:-10}" -sS -o "$output" -w '%{http_code}' "$@" 2>"$output.curl.err" || printf '000')"
  printf '%s' "$status"
}

show_response() {
  local label="$1" status="$2" body="$3"
  echo "--- $label: HTTP $status ---"
  sed -e 's/[[:space:]]\+/ /g' "$body" | head -c 1200 || true
  echo
  if [ -s "$body.curl.err" ]; then
    echo "curl: $(tr '\n' ' ' < "${body}.curl.err")"
  fi
}

assert_status() {
  local label="$1" expected="$2" actual="$3" body="$4"
  if [ "$actual" = "$expected" ]; then
    pass "$label (HTTP $actual)"
  else
    fail "$label: expected HTTP $expected, got $actual"
    show_response "$label" "$actual" "$body"
  fi
}

assert_json() {
  local label="$1" body="$2" expr="$3"
  if python3 - "$expr" "$body" <<'PY'
import json, sys
expr = sys.argv[1]
try:
    with open(sys.argv[2], encoding='utf-8') as fh:
        data = json.load(fh)
    # Assertions are intentionally tiny and explicit; this is not a parser
    # for arbitrary shell input.
    if expr == 'validation_token': ok = bool(data.get('data', {}).get('validation_token'))
    elif expr == 'no_validation_token': ok = not bool(data.get('data', {}).get('validation_token'))
    elif expr == 'script_id': ok = bool(data.get('data', {}).get('id') or data.get('data', {}).get('script', {}).get('id'))
    elif expr == 'task_id': ok = bool(data.get('data', {}).get('task', {}).get('id') or data.get('data', {}).get('id'))
    elif expr == 'same_script_id': ok = bool(data.get('data', {}).get('id'))
    elif expr == 'result_trace_fields':
        text = json.dumps(data)
        ok = all(k in text for k in ('plan_line_no', 'plan_device_sn', 'plan_order'))
    elif expr == 'task_snapshot_hash':
        ok = bool(data.get('data', {}).get('script_content_sha256'))
    else: ok = False
except Exception:
    ok = False
sys.exit(0 if ok else 1)
PY
  then pass "$label"; else fail "$label"; show_response "$label" "n/a" "$body"; fi
}

AUTH_ARGS=(-H "Authorization: Bearer ${TOKEN}")
if [ -n "$TOKEN" ]; then :; else
  echo "[WARN] OMC_TOKEN/AUTH_TOKEN is empty; authenticated assertions should fail visibly."
fi

echo "MML script import E2E against $BASE_URL (SN=$SN)"

health="$TMP_DIR/health.json"
status="$(request "$health" "$BASE_URL/healthz")"
assert_status "health check" 200 "$status" "$health"

invalid="$TMP_DIR/invalid.json"
status="$(request "$invalid" -X POST "$API/mml/scripts/import/validate" "${AUTH_ARGS[@]}" -F "file=@$INVALID_FIXTURE;filename=import-invalid.txt")"
assert_status "invalid TXT is rejected" 422 "$status" "$invalid"
assert_json "invalid TXT cannot produce a save token" "$invalid" no_validation_token

valid_fixture="$TMP_DIR/import-valid.txt"
sed "s/1202000091177SP0005/$SN/g" "$VALID_FIXTURE" >"$valid_fixture"
valid="$TMP_DIR/valid.json"
status="$(request "$valid" -X POST "$API/mml/scripts/import/validate" "${AUTH_ARGS[@]}" -F "file=@$valid_fixture;filename=import-valid.txt")"
assert_status "valid TXT validation" 200 "$status" "$valid"
assert_json "valid TXT returns one-time token" "$valid" validation_token
TOKEN_VALUE="$(json_value data.validation_token "$valid")"

if [ -n "$TOKEN_VALUE" ]; then
  save="$TMP_DIR/save.json"
  save_body="{\"validation_token\":\"$TOKEN_VALUE\",\"script_name\":\"E2E TXT import\",\"description\":\"task-13 endpoint verification\",\"tags\":[\"e2e\"]}"
  status="$(request "$save" -X POST "$API/mml/scripts/import" "${AUTH_ARGS[@]}" -H 'Content-Type: application/json' -d "$save_body")"
  assert_status "validated TXT can be saved" 201 "$status" "$save"
  assert_json "save returns script id" "$save" script_id
  SCRIPT_ID="$(json_value data.id "$save")"
  [ -n "$SCRIPT_ID" ] || SCRIPT_ID="$(json_value data.script.id "$save")"

  replay="$TMP_DIR/replay.json"
  status="$(request "$replay" -X POST "$API/mml/scripts/import" "${AUTH_ARGS[@]}" -H 'Content-Type: application/json' -d "$save_body")"
  assert_status "same token replay is idempotent" 201 "$status" "$replay"
  REPLAY_SCRIPT_ID="$(json_value data.id "$replay")"
  if [ -n "$SCRIPT_ID" ] && [ "$REPLAY_SCRIPT_ID" = "$SCRIPT_ID" ]; then
    pass "replay returns the original script"
  else
    fail "replay returns the original script"
    show_response "replay returns the original script" "$status" "$replay"
  fi

  if [ -n "$SCRIPT_ID" ]; then
    execution="$TMP_DIR/execution.json"
    execution_body='{"task_name":"E2E TXT execution","execute_type":"immediate","confirm_warnings":true}'
    status="$(request "$execution" -X POST "$API/mml/scripts/$SCRIPT_ID/executions" "${AUTH_ARGS[@]}" -H 'Content-Type: application/json' -d "$execution_body")"
    assert_status "execution uses server snapshot" 201 "$status" "$execution"
    assert_json "execution returns task id" "$execution" task_id
    TASK_ID="$(json_value data.task.id "$execution")"
    [ -n "$TASK_ID" ] && {
      task="$TMP_DIR/task.json"
      status="$(request "$task" -X GET "$API/mml/tasks/$TASK_ID" "${AUTH_ARGS[@]}")"
      assert_status "task snapshot is readable" 200 "$status" "$task"
      assert_json "task stores immutable script hash" "$task" task_snapshot_hash
      result="$TMP_DIR/result.json"
      status="$(request "$result" -X GET "$API/mml/tasks/$TASK_ID/results" "${AUTH_ARGS[@]}")"
      assert_status "task results are traceable" 200 "$status" "$result"
      assert_json "results expose immutable plan trace fields" "$result" result_trace_fields
    }
  fi
else
  fail "save/execution phase not attempted because validation returned no token"
fi

echo "MML script import E2E: $([ "$FAIL" -eq 0 ] && echo PASS || echo FAIL) (pass=$PASS fail=$FAIL)"
[ "$FAIL" -eq 0 ]

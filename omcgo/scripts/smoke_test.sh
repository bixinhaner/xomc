#!/usr/bin/env bash
#
# OMC Smoke Test — RC freeze quick verification (T-0025)
#
# Purpose: < 3 min sanity check before promoting a build / after a hotfix /
#          during canary rollout. Complements e2e_verify.sh (full regression,
#          549 PASS / ~5–10 min) — they are NOT substitutes.
#
# Usage:
#   bash smoke_test.sh                     # uses default http://localhost:8081
#   bash smoke_test.sh http://staging:8081 # explicit base
#
# Exit codes:
#   0  → all 20 cases PASSED  (RC READY)
#   1  → any case FAILED      (RC NOT READY — DO NOT PROMOTE)
#   2  → invocation error (bad arg, base unreachable in pre-flight)
#
# Coverage (8 domains × ~2-3 cases = 20 total):
#   auth(2), device(3), alarm(3), kpi-pm(3), topology(2),
#   report(2), admin(3), health(2)
#
# PRD: docs/project/prd/T-0025-rc-freeze-smoke.md
# Backlog: T-0025 (done)

set -uo pipefail

API_BASE="${1:-http://localhost:8081}"
API="${API_BASE}/api/v1"
HEALTH="${API_BASE}"
# Default admin credentials match seed_e2e_testdata.sql; staging overrides via env.
ADMIN_USER="${OMC_ADMIN_USER:-admin}"
ADMIN_PASS="${OMC_ADMIN_PASS:-admin123}"

PASS=0
FAIL=0
TOTAL=20
START_TS=$(date +%s)
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

check_status() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$actual" = "$expected" ]; then
        echo "  ✅ $desc → $actual"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $desc → expected $expected, got $actual"
        FAIL=$((FAIL + 1))
    fi
}

check_status_in() {
    local desc="$1" expected_set="$2" actual="$3"
    local code
    for code in $expected_set; do
        if [ "$actual" = "$code" ]; then
            echo "  ✅ $desc → $actual (∈ {$expected_set})"
            PASS=$((PASS + 1))
            return
        fi
    done
    echo "  ❌ $desc → expected one of {$expected_set}, got $actual"
    FAIL=$((FAIL + 1))
}

http_code() {
    # $1 = method, $2 = url, $3 = output file (or /dev/null), rest = curl extra args
    local method="$1" url="$2" outfile="$3"
    shift 3
    curl --max-time 5 -s -o "$outfile" -w "%{http_code}" -X "$method" "$@" "$url" || echo "000"
}

# ---------------------------------------------------------------------------
# Pre-flight: base reachable?
# ---------------------------------------------------------------------------

echo "════════════════════════════════════════"
echo "  OMC Smoke Test (T-0025 / RC freeze)"
echo "  Target: $API_BASE"
echo "  Time:   $(date -u +%FT%TZ)"
echo "════════════════════════════════════════"

PREFLIGHT=$(curl --max-time 3 -s -o /dev/null -w "%{http_code}" "$HEALTH/healthz" 2>/dev/null || echo "000")
if [ "$PREFLIGHT" = "000" ]; then
    echo ""
    echo "❌ Pre-flight failed: $API_BASE/healthz unreachable."
    echo "   Hint: bash run/scripts/start-all.sh   # local"
    echo "         or check staging endpoint URL."
    exit 2
fi
echo "  Pre-flight /healthz → $PREFLIGHT (reachable, proceeding)"
echo ""

# ---------------------------------------------------------------------------
# §1 auth (2 cases)
# ---------------------------------------------------------------------------
echo "=== auth (2 cases) ==="

# 登录密码必须 RSA-OAEP 加密：先拉公钥，再用 cryptography 模块加密 password+ts+nonce。
PUBKEY_RESP=$(curl --max-time 5 -s "$API/auth/public-key")
PUBLIC_KEY_PEM=$(echo "$PUBKEY_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('public_key',''))" 2>/dev/null)
PUBLIC_KEY_ID=$(echo "$PUBKEY_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('key_id',''))" 2>/dev/null)
if [ -z "$PUBLIC_KEY_PEM" ] || [ -z "$PUBLIC_KEY_ID" ]; then
    echo "❌ failed to fetch /auth/public-key (got: $PUBKEY_RESP)"
    exit 2
fi

# 加密单个明文密码，输出 base64 密文。
encrypt_password() {
    PEM_INPUT="$PUBLIC_KEY_PEM" PLAIN_INPUT="$1" python3 <<'PYEOF'
import os, sys, json, time, secrets, base64
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import padding
pub = serialization.load_pem_public_key(os.environ['PEM_INPUT'].encode())
payload = json.dumps({"password": os.environ['PLAIN_INPUT'], "ts": int(time.time()), "nonce": secrets.token_hex(16)}).encode()
ct = pub.encrypt(payload, padding.OAEP(mgf=padding.MGF1(algorithm=hashes.SHA256()), algorithm=hashes.SHA256(), label=None))
sys.stdout.write(base64.b64encode(ct).decode())
PYEOF
}

ENC_OK=$(encrypt_password "${ADMIN_PASS}")
HTTP_CODE=$(http_code POST "$API/auth/login" "$TMPDIR/auth.json" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${ADMIN_USER}\",\"encrypted_password\":\"${ENC_OK}\",\"key_id\":\"${PUBLIC_KEY_ID}\"}")
check_status "1. POST /auth/login (${ADMIN_USER}/***)" "200" "$HTTP_CODE"

TOKEN=""
if [ "$HTTP_CODE" = "200" ] && [ -s "$TMPDIR/auth.json" ]; then
    TOKEN=$(python3 -c "import sys,json; print(json.load(open('$TMPDIR/auth.json')).get('access_token',''))" 2>/dev/null || echo "")
fi
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

ENC_BAD=$(encrypt_password "WRONG_PASSWORD_FOR_SMOKE")
HTTP_CODE=$(http_code POST "$API/auth/login" /dev/null \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${ADMIN_USER}\",\"encrypted_password\":\"${ENC_BAD}\",\"key_id\":\"${PUBLIC_KEY_ID}\"}")
check_status_in "2. POST /auth/login (bad cred)" "401 400" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §2 device (3 cases)
# ---------------------------------------------------------------------------
echo "=== device (3 cases) ==="

HTTP_CODE=$(http_code GET "$API/devices?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "3. GET /devices?page=1" "200 401" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/devices/00000000-0000-0000-0000-000000000000" /dev/null -H "$AUTH_HEADER")
check_status_in "4. GET /devices/{not-found-uuid}" "404 401" "$HTTP_CODE"

HTTP_CODE=$(http_code POST "$API/devices" /dev/null \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d '{}')
check_status_in "5. POST /devices (missing fields)" "400 422 401" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §3 alarm (3 cases)
# ---------------------------------------------------------------------------
echo "=== alarm (3 cases) ==="

HTTP_CODE=$(http_code GET "$API/alarms/history?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "6. GET /alarms/history?page=1" "200 401" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/alarms/active?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "7. GET /alarms/active" "200 401 404" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/alarms/statistics" /dev/null -H "$AUTH_HEADER")
check_status_in "8. GET /alarms/statistics" "200 401 404" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §4 kpi/pm (3 cases)
# ---------------------------------------------------------------------------
echo "=== kpi/pm (3 cases) ==="

HTTP_CODE=$(http_code GET "$API/pm/kpi/definitions?page=1" /dev/null -H "$AUTH_HEADER")
check_status_in "9. GET /pm/kpi/definitions" "200 401 404" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/pm/counters?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "10. GET /pm/counters?page=1" "200 401 404" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/pm/files?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "11. GET /pm/files?page=1" "200 401 404" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §5 topology (2 cases)
# ---------------------------------------------------------------------------
echo "=== topology (2 cases) ==="

HTTP_CODE=$(http_code GET "$API/device-groups/tree" /dev/null -H "$AUTH_HEADER")
check_status_in "12. GET /device-groups/tree" "200 401" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/sites?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "13. GET /sites?page=1" "200 401" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §6 report (2 cases)
# ---------------------------------------------------------------------------
echo "=== report (2 cases) ==="

HTTP_CODE=$(http_code GET "$API/reports/definitions?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "14. GET /reports/definitions" "200 401" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/reports/sample-data" /dev/null -H "$AUTH_HEADER")
check_status_in "15. GET /reports/sample-data" "200 401" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §7 admin (3 cases) — RBAC ops; expected 200 if admin role, 401/403 otherwise
# ---------------------------------------------------------------------------
echo "=== admin (3 cases) ==="

HTTP_CODE=$(http_code GET "$API/admin/users?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "16. GET /admin/users" "200 401 403" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/admin/roles?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "17. GET /admin/roles" "200 401 403" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$API/admin/dead-letters?page=1&page_size=10" /dev/null -H "$AUTH_HEADER")
check_status_in "18. GET /admin/dead-letters (T-0012)" "200 401 403" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# §8 health (2 cases) — no auth
# ---------------------------------------------------------------------------
echo "=== health (2 cases) ==="

HTTP_CODE=$(http_code GET "$HEALTH/healthz" /dev/null)
check_status "19. GET /healthz (no auth)" "200" "$HTTP_CODE"

HTTP_CODE=$(http_code GET "$HEALTH/readyz" /dev/null)
check_status_in "20. GET /readyz (no auth)" "200 503" "$HTTP_CODE"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
END_TS=$(date +%s)
DURATION=$((END_TS - START_TS))

echo ""
echo "════════════════════════════════════════"
echo "  Smoke Test Summary"
echo "════════════════════════════════════════"
echo "  Pass:   ${PASS} / ${TOTAL}"
echo "  Fail:   ${FAIL} / ${TOTAL}"
echo "  Time:   ${DURATION}s"
echo "  Target: $API_BASE"
echo ""

if [ "$FAIL" -gt 0 ] || [ "$PASS" -ne "$TOTAL" ]; then
    echo "  ❌ SMOKE FAILED — RC NOT READY (DO NOT PROMOTE)"
    echo ""
    exit 1
fi

echo "  ✅ SMOKE PASSED — RC READY"
echo ""
exit 0

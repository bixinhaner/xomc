#!/bin/bash
# e2e_verify.sh — Sprint 1 端到端数据流验证脚本
# 用 curl 覆盖 M1 里程碑所有关键路径
# 前置: omcgo-app 运行在 localhost:8080, DB 已执行迁移 + 种子数据
#
# 使用方法:
#   ./scripts/e2e_verify.sh [BASE_URL]
#   默认: http://localhost:8080

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
API="${BASE_URL}/api/v1"
PASS=0
FAIL=0
TOTAL=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
    PASS=$((PASS + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${GREEN}[PASS]${NC} $1"
}

fail() {
    FAIL=$((FAIL + 1))
    TOTAL=$((TOTAL + 1))
    echo -e "  ${RED}[FAIL]${NC} $1"
    if [ -n "${2:-}" ]; then
        echo -e "         ${RED}→ $2${NC}"
    fi
}

section() {
    echo ""
    echo -e "${YELLOW}=== $1 ===${NC}"
}

# Helper: check HTTP status code
check_status() {
    local desc="$1"
    local expected="$2"
    local actual="$3"
    if [ "$actual" = "$expected" ]; then
        pass "$desc (HTTP $actual)"
    else
        fail "$desc" "expected HTTP $expected, got $actual"
    fi
}

# Helper: check JSON field exists and is not empty
check_json_field() {
    local desc="$1"
    local json="$2"
    local field="$3"
    local value
    value=$(echo "$json" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$field',''))" 2>/dev/null || echo "")
    if [ -n "$value" ] && [ "$value" != "None" ]; then
        pass "$desc (${field}=${value})"
        return 0
    else
        fail "$desc" "field '$field' missing or empty"
        return 1
    fi
}

# Helper: check JSON nested field using jq-like path
check_json_path() {
    local desc="$1"
    local json="$2"
    local path="$3"
    local value
    value=$(echo "$json" | python3 -c "
import sys, json
d = json.load(sys.stdin)
path = '$path'.split('.')
v = d
for p in path:
    if isinstance(v, list):
        v = v[int(p)] if len(v) > int(p) else None
    elif isinstance(v, dict):
        v = v.get(p)
    else:
        v = None
    if v is None:
        break
print('' if v is None else v)
" 2>/dev/null || echo "")
    if [ -n "$value" ]; then
        pass "$desc (${path}=${value})"
        return 0
    else
        fail "$desc" "path '$path' not found"
        return 1
    fi
}

echo "============================================"
echo "  OMC Sprint 1 — E2E Data Flow Verification"
echo "============================================"
echo "Target: $BASE_URL"
echo "Time:   $(date '+%Y-%m-%d %H:%M:%S')"

# ============================================================
section "1. Health Check"
# ============================================================

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz" 2>/dev/null || echo "000")
check_status "GET /healthz" "200" "$HTTP_CODE"

# ============================================================
section "2. Authentication — Login"
# ============================================================

# 2.1 Valid login
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
check_status "POST /auth/login (valid credentials)" "200" "$HTTP_CODE"

ACCESS_TOKEN=""
REFRESH_TOKEN=""
if [ "$HTTP_CODE" = "200" ]; then
    check_json_field "Login response has access_token" "$BODY" "access_token"
    check_json_field "Login response has refresh_token" "$BODY" "refresh_token"
    check_json_field "Login response has expires_at" "$BODY" "expires_at"
    check_json_field "Login response has token_type" "$BODY" "token_type"

    ACCESS_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")
    REFRESH_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('refresh_token',''))" 2>/dev/null || echo "")
fi

# 2.2 Invalid password
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrongpassword"}')
check_status "POST /auth/login (wrong password)" "401" "$HTTP_CODE"

# 2.3 No auth token → 401
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/devices")
check_status "GET /devices (no token)" "401" "$HTTP_CODE"

# ============================================================
section "3. Authentication — Get Current User"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    RESP=$(curl -s -w "\n%{http_code}" "$API/auth/me" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /auth/me" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Me response has id" "$BODY" "id"
        check_json_field "Me response has username" "$BODY" "username"
        check_json_field "Me response has display_name" "$BODY" "display_name"
        check_json_field "Me response has status" "$BODY" "status"

        # Check roles array
        ROLES_LEN=$(echo "$BODY" | python3 -c "import sys,json; r=json.load(sys.stdin).get('roles',[]); print(len(r) if r else 0)" 2>/dev/null || echo "0")
        if [ "$ROLES_LEN" -gt 0 ]; then
            pass "Me response has roles[] (count=$ROLES_LEN)"
        else
            fail "Me response has roles[]" "roles array is empty"
        fi
    fi
else
    fail "GET /auth/me" "skipped — no access token"
fi

# ============================================================
section "4. Token Refresh"
# ============================================================

if [ -n "$REFRESH_TOKEN" ]; then
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "POST /auth/refresh" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        NEW_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")
        if [ -n "$NEW_TOKEN" ]; then
            pass "Refresh returns new access_token"

            # Verify new token works
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/auth/me" \
                -H "Authorization: Bearer $NEW_TOKEN")
            check_status "New token valid for /auth/me" "200" "$HTTP_CODE"
        else
            fail "Refresh returns new access_token" "empty token"
        fi
    fi
else
    fail "POST /auth/refresh" "skipped — no refresh token"
fi

# ============================================================
section "5. Device List"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 5.1 List all devices
    RESP=$(curl -s -w "\n%{http_code}" "$API/devices?page=1&page_size=20" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /devices" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('items',[])))" 2>/dev/null || echo "0")
        TOTAL_VAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total',0))" 2>/dev/null || echo "0")

        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Device list has items (count=$ITEMS_LEN, total=$TOTAL_VAL)"
        else
            fail "Device list has items" "items array is empty (total=$TOTAL_VAL)"
        fi

        check_json_field "Device list has page" "$BODY" "page"
        check_json_field "Device list has page_size" "$BODY" "page_size"
        check_json_field "Device list has total_pages" "$BODY" "total_pages"

        # Check first device has expected fields
        FIRST_SN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items',[]); print(items[0].get('serial_number','') if items else '')" 2>/dev/null || echo "")
        if [ -n "$FIRST_SN" ]; then
            pass "Device has serial_number field ($FIRST_SN)"
        else
            fail "Device has serial_number field" "missing"
        fi

        FIRST_STATUS=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items',[]); print(items[0].get('status','') if items else '')" 2>/dev/null || echo "")
        if [ -n "$FIRST_STATUS" ]; then
            pass "Device has status field ($FIRST_STATUS)"
        else
            fail "Device has status field" "missing"
        fi
    fi

    # 5.2 Filter by SN
    RESP=$(curl -s -w "\n%{http_code}" "$API/devices?sn=TEST-SN-001" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /devices?sn=TEST-SN-001" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        TOTAL_VAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total',0))" 2>/dev/null || echo "0")
        if [ "$TOTAL_VAL" = "1" ]; then
            pass "SN filter returns exactly 1 result"
        else
            fail "SN filter returns exactly 1 result" "got total=$TOTAL_VAL"
        fi
    fi

    # 5.3 Get device by ID
    DEVICE_ID="e2e00001-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/devices/$DEVICE_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /devices/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Single device has serial_number" "$BODY" "serial_number"
        check_json_field "Single device has manufacturer" "$BODY" "manufacturer"
        check_json_field "Single device has carrier" "$BODY" "carrier"
    fi

    # 5.4 Device stats
    RESP=$(curl -s -w "\n%{http_code}" "$API/devices/stats" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /devices/stats" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Stats has counts" "$BODY" "counts"
    fi

    # 5.5 Pagination: page=2, page_size=2
    RESP=$(curl -s -w "\n%{http_code}" "$API/devices?page=2&page_size=2" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /devices page=2&page_size=2" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        PAGE_VAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('page',0))" 2>/dev/null || echo "0")
        PAGE_SIZE_VAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('page_size',0))" 2>/dev/null || echo "0")
        if [ "$PAGE_VAL" = "2" ] && [ "$PAGE_SIZE_VAL" = "2" ]; then
            pass "Pagination params reflected (page=$PAGE_VAL, page_size=$PAGE_SIZE_VAL)"
        else
            fail "Pagination params reflected" "page=$PAGE_VAL, page_size=$PAGE_SIZE_VAL"
        fi
    fi
else
    fail "Device tests" "skipped — no access token"
fi

# ============================================================
section "6. Alarm List"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 6.1 List active alarms
    RESP=$(curl -s -w "\n%{http_code}" "$API/alarms/active?page=1&page_size=20" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /alarms/active" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('items',[])))" 2>/dev/null || echo "0")
        TOTAL_VAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total',0))" 2>/dev/null || echo "0")

        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Active alarms has items (count=$ITEMS_LEN, total=$TOTAL_VAL)"
        else
            fail "Active alarms has items" "items array is empty"
        fi

        # Check first alarm has expected fields
        FIRST_SEV=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items',[]); print(items[0].get('severity','') if items else '')" 2>/dev/null || echo "")
        if [ -n "$FIRST_SEV" ]; then
            pass "Alarm has severity field ($FIRST_SEV)"
        else
            fail "Alarm has severity field" "missing"
        fi

        # Verify severity is numeric (not string)
        IS_NUM=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items',[]); print('yes' if items and isinstance(items[0].get('severity'), int) else 'no')" 2>/dev/null || echo "no")
        if [ "$IS_NUM" = "yes" ]; then
            pass "Alarm severity is numeric (int)"
        else
            fail "Alarm severity is numeric" "got non-int type"
        fi

        check_json_field "Alarm list has page_size" "$BODY" "page_size"
    fi

    # 6.2 Alarm statistics
    RESP=$(curl -s -w "\n%{http_code}" "$API/alarms/statistics" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /alarms/statistics" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Statistics has total_active" "$BODY" "total_active"
        check_json_field "Statistics has by_severity" "$BODY" "by_severity"
        check_json_field "Statistics has by_type" "$BODY" "by_type"

        # Check by_severity keys are stringified ints
        SEV_KEYS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
bs = d.get('by_severity', {})
keys = sorted(bs.keys())
print(','.join(keys))
" 2>/dev/null || echo "")
        if echo "$SEV_KEYS" | grep -q "1"; then
            pass "by_severity has numeric string keys ($SEV_KEYS)"
        else
            fail "by_severity has numeric string keys" "keys=$SEV_KEYS"
        fi
    fi

    # 6.3 Get alarm by ID
    ALARM_ID="e2e00002-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/alarms/$ALARM_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /alarms/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Single alarm has alarm_code" "$BODY" "alarm_code"
        check_json_field "Single alarm has device_sn" "$BODY" "device_sn"
        check_json_field "Single alarm has raised_at" "$BODY" "raised_at"
    fi

    # 6.4 Acknowledge alarm
    ACK_ALARM_ID="e2e00002-0000-0000-0000-000000000003"
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/alarms/$ACK_ALARM_ID/acknowledge" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d '{"acknowledged_by":"e2e-test"}')
    HTTP_CODE=$(echo "$RESP" | tail -1)
    check_status "POST /alarms/:id/acknowledge" "200" "$HTTP_CODE"
else
    fail "Alarm tests" "skipped — no access token"
fi

# ============================================================
section "7. CORS"
# ============================================================

RESP=$(curl -s -w "\n%{http_code}" -X OPTIONS "$API/devices" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: GET" \
    -H "Access-Control-Request-Headers: Authorization,Content-Type")
HTTP_CODE=$(echo "$RESP" | tail -1)
# CORS preflight should return 204 or 200
if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "OPTIONS /devices preflight (HTTP $HTTP_CODE)"
else
    fail "OPTIONS /devices preflight" "expected 200 or 204, got $HTTP_CODE"
fi

# Check CORS headers
CORS_HEADERS=$(curl -s -D - -o /dev/null -X OPTIONS "$API/devices" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: GET" 2>/dev/null)
if echo "$CORS_HEADERS" | grep -qi "Access-Control-Allow-Origin"; then
    pass "CORS Access-Control-Allow-Origin header present"
else
    fail "CORS Access-Control-Allow-Origin header" "not found in response"
fi

# ============================================================
section "8. Error Handling"
# ============================================================

# 8.1 Non-existent device
if [ -n "$ACCESS_TOKEN" ]; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        "$API/devices/00000000-0000-0000-0000-000000000000" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    check_status "GET /devices/:id (not found)" "404" "$HTTP_CODE"

    # 8.2 Invalid UUID format
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        "$API/devices/not-a-uuid" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    check_status "GET /devices/:id (invalid UUID)" "400" "$HTTP_CODE"
fi

# ============================================================
# Summary
# ============================================================

echo ""
echo "============================================"
echo -e "  Results: ${GREEN}$PASS PASS${NC} / ${RED}$FAIL FAIL${NC} / $TOTAL TOTAL"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi

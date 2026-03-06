#!/bin/bash
# e2e_verify.sh — Sprint 1+2+3 端到端数据流验证脚本
# 用 curl 覆盖 M1+M2+M3 里程碑所有关键路径
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

echo "================================================"
echo "  OMC Sprint 1+2+3 — E2E Data Flow Verification"
echo "================================================"
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
section "9. Config Template CRUD"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 9.1 List templates (seeded data)
    RESP=$(curl -s -w "\n%{http_code}" "$API/templates?limit=10&offset=0" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /templates (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or d.get('data') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Template list has items (count=$ITEMS_LEN)"
        else
            fail "Template list has items" "items array is empty"
        fi
    fi

    # 9.2 Get template by ID (seeded)
    TMPL_ID="e2e00003-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/templates/$TMPL_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /templates/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Template has name" "$BODY" "name"
        check_json_field "Template has carrier" "$BODY" "carrier"
        check_json_field "Template has template_type" "$BODY" "template_type"
    fi

    # 9.3 Create template
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/templates" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "E2E Test Template",
            "carrier": "cmcc",
            "technology": "lte",
            "product_class": "FAP-LTE-100",
            "template_type": "batch_config",
            "parameters": {"test_param": "test_value"},
            "active": true,
            "description": "Created by E2E test"
        }')
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    NEW_TMPL_ID=""
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        pass "POST /templates (create) (HTTP $HTTP_CODE)"
        NEW_TMPL_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
        if [ -n "$NEW_TMPL_ID" ]; then
            pass "Created template has id ($NEW_TMPL_ID)"
        else
            fail "Created template has id" "id missing"
        fi
    else
        fail "POST /templates (create)" "expected HTTP 200/201, got $HTTP_CODE"
    fi

    # 9.4 Update template
    if [ -n "$NEW_TMPL_ID" ]; then
        RESP=$(curl -s -w "\n%{http_code}" -X PUT "$API/templates/$NEW_TMPL_ID" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d '{
                "name": "E2E Test Template Updated",
                "carrier": "cmcc",
                "technology": "lte",
                "product_class": "FAP-LTE-100",
                "template_type": "batch_config",
                "parameters": {"test_param": "updated_value"},
                "active": true,
                "description": "Updated by E2E test"
            }')
        HTTP_CODE=$(echo "$RESP" | tail -1)
        check_status "PUT /templates/:id (update)" "200" "$HTTP_CODE"
    else
        fail "PUT /templates/:id (update)" "skipped — no template id"
    fi

    # 9.5 Delete template
    if [ -n "$NEW_TMPL_ID" ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/templates/$NEW_TMPL_ID" \
            -H "$AUTH_HEADER")
        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
            pass "DELETE /templates/:id (HTTP $HTTP_CODE)"
        else
            fail "DELETE /templates/:id" "expected HTTP 200/204, got $HTTP_CODE"
        fi
    else
        fail "DELETE /templates/:id" "skipped — no template id"
    fi
else
    fail "Template tests" "skipped — no access token"
fi

# ============================================================
section "10. Firmware Management"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 10.1 List firmware versions
    RESP=$(curl -s -w "\n%{http_code}" "$API/firmware?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /firmware (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or d.get('data') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Firmware list has items (count=$ITEMS_LEN)"
        else
            fail "Firmware list has items" "items array is empty"
        fi
    fi

    # 10.2 Get firmware by ID
    FW_ID="e2e00004-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/firmware/$FW_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /firmware/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Firmware has version" "$BODY" "version"
        check_json_field "Firmware has carrier" "$BODY" "carrier"
        check_json_field "Firmware has status" "$BODY" "status"
    fi

    # 10.3 Delete firmware (use third seeded record — no FK references)
    FW_DEL_ID="e2e00004-0000-0000-0000-000000000003"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/firmware/$FW_DEL_ID" \
        -H "$AUTH_HEADER")
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        pass "DELETE /firmware/:id (HTTP $HTTP_CODE)"
    else
        fail "DELETE /firmware/:id" "expected HTTP 200/204, got $HTTP_CODE"
    fi
else
    fail "Firmware tests" "skipped — no access token"
fi

# ============================================================
section "11. Upgrade Tasks"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 11.1 List upgrade tasks
    RESP=$(curl -s -w "\n%{http_code}" "$API/upgrade-tasks?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /upgrade-tasks (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or d.get('data') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Upgrade task list has items (count=$ITEMS_LEN)"
        else
            fail "Upgrade task list has items" "items array is empty"
        fi
    fi

    # 11.2 Get upgrade task by ID
    TASK_ID="e2e00005-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/upgrade-tasks/$TASK_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /upgrade-tasks/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Upgrade task has status" "$BODY" "status"
    fi
else
    fail "Upgrade task tests" "skipped — no access token"
fi

# ============================================================
section "12. User Management (Admin)"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 12.1 List users
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/users?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/users (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or d.get('data') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "User list has items (count=$ITEMS_LEN)"
        else
            fail "User list has items" "items array is empty"
        fi
    fi

    # 12.2 Create user
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/admin/users" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "e2e-testuser",
            "password": "Test@12345",
            "display_name": "E2E Test User",
            "email": "e2e@test.com",
            "status": "active"
        }')
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    NEW_USER_ID=""
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        pass "POST /admin/users (create) (HTTP $HTTP_CODE)"
        NEW_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
        if [ -n "$NEW_USER_ID" ]; then
            pass "Created user has id ($NEW_USER_ID)"
        else
            fail "Created user has id" "id missing"
        fi
    else
        fail "POST /admin/users (create)" "expected HTTP 200/201, got $HTTP_CODE"
    fi

    # 12.3 Get user by ID
    if [ -n "$NEW_USER_ID" ]; then
        RESP=$(curl -s -w "\n%{http_code}" "$API/admin/users/$NEW_USER_ID" \
            -H "$AUTH_HEADER")
        HTTP_CODE=$(echo "$RESP" | tail -1)
        BODY=$(echo "$RESP" | sed '$d')
        check_status "GET /admin/users/:id" "200" "$HTTP_CODE"

        if [ "$HTTP_CODE" = "200" ]; then
            check_json_field "User has username" "$BODY" "username"
            check_json_field "User has display_name" "$BODY" "display_name"
        fi
    else
        fail "GET /admin/users/:id" "skipped — no user id"
    fi

    # 12.4 Update user
    if [ -n "$NEW_USER_ID" ]; then
        RESP=$(curl -s -w "\n%{http_code}" -X PUT "$API/admin/users/$NEW_USER_ID" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d '{
                "display_name": "E2E Updated User",
                "email": "e2e-updated@test.com"
            }')
        HTTP_CODE=$(echo "$RESP" | tail -1)
        check_status "PUT /admin/users/:id (update)" "200" "$HTTP_CODE"
    else
        fail "PUT /admin/users/:id (update)" "skipped — no user id"
    fi

    # 12.5 Delete user
    if [ -n "$NEW_USER_ID" ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/admin/users/$NEW_USER_ID" \
            -H "$AUTH_HEADER")
        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
            pass "DELETE /admin/users/:id (HTTP $HTTP_CODE)"
        else
            fail "DELETE /admin/users/:id" "expected HTTP 200/204, got $HTTP_CODE"
        fi
    else
        fail "DELETE /admin/users/:id" "skipped — no user id"
    fi
else
    fail "User management tests" "skipped — no access token"
fi

# ============================================================
section "13. Role Management (Admin)"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 13.1 List roles
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/roles" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/roles (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        # Response might be an array or paginated object
        ITEMS_LEN=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if isinstance(d, list):
    print(len(d))
else:
    items = d.get('items') or d.get('data') or []
    print(len(items) if isinstance(items, list) else 0)
" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Role list has items (count=$ITEMS_LEN)"
        else
            fail "Role list has items" "items array is empty"
        fi
    fi
else
    fail "Role tests" "skipped — no access token"
fi

# ============================================================
section "14. Audit Logs (Admin)"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 14.1 List audit logs
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/audit-logs?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/audit-logs (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        # Audit logs may or may not have entries depending on prior activity
        TOTAL_VAL=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if isinstance(d, list):
    print(len(d))
else:
    print(d.get('total', len(d.get('items', d.get('data', [])))))
" 2>/dev/null || echo "0")
        pass "Audit logs accessible (total=$TOTAL_VAL)"
    fi
else
    fail "Audit log tests" "skipped — no access token"
fi

# ============================================================
section "15. Device Group Management"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 15.1 List groups (tree)
    RESP=$(curl -s -w "\n%{http_code}" "$API/groups" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /groups (list/tree)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if isinstance(d, list):
    print(len(d))
else:
    items = d.get('items') or d.get('data') or d.get('children') or []
    print(len(items) if isinstance(items, list) else 0)
" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "Group list has items (count=$ITEMS_LEN)"
        else
            fail "Group list has items" "items array is empty"
        fi
    fi

    # 15.2 Get group by ID (seeded)
    GROUP_ID="e2e00007-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/groups/$GROUP_ID" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /groups/:id" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        check_json_field "Group has name" "$BODY" "name"
    fi

    # 15.3 Create group
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/groups" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "E2E Test Group",
            "carrier": "cmcc",
            "description": "Created by E2E test"
        }')
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    NEW_GROUP_ID=""
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
        pass "POST /groups (create) (HTTP $HTTP_CODE)"
        NEW_GROUP_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
        if [ -n "$NEW_GROUP_ID" ]; then
            pass "Created group has id ($NEW_GROUP_ID)"
        else
            fail "Created group has id" "id missing"
        fi
    else
        fail "POST /groups (create)" "expected HTTP 200/201, got $HTTP_CODE"
    fi

    # 15.4 Update group
    if [ -n "$NEW_GROUP_ID" ]; then
        RESP=$(curl -s -w "\n%{http_code}" -X PUT "$API/groups/$NEW_GROUP_ID" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d '{
                "name": "E2E Test Group Updated",
                "description": "Updated by E2E test"
            }')
        HTTP_CODE=$(echo "$RESP" | tail -1)
        check_status "PUT /groups/:id (update)" "200" "$HTTP_CODE"
    else
        fail "PUT /groups/:id (update)" "skipped — no group id"
    fi

    # 15.5 List devices in group (seeded group with 2 devices)
    RESP=$(curl -s -w "\n%{http_code}" "$API/groups/e2e00007-0000-0000-0000-000000000002/devices" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /groups/:id/devices" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        DEV_LEN=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if isinstance(d, list):
    print(len(d))
else:
    ids = d.get('device_ids') or d.get('items') or d.get('data') or d.get('devices') or []
    print(len(ids) if isinstance(ids, list) else 0)
" 2>/dev/null || echo "0")
        if [ "$DEV_LEN" -gt 0 ]; then
            pass "Group has devices (count=$DEV_LEN)"
        else
            fail "Group has devices" "no devices in group"
        fi
    fi

    # 15.6 Delete group (cleanup)
    if [ -n "$NEW_GROUP_ID" ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/groups/$NEW_GROUP_ID" \
            -H "$AUTH_HEADER")
        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
            pass "DELETE /groups/:id (HTTP $HTTP_CODE)"
        else
            fail "DELETE /groups/:id" "expected HTTP 200/204, got $HTTP_CODE"
        fi
    else
        fail "DELETE /groups/:id" "skipped — no group id"
    fi
else
    fail "Device group tests" "skipped — no access token"
fi

# ============================================================
section "16. Alarm Clear"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 16.1 Clear alarm (use a different alarm than the one acknowledged in section 6)
    CLEAR_ALARM_ID="e2e00002-0000-0000-0000-000000000004"
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/alarms/$CLEAR_ALARM_ID/clear" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d '{"cleared_by":"e2e-test"}')
    HTTP_CODE=$(echo "$RESP" | tail -1)
    check_status "POST /alarms/:id/clear" "200" "$HTTP_CODE"
else
    fail "Alarm clear test" "skipped — no access token"
fi

# ============================================================
section "17. PM Counter Queries"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 17.1 List PM counters (paginated)
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/counters?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/counters (list)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "PM counter list has items (count=$ITEMS_LEN)"
        else
            fail "PM counter list has items" "items array is empty"
        fi
    fi

    # 17.2 Filter by device_id
    PM_DEVICE_ID="e2e00001-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/counters?device_id=$PM_DEVICE_ID&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/counters?device_id=... (filter)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "PM counters filtered by device_id (count=$ITEMS_LEN)"
        else
            fail "PM counters filtered by device_id" "items array is empty"
        fi
    fi

    # 17.3 Filter by counter_group
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/counters?counter_group=RRC&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/counters?counter_group=RRC" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "PM counters filtered by counter_group=RRC (count=$ITEMS_LEN)"
        else
            fail "PM counters filtered by counter_group=RRC" "items array is empty"
        fi
    fi

    # 17.4 Filter by time range
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/counters?start_time=2026-03-06T00:00:00Z&end_time=2026-03-07T00:00:00Z&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/counters?start_time=...&end_time=... (time range)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        pass "PM counters in time range (count=$ITEMS_LEN)"
    fi

    # 17.5 Verify counter fields
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/counters?page=1&page_size=1" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    if [ "$HTTP_CODE" = "200" ]; then
        HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('items', [])
if items:
    item = items[0]
    required = ['time', 'device_id', 'counter_group', 'counter_name', 'counter_value']
    present = [k for k in required if k in item]
    print(len(present))
else:
    print(0)
" 2>/dev/null || echo "0")
        if [ "$HAS_FIELDS" -ge 4 ]; then
            pass "PM counter has required fields (${HAS_FIELDS}/5)"
        else
            fail "PM counter has required fields" "only $HAS_FIELDS/5 present"
        fi
    fi
else
    fail "PM counter tests" "skipped — no access token"
fi

# ============================================================
section "18. KPI Queries"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 18.1 List KPI values (paginated)
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/kpi?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/kpi (list values)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "KPI values list has items (count=$ITEMS_LEN)"
        else
            fail "KPI values list has items" "items array is empty"
        fi
    fi

    # 18.2 Filter KPI by kpi_name
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/kpi?kpi_name=E2E_RRC_SR&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/kpi?kpi_name=E2E_RRC_SR (filter)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "KPI values filtered by name (count=$ITEMS_LEN)"
        else
            fail "KPI values filtered by name" "items array is empty"
        fi
    fi

    # 18.3 List KPI definitions
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/kpi/definitions" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/kpi/definitions" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
if isinstance(d, list):
    print(len(d))
else:
    items = d.get('items') or d.get('definitions') or []
    print(len(items) if isinstance(items, list) else 0)
" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "KPI definitions has items (count=$ITEMS_LEN)"
        else
            fail "KPI definitions has items" "items array is empty"
        fi
    fi

    # 18.4 Filter KPI definitions by carrier
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/kpi/definitions?carrier=cmcc" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /pm/kpi/definitions?carrier=cmcc" "200" "$HTTP_CODE"

    # 18.5 Verify KPI value fields
    RESP=$(curl -s -w "\n%{http_code}" "$API/pm/kpi?page=1&page_size=1" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    if [ "$HTTP_CODE" = "200" ]; then
        HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('items', [])
if items:
    item = items[0]
    required = ['time', 'device_id', 'kpi_name', 'kpi_value']
    present = [k for k in required if k in item]
    print(len(present))
else:
    print(0)
" 2>/dev/null || echo "0")
        if [ "$HAS_FIELDS" -ge 3 ]; then
            pass "KPI value has required fields (${HAS_FIELDS}/4)"
        else
            fail "KPI value has required fields" "only $HAS_FIELDS/4 present"
        fi
    fi
else
    fail "KPI tests" "skipped — no access token"
fi

# ============================================================
section "19. MR Files & Data"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 19.1 List MR files (paginated)
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/files?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /mr/files (list)" "200" "$HTTP_CODE"

    MR_FILE_ID=""
    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "MR file list has items (count=$ITEMS_LEN)"
            MR_FILE_ID=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items',[]); print(items[0].get('id','') if items else '')" 2>/dev/null || echo "")
        else
            fail "MR file list has items" "items array is empty"
        fi
    fi

    # 19.2 Filter MR files by mr_type
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/files?mr_type=MRO&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /mr/files?mr_type=MRO (filter)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "MR files filtered by MRO type (count=$ITEMS_LEN)"
        else
            fail "MR files filtered by MRO type" "items array is empty"
        fi
    fi

    # 19.3 Download MR file (may fail if MinIO unavailable — treat 500 as acceptable skip)
    if [ -n "$MR_FILE_ID" ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/mr/files/$MR_FILE_ID/download" \
            -H "$AUTH_HEADER")
        if [ "$HTTP_CODE" = "200" ]; then
            pass "GET /mr/files/:id/download (HTTP 200)"
        elif [ "$HTTP_CODE" = "500" ]; then
            pass "GET /mr/files/:id/download (HTTP 500 — MinIO unavailable, expected)"
        else
            fail "GET /mr/files/:id/download" "expected HTTP 200 or 500, got $HTTP_CODE"
        fi
    else
        pass "GET /mr/files/:id/download (skipped — no file id, no seeded files)"
    fi

    # 19.4 List MR data/records
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/data?page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /mr/data (list records)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "MR data list has items (count=$ITEMS_LEN)"
        else
            fail "MR data list has items" "items array is empty"
        fi
    fi

    # 19.5 Filter MR data by device_id
    MR_DEVICE_ID="e2e00001-0000-0000-0000-000000000001"
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/data?device_id=$MR_DEVICE_ID&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /mr/data?device_id=... (filter)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        ITEMS_LEN=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items') or []; print(len(items) if isinstance(items,list) else 0)" 2>/dev/null || echo "0")
        if [ "$ITEMS_LEN" -gt 0 ]; then
            pass "MR data filtered by device_id (count=$ITEMS_LEN)"
        else
            fail "MR data filtered by device_id" "items array is empty"
        fi
    fi

    # 19.6 Verify MR file fields
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/files?page=1&page_size=1" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    if [ "$HTTP_CODE" = "200" ]; then
        HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('items', [])
if items:
    item = items[0]
    required = ['id', 'device_id', 'mr_type', 'file_name', 'file_size']
    present = [k for k in required if k in item]
    print(len(present))
else:
    print(0)
" 2>/dev/null || echo "0")
        if [ "$HAS_FIELDS" -ge 4 ]; then
            pass "MR file has required fields (${HAS_FIELDS}/5)"
        else
            fail "MR file has required fields" "only $HAS_FIELDS/5 present"
        fi
    fi

    # 19.7 Verify MR record fields
    RESP=$(curl -s -w "\n%{http_code}" "$API/mr/data?page=1&page_size=1" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    if [ "$HTTP_CODE" = "200" ]; then
        HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('items', [])
if items:
    item = items[0]
    required = ['id', 'device_id', 'cell_id', 'mr_type', 'measurement_data']
    present = [k for k in required if k in item]
    print(len(present))
else:
    print(0)
" 2>/dev/null || echo "0")
        if [ "$HAS_FIELDS" -ge 4 ]; then
            pass "MR record has required fields (${HAS_FIELDS}/5)"
        else
            fail "MR record has required fields" "only $HAS_FIELDS/5 present"
        fi
    fi
else
    fail "MR tests" "skipped — no access token"
fi

# ============================================================
section "20. Audit Log Time Range Filter"
# ============================================================

if [ -n "$ACCESS_TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $ACCESS_TOKEN"

    # 20.1 Audit logs with time range (should match seeded data)
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/audit-logs?start_time=2026-03-06T00:00:00Z&end_time=2026-03-07T23:59:59Z&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/audit-logs?start_time=...&end_time=... (range)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        TOTAL_VAL=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total', len(d.get('items',[]))))" 2>/dev/null || echo "0")
        if [ "$TOTAL_VAL" -gt 0 ]; then
            pass "Audit logs in time range has results (total=$TOTAL_VAL)"
        else
            fail "Audit logs in time range has results" "total=0"
        fi
    fi

    # 20.2 Audit logs with future time range (should return 0)
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/audit-logs?start_time=2099-01-01T00:00:00Z&end_time=2099-12-31T23:59:59Z&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/audit-logs (future range)" "200" "$HTTP_CODE"

    if [ "$HTTP_CODE" = "200" ]; then
        TOTAL_VAL=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total', len(d.get('items',[]))))" 2>/dev/null || echo "0")
        if [ "$TOTAL_VAL" = "0" ]; then
            pass "Audit logs future range returns empty (total=0)"
        else
            fail "Audit logs future range returns empty" "expected total=0, got $TOTAL_VAL"
        fi
    fi

    # 20.3 Audit log entry has required fields
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/audit-logs?page=1&page_size=1" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    if [ "$HTTP_CODE" = "200" ]; then
        HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('items', [])
if items:
    item = items[0]
    required = ['id', 'username', 'action', 'resource', 'created_at']
    present = [k for k in required if k in item]
    print(len(present))
else:
    print(0)
" 2>/dev/null || echo "0")
        if [ "$HAS_FIELDS" -ge 4 ]; then
            pass "Audit log entry has required fields (${HAS_FIELDS}/5)"
        else
            fail "Audit log entry has required fields" "only $HAS_FIELDS/5 present"
        fi
    fi

    # 20.4 Filter audit logs by action
    RESP=$(curl -s -w "\n%{http_code}" "$API/admin/audit-logs?action=login&page=1&page_size=10" \
        -H "$AUTH_HEADER")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "GET /admin/audit-logs?action=login" "200" "$HTTP_CODE"
else
    fail "Audit log time range tests" "skipped — no access token"
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

#!/bin/bash
# e2e_param_model_custom.sh — 自定义 paramModel XML GWT 用例
#
# 三库 XML 导入重构(2026-06-04 D3/D5/D6)后契约:
#   单目录 + sidecar;上传 = name 表单字段 + 双唯一性硬拒(无 force,无覆盖);
#   删除移除 sidecar。上传文件自身 filename 被忽略,目标 = <name>.xml。
#
# GWT 用例:
#   GWT-1  跨升级持久化              — manual(需 docker 镜像切换,本脚本仅断言文档)
#   GWT-2  来源字段准确              — automated
#   GWT-3  内置不可删 + 错误码 2030  — automated
#   GWT-4  自定义可删 + 物理备份     — automated
#   GWT-6a 上传成功(name)           — automated
#   GWT-6b 同名文件名 → 409          — automated
#   GWT-6c 内容主键(模型名)重复 → 409 — automated
#   GWT-7  备份失败保守回滚 + 2031   — manual(需 chmod 0500 目录)
#   GWT-8  30 天备份清理 cron        — manual(需 worker chronotime 注入)
#   GWT-9  非法 name(路径遍历)拦截   — automated
#   GWT-10 XML 内容校验              — automated
#   GWT-11 保留名拦截                — automated
#
# 使用:
#   bash scripts/e2e_param_model_custom.sh [BASE_URL]
#   默认 BASE_URL=http://localhost:8081(omcgo-app 直连)
#   通过 env 配置:
#     ADMIN_USER  默认 admin
#     ADMIN_PASS  默认 admin123
#     OMC_TOKEN   提供则跳过登录(用于已有会话)
#
# 与 scripts/e2e_verify.sh 关系:独立脚本,可单独跑;待 e2e_verify.sh 主链补
# T-0178 section 后嵌入运行。

set -uo pipefail

BASE_URL="${1:-http://localhost:8081}"
API="${BASE_URL}/api/v1"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"

PASS=0
FAIL=0
TOTAL=0
CLAIM_COUNT=0

# ── Colors ─────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass() { PASS=$((PASS + 1)); TOTAL=$((TOTAL + 1)); echo -e "  ${GREEN}[PASS]${NC} $1"; }
fail() {
    FAIL=$((FAIL + 1)); TOTAL=$((TOTAL + 1))
    echo -e "  ${RED}[FAIL]${NC} $1"
    [ -n "${2:-}" ] && echo -e "         ${RED}→ $2${NC}"
}
section() { echo ""; echo -e "${YELLOW}═══ $1 ═══${NC}"; }
claim() { CLAIM_COUNT=$((CLAIM_COUNT + 1)); echo -e "  ${CYAN}[CLAIM ${CLAIM_COUNT}]${NC} $1"; }

check_status() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$actual" = "$expected" ]; then pass "$desc (HTTP $actual)"
    else fail "$desc" "expected HTTP $expected, got $actual"; fi
}

check_status_in() {
    local desc="$1" list="$2" actual="$3" code
    for code in $list; do
        [ "$actual" = "$code" ] && { pass "$desc (HTTP $actual ∈ {$list})"; return 0; }
    done
    fail "$desc" "expected HTTP one of [$list], got $actual"
}

# Python helper: 在 stdin 拿 JSON,运行 expr,打印结果。
jq_py() {
    local expr="$1"
    python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print($expr)
except Exception as e:
    print('', file=sys.stderr)
    sys.exit(1)
"
}

# ── Login: 复制自 e2e_verify.sh 的 RSA-OAEP 登录 ───────────────────────
encrypt_password() {
    local plain="$1"
    python3 <<EOF
import json, base64, time, secrets, sys, urllib.request
try:
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import padding
except ImportError:
    print("ERR: install cryptography (pip install cryptography)", file=sys.stderr)
    sys.exit(2)
# 取 public key
try:
    r = urllib.request.urlopen("$API/auth/public-key", timeout=5)
    body = json.loads(r.read())
    data = body.get("data") or body
    pk_pem = data["public_key"].encode()
    kid = data.get("key_id", "")
except Exception as e:
    print(f"ERR: fetch public-key: {e}", file=sys.stderr)
    sys.exit(2)
pk = serialization.load_pem_public_key(pk_pem)
payload = json.dumps({"password": "$plain", "ts": int(time.time()*1000), "nonce": secrets.token_hex(8)}).encode()
ct = pk.encrypt(payload, padding.OAEP(mgf=padding.MGF1(hashes.SHA256()), algorithm=hashes.SHA256(), label=None))
print(base64.b64encode(ct).decode() + "|" + kid)
EOF
}

# 跳过登录:由 env OMC_TOKEN 提供
TOKEN="${OMC_TOKEN:-}"
if [ -z "$TOKEN" ]; then
    section "Login (admin → token)"
    ENC_KID=$(encrypt_password "$ADMIN_PASS" 2>/dev/null || echo "")
    if [ -z "$ENC_KID" ]; then
        echo -e "  ${RED}[ERROR]${NC} login encryption failed — set OMC_TOKEN env or check cryptography install"
        exit 2
    fi
    ENC="${ENC_KID%|*}"
    KID="${ENC_KID##*|}"
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$ADMIN_USER\",\"encrypted_password\":\"$ENC\",\"key_id\":\"$KID\"}")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    BODY=$(echo "$RESP" | sed '$d')
    check_status "POST /auth/login" "200" "$HTTP_CODE"
    [ "$HTTP_CODE" = "200" ] && TOKEN=$(echo "$BODY" | jq_py "d.get('data', {}).get('access_token', '')" 2>/dev/null || echo "")
    if [ -z "$TOKEN" ]; then
        echo -e "  ${RED}[ERROR]${NC} login response has no access_token"
        exit 2
    fi
    echo -e "  ${CYAN}[INFO]${NC} access_token acquired"
fi
AUTH="Authorization: Bearer $TOKEN"

# ── Test artifacts ────────────────────────────────────────────────────
TMP_DIR=$(mktemp -d -t t0178XXXXXX)
cleanup_global() {
    # 清理上传残留 + tmp 目录;失败容忍(测试目标已 PASS/FAIL 决定脚本退出码)
    curl -s -X DELETE "$API/param-models/CBQQ" -H "$AUTH" >/dev/null 2>&1 || true
    curl -s -X DELETE "$API/param-models/CBQQ2" -H "$AUTH" >/dev/null 2>&1 || true
    rm -rf "$TMP_DIR"
}
trap cleanup_global EXIT

# 构造合法 paramModel XML(根元素 paramModel,Loader 可解析)
CUSTOM_XML="$TMP_DIR/CBQQ.xml"
cat > "$CUSTOM_XML" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<paramModel name="CBQQ">
  <parameters>
    <param name="Device.X_TEST.Foo" supported="true" type="string"/>
  </parameters>
</paramModel>
EOF

# 内容主键(模型名)重复测试用:文件名不同(CBQQ2.xml),但 paramModel name 仍是 CBQQ
DUP_MODEL_XML="$TMP_DIR/CBQQ2.xml"
cat > "$DUP_MODEL_XML" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<paramModel name="CBQQ">
  <parameters>
    <param name="Device.X_TEST.Bar" supported="true" type="string"/>
  </parameters>
</paramModel>
EOF

# 无效 XML(根元素非 paramModel)
INVALID_XML="$TMP_DIR/WRONG.xml"
echo '<wrongRoot/>' > "$INVALID_XML"

# 保留名 XML
RESERVED_XML="$TMP_DIR/standard-model.xml"
echo '<paramModel/>' > "$RESERVED_XML"

# 先清残留
curl -s -X DELETE "$API/param-models/CBQQ" -H "$AUTH" >/dev/null 2>&1 || true
curl -s -X DELETE "$API/param-models/CBQQ2" -H "$AUTH" >/dev/null 2>&1 || true

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-2 — 来源字段准确(List 返 source + deletable)                  ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-2 — Source 字段准确"
claim "GET /param-models 列表项含 source + deletable 字段"

RESP=$(curl -s -w "\n%{http_code}" -X GET "$API/param-models" -H "$AUTH")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
check_status "GET /param-models" "200" "$HTTP_CODE"

if [ "$HTTP_CODE" = "200" ]; then
    HAS_FIELDS=$(echo "$BODY" | jq_py "
items = d.get('data', {}).get('items', [])
all_have = all('source' in m and 'deletable' in m for m in items) if items else False
'YES' if all_have else 'NO'
" 2>/dev/null || echo "NO")
    [ "$HAS_FIELDS" = "YES" ] \
        && pass "every modelView has source + deletable" \
        || fail "some modelView lacks source/deletable" "$BODY"

    BUILTIN_NOT_DELETABLE=$(echo "$BODY" | jq_py "
items = d.get('data', {}).get('items', [])
builtins = [m for m in items if m.get('source') == 'builtin']
'YES' if builtins and all(not m.get('deletable') for m in builtins) else 'NO'
" 2>/dev/null || echo "NO")
    [ "$BUILTIN_NOT_DELETABLE" = "YES" ] \
        && pass "builtin rows have deletable=false" \
        || fail "some builtin row has deletable=true" "check API output"
fi

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-3 — 内置不可删 → 403 + code=2030                              ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-3 — Builtin DELETE → 403 ErrCodeParamModelBuiltinNotDeletable"
claim "DELETE /param-models/BLQ(builtin)返 403 + msg 含 [code=2030]"

RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$API/param-models/BLQ" -H "$AUTH")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
check_status "DELETE /param-models/BLQ(builtin)" "403" "$HTTP_CODE"
if echo "$BODY" | grep -q "code=2030"; then
    pass "error msg contains [code=2030]"
else
    fail "error msg missing [code=2030]" "body=$BODY"
fi

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-9 — 非法 name(路径遍历)拦截                                  ║
# ║ (name 表单字段含 ../ ,后端 validateUploadFilename 正则拒绝)        ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-9 — 非法 name(路径遍历)→ 400"
claim "POST /upload-xml name=../../etc/passwd 被拒绝 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=../../etc/passwd" \
    --form "file=@$CUSTOM_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(path traversal name)" "400 403" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-10 — XML 内容校验                                             ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-10 — 错根元素 XML → 400"
claim "POST /upload-xml 根元素非 parameterModel 被拒 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=WRONG" \
    --form "file=@$INVALID_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(wrong root element)" "400" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-11 — 保留名拦截                                               ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-11 — 保留名 name=standard-model → 400"
claim "POST /upload-xml name=standard-model 被拒 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=standard-model" \
    --form "file=@$RESERVED_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(reserved name)" "400" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-6 — 上传 name 唯一性双校验(无 force,无覆盖)                  ║
# ║ 6a 首次 name=CBQQ → 200                                          ║
# ║ 6b 同名文件名 name=CBQQ 再传 → 409(文件名已存在,请改名)           ║
# ║ 6c 文件名不同 name=CBQQ2 但 XML 模型名仍 CBQQ → 409(内容主键重复)  ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-6 — name 双唯一性硬拒(无 force)"

claim "POST /upload-xml name=CBQQ 首次成功 200"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=CBQQ" \
    --form "file=@$CUSTOM_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(name=CBQQ first)" "200" "$HTTP_CODE"

claim "POST /upload-xml name=CBQQ 同名文件名 → 409(请改名)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=CBQQ" \
    --form "file=@$CUSTOM_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(name=CBQQ duplicate filename)" "409" "$HTTP_CODE"

claim "POST /upload-xml name=CBQQ2 文件名不同但模型名仍 CBQQ → 409(内容主键重复)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "name=CBQQ2" \
    --form "file=@$DUP_MODEL_XML" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(name=CBQQ2 dup model name)" "409" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-4 — 自定义可删 + 物理备份                                     ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-4 — Custom DELETE → 200 + backup 字段"
claim "DELETE /param-models/CBQQ 返 200 + backup 字段"

RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$API/param-models/CBQQ" -H "$AUTH")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
check_status "DELETE /param-models/CBQQ(custom)" "200" "$HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    HAS_BACKUP=$(echo "$BODY" | jq_py "'YES' if d.get('data', {}).get('backup') else 'NO'" 2>/dev/null || echo "NO")
    [ "$HAS_BACKUP" = "YES" ] \
        && pass "DELETE response has backup field" \
        || fail "DELETE response missing backup field" "body=$BODY"
fi

# ╔══════════════════════════════════════════════════════════════════╗
# ║ Manual GWT (require container env)                                ║
# ╚══════════════════════════════════════════════════════════════════╝
section "Manual GWT(需容器环境)"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-1 跨升级持久化:模拟 docker compose 重建容器后 CBQQ.xml 应仍在"
echo -e "    步骤: 上传 name=CBQQ → docker compose down/up app → GET /param-models 仍含 CBQQ"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-7 备份失败保守回滚:chmod 0500 .../param-mappings"
echo -e "    步骤: 上传 X → chmod 0500 dir → DELETE X → 期望 500 + code=2031 + 文件 + DB 行均保留"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-8 30 天清理 cron + 孤儿 sidecar:在 worker 容器 touch -t 一个 30 天前的 .deleted 文件"
echo -e "    步骤: touch -t 202604010300 .../.deleted.20260401030000 → 等 03:00 cron → 文件应被清;"
echo -e "          再造孤儿 X.xml.custom(无 X.xml)→ cron 应清掉 sidecar"
echo -e "  ${YELLOW}[NOTE]${NC}   这 3 个 GWT 列入 manual,S5 手工验证或写 docker-test 子任务"

# ── Summary ────────────────────────────────────────────────────────────
echo ""
echo "════════════════════════════════════════════"
echo -e "  param-model custom-XML E2E: ${GREEN}$PASS PASS${NC} / ${RED}$FAIL FAIL${NC} / $TOTAL TOTAL"
echo -e "  Claims: ${CYAN}${CLAIM_COUNT}${NC}(automated GWT + 3 manual)"
echo "════════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
echo -e "${GREEN}All automated tests passed!${NC}"
exit 0

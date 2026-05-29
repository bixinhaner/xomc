#!/bin/bash
# e2e_param_model_custom.sh — T-0178 自定义 paramModel XML 11 GWT 用例
#
# 覆盖 PRD docs/project/prd/F02-param-model-custom-xml.md §3 的 11 个 GWT:
#   GWT-1  跨升级持久化              — manual(需 docker 镜像切换,本脚本仅断言文档)
#   GWT-2  来源字段准确              — automated
#   GWT-3  内置不可删 + 错误码 2030  — automated
#   GWT-4  自定义可删 + 物理备份     — automated
#   GWT-5  Self-healing 同名回退     — automated
#   GWT-6  同名上传 force=true       — automated
#   GWT-7  备份失败保守回滚 + 2031   — manual(需 chmod 0500 customDir)
#   GWT-8  30 天备份清理 cron        — manual(需 worker chronotime 注入)
#   GWT-9  路径遍历拦截              — automated
#   GWT-10 XML 内容校验              — automated
#   GWT-11 同名占位文件拦截          — automated
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
    # GWT-5 中断残留:若 BLQ 卡在 custom override 状态,删 + reload 恢复 builtin
    BLQ_SRC=$(curl -s -X GET "$API/param-models/BLQ" -H "$AUTH" 2>/dev/null \
        | python3 -c "import sys, json; print(json.load(sys.stdin).get('data', {}).get('source', ''))" 2>/dev/null \
        || echo "")
    if [ "$BLQ_SRC" = "custom" ]; then
        curl -s -X DELETE "$API/param-models/BLQ" -H "$AUTH" >/dev/null 2>&1 || true
        curl -s -X POST "$API/param-models/import-directory?mode=reload" -H "$AUTH" >/dev/null 2>&1 || true
    fi
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

# 同名覆盖测试用(故意覆盖既有 builtin BLQ — 测 self-healing)
BLQ_OVERRIDE="$TMP_DIR/BLQ.xml"
cat > "$BLQ_OVERRIDE" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<paramModel name="BLQ">
  <parameters>
    <param name="Device.X_OVERRIDE.Foo" supported="true" type="string"/>
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
# ║ GWT-9 — 路径遍历拦截                                              ║
# ║ (multipart 上传 Filename 字段含 ../ ,后端用 filepath.Base + 正则) ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-9 — 路径遍历文件名 → 400"
claim "POST /upload-xml ../../etc/passwd 文件名被拒绝 400"

# 用 curl --form 'file=@local;filename=../../etc/passwd' 显式造非法 filename
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "file=@$CUSTOM_XML;filename=../../etc/passwd" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(path traversal filename)" "400 403" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-10 — XML 内容校验                                             ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-10 — 错根元素 XML → 400"
claim "POST /upload-xml 根元素非 paramModel 被拒 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "file=@$INVALID_XML;filename=WRONG.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(wrong root element)" "400" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-11 — 保留名拦截                                               ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-11 — 保留名 standard-model.xml → 400"
claim "POST /upload-xml standard-model.xml 被拒 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "file=@$RESERVED_XML;filename=standard-model.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status_in "POST /upload-xml(reserved filename)" "400" "$HTTP_CODE"

# ╔══════════════════════════════════════════════════════════════════╗
# ║ GWT-6 — 同名上传 force=true 链路                                  ║
# ║ (先上 CBQQ → 200; 再上 CBQQ 无 force → 409; 加 ?force=true → 200) ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-6 — 同名 Upload 409 → force=true → 200 + backup"

claim "POST /upload-xml CBQQ.xml 首次成功 200"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "file=@$CUSTOM_XML;filename=CBQQ.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(CBQQ.xml first)" "200" "$HTTP_CODE"

claim "POST /upload-xml CBQQ.xml 同名无 force → 409"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml" \
    -H "$AUTH" \
    --form "file=@$CUSTOM_XML;filename=CBQQ.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(CBQQ.xml same-name, no force)" "409" "$HTTP_CODE"

claim "POST /upload-xml?force=true 强制覆盖返 200 + backup 字段"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml?force=true" \
    -H "$AUTH" \
    --form "file=@$CUSTOM_XML;filename=CBQQ.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
check_status "POST /upload-xml?force=true(CBQQ.xml overwrite)" "200" "$HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    HAS_BACKUP=$(echo "$BODY" | jq_py "'YES' if d.get('data', {}).get('backup') else 'NO'" 2>/dev/null || echo "NO")
    [ "$HAS_BACKUP" = "YES" ] \
        && pass "response has backup field" \
        || fail "response missing backup field" "body=$BODY"
fi

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
# ║ GWT-5 — Self-healing 同名覆盖回退                                 ║
# ║ 1. 上传 BLQ.xml 覆盖内置 → source 应变 custom                     ║
# ║ 2. 删除 BLQ → 触发 Reload → source 应回退 builtin(自愈)           ║
# ║                                                                  ║
# ║ 注:此场景**修改内置 paramModel 状态**,需要测试环境隔离;          ║
# ║ 失败容忍设计 — 仅校验 API 返码,不强校验 DB 内部状态。            ║
# ╚══════════════════════════════════════════════════════════════════╝
section "GWT-5 — Self-healing 同名覆盖 → 删 custom → 自愈回 builtin"
claim "上传 BLQ.xml(custom 覆盖 builtin)→ source=custom"
claim "删除 BLQ(custom)→ Reload → source 回 builtin"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$API/param-models/upload-xml?force=true" \
    -H "$AUTH" \
    --form "file=@$BLQ_OVERRIDE;filename=BLQ.xml" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status "POST /upload-xml(BLQ.xml override)" "200" "$HTTP_CODE"

if [ "$HTTP_CODE" = "200" ]; then
    sleep 1 # 让 Reload 完成
    RESP=$(curl -s -X GET "$API/param-models/BLQ" -H "$AUTH")
    SRC=$(echo "$RESP" | jq_py "d.get('data', {}).get('source', '')" 2>/dev/null || echo "")
    [ "$SRC" = "custom" ] \
        && pass "GET /param-models/BLQ 显 source=custom 覆盖生效" \
        || fail "BLQ source != custom after override" "got source=$SRC"

    # 删 custom → 期望 200(因 source=custom 可删)
    RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$API/param-models/BLQ" -H "$AUTH")
    HTTP_CODE=$(echo "$RESP" | tail -1)
    check_status "DELETE /param-models/BLQ(custom override)" "200" "$HTTP_CODE"

    # 触发 reload 让 Loader 从 builtin 重新载入
    curl -s -X POST "$API/param-models/import-directory?mode=reload" -H "$AUTH" >/dev/null
    sleep 1

    RESP=$(curl -s -X GET "$API/param-models/BLQ" -H "$AUTH")
    SRC=$(echo "$RESP" | jq_py "d.get('data', {}).get('source', '')" 2>/dev/null || echo "")
    [ "$SRC" = "builtin" ] \
        && pass "self-healing: BLQ source 回 builtin" \
        || fail "self-healing failed: BLQ source=$SRC after delete+reload"
fi

# ╔══════════════════════════════════════════════════════════════════╗
# ║ Manual GWT (require container env)                                ║
# ╚══════════════════════════════════════════════════════════════════╝
section "Manual GWT(需容器环境)"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-1 升级跨升级持久化:模拟 docker compose 重建容器后 CBQQ.xml 应仍在"
echo -e "    步骤: 上传 CBQQ → docker compose down/up app → GET /param-models 仍含 CBQQ"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-7 备份失败保守回滚:chmod 0500 /opt/omc/data/param-mappings-custom"
echo -e "    步骤: 上传 X → chmod 0500 dir → DELETE X → 期望 500 + code=2031 + 文件 + DB 行均保留"
echo -e "  ${CYAN}[MANUAL]${NC} GWT-8 30 天清理 cron:在 worker 容器 touch -t 一个 30 天前的 .deleted 文件"
echo -e "    步骤: touch -t 202604010300 .../.deleted.20260401030000 → 等 03:00 cron → 文件应被清"
echo -e "  ${YELLOW}[NOTE]${NC}   这 3 个 GWT 列入 P5 manual,S5 手工验证或写 docker-test 子任务"

# ── Summary ────────────────────────────────────────────────────────────
echo ""
echo "════════════════════════════════════════════"
echo -e "  T-0178 P5 E2E: ${GREEN}$PASS PASS${NC} / ${RED}$FAIL FAIL${NC} / $TOTAL TOTAL"
echo -e "  Claims: ${CYAN}${CLAIM_COUNT}${NC}(8 automated GWT + 3 manual)"
echo "════════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
echo -e "${GREEN}All automated tests passed!${NC}"
exit 0

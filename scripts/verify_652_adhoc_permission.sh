#!/bin/bash
# verify_652_adhoc_permission.sh
# 验证 issue #652：聚合任务权限控制 —— 自建任务非 owner 操作 403 / 超管放行 / 内置任务全员可读
#
# 流程：
#   1. admin 登录拿 token
#   2. pg 直插一个普通用户 alice（operator 角色，密码同 admin），登录拿 token
#   3. admin 创建自建任务 → task_id；记下任务，creator=admin
#   4. 改 creator='alice' 模拟"alice 自建"，让 admin 充当"非 owner"角色
#      —— 但 admin 是超管，仍应放行（场景 A）
#   5. 真正负向：admin 创建 task → 用 alice token 操作 → 期望 403（场景 B）
#   6. 普通用户读内置任务结果 → 200（场景 C）
#
# 期望全部 PASS。

set -uo pipefail

API="${API:-http://localhost:8081/api/v1}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"
ALICE_USER="${ALICE_USER:-alice652}"
ALICE_PASS="${ALICE_PASS:-admin123}"
PG_CONT="${PG_CONT:-omc-docker-postgres-1}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
PASS=0; FAIL=0
ok()   { echo -e "${GREEN}[PASS]${NC} $*"; PASS=$((PASS+1)); }
bad()  { echo -e "${RED}[FAIL]${NC} $*"; FAIL=$((FAIL+1)); }
log()  { echo -e "${CYAN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

# ── helpers ──────────────────────────────────────────────────────
fetch_pubkey() {
    curl -s --max-time 5 "$API/auth/public-key"
}

encrypt_pwd() {
    local pem="$1" plain="$2"
    PEM_INPUT="$pem" PLAIN_INPUT="$plain" python3 <<'PY'
import os, json, time, secrets, base64
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import padding
pub = serialization.load_pem_public_key(os.environ['PEM_INPUT'].encode())
payload = json.dumps({"password": os.environ['PLAIN_INPUT'],
                      "ts": int(time.time()),
                      "nonce": secrets.token_hex(16)}).encode()
ct = pub.encrypt(payload, padding.OAEP(mgf=padding.MGF1(hashes.SHA256()),
                                       algorithm=hashes.SHA256(), label=None))
print(base64.b64encode(ct).decode(), end="")
PY
}

login() {
    local user="$1" pass="$2"
    local pk=$(fetch_pubkey)
    local pem=$(echo "$pk" | python3 -c "import sys,json;d=json.load(sys.stdin);print((d.get('data') or {}).get('public_key',''))")
    local kid=$(echo "$pk" | python3 -c "import sys,json;d=json.load(sys.stdin);print((d.get('data') or {}).get('key_id',''))")
    [ -z "$pem" ] && { echo ""; return 1; }
    local enc=$(encrypt_pwd "$pem" "$pass")
    local resp=$(curl -s -X POST "$API/auth/login" \
        -H 'Content-Type: application/json' \
        -d "{\"username\":\"$user\",\"encrypted_password\":\"$enc\",\"key_id\":\"$kid\"}")
    echo "$resp" | python3 -c "import sys,json
try:
    d=json.load(sys.stdin)
    print((d.get('data') or {}).get('access_token',''))
except: print('')"
}

http_code() {  # http_code METHOD URL BODY_OUT [curl-opts...]
    local method="$1" url="$2" out="$3"; shift 3
    curl -s -o "$out" -w '%{http_code}' -X "$method" "$url" "$@"
}

# ── §0 admin 登录 ────────────────────────────────────────────────
log "§0 admin 登录"
ADMIN_TOKEN=$(login "$ADMIN_USER" "$ADMIN_PASS")
if [ -z "$ADMIN_TOKEN" ]; then
    bad "admin 登录失败，token 为空。中止。"
    exit 1
fi
ok "admin 登录成功 (token len=${#ADMIN_TOKEN})"
ADMIN_AUTH="Authorization: Bearer $ADMIN_TOKEN"

# ── §1 准备普通用户 alice ───────────────────────────────────────
log "§1 直插普通用户 ${ALICE_USER} (临时角色 pm-tester, source='admin' 非 super admin)"
# 用 admin 当前 bcrypt 哈希复制给 alice，确保密码哈希参数一致（避开 bcrypt cost mismatch）
# 同时建一个临时角色 pm-tester（role name 不是 admin/super_admin，避免命中 handler.isAdmin 旁路），
# 给它挂 PM adhoc 全部 10 个端点权限，让 RBAC 中间件放行。
PM_TESTER_ROLE_ID='10000000-0000-0000-0000-000000000652'
docker exec "$PG_CONT" psql -U omcgo -d omcgo -c "
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username='$ALICE_USER');
DELETE FROM users WHERE username='$ALICE_USER';
DELETE FROM role_api_permissions WHERE role_id='$PM_TESTER_ROLE_ID'::uuid;
DELETE FROM roles WHERE id='$PM_TESTER_ROLE_ID'::uuid;

INSERT INTO roles (id, name, description, is_system, created_at, updated_at)
VALUES ('$PM_TESTER_ROLE_ID'::uuid, 'pm-tester-652', 'issue #652 verify role', false, now(), now());

INSERT INTO role_api_permissions (role_id, endpoint_id, created_at)
SELECT '$PM_TESTER_ROLE_ID'::uuid, id, now()
FROM api_endpoints
WHERE path LIKE '/api/v1/pm/adhoc/%';

INSERT INTO users (id, username, password_hash, display_name, status, password_changed_at, failed_login_attempts, created_at, updated_at, source, must_change_password)
SELECT gen_random_uuid(), '$ALICE_USER', password_hash, 'Alice (#652 test)', 'active', now(), 0, now(), now(), 'admin', false
FROM users WHERE username='$ADMIN_USER';

INSERT INTO user_roles (user_id, role_id, is_default, created_at)
SELECT u.id, '$PM_TESTER_ROLE_ID'::uuid, true, now() FROM users u WHERE u.username='$ALICE_USER';

SELECT u.id, u.username, u.source, r.name AS role
FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id
WHERE u.username='$ALICE_USER';
"

log "§1.0 重启 app 让 Casbin policy reload，使 alice 的 role assignment 生效"
( cd /Users/shangyingbin/project/omc-docker && docker compose restart app > /dev/null 2>&1 )
# 等 app 完全启动：以 /auth/public-key 能稳定返回 200 + 含 public_key 字段为准
for i in {1..60}; do
    sleep 1
    R=$(curl -s --max-time 2 "$API/auth/public-key" 2>/dev/null)
    if echo "$R" | grep -q '"public_key"'; then
        ok "app 已就绪 (等待 ${i}s)"
        break
    fi
done

log "§1.1 alice 登录"
ALICE_TOKEN=$(login "$ALICE_USER" "$ALICE_PASS")
if [ -z "$ALICE_TOKEN" ]; then
    bad "alice 登录失败。中止。"
    exit 1
fi
ok "alice 登录成功 (token len=${#ALICE_TOKEN})"
ALICE_AUTH="Authorization: Bearer $ALICE_TOKEN"

# ── §2 admin 创建一个自建任务 ───────────────────────────────────
log "§2 admin 创建自建聚合任务"
# pm/adhoc Create 接口的最小请求体（按 handler.Create 推断）
CREATE_BODY='{
  "name": "issue-652-verify-task",
  "mode": "oneshot",
  "dimension": "network",
  "technology": "nr",
  "metric_paths": ["KGNB0510"],
  "granularities": ["1h"],
  "window_start": "2026-06-24T00:00:00Z",
  "window_end": "2026-06-25T00:00:00Z"
}'
TMP=$(mktemp)
CREATE_CODE=$(http_code POST "$API/pm/adhoc/tasks" "$TMP" \
    -H 'Content-Type: application/json' -H "$ADMIN_AUTH" -d "$CREATE_BODY")
log "  POST /pm/adhoc/tasks → HTTP $CREATE_CODE"
cat "$TMP" | head -c 400; echo
TASK_ID=$(python3 -c "import sys,json;d=json.load(open('$TMP'));print((d.get('data') or {}).get('id',''))" 2>/dev/null)
if [ -z "$TASK_ID" ]; then
    bad "创建任务失败（无 id），response: $(cat $TMP)"
    exit 1
fi
ok "task created id=$TASK_ID creator=admin"

# ── §3 负向：alice 操作 admin 创建的任务 → 期望 403 ─────────────
log "§3 alice 非owner非超管 对该任务的5个写读接口"

assert_403() {
    local label="$1" code="$2" body="$3"
    if [ "$code" = "403" ] && echo "$body" | grep -q "not task owner"; then
        ok "$label → 403 + 'not task owner'"
    else
        bad "$label → $code, body: $(echo $body | head -c 200)"
    fi
}

T=$(mktemp)
PATCH_BODY='{"name":"hacked","metric_paths":["KGNB0510"],"granularities":["1h"],"window_start":"2026-06-24T00:00:00Z","window_end":"2026-06-25T00:00:00Z"}'
CODE=$(http_code PATCH  "$API/pm/adhoc/tasks/$TASK_ID" "$T" -H 'Content-Type: application/json' -H "$ALICE_AUTH" -d "$PATCH_BODY")
assert_403 "PATCH /tasks/:id (编辑)" "$CODE" "$(cat $T)"

CODE=$(http_code DELETE "$API/pm/adhoc/tasks/$TASK_ID" "$T" -H "$ALICE_AUTH")
assert_403 "DELETE /tasks/:id (取消)" "$CODE" "$(cat $T)"

CODE=$(http_code DELETE "$API/pm/adhoc/tasks/$TASK_ID/definition" "$T" -H "$ALICE_AUTH")
assert_403 "DELETE /tasks/:id/definition (硬删)" "$CODE" "$(cat $T)"

CODE=$(http_code GET    "$API/pm/adhoc/tasks/$TASK_ID/results" "$T" -H "$ALICE_AUTH")
assert_403 "GET /tasks/:id/results" "$CODE" "$(cat $T)"

CODE=$(http_code GET    "$API/pm/adhoc/tasks/$TASK_ID/runs" "$T" -H "$ALICE_AUTH")
assert_403 "GET /tasks/:id/runs" "$CODE" "$(cat $T)"

CODE=$(http_code GET    "$API/pm/adhoc/tasks/$TASK_ID/filter-options" "$T" -H "$ALICE_AUTH")
assert_403 "GET /tasks/:id/filter-options" "$CODE" "$(cat $T)"

# ── §4 admin (super admin) 操作他人任务 → 期望非 403 ─────────────
log "§4 admin super-admin 操作同一任务"
CODE=$(http_code GET    "$API/pm/adhoc/tasks/$TASK_ID/results" "$T" -H "$ADMIN_AUTH")
if [ "$CODE" != "403" ]; then
    ok "admin GET /results -> $CODE 超管放行"
else
    bad "admin GET /results -> 403 超管被误拦"
fi

# ── §5 内置任务: 普通用户读 → 期望非 403 ─────────────────────────
log "§5 alice 读内置任务 is_builtin=true"
BUILTIN_ID=$(docker exec "$PG_CONT" psql -U omcgo -d omcgo -t -A -c "SELECT id FROM pm_tasks WHERE is_builtin=true LIMIT 1;" 2>/dev/null | tr -d '[:space:]')
if [ -z "$BUILTIN_ID" ]; then
    warn "没找到内置任务，跳过 §5"
else
    log "  内置任务 id=$BUILTIN_ID"
    CODE=$(http_code GET "$API/pm/adhoc/tasks/$BUILTIN_ID/results" "$T" -H "$ALICE_AUTH")
    if [ "$CODE" != "403" ]; then
        ok "alice GET 内置任务 /results -> $CODE 内置全员可读"
    else
        bad "alice GET 内置任务 /results -> 403 误拦内置"
    fi
fi

# ── §6 alice 操作自己的任务 → 期望非 403 ─────────────────
log "§6 alice 自建任务 owner"
ALICE_CREATE_BODY=$(echo "$CREATE_BODY" | sed 's/issue-652-verify-task/issue-652-alice-own/')
CODE=$(http_code POST "$API/pm/adhoc/tasks" "$T" \
    -H 'Content-Type: application/json' -H "$ALICE_AUTH" -d "$ALICE_CREATE_BODY")
log "  alice POST /pm/adhoc/tasks → HTTP $CODE"
ALICE_TASK_ID=$(python3 -c "import sys,json;
try:
    d=json.load(open('$T'));print((d.get('data') or {}).get('id',''))
except: print('')" 2>/dev/null)
if [ -z "$ALICE_TASK_ID" ]; then
    warn "alice 创建任务失败 (code=$CODE) body=$(cat $T | head -c 200)，跳过 §6"
else
    log "  alice task_id=$ALICE_TASK_ID"
    CODE=$(http_code GET "$API/pm/adhoc/tasks/$ALICE_TASK_ID/results" "$T" -H "$ALICE_AUTH")
    if [ "$CODE" != "403" ]; then
        ok "alice 读自己任务 /results -> $CODE owner放行"
    else
        bad "alice 读自己任务 /results -> 403 自己被误拦"
    fi
    CODE=$(http_code DELETE "$API/pm/adhoc/tasks/$ALICE_TASK_ID/definition" "$T" -H "$ALICE_AUTH")
    if [ "$CODE" != "403" ]; then
        ok "alice 硬删自己任务 -> $CODE owner放行"
    else
        bad "alice 硬删自己任务 -> 403 自己被误拦"
    fi
fi

# ── §7 清理 ─────────────────────────────────────────────────────
log "§7 清理: 硬删 admin 创建的测试任务 + 删 alice 行"
http_code DELETE "$API/pm/adhoc/tasks/$TASK_ID" /dev/null -H "$ADMIN_AUTH" > /dev/null
http_code DELETE "$API/pm/adhoc/tasks/$TASK_ID/definition" /dev/null -H "$ADMIN_AUTH" > /dev/null
docker exec "$PG_CONT" psql -U omcgo -d omcgo -c "
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE username='$ALICE_USER');
DELETE FROM users WHERE username='$ALICE_USER';
DELETE FROM role_api_permissions WHERE role_id='$PM_TESTER_ROLE_ID'::uuid;
DELETE FROM roles WHERE id='$PM_TESTER_ROLE_ID'::uuid;
" > /dev/null 2>&1

# ── 汇总 ────────────────────────────────────────────────────────
echo
echo "==========================================="
echo "  Results: PASS=$PASS  FAIL=$FAIL"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1

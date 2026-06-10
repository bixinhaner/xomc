# shellcheck shell=bash
# =============================================================================
# omcgo/scripts/smoke/lib.sh — 业务冒烟测试共享库
#
# 被各 smoke_<业务域>.sh source 使用：
#
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "$SCRIPT_DIR/lib.sh"
#   smoke_init "设备管理(F06)" "$@"
#   smoke_login
#   req GET "/api/v1/devices"
#   check_ret_ok "设备列表可查"
#   smoke_summary
#
# 约定（与后端实现对齐，2026-06 实测）：
#   - BASE_URL：位置参数 $1 > 环境变量 OMC_BASE_URL > http://localhost:8081
#   - 统一响应信封 {ret, msg, data}，ret=1 成功 / ret=0 失败（失败附 biz_code）
#   - 分页风格 page/page_size；列表多在 data.items（任务历史在 data.tasks）
#   - 登录必须 RSA-OAEP/SHA-256 加密密码（明文登录默认禁用，biz_code 7004）
#   - 退出码：0 全部通过 / 1 存在 FAIL / 2 调用错误（栈不可达、登录失败）
#
# 依赖：bash、curl、python3（+ cryptography 库，仅登录加密用）
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# 全局状态
# ---------------------------------------------------------------------------
SMOKE_SUITE=""
BASE_URL=""
API=""
TOKEN=""
PUBLIC_KEY_PEM=""
PUBLIC_KEY_ID=""
HTTP_CODE=""
BODY=""
PASS_CNT=0
FAIL_CNT=0
SKIP_CNT=0
KNOWN_CNT=0
FAILED_CASES=()
SMOKE_TMPDIR=""
# 每次运行唯一后缀，冒烟自建实体（用户/模板/分组…）命名带上它，避免残留冲突
SMOKE_TAG="smk$(date +%s)$RANDOM"

ADMIN_USER="${OMC_ADMIN_USER:-admin}"
ADMIN_PASS="${OMC_ADMIN_PASS:-admin123}"

# ---------------------------------------------------------------------------
# 初始化与收尾
# ---------------------------------------------------------------------------

# smoke_init "套件中文名" "$@"   —— 解析 BASE_URL、预检 /healthz
smoke_init() {
    SMOKE_SUITE="$1"
    shift || true
    BASE_URL="${1:-${OMC_BASE_URL:-http://localhost:8081}}"
    BASE_URL="${BASE_URL%/}"
    API="$BASE_URL/api/v1"
    SMOKE_TMPDIR=$(mktemp -d)
    trap 'rm -rf "$SMOKE_TMPDIR"' EXIT

    echo "════════════════════════════════════════════════════════"
    echo "  OMC 业务冒烟 · $SMOKE_SUITE"
    echo "  Target: $BASE_URL"
    echo "  Time:   $(date '+%Y-%m-%d %H:%M:%S')"
    echo "════════════════════════════════════════════════════════"

    local code
    code=$(curl --max-time 5 -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz" 2>/dev/null || echo "000")
    if [ "$code" != "200" ]; then
        echo "❌ 预检失败：$BASE_URL/healthz → ${code}（栈未启动或地址不对）"
        echo "   本地容器栈 app 直连示例：bash $0 http://localhost:18091"
        exit 2
    fi
    echo "  预检 /healthz → 200，开始执行"
    echo ""
}

# smoke_summary —— 打印汇总并按结果退出（同时输出机器可读 RESULT 行供 run_all.sh 聚合）
smoke_summary() {
    local total=$((PASS_CNT + FAIL_CNT))
    echo ""
    echo "────────────────────────────────────────────────────────"
    echo "  $SMOKE_SUITE 汇总: PASS=$PASS_CNT FAIL=$FAIL_CNT SKIP=$SKIP_CNT KNOWN_BUG=$KNOWN_CNT (断言 $total)"
    if [ "$FAIL_CNT" -gt 0 ]; then
        echo "  失败用例:"
        local c
        for c in "${FAILED_CASES[@]}"; do echo "    ✗ $c"; done
    fi
    echo "RESULT|$SMOKE_SUITE|pass=$PASS_CNT|fail=$FAIL_CNT|skip=$SKIP_CNT|known=$KNOWN_CNT"
    echo "────────────────────────────────────────────────────────"
    [ "$FAIL_CNT" -eq 0 ] && exit 0 || exit 1
}

# ---------------------------------------------------------------------------
# 登录（RSA-OAEP/SHA-256 加密，照搬 e2e_verify.sh 经验证写法）
# 注意：公钥获取必须在主作用域完成（不能在 $(...) 子 shell 里首调，否则缓存丢失）
# ---------------------------------------------------------------------------

encrypt_password() {
    local plain="$1"
    if [ -z "$PUBLIC_KEY_PEM" ]; then
        local pk_resp
        pk_resp=$(curl --max-time 5 -s "$API/auth/public-key")
        PUBLIC_KEY_PEM=$(printf '%s' "$pk_resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('public_key',''))" 2>/dev/null)
        PUBLIC_KEY_ID=$(printf '%s' "$pk_resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('key_id',''))" 2>/dev/null)
        if [ -z "$PUBLIC_KEY_PEM" ] || [ -z "$PUBLIC_KEY_ID" ]; then
            echo "ERROR: /auth/public-key 获取失败" >&2
            return 1
        fi
    fi
    PEM_INPUT="$PUBLIC_KEY_PEM" PLAIN_INPUT="$plain" python3 <<'PYEOF'
import os, sys, json, time, secrets, base64
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import padding
plain = os.environ['PLAIN_INPUT']
pub = serialization.load_pem_public_key(os.environ['PEM_INPUT'].encode())
payload = json.dumps({
    "password": plain,
    "ts": int(time.time()),
    "nonce": secrets.token_hex(16),
}).encode()
ct = pub.encrypt(payload, padding.OAEP(
    mgf=padding.MGF1(algorithm=hashes.SHA256()),
    algorithm=hashes.SHA256(), label=None))
sys.stdout.write(base64.b64encode(ct).decode())
PYEOF
}

# smoke_login [用户名] [密码] —— 登录拿 JWT 存入 TOKEN；失败 exit 2
smoke_login() {
    local user="${1:-$ADMIN_USER}" pass="${2:-$ADMIN_PASS}"
    # 主作用域预取公钥（见上方注意事项）
    encrypt_password "warmup" >/dev/null || { echo "❌ 公钥获取失败"; exit 2; }
    local enc
    enc=$(encrypt_password "$pass")
    local resp
    resp=$(curl --max-time 10 -s -X POST "$API/auth/login" \
        -H 'Content-Type: application/json' \
        -d "{\"username\":\"$user\",\"encrypted_password\":\"$enc\",\"key_id\":\"$PUBLIC_KEY_ID\"}")
    TOKEN=$(printf '%s' "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('access_token',''))" 2>/dev/null)
    if [ -z "$TOKEN" ]; then
        echo "❌ 登录失败（user=${user}）：$(printf '%s' "$resp" | head -c 300)"
        exit 2
    fi
    echo "  登录成功（user=${user}）"
}

# ---------------------------------------------------------------------------
# 请求封装：执行后设置全局 HTTP_CODE / BODY
# ---------------------------------------------------------------------------

# req METHOD PATH [JSON_BODY] [额外 curl 参数...]
#   PATH 以 / 开头：/api/v1/... 自动拼 BASE_URL；省略前缀的视为 API 相对路径
req() {
    local method="$1" path="$2" body="${3:-}"
    shift 2; [ $# -gt 0 ] && shift
    local url
    case "$path" in
        http*) url="$path" ;;
        /api/*|/healthz|/readyz|/nedirect*|/smallcell*) url="$BASE_URL$path" ;;
        *) url="$API/${path#/}" ;;
    esac
    local args=(--max-time 15 -s -X "$method" -H "Authorization: Bearer $TOKEN")
    if [ -n "$body" ]; then
        args+=(-H 'Content-Type: application/json' -d "$body")
    fi
    args+=("$@")
    local resp
    resp=$(curl "${args[@]}" -w $'\n%{http_code}' "$url" 2>/dev/null) || { HTTP_CODE="000"; BODY=""; return; }
    HTTP_CODE=$(printf '%s' "$resp" | tail -1)
    BODY=$(printf '%s' "$resp" | sed '$d')
}

# req_noauth METHOD PATH [JSON_BODY] —— 免认证请求（healthz/public 接口/负路径验证）
req_noauth() {
    local method="$1" path="$2" body="${3:-}"
    local url
    case "$path" in
        http*) url="$path" ;;
        /api/*|/healthz|/readyz|/nedirect*|/smallcell*) url="$BASE_URL$path" ;;
        *) url="$API/${path#/}" ;;
    esac
    local args=(--max-time 15 -s -X "$method")
    [ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
    local resp
    resp=$(curl "${args[@]}" -w $'\n%{http_code}' "$url" 2>/dev/null) || { HTTP_CODE="000"; BODY=""; return; }
    HTTP_CODE=$(printf '%s' "$resp" | tail -1)
    BODY=$(printf '%s' "$resp" | sed '$d')
}

# req_upload PATH FIELD=@file [更多 -F 参数...] —— multipart 上传
req_upload() {
    local path="$1"; shift
    local url="$BASE_URL$path"
    local fargs=()
    local a
    for a in "$@"; do fargs+=(-F "$a"); done
    local resp
    resp=$(curl --max-time 30 -s -X POST -H "Authorization: Bearer $TOKEN" "${fargs[@]}" -w $'\n%{http_code}' "$url" 2>/dev/null) || { HTTP_CODE="000"; BODY=""; return; }
    HTTP_CODE=$(printf '%s' "$resp" | tail -1)
    BODY=$(printf '%s' "$resp" | sed '$d')
}

# ---------------------------------------------------------------------------
# JSON 取值（基于全局 BODY；点分路径，列表用数字下标，如 data.items.0.id）
# ---------------------------------------------------------------------------

jget() {
    local path="$1"
    printf '%s' "$BODY" | JPATH="$path" python3 -c "
import sys, json, os
try:
    v = json.load(sys.stdin)
except Exception:
    print(''); sys.exit(0)
for p in os.environ['JPATH'].split('.'):
    if isinstance(v, list):
        v = v[int(p)] if p.lstrip('-').isdigit() and -len(v) <= int(p) < len(v) else None
    elif isinstance(v, dict):
        v = v.get(p)
    else:
        v = None
    if v is None:
        break
if v is None:
    print('')
elif isinstance(v, (dict, list)):
    print(json.dumps(v, ensure_ascii=False))
else:
    print(v)
" 2>/dev/null || echo ""
}

# jlen "data.items" —— 列表长度（null/缺失/非列表 → 0；兼容 items=null 的端点）
jlen() {
    local path="$1"
    printf '%s' "$BODY" | JPATH="$path" python3 -c "
import sys, json, os
try:
    v = json.load(sys.stdin)
except Exception:
    print(0); sys.exit(0)
for p in os.environ['JPATH'].split('.'):
    if isinstance(v, list):
        v = v[int(p)] if p.lstrip('-').isdigit() and -len(v) <= int(p) < len(v) else None
    elif isinstance(v, dict):
        v = v.get(p)
    else:
        v = None
    if v is None:
        break
print(len(v) if isinstance(v, list) else 0)
" 2>/dev/null || echo 0
}

# ---------------------------------------------------------------------------
# 断言（全部操作全局 HTTP_CODE / BODY）
# ---------------------------------------------------------------------------

pass()  { echo "  ✅ $1"; PASS_CNT=$((PASS_CNT + 1)); }
fail()  { echo "  ❌ $1 — ${2:-}"; FAIL_CNT=$((FAIL_CNT + 1)); FAILED_CASES+=("$1: ${2:-}"); }
skip()  { echo "  ⏭️  $1 — ${2:-跳过}"; SKIP_CNT=$((SKIP_CNT + 1)); }
# 已知后端 bug：报告但不计入失败（修复后改回正常断言）
known_bug() { echo "  🐞 $1 — 已知问题: ${2:-}"; KNOWN_CNT=$((KNOWN_CNT + 1)); }
section() { echo ""; echo "—— $1 ——"; }

# check_status "desc" 200 —— HTTP 状态精确匹配
check_status() {
    local desc="$1" expected="$2"
    if [ "$HTTP_CODE" = "$expected" ]; then pass "$desc → $HTTP_CODE"
    else fail "$desc" "期望 HTTP ${expected}，实际 ${HTTP_CODE}，body: $(printf '%s' "$BODY" | head -c 200)"; fi
}

# check_status_in "desc" "200 404" —— HTTP 状态在白名单内
check_status_in() {
    local desc="$1" expected_set="$2" code
    for code in $expected_set; do
        if [ "$HTTP_CODE" = "$code" ]; then pass "$desc → $HTTP_CODE (∈ {$expected_set})"; return 0; fi
    done
    fail "$desc" "期望 HTTP ∈ {$expected_set}，实际 ${HTTP_CODE}，body: $(printf '%s' "$BODY" | head -c 200)"
    return 1
}

# check_ret_ok "desc" —— HTTP 2xx 且信封 ret=1（成功）
check_ret_ok() {
    local desc="$1" ret
    ret=$(jget ret)
    if [[ "$HTTP_CODE" == 2* ]] && [ "$ret" = "1" ]; then pass "$desc → $HTTP_CODE ret=1"
    else fail "$desc" "期望 2xx+ret=1，实际 HTTP $HTTP_CODE ret=${ret}，body: $(printf '%s' "$BODY" | head -c 200)"; fi
}

# check_ret_fail "desc" —— 负路径：HTTP 4xx/5xx 或 ret=0（参数校验被正确拒绝）
check_ret_fail() {
    local desc="$1" ret
    ret=$(jget ret)
    if [[ "$HTTP_CODE" == 4* || "$HTTP_CODE" == 5* ]] || [ "$ret" = "0" ]; then pass "$desc → $HTTP_CODE ret=${ret}（按预期被拒绝）"
    else fail "$desc" "期望被拒绝(4xx/5xx 或 ret=0)，实际 HTTP $HTTP_CODE ret=$ret"; fi
}

# check_field "desc" "data.items.0.id" —— 字段存在且非空（值回显便于排查）
check_field() {
    local desc="$1" path="$2" v
    v=$(jget "$path")
    if [ -n "$v" ] && [ "$v" != "None" ] && [ "$v" != "null" ]; then pass "$desc ($path=$(printf '%s' "$v" | head -c 60))"
    else fail "$desc" "字段 $path 缺失或为空"; fi
}

# check_list_nonempty "desc" "data.items" —— 列表非空（强断言，用于 builtin 字典类）
check_list_nonempty() {
    local desc="$1" path="$2" n
    n=$(jlen "$path")
    if [ "$n" -gt 0 ]; then pass "$desc ($path 共 $n 条)"
    else fail "$desc" "列表 $path 为空（期望非空）"; fi
}

# check_list_or_empty "desc" "data.items" —— 列表可查即过（容忍稀疏种子，回显条数）
check_list_or_empty() {
    local desc="$1" path="$2" ret n
    ret=$(jget ret)
    if [[ "$HTTP_CODE" == 2* ]] && [ "$ret" = "1" ]; then
        n=$(jlen "$path")
        pass "$desc → 200 ret=1 ($path 共 $n 条)"
    else
        fail "$desc" "期望 2xx+ret=1，实际 HTTP $HTTP_CODE ret=${ret}，body: $(printf '%s' "$BODY" | head -c 200)"
    fi
}

# check_count_ge "desc" "data.total" 1 —— 数值字段 ≥ N
check_count_ge() {
    local desc="$1" path="$2" min="$3" v
    v=$(jget "$path")
    if [ -n "$v" ] && [ "$v" -ge "$min" ] 2>/dev/null; then pass "$desc ($path=$v ≥ $min)"
    else fail "$desc" "期望 $path ≥ ${min}，实际 '$v'"; fi
}

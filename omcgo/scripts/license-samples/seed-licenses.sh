#!/bin/bash
# seed-licenses.sh — 一键导入 license 测试样本到 OMC 实例
#
# 用法：
#   ./seed-licenses.sh                       # 默认 http://localhost:8081，admin/admin123 自动加密登录
#   API=http://172.21.175.129:8081 ./seed-licenses.sh
#   OMC_USER=admin OMC_PASS=OMC@123456 ./seed-licenses.sh
#   TOKEN=<bearer> ./seed-licenses.sh        # 跳过登录，直接用预拿的 token
#
# 登录路径：
#   后端禁用明文密码登录（biz_code 7004）。脚本走加密登录：
#     1) GET  /auth/public-key   → 拿 RSA 公钥（PEM）+ key_id
#     2) 构造 {password, ts, nonce} JSON，openssl pkeyutl RSA-OAEP/SHA-256 加密 → base64
#     3) POST /auth/login        → {username, encrypted_password, key_id} → access_token
#   依赖：openssl 1.1.1+（macOS 默认 LibreSSL 也支持 pkeyutl 选项）+ python3
#
# 输出：每个 license 一行结果（imported/skipped/failed）+ 末尾当前 license 列表预览
# 重复执行：已存在的 license_code 会归到 skipped，幂等。

set -uo pipefail

API="${API:-http://localhost:8081}"
# 注意：用 OMC_USER / OMC_PASS 而不是 USER / PASS，避免与 shell 内置 $USER 冲突
# （macOS/Linux 登录会自动 export USER=<system-user>，覆盖 default admin）。
OMC_USER="${OMC_USER:-admin}"
OMC_PASS="${OMC_PASS:-admin123}"
TOKEN="${TOKEN:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ---- 健壮 JSON 字段提取（防 NoneType 报错） ----
py_get() {
  # py_get '<json>' 'a.b.c'  → 多层 get，任一层为 None / 非 dict 返空串
  python3 -c '
import json, sys
raw = sys.stdin.read()
try:
  d = json.loads(raw)
except Exception:
  print(""); sys.exit(0)
keys = sys.argv[1].split(".") if len(sys.argv) > 1 else []
cur = d
for k in keys:
  if isinstance(cur, dict):
    cur = cur.get(k)
  else:
    cur = None
    break
print(cur if isinstance(cur, (str, int, float)) else "")
' "$1"
}

# ---- Token 获取 ----
if [ -n "$TOKEN" ]; then
  echo "✓ 使用预传入 TOKEN（${TOKEN:0:24}...）"
else
  echo "→ 加密登录 ${API} (用户 ${OMC_USER}) ..."

  # 1) 拉公钥
  PK_RESP=$(curl -sS "$API/api/v1/auth/public-key" || true)
  if [ -z "$PK_RESP" ]; then
    echo "✗ 拉公钥失败（API 不可达？）" >&2
    exit 1
  fi

  KEY_ID=$(printf '%s' "$PK_RESP" | py_get "data.key_id")
  [ -z "$KEY_ID" ] && KEY_ID=$(printf '%s' "$PK_RESP" | py_get "key_id")

  PEM=$(printf '%s' "$PK_RESP" | python3 -c '
import json, sys
d = json.load(sys.stdin)
data = d.get("data") if isinstance(d.get("data"), dict) else d
print(data.get("public_key", ""))
')

  if [ -z "$KEY_ID" ] || [ -z "$PEM" ]; then
    echo "✗ 公钥响应缺字段 key_id/public_key" >&2
    echo "  原始响应: ${PK_RESP:0:300}" >&2
    exit 1
  fi
  echo "  ✓ 公钥已拿  key_id=${KEY_ID:0:16}..."

  # 2) 写公钥临时文件 + 构造 payload + RSA-OAEP/SHA-256 加密
  PEM_FILE=$(mktemp -t omc-pub-XXXXXX)
  PAYLOAD_FILE=$(mktemp -t omc-payload-XXXXXX)
  trap "rm -f $PEM_FILE $PAYLOAD_FILE" EXIT
  printf '%s\n' "$PEM" > "$PEM_FILE"

  TS=$(date +%s)
  NONCE=$(openssl rand -hex 16)
  printf '{"password":"%s","ts":%d,"nonce":"%s"}' "$OMC_PASS" "$TS" "$NONCE" > "$PAYLOAD_FILE"

  ENC=$(openssl pkeyutl -encrypt -pubin -inkey "$PEM_FILE" \
    -pkeyopt rsa_padding_mode:oaep \
    -pkeyopt rsa_oaep_md:sha256 \
    -pkeyopt rsa_mgf1_md:sha256 \
    -in "$PAYLOAD_FILE" 2>/dev/null | base64 | tr -d '\n ')

  if [ -z "$ENC" ]; then
    echo "✗ openssl RSA-OAEP 加密失败" >&2
    echo "  需要 openssl 1.1.1+ 或 LibreSSL 3.x 支持 rsa_oaep_md/rsa_mgf1_md" >&2
    echo "  当前: $(openssl version)" >&2
    exit 1
  fi
  echo "  ✓ 加密完成  encryptedPassword length=${#ENC}"

  # 3) 调登录端点
  LOGIN_BODY=$(printf '{"username":"%s","encrypted_password":"%s","key_id":"%s"}' "$OMC_USER" "$ENC" "$KEY_ID")
  LOGIN_RESP=$(curl -sS -X POST "$API/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "$LOGIN_BODY")

  TOKEN=$(printf '%s' "$LOGIN_RESP" | py_get "data.access_token")
  [ -z "$TOKEN" ] && TOKEN=$(printf '%s' "$LOGIN_RESP" | py_get "access_token")

  if [ -z "$TOKEN" ]; then
    echo "✗ 登录失败" >&2
    echo "  响应: ${LOGIN_RESP:0:400}" >&2
    echo "  提示: 用户名密码错？或试 TOKEN=<bearer> ./seed-licenses.sh 绕过登录" >&2
    exit 1
  fi
  echo "  ✓ token 已获取（${TOKEN:0:24}...）"
fi

# ---- 遍历 *.json 文件 import ----
imported=0; skipped=0; failed=0
for f in [0-9][0-9]-*.json; do
  [ -f "$f" ] || continue

  echo ""
  echo "→ 导入 $f"

  RESP=$(curl -sS -w "\n%{http_code}" -X POST "$API/api/v1/licenses/import" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$f")

  HTTP_CODE=$(echo "$RESP" | tail -1)
  BODY=$(echo "$RESP" | sed '$d')

  if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    LIC_ID=$(printf '%s' "$BODY" | py_get "data.license.id")
    [ -z "$LIC_ID" ] && LIC_ID=$(printf '%s' "$BODY" | py_get "license.id")
    SIG=$(printf '%s' "$BODY" | py_get "data.signature_status")
    [ -z "$SIG" ] && SIG=$(printf '%s' "$BODY" | py_get "signature_status")
    echo "  ✓ imported  id=${LIC_ID:0:8}...  signature_status=${SIG:-n/a}"
    imported=$((imported+1))
  elif [ "$HTTP_CODE" = "409" ] || echo "$BODY" | grep -qiE "duplicate|already exists|unique"; then
    echo "  ⊙ skipped (already exists, http=$HTTP_CODE)"
    skipped=$((skipped+1))
  else
    echo "  ✗ failed  http=$HTTP_CODE  body=${BODY:0:200}"
    failed=$((failed+1))
  fi
done

echo ""
echo "════════════════════════════════════"
echo "  imported: $imported   skipped: $skipped   failed: $failed"
echo "════════════════════════════════════"

# ---- 列出当前 license 数 ----
echo ""
echo "→ 当前 licenses（前 20 条）："
curl -sS "$API/api/v1/licenses?page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c '
import json, sys
try:
  d = json.load(sys.stdin)
except Exception:
  print("  (响应非 JSON)"); sys.exit(0)
data = d.get("data") if isinstance(d.get("data"), dict) else d
items = data.get("items") or []
total = data.get("total") or "?"
print("  total={}".format(total))
for it in items[:20]:
  status = it.get("status") or "?"
  code = it.get("license_code") or "?"
  name = it.get("license_name") or ""
  dt = it.get("device_type") or "-"
  rg = it.get("region") or "-"
  mx = it.get("max_devices")
  print("    [{}] {:<35} {} | {}/{} | max={}".format(status, code, name, dt, rg, mx))
'

#!/bin/bash
# seed-licenses.sh — 一键导入 license 测试样本到本地 OMC 实例
#
# 用法：
#   ./seed-licenses.sh                       # 默认 http://localhost:8081，admin/admin123
#   API=http://172.21.175.129:8081 ./seed-licenses.sh
#   USER=admin PASS=admin123 ./seed-licenses.sh
#
# 输出：每个 license 一行结果（imported/skipped/failed + license_id 或错误码）
#
# 前置：本机能访问 OMC 后端 :8081，admin/admin123 账号存在。
# 注意：本脚本只 import（status=pending）；激活/撤销需走 UI 或单独 curl。

set -uo pipefail

API="${API:-http://localhost:8081}"
USER="${USER:-admin}"
PASS="${PASS:-admin123}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ---- 1. 登录获取 token ----
echo "→ 登录 $API 获取 access_token..."
TOKEN=$(curl -sS -X POST "$API/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" \
  | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("data",{}).get("access_token") or d.get("access_token") or "")' )

if [ -z "$TOKEN" ]; then
  echo "✗ 登录失败，无法获取 access_token。检查 API/USER/PASS 后重试。"
  exit 1
fi
echo "✓ token 已获取（${TOKEN:0:24}...）"

# ---- 2. 遍历 *.json 文件 import ----
imported=0; skipped=0; failed=0
for f in [0-9][0-9]-*.json; do
  [ -f "$f" ] || continue

  echo ""
  echo "→ 导入 $f"

  # 用 python 美化 + 直接当 body 传
  RESP=$(curl -sS -w "\n%{http_code}" -X POST "$API/api/v1/licenses/import" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$f")

  HTTP_CODE=$(echo "$RESP" | tail -1)
  BODY=$(echo "$RESP" | sed '$d')

  if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    LIC_ID=$(echo "$BODY" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("data",{}).get("license",{}).get("id") or d.get("license",{}).get("id") or "(no id)")' 2>/dev/null || echo "(parse failed)")
    SIG_STATUS=$(echo "$BODY" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("data",{}).get("signature_status") or d.get("signature_status") or "n/a")' 2>/dev/null)
    echo "  ✓ imported  license_id=$LIC_ID  signature_status=$SIG_STATUS"
    imported=$((imported+1))
  elif [ "$HTTP_CODE" = "409" ] || echo "$BODY" | grep -qiE "duplicate|already exists|unique"; then
    echo "  ⊙ skipped (already exists)"
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

# ---- 3. 列出当前 license 数 ----
echo ""
echo "→ 当前 licenses 列表（前 20 条）："
curl -sS "$API/api/v1/licenses?page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c '
import json,sys
d=json.load(sys.stdin)
items=d.get("data",{}).get("items") or d.get("items") or []
total=d.get("data",{}).get("total") or d.get("total") or "?"
print(f"  total={total}")
for it in items[:20]:
    print(f"    [{it.get(\"status\")}] {it.get(\"license_code\"):<35} {it.get(\"license_name\")} | {it.get(\"device_type\") or \"-\"}/{it.get(\"region\") or \"-\"} | max={it.get(\"max_devices\")}")
'

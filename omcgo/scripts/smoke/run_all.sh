#!/usr/bin/env bash
# =============================================================================
# omcgo/scripts/smoke/run_all.sh — 按业务域顺序执行全部冒烟脚本并汇总
#
# 用法：
#   bash run_all.sh                          # 默认 http://localhost:8081
#   bash run_all.sh http://localhost:18091   # 本机容器栈 app 直连
#   SMOKE_ONLY="device alarm" bash run_all.sh <BASE>   # 只跑指定业务域
#   SMOKE_SKIP="acs" bash run_all.sh <BASE>            # 跳过指定业务域
#
# 退出码：0 全部业务域通过 / 1 任一业务域有 FAIL / 2 调用错误
# =============================================================================

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="${1:-${OMC_BASE_URL:-http://localhost:8081}}"

# 执行顺序：先认证/南向链路（其余域的会话与设备数据来源），再各业务域
SUITES=(
    auth acs device topology config product alarm pm mr
    software ufte backup dashboard ops report mml filemanager logs
    admin system task notification northbound provision interop nedirect
)

if [ -n "${SMOKE_ONLY:-}" ]; then
    SUITES=($SMOKE_ONLY)
fi

declare -a ROWS
TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_SKIP=0
TOTAL_KNOWN=0
FAILED_SUITES=()
START_TS=$(date +%s)

echo "════════════════════════════════════════════════════════"
echo "  OMC 业务冒烟全量执行（${#SUITES[@]} 个业务域）"
echo "  Target: $BASE_URL"
echo "════════════════════════════════════════════════════════"

for suite in "${SUITES[@]}"; do
    case " ${SMOKE_SKIP:-} " in *" $suite "*) echo ">> 跳过 $suite (SMOKE_SKIP)"; continue ;; esac
    script="$SCRIPT_DIR/smoke_${suite}.sh"
    if [ ! -f "$script" ]; then
        echo ">> 缺少脚本 $script，跳过"
        continue
    fi
    echo ""
    echo ">> [$suite] bash ${script##*/} $BASE_URL"
    out=$(bash "$script" "$BASE_URL" 2>&1)
    rc=$?
    echo "$out" | sed 's/^/   /'
    # 解析机器可读行 RESULT|套件名|pass=N|fail=N|skip=N|known=N
    line=$(echo "$out" | grep '^RESULT|' | tail -1)
    if [ -n "$line" ]; then
        p=$(echo "$line" | sed 's/.*pass=\([0-9]*\).*/\1/')
        f=$(echo "$line" | sed 's/.*fail=\([0-9]*\).*/\1/')
        s=$(echo "$line" | sed 's/.*skip=\([0-9]*\).*/\1/')
        k=$(echo "$line" | sed 's/.*known=\([0-9]*\).*/\1/')
    else
        p=0; f=1; s=0; k=0   # 脚本异常退出（如 exit 2）按失败记
    fi
    TOTAL_PASS=$((TOTAL_PASS + p))
    TOTAL_FAIL=$((TOTAL_FAIL + f))
    TOTAL_SKIP=$((TOTAL_SKIP + s))
    TOTAL_KNOWN=$((TOTAL_KNOWN + k))
    status="PASS"
    if [ "$rc" -ne 0 ] || [ "$f" -gt 0 ]; then status="FAIL"; FAILED_SUITES+=("$suite"); fi
    ROWS+=("$(printf '%-14s %-6s pass=%-4s fail=%-3s skip=%-3s known=%s' "$suite" "$status" "$p" "$f" "$s" "$k")")
done

ELAPSED=$(( $(date +%s) - START_TS ))
echo ""
echo "════════════════════════════════════════════════════════"
echo "  冒烟总汇总（耗时 ${ELAPSED}s）"
echo "────────────────────────────────────────────────────────"
for row in "${ROWS[@]}"; do echo "  $row"; done
echo "────────────────────────────────────────────────────────"
echo "  合计: PASS=$TOTAL_PASS FAIL=$TOTAL_FAIL SKIP=$TOTAL_SKIP KNOWN_BUG=$TOTAL_KNOWN"
if [ ${#FAILED_SUITES[@]} -gt 0 ]; then
    echo "  ❌ 失败业务域: ${FAILED_SUITES[*]}"
    exit 1
fi
echo "  ✅ 全部业务域通过"
exit 0

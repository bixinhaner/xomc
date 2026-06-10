#!/usr/bin/env bash
# =============================================================================
# smoke_nedirect.sh — F07 网元直连 业务冒烟
#
# 特殊性（与其他域不同）：
#   - nedirect 是独立 stdlib HTTP server（internal/nedirect/server.go），
#     不挂在 app :8081 上；dev 默认 ne_direct.enabled=false 整域不监听。
#   - 端口默认 :7549（config.dev.yaml ne_direct.port），可用环境变量
#     OMC_NEDIRECT_URL 覆盖（如 http://localhost:7549）。
#   - 响应是裸 JSON（writeJSON），不走统一信封 {ret,msg,data}，
#     断言一律用 check_status / check_status_in / check_field。
#   - 认证与管理面同源：Bearer JWT（app 登录所得）或 X-API-Key。
#
# 策略（对齐 /tmp/smoke_routes.json smokeKeyPaths）：
#   1. curl --max-time 2 探测 nedirect 根路径，连不通 → 整域 SKIP（每条路由
#      各记一个 skip），summary 仍 exit 0；
#   2. 连通 → 只测 status / sessions 读路径（带 query 参数 + 负路径）；
#      write/destructive 路由（register/config/fault/connect[/disconnect/command]）
#      按本域专项要求一律不触发，记 skip。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F07 网元直连" "$@"
smoke_login

# nedirect 地址独立于 BASE_URL（BASE_URL 是 app，仅用于登录/取设备 SN）
NEDIRECT_URL="${OMC_NEDIRECT_URL:-http://localhost:7549}"
NEDIRECT_URL="${NEDIRECT_URL%/}"

# ---------------------------------------------------------------------------
# 预检：探测 nedirect 独立端口（不用 smoke_init 的 /healthz 逻辑——那是 app 的）
# ---------------------------------------------------------------------------
section "nedirect 服务探测（${NEDIRECT_URL}）"
# 注意：curl 连接失败时 -w 仍会输出 "000"，不能再 || echo（否则拿到 "000000"）
NEDIRECT_CODE=$(curl --max-time 2 -s -o /dev/null -w "%{http_code}" "$NEDIRECT_URL/" 2>/dev/null)
[ -z "$NEDIRECT_CODE" ] && NEDIRECT_CODE="000"

if [ "$NEDIRECT_CODE" = "000" ]; then
    echo "  nedirect 端口不可达（dev 默认 ne_direct.enabled=false，整域不监听）→ 全域 SKIP"
    skip "POST /nedirect/register" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    skip "POST /nedirect/config（destructive，直连下发配置）" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    skip "GET /nedirect/status?serial_number=" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    skip "POST /nedirect/fault" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    skip "POST /nedirect/connect（含同组 /disconnect、/command）" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    skip "GET /nedirect/sessions?device_sn=" "nedirect 服务未启用（${NEDIRECT_URL} 不可达）"
    smoke_summary
fi

echo "  nedirect 端口可达（根路径 → HTTP ${NEDIRECT_CODE}），按启用环境只测读路径"

# ---------------------------------------------------------------------------
# 数据自备：从 app 取一台设备 SN（取不到则用必不存在的 SN，断言相应放宽）
# ---------------------------------------------------------------------------
section "数据准备：从 app 取设备 SN"
req GET "/api/v1/devices?page=1&page_size=1"
check_ret_ok "设备列表可查（取 SN 用）"
DEV_SN=$(jget data.items.0.serial_number)
if [ -n "$DEV_SN" ]; then
    echo "  使用设备 SN: ${DEV_SN}"
else
    echo "  活栈无设备，status 用不存在 SN 走 404 路径"
fi
MISSING_SN="${SMOKE_TAG}-NOT-EXIST"

# ---------------------------------------------------------------------------
# GET /nedirect/status —— 设备状态读路径（裸 JSON，无信封）
# ---------------------------------------------------------------------------
section "status 读路径"
if [ -n "$DEV_SN" ]; then
    req GET "${NEDIRECT_URL}/nedirect/status?serial_number=${DEV_SN}"
    check_status "status 真实 SN 可查" 200
    check_field "status 回显 serial_number" "serial_number"
    check_field "status 回显 device_id" "device_id"
else
    skip "status 真实 SN 可查" "活栈无设备且 nedirect 不提供建设备入口"
fi

req GET "${NEDIRECT_URL}/nedirect/status?serial_number=${MISSING_SN}"
check_status "status 不存在 SN → 404" 404

req GET "${NEDIRECT_URL}/nedirect/status"
check_status "status 缺 serial_number → 400" 400

req_noauth GET "${NEDIRECT_URL}/nedirect/status?serial_number=${MISSING_SN}"
check_status "status 未认证 → 401" 401

# ---------------------------------------------------------------------------
# GET /nedirect/sessions —— 会话列表读路径（裸 JSON SessionListResult）
# ---------------------------------------------------------------------------
section "sessions 读路径"
QUERY_SN="${DEV_SN:-$MISSING_SN}"
req GET "${NEDIRECT_URL}/nedirect/sessions?device_sn=${QUERY_SN}"
check_status "sessions 按 device_sn 过滤可查" 200

req GET "${NEDIRECT_URL}/nedirect/sessions?device_sn=${QUERY_SN}&status=active"
check_status "sessions 按 status=active 过滤可查" 200

req_noauth GET "${NEDIRECT_URL}/nedirect/sessions?device_sn=${QUERY_SN}"
check_status "sessions 未认证 → 401" 401

# ---------------------------------------------------------------------------
# write/destructive 路由 —— 本域专项要求：启用环境也只测读路径，不触发写/下发
# （register 会写设备表、config 直连下发、fault 造告警、connect 建真实会话）
# ---------------------------------------------------------------------------
section "write/destructive 路由（按专项要求不触发）"
skip "POST /nedirect/register" "本域专项要求：启用环境仅测 status/sessions 读路径"
skip "POST /nedirect/config（destructive，直连下发配置）" "本域专项要求：启用环境仅测 status/sessions 读路径"
skip "POST /nedirect/fault" "本域专项要求：启用环境仅测 status/sessions 读路径"
skip "POST /nedirect/connect（含同组 /disconnect、/command）" "本域专项要求：启用环境仅测 status/sessions 读路径"

smoke_summary

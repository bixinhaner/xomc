#!/usr/bin/env bash
# =============================================================================
# smoke_provision.sh — F09 自动开站 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=provision 全部 4 条路由）：
#   GET  /api/v1/provisioning/tasks        列表（含 status / device_id 过滤 + 非法参数负路径）
#   GET  /api/v1/provisioning/tasks/:id    详情（取列表第一条；非法 UUID / 不存在 ID 负路径）
#   POST /api/v1/provisioning/tasks        destructive → 只测参数校验负路径（缺必填/非法格式/不存在设备 → 404）
#   POST /api/v1/provisioning/tasks/:id/retry  destructive → 只测负路径（不存在 ID / 非法 UUID）
#
# 红线遵守：绝不对真实设备触发开站下发——POST 仅用不存在的 device_id（UUID 格式合法
# 但设备不存在；issue #126 第10项修复后 handler 入库前预检 device 存在性 → 直接 404，
# 既不下发也不建任何任务）；retry 仅打不存在 ID / 非法 UUID，不碰列表里的真实任务。
#
# 用法：bash smoke_provision.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F09 自动开站" "$@"
smoke_login

# 不存在的设备/任务 UUID（格式合法，库中不存在）
NO_SUCH_ID="aaaaaaaa-bbbb-cccc-dddd-eeeeffff0000"

# ---------------------------------------------------------------------------
section "开站任务列表（GET /provisioning/tasks）"
# ---------------------------------------------------------------------------
req GET "/api/v1/provisioning/tasks"
check_ret_ok "开站任务列表可查"
check_list_nonempty "任务列表非空（活栈有种子任务）" "data.items"
check_count_ge "total 字段 ≥ 1" "data.total" 1
check_field "首条任务含 id" "data.items.0.id"
check_field "首条任务含 device_id" "data.items.0.device_id"
check_field "首条任务含 status" "data.items.0.status"

FIRST_ID=$(jget data.items.0.id)
FIRST_DEVICE_ID=$(jget data.items.0.device_id)

# status 过滤（合法枚举值，活栈数据稀疏 → 可查即过）
req GET "/api/v1/provisioning/tasks?status=completed"
check_list_or_empty "按 status=completed 过滤可查" "data.items"
req GET "/api/v1/provisioning/tasks?status=failed"
check_list_or_empty "按 status=failed 过滤可查" "data.items"

# device_id 过滤（用列表第一条的 device_id，应至少命中它自己）
if [ -n "$FIRST_DEVICE_ID" ]; then
    req GET "/api/v1/provisioning/tasks?device_id=$FIRST_DEVICE_ID"
    check_ret_ok "按 device_id 过滤可查"
    check_list_nonempty "device_id 过滤命中自身" "data.items"
else
    skip "按 device_id 过滤" "列表为空取不到 device_id"
fi

# 负路径：device_id 非 UUID → 400
req GET "/api/v1/provisioning/tasks?device_id=not-a-uuid"
check_ret_fail "device_id 非法格式被拒绝"
check_status "device_id 非法格式 HTTP 400" 400

# ---------------------------------------------------------------------------
section "开站任务详情（GET /provisioning/tasks/:id）"
# ---------------------------------------------------------------------------
if [ -n "$FIRST_ID" ]; then
    req GET "/api/v1/provisioning/tasks/$FIRST_ID"
    check_ret_ok "任务详情可查（列表第一条）"
    check_field "详情 id 回显一致" "data.id"
    DETAIL_ID=$(jget data.id)
    if [ "$DETAIL_ID" = "$FIRST_ID" ]; then
        pass "详情 id 与列表一致 ($DETAIL_ID)"
    else
        fail "详情 id 与列表一致" "期望 ${FIRST_ID}，实际 ${DETAIL_ID}"
    fi
    check_field "详情含 status" "data.status"
    check_field "详情含 max_retries" "data.max_retries"
else
    skip "任务详情" "列表为空取不到任务 ID"
fi

# 负路径：非法 UUID → 400；不存在 ID → 404
req GET "/api/v1/provisioning/tasks/not-a-uuid"
check_ret_fail "详情非法 UUID 被拒绝"
check_status "详情非法 UUID HTTP 400" 400
req GET "/api/v1/provisioning/tasks/$NO_SUCH_ID"
check_ret_fail "详情不存在 ID 被拒绝"
check_status "详情不存在 ID HTTP 404" 404

# ---------------------------------------------------------------------------
section "建任务只测参数校验（POST /provisioning/tasks，destructive 禁真发）"
# ---------------------------------------------------------------------------
# 缺必填 device_id → 400（binding required）
req POST "/api/v1/provisioning/tasks" '{}'
check_ret_fail "缺 device_id 被拒绝"
check_status "缺 device_id HTTP 400" 400

# device_id 非 UUID 格式 → 400
req POST "/api/v1/provisioning/tasks" '{"device_id":"not-a-uuid"}'
check_ret_fail "device_id 非 UUID 被拒绝"
check_status "device_id 非 UUID HTTP 400" 400

# 非法 JSON body → 400
req POST "/api/v1/provisioning/tasks" '{"device_id":'
check_ret_fail "非法 JSON body 被拒绝"

# 不存在的设备 ID：handler 入库前预检 device 存在性，不存在 → 404，不建孤儿任务
# （issue #126 第10项已修复：provisioning_tasks 对 device_id 无外键，需 handler 兜底）
req POST "/api/v1/provisioning/tasks" "{\"device_id\":\"$NO_SUCH_ID\"}"
check_ret_fail "不存在设备 ID 建任务被拒绝（设备存在性预检）"
check_status "不存在设备 ID HTTP 404" 404

# ---------------------------------------------------------------------------
section "重试只测负路径（POST /provisioning/tasks/:id/retry，destructive 禁真发）"
# ---------------------------------------------------------------------------
# 不存在的任务 ID → 404
req POST "/api/v1/provisioning/tasks/$NO_SUCH_ID/retry"
check_ret_fail "retry 不存在任务 ID 被拒绝"
check_status "retry 不存在 ID HTTP 404" 404

# 非法 UUID → 400
req POST "/api/v1/provisioning/tasks/not-a-uuid/retry"
check_ret_fail "retry 非法 UUID 被拒绝"
check_status "retry 非法 UUID HTTP 400" 400

# 状态机守卫：非 failed 状态任务不可 retry。issue #126 第10项修复后建任务需
# 设备存在，无法再用不存在设备自建孤儿任务做安全样本；列表里的真实任务 destructive
# 禁触达 → 跳过此守卫断言（由带真实 failed 任务的环境或单测 state_machine_test 兜底）。
skip "retry 非 failed 状态守卫" "设备存在性预检后无法安全自建非 failed 任务；destructive 禁触达真实任务"

smoke_summary

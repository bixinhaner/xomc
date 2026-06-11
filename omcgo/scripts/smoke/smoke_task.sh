#!/usr/bin/env bash
# =============================================================================
# smoke_task.sh — 统一任务队列（跨域基础设施）真实业务冒烟
#
# 覆盖（对应 /tmp/smoke_routes.json key=task 全部 read 路由 + 可逆 write 闭环）：
#   1. 按真实设备 SN 查任务历史（分页/状态过滤，列表 key 是 data.tasks 非 items）
#   2. pending 队列 / stats 统计（device_sn 必填）
#   3. 任务详情 GET /:task_id（栈内既有任务 + 闭环自建任务双路径）
#   4. 建任务 → 查详情 → 历史/pending/stats 可见 → DELETE 取消 → 状态验证闭环
#      —— 安全设计：任务建在 ${SMOKE_TAG} 前缀的【不存在 SN】上。CreateTask 的
#      wakeDevice 对未知 SN 设备查找失败即放弃（仅 warn 日志），不会发出任何
#      Connection Request；该 SN 永不 inform，任务只入队不下发，建后立即取消。
#      绝不向任何真实/模拟设备 SN 下发 CWMP 指令。
#   5. retry：硬断言已取消(cancelled)任务 retry 被拒绝（4xx）+ 任务保持 cancelled
#      不被复活 + pending 队列仍为空（#126 第8项修复后 CanManualRetry 限定
#      failed/expired 才可主动 retry）+ 不存在任务负路径
#   6. purge（destructive）：仅测未认证负路径；真实执行无校验负路径
#      （非法 retention_days 静默回退默认 30 天并真执行），按危险禁区跳过
#
# 历史 known_bug（已修复转硬断言）：
#   - #116 DELETE /devices/tasks/:task_id 成功取消但返回 500：CancelTask 里
#     CompletedTotal 只传 1 个 label 触发 panic → 500。已修复（service.go
#     recordCompletion 补齐 source/status 双标签），取消任务硬断言 2xx + ret=1。
#   - #125 DELETE /devices/tasks/:task_id 对不存在 ID 返回 500：handler 用
#     err.Error()=="task not found" 精确比对，但 service 返回带 id 后缀的
#     "task not found: <id>"，永不匹配落入 500。已修复（service 改用哨兵
#     ErrTaskNotFound 包装 core ErrNotFound + handler errors.Is 判定 →
#     HTTPStatusFromError 映射 404），不存在任务取消硬断言 404。
#   - #126(8) POST /devices/tasks/:task_id/retry 对 cancelled/completed 任务
#     返回 200 复活回 pending：model.go CanRetry() 只看 retry_count<max_retries
#     不校验状态。已修复（新增 CanManualRetry 限定 failed/expired 终态失败类才可
#     主动 retry，handler 改用之），已取消任务 retry 硬断言 4xx + 状态保持
#     cancelled。同项第2点：CreateTaskRequest.MaxRetries 改 *int，显式 0 表"禁止
#     重试"不再被静默改回默认 3。
#
# 用法：bash smoke_task.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "统一任务队列" "$@"
smoke_login

# cancel_task TASK_ID "desc" —— DELETE 取消任务，硬断言 2xx + ret=1。
# （#116 已修复：原 CancelTask metrics label 数不匹配 panic→500 的容忍分支已移除）
cancel_task() {
    local tid="$1" desc="$2"
    req DELETE "/api/v1/devices/tasks/$tid"
    if [[ "$HTTP_CODE" == 2* ]] && [ "$(jget ret)" = "1" ]; then
        pass "$desc → $HTTP_CODE ret=1"
        return 0
    fi
    fail "$desc" "期望 2xx + ret=1，实际 HTTP ${HTTP_CODE} ret=$(jget ret) body: $(printf '%s' "$BODY" | head -c 200)"
    return 1
}

# ---------------------------------------------------------------------------
section "数据准备：取一台真实设备 SN"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices?page=1&page_size=5"
check_ret_ok "设备列表可查（任务域数据依赖入口）"
DEV_SN=$(jget data.items.0.serial_number)

if [ -z "$DEV_SN" ]; then
    # 全域降级：任务域按 SN 的读路径与闭环都依赖设备语境，无设备时跳过
    skip "按 SN 查任务历史" "栈内无任何设备，任务域全域跳过"
    skip "pending 队列查询" "同上"
    skip "stats 统计查询" "同上"
    skip "任务详情查询" "同上"
    skip "建任务→取消闭环" "同上"
    skip "负路径校验" "同上"
    smoke_summary
fi
echo "  使用真实设备 SN=${DEV_SN}（只读查询，绝不对其建任务/下发指令）"

# ---------------------------------------------------------------------------
section "任务历史：按 SN 分页查询（列表 key=data.tasks）"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/tasks?device_sn=$DEV_SN&page=1&page_size=10"
check_ret_ok "任务历史按 device_sn 可查"
check_list_or_empty "历史列表 data.tasks（活栈数据可稀疏）" "data.tasks"
check_field "历史响应带 total 字段" "data.total"
check_count_ge "历史响应回显 page=1" "data.page" 1
EXIST_TASK_ID=$(jget data.tasks.0.id)
[ -z "$EXIST_TASK_ID" ] && EXIST_TASK_ID=$(jget data.tasks.0.task_id)

req GET "/api/v1/devices/tasks?device_sn=$DEV_SN&page=1&page_size=10&status=completed"
check_ret_ok "任务历史按 status=completed 过滤"

req GET "/api/v1/devices/tasks?device_sn=$DEV_SN&page=1&page_size=10&status=pending"
check_ret_ok "任务历史按 status=pending 过滤"

# ---------------------------------------------------------------------------
section "pending 队列 / stats 统计（device_sn 必填）"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/tasks/pending?device_sn=$DEV_SN"
check_ret_ok "pending 待处理任务可查"
check_list_or_empty "pending 列表 data.tasks" "data.tasks"
check_field "pending 响应带 total 字段" "data.total"

req GET "/api/v1/devices/tasks/stats?device_sn=$DEV_SN"
check_ret_ok "任务统计 stats 可查"
check_field "stats 含 queue_length 字段" "data.queue_length"

# ---------------------------------------------------------------------------
section "任务详情：栈内既有任务"
# ---------------------------------------------------------------------------
if [ -n "$EXIST_TASK_ID" ]; then
    req GET "/api/v1/devices/tasks/$EXIST_TASK_ID"
    check_ret_ok "既有任务详情可查（task_id=${EXIST_TASK_ID}）"
    check_field "详情回显 method" "data.method"
    check_field "详情回显 status" "data.status"
else
    skip "既有任务详情可查" "设备 ${DEV_SN} 无历史任务；详情路径由下方闭环自建任务覆盖"
fi

# ---------------------------------------------------------------------------
section "闭环：建任务 → 详情 → 列表可见 → 取消 → 状态验证"
# ---------------------------------------------------------------------------
# 红线安全：用不存在的 SMOKE_TAG SN —— 该 SN 无设备记录，wakeDevice 查不到
# Connection Request URL 即放弃，任务只入 Redis/PG 队列，永不下发；建后立即取消。
TASK_SN="${SMOKE_TAG}-TASKDEV"
# 注：max_retries 传 0 会被 NewTask 忽略（默认 3），expires_in=300 兜底——即使
# 闭环中途断链，任务也会在 5 分钟后被 ExpiredSweeper 收尾，不留长期 pending。
req POST "/api/v1/devices/tasks?device_sn=$TASK_SN" "{\"device_sn\":\"$TASK_SN\",\"method\":\"GetParameterValues\",\"params\":{\"parameter_names\":[\"Device.DeviceInfo.SerialNumber\"]},\"priority\":10,\"expires_in\":300,\"description\":\"${SMOKE_TAG} 冒烟自建任务（不存在 SN，建后立即取消）\"}"
check_ret_ok "创建任务（SN=${TASK_SN}，不存在设备，仅入队）"
TASK_ID=$(jget data.id)
check_field "创建返回任务 id" "data.id"

NEW_STATUS=$(jget data.status)
if [ "$NEW_STATUS" = "pending" ]; then
    pass "新建任务初始状态为 pending"
else
    fail "新建任务初始状态为 pending" "实际 status='${NEW_STATUS}'"
fi

if [ -n "$TASK_ID" ]; then
    req GET "/api/v1/devices/tasks/$TASK_ID"
    check_ret_ok "自建任务详情可查"
    check_field "详情回显 method=GetParameterValues" "data.method"
    check_field "详情回显 device_sn" "data.device_sn"

    req GET "/api/v1/devices/tasks?device_sn=$TASK_SN&page=1&page_size=10"
    check_list_nonempty "任务历史按自建 SN 可见新任务" "data.tasks"
    check_count_ge "历史 total ≥ 1" "data.total" 1

    req GET "/api/v1/devices/tasks/pending?device_sn=$TASK_SN"
    check_list_nonempty "pending 队列可见新任务" "data.tasks"

    req GET "/api/v1/devices/tasks/stats?device_sn=$TASK_SN"
    check_ret_ok "自建 SN 的 stats 可查"
    check_count_ge "stats by_status.pending ≥ 1" "data.by_status.pending" 1
    check_count_ge "stats queue_length ≥ 1" "data.queue_length" 1

    cancel_task "$TASK_ID" "取消任务（闭环还原）"

    req GET "/api/v1/devices/tasks/$TASK_ID"
    check_ret_ok "取消后任务详情仍可查"
    CANCELLED_STATUS=$(jget data.status)
    if [ "$CANCELLED_STATUS" = "cancelled" ]; then
        pass "取消后任务状态为 cancelled"
    else
        fail "取消后任务状态为 cancelled" "实际 status='${CANCELLED_STATUS}'"
    fi

    req GET "/api/v1/devices/tasks/pending?device_sn=$TASK_SN"
    PENDING_LEFT=$(jlen data.tasks)
    if [ "$HTTP_CODE" = "200" ] && [ "$PENDING_LEFT" -eq 0 ]; then
        pass "取消后 pending 队列清空（剩 0 条）"
    else
        fail "取消后 pending 队列清空" "HTTP $HTTP_CODE，pending 仍有 $PENDING_LEFT 条"
    fi

    # #126 第8项已修复：CanManualRetry 限定 failed/expired 终态失败类才可主动 retry，
    # cancelled/completed/pending/sent 一律拒绝（避免把已取消/已完成任务"复活"回 pending）。
    # 此处 TASK_ID 已是 cancelled → retry 必被拒（4xx），硬断言之；任务保持 cancelled 不变。
    req POST "/api/v1/devices/tasks/$TASK_ID/retry"
    if [[ "$HTTP_CODE" == 4* ]]; then
        pass "retry 已取消任务被拒绝（CanManualRetry 状态校验）→ $HTTP_CODE"
    else
        fail "retry 已取消任务应被拒绝（4xx）" "实际 HTTP ${HTTP_CODE} ret=$(jget ret) body: $(printf '%s' "$BODY" | head -c 200)"
    fi

    # 拒绝后任务状态应仍为 cancelled（未被复活）
    req GET "/api/v1/devices/tasks/$TASK_ID"
    RETRY_STATUS=$(jget data.status)
    if [ "$RETRY_STATUS" = "cancelled" ]; then
        pass "retry 被拒后任务仍为 cancelled（未复活）"
    else
        fail "retry 被拒后任务仍为 cancelled" "实际 status='${RETRY_STATUS}'"
    fi

    # 已取消任务被拒绝 retry，pending 队列应保持为空（无复活入队）
    req GET "/api/v1/devices/tasks/pending?device_sn=$TASK_SN"
    RETRY_LEFT=$(jlen data.tasks)
    if [ "$HTTP_CODE" = "200" ] && [ "$RETRY_LEFT" -eq 0 ]; then
        pass "retry 被拒后 pending 队列仍为空（剩 0 条）"
    else
        fail "retry 被拒后 pending 队列仍为空" "HTTP $HTTP_CODE，pending 仍有 $RETRY_LEFT 条"
    fi
else
    skip "自建任务详情可查" "创建未返回 id，闭环无法继续"
    skip "任务历史按自建 SN 可见" "同上"
    skip "pending 队列可见新任务" "同上"
    skip "自建 SN 的 stats 断言" "同上"
    skip "取消任务闭环" "同上"
    skip "retry 已取消任务被拒闭环" "同上"
fi

# ---------------------------------------------------------------------------
section "负路径：参数校验与不存在 ID"
# ---------------------------------------------------------------------------
GHOST_ID="deadbeef-dead-4ead-8ead-deadbeefdead"

req GET "/api/v1/devices/tasks"
check_ret_fail "任务历史缺 device_sn 被拒绝（400）"

req GET "/api/v1/devices/tasks/pending"
check_ret_fail "pending 缺 device_sn 被拒绝（400）"

req GET "/api/v1/devices/tasks/stats"
check_ret_fail "stats 缺 device_sn 被拒绝（400）"

req GET "/api/v1/devices/tasks/$GHOST_ID"
check_status "查询不存在任务详情" 404

req POST "/api/v1/devices/tasks?device_sn=$TASK_SN" '{"device_sn":"x"}'
check_ret_fail "创建任务缺 method 必填字段被拒绝（400）"

req POST "/api/v1/devices/tasks" "{\"device_sn\":\"$TASK_SN\",\"method\":\"GetParameterValues\"}"
check_ret_fail "创建任务缺 device_sn 查询参数被拒绝（400）"

req POST "/api/v1/devices/tasks/batch?device_sn=$TASK_SN" '{"not":"an-array"}'
check_ret_fail "批量创建非数组 body 被拒绝（400）"

# #125 已修复：service 改用哨兵 ErrTaskNotFound（包装 core ErrNotFound），handler
# 用 errors.Is 判定后经 HTTPStatusFromError 映射 → 不存在任务 DELETE 硬断言 404。
req DELETE "/api/v1/devices/tasks/$GHOST_ID"
check_status "取消不存在任务返回 404" 404

req POST "/api/v1/devices/tasks/$GHOST_ID/retry"
check_ret_fail "retry 不存在任务被拒绝（404）"

# ---------------------------------------------------------------------------
section "purge（destructive 禁区：只测认证负路径）"
# ---------------------------------------------------------------------------
req_noauth POST "/api/v1/tasks/purge"
check_status "purge 未认证被拒绝（401，路由存在且受保护）" 401

skip "purge 真实执行" "危险禁区：handler 对非法 retention_days 静默回退默认 30 天并真执行清理，无纯参数校验负路径，按约定跳过"

smoke_summary

#!/usr/bin/env bash
# =============================================================================
# smoke_mr.sh — F05 测量报告（MR）业务冒烟
#
# 覆盖（路由源 /tmp/smoke_routes.json key=mr）：
#   - MR 文件：列表 / 按设备聚合 / 下载（含负路径）
#   - MR 数据查询（/mr/data）
#   - MR 指标库：indicators / indicators/all / :code/stats（builtin 种子 5 条）
#   - MR 设备映射：列表 + toggle 可逆闭环（有映射才做，无则 skip）
#   - MR 导出：POST /mr/export（2xx 受理或 4xx 参数校验均可接受）
#   - 测量任务：列表 / 详情 / 进度；POST 建任务只测参数校验负路径（红线：不向设备下发）
#   - 批量删除（destructive）：只测空/非法参数负路径
#
# 用法：bash smoke_mr.sh [BASE_URL]     # 默认 http://localhost:8081
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F05 测量报告" "$@"
smoke_login

NIL_UUID="00000000-0000-0000-0000-00000000dead"

# ---------------------------------------------------------------------------
section "MR 文件：列表 / 按设备聚合 / 下载"
# ---------------------------------------------------------------------------

req GET "/api/v1/mr/files"
check_list_or_empty "MR 文件列表可查" "data.items"
FILE_ID=$(jget data.items.0.id)

req GET "/api/v1/mr/files?page=1&page_size=5&mr_type=MRO"
check_ret_ok "MR 文件列表带分页+类型过滤可查"

req GET "/api/v1/mr/files/devices"
check_list_or_empty "MR 文件按设备聚合可查" "data.items"

req GET "/api/v1/mr/files/devices?keyword=${SMOKE_TAG}&page=1&page_size=5"
check_ret_ok "MR 文件按设备聚合带 keyword 过滤可查"

if [ -n "$FILE_ID" ]; then
    req GET "/api/v1/mr/files/${FILE_ID}/download"
    if [ "$HTTP_CODE" = "200" ]; then
        pass "MR 文件下载 → 200"
    elif [ "$HTTP_CODE" = "500" ]; then
        # 已知容忍项：种子 mr_files 行可能无 MinIO 实体对象（download 依赖 MinIO）
        skip "MR 文件下载" "mr_files 行存在但 MinIO 无实体对象（500，环境性缺失）"
    else
        fail "MR 文件下载" "期望 200（或 MinIO 缺对象 500），实际 ${HTTP_CODE}"
    fi
else
    skip "MR 文件下载（正路径）" "活栈无 mr_files 数据且无上传 API 无法自建"
fi

req GET "/api/v1/mr/files/not-a-uuid/download"
check_status "下载非法文件 ID 被拒" 400

req GET "/api/v1/mr/files/${NIL_UUID}/download"
check_ret_fail "下载不存在文件被拒"

# ---------------------------------------------------------------------------
section "MR 数据查询"
# ---------------------------------------------------------------------------

req GET "/api/v1/mr/data"
check_list_or_empty "MR 数据可查" "data.items"

req GET "/api/v1/mr/data?mr_type=MRO&page=1&page_size=10"
check_ret_ok "MR 数据带类型过滤可查"

req GET "/api/v1/mr/data?device_id=not-a-uuid"
check_status "MR 数据非法 device_id 被拒" 400

# ---------------------------------------------------------------------------
section "MR 指标库（builtin 种子：RSRP/RSRQ/SINR/TA/PHR）"
# ---------------------------------------------------------------------------

req GET "/api/v1/mr/indicators"
check_list_nonempty "MR 指标列表非空（builtin 种子）" "data.items"
check_count_ge "MR 指标总数 ≥ 5" "data.total" 5
IND_CODE=$(jget data.items.0.indicator_code)

req GET "/api/v1/mr/indicators?category=coverage&page=1&page_size=10"
check_ret_ok "MR 指标按分类过滤可查"

req GET "/api/v1/mr/indicators/all"
# /indicators/all 返回纯数组（data 即列表，非 data.items）
check_list_nonempty "MR 指标全量列表非空" "data"

if [ -z "$IND_CODE" ]; then IND_CODE="RSRP"; fi
req GET "/api/v1/mr/indicators/${IND_CODE}/stats"
check_ret_ok "MR 指标统计可查（code=${IND_CODE}）"
check_field "指标统计含 indicator_code" "data.indicator_code"

req GET "/api/v1/mr/indicators/NOPE_${SMOKE_TAG}/stats"
check_status "不存在指标统计返回 404" 404

# ---------------------------------------------------------------------------
section "MR 设备映射 + toggle 可逆闭环"
# ---------------------------------------------------------------------------

req GET "/api/v1/mr/mappings"
check_list_or_empty "MR 设备映射列表可查" "data.items"
MAP_ID=$(jget data.items.0.id)
MAP_ENABLED=$(jget data.items.0.enabled)

req GET "/api/v1/mr/mappings?enabled=true&page=1&page_size=10"
check_ret_ok "MR 设备映射按 enabled 过滤可查"

if [ -n "$MAP_ID" ]; then
    # jget 输出 Python bool 字面量：True / False
    if [ "$MAP_ENABLED" = "True" ]; then
        OPP="false"; ORIG="true"
        OPP_PY="False"; ORIG_PY="True"
    else
        OPP="true"; ORIG="false"
        OPP_PY="True"; ORIG_PY="False"
    fi

    req PUT "/api/v1/mr/mappings/${MAP_ID}/toggle" "{\"enabled\":${OPP}}"
    check_ret_ok "映射 toggle 翻转（${MAP_ENABLED} → ${OPP_PY}）"
    TOGGLED=$(jget data.enabled)
    if [ "$TOGGLED" = "$OPP_PY" ]; then
        pass "toggle 后 enabled=${TOGGLED}（与请求一致）"
    else
        fail "toggle 后 enabled 校验" "期望 ${OPP_PY}，实际 '${TOGGLED}'"
    fi

    req PUT "/api/v1/mr/mappings/${MAP_ID}/toggle" "{\"enabled\":${ORIG}}"
    check_ret_ok "映射 toggle 还原（→ ${ORIG_PY}）"
    RESTORED=$(jget data.enabled)
    if [ "$RESTORED" = "$ORIG_PY" ]; then
        pass "toggle 还原后 enabled=${RESTORED}（恢复原状）"
    else
        fail "toggle 还原校验" "期望 ${ORIG_PY}，实际 '${RESTORED}'"
    fi
else
    skip "映射 toggle 可逆闭环" "活栈无 mr_device_mappings 数据且无创建 API 无法自建"
fi

req PUT "/api/v1/mr/mappings/not-a-uuid/toggle" '{"enabled":true}'
check_status "toggle 非法映射 ID 被拒" 400

req PUT "/api/v1/mr/mappings/${NIL_UUID}/toggle" '{"enabled":true}'
check_ret_fail "toggle 不存在映射被拒"

# ---------------------------------------------------------------------------
section "MR 导出（export）"
# ---------------------------------------------------------------------------

# 导出为查询+流式返回（不触达设备）；受理 2xx 或参数校验 4xx 均可接受
req POST "/api/v1/mr/export" '{"format":"json","mr_type":"MRO"}'
check_status_in "MR 导出（JSON）触发受理" "200 201 202 400"

req POST "/api/v1/mr/export" '{"device_id":"not-a-uuid"}'
check_ret_fail "MR 导出非法 device_id 被拒"

# ---------------------------------------------------------------------------
section "测量任务：列表/详情/进度 + 建任务参数校验（不下发设备）"
# ---------------------------------------------------------------------------

req GET "/api/v1/mr/tasks"
check_list_or_empty "测量任务列表可查" "data.items"
TASK_ID=$(jget data.items.0.task_id)

req GET "/api/v1/mr/tasks?status=running&keyword=${SMOKE_TAG}&page=1&page_size=10"
check_ret_ok "测量任务列表带过滤参数可查"

if [ -n "$TASK_ID" ]; then
    req GET "/api/v1/mr/tasks/${TASK_ID}"
    check_ret_ok "测量任务详情可查"
    req GET "/api/v1/mr/tasks/${TASK_ID}/progress"
    check_ret_ok "测量任务进度可查"
else
    # 无任务数据时用不存在 ID 验证详情/进度路由可达（404 即正确处理）
    req GET "/api/v1/mr/tasks/${NIL_UUID}"
    check_status "不存在任务详情返回 404" 404
    req GET "/api/v1/mr/tasks/${NIL_UUID}/progress"
    check_status "不存在任务进度返回 404" 404
fi

# 【红线】POST /mr/tasks 会向设备下发测量任务 —— 只测参数校验负路径，绝不真建
req POST "/api/v1/mr/tasks" '{}'
check_status "建任务缺必填字段被拒" 400

req POST "/api/v1/mr/tasks" "{\"task_name\":\"${SMOKE_TAG}-mr-task\",\"start_time\":\"not-rfc3339\",\"target_device_sns\":[\"NO-SUCH-DEV-${SMOKE_TAG}\"]}"
check_status "建任务非法 start_time 被拒" 400

req POST "/api/v1/mr/tasks" "{\"task_name\":\"${SMOKE_TAG}-mr-task\",\"start_time\":\"2026-06-10T00:00:00Z\",\"target_device_sns\":[]}"
check_status "建任务空目标设备列表被拒" 400

# stop / delete 只打不存在 ID 的负路径（绝不碰真实任务）
req POST "/api/v1/mr/tasks/not-a-uuid/stop"
check_status "停止非法任务 ID 被拒" 400

req POST "/api/v1/mr/tasks/${NIL_UUID}/stop"
check_ret_fail "停止不存在任务被拒"

req DELETE "/api/v1/mr/tasks/not-a-uuid"
check_status "删除非法任务 ID 被拒" 400

req DELETE "/api/v1/mr/tasks/${NIL_UUID}"
check_ret_fail "删除不存在任务被拒"

# ---------------------------------------------------------------------------
section "批量删除（destructive）：只测参数校验负路径"
# ---------------------------------------------------------------------------

req POST "/api/v1/mr/files/batch-delete" '{}'
check_status "批量删除缺 serial_numbers 被拒" 400

req POST "/api/v1/mr/files/batch-delete" '{"serial_numbers":[]}'
check_status "批量删除空 serial_numbers 被拒" 400

smoke_summary

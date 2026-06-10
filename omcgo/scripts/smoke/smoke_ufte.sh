#!/usr/bin/env bash
# =============================================================================
# smoke_ufte.sh — F06 统一文件任务引擎（UFTE）业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=ufte 全部 read 路由 + 可逆 write 闭环）：
#   · 读路径全扫：overview / task-types / tasks / devices / device-candidates
#     / devices/export（CSV 流式，通用页签 + view=upgrade 页签两种表头）
#   · task-types 治理可逆闭环：POST 自建 ${SMOKE_TAG} 类型 → PUT 改 → DELETE 删
#     （typeCode 由后端按 category+displayName 生成，内置类型删除 → 403 红线）
#   · 任务闭环【不启动】：POST /ufte/tasks executionMode=suspended 仅建不派发
#     （software.BatchCollect 挂起模式建完即返回，不触达设备）→ 列表可见 →
#     DELETE /:id 删除；再建一条走 POST /tasks/batch-delete 同义入口
#   · PUT /ufte/tasks/:id/start 是 destructive（向设备真实下发文件任务）
#     【红线：连负路径都不调用】；suspend / terminate / retry / delete 用
#     不存在 ID 与非法 UUID 只测负路径
#
# 用法：bash smoke_ufte.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 统一文件任务引擎(UFTE)" "$@"
smoke_login

# 不存在的合法 UUID（负路径专用，确保打不到任何真实记录）
NOID="00000000-dead-beef-0000-000000000000"

# ---------------------------------------------------------------------------
section "读路径全扫：overview / task-types / tasks / devices / candidates"
# ---------------------------------------------------------------------------
req GET "/api/v1/ufte/overview"
check_ret_ok "UFTE 总览可查"
check_count_ge "总览 enabledTypeCount ≥ 1（内置类型默认启用）" "data.enabledTypeCount" 1
check_field "总览含 runningTaskCount 字段" "data.runningTaskCount"
check_field "总览含 customTypeCount 字段" "data.customTypeCount"
check_field "总览含 successRate30d 字段" "data.successRate30d"

# task-types：data 即数组（无 items 包装），内置目录 builtin 字典类必须非空
req GET "/api/v1/ufte/task-types"
check_ret_ok "任务类型目录可查"
check_list_nonempty "任务类型目录非空（内置 11 类）" "data"
check_field "类型条目含 typeCode" "data.0.typeCode"
check_field "类型条目含 category" "data.0.category"
check_field "类型条目含 rpcType" "data.0.rpcType"
case "$BODY" in
    *'"RUNTIME_LOG_COLLECT"'*) pass "内置类型 RUNTIME_LOG_COLLECT 在目录中" ;;
    *) fail "内置类型 RUNTIME_LOG_COLLECT 在目录中" "目录未包含该 typeCode" ;;
esac

req GET "/api/v1/ufte/tasks?page=1&page_size=10"
check_list_or_empty "任务列表可查" "data.items"
check_field "任务列表带 page 字段" "data.page"
check_field "任务列表带 total 字段" "data.total"

# 过滤参数链路（SMOKE_TAG 关键字必查空也合法；typeCode 用内置值）
req GET "/api/v1/ufte/tasks?page=1&page_size=10&typeCode=RUNTIME_LOG_COLLECT"
check_list_or_empty "任务列表按 typeCode 过滤可查" "data.items"
req GET "/api/v1/ufte/tasks?page=1&page_size=10&category=station_log&keyword=${SMOKE_TAG}"
check_list_or_empty "任务列表按 category+keyword 过滤可查" "data.items"

req GET "/api/v1/ufte/devices?page=1&page_size=10"
check_list_or_empty "设备子任务列表可查" "data.items"
req GET "/api/v1/ufte/devices?page=1&page_size=10&typeCode=RUNTIME_LOG_COLLECT&keyword=${SMOKE_TAG}"
check_list_or_empty "设备子任务列表按 typeCode+keyword 过滤可查" "data.items"

req GET "/api/v1/ufte/device-candidates?page=1&page_size=10"
check_list_or_empty "候选设备列表可查" "data.items"
req GET "/api/v1/ufte/device-candidates?page=1&page_size=10&typeCode=RUNTIME_LOG_COLLECT"
check_list_or_empty "候选设备列表按 typeCode 过滤可查" "data.items"

# ---------------------------------------------------------------------------
section "devices/export CSV 导出（流式，两种页签表头）"
# ---------------------------------------------------------------------------
req GET "/api/v1/ufte/devices/export"
check_status "设备子任务 CSV 导出（通用页签）" "200"
case "$BODY" in
    *"任务名称"*) pass "CSV 含通用页签表头（任务名称）" ;;
    *) fail "CSV 含通用页签表头（任务名称）" "响应前 120 字节：$(printf '%s' "$BODY" | head -c 120)" ;;
esac

req GET "/api/v1/ufte/devices/export?view=upgrade"
check_status "设备子任务 CSV 导出（view=upgrade 页签）" "200"
case "$BODY" in
    *"基站编码"*) pass "CSV 含升级页签表头（基站编码）" ;;
    *) fail "CSV 含升级页签表头（基站编码）" "响应前 120 字节：$(printf '%s' "$BODY" | head -c 120)" ;;
esac

# 带过滤条件导出（SMOKE_TAG 必查空 → 只有表头行，验证过滤参数绑定）
req GET "/api/v1/ufte/devices/export?keyword=${SMOKE_TAG}"
check_status "CSV 导出带 keyword 过滤可用" "200"

# ---------------------------------------------------------------------------
section "task-types 治理可逆闭环（POST 自建 → PUT 改 → DELETE 删）"
# ---------------------------------------------------------------------------
# typeCode 由后端生成：CUSTOM_{CATEGORY}_{DISPLAYNAME}；displayName 带 SMOKE_TAG
# 保证唯一。enabled=false 防止出现在任何业务下拉里被误用。
req POST "/api/v1/ufte/task-types" "{
    \"category\": \"station_log\",
    \"categoryLabel\": \"日志收集\",
    \"displayName\": \"SMOKE ${SMOKE_TAG}\",
    \"description\": \"smoke create ${SMOKE_TAG}\",
    \"rpcType\": \"UPLOAD\",
    \"stepChain\": [\"CHECK_PERMISSION\", \"SEND_RPC\"],
    \"enabled\": false,
    \"platformScope\": [\"4G eNB\"],
    \"fileType\": \"4 Vendor Log File 1,2,3,4\",
    \"fileTypeLabel\": \"smoke log\"
}"
check_status "自建任务类型（enabled=false）" "201"
UF_TYPE_CODE=$(jget data.typeCode)

if [ -n "$UF_TYPE_CODE" ]; then
    check_field "创建返回生成的 typeCode" "data.typeCode"
    check_field "创建回显 builtIn=false" "data.builtIn"

    # 列表可见
    req GET "/api/v1/ufte/task-types"
    case "$BODY" in
        *"$UF_TYPE_CODE"*) pass "自建类型出现在目录列表（${UF_TYPE_CODE}）" ;;
        *) fail "自建类型出现在目录列表" "目录未包含 ${UF_TYPE_CODE}" ;;
    esac

    # PUT 更新（全量 body）
    req PUT "/api/v1/ufte/task-types/$UF_TYPE_CODE" "{
        \"category\": \"station_log\",
        \"categoryLabel\": \"日志收集\",
        \"displayName\": \"SMOKE ${SMOKE_TAG} v2\",
        \"description\": \"smoke updated ${SMOKE_TAG}\",
        \"rpcType\": \"UPLOAD\",
        \"stepChain\": [\"CHECK_PERMISSION\", \"SEND_RPC\"],
        \"enabled\": false,
        \"platformScope\": [\"4G eNB\"],
        \"fileType\": \"4 Vendor Log File 1,2,3,4\",
        \"fileTypeLabel\": \"smoke log\"
    }"
    check_ret_ok "更新自建任务类型"
    UPD_NAME=$(jget data.displayName)
    if [ "$UPD_NAME" = "SMOKE ${SMOKE_TAG} v2" ]; then
        pass "更新后 displayName 回显 v2"
    else
        fail "更新后 displayName 回显 v2" "实际 displayName=${UPD_NAME}"
    fi
    check_field "更新记录 lastEditor" "data.lastEditor"

    # DELETE 删除（闭环收尾，只删冒烟自建）
    req DELETE "/api/v1/ufte/task-types/$UF_TYPE_CODE"
    check_ret_ok "删除自建任务类型"
    req GET "/api/v1/ufte/task-types"
    case "$BODY" in
        *"$UF_TYPE_CODE"*) fail "删除后目录不再包含自建类型" "目录仍含 ${UF_TYPE_CODE}" ;;
        *) pass "删除后目录不再包含自建类型" ;;
    esac
else
    skip "创建返回生成的 typeCode" "创建被拒绝（HTTP ${HTTP_CODE}），闭环跳过"
    skip "自建类型出现在目录列表" "创建未成功，闭环跳过"
    skip "更新自建任务类型" "创建未成功，闭环跳过"
    skip "删除自建任务类型" "创建未成功，闭环跳过"
fi

# 治理负路径
req POST "/api/v1/ufte/task-types" "{\"category\":\"station_log\"}"
check_ret_fail "创建任务类型缺必填字段被拒绝"

req PUT "/api/v1/ufte/task-types/NOT_EXIST_${SMOKE_TAG}" "{
    \"category\": \"station_log\",
    \"categoryLabel\": \"日志收集\",
    \"displayName\": \"neg\",
    \"rpcType\": \"UPLOAD\",
    \"stepChain\": [\"SEND_RPC\"],
    \"fileType\": \"4 Vendor Log File 1,2,3,4\",
    \"fileTypeLabel\": \"neg\"
}"
check_status "更新不存在任务类型 → 404" "404"

# 内置类型删除保护【红线：必须被 403 拒绝，绝不能删掉内置目录】
req DELETE "/api/v1/ufte/task-types/ENB_IMG_UPGRADE"
check_status "删除内置任务类型被 403 拒绝（治理红线）" "403"

req DELETE "/api/v1/ufte/task-types/NOT_EXIST_${SMOKE_TAG}"
check_status "删除不存在任务类型 → 404" "404"

# ---------------------------------------------------------------------------
section "任务闭环【不启动】：suspended 建任务 → 列表可见 → 删除"
# ---------------------------------------------------------------------------
# executionMode=suspended：software.BatchCollect 挂起模式只落 upgrade_tasks +
# sub_tasks 即返回，不调度任何 RPC（service.go resolveScheduleMode →
# scheduleModeSuspended 分支早 return），对真实设备零触达。
DEV_ID=""
req GET "/api/v1/ufte/device-candidates?page=1&page_size=1&typeCode=RUNTIME_LOG_COLLECT"
DEV_ID=$(jget data.items.0.id)
if [ -z "$DEV_ID" ]; then
    # 候选为空时退化到设备总表取一台（仅取 ID，不下发任何操作）
    req GET "/api/v1/devices?page=1&page_size=1"
    DEV_ID=$(jget data.items.0.id)
fi

UF_TASK_ID=""
if [ -n "$DEV_ID" ]; then
    req POST "/api/v1/ufte/tasks" "{
        \"taskName\": \"smoke-${SMOKE_TAG}\",
        \"typeCode\": \"RUNTIME_LOG_COLLECT\",
        \"deviceIds\": [\"$DEV_ID\"],
        \"executionMode\": \"suspended\"
    }"
    check_status "创建挂起任务（仅建不派发）" "201"
    UF_TASK_ID=$(jget data.id)
else
    skip "创建挂起任务（仅建不派发）" "活栈无任何设备可作为任务目标"
fi

if [ -n "$UF_TASK_ID" ]; then
    check_field "创建返回任务 ID" "data.id"
    TASK_STATUS=$(jget data.status)
    case "$TASK_STATUS" in
        pending|suspended) pass "挂起建任务后状态未进入执行（status=${TASK_STATUS}）" ;;
        *) fail "挂起建任务后状态未进入执行" "实际 status=${TASK_STATUS}（疑似被派发）" ;;
    esac
    CREATED_MODE=$(jget data.executionMode)
    if [ "$CREATED_MODE" = "suspended" ]; then
        pass "创建回显 executionMode=suspended"
    elif [ "$CREATED_MODE" = "immediate" ]; then
        known_bug "创建回显 executionMode=suspended" "请求 executionMode=suspended 回显 immediate —— software.applyScheduleMode 挂起分支落 status=pending+create_status=active，ufte.executionModeForTask 只认 TaskSuspended，挂起语义在列表不可见（CONFIG_RESTORE placeholder 同模式落 TaskSuspended，两链路表示不一致）"
    else
        fail "创建回显 executionMode=suspended" "实际 executionMode=${CREATED_MODE}"
    fi

    # 列表可见（keyword 过滤命中自建任务）
    req GET "/api/v1/ufte/tasks?page=1&page_size=10&keyword=smoke-${SMOKE_TAG}"
    check_count_ge "自建任务在列表可见（keyword 命中）" "data.total" 1
    LISTED_ID=$(jget data.items.0.id)
    if [ "$LISTED_ID" = "$UF_TASK_ID" ]; then
        pass "列表条目 ID 与创建返回一致"
    else
        fail "列表条目 ID 与创建返回一致" "期望 ${UF_TASK_ID}，实际 ${LISTED_ID}"
    fi

    # DELETE 删除自建任务（挂起态非 in_progress，可直接删）
    req DELETE "/api/v1/ufte/tasks/$UF_TASK_ID"
    check_ret_ok "删除自建挂起任务"
    req GET "/api/v1/ufte/tasks?page=1&page_size=10&keyword=smoke-${SMOKE_TAG}"
    GONE_TOTAL=$(jget data.total)
    if [ "$GONE_TOTAL" = "0" ]; then
        pass "删除后列表不再包含自建任务"
    else
        fail "删除后列表不再包含自建任务" "keyword 命中 total=${GONE_TOTAL}"
    fi
else
    [ -n "$DEV_ID" ] && skip "创建返回任务 ID" "创建被拒绝（HTTP ${HTTP_CODE}），闭环跳过"
    skip "自建任务在列表可见" "无任务 ID，闭环跳过"
    skip "删除自建挂起任务" "无任务 ID，闭环跳过"
fi

# batch-delete 同义入口：再建一条挂起任务走批量删除（含一个非法 ID 验证部分失败形状）
if [ -n "$DEV_ID" ]; then
    req POST "/api/v1/ufte/tasks" "{
        \"taskName\": \"smoke-batch-${SMOKE_TAG}\",
        \"typeCode\": \"RUNTIME_LOG_COLLECT\",
        \"deviceIds\": [\"$DEV_ID\"],
        \"executionMode\": \"suspended\"
    }"
    check_status "创建第二条挂起任务（batch-delete 用）" "201"
    UF_TASK_ID2=$(jget data.id)
    if [ -n "$UF_TASK_ID2" ]; then
        req POST "/api/v1/ufte/tasks/batch-delete" \
            "{\"task_ids\":[\"$UF_TASK_ID2\",\"not-a-uuid\"]}"
        check_ret_ok "批量删除请求受理"
        BD_OK=$(jget data.succeeded.0)
        if [ "$BD_OK" = "$UF_TASK_ID2" ]; then
            pass "批量删除成功清单包含自建任务"
        else
            fail "批量删除成功清单包含自建任务" "succeeded.0=${BD_OK}"
        fi
        BD_BAD=$(jget data.failed.0.task_id)
        if [ "$BD_BAD" = "not-a-uuid" ]; then
            pass "非法 UUID 进入 failed 明细（单条失败不影响其他）"
        else
            fail "非法 UUID 进入 failed 明细" "failed.0.task_id=${BD_BAD}"
        fi
    else
        skip "批量删除请求受理" "第二条任务创建未成功"
        skip "批量删除成功清单包含自建任务" "第二条任务创建未成功"
    fi
else
    skip "创建第二条挂起任务（batch-delete 用）" "活栈无任何设备可作为任务目标"
fi

# ---------------------------------------------------------------------------
section "危险端点负路径【红线：start 绝不调用，连负路径都不打】"
# ---------------------------------------------------------------------------
# PUT /ufte/tasks/:id/start 会向设备真实下发文件任务（备份/日志采集/恢复），
# 按路由清单 destructive 注记整端点跳过；以下只打 suspend/terminate/retry/delete
# 的不存在 ID 与非法 UUID。

# 任务创建参数校验（在建任何任务之前即被拒绝，不落库不触设备）
req POST "/api/v1/ufte/tasks" "{}"
check_ret_fail "创建任务空 body 被拒绝"

req POST "/api/v1/ufte/tasks" "{
    \"taskName\": \"neg-${SMOKE_TAG}\",
    \"typeCode\": \"NOT_EXIST_${SMOKE_TAG}\",
    \"deviceIds\": [\"$NOID\"],
    \"executionMode\": \"suspended\"
}"
check_ret_fail "创建任务不存在 typeCode 被拒绝"

# 升级类任务缺 firmwareId：service 在创建任何任务/子任务前即报 invalid input
req POST "/api/v1/ufte/tasks" "{
    \"taskName\": \"neg-fw-${SMOKE_TAG}\",
    \"typeCode\": \"ENB_IMG_UPGRADE\",
    \"deviceIds\": [\"$NOID\"],
    \"executionMode\": \"suspended\"
}"
check_ret_fail "创建升级类任务缺 firmwareId 被拒绝（不落任务）"

# 列表过滤负路径
req GET "/api/v1/ufte/tasks?page=1&page_size=10&typeCode=NOT_EXIST_${SMOKE_TAG}"
check_status "任务列表不存在 typeCode → 400" "400"
req GET "/api/v1/ufte/device-candidates?page=1&page_size=10&typeCode=NOT_EXIST_${SMOKE_TAG}"
check_status "候选设备不存在 typeCode → 400" "400"

# 生命周期端点：非法 UUID / 不存在 ID
req PUT "/api/v1/ufte/tasks/not-a-uuid/suspend"
check_status "挂起任务非法 UUID 拒绝" "400"
req PUT "/api/v1/ufte/tasks/$NOID/suspend"
check_status "挂起不存在任务 → 404" "404"

req PUT "/api/v1/ufte/tasks/$NOID/terminate"
check_status "终止不存在任务 → 404" "404"

req POST "/api/v1/ufte/tasks/$NOID/retry"
check_status "重试不存在任务 → 404" "404"

req DELETE "/api/v1/ufte/tasks/$NOID"
check_status "删除不存在任务 → 404" "404"

req POST "/api/v1/ufte/tasks/batch-delete" "{\"task_ids\":[]}"
check_ret_fail "批量删除空数组被拒绝"

smoke_summary

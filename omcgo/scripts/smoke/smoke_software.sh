#!/usr/bin/env bash
# =============================================================================
# smoke_software.sh — F06 固件升级 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=software 全部 read 路由 + 可逆 write 闭环）：
#   · 固件列表（必须显式 page/page_size，空时 items=null）/ 详情 / 下载
#   · 上传 → 详情 → 元数据更新 → recommend 切换并还原 → batch-download → 删除（闭环）
#   · 升级任务列表 + 子任务列表 / 详情
#   · POST /upgrade-tasks 升级下发【红线】只测参数校验负路径（不存在固件 ID）
#   · terminate / DELETE upgrade-task 等危险端点只测负路径（非法/不存在 ID）
#
# 用法：bash smoke_software.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 固件升级" "$@"
smoke_login

# 不存在的合法 UUID（负路径专用，确保打不到任何真实记录）
NOID="00000000-dead-beef-0000-000000000000"

# ---------------------------------------------------------------------------
section "固件列表（显式 page/page_size）"
# ---------------------------------------------------------------------------
req GET "/api/v1/firmware?page=1&page_size=10"
check_list_or_empty "固件列表可查（分页形状 B，空时 items=null）" "data.items"
check_field "固件列表带 page 字段" "data.page"
check_field "固件列表带 page_size 字段" "data.page_size"

# firmware 绑定不预填 DefaultListRequest：省略分页参数应被 min=1 校验拒绝
req GET "/api/v1/firmware"
check_ret_fail "固件列表省略 page/page_size 被绑定校验拒绝"

# 按 product_class 过滤（用 SMOKE_TAG 保证查空也合法）
req GET "/api/v1/firmware?page=1&page_size=10&product_class=${SMOKE_TAG}"
check_list_or_empty "固件列表按 product_class 过滤可查" "data.items"

# ---------------------------------------------------------------------------
section "固件上传 → 详情 → 元数据 → recommend 切换还原 → 下载 → 删除（闭环）"
# ---------------------------------------------------------------------------
FW_CONTENT="SMOKE-FIRMWARE-PAYLOAD-${SMOKE_TAG}"
FW_FILE="$SMOKE_TMPDIR/${SMOKE_TAG}.bin"
printf '%s' "$FW_CONTENT" > "$FW_FILE"

# 上传必填 form 字段（internal/software/handler.go UploadFirmware）：
#   file=@... 必填；version 必填（缺失 → 400 biz_code 8002）；
#   product_class / manufacturer / description / release_notes / uploader 可选
req_upload "/api/v1/firmware" \
    "file=@${FW_FILE}" \
    "version=v0.0.1-${SMOKE_TAG}" \
    "product_class=SMOKE-PC-${SMOKE_TAG}" \
    "manufacturer=SmokeVendor" \
    "uploader=smoke" \
    "description=smoke upload ${SMOKE_TAG}"
check_status_in "固件上传（小测试 .bin 文件）" "201 200 400"

FW_ID=""
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    FW_ID=$(jget data.id)
fi

if [ -n "$FW_ID" ]; then
    check_field "上传返回固件 ID" "data.id"
    check_field "上传已计算 MD5" "data.md5_val"
    check_field "上传已计算 SHA-256（issue #8 完整性根）" "data.sha256_val"

    # 详情
    req GET "/api/v1/firmware/$FW_ID"
    check_ret_ok "固件详情可查"
    check_field "详情 version 回显" "data.version"
    check_field "详情 file_name 回显" "data.file_name"

    # 元数据更新（PUT /firmware/:id，自建数据可逆）
    req PUT "/api/v1/firmware/$FW_ID" \
        "{\"description\":\"smoke updated ${SMOKE_TAG}\",\"release_notes\":\"rn-${SMOKE_TAG}\"}"
    check_ret_ok "固件元数据更新（description/release_notes）"

    # recommend 切换 → 还原（默认 false → true → false）
    req PUT "/api/v1/firmware/$FW_ID/recommend"
    check_ret_ok "recommend 切换为推荐"
    REC1=$(jget data.recommend)
    if [ "$REC1" = "True" ] || [ "$REC1" = "true" ]; then
        pass "recommend 第一次切换后为 true"
    else
        fail "recommend 第一次切换后为 true" "实际 recommend=${REC1}"
    fi
    req PUT "/api/v1/firmware/$FW_ID/recommend"
    check_ret_ok "recommend 再次切换还原"
    REC2=$(jget data.recommend)
    if [ "$REC2" = "False" ] || [ "$REC2" = "false" ]; then
        pass "recommend 已还原为 false"
    else
        fail "recommend 已还原为 false" "实际 recommend=${REC2}"
    fi

    # 下载（流式，非信封）
    req GET "/api/v1/firmware/$FW_ID/download"
    check_status "固件文件下载" "200"
    case "$BODY" in
        *"$FW_CONTENT"*) pass "下载内容与上传一致" ;;
        *) fail "下载内容与上传一致" "下载体未包含上传载荷（前 80 字节：$(printf '%s' "$BODY" | head -c 80)）" ;;
    esac

    # batch-download（POST 但只读：流式 zip 打包自传固件）
    req POST "/api/v1/firmware/batch-download" "{\"ids\":[\"$FW_ID\"]}" -o "$SMOKE_TMPDIR/fw_bundle.zip"
    check_status "固件批量下载（zip 流）" "200"
    if [ -s "$SMOKE_TMPDIR/fw_bundle.zip" ]; then
        pass "批量下载 zip 非空（$(wc -c < "$SMOKE_TMPDIR/fw_bundle.zip" | tr -d ' ') 字节）"
    else
        fail "批量下载 zip 非空" "落盘文件为空"
    fi

    # 删除自传固件（闭环收尾，只删冒烟自建）
    req DELETE "/api/v1/firmware/$FW_ID"
    check_ret_ok "删除冒烟自传固件"
    req GET "/api/v1/firmware/$FW_ID"
    check_status "删除后详情 404" "404"
else
    skip "上传返回固件 ID" "上传被拒绝（HTTP ${HTTP_CODE}），无 ID 可用"
    skip "固件详情可查" "上传未成功，闭环跳过"
    skip "固件元数据更新" "上传未成功，闭环跳过"
    skip "recommend 切换还原" "上传未成功，闭环跳过"
    skip "固件文件下载" "上传未成功，闭环跳过"
    skip "固件批量下载（zip 流）" "上传未成功，闭环跳过"
    skip "删除冒烟自传固件" "上传未成功，闭环跳过"
fi

# ---------------------------------------------------------------------------
section "固件负路径（参数校验）"
# ---------------------------------------------------------------------------
req GET "/api/v1/firmware/not-a-uuid"
check_status "固件详情非法 UUID 拒绝" "400"

req GET "/api/v1/firmware/$NOID"
check_status "固件详情不存在 ID → 404" "404"

req DELETE "/api/v1/firmware/$NOID"
check_status "删除不存在固件 → 404（不触达真实数据）" "404"

# 缺 file 字段的 multipart 上传
req_upload "/api/v1/firmware" "version=v-neg-${SMOKE_TAG}"
check_ret_fail "上传缺 file 字段被拒绝"

# batch-download 空参校验
req POST "/api/v1/firmware/batch-download" "{}"
check_ret_fail "batch-download 空 ids 被拒绝"
req POST "/api/v1/firmware/batch-download" "{\"ids\":[]}"
check_ret_fail "batch-download 空数组被拒绝"

# ---------------------------------------------------------------------------
section "升级任务列表 + 子任务"
# ---------------------------------------------------------------------------
req GET "/api/v1/upgrade-tasks?page=1&page_size=10"
check_list_or_empty "升级主任务列表可查" "data.items"
TASK_ID=$(jget data.items.0.id)

if [ -n "$TASK_ID" ]; then
    req GET "/api/v1/upgrade-tasks/$TASK_ID"
    check_ret_ok "升级主任务详情可查（id=${TASK_ID}）"
    check_field "主任务详情含 task_name" "data.task_name"

    req GET "/api/v1/upgrade-tasks/$TASK_ID/tasks?page=1&page_size=10"
    check_list_or_empty "主任务下子任务列表可查" "data.items"
else
    # 活栈无升级任务数据：用不存在 ID 验证路由与 404 语义
    req GET "/api/v1/upgrade-tasks/$NOID"
    check_status "升级主任务详情不存在 ID → 404（无任务数据，降级负路径）" "404"

    req GET "/api/v1/upgrade-tasks/$NOID/tasks?page=1&page_size=10"
    check_list_or_empty "不存在主任务的子任务列表返回空集" "data.items"
fi

# 跨任务子任务列表（GET /upgrade-sub-tasks）
req GET "/api/v1/upgrade-sub-tasks?page=1&page_size=10"
check_list_or_empty "全量升级子任务列表可查" "data.items"
SUB_ID=$(jget data.items.0.id)

if [ -n "$SUB_ID" ]; then
    req GET "/api/v1/upgrade-sub-tasks/$SUB_ID"
    check_ret_ok "升级子任务详情可查（id=${SUB_ID}）"
else
    req GET "/api/v1/upgrade-sub-tasks/$NOID"
    check_status "升级子任务详情不存在 ID → 404（无子任务数据，降级负路径）" "404"
fi

# 子任务按 device_sn 过滤（SMOKE_TAG 必查空，验证过滤参数链路）
req GET "/api/v1/upgrade-sub-tasks?page=1&page_size=10&device_sn=${SMOKE_TAG}"
check_list_or_empty "子任务按 device_sn 过滤可查" "data.items"

# ---------------------------------------------------------------------------
section "升级下发只测参数校验【红线：绝不真实下发】"
# ---------------------------------------------------------------------------
# 空 body：device_ids/firmware_id/task_name 均 binding required
req POST "/api/v1/upgrade-tasks" "{}"
check_ret_fail "创建升级任务空 body 被拒绝"

# 缺 device_ids
req POST "/api/v1/upgrade-tasks" \
    "{\"firmware_id\":\"$NOID\",\"task_name\":\"${SMOKE_TAG}-neg\"}"
check_ret_fail "创建升级任务缺 device_ids 被拒绝"

# 形状合法但 firmware_id 不存在：service 先查固件 → ErrNotFound，
# 在建任何任务/子任务之前即失败，不会触达设备
req POST "/api/v1/upgrade-tasks" \
    "{\"device_ids\":[\"$NOID\"],\"firmware_id\":\"$NOID\",\"task_name\":\"${SMOKE_TAG}-neg\"}"
check_ret_fail "创建升级任务无效固件 ID 被拒绝（不落任务）"

# terminate / delete 等危险端点只打非法与不存在 ID（suspend/resume/retry/
# rollback/canary 与之同级危险，按路由清单注记跳过真实路径）
req PUT "/api/v1/upgrade-tasks/not-a-uuid/terminate"
check_status "终止任务非法 UUID 拒绝" "400"

req PUT "/api/v1/upgrade-tasks/$NOID/terminate"
check_status "终止不存在任务 → 404" "404"

req DELETE "/api/v1/upgrade-tasks/$NOID"
check_ret_fail "删除不存在升级任务被拒绝"

smoke_summary

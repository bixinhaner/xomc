#!/usr/bin/env bash
# =============================================================================
# smoke_filemanager.sh — F06 文件管理 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=filemanager 全部路由）：
#   - GET    /api/v1/files                              文件列表（分页/过滤/搜索）
#   - POST   /api/v1/files                              multipart 上传（闭环自建）
#   - GET    /api/v1/files/:id                          详情
#   - GET    /api/v1/files/:id/download                 下载（断言内容一致）
#   - DELETE /api/v1/files/:id                          删除（闭环收尾）
#   - POST   /api/v1/files/:id/distribute               分发 — 只测校验负路径，绝不真发
#   - POST   /api/v1/pm/files/batch-download            四类批量下载（含 by-id 变体）
#   - POST   /api/v1/mr/files/batch-download
#   - POST   /api/v1/backup/config-snapshots/batch-download
#   - POST   /api/v1/backup/device-licenses/batch-download
#
# 后端契约（internal/filemanager/handler.go + internal/bundle/batch_handler.go）：
#   - 上传 form 字段：file（必填）、file_type、description、device_sn、uploader；成功 201
#   - distribute body：{"device_sns":["..."]}（binding:required，缺失→400；文件不存在→404）
#   - 批量下载 body：{"ids":[...]} 或 {"serial_numbers":[...]}；空列表→400；
#     不存在目标→已先写 200 头再流式解析 → 200 空 zip（容忍 200/400/404）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F06 文件管理" "$@"
smoke_login

NOEXIST_ID="00000000-0000-0000-0000-00000000dead"

# ---------------------------------------------------------------------------
section "files 列表（读路径）"
# ---------------------------------------------------------------------------
req GET "/api/v1/files"
check_ret_ok "文件列表可查"
check_list_or_empty "文件列表 data.items" "data.items"
check_field "文件列表含 total 字段" "data.total"

req GET "/api/v1/files?page=1&page_size=5&file_type=other"
check_ret_ok "文件列表带分页 + file_type 过滤"

req GET "/api/v1/files?status=ready&device_sn=SMK-NO-SUCH-DEV"
check_ret_ok "文件列表带 status + device_sn 过滤（结果可空）"

# ---------------------------------------------------------------------------
section "上传 → 详情 → 搜索 → 下载 → 删除 闭环"
# ---------------------------------------------------------------------------
UP_NAME="${SMOKE_TAG}_smokefile.txt"
UP_FILE="$SMOKE_TMPDIR/$UP_NAME"
printf 'smoke filemanager content %s\nline2 with text\n' "$SMOKE_TAG" > "$UP_FILE"

req_upload "/api/v1/files" \
    "file=@${UP_FILE}" \
    "file_type=other" \
    "description=smoke 冒烟自建文件 ${SMOKE_TAG}" \
    "uploader=smoke"
check_status_in "上传冒烟文件（multipart）" "200 201"
check_field "上传返回文件 ID" "data.id"
FID=$(jget data.id)

if [ -n "$FID" ]; then
    # 详情
    req GET "/api/v1/files/$FID"
    check_ret_ok "文件详情可查"
    check_field "详情含 file_name" "data.file_name"
    check_field "详情含 file_size" "data.file_size"
    GOT_NAME=$(jget data.file_name)
    if [ "$GOT_NAME" = "$UP_NAME" ]; then
        pass "详情 file_name 与上传一致 ($GOT_NAME)"
    else
        fail "详情 file_name 与上传一致" "期望 ${UP_NAME}，实际 ${GOT_NAME}"
    fi

    # 搜索（自建数据保证命中，可用强断言）
    req GET "/api/v1/files?search=${SMOKE_TAG}"
    check_list_nonempty "按 SMOKE_TAG 搜索命中自建文件" "data.items"

    # 下载并比对内容
    DL_FILE="$SMOKE_TMPDIR/downloaded.bin"
    req GET "/api/v1/files/$FID/download" "" -o "$DL_FILE"
    check_status "文件下载" 200
    if cmp -s "$UP_FILE" "$DL_FILE"; then
        pass "下载内容与上传逐字节一致 ($(wc -c < "$UP_FILE" | tr -d ' ') 字节)"
    else
        fail "下载内容与上传逐字节一致" "cmp 比对不一致（上传 $(wc -c < "$UP_FILE" | tr -d ' ')B / 下载 $(wc -c < "$DL_FILE" 2>/dev/null | tr -d ' ')B）"
    fi
else
    skip "文件详情可查" "上传未返回文件 ID，无法继续闭环"
    skip "按 SMOKE_TAG 搜索命中自建文件" "上传未返回文件 ID"
    skip "文件下载与内容比对" "上传未返回文件 ID"
fi

# ---------------------------------------------------------------------------
section "distribute 分发 — 只测参数校验负路径（红线：不向任何设备真发）"
# ---------------------------------------------------------------------------
# 不存在文件 ID + 假 SN：service 先查文件 → 404，不会创建任何任务
req POST "/api/v1/files/$NOEXIST_ID/distribute" "{\"device_sns\":[\"SMK-NO-SUCH-${SMOKE_TAG}\"]}"
check_ret_fail "不存在文件 distribute 被拒"

# 非法 UUID 路径段
req POST "/api/v1/files/not-a-uuid/distribute" '{"device_sns":["SMK-NO-SUCH-SN"]}'
check_ret_fail "非法文件 ID distribute 被拒"

if [ -n "$FID" ]; then
    # 真实文件 + 缺 device_sns 字段：binding required → 400，不会下发
    req POST "/api/v1/files/$FID/distribute" '{}'
    check_ret_fail "缺 device_sns 字段 distribute 被拒"
else
    skip "缺 device_sns 字段 distribute 被拒" "上传未返回文件 ID"
fi

# ---------------------------------------------------------------------------
section "删除冒烟文件（闭环收尾）+ 负路径"
# ---------------------------------------------------------------------------
if [ -n "$FID" ]; then
    req DELETE "/api/v1/files/$FID"
    check_ret_ok "删除冒烟自建文件"
    req GET "/api/v1/files/$FID"
    check_ret_fail "删除后详情查询被拒（404）"
else
    skip "删除冒烟自建文件" "上传未返回文件 ID"
    skip "删除后详情查询被拒" "上传未返回文件 ID"
fi

req GET "/api/v1/files/not-a-uuid"
check_ret_fail "非法 UUID 详情被拒"
req GET "/api/v1/files/$NOEXIST_ID"
check_ret_fail "不存在文件详情被拒"
req DELETE "/api/v1/files/$NOEXIST_ID"
check_ret_fail "不存在文件删除被拒"

# ---------------------------------------------------------------------------
section "四类批量下载 Tab（pm / mr / config-snapshots / device-licenses）"
# ---------------------------------------------------------------------------
# —— 空列表：handler 在写响应头之前显式 400 ——
req POST "/api/v1/pm/files/batch-download" '{}'
check_ret_fail "PM 批量下载空列表优雅拒绝"
req POST "/api/v1/mr/files/batch-download" '{"serial_numbers":[]}'
check_ret_fail "MR 批量下载空列表优雅拒绝"
req POST "/api/v1/backup/config-snapshots/batch-download" '{}'
check_ret_fail "配置快照批量下载空列表优雅拒绝"
req POST "/api/v1/backup/device-licenses/batch-download" '{}'
check_ret_fail "设备 License 批量下载空列表优雅拒绝"

# —— 不存在 SN/ID：后端已先写 200 头再解析 → 200 空 zip；4xx 也算优雅拒绝 ——
FAKE_SN="SMK-NO-SUCH-${SMOKE_TAG}"

req POST "/api/v1/pm/files/batch-download" "{\"serial_numbers\":[\"${FAKE_SN}\"]}"
check_status_in "PM 批量下载不存在 SN" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

req POST "/api/v1/mr/files/batch-download" "{\"serial_numbers\":[\"${FAKE_SN}\"]}"
check_status_in "MR 批量下载不存在 SN" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

req POST "/api/v1/backup/config-snapshots/batch-download" "{\"serial_numbers\":[\"${FAKE_SN}\"]}"
check_status_in "配置快照批量下载不存在 SN" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

req POST "/api/v1/backup/device-licenses/batch-download" "{\"serial_numbers\":[\"${FAKE_SN}\"]}"
check_status_in "设备 License 批量下载不存在 SN" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

# —— by-id 变体（pm/mr 各有 /files/by-id/batch-download，按 file_id 打包）——
req POST "/api/v1/pm/files/by-id/batch-download" "{\"ids\":[\"$NOEXIST_ID\"]}"
check_status_in "PM by-id 批量下载不存在 ID" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

req POST "/api/v1/mr/files/by-id/batch-download" "{\"ids\":[\"$NOEXIST_ID\"]}"
check_status_in "MR by-id 批量下载不存在 ID" "200 400 404"
echo "      回显: HTTP $HTTP_CODE, body $(printf '%s' "$BODY" | wc -c | tr -d ' ') 字节"

smoke_summary

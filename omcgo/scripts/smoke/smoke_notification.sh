#!/usr/bin/env bash
# =============================================================================
# smoke_notification.sh — 通知中心（跨域基础设施）业务冒烟
#
# 覆盖：
#   - 消息列表 / 未读数 / 过滤参数（is_read、type、分页校验负路径）
#   - 单条标记已读（有消息才做；绝不调 read-all，避免污染用户未读状态）
#   - sync-stale 卡死消息修正（幂等 POST 两次）
#   - 通知模板 CRUD 闭环（建 → 查 → 改 → 过滤列表 → 删 → 404 回查）
#   - 通知发送历史（列表 / 过滤 / 详情 / 非法 ID 负路径）
#   - SSE /api/v1/events/stream 建连（无 token 401 + 带 token 收 200 头/任意字节）
#   - Alertmanager webhook 免 JWT 入口（空告警 / 单条告警 / 非法 payload）
#   - 破坏性端点（DELETE 全清 / 单删）只测负路径，绝不真清
#
# 用法：bash smoke_notification.sh [BASE_URL]   # 默认 http://localhost:8081
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "通知中心" "$@"
smoke_login

# ---------------------------------------------------------------------------
# 1. 消息列表 + 未读数
# ---------------------------------------------------------------------------
section "消息列表与未读数"

req GET "/api/v1/notifications?page=1&page_size=20"
check_list_or_empty "当前用户消息列表可查" "data.items"
NOTIF_ID=$(jget data.items.0.id)

req GET "/api/v1/notifications?page=1&page_size=20&is_read=false"
check_list_or_empty "未读过滤（is_read=false）可查" "data.items"

req GET "/api/v1/notifications?page=1&page_size=10&type=task_complete"
check_ret_ok "类型过滤（type=task_complete）可查"

req GET "/api/v1/notifications?page=0&page_size=20"
check_ret_fail "非法分页（page=0）被拒绝"

req GET "/api/v1/notifications/unread-count"
check_ret_ok "未读数可查"
check_field "未读数返回 count 字段" "data.count"

# ---------------------------------------------------------------------------
# 2. 单条标记已读（有消息才做；不调 read-all —— 全量污染未读状态不可接受）
# ---------------------------------------------------------------------------
section "单条消息标记已读"

if [ -n "$NOTIF_ID" ]; then
    req PUT "/api/v1/notifications/${NOTIF_ID}/read"
    check_ret_ok "PUT :id/read 标记已读（id=${NOTIF_ID}）"
else
    skip "PUT :id/read 标记已读" "当前用户无任何消息（通知仅由任务事件产生，无法直接自建）"
fi

req PUT "/api/v1/notifications/not-a-uuid/read"
check_ret_fail "非法 ID 标记已读被拒绝"

# ---------------------------------------------------------------------------
# 3. sync-stale 卡死消息修正（幂等）
# ---------------------------------------------------------------------------
section "sync-stale 卡死消息修正（幂等 POST）"

req POST "/api/v1/notifications/sync-stale"
check_ret_ok "第一次 sync-stale"
check_field "返回 updated 字段" "data.updated"

req POST "/api/v1/notifications/sync-stale"
check_ret_ok "第二次 sync-stale（幂等可重入）"

# ---------------------------------------------------------------------------
# 4. 通知模板 CRUD 闭环
# ---------------------------------------------------------------------------
section "通知模板 CRUD 闭环"

req GET "/api/v1/notifications/templates?page=1&page_size=20"
check_list_or_empty "模板列表可查" "data.items"

TPL_NAME="${SMOKE_TAG}-tpl"
req POST "/api/v1/notifications/templates" \
    "{\"name\":\"${TPL_NAME}\",\"channel\":\"email\",\"language\":\"zh-CN\",\"subject\":\"${SMOKE_TAG} 冒烟测试\",\"body\":\"hello {{name}}\",\"variables\":[\"name\"],\"enabled\":true}"
check_status "创建模板（${TPL_NAME}）" 201
TPL_ID=$(jget data.id)

if [ -n "$TPL_ID" ]; then
    req GET "/api/v1/notifications/templates/${TPL_ID}"
    check_ret_ok "模板详情可查"
    check_field "详情 name 回显" "data.name"

    req PUT "/api/v1/notifications/templates/${TPL_ID}" \
        "{\"subject\":\"${SMOKE_TAG}-updated\",\"enabled\":false}"
    check_ret_ok "更新模板（subject + enabled）"
    UPD_SUBJ=$(jget data.subject)
    if [ "$UPD_SUBJ" = "${SMOKE_TAG}-updated" ]; then
        pass "更新后 subject 生效（${UPD_SUBJ}）"
    else
        fail "更新后 subject 生效" "期望 ${SMOKE_TAG}-updated，实际 '${UPD_SUBJ}'"
    fi

    req GET "/api/v1/notifications/templates?channel=email&enabled=false&page=1&page_size=100"
    check_ret_ok "模板 channel/enabled 过滤列表可查"

    req DELETE "/api/v1/notifications/templates/${TPL_ID}"
    check_ret_ok "删除模板"

    req GET "/api/v1/notifications/templates/${TPL_ID}"
    check_status "删除后回查 404" 404
else
    skip "模板详情/更新/删除闭环" "创建模板未返回 data.id（创建失败已被上一断言捕获）"
fi

req POST "/api/v1/notifications/templates" \
    "{\"name\":\"${SMOKE_TAG}-bad\",\"channel\":\"pigeon\",\"subject\":\"x\",\"body\":\"y\"}"
check_ret_fail "非法 channel 创建模板被拒绝"

# ---------------------------------------------------------------------------
# 5. 通知发送历史（只读，写入侧走内部 dispatcher）
# ---------------------------------------------------------------------------
section "通知发送历史"

req GET "/api/v1/notifications/history?page=1&page_size=20"
check_list_or_empty "发送历史列表可查" "data.items"
HIST_ID=$(jget data.items.0.id)

req GET "/api/v1/notifications/history?channel=email&status=sent&page=1&page_size=20"
check_list_or_empty "历史 channel/status 过滤可查" "data.items"

if [ -n "$HIST_ID" ]; then
    req GET "/api/v1/notifications/history/${HIST_ID}"
    check_ret_ok "历史详情可查（id=${HIST_ID}）"
else
    skip "历史详情可查" "活栈无发送历史（未配置 SMTP/webhook 出口，正常为空）"
fi

req GET "/api/v1/notifications/history/not-a-uuid"
check_ret_fail "非法历史 ID 被拒绝"

# ---------------------------------------------------------------------------
# 6. SSE 事件流建连
#    后端不主动 flush 响应头：首字节最早是 30s keepalive 注释，或 hub 侧关连接
#    （实测 ~18s 收到 200 头）。故 max-time 给 35s；收到 200 头或任意字节即 PASS，
#    curl 超时退出码 28 属预期（长连接未断开）。
# ---------------------------------------------------------------------------
section "SSE 事件流（/api/v1/events/stream）"

req_noauth GET "/api/v1/events/stream"
check_status "SSE 无 token 建连被拒绝" 401

echo "  （SSE 建连等待首字节，最长 35s —— 首字节为 30s keepalive 或 hub 关连接）"
SSE_OUT="$SMOKE_TMPDIR/sse_body.txt"
: > "$SSE_OUT"
SSE_CODE=$(curl --max-time 35 -s -N -o "$SSE_OUT" -w "%{http_code}" \
    -H "Authorization: Bearer $TOKEN" -H "Accept: text/event-stream" \
    "$API/events/stream" 2>/dev/null)
SSE_RC=$?
SSE_BYTES=$(wc -c < "$SSE_OUT" 2>/dev/null | tr -d ' ')
if [ "$SSE_CODE" = "200" ]; then
    pass "SSE 带 token 建连成功（HTTP=200 bytes=${SSE_BYTES} curl_rc=${SSE_RC}）"
elif [ "${SSE_BYTES:-0}" -gt 0 ] 2>/dev/null; then
    pass "SSE 带 token 建连收到数据（HTTP=${SSE_CODE} bytes=${SSE_BYTES} curl_rc=${SSE_RC}）"
else
    fail "SSE 带 token 建连" "35s 内既无 200 头也未收到任何字节（HTTP=${SSE_CODE} curl_rc=${SSE_RC}）"
fi

# ---------------------------------------------------------------------------
# 7. Alertmanager 告警 webhook（免 JWT 入口；dev 栈 token 未配置 → 200，
#    若环境配置了 token 校验则 401 同样视为入口可达）
# ---------------------------------------------------------------------------
section "Alertmanager 告警 webhook（免 JWT 入口）"

req_noauth POST "/api/v1/alerts/webhook" '{"status":"firing","alerts":[]}'
check_status_in "webhook 空告警列表受理（入口可达）" "200 202 401"

req_noauth POST "/api/v1/alerts/webhook" \
    "{\"status\":\"firing\",\"alerts\":[{\"status\":\"firing\",\"labels\":{\"alertname\":\"${SMOKE_TAG}-probe\",\"severity\":\"info\"},\"annotations\":{\"summary\":\"冒烟探测，请忽略\"},\"startsAt\":\"2026-06-10T00:00:00Z\"}]}"
check_status_in "webhook 单条告警受理" "200 202 401"

req_noauth POST "/api/v1/alerts/webhook" '{"alerts":"notanarray"}'
check_ret_fail "webhook 非法 payload 被拒绝"

# ---------------------------------------------------------------------------
# 8. 破坏性端点只测负路径（绝不带认证调 DELETE 全清）
# ---------------------------------------------------------------------------
section "破坏性端点负路径（不真清）"

req_noauth DELETE "/api/v1/notifications"
check_status "无认证清空全部消息被拒绝" 401

req DELETE "/api/v1/notifications/not-a-uuid"
check_ret_fail "非法 ID 单删消息被拒绝"

smoke_summary

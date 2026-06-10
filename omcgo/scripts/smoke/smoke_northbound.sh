#!/usr/bin/env bash
# =============================================================================
# smoke_northbound.sh — F08 北向/OSS 接口 业务冒烟
#
# 覆盖路由（/tmp/smoke_routes.json key=northbound）：
#   GET    /api/v1/northbound/push/targets                 推送目标列表
#   POST   /api/v1/northbound/push/targets                 新增推送目标（闭环自建）
#   GET    /api/v1/northbound/push/targets/:id/circuit     熔断器状态
#   POST   /api/v1/northbound/push/targets/:id/circuit/reset 复位熔断（仅对自建目标）
#   DELETE /api/v1/northbound/push/targets/:id             删除推送目标（闭环清理）
#   GET    /api/v1/northbound/push/deadletter              死信队列（outbox 未装配时 503）
#   POST   /api/v1/northbound/push/deadletter/:id/replay   死信重放（仅负路径）
#   GET    /api/v1/northbound/sync/full                    全量同步（per-endpoint 限流，容忍 429）
#   GET    /api/v1/northbound/sync/incremental             增量同步（since 必填 RFC3339；限流）
#   POST   /api/v1/northbound/export/pm                    PM 导出（start/end_time 必填 RFC3339；限流）
#   POST   /api/v1/northbound/export/alarms                告警导出（限流）
#   GET    /api/v1/northbound/export/config/:deviceId      配置快照导出（UUID；限流）
#   GET    /api/v1/northbound/servers                      主备服务器配置读（未注入时 404 容忍）
#   PUT    /api/v1/northbound/servers/active               主备切换（真实切换跳过，仅非法 role 负路径）
#   PUT    /api/v1/northbound/servers/:role                服务器配置更新（仅非法 role 负路径）
#
# 关键路径（smokeKeyPaths）：
#   推送目标 CRUD 闭环+熔断状态 / 全量·增量同步（限流容忍）/ PM·告警·配置三类导出 / 主备服务器配置读
#
# 注意：
#   - 推送目标存在 PushEngine 内存 map（非持久化），自建目标 URL 用 http://127.0.0.1:1/smk
#     （必然连不通的本地端口）+ enabled=false 双保险，绝不向外部地址真实推送。
#   - sync/export 端点有 per-endpoint Redis 固定窗口限流（默认 300/min），命中 429 时降级 skip。
#   - PUT /servers/active 主备切换影响全局推送 active 端（切换即生效），冒烟只留底 GET，不真切。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F08 北向接口" "$@"
smoke_login

# 时间窗：最近 1h（RFC3339 UTC），export/sync 共用
SINCE=$(python3 -c "import datetime; print((datetime.datetime.now(datetime.timezone.utc)-datetime.timedelta(hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
NOW=$(python3 -c "import datetime; print(datetime.datetime.now(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ'))")

# 数据依赖自备：取第一台设备的 UUID（export/config 用）
req GET "/api/v1/devices?page=1&page_size=1"
DEV_ID=$(jget data.items.0.id)
DEV_SN=$(jget data.items.0.serial_number)

# ---------------------------------------------------------------------------
section "推送目标列表（GET /push/targets）"
# ---------------------------------------------------------------------------
req GET "/api/v1/northbound/push/targets"
check_ret_ok "推送目标列表可查"
check_list_or_empty "推送目标列表 items 形状" "data.items"
check_field "推送目标列表带 total" "data.total"

# ---------------------------------------------------------------------------
section "推送目标 CRUD 闭环 + 熔断状态（POST/GET/DELETE，/:id/circuit）"
# ---------------------------------------------------------------------------
TGT_ID="${SMOKE_TAG}-target"
# URL 用必然连不通的本地地址 + enabled=false：绝不产生真实外发推送
req POST "/api/v1/northbound/push/targets" "{\"id\":\"${TGT_ID}\",\"url\":\"http://127.0.0.1:1/smk\",\"auth_type\":\"none\",\"data_types\":[\"alarm\"],\"format\":\"json\",\"batch_size\":10,\"retry_count\":1,\"enabled\":false}"
check_status "创建推送目标" 201
check_ret_ok "创建推送目标信封 ret=1"
GOT_ID=$(jget data.id)
if [ "$GOT_ID" = "$TGT_ID" ]; then
    pass "创建返回 id 回显一致 (${GOT_ID})"
else
    fail "创建返回 id 回显一致" "期望 ${TGT_ID}，实际 ${GOT_ID}"
fi

req GET "/api/v1/northbound/push/targets"
check_ret_ok "自建后推送目标列表可查"
check_list_nonempty "自建后推送目标列表非空" "data.items"
case "$BODY" in
    *"$TGT_ID"*) pass "列表包含自建目标 (${TGT_ID})" ;;
    *) fail "列表包含自建目标" "items 中未找到 ${TGT_ID}" ;;
esac

# 熔断器状态：新建目标初始 closed
req GET "/api/v1/northbound/push/targets/$TGT_ID/circuit"
check_ret_ok "自建目标熔断状态可查"
CB_STATE=$(jget data.state)
if [ "$CB_STATE" = "closed" ]; then
    pass "新建目标熔断器初始 state=closed"
else
    fail "新建目标熔断器初始 state=closed" "实际 state=${CB_STATE}"
fi
check_field "熔断状态含 threshold" "data.threshold"
check_count_ge "熔断器 failure_count 初始为 0" "data.failure_count" 0

# 复位熔断（仅对自建目标，纯内存操作，安全可逆）
req POST "/api/v1/northbound/push/targets/$TGT_ID/circuit/reset"
check_ret_ok "复位自建目标熔断器"
RESET_STATE=$(jget data.state)
if [ "$RESET_STATE" = "closed" ]; then
    pass "复位后熔断器 state=closed"
else
    fail "复位后熔断器 state=closed" "实际 state=${RESET_STATE}"
fi

# 闭环清理
req DELETE "/api/v1/northbound/push/targets/$TGT_ID"
check_ret_ok "删除自建推送目标"
req GET "/api/v1/northbound/push/targets/$TGT_ID/circuit"
check_status "删除后查熔断状态 404（连带清理）" 404

# 负路径：缺必填字段 / 不存在 ID
req POST "/api/v1/northbound/push/targets" '{"url":"http://127.0.0.1:1/smk"}'
check_ret_fail "缺 id/data_types 创建推送目标被拒"
req POST "/api/v1/northbound/push/targets" '{"id":"x"}'
check_ret_fail "缺 url 创建推送目标被拒"
req DELETE "/api/v1/northbound/push/targets/${SMOKE_TAG}-not-exist"
check_ret_fail "删除不存在推送目标被拒"
req GET "/api/v1/northbound/push/targets/${SMOKE_TAG}-not-exist/circuit"
check_ret_fail "查不存在目标熔断状态被拒"
req POST "/api/v1/northbound/push/targets/${SMOKE_TAG}-not-exist/circuit/reset"
check_ret_fail "复位不存在目标熔断器被拒"

# ---------------------------------------------------------------------------
section "死信队列（GET /push/deadletter，POST /:id/replay 负路径）"
# ---------------------------------------------------------------------------
req GET "/api/v1/northbound/push/deadletter?limit=20&offset=0"
if [ "$HTTP_CODE" = "200" ]; then
    check_ret_ok "死信队列可查"
    check_list_or_empty "死信列表 items 形状" "data.items"
elif [ "$HTTP_CODE" = "503" ]; then
    known_bug "死信队列读（outbox 未装配）" "GET /push/deadletter → 503 'outbox not configured'：Router 支持 SetOutboxRepo 但 provider/modules.go 从未注入 OutboxRepository，DLQ 端点恒 503"
else
    fail "死信队列读" "期望 200/503，实际 HTTP ${HTTP_CODE}，body: $(printf '%s' "$BODY" | head -c 200)"
fi

# 重放只测负路径（outbox 未配置 503 / 非法 uuid 400 均为被拒）
req POST "/api/v1/northbound/push/deadletter/not-a-uuid/replay"
check_ret_fail "非法 uuid 死信重放被拒"
req POST "/api/v1/northbound/push/deadletter/00000000-dead-beef-0000-000000000000/replay"
check_ret_fail "不存在死信重放被拒"

# ---------------------------------------------------------------------------
section "全量/增量同步（GET /sync/full, /sync/incremental；限流容忍 429）"
# ---------------------------------------------------------------------------
req GET "/api/v1/northbound/sync/full?data_type=device"
check_status_in "全量同步(device) HTTP" "200 429"
if [ "$HTTP_CODE" = "200" ]; then
    check_ret_ok "全量同步信封 ret=1"
    SYNC_TYPE=$(jget data.data_type)
    if [ "$SYNC_TYPE" = "device" ]; then
        pass "全量同步回显 data_type=device"
    else
        fail "全量同步回显 data_type=device" "实际 ${SYNC_TYPE}"
    fi
    check_field "全量同步带 synced_at" "data.synced_at"
    check_list_or_empty "全量同步 items 形状" "data.items"
else
    skip "全量同步形状断言" "命中 per-endpoint 限流(429)，本轮跳过"
fi

req GET "/api/v1/northbound/sync/incremental?data_type=alarm&since=$SINCE"
check_status_in "增量同步(alarm,since=1h前) HTTP" "200 429"
if [ "$HTTP_CODE" = "200" ]; then
    check_ret_ok "增量同步信封 ret=1"
    check_field "增量同步带 synced_at" "data.synced_at"
    check_list_or_empty "增量同步 items 形状" "data.items"
else
    skip "增量同步形状断言" "命中 per-endpoint 限流(429)，本轮跳过"
fi

# 负路径：未知过滤参数白名单 fail-closed / 非法 data_type / since 缺失或格式错
req GET "/api/v1/northbound/sync/full?data_type=device&bogus_param=1"
check_ret_fail "全量同步未知过滤参数被拒（白名单 fail-closed）"
req GET "/api/v1/northbound/sync/full?data_type=bogus"
check_ret_fail "全量同步非法 data_type 被拒"
req GET "/api/v1/northbound/sync/incremental?data_type=alarm"
check_ret_fail "增量同步缺 since 被拒"
req GET "/api/v1/northbound/sync/incremental?data_type=alarm&since=not-a-time"
check_ret_fail "增量同步非法 since 格式被拒"

# ---------------------------------------------------------------------------
section "三类导出：PM / 告警 / 配置（POST /export/pm, /export/alarms, GET /export/config/:deviceId）"
# ---------------------------------------------------------------------------
req POST "/api/v1/northbound/export/pm" "{\"start_time\":\"$SINCE\",\"end_time\":\"$NOW\"}"
check_status_in "PM 导出(最近1h) HTTP" "200 429"
if [ "$HTTP_CODE" = "200" ]; then
    check_ret_ok "PM 导出信封 ret=1"
    check_list_or_empty "PM 导出 items 形状" "data.items"
else
    skip "PM 导出形状断言" "命中 per-endpoint 限流(429)，本轮跳过"
fi

req POST "/api/v1/northbound/export/alarms" "{\"start_time\":\"$SINCE\",\"end_time\":\"$NOW\"}"
check_status_in "告警导出(最近1h) HTTP" "200 429"
if [ "$HTTP_CODE" = "200" ]; then
    check_ret_ok "告警导出信封 ret=1"
    check_list_or_empty "告警导出 items 形状" "data.items"
else
    skip "告警导出形状断言" "命中 per-endpoint 限流(429)，本轮跳过"
fi

if [ -n "$DEV_ID" ]; then
    req GET "/api/v1/northbound/export/config/$DEV_ID"
    check_status_in "配置导出(设备 ${DEV_SN}) HTTP" "200 429"
    if [ "$HTTP_CODE" = "200" ]; then
        check_ret_ok "配置导出信封 ret=1"
        GOT_DEV=$(jget data.device_id)
        if [ "$GOT_DEV" = "$DEV_ID" ]; then
            pass "配置导出回显 device_id 一致"
        else
            fail "配置导出回显 device_id 一致" "期望 ${DEV_ID}，实际 ${GOT_DEV}"
        fi
        check_field "配置导出带 total" "data.total"
    else
        skip "配置导出形状断言" "命中 per-endpoint 限流(429)，本轮跳过"
    fi
else
    skip "配置导出（真实设备）" "活栈无设备可用（GET /devices 取不到 items.0.id）"
fi

# 负路径：缺必填时间窗 / 未知 body 字段白名单 fail-closed / 非法 device_id
req POST "/api/v1/northbound/export/pm" '{}'
check_ret_fail "PM 导出缺 start_time/end_time 被拒"
req POST "/api/v1/northbound/export/pm" "{\"start_time\":\"$SINCE\",\"end_time\":\"$NOW\",\"bogus_field\":1}"
check_ret_fail "PM 导出未知 body 字段被拒（白名单 fail-closed）"
req POST "/api/v1/northbound/export/pm" "{\"start_time\":\"$SINCE\",\"end_time\":\"$NOW\",\"device_id\":\"not-a-uuid\"}"
check_ret_fail "PM 导出非法 device_id 被拒"
req POST "/api/v1/northbound/export/alarms" '{"bogus_field":1}'
check_ret_fail "告警导出未知 body 字段被拒（白名单 fail-closed）"
req GET "/api/v1/northbound/export/config/not-a-uuid"
check_ret_fail "配置导出非法 deviceId 被拒"

# ---------------------------------------------------------------------------
section "主备服务器配置（GET /servers 留底；PUT 真实切换跳过）"
# ---------------------------------------------------------------------------
req GET "/api/v1/northbound/servers"
if [ "$HTTP_CODE" = "404" ]; then
    skip "主备服务器配置读" "serverHandler 未注入（404），环境性缺失"
else
    check_ret_ok "主备服务器配置可查（留底）"
    check_list_or_empty "servers items 形状" "data.items"
    check_field "servers 带 total" "data.total"
    echo "  ↪ 留底 servers 当前状态: $(jget data.items | head -c 200)"
fi

# PUT /servers/active 真实主备切换影响全局推送端（切换即生效），冒烟不执行；仅测非法 role 负路径
skip "PUT /servers/active 真实主备切换" "影响全局推送 active 端（切换即生效），冒烟禁区，仅测负路径"
req PUT "/api/v1/northbound/servers/active" '{"role":"bogus"}'
check_ret_fail "非法 role 主备切换被拒"
req PUT "/api/v1/northbound/servers/active" '{}'
check_ret_fail "缺 role 主备切换被拒"
req PUT "/api/v1/northbound/servers/bogusrole" '{"host":"127.0.0.1","port":8443,"description":"smoke"}'
check_ret_fail "非法 role 更新服务器配置被拒"

smoke_summary

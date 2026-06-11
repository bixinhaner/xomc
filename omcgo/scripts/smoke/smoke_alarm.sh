#!/usr/bin/env bash
# =============================================================================
# smoke_alarm.sh — F04 告警管理 业务冒烟
#
# 覆盖三个子域（/tmp/smoke_routes.json: alarm + alarmdef + eventlog）：
#
# alarm（omcgo/internal/alarm/handler.go + filter_handler.go）
#   GET  /alarms/active | /history | /statistics | /history/statistics | /:id
#   POST /alarms/:id/acknowledge + /alarms/active/batch/unacknowledge（有告警时闭环还原）
#   POST /alarms/:id/clear、/alarms/history/batch/delete（destructive → 仅负路径）
#   POST /alarms/sync/:device_sn（幽灵 SN，GPV 任务 10 分钟过期，不触达真实设备）
#   /alarms/alarm-filters CRUD + toggle 闭环（自建自删，enabled=false + 不命中任何源）
#
# alarmdef（omcgo/internal/alarm/definition/handler.go + file_handler.go，super_admin）
#   GET  /alarm-definitions | /ne-types | /unknown-stats | /:identifier | /alarm-severity-levels
#   POST/PUT/DELETE /alarm-definitions 自定义定义 CRUD 闭环
#   GET  /alarm-definitions/file-content?loaded_from=（builtin 可下载）
#   DELETE /alarm-definitions/files/*（builtin 403 守门负路径）
#   POST /alarm-definitions/upload-xml 上传→校验→删 custom 文件闭环
#       （守护：栈上存在 API 自建定义时跳过——upload 触发 destructive 孤儿清理会删掉它们）
#   super_admin 403 边界：临时建普通用户打一发 → 403 → 删用户还原
#
# eventlog（omcgo/internal/eventlog/handler.go）
#   GET /event-logs | /statistics | /:id（活栈数据稀疏，空容忍）
#
# 红线：不向任何真实设备 SN 下发 reboot/升级/恢复/SPV 等指令；
#       告警同步只打不存在的幽灵 SN（GetParameterValues 只读任务，自然过期）。
#
# 用法：bash smoke_alarm.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F04 告警管理" "$@"
smoke_login

RANDOM_UUID="00000000-dead-beef-0000-000000000000"

# ---------------------------------------------------------------------------
section "活跃/历史告警列表（含过滤参数）"
# ---------------------------------------------------------------------------
req GET "/api/v1/alarms/active"
check_list_or_empty "活跃告警列表可查" "data.items"
check_field "活跃告警列表含 total" "data.total"
ACTIVE_TOTAL=$(jget "data.total")
FIRST_ALARM_ID=$(jget "data.items.0.id")
FIRST_ALARM_STATUS=$(jget "data.items.0.status")

req GET "/api/v1/alarms/active?severity=1&page=1&page_size=5"
check_ret_ok "活跃告警按严重级过滤(severity=1)可查"

req GET "/api/v1/alarms/active?severity=1,2,3,4&status=active"
check_ret_ok "活跃告警多严重级+状态过滤可查"

req GET "/api/v1/alarms/active?device_id=not-a-uuid"
check_ret_fail "活跃告警 device_id 非 UUID 被拒绝"

req GET "/api/v1/alarms/history"
check_list_or_empty "历史告警列表可查" "data.items"

req GET "/api/v1/alarms/history?severity=1,2&page=1&page_size=5"
check_ret_ok "历史告警过滤分页可查"

req GET "/api/v1/alarms/history?device_id=not-a-uuid"
check_ret_fail "历史告警 device_id 非 UUID 被拒绝"

# ---------------------------------------------------------------------------
section "告警统计（活跃 + 历史）"
# ---------------------------------------------------------------------------
req GET "/api/v1/alarms/statistics"
check_ret_ok "活跃告警统计可查"
check_field "统计含 total_active" "data.total_active"
check_field "统计含 by_severity" "data.by_severity"

req GET "/api/v1/alarms/history/statistics"
check_ret_ok "历史告警统计可查"

# ---------------------------------------------------------------------------
section "告警详情 + 确认↔反确认闭环"
# ---------------------------------------------------------------------------
if [ -n "$FIRST_ALARM_ID" ]; then
    req GET "/api/v1/alarms/$FIRST_ALARM_ID"
    check_ret_ok "告警详情可查(id=${FIRST_ALARM_ID})"
    check_field "详情含 id" "data.id"

    if [ "$FIRST_ALARM_STATUS" = "active" ]; then
        req POST "/api/v1/alarms/$FIRST_ALARM_ID/acknowledge" "{\"acknowledged_by\":\"$SMOKE_TAG\"}"
        check_ret_ok "告警确认成功"

        req GET "/api/v1/alarms/$FIRST_ALARM_ID"
        ACK_STATUS=$(jget "data.status")
        if [ "$ACK_STATUS" = "acknowledged" ]; then
            pass "确认后状态变为 acknowledged"
        else
            fail "确认后状态变为 acknowledged" "实际 status=${ACK_STATUS}"
        fi

        req POST "/api/v1/alarms/active/batch/unacknowledge" "{\"ids\":[\"$FIRST_ALARM_ID\"]}"
        check_ret_ok "批量反确认还原成功"

        req GET "/api/v1/alarms/$FIRST_ALARM_ID"
        UNACK_STATUS=$(jget "data.status")
        if [ "$UNACK_STATUS" = "active" ]; then
            pass "反确认后状态还原为 active"
        else
            fail "反确认后状态还原为 active" "实际 status=${UNACK_STATUS}"
        fi
    else
        skip "确认↔反确认闭环" "首条活跃告警状态为 ${FIRST_ALARM_STATUS}（非 active），避免改动现状"
    fi
else
    skip "告警详情/确认闭环" "活栈当前 0 条活跃告警（total=${ACTIVE_TOTAL}；cpe_simulator 注入告警事件不可靠）"
fi

# 确认相关负路径（无论有无数据都要打）
req POST "/api/v1/alarms/not-a-uuid/acknowledge" "{\"acknowledged_by\":\"$SMOKE_TAG\"}"
check_ret_fail "确认非法 ID 被拒绝"

req POST "/api/v1/alarms/$RANDOM_UUID/acknowledge" "{\"acknowledged_by\":\"$SMOKE_TAG\"}"
check_ret_fail "确认不存在告警被拒绝"

req POST "/api/v1/alarms/active/batch/acknowledge" "{}"
check_ret_fail "批量确认缺 ids 被拒绝"

req POST "/api/v1/alarms/active/batch/unacknowledge" "{}"
check_ret_fail "批量反确认缺 ids 被拒绝"

req GET "/api/v1/alarms/$RANDOM_UUID"
check_ret_fail "查询不存在告警被拒绝(期望 404)"

# ---------------------------------------------------------------------------
section "清除/删历史（destructive，仅参数校验负路径）"
# ---------------------------------------------------------------------------
req POST "/api/v1/alarms/not-a-uuid/clear"
check_ret_fail "清除非法 ID 被拒绝"

req POST "/api/v1/alarms/$RANDOM_UUID/clear"
check_ret_fail "清除不存在告警被拒绝"

req POST "/api/v1/alarms/active/batch/clear" "{}"
check_ret_fail "批量清除缺 ids 被拒绝"

req POST "/api/v1/alarms/history/batch/delete" "{}"
check_ret_fail "批量删历史缺 ids 被拒绝"

req POST "/api/v1/alarms/history/batch/delete" "{\"ids\":\"oops\"}"
check_ret_fail "批量删历史 ids 非数组被拒绝"

# ---------------------------------------------------------------------------
section "告警同步 sync（幽灵 SN，不触达真实设备）"
# ---------------------------------------------------------------------------
# TriggerSync 仅创建 GetParameterValues 只读任务（ExpiresIn=600s）；
# 用不存在的幽灵 SN：无设备会取走任务，10 分钟自然过期，零真实下发。
GHOST_SN="${SMOKE_TAG}GHOST"
req POST "/api/v1/alarms/sync/$GHOST_SN"
check_ret_ok "告警同步触发成功(幽灵 SN=${GHOST_SN})"
check_field "同步响应回显 device_sn" "data.device_sn"

# ---------------------------------------------------------------------------
section "过滤规则 alarm-filters CRUD + toggle 闭环"
# ---------------------------------------------------------------------------
req GET "/api/v1/alarms/alarm-filters"
check_list_or_empty "过滤规则列表可查" "data.items"

# 自建规则：enabled=false + alarm_sources 不命中任何真实源 → 对活栈零影响
FLT_NAME="${SMOKE_TAG}-flt"
req POST "/api/v1/alarms/alarm-filters" "{\"name\":\"$FLT_NAME\",\"filter_type\":\"alarm_source\",\"alarm_sources\":[\"${SMOKE_TAG}-src\"],\"action\":\"ignore\",\"priority\":1,\"enabled\":false}"
check_status "创建过滤规则" 201
FLT_ID=$(jget "data.id")
check_field "创建返回规则 id" "data.id"

if [ -n "$FLT_ID" ]; then
    req GET "/api/v1/alarms/alarm-filters/$FLT_ID"
    check_ret_ok "过滤规则详情可查"
    FLT_GOT_NAME=$(jget "data.name")
    if [ "$FLT_GOT_NAME" = "$FLT_NAME" ]; then
        pass "详情 name 与创建一致"
    else
        fail "详情 name 与创建一致" "期望 ${FLT_NAME}，实际 ${FLT_GOT_NAME}"
    fi

    req PUT "/api/v1/alarms/alarm-filters/$FLT_ID" "{\"priority\":42,\"acknowledge_desc\":\"smoke updated\"}"
    check_ret_ok "更新过滤规则成功"
    FLT_PRI=$(jget "data.priority")
    if [ "$FLT_PRI" = "42" ]; then
        pass "更新后 priority=42"
    else
        fail "更新后 priority=42" "实际 priority=${FLT_PRI}"
    fi

    req POST "/api/v1/alarms/alarm-filters/$FLT_ID/toggle"
    check_ret_ok "toggle 启用成功"
    req GET "/api/v1/alarms/alarm-filters/$FLT_ID"
    FLT_EN=$(jget "data.enabled")
    if [ "$FLT_EN" = "True" ] || [ "$FLT_EN" = "true" ]; then
        pass "toggle 后 enabled=true"
    else
        fail "toggle 后 enabled=true" "实际 enabled=${FLT_EN}"
    fi

    req POST "/api/v1/alarms/alarm-filters/$FLT_ID/toggle"
    check_ret_ok "toggle 还原禁用成功"
    req GET "/api/v1/alarms/alarm-filters/$FLT_ID"
    FLT_EN2=$(jget "data.enabled")
    if [ "$FLT_EN2" = "False" ] || [ "$FLT_EN2" = "false" ]; then
        pass "再 toggle 后 enabled 还原 false"
    else
        fail "再 toggle 后 enabled 还原 false" "实际 enabled=${FLT_EN2}"
    fi

    req DELETE "/api/v1/alarms/alarm-filters/$FLT_ID"
    check_ret_ok "删除过滤规则成功"

    req GET "/api/v1/alarms/alarm-filters/$FLT_ID"
    check_ret_fail "删除后规则查不到(期望 404)"
else
    skip "过滤规则闭环后续步骤" "创建未返回 id，无法继续"
fi

# 过滤规则负路径
req POST "/api/v1/alarms/alarm-filters" "{}"
check_ret_fail "创建规则缺必填字段被拒绝"

req POST "/api/v1/alarms/alarm-filters" "{\"name\":\"x\",\"filter_type\":\"alarm_source\",\"action\":\"bogus\"}"
check_ret_fail "创建规则非法 action 被拒绝"

req PUT "/api/v1/alarms/alarm-filters/not-a-uuid" "{\"priority\":1}"
check_ret_fail "更新非法规则 ID 被拒绝"

req DELETE "/api/v1/alarms/alarm-filters/not-a-uuid"
check_ret_fail "删除非法规则 ID 被拒绝"

# ---------------------------------------------------------------------------
section "告警定义库（builtin 字典必有数据）"
# ---------------------------------------------------------------------------
req GET "/api/v1/alarm-definitions?page=1&page_size=20"
check_ret_ok "告警定义列表可查"
check_list_nonempty "告警定义列表非空(builtin)" "data.items"
check_count_ge "告警定义 total ≥ 1" "data.total" 1
DEF_IDENTIFIER=$(jget "data.items.0.identifier")
DEF_NE_TYPE=$(jget "data.items.0.ne_type")

req GET "/api/v1/alarm-definitions/ne-types"
check_ret_ok "ne-types 聚合可查"
check_list_nonempty "ne-types 列表非空" "data.items"
# 找一个 builtin 文件路径（守门负路径用）；最多翻前 10 条
BUILTIN_FILE=""
i=0
while [ $i -lt 10 ]; do
    SRC=$(jget "data.items.$i.source")
    [ -z "$SRC" ] && break
    if [ "$SRC" = "builtin" ]; then
        BUILTIN_FILE=$(jget "data.items.$i.loaded_from")
        break
    fi
    i=$((i + 1))
done

req GET "/api/v1/alarm-severity-levels"
check_ret_ok "告警严重级别可查"
check_list_nonempty "严重级别非空(4 行种子)" "data.items"
SEV_CODE=$(jget "data.items.0.code")

req GET "/api/v1/alarm-definitions/unknown-stats"
check_ret_ok "未知告警统计可查"
check_field "未知告警统计含 days" "data.days"

req GET "/api/v1/alarm-definitions/unknown-stats?productId=not-a-uuid"
check_ret_fail "未知告警统计非法 productId 被拒绝"

if [ -n "$DEF_NE_TYPE" ]; then
    NE_ENC=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$DEF_NE_TYPE")
    req GET "/api/v1/alarm-definitions?ne_type=${NE_ENC}&page=1&page_size=5"
    check_list_nonempty "按 ne_type=${DEF_NE_TYPE} 过滤非空" "data.items"
fi

if [ -n "$DEF_IDENTIFIER" ]; then
    ID_ENC=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$DEF_IDENTIFIER")
    req GET "/api/v1/alarm-definitions/$ID_ENC"
    check_ret_ok "定义详情可查(identifier=${DEF_IDENTIFIER})"
    check_field "详情含 severity_name" "data.severity_name"
else
    skip "定义详情" "列表未返回 identifier"
fi

req GET "/api/v1/alarm-definitions/__smoke_no_such_identifier__"
check_ret_fail "查询不存在定义被拒绝(期望 404)"

# ---------------------------------------------------------------------------
section "自定义告警定义 CRUD 闭环"
# ---------------------------------------------------------------------------
if [ -n "$SEV_CODE" ] && [ -n "$DEF_NE_TYPE" ]; then
    SMK_IDENT="SMK-${SMOKE_TAG}"
    req POST "/api/v1/alarm-definitions" "{\"identifier\":\"$SMK_IDENT\",\"ne_type\":\"$DEF_NE_TYPE\",\"cn_name\":\"冒烟测试定义\",\"en_name\":\"smoke def\",\"severity_code\":$SEV_CODE}"
    check_status "创建自定义定义" 201
    check_field "创建返回定义 id" "data.id"

    req GET "/api/v1/alarm-definitions/$SMK_IDENT"
    check_ret_ok "自建定义详情可查"

    req PUT "/api/v1/alarm-definitions/$SMK_IDENT" "{\"description\":\"smoke updated\"}"
    check_ret_ok "更新自建定义成功"
    DEF_DESC=$(jget "data.description")
    if [ "$DEF_DESC" = "smoke updated" ]; then
        pass "更新后 description 生效"
    else
        fail "更新后 description 生效" "实际 description=${DEF_DESC}"
    fi

    req DELETE "/api/v1/alarm-definitions/$SMK_IDENT"
    check_ret_ok "删除自建定义成功"

    req GET "/api/v1/alarm-definitions/$SMK_IDENT"
    check_ret_fail "删除后定义查不到(期望 404)"
else
    skip "自定义定义 CRUD 闭环" "未取到严重级别 code 或 builtin ne_type"
fi

# 定义写路径负路径
req POST "/api/v1/alarm-definitions" "{}"
check_ret_fail "创建定义缺必填字段被拒绝"

req POST "/api/v1/alarm-definitions" "{\"identifier\":\"SMK-BAD-${SMOKE_TAG}\",\"ne_type\":\"ENB\",\"severity_code\":99999999}"
check_ret_fail "创建定义非法 severity_code 被拒绝"

req DELETE "/api/v1/alarm-definitions/__smoke_no_such_identifier__"
check_ret_fail "删除不存在定义被拒绝(期望 404)"

# ---------------------------------------------------------------------------
section "定义库 XML 文件管理（file-content / builtin 删除守门 / 上传闭环）"
# ---------------------------------------------------------------------------
if [ -n "$BUILTIN_FILE" ]; then
    LF_ENC=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$BUILTIN_FILE")
    req GET "/api/v1/alarm-definitions/file-content?loaded_from=${LF_ENC}"
    check_status "builtin XML 原文件可下载(${BUILTIN_FILE})" 200

    req DELETE "/api/v1/alarm-definitions/files/$BUILTIN_FILE"
    check_status "删除 builtin 文件被 403 拒绝" 403
else
    skip "builtin 文件下载/删除守门" "ne-types 未发现 source=builtin 的文件"
fi

req GET "/api/v1/alarm-definitions/file-content?loaded_from=../../etc/passwd"
check_ret_fail "file-content 路径遍历被拒绝"

req DELETE "/api/v1/alarm-definitions/files/alarm-definitions/${SMOKE_TAG}_nosuch.xml"
check_ret_fail "删除不存在的 XML 文件被拒绝"

# 上传闭环守护：upload-xml 成功后触发 destructive 孤儿清理（删除所有无 XML 背书的
# API 自建定义行）。若栈上已有 loaded_from 为空的定义（他人自建数据），跳过以防误删。
req GET "/api/v1/alarm-definitions?loaded_from=__empty__&page=1&page_size=1"
API_DEF_TOTAL=$(jget "data.total")
if [ -n "$API_DEF_TOTAL" ] && [ "$API_DEF_TOTAL" -gt 0 ] 2>/dev/null; then
    skip "upload-xml 上传闭环" "栈上存在 ${API_DEF_TOTAL} 条 API 自建定义(loaded_from 为空)，上传会触发孤儿清理删掉它们"
else
    # ne_type 列是 varchar(16)：超长 neType 会被上传守门 400 拒绝（#123 修复后硬校验）。
    # 取 SMOKE_TAG 的时间戳段拼 "SMK" 前缀，恰好 ≤16 字符且每次运行唯一。
    NE_SMK="SMK${SMOKE_TAG:3:13}"
    SMK_XML="$SMOKE_TMPDIR/${NE_SMK}.xml"
    cat > "$SMK_XML" <<XMLEOF
<?xml version="1.0" encoding="UTF-8"?>
<alarmModel neType="${NE_SMK}" deviceType="9" totalCount="1">
    <alarms>
        <alarm identifier="SMKID${SMOKE_TAG}" cnName="冒烟上传测试" enName="Smoke Upload Test" severity="Major" eventType="30003" cnProbableCause="smoke" enProbableCause="smoke" isShow="Y" />
    </alarms>
</alarmModel>
XMLEOF
    req_upload "/api/v1/alarm-definitions/upload-xml" "file=@$SMK_XML"
    check_status "上传自定义告警 XML" 201
    UP_NETYPE=$(jget "data.ne_type")
    if [ "$UP_NETYPE" = "$NE_SMK" ]; then
        pass "上传响应 ne_type=${NE_SMK}"
    else
        fail "上传响应 ne_type=${NE_SMK}" "实际 ne_type=${UP_NETYPE}"
    fi

    req GET "/api/v1/alarm-definitions?ne_type=${NE_SMK}&page=1&page_size=5"
    check_list_nonempty "上传后定义入库(ne_type=${NE_SMK})" "data.items"

    # 重复上传（无 force）→ 409
    req_upload "/api/v1/alarm-definitions/upload-xml" "file=@$SMK_XML"
    check_status "重复上传同 neType 返回 409" 409

    req DELETE "/api/v1/alarm-definitions/files/alarm-definitions/${NE_SMK}.xml"
    check_ret_ok "删除 custom XML 文件成功(sidecar 可删)"

    req GET "/api/v1/alarm-definitions?ne_type=${NE_SMK}&page=1&page_size=5"
    SMK_LEFT=$(jget "data.total")
    if [ "$SMK_LEFT" = "0" ]; then
        pass "删除后定义行清空(ne_type=${NE_SMK})"
    else
        fail "删除后定义行清空(ne_type=${NE_SMK})" "实际 total=${SMK_LEFT}"
    fi
fi

# 上传负路径：非 XML 内容 → 400
BAD_XML="$SMOKE_TMPDIR/bad.xml"
printf 'not an xml at all' > "$BAD_XML"
req_upload "/api/v1/alarm-definitions/upload-xml" "file=@$BAD_XML"
check_ret_fail "上传非法 XML 被拒绝"

# 上传负路径：neType 超长（>16，ne_type 列 varchar(16)）→ 400 守门拒绝。
# #123 修复回归：此前会 201 假成功（reload 失败仅 Warn）但 0 行入库。
# 守门在落盘/重载之前拒绝，不触发孤儿清理，可不受上方守护限制独立执行。
LONG_NE="SMK${SMOKE_TAG:3:13}LONG"
LONG_XML="$SMOKE_TMPDIR/long_ne.xml"
cat > "$LONG_XML" <<XMLEOF
<?xml version="1.0" encoding="UTF-8"?>
<alarmModel neType="${LONG_NE}" deviceType="9" totalCount="0">
    <alarms></alarms>
</alarmModel>
XMLEOF
req_upload "/api/v1/alarm-definitions/upload-xml" "file=@$LONG_XML"
check_status "上传超长 neType(>16 字符) 返回 400" 400

# ---------------------------------------------------------------------------
section "super_admin 403 边界（普通用户访问定义库）"
# ---------------------------------------------------------------------------
# alarm-definitions 挂在 RequireSuperAdmin 组：仅 builtin 用户（admin）放行。
# 临时建一个普通用户（无角色即可，403 与角色无关）→ 登录 → 打 403 → 删用户还原。
TMP_USER="${SMOKE_TAG}u"
TMP_PASS="Smk@1234abcd"
TMP_ENC=$(encrypt_password "$TMP_PASS")
req POST "/api/v1/admin/users" "{\"username\":\"$TMP_USER\",\"encrypted_password\":\"$TMP_ENC\",\"key_id\":\"$PUBLIC_KEY_ID\",\"display_name\":\"smoke temp\"}"
TMP_UID=$(jget "data.id")
if [[ "$HTTP_CODE" == 2* ]] && [ -n "$TMP_UID" ]; then
    pass "创建临时普通用户(${TMP_USER})"
    TMP_LOGIN_ENC=$(encrypt_password "$TMP_PASS")
    TMP_RESP=$(curl --max-time 10 -s -X POST "$API/auth/login" \
        -H 'Content-Type: application/json' \
        -d "{\"username\":\"$TMP_USER\",\"encrypted_password\":\"$TMP_LOGIN_ENC\",\"key_id\":\"$PUBLIC_KEY_ID\"}")
    TMP_TOKEN=$(printf '%s' "$TMP_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('access_token',''))" 2>/dev/null)
    if [ -n "$TMP_TOKEN" ]; then
        ADMIN_TOKEN="$TOKEN"
        TOKEN="$TMP_TOKEN"
        req GET "/api/v1/alarm-definitions?page=1&page_size=1"
        check_status "普通用户访问定义库被 403 拒绝" 403
        TOKEN="$ADMIN_TOKEN"
    else
        skip "普通用户 403 边界" "临时用户登录失败（可能被首登改密/锁定策略拦截）：$(printf '%s' "$TMP_RESP" | head -c 120)"
    fi
    req DELETE "/api/v1/admin/users/$TMP_UID"
    check_ret_ok "清理临时用户"
else
    skip "普通用户 403 边界" "临时用户创建失败 HTTP ${HTTP_CODE}：$(printf '%s' "$BODY" | head -c 120)"
fi

# 未认证 → 401
req_noauth GET "/api/v1/alarm-definitions"
check_status "未认证访问定义库被 401 拒绝" 401

# ---------------------------------------------------------------------------
section "设备事件日志 event-logs"
# ---------------------------------------------------------------------------
req GET "/api/v1/event-logs"
check_list_or_empty "事件日志列表可查" "data.items"
EVT_ID=$(jget "data.items.0.id")

req GET "/api/v1/event-logs?event_type=boot&page=1&page_size=5"
check_ret_ok "事件日志按类型过滤可查"

req GET "/api/v1/event-logs/statistics"
check_ret_ok "事件日志统计可查"
check_field "统计含 total" "data.total"

if [ -n "$EVT_ID" ]; then
    req GET "/api/v1/event-logs/$EVT_ID"
    check_ret_ok "事件日志详情可查(id=${EVT_ID})"
    check_field "详情含 id" "data.id"
else
    req GET "/api/v1/event-logs/$RANDOM_UUID"
    check_ret_fail "查询不存在事件日志被拒绝(期望 404；活栈无事件日志数据，详情降级走负路径)"
fi

req GET "/api/v1/event-logs/not-a-uuid"
check_ret_fail "事件日志非法 ID 被拒绝"

smoke_summary

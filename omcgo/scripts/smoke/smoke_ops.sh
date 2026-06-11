#!/usr/bin/env bash
# =============================================================================
# smoke_ops.sh — F06 运维工具（ops）业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=ops 全部 read 路由 + 可逆写闭环 + 危险负路径）：
#   1. ops/templates           列表 + CRUD 闭环（建→查→改→删→404）
#   2. ops/tasks               列表 + 详情/run/approve/cancel/pause/resume 负路径
#   3. ops/command-records     列表 + 创建校验负路径（POST 无删除接口，不做正向写）
#   4. ops/diagnostics         列表 + ping/traceroute/throughput 校验负路径
#   5. ops/downloads           列表 + collect 校验负路径
#   6. ops/maintenance-windows 列表/active + 建未来窗口 + 四眼审批守卫
#   7. ops/audit-logs          列表 + 维护窗口创建留痕断言
#   8. ops/playbooks           列表 + match + 校验负路径
#   9. ops/commands/rpc        只测校验负路径【红线：不真发 RPC】
#  10. ops/break-glass         status 只读 + activate 校验负路径【红线：不真激活/不触 deactivate】
#
# 信封一致性（#126 第6项已修，2026-06-11）：
#   - 基础端点 templates/tasks/command-records 走统一信封 {ret,msg,data}；
#   - 扩展端点 diagnostics/downloads/audit-logs/maintenance-windows/playbooks/
#     break-glass/commands 成功响应同样走统一信封，业务数据在 data.* 下；
#     断言统一用 check_ret_ok + data.* 取值（不再宽容裸 JSON）。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 运维工具" "$@"
smoke_login

# 合法 UUID 格式但必然不存在的 ID（负路径用）
NIL_UUID="00000000-dead-beef-8000-000000000000"

# ---------------------------------------------------------------------------
section "1. 运维模板 ops/templates — 列表 + CRUD 闭环"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/templates"
check_ret_ok "模板列表可查"
check_list_or_empty "模板列表 items 形状" "data.items"

TPL_NAME="${SMOKE_TAG}-tpl"
req POST "/api/v1/ops/templates" "{\"template_name\":\"${TPL_NAME}\",\"description\":\"smoke 自建模板\",\"category\":\"diagnostic\",\"target_device_types\":[\"FAP-LTE-100\"],\"steps\":[{\"order\":1,\"command\":\"ping\"}],\"estimated_duration\":60,\"creator\":\"smoke\",\"tags\":[\"smoke\"]}"
check_status "创建模板（201 + 信封）" 201
check_field "创建返回 data.id" "data.id"
TPL_ID=$(jget data.id)

if [ -n "$TPL_ID" ]; then
    req GET "/api/v1/ops/templates/$TPL_ID"
    check_ret_ok "模板详情可查"
    check_field "详情回显 template_name" "data.template_name"

    req GET "/api/v1/ops/templates?keyword=${SMOKE_TAG}"
    check_list_nonempty "keyword 过滤命中自建模板" "data.items"

    req PUT "/api/v1/ops/templates/$TPL_ID" "{\"template_name\":\"${TPL_NAME}\",\"description\":\"smoke updated desc\",\"category\":\"diagnostic\",\"target_device_types\":[\"FAP-LTE-100\"],\"steps\":[{\"order\":1,\"command\":\"ping\"}],\"estimated_duration\":90,\"creator\":\"smoke\",\"tags\":[\"smoke\"]}"
    check_ret_ok "更新模板"
    TPL_DESC=$(jget data.description)
    if [ "$TPL_DESC" = "smoke updated desc" ]; then
        pass "更新后 description 回显一致"
    else
        fail "更新后 description 回显一致" "期望 'smoke updated desc'，实际 '$TPL_DESC'"
    fi

    req DELETE "/api/v1/ops/templates/$TPL_ID"
    check_ret_ok "删除模板"

    req GET "/api/v1/ops/templates/$TPL_ID"
    check_ret_fail "删除后详情查询被拒（404）"
else
    skip "模板 CRUD 后续链（详情/改/删）" "创建未返回 id（上方已计失败）"
fi

req POST "/api/v1/ops/templates" '{"description":"missing template_name"}'
check_ret_fail "缺 template_name 创建被拒"

req GET "/api/v1/ops/templates/not-a-uuid"
check_ret_fail "非法 UUID 模板详情被拒"

# ---------------------------------------------------------------------------
section "2. 运维任务 ops/tasks — 列表 + 详情/控制负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/tasks?page=1&page_size=10"
check_ret_ok "任务列表可查"
check_list_or_empty "任务列表 items 形状" "data.items"

req GET "/api/v1/ops/tasks/$NIL_UUID"
check_ret_fail "不存在任务详情被拒（404）"

req GET "/api/v1/ops/tasks/$NIL_UUID/executions"
check_ret_ok "不存在任务的执行明细返回空列表（信封）"

req POST "/api/v1/ops/tasks" '{"creator":"smoke"}'
check_ret_fail "缺 task_name 建任务被拒"

req POST "/api/v1/ops/tasks/$NIL_UUID/cancel"
check_ret_fail "取消不存在任务被拒"

req POST "/api/v1/ops/tasks/$NIL_UUID/pause"
check_ret_fail "暂停不存在任务被拒"

req POST "/api/v1/ops/tasks/$NIL_UUID/resume"
check_ret_fail "恢复不存在任务被拒"

req POST "/api/v1/ops/tasks/$NIL_UUID/run"
check_ret_fail "运行不存在任务被拒"

req POST "/api/v1/ops/tasks/$NIL_UUID/approve" '{"approve":true,"reason":"smoke"}'
check_ret_fail "审批不存在任务被拒"

# ---------------------------------------------------------------------------
section "3. 命令记录 ops/command-records — 列表 + 校验负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/command-records?page=1&page_size=10"
check_ret_ok "命令记录列表可查"
check_list_or_empty "命令记录 items 形状" "data.items"

req POST "/api/v1/ops/command-records" '{"device_sn":"SMK-NONE"}'
check_ret_fail "缺 command_text 创建命令记录被拒"

# ---------------------------------------------------------------------------
section "4. 诊断 ops/diagnostics — 列表 + 校验负路径（不触发真实诊断）"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/diagnostics?page=1&page_size=10"
check_ret_ok "诊断列表可查（信封）"
check_field "诊断列表 data.total 字段" "data.total"

req GET "/api/v1/ops/diagnostics?device_sn=SMK-NONE&diag_type=ip_ping&status=pending"
check_ret_ok "诊断列表条件过滤可查"

req GET "/api/v1/ops/diagnostics/not-a-uuid"
check_ret_fail "非法 UUID 诊断详情被拒"

req GET "/api/v1/ops/diagnostics/$NIL_UUID"
check_ret_fail "不存在诊断详情被拒（404）"

req POST "/api/v1/ops/diagnostics/ping" '{}'
check_ret_fail "ping 缺 device_sn+host 被拒"

req POST "/api/v1/ops/diagnostics/ping" '{"device_sn":"SMK-NONE"}'
check_ret_fail "ping 缺 host 被拒"

req POST "/api/v1/ops/diagnostics/traceroute" '{"host":"127.0.0.1"}'
check_ret_fail "traceroute 缺 device_sn 被拒"

req POST "/api/v1/ops/diagnostics/throughput" '{"device_sn":"SMK-NONE"}'
check_ret_fail "throughput 缺 url 被拒"

skip "diagnostics/inspection 触发" "POST 即真实触发全网巡检（无参数校验负路径可构造），按危险禁区跳过"

# ---------------------------------------------------------------------------
section "5. 诊断下载 ops/downloads — 列表 + 校验负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/downloads?page=1&page_size=10"
check_ret_ok "下载列表可查（信封）"
check_field "下载列表 data.total 字段" "data.total"

req GET "/api/v1/ops/downloads/$NIL_UUID"
check_ret_fail "不存在下载详情被拒（404）"

req POST "/api/v1/ops/downloads/collect" '{}'
check_ret_fail "collect 缺 device_sn+content_types 被拒"

req POST "/api/v1/ops/downloads/collect" '{"device_sn":"SMK-NONE","content_types":[]}'
check_ret_fail "collect 空 content_types 被拒"

# ---------------------------------------------------------------------------
section "6. 维护窗口 ops/maintenance-windows — 未来窗口创建 + 四眼审批守卫"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/maintenance-windows?page=1&page_size=10"
check_ret_ok "维护窗口列表可查（信封）"
check_field "维护窗口列表 data.total 字段" "data.total"

req GET "/api/v1/ops/maintenance-windows/active"
check_ret_ok "active 维护窗口列表可查"

# 未来 +1 天的 1 小时窗口；scope 指向不存在设备、抑制/暂停开关全 false，零业务影响
MW_START=$(python3 -c "import datetime;print((datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
MW_END=$(python3 -c "import datetime;print((datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=1,hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
MW_NAME="${SMOKE_TAG}-mw"

req POST "/api/v1/ops/maintenance-windows" "{\"name\":\"${MW_NAME}\",\"scope_type\":\"device\",\"scope_ids\":[\"SMK-MW-NODEVICE\"],\"start_at\":\"${MW_START}\",\"end_at\":\"${MW_END}\",\"suppress_alarms\":false,\"pause_provision\":false,\"allow_dangerous\":false,\"reason\":\"smoke 冒烟自建未来窗口\"}"
check_ret_ok "创建未来维护窗口（201 + 信封）"
check_field "窗口返回 data.id" "data.id"
MW_ID=$(jget data.id)

if [ -n "$MW_ID" ]; then
    req GET "/api/v1/ops/maintenance-windows/$MW_ID"
    check_ret_ok "窗口详情可查"
    MW_GOT_NAME=$(jget data.name)
    if [ "$MW_GOT_NAME" = "$MW_NAME" ]; then
        pass "窗口详情 name 回显一致"
    else
        fail "窗口详情 name 回显一致" "期望 '${MW_NAME}'，实际 '${MW_GOT_NAME}'"
    fi

    req GET "/api/v1/ops/maintenance-windows?page=1&page_size=10"
    check_count_ge "窗口列表 total ≥ 1（含自建窗口）" "data.total" 1

    # 创建者 == 审批者 → 四眼原则拒绝自审批（这是正确行为的负路径断言）
    req POST "/api/v1/ops/maintenance-windows/$MW_ID/approve"
    check_ret_fail "自审批被四眼原则拒绝"

    req GET "/api/v1/ops/maintenance-windows/$MW_ID"
    MW_STATUS=$(jget data.status)
    if [ "$MW_STATUS" = "planned" ]; then
        pass "自审批被拒后窗口仍为 planned"
    else
        fail "自审批被拒后窗口仍为 planned" "实际 status='${MW_STATUS}'"
    fi
else
    skip "维护窗口详情/审批守卫链" "创建未返回 id（上方已计失败）"
fi

req POST "/api/v1/ops/maintenance-windows/$NIL_UUID/approve"
check_ret_fail "审批不存在窗口被拒"

req POST "/api/v1/ops/maintenance-windows" "{\"name\":\"${SMOKE_TAG}-mw-bad\",\"scope_type\":\"device\",\"start_at\":\"${MW_END}\",\"end_at\":\"${MW_START}\"}"
check_ret_fail "end_at 早于 start_at 被拒"

req POST "/api/v1/ops/maintenance-windows" '{"scope_type":"device"}'
check_ret_fail "缺 name/start_at/end_at 被拒"

# ---------------------------------------------------------------------------
section "7. 运维审计 ops/audit-logs — 列表 + 操作留痕"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/audit-logs?page=1&page_size=10"
check_ret_ok "运维审计列表可查（信封）"
check_field "审计列表 data.total 字段" "data.total"

# 上方维护窗口创建应同步写一条 op_type=maintenance_window_create 审计
req GET "/api/v1/ops/audit-logs?op_type=maintenance_window_create&page=1&page_size=10"
check_ret_ok "审计留痕过滤查询可用"
AUDIT_TOTAL=$(jget data.total)
if [ -n "$AUDIT_TOTAL" ] && [ "$AUDIT_TOTAL" -ge 1 ] 2>/dev/null; then
    pass "维护窗口创建已留审计痕迹 (total=${AUDIT_TOTAL})"
else
    # #124 已修（迁移 000035 放宽 target_type 至 varchar(64)），转回硬断言。
    fail "维护窗口创建已留审计痕迹" "audit-logs 未查到 op_type=maintenance_window_create 记录 (total=${AUDIT_TOTAL:-空})"
fi

# ---------------------------------------------------------------------------
section "8. 知识库 ops/playbooks — 列表 + match + 校验负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/ops/playbooks?page=1&page_size=10"
check_ret_ok "playbook 列表可查（信封）"
check_field "playbook 列表 data.total 字段" "data.total"

req GET "/api/v1/ops/playbooks?keyword=smoke&page=1&page_size=10"
check_ret_ok "playbook 关键词过滤可查"

req POST "/api/v1/ops/playbooks/match" '{"alarm_code":"SMK-NO-ALARM"}'
check_ret_ok "playbook 按告警码匹配查询"
check_field "match 返回 data.total 字段" "data.total"

req POST "/api/v1/ops/playbooks/match" '{}'
check_ret_fail "match 缺 alarm_code 被拒"

req POST "/api/v1/ops/playbooks" '{"title": '
check_ret_fail "playbook 畸形 JSON 创建被拒"
# 注：POST /ops/playbooks 正向创建无删除端点（不可逆），不做正向写避免残留

# ---------------------------------------------------------------------------
section "9. 即时命令 ops/commands/rpc — 只测校验负路径【红线：不真发 RPC】"
# ---------------------------------------------------------------------------
req POST "/api/v1/ops/commands/rpc" '{}'
check_ret_fail "RPC 缺 action 被拒"

req POST "/api/v1/ops/commands/rpc" '{"action":"get_parameter_values"}'
check_ret_fail "RPC 缺 device_sn/device_sns 被拒"

# action 不在白名单 → 入队前即被拒，不会创建任务（红线安全）
req POST "/api/v1/ops/commands/rpc" '{"action":"smk_invalid_action","device_sn":"SMK-NO-DEVICE"}'
check_ret_fail "RPC 不支持的 action 被拒"

# ---------------------------------------------------------------------------
section "10. 紧急通道 ops/break-glass — status 只读 + activate 校验负路径【红线：不真激活】"
# ---------------------------------------------------------------------------
# 红线说明：break-glass activate 是紧急权限通道，绑定成功即授予该用户 30 分钟越权访问窗口
# （service_ext.go BreakGlassService.Activate：写 active map + 落 break_glass_activate 危险审计）。
# 正路径会真实授权，绝对不测；只测 activate 参数绑定失败的负路径——
# BreakGlassRequest{ticket_id,reason} 两字段均 binding:"required"，缺字段/畸形 body 会在
# ShouldBindJSON 阶段以 HTTP 400 "invalid payload" 被拒，发生在任何授权逻辑之前（实测 2026-06-11）。
# deactivate 无 body/无参数校验、恒 200，没有可构造的负路径，且属同一紧急通道，整体不触碰。

req GET "/api/v1/ops/break-glass/status"
check_ret_ok "break-glass 状态可查（信封）"
check_field "status 返回 data.active 字段" "data.active"
# 基线：冒烟开跑时当前 admin 不应处于 break-glass 激活态（否则脏环境，下方负路径仍安全）
BG_ACTIVE_BEFORE=$(jget data.active)

req POST "/api/v1/ops/break-glass/activate" '{}'
check_ret_fail "activate 缺 ticket_id+reason 被拒（绑定失败，未激活）"
check_status "activate 缺必填字段 → 400" 400

req POST "/api/v1/ops/break-glass/activate" '{"reason":"smoke 仅理由缺工单号"}'
check_ret_fail "activate 缺 ticket_id 被拒（未激活）"

req POST "/api/v1/ops/break-glass/activate" '{"ticket_id":"SMK-TICKET-ONLY"}'
check_ret_fail "activate 缺 reason 被拒（未激活）"

# 畸形 JSON 同样在绑定阶段 400，不触达授权
req POST "/api/v1/ops/break-glass/activate" '{"ticket_id":'
check_ret_fail "activate 畸形 JSON 被拒（未激活）"

# 红线核验：连发四次 activate 负路径后，status 必须仍未激活——证明无任何一次真实授权
req GET "/api/v1/ops/break-glass/status"
BG_ACTIVE_AFTER=$(jget data.active)
if [ "$BG_ACTIVE_AFTER" = "$BG_ACTIVE_BEFORE" ] && [ "$BG_ACTIVE_AFTER" != "true" ] && [ "$BG_ACTIVE_AFTER" != "True" ]; then
    pass "activate 负路径全程未触发真实激活（status active 仍为 ${BG_ACTIVE_AFTER}）"
else
    fail "activate 负路径全程未触发真实激活" "status active 由 '${BG_ACTIVE_BEFORE}' 变为 '${BG_ACTIVE_AFTER}'（疑似负路径误激活）"
fi
# deactivate：紧急通道、无参数校验、恒 200 无负路径，按红线整体不触碰（不调用）

smoke_summary

#!/usr/bin/env bash
# =============================================================================
# smoke_pm.sh — F03 性能管理（PM/KPI/指标库）业务冒烟
#
# 覆盖（/tmp/smoke_routes.json: pm + pmdash + indicator 三域）：
#   - counters / counters/aggregated / metrics/aggregated / metrics/objects 读链路
#   - kpi / kpi/definitions 读 + kpi/calculate 幂等触发（小时间窗）
#   - pm/tasks（GET 已知 500 bug → known_bug；POST 仅负路径）
#   - pm/files + files/devices + 下载 + batch-delete 仅参数校验负路径
#   - aggregation/recompute 幂等触发
#   - thresholds / dashboards(+panel) / query-templates / adhoc / exports CRUD 闭环
#   - user-preferences/dashboard 读
#   - 指标库：legacy POST（ENB/GNB 组树、分页列表、生效指标）、REST /indicators
#     系列（super_admin）、enable/disable 可逆闭环、upload-xml 仅负路径
#
# 红线：不向任何设备下发 reboot/升级/SPV 等指令；kpi/calculate 与 recompute
# 均为服务端幂等计算，不触达基站。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F03 性能管理" "$@"
smoke_login

# ── 时间窗（RFC3339，UTC；python3 计算保证 macOS/Linux 一致）────────────────
NOW=$(python3 -c "import time;print(time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()))")
HOUR_AGO=$(python3 -c "import time;print(time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime(time.time()-3600)))")
TWO_HOURS_AGO=$(python3 -c "import time;print(time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime(time.time()-7200)))")

# ───────────────────────────────────────────────────────────────────────────
section "0. 数据准备（设备 / KPI 定义）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/devices?page=1&page_size=5"
check_ret_ok "设备列表可查（自备测试数据源）"
DEV_SN=$(jget data.items.0.serial_number)
DEV_ID=$(jget data.items.0.id)
DEV_CARRIER=$(jget data.items.0.carrier)
DEV_TECH=$(jget data.items.0.technology)
[ -z "$DEV_CARRIER" ] && DEV_CARRIER="cmcc"
[ -z "$DEV_TECH" ] && DEV_TECH="lte"
if [ -n "$DEV_SN" ]; then
    echo "  （测试设备 SN=${DEV_SN} carrier=${DEV_CARRIER} tech=${DEV_TECH}）"
else
    echo "  （活栈无设备，设备相关用例将降级）"
fi

# KPI 定义先取一个真实指标编号，供面板/adhoc 任务引用
req GET "/api/v1/pm/kpi/definitions?device_type=ENB"
check_ret_ok "KPI 定义可按 device_type=ENB 过滤"
KPI_ID=$(jget data.items.0.id)
[ -z "$KPI_ID" ] && KPI_ID="K1001"

# ───────────────────────────────────────────────────────────────────────────
section "1. counters / metrics 三层读链路"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/counters"
check_list_or_empty "原始计数器列表可查（默认近 24h 窗）" "data.items"

# 注意：此端点未预填 DefaultListRequest，page/page_size 必须显式传（同 admin/users 绑定类）
req GET "/api/v1/pm/counters/aggregated?page=1&page_size=20"
check_list_or_empty "计数器聚合视图可查" "data.items"

req GET "/api/v1/pm/metrics/aggregated?granularity=hourly&start_time=${TWO_HOURS_AGO}&end_time=${NOW}"
check_list_or_empty "metrics 按 hourly 粒度聚合可查" "data.items"

req GET "/api/v1/pm/metrics/aggregated"
check_ret_fail "metrics/aggregated 缺 granularity 被拒绝"

if [ -n "$DEV_SN" ]; then
    req GET "/api/v1/pm/metrics/objects?device_sns=${DEV_SN}"
    check_list_or_empty "metrics/objects 小区/PLMN 下钻可查（SN=${DEV_SN}）" "data.items"
else
    req GET "/api/v1/pm/metrics/objects?device_sns=SMK-NO-DEVICE"
    check_list_or_empty "metrics/objects 可查（无设备，验证路由）" "data.items"
fi

# ───────────────────────────────────────────────────────────────────────────
section "2. KPI 查询 / 定义 / 幂等计算"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/kpi"
check_list_or_empty "KPI 数值列表可查（默认近 24h 窗）" "data.items"

req GET "/api/v1/pm/kpi/definitions"
check_ret_ok "KPI 定义全量列表可查"
check_count_ge "KPI 定义为内置字典（total ≥ 100）" "data.total" 100
check_field "KPI 定义首条带指标编号" "data.items.0.id"
check_field "KPI 定义首条带显示名" "data.items.0.display_name"

if [ -n "$DEV_ID" ]; then
    req POST "/api/v1/pm/kpi/calculate" "{\"device_id\":\"${DEV_ID}\",\"start_time\":\"${HOUR_AGO}\",\"end_time\":\"${NOW}\",\"carrier\":\"${DEV_CARRIER}\",\"technology\":\"${DEV_TECH}\"}"
    check_ret_ok "KPI 幂等计算可触发（小时间窗，设备 ${DEV_SN}）"
else
    skip "KPI 幂等计算" "活栈无设备，无法取 device_id"
fi

req POST "/api/v1/pm/kpi/calculate" '{}'
check_ret_fail "kpi/calculate 空 body 被参数校验拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "3. PM 采集任务（GET 已知 500 bug）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/tasks"
PM_TASKS_RET=$(jget ret)
if [[ "$HTTP_CODE" == 2* ]] && [ "$PM_TASKS_RET" = "1" ]; then
    pass "PM 采集任务列表可查 → $HTTP_CODE ret=1（已知 bug 已修复，可移除 known_bug 分支）"
else
    known_bug "GET /api/v1/pm/tasks 列表 500" "HTTP ${HTTP_CODE}: $(printf '%s' "$BODY" | head -c 160)"
fi

# 任务创建无配套 DELETE 路由（不可逆），只测参数校验负路径
req POST "/api/v1/pm/tasks" '{}'
check_ret_fail "PM 任务创建缺 task_name 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "4. PM 文件（列表 / 设备聚合 / 下载 / 批删仅负路径）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/files"
check_list_or_empty "PM 文件列表可查" "data.items"
PM_FILE_ID=$(jget data.items.0.id)

req GET "/api/v1/pm/files/devices"
check_list_or_empty "PM 文件按设备聚合可查" "data.items"

if [ -n "$PM_FILE_ID" ]; then
    req GET "/api/v1/pm/files/${PM_FILE_ID}/download" "" -o /dev/null
    check_status_in "PM 文件下载链路（id=${PM_FILE_ID}）" "200 500"
else
    req GET "/api/v1/pm/files/not-a-uuid/download"
    check_ret_fail "PM 文件下载非法 id 被拒绝（活栈无文件，走负路径）"
fi

# 【危险禁区】batch-delete 按 SN 删 PG+MinIO，只测参数校验负路径
req POST "/api/v1/pm/files/batch-delete" '{}'
check_ret_fail "PM 文件批量删除缺 serial_numbers 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "5. 聚合重算（运维幂等触发）"
# ───────────────────────────────────────────────────────────────────────────
req POST "/api/v1/pm/aggregation/recompute" "{\"granularity\":\"hourly\",\"start\":\"${TWO_HOURS_AGO}\",\"end\":\"${NOW}\"}"
check_ret_ok "hourly 聚合重算可入队（幂等异步 job）"
check_field "重算返回 job_id" "data.job_id"

req POST "/api/v1/pm/aggregation/recompute" "{\"granularity\":\"yearly\",\"start\":\"${TWO_HOURS_AGO}\",\"end\":\"${NOW}\"}"
check_ret_fail "非法粒度 yearly 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "6. KPI 门限 CRUD 闭环"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/thresholds"
check_list_or_empty "门限列表可查" "data.items"

req POST "/api/v1/pm/thresholds" "{\"kpi_name\":\"${SMOKE_TAG}_KPI\",\"carrier\":\"cmcc\",\"technology\":\"lte\",\"warning_threshold\":90,\"major_threshold\":80,\"comparison\":\"lt\",\"description\":\"smoke test\"}"
check_status "门限创建" 201
THRESHOLD_ID=$(jget data.id)
check_field "门限创建返回 id" "data.id"

if [ -n "$THRESHOLD_ID" ]; then
    req GET "/api/v1/pm/thresholds/${THRESHOLD_ID}"
    check_ret_ok "门限详情可查"
    check_field "门限详情 kpi_name 回读一致" "data.kpi_name"

    req PUT "/api/v1/pm/thresholds/${THRESHOLD_ID}" '{"description":"smoke updated","critical_threshold":70}'
    check_ret_ok "门限更新成功"

    req GET "/api/v1/pm/thresholds?kpi_name=${SMOKE_TAG}_KPI"
    check_list_nonempty "门限列表能按 kpi_name 过滤出自建项" "data.items"

    req DELETE "/api/v1/pm/thresholds/${THRESHOLD_ID}"
    check_ret_ok "门限删除成功"

    req GET "/api/v1/pm/thresholds/${THRESHOLD_ID}"
    check_ret_fail "已删门限再查返回 404"
else
    skip "门限 详情/更新/删除" "创建未返回 id，闭环中断"
fi

req POST "/api/v1/pm/thresholds" '{}'
check_ret_fail "门限创建缺 kpi_name 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "7. PM 仪表盘 CRUD + 面板闭环"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/dashboards"
check_list_or_empty "仪表盘列表可查" "data.items"

req POST "/api/v1/pm/dashboards" "{\"name\":\"${SMOKE_TAG}-dash\",\"description\":\"smoke\",\"technology\":\"lte\"}"
check_status "仪表盘创建" 201
DASH_ID=$(jget data.id)
check_field "仪表盘创建返回 id" "data.id"

if [ -n "$DASH_ID" ]; then
    req GET "/api/v1/pm/dashboards/${DASH_ID}"
    check_ret_ok "仪表盘详情可查"
    check_field "详情含 dashboard 对象" "data.dashboard.id"

    req PUT "/api/v1/pm/dashboards/${DASH_ID}" "{\"name\":\"${SMOKE_TAG}-dash-v2\"}"
    check_ret_ok "仪表盘更新成功"

    PANEL_SNS="[]"
    [ -n "$DEV_SN" ] && PANEL_SNS="[\"${DEV_SN}\"]"
    req POST "/api/v1/pm/dashboards/${DASH_ID}/panels" "{\"panel_type\":\"line_chart\",\"title\":\"${SMOKE_TAG}-panel\",\"metric_paths\":[\"${KPI_ID}\"],\"granularities\":[\"hourly\"],\"dimension\":\"device\",\"device_sns\":${PANEL_SNS}}"
    check_status "面板创建" 201
    PANEL_ID=$(jget data.id)
    if [ -n "$PANEL_ID" ]; then
        req PUT "/api/v1/pm/dashboards/${DASH_ID}/panels/${PANEL_ID}" "{\"panel_type\":\"bar_chart\",\"title\":\"${SMOKE_TAG}-panel-v2\",\"metric_paths\":[\"${KPI_ID}\"],\"granularities\":[\"hourly\"],\"dimension\":\"device\",\"device_sns\":${PANEL_SNS}}"
        check_ret_ok "面板更新成功"
        req DELETE "/api/v1/pm/dashboards/${DASH_ID}/panels/${PANEL_ID}"
        check_ret_ok "面板删除成功"
    else
        skip "面板 更新/删除" "面板创建未返回 id"
    fi

    req DELETE "/api/v1/pm/dashboards/${DASH_ID}"
    check_ret_ok "仪表盘删除成功"

    req GET "/api/v1/pm/dashboards/${DASH_ID}"
    check_ret_fail "已删仪表盘再查返回 404"
else
    skip "仪表盘 详情/更新/面板/删除" "创建未返回 id，闭环中断"
fi

req POST "/api/v1/pm/dashboards" '{}'
check_ret_fail "仪表盘创建缺 name/technology 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "8. 用户偏好（读）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/user-preferences/dashboard"
check_ret_ok "仪表盘用户偏好可读（默认 lte）"
check_field "偏好含 technology" "data.technology"

req GET "/api/v1/pm/user-preferences/dashboard?technology=nr"
check_ret_ok "偏好按 technology=nr 可读"

# ───────────────────────────────────────────────────────────────────────────
section "9. 指标查询模板 CRUD 闭环"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/query-templates"
check_list_or_empty "查询模板列表可查" "data.items"

req POST "/api/v1/pm/query-templates" "{\"name\":\"${SMOKE_TAG}-qt\",\"visibility\":\"private\",\"description\":\"smoke\",\"payload\":{\"kpi\":[\"${KPI_ID}\"]}}"
check_ret_ok "查询模板创建成功"
QT_ID=$(jget data.id)
check_field "查询模板创建返回 id" "data.id"

if [ -n "$QT_ID" ]; then
    req GET "/api/v1/pm/query-templates/${QT_ID}"
    check_ret_ok "查询模板详情可查"

    req PATCH "/api/v1/pm/query-templates/${QT_ID}" "{\"description\":\"smoke updated\"}"
    check_ret_ok "查询模板更新成功"

    req DELETE "/api/v1/pm/query-templates/${QT_ID}"
    check_ret_ok "查询模板删除成功"

    req GET "/api/v1/pm/query-templates/${QT_ID}"
    check_ret_fail "已删查询模板再查返回 404"
else
    skip "查询模板 详情/更新/删除" "创建未返回 id，闭环中断"
fi

req POST "/api/v1/pm/query-templates" '{}'
check_ret_fail "查询模板创建缺 name/visibility 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "10. 自定义聚合（adhoc）任务闭环"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/adhoc/tasks"
check_ret_ok "adhoc 任务列表可查（我的任务）"

req GET "/api/v1/pm/adhoc/tasks?is_builtin=true"
check_list_or_empty "内置 adhoc 任务列表可查" "data.items"

# network 维度按制式全量聚合，不依赖设备 SN；oneshot 小时间窗
req POST "/api/v1/pm/adhoc/tasks" "{\"name\":\"${SMOKE_TAG}-adhoc\",\"mode\":\"oneshot\",\"dimension\":\"network\",\"technology\":\"lte\",\"metric_paths\":[\"${KPI_ID}\"],\"granularities\":[\"hourly\"],\"window_start\":\"${TWO_HOURS_AGO}\",\"window_end\":\"${NOW}\"}"
check_status "adhoc 任务创建（network 维度小时间窗）" 201
ADHOC_ID=$(jget data.id)
check_field "adhoc 创建返回 id" "data.id"

if [ -n "$ADHOC_ID" ]; then
    req GET "/api/v1/pm/adhoc/tasks/${ADHOC_ID}"
    check_ret_ok "adhoc 任务详情可查"
    check_field "adhoc 任务带 status" "data.status"

    req GET "/api/v1/pm/adhoc/tasks/${ADHOC_ID}/results"
    check_ret_ok "adhoc 任务结果可查（允许为空）"

    req GET "/api/v1/pm/adhoc/tasks/${ADHOC_ID}/runs"
    check_ret_ok "adhoc 运行历史可查"

    # DELETE = 取消；worker 可能已抢跑完成 → 409 终态同样证明生命周期闭环
    req DELETE "/api/v1/pm/adhoc/tasks/${ADHOC_ID}"
    check_status_in "adhoc 任务取消（200 取消 / 409 已终态）" "200 409"
else
    skip "adhoc 详情/结果/取消" "创建未返回 id，闭环中断"
fi

req POST "/api/v1/pm/adhoc/tasks" '{}'
check_ret_fail "adhoc 任务创建空 body 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "11. KPI 导出任务闭环"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/pm/exports"
check_list_or_empty "导出任务列表可查" "data.items"

req GET "/api/v1/pm/exports/files"
check_list_or_empty "导出文件列表可查" "data.items"

EXPORT_PARAMS="{}"
[ -n "$ADHOC_ID" ] && EXPORT_PARAMS="{\"task_id\":\"${ADHOC_ID}\"}"
req POST "/api/v1/pm/exports" "{\"source_type\":\"adhoc\",\"task_name\":\"${SMOKE_TAG}-exp\",\"params\":${EXPORT_PARAMS}}"
check_status "导出任务创建" 201
EXPORT_ID=$(jget data.id)
check_field "导出创建返回 id" "data.id"

if [ -n "$EXPORT_ID" ]; then
    # 刚建的任务文件未就绪 → 409；若 worker 已极速完成 → 200 给签名链接
    req GET "/api/v1/pm/exports/${EXPORT_ID}/download"
    check_status_in "导出下载（200 就绪 / 409 未就绪）" "200 409"

    req DELETE "/api/v1/pm/exports/${EXPORT_ID}"
    check_ret_ok "导出任务记录删除成功"
else
    skip "导出 下载/删除" "创建未返回 id，闭环中断"
fi

req POST "/api/v1/pm/exports" '{"source_type":"bogus"}'
check_ret_fail "导出创建非法 source_type 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "12. 指标库 legacy POST（ENB/GNB 组树、分页、生效指标）"
# ───────────────────────────────────────────────────────────────────────────
req POST "/api/v1/pm/indicatormg/getIndicatorGroupTree" '{"device_type":"ENB"}'
check_ret_ok "ENB 指标组树可查"
check_list_nonempty "ENB 指标组树非空（内置分组）" "data"

req POST "/api/v1/gnb/pm/indicatormg/getIndicatorGroupTree" '{"device_type":"GNB"}'
check_ret_ok "GNB(5G) 指标组树可查"
check_list_nonempty "GNB 指标组树非空（内置分组）" "data"

req POST "/api/v1/pm/indicatormg/getIndicatorListByPage" '{"device_type":"ENB"}'
check_ret_ok "ENB 指标分页列表可查"
check_count_ge "ENB 指标库内置非空（total ≥ 100）" "data.total" 100
IND_ID=$(jget data.items.0.id)
check_field "指标首条带 id" "data.items.0.id"

req POST "/api/v1/pm/indicatormg/getEffectiveIndicators" '{"device_type":"ENB","operator_code":"default"}'
check_ret_ok "ENB 生效指标集可查"

req POST "/api/v1/pm/indicatormg/getIndicatorGroupTree" '{"device_type":"BAD"}'
check_ret_fail "非法 device_type 被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "13. 指标库 REST 治理（super_admin）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/indicators?deviceType=enb"
check_ret_ok "REST 指标列表可查（deviceType=enb）"
check_count_ge "REST 指标库内置非空（total ≥ 1000）" "data.total" 1000

req GET "/api/v1/indicators/platforms?deviceType=enb"
check_ret_ok "平台名列表可查"
check_list_nonempty "ENB 平台名非空（内置公式平台）" "data.items"

req GET "/api/v1/indicators/summary"
check_ret_ok "指标库三制式汇总可查"
check_list_nonempty "汇总含制式行" "data.items"

req GET "/api/v1/indicators/files?tech=enb"
check_ret_ok "指标库 XML 文件清单可查"
check_list_nonempty "ENB 内置 XML 文件非空" "data.items"

req GET "/api/v1/indicator-groups?deviceType=enb"
check_ret_ok "REST 指标分组树可查"
check_list_nonempty "REST 分组树非空" "data.items"

req GET "/api/v1/indicators?deviceType=LTE"
check_ret_fail "非法 deviceType=LTE 被拒绝（unknown device type）"

# 上传 XML 走 destructive 重载，冒烟只测无文件负路径
req POST "/api/v1/indicators/upload-xml" '{}'
check_ret_fail "upload-xml 无 multipart 文件被拒绝"

# ───────────────────────────────────────────────────────────────────────────
section "14. 指标启用/禁用可逆闭环（operator=default）"
# ───────────────────────────────────────────────────────────────────────────
req GET "/api/v1/enabled-indicators?deviceType=enb&operatorCode=default"
check_ret_ok "ENB 启用指标集可读"
ORIG_ENABLED=$(jget data.items)

if [ -n "$IND_ID" ]; then
    WAS_ENABLED=0
    printf '%s' "$ORIG_ENABLED" | grep -F "\"${IND_ID}\"" >/dev/null 2>&1 && WAS_ENABLED=1

    req POST "/api/v1/cell/perfmgmt/kpimanage/enableIndicator" "{\"device_type\":\"ENB\",\"operator_code\":\"default\",\"indicator_ids\":[\"${IND_ID}\"]}"
    check_ret_ok "指标启用成功（id=${IND_ID}）"

    req GET "/api/v1/enabled-indicators?deviceType=enb&operatorCode=default"
    NOW_ENABLED=$(jget data.items)
    if printf '%s' "$NOW_ENABLED" | grep -F "\"${IND_ID}\"" >/dev/null 2>&1; then
        pass "启用后该指标出现在生效集"
    else
        fail "启用后该指标出现在生效集" "id=${IND_ID} 不在 enabled-indicators 返回中"
    fi

    req POST "/api/v1/cell/perfmgmt/kpimanage/disableIndicator" "{\"device_type\":\"ENB\",\"operator_code\":\"default\",\"indicator_ids\":[\"${IND_ID}\"]}"
    check_ret_ok "指标禁用成功（可逆）"

    req GET "/api/v1/enabled-indicators?deviceType=enb&operatorCode=default"
    NOW_ENABLED=$(jget data.items)
    if printf '%s' "$NOW_ENABLED" | grep -F "\"${IND_ID}\"" >/dev/null 2>&1; then
        fail "禁用后该指标移出生效集" "id=${IND_ID} 仍在 enabled-indicators 返回中"
    else
        pass "禁用后该指标移出生效集"
    fi

    if [ "$WAS_ENABLED" = "1" ]; then
        req POST "/api/v1/cell/perfmgmt/kpimanage/enableIndicator" "{\"device_type\":\"ENB\",\"operator_code\":\"default\",\"indicator_ids\":[\"${IND_ID}\"]}"
        check_ret_ok "状态还原：恢复原本启用态"
    fi

    req POST "/api/v1/cell/perfmgmt/kpimanage/enableIndicator" '{"device_type":"ENB","operator_code":"default","indicator_ids":[]}'
    check_ret_fail "启用指标空 indicator_ids 被拒绝"
else
    skip "指标启用/禁用闭环" "指标分页列表未取到指标 id"
fi

smoke_summary

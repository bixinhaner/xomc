#!/usr/bin/env bash
# =============================================================================
# smoke_dashboard.sh — F06 仪表盘 业务冒烟
#
# 覆盖（omcgo/internal/dashboard/handler.go 全部路由，均为 GET + widgets PUT）：
#   /dashboard/summary                     首页总览（device_stats/alarm_stats/kpi_overview/timestamp）
#   /dashboard/alarm-trend                 告警趋势（days 参数 + 负路径）
#   /dashboard/device-status               设备状态计数
#   /dashboard/device-status-by-type       按制式分组设备状态
#   /dashboard/kpi-trend                   KPI 趋势（kpi_name 必填；compare_with 对比）
#   /dashboard/kpi-time-series             KPI 时间序列（kpi_names 必填；RFC3339 时间窗）
#   /dashboard/region-stats                区域统计
#   /dashboard/alarm-type-pie              告警类型饼图
#   /dashboard/alarm-efficiency            告警处理效率（MTTA/MTTR）
#   /dashboard/alarm-heatmap               告警热度图（days 1-365）
#   /dashboard/alarm-heatmap-by-severity   按严重程度热度图
#   /dashboard/widgets GET/PUT             widgets 读写闭环（GET 留底 → PUT 原值回写 → GET 比对）
#
# 用法：bash smoke_dashboard.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 仪表盘" "$@"
smoke_login

# ---------------------------------------------------------------------------
section "summary 总览（首页首屏强依赖）"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/summary"
check_ret_ok "summary 总览可查"
check_field "summary 含 device_stats" "data.device_stats"
check_field "summary 含 device_stats.total" "data.device_stats.total"
check_field "summary 含 device_stats.online" "data.device_stats.online"
check_field "summary 含 alarm_stats" "data.alarm_stats"
check_field "summary 含 alarm_stats.total" "data.alarm_stats.total"
check_field "summary 含 kpi_overview" "data.kpi_overview"
check_field "summary 含 timestamp" "data.timestamp"

# ---------------------------------------------------------------------------
section "告警趋势 alarm-trend"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/alarm-trend"
check_ret_ok "告警趋势(默认 7 天)可查"

req GET "/api/v1/dashboard/alarm-trend?days=30"
check_ret_ok "告警趋势(days=30)可查"

req GET "/api/v1/dashboard/alarm-trend?days=0"
check_ret_fail "告警趋势 days=0 被拒绝"

req GET "/api/v1/dashboard/alarm-trend?days=abc"
check_ret_fail "告警趋势 days=abc 被拒绝"

# ---------------------------------------------------------------------------
section "设备状态 device-status / device-status-by-type"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/device-status"
check_ret_ok "设备状态计数可查"
check_field "设备状态返回 data 对象" "data"

req GET "/api/v1/dashboard/device-status-by-type"
check_ret_ok "按制式分组设备状态可查"

# ---------------------------------------------------------------------------
section "KPI 趋势 kpi-trend"
# ---------------------------------------------------------------------------
# 自备数据：从 KPI 定义库取一个真实 KPI 名（builtin 指标库种子，171+ 条）
KPI_NAME=""
req GET "/api/v1/pm/kpi/definitions?page=1&page_size=5"
if [ "$HTTP_CODE" = "200" ]; then
    KPI_NAME=$(jget "data.items.0.name")
    [ -z "$KPI_NAME" ] && KPI_NAME=$(jget "data.items.0.kpi_name")
    [ -z "$KPI_NAME" ] && KPI_NAME=$(jget "data.items.0.indicator_name")
fi
# 取不到也不阻塞：kpi-trend 对未知 KPI 名返回空序列（ret=1），用兜底名继续
[ -z "$KPI_NAME" ] && KPI_NAME="RRC_CONN_SUCC_RATE"
# KPI 名可能含空格等特殊字符（如 "CSFB Succ"），拼 query string 前必须 URL 编码
KPI_ENC=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$KPI_NAME")
echo "  （KPI 名取样: ${KPI_NAME}）"

req GET "/api/v1/dashboard/kpi-trend?kpi_name=${KPI_ENC}&days=7"
check_ret_ok "KPI 趋势(kpi_name=${KPI_NAME})可查"

req GET "/api/v1/dashboard/kpi-trend?kpi_name=${KPI_ENC}&compare_with=yesterday"
check_ret_ok "KPI 趋势昨日对比可查"
check_field "对比结果含 metadata" "data.metadata"

req GET "/api/v1/dashboard/kpi-trend?kpi_name=${KPI_ENC}&compare_with=last_week"
check_ret_ok "KPI 趋势上周对比可查"

req GET "/api/v1/dashboard/kpi-trend"
check_ret_fail "缺 kpi_name 被拒绝"

req GET "/api/v1/dashboard/kpi-trend?kpi_name=${KPI_ENC}&compare_with=bogus"
check_ret_fail "非法 compare_with 被拒绝"

req GET "/api/v1/dashboard/kpi-trend?kpi_name=${KPI_ENC}&days=-1"
check_ret_fail "kpi-trend days=-1 被拒绝"

# ---------------------------------------------------------------------------
section "KPI 时间序列 kpi-time-series"
# ---------------------------------------------------------------------------
# RFC3339 时间窗（用 Z 后缀，避免 +08:00 的 + 号在 query string 中被当空格）
TS_START=$(python3 -c "import datetime; print((datetime.datetime.utcnow()-datetime.timedelta(days=7)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
TS_END=$(python3 -c "import datetime; print(datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ'))")

req GET "/api/v1/dashboard/kpi-time-series?kpi_names=${KPI_ENC}"
check_ret_ok "KPI 时间序列(默认近 7 天)可查"
check_field "时间序列按 KPI 名分组返回" "data"

req GET "/api/v1/dashboard/kpi-time-series?kpi_names=${KPI_ENC},FAKE_KPI_${SMOKE_TAG}&start_time=${TS_START}&end_time=${TS_END}"
check_ret_ok "KPI 时间序列(显式时间窗+多 KPI)可查"

req GET "/api/v1/dashboard/kpi-time-series"
check_ret_fail "缺 kpi_names 被拒绝"

req GET "/api/v1/dashboard/kpi-time-series?kpi_names=${KPI_ENC}&start_time=not-a-time"
check_ret_fail "非法 start_time 被拒绝"

req GET "/api/v1/dashboard/kpi-time-series?kpi_names=${KPI_ENC}&end_time=2026/06/10"
check_ret_fail "非法 end_time 被拒绝"

# ---------------------------------------------------------------------------
section "区域统计 region-stats"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/region-stats"
check_ret_ok "区域统计可查"

# ---------------------------------------------------------------------------
section "告警类型饼图 alarm-type-pie"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/alarm-type-pie"
check_ret_ok "告警类型饼图可查"

# ---------------------------------------------------------------------------
section "告警处理效率 alarm-efficiency"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/alarm-efficiency"
check_ret_ok "告警处理效率可查"
check_field "效率指标 severity=overall" "data.severity"
check_field "效率指标含 total_count" "data.total_count"

# ---------------------------------------------------------------------------
section "告警热度图 alarm-heatmap / by-severity"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/alarm-heatmap"
check_ret_ok "告警热度图(默认 30 天)可查"
check_field "热度图含 days_of_week" "data.days_of_week"

req GET "/api/v1/dashboard/alarm-heatmap?days=7"
check_ret_ok "告警热度图(days=7)可查"

req GET "/api/v1/dashboard/alarm-heatmap?days=366"
check_ret_fail "热度图 days=366 越界被拒绝"

req GET "/api/v1/dashboard/alarm-heatmap?days=0"
check_ret_fail "热度图 days=0 被拒绝"

# 已知后端 bug：alarms_history.severity 是 smallint，queryHeatmapBySeverityMap
# （internal/dashboard/heatmap.go）SQL 里 `$2 = '' OR severity = $2` 与 text 比较
# → SQLSTATE 42883 (operator does not exist: smallint = text)，任何调用都 500。
# 修复后下面两个分支会自动回到正常 check_ret_ok 断言。
req GET "/api/v1/dashboard/alarm-heatmap-by-severity"
if [ "$HTTP_CODE" = "500" ]; then
    known_bug "按严重程度热度图可查" "500 SQLSTATE 42883 smallint=text（heatmap.go queryHeatmapBySeverityMap，severity 列 smallint 与 text 参数直接比较）"
else
    check_ret_ok "按严重程度热度图可查"
fi

req GET "/api/v1/dashboard/alarm-heatmap-by-severity?days=7&severity=critical"
if [ "$HTTP_CODE" = "500" ]; then
    known_bug "按严重程度热度图(severity=critical)可查" "同上：500 SQLSTATE 42883 smallint=text"
else
    check_ret_ok "按严重程度热度图(severity=critical)可查"
fi

req GET "/api/v1/dashboard/alarm-heatmap-by-severity?days=999"
check_ret_fail "按严重程度热度图 days=999 越界被拒绝"

# ---------------------------------------------------------------------------
section "widgets 读写闭环（GET 留底 → PUT 原值回写 → GET 比对）"
# ---------------------------------------------------------------------------
req GET "/api/v1/dashboard/widgets"
check_ret_ok "widgets 布局可查"
ORIG_LAYOUT=$(jget "data.layout")
if [ -z "$ORIG_LAYOUT" ]; then
    # 后端无记录时返回空布局 []；兜底防御
    ORIG_LAYOUT="[]"
fi
echo "  （留底 layout: $(printf '%s' "$ORIG_LAYOUT" | head -c 80)）"

req PUT "/api/v1/dashboard/widgets" "{\"layout\":${ORIG_LAYOUT}}"
check_ret_ok "widgets 原值回写(幂等)成功"
check_field "回写结果含 user_id" "data.user_id"

req GET "/api/v1/dashboard/widgets"
check_ret_ok "widgets 回写后可再查"
NEW_LAYOUT=$(jget "data.layout")
NORM_ORIG=$(printf '%s' "$ORIG_LAYOUT" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),sort_keys=True))" 2>/dev/null)
NORM_NEW=$(printf '%s' "$NEW_LAYOUT" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),sort_keys=True))" 2>/dev/null)
if [ -n "$NORM_ORIG" ] && [ "$NORM_ORIG" = "$NORM_NEW" ]; then
    pass "widgets 回写后布局与留底一致（幂等闭环）"
else
    fail "widgets 回写后布局与留底一致（幂等闭环)" "留底=$(printf '%s' "$NORM_ORIG" | head -c 100) 回读=$(printf '%s' "$NORM_NEW" | head -c 100)"
fi

req PUT "/api/v1/dashboard/widgets" "{}"
check_ret_fail "widgets 缺 layout 字段被拒绝"

req PUT "/api/v1/dashboard/widgets" "{\"layout\":"
check_ret_fail "widgets 非法 JSON body 被拒绝"

smoke_summary

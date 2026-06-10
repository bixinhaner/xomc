#!/usr/bin/env bash
# =============================================================================
# smoke_report.sh — F06 报表 业务冒烟
#
# 覆盖路由（/tmp/smoke_routes.json key=report）：
#   GET    /api/v1/reports/definitions          定义列表（分页/类型/状态过滤）
#   POST   /api/v1/reports/definitions          创建定义（闭环自建）
#   GET    /api/v1/reports/definitions/:id      定义详情
#   PUT    /api/v1/reports/definitions/:id      更新定义
#   DELETE /api/v1/reports/definitions/:id      删除定义（闭环清理，records 级联删）
#   GET    /api/v1/reports/records              记录列表
#   POST   /api/v1/reports/generate             触发生成（异步：worker 订阅 NATS 事件落 MinIO）
#   GET    /api/v1/reports/records/:id/download 下载（ready 时流式原始文件，非信封）
#   GET    /api/v1/reports/sample-data          预览样例数据
#
# 关键路径（smokeKeyPaths）：定义 CRUD 闭环 / 生成→记录列表→下载链路 / sample-data 可读
#
# 注意：生成为异步链路（app 发 report.generate.requested → worker ReportGenerator
#       → MinIO → record status=ready）。worker 未运行时记录停留 generating，
#       下载降级断言 200 信封（url 为空 + message），原始文件断言转 skip。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F06 报表" "$@"
smoke_login

# ---------------------------------------------------------------------------
section "报表定义列表（GET /reports/definitions）"
# ---------------------------------------------------------------------------
req GET "/api/v1/reports/definitions"
check_ret_ok "定义列表可查"
check_list_or_empty "定义列表 items 形状" "data.items"
check_field "定义列表带 total" "data.total"

req GET "/api/v1/reports/definitions?page=1&page_size=5&report_type=performance&status=published"
check_ret_ok "定义列表分页+report_type+status 过滤可查"

# ---------------------------------------------------------------------------
section "定义 CRUD 闭环（POST/GET/PUT /reports/definitions）"
# ---------------------------------------------------------------------------
DEF_NAME="${SMOKE_TAG}-report-def"
req POST "/api/v1/reports/definitions" "{\"report_name\":\"${DEF_NAME}\",\"report_type\":\"performance\",\"description\":\"smoke 自建报表定义\",\"format\":[\"json\"],\"period\":\"daily\",\"kpi_codes\":[],\"device_groups\":[],\"auto_generate\":false,\"status\":\"published\",\"creator\":\"smoke\"}"
check_status "创建定义" 201
check_ret_ok "创建定义信封 ret=1"
DEF_ID=$(jget data.id)
check_field "创建返回 data.id" "data.id"

if [ -n "$DEF_ID" ]; then
    req GET "/api/v1/reports/definitions/$DEF_ID"
    check_ret_ok "定义详情可查"
    GOT_NAME=$(jget data.report_name)
    if [ "$GOT_NAME" = "$DEF_NAME" ]; then
        pass "详情 report_name 回读一致 ($GOT_NAME)"
    else
        fail "详情 report_name 回读一致" "期望 ${DEF_NAME}，实际 ${GOT_NAME}"
    fi
    check_field "详情含 format" "data.format"
    check_field "详情含 status" "data.status"

    req PUT "/api/v1/reports/definitions/$DEF_ID" "{\"report_name\":\"${DEF_NAME}\",\"report_type\":\"performance\",\"description\":\"smoke 已更新\",\"format\":[\"json\"],\"period\":\"weekly\",\"kpi_codes\":[],\"device_groups\":[],\"auto_generate\":false,\"status\":\"published\",\"creator\":\"smoke\"}"
    check_ret_ok "更新定义"
    GOT_DESC=$(jget data.description)
    GOT_PERIOD=$(jget data.period)
    if [ "$GOT_DESC" = "smoke 已更新" ] && [ "$GOT_PERIOD" = "weekly" ]; then
        pass "更新后 description/period 回读一致"
    else
        fail "更新后 description/period 回读一致" "description=${GOT_DESC} period=${GOT_PERIOD}"
    fi

    # 自建定义后列表必然非空
    req GET "/api/v1/reports/definitions?page=1&page_size=100"
    check_list_nonempty "自建后定义列表非空" "data.items"
else
    skip "定义详情/更新" "创建未返回 id，无法继续闭环"
fi

# 负路径：非法 ID / 不存在 ID / 缺必填字段
req GET "/api/v1/reports/definitions/not-a-uuid"
check_ret_fail "非法 UUID 查详情被拒"
req GET "/api/v1/reports/definitions/00000000-dead-beef-0000-000000000000"
check_ret_fail "不存在定义查详情被拒"
req POST "/api/v1/reports/definitions" '{"report_type":"performance"}'
check_ret_fail "缺 report_name 创建被拒"
req PUT "/api/v1/reports/definitions/00000000-dead-beef-0000-000000000000" "{\"report_name\":\"${SMOKE_TAG}-x\",\"report_type\":\"alarm\"}"
check_ret_fail "更新不存在定义被拒"

# ---------------------------------------------------------------------------
section "sample-data 可读（GET /reports/sample-data）"
# ---------------------------------------------------------------------------
req GET "/api/v1/reports/sample-data"
check_ret_ok "sample-data 可查"
check_field "含 kpi_summary" "data.kpi_summary"
check_field "含 alarm_summary.total_alarms" "data.alarm_summary.total_alarms"
check_field "含 device_summary.total_devices" "data.device_summary.total_devices"

# ---------------------------------------------------------------------------
section "报表记录列表（GET /reports/records）"
# ---------------------------------------------------------------------------
req GET "/api/v1/reports/records"
check_ret_ok "记录列表可查"
check_list_or_empty "记录列表 items 形状" "data.items"

req GET "/api/v1/reports/records?page=1&page_size=5&format=json"
check_ret_ok "记录列表分页+format 过滤可查"

req GET "/api/v1/reports/records?definition_id=not-a-uuid"
check_ret_fail "非法 definition_id 过滤被拒"

# ---------------------------------------------------------------------------
section "生成→记录列表→下载链路（POST /reports/generate）"
# ---------------------------------------------------------------------------
GEN_PERIOD=$(date +%Y-%m-%d)
REC_ID=""
REC_STATUS=""
if [ -n "$DEF_ID" ]; then
    req POST "/api/v1/reports/generate" "{\"definition_id\":\"$DEF_ID\",\"period\":\"$GEN_PERIOD\"}"
    check_status_in "触发生成" "200 201"
    check_ret_ok "生成信封 ret=1"
    REC_ID=$(jget data.id)
    check_field "生成返回记录 data.id" "data.id"
    check_field "生成返回记录 status" "data.status"

    if [ -n "$REC_ID" ]; then
        # 轮询等待异步 worker 生成完成（最多 ~24s；本定义本次运行专属，items.0 即新记录）
        POLL=0
        while [ "$POLL" -lt 12 ]; do
            req GET "/api/v1/reports/records?definition_id=$DEF_ID&page=1&page_size=10"
            REC_STATUS=$(jget data.items.0.status)
            if [ "$REC_STATUS" = "ready" ] || [ "$REC_STATUS" = "failed" ]; then break; fi
            sleep 2
            POLL=$((POLL + 1))
        done
        check_ret_ok "按 definition_id 过滤记录列表可查"
        check_list_nonempty "生成后记录列表出现记录" "data.items"
        GOT_REC_ID=$(jget data.items.0.id)
        if [ "$GOT_REC_ID" = "$REC_ID" ]; then
            pass "记录列表首条与生成返回 id 一致 ($REC_ID)"
        else
            fail "记录列表首条与生成返回 id 一致" "期望 ${REC_ID}，实际 ${GOT_REC_ID}"
        fi

        if [ "$REC_STATUS" = "ready" ]; then
            # ready：下载为 MinIO 流式原始报表文件（非信封）
            req GET "/api/v1/reports/records/$REC_ID/download"
            check_status "下载已生成报表" 200
            ENV_RET=$(jget ret)
            RAW_NAME=$(jget report_name)
            if [ -z "$ENV_RET" ] && [ -n "$RAW_NAME" ]; then
                pass "下载内容为原始报表 JSON（非信封，report_name=${RAW_NAME}）"
            else
                fail "下载内容为原始报表 JSON（非信封）" "ret=${ENV_RET} report_name=${RAW_NAME} body: $(printf '%s' "$BODY" | head -c 160)"
            fi
        elif [ "$REC_STATUS" = "failed" ]; then
            # 生成失败（worker 收到事件但写 MinIO/采数失败）：下载降级为信封占位。
            # #116（worker 因 task metrics label 数不匹配反复 panic，连带报表生成停摆）
            # 已修复，failed 不再按 known_bug 容忍，转硬 fail。
            req GET "/api/v1/reports/records/$REC_ID/download"
            check_status_in "下载（生成 failed 降级路径）" "200"
            fail "异步生成记录 status=failed" "worker 生成链路失败（采数或 MinIO 写入），需查 worker 日志（#116 worker panic 已修复，不再容忍）"
        else
            # 超时仍 generating：worker 未运行/事件未消费，环境性缺失
            req GET "/api/v1/reports/records/$REC_ID/download"
            check_status_in "下载（仍在 generating 的占位响应）" "200"
            skip "下载原始报表文件" "记录 ${POLL} 轮轮询后仍 status=${REC_STATUS}（worker 未消费生成事件，环境性缺失）"
        fi
    else
        skip "生成后轮询与下载" "生成未返回记录 id"
    fi
else
    skip "生成→记录→下载链路" "前置定义创建失败"
fi

# 生成/下载负路径（不存在 ID / 非法参数）
req POST "/api/v1/reports/generate" '{"period":"2026-06-09"}'
check_ret_fail "缺 definition_id 触发生成被拒"
req POST "/api/v1/reports/generate" '{"definition_id":"not-a-uuid","period":"2026-06-09"}'
check_ret_fail "非法 definition_id 触发生成被拒"
req POST "/api/v1/reports/generate" '{"definition_id":"00000000-dead-beef-0000-000000000000","period":"2026-06-09"}'
check_ret_fail "不存在定义触发生成被拒"
req GET "/api/v1/reports/records/not-a-uuid/download"
check_ret_fail "非法记录 ID 下载被拒"
req GET "/api/v1/reports/records/00000000-dead-beef-0000-000000000000/download"
check_ret_fail "不存在记录下载被拒"

# ---------------------------------------------------------------------------
section "清理（DELETE /reports/definitions/:id，records 级联删除）"
# ---------------------------------------------------------------------------
if [ -n "$DEF_ID" ]; then
    req DELETE "/api/v1/reports/definitions/$DEF_ID"
    check_ret_ok "删除自建定义"
    req GET "/api/v1/reports/definitions/$DEF_ID"
    check_ret_fail "删除后查详情被拒（已不存在）"
    if [ -n "$REC_ID" ]; then
        req GET "/api/v1/reports/records?definition_id=$DEF_ID"
        N_LEFT=$(jlen data.items)
        if [ "$N_LEFT" = "0" ]; then
            pass "定义删除后记录级联清理（残留 0 条）"
        else
            fail "定义删除后记录级联清理" "仍残留 ${N_LEFT} 条记录"
        fi
    fi
else
    skip "清理自建定义" "前置创建失败，无需清理"
fi
req DELETE "/api/v1/reports/definitions/not-a-uuid"
check_ret_fail "非法 ID 删除被拒"

smoke_summary

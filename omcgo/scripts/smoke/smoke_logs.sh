#!/usr/bin/env bash
# =============================================================================
# smoke_logs.sh — F06 日志类 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key ∈ {syslog, stationlog, rebootrecord} 全部路由）：
#   - GET    /api/v1/logs/system                         系统日志分页查询
#   - GET    /api/v1/logs/ne-messages                    网元报文日志查询
#   - GET    /api/v1/station-logs                        基站日志列表（运行/故障）
#   - GET    /api/v1/station-logs/:id/download           基站日志下载（有数据才测）
#   - DELETE /api/v1/station-logs/:id                    destructive — 只测校验负路径
#   - GET    /api/v1/device-abnormal-reboots             异常重启记录列表
#   - GET    /api/v1/device-abnormal-reboots/:id         异常重启记录详情
#   - GET    /api/v1/device-abnormal-reboots/:id/download 异常重启文件下载（file_received 才有文件）
#   - DELETE /api/v1/device-abnormal-reboots/:id         destructive — 只测校验负路径
#   - GET    /api/v1/reboot-records                      统一重启记录列表（两张互斥表合成）
#   - GET    /api/v1/reboot-records/statistics           按设备聚合统计
#
# 后端契约（internal/{syslog,stationlog,rebootrecord}/handler.go）：
#   - logs/system|ne-messages 嵌 model.ListRequest（page min=1 / page_size min=1,max=1000，
#     显式传非法值 → 400）；ne-messages 的 device_id 必须 UUID；
#   - station-logs / device-abnormal-reboots：device_id 必须 UUID；列表 data{items,total,page,size}；
#     abnormal-reboot 详情不存在 → 404；download 仅 record_status=file_received 有文件（detected → 409）；
#   - reboot-records：parseFilter 宽容解析（非法 reboot_type/时间静默回退），无 binding 负路径；
#     statistics 返回 data{items,total} 不分页。
#
# 数据特性：本域全部为设备上传/事件产生的只读数据，无法通过 API 自建 —— 列表一律
# check_list_or_empty 容忍空；下载类有数据才测，无数据 skip。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F06 日志类" "$@"
smoke_login

NOEXIST_ID="00000000-0000-0000-0000-00000000dead"
TIME_FROM="2020-01-01T00:00:00Z"
TIME_TO="2030-01-01T00:00:00Z"

# ---------------------------------------------------------------------------
section "系统日志分页查询（/logs/system）"
# ---------------------------------------------------------------------------
req GET "/api/v1/logs/system?page=1&page_size=10"
check_ret_ok "系统日志列表可查（显式 page/page_size）"
check_list_or_empty "系统日志 data.items" "data.items"
check_field "系统日志含 total 字段" "data.total"

req GET "/api/v1/logs/system?page=1&page_size=10&level=error&source=app"
check_ret_ok "系统日志 level+source 过滤（结果可空）"

req GET "/api/v1/logs/system?page=1&page_size=10&start_time=${TIME_FROM}&end_time=${TIME_TO}"
check_ret_ok "系统日志 RFC3339 时间窗过滤"

# 负路径：binding min=1 / max=1000
req GET "/api/v1/logs/system?page=0&page_size=10"
check_ret_fail "系统日志 page=0 被拒（binding min=1）"
req GET "/api/v1/logs/system?page=1&page_size=2000"
check_ret_fail "系统日志 page_size=2000 被拒（binding max=1000）"

# ---------------------------------------------------------------------------
section "网元报文日志查询（/logs/ne-messages）"
# ---------------------------------------------------------------------------
req GET "/api/v1/logs/ne-messages?page=1&page_size=10"
check_ret_ok "网元报文日志列表可查（显式 page/page_size）"
check_list_or_empty "网元报文日志 data.items" "data.items"
check_field "网元报文日志含 total 字段" "data.total"

req GET "/api/v1/logs/ne-messages?page=1&page_size=10&device_sn=SMK-NO-SUCH-${SMOKE_TAG}&message_type=Inform&direction=rx"
check_ret_ok "网元报文日志 device_sn+message_type+direction 过滤（结果可空）"

req GET "/api/v1/logs/ne-messages?page=1&page_size=10&start_time=${TIME_FROM}&end_time=${TIME_TO}"
check_ret_ok "网元报文日志 RFC3339 时间窗过滤"

# 负路径：device_id 必须 UUID
req GET "/api/v1/logs/ne-messages?page=1&page_size=10&device_id=not-a-uuid"
check_ret_fail "网元报文日志非法 device_id 被拒"

# ---------------------------------------------------------------------------
section "基站日志列表 + 下载（/station-logs）"
# ---------------------------------------------------------------------------
req GET "/api/v1/station-logs?page=1&page_size=10"
check_ret_ok "基站日志列表可查"
check_list_or_empty "基站日志 data.items" "data.items"
check_field "基站日志含 total 字段" "data.total"
STLOG_ID=$(jget data.items.0.id)
STLOG_TYPE=$(jget data.items.0.log_type)

req GET "/api/v1/station-logs?page=1&page_size=10&log_type=running"
check_ret_ok "基站日志 log_type=running 过滤"
req GET "/api/v1/station-logs?page=1&page_size=10&log_type=fault"
check_ret_ok "基站日志 log_type=fault 过滤"

# 负路径：device_id 必须 UUID
req GET "/api/v1/station-logs?device_id=not-a-uuid"
check_ret_fail "基站日志非法 device_id 被拒"

# 下载：有数据才测（数据由真实设备上传 FileType 6/8 产生，无法自建）
if [ -n "$STLOG_ID" ]; then
    req GET "/api/v1/station-logs/${STLOG_ID}/download?log_type=${STLOG_TYPE:-running}"
    check_ret_ok "基站日志下载预签名 URL 可生成"
    check_field "下载返回 url 字段" "data.url"
else
    skip "基站日志下载预签名 URL 可生成" "无基站日志数据（需真实设备上传 FileType 6/8，无法自建）"
fi

# download / delete 负路径（destructive 只测校验，绝不删真实数据）
req GET "/api/v1/station-logs/not-a-uuid/download"
check_ret_fail "基站日志非法 ID 下载被拒"
req DELETE "/api/v1/station-logs/not-a-uuid"
check_ret_fail "基站日志非法 ID 删除被拒"
req DELETE "/api/v1/station-logs/${NOEXIST_ID}"
check_ret_fail "基站日志不存在 ID 删除被拒"

# ---------------------------------------------------------------------------
section "异常重启记录列表 + 详情（/device-abnormal-reboots）"
# ---------------------------------------------------------------------------
req GET "/api/v1/device-abnormal-reboots?page=1&page_size=10"
check_ret_ok "异常重启记录列表可查"
check_list_or_empty "异常重启记录 data.items" "data.items"
check_field "异常重启记录含 total 字段" "data.total"
REBOOT_ID=$(jget data.items.0.id)

req GET "/api/v1/device-abnormal-reboots?page=1&page_size=10&record_status=file_received&device_type=eNB"
check_ret_ok "异常重启记录 record_status+device_type 过滤（结果可空）"

req GET "/api/v1/device-abnormal-reboots?page=1&page_size=10&device_sn=SMK-NO-SUCH-${SMOKE_TAG}&start_time=${TIME_FROM}&end_time=${TIME_TO}"
check_ret_ok "异常重启记录 device_sn+时间窗过滤（结果可空）"

# 负路径：device_id 必须 UUID
req GET "/api/v1/device-abnormal-reboots?device_id=not-a-uuid"
check_ret_fail "异常重启记录非法 device_id 被拒"

# 详情：有数据查真实详情，没数据用确定性负路径打到 /:id 路由
if [ -n "$REBOOT_ID" ]; then
    req GET "/api/v1/device-abnormal-reboots/${REBOOT_ID}"
    check_ret_ok "异常重启记录详情可查"
    check_field "详情含 device_sn" "data.device_sn"
    check_field "详情含 record_status" "data.record_status"
else
    skip "异常重启记录详情可查（真实数据）" "无异常重启记录（需真实设备异常重启产生，无法自建）"
fi
req GET "/api/v1/device-abnormal-reboots/${NOEXIST_ID}"
check_status "异常重启记录不存在 ID 详情 → 404" 404
req GET "/api/v1/device-abnormal-reboots/not-a-uuid"
check_ret_fail "异常重启记录非法 ID 详情被拒"

# 下载：仅 record_status=file_received 的记录有文件，有才测
DL_REBOOT_ID=""
req GET "/api/v1/device-abnormal-reboots?page=1&page_size=1&record_status=file_received"
if [ "$(jlen data.items)" -gt 0 ]; then
    DL_REBOOT_ID=$(jget data.items.0.id)
fi
if [ -n "$DL_REBOOT_ID" ]; then
    req GET "/api/v1/device-abnormal-reboots/${DL_REBOOT_ID}/download"
    check_ret_ok "异常重启日志文件下载预签名 URL 可生成"
    check_field "下载返回 url 字段" "data.url"
else
    skip "异常重启日志文件下载" "无 file_received 状态记录（文件由设备上传产生，无法自建）"
fi
req GET "/api/v1/device-abnormal-reboots/${NOEXIST_ID}/download"
check_status "异常重启不存在 ID 下载 → 404" 404

# destructive DELETE — 只测校验负路径
req DELETE "/api/v1/device-abnormal-reboots/not-a-uuid"
check_ret_fail "异常重启记录非法 ID 删除被拒"
req DELETE "/api/v1/device-abnormal-reboots/${NOEXIST_ID}"
check_ret_fail "异常重启记录不存在 ID 删除被拒"

# ---------------------------------------------------------------------------
section "统一重启记录列表 + 统计（/reboot-records，两张互斥表合成）"
# ---------------------------------------------------------------------------
req GET "/api/v1/reboot-records?page=1&page_size=10"
check_ret_ok "重启记录列表可查"
check_list_or_empty "重启记录 data.items" "data.items"
check_field "重启记录含 total 字段" "data.total"
check_field "重启记录含 page 字段" "data.page"

req GET "/api/v1/reboot-records?page=1&page_size=10&reboot_type=normal"
check_ret_ok "重启记录 reboot_type=normal 过滤（仅 event_logs）"
req GET "/api/v1/reboot-records?page=1&page_size=10&reboot_type=abnormal"
check_ret_ok "重启记录 reboot_type=abnormal 过滤（仅 station_fault_logs）"

req GET "/api/v1/reboot-records?page=1&page_size=10&device_sn=SMK-NO-SUCH-${SMOKE_TAG}&device_type=gNB&start_time=${TIME_FROM}&end_time=${TIME_TO}"
check_ret_ok "重启记录 device_sn+device_type+时间窗过滤（结果可空）"

# parseFilter 宽容解析：非法 reboot_type / 时间静默回退为全部，仍应 200
req GET "/api/v1/reboot-records?page=1&page_size=10&reboot_type=bogus&start_time=not-a-time"
check_ret_ok "重启记录非法过滤值静默回退（宽容解析仍 200）"

req GET "/api/v1/reboot-records/statistics"
check_ret_ok "重启统计（按设备聚合）可查"
check_list_or_empty "重启统计 data.items" "data.items"
check_field "重启统计含 total 字段" "data.total"

req GET "/api/v1/reboot-records/statistics?reboot_type=abnormal&device_type=eNB"
check_ret_ok "重启统计带 reboot_type+device_type 过滤"

smoke_summary

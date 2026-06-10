#!/usr/bin/env bash
# =============================================================================
# smoke_acs.sh — F01 南向 TR069 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=trace + smokeKeyPaths）：
#   1. CPE 模拟器 bootstrap Inform 全会话（cpe_simulator.py --once，会话闭合）
#   2. app 侧设备自动注册上线（keyword 模糊查 → items 内精确比对 SN）
#   3. trace 抓包闭环：建任务 → 查详情 → 按 SN 查活跃任务 → 再发 Inform →
#      messages ≥1 → payload → export → 查 export job → stop → delete
#   4. 全部 read 路由 + 危险/写路径参数校验负路径
#   5. 清理：自建 SN 设备 DELETE（回收站）→ /recycle/permanent 永久删
#
# 用法：bash smoke_acs.sh [BASE_URL]            （默认 http://localhost:8081）
# 环境：OMC_ACS_URL（默认 http://localhost:7557）；ACS 不可达时 CPE/trace 闭环 skip
# 红线：仅被动抓包（trace 任务不向设备下发任何指令）；绝不真发 reboot/升级/SPV
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F01 南向 TR069" "$@"
smoke_login

ACS_URL="${OMC_ACS_URL:-http://localhost:7557}"
ACS_URL="${ACS_URL%/}"
ACS_ENDPOINT="$ACS_URL/smallcell/AcsService"
SIM="$SCRIPT_DIR/../cpe_simulator.py"

SN="SMK-ACS-$(date +%s)"
DEVICE_ID=""
TASK_ID=""
NIL_UUID="00000000-0000-0000-0000-0000000000ff"

# run_sim <event> —— 跑一次模拟器单次会话，设置 SIM_RC / SIM_LOG
run_sim() {
    local event="$1"
    SIM_LOG="$SMOKE_TMPDIR/sim_${event}_$$_$RANDOM.log"
    python3 "$SIM" --acs "$ACS_ENDPOINT" --sn "$SN" --event "$event" --once >"$SIM_LOG" 2>&1
    SIM_RC=$?
}

# find_device_by_sn —— GET /devices?keyword=$SN 后在 items 内精确比对 serial_number
# （keyword 是模糊匹配，必须遍历）。设置 DEV_IDX(-1=未找到)/DEVICE_ID/DEV_ONLINE/DEV_LAST_INFORM
find_device_by_sn() {
    req GET "/api/v1/devices?keyword=$SN&page=1&page_size=50"
    DEV_IDX=-1; DEVICE_ID=""; DEV_ONLINE=""; DEV_LAST_INFORM=""
    local n i s
    n=$(jlen data.items)
    i=0
    while [ "$i" -lt "$n" ]; do
        s=$(jget "data.items.$i.serial_number")
        if [ "$s" = "$SN" ]; then
            DEV_IDX=$i
            DEVICE_ID=$(jget "data.items.$i.id")
            DEV_ONLINE=$(jget "data.items.$i.is_online")
            DEV_LAST_INFORM=$(jget "data.items.$i.last_inform_at")
            break
        fi
        i=$((i + 1))
    done
}

# ---------------------------------------------------------------------------
section "0. ACS 预检（${ACS_URL}）"
# ---------------------------------------------------------------------------
ACS_OK=0
acs_code=$(curl --max-time 5 -s -o /dev/null -w "%{http_code}" "$ACS_URL/healthz" 2>/dev/null || echo "000")
if [ "$acs_code" = "200" ]; then
    pass "ACS /healthz → 200"
    ACS_OK=1
else
    skip "ACS /healthz 不可达（code=${acs_code}）" "CPE 模拟会话/设备注册/trace 抓包闭环整段跳过"
fi

# ---------------------------------------------------------------------------
section "1. CPE 模拟器 bootstrap Inform 全会话"
# ---------------------------------------------------------------------------
if [ "$ACS_OK" = "1" ]; then
    run_sim bootstrap
    if [ "$SIM_RC" = "0" ]; then
        pass "cpe_simulator --once 退出码 0（SN=${SN}）"
    else
        fail "cpe_simulator --once 退出码 0（SN=${SN}）" "rc=${SIM_RC}，日志尾部: $(tail -3 "$SIM_LOG" | tr '\n' ' ' | head -c 200)"
    fi
    if grep -q "204 No Content" "$SIM_LOG"; then
        pass "会话完整闭合（Inform → … → 204 No Content）"
    else
        fail "会话完整闭合（Inform → … → 204 No Content）" "日志无 204 特征: $(tail -3 "$SIM_LOG" | tr '\n' ' ' | head -c 200)"
    fi
    if grep -E "错误: +0" "$SIM_LOG" >/dev/null; then
        pass "会话统计 0 错误"
    else
        fail "会话统计 0 错误" "$(grep "错误" "$SIM_LOG" | head -1)"
    fi
else
    skip "cpe_simulator bootstrap 会话" "ACS 不可达"
fi

# ---------------------------------------------------------------------------
section "2. 设备自动注册上线（app 侧验证）"
# ---------------------------------------------------------------------------
if [ "$ACS_OK" = "1" ]; then
    find_device_by_sn
    check_ret_ok "GET /devices?keyword=$SN 可查"
    if [ "$DEV_IDX" -ge 0 ]; then
        pass "items 内精确匹配到 serial_number=${SN}（id=$(printf '%s' "$DEVICE_ID" | head -c 36)）"
        if [ "$DEV_ONLINE" = "True" ]; then
            pass "设备 is_online=true"
        else
            fail "设备 is_online=true" "实际 is_online=$DEV_ONLINE"
        fi
        if [ -n "$DEV_LAST_INFORM" ] && [ "$DEV_LAST_INFORM" != "None" ]; then
            pass "last_inform_at 非空（${DEV_LAST_INFORM}）"
        else
            fail "last_inform_at 非空" "实际为空"
        fi
    else
        fail "items 内精确匹配到 serial_number=$SN" "未找到（items 共 $(jlen data.items) 条）"
        fail "设备 is_online=true" "设备未注册"
        fail "last_inform_at 非空" "设备未注册"
    fi
else
    skip "设备自动注册验证" "ACS 不可达，未发 Inform"
fi

# ---------------------------------------------------------------------------
section "3. trace 任务列表读基线"
# ---------------------------------------------------------------------------
req GET "/api/v1/trace/tasks"
check_list_or_empty "GET /trace/tasks 任务列表可查" "data.items"

req GET "/api/v1/trace/tasks?device_sn=$SN&status=running&page=1&page_size=10"
check_ret_ok "GET /trace/tasks 带 device_sn/status 过滤可查"

# ---------------------------------------------------------------------------
section "4. trace 抓包闭环（建 → 查 → 活跃任务 → 捕获 → payload → export → 停 → 删）"
# ---------------------------------------------------------------------------
if [ "$ACS_OK" = "1" ]; then
    # 4.1 建任务（被动捕获，不向设备下发任何指令）
    req POST "/api/v1/trace/tasks" "{\"device_sn\":\"$SN\",\"duration_minutes\":10}"
    check_status "POST /trace/tasks 创建抓包任务" 201
    TASK_ID=$(jget data.id)
    check_field "创建返回任务 id" "data.id"

    if [ -n "$TASK_ID" ]; then
        # 4.2 查详情
        req GET "/api/v1/trace/tasks/$TASK_ID"
        check_ret_ok "GET /trace/tasks/:id 任务详情可查"
        v=$(jget data.device_sn)
        if [ "$v" = "$SN" ]; then pass "详情 device_sn 回显一致"; else fail "详情 device_sn 回显一致" "期望 ${SN}，实际 $v"; fi
        v=$(jget data.status)
        if [ "$v" = "running" ]; then pass "新建任务 status=running"; else fail "新建任务 status=running" "实际 $v"; fi

        # 4.3 按 SN 查活跃任务
        req GET "/api/v1/trace/devices/$SN/active-task"
        check_ret_ok "GET /trace/devices/:sn/active-task 可查"
        v=$(jget data.id)
        if [ "$v" = "$TASK_ID" ]; then pass "active-task 命中刚建的任务"; else fail "active-task 命中刚建的任务" "期望 ${TASK_ID}，实际 $v"; fi

        # 4.4 再发一次 periodic Inform 触发捕获（白名单 NATS 实时推送 + 30s 周期刷新兜底）
        sleep 2
        run_sim periodic
        if [ "$SIM_RC" = "0" ]; then
            pass "trace 窗口内 periodic Inform 会话完成"
        else
            fail "trace 窗口内 periodic Inform 会话完成" "rc=$SIM_RC: $(tail -3 "$SIM_LOG" | tr '\n' ' ' | head -c 200)"
        fi

        # 4.5 轮询 messages（捕获经 NATS → worker 批量落库，flush 500ms；
        #     白名单若走 30s 兜底刷新，第 4 轮补发一次 Inform）
        MSG_TOTAL=0
        attempt=1
        while [ "$attempt" -le 12 ]; do
            req GET "/api/v1/trace/tasks/$TASK_ID/messages?page=1&page_size=20"
            MSG_TOTAL=$(jget data.total)
            [ -z "$MSG_TOTAL" ] && MSG_TOTAL=0
            if [ "$MSG_TOTAL" -ge 1 ] 2>/dev/null; then break; fi
            if [ "$attempt" = "4" ]; then run_sim periodic; fi
            sleep 3
            attempt=$((attempt + 1))
        done
        if [ "$MSG_TOTAL" -ge 1 ] 2>/dev/null; then
            pass "GET /trace/tasks/:id/messages 捕获报文 ≥1 条（total=${MSG_TOTAL}，第 $attempt 轮）"
        else
            fail "GET /trace/tasks/:id/messages 捕获报文 ≥1 条" "轮询 12 轮后 total=$MSG_TOTAL"
        fi

        # 4.6 payload 读链路（XML 直出，不走信封）。
        # 注意：捕获含 Empty POST / 204 边界报文（payload_size_bytes=0 属正常），
        # 必须挑一条 payload_size_bytes>0 的报文（如 Inform）再取报文体。
        MSG_ID=""
        MSG_SIZE=0
        n=$(jlen data.items)
        i=0
        while [ "$i" -lt "$n" ]; do
            sz=$(jget "data.items.$i.payload_size_bytes")
            if [ -n "$sz" ] && [ "$sz" -gt 0 ] 2>/dev/null; then
                MSG_ID=$(jget "data.items.$i.id")
                MSG_SIZE=$sz
                break
            fi
            i=$((i + 1))
        done
        if [ -n "$MSG_ID" ]; then
            req GET "/api/v1/trace/tasks/$TASK_ID/messages/$MSG_ID/payload"
            check_status "GET messages/:msgId/payload 报文体可读" 200
            if [ -n "$BODY" ]; then
                pass "payload 非空（${#BODY} 字节，库记录 ${MSG_SIZE} 字节）"
            else
                fail "payload 非空" "响应体为空（库记录 ${MSG_SIZE} 字节）"
            fi
        else
            skip "payload 读链路" "无 payload_size_bytes>0 的捕获报文可取"
            skip "payload 非空" "无 payload_size_bytes>0 的捕获报文可取"
        fi

        # 4.7 export 异步导出（202 + job）
        req POST "/api/v1/trace/tasks/$TASK_ID/export"
        check_status "POST /trace/tasks/:id/export 受理" 202
        JOB_ID=$(jget data.id)
        check_field "export 返回 job id" "data.id"
        if [ -n "$JOB_ID" ]; then
            sleep 2
            req GET "/api/v1/trace/exports/$JOB_ID"
            check_ret_ok "GET /trace/exports/:jobId 任务状态可查"
            v=$(jget data.status)
            case "$v" in
                queued|running|done) pass "export job 状态正常（${v}）" ;;
                *) fail "export job 状态正常" "实际 status=$v error=$(jget data.error_message)" ;;
            esac
        else
            skip "export job 状态查询" "未拿到 job id"
        fi

        # 4.8 stop（必须先停才能删）
        req POST "/api/v1/trace/tasks/$TASK_ID/stop" "{}"
        check_ret_ok "POST /trace/tasks/:id/stop 停止任务"
        v=$(jget data.status)
        if [ "$v" = "stopped" ]; then pass "停止后 status=stopped"; else fail "停止后 status=stopped" "实际 $v"; fi

        # 停止后 active-task 应返回 200 + data=null
        req GET "/api/v1/trace/devices/$SN/active-task"
        v=$(jget data)
        if [ "$HTTP_CODE" = "200" ] && [ -z "$v" ]; then
            pass "停止后 active-task 返回 200 + data=null"
        else
            fail "停止后 active-task 返回 200 + data=null" "HTTP $HTTP_CODE data=$(printf '%s' "$v" | head -c 80)"
        fi

        # 4.9 删除任务（连带 messages/export_jobs 级联清理）
        req DELETE "/api/v1/trace/tasks/$TASK_ID"
        check_ret_ok "DELETE /trace/tasks/:id 删除任务"
        v=$(jget data.deleted)
        if [ "$v" = "True" ]; then pass "删除回执 deleted=true"; else fail "删除回执 deleted=true" "实际 $v"; fi

        req GET "/api/v1/trace/tasks/$TASK_ID"
        check_status "删除后任务详情 → 404" 404
    else
        skip "trace 闭环后续步骤" "建任务未返回 id"
    fi
else
    skip "trace 抓包闭环" "ACS 不可达，无法触发捕获"
fi

# ---------------------------------------------------------------------------
section "5. 参数校验负路径（trace 写/危险入口）"
# ---------------------------------------------------------------------------
req POST "/api/v1/trace/tasks" "{}"
check_ret_fail "建任务缺 device_sn → 拒绝"

req POST "/api/v1/trace/tasks" "{\"device_sn\":\"SMK-NEG-${SMOKE_TAG}\",\"duration_minutes\":999}"
check_ret_fail "建任务 duration_minutes>60 → 拒绝"

req GET "/api/v1/trace/tasks/not-a-uuid"
check_status "任务详情非法 UUID → 400" 400

req GET "/api/v1/trace/tasks/$NIL_UUID"
check_status "任务详情不存在 ID → 404" 404

req GET "/api/v1/trace/tasks/$NIL_UUID/messages"
check_status_in "不存在任务查 messages → 拒绝/空" "200 404"

req POST "/api/v1/trace/tasks/$NIL_UUID/stop" "{}"
check_status "停止不存在任务 → 404" 404

req DELETE "/api/v1/trace/tasks/$NIL_UUID"
check_status "删除不存在任务 → 404" 404

req POST "/api/v1/trace/tasks/$NIL_UUID/export"
check_status "导出不存在任务 → 404" 404

req GET "/api/v1/trace/exports/not-a-uuid"
check_status "export job 非法 UUID → 400" 400

req POST "/api/v1/trace/tasks/batch-delete" "{\"ids\":[]}"
check_ret_fail "batch-delete 空 ids → 拒绝"

req POST "/api/v1/trace/tasks/batch-delete" "{\"ids\":[\"$NIL_UUID\"]}"
check_ret_ok "batch-delete 不存在 ID → 200 返回逐条结果"
v=$(jget data.failed)
if [ "$v" = "1" ]; then pass "batch-delete 回执 failed=1"; else fail "batch-delete 回执 failed=1" "实际 failed=$v body=$(printf '%s' "$BODY" | head -c 120)"; fi

# ---------------------------------------------------------------------------
section "6. 清理（自建 SN 设备 → 回收站 → 永久删）"
# ---------------------------------------------------------------------------
if [ "$ACS_OK" = "1" ] && [ -n "$DEVICE_ID" ]; then
    req DELETE "/api/v1/devices/$DEVICE_ID"
    check_ret_ok "DELETE /devices/:id 软删进回收站"

    req DELETE "/api/v1/devices/recycle/permanent" "{\"ids\":[\"$DEVICE_ID\"]}"
    check_ret_ok "DELETE /devices/recycle/permanent 永久删除"

    find_device_by_sn
    if [ "$DEV_IDX" = "-1" ]; then
        pass "设备列表已无 ${SN}（清理完成）"
    else
        fail "设备列表已无 $SN" "仍能查到（idx=${DEV_IDX}）"
    fi
else
    skip "自建设备清理" "未注册设备（ACS 不可达或注册失败）"
fi

smoke_summary

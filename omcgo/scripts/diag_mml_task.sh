#!/usr/bin/env bash
#
# diag_mml_task.sh — 一键扫描 MML RPC 任务执行链路上的 8 个 checkpoint，
# 报告卡在哪一步。配套文档：docs/operations/troubleshoot-mml-rpc.md。
#
# 用法：
#   bash omcgo/scripts/diag_mml_task.sh --sn <SN> --mml-task <UUID> [选项]
#
# 必填：
#   --sn <SN>             设备序列号
#   --mml-task <UUID>     mml_tasks.id
#
# 可选（覆盖默认连接配置 / 日志路径）：
#   --pg-host HOST        默认 localhost
#   --pg-port PORT        默认 5432
#   --pg-user USER        默认 omcgo
#   --pg-pass PASS        默认 omcgo123
#   --pg-db   DB          默认 omcgo
#   --redis-host HOST     默认 localhost
#   --redis-port PORT     默认 6379
#   --acs-log PATH        默认 <repo>/run/logs/acs/acs.log
#   --app-log PATH        默认 <repo>/run/logs/app/app.log
#   --no-color            禁用彩色输出
#   --json                以 JSON 输出而非 Markdown（便于程序消费）
#
# 退出码：
#   0 — 链路完整推进到 [8]（任务已收尾）
#   1 — 中途卡住（最后一个 ✓ checkpoint 之后停滞）
#   2 — 缺少必填参数 / 工具不可用
#
# 示例：
#   bash omcgo/scripts/diag_mml_task.sh \
#     --sn 1202000240194DP0026 \
#     --mml-task 6e58a13a-4b6d-4c5b-9e9d-83e7a4b1c2f0
#

set -uo pipefail

# ---- 默认参数 ----
SN=""
MML=""
PG_HOST=${PGHOST:-localhost}
PG_PORT=${PGPORT:-5432}
PG_USER=${PGUSER:-omcgo}
PG_PASS=${PGPASSWORD:-omcgo123}
PG_DB=${PGDATABASE:-omcgo}
REDIS_HOST=localhost
REDIS_PORT=6379

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ACS_LOG="$REPO_ROOT/../run/logs/acs/acs.log"
APP_LOG="$REPO_ROOT/../run/logs/app/app.log"

USE_COLOR=1
OUTPUT=md   # md | json

while [[ $# -gt 0 ]]; do
    case $1 in
        --sn) SN=$2; shift 2 ;;
        --mml-task) MML=$2; shift 2 ;;
        --pg-host) PG_HOST=$2; shift 2 ;;
        --pg-port) PG_PORT=$2; shift 2 ;;
        --pg-user) PG_USER=$2; shift 2 ;;
        --pg-pass) PG_PASS=$2; shift 2 ;;
        --pg-db) PG_DB=$2; shift 2 ;;
        --redis-host) REDIS_HOST=$2; shift 2 ;;
        --redis-port) REDIS_PORT=$2; shift 2 ;;
        --acs-log) ACS_LOG=$2; shift 2 ;;
        --app-log) APP_LOG=$2; shift 2 ;;
        --no-color) USE_COLOR=0; shift ;;
        --json) OUTPUT=json; shift ;;
        -h|--help)
            sed -n '2,40p' "$0"
            exit 0
            ;;
        *) echo "Unknown arg: $1" >&2; exit 2 ;;
    esac
done

# ---- 校验 ----
if [[ -z "$SN" || -z "$MML" ]]; then
    echo "Usage: $0 --sn <SN> --mml-task <UUID>" >&2
    echo "  (See $0 --help for full options.)" >&2
    exit 2
fi

for cmd in psql redis-cli jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Required command not found: $cmd" >&2
        echo "  brew install postgresql redis jq  # macOS" >&2
        exit 2
    fi
done

if [[ "$USE_COLOR" == 1 && -t 1 ]]; then
    GRN=$'\e[32m'; RED=$'\e[31m'; YLW=$'\e[33m'; CYN=$'\e[36m'; DIM=$'\e[2m'; RST=$'\e[0m'
else
    GRN=""; RED=""; YLW=""; CYN=""; DIM=""; RST=""
fi

PSQL="env PGPASSWORD=$PG_PASS psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -t -A -F$'\t'"
REDIS="redis-cli -h $REDIS_HOST -p $REDIS_PORT"

# ---- 状态变量 ----
declare -a CHECK_NAMES CHECK_STATUS CHECK_DETAILS
LAST_OK=0   # 最后一个 ✓ 的 checkpoint 序号
NEXT_HINT=""

record() {
    # record <idx> <name> <status:ok|fail|warn|na> <detail>
    CHECK_NAMES[$1]=$2
    CHECK_STATUS[$1]=$3
    CHECK_DETAILS[$1]=$4
    if [[ $3 == ok ]]; then
        LAST_OK=$1
    fi
}

# ---- Checkpoint 1: mml_tasks ----
mml_row=$($PSQL -c "SELECT status, execute_type, COALESCE(array_length(device_sns,1),0), COALESCE(jsonb_array_length(commands),0), success_count, failed_count, COALESCE(started_at::text, ''), COALESCE(finished_at::text, '') FROM mml_tasks WHERE id='$MML';" 2>/dev/null)
if [[ -z "$mml_row" ]]; then
    record 1 "mml_tasks 写入" fail "未找到 id=$MML 的行"
    NEXT_HINT="execute API 没把 mml_task 落库；查 APP 日志关键字 'mml task created' / handler 异常栈"
else
    IFS=$'\t' read -r mml_status mml_etype mml_dev_n mml_cmd_n mml_succ mml_fail mml_started mml_finished <<< "$mml_row"
    detail="status=$mml_status execute_type=$mml_etype devices=$mml_dev_n commands=$mml_cmd_n success=$mml_succ failed=$mml_fail"
    record 1 "mml_tasks 写入" ok "$detail"
fi

# ---- Checkpoint 2: device_tasks ----
if [[ ${CHECK_STATUS[1]:-} == ok ]]; then
    dt_count=$($PSQL -c "SELECT COUNT(*) FROM device_tasks WHERE source='mml' AND source_id='$MML';" 2>/dev/null | tr -d ' ')
    dt_count=${dt_count:-0}

    if [[ "$dt_count" == "0" ]]; then
        record 2 "device_tasks 派生" fail "0 行（Fanouter 未派生）"
        NEXT_HINT="grep 'build tr069 params failed|skip command without rpc_method' $APP_LOG"
    else
        # 取这台设备这条任务的 device_task
        dt_row=$($PSQL -c "SELECT id, status, COALESCE(cwmp_id,''), method, COALESCE(sent_at::text,''), (result IS NOT NULL), params FROM device_tasks WHERE source='mml' AND source_id='$MML' AND device_sn='$SN' ORDER BY command_index, device_index LIMIT 1;" 2>/dev/null)
        if [[ -z "$dt_row" ]]; then
            record 2 "device_tasks 派生" warn "找到 $dt_count 行，但设备 $SN 没行（请检查 SN）"
            NEXT_HINT="确认 SN 拼写或在 mml_tasks.device_sns 是否含 $SN"
        else
            IFS=$'\t' read -r DT_ID dt_status dt_cwmpid dt_method dt_sent_at dt_has_result dt_params <<< "$dt_row"
            shape=$(echo "$dt_params" | jq -c 'if has("names") then "names("+(.names|length|tostring)+")" elif has("values") then "values("+(.values|length|tostring)+")" elif has("object_name") then "object_name" else (keys|join(",")) end' 2>/dev/null)
            record 2 "device_tasks 派生" ok "total=$dt_count this_sn=$DT_ID method=$dt_method status=$dt_status shape=$shape"
        fi
    fi
else
    record 2 "device_tasks 派生" na "上游未通过"
fi

# ---- Checkpoint 3: Redis 队列 ----
if [[ ${CHECK_STATUS[2]:-} == ok ]]; then
    queue_members=$($REDIS ZRANGE "acs:taskq:$SN" 0 -1 2>/dev/null)
    if [[ -z "$queue_members" ]]; then
        # 队列空可能是已派发，看 task hash
        task_data=$($REDIS HGET "acs:task:$DT_ID" data 2>/dev/null)
        if [[ -n "$task_data" ]]; then
            t_status=$(echo "$task_data" | jq -r .status 2>/dev/null)
            record 3 "Redis 队列" ok "队列已派发；acs:task:$DT_ID 存在，status=$t_status"
        else
            if [[ "$dt_status" == "pending" ]]; then
                record 3 "Redis 队列" fail "PG=pending 但 Redis 无队列项也无 task hash（不一致）"
                NEXT_HINT="grep 'RestorePendingQueues|task recovered' $APP_LOG"
            else
                record 3 "Redis 队列" warn "Redis 无队列项；PG status=$dt_status，可能任务已收尾或 task TTL 过期"
            fi
        fi
    else
        if echo "$queue_members" | grep -q "$DT_ID"; then
            record 3 "Redis 队列" ok "task_id 在 acs:taskq:$SN 中等待派发"
        else
            record 3 "Redis 队列" warn "队列有 task 但不含 $DT_ID（已派发或被其他任务挤掉）"
        fi
    fi
else
    record 3 "Redis 队列" na "上游未通过"
fi

# ---- Checkpoint 4: CPE Inform ----
heartbeat=$($REDIS GET "acs:heartbeat:$SN" 2>/dev/null)
hb_ttl=$($REDIS TTL "acs:heartbeat:$SN" 2>/dev/null)
if [[ -n "$heartbeat" && "$hb_ttl" != "-2" ]]; then
    record 4 "CPE Inform" ok "heartbeat=$heartbeat ttl=${hb_ttl}s"
else
    record 4 "CPE Inform" fail "heartbeat 不存在（CPE 离线 / 网络隔离 / 未上来）"
    NEXT_HINT="抓包 'tcp port 7547' 或 grep 'ACS Inform processing' $ACS_LOG | grep $SN"
fi

# ---- Checkpoint 5: ACS 下发 SOAP ----
if [[ -f "$ACS_LOG" && ${CHECK_STATUS[2]:-} == ok ]]; then
    send_log=$(grep "ACS sending RPC request from task" "$ACS_LOG" 2>/dev/null \
        | jq -c "select(.task_id==\"$DT_ID\")" 2>/dev/null | tail -1)
    if [[ -z "$send_log" ]]; then
        record 5 "ACS 下发 SOAP" fail "ACS log 未见下发记录"
        NEXT_HINT="ACS 没跑到 PopTask 这一步；看 [4] CPE 是否真的 Inform，或 ACS 进程是否健康"
    else
        cwmp=$(echo "$send_log" | jq -r .cwmp_id)
        size=$(echo "$send_log" | jq -r .soap_size)
        body=$(echo "$send_log" | jq -r .soap_body)

        # 形态判定：是否含空 ParameterNames（Q4 根因特征）
        if echo "$body" | grep -qE 'ParameterNames[^>]*arrayType="xsd:string\[0\]"'; then
            record 5 "ACS 下发 SOAP" warn "下发了，但 ParameterNames 为空（Q4 根因！）cwmp_id=$cwmp size=$size"
            NEXT_HINT="部署 BuildTR069Params 修复版；或检查 mml_command 的 param_refs 是否绑定"
        elif echo "$body" | grep -qE 'cwmp:Fault'; then
            record 5 "ACS 下发 SOAP" warn "下发的报文是 Fault（异常路径）cwmp_id=$cwmp"
        else
            record 5 "ACS 下发 SOAP" ok "cwmp_id=$cwmp size=$size body looks valid"
        fi
        CWMP_ID=$cwmp
    fi
else
    if [[ ! -f "$ACS_LOG" ]]; then
        record 5 "ACS 下发 SOAP" na "ACS 日志路径不存在: $ACS_LOG（用 --acs-log 指定）"
    else
        record 5 "ACS 下发 SOAP" na "上游未通过"
    fi
fi

# ---- Checkpoint 6: CPE 响应 ----
if [[ -f "$ACS_LOG" && ${CHECK_STATUS[5]:-} == ok && -n "${CWMP_ID:-}" ]]; then
    resp_log=$(grep -E "RPC response received|ACS received SOAP Fault" "$ACS_LOG" 2>/dev/null \
        | jq -c "select(.cwmp_id==\"$CWMP_ID\")" 2>/dev/null | tail -1)
    fault_log=$(grep "task failed with SOAP fault" "$ACS_LOG" 2>/dev/null \
        | jq -c "select(.task_id==\"$DT_ID\")" 2>/dev/null | tail -1)
    if [[ -n "$fault_log" ]]; then
        fc=$(echo "$fault_log" | jq -r .fault_code)
        fm=$(echo "$fault_log" | jq -r .fault_msg)
        record 6 "CPE 响应" warn "CPE 返回 Fault code=$fc msg=$fm"
        NEXT_HINT="多半是协议参数问题（Q4 根因 / param 路径不存在 / 类型错）。看 fault_msg + soap_body 对照协议"
    elif [[ -n "$resp_log" ]]; then
        record 6 "CPE 响应" ok "ACS 收到响应（cwmp_id=$CWMP_ID）"
    else
        record 6 "CPE 响应" fail "ACS log 未见响应（CPE 没回 / 报文非法 / Cookie 失效）"
        NEXT_HINT="抓包确认 CPE 是否真的 POST 了响应；grep 'no valid session cookie' $ACS_LOG"
    fi
else
    record 6 "CPE 响应" na "上游未通过"
fi

# ---- Checkpoint 7: ACS 标记完成 ----
if [[ ${CHECK_STATUS[2]:-} == ok ]]; then
    case "$dt_status" in
        completed)
            record 7 "ACS 标记 completed" ok "device_task.status=completed has_result=$dt_has_result"
            ;;
        failed)
            err_msg=$($PSQL -c "SELECT COALESCE(error_message,'') FROM device_tasks WHERE id='$DT_ID';" 2>/dev/null)
            record 7 "ACS 标记 failed" warn "device_task.status=failed error=$err_msg"
            NEXT_HINT="任务级失败已标记，[8] 仍会聚合（finalizeIfComplete 把 failed 计入）"
            ;;
        sent)
            record 7 "ACS 标记完成" fail "device_task 持续 status=sent，sent_at=$dt_sent_at"
            NEXT_HINT="ACS 没收到响应（看 [6]）或 cwmp_id 没匹配；grep 'get task by cwmp_id' $ACS_LOG"
            ;;
        pending)
            record 7 "ACS 标记完成" na "尚未派发"
            ;;
        *)
            record 7 "ACS 标记完成" warn "未知 status=$dt_status"
            ;;
    esac
else
    record 7 "ACS 标记完成" na "上游未通过"
fi

# ---- Checkpoint 8: MML 聚合器 ----
if [[ -f "$APP_LOG" ]]; then
    finalized=$(grep "mml task finalized" "$APP_LOG" 2>/dev/null \
        | jq -c "select(.mml_task_id==\"$MML\")" 2>/dev/null | tail -1)
    no_handler=$(grep "no completion handler registered for task source" "$APP_LOG" 2>/dev/null \
        | jq -c 'select(.source=="mml")' 2>/dev/null | tail -1)

    if [[ -n "$finalized" ]]; then
        fs=$(echo "$finalized" | jq -r .status)
        fr=$(echo "$finalized" | jq -r .result)
        record 8 "MML 聚合器收尾" ok "mml task finalized status=$fs result=$fr"
    elif [[ -n "$no_handler" ]]; then
        record 8 "MML 聚合器收尾" fail "事件到了但 router 没有 mml handler（modules.go Register 漏调？）"
        NEXT_HINT="检查 cmd/app/provider/modules.go:272 完成路由注册"
    else
        bridge_subscribed=$(grep "task completion event bridge subscribed" "$APP_LOG" 2>/dev/null | tail -1)
        if [[ -z "$bridge_subscribed" ]]; then
            record 8 "MML 聚合器收尾" fail "APP 启动时桥接未订阅 NATS"
            NEXT_HINT="grep 'subscribe task completion bridge' $APP_LOG （看 error）；确认 NATS 服务可达"
        elif [[ "$mml_status" != "completed" && "$mml_status" != "failed" ]]; then
            record 8 "MML 聚合器收尾" fail "桥接已订阅但 mml task 未 finalize"
            NEXT_HINT="nats sub 'task.>' 实时观察是否有 task.completed 事件被发布"
        else
            record 8 "MML 聚合器收尾" ok "mml_tasks.status=$mml_status（mml task finalized 日志可能已轮转）"
        fi
    fi
else
    record 8 "MML 聚合器收尾" na "APP 日志路径不存在: $APP_LOG"
fi

# ---- 输出 ----
if [[ "$OUTPUT" == json ]]; then
    printf '{"sn":"%s","mml_task":"%s","checkpoints":[' "$SN" "$MML"
    sep=""
    for i in 1 2 3 4 5 6 7 8; do
        printf '%s{"step":%d,"name":"%s","status":"%s","detail":%s}' \
            "$sep" "$i" "${CHECK_NAMES[$i]}" "${CHECK_STATUS[$i]}" \
            "$(printf '%s' "${CHECK_DETAILS[$i]}" | jq -R .)"
        sep=","
    done
    printf '],"next_hint":%s}\n' "$(printf '%s' "$NEXT_HINT" | jq -R .)"
else
    sym() {
        case $1 in
            ok)   echo "${GRN}✓${RST}" ;;
            fail) echo "${RED}✗${RST}" ;;
            warn) echo "${YLW}!${RST}" ;;
            na)   echo "${DIM}─${RST}" ;;
        esac
    }
    echo
    echo "${CYN}===== MML RPC 任务诊断报告 =====${RST}"
    echo "  SN:        $SN"
    echo "  MML Task:  $MML"
    echo "  Time:      $(date '+%Y-%m-%d %H:%M:%S')"
    echo
    echo "  Step | Status | Checkpoint               | Detail"
    echo "  -----+--------+--------------------------+----------------------------------"
    for i in 1 2 3 4 5 6 7 8; do
        printf "  [%d]  |   %s    | %-24s | %s\n" \
            "$i" "$(sym "${CHECK_STATUS[$i]}")" "${CHECK_NAMES[$i]}" "${CHECK_DETAILS[$i]}"
    done
    echo
    if [[ -n "$NEXT_HINT" ]]; then
        echo "${YLW}下一步建议：${RST}$NEXT_HINT"
    fi
    echo "${DIM}详情参见 docs/operations/troubleshoot-mml-rpc.md 对应章节。${RST}"
    echo
fi

# ---- 退出码 ----
if [[ ${CHECK_STATUS[8]:-} == ok ]]; then
    exit 0
fi
exit 1

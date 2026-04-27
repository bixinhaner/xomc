#!/usr/bin/env bash
#
# diag_mml_task.sh — 一键扫描 MML RPC 任务执行链路上的 8 个 checkpoint，
# 报告卡在哪一步。配套文档：docs/operations/troubleshoot-mml-rpc.md。
#
# 默认 mode=docker：所有数据来源（PG/Redis/日志）走 docker compose
#   - psql / redis-cli  → docker compose exec -T <svc> ...
#   - acs/app 日志       → docker compose logs --since <dur> <svc> 缓存到 /tmp
#   宿主只需要 docker + jq，不需要装 postgresql-client / redis 包。
#
# 用法：
#   bash omcgo/scripts/diag_mml_task.sh --auto                   # 最简
#   bash omcgo/scripts/diag_mml_task.sh --sn <SN> --mml-task <UUID>
#
# 必填（除非 --auto 自动回填）：
#   --sn <SN>             设备序列号
#   --mml-task <UUID>     mml_tasks.id
#
# 自动检测：
#   --auto                自动从数据库取最新一条 mml_task 及其首个 device_sn。
#                         可与 --sn 或 --mml-task 配合（只补缺失的那一个）：
#                           --auto                       最新 mml_task + 首个 SN
#                           --auto --sn <SN>             含此 SN 的最新 mml_task
#                           --auto --mml-task <UUID>     此 task 的首个 SN
#   --offset N            配合 --auto，跳过前 N 条选第 N+1 条（默认 0）
#
# Docker 模式（默认）：
#   --compose-file PATH   docker-compose.yml 路径（默认 <repo>/deployments/docker/docker-compose.yml）
#   --postgres-svc NAME   PG 服务名（默认 postgres）
#   --redis-svc NAME      Redis 服务名（默认 redis）
#   --acs-svc NAME        ACS 服务名（默认 acs）
#   --app-svc NAME        APP 服务名（默认 app）
#   --logs-since DUR      docker compose logs --since 参数（默认 30m）
#
# Host 模式（直连，宿主已装 psql/redis-cli 时用）：
#   --mode host           切到直连模式（默认 docker）
#   --pg-host HOST        默认 localhost
#   --pg-port PORT        默认 5432
#   --redis-host HOST     默认 localhost
#   --redis-port PORT     默认 6379
#   --acs-log PATH        默认 <repo>/run/logs/acs/acs.log
#   --app-log PATH        默认 <repo>/run/logs/app/app.log
#
# 通用：
#   --pg-user USER        默认 omcgo
#   --pg-pass PASS        默认 omcgo123
#   --pg-db DB            默认 omcgo
#   --no-color            禁用彩色输出
#   --json                以 JSON 输出（程序消费）
#   -h, --help            显示本说明
#
# 退出码：
#   0 — 链路完整推进到 [8]（任务已收尾）
#   1 — 中途卡住
#   2 — 参数错 / 工具不可用 / 自动检测未找到 mml_task
#

set -uo pipefail

# ---- 默认参数 ----
SN=""
MML=""
AUTO=0
OFFSET=0

MODE=docker

# 仓库根（用 git；失败回退到脚本所在路径推算）
if REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null); then :; else
    REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fi

COMPOSE_FILE="$REPO_ROOT/deployments/docker/docker-compose.yml"
POSTGRES_SVC=postgres
REDIS_SVC=redis
ACS_SVC=acs
APP_SVC=app
LOGS_SINCE=30m

PG_HOST=${PGHOST:-localhost}
PG_PORT=${PGPORT:-5432}
PG_USER=${PGUSER:-omcgo}
PG_PASS=${PGPASSWORD:-omcgo123}
PG_DB=${PGDATABASE:-omcgo}
REDIS_HOST=localhost
REDIS_PORT=6379

# host mode 默认日志路径
ACS_LOG="$REPO_ROOT/run/logs/acs/acs.log"
APP_LOG="$REPO_ROOT/run/logs/app/app.log"

USE_COLOR=1
OUTPUT=md

while [[ $# -gt 0 ]]; do
    case $1 in
        --sn) SN=$2; shift 2 ;;
        --mml-task) MML=$2; shift 2 ;;
        --auto) AUTO=1; shift ;;
        --offset) OFFSET=$2; shift 2 ;;
        --mode) MODE=$2; shift 2 ;;
        --compose-file) COMPOSE_FILE=$2; shift 2 ;;
        --postgres-svc) POSTGRES_SVC=$2; shift 2 ;;
        --redis-svc) REDIS_SVC=$2; shift 2 ;;
        --acs-svc) ACS_SVC=$2; shift 2 ;;
        --app-svc) APP_SVC=$2; shift 2 ;;
        --logs-since) LOGS_SINCE=$2; shift 2 ;;
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
            sed -n '2,62p' "$0"
            exit 0
            ;;
        *) echo "Unknown arg: $1" >&2; exit 2 ;;
    esac
done

# ---- 工具依赖检查 ----
need_tools=(jq)
case "$MODE" in
    docker) need_tools+=(docker) ;;
    host)   need_tools+=(psql redis-cli) ;;
    *) echo "Unknown --mode: $MODE (expected docker | host)" >&2; exit 2 ;;
esac

for cmd in "${need_tools[@]}"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Required command not found: $cmd" >&2
        case "$MODE" in
            docker) echo "  brew install jq        # macOS / Linux 装 docker + jq 即可" >&2 ;;
            host)   echo "  brew install postgresql redis jq" >&2 ;;
        esac
        exit 2
    fi
done

# Docker 模式特检：compose-file 存在
if [[ "$MODE" == docker ]]; then
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        echo "Compose file not found: $COMPOSE_FILE" >&2
        echo "  指定 --compose-file 或 --mode host" >&2
        exit 2
    fi
fi

# ---- PG / Redis 抽象 ----
# psql_q  <psql args...>     例：psql_q -c "SELECT ..."  /  psql_q -F$'\t' -c "..."
# redis_q <redis-cli args...> 例：redis_q ZRANGE acs:taskq:$SN 0 -1
psql_q() {
    case "$MODE" in
        host)
            PGPASSWORD=$PG_PASS psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -t -A "$@"
            ;;
        docker)
            # exec -T 关掉 TTY，否则 docker 会带控制字符
            docker compose -f "$COMPOSE_FILE" exec -T \
                -e PGPASSWORD="$PG_PASS" \
                "$POSTGRES_SVC" \
                psql -U "$PG_USER" -d "$PG_DB" -t -A "$@"
            ;;
    esac
}

redis_q() {
    case "$MODE" in
        host)
            redis-cli -h $REDIS_HOST -p $REDIS_PORT "$@"
            ;;
        docker)
            docker compose -f "$COMPOSE_FILE" exec -T "$REDIS_SVC" redis-cli "$@"
            ;;
    esac
}

# ---- 日志抽象 ----
# Docker 模式下把 docker compose logs 拉到 /tmp 缓存文件再 grep；
# host 模式直接用磁盘文件。
TMPDIR_DIAG=$(mktemp -d -t diag-mml-XXXX)
trap 'rm -rf "$TMPDIR_DIAG"' EXIT

prepare_logs() {
    case "$MODE" in
        host)
            # NEXT_HINT 引用的日志路径是宿主上的真实文件
            ACS_LOG_REF="grep ... $ACS_LOG"
            APP_LOG_REF="grep ... $APP_LOG"
            ;;
        docker)
            ACS_LOG="$TMPDIR_DIAG/acs.log"
            APP_LOG="$TMPDIR_DIAG/app.log"
            docker compose -f "$COMPOSE_FILE" logs --no-color --since "$LOGS_SINCE" "$ACS_SVC" \
                > "$ACS_LOG" 2>/dev/null || true
            docker compose -f "$COMPOSE_FILE" logs --no-color --since "$LOGS_SINCE" "$APP_SVC" \
                > "$APP_LOG" 2>/dev/null || true
            # docker compose logs 每行有 "<svc> | " 前缀，过滤掉只留 JSON
            for f in "$ACS_LOG" "$APP_LOG"; do
                if [[ -s "$f" ]]; then
                    sed -E 's/^[[:space:]]*[a-zA-Z0-9_-]+[[:space:]]*\|[[:space:]]?//' "$f" > "$f.clean"
                    mv "$f.clean" "$f"
                fi
            done
            # NEXT_HINT 引用的命令是"实时拉日志再 grep"——脚本临时目录会在
            # 退出时清掉，给文件路径会让用户复制粘贴失败。
            ACS_LOG_REF="docker compose -f $COMPOSE_FILE logs --since $LOGS_SINCE $ACS_SVC | grep"
            APP_LOG_REF="docker compose -f $COMPOSE_FILE logs --since $LOGS_SINCE $APP_SVC | grep"
            ;;
    esac
}

# ---- 自动检测 ----
auto_detect() {
    local where="" picked
    if [[ -n "$MML" ]]; then
        where="WHERE id='$MML'"
    elif [[ -n "$SN" ]]; then
        # device_sns 是 jsonb 数组：用 @> 包含查询匹配数组元素
        where="WHERE device_sns @> to_jsonb('$SN'::text)"
    fi
    picked=$(psql_q -F$'\t' -c "
        SELECT id::text,
               COALESCE(device_sns->>0, ''),
               COALESCE(task_name, ''),
               status,
               created_at::text
        FROM mml_tasks $where
        ORDER BY created_at DESC
        OFFSET $OFFSET LIMIT 1;" 2>/dev/null)

    if [[ -z "$picked" ]]; then
        echo "auto-detect: 未找到符合条件的 mml_task（offset=$OFFSET, sn='$SN', mml='$MML'）" >&2
        return 1
    fi

    local p_id p_sn p_name p_status p_created
    IFS=$'\t' read -r p_id p_sn p_name p_status p_created <<< "$picked"

    [[ -z "$MML" ]] && MML=$p_id
    [[ -z "$SN"  ]] && SN=$p_sn

    echo "auto-detected:" >&2
    echo "  mml_task : $p_id  ($p_name, status=$p_status, created=$p_created)" >&2
    echo "  device_sn: $SN" >&2
}

if [[ "$AUTO" == 1 || ( -z "$SN" && -z "$MML" ) ]]; then
    if ! auto_detect; then
        exit 2
    fi
fi

# ---- 校验 ----
if [[ -z "$SN" || -z "$MML" ]]; then
    echo "Usage: $0 --sn <SN> --mml-task <UUID>   (or --auto to detect)" >&2
    echo "  (See $0 --help for full options.)" >&2
    exit 2
fi

# ---- 颜色 ----
if [[ "$USE_COLOR" == 1 && -t 1 ]]; then
    GRN=$'\e[32m'; RED=$'\e[31m'; YLW=$'\e[33m'; CYN=$'\e[36m'; DIM=$'\e[2m'; RST=$'\e[0m'
else
    GRN=""; RED=""; YLW=""; CYN=""; DIM=""; RST=""
fi

# 拉日志（docker 模式必须，host 模式 no-op）
prepare_logs

# ---- 状态变量 ----
declare -a CHECK_NAMES CHECK_STATUS CHECK_DETAILS
LAST_OK=0
NEXT_HINT=""

record() {
    CHECK_NAMES[$1]=$2
    CHECK_STATUS[$1]=$3
    CHECK_DETAILS[$1]=$4
    if [[ $3 == ok ]]; then
        LAST_OK=$1
    fi
}

# ---- Checkpoint 1: mml_tasks ----
mml_row=$(psql_q -F$'\t' -c "SELECT status, execute_type, COALESCE(jsonb_array_length(device_sns),0), COALESCE(jsonb_array_length(commands),0), success_count, failed_count, COALESCE(started_at::text, ''), COALESCE(finished_at::text, '') FROM mml_tasks WHERE id='$MML';" 2>/dev/null)
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
    dt_count=$(psql_q -c "SELECT COUNT(*) FROM device_tasks WHERE source='mml' AND source_id='$MML';" 2>/dev/null | tr -d ' ')
    dt_count=${dt_count:-0}

    if [[ "$dt_count" == "0" ]]; then
        record 2 "device_tasks 派生" fail "0 行（Fanouter 未派生）"
        NEXT_HINT="${APP_LOG_REF} -E 'build tr069 params failed|skip command without rpc_method'"
    else
        dt_row=$(psql_q -F$'\t' -c "SELECT id, status, COALESCE(cwmp_id,''), method, COALESCE(sent_at::text,''), (result IS NOT NULL), params FROM device_tasks WHERE source='mml' AND source_id='$MML' AND device_sn='$SN' ORDER BY command_index, device_index LIMIT 1;" 2>/dev/null)
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
    queue_members=$(redis_q ZRANGE "acs:taskq:$SN" 0 -1 2>/dev/null)
    if [[ -z "$queue_members" ]]; then
        task_data=$(redis_q HGET "acs:task:$DT_ID" data 2>/dev/null)
        if [[ -n "$task_data" ]]; then
            t_status=$(echo "$task_data" | jq -r .status 2>/dev/null)
            record 3 "Redis 队列" ok "队列已派发；acs:task:$DT_ID 存在，status=$t_status"
        else
            if [[ "$dt_status" == "pending" ]]; then
                record 3 "Redis 队列" fail "PG=pending 但 Redis 无队列项也无 task hash（不一致）"
                NEXT_HINT="${APP_LOG_REF} -E 'RestorePendingQueues|task recovered'"
            else
                record 3 "Redis 队列" warn "Redis 无队列项；PG status=$dt_status，可能任务已收尾或 task TTL 过期"
            fi
        fi
    else
        if echo "$queue_members" | grep -q "$DT_ID"; then
            record 3 "Redis 队列" ok "task_id 在 acs:taskq:$SN 中等待派发"
        else
            record 3 "Redis 队列" warn "队列有 task 但不含 ${DT_ID}（已派发或被其他任务挤掉）"
        fi
    fi
else
    record 3 "Redis 队列" na "上游未通过"
fi

# ---- Checkpoint 4: CPE Inform ----
heartbeat=$(redis_q GET "acs:heartbeat:$SN" 2>/dev/null)
hb_ttl=$(redis_q TTL "acs:heartbeat:$SN" 2>/dev/null)
if [[ -n "$heartbeat" && "$hb_ttl" != "-2" ]]; then
    record 4 "CPE Inform" ok "heartbeat=$heartbeat ttl=${hb_ttl}s"
else
    record 4 "CPE Inform" fail "heartbeat 不存在（CPE 离线 / 网络隔离 / 未上来）"
    NEXT_HINT="抓包 'tcp port 7547' 或 ${ACS_LOG_REF} 'ACS Inform processing' | grep ${SN}"
fi

# ---- Checkpoint 5: ACS 下发 SOAP ----
if [[ -s "$ACS_LOG" && ${CHECK_STATUS[2]:-} == ok ]]; then
    send_log=$(grep "ACS sending RPC request from task" "$ACS_LOG" 2>/dev/null \
        | jq -c "select(.task_id==\"$DT_ID\")" 2>/dev/null | tail -1)
    if [[ -z "$send_log" ]]; then
        record 5 "ACS 下发 SOAP" fail "ACS log 未见下发记录（窗口 --logs-since=${LOGS_SINCE}）"
        NEXT_HINT="放宽时间窗口：--logs-since 1h；或看 [4] CPE 是否真的 Inform"
    else
        cwmp=$(echo "$send_log" | jq -r .cwmp_id)
        size=$(echo "$send_log" | jq -r .soap_size)
        body=$(echo "$send_log" | jq -r .soap_body)

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
    if [[ ! -s "$ACS_LOG" ]]; then
        record 5 "ACS 下发 SOAP" na "ACS 日志为空: ${ACS_LOG}（用 --logs-since 放宽 / --acs-svc 改服务名）"
    else
        record 5 "ACS 下发 SOAP" na "上游未通过"
    fi
fi

# ---- Checkpoint 6: CPE 响应 ----
if [[ -s "$ACS_LOG" && ${CHECK_STATUS[5]:-} == ok && -n "${CWMP_ID:-}" ]]; then
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
        record 6 "CPE 响应" ok "ACS 收到响应（cwmp_id=${CWMP_ID}）"
    else
        record 6 "CPE 响应" fail "ACS log 未见响应（CPE 没回 / 报文非法 / Cookie 失效）"
        NEXT_HINT="抓包确认 CPE 是否真的 POST 了响应；${ACS_LOG_REF} 'no valid session cookie'"
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
            err_msg=$(psql_q -c "SELECT COALESCE(error_message,'') FROM device_tasks WHERE id='$DT_ID';" 2>/dev/null)
            record 7 "ACS 标记 failed" warn "device_task.status=failed error=$err_msg"
            NEXT_HINT="任务级失败已标记，[8] 仍会聚合（finalizeIfComplete 把 failed 计入）"
            ;;
        sent)
            record 7 "ACS 标记完成" fail "device_task 持续 status=sent，sent_at=$dt_sent_at"
            NEXT_HINT="ACS 没收到响应（看 [6]）或 cwmp_id 没匹配；${ACS_LOG_REF} 'get task by cwmp_id'"
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
# 仅当上游已经走到"任务有派发且至少一个 device_task 收尾"时才该有 [8] 判断；
# 否则走 na 避免误报（app 进程长时间没重启时 "subscribed" 启动日志会落在窗口外，
# 但这跟当前任务卡不卡无关）。
if [[ "$mml_status" == "completed" || "$mml_status" == "failed" ]]; then
    upstream_finalized=1
elif [[ "${dt_status:-}" == "completed" || "${dt_status:-}" == "failed" ]]; then
    upstream_finalized=1
else
    upstream_finalized=0
fi

if [[ "$upstream_finalized" == 0 ]]; then
    record 8 "MML 聚合器收尾" na "上游未到收尾阶段（mml=${mml_status:-?}, dt=${dt_status:-?}）"
elif [[ ! -s "$APP_LOG" ]]; then
    record 8 "MML 聚合器收尾" na "APP 日志为空: ${APP_LOG}（放宽 --logs-since 或换 --app-svc）"
else
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
    elif [[ "$mml_status" == "completed" || "$mml_status" == "failed" ]]; then
        # mml_tasks 已收尾，但日志没找到 finalized — 多半是日志窗口轮转过了
        record 8 "MML 聚合器收尾" ok "mml_tasks.status=$mml_status（finalized 日志已不在 --logs-since=${LOGS_SINCE} 窗口内）"
    else
        record 8 "MML 聚合器收尾" fail "device_task 已收尾但 mml task 未 finalize（NATS 事件链断？）"
        NEXT_HINT="docker compose -f $COMPOSE_FILE exec nats nats sub 'task.>' 观察事件；并检查 'task completion event bridge subscribed' 启动日志"
    fi
fi

# ---- 输出 ----
if [[ "$OUTPUT" == json ]]; then
    printf '{"sn":"%s","mml_task":"%s","mode":"%s","checkpoints":[' "$SN" "$MML" "$MODE"
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
    echo "  Mode:      $MODE"
    if [[ "$MODE" == docker ]]; then
        echo "  Compose:   $COMPOSE_FILE  (logs --since=$LOGS_SINCE)"
    fi
    echo "  SN:        $SN"
    echo "  MML Task:  $MML"
    echo "  Time:      $(date '+%Y-%m-%d %H:%M:%S')"
    echo
    # 中英文混合列宽对齐：CJK 全角字符按显示宽 2 计算，pad 到 22 col。
    # 优先用 python3（最稳）；没装则用 awk 的字符级近似（UTF-8 locale）；都不行就 fallback 到 printf 的字节宽度。
    if command -v python3 >/dev/null 2>&1; then
        pad_display() {
            python3 -c "
import sys, unicodedata
s = sys.argv[1]; target = int(sys.argv[2])
w = sum(2 if unicodedata.east_asian_width(c) in ('W', 'F') else 1 for c in s)
print(s + ' ' * max(0, target - w))
" "$1" "${2:-22}"
        }
    else
        pad_display() { printf '%-30s' "$1"; }
    fi

    echo "  Step | Status | Checkpoint             | Detail"
    echo "  -----+--------+------------------------+----------------------------------"
    for i in 1 2 3 4 5 6 7 8; do
        name_padded=$(pad_display "${CHECK_NAMES[$i]}" 22)
        printf "  [%d]  |   %s    | %s | %s\n" \
            "$i" "$(sym "${CHECK_STATUS[$i]}")" "$name_padded" "${CHECK_DETAILS[$i]}"
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

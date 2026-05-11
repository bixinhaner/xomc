#!/usr/bin/env bash
# diag_mml_gpn_probe.sh
#
# MML 命令路径诊断工具
# =======================
# 向一台在线 CPE 发送 GetParameterNames("Device.", false)，拿回真实数据模型，
# 与 mml_params 表里的 tr069_path 比对，找出"DB 配了但 CPE 实际不存在"的路径。
# 这类路径是 LST/MOD/DSP/SET 等 MML 命令被 CPE silent drop 的根本原因。
#
# 数据通路依据：
#   - 任务创建 → POST /api/v1/devices/tasks?device_sn=<SN>（X-API-Key 鉴权）
#   - 任务结果落库 → internal/acs/handler.go:695-699 把原始 SOAP body 存进
#     device_tasks.result.raw_response（JSONB）
#   - GPN(Device., NextLevel=false) 按 TR-069 §A.3.2.4 返回 Device. 下全部
#     可访问 path 的完整集合（叶子 + 中间节点 + 多实例展开后的具体索引）
#
# 用法 / 故障排查见 docs/operations/diag-mml-gpn-probe.md。
# 设计依据：解决"LST DEVICE_INFO 命令报文被 baicells CPE 丢弃"的诊断 gap。

set -euo pipefail

# ────────────────────────────────────────────────────────────────────
# 默认参数 & 帮助
# ────────────────────────────────────────────────────────────────────
DSN=""
API_URL="${OMC_API_URL:-http://localhost:8081}"
API_KEY="${OMCCTL_API_KEY:-}"
DEVICE_SN="AUTO"
TIMEOUT_SEC=120
OUTPUT_DIR=""
ROOT_PATH="Device."
NEXT_LEVEL="false"

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

发送 GetParameterNames 探针到一台在线 CPE，对比 mml_params 表，输出诊断报告。

必需参数:
  --dsn        <pg_dsn>   PostgreSQL 连接串 (e.g. postgres://omc:omc@host:5432/omcgo)
  --api-key    <key>      OMC App 的 X-API-Key（或设环境变量 OMCCTL_API_KEY）

可选参数:
  --api        <url>      OMC App API 基地址，默认 \$OMC_API_URL 或 http://localhost:8081
  --device-sn  <SN>       指定 CPE SN，默认 AUTO（自动挑一台 status='active' 且
                          last_inform_at < 10 分钟的设备）
  --timeout    <sec>      等 task 完成的超时秒数，默认 120
  --output     <dir>      输出目录，默认 /tmp/mml-diag-<timestamp>
  --root-path  <path>     GPN 起始路径，默认 Device.（TR-069 spec 唯一根之一）
  --next-level            若设置则 NextLevel=true（仅返回根的直接子节点）。
                          默认 false：返回 Device. 下的完整数据模型树。
  -h | --help             显示本帮助

输出文件：
  report.md            人看的摘要 + 头部样本
  raw_response.xml     CPE 返回的原始 SOAP body（调试用）
  cpe_paths.txt        CPE 真实 path 列表（每行一条，sort -u）
  db_paths.txt         mml_params 表全部不重复 tr069_path
  missing_in_cpe.txt   ❌ DB 配了但 CPE 不存在 — 这些就是 MML 命令被丢的根因
  extra_in_cpe.txt     ℹ️ CPE 有但 DB 未收录 — 可补充
  matched.txt          ✅ 双方都有
  fixup.sql            修复草稿（注释形式，需 review 后手动启用）
  meta.json            运行元数据（设备/task/统计数）

示例:
  $(basename "$0") --dsn "postgres://omc:omc@localhost:5432/omcgo" \\
                   --api-key "\$OMCCTL_API_KEY" \\
                   --api http://10.0.0.1:8081

  # 指定具体设备 + 自定义输出目录
  $(basename "$0") --dsn ... --api-key ... \\
                   --device-sn "BAI-12345" \\
                   --output ./diag-output

Exit codes:
  0  成功
  1  使用错误（缺参数 / 缺依赖）
  2  未找到在线设备
  3  任务创建失败
  4  任务超时未完成
  5  任务返回 SOAP Fault
  6  响应无法解析（XML 结构异常）
EOF
}

# ────────────────────────────────────────────────────────────────────
# 参数解析
# ────────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
    --dsn)          DSN="$2"; shift 2 ;;
    --api)          API_URL="$2"; shift 2 ;;
    --api-key)      API_KEY="$2"; shift 2 ;;
    --device-sn)    DEVICE_SN="$2"; shift 2 ;;
    --timeout)      TIMEOUT_SEC="$2"; shift 2 ;;
    --output)       OUTPUT_DIR="$2"; shift 2 ;;
    --root-path)    ROOT_PATH="$2"; shift 2 ;;
    --next-level)   NEXT_LEVEL="true"; shift ;;
    -h|--help)      usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

[[ -z "$DSN" ]] && { echo "ERROR: --dsn is required (or set PGSERVICE/PGHOST etc)" >&2; exit 1; }
[[ -z "$API_KEY" ]] && { echo "ERROR: --api-key (or OMCCTL_API_KEY env) is required" >&2; exit 1; }

OUTPUT_DIR="${OUTPUT_DIR:-/tmp/mml-diag-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$OUTPUT_DIR"

# ────────────────────────────────────────────────────────────────────
# 依赖预检
# ────────────────────────────────────────────────────────────────────
need() {
    command -v "$1" >/dev/null 2>&1 || {
        echo "ERROR: missing dependency: $1" >&2
        exit 1
    }
}
need psql
need curl
need jq
need xmllint

# ────────────────────────────────────────────────────────────────────
# 公用函数
# ────────────────────────────────────────────────────────────────────
log() { printf '[%(%H:%M:%S)T] %s\n' -1 "$*" >&2; }

# psql_at: -At 无对齐无表头，安全输出单/多行；-v 参数化避免 SQL 注入
psql_at() {
    local sql="$1"; shift
    psql "$DSN" -X --no-psqlrc --set ON_ERROR_STOP=1 -At "$@" -c "$sql"
}

# api_post / api_get: 走 X-API-Key 鉴权；-f 失败即非零退出
api_post() {
    curl -fsSL --max-time 30 \
        -H "X-API-Key: $API_KEY" \
        -H "Content-Type: application/json" \
        -X POST --data "$2" "$API_URL$1"
}

# ────────────────────────────────────────────────────────────────────
# 步骤 1：选设备
# ────────────────────────────────────────────────────────────────────
if [[ "$DEVICE_SN" == "AUTO" ]]; then
    log "AUTO 模式：查找最近 10 分钟有 inform 的在线设备…"
    DEVICE_SN=$(psql_at "
        SELECT serial_number FROM devices
        WHERE status = 'active'
          AND last_inform_at IS NOT NULL
          AND last_inform_at > NOW() - INTERVAL '10 minutes'
        ORDER BY last_inform_at DESC
        LIMIT 1")
    if [[ -z "$DEVICE_SN" ]]; then
        echo "ERROR: 没找到在线设备（status='active' 且 last_inform_at < 10 分钟）" >&2
        echo "  → 检查：psql \"\$DSN\" -c \"SELECT serial_number,status,last_inform_at FROM devices ORDER BY last_inform_at DESC NULLS LAST LIMIT 10\"" >&2
        exit 2
    fi
    log "选中设备: $DEVICE_SN"
fi

DEVICE_META=$(psql_at \
    "SELECT serial_number || '|' || status || '|' || COALESCE(last_inform_at::text,'never')
     FROM devices WHERE serial_number = :'sn'" \
    -v sn="$DEVICE_SN")
if [[ -z "$DEVICE_META" ]]; then
    echo "ERROR: 设备不存在: $DEVICE_SN" >&2
    exit 2
fi
log "设备元信息（sn|status|last_inform_at）: $DEVICE_META"

# ────────────────────────────────────────────────────────────────────
# 步骤 2：创建 GetParameterNames task
# ────────────────────────────────────────────────────────────────────
log "POST /api/v1/devices/tasks — method=GetParameterNames path=$ROOT_PATH next_level=$NEXT_LEVEL"
REQ_BODY=$(jq -n \
    --arg sn "$DEVICE_SN" \
    --arg path "$ROOT_PATH" \
    --argjson nl "$NEXT_LEVEL" \
    '{device_sn:$sn, method:"GetParameterNames",
      params:{path:$path, next_level:$nl},
      description:"mml-diag GPN probe", priority:5}')

RESP=$(api_post "/api/v1/devices/tasks?device_sn=$DEVICE_SN" "$REQ_BODY") || {
    echo "ERROR: 任务创建 HTTP 调用失败" >&2
    exit 3
}

# 后端 response 包络是 {ret, msg, data:{...task...}}；保险起见做 fallback
TASK_ID=$(echo "$RESP" | jq -r '.data.id // .id // empty')
if [[ -z "$TASK_ID" || "$TASK_ID" == "null" ]]; then
    echo "ERROR: 任务创建响应无 task id" >&2
    echo "  response: $RESP" >&2
    exit 3
fi
log "task 创建: $TASK_ID"

# ────────────────────────────────────────────────────────────────────
# 步骤 3：轮询等待 task 完成
# ────────────────────────────────────────────────────────────────────
log "等待 task 完成（超时 ${TIMEOUT_SEC}s，每 2s 轮询）…"
DEADLINE=$(($(date +%s) + TIMEOUT_SEC))
while true; do
    ROW=$(psql_at \
        "SELECT status || '|' || COALESCE(error_code::text,'') || '|' || COALESCE(error_message,'')
         FROM device_tasks WHERE id = :'tid' AND device_sn = :'sn'" \
        -v tid="$TASK_ID" -v sn="$DEVICE_SN")
    [[ -z "$ROW" ]] && { echo "ERROR: task 在 DB 中找不到（被清理？）: $TASK_ID" >&2; exit 4; }

    STATUS="${ROW%%|*}"
    case "$STATUS" in
    completed)
        log "task 完成"
        break
        ;;
    failed)
        REST="${ROW#*|}"
        echo "ERROR: task 失败 — code=${REST%%|*} msg=${REST#*|}" >&2
        exit 5
        ;;
    "")
        echo "ERROR: task status 为空" >&2
        exit 4
        ;;
    *) ;;  # pending / sent / in_progress → 继续等
    esac

    if (( $(date +%s) > DEADLINE )); then
        echo "ERROR: 等待超时 ${TIMEOUT_SEC}s，task 仍为 $STATUS" >&2
        echo "  → 设备可能未在线 / Connection Request 未生效 / inform 间隔太长" >&2
        echo "  → 检查 ACS 日志：grep $TASK_ID  acs.log" >&2
        exit 4
    fi
    sleep 2
done

# ────────────────────────────────────────────────────────────────────
# 步骤 4：提取 raw_response 并解析 ParameterList
# ────────────────────────────────────────────────────────────────────
log "提取 SOAP 响应…"
psql_at \
    "SELECT result->>'raw_response' FROM device_tasks WHERE id = :'tid' AND device_sn = :'sn'" \
    -v tid="$TASK_ID" -v sn="$DEVICE_SN" > "$OUTPUT_DIR/raw_response.xml"

if [[ ! -s "$OUTPUT_DIR/raw_response.xml" ]]; then
    echo "ERROR: device_tasks.result.raw_response 为空" >&2
    echo "  → ACS 可能 mark completed 但未写入 raw_response（异常路径）" >&2
    exit 6
fi

# GetParameterNamesResponse 结构（TR-069 §A.3.2.4）：
#   <ParameterList SOAP-ENC:arrayType="cwmp:ParameterInfoStruct[N]">
#     <ParameterInfoStruct><Name>Device.X.Y</Name><Writable>1</Writable></ParameterInfoStruct>
#     ...
#   </ParameterList>
#
# 用 xmllint local-name() 绕开命名空间差异（不同设备 cwmp/无前缀混杂）
log "用 xmllint 解析 ParameterInfoStruct/Name 节点…"
xmllint --xpath \
    "//*[local-name()='ParameterInfoStruct']/*[local-name()='Name']/text()" \
    "$OUTPUT_DIR/raw_response.xml" 2>/dev/null \
    | tr '\n' '\n' | awk 'NF { gsub(/^[ \t]+|[ \t]+$/,""); print }' \
    | sort -u > "$OUTPUT_DIR/cpe_paths.txt" || true

CPE_COUNT=$(wc -l < "$OUTPUT_DIR/cpe_paths.txt" | tr -d ' ')
if [[ "$CPE_COUNT" -eq 0 ]]; then
    echo "ERROR: xmllint 解析后 0 条 path" >&2
    echo "  raw_response.xml 前 30 行：" >&2
    head -30 "$OUTPUT_DIR/raw_response.xml" >&2
    exit 6
fi
log "CPE 返回 $CPE_COUNT 条 path"

# ────────────────────────────────────────────────────────────────────
# 步骤 5：取 DB mml_params 全部 path
# ────────────────────────────────────────────────────────────────────
log "从 mml_params 表取全部 tr069_path（distinct）…"
psql_at \
    "SELECT DISTINCT tr069_path FROM mml_params
     WHERE tr069_path IS NOT NULL AND tr069_path <> ''
     ORDER BY tr069_path" > "$OUTPUT_DIR/db_paths.txt"
DB_COUNT=$(wc -l < "$OUTPUT_DIR/db_paths.txt" | tr -d ' ')
log "DB 有 $DB_COUNT 条不重复 path"

# ────────────────────────────────────────────────────────────────────
# 步骤 6：差异计算
# ────────────────────────────────────────────────────────────────────
# comm 要求文件已 sort（两边都用 sort -u 保证）
comm -23 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/missing_in_cpe.txt"
comm -13 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/extra_in_cpe.txt"
comm -12 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/matched.txt"

MISS=$(wc -l <  "$OUTPUT_DIR/missing_in_cpe.txt" | tr -d ' ')
EXTRA=$(wc -l < "$OUTPUT_DIR/extra_in_cpe.txt"   | tr -d ' ')
MATCH=$(wc -l < "$OUTPUT_DIR/matched.txt"        | tr -d ' ')
log "diff: matched=$MATCH  missing=$MISS  extra=$EXTRA"

# ────────────────────────────────────────────────────────────────────
# 步骤 7：报告 + 元数据
# ────────────────────────────────────────────────────────────────────
{
    echo "# MML 路径诊断报告"
    echo
    echo "- 生成时间: $(date)"
    echo "- 设备 SN: \`$DEVICE_SN\`"
    echo "- 设备元信息: \`$DEVICE_META\`"
    echo "- GPN 参数: path=\`$ROOT_PATH\` next_level=\`$NEXT_LEVEL\`"
    echo "- task_id: \`$TASK_ID\`"
    echo
    echo "## 数据摘要"
    echo
    echo "| 维度 | 数量 |"
    echo "|---|---|"
    echo "| CPE 真实 path | $CPE_COUNT |"
    echo "| DB mml_params 不重复 path | $DB_COUNT |"
    echo "| ✅ 双方都有 (matched) | $MATCH |"
    echo "| ❌ DB 配了但 CPE 不存在 | $MISS |"
    echo "| ℹ️ CPE 有但 DB 未收录 | $EXTRA |"
    echo
    echo "## ❌ 需要修复的 DB path"
    echo
    echo "**这些就是 MML 命令被 CPE 丢弃的根因**。修复选项："
    echo
    echo "1. **path 拼写错** → \`UPDATE mml_params SET tr069_path = '<正确>' WHERE tr069_path = '<错>'\`"
    echo "2. **path 不属于此 product** → 解除 \`mml_command_params_rel\` 绑定，或按 product_id 拆 param 库"
    echo "3. **CPE 数据模型未实现** → 通知设备侧补齐，DB 暂保留"
    echo
    echo "### 前 50 条样本（完整见 missing_in_cpe.txt）"
    echo
    echo '```'
    head -50 "$OUTPUT_DIR/missing_in_cpe.txt"
    if [[ "$MISS" -gt 50 ]]; then
        echo "... 还有 $((MISS - 50)) 条"
    fi
    echo '```'
    echo
    echo "## ℹ️ CPE 有但 DB 未收录"
    echo
    echo "可作为后续 mml_params 扩充候选。前 20 条："
    echo
    echo '```'
    head -20 "$OUTPUT_DIR/extra_in_cpe.txt"
    if [[ "$EXTRA" -gt 20 ]]; then
        echo "... 还有 $((EXTRA - 20)) 条"
    fi
    echo '```'
    echo
    echo "## 文件清单"
    echo
    echo "| 文件 | 说明 |"
    echo "|---|---|"
    echo "| raw_response.xml | 原始 SOAP body（调试用） |"
    echo "| cpe_paths.txt | CPE 真实 path 列表 |"
    echo "| db_paths.txt | DB 全部不重复 path |"
    echo "| missing_in_cpe.txt | ❌ 修复目标 |"
    echo "| extra_in_cpe.txt | ℹ️ 候选扩充 |"
    echo "| matched.txt | ✅ 已对齐 |"
    echo "| fixup.sql | 修复草稿（注释，需 review） |"
    echo "| meta.json | 运行元数据 |"
} > "$OUTPUT_DIR/report.md"

# ────────────────────────────────────────────────────────────────────
# 步骤 8：修复草稿 SQL（仅注释，不直接 DELETE/UPDATE）
# ────────────────────────────────────────────────────────────────────
{
    echo "-- mml-diag GPN probe 修复草稿"
    echo "-- 时间: $(date)"
    echo "-- 设备: $DEVICE_SN  task_id: $TASK_ID"
    echo "--"
    echo "-- ⚠️  本文件全部为注释。每条路径都需人工 review 后取消注释才会执行。"
    echo "-- 工具 NOT 直接修改 DB，避免误删运维不熟悉的历史数据。"
    echo "--"
    echo "-- 三种典型修法（按情况二选一）："
    echo "--   A. path 拼错 → UPDATE mml_params SET tr069_path='<correct>' WHERE tr069_path='<wrong>';"
    echo "--   B. 不属于此设备 → DELETE FROM mml_command_params_rel WHERE param_id IN ("
    echo "--                       SELECT id FROM mml_params WHERE tr069_path='<wrong>');"
    echo "--   C. CPE 该补 → 保留 DB 不动，通知设备侧"
    echo
    while IFS= read -r p; do
        [[ -z "$p" ]] && continue
        esc=${p//\'/\'\'}
        echo "-- ❌ MISSING IN CPE: $p"
        echo "-- UPDATE mml_params SET tr069_path = '<CORRECT_PATH>' WHERE tr069_path = '$esc';"
        echo
    done < "$OUTPUT_DIR/missing_in_cpe.txt"
} > "$OUTPUT_DIR/fixup.sql"

# ────────────────────────────────────────────────────────────────────
# 步骤 9：元数据 JSON（机器消费）
# ────────────────────────────────────────────────────────────────────
jq -n \
    --arg ts "$(date -Iseconds 2>/dev/null || date +%FT%T)" \
    --arg sn "$DEVICE_SN" \
    --arg meta "$DEVICE_META" \
    --arg tid "$TASK_ID" \
    --arg path "$ROOT_PATH" \
    --argjson nl "$NEXT_LEVEL" \
    --argjson cpe "$CPE_COUNT" \
    --argjson db "$DB_COUNT" \
    --argjson m "$MATCH" \
    --argjson miss "$MISS" \
    --argjson extra "$EXTRA" \
    '{
      generated_at: $ts,
      device_sn: $sn,
      device_meta: $meta,
      task_id: $tid,
      gpn: {root_path: $path, next_level: $nl},
      counts: {cpe: $cpe, db: $db, matched: $m, missing_in_cpe: $miss, extra_in_cpe: $extra}
    }' > "$OUTPUT_DIR/meta.json"

log "──────────────────────────────────────────────"
log "完成。结果目录: $OUTPUT_DIR"
log "报告:          $OUTPUT_DIR/report.md"
log "diff 详情:     $OUTPUT_DIR/{missing,extra,matched}_in_cpe.txt"
log "修复草稿:      $OUTPUT_DIR/fixup.sql"
log "原始 XML:      $OUTPUT_DIR/raw_response.xml"
log "──────────────────────────────────────────────"
exit 0

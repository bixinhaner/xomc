#!/usr/bin/env bash
# trigger_device_config_upload.sh
#
# 触发设备上传配置文件（TR-069 Upload, FileType=3 厂商配置文件）
# =================================================
# 调 OMC 既有 backup API:
#   POST /api/v1/backup/tasks  body={task_type:"config_only", target_type:"device",
#                                   target_ids:[<device_id_uuid>]}
# 由 backup service 走完整链路:
#   生成 JWT 签名的 upload URL → Upload RPC 入队 → ACS 通过 Connection Request 唤醒
#   → CPE PUT 文件到 /upload/<token> → upload handler 落 MinIO → file_path_recorder
#   把对象路径回写 backup_tasks.file_path
#
# 工具职责:
#   1. 设备 SN → device.id UUID 反查（PG）
#   2. POST 创建 backup task，拿 task_id
#   3. 轮询 GET /api/v1/backup/tasks/:id 直到 status ∈ {completed, failed}
#   4. 输出 file_path（MinIO 对象路径）+ MinIO presigned URL（如可生成）
#
# 零参数运行（推荐）：自动从 OMC App 配置文件读 DSN 与 API URL，
# 自动生成临时 API Key（用完撤销）。覆盖任何一项：传 --config / --dsn / --api / --api-key。
#
# 数据通路依据：
#   - 任务创建：handler.go:152 POST /api/v1/backup/tasks
#   - task↔file_path 回填：internal/backup/file_path_recorder.go (T-0079)
#   - Upload RPC SOAP 模板：pkg/soap/templates.go UploadData
#   - upload handler：internal/acs/upload/handler.go（FileType=3 走 backup 桥）

set -euo pipefail

# ────────────────────────────────────────────────────────────────────
# 默认参数
# ────────────────────────────────────────────────────────────────────
CONFIG=""
DSN=""
API_URL=""
API_KEY="${OMCCTL_API_KEY:-}"
API_KEY_USER="admin"
DEVICE_SN=""
TIMEOUT_SEC=180
POLL_INTERVAL=3
OUTPUT_DIR=""
DOWNLOAD="false"   # 是否在 task 完成后从 MinIO 拉取真实文件
KEEP_API_KEY="false"

# psql 路由模式（同 diag_mml_gpn_probe.sh，见该脚本说明）
PSQL_DOCKER_MODE="${PSQL_DOCKER:-auto}"
PSQL_DOCKER_CONTAINER=""

AUTO_GENERATED_KEY_ID=""

usage() {
    cat <<EOF
Usage: $(basename "$0") --device-sn <SN> [options]

触发指定设备上传配置文件（TR-069 Upload FileType=3）并等待落地 MinIO。

【零参数运行（推荐）】
   $(basename "$0") --device-sn ABC123
   工具自动：
     1. 搜索 OMC App 配置文件（顺序同 diag_mml_gpn_probe.sh）
     2. 从配置读 db.dsn 与 server.port，推断 API URL
     3. 自动为 admin 用户生成 1 天有效期的临时 API Key（退出时撤销）
     4. 反查 device.id UUID，发起 config_only backup task
     5. 轮询直到 completed/failed 或超时
     6. 输出 file_path

参数：
  --device-sn <SN>        CPE SN（**必填**）
  --config    <path>      OMC App 配置文件路径
  --dsn       <pg_dsn>    覆盖 config.db.dsn
  --api       <url>       覆盖 API URL
  --api-key   <key>       现成 API Key（或 \$OMCCTL_API_KEY）；否则自动生成
  --user      <username>  自动生成 key 的归属用户，默认 admin
  --keep-api-key          不撤销自动生成的临时 key（默认退出时撤销）
  --timeout   <sec>       等 task 完成的总超时秒数，默认 180
  --poll      <sec>       轮询间隔，默认 3
  --output    <dir>       输出目录，默认 /tmp/device-config-<SN>-<timestamp>
  --download              在 task 完成后从 MinIO 拉取真实配置文件到 output 目录
                          （需 mc 客户端或 docker-minio 容器内调 mc）
  --psql-docker <mode>    psql 路由模式（auto/off/<container>），同 diag_mml_gpn_probe.sh
  -h | --help             显示本帮助

输出文件（位于 \$OUTPUT_DIR）：
  task.json            backup_task 最终状态（file_path / status / 时长）
  device.json          设备元数据（id/SN/manufacturer/firmware）
  meta.json            运行元数据（API Key / config path / timestamps）
  config.<ext>         （--download 时）真实配置文件副本

Exit codes:
  0  成功（task=completed 且 file_path 非空）
  1  使用错误（缺依赖 / 缺 --device-sn / 配置定位失败）
  2  设备查询失败 / 设备不存在 / 软删除
  3  task 创建失败 / API Key 生成失败
  4  task 超时未完成
  5  task=failed
  6  task=completed 但 file_path 仍为空（recorder 未回填或 ACS 路径异常）
EOF
}

# ────────────────────────────────────────────────────────────────────
# 参数解析
# ────────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
    --config)        CONFIG="$2"; shift 2 ;;
    --dsn)           DSN="$2"; shift 2 ;;
    --api)           API_URL="$2"; shift 2 ;;
    --api-key)       API_KEY="$2"; shift 2 ;;
    --user)          API_KEY_USER="$2"; shift 2 ;;
    --keep-api-key)  KEEP_API_KEY="true"; shift ;;
    --device-sn)     DEVICE_SN="$2"; shift 2 ;;
    --timeout)       TIMEOUT_SEC="$2"; shift 2 ;;
    --poll)          POLL_INTERVAL="$2"; shift 2 ;;
    --output)        OUTPUT_DIR="$2"; shift 2 ;;
    --download)      DOWNLOAD="true"; shift ;;
    --psql-docker)   PSQL_DOCKER_MODE="$2"; shift 2 ;;
    -h|--help)       usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

[[ -z "$DEVICE_SN" ]] && { echo "ERROR: --device-sn 必填" >&2; usage >&2; exit 1; }

OUTPUT_DIR="${OUTPUT_DIR:-/tmp/device-config-${DEVICE_SN}-$(date +%Y%m%d-%H%M%S)}"
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
need curl; need jq
# psql 走 detect_psql_route 决策

log() { echo "[$(date +%H:%M:%S)] $*" >&2; }

# ────────────────────────────────────────────────────────────────────
# psql 路由（与 diag_mml_gpn_probe.sh 完全一致）
# ────────────────────────────────────────────────────────────────────
detect_psql_route() {
    case "$PSQL_DOCKER_MODE" in
    off)
        command -v psql >/dev/null 2>&1 || {
            echo "ERROR: --psql-docker off 但宿主机无 psql" >&2; exit 1
        }
        log "psql: host (--psql-docker off)"
        ;;
    auto)
        local c cid
        if command -v docker >/dev/null 2>&1; then
            for c in goomc-postgres goomc_postgres_1 docker-postgres-1 deployments-postgres-1 postgres; do
                if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$c"; then
                    PSQL_DOCKER_CONTAINER="$c"
                    log "psql: docker exec -i $c (auto)"
                    return
                fi
            done
            cid=$(docker ps --format '{{.ID}} {{.Image}}' 2>/dev/null | awk '/timescale|postgres/{print $1; exit}')
            if [[ -n "$cid" ]]; then
                PSQL_DOCKER_CONTAINER="$cid"
                log "psql: docker exec -i $cid (image match)"
                return
            fi
        fi
        if command -v psql >/dev/null 2>&1; then
            log "psql: host (fallback)"
            return
        fi
        echo "ERROR: docker 找不到 postgres 容器，宿主机也无 psql" >&2; exit 1
        ;;
    *)
        command -v docker >/dev/null 2>&1 || {
            echo "ERROR: --psql-docker $PSQL_DOCKER_MODE 但无 docker" >&2; exit 1
        }
        docker ps --format '{{.Names}}\t{{.ID}}' 2>/dev/null \
            | awk -v t="$PSQL_DOCKER_MODE" '$1==t || $2==t{found=1} END{exit !found}' || {
            echo "ERROR: 容器 '$PSQL_DOCKER_MODE' 未运行" >&2; exit 1
        }
        PSQL_DOCKER_CONTAINER="$PSQL_DOCKER_MODE"
        log "psql: docker exec -i $PSQL_DOCKER_MODE (explicit)"
        ;;
    esac
}

docker_dsn() {
    echo "$DSN" | sed -E 's#@[^/@:]+(:[0-9]+)?/#@localhost:5432/#'
}

psql_at() {
    local sql="$1"; shift
    if [[ -n "$PSQL_DOCKER_CONTAINER" ]]; then
        docker exec -i "$PSQL_DOCKER_CONTAINER" \
            psql "$(docker_dsn)" -X --no-psqlrc --set ON_ERROR_STOP=1 -At "$@" -c "$sql"
    else
        psql "$DSN" -X --no-psqlrc --set ON_ERROR_STOP=1 -At "$@" -c "$sql"
    fi
}

# ────────────────────────────────────────────────────────────────────
# Config 自发现 + DSN/API 推断（同 diag_mml_gpn_probe.sh）
# ────────────────────────────────────────────────────────────────────
discover_config() {
    if [[ -n "$CONFIG" ]]; then
        [[ -r "$CONFIG" ]] || { echo "ERROR: --config 不可读: $CONFIG" >&2; exit 1; }
        log "config: $CONFIG (--config)"; return
    fi
    if [[ -n "${OMC_CONFIG:-}" ]]; then
        CONFIG="$OMC_CONFIG"
        [[ -r "$CONFIG" ]] || { echo "ERROR: \$OMC_CONFIG 不可读: $CONFIG" >&2; exit 1; }
        log "config: $CONFIG (\$OMC_CONFIG)"; return
    fi
    local env_profile="${OMCGO_ENV:-prod}"
    local candidates=(
        "/etc/omcgo/app.${env_profile}.yaml"
        "/etc/omcgo/app.dev.yaml"
        "./omcgo/cmd/app/etc/config.local.yaml"
        "./omcgo/cmd/app/etc/config.dev.yaml"
        "./cmd/app/etc/config.local.yaml"
        "./cmd/app/etc/config.dev.yaml"
    )
    for c in "${candidates[@]}"; do
        if [[ -r "$c" ]]; then
            CONFIG="$c"
            log "config: $CONFIG (auto-discovered)"; return
        fi
    done
    echo "ERROR: 找不到 OMC App 配置文件，候选：" >&2
    printf '  - %s\n' "${candidates[@]}" >&2
    exit 1
}

yaml_get_scalar() {
    awk -v sec="$2" -v key="$3" '
        $0 ~ "^" sec ":" { flag=1; next }
        flag && /^[^ #]/ { flag=0 }
        flag && $0 ~ "^  " key ":" {
            sub("^  " key ":[ \t]*", "")
            sub(/^"/, ""); sub(/"$/, "")
            sub(/[ \t]*#.*$/, "")
            print
            exit
        }
    ' "$1"
}

discover_config

if [[ -z "$DSN" ]]; then
    DSN=$(yaml_get_scalar "$CONFIG" db dsn)
    [[ -z "$DSN" ]] && { echo "ERROR: 从 $CONFIG 读 db.dsn 失败" >&2; exit 1; }
    log "DSN: ${DSN%%@*}@*** (from $CONFIG)"
else
    log "DSN: ${DSN%%@*}@*** (--dsn)"
fi

if [[ -z "$API_URL" ]]; then
    PORT=$(yaml_get_scalar "$CONFIG" server port)
    [[ -z "$PORT" ]] && PORT=8081
    API_URL="http://127.0.0.1:$PORT"
    log "API URL: $API_URL (inferred from server.port=$PORT)"
else
    log "API URL: $API_URL (--api)"
fi

detect_psql_route

psql_at "SELECT 1" >/dev/null 2>&1 || {
    echo "ERROR: 无法连接 PG: $DSN（容器模式: $PSQL_DOCKER_CONTAINER）" >&2; exit 2
}
curl -fsS --max-time 5 "$API_URL/health" >/dev/null 2>&1 \
    || curl -fsS --max-time 5 "$API_URL/api/v1/auth/public-key" >/dev/null 2>&1 \
    || { echo "ERROR: 无法连接 API: $API_URL" >&2; exit 2; }

# ────────────────────────────────────────────────────────────────────
# API Key bootstrap（同 diag_mml_gpn_probe.sh，简化）
# ────────────────────────────────────────────────────────────────────
gen_hex32() {
    if [[ -r /dev/urandom ]]; then
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    elif command -v openssl >/dev/null; then
        openssl rand -hex 16
    else
        echo "ERROR: 无随机源" >&2; exit 3
    fi
}

bootstrap_api_key() {
    [[ -n "$API_KEY" ]] && { log "API Key: 使用提供值"; return; }

    local has_pgcrypto
    has_pgcrypto=$(psql_at "SELECT count(*) FROM pg_extension WHERE extname='pgcrypto'")
    [[ "$has_pgcrypto" == "1" ]] || {
        echo "ERROR: pgcrypto 未启用，无法自动生成 API Key" >&2
        echo "  → 由 DBA 执行 CREATE EXTENSION pgcrypto 或提供 --api-key" >&2
        exit 3
    }

    local uid
    uid=$(psql_at "SELECT id FROM users WHERE username = :'u'" -v u="$API_KEY_USER")
    [[ -n "$uid" ]] || { echo "ERROR: user '$API_KEY_USER' 未找到" >&2; exit 3; }

    API_KEY=$(gen_hex32)
    local expiry="$(date -u -d "+1 day" +"%Y-%m-%d %H:%M:%S+00" 2>/dev/null \
        || date -u -v+1d +"%Y-%m-%d %H:%M:%S+00")"
    AUTO_GENERATED_KEY_ID=$(psql_at "
        INSERT INTO api_keys (id, user_id, key_hash, description, expires_at, created_at)
        VALUES (gen_random_uuid(), :'uid'::uuid, encode(digest(:'k', 'sha256'), 'hex'),
                'auto-gen by trigger_device_config_upload', :'e'::timestamptz, NOW())
        RETURNING id
    " -v uid="$uid" -v k="$API_KEY" -v e="$expiry")
    log "API Key: 已为 '$API_KEY_USER' 生成临时 key (id=$AUTO_GENERATED_KEY_ID, 1 天有效)"
}

revoke_api_key() {
    [[ "$KEEP_API_KEY" == "true" ]] && return
    [[ -z "$AUTO_GENERATED_KEY_ID" ]] && return
    psql_at "UPDATE api_keys SET revoked_at = NOW() WHERE id = :'id'::uuid" \
        -v id="$AUTO_GENERATED_KEY_ID" >/dev/null 2>&1 || true
    log "API Key: 临时 key (id=$AUTO_GENERATED_KEY_ID) 已撤销"
}
trap revoke_api_key EXIT

bootstrap_api_key

# ────────────────────────────────────────────────────────────────────
# 步骤 1：反查 device.id UUID
# ────────────────────────────────────────────────────────────────────
DEVICE_INFO=$(psql_at "
    SELECT json_build_object(
        'id', id,
        'serial_number', serial_number,
        'carrier', carrier,
        'manufacturer', manufacturer,
        'product_class', product_class,
        'status', status,
        'firmware_version', firmware_version,
        'last_inform_at', to_char(last_inform_at, 'YYYY-MM-DD HH24:MI:SS')
    )::text
    FROM devices
    WHERE serial_number = :'sn' AND deleted_at IS NULL
")
[[ -z "$DEVICE_INFO" ]] && { echo "ERROR: 设备 SN '$DEVICE_SN' 未找到（或已软删除）" >&2; exit 2; }

DEVICE_ID=$(echo "$DEVICE_INFO" | jq -r '.id')
echo "$DEVICE_INFO" > "$OUTPUT_DIR/device.json"
log "device: id=$DEVICE_ID  status=$(echo "$DEVICE_INFO" | jq -r '.status')  last_inform_at=$(echo "$DEVICE_INFO" | jq -r '.last_inform_at')"

# ────────────────────────────────────────────────────────────────────
# 步骤 2：创建 backup task
# ────────────────────────────────────────────────────────────────────
TASK_RESP=$(curl -fsS -X POST \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    --max-time 10 \
    -d "$(jq -nc --arg id "$DEVICE_ID" '{
        task_type: "config_only",
        target_type: "device",
        target_ids: [$id]
    }')" \
    "$API_URL/api/v1/backup/tasks") || {
    echo "ERROR: POST /api/v1/backup/tasks 失败" >&2
    exit 3
}

# 兼容 response.OK 的 envelope（{success, data:{...}}）或裸 BackupTask
TASK_ID=$(echo "$TASK_RESP" | jq -r '.data.id // .id // empty')
[[ -z "$TASK_ID" ]] && { echo "ERROR: 响应无 task id: $TASK_RESP" >&2; exit 3; }
log "backup task created: $TASK_ID"

# ────────────────────────────────────────────────────────────────────
# 步骤 3：轮询直到完成 / 失败 / 超时
# ────────────────────────────────────────────────────────────────────
START=$(date +%s)
LAST_STATUS=""
while true; do
    sleep "$POLL_INTERVAL"
    elapsed=$(( $(date +%s) - START ))
    if (( elapsed > TIMEOUT_SEC )); then
        echo "ERROR: task 等 ${TIMEOUT_SEC}s 仍未完成（last_status=$LAST_STATUS）" >&2
        echo "  → docker exec docker-app-1 grep $TASK_ID /run/logs/app/app.log" >&2
        echo "  → docker exec docker-acs-1 grep $DEVICE_SN /run/logs/acs/acs.log" >&2
        echo "  → docker exec docker-postgres-1 psql -U omcgo -d omcgo -c \\" >&2
        echo "      \"SELECT status, file_path, error_message FROM backup_tasks WHERE id='$TASK_ID'\"" >&2
        exit 4
    fi

    TASK_JSON=$(curl -fsS -H "X-API-Key: $API_KEY" --max-time 5 \
        "$API_URL/api/v1/backup/tasks/$TASK_ID" 2>/dev/null) || continue

    # 兼容 envelope
    INNER=$(echo "$TASK_JSON" | jq -r '.data // .')
    STATUS=$(echo "$INNER" | jq -r '.status // empty')
    PROGRESS=$(echo "$INNER" | jq -r '.progress // 0')

    if [[ "$STATUS" != "$LAST_STATUS" ]]; then
        log "task $TASK_ID: status=$STATUS progress=$PROGRESS% (elapsed=${elapsed}s)"
        LAST_STATUS="$STATUS"
    fi

    case "$STATUS" in
    completed)
        echo "$INNER" > "$OUTPUT_DIR/task.json"
        break
        ;;
    failed)
        echo "$INNER" > "$OUTPUT_DIR/task.json"
        echo "ERROR: task=failed  error=$(echo "$INNER" | jq -r '.error_message // "(none)"')" >&2
        exit 5
        ;;
    pending|running|"")
        continue
        ;;
    *)
        log "task status 未知值: $STATUS — 继续轮询"
        ;;
    esac
done

# ────────────────────────────────────────────────────────────────────
# 步骤 4：解析 file_path
# ────────────────────────────────────────────────────────────────────
FILE_PATH=$(echo "$INNER" | jq -r '.file_path // empty')

if [[ -z "$FILE_PATH" || "$FILE_PATH" == "null" ]]; then
    # 兼容 recorder 回填稍晚的窗口：再等一轮 poll 后从 DB 直接读
    sleep "$POLL_INTERVAL"
    FILE_PATH=$(psql_at "SELECT file_path FROM backup_tasks WHERE id = :'id'::uuid" -v id="$TASK_ID")
fi

# 元数据落地
jq -nc \
    --arg device_sn "$DEVICE_SN" \
    --arg device_id "$DEVICE_ID" \
    --arg task_id "$TASK_ID" \
    --arg file_path "${FILE_PATH:-}" \
    --arg api_key_id "${AUTO_GENERATED_KEY_ID:-(provided)}" \
    --arg config_path "$CONFIG" \
    --arg api_url "$API_URL" \
    --arg started "$(date -u -d "@$START" +%FT%TZ 2>/dev/null || date -u -r "$START" +%FT%TZ)" \
    --arg completed "$(date -u +%FT%TZ)" \
    '{device_sn: $device_sn, device_id: $device_id, backup_task_id: $task_id,
      file_path: $file_path, api_key_id: $api_key_id, config_path: $config_path,
      api_url: $api_url, started_at: $started, completed_at: $completed}' \
    > "$OUTPUT_DIR/meta.json"

echo ""
echo "════════════════════════════════════════════════════════════════════"
echo "  ✅ 设备配置文件上传完成"
echo "════════════════════════════════════════════════════════════════════"
echo "  device_sn      : $DEVICE_SN"
echo "  device_id      : $DEVICE_ID"
echo "  backup_task_id : $TASK_ID"
echo "  file_path      : ${FILE_PATH:-(空 — 见下方排查)}"
echo "  task elapsed   : $(( $(date +%s) - START ))s"
echo "  输出           : $OUTPUT_DIR"
echo "════════════════════════════════════════════════════════════════════"

if [[ -z "$FILE_PATH" ]]; then
    echo "  ⚠️  task=completed 但 file_path 仍空" >&2
    echo "  排查方向：" >&2
    echo "    1. ACS upload handler 是否被命中？" >&2
    echo "       docker exec docker-acs-1 grep '$DEVICE_SN' /run/logs/acs/acs.log | grep -i upload" >&2
    echo "    2. file_path_recorder 是否监听 AutonomousTransferComplete 事件？" >&2
    echo "       docker exec docker-app-1 grep file_path_recorder /run/logs/app/app.log" >&2
    echo "    3. backup_tasks 行直接看：" >&2
    echo "       SELECT * FROM backup_tasks WHERE id='$TASK_ID';" >&2
    exit 6
fi

# ────────────────────────────────────────────────────────────────────
# 步骤 5（可选）：从 MinIO 拉取文件副本
# ────────────────────────────────────────────────────────────────────
if [[ "$DOWNLOAD" == "true" ]]; then
    log "downloading from MinIO: $FILE_PATH"
    # 优先用宿主 mc；否则走 docker-minio 容器内的 mc。
    if command -v mc >/dev/null 2>&1; then
        # 假设 mc alias 已配置；fallback 提示用户
        log "宿主 mc 可用 — 请按需 mc cp omc/$FILE_PATH $OUTPUT_DIR/"
    elif docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^docker-minio-1$'; then
        # 走 minio 容器 mc（minio 镜像默认带 mc）
        local_name="config.bin"
        case "$FILE_PATH" in
        *.tar.gz|*.tgz) local_name="config.tar.gz" ;;
        *.gz)           local_name="config.gz" ;;
        *.xml)          local_name="config.xml" ;;
        *.cfg)          local_name="config.cfg" ;;
        esac
        docker exec docker-minio-1 sh -c "mc alias set local http://localhost:9000 \
            \$MINIO_ROOT_USER \$MINIO_ROOT_PASSWORD >/dev/null 2>&1 || true; \
            mc cp local/backup-files/$FILE_PATH /tmp/$local_name" >/dev/null 2>&1 && \
        docker cp "docker-minio-1:/tmp/$local_name" "$OUTPUT_DIR/$local_name" && \
        log "下载完成: $OUTPUT_DIR/$local_name" || \
        log "⚠️  下载失败 — 请手动从 MinIO 拉取 $FILE_PATH"
    else
        log "⚠️  宿主无 mc 也无 docker-minio-1 容器 — 跳过下载"
        log "  → 手动拉取：mc cp <alias>/backup-files/$FILE_PATH ."
    fi
fi

exit 0

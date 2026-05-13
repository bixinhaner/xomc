#!/usr/bin/env bash
# diag_mml_gpn_probe.sh
#
# MML 命令路径根因诊断工具
# =======================
# 向一台在线 CPE 发送 GetParameterNames("Device.", false)，拿回真实数据模型，
# 与 mml_params 表里的 tr069_path 比对，找出"DB 配了但 CPE 实际不存在"的路径。
# 这类路径是 LST/MOD/DSP/SET 等 MML 命令被 CPE silent drop 的根本原因。
#
# 零参数运行（推荐）：自动从 OMC App 配置文件读 DSN 与 API URL，自动生成
# 临时 API Key（用完撤销）。覆盖任何一项：传 --config / --dsn / --api / --api-key。
#
# 数据通路依据：
#   - 任务创建 → POST /api/v1/devices/tasks?device_sn=<SN>（X-API-Key 鉴权）
#   - 任务结果落库 → internal/acs/handler.go:695-699 把原始 SOAP body 存进
#     device_tasks.result.raw_response（JSONB）
#   - GPN(Device., NextLevel=false) 按 TR-069 §A.3.2.4 返回 Device. 下全部
#     可访问 path 的完整集合（叶子 + 中间节点 + 多实例展开后的具体索引）
#
# 用法 / 故障排查见 docs/operations/diag-mml-gpn-probe.md。

set -euo pipefail

# ────────────────────────────────────────────────────────────────────
# 默认参数
# ────────────────────────────────────────────────────────────────────
CONFIG=""
DSN=""
API_URL=""
API_KEY="${OMCCTL_API_KEY:-}"
API_KEY_USER="admin"
DEVICE_SN="AUTO"
TIMEOUT_SEC=120
OUTPUT_DIR=""
ROOT_PATH="Device."
NEXT_LEVEL="false"
KEEP_API_KEY="false"
# T-DIAG-DOCKER：psql 路由模式。
# --psql-docker auto    自动检测（默认）：**优先 docker exec 到 postgres 容器**，
#                       找不到容器才 fallback 到宿主机 psql。
#                       2026-05-12 反转优先级（之前宿主 psql 优先，但 docker 化
#                       postgres 通常宿主访问 localhost:5432 不通，进死路）。
# --psql-docker off     禁用 docker，强制要求宿主机有 psql + 能直连 DSN
# --psql-docker <name>  显式指定容器名 / ID
PSQL_DOCKER_MODE="${PSQL_DOCKER:-auto}"
PSQL_DOCKER_CONTAINER=""   # 实际使用的容器（auto 模式下自动填充）

# 自动生成的 key id（用于退出时撤销）
AUTO_GENERATED_KEY_ID=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

向在线 CPE 发 GetParameterNames，对比 mml_params 表，输出诊断报告。

【零参数运行（推荐）】
   $(basename "$0")
   工具自动：
     1. 搜索 OMC App 配置文件 (按下列顺序)
        a. \$OMC_CONFIG 环境变量
        b. /etc/omcgo/app.\${OMCGO_ENV:-prod}.yaml （容器内）
        c. /etc/omcgo/app.dev.yaml （容器 fallback）
        d. ./omcgo/cmd/app/etc/config.local.yaml （仓库根）
        e. ./omcgo/cmd/app/etc/config.dev.yaml
        f. ./cmd/app/etc/config.dev.yaml （脚本在 omcgo/ 下）
     2. 从配置读 db.dsn 与 server.port，推断 API URL
     3. 自动为 admin 用户生成 1 天有效期的临时 API Key（退出时撤销）
     4. 跑诊断

可选覆盖参数（每项都有自动回退）：
  --config    <path>      OMC App 配置文件路径
  --dsn       <pg_dsn>    覆盖 config.db.dsn
  --api       <url>       覆盖根据 config.server.port 推断的 API URL
  --api-key   <key>       使用现成 API Key（或设环境变量 OMCCTL_API_KEY）；
                          否则自动生成临时 key
  --user      <username>  自动生成 key 时关联的用户名，默认 admin
  --keep-api-key          不撤销自动生成的临时 key（默认退出时撤销）
  --device-sn <SN>        CPE SN，默认 AUTO（自动挑一台 status='active' 且
                          last_inform_at < 10 分钟的设备）
  --timeout   <sec>       等 task 完成的超时秒数，默认 120
  --output    <dir>       输出目录，默认 /tmp/mml-diag-<timestamp>
  --root-path <path>      GPN 起始路径，默认 Device.
  --next-level            NextLevel=true（仅返回根的直接子节点）
  --psql-docker <mode>    psql 路由模式（默认 auto）：
                          auto    优先 docker exec 到 postgres 容器；找不到
                                  容器再 fallback 宿主机 psql（推荐；2026-05-12
                                  反转优先级，避免 docker 化 postgres 在宿主
                                  localhost:5432 不通走死路）
                          off     禁用 docker，强制宿主机 psql + DSN 直连
                          <name>  显式指定容器名/ID（如 docker-postgres-1）
  -h | --help             显示本帮助

输出文件（位于 \$OUTPUT_DIR）：
  report.md           人看的摘要 + 头部样本
  raw_response.xml    CPE 原始 SOAP body
  cpe_paths.txt       CPE 真实 path 列表
  db_paths.txt        DB mml_params 全部不重复 path
  missing_in_cpe.txt  ❌ DB 配了但 CPE 不存在 = MML 命令被丢的根因
  extra_in_cpe.txt    ℹ️ CPE 有但 DB 未收录 = 候选扩充
  matched.txt         ✅ 双方都有
  fixup.sql           修复草稿（注释形式，需 review）
  meta.json           运行元数据

Exit codes:
  0  成功
  1  使用错误（缺依赖 / 配置文件无法定位 / 参数非法）
  2  未找到在线设备 / 配置文件不可读
  3  任务创建失败 / API Key 生成失败
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
    --config)        CONFIG="$2"; shift 2 ;;
    --dsn)           DSN="$2"; shift 2 ;;
    --api)           API_URL="$2"; shift 2 ;;
    --api-key)       API_KEY="$2"; shift 2 ;;
    --user)          API_KEY_USER="$2"; shift 2 ;;
    --keep-api-key)  KEEP_API_KEY="true"; shift ;;
    --device-sn)     DEVICE_SN="$2"; shift 2 ;;
    --timeout)       TIMEOUT_SEC="$2"; shift 2 ;;
    --output)        OUTPUT_DIR="$2"; shift 2 ;;
    --root-path)     ROOT_PATH="$2"; shift 2 ;;
    --next-level)    NEXT_LEVEL="true"; shift ;;
    --psql-docker)   PSQL_DOCKER_MODE="$2"; shift 2 ;;
    -h|--help)       usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

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
need curl; need jq; need xmllint
# psql 走 detect_psql_route 决策（host psql 或 docker fallback）。

# ────────────────────────────────────────────────────────────────────
# 公用函数
# ────────────────────────────────────────────────────────────────────
# 用 date 子进程，避免 printf %(...)T 在 bash < 4.2 / macOS 默认 bash 3.2 上失败
log() { echo "[$(date +%H:%M:%S)] $*" >&2; }

# detect_psql_route 决定 psql_at 走宿主机 psql 还是 docker exec。
# 调用前提：PSQL_DOCKER_MODE 已由 --psql-docker / $PSQL_DOCKER 设置（默认 auto）。
detect_psql_route() {
    case "$PSQL_DOCKER_MODE" in
    off)
        # 强制宿主 psql
        if ! command -v psql >/dev/null 2>&1; then
            echo "ERROR: --psql-docker off 但宿主机无 psql" >&2
            echo "  → apt-get install postgresql-client（Debian/Ubuntu）" >&2
            echo "  → 或去掉 --psql-docker off 让脚本自动走 docker 兜底" >&2
            exit 1
        fi
        PSQL_DOCKER_CONTAINER=""
        log "psql: host (--psql-docker off)"
        return
        ;;
    auto)
        # T-DIAG-DOCKER（2026-05-12 反转优先级）：
        # 宿主机有 psql ≠ 能连得通 — docker 化的 postgres 通常不暴露 5432 给宿主
        # （或 DSN 里的 host 是容器编排别名 'postgres'，宿主 DNS 不解析）。
        # 优先用 docker exec 进 postgres 容器本身跑 psql，避开网络死结；
        # 仅当 docker 不可用或找不到 postgres 容器时才 fallback 到宿主 psql。
        local c cid
        if command -v docker >/dev/null 2>&1; then
            # 1. well-known 容器名匹配
            for c in goomc-postgres goomc_postgres_1 docker-postgres-1 deployments-postgres-1 postgres; do
                if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$c"; then
                    PSQL_DOCKER_CONTAINER="$c"
                    log "psql: docker exec -i $c (auto; named container match)"
                    return
                fi
            done
            # 2. 镜像名匹配（timescale / postgres）— 任意运行中的 PG 实例
            cid=$(docker ps --format '{{.ID}} {{.Image}}' 2>/dev/null | awk '/timescale|postgres/{print $1; exit}')
            if [[ -n "$cid" ]]; then
                PSQL_DOCKER_CONTAINER="$cid"
                log "psql: docker exec -i $cid (auto; image match)"
                return
            fi
        fi
        # 3. fallback：宿主机 psql（仅当无 docker 或无 postgres 容器时）
        if command -v psql >/dev/null 2>&1; then
            PSQL_DOCKER_CONTAINER=""
            log "psql: host (fallback; no docker postgres container found)"
            return
        fi
        # 4. 都没有
        echo "ERROR: docker 找不到 postgres 容器，宿主机也无 psql" >&2
        echo "  → 启动容器：bash run/scripts/start-deps.sh" >&2
        echo "  → 或装 psql：apt-get install postgresql-client" >&2
        echo "  → 或显式指定：--psql-docker <container-name>" >&2
        exit 1
        ;;
    *)
        # 显式容器名 / ID
        command -v docker >/dev/null 2>&1 || {
            echo "ERROR: 指定了 --psql-docker $PSQL_DOCKER_MODE 但宿主机无 docker" >&2
            exit 1
        }
        docker ps --format '{{.Names}}\t{{.ID}}' 2>/dev/null \
            | awk -v t="$PSQL_DOCKER_MODE" '$1==t || $2==t{found=1} END{exit !found}' || {
            echo "ERROR: 容器 '$PSQL_DOCKER_MODE' 未运行" >&2
            echo "  → docker ps  查看可用容器" >&2
            exit 1
        }
        PSQL_DOCKER_CONTAINER="$PSQL_DOCKER_MODE"
        log "psql: docker exec -i $PSQL_DOCKER_MODE (explicit --psql-docker)"
        return
        ;;
    esac
}

# docker_dsn 把 DSN 里的 host 改写为容器自视角的 localhost。
# 原因：YAML 配置的 host 通常是 'localhost'（宿主机视角端口映射）或
# 'postgres'（容器编排别名）；从容器内连本机 postgres 服务用 localhost
# (即 容器 内的 127.0.0.1) 最稳，因为我们的 docker exec 是进入 postgres 容器自身。
docker_dsn() {
    # 替换 @<host>(:<port>)?/ → @localhost:5432/
    echo "$DSN" | sed -E 's#@[^/@:]+(:[0-9]+)?/#@localhost:5432/#'
}

psql_at() {
    local sql="$1"; shift
    # SQL 通过 stdin 送给 psql 而不是 -c "$sql"，让 client-side 变量插值
    # (:'u' / :'name') 生效（-c 模式不做变量插值，会触发 server syntax error）。
    #
    # awk 过滤 psql 在 stdin 模式下追加的 command tag 行（INSERT N M / UPDATE N
    # / DELETE N）— -c 模式无此行但不支持变量插值；stdin 模式必出现且 -At 不
    # suppress 它，否则 INSERT...RETURNING 的结果会被 "INSERT 0 1" 污染（典型
    # 现象：bash 变量含换行 + command tag → 后续 SQL 拼接出错）。
    # awk 始终 exit 0 即使所有行被过滤，与 set -o pipefail 兼容。
    if [[ -n "$PSQL_DOCKER_CONTAINER" ]]; then
        echo "$sql" | docker exec -i "$PSQL_DOCKER_CONTAINER" \
            psql "$(docker_dsn)" -X --no-psqlrc --set ON_ERROR_STOP=1 -At "$@" \
            | awk '!/^(INSERT|UPDATE|DELETE|SELECT|COPY) [0-9]+( [0-9]+)?$/'
    else
        echo "$sql" | psql "$DSN" -X --no-psqlrc --set ON_ERROR_STOP=1 -At "$@" \
            | awk '!/^(INSERT|UPDATE|DELETE|SELECT|COPY) [0-9]+( [0-9]+)?$/'
    fi
}

# ────────────────────────────────────────────────────────────────────
# 步骤 0：自动发现 config / DSN / API URL
# ────────────────────────────────────────────────────────────────────
discover_config() {
    # 优先级 1: --config
    if [[ -n "$CONFIG" ]]; then
        [[ -r "$CONFIG" ]] || { echo "ERROR: --config 文件不可读: $CONFIG" >&2; exit 2; }
        log "config: $CONFIG (--config)"
        return
    fi
    # 优先级 2: $OMC_CONFIG
    if [[ -n "${OMC_CONFIG:-}" ]]; then
        CONFIG="$OMC_CONFIG"
        [[ -r "$CONFIG" ]] || { echo "ERROR: \$OMC_CONFIG 不可读: $CONFIG" >&2; exit 2; }
        log "config: $CONFIG (\$OMC_CONFIG)"
        return
    fi
    # 优先级 3-7: 标准位置探测
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
            log "config: $CONFIG (auto-discovered)"
            return
        fi
    done
    echo "ERROR: 找不到 OMC App 配置文件。尝试过以下位置：" >&2
    printf '  - %s\n' "${candidates[@]}" >&2
    echo "  显式指定：--config <path> 或设 \$OMC_CONFIG" >&2
    exit 2
}

# yaml_get_scalar：从 YAML 提取顶级 section 下的标量字段。
# 限制：仅支持 2 层缩进固定 2 空格（项目所有 config 均符合此格式）。
# 用法：yaml_get_scalar <file> <section> <key>
#   yaml_get_scalar config.yaml db dsn
#   yaml_get_scalar config.yaml server port
yaml_get_scalar() {
    awk -v sec="$2" -v key="$3" '
        $0 ~ "^" sec ":" { flag=1; next }
        flag && /^[^ #]/ { flag=0 }
        flag && $0 ~ "^  " key ":" {
            sub("^  " key ":[ \t]*", "")
            sub(/^"/, ""); sub(/"$/, "")
            sub(/[ \t]*#.*$/, "")    # strip trailing comment
            print
            exit
        }
    ' "$1"
}

discover_config

# DSN：优先 --dsn，否则从 config 读
if [[ -z "$DSN" ]]; then
    DSN=$(yaml_get_scalar "$CONFIG" db dsn)
    [[ -z "$DSN" ]] && { echo "ERROR: 从 $CONFIG 读 db.dsn 失败" >&2; exit 2; }
    log "DSN: ${DSN%%@*}@*** (from $CONFIG)"  # 隐藏密码后半段
else
    log "DSN: ${DSN%%@*}@*** (--dsn)"
fi

# API URL：优先 --api，否则根据 config server.port 推断
if [[ -z "$API_URL" ]]; then
    PORT=$(yaml_get_scalar "$CONFIG" server port)
    [[ -z "$PORT" ]] && PORT=8081
    API_URL="http://127.0.0.1:$PORT"
    log "API URL: $API_URL (inferred from server.port=$PORT)"
else
    log "API URL: $API_URL (--api)"
fi

# 决定 psql 路由（宿主 psql / docker exec）；必须在第一次 psql_at 之前。
detect_psql_route

# 验证 PG 与 API 可达
psql_at "SELECT 1" >/dev/null 2>&1 || {
    if [[ -n "$PSQL_DOCKER_CONTAINER" ]]; then
        echo "ERROR: 无法连接 PG（docker 路径）: 容器 $PSQL_DOCKER_CONTAINER + DSN $(docker_dsn)" >&2
        echo "  → docker exec -it $PSQL_DOCKER_CONTAINER psql -U omcgo -d omcgo -c 'SELECT 1' 排查" >&2
        echo "  → docker ps  确认 postgres 容器运行" >&2
    else
        echo "ERROR: 无法连接 PG（host psql 路径）: $DSN" >&2
        echo "  → 当前走宿主机 psql；DSN host=$(echo "$DSN" | sed -E 's#.*@([^/:]+).*#\1#') 在宿主可能不可达（docker 容器隔离 / 端口未映射）" >&2
        echo "  → 建议：启动 postgres 容器（bash run/scripts/start-deps.sh），脚本会自动切 docker exec" >&2
        echo "  → 或显式 --psql-docker <container-name>" >&2
    fi
    exit 2
}
curl -fsS --max-time 5 "$API_URL/health" >/dev/null 2>&1 || \
curl -fsS --max-time 5 "$API_URL/api/v1/auth/public-key" >/dev/null 2>&1 || {
    echo "ERROR: 无法连接 API: $API_URL" >&2
    echo "  → curl -v $API_URL/api/v1/auth/public-key 排查" >&2
    exit 2
}

# ────────────────────────────────────────────────────────────────────
# 步骤 0.5：API Key — 优先用现成，否则自动生成临时 key
# ────────────────────────────────────────────────────────────────────
ensure_pgcrypto() {
    local has
    has=$(psql_at "SELECT count(*) FROM pg_extension WHERE extname='pgcrypto'")
    [[ "$has" == "1" ]] || {
        echo "ERROR: pgcrypto 扩展未启用（自动生成 API Key 需要它）" >&2
        if [[ -n "$PSQL_DOCKER_CONTAINER" ]]; then
            echo "  → 一键启用（docker exec）：" >&2
            echo "      docker exec -i $PSQL_DOCKER_CONTAINER psql -U omcgo -d omcgo -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;'" >&2
        else
            echo "  → 一键启用（host psql）：" >&2
            echo "      psql \"\$DSN\" -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;'" >&2
        fi
        echo "  → 或拉取最新代码 + 跑迁移（migrations/000088_enable_pgcrypto.sql 已加入版本化流程）" >&2
        echo "  → 或提供 --api-key 跳过自动生成" >&2
        exit 3
    }
}

gen_hex32() {
    if [[ -r /dev/urandom ]]; then
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    elif command -v openssl >/dev/null; then
        openssl rand -hex 16
    else
        echo "ERROR: 缺少随机源（/dev/urandom 与 openssl 都不可用）" >&2
        exit 3
    fi
}

auto_generate_key() {
    ensure_pgcrypto

    local uid
    uid=$(psql_at "SELECT id FROM users WHERE username = :'u'" -v u="$API_KEY_USER")
    if [[ -z "$uid" ]]; then
        echo "ERROR: 用户不存在: $API_KEY_USER" >&2
        echo "  → psql \"\$DSN\" -c 'SELECT username FROM users LIMIT 20'" >&2
        exit 3
    fi

    local plain="omk_$(gen_hex32)"
    local prefix="${plain:0:8}"
    local key_id
    key_id=$(psql_at "
        INSERT INTO api_keys (user_id, name, key_prefix, key_hash, scopes, expires_at)
        VALUES (:'uid'::uuid, :'name', :'prefix',
                crypt(:'plain', gen_salt('bf', 10)),
                '{}', NOW() + INTERVAL '1 day')
        RETURNING id" \
        -v uid="$uid" \
        -v name="mml-diag-probe-auto-$(date +%Y%m%d-%H%M%S)" \
        -v prefix="$prefix" \
        -v plain="$plain")
    [[ -z "$key_id" ]] && { echo "ERROR: 自动生成 API Key 失败" >&2; exit 3; }

    AUTO_GENERATED_KEY_ID="$key_id"
    API_KEY="$plain"
    log "已自动生成临时 API Key (user=$API_KEY_USER, key_id=$key_id, 1 天后过期)"
}

revoke_auto_key() {
    [[ -z "$AUTO_GENERATED_KEY_ID" ]] && return
    [[ "$KEEP_API_KEY" == "true" ]] && {
        log "保留临时 API Key (--keep-api-key)：key_id=$AUTO_GENERATED_KEY_ID"
        return
    }
    psql_at "UPDATE api_keys SET revoked_at = NOW() WHERE id = :'id'::uuid" \
        -v id="$AUTO_GENERATED_KEY_ID" >/dev/null 2>&1 \
        && log "已撤销临时 API Key: $AUTO_GENERATED_KEY_ID" \
        || log "WARN: 撤销 API Key 失败: $AUTO_GENERATED_KEY_ID (手动清理：UPDATE api_keys SET revoked_at=NOW() WHERE id='$AUTO_GENERATED_KEY_ID')"
}

# 任何退出路径都尝试撤销自动生成的 key
trap revoke_auto_key EXIT

if [[ -z "$API_KEY" ]]; then
    auto_generate_key
else
    log "API Key: 使用 ${API_KEY:0:8}*** (--api-key 或 \$OMCCTL_API_KEY)"
fi

api_post() {
    # 不用 -f：HTTP 错误时仍写 response body 到 stdout/tmp，给 caller 看具体错误
    # （后端 500/400 经常带 envelope 错误信息含 biz_code/msg，吞掉 = 失明诊断）
    local path="$1" body="$2"
    local tmp; tmp=$(mktemp)
    local code
    code=$(curl -sSL --max-time 30 \
        -o "$tmp" -w "%{http_code}" \
        -H "X-API-Key: $API_KEY" \
        -H "Content-Type: application/json" \
        -X POST --data "$body" "$API_URL$path")
    if [[ "$code" =~ ^2 ]]; then
        cat "$tmp"
        rm -f "$tmp"
        return 0
    fi
    local resp; resp=$(cat "$tmp")
    rm -f "$tmp"
    echo "ERROR: HTTP $code from POST $path" >&2
    echo "  response: $resp" >&2
    return 1
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
    completed) log "task 完成"; break ;;
    failed)
        REST="${ROW#*|}"
        echo "ERROR: task 失败 — code=${REST%%|*} msg=${REST#*|}" >&2
        exit 5
        ;;
    "") echo "ERROR: task status 为空" >&2; exit 4 ;;
    *) ;;
    esac

    if (( $(date +%s) > DEADLINE )); then
        echo "ERROR: 等待超时 ${TIMEOUT_SEC}s，task 仍为 $STATUS" >&2
        echo "  → 设备可能未在线 / Connection Request 未生效 / inform 间隔太长" >&2
        echo "  → 检查 ACS 日志：grep $TASK_ID acs.log" >&2
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
# 用 xmllint local-name() 绕开命名空间差异。
log "用 xmllint 解析 ParameterInfoStruct/Name 节点…"
xmllint --xpath \
    "//*[local-name()='ParameterInfoStruct']/*[local-name()='Name']/text()" \
    "$OUTPUT_DIR/raw_response.xml" 2>/dev/null \
    | awk 'NF { gsub(/^[ \t]+|[ \t]+$/,""); print }' \
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
comm -23 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/missing_in_cpe.txt"
comm -13 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/extra_in_cpe.txt"
comm -12 "$OUTPUT_DIR/db_paths.txt"  "$OUTPUT_DIR/cpe_paths.txt" > "$OUTPUT_DIR/matched.txt"

MISS=$(wc -l <  "$OUTPUT_DIR/missing_in_cpe.txt" | tr -d ' ')
EXTRA=$(wc -l < "$OUTPUT_DIR/extra_in_cpe.txt"   | tr -d ' ')
MATCH=$(wc -l < "$OUTPUT_DIR/matched.txt"        | tr -d ' ')
log "diff: matched=$MATCH  missing=$MISS  extra=$EXTRA"

# ────────────────────────────────────────────────────────────────────
# 步骤 7：报告
# ────────────────────────────────────────────────────────────────────
{
    echo "# MML 路径诊断报告"
    echo
    echo "- 生成时间: $(date)"
    echo "- 配置文件: \`$CONFIG\`"
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
    echo "2. **path 不属于此 product** → 解除 \`mml_command_params_rel\` 绑定"
    echo "3. **CPE 数据模型未实现** → 通知设备侧补齐，DB 暂保留"
    echo
    echo "### 前 50 条样本（完整见 missing_in_cpe.txt）"
    echo
    echo '```'
    head -50 "$OUTPUT_DIR/missing_in_cpe.txt"
    if [[ "$MISS" -gt 50 ]]; then echo "... 还有 $((MISS - 50)) 条"; fi
    echo '```'
    echo
    echo "## ℹ️ CPE 有但 DB 未收录"
    echo
    echo "可作为后续 mml_params 扩充候选。前 20 条："
    echo
    echo '```'
    head -20 "$OUTPUT_DIR/extra_in_cpe.txt"
    if [[ "$EXTRA" -gt 20 ]]; then echo "... 还有 $((EXTRA - 20)) 条"; fi
    echo '```'
} > "$OUTPUT_DIR/report.md"

# ────────────────────────────────────────────────────────────────────
# 步骤 8：修复草稿 SQL
# ────────────────────────────────────────────────────────────────────
{
    echo "-- mml-diag GPN probe 修复草稿"
    echo "-- 时间: $(date)"
    echo "-- 设备: $DEVICE_SN  task_id: $TASK_ID"
    echo "--"
    echo "-- ⚠️  本文件全部为注释。每条路径都需人工 review 后取消注释才会执行。"
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
# 步骤 9：元数据 JSON
# ────────────────────────────────────────────────────────────────────
jq -n \
    --arg ts "$(date -Iseconds 2>/dev/null || date +%FT%T)" \
    --arg cfg "$CONFIG" \
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
      config: $cfg,
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

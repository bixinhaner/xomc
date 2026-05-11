#!/usr/bin/env bash
# gen_api_key.sh
#
# 直接在 DB 层为指定用户生成一个 API Key，输出明文（仅此一次）。
#
# 为什么不走 API：项目登录强制 RSA-OAEP 加密密码（参 admin/model.go LoginRequest），
# 命令行直接 curl 登录拿 JWT 不可行。本脚本绕开 RSA 链路，直接 INSERT api_keys。
#
# 兼容性：pgcrypto 的 crypt('xxx', gen_salt('bf', 10)) 产出 $2a$10$... bcrypt 哈希，
# 与 Go bcrypt.CompareHashAndPassword（golang.org/x/crypto/bcrypt）100% 兼容。
# 项目里 APIKeyService.Validate（admin/apikey_service.go:98）就是用 bcrypt 比对。
#
# 用法：见 docs/operations/diag-mml-gpn-probe.md §3 "鉴权前置"。

set -euo pipefail

DSN=""
USERNAME=""
KEY_NAME="cli-tool"
EXPIRES_DAYS=30

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

为指定用户生成一个 API Key，输出明文（仅此一次）。

必需参数:
  --dsn       <pg_dsn>    PostgreSQL 连接串
  --user      <username>  目标用户名（users.username）

可选参数:
  --name      <name>      key 名称，便于日后区分用途；默认 "cli-tool"
  --expires   <days>      过期天数，默认 30；传 0 表示永不过期
  -h | --help             显示本帮助

示例:
  $(basename "$0") --dsn "postgres://omc:omc@localhost:5432/omcgo" \\
                   --user admin \\
                   --name mml-diag-probe \\
                   --expires 7

  # 把输出存到环境变量
  export OMCCTL_API_KEY="\$( $(basename "$0") --dsn ... --user admin )"

PG 要求：
  需启用 pgcrypto 扩展。omcgo 标准部署已启用；自检：
    psql "\$DSN" -c "SELECT 1 FROM pg_extension WHERE extname='pgcrypto'"
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
    --dsn)      DSN="$2"; shift 2 ;;
    --user)     USERNAME="$2"; shift 2 ;;
    --name)     KEY_NAME="$2"; shift 2 ;;
    --expires)  EXPIRES_DAYS="$2"; shift 2 ;;
    -h|--help)  usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

[[ -z "$DSN" ]] && { echo "ERROR: --dsn required" >&2; exit 1; }
[[ -z "$USERNAME" ]] && { echo "ERROR: --user required" >&2; exit 1; }

command -v psql >/dev/null 2>&1 || { echo "ERROR: psql not found" >&2; exit 1; }

# ────────────────────────────────────────────────────────────────────
# 预检 pgcrypto
# ────────────────────────────────────────────────────────────────────
HAS_PGCRYPTO=$(psql "$DSN" -X -At --set ON_ERROR_STOP=1 \
    -c "SELECT count(*) FROM pg_extension WHERE extname='pgcrypto'")
if [[ "$HAS_PGCRYPTO" != "1" ]]; then
    echo "ERROR: pgcrypto extension not installed; run as superuser:" >&2
    echo "  CREATE EXTENSION IF NOT EXISTS pgcrypto;" >&2
    exit 2
fi

# ────────────────────────────────────────────────────────────────────
# 查用户
# ────────────────────────────────────────────────────────────────────
USER_ID=$(psql "$DSN" -X -At --set ON_ERROR_STOP=1 \
    -c "SELECT id FROM users WHERE username = :'u'" -v u="$USERNAME")
if [[ -z "$USER_ID" ]]; then
    echo "ERROR: user not found: $USERNAME" >&2
    echo "  → psql \"\$DSN\" -c 'SELECT username FROM users LIMIT 20' 查看可用账号" >&2
    exit 2
fi

# ────────────────────────────────────────────────────────────────────
# 生成明文 key：omk_ + 32 hex chars（与 apikey_service.go 格式严格一致）
# 优先 /dev/urandom，回退 openssl
# ────────────────────────────────────────────────────────────────────
gen_hex32() {
    if [[ -r /dev/urandom ]]; then
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    else
        command -v openssl >/dev/null && openssl rand -hex 16 || {
            echo "ERROR: neither /dev/urandom nor openssl available" >&2
            exit 1
        }
    fi
}
PLAIN_KEY="omk_$(gen_hex32)"
KEY_PREFIX="${PLAIN_KEY:0:8}"   # "omk_" + 前 4 hex，与 apikey_service.go:46 一致

# ────────────────────────────────────────────────────────────────────
# bcrypt 哈希 + INSERT
# ────────────────────────────────────────────────────────────────────
# crypt() 用 gen_salt('bf', 10) 产出 $2a$10$... 格式，cost=10 与 Go bcrypt.DefaultCost 一致
EXPIRES_CLAUSE="NULL"
if [[ "$EXPIRES_DAYS" -gt 0 ]]; then
    EXPIRES_CLAUSE="NOW() + INTERVAL '$EXPIRES_DAYS days'"
fi

# 用 stdin 喂 SQL，避免 plain_key 拼到命令行被进程列表泄露
KEY_ID=$(psql "$DSN" -X -At --set ON_ERROR_STOP=1 \
    -v uid="$USER_ID" \
    -v name="$KEY_NAME" \
    -v prefix="$KEY_PREFIX" \
    -v plain="$PLAIN_KEY" \
    <<SQL
INSERT INTO api_keys (user_id, name, key_prefix, key_hash, scopes, expires_at)
VALUES (
    :'uid'::uuid,
    :'name',
    :'prefix',
    crypt(:'plain', gen_salt('bf', 10)),
    '{}',
    $EXPIRES_CLAUSE
)
RETURNING id;
SQL
)

if [[ -z "$KEY_ID" ]]; then
    echo "ERROR: INSERT failed (see psql output above)" >&2
    exit 1
fi

# ────────────────────────────────────────────────────────────────────
# 输出
# ────────────────────────────────────────────────────────────────────
{
    echo "API Key 已生成"
    echo "  user:        $USERNAME ($USER_ID)"
    echo "  key_id:      $KEY_ID"
    echo "  name:        $KEY_NAME"
    if [[ "$EXPIRES_DAYS" -gt 0 ]]; then
        echo "  expires_at:  $(date -d "+$EXPIRES_DAYS days" -Iseconds 2>/dev/null || \
                              date -v"+${EXPIRES_DAYS}d" -Iseconds 2>/dev/null || \
                              echo "$EXPIRES_DAYS days from now")"
    else
        echo "  expires_at:  never"
    fi
    echo
    echo "⚠️  下面这行是明文 key，仅此一次显示。请立刻保存到安全位置："
    echo
} >&2

# 明文 key 单独输出到 stdout，方便 export OMCCTL_API_KEY=$(./gen_api_key.sh ...)
echo "$PLAIN_KEY"

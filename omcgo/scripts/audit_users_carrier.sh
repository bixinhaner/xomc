#!/usr/bin/env bash
#
# audit_users_carrier.sh — v1.0 部署前数据 audit：在 DROP COLUMN users.carrier 之前
# 列出所有需要业务确认的高风险用户，避免误降级真正的超管账号。
#
# 决议背景：docs/prd/system/users.md §11.11 (v1.0)
#   - 旧规则：超管 = users.carrier IS NULL
#   - 新规则：超管 = users.source = 'builtIn'
#   - 字段 source 创建时即固定，**不可运行期改写**
#   - 因此现存 carrier IS NULL 但 source != 'builtIn' 的"假超管"，在 v1.0 后会
#     失去超管旁路（变成普通 admin 用户），看不见所有设备分组
#
# 用法：
#   bash omcgo/scripts/audit_users_carrier.sh                # docker 模式（默认）
#   bash omcgo/scripts/audit_users_carrier.sh --mode host    # 直连 PG
#   bash omcgo/scripts/audit_users_carrier.sh --csv > out.csv  # CSV 输出
#
# 退出码：
#   0 = 无 R1/R2 风险，可以放心执行 migrate-up（000055_drop_users_carrier.sql）
#   1 = 发现 R2 "假超管"或 R1 异常内置用户，**业务必须逐个确认后**才能 migrate
#   2 = 参数错或工具/连接不可用

set -uo pipefail

MODE=docker
OUTPUT=md

if REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null); then :; else
    REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fi
COMPOSE_FILE="$REPO_ROOT/deployments/docker/docker-compose.yml"
POSTGRES_SVC=postgres
PG_USER=${PGUSER:-omcgo}
PG_PASS=${PGPASSWORD:-omcgo123}
PG_DB=${PGDATABASE:-omcgo}
PG_HOST=${PGHOST:-localhost}
PG_PORT=${PGPORT:-5432}

while [[ $# -gt 0 ]]; do
    case $1 in
        --mode) MODE=$2; shift 2 ;;
        --compose-file) COMPOSE_FILE=$2; shift 2 ;;
        --postgres-svc) POSTGRES_SVC=$2; shift 2 ;;
        --pg-host) PG_HOST=$2; shift 2 ;;
        --pg-port) PG_PORT=$2; shift 2 ;;
        --pg-user) PG_USER=$2; shift 2 ;;
        --pg-pass) PG_PASS=$2; shift 2 ;;
        --pg-db) PG_DB=$2; shift 2 ;;
        --csv) OUTPUT=csv; shift ;;
        --md) OUTPUT=md; shift ;;
        -h|--help) sed -n '2,21p' "$0"; exit 0 ;;
        *) echo "Unknown arg: $1" >&2; exit 2 ;;
    esac
done

psql_q() {
    case "$MODE" in
        host)   PGPASSWORD="$PG_PASS" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" -t -A -F$'\t' "$@" ;;
        docker) docker compose -f "$COMPOSE_FILE" exec -T -e PGPASSWORD="$PG_PASS" \
                    "$POSTGRES_SVC" psql -U "$PG_USER" -d "$PG_DB" -t -A -F$'\t' "$@" ;;
        *) echo "Unknown mode: $MODE" >&2; exit 2 ;;
    esac
}

# 防御：如果 carrier 列已经被删（migrate 已执行过），直接报告"已迁移"并退出 0。
COL_EXISTS=$(psql_q -c "SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='carrier' LIMIT 1;" 2>/dev/null || true)
if [[ -z "$COL_EXISTS" ]]; then
    if [[ "$OUTPUT" == csv ]]; then
        echo "status,message"
        echo "already_migrated,users.carrier 已被删除（迁移 000055 已执行），无需再 audit"
    else
        echo "✓ users.carrier 列已不存在 — v1.0 迁移 000055_drop_users_carrier.sql 已执行过"
        echo "  本脚本无需再运行。"
    fi
    exit 0
fi

# ==================================================
# T1：总览统计 — source × carrier 交叉表
# ==================================================
SQL_OVERVIEW=$(cat <<'EOF'
SELECT
    source,
    COALESCE(carrier, '<NULL>') AS carrier,
    COUNT(*) AS cnt
FROM users
GROUP BY source, carrier
ORDER BY source, carrier;
EOF
)

# ==================================================
# T2：R1 异常 — 内置用户绑了 carrier（预期不出现，出现需业务确认是否是历史脏数据）
# ==================================================
SQL_R1=$(cat <<'EOF'
SELECT id, username, source, carrier, status,
       COALESCE(last_login_at::text, '<never>') AS last_login_at,
       created_at::date AS created
FROM users
WHERE source = 'builtIn' AND carrier IS NOT NULL
ORDER BY created_at;
EOF
)

# ==================================================
# T3：R2 核心风险 — 非内置但 carrier IS NULL（"假超管"）
# 这些用户在 v1.0 后将失去超管旁路。业务必须逐个确认：
#   (a) 接受降级（保持 source='admin'，按角色 + role_device_groups 走数据权限）
#   (b) 删除该用户 + 在 seed 加新 builtIn 账号（部署级运维，本脚本不处理）
# ==================================================
SQL_R2=$(cat <<'EOF'
SELECT u.id, u.username, u.source, u.status,
       COALESCE(u.last_login_at::text, '<never>') AS last_login_at,
       u.created_at::date AS created,
       COALESCE(string_agg(r.name, ',' ORDER BY r.name), '<no_role>') AS roles
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
WHERE u.carrier IS NULL AND u.source <> 'builtIn'
GROUP BY u.id, u.username, u.source, u.status, u.last_login_at, u.created_at
ORDER BY u.created_at;
EOF
)

# ==================================================
# T4：LDAP 用户的 carrier 分布（信息性，不阻塞）
# 旧 sync_carrier 默认 NULL，预期 LDAP 用户 carrier 全 NULL；
# 如有非 NULL 说明历史 sync_carrier 配置过具体值，迁移后该值会丢失。
# ==================================================
SQL_LDAP=$(cat <<'EOF'
SELECT COALESCE(carrier, '<NULL>') AS carrier, COUNT(*) AS cnt
FROM users
WHERE source = 'LDAP'
GROUP BY carrier
ORDER BY carrier;
EOF
)

T1=$(psql_q -c "$SQL_OVERVIEW" 2>&1)
if [[ "$T1" == *"ERROR"* || "$T1" == *"could not"* ]]; then
    echo "❌ 数据库查询失败：" >&2
    echo "$T1" >&2
    exit 2
fi

T2_R1=$(psql_q -c "$SQL_R1")
T3_R2=$(psql_q -c "$SQL_R2")
T4_LDAP=$(psql_q -c "$SQL_LDAP")

# 计数
cnt_r1=$([[ -z "$T2_R1" ]] && echo 0 || echo "$T2_R1" | grep -c '^')
cnt_r2=$([[ -z "$T3_R2" ]] && echo 0 || echo "$T3_R2" | grep -c '^')

# CSV 输出
if [[ "$OUTPUT" == csv ]]; then
    echo "section,col1,col2,col3,col4,col5,col6,col7"
    echo "$T1"   | awk -F'\t' '{printf "T1_overview,%s,%s,%s,,,,\n",$1,$2,$3}'
    echo "$T2_R1"| awk -F'\t' '$1!=""{printf "T2_R1_anomalous_builtin,%s,%s,%s,%s,%s,%s,%s\n",$1,$2,$3,$4,$5,$6,$7}'
    echo "$T3_R2"| awk -F'\t' '$1!=""{printf "T3_R2_pseudo_super,%s,%s,%s,%s,%s,%s,%s\n",$1,$2,$3,$4,$5,$6,$7}'
    echo "$T4_LDAP"| awk -F'\t' '$1!=""{printf "T4_ldap_carrier,%s,%s,,,,,\n",$1,$2}'
    [[ $cnt_r1 -gt 0 || $cnt_r2 -gt 0 ]] && exit 1 || exit 0
fi

# Markdown 输出
echo
echo "===== users.carrier 删除前 audit (v1.0 §11.11) ====="
echo "  数据库: $PG_DB ($MODE 模式)"
echo "  时间  : $(date '+%Y-%m-%d %H:%M:%S')"
echo

echo "## T1 总览：source × carrier 交叉分布"
printf "  %-10s | %-12s | %s\n" "source" "carrier" "count"
printf -- "  -----------+--------------+------\n"
echo "$T1" | awk -F'\t' '$1!=""{printf "  %-10s | %-12s | %s\n", $1, $2, $3}'
echo

echo "## T2 R1：内置用户绑定了 carrier（预期 0，>0 需业务确认）"
if [[ $cnt_r1 -eq 0 ]]; then
    echo "  ✓ 0 行（正常）"
else
    printf "  %-36s | %-16s | %-8s | %-12s | %-12s | %-10s | %s\n" "id" "username" "source" "carrier" "status" "last_login" "created"
    printf -- "  -------------------------------------+------------------+----------+--------------+--------------+------------+----------\n"
    echo "$T2_R1" | awk -F'\t' '$1!=""{printf "  %-36s | %-16s | %-8s | %-12s | %-12s | %-10s | %s\n", $1, $2, $3, $4, $5, $6, $7}'
fi
echo

echo "## T3 R2：carrier IS NULL 但 source != 'builtIn' 的\"假超管\""
echo "         **删 carrier 后这些用户将失去超管旁路，变为普通用户**"
if [[ $cnt_r2 -eq 0 ]]; then
    echo "  ✓ 0 行（无风险）"
else
    printf "  %-36s | %-16s | %-8s | %-12s | %-10s | %-10s | %s\n" "id" "username" "source" "status" "last_login" "created" "roles"
    printf -- "  -------------------------------------+------------------+----------+--------------+------------+------------+----------\n"
    echo "$T3_R2" | awk -F'\t' '$1!=""{printf "  %-36s | %-16s | %-8s | %-12s | %-10s | %-10s | %s\n", $1, $2, $3, $4, $5, $6, $7}'
fi
echo

echo "## T4 LDAP 用户的 carrier 分布（信息性）"
if [[ -z "$T4_LDAP" ]]; then
    echo "  · 无 LDAP 用户"
else
    printf "  %-12s | %s\n" "carrier" "count"
    printf -- "  -------------+------\n"
    echo "$T4_LDAP" | awk -F'\t' '$1!=""{printf "  %-12s | %s\n", $1, $2}'
fi
echo

# 汇总建议
echo "===== 建议 ====="
if [[ $cnt_r1 -eq 0 && $cnt_r2 -eq 0 ]]; then
    echo "✓ 无风险，可以执行：make migrate-up（触发 000055_drop_users_carrier.sql）"
    exit 0
fi

if [[ $cnt_r1 -gt 0 ]]; then
    echo "⚠ R1: 内置用户 (source='builtIn') 不应携带 carrier 值，存在 $cnt_r1 行异常。"
    echo "  · 极可能是 v0.2 之前的脏数据；carrier 列被删后该值自然消失，不影响超管身份。"
    echo "  · 业务确认无误后此条非阻塞。"
fi

if [[ $cnt_r2 -gt 0 ]]; then
    echo "⚠ R2: 发现 $cnt_r2 个 \"假超管\"（carrier IS NULL 但 source != 'builtIn'）。"
    echo "  · v1.0 部署后这些用户**自动降级为普通用户**，可见域 = 角色绑定的 device_groups 并集。"
    echo "  · 业务必须逐个确认："
    echo "      (a) 该用户实际不是超管，仅 carrier 字段误配 NULL → 接受降级"
    echo "      (b) 该用户是真实超管，需要保留 → 部署级运维处理（删除 + 在 seed 加新 builtIn 账号）"
    echo "  · 确认完成后再次运行本脚本，期望 R2 行数变为 0 或剩余行数被业务签字接受。"
    echo
    echo "  阻塞 migrate：请在运维 Runbook 上记录本审计结果与处置决议后再执行 000055 迁移。"
fi

# 退出码：有 R1 或 R2 时返回 1（CI gate 用）
exit 1

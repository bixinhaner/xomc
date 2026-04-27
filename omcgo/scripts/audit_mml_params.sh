#!/usr/bin/env bash
#
# audit_mml_params.sh — 审计 mml_params 表里所有 tr069_path 的 TR-069 协议合规性。
#
# 检测三类问题（与 internal/mml/tr069_payload.go validatePath 完全对齐）：
#   1. bad_prefix      顶层非 Device. / InternetGatewayDevice.
#   2. placeholder     含 {i}/{n}/{idx} 占位符未替换
#   3. bad_chars       含非 ASCII 字母数字 / . [] {} _ - 之外的字符
#
# 用法：
#   bash omcgo/scripts/audit_mml_params.sh                # docker 模式（默认）
#   bash omcgo/scripts/audit_mml_params.sh --mode host    # 直连
#   bash omcgo/scripts/audit_mml_params.sh --csv > out    # CSV 输出
#
# 退出码：0=全部合规  1=发现非法路径  2=参数错/工具不可用

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
        -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
        *) echo "Unknown arg: $1" >&2; exit 2 ;;
    esac
done

psql_q() {
    case "$MODE" in
        host)   PGPASSWORD=$PG_PASS psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -t -A "$@" ;;
        docker) docker compose -f "$COMPOSE_FILE" exec -T -e PGPASSWORD="$PG_PASS" \
                    "$POSTGRES_SVC" psql -U "$PG_USER" -d "$PG_DB" -t -A "$@" ;;
    esac
}

# 同步 validatePath 的判定规则到 SQL：
#   - bad_prefix:  NOT (path LIKE 'Device.%' OR path LIKE 'InternetGatewayDevice.%')
#   - placeholder: path 含 {i}/{n}/{idx}
#   - bad_chars:   path 含非 [A-Za-z0-9_.\[\]\-] 字符（用反向 regex）
SQL=$(cat <<'EOF'
WITH classified AS (
    SELECT id, param_code, tr069_path, param_version, value_type,
           CASE
               WHEN tr069_path !~ '^(Device|InternetGatewayDevice)\.'                   THEN 'bad_prefix'
               WHEN tr069_path ~  '\{[A-Za-z]+\}'                                       THEN 'placeholder'
               WHEN tr069_path !~ '^[A-Za-z0-9_.\[\]\-]+$'                              THEN 'bad_chars'
               ELSE 'ok'
           END AS reason
    FROM mml_params
)
SELECT reason, param_version, tr069_path, COUNT(*) AS rows
FROM classified
WHERE reason != 'ok'
GROUP BY reason, param_version, tr069_path
ORDER BY reason, rows DESC;
EOF
)

result=$(psql_q -F$'\t' -c "$SQL" 2>&1)
if [[ -z "$result" ]]; then
    if [[ "$OUTPUT" == csv ]]; then
        echo "reason,param_version,tr069_path,rows"
    else
        echo "✓ mml_params 全部 tr069_path 合规（无 bad_prefix / placeholder / bad_chars）"
    fi
    exit 0
fi

if [[ "$OUTPUT" == csv ]]; then
    echo "reason,param_version,tr069_path,rows"
    echo "$result" | awk -F'\t' '{printf "%s,%s,%s,%s\n", $1, $2, $3, $4}'
    exit 1
fi

# Markdown 输出 + 统计 summary（兼容 macOS 默认 bash 3.x，不用关联数组）
sum_rows() {
    # $1 = reason 名；从 result 里抓该 reason 的 rows 列累加
    echo "$result" | awk -F'\t' -v r="$1" '$1==r {sum += $4} END {print sum+0}'
}
cnt_bad_prefix=$(sum_rows bad_prefix)
cnt_placeholder=$(sum_rows placeholder)
cnt_bad_chars=$(sum_rows bad_chars)
total_bad=$((cnt_bad_prefix + cnt_placeholder + cnt_bad_chars))

echo
echo "===== mml_params TR-069 合规性审计 ====="
echo "  数据库: $PG_DB ($MODE 模式)"
echo "  时间   : $(date '+%Y-%m-%d %H:%M:%S')"
echo
echo "汇总:"
printf "  %-12s %d 条\n" "bad_prefix"  "$cnt_bad_prefix"
printf "  %-12s %d 条\n" "placeholder" "$cnt_placeholder"
printf "  %-12s %d 条\n" "bad_chars"   "$cnt_bad_chars"
echo "  --------------------"
printf "  %-12s %d 条\n" "TOTAL_BAD" "$total_bad"
echo
echo "明细 (前 50 条):"
printf "  %-12s | %-10s | %s\n" "reason" "version" "tr069_path"
printf -- "  ------------------------------------------------\n"
echo "$result" | head -50 | awk -F'\t' '{printf "  %-12s | %-10s | %s\n", $1, $2, $3}'
echo
if (( total_bad > 0 )); then
    echo "→ 影响与修复："
    echo "  · 协议层防御已就位 — internal/mml/tr069_payload.go validatePath 会"
    echo "    在 fanout 时自动跳过这些不合规路径，**不会进入下发给 CPE 的 SOAP 报文**。"
    echo "  · bad_prefix 类（如 boardconf.*）通常是 OMC 历史内部配置项，本就不"
    echo "    通过 TR-069 下发；如果业务上确实需要走 TR-069，请联系厂商索取真实路径。"
    echo "  · placeholder / bad_chars 已通过 seed/000029 自动修复（{i}/{j}/... → 1）。"
    echo "  · 仍想批量修 bad_prefix：写新 migration 参考 seed/000029_fix_mml_params_invalid_paths.sql"
    echo "  · 排查具体一次下发：omcgo/scripts/diag_mml_task.sh --auto --diagnose-rpc"
    echo
fi
exit 1

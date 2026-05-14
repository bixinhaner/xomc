#!/usr/bin/env bash
#
# diag_mml_orphan_codes.sh — 反查 mml_custom_command 中 command_code
# 在新 mml_commands 表里**已不存在**的"孤儿"记录。
#
# 背景（Sprint B / T-0119）：
#   standard-model.xml 重建后，老 mml_custom_command 用户保存的"自定义命令"
#   可能引用了已下线的 command_code（如 "MOD_X_OLD_VENDOR" 在新 standard
#   model 里没有对应项）。后端 ExecuteCommand 已加 orphan=true fallback
#   不再 500，但用户实际"执行"时任务会创建但 0 设备派发，体验不好。
#
# 此脚本：
#   1. 反查所有 mml_custom_command.command_code 不在 mml_commands.command_code
#      表里的记录
#   2. 输出 markdown 报表（id / scope / creator / 命令名 / 老 code）
#   3. 给出 3 种处置建议：
#      a. 用户自助删除（DELETE FROM mml_custom_command WHERE id=...）
#      b. 手工映射到新 code（UPDATE mml_custom_command SET command_code=... WHERE id=...）
#      c. 整批迁移到"已下线"分类（标 deprecated tag）便于 UI 隐藏
#
# 默认 mode=docker：所有 PG 访问走 docker compose exec
#   宿主只需要 docker；不需要装 postgresql-client。
#
# 用法：
#   bash omcgo/scripts/diag_mml_orphan_codes.sh                # 仅报告
#   bash omcgo/scripts/diag_mml_orphan_codes.sh --output /tmp/orphan-report.md
#   bash omcgo/scripts/diag_mml_orphan_codes.sh --delete-all   # 危险：直接删
#   bash omcgo/scripts/diag_mml_orphan_codes.sh --dry-run-delete  # 仅打 DELETE SQL 不执行
#
# 退出码：
#   0  无孤儿，或 dry-run 成功
#   1  发现孤儿（仅报告模式）
#   2  实际删除失败
#   3  参数错误 / 环境缺失
set -euo pipefail

# ---------- Defaults ----------
MODE="docker"
DOCKER_PG_SVC="${DOCKER_PG_SVC:-docker-postgres-1}"
PG_USER="${PG_USER:-omcgo}"
PG_DB="${PG_DB:-omcgo}"
OUTPUT=""
ACTION="report"   # report | delete-all | dry-run-delete

# ---------- Args ----------
while [ $# -gt 0 ]; do
    case "$1" in
        --output)         OUTPUT="$2"; shift 2;;
        --delete-all)     ACTION="delete-all"; shift;;
        --dry-run-delete) ACTION="dry-run-delete"; shift;;
        --host-pg)        MODE="host"; shift;;
        -h|--help)
            head -45 "$0" | tail -42
            exit 0;;
        *)
            echo "ERROR: unknown arg: $1" >&2
            echo "Use --help for usage." >&2
            exit 3;;
    esac
done

# ---------- Helper: psql ----------
psql_query() {
    local sql="$1"
    if [ "$MODE" = "docker" ]; then
        docker exec -i "$DOCKER_PG_SVC" psql -U "$PG_USER" -d "$PG_DB" -A -F'|' -t -c "$sql"
    else
        PGPASSWORD="${PG_PASSWORD:-omcgo123}" psql -h localhost -U "$PG_USER" -d "$PG_DB" -A -F'|' -t -c "$sql"
    fi
}

# ---------- Step 1: Count orphans ----------
echo "==> Querying orphan mml_custom_command rows..." >&2

ORPHAN_QUERY="
SELECT cc.id,
       cc.command_scope,
       cc.creator,
       cc.command_name,
       cc.command_code,
       cc.created_at,
       cc.operation_type
FROM mml_custom_command cc
LEFT JOIN mml_commands mc ON mc.command_code = cc.command_code
WHERE mc.id IS NULL
ORDER BY cc.created_at DESC;
"

ROWS=$(psql_query "$ORPHAN_QUERY" || true)
COUNT=$(echo "$ROWS" | grep -c '|' 2>/dev/null || echo 0)
COUNT=${COUNT##* }  # strip leading spaces

echo "==> Found $COUNT orphan record(s)" >&2

# ---------- Step 2: Generate markdown report ----------
REPORT_FILE="${OUTPUT:-/tmp/mml-orphan-report-$(date +%Y%m%d-%H%M%S).md}"

{
    echo "# mml_custom_command 孤儿码反查报告"
    echo ""
    echo "**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "**孤儿记录数**: $COUNT"
    echo "**数据源**: $DOCKER_PG_SVC ($PG_DB)"
    echo ""
    echo "## 背景"
    echo ""
    echo "T-0119 standard-model 重建后 \`mml_commands\` 表全量重建（~840 行）。"
    echo "下面这些 \`mml_custom_command\` 行的 \`command_code\` 在新表中已不存在，"
    echo "用户点击执行时会被后端降级为 \`orphan=true\` 透传，任务创建成功但 0"
    echo "设备派发，体验不佳。"
    echo ""

    if [ "$COUNT" -eq 0 ]; then
        echo "## 结果：✅ 无孤儿记录"
        echo ""
        echo "所有 \`mml_custom_command.command_code\` 在新 \`mml_commands\` 表中均能找到。"
    else
        echo "## 孤儿记录列表"
        echo ""
        echo "| ID | scope | creator | 命令名 | 孤儿 code | op | 创建时间 |"
        echo "|----|-------|---------|--------|-----------|----|---------:|"
        echo "$ROWS" | while IFS='|' read -r id scope creator name code created_at op; do
            [ -z "$id" ] && continue
            # Truncate id to first 8 chars for readability
            short_id="${id:0:8}…"
            echo "| \`$short_id\` | $scope | $creator | $name | \`$code\` | $op | $created_at |"
        done

        echo ""
        echo "## 处置建议（3 选 1）"
        echo ""
        echo "### A. 自助删除（用户手动确认每条）"
        echo ""
        echo '```bash'
        echo "$ROWS" | while IFS='|' read -r id scope creator name code created_at op; do
            [ -z "$id" ] && continue
            echo "# DELETE: $name ($code, by $creator)"
            echo "DELETE FROM mml_custom_command WHERE id = '$id';"
        done
        echo '```'

        echo ""
        echo "### B. 手工映射到新 code（推荐 — 保留用户配置）"
        echo ""
        echo "用 \`grep\` 在 \`mml_commands.command_code\` 列里找最接近的新码，然后："
        echo ""
        echo '```sql'
        echo "-- 示例：找到 NEW_CODE_HERE 后"
        echo "UPDATE mml_custom_command SET command_code = 'NEW_CODE_HERE' WHERE id = '<old_id>';"
        echo '```'

        echo ""
        echo "### C. 批量删除（脚本一键 — 危险，仅在确认无价值时）"
        echo ""
        echo '```bash'
        echo "bash $(basename "$0") --delete-all"
        echo '```'

        echo ""
        echo "## 查询新表里的近似 code（辅助 B 方案）"
        echo ""
        echo '```sql'
        echo "-- 在 PG 里跑，每个孤儿码做模糊匹配"
        echo "$ROWS" | while IFS='|' read -r id scope creator name code created_at op; do
            [ -z "$id" ] && continue
            # Extract the "stem" — drop LST_/MOD_/ADD_/RMV_ prefix
            stem=$(echo "$code" | sed -E 's/^(LST|MOD|ADD|RMV|DSP|ACT|DEA|RST|CLR|UPG)_//')
            echo "-- orphan: $code → search stem '$stem'"
            echo "SELECT command_code FROM mml_commands WHERE command_code ILIKE '%${stem}%' LIMIT 5;"
        done
        echo '```'
    fi
} > "$REPORT_FILE"

echo "==> Report written: $REPORT_FILE" >&2

# ---------- Step 3: Action handling ----------
case "$ACTION" in
    report)
        if [ "$COUNT" -gt 0 ]; then
            echo "==> Exit 1 (orphans found; run with --dry-run-delete or --delete-all to act)" >&2
            cat "$REPORT_FILE"
            exit 1
        fi
        cat "$REPORT_FILE"
        exit 0
        ;;

    dry-run-delete)
        if [ "$COUNT" -eq 0 ]; then
            echo "==> Nothing to delete (no orphans)" >&2
            exit 0
        fi
        echo "==> DRY-RUN: SQL that *would* run if --delete-all:"
        echo "$ROWS" | while IFS='|' read -r id scope creator name code created_at op; do
            [ -z "$id" ] && continue
            echo "DELETE FROM mml_custom_command WHERE id = '$id'; -- $name ($code)"
        done
        echo ""
        echo "(no rows actually deleted)" >&2
        exit 0
        ;;

    delete-all)
        if [ "$COUNT" -eq 0 ]; then
            echo "==> Nothing to delete (no orphans)" >&2
            exit 0
        fi
        echo "==> WARNING: deleting $COUNT orphan record(s) in 5s..." >&2
        echo "    Press Ctrl-C to abort." >&2
        sleep 5
        # Single transaction
        DEL_SQL="BEGIN;"
        IFS=$'\n'
        for row in $ROWS; do
            [ -z "$row" ] && continue
            id=$(echo "$row" | cut -d'|' -f1)
            DEL_SQL="$DEL_SQL DELETE FROM mml_custom_command WHERE id = '$id';"
        done
        DEL_SQL="$DEL_SQL COMMIT;"
        if psql_query "$DEL_SQL" >/dev/null; then
            echo "==> Successfully deleted $COUNT orphan record(s)" >&2
            exit 0
        else
            echo "==> ERROR: deletion failed (rolled back)" >&2
            exit 2
        fi
        ;;
esac

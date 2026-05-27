#!/usr/bin/env bash
# check-schema-drift.sh — 校验 goose 元数据与磁盘 / DB 真实状态一致。
#
# 解决的问题（2026-05-27 事故复盘）：
#   goose 启动后会以 goose_db_version 表的 version_id 为权威，**只要某号 applied=true
#   就跳过对应文件**。但 goose 没有内置 file-content checksum，下列情形都能让
#   "已 applied"记录与真实 DB schema 偏移：
#     · 别人提交一个 000NNN_*.sql，跑 goose 应用，后续 commit 重写了文件内容（同号、
#       不同 SQL），下次 goose 不会重新跑 → DB 缺最新 schema。
#     · 误用 `--path` 指向错误目录、或本地 dev 跑了未提交的 migration、留下 fake
#       applied 记录后把文件删了 / 改了。
#     · 直接 `DELETE FROM goose_db_version` 或 `DROP TABLE` 等手工干预后没回写。
#
#   2026-05-26 mr_customize_task 事故就是第二种：21:02:49 goose 跑 195/196 留下记录，
#   21:22:07 commit 才提交真正的 195 (mr CREATE TABLE) → 表从来没建出来，DB 跑到 197
#   ALTER 时报 SQLSTATE 42P01。
#
# 检测策略（不依赖 checksum，从静态文件 + DB 查询能做的几个 lightweight check）：
#   A. 文件 orphan：goose_db_version 里的 version_id 在 migrations/ 里无对应文件 →
#      drift 严重，提示手工核对（可能是文件被删了，或 goose 跑了错路径）
#   B. 未 applied 高号：磁盘上有 000NNN_*.sql 但 goose 没记录 applied=true 且 NNN < 最大
#      已 applied 号 → "中间有跳号没跑"，下次 goose 启动会回头跑这条，可能引入不可预期变更
#   C. 任意 CREATE TABLE 在最后 N 条已 applied migration 中声明，但表不存在 →
#      最强信号（这就是 195/196 那次事故的特征）。只检查最近 20 条以控制成本。
#
# 用法：
#   ./check-schema-drift.sh                          # 默认连本地 postgres
#   DSN=postgres://... ./check-schema-drift.sh       # 自定义连接
#   ./check-schema-drift.sh --table goose_db_version --table goose_db_version_seed
#   ./check-schema-drift.sh --path /custom/migrations --table goose_db_version
#
# 退出码：
#   0 — 无 drift
#   1 — 发现 drift（CI 阻塞 / release-gate 拒绝）

set -euo pipefail

DSN="${DSN:-postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEFAULT_MIG_DIR="$SCRIPT_DIR/../migrations"

# 解析 --path / --table 多次出现支持"主目录 + seed"两套对照
PATHS=()
TABLES=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --path)   PATHS+=("$2"); shift 2 ;;
    --table)  TABLES+=("$2"); shift 2 ;;
    --dsn)    DSN="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,40p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ ${#PATHS[@]} -eq 0 ]]; then
  PATHS=("$DEFAULT_MIG_DIR" "$DEFAULT_MIG_DIR/seed")
  TABLES=("goose_db_version" "goose_db_version_seed")
fi

if [[ ${#PATHS[@]} -ne ${#TABLES[@]} ]]; then
  echo "❌ --path 数量与 --table 数量必须一一对应" >&2
  exit 2
fi

# 需要 psql 客户端；docker 环境下可走 docker exec
PSQL_CMD="${PSQL_CMD:-}"
if [[ -z "$PSQL_CMD" ]]; then
  if command -v psql >/dev/null 2>&1; then
    PSQL_CMD="psql"
  elif command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q docker-postgres-1; then
    PSQL_CMD="docker exec -i docker-postgres-1 psql"
  else
    echo "❌ 找不到 psql 命令也没有 docker-postgres-1 容器；设 PSQL_CMD 环境变量手工指定" >&2
    exit 2
  fi
fi

run_psql() {
  $PSQL_CMD -t -A -F'|' "$DSN" -c "$1"
}

OVERALL_FAIL=0

# ── 检查 1：goose 已 applied 但磁盘无文件 ──────────────────────────
# ── 检查 2：磁盘有文件但 goose 未应用 + 中间跳号                  ──
for i in "${!PATHS[@]}"; do
  DIR="${PATHS[$i]}"
  TBL="${TABLES[$i]}"
  echo "🔍 对照 $DIR  vs  $TBL"

  if [[ ! -d "$DIR" ]]; then
    echo "   ⚠️  目录不存在，跳过"; echo ""; continue
  fi

  # 磁盘上的版本号集合
  DISK_VERSIONS=$(cd "$DIR" && ls -1 [0-9][0-9][0-9][0-9][0-9][0-9]_*.sql 2>/dev/null | \
                  sed 's/^0*//; s/_.*//' | sort -n | uniq)

  # goose 表里的版本号集合（applied=true，剔除 0 这个 baseline）
  GOOSE_VERSIONS=$(run_psql \
    "SELECT version_id FROM $TBL WHERE is_applied = true AND version_id > 0 ORDER BY version_id" 2>/dev/null || echo "")
  if [[ -z "$GOOSE_VERSIONS" ]]; then
    echo "   ⚠️  $TBL 表无数据或无法访问，跳过"; echo ""; continue
  fi

  # 检查 1：goose 有但磁盘无 → fake applied / 文件已删
  ORPHANS=$(comm -23 <(echo "$GOOSE_VERSIONS") <(echo "$DISK_VERSIONS") | grep -v '^$' || true)
  if [[ -n "$ORPHANS" ]]; then
    echo "   ❌ goose 记录 applied 但磁盘无对应 migration 文件（drift）："
    while IFS= read -r v; do
      echo "        version $v — 可能是 fake applied 或文件被删"
    done <<< "$ORPHANS"
    echo ""
    OVERALL_FAIL=1
  fi

  # 检查 2：磁盘有文件但 goose 没 applied + 中间跳号
  # goose 最大 applied 号 vs 磁盘最大文件号；中间任意一条磁盘有 + goose 无 → "中间漏跑"
  GOOSE_MAX=$(echo "$GOOSE_VERSIONS" | tail -1)
  MISSING_BEFORE_MAX=$(comm -23 <(echo "$DISK_VERSIONS") <(echo "$GOOSE_VERSIONS") | \
                       awk -v max="$GOOSE_MAX" '$1 <= max { print }')
  if [[ -n "$MISSING_BEFORE_MAX" ]]; then
    echo "   ❌ 磁盘有 migration 文件但 goose 未应用 + 编号低于已 applied 最大值（中间漏跑）："
    while IFS= read -r v; do
      # 找具体文件名
      FNAME=$(cd "$DIR" && ls -1 $(printf "%06d_*.sql" "$v") 2>/dev/null | head -1)
      echo "        version $v ($FNAME) — goose 跳过这条会下次启动重新尝试，可能引入预期外变更"
    done <<< "$MISSING_BEFORE_MAX"
    echo ""
    OVERALL_FAIL=1
  fi

  if [[ -z "$ORPHANS" && -z "$MISSING_BEFORE_MAX" ]]; then
    echo "   ✅ goose 记录与磁盘文件一致"
  fi
  echo ""
done

# ── 检查 3：最近 20 条 applied migration 里的 CREATE TABLE 真在 DB 里 ──
# 用启发式 grep 抓 CREATE TABLE [IF NOT EXISTS] <name>，对每个 name 查
# pg_tables 看是否存在。CREATE INDEX / CREATE VIEW 等不查（成本太大）。
echo "🔍 抽检：最近 20 条 applied migration 声明的表是否真在 DB 中"
LATEST=$(run_psql \
  "SELECT version_id FROM ${TABLES[0]} WHERE is_applied = true AND version_id > 0
   ORDER BY version_id DESC LIMIT 20" 2>/dev/null || echo "")

MISSING_TABLES=()
for v in $LATEST; do
  FNAME=$(cd "${PATHS[0]}" && ls -1 $(printf "%06d_*.sql" "$v") 2>/dev/null | head -1)
  [[ -z "$FNAME" ]] && continue
  # 抓出 CREATE TABLE 名（兼容 IF NOT EXISTS / schema-qualified / 表名带反引号）
  TABLES_DECLARED=$(grep -iE '^[[:space:]]*CREATE[[:space:]]+TABLE[[:space:]]+(IF[[:space:]]+NOT[[:space:]]+EXISTS[[:space:]]+)?' \
                    "${PATHS[0]}/$FNAME" | \
                    sed -E 's/^[[:space:]]*CREATE[[:space:]]+TABLE[[:space:]]+(IF[[:space:]]+NOT[[:space:]]+EXISTS[[:space:]]+)?//I' | \
                    sed -E 's/[[:space:]]*\(.*//' | \
                    sed -E 's/^"?([a-zA-Z_][a-zA-Z0-9_]*)\.?"?([a-zA-Z_][a-zA-Z0-9_]*)?"?.*/\2 \1/' | \
                    awk '{ if ($1 == "") print $2; else print $1 }' | \
                    sort -u)
  [[ -z "$TABLES_DECLARED" ]] && continue
  while IFS= read -r tbl; do
    EXISTS=$(run_psql "SELECT 1 FROM pg_tables WHERE tablename = '$tbl' LIMIT 1" 2>/dev/null || echo "")
    if [[ -z "$EXISTS" ]]; then
      MISSING_TABLES+=("$FNAME → $tbl")
    fi
  done <<< "$TABLES_DECLARED"
done

if [[ ${#MISSING_TABLES[@]} -gt 0 ]]; then
  echo "   ❌ 已 applied 的 migration 声明的表在 DB 中不存在（drift）："
  for m in "${MISSING_TABLES[@]}"; do
    echo "        $m"
  done
  echo ""
  echo "   修复：手工核对是否表被 DROP 过、或 goose 记录假 applied。"
  echo "   常见复原：DELETE FROM goose_db_version WHERE version_id = <NUM>; 再重跑 migrate"
  echo ""
  OVERALL_FAIL=1
else
  echo "   ✅ 最近 20 条 migration 声明的表都在 DB 中"
  echo ""
fi

# ── 总结 ───────────────────────────────────────────────────────
if [[ "$OVERALL_FAIL" -eq 0 ]]; then
  echo "════════════════════════════════════════"
  echo "✅ schema drift 检查全部通过"
  echo "════════════════════════════════════════"
  exit 0
else
  echo "════════════════════════════════════════"
  echo "❌ 发现 schema drift，请按上文修复"
  echo "════════════════════════════════════════"
  exit 1
fi

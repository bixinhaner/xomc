#!/usr/bin/env bash
# check-migrations.sh
# 校验 omcgo/migrations/ 下迁移文件的编号连续性、up/down 标记、命名规范。
# 用途：
#   - PR CI Gate
#   - `make check` / 本地预检
# 退出码：
#   0: 全部通过
#   1: 发现问题（不阻塞 hotfix，但必须在 release-gate 前修复）

set -euo pipefail

MIG_DIR="${MIG_DIR:-$(dirname "$0")/../migrations}"

if [[ ! -d "$MIG_DIR" ]]; then
  echo "❌ migrations 目录不存在：$MIG_DIR" >&2
  exit 1
fi

cd "$MIG_DIR"

echo "🔍 检查目录：$(pwd)"
echo ""

# ── 收集所有迁移文件（兼容 bash 3.2） ───────────────────────────
FILES=()
while IFS= read -r line; do
  FILES+=("$line")
done < <(ls -1 [0-9][0-9][0-9][0-9][0-9][0-9]_*.sql 2>/dev/null | sort)

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "❌ 未发现任何迁移文件（期望格式 000NNN_*.sql）"
  exit 1
fi

echo "📋 共发现 ${#FILES[@]} 个迁移文件"
echo ""

# ── 检查编号连续性（兼容 bash 3.2，不用 -A 关联数组） ─────────────
FAIL=0
NUMS=()
for f in "${FILES[@]}"; do
  NUM="${f:0:6}"
  NUMS+=("$NUM")
done

# 排序并取首尾
SORTED=$(printf '%s\n' "${NUMS[@]}" | sort -n)
MIN=$(echo "$SORTED" | head -1)
MAX=$(echo "$SORTED" | tail -1)
NUMS_LIST=$(echo "$SORTED" | tr '\n' ' ')

echo "📊 编号区间：$MIN → $MAX"
echo ""

GAPS=()
for ((i = 10#$MIN; i <= 10#$MAX; i++)); do
  NUM=$(printf "%06d" "$i")
  # 用空格包裹避免前缀匹配
  case " $NUMS_LIST " in
    *" $NUM "*) ;;
    *) GAPS+=("$NUM") ;;
  esac
done

if [[ ${#GAPS[@]} -gt 0 ]]; then
  echo "⚠️  编号不连续，缺失以下编号："
  for g in "${GAPS[@]}"; do
    echo "     - $g"
  done
  echo ""
  echo "   修复建议（任选一）："
  echo "   (a) 创建占位迁移：在缺口写 \"-- noop: reserved\" 的 up/down，保持编号连续"
  echo "   (b) 重新编号：将现有迁移重编号使其连续（需同步更新 goose 记录，谨慎操作）"
  echo ""
  FAIL=1
else
  echo "✅ 编号连续（无跳跃）"
  echo ""
fi

# ── 检查 goose up/down 标记 ─────────────────────────────────────
MISSING_GOOSE=()
for f in "${FILES[@]}"; do
  if ! grep -q -- "-- +goose Up" "$f"; then
    MISSING_GOOSE+=("$f (missing Up)")
  fi
  if ! grep -q -- "-- +goose Down" "$f"; then
    MISSING_GOOSE+=("$f (missing Down)")
  fi
done

if [[ ${#MISSING_GOOSE[@]} -gt 0 ]]; then
  echo "⚠️  以下文件缺少 goose 标记："
  for m in "${MISSING_GOOSE[@]}"; do
    echo "     - $m"
  done
  echo ""
  FAIL=1
else
  echo "✅ 所有文件 goose Up/Down 标记齐全"
  echo ""
fi

# ── 检查 Down 段是否真空（无任何 SQL 语句） ─────────────────────
# 统计 Down 段内的"非注释非空行"数：grep -v 已剥掉 `-- +goose Down` 标记，
# 所以 0 行 = 真空；>= 1 行 = 至少有一条 SQL（包括占位的 `SELECT 1;`）
EMPTY_DOWN=()
for f in "${FILES[@]}"; do
  DOWN_BODY=$(awk '/-- \+goose Down/,0' "$f" | \
              grep -v '^\s*--' | \
              grep -v '^\s*$' | \
              wc -l)
  if [[ "$DOWN_BODY" -lt 1 ]]; then
    EMPTY_DOWN+=("$f")
  fi
done

if [[ ${#EMPTY_DOWN[@]} -gt 0 ]]; then
  echo "⚠️  以下文件 Down 段疑似为空（无回滚 SQL）："
  for e in "${EMPTY_DOWN[@]}"; do
    echo "     - $e"
  done
  echo ""
  echo "   说明：如确为无法回滚（如删除数据），请在 Down 段明确注释理由；"
  echo "   否则按 CLAUDE.md §10 应提供可执行的回滚 SQL。"
  echo ""
  # Down 空不算 hard fail，只告警
fi

# ── 检查命名规范 ────────────────────────────────────────────────
BAD_NAME=()
for f in "${FILES[@]}"; do
  if [[ ! "$f" =~ ^[0-9]{6}_[a-z0-9_]+\.sql$ ]]; then
    BAD_NAME+=("$f")
  fi
done

if [[ ${#BAD_NAME[@]} -gt 0 ]]; then
  echo "⚠️  以下文件命名不规范（期望 000NNN_snake_case.sql）："
  for b in "${BAD_NAME[@]}"; do
    echo "     - $b"
  done
  echo ""
  FAIL=1
else
  echo "✅ 命名规范"
  echo ""
fi

# ── 总结 ───────────────────────────────────────────────────────
if [[ "$FAIL" -eq 0 ]]; then
  echo "════════════════════════════════════════"
  echo "✅ 迁移检查全部通过"
  echo "════════════════════════════════════════"
  exit 0
else
  echo "════════════════════════════════════════"
  echo "❌ 迁移检查发现问题，请按上文提示修复"
  echo "════════════════════════════════════════"
  exit 1
fi

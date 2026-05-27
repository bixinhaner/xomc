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

# CI 模式（--strict / 环境变量 CHECK_MIG_STRICT=1）：
#   只把"会让 goose / app 启动 panic"的硬错（重复版本号 + 缺 goose Up/Down 标记）
#   归为非零退出码；老仓库里 pre-existing 的编号 gap（66-79 等）+ 命名不规范
#   只 warn 不 fail。在 PR CI 上启用这个模式可以防止新的撞号事故，又不会因为
#   旧的历史包袱反复挂红。
#
# 默认模式（无参数）保留 hard fail 全集，给本地开发 / release-gate 用。
STRICT="${CHECK_MIG_STRICT:-0}"
for arg in "$@"; do
  [[ "$arg" == "--strict" ]] && STRICT=1
done

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

# ── ⚠️  HARD FAIL：同目录内禁止版本号重复 ─────────────────────────
# 历史教训：
#   · 2026-05-26 b00bd33a (mml) 与 7b8bb3b9 (pm) 撞 000191
#       → docker-migrate-schema panic "duplicate version 191 detected"
#       → 整条依赖链 acs / app / worker / web 全停
#   · goose 启动时 sort migrations 走 Migrations.Less，撞号直接 panic 退出
#     而不是跳过 —— PR 合并前必须拦住
DUPS=$(printf '%s\n' "${NUMS[@]}" | sort | uniq -d)
if [[ -n "$DUPS" ]]; then
  echo "❌ 同目录内存在重复版本号（goose 启动 panic）："
  while IFS= read -r d; do
    echo "     - 版本 $d 重复："
    for f in "${FILES[@]}"; do
      [[ "$f" == "$d"* ]] && echo "         · $f"
    done
  done <<< "$DUPS"
  echo ""
  echo "   修复：把后合入的文件改名到下一个空闲版本号（§5.5 多人协作规则）"
  echo ""
  FAIL=1
fi

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
  # CI strict 模式下编号 gap 是历史包袱，仅 warn 不 FAIL
  [[ "$STRICT" == "1" ]] || FAIL=1
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
  # CI strict：命名不规范是历史包袱，仅 warn 不 FAIL（影响审计但不影响启动）
  [[ "$STRICT" == "1" ]] || FAIL=1
else
  echo "✅ 命名规范"
  echo ""
fi

# ── 跨目录冲突：seed/ 与 migrations/ 不可有同 version_id ──────────
# goose 用 goose_db_version 表里的 version_id 全局唯一作为 PRIMARY KEY；
# 即便 migrations/000NNN.sql 与 migrations/seed/000NNN.sql 路径不同，跑到第二个
# 同号迁移时也会被静默跳过（不应用），导致预期数据没落库。
# 历史教训：commit 4742b792 北向 seed 用 000066 与 既有 000066_seed_role_menus_builtin.sql
# 撞号，被 commit 2d8486c5 重命名修复。本检查从根上杜绝。
SEED_DIR="$(pwd)/seed"
if [[ -d "$SEED_DIR" ]]; then
  SEED_FILES=()
  while IFS= read -r line; do
    SEED_FILES+=("$line")
  done < <(cd "$SEED_DIR" && ls -1 [0-9][0-9][0-9][0-9][0-9][0-9]_*.sql 2>/dev/null | sort)

  if [[ ${#SEED_FILES[@]} -gt 0 ]]; then
    echo "📋 seed 目录共发现 ${#SEED_FILES[@]} 个迁移文件"

    # 收集 seed 版本号
    SEED_NUMS=()
    for f in "${SEED_FILES[@]}"; do
      SEED_NUMS+=("${f:0:6}")
    done
    SEED_NUMS_LIST=$(printf '%s\n' "${SEED_NUMS[@]}" | sort -n | tr '\n' ' ')

    # 与 migrations/ 集合交集 = 同号冲突
    COLLIDE=()
    for sn in "${SEED_NUMS[@]}"; do
      case " $NUMS_LIST " in
        *" $sn "*) COLLIDE+=("$sn") ;;
      esac
    done

    if [[ ${#COLLIDE[@]} -gt 0 ]]; then
      # 仅 WARN 不 FAIL：origin 主线已有多处 pre-existing 同号（27/28/29/57-60/65），
      # 现网默认接受（先 schema 再 seed 顺序 + idempotent 写法 + 内容互不冲突）。
      # 新增冲突务必在 review 时基于此输出修正：把后合入的文件抬到更大版本号。
      echo "⚠️  跨目录同号冲突（goose version_id 全局 PRIMARY KEY 全表唯一，"
      echo "    第二个同号迁移会被 goose 静默跳过 → 数据可能不应用）："
      for c in "${COLLIDE[@]}"; do
        MFILE=$(printf '%s\n' "${FILES[@]}" | grep "^$c" | head -1)
        SFILE=$(printf '%s\n' "${SEED_FILES[@]}" | grep "^$c" | head -1)
        echo "     - $c"
        echo "         migrations/    : $MFILE"
        echo "         migrations/seed/: $SFILE"
      done
      echo ""
      echo "   ⚠️  存在 ${#COLLIDE[@]} 处冲突。新加冲突务必修复（重命名后合入文件为更大版本号）；"
      echo "       既有冲突可由 release-gate 集中清理 — 不阻塞当前 CI。"
      echo ""
    else
      echo "✅ migrations/ 与 seed/ 无跨目录同号冲突"
      echo ""
    fi

    # seed 目录内自检（不要求连续，但要求 up/down 标记齐全 + 命名规范）
    SEED_MISSING_GOOSE=()
    SEED_BAD_NAME=()
    for f in "${SEED_FILES[@]}"; do
      if ! grep -q -- "-- +goose Up" "$SEED_DIR/$f"; then
        SEED_MISSING_GOOSE+=("seed/$f (missing Up)")
      fi
      if ! grep -q -- "-- +goose Down" "$SEED_DIR/$f"; then
        SEED_MISSING_GOOSE+=("seed/$f (missing Down)")
      fi
      if [[ ! "$f" =~ ^[0-9]{6}_[a-z0-9_]+\.sql$ ]]; then
        SEED_BAD_NAME+=("seed/$f")
      fi
    done

    # seed/ 目录内自身同号重复（同主目录一样 hard fail）
    SEED_DUPS=$(printf '%s\n' "${SEED_NUMS[@]}" | sort | uniq -d)
    if [[ -n "$SEED_DUPS" ]]; then
      echo "❌ seed/ 目录内存在重复版本号（goose 启动 panic）："
      while IFS= read -r d; do
        echo "     - 版本 $d 重复："
        for f in "${SEED_FILES[@]}"; do
          [[ "$f" == "$d"* ]] && echo "         · seed/$f"
        done
      done <<< "$SEED_DUPS"
      echo ""
      FAIL=1
    fi

    if [[ ${#SEED_MISSING_GOOSE[@]} -gt 0 ]]; then
      echo "⚠️  seed 目录文件缺少 goose 标记："
      for m in "${SEED_MISSING_GOOSE[@]}"; do
        echo "     - $m"
      done
      echo ""
      FAIL=1
    fi
    if [[ ${#SEED_BAD_NAME[@]} -gt 0 ]]; then
      echo "⚠️  seed 目录文件命名不规范："
      for b in "${SEED_BAD_NAME[@]}"; do
        echo "     - $b"
      done
      echo ""
      [[ "$STRICT" == "1" ]] || FAIL=1
    fi
  fi
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

#!/usr/bin/env bash
# =============================================================================
# OMC 离线发布产物清理
#
# 清理 build-images.sh 与 build-release.sh 在 archive/ 下产生的版本目录:
#   - archive/project/<版本>/   ← build-release.sh
#   - archive/infra/<版本>/     ← build-images.sh
#
# 默认交互式询问每类要"保留最新的 N 个版本"(按目录 mtime 倒序);
# 输入 0 = 全部清理;直接回车采用默认值(--default-keep,默认 3)。
# 也支持非交互模式 -y 自动按默认值跑。
#
# project / infra 两类**各自独立**询问/保留,互不影响。
#
# 用法:
#   ./clean.sh                              # 交互式询问保留数量
#   ./clean.sh -y                           # 非交互,按 --default-keep
#   ./clean.sh --default-keep 5             # 默认保留数(交互回车 / -y 使用)
#   ./clean.sh --keep-project 3 --keep-infra 1
#                                           # 非交互,显式指定保留数
#   ./clean.sh --keep-project 0             # 全部清理 project,infra 走默认
#   ./clean.sh --dry-run                    # 只列要删的,不实际 rm
#   ./clean.sh --archive <dir>              # 自定义 archive 目录(测试用)
#   ./clean.sh -h | --help                  # 本帮助
#
# 参数:
#   -y, --yes                  非交互,使用 --default-keep / 显式 keep 值
#   --default-keep <N>         交互回车 / -y 时的默认保留数(默认 3)
#   --keep-project <N>         显式保留 project 最新 N 个(0=全删)
#   --keep-infra <N>           显式保留 infra 最新 N 个(0=全删)
#   --dry-run                  只列要删的目录,不真删
#   --archive <dir>            归档目录(默认 deployments/release/archive)
#   -h, --help                 本帮助
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARCHIVE="$SCRIPT_DIR/archive"
DEFAULT_KEEP=3
NON_INTERACTIVE=0
KEEP_PROJECT_OPT=""   # 空 = 未显式指定,交互/-y 取 DEFAULT_KEEP
KEEP_INFRA_OPT=""
DRY_RUN=0

while [ $# -gt 0 ]; do
  case "$1" in
    -y|--yes)         NON_INTERACTIVE=1; shift ;;
    --default-keep)   DEFAULT_KEEP="$2"; shift 2 ;;
    --keep-project)   KEEP_PROJECT_OPT="$2"; shift 2 ;;
    --keep-infra)     KEEP_INFRA_OPT="$2"; shift 2 ;;
    --dry-run)        DRY_RUN=1; shift ;;
    --archive)        ARCHIVE="$2"; shift 2 ;;
    -h|--help)        sed -n '3,40p' "$0"; exit 0 ;;
    *)                echo "未知参数:$1(-h 查看用法)" >&2; exit 1 ;;
  esac
done

log() { printf '%s\n' "$*" >&2; }

# 校验非负整数
is_non_negative_int() {
  case "$1" in
    ''|*[!0-9]*) return 1 ;;
    *)           return 0 ;;
  esac
}

# list_versions <dir> — 列出 <dir> 下的子目录,按 mtime 倒序(最新在前),
# 仅输出目录名(不含路径)。空目录 / 不存在时静默,输出空。
list_versions() {
  local d="$1"
  [ -d "$d" ] || return 0
  # find + stat 跨平台 mtime 字段不同:macOS BSD stat -f, Linux GNU stat -c。
  # 用 find -maxdepth 1 + ls -t 兜底(广泛兼容)。
  ( cd "$d" && ls -1td */ 2>/dev/null | sed 's:/$::' ) || true
}

# ask_keep <kind 中文标签> <dir> — 交互询问该类保留多少个;
# 输出选定的 keep 数到 stdout。
ask_keep() {
  local label="$1"
  local d="$2"
  local versions count default_show prompt input keep

  versions=$(list_versions "$d") || versions=""
  if [ -z "$versions" ]; then
    log "[$label] $d 下没有版本目录,跳过。"
    echo "-"   # 哨兵值,调用方识别为 skip
    return 0
  fi
  count=$(echo "$versions" | wc -l | tr -d ' ')

  log ""
  log "[$label] 当前 $count 个版本(按修改时间倒序,最新在前):"
  echo "$versions" | nl -ba -w3 -s'  ' >&2

  if [ "$NON_INTERACTIVE" -eq 1 ]; then
    keep="$DEFAULT_KEEP"
    log "[$label] 非交互模式 → 保留最新 $keep 个"
    echo "$keep"
    return 0
  fi

  default_show="$DEFAULT_KEEP"
  prompt="[$label] 请输入要保留的最新版本数 (回车=$default_show, 0=全部清理): "
  while :; do
    printf '%s' "$prompt" >&2
    if ! IFS= read -r input; then
      input=""
    fi
    input="${input:-$default_show}"
    if is_non_negative_int "$input"; then
      keep="$input"
      break
    fi
    log "输入非法,请输入非负整数。"
  done
  echo "$keep"
}

# clean_kind <label> <dir> <keep>
clean_kind() {
  local label="$1"
  local d="$2"
  local keep="$3"
  local versions to_delete count

  [ -d "$d" ] || { log "[$label] $d 不存在,跳过。"; return 0; }
  versions=$(list_versions "$d") || versions=""
  if [ -z "$versions" ]; then
    log "[$label] $d 下没有版本目录,跳过。"
    return 0
  fi
  count=$(echo "$versions" | wc -l | tr -d ' ')

  if [ "$keep" -ge "$count" ]; then
    log "[$label] 保留 $keep ≥ 当前 $count,无需清理。"
    return 0
  fi

  # 从第 (keep+1) 行起就是要删的
  to_delete=$(echo "$versions" | tail -n "+$((keep + 1))")
  log "[$label] 将删除 $(echo "$to_delete" | wc -l | tr -d ' ') 个旧版本(保留最新 $keep 个):"
  echo "$to_delete" | sed 's/^/  - /' >&2

  if [ "$DRY_RUN" -eq 1 ]; then
    log "[$label] --dry-run,未实际删除。"
    return 0
  fi

  while IFS= read -r ver; do
    [ -z "$ver" ] && continue
    rm -rf "$d/$ver"
  done <<< "$to_delete"
  log "[$label] 已清理。"
}

# 验证 --keep-project / --keep-infra 显式值
if [ -n "$KEEP_PROJECT_OPT" ] && ! is_non_negative_int "$KEEP_PROJECT_OPT"; then
  echo "--keep-project 必须是非负整数:$KEEP_PROJECT_OPT" >&2
  exit 1
fi
if [ -n "$KEEP_INFRA_OPT" ] && ! is_non_negative_int "$KEEP_INFRA_OPT"; then
  echo "--keep-infra 必须是非负整数:$KEEP_INFRA_OPT" >&2
  exit 1
fi
if ! is_non_negative_int "$DEFAULT_KEEP"; then
  echo "--default-keep 必须是非负整数:$DEFAULT_KEEP" >&2
  exit 1
fi

log "归档目录: $ARCHIVE"
[ "$DRY_RUN" -eq 1 ] && log "模式: dry-run(只列不删)"

# ── project (build-release.sh 产出) ─────────────────────────────────
PROJECT_DIR="$ARCHIVE/project"
if [ -n "$KEEP_PROJECT_OPT" ]; then
  keep_project="$KEEP_PROJECT_OPT"
  log ""
  log "[项目交付包] --keep-project=$keep_project"
else
  keep_project=$(ask_keep "项目交付包" "$PROJECT_DIR")
fi
[ "$keep_project" = "-" ] || clean_kind "项目交付包" "$PROJECT_DIR" "$keep_project"

# ── infra (build-images.sh 产出) ────────────────────────────────────
INFRA_DIR="$ARCHIVE/infra"
if [ -n "$KEEP_INFRA_OPT" ]; then
  keep_infra="$KEEP_INFRA_OPT"
  log ""
  log "[基础设施包] --keep-infra=$keep_infra"
else
  keep_infra=$(ask_keep "基础设施包" "$INFRA_DIR")
fi
[ "$keep_infra" = "-" ] || clean_kind "基础设施包" "$INFRA_DIR" "$keep_infra"

# ── 刷新下载索引 ────────────────────────────────────────────────────
if [ "$DRY_RUN" -ne 1 ] && [ -x "$SCRIPT_DIR/gen-index.sh" ]; then
  log ""
  log "刷新下载索引 archive/index.html ..."
  "$SCRIPT_DIR/gen-index.sh" --archive "$ARCHIVE" >/dev/null
  log "已刷新。"
fi

log ""
log "完成。"

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
# 询问前先做一轮预清理:扫描两类归档,删除"0 个交付文件"的空版本目录
# (build 失败常残留的空壳,下载索引上显示"0 个交付文件"),这步无须确认,
# --dry-run 时同样只列不删。
#
# 用法:
#   ./clean.sh                              # 交互式询问保留数量
#   ./clean.sh -y                           # 非交互,按 --default-keep
#   ./clean.sh --default-keep 5             # 默认保留数(交互回车 / -y 使用)
#   ./clean.sh --keep-project 3 --keep-infra 1
#                                           # 非交互,显式指定保留数
#   ./clean.sh --keep-project 0             # 全部清理 project,infra 走默认
#   ./clean.sh --list                       # 仅列出当前版本(含 mtime / 大小 / 文件数),退出
#   ./clean.sh --delete-project v1,v2       # 精确删 project 下指定版本(逗号分隔多个)
#   ./clean.sh --delete-infra v3            # 精确删 infra 下指定版本
#   ./clean.sh --delete-project v1 --delete-infra v3
#                                           # 两类同时精确删
#   ./clean.sh --delete-project v1 --dry-run
#                                           # 列要删但不真删
#   ./clean.sh --dry-run                    # 只列要删的,不实际 rm
#   ./clean.sh --archive <dir>              # 自定义 archive 目录(测试用)
#   ./clean.sh -h | --help                  # 本帮助
#
# 参数:
#   -y, --yes                  非交互,使用 --default-keep / 显式 keep 值
#   --default-keep <N>         交互回车 / -y 时的默认保留数(默认 3)
#   --keep-project <N>         显式保留 project 最新 N 个(0=全删)
#   --keep-infra <N>           显式保留 infra 最新 N 个(0=全删)
#   -l, --list                 仅列出 project + infra 下所有版本,退出(不做任何修改)
#   --delete-project <vs>      精确删 project 下指定版本(逗号分隔多个);
#                              与 keep-N / 询问流程互斥,也跳过空目录预清理。
#                              指定的版本不存在 → 整体报错退出 1,避免静默无操作。
#   --delete-infra <vs>        同上,但作用于 infra
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
LIST_ONLY=0
DELETE_PROJECT_VERS=""   # 逗号分隔列表;非空 = 精确删模式
DELETE_INFRA_VERS=""

while [ $# -gt 0 ]; do
  case "$1" in
    -y|--yes)             NON_INTERACTIVE=1; shift ;;
    --default-keep)       DEFAULT_KEEP="$2"; shift 2 ;;
    --keep-project)       KEEP_PROJECT_OPT="$2"; shift 2 ;;
    --keep-infra)         KEEP_INFRA_OPT="$2"; shift 2 ;;
    -l|--list)            LIST_ONLY=1; shift ;;
    --delete-project)     DELETE_PROJECT_VERS="$2"; shift 2 ;;
    --delete-infra)       DELETE_INFRA_VERS="$2"; shift 2 ;;
    --dry-run)            DRY_RUN=1; shift ;;
    --archive)            ARCHIVE="$2"; shift 2 ;;
    -h|--help)            sed -n '3,55p' "$0"; exit 0 ;;
    *)                    echo "未知参数:$1(-h 查看用法)" >&2; exit 1 ;;
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

# has_artifact <ver_dir> — 该版本目录下是否存在至少一个交付包(.tar.*,
# 排除 .sha256 校验文件)。与 gen-index.sh 对"交付文件"的定义口径一致:
# build-release.sh 失败时常留空版本目录(只有 .work 或不留任何东西),
# 这些目录在下载索引上显示"0 个交付文件",清理时应优先删除。
has_artifact() {
  local d="$1"
  [ -d "$d" ] || return 1
  local f
  for f in "$d"/*.tar.xz "$d"/*.tar.gz "$d"/*.tar.zst "$d"/*.tgz; do
    [ -f "$f" ] || continue
    case "$f" in *.sha256) continue ;; esac
    return 0
  done
  return 1
}

# prune_empty <label> <dir> — 扫描 <dir> 下所有版本子目录,删除不含任何
# 交付包的"空壳"(通常是 build 中途 die 留下的)。
# 此步骤无须用户确认,因为这些目录从下载视角看就是垃圾;在 keep-N 询问
# 之前先跑一遍,让交互列表只包含"真有产物"的版本。
prune_empty() {
  local label="$1"
  local d="$2"
  local versions empty=()
  [ -d "$d" ] || return 0
  versions=$(list_versions "$d") || return 0
  [ -z "$versions" ] && return 0
  while IFS= read -r ver; do
    [ -z "$ver" ] && continue
    has_artifact "$d/$ver" || empty+=( "$ver" )
  done <<< "$versions"
  [ "${#empty[@]}" -eq 0 ] && return 0

  log ""
  log "[$label] 检测到 ${#empty[@]} 个空版本目录(无 .tar.* 交付文件),将清理:"
  printf '  - %s\n' "${empty[@]}" >&2
  if [ "$DRY_RUN" -eq 1 ]; then
    log "[$label] --dry-run,未实际删除。"
    return 0
  fi
  for ver in "${empty[@]}"; do
    rm -rf "$d/$ver"
  done
  log "[$label] 空目录已清理。"
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

# list_versions_detailed <label> <dir> — 列出 <dir> 下所有版本目录,
# 含 mtime / 大小 / 交付文件数,便于 --list 一眼看清现状。
# 列宽固定,适合脚本输出对齐。
list_versions_detailed() {
  local label="$1"
  local d="$2"
  local versions count v size mtime n_artifacts
  log ""
  log "[$label] $d"
  versions=$(list_versions "$d") || versions=""
  if [ -z "$versions" ]; then
    log "  (空,无版本目录)"
    return 0
  fi
  count=$(echo "$versions" | wc -l | tr -d ' ')
  log "  共 $count 个(按 mtime 倒序,最新在前):"
  printf '  %-4s  %-32s  %-16s  %-6s  %s\n' "#" "版本目录" "修改时间" "大小" "交付文件数" >&2
  local i=0
  while IFS= read -r v; do
    [ -z "$v" ] && continue
    i=$((i + 1))
    size=$(du -sh "$d/$v" 2>/dev/null | awk '{print $1}')
    # date -r 在 macOS BSD 和 GNU coreutils 上都支持
    mtime=$(date -r "$d/$v" '+%Y-%m-%d %H:%M' 2>/dev/null || echo '-')
    n_artifacts=0
    local f
    for f in "$d/$v"/*.tar.xz "$d/$v"/*.tar.gz "$d/$v"/*.tar.zst "$d/$v"/*.tgz; do
      [ -f "$f" ] || continue
      case "$f" in *.sha256) continue ;; esac
      n_artifacts=$((n_artifacts + 1))
    done
    printf '  %-4s  %-32s  %-16s  %-6s  %s\n' "$i" "$v" "$mtime" "$size" "$n_artifacts" >&2
  done <<< "$versions"
}

# delete_specific <label> <dir> <comma-separated versions>
# 精确删指定版本。任何一个不存在 → 整体报错退出 1(避免静默无操作)。
# --dry-run 时只列要删的目录,不真删。
delete_specific() {
  local label="$1"
  local d="$2"
  local vers_csv="$3"
  local versions_input=()
  local missing=()
  local found=()

  if [ ! -d "$d" ]; then
    echo "[$label] 错误:$d 不存在" >&2
    return 1
  fi

  # 逗号 / 空白分割
  IFS=', ' read -r -a versions_input <<< "$vers_csv"

  local v
  for v in "${versions_input[@]}"; do
    [ -z "$v" ] && continue
    # 安全:禁止 .. / 绝对路径 / 含 /
    case "$v" in
      */*|.|..|/*) echo "[$label] 错误:非法版本名 '$v'(含路径分隔符)" >&2; return 1 ;;
    esac
    if [ -d "$d/$v" ]; then
      found+=( "$v" )
    else
      missing+=( "$v" )
    fi
  done

  if [ "${#missing[@]}" -gt 0 ]; then
    echo "[$label] 错误:以下版本不存在,中止整批删除:" >&2
    printf '  - %s\n' "${missing[@]}" >&2
    log ""
    log "[$label] $d 下当前可选版本:"
    list_versions "$d" | sed 's/^/  - /' >&2
    return 1
  fi

  if [ "${#found[@]}" -eq 0 ]; then
    log "[$label] 未指定任何版本,跳过。"
    return 0
  fi

  log ""
  log "[$label] 将精确删除 ${#found[@]} 个版本:"
  printf '  - %s\n' "${found[@]}" >&2
  if [ "$DRY_RUN" -eq 1 ]; then
    log "[$label] --dry-run,未实际删除。"
    return 0
  fi
  for v in "${found[@]}"; do
    rm -rf "$d/$v"
  done
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

PROJECT_DIR="$ARCHIVE/project"
INFRA_DIR="$ARCHIVE/infra"

# ── 仅列出模式 — 跳过所有修改性步骤 ────────────────────────────────
if [ "$LIST_ONLY" -eq 1 ]; then
  list_versions_detailed "项目交付包" "$PROJECT_DIR"
  list_versions_detailed "基础设施包" "$INFRA_DIR"
  log ""
  log "完成(仅列出,未修改任何文件)。"
  exit 0
fi

# ── 精确删除模式 — 与 keep-N 互斥,也跳过空目录预清理 ──────────────
if [ -n "$DELETE_PROJECT_VERS" ] || [ -n "$DELETE_INFRA_VERS" ]; then
  if [ -n "$KEEP_PROJECT_OPT$KEEP_INFRA_OPT" ] || [ "$NON_INTERACTIVE" -eq 1 ]; then
    log "(提示:--delete-* 与 --keep-* / -y 互斥,忽略后者)"
  fi
  [ -n "$DELETE_PROJECT_VERS" ] && delete_specific "项目交付包" "$PROJECT_DIR" "$DELETE_PROJECT_VERS"
  [ -n "$DELETE_INFRA_VERS"   ] && delete_specific "基础设施包" "$INFRA_DIR"   "$DELETE_INFRA_VERS"

  # 刷新下载索引
  if [ "$DRY_RUN" -ne 1 ] && [ -x "$SCRIPT_DIR/gen-index.sh" ]; then
    log ""
    log "刷新下载索引 archive/index.html ..."
    "$SCRIPT_DIR/gen-index.sh" --archive "$ARCHIVE" >/dev/null
    log "已刷新。"
  fi
  log ""
  log "完成。"
  exit 0
fi

# ── 0. 预清理:删除"0 个交付文件"的空版本目录 ───────────────────────
# build 失败时常留下空版本目录(下载索引上显示"0 个交付文件"),
# 它们既不属于"最新 N 个版本",在 keep-N 询问列表里又会占位、误导。
# 先在 keep-N 询问之前把它们扫掉。
prune_empty "项目交付包" "$PROJECT_DIR"
prune_empty "基础设施包" "$INFRA_DIR"

# ── project (build-release.sh 产出) ─────────────────────────────────
if [ -n "$KEEP_PROJECT_OPT" ]; then
  keep_project="$KEEP_PROJECT_OPT"
  log ""
  log "[项目交付包] --keep-project=$keep_project"
else
  keep_project=$(ask_keep "项目交付包" "$PROJECT_DIR")
fi
[ "$keep_project" = "-" ] || clean_kind "项目交付包" "$PROJECT_DIR" "$keep_project"

# ── infra (build-images.sh 产出) ────────────────────────────────────
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

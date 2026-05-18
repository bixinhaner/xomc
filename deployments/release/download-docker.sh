#!/usr/bin/env bash
# =============================================================================
# OMC Docker 引擎离线包下载工具
#
# 下载 Docker 静态二进制包（按 ARCHES 逐架构），存入 docker-cache/<arch>/，
# 供 build-release.sh 按架构打入交付包；运维侧用包内 install-docker.sh 离线
# 安装 Docker。
#
# 下载版本与地址由 release.conf 的 DOCKER_VERSION / DOCKER_URL_TEMPLATE 控制。
# Docker 引擎不常变更——仅在调整版本时运行本工具。
#
# 用法： ./download-docker.sh [--arch amd64|arm64] [--force]
#   --force  已存在也重新下载
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=release.conf
source "$SCRIPT_DIR/release.conf"

log()  { echo -e "\033[1;35m[docker-dl]\033[0m $*"; }
warn() { echo -e "\033[1;33m[docker-dl][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[docker-dl][错误]\033[0m $*" >&2; exit 1; }

# ── 参数解析 ────────────────────────────────────────────────────────────
FORCE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --arch)    ARCHES="$2"; shift 2 ;;
    --force)   FORCE=1; shift ;;
    -h|--help) sed -n '3,15p' "$0"; exit 0 ;;
    *)         die "未知参数：$1（-h 查看用法）" ;;
  esac
done

[ -n "${DOCKER_VERSION:-}" ]      || die "release.conf 未配置 DOCKER_VERSION"
[ -n "${DOCKER_URL_TEMPLATE:-}" ] || die "release.conf 未配置 DOCKER_URL_TEMPLATE"

# ── 下载工具：curl 优先，其次 wget ──────────────────────────────────────
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fL --progress-bar -o "$1" "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q --show-progress -O "$1" "$2"; }
else
  die "需要 curl 或 wget 才能下载"
fi

# 项目架构名 → Docker 官方静态二进制目录名
docker_arch() {
  case "$1" in
    amd64) echo x86_64 ;;
    arm64) echo aarch64 ;;
    *)     echo "$1" ;;
  esac
}

CACHE="$SCRIPT_DIR/docker-cache"
log "Docker 版本：$DOCKER_VERSION   架构：$ARCHES"

for ARCH in $ARCHES; do
  DARCH="$(docker_arch "$ARCH")"
  URL="${DOCKER_URL_TEMPLATE//\{arch\}/$DARCH}"
  DEST="$CACHE/$ARCH"
  OUT="$DEST/docker-$DOCKER_VERSION.tgz"
  mkdir -p "$DEST"

  if [ -f "$OUT" ] && [ "$FORCE" = 0 ]; then
    log "[$ARCH] 已存在，跳过：docker-cache/$ARCH/docker-$DOCKER_VERSION.tgz（--force 强制重下）"
    continue
  fi

  log "[$ARCH] 下载 $URL"
  # 先下到 .tmp，成功再就位，避免中断留下半包
  if ! fetch "$OUT.tmp" "$URL"; then
    rm -f "$OUT.tmp"
    die "[$ARCH] 下载失败：$URL（检查网络 / DOCKER_VERSION 是否存在）"
  fi
  mv "$OUT.tmp" "$OUT"
  log "[$ARCH] 完成 → docker-cache/$ARCH/docker-$DOCKER_VERSION.tgz"
done

log "全部完成。docker-cache/ 内的包将由 build-release.sh 按架构打入交付包。"
ls -lh "$CACHE"/*/docker-*.tgz 2>/dev/null || true

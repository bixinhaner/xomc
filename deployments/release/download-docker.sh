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
# 用法： ./download-docker.sh [--arch amd64] [--force]
#   --arch   当前【只支持 amd64】，非法值会被拒绝（见 release.conf 注释）
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
[ -n "${COMPOSE_VERSION:-}" ]      || die "release.conf 未配置 COMPOSE_VERSION"
[ -n "${COMPOSE_URL_TEMPLATE:-}" ] || die "release.conf 未配置 COMPOSE_URL_TEMPLATE"

# 仅支持 amd64（见 release.conf 注释 "架构支持"）
for _a in $ARCHES; do
  [ "$_a" = "amd64" ] || die "本工具仅支持 amd64 架构，传入：${_a}。
        如确实需要 arm64：见 release.conf 中关于 架构支持 的注释，
        改 ARCHES + 移除本脚本的校验后自行验证。"
done

# ── 下载工具：curl 优先，其次 wget ──────────────────────────────────────
if command -v curl >/dev/null 2>&1; then
  # --connect-timeout 15：TCP 握手 15s 不通就被判死（原默认 130s+）
  # --max-time 600：整体不超过 10 min（compose 单文件 ~50MB，够用）
  # --retry 2：临时错重试 2 次
  fetch() { curl -fL --connect-timeout 15 --max-time 600 --retry 2 --progress-bar -o "$1" "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q --show-progress --connect-timeout=15 --tries=2 -O "$1" "$2"; }
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
    log "[$ARCH] docker tgz 已存在，跳过：docker-cache/$ARCH/docker-$DOCKER_VERSION.tgz（--force 强制重下）"
  else
    log "[$ARCH] 下载 $URL"
    # 先下到 .tmp，成功再就位，避免中断留下半包
    if ! fetch "$OUT.tmp" "$URL"; then
      rm -f "$OUT.tmp"
      die "[$ARCH] 下载失败：$URL（检查网络 / DOCKER_VERSION 是否存在）"
    fi
    mv "$OUT.tmp" "$OUT"
    log "[$ARCH] 完成 → docker-cache/$ARCH/docker-$DOCKER_VERSION.tgz"
  fi

  # ── 同步下载 Docker Compose v2 静态二进制 ──────────────
  # compose 二进制名由 GitHub Release 资产名决定：docker-compose-linux-{x86_64|aarch64}
  # 交付包内以固定名 docker-compose 存放，install-docker.sh 同目录检测即装
  COMPOSE_OUT="$DEST/docker-compose-$COMPOSE_VERSION"
  COMPOSE_LINK="$DEST/docker-compose"

  if [ -f "$COMPOSE_OUT" ] && [ "$FORCE" = 0 ]; then
    log "[$ARCH] compose 已存在，跳过：docker-cache/$ARCH/docker-compose-$COMPOSE_VERSION"
  else
    # 多镜像源 fallback：依次试 COMPOSE_URL_TEMPLATES，命中即停
    if [ -n "${COMPOSE_URL_TEMPLATES+x}" ] && [ "${#COMPOSE_URL_TEMPLATES[@]}" -gt 0 ]; then
      _CANDIDATES=( "${COMPOSE_URL_TEMPLATES[@]}" )
    else
      _CANDIDATES=( "$COMPOSE_URL_TEMPLATE" )
    fi
    _OK=0
    for _TPL in "${_CANDIDATES[@]}"; do
      COMPOSE_URL="${_TPL//\{arch\}/$DARCH}"
      log "[$ARCH] 尝试下载 compose v2: $COMPOSE_URL"
      if fetch "$COMPOSE_OUT.tmp" "$COMPOSE_URL"; then
        _OK=1
        break
      fi
      warn "[$ARCH] 该镜像源不可用，试下一个：$COMPOSE_URL"
      rm -f "$COMPOSE_OUT.tmp"
    done
    if [ "$_OK" = 0 ]; then
      die "[$ARCH] compose 下载失败：所有镜像源均不可用（检查网络 / COMPOSE_VERSION / release.conf::COMPOSE_URL_TEMPLATES）"
    fi
    chmod +x "$COMPOSE_OUT.tmp"
    mv "$COMPOSE_OUT.tmp" "$COMPOSE_OUT"
    log "[$ARCH] compose 完成 → docker-cache/$ARCH/docker-compose-$COMPOSE_VERSION"
  fi
  # 固定名软链（build-images.sh 以此名拷入交付包）
  ln -sfn "docker-compose-$COMPOSE_VERSION" "$COMPOSE_LINK"
done

log "全部完成。docker-cache/ 内的包将由 build-release.sh 按架构打入交付包。"
ls -lh "$CACHE"/*/docker-*.tgz "$CACHE"/*/docker-compose-* 2>/dev/null || true

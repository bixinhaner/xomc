#!/usr/bin/env bash
# =============================================================================
# OMC 基础设施镜像构建工具
#
# 拉取基础设施 Docker 镜像并导出为离线 tar，产物存入 images-cache/，供
# build-release.sh 复用。
#
# 基础设施镜像（postgres/redis/nats/minio/nginx）【不常变更】——仅在调整
# release.conf 里的镜像版本时才需运行本工具；日常发版只跑 build-release.sh。
#
# 用法： ./build-images.sh [--arch amd64|arm64] [--with-monitoring]
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=release.conf
source "$SCRIPT_DIR/release.conf"

log()  { echo -e "\033[1;36m[images]\033[0m $*"; }
warn() { echo -e "\033[1;33m[images][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[images][错误]\033[0m $*" >&2; exit 1; }

# ── 参数解析 ────────────────────────────────────────────────────────────
WITH_MONITORING=0
while [ $# -gt 0 ]; do
  case "$1" in
    --arch)            ARCHES="$2"; shift 2 ;;
    --with-monitoring) WITH_MONITORING=1; shift ;;
    -h|--help)         sed -n '3,14p' "$0"; exit 0 ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── 前置检查 ────────────────────────────────────────────────────────────
command -v docker >/dev/null 2>&1 || die "缺少 docker"
if ! docker info >/dev/null 2>&1; then
  die "docker 不可用：当前用户可能不在 docker 组。请执行：
      sudo usermod -aG docker \$USER && newgrp docker   （或重新登录）
      然后重新运行本脚本。"
fi

CACHE="$SCRIPT_DIR/images-cache"
mkdir -p "$CACHE"

log "目标架构：$ARCHES   监控栈：$([ "$WITH_MONITORING" = 1 ] && echo 含 || echo 不含)"

# ── 逐架构拉取并导出 ────────────────────────────────────────────────────
# 镜像导出【不依赖构建机自身架构】：用 `docker pull --platform linux/$ARCH`
# 显式逐架构拉取。每个架构 pull 前先 `docker rmi` 清本地同名镜像，确保
# `docker save` 导出的就是本架构（不受遗留缓存 / containerd 多架构混存影响）。
for ARCH in $ARCHES; do
  log "=================== 架构 $ARCH ==================="
  IMG_LIST=( "${INFRA_IMAGES[@]}" )
  [ "$WITH_MONITORING" = 1 ] && IMG_LIST+=( "${MONITORING_IMAGES[@]}" )

  for IMG in "${IMG_LIST[@]}"; do
    docker rmi -f "$IMG" >/dev/null 2>&1 || true   # 清缓存，使 save 架构确定
    docker pull --platform "linux/$ARCH" "$IMG"
  done

  docker save -o "$CACHE/infra-images-$ARCH.tar" "${INFRA_IMAGES[@]}"
  log "[$ARCH] 导出 → images-cache/infra-images-$ARCH.tar"
  if [ "$WITH_MONITORING" = 1 ]; then
    docker save -o "$CACHE/monitoring-images-$ARCH.tar" "${MONITORING_IMAGES[@]}"
    log "[$ARCH] 导出 → images-cache/monitoring-images-$ARCH.tar"
  fi
done

# ── 记录本批镜像清单与构建时间（供 build-release.sh 带入交付包、供排错追溯）──
{
  echo "built_at=$(date -Is)"
  echo "arches=$ARCHES"
  echo "with_monitoring=$WITH_MONITORING"
  echo "# infra images:"
  printf '  %s\n' "${INFRA_IMAGES[@]}"
  if [ "$WITH_MONITORING" = 1 ]; then
    echo "# monitoring images:"
    printf '  %s\n' "${MONITORING_IMAGES[@]}"
  fi
} > "$CACHE/images.manifest"

log "完成。镜像缓存位于 images-cache/，可被 build-release.sh 反复复用。"
ls -lh "$CACHE"/*.tar 2>/dev/null || true

#!/usr/bin/env bash
# =============================================================================
# OMC 基础设施包构建工具
#
# 拉取基础设施 Docker 镜像并导出为离线 tar，连同 Docker 引擎离线包一起组装、
# 压缩成【基础设施交付包】omc-infra-<版本>-<架构>.tar.xz，按版本归档到
# archive/infra/。基础设施【不常变更】——仅在调整 release.conf 的镜像版本或
# Docker 版本时运行本工具；日常发版只跑 build-release.sh。
#
# 基础设施包与项目包（build-release.sh 产出）是【两个独立交付包】，各自维护
# 版本号：基础设施版本 INFRA_VERSION（见 release.conf，纯 semver，从 0.0.1 起）。
#
# 基础设施包内容：Docker 引擎离线安装包（docker-cache/ 由 download-docker.sh
# 下载）+ 基础镜像 tar（docker save）+ install-docker.sh。
#
# === 镜像缓存策略 ===
# 1. 跨架构镜像通过 *-arch-saved 后缀 tag 隔离，避免覆盖 redis:7-alpine 等原 tag
# 2. 本机架构原 tag 始终保持本机架构镜像，docker-compose 可同时使用
# 3. 镜像不在脚本中删除，作为下次构建的本地缓存
# 4. tar 内同时包含 *-saved 与原 tag，docker load 后部署侧可直接使用原 tag
#
# 用法： ./build-images.sh [-v 基础设施版本] [--arch amd64|arm64]
#                          [--with-monitoring | --monitoring-only]
#   --with-monitoring   基础设施 + 监控栈镜像都构建
#   --monitoring-only   只补监控栈镜像（不重拉基础设施，infra-images-*.tar 不动）
#   ★ 用普通用户运行（docker 权限靠 docker 组，勿 sudo 整个脚本）。
# =============================================================================
set -euo pipefail

# 本机架构（用于跨架构 pull 后修复原 tag，保护 docker-compose）
HOST_ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=release.conf
source "$SCRIPT_DIR/release.conf"

log()  { echo -e "\033[1;36m[images]\033[0m $*"; }
warn() { echo -e "\033[1;33m[images][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[images][错误]\033[0m $*" >&2; exit 1; }

# ── 参数解析 ────────────────────────────────────────────────────────────
WITH_MONITORING=0
MONITORING_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version)      INFRA_VERSION="$2"; shift 2 ;;
    --arch)            ARCHES="$2"; shift 2 ;;
    --with-monitoring) WITH_MONITORING=1; shift ;;
    --monitoring-only) MONITORING_ONLY=1; WITH_MONITORING=1; shift ;;
    -h|--help)         sed -n '3,26p' "$0"; exit 0 ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── 前置检查 ────────────────────────────────────────────────────────────
command -v docker >/dev/null 2>&1 || die "缺少 docker"
for tool in tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少打包工具：$tool"
done
if ! docker info >/dev/null 2>&1; then
  die "docker 不可用：当前用户可能不在 docker 组。请执行：
      sudo usermod -aG docker \$USER && newgrp docker   （或重新登录）
      然后重新运行本脚本。"
fi

# 压缩方式 → tar 选项与扩展名
case "$PKG_COMPRESS" in
  xz)   TAR_OPT="-cJf"; EXT="tar.xz" ;;
  gzip) TAR_OPT="-czf"; EXT="tar.gz" ;;
  zstd) TAR_OPT="--zstd -cf"; EXT="tar.zst"
        command -v zstd >/dev/null 2>&1 || die "PKG_COMPRESS=zstd 但未安装 zstd" ;;
  *)    die "release.conf 的 PKG_COMPRESS 取值非法：$PKG_COMPRESS（应为 xz/gzip/zstd）" ;;
esac

CACHE="$SCRIPT_DIR/images-cache"
mkdir -p "$CACHE"

# --monitoring-only：基础设施必须已构建过；版本与构建时间沿用既有 manifest
if [ "$MONITORING_ONLY" = 1 ]; then
  for ARCH in $ARCHES; do
    [ -f "$CACHE/infra-images-$ARCH.tar" ] || \
      die "缺 images-cache/infra-images-$ARCH.tar —— --monitoring-only 需先有基础设施镜像。
      请先不带该参数跑一次 ./build-images.sh 构建基础设施。"
  done
  if [ -f "$CACHE/images.manifest" ]; then
    INFRA_VERSION="$(grep -E '^infra_version=' "$CACHE/images.manifest" | cut -d= -f2 || echo "$INFRA_VERSION")"
    BUILT_AT="$(grep -E '^built_at=' "$CACHE/images.manifest" | cut -d= -f2 || date -Is)"
  else
    BUILT_AT="$(date -Is)"
  fi
else
  BUILT_AT="$(date -Is)"
fi

if [ "$MONITORING_ONLY" = 1 ]; then
  log "模式：仅补监控栈（基础设施 infra-images-*.tar 保持不动）"
fi
log "本机架构：$HOST_ARCH   构建架构：$ARCHES   监控栈：$([ "$WITH_MONITORING" = 1 ] && echo 含 || echo 不含)"
log "基础设施版本：$INFRA_VERSION"

# ── 拉取并打 *-saved tag ─────────────────────────────────────────────────
# 命名隔离：每架构镜像在本机以 <image>-<arch>-saved 留存，避免互相覆盖；命中即跳 pull。
# 跨架构 pull 后立即用本机架构再 pull 一次，把原 tag 修复回本机架构（保护 docker-compose）。
prepare_image() {
  local IMG="$1" ARCH="$2"
  local SAVED_TAG="${IMG}-${ARCH}-saved"
  if docker image inspect "$SAVED_TAG" >/dev/null 2>&1; then
    log "✓ 复用本地缓存: $SAVED_TAG"
    return 0
  fi
  docker pull --platform "linux/$ARCH" "$IMG"
  docker tag "$IMG" "$SAVED_TAG"
  log "✓ 已拉取并打标: $IMG → $SAVED_TAG"
  if [ "$ARCH" != "$HOST_ARCH" ]; then
    docker pull --platform "linux/$HOST_ARCH" "$IMG"
    log "✓ 已修复本机架构原 tag: $IMG (${HOST_ARCH})"
  fi
}

# 双 tag 导出：tar 内同时包含 *-saved 与原 tag。
# 实现：save 前先把原 tag 临时指向当前架构 *-saved（在 docker 索引层），save 后
# 若是跨架构循环再用本机架构 pull 修复回原 tag。HOST_ARCH 循环则原 tag 已对应。
save_with_dual_tags() {
  local OUT_TAR="$1" ARCH="$2"; shift 2
  local IMGS=("$@") REFS=() IMG SAVED_TAG
  for IMG in "${IMGS[@]}"; do
    SAVED_TAG="${IMG}-${ARCH}-saved"
    docker tag "$SAVED_TAG" "$IMG"
    REFS+=( "$SAVED_TAG" "$IMG" )
  done
  docker save -o "$OUT_TAR" "${REFS[@]}"
  if [ "$ARCH" != "$HOST_ARCH" ]; then
    for IMG in "${IMGS[@]}"; do
      docker pull --platform "linux/$HOST_ARCH" "$IMG"
      log "✓ 已修复本机架构原 tag: $IMG (${HOST_ARCH})"
    done
  fi
}

# ── 逐架构拉取并导出 ────────────────────────────────────────────────────
for ARCH in $ARCHES; do
  log "=================== 架构 $ARCH ==================="

  # 本架构本次要拉的镜像清单
  if [ "$MONITORING_ONLY" = 1 ]; then
    IMG_LIST=( "${MONITORING_IMAGES[@]}" )
  else
    IMG_LIST=( "${INFRA_IMAGES[@]}" )
    [ "$WITH_MONITORING" = 1 ] && IMG_LIST+=( "${MONITORING_IMAGES[@]}" )
  fi

  for IMG in "${IMG_LIST[@]}"; do
    prepare_image "$IMG" "$ARCH"
  done

  # 基础设施 tar：monitoring-only 模式下不动它
  if [ "$MONITORING_ONLY" = 0 ]; then
    save_with_dual_tags "$CACHE/infra-images-$ARCH.tar" "$ARCH" "${INFRA_IMAGES[@]}"
    log "[$ARCH] 导出 → images-cache/infra-images-$ARCH.tar （含 *-saved + 原 tag）"
  fi
  # 监控 tar：--with-monitoring / --monitoring-only 都会产出
  if [ "$WITH_MONITORING" = 1 ]; then
    save_with_dual_tags "$CACHE/monitoring-images-$ARCH.tar" "$ARCH" "${MONITORING_IMAGES[@]}"
    log "[$ARCH] 导出 → images-cache/monitoring-images-$ARCH.tar （含 *-saved + 原 tag）"
  fi
done

# ── 记录基础设施版本与镜像清单（带入交付包，供追溯）────────────────────────
{
  echo "infra_version=$INFRA_VERSION"
  echo "built_at=$BUILT_AT"
  echo "arches=$ARCHES"
  echo "with_monitoring=$WITH_MONITORING"
  echo "# infra images:"
  printf '  %s\n' "${INFRA_IMAGES[@]}"
  if [ "$WITH_MONITORING" = 1 ]; then
    echo "# monitoring images:"
    printf '  %s\n' "${MONITORING_IMAGES[@]}"
  fi
} > "$CACHE/images.manifest"

# ── 组装并压缩基础设施交付包 → archive/infra/<版本>/ ──────────────────────
# 内容：基础镜像 tar + Docker 引擎离线包 + install-docker.sh。
ARCHIVE="$SCRIPT_DIR/archive"
DOCKER_CACHE="$SCRIPT_DIR/docker-cache"
OUT="$ARCHIVE/infra/$INFRA_VERSION"
WORK="$SCRIPT_DIR/dist/.work-infra"
rm -rf "$WORK"
mkdir -p "$WORK" "$OUT"

for ARCH in $ARCHES; do
  log "=================== 打包 $ARCH ==================="
  if [ ! -f "$CACHE/infra-images-$ARCH.tar" ]; then
    warn "[$ARCH] 缺 images-cache/infra-images-$ARCH.tar —— 跳过该架构打包。"
    continue
  fi
  PKG_NAME="omc-infra-$INFRA_VERSION-$ARCH"
  STAGE="$WORK/$PKG_NAME"
  mkdir -p "$STAGE/images" "$STAGE/docker"

  # 基础镜像 + 镜像清单
  cp "$CACHE/infra-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/monitoring-images-$ARCH.tar" ] && \
    cp "$CACHE/monitoring-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/images.manifest" ] && cp "$CACHE/images.manifest" "$STAGE/images/"

  # Docker 引擎离线安装包（download-docker.sh 下载到 docker-cache/<arch>/）
  # install-docker.sh 末尾会调用 setup-docker-mirror.sh 引导加速镜像，两个都拷
  cp "$SCRIPT_DIR/bundle/docker/install-docker.sh"       "$STAGE/docker/"
  cp "$SCRIPT_DIR/bundle/docker/setup-docker-mirror.sh"  "$STAGE/docker/"
  if ls "$DOCKER_CACHE/$ARCH"/docker-*.tgz >/dev/null 2>&1; then
    cp "$DOCKER_CACHE/$ARCH"/docker-*.tgz "$STAGE/docker/"
    DOCKER_IN_PKG="${DOCKER_VERSION:-未知}"
  else
    warn "[$ARCH] docker-cache/$ARCH/ 无 Docker 引擎包 —— 基础设施包不含离线装 Docker 的包。"
    warn "        若目标机未装 Docker，请先运行： ./download-docker.sh"
    DOCKER_IN_PKG="未含"
  fi

  # VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
infra_version=$INFRA_VERSION
arch=$ARCH
docker_version=$DOCKER_IN_PKG
build_time=$BUILT_AT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 基础设施包 — $INFRA_VERSION ($ARCH)

- 基础设施版本：$INFRA_VERSION
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- Docker 版本：$DOCKER_IN_PKG
- 构建时间：$BUILT_AT

本包含【Docker 引擎离线安装包 + 基础镜像】（PostgreSQL / Redis / NATS /
MinIO / Nginx）。与项目包 omc-test-* / omc-release-* 相互独立，各自版本号。
首次部署或基础设施升级时使用。

部署步骤（先校验完整性）：
\`\`\`bash
sha256sum -c checksums.sha256
# 1. 安装 Docker（目标机未装时）
cd docker && sudo bash install-docker.sh && cd ..
# 2. 导入基础镜像（tar 内同时含 *-saved 与原 tag，运维只需关心原 tag）
docker load -i images/infra-images-$ARCH.tar
\`\`\`
之后再部署项目包，详见项目包内 \`docs/OMC内网离线部署手册（运维侧）.md\`。
EOF
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 压缩打包 → archive/infra/<版本>/
  log "[$ARCH] 压缩打包基础设施包（$PKG_COMPRESS）..."
  # shellcheck disable=SC2086
  ( cd "$WORK" && tar $TAR_OPT "$OUT/$PKG_NAME.$EXT" "$PKG_NAME" )
  ( cd "$OUT" && sha256sum "$PKG_NAME.$EXT" > "$PKG_NAME.$EXT.sha256" )
  log "[$ARCH] 产出：archive/infra/$INFRA_VERSION/$PKG_NAME.$EXT"
done

rm -rf "$WORK"

# ── 基础设施包构建说明 RELEASE.txt ───────────────────────────────────────
{
  echo "infra_version=$INFRA_VERSION"
  echo "docker_version=${DOCKER_VERSION:-未含}"
  echo "build_time=$BUILT_AT"
  echo "arches=$ARCHES"
  echo "compress=$PKG_COMPRESS"
  echo ""
  echo "# ── 基础设施包构建说明 ─────────────────────────────────────"
  echo "基础设施版本：$INFRA_VERSION"
  echo "Docker 版本 ：${DOCKER_VERSION:-未含}"
  echo ""
  echo "基础设施镜像清单："
  printf '  %s\n' "${INFRA_IMAGES[@]}"
  echo ""
  echo "交付文件："
  ( cd "$OUT" && ls -1 omc-infra-*."$EXT" 2>/dev/null || echo "（无 —— 检查 images-cache）" )
} > "$OUT/RELEASE.txt"

# ── 刷新 archive/index.html（HTTP 下载索引）──────────────────────────────
"$SCRIPT_DIR/gen-index.sh"
log "已刷新下载索引：archive/index.html"

# ── 完成 ────────────────────────────────────────────────────────────────
echo
log "全部完成。"
log "  基础设施版本 ：$INFRA_VERSION"
log "  归档位置     ：archive/infra/$INFRA_VERSION/"
ls -lh "$OUT"/omc-infra-*."$EXT" 2>/dev/null || true

# ── 本地缓存汇总 ───────────────────────────────────────────────────────
# *-saved tag 留作下次构建的本地缓存（命中即跳 pull）；脚本不再 docker rmi。
SAVED_COUNT=$(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null \
              | grep -E -- '-(amd64|arm64)-saved$' | wc -l | tr -d ' ' || echo 0)
echo
log "── 本地缓存汇总 ───────────────────────────────────────────"
log "  保留 *-saved tag 数量：$SAVED_COUNT"
log "  说明：本工具不再清理本地镜像，*-saved tag 作为下次增量构建的缓存。"
log "        本机架构原 tag 始终指向本机架构镜像，docker-compose 可同时使用。"
log "        如需手工清理：docker images | grep -- '-saved' | awk '{print \$1\":\"\$2}' | xargs docker rmi -f"
echo
log "项目交付包由 ./build-release.sh 生成，与本包独立。"
log "起 HTTP 下载服务： ./serve.sh    然后浏览器访问 http://<构建机IP>:8000/"

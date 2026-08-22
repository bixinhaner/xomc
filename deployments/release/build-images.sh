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
# === 镜像缓存策略（amd64-only） ===
# 1. 通过 *-amd64-saved 后缀 tag 作为本地缓存，命中即跳 docker pull
#    （保留 -amd64- 段命名作历史缓存兼容；不再有跨架构需求）
# 2. 原 tag（如 redis:7-alpine）始终指向 amd64 镜像，docker-compose 可同时使用
# 3. 镜像不在脚本中删除，作为下次构建的本地缓存（手工清理见末尾提示）
# 4. tar 内同时包含 *-saved 与原 tag，运维侧 docker load 后直接用原 tag
#
# 默认行为（v2 起调整）：拉取并打包 INFRA + MONITORING **全套镜像**。
# 想去掉监控栈用 --infra-only。
#
# 用法：
#   ./build-images.sh                            # 默认：基础设施 + 监控栈全套
#   ./build-images.sh --infra-only               # 仅基础设施（不含监控栈）
#   ./build-images.sh --monitoring-only          # 只补监控栈（不重拉 infra）
#   ./build-images.sh -v 0.0.2                   # 手动指定基础设施版本（仍会自动追加时间戳）
#   ./build-images.sh -h | --help                # 本帮助
#
# 参数：
#   -v, --version <ver>     基础设施版本号的基础部分（默认取 release.conf 的 INFRA_VERSION）。
#                           无论是否手动指定，最终版本都会自动追加 -YYYYMMDD-HHMM
#                           时间戳（例：-v 0.0.2 → 0.0.2-20260522-1530），保证镜像 tag 唯一。
#   --arch <amd64>          目标架构。【当前只支持 amd64】，非法值会被拒绝；
#                           原因见 release.conf 中"架构支持"注释
#   --infra-only            仅拉 + 打包基础设施镜像，不要监控栈（v2 新增）
#   --monitoring-only       仅补监控栈镜像，infra-images-*.tar 保持不动
#   --with-monitoring       【已废弃】v2 起默认即含监控栈；保留为 no-op + 提示
#   -h, --help              本帮助
#
# ★ 用普通用户运行（docker 权限靠 docker 组，勿 sudo 整个脚本）。
# =============================================================================
set -euo pipefail

# 本机架构（amd64-only：必须在 amd64 host 上运行）
HOST_ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=release.conf
source "$SCRIPT_DIR/release.conf"

log()  { echo -e "\033[1;36m[images]\033[0m $*"; }
warn() { echo -e "\033[1;33m[images][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[images][错误]\033[0m $*" >&2; exit 1; }

# ── 参数解析 ────────────────────────────────────────────────────────────
# v2 默认：监控栈也含；--infra-only 关闭监控栈；--with-monitoring 兼容老调用。
WITH_MONITORING=1
MONITORING_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version)      INFRA_VERSION="$2"; shift 2 ;;
    --arch)            ARCHES="$2"; shift 2 ;;
    --infra-only)      WITH_MONITORING=0; shift ;;
    --with-monitoring) warn "--with-monitoring 已废弃：v2 起默认即含监控栈，本标志为 no-op"
                       WITH_MONITORING=1; shift ;;
    --monitoring-only) MONITORING_ONLY=1; WITH_MONITORING=1; shift ;;
    -h|--help)         sed -n '3,40p' "$0"; exit 0 ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# 仅支持 amd64（见 release.conf 注释 "架构支持"）
for _a in $ARCHES; do
  [ "$_a" = "amd64" ] || die "本工具仅支持 amd64 架构，传入：${_a}。
        如确实需要 arm64：见 release.conf 中关于 架构支持 的注释，
        改 ARCHES + 移除本脚本的校验后自行验证。"
done

# host 架构也必须是 amd64（amd64-only refactor 后已不支持跨架构 pull）
[ "$HOST_ARCH" = "amd64" ] || die "本工具只支持在 amd64 host 上运行（当前 host：${HOST_ARCH}）。
        amd64-only refactor 后已移除跨架构 pull 代码路径；arm64 host 上构建
        amd64 包请自行恢复跨架构逻辑或换 amd64 host 跑。"

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
  xz)   TAR_OPT="-cJf"; EXT="tar.xz"
        # xz 默认单线程，开 -T0 跟随机器核数多线程压缩（tar 会读该环变传给 xz）
        export XZ_OPT="${XZ_OPT:--T0}" ;;
  gzip) TAR_OPT="-czf"; EXT="tar.gz" ;;
  zstd) TAR_OPT="--zstd -cf"; EXT="tar.zst"
        command -v zstd >/dev/null 2>&1 || die "PKG_COMPRESS=zstd 但未安装 zstd" ;;
  *)    die "release.conf 的 PKG_COMPRESS 取值非法：$PKG_COMPRESS（应为 xz/gzip/zstd）" ;;
esac

CACHE="$SCRIPT_DIR/images-cache"
mkdir -p "$CACHE"

# --monitoring-only：基础设施必须已构建过；版本与构建时间沿用既有 manifest
# 无论 INFRA_VERSION 来自 release.conf 默认值还是 -v 参数，最终版本号统一追加
# -YYYYMMDD-HHMM 时间戳，保证镜像 tag 与归档目录永远唯一。--monitoring-only 下
# 从既有 manifest 沉底读回全版本（已含时间戳），避免补包冲出另一个版本目录。
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
    INFRA_VERSION="${INFRA_VERSION}-$(date +%Y%m%d-%H%M)"
    BUILT_AT="$(date -Is)"
  fi
else
  INFRA_VERSION="${INFRA_VERSION}-$(date +%Y%m%d-%H%M)"
  BUILT_AT="$(date -Is)"
fi

if [ "$MONITORING_ONLY" = 1 ]; then
  log "模式：仅补监控栈（基础设施 infra-images-*.tar 保持不动）"
fi
log "本机架构：$HOST_ARCH   构建架构：$ARCHES   监控栈：$([ "$WITH_MONITORING" = 1 ] && echo 含 || echo 不含)"
log "基础设施版本：$INFRA_VERSION"

# ── 拉取并打 *-saved tag（amd64-only）────────────────────────────────────
# 命名约定：<image>-amd64-saved 作为本地缓存 tag，命中即跳 docker pull。
# （-amd64- 段保留作历史缓存兼容；不再有跨架构需求，amd64-only 已在入口断言）
prepare_image() {
  local IMG="$1" ARCH="$2"
  local SAVED_TAG="${IMG}-${ARCH}-saved"
  if docker image inspect "$SAVED_TAG" >/dev/null 2>&1; then
    log "✓ 复用本地缓存: $SAVED_TAG"
    return 0
  fi
  # The exact configured reference may already have been loaded from an
  # earlier package or a registry mirror. Add the cache tag without pulling;
  # never substitute a different repository or version implicitly.
  if docker image inspect "$IMG" >/dev/null 2>&1; then
    docker tag "$IMG" "$SAVED_TAG"
    log "✓ 复用本地镜像并补缓存标签: $IMG → $SAVED_TAG"
    return 0
  fi
  docker pull --platform "linux/$ARCH" "$IMG"
  docker tag "$IMG" "$SAVED_TAG"
  log "✓ 已拉取并打标: $IMG → $SAVED_TAG"
}

# 双 tag 导出：tar 内同时包含 *-saved 与原 tag（运维侧 docker load 后直接用原 tag）。
# 实现：save 前把原 tag 指向 *-saved；amd64-only 后 host 与目标架构同源，无需再做
# "跨架构 pull 修复原 tag" 的二次拉取。
save_with_dual_tags() {
  local OUT_TAR="$1" ARCH="$2"; shift 2
  local IMGS=("$@") REFS=() IMG SAVED_TAG
  for IMG in "${IMGS[@]}"; do
    SAVED_TAG="${IMG}-${ARCH}-saved"
    docker tag "$SAVED_TAG" "$IMG"
    REFS+=( "$SAVED_TAG" "$IMG" )
  done
  docker save -o "$OUT_TAR" "${REFS[@]}"
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
  # setup-mirrors.sh 放在 infra 包顶层（Docker / npm / Golang 三合一加速，非 Docker 专属）
  # install-docker.sh 末尾会调用 ../setup-mirrors.sh 引导用户配置
  cp "$SCRIPT_DIR/bundle/setup-mirrors.sh"          "$STAGE/"
  cp "$SCRIPT_DIR/bundle/docker/install-docker.sh"  "$STAGE/docker/"
  cp "$REPO_ROOT/deployments/docker/docker-network-lib.sh" "$STAGE/docker/docker-network-lib.sh"
  if ls "$DOCKER_CACHE/$ARCH"/docker-*.tgz >/dev/null 2>&1; then
    cp "$DOCKER_CACHE/$ARCH"/docker-*.tgz "$STAGE/docker/"
    DOCKER_IN_PKG="${DOCKER_VERSION:-未知}"
  else
    warn "[$ARCH] docker-cache/$ARCH/ 无 Docker 引擎包 —— 基础设施包不含离线装 Docker 的包。"
    warn "        若目标机未装 Docker，请先运行： ./download-docker.sh"
    DOCKER_IN_PKG="未含"
  fi

  # Docker Compose v2 二进制（仅名 docker-compose，install-docker.sh 同目录检测即装）
  # ln -sfn 产出的软链指向同目录实体，cp -L 解引用，避免 tar 包内被当成金中软链
  if [ -f "$DOCKER_CACHE/$ARCH/docker-compose" ]; then
    cp -L "$DOCKER_CACHE/$ARCH/docker-compose" "$STAGE/docker/docker-compose"
    chmod +x "$STAGE/docker/docker-compose"
    COMPOSE_IN_PKG="${COMPOSE_VERSION:-未知}"
  else
    warn "[$ARCH] docker-cache/$ARCH/ 无 docker-compose 二进制 —— 交付包不含离线 compose v2。"
    warn "        请先运行： ./download-docker.sh（同时会下载 compose）"
    COMPOSE_IN_PKG="未含"
  fi

  # Docker Buildx v0.x 二进制（仅名 docker-buildx）——Docker 23+ BuildKit 必需
  if [ -f "$DOCKER_CACHE/$ARCH/docker-buildx" ]; then
    cp -L "$DOCKER_CACHE/$ARCH/docker-buildx" "$STAGE/docker/docker-buildx"
    chmod +x "$STAGE/docker/docker-buildx"
    BUILDX_IN_PKG="${BUILDX_VERSION:-未知}"
  else
    warn "[$ARCH] docker-cache/$ARCH/ 无 docker-buildx 二进制 —— 交付包不含离线 buildx。"
    warn "        请先运行： ./download-docker.sh（同时会下载 buildx）"
    BUILDX_IN_PKG="未含"
  fi

  # VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
infra_version=$INFRA_VERSION
arch=$ARCH
docker_version=$DOCKER_IN_PKG
compose_version=$COMPOSE_IN_PKG
buildx_version=$BUILDX_IN_PKG
build_time=$BUILT_AT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 基础设施包 — $INFRA_VERSION ($ARCH)

- 基础设施版本：$INFRA_VERSION
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- Docker 版本：$DOCKER_IN_PKG
- Compose v2 版本：$COMPOSE_IN_PKG
- Buildx 版本：$BUILDX_IN_PKG
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
  echo "compose_version=${COMPOSE_VERSION:-未含}"
  echo "build_time=$BUILT_AT"
  echo "arches=$ARCHES"
  echo "compress=$PKG_COMPRESS"
  echo ""
  echo "# ── 基础设施包构建说明 ─────────────────────────────────────"
  echo "基础设施版本：$INFRA_VERSION"
  echo "Docker 版本 ：${DOCKER_VERSION:-未含}"
  echo "Compose v2  ：${COMPOSE_VERSION:-未含}"
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

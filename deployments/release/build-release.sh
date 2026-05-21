#!/usr/bin/env bash
# =============================================================================
# OMC 项目交付包构建工具（编译二进制 + 组装 + 压缩 + 版本归档）
#
# 编译 OMC 二进制、构建前端、组装并压缩【项目交付包】，按版本归档到
# archive/project/。【频繁运行 —— 每次发版执行】。
#
# 项目交付包【只含 OMC 本体】：二进制 + 前端 + 配置 + 数据库迁移 + 部署模板。
# Docker 引擎与基础镜像在【独立的基础设施包 omc-infra-*】里（由 build-images.sh
# 生成），两个包各自维护版本号，互不耦合。
#
# 发布渠道（阶段标识）—— 决定交付包文件名前缀：
#   test    → omc-test-<版本>-<架构>.tar.xz     （测试阶段，默认）
#   release → omc-release-<版本>-<架构>.tar.xz  （正式发布）
#   默认取 release.conf 的 RELEASE_CHANNEL，--channel 可临时覆盖。
#
# 项目版本号（纯 semver，与基础设施版本独立）：
#   不带 -v  → 自动生成（小版本）：<RELEASE_BASE_VERSION>-<构建时间戳>
#   带  -v   → 手动指定（大版本）：例如  ./build-release.sh -v 1.0.0
#
# 产物：archive/project/<版本>/omc-<渠道>-<版本>-<架构>.tar.<压缩>（+ .sha256）
#       并自动刷新 archive/index.html，供 serve.sh 起 HTTP 服务下载。
#
# 用法： ./build-release.sh [-v 版本] [--channel test|release] [--arch amd64]
#   --arch  当前【只支持 amd64】，非法值会被拒绝；原因见 release.conf "架构支持" 注释。
#   ★ 用普通用户运行，不要 sudo（sudo 重置 PATH 会找不到 go/npm，产物归 root）。
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=release.conf
source "$SCRIPT_DIR/release.conf"

log()  { echo -e "\033[1;32m[release]\033[0m $*"; }
warn() { echo -e "\033[1;33m[release][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[release][错误]\033[0m $*" >&2; exit 1; }

# ── 参数解析 ────────────────────────────────────────────────────────────
VERSION=""
CHANNEL="${RELEASE_CHANNEL:-test}"
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version) VERSION="$2"; shift 2 ;;
    --channel)    CHANNEL="$2"; shift 2 ;;
    --arch)       ARCHES="$2"; shift 2 ;;
    -h|--help)    sed -n '3,26p' "$0"; exit 0 ;;
    *)            die "未知参数：$1（-h 查看用法）" ;;
  esac
done

case "$CHANNEL" in
  test|release) ;;
  *) die "--channel 取值非法：$CHANNEL（应为 test 或 release）" ;;
esac

# 仅支持 amd64（见 release.conf 注释 "架构支持"）
for _a in $ARCHES; do
  [ "$_a" = "amd64" ] || die "本工具仅支持 amd64 架构，传入：${_a}。
        如确实需要 arm64：见 release.conf 中关于 架构支持 的注释,
        改 ARCHES + 移除本脚本的校验后自行验证。"
done

# ── 前置检查 ────────────────────────────────────────────────────────────
# 本工具是构建脚本，应以【普通用户】运行，不要 sudo。本工具【不需要 docker】。
if [ -n "${SUDO_USER:-}" ]; then
  warn "检测到通过 sudo 运行——sudo 重置了 PATH，可能找不到 go/npm，且产物会归 root。"
  warn "强烈建议改用普通用户直接执行： ./build-release.sh"
fi
# v3 起【全 docker compose 部署】：业务镜像（app/acs/worker/web）由 docker build
# 直接产出，docker save 后随项目交付包发布。host 端不再编译 Go 二进制，也不再本
# 地 npm build；构建机仅需 docker + 打包工具。
for tool in docker tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少构建工具：$tool"
done
if ! docker info >/dev/null 2>&1; then
  die "docker 不可用：当前用户可能不在 docker 组。请执行：
      sudo usermod -aG docker \$USER && newgrp docker   （或重新登录）
      然后重新运行本脚本。"
fi
[ -n "${PROJECT_IMAGE_PREFIX:-}" ] || die "release.conf 未配置 PROJECT_IMAGE_PREFIX"
[ "${#BUSINESS_IMAGES[@]:-0}" -gt 0 ] || die "release.conf 未配置 BUSINESS_IMAGES"

# 压缩方式 → tar 选项与扩展名
case "$PKG_COMPRESS" in
  xz)   TAR_OPT="-cJf"; EXT="tar.xz" ;;
  gzip) TAR_OPT="-czf"; EXT="tar.gz" ;;
  zstd) TAR_OPT="--zstd -cf"; EXT="tar.zst"
        command -v zstd >/dev/null 2>&1 || die "PKG_COMPRESS=zstd 但未安装 zstd" ;;
  *)    die "release.conf 的 PKG_COMPRESS 取值非法：$PKG_COMPRESS（应为 xz/gzip/zstd）" ;;
esac

# ── 项目版本号解析 ──────────────────────────────────────────────────────
if [ -n "$VERSION" ]; then
  log "项目版本【手动指定 / 大版本】：$VERSION"
else
  VERSION="${RELEASE_BASE_VERSION}-$(date +%Y%m%d-%H%M)"
  log "项目版本【自动生成 / 小版本】：$VERSION"
fi
GIT_COMMIT="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo n/a)"

ARCHIVE="$SCRIPT_DIR/archive"
OUT="$ARCHIVE/project/$VERSION"
WORK="$SCRIPT_DIR/dist/.work"
rm -rf "$WORK"
mkdir -p "$WORK" "$OUT"

log "发布渠道：$CHANNEL   压缩方式：$PKG_COMPRESS   目标架构：$ARCHES"
log "业务镜像前缀：$PROJECT_IMAGE_PREFIX   业务镜像：${BUSINESS_IMAGES[*]}"

# ── 1. 逐架构构建业务镜像 + 组装 + 压缩归档 ───────────────────────────────
# v3 起所有业务进程（app / acs / worker / web）都在容器内运行；前端和 Go 二进制
# 均由对应 Dockerfile 的多阶段构建产出，host 端不再单独编译。
# 业务镜像 tag 规则：<PROJECT_IMAGE_PREFIX>/<svc>:<project_version>，docker save
# 后随交付包发布；目标机 docker load 即用，运维侧不再有 host 二进制依赖。
for ARCH in $ARCHES; do
  log "=================== 架构 $ARCH ==================="
  PKG_NAME="omc-$CHANNEL-$VERSION-$ARCH"
  STAGE="$WORK/$PKG_NAME"
  mkdir -p "$STAGE"/{etc,docs,images}

  # 1.1 构建业务镜像（amd64-only：host 已在入口断言为 amd64，docker build 默认
  # 按 host 架构产出，无需 --platform；arm64 host 上自行恢复 buildx 跨架构逻辑）
  log "[$ARCH] docker build 业务镜像（${BUSINESS_IMAGES[*]}）..."
  IMG_REFS=()
  for SVC in "${BUSINESS_IMAGES[@]}"; do
    DOCKERFILE="$REPO_ROOT/deployments/docker/Dockerfile.$SVC"
    [ -f "$DOCKERFILE" ] || die "缺 Dockerfile：$DOCKERFILE"
    TAG="$PROJECT_IMAGE_PREFIX/$SVC:$VERSION"
    log "[$ARCH]   docker build -t $TAG -f $DOCKERFILE"
    ( cd "$REPO_ROOT" && docker build \
        -t "$TAG" \
        -f "$DOCKERFILE" \
        --build-arg APK_MIRROR=mirrors.aliyun.com \
        . )
    IMG_REFS+=( "$TAG" )
  done

  # 1.2 docker save 业务镜像到 images/business-images-<version>-<arch>.tar
  log "[$ARCH] docker save → images/business-images-$VERSION-$ARCH.tar"
  docker save -o "$STAGE/images/business-images-$VERSION-$ARCH.tar" "${IMG_REFS[@]}"

  # 1.3 配置 / 迁移 / 字典 / Casbin（可挂载到容器覆盖镜像内默认值）
  log "[$ARCH] 收集字典、迁移、配置 ..."
  cp -r "$REPO_ROOT/omcgo/data"         "$STAGE/data"
  cp -r "$REPO_ROOT/omcgo/configs"      "$STAGE/configs"
  cp -r "$REPO_ROOT/omcgo/migrations"   "$STAGE/migrations"
  cp "$REPO_ROOT/omcgo/cmd/app/etc/config.prod.yaml"    "$STAGE/etc/app.prod.yaml"
  cp "$REPO_ROOT/omcgo/cmd/acs/etc/config.prod.yaml"    "$STAGE/etc/acs.prod.yaml"
  cp "$REPO_ROOT/omcgo/cmd/worker/etc/config.prod.yaml" "$STAGE/etc/worker.prod.yaml"

  # 1.4 部署模板 + compose 文件 + nginx 配置 + 监控栈配置
  log "[$ARCH] 拷入部署模板 + 监控栈配置 ..."
  cp -r "$SCRIPT_DIR/bundle/deploy"       "$STAGE/deploy"
  cp -r "$REPO_ROOT/deployments/monitoring" "$STAGE/deploy/monitoring"
  cp "$REPO_ROOT/deployments/docker/nginx.conf"   "$STAGE/deploy/nginx.conf"
  cp "$REPO_ROOT/deployments/docker/default.conf" "$STAGE/deploy/default.conf"
  cat > "$STAGE/deploy/.env" <<EOF
# 项目版本（业务镜像 tag 取自此处）
PROJECT_VERSION=$VERSION
IMAGE_PREFIX=$PROJECT_IMAGE_PREFIX
# 业务镜像
IMAGE_APP=$PROJECT_IMAGE_PREFIX/app:$VERSION
IMAGE_ACS=$PROJECT_IMAGE_PREFIX/acs:$VERSION
IMAGE_WORKER=$PROJECT_IMAGE_PREFIX/worker:$VERSION
IMAGE_WEB=$PROJECT_IMAGE_PREFIX/web:$VERSION
# 基础设施镜像
IMAGE_POSTGRES=$IMAGE_POSTGRES
IMAGE_REDIS=$IMAGE_REDIS
IMAGE_NATS=$IMAGE_NATS
IMAGE_MINIO=$IMAGE_MINIO
IMAGE_NGINX=$IMAGE_NGINX
# 监控栈镜像
IMAGE_PROMETHEUS=$IMAGE_PROMETHEUS
IMAGE_ALERTMANAGER=$IMAGE_ALERTMANAGER
IMAGE_GRAFANA=$IMAGE_GRAFANA
IMAGE_LOKI=$IMAGE_LOKI
IMAGE_TEMPO=$IMAGE_TEMPO
IMAGE_OTELCOL=$IMAGE_OTELCOL
IMAGE_NATS_EXPORTER=$IMAGE_NATS_EXPORTER
# 数据库 / 对象存储 / Grafana 默认口令——【部署前必改为强口令】
# 与 /opt/omc/etc/*.prod.yaml 中的 dsn / minio.access_key / minio.secret_key 保持一致
POSTGRES_USER=omcgo
POSTGRES_PASSWORD=omcgo123
POSTGRES_DB=omcgo
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin
# OMC 运行环境（容器内 entrypoint.sh 读）
OMCGO_ENV=prod
# JWT 密钥（app 容器读，生产勿用默认值）
OMCGO_JWT_SECRET=8f7a9b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6
EOF

  # 1.5 运维侧部署文档
  cp "$REPO_ROOT/docs/operations/OMC内网离线部署手册（运维侧）.md" "$STAGE/docs/"

  # 1.6 VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
project_version=$VERSION
channel=$CHANNEL
arch=$ARCH
image_prefix=$PROJECT_IMAGE_PREFIX
business_images=${BUSINESS_IMAGES[*]}
build_time=$(date -Is)
git_commit=$GIT_COMMIT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 项目交付包 — $VERSION ($ARCH)

- 项目版本：$VERSION
- 发布渠道：$CHANNEL（test=测试阶段 / release=正式发布）
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- 业务镜像：${BUSINESS_IMAGES[*]/#/$PROJECT_IMAGE_PREFIX/}（tag = $VERSION）
- 构建时间：$(date -Is)　git commit：$GIT_COMMIT

本包【全 docker compose 部署】，含业务镜像 tar（docker save）+ compose
文件 + 配置模板 + 迁移 / 字典 / Casbin（可挂载覆盖）+ 监控栈配置 + 运维脚本。
Docker 引擎、compose v2 二进制、基础镜像在【独立的基础设施包 omc-infra-*】里——
首次部署需先用基础设施包装好 Docker / Compose、导入基础镜像，再部署本包。

部署步骤见 \`docs/OMC内网离线部署手册（运维侧）.md\`。先校验完整性：
\`\`\`bash
sha256sum -c checksums.sha256
\`\`\`
PostgreSQL / MinIO / JWT / Grafana 默认口令必须在部署时修改（deploy/.env）。
EOF
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 1.7 压缩打包 → archive/project/<版本>/
  log "[$ARCH] 压缩打包（$PKG_COMPRESS）..."
  # shellcheck disable=SC2086
  ( cd "$WORK" && tar $TAR_OPT "$OUT/$PKG_NAME.$EXT" "$PKG_NAME" )
  ( cd "$OUT" && sha256sum "$PKG_NAME.$EXT" > "$PKG_NAME.$EXT.sha256" )
  log "[$ARCH] 产出：archive/project/$VERSION/$PKG_NAME.$EXT"
done

rm -rf "$WORK"

# ── 3. 版本构建说明 RELEASE.txt ──────────────────────────────────────────
{
  echo "project_version=$VERSION"
  echo "channel=$CHANNEL"
  echo "build_time=$(date -Is)"
  echo "git_commit=$GIT_COMMIT"
  echo "arches=$ARCHES"
  echo "compress=$PKG_COMPRESS"
  echo "image_prefix=$PROJECT_IMAGE_PREFIX"
  echo "business_images=${BUSINESS_IMAGES[*]}"
  echo ""
  echo "# ── 版本构建说明 ───────────────────────────────────────────"
  echo "项目版本：$VERSION"
  echo "发布渠道：$CHANNEL"
  echo "git commit：$GIT_COMMIT"
  echo "业务镜像前缀：$PROJECT_IMAGE_PREFIX  （tag = $VERSION）"
  echo "业务服务：${BUSINESS_IMAGES[*]}"
  echo ""
  echo "说明：本包【全 docker compose 部署】，含业务镜像 tar + compose 文件 + 配置。"
  echo "      Docker 引擎、compose v2、基础镜像在独立的基础设施包 omc-infra-* 里，首次部署需配合使用。"
  echo ""
  echo "交付文件："
  ( cd "$OUT" && ls -1 omc-*."$EXT" )
} > "$OUT/RELEASE.txt"

# ── 3. 刷新 archive/index.html（HTTP 下载索引）───────────────────────────
"$SCRIPT_DIR/gen-index.sh"
log "已刷新下载索引：archive/index.html"

# ── 完成 ────────────────────────────────────────────────────────────────
echo
log "全部完成。"
log "  项目版本 ：$VERSION"
log "  发布渠道 ：$CHANNEL"
log "  归档位置 ：archive/project/$VERSION/"
ls -lh "$OUT"/omc-*."$EXT" 2>/dev/null || true
echo
log "起 HTTP 下载服务： ./serve.sh    然后浏览器访问 http://<构建机IP>:8000/"
log "基础设施包（Docker 引擎 + 基础镜像）由 ./build-images.sh 生成，与本包独立。"

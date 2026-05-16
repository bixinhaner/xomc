#!/usr/bin/env bash
# =============================================================================
# OMC 交付包构建工具（编译二进制 + 组装交付包）
#
# 编译 OMC 二进制、构建前端、组装离线交付包。【频繁运行 —— 每次发版执行】。
#
# 基础设施 Docker 镜像【不在本工具内拉取】——由独立工具 build-images.sh 预先
# 导出到 images-cache/，本工具直接复用。基础设施不常变，无需每次发版重拉。
#
# 版本号：
#   不带 -v  → 自动生成（小版本）：<RELEASE_BASE_VERSION>-<构建时间戳>
#   带  -v   → 手动指定（大版本）：例如  ./build-release.sh -v 2.0.0
#
# 用法： ./build-release.sh [-v 版本] [--arch amd64|arm64]
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
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version) VERSION="$2"; shift 2 ;;
    --arch)       ARCHES="$2"; shift 2 ;;
    -h|--help)    sed -n '3,17p' "$0"; exit 0 ;;
    *)            die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── 前置检查 ────────────────────────────────────────────────────────────
# 本工具是构建脚本，应以【普通用户】运行，不要 sudo：sudo 会重置 PATH
# 导致找不到 go/npm，并让产物归 root。本工具【不需要 docker】（不碰镜像）。
if [ -n "${SUDO_USER:-}" ]; then
  warn "检测到通过 sudo 运行——sudo 重置了 PATH，可能找不到 go/npm，且产物会归 root。"
  warn "强烈建议改用普通用户直接执行： ./build-release.sh"
fi
for tool in go npm tar sha256sum; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    if [ "$tool" = go ] || [ "$tool" = npm ]; then
      die "缺少构建工具：$tool。
      若 \`$tool version\` 在你的 shell 里能正常运行，多半是用了 sudo——sudo 重置了
      PATH 导致找不到。请【用普通用户直接运行】本脚本（不要 sudo）。"
    fi
    die "缺少构建工具：$tool"
  fi
done

# 基础设施镜像缓存必须已由 build-images.sh 生成
CACHE="$SCRIPT_DIR/images-cache"
MISSING=""
for ARCH in $ARCHES; do
  [ -f "$CACHE/infra-images-$ARCH.tar" ] || MISSING="$MISSING $ARCH"
done
if [ -n "$MISSING" ]; then
  die "缺少基础设施镜像缓存（架构：$MISSING）。
      基础设施镜像由独立工具生成，请先运行一次：
          ./build-images.sh
      基础设施镜像不常变更，生成一次后可被反复复用，无需每次发版重跑。"
fi

# ── 版本号解析 ──────────────────────────────────────────────────────────
if [ -n "$VERSION" ]; then
  log "版本【手动指定 / 大版本】：$VERSION"
else
  VERSION="${RELEASE_BASE_VERSION}-$(date +%Y%m%d-%H%M)"
  log "版本【自动生成 / 小版本】：$VERSION  （基线 $RELEASE_BASE_VERSION，发大版本用 -v 覆盖）"
fi
GIT_COMMIT="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo n/a)"

DIST="$SCRIPT_DIR/dist"
WORK="$DIST/.work"
rm -rf "$WORK"
mkdir -p "$WORK" "$DIST"

log "架构：$ARCHES"

# ── 1. 构建前端（架构无关，只做一次）──────────────────────────────────────
log "构建前端 omcmb/webcode ..."
(
  cd "$REPO_ROOT/omcmb/webcode"
  npm ci --include=dev --legacy-peer-deps
  npm run build
)
[ -f "$REPO_ROOT/omcmb/webcode/dist/index.html" ] || die "前端构建产物缺失"

# ── 2. 逐架构组装 ────────────────────────────────────────────────────────
for ARCH in $ARCHES; do
  log "=================== 架构 $ARCH ==================="
  PKG_NAME="omc-release-$VERSION-$ARCH"
  STAGE="$WORK/$PKG_NAME"
  mkdir -p "$STAGE"/{bin,web,etc,images,docs}

  # 2.1 交叉编译 Go 二进制（静态）
  log "[$ARCH] 编译 Go 二进制 ..."
  (
    cd "$REPO_ROOT/omcgo"
    for SVC in app acs worker migrate seed; do
      CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
        go build -ldflags="-s -w" -o "$STAGE/bin/omcgo-$SVC" "./cmd/$SVC"
    done
    CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
      go build -ldflags="-s -w" -o "$STAGE/bin/omcctl" "./cmd/omcctl"
  )

  # 2.2 前端 / 字典 / Casbin / 迁移 / 配置模板
  log "[$ARCH] 收集前端、字典、迁移、配置 ..."
  cp -r "$REPO_ROOT/omcmb/webcode/dist" "$STAGE/web/dist"
  cp -r "$REPO_ROOT/omcgo/data"         "$STAGE/data"
  cp -r "$REPO_ROOT/omcgo/configs"      "$STAGE/configs"
  cp -r "$REPO_ROOT/omcgo/migrations"   "$STAGE/migrations"
  cp "$REPO_ROOT/omcgo/cmd/app/etc/config.prod.yaml"    "$STAGE/etc/app.prod.yaml"
  cp "$REPO_ROOT/omcgo/cmd/acs/etc/config.prod.yaml"    "$STAGE/etc/acs.prod.yaml"
  cp "$REPO_ROOT/omcgo/cmd/worker/etc/config.prod.yaml" "$STAGE/etc/worker.prod.yaml"

  # 2.3 部署模板（compose / systemd / 脚本）+ Docker 离线安装件 + nginx 配置
  log "[$ARCH] 拷入部署模板 ..."
  cp -r "$SCRIPT_DIR/bundle/deploy"  "$STAGE/deploy"
  cp -r "$SCRIPT_DIR/bundle/docker"  "$STAGE/docker"
  cp "$REPO_ROOT/deployments/docker/nginx.conf"   "$STAGE/deploy/nginx.conf"
  cp "$REPO_ROOT/deployments/docker/default.conf" "$STAGE/deploy/default.conf"
  cat > "$STAGE/deploy/.env" <<EOF
IMAGE_POSTGRES=$IMAGE_POSTGRES
IMAGE_REDIS=$IMAGE_REDIS
IMAGE_NATS=$IMAGE_NATS
IMAGE_MINIO=$IMAGE_MINIO
IMAGE_NGINX=$IMAGE_NGINX
EOF
  if ! ls "$STAGE"/docker/docker-*.tgz >/dev/null 2>&1; then
    warn "[$ARCH] bundle/docker/ 内未放置 docker-*.tgz —— 交付包将不含 Docker 引擎离线安装件。"
    warn "        若目标机已装 Docker 可忽略；否则请按 bundle/docker/README.md 放入。"
  fi

  # 2.4 运维侧部署文档
  cp "$REPO_ROOT/docs/operations/OMC内网离线部署手册（运维侧）.md" "$STAGE/docs/"

  # 2.5 基础设施镜像 —— 从 images-cache 复用（由 build-images.sh 预先生成）
  log "[$ARCH] 复用 images-cache 的基础设施镜像 ..."
  cp "$CACHE/infra-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/monitoring-images-$ARCH.tar" ] && \
    cp "$CACHE/monitoring-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/images.manifest" ] && cp "$CACHE/images.manifest" "$STAGE/images/"

  # 2.6 VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
version=$VERSION
arch=$ARCH
build_time=$(date -Is)
git_commit=$GIT_COMMIT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 离线交付包 — $VERSION ($ARCH)

- 版本：$VERSION
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- 构建时间：$(date -Is)
- git commit：$GIT_COMMIT

## 部署

完整步骤见 \`docs/OMC内网离线部署手册（运维侧）.md\`。先校验完整性：

\`\`\`bash
sha256sum -c checksums.sha256
\`\`\`

PostgreSQL / MinIO / JWT / Grafana 的默认口令必须在部署时修改（见运维侧手册 §5 步骤 4）。
EOF
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 2.7 打包 + 校验和
  log "[$ARCH] 打包 ..."
  ( cd "$WORK" && tar czf "$DIST/$PKG_NAME.tar.gz" "$PKG_NAME" )
  ( cd "$DIST" && sha256sum "$PKG_NAME.tar.gz" > "$PKG_NAME.tar.gz.sha256" )
  log "[$ARCH] 产出：dist/$PKG_NAME.tar.gz"
done

rm -rf "$WORK"
log "全部完成。交付包位于：$DIST/"
ls -lh "$DIST"/*.tar.gz 2>/dev/null || true

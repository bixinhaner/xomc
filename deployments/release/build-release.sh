#!/usr/bin/env bash
# =============================================================================
# OMC 离线交付包生成工具
#
# 在【构建侧】（有公网、装有 Go 1.25+ / Node 20+ / Docker）执行，产出可送入
# 内网的自包含离线交付包。详见 docs/operations/OMC离线交付包构建手册（构建侧）.md。
#
# 用法：
#   ./build-release.sh [-v 版本] [--arch amd64|arm64] [--with-monitoring]
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
WITH_MONITORING=0
usage() {
  sed -n '3,12p' "$0"
  exit "${1:-0}"
}
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version)      VERSION="$2"; shift 2 ;;
    --arch)            ARCHES="$2"; shift 2 ;;
    --with-monitoring) WITH_MONITORING=1; shift ;;
    -h|--help)         usage 0 ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── 前置检查 ────────────────────────────────────────────────────────────
for tool in go npm docker tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少构建工具：$tool"
done

[ -n "$VERSION" ] || VERSION="$(cd "$REPO_ROOT" && git describe --tags --always 2>/dev/null || echo dev)"
GIT_COMMIT="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo n/a)"

DIST="$SCRIPT_DIR/dist"
WORK="$DIST/.work"
rm -rf "$WORK"
mkdir -p "$WORK" "$DIST"

log "版本：$VERSION  架构：$ARCHES  监控栈：$([ "$WITH_MONITORING" = 1 ] && echo 含 || echo 不含)"

# ── 1. 构建前端（架构无关，只做一次）──────────────────────────────────────
log "构建前端 omcmb/webcode ..."
(
  cd "$REPO_ROOT/omcmb/webcode"
  npm ci --include=dev --legacy-peer-deps
  npm run build
)
[ -f "$REPO_ROOT/omcmb/webcode/dist/index.html" ] || die "前端构建产物缺失"

# ── 2. 逐架构构建 ────────────────────────────────────────────────────────
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
  # 生成 deploy/.env —— docker-compose 引用的镜像标签，与 release.conf 单一来源一致
  cat > "$STAGE/deploy/.env" <<EOF
IMAGE_POSTGRES=$IMAGE_POSTGRES
IMAGE_REDIS=$IMAGE_REDIS
IMAGE_NATS=$IMAGE_NATS
IMAGE_MINIO=$IMAGE_MINIO
IMAGE_NGINX=$IMAGE_NGINX
EOF
  # Docker 静态二进制包是否就位（运维侧离线装 Docker 用）
  if ! ls "$STAGE"/docker/docker-*.tgz >/dev/null 2>&1; then
    warn "[$ARCH] bundle/docker/ 内未放置 docker-*.tgz —— 交付包将不含 Docker 引擎离线安装件。"
    warn "        若目标机已装 Docker 可忽略；否则请按 bundle/docker/README.md 放入对应架构的包。"
  fi

  # 2.4 运维侧部署文档（仅运维手册随交付包走；构建手册留在仓库）
  cp "$REPO_ROOT/docs/operations/OMC内网离线部署手册（运维侧）.md" "$STAGE/docs/"

  # 2.5 Docker 镜像（按架构 pull + save）
  #
  # 镜像导出【不依赖构建机自身架构】：下面用 `docker pull --platform linux/$ARCH`
  # 显式逐架构拉取（拉取只下载分层、不执行，跨架构可行）。
  #
  # 但 `docker save` 读的是构建机【本地 docker 镜像库】，会受环境影响：
  # 构建机遗留的同名不同架构缓存、或 containerd 镜像库一个 tag 混存多架构，
  # 都可能让 save 导出错架构。故每个架构在 pull 前先 `docker rmi` 清本地同名
  # 镜像，保证 `docker save` 导出的就是本架构（部署方案 §4.4）。
  IMG_LIST=( "${INFRA_IMAGES[@]}" )
  [ "$WITH_MONITORING" = 1 ] && IMG_LIST+=( "${MONITORING_IMAGES[@]}" )
  log "[$ARCH] 拉取并导出 ${#IMG_LIST[@]} 个 Docker 镜像 ..."
  for IMG in "${IMG_LIST[@]}"; do
    docker rmi -f "$IMG" >/dev/null 2>&1 || true   # 清本地缓存，使 save 架构确定
    docker pull --platform "linux/$ARCH" "$IMG"
  done
  docker save -o "$STAGE/images/infra-images-$ARCH.tar" "${INFRA_IMAGES[@]}"
  if [ "$WITH_MONITORING" = 1 ]; then
    docker save -o "$STAGE/images/monitoring-images-$ARCH.tar" "${MONITORING_IMAGES[@]}"
  fi
  printf '%s\n' "${IMG_LIST[@]}" > "$STAGE/images/images.manifest"

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

初始管理员账号 / 默认密码须在首次登录后立即修改；
PostgreSQL / MinIO / JWT / Grafana 的默认口令必须在部署时改掉（见部署方案 §5 步骤 4）。
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

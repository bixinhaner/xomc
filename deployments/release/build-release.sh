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
# 用法： ./build-release.sh [-v 版本] [--channel test|release] [--arch amd64|arm64]
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

# ── 前置检查 ────────────────────────────────────────────────────────────
# 本工具是构建脚本，应以【普通用户】运行，不要 sudo。本工具【不需要 docker】。
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

# ── 1. 构建前端（架构无关，只做一次）──────────────────────────────────────
log "构建前端 omcmb/webcode ..."
(
  cd "$REPO_ROOT/omcmb/webcode"
  npm ci --include=dev --legacy-peer-deps
  npm run build
)
[ -f "$REPO_ROOT/omcmb/webcode/dist/index.html" ] || die "前端构建产物缺失"

# ── 2. 逐架构组装 + 压缩归档 ─────────────────────────────────────────────
for ARCH in $ARCHES; do
  log "=================== 架构 $ARCH ==================="
  PKG_NAME="omc-$CHANNEL-$VERSION-$ARCH"
  STAGE="$WORK/$PKG_NAME"
  mkdir -p "$STAGE"/{bin,web,etc,docs}

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

  # 2.3 部署模板 + nginx 配置（Docker 引擎与基础镜像在独立的基础设施包里）
  log "[$ARCH] 拷入部署模板 ..."
  cp -r "$SCRIPT_DIR/bundle/deploy" "$STAGE/deploy"
  cp "$REPO_ROOT/deployments/docker/nginx.conf"   "$STAGE/deploy/nginx.conf"
  cp "$REPO_ROOT/deployments/docker/default.conf" "$STAGE/deploy/default.conf"
  cat > "$STAGE/deploy/.env" <<EOF
IMAGE_POSTGRES=$IMAGE_POSTGRES
IMAGE_REDIS=$IMAGE_REDIS
IMAGE_NATS=$IMAGE_NATS
IMAGE_MINIO=$IMAGE_MINIO
IMAGE_NGINX=$IMAGE_NGINX
EOF

  # 2.4 运维侧部署文档
  cp "$REPO_ROOT/docs/operations/OMC内网离线部署手册（运维侧）.md" "$STAGE/docs/"

  # 2.5 VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
project_version=$VERSION
channel=$CHANNEL
arch=$ARCH
build_time=$(date -Is)
git_commit=$GIT_COMMIT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 项目交付包 — $VERSION ($ARCH)

- 项目版本：$VERSION
- 发布渠道：$CHANNEL（test=测试阶段 / release=正式发布）
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- 构建时间：$(date -Is)　git commit：$GIT_COMMIT

本包【只含 OMC 本体】（二进制 + 前端 + 配置 + 数据库迁移 + 部署模板）。
Docker 引擎与基础镜像在【独立的基础设施包 omc-infra-*】里——首次部署需先用
基础设施包装好 Docker、导入基础镜像，再部署本包。

部署步骤见 \`docs/OMC内网离线部署手册（运维侧）.md\`。先校验完整性：
\`\`\`bash
sha256sum -c checksums.sha256
\`\`\`
PostgreSQL / MinIO / JWT / Grafana 默认口令必须在部署时修改。
EOF
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 2.6 压缩打包 → archive/project/<版本>/
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
  echo ""
  echo "# ── 版本构建说明 ───────────────────────────────────────────"
  echo "项目版本：$VERSION"
  echo "发布渠道：$CHANNEL"
  echo "git commit：$GIT_COMMIT"
  echo ""
  echo "说明：本包只含 OMC 本体；Docker 引擎与基础镜像在独立的基础设施包"
  echo "      omc-infra-* 里，首次部署需配合使用。"
  echo ""
  echo "交付文件："
  ( cd "$OUT" && ls -1 omc-*."$EXT" )
} > "$OUT/RELEASE.txt"

# ── 4. 刷新 archive/index.html（HTTP 下载索引）───────────────────────────
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

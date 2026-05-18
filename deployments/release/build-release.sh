#!/usr/bin/env bash
# =============================================================================
# OMC 交付包构建工具（编译二进制 + 组装 + 压缩 + 版本归档）
#
# 编译 OMC 二进制、构建前端、组装并压缩离线交付包，按版本归档到 archive/。
# 【频繁运行 —— 每次发版执行】。
#
# 基础设施 Docker 镜像由独立工具 build-images.sh 预先导出到 images-cache/，
# 本工具直接复用，不重复拉取。
#
# 版本号（项目版本，与基础设施版本独立）：
#   不带 -v  → 自动生成（小版本）：<RELEASE_BASE_VERSION>-<构建时间戳>
#   带  -v   → 手动指定（大版本）：例如  ./build-release.sh -v 2.0.0
#
# 产物：archive/<项目版本>/omc-release-<版本>-<架构>.tar.<压缩> （+ .sha256）
#       并自动刷新 archive/index.html，供 serve.sh 起 HTTP 服务下载。
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
    -h|--help)    sed -n '3,24p' "$0"; exit 0 ;;
    *)            die "未知参数：$1（-h 查看用法）" ;;
  esac
done

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

# 基础设施镜像缓存必须已由 build-images.sh 生成
CACHE="$SCRIPT_DIR/images-cache"
MISSING=""
for ARCH in $ARCHES; do
  [ -f "$CACHE/infra-images-$ARCH.tar" ] || MISSING="$MISSING $ARCH"
done
if [ -n "$MISSING" ]; then
  die "缺少基础设施镜像缓存（架构：$MISSING）。
      请先运行一次：  ./build-images.sh
      基础设施镜像不常变更，生成一次后可反复复用。"
fi

# 从镜像缓存读取基础设施版本（build-images.sh 写入）
INFRA_VER="$(grep -E '^infra_version=' "$CACHE/images.manifest" 2>/dev/null | cut -d= -f2 || true)"
INFRA_BUILT="$(grep -E '^built_at=' "$CACHE/images.manifest" 2>/dev/null | cut -d= -f2 || true)"
[ -n "$INFRA_VER" ] || { INFRA_VER="unknown"; warn "images-cache 未含基础设施版本，建议重跑 build-images.sh。"; }

# ── 项目版本号解析 ──────────────────────────────────────────────────────
if [ -n "$VERSION" ]; then
  log "项目版本【手动指定 / 大版本】：$VERSION"
else
  VERSION="${RELEASE_BASE_VERSION}-$(date +%Y%m%d-%H%M)"
  log "项目版本【自动生成 / 小版本】：$VERSION"
fi
GIT_COMMIT="$(cd "$REPO_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo n/a)"

ARCHIVE="$SCRIPT_DIR/archive"
OUT="$ARCHIVE/$VERSION"
WORK="$SCRIPT_DIR/dist/.work"
rm -rf "$WORK"
mkdir -p "$WORK" "$OUT"

# ── 基础设施是否更新（对比上一次发布的 infra 版本）──────────────────────
PREV_DIR="$(ls -dt "$ARCHIVE"/*/ 2>/dev/null | grep -v "/$VERSION/" | head -1 || true)"
if [ -n "$PREV_DIR" ] && [ -f "${PREV_DIR}RELEASE.txt" ]; then
  PREV_INFRA="$(grep -E '^infra_version=' "${PREV_DIR}RELEASE.txt" | cut -d= -f2 || true)"
  if [ "$PREV_INFRA" = "$INFRA_VER" ]; then
    INFRA_NOTE="未更新（沿用 $INFRA_VER）"
  else
    INFRA_NOTE="已更新（${PREV_INFRA:-?} → $INFRA_VER）"
  fi
else
  INFRA_NOTE="首次发布（$INFRA_VER）"
fi

log "压缩方式：$PKG_COMPRESS   基础设施：$INFRA_NOTE"

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

  # 2.3 部署模板 + Docker 离线安装件 + nginx 配置
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
  # docker/：install-docker.sh（静态模板）+ 对应架构的 Docker 引擎包
  # （由 download-docker.sh 下载到 docker-cache/<arch>/）
  mkdir -p "$STAGE/docker"
  cp "$SCRIPT_DIR/bundle/docker/install-docker.sh" "$STAGE/docker/"
  if ls "$SCRIPT_DIR/docker-cache/$ARCH"/docker-*.tgz >/dev/null 2>&1; then
    cp "$SCRIPT_DIR/docker-cache/$ARCH"/docker-*.tgz "$STAGE/docker/"
  else
    warn "[$ARCH] docker-cache/$ARCH/ 无 Docker 引擎包 —— 交付包不含离线装 Docker 的包。"
    warn "        若目标机未装 Docker，请先运行： ./download-docker.sh"
  fi

  # 2.4 运维侧部署文档
  cp "$REPO_ROOT/docs/operations/OMC内网离线部署手册（运维侧）.md" "$STAGE/docs/"

  # 2.5 基础设施镜像 —— 从 images-cache 复用
  log "[$ARCH] 复用 images-cache 的基础设施镜像（$INFRA_VER）..."
  cp "$CACHE/infra-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/monitoring-images-$ARCH.tar" ] && \
    cp "$CACHE/monitoring-images-$ARCH.tar" "$STAGE/images/"
  [ -f "$CACHE/images.manifest" ] && cp "$CACHE/images.manifest" "$STAGE/images/"

  # 2.6 VERSION / README / 校验和
  cat > "$STAGE/VERSION" <<EOF
project_version=$VERSION
infra_version=$INFRA_VER
arch=$ARCH
build_time=$(date -Is)
git_commit=$GIT_COMMIT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 离线交付包 — $VERSION ($ARCH)

- 项目版本：$VERSION
- 基础设施版本：$INFRA_VER
- 架构：$ARCH（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- 构建时间：$(date -Is)　git commit：$GIT_COMMIT

部署步骤见 \`docs/OMC内网离线部署手册（运维侧）.md\`。先校验完整性：
\`\`\`bash
sha256sum -c checksums.sha256
\`\`\`
PostgreSQL / MinIO / JWT / Grafana 默认口令必须在部署时修改。
EOF
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 2.7 压缩打包 → archive/<版本>/
  log "[$ARCH] 压缩打包（$PKG_COMPRESS）..."
  # shellcheck disable=SC2086
  ( cd "$WORK" && tar $TAR_OPT "$OUT/$PKG_NAME.$EXT" "$PKG_NAME" )
  ( cd "$OUT" && sha256sum "$PKG_NAME.$EXT" > "$PKG_NAME.$EXT.sha256" )
  log "[$ARCH] 产出：archive/$VERSION/$PKG_NAME.$EXT"
done

rm -rf "$WORK"

# ── 3. 版本构建说明 RELEASE.txt ──────────────────────────────────────────
{
  echo "project_version=$VERSION"
  echo "infra_version=$INFRA_VER"
  echo "build_time=$(date -Is)"
  echo "git_commit=$GIT_COMMIT"
  echo "arches=$ARCHES"
  echo "compress=$PKG_COMPRESS"
  echo ""
  echo "# ── 版本构建说明 ───────────────────────────────────────────"
  echo "项目版本    ：$VERSION"
  echo "基础设施版本：$INFRA_VER（镜像构建于 ${INFRA_BUILT:-未知}）"
  echo "基础设施变更：$INFRA_NOTE"
  echo ""
  echo "基础设施镜像清单："
  printf '  %s\n' "${INFRA_IMAGES[@]}"
  echo ""
  echo "交付文件："
  ( cd "$OUT" && ls -1 omc-release-*."$EXT" )
} > "$OUT/RELEASE.txt"

# ── 4. 刷新 archive/index.html（HTTP 下载索引）───────────────────────────
regen_index() {
  local idx="$ARCHIVE/index.html" row=""
  for d in $(ls -dt "$ARCHIVE"/*/ 2>/dev/null); do
    local v rel pv iv bt
    v="$(basename "$d")"
    rel="$d/RELEASE.txt"
    [ -f "$rel" ] || continue
    pv="$(grep -E '^project_version=' "$rel" | cut -d= -f2)"
    iv="$(grep -E '^infra_version=' "$rel" | cut -d= -f2)"
    bt="$(grep -E '^build_time=' "$rel" | cut -d= -f2)"
    local links=""
    for f in "$d"/omc-release-*.tar.*; do
      [ -f "$f" ] || continue
      case "$f" in *.sha256) continue ;; esac
      local bn; bn="$(basename "$f")"
      local sz; sz="$(du -h "$f" | cut -f1)"
      links="$links<a href=\"$v/$bn\">$bn</a> ($sz)　"
    done
    row="$row<tr><td>$pv</td><td>$iv</td><td>$bt</td><td>$links<a href=\"$v/RELEASE.txt\">RELEASE.txt</a></td></tr>"
  done
  cat > "$idx" <<HTML
<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8">
<title>OMC 离线交付包下载</title>
<style>
body{font-family:sans-serif;margin:2rem;color:#222}
h1{font-size:1.3rem}
table{border-collapse:collapse;width:100%}
th,td{border:1px solid #ccc;padding:.5rem .7rem;text-align:left;font-size:.9rem}
th{background:#f0f0f0}
a{color:#1668dc;text-decoration:none}a:hover{text-decoration:underline}
</style></head><body>
<h1>OMC 离线交付包下载</h1>
<p>按目标机架构下载对应包（<code>uname -m</code>：x86_64→amd64，aarch64→arm64）；
下载后 <code>sha256sum -c</code> 校验，部署见包内运维侧手册。</p>
<table><thead><tr><th>项目版本</th><th>基础设施版本</th><th>构建时间</th><th>下载</th></tr></thead>
<tbody>$row</tbody></table>
<p style="color:#888;font-size:.8rem">索引刷新于 $(date -Is)</p>
</body></html>
HTML
}
regen_index
log "已刷新下载索引：archive/index.html"

# ── 完成 ────────────────────────────────────────────────────────────────
echo
log "全部完成。"
log "  项目版本     ：$VERSION"
log "  基础设施版本 ：$INFRA_VER（$INFRA_NOTE）"
log "  归档位置     ：archive/$VERSION/"
ls -lh "$OUT"/omc-release-*."$EXT" 2>/dev/null || true
echo
log "起 HTTP 下载服务： ./serve.sh    然后浏览器访问 http://<构建机IP>:8000/"

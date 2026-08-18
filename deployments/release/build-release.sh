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
# 项目版本号（交付包与镜像 tag 始终使用 X.Y.Z-YYYYMMDD-HHMM）：
#   不带 -v  → 自动生成：<RELEASE_BASE_VERSION>-<构建时间戳>
#   带  -v   → 指定基础版本：<X.Y.Z>-<构建时间戳>（指定的版本号会自动追加时间戳，
#              保证镜像 tag 永远唯一，使 install.sh 的 images_exist 智能跳过逻辑安全）
#   release 渠道的前端左下角显示基础版本 X.Y.Z；test 渠道继续显示完整版本。
#   例： ./build-release.sh -v 1.0.0   →  1.0.0-20260522-1530
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
RELEASE_HTTPS_CERT_REPO_DIR="$REPO_ROOT/deployments/release/bundle/deploy/nginx-cert"
RELEASE_HTTPS_CERT_SOURCE_DIR="$RELEASE_HTTPS_CERT_REPO_DIR"
RELEASE_HTTPS_CERT_PACKAGE_DIR="deploy/nginx-cert"

release_https_cert_public_fingerprint() { # release_https_cert_public_fingerprint <cert|key> <path>
  local kind="$1" path="$2"
  case "$kind" in
    cert) openssl x509 -in "$path" -pubkey -noout ;;
    key)  openssl pkey -in "$path" -pubout ;;
    *) return 1 ;;
  esac | openssl pkey -pubin -outform DER | sha256sum | awk '{print $1}'
}

validate_release_https_cert_assets() { # validate_release_https_cert_assets <dir>
  local dir="$1" cert key cert_fp key_fp
  cert="$dir/cert.pem"
  key="$dir/key.pem"

  [ -d "$dir" ] ||
    die "缺少 OMC HTTPS 8443 证书资产目录：$dir
      请把老 OMC 的 cert.pem/key.pem 放入仓库固定目录 deployments/release/bundle/deploy/nginx-cert/。"
  [ -f "$cert" ] || die "缺少 OMC HTTPS 8443 证书资产：$cert"
  [ -f "$key" ] || die "缺少 OMC HTTPS 8443 私钥资产：$key"
  [ -r "$cert" ] || die "OMC HTTPS 8443 证书资产不可读：$cert"
  [ -r "$key" ] || die "OMC HTTPS 8443 私钥资产不可读：$key"
  command -v openssl >/dev/null 2>&1 ||
    die "缺少 openssl，无法校验 OMC HTTPS 8443 证书与私钥是否匹配"

  cert_fp="$(release_https_cert_public_fingerprint cert "$cert" 2>/dev/null)" ||
    die "OMC HTTPS 8443 证书资产解析失败：$cert"
  key_fp="$(release_https_cert_public_fingerprint key "$key" 2>/dev/null)" ||
    die "OMC HTTPS 8443 私钥资产解析失败：$key"
  [ -n "$cert_fp" ] && [ -n "$key_fp" ] && [ "$cert_fp" = "$key_fp" ] ||
    die "OMC HTTPS 8443 证书与私钥不匹配：$cert / $key"
}

copy_release_https_cert_assets() { # copy_release_https_cert_assets <stage>
  local stage="$1" dst
  dst="$stage/$RELEASE_HTTPS_CERT_PACKAGE_DIR"
  mkdir -p "$dst"
  install -m 0644 "$RELEASE_HTTPS_CERT_SOURCE_DIR/cert.pem" "$dst/cert.pem"
  install -m 0600 "$RELEASE_HTTPS_CERT_SOURCE_DIR/key.pem" "$dst/key.pem"
}

# 使用构建机当地时间，并保留时区偏移（例如 +08:00），避免与版本目录中的
# 本地时间戳（YYYYMMDD-HHMM）相差数小时。%z 同时兼容 GNU/Linux 与 BSD/macOS。
iso_time() {
  local stamp tz
  stamp="$(date '+%Y-%m-%dT%H:%M:%S%z')"
  tz="${stamp:19:5}"
  printf '%s%s:%s\n' "${stamp:0:19}" "${tz:0:3}" "${tz:3:2}"
}

# ── 参数解析 ────────────────────────────────────────────────────────────
VERSION=""
CHANNEL="${RELEASE_CHANNEL:-test}"
VERIFY_ONLY=0
VERIFY_HTTPS_CERT_ASSETS_ONLY=0
VERIFY_HTTPS_CERT_PACKAGE_LAYOUT_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version) VERSION="$2"; shift 2 ;;
    --channel)    CHANNEL="$2"; shift 2 ;;
    --arch)       ARCHES="$2"; shift 2 ;;
    --verify-only) VERIFY_ONLY=1; shift ;;
    --verify-https-cert-assets-only) VERIFY_HTTPS_CERT_ASSETS_ONLY=1; shift ;;
    --verify-https-cert-package-layout-only) VERIFY_HTTPS_CERT_PACKAGE_LAYOUT_ONLY=1; shift ;;
    -h|--help)    sed -n '3,26p' "$0"; exit 0 ;;
    *)            die "未知参数：$1（-h 查看用法）" ;;
  esac
done

case "$CHANNEL" in
  test|release) ;;
  *) die "--channel 取值非法：${CHANNEL}（应为 test 或 release）" ;;
esac

# 仅支持 amd64（见 release.conf 注释 "架构支持"）
for _a in $ARCHES; do
  [ "$_a" = "amd64" ] || die "本工具仅支持 amd64 架构，传入：${_a}。
        如确实需要 arm64：见 release.conf 中关于 架构支持 的注释,
        改 ARCHES + 移除本脚本的校验后自行验证。"
done

if [ "$VERIFY_HTTPS_CERT_ASSETS_ONLY" = 1 ]; then
  command -v sha256sum >/dev/null 2>&1 || die "缺少构建工具：sha256sum"
  validate_release_https_cert_assets "$RELEASE_HTTPS_CERT_SOURCE_DIR"
  log "OMC HTTPS 8443 证书资产校验通过：$RELEASE_HTTPS_CERT_SOURCE_DIR"
  exit 0
fi

if [ "$VERIFY_HTTPS_CERT_PACKAGE_LAYOUT_ONLY" = 1 ]; then
  command -v sha256sum >/dev/null 2>&1 || die "缺少构建工具：sha256sum"
  validate_release_https_cert_assets "$RELEASE_HTTPS_CERT_SOURCE_DIR"
  cert_stage="$(mktemp -d "${TMPDIR:-/tmp}/omc-release-cert-stage.XXXXXX")"
  trap 'rm -rf "$cert_stage"' EXIT
  copy_release_https_cert_assets "$cert_stage"
  [ -f "$cert_stage/$RELEASE_HTTPS_CERT_PACKAGE_DIR/cert.pem" ] ||
    die "最终发布包证书路径缺失：$RELEASE_HTTPS_CERT_PACKAGE_DIR/cert.pem"
  [ -f "$cert_stage/$RELEASE_HTTPS_CERT_PACKAGE_DIR/key.pem" ] ||
    die "最终发布包私钥路径缺失：$RELEASE_HTTPS_CERT_PACKAGE_DIR/key.pem"
  validate_release_https_cert_assets "$cert_stage/$RELEASE_HTTPS_CERT_PACKAGE_DIR"
  log "OMC HTTPS 8443 证书发布包路径校验通过：$RELEASE_HTTPS_CERT_PACKAGE_DIR/cert.pem / key.pem"
  exit 0
fi

log "运行发布前回归门禁 ..."
RELEASE_VERIFY_SCRIPTS=(
  "$SCRIPT_DIR/bundle/deploy/storage-compose_test.sh"
  "$SCRIPT_DIR/bundle/deploy/nginx-https-file-entry_test.sh"
  "$SCRIPT_DIR/bundle/deploy/license-keystore-release_test.sh"
  "$REPO_ROOT/deployments/monitoring/tests/validate-tempo-memory-budget.sh"
  "$SCRIPT_DIR/validate-release-archive-portability.sh"
)
for release_verify_script in "${RELEASE_VERIFY_SCRIPTS[@]}"; do
  if ! bash "$release_verify_script"; then
    die "发布前回归门禁失败：$release_verify_script"
  fi
done
[ "$VERIFY_ONLY" = 1 ] && exit 0

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
validate_release_https_cert_assets "$RELEASE_HTTPS_CERT_SOURCE_DIR"
log "OMC HTTPS 8443 证书资产：$RELEASE_HTTPS_CERT_SOURCE_DIR → $RELEASE_HTTPS_CERT_PACKAGE_DIR"
if ! docker info >/dev/null 2>&1; then
  die "docker 不可用：当前用户可能不在 docker 组。请执行：
      sudo usermod -aG docker \$USER && newgrp docker   （或重新登录）
      然后重新运行本脚本。"
fi
# Dockerfile.app/acs/worker 采用 RUN --mount=type=cache (BuildKit 语法)以加速 Go module / build cache。
# 经典 builder 不识别该语法会报 "the --mount option requires BuildKit"。
# 这里强制开 BuildKit，老 docker (>=18.09) 都支持。
export DOCKER_BUILDKIT="${DOCKER_BUILDKIT:-1}"
# macOS bsdtar/copyfile may otherwise materialize extended attributes as
# AppleDouble files (._name). The runtime XML loader treats those as real
# configuration files and fails on their binary header.
export COPYFILE_DISABLE=1
[ -n "${PROJECT_IMAGE_PREFIX:-}" ] || die "release.conf 未配置 PROJECT_IMAGE_PREFIX"
if [ -z "${BUSINESS_IMAGES+x}" ] || [ "${#BUSINESS_IMAGES[@]}" -eq 0 ]; then
  die "release.conf 未配置 BUSINESS_IMAGES"
fi

# 压缩方式 → tar 选项与扩展名
case "$PKG_COMPRESS" in
  xz)   TAR_OPT="-cJf"; EXT="tar.xz"
        # xz 默认单线程，开 -T0 跟随机器核数多线程压缩（tar 会读该环变传给 xz）
        export XZ_OPT="${XZ_OPT:--T0}" ;;
  gzip) TAR_OPT="-czf"; EXT="tar.gz" ;;
  zstd) TAR_OPT="--zstd -cf"; EXT="tar.zst"
        command -v zstd >/dev/null 2>&1 || die "PKG_COMPRESS=zstd 但未安装 zstd" ;;
  *)    die "release.conf 的 PKG_COMPRESS 取值非法：${PKG_COMPRESS}（应为 xz/gzip/zstd）" ;;
esac

# ── 项目版本号解析 ──────────────────────────────────────────────────────
# 无论是否传 -v，最终版本号统一追加 -YYYYMMDD-HHMM 时间戳，保证镜像 tag 永远唯一。
TIMESTAMP="$(date +%Y%m%d-%H%M)"
if [ -n "$VERSION" ]; then
  VERSION="${VERSION}-${TIMESTAMP}"
  log "项目版本【手动指定基础版本 + 时间戳】：$VERSION"
else
  VERSION="${RELEASE_BASE_VERSION}-${TIMESTAMP}"
  log "项目版本【自动生成】：$VERSION"
fi
# 交付包、镜像 tag 和后端版本继续使用带时间戳的 VERSION；只有前端展示版本
# 按发布渠道裁剪，避免 release 页面显示构建时间。
if [ "$CHANNEL" = "release" ]; then
  DISPLAY_VERSION="${VERSION%-${TIMESTAMP}}"
else
  DISPLAY_VERSION="$VERSION"
fi
GIT_COMMIT="$(cd "$REPO_ROOT" && git rev-parse HEAD 2>/dev/null || echo n/a)"

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

  # 1.1 构建业务镜像。目标架构必须显式传给 Docker；否则 Apple Silicon 构建机会
  # 产出 arm64 镜像，即使包名和 release.conf 标记为 amd64，目标机将 exec format error。
  #
  # --network host：build container 共用宿主网络栈，绕开默认 bridge 网络
  # （172.17.0.0/16）。某些服务器的 FORWARD/DOCKER-FORWARD 链被 ufw / fail2ban
  # / 自家安全脚本改坏，bridge → 公网包被 DROP，表现为容器内 DNS 超时、apk/
  # go mod download 全失败，而宿主网络完全正常（典型症状：
  # `dial tcp: lookup goproxy.cn on 223.6.6.6:53: i/o timeout`）。
  # 走 host 网络后只要宿主能联网就能 build，免去现场排查 iptables。
  log "[$ARCH] docker build 业务镜像（${BUSINESS_IMAGES[*]}）..."
  IMG_REFS=()
  for SVC in "${BUSINESS_IMAGES[@]}"; do
    DOCKERFILE="$REPO_ROOT/deployments/docker/Dockerfile.$SVC"
    [ -f "$DOCKERFILE" ] || die "缺 Dockerfile：$DOCKERFILE"
    TAG="$PROJECT_IMAGE_PREFIX/$SVC:$VERSION"
    log "[$ARCH]   docker build -t $TAG -f $DOCKERFILE"
    # 正式构建期注入版本信息：web 用于页面展示，app 用于识别 OMC 发布切换。
    EXTRA_ARGS=()
    [ "$SVC" = "web" ] && EXTRA_ARGS+=( --build-arg "APP_VERSION=$DISPLAY_VERSION" )
    if [ "$SVC" = "app" ]; then
      EXTRA_ARGS+=(
        --build-arg "RELEASE_VERSION=$VERSION"
        --build-arg "GIT_COMMIT=$GIT_COMMIT"
      )
    fi
    ( cd "$REPO_ROOT" && docker build \
        --network host \
        --platform "linux/$ARCH" \
        -t "$TAG" \
        -f "$DOCKERFILE" \
        --build-arg APK_MIRROR=mirrors.aliyun.com \
        ${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"} \
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
  mkdir -p "$STAGE/license/keystore"
  install -m 0644 "$REPO_ROOT/license-run-time/keystore/omcPublicKey.store" \
    "$STAGE/license/keystore/omcPublicKey.store"

  # 1.4 部署模板 + compose 文件 + nginx 配置 + 监控栈配置
  log "[$ARCH] 拷入部署模板 + 监控栈配置 ..."
  cp -r "$SCRIPT_DIR/bundle/deploy"       "$STAGE/deploy"
  cp -r "$REPO_ROOT/deployments/monitoring" "$STAGE/deploy/monitoring"
  cp "$REPO_ROOT/deployments/docker/nginx.conf"   "$STAGE/deploy/nginx.conf"
  cp "$REPO_ROOT/deployments/docker/default.conf" "$STAGE/deploy/default.conf"
  copy_release_https_cert_assets "$STAGE"

  # 关键：强制 world-read。cp 不带 -p 时会按构建机 umask 写入 mode，
  # 若 umask=027 则配置文件落 0640，prometheus/loki/tempo/alertmanager 等
  # 非 root 容器（UID 65534/10001 等）读不动，启动直接 fail "permission denied"。
  # 大写 X 只补目录的 x，不会给普通文件加可执行位。
  chmod -R a+rX "$STAGE/deploy/monitoring"
  cat > "$STAGE/deploy/.env" <<EOF
# 项目版本（业务镜像 tag 取自此处）
PROJECT_VERSION=$VERSION
IMAGE_PREFIX=$PROJECT_IMAGE_PREFIX
# 业务镜像
IMAGE_APP=$PROJECT_IMAGE_PREFIX/app:$VERSION
IMAGE_ACS=$PROJECT_IMAGE_PREFIX/acs:$VERSION
IMAGE_WORKER=$PROJECT_IMAGE_PREFIX/worker:$VERSION
IMAGE_WEB=$PROJECT_IMAGE_PREFIX/web:$VERSION
# 基础设施镜像（主库纯 PG / 时序库 TimescaleDB —— 物理分离，镜像分离）
IMAGE_POSTGRES=$IMAGE_POSTGRES
IMAGE_POSTGRES_TSDB=$IMAGE_POSTGRES_TSDB
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
IMAGE_NGINX_EXPORTER=$IMAGE_NGINX_EXPORTER
IMAGE_NODE_EXPORTER=$IMAGE_NODE_EXPORTER
IMAGE_CADVISOR=$IMAGE_CADVISOR
# 密钥键不再发默认值（#175 凭证治本）：install.sh 首次安装用 ensure_secrets 自动生成强随机，
# 存到 etc/secrets.env（uninstall 保留、--purge 删），并在起容器前覆盖回本文件。占位 REPLACE_ME
# 被生产凭证校验（GuardProductionSecrets）识别；正常流程下 ensure_secrets 已替换为强随机。
# 非密钥的固定用户名 POSTGRES_USER / POSTGRES_DB / GRAFANA_ADMIN_USER 保留。
POSTGRES_USER=omcgo
POSTGRES_PASSWORD=REPLACE_ME
POSTGRES_DB=omcgo
# 时序库 postgres-tsdb（独立 TimescaleDB 实例：PM/KPI 超表 / 告警历史 / MR / trace）。
# 口令【独立生成】（#347）：install.sh ensure_secrets 为 POSTGRES_TSDB_PASSWORD 生成强随机，
# 不复用主库 POSTGRES_PASSWORD。USER/DB 为固定名（同主库习惯），TSDB_HOST=compose 内服务名。
POSTGRES_TSDB_USER=omcgo
POSTGRES_TSDB_PASSWORD=REPLACE_ME
POSTGRES_TSDB_DB=omcgo
TSDB_HOST=postgres-tsdb
MINIO_ROOT_USER=REPLACE_ME
MINIO_ROOT_PASSWORD=REPLACE_ME
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=REPLACE_ME
# 有状态服务数据目录（非密钥）。留空继续使用原 Docker 命名卷，升级不会隐式切换数据。
# 新部署建议先运行 deploy/plan-resources.sh 自动填入最大可用盘，再按物理 SSD/NVMe 人工拆分。
# 已有数据修改这些值前必须停服并完成数据复制；脚本不会自动迁移。
POSTGRES_DATA_PATH=
TSDB_DATA_PATH=
REDIS_DATA_PATH=
REDIS_PM_DATA_PATH=
NATS_DATA_PATH=
MINIO_DATA_PATH=
# 写入保护运行时路径解析。默认生产部署使用 docker compose -p omcgo；
# Docker Root Dir 如被安装到非 /var/lib/docker，请在部署前改成 docker info 的值。
OMCGO_DOCKER_ROOT_DIR=/var/lib/docker
OMCGO_DOCKER_VOLUME_PREFIX=omcgo
OMCGO_HOST_LOGS_PATH=/opt/omc/run/logs
OMCGO_HOST_DATA_PATH=/opt/omc/data
# OMC 运行环境（容器内 entrypoint.sh 读）
OMCGO_ENV=prod
# JWT 密钥（app 容器读）—— 由 install.sh ensure_secrets 自动生成（#175）
OMCGO_JWT_SECRET=REPLACE_ME
# TR-069 ConnReq/STUN 共享密钥（app/acs 容器读）—— 由 install.sh ensure_secrets 自动生成（#175）
OMC_SHARED_SECRET=REPLACE_ME
# 本机对外 IP / 域名（基站可达地址）——【部署前必填】。app/acs/worker 容器读，
# worker 据此生成 PM 文件上传 URL 下发给基站；不能用 localhost / 127.0.0.1，
# 否则基站无法回传文件。示例：OMC_PUBLIC_HOST=172.19.1.132
OMC_PUBLIC_HOST=
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
build_time=$(iso_time)
git_commit=$GIT_COMMIT
EOF
  cat > "$STAGE/README.md" <<EOF
# OMC 项目交付包 — $VERSION ($ARCH)

- 项目版本：$VERSION
- 发布渠道：${CHANNEL}（test=测试阶段 / release=正式发布）
- 架构：${ARCH}（目标机 \`uname -m\`：x86_64→amd64，aarch64→arm64）
- 业务镜像：${BUSINESS_IMAGES[*]/#/$PROJECT_IMAGE_PREFIX/}（tag = ${VERSION}）
- 构建时间：$(iso_time)　git commit：$GIT_COMMIT

本包【全 docker compose 部署】，含业务镜像 tar（docker save）+ compose
文件 + 配置模板 + 迁移 / 字典 / Casbin（可挂载覆盖）+ 监控栈配置 + 运维脚本。
Docker 引擎、compose v2 二进制、基础镜像在【独立的基础设施包 omc-infra-*】里——
首次部署需先用基础设施包装好 Docker / Compose、导入基础镜像，再部署本包。
包内 \`deploy/nginx-cert/cert.pem\` 与 \`deploy/nginx-cert/key.pem\` 会由
\`deploy/install.sh\` 自动安装到宿主机 \`/etc/nginx/cert/\`，用于启用基站
HTTPS 8443 ACS 入口。

部署步骤见 \`docs/OMC内网离线部署手册（运维侧）.md\`。先校验完整性：
\`\`\`bash
sha256sum -c checksums.sha256
\`\`\`
PostgreSQL / MinIO / JWT / Grafana 默认口令必须在部署时修改（deploy/.env）。
\`OMC_PUBLIC_HOST\` 必须在部署时填本机对外 IP（基站可达），否则基站无法回传 PM 文件（deploy/.env）。
EOF
  # 防止上游复制阶段已经带入 AppleDouble 文件；只清理本次临时 staging。
  find "$STAGE" -type f -name '._*' -delete
  ( cd "$STAGE" && find . -type f ! -name checksums.sha256 -print0 \
      | sort -z | xargs -0 sha256sum > checksums.sha256 )

  # 1.7 压缩打包 → archive/project/<版本>/
  log "[$ARCH] 压缩打包（${PKG_COMPRESS}）..."
  # shellcheck disable=SC2086
  # GNU tar 与 bsdtar 都支持 --no-xattrs；避免 macOS provenance 等主机元数据
  # 进入 pax header，导致 Linux 解包产生大量未知扩展属性警告。
  ( cd "$WORK" && tar --no-xattrs $TAR_OPT "$OUT/$PKG_NAME.$EXT" "$PKG_NAME" )
  ( cd "$OUT" && sha256sum "$PKG_NAME.$EXT" > "$PKG_NAME.$EXT.sha256" )
  log "[$ARCH] 产出：archive/project/$VERSION/$PKG_NAME.$EXT"
done

rm -rf "$WORK"

# ── 3. 版本构建说明 RELEASE.txt ──────────────────────────────────────────
{
  echo "project_version=$VERSION"
  echo "channel=$CHANNEL"
  echo "build_time=$(iso_time)"
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
  echo "业务镜像前缀：$PROJECT_IMAGE_PREFIX  （tag = ${VERSION}）"
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

# ── 4. 自动刷新 HTTP 下载服务（daemon 模式），让新包立刻可被下载 ──────────
# 走 serve.sh restart（= 内部 daemon 化的 serve.sh start）：未启动→启动；已在跑→
# 停旧实例后用上次端口重启。serve.sh 在 root+systemd 机器上用 systemd 瞬态服务托管
# （脱离登录会话，关 ssh / 退出登录不掉），非 root/无 systemd 才回退 setsid+nohup。
# 本脚本【不自己 nohup】，统一交给 serve.sh 的 daemon 逻辑。
# 即使重启失败也不报错（build 主流程已完成，serve 只是便利）。
if [ -x "$SCRIPT_DIR/serve.sh" ]; then
  log "刷新 HTTP 下载服务（serve.sh restart，daemon 模式）..."
  if "$SCRIPT_DIR/serve.sh" restart; then
    log "HTTP 下载服务已重启，可访问 http://<构建机IP>:$(cat "$SCRIPT_DIR/.serve.port" 2>/dev/null || echo 8000)/"
  else
    warn "serve.sh restart 失败（不影响发布包），可手动 ./serve.sh start 启动"
  fi
else
  log "起 HTTP 下载服务： ./serve.sh    然后浏览器访问 http://<构建机IP>:8000/"
fi
log "基础设施包（Docker 引擎 + 基础镜像）由 ./build-images.sh 生成，与本包独立。"

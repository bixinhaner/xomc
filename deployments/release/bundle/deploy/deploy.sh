#!/usr/bin/env bash
# =============================================================================
# OMC 一键部署脚本 — 内网交付侧（全 docker compose 部署）
#
# 把项目包从「解压后」的状态一键推到「全栈在跑」。覆盖：
#   precheck → 旧 systemd 自动迁移 → 建目录 → load 镜像（基础设施 + 监控 + 业务）
#   → 默认口令检查 → migrate（容器） → seed（容器） → up 全栈（infra+app+web+monitoring）
#   → healthcheck
#
# ★ 全 docker compose 部署：所有 OMC 服务（app/acs/worker/web/监控栈）均跑在
#   docker 容器里，宿主机上不再放业务二进制 / systemd 单元。
# ★ 本脚本【从项目包根目录】运行，即解压后看到 deploy/ etc/ images/ 的那一层。
# ★ 用 root 执行（docker / 写 /opt 都需要 root）。
#
# 用法：
#   sudo bash deploy/deploy.sh                          # 全套首次部署
#   sudo bash deploy/deploy.sh --skip-infra             # 已部署 infra 镜像，仅升级业务镜像
#   sudo bash deploy/deploy.sh --skip-migrate           # 不跑 migrate / seed
#   sudo bash deploy/deploy.sh --skip-web               # 不起 web 容器
#   sudo bash deploy/deploy.sh --skip-monitoring        # 不起监控栈
#   sudo bash deploy/deploy.sh --check-only             # 仅检查环境，不做修改
#   sudo bash deploy/deploy.sh --infra-dir /opt/omc/infra   # 自定义 infra 目录
#   sudo bash deploy/deploy.sh --overwrite-etc          # 用新包模板覆盖 /opt/omc/etc
#   sudo bash deploy/deploy.sh -h | --help              # 本帮助
#
# 参数：
#   --skip-infra      跳过：基础镜像 load（infra 镜像已 load 过）
#   --skip-migrate    跳过：migrate + seed 容器
#   --skip-web        跳过：web compose 文件
#   --skip-monitoring 跳过：monitoring compose 文件
#   --check-only      仅 precheck，不做任何修改（dry run）
#   --infra-dir <p>   基础设施镜像目录（默认 /opt/omc/infra）
#                     基础设施包 omc-infra-*.tar.xz 需提前解压到此目录。
#   --omc-root <p>    OMC 安装根（默认 /opt/omc）
#   --overwrite-etc   用新包 etc/ 模板覆盖 /opt/omc/etc/（旧 etc 自动备份）
#   --yes             所有交互式提示直接默认（适合 CI / 批处理）
#   -h | --help       本帮助
#
# 退出码：
#   0  全部 OK
#   1  precheck 失败
#   2  基础设施未就绪
#   3  migrate / seed 失败
#   4  healthcheck 失败
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PKG_ROOT="$(cd "$DEPLOY_DIR/.." && pwd)"   # 项目包根（deploy/ etc/ images/ 上一级）

log()  { echo -e "\033[1;32m[deploy]\033[0m $*"; }
warn() { echo -e "\033[1;33m[deploy][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[deploy][错误]\033[0m $*" >&2; exit "${2:-1}"; }
sep()  { echo -e "\033[1;34m──────────────── $* ────────────────\033[0m"; }

# ── 参数解析 ────────────────────────────────────────────────────────────
SKIP_INFRA=0
SKIP_MIGRATE=0
SKIP_WEB=0
SKIP_MONITORING=0
CHECK_ONLY=0
ASSUME_YES=0
OVERWRITE_ETC=0
INFRA_DIR="/opt/omc/infra"
OMC_ROOT="/opt/omc"
COMPOSE_PROJECT="omcgo"

while [ $# -gt 0 ]; do
  case "$1" in
    --skip-infra)      SKIP_INFRA=1; shift ;;
    --skip-migrate)    SKIP_MIGRATE=1; shift ;;
    --skip-web)        SKIP_WEB=1; shift ;;
    --skip-monitoring) SKIP_MONITORING=1; shift ;;
    --check-only)      CHECK_ONLY=1; shift ;;
    --infra-dir)       INFRA_DIR="$2"; shift 2 ;;
    --omc-root)        OMC_ROOT="$2"; shift 2 ;;
    --overwrite-etc)   OVERWRITE_ETC=1; shift ;;
    --yes)             ASSUME_YES=1; shift ;;
    -h|--help)         sed -n '3,49p' "$0"; exit 0 ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

confirm() {
  [ "$ASSUME_YES" = 1 ] && return 0
  local yn
  read -rp "$1 [Y/n] " yn
  case "${yn:-Y}" in [Yy]*|"") return 0 ;; *) return 1 ;; esac
}

# =============================================================================
# Step 1. precheck
# =============================================================================
sep "1/9 precheck"

# 工具齐全
for tool in docker tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少工具：$tool（请先装 Docker 等基础工具）" 1
done

# docker compose v2
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  die "未检测到 docker compose v2 / docker-compose v1（请先 install-docker.sh）" 1
fi
log "docker compose 命令：$COMPOSE"

# docker 服务可用
docker info >/dev/null 2>&1 || die "docker 服务不可用，请先 systemctl start docker" 1

# 项目包结构（全 compose 部署：不再要求 bin/，要求 images/ deploy/ etc/）
[ -d "$PKG_ROOT/etc" ]    || die "项目包目录结构异常：缺 etc/" 1
[ -d "$PKG_ROOT/deploy" ] || die "项目包目录结构异常：缺 deploy/" 1
[ -d "$PKG_ROOT/images" ] || die "项目包目录结构异常：缺 images/（业务镜像 tar）" 1
[ -f "$PKG_ROOT/deploy/.env" ]                         || die "缺 deploy/.env（由 build-release.sh 生成）" 1
[ -f "$PKG_ROOT/deploy/docker-compose.infra.yml" ]     || die "缺 deploy/docker-compose.infra.yml" 1
[ -f "$PKG_ROOT/deploy/docker-compose.app.yml" ]       || die "缺 deploy/docker-compose.app.yml" 1
[ "$SKIP_WEB" = 1 ]        || [ -f "$PKG_ROOT/deploy/docker-compose.web.yml" ]        || die "缺 deploy/docker-compose.web.yml（或加 --skip-web）" 1
[ "$SKIP_MONITORING" = 1 ] || [ -f "$PKG_ROOT/deploy/docker-compose.monitoring.yml" ] || die "缺 deploy/docker-compose.monitoring.yml（或加 --skip-monitoring）" 1

# 项目版本号
VERSION="$(awk -F= '/^project_version=/{print $2}' "$PKG_ROOT/VERSION" 2>/dev/null || echo unknown)"
log "项目版本：$VERSION"

# infra 目录（除非 --skip-infra）
if [ "$SKIP_INFRA" = 0 ]; then
  [ -d "$INFRA_DIR" ] || die "基础设施目录不存在：$INFRA_DIR
  · 首次部署需先：cd $INFRA_DIR && tar -xJf omc-infra-<版本>-<架构>.tar.xz --strip-components=1
  · 或加 --skip-infra 跳过基础设施镜像 load" 1
  [ -d "$INFRA_DIR/images" ] || die "基础设施目录缺 images/：$INFRA_DIR/images" 1
fi

log "precheck 通过"

if [ "$CHECK_ONLY" = 1 ]; then
  log "--check-only：不做任何修改，退出"
  exit 0
fi

# =============================================================================
# Step 2. 旧 systemd 单元自动迁移（兼容旧版宿主机二进制部署）
#   旧版本曾用 systemd 跑 omcgo-{app,acs,worker} 二进制；本版本改为容器，
#   必须先停掉并禁用旧 systemd 单元，否则会与容器抢 9091/7547 等端口。
# =============================================================================
sep "2/9 旧 systemd 单元自动迁移"

if command -v systemctl >/dev/null 2>&1; then
  HAS_OLD=0
  for svc in omcgo-app omcgo-acs omcgo-worker; do
    UNIT="/etc/systemd/system/$svc.service"
    if [ -f "$UNIT" ] || systemctl list-unit-files "$svc.service" >/dev/null 2>&1; then
      HAS_OLD=1
      log "检测到旧 systemd 单元：$svc"
      systemctl stop "$svc" 2>/dev/null || true
      systemctl disable "$svc" 2>/dev/null || true
      if [ -f "$UNIT" ]; then
        BAK="$UNIT.bak.$(date +%Y%m%d%H%M%S)"
        mv "$UNIT" "$BAK"
        log "  · 备份并移除单元 → $BAK"
      fi
    fi
  done
  if [ "$HAS_OLD" = 1 ]; then
    systemctl daemon-reload
    log "旧 systemd 单元已停掉并禁用（业务进程将由 docker compose 接管）"
    warn "宿主机旧 OMC 二进制（如 /opt/omc/current/bin/）保留未删，请运维确认无残留进程后自行清理"
  else
    log "未检测到旧 systemd 单元，跳过"
  fi
else
  log "systemctl 不存在（非 systemd 系统），跳过旧单元迁移"
fi

# =============================================================================
# Step 3. 建立 OMC 目录布局 + current 软链
# =============================================================================
sep "3/9 建立目录结构 + current 软链"

mkdir -p "$OMC_ROOT/releases" "$OMC_ROOT/etc" "$OMC_ROOT/packages" \
         "$OMC_ROOT/run/logs/app"   "$OMC_ROOT/run/logs/acs" \
         "$OMC_ROOT/run/logs/worker" "$OMC_ROOT/run/logs/nginx"

RELEASE_DIR="$OMC_ROOT/releases/$VERSION"
if [ -d "$RELEASE_DIR" ] && [ "$(readlink -f "$PKG_ROOT" 2>/dev/null)" != "$(readlink -f "$RELEASE_DIR" 2>/dev/null)" ]; then
  warn "已存在版本目录 $RELEASE_DIR，将覆盖（旧文件 → .bak.<时间戳>）"
  confirm "继续吗？" || die "用户取消" 1
  mv "$RELEASE_DIR" "$RELEASE_DIR.bak.$(date +%Y%m%d%H%M%S)"
fi

# 如果不是直接在版本目录里跑，复制到版本目录
if [ "$(readlink -f "$PKG_ROOT")" != "$(readlink -f "$RELEASE_DIR")" ]; then
  log "复制项目包到 $RELEASE_DIR ..."
  mkdir -p "$RELEASE_DIR"
  cp -a "$PKG_ROOT/." "$RELEASE_DIR/"
fi

# 实例配置：首次复制模板；非首次默认保留以保护已改口令
etc_is_empty=0
[ -z "$(ls -A "$OMC_ROOT/etc" 2>/dev/null)" ] && etc_is_empty=1

if [ "$etc_is_empty" = 1 ]; then
  log "首次部署：复制配置模板到 $OMC_ROOT/etc/（首次必修改默认口令！）"
  cp -rn "$RELEASE_DIR/etc/." "$OMC_ROOT/etc/"
else
  do_overwrite=0
  if [ "$OVERWRITE_ETC" = 1 ]; then
    do_overwrite=1
  elif [ "$ASSUME_YES" = 0 ]; then
    warn "$OMC_ROOT/etc/ 已有实例配置（包含可能已改好的强口令 / JWT 密钥 / TLS 证书路径等）"
    warn "  选 y 将覆盖为新包模板（原 etc 自动备份到 etc.bak.<时间戳>）"
    warn "  选 N 保留现有配置不动（默认）"
    read -rp "是否用新包模板覆盖 $OMC_ROOT/etc/？ [y/N] " yn
    case "${yn:-N}" in [Yy]*) do_overwrite=1 ;; esac
  fi

  if [ "$do_overwrite" = 1 ]; then
    BAK="$OMC_ROOT/etc.bak.$(date +%Y%m%d%H%M%S)"
    log "备份原 etc → $BAK"
    mv "$OMC_ROOT/etc" "$BAK"
    mkdir -p "$OMC_ROOT/etc"
    cp -r "$RELEASE_DIR/etc/." "$OMC_ROOT/etc/"
    warn "etc 已重置为新包模板 —— 请从 $BAK 取回已改口令 / JWT / TLS / 自定义项"
    warn "  参考 diff：diff -ru $BAK $OMC_ROOT/etc | less"
  else
    log "$OMC_ROOT/etc/ 已有实例配置，保留不覆盖（如需覆盖加 --overwrite-etc）"
  fi
fi

# 切 current 软链（原子）
ln -sfn "$RELEASE_DIR" "$OMC_ROOT/current"
log "current → $RELEASE_DIR"

# 检查一组镜像是否全部已在本地
# 用法：images_exist IMAGE1 IMAGE2 ...
# 返回：0=全部存在  1=有缺失
images_exist() {
  local img
  for img in "$@"; do
    [ -z "$img" ] && continue
    if ! docker image inspect "$img" >/dev/null 2>&1; then
      return 1
    fi
  done
  return 0
}

# =============================================================================
# Step 4. load 镜像（基础设施 + 监控 + 业务）
# =============================================================================
sep "4/9 load 镜像"

# 提前加载 .env 获取镜像名（IMAGE_* 变量），供 images_exist 判定使用
ENV_FILE="$OMC_ROOT/current/deploy/.env"
if [ -f "$ENV_FILE" ]; then
  set -a; source "$ENV_FILE"; set +a
fi

if [ "$SKIP_INFRA" = 0 ]; then
  INFRA_IMAGES=("$IMAGE_POSTGRES" "$IMAGE_REDIS" "$IMAGE_NATS" "$IMAGE_MINIO" "${IMAGE_NGINX:-}")
  MON_IMAGES=("${IMAGE_PROMETHEUS:-}" "${IMAGE_ALERTMANAGER:-}" "${IMAGE_GRAFANA:-}" "${IMAGE_LOKI:-}" "${IMAGE_TEMPO:-}" "${IMAGE_OTELCOL:-}" "${IMAGE_NATS_EXPORTER:-}")

  if images_exist "${INFRA_IMAGES[@]}" "${MON_IMAGES[@]}"; then
    log "基础设施 + 监控镜像已存在，跳过 load，重启容器"
    $COMPOSE -p "$COMPOSE_PROJECT" restart postgres redis nats minio 2>/dev/null || true
  else
    log "load 基础设施 + 监控镜像（$INFRA_DIR/images/）"
    for tar in "$INFRA_DIR/images"/*.tar; do
      [ -f "$tar" ] || continue
      log "  · docker load < $(basename "$tar")"
      docker load -i "$tar"
    done
  fi
else
  log "--skip-infra：跳过基础设施 / 监控镜像 load"
fi

BIZ_IMAGES=("$IMAGE_APP" "$IMAGE_ACS" "$IMAGE_WORKER" "$IMAGE_WEB")

if images_exist "${BIZ_IMAGES[@]}"; then
  log "业务镜像已存在，跳过 load，重启业务容器"
  $COMPOSE -p "$COMPOSE_PROJECT" restart app acs worker web 2>/dev/null || true
  biz_loaded=1
else
  log "load 业务镜像（$RELEASE_DIR/images/）"
  biz_loaded=0
  for tar in "$RELEASE_DIR/images"/*.tar; do
    [ -f "$tar" ] || continue
    log "  · docker load < $(basename "$tar")"
    docker load -i "$tar"
    biz_loaded=1
  done
  [ "$biz_loaded" = 1 ] || die "$RELEASE_DIR/images/ 下无业务镜像 tar，无法继续" 1
fi

# =============================================================================
# Step 5. 默认口令检查（PostgreSQL / MinIO / Grafana）
# =============================================================================
sep "5/9 默认口令安全检查"

# ENV_FILE 已在 Step 4 定义并 source，这里仅做存在性兜底
[ -f "$ENV_FILE" ] || die "缺 $ENV_FILE（由 build-release.sh 生成）" 1

if grep -qE '^(POSTGRES_PASSWORD=omcgo123|MINIO_ROOT_PASSWORD=minioadmin|GRAFANA_ADMIN_PASSWORD=admin)$' "$ENV_FILE"; then
  warn "检测到 deploy/.env 含默认口令（POSTGRES_PASSWORD=omcgo123 / MINIO_ROOT_PASSWORD=minioadmin / GRAFANA_ADMIN_PASSWORD=admin）"
  warn "  → 生产环境务必改强口令：vi $ENV_FILE"
  warn "  → 同时 $OMC_ROOT/etc/*.prod.yaml 内 dsn / minio 配置需保持一致"
  confirm "已知风险，继续部署？" || die "用户取消，请先改口令" 1
fi

# =============================================================================
# Step 6. 组装 docker compose 命令（全栈一次 up）
# =============================================================================
sep "6/9 组装 docker compose 命令"

cd "$OMC_ROOT/current/deploy"

COMPOSE_FILES=( -f docker-compose.infra.yml -f docker-compose.app.yml )
[ "$SKIP_WEB" = 0 ]        && COMPOSE_FILES+=( -f docker-compose.web.yml )
[ "$SKIP_MONITORING" = 0 ] && COMPOSE_FILES+=( -f docker-compose.monitoring.yml )

DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${COMPOSE_FILES[@]}" )
log "compose 命令：${DC[*]}"

# =============================================================================
# Step 7. 启动基础设施 + 等就绪 → 跑 migrate / seed（一次性容器）
# =============================================================================
sep "7/9 启动基础设施 + 执行 migrate / seed"

# 7.1 起基础设施（postgres / redis / nats / minio）
log "启动基础设施容器 ..."
"${DC[@]}" up -d postgres redis nats minio

log "等待基础设施 ready（最多 90s）..."
WAIT=0
PG_OK=0; RD_OK=0
while [ $WAIT -lt 90 ]; do
  sleep 3; WAIT=$((WAIT+3))
  PG_CID="$("${DC[@]}" ps -q postgres 2>/dev/null || true)"
  RD_CID="$("${DC[@]}" ps -q redis 2>/dev/null || true)"
  if [ -n "$PG_CID" ]; then
    docker exec "$PG_CID" pg_isready -U "${POSTGRES_USER:-omcgo}" >/dev/null 2>&1 && PG_OK=1 || PG_OK=0
  fi
  if [ -n "$RD_CID" ]; then
    docker exec "$RD_CID" redis-cli ping >/dev/null 2>&1 && RD_OK=1 || RD_OK=0
  fi
  [ "$PG_OK" = 1 ] && [ "$RD_OK" = 1 ] && break
  echo "  ... ${WAIT}s (PG=$PG_OK RD=$RD_OK)"
done
if [ "$PG_OK" != 1 ] || [ "$RD_OK" != 1 ]; then
  die "基础设施 90s 内未就绪：PG=$PG_OK RD=$RD_OK
  手动检查：${DC[*]} ps
            ${DC[*]} logs postgres redis" 2
fi
log "基础设施已就绪 (PG / Redis)"

# 7.2 验证 omcgo-net 网络已创建
log "验证 omcgo-net 网络 ..."
if ! docker network inspect omcgo-net >/dev/null 2>&1; then
  die "omcgo-net 网络未创建，请检查 docker-compose.infra.yml" 2
fi

# 7.3 migrate-schema（一次性容器，跑完即退；用 .env 中的 dsn）
if [ "$SKIP_MIGRATE" = 0 ]; then
  log "执行 db migrate（容器：migrate-schema）..."
  if "${DC[@]}" run --rm migrate-schema; then
    log "migrate 成功"
  else
    die "migrate 失败：${DC[*]} run --rm migrate-schema" 3
  fi

  # 7.4 seed（首次部署跑一次；用 .seed.done 标记防重复）
  SEED_MARK="$OMC_ROOT/etc/.seed.done"
  if [ -f "$SEED_MARK" ]; then
    log "已有 $SEED_MARK，跳过 seed（如需重灌请删除该文件再跑）"
  else
    log "执行 db seed（容器：migrate-seed，首次部署）..."
    if "${DC[@]}" run --rm migrate-seed; then
      touch "$SEED_MARK"
      log "seed 成功"
    else
      warn "seed 失败（部分种子可能已存在，不影响主流程；如确需排查请看日志）"
    fi
  fi
else
  log "--skip-migrate：跳过 migrate / seed"
fi

# =============================================================================
# Step 8. up 业务 + web + 监控
# =============================================================================
sep "8/9 启动业务 + web + 监控"

log "${DC[*]} up -d"
"${DC[@]}" up -d

log "等待业务容器启动（10s）..."
sleep 10

# =============================================================================
# Step 9. healthcheck
# =============================================================================
sep "9/9 健康检查"

if bash "$OMC_ROOT/current/deploy/healthcheck.sh"; then
  HEALTH_OK=1
else
  HEALTH_OK=0
fi

echo
sep "部署完成"
log "项目版本：$VERSION"
log "OMC 根目录：$OMC_ROOT"
log "  current → $(readlink -f "$OMC_ROOT/current")"
log "  实例配置：$OMC_ROOT/etc/"
log "  运行日志：$OMC_ROOT/run/logs/{app,acs,worker,nginx}"
echo
log "compose 控制命令："
log "  · 查看状态：  cd $OMC_ROOT/current/deploy && ${DC[*]} ps"
log "  · 查看日志：  cd $OMC_ROOT/current/deploy && ${DC[*]} logs -f <service>"
log "  · 停止全栈：  cd $OMC_ROOT/current/deploy && ${DC[*]} down"
echo
log "访问地址："
log "  · Web UI ：       http://<服务器IP>:8080"
log "  · MinIO Console：http://<服务器IP>:9001"
[ "$SKIP_MONITORING" = 0 ] && log "  · Grafana：       http://<服务器IP>:3000"
log "  · 健康检查：       bash $OMC_ROOT/current/deploy/healthcheck.sh"
echo
log "初始账号（首次登录强制改）："
log "  · Web UI：    admin / admin123"
log "  · MinIO：     minioadmin / minioadmin"
log "  · PostgreSQL：omcgo / omcgo123"
[ "$SKIP_MONITORING" = 0 ] && log "  · Grafana：    admin / admin"
echo

if [ "$HEALTH_OK" != 1 ]; then
  die "健康检查未全通过（部分服务异常），请按上面 healthcheck 输出排查" 4
fi
log "全部 OK 🎉"

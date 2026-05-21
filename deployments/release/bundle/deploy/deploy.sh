#!/usr/bin/env bash
# =============================================================================
# OMC 一键部署脚本 — 内网交付侧
#
# 把项目包从「解压后」的状态一键推到「全栈在跑」。覆盖：
#   precheck → load 镜像 → infra up → 等就绪 → migrate → seed →
#   install systemd → web up → healthcheck
#
# 设计来源：docs/design/deployments-release-enhancements-20260520.md §3.1 D4
# 上下文：deployment 布局参考 docs/operations/OMC内网离线部署手册（运维侧）.md §5
#         /opt/omc/{infra,etc,run,releases/<版本>,current→releases/<版本>}
#
# ★ 本脚本【从项目包根目录】运行，即解压后看到 bin/ web/ deploy/ etc/ 的那一层。
# ★ 用 root 执行（systemctl / docker / 写 /opt 都需要 root）。
#
# 用法：
#   sudo bash deploy/deploy.sh                          # 全套首次部署
#   sudo bash deploy/deploy.sh --skip-infra             # 已部署 infra，仅升级 app
#   sudo bash deploy/deploy.sh --skip-migrate           # 不跑 migrate
#   sudo bash deploy/deploy.sh --skip-web               # 不起 web 容器
#   sudo bash deploy/deploy.sh --check-only             # 仅检查环境，不做修改
#   sudo bash deploy/deploy.sh --infra-dir /opt/omc/infra   # 自定义 infra 目录
#   sudo bash deploy/deploy.sh --overwrite-etc          # 用新包模板覆盖 /opt/omc/etc（旧 etc 自动备份）
#   sudo bash deploy/deploy.sh -h | --help              # 本帮助
#
# 参数：
#   --skip-infra      跳过：基础镜像 load + infra compose up + 等就绪
#                     适用：基础设施已部署过的"日常升级"场景
#   --skip-migrate    跳过：db migrate + seed
#                     适用：手动已迁过 / 调试阶段
#   --skip-web        跳过：web compose up
#                     适用：尚未交付前端，或前端走外部 nginx
#   --check-only      仅 precheck，不做任何修改（dry run）
#   --infra-dir <p>   基础设施安装目录（默认 /opt/omc/infra）。
#                     基础设施包 omc-infra-*.tar.xz 需提前解压到此目录。
#   --omc-root <p>    OMC 安装根（默认 /opt/omc）—— releases / etc / current 都在它下
#   --overwrite-etc   用新包 etc/ 模板覆盖 /opt/omc/etc/。覆盖前自动备份到
#                     /opt/omc/etc.bak.<时间戳>。默认（含 --yes）保留现有 etc
#                     不覆盖，避免冲掉已改好的强口令。
#                     交互模式下若检测到已有 etc，会询问是否覆盖（默认 N）。
#   --yes             所有交互式提示直接默认（适合 CI / 批处理）
#   -h | --help       本帮助
#
# 退出码：
#   0  全部 OK
#   1  precheck 失败
#   2  基础设施未就绪（infra up 后等待超时）
#   3  migrate / seed 失败
#   4  healthcheck 失败
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PKG_ROOT="$(cd "$DEPLOY_DIR/.." && pwd)"   # 项目包根（bin/ web/ deploy/ etc/ 上一级）

log()  { echo -e "\033[1;32m[deploy]\033[0m $*"; }
warn() { echo -e "\033[1;33m[deploy][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[deploy][错误]\033[0m $*" >&2; exit "${2:-1}"; }
sep()  { echo -e "\033[1;34m──────────────── $* ────────────────\033[0m"; }

# ── 参数解析 ────────────────────────────────────────────────────────────
SKIP_INFRA=0
SKIP_MIGRATE=0
SKIP_WEB=0
CHECK_ONLY=0
ASSUME_YES=0
OVERWRITE_ETC=0
INFRA_DIR="/opt/omc/infra"
OMC_ROOT="/opt/omc"
while [ $# -gt 0 ]; do
  case "$1" in
    --skip-infra)    SKIP_INFRA=1; shift ;;
    --skip-migrate)  SKIP_MIGRATE=1; shift ;;
    --skip-web)      SKIP_WEB=1; shift ;;
    --check-only)    CHECK_ONLY=1; shift ;;
    --infra-dir)     INFRA_DIR="$2"; shift 2 ;;
    --omc-root)      OMC_ROOT="$2"; shift 2 ;;
    --overwrite-etc) OVERWRITE_ETC=1; shift ;;
    --yes)           ASSUME_YES=1; shift ;;
    -h|--help)       sed -n '3,49p' "$0"; exit 0 ;;
    *)               die "未知参数：$1（-h 查看用法）" ;;
  esac
done

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

confirm() {
  # confirm <prompt>
  [ "$ASSUME_YES" = 1 ] && return 0
  local yn
  read -rp "$1 [Y/n] " yn
  case "${yn:-Y}" in [Yy]*|"") return 0 ;; *) return 1 ;; esac
}

# =============================================================================
# Step 1. precheck
# =============================================================================
sep "1/10 precheck"

# 工具齐全
for tool in docker tar systemctl sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少工具：$tool（请先装 Docker 等基础工具）" 1
done

# docker compose v2
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  die "未检测到 docker compose v2 / docker-compose v1" 1
fi
log "docker compose 命令：$COMPOSE"

# docker 服务可用
docker info >/dev/null 2>&1 || die "docker 服务不可用，请先 systemctl start docker" 1

# 项目包结构
[ -d "$PKG_ROOT/bin" ]    || die "项目包目录结构异常：缺 bin/（请确认本脚本在解压后的项目包内）" 1
[ -d "$PKG_ROOT/etc" ]    || die "项目包目录结构异常：缺 etc/" 1
[ -d "$PKG_ROOT/deploy" ] || die "项目包目录结构异常：缺 deploy/" 1
[ -x "$PKG_ROOT/bin/omcgo-migrate" ] || die "二进制缺失：bin/omcgo-migrate" 1
[ -x "$PKG_ROOT/bin/omcgo-app" ]     || die "二进制缺失：bin/omcgo-app" 1

# 项目版本号（VERSION 文件由 build-release.sh 写入）
VERSION="$(awk -F= '/^project_version=/{print $2}' "$PKG_ROOT/VERSION" 2>/dev/null || echo unknown)"
log "项目版本：$VERSION"

# infra 目录（除非 --skip-infra）
if [ "$SKIP_INFRA" = 0 ]; then
  [ -d "$INFRA_DIR" ] || die "基础设施目录不存在：$INFRA_DIR
  · 首次部署需先：cd $INFRA_DIR && tar -xJf omc-infra-<版本>-<架构>.tar.xz --strip-components=1
  · 或加 --skip-infra 跳过基础设施步" 1
  [ -d "$INFRA_DIR/images" ] || die "基础设施目录缺 images/：$INFRA_DIR/images" 1
fi

log "precheck 通过"

if [ "$CHECK_ONLY" = 1 ]; then
  log "--check-only：不做任何修改，退出"
  exit 0
fi

# =============================================================================
# Step 2. 建立 OMC 目录布局 + current 软链
# =============================================================================
sep "2/10 建立目录结构 + current 软链"

mkdir -p "$OMC_ROOT/releases" "$OMC_ROOT/etc" "$OMC_ROOT/run/logs" "$OMC_ROOT/packages"

RELEASE_DIR="$OMC_ROOT/releases/$VERSION"
if [ -d "$RELEASE_DIR" ] && [ "$(readlink -f "$PKG_ROOT" 2>/dev/null)" != "$(readlink -f "$RELEASE_DIR" 2>/dev/null)" ]; then
  warn "已存在版本目录 $RELEASE_DIR，将覆盖（旧文件 → .bak.<时间戳>）"
  confirm "继续吗？" || die "用户取消" 1
  mv "$RELEASE_DIR" "$RELEASE_DIR.bak.$(date +%Y%m%d%H%M%S)"
fi

# 如果不是直接在版本目录里跑，rsync 到版本目录
if [ "$(readlink -f "$PKG_ROOT")" != "$(readlink -f "$RELEASE_DIR")" ]; then
  log "复制项目包到 $RELEASE_DIR ..."
  mkdir -p "$RELEASE_DIR"
  cp -a "$PKG_ROOT/." "$RELEASE_DIR/"
fi

# 实例配置：首次 = 复制模板；非首次 = 默认不覆盖以保护已改口令，需覆盖走以下三条路径：
#   1） --overwrite-etc                              → 直接覆盖（CI 友好）
#   2） 交互模式（未 --yes）并选 y                  → 覆盖
#   3）其他                                          → 保留不覆盖
# 覆盖前会先把原 etc/ 整个 mv 到 etc.bak.<时间戳>，可随时 diff/回滚。
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

# =============================================================================
# Step 3. load 基础镜像（仅首次 / 强制 --skip-infra=0）
# =============================================================================
if [ "$SKIP_INFRA" = 0 ]; then
  sep "3/10 load 基础镜像"
  for tar in "$INFRA_DIR/images"/*.tar; do
    [ -f "$tar" ] || continue
    log "docker load < $(basename "$tar")"
    docker load -i "$tar"
  done
else
  sep "3/10 load 基础镜像（--skip-infra 跳过）"
fi

# =============================================================================
# Step 4. 默认口令检查（PostgreSQL / MinIO）
# =============================================================================
sep "4/10 默认口令安全检查"

CONF="$OMC_ROOT/etc/app.prod.yaml"
INFRA_COMPOSE="$OMC_ROOT/current/deploy/docker-compose.infra.yml"

if grep -qE 'omcgo123|minioadmin' "$INFRA_COMPOSE" 2>/dev/null; then
  warn "检测到 docker-compose.infra.yml 含默认口令 omcgo123 / minioadmin"
  warn "  → 生产环境务必改强口令，并与 $CONF 内的 dsn / minio 配置保持一致"
  warn "  → 修改文件：$INFRA_COMPOSE 与 $OMC_ROOT/etc/*.prod.yaml"
  confirm "已知风险，继续部署？" || die "用户取消，请先改口令" 1
fi

# =============================================================================
# Step 5. infra compose up + 等就绪
# =============================================================================
if [ "$SKIP_INFRA" = 0 ]; then
  sep "5/10 启动基础设施容器（postgres / redis / nats / minio）"
  ( cd "$OMC_ROOT/current/deploy" && $COMPOSE -f docker-compose.infra.yml up -d )

  log "等待基础设施 ready（最多 90s）..."
  WAIT=0
  while [ $WAIT -lt 90 ]; do
    sleep 3; WAIT=$((WAIT+3))
    PG_OK=0;  docker exec "$($COMPOSE -f "$OMC_ROOT/current/deploy/docker-compose.infra.yml" ps -q postgres 2>/dev/null || echo dummy)" pg_isready -U omcgo >/dev/null 2>&1 && PG_OK=1
    RD_OK=0;  docker exec "$($COMPOSE -f "$OMC_ROOT/current/deploy/docker-compose.infra.yml" ps -q redis 2>/dev/null    || echo dummy)" redis-cli ping >/dev/null 2>&1 && RD_OK=1
    [ "$PG_OK" = 1 ] && [ "$RD_OK" = 1 ] && break
    echo "  ... ${WAIT}s (PG=$PG_OK RD=$RD_OK)"
  done
  if [ "$PG_OK" != 1 ] || [ "$RD_OK" != 1 ]; then
    die "基础设施 90s 内未就绪：PG=$PG_OK RD=$RD_OK
    手动检查：$COMPOSE -f $OMC_ROOT/current/deploy/docker-compose.infra.yml ps
              docker logs <容器名>" 2
  fi
  log "基础设施已就绪 (PG / Redis)"
else
  sep "5/10 启动基础设施（--skip-infra 跳过）"
fi

# =============================================================================
# Step 6. 执行 db migrate
# =============================================================================
if [ "$SKIP_MIGRATE" = 0 ]; then
  sep "6/10 执行 db migrate"
  if ( cd "$OMC_ROOT/current" && ./bin/omcgo-migrate --config "$OMC_ROOT/etc/app.prod.yaml" up ); then
    log "migrate 成功"
  else
    die "migrate 失败，检查 db 连接 / 迁移文件" 3
  fi
else
  sep "6/10 db migrate（--skip-migrate 跳过）"
fi

# =============================================================================
# Step 7. 执行 db seed（仅首次部署时跑一次）
# =============================================================================
if [ "$SKIP_MIGRATE" = 0 ] && [ -x "$OMC_ROOT/current/bin/omcgo-seed" ]; then
  sep "7/10 执行 db seed（首次部署）"
  SEED_MARK="$OMC_ROOT/etc/.seed.done"
  if [ -f "$SEED_MARK" ]; then
    log "已有 $SEED_MARK，跳过 seed（如需重灌请删除该文件再跑）"
  else
    if ( cd "$OMC_ROOT/current" && ./bin/omcgo-seed --config "$OMC_ROOT/etc/app.prod.yaml" ); then
      touch "$SEED_MARK"
      log "seed 成功"
    else
      warn "seed 失败（部分种子可能已存在，不影响主流程；如确需排查请看 db 日志）"
    fi
  fi
else
  sep "7/10 db seed（跳过）"
fi

# =============================================================================
# Step 8. 安装 / 启用 systemd 单元（app / acs / worker）
# =============================================================================
sep "8/10 安装 systemd 单元（app / acs / worker，开机自启）"

for svc in omcgo-app omcgo-acs omcgo-worker; do
  SRC="$OMC_ROOT/current/deploy/systemd/$svc.service"
  DST="/etc/systemd/system/$svc.service"
  [ -f "$SRC" ] || die "缺 systemd 单元模板：$SRC" 1
  if [ -f "$DST" ] && ! cmp -s "$SRC" "$DST"; then
    cp -a "$DST" "$DST.bak.$(date +%Y%m%d%H%M%S)"
    log "备份原单元 → $DST.bak.<时间戳>"
  fi
  install -m 0644 "$SRC" "$DST"
done
systemctl daemon-reload
for svc in omcgo-app omcgo-acs omcgo-worker; do
  log "systemctl enable --now $svc"
  systemctl enable --now "$svc"
done

# =============================================================================
# Step 9. web compose up
# =============================================================================
if [ "$SKIP_WEB" = 0 ]; then
  sep "9/10 启动 web 容器（nginx 静态资源 + 反代）"
  ( cd "$OMC_ROOT/current/deploy" && $COMPOSE -f docker-compose.web.yml up -d )
else
  sep "9/10 启动 web（--skip-web 跳过）"
fi

# =============================================================================
# Step 10. healthcheck
# =============================================================================
sep "10/10 健康检查"

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
echo
log "访问地址："
log "  · Web UI ：       http://<服务器IP>:8080"
log "  · MinIO Console：http://<服务器IP>:9001"
log "  · 健康检查：       bash $OMC_ROOT/current/deploy/healthcheck.sh"
echo
log "初始账号（首次登录强制改）："
log "  · Web UI：    admin / admin123"
log "  · MinIO：     minioadmin / minioadmin"
log "  · PostgreSQL：omcgo / omcgo123"
echo

if [ "$HEALTH_OK" != 1 ]; then
  die "健康检查未全通过（部分服务异常），请按上面 healthcheck 输出排查" 4
fi
log "全部 OK 🎉"

#!/usr/bin/env bash
# =============================================================================
# OMC 一键部署脚本（安装 / 升级）— 内网交付侧（全 docker compose 部署）
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
# ★ 卸载请用同目录的 uninstall.sh（本脚本只负责安装 / 升级）。
#
# 用法：
#   sudo bash deploy/install.sh                          # 全套首次部署 / 升级
#   sudo bash deploy/install.sh --skip-infra             # 已部署 infra 镜像，仅升级业务镜像
#   sudo bash deploy/install.sh --skip-migrate           # 不跑 migrate / seed
#   sudo bash deploy/install.sh --skip-web               # 不起 web 容器
#   sudo bash deploy/install.sh --skip-monitoring        # 不起监控栈
#   sudo bash deploy/install.sh --check-only             # 仅检查环境，不做修改
#   sudo bash deploy/install.sh --infra-dir /opt/omc/infra   # 自定义 infra 目录
#   sudo bash deploy/install.sh --overwrite-etc          # 用新包模板覆盖 /opt/omc/etc
#   sudo bash deploy/install.sh -h | --help              # 本帮助
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

log()  { echo -e "\033[1;32m[install]\033[0m $*"; }
warn() { echo -e "\033[1;33m[install][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[install][错误]\033[0m $*" >&2; exit "${2:-1}"; }
sep()  { echo -e "\033[1;34m──────────────── $* ────────────────\033[0m"; }

# 凭证治本（#175）纯函数库（rand_hex / secrets_get_val / secrets_is_default_value /
# secrets_generate_to / secrets_import_to / secrets_apply_to_env），随包同目录交付。
if [ -f "$DEPLOY_DIR/secrets-lib.sh" ]; then
  . "$DEPLOY_DIR/secrets-lib.sh"
else
  die "缺 $DEPLOY_DIR/secrets-lib.sh（凭证生成库，由 build-release.sh 随包发布）" 1
fi

if [ -f "$DEPLOY_DIR/data-upgrade-lib.sh" ]; then
  . "$DEPLOY_DIR/data-upgrade-lib.sh"
else
  die "缺 $DEPLOY_DIR/data-upgrade-lib.sh（data 升级保护库，由 build-release.sh 随包发布）" 1
fi
if [ -f "$DEPLOY_DIR/config-upgrade-lib.sh" ]; then
  . "$DEPLOY_DIR/config-upgrade-lib.sh"
else
  die "缺 $DEPLOY_DIR/config-upgrade-lib.sh（实例配置升级库，由 build-release.sh 随包发布）" 1
fi
if [ -f "$DEPLOY_DIR/storage-paths-lib.sh" ]; then
  . "$DEPLOY_DIR/storage-paths-lib.sh"
else
  die "缺 $DEPLOY_DIR/storage-paths-lib.sh（有状态服务数据路径库，由 build-release.sh 随包发布）" 1
fi
if [ -f "$DEPLOY_DIR/resource-env-lib.sh" ]; then
  . "$DEPLOY_DIR/resource-env-lib.sh"
else
  die "缺 $DEPLOY_DIR/resource-env-lib.sh（完整资源规划契约库）" 1
fi
if [ -f "$DEPLOY_DIR/resource-plan-metrics.sh" ]; then
  . "$DEPLOY_DIR/resource-plan-metrics.sh"
else
  die "缺 $DEPLOY_DIR/resource-plan-metrics.sh（资源计划 Prometheus 指标生成器）" 1
fi
if [ -f "$DEPLOY_DIR/monitoring-profile-lib.sh" ]; then
  . "$DEPLOY_DIR/monitoring-profile-lib.sh"
else
  die "缺 $DEPLOY_DIR/monitoring-profile-lib.sh（监控部署模式状态库，由 build-release.sh 随包发布）" 1
fi

# 升级时 deploy/.env 里【运维自定义】的键 —— 跨版本继承,不被新包默认值覆盖。
# 注：6 个密钥键虽仍在此列（升级时把上一版有效凭证带进新 .env，供 ensure_secrets 首迁导入），
# 但密钥的【唯一权威源】是 etc/secrets.env —— ensure_secrets 在 source/起 infra 前用它覆盖 .env，
# 故 .env.saved 即便被某次默认口令安装写脏，也不再污染密钥（secrets.env 一经生成永不重生成）。
# 【版本相关】键(PROJECT_VERSION / IMAGE_*)不在此列,始终用新包值。
# 注：POSTGRES_TSDB_USER/PASSWORD/DB（时序库凭据，#347）跨版本继承；TSDB_HOST 是 compose 服务名
# （随包固定值），故【不】列入继承白名单，始终用新包值。
ENV_PRESERVE_KEYS="POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_TSDB_USER POSTGRES_TSDB_PASSWORD POSTGRES_TSDB_DB MINIO_ROOT_USER MINIO_ROOT_PASSWORD GRAFANA_ADMIN_USER GRAFANA_ADMIN_PASSWORD OMCGO_JWT_SECRET OMC_SHARED_SECRET OMC_PUBLIC_HOST POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH"

# merge_env_preserve <prev_env> <new_env>
# 升级继承:以新包 .env 为基底(拿到新镜像 tag),把上一版 .env 中白名单键的值
# (口令 / JWT / OMC_PUBLIC_HOST)覆盖进来 —— 解决"升级丢运维配置"(§9.5)。
# 值含特殊字符(base64 JWT、含 = 的口令等)由 awk 当数据处理,不经 shell 展开,安全。
# 首次部署(无 prev)直接用新包默认值。
merge_env_preserve() {
  local prev="$1" new="$2" tmp
  [ -f "$new" ] || return 0
  if [ -z "$prev" ] || [ ! -f "$prev" ]; then
    log ".env:首次部署(无上一版),使用交付包默认值 —— 记得在 $new 填 OMC_PUBLIC_HOST 与强口令"
    return 0
  fi
  tmp="$(mktemp)" || { warn ".env 合并:mktemp 失败,沿用新包默认值"; return 0; }
  if awk -v keys="$ENV_PRESERVE_KEYS" '
      BEGIN { n=split(keys, A, " "); for (i=1;i<=n;i++) want[A[i]]=1 }
      FNR==NR {                                    # 第一份 = 上一版(prev)
        if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          p=index($0,"="); k=substr($0,1,p-1)
          if (k in want) { val[k]=substr($0,p+1); have[k]=1 }
        }
        next
      }
      {                                            # 第二份 = 新包(new,基底)
        if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          p=index($0,"="); k=substr($0,1,p-1)
          if ((k in want) && (k in have)) { print k"="val[k]; seen[k]=1; next }
          if (k in want) seen[k]=1
        }
        print
      }
      END { for (k in have) if (!(k in seen)) print k"="val[k] }
    ' "$prev" "$new" > "$tmp"; then
    cat "$tmp" > "$new"        # 覆写内容,保留 new 原 inode / 权限
    rm -f "$tmp"
    log ".env:已从上一版继承运维自定义值(口令 / JWT / OMC_PUBLIC_HOST);镜像 tag 用新包。如需改值,编辑 $new 后重启业务容器"
  else
    rm -f "$tmp"
    warn ".env 合并失败,沿用交付包默认值;请手动核对 $new 的 OMC_PUBLIC_HOST 与口令"
  fi
}

# ensure_secrets() 定义在 secrets-lib.sh（依赖 log/warn/docker/OMC_ROOT/COMPOSE_PROJECT，
# 上面已 source；这些变量在调用点 Step 4 之前均已就绪）。

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
    -h|--help)         sed -n '3,52p' "$0"; exit 0 ;;
    --uninstall)       die "卸载请用 uninstall.sh：sudo bash $DEPLOY_DIR/uninstall.sh -h" ;;
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

# resolve_resource_env_candidate —— 安装前资源契约候选优先级：
#   1) 新交付包内由 plan-resources.sh 生成的文件；
#   2) 当前生效 release；
#   3) uninstall 保留的 saved 快照。
# 一旦较高优先级候选存在，就必须校验该文件，禁止因其非法而回退到较旧候选。
resolve_resource_env_candidate() {
  if [ -f "$PKG_ROOT/deploy/resources.env" ]; then
    printf '%s\n' "$PKG_ROOT/deploy/resources.env"
  elif [ -f "$OMC_ROOT/current/deploy/resources.env" ]; then
    printf '%s\n' "$OMC_ROOT/current/deploy/resources.env"
  elif [ -f "$OMC_ROOT/etc/resources.env.saved" ]; then
    printf '%s\n' "$OMC_ROOT/etc/resources.env.saved"
  else
    return 1
  fi
}

# heal_main_pg_timescaledb_downgrade —— 升级自愈（#347 主库 timescaledb → 纯 PG 降级）
# 旧版 release 主库用 timescaledb 镜像初始化，卷内 postgresql.conf 写死
# shared_preload_libraries='timescaledb'；本版主库降级 postgres:16-alpine（无该库），旧卷
# 直接起会 FATAL: could not access file "timescaledb"。三条件全满足才动手（否则跳过=幂等）：
#   ① 目标主库镜像 $IMAGE_POSTGRES 是纯 PG（不含 timescale）；
#   ② 主库卷 ${COMPOSE_PROJECT}_pgdata 已存在（存量升级，非全新装）；
#   ③ 卷内 postgresql.conf 有【未注释】的 timescaledb 预加载。
# 动作：用【包内】timescaledb 镜像临时挂卷拉起 → DROP EXTENSION timescaledb CASCADE（删主库
#   残留时序超表，时序数据已分离到 postgres-tsdb）→ 注释 conf 预加载行 → 停容器。
# 离线友好：只用包内已 docker load 的 $IMAGE_POSTGRES_TSDB，绝不拉 alpine 等外网镜像。
heal_main_pg_timescaledb_downgrade() {
  local vol="${COMPOSE_PROJECT}_pgdata"
  local heal_img="${IMAGE_POSTGRES_TSDB:-timescale/timescaledb:2.25.2-pg16}"
  case "${IMAGE_POSTGRES:-}" in *timescale*) return 0 ;; esac          # 目标本就 timescaledb，无陷阱
  docker volume inspect "$vol" >/dev/null 2>&1 || return 0             # 全新装，无存量卷
  local has_pre
  has_pre="$(docker run --rm --entrypoint sh -v "$vol":/d:ro "$heal_img" \
    -c "grep -E '^[[:space:]]*shared_preload_libraries[[:space:]]*=.*timescaledb' /d/postgresql.conf 2>/dev/null" 2>/dev/null || true)"
  [ -n "$has_pre" ] || return 0                                        # 已剥离/本就纯 PG → 跳过

  warn "检测到【存量主库卷被 timescaledb 初始化】，而本版主库已降级纯 PG（$IMAGE_POSTGRES）。"
  warn "  直接启动会 FATAL: could not access file \"timescaledb\"。"
  warn "  自愈将剥离 timescaledb：DROP EXTENSION ... CASCADE（删主库残留时序超表——这些时序数据"
  warn "  已物理分离到 postgres-tsdb 实例）+ 注释 postgresql.conf 预加载行。业务表（普通表）不受影响。"
  confirm "确认对主库卷 $vol 执行 timescaledb 剥离自愈？" || \
    die "用户取消自愈。如需保数据，可临时把 deploy/.env 的 IMAGE_POSTGRES 改回 timescaledb 镜像后重装。" 1

  local cname="${COMPOSE_PROJECT}-pgheal" db="${POSTGRES_DB:-omcgo}" usr="${POSTGRES_USER:-omcgo}"
  docker rm -f "$cname" >/dev/null 2>&1 || true
  # 重试场景下，上一轮 7.1 可能已建出 restart-loop 的 compose 主库容器在持卷，先移除释放卷（7.1 会重建）。
  docker rm -f "${COMPOSE_PROJECT}-postgres-1" >/dev/null 2>&1 || true
  log "自愈：用 $heal_img 临时挂卷启动主库 ..."
  docker run -d --name "$cname" -e POSTGRES_PASSWORD=heal \
    -v "$vol":/var/lib/postgresql/data "$heal_img" >/dev/null 2>&1 \
    || die "自愈：临时容器启动失败（$cname）" 3
  local ok=0 i
  for i in $(seq 1 30); do
    docker exec "$cname" pg_isready -U "$usr" >/dev/null 2>&1 && { ok=1; break; }
    sleep 1
  done
  if [ "$ok" != 1 ]; then
    docker logs --tail 20 "$cname" 2>&1 || true
    docker rm -f "$cname" >/dev/null 2>&1 || true
    die "自愈：临时库 30s 未就绪，无法剥离" 3
  fi
  docker exec "$cname" psql -U "$usr" -d "$db" -c "DROP EXTENSION IF EXISTS timescaledb CASCADE;" >/dev/null 2>&1 \
    || warn "自愈：DROP EXTENSION 返回非零（可能已无扩展），继续清理 conf"
  docker exec "$cname" sh -c "sed -i.bak-347 's/^[[:space:]]*shared_preload_libraries/#347 &/; s/^[[:space:]]*timescaledb\\./#347 &/' /var/lib/postgresql/data/postgresql.conf" 2>/dev/null || true
  docker exec "$cname" sh -c "test -f /var/lib/postgresql/data/postgresql.auto.conf && sed -i '/timescaledb/d' /var/lib/postgresql/data/postgresql.auto.conf || true" 2>/dev/null || true
  docker stop "$cname" >/dev/null 2>&1 || true
  docker rm -f "$cname" >/dev/null 2>&1 || true
  log "自愈完成：主库卷已剥离 timescaledb，可在纯 PG（$IMAGE_POSTGRES）上启动。"
}

# =============================================================================
# Step 1. precheck
# =============================================================================
sep "1/9 precheck"

# 工具齐全
for tool in docker tar sha256sum; do
  command -v "$tool" >/dev/null 2>&1 || die "缺少工具：$tool（请先装 Docker 等基础工具）" 1
done

# docker compose 命令选择
# 2026-05-29:V1 Python (apt 装的 docker-compose) 不识别 compose v3 写法,部署
# 时报 "Unsupported config option for services/networks/volumes"。优先用
# `docker compose` V2 plugin;若只有 V1 standalone,尝试从 infra bundle 自动装
# V2 plugin,装不上则硬退出并给出修复指引。
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_VER="$(docker-compose version --short 2>/dev/null || true)"
  case "$COMPOSE_VER" in
    2.*)
      # standalone V2 binary,OK 直接用
      COMPOSE="docker-compose"
      ;;
    1.*|"")
      log "检测到 docker-compose V1 (Python: $COMPOSE_VER),不兼容 compose v3 文件,尝试自动安装 V2 插件 ..."
      BUNDLE_V2=""
      for cand in "$INFRA_DIR/docker/docker-compose" "$PKG_ROOT/../infra/docker/docker-compose"; do
        if [ -f "$cand" ]; then BUNDLE_V2="$cand"; break; fi
      done
      if [ -n "$BUNDLE_V2" ]; then
        install -d /usr/local/lib/docker/cli-plugins
        install -m 0755 "$BUNDLE_V2" /usr/local/lib/docker/cli-plugins/docker-compose
        if docker compose version >/dev/null 2>&1; then
          COMPOSE="docker compose"
          log "自动安装 V2 插件成功:$(docker compose version | head -1)"
        else
          die "V2 插件安装到 cli-plugins/ 后 \`docker compose version\` 仍失败,请手动排查 dockerd 状态" 1
        fi
      else
        die "docker-compose 是 V1 (Python),无法解析 compose v3 文件;
        infra bundle 中未找到 V2 binary($INFRA_DIR/docker/docker-compose 不存在)。
        修复方法二选一:
          1. 重跑 sudo bash /opt/omc/infra/docker/install-docker.sh
             (V2 plugin 会自动装到 /usr/local/lib/docker/cli-plugins/)
          2. 手动 apt install docker-compose-plugin
                 (Ubuntu 22.04+ docker 官方源)" 1
      fi
      ;;
    *)
      die "无法识别 docker-compose 版本:$COMPOSE_VER" 1
      ;;
  esac
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

# 资源契约是部署必需输入。必须在 --check-only 退出、current 切换和任何容器重启之前
# 校验真正会被本次安装采用的候选，不能等到 Step 6 组装 Compose 才发现旧三行文件。
RESOURCE_ENV_CANDIDATE="$(resolve_resource_env_candidate)" ||
  die "未找到 resources.env；请先运行 bash $PKG_ROOT/deploy/plan-resources.sh，禁止静默回退 Compose 默认限额" 1
resource_env_validate "$RESOURCE_ENV_CANDIDATE" ||
  die "resources.env 不是完整资源规划：${RESOURCE_ENV_CANDIDATE}；请重新运行 plan-resources.sh" 1
log "资源规划预检通过：$RESOURCE_ENV_CANDIDATE"

# 兜底：旧版 build-release.sh 在 umask=027 机器上构建时 monitoring/ 配置会落 0640，
# prometheus/loki/tempo/alertmanager 等非 root 容器读不动直接 fail。
# 新版 build-release.sh 已 baked chmod a+rX 到 tar；这里再 defensive 兜一遍。
if [ "$SKIP_MONITORING" = 0 ] && [ -d "$PKG_ROOT/deploy/monitoring" ]; then
  chmod -R a+rX "$PKG_ROOT/deploy/monitoring"
fi

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

# 注:三库导入XML重构 Phase 2 后,custom XML 不再用独立 *-custom 目录(单目录 + sidecar)。
#     整个 data 目录的播种 / 升级反向合并见下方 RELEASE_DIR 就绪后的「data 外置」步骤。

# 升级前快照当前生效的 deploy/.env,供下方合并继承运维自定义值(口令/JWT/OMC_PUBLIC_HOST)。
# 必须在可能 mv/覆盖旧版本目录之前抓取 —— current 软链此刻仍指向上一版;首次部署无 current → 空。
# 兼容 uninstall.sh：若 current 已被卸载移除,退而读卸载时保存的凭据 $OMC_ROOT/etc/.env.saved。
PREV_ENV_SNAPSHOT=""
if [ -f "$OMC_ROOT/current/deploy/.env" ]; then
  PREV_ENV_SNAPSHOT="$(mktemp)" || PREV_ENV_SNAPSHOT=""
  [ -n "$PREV_ENV_SNAPSHOT" ] && { cp "$OMC_ROOT/current/deploy/.env" "$PREV_ENV_SNAPSHOT" 2>/dev/null || PREV_ENV_SNAPSHOT=""; }
elif [ -f "$OMC_ROOT/etc/.env.saved" ]; then
  log ".env:未见 current/deploy/.env,改用 uninstall.sh 保存的凭据 $OMC_ROOT/etc/.env.saved 继承口令/JWT"
  PREV_ENV_SNAPSHOT="$(mktemp)" || PREV_ENV_SNAPSHOT=""
  [ -n "$PREV_ENV_SNAPSHOT" ] && { cp "$OMC_ROOT/etc/.env.saved" "$PREV_ENV_SNAPSHOT" 2>/dev/null || PREV_ENV_SNAPSHOT=""; }
fi

# resources.env(plan-resources.sh 生成的资源限额,operator 独有,不随交付包)同样快照:
# 它是独立文件、无包内默认值可合并,故整文件继承(而非走 ENV_PRESERVE_KEYS 键级合并)。
PREV_RESOURCES_SNAPSHOT=""
if [ "$RESOURCE_ENV_CANDIDATE" = "$PKG_ROOT/deploy/resources.env" ]; then
  :
elif [ "$RESOURCE_ENV_CANDIDATE" = "$OMC_ROOT/current/deploy/resources.env" ]; then
  PREV_RESOURCES_SNAPSHOT="$(mktemp)" || PREV_RESOURCES_SNAPSHOT=""
  [ -n "$PREV_RESOURCES_SNAPSHOT" ] && { cp "$RESOURCE_ENV_CANDIDATE" "$PREV_RESOURCES_SNAPSHOT" 2>/dev/null || PREV_RESOURCES_SNAPSHOT=""; }
elif [ "$RESOURCE_ENV_CANDIDATE" = "$OMC_ROOT/etc/resources.env.saved" ]; then
  PREV_RESOURCES_SNAPSHOT="$(mktemp)" || PREV_RESOURCES_SNAPSHOT=""
  [ -n "$PREV_RESOURCES_SNAPSHOT" ] && { cp "$RESOURCE_ENV_CANDIDATE" "$PREV_RESOURCES_SNAPSHOT" 2>/dev/null || PREV_RESOURCES_SNAPSHOT=""; }
fi
[ "$RESOURCE_ENV_CANDIDATE" = "$PKG_ROOT/deploy/resources.env" ] ||
  [ -n "$PREV_RESOURCES_SNAPSHOT" ] ||
  die "resources.env 快照失败，未切换 current；请检查临时目录空间与文件权限" 1

# 保存上一版随包 builtin 基线，用于区分“未修改的旧 builtin”与“运维在原 builtin
# 文件上做过的扩展”。仅看 .custom 不够：指标库 force 覆盖 GSM.xml/BSC 平台时为了
# 保持 builtin 不可删除不会写 sidecar，但这种 75→167 条的现网扩展升级时仍必须保留。
PREV_DATA_BASELINE=""
if [ -d "$OMC_ROOT/current/data" ]; then
  PREV_DATA_BASELINE="$(mktemp -d)" || PREV_DATA_BASELINE=""
  if [ -n "$PREV_DATA_BASELINE" ]; then
    cp -a "$OMC_ROOT/current/data/." "$PREV_DATA_BASELINE/" 2>/dev/null \
      || { warn "上一版 builtin 基线快照失败;将沿用旧版刷新判定"; rm -rf "$PREV_DATA_BASELINE"; PREV_DATA_BASELINE=""; }
  fi
fi

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

# 升级继承:把上一版 .env 的运维自定义值(口令/JWT/OMC_PUBLIC_HOST)合并进新包 .env(§9.5)。
# 新包仍负责版本相关键(IMAGE_*/PROJECT_VERSION)。首次部署无快照 → 用新包默认。
merge_env_preserve "$PREV_ENV_SNAPSHOT" "$RELEASE_DIR/deploy/.env"
[ -n "$PREV_ENV_SNAPSHOT" ] && rm -f "$PREV_ENV_SNAPSHOT" 2>/dev/null || true

# 资源限额 resources.env 整文件继承到新 release(交付包不含此文件,故仅在上一版存在时拷入)。
if [ -n "$PREV_RESOURCES_SNAPSHOT" ] && [ ! -f "$RELEASE_DIR/deploy/resources.env" ]; then
  cp "$PREV_RESOURCES_SNAPSHOT" "$RELEASE_DIR/deploy/resources.env" 2>/dev/null ||
    die "resources.env 继承复制失败，未切换 current：$RELEASE_DIR/deploy/resources.env" 1
  log "resources.env:已从上一版继承资源限额(plan-resources.sh 调优值不丢)"
fi
[ -n "$PREV_RESOURCES_SNAPSHOT" ] && rm -f "$PREV_RESOURCES_SNAPSHOT" 2>/dev/null || true

# 防 TOCTOU、复制故障和目标目录陈旧文件：对即将成为 current 的实际文件再校验一次。
[ -f "$RELEASE_DIR/deploy/resources.env" ] ||
  die "复制/继承后的 resources.env 缺失，未切换 current；请重新运行 plan-resources.sh" 1
resource_env_validate "$RELEASE_DIR/deploy/resources.env" ||
  die "复制/继承后的 resources.env 未通过完整资源规划校验，未切换 current：$RELEASE_DIR/deploy/resources.env" 1

# ── data 外置 + 升级反向合并(三库导入XML重构 Phase 2,D1/D2)───────────────────
# 模型 B:整个 data 目录外置到 $OMC_ROOT/data,bind-mount(RW)进 app/worker;
# 镜像不再 COPY data($RELEASE_DIR/data 是本次发版随包的 builtin 基线)。
#   · 首次部署:播种整包 data → $OMC_ROOT/data(硬依赖 —— 失败则字典为空,中止)。
#   · 升级    :先快照 $OMC_ROOT/data → .bak.<ts>(回滚点),再"现网赢、新版补充"
#               反向合并(cp -an no-clobber:仅补现网缺失的 builtin 文件,现网已上传的
#               自定义 XML + .custom sidecar + 改过的文件一律保留)。
NEW_DATA="$RELEASE_DIR/data"
if [ ! -d "$NEW_DATA" ]; then
  die "交付包缺少 data/ 目录($NEW_DATA);模型 B 硬依赖字典播种,无法继续" 4
fi
if [ ! -d "$OMC_ROOT/data" ] || [ -z "$(ls -A "$OMC_ROOT/data" 2>/dev/null)" ]; then
  log "首次部署：播种 data 基线 → $OMC_ROOT/data"
  mkdir -p "$OMC_ROOT/data"
  cp -a "$NEW_DATA/." "$OMC_ROOT/data/" || die "播种 data 失败;字典将为空,中止部署" 4
else
  DATA_SNAP="$OMC_ROOT/data.bak.$(date +%Y%m%d%H%M%S)"
  log "升级：快照现网 data → $DATA_SNAP(回滚用)"
  cp -a "$OMC_ROOT/data" "$DATA_SNAP" || warn "快照 data 失败(磁盘满?);继续合并但无回滚点"
  log "升级：反向合并(现网赢、新版补充)新版 builtin → $OMC_ROOT/data"
  cp -an "$NEW_DATA/." "$OMC_ROOT/data/" || warn "反向合并 data 出现错误;请人工核对 $OMC_ROOT/data"
  # builtin 刷新(#154):cp -an 只补缺失、不更新已存在文件 → 内置字典(如 BSC 网关→BSC 产品
  # 改名)升级不生效,且 dictloader 会用过时 host XML 把旧值 UPSERT 回来。这里对新包 builtin
  # 文件做差异覆盖:host 无对应 .custom sidecar(=运维未自定义)且内容有变 → 用新版覆盖;
  # 有 .custom(运维上传/改过)一律保留。sidecar 约定见 internal/*/source.go::CustomMarkerSuffix。
  _refreshed=0
  while IFS= read -r -d '' _nf; do
    _rel="${_nf#"$NEW_DATA"/}"
    case "$_rel" in *.custom) continue ;; esac           # 新包不应含 sidecar,防御性跳过
    [ -f "$OMC_ROOT/data/$_rel.custom" ] && continue       # 运维自定义,保留不覆盖
    _live="$OMC_ROOT/data/$_rel"
    _previous="${PREV_DATA_BASELINE:+$PREV_DATA_BASELINE/$_rel}"
    if [ -n "$_previous" ] && [ -f "$_previous" ] && ! cmp -s "$_live" "$_previous"; then
      log "  · 保留运维修改的 builtin 字典:$_rel"
      continue
    fi
    if should_refresh_builtin "$_nf" "$_live" "$_previous"; then
      if cp -a "$_nf" "$OMC_ROOT/data/$_rel"; then
        log "  · 刷新 builtin 字典:$_rel"; _refreshed=$((_refreshed + 1))
      else
        warn "  · 刷新 builtin 失败:$_rel(请人工核对)"
      fi
    fi
  done < <(find "$NEW_DATA" -type f -print0)
  [ "$_refreshed" -gt 0 ] && log "升级:刷新 $_refreshed 个 builtin 字典文件(运维自定义 .custom 已保留)"
fi
[ -n "$PREV_DATA_BASELINE" ] && rm -rf "$PREV_DATA_BASELINE" 2>/dev/null || true
# 容器(UID 10001 = Dockerfile 内 omcgo 非 root 用户)需可写 data:
# 上传/删除 XML、写 .custom sidecar、worker 清理过期备份与孤儿 sidecar。
chown -R 10001:10001 "$OMC_ROOT/data" 2>/dev/null \
  || warn "chown 10001:10001 $OMC_ROOT/data 失败(UID 不存在 host 上属正常);容器内仍以 10001 写入"

# 单目录 + sidecar 后不再使用独立 *-custom 目录。issue #206:旧版自定义 KPI 指标库 /
# 告警定义 / 参数映射仍住在这些旧 *-custom 目录,直接 rm -rf 会丢用户数据(reload 找不到
# 文件 → 公式被删不重插 → KPI 不解析)。改为"先抢救迁移、再删空目录":
#   · 遍历旧目录内每个 *.xml,按其相对子路径 cp -an(no-clobber:现网单目录已有同名文件赢,
#     不覆盖)到对应单目录;
#   · 迁移成功的文件 touch 同名 .custom sidecar(标记为运维自定义,builtin 刷新循环不覆盖、
#     dictloader 视为用户来源);
#   · 全部处理完再删空旧目录。上面的 data.bak.<ts> 快照仍是回滚点。
# 旧目录 → 单目录映射(与 config.dev.yaml 字典目录一致):
#   param-mappings-custom → param-mappings;indicator-library-custom → indicator-library;
#   alarm-definitions-custom → alarm-definitions。
_legacy_custom_map="param-mappings-custom:param-mappings indicator-library-custom:indicator-library alarm-definitions-custom:alarm-definitions"
for _pair in $_legacy_custom_map; do
  _cdir="${_pair%%:*}"
  _target="${_pair##*:}"
  _csrc="$OMC_ROOT/data/$_cdir"
  [ -d "$_csrc" ] || continue
  _rescued=0
  while IFS= read -r -d '' _cf; do
    _crel="${_cf#"$_csrc"/}"                               # 保留旧目录内的相对子路径
    _dst="$OMC_ROOT/data/$_target/$_crel"
    mkdir -p "$(dirname "$_dst")" 2>/dev/null || true
    # 不能用 cp -an 退出码判断"是否真迁入":GNU coreutils / BusyBox 上
    # no-clobber 跳过仍返回 exit 0(只有 macOS BSD cp 返回非零),会给现网
    # 内置文件误打 .custom sidecar(内置文件变 UI 可删 + builtin 刷新永久跳过)。
    # 改为先判目标是否存在,再决定复制 + 打 sidecar。
    if [ -e "$_dst" ]; then
      # 单目录已有同名文件(现网赢),旧自定义副本丢弃即可,绝不补 sidecar。
      # 现网那份是否带 sidecar 由 builtin 刷新循环/上传流程维护,这里不干预。
      log "  · 跳过(现网已有同名,no-clobber):$_cdir/$_crel"
    elif cp -p "$_cf" "$_dst" 2>/dev/null; then
      # 真正迁入了新文件 → 补 .custom sidecar 标记为运维自定义。
      touch "$_dst.custom" 2>/dev/null || true
      log "  · 抢救自定义 XML:$_cdir/$_crel → $_target/$_crel(+.custom)"
      _rescued=$((_rescued + 1))
    else
      warn "  · 抢救自定义 XML 失败:$_cdir/$_crel(请从 data.bak.<ts> 人工核对)"
    fi
  done < <(find "$_csrc" -type f -name '*.xml' -print0)
  [ "$_rescued" -gt 0 ] && log "升级:从 $_cdir 抢救 $_rescued 个自定义 XML → $_target(单目录+sidecar)"
  log "清理废弃 custom 目录:$_csrc(内容已迁移至 $_target)"
  rm -rf "$_csrc"
done

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
    if upgrade_acs_session_limit \
      "$OMC_ROOT/etc/acs.prod.yaml" \
      "$RELEASE_DIR/etc/acs.prod.yaml"; then
      case "${ACS_SESSION_LIMIT_UPGRADE_RESULT:-noop}" in
        migrated)
          log "升级 ACS 会话容量：session.max_concurrent 10000 → 30000（其他实例配置保持不变）"
          ;;
        preserved)
          log "ACS session.max_concurrent 为运维自定义值，升级时保持不变"
          ;;
      esac
    else
      warn "ACS session.max_concurrent 自动迁移失败，保留现网配置；请人工核对新包模板"
    fi
    log "$OMC_ROOT/etc/ 已有实例配置，保留不覆盖（如需覆盖加 --overwrite-etc）"
  fi
fi

# 切 current 软链（原子）
ln -sfn "$RELEASE_DIR" "$OMC_ROOT/current"
log "current → $RELEASE_DIR"

# 凭据落点 $OMC_ROOT/etc/.env.saved：留给 uninstall.sh 删 release 后、下次 install 继承用。
# 始终用当前生效 .env 刷新,保证与正在使用的数据卷口令一致。
cp -f "$RELEASE_DIR/deploy/.env" "$OMC_ROOT/etc/.env.saved" 2>/dev/null || true
# resources.env 同样落 etc/ 快照,供 uninstall→reinstall 继承(与 .env.saved 对称)。
[ -f "$RELEASE_DIR/deploy/resources.env" ] && cp -f "$RELEASE_DIR/deploy/resources.env" "$OMC_ROOT/etc/resources.env.saved" 2>/dev/null || true

# 凭证就位（#175）：必须在下方 Step 4 `source .env`（把 .env 导入 shell 环境，compose 取值
# 优先 shell env）与 Step 7 起 infra 之前执行，否则容器仍拿到 REPLACE_ME/默认值。
ensure_secrets

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
# 必须在 source .env 之后应用：旧 .env 可能显式写了 tracer=true。
# 同时把安装 profile 持久化，供后续独立运行的 svc/healthcheck 使用。
monitoring_profile_apply_install "$ENV_FILE" "$SKIP_MONITORING" ||
  die "无法持久化 monitoring profile 到 $ENV_FILE" 1
storage_prepare_configured_env_paths "$ENV_FILE" ||
  die "有状态服务数据路径校验/创建失败；请检查 $ENV_FILE 中五个 *_DATA_PATH" 1

if [ "$SKIP_INFRA" = 0 ]; then
  INFRA_IMAGES=("$IMAGE_POSTGRES" "${IMAGE_POSTGRES_TSDB:-}" "$IMAGE_REDIS" "$IMAGE_NATS" "$IMAGE_MINIO" "${IMAGE_NGINX:-}")
  MON_IMAGES=("${IMAGE_PROMETHEUS:-}" "${IMAGE_ALERTMANAGER:-}" "${IMAGE_GRAFANA:-}" "${IMAGE_LOKI:-}" "${IMAGE_TEMPO:-}" "${IMAGE_OTELCOL:-}" "${IMAGE_NATS_EXPORTER:-}")

  if images_exist "${INFRA_IMAGES[@]}" "${MON_IMAGES[@]}"; then
    log "基础设施 + 监控镜像已存在，跳过 load，重启容器"
    $COMPOSE -p "$COMPOSE_PROJECT" restart postgres postgres-tsdb redis nats minio 2>/dev/null || true
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

if grep -qE '^(POSTGRES_PASSWORD=omcgo123|MINIO_ROOT_PASSWORD=minioadmin|GRAFANA_ADMIN_PASSWORD=admin|OMC_SHARED_SECRET=dps)$' "$ENV_FILE"; then
  warn "检测到 deploy/.env 含默认口令（POSTGRES_PASSWORD=omcgo123 / MINIO_ROOT_PASSWORD=minioadmin / GRAFANA_ADMIN_PASSWORD=admin / OMC_SHARED_SECRET=dps）"
  warn "  → 生产环境务必改强口令：vi $ENV_FILE"
  warn "  → etc/*.prod.yaml 用 \${VAR} 引用 .env，无需手改；生产凭证校验(GuardProductionSecrets)会拒绝默认值启动"
  confirm "已知风险，继续部署？" || die "用户取消，请先改口令" 1
fi

# =============================================================================
# Step 6. 组装 docker compose 命令（全栈一次 up）
# =============================================================================
sep "6/9 组装 docker compose 命令"

cd "$OMC_ROOT/current/deploy"

# 资源限额：compose 经 --env-file 读取 resources.env(plan-resources.sh 生成)。
# 注意：一旦显式传任一 --env-file，compose 不再自动加载 ./.env，故 .env 也必须显式传。
ENV_FILES=()
[ -f .env ] && ENV_FILES+=( --env-file .env )
[ -f resources.env ] ||
  die "current/deploy/resources.env 缺失；禁止静默回退 Compose 默认限额，请重新运行 plan-resources.sh"
resource_env_validate resources.env ||
  die "resources.env 不是完整资源规划；请重新运行 plan-resources.sh，禁止缺失项静默回退 Compose 默认值"
resource_plan_metrics_write resources.env ||
  die "无法生成 resources.env 对应的 Prometheus 资源计划指标"
ENV_FILES+=( --env-file resources.env )
log "已检出完整 resources.env → 按其资源限额部署（plan-resources.sh 生成）"

COMPOSE_FILES=( -f docker-compose.infra.yml -f docker-compose.app.yml )
[ "$SKIP_WEB" = 0 ]        && COMPOSE_FILES+=( -f docker-compose.web.yml )
[ "$SKIP_MONITORING" = 0 ] && COMPOSE_FILES+=( -f docker-compose.monitoring.yml )

DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${ENV_FILES[@]}" "${COMPOSE_FILES[@]}" )
log "compose 命令：${DC[*]}"

# =============================================================================
# Step 7. 启动基础设施 + 等就绪 → 跑 migrate / seed（一次性容器）
# =============================================================================
sep "7/9 启动基础设施 + 执行 migrate / seed"

# 7.0 升级自愈（#347）：存量主库卷若被 timescaledb 初始化过，先剥离再起纯 PG，
#     否则 postgres:16-alpine 读到卷内 shared_preload_libraries='timescaledb' 会 FATAL。
#     幂等：全新装 / 已剥离 / 目标镜像仍 timescaledb 时自动跳过。
heal_main_pg_timescaledb_downgrade

# 7.1 起基础设施（postgres / postgres-tsdb / redis / nats / minio）
log "启动基础设施容器 ..."
"${DC[@]}" up -d postgres postgres-tsdb redis nats minio

log "等待基础设施 ready（最多 90s）..."
WAIT=0
PG_OK=0; TS_OK=0; RD_OK=0
while [ $WAIT -lt 90 ]; do
  sleep 3; WAIT=$((WAIT+3))
  PG_CID="$("${DC[@]}" ps -q postgres 2>/dev/null || true)"
  TS_CID="$("${DC[@]}" ps -q postgres-tsdb 2>/dev/null || true)"
  RD_CID="$("${DC[@]}" ps -q redis 2>/dev/null || true)"
  if [ -n "$PG_CID" ]; then
    docker exec "$PG_CID" pg_isready -U "${POSTGRES_USER:-omcgo}" >/dev/null 2>&1 && PG_OK=1 || PG_OK=0
  fi
  # 时序库（#347）：用 TSDB 凭据 pg_isready，与主库分别就绪判定。
  if [ -n "$TS_CID" ]; then
    docker exec "$TS_CID" pg_isready -U "${POSTGRES_TSDB_USER:-omcgo}" >/dev/null 2>&1 && TS_OK=1 || TS_OK=0
  fi
  if [ -n "$RD_CID" ]; then
    docker exec "$RD_CID" redis-cli ping >/dev/null 2>&1 && RD_OK=1 || RD_OK=0
  fi
  [ "$PG_OK" = 1 ] && [ "$TS_OK" = 1 ] && [ "$RD_OK" = 1 ] && break
  echo "  ... ${WAIT}s (PG=$PG_OK TSDB=$TS_OK RD=$RD_OK)"
done
if [ "$PG_OK" != 1 ] || [ "$TS_OK" != 1 ] || [ "$RD_OK" != 1 ]; then
  die "基础设施 90s 内未就绪：PG=$PG_OK TSDB=$TS_OK RD=$RD_OK
  手动检查：${DC[*]} ps
            ${DC[*]} logs postgres postgres-tsdb redis" 2
fi
log "基础设施已就绪 (PG / TSDB / Redis)"

# 7.2 验证 omcgo-net 网络已创建
log "验证 omcgo-net 网络 ..."
if ! docker network inspect omcgo-net >/dev/null 2>&1; then
  die "omcgo-net 网络未创建，请检查 docker-compose.infra.yml" 2
fi

# 7.3 migrate-schema（一次性容器，跑完即退；用 .env 中的 dsn）
#     使用 `up --exit-code-from` 而非 `run`，因为 `run` 创建的一次性容器
#     存在 DNS 解析缺陷（无法解析服务名），而 `up` 作为正式服务启动时
#     网络集成完整，DNS 解析正常。保留重试逻辑作为兆底。
if [ "$SKIP_MIGRATE" = 0 ]; then
  log "执行 db migrate（容器：migrate-schema）..."
  MIGRATE_OK=0
  for attempt in 1 2 3; do
    if "${DC[@]}" up --exit-code-from migrate-schema migrate-schema; then
      MIGRATE_OK=1
      break
    fi
    if [ $attempt -lt 3 ]; then
      warn "migrate 第 ${attempt} 次失败，等待 5s 重试..."
      sleep 5
    fi
  done
  "${DC[@]}" rm -f migrate-schema 2>/dev/null || true
  if [ "$MIGRATE_OK" = 1 ]; then
    log "migrate 成功"
  else
    die "migrate 失败（已重试 3 次）：${DC[*]} up --exit-code-from migrate-schema migrate-schema" 3
  fi

  # 7.4 seed（goose 单链路 migrations/seed/，每次部署都跑 —— goose 用
  #     goose_db_version_seed 版本表自动追踪已应用项，新加 seed 自动 catch up）
  log "执行 db seed（容器：migrate-seed-sql，goose 幂等）..."
  if "${DC[@]}" up --exit-code-from migrate-seed-sql migrate-seed-sql; then
    log "seed 成功"
  else
    die "seed 失败：${DC[*]} up --exit-code-from migrate-seed-sql migrate-seed-sql" 3
  fi
  "${DC[@]}" rm -f migrate-seed-sql 2>/dev/null || true

  # 7.5 时序库 schema 迁移（容器：migrate-tsdb-schema，#347）—— 独立 TimescaleDB
  #     实例 postgres-tsdb，goose 版本表 goose_db_version_tsdb，与主库 migrate 互不影响。
  #     与 7.3 同样用 `up --exit-code-from` + 3 次重试，显式捕获失败（优于仅靠 Step 8
  #     up -d 的隐式 depends_on——后者失败时诊断信息差、无重试）。goose 幂等，升级重跑安全。
  log "执行时序库 migrate（容器：migrate-tsdb-schema，goose 幂等）..."
  TSDB_MIGRATE_OK=0
  for attempt in 1 2 3; do
    if "${DC[@]}" up --exit-code-from migrate-tsdb-schema migrate-tsdb-schema; then
      TSDB_MIGRATE_OK=1
      break
    fi
    if [ $attempt -lt 3 ]; then
      warn "时序库 migrate 第 ${attempt} 次失败，等待 5s 重试..."
      sleep 5
    fi
  done
  "${DC[@]}" rm -f migrate-tsdb-schema 2>/dev/null || true
  if [ "$TSDB_MIGRATE_OK" = 1 ]; then
    log "时序库 migrate 成功"
  else
    die "时序库 migrate 失败（已重试 3 次）：${DC[*]} up --exit-code-from migrate-tsdb-schema migrate-tsdb-schema" 3
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

log "等待业务容器启动（最多 90s，健康检查每 5s 重试）..."
HEALTHCHECK_TIMEOUT=90
HEALTHCHECK_INTERVAL=5
HEALTHCHECK_WAIT=0
HEALTHCHECK_LOG="$(mktemp)"
HEALTH_OK=0
while [ "$HEALTHCHECK_WAIT" -lt "$HEALTHCHECK_TIMEOUT" ]; do
  if bash "$OMC_ROOT/current/deploy/healthcheck.sh" >"$HEALTHCHECK_LOG" 2>&1; then
    HEALTH_OK=1
    break
  fi
  sleep "$HEALTHCHECK_INTERVAL"
  HEALTHCHECK_WAIT=$((HEALTHCHECK_WAIT + HEALTHCHECK_INTERVAL))
done

# =============================================================================
# Step 9. healthcheck
# =============================================================================
sep "9/9 健康检查"

if [ "$HEALTH_OK" -eq 1 ]; then
  cat "$HEALTHCHECK_LOG"
else
  cat "$HEALTHCHECK_LOG"
  warn "健康检查在 ${HEALTHCHECK_TIMEOUT}s 内未通过"
fi
rm -f "$HEALTHCHECK_LOG"

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
log "  · 卸载(保留数据)：sudo bash $OMC_ROOT/current/deploy/uninstall.sh --force"
echo
log "访问地址："
log "  · Web 管理界面：  http://<服务器IP>:8081   （运维浏览器登录）"
log "  · 基站 ACS URL：  http://<服务器IP>:8080   （TR-069，基站设备侧填，人不浏览）"
log "  · MinIO Console：http://<服务器IP>:9001"
[ "$SKIP_MONITORING" = 0 ] && log "  · Grafana：       http://<服务器IP>:3030   （宿主 3030 → 容器 3000）"
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

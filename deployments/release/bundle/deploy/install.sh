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
#   sudo bash deploy/install.sh --fresh-install --yes --public-host 172.24.224.78
#                                                        # 清理旧数据后全新安装
#   sudo bash deploy/install.sh --fresh-install --floor-tolerance-pct 80 ...
#                                                        # 自定义资源下限缺口容忍度（0-99）
#   sudo bash deploy/install.sh --lang en --check-only # English installation output
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
#   --fresh-install   停止旧栈并删除 OMC 数据/配置/项目 volumes 后全新安装（危险）
#   --public-host <h> 全新安装时写入基站可达的 OMC_PUBLIC_HOST
#   --floor-tolerance-pct <N>
#                     全新安装资源规划的组件下限缺口容忍度（0-99，默认 60）
#   --lang <cn|en>    安装提示语言（默认 en，也可用 OMC_LANG=cn 切换中文）
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

export OMC_LANG="${OMC_LANG:-en}"

install_message() {
  if [ "$OMC_LANG" = en ]; then
    printf '%s' "${2:-$1}"
  else
    printf '%s' "$1"
  fi
}
log()  { echo -e "\033[1;32m[install]\033[0m $(install_message "$1" "${2:-}")"; }
warn() { echo -e "\033[1;33m[install][$( [ "$OMC_LANG" = en ] && echo Warning || echo 警告 )]\033[0m $(install_message "$1" "${2:-}")" >&2; }
die()  {
  local cn="$1" en="$1" code=1
  if [[ "${2:-}" =~ ^[0-9]+$ ]]; then
    code="$2"
  else
    en="${2:-$1}"
    code="${3:-1}"
  fi
  echo -e "\033[1;31m[install][$( [ "$OMC_LANG" = en ] && echo Error || echo 错误 )]\033[0m $(install_message "$cn" "$en")" >&2
  exit "$code"
}
sep()  { echo -e "\033[1;34m──────────────── $(install_message "$1" "${2:-}") ────────────────\033[0m"; }

show_help() {
  if [ "$OMC_LANG" = en ]; then
    cat <<'EOF'
OMC installation / upgrade script

Usage:
  sudo bash deploy/install.sh [options]

Options:
  --skip-infra                 Skip loading infrastructure images
  --skip-migrate               Skip migrate and seed containers
  --skip-web                   Do not start the web container
  --skip-monitoring            Do not start the monitoring stack
  --check-only                 Run prechecks only; do not modify the system
  --fresh-install              Remove OMC data and Docker volumes before install
  --public-host <host>         Set the host reachable by base stations
  --floor-tolerance-pct <N>    Resource floor-gap tolerance, 0-99 (default 60)
  --lang <cn|en>               Installation output language (default en)
  --yes                        Accept interactive confirmations
  -h, --help                   Show this help
EOF
  else
    sed -n '3,55p' "$0"
  fi
}

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
if [ -f "$DEPLOY_DIR/compose-env-lib.sh" ]; then
  . "$DEPLOY_DIR/compose-env-lib.sh"
else
  die "缺 $DEPLOY_DIR/compose-env-lib.sh（Compose 环境优先级加载库）" 1
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
if [ -f "$DEPLOY_DIR/gpv-handoff-lib.sh" ]; then
  . "$DEPLOY_DIR/gpv-handoff-lib.sh"
else
  die "缺 $DEPLOY_DIR/gpv-handoff-lib.sh（GPV 无损升级接力库，由 build-release.sh 随包发布）" 1
fi
if [ -f "$DEPLOY_DIR/redis-cutover-lib.sh" ]; then
  . "$DEPLOY_DIR/redis-cutover-lib.sh"
else
  die "缺 $DEPLOY_DIR/redis-cutover-lib.sh（旧单 Redis 安全切换库）" 1
fi

# 升级时 deploy/.env 里【运维自定义】的键 —— 跨版本继承,不被新包默认值覆盖。
# 注：6 个密钥键虽仍在此列（升级时把上一版有效凭证带进新 .env，供 ensure_secrets 首迁导入），
# 但密钥的【唯一权威源】是 etc/secrets.env —— ensure_secrets 在 source/起 infra 前用它覆盖 .env，
# 故 .env.saved 即便被某次默认口令安装写脏，也不再污染密钥（secrets.env 一经生成永不重生成）。
# 【版本相关】键(PROJECT_VERSION / IMAGE_*)不在此列,始终用新包值。
# 注：POSTGRES_TSDB_USER/PASSWORD/DB（时序库凭据，#347）跨版本继承；TSDB_HOST 是 compose 服务名
# （随包固定值），故【不】列入继承白名单，始终用新包值。
ENV_PRESERVE_KEYS="POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_TSDB_USER POSTGRES_TSDB_PASSWORD POSTGRES_TSDB_DB MINIO_ROOT_USER MINIO_ROOT_PASSWORD GRAFANA_ADMIN_USER GRAFANA_ADMIN_PASSWORD OMCGO_JWT_SECRET OMC_SHARED_SECRET OMC_PUBLIC_HOST POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH REDIS_PM_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH PM_AGGREGATION_FINALIZE_CONCURRENCY GPV_PROVISION_QUEUE GPV_PROVISION_CONCURRENCY GPV_PROVISION_QUEUE_DEPTH GPV_RPC_DURABLE GPV_RPC_SOURCE_CONSUMER GPV_RPC_START_SEQUENCE GPV_RPC_CONCURRENCY GPV_RPC_QUEUE_DEPTH GPV_ACK_WAIT GPV_MAX_DELIVER GPV_MAX_ACK_PENDING"

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
          p=index($0,"="); k=substr($0,1,p-1); candidate=substr($0,p+1)
          # 空的旧值不能覆盖新包有效值（尤其 OMC_PUBLIC_HOST 和数据路径）。
          if ((k in want) && candidate != "") { val[k]=candidate; have[k]=1 }
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
    log ".env:已从上一版继承运维自定义值(口令 / JWT / OMC_PUBLIC_HOST);镜像 tag 用新包。如需改值,编辑 $new 后重启业务容器" ".env: inherited operator values (credentials / JWT / OMC_PUBLIC_HOST) from the previous release; image tags use the new package. Edit $new and restart services to change them"
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
FRESH_INSTALL=0
FLOOR_TOLERANCE_PCT=60
PUBLIC_HOST_OVERRIDE="${OMC_PUBLIC_HOST:-}"
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
    --fresh-install)   FRESH_INSTALL=1; shift ;;
    --public-host)     PUBLIC_HOST_OVERRIDE="${2:?--public-host 需要 IP 或域名}"; shift 2 ;;
    --floor-tolerance-pct) FLOOR_TOLERANCE_PCT="${2:?--floor-tolerance-pct 需要 0-99 的整数}"; shift 2 ;;
    --floor-tolerance-pct=*) FLOOR_TOLERANCE_PCT="${1#*=}"; shift ;;
    --lang|--language) OMC_LANG="${2:?--lang 需要 cn 或 en}"; shift 2 ;;
    --lang=*|--language=*) OMC_LANG="${1#*=}"; shift ;;
    --yes)             ASSUME_YES=1; shift ;;
    -h|--help)         show_help; exit 0 ;;
    --uninstall)       die "卸载请用 uninstall.sh：sudo bash $DEPLOY_DIR/uninstall.sh -h" ;;
    *)                 die "未知参数：$1（-h 查看用法）" ;;
  esac
done

case "$OMC_LANG" in
  cn|en) ;;
  *) die "--lang 仅支持 cn 或 en，收到：$OMC_LANG" "--lang accepts only cn or en, got: $OMC_LANG" 1 ;;
esac

case "$FLOOR_TOLERANCE_PCT" in
  ''|*[!0-9]*) die "--floor-tolerance-pct 仅支持 0-99 的整数，收到：$FLOOR_TOLERANCE_PCT" 1 ;;
esac
[ "$FLOOR_TOLERANCE_PCT" -lt 100 ] ||
  die "--floor-tolerance-pct 必须小于 100，收到：$FLOOR_TOLERANCE_PCT" 1

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

confirm() {
  [ "$ASSUME_YES" = 1 ] && return 0
  local yn prompt
  prompt="$(install_message "$1" "${2:-}")"
  read -rp "$prompt [Y/n] " yn
  case "${yn:-Y}" in [Yy]*|"") return 0 ;; *) return 1 ;; esac
}

set_env_value() { # set_env_value <file> <key> <value>
  local file="$1" key="$2" value="$3" tmp
  tmp="$(mktemp)" || return 1
  if awk -v key="$key" -v value="$value" '
      BEGIN { replaced=0 }
      $0 ~ "^" key "=" {
        if (!replaced) print key "=" value
        replaced=1
        next
      }
      { print }
      END { if (!replaced) print key "=" value }
    ' "$file" > "$tmp"; then
    cat "$tmp" > "$file"
    rm -f "$tmp"
    return 0
  fi
  rm -f "$tmp"
  return 1
}

fresh_install_reset() {
  local package_env="$PKG_ROOT/deploy/.env"
  local old_deploy="$OMC_ROOT/current/deploy"
  local old_env_args=()
  local old_compose_files=()
  local data_key path volumes volume

  [ "$CHECK_ONLY" = 0 ] || die "--fresh-install 不能与 --check-only 同时使用" 1
  [ -f "$package_env" ] || die "全新安装缺少 $package_env" 1
  [ -f "$PKG_ROOT/deploy/plan-resources.sh" ] || die "全新安装缺少 plan-resources.sh" 1
  command -v docker >/dev/null 2>&1 || die "缺少 docker，无法执行全新安装清理" 1
  docker compose version >/dev/null 2>&1 || die "缺少 docker compose v2，无法执行全新安装清理" 1

  if [ -f "$old_deploy/docker-compose.infra.yml" ]; then
    [ -f "$old_deploy/.env" ] && old_env_args+=( --env-file "$old_deploy/.env" )
    [ -f "$old_deploy/resources.env" ] && old_env_args+=( --env-file "$old_deploy/resources.env" )
    for file in docker-compose.infra.yml docker-compose.app.yml docker-compose.web.yml docker-compose.monitoring.yml; do
      [ -f "$old_deploy/$file" ] && old_compose_files+=( -f "$old_deploy/$file" )
    done
    log "全新安装：停止旧 OMC 栈 ..." "Fresh install: stopping the existing OMC stack ..."
    ( cd "$old_deploy" && docker compose -p "$COMPOSE_PROJECT" "${old_env_args[@]}" \
        "${old_compose_files[@]}" down --remove-orphans ) || warn "旧 OMC 栈停止返回非零，继续执行数据清理"
  fi

  [ -n "$PUBLIC_HOST_OVERRIDE" ] ||
    PUBLIC_HOST_OVERRIDE="$(deploy_env_file_value OMC_PUBLIC_HOST "$package_env" 2>/dev/null || true)"
  deploy_env_public_host_valid "$PUBLIC_HOST_OVERRIDE" ||
    die "全新安装必须提供有效的 --public-host（例如 172.24.224.78）" 1
  set_env_value "$package_env" OMC_PUBLIC_HOST "$PUBLIC_HOST_OVERRIDE" ||
    die "无法写入 $package_env 的 OMC_PUBLIC_HOST" 1

  log "全新安装：按目标主机重新规划资源（组件下限缺口容忍度 ${FLOOR_TOLERANCE_PCT}%，最低运行预算仍为硬门禁）..." "Fresh install: planning resources for this host (floor-gap tolerance ${FLOOR_TOLERANCE_PCT}%; minimum runtime budget remains enforced) ..."
  fresh_plan_args=( --floor-tolerance-pct "$FLOOR_TOLERANCE_PCT" --lang "$OMC_LANG" )
  [ "$SKIP_MONITORING" = 1 ] && fresh_plan_args+=( --skip-monitoring )
  ( cd "$PKG_ROOT" && OMC_STORAGE_ENV_FILE="$package_env" \
      bash "$PKG_ROOT/deploy/plan-resources.sh" "${fresh_plan_args[@]}" ) ||
    die "全新安装资源规划失败；请查看上方资源规划提示。最低运行预算不能通过 --floor-tolerance-pct 绕过；低内存主机可使用 --skip-monitoring 重试。" "Fresh-install resource planning failed; see the message above. The minimum runtime budget cannot be bypassed with --floor-tolerance-pct; retry with --skip-monitoring on a low-memory host." 1

  log "全新安装将清理以下数据目录：" "Fresh install will remove the following data directories:"
  for data_key in POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH REDIS_PM_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH; do
    path="$(storage_env_get "$package_env" "$data_key")"
    case "$path" in
      /*) ;;
      *) die "全新安装数据路径无效：$data_key=$path" 1 ;;
    esac
    case "$path" in
      /|/home|/opt|/opt/omc|/var|/tmp)
        die "全新安装拒绝清理危险数据路径：$data_key=$path" 1 ;;
    esac
    log "  $data_key=$path" "  $data_key=$path"
  done

  confirm "全新安装将永久删除 OMC 数据、配置、日志和项目 Docker volumes，继续吗？" "Fresh install will permanently delete OMC data, configuration, logs, and project Docker volumes. Continue?" ||
    die "用户取消全新安装，未删除任何 OMC 数据" "Fresh install cancelled; no OMC data was deleted." 1

  log "全新安装：删除 bind-mount 数据目录 ..." "Fresh install: removing bind-mount data directories ..."
  for data_key in POSTGRES_DATA_PATH TSDB_DATA_PATH REDIS_DATA_PATH REDIS_PM_DATA_PATH NATS_DATA_PATH MINIO_DATA_PATH; do
    path="$(storage_env_get "$package_env" "$data_key")"
    rm -rf -- "$path"
  done
  rm -rf -- "$OMC_ROOT/data" "$OMC_ROOT/etc" "$OMC_ROOT/current" "$OMC_ROOT/run/logs"

  volumes="$(docker volume ls -q --filter label=com.docker.compose.project="$COMPOSE_PROJECT")"
  for volume in $volumes; do
    docker volume rm "$volume" >/dev/null ||
      die "无法删除 Docker volume：$volume；请确认旧 OMC 容器已停止" 1
  done
  log "全新安装：旧 OMC 数据已删除" "Fresh install: old OMC data has been deleted"
}

if [ "$FRESH_INSTALL" = 1 ]; then
  fresh_install_reset
fi

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

# precheck_skipped_infra_images —— --skip-infra 表示目标机已具备本次版本要求的
# 基础设施/监控镜像。必须在复制 release、改写 data 和切换 current 前验证该前提；
# 否则镜像仓库名或版本升级时会在 Step 4 才失败，留下“软链已切但业务仍跑旧镜像”的
# 半升级状态。业务镜像由项目包在 Step 4 负责加载，不属于这里的前置条件。
precheck_skipped_infra_images() {
  local image_keys=(
    IMAGE_POSTGRES IMAGE_POSTGRES_TSDB IMAGE_REDIS IMAGE_NATS IMAGE_MINIO
  )
  if [ "$SKIP_MONITORING" = 0 ]; then
    image_keys+=(
      IMAGE_PROMETHEUS IMAGE_ALERTMANAGER IMAGE_GRAFANA IMAGE_LOKI IMAGE_TEMPO
      IMAGE_OTELCOL IMAGE_NATS_EXPORTER IMAGE_NGINX_EXPORTER IMAGE_NODE_EXPORTER
      IMAGE_CADVISOR
    )
  fi

  local key image
  local missing_images=()
  for key in "${image_keys[@]}"; do
    image="$(deploy_env_file_value "$key" "$PKG_ROOT/deploy/.env" 2>/dev/null || true)"
    [ -z "$image" ] && continue
    docker image inspect "$image" >/dev/null 2>&1 || missing_images+=("$image")
  done

  if [ "${#missing_images[@]}" -gt 0 ]; then
    die "--skip-infra 前置条件不满足，缺少本地基础设施/监控镜像：${missing_images[*]}。未复制 release、未切换 current、未改写业务数据；请先解压匹配架构的基础设施包到 ${INFRA_DIR}，并去掉 --skip-infra 重试。" 1
  fi
  log "--skip-infra 镜像预检通过：本次基础设施/监控镜像均已在本机" "--skip-infra image precheck passed: all infrastructure and monitoring images are available locally"
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
  confirm "确认对主库卷 $vol 执行 timescaledb 剥离自愈？" "Confirm TimescaleDB detach recovery for volume $vol?" || \
    die "用户取消自愈。如需保数据，可临时把 deploy/.env 的 IMAGE_POSTGRES 改回 timescaledb 镜像后重装。" "Recovery cancelled. To preserve data, temporarily set IMAGE_POSTGRES back to the TimescaleDB image in deploy/.env and reinstall." 1

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
sep "1/9 precheck" "1/9 Precheck"

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
log "docker compose 命令：$COMPOSE" "Docker Compose command: $COMPOSE"

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

# OMC_PUBLIC_HOST 是基站回传 PM/MR 文件所需的运维地址，不能等到复制包、
# 切换 current 或覆盖 etc 后才校验。新包显式配置优先；新包留空时继承现行
# release 或 uninstall 保存的 .env。这样地址缺失只会阻断 precheck，不会留下半升级状态。
PUBLIC_HOST_CANDIDATE=""
if [ -f "$PKG_ROOT/deploy/.env" ]; then
  PUBLIC_HOST_CANDIDATE="$(deploy_env_file_value OMC_PUBLIC_HOST "$PKG_ROOT/deploy/.env" 2>/dev/null || true)"
fi
if [ -z "$PUBLIC_HOST_CANDIDATE" ]; then
  for previous_env in "$OMC_ROOT/current/deploy/.env" "$OMC_ROOT/etc/.env.saved"; do
    if [ -f "$previous_env" ]; then
      PUBLIC_HOST_CANDIDATE="$(deploy_env_file_value OMC_PUBLIC_HOST "$previous_env" 2>/dev/null || true)"
      [ -n "$PUBLIC_HOST_CANDIDATE" ] && break
    fi
  done
fi
deploy_env_public_host_valid "$PUBLIC_HOST_CANDIDATE" ||
  die "OMC_PUBLIC_HOST 未配置为基站可达主机（当前: ${PUBLIC_HOST_CANDIDATE:-<空>}）；请先在 $PKG_ROOT/deploy/.env 配置服务器对基站可达的 IP/域名后重试" 1
log "OMC_PUBLIC_HOST 预检通过：$PUBLIC_HOST_CANDIDATE" "OMC_PUBLIC_HOST precheck passed: $PUBLIC_HOST_CANDIDATE"

# 资源契约是部署必需输入。必须在 --check-only 退出、current 切换和任何容器重启之前
# 校验真正会被本次安装采用的候选，不能等到 Step 6 组装 Compose 才发现旧三行文件。
RESOURCE_ENV_CANDIDATE="$(resolve_resource_env_candidate)" ||
  die "未找到 resources.env；请先运行 bash $PKG_ROOT/deploy/plan-resources.sh，禁止静默回退 Compose 默认限额" 1
resource_env_validate "$RESOURCE_ENV_CANDIDATE" ||
  die "resources.env 不是完整资源规划：${RESOURCE_ENV_CANDIDATE}；请重新运行 plan-resources.sh" 1
log "资源规划预检通过：$RESOURCE_ENV_CANDIDATE" "Resource plan precheck passed: $RESOURCE_ENV_CANDIDATE"

# Worker 新版本会在启动时按真实并发核验 TSDB 连接池。这个只读门禁必须在
# Step 2 旧 systemd 停服以及任何数据/配置改写之前执行，避免配置不足时把
# 原服务停掉后才发现新 Worker 无法启动。历史默认 40/96 会在 Step 3 自动迁移；
# 其他低于 128 的运维自定义值要求先显式调整。
if ! validate_worker_tsdb_pool_precheck \
  "$OMC_ROOT/etc/worker.prod.yaml" \
  "$PKG_ROOT/etc/worker.prod.yaml"; then
  live_worker_tsdb_pool="$(worker_tsdb_max_conns "$OMC_ROOT/etc/worker.prod.yaml" 2>/dev/null || echo 未安装/无法读取)"
  package_worker_tsdb_pool="$(worker_tsdb_max_conns "$PKG_ROOT/etc/worker.prod.yaml" 2>/dev/null || echo 无法读取)"
  die "Worker TSDB 连接池部署前门禁失败：现网=${live_worker_tsdb_pool}，新包=${package_worker_tsdb_pool}，安全预算=${WORKER_TSDB_SAFE_POOL}；未停止旧服务、未迁移数据、未改配置" 1
fi
log "Worker TSDB 连接池预检通过" "Worker TSDB connection pool precheck passed"

# 兜底：旧版 build-release.sh 在 umask=027 机器上构建时 monitoring/ 配置会落 0640，
# prometheus/loki/tempo/alertmanager 等非 root 容器读不动直接 fail。
# 新版 build-release.sh 已 baked chmod a+rX 到 tar；这里再 defensive 兜一遍。
if [ "$SKIP_MONITORING" = 0 ] && [ -d "$PKG_ROOT/deploy/monitoring" ]; then
  chmod -R a+rX "$PKG_ROOT/deploy/monitoring"
fi

# 项目版本号
VERSION="$(awk -F= '/^project_version=/{print $2}' "$PKG_ROOT/VERSION" 2>/dev/null || echo unknown)"
log "项目版本：$VERSION" "Project version: $VERSION"

# infra 目录（除非 --skip-infra）
if [ "$SKIP_INFRA" = 0 ]; then
  [ -d "$INFRA_DIR" ] || die "基础设施目录不存在：$INFRA_DIR
  · 首次部署需先：cd $INFRA_DIR && tar -xJf omc-infra-<版本>-<架构>.tar.xz --strip-components=1
  · 或加 --skip-infra 跳过基础设施镜像 load" 1
  [ -d "$INFRA_DIR/images" ] || die "基础设施目录缺 images/：$INFRA_DIR/images" 1
else
  precheck_skipped_infra_images
fi

log "precheck 通过" "Precheck passed"

if [ "$CHECK_ONLY" = 1 ]; then
  log "--check-only：不做任何修改，退出"
  exit 0
fi

# =============================================================================
# Step 2. 旧 systemd 单元自动迁移（兼容旧版宿主机二进制部署）
#   旧版本曾用 systemd 跑 omcgo-{app,acs,worker} 二进制；本版本改为容器，
#   必须先停掉并禁用旧 systemd 单元，否则会与容器抢 9091/7547 等端口。
# =============================================================================
sep "2/9 旧 systemd 单元自动迁移" "2/9 Migrate legacy systemd units"

HANDOFF_PREPARED=0
GPV_SYSTEMD_HANDOFF_PREPARED=0
SYSTEMD_HANDOFF_IMAGE="$(gpv_handoff_env_get "$PKG_ROOT/deploy/.env" IMAGE_APP 2>/dev/null || true)"
if ! gpv_handoff_migrate_legacy_systemd \
  "$SYSTEMD_HANDOFF_IMAGE" \
  "$PKG_ROOT/images" \
  "$OMC_ROOT/etc/app.prod.yaml"; then
  die "旧 systemd app 的 GPV handoff 失败；未停止或禁用任何旧业务服务" 2
fi
if [ "$GPV_SYSTEMD_HANDOFF_PREPARED" = 1 ]; then
  HANDOFF_PREPARED=1
  log "旧 systemd app 的 GPV durable 已预创建，并已按 ACS → worker → app 顺序停服"
  warn "宿主机旧 OMC 二进制（如 /opt/omc/current/bin/）保留未删，请运维确认无残留进程后自行清理"
fi

# =============================================================================
# Step 3. 建立 OMC 目录布局 + current 软链
# =============================================================================
sep "3/9 建立目录结构 + current 软链" "3/9 Prepare directories and current symlink"

mkdir -p "$OMC_ROOT/releases" "$OMC_ROOT/etc" "$OMC_ROOT/packages" \
         "$OMC_ROOT/run/logs/app"   "$OMC_ROOT/run/logs/acs" \
         "$OMC_ROOT/run/logs/acs-candidate" \
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
  confirm "继续吗？" "Continue?" || die "用户取消" "Operation cancelled" 1
  mv "$RELEASE_DIR" "$RELEASE_DIR.bak.$(date +%Y%m%d%H%M%S)"
fi

# 如果不是直接在版本目录里跑，复制到版本目录
if [ "$(readlink -f "$PKG_ROOT")" != "$(readlink -f "$RELEASE_DIR")" ]; then
  log "复制项目包到 $RELEASE_DIR ..." "Copying release package to $RELEASE_DIR ..."
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
  log "resources.env:已从上一版继承资源限额(plan-resources.sh 调优值不丢)" "resources.env: inherited resource limits from the previous release"
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
  log "首次部署：播种 data 基线 → $OMC_ROOT/data" "Fresh install: seeding data baseline -> $OMC_ROOT/data"
  mkdir -p "$OMC_ROOT/data"
  cp -a "$NEW_DATA/." "$OMC_ROOT/data/" || die "播种 data 失败;字典将为空,中止部署" 4
else
  DATA_SNAP="$OMC_ROOT/data.bak.$(date +%Y%m%d%H%M%S)"
  log "升级：快照现网 data → $DATA_SNAP(回滚用)" "Upgrade: snapshot live data -> $DATA_SNAP (rollback copy)"
  cp -a "$OMC_ROOT/data" "$DATA_SNAP" || warn "快照 data 失败(磁盘满?);继续合并但无回滚点"
  log "升级：反向合并(现网赢、新版补充)新版 builtin → $OMC_ROOT/data" "Upgrade: merge new built-in data into $OMC_ROOT/data (live data wins)"
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
  log "首次部署：复制配置模板到 $OMC_ROOT/etc/（首次必修改默认口令！）" "Fresh install: copying configuration templates to $OMC_ROOT/etc/ (change default credentials before production use)"
  cp -rn "$RELEASE_DIR/etc/." "$OMC_ROOT/etc/"
else
  do_overwrite=0
  if [ "$OVERWRITE_ETC" = 1 ]; then
    do_overwrite=1
  elif [ "$ASSUME_YES" = 0 ]; then
    warn "$OMC_ROOT/etc/ 已有实例配置（包含可能已改好的强口令 / JWT 密钥 / TLS 证书路径等）" "$OMC_ROOT/etc/ contains instance configuration (possibly including custom credentials, JWT keys, and TLS certificate paths)"
    warn "  选 y 将覆盖为新包模板（原 etc 自动备份到 etc.bak.<时间戳>）" "  Enter y to replace it with the package template (the old etc is backed up automatically)"
    warn "  选 N 保留现有配置不动（默认）" "  Enter N to keep the current configuration (default)"
    read -rp "$(install_message "是否用新包模板覆盖 $OMC_ROOT/etc/？" "Replace $OMC_ROOT/etc/ with the package template?") [y/N] " yn
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
    for service_config in acs.prod.yaml app.prod.yaml worker.prod.yaml; do
      if upgrade_prod_database_dsns \
        "$OMC_ROOT/etc/$service_config" \
        "$RELEASE_DIR/etc/$service_config"; then
        log "升级 $service_config：已同步生产数据库 DSN 模板，使用当前 .env 凭证" "Upgrade $service_config: synchronized production database DSN template using current .env credentials"
      else
        die "$service_config 数据库 DSN 自动同步失败；未切换 current" 1
      fi
    done
    upgrade_app_param_sync_recovery_limit \
      "$OMC_ROOT/etc/app.prod.yaml" \
      "$RELEASE_DIR/etc/app.prod.yaml" ||
      die "app.prod.yaml 的 param_sync.recovery_run_limit 自动迁移失败；未切换 current" 1
    case "${PARAM_SYNC_RECOVERY_LIMIT_UPGRADE_RESULT:-noop}" in
      migrated)
        log "升级参数同步恢复容量：recovery_run_limit 20 → 200（其他实例配置保持不变）"
        ;;
      preserved)
        log "参数同步 recovery_run_limit 为运维自定义值，升级时保持不变"
        ;;
    esac
    upgrade_app_gpv_response_config \
      "$OMC_ROOT/etc/app.prod.yaml" \
      "$RELEASE_DIR/etc/app.prod.yaml" ||
      die "app.prod.yaml 缺少 provision.gpv_response 且自动补齐失败；未切换 current" 1
    for service_config in app.prod.yaml worker.prod.yaml; do
      upgrade_pm_redis_config \
        "$OMC_ROOT/etc/$service_config" \
        "$RELEASE_DIR/etc/$service_config" ||
        die "$service_config 缺少 pm_redis 且自动补齐失败；未切换 current" 1
    done
    upgrade_worker_tsdb_pool \
      "$OMC_ROOT/etc/worker.prod.yaml" \
      "$RELEASE_DIR/etc/worker.prod.yaml" ||
      die "worker.prod.yaml 的 tsdb.max_conns 无法读取或安全迁移；未切换 current" 1
    case "${WORKER_TSDB_POOL_UPGRADE_RESULT:-invalid}" in
      migrated)
        log "升级 Worker TSDB 连接池：历史默认值 → $(worker_tsdb_max_conns "$RELEASE_DIR/etc/worker.prod.yaml")（其他实例配置保持不变）"
        ;;
      preserved)
        log "Worker tsdb.max_conns 为满足新预算的运维自定义值，升级时保持不变"
        ;;
      insufficient)
        die "worker.prod.yaml 的 tsdb.max_conns=$(worker_tsdb_max_conns "$OMC_ROOT/etc/worker.prod.yaml") 低于新版本安全预算 $(worker_tsdb_max_conns "$RELEASE_DIR/etc/worker.prod.yaml")；请提升该值后重试，未切换 current" 1
        ;;
      noop)
        ;;
      *)
        die "worker.prod.yaml 的 tsdb.max_conns 升级结果异常；未切换 current" 1
        ;;
    esac
    log "$OMC_ROOT/etc/ 已有实例配置，保留不覆盖（如需覆盖加 --overwrite-etc）" "$OMC_ROOT/etc/ contains instance configuration; keeping it unchanged (use --overwrite-etc to replace it)"
  fi
fi

# 切 current 软链（原子）
ln -sfn "$RELEASE_DIR" "$OMC_ROOT/current"
log "current → $RELEASE_DIR" "current -> $RELEASE_DIR"

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
sep "4/9 load 镜像" "4/9 Load images"

# 提前加载 .env 获取镜像名（IMAGE_* 变量），供 images_exist 判定使用。
# 必须随后按 --env-file 的相同顺序加载 resources.env：调用进程环境优先级高于
# --env-file，若只 source .env，会导致 resources.env 的同名调优值永远不生效。
ENV_FILE="$OMC_ROOT/current/deploy/.env"
RESOURCE_ENV_FILE="$OMC_ROOT/current/deploy/resources.env"
deploy_env_load "$ENV_FILE" "$RESOURCE_ENV_FILE"
if ! deploy_env_public_host_valid "${OMC_PUBLIC_HOST:-}"; then
  die "OMC_PUBLIC_HOST 未配置为基站可达主机（当前: ${OMC_PUBLIC_HOST:-<空>}）；请在 deploy/.env 中配置后重试" 1
fi
# 必须在 source .env 之后应用：旧 .env 可能显式写了 tracer=true。
# 同时把安装 profile 持久化，供后续独立运行的 svc/healthcheck 使用。
monitoring_profile_apply_install "$ENV_FILE" "$SKIP_MONITORING" ||
  die "无法持久化 monitoring profile 到 $ENV_FILE" 1
storage_prepare_configured_env_paths "$ENV_FILE" ||
  die "有状态服务数据路径校验/创建失败；请检查 $ENV_FILE 中五个 *_DATA_PATH" 1

if [ "$SKIP_INFRA" = 0 ]; then
  INFRA_IMAGES=("$IMAGE_POSTGRES" "${IMAGE_POSTGRES_TSDB:-}" "$IMAGE_REDIS" "$IMAGE_NATS" "$IMAGE_MINIO" "${IMAGE_NGINX:-}")
  MON_IMAGES=("${IMAGE_PROMETHEUS:-}" "${IMAGE_ALERTMANAGER:-}" "${IMAGE_GRAFANA:-}" "${IMAGE_LOKI:-}" "${IMAGE_TEMPO:-}" "${IMAGE_OTELCOL:-}" "${IMAGE_NATS_EXPORTER:-}" "${IMAGE_NGINX_EXPORTER:-}" "${IMAGE_NODE_EXPORTER:-}" "${IMAGE_CADVISOR:-}")

  if images_exist "${INFRA_IMAGES[@]}" "${MON_IMAGES[@]}"; then
    log "基础设施 + 监控镜像已存在，跳过 load；handoff 完成前保持现有容器不动"
  else
    log "load 基础设施 + 监控镜像（$INFRA_DIR/images/）"
    for tar in "$INFRA_DIR/images"/*.tar; do
      [ -f "$tar" ] || continue
      log "  · docker load < $(basename "$tar")"
      docker load -i "$tar"
    done
  fi
else
  log "--skip-infra：跳过基础设施 / 监控镜像 load" "--skip-infra: skipping infrastructure / monitoring image loading"
fi

BIZ_IMAGES=("$IMAGE_APP" "$IMAGE_ACS" "$IMAGE_WORKER" "$IMAGE_WEB")

if images_exist "${BIZ_IMAGES[@]}"; then
  log "业务镜像已存在，跳过 load；handoff 完成前保持现有 app 不动" "Business images already exist; skipping load and keeping the current App unchanged until handoff completes"
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

# 项目包与基础设施包分离交付，但运行时必须完全离线：所有本次 Compose
# 会使用的镜像都必须已经存在于本机。禁止让 Compose 在启动阶段隐式 pull，
# 否则内网安装会变成部分成功、部分联网拉取的不可复现状态。
REQUIRED_IMAGES=("$IMAGE_POSTGRES" "${IMAGE_POSTGRES_TSDB:-}" "$IMAGE_REDIS" "$IMAGE_NATS" "$IMAGE_MINIO" "$IMAGE_APP" "$IMAGE_ACS" "$IMAGE_WORKER")
[ "$SKIP_WEB" = 1 ] || REQUIRED_IMAGES+=("$IMAGE_WEB")
if [ "$SKIP_MONITORING" = 0 ]; then
  REQUIRED_IMAGES+=("${IMAGE_PROMETHEUS:-}" "${IMAGE_ALERTMANAGER:-}" "${IMAGE_GRAFANA:-}" "${IMAGE_LOKI:-}" "${IMAGE_TEMPO:-}" "${IMAGE_OTELCOL:-}" "${IMAGE_NATS_EXPORTER:-}" "${IMAGE_NGINX_EXPORTER:-}" "${IMAGE_NODE_EXPORTER:-}" "${IMAGE_CADVISOR:-}")
fi
missing_images=()
for image in "${REQUIRED_IMAGES[@]}"; do
  [ -z "$image" ] && continue
  docker image inspect "$image" >/dev/null 2>&1 || missing_images+=("$image")
done
if [ "${#missing_images[@]}" -gt 0 ]; then
  die "离线安装缺少本地镜像：${missing_images[*]}。请先解压匹配架构的基础设施包到 $INFRA_DIR，并重新执行安装（不要使用 --skip-infra）；项目包业务镜像由本步骤负责 load。" 1
fi
log "离线镜像校验通过：本次 Compose 所需镜像均已在本机" "Offline image check passed: all required images are available locally"

# =============================================================================
# Step 5. 默认口令检查（PostgreSQL / MinIO / Grafana）
# =============================================================================
sep "5/9 默认口令安全检查" "5/9 Check default credentials"

# ENV_FILE 已在 Step 4 定义并 source，这里仅做存在性兜底
[ -f "$ENV_FILE" ] || die "缺 $ENV_FILE（由 build-release.sh 生成）" 1

if grep -qE '^(POSTGRES_PASSWORD=omcgo123|MINIO_ROOT_PASSWORD=minioadmin|GRAFANA_ADMIN_PASSWORD=admin|OMC_SHARED_SECRET=dps)$' "$ENV_FILE"; then
  warn "检测到 deploy/.env 含默认口令（POSTGRES_PASSWORD=omcgo123 / MINIO_ROOT_PASSWORD=minioadmin / GRAFANA_ADMIN_PASSWORD=admin / OMC_SHARED_SECRET=dps）"
  warn "  → 生产环境务必改强口令：vi $ENV_FILE"
  warn "  → etc/*.prod.yaml 用 \${VAR} 引用 .env，无需手改；生产凭证校验(GuardProductionSecrets)会拒绝默认值启动"
  confirm "已知风险，继续部署？" "Known risk detected. Continue deployment?" || die "用户取消，请先改口令" "Operation cancelled. Change the credentials first." 1
fi

# =============================================================================
# Step 6. 组装 docker compose 命令（全栈一次 up）
# =============================================================================
sep "6/9 组装 docker compose 命令" "6/9 Assemble Docker Compose command"

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
log "已检出完整 resources.env → 按其资源限额部署（plan-resources.sh 生成）" "Complete resources.env found -> deploying with its limits"

COMPOSE_FILES=( -f docker-compose.infra.yml -f docker-compose.app.yml )
[ "$SKIP_WEB" = 0 ]        && COMPOSE_FILES+=( -f docker-compose.web.yml )
[ "$SKIP_MONITORING" = 0 ] && COMPOSE_FILES+=( -f docker-compose.monitoring.yml )

DC=( $COMPOSE -p "$COMPOSE_PROJECT" "${ENV_FILES[@]}" "${COMPOSE_FILES[@]}" )
log "compose 命令：${DC[*]}" "Compose command: ${DC[*]}"

APP_CID="$("${DC[@]}" ps -q app 2>/dev/null || true)"
NATS_CID="$("${DC[@]}" ps -q nats 2>/dev/null || true)"
APP_RUNNING=0
NATS_RUNNING=0
[ -n "$APP_CID" ] && [ "$(docker inspect -f '{{.State.Running}}' "$APP_CID" 2>/dev/null || true)" = "true" ] && APP_RUNNING=1
[ -n "$NATS_CID" ] && [ "$(docker inspect -f '{{.State.Running}}' "$NATS_CID" 2>/dev/null || true)" = "true" ] && NATS_RUNNING=1
if [ "$APP_RUNNING" = 1 ] && [ "$NATS_RUNNING" != 1 ]; then
  die "旧 app 仍在运行但 NATS 不可用，无法读取 GPV consumer AckFloor；未停止旧 app" 2
fi
if [ "$HANDOFF_PREPARED" != 1 ] && [ "$NATS_RUNNING" = 1 ]; then
  log "在停止/重建旧 app 前预创建 GPV RPC 固定 durable ..." "Pre-creating the fixed GPV RPC durable before stopping/recreating the old App ..."
  gpv_handoff_prepare ||
    die "GPV consumer handoff 失败；未停止旧 app，修复 NATS/consumer 配置后重试" 2
  HANDOFF_PREPARED=1
fi

# =============================================================================
# Step 7. 启动基础设施 + 等就绪 → 跑 migrate / seed（一次性容器）
# =============================================================================
sep "7/9 启动基础设施 + 执行 migrate / seed" "7/9 Start infrastructure and run migrations / seed"

# 7.0 升级自愈（#347）：存量主库卷若被 timescaledb 初始化过，先剥离再起纯 PG，
#     否则 postgres:16-alpine 读到卷内 shared_preload_libraries='timescaledb' 会 FATAL。
#     幂等：全新装 / 已剥离 / 目标镜像仍 timescaledb 时自动跳过。
heal_main_pg_timescaledb_downgrade

if ! prepare_legacy_redis_cutover; then
  die "旧单 Redis 切换准备失败；已拒绝并行挂载或无校验升级" 2
fi

# 7.1 起基础设施（postgres / postgres-tsdb / redis-core / redis-pm / nats / minio）
log "启动基础设施容器 ..." "Starting infrastructure containers ..."
"${DC[@]}" up --pull never -d postgres postgres-tsdb redis-core redis-pm nats minio

log "等待基础设施 ready（最多 90s）..." "Waiting for infrastructure readiness (up to 90s) ..."
WAIT=0
PG_OK=0; TS_OK=0; RD_CORE_OK=0; RD_PM_OK=0; NATS_OK=0
while [ $WAIT -lt 90 ]; do
  sleep 3; WAIT=$((WAIT+3))
  PG_CID="$("${DC[@]}" ps -q postgres 2>/dev/null || true)"
  TS_CID="$("${DC[@]}" ps -q postgres-tsdb 2>/dev/null || true)"
  RD_CORE_CID="$("${DC[@]}" ps -q redis-core 2>/dev/null || true)"
  RD_PM_CID="$("${DC[@]}" ps -q redis-pm 2>/dev/null || true)"
  NATS_CID="$("${DC[@]}" ps -q nats 2>/dev/null || true)"
  if [ -n "$PG_CID" ]; then
    docker exec "$PG_CID" pg_isready -U "${POSTGRES_USER:-omcgo}" >/dev/null 2>&1 && PG_OK=1 || PG_OK=0
  fi
  # 时序库（#347）：用 TSDB 凭据 pg_isready，与主库分别就绪判定。
  if [ -n "$TS_CID" ]; then
    docker exec "$TS_CID" pg_isready -U "${POSTGRES_TSDB_USER:-omcgo}" >/dev/null 2>&1 && TS_OK=1 || TS_OK=0
  fi
  if [ -n "$RD_CORE_CID" ]; then
    docker exec "$RD_CORE_CID" redis-cli ping >/dev/null 2>&1 && RD_CORE_OK=1 || RD_CORE_OK=0
  fi
  if [ -n "$RD_PM_CID" ]; then
    docker exec "$RD_PM_CID" redis-cli ping >/dev/null 2>&1 && RD_PM_OK=1 || RD_PM_OK=0
  fi
  if [ -n "$NATS_CID" ]; then
    [ "$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "$NATS_CID" 2>/dev/null || true)" = "healthy" ] &&
      NATS_OK=1 || NATS_OK=0
  fi
  [ "$PG_OK" = 1 ] && [ "$TS_OK" = 1 ] && [ "$RD_CORE_OK" = 1 ] && [ "$RD_PM_OK" = 1 ] && [ "$NATS_OK" = 1 ] && break
  echo "  ... ${WAIT}s (PG=$PG_OK TSDB=$TS_OK RedisCore=$RD_CORE_OK RedisPM=$RD_PM_OK NATS=$NATS_OK)"
done
if [ "$PG_OK" != 1 ] || [ "$TS_OK" != 1 ] || [ "$RD_CORE_OK" != 1 ] || [ "$RD_PM_OK" != 1 ] || [ "$NATS_OK" != 1 ]; then
  die "基础设施 90s 内未就绪：PG=$PG_OK TSDB=$TS_OK RedisCore=$RD_CORE_OK RedisPM=$RD_PM_OK NATS=$NATS_OK
  手动检查：${DC[*]} ps
            ${DC[*]} logs postgres postgres-tsdb redis-core redis-pm" 2
fi
log "基础设施已就绪 (PG / TSDB / Redis Core / Redis PM / NATS)" "Infrastructure is ready (PG / TSDB / Redis Core / Redis PM / NATS)"
if ! verify_legacy_redis_cutover; then
  die "redis-core 未继承旧 Redis 的同一数据目录和完整键数；业务写入方保持停止" 2
fi
if ! migrate_legacy_pm_redis; then
  die "旧 Redis 的 PM 聚合状态迁移到 redis-pm 失败；App/Worker 保持停止" 2
fi

if [ "$HANDOFF_PREPARED" != 1 ]; then
  log "首次部署/原 NATS 未运行：按实际流状态安全初始化 GPV RPC 固定 durable ..." "Fresh install or inactive NATS: safely initializing the fixed GPV RPC durable from the current stream state ..."
  gpv_handoff_prepare --bootstrap-if-missing ||
    die "GPV consumer handoff 失败；尚未启动 app，修复 NATS/consumer 配置后重试" 2
  HANDOFF_PREPARED=1
fi

# 7.2 验证 omcgo-net 网络已创建
log "验证 omcgo-net 网络 ..." "Verifying omcgo-net network ..."
if ! docker network inspect omcgo-net >/dev/null 2>&1; then
  die "omcgo-net 网络未创建，请检查 docker-compose.infra.yml" 2
fi

# 7.3 migrate-schema（一次性容器，跑完即退；用 .env 中的 dsn）
#     使用 `up --exit-code-from` 而非 `run`，因为 `run` 创建的一次性容器
#     存在 DNS 解析缺陷（无法解析服务名），而 `up` 作为正式服务启动时
#     网络集成完整，DNS 解析正常。保留重试逻辑作为兆底。
run_oneshot_migration() {
  local service="$1" output status
  if output="$("${DC[@]}" up --pull never --exit-code-from "$service" "$service" 2>&1)"; then
    return 0
  else
    status=$?
    printf '%s\n' "$output" >&2
    return "$status"
  fi
}

if [ "$SKIP_MIGRATE" = 0 ]; then
  log "执行 db migrate（容器：migrate-schema）..." "Running database migration (container: migrate-schema) ..."
  MIGRATE_OK=0
  for attempt in 1 2 3; do
    if run_oneshot_migration migrate-schema; then
      MIGRATE_OK=1
      break
    fi
    if [ $attempt -lt 3 ]; then
      warn "migrate 第 ${attempt} 次失败，等待 5s 重试..." "Migrate attempt ${attempt} failed; retrying in 5s ..."
      sleep 5
    fi
  done
  "${DC[@]}" rm -f migrate-schema 2>/dev/null || true
  if [ "$MIGRATE_OK" = 1 ]; then
    log "migrate 成功" "Migrate completed successfully"
  else
    die "migrate 失败（已重试 3 次）：${DC[*]} up --exit-code-from migrate-schema migrate-schema" "Migrate failed after 3 attempts: ${DC[*]} up --exit-code-from migrate-schema migrate-schema" 3
  fi

  # 7.4 seed（goose 单链路 migrations/seed/，每次部署都跑 —— goose 用
  #     goose_db_version_seed 版本表自动追踪已应用项，新加 seed 自动 catch up）
  log "执行 db seed（容器：migrate-seed-sql，goose 幂等）..." "Running database seed (container: migrate-seed-sql, idempotent) ..."
  if run_oneshot_migration migrate-seed-sql; then
    log "seed 成功" "Database seed completed successfully"
  else
    die "seed 失败：${DC[*]} up --exit-code-from migrate-seed-sql migrate-seed-sql" "Seed failed: ${DC[*]} up --exit-code-from migrate-seed-sql migrate-seed-sql" 3
  fi
  "${DC[@]}" rm -f migrate-seed-sql 2>/dev/null || true

  # 7.5 时序库 schema 迁移（容器：migrate-tsdb-schema，#347）—— 独立 TimescaleDB
  #     实例 postgres-tsdb，goose 版本表 goose_db_version_tsdb，与主库 migrate 互不影响。
  #     与 7.3 同样用 `up --exit-code-from` + 3 次重试，显式捕获失败（优于仅靠 Step 8
  #     up -d 的隐式 depends_on——后者失败时诊断信息差、无重试）。goose 幂等，升级重跑安全。
  log "执行时序库 migrate（容器：migrate-tsdb-schema，goose 幂等）..." "Running TSDB migration (container: migrate-tsdb-schema, idempotent) ..."
  TSDB_MIGRATE_OK=0
  for attempt in 1 2 3; do
    if run_oneshot_migration migrate-tsdb-schema; then
      TSDB_MIGRATE_OK=1
      break
    fi
    if [ $attempt -lt 3 ]; then
      warn "时序库 migrate 第 ${attempt} 次失败，等待 5s 重试..." "TSDB migration attempt ${attempt} failed; retrying in 5s ..."
      sleep 5
    fi
  done
  "${DC[@]}" rm -f migrate-tsdb-schema 2>/dev/null || true
  if [ "$TSDB_MIGRATE_OK" = 1 ]; then
    log "时序库 migrate 成功" "TSDB migration completed successfully"
  else
    die "时序库 migrate 失败（已重试 3 次）：${DC[*]} up --exit-code-from migrate-tsdb-schema migrate-tsdb-schema" "TSDB migration failed after 3 attempts: ${DC[*]} up --exit-code-from migrate-tsdb-schema migrate-tsdb-schema" 3
  fi

  # 当前软件尚未封版，只允许维护 000001 基线；Goose 对已经标记 version=1 的库
  # 不会重跑更新后的基线。显式执行一份只含 ADD IF NOT EXISTS / CREATE IF NOT EXISTS
  # 的兼容桥，确保已有环境先补齐新列和摘要表，再启动会写这些字段的 worker。
  # 新装环境也安全：基线已创建对象，本步骤为空操作。
  TSDB_RECONCILE_SQL="$DEPLOY_DIR/tsdb-schema-reconcile.sql"
  [ -r "$TSDB_RECONCILE_SQL" ] || die "缺少时序库兼容协调脚本：$TSDB_RECONCILE_SQL" 3
  log "执行时序库基线兼容协调（幂等）..."
  if "${DC[@]}" exec -T postgres-tsdb \
      psql -v ON_ERROR_STOP=1 -U "$POSTGRES_TSDB_USER" -d "$POSTGRES_TSDB_DB" \
      -f - < "$TSDB_RECONCILE_SQL"; then
    log "时序库基线兼容协调成功"
  else
    die "时序库基线兼容协调失败；未启动新业务容器" 3
  fi
else
  log "--skip-migrate：跳过 migrate / seed" "--skip-migrate: skipping migrations and seed"
fi

# =============================================================================
# Step 8. up 业务 + web + 监控
# =============================================================================
sep "8/9 启动业务 + web + 监控" "8/9 Start business services, web, and monitoring"

acs_ha_wait_ready() {
  local service="$1" service_cid service_ip wait_seconds=0
  while [ "$wait_seconds" -lt 90 ]; do
    service_cid="$("${DC[@]}" ps -q "$service" 2>/dev/null | head -n1)"
    service_ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$service_cid" 2>/dev/null || true)"
    if [ -n "$service_ip" ] && curl -fsS --max-time 3 "http://${service_ip}:7557/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
    wait_seconds=$((wait_seconds + 2))
  done
  return 1
}

acs_ha_report_not_ready() {
  local service="$1" service_cid service_ip state oom exit_code restart_count ready_response
  service_cid="$("${DC[@]}" ps -q "$service" 2>/dev/null | head -n1)"
  if [ -z "$service_cid" ]; then
    log "ACS readiness 诊断：$service 没有容器 ID" "ACS readiness diagnosis: no container ID for $service"
    return 0
  fi
  state="$(docker inspect -f '{{.State.Status}}' "$service_cid" 2>/dev/null || echo unknown)"
  oom="$(docker inspect -f '{{.State.OOMKilled}}' "$service_cid" 2>/dev/null || echo unknown)"
  exit_code="$(docker inspect -f '{{.State.ExitCode}}' "$service_cid" 2>/dev/null || echo unknown)"
  restart_count="$(docker inspect -f '{{.RestartCount}}' "$service_cid" 2>/dev/null || echo unknown)"
  log "ACS readiness 诊断：$service state=$state oom=$oom exit=$exit_code restarts=$restart_count" "ACS readiness diagnosis: $service state=$state oom=$oom exit=$exit_code restarts=$restart_count"
  service_ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$service_cid" 2>/dev/null || true)"
  if [ -n "$service_ip" ]; then
    ready_response="$(curl -sS --max-time 5 "http://${service_ip}:7557/readyz" 2>&1 || true)"
    log "ACS readiness 响应：${ready_response:-<无响应>}" "ACS readiness response: ${ready_response:-<no response>}"
  fi
  log "ACS 最近日志（$service，最多 80 行）：" "Recent ACS logs ($service, up to 80 lines):"
  "${DC[@]}" logs --tail=80 "$service" 2>&1 || true
}

app_wait_ready() {
  local app_cid state oom exit_code restart_count wait_seconds=0
  log "先启动 App，等待 metrics /healthz（最长 ${APP_START_TIMEOUT}s）..." "Starting App first; waiting for metrics /healthz (up to ${APP_START_TIMEOUT}s) ..."
  "${DC[@]}" up --pull never -d --no-deps app
  while [ "$wait_seconds" -lt "$APP_START_TIMEOUT" ]; do
    app_cid="$("${DC[@]}" ps -q app 2>/dev/null | head -n1)"
    if [ -n "$app_cid" ] && curl -fsS --max-time 3 http://127.0.0.1:9091/healthz >/dev/null 2>&1; then
      log "App 已就绪（${wait_seconds}s）" "App is ready (${wait_seconds}s)"
      return 0
    fi
    sleep 3
    wait_seconds=$((wait_seconds + 3))
  done

  app_cid="$("${DC[@]}" ps -q app 2>/dev/null | head -n1)"
  state="$(docker inspect -f '{{.State.Status}}' "$app_cid" 2>/dev/null || echo missing)"
  oom="$(docker inspect -f '{{.State.OOMKilled}}' "$app_cid" 2>/dev/null || echo unknown)"
  exit_code="$(docker inspect -f '{{.State.ExitCode}}' "$app_cid" 2>/dev/null || echo unknown)"
  restart_count="$(docker inspect -f '{{.RestartCount}}' "$app_cid" 2>/dev/null || echo unknown)"
  log "App 启动就绪超时：state=$state oom=$oom exit=$exit_code restarts=$restart_count" "App readiness timed out: state=$state oom=$oom exit=$exit_code restarts=$restart_count"
  log "App 最近日志（最多 80 行）：" "Recent App logs (up to 80 lines):"
  "${DC[@]}" logs --tail=80 app 2>&1 || true
  return 1
}

stop_existing_worker() {
  local worker_cid worker_state
  worker_cid="$("${DC[@]}" ps -q worker 2>/dev/null | head -n1)"
  [ -n "$worker_cid" ] || return 0
  worker_state="$(docker inspect -f '{{.State.Status}}' "$worker_cid" 2>/dev/null || echo unknown)"
  [ "$worker_state" = running ] || return 0
  log "停止现有 Worker，避免 App 启动期争抢数据库连接 ..." "Stopping the existing Worker to avoid database contention during App startup ..."
  "${DC[@]}" stop worker ||
    die "现有 Worker 停止失败；为避免 App 启动期数据库争抢，已中止升级" "Could not stop the existing Worker; upgrade aborted to avoid database contention during App startup" 2
}

web_acs_dynamic_upstream_loaded() {
  local web_cid="$1" rendered
  rendered="$(docker exec "$web_cid" nginx -T 2>&1)" || return 1
  printf '%s\n' "$rendered" | grep -Fq 'server acs:7557 resolve;' || return 1
  printf '%s\n' "$rendered" | grep -Fq 'zone acs_backend' || return 1
}

APP_START_TIMEOUT="${OMC_APP_START_TIMEOUT:-180}"
case "$APP_START_TIMEOUT" in
  ''|*[!0-9]*) die "OMC_APP_START_TIMEOUT 必须是正整数" 2 ;;
esac
[ "$APP_START_TIMEOUT" -gt 0 ] || die "OMC_APP_START_TIMEOUT 必须大于 0" 2

ACS_HA_EXISTING=0
acs_ha_prepare_candidate() {
  local old_acs old_web wait_seconds
  old_acs="$("${DC[@]}" ps -q acs 2>/dev/null | head -n1)"
  old_web="$("${DC[@]}" ps -q web 2>/dev/null | head -n1)"
  # 首次安装没有旧流量入口，由下面的完整 up 同时创建双实例即可。
  [ -n "$old_acs" ] || return 0
  ACS_HA_EXISTING=1
  if [ "$SKIP_WEB" = 1 ]; then
    die "--skip-web 不支持存量 ACS 无损升级；请启用 web 网关，或由外部负载均衡完成双实例切换" 2
  fi
  [ -n "$old_web" ] || die "存量 ACS 正在运行但 web 网关不存在，无法执行无损发布" 2

  # 兼容首轮从旧拓扑升级：旧 web 若尚未使用动态 DNS upstream，候选实例即使
  # 就绪也不会被纳入路由。先在旧 ACS 仍服务时只刷新 web，再继续接力。
  if ! web_acs_dynamic_upstream_loaded "$old_web"; then
    log "旧 web 尚未加载动态 ACS upstream，先刷新 web 动态 ACS upstream ..." "The existing web has no dynamic ACS upstream; refreshing web first ..."
    "${DC[@]}" up --pull never -d --no-deps web
    wait_seconds=0
    while [ "$wait_seconds" -lt 30 ]; do
      old_web="$("${DC[@]}" ps -q web 2>/dev/null | head -n1)"
      if [ -n "$old_web" ] && web_acs_dynamic_upstream_loaded "$old_web"; then
        break
      fi
      sleep 2
      wait_seconds=$((wait_seconds + 2))
    done
    web_acs_dynamic_upstream_loaded "$old_web" ||
      die "web 30s 内未加载动态 ACS upstream；正式 ACS 未替换，已中止发布" 2
  fi

  log "先更新 ACS 接力实例，正式 ACS 继续承载现有流量 ..." "Updating the ACS candidate first; the primary ACS continues serving traffic ..."
  "${DC[@]}" up --pull never -d --no-deps acs-candidate
  if ! acs_ha_wait_ready acs-candidate; then
    acs_ha_report_not_ready acs-candidate
    die "ACS 接力实例 90s 内未就绪；正式 ACS 未替换，已中止发布" 2
  fi
  # nginx.conf 的 Docker DNS valid=10s。候选实例刚加入共享别名 `acs` 时，
  # 必须覆盖完整缓存周期后再停止正式实例，否则旧 worker 仍可能只持有旧 IP。
  log "ACS 接力实例已就绪，等待 Nginx 动态 DNS 完成一轮刷新 ..." "ACS candidate is ready; waiting for one Nginx dynamic DNS refresh cycle ..."
  sleep 12
  if ! acs_ha_wait_ready acs-candidate; then
    acs_ha_report_not_ready acs-candidate
    die "DNS 刷新后 ACS 接力实例已不就绪；正式 ACS 未替换，已中止发布" 2
  fi
  log "Nginx 已具备接力上游，允许替换正式 ACS" "Nginx has the failover upstream; primary ACS replacement is allowed"
}

acs_ha_prepare_candidate

if [ "$ACS_HA_EXISTING" = 1 ]; then
  # 不再调用无服务范围的 compose up：实测 Compose 会把刚预热的 candidate
  # 与 primary 同时重建，破坏接力不变量。正式实例必须单独替换、直连业务端口
  # 验证就绪，再覆盖一轮 Nginx DNS 缓存；其余业务显式 --no-deps 排除两个 ACS。
  log "接力实例持续承载流量，单独替换正式 ACS ..." "The candidate continues serving traffic; replacing the primary ACS separately ..."
  "${DC[@]}" up --pull never -d --no-deps acs
  if ! acs_ha_wait_ready acs; then
    acs_ha_report_not_ready acs
    die "正式 ACS 90s 内未就绪；接力实例仍在服务，已中止其余业务更新" 2
  fi
  log "正式 ACS 已就绪，等待 Nginx 动态 DNS 完成一轮刷新 ..." "Primary ACS is ready; waiting for one Nginx dynamic DNS refresh cycle ..."
  sleep 12
  if ! acs_ha_wait_ready acs || ! acs_ha_wait_ready acs-candidate; then
    acs_ha_report_not_ready acs
    acs_ha_report_not_ready acs-candidate
    die "ACS 双实例在 DNS 刷新后未全部就绪，已中止其余业务更新" 2
  fi

  stop_existing_worker
  if ! app_wait_ready; then
    die "App ${APP_START_TIMEOUT}s 内未就绪；Worker/Web 尚未启动，请根据上方 App 日志排查数据库超时或资源不足" 4
  fi
  remaining_services=(worker)
  [ "$SKIP_WEB" = 1 ] || remaining_services+=(web)
  log "App 已就绪，启动 Worker/Web（显式排除 ACS 依赖）..." "App is ready; starting Worker/Web (excluding ACS dependencies) ..."
  "${DC[@]}" up --pull never -d --no-deps "${remaining_services[@]}"
else
  # 首次安装没有存量南向流量，但仍先让 App 完成启动期数据库同步，避免 Worker 并发抢占连接。
  log "首次安装：先启动 ACS 双实例 ..." "Fresh install: starting both ACS instances first ..."
  "${DC[@]}" up --pull never -d --no-deps acs acs-candidate
  if ! acs_ha_wait_ready acs || ! acs_ha_wait_ready acs-candidate; then
    acs_ha_report_not_ready acs
    acs_ha_report_not_ready acs-candidate
    die "首次安装 ACS 双实例未就绪，未启动 App/Worker/Web" 2
  fi
  if ! app_wait_ready; then
    die "App ${APP_START_TIMEOUT}s 内未就绪；Worker/Web 尚未启动，请根据上方 App 日志排查数据库超时或资源不足" 4
  fi
  remaining_services=(worker)
  [ "$SKIP_WEB" = 1 ] || remaining_services+=(web)
  log "App 已就绪，启动 Worker/Web ..." "App is ready; starting Worker/Web ..."
  "${DC[@]}" up --pull never -d --no-deps "${remaining_services[@]}"
fi

# Compose records the resolved bind-mount source inode when a container is
# created. OMC_ROOT/current is switched to the new immutable release above,
# but an unchanged monitoring image/config leaves the old container attached
# to the previous release directory. Recreate only the stateless services that
# mount release-local configuration; --no-deps protects all data services and
# named volumes remain attached.
if [ "$SKIP_MONITORING" = 0 ]; then
  log "刷新版本目录 bind mount（仅监控无状态容器，保留数据卷）..." "Refreshing release bind mounts (monitoring stateless containers only; data volumes preserved) ..."
  "${DC[@]}" up --pull never -d --force-recreate --no-deps prometheus alertmanager grafana loki otelcol tempo \
    nats-exporter nginx-exporter node-exporter cadvisor
fi

HEALTHCHECK_INTERVAL=5
HEALTHCHECK_TIMEOUT="${OMC_HEALTHCHECK_TIMEOUT:-90}"
HEALTHCHECK_FINAL_GRACE="${OMC_HEALTHCHECK_FINAL_GRACE:-0}"
HEALTHCHECK_PROBE_TIMEOUT="${OMC_HEALTHCHECK_PROBE_TIMEOUT:-30}"
case "$HEALTHCHECK_TIMEOUT" in ''|*[!0-9]*) die "OMC_HEALTHCHECK_TIMEOUT 必须是正整数" 1 ;; esac
case "$HEALTHCHECK_FINAL_GRACE" in ''|*[!0-9]*) die "OMC_HEALTHCHECK_FINAL_GRACE 必须是非负整数" 1 ;; esac
case "$HEALTHCHECK_PROBE_TIMEOUT" in ''|*[!0-9]*) die "OMC_HEALTHCHECK_PROBE_TIMEOUT 必须是正整数" 1 ;; esac
[ "$HEALTHCHECK_TIMEOUT" -gt 0 ] || die "OMC_HEALTHCHECK_TIMEOUT 必须大于 0" 1
[ "$HEALTHCHECK_PROBE_TIMEOUT" -gt 0 ] || die "OMC_HEALTHCHECK_PROBE_TIMEOUT 必须大于 0" 1
log "动态等待业务容器启动（最长 ${HEALTHCHECK_TIMEOUT}s，单轮探针最多 ${HEALTHCHECK_PROBE_TIMEOUT}s，每 ${HEALTHCHECK_INTERVAL}s 重试；通过后立即继续）..." "Waiting for business containers (up to ${HEALTHCHECK_TIMEOUT}s, retry every ${HEALTHCHECK_INTERVAL}s) ..."
HEALTHCHECK_LOG="$(mktemp)"
HEALTH_OK=0
HEALTHCHECK_PROBE_TIMEOUTS=0
HEALTHCHECK_DEADLINE=$(( $(date +%s) + HEALTHCHECK_TIMEOUT ))
while :; do
  HEALTHCHECK_REMAINING=$(( HEALTHCHECK_DEADLINE - $(date +%s) ))
  [ "$HEALTHCHECK_REMAINING" -gt 0 ] || break
  # 单轮探针必须有独立上限；某个 docker exec/网络探针卡住时，仍要回到循环
  # 继续重试，而不能独占整个总等待窗口。
  HEALTHCHECK_PROBE_REMAINING="$HEALTHCHECK_PROBE_TIMEOUT"
  [ "$HEALTHCHECK_REMAINING" -lt "$HEALTHCHECK_PROBE_REMAINING" ] &&
    HEALTHCHECK_PROBE_REMAINING="$HEALTHCHECK_REMAINING"
  if timeout "${HEALTHCHECK_PROBE_REMAINING}s" bash "$OMC_ROOT/current/deploy/healthcheck.sh" --lang "$OMC_LANG" --startup >"$HEALTHCHECK_LOG" 2>&1; then
    HEALTH_OK=1
    break
  elif [ "$?" -eq 124 ]; then
    HEALTHCHECK_PROBE_TIMEOUTS=$((HEALTHCHECK_PROBE_TIMEOUTS + 1))
    log "健康检查单轮超过 ${HEALTHCHECK_PROBE_REMAINING}s，继续第 ${HEALTHCHECK_PROBE_TIMEOUTS} 次重试 ..."
  fi
  HEALTHCHECK_REMAINING=$(( HEALTHCHECK_DEADLINE - $(date +%s) ))
  [ "$HEALTHCHECK_REMAINING" -gt 0 ] || break
  HEALTHCHECK_SLEEP="$HEALTHCHECK_INTERVAL"
  [ "$HEALTHCHECK_REMAINING" -lt "$HEALTHCHECK_SLEEP" ] && HEALTHCHECK_SLEEP="$HEALTHCHECK_REMAINING"
  sleep "$HEALTHCHECK_SLEEP"
done

# 初始化期间监控/字典/业务端点可能恰好跨过主窗口；再给一次短复核，避免把
# “容器已稳定、端点刚完成启动”误报为安装失败。最终仍以完整 healthcheck 为准。
if [ "$HEALTH_OK" -eq 0 ] && [ "$HEALTHCHECK_FINAL_GRACE" -gt 0 ]; then
  log "主健康等待窗口结束，进行 ${HEALTHCHECK_FINAL_GRACE}s 最终复核 ..."
  HEALTHCHECK_FINAL_PROBE_TIMEOUT="$HEALTHCHECK_FINAL_GRACE"
  [ "$HEALTHCHECK_FINAL_PROBE_TIMEOUT" -gt "$HEALTHCHECK_PROBE_TIMEOUT" ] &&
    HEALTHCHECK_FINAL_PROBE_TIMEOUT="$HEALTHCHECK_PROBE_TIMEOUT"
  if timeout "${HEALTHCHECK_FINAL_PROBE_TIMEOUT}s" bash "$OMC_ROOT/current/deploy/healthcheck.sh" --lang "$OMC_LANG" >"$HEALTHCHECK_LOG" 2>&1; then
    HEALTH_OK=1
  fi
fi

# =============================================================================
# Step 9. healthcheck
# =============================================================================
sep "9/9 健康检查" "9/9 Health check"

if [ "$HEALTH_OK" -eq 1 ]; then
  cat "$HEALTHCHECK_LOG"
else
  warn "健康检查在 ${HEALTHCHECK_TIMEOUT}s 主窗口 + ${HEALTHCHECK_FINAL_GRACE}s 最终复核内未通过"
  [ "$HEALTHCHECK_PROBE_TIMEOUTS" -gt 0 ] &&
    warn "其中 ${HEALTHCHECK_PROBE_TIMEOUTS} 轮健康检查因单轮超时结束；这不等同于业务容器异常"
  log "健康检查失败摘要（仅显示失败项）："
  if ! awk '/  \[FAIL\]/ || /^结果：/ || /^存在失败项/ { print; found=1 } END { exit(found ? 0 : 1) }' "$HEALTHCHECK_LOG"; then
    log "  （健康检查未返回结构化失败项，可能在单轮超时中断）"
  fi

  unstable_services=()
  for service in app acs acs-candidate worker; do
    service_cid="$("${DC[@]}" ps -q "$service" 2>/dev/null | head -n1)"
    if [ -z "$service_cid" ]; then
      unstable_services+=("$service:missing")
      continue
    fi
    service_state="$(docker inspect -f '{{.State.Status}}' "$service_cid" 2>/dev/null || echo unknown)"
    service_oom="$(docker inspect -f '{{.State.OOMKilled}}' "$service_cid" 2>/dev/null || echo unknown)"
    service_exit="$(docker inspect -f '{{.State.ExitCode}}' "$service_cid" 2>/dev/null || echo unknown)"
    service_restarts="$(docker inspect -f '{{.RestartCount}}' "$service_cid" 2>/dev/null || echo unknown)"
    log "  $service：state=$service_state oom=$service_oom exit=$service_exit restarts=$service_restarts"
    if [ "$service_state" != running ] || [ "$service_oom" = true ] || [ "$service_exit" != 0 ] || [ "$service_restarts" != 0 ]; then
      unstable_services+=("$service")
    fi
  done
  if [ "${#unstable_services[@]}" -gt 0 ]; then
    log "异常业务容器最近日志（每个最多 20 行）："
    for service in "${unstable_services[@]}"; do
      service="${service%%:*}"
      log "  $service："
      "${DC[@]}" logs --tail=20 "$service" 2>&1 || true
    done
  else
    log "业务容器均稳定运行，跳过正常运行日志；请稍后重试 healthcheck.sh 获取完整状态。"
  fi
fi
rm -f "$HEALTHCHECK_LOG"

echo
sep "部署完成" "Installation complete"
log "项目版本：$VERSION" "Project version: $VERSION"
log "OMC 根目录：$OMC_ROOT" "OMC root: $OMC_ROOT"
log "  current → $(readlink -f "$OMC_ROOT/current")" "  current -> $(readlink -f "$OMC_ROOT/current")"
log "  实例配置：$OMC_ROOT/etc/" "  Instance configuration: $OMC_ROOT/etc/"
log "  运行日志：$OMC_ROOT/run/logs/{app,acs,worker,nginx}" "  Runtime logs: $OMC_ROOT/run/logs/{app,acs,worker,nginx}"
echo
log "compose 控制命令：" "Compose control commands:"
log "  · 查看状态：  cd $OMC_ROOT/current/deploy && ${DC[*]} ps" "  · Status: cd $OMC_ROOT/current/deploy && ${DC[*]} ps"
log "  · 查看日志：  cd $OMC_ROOT/current/deploy && ${DC[*]} logs -f <service>" "  · Logs: cd $OMC_ROOT/current/deploy && ${DC[*]} logs -f <service>"
log "  · 停止全栈：  cd $OMC_ROOT/current/deploy && ${DC[*]} down" "  · Stop stack: cd $OMC_ROOT/current/deploy && ${DC[*]} down"
log "  · 卸载(保留数据)：sudo bash $OMC_ROOT/current/deploy/uninstall.sh --force" "  · Uninstall (keep data): sudo bash $OMC_ROOT/current/deploy/uninstall.sh --force"
echo
log "访问地址：" "Access URLs:"
log "  · Web 管理界面：  http://<服务器IP>:8081   （运维浏览器登录）" "  · Web console: http://<server-ip>:8081 (operator login)"
log "  · 基站 ACS URL：  http://<服务器IP>:8080   （TR-069，基站设备侧填，人不浏览）" "  · ACS URL for devices: http://<server-ip>:8080 (TR-069)"
log "  · MinIO Console：http://<服务器IP>:9001" "  · MinIO Console: http://<server-ip>:9001"
[ "$SKIP_MONITORING" = 0 ] && log "  · Grafana：       http://<服务器IP>:3030   （宿主 3030 → 容器 3000）" "  · Grafana: http://<server-ip>:3030"
log "  · 健康检查：       bash $OMC_ROOT/current/deploy/healthcheck.sh" "  · Health check: bash $OMC_ROOT/current/deploy/healthcheck.sh"
echo
log "初始账号（首次登录强制改）：" "Initial accounts (change on first login):"
log "  · Web UI：    admin / admin123" "  · Web UI: admin / admin123"
log "  · MinIO：     minioadmin / minioadmin" "  · MinIO: minioadmin / minioadmin"
log "  · PostgreSQL：omcgo / omcgo123" "  · PostgreSQL: omcgo / omcgo123"
[ "$SKIP_MONITORING" = 0 ] && log "  · Grafana：    admin / admin" "  · Grafana: admin / admin"
echo

if [ "$HEALTH_OK" != 1 ]; then
  die "健康检查未全通过（部分服务异常），请按上面 healthcheck 输出排查" 4
fi
log "全部 OK 🎉" "All checks passed."

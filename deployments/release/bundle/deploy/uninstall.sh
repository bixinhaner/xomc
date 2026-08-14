#!/usr/bin/env bash
# =============================================================================
# OMC 卸载脚本 — 内网交付侧（全 docker compose 部署）
#
# ★ 默认【保留数据】：拆掉正在跑的栈（容器 / 网络 / 业务镜像 / 代码目录），但
#   保留所有持久化数据与凭据，使下次 install.sh 能直接复用现有 DB / MinIO：
#     · 保留：所有数据卷（pgdata/tsdbdata/miniodata/redisdata/natsdata + 监控各卷）
#     · 保留：$OMC_ROOT/data（UI 上传的自定义参数模型/指标/告警 XML + 升级快照）
#     · 保留：$OMC_ROOT/etc（实例配置）+ 凭据快照 .env.saved + 主凭据 secrets.env（#175，
#             密钥唯一权威源；与数据卷同生命周期：默认保留 → reinstall 沿用同套口令连得上旧卷）
#             （口令 / JWT / OMC_PUBLIC_HOST —— install.sh 重装时据此继承，
#              避免新口令与旧数据卷不匹配连不上 DB）
#     · 删除：容器 / docker 网络 / omcgo/* 业务镜像 / 代码运行目录
#             ($OMC_ROOT/{current,releases,run,packages})
#
# ★ --purge：彻底清除（连数据卷 + 整个 $OMC_ROOT 一并删，不可恢复）。
# ★ 默认 dry-run，仅打印将做的动作；加 --force 才真删（并二次确认）。
# ★ 用 root 执行。安装 / 升级请用同目录的 install.sh。
#
# 用法：
#   sudo bash deploy/uninstall.sh                 # dry-run，列出"保留数据"卸载将做什么
#   sudo bash deploy/uninstall.sh --force         # 真卸载，保留数据/凭据（可重装复用）
#   sudo bash deploy/uninstall.sh --force --keep-images   # 同上但保留 omcgo/* 业务镜像
#   sudo bash deploy/uninstall.sh --purge         # dry-run，列出"彻底清除"将做什么
#   sudo bash deploy/uninstall.sh --purge --force # 彻底清除：连数据卷 + $OMC_ROOT 全删
#   sudo bash deploy/uninstall.sh -h | --help     # 本帮助
#
# 参数：
#   --force        关闭 dry-run（必须显式加才会真删，且会再做一次交互确认）
#   --purge        彻底清除：删数据卷 + 整个 $OMC_ROOT（默认仅保留数据卸载）
#   --keep-images  不删 omcgo/* 业务镜像
#   --omc-root <p> OMC 安装根（默认 /opt/omc）
#   --lang <cn|en>  输出语言（默认 en）
#   --yes          跳过交互确认（CI / 批处理）
#   -h | --help    本帮助
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
OMC_LANG="${OMC_LANG:-en}"

localize() {
  local cn="$1" en="${2:-$1}"
  if [ "$OMC_LANG" = en ]; then printf '%s' "$en"; else printf '%s' "$cn"; fi
}
log()  { echo -e "\033[1;32m[uninstall]\033[0m $(localize "$1" "${2:-$1}")"; }
warn() { echo -e "\033[1;33m[uninstall][$( [ "$OMC_LANG" = en ] && echo Warning || echo 警告 )]\033[0m $(localize "$1" "${2:-$1}")" >&2; }
die()  {
  local cn="$1" en="$1" code=1
  if [[ "${2:-}" =~ ^[0-9]+$ ]]; then code="$2"; else en="${2:-$1}"; code="${3:-1}"; fi
  echo -e "\033[1;31m[uninstall][$( [ "$OMC_LANG" = en ] && echo Error || echo 错误 )]\033[0m $(localize "$cn" "$en")" >&2
  exit "$code"
}

# force_remove_volumes <多行卷名（每行一个）> —— 逐个 docker volume rm -f 强删。
# 背景：`compose down -v` 的删卷步骤偶发卡死在「Removing …」长时间不返回
# （卷被残留挂载/进程占用，或 dockerd 删除协程 wedge），导致整个卸载挂住。
# 故卸载时不给 compose down 传 -v，改由本函数把删卷拿出来可控处理：
#   1) docker volume rm -f（带 timeout，避免单卷卡死拖垮全流程）
#   2) 卡住则 lazy-umount 其挂载点释放占用后再重试一次
force_remove_volumes() {
  local vols="$1" vol mp TO=""
  command -v timeout >/dev/null 2>&1 && TO="timeout 30"
  while IFS= read -r vol; do
    [ -z "$vol" ] && continue
    docker volume inspect "$vol" >/dev/null 2>&1 || continue   # 已不存在
    if $TO docker volume rm -f "$vol" >/dev/null 2>&1; then
      log "        · 已删卷 $vol" "        · Removed volume $vol"
      continue
    fi
    warn "卷 $vol 删不掉，尝试 lazy-umount 其挂载点后重试 ..." "Unable to remove volume $vol; retrying after lazy-unmounting its mount point ..."
    mp=$(docker volume inspect -f '{{.Mountpoint}}' "$vol" 2>/dev/null || true)
    [ -n "$mp" ] && mount | grep -q " $mp " && umount -l "$mp" 2>/dev/null || true
    if $TO docker volume rm -f "$vol" >/dev/null 2>&1; then
      log "        · 已删卷 $vol（重试成功）" "        · Removed volume $vol (retry succeeded)"
    else
      warn "卷 $vol 仍删除失败，请手动处理：docker volume rm -f $vol" "Still unable to remove volume $vol; handle it manually: docker volume rm -f $vol"
      warn "  （若仍卡，systemctl restart docker 清掉 wedge 的删除协程后再删，或 rm -rf 其挂载点目录）" "  (If it remains stuck, restart Docker with systemctl restart docker, then retry; or remove the mount-point directory with rm -rf.)"
    fi
  done <<< "$vols"
}

# ── 参数解析 ────────────────────────────────────────────────────────────
FORCE=0
PURGE=0
KEEP_IMAGES=0
ASSUME_YES=0
OMC_ROOT="/opt/omc"
COMPOSE_PROJECT="omcgo"

while [ $# -gt 0 ]; do
  case "$1" in
    --force)       FORCE=1; shift ;;
    --purge)       PURGE=1; shift ;;
    --keep-images) KEEP_IMAGES=1; shift ;;
    --omc-root)    OMC_ROOT="$2"; shift 2 ;;
    --lang|--language) OMC_LANG="${2:?--lang 需要 cn 或 en}"; shift 2 ;;
    --lang=*|--language=*) OMC_LANG="${1#*=}"; shift ;;
    --yes)         ASSUME_YES=1; shift ;;
    # 兼容旧 deploy.sh --uninstall 习惯：--keep-data 已是默认，接受为 no-op。
    --keep-data)   shift ;;
    -h|--help)
      if [ "$OMC_LANG" = en ]; then
        cat <<'EOF'
OMC uninstall script

Usage:
  sudo bash deploy/uninstall.sh [options]

Options:
  --force              Disable dry-run and perform the uninstall
  --purge              Delete persistent data and the entire OMC root
  --keep-images        Keep omcgo/* business images
  --omc-root <path>    OMC installation root (default /opt/omc)
  --lang <cn|en>       Output language (default en)
  --yes                Skip interactive confirmation
  -h, --help           Show this help
EOF
      else
        sed -n '3,40p' "$0"
      fi
      exit 0 ;;
    *)             die "未知参数：$1（-h 查看用法）" "Unknown option: $1 (see -h for usage)" ;;
  esac
done

case "$OMC_LANG" in
  cn|en) ;;
  *) die "--lang 仅支持 cn 或 en，收到：$OMC_LANG" "--lang accepts only cn or en, got: $OMC_LANG" 2 ;;
esac

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）" "Run as root (sudo bash $0 ...)." 1

confirm() {
  [ "$ASSUME_YES" = 1 ] && return 0
  local yn prompt_cn="$1" prompt_en="${2:-$1}"
  read -rp "$(localize "$prompt_cn" "$prompt_en") [Y/n] " yn
  case "${yn:-Y}" in [Yy]*|"") return 0 ;; *) return 1 ;; esac
}

REAL=$FORCE
DRY=""
[ "$REAL" = 0 ] && DRY="[dry-run] "

command -v docker >/dev/null 2>&1 || die "未检测到 docker,无法卸载" "Docker was not found; uninstall cannot continue." 1

# compose 命令探测（仅用于 down 容器；探测不到则用 docker rm -f 兜底）
if docker compose version >/dev/null 2>&1; then
  UN_COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  UN_COMPOSE="docker-compose"
else
  warn "未检测到 compose 命令;容器停止步骤将用 docker rm -f 兜底" "Compose was not found; the container stop step will fall back to docker rm -f."
  UN_COMPOSE=""
fi

# ── 收集"将要处理"的清单 ─────────────────────────────────────────────────
log "扫描当前状态 ..." "Scanning the current state ..."
RUNNING_CONTAINERS="$(docker ps -a --filter "label=com.docker.compose.project=$COMPOSE_PROJECT" --format '{{.Names}}' 2>/dev/null || true)"
VOLUMES_LIST="$(docker volume ls -q --filter "label=com.docker.compose.project=$COMPOSE_PROJECT" 2>/dev/null || true)"
NETWORKS_LIST="$(docker network ls -q --filter "label=com.docker.compose.project=$COMPOSE_PROJECT" 2>/dev/null || true)"
BUSINESS_IMAGES_LIST="$(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep '^omcgo/' || true)"
OMC_SIZE=""
[ -d "$OMC_ROOT" ] && OMC_SIZE="$(du -sh "$OMC_ROOT" 2>/dev/null | awk '{print $1}')"

echo
if [ "$PURGE" = 1 ]; then
  log "${DRY}卸载计划:【--purge 彻底清除,不可恢复】" "${DRY}Uninstall plan: [--purge permanently removes everything and cannot be undone]"
else
  log "${DRY}卸载计划:【默认 保留数据,可重装复用】" "${DRY}Uninstall plan: [default keeps data for reuse after reinstall]"
fi
log "${DRY}  · 项目:$COMPOSE_PROJECT" "${DRY}  · Project: $COMPOSE_PROJECT"
log "${DRY}  · OMC 根目录:$OMC_ROOT $([ -n "$OMC_SIZE" ] && echo "($OMC_SIZE)")" "${DRY}  · OMC root: $OMC_ROOT $([ -n "$OMC_SIZE" ] && echo "($OMC_SIZE)")"
if [ -n "$UN_COMPOSE" ]; then
  COMPOSE_DISPLAY="$UN_COMPOSE"
  COMPOSE_DISPLAY_EN="$UN_COMPOSE"
else
  COMPOSE_DISPLAY="docker rm -f 兜底"
  COMPOSE_DISPLAY_EN="docker rm -f fallback"
fi
log "${DRY}  · compose 命令:$COMPOSE_DISPLAY" "${DRY}  · Compose command: $COMPOSE_DISPLAY_EN"
echo

# 1) 容器
if [ -n "$RUNNING_CONTAINERS" ]; then
  CONTAINER_COUNT="$(echo "$RUNNING_CONTAINERS" | wc -l | tr -d ' ')"
  log "${DRY}1) 将停止 + 删除以下容器(${CONTAINER_COUNT} 个):" "${DRY}1) The following ${CONTAINER_COUNT} containers will be stopped and removed:"
  echo "$RUNNING_CONTAINERS" | sed 's/^/     - /'
else
  log "${DRY}1) 无运行中容器(可能已 down 过或本就没起)" "${DRY}1) No running containers (already down or never started)"
fi

# 2) 数据卷
if [ "$PURGE" = 1 ]; then
  if [ -n "$VOLUMES_LIST" ]; then
    VOLUME_COUNT="$(echo "$VOLUMES_LIST" | wc -l | tr -d ' ')"
    log "${DRY}2) 将删除以下数据卷(${VOLUME_COUNT} 个,含 pg/minio/redis/nats/监控 所有持久化数据):" "${DRY}2) The following ${VOLUME_COUNT} data volumes will be deleted, including all PostgreSQL, MinIO, Redis, NATS, and monitoring data:"
    echo "$VOLUMES_LIST" | sed 's/^/     - /'
  else
    log "${DRY}2) 无数据卷可删" "${DRY}2) No data volumes to delete"
  fi
else
  VOLUME_COUNT="$(echo "$VOLUMES_LIST" | grep -c . || echo 0)"
  log "${DRY}2) 保留以下数据卷(${VOLUME_COUNT} 个,重装可复用):" "${DRY}2) The following ${VOLUME_COUNT} data volumes will be kept for reuse after reinstall:"
  [ -n "$VOLUMES_LIST" ] && echo "$VOLUMES_LIST" | sed 's/^/     · /'
fi

# 3) 网络
if [ -n "$NETWORKS_LIST" ]; then
  NETWORK_COUNT="$(echo "$NETWORKS_LIST" | wc -l | tr -d ' ')"
  log "${DRY}3) 将删除以下 docker 网络(${NETWORK_COUNT} 个):" "${DRY}3) The following ${NETWORK_COUNT} Docker networks will be deleted:"
  echo "$NETWORKS_LIST" | sed 's/^/     - /'
fi

# 4) 业务镜像
if [ "$KEEP_IMAGES" = 1 ]; then
  IMAGE_COUNT="$(echo "$BUSINESS_IMAGES_LIST" | grep -c . || echo 0)"
  log "${DRY}4) --keep-images 保留 omcgo/* 业务镜像(${IMAGE_COUNT} 个)" "${DRY}4) --keep-images keeps ${IMAGE_COUNT} omcgo/* business images"
else
  if [ -n "$BUSINESS_IMAGES_LIST" ]; then
    IMAGE_COUNT="$(echo "$BUSINESS_IMAGES_LIST" | wc -l | tr -d ' ')"
    log "${DRY}4) 将删除以下业务镜像(${IMAGE_COUNT} 个):" "${DRY}4) The following ${IMAGE_COUNT} business images will be deleted:"
    echo "$BUSINESS_IMAGES_LIST" | sed 's/^/     - /'
  else
    log "${DRY}4) 无 omcgo/* 业务镜像可删" "${DRY}4) No omcgo/* business images to delete"
  fi
fi

# 5) OMC 根目录
if [ -d "$OMC_ROOT" ]; then
  if [ "$PURGE" = 1 ]; then
    log "${DRY}5) 将删除整个 OMC 根目录:$OMC_ROOT(含 data/ etc/ current/ releases/ run/ 等全部)" "${DRY}5) The entire OMC root will be deleted: $OMC_ROOT (including data/, etc/, current/, releases/, run/, and everything else)"
  else
    log "${DRY}5) 将删除 $OMC_ROOT 下 current/ releases/ run/ packages/(代码运行目录);" "${DRY}5) The following runtime directories will be deleted under $OMC_ROOT: current/, releases/, run/, packages/"
    log "${DRY}   保留 $OMC_ROOT/data(外置 XML + 升级快照) + $OMC_ROOT/etc(实例配置) + 凭据快照 .env.saved" "${DRY}   Keeping $OMC_ROOT/data (external XML and upgrade snapshots), $OMC_ROOT/etc (instance configuration), and the .env.saved credential snapshot"
  fi
else
  log "${DRY}5) $OMC_ROOT 不存在" "${DRY}5) $OMC_ROOT does not exist"
fi

echo
log "${DRY}注意:本脚本【不卸载 Docker 引擎本身】,如需卸载请额外跑 install-docker.sh --uninstall" "${DRY}Note: this script does not uninstall the Docker engine. To uninstall it, run install-docker.sh --uninstall separately."
if [ "$PURGE" = 1 ]; then
  log "${DRY}注意:--purge 数据卷与 $OMC_ROOT 一旦删除,数据库 / 对象存储 / 监控历史 / 自定义 XML 全部不可恢复" "${DRY}Warning: once --purge deletes the volumes and $OMC_ROOT, databases, object storage, monitoring history, and custom XML cannot be recovered."
fi
echo

if [ "$REAL" = 0 ]; then
  log "[dry-run] 未实际执行任何动作。要真删,加 --force 再跑一次:" "[dry-run] No actions were performed. Add --force and run again to delete:"
  EXTRA_ARGS="$([ "$PURGE" = 1 ] && echo "--purge ")$([ "$KEEP_IMAGES" = 1 ] && echo "--keep-images ")"
  log "[dry-run]   sudo bash $0 --force $EXTRA_ARGS" "[dry-run]   sudo bash $0 --force $EXTRA_ARGS"
  exit 0
fi

# ── 真删 — 二次交互确认(--yes 跳过)──────────────────────────────────────
if [ "$ASSUME_YES" != 1 ]; then
  echo
  if [ "$PURGE" = 1 ]; then
    warn "【--purge 彻底清除】以上操作不可逆,数据卷 + $OMC_ROOT 全删,确认后立即执行。" "[--purge permanent deletion] These actions are irreversible; all data volumes and $OMC_ROOT will be deleted after confirmation."
    confirm "确认彻底清除(连数据)?(默认 N)" "Confirm permanent deletion, including data? (default N)" || { log "用户取消,未执行任何操作。" "Cancelled by user; no actions were performed."; exit 0; }
  else
    warn "以上操作【不可逆】(容器/网络/业务镜像/代码目录),数据卷与 $OMC_ROOT/data、etc 保留。" "These actions are irreversible (containers, networks, business images, and runtime directories); data volumes and $OMC_ROOT/data and etc will be kept."
    confirm "确认卸载(保留数据)?(默认 N)" "Confirm uninstall while keeping data? (default N)" || { log "用户取消,未执行任何操作。" "Cancelled by user; no actions were performed."; exit 0; }
  fi
fi

# 真删前 cd 到 /tmp,避免 cwd 在被删目录(脚本本体所在的 deploy/ 也属于 $OMC_ROOT)
cd /tmp

# 1) compose down(stop + remove 容器/网络;不传 -v,删卷交给下面可控处理)
if [ -n "$UN_COMPOSE" ] && [ -d "$DEPLOY_DIR" ]; then
  log "[1/5] $UN_COMPOSE down(stop + remove 容器/网络) ..." "[1/5] $UN_COMPOSE down (stop and remove containers/networks) ..."
  DOWN_FILES=( -f "$DEPLOY_DIR/docker-compose.infra.yml" -f "$DEPLOY_DIR/docker-compose.app.yml" )
  [ -f "$DEPLOY_DIR/docker-compose.web.yml" ]        && DOWN_FILES+=( -f "$DEPLOY_DIR/docker-compose.web.yml" )
  [ -f "$DEPLOY_DIR/docker-compose.monitoring.yml" ] && DOWN_FILES+=( -f "$DEPLOY_DIR/docker-compose.monitoring.yml" )
  ( cd "$DEPLOY_DIR" && $UN_COMPOSE -p "$COMPOSE_PROJECT" "${DOWN_FILES[@]}" down --remove-orphans ) || warn "compose down 报错,继续" "Compose down returned an error; continuing."
elif [ -n "$RUNNING_CONTAINERS" ]; then
  log "[1/5] compose 文件丢失,fallback 用 docker rm -f 强删容器 ..." "[1/5] Compose files are missing; falling back to docker rm -f to force-remove containers ..."
  echo "$RUNNING_CONTAINERS" | xargs -r docker rm -f >/dev/null 2>&1 || true
  [ -n "$NETWORKS_LIST" ] && echo "$NETWORKS_LIST" | xargs -r docker network rm >/dev/null 2>&1 || true
fi

# 1b) 数据卷:仅 --purge 时强删;默认保留
if [ "$PURGE" = 1 ] && [ -n "$VOLUMES_LIST" ]; then
  log "[1b/5] --purge 强删数据卷(docker volume rm -f) ..." "[1b/5] --purge force-removing data volumes (docker volume rm -f) ..."
  force_remove_volumes "$VOLUMES_LIST"
else
  log "[1b/5] 保留数据卷(默认;--purge 才删)" "[1b/5] Keeping data volumes (default; --purge is required to delete them)"
fi

# 2) 业务镜像
if [ "$KEEP_IMAGES" = 0 ] && [ -n "$BUSINESS_IMAGES_LIST" ]; then
  log "[2/5] 删 omcgo/* 业务镜像 ..." "[2/5] Deleting omcgo/* business images ..."
  echo "$BUSINESS_IMAGES_LIST" | xargs -r docker rmi 2>/dev/null || warn "部分镜像删失败(可能被其他容器引用),继续" "Some images could not be deleted (possibly referenced by other containers); continuing."
fi

# 3) OMC 目录
if [ -d "$OMC_ROOT" ]; then
  if [ "$PURGE" = 1 ]; then
    log "[3/5] --purge 删整个 $OMC_ROOT ..." "[3/5] --purge deleting the entire $OMC_ROOT ..."
    rm -rf "$OMC_ROOT"
  else
    # 删前先把当前生效凭据快照到 etc/.env.saved,供下次 install.sh 继承(口令/JWT 与保留的数据卷一致)。
    if [ -f "$OMC_ROOT/current/deploy/.env" ]; then
      mkdir -p "$OMC_ROOT/etc"
      if cp -f "$OMC_ROOT/current/deploy/.env" "$OMC_ROOT/etc/.env.saved" 2>/dev/null; then
        chmod 600 "$OMC_ROOT/etc/.env.saved" ||
          die "[3/5] 凭据快照权限收紧为 600 失败，停止卸载" "[3/5] Failed to restrict the credential snapshot to 0600; uninstall stopped" 1
        log "[3/5] 已保存凭据快照 → $OMC_ROOT/etc/.env.saved(重装继承用)" "[3/5] Saved credential snapshot -> $OMC_ROOT/etc/.env.saved (used for reinstall inheritance)"
      else
        warn "[3/5] 凭据快照失败;重装前请手动核对 $OMC_ROOT/etc 与数据卷口令一致" "[3/5] Failed to save the credential snapshot; verify that $OMC_ROOT/etc credentials match the data volumes before reinstalling."
      fi
    fi
    log "[3/5] 删代码运行目录(current/releases/run/packages),保留 data/ etc/ ..." "[3/5] Deleting runtime directories (current/releases/run/packages), keeping data/ and etc/ ..."
    for sub in current releases run packages; do
      [ -e "$OMC_ROOT/$sub" ] && rm -rf "${OMC_ROOT:?}/$sub"
    done
  fi
fi

# 4) 完成
log "[4/5] 卸载完成。" "[4/5] Uninstall completed."
log "[5/5] 后续:" "[5/5] Next steps:"
if [ "$PURGE" = 1 ]; then
  log "        · 已彻底清除(数据卷 + $OMC_ROOT)。重新部署:解压交付包后 sudo bash deploy/install.sh" "        · Permanently removed (data volumes + $OMC_ROOT). To redeploy, extract the delivery package and run sudo bash deploy/install.sh"
  log "        · Docker 引擎本身:sudo bash install-docker.sh --uninstall(如需)" "        · Docker engine: sudo bash install-docker.sh --uninstall (if needed)"
  log "        · /etc/systemd/system/omcgo*.service(早期 systemd 单元):sudo systemctl disable --now omcgo*; sudo rm /etc/systemd/system/omcgo*.service" "        · /etc/systemd/system/omcgo*.service (legacy systemd units): sudo systemctl disable --now omcgo*; sudo rm /etc/systemd/system/omcgo*.service"
else
  log "        · 已保留:数据卷 + $OMC_ROOT/data + $OMC_ROOT/etc(含凭据 .env.saved)" "        · Kept: data volumes + $OMC_ROOT/data + $OMC_ROOT/etc (including .env.saved credentials)"
  log "        · 重新部署即复用现有数据:解压新交付包后 sudo bash deploy/install.sh" "        · Redeployment reuses the existing data: extract the new delivery package and run sudo bash deploy/install.sh"
  log "          (install.sh 会从 $OMC_ROOT/etc/.env.saved 继承口令/JWT,与保留的数据卷匹配)" "          (install.sh inherits credentials/JWT from $OMC_ROOT/etc/.env.saved to match the retained data volumes)"
  log "        · 如需连数据彻底清除:sudo bash $0 --purge --force" "        · To permanently remove the data as well: sudo bash $0 --purge --force"
fi
exit 0

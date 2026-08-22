#!/usr/bin/env bash
# =============================================================================
# Docker 离线安装脚本（静态二进制方式，跨发行版通用）
#
# ★ 全程离线：本脚本【不联网、不下载】任何东西，只解压【本目录内已有的】
#   docker-<版本>.tgz 安装。该 tgz 由构建侧一次性从 download.docker.com 下载好，
#   随交付包一起送入内网（见 ../../README.md / bundle/docker/README.md）。
#   内网运维侧无需任何下载。
#
# 设计来源：docs/design/deployments-release-enhancements-20260520.md §3.2
#
# 用法：
#   sudo bash install-docker.sh                          # 交互：装完后引导选加速镜像；
#                                                          /var 可用 < 15G 时还会问要不要把数据目录切到 /home
#   sudo bash install-docker.sh --mirror daocloud        # 一气呵成，装完直接配 DaoCloud 加速
#   sudo bash install-docker.sh --no-mirror              # 装完不动 daemon.json，跳过加速
#   sudo bash install-docker.sh --skip-if-installed      # 已装则只断言 DOCKER_BIP 规划后退出
#   sudo bash install-docker.sh --uninstall              # 卸载 docker 引擎(默认 dry-run)
#   sudo bash install-docker.sh --uninstall --force --keep-data
#                                                        # 真删 dockerd / 二进制 / systemd unit,但保留 /var/lib/docker
#   sudo bash install-docker.sh -h | --help              # 本帮助
#
# 参数：
#   --mirror <name>       装完后立即配置加速镜像并 restart docker。
#                         取值：official / daocloud / xuanyuan
#                         传 official 等价于"不设置镜像"，回归 docker hub 官方。
#                         其它任意 URL 请单独运行 ../setup-mirrors.sh 后手编 daemon.json
#   --no-mirror           装完不引导加速、不动 daemon.json（用户后期可单独运行
#                         ../setup-mirrors.sh）
#   --skip-if-installed   已检测到 docker 时跳过引擎安装，但仍断言 DOCKER_BIP 规划（install.sh 调用时用）
#
#   ── 卸载模式 ─────────────────────────────────────────────────────────────
#   --uninstall           进入卸载模式(默认 dry-run,仅打印将要做的动作,不实际执行)
#                         覆盖 9 步:停容器 → disable systemd → 删 install-docker.sh
#                         的 systemd unit → **apt/yum/dnf 卸载系统 docker 包**
#                         (docker.io / docker-ce / docker-compose-plugin /
#                         buildx-plugin / containerd.io 等,全自动检测) →
#                         删 /usr/local/bin docker 二进制 → 删 cli-plugins →
#                         (可选)删数据目录 → 删 /etc/docker → 删 docker 组
#   --force               关闭 dry-run(必须显式加上才会真删,且会再做一次交互确认)
#   --keep-data           不删 /var/lib/docker / /var/lib/containerd / /home/{docker,containerd}-data
#   -h | --help           本帮助
#
# 数据目录（自动检测 + 交互选择，无 CLI 选项）：
#   默认 docker → /var/lib/docker，containerd → /var/lib/containerd。
#   /var 可用空间 < 15G 时，脚本会询问是否切换到：
#     docker      → /home/docker-data        （写入 /etc/docker/daemon.json）
#     containerd  → /home/containerd-data   （写入 containerd.service ExecStart --root）
#   适用 baicells 等紧凑 /var 分区场景（避免装完一拉镜像就撑爆 /var）。
#   非交互（stdin 非 TTY，例如被 install.sh 调起）：自动按 Y 切换。
#
# 行为：
#   1. 校验 root + 本目录有且仅一个 docker-*.tgz
#   2. 检测 /var 可用空间；< 15G 时引导改用 /home/{docker,containerd}-data
#   3. tar 解压到 /usr/local/bin/{docker,dockerd,containerd,...}
#   4. 写 containerd.service（含可选 --root）+ docker.service systemd 单元
#   5. 如选切换：写 /etc/docker/daemon.json 的 data-root（python3 merge 保其它键）
#   6. systemctl daemon-reload + enable --now 两服务（开机自启）
#   7. docker version 验证
#   8. （除非 --no-mirror）引导 / 直接配置加速镜像
#   9. 提示当前用户加入 docker 组（如有 $SUDO_USER）
# =============================================================================
set -euo pipefail

# 在 cd 之前先记录脚本绝对路径，确保 -h 帮助块仍可被 sed 读到
SELF="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"

log()  { echo -e "\033[1;36m[install-docker]\033[0m $*"; }
warn() { echo -e "\033[1;33m[install-docker][警告]\033[0m $*" >&2; }
die()  { echo -e "\033[1;31m[install-docker][错误]\033[0m $*" >&2; exit 1; }

SCRIPT_DIR="$(dirname "$SELF")"
if [ -f "$SCRIPT_DIR/docker-network-lib.sh" ]; then
  . "$SCRIPT_DIR/docker-network-lib.sh"
elif [ -f "$SCRIPT_DIR/../../../docker/docker-network-lib.sh" ]; then
  . "$SCRIPT_DIR/../../../docker/docker-network-lib.sh"
else
  die "缺少 docker-network-lib.sh；无法按 DOCKER_BIP 规划 Docker 网段"
fi

# ── 参数解析 ────────────────────────────────────────────────────────────
MIRROR=""
NO_MIRROR=0
SKIP_IF_INSTALLED=0
UNINSTALL=0
FORCE=0
KEEP_DATA=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mirror)              MIRROR="$2"; shift 2 ;;
    --no-mirror)           NO_MIRROR=1; shift ;;
    --skip-if-installed)   SKIP_IF_INSTALLED=1; shift ;;
    --uninstall)           UNINSTALL=1; shift ;;
    --force)               FORCE=1; shift ;;
    --keep-data)           KEEP_DATA=1; shift ;;
    -h|--help)             awk 'NR>=3 && /^# ====/ {exit} NR>=3 {print}' "$SELF"; exit 0 ;;
    *)                     die "未知参数：$1（-h 查看用法）" ;;
  esac
done

# ── Docker 网段规划 ───────────────────────────────────────────────────────
if [ "$UNINSTALL" = 0 ]; then
  docker_network_require_python3 ||
    die "缺少 python3，无法计算和校验 Docker 网段，不能继续安装 Docker" "python3 is required to calculate and validate Docker networks; Docker installation cannot continue"
  if ! docker_network_resolve_bip; then
    die "无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）" \
      "Unable to resolve the Docker network plan (neither the default nor a custom DOCKER_BIP is usable)"
  fi
  docker_network_plan || die "DOCKER_BIP 无效或无法派生 Docker 网段：$DOCKER_BIP"
  log "Docker 网段来源：${DOCKER_BIP_SOURCE:-unknown}，规划输入：$DOCKER_BIP"
fi

cleanup_empty_unplanned_docker_networks() {
  local removed_networks network_name subnet
  removed_networks="$(docker_network_cleanup_unplanned_networks || true)"
  while IFS=$'\t' read -r network_name subnet; do
    [ -n "$network_name" ] || continue
    log "清理无容器的规划外 Docker 网络：${network_name} (${subnet})"
  done <<< "$removed_networks"
}

assert_no_unplanned_docker_networks() {
  command -v docker >/dev/null 2>&1 || return 0
  docker info >/dev/null 2>&1 || return 0

  cleanup_empty_unplanned_docker_networks

  local unplanned_networks
  unplanned_networks="$(docker_network_unplanned_networks || true)"
  [ -z "$unplanned_networks" ] ||
    die "检测到未纳入 DOCKER_BIP 规划的 Docker 网络：${unplanned_networks}；请先停止并删除，或重新规划 DOCKER_BIP" \
      "Docker networks outside the DOCKER_BIP plan were detected: ${unplanned_networks}; stop and remove them, or re-plan DOCKER_BIP"
}

# ── assert_docker_network：幂等断言 DOCKER_BIP 网段(bip)+ 自动池(#155)────────────
# docker0 的 bip 由 /etc/docker/daemon.json 控制(compose 管不到 docker0)。历史 bug:bip 只
# 在"全新装 dockerd"分支写,docker 已装即整段跳过、--skip-if-installed 更直接 exit 0 →
# 一旦 daemon.json 被引擎升级覆盖 / 清空 / 手改丢失,docker0 会回落 Docker 默认网段并产生冲突,
# 且后续部署不修复、静默回退。本函数让每次部署都断言 bip:已正确→不动不重启;漂移→写
# daemon.json(python3 merge 保其它键)+ (docker 在跑时)重启使 docker0 生效。
assert_docker_network() {
  local DAEMON_JSON="${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}" cur_bip="" cur_pool_base="" cur_pool_size="" network_state=""
  docker_network_plan || die "DOCKER_BIP 无效或无法派生 Docker 网段：$DOCKER_BIP"
  mkdir -p /etc/docker
  if [ -f "$DAEMON_JSON" ] && [ -s "$DAEMON_JSON" ]; then
    network_state="$(python3 - "$DAEMON_JSON" <<'PYEOF'
import json
import sys

try:
    with open(sys.argv[1], encoding="utf-8") as handle:
        data = json.load(handle)
except (OSError, json.JSONDecodeError):
    data = {}

pools = data.get("default-address-pools")
pool = pools[0] if isinstance(pools, list) and pools and isinstance(pools[0], dict) else {}
print(f"{data.get('bip', '')}\t{pool.get('base', '')}\t{pool.get('size', '')}")
PYEOF
    )" || network_state=""
    IFS=$'\t' read -r cur_bip cur_pool_base cur_pool_size <<< "$network_state"
  fi
  if [ "$cur_bip" = "$DOCKER_BIP" ] &&
     [ "$cur_pool_base" = "$DOCKER_ADDR_POOL_BASE" ] &&
     [ "$cur_pool_size" = "$DOCKER_ADDR_POOL_SIZE" ]; then
    log "Docker 网段策略已生效：bip=${DOCKER_BIP}，Compose=${DOCKER_COMPOSE_SUBNET}，地址池=${DOCKER_ADDR_POOL_BASE}/${DOCKER_ADDR_POOL_SIZE}，无需改动"
    return 0
  fi
  [ -n "$cur_bip" ] && log "Docker 网段策略漂移：bip='${cur_bip}'/pool='${cur_pool_base}/${cur_pool_size}'，按 DOCKER_BIP 重新断言"
  [ -f "$DAEMON_JSON" ] && cp -a "$DAEMON_JSON" "$DAEMON_JSON.bak.$(date +%Y%m%d%H%M%S)" || true
  if ! python3 - "$DAEMON_JSON" "$DOCKER_BIP" "$DOCKER_ADDR_POOL_BASE" "$DOCKER_ADDR_POOL_SIZE" <<'PYEOF'
import json, os, sys
p, bip, pool_base, pool_size = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
data = {}
if os.path.exists(p) and os.path.getsize(p) > 0:
    try: data = json.load(open(p))
    except json.JSONDecodeError: data = {}
data['bip'] = bip
data['default-address-pools'] = [{'base': pool_base, 'size': pool_size}]
with open(p, 'w') as f:
    json.dump(data, f, indent=2, ensure_ascii=False); f.write('\n')
PYEOF
  then
    die "写 ${DAEMON_JSON} 网段失败；禁止在 Docker 网段未按 DOCKER_BIP 规划时继续"
  fi
  log "已写入 ${DAEMON_JSON}：bip=${DOCKER_BIP}, default-address-pools=${DOCKER_ADDR_POOL_BASE}(/${DOCKER_ADDR_POOL_SIZE})"
  # docker 在运行 → 重启使 docker0 新 bip 生效(全新装时 docker 尚未起,交给后面的
  # systemctl enable --now)。Compose 网段由同一个 DOCKER_BIP 规划派生；仅网桥实际漂移时才重启,
  # 稳态零打扰(业务容器会随 docker 重启而重启)。
  if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet docker 2>/dev/null; then
    log "bip 变更且 docker 在运行,重启 docker 使 docker0 生效(业务容器会随之重启)..."
    systemctl daemon-reload
    systemctl restart docker || warn "docker 重启失败,请手动 systemctl restart docker 使 bip 生效"
  fi
}

cd "$(dirname "$SELF")"

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

# ── 卸载模式 — 在所有"已装跳过"逻辑之前分流 ──────────────────────────
if [ "$UNINSTALL" = 1 ]; then
  echo
  log "Docker 卸载模式"

  DRY="[dry-run]"
  REAL=0
  if [ "$FORCE" = 1 ]; then
    REAL=1
    DRY=""
  fi

  # 扫描当前状态
  DOCKER_VER="(未装)"
  command -v docker >/dev/null 2>&1 && DOCKER_VER="$(docker --version 2>/dev/null || echo unknown)"
  RUNNING_CONTAINERS="$(docker ps -q 2>/dev/null | wc -l | tr -d ' ' 2>/dev/null || echo 0)"

  # daemon.json 里 data-root 真实路径(可能被用户切到 /home)
  DOCKER_ROOT_DEFAULT="/var/lib/docker"
  CONTAINERD_ROOT_DEFAULT="/var/lib/containerd"
  ACTUAL_DOCKER_ROOT="$DOCKER_ROOT_DEFAULT"
  if [ -f /etc/docker/daemon.json ] && command -v python3 >/dev/null 2>&1; then
    DR="$(python3 -c "import json; d=json.load(open('/etc/docker/daemon.json')); print(d.get('data-root',''))" 2>/dev/null || true)"
    [ -n "$DR" ] && ACTUAL_DOCKER_ROOT="$DR"
  fi
  # containerd root 在 systemd unit 里(ExecStart ... --root /home/containerd-data)
  ACTUAL_CONTAINERD_ROOT="$CONTAINERD_ROOT_DEFAULT"
  if [ -f /etc/systemd/system/containerd.service ]; then
    CR="$(grep -oE -- '--root[ =]+\S+' /etc/systemd/system/containerd.service | head -1 | awk '{print $2}')"
    [ -n "$CR" ] && ACTUAL_CONTAINERD_ROOT="$CR"
  fi

  DOCKER_DATA_SIZE=""
  [ -d "$ACTUAL_DOCKER_ROOT" ] && DOCKER_DATA_SIZE="$(du -sh "$ACTUAL_DOCKER_ROOT" 2>/dev/null | awk '{print $1}')"
  CONTAINERD_DATA_SIZE=""
  [ -d "$ACTUAL_CONTAINERD_ROOT" ] && CONTAINERD_DATA_SIZE="$(du -sh "$ACTUAL_CONTAINERD_ROOT" 2>/dev/null | awk '{print $1}')"

  echo
  log "${DRY}卸载计划:"
  log "${DRY}  · 当前 Docker:$DOCKER_VER"
  log "${DRY}  · 运行中容器数:$RUNNING_CONTAINERS"
  log "${DRY}  · docker data-root:$ACTUAL_DOCKER_ROOT $([ -n "$DOCKER_DATA_SIZE" ] && echo "($DOCKER_DATA_SIZE)")"
  log "${DRY}  · containerd root:$ACTUAL_CONTAINERD_ROOT $([ -n "$CONTAINERD_DATA_SIZE" ] && echo "($CONTAINERD_DATA_SIZE)")"
  echo

  # 系统包管理器检测 + 已装 docker 包扫描
  SYS_PKG_MGR=""
  SYS_DOCKER_PKGS=""
  # Debian/Ubuntu 系包列表(覆盖发行版仓库 + Docker 官方仓库)
  DPKG_CANDIDATES="docker.io docker-doc docker-compose podman-docker docker-ce docker-ce-cli docker-ce-rootless-extras docker-compose-plugin docker-buildx-plugin docker-scan-plugin containerd containerd.io runc"
  # RHEL/CentOS/Rocky/openEuler 系包列表
  RPM_CANDIDATES="docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine docker-ce docker-ce-cli docker-ce-rootless-extras docker-compose-plugin docker-buildx-plugin docker-compose containerd containerd.io runc"
  if command -v dpkg >/dev/null 2>&1; then
    SYS_PKG_MGR="apt"
    for p in $DPKG_CANDIDATES; do
      dpkg -s "$p" >/dev/null 2>&1 && SYS_DOCKER_PKGS="$SYS_DOCKER_PKGS $p"
    done
  elif command -v rpm >/dev/null 2>&1; then
    if command -v dnf >/dev/null 2>&1; then SYS_PKG_MGR="dnf"; else SYS_PKG_MGR="yum"; fi
    for p in $RPM_CANDIDATES; do
      rpm -q "$p" >/dev/null 2>&1 && SYS_DOCKER_PKGS="$SYS_DOCKER_PKGS $p"
    done
  fi
  SYS_DOCKER_PKGS="$(echo "$SYS_DOCKER_PKGS" | xargs)"   # trim

  if [ "$RUNNING_CONTAINERS" -gt 0 ] 2>/dev/null; then
    log "${DRY}1) 将停止 + 删除 $RUNNING_CONTAINERS 个运行中容器(包含可能的 OMC 业务容器)"
  else
    log "${DRY}1) 无运行中容器"
  fi
  log "${DRY}2) systemctl disable --now docker containerd"
  log "${DRY}3) 删 systemd unit:/etc/systemd/system/docker.service / containerd.service(install-docker.sh 写的)"
  if [ -n "$SYS_DOCKER_PKGS" ]; then
    log "${DRY}4) 卸载系统包(${SYS_PKG_MGR}):$SYS_DOCKER_PKGS"
    log "${DRY}     (含 apt 装的 docker.io / docker-compose V1 / cli-plugins 等;卸载会自动清掉系统 systemd unit)"
  else
    log "${DRY}4) 未检测到系统包管理器装的 docker(${SYS_PKG_MGR:-未知}),跳过"
  fi
  log "${DRY}5) 删 /usr/local/bin/{docker,dockerd,containerd,runc,docker-init,docker-proxy,ctr,containerd-shim*}(install-docker.sh 装的)"
  log "${DRY}6) 删 /usr/local/lib/docker/cli-plugins/(docker-compose V2 + buildx)"
  if [ "$KEEP_DATA" = 1 ]; then
    log "${DRY}7) --keep-data 保留 docker / containerd 数据目录(可日后重装恢复镜像 / 容器)"
  else
    log "${DRY}7) 将删:"
    log "${DRY}     - $ACTUAL_DOCKER_ROOT($DOCKER_DATA_SIZE)"
    log "${DRY}     - $ACTUAL_CONTAINERD_ROOT($CONTAINERD_DATA_SIZE)"
    [ "$ACTUAL_DOCKER_ROOT"     != "/var/lib/docker"     ] && log "${DRY}     - /var/lib/docker(若残留)"
    [ "$ACTUAL_CONTAINERD_ROOT" != "/var/lib/containerd" ] && log "${DRY}     - /var/lib/containerd(若残留)"
  fi
  log "${DRY}8) 删 /etc/docker(daemon.json + certs.d/)"
  log "${DRY}9) 删 docker 用户组"
  echo
  log "${DRY}注意:本脚本【不删 /opt/omc 等 OMC 业务数据】,如需一并清理:"
  log "${DRY}        先跑 sudo bash uninstall.sh --force,再跑本脚本"
  echo

  if [ "$REAL" = 0 ]; then
    log "[dry-run] 未实际执行任何动作。要真删,加 --force 再跑一次:"
    log "[dry-run]   sudo bash $0 --uninstall --force $([ "$KEEP_DATA" = 1 ] && echo "--keep-data")"
    exit 0
  fi

  # 真删 — 二次交互确认
  if [ -t 0 ]; then
    echo
    warn "以上操作【不可逆】,确认后立即执行。"
    read -rp "确认卸载 Docker 引擎?(默认 N) [y/N] " yn
    case "${yn:-N}" in
      [Yy]*) ;;
      *)     log "用户取消,未执行任何操作。"; exit 0 ;;
    esac
  else
    log "(非交互模式 + --force:跳过二次确认,直接执行)"
  fi

  cd /tmp

  # 1) 停所有容器
  log "[1/9] 停止所有运行中容器 ..."
  CONTAINERS="$(docker ps -q 2>/dev/null || true)"
  [ -n "$CONTAINERS" ] && echo "$CONTAINERS" | xargs -r docker stop >/dev/null 2>&1 || true

  # 2) systemctl
  log "[2/9] systemctl disable --now docker containerd ..."
  systemctl disable --now docker      >/dev/null 2>&1 || true
  systemctl disable --now containerd  >/dev/null 2>&1 || true

  # 3) systemd unit(install-docker.sh 写入的)
  log "[3/9] 删 systemd unit(install-docker.sh 写入的) ..."
  rm -f /etc/systemd/system/docker.service /etc/systemd/system/containerd.service
  systemctl daemon-reload >/dev/null 2>&1 || true

  # 4) 系统包管理器卸载(apt / yum / dnf 装的 docker.io / docker-ce / compose-plugin 等)
  if [ -n "$SYS_DOCKER_PKGS" ]; then
    log "[4/9] 用 $SYS_PKG_MGR 卸载系统 docker 包:$SYS_DOCKER_PKGS ..."
    case "$SYS_PKG_MGR" in
      apt)
        # shellcheck disable=SC2086
        DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y $SYS_DOCKER_PKGS >/dev/null 2>&1 || \
          warn "apt remove 部分失败,可能因依赖锁定;手动 'apt purge $SYS_DOCKER_PKGS' 重试"
        apt-get autoremove --purge -y >/dev/null 2>&1 || true
        ;;
      yum|dnf)
        # shellcheck disable=SC2086
        "$SYS_PKG_MGR" remove -y $SYS_DOCKER_PKGS >/dev/null 2>&1 || \
          warn "$SYS_PKG_MGR remove 部分失败,可能因依赖锁定;手动重试"
        ;;
    esac
  else
    log "[4/9] 系统包管理器未检测到 docker 包,跳过"
  fi

  # 5) /usr/local/bin/(install-docker.sh 装的二进制;apt remove 不会动这里)
  log "[5/9] 删 /usr/local/bin/ 下 docker 二进制(install-docker.sh 装的) ..."
  rm -f /usr/local/bin/docker          /usr/local/bin/dockerd \
        /usr/local/bin/containerd      /usr/local/bin/containerd-shim \
        /usr/local/bin/containerd-shim-runc-v2 \
        /usr/local/bin/runc            /usr/local/bin/docker-init \
        /usr/local/bin/docker-proxy    /usr/local/bin/ctr

  # 6) cli-plugins(install-docker.sh 装的;apt 装的 plugin 已被步骤 4 清掉)
  log "[6/9] 删 /usr/local/lib/docker/cli-plugins/ ..."
  rm -rf /usr/local/lib/docker/cli-plugins
  rmdir /usr/local/lib/docker 2>/dev/null || true

  # 7) 数据目录
  if [ "$KEEP_DATA" = 0 ]; then
    log "[7/9] 删数据目录 $ACTUAL_DOCKER_ROOT / $ACTUAL_CONTAINERD_ROOT ..."
    rm -rf "$ACTUAL_DOCKER_ROOT"      "$ACTUAL_CONTAINERD_ROOT"
    # 兜底:如果默认路径也存在(用户中途切过),一并清
    [ "$ACTUAL_DOCKER_ROOT"     != "/var/lib/docker"     ] && rm -rf /var/lib/docker     2>/dev/null || true
    [ "$ACTUAL_CONTAINERD_ROOT" != "/var/lib/containerd" ] && rm -rf /var/lib/containerd 2>/dev/null || true
  else
    log "[7/9] --keep-data 保留数据目录"
  fi

  # 8) /etc/docker
  log "[8/9] 删 /etc/docker ..."
  rm -rf /etc/docker

  # 9) docker 组
  log "[9/9] 删 docker 用户组 ..."
  groupdel docker >/dev/null 2>&1 || true

  echo
  log "Docker 卸载完成。"
  log "残留(若需要彻底清):"
  log "  · OMC 业务数据 /opt/omc/(若用 uninstall.sh 保留过(默认保留数据))"
  log "  · /home/{docker,containerd}-data 自定义数据路径(若用户改过)"
  exit 0
fi

# ── 已装则按需返回 ──────────────────────────────────────────────────────
# 2026-05-29 改：docker 已装的环境(尤其 apt 装 docker.io + docker-compose 老仓
# 库),Compose 是 Python V1,无法解析 compose v3.x 语法。这里把"已装跳过"拆为
# 两档:
#   - --skip-if-installed (install.sh 内部探测用):docker 已装则只执行网段断言后 exit 0
#   - 默认(运维直跑):跳过 dockerd/containerd 二进制 + systemd unit 安装,
#                     但**继续**走到下面的 cli-plugins 安装(docker-compose V2
#                     plugin + docker-buildx),保证 `docker compose` 可用。
SKIP_DOCKERD=0
if command -v docker >/dev/null 2>&1; then
  if [ "$SKIP_IF_INSTALLED" = 1 ]; then
    log "已检测到 Docker：$(docker --version)，跳过安装；仍断言 docker0 网段（--skip-if-installed）"
    assert_docker_network   # #155：即便不装 docker,也每次部署断言 docker0 bip
    assert_no_unplanned_docker_networks
    exit 0
  fi
  log "已检测到 Docker：$(docker --version)，跳过 dockerd/containerd 二进制与 systemd unit 安装"
  log "继续补装 docker compose V2 / docker buildx 插件,确保 \`docker compose\` 可用"
  SKIP_DOCKERD=1
fi

# ── 离线包定位（必须本目录有且仅一个 docker-*.tgz）────────────────────────
# SKIP_DOCKERD=1 时不解 dockerd/containerd 二进制,但仍需进到下面安装 cli-plugins,
# 所以离线包定位也跳过,直接走到 plugin 安装段。
if [ "$SKIP_DOCKERD" = 1 ]; then
  TGZ=""
else
TGZ_COUNT="$(ls docker-*.tgz 2>/dev/null | wc -l | tr -d ' ')"
if [ "$TGZ_COUNT" = 0 ]; then
  echo "错误：本目录未找到 docker-*.tgz。"
  echo "      Docker 静态二进制包应由构建侧预先下载并打入交付包 docker/ 目录，"
  echo "      本脚本不会联网下载。请联系交付方补齐交付包，"
  echo "      或在目标机改用已安装好的 Docker（≥ 20.10）后跳过本步。"
  exit 1
elif [ "$TGZ_COUNT" -gt 1 ]; then
  echo "错误：本目录存在多个 docker-*.tgz，无法确定应安装哪个版本："
  ls docker-*.tgz
  echo "      请只保留一个目标版本的包后重试。"
  exit 1
fi
TGZ="$(ls docker-*.tgz)"
log "使用 Docker 安装包：$TGZ"
fi

if [ "$SKIP_DOCKERD" = 0 ]; then

# ── 数据目录选择（/var 紧时引导切到 /home，避免装完撑爆）─────────────────
# 默认 docker → /var/lib/docker，containerd → /var/lib/containerd（不动）。
# /var 可用 < 15G 时打 warn 并交互询问，确认后写 /home/{docker,containerd}-data。
# 详见 design doc deployments-release-enhancements-20260520.md "数据目录" 段。
DATA_ROOT=""           # 空 = 走 docker 默认 /var/lib/docker
CONTAINERD_ROOT=""     # 空 = 走 containerd 默认 /var/lib/containerd
THRESHOLD_GB=15

avail_gb() {
  local kb
  kb=$(df -k "$1" 2>/dev/null | tail -1 | awk '{print $4+0}')
  [ -z "$kb" ] && kb=0
  echo $((kb / 1024 / 1024))
}

VAR_AVAIL_GB=$(avail_gb /var)
if [ "$VAR_AVAIL_GB" -lt "$THRESHOLD_GB" ]; then
  HOME_AVAIL_GB=$(avail_gb /home)
  echo
  warn "/var 可用空间 ${VAR_AVAIL_GB}G < ${THRESHOLD_GB}G —— docker 数据放 /var 容易撑爆"
  echo
  echo "  建议改用 /home（/home 可用 ${HOME_AVAIL_GB}G）："
  echo "    docker 数据    → /home/docker-data"
  echo "    containerd 数据 → /home/containerd-data"
  if [ "$HOME_AVAIL_GB" -lt "$THRESHOLD_GB" ]; then
    echo
    warn "/home 可用 ${HOME_AVAIL_GB}G 也 < ${THRESHOLD_GB}G —— 两个分区都紧张，仍可切但风险类似"
  fi
  echo

  if [ -t 0 ]; then
    read -rp "切换到 /home？[Y/n] " yn
  else
    yn=""
    log "（非交互模式：stdin 非 TTY → 自动按 Y 处理）"
  fi

  case "${yn:-Y}" in
    [Yy]*|"")
      DATA_ROOT="/home/docker-data"
      CONTAINERD_ROOT="/home/containerd-data"
      log "已切换：docker data-root → ${DATA_ROOT}"
      log "         containerd root  → ${CONTAINERD_ROOT}"
      ;;
    *)
      warn "继续使用默认 /var/lib，docker 装完后磁盘可能很快撑爆"
      ;;
  esac
fi

# 如选切换：建目录并设权限（docker 数据目录权限通常 0711）
if [ -n "$DATA_ROOT" ];       then mkdir -p "$DATA_ROOT";       chmod 0711 "$DATA_ROOT";       fi
if [ -n "$CONTAINERD_ROOT" ]; then mkdir -p "$CONTAINERD_ROOT"; chmod 0711 "$CONTAINERD_ROOT"; fi

# ── 解压二进制 ──────────────────────────────────────────────────────────
log "解压 $TGZ ..."
tar xzf "$TGZ"
install -m 0755 docker/* /usr/local/bin/
rm -rf docker/

fi  # ← end "if [ \"$SKIP_DOCKERD\" = 0 ]" 包住数据目录选择 + 解压二进制

# ── docker compose / buildx 插件 ──────────────────────────────────────
# 2026-05-29:无论 SKIP_DOCKERD 与否,**永远安装** cli-plugins。已装 docker 的
# 环境最常见的坑就是系统的 docker-compose 是老 V1 Python(`apt install
# docker-compose`),不识别 compose v3 写法,部署 OMC 时报
# "Unsupported config option for services/networks/volumes"。
# 在 /usr/local/lib/docker/cli-plugins/ 下放 V2 binary 后,`docker compose`
# (带空格)即可走 V2,install.sh 的检测会优先用它。
if [ -f docker-compose ]; then
  install -d /usr/local/lib/docker/cli-plugins
  install -m 0755 docker-compose /usr/local/lib/docker/cli-plugins/docker-compose
  log "已安装 docker compose V2 插件:$(docker compose version 2>&1 | head -1 || echo '稍后启动 dockerd 后验证')"
fi

# docker buildx 插件 —— Docker 23+ 在 BuildKit 开启时必需
if [ -f docker-buildx ]; then
  install -d /usr/local/lib/docker/cli-plugins
  install -m 0755 docker-buildx /usr/local/lib/docker/cli-plugins/docker-buildx
fi

if [ "$SKIP_DOCKERD" = 0 ]; then

# ── systemd 单元：containerd ────────────────────────────────────────────
# ExecStart 按需附加 --root <DIR>（来自上面的数据目录选择）
CONTAINERD_EXEC="/usr/local/bin/containerd"
[ -n "$CONTAINERD_ROOT" ] && CONTAINERD_EXEC="${CONTAINERD_EXEC} --root ${CONTAINERD_ROOT}"

cat > /etc/systemd/system/containerd.service <<EOF
[Unit]
Description=containerd container runtime
After=network.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=${CONTAINERD_EXEC}
Restart=always
RestartSec=5
Delegate=yes
KillMode=process
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

# ── systemd 单元：dockerd ───────────────────────────────────────────────
cat > /etc/systemd/system/docker.service <<'EOF'
[Unit]
Description=Docker Application Container Engine
After=network-online.target containerd.service
Wants=network-online.target
Requires=containerd.service

[Service]
Type=notify
ExecStart=/usr/local/bin/dockerd --containerd=/run/containerd/containerd.sock
Restart=always
RestartSec=5
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
Delegate=yes
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

# ── docker data-root 写入 daemon.json（仅当用户选了切换）─────────────────
# 用 python3 merge 保留 daemon.json 其它键（后续 setup-mirrors.sh 写
# registry-mirrors 不会被覆盖）。
if [ -n "$DATA_ROOT" ]; then
  DAEMON_JSON=/etc/docker/daemon.json
  mkdir -p /etc/docker
  [ -f "$DAEMON_JSON" ] && cp -a "$DAEMON_JSON" "$DAEMON_JSON.bak.$(date +%Y%m%d%H%M%S)"
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$DAEMON_JSON" "$DATA_ROOT" <<'PYEOF'
import json, os, sys
p, dr = sys.argv[1], sys.argv[2]
data = {}
if os.path.exists(p) and os.path.getsize(p) > 0:
    try:
        data = json.load(open(p))
    except json.JSONDecodeError:
        data = {}
data['data-root'] = dr
with open(p, 'w') as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
    f.write('\n')
PYEOF
  else
    if [ -s "$DAEMON_JSON" ]; then
      warn "未装 python3 且 daemon.json 已有内容；data-root 未写入。
            请手动在 $DAEMON_JSON 加： \"data-root\": \"$DATA_ROOT\""
    else
      printf '{\n  "data-root": "%s"\n}\n' "$DATA_ROOT" > "$DAEMON_JSON"
    fi
  fi
  log "已写入 ${DAEMON_JSON}：data-root = ${DATA_ROOT}"
fi

# ── docker0 网段断言(bip + 自动池)── 抽到 assert_docker_network(#155)。全新装路径:
# 此刻 docker 尚未 enable --now,函数只写 daemon.json、不重启,随后 systemctl 启动即带 bip。
assert_docker_network

# ── 启用并启动（开机自启）────────────────────────────────────────────────
getent group docker >/dev/null 2>&1 || groupadd docker

systemctl daemon-reload
systemctl enable --now containerd
systemctl enable --now docker

echo
docker version
assert_no_unplanned_docker_networks
log "Docker 安装完成。"

fi  # ← end "if [ \"$SKIP_DOCKERD\" = 0 ]" 包住 systemd unit + daemon.json + systemctl 启动

# docker 已装(SKIP_DOCKERD=1,运维直跑未带 --skip-if-installed):上面全新装段被整段跳过,
# 这里补断言 docker0 网段,确保 bip 漂移也能被每次部署修复(#155)。
if [ "$SKIP_DOCKERD" = 1 ]; then
  assert_docker_network
  assert_no_unplanned_docker_networks
fi

# ── 加速镜像配置 ────────────────────────────────────────────────────────
if [ "$NO_MIRROR" = 1 ]; then
  log "--no-mirror：跳过加速配置。后期可单独运行 bash ../setup-mirrors.sh"
elif [ -n "$MIRROR" ]; then
  log "按 --mirror=$MIRROR 配置 Docker 加速镜像 ..."
  bash "$(dirname "$0")/../setup-mirrors.sh" --docker "$MIRROR"
else
  echo
  echo "─────────────────────────────────────────────────"
  echo " Docker 已装好。要不要配置【加速】（Docker / npm / Golang 三合一）？"
  echo "  · 推荐配置（内网拉镜像 / npm install / go install 都会快很多）"
  echo "  · 跳过也行（后期可单独运行 ../setup-mirrors.sh）"
  echo "─────────────────────────────────────────────────"
  read -rp "现在配置加速？[Y/n] " yn
  case "${yn:-Y}" in
    [Yy]*|"") bash "$(dirname "$0")/../setup-mirrors.sh" ;;
    *)         log "跳过加速配置" ;;
  esac
fi

# ── 提示加入 docker 组 ──────────────────────────────────────────────────
if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
  if ! id -nG "$SUDO_USER" 2>/dev/null | tr ' ' '\n' | grep -qx docker; then
    echo
    log "提示：把当前用户加入 docker 组可免 sudo 操作 docker："
    echo "  sudo usermod -aG docker $SUDO_USER"
    echo "  # 之后重新登录或：newgrp docker"
  fi
fi

echo
DATA_ROOT_DISPLAY="${DATA_ROOT:-/var/lib/docker (默认)}"
CONTAINERD_ROOT_DISPLAY="${CONTAINERD_ROOT:-/var/lib/containerd (默认)}"
log "数据目录："
log "  · docker data-root  ：${DATA_ROOT_DISPLAY}"
log "  · containerd root   ：${CONTAINERD_ROOT_DISPLAY}"
log "  · 校验：docker info | grep -E 'Docker Root Dir|Containerd'"
echo
log "全部完成。下一步建议："
log "  · 一键部署 OMC：    sudo bash ../deploy/install.sh"
log "  · 重新选择加速器：  sudo bash ../setup-mirrors.sh"
log "  · 看当前加速配置：  bash ../setup-mirrors.sh --show"

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
#   sudo bash install-docker.sh --skip-if-installed      # 已装则静默 0 退出（脚本里调）
#   sudo bash install-docker.sh -h | --help              # 本帮助
#
# 参数：
#   --mirror <name>       装完后立即配置加速镜像并 restart docker。
#                         取值：official / daocloud / xuanyuan
#                         传 official 等价于"不设置镜像"，回归 docker hub 官方。
#                         其它任意 URL 请单独运行 ../setup-mirrors.sh 后手编 daemon.json
#   --no-mirror           装完不引导加速、不动 daemon.json（用户后期可单独运行
#                         ../setup-mirrors.sh）
#   --skip-if-installed   已检测到 docker 时静默 0 退出（deploy.sh 调用时用）
#   -h | --help           本帮助
#
# 数据目录（自动检测 + 交互选择，无 CLI 选项）：
#   默认 docker → /var/lib/docker，containerd → /var/lib/containerd。
#   /var 可用空间 < 15G 时，脚本会询问是否切换到：
#     docker      → /home/docker-data        （写入 /etc/docker/daemon.json）
#     containerd  → /home/containerd-data   （写入 containerd.service ExecStart --root）
#   适用 baicells 等紧凑 /var 分区场景（避免装完一拉镜像就撑爆 /var）。
#   非交互（stdin 非 TTY，例如被 deploy.sh 调起）：自动按 Y 切换。
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

# ── 参数解析 ────────────────────────────────────────────────────────────
MIRROR=""
NO_MIRROR=0
SKIP_IF_INSTALLED=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mirror)              MIRROR="$2"; shift 2 ;;
    --no-mirror)           NO_MIRROR=1; shift ;;
    --skip-if-installed)   SKIP_IF_INSTALLED=1; shift ;;
    -h|--help)             awk 'NR>=3 && /^# ====/ {exit} NR>=3 {print}' "$SELF"; exit 0 ;;
    *)                     die "未知参数：$1（-h 查看用法）" ;;
  esac
done

cd "$(dirname "$SELF")"

[ "$(id -u)" = 0 ] || die "请以 root 执行（sudo bash $0 ...）"

# ── 已装则按需返回 ──────────────────────────────────────────────────────
if command -v docker >/dev/null 2>&1; then
  if [ "$SKIP_IF_INSTALLED" = 1 ]; then
    log "已检测到 Docker：$(docker --version)，跳过（--skip-if-installed）"
    exit 0
  fi
  log "已检测到 Docker：$(docker --version)，跳过安装"
  log "如需配置加速：bash ../setup-mirrors.sh                    # Docker / npm / Golang 三合一"
  exit 0
fi

# ── 离线包定位（必须本目录有且仅一个 docker-*.tgz）────────────────────────
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

# docker compose 插件（若交付包内提供）
if [ -f docker-compose ]; then
  install -d /usr/local/lib/docker/cli-plugins
  install -m 0755 docker-compose /usr/local/lib/docker/cli-plugins/docker-compose
fi

# docker buildx 插件（若交付包内提供）——Docker 23+ 在 BuildKit 开启时必需
if [ -f docker-buildx ]; then
  install -d /usr/local/lib/docker/cli-plugins
  install -m 0755 docker-buildx /usr/local/lib/docker/cli-plugins/docker-buildx
fi

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

# ── 启用并启动（开机自启）────────────────────────────────────────────────
getent group docker >/dev/null 2>&1 || groupadd docker

systemctl daemon-reload
systemctl enable --now containerd
systemctl enable --now docker

echo
docker version
log "Docker 安装完成。"

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
log "  · 一键部署 OMC：    sudo bash ../deploy/deploy.sh"
log "  · 重新选择加速器：  sudo bash ../setup-mirrors.sh"
log "  · 看当前加速配置：  bash ../setup-mirrors.sh --show"

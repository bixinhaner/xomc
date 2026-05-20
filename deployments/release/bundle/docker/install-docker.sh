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
#   sudo bash install-docker.sh                          # 交互：装完后引导选加速镜像
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
# 行为：
#   1. 校验 root + 本目录有且仅一个 docker-*.tgz
#   2. tar 解压到 /usr/local/bin/{docker,dockerd,containerd,...}
#   3. 写 containerd.service + docker.service systemd 单元
#   4. systemctl daemon-reload + enable --now 两服务（开机自启）
#   5. docker version 验证
#   6. （除非 --no-mirror）引导 / 直接配置加速镜像
#   7. 提示当前用户加入 docker 组（如有 $SUDO_USER）
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
    -h|--help)             sed -n '3,38p' "$SELF"; exit 0 ;;
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

# ── systemd 单元：containerd ────────────────────────────────────────────
cat > /etc/systemd/system/containerd.service <<'EOF'
[Unit]
Description=containerd container runtime
After=network.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/containerd
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
log "全部完成。下一步建议："
log "  · 一键部署 OMC：    sudo bash ../deploy/deploy.sh"
log "  · 重新选择加速器：  sudo bash ../setup-mirrors.sh"
log "  · 看当前加速配置：  bash ../setup-mirrors.sh --show"

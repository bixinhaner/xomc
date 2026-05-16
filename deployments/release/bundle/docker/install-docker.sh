#!/usr/bin/env bash
# =============================================================================
# Docker 离线安装脚本（静态二进制方式，跨发行版通用）
#
# ★ 全程离线：本脚本【不联网、不下载】任何东西，只解压【本目录内已有的】
#   docker-<版本>.tgz 安装。该 tgz 由构建侧一次性从 download.docker.com 下载好，
#   随交付包一起送入内网（见 ../../README.md / bundle/docker/README.md）。
#   内网运维侧无需任何下载。
#
# 在交付包的 docker/ 目录下以 root 执行：
#   cd /opt/omc/current/docker && bash install-docker.sh
# =============================================================================
set -euo pipefail

cd "$(dirname "$0")"

[ "$(id -u)" = 0 ] || { echo "请以 root 执行"; exit 1; }

if command -v docker >/dev/null 2>&1; then
  echo "检测到已安装 Docker：$(docker --version)，跳过安装。"
  exit 0
fi

# 安装的 Docker 版本【不是自动下载、不是 latest】，而是构建侧预先选定、
# 放入本目录的那个 docker-*.tgz。要求目录内有且仅有一个，版本才确定。
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
echo "使用 Docker 安装包：$TGZ"

echo "解压 $TGZ ..."
tar xzf "$TGZ"                         # 解出 docker/ 目录
install -m 0755 docker/* /usr/local/bin/
rm -rf docker/

# docker compose 插件（若交付包内提供）
if [ -f docker-compose ]; then
  install -d /usr/local/lib/docker/cli-plugins
  install -m 0755 docker-compose /usr/local/lib/docker/cli-plugins/docker-compose
fi

# containerd systemd 单元
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

# dockerd systemd 单元
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

getent group docker >/dev/null 2>&1 || groupadd docker

systemctl daemon-reload
systemctl enable --now containerd
systemctl enable --now docker

echo
docker version
echo "Docker 安装完成。"

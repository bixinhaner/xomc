#!/bin/bash
# Docker DNS 问题修复脚本
# 使用方法: bash fix-docker-dns.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/docker-network-lib.sh"

echo "=== Docker DNS 问题修复 ==="
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行,请先启动 Docker"
    exit 1
fi

HOST_OS="$(uname -s 2>/dev/null || echo unknown)"
DOCKER_OPERATING_SYSTEM="$(docker info --format '{{.OperatingSystem}}' 2>/dev/null || true)"
if [ "$HOST_OS" = "Darwin" ] || [[ "$DOCKER_OPERATING_SYSTEM" == *"Docker Desktop"* ]]; then
    echo "⚠️  Docker Desktop/macOS 仅执行诊断，不写入 /etc/docker/daemon.json，也不通过 systemctl 重启 Docker。"
    echo "当前 bridge 网络："
    docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}} {{end}}' 2>/dev/null || echo "  无法读取 bridge 网络"
    if docker image inspect alpine:3.19 >/dev/null 2>&1; then
        if docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com >/dev/null 2>&1; then
            echo "✅ bridge 网络 DNS 解析正常"
        else
            echo "❌ bridge 网络 DNS 解析失败"
        fi
    else
        echo "ℹ️  本地缺少 alpine:3.19，跳过容器 DNS 探针"
    fi
    echo "请在 Docker Desktop → Settings → Docker Engine 中手工配置 DNS，并按 Docker Desktop UI 重启。"
    exit 0
fi
[ "$HOST_OS" = "Linux" ] || {
    echo "❌ 当前系统不支持写入 Linux Docker daemon 配置：$HOST_OS"
    exit 1
}
if ! docker_network_require_python3; then
    echo "❌ 缺少 python3，无法安全计算和写入 Docker 网段策略"
    exit 1
fi
if ! docker_network_resolve_bip; then
    echo "❌ 无法解析 Docker 网段规划（默认值或自定义 DOCKER_BIP 均不可用）"
    exit 1
fi
docker_network_plan || {
    echo "❌ DOCKER_BIP 无效或无法派生 Docker 网段: $DOCKER_BIP"
    exit 1
}

echo "1️⃣  备份当前 Docker 配置..."
if [ -f /etc/docker/daemon.json ]; then
    sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.backup.$(date +%Y%m%d_%H%M%S)
    echo "✅ 已备份到 /etc/docker/daemon.json.backup.*"
else
    echo "ℹ️  /etc/docker/daemon.json 不存在,将创建新文件"
fi

echo ""
echo "2️⃣  配置 DNS 服务器..."

if [ -f /etc/docker/daemon.json ]; then
    echo "ℹ️  检测到现有配置,将合并 DNS 配置..."
    
    # 使用 jq 合并配置 (如果已安装)
    if command -v jq &> /dev/null; then
        # 备份原文件
        sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.backup.$(date +%Y%m%d_%H%M%S)
        
        # 合并 DNS 配置 (保留其他配置)
        sudo jq '. + {
            "dns": ["223.5.5.5", "223.6.6.6", "114.114.114.114", "8.8.8.8"],
            "dns-search": [],
            "dns-opts": ["timeout:2", "attempts:3"]
        }' /etc/docker/daemon.json | sudo tee /etc/docker/daemon.json.new > /dev/null
        
        sudo mv /etc/docker/daemon.json.new /etc/docker/daemon.json
        echo "✅ DNS 配置已合并到 /etc/docker/daemon.json"
    else
        # 没有 jq,手动处理
        echo "⚠️  未安装 jq,将手动合并配置..."
        echo ""
        echo "请选择操作:"
        echo "  1) 安装 jq 后自动合并 (推荐)"
        echo "  2) 手动编辑 /etc/docker/daemon.json"
        echo "  3) 跳过,稍后手动配置"
        read -p "请选择 [1-3]: " choice
        
        case $choice in
            1)
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    brew install jq
                else
                    sudo apt-get install -y jq || sudo yum install -y jq
                fi
                # 重新执行合并
                exec "$0"
                ;;
            2)
                echo ""
                echo "请编辑 /etc/docker/daemon.json,添加以下内容:"
                echo ""
                cat << 'EOF'
  "dns": ["223.5.5.5", "223.6.6.6", "114.114.114.114", "8.8.8.8"],
  "dns-search": [],
  "dns-opts": ["timeout:2", "attempts:3"],
EOF
                echo ""
                sudo vi /etc/docker/daemon.json
                echo "✅ 配置已更新"
                ;;
            3)
                echo "⏭️  跳过 DNS 配置"
                echo "   请手动编辑 /etc/docker/daemon.json 添加 DNS 配置"
                ;;
            *)
                echo "❌ 无效选择"
                exit 1
                ;;
        esac
    fi
else
    echo "ℹ️  /etc/docker/daemon.json 不存在,将创建新文件"
    sudo cat > /etc/docker/daemon.json << 'EOF'
{
  "dns": [
    "223.5.5.5",
    "223.6.6.6",
    "114.114.114.114",
    "8.8.8.8"
  ],
  "dns-search": [],
  "dns-opts": [
    "timeout:2",
    "attempts:3"
  ]
}
EOF
    echo "✅ DNS 配置已写入 /etc/docker/daemon.json"
fi

echo ""
echo "2️⃣  按 DOCKER_BIP 锁定 Docker 网段..."
if ! command -v python3 > /dev/null 2>&1; then
    echo "❌ 缺少 python3,无法安全写入 Docker 网段策略"
    exit 1
fi
sudo python3 - /etc/docker/daemon.json "$DOCKER_BIP" "$DOCKER_ADDR_POOL_BASE" "$DOCKER_ADDR_POOL_SIZE" <<'PYEOF'
import json
import os
import sys

path, bip, pool_base, pool_size = sys.argv[1:]
data = {}
if os.path.exists(path) and os.path.getsize(path):
    try:
        with open(path, encoding="utf-8") as handle:
            data = json.load(handle)
    except json.JSONDecodeError:
        data = {}
data["bip"] = bip
data["default-address-pools"] = [{"base": pool_base, "size": int(pool_size)}]
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle, indent=2, ensure_ascii=False)
    handle.write("\n")
PYEOF
echo "✅ Docker 网段已按 DOCKER_BIP=$DOCKER_BIP 规划，地址池=$DOCKER_ADDR_POOL_BASE/$DOCKER_ADDR_POOL_SIZE"

echo ""
echo "3️⃣  重启 Docker 服务..."
sudo systemctl restart docker
echo "✅ Docker 服务已重启"

BRIDGE_SUBNETS="$(docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}} {{end}}' 2>/dev/null || true)"
if [ "$BRIDGE_SUBNETS" != "${DOCKER_NETWORK_SUBNET} " ]; then
    echo "❌ Docker bridge 不符合 DOCKER_BIP=$DOCKER_BIP 规划: ${BRIDGE_SUBNETS:-未知} (期望 ${DOCKER_NETWORK_SUBNET})"
    exit 1
fi

echo ""
echo "4️⃣  清理 Docker 构建缓存..."
docker builder prune -f
echo "✅ 构建缓存已清理"

echo ""
echo "5️⃣  测试 DNS 解析..."
if docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com > /dev/null 2>&1; then
    echo "✅ DNS 解析正常"
else
    echo "⚠️  DNS 解析可能仍有问题,请检查网络"
fi

echo ""
echo "6️⃣  重新构建镜像..."
echo "   cd deployments/docker"
echo "   docker compose build"
echo ""
echo "=== 修复完成 ==="
echo ""
echo "如果问题仍然存在,请尝试:"
echo "  1. 检查防火墙设置"
echo "  2. 使用方案 2: 在 docker-compose.yml 中添加 dns 配置"
echo "  3. 使用方案 3: 修改 Dockerfile 使用国内基础镜像"

#!/bin/bash
# Docker DNS 问题诊断脚本
# 使用方法: bash diagnose-dns.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/docker-network-lib.sh"

echo "========================================="
echo "  Docker DNS 问题诊断工具"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查结果计数
PASS=0
FAIL=0
WARN=0

check_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
        FAIL=$((FAIL + 1))
    fi
}

warn_result() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
    WARN=$((WARN + 1))
}

echo "1️⃣  宿主机网络检查"
echo "-----------------------------------------"

# 检查 DNS 解析
echo -n "  测试 DNS 解析 (mirrors.aliyun.com)... "
if nslookup mirrors.aliyun.com > /dev/null 2>&1; then
    check_result 0 "DNS 解析正常"
else
    check_result 1 "DNS 解析失败"
fi

echo -n "  测试 DNS 解析 (goproxy.cn)... "
if nslookup goproxy.cn > /dev/null 2>&1; then
    check_result 0 "DNS 解析正常"
else
    check_result 1 "DNS 解析失败"
fi

# 检查网络连通性
echo -n "  测试 HTTPS 连接 (mirrors.aliyun.com)... "
if curl -s --connect-timeout 5 https://mirrors.aliyun.com > /dev/null; then
    check_result 0 "HTTPS 连接正常"
else
    check_result 1 "HTTPS 连接失败"
fi

echo ""
echo "2️⃣  Docker 环境检查"
echo "-----------------------------------------"

# 检查 Docker 是否运行
echo -n "  Docker 服务状态... "
if docker info > /dev/null 2>&1; then
    check_result 0 "Docker 运行正常"
else
    check_result 1 "Docker 未运行"
    echo "  ${RED}请先启动 Docker${NC}"
    exit 1
fi

# 检查 Docker DNS 配置
echo -n "  Docker daemon DNS 配置... "
if [ -f /etc/docker/daemon.json ]; then
    if grep -q '"dns"' /etc/docker/daemon.json; then
        check_result 0 "已配置 DNS"
        echo "    配置内容:"
        grep -A 5 '"dns"' /etc/docker/daemon.json | sed 's/^/    /'
    else
        warn_result "daemon.json 存在但未配置 DNS"
    fi
else
    warn_result "/etc/docker/daemon.json 不存在"
fi

# 检查 Docker 网络
echo -n "  Docker 网络状态... "
if docker network ls | grep -q bridge; then
    check_result 0 "bridge 网络正常"
else
    check_result 1 "bridge 网络异常"
fi

if docker_network_resolve_bip && docker_network_plan; then
    BRIDGE_SUBNETS="$(docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}} {{end}}' 2>/dev/null || true)"
else
    BRIDGE_SUBNETS=""
fi
if [ "$BRIDGE_SUBNETS" = "${DOCKER_NETWORK_SUBNET} " ]; then
    check_result 0 "bridge 网络符合 DOCKER_BIP 规划 ($BRIDGE_SUBNETS)"
else
    check_result 1 "bridge 网络不符合 DOCKER_BIP 规划 (${BRIDGE_SUBNETS:-未知}，期望 ${DOCKER_NETWORK_SUBNET:-未知})"
    echo "  ${RED}请先按 DOCKER_BIP 规划运行 install-docker.sh，避免创建未规划网络${NC}"
    exit 1
fi

echo ""
echo "3️⃣  容器 DNS 检查"
echo "-----------------------------------------"

# 检查容器 DNS 配置
echo "  容器内 /etc/resolv.conf 内容:"
docker run --network bridge --rm alpine:3.19 cat /etc/resolv.conf 2>/dev/null | sed 's/^/    /' || {
    check_result 1 "无法启动测试容器"
}

# 测试容器 DNS 解析
echo -n "  容器内 DNS 解析 (mirrors.aliyun.com)... "
if docker run --network bridge --rm alpine:3.19 nslookup mirrors.aliyun.com > /dev/null 2>&1; then
    check_result 0 "容器 DNS 解析正常"
else
    check_result 1 "容器 DNS 解析失败"
fi

echo -n "  容器内 DNS 解析 (goproxy.cn)... "
if docker run --network bridge --rm alpine:3.19 nslookup goproxy.cn > /dev/null 2>&1; then
    check_result 0 "容器 DNS 解析正常"
else
    check_result 1 "容器 DNS 解析失败"
fi

# 测试容器网络
echo -n "  容器内 HTTPS 连接... "
if docker run --network bridge --rm alpine:3.19 wget --spider -q https://mirrors.aliyun.com 2>/dev/null; then
    check_result 0 "容器 HTTPS 连接正常"
else
    check_result 1 "容器 HTTPS 连接失败"
fi

echo ""
echo "4️⃣  Docker 构建测试"
echo "-----------------------------------------"

# 测试 Alpine 包管理器
echo -n "  测试 apk add (git)... "
if docker run --network bridge --rm alpine:3.19 sh -c "apk add --no-cache git > /dev/null 2>&1"; then
    check_result 0 "apk add 正常"
else
    check_result 1 "apk add 失败 (DNS 问题)"
fi

# 测试 Go 模块下载
echo -n "  测试 go mod download... "
if docker run --network bridge --rm -e GOPROXY=https://mirrors.aliyun.com/goproxy/,direct \
    golang:1.25-alpine sh -c \
    "cd /tmp && go mod init test && go get github.com/Masterminds/squirrel@v1.5.4" > /dev/null 2>&1; then
    check_result 0 "go mod download 正常"
else
    check_result 1 "go mod download 失败"
fi

echo ""
echo "5️⃣  系统变化检查"
echo "-----------------------------------------"

# 检查 Docker 版本
echo "  Docker 版本:"
docker --version | sed 's/^/    /'

# 检查 Docker 重启时间
echo -n "  Docker 最后重启时间... "
if command -v systemctl > /dev/null 2>&1; then
    systemctl show docker --property=ActiveEnterTimestamp | sed 's/ActiveEnterTimestamp=//'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macOS, 请检查 Docker Desktop 日志"
fi

# 检查系统 DNS 变更
echo "  系统 DNS 配置 (/etc/resolv.conf):"
cat /etc/resolv.conf 2>/dev/null | grep -E "^nameserver" | sed 's/^/    /' || echo "    无法读取"

# 检查防火墙
echo -n "  防火墙状态... "
if command -v ufw > /dev/null 2>&1; then
    if ufw status | grep -q "active"; then
        warn_result "防火墙已启用,可能影响 DNS"
        ufw status | head -5 | sed 's/^/    /'
    else
        check_result 0 "防火墙未启用"
    fi
elif command -v firewall-cmd > /dev/null 2>&1; then
    if firewall-cmd --state 2>/dev/null | grep -q "running"; then
        warn_result "firewalld 运行中"
    else
        check_result 0 "防火墙未运行"
    fi
else
    echo "未检测到防火墙"
fi

echo ""
echo "========================================="
echo "  诊断结果汇总"
echo "========================================="
echo -e "  ${GREEN}通过: $PASS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "  ${YELLOW}警告: $WARN${NC}"
echo ""

if [ $FAIL -gt 0 ]; then
    echo -e "${RED}❌ 发现问题,需要修复${NC}"
    echo ""
    echo "推荐修复方案:"
    echo "  1. 配置 Docker daemon DNS (推荐)"
    echo "     cd deployments/docker"
    echo "     bash fix-docker-dns.sh"
    echo ""
    echo "  2. 重启 Docker"
    echo "     macOS: Docker Desktop → Restart"
    echo "     Linux: sudo systemctl restart docker"
    echo ""
    echo "  3. 清理缓存并重新构建"
    echo "     docker builder prune -f"
    echo "     docker compose build"
else
    echo -e "${GREEN}✅ 所有检查通过${NC}"
    echo ""
    echo "如果构建仍然失败,可能是:"
    echo "  - 临时网络波动"
    echo "  - 镜像源暂时不可用"
    echo "  - 建议等待几分钟后重试"
fi

echo ""
echo "========================================="

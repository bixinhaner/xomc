#!/bin/bash
# Docker 网络故障诊断脚本
# 用于排查: DNS 配置正确但网络不通的问题

set -e

echo "========================================="
echo "  Docker 网络故障诊断工具"
echo "========================================="
echo ""

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "1️⃣  Docker 网络配置检查"
echo "-----------------------------------------"

# 检查 Docker 网桥
echo "Docker 网络列表:"
docker network ls | grep bridge

echo "Docker 173.x 网段检查:"
BRIDGE_SUBNETS="$(docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}} {{end}}' 2>/dev/null || true)"
if ! printf '%s' "$BRIDGE_SUBNETS" | grep -Eq '(^|[[:space:]])173\.'; then
    echo -e "${RED}❌ Docker bridge 仍使用 172.x：${BRIDGE_SUBNETS:-未知}${NC}"
    echo "  请先运行: sudo bash /opt/omc/infra/docker/install-docker.sh --skip-if-installed --no-mirror"
    exit 1
else
    echo -e "${GREEN}✅ Docker bridge 使用 173.x：$BRIDGE_SUBNETS${NC}"
fi

echo ""
echo "bridge 网络详情:"
docker network inspect bridge | jq '.[0].IPAM.Config'

echo ""
echo "2️⃣  宿主机网络检查"
echo "-----------------------------------------"

# 检查宿主机 DNS
echo "宿主机 DNS 配置:"
cat /etc/resolv.conf | grep nameserver

echo ""
echo "宿主机网络连通性:"
echo -n "  ping 223.5.5.5... "
if ping -c 1 -W 2 223.5.5.5 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 通${NC}"
else
    echo -e "${RED}❌ 不通${NC}"
fi

echo -n "  ping 8.8.8.8... "
if ping -c 1 -W 2 8.8.8.8 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 通${NC}"
else
    echo -e "${RED}❌ 不通${NC}"
fi

echo ""
echo "3️⃣  Docker 网络转发检查"
echo "-----------------------------------------"

# 检查 IP 转发
echo -n "IP 转发 (net.ipv4.ip_forward)... "
if sysctl net.ipv4.ip_forward 2>/dev/null | grep -q "= 1"; then
    echo -e "${GREEN}✅ 已启用${NC}"
else
    echo -e "${RED}❌ 未启用${NC}"
    echo "  修复: sudo sysctl -w net.ipv4.ip_forward=1"
fi

# 检查 iptables
echo ""
echo "iptables 规则:"
sudo iptables -L -n | head -20

echo ""
echo "iptables NAT 表:"
sudo iptables -t nat -L -n | head -20

echo ""
echo "4️⃣  Docker 容器网络测试"
echo "-----------------------------------------"

# 启动测试容器
echo "启动测试容器..."
CONTAINER_ID=$(docker run --network bridge -d --rm alpine:3.19 sleep 300)

echo ""
echo "容器 IP 地址:"
docker exec $CONTAINER_ID ip addr show eth0

echo ""
echo "容器路由表:"
docker exec $CONTAINER_ID route -n

echo ""
echo "容器内网络测试:"
echo -n "  ping 网关... "
GATEWAY=$(docker exec $CONTAINER_ID route -n | grep '^0.0.0.0' | awk '{print $2}')
if [ -n "$GATEWAY" ]; then
    if docker exec $CONTAINER_ID ping -c 1 -W 2 $GATEWAY > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 通 (网关: $GATEWAY)${NC}"
    else
        echo -e "${RED}❌ 不通 (网关: $GATEWAY)${NC}"
    fi
else
    echo -e "${RED}❌ 未找到网关${NC}"
fi

echo -n "  ping 8.8.8.8 (外部)... "
if docker exec $CONTAINER_ID ping -c 1 -W 2 8.8.8.8 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 通${NC}"
else
    echo -e "${RED}❌ 不通${NC}"
fi

echo -n "  ping 宿主机... "
if [ -n "$GATEWAY" ] && docker exec "$CONTAINER_ID" ping -c 1 -W 2 "$GATEWAY" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 通 ($GATEWAY)${NC}"
else
    echo -e "${RED}❌ 不通 (网关: ${GATEWAY:-未知})${NC}"
fi

# 清理
docker stop $CONTAINER_ID > /dev/null 2>&1

echo ""
echo "5️⃣  防火墙检查"
echo "-----------------------------------------"

# 检查 firewalld
echo -n "firewalld 状态... "
if systemctl is-active firewalld > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  运行中${NC}"
    echo "  这可能阻止 Docker 网络转发"
    echo "  修复: sudo systemctl stop firewalld"
else
    echo -e "${GREEN}✅ 未运行${NC}"
fi

# 检查 ufw
echo -n "ufw 状态... "
if ufw status 2>/dev/null | grep -q "active"; then
    echo -e "${YELLOW}⚠️  已启用${NC}"
    echo "  这可能阻止 Docker 网络转发"
    echo "  修复: sudo ufw disable"
else
    echo -e "${GREEN}✅ 未启用${NC}"
fi

# 检查 iptables FORWARD 链
echo ""
echo "iptables FORWARD 链策略:"
sudo iptables -L FORWARD -n | head -5

echo ""
echo "========================================="
echo "  诊断建议"
echo "========================================="
echo ""
echo "如果容器无法 ping 通外部网络,但宿主机可以:"
echo ""
echo "可能的原因和解决方案:"
echo ""
echo "1️⃣  防火墙阻止 Docker 转发 (最常见)"
echo "   sudo systemctl stop firewalld"
echo "   或"
echo "   sudo ufw disable"
echo ""
echo "2️⃣  IP 转发未启用"
echo "   sudo sysctl -w net.ipv4.ip_forward=1"
echo "   echo 'net.ipv4.ip_forward=1' | sudo tee -a /etc/sysctl.conf"
echo ""
echo "3️⃣  Docker 网络异常,重启 Docker"
echo "   sudo systemctl restart docker"
echo ""
echo "4️⃣  iptables 规则被清空,重建 Docker 网络"
echo "   sudo systemctl stop docker"
echo "   sudo iptables -t nat -F"
echo "   sudo iptables -t filter -F"
echo "   sudo systemctl start docker"
echo ""
echo "========================================="

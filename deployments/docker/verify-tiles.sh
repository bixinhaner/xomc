#!/bin/bash
# 离线地图配置验证脚本
# 用于检查 tiles 目录、tiles.json、nginx 配置是否正确

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TILES_DIR="${SCRIPT_DIR}/../../../tiles"
TILES_JSON="${TILES_DIR}/tiles.json"
NGINX_CONF="${SCRIPT_DIR}/default.conf"

echo "🔍 离线地图配置验证"
echo "===================="
echo ""

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
}

# 检查 tiles 目录
echo "1. 检查 tiles 目录..."
if [ -d "$TILES_DIR" ]; then
    check_pass "tiles 目录存在: ${TILES_DIR}"

    # 检查瓦片文件
    TILE_COUNT=$(find "${TILES_DIR}" -name "*.png" 2>/dev/null | wc -l)
    if [ "$TILE_COUNT" -gt 0 ]; then
        check_pass "发现 ${TILE_COUNT} 个瓦片文件"
    else
        check_warn "未发现瓦片文件 (.png)"
    fi
else
    check_fail "tiles 目录不存在: ${TILES_DIR}"
    echo "   请创建 tiles 目录并放入瓦片文件"
fi
echo ""

# 检查 tiles.json
echo "2. 检查 tiles.json..."
if [ -f "$TILES_JSON" ]; then
    check_pass "tiles.json 存在"

    # 验证 JSON 格式
    if command -v jq &> /dev/null; then
        if jq empty "$TILES_JSON" 2>/dev/null; then
            check_pass "tiles.json 格式正确"

            # 显示关键信息
            NAME=$(jq -r '.name // "N/A"' "$TILES_JSON")
            MINZOOM=$(jq -r '.minzoom // "N/A"' "$TILES_JSON")
            MAXZOOM=$(jq -r '.maxzoom // "N/A"' "$TILES_JSON")
            CENTER=$(jq -r '.center // "N/A"' "$TILES_JSON")
            BOUNDS=$(jq -r '.bounds // "N/A"' "$TILES_JSON")

            echo "   名称: ${NAME}"
            echo "   缩放: ${MINZOOM} - ${MAXZOOM}"
            echo "   中心点: ${CENTER}"
            echo "   边界: ${BOUNDS}"
        else
            check_fail "tiles.json JSON 格式错误"
        fi
    else
        check_warn "未安装 jq，跳过 JSON 格式验证"
    fi
else
    check_fail "tiles.json 不存在: ${TILES_JSON}"
    echo "   请创建 tiles.json 或使用 Maperitive 生成"
fi
echo ""

# 检查 nginx 配置
echo "3. 检查 nginx 配置..."
if [ -f "$NGINX_CONF" ]; then
    if grep -q "/tiles-metadata" "$NGINX_CONF"; then
        check_pass "nginx 配置包含 /tiles-metadata 端点"

        # 检查是否指向 tiles.json
        if grep -q "tiles.json" "$NGINX_CONF"; then
            check_pass "/tiles-metadata 指向 tiles.json"
        else
            check_warn "/tiles-metadata 未指向 tiles.json"
        fi
    else
        check_fail "nginx 配置缺少 /tiles-metadata 端点"
    fi

    if grep -q "/tiles/" "$NGINX_CONF"; then
        check_pass "nginx 配置包含 /tiles/ 瓦片端点"
    else
        check_fail "nginx 配置缺少 /tiles/ 瓦片端点"
    fi
else
    check_warn "nginx 配置文件不存在: ${NGINX_CONF}"
fi
echo ""

# 检查 docker-compose.yml
echo "4. 检查 docker-compose.yml..."
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
if [ -f "$COMPOSE_FILE" ]; then
    if grep -q "./tiles:/usr/share/nginx/tiles" "$COMPOSE_FILE"; then
        check_pass "docker-compose.yml 已挂载 tiles 目录"
    else
        check_fail "docker-compose.yml 未挂载 tiles 目录"
        echo "   请在 web 服务 volumes 中添加:"
        echo "   - ./tiles:/usr/share/nginx/tiles:ro"
    fi
else
    check_warn "docker-compose.yml 不存在: ${COMPOSE_FILE}"
fi
echo ""

# 测试服务端点（如果服务正在运行）
echo "5. 测试服务端点..."
if command -v curl &> /dev/null; then
    # 测试元数据端点
    METADATA_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/tiles-metadata" 2>/dev/null || echo "000")

    if [ "$METADATA_STATUS" = "200" ]; then
        check_pass "/tiles-metadata 端点可访问 (HTTP 200)"

        # 显示返回的内容摘要
        METADATA_CONTENT=$(curl -s "http://localhost:8081/tiles-metadata" 2>/dev/null)
        if command -v jq &> /dev/null; then
            METADATA_NAME=$(echo "$METADATA_CONTENT" | jq -r '.name // "N/A"')
            echo "   服务端地图名称: ${METADATA_NAME}"
        fi
    elif [ "$METADATA_STATUS" = "404" ]; then
        check_warn "/tiles-metadata 返回 404 (可能 tiles.json 不存在)"
    elif [ "$METADATA_STATUS" = "000" ]; then
        check_warn "无法连接到 http://localhost:8081 (服务未启动?)"
    else
        check_warn "/tiles-metadata 返回 HTTP ${METADATA_STATUS}"
    fi

    # 测试瓦片端点
    TILE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/tiles/6/32/42.png" 2>/dev/null || echo "000")

    if [ "$TILE_STATUS" = "200" ]; then
        check_pass "/tiles/ 端点可访问 (HTTP 200)"
    elif [ "$TILE_STATUS" = "404" ]; then
        check_warn "/tiles/ 返回 404 (瓦片文件可能不存在)"
    elif [ "$TILE_STATUS" = "000" ]; then
        check_warn "无法连接到 http://localhost:8081 (服务未启动?)"
    else
        check_warn "/tiles/ 返回 HTTP ${TILE_STATUS}"
    fi
else
    check_warn "未安装 curl，跳过端点测试"
fi
echo ""

echo "===================="
echo "✨ 验证完成"
echo ""
echo "提示："
echo "  - 如果服务未启动，请先运行: docker-compose up -d web"
echo "  - 如需生成新的 tiles.json，可使用 Maperitive 工具"
echo "  - 详细配置请参考: docs/design/offline-map-code-changes.md"

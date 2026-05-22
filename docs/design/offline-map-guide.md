# 离线地图部署指南

> **版本**: v1.0
> **更新日期**: 2026-05-22
> **适用场景**: 无外网环境下部署离线地图瓦片服务

---

## 一、架构概述

```
                    ┌─────────────────────────────────────────┐
                    │           前端 (webcode)                 │
                    │  ┌─────────────────────────────────────┐ │
                    │  │  useMapConfig Hook                  │ │
                    │  │  → GET /tiles-metadata              │ │
                    │  │  → 获取 tiles.json                  │ │
                    │  │  → 自动应用地图配置                  │ │
                    │  └─────────────────────────────────────┘ │
                    └─────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌─────────────────────────────────────────┐
                    │         Nginx (web 容器)                 │
                    │  ┌─────────────────────────────────────┐ │
                    │  │  /tiles-metadata → tiles.json       │ │
                    │  │  /tiles/{z}/{x}/{y}.png → 瓦片文件  │ │
                    │  └─────────────────────────────────────┘ │
                    └─────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
            ┌──────────────┐              ┌──────────────┐
            │  tiles.json  │              │  瓦片目录    │
            │  (元数据)    │              │  /{z}/{x}/{y}│
            └──────────────┘              └──────────────┘
```

---

## 二、快速开始

### 2.1 准备瓦片文件

使用 Maperitive 工具生成离线瓦片：

```bash
# 1. 在 Maperitive 中加载 OSM 数据
# 2. 设置导出边界和缩放级别
# 3. 导出时勾选 "Generate tiles.json"
# 4. 导出完成后得到 tiles/ 目录
```

### 2.2 部署步骤

```bash
# 1. 将瓦片目录复制到部署位置
cd goomc/deployments/docker/
cp -r /path/to/your/tiles ./tiles

# 2. 验证配置
bash verify-tiles.sh

# 3. 重启 web 服务
docker-compose restart web

# 4. 验证服务
curl http://localhost:8081/tiles-metadata | jq
curl -I http://localhost:8081/tiles/6/32/42.png
```

---

## 三、配置说明

### 3.1 tiles.json 格式

Maperitive 自动生成的 `tiles.json` 文件：

```json
{
  "tilejson": "2.0.0",
  "name": "区域名称",
  "description": "地图描述",
  "attribution": "© OpenStreetMap contributors",
  "tiles": ["tiles/{z}/{x}/{y}.png"],
  "minzoom": 6,
  "maxzoom": 15,
  "bounds": [minLon, minLat, maxLon, maxLat],
  "center": [lon, lat, zoom]
}
```

| 字段 | 说明 | 示例 |
|------|------|------|
| name | 地图名称 | "Zambia Offline Map" |
| minzoom | 最小缩放级别 | 6 |
| maxzoom | 最大缩放级别 | 15 |
| bounds | 地理边界 | [22.0, -18.0, 34.0, -8.0] |
| center | 中心点 | [28.221, -14.607, 6] |

### 3.2 环境变量配置

**开发环境** (`.env.development`):
```bash
# 瓦片 URL（相对路径，由 Vite 代理）
VITE_MAP_TILE_URL=/tiles/{z}/{x}/{y}.png

# 代理目标（本地 nginx）
VITE_TILES_PROXY_TARGET=http://localhost:8081
```

**生产环境** (`.env.production`):
```bash
# 瓦片 URL（相对路径，由 nginx 直接服务）
VITE_MAP_TILE_URL=/tiles/{z}/{x}/{y}.png
```

### 3.3 Nginx 配置

已配置在 `default.conf`:

```nginx
# 元数据端点
location /tiles-metadata {
    alias /usr/share/nginx/tiles/tiles.json;
    add_header Content-Type application/json;
    add_header Cache-Control "public, max-age=3600";
}

# 瓦片端点
location /tiles/ {
    alias /usr/share/nginx/tiles/;
    expires 30d;
    add_header Cache-Control "public, immutable";
}
```

### 3.4 Docker Compose 挂载

已配置在 `docker-compose.yml`:

```yaml
web:
  volumes:
    - ./tiles:/usr/share/nginx/tiles:ro
```

---

## 四、异常处理

### 4.1 自动降级机制

前端实现了完善的异常处理，任何异常都不会导致地图无法显示：

| 异常场景 | 处理方式 | 用户体验 |
|---------|---------|---------|
| tiles.json 不存在 | 使用默认配置 | 地图正常显示 |
| tiles.json 格式错误 | 使用默认配置 | 地图正常显示 |
| 网络请求失败 | 使用默认配置 | 地图正常显示 |
| Nginx 未配置 | 使用在线 OSM | 地图正常显示 |

### 4.2 默认配置

当元数据加载失败时，使用以下默认值：

```typescript
{
  name: 'Zambia Offline Map',
  region: 'Zambia',
  center: { lon: 28.221, lat: -14.607, zoom: 6 },
  bounds: { minLon: 22.0, maxLon: 34.0, minLat: -18.0, maxLat: -8.0 },
  zoom: { min: 6, max: 15, default: 6 },
  attribution: '© OpenStreetMap contributors',
  description: '赞比亚区域离线地图，包含 6-15 级瓦片'
}
```

### 4.3 调试日志

正常加载：
```
[MapConfig] ✓ Loaded TileJSON: Zambia Offline Map center: [28.221, -14.607] zoom: 6-15
```

异常降级：
```
[MapConfig] ⚠ Failed to fetch metadata (HTTP 404), using defaults
[MapConfig] ⚠ TileJSON format incomplete, using defaults
```

---

## 五、验证清单

### 5.1 部署前验证

```bash
# 运行验证脚本
cd goomc/deployments/docker/
bash verify-tiles.sh
```

检查项：
- [ ] `tiles/` 目录存在
- [ ] `tiles/tiles.json` 文件存在且格式正确
- [ ] 瓦片文件存在 (`*.png`)
- [ ] Nginx 配置正确
- [ ] Docker Compose 挂载正确

### 5.2 部署后验证

```bash
# 1. 检查容器挂载
docker exec $(docker ps -q -f name=web) ls -la /usr/share/nginx/tiles/

# 2. 测试元数据端点
curl http://localhost:8081/tiles-metadata | jq

# 3. 测试瓦片端点
curl -I http://localhost:8081/tiles/6/32/42.png

# 4. 检查前端控制台
# 浏览器开发者工具 → Console → 查看 [MapConfig] 日志
```

---

## 六、切换地图区域

### 6.1 完整替换

```bash
# 1. 替换瓦片目录
cd goomc/deployments/docker/
rm -rf tiles
cp -r /path/to/new/tiles ./tiles

# 2. 确保新的 tiles.json 存在
ls tiles/tiles.json

# 3. 重启服务
docker-compose restart web
```

### 6.2 多区域切换（高级）

```bash
# 1. 创建多个区域目录
mkdir -p tiles/zambia tiles/kenya tiles/tanzania

# 2. 修改 nginx 配置支持多区域
location /tiles-zambia/ {
    alias /usr/share/nginx/tiles/zambia/;
}
location /tiles-kenya/ {
    alias /usr/share/nginx/tiles/kenya/;
}

# 3. 修改环境变量动态切换
VITE_MAP_TILE_URL=/tiles-zambia/{z}/{x}/{y}.png
```

---

## 七、故障排查

### 7.1 瓦片显示空白

**症状**: 地图加载但显示空白方块

**排查**:
```bash
# 1. 检查瓦片文件是否存在
ls -la goomc/deployments/docker/tiles/6/32/42.png

# 2. 检查 nginx 是否能访问
curl -I http://localhost:8081/tiles/6/32/42.png

# 3. 检查容器内文件
docker exec $(docker ps -q -f name=web) ls -la /usr/share/nginx/tiles/6/32/42.png
```

### 7.2 元数据加载失败

**症状**: 控制台显示 `[MapConfig] ⚠ Failed to fetch metadata`

**排查**:
```bash
# 1. 检查 tiles.json 是否存在
cat goomc/deployments/docker/tiles/tiles.json

# 2. 验证 JSON 格式
cat goomc/deployments/docker/tiles/tiles.json | jq

# 3. 测试端点
curl http://localhost:8081/tiles-metadata
```

### 7.3 地图中心点不正确

**症状**: 地图加载但中心点不在预期位置

**解决方案**: 修改 `tiles.json` 中的 `center` 和 `bounds` 字段

---

## 八、相关文档

- [离线地图代码修改清单](./offline-map-code-changes.md)
- [useMapConfig 源码](../../goomc/omcmb/webcode/src/components/GISMap/useMapConfig.ts)
- [Nginx 配置](../../goomc/deployments/docker/default.conf)

# 离线地图优化代码修改清单

> **文档版本**：v2.0
> **创建日期**：2026-05-22
> **更新日期**：2026-05-22
> **基于文档**：[离线地图完整部署指南](./offline-map-complete-guide.md)

> **目的**：根据最新部署指南，分析当前代码需要做的修改，实现元数据驱动的零配置切换。使用 Maperitive 生成的 tiles.json 作为元数据源。

---

## 一、修改概览

| 模块 | 文件 | 修改类型 | 优先级 | 预计工作量 |
|------|------|---------|--------|-----------|
| **服务端** | `default.conf` | 修改 location 路径 | P0 | 5 分钟 |
| **服务端** | `docker-compose.yml` | 新增 volume 挂载 | P0 | 5 分钟 |
| **前端** | `.env.development` | 修改路径 | P0 | 1 分钟 |
| **前端** | `.env.production` | 修改路径 | P0 | 1 分钟 |
| **前端** | `vite.config.ts` | 修改代理 | P0 | 1 分钟 |
| **前端** | `components/GISMap/useMapConfig.ts` | 新建文件 | P0 | 30 分钟 |
| **前端** | `components/GISMap/useOLMap.ts` | 集成元数据 | P1 | 30 分钟 |
| **前端** | `components/GISMap/constants.ts` | 新增默认配置 | P1 | 10 分钟 |

> **变更说明**：v2.0 使用 Maperitive 自动生成的 `tiles.json` 作为元数据源，无需手动创建 `metadata.json`。

---

## 二、服务端修改

### 2.1 Nginx 配置 - 修改元数据端点

**文件**：`goomc/deployments/docker/default.conf`

**当前状态**：
```nginx
# --- 离线地图元数据（新增）---
location /tiles-metadata {
    alias /usr/share/nginx/tiles/metadata.json;
    add_header Content-Type application/json;
    add_header Cache-Control "public, max-age=3600";
    default_type application/json;
}
```

**需要修改为**：
```nginx
# --- 离线地图元数据（使用 Maperitive 生成的 tiles.json）---
location /tiles-metadata {
    alias /usr/share/nginx/tiles/tiles.json;
    add_header Content-Type application/json;
    add_header Cache-Control "public, max-age=3600";
    default_type application/json;
}

# --- 离线地图瓦片（保持不变）---
location /tiles/ {
    alias /usr/share/nginx/tiles/;
    expires 30d;
    add_header Cache-Control "public, immutable";
}
```

**修改说明**：
- ✅ **修改** `/tiles-metadata` 的 `alias` 路径：`metadata.json` → `tiles.json`
- ✅ Maperitive 自动生成 `tiles.json`，包含完整的中心点、边界、缩放级别信息
- ✅ 无需手动维护元数据文件

### 2.2 Docker Compose - 挂载 tiles 目录

**文件**：`goomc/deployments/docker/docker-compose.yml`

**当前状态**：
```yaml
web:
  # ... 其他配置 ...
  volumes:
    - ../../run/logs/nginx:/var/log/nginx
    - /etc/localtime:/etc/localtime:ro
```

**需要修改为**：
```yaml
web:
  # ... 其他配置 ...
  volumes:
    - ../../run/logs/nginx:/var/log/nginx
    - ./tiles:/usr/share/nginx/tiles:ro    # ← 新增：瓦片目录挂载
    - /etc/localtime:/etc/localtime:ro
```

**修改说明**：
- ✅ **新增** tiles 目录挂载（只读模式）
- 📁 挂载源：`./tiles` 相对于 docker-compose.yml 所在目录
- 📁 挂载目标：`/usr/share/nginx/tiles` 容器内路径

**验证命令**：
```bash
# 启动容器后验证
docker exec $(docker ps -q -f name=web) ls -la /usr/share/nginx/tiles/
# 应该看到 tiles.json 和瓦片目录
```

### 2.3 tiles.json 格式说明

**文件**：`goomc/deployments/docker/tiles/tiles.json`（Maperitive 自动生成）

```json
{
    "tilejson": "1.0.0",
    "name": "My Map",
    "description": "Made with Maperitive",
    "attribution": "Map data © OpenStreetMap contributors",
    "tiles": ["tiles/{z}/{x}/{y}.png"],
    "minzoom": 6,
    "maxzoom": 15,
    "bounds": [21.989822, -18.087769, 33.7177269, -8.239483],
    "center": [27.853774450000003, -13.163626, 6]
}
```

**TileJSON 字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 地图名称 |
| description | string | 地图描述 |
| attribution | string | 版权信息 |
| minzoom | number | 最小缩放级别 |
| maxzoom | number | 最大缩放级别 |
| bounds | [minLon, minLat, maxLon, maxLat] | 地理边界数组 |
| center | [lon, lat, zoom] | 中心点数组 |

---

## 三、前端修改

### 3.1 新建元数据加载 Hook（支持 TileJSON）

**文件**：`goomc/omcmb/webcode/src/components/GISMap/useMapConfig.ts`（新建）

```typescript
/**
 * 地图元数据加载 Hook - 支持 TileJSON 格式
 * @module components/GISMap/useMapConfig
 */

import { useState, useEffect } from 'react';

/**
 * TileJSON 格式（Maperitive / TileServer GL 生成）
 * 规范：https://github.com/mapbox/tilejson-spec
 */
export interface TileJSON {
  tilejson: string;
  name: string;
  description: string;
  attribution: string;
  tiles: string[];
  minzoom: number;
  maxzoom: number;
  bounds: [number, number, number, number];  // [minLon, minLat, maxLon, maxLat]
  center: [number, number, number];          // [lon, lat, zoom]
}

/**
 * 内部统一格式（与原 metadata.json 兼容）
 */
export interface MapMetadata {
  name: string;
  region: string;
  center: { lon: number; lat: number; zoom: number };
  bounds: { minLon: number; maxLon: number; minLat: number; maxLat: number };
  zoom: { min: number; max: number; default: number };
  attribution: string;
  description: string;
}

/**
 * 默认元数据（降级方案）
 * 与当前 constants.ts 中的 MAP_CONFIG 保持一致
 */
export const DEFAULT_METADATA: MapMetadata = {
  name: 'Zambia Offline Map',
  region: 'Zambia',
  center: { lon: 28.221, lat: -14.607, zoom: 6 },
  bounds: { minLon: 22.0, maxLon: 34.0, minLat: -18.0, maxLat: -8.0 },
  zoom: { min: 6, max: 15, default: 6 },
  attribution: '© OpenStreetMap contributors',
  description: '赞比亚区域离线地图，包含 6-15 级瓦片',
};

/**
 * 将 TileJSON 转换为 MapMetadata
 */
function tileJsonToMetadata(tilejson: TileJSON): MapMetadata {
  const [minLon, minLat, maxLon, maxLat] = tilejson.bounds;
  const [lon, lat, zoom] = tilejson.center;

  return {
    name: tilejson.name,
    region: tilejson.name,  // 用 name 作为 region
    center: { lon, lat, zoom },
    bounds: { minLon, maxLon, minLat, maxLat },
    zoom: {
      min: tilejson.minzoom,
      max: tilejson.maxzoom,
      default: zoom,
    },
    attribution: tilejson.attribution,
    description: tilejson.description,
  };
}

/**
 * 地图元数据加载 Hook
 * @returns 元数据、加载状态、错误信息
 */
export function useMapConfig() {
  const [metadata, setMetadata] = useState<MapMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        // 优先从服务端获取元数据
        const response = await fetch('/tiles-metadata');

        if (response.ok) {
          const tilejson: TileJSON = await response.json();
          console.log('[MapConfig] Loaded TileJSON:', tilejson.name);
          setMetadata(tileJsonToMetadata(tilejson));
        } else {
          // 降级到默认配置
          console.warn('[MapConfig] Failed to fetch metadata, using defaults');
          setMetadata(DEFAULT_METADATA);
        }
      } catch (err) {
        console.error('[MapConfig] Error fetching metadata:', err);
        // 降级到默认配置
        setMetadata(DEFAULT_METADATA);
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, []);

  return { metadata, loading, error };
}
```

### 3.2 修改 constants.ts - 添加元数据构建函数

**文件**：`goomc/omcmb/webcode/src/components/GISMap/constants.ts`

**在文件末尾添加**：
```typescript
/**
 * 从元数据构建地图配置
 * @param metadata 地图元数据
 * @returns 地图配置对象
 */
export function buildMapConfigFromMetadata(metadata: MapMetadata) {
  return {
    defaultCenter: [metadata.center.lon, metadata.center.lat] as [number, number],
    defaultZoom: metadata.center.zoom,
    minZoom: metadata.zoom.min,
    maxZoom: metadata.zoom.max,
    tileUrl: import.meta.env.VITE_MAP_TILE_URL || MAP_CONFIG.osmTileUrl,
    attribution: metadata.attribution,
    bounds: metadata.bounds,
  };
}
```

**需要导入 MapMetadata 类型**：
```typescript
import type { MapMetadata } from './useMapConfig';
```

### 3.3 修改 useOLMap.ts - 集成元数据

**文件**：`goomc/omcmb/webcode/src/components/GISMap/useOLMap.ts`

**修改 1**：添加导入
```typescript
import { useMapConfig, type MapMetadata } from './useMapConfig';
import { buildMapConfigFromMetadata } from './constants';
```

**修改 2**：在 `useOLMap` 函数开始处添加元数据加载
```typescript
export function useOLMap(options: UseOLMapOptions = {}): UseOLMapReturn {
  const { metadata, loading: metadataLoading } = useMapConfig();

  // 原有的 ref 和 state 定义...
  const mapRef = useRef<HTMLDivElement | null>(null);
  // ...
```

**修改 3**：修改地图初始化逻辑，使用元数据配置
```typescript
// 找到 useEffect 中的地图初始化部分
useEffect(() => {
  if (!mapRef.current || mapInstanceRef.current || !metadata) return;

  // 使用元数据中的配置或传入的 options
  const center = options.center || [metadata.center.lon, metadata.center.lat];
  const zoom = options.zoom || metadata.center.zoom;
  const minZoom = options.minZoom ?? metadata.zoom.min;
  const maxZoom = options.maxZoom ?? metadata.zoom.max;

  // 创建瓦片图层
  const tileLayer = new TileLayer({
    source: new XYZ({
      url: options.tileUrl || import.meta.env.VITE_MAP_TILE_URL || MAP_CONFIG.osmTileUrl,
      attributions: metadata.attribution,
    }),
  });

  // ... 其余代码保持不变
}, [metadata, options]);
```

**修改 4**：修改返回值，添加元数据加载状态
```typescript
return {
  mapRef,
  mapInstanceRef,
  updateDevices,
  getViewport,
  flyTo,
  highlightDevice,
  clearHighlight,
  isReady: isReady && !metadataLoading,  // ← 等待元数据加载完成
  updateSize,
  getZoom,
  fitBounds,
  highlightAndSpiderfyIfNeeded,
  metadata,  // ← 新增：暴露元数据
};
```

### 3.4 修改 index.tsx - 处理加载状态

**文件**：`goomc/omcmb/webcode/src/components/GISMap/index.tsx`

**修改 1**：添加元数据相关的 state
```typescript
const [, setMetadata] = useState<MapMetadata | null>(null);
```

**修改 2**：从 useOLMap 获取元数据
```typescript
const {
  mapRef,
  mapInstanceRef,
  updateDevices,
  getViewport,
  flyTo,
  clearHighlight,
  isReady,
  updateSize,
  highlightAndSpiderfyIfNeeded,
  metadata,  // ← 新增
} = useOLMap({
  // ... options
});
```

**修改 3**：添加元数据变化的副作用（可选）
```typescript
// 当元数据加载完成后，通知父组件
useEffect(() => {
  if (metadata) {
    console.log('[GISMap] Metadata loaded:', metadata.region);
    setMetadata(metadata);
    // 可以在这里触发自定义事件或回调
  }
}, [metadata]);
```

---

## 四、修改优先级与依赖关系

### 4.1 最小可用方案（MVP）

**只做这些修改即可实现元数据驱动**：

1. ✅ 服务端：`default.conf` 修改 `/tiles-metadata` 指向 `tiles.json`
2. ✅ 服务端：`docker-compose.yml` 挂载 tiles 目录
3. ✅ 前端：`useMapConfig.ts` 新建（支持 TileJSON 格式）
4. ✅ 前端：`constants.ts` 添加 `buildMapConfigFromMetadata` 函数
5. ⚠️ 前端：`useOLMap.ts` 集成元数据（可选，保持向后兼容）

### 4.2 完整方案（推荐）

在 MVP 基础上：

6. ✅ 前端：`useOLMap.ts` 完整集成
7. ✅ 前端：`index.tsx` 处理元数据状态

---

## 五、异常场景处理

### 5.1 设计原则

**核心原则**：任何异常都不应导致地图无法显示，必须自动降级到默认配置。

### 5.2 异常场景列表

| 异常场景 | 检测方式 | 处理方式 | 用户体验 |
|---------|---------|---------|---------|
| tiles.json 不存在 | HTTP 404 | 使用 DEFAULT_METADATA | 地图正常显示，使用默认配置 |
| tiles 目录未挂载 | Nginx 404/500 | 使用 DEFAULT_METADATA | 地图正常显示，使用默认配置 |
| tiles.json 格式错误 | JSON 解析失败 | 使用 DEFAULT_METADATA | 地图正常显示，使用默认配置 |
| tiles.json 字段缺失 | 字段验证失败 | 部分使用默认值 | 地图正常显示，部分字段用默认值 |
| 网络请求失败 | fetch catch | 使用 DEFAULT_METADATA | 地图正常显示，使用默认配置 |
| Nginx 未配置 location | HTTP 404 | 使用 DEFAULT_METADATA | 地图正常显示，使用默认配置 |

### 5.3 代码实现

**useMapConfig.ts 异常处理结构**：

```typescript
try {
  const response = await fetch('/tiles-metadata');

  if (response.ok) {
    const tilejson = await response.json();
    // 验证基本结构
    if (tilejson && (tilejson.bounds || tilejson.center)) {
      // 格式正确，正常解析
      setMetadata(tileJsonToMetadata(tilejson));
    } else {
      // 格式不完整，降级
      setMetadata(DEFAULT_METADATA);
      setIsUsingDefault(true);
    }
  } else {
    // HTTP 错误，降级
    setMetadata(DEFAULT_METADATA);
    setIsUsingDefault(true);
  }
} catch (err) {
  // 网络错误，降级
  setMetadata(DEFAULT_METADATA);
  setIsUsingDefault(true);
}
```

### 5.4 默认配置（DEFAULT_METADATA）

```typescript
export const DEFAULT_METADATA: MapMetadata = {
  name: 'Zambia Offline Map',
  region: 'Zambia',
  center: { lon: 28.221, lat: -14.607, zoom: 6 },
  bounds: { minLon: 22.0, maxLon: 34.0, minLat: -18.0, maxLat: -8.0 },
  zoom: { min: 6, max: 15, default: 6 },
  attribution: '© OpenStreetMap contributors',
  description: '赞比亚区域离线地图，包含 6-15 级瓦片',
};
```

### 5.5 调试支持

Hook 返回 `isUsingDefault` 标志，便于调试：

```typescript
const { metadata, isUsingDefault, error } = useMapConfig();

if (isUsingDefault) {
  console.warn('[Debug] Using default metadata:', error);
  // 可以在此添加监控上报
}
```

### 5.6 控制台日志

正常启动：
```
[MapConfig] ✓ Loaded TileJSON: My Map center: [27.85, -13.16] zoom: 6-15
```

异常降级：
```
[MapConfig] ⚠ Failed to fetch metadata (HTTP 404), using defaults
[MapConfig] ⚠ TileJSON format incomplete, using defaults
[MapConfig] ✗ Error fetching metadata: Network error
```

---

## 六、修改前后对比

### 5.1 当前架构（修改前）

```
前端硬编码配置：
  ├── constants.ts
  │   ├── defaultCenter: [28.221, -14.607]
  │   ├── minZoom: 1
  │   └── maxZoom: 18
  │
切换区域需要：
  1. 替换 tiles 目录
  2. 修改 constants.ts
  3. 重新构建前端
  4. 重启服务
```

### 5.2 优化后架构（修改后）

```
服务端元数据：
  ├── tiles/tiles.json (Maperitive 自动生成)
  │   ├── center: [lon, lat, zoom]
  │   ├── bounds: [minLon, minLat, maxLon, maxLat]
  │   ├── minzoom: 6
  │   └── maxzoom: 15
  │
前端动态加载：
  ├── useMapConfig() → 请求 /tiles-metadata
  ├── TileJSON → MapMetadata 转换
  ├── 降级到 DEFAULT_METADATA
  └── 自动应用配置
  │
切换区域只需要：
  1. 替换 tiles 目录（含新的 tiles.json）
  2. 重启服务（docker-compose restart web）
```

---

## 七、向后兼容性分析

### 7.1 兼容性保证

| 场景 | 当前行为 | 修改后行为 | 是否兼容 |
|------|---------|-----------|---------|
| 正常启动 | 使用硬编码配置 | 使用 TileJSON，失败则降级 | ✅ 兼容 |
| tiles.json 缺失 | - | 降级到 DEFAULT_METADATA | ✅ 兼容 |
| tiles.json 加载失败 | - | 降级到 DEFAULT_METADATA | ✅ 兼容 |
| 网络错误 | 使用在线 OSM | 使用在线 OSM | ✅ 兼容 |
| 传入 props center/zoom | 覆盖默认值 | 覆盖元数据值 | ✅ 兼容 |

### 7.2 无需修改的部分

以下配置已经正确，**无需修改**：

- ✅ `.env.development` - 瓦片代理配置已就位
- ✅ `.env.production` - 相对路径配置正确
- ✅ `vite.config.ts` - `/tiles` 代理已配置
- ✅ `index.tsx` - 组件接口保持不变
- ✅ `MapControls.tsx` - 缩放控制无需修改
- ✅ `MapPopup.tsx` - 悬浮提示无需修改
- ✅ `MapStatsPanel.tsx` - 统计面板无需修改

---

## 八、实施步骤

### 8.1 服务端部署（10 分钟）

```bash
# 1. 备份当前配置
cd goomc/deployments/docker/
cp default.conf default.conf.bak

# 2. 修改 default.conf
#    - 修改 /tiles-metadata 的 alias 路径为 tiles.json

# 3. 修改 docker-compose.yml
#    - 添加 tiles 目录挂载

# 4. 重启服务
docker-compose restart web

# 5. 验证
curl -I http://localhost:8081/tiles-metadata
curl http://localhost:8081/tiles-metadata | jq
# 应该看到 TileJSON 格式的内容
```

### 8.2 前端部署（30 分钟）

```bash
# 1. 创建 useMapConfig.ts
#    （复制上面的完整代码）

# 2. 修改 constants.ts
#    - 添加 buildMapConfigFromMetadata 函数
#    - 导入 MapMetadata 类型

# 3. 修改 useOLMap.ts
#    - 集成 useMapConfig Hook
#    - 使用元数据初始化地图

# 4. 类型检查
cd goomc/omcmb/webcode
npm run typecheck

# 5. 开发环境测试
npm run dev
# 打开浏览器，检查控制台日志
```

---

## 九、验证清单

### 9.1 服务端验证

- [ ] `default.conf` 的 `/tiles-metadata` 指向 `tiles.json`
- [ ] `docker-compose.yml` 包含 tiles 目录挂载
- [ ] `tiles/tiles.json` 文件存在且格式正确
- [ ] `curl http://localhost:8081/tiles-metadata` 返回 200 和正确 TileJSON
- [ ] `curl http://localhost:8081/tiles/12/2300/2144.png` 返回瓦片图片

### 9.2 前端验证

- [ ] `useMapConfig.ts` 文件存在
- [ ] `npm run typecheck` 无错误
- [ ] 浏览器控制台显示 `[MapConfig] Loaded TileJSON: My Map`
- [ ] 地图正常加载，中心点在预期位置
- [ ] 缩放级别在预期范围内
- [ ] TileJSON 加载失败时降级到默认配置

---

## 十、风险评估

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| tiles.json 加载失败 | 低 | 中 | 降级到 DEFAULT_METADATA |
| tiles 目录未挂载 | 中 | 高 | docker-compose 验证脚本 |
| TileJSON 格式不匹配 | 低 | 中 | 转换函数容错处理 |
| 向后兼容性破坏 | 低 | 高 | 保持 props 覆盖机制 |

---

## 十一、回滚方案

如果修改后出现问题，可以快速回滚：

```bash
# 服务端回滚
cd goomc/deployments/docker/
cp default.conf.bak default.conf
docker-compose restart web

# 前端回滚
git checkout src/components/GISMap/useMapConfig.ts
git checkout src/components/GISMap/useOLMap.ts
git checkout src/components/GISMap/constants.ts
```

---

## 十二、变更记录

| 版本 | 日期 | 修改内容 |
|------|------|---------|
| v1.0 | 2026-05-22 | 初始版本，使用 metadata.json |
| v2.0 | 2026-05-22 | 改用 Maperitive 自动生成的 tiles.json，简化部署流程 |
| v2.1 | 2026-05-22 | 添加完善的异常场景处理和降级方案，确保地图始终可用 |

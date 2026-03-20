# 通用基站 GIS 地图功能设计方案

> **文档版本**：v1.1
> **创建日期**：2026-03-20
> **更新日期**：2026-03-20
> **适用场景**：OMC 系统基站地理分布展示与管理

---

## 一、需求分析

### 1.1 业务场景

| 场景 | 描述 | 用户 | 核心诉求 |
|------|------|------|----------|
| 全网概览 | 查看全国/全省基站分布 | 网管人员 | 快速了解网络覆盖 |
| 状态监控 | 实时监控设备在线/离线状态 | 运维人员 | 及时发现异常设备 |
| 故障定位 | 快速定位故障设备位置 | 故障处理人员 | 缩短故障处理时间 |
| 设备组筛选 | 按设备组查看特定组下设备 | 区域负责人 | 聚焦管辖范围 |
| 容量规划 | 分析设备密度分布 | 规划人员 | 辅助网络规划决策 |
| 巡检导航 | 为现场人员提供导航 | 外勤人员 | 快速到达设备现场 |

### 1.2 功能范围

**包含**：
- ✅ 设备地理位置展示（经纬度定位）
- ✅ 设备状态可视化（颜色/图标区分）
- ✅ 多级缩放与平移
- ✅ 设备组筛选（树形结构）
- ✅ 状态筛选
- ✅ 关键词搜索
- ✅ 设备详情查看
- ✅ 区域统计面板
- ✅ 高密度区域聚合显示

**不包含**：
- ❌ 实时轨迹追踪（非GPS定位场景）
- ❌ 复杂GIS分析（地形分析、覆盖预测）
- ❌ 室内定位导航

### 1.3 约束条件

| 约束类型 | 具体要求 |
|----------|----------|
| 部署环境 | 支持内网部署，可能无法访问外网地图服务 |
| 数据规模 | 10万+ 基站，预留100万扩展能力 |
| 响应性能 | 首屏加载 < 3s，地图渲染 < 1s |
| 浏览器兼容 | Chrome 90+、Edge 90+、Firefox 88+ |
| 数据安全 | 基站坐标数据不泄露，需权限控制 |

---

## 二、技术选型

### 2.1 地图技术方案对比

| 方案 | 优点 | 缺点 | 内网支持 | 推荐度 |
|------|------|------|----------|--------|
| **简化SVG地图** | 轻量无依赖、加载快、完全离线 | 交互有限、精度低、无POI | ✅ 完美 | ⭐⭐⭐ |
| **Leaflet + 离线瓦片** | 开源免费、插件丰富、可定制 | 需准备瓦片数据、配置复杂 | ✅ 需配置 | ⭐⭐⭐⭐ |
| **高德/百度地图** | 功能强大、POI丰富、导航支持 | 需外网、有授权限制、数据出境风险 | ❌ 不支持 | ⭐⭐ |
| **Mapbox GL** | 矢量切片、效果炫酷、可定制 | 收费、需外网 | ❌ 不支持 | ⭐⭐ |
| **OpenLayers** | 功能全面、专业GIS能力、支持离线瓦片 | 体积较大(~500KB)、学习曲线适中 | ✅ 完美 | ⭐⭐⭐⭐⭐ |

### 2.2 选定方案：OpenLayers

**选择理由**：
1. **专业GIS能力**：支持多种地图源、矢量图层、复杂交互
2. **离线支持**：可部署离线瓦片，完全内网运行
3. **功能丰富**：聚合、热力图、绘制、测量等专业功能
4. **社区活跃**：文档完善，生态丰富
5. **与现有系统集成**：项目已有 OpenLayers 使用经验

### 2.3 技术架构

```
┌───────────────────────────────────────────────────────────┐
│                    应用层 (React 组件)                     │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  GISMap (主组件)                                     │  │
│  │  ├── MapControls (缩放/全屏)                        │  │
│  │  ├── MapMarker (设备标记)                           │  │
│  │  ├── MapCluster (聚合圈)                            │  │
│  │  └── MapPopup (详情弹窗)                            │  │
│  └─────────────────────────────────────────────────────┘  │
├───────────────────────────────────────────────────────────┤
│                    地图引擎 (OpenLayers)                   │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  ol/Map → ol/View → ol/layer/Tile (底图)            │  │
│  │           → ol/layer/Vector (设备标记)              │  │
│  │           → ol/source/Cluster (聚合)                │  │
│  └─────────────────────────────────────────────────────┘  │
├───────────────────────────────────────────────────────────┤
│                    数据层 (API + React Query)              │
└───────────────────────────────────────────────────────────┘
```

### 2.4 OpenLayers 核心依赖

```json
{
  "dependencies": {
    "ol": "^10.0.0",
    "@types/ol": "^8.0.0"
  }
}
```

### 2.5 地图源配置

| 模式 | 地图源 | 说明 |
|------|--------|------|
| 在线模式 | OpenStreetMap | 默认，无需配置 |
| 离线模式 | 自定义瓦片服务 | 需部署离线瓦片 |

```typescript
// 环境变量配置
VITE_MAP_TILE_URL=/api/v1/tiles/{z}/{x}/{y}.png  // 离线瓦片地址
VITE_MAP_DEFAULT_CENTER=104.0,35.0               // 默认中心点
VITE_MAP_DEFAULT_ZOOM=4                          // 默认缩放级别
```

---

## 三、功能设计

### 3.1 功能架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        GIS 地图页面                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      地图主体区域 (OpenLayers)            │   │
│  │  ┌────────────────────────────────────────────────────┐  │   │
│  │  │                                                    │  │   │
│  │  │     [设备标记]  [聚合圈]  [连线(可选)]               │  │   │
│  │  │                                                    │  │   │
│  │  │              瓦片地图底图 (OSM/离线瓦片)             │  │   │
│  │  │                                                    │  │   │
│  │  └────────────────────────────────────────────────────┘  │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐                   │   │
│  │  │ 缩放控制 │  │ 图层切换 │  │ 全屏按钮 │                  │   │
│  │  └─────────┘  └─────────┘  └─────────┘                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────────┐ │
│  │ 筛选面板    │  │ 统计面板    │  │ 详情卡片 (选中时显示)       │ │
│  │ (左侧)     │  │ (右下浮动)  │  │ (右上浮动)                  │ │
│  │ 设备组树    │  │ 状态统计    │  │ 设备信息                    │ │
│  └────────────┘  └────────────┘  └────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 功能清单

| 编号 | 功能模块 | 功能名称 | 描述 | 优先级 |
|------|----------|----------|------|--------|
| M01 | 地图展示 | 底图渲染 | 使用 OpenLayers 渲染瓦片地图 | P0 |
| M02 | 地图展示 | 缩放控制 | 支持+/-按钮和滚轮缩放 | P0 |
| M03 | 地图展示 | 平移拖拽 | 支持鼠标拖拽平移地图 | P0 |
| M04 | 地图展示 | 全屏模式 | 支持全屏查看 | P1 |
| M05 | 设备标记 | 状态标记 | 按状态显示不同颜色标记 | P0 |
| M06 | 设备标记 | 聚合显示 | 高密度区域聚合为圆圈 | P0 |
| M07 | 设备标记 | 悬停提示 | 悬停显示设备名称 | P0 |
| M08 | 设备标记 | 点击详情 | 点击显示设备详情卡片 | P0 |
| M09 | 筛选功能 | 设备组筛选 | 按设备组树形筛选 | P0 |
| M10 | 筛选功能 | 状态筛选 | 按在线/离线/告警筛选 | P0 |
| M11 | 筛选功能 | 类型筛选 | 按设备类型筛选 | P1 |
| M12 | 筛选功能 | 关键词搜索 | 按名称/序列号搜索 | P0 |
| M13 | 统计面板 | 状态统计 | 显示各状态设备数量 | P0 |
| M14 | 统计面板 | 设备组统计 | 显示当前设备组内设备数 | P1 |
| M15 | 详情卡片 | 设备信息 | 显示选中设备详细信息 | P0 |
| M16 | 详情卡片 | 快捷操作 | 跳转详情、查看告警等 | P1 |

### 3.3 交互设计

#### 3.3.1 交互流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户交互流程                             │
└─────────────────────────────────────────────────────────────────┘

页面加载
    │
    ├──► 初始化 OpenLayers 地图
    │
    ├──► 并行请求 ──┬──► 获取设备组树
    │              │
    │              ├──► 获取设备列表/聚合数据
    │              │
    │              └──► 获取统计数据
    │
    └──► 渲染设备标记

用户操作分支：

[设备组筛选] ──► 更新 groupId ──► 重新请求设备数据 ──► 更新地图标记

[状态筛选] ──► 更新 status ──► 重新请求设备数据 ──► 更新地图标记

[缩放操作] ──► 判断缩放级别 ──┬──► 大范围(zoom<12)：显示聚合数据
                            │
                            └──► 小范围(zoom≥12)：显示独立标记

[点击标记] ──┬──► 独立标记：显示设备详情卡片
            │
            └──► 聚合圈：放大地图，展开聚合

[悬停标记] ──► 显示 Overlay（设备名称、状态）

[拖拽平移] ──► 移动完成后，请求新区域的聚合数据
```

#### 3.3.2 聚合显示规则（OpenLayers Cluster）

| 缩放级别 | 显示方式 | 聚合距离 | 说明 |
|----------|----------|----------|------|
| zoom 1-6 (全国) | 省级聚合 | 100px | 显示各省设备总数 |
| zoom 7-10 (省级) | 市级聚合 | 60px | 显示各市设备总数 |
| zoom 11-12 (市级) | 网格聚合 | 40px | 按网格聚合 |
| zoom 13+ (详细) | 独立标记 | 0 | 显示每个设备独立标记 |

#### 3.3.3 设备状态可视化规则

| 设备状态 | 标记颜色 | 标记样式 | 脉冲效果 |
|----------|----------|----------|----------|
| 在线 (online) | 🟢 #52C41A | 实心圆点 | 无 |
| 离线 (offline) | ⚫ #8C8C8C | 空心圆点 | 无 |
| 告警 (alarm) | 🟠 #FA8C16 | 实心圆点 | 橙色脉冲 |
| 故障 (fault) | 🔴 #F5222D | 实心圆点 | 红色脉冲 |
| 维护中 (maintenance) | 🔵 #1890FF | 实心圆点 | 无 |

#### 3.3.4 设备组与设备关系

```
┌─────────────────────────────────────────────────────────────────┐
│                      设备组树形结构                              │
└─────────────────────────────────────────────────────────────────┘

设备组 (DeviceGroup)                    设备 (Device)
┌─────────────────────┐                ┌─────────────────────┐
│ 全国 (root)          │                │ id: string          │
│  ├── 北京            │                │ name: string        │
│  │   ├── 朝阳区      │ ──包含──►      │ sn: string          │
│  │   └── 海淀区      │                │ status: DeviceStatus│
│  ├── 上海            │                │ groupId: string     │
│  │   ├── 浦东新区    │                │ longitude: number   │
│  │   └── 静安区      │                │ latitude: number    │
│  └── ...            │                │ ...                 │
└─────────────────────┘                └─────────────────────┘

业务规则：
1. 设备必须属于某个设备组
2. 设备组支持多级嵌套（最多5级）
3. 筛选设备组时，显示该组及其子组下的所有设备
```

---

## 四、数据模型设计

### 4.1 实体定义

```typescript
/**
 * 设备地理信息
 */
interface DeviceGeo {
  /** 设备ID */
  id: string;
  /** 设备名称 */
  name: string;
  /** 设备序列号 */
  sn: string;
  /** 经度 (东经为正) */
  longitude: number;
  /** 纬度 (北纬为正) */
  latitude: number;
  /** 设备状态 */
  status: DeviceStatus;
  /** 设备类型 */
  type?: DeviceType;
  /** 所属设备组ID */
  groupId: string;
  /** 设备组名称（冗余，便于展示） */
  groupName?: string;
  /** 详细地址 */
  address?: string;
  /** 告警数量 */
  alarmCount?: number;
}

/**
 * 设备状态枚举
 */
type DeviceStatus =
  | 'online'      // 在线
  | 'offline'     // 离线
  | 'alarm'       // 告警
  | 'fault'       // 故障
  | 'maintenance' // 维护中
  ;

/**
 * 设备类型枚举
 */
type DeviceType =
  | 'macro'   // 宏站
  | 'small'   // 小基站
  | 'pico'    // 皮站
  | 'femto'   // 飞站
  | 'rru'     // RRU
  ;

/**
 * 设备组（树形结构）
 */
interface DeviceGroup {
  /** 设备组ID */
  id: string;
  /** 设备组名称 */
  name: string;
  /** 父设备组ID */
  parentId: string | null;
  /** 层级 (1-5) */
  level: number;
  /** 子设备组 */
  children?: DeviceGroup[];
  /** 设备数量（含子组） */
  deviceCount?: number;
  /** 运营商 */
  carrier?: string;
  /** 描述 */
  description?: string;
}

/**
 * 设备聚合项
 */
interface DeviceCluster {
  /** 聚合ID */
  id: string;
  /** 聚合中心点经度 */
  longitude: number;
  /** 聚合中心点纬度 */
  latitude: number;
  /** 聚合内设备数量 */
  count: number;
  /** 各状态数量统计 */
  statusCount: Record<DeviceStatus, number>;
  /** 聚合范围（网格） */
  bounds?: {
    minLng: number;
    maxLng: number;
    minLat: number;
    maxLat: number;
  };
}

/**
 * 地图统计数据
 */
interface MapStats {
  /** 总设备数 */
  total: number;
  /** 各状态数量 */
  statusCount: Record<DeviceStatus, number>;
  /** 各类型数量 */
  typeCount?: Record<DeviceType, number>;
  /** 当前视图内设备数 */
  viewportCount?: number;
}

/**
 * 地图视图状态
 */
interface MapViewport {
  /** 中心经度 */
  centerLng: number;
  /** 中心纬度 */
  centerLat: number;
  /** 缩放级别 */
  zoom: number;
  /** 边界范围 */
  bounds: {
    minLng: number;
    maxLng: number;
    minLat: number;
    maxLat: number;
  };
}

/**
 * 地图筛选参数
 */
interface MapFilterParams {
  /** 设备组ID（筛选该组及其子组下所有设备） */
  groupId?: string;
  /** 状态筛选 */
  status?: DeviceStatus[];
  /** 类型筛选 */
  type?: DeviceType[];
  /** 搜索关键词（名称/序列号） */
  keyword?: string;
}

/**
 * 地图配置
 */
interface MapConfig {
  /** 默认中心点 */
  defaultCenter: [number, number]; // [lng, lat]
  /** 默认缩放级别 */
  defaultZoom: number;
  /** 最小缩放级别 */
  minZoom: number;
  /** 最大缩放级别 */
  maxZoom: number;
  /** 瓦片服务地址 */
  tileUrl?: string;
}
```

### 4.2 状态样式配置

```typescript
/**
 * 设备状态样式配置
 */
const DEVICE_STATUS_CONFIG: Record<DeviceStatus, {
  color: string;
  bgColor: string;
  borderColor: string;
  text: string;
  i18nKey: string;
  pulse: boolean;
}> = {
  online: {
    color: '#52C41A',
    bgColor: '#F6FFED',
    borderColor: '#B7EB8F',
    text: '在线',
    i18nKey: 'status.online',
    pulse: false,
  },
  offline: {
    color: '#8C8C8C',
    bgColor: '#F5F5F5',
    borderColor: '#D9D9D9',
    text: '离线',
    i18nKey: 'status.offline',
    pulse: false,
  },
  alarm: {
    color: '#FA8C16',
    bgColor: '#FFF7E6',
    borderColor: '#FFD591',
    text: '告警',
    i18nKey: 'status.alarm',
    pulse: true,
  },
  fault: {
    color: '#F5222D',
    bgColor: '#FFF1F0',
    borderColor: '#FFA39E',
    text: '故障',
    i18nKey: 'status.fault',
    pulse: true,
  },
  maintenance: {
    color: '#1890FF',
    bgColor: '#E6F7FF',
    borderColor: '#91D5FF',
    text: '维护中',
    i18nKey: 'status.maintenance',
    pulse: false,
  },
};
```

---

## 五、API 接口设计

### 5.1 接口清单

| API | 方法 | 路径 | 用途 |
|-----|------|------|------|
| 获取设备组树 | GET | /api/v1/groups | 获取设备组树形结构 |
| 获取设备地理列表 | GET | /api/v1/devices/geo | 获取设备列表（含坐标） |
| 获取聚合数据 | POST | /api/v1/devices/geo/aggregate | 按网格/区域聚合 |
| 获取地图统计 | GET | /api/v1/devices/geo/stats | 获取统计数据 |
| 获取设备详情 | GET | /api/v1/devices/:id | 获取单个设备详情 |

### 5.2 API 详细定义

#### API-001: 获取设备组树

```
GET /api/v1/groups

Response:
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "group-001",
        "name": "全国",
        "parent_id": null,
        "carrier": "cmcc",
        "description": "中国移动物联网",
        "sort_order": 1,
        "children": [
          {
            "id": "group-002",
            "name": "北京",
            "parent_id": "group-001",
            "children": [
              {
                "id": "group-003",
                "name": "朝阳区",
                "parent_id": "group-002"
              }
            ]
          }
        ],
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### API-002: 获取设备地理列表

```
GET /api/v1/devices/geo

Query Parameters:
┌─────────────┬────────┬────────┬─────────────────────────────────────────────┐
│ 参数名       │ 类型   │ 必填   │ 说明                                         │
├─────────────┼────────┼────────┼─────────────────────────────────────────────┤
│ group_id    │ string │ 否     │ 设备组ID（含子组）                            │
│ status      │ string │ 否     │ 状态，多个用逗号分隔                          │
│ type        │ string │ 否     │ 类型，多个用逗号分隔                          │
│ keyword     │ string │ 否     │ 搜索关键词（名称/序列号）                     │
│ bounds      │ string │ 否     │ 视图边界 "minLng,maxLng,minLat,maxLat"       │
│ page        │ number │ 否     │ 页码，默认1                                  │
│ page_size   │ number │ 否     │ 每页条数，默认1000                           │
└─────────────┴────────┴────────┴─────────────────────────────────────────────┘

Response:
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "device-001",
        "name": "北京朝阳设备01",
        "sn": "SN2024001",
        "longitude": 116.46,
        "latitude": 39.92,
        "status": "online",
        "type": "small",
        "group_id": "group-003",
        "group_name": "朝阳区",
        "address": "北京市朝阳区建国路88号",
        "alarm_count": 0
      }
    ],
    "total": 10000
  }
}
```

#### API-003: 获取聚合数据

```
POST /api/v1/devices/geo/aggregate

Request Body:
{
  "bounds": {
    "min_lng": 116.0,
    "max_lng": 117.0,
    "min_lat": 39.0,
    "max_lat": 40.0
  },
  "zoom": 8,
  "grid_size": 50,        // 网格大小（像素）
  "filters": {
    "group_id": "group-002",    // 设备组ID
    "status": ["online", "alarm"]
  }
}

Response:
{
  "code": 0,
  "data": {
    "clusters": [
      {
        "id": "cluster-001",
        "longitude": 116.45,
        "latitude": 39.91,
        "count": 15,
        "status_count": {
          "online": 12,
          "offline": 1,
          "alarm": 2,
          "fault": 0,
          "maintenance": 0
        },
        "bounds": {
          "min_lng": 116.40,
          "max_lng": 116.50,
          "min_lat": 39.88,
          "max_lat": 39.94
        }
      }
    ],
    "zoom_level": 8,
    "grid_size": 50
  }
}
```

#### API-004: 获取地图统计

```
GET /api/v1/devices/geo/stats

Query Parameters:
┌─────────────┬────────┬────────┬─────────────────────────────────────────────┐
│ 参数名       │ 类型   │ 必填   │ 说明                                         │
├─────────────┼────────┼────────┼─────────────────────────────────────────────┤
│ group_id    │ string │ 否     │ 设备组ID（含子组）                            │
│ bounds      │ string │ 否     │ 视图边界（仅统计视图内）                      │
└─────────────┴────────┴────────┴─────────────────────────────────────────────┘

Response:
{
  "code": 0,
  "data": {
    "total": 10000,
    "status_count": {
      "online": 8500,
      "offline": 800,
      "alarm": 500,
      "fault": 100,
      "maintenance": 100
    },
    "type_count": {
      "macro": 2000,
      "small": 6000,
      "pico": 1500,
      "femto": 500
    },
    "viewport_count": 150
  }
}
```

---

## 六、前端架构设计（OpenLayers）

### 6.1 组件架构

```
src/components/GISMap/
├── index.tsx                    # GISMap 主组件（对外入口）
├── core/
│   ├── OLMap.ts                 # OpenLayers 地图核心类
│   ├── layers/
│   │   ├── TileLayer.ts         # 瓦片底图层
│   │   ├── DeviceLayer.ts       # 设备标记层（Vector）
│   │   └── ClusterLayer.ts      # 聚合层
│   └── interactions/
│       ├── HoverInteraction.ts  # 悬停交互
│       └── ClickInteraction.ts  # 点击交互
├── components/
│   ├── MapMarker.tsx            # 设备标记样式组件
│   ├── MapCluster.tsx           # 聚合圆圈组件
│   ├── MapPopup.tsx             # 弹窗/详情卡片组件
│   ├── MapControls.tsx          # 地图控制按钮（缩放/全屏）
│   ├── MapStatsPanel.tsx        # 统计面板组件
│   ├── MapLegend.tsx            # 图例组件
│   └── GroupTree.tsx            # 设备组树形筛选组件
├── hooks/
│   ├── useOLMap.ts              # OpenLayers 地图初始化 Hook
│   ├── useDeviceLayer.ts        # 设备图层 Hook
│   ├── useClusterSource.ts      # 聚合数据源 Hook
│   ├── useMapGeoData.ts         # 地图数据 Hook
│   ├── useMapStats.ts           # 统计数据 Hook
│   └── useMapViewport.ts        # 视图状态 Hook
├── utils/
│   ├── featureUtils.ts          # OpenLayers Feature 工具
│   ├── styleUtils.ts            # 样式工具（颜色/图标）
│   └── geoUtils.ts              # 地理计算工具
├── constants.ts                 # 常量配置
├── types.ts                     # 类型定义
└── styles.module.css            # 样式文件

src/services/api/
└── mapApi.ts                    # 地图相关 API

src/hooks/api/
└── useMap.ts                    # 地图 React Query Hooks
```

### 6.2 OpenLayers 核心实现

```typescript
// core/OLMap.ts
import Map from 'ol/Map';
import View from 'ol/View';
import TileLayer from 'ol/layer/Tile';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Cluster from 'ol/source/Cluster';
import OSM from 'ol/source/OSM';
import XYZ from 'ol/source/XYZ';
import { fromLonLat, toLonLat } from 'ol/proj';
import { defaults as defaultControls } from 'ol/control';
import { Style, Circle, Fill, Stroke, Text } from 'ol/style';

export class OLMapCore {
  private map: Map | null = null;
  private deviceSource: VectorSource | null = null;
  private clusterSource: Cluster | null = null;

  /**
   * 初始化地图
   */
  init(container: HTMLElement, options: MapOptions): void {
    // 创建瓦片图层
    const tileLayer = new TileLayer({
      source: options.tileUrl
        ? new XYZ({ url: options.tileUrl })  // 离线瓦片
        : new OSM(),                          // OpenStreetMap
    });

    // 创建设备数据源
    this.deviceSource = new VectorSource();

    // 创建聚合数据源
    this.clusterSource = new Cluster({
      source: this.deviceSource,
      distance: options.clusterDistance || 40,
    });

    // 创建设备图层
    const deviceLayer = new VectorLayer({
      source: this.clusterSource,
      style: this.createClusterStyle.bind(this),
      zIndex: 10,
    });

    // 创建地图实例
    this.map = new Map({
      target: container,
      layers: [tileLayer, deviceLayer],
      view: new View({
        center: fromLonLat(options.center || [104.0, 35.0]),
        zoom: options.zoom || 4,
        minZoom: options.minZoom || 1,
        maxZoom: options.maxZoom || 18,
      }),
      controls: defaultControls({ zoom: false }),
    });

    this.bindEvents();
  }

  /**
   * 创建聚合样式
   */
  private createClusterStyle(feature: any): Style {
    const size = feature.get('features').length;

    if (size === 1) {
      // 单个设备标记
      const device = feature.get('features')[0];
      return this.createDeviceStyle(device);
    }

    // 聚合圈样式
    const radius = Math.max(15, Math.min(40, 10 + size / 10));
    return new Style({
      image: new Circle({
        radius,
        fill: new Fill({ color: 'rgba(24, 144, 255, 0.8)' }),
        stroke: new Stroke({ color: '#fff', width: 2 }),
      }),
      text: new Text({
        text: size.toString(),
        fill: new Fill({ color: '#fff' }),
        font: 'bold 12px sans-serif',
      }),
    });
  }

  /**
   * 创建设备标记样式
   */
  private createDeviceStyle(device: any): Style {
    const status = device.get('status');
    const config = DEVICE_STATUS_CONFIG[status];

    return new Style({
      image: new Circle({
        radius: 8,
        fill: new Fill({ color: config.color }),
        stroke: new Stroke({ color: '#fff', width: 2 }),
      }),
    });
  }

  /**
   * 更新设备数据
   */
  updateDevices(devices: DeviceGeo[]): void {
    if (!this.deviceSource) return;

    // 清除现有数据
    this.deviceSource.clear();

    // 添加新数据
    const features = devices.map(device => {
      const feature = new Feature({
        geometry: new Point(fromLonLat([device.longitude, device.latitude])),
        ...device,
      });
      feature.setId(device.id);
      return feature;
    });

    this.deviceSource.addFeatures(features);
  }

  /**
   * 获取当前视图状态
   */
  getViewport(): MapViewport {
    const view = this.map!.getView();
    const center = toLonLat(view.getCenter()!);
    const extent = view.calculateExtent(this.map!.getSize());

    return {
      centerLng: center[0],
      centerLat: center[1],
      zoom: view.getZoom()!,
      bounds: {
        minLng: toLonLat([extent[0], extent[1]])[0],
        maxLng: toLonLat([extent[2], extent[3]])[0],
        minLat: toLonLat([extent[0], extent[1]])[1],
        maxLat: toLonLat([extent[2], extent[3]])[1],
      },
    };
  }

  /**
   * 销毁地图
   */
  destroy(): void {
    this.map?.setTarget(undefined);
    this.map = null;
    this.deviceSource = null;
    this.clusterSource = null;
  }
}
```

### 6.3 核心组件接口

```typescript
/**
 * GISMap 主组件 Props
 */
interface GISMapProps {
  /** 设备数据列表 */
  devices: MapDevice[];
  /** 地图高度 */
  height?: string | number;
  /** 默认中心点 [lng, lat] */
  defaultCenter?: [number, number];
  /** 默认缩放级别 */
  defaultZoom?: number;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 是否显示统计面板 */
  showStats?: boolean;
  /** 是否显示图例 */
  showLegend?: boolean;
  /** 是否显示控制按钮 */
  showControls?: boolean;
  /** 自定义样式 */
  className?: string;
}

/**
 * MapDevice - 地图设备标记数据
 */
interface MapDevice {
  /** 纬度 */
  lat: number;
  /** 经度 */
  lng: number;
  /** 设备名称 */
  name: string;
  /** 设备状态 */
  status: DeviceStatus;
  /** 序列号/ID */
  sn: string;
  /** 设备组ID */
  groupId?: string;
  /** 附加数据 */
  data?: Record<string, any>;
}

/**
 * GroupTree 组件 Props
 */
interface GroupTreeProps {
  /** 选中的设备组ID */
  selectedGroupId?: string;
  /** 选中变化回调 */
  onSelect: (groupId: string) => void;
  /** 是否显示搜索 */
  showSearch?: boolean;
}
```

### 6.4 React Query Hooks

```typescript
// hooks/useMapGeoData.ts
import { useQuery } from '@tanstack/react-query';
import { mapApi } from '@/services/api/mapApi';

interface UseMapGeoDataParams {
  groupId?: string;
  status?: DeviceStatus[];
  keyword?: string;
  bounds?: MapBounds;
  enabled?: boolean;
}

export function useMapGeoData(params: UseMapGeoDataParams) {
  return useQuery({
    queryKey: ['map', 'geo', params],
    queryFn: () => mapApi.getDevicesGeo(params),
    staleTime: 5 * 60 * 1000, // 5分钟
    enabled: params.enabled !== false,
  });
}

// hooks/useMapAggregation.ts
export function useMapAggregation(params: {
  bounds: MapBounds;
  zoom: number;
  filters?: MapFilterParams;
}) {
  return useQuery({
    queryKey: ['map', 'aggregation', params],
    queryFn: () => mapApi.getAggregation(params),
    staleTime: 2 * 60 * 1000, // 2分钟
    enabled: params.zoom < 12, // 仅在缩放级别较小时请求
  });
}

// hooks/useMapStats.ts
export function useMapStats(params?: { groupId?: string }) {
  return useQuery({
    queryKey: ['map', 'stats', params],
    queryFn: () => mapApi.getStats(params),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 60 * 1000, // 每分钟刷新
  });
}

// hooks/useDeviceGroups.ts
export function useDeviceGroups() {
  return useQuery({
    queryKey: ['device-groups', 'tree'],
    queryFn: () => mapApi.getDeviceGroups(),
    staleTime: 10 * 60 * 1000, // 10分钟
  });
}
```

### 6.5 API 服务层

```typescript
// services/api/mapApi.ts
import http from '../http';
import type { DeviceGeo, DeviceCluster, MapStats, DeviceGroup, MapFilterParams } from '@/types/map';

export const mapApi = {
  /**
   * 获取设备组树
   */
  async getDeviceGroups(): Promise<DeviceGroup[]> {
    const { data } = await http.get<{ items: BackendDeviceGroup[] }>('/groups');
    return (data.items || []).map(mapBackendDeviceGroup);
  },

  /**
   * 获取设备地理数据
   */
  async getDevicesGeo(params: {
    groupId?: string;
    status?: string[];
    keyword?: string;
    bounds?: string;
  }): Promise<{ items: DeviceGeo[]; total: number }> {
    const { data } = await http.get('/devices/geo', {
      params: {
        group_id: params.groupId,
        status: params.status?.join(','),
        keyword: params.keyword,
        bounds: params.bounds,
      },
    });
    return {
      items: data.items.map(mapBackendDeviceGeo),
      total: data.total,
    };
  },

  /**
   * 获取聚合数据
   */
  async getAggregation(params: {
    bounds: MapBounds;
    zoom: number;
    gridSize?: number;
    filters?: MapFilterParams;
  }): Promise<{ clusters: DeviceCluster[] }> {
    const { data } = await http.post('/devices/geo/aggregate', {
      bounds: {
        min_lng: params.bounds.minLng,
        max_lng: params.bounds.maxLng,
        min_lat: params.bounds.minLat,
        max_lat: params.bounds.maxLat,
      },
      zoom: params.zoom,
      grid_size: params.gridSize || 50,
      filters: {
        group_id: params.filters?.groupId,
        status: params.filters?.status,
      },
    });
    return {
      clusters: data.clusters.map(mapBackendCluster),
    };
  },

  /**
   * 获取统计数据
   */
  async getStats(params?: { groupId?: string; bounds?: string }): Promise<MapStats> {
    const { data } = await http.get('/devices/geo/stats', {
      params: {
        group_id: params?.groupId,
        bounds: params?.bounds,
      },
    });
    return mapBackendStats(data);
  },
};
```

---

## 七、性能优化策略

### 7.1 数据加载优化

| 策略 | 实现方式 | 效果 |
|------|----------|------|
| **聚合显示** | 大范围使用聚合API，减少标记点数量 | 渲染点从10万→1000 |
| **视口裁剪** | 只请求和渲染当前视口内的数据 | 减少不必要的数据传输 |
| **分块加载** | 滚动/平移时增量加载新区域数据 | 首屏加载更快 |
| **防抖请求** | 平移/缩放停止后才请求数据 | 减少请求频率 |

### 7.2 渲染优化

| 策略 | 实现方式 | 效果 |
|------|----------|------|
| **虚拟标记** | 使用 Canvas 渲染而非 DOM 元素 | 大数据量不卡顿 |
| **简化图形** | 远距离时使用简单圆形代替复杂图标 | 减少绘制开销 |
| **按需渲染** | 仅渲染可见区域内的标记 | 提升帧率 |
| **CSS 硬件加速** | 使用 transform 而非 left/top 定位 | 动画更流畅 |

### 7.3 缓存策略

```typescript
// React Query 缓存配置
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,  // 5分钟内数据视为新鲜
      cacheTime: 30 * 60 * 1000, // 缓存保留30分钟
      refetchOnWindowFocus: false,
    },
  },
});

// 地图数据特殊缓存
const MAP_QUERY_OPTIONS = {
  staleTime: 2 * 60 * 1000,  // 聚合数据2分钟刷新
  cacheTime: 10 * 60 * 1000,
};
```

### 7.4 性能指标

| 指标 | 目标值 | 测量方式 |
|------|--------|----------|
| 首屏加载时间 | < 3s | Performance API |
| 地图渲染时间 | < 1s | console.time |
| 缩放/平移响应 | < 100ms | 交互延迟测量 |
| 1000标记渲染 | < 500ms | 基准测试 |
| 内存占用 | < 200MB | DevTools Memory |

---

## 八、离线地图方案（OpenLayers）

### 8.1 在线模式（默认）

使用 OpenStreetMap 作为底图，无需配置：

```typescript
import OSM from 'ol/source/OSM';

const tileLayer = new TileLayer({
  source: new OSM(),
});
```

### 8.2 离线模式

**部署结构**：

```
server/
├── tiles/                 # 瓦片数据目录
│   ├── {z}/              # 缩放级别 (1-12)
│   │   ├── {x}/          # X坐标
│   │   │   └── {y}.png   # 瓦片图片
│   │   └── ...
│   └── ...
└── nginx.conf            # Nginx 配置，提供瓦片服务

前端配置：
VITE_MAP_TILE_URL=/api/v1/tiles/{z}/{x}/{y}.png
```

**OpenLayers 离线瓦片配置**：

```typescript
import XYZ from 'ol/source/XYZ';

const tileLayer = new TileLayer({
  source: new XYZ({
    url: import.meta.env.VITE_MAP_TILE_URL || 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png',
  }),
});
```

**瓦片准备**：
1. 下载 OpenStreetMap 或 GeoQ 中国瓦片
2. 使用 `renderdog` 生成自定义样式
3. 仅下载中国区域，级别1-12，约 5-10GB
4. 部署到内部 Nginx 服务器

### 8.3 瓦片服务 API

```
GET /api/v1/tiles/{z}/{x}/{y}.png

Response: image/png (瓦片图片)
```

Nginx 配置示例：

```nginx
server {
    listen 80;
    server_name tiles.internal;

    location /tiles/ {
        alias /data/tiles/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### 8.4 数据安全

| 安全措施 | 说明 |
|----------|------|
| 坐标加密 | 使用国测局坐标（GCJ-02）或自定义偏移 |
| 权限控制 | API 接口验证用户权限，敏感区域不可见 |
| 数据脱敏 | 导出数据时去除精确坐标 |
| 访问日志 | 记录地图数据访问，审计追溯 |

---

## 九、扩展能力（OpenLayers）

### 9.1 OpenLayers 插件扩展

```typescript
// 热力图
import Heatmap from 'ol/layer/Heatmap';

const heatmapLayer = new Heatmap({
  source: deviceSource,
  radius: 10,
  blur: 15,
});

// 绘制工具（框选）
import Draw from 'ol/interaction/Draw';
import { createBox } from 'ol/interaction/Draw';

const drawInteraction = new Draw({
  type: 'Circle',
  geometryFunction: createBox(),
});

// 测量工具
import { getLength, getArea } from 'ol/sphere';

function measureDistance(line: LineString): number {
  return getLength(line, { projection: 'EPSG:3857' });
}
```

### 9.2 自定义主题

```typescript
// themes/dark.ts
export const darkTheme: MapTheme = {
  background: '#1a1a2e',
  landColor: '#16213e',
  borderColor: '#0f3460',
  markerColors: {
    online: '#00ff88',
    offline: '#666666',
    alarm: '#ffaa00',
    fault: '#ff4444',
  },
};

// CSS 变量应用
:root {
  --map-bg: #ffffff;
  --map-marker-online: #52C41A;
  --map-marker-offline: #8C8C8C;
  --map-marker-alarm: #FA8C16;
  --map-marker-fault: #F5222D;
}

[data-theme='dark'] {
  --map-bg: #1a1a2e;
  --map-marker-online: #00ff88;
  --map-marker-offline: #666666;
  --map-marker-alarm: #ffaa00;
  --map-marker-fault: #ff4444;
}
```

### 9.3 事件扩展

```typescript
interface MapEvents {
  // 基础事件
  'click:device': (device: MapDevice) => void;
  'click:cluster': (cluster: DeviceCluster) => void;
  'viewport:change': (viewport: MapViewport) => void;

  // 扩展事件
  'selection:box': (bounds: MapBounds, devices: MapDevice[]) => void;
  'device:hover': (device: MapDevice | null) => void;
  'zoom:change': (zoom: number) => void;
}

// 使用示例
<GISMap
  onClickDevice={(device) => console.log('点击:', device)}
  onSelectionBox={(bounds, devices) => handleBoxSelection(bounds, devices)}
/>
```

### 9.4 OpenLayers 高级功能

| 功能 | OpenLayers 实现 | 用途 |
|------|-----------------|------|
| 框选设备 | Draw + Box | 批量选择设备 |
| 测距 | Draw + LineString | 测量两点距离 |
| 路径绘制 | Draw + LineString | 规划巡检路线 |
| 区域标注 | Draw + Polygon | 标注重点区域 |
| 热力图 | Heatmap Layer | 设备密度可视化 |
| 轨迹回放 | Vector + Animation | 设备移动轨迹 |

---

## 十、实现检查清单

### 10.1 开发前

- [ ] 确认部署环境（内网/外网）
- [ ] 确定地图源方案（在线OSM/离线瓦片）
- [ ] 确认设备数据接口是否就绪
- [ ] 准备测试数据（设备坐标）
- [ ] 安装 OpenLayers 依赖 (`npm install ol`)

### 10.2 开发中

- [ ] 实现 OpenLayers 地图核心类 (OLMapCore)
- [ ] 实现设备标记图层
- [ ] 实现聚合显示逻辑
- [ ] 实现设备组树形筛选组件
- [ ] 实现统计面板
- [ ] 实现筛选联动
- [ ] 添加国际化支持
- [ ] 处理加载和错误状态

### 10.3 开发后

- [ ] TypeScript 检查通过
- [ ] ESLint 检查通过
- [ ] 性能测试（1000+ 标记）
- [ ] 浏览器兼容性测试
- [ ] 响应式布局测试
- [ ] 代码审查

---

## 附录

### A. 相关文件

| 文件 | 路径 |
|------|------|
| 地图组件 | src/components/GISMap/ |
| 地图API | src/services/api/mapApi.ts |
| 地图Hooks | src/hooks/api/useMap.ts |
| 类型定义 | src/types/map.ts |
| 设备组树组件 | src/components/GISMap/components/GroupTree.tsx |
| 页面使用 | src/pages/topology/GISMapView/ |

### B. 参考资料

- [OpenLayers 官方文档](https://openlayers.org/en/latest/apidoc/)
- [OpenLayers Examples](https://openlayers.org/en/latest/examples/)
- [React Query 文档](https://tanstack.com/query/latest)
- [坐标系统说明](https://openlayers.org/en/latest/doc/tutorials/concepts.html#projections)

### C. 变更记录

| 版本 | 日期 | 修改人 | 修改内容 |
|------|------|--------|----------|
| v1.0 | 2026-03-20 | - | 初始版本 |
| v1.1 | 2026-03-20 | - | 技术方案改为 OpenLayers，筛选改为设备组 |

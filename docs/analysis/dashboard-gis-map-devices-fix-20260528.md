# 仪表板 GIS 地图设备点位显示问题分析与解决方案

> **文档版本**: v1.0
> **创建日期**: 2026-05-28
> **系统名称**: OMC 统一网管系统
> **模块名称**: 仪表板 (Dashboard) - 设备地图模块
> **问题类型**: 数据源不一致导致点位不显示

---

## 目录

1. [问题描述](#1-问题描述)
2. [问题分析](#2-问题分析)
3. [架构分析](#3-架构分析)
4. [解决方案](#4-解决方案)
5. [实施步骤](#5-实施步骤)
6. [测试验证](#6-测试验证)
7. [相关文件](#7-相关文件)

---

## 1. 问题描述

### 1.1 问题现象

| 界面 | 预期行为 | 实际行为 | 截图对比 |
|------|---------|---------|---------|
| GIS地图界面 (`/topology/gis`) | 显示所有真实设备点位（约19个，位于赞比亚） | ✅ 正常显示 | 图一 |
| 仪表板地图 (`/dashboard`) | 显示与GIS地图界面相同的设备点位 | ❌ 无点位显示 | 图二 |

### 1.2 影响范围

- **影响模块**: 仪表板页面 → 设备地图模块
- **影响用户**: 所有使用仪表板的运维人员
- **严重程度**: 中等（核心功能不可用，但有GIS地图界面作为替代）

---

## 2. 问题分析

### 2.1 根本原因

**仪表板页面使用了硬编码的 MOCK 数据，而非真实的 API 数据**

```typescript
// pages/dashboard/index.tsx (第 85-96 行)

const MOCK_MAP_DEVICES: MapDevice[] = [
  { id: '1', lat: 39.9, lng: 116.4, status: 'onlineActive', name: '北京基站-001', sn: 'SN-BJ001' },
  { id: '2', lat: 31.2, lng: 121.5, status: 'onlineActive', name: '上海基站-002', sn: 'SN-SH002' },
  { id: '3', lat: 23.1, lng: 113.3, status: 'onlineActive', name: '广州基站-003', sn: 'SN-GZ003', alarmCount: 2 },
  // ... 共 10 个固定点位，全部位于中国境内
];
```

### 2.2 数据源对比

| 方面 | 仪表板页面 | GIS地图界面 |
|------|-----------|------------|
| **数据来源** | 硬编码 `MOCK_MAP_DEVICES` | 真实 API `topologyApi.getDevicesGeo()` |
| **数据量** | 固定 10 个点位 | 动态获取（pageSize: 100） |
| **坐标位置** | 中国城市（经度 100-120°，纬度 20-45°） | 赞比亚设备（经度 22-34°，纬度 -18°~-8°） |
| **数据更新** | 静态不变 | 实时更新 |

### 2.3 坐标偏差分析

```
┌─────────────────────────────────────────────────────────────────┐
│                        坐标位置对比                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  仪表板 MOCK 数据中心:  北京 (116.4°E, 39.9°N)                   │
│  GIS地图实际中心:       赞比亚 (28.2°E, -14.6°N)                 │
│                                                                 │
│  直线距离: 约 8,000+ 公里                                        │
│                                                                 │
│  地图默认视口范围（缩放级别 6）:                                 │
│    - 宽度: 约 1,500 公里                                         │
│    - 高度: 约 1,000 公里                                         │
│                                                                 │
│  结论: MOCK 数据点位完全不在地图可视范围内                         │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 架构分析

### 3.1 现有共用架构

```
┌────────────────────────────────────────────────────────────────────┐
│                      GISMap 组件共用架构                              │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌─────────────────┐    ┌─────────────────┐    ┌──────────────┐  │
│  │  GISMap 组件     │───→│   useOLMap      │───→│ useMapConfig │  │
│  │  (index.tsx)     │    │   (useOLMap.ts) │    │ (useMapConfig│  │
│  └─────────────────┘    │                  │    │    .ts)      │  │
│         │               └─────────────────┘    └──────┬───────┘  │
│         │                                                 │          │
│         │                                                 ↓          │
│         │                                          GET /tiles-metadata  │
│         │                                                 │          │
│         │                                                 ↓          │
│         │                                          ┌──────────────┐    │
│         │                                          │ tiles.json   │    │
│         │                                          │ 中心点、边界  │    │
│         │                                          │ 缩放级别      │    │
│         │                                          └──────────────┘    │
│         │                                                 │          │
│         │                                          失败时降级            │
│         │                                                 ↓          │
│         │                                          DEFAULT_METADATA     │
│         │                                          (赞比亚中心)         │
│         │                                                              │
│         └─────────────────────────────────────────────────────────────│
│                                                                        │
│  数据层:                                                               │
│  ┌─────────────────┐    ┌─────────────────┐    ┌──────────────┐      │
│  │ useMapDevicesGeo│───→│ topologyApi     │───→│ GET          │      │
│  │ (useTopology.ts)│    │ (topologyApi.ts)│    │ /devices/geo │      │
│  └─────────────────┘    └─────────────────┘    └──────────────┘      │
└────────────────────────────────────────────────────────────────────┘
```

### 3.2 GISMap 组件的动态中心点机制

GISMap 组件已实现自动从 `tiles.json` 加载地图中心点的功能：

| 文件 | 功能 |
|------|------|
| `useMapConfig.ts` | 从 `/tiles-metadata` 加载 TileJSON，提取中心点和边界 |
| `useOLMap.ts` | 使用 `useMapConfig` 的元数据初始化地图 |
| `constants.ts` | 提供 `DEFAULT_METADATA`（赞比亚中心）作为降级方案 |

**加载优先级**：
1. 优先使用 `tiles.json` 中定义的中心点
2. `tiles.json` 不存在或格式错误时，降级到 `DEFAULT_METADATA`
3. 最终确保地图始终有可用的中心点配置

### 3.3 公共接口

设备地理数据获取已通过 `topologyApi.getDevicesGeo()` 统一提供：

```typescript
// topologyApi.getDevicesGeo()
// 接口: GET /devices/geo
// 参数: MapFilterParams { groupIds?, status?, pageSize? }
// 返回: { items: DeviceGeo[], total: number }
```

此接口被 GISMapView 使用，仪表板应复用同一接口。

---

## 4. 解决方案

### 4.1 核心思路

**让仪表板使用与 GISMapView 相同的数据源（`useMapDevicesGeo` hook），并利用 GISMap 组件已有的动态中心点机制。**

### 4.2 设计原则

| 原则 | 说明 |
|------|------|
| **复用共用逻辑** | 使用现有的 `useMapDevicesGeo` hook 和 GISMap 组件 |
| **动态中心点** | 地图中心点由 `tiles.json` 决定，不在代码中硬编码 |
| **数据一致性** | 两个界面显示相同的设备点位 |
| **最小改动** | 只修改仪表板页面的数据获取部分 |

### 4.3 架构改进

```
┌────────────────────────────────────────────────────────────────────┐
│                      改进后的数据流                                  │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    仪表板页面 (Dashboard)                      │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │ useMapDevicesGeo() ──→ topologyApi.getDevicesGeo()     │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  │                          ↓                                   │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │ 转换数据格式: DeviceGeo[] ──→ MapDevice[]                │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  │                          ↓                                   │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │ <GISMap devices={mapDevices} />                        │  │
│  │  │   ↓ 自动从 tiles.json 获取中心点                         │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                 GISMapView 页面 (保持不变)                     │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │ useMapDevicesGeo() ──→ topologyApi.getDevicesGeo()     │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  │                          ↓                                   │  │
│  │  ┌─────────────────────────────────────────────────────────┐  │  │
│  │  │ <GISMap devices={mapDevices} />                        │  │
│  │  │   ↓ 自动从 tiles.json 获取中心点                         │  │
│  │  └─────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
```

---

## 5. 实施步骤

### 5.1 文件修改清单

| 文件 | 修改类型 | 说明 |
|------|---------|------|
| `pages/dashboard/index.tsx` | 修改 | 替换 MOCK 数据为真实 API 数据 |

### 5.2 详细修改步骤

#### 步骤 1: 移除硬编码 MOCK 数据

```typescript
// ❌ 删除第 85-96 行
const MOCK_MAP_DEVICES: MapDevice[] = [
  { id: '1', lat: 39.9, lng: 116.4, status: 'onlineActive', name: '北京基站-001', sn: 'SN-BJ001' },
  // ... 删除全部 10 个点位
];
```

#### 步骤 2: 添加导入

```typescript
// ✅ 在文件顶部添加
import { useMapDevicesGeo } from '@core/hooks/api/useTopology';
```

#### 步骤 3: 添加数据获取逻辑

```typescript
// ✅ 在 DashboardPage 组件内添加

// 获取设备地理数据（与 GISMapView 相同的数据源）
const mapFilterParams = useMemo(() => ({
  // 仪表板场景：获取所有状态的设备
  status: undefined,
  // 获取所有设备组的设备
  groupIds: undefined,
  // 启用请求
  enabled: true,
  // 限制数量，避免仪表板加载过慢
  pageSize: 100,
}), []);

const { data: devicesGeoData, isLoading: isMapLoading } = useMapDevicesGeo(mapFilterParams);

// 转换 DeviceGeo 为 MapDevice 格式
const mapDevices = useMemo(() => {
  if (!devicesGeoData?.items?.length) return [];
  return devicesGeoData.items.map(device => ({
    id: device.id,
    lat: device.latitude,
    lng: device.longitude,
    name: device.name,
    status: device.status,
    sn: device.sn,
    groupName: device.groupName,
    address: device.address,
    alarmCount: device.alarmCount,
  }));
}, [devicesGeoData]);
```

#### 步骤 4: 更新 GISMap 组件调用

```typescript
// ❌ 修改前 (第 415-420 行)
<GISMap
  devices={MOCK_MAP_DEVICES}
  height={280}
  showStats={false}
  onDeviceClick={handleDeviceClick}
/>

// ✅ 修改后
<GISMap
  devices={mapDevices}
  height={280}
  showStats={false}
  loading={isMapLoading}
  onDeviceClick={handleDeviceClick}
/>
```

### 5.3 完整代码对比

```typescript
// 修改前
export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: dashboardData, isLoading } = useDashboardData();
  // ... 其他代码

  // ❌ 硬编码 MOCK 数据
  const MOCK_MAP_DEVICES: MapDevice[] = [
    { id: '1', lat: 39.9, lng: 116.4, status: 'onlineActive', name: '北京基站-001', sn: 'SN-BJ001' },
    // ...
  ];

  return (
    <div>
      {/* ... 其他组件 */}
      <GISMap
        devices={MOCK_MAP_DEVICES}  // ❌ 使用 MOCK 数据
        height={280}
        showStats={false}
        onDeviceClick={handleDeviceClick}
      />
    </div>
  );
}

// 修改后
export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: dashboardData, isLoading } = useDashboardData();
  
  // ✅ 使用真实 API 获取设备数据
  const mapFilterParams = useMemo(() => ({
    status: undefined,
    groupIds: undefined,
    enabled: true,
    pageSize: 100,
  }), []);
  
  const { data: devicesGeoData, isLoading: isMapLoading } = useMapDevicesGeo(mapFilterParams);
  
  const mapDevices = useMemo(() => {
    if (!devicesGeoData?.items?.length) return [];
    return devicesGeoData.items.map(device => ({
      id: device.id,
      lat: device.latitude,
      lng: device.longitude,
      name: device.name,
      status: device.status,
      sn: device.sn,
      groupName: device.groupName,
      address: device.address,
      alarmCount: device.alarmCount,
    }));
  }, [devicesGeoData]);

  return (
    <div>
      {/* ... 其他组件 */}
      <GISMap
        devices={mapDevices}  // ✅ 使用真实数据
        height={280}
        showStats={false}
        loading={isMapLoading}
        onDeviceClick={handleDeviceClick}
      />
    </div>
  );
}
```

---

## 6. 测试验证

### 6.1 功能测试

| 测试项 | 测试步骤 | 预期结果 |
|--------|---------|---------|
| **点位显示** | 访问仪表板页面，查看设备地图模块 | 显示与GIS地图界面相同的设备点位 |
| **坐标正确性** | 检查地图中心点和点位位置 | 位于赞比亚区域，非中国区域 |
| **数据一致性** | 对比仪表板和GIS地图界面的点位数量 | 两个界面显示的点位数量一致 |
| **中心点动态性** | 修改 `tiles.json` 中的中心点，刷新页面 | 地图自动使用新的中心点 |
| **降级行为** | 删除或重命名 `tiles.json`，刷新页面 | 地图使用 DEFAULT_METADATA（赞比亚中心） |

### 6.2 性能测试

| 测试项 | 测试步骤 | 预期结果 |
|--------|---------|---------|
| **加载时间** | 测量仪表板页面加载时间 | 与修改前相当（API 请求耗时可接受） |
| **渲染性能** | 放大/缩小地图，检查流畅度 | 无明显卡顿 |

### 6.3 边界场景测试

| 场景 | 测试步骤 | 预期结果 |
|------|---------|---------|
| **无设备数据** | 清空设备表，访问仪表板 | 地图显示但不显示点位，不报错 |
| **网络异常** | 断开后端服务，访问仪表板 | 显示加载失败状态，不崩溃 |
| **大量设备** | 创建 1000+ 设备，访问仪表板 | 正常显示，pageSize 限制数据量 |

---

## 7. 相关文件

### 7.1 前端文件

| 文件路径 | 说明 |
|---------|------|
| `omcmb/webcode/src/pages/dashboard/index.tsx` | 仪表板主页面（需修改） |
| `omcmb/webcode/src/pages/topology/GISMapView/index.tsx` | GIS地图界面（参考实现） |
| `omcmb/webcode/src/components/GISMap/index.tsx` | GISMap 主组件 |
| `omcmb/webcode/src/components/GISMap/useOLMap.ts` | OpenLayers 地图 Hook |
| `omcmb/webcode/src/components/GISMap/useMapConfig.ts` | 地图元数据加载 Hook |
| `omcmb/webcode/src/components/GISMap/constants.ts` | 地图常量配置 |

### 7.2 业务层文件

| 文件路径 | 说明 |
|---------|------|
| `omcmb/frontend-core/src/hooks/api/useTopology.ts` | 拓扑数据 Hooks（包含 `useMapDevicesGeo`） |
| `omcmb/frontend-core/src/services/api/topologyApi.ts` | 拓扑 API（包含 `getDevicesGeo`） |
| `omcmb/frontend-core/src/types/map.ts` | 地图相关类型定义 |

### 7.3 后端接口

| 接口 | 说明 |
|------|------|
| `GET /devices/geo` | 获取设备地理数据 |
| `GET /tiles-metadata` | 获取地图元数据（TileJSON 格式） |

---

## 8. 附录

### 8.1 数据格式转换

`DeviceGeo` → `MapDevice` 转换映射：

| DeviceGeo 字段 | MapDevice 字段 | 说明 |
|----------------|----------------|------|
| `id` | `id` | 设备 ID |
| `latitude` | `lat` | 纬度 |
| `longitude` | `lng` | 经度 |
| `name` | `name` | 设备名称 |
| `status` | `status` | 设备状态 |
| `sn` | `sn` | 设备序列号 |
| `groupName` | `groupName` | 设备组名称 |
| `address` | `address` | 地址 |
| `alarmCount` | `alarmCount` | 告警数量 |

### 8.2 tiles.json 格式示例

```json
{
  "tilejson": "2.2.0",
  "name": "Zambia Offline Map",
  "description": "赞比亚区域离线地图",
  "attribution": "© OpenStreetMap contributors",
  "minzoom": 6,
  "maxzoom": 15,
  "bounds": [22.0, -18.0, 34.0, -8.0],
  "center": [28.221, -14.607, 6],
  "tiles": ["/tiles/{z}/{x}/{y}.png"]
}
```

### 8.3 提交信息建议

```
fix(dashboard): 修复仪表板设备地图点位显示问题

What: 将仪表板设备地图从硬编码 MOCK 数据改为使用真实 API 数据
Why: 仪表板地图与 GIS 地图界面数据源不一致，导致点位不显示
Impact: 仪表板设备地图现在显示与 GIS 地图界面相同的真实设备点位

Related: N/A
```

---

## 9. 仪表板告警模块数据一致性分析

> **分析日期**: 2026-05-28
> **分析范围**: 告警汇总、告警级别分布、近7天告警趋势、设备状态分布（按类型）

### 9.1 数据对齐状态矩阵

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        数据对齐状态矩阵                                       │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  模块                    │ 仪表板  │ 告警管理  │ 数据对齐  │ 状态         │
│  ─────────────────────────┼─────────┼──────────┼──────────┼──────────────│
│  告警汇总                  │  真实API │   真实API │    ✅    │ 已对齐        │
│  告警级别分布              │  真实API │   真实API │    ✅    │ 已对齐        │
│  近7天告警趋势             │  硬编码  │    N/A   │    ❌    │ 未使用真实API │
│  设备状态分布（按类型）    │  硬编码  │    N/A   │    ❌    │ 未使用真实API │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 9.2 详细分析

#### 9.2.1 告警汇总 ✅ 已对齐

**仪表板代码** (`pages/dashboard/index.tsx:91-95`):
```typescript
const { data: alarmCount } = useAlarmCount();
const { data: currentAlarmData } = useCurrentAlarms({
  page: 1,
  pageSize: 6,
});
```

**告警管理代码** (`pages/alarm/CurrentAlarms/index.tsx:224-229`):
```typescript
const { data, isLoading, refetch } = useCurrentAlarms(queryParams);
const { data: alarmCount } = useAlarmCount();
```

**数据源**: 两者完全一致
- API: `GET /alarms/statistics` + `GET /alarms/active`
- Hook: `useAlarmCount()` + `useCurrentAlarms()`
- 刷新频率: 告警数量 15秒，告警列表 30秒

#### 9.2.2 告警级别分布 ✅ 已对齐

**仪表板代码** (`pages/dashboard/index.tsx:137-141`):
```typescript
const critical = alarmCount?.critical ?? 8;
const major = alarmCount?.major ?? 15;
const minor = alarmCount?.minor ?? 12;
const warning = alarmCount?.warning ?? 8;
```

**数据源**: 与告警汇总共享同一 `alarmCount` 数据

#### 9.2.3 近7天告警趋势 ❌ 硬编码

**仪表板代码** (`pages/dashboard/index.tsx:193-198`):
```typescript
const alarmTrendSeries = useMemo(() => [
  { name: t('alarm.severity.critical'), data: [5, 8, 6, 9, 7, 10, 8], color: '#F5222D' },
  { name: t('alarm.severity.major'), data: [12, 15, 11, 18, 14, 17, 15], color: '#FA8C16' },
  { name: t('alarm.severity.minor'), data: [8, 10, 9, 12, 11, 13, 12], color: '#FADB14' },
  { name: t('alarm.severity.warning'), data: [6, 7, 5, 8, 6, 9, 8], color: '#1677FF' },
], [t]);
```

**问题**: 数据完全硬编码，从不更新

**可用 API** (`frontend-core/src/services/api/dashboardApi.ts:242-249`):
```typescript
async getAlarmTrend(days: number = 7): Promise<DashboardChartData['alarmTrend']> {
  const { data } = await http.get<BackendAlarmTrendItem[]>('/dashboard/alarm-trend', { params: { days } });
  return data;
}
```

**可用 Hook** (`frontend-core/src/hooks/api/useDashboard.ts:39-45`):
```typescript
export function useAlarmTrend(days = 7) {
  return useQuery({
    queryKey: ['dashboard', 'alarm-trend', days],
    queryFn: () => api.getAlarmTrend(days),
    refetchInterval: 60000,
  });
}
```

#### 9.2.4 设备状态分布（按类型）❌ 硬编码

**仪表板代码** (`pages/dashboard/index.tsx:175-180`):
```typescript
const deviceStatusXData = ['eNB', 'gNB', 'CPE', 'eGW'];
const deviceStatusSeries = useMemo(() => [
  { name: t('dashboard.chart.online'), data: [432, 318, 265, 122], color: '#52C41A' },
  { name: t('dashboard.chart.offline'), data: [45, 28, 33, 14], color: '#8C8C8C' },
  { name: t('dashboard.chart.alarm'), data: [12, 8, 15, 8], color: '#FA8C16' },
], [t]);
```

**问题**: 数据完全硬编码，从不更新

**可用 API** (`frontend-core/src/services/api/dashboardApi.ts:251-257`):
```typescript
async getDeviceStatusPie(): Promise<DashboardChartData['deviceStatusPie']> {
  const { data } = await http.get<BackendDeviceStatusMap>('/dashboard/device-status');
  return mapDeviceStatusMap(data);
}
```

### 9.3 结论

| 模块 | 状态 | 问题 | 说明 |
|------|------|------|------|
| 告警汇总 | ✅ 已对齐 | - | 使用真实 API，与告警管理页面数据一致 |
| 告警级别分布 | ⚠️ 部分对齐 | **颜色不一致** | 数据已对齐，但饼图颜色与告警汇总标签不一致 |
| 近7天告警趋势 | ❌ 待修复 | 硬编码数据 | 需改用 `useAlarmTrend()` hook |
| 设备状态分布 | ❌ 待修复 | 硬编码数据 | 需改用 `useDeviceStatusPie()` 或 `useDashboardChartData()` hook |

### 9.4 修复建议

### 9.4 颜色对齐问题 ⚠️

**问题**: 告警级别分布（饼图）的颜色与告警汇总中的告警级别标签颜色不一致

**告警汇总标准颜色** (`SEVERITY_COLOR`):
```typescript
const SEVERITY_COLOR: Record<string, string> = {
  critical: '#F5222D',  // 红色
  major: '#FA8C16',     // 橙色
  minor: '#FADB14',     // 黄色
  warning: '#1677FF',   // 蓝色
};
```

**告警级别分布当前代码** (缺少 color 字段):
```typescript
const alarmPieData = useMemo(
  () => [
    { name: t('alarm.severity.critical'), value: critical },     // ❌ 缺少 color
    { name: t('alarm.severity.major'), value: major },          // ❌ 缺少 color
    { name: t('alarm.severity.minor'), value: minor },          // ❌ 缺少 color
    { name: t('alarm.severity.warning'), value: warning },      // ❌ 缺少 color
  ],
  [critical, major, minor, warning, t]
);
```

### 9.5 修复建议

#### 修复1：告警级别分布颜色对齐 ✅ 已完成

```typescript
// 移除默认值，使用 0 替代硬编码的 8/15/12/8
const critical = alarmCount?.critical ?? 0;
const major = alarmCount?.major ?? 0;
const minor = alarmCount?.minor ?? 0;
const warning = alarmCount?.warning ?? 0;

const alarmPieData = useMemo(
  () => [
    { name: t('alarm.severity.critical'), value: critical, color: SEVERITY_COLOR.critical },
    { name: t('alarm.severity.major'), value: major, color: SEVERITY_COLOR.major },
    { name: t('alarm.severity.minor'), value: minor, color: SEVERITY_COLOR.minor },
    { name: t('alarm.severity.warning'), value: warning, color: SEVERITY_COLOR.warning },
  ],
  [critical, major, minor, warning, t]
);
```

#### 修复2：近7天告警趋势 ✅ 已完成

```typescript
// 1. 添加导入
import { useAlarmTrend } from '@core/hooks/api/useDashboard';

// 2. 获取真实数据
const { data: alarmTrendData } = useAlarmTrend(7);

// 3. 转换数据格式，无数据时返回空数组而非默认值
const alarmTrendSeries = useMemo(() => {
  // 没有数据时返回空数组，不显示虚假数据
  if (!alarmTrendData?.length) {
    return [];
  }

  const critical = alarmTrendData.map((d) => d.critical ?? 0);
  const major = alarmTrendData.map((d) => d.major ?? 0);
  const minor = alarmTrendData.map((d) => d.minor ?? 0);
  const warning = alarmTrendData.map((d) => d.warning ?? 0);

  return [
    { name: t('alarm.severity.critical'), data: critical, color: SEVERITY_COLOR.critical },
    { name: t('alarm.severity.major'), data: major, color: SEVERITY_COLOR.major },
    { name: t('alarm.severity.minor'), data: minor, color: SEVERITY_COLOR.minor },
    { name: t('alarm.severity.warning'), data: warning, color: SEVERITY_COLOR.warning },
  ];
}, [alarmTrendData, t]);
```

#### 修复3：设备状态分布 ⚠️ 需后端支持

**问题**：后端 `GET /dashboard/device-status` 只返回按状态分组的总体计数：
```go
// 后端返回：map[model.DeviceStatus]int64
{
  "online": 1137,
  "offline": 120,
  "alarm": 27
}
```

**仪表板需要**：按设备类型（eNB/gNB/CPE/eGW）分组的状态数据：
```typescript
// 前端期望：
{
  eNB: { online: 432, offline: 45, alarm: 12 },
  gNB: { online: 318, offline: 28, alarm: 8 },
  CPE: { online: 265, offline: 33, alarm: 15 },
  eGW: { online: 122, offline: 14, alarm: 8 }
}
```

**解决方案**：需要后端新增 API：
```go
// GET /api/v1/dashboard/device-status-by-type
func (h *Handler) GetDeviceStatusByType(c *gin.Context) {
    // 返回按 technology 分组的设备状态统计
    // SELECT technology, status, COUNT(*) FROM devices GROUP BY technology, status
}
```

**最终方案**：后端新增 `GET /dashboard/device-status-by-type` 接口，前端使用真实 API 数据。

### 9.6 修复状态总结

| 修复项 | 状态 | 说明 |
|--------|------|------|
| 告警级别分布颜色对齐 | ✅ 已完成 | 添加 color 字段与 SEVERITY_COLOR 对齐 |
| 告警级别分布空数据 | ✅ 已完成 | 默认值改为 0，无数据时饼图显示为空 |
| 近7天告警趋势 | ✅ 已完成 | 使用 useAlarmTrend(7) hook 获取真实数据 |
| 近7天告警趋势空数据 | ✅ 已完成 | 无数据使用全 0 数组，保留坐标轴框架和图例 |
| 设备状态分布 | ✅ 已完成 | 后端新增 API，前端使用真实数据 |
| Y 轴优化 | ✅ 已完成 | 添加 min:0, interval:1，强制显示整数刻度 |

### 9.6.1 后端 API 实现

| 接口 | 用途 | 状态 |
|------|------|------|
| `GET /dashboard/device-status-by-type` | 按制式（technology）分组的状态统计 | ✅ 已实现 |

**实际返回格式**：
```json
{
  "lte": { "online": 432, "offline": 45, "alarm": 12 },
  "nr":  { "online": 318, "offline": 28, "alarm": 8 },
  "gsm": { "online": 50,  "offline": 5,  "alarm": 0 }
}
```

**后端实现** (`omcgo/internal/dashboard/service.go`):
```go
type DeviceStatusByType map[string]DeviceStatusCounts

type DeviceStatusCounts struct {
    Online  int64 `json:"online"`
    Offline int64 `json:"offline"`
    Alarm   int64 `json:"alarm"`
}

func (s *Service) GetDeviceStatusByType(ctx context.Context) (DeviceStatusByType, error)
```

**SQL 查询**：
```sql
SELECT
    technology,
    COUNT(*) FILTER (WHERE status = 'active') AS online,
    COUNT(*) FILTER (WHERE status IN ('offline', 'decommissioned')) AS offline,
    COUNT(*) FILTER (WHERE EXISTS (
        SELECT 1 FROM alarms_active aa WHERE aa.device_id = devices.id
    )) AS alarm
FROM devices
WHERE deleted_at IS NULL
GROUP BY technology
ORDER BY technology;
```

**前端实现** (`omcmb/frontend-core/src/services/api/dashboardApi.ts`):
```typescript
type BackendDeviceStatusCounts = {
  online: number;
  offline: number;
  alarm: number;
};

type BackendDeviceStatusByType = Record<string, BackendDeviceStatusCounts>;

async getDeviceStatusByType(): Promise<BackendDeviceStatusByType> {
  const { data } = await http.get<BackendDeviceStatusByType>(
    '/dashboard/device-status-by-type'
  );
  return data;
}
```

**前端 Hook** (`omcmb/frontend-core/src/hooks/api/useDashboard.ts`):
```typescript
export function useDeviceStatusByType() {
  return useQuery({
    queryKey: ['dashboard', 'device-status-by-type'],
    queryFn: () => api.getDeviceStatusByType(),
    refetchInterval: 30000,
  });
}
```

**前端页面使用** (`omcmb/webcode/src/pages/dashboard/index.tsx`):
```typescript
const { data: deviceStatusByTypeData } = useDeviceStatusByType();

const deviceStatusData = useMemo(() => {
  if (!deviceStatusByTypeData) {
    return { xData: [], series: [] };
  }

  const xData = Object.keys(deviceStatusByTypeData);
  const onlineData = xData.map(key => deviceStatusByTypeData[key]?.online ?? 0);
  const offlineData = xData.map(key => deviceStatusByTypeData[key]?.offline ?? 0);
  const alarmData = xData.map(key => deviceStatusByTypeData[key]?.alarm ?? 0);

  return {
    xData,
    series: [
      { name: t('dashboard.chart.online'), data: onlineData, color: '#52C41A' },
      { name: t('dashboard.chart.offline'), data: offlineData, color: '#8C8C8C' },
      { name: t('dashboard.chart.alarm'), data: alarmData, color: '#FA8C16' },
    ],
  };
}, [deviceStatusByTypeData, t]);
```

### 9.7 空数据展示行为

修复后的空数据展示：

| 图表 | 无数据时行为 |
|------|------------|
| **告警级别分布（饼图）** | 所有值为 0，ECharts 自动显示"无数据"状态 |
| **近7天告警趋势（折线图）** | 使用全 0 数组，保留完整坐标轴、图例和网格线 |

#### 近7天告警趋势空数据状态

```typescript
// X 轴数据（始终存在，提供坐标框架）
const trendXData = useMemo(() => {
  const days: string[] = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    days.push(`${d.getMonth() + 1}/${d.getDate()}`);
  }
  return days; // ['5/22', '5/23', '5/24', '5/25', '5/26', '5/27', '5/28']
}, []);

// Y 轴数据（无数据时使用全 0 数组）
const alarmTrendSeries = useMemo(() => {
  const daysCount = 7;

  // 没有数据时使用全 0 数组，保留坐标轴框架
  if (!alarmTrendData?.length) {
    const zeroData = new Array(daysCount).fill(0);
    return [
      { name: t('alarm.severity.critical'), data: zeroData, color: SEVERITY_COLOR.critical },
      { name: t('alarm.severity.major'), data: zeroData, color: SEVERITY_COLOR.major },
      { name: t('alarm.severity.minor'), data: zeroData, color: SEVERITY_COLOR.minor },
      { name: t('alarm.severity.warning'), data: zeroData, color: SEVERITY_COLOR.warning },
    ];
  }

  // 从真实数据中提取各级别的趋势
  const critical = alarmTrendData.map((d) => d.critical ?? 0);
  const major = alarmTrendData.map((d) => d.major ?? 0);
  const minor = alarmTrendData.map((d) => d.minor ?? 0);
  const warning = alarmTrendData.map((d) => d.warning ?? 0);

  return [
    { name: t('alarm.severity.critical'), data: critical, color: SEVERITY_COLOR.critical },
    { name: t('alarm.severity.major'), data: major, color: SEVERITY_COLOR.major },
    { name: t('alarm.severity.minor'), data: minor, color: SEVERITY_COLOR.minor },
    { name: t('alarm.severity.warning'), data: warning, color: SEVERITY_COLOR.warning },
  ];
}, [alarmTrendData, t]);
```

**效果**：图表显示完整的坐标轴、图例、网格线和折线（全部为 0），用户可以看到"严重/主要/次要/警告"的图例和日期轴，明确知道这是告警趋势图表，只是当前没有数据。

### 9.8 代码变更

**文件**: `omcmb/webcode/src/pages/dashboard/index.tsx`

**变更1**: 添加 useAlarmTrend 导入
```typescript
import { useDashboardData, useAlarmTrend } from '@core/hooks/api/useDashboard';
```

**变更2**: 调用 useAlarmTrend hook
```typescript
const { data: alarmTrendData } = useAlarmTrend(7);
```

**变更3**: 修复告警级别默认值（空数据时不显示虚假数据）
```typescript
// 修复前：默认值为硬编码数字
const critical = alarmCount?.critical ?? 8;
const major = alarmCount?.major ?? 15;
const minor = alarmCount?.minor ?? 12;
const warning = alarmCount?.warning ?? 8;

// 修复后：默认值为 0
const critical = alarmCount?.critical ?? 0;
const major = alarmCount?.major ?? 0;
const minor = alarmCount?.minor ?? 0;
const warning = alarmCount?.warning ?? 0;
```

**变更4**: 修复 alarmPieData 颜色对齐
```typescript
const alarmPieData = useMemo(
  () => [
    { name: t('alarm.severity.critical'), value: critical, color: SEVERITY_COLOR.critical },
    { name: t('alarm.severity.major'), value: major, color: SEVERITY_COLOR.major },
    { name: t('alarm.severity.minor'), value: minor, color: SEVERITY_COLOR.minor },
    { name: t('alarm.severity.warning'), value: warning, color: SEVERITY_COLOR.warning },
  ],
  [critical, major, minor, warning, t]
);
```

**变更5**: 修复 alarmTrendSeries 使用真实数据，空数据使用全 0 数组
```typescript
// 修复前：返回硬编码默认值
if (!alarmTrendData?.length) {
  return [
    { name: t('alarm.severity.critical'), data: [5, 8, 6, 9, 7, 10, 8], ... },
    // ...
  ];
}

// 修复后：使用全 0 数组，保留坐标轴框架和图例
if (!alarmTrendData?.length) {
  const zeroData = new Array(daysCount).fill(0);
  return [
    { name: t('alarm.severity.critical'), data: zeroData, color: SEVERITY_COLOR.critical },
    { name: t('alarm.severity.major'), data: zeroData, color: SEVERITY_COLOR.major },
    { name: t('alarm.severity.minor'), data: zeroData, color: SEVERITY_COLOR.minor },
    { name: t('alarm.severity.warning'), data: zeroData, color: SEVERITY_COLOR.warning },
  ];
}

// 从真实数据中提取各级别的趋势
const critical = alarmTrendData.map((d) => d.critical ?? 0);
const major = alarmTrendData.map((d) => d.major ?? 0);
const minor = alarmTrendData.map((d) => d.minor ?? 0);
const warning = alarmTrendData.map((d) => d.warning ?? 0);
```

**变更6**: 设备状态分布使用全 0 数据，移除硬编码
```typescript
// 修复前：硬编码数据
const deviceStatusSeries = useMemo(() => [
  { name: t('dashboard.chart.online'), data: [432, 318, 265, 122], color: '#52C41A' },
  { name: t('dashboard.chart.offline'), data: [45, 28, 33, 14], color: '#8C8C8C' },
  { name: t('dashboard.chart.alarm'), data: [12, 8, 15, 8], color: '#FA8C16' },
], [t]);

// 修复后：使用全 0 数据
const deviceStatusSeries = useMemo(() => {
  const zeroData = new Array(deviceStatusXData.length).fill(0);
  return [
    { name: t('dashboard.chart.online'), data: zeroData, color: '#52C41A' },
    { name: t('dashboard.chart.offline'), data: zeroData, color: '#8C8C8C' },
    { name: t('dashboard.chart.alarm'), data: zeroData, color: '#FA8C16' },
  ];
}, [t, deviceStatusXData.length]);
```

**变更7**: Y 轴优化（LineChart 组件）
```typescript
// components/Charts/LineChart.tsx
yAxis: {
  type: 'value',
  name: yAxisName,
  min: 0,                    // 确保 Y 轴从 0 开始
  minInterval: 1,            // 确保刻度为整数
  max: (value) => {
    // 当最大值为 0 时，设置默认上限为 10，避免刻度太密集
    return value.max === 0 ? 10 : undefined;
  },
  // ...
}
```

---

## 10. 接口分析：GET /api/v1/dashboard/device-status

> **分析日期**: 2026-05-28
> **接口用途**: 设备状态统计

### 10.1 当前返回格式

**后端实现**: `omcgo/internal/dashboard/service.go:GetDeviceStatus()`

```go
// 返回类型: map[model.DeviceStatus]int64
func (s *Service) GetDeviceStatus(ctx context.Context) (map[model.DeviceStatus]int64, error)
```

**实际返回示例**:
```json
{
  "data": {
    "maintenance": 1500,
    "offline": 29221,
    "registered": 3000
  },
  "msg": "ok",
  "ret": 1
}
```

**数据结构**:
- 简单的状态标签 → 计数映射
- 只返回总体统计，**不包含设备类型分组**

### 10.2 需要的返回格式

**前端期望** (`omcmb/webcode/src/pages/dashboard/index.tsx:571-579`):

仪表板需要按设备类型（eNB/gNB/CPE/eGW）分组的状态数据，用于堆叠柱状图展示：

```typescript
// 期望的数据结构
interface DeviceStatusByType {
  [technology: string]: {
    online: number;
    offline: number;
    alarm: number;
  };
}

// 示例
{
  "eNB": { "online": 432, "offline": 45, "alarm": 12 },
  "gNB": { "online": 318, "offline": 28, "alarm": 8 },
  "CPE": { "online": 265, "offline": 33, "alarm": 15 },
  "eGW": { "online": 122, "offline": 14, "alarm": 8 }
}
```

### 10.3 接口使用位置

| 位置 | 文件路径 | 用途 |
|------|---------|------|
| **后端 Handler** | `omcgo/internal/dashboard/handler.go:75-83` | `GetDeviceStatus()` 处理请求 |
| **后端 Service** | `omcgo/internal/dashboard/service.go:372-382` | 调用 `deviceService.CountByStatus()` |
| **后端路由** | `omcgo/internal/dashboard/handler.go:34` | `dashboard.GET("/device-status", h.GetDeviceStatus)` |
| **前端 API** | `omcmb/frontend-core/src/services/api/dashboardApi.ts:252-257` | `getDeviceStatusPie()` 方法 |
| **前端 Hook** | `omcmb/frontend-core/src/hooks/api/useDashboard.ts:47-53` | `useDeviceStatusPie()` hook |
| **前端页面** | `omcmb/webcode/src/pages/dashboard/index.tsx` | 当前未使用（已用全 0 数据替代） |

### 10.4 数据源分析

**当前 SQL 查询** (通过 `deviceService.CountByStatus()`):
```sql
-- 返回按状态分组的设备总数
SELECT status, COUNT(*) FROM devices GROUP BY status;
```

**需要的 SQL 查询** (新增 API):
```sql
-- 返回按 technology 和 status 双维度分组的设备数
SELECT
    technology,
    status,
    COUNT(*) as count
FROM devices
GROUP BY technology, status
ORDER BY technology, status;
```

### 10.5 解决方案

#### 方案 A：新增独立 API（推荐）

新增 `GET /api/v1/dashboard/device-status-by-type` 接口：

**优点**:
- 不破坏现有接口
- 前端可以灵活选择使用哪个接口
- 职责分离更清晰

**后端实现**:
```go
// omcgo/internal/dashboard/handler.go
dashboard.GET("/device-status-by-type", h.GetDeviceStatusByType)

// omcgo/internal/dashboard/service.go
func (s *Service) GetDeviceStatusByType(ctx context.Context) (map[string]DeviceStatusCounts, error) {
    query := `
        SELECT technology, status, COUNT(*)
        FROM devices
        GROUP BY technology, status`
    // ... 解析结果为 map[string]DeviceStatusCounts
}

type DeviceStatusCounts struct {
    Online  int64 `json:"online"`
    Offline int64 `json:"offline"`
    Alarm   int64 `json:"alarm"`
}
```

#### 方案 B：扩展现有 API（不推荐）

修改现有 `/device-status` 接口，添加查询参数 `?group_by=type`：

**缺点**:
- 破坏现有契约
- 需要所有调用方适配

### 10.6 前端适配（方案 A）

```typescript
// omcmb/frontend-core/src/services/api/dashboardApi.ts
async getDeviceStatusByType(): Promise<DeviceStatusByType> {
  const { data } = await http.get<DeviceStatusByType>(
    '/dashboard/device-status-by-type'
  );
  return data;
}

// omcmb/frontend-core/src/hooks/api/useDashboard.ts
export function useDeviceStatusByType() {
  return useQuery({
    queryKey: ['dashboard', 'device-status-by-type'],
    queryFn: () => api.getDeviceStatusByType(),
    refetchInterval: 30000,
  });
}

// omcmb/webcode/src/pages/dashboard/index.tsx
const { data: deviceStatusByType } = useDeviceStatusByType();

// 转换为图表格式
const deviceStatusXData = Object.keys(deviceStatusByType || {});
const deviceStatusSeries = useMemo(() => {
  if (!deviceStatusByType) return [];
  return [
    {
      name: t('dashboard.chart.online'),
      data: deviceStatusXData.map(k => deviceStatusByType[k]?.online || 0),
      color: '#52C41A'
    },
    {
      name: t('dashboard.chart.offline'),
      data: deviceStatusXData.map(k => deviceStatusByType[k]?.offline || 0),
      color: '#8C8C8C'
    },
    {
      name: t('dashboard.chart.alarm'),
      data: deviceStatusXData.map(k => deviceStatusByType[k]?.alarm || 0),
      color: '#FA8C16'
    }
  ];
}, [deviceStatusByType, deviceStatusXData, t]);
```

---

## 11. TOP10告警设备数据一致性分析

> **分析日期**: 2026-05-28
> **分析范围**: TOP10告警设备卡片数据源与对齐状态

### 11.1 数据对齐状态

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        TOP10告警设备数据对齐状态                              │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  模块                    │ 仪表板  │ 告警管理  │ 数据对齐  │ 状态         │
│  ─────────────────────────┼─────────┼──────────┼──────────┼──────────────│
│  TOP10告警设备             │  硬编码  │  真实API │    ❌    │ 未使用真实API │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 11.2 详细分析

#### 11.2.1 仪表板当前实现 ❌ 硬编码

**仪表板代码** (`omcmb/webcode/src/pages/dashboard/index.tsx`):

```typescript
// 硬编码的设备名称
const top10Devices = [
  '成都基站-005', '广州基站-003', '昆明基站-010', '哈尔滨-009',
  '北京-001', '上海-002', '西安-007', '南京-008', '深圳-004', '兰州-006',
];

// 硬编码的告警次数
const top10Series = useMemo(() => [
  { name: t('dashboard.alarmCount'), data: [24, 21, 18, 16, 14, 12, 10, 8, 6, 4] },
], [t]);
```

**问题**:
- 数据完全硬编码，永不更新
- 设备名称为虚构的测试数据（如"成都基站-005"）
- 告警次数为固定数字 [24, 21, 18, 16, 14, 12, 10, 8, 6, 4]
- 与真实告警数据无任何关联

#### 11.2.2 告警管理界面数据源 ✅ 真实API

**告警管理代码** (`omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx`):

```typescript
const { data, isLoading, refetch } = useCurrentAlarms(queryParams);
```

**数据源**:
- API: `GET /alarms/active`
- Hook: `useCurrentAlarms()`
- 数据: 实时活动告警列表
- 刷新频率: 30秒

#### 11.2.3 可用的真实API

**后端实现** (`omcgo/internal/dashboard/service.go`):

```go
// GET /dashboard/summary
// recent_alarms 字段来源于 alarmStore.ListActive()
// 获取前5条活动告警，按设备SN分组聚合

type FrontendRecentAlarm struct {
    DeviceName string `json:"device_name"`
    AlarmCount int64  `json:"alarm_count"`
    Severity   string `json:"severity"`
}
```

**前端API** (`omcmb/frontend-core/src/services/api/dashboardApi.ts:259-265`):

```typescript
/** Top alarm devices — extracted from /dashboard/summary recent_alarms */
async getTopAlarmDevices(): Promise<DashboardChartData['topAlarmDevices']> {
  const { data } = await http.get<BackendDashboardSummary>('/dashboard/summary');
  return mapRecentAlarmsToTopDevices(data.recent_alarms || []);
}
```

**前端Hook** (`omcmb/frontend-core/src/hooks/api/useDashboard.ts`):

```typescript
export function useTopAlarmDevices() {
  return useQuery({
    queryKey: ['dashboard', 'top-alarm-devices'],
    queryFn: () => api.getTopAlarmDevices(),
    refetchInterval: 60000,  // 60秒刷新
  });
}
```

### 11.3 数据源关系图

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         数据源关系图                                         │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  后端数据源:                                                               │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ alarmStore.ListActive() → 活动告警列表                                │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                              │                                             │
│                ┌─────────────┴─────────────┐                              │
│                ↓                           ↓                              │
│  ┌──────────────────────┐      ┌──────────────────────┐                  │
│  │ GET /alarms/active    │      │ GET /dashboard/summary                │
│  │ (告警管理界面)         │      │   recent_alarms字段                     │
│  └──────────────────────┘      └──────────────────────┘                  │
│                │                           │                              │
│                ↓                           ↓                              │
│  ┌──────────────────────┐      ┌──────────────────────┐                  │
│  │ useCurrentAlarms()    │      │ useTopAlarmDevices() │                  │
│  │ (告警管理界面)         │      │   60秒刷新           │                  │
│  └──────────────────────┘      └──────────────────────┘                  │
│                │                           │                              │
│                ↓                           │                              │
│  ┌──────────────────────┐                 │                              │
│  │ 告警管理界面           │                 │                              │
│  │ ✅ 使用真实数据        │                 │                              │
│  └──────────────────────┘                 │                              │
│                                            │                              │
│                                            ↓                              │
│                                    ┌──────────────────────┐              │
│                                    │ 仪表板TOP10告警设备    │              │
│                                    │ ❌ 使用硬编码MOCK数据   │              │
│                                    └──────────────────────┘              │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 11.4 结论

| 维度 | 状态 | 说明 |
|------|------|------|
| API可用性 | ✅ 可用 | `useTopAlarmDevices()` hook 已存在 |
| 数据源一致性 | ✅ 一致 | 与告警管理界面同源 (`alarmStore.ListActive()`) |
| 仪表板实现 | ❌ 硬编码 | 使用固定MOCK数据，未调用真实API |
| 数据对齐 | ❌ 未对齐 | 需要修改代码使用真实API |

### 11.5 修复方案

#### 修复步骤

**步骤1**: 添加导入
```typescript
// omcmb/webcode/src/pages/dashboard/index.tsx
import { useDashboardData, useAlarmTrend, useTopAlarmDevices } from '@core/hooks/api/useDashboard';
```

**步骤2**: 调用 useTopAlarmDevices hook
```typescript
// 获取TOP10告警设备真实数据
const { data: topAlarmDevicesData } = useTopAlarmDevices();
```

**步骤3**: 转换数据格式
```typescript
// 转换为图表格式
const top10Devices = useMemo(() => {
  if (!topAlarmDevicesData?.length) return [];
  return topAlarmDevicesData.map(d => d.deviceName);
}, [topAlarmDevicesData]);

const top10Series = useMemo(() => {
  if (!topAlarmDevicesData?.length) {
    // 无数据时返回空数组
    return [{ name: t('dashboard.alarmCount'), data: [] }];
  }
  return [{
    name: t('dashboard.alarmCount'),
    data: topAlarmDevicesData.map(d => d.alarmCount),
  }];
}, [topAlarmDevicesData, t]);
```

**步骤4**: 移除硬编码数据
```typescript
// 删除以下硬编码代码
// const top10Devices = [
//   '成都基站-005', '广州基站-003', '昆明基站-010', '哈尔滨-009',
//   '北京-001', '上海-002', '西安-007', '南京-008', '深圳-004', '兰州-006',
// ];
// const top10Series = useMemo(() => [
//   { name: t('dashboard.alarmCount'), data: [24, 21, 18, 16, 14, 12, 10, 8, 6, 4] },
// ], [t]);
```

### 11.6 数据量说明

**注意**: 后端 `GET /dashboard/summary` 的 `recent_alarms` 字段当前只返回**前5条活动告警**聚合后的设备数据，而非TOP10。

后端逻辑：
```go
// omcgo/internal/dashboard/service.go
filter.PageSize = 5  // 硬编码，固定5条
filter.SortBy = "raised_at"
filter.SortDir = "desc"
```

如需完整的TOP10数据，后端需要修改 `filter.PageSize = 5` 为 `filter.PageSize = 10`。

### 11.6.1 标题变更历史

**问题**: 原标题"TOP10告警设备"与实际数据量不符（后端只返回5条），容易造成用户困惑。

**解决方案**: 将标题改为"高频告警设备"，不限定具体条数，更加通用。

**变更记录**:
| 日期 | 标题（中文） | 标题（英文） | 原因 |
|------|------------|------------|------|
| 2026-05-28 之前 | TOP10告警设备 | TOP 10 Alarm Devices | 初始设计 |
| 2026-05-28 | 高频告警设备 | Top Alarm Devices | 后端只返回5条，标题不限定数量更准确 |

### 11.7 修复状态

| 修复项 | 状态 | 说明 |
|--------|------|------|
| TOP10告警设备数据源 | ✅ 已完成 | 已使用 `useTopAlarmDevices()` 获取真实数据 |
| 移除硬编码数据 | ✅ 已完成 | 已删除硬编码的设备名称和告警次数 |
| 空状态展示 | ✅ 已完成 | 无数据时显示 EmptyState 组件 |
| 标题优化 | ✅ 已完成 | 将"TOP10告警设备"改为"高频告警设备"，避免与实际数据量不符 |

---

## 12. 系统管理员卡片数据分析与修复方案

> **分析日期**: 2026-05-28
> **分析范围**: 仪表板系统管理员卡片数据源

### 12.1 数据对齐状态

```
┌────────────────────────────────────────────────────────────────────────────┐
│                    系统管理员卡片数据对齐状态                                 │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  数据项              │ 当前值      │ 数据来源   │ 是否有API  │ 修复优先级    │
│  ───────────────────┼─────────────┼────────────┼────────────┼──────────────│
│  用户头像             │ 静态图标     │ 硬编码     │ ✅ userStore │ P1           │
│  用户名/显示名        │ "系统管理员"  │ 硬编码     │ ✅ userStore │ P1           │
│  邮箱                │ admin@omc.com │ 硬编码     │ ✅ userStore │ P1           │
│  在线状态            │ "在线"       │ 硬编码     │ ⚠️  暂无API  │ P2           │
│  最后登录时间        │ "09:00"      │ 硬编码     │ ✅ userStore │ P1           │
│  今日操作数          │ 156          │ 硬编码     │ ✅ audit_logs │ P1          │
│  已处理告警数        │ 23           │ 硬编码     │ ❌ 需新增API │ P3           │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 12.2 详细分析

#### 12.2.1 卡片位置

`omcmb/webcode/src/pages/dashboard/index.tsx` 第 527-573 行

#### 12.2.2 当前实现（全部硬编码）

```typescript
// 第 536-570 行
<Avatar size={64} icon={<UserOutlined />} style={{ background: token.colorPrimary }} />
<Title level={5} style={{ margin: 0 }}>
  {t('dashboard.sysAdmin')}
</Title>
<Text type="secondary" style={{ fontSize: 13 }}>
  admin@omc.com  {/* ❌ 硬编码邮箱 */}
</Text>
<div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
  <Badge color="green" text={t('status.online')} />  {/* ❌ 硬编码在线状态 */}
  <Text type="secondary" style={{ fontSize: 12 }}>
    {t('dashboard.lastLogin')} 09:00  {/* ❌ 硬编码登录时间 */}
  </Text>
</div>
<div style={{ /* 统计数据网格 */ }}>
  <div style={{ textAlign: 'center' }}>
    <div style={{ fontWeight: 700, fontSize: 18, color: token.colorPrimary }}>156</div>
    {/* ❌ 硬编码今日操作数 */}
    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>{t('dashboard.todayOps')}</div>
  </div>
  <div style={{ textAlign: 'center' }}>
    <div style={{ fontWeight: 700, fontSize: 18, color: '#52C41A' }}>23</div>
    {/* ❌ 硬编码已处理告警数 */}
    <div style={{ fontSize: 12, color: token.colorTextSecondary }}>{t('dashboard.processedAlarms')}</div>
  </div>
</div>
```

### 12.3 可用的现有资源

#### 12.3.1 用户数据 (userStore)

**位置**: `omcmb/frontend-core/src/store/userStore.ts`

**可用字段**:
```typescript
interface User {
  id: string;
  username: string;
  displayName: string;      // ✅ 显示名称
  email: string;             // ✅ 邮箱
  role: UserRole;            // ✅ 角色
  avatar?: string;           // ✅ 头像 URL (可选)
  lastLoginTime?: string;    // ✅ 最后登录时间 (ISO 8601)
  // ... 其他字段
}
```

**使用方式**:
```typescript
import { useUserStore } from '@core/store/userStore';

const { currentUser } = useUserStore();
// currentUser?.displayName  → 显示名称
// currentUser?.email         → 邮箱
// currentUser?.lastLoginTime → 最后登录时间
// currentUser?.avatar        → 头像 URL
```

#### 12.3.2 操作日志统计 (audit_logs)

**后端表**: `audit_logs` (PostgreSQL)

**后端 API**: `GET /api/v1/audit-logs`
- 位置: `omcgo/internal/admin/handler.go:259`
- Repository: `omcgo/internal/admin/pg_audit_repository.go`
- 支持按用户、操作类型、时间范围过滤

**可用字段**:
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID,
    username VARCHAR(100),
    action VARCHAR(50),          -- login, config, upgrade, reboot, delete
    resource VARCHAR(100),
    resource_id VARCHAR(100),
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ
);
```

**统计今日操作数 SQL**:
```sql
SELECT COUNT(*) as today_ops
FROM audit_logs
WHERE user_id = $1
  AND DATE(created_at) = CURRENT_DATE;
```

### 12.4 修复方案

#### 方案 A：前端修复（P1 - 立即可实施）

**修复范围**: 用户信息、最后登录时间

| 数据项 | 修复方式 |
|--------|---------|
| 用户头像/邮箱/显示名 | 从 `useUserStore().currentUser` 获取 |
| 最后登录时间 | 从 `currentUser.lastLoginTime` 格式化 |
| 在线状态 | 登录即为在线（显示固定"在线"徽章） |

**代码变更**:
```typescript
// 1. 添加导入
import { useUserStore } from '@core/store/userStore';

// 2. 获取当前用户信息
const { currentUser } = useUserStore();

// 3. 格式化最后登录时间
const formatLastLogin = useCallback((lastLoginTime?: string): string => {
  if (!lastLoginTime) return '--';
  const date = new Date(lastLoginTime);
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}, []);

// 4. 更新 JSX
<Avatar
  size={64}
  src={currentUser?.avatar}
  icon={!currentUser?.avatar ? <UserOutlined /> : undefined}
  style={{ background: currentUser?.avatar ? undefined : token.colorPrimary }}
/>
<Title level={5} style={{ margin: 0 }}>
  {currentUser?.displayName || currentUser?.username || t('dashboard.sysAdmin')}
</Title>
<Text type="secondary" style={{ fontSize: 13 }}>
  {currentUser?.email || '--'}
</Text>
<div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
  <Badge color="green" text={t('status.online')} />
  <Text type="secondary" style={{ fontSize: 12 }}>
    {t('dashboard.lastLogin')} {formatLastLogin(currentUser?.lastLoginTime)}
  </Text>
</div>
```

#### 方案 B：今日操作数（P1 - 需后端支持）

**后端需求**: 新增 `GET /api/v1/dashboard/user-stats` 接口

**请求**: 无参数（从 JWT token 获取当前用户 ID）

**响应**:
```json
{
  "today_ops": 156,
  "processed_alarms": 23
}
```

**后端实现** (`omcgo/internal/dashboard/`):
```go
// handler.go
dashboard.GET("/user-stats", h.GetUserStats)

// service.go
func (s *Service) GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
    // 今日操作数
    todayOps, err := s.getTodayOpsCount(ctx, userID)
    // 已处理告警数 (需要告警模块支持，暂时返回 0 或 NULL)
    processedAlarms := int64(0) // TODO: 等待告警处理记录表

    return &UserStats{
        TodayOps:        todayOps,
        ProcessedAlarms: processedAlarms,
    }, nil
}

func (s *Service) getTodayOpsCount(ctx context.Context, userID uuid.UUID) (int64, error) {
    query := `
        SELECT COUNT(*)
        FROM audit_logs
        WHERE user_id = $1
          AND DATE(created_at) = CURRENT_DATE
    `
    var count int64
    err := s.pool.QueryRow(ctx, query, userID).Scan(&count)
    return count, err
}
```

**前端适配**:
```typescript
// frontend-core/src/services/api/dashboardApi.ts
async getUserStats(): Promise<{ today_ops: number; processed_alarms: number }> {
  const { data } = await http.get('/dashboard/user-stats');
  return data;
}

// frontend-core/src/hooks/api/useDashboard.ts
export function useUserStats() {
  return useQuery({
    queryKey: ['dashboard', 'user-stats'],
    queryFn: () => api.getUserStats(),
    refetchInterval: 60000, // 60秒刷新
  });
}

// pages/dashboard/index.tsx
const { data: userStats } = useUserStats();

// 统计数据
<div style={{ fontWeight: 700, fontSize: 18, color: token.colorPrimary }}>
  {userStats?.today_ops ?? 0}
</div>
<div style={{ fontWeight: 700, fontSize: 18, color: '#52C41A' }}>
  {userStats?.processed_alarms ?? 0}
</div>
```

#### 方案 C：已处理告警数（P3 - 需告警模块支持）

**问题**: 后端当前没有"告警处理记录"表，无法统计用户已处理的告警数量

**解决方案**:
1. 后端新增 `alarm_handled` 表记录告警处理操作
2. 或者使用 `audit_logs` 表统计 `action='ack'` 或 `action='clear'` 的告警数量

**临时方案**: 暂时显示 0 或不显示此字段

### 12.5 修复优先级

| 优先级 | 修复项 | 工作量 | 阻塞 |
|--------|--------|--------|------|
| **P1** | 用户信息从 userStore 获取 | 前端 0.5h | 无 |
| **P1** | 今日操作数 API | 后端 1h + 前端 0.5h | 无 |
| **P2** | 在线状态（可选） | 前端 0.5h | 需后端支持会话管理 |
| **P3** | 已处理告警数 | 后端 2h + 前端 0.5h | 需告警模块配合 |

### 12.6 实施建议

**第一阶段**（立即实施）:
- 修复用户信息显示（头像、邮箱、显示名、最后登录时间）
- 今日操作数暂时显示 0，等待后端 API

**第二阶段**（后续迭代）:
- 后端实现 `GET /api/v1/dashboard/user-stats` 接口
- 前端对接真实数据

**第三阶段**（可选）:
- 实现在线状态检测
- 实现已处理告警数统计

### 12.7 数据源关系图

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         系统管理员卡片数据流                                 │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  用户信息 (P1):                                                           │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ 登录时: authApi.getMe() → userStore.login(user)                      │  │
│  │ 运行时: useUserStore().currentUser                                   │  │
│  │   ├─ displayName / email / avatar                                     │  │
│  │   └─ lastLoginTime                                                   │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                              │                                             │
│                              ↓                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ 仪表板系统管理员卡片                                                   │  │
│  │   ├─ 头像: currentUser?.avatar ?? UserOutlined                        │  │
│  │   ├─ 显示名: currentUser?.displayName ?? username                      │  │
│  │   ├─ 邮箱: currentUser?.email                                         │  │
│  │   └─ 最后登录: formatLastLogin(currentUser.lastLoginTime)            │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
│  今日操作数 (P1):                                                         │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ audit_logs 表 (后端已有)                                              │  │
│  │   ├─ user_id                                                         │  │
│  │   ├─ action                                                          │  │
│  │   └─ created_at                                                      │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                              │                                             │
│                              ↓ 需新增接口                                   │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ GET /api/v1/dashboard/user-stats (待实现)                             │  │
│  │   SELECT COUNT(*) FROM audit_logs                                    │  │
│  │   WHERE user_id = $1 AND DATE(created_at) = CURRENT_DATE             │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                              │                                             │
│                              ↓                                             │
│  今日操作数: userStats?.today_ops ?? 0                                   │
│                                                                            │
│  已处理告警数 (P3):                                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ 暂无数据源，需要后端新增 alarm_handled 表或使用 audit_logs 统计        │  │
│  │ 临时方案: 显示 0 或隐藏此字段                                         │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

**文档结束**

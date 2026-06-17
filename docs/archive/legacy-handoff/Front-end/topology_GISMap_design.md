# 通用基站 GIS 地图功能设计方案

> **文档版本**：v1.8
> **创建日期**：2026-03-20
> **更新日期**：2026-03-23
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
│  │  ├── MapControls (缩放：放大/缩小)                  │  │
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
VITE_MAP_DEFAULT_CENTER=104.0,35.0,4             // 多地区部署时的兜底中心点（格式：lng,lat,zoom）
```

### 2.6 中心点决策链

> **核心设计**：支持多地区部署，中心点由优先级决策而非硬编码

```
优先级 1: tiles.json 中的 center 字段
         (离线地图元数据返回的中心点 - 最优先)
         ↓ (不存在或异常)
         
优先级 2: /api/v1/devices/geo 设备数据计算
         (根据实际设备位置自动计算中心点 - 智能适配)
         ├─ 计算设备 bounds（经纬度范围）
         ├─ 中心点 = [(minLng+maxLng)/2, (minLat+maxLat)/2]
         └─ 缩放级别根据设备分布范围自动调整
         ↓ (无设备数据)
         
优先级 3: 环境变量 VITE_MAP_DEFAULT_CENTER
         (多地区部署时的兜底配置)
         ↓ (未配置)
         
优先级 4: 代码内置默认值
         (全球通用默认值 [0, 20, 2])
```

**多地区部署示例**：

```bash
# .env.development（赞比亚）
VITE_MAP_DEFAULT_CENTER=28.221,-14.607,6

# .env.staging（中国）
VITE_MAP_DEFAULT_CENTER=104.0,35.0,4

# .env.production（其他地区，由设备数据决定）
# 不设置，使用全球默认值，优先从设备数据计算中心点
```

**优势**：
- ✅ 自动适配部署地区（无需手动修改代码）
- ✅ 基于真实设备数据智能定位
- ✅ 多地区部署只需环境变量配置
- ✅ 页面层无硬编码坐标

---

## 三、功能设计

### 3.1 功能架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        GIS 地图页面                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      地图主体区域 (OpenLayers)            │   │
│  │  ┌────────────────────┐  ┌─────────────────────────────┐  │   │
│  │  │ 节点查找 (左上角)   │  │                             │  │   │
│  │  │ 🔍 [搜索设备...]    │  │  [设备标记+告警角标] [聚合圈] │  │   │
│  │  │ ┌─────────────────┐│  │                             │  │   │
│  │  │ │搜索结果列表      ││  │      瓦片地图底图            │  │   │
│  │  │ │ 名称|SN|状态    ││  │                             │  │   │
│  │  │ │ 名称|SN|状态    ││  │                             │  │   │
│  │  │ └─────────────────┘│  │                             │  │   │
│  │  └────────────────────┘  └─────────────────────────────┘  │   │
│  │  ┌─────────┐                                              │   │
│  │  │ 缩放控制 │                                              │   │
│  │  └─────────┘                                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────────┐ │
│  │ 筛选面板    │  │ 统计面板    │  │ 悬浮提示 (hover时显示)      │ │
│  │ (左侧)     │  │ (右下浮动)  │  │ (跟随鼠标)                  │ │
│  │ 设备组树    │  │ 状态统计    │  │ 状态/序列号/设备组/地址/坐标/告警 │
│  │ 状态筛选    │  │ 告警统计    │  │                            │ │
│  └────────────┘  └────────────┘  └────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 功能清单

| 编号 | 功能模块 | 功能名称 | 描述 | 优先级 |
|------|----------|----------|------|--------|
| M01 | 地图展示 | 底图渲染 | 使用 OpenLayers 渲染瓦片地图 | P0 |
| M02 | 地图展示 | 缩放控制 | 支持+/-按钮和滚轮缩放 | P0 |
| M03 | 地图展示 | 平移拖拽 | 支持鼠标拖拽平移地图 | P0 |
| M04 | 地图展示 | 缩放工具列 | 仅保留放大和缩小操作 | P1 |
| M05 | 设备标记 | 状态标记 | 按状态显示不同颜色标记，显示告警数量 | P0 |
| M06 | 设备标记 | 聚合显示 | 高密度区域聚合为圆圈 | P0 |
| M07 | 设备标记 | 悬停提示 | 悬停显示状态、序列号、设备组、地址、坐标、告警 | P0 |
| M08 | 设备标记 | 点击详情 | 点击显示设备详情卡片 | P1 |
| M09 | 筛选功能 | 设备组筛选 | 按设备组树形多选筛选（目录节点显示 xxx group） | P0 |
| M10 | 筛选功能 | 状态筛选 | 按在线/离线筛选 | P0 |
| M11 | 筛选功能 | 类型筛选 | 按设备类型筛选 | P1 |
| M12 | 地图搜索 | 节点查找 | 左上角搜索框，支持设备名称/序列号搜索，显示结果列表，点击定位高亮 | P0 |
| M13 | 统计面板 | 状态统计 | 显示各状态设备数量 | P0 |
| M14 | 统计面板 | 设备组统计 | 显示当前设备组内设备数 | P1 |
| M15 | 详情卡片 | 设备信息 | 显示选中设备详细信息（含告警数） | P0 |

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

[悬停标记] ──► 显示悬浮提示面板（状态、序列号、设备组、地址、坐标、告警）

[拖拽平移] ──► 移动完成后，请求新区域的聚合数据

[节点查找] ──► 输入关键词 ──┬──► 实时搜索匹配设备名称/序列号
                          │
                          ├──► 显示搜索结果列表（名称|SN|状态）
                          │
                          └──► 点击列表项 ──► 定位到设备位置 ──► 高亮显示节点并弹出悬浮提示
```

#### 3.3.2 聚合显示规则（OpenLayers Cluster）

| 缩放级别 | 显示方式 | 聚合距离 | 说明 |
|----------|----------|----------|------|
| zoom 1-6 (全国) | 省级聚合 | 100px | 显示各省设备总数 |
| zoom 7-10 (省级) | 市级聚合 | 60px | 显示各市设备总数 |
| zoom 11-12 (市级) | 网格聚合 | 40px | 按网格聚合 |
| zoom 13+ (详细) | 独立标记 | 0 | 显示每个设备独立标记 |

#### 3.3.3 设备状态可视化规则

| 设备状态 | 标记颜色 | 标记样式 | 说明 |
|----------|----------|----------|------|
| 在线 (online) | 🟢 #52C41A | 实心圆点 + 告警角标 | 告警数量显示在右上角角标 |
| 离线 (offline) | 🔴 #b60808 | 实心红色圆点 | 无告警显示 |

**告警角标规则**：
- 在线设备若有告警，在标记右上角显示红色角标数字
- 角标样式：红色圆形背景 + 白色数字
- 数字超过 99 显示 "99+"
- 离线设备不显示告警角标

#### 3.3.4 设备组树形结构

```
┌─────────────────────────────────────────────────────────────────┐
│                      设备组树形结构                              │
└─────────────────────────────────────────────────────────────────┘

设备组 (DeviceGroup) - 纯目录结构
┌─────────────────────────────────────────────────────────────────┐
│  图标说明：                                                      │
│  [+] 表示已收起，可展开        [-] 表示已展开，可收起            │
│  叶子节点无图标，仅显示文字                                       │
└─────────────────────────────────────────────────────────────────┘

展开/收起状态示例：
┌─────────────────────────────┐
│ ☑ [−] China group           │  ← 根节点，已展开（有子组）
│      ☑ [−] Beijing group    │  ← 一级子组，已展开（有子组）
│          ☑ [+] Chaoyang group   │ ← 二级子组，已收起（有子组）
│          ☐ [−] Haidian group    │ ← 二级子组，已展开（有子组）
│              ☐ Zhongguancun group   │ ← 叶子节点（无子组，无图标）
│              ☐ Wudaokou group       │ ← 叶子节点（无子组，无图标）
│      ☑ [+] Shanghai group   │  ← 一级子组，已收起（有子组）
│      ☐ [+] Guangdong group  │  ← 一级子组，已收起（有子组）
└─────────────────────────────┘

展开/收起图标规则：
┌────────────────┬──────────────────────────────────────────────────┐
│ 图标            │ 说明                                              │
├────────────────┼──────────────────────────────────────────────────┤
│ [+]            │ 已收起状态，点击可展开子组（仅非叶子节点显示）      │
│ [−]            │ 已展开状态，点击可收起子组（仅非叶子节点显示）      │
│ 无图标          │ 叶子节点（无子组），仅显示组名文字                  │
└────────────────┴──────────────────────────────────────────────────┘

节点说明：
┌────────────────┬──────────────────────────────────────────────────┐
│ 节点类型        │ 说明                                              │
├────────────────┼──────────────────────────────────────────────────┤
│ 非叶子节点      │ 有子组的设备组，显示 [+]/[-] 图标                 │
│                │ 命名格式: xxx group                               │
│                │ 例如: China group, Beijing group                  │
├────────────────┼──────────────────────────────────────────────────┤
│ 叶子节点        │ 无子组的设备组，不显示任何图标，仅显示组名          │
│                │ 例如: Zhongguancun group, Wudaokou group          │
└────────────────┴──────────────────────────────────────────────────┘

业务规则：
1. 设备组为纯目录结构，所有节点都是 xxx group 格式
2. 设备组支持多级嵌套（最多5级）
3. 设备挂载在设备组下，通过地图标记展示
4. **设备组支持多选**：可同时选择多个设备组
5. **展开/收起使用 +/- 图标**：+ 表示收起可展开，- 表示展开可收起
6. **叶子节点无图标**：没有子组的节点只显示组名文字

多选交互规则：
┌──────────────────────────────────────────────────────────────────┐
│  设备组多选模式                                                   │
├──────────────────────────────────────────────────────────────────┤
│  ☑ [−] Beijing group           ← 选中且已展开（有子组）          │
│      ☑ [+] Chaoyang group      ← 选中且已收起（有子组）          │
│      ☐ [−] Haidian group       ← 未选中且已展开（有子组）        │
│          ☐ Zhongguancun group  ← 未选中（叶子节点，无图标）      │
│          ☐ Wudaokou group      ← 未选中（叶子节点，无图标）      │
│  ☑ [+] Shanghai group          ← 选中且已收起（有子组）          │
│  ☐ [+] Guangdong group         ← 未选中且已收起（有子组）        │
│                                                                  │
│  已选择 3 个设备组                                                │
└──────────────────────────────────────────────────────────────────┘

- 选择父级设备组时，自动包含其所有子组
- 支持跨级多选（如同时选择 Beijing group 和 Shanghai group）
- 底部显示已选设备组数量
- 提供"全选"和"清空"快捷操作
- 点击 [+] 展开子组，点击 [−] 收起子组
- 叶子节点无展开/收起功能，不显示图标
```

#### 3.3.5 悬浮提示面板设计

```
┌─────────────────────────────────────────┐
│  📍 Beijing Chaoyang Site 001           │
│  ─────────────────────────────────────  │
│  状态: 🟢 在线          告警: 🔴 3       │
│  ─────────────────────────────────────  │
│  序列号: SN2024001                      │
│  设备组: Chaoyang group                 │
│  地址: 北京市朝阳区建国路88号            │
│  坐标: 116.4632, 39.9213                │
└─────────────────────────────────────────┘
```

**悬浮提示面板规则**：
- **触发方式**：鼠标悬停在设备标记上时显示
- **显示内容**：
  - **标题行**：设备名称
  - **状态行**：设备状态（在线/离线）+ 告警数量
  - **详情行**：
    - 序列号 (SN)
    - 设备组路径（从根到当前组的完整路径）
    - 详细地址
    - 地理坐标（经度, 纬度）
- **不包含**：查看详情按钮、查看告警按钮、导航按钮
- **样式**：白色背景卡片，阴影效果，跟随鼠标位置
- **消失条件**：鼠标移出设备标记区域
- **最大宽度**：280px，超出内容自动换行

#### 3.3.6 节点查找与搜索结果列表

```
┌─────────────────────────────────────────────────────────────────┐
│                     节点查找与搜索结果                            │
└─────────────────────────────────────────────────────────────────┘

搜索框（地图左上角）：
┌─────────────────────────────────────────────────┐
│ 🔍  搜索设备名称 / 序列号定位...          [×] ▼ │
└─────────────────────────────────────────────────┘

搜索结果列表（下拉展开）：
┌─────────────────────────────────────────────────┐
│ 🔍  Beijing Chaoyang...                [×] ▼   │  ← 输入中
├─────────────────────────────────────────────────┤
│  📍 Beijing Chaoyang Site 001                   │  ← 结果项 1
│     SN: SN2024001        🟢 在线                │
│  ─────────────────────────────────────────────  │
│  📍 Beijing Chaoyang Site 002                   │  ← 结果项 2（hover 高亮）
│     SN: SN2024002        🔴 离线                │
│  ─────────────────────────────────────────────  │
│  📍 Beijing Chaoyang Device 003                 │  ← 结果项 3
│     SN: SN2024003        🟢 在线                │
│  ─────────────────────────────────────────────  │
│  ...                                            │
│                                                 │
│  共找到 15 个结果                               │
└─────────────────────────────────────────────────┘
```

**搜索结果列表规则**：

| 属性 | 规则 |
|------|------|
| 触发方式 | 在搜索框输入关键词时实时显示（防抖 300ms） |
| 显示条件 | 输入内容非空且匹配到结果时显示 |
| 列表位置 | 紧贴搜索框下方，最大高度 320px，超出滚动 |
| 每项内容 | 设备名称（第一行）+ 序列号 + 状态（第二行） |
| 状态显示 | 🟢 在线 / 🔴 离线 |
| 交互反馈 | hover 时背景高亮（#F5F5F5） |

**点击结果项交互**：

```
点击搜索结果项
    │
    ├──► 关闭搜索结果列表
    │
    ├──► 地图平移到设备坐标位置
    │
    ├──► 调整缩放级别（确保可见，zoom >= 14）
    │
    └──► 高亮显示目标节点
         │
         ├──► 外圈脉冲动画（3次）
         │
         └──► 显示悬浮提示面板
```

**节点高亮效果**：

```
正常状态：        高亮状态：
   ┌───┐            ┌───────┐
   │ ● │            │ ◎ ◎ ◎ │  ← 3层脉冲圈动画
   └───┘            │   ●   │  ← 原始标记
                   └───────┘
```

**高亮样式规范**：

| 属性 | 值 |
|------|---|
| 脉冲圈颜色 | rgba(24, 144, 255, 0.3) |
| 脉冲圈数量 | 3 层，由内向外扩散 |
| 动画时长 | 2s，重复 3 次后停止 |
| 脉冲圈半径 | 20px → 40px → 60px |
| 标记放大 | 原始半径 × 1.5 |

**空结果状态**：

```
┌─────────────────────────────────────────────────┐
│ 🔍  xyz123...                           [×] ▼  │
├─────────────────────────────────────────────────┤
│                                                 │
│              🔍                                 │
│         未找到匹配设备                          │
│     请尝试其他关键词搜索                        │
│                                                 │
└─────────────────────────────────────────────────┘
```

**数据模型**：

```typescript
/**
 * 搜索结果项
 */
interface DeviceSearchResult {
  /** 设备ID */
  id: string;
  /** 设备名称 */
  name: string;
  /** 序列号 */
  sn: string;
  /** 设备状态 */
  status: DeviceStatus;
  /** 经度 */
  longitude: number;
  /** 纬度 */
  latitude: number;
  /** 设备组名称 */
  groupName?: string;
}
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
  /** 聚合内告警总数 */
  alarmCount: number;
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
  /** 总告警数 */
  alarmCount: number;
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
  /** 设备组ID列表（筛选这些组及其子组下所有设备，支持多选） */
  groupIds?: string[];
  /** 状态筛选（仅支持 online/offline） */
  status?: DeviceStatus[];
  /** 类型筛选 */
  type?: DeviceType[];
  /** 搜索关键词（名称/序列号） */
  keyword?: string;
  /** 是否启用查询（控制请求发送）*/
  enabled?: boolean;
  /** 分页大小 */
  pageSize?: number;
  /** 视图边界（用于动态加载）*/
  bounds?: string;
}

/**
 * 地图配置
 */
interface MapConfig {
  /** 默认中心点（多地区部署时的兜底值）*/
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

/**
 * 中心点决策结果
 */
interface CenterPointDecision {
  /** 中心点坐标 [lng, lat] */
  center: [number, number];
  /** 缩放级别 */
  zoom: number;
  /** 决策来源 */
  source: 'metadata' | 'device_data' | 'env_config' | 'default';
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
}> = {
  online: {
    color: '#52C41A',
    bgColor: '#F6FFED',
    borderColor: '#B7EB8F',
    text: '在线',
    i18nKey: 'status.online',
  },
  offline: {
    color: '#8C8C8C',
    bgColor: '#F5F5F5',
    borderColor: '#b60808',
    text: '离线',
    i18nKey: 'status.offline',
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
        "name": "China group",
        "parent_id": null,
        "carrier": "cmcc",
        "description": "中国移动全国网络",
        "sort_order": 1,
        "children": [
          {
            "id": "group-002",
            "name": "Beijing group",
            "parent_id": "group-001",
            "children": [
              {
                "id": "group-003",
                "name": "Chaoyang group",
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
│ group_ids   │ string │ 否     │ 设备组ID列表，多个用逗号分隔（含子组）         │
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
        "name": "Beijing Chaoyang Device 01",
        "sn": "SN2024001",
        "longitude": 116.46,
        "latitude": 39.92,
        "status": "online",
        "type": "small",
        "group_id": "group-003",
        "group_name": "Chaoyang group",
        "address": "北京市朝阳区建国路88号",
        "alarm_count": 3
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
    "group_ids": ["group-002", "group-003"],    // 设备组ID列表（多选）
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
          "offline": 3
        },
        "alarm_count": 5,
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
│ group_ids   │ string │ 否     │ 设备组ID列表，多个用逗号分隔（含子组）         │
│ bounds      │ string │ 否     │ 视图边界（仅统计视图内）                      │
└─────────────┴────────┴────────┴─────────────────────────────────────────────┘

Response:
{
  "code": 0,
  "data": {
    "total": 10000,
    "status_count": {
      "online": 9200,
      "offline": 800
    },
    "alarm_count": 500,
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

> **设计原则**：遵循现有 `omcmb/webcode` 项目结构规范，扩展现有文件而非创建新模块。

```
src/
├── components/
│   └── GISMap/                      # GISMap 组件目录
│       ├── index.tsx                # GISMap 主组件（对外入口）
│       ├── MapMarker.tsx            # 设备标记样式组件
│       ├── MapCluster.tsx           # 聚合圆圈组件
│       ├── MapPopup.tsx             # 悬浮提示组件（名称+状态+告警数，无操作按钮）
│       ├── MapControls.tsx          # 地图控制按钮（仅放大/缩小）
│       ├── MapStatsPanel.tsx        # 统计面板组件
│       ├── GroupTree.tsx            # 设备组树形筛选组件
│       ├── useOLMap.ts              # OpenLayers 地图初始化 Hook（核心逻辑）
│       ├── useDeviceLayer.ts        # 设备图层 Hook
│       ├── useClusterSource.ts      # 聚合数据源 Hook
│       ├── featureUtils.ts          # OpenLayers Feature 工具
│       ├── styleUtils.ts            # 样式工具（颜色/图标）
│       ├── geoUtils.ts              # 地理计算工具
│       ├── constants.ts             # 常量配置
│       └── styles.module.css        # 样式文件
│
├── utils/
│   └── mapValidation.ts             # 【新增】地图中心点计算、参数验证等工具函数
│
├── services/api/
│   └── topologyApi.ts               # 【扩展】新增地图相关 API 方法
│
├── hooks/api/
│   └── useTopology.ts               # 【扩展】新增地图相关 React Query Hooks
│
├── types/
│   └── map.ts                       # 【新增】地图相关类型定义
│
└── pages/topology/GISMapView/
    └── index.tsx                    # 页面入口（已存在，需适配）
```

**与现有项目结构的对齐说明**：

| 设计方案 | 现有项目 | 说明 |
|----------|----------|------|
| `topologyApi.ts` 扩展 | 已存在 `topologyApi.ts` | 复用现有文件，新增地图 API 方法 |
| `useTopology.ts` 扩展 | 已存在 `useTopology.ts` | 复用现有文件，新增地图 Hooks |
| `src/types/map.ts` | 现有 `src/types/` 目录 | 遵循现有类型定义规范 |
| GISMap 组件扁平结构 | 现有 GISMap 组件 | 简化结构，避免过度分层 |

### 6.2 OpenLayers 核心实现（Hook 风格）

> **设计原则**：使用自定义 Hook 封装 OpenLayers 逻辑，符合 React 函数式组件风格。

```typescript
// components/GISMap/useOLMap.ts
import { useEffect, useRef, useCallback, useState } from 'react';
import Map from 'ol/Map';
import View from 'ol/View';
import TileLayer from 'ol/layer/Tile';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Cluster from 'ol/source/Cluster';
import OSM from 'ol/source/OSM';
import XYZ from 'ol/source/XYZ';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import { fromLonLat, toLonLat } from 'ol/proj';
import { defaults as defaultControls } from 'ol/control';
import { Style, Circle, Fill, Stroke, Text } from 'ol/style';
import type { MapDevice, MapViewport, MapOptions } from '@/types/map';
import { DEVICE_STATUS_CONFIG } from './constants';

interface UseOLMapReturn {
  mapRef: React.RefObject<HTMLDivElement>;
  updateDevices: (devices: MapDevice[]) => void;
  getViewport: () => MapViewport | null;
  flyTo: (lng: number, lat: number, zoom?: number) => void;
  highlightDevice: (deviceId: string) => void;
}

export function useOLMap(options: MapOptions): UseOLMapReturn {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<Map | null>(null);
  const deviceSourceRef = useRef<VectorSource | null>(null);
  const clusterSourceRef = useRef<Cluster | null>(null);
  const [isReady, setIsReady] = useState(false);

  // 初始化地图
  useEffect(() => {
    if (!mapRef.current || mapInstanceRef.current) return;

    // 创建瓦片图层
    const tileLayer = new TileLayer({
      source: options.tileUrl
        ? new XYZ({ url: options.tileUrl })  // 离线瓦片
        : new OSM(),                          // OpenStreetMap
    });

    // 创建设备数据源
    deviceSourceRef.current = new VectorSource();

    // 创建聚合数据源
    clusterSourceRef.current = new Cluster({
      source: deviceSourceRef.current,
      distance: options.clusterDistance || 40,
    });

    // 创建设备图层
    const deviceLayer = new VectorLayer({
      source: clusterSourceRef.current,
      style: createClusterStyle,
      zIndex: 10,
    });

    // 创建地图实例
    mapInstanceRef.current = new Map({
      target: mapRef.current,
      layers: [tileLayer, deviceLayer],
      view: new View({
        center: fromLonLat(options.center || [104.0, 35.0]),
        zoom: options.zoom || 4,
        minZoom: options.minZoom || 1,
        maxZoom: options.maxZoom || 18,
      }),
      controls: defaultControls({ zoom: false }),
    });

    setIsReady(true);

    // 绑定事件
    bindMapEvents(mapInstanceRef.current, options);

    // 清理函数
    return () => {
      mapInstanceRef.current?.setTarget(undefined);
      mapInstanceRef.current = null;
      deviceSourceRef.current = null;
      clusterSourceRef.current = null;
    };
  }, []);

  // 创建聚合样式
  const createClusterStyle = useCallback((feature: Feature): Style => {
    const features = feature.get('features') as Feature[];
    const size = features?.length || 0;

    if (size === 1) {
      // 单个设备标记
      return createDeviceStyle(features[0]);
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
  }, []);

  // 创建设备标记样式
  const createDeviceStyle = useCallback((feature: Feature): Style => {
    const status = feature.get('status') as keyof typeof DEVICE_STATUS_CONFIG;
    const config = DEVICE_STATUS_CONFIG[status] || DEVICE_STATUS_CONFIG.offline;

    return new Style({
      image: new Circle({
        radius: 8,
        fill: new Fill({ color: config.color }),
        stroke: new Stroke({ color: '#fff', width: 2 }),
      }),
    });
  }, []);

  // 更新设备数据
  const updateDevices = useCallback((devices: MapDevice[]) => {
    if (!deviceSourceRef.current) return;

    // 清除现有数据
    deviceSourceRef.current.clear();

    // 添加新数据
    const features = devices.map(device => {
      const feature = new Feature({
        geometry: new Point(fromLonLat([device.lng, device.lat])),
        ...device,
      });
      feature.setId(device.id);
      return feature;
    });

    deviceSourceRef.current.addFeatures(features);
  }, []);

  // 获取当前视图状态
  const getViewport = useCallback((): MapViewport | null => {
    if (!mapInstanceRef.current) return null;

    const map = mapInstanceRef.current;
    const view = map.getView();
    const center = toLonLat(view.getCenter()!);
    const extent = view.calculateExtent(map.getSize());

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
  }, []);

  // 飞行到指定位置
  const flyTo = useCallback((lng: number, lat: number, zoom = 14) => {
    if (!mapInstanceRef.current) return;

    const view = mapInstanceRef.current.getView();
    view.animate({
      center: fromLonLat([lng, lat]),
      zoom,
      duration: 1000,
    });
  }, []);

  // 高亮设备
  const highlightDevice = useCallback((deviceId: string) => {
    // 实现高亮动画逻辑
    // ...
  }, []);

  return {
    mapRef,
    updateDevices,
    getViewport,
    flyTo,
    highlightDevice,
    isReady,
  };
}

// 绑定地图事件
function bindMapEvents(map: Map, options: MapOptions): void {
  // 视图变化事件
  map.on('moveend', () => {
    if (options.onViewportChange) {
      const view = map.getView();
      const center = toLonLat(view.getCenter()!);
      options.onViewportChange({
        centerLng: center[0],
        centerLat: center[1],
        zoom: view.getZoom()!,
        bounds: { /* ... */ },
      });
    }
  });

  // 点击事件
  map.on('click', (evt) => {
    const features = map.getFeaturesAtPixel(evt.pixel);
    if (features.length > 0 && options.onDeviceClick) {
      const feature = features[0] as Feature;
      const device = feature.get('features')?.[0] || feature;
      options.onDeviceClick(device.getProperties() as MapDevice);
    }
  });
}
```

### 6.3 核心组件接口

```typescript
// types/map.ts

/**
 * GISMap 主组件 Props
 */
interface GISMapProps {
  /** 设备数据列表 */
  devices: MapDevice[];
  /** 地图高度 */
  height?: string | number;
  /** 默认中心点 [lng, lat]（由 GISMapView 通过中心点决策链计算） */
  defaultCenter?: [number, number];
  /** 默认缩放级别（由 GISMapView 通过中心点决策链计算） */
  defaultZoom?: number;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 是否显示统计面板 */
  showStats?: boolean;
  /** 是否显示控制按钮 */
  showControls?: boolean;
  /** 自定义样式 */
  className?: string;
}

/**
 * MapDevice - 地图设备标记数据
 */
interface MapDevice {
  /** 设备ID */
  id: string;
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
  /** 设备组名称 */
  groupName?: string;
  /** 告警数量 */
  alarmCount?: number;
  /** 详细地址 */
  address?: string;
}

/**
 * GroupTree 组件 Props
 */
interface GroupTreeProps {
  /** 选中的设备组ID列表（支持多选） */
  selectedGroupIds?: string[];
  /** 选中变化回调 */
  onSelect: (groupIds: string[]) => void;
  /** 是否显示搜索 */
  showSearch?: boolean;
}

/**
 * 地图配置选项
 */
interface MapOptions {
  /** 瓦片服务地址（离线模式） */
  tileUrl?: string;
  /** 默认中心点 */
  center?: [number, number];
  /** 默认缩放级别 */
  zoom?: number;
  /** 最小缩放级别 */
  minZoom?: number;
  /** 最大缩放级别 */
  maxZoom?: number;
  /** 聚合距离 */
  clusterDistance?: number;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
}
```

### 6.4 React Query Hooks（扩展 useTopology.ts）

> **设计原则**：扩展现有 `useTopology.ts`，新增地图相关 hooks，遵循项目现有模式。

```typescript
// hooks/api/useTopology.ts（扩展部分）

import { useQuery } from '@tanstack/react-query';
import { topologyApi } from '@/services/api/topologyApi';
import { useMock } from '@/services/apiSwitch';
import type { MapFilterParams, DeviceGeo, DeviceCluster, MapStats } from '@/types/map';

// ── 现有 hooks 保持不变 ──
// useDomains, useDomainTree, useSites, useTopoNodes, useTopoEdges, useTopoGraph, useGeoData
// useCreateGroup, useUpdateGroup, useDeleteGroup, useGroupDevices, useAddDeviceToGroup, useRemoveDeviceFromGroup

// ── 新增地图相关 hooks ──

/**
 * 获取设备地理数据（支持筛选）
 */
export function useMapDevicesGeo(params: MapFilterParams) {
  return useQuery({
    queryKey: ['topology', 'map', 'geo', params],
    queryFn: () =>
      useMock
        ? Promise.resolve({ items: [], total: 0 }) // Mock 实现
        : topologyApi.getDevicesGeo(params),
    staleTime: 5 * 60 * 1000, // 5分钟
    enabled: params.enabled !== false,
  });
}

/**
 * 获取聚合数据（大范围视图）
 */
export function useMapAggregation(params: {
  bounds: MapBounds;
  zoom: number;
  filters?: MapFilterParams;
}) {
  return useQuery({
    queryKey: ['topology', 'map', 'aggregation', params],
    queryFn: () =>
      useMock
        ? Promise.resolve({ clusters: [] }) // Mock 实现
        : topologyApi.getAggregation(params),
    staleTime: 2 * 60 * 1000, // 2分钟
    enabled: params.zoom < 12, // 仅在缩放级别较小时请求
  });
}

/**
 * 获取地图统计数据
 */
export function useMapStats(params?: { groupIds?: string[]; bounds?: string }) {
  return useQuery({
    queryKey: ['topology', 'map', 'stats', params],
    queryFn: () =>
      useMock
        ? Promise.resolve({ total: 0, statusCount: {}, alarmCount: 0 }) // Mock 实现
        : topologyApi.getMapStats(params),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 60 * 1000, // 每分钟刷新
  });
}

/**
 * 搜索设备（节点查找）
 */
export function useMapDeviceSearch(keyword: string) {
  return useQuery({
    queryKey: ['topology', 'map', 'search', keyword],
    queryFn: () =>
      useMock
        ? Promise.resolve([]) // Mock 实现
        : topologyApi.searchDevices(keyword),
    staleTime: 30 * 1000, // 30秒
    enabled: keyword.length >= 2,
  });
}
```

### 6.5 API 服务层（扩展 topologyApi.ts）

> **设计原则**：扩展现有 `topologyApi.ts`，新增地图相关 API 方法，遵循项目现有模式（BackendXxx 接口 + mapBackendXxx 转换函数）。

```typescript
// services/api/topologyApi.ts（扩展部分）

import http from '../http';
import type { DeviceGeo, DeviceCluster, MapStats, MapFilterParams, DeviceSearchResult } from '@/types/map';

// ── 新增 Backend 类型定义（snake_case） ──

interface BackendDeviceGeo {
  id: string;
  name: string;
  sn: string;
  longitude: number;
  latitude: number;
  status: string;
  type?: string;
  group_id: string;
  group_name?: string;
  address?: string;
  alarm_count?: number;
}

interface BackendDeviceCluster {
  id: string;
  longitude: number;
  latitude: number;
  count: number;
  status_count: Record<string, number>;
  alarm_count: number;
  bounds?: {
    min_lng: number;
    max_lng: number;
    min_lat: number;
    max_lat: number;
  };
}

interface BackendMapStats {
  total: number;
  status_count: Record<string, number>;
  alarm_count: number;
  type_count?: Record<string, number>;
  viewport_count?: number;
}

interface BackendSearchResult {
  id: string;
  name: string;
  sn: string;
  status: string;
  longitude: number;
  latitude: number;
  group_name?: string;
}

// ── 新增转换函数 ──

function mapBackendDeviceGeo(bd: BackendDeviceGeo): DeviceGeo {
  return {
    id: bd.id,
    name: bd.name,
    sn: bd.sn,
    longitude: bd.longitude,
    latitude: bd.latitude,
    status: bd.status as DeviceStatus,
    type: bd.type as DeviceType,
    groupId: bd.group_id,
    groupName: bd.group_name,
    address: bd.address,
    alarmCount: bd.alarm_count,
  };
}

function mapBackendCluster(bc: BackendDeviceCluster): DeviceCluster {
  return {
    id: bc.id,
    longitude: bc.longitude,
    latitude: bc.latitude,
    count: bc.count,
    statusCount: bc.status_count as Record<DeviceStatus, number>,
    alarmCount: bc.alarm_count,
    bounds: bc.bounds ? {
      minLng: bc.bounds.min_lng,
      maxLng: bc.bounds.max_lng,
      minLat: bc.bounds.min_lat,
      maxLat: bc.bounds.max_lat,
    } : undefined,
  };
}

function mapBackendStats(bs: BackendMapStats): MapStats {
  return {
    total: bs.total,
    statusCount: bs.status_count as Record<DeviceStatus, number>,
    alarmCount: bs.alarm_count,
    typeCount: bs.type_count as Record<DeviceType, number> | undefined,
    viewportCount: bs.viewport_count,
  };
}

function mapBackendSearchResult(bs: BackendSearchResult): DeviceSearchResult {
  return {
    id: bs.id,
    name: bs.name,
    sn: bs.sn,
    status: bs.status as DeviceStatus,
    longitude: bs.longitude,
    latitude: bs.latitude,
    groupName: bs.group_name,
  };
}

// ── 扩展现有 topologyApi 对象 ──

export const topologyApi = {
  // ... 现有方法保持不变 ...

  // ── 新增地图相关 API 方法 ──

  /**
   * 获取设备地理数据（支持筛选）
   */
  async getDevicesGeo(params: MapFilterParams): Promise<{ items: DeviceGeo[]; total: number }> {
    const { data } = await http.get<{ items: BackendDeviceGeo[]; total: number }>('/devices/geo', {
      params: {
        group_ids: params.groupIds?.join(','),
        status: params.status?.join(','),
        keyword: params.keyword,
        bounds: params.bounds,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return {
      items: (data.items || []).map(mapBackendDeviceGeo),
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
    const { data } = await http.post<{ clusters: BackendDeviceCluster[] }>('/devices/geo/aggregate', {
      bounds: {
        min_lng: params.bounds.minLng,
        max_lng: params.bounds.maxLng,
        min_lat: params.bounds.minLat,
        max_lat: params.bounds.maxLat,
      },
      zoom: params.zoom,
      grid_size: params.gridSize || 50,
      filters: {
        group_ids: params.filters?.groupIds,
        status: params.filters?.status,
      },
    });
    return {
      clusters: (data.clusters || []).map(mapBackendCluster),
    };
  },

  /**
   * 获取地图统计数据
   */
  async getMapStats(params?: { groupIds?: string[]; bounds?: string }): Promise<MapStats> {
    const { data } = await http.get<BackendMapStats>('/devices/geo/stats', {
      params: {
        group_ids: params?.groupIds?.join(','),
        bounds: params?.bounds,
      },
    });
    return mapBackendStats(data);
  },

  /**
   * 搜索设备（节点查找）
   */
  async searchDevices(keyword: string): Promise<DeviceSearchResult[]> {
    const { data } = await http.get<{ items: BackendSearchResult[] }>('/devices/search', {
      params: { keyword },
    });
    return (data.items || []).map(mapBackendSearchResult);
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

| 文件 | 路径 | 说明 |
|------|------|------|
| GISMap 组件 | src/components/GISMap/ | 地图组件目录（扁平结构） |
| 主入口组件 | src/components/GISMap/index.tsx | GISMap 主组件 |
| OpenLayers Hook | src/components/GISMap/useOLMap.ts | 地图核心逻辑 Hook |
| 设备组树组件 | src/components/GISMap/GroupTree.tsx | 设备组树形筛选组件 |
| 拓扑 API（扩展） | src/services/api/topologyApi.ts | 新增地图相关 API 方法 |
| 拓扑 Hooks（扩展） | src/hooks/api/useTopology.ts | 新增地图相关 React Query Hooks |
| 地图类型定义 | src/types/map.ts | 地图相关类型定义 |
| GISMap 页面 | src/pages/topology/GISMapView/index.tsx | 页面入口（已存在，需适配） |

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
| v1.2 | 2026-03-23 | - | 状态筛选简化为在线/离线；节点增加告警角标；悬浮提示增加告警数并移除操作按钮；设备组命名改为 xxx group 格式；缩放工具列仅保留放大/缩小 |
| v1.3 | 2026-03-23 | - | 悬浮提示面板增加序列号、设备组、地址、坐标字段；设备组树区分目录节点(xxx group)和叶子节点(站点名称)；增加地图左上角节点查找功能 |
| v1.4 | 2026-03-23 | - | 设备组筛选支持多选；MapFilterParams.groupIds 替换 groupId；API 接口支持多个设备组ID |
| v1.5 | 2026-03-23 | - | 设备组树简化为纯目录结构，移除 Site 叶子节点；设备组后不显示数量；多选汇总只显示组数量 |
| v1.6 | 2026-03-23 | - | 设备组展开/收起图标从箭头改为 +/- 号；搜索结果以列表形式展示（名称|SN|状态），点击定位并高亮节点 |
| v1.7 | 2026-03-23 | - | 离线状态标记改为实心灰色圆点（原空心白色圆点） |
| v1.8 | 2026-03-23 | - | 设备组树移除文件夹图标，仅保留 +/- 展开收起标识，叶子节点无图标 |

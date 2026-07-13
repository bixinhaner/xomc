# GIS 扇区覆盖能力设计

> **文档日期**：2026-07-10
>
> **文档状态**：已实施（v1 范围）
>
> **Issue**：GIS 地图支持扇区方向与覆盖示意
>
> **范围**：v1 `webcode` 的 `/topology/gis-map`、GIS 共用数据契约
>
> **代码基线**：`origin/main@f6ee9e43f`（2026-07-10）

## 实施结果（2026-07-10）

本设计的 v1 最小闭环已交付，实际边界如下：

- 新增 `GET /api/v1/devices/{id}/antenna-sectors`，从 `device_parameters` 的 `antenna` 分组运行时解析扇区；接口沿用既有设备查看权限。
- 解析器兼容无编号字段、`Azimuth2/3` 等编号字段以及 `CellConfig.{i}` 实例路径。非法、非有限或不满足几何范围的数值会被标记为缺失，不绘制误导性的方向或覆盖范围。
- v1 GIS 在缩放等级至少为 13 时为当前选中设备绘制方向线和覆盖环形扇面。方位角以正北为 $0^\circ$、顺时针增加；水平覆盖角固定为 $120^\circ$，即中心法线两侧各 $60^\circ$。
- 弹窗按 Tab 展示各扇区。方位角与机械下倾可在地图内预览后提交现有异步 `SetParameterValues` 任务；“取消”恢复本地预览。任务已提交不代表设备已生效。
- 垂直波瓣宽度、天线高度和水平波瓣宽度目前只读展示。垂直波瓣宽度缺少当前产品参数模型的可写映射，因此不参与本期下发。
- v2/v3、多个设备覆盖固定对比、人工覆盖表、传播模型和自动化浏览器组件测试不在本期范围。

验证记录：后端 `go build ./... && go test ./...`、前端 `npm run typecheck`（含三皮肤路由/菜单守卫）均已通过；GIS 浏览器验证覆盖扇区 Tab、固定 $120^\circ$ 扇面、编辑预览与取消恢复。

## 1. 目标与结论

本 Issue 只处理 GIS 基站天线扇区方向和地面覆盖范围展示，并允许同时观察多个相邻基站或多个小区的覆盖方向。左侧筛选栏默认收起属于另一个独立 Issue，不在本文范围内。

推荐采用“原始参数单一数据源 + 服务层规范化 + 按设备加载详情 + v1 OpenLayers 渲染”的方案：

- 扇区不是 `MapDevice` 上附带的几个临时字段，而是独立的设备扇区集合，一台设备可有 1～N 个扇区。
- `/devices/geo` 继续承担大量点位的轻量查询；点击设备后再查询扇区详情，避免地图移动时反复传输天线参数。
- 首期直接读取现有 `device_parameters`，不新建一张复制全部天线值的扇区主表，避免设备上报值与业务表形成双数据源。
- 新增天线参数解析器，将不同产品路径、无编号字段、编号字段和实例路径统一为 `AntennaSectorDTO`。
- 方向示意由水平方位角和水平覆盖角决定；距离覆盖还需要天线高度、总下倾角和垂直 3dB 波束宽度。当前代码只能证明存在名称相近的字段，物理口径未确认前不得伪装成精确覆盖。
- 不复制老项目“缺数据就不显示且没有提示”“非法角度改用固定半径”“一次只能显示一个站”等缺陷。
- 前端本期只考虑 v1 `webcode`，使用 OpenLayers 矢量多边形呈现扇区；v2/v3 暂不纳入设计和实施范围。

## 2. 输入材料与可信边界

分析依据如下：

- `gis-6-feature-audit.md` 对老项目 6 个 GIS 功能点的审计结果。
- 老项目 `eNBTopo_tab.jsp`、`topo-helper.js`、`ol-topo-helper.js` 的实际交互与绘制代码。
- 附件中的参数说明、地图效果、设备编辑界面和历史缺陷记录。
- 当前项目 v1 GIS 页面、OpenLayers Hook、Geo API 和设备详情天线参数装配。

老项目的 `DeviceTopoServiceImpl.java` 文件受 Safenet 保护，仓库中不是可直接阅读的 Java 源码。因此，老后端 SQL 和写入细节只能由审计报告、JSP 请求参数、前端字段映射交叉确认；本文不会把无法直接验证的老后端实现当作新系统设计依据。

## 3. 老项目实现还原

### 3.1 数据模型

老项目把一个基站节点映射为一组天线参数：

| 字段 | 含义 | 老项目用途 |
|---|---|---|
| `latitude` / `longitude` | 基站坐标 | 扇区中心点 |
| `height` | 天线中心距地面高度 | 覆盖半径计算 |
| `mechanical_downtilt` | 机械下倾角 | 总下倾角 |
| `electronic_downtilt` | 电子下倾角 | 总下倾角 |
| `vertical_3dB_beam_width` | 垂直 3dB 波束宽度 | 近点/远点计算 |
| `horizontal_azimuth` | 水平方位角 | 扇区中心方向，正北为 0°，顺时针 |
| `angles` | 水平覆盖角 | 代码固定为 115° |

`topo-helper.js::transformNode()` 将 `horizontal_azimuth` 映射为 `direct`，并计算 `radius` 和 `minRadius`。测试序列号以 `testsn` 开头时，部分空字段会被写入测试默认值；生产设备没有通用默认值。

### 3.2 覆盖半径公式

老项目计算逻辑可整理为：

```text
totalTilt = mechanicalDowntilt + electronicDowntilt
halfVerticalBeam = verticalBeamWidth / 2

farAngle  = totalTilt - halfVerticalBeam
nearAngle = totalTilt + halfVerticalBeam

farRadius  = antennaHeight / tan(farAngle)
nearRadius = antennaHeight / tan(nearAngle)
```

含义是垂直波束上下边界与地面的交点：

- `nearRadius`：主瓣近点半径。
- `farRadius`：主瓣远点半径。
- 地面覆盖区域位于近点和远点之间，不是从基站中心一直填充到远点。

老代码随后把远点限制在 50m～10km，近点限制为不小于 10m；角度非法时直接使用 500m/100m 固定值。

### 3.3 两层扇区呈现

老项目有两种不同含义的扇区：

1. **白色方向扇区**
   - 点击基站节点后显示。
   - 固定像素大小，外半径 25px、内半径 8px。
   - 只用于指示天线朝向，不代表真实覆盖距离。

2. **蓝色渐变覆盖扇区**
   - 再点击白色方向扇区后展开。
   - 使用近点/远点半径生成 12 层透明度递减的环形扇区。
   - 为保证视觉可见，老代码又按当前地图分辨率把显示半径至少放大到约 80px，并允许放大到 50km。

因此老项目的蓝色范围不是严格的物理覆盖预测，而是“基于天线几何参数的示意范围 + 为可见性进行的屏幕尺寸修正”。新系统必须在 UI 中称为“扇区覆盖示意”，不能称为无线规划仿真结果。

### 3.4 交互流程

老项目交互链路如下：

```text
点击聚合点 -> 展开重叠节点
点击单个基站 -> 校验参数 -> 画白色方向扇区 -> 显示详情/高亮
点击白色扇区 -> 展开蓝色覆盖扇区
点击蓝色扇区 -> 收起蓝色覆盖扇区
点击其他基站或地图空白 -> 清除当前全部扇区
```

天线位置和部分参数可在地图位置弹窗、设备列表编辑抽屉中维护。老界面展示了机械下倾角、电子下倾角、垂直 3dB 波束宽度和水平方位角；地图位置弹窗实际只允许修改其中部分字段。

### 3.5 老项目已知问题

老实现有以下问题，不能直接搬运：

1. `cellSectorLayer` 和 `signalCoverageSectorLayer` 都是单例，新点击会移除旧图层，无法同时比较多个基站方向。附件中的历史记录也明确提出了多站信号范围同时观察需求。
2. 缺少任一必需参数时完全不画扇区，且没有告诉用户缺少什么。
3. 使用 `if (height && angle...)` 判断数值，合法的 0° 会被当成空值。
4. 对异常角度使用 `abs()` 或固定 500m/100m，会把错误配置伪装成看似正常的覆盖范围。
5. 物理半径和最低屏幕像素半径混在同一份变量里，地图量测得到的范围可能不是实际计算半径。
6. 水平覆盖角存在口径冲突：附件参数说明写“固定 120°”，老代码实际固定 115°。
7. 老模型本质上是“一台基站一组天线参数”，不能准确表达三扇区站或多小区设备。

## 4. 当前项目现状与差距

本节已按 `origin/main@f6ee9e43f` 重新核对。该次 `main` 更新合入自动归组和 PM 导出等修改，没有改动 GIS、Geo API、天线参数装配或参数模型，所以下述代码定位仍成立。

### 4.1 当前 Geo 数据链路

当前链路是：

```text
GISMapView
  -> useMapDevicesGeo
  -> topologyApi.getDevicesGeo
  -> GET /api/v1/devices/geo
  -> device.Handler.ListGeo
  -> DeviceService.ListGeo
  -> PgDeviceRepository.ListGeo
```

`GeoDevice` / `DeviceGeo` / `MapDevice` 当前只包含坐标、状态、分组、IP、MAC、PCI、UE 数和告警摘要，不含天线高度、方位角、下倾角或波束宽度。

`/devices/geo` 会随视口、状态和设备组筛选频繁请求，单次最多返回约 2000 个点。把完整扇区明细直接加入该接口，会增加列表 SQL、响应体和地图拖动时的网络成本。

### 4.2 当前天线参数能力

当前项目已有两类相关数据，但都不能直接满足 GIS：

1. `device_info.height` 是设备运维信息中的安装高度，可手动维护。
2. 设备详情通过 `device_parameters` 动态装配 `AntennaInfo`，包含 `Azimuth`、`Beamwidth`、`Downtilt`、`Height` 等字段。

主要缺口：

- `AntennaInfo` 没有区分机械下倾角和电子下倾角。
- 当前装配逻辑只识别不带编号的字段后缀，不能可靠表达 `Azimuth2/Azimuth3` 等多扇区参数。
- `Device.DeviceInfo.ElectronicDowntilt` 虽已进入参数模型，但当前定义为 `BOOLEAN`，实际映射到 `aldconfig...setTilt10` 状态，不能作为电子下倾角度数使用。
- `device_registrations` 虽有 `height/azimuth/tilt_angle/beam_width`，它属于注册预配置数据，不等于运行中设备的权威扇区数据。
- 目前没有扇区只读 API、字段来源说明、缺失状态或数据质量统计；参数修改可以复用现有 MML/参数任务链路，不应另造 GIS 写库入口。

### 4.3 运行库字段与数据核查

2026-07-10 对本机运行中的 PostgreSQL 主库进行只读核查，结果如下。

#### 现有表字段

| 表 | 已有相关字段 | 结论 |
|---|---|---|
| `devices` | `latitude`、`longitude` | 可直接作为扇区中心点 |
| `device_info` | `height`、`gps_height` | 只有安装高度候选；`gps_height` 不能替代天线挂高 |
| `device_registrations` | `latitude`、`longitude`、`height`、`azimuth`、`tilt_angle`、`beam_width` | 仅注册预配置数据，不是运行中设备扇区表 |
| `device_parameters` | `parameter_path`、`parameter_value` | 可以保存原始天线路径和值，但没有规范化业务列 |

当前数据库**不存在**以下表或直接字段：

- `device_antenna_sectors` 表。
- `mechanical_downtilt_deg`。
- `electronic_downtilt_deg`。
- `vertical_beamwidth_deg`。
- `horizontal_azimuth_deg`。
- `horizontal_beamwidth_deg`。
- `sector_no`、`field_sources`、`coverage_status`、`near_radius_m`、`far_radius_m`。

#### 当前数据覆盖率

| 数据项 | 运行库结果 |
|---|---|
| 未删除设备 | 33,722 台 |
| 有经纬度设备 | 33,722 台 |
| `device_info` 记录 | 2 条 |
| `device_info.height` 非空 | 0 条 |
| `device_registrations` | 0 条 |
| 存在 AntennaInfo 参数的设备 | 1 台 |

唯一有天线参数的设备保存了两组 `Azimuth/Beamwidth/Downtilt/Height`：

| 参数 | 扇区 1 | 扇区 2 |
|---|---:|---:|
| `Azimuth` | 0 | 0 |
| `Beamwidth` | 0 | 0 |
| `Downtilt` | 0 | 0 |
| `Height` | 2 | 5 |
| `HeightType` | AGL | AGL |

当前值不足以计算有效覆盖：波束宽度为 0，且没有数值型电子下倾角。

#### 接口字段与现有来源关系

| 拟新增接口字段 | 当前候选来源 | 当前是否可直接使用 |
|---|---|---|
| `latitude/longitude` | `devices.latitude/longitude` | 是 |
| `antenna_height_m` | `device_parameters` 的 `AntennaInfo.Height{N}`；注册阶段可参考 `device_registrations.height` | 有路径，但覆盖率极低 |
| `horizontal_azimuth_deg` | `AntennaInfo.Azimuth{N}` | 有路径，但目前仅 1 台且值为 0 |
| `mechanical_downtilt_deg` | `AntennaInfo.Downtilt{N}` | 仅是候选映射，需产品语义确认 |
| `electronic_downtilt_deg` | 当前无可信数值路径 | `Device.DeviceInfo.ElectronicDowntilt` 是 BOOLEAN 状态，不能直接当角度 |
| `vertical_beamwidth_deg` | `AntennaInfo.Beamwidth{N}` | 无法确认是垂直 3dB 波束宽度，当前值为 0 |
| `horizontal_beamwidth_deg` | 无明确独立字段 | 否；只能新增来源或使用明确标注的产品默认值 |
| `sector_no` | 路径后缀或 `{i}` 实例 | 需要归一化派生 |
| `near_radius_m/far_radius_m` | 天线参数公式 | 派生字段，当前数据无法有效计算 |
| `coverage_status/missing_fields/assumptions` | 服务层校验 | 新增派生字段 |
| `field_sources` | `device_parameters.parameter_path` 与产品映射 | 可由解析器随响应派生，不要求新表 |

结论：接口示例描述的是**目标规范化 DTO**，不是现有数据库字段的直接透传。首期不需要迁移复制数据，但必须先完成字段语义确认、路径归一化和数据完整率评估；否则接口虽然可以开发出来，但绝大多数设备只能返回 `incomplete`。

#### 测试服务器 API 核查

2026-07-10 使用测试账号通过现有只读 API 核查测试服务器，未触发参数同步或配置下发：

| 数据项 | 结果 |
|---|---:|
| 设备总数 | 17 台 |
| `/devices/geo` 返回设备 | 11 台 |
| Geo 设备中可装配 `AntennaInfo` | 10 台 |
| `device_info.height` 有值 | 0 台 |
| 存在数值型 `ElectronicDowntilt` | 0 台 |
| 当前参数组合可计算有效覆盖 | 0 台 |

测试服务器已经采集到两类原始路径：

1. `Device.DeviceInfo.AntennaInfo.Azimuth/Azimuth2/Azimuth3` 等编号字段。
2. `Device.DeviceInfo.CellConfig.1.AntennaInfo.Azimuth/Beamwidth/Downtilt/Height` 等实例字段。

但实际值存在以下问题：

- `Azimuth` 均为 0。0° 本身可能表示正北，但结合其他字段全为默认值，不能确认是现场配置。
- `Downtilt` 均为 0。
- `Height` 均为 0 或空。
- `Beamwidth` 为 0 或 360，不满足几何覆盖计算所需的有效范围 `0 < beamwidth < 180`。
- 未发现 `ElectronicDowntilt` 实际参数。
- 设备详情中的 `gps_height` 多数有值，但它是 GPS 海拔/高度，不能当作天线离地挂高。

因此当前能力应准确表述为：**参数模型和参数存储支持读取部分天线字段，但 GIS 业务尚未支持，测试服务器数据也不足以直接生成扇区覆盖。**

### 4.4 当前地图渲染能力

v1 `useOLMap.ts` 已有底图、设备聚合、Spiderfy、高亮、测距等独立图层，但没有扇区图层。页面传入的 `onDeviceClick` 目前为空，设备点击只锁定现有详情卡片。

本期只改 v1。v1 使用 OpenLayers 和共享 `GISMap/useOLMap`，适合新增独立的 OpenLayers 扇区图层控制器。计算、API 和查询状态仍放在 `frontend-core`，OpenLayers 类保留在 `webcode`，避免把 UI 引擎依赖带入共享业务层。

### 4.5 最新代码证据矩阵

| 代码位置 | 当前事实 | 对方案的约束 |
|---|---|---|
| `omcgo/internal/device/detail_dto.go` | `AntennaInfo` 只有一组字符串字段 | 不能作为多扇区 API 契约直接复用 |
| `omcgo/internal/device/detail_assembler.go` | 仅以 `HasSuffix("Azimuth")` 等匹配无编号字段 | `Azimuth2/3`、`Beamwidth2/3` 等需要新解析规则 |
| `omcgo/internal/device/device_param_pg_repository.go` | 已有按设备和路径前缀查询 | 新接口可复用参数仓储，不需要新增扇区仓储 |
| `omcgo/migrations/000001_init_schema.sql` | `(device_id, parameter_path varchar_pattern_ops)` 已有前缀索引 | 单设备按需解析具备数据库索引基础 |
| `omcgo/data/param-mappings/*.xml` | 多个产品已有编号天线标准路径，且多数为 `READ_WRITE` | 可复用现有参数任务修改；但仍需逐产品确认字段物理语义 |
| `omcgo/data/param-mappings/standard-model.xml` | `ElectronicDowntilt` 为 `BOOLEAN` | 不能映射为角度，也不能参与覆盖公式 |
| `omcgo/internal/device/device_repository.go` | `GeoDevice` 和 `ListGeo` SQL 不含天线字段 | 保持 Geo 主链路不变，点击后按设备加载 |
| `omcmb/frontend-core/src/types/map.ts` | `DeviceGeo`、`MapDevice` 不含扇区集合 | 新增独立 `AntennaSector` 类型，不污染点位类型 |
| `omcmb/webcode/src/pages/topology/GISMapView/index.tsx` | `onDeviceClick={undefined}` | 页面需接入选择设备和扇区查询状态 |
| `omcmb/webcode/src/components/GISMap/useOLMap.ts` | 已有设备、Spiderfy、测距等 VectorLayer，无扇区层 | 新扇区层应独立于 Cluster source，并支持稳定 Feature ID |

## 5. 方案比较

### 方案 A：最小复制老项目

做法：给 `device_info` 增加 4～6 个字段，全部附加到 `/devices/geo`，前端点击节点时画一个扇区。

优点：改动少，能较快复现老项目单站效果。

缺点：只能表达单扇区；Geo 大列表变重；多站对比仍无法实现；未来支持三扇区时需要再次迁移接口和数据。

结论：不推荐。

### 方案 B：基于原始参数动态规范化，按需加载（推荐）

做法：保留 `device_parameters` 为设备上报参数的唯一事实来源；新增解析器把产品标准路径动态组装成一台设备的 1～N 个扇区；Geo 列表保持轻量；点击设备时加载扇区详情；地图层按设备/扇区 ID 管理多个持久覆盖图层。

优点：满足老功能和附件中的多站对比需求；支持未来多小区；不会拖慢地图列表主链路；不复制现有参数，也不引入同步时序和来源冲突。

缺点：必须处理不同产品路径、编号规则和字段语义；每次首次点击需查询该设备参数，但可以通过索引和 React Query 缓存控制成本。

结论：推荐。

### 方案 C：完整无线覆盖规划

做法：除天线参数外，再引入频段、发射功率、天线增益、地形、建筑物和传播模型，计算 RSRP 等值覆盖。

优点：接近专业网规能力。

缺点：远超本需求；当前数据质量和地图底图都不足以支持可信预测。

结论：本期明确不做。当前扇区只表示几何覆盖示意。

## 6. 推荐设计

### 6.1 后端领域模型

首期新增的是服务层 DTO 和解析器，不新增 `device_antenna_sectors` 表：

```go
type AntennaSectorDTO struct {
    SectorNo              int
    CellIdentity          string
    AntennaHeightM        *float64
    MechanicalDowntiltDeg *float64
    ElectronicDowntiltDeg *float64
    VerticalBeamwidthDeg  *float64
    HorizontalAzimuthDeg  *float64
    HorizontalBeamwidthDeg *float64
    FieldSources          map[string]string
    CoverageStatus        string
    MissingFields         []string
    Assumptions           []string
}
```

`FieldSources` 保存实际命中的 `parameter_path` 或产品默认配置名称，便于解释字段来自哪里。近点、远点和状态均为服务层派生值，不落库。

只有产品后续明确要求“人工值不下发设备、且长期覆盖设备上报值”时，才新增 `device_antenna_sector_overrides`。该表只保存覆盖字段和审计信息，不复制全部设备当前值；读取优先级为：

```text
manual override > device_parameters report > explicit product default
```

### 6.2 参数归一化

不同产品的 TR069 路径不能散落在 GIS handler 或前端中。解析器读取已映射到标准路径的 `device_parameters`，再按受测试保护的规则组装扇区。

首期兼容规则：

- 无编号的 `Azimuth/Downtilt/Height/Beamwidth` 映射为 `sector_no=1`。
- `Azimuth2/Downtilt2/Height2` 映射为 `sector_no=2`，编号 3 同理。
- 当前 `ElectronicDowntilt` 是布尔状态，不参与角度计算。只有产品模型新增并验证数值型电子下倾角路径后，才可映射到 `electronic_downtilt_deg`。
- 不允许用 GPS 海拔 `gps_height` 代替天线挂高。两者物理含义不同。
- `CellConfig.{i}.AntennaInfo.*` 按实例号形成扇区；不能与设备级编号字段无条件拼接，产品映射需定义两者的优先关系。
- `device_registrations` 不作为运行时自动兜底，避免注册预配置掩盖设备真实缺失。

### 6.3 API 设计

保持 `/devices/geo` 轻量，新增：

```http
GET /api/v1/devices/{deviceId}/antenna-sectors
```

该接口从现有坐标和 `device_parameters` 即时解析，不修改 `/devices/geo`，也不以新表或数据迁移为前置条件。首期为只读接口。

读取响应示例：

```json
{
  "device_id": "...",
  "latitude": 31.9773,
  "longitude": 118.763,
  "sectors": [
    {
      "sector_no": 1,
      "cell_identity": "...",
      "antenna_height_m": 16,
      "mechanical_downtilt_deg": 8,
      "electronic_downtilt_deg": 6,
      "vertical_beamwidth_deg": 8,
      "horizontal_azimuth_deg": 30,
      "horizontal_beamwidth_deg": 120,
      "near_radius_m": 49.24,
      "far_radius_m": 90.74,
      "coverage_status": "complete",
      "missing_fields": [],
      "assumptions": []
    }
  ]
}
```

如需编辑天线参数，应复用现有 MML/参数下发能力并遵守产品参数的 `READ_WRITE` 属性，不在 GIS 中直接更新 `device_parameters`。本期 GIS 已为方位角和机械下倾接入该异步任务链路；BOOLEAN 的 `ElectronicDowntilt`、垂直波瓣宽度及 READ_ONLY 字段不得开放为角度编辑项。人工覆盖表及其写接口作为后续独立需求评审。

### 6.4 计算规则

后端作为权威计算方，返回近点和远点。前端保留纯函数用于编辑预览，两端使用同一组 JSON 契约样例做黄金测试。

有效计算条件：

```text
h > 0
0 <= azimuth < 360
0 < horizontalBeamwidth < 180
0 < totalTilt - verticalBeamwidth/2 < 90
0 < totalTilt + verticalBeamwidth/2 < 90
nearRadius < farRadius
```

计算：

```text
nearRadius = h / tan(totalTilt + verticalBeamwidth/2)
farRadius  = h / tan(totalTilt - verticalBeamwidth/2)
```

与老项目不同：

- 不对非法结果取绝对值。
- 不用 500m/100m 冒充计算结果。
- 可设置显示安全上限，例如 10km；超过上限时返回 `out_of_range`，UI 显示警告并按上限裁剪，但保留原始计算值。
- 水平覆盖角优先使用实际字段；缺失时可使用产品级默认值。附件要求是 120°，老代码是 115°，实施前必须将默认值锁定为产品配置，不能继续写死在前端。

经纬度扇区边界使用球面目的点公式计算，不直接在 Web Mercator 平面上用 `center + meter` 拼坐标。这样在不同纬度、在线/离线底图下方向和距离口径保持一致。

### 6.5 前端共享业务层

在 `frontend-core` 增加：

- `AntennaSector`、`SectorCoverage`、`SectorCoverageStatus` 类型。
- `topologyApi.getAntennaSectors()`。
- `useDeviceAntennaSectors(deviceId)` Hook。
- 扇区几何纯函数：输入中心经纬度、近/远半径、方位角、水平波束宽度，输出 WGS84 Polygon 坐标。
- 多扇区选择状态：`deviceId:sectorNo` 作为稳定键。

共享层不依赖 OpenLayers、Ant Design 或任何皮肤组件。

### 6.6 地图图层与多站对比

v1 OpenLayers 建议新增两个独立 VectorLayer：

1. `sectorDirectionLayer`：固定视觉尺寸的方向提示，位于设备点上方。
2. `sectorCoverageLayer`：真实近/远半径对应的环形扇区，位于设备点下方、底图上方。

图层中的 Feature 使用以下稳定 ID：

```text
direction:{deviceId}:{sectorNo}
coverage:{deviceId}:{sectorNo}
```

不能再像老项目一样只保存一个 `cellSectorLayer`。地图允许固定多个覆盖扇区：

- 点击设备点：加载并显示该设备所有完整扇区的方向提示。
- 点击方向提示：展开/收起该扇区覆盖范围，并默认固定在地图上。
- 点击其他设备不会自动清除已经固定的覆盖范围。
- 工具栏提供“清除全部覆盖”命令，并显示当前固定数量。
- 建议最多固定 20 个扇区；达到上限时提示先清理，避免 Canvas 过度绘制。
- 低缩放级别不绘制覆盖多边形；建议 `zoom >= 13` 才显示真实范围。
- 进入测距模式、切换大范围筛选或设备被移出当前可见权限时，清理相关临时方向提示；固定覆盖是否保留由明确规则处理，不能留下无权限数据。

### 6.7 用户交互

推荐流程：

```text
点击设备/Spiderfy 成员
  -> 显示现有设备卡片
  -> 加载扇区数据
  -> 完整：画方向提示
  -> 不完整：卡片展示“扇区参数不完整”及缺失字段

点击方向提示
  -> 展开覆盖范围并固定

继续点击周边设备
  -> 保留已固定覆盖
  -> 可同时比较多个 BTS/小区方向

点击覆盖范围
  -> 选中对应扇区，在详情卡片显示参数、近点/远点和数据来源

点击“清除全部覆盖”
  -> 清空固定覆盖和方向提示
```

设备卡片新增“扇区”区块，展示：扇区序号、小区标识、方位角、水平波束宽度、机械/电子下倾角、垂直波束宽度、天线高度、近点、远点、数据来源和更新时间。

缺数据时不得静默失败。例如：

```text
扇区覆盖暂不可用：缺少电子下倾角、垂直 3dB 波束宽度。
```

本期编辑能力位于 GIS 卡片，限方位角和机械下倾两个字段，并提供本地扇面预览。保存后仅显示“任务已提交”，扇区查询标记为过期但不立即用旧上报值覆盖预览；取消编辑则恢复原始扇面。

## 7. 数据质量与降级策略

| 数据状态 | 地图行为 | 卡片行为 |
|---|---|---|
| 全部参数完整 | 显示方向和覆盖范围 | 展示计算值及来源 |
| 只有方位角/水平角 | 显示虚线方向扇区，不显示距离覆盖 | 明确缺少高度/下倾角/垂直波束 |
| 缺方位角 | 不绘制扇区 | 明确提示缺方位角 |
| 角度非法 | 不绘制真实覆盖 | 显示非法字段和值 |
| 使用产品默认水平角 | 可绘制，样式加“估算”标识 | `assumptions` 显示默认值来源 |
| API 失败 | 保留设备点，不显示扇区 | 显示可重试错误，不影响主地图 |

不建议为生产设备自动写入测试默认值。默认值只用于展示估算，并必须随响应携带来源和假设。

## 8. 性能与安全边界

- Geo 点位查询不 JOIN 或扫描天线参数；扇区仅在选择设备后按需查询。
- 扇区详情按设备查询并由 React Query 缓存，建议 `staleTime=5min`。
- 固定覆盖最多 20 个扇区；每个 Polygon 弧线采样 32～64 个点即可。
- 覆盖层不进入设备 Cluster source，避免扇区 Feature 被当成设备聚合。
- 视口外固定扇区可暂不渲染，但保留选择状态；回到视口后恢复。
- 设备数据权限必须同时作用于扇区读取和更新接口。设备失去可见权限后，前端必须移除缓存和图层。
- 首期接口只读，不产生审计写入；后续参数下发沿用现有任务和审计链路。

## 9. 分阶段实施建议

### Phase 1：只读解析与单站闭环

- 先确认 `Downtilt`、`Beamwidth` 和 `ElectronicDowntilt` 在各产品模型中的物理语义与单位。
- 对运行库执行数据完整率扫描，确定可形成有效扇区的设备数量。
- 新增多扇区 DTO、参数解析器、只读 API、计算服务和数据校验，不新增业务表。
- 为无编号、编号 2/3 和 `CellConfig.{i}` 三类路径建立单元测试。
- v1 支持点击站点显示方向/覆盖，参数缺失有提示。

交付结果：可以可靠复现老项目单站扇区，但修复静默失败和非法默认半径问题。

### Phase 2：v1 多扇区与多站对比

- 支持 `sector_no=1..N` 及编号参数路径。
- 支持多个固定覆盖、清除全部和数量上限。
- 设备信息编辑抽屉增加扇区维护和字段来源展示。

交付结果：满足附件中“同时观察多个站信号范围”的业务诉求。

### Phase 3：数据治理与可选覆盖

- 扇区参数完整率统计和缺失设备筛选。
- 评审批量导入/导出和人工覆盖是否需要独立覆盖表。
- 产品级水平波束默认值配置。

不在本期：地形、建筑物、传播模型、RSRP 预测、UE 实时落点与覆盖归属判定。

## 10. 验收用例

### 10.1 计算与方向

1. 方位角 0° 指向正北，90° 指向正东，180° 指向正南，270° 指向正西。
2. 使用已知参数校验近点/远点公式，Go 与 TypeScript 黄金样例完全一致。
3. 合法 0° 方位角不能被当作空值。
4. `totalTilt - beamwidth/2 <= 0` 时返回非法状态，不生成伪覆盖。
5. 水平角采用产品配置；没有配置时明确标注 120° 假设，不得暗中使用 115°。

### 10.2 地图交互

1. 点击单站后显示其全部有效方向扇区。
2. 点击方向扇区后显示近点到远点之间的覆盖区域。
3. 连续固定至少 3 个相邻基站覆盖，前一个不会被后一个清除。
4. 同站三个扇区可同时显示，方向和扇区编号正确。
5. 聚合点先 Spiderfy，再选择具体设备，不能把聚合中心当作扇区中心。
6. 搜索定位、测距、筛选、视口动态加载和扇区图层之间没有竞态或残留。
7. 参数缺失、接口失败、权限不足都有可理解提示，设备点仍可正常使用。

### 10.3 性能和权限

1. 未选择设备时，新增能力不显著增加 `/devices/geo` 响应体和 SQL 耗时。
2. 固定 20 个扇区时地图平移和缩放保持可操作。
3. 无设备查看权限的用户无法读取或更新对应扇区。
4. 只读接口沿用设备查看权限；后续参数修改必须复用现有任务权限和审计链路。

## 11. 预计影响文件

以下仅是实施范围，不代表本次已修改：

### 后端

- `omcgo/internal/device/antenna_sector_dto.go`
- `omcgo/internal/device/antenna_sector_assembler.go`
- `omcgo/internal/device/antenna_sector_service.go`
- `omcgo/internal/device/device_handler.go`
- 产品参数映射（仅在语义确认后需要修正）与对应测试

### 共享前端

- `omcmb/frontend-core/src/types/map.ts`
- `omcmb/frontend-core/src/services/api/topologyApi.ts`
- `omcmb/frontend-core/src/hooks/api/useTopology.ts`
- `omcmb/frontend-core/src/utils/sectorCoverage.ts`
- 中英文国际化资源与黄金计算样例

### v1

- `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`
- `omcmb/webcode/src/components/GISMap/index.tsx`
- `omcmb/webcode/src/components/GISMap/useOLMap.ts`
- 新增独立扇区图层控制模块，避免继续扩大 `useOLMap.ts`

## 12. 待产品确认

实施前需要确认两项业务口径：

1. **缺少实际水平波束宽度时，默认值是否统一为 120°。**附件参数说明是 120°，老代码是 115°；本文推荐产品配置默认 120°。
2. **Phase 1 是否必须立即支持多站固定。**本文推荐解析契约从第一天按多扇区设计，但可以把多站固定交互放入 Phase 2，以降低首期联调风险。
3. **`Beamwidth` 的物理口径。**现有路径名称不能证明它一定是垂直 3dB 波束宽度；必须按产品型号确认，未确认前只能画方向示意，不能计算距离覆盖。
4. **电子下倾角来源。**当前 `ElectronicDowntilt` 是布尔状态，若覆盖公式必须包含电子下倾角，需要设备侧提供新的数值参数路径或产品映射。

除以上两点外，本文建议的其余边界可以直接进入实施计划。

## 13. 明确不在本期范围

- `omcmb/webcode-v2` GIS 页面。
- `omcmb/webcode-v3` GIS 页面。
- 三皮肤 GIS 交互一致性改造。
- GIS 左侧筛选栏默认收起。

后续如需补齐 v2/v3，应单独评估各自地图引擎和页面结构，不与本次 v1 实施捆绑。

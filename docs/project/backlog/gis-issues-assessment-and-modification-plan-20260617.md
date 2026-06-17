# GIS Issue 评估与逐项修改方案（2026-06-17）

## 1. 目标与范围

本文档用于统一 GIS 相关问题的 Issue 口径，并给出按现有架构可落地的逐项修改方案。

范围包含 4 个主 Issue（与测试问题汇总 20260601 第 153-156 条对齐）。其中 Issue A（地图页面渲染卡住 + 中心点策略统一）本轮一并更新；中心点这部分重点补充“首屏中心点由 metadata / fallback 统一决策，页面层不再硬编码”的描述。本次重点整理 Issue B-D 的分析、归因与实施顺序。

1. 地图页面渲染卡住：`/tiles-metadata` 返回 500 导致加载阻塞（P1, Bug）
2. 设备状态相关信息异常，未正常显示：在线/在线未激活/离线个数（P1, Bug）
3. 地图相关节点加载不太合理，渲染逻辑与原先的分层渲染、层层展开等不符合（P1, Bug）
4. 搜索的时候定位到节点的渲染逻辑与之前的版本存在差异，未能不断放大然后定位到具体节点（P1, Bug）

> 说明：上面 4 条分别对应测试问题汇总 20260601 的第 153、154、155、156 条。
> 说明：这里是把你截图里的 4 条主问题与此前 GIS 文档中的上下文合并整理，而不是简单覆盖原文。
> 说明：中心点策略统一已并入 Issue A，和 `/tiles-metadata` 一起作为同一条地图初始化链路的问题处理。

---

## 2. 架构对齐原则

- 后端：`handler -> service -> repository`，语义收敛在 service，SQL 细节在 repository。
- 前端：`frontend-core` 负责数据契约和 hook，`webcode` 负责页面编排。
- GIS 核心交互应集中在 `webcode/src/components/GISMap/useOLMap.ts`，页面层不承载复杂策略。
- “未分组设备”属于领域规则，不应在多个入口以不同方式解释。

---

## 3. Issue 逐项评估与修改

## Issue A（P1, Bug）
### 标题
地图页面渲染卡住 + 中心点策略统一：`/tiles-metadata` 慢 500 导致加载阻塞，且无离线地图时首屏中心点应统一由 metadata / fallback 决策

### Issue 提交稿（可直接用于 GitHub Bug 模板）

#### 现状描述
GIS 地图页面在 `/tiles-metadata` 返回 500 且响应较慢时，会长时间停留在 loading，用户感知为页面卡住。

同时，无离线地图场景下的首屏中心点仍存在页面层硬编码与配置层分散的问题，导致初始化链路虽能降级，但首屏落点不统一，容易出现“能打开但第一屏看不到设备”的体验偏差。这个问题本质上不是单纯“默认值”问题，而是中心点决策链缺少单一入口：页面层、metadata 配置、地图初始化三处在共同参与首屏落点判断。

#### 复现步骤
1. 启动前端并进入 GIS 地图页面。
2. 让 `GET /tiles-metadata` 返回慢 500（例如 5-30 秒后返回）。
3. 观察页面加载状态与地图可操作时间。

#### 期望行为
- `/tiles-metadata` 在 500、404、超时、JSON 解析异常下都应在最大等待时间内结束。
- 页面应快速降级到默认 metadata + 在线 OSM，不应长时间转圈。
- 无离线地图场景下，中心点应统一由 metadata / fallback 决策，不应由页面层硬编码抢占。
- 中心点策略优先级应明确：`metadata.center` > 默认 metadata 的中心点 > `MAP_CONFIG.defaultCenter`。

#### 实际行为
- 接口最终返回 500，但在响应返回前 metadataLoading 长时间不结束。
- GIS 地图初始化链路被阻塞，页面持续转圈，直到接口错误返回后才渲染。
- 页面层仍存在硬编码中心点，导致不同初始化路径下首屏落点不一致。
- 在元数据成功、元数据降级、页面层硬编码三种路径下，首屏中心点的最终来源不统一，用户体验出现漂移。

#### 影响面
- Staging、本地开发均可复现。
- 影响功能域：F06 GIS / Topology。
- 影响用户：所有进入 GIS 地图页的用户。

#### 初步定位
- `omcmb/webcode/src/components/GISMap/useMapConfig.ts`：原先未设置 metadata 请求最大超时。
- `omcmb/webcode/src/components/GISMap/useOLMap.ts`：初始化依赖 metadataLoading 结束。
- `omcmb/webcode/vite.config.ts`：原先 `/tiles-metadata` 代理未配置显式超时。
- `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`：存在 `defaultCenter={[26, -13]}` 的页面层硬编码。

#### 严重等级
P1（主流程受阻，页面长时间不可操作）

#### 关联
- 问题来源：测试问题汇总20260601 文档第 153 条。
- 关联 Issue：#468
- Issue 链接：https://github.com/569423176-sketch/goomc/issues/468
- 关联文档：本文件 Issue A。
- 建议分支：`fix/gis-tiles-metadata-fallback`。

### 当前评估结论
确认存在，属于 GIS 初始化链路的可用性问题，应作为第一优先级处理。

> 备注：本轮整理不再追加 Issue A 的新分析或修改建议，以下 Issue B-D 为当前重点。

### 现场现象（已复现）
- 请求：`GET http://localhost:3000/tiles-metadata`
- 状态：`500 Internal Server Error`
- 结果：地图页面长时间停留在 loading，用户感知为“页面卡住”

### 代码证据（现状）
- 元数据请求入口：
  - `omcmb/webcode/src/components/GISMap/useMapConfig.ts`
- 地图初始化依赖元数据与瓦片可用性结果：
  - `omcmb/webcode/src/components/GISMap/useOLMap.ts`
- `tiles-metadata` 前端代理配置：
  - `omcmb/webcode/vite.config.ts`

### 根因
- `tiles-metadata` 异常（500）场景下，降级路径和加载状态结束时机不够稳健。
- GIS 初始化依赖链较长（metadata -> tiles availability -> map init），任何环节未及时降级都可能表现为“卡住”。
- 中心点决策分散在页面层、metadata 默认值和地图组件内部，缺少单一决策入口。

### 修改方案（按分层）

**文件清单与修改顺序**：

1. **创建新文件：utils/mapValidation.ts**
   - 实现 `calculateCenterFromDevices(devices)` - 从设备数据计算中心点
   - 实现 `parseEnvCenter()` - 解析 VITE_MAP_DEFAULT_CENTER 环境变量
   - 支持自动缩放级别调整

2. **修改 useMapConfig.ts**：
  - 为 metadata 请求增加硬超时（默认 2500ms，可通过 `VITE_MAP_METADATA_TIMEOUT_MS` 调整）。
  - 将 `500/404/超时/JSON 解析异常` 统一收敛到降级出口。
  - 返回 `isUsingDefault: boolean` 标志降级状态
  - 确保状态始终收敛到终态：`loading=false`，并记录错误类别。
  - **失败冷却**：记录最近一次失败时间戳，30s 窗口内组件再次挂载直接复用 `DEFAULT_METADATA`，
    避免用户在 metadata 持续不可用时每次进入地图页都再等满超时。冷却过期后自动重试，保留恢复能力。

3. **修改 GISMapView/index.tsx**：
   - 移除页面层 `defaultCenter={[26, -13]}` 硬编码
   - 实现 4 层优先级决策逻辑（参见上文）
   - 使用 `useMemo` 计算 `initialMapCenter` 和 `initialZoom`
   - 查询 `/api/v1/devices/geo` 的前 1000 个设备用于计算
   - 当 metadata 不可用但有设备数据时，调用 `calculateCenterFromDevices`
   - 将计算结果传给 GISMap 组件

4. **修改 useOLMap.ts**：
  - 保持"metadataLoading 结束后初始化"逻辑，但依赖于上一步确保 loading 不无限挂起。
  - 在 metadata 降级场景下直接走在线 OSM 配置。
  - 接受 `defaultCenter` 和 `defaultZoom` 参数用于初始化

5. **修改 constants.ts**：
  - 添加 `parseDefaultCenter()` 函数读取 VITE_MAP_DEFAULT_CENTER
  - 支持验证坐标有效性（经度 -180~180，纬度 -90~90）
  - 更新 MAP_CONFIG.defaultCenter 为全球通用默认值 [0, 20, 2]

6. **修改 vite.config.ts**（接口改造单独纳入本 Issue）：
  - 为 `/tiles-metadata` 增加 `timeout/proxyTimeout: 2500`（与客户端超时对齐）

7. **文档更新**：
   - topology_GISMap_design.md: 已添加第 2.6 节"中心点决策链"
   - gis-issues-assessment-and-modification-plan-20260617.md: 本文档已补充
   - offline-map-*.md: 同步更新中心点决策链说明

### 原修改方案（保留用于参考）
1. `useMapConfig.ts`：
  - 为 metadata 请求增加硬超时（建议 2500ms）。
  - 将 `500/404/超时/JSON 解析异常` 统一收敛到降级出口。
  - 确保状态始终收敛到终态：`loading=false`，并记录错误类别。
2. `useOLMap.ts`：
  - 保持“metadataLoading 结束后初始化”逻辑，但依赖于上一步确保 loading 不无限挂起。
  - 在 metadata 降级场景下直接走在线 OSM 配置。
3. `GISMapView/index.tsx`：
   - 移除页面层 `defaultCenter` 硬编码，让中心点回到统一决策链。
4. `vite.config.ts`（接口改造单独纳入本 Issue）：
  - 为 `/tiles-metadata` 增加 `timeout/proxyTimeout`（默认 2500ms，且支持环境变量统一配置）。
5. 验证回归：
  - 构造 `500/404/timeout/json_parse` 四类场景。
  - 验证页面在阈值内结束转圈并可操作。

### 验收标准
- 当 `/tiles-metadata` 返回慢 500 时，页面在最大超时阈值内结束 loading 并可操作。
- 无离线地图时自动降级在线底图，地图层与设备层可正常渲染。
- 无离线地图时首屏中心点统一，页面不再因硬编码导致第一屏看不到设备。
- 日志可区分 `http_xxx`、`timeout_xxx`、`json_parse_error`、`network_error_xxx`。
- 提供 500/404/timeout/json_parse 四场景回归记录。

### 风险与回滚
- 风险：降级过快可能掩盖真实离线地图配置问题。
- 回滚：保留错误提示与诊断日志，允许按开关恢复严格模式。

### 工作量预估
- 开发 0.5~1.0 人日
- 回归验证 0.5 人日

### 关联子问题：无离线地图场景中心点策略统一

该问题不单列为主 Issue，而是作为 Issue A 的补充子问题一起处理，因为它与 GIS 初始化链路、默认 metadata 降级和首屏渲染是同一条路径。本轮已并入 Issue A 的修复范围。

#### 简要结论
- 无离线地图时，当前中心点存在硬编码与配置分散问题，容易导致首屏看不到设备。
- 该问题与 Issue A 同属“地图初始化与首屏可用性”范畴，建议合并在同一修复 PR 中。
- 需要补齐的是：让首屏中心点只走一条决策链，避免页面、hook、默认配置分别决定。
#### 中心点多地区适配方案（核心设计）

**4层优先级决策链**（从高到低）：

```
优先级 1: tiles.json 中的 center 字段
         (离线地图元数据返回的中心点 - 最优先)
         ↓ (不存在或异常)
         
优先级 2: /api/v1/devices/geo 设备数据计算
         (根据实际设备位置自动计算中心点 - 智能适配)
         ├─ 计算设备 bounds（经纬度范围）
         ├─ 中心点 = [(minLng+maxLng)/2, (minLat+maxLat)/2]
         ├─ 缩放级别根据设备分布范围自动调整
         │  - 范围 > 50°：zoom=4
         │  - 范围 > 20°：zoom=6
         │  - 范围 > 5°：zoom=8
         │  - 范围 > 1°：zoom=11
         │  - 其他：zoom=13
         └─ 处理空设备列表时回落到优先级 3
         ↓ (无设备数据)
         
优先级 3: 环境变量 VITE_MAP_DEFAULT_CENTER
         (多地区部署时的兜底配置，格式: "lng,lat,zoom")
         ↓ (未配置)
         
优先级 4: 代码内置默认值
         (全球通用默认值 [0, 20, 2])
```

**多地区部署示例**（.env 配置）：

```bash
# .env.development（赞比亚 - Zambia）
VITE_MAP_DEFAULT_CENTER=28.221,-14.607,6

# .env.staging（中国 - China）
VITE_MAP_DEFAULT_CENTER=104.0,35.0,4

# .env.production（其他地区 - 由设备数据决定）
# 不设置，使用全球默认值
# 优先从实际设备数据自动计算中心点
```

**实现方式**：

1. **useMapConfig.ts**：
  - 负责加载 tiles.json 元数据，2500ms 超时
   - 返回 `isUsingDefault: boolean` 标志是否使用默认值
   - 成功时返回元数据中的 center；失败时返回 null

2. **utils/mapValidation.ts**（新增工具函数）：
   - `calculateCenterFromDevices(devices)`: 从设备数据计算中心点和缩放级别
   - `parseEnvCenter()`: 解析 VITE_MAP_DEFAULT_CENTER 环境变量

3. **GISMapView/index.tsx**：
   - 移除硬编码的 `defaultCenter={[26, -13]}`
   - 实现 4 层优先级决策链逻辑
   - 根据 metadata/devices/env 计算最终的 initialMapCenter
   - 将计算结果传给 GISMap 组件

4. **constants.ts**：
   - 支持 VITE_MAP_DEFAULT_CENTER 环境变量解析
   - MAP_CONFIG.defaultCenter 作为最后一层兜底

**优势**：
- ✅ 自动适配部署地区（无需手动修改代码）
- ✅ 基于真实设备数据智能定位
- ✅ 多地区部署只需环境变量配置
- ✅ 页面层无硬编码坐标
- ✅ 离线地图和在线地图无缝切换
#### 策略说明
- 页面层不再直接传 `defaultCenter` 参与首屏落点决定。
- `useMapConfig` 负责把服务端 metadata 转成可用配置，并在异常时回落到默认 metadata。
- `useOLMap` 只消费上游配置结果，不再自行引入额外中心点规则。
- 当离线地图不可用时，首屏中心点应回落到默认 metadata 的中心点或在线底图默认中心，不应由页面层再补一个独立坐标。

#### 代码证据
- `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`：`defaultCenter={[26, -13]}` 硬编码。
- `omcmb/webcode/src/components/GISMap/useMapConfig.ts`：默认 metadata 中的 center/bounds。
- `omcmb/webcode/src/components/GISMap/useOLMap.ts`：初始化时使用 center/config 进行首屏定位。

---

## Issue B（P1, Bug）
### 标题
设备状态相关信息异常，未正常显示：在线/在线未激活/离线个数（伴随设备组口径问题）

### Issue 提交稿（可直接用于 GitHub Bug 模板）

#### 现状描述
GIS 地图页面中，设备状态统计区域显示异常，在线/在线未激活/离线个数未正确反映当前设备数据；同时设备组筛选、默认组/未分组语义与统计口径存在联动问题。

#### 复现步骤
1. 进入 GIS 地图页面。
2. 保持默认筛选或切换设备组/状态筛选。
3. 观察设备状态统计值是否与地图点位和接口返回一致。

#### 期望行为
- 在线、在线未激活、离线数量应与后端统计口径一致。
- 状态计数应随设备组/筛选条件变化而准确更新。
- 设备组树的“默认组 / 未分组 / 全选”语义应与 geo/stats 两条请求保持一致。

#### 实际行为
- 设备状态相关信息显示异常，出现统计值为 0 或与点位不一致的情况。
- 当设备组选中逻辑涉及“默认设备组 / 未分组设备”时，统计和列表口径更容易分叉。

#### 影响面
- Staging、本地开发均可复现。
- 影响功能域：F06 GIS / Topology。
- 影响用户：所有查看 GIS 状态统计的用户。

#### 初步定位
- `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`：状态统计数据来源与展示处。
- `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`：设备组树、`selectedGroupIds`、`isAllSelected` 与统计请求的联动。
- `omcgo/internal/device/device_repository.go`：GeoStats 状态统计语义。
- `omcgo/internal/device/device_info_pg_repository.go`：设备列表中未分组语义参考实现。

#### 严重等级
P1（主流程数据可信度问题）

#### 关联
- 问题来源：测试问题汇总20260601 文档第 154 条。
- 建议分支：`fix/gis-status-count-align`。

### 当前评估结论
确认存在，且是后端语义分叉问题，不是单纯前端展示问题。

### 代码证据（现状）
- 前端 geo 与 stats 参数构造不一致：
  - `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`
- 后端 GeoStats 直接按 `dg.id in group_ids`，未处理 `DefaultLevel2GroupID` 特殊语义：
  - `omcgo/internal/device/device_repository.go` (`GetGeoStats`)
- 设备列表仓储已有正确“未分组 = NOT EXISTS device_group_members”语义：
  - `omcgo/internal/device/device_info_pg_repository.go`
- 伪组常量定义：
  - `omcgo/global/consts.go` (`DefaultLevel2GroupID`)

### 根因
- “未分组设备”语义未在 Geo 链路复用，导致接口口径与设备列表口径分离。
- 前端请求参数在两个 hook 调用点采用不同策略，放大了口径差异。
- 设备组树存在“全选后 geo 请求走 undefined，但 stats 仍传 groupIds”的路径，导致组选中场景下统计与点位口径分叉。
- `GetTreeWithCounts` 对 `DefaultLevel2GroupID` 的“未分组”语义与 `GetGeoStats` 的 group 过滤口径未完全统一。

### 修改方案（按分层）
1. 后端 service 层新增统一 group 过滤语义归一逻辑（包含 `DefaultLevel2GroupID` 分支）。
2. 后端 repository 层统一改造 Geo 查询：
   - `ListGeo`
   - `GetGeoStats`
   目标：与 `device_info_pg_repository` 保持一致语义。
3. 前端页面层统一构造 geo/stats 的 groupIds 参数（全选策略与部分选中策略一致）。
4. 设备组树/未分组设备的计数与选择语义统一，避免“组选中=全部”时 stats 与 geo 分叉。

### 验收标准
- 全选、仅未分组、仅默认组、组合选择下，`/devices/geo` 与 `/devices/geo/stats` 结果口径一致。
- 左侧状态数与地图点位一致，不再出现“有点位但状态全 0”。
- 新增后端回归测试 + 前端联调验证记录。

### 风险与回滚
- 风险：修改 SQL 过滤语义可能影响历史“默认组”统计习惯。
- 回滚：保留变更前查询路径为 feature flag 或单提交回滚点。

### 工作量预估
- 开发 1.0~1.5 人日
- 联调+验证 0.5 人日

---

## Issue C（P1, Bug）
### 标题
地图相关节点加载不太合理，渲染逻辑与原先的分层渲染、层层展开等不符合

### Issue 提交稿（可直接用于 GitHub Bug 模板）

#### 现状描述
GIS 地图节点加载后，当前渲染逻辑与原先的分层渲染、层层展开行为不一致，用户在不同 zoom 下看到的聚合层次变化不明显。

#### 复现步骤
1. 进入 GIS 地图页面并加载设备数据。
2. 在较低 zoom 到中高 zoom 之间缩放地图。
3. 观察节点聚合、展开和层次变化。

#### 期望行为
- 节点应随 zoom 逐步分层展开。
- 聚合距离与展示粒度应符合原先版本的视觉分层逻辑。

#### 实际行为
- 渲染逻辑与原先的分层渲染、层层展开不符。
- 中间 zoom 段层次不明显，节点聚合变化不连续。

#### 影响面
- Staging、本地开发均可复现。
- 影响功能域：F06 GIS / Topology。

#### 初步定位
- `omcmb/webcode/src/components/GISMap/useOLMap.ts`：聚合距离与缩放策略。
- `omcmb/webcode/src/components/GISMap/constants.ts`：聚合与 zoom 配置。
- `omcmb/original-omc/OMCWebServer/src/main/webapp/js/ol/ol-topo-helper.js`：旧版分层渲染参考。

#### 严重等级
P1（核心地图展示逻辑偏差）

#### 关联
- 问题来源：测试问题汇总20260601 文档第 155 条。
- 建议分支：`fix/gis-cluster-layering-align`。

### 当前评估结论
确认存在，属于地图分层渲染逻辑偏差，应单独作为 Issue 处理。

### 代码证据（现状）
- 聚合距离策略仅 3 档，缩放中段层次不足：
  - `omcmb/webcode/src/components/GISMap/useOLMap.ts` (`updateClusterDistance`)
- 参考旧版行为：
  - `omcmb/original-omc/OMCWebServer/src/main/webapp/js/ol/ol-topo-helper.js`

### 根因
- 聚合和分层渲染逻辑集中在 `useOLMap`，但策略已退化为少档/单段实现。
- 页面与 hook 间存在能力暴露但未统一使用的路径（行为不一致）。

### 修改方案（按模块）
1. `constants.ts` 补齐缩放档位与动画配置。
2. `useOLMap.ts`：
   - 聚合距离按更细档位动态调整。
   - 恢复分层渲染与层层展开效果。
3. 增加交互回归测试（缩放层级、cluster 命中）。

### 验收标准
- zoom 6~15 过程中聚合呈“层层展开”。
- 节点在中高 zoom 段能呈现出清晰的层次变化。
- 聚合与单点切换符合原版本的视觉逻辑。

### 风险与回滚
- 风险：动画链路变复杂，可能引入时序抖动。
- 回滚：保留 smooth 单段动画开关作为应急兜底。

### 工作量预估
- 开发 1.5~2.0 人日
- 交互回归验证 0.5 人日

---

### Issue D（P1, Bug）
### 标题
搜索的时候定位到节点的渲染逻辑与之前的版本存在差异，未能不断放大然后定位到具体节点

### Issue 提交稿（可直接用于 GitHub Bug 模板）

#### 现状描述
GIS 地图搜索定位时，当前渲染逻辑无法像旧版本那样不断放大并最终定位到具体节点，搜索体验与原版本不一致。

#### 复现步骤
1. 在 GIS 地图页搜索一个目标设备。
2. 点击搜索结果。
3. 观察地图是否逐级放大并精确定位到具体节点。

#### 期望行为
- 搜索定位应具备渐进式放大效果。
- 最终应定位到具体节点并保持高亮。

#### 实际行为
- 搜索定位渲染逻辑与之前版本存在差异。
- 无法持续放大后精确定位到目标节点。

#### 影响面
- Staging、本地开发均可复现。
- 影响功能域：F06 GIS / Topology。

#### 初步定位
- `omcmb/webcode/src/components/GISMap/index.tsx`：搜索结果触发定位逻辑。
- `omcmb/webcode/src/components/GISMap/useOLMap.ts`：`flyTo` / `highlightAndSpiderfyIfNeeded` 动画链路。
- `omcmb/webcode/src/components/GISMap/constants.ts`：渐进式 zoom 配置。

#### 严重等级
P1（核心搜索定位体验退化）

#### 关联
- 问题来源：测试问题汇总20260601 文档第 156 条。
- 建议分支：`fix/gis-search-progressive-locate`。

### 当前评估结论
确认存在，属于搜索定位动画链路退化问题，应单独作为 Issue 处理。

### 代码证据（现状）
- 搜索定位实际走 `highlightAndSpiderfyIfNeeded -> flyTo(progressive)`，但 progressive 目前是单段动画：
  - `omcmb/webcode/src/components/GISMap/index.tsx`
  - `omcmb/webcode/src/components/GISMap/useOLMap.ts`
- `PROGRESSIVE_ZOOM_STEPS` 常量存在但未形成完整多段链式体验：
  - `omcmb/webcode/src/components/GISMap/constants.ts`
- 参考旧版行为：
  - `omcmb/original-omc/OMCWebServer/src/main/webapp/js/ol/ol-topo-helper.js`

### 根因
- 搜索定位动画逻辑已退化为少档/单段实现。
- 页面与 hook 间存在能力暴露但未统一使用的路径（行为不一致）。

### 修改方案（按模块）
1. `constants.ts` 补齐渐进式 zoom 配置。
2. `useOLMap.ts`：
   - 搜索定位恢复多段渐进动画（保留 smooth 作为降级模式）。
   - 命中 cluster 时自动展开并高亮目标点。
3. `GISMap/index.tsx` 收敛调用路径，避免同功能多入口造成行为差异。
4. 增加搜索定位回归测试（渐进放大、cluster 命中）。

### 验收标准
- 搜索定位具备多段渐进放大体验。
- 目标点在聚合中时，可自动展开并稳定高亮。
- 定位结束后最终节点可见且高亮明确。

### 风险与回滚
- 风险：动画链路变复杂，可能引入时序抖动。
- 回滚：保留 smooth 单段动画开关作为应急兜底。

### 工作量预估
- 开发 1.0~1.5 人日
- 交互回归验证 0.5 人日

---

## Issue E（P0, Bug）
### 标题
拖动地图时 `/devices/geo` 请求堆积、不取消，弱网或大数据集下表现为「卡顿+长时 pending」

### 当前评估结论
确认存在，影响所有进入 GIS 地图页的用户。Network 面板可观察到拖动地图后多个 `/devices/geo?bounds=...` 请求并列 pending，老请求不会被取消。本质是请求生命周期没有和视口生命周期对齐。

### 复现步骤
1. 进入 GIS 地图页，等待首次加载完成。
2. 在地图上连续拖动若干次（每次拖动间隔 >300ms）。
3. 打开 DevTools → Network → XHR，过滤 `geo`。
4. 观察：拖动停下后多条 `/devices/geo` 仍为 pending，且按拖动次数累积。

### 代码证据（现状）
- React Query queryFn 未接收/转发 AbortSignal：
  - `omcmb/frontend-core/src/hooks/api/useTopology.ts` (`useMapDevicesGeo / useMapStats / useMapAggregation`)
- axios 请求未携带 signal，无法被取消：
  - `omcmb/frontend-core/src/services/api/topologyApi.ts` (`getDevicesGeo / getMapStats / getAggregation`)
- `getLoadStrategy(zoom)` 在 zoom≥15 时 `pageSize=10000`，单请求耗时被放大：
  - `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`
- 视口防抖硬编码 300ms，散落多处，难以统一调优：
  - `omcmb/webcode/src/pages/topology/GISMapView/index.tsx`
  - `omcmb/webcode/src/components/GISMap/useOLMap.ts` 内 `bindMapEvents` 100ms

### 根因
- 旧请求未取消：queryFn 没接 `signal`，axios 没 `{ signal }`，React Query 即使 queryKey 变化也无法取消底层网络。
- 旧请求未屏蔽 UI：缺少 `placeholderData: keepPreviousData`，拖动期间画面会闪空，加重「卡顿」感。
- pageSize 过大：zoom≥15 直接 10000，叠加后端无优化时单请求耗时 1~3s+。
- 多处防抖时序硬编码，无单一入口调节。

### 修改方案（按分层）
1. `topologyApi.ts`：`getDevicesGeo / getAggregation / getMapStats` 新增可选第二参 `signal?: AbortSignal`，传给 axios `{ signal }`（axios v1 原生支持）。
2. `useTopology.ts`：三个 map hooks 改成 `queryFn: ({ signal }) => topologyApi.xxx(params, signal)`；同时加 `placeholderData: keepPreviousData`，避免拖动期间画面闪空。
3. `GISMapView/index.tsx`：
   - 视口防抖时间统一改为读取 `MAP_CONFIG.viewportDebounce`，删除硬编码 300。
   - `getLoadStrategy` 的 pageSize 上限收到 2000（与 `VIEWPORT_CULLING.enableThreshold` 对齐），中段 zoom 降到 1000，避免极端值放大后端耗时。
4. （可选 P1）`useOLMap.ts` 的 bindMapEvents 100ms 是否合并入页面层 300ms，留作 Issue C 一并评估。

### 验收标准
- 连续拖动地图 5 次后，Network 中 `/devices/geo` pending 请求数 ≤ 1（前一发被自动 abort）。
- 拖动过程中画面不出现「空白闪烁」，旧设备 marker 平滑过渡到新数据。
- zoom=15+ 时单次请求 pageSize ≤ 2000，响应时间显著低于先前 10000 的水平。
- typecheck + lint 通过。

### 风险与回滚
- 风险：AbortController 改造涉及 frontend-core 公共 API。已采用「可选参数」形式，旧调用方零影响。
- 回滚：每个 hook 移除 `placeholderData` + signal 透传即可，原行为可恢复。

### 工作量预估
- 开发 0.3~0.5 人日
- 联调验证 0.3 人日

### 关联
- 问题来源：测试拖动期间截图（Network 面板大量 pending `/devices/geo`）。
- Issue 链接：https://github.com/569423176-sketch/goomc/issues/508
- 建议分支：`fix/gis-geo-request-cancel-and-initial-fit`（与 Issue F 合并）。

---

## Issue F（P0, Bug）
### 标题
首屏地图视口落在 `tiles-metadata.center`，但实际设备分布不在该区域时用户看不到任何节点

### 当前评估结论
确认存在。当离线瓦片 metadata 的 `center` 与本地/测试库的实际设备坐标不在同一区域时（例如 metadata 是赞比亚、设备落在中国移动域），用户首屏看到地图但完全没有设备点；只有手动拖图到设备分布区域才能看到，体验偏差严重。

### 复现步骤
1. 让 `tiles-metadata.json` 中心点配置在 A 区域（例如赞比亚 [28, -15]）。
2. seed 数据中所有设备坐标落在 B 区域（例如东亚）。
3. 进入 GIS 地图页。
4. 观察首屏：地图渲染正常但没有任何设备 marker。

### 代码证据（现状）
- 首屏中心点 4 层策略，metadata 优先级最高：
  - `omcmb/webcode/src/pages/topology/GISMapView/index.tsx` (`initialCenter / initialZoom` useMemo)
- 计算设备分布兜底的工具已有：
  - `omcmb/webcode/src/utils/mapValidation.ts` (`calculateCenterFromDevices`)
- OL Map 仅在初始化时消费一次 center/zoom，后续不响应 props 变化：
  - `omcmb/webcode/src/components/GISMap/useOLMap.ts` (`isMapInitializedRef` guard)

### 根因
- 4 层策略的优先级 1 是「无脑使用 metadata.center」，没有验证 metadata.bounds 是否覆盖实际设备分布。
- 优先级 2「按设备分布计算 center」存在鸡生蛋问题：首发请求未必带 bounds（mapViewport=null），即使返回也是优先级 1 已经落地之后。
- OL Map 视图初始化一次性消费 center/zoom，后续即使算出更合适的中心也不会自动跟进。

### 修改方案（按分层）
不破坏 Issue A 已确认的 4 层策略优先级，新增一次性「fit-to-devices」兜底机制：

1. `GISMapView/index.tsx`：新增 `useEffect`，依赖 `mapDevices`：
   - 用 `hasAutoFittedRef` 守卫只触发一次。
   - 通过 `mapRef.current.getViewport()` 判断当前可视范围内是否包含任意一个设备。
   - 若包含 → 标记完成，什么也不做。
   - 若一个都没有 → 调用 `calculateCenterFromDevices(mapDevices)`，再 `mapRef.current.flyTo(centerLng, centerLat, zoom, { progressive: false })`。
2. 不修改 4 层策略本身，不改 `useOLMap` 初始化逻辑，零侵入。
3. 后续若 metadata 与设备一致，effect 走 `hasAnyInView=true` 分支自然 noop。

### 验收标准
- metadata.center 与设备分布同区域：首屏行为完全不变（兜底分支 noop）。
- metadata.center 与设备分布不同区域：首屏出现地图后短时间内自动 flyTo 到设备区域，可见到设备点。
- 用户手动拖动后，effect 不再触发（仅一次性）。

### 风险与回滚
- 风险：兜底 flyTo 的初始动画可能让用户感觉「地图自己动了一下」。已选用 `progressive: false`（平滑单段），动画时长 < 600ms。
- 回滚：删除 effect 即可恢复原行为。

### 工作量预估
- 开发 0.2 人日
- 联调验证 0.2 人日

### 关联
- 问题来源：测试本地 docker 环境进入 GIS 页面截图（地图渲染正常但无设备 marker）。
- Issue 链接：https://github.com/569423176-sketch/goomc/issues/509
- 关联 Issue A：本兜底机制是 Issue A 4 层策略的补充，不冲突。
- 建议分支：`fix/gis-geo-request-cancel-and-initial-fit`（与 Issue E 合并）。

---

## 4. 合并说明

- 153-156 四条作为本轮主问题，按截图原文保留。
- 之前文档中"聚合分层 + 搜索定位"曾合并描述，这次按截图拆分为两个独立 Issue，便于分别评估和提单。
- 之前文档中的"无离线地图场景中心点策略统一"不删除，改为补充关注项保留，避免丢失既有上下文。
- 本轮分析和实施建议主要聚焦 Issue B-D，Issue A 保持现有结论不再展开。
- Issue E（请求堆积）+ Issue F（首屏中心 race）在 Issue B 修复期间通过测试复现新增，因影响所有用户已提升至 P0，建议同 PR 处理。

---

## 6. 实施顺序建议（按收益/风险）

1. Issue E + F（P0）：先解决日常拖图必现的请求堆积 + 首屏看不到设备问题（一个 PR）。
2. Issue B（P1）：解决设备状态统计异常问题（已在 PR #495 处理）。
3. Issue C（P1）：解决节点分层渲染不合理问题。
4. Issue D（P1）：解决搜索定位渐进放大回归问题。
5. Issue A：作为已确认项保留，不纳入本轮修改顺序。

---

## 7. 分支与 PR 切片建议

- PR-A（Issue A）
  - 建议分支：`fix/gis-tiles-metadata-fallback`
  - 涉及：`omcmb`
- PR-B（Issue B）
  - 建议分支：`fix/gis-status-count-align`（已在 PR #495）
  - 涉及：`omcgo + omcmb`（跨栈）
- PR-C（Issue C）
  - 建议分支：`fix/gis-cluster-layering-align`
  - 涉及：`omcmb`
- PR-D（Issue D）
  - 建议分支：`fix/gis-search-progressive-locate`
  - 涉及：`omcmb`
- PR-EF（Issue E + Issue F，合并 PR）
  - 建议分支：`fix/gis-geo-request-cancel-and-initial-fit`
  - 涉及：`omcmb`（frontend-core + webcode）

每个 PR 独立验收，避免大改动互相干扰。

---

## 8. 提交前检查清单

- [ ] 每个 Issue 都能映射到明确 PR 范围
- [ ] 每个 PR 都有回归用例或手工验证步骤
- [ ] GIS 页面关键行为（统计、首屏、搜索、缩放）有对照截图/录屏
- [ ] 变更遵循现有分层，不在页面层引入领域规则

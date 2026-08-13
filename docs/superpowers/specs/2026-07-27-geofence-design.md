# OMC 电子围栏最终设计

> 状态：代码闭环已实现，等待真实设备验收
> 需求 Issue：`#269`（105820 基于皮基站地理位置的接入管控）
> 业务事实来源：`docs/archive/legacy-handoff/Back-end/老OMC系统-基站电子围栏与位置检测去激活功能说明.md`

## 1. 目标与边界

在现有 OMC 内实现老系统的电子围栏配置、设备绑定、位置判定和设备处置闭环：

```text
Map 配置围栏
  → 发布并启用
  → 绑定设备
  → 设备位置上报
  → 连续采样判定
  → 记录位置安全状态
  → 按策略通知/去激活
  → 安全返回后人工或自动恢复
```

V1 复用现有 PostgreSQL、EventBus、异步任务、TR-069 和 OpenLayers，不引入 PostGIS、
新微服务或第二套地图。`enforce` 代码链按 Issue #304 修订为产品能力驱动的小区去激活状态机，
生产启用仍以真实设备完成 Admin/AdminRF、RF、IPSec 写入/回读、只读 OpState 终态及安全恢复
验收为门禁；任务入队、SPV 完成或仅 RF/IPSec 回读均不能单独视为控制成功。

不在当前闭环内：

- 首次接入资格判断；
- 自动按地图位置绑定设备；
- 历史轨迹平台；
- 物理删除历史围栏和判定记录；
- 没有正式协议的安全网关占位调用。

## 2. 稳定业务决策

1. 围栏归属一个运营商，跨运营商不可见、不可绑定。
2. 支持多边形允许区域和单设备基准点半径两种规则。
3. 同一设备每种规则类型最多一个活动绑定；多边形围栏不做并集或交集聚合，手工加入另一
   多边形围栏必须以可审计的原子改绑替换原活动绑定。
4. 设备离开围栏不解除绑定。
5. GPS 无效、过期或异常时结果为 `unknown`，不能当作安全或越界。
6. 默认越界策略为 `notify_only`，默认恢复为人工确认。
7. 只有位置策略此前成功关闭的设备才可能自动恢复。
8. 围栏名称允许修改但 `fence_id` 不变；修改名称不创建几何版本、不改变绑定、不触发控制。禁用或归档按已确认策略对原围栏内设备去激活，但不隐式激活设备。
9. 围栏版本不可变；编辑创建草稿，发布切换当前版本。
10. 系统总开关和运营商开关均默认关闭；用户明确确认开启后进入 `enforce`，关闭回到
    `off`。`observe` 保留为联调/诊断模式，可在 GIS 高级设置中按运营商选择，但不是
    老业务“启用”开关的默认语义。
11. PostgreSQL 是权威状态；Redis 只用于队列、锁和短期防重。
12. 判定、设备处置和通知解耦，任何一个失败不能篡改其他事实。
13. 页面保存成功后必须回查服务端，不以本地开关或 toast 作为最终结果。
14. 批量绑定逐设备持久化结果，一项失败不回滚已成功项。
15. `archive` 代替物理删除并保留版本、绑定和判定审计。
16. 围栏写操作复用现有 `audit_logs`；以单条业务语义记录替代同请求的通用 HTTP
    记录，预览请求不写合规审计。
17. #105820 的多边形围栏采用单一有效归属：同一设备同一时刻只能有一个 active 的
    `polygon_allow_zone` 绑定；手工加入另一围栏是“改绑”，不是叠加第二个有效围栏。

## 3. 系统结构

```text
TR-069 位置同步
  → device_location_observations + event_outbox（同事务）
  → Outbox Relay
  → device.location.observed
  → Geofence Coordinator
  → evaluator
  → evaluation + binding state + effective state + geofence events（同事务）
  → 告警
  → 围栏控制动作（SPV）
  → 关联回读（GPV）
  → verified / partial_failed / failed
```

管理请求遵循现有分层：

```text
Gin handler → geofence service → Squirrel/pgx repository → PostgreSQL
React page → frontend-core hook → frontend-core API → /api/v1
```

### 3.1 位置与判定

- `device_location_observations` 保存每台设备最新位置及递增版本；
- 经纬度使用 WGS84，GeoJSON 顺序始终为 `[longitude, latitude]`；
- evaluator 是无副作用纯逻辑，处理多边形、基准半径、边界距离、连续采样和回区；
- Coordinator 按设备串行锁定状态，将不可变 evaluation 和状态边沿事件原子落库；
- 判定唯一身份为 `binding_id + geofence_version_id + observation_version`。

设备有效状态：

```text
unmanaged | unknown | inside | outside
```

对 #105820 多边形围栏，设备只读取唯一 active 的 `polygon_allow_zone` 绑定；没有绑定为
`unmanaged`，无法完成判定为 `unknown`。`baseline_radius` 属于独立的位置检测规则，是否
参与设备处置必须由其自身策略决定，不能与多边形围栏暗中做“任一 outside 即去激活”的聚合。

### 3.2 可靠事件

`event_outbox` 使用至少一次发布：

- 写业务事实和 Outbox 在同一 PostgreSQL 事务；
- `dedupe_key` 唯一；
- Relay 使用带过期时间的 claim；
- 消费者通过业务唯一键保持幂等；
- 发布失败重试，超过上限进入 dead，不回滚已提交业务事实。

### 3.3 批量绑定

手工绑定接受 UUID 和 SN，最多 1000 个原始输入：

1. 服务端解析、去重并应用设备组权限；
2. 预览返回 `eligible/move/skipped` 和稳定原因码；`move` 明确给出原围栏和目标围栏；
3. 用户携带预览指纹和原因确认；
4. 复用 `async_jobs` 创建 `geofence_manual_bind`；
5. `geofence_batch_items` 保存每个输入结果；`move` 还保存用户预览确认时的原绑定 ID，
   防止排队期间归属变化后静默移动未经确认的新绑定；
6. worker 每项独立事务重新校验围栏、版本、运营商和当前归属；
7. `move` 在同一事务中把原 active 绑定标记为 removed（原因 `reassigned`），再创建目标绑定；
8. 成功绑定后用原始最新位置事件触发一次幂等判定；
9. 前端轮询作业总览并展示逐项结果。

不存在或无权访问的 SN 使用相同结果，避免枚举设备。显式无权访问 UUID 整批拒绝。

## 4. 数据模型

| 表 | 职责 |
|---|---|
| `devices.location_source_mode` | 设备位置权威来源：`tr069` 或 `external` |
| `geofence_carrier_settings` | 运营商模式和默认基准半径 |
| `geofence_definitions` | 围栏身份、运营商、规则类型和生命周期 |
| `geofence_versions` | 不可变几何与策略版本 |
| `device_geofence_bindings` | 设备和规则的绑定关系 |
| `device_geofence_states` | 单绑定连续采样状态 |
| `device_geofence_effective_states` | 单设备聚合位置安全状态 |
| `geofence_evaluations` | 每次不可变判定证据 |
| `geofence_batch_items` | 批量绑定逐项结果 |
| `event_outbox` | 可靠业务事件 |

当前不创建控制动作、步骤、网关请求或设备任务扩展表。取得正式控制契约后，再根据真实
回执和幂等边界补充数据模型。

## 5. 运行模式

系统模式和运营商模式共同决定有效模式，取两者中更保守的一个：

| 系统模式 | 运营商模式 | 有效模式 |
|---|---|---|
| off | 任意 | off |
| observe | off | off |
| observe | observe | observe |
| enforce | observe | observe |
| enforce | enforce | enforce |

当前版本接受 `off/observe/enforce`：

- `off`：继续保存位置事实，不执行围栏判定；
- `observe`：完成判定、历史和事件，不创建设备控制；
- `enforce`：代码链可配置；生产环境在控制、回读、告警和恢复的真实设备验收通过后启用。

更新系统与运营商模式必须在同一事务中完成。页面先预览影响，再确认保存并回查。系统
模式只能由超级管理员在“系统管理 → 系统配置 → 电子围栏”修改；GIS 地图只管理当前
运营商开关，非超管不能借运营商设置修改系统模式。

## 6. 围栏生命周期

```text
draft → enabled ↔ disabled → archived
```

- 创建时生成定义和第一个草稿版本；
- 已发布版本不可修改，编辑创建新草稿；
- 发布草稿后更新 `current_version_id`；
- 启用、禁用和归档使用服务端影响预览、指纹和操作原因；
- 归档只允许从 disabled 进入；
- 活动批量作业会阻止归档；
- 禁用和归档不直接产生恢复或控制动作，但会为受影响绑定设备写入独立生命周期重评估事件；
- 绑定可 `active → suspended → active`，或软移除为 `removed`。

## 7. API 契约

### 7.1 设置与地图

```text
GET  /api/v1/geofences/settings
POST /api/v1/geofences/settings/preview
PUT  /api/v1/geofences/settings
GET  /api/v1/geofences/map
```

地图查询支持：

```text
bounds=minLng,maxLng,minLat,maxLat
carrier=cmcc|ctcc|cucc
status=draft|enabled|disabled
name=<keyword>
```

只返回与视口相交的当前发布版本；归档默认不返回。

### 7.2 定义、版本与生命周期

```text
GET  /api/v1/geofences
POST /api/v1/geofences
GET  /api/v1/geofences/:id
GET  /api/v1/geofences/:id/versions
POST /api/v1/geofences/:id/versions
POST /api/v1/geofences/:id/publish
POST /api/v1/geofences/:id/{enable|disable|archive}-preview
POST /api/v1/geofences/:id/{enable|disable|archive}
```

### 7.3 绑定与作业

```text
GET    /api/v1/geofences/:id/bindings
GET    /api/v1/geofences/:id/bindings/export
POST   /api/v1/geofences/:id/binding-preview
POST   /api/v1/geofences/:id/candidate-preview
POST   /api/v1/geofences/:id/bindings
GET    /api/v1/geofence-jobs/:id
GET    /api/v1/geofence-jobs/:id/items
POST   /api/v1/geofence-bindings/:id/suspend
POST   /api/v1/geofence-bindings/:id/resume
DELETE /api/v1/geofence-bindings/:id
```

### 7.4 第三方位置兼容接口

为兼容老需求单附件，当前项目提供受 JWT/API Key 保护的精确路径：

```text
POST /fence/batchUpdateDeviceLocation
Content-Type: application/json
```

请求体沿用附件的 `devices`、`vesselName`、`serialNumber`、字符串经纬度和
`yyyy-MM-dd HH:mm:ss` 的 `updateTime`。单次最多 1000 条，服务端按最多 100 条的并发批次
处理；每条成功写入 `device_location_observations` 并通过 Outbox 发布统一位置事件。响应保持
老接口的 `{success,message,data}` 结构，`data` 返回总数、成功数、失败数和逐设备失败原因；同一设备相同 `updateTime` 的重试按幂等成功处理，不重复递增位置版本或发布位置事件，早于已存观测时间的记录按逐设备失败返回。

第三方位置可写入 `location_source_mode=tr069` 和 `external` 两种模式，不要求调用方预先把设备
切换为 `external`，一次接口调用也不会隐式修改设备模式。默认 `tr069` 模式同时接受 TR069 GPS
和第三方位置，两者按观测时间防乱序；`external` 是管理员为不能自主上报 GPS 的船载设备明确
启用的防覆盖模式，只接受第三方位置。`external` 设备仍可上报普通 TR069 参数，但其 TR069 GPS
坐标不会写入位置事实或初始化 OMC 接受坐标。

`updateTime` 本身不携带时区，按 OMC 系统配置 `basic.timezoneCode` 解释后保存为绝对时间，不能
固定按 UTC 或容器本地时区解析。规范字段 `vesselName` 继续必填；历史别名只用于兼容，不等于
放宽字段完整性要求。

当前接口同步返回的顶层 `success=true` 只表示批次请求已完成处理，调用方必须检查
`successCount/failCount/failedDevices`；只有逐项成功的位置才已持久化。围栏判定、告警和设备控制
异步执行。
接口支持 `Idempotency-Key`：服务端持久化请求指纹和逐设备结果，相同 key 且请求相同会重放原结果，
相同 key 但请求不同返回冲突，处理中重复请求返回冲突。没有提供 key 的旧调用保持兼容，但不提供
批次重放保证。当前仍未建设断点可恢复的持久化批次作业，HTTP 响应也不能写成控制执行完成凭证。

历史调用方使用的 `serialName/updatetime` 只在该 HTTP Adapter 归一化为
`vesselName/updateTime`；规范字段优先，领域服务和位置事实模型只接收规范字段。

所有接口使用稳定错误码和标准响应 envelope；后端执行 RBAC、设备组可见性和运营商
隔离，前端按钮隐藏不能代替服务端权限。

非超管的围栏列表、地图和按 ID 读写统一从服务端可见设备组推导可管理运营商范围；
空范围 fail-closed，基准圆的 `owner_device_id` 还必须是调用者真实可见的设备。不得
接受前端传入的设备组范围，也不得仅在绑定列表上过滤。

## 8. GIS Map 交互

继续扩展现有 `GISMapView` 和 OpenLayers：

- 顶部 `Geo-fence` 工具进入围栏模式；
- 按当前视口加载围栏；
- 独立 VectorLayer 渲染，不污染设备、扇区、测距图层；
- 单击围栏选中，双击或按钮定位；
- 面板提供设置、列表、搜索、新增、编辑、启停、归档、绑定设备和绑定清单 CSV 导出；
- 绘制结束只形成表单草稿，明确保存后才调用后端；
- 编辑已发布围栏时，名称、运营商和规则类型作为稳定身份只读；重绘和策略修改创建
  不可变新版本，发布成功后才切换当前版本；
- 退出工具或取消编辑时清理临时绘制，不改变服务端状态；
- 所有用户可见文字走 `frontend-core` 国际化。

第一阶段按老 OMC 高频顺序交付：

1. 系统/运营商设置；
2. 围栏列表、搜索和定位；
3. 新增、绘制、发布和启用；
4. 编辑、重绘、禁用和归档；
5. 批量绑定、绑定列表和逐项结果。

不在前端按设备坐标猜测绑定候选。服务端 `candidate-preview` 负责按当前发布几何、
设备运营商和设备组权限返回区域内、区域外和无位置设备；候选预览只展示结果，不自动
改变绑定。管理员仍可输入 SN 做精确预览并创建绑定作业。

## 9. Observe 与正式控制边界

当前实现只把设备聚合 `outside` 边沿投影为一条
`GEOFENCE_LOCATION_OUTSIDE` 活动告警，并在聚合 `inside` 边沿通过现有告警引擎清除。
逐围栏边沿仅作为判定事实，不重复产生设备告警。

`observe` 只记录判定和告警，不创建设备控制。`enforce` 已接入现有 CWMP 任务链：先持久化
控制动作和设备参数快照，再提交 SPV；SPV 终态后提交关联 GPV。Issue #304 起，`verified`
必须同时满足 RF、IPSec、产品可写小区管理控制回读一致，并且所有目标小区只读 OpState 已进入
预期终态。缺值、值不一致、OpState 未收敛、设备失败或超时均不得生成影子成功回执。

仓库没有证据表明还存在独立安全网关控制接口，因此不增加 SecGW 占位适配器；如后续取得
正式协议，再作为独立步骤接入，不能从 `SecGWServer1/2/3` 设备参数反推接口。

### 9.1 产品能力驱动的小区去激活

设备详情 Quick Settings、参数模型和产品 XML 已确认以下控制路径：

完整矩阵、现场证据和启用门禁见
[2026-08-12-geofence-cell-deactivation-capabilities.md](2026-08-12-geofence-cell-deactivation-capabilities.md)。
产品差异由 `internal/core/carrier/` 根据 ParamModel 的 Access、标准/私有映射和设备实际快照解析，
geofence 状态机不得散落 productClass 分支。

IPSec 参数在标准模型中为 `READ_WRITE/BOOLEAN`，IPSec、RF 和已确认管理控件的开关值使用
`1=开启、0=关闭`。`{i}` 是设备实例模型中的动态索引，围栏控制在 Worker 中读取设备当前
`device_parameters` 快照，只选择实际存在的参数；Access 以匹配的 ParamModel/XML 为权威，
仅在模型缺失时回退到快照的 Writable 标记。目标小区优先采用明确的 `InUse=true` 集合，其次采用
合法 `NumOfCells` 的 `1..N`；两者缺失、非法或与当前快照矛盾时，为保证业务连续性，回退到
产品装配 `radioModes` 与 ParamModel `NumOfCells.enumValues` 共同声明的设备最大能力。例如 MLN
最大能力为三小区，即使当前产品类为 SC，无法确认有效小区时仍以 `1..3` 为目标；共享 BLQ 模型的
BAIBLQ/SC 与 QRTB（436Q，例如 mBS31001）则分别按产品装配收敛为最大 1 和 2。249 属于 MLQ，
不得与这里的 BAIBLQ 混称。目标集中每个实例仍必须在当前快照具备完整
RF/OpState 证据；缺项时动作失败，不能静默缩小集合或按未建模路径猜测下发。
控制按实例编号排序后为每个目标小区生成参数；不存在可靠 RF、Admin/AdminRF、IPSec 和只读
OpState 时拒绝创建控制 task，不回退到固定 `.1`。参数路径
确认不等于设备执行成功，仍必须以 TR-069 task 终态和设备回执为准。

控制动作以稳定 `action_key/command_key` 防重；动作记录保存契约版本、控制前值、请求值、
控制回读、OpState 终态、轮询进度、截止时间、失败原因和父动作。回区只恢复本系统按新契约
`verified` 的去激活动作，并只恢复本次实际改动且已 GPV 验证的参数；恢复小区集合还要与当前
`InUse/NumOfCells` 解析结果求交集，无法确认时只回退到原动作已拥有的最大实例集合，不能扩展到
新出现的实例。不存在已验证所有权时不自动激活。旧 RF/IPSec-only `verified` 记录保留历史事实，
但不升级为真实小区去激活证据。

仓库核查确认：`SecGWServer1/2/3` 是设备南向参数，不是网关控制协议；老 OMC 的
`FenceClient/FenceController` 已加密，189 页面也未提供控制请求和回执证据。不得基于
这些字段猜测接口。

## 10. 业务闭环追踪

| 老业务步骤 | 后端 | 前端 | 当前结论 |
|---|---|---|---|
| Map 入口、围栏显示和定位 | 已完成地图读接口 | 已完成只读图层 | 已闭环 |
| 系统/运营商开关 | 已完成设置与预览 | 已完成设置面板 | 已闭环 |
| 列表、搜索、创建、编辑 | 已完成定义/版本接口 | 已完成面板和绘制表单 | 已闭环 |
| 单围栏启停和删除 | 已完成预览与软归档 | 已完成预览确认 | 已闭环 |
| SN 批量绑定与结果 | 已完成异步作业 | 已完成预览、轮询和逐项结果 | 已闭环 |
| 导出围栏关联设备 | 复用权限过滤后的绑定查询输出 UTF-8 CSV | 绑定抽屉直接下载 | 已闭环 |
| 新增、修改、启停、归档、设置和绑定审计 | 已复用现有 `audit_logs` 记录对象、原因、结果和失败原因 | 复用现有审计查询页面 | 已闭环 |
| 位置上报到连续判定 | 已完成 | 绑定设备视图展示判定状态、候选次数、边界距离、位置版本、时间和错误码 | Observe 业务闭环 |
| 越界告警与通知 | 已投影到现有活动告警和历史归档 | 复用现有告警页面及通知规则 | Observe 告警闭环 |
| 产品能力驱动的小区去激活 | 解析 Admin/AdminRF、RF、IPSec、OpState，持久化轮询终态 | 分开展示 RF、激活状态、请求值、回读值及失败原因 | Issue #304 修订中，按产品证据灰度启用 |
| 安全返回与恢复 | 只恢复本系统实际改变且验证成功的参数，并恢复原 OpState 终态 | 展示恢复动作、控制回读和运行终态 | Issue #304 修订中，待真实设备验收 |

任何阶段不得把“后端接口存在”“任务已入队”或“页面提示成功”写成整条业务已经闭环。

## 11. 验收标准

### 11.1 当前管理与 Observe 阶段

- 189 已确认的入口、开关、列表、绘制、编辑、启停、归档和绑定流程可在新页面完成；
- 页面刷新后结果与数据库一致；
- 不同运营商和设备权限范围互不可见；
- GPS 异常不被误判为安全；
- 重复位置事件不产生重复 evaluation；
- `off/observe` 不创建设备控制任务；
- 禁用、解绑和归档不恢复设备；
- 批量绑定每个输入都有可查询终态。
- 导出清单只包含当前用户可见的绑定，不包含原始经纬度；
- 围栏实际写操作只产生一条可按 `geofence` 检索的业务审计，包含操作人、来源 IP、
  对象 ID、目标状态、原因、结果和失败原因；预览不写合规审计。

### 11.2 Enforce 阶段

- 设备 Admin/AdminRF、IPSec、RF、只读 OpState 和告警步骤都有独立证据；尚无安全网关接口时不伪造回执；
- 相同状态边沿只产生一个逻辑动作；
- 离线、失败、超时和部分成功可以重试或转人工；
- 只有 geofence 来源的成功关闭允许恢复；
- 恢复链任一前置失败都不会恢复 Admin/AdminRF；
- 真实测试站完成越界、去激活、回区和恢复灰度验证。

## 12. 防止过度开发

每个新增文件或接口必须能够映射到第 10 节某一业务步骤，并满足：

1. 优先完成一个可验收垂直切片，再建设下一切片；
2. 不为尚无正式协议的网关、通知渠道或导出格式创建占位实现；
3. 不重复已有 EventBus、异步作业、任务和地图基础设施；
4. 测试保护权限、状态转换、幂等和真实用户行为，不测试框架本身；
5. 检查点和验证结果统一写入本设计及单一验收记录，不再增加重复计划文档；
6. 发现无调用方且无后续需求映射的代码，先记录证据，再删除。

## 13. 需求单 #105820 严格验收矩阵

本节是新项目对老需求单的唯一验收映射。需求单自测写“通过”不代表新项目已经具备同等证据；
每项必须同时满足行为、证据和回归测试要求。

| 需求单标准 | 当前状态 | 缺口/必须补齐的证据 |
|---|---|---|
| 系统级开关默认关闭并可开启 | 已实现 | 基线默认 `off`；仅超管在系统配置页预览后开启；关闭时 GIS 入口不可见；仍需部署后的重启持久化验收 |
| 运营商级开关 | 已实现 | GIS 明确选择运营商后预览并切换；开启进入 `enforce`，关闭进入 `off`；服务端禁止非超管修改系统模式 |
| 围栏增删改查 | 已实现管理面 | 真实浏览器流程、名称长度边界、版本发布和软归档回归 |
| 批量导入绑定设备 | 核心链路已实现 | 已实现 `eligible/move/skipped` 预览、原绑定快照校验和事务内原子改绑；仍需真实 PostgreSQL、2G/4G/5G、权限、部分失败、作业重试和最终回查验收 |
| 第三方批量位置上报触发激活/去激活 | 代码链按 #304 修订 | `/fence/batchUpdateDeviceLocation` 已接入位置事实、Outbox、批次幂等、围栏事件、SPV/GPV 和 OpState 轮询；仍需 1000 台压测及逐产品真实设备验收 |
| 操作日志 | 已实现基础审计 | 严格断言每个业务写操作一条审计，包含 actor、IP、对象、原因、结果和失败原因 |
| web 进程记录第三方接口 | 已复用全局入口日志 | 已记录请求 ID、路径、响应码、耗时和客户端 IP，且不记录请求体；业务失败明细由接口结果和批次记录查询 |

### 13.1 当前仓库核查结论

- `device.location.observed` 的现有生产来源是设备位置同步事务；第三方位置规范 `POST /fence/batchUpdateDeviceLocation` 已实现为受认证保护的兼容入口，当前入口按 100 条并发批次处理并返回逐设备结果。
- 已知第三方请求字段为 `vesselName`、`serialNumber`、字符串格式的 `longitude`/`latitude` 和 `updateTime`；响应需要逐设备返回成功/失败统计及 `failedDevices`。附件建议单次最多 1000 个设备、每 10 分钟上报变化设备，并要求 OMC 按每批 100 个异步处理。
- 189 实测规范字段 `vesselName/updateTime` 与历史截图别名 `serialName/updatetime` 均可受理；新接口只把前者作为规范，兼容别名必须停留在第三方 Adapter。
- 189 的手工绑定是单一有效归属：把两台 `fence3` 设备加入临时围栏后，`fence3` 从 15 台降为 13 台；删除临时围栏不会自动恢复原归属。当前代码已由唯一索引、`move` 预览和 worker 原子改绑共同落实；排队期间原绑定发生变化时跳过，避免移动未经用户确认的新归属。
- 189 使用未绑定离线站 `1202000240194DP0015` 再次完成创建、Batch Input、启用、停用和删除并恢复现场；它证明离线设备也可进入配置流程，但不构成 RF/IPSec 指令送达或执行成功证据。
- `geofence.device.exited/entered` 已由围栏控制监视器消费；产品适配器按 ParamModel 解析 Admin/AdminRF、RF、IPSec 和只读 OpState，创建专用 `TaskSourceGeofence` 任务，并在 `command_key` 中按 `geofence:<device>:<effective_state_version>:deactivate|activate` 记录动作。
- 控制动作在任务前落库；SPV 成功后用同一动作 ID 创建 GPV，精确核对每个控制路径和值，再持久化轮询 OpState。任务入队、SPV 完成、RF/IPSec 回读或页面 toast 都不会被写成真实去激活成功。
- 回围事件只从最近一条 `contract_version>=2`、终态 `verified` 且尚未恢复的本系统去激活动作生成 `:activate`；相同动作键防重，没有新契约已验证控制所有权的设备不会因普通回围事件被恢复。
- `enforce` 状态机、动作幂等、失败关闭和恢复所有权已经实现；缺任一产品能力时 fail closed，正式启用仍须通过授权设备的 SPV、GPV、OpState、离线、超时和重试验收。
- 关闭系统/运营商开关、禁用围栏、解绑或归档时，既有位置安全告警是否自动清除必须明确产品决策；默认不自动恢复设备，也不能静默清除控制失败。
- 禁用/归档时，事务内筛选活动绑定、最后确认 `inside` 且当前版本策略明确为 `deactivate`
  的设备，写入
  `geofence.lifecycle.deactivation_required` Outbox；控制监视器使用 Outbox 事件 ID 生成稳定
  `command_key` 幂等提交完整小区去激活动作。Observe 围栏不会创建设备任务；当前版本又禁止
  发布 `deactivate` 策略，因此该链路不会绕过 enforce 门禁。随后仍通过
  `geofence.lifecycle.reevaluate` 把规则从有效状态计算中移除；该流程不产生自动激活。
- 设备位置接入模式已收敛为设备级 `tr069/external`；第三方位置在两种模式下都可受理，默认
  `tr069` 还接受设备自身 GPS。船载且不能自主上报 GPS 的设备由设备详情明确切换为 `external`，
  以阻止后续 TR069 GPS 覆盖。位置观测仓储统一执行时间乱序保护；外部模式下忽略 TR069 GPS，
  但不阻断同一批次的其他设备参数同步。

### 13.2 老详细设计已确认的实现事实

用户补充的《基于皮基站地理位置的接入管控详细设计文档》进一步确认，老方案不是只有
业务口号，而是明确要求以下控制链：

```text
TR-069 Value Change 上报经纬度
  → 判断设备是否在启用围栏内
  → 对比历史状态
  → 状态变化时下发激活/去激活 TR-069 指令
  → 设备执行并返回结果
  → 生成告警和操作日志
```

同时确认了以下老设计约束：

- 系统级开关默认关闭，运营商级开关在地图页面开启；
- 围栏为不规则多边形，老方案创建后遍历 `small_cell_infos`，用射线法自动建立设备关联；
- 创建围栏时会扫描位置候选，但设备的当前生效多边形围栏只有一个；手工批量加入另一围栏会转移归属；
- 因此位置更新按设备当前 `fence_id` 对应的启用围栏判定，不采用多个多边形围栏的并集或交集；
- 越界由历史“在围栏内”到当前“不在围栏内”触发去激活，回区反向触发激活；
- 老设计把精度配置写成 `fence.judge.precision=10` 米，并提出边界缓冲和指令失败重试 3 次；
- 老方案使用 MySQL、`small_cell_infos.fence_id` 和 RabbitMQ，这些是老项目实现选择，不应原样搬到当前 PostgreSQL、EventBus/outbox 和独立绑定状态模型。

这说明当前项目具备控制执行基础设施，但老资料不能证明具体产品的小区管理参数和 OpState 终态；
剩余门禁是逐产品能力证明、部署版本一致性和真实设备验收证据。

## 14. 后续开发门禁与垂直切片

后续开发遵循“接口是测试面、先一条垂直切片再扩展”的 TDD 约束，不先横向铺开所有表和接口。

### 14.1 P0：取得并固定真实控制契约

必须先从老后端、第三方平台和授权测试站取得脱敏材料：

1. 第三方位置 URL 和主体字段已确认；仍需产品确认生产鉴权、幂等键、时间乱序处理，以及是否长期兼容 `serialName/updatetime` 别名；
2. 各产品 Admin/AdminRF、RF、IPSec 和只读 OpState 仍需真实设备确认实例范围、值语义、下发顺序、终态延迟和失败语义；
3. 若激活/去激活前后还需要 SecGW IPSec 操作，再取得安全网关请求、响应、超时、错误码和幂等键；若老项目没有独立 SecGW 调用，应形成“无独立网关接口”的确认记录；
4. 越界冻结时间、回围恢复条件、人工接管和审计要求；
5. 能同时提供位置样本、请求 ID、网关回执、设备任务回执、告警和日志的测试站。

材料不齐时不得新增占位控制调用或影子成功状态。`enforce` 已复用项目现有 CWMP 参数任务
与回读链路实现，但基线仍默认 `off`；未完成 14.4 的真实设备验收前，正式环境不得开启。

### 14.2 P1：第三方位置上报垂直切片

先实现一条可验收路径：

```text
第三方位置上报
  → 鉴权与幂等
  → 位置事实 + Outbox
  → 围栏 Coordinator
  → evaluation / effective state
  → 可关联的控制输入事件
```

先写公共接口行为测试：重复批次不重复生成位置版本；非法坐标、过期时间和未知设备拒绝；
成功请求可按 request ID 查到位置事实和判定结果。该切片不调用真实控制，直到契约测试固定。

### 14.3 P2：越界控制与回执

以设备有效状态边沿为唯一动作输入，建立“已入队、执行中、成功、失败、超时、人工接管”
的可追踪状态。网关和设备控制各自有回执，不能用任务入队代表执行成功。所有消费者以
`device_id + effective_state_version + action_type` 幂等。

先实现越界去激活一条完整链，再实现回围恢复；每个切片都必须覆盖成功、重复、超时、部分
失败和重试。恢复前重新读取最新位置、围栏状态和控制所有权，前置步骤未确认成功不得开启 RF。

### 14.4 P3：开启 enforce 与真实验收

只有以下证据齐全才允许 `observe → enforce`：

- 第三方位置接口可重放且幂等；
- IPSec、RF、告警各有可关联回执；
- 越界、回围、重复、乱序、离线、超时和部分失败均有自动化测试；
- 真实测试站完成灰度，能从位置样本追到请求、回执、设备终态、告警和结构化日志；
- 失败恢复和人工接管流程有操作审计。

### 14.5 已定版与待评审差异

1. **关联方式已定版**：创建/编辑围栏后由服务端 `candidate-preview` 返回区域内、区域外和无位置设备，管理员选择并确认绑定；不按前端坐标猜测，也不直接恢复老表字段自动写入。
2. **多围栏规则已定版**：#105820 多边形围栏为单一有效归属，改绑替换原 active 绑定；`baseline_radius` 是独立规则类型，不与多边形暗中做并集聚合。
3. **删除/禁用语义已定版**：禁用或归档时，仅对活动绑定、最后确认在围栏内且策略明确为
   `deactivate` 的设备提交幂等去激活；Observe 围栏不创建设备任务，已在围栏外的设备不重复
   制造控制边沿，`unknown` 不自动猜测，流程绝不自动激活。
4. **位置来源已定版**：第三方平台位置在 `tr069/external` 两种模式下均可受理且不会隐式切换
   模式；TR069 GPS 仅在 `tr069` 模式受理。`external` 是管理员明确启用的设备 GPS 防覆盖模式，
   不是第三方接口的调用前置条件。受理后的观测按 `updateTime`/观测时间防乱序，相同时间幂等。

### 14.6 架构深化方向

按架构技能的“深模块”和“接口即测试面”原则，后续优先深化两个模块：

1. **位置接入模块**：对外只暴露“接受一批位置观测并返回逐项受理结果”，内部负责鉴权、坐标校验、时间解析、批量上限、来源/设备时钟排序、幂等、版本和 Outbox；TR-069 与第三方平台各自作为 Adapter，第三方 Adapter 兼容 `/fence/batchUpdateDeviceLocation`，避免把来源判断散落到 Coordinator。
2. **控制编排模块**：对外只暴露“提交某个有效状态版本的处置”，内部负责网关、设备任务、回执、重试、补偿和人工接管；告警继续订阅事实事件，不直接承担控制职责。

位置来源 Adapter 按上述兼容矩阵落地。控制模块继续复用现有 CWMP 任务链；只有确认存在独立
安全网关调用时才增加第二个控制 Adapter，在此之前不创建空实现。

## 15. 证据与验收记录规范

每个验收场景必须保留同一 `request_id` 或测试关联 ID 下的：输入位置、来源、观测版本、
规则版本、判定结果、控制请求、控制回执、告警变化、设备终态和结构化日志。截图只能作为
辅助证据，不能替代请求/回执和数据库事实。

关联问题单必须转成回归用例：

- #107427：名称长度上限和数据库错误不得泄漏到页面；
- #107428：绑定设备状态必须来自真实设备读模型，不能统一显示离线；
- #107434：所有用户可见提示走 i18n，新增和更新文案准确区分；
- #107503：回到围栏后必须验证恢复前置条件和最终激活结果。

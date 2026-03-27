# 0023 设备列表管理系统 — 差距分析与实现方���

> 基于《2/4/5G设备列表管理系统 后端开发设计文档》与当前代码库的对比分析。

---

## 1. 双表架构设计

### 1.1 总体思路

采用 **`devices` + `device_info` 双表架构**：

| 表 | 定位 | 写入来源 | 典型字段 |
|----|------|---------|---------|
| `devices`（已有） | 设备核心身份与连接信息 | ACS/Inform 自动写入 | serial_number, oui, status, ip_address, last_inform_at |
| `device_info`（新建） | 设备运维管理扩展信息 | 运维人员手动 + 参数自动同步 | device_name, eci, pci, freq_point, rf_status, address, remark |

两表通过 `device_id` 做 1:1 关联。列表查询时 `LEFT JOIN device_info` 获取完整视图。

### 1.2 现有 `devices` 表（保持不变）

```
migrations/000001_create_devices.up.sql，按 carrier 分区
```

| 字段 | 说明 | 来源 |
|------|------|------|
| id (UUID PK) | 主键 | 系统生成 |
| serial_number | 设备序列号 | TR069 Inform |
| oui, product_class, manufacturer, model_name | 设备身份 | TR069 Inform |
| carrier, technology | 运营商/制式 | Inform 自动识别 |
| status | 连接状态 (active/offline/maintenance/...) | ACS 心跳 + 手动 |
| firmware_version | 软件版本 | TR069 Inform |
| ip_address, connection_request_url | 网络连接 | TR069 Inform |
| nat_detected, udp_connection_request_address | NAT 穿越 | TR069 Inform |
| last_inform_at, last_inform_events, inform_interval | 心跳信息 | TR069 Inform |
| site_name, site_id, latitude, longitude | 站点信息 | 手动/导入 |
| extension_data (JSONB) | 灵活扩展 | 按需 |
| created_at, updated_at | 时间戳 | 自动 |

### 1.3 新建 `device_info` 表

存储设计文档中定义的、`devices` 表不包含的扩展字段。

```sql
CREATE TABLE device_info (
    device_id         UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,

    -- 运维标识
    device_name       VARCHAR(128),                -- 设备名称（用户自定义）
    address           VARCHAR(256),                -- 物理部署地址
    remark            TEXT,                         -- 备注
    project_status    VARCHAR(20),                  -- 工程状态：在建/已交付/运维中/停用
    height            DECIMAL(10,2),                -- 高度(米)

    -- 无线参数（从 TR069 参数同步）
    eci               VARCHAR(64),                  -- ECI 标识
    pci               VARCHAR(64),                  -- PCI 值
    cell_id           VARCHAR(64),                  -- 小区ID
    freq_point        VARCHAR(32),                  -- 频点(MHz)
    bandwidth         DECIMAL(8,2),                 -- 带宽(MHz)
    transmit_power    DECIMAL(8,2),                 -- 发射功率(dBm)
    plmn              VARCHAR(32),                  -- PLMN 编码

    -- 状态（从设备/参数同步）
    rf_status         VARCHAR(20),                  -- 射频状态：开启/关闭/异常
    cell_status       VARCHAR(20),                  -- 小区状态：正常/故障/未配置/退服
    mme_status        VARCHAR(20),                  -- MME 状态：正常/异常/未连接
    sync_status       VARCHAR(32),                  -- 同步状态：GPS同步/北斗同步/NTP同步/异常
    kpi_status        VARCHAR(20),                  -- KPI 状态：正常/异常/无数据

    -- 硬件信息
    mac               VARCHAR(64),                  -- MAC 地址
    hardware_version  VARCHAR(64),                  -- 硬件版本

    -- 时间记录
    first_online_time TIMESTAMPTZ,                  -- 首次上线时间
    last_offline_time TIMESTAMPTZ,                  -- 最后下线时间
    run_time          BIGINT DEFAULT 0,             -- 累计运行时长(秒)

    -- 审计
    creator           VARCHAR(64),                  -- 创建人
    updater           VARCHAR(64),                  -- 最近更新人

    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_device_info_rf_status    ON device_info (rf_status);
CREATE INDEX idx_device_info_cell_status  ON device_info (cell_status);
CREATE INDEX idx_device_info_project_status ON device_info (project_status);
CREATE INDEX idx_device_info_search ON device_info USING gin (
    (COALESCE(device_name,'') || ' ' || COALESCE(address,'')) gin_trgm_ops
);

-- updated_at 触发器
CREATE TRIGGER trigger_device_info_updated_at
    BEFORE UPDATE ON device_info
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 1.4 数据流向

```
                    ┌─────────────────────────────────────┐
                    │           device_parameters          │
                    │   (TR069 完整参数树, K-V 模型)       │
                    └──────────────┬──────────────────────┘
                                   │ 参数同步（定时/事件触发）
                                   ▼
┌──────────────┐   1:1   ┌──────────────────┐
│   devices    │ ◄─────► │   device_info    │
│  (核心身份)   │         │  (运维扩展信息)   │
│  ACS 自动写入 │         │  手动 + 自动同步  │
└──────────────┘         └──────────────────┘
        │                         │
        └────────┬────────────────┘
                 │ LEFT JOIN
                 ▼
        ┌─────────────────┐
        │   设备列表 API    │
        │  GET /api/v1/devices │
        └─────────────────┘
```

**同步机制**：
- **手动字段**（device_name, address, remark, project_status, height）：通过管理 API 手动填写
- **参数字段**（eci, pci, freq_point, bandwidth 等）：设备 Inform 后从 `device_parameters` 自动提取同步
- **状态字段**（rf_status, cell_status, kpi_status 等）：事件驱动更新（告警/参数变更事件触发）
- **时间字段**（first_online_time, last_offline_time, run_time）：心跳/状态变更时更新

---

## 2. 功能差距矩阵

### 2.1 已实现功能

| 设计文档功能 | 当前实现 | 对应代码 |
|-------------|---------|---------|
| 设备列表分页查询 | ✅ 已实现 | `device/handler.go` GET /api/v1/devices |
| 按 status/carrier/technology 精准过滤 | ✅ 已实现 | `device/pg_repository.go` List() |
| 按 SN/站点名模糊搜索 | ✅ 已实现 | DeviceFilter.Search |
| 设备详情查询 | ✅ 基础信息已实现 | GET /api/v1/devices/:id |
| 设备参数配置（读/写/同步/发现） | ✅ 完整实现 | `device/param_handler.go` 10 个端点 |
| 设备重启 | ✅ 已实现 | POST /api/v1/devices/:id/reboot |
| 设备分组管理 | ✅ 已实现（独立模块） | `topology/` 模块，层级树结构 |
| 排序（多字段） | ✅ 已实现 | ListRequest.SortBy/SortDir，9 个可排序列 |

### 2.2 未实现功能

| # | 功能 | 设计文档章节 | 优先级 | 复杂度 |
|---|------|------------|--------|--------|
| G01 | `device_info` 表与模块搭建 | 2.1.1 | **P0** | 中 |
| G02 | 设备列表 JOIN 查询 + 扩展过滤 | 3.2.1 | **P0** | 中 |
| G03 | 参数自动同步到 `device_info` | — | **P0** | 中 |
| G04 | 设备列表导出（CSV/Excel） | 3.2.6 | **P1** | 中 |
| G05 | 列自定义配置（用户级） | 3.2.2, 4.3 | **P1** | 低 |
| G06 | 枚举值查询接口 | 3.3 | **P1** | 低 |
| G07 | 多字段模糊搜索扩展 | 4.2.1 | **P1** | 低 |
| G08 | 设备操作：激活/去激活 | 3.2.4 | **P1** | 低 |
| G09 | 设备操作：射频开关 | 3.2.4 | **P1** | 中 |
| G10 | 设备操作：日志收集 | 3.2.4 | **P2** | 中 |
| G11 | 设备操作：报文收集 | 3.2.4 | **P2** | 中 |
| G12 | 设备详情聚合（告警/KPI/License 摘要） | 3.2.3 | **P2** | 中 |
| G13 | 逻辑删除 | 5.2 | **P2** | 中 |
| G14 | 敏感字段加密（MAC/经纬度） | 5.3 | **P3** | 中 |

---

## 3. 各功能实现方案

### G01: `device_info` 表与模块搭建

**需求**：新建 `device_info` 表，实现 CRUD，提供独立的管理 API。

**新增文件**：

```
internal/device/
    device_info_model.go          — DeviceInfo 结构体
    device_info_repository.go     — 接口定义
    device_info_pg_repository.go  — PostgreSQL 实现
    device_info_handler.go        — HTTP 处理
migrations/
    000XXX_create_device_info.up.sql
    000XXX_create_device_info.down.sql
```

**DeviceInfo 模型**：

```go
type DeviceInfo struct {
    DeviceID        uuid.UUID  `json:"device_id"`
    DeviceName      string     `json:"device_name"`
    Address         string     `json:"address"`
    Remark          string     `json:"remark"`
    ProjectStatus   string     `json:"project_status"`
    Height          *float64   `json:"height"`

    // 无线参数（自动同步）
    ECI             string     `json:"eci"`
    PCI             string     `json:"pci"`
    CellID          string     `json:"cell_id"`
    FreqPoint       string     `json:"freq_point"`
    Bandwidth       *float64   `json:"bandwidth"`
    TransmitPower   *float64   `json:"transmit_power"`
    PLMN            string     `json:"plmn"`

    // 状态
    RFStatus        string     `json:"rf_status"`
    CellStatus      string     `json:"cell_status"`
    MMEStatus       string     `json:"mme_status"`
    SyncStatus      string     `json:"sync_status"`
    KPIStatus       string     `json:"kpi_status"`

    // 硬件
    MAC             string     `json:"mac"`
    HardwareVersion string     `json:"hardware_version"`

    // 时间
    FirstOnlineTime *time.Time `json:"first_online_time"`
    LastOfflineTime *time.Time `json:"last_offline_time"`
    RunTime         int64      `json:"run_time"`

    // 审计
    Creator         string     `json:"creator"`
    Updater         string     `json:"updater"`

    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

**接口设计**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/:id/info` | GET | 获取设备扩展信息 |
| `/api/v1/devices/:id/info` | PUT | 更新手动填写字段（name/address/remark/project_status/height） |

**实现要点**：
- 设备通过 `RegisterFromInform` 注册时，自动创建空的 `device_info` 记录
- PUT 仅允许更新手动字段，无线参数/状态字段由同步机制自动填充
- `device_info` 不单独查，主要随 `devices` 一起 JOIN 查询

---

### G02: 设备列表 JOIN 查询 + 扩展过滤

**需求**：设备列表 API 返回 `devices` + `device_info` 合并数据，支持按 `device_info` 字段过滤/排序。

**改动文件**：`device/pg_repository.go`、`device/repository.go`、`device/handler.go`

**DeviceFilter 扩展**：

```go
type DeviceFilter struct {
    // --- 现有 (devices 表) ---
    Carrier      *model.CarrierCode
    Technology   *model.Technology
    Status       *model.DeviceStatus
    OUI          *string
    SN           *string
    Search       *string
    // --- 新增 (device_info 表) ---
    Manufacturer *string    // devices.manufacturer 精准
    ProductClass *string    // devices.product_class 精准
    RFStatus     *string    // device_info.rf_status 精准
    CellStatus   *string    // device_info.cell_status 精准
    ProjectStatus *string   // device_info.project_status 精准
    model.ListRequest
}
```

**List() 查询改造**：

```go
// 从单表查询改为 LEFT JOIN
builder := psql.Select(deviceWithInfoColumns()...).
    From("devices d").
    LeftJoin("device_info di ON di.device_id = d.id")

// 新增过滤条件
if filter.RFStatus != nil {
    builder = builder.Where(sq.Eq{"di.rf_status": *filter.RFStatus})
}
if filter.CellStatus != nil {
    builder = builder.Where(sq.Eq{"di.cell_status": *filter.CellStatus})
}
```

**返回模型**：API 响应中 `device_info` 字段平铺到设备对象中（前端无需感知双表）：

```json
{
  "id": "...",
  "serial_number": "BCI-SN-001",
  "status": "active",
  "manufacturer": "Baicells",
  "device_name": "朝阳区基站A",
  "eci": "460001234",
  "pci": "120",
  "rf_status": "开启",
  "cell_status": "正常",
  "address": "北京市朝阳区XX路XX号",
  ...
}
```

**排序扩展**：

```go
var allowedSortColumns = map[string]string{
    // 现有
    "created_at":     "d.created_at",
    "serial_number":  "d.serial_number",
    "status":         "d.status",
    "last_inform_at": "d.last_inform_at",
    // 新增
    "device_name":    "di.device_name",
    "rf_status":      "di.rf_status",
    "cell_status":    "di.cell_status",
    "bandwidth":      "di.bandwidth",
    "transmit_power": "di.transmit_power",
}
```

---

### G03: 参数自动同步到 `device_info`

**需求**：设备 Inform 后，自动将关键 TR069 参数提取到 `device_info` 表。

**新增文件**：`internal/device/info_sync.go`

**同步策略**：

| 触发时机 | 同步内容 |
|---------|---------|
| 设备注册（Bootstrap Inform） | 创建 `device_info`，记录 first_online_time，提取 MAC/硬件版本 |
| 周期性 Inform | 更新无线参数（eci/pci/freq_point/bandwidth 等） |
| 参数变更事件 (`device.inform.value_change`) | 更新变更的参数字段 |
| 状态变更（active→offline） | 记录 last_offline_time，累加 run_time |
| 告警事件 | 更新 rf_status/cell_status/kpi_status |

**参数路径映射**（Carrier 接口方法）：

```go
// 每个运营商适配器实现此方法，返回 TR069 参数路径 → device_info 字段的映射
type Carrier interface {
    // ...existing methods...
    GetInfoParamMapping(tech Technology) map[string]string
}

// CMCC LTE 示例
func (c *CMCCCarrier) GetInfoParamMapping(tech Technology) map[string]string {
    if tech == TechLTE {
        return map[string]string{
            "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity": "eci",
            "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID":        "pci",
            "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL":         "freq_point",
            "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth":      "bandwidth",
            "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower": "transmit_power",
            "Device.DeviceInfo.X_VENDOR_MACAddress":                                "mac",
            "Device.DeviceInfo.HardwareVersion":                                    "hardware_version",
        }
    }
    // NR mapping...
}
```

**实现要点**：
- 同步逻辑订阅 `device.inform.periodic` 和 `device.inform.value_change` 事件
- 批量同步：复用 `BatchInformProcessor`，在 flush 时顺带更新 `device_info`
- 仅在参数值变化时写入，避免无意义 UPDATE

---

### G04: 设备列表导出（CSV/Excel）

**需求**：支持全量导出和条件导出，格式支持 CSV 和 Excel。

**新增文件**：

```
internal/device/export_handler.go    — HTTP 处理
internal/device/export_service.go    — 导出逻辑（流式写入）
```

**接口设计**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/export` | POST | 条件导出，请求体含过滤条件 + format(csv/xlsx) |

**实现要点**：
- 流式分页读取（每次 500 条），避免内存溢出
- CSV：`encoding/csv` 直接写 `http.ResponseWriter`
- Excel：`github.com/xuri/excelize/v2` 流式写入（`StreamWriter`）
- 导出字段包含 `devices` + `device_info` 合并数据
- 导出列可基于用户列配置（G05），无配置则使用默认列
- V1 同步处理（设 5 分钟超时），V2 再引入异步任务

**默认导出列**：

```go
var defaultExportColumns = []ExportColumn{
    {Field: "serial_number", Header: "设备序列号"},
    {Field: "device_name", Header: "设备名称"},
    {Field: "status", Header: "连接状态"},
    {Field: "carrier", Header: "运营商"},
    {Field: "technology", Header: "网络制式"},
    {Field: "manufacturer", Header: "制造商"},
    {Field: "model_name", Header: "设备型号"},
    {Field: "ip_address", Header: "IP地址"},
    {Field: "eci", Header: "ECI"},
    {Field: "pci", Header: "PCI"},
    {Field: "rf_status", Header: "射频状态"},
    {Field: "cell_status", Header: "小区状态"},
    {Field: "firmware_version", Header: "软件版本"},
    {Field: "site_name", Header: "站点名称"},
    {Field: "address", Header: "物理地址"},
    {Field: "last_inform_at", Header: "最后心跳时间"},
}
```

---

### G05: 列自定义配置（用户级）

**需求**：每个用户可配置设备列表显示哪些列。

**新增迁移**：

```sql
CREATE TABLE user_column_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    module      VARCHAR(32) NOT NULL DEFAULT 'device_list',
    columns     JSONB NOT NULL,        -- ["serial_number","status","device_name",...]
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module)
);
```

**接口设计**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/columns` | GET | 获取当前用户列配置（无配置返回默认） |
| `/api/v1/devices/columns` | PUT | 保存用户列配置 |

**可用列来源**：`devices` 表字段 + `device_info` 表字段，合并为统一的列列表。

---

### G06: 枚举值查询接口

**需求**：前端下拉选单需要后端提供枚举值。

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/enums` | GET | 返回所有枚举值 |

**响应**：

```json
{
  "carrier": [
    {"value": "cmcc", "label": "中国移动"},
    {"value": "ctcc", "label": "中国电信"},
    {"value": "cucc", "label": "中国联通"}
  ],
  "technology": [{"value": "lte", "label": "LTE (4G)"}, {"value": "nr", "label": "NR (5G)"}],
  "status": [{"value": "active", "label": "在线"}, {"value": "offline", "label": "离线"}, ...],
  "rf_status": [{"value": "on", "label": "开启"}, {"value": "off", "label": "关闭"}, {"value": "error", "label": "异常"}],
  "cell_status": [{"value": "normal", "label": "正常"}, {"value": "fault", "label": "故障"}, ...],
  "project_status": [{"value": "building", "label": "在建"}, {"value": "delivered", "label": "已交付"}, ...]
}
```

纯内存计算，无数据库查询。

---

### G07: 多字段模糊搜索扩展

**当前**：`DeviceFilter.Search` 仅匹配 `serial_number` 和 `site_name`。

**扩展后**：关键词同时匹配 `devices` + `device_info` 中的多个字段。

```go
if filter.Search != nil {
    keyword := "%" + *filter.Search + "%"
    builder = builder.Where(
        sq.Or{
            sq.ILike{"d.serial_number": keyword},
            sq.ILike{"d.site_name": keyword},
            sq.ILike{"d.manufacturer": keyword},
            sq.ILike{"d.model_name": keyword},
            sq.ILike{"d.firmware_version": keyword},
            sq.Expr("host(d.ip_address)::text ILIKE ?", keyword),
            sq.ILike{"di.device_name": keyword},   // 新增
            sq.ILike{"di.address": keyword},        // 新增
        },
    )
}
```

---

### G08: 设备操作 — 激活/去激活

复用现有 `TransitionStatus` 逻辑。

```
PUT /api/v1/devices/:id/activate       — 状态 → active
PUT /api/v1/devices/:id/deactivate     — 状态 → maintenance
```

`state_machine.go` 已支持 `Active ↔ Maintenance` 转换，无需���改。

---

### G09: 设备操作 — 射频开关

通过 TR069 SetParameterValues 下发射频参数。

```
PUT /api/v1/devices/:id/rf-switch      — body: {"enabled": true/false}
```

实现：通过 `Carrier` 接口获取 RF 控制参数路径 → `SetParameters()` 下发 → 下发成功后更新 `device_info.rf_status`。

---

### G10: 设备操作 — 日志收集

通过 TR069 Upload RPC 触发设备上传日志到 MinIO。

```
POST /api/v1/devices/:id/log-collect
body: {"log_type": "system", "start_time": "...", "end_time": "..."}
```

流程：生成 MinIO presigned URL → Upload RPC 命令入队 → 设备上传 → TransferComplete 回调。

---

### G11: 设备操作 — 报文收集

厂商特定功能，通过参数下发触发抓包 + Upload 上传。

```
POST /api/v1/devices/:id/packet-capture
body: {"duration": 60}
```

需要 `Carrier` 接口适配器提供抓包控制参数路径。

---

### G12: 设备详情聚合视图

```
GET /api/v1/devices/:id/summary
```

并行查询 device + device_info + alarm + kpi + license，用 `errgroup` 聚合返回：

```json
{
  "device": { /* devices + device_info 合并 */ },
  "alarm_summary": {"critical": 2, "major": 5, "total_active": 10},
  "latest_kpi": {"rsrp": -85.2, "sinr": 12.5},
  "license": {"status": "active", "expires_at": "..."}
}
```

---

### G13: 逻辑删除

```sql
ALTER TABLE devices ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_devices_deleted ON devices (deleted_at) WHERE deleted_at IS NULL;
```

- `pg_repository.go` 所有查询加 `WHERE deleted_at IS NULL`
- `Delete()` 改为 `UPDATE SET deleted_at = NOW(), status = 'decommissioned'`
- `device_info` 行跟随 `ON DELETE CASCADE` 或同步标记

---

### G14: 敏感字段加密

**建议推迟到 V2**。理由：
1. 经纬度在 topology/geo 计算中使用，加密后无法做范围查询
2. 安全要求需与运营商确认后再定

---

## 4. 实施优先级与排期

### Phase 1（基础搭建，~8d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G01 | `device_info` 表 + CRUD + API | 2d | 迁移 |
| G03 | 参数自动同步机制 | 2d | G01, Carrier 接口 |
| G02 | 列表 JOIN 查询 + 扩展过滤 | 2d | G01 |
| G06 | 枚举值接口 | 0.5d | 无 |
| G07 | 多字段模糊搜索 | 0.5d | G02 |
| G08 | 激活/去激活 | 0.5d | 无 |

### Phase 2（功能完善，~7d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G05 | 列自定义配置 | 1d | 迁移 |
| G04 | 设备列表导出 | 2d | G02 |
| G09 | 射频开关 | 1d | Carrier 接口 |
| G12 | 设备详情聚合 | 1.5d | alarm/kpi 模块 |
| G13 | 逻辑删除 | 1.5d | 全局改造 |

### Phase 3（高级操作，~4d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G10 | 日志收集 | 2d | transfer 模块 |
| G11 | 报文收��� | 2d | Carrier + transfer |

### Phase 4（安全加固）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G14 | 敏感字段加密 | 3d | 运营商安全要求确认 |

---

## 5. 与设计文档的差异说明

| 设计文档内容 | 本方案处理 | 理由 |
|-------------|----------|------|
| `device_info` 单表存所有字段 | 拆分为 `devices`（核心）+ `device_info`（扩展） | `devices` 已有完整的 ACS 写入链路，不宜合并 |
| BIGINT 自增主键 | `device_info.device_id` 引用 `devices.id` (UUID) | 项目统一 UUID |
| `network_type`（2G/4G/5G） | 沿用 `devices.technology`（lte/nr） | 系统定位 4G/5G 小基站，无 2G |
| `connect_status`（在线/离线/异常/未激活/维护中） | 沿用 `devices.status` 7 态枚举 | 已有完整状态机 |
| `device_group` 字段在设备表 | 保持独立 `device_groups` 表（topology 模块） | 已实现层级分组 + 成员关联 |
| 逻辑删除 `is_delete` | 使用 `deleted_at` 时间戳 | 更灵活，可知删除时间 |
| 独立操作日志表 | 复用 `syslog/` 审计日志模块 | 避免重复建设 |
| 射频参数仅在主表 | 同时保留 `device_parameters`（完整参数树）+ `device_info`（快捷列） | `device_parameters` 是 TR069 标准模型，`device_info` 是查询优化 |

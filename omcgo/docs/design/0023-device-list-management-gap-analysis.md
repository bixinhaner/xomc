# 0023 设备列表管理系统 — 差距分析与实现方案

> 基于《2/4/5G设备列表管理系统 后端开发设计文档》及 `files/Back-end/` 目录下全部 9 份设计文档与当前代码库的对比分析。

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
    device_id         UUID PRIMARY KEY,
    -- 注意：devices 是分区表，PG 不支持对分区表的外键引用，
    -- 因此不加 REFERENCES 约束，由应用层维护 1:1 关系。

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
CREATE INDEX idx_device_info_rf_status      ON device_info (rf_status);
CREATE INDEX idx_device_info_cell_status    ON device_info (cell_status);
CREATE INDEX idx_device_info_project_status ON device_info (project_status);

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
| 固件上传/列表/删除 | ✅ 已实现 | `software/` 模块 |
| 升级任务（单设备/批量） | ✅ 已实现 | `software/` TriggerUpgrade/BatchUpgrade |
| STUN/UDP 穿越 | ✅ 已实现 | `acs/stun/` 模块 |
| Connection Request 调度 | ✅ 已实现 | `acs/connreq/` Dispatcher |

### 2.2 Phase 1 已实现功能（本轮实施）

| # | 功能 | 状态 | 实现文件 |
|---|------|------|---------|
| G01 | `device_info` 表与模块搭建 | ✅ 已完成 | `migrations/000062_create_device_info.up.sql`、`device/device_info_model.go`、`device/device_info_repository.go`、`device/device_info_pg_repository.go`、`device/device_info_handler.go` |
| G02 | 设备列表 JOIN 查询 + 扩展过滤 | ✅ 已完成 | `device/device_info_pg_repository.go` ListDevicesWithInfo()、`device/repository.go` DeviceFilter 扩展、`device/handler.go` 新增查询参数 |
| G03 | 参数自动同步到 `device_info` | ✅ 已完成 | `device/info_sync.go` InfoSyncer、`core/carrier/carrier.go` GetInfoParamMapping()、三运营商适配器实现 |
| G06 | 枚举值查询接口 | ✅ 已完成 | `device/device_info_handler.go` GET /api/v1/devices/enums |
| G07 | 多字段模糊搜索扩展 | ✅ 已完成 | `device/device_info_pg_repository.go` ListDevicesWithInfo() 搜索条件 |
| G08 | 设备操作：激活/去激活 | ✅ 已完成 | `device/device_info_handler.go` PUT /activate、/deactivate |

### 2.3 未实现功能（原始差距）

来源：《2/4/5G设备列表管理系统 后端开发设计文档》

| # | 功能 | 设计文档章节 | 优先级 | 复杂度 |
|---|------|------------|--------|--------|
| G04 | 设备列表导出（CSV/Excel） | 3.2.6 | **P1** | 中 |
| G05 | 列自定义配置（用户级） | 3.2.2, 4.3 | **P1** | 低 |
| G09 | 设备操作：射频开关 | 3.2.4 | **P1** | 中 |
| G10 | 设备操作：日志收集 | 3.2.4 | **P2** | 中 |
| G11 | 设备操作：报文收集 | 3.2.4 | **P2** | 中 |
| G12 | 设备详情聚合（告警/KPI/License 摘要） | 3.2.3 | **P2** | 中 |
| G13 | 逻辑删除 | 5.2 | **P2** | 中 |
| G14 | 敏感字段加密（MAC/经纬度） | 5.3 | **P3** | 中 |

### 2.4 新增未实现功能（扩展分析）

来源：`files/Back-end/` 目录下 8 份补充设计文档。

| # | 功能 | 来源文档 | 优先级 | 复杂度 | 说明 |
|---|------|---------|--------|--------|------|
| G15 | 设备日志收集 — 即时模式 | 日志收集功能逻辑设计文档 §3 | **P1** | 高 | 通过 TR069 Upload RPC 触发运行日志/安全日志上传，7 态任务状态机，超时/重试 |
| G16 | 设备日志收集 — 周期模式 | 日志收集功能逻辑设计文档 §4 | **P2** | 高 | 定期自动收集日志，cron 调度，粒度配置（每小时/每天/自定义） |
| G17 | 日志收集 — 平台适配 | 日志收集功能逻辑设计文档 §6 | **P1** | 中 | 4G/5G 平台参数路径差异（FileType、URL、Username、Password），Carrier 适配��扩展 |
| G18 | 固件升级回退 | 设备升级回退流程设计文档 §7 | **P1** | 高 | 回退 = GetParameterValues 查询 ROLLBACK_ENABLE → SetParameterValues 触发回退 → 等待 RebootComplete |
| G19 | 升级任务挂起/恢复/终止 | 设备升级回退流程设计文档 §8 | **P2** | 中 | 任务暂停（不再下发）/恢复（继续排队）/终止（标记放弃），当前仅支持创建和完成 |
| G20 | 5G 升级完成事件处理 | 设备升级回退流程设计文档 §5.2 | **P1** | 中 | 5G 设备 Inform 携带事件码 102 (M_Download + TRANSFER COMPLETE) 时标记升级成功 |
| G21 | 设备注册 — 批量 Excel 导入 | 设备注册功能说明 §3 方式二 | **P1** | 中 | 下载 Excel 模板 → 填写设备信息 → 上传解析 → 校验（SN格式/GPS范围/高度） → 入库 → 错误报告 |
| G22 | 设备预注册 | 设备注册功能说明 §3 方式一 | **P2** | 低 | 管理员手动录入 SN 等基本信息，设备上线前预创建记录，首次 Inform 时匹配合并 |
| G23 | 异常重启日志 — BOOT 检测 | 设备异常重启日志 §4 | **P1** | 高 | Inform 事件码包含 `1 BOOT`（非 0 BOOTSTRAP）时判定为异常重启，提取 MainReason/DetailReason 参数 |
| G24 | 异常重启日志 — 自动收集 | 设备异常重启日志 §5 | **P1** | 高 | BOOT 检测后自动��队，通过 GetParameterValues 获取故障日志 URL，Upload RPC 收集日志文件 |
| G25 | 异常重启日志 — 手动收集 | 设备异常重启日志 §6 | **P2** | 中 | 运维人员手动触发重启日志收集，SetParameterValues 设置 FaultLogURL → Upload → TransferComplete |
| G26 | 异常重启日志 — 列表/导出/清理 | 设备异常重启日志 §8-§11 | **P2** | 中 | 列表查询（按时间/SN/原因过滤）、详情查看、文件下载、批量导出、自动清理策略（单设备上限 + 总量上限 + 磁盘阈值） |
| G27 | STUN UDP — 心跳保活增强 | STUN-UDPServer §5 | **P3** | 低 | 当前 STUN 模块已实现基础功能，需增强: UDP 心跳超时检测���连接状态���步到 device_info |

---

## 3. 各功能实现方案

### G01: `device_info` 表与模块搭建 ✅ 已完成

**实现文件**：
- `migrations/000062_create_device_info.up.sql` — DDL（无外键约束，因 devices 是分区表）
- `internal/device/device_info_model.go` — DeviceInfo、DeviceWithInfo、UpdateDeviceInfoRequest 结构体
- `internal/device/device_info_repository.go` — DeviceInfoRepository 接口
- `internal/device/device_info_pg_repository.go` — PostgreSQL 实现（Squirrel + pgx）
- `internal/device/device_info_handler.go` — GET/PUT /devices/:id/info

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/:id/info` | GET | 获取设备扩展信息 |
| `/api/v1/devices/:id/info` | PUT | 更新手动填写字段（name/address/remark/project_status/height） |

**实现要点**：
- 设备通过 `RegisterFromInform` 注册时，自动创建空的 `device_info` 记录（设置 first_online_time）
- PUT 仅允许更新手动字段，无线参数/状态字段由同步机制自动填充
- `DeviceService.SetDeviceInfoRepo()` 注入，nil 安全降级

---

### G02: 设备列表 JOIN 查询 + 扩展过滤 ✅ 已完成

**改动文件**：
- `device/device_info_pg_repository.go` — ListDevicesWithInfo() 方法
- `device/repository.go` — DeviceFilter 扩展
- `device/handler.go` — 新增查询参数

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
    // --- 新增 ---
    Manufacturer  *string    // devices.manufacturer 精准
    ProductClass  *string    // devices.product_class 精准
    RFStatus      *string    // device_info.rf_status 精准
    CellStatus    *string    // device_info.cell_status 精准
    ProjectStatus *string    // device_info.project_status 精准
    model.ListRequest
}
```

**查询模式**：`FROM devices d LEFT JOIN device_info di ON di.device_id = d.id`，返回 `DeviceWithInfo` 平铺结构。

**排序扩展**：支持 `device_name`、`rf_status`、`cell_status`、`bandwidth`、`transmit_power` 等 device_info 字段排序。

**降级策略**：`ListDevicesWithInfo()` 在 deviceInfoRepo 为 nil 时，降级为标准 `List()` 结果包装。

---

### G03: 参数自动同步到 `device_info` ✅ 已完成

**实现文件**：
- `internal/device/info_sync.go` — InfoSyncer
- `internal/core/carrier/carrier.go` — Carrier 接口新增 GetInfoParamMapping()
- `internal/core/carrier/cmcc/adapter.go` — CMCC LTE + NR 参数映射
- `internal/core/carrier/ctcc/adapter.go` — CTCC LTE + NR 参数映射
- `internal/core/carrier/cucc/adapter.go` — CUCC NR 参数映射

**同步策略**：

| 触发时机 | 同步内容 |
|---------|---------|
| 设备注册（Bootstrap Inform） | 创建 `device_info`，记录 first_online_time |
| 周期性 Inform（UpdateFromInform） | 通过 InfoSyncer.SyncFromParameters 更新无线参数 |
| 设备离线（HeartbeatMonitor） | InfoSyncer.RecordOffline 记录 last_offline_time |

**Carrier 接口方法**：

```go
// 每个运营商适配器实现此方法，返回 TR069 参数路径 → device_info 字段的映射
GetInfoParamMapping(tech Technology) map[string]string

// CMCC LTE 示例映射
"Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity" → "eci"
"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID"        → "pci"
"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL"         → "freq_point"
"Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth"      → "bandwidth"
```

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

### G06: 枚举值查询接口 ✅ 已完成

**实现文件**：`internal/device/device_info_handler.go`

**端点**：`GET /api/v1/devices/enums`

返回枚举值：carrier、technology、status、rf_status、cell_status、mme_status、sync_status、project_status。纯内存计算，无数据库查询。

---

### G07: 多字段模糊搜索扩展 ✅ 已完成

**实现文件**：`internal/device/device_info_pg_repository.go` ListDevicesWithInfo()

搜索条件已扩展为同时匹配：`d.serial_number`、`d.site_name`、`d.manufacturer`、`d.model_name`、`di.device_name`、`di.address`。

---

### G08: 设备操作 — 激活/去激活 ✅ 已完成

**实现文件**：`internal/device/device_info_handler.go`

```
PUT /api/v1/devices/:id/activate       — 状态 → active
PUT /api/v1/devices/:id/deactivate     — 状态 → maintenance
```

复用 `state_machine.go` 的 `TransitionStatus` 逻辑。

---

### G09: 设备操作 — 射频开关

通过 TR069 SetParameterValues 下发射频参数。

```
PUT /api/v1/devices/:id/rf-switch      — body: {"enabled": true/false}
```

实现：通过 `Carrier` 接口获取 RF 控制参数路径 → `SetParameters()` 下发 → 下发成功后更新 `device_info.rf_status`。

---

### G10: 设备操作 — 日志收集（基础）

通过 TR069 Upload RPC 触发设备上传日志到 MinIO。

```
POST /api/v1/devices/:id/log-collect
body: {"log_type": "system", "start_time": "...", "end_time": "..."}
```

流程：生成 MinIO presigned URL → Upload RPC 命令入队 → 设备上传 → TransferComplete 回调。

> 注：G15/G16/G17 是日志收集的完整实现方案，G10 为最简化版本。

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

### G15: 设备日志收集 — 即时模式（新增）

**来源**：《日志收集功能逻辑设计文档》§3

**需求**：运维人员手动触发设备运行日志/安全日志收集，支持 4G/5G 差异化参数路径。

**新增文件**：

```
internal/device/logcollect/
    model.go              — LogCollectTask 结构体、7 态状态机
    repository.go         — 接口定义
    pg_repository.go      — PostgreSQL 实现
    handler.go            — HTTP 处理
    service.go            — 业务逻辑（入队、状态流转、超时检测）
migrations/
    000XXX_create_log_collect_tasks.up.sql
```

**任务状态机**（7 态）：

```
PENDING(0) → QUEUED(1) → DOWNLOADING(2) → UPLOADING(3) → COMPLETED(4)
                                                      ├─→ FAILED(5)
                                                      └─→ TIMEOUT(6)
```

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/:id/log-collect` | POST | 创建即时日志收集任务 |
| `/api/v1/devices/:id/log-collect` | GET | 查询设备日志收集记录 |
| `/api/v1/log-collect/tasks` | GET | 全局日志收集任务列表 |
| `/api/v1/log-collect/tasks/:task_id` | GET | 任务详情 |
| `/api/v1/log-collect/tasks/:task_id/file` | GET | 下载日志文件 |

**平台适配（Carrier 接口扩展）**：

```go
// 新增 Carrier 方法
GetLogCollectParams(tech Technology) LogCollectParamPaths

type LogCollectParamPaths struct {
    FileType string  // Upload RPC 的 FileType 参数值
    URL      string  // 设备上传目标 URL 参数路径
    Username string  // FTP/HTTP 认证用户名参数路径
    Password string  // FTP/HTTP 认证密码参数路径
}
```

**实现要点**：
- 入队前检查设备在线状态和已有任务（避免重复）
- 生成 MinIO presigned URL 作为上传目标
- 通过 `cmdQueue` 下发 Upload RPC
- 监听 TransferComplete 事件标记完成
- 超时检测：定时扫描 QUEUED/DOWNLOADING 状态超过阈值的任务

---

### G16: 设备日志收集 — 周期模式（新增）

**来源**：《日志收集功能逻辑设计文档》§4

**需求**：按计划定期自动收集设备日志。

**新增文件**：

```
internal/device/logcollect/
    periodic_service.go   — 周期任务调度
migrations/
    000XXX_create_log_collect_schedules.up.sql
```

**数据模型**：

```sql
CREATE TABLE log_collect_schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL,
    log_type    VARCHAR(32) NOT NULL,      -- running_log / security_log
    cron_expr   VARCHAR(64) NOT NULL,      -- cron 表达式
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_by  VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**实现要点**：
- 使用 `robfig/cron/v3` 调度
- 每次触发创建一条 `log_collect_tasks` 记录
- 复用 G15 的即时收集流程

---

### G17: 日志收集 — 平台适配（新增）

**来源**：《日志收集功能逻辑设计文档》§6

**需求**：4G LTE 和 5G NR 平台的日志收集参数路径不同。

**改动文件**：三个运营商适配器（cmcc/ctcc/cucc adapter.go）

**4G LTE 参数路径**：
```
FileType = "4 Vendor Log File"
URL      = "Device.DeviceInfo.VendorLogFile.1.URL"
Username = "Device.DeviceInfo.VendorLogFile.1.Username"
Password = "Device.DeviceInfo.VendorLogFile.1.Password"
```

**5G NR 参数路径**：
```
FileType = "4 Vendor Log File"
URL      = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.URL"
Username = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.Username"
Password = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.Password"
```

---

### G18: 固件升级回退（新增）

**来源**：《设备升级回退流程设计文档》§7

**需求**：升级后发现问题，回退到前一版本固件。

**改动文件**：`internal/software/service.go`、`internal/software/handler.go`

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/upgrade-tasks/:id/rollback` | POST | 对已完成的升级任务发起回退 |

**回退流程**：

```
1. GetParameterValues(SoftwareImage.{i}.Active) 确认当前激活分区
2. GetParameterValues(SoftwareImage.{i}.RollbackEnable) 检查是否支持回退
3. SetParameterValues(SoftwareImage.{i}.RollbackEnable = true) 触发回退
4. 等待设备重启 + Inform (event=102 或 1 BOOT)
5. GetParameterValues 确认版本已恢复
```

**平台差异**：

| 平台 | 回退参数路径 |
|------|------------|
| 4G LTE | `Device.Services.FAPService.1.FAPControl.LTE.SoftwareImage.{i}.RollbackEnable` |
| 5G NR | `Device.Services.FAPService.1.FAPControl.NR.SoftwareImage.{i}.RollbackEnable` |

---

### G19: 升级任务挂起/恢复/终止（新增）

**来源**：《设备升级回退流程设计文档》§8

**需求**：批量升级场景下，需要暂停/恢复/终止正在进行的升级任务。

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/upgrade-tasks/:id/suspend` | PUT | 挂起（不再向设备下发） |
| `/api/v1/upgrade-tasks/:id/resume` | PUT | 恢复（重新排入队列） |
| `/api/v1/upgrade-tasks/:id/terminate` | PUT | 终止（标记放弃） |

---

### G20: 5G 升级完成事件处理（新增）

**来源**：《设备升级回退流程设计文档》§5.2

**需求**：5G 设备升级完成后通过 Inform 事件码 `102`（M Download + TRANSFER COMPLETE）通知。

**改动文件**：`internal/device/inform_handler.go`

**实现要点**：
- InformHandler 解析事件码列表，检测 `102` 或 `M Download`
- 匹配到活跃的升级任务后标记为 COMPLETED
- 更新 `devices.firmware_version` 为新版本

---

### G21: 设备注册 — 批量 Excel 导入（新增）

**来源**：《设备注册功能说明》§3 方式二

**需求**：通过 Excel 模板批量导入设备信息，用于预注册。

**新增文件**：

```
internal/device/import_handler.go     — 文件上传 + 解析
internal/device/import_service.go     — 校验 + 批量入库
```

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/import/template` | GET | 下载 Excel 模板 |
| `/api/v1/devices/import` | POST | 上传 Excel 文件，批量导入 |

**Excel 模板字段**：设备序列号（必填）、设备名称、站点名称、经度、纬度、高度、地址、备注、设备分组

**校验规则**：
- SN 格式校验（长度、字符集）
- SN 唯一性检查（不允许与已有设备重复）
- 经度 -180~180，纬度 -90~90
- 高度 0~1000m
- 设备分组存在性校验

**实现要点**：
- 使用 `excelize/v2` 解析上传文件
- 分批校验 + 入库（每批 100 条）
- 返回导入结果（成功数 + 失败列表 + 失败原因）
- 预注册设备 status = `registered`，等待首次 Inform 匹配

---

### G22: 设备预注册（新增）

**来源**：《设备注册功能说明》§3 方式一

**需求**：管理员手动录入设备 SN 等基本信息，设备上线前预创建记录。

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/pre-register` | POST | 预注册单台设备 |

**实现要点**：
- 创建 `devices` 记录（status=registered）+ `device_info` 记录
- 首次 Inform 时通过 SN 匹配，更新为 active 并合并 Inform 信息
- 已在 `RegisterFromInform` 中有 `GetBySerialNumber` 逻辑，扩展为: 存在则 update，不存在则 create

---

### G23: 异常重启日志 — BOOT 检测（新增）

**来源**：《设备异常重启日志功能流程规范文档》§4

**需求**：检测设备异常重启（Inform 事件码 `1 BOOT` 但不含 `0 BOOTSTRAP`），提取故障原因。

**改动文件**：`internal/device/inform_handler.go`

**检测逻辑**：

```go
// 在 handleInform 中
events := inform.Events // e.g. ["1 BOOT", "2 PERIODIC"]
hasBoot := slices.Contains(events, "1 BOOT")
hasBootstrap := slices.Contains(events, "0 BOOTSTRAP")

if hasBoot && !hasBootstrap {
    // 异常重启！提取原因
    mainReason := paramValues["Device.DeviceInfo.X_VENDOR_MainFaultReason"]
    detailReason := paramValues["Device.DeviceInfo.X_VENDOR_DetailFaultReason"]
    // 发布事件 device.reboot.abnormal
}
```

**平台差异（故障原因参数路径）**：

| 平台 | MainReason | DetailReason |
|------|-----------|-------------|
| 4G LTE | `Device.DeviceInfo.X_BAICELLS_MainFaultReason` | `Device.DeviceInfo.X_BAICELLS_DetailFaultReason` |
| 5G NR | `Device.Services.FAPService.1.FAPControl.NR.X_BAICELLS_MainFaultReason` | 同级 DetailFaultReason |

---

### G24: 异常重启日志 — 自动收集（新增）

**来源**：《设备异常重启日志功能流程规范文档》§5

**需求**：检测到异常重启后，自动收集设备故障日志。

**新增文件**：

```
internal/device/rebootlog/
    model.go              — RebootLogEntry 结构体
    repository.go         — 接口定义
    pg_repository.go      — PostgreSQL 实现
    service.go            — 自动/手动收集逻辑
    handler.go            — HTTP 处理
migrations/
    000XXX_create_device_reboot_logs.up.sql
```

**数据模型**：

```sql
CREATE TABLE device_reboot_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    serial_number   VARCHAR(128) NOT NULL,
    reboot_time     TIMESTAMPTZ NOT NULL,
    main_reason     VARCHAR(256),
    detail_reason   TEXT,
    collection_mode VARCHAR(20) NOT NULL,  -- auto / manual
    log_file_path   VARCHAR(512),          -- MinIO 路径
    log_file_size   BIGINT,
    status          VARCHAR(20) NOT NULL,  -- detected / collecting / collected / failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reboot_logs_device ON device_reboot_logs (device_id, reboot_time DESC);
```

**流程**：
1. 订阅 `device.reboot.abnormal` 事件
2. 创建 `device_reboot_logs` 记录（status=detected）
3. 通过 GetParameterValues 获取 FaultLogFile URL
4. 如有日志文件 → Upload RPC 收集到 MinIO
5. TransferComplete → 更新 status=collected

---

### G25: 异常重启日志 — 手动收集（新增）

**来源**：《设备异常重启日志功能流程规范文档》§6

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/devices/:id/reboot-logs/collect` | POST | 手动触发重启日志收集 |

**流程**：SetParameterValues 设置 FaultLogURL → 设备主动上传 → TransferComplete 回调。

---

### G26: 异常重启日志 — 列表/导出/清理（新增）

**来源**：《设备异常重启日志功能流程规范文档》§8-§11

**API 端点**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/reboot-logs` | GET | 异常重启日志列表（分页、过滤） |
| `/api/v1/reboot-logs/:id` | GET | 日志详情 |
| `/api/v1/reboot-logs/:id/file` | GET | 下载日志文件 |
| `/api/v1/reboot-logs/export` | POST | 批量导出 |

**清理策略**：
- 单设备日志上限（如 100 条），超出删除最早记录
- 全局总量上限（如 100,000 条）
- 磁盘使用率阈值（如 > 80% 时触发清理）
- 定时清理任务（cron）

---

### G27: STUN UDP — 心跳保活增强（新增）

**来源**：《STUN-UDPServer业务说明》§5

**当前状态**：`acs/stun/` 模块已实现基础 STUN 功能和 Store。

**增强需求**：
- UDP 心跳超时检测（超过 2×interval 未收到心跳标记设备 unreachable）
- 连接状态同步到 `device_info`（通过 InfoSyncer）

---

## 4. 实施优先级与排期

### Phase 1 ✅ 已完成（基础搭建）

| 序号 | 功能 | 状态 |
|------|------|------|
| G01 | `device_info` 表 + CRUD + API | ✅ 完成 |
| G03 | 参数��动同步机制 | ✅ 完成 |
| G02 | 列表 JOIN 查询 + 扩展过滤 | ✅ 完成 |
| G06 | ��举值接口 | ✅ 完成 |
| G07 | 多字段模糊搜索 | ✅ 完成 |
| G08 | 激活/去激活 | ✅ 完成 |

### Phase 2（核心操作，~10d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G09 | 射频开关 | 1d | Carrier 接口 |
| G04 | 设备列表导出 | 2d | G02 |
| G21 | 批量 Excel 导入 | 2d | excelize 库 |
| G22 | 设备预注册 | 1d | G21 |
| G12 | 设备详情聚合 | 1.5d | alarm/kpi 模块 |
| G20 | 5G 升级完成事件 | 1d | software 模块 |
| G05 | 列自定义配置 | 1d | 迁移 |

### Phase 3（日志收集体系，~12d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G15 | 日志收集 — 即时模式 | 3d | Upload RPC, MinIO |
| G17 | 日志收集 — 平台适配 | 1d | Carrier 接口 |
| G16 | 日志收集 — 周期模式 | 2d | G15 |
| G23 | 异常重启 — BOOT 检测 | 1.5d | InformHandler |
| G24 | 异常重启 — 自动收集 | 2d | G23, Upload RPC |
| G25 | 异常重启 — 手动收集 | 1d | G24 |
| G26 | 异常重启 — 列表/导出/清理 | 1.5d | G24 |

### Phase 4（升级增强，~5d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G18 | 固件升级回退 | 2d | software 模块 |
| G19 | 升级任务挂起/恢复/终止 | 1.5d | G18 |
| G11 | 报文收集 | 1.5d | Carrier + Upload |

### Phase 5（安全与增强，~5d）

| 序号 | 功能 | 工作量 | 依赖 |
|------|------|--------|------|
| G13 | 逻辑删除 | 1.5d | 全局改造 |
| G27 | STUN 心跳增强 | 1d | stun 模块 |
| G14 | 敏感字段加密 | 3d | 运营商安全要求确认 |

---

## 5. 设计文档来源索引

| 文档 | 路径 | 涉及 Gap 项 |
|------|------|-----------|
| 2/4/5G设备列表管理系统 — 后端开发设计文档 | `files/Back-end/2_4_5G设备列表管理系统 - 后端开发设计文档.md` | G01-G14（原始） |
| 日志收集功能逻辑设计文档 | `files/Back-end/日志收集功能逻辑设计文档.md` | G15, G16, G17 |
| 设备升级回退流程设计文档 | `files/Back-end/设备升级回退流程设计文档.md` | G18, G19, G20 |
| 设备注册功能说明 | `files/Back-end/设备注册功能说明.md` | G21, G22 |
| 设备异常重启日志功能流程规范文档 | `files/Back-end/设备异常重启日志功能流程规范文档.md` | G23, G24, G25, G26 |
| STUN-UDPServer业务说明 | `files/Back-end/STUN-UDPServer业务说明.md` | G27 |
| TR069报文全解析 | `files/Back-end/TR069报文全解析.md` | 参数路径参考 |
| 系统管理模块开发设计文档 | `files/Back-end/系统管理模块开发设计文档.md` | admin 模块（已实现） |
| 设备注册与设备分组管理系统 | `files/Back-end/设备注册与设备分组管理系统 开发设计文档.md` | G21, G22 + topology（已实现） |

---

## 7. TR069 报文字段全景分析

> 基于《TR069报文全解析.md》中实际抓包的 252 个参数，分析每个字段的存储去向和利用方式。

### 7.1 存储架构与设计原则

#### 核心原则：device_info 是 device_parameters 的「物化视图」

device_info 表**仅提取设备列表页必须展示/过滤/排序的少量字段**（控制在 25 列以内），
其余所有 TR069 参数统一存入 device_parameters K-V 表。设备详情页、配置下发、参数对比等场景直接从 device_parameters 读取。

**不在 device_info 中使用 JSONB 列存储 TR069 参数**。理由：

| # | 问题 | 说明 |
|---|------|------|
| 1 | **数据冗余** | device_parameters 已存储全量参数，JSONB 是重复副本，引入一致性风险 |
| 2 | **丢失参数路径** | JSONB 结构化后丢失 TR069 原始路径，配置下发（SetParameterValues）时需反向查找 |
| 3 | **更新代价高** | 修改 JSONB 中单个字段需 读取→反序列化→修改→序列化→写回 整个 JSON |
| 4 | **违背 TR069 数据模型** | TR069 参数树天然是层级 K-V 结构，device_parameters 正是这种结构的自然映射 |
| 5 | **多实例不可控** | MME 16 组×3 字段=48 参数，License 4 组×8 字段=32 参数，小区数动态变化——JSONB 无法规范化 |

#### 行业规范依据

| 规范 | 要求 | 对存储设计的指导 |
|------|------|-----------------|
| **3GPP 32.600** MO 信息模型 | MO 实例数可变（小区、MME 连接、License 条目） | 多实例对象不应拍平到设备主表，K-V 天然支持 |
| **BBF TR-069 Amendment 6** | 参数路径 = 对象层级地址，多实例用 `{i}` | 保留原始路径是配置下发的前提，K-V 完美保留 |
| **运营商北向接口规范** | 设备列表固定 15-20 列，详情页按子树分 Tab | device_info 精简列 + device_parameters 按前缀查询 |
| **商用网管实践**���U2000/NetNumen） | 列表页固定列，详情页从参数 K-V 存储动态组装 | 不把 MME Pool、License 塞进设备主表 |

#### 252 个参数的存储去向

| 存储层 | 参数数量 | 说明 |
|--------|---------|------|
| devices 表具名列 | ~8 | 设备身份/版本/IP，Inform 自动写入 |
| device_info 具名列 | ~18 | 列表展示/过滤必需的运行状态字段（仅新增 `num_of_cells` 1 列） |
| device_parameters K-V | **全部 252** | 所有参数完整保留原始路径（含已提取到快查列的） |
| 独立业务表 | 告警参数 | CurrentAlarm → alarm 模块独立管理 |

#### 容量估算（10 万设备）

| 方案 | 存储量 | 写入模式 |
|------|--------|---------|
| device_info 26 列 | ~50 MB（10 万行 × 500B） | 每 5 分钟 Inform 更新 ~13 列 |
| device_parameters 252 参数/设备 | ~5 GB（2500 万行 × 200B） | Inform + ACS 查询后批量 UPSERT |

5 GB 对 PostgreSQL 完全可接受（有 `varchar_pattern_ops` 前缀索引），
不值得在 device_info 增加 JSONB 冗余列来避免一次前缀查询。

### 7.1.1 参数来源分类

| 分类 | 来源 | 参数数量 | 特点 |
|------|------|---------|------|
| **A. Inform 自动携带** | 每次 Inform 设备主动上报 | 37 | 实时性最高，自动获取，无需额外 RPC |
| **B. ACS 主动查询** | GetParameterValues 按需获取 | 215 | 需 ACS 主动发起查询，可按需调整查询频率 |

**三层存储模型**：

```
┌──────────────────────────────────────────────────────────────┐
│  Tier 1: 快查列 — devices / device_info 表的具名列            │
│  用途：设备列表展示、过滤、排序（~26 列）                      │
│  准入标准：列表页必须展示 或 必须支持 WHERE/ORDER BY           │
│  示例：rf_status, cell_status, eci, pci, bandwidth            │
├──────────────────────────────────────────────────────────────┤
│  Tier 2: 完整参数树 — device_parameters K-V 表                │
│  用途：所有 TR069 参数按原始路径 K-V 存储                      │
│  场景：设备详情页、参数配置/对比/审计、配置下发                  │
│  已实现：Inform 后全量写入 device_parameters                   │
├──────────────────────────────────────────────────────────────┤
│  Tier 3: 独立业务表 — alarms / license / etc                  │
│  用途：有独立生命周期的业务数据                                 │
│  示例：告警 → alarm 模块，License → license 模块               │
└──────────────────────────────────────────────────────────────┘
```

---

### 7.2 Inform 参数详细分析（37 个）

Inform 参数在每次心跳时自动携带，实时性最高。

#### 7.2.1 已捕获到快查列（8 个） ✅

| # | TR069 参数路径 | 值示例 | 存储位置 | 快查列 |
|---|---------------|--------|---------|--------|
| 10 | `Device.DeviceInfo.SoftwareVersion` | BaiBLQ_5.1.10 | devices | firmware_version |
| 3 | `Device.DeviceInfo.HardwareVersion` | A01 | device_info | hardware_version |
| 24 | `FAPService.1.FAPControl.LTE.RFTxStatus` | true | device_info | rf_status |
| 23 | `FAPService.1.FAPControl.LTE.OpState` | true | device_info | cell_status |
| 2 | `Device.DeviceInfo.FAP_adminstate` | true | device_info | cell_status |
| 22 | `FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList` | 314030,46000 | device_info | plmn |
| 16 | `Device.DeviceInfo.X_COM_STATION_RUN_Time` | 40d 4h 58m 29s | device_info | run_time |
| 18 | `Device.ManagementServer.ConnectionRequestURL` | http://172.21.100.43:7547 | devices | connection_request_url |

#### 7.2.2 未捕获到快查列的有价值参数（12 个）

| # | TR069 参数路径 | 值示例 | 建议存储 | 建议列名 | 优先级 | 说明 |
|---|---------------|--------|---------|---------|--------|------|
| 15 | `Device.DeviceInfo.X_COM_GPS_Status` | 0 | device_info 快查列 | `gps_status` | **P1** | GPS 状态，影响定位和同步，列表需展示/过滤 |
| 21 | `FAPService.1.FAPControl.LTE.CellOpState` | 0 | device_info 快查列 | 合入 `cell_status` 计算 | **P1** | 小区运行状态（0=停止,1=运行），三维判定 cell_status |
| 28 | `FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus` | Critical | device_info 快查列 | `alarm_severity` | **P1** | 设备最高告警级别，列表红/橙/黄视觉指示 |
| 1 | `Device.DeviceInfo.DnPrefix` | 00256D | device_parameters | — | P3 | DN 前缀，仅详情页参考，不需列表过滤 |
| 11 | `Device.DeviceInfo.X_COM_1588_Status` | 0 | device_parameters | — | P3 | IEEE 1588 同步状态，参与 sync_status 计算 |
| 12 | `Device.DeviceInfo.X_COM_BDS_Status` | 0 | device_parameters | — | P3 | 北斗卫星状态，参与 sync_status 计算 |
| 14 | `Device.DeviceInfo.X_COM_GLONASS_Status` | 0 | device_parameters | — | P3 | GLONASS 卫星状态，参与 sync_status 计算 |
| 25 | `FAPService.1.FAPControl.LTE.Slot` | 1 | device_parameters | — | P3 | 槽位号 |
| 29-35 | `FAPService.2.FAPControl.LTE.*` | — | device_parameters | — | P2 | 小区 2 参数（多实例，按前缀查询） |
| 36-37 | `FAPService.Ipsec.IPSEC_TUNNEL{1,2}_STATUS` | false | device_parameters | — | P3 | IPSec 隧道状态 |
| 4 | `Device.DeviceInfo.ProvisioningCode` | (空) | device_parameters | — | P3 | 配置码，开站流程参考 |
| 5-8 | `Device.DeviceInfo.SAS.*` | — | device_parameters | — | P3 | CBRS SAS 参数（海外部署） |

#### 7.2.3 Inform 参数存储结论

**现状**：Inform 的全部 37 个参数已存入 `device_parameters`（K-V），但仅 8 个提取到快查列。

**建议新增到快查列的 Inform 字段**（仅 2 个，控制列数）：

| 新增列 | 类型 | 来源参数 | 理由 |
|--------|------|---------|------|
| `gps_status` | VARCHAR(20) | `X_COM_GPS_Status` | GPS 状态是基站运维核心指标，列表需展示和过滤 |
| `alarm_severity` | VARCHAR(20) | `X_RADISYS_COM_AlarmStatus` | 设备最高告警级别，红/橙/黄视觉指示 |

**建议增强 cell_status 计算逻辑**：
- `FAP_adminstate`（管理状态）+ `OpState`（运行状态）+ `CellOpState`（小区运行状态）→ 三者综合判定 `cell_status`
- 不新增列，在现有 `cell_status` 列上改进计算逻辑

**其余 Inform 参数全部留在 device_parameters**：
- 多小区参数（`FAPService.2.*`、`FAPService.3.*`）→ 设备详情页按前缀 `FAPService.2.%` 查询
- 卫星状态（BDS/GPS/GLONASS/1588）→ 参与 `sync_status` 快查列的计算逻辑，原始值留 K-V
- IPSec 隧道状态 → 设备详情页展示，不需列表过滤

---

### 7.3 GetParameterValues 参数详细分析（215 个）

以下按 ACS 查询分组分析，与报文中的"第 N 组"对应。

#### 7.3.1 设备身份与状态查询

| 组 | TR069 参数 | 值示例 | 当前存储 | 建议 |
|----|-----------|--------|---------|------|
| 3 | `X_COM_MODULE_TYPE` | pBS3202 | devices.model_name ✅ | 已处理 |
| 10 | `IP.Interface.1.IPv4Address.1.IPAddress` | 172.21.100.43 | devices.ip_address ✅ | 已处理 |
| 2 | `SoftwareCtrl.SystemBackupVersion` | (空) | device_parameters | Tier 2 保留（详情页展示） |
| 16 | `ETH1_STATUS_SPEED` | 100Mb/s | device_parameters | Tier 2 保留（详情页展示） |
| 11 | `3GPPSpecVersion` | E_UTRA | device_parameters | Tier 3 保留 |
| 17 | `ManagementServer.X_COM_ssl_enable` | true | device_parameters | Tier 3 保留 |
| 15 | `X_COM_ApLteturboEnable` | 0 | device_parameters | Tier 3 保留 |

#### 7.3.2 GPS / 定位查询

| 组 | TR069 参数 | 值示例 | 当前存储 | 建议 |
|----|-----------|--------|---------|------|
| 5 | `FAP.GPS.LockedLatitude` | 0 | devices.latitude ✅ | 已处理 |
| 5 | `FAP.GPS.LockedLongitude` | 0 | devices.longitude ✅ | 已处理 |
| 5 | `FAP.GPS.LockedLatitude2` | 0 | device_parameters | Tier 3（小区 2 位置） |
| 5 | `FAP.GPS.LockedLongitude2` | 0 | device_parameters | Tier 3（小区 2 位置） |
| 4 | `X_COM_GPS_Satellite_count` | 0 | device_parameters | Tier 2 保留（详情页展示，不需列表过滤） |
| 4 | `X_COM_GPS_Satellite_level` | (空) | device_parameters | Tier 3 |
| 6 | `AntennaInfo.Height` | 0 | device_info.height ✅ | 已处理 |
| 6 | `AntennaInfo.Height2` | 0 | device_parameters | Tier 3（小区 2 高度） |

#### 7.3.3 天线参数查询

| 组 | TR069 参数 | 值示例 | 当前存储 | 建议 |
|----|-----------|--------|---------|------|
| 11 | `AntennaInfo.Azimuth` | 0 | device_parameters | Tier 2（详情页天线 Tab 展示） |
| 11 | `AntennaInfo.Beamwidth` | 0 | device_parameters | Tier 2 |
| 11 | `AntennaInfo.Downtilt` | 0 | device_parameters | Tier 2 |
| 11 | `AntennaInfo.Gain` | 0 | device_parameters | Tier 2 |
| 11 | `AntennaInfo.HeightType` | AGL | device_parameters | Tier 2 |
| 11 | `AntennaPortsCount` | 2 | device_parameters | Tier 2 |
| 11 | `indoorDeployment` | false | device_parameters | Tier 2（CBRS 相关，非列表必需） |
| 11 | `cbsdCategory` | B | device_parameters | Tier 2（CBRS 海外） |

**天线参数展示方式**：设备详情页按前缀 `AntennaInfo.%` 查询 device_parameters，Service 层组装为结构化 DTO 返回前端。不在 device_info 冗余存储。

#### 7.3.4 射频控制查询

| 组 | TR069 参数 | 值示例 | 当前存储 | 建议 |
|----|-----------|--------|---------|------|
| 1 | `CellConfig.LTE.RAN.RF.X_COM_RadioEnable` | true | device_parameters | 与 `rf_status` 逻辑合并：`RFTxStatus`(Inform 自动) + `RadioEnable`(查询确认) |

#### 7.3.5 许可证信息查询（约 42 个参数）

**当前状态**：全部存入 device_parameters，未提取到快查列。

**报文结构**（4 个 Capacity 条目）：

| 参数 | 含义 | 示例值 |
|------|------|--------|
| `X_COM_LICENSE.Code` | 许可证代码 | FAP |
| `X_COM_LICENSE.GenerateDate` | 生成日期 | 20240831 |
| `X_COM_LICENSE.SeqNum` | 序列号 | 14 |
| `X_COM_LICENSE.Version` | 版本号 | 1 |
| `X_COM_LICENSE.Capacity.{i}.ID` | 容量 ID | FAP044 |
| `X_COM_LICENSE.Capacity.{i}.Description` | 描述 | Hardware Locked License... |
| `X_COM_LICENSE.Capacity.{i}.State` | 状态(1=有效,0=未启用) | 1 |
| `X_COM_LICENSE.Capacity.{i}.ValidPeriod` | 有效天数 | 180 |
| `X_COM_LICENSE.Capacity.{i}.RemainingPeriod` | 剩余天数 | 0 |
| `X_COM_LICENSE.Capacity.{i}.DelayAvaiable` | 宽限天数 | 2 |

**存储建议**：

| 存储目标 | 内容 | 说明 |
|---------|------|------|
| device_info 快查列 | `license_status` VARCHAR(20) | 整体状态计算值：active / expiring / expired |
| device_info 快查列 | `license_expire_days` INTEGER | 最近到期天数（取所有 Capacity 中最小的 RemainingPeriod） |
| device_parameters | 全部 42 个参数 | K-V 原始值 ✅ 已有 |

**不使用 JSONB 存储 License 摘要**。理由：
- device_parameters 已保留完整 License 参数路径，详情页按前缀 `X_COM_LICENSE.%` 查询即可
- JSONB 冗余会引入 42 个参数的双写一致性问题
- `license_status` 和 `license_expire_days` 两个计算值足以支持列表页展示和过滤

**License 详情页展示方式**：Service 层从 device_parameters 按前缀查询，组装为结构化 DTO：
```go
// 从 device_parameters 查询 License 参数
rows := repo.FindByPrefix(ctx, deviceID, "Device.Services.FAPService.1.X_COM_LICENSE.")
// Service 层组装为 LicenseDetail DTO 返回前端
```

#### 7.3.6 MME 池配置查询（约 48 个参数）

**关键注释**（来自报文文档）：
> "MME的数据要加入主表，MME1Status，MMEIp1，PLMNID 数据形成 json 结构 `[{MME1Status:1,MMEIp1:172.19.9.34,PLMNID:46000}]`"

**存储建议**：

| 存储目标 | 内容 | 说明 |
|---------|------|------|
| device_info 快查列 | `mme_status` VARCHAR(20) | 整体 MME 连接状态计算值（根据有效 MME 数量判定） |
| device_parameters | 全部 48 个参数 K-V | ✅ 已有，原始路径完整保留 |

**不使用 JSONB 存储 MME 池**。理由：
- 16 组 × 3 字段 = 48 个参数，device_parameters 已按 `MmePoolConfigParam.{i}.` 前缀完整存储
- 配置下发（修改 MME IP）需要原始 TR069 路径，JSONB 丢失了路径信息
- `mme_status` 计算值（"connected"/"partial"/"disconnected"）足以支持列表过滤

**MME 详情页展示方式**：
```go
// 从 device_parameters 查询 MME 参数
rows := repo.FindByPrefix(ctx, deviceID, "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.")
// Service 层过滤有效条目（Status=1 或 IP 非 0.0.0.0），组装为 []MMEEntry DTO
```

> 注意：16 个 MME 条目中大部分为空（IP=0.0.0.0, Status=0），展示时由 Service 层过滤，仅返回有效条目。

#### 7.3.7 告警信息查询（约 30 个参数）

**当前状态**：告警数据由独立 `alarm` 模块处理。

| 参数类别 | 存储去向 | 说明 |
|---------|---------|------|
| `FaultMgmt.CurrentAlarm.{i}.*` | Tier 4: alarm 模块 | 告警有独立生命周期（发生→确认→清除），不适合存 device_info |
| 告警摘要（活跃数/最高级别） | device_info Tier 1 `alarm_severity` | 从 Inform `AlarmStatus` 参数或 alarm 模块聚合 |

#### 7.3.8 同步状态查询

| 组 | TR069 参数 | 值示例 | 建议 |
|----|-----------|--------|------|
| 19 | `ManagementServer.tfcsManagerPrimsrc` | 7 | 组合计算 `sync_status` |
| 19 | `ManagementServer.tfcsSyncState` | DISP | 组合计算 `sync_status` |

**sync_status 综合计算逻辑**：

```
tfcsSyncState + tfcsManagerPrimsrc + X_COM_GPS_Status + X_COM_BDS_Status + X_COM_1588_Status
→ 综合判定：GPS同步 / 北斗同步 / 1588同步 / NTP同步 / 未同步 / 异常
```

#### 7.3.9 载波聚合与多小区

**关键注释**（来自报文文档）：
> "关于小区2，小区3的参数是否保存，这个是根据基站的载波模式处理的：CA/SC 只处理主小区，DC 处理小区1和小区2，TC 模式处理小区123"

| 参数 | 值示例 | 建议存储 | 说明 |
|------|--------|---------|------|
| `FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells` | 1 | device_info 快查列 `num_of_cells` | 载波模式判断基础，列表需展示/过滤 |
| `FAPService.2.FAPControl.LTE.*` (6 个参数) | — | device_parameters | 多实例参数，按前缀 `FAPService.2.%` 查询 |
| `FAPService.3.FAPControl.LTE.*` (如有) | — | device_parameters | 多实例参数，按前缀 `FAPService.3.%` 查询 |

**多小区参数展示方式**：设备详情页根据 `num_of_cells` 值决定展示哪些小区 Tab，每个 Tab 从 device_parameters 按 `FAPService.{n}.%` 前缀查询。不在 device_info 冗余存储。

---

### 7.4 device_info 表字段最终清单

基于「device_info 是 device_parameters 的物化视图」原则，device_info 仅保留列表页必需的快查列。

#### 现有列清单（保持不变）

```sql
-- 手动填写（5 列）
device_name       VARCHAR(128),    -- 设备名称（用户自定义）
address           VARCHAR(256),    -- 部署地址
remark            TEXT,            -- 备注
project_status    VARCHAR(20),     -- 工程状态
height            DOUBLE PRECISION,-- 天线高度

-- TR069 自动同步（12 列）
eci               VARCHAR(32),     -- E-UTRAN Cell ID
pci               INTEGER,         -- 物理小区标识
cell_id           VARCHAR(32),     -- 小区 ID
freq_point        INTEGER,         -- 频点
bandwidth         VARCHAR(10),     -- 带宽
transmit_power    INTEGER,         -- 发射功率
plmn              VARCHAR(20),     -- PLMN 列表
rf_status         VARCHAR(20),     -- 射频状态
cell_status       VARCHAR(20),     -- 小区状态（三维综合判定）
mme_status        VARCHAR(20),     -- MME 连接状态（计算值）
sync_status       VARCHAR(20),     -- 同步状态（计算值）
mac               VARCHAR(20),     -- MAC 地址
hardware_version  VARCHAR(64),     -- 硬件版本

-- 时间（3 列）
first_online_time TIMESTAMPTZ,     -- 首次上线
last_offline_time TIMESTAMPTZ,     -- 最后离线
run_time          VARCHAR(64),     -- 累计运行时长

-- 审计（4 列）
creator           VARCHAR(64),     -- 创建人
updater           VARCHAR(64),     -- 更新人
created_at        TIMESTAMPTZ,     -- 创建时间
updated_at        TIMESTAMPTZ,     -- 更新时间
```

#### 本次新增列（仅 4 列）

| 新增列 | 类型 | 来源 | 理由 |
|--------|------|------|------|
| `num_of_cells` | INTEGER DEFAULT 1 | `CA.PARAMS.NumOfCells` | 载波模式判断基础（SC/CA/DC/TC），列表需展示/过滤 |
| `gps_status` | VARCHAR(20) | `X_COM_GPS_Status` | GPS 状态是基站运维核心指标，列表需展示/过滤 |
| `alarm_severity` | VARCHAR(20) | `X_RADISYS_COM_AlarmStatus` | 设备最高告警级别，列表红/橙/黄视觉指示 |
| `license_status` | VARCHAR(20) | 从 License 参数计算 | 许可证整体状态（active/expiring/expired），列表过滤 |

**不新增的字段及理由**：

| 曾考虑的字段 | 不新增的理由 |
|-------------|------------|
| `mme_pool` JSONB | 冗余 device_parameters，丢失原始路径，详情页前缀查询即可 |
| `antenna_info` JSONB | 天线参数仅详情页展示，不需列表过滤 |
| `license_summary` JSONB | 42 个参数双写一致性风险，`license_status` 计算值已够列表用 |
| `cell2_params` JSONB | 多实例参数，device_parameters 天然支持 `FAPService.2.%` 前缀查询 |
| `satellite_status` JSONB | 参与 `sync_status` 计算即可，不需独立存储 |
| `gps_satellite_count` | 仅诊断用，不需列表过滤 |
| `indoor_outdoor` | CBRS 相关，非通用列表需求 |
| `backup_version` | 仅详情页展示 |
| `eth_speed` | 仅诊断用 |
| `dn_prefix` | 仅详情页参考 |
| `license_expire_days` | 可在 `license_status` 中体现（expiring = 30 天内），详细天数从 K-V 实时计算 |

#### 迁移 SQL

```sql
-- migrations/000063_extend_device_info.up.sql
ALTER TABLE device_info
    ADD COLUMN num_of_cells     INTEGER DEFAULT 1,
    ADD COLUMN gps_status       VARCHAR(20),
    ADD COLUMN alarm_severity   VARCHAR(20),
    ADD COLUMN license_status   VARCHAR(20);

CREATE INDEX idx_device_info_gps_status ON device_info (gps_status);
CREATE INDEX idx_device_info_alarm_severity ON device_info (alarm_severity);
CREATE INDEX idx_device_info_license_status ON device_info (license_status);
```

**总计 device_info 列数**：5（手动）+ 12（现有自动）+ 4（本次新增）+ 3（时间）+ 4（审计）+ 1（PK）= **29 列**，精简可控。

---

### 7.5 参数同步机制增强

#### 7.5.1 同步架构：device_parameters 写入 → 快查列计算

所有 TR069 参数**首先写入 device_parameters**（Inform 自动 + ACS 查询），然后通过**计算逻辑**将少量快查字段同步到 device_info。

```
CPE → Inform → device_parameters (全量 K-V 写入)
                    ↓
              InfoSyncer.SyncQuickColumns()
                    ↓
              device_info (仅更新快查列的计算值)
```

#### 7.5.2 Inform 快查列同步增强

当前 `InfoSyncer.SyncFromParameters()` 仅同步 Carrier 适配器映射中的参数。

**建议增强**：新增通用快查列提取，不依赖 Carrier 映射：

```go
// 通用 Inform 参数 → device_info 快查列（所有运营商通用）
var informDirectMapping = map[string]string{
    "Device.DeviceInfo.X_COM_GPS_Status":                                  "gps_status",
    "Device.DeviceInfo.X_COM_STATION_RUN_Time":                            "run_time",
    "Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus":   "alarm_severity",
}
```

#### 7.5.3 ACS 查询频率分级

当前 ACS 在 Inform 后发起多组 GetParameterValues，结果写入 device_parameters。建议按查询频率分级：

| 频率 | 触发时机 | 参数组 | 写入目标 |
|------|---------|--------|---------|
| **每次 Inform** | 心跳时 | GPS 状态、同步源/同步状态、RadioEnable | device_parameters → 快查列计算 |
| **每日一次** | 定时任务 | License、MME 池、天线参数、ETH 速率 | device_parameters → 快查列计算 |
| **首次/变更时** | Bootstrap / ValueChange | 模块类型、经纬度/高度、NumOfCells | device_parameters → 快查列 |

#### 7.5.4 快查列计算逻辑

快查列的值不是直接映射单个 TR069 参数，而是从 device_parameters 中的多个参数**计算**得出。

**mme_status 计算**（从 device_parameters 中 MME 前缀查询）：

```go
func calcMMEStatus(params map[string]string) string {
    activeCount := 0
    for i := 1; i <= 16; i++ {
        prefix := fmt.Sprintf("...MmePoolConfigParam.%d.", i)
        if params[prefix+"MME1Status"] == "1" { activeCount++ }
    }
    switch {
    case activeCount == 0: return "disconnected"
    case activeCount < 2:  return "partial"
    default:               return "connected"
    }
}
```

**license_status 计算**（从 device_parameters 中 License 前缀查询）：

```go
func calcLicenseStatus(params map[string]string) string {
    hasActive, minRemain := false, math.MaxInt32
    for i := 1; i <= 32; i++ {
        prefix := fmt.Sprintf("...X_COM_LICENSE.Capacity.%d.", i)
        if params[prefix+"State"] == "1" {
            hasActive = true
            remain, _ := strconv.Atoi(params[prefix+"RemainingPeriod"])
            if remain < minRemain { minRemain = remain }
        }
    }
    switch {
    case !hasActive:      return "expired"
    case minRemain <= 30: return "expiring"
    default:              return "active"
    }
}
```

**cell_status 三维判定**（当前仅从单参数取值，应综合判定）：

```
FAP_adminstate = false                                          → "未激活"
FAP_adminstate = true && OpState = false                        → "故障"
FAP_adminstate = true && OpState = true && CellOpState = 0      → "退服"
FAP_adminstate = true && OpState = true && CellOpState = 1      → "正常"
```

#### 7.5.5 设备详情页数据组装

设备详情页不从 device_info 读取复合数据，而是从 device_parameters 按前缀查询后由 Service 层组装 DTO：

| 详情页 Tab | 查询前缀 | 组装逻辑 |
|-----------|---------|---------|
| MME 连接 | `...MmePoolConfigParam.%` | 过滤 Status=1 或 IP 非零的条目 |
| License | `...X_COM_LICENSE.%` | 按 Capacity.{i} 分组 |
| 天线参数 | `AntennaInfo.%` | 拍平为 AntennaDetail DTO |
| 小区 2/3 | `FAPService.2.%` / `FAPService.3.%` | 根据 num_of_cells 决定是否展示 |
| GPS/同步 | `X_COM_GPS_%` / `X_COM_BDS_%` / `X_COM_1588_%` | 汇总为同步状态视图 |

---

### 7.6 对已有 Gap 项的影响

| Gap 项 | 影响 |
|--------|------|
| **G03（参数同步）** | Carrier 适配器 `GetInfoParamMapping()` 需扩展新字段映射；新增通用 Inform 快查列计算 |
| **G12（设备详情聚合）** | License/MME/天线数据从 device_parameters 按前缀查询，Service 层组装 DTO，不需 JSONB 中间层 |
| **G02（列表过滤）** | DeviceFilter 需新增 gps_status、alarm_severity、license_status 过滤条件 |
| **G05（列自定义）** | 可用列集合扩大：新增 gps_status、alarm_severity、license_status、num_of_cells |
| **G04（导出）** | 导出快查列字段扩大；复合数据（MME/License）需从 device_parameters 查询后平铺 |

---

### 7.7 新增 Gap 项

基于 TR069 报文分析，补充以下差距项：

| # | 功能 | 优先级 | 复杂度 | 说明 |
|---|------|--------|--------|------|
| G28 | device_info 表扩展 — 新增 4 列 | **P1** | 低 | 迁移 000063：`num_of_cells`, `gps_status`, `alarm_severity`, `license_status` |
| G29 | Inform 通用快查列同步 | **P1** | 低 | 不依赖 Carrier 适配器的通用 Inform 参数 → 快查列映射（GPS 状态、告警级别） |
| G30 | mme_status 快查列计算逻辑 | **P1** | 中 | 从 device_parameters MME 前缀查询，统计有效 MME 数 → connected/partial/disconnected |
| G31 | license_status 快查列计算逻辑 | **P1** | 中 | 从 device_parameters License 前缀查询，计算整体状态 → active/expiring/expired |
| G32 | cell_status 三维判定逻辑 | **P1** | 低 | FAP_adminstate + OpState + CellOpState 综合判定，替代当前单参数取值 |
| G33 | ACS 查询频率分级策略 | **P2** | 中 | 按 §7.5.3 分级：每次 Inform / 每日 / 首次变更，降低不必要的 RPC 开销 |
| G34 | 设备详情页复合数据 DTO 组装 | **P2** | 中 | Service 层从 device_parameters 按前缀查询，组装 MME/License/天线/多小区 DTO |
| G35 | 多小区参数处理（CA/DC/TC） | **P2** | 中 | 根据 num_of_cells 决定详情页展示哪些小区 Tab，从 FAPService.{n}.% 前缀查询 |
| G36 | sync_status 综合计算 | **P2** | 中 | GPS + BDS + GLONASS + 1588 + tfcsSync 五源综合判定同步状态 |

---

## 6. 与设计文档的差异说明

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
| 外键 REFERENCES devices(id) | 无外键约束，应用层维护 1:1 | PG 分区表不支持被外键引用 |

# DD-09: 设备管理与拓扑（F06 核心部分）

> 关联功能域：F06（OMC-R 核心功能）
> 关联 backend-design.md 章节：第五章（PostgreSQL Schema）、第十三章（统一设备视图 13.3）
> 实施阶段：Phase 1（基础注册）→ Phase 2（状态管理）→ Phase 4（完整拓扑）
> 依赖文档：DD-02, DD-03, DD-04, DD-07

---

## 1. 概述

### 1.1 模块定位

设备管理（`internal/omcr/device/` 和 `internal/omcr/topology/`）是 OMC-R 核心功能的基础，负责设备全生命周期管理和网络拓扑维护。

### 1.2 核心职责

- 设备注册（从 Bootstrap Inform 自动创建）
- 设备状态机管理（7 种状态）
- 设备参数存储与查询
- 心跳监控与离线检测
- 拓扑管理（设备分组、层级树）
- 设备列表查询与过滤

---

## 2. 接口设计

### 2.1 DeviceRepository — `internal/omcr/device/repository.go`

```go
type DeviceRepository interface {
    Create(ctx context.Context, device *model.Device) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.Device, error)
    GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
    Update(ctx context.Context, device *model.Device) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filter DeviceFilter) (*model.ListResponse[model.Device], error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status model.DeviceStatus) error
    UpdateLastInform(ctx context.Context, sn string, at time.Time, events []string) error
    SetDataModelID(ctx context.Context, deviceID uuid.UUID, modelID uuid.UUID) error
    CountByStatus(ctx context.Context, carrier *model.CarrierCode) (map[model.DeviceStatus]int64, error)
}

type DeviceFilter struct {
    Carrier    *model.CarrierCode
    Technology *model.Technology
    Status     *model.DeviceStatus
    OUI        *string
    SiteName   *string
    Search     *string // 模糊搜索序列号/站名
    model.ListRequest
}
```

### 2.2 DeviceService — `internal/omcr/device/service.go`

```go
type DeviceService struct {
    repo       DeviceRepository
    paramRepo  DeviceParameterRepository
    eventBus   event.EventBus
    redis      redis.UniversalClient
    logger     *zap.Logger
}

// RegisterFromInform 从 Inform 注册新设备
func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*model.Device, error)

// UpdateFromInform 更新已有设备信息
func (s *DeviceService) UpdateFromInform(ctx context.Context, inform *tr069.InformMessage) error

// TransitionStatus 设备状态转移
func (s *DeviceService) TransitionStatus(ctx context.Context, deviceID uuid.UUID, newStatus model.DeviceStatus) error

// CheckHeartbeat 检查设备心跳，标记离线
func (s *DeviceService) CheckHeartbeat(ctx context.Context) error
```

### 2.3 DeviceParameterRepository — `internal/omcr/device/param_repository.go`

```go
type DeviceParameterRepository interface {
    BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
    GetByDevice(ctx context.Context, deviceID uuid.UUID, paths []string) ([]model.DeviceParameter, error)
    GetAll(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
    DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error
}
```

### 2.4 TopologyService — `internal/omcr/topology/service.go`

```go
type TopologyService struct {
    repo   TopologyRepository
    logger *zap.Logger
}

type DeviceGroup struct {
    ID        uuid.UUID     `json:"id"`
    Name      string        `json:"name"`
    ParentID  *uuid.UUID    `json:"parent_id,omitempty"`
    GroupType string        `json:"group_type"` // region, site, custom
    Children  []DeviceGroup `json:"children,omitempty"`
    Devices   []uuid.UUID   `json:"devices,omitempty"`
}

func (s *TopologyService) GetTree(ctx context.Context) ([]DeviceGroup, error)
func (s *TopologyService) CreateGroup(ctx context.Context, group *DeviceGroup) error
func (s *TopologyService) AssignDevice(ctx context.Context, deviceID, groupID uuid.UUID) error
```

### 2.5 REST API

```
GET    /api/v1/devices                   列表（分页、过滤）
GET    /api/v1/devices/{id}              详情
GET    /api/v1/devices/{id}/parameters   获取设备参数
POST   /api/v1/devices/{id}/parameters   设置设备参数（排队命令）
POST   /api/v1/devices/{id}/reboot       远程重启
POST   /api/v1/devices/{id}/factory-reset 恢复出厂
GET    /api/v1/topology/tree             设备层级树
GET    /api/v1/topology/groups           设备分组
```

---

## 3. 数据模型

### 3.1 数据库 Schema

```sql
-- 设备表（按运营商分区）
CREATE TABLE devices (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number        VARCHAR(64) NOT NULL UNIQUE,
    oui                  VARCHAR(6) NOT NULL,
    product_class        VARCHAR(64),
    manufacturer         VARCHAR(128),
    carrier              VARCHAR(4) NOT NULL,
    technology           VARCHAR(3) NOT NULL,
    data_model_id        UUID,
    status               VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version     VARCHAR(64),
    ip_address           INET,
    connection_request_url VARCHAR(256),
    last_inform_at       TIMESTAMPTZ,
    last_inform_events   JSONB,
    inform_interval      INTEGER DEFAULT 300,
    site_name            VARCHAR(128),
    site_id              VARCHAR(64),
    latitude             DOUBLE PRECISION,
    longitude            DOUBLE PRECISION,
    extension_data       JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY LIST (carrier);

CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');

-- 设备参数表
CREATE TABLE device_parameters (
    device_id        UUID NOT NULL REFERENCES devices(id),
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

-- 设备分组表
CREATE TABLE device_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,
    parent_id   UUID REFERENCES device_groups(id),
    group_type  VARCHAR(20) NOT NULL DEFAULT 'custom',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 设备-分组关联
CREATE TABLE device_group_members (
    device_id UUID NOT NULL REFERENCES devices(id),
    group_id  UUID NOT NULL REFERENCES device_groups(id),
    PRIMARY KEY (device_id, group_id)
);
```

### 3.2 Redis 数据结构

```
acs:heartbeat:{device_serial}   → String（最后 Inform 时间戳）
                                  TTL = 2 × inform_interval
```

---

## 4. 详细设计

### 4.1 设备注册流程（从 Bootstrap Inform）

```
ACS 收到 Bootstrap Inform
  │
  ├── 1. 提取 DeviceId（OUI, ProductClass, SerialNumber）
  │
  ├── 2. 查询 devices 表（按 serial_number）
  │     ├── 存在 → 更新设备信息 → UpdateFromInform()
  │     └── 不存在 → 创建新设备 → RegisterFromInform()
  │
  ├── 3. 确定运营商（carrier）：
  │     ├── 根据 ACS 配置（单运营商 ACS）
  │     ├── 根据 IP 段映射
  │     └── 根据认证信息
  │
  ├── 4. 设置初始状态 = "discovered"
  │
  ├── 5. 发布事件 device.inform.bootstrap
  │
  └── 6. 更新心跳 Redis key
```

### 4.2 设备状态机

```
discovered ──→ registered ──→ provisioning ──→ active
                                                │
                                    ┌───────────┤
                                    ↓           ↓
                              maintenance    offline
                                    │           │
                                    └───→ decommissioned
```

**有效转移**：

| 当前状态 | 可转移到 |
|---------|---------|
| discovered | registered |
| registered | provisioning |
| provisioning | active, registered（失败回退）|
| active | maintenance, offline |
| maintenance | active |
| offline | active（重新上线）|
| active/maintenance/offline | decommissioned |

### 4.3 心跳与离线检测

```
定时任务（每 60 秒）：
  1. 扫描所有 status=active 的设备
  2. 检查 Redis acs:heartbeat:{device_serial} 是否存在
  3. 不存在 → 设备心跳超时 → status 改为 offline
  4. 发布事件 device.connection.lost
```

---

## 5. 实施子阶段

### 阶段 9a：Device 模型 + Repository + 从 Inform 注册（Phase 1）

**交付物**：
- `device/repository.go` — PostgreSQL CRUD
- `device/service.go` — RegisterFromInform、UpdateFromInform
- 数据库迁移 001_create_devices

**验证**：ACS 收到 Bootstrap Inform 后 devices 表中出现新记录

### 阶段 9b：状态机 + 心跳检测（Phase 2）

**交付物**：
- 状态转移逻辑
- 心跳定时检查任务
- device.connection.lost 事件发布

**验证**：设备超时后自动标记为 offline

### 阶段 9c：拓扑管理 + 分组（Phase 4）

**交付物**：
- `topology/service.go`
- 设备分组 CRUD + 层级树查询

**验证**：创建分组、分配设备、查询层级树

---

## 6. 文件清单

```
internal/omcr/device/repository.go
internal/omcr/device/service.go
internal/omcr/device/param_repository.go
internal/omcr/topology/service.go
internal/omcr/topology/repository.go
```

---

## 7. 测试策略

- DeviceRepository：PostgreSQL 集成测试
- 设备注册：从 Inform 创建设备的端到端测试
- 状态机：有效/无效转移 table-driven 测试
- 心跳检测：Redis TTL 过期后设备离线

---

## 8. 参考

- backend-design.md 第五章：PostgreSQL Schema
- backend-design.md 第十三章：统一设备视图（13.3）
- doc/features/06-omc-core-functions.md：F06 OMC-R 核心功能

# 05 — 设备管理服务

> device-rpc 服务：设备生命周期、拓扑管理、自动开站、运营商适配

---

## 1. 服务概述

device-rpc 是 go-zero zRPC 服务，承载功能域最多：

| 功能域 | 能力 |
|--------|------|
| F06 OMC-R 核心 | 设备 CRUD、状态管理、拓扑树、设备组 |
| F07 网元直连 | 移动专有直连通道（通过适配器） |
| F09 自动开站 | 设备发现 → 模板匹配 → 配置下发 → 验证激活 |

---

## 2. Proto 接口定义

```protobuf
// api/proto/device.proto
syntax = "proto3";

package device;
option go_package = "omcgo/service/device/rpc/pb";

service DeviceService {
    // ========== 设备 CRUD ==========
    rpc RegisterDevice(RegisterDeviceReq) returns (DeviceResp);
    rpc GetDevice(GetDeviceReq) returns (DeviceResp);
    rpc ListDevices(ListDevicesReq) returns (ListDevicesResp);
    rpc UpdateDevice(UpdateDeviceReq) returns (DeviceResp);
    rpc DeleteDevice(DeleteDeviceReq) returns (DeleteDeviceResp);

    // ========== 设备状态 ==========
    rpc TransitionStatus(TransitionStatusReq) returns (TransitionStatusResp);
    rpc UpdateHeartbeat(UpdateHeartbeatReq) returns (UpdateHeartbeatResp);
    rpc CheckOfflineDevices(CheckOfflineReq) returns (CheckOfflineResp);

    // ========== 设备参数 ==========
    rpc GetDeviceParameters(GetDeviceParamsReq) returns (DeviceParamsResp);
    rpc SaveDeviceParameters(SaveDeviceParamsReq) returns (SaveDeviceParamsResp);

    // ========== 拓扑管理 ==========
    rpc GetTopologyTree(GetTopologyTreeReq) returns (TopologyTreeResp);
    rpc ListGroups(ListGroupsReq) returns (ListGroupsResp);
    rpc CreateGroup(CreateGroupReq) returns (GroupResp);
    rpc UpdateGroup(UpdateGroupReq) returns (GroupResp);
    rpc DeleteGroup(DeleteGroupReq) returns (DeleteGroupResp);
    rpc AssignDeviceToGroup(AssignDeviceReq) returns (AssignDeviceResp);
    rpc RemoveDeviceFromGroup(RemoveDeviceReq) returns (RemoveDeviceResp);

    // ========== 自动开站 ==========
    rpc StartProvisioning(StartProvisioningReq) returns (ProvisioningTaskResp);
    rpc GetProvisioningTask(GetProvisioningTaskReq) returns (ProvisioningTaskResp);
    rpc ListProvisioningTasks(ListProvisioningTasksReq) returns (ListProvisioningTasksResp);
    rpc RetryProvisioning(RetryProvisioningReq) returns (ProvisioningTaskResp);
}

// ========== 设备消息类型 ==========

message RegisterDeviceReq {
    string serial_number = 1;
    string oui = 2;
    string product_class = 3;
    string manufacturer = 4;
    string carrier = 5;
    string technology = 6;
    string firmware_version = 7;
    string hardware_version = 8;
    string ip_address = 9;
    string connection_request_url = 10;
}

message DeviceResp {
    int64 id = 1;
    string serial_number = 2;
    string oui = 3;
    string product_class = 4;
    string manufacturer = 5;
    string carrier = 6;
    string technology = 7;
    string status = 8;               // discovered/registered/provisioning/active/maintenance/offline/decommissioned
    string firmware_version = 9;
    string hardware_version = 10;
    string ip_address = 11;
    string connection_request_url = 12;
    string site_name = 13;
    int64 data_model_id = 14;
    int64 last_inform_at = 15;
    int64 created_at = 16;
    int64 updated_at = 17;
}

message ListDevicesReq {
    string carrier = 1;
    string technology = 2;
    string status = 3;
    string oui = 4;
    string site_name = 5;
    string search = 6;               // 模糊搜索（序列号、名称）
    int64 group_id = 7;
    int32 page = 8;
    int32 page_size = 9;
}

message ListDevicesResp {
    repeated DeviceResp devices = 1;
    int64 total = 2;
}

message UpdateDeviceReq {
    int64 id = 1;
    string site_name = 2;
    string carrier = 3;
    string technology = 4;
}

message DeleteDeviceReq {
    int64 id = 1;
}

message DeleteDeviceResp {
    bool success = 1;
}

// ========== 状态管理 ==========

message TransitionStatusReq {
    int64 device_id = 1;
    string target_status = 2;
    string reason = 3;
}

message TransitionStatusResp {
    bool success = 1;
    string previous_status = 2;
    string current_status = 3;
}

message UpdateHeartbeatReq {
    string serial_number = 1;
    string ip_address = 2;
    int64 timestamp = 3;
}

message UpdateHeartbeatResp {
    bool success = 1;
}

message CheckOfflineReq {
    int32 timeout_minutes = 1;       // 超时阈值（默认: 2 × inform_interval）
}

message CheckOfflineResp {
    int32 offline_count = 1;
    repeated string offline_serials = 2;
}

// ========== 设备参数 ==========

message GetDeviceParamsReq {
    int64 device_id = 1;
    repeated string parameter_paths = 2;  // 空=全部
}

message DeviceParamsResp {
    repeated ParameterValue parameters = 1;
}

message ParameterValue {
    string name = 1;
    string value = 2;
    string type = 3;
    bool writable = 4;
}

message SaveDeviceParamsReq {
    int64 device_id = 1;
    repeated ParameterValue parameters = 2;
}

message SaveDeviceParamsResp {
    int32 saved_count = 1;
}

// ========== 拓扑 ==========

message GetTopologyTreeReq {
    int64 root_group_id = 1;          // 0 = 从根开始
    int32 max_depth = 2;              // 最大深度（0=无限）
}

message TopologyTreeResp {
    repeated TreeNode nodes = 1;
}

message TreeNode {
    int64 group_id = 1;
    string name = 2;
    string type = 3;                  // region / site / custom
    int64 parent_id = 4;
    int32 device_count = 5;
    repeated TreeNode children = 6;
}

message ListGroupsReq {
    int64 parent_id = 1;
    string type = 2;
    int32 page = 3;
    int32 page_size = 4;
}

message ListGroupsResp {
    repeated GroupResp groups = 1;
    int64 total = 2;
}

message GroupResp {
    int64 id = 1;
    string name = 2;
    string type = 3;
    int64 parent_id = 4;
    string description = 5;
    int32 device_count = 6;
}

message CreateGroupReq {
    string name = 1;
    string type = 2;
    int64 parent_id = 3;
    string description = 4;
}

message UpdateGroupReq {
    int64 id = 1;
    string name = 2;
    string description = 3;
}

message DeleteGroupReq {
    int64 id = 1;
}

message DeleteGroupResp {
    bool success = 1;
}

message AssignDeviceReq {
    int64 group_id = 1;
    int64 device_id = 2;
}

message AssignDeviceResp {
    bool success = 1;
}

message RemoveDeviceReq {
    int64 group_id = 1;
    int64 device_id = 2;
}

message RemoveDeviceResp {
    bool success = 1;
}

// ========== 自动开站 ==========

message StartProvisioningReq {
    int64 device_id = 1;
    int64 template_id = 2;           // 0 = 自动匹配模板
}

message ProvisioningTaskResp {
    int64 task_id = 1;
    int64 device_id = 2;
    string device_serial = 3;
    string state = 4;                 // discovered/identifying/matching/configuring/verifying/completed/failed
    int64 template_id = 5;
    string error_message = 6;
    int32 retry_count = 7;
    int64 started_at = 8;
    int64 completed_at = 9;
}

message GetProvisioningTaskReq {
    int64 task_id = 1;
}

message ListProvisioningTasksReq {
    string state = 1;
    string carrier = 2;
    int32 page = 3;
    int32 page_size = 4;
}

message ListProvisioningTasksResp {
    repeated ProvisioningTaskResp tasks = 1;
    int64 total = 2;
}

message RetryProvisioningReq {
    int64 task_id = 1;
}
```

---

## 3. 设备生命周期状态机

### 3.1 状态定义

| 状态 | 含义 | 进入条件 |
|------|------|---------|
| `discovered` | 新发现设备 | ACS 收到 Bootstrap Inform |
| `registered` | 已注册设备 | 设备信息入库完成 |
| `provisioning` | 开站中 | 自动开站工作流启动 |
| `active` | 正常在线 | 开站验证通过 / 手动激活 |
| `maintenance` | 维护模式 | 管理员手动设置 |
| `offline` | 离线 | 心跳超时 |
| `decommissioned` | 已退网 | 管理员手动退网 |

### 3.2 合法状态转换

```
discovered ──→ registered ──→ provisioning ──→ active
                    │                            │ ↑
                    └──→ active ←────────────────┘ │
                           │                       │
                           ├──→ maintenance ───────┘
                           │
                           ├──→ offline ──→ active (恢复心跳)
                           │
                           └──→ decommissioned (终态)
```

### 3.3 状态转换矩阵

| 当前状态 → 目标状态 | discovered | registered | provisioning | active | maintenance | offline | decommissioned |
|:-------------------|:----------:|:----------:|:------------:|:------:|:-----------:|:-------:|:--------------:|
| discovered         | — | Y | — | — | — | — | — |
| registered         | — | — | Y | Y | — | — | — |
| provisioning       | — | — | — | Y | — | — | — |
| active             | — | — | — | — | Y | Y | Y |
| maintenance        | — | — | — | Y | — | — | Y |
| offline            | — | — | — | Y | — | — | Y |
| decommissioned     | — | — | — | — | — | — | — |

### 3.4 实现

```go
// common/model/device_status.go

var validTransitions = map[DeviceStatus][]DeviceStatus{
    StatusDiscovered:      {StatusRegistered},
    StatusRegistered:      {StatusProvisioning, StatusActive},
    StatusProvisioning:    {StatusActive, StatusRegistered}, // 失败可回退
    StatusActive:          {StatusMaintenance, StatusOffline, StatusDecommissioned},
    StatusMaintenance:     {StatusActive, StatusDecommissioned},
    StatusOffline:         {StatusActive, StatusDecommissioned},
}

func CanTransition(from, to DeviceStatus) bool {
    targets, ok := validTransitions[from]
    if !ok {
        return false
    }
    for _, t := range targets {
        if t == to {
            return true
        }
    }
    return false
}
```

---

## 4. 自动开站 Saga 工作流

### 4.1 工作流状态机

```
  Bootstrap Inform
       │
       ▼
  ┌──────────┐
  │discovered│ ← ACS 注册新设备
  └────┬─────┘
       │
       ▼
  ┌────────────┐
  │identifying │ ← 提取 OUI + ProductClass
  └────┬───────┘
       │
       ▼
  ┌──────────┐
  │ matching │ ← config-rpc.ResolveDataModel + config-rpc.MatchTemplate
  └────┬─────┘
       │
       ▼
  ┌─────────────┐
  │ configuring │ ← acs-rpc.QueueCommand (Get → Set → Download → Reboot)
  └────┬────────┘
       │
       ▼
  ┌───────────┐
  │ verifying │ ← acs-rpc.QueueCommand (GetParameterValues) + 比对
  └────┬──────┘
       │
       ├── 成功 ──→ completed (device.status = active)
       │
       └── 失败 ──→ failed (可重试)
```

### 4.2 跨服务编排（Saga 模式）

自动开站跨越 3 个服务，采用编排式 Saga：

```go
// service/device/rpc/internal/logic/provisioninglogic.go

func (l *ProvisioningLogic) StartProvisioning(in *pb.StartProvisioningReq) error {
    task := &ProvisioningTask{DeviceID: in.DeviceId, State: "identifying"}
    l.svcCtx.TaskRepo.Create(l.ctx, task)

    // Step 1: 识别设备
    device, err := l.svcCtx.DeviceRepo.GetByID(l.ctx, in.DeviceId)
    if err != nil {
        return l.failTask(task, "identify", err)
    }

    // Step 2: 匹配数据模型 → config-rpc
    task.State = "matching"
    l.svcCtx.TaskRepo.UpdateState(l.ctx, task)

    dataModel, err := l.svcCtx.ConfigRpc.ResolveDataModel(l.ctx, &config.ResolveDataModelReq{
        Carrier:      device.Carrier,
        Technology:   device.Technology,
        Oui:          device.OUI,
        ProductClass: device.ProductClass,
    })
    if err != nil {
        return l.failTask(task, "match_model", err)
    }

    // Step 3: 匹配配置模板 → config-rpc
    tmpl, err := l.svcCtx.ConfigRpc.MatchTemplate(l.ctx, &config.MatchTemplateReq{
        Carrier:      device.Carrier,
        Technology:   device.Technology,
        ProductClass: device.ProductClass,
    })
    if err != nil {
        return l.failTask(task, "match_template", err)
    }

    // Step 4: 下发配置 → acs-rpc
    task.State = "configuring"
    l.svcCtx.TaskRepo.UpdateState(l.ctx, task)

    commands := buildCommandsFromTemplate(tmpl)
    for _, cmd := range commands {
        _, err := l.svcCtx.AcsRpc.QueueCommand(l.ctx, &acs.QueueCommandReq{
            DeviceSerial: device.SerialNumber,
            CommandType:  cmd.Type,
            Priority:     cmd.Priority,
            Payload:      cmd.Payload,
            CommandKey:   fmt.Sprintf("provision_%d_%s", task.ID, cmd.Type),
        })
        if err != nil {
            return l.failTask(task, "configure", err)
        }
    }

    // Step 5: 验证（在设备下次 Inform 时通过事件触发）
    task.State = "verifying"
    l.svcCtx.TaskRepo.UpdateState(l.ctx, task)

    return nil
}
```

### 4.3 模板匹配优先级

| 优先级 | 匹配条件 | 示例 |
|:------:|---------|------|
| 1（最高） | carrier + tech + product_class | cmcc + lte + "HW-LTE-Pico-3000" |
| 2 | carrier + tech | cmcc + lte（默认模板） |
| 3（最低） | tech | lte（全局模板） |

---

## 5. 运营商适配器集成

### 5.1 Carrier 接口

```go
// common/carrier/carrier.go

type Carrier interface {
    // 运营商代码
    Code() CarrierCode

    // 参数路径映射（运营商特有参数名 → 标准参数名）
    NormalizeParameterPath(path string) string

    // 设备验证规则
    ValidateDevice(device *model.Device) error

    // KPI 计算公式（运营商可能有不同的 KPI 定义）
    GetKPIFormulas(tech Technology) []KPIFormula

    // Inform 事件处理差异
    HandleInformEvent(event *InformEvent) (*InformResult, error)

    // 开站流程差异
    GetProvisioningSteps(device *model.Device) []ProvisionStep

    // 北向接口格式差异
    FormatNorthboundData(data interface{}) ([]byte, error)
}
```

### 5.2 CarrierRegistry

```go
// common/carrier/registry.go

type CarrierRegistry struct {
    carriers map[CarrierCode]Carrier
}

func NewCarrierRegistry() *CarrierRegistry {
    r := &CarrierRegistry{carriers: make(map[CarrierCode]Carrier)}
    r.Register(cmcc.NewAdapter())
    r.Register(ctcc.NewAdapter())
    r.Register(cucc.NewAdapter())
    return r
}

func (r *CarrierRegistry) Get(code CarrierCode) (Carrier, error) {
    c, ok := r.carriers[code]
    if !ok {
        return nil, fmt.Errorf("unknown carrier: %s", code)
    }
    return c, nil
}
```

### 5.3 在 device-rpc 中使用

```go
// service/device/rpc/internal/svc/servicecontext.go

type ServiceContext struct {
    Config          config.Config
    DeviceModel     model.DeviceModel
    CarrierRegistry *carrier.CarrierRegistry
    ConfigRpc       configclient.ConfigService
    AcsRpc          acsclient.AcsControl
    // ...
}

// Logic 层使用
func (l *RegisterDeviceLogic) RegisterDevice(in *pb.RegisterDeviceReq) (*pb.DeviceResp, error) {
    // 获取运营商适配器
    carrierAdapter, err := l.svcCtx.CarrierRegistry.Get(carrier.CarrierCode(in.Carrier))
    if err != nil {
        return nil, err
    }

    device := &model.Device{ /* ... */ }

    // 运营商特有的设备验证
    if err := carrierAdapter.ValidateDevice(device); err != nil {
        return nil, fmt.Errorf("carrier validation: %w", err)
    }

    // ...
}
```

---

## 6. 数据库 Schema

### 6.1 devices（按运营商分区）

```sql
CREATE TABLE devices (
    id                      BIGSERIAL,
    serial_number           VARCHAR(64)  NOT NULL,
    oui                     VARCHAR(10)  NOT NULL,
    product_class           VARCHAR(100),
    manufacturer            VARCHAR(200),
    carrier                 VARCHAR(10)  NOT NULL,   -- 分区键
    technology              VARCHAR(10)  NOT NULL,
    status                  VARCHAR(20)  NOT NULL DEFAULT 'discovered',
    firmware_version        VARCHAR(100),
    hardware_version        VARCHAR(100),
    ip_address              INET,
    connection_request_url  TEXT,
    site_name               VARCHAR(200),
    data_model_id           BIGINT REFERENCES data_model_definitions(id),
    last_inform_at          TIMESTAMPTZ,
    extension_data          JSONB,        -- 运营商扩展字段
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, carrier)
) PARTITION BY LIST (carrier);

CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');

CREATE UNIQUE INDEX idx_devices_serial ON devices (serial_number);
CREATE INDEX idx_devices_carrier_status ON devices (carrier, status);
CREATE INDEX idx_devices_last_inform ON devices (last_inform_at);
CREATE INDEX idx_devices_site ON devices (site_name);
```

### 6.2 provisioning_tasks

```sql
CREATE TABLE provisioning_tasks (
    id              BIGSERIAL PRIMARY KEY,
    device_id       BIGINT       NOT NULL,
    device_serial   VARCHAR(64)  NOT NULL,
    state           VARCHAR(20)  NOT NULL DEFAULT 'discovered',
    template_id     BIGINT,
    error_message   TEXT,
    retry_count     INT          NOT NULL DEFAULT 0,
    command_log     JSONB,       -- 已执行命令列表
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pt_device ON provisioning_tasks (device_id);
CREATE INDEX idx_pt_state ON provisioning_tasks (state);
```

---

## 7. 心跳与离线检测

```
ACS Periodic Inform → device-rpc.UpdateHeartbeat
  │
  ├── 更新 Redis: SET acs:heartbeat:{serial} {timestamp} EX {2 × inform_interval}
  │
  └── 更新 PostgreSQL: UPDATE devices SET last_inform_at = NOW()

Worker 定时任务（每分钟）:
  │
  ├── 扫描 Redis: 查找过期的 heartbeat key
  │
  ├── 对过期设备: device-rpc.TransitionStatus → offline
  │
  └── 发布事件: NATS device.inform.connection_lost
```

# TR069 参数处理实现方案

> 基于 `files/Back-end/TR069报文全解析.md` 的功能需求分析与实现方案。

---

## 1. 文档概述

### 1.1 来源文档分析

TR069 报文全解析文档共包含 **252 个参数**，分布在：

| 报文类型 | 参数组 | 参数数量 |
|---------|--------|---------|
| Inform（首次） | 1 组 | 37 个 |
| GetParameterValuesResponse | 19 组 | 215 个 |

### 1.2 参数分类

| 分类 | 数量 | 存储位置 |
|------|------|---------|
| **主表字段**（标记 ✅） | 14 个 | `devices` 表 |
| **MME 池配置** | 48 个 | `devices.mme_pool` (JSONB) |
| **许可证信息** | 42 个 | `device_licenses` 表（新建） |
| **当前告警** | 30 个 | `alarms` 表（已有） |
| **普通参数** | 118 个 | `device_parameters` 表（已有） |

---

## 2. 当前实现差距分析

### 2.1 Device 模型对比

#### 现有字段

```go
// omcgo/internal/core/model/device.go
type Device struct {
    ID                   uuid.UUID
    SerialNumber         string
    OUI                  string
    ProductClass         string
    Manufacturer         string
    ModelName            string
    Carrier              CarrierCode
    Technology           Technology
    DataModelID          *uuid.UUID
    Status               DeviceStatus
    FirmwareVersion      string  // ✅ 已有（对应 software_version）
    IPAddress            string  // ✅ 已有
    ConnectionRequestURL string  // ✅ 已有
    LastInformAt         *time.Time
    LastInformEvents     []string
    InformInterval       int
    SiteName             string
    SiteID               string
    Latitude             float64   // ✅ 已有
    Longitude            float64   // ✅ 已有
    ExtensionData        map[string]interface{}
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

#### 需要新增的字段

| 字段名 | 类型 | 来源参数 | 来源报文 | 说明 |
|--------|------|---------|---------|------|
| `HardwareVersion` | string | `Device.DeviceInfo.HardwareVersion` | Inform | 硬件版本 |
| `RunTime` | string | `Device.DeviceInfo.X_COM_STATION_RUN_Time` | Inform | 运行时间（如 "40d 4h 58m 29s"） |
| `PLMN` | string | `Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList` | Inform | PLMN 列表（逗号分隔） |
| `CellStatus` | string | `Device.Services.FAPService.1.FAPControl.LTE.OpState` | Inform | 小区运行状态 |
| `RFStatus` | bool | `Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus` | Inform | 射频发射状态 |
| `Height` | float64 | `Device.DeviceInfo.AntennaInfo.Height` | GPVResponse | 天线高度 |
| `SyncSource` | int | `Device.ManagementServer.tfcsManagerPrimsrc` | GPVResponse | 同步主源（0-7） |
| `SyncState` | string | `Device.ManagementServer.tfcsSyncState` | GPVResponse | 同步状态（如 "DISP"） |
| `MMEPool` | json.RawMessage | `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.*` | GPVResponse | MME 池配置（JSON 数组） |

### 2.2 参数提取逻辑差距

#### Inform 参数提取（当前）

```go
// omcgo/internal/device/service.go
// 当前只提取：
device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
device.ConnectionRequestURL = findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
```

#### 需要新增的 Inform 参数提取

```go
// 需要从 Inform ParameterList 提取：
device.HardwareVersion = findParamValue(params, "Device.DeviceInfo.HardwareVersion")
device.RunTime = findParamValue(params, "Device.DeviceInfo.X_COM_STATION_RUN_Time")
device.PLMN = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList")
device.CellStatus = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.OpState") == "true"
device.RFStatus = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus") == "true"
device.IPAddress = findParamValue(params, "Device.IP.Interface.1.IPv4Address.1.IPAddress") // 替代现有逻辑
```

#### GPVResponse 参数提取（新增）

当前系统**未实现** GetParameterValuesResponse 的参数提取。需要新增：

1. **模块/型号查询**（第三组）
   ```go
   device.ModelName = findParamValue(params, "Device.DeviceInfo.X_COM_MODULE_TYPE")
   ```

2. **经纬度查询**（第五组）
   ```go
   device.Latitude = parseFloat(findParamValue(params, "Device.FAP.GPS.LockedLatitude"))
   device.Longitude = parseFloat(findParamValue(params, "Device.FAP.GPS.LockedLongitude"))
   ```

3. **天线高度查询**（第六组）
   ```go
   device.Height = parseFloat(findParamValue(params, "Device.DeviceInfo.AntennaInfo.Height"))
   ```

4. **同步源查询**（第十九组）
   ```go
   device.SyncSource = parseInt(findParamValue(params, "Device.ManagementServer.tfcsManagerPrimsrc"))
   device.SyncState = findParamValue(params, "Device.ManagementServer.tfcsSyncState")
   ```

5. **MME 池配置查询**（第十四组）
   ```go
   // 需要解析 16 个 MME 配置，生成 JSON 数组
   mmePool := parseMMEPoolConfig(params)
   device.MMEPool = mmePool
   ```

### 2.3 多小区配置差距

文档明确指出：

> 上述关于小区2，小区3的参数是否保存，这个是根据基站的载波模式处理的，比如 CA/SC 只处理主小区，如果是 DC 处理小区1和小区2，TC模式处理小区123。

当前实现**未支持**多小区配置。需要：

1. 新增载波模式字段（CA/SC/DC/TC）
2. 根据载波模式决定保存哪些小区参数
3. 设计多小区数据存储结构

---

## 3. 实现方案

### 3.1 数据库变更

#### 3.1.1 devices 表新增字段

```sql
-- Migration: 000036_add_device_tr069_fields.up.sql

ALTER TABLE devices
    ADD COLUMN hardware_version VARCHAR(64),
    ADD COLUMN run_time VARCHAR(32),
    ADD COLUMN plmn VARCHAR(128),
    ADD COLUMN cell_status BOOLEAN,
    ADD COLUMN rf_status BOOLEAN,
    ADD COLUMN height DOUBLE PRECISION,
    ADD COLUMN sync_source INTEGER,
    ADD COLUMN sync_state VARCHAR(16),
    ADD COLUMN mme_pool JSONB,
    ADD COLUMN carrier_mode VARCHAR(8) DEFAULT 'SC'; -- CA/SC/DC/TC

COMMENT ON COLUMN devices.hardware_version IS '硬件版本，来自 Inform';
COMMENT ON COLUMN devices.run_time IS '设备运行时间，格式如 "40d 4h 58m 29s"';
COMMENT ON COLUMN devices.plmn IS 'PLMN 列表，逗号分隔';
COMMENT ON COLUMN devices.cell_status IS '小区运行状态';
COMMENT ON COLUMN devices.rf_status IS '射频发射状态';
COMMENT ON COLUMN devices.height IS '天线高度（米）';
COMMENT ON COLUMN devices.sync_source IS '同步主源（0-7）';
COMMENT ON COLUMN devices.sync_state IS '同步状态（DISP 等）';
COMMENT ON COLUMN devices.mme_pool IS 'MME 池配置 JSON 数组';
COMMENT ON COLUMN devices.carrier_mode IS '载波模式：CA/SC/DC/TC';
```

#### 3.1.2 新建设备许可证表

```sql
-- Migration: 000037_create_device_licenses.up.sql

CREATE TABLE device_licenses (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    license_code    VARCHAR(16) NOT NULL,        -- FAP
    version         INTEGER NOT NULL,
    author          VARCHAR(64),
    generate_date   VARCHAR(8),                  -- YYYYMMDD
    seq_num         INTEGER,
    max_capacity    INTEGER DEFAULT 32,
    created_at      TIMESTAMTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, device_id)
);

-- 许可证容量项
CREATE TABLE device_license_capacities (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    license_id      UUID NOT NULL,
    capacity_id     VARCHAR(16) NOT NULL,        -- FAP044, FAP025, etc.
    description     TEXT,
    state           INTEGER NOT NULL,            -- 0=未启用, 1=有效
    valid_period    INTEGER,                     -- 有效期限（天）
    remaining_days  INTEGER,                     -- 剩余天数
    value           VARCHAR(64),
    delay_available INTEGER,
    delay_control   INTEGER,
    PRIMARY KEY (id, license_id)
);

CREATE INDEX idx_device_licenses_device ON device_licenses (device_id);
CREATE INDEX idx_license_capacities_license ON device_license_capacities (license_id);
```

#### 3.1.3 新增多小区配置表

```sql
-- Migration: 000038_create_device_cells.up.sql

CREATE TABLE device_cells (
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    device_id           UUID NOT NULL,
    cell_index          INTEGER NOT NULL,             -- 1, 2, 3
    cell_op_state       INTEGER,                      -- 小区运行状态
    plmn                VARCHAR(128),                 -- PLMN 列表
    op_state            BOOLEAN,                      -- LTE 运行状态
    rf_tx_status        BOOLEAN,                      -- 射频发射状态
    slot                INTEGER,                      -- 槽位
    epc_enable_state    INTEGER,                      -- 嵌入式 EPC 启用状态
    epc_mode            INTEGER,                      -- 嵌入式 EPC 模式
    alarm_status        VARCHAR(32),                  -- 告警状态
    created_at          TIMESTAMTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, device_id, cell_index)
);

CREATE INDEX idx_device_cells_device ON device_cells (device_id);
CREATE UNIQUE INDEX idx_device_cells_device_index ON device_cells (device_id, cell_index);
```

### 3.2 模型变更

#### 3.2.1 Device 模型扩展

```go
// omcgo/internal/core/model/device.go

type Device struct {
    // ... 现有字段 ...

    // 新增 TR069 参数字段
    HardwareVersion string          `json:"hardware_version" db:"hardware_version"`
    RunTime         string          `json:"run_time" db:"run_time"`
    PLMN            string          `json:"plmn" db:"plmn"`
    CellStatus      bool            `json:"cell_status" db:"cell_status"`
    RFStatus        bool            `json:"rf_status" db:"rf_status"`
    Height          float64         `json:"height" db:"height"`
    SyncSource      int             `json:"sync_source" db:"sync_source"`
    SyncState       string          `json:"sync_state" db:"sync_state"`
    MMEPool         json.RawMessage `json:"mme_pool" db:"mme_pool"`
    CarrierMode     string          `json:"carrier_mode" db:"carrier_mode"`
}

// MMEPoolConfig represents a single MME pool entry.
type MMEPoolConfig struct {
    MMEStatus int    `json:"mme_status"` // 0=禁用, 1=启用
    MMEIP     string `json:"mme_ip"`
    PLMNID    string `json:"plmn_id"`
}

// DeviceCell represents a single cell configuration.
type DeviceCell struct {
    ID             uuid.UUID `json:"id" db:"id"`
    DeviceID       uuid.UUID `json:"device_id" db:"device_id"`
    CellIndex      int       `json:"cell_index" db:"cell_index"`
    CellOpState    int       `json:"cell_op_state" db:"cell_op_state"`
    PLMN           string    `json:"plmn" db:"plmn"`
    OpState        bool      `json:"op_state" db:"op_state"`
    RFTxStatus     bool      `json:"rf_tx_status" db:"rf_tx_status"`
    Slot           int       `json:"slot" db:"slot"`
    EPCEnableState int       `json:"epc_enable_state" db:"epc_enable_state"`
    EPCMode        int       `json:"epc_mode" db:"epc_mode"`
    AlarmStatus    string    `json:"alarm_status" db:"alarm_status"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
```

### 3.3 代码变更

#### 3.3.1 参数提取器（新增）

```go
// omcgo/internal/device/param_extractor.go

package device

import (
    "encoding/json"
    "strconv"
    "strings"

    "github.com/omcgo/omcgo/internal/core/model"
    "github.com/omcgo/omcgo/pkg/tr069"
)

// InformParamExtractor extracts device fields from Inform ParameterList.
type InformParamExtractor struct{}

func NewInformParamExtractor() *InformParamExtractor {
    return &InformParamExtractor{}
}

// ExtractToDevice extracts Inform parameters into Device model.
func (e *InformParamExtractor) ExtractToDevice(params []tr069.ParameterValueStruct, device *model.Device) {
    // 硬件版本
    if v := findParamValue(params, "Device.DeviceInfo.HardwareVersion"); v != "" {
        device.HardwareVersion = v
    }

    // 运行时间
    if v := findParamValue(params, "Device.DeviceInfo.X_COM_STATION_RUN_Time"); v != "" {
        device.RunTime = v
    }

    // PLMN 列表
    if v := findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList"); v != "" {
        device.PLMN = v
    }

    // 小区状态
    if v := findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.OpState"); v != "" {
        device.CellStatus = strings.ToLower(v) == "true"
    }

    // 射频状态
    if v := findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"); v != "" {
        device.RFStatus = strings.ToLower(v) == "true"
    }

    // IP 地址（优先使用 Inform 中的）
    if v := findParamValue(params, "Device.IP.Interface.1.IPv4Address.1.IPAddress"); v != "" {
        device.IPAddress = v
    }

    // 软件版本
    if v := findParamValue(params, "Device.DeviceInfo.SoftwareVersion"); v != "" {
        device.FirmwareVersion = v
    }
}

// GPVParamExtractor extracts device fields from GetParameterValuesResponse.
type GPVParamExtractor struct{}

func NewGPVParamExtractor() *GPVParamExtractor {
    return &GPVParamExtractor{}
}

// ExtractToDevice extracts GPVResponse parameters into Device model.
func (e *GPVParamExtractor) ExtractToDevice(params []tr069.ParameterValueStruct, device *model.Device) {
    // 型号名称
    if v := findParamValue(params, "Device.DeviceInfo.X_COM_MODULE_TYPE"); v != "" {
        device.ModelName = v
    }

    // 经纬度
    if v := findParamValue(params, "Device.FAP.GPS.LockedLatitude"); v != "" {
        if lat, err := strconv.ParseFloat(v, 64); err == nil {
            device.Latitude = lat
        }
    }
    if v := findParamValue(params, "Device.FAP.GPS.LockedLongitude"); v != "" {
        if lng, err := strconv.ParseFloat(v, 64); err == nil {
            device.Longitude = lng
        }
    }

    // 天线高度
    if v := findParamValue(params, "Device.DeviceInfo.AntennaInfo.Height"); v != "" {
        if h, err := strconv.ParseFloat(v, 64); err == nil {
            device.Height = h
        }
    }

    // 同步源
    if v := findParamValue(params, "Device.ManagementServer.tfcsManagerPrimsrc"); v != "" {
        if src, err := strconv.Atoi(v); err == nil {
            device.SyncSource = src
        }
    }
    if v := findParamValue(params, "Device.ManagementServer.tfcsSyncState"); v != "" {
        device.SyncState = v
    }

    // MME 池配置
    mmePool := e.extractMMEPool(params)
    if len(mmePool) > 0 {
        data, _ := json.Marshal(mmePool)
        device.MMEPool = data
    }
}

// extractMMEPool extracts MME pool configuration from parameters.
func (e *GPVParamExtractor) extractMMEPool(params []tr069.ParameterValueStruct) []model.MMEPoolConfig {
    var pool []model.MMEPoolConfig

    // MME 配置参数路径前缀
    prefix := "Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam."

    // 构建索引映射
    mmeMap := make(map[int]*model.MMEPoolConfig)

    for _, p := range params {
        if !strings.HasPrefix(p.Name, prefix) {
            continue
        }

        // 解析参数路径: ...MmePoolConfigParam.{index}.{field}
        parts := strings.Split(strings.TrimPrefix(p.Name, prefix), ".")
        if len(parts) != 2 {
            continue
        }

        index, err := strconv.Atoi(parts[0])
        if err != nil {
            continue
        }

        if mmeMap[index] == nil {
            mmeMap[index] = &model.MMEPoolConfig{}
        }

        field := parts[1]
        switch field {
        case "MME1Status":
            if v, err := strconv.Atoi(p.Value); err == nil {
                mmeMap[index].MMEStatus = v
            }
        case "MMEIp1":
            mmeMap[index].MMEIP = p.Value
        case "PLMNID":
            mmeMap[index].PLMNID = p.Value
        }
    }

    // 按索引排序输出
    for i := 1; i <= 16; i++ {
        if mmeMap[i] != nil {
            pool = append(pool, *mmeMap[i])
        }
    }

    return pool
}
```

#### 3.3.2 多小区参数提取

```go
// omcgo/internal/device/cell_extractor.go

package device

import (
    "strconv"
    "strings"

    "github.com/google/uuid"
    "github.com/omcgo/omcgo/internal/core/model"
    "github.com/omcgo/omcgo/pkg/tr069"
)

// CellExtractor extracts multi-cell configuration from Inform parameters.
type CellExtractor struct{}

func NewCellExtractor() *CellExtractor {
    return &CellExtractor{}
}

// ExtractCells extracts cell configurations based on carrier mode.
// carrierMode: CA/SC (only cell 1), DC (cells 1-2), TC (cells 1-3)
func (e *CellExtractor) ExtractCells(params []tr069.ParameterValueStruct, deviceID uuid.UUID, carrierMode string) []model.DeviceCell {
    maxCells := e.getMaxCells(carrierMode)
    var cells []model.DeviceCell

    for i := 1; i <= maxCells; i++ {
        cell := e.extractCell(params, deviceID, i)
        if cell != nil {
            cells = append(cells, *cell)
        }
    }

    return cells
}

func (e *CellExtractor) getMaxCells(carrierMode string) int {
    switch strings.ToUpper(carrierMode) {
    case "CA", "SC":
        return 1
    case "DC":
        return 2
    case "TC":
        return 3
    default:
        return 1
    }
}

func (e *CellExtractor) extractCell(params []tr069.ParameterValueStruct, deviceID uuid.UUID, index int) *model.DeviceCell {
    prefix := "Device.Services.FAPService." + strconv.Itoa(index) + ".FAPControl.LTE."

    cell := &model.DeviceCell{
        DeviceID:  deviceID,
        CellIndex: index,
    }

    found := false

    for _, p := range params {
        if !strings.HasPrefix(p.Name, prefix) {
            continue
        }

        found = true
        field := strings.TrimPrefix(p.Name, prefix)

        switch field {
        case "CellOpState":
            if v, err := strconv.Atoi(p.Value); err == nil {
                cell.CellOpState = v
            }
        case "Gateway.ExistPlmnidList":
            cell.PLMN = p.Value
        case "OpState":
            cell.OpState = strings.ToLower(p.Value) == "true"
        case "RFTxStatus":
            cell.RFTxStatus = strings.ToLower(p.Value) == "true"
        case "Slot":
            if v, err := strconv.Atoi(p.Value); err == nil {
                cell.Slot = v
            }
        case "X_COM_HSS.EMBEDDED_EPCEnableState":
            if v, err := strconv.Atoi(p.Value); err == nil {
                cell.EPCEnableState = v
            }
        case "X_COM_HSS.EMBEDDED_EPCMode":
            if v, err := strconv.Atoi(p.Value); err == nil {
                cell.EPCMode = v
            }
        case "X_RADISYS_COM_AlarmStatus":
            cell.AlarmStatus = p.Value
        }
    }

    if !found {
        return nil
    }

    return cell
}
```

#### 3.3.3 Service 层集成

```go
// omcgo/internal/device/service.go (修改)

// 在 RegisterFromInform 和 UpdateFromInform 中添加：

func (s *DeviceService) RegisterFromInform(ctx context.Context, inform *tr069.InformMessage, carrier model.CarrierCode) (*model.Device, error) {
    // ... 现有逻辑 ...

    // 新增：使用参数提取器
    informExtractor := NewInformParamExtractor()
    informExtractor.ExtractToDevice(inform.ParameterList, device)

    // 新增：提取多小区配置（默认 SC 模式）
    cellExtractor := NewCellExtractor()
    cells := cellExtractor.ExtractCells(inform.ParameterList, device.ID, "SC")

    // ... 保存设备 ...

    // 保存小区配置
    if len(cells) > 0 {
        if err := s.cellRepo.BatchUpsert(ctx, cells); err != nil {
            s.logger.Error("save cell configs", zap.Error(err))
        }
    }

    // ... 其余逻辑 ...
}
```

### 3.4 GPVResponse 事件处理（新增）

需要新增事件处理器来处理 GetParameterValuesResponse：

```go
// omcgo/internal/device/gpv_handler.go

package device

import (
    "context"
    "encoding/json"

    "github.com/omcgo/omcgo/internal/core/event"
    "github.com/omcgo/omcgo/pkg/tr069"
    "go.uber.org/zap"
)

// GPVResponsePayload represents the payload of a GPV response event.
type GPVResponsePayload struct {
    DeviceSN     string                       `json:"device_sn"`
    Method       string                       `json:"method"`       // 查询组标识
    ParameterList []tr069.ParameterValueStruct `json:"parameter_list"`
}

// GPVHandler handles GetParameterValuesResponse events.
type GPVHandler struct {
    service    *DeviceService
    extractor  *GPVParamExtractor
    logger     *zap.Logger
}

func NewGPVHandler(service *DeviceService, logger *zap.Logger) *GPVHandler {
    return &GPVHandler{
        service:   service,
        extractor: NewGPVParamExtractor(),
        logger:    logger,
    }
}

func (h *GPVHandler) Subscribe(bus event.EventBus) error {
    // 订阅参数查询响应事件
    if _, err := bus.QueueSubscribe(event.SubjectCommandGetParamsResponse, "device-manager", h.handleGetParamsResponse); err != nil {
        return err
    }
    return nil
}

func (h *GPVHandler) handleGetParamsResponse(ctx context.Context, evt event.Event) error {
    var payload GPVResponsePayload
    if err := evt.DecodePayload(&payload); err != nil {
        return err
    }

    device, err := h.service.GetBySerialNumber(ctx, payload.DeviceSN)
    if err != nil || device == nil {
        return err
    }

    // 提取参数到设备模型
    h.extractor.ExtractToDevice(payload.ParameterList, device)

    // 更新设备
    if err := h.service.UpdateDeviceFields(ctx, device); err != nil {
        h.logger.Error("update device from GPV response",
            zap.Error(err),
            zap.String("serial_number", payload.DeviceSN))
        return err
    }

    h.logger.Info("device updated from GPV response",
        zap.String("serial_number", payload.DeviceSN))

    return nil
}
```

---

## 4. 实施计划

### 4.1 阶段一：数据库 Schema 变更

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 创建迁移文件 | `000036_add_device_tr069_fields.up/down.sql` |
| 2 | 创建许可证表 | `000037_create_device_licenses.up/down.sql` |
| 3 | 创建多小区表 | `000038_create_device_cells.up/down.sql` |
| 4 | 更新 Device 模型 | `omcgo/internal/core/model/device.go` |
| 5 | 新增 MMEPool/DeviceCell 模型 | `omcgo/internal/core/model/device.go` |

### 4.2 阶段二：参数提取逻辑

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | Inform 参数提取器 | `omcgo/internal/device/param_extractor.go` |
| 2 | GPVResponse 参数提取器 | `omcgo/internal/device/param_extractor.go` |
| 3 | 多小区参数提取器 | `omcgo/internal/device/cell_extractor.go` |
| 4 | MME 池配置解析 | `omcgo/internal/device/param_extractor.go` |

### 4.3 阶段三：Service 层集成

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 更新 RegisterFromInform | `omcgo/internal/device/service.go` |
| 2 | 更新 UpdateFromInform | `omcgo/internal/device/service.go` |
| 3 | 新增 GPVResponse 处理 | `omcgo/internal/device/gpv_handler.go` |
| 4 | 新增 CellRepository | `omcgo/internal/device/cell_repository.go` |

### 4.4 阶段四：测试与验证

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 参数提取器单元测试 | `omcgo/internal/device/param_extractor_test.go` |
| 2 | 多小区配置测试 | `omcgo/internal/device/cell_extractor_test.go` |
| 3 | E2E 测试更新 | `omcgo/scripts/e2e_verify.sh` |

---

## 5. 参数清单

### 5.1 主表字段映射（14 个）

| 序号 | TR069 参数路径 | 主表字段 | 来源 |
|------|---------------|---------|------|
| 1 | `Device.DeviceInfo.HardwareVersion` | hardware_version | Inform |
| 2 | `Device.DeviceInfo.SoftwareVersion` | firmware_version | Inform |
| 3 | `Device.DeviceInfo.X_COM_MODULE_TYPE` | model_name | GPVResponse |
| 4 | `Device.DeviceInfo.X_COM_STATION_RUN_Time` | run_time | Inform |
| 5 | `Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList` | plmn | Inform |
| 6 | `Device.Services.FAPService.1.FAPControl.LTE.OpState` | cell_status | Inform |
| 7 | `Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus` | rf_status | Inform |
| 8 | `Device.FAP.GPS.LockedLatitude` | latitude | GPVResponse |
| 9 | `Device.FAP.GPS.LockedLongitude` | longitude | GPVResponse |
| 10 | `Device.DeviceInfo.AntennaInfo.Height` | height | GPVResponse |
| 11 | `Device.IP.Interface.1.IPv4Address.1.IPAddress` | ip_address | Inform |
| 12 | `Device.ManagementServer.tfcsManagerPrimsrc` | sync_source | GPVResponse |
| 13 | `Device.ManagementServer.tfcsSyncState` | sync_state | GPVResponse |
| 14 | `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.*` | mme_pool | GPVResponse |

### 5.2 多小区配置字段（8 个/小区）

| TR069 参数路径 | 字段 |
|---------------|------|
| `Device.Services.FAPService.{n}.FAPControl.LTE.CellOpState` | cell_op_state |
| `Device.Services.FAPService.{n}.FAPControl.LTE.Gateway.ExistPlmnidList` | plmn |
| `Device.Services.FAPService.{n}.FAPControl.LTE.OpState` | op_state |
| `Device.Services.FAPService.{n}.FAPControl.LTE.RFTxStatus` | rf_tx_status |
| `Device.Services.FAPService.{n}.FAPControl.LTE.Slot` | slot |
| `Device.Services.FAPService.{n}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCEnableState` | epc_enable_state |
| `Device.Services.FAPService.{n}.FAPControl.LTE.X_COM_HSS.EMBEDDED_EPCMode` | epc_mode |
| `Device.Services.FAPService.{n}.FAPControl.X_RADISYS_COM_AlarmStatus` | alarm_status |

### 5.3 MME 池配置字段（3 个/条目）

| TR069 参数路径 | 字段 |
|---------------|------|
| `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.{n}.MME1Status` | mme_status |
| `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.{n}.MMEIp1` | mme_ip |
| `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.{n}.PLMNID` | plmn_id |

### 5.4 许可证字段（单独表）

| TR069 参数路径 | 字段 |
|---------------|------|
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Code` | license_code |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Version` | version |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Author` | author |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.GenerateDate` | generate_date |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.SeqNum` | seq_num |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.{n}.*` | 容量表 |

---

## 6. 风险与注意事项

### 6.1 数据迁移

- 新增字段都有默认值，不影响现有数据
- 迁移脚本需要支持回滚

### 6.2 性能考虑

- MME 池配置使用 JSONB 存储，支持索引查询
- 多小区配置单独建表，避免主表膨胀

### 6.3 兼容性

- 新增字段为可选，不影响现有 Inform 处理流程
- GPVResponse 处理为新增功能，不影响现有逻辑

### 6.4 载波模式

- 需要确认如何获取设备载波模式（CA/SC/DC/TC）
- 可能需要从特定参数推断或配置

---

## 7. 附录

### 7.1 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `migrations/000036_add_device_tr069_fields.up.sql` | 新增 | 设备表字段扩展 |
| `migrations/000036_add_device_tr069_fields.down.sql` | 新增 | 回滚脚本 |
| `migrations/000037_create_device_licenses.up.sql` | 新增 | 许可证表 |
| `migrations/000037_create_device_licenses.down.sql` | 新增 | 回滚脚本 |
| `migrations/000038_create_device_cells.up.sql` | 新增 | 多小区表 |
| `migrations/000038_create_device_cells.down.sql` | 新增 | 回滚脚本 |
| `internal/core/model/device.go` | 修改 | 新增字段和模型 |
| `internal/device/param_extractor.go` | 新增 | 参数提取器 |
| `internal/device/cell_extractor.go` | 新增 | 多小区提取器 |
| `internal/device/cell_repository.go` | 新增 | 多小区仓储 |
| `internal/device/gpv_handler.go` | 新增 | GPVResponse 处理 |
| `internal/device/service.go` | 修改 | 集成参数提取 |
| `internal/device/pg_repository.go` | 修改 | 支持新字段 |

### 7.2 测试文件清单

| 文件 | 说明 |
|------|------|
| `internal/device/param_extractor_test.go` | 参数提取器测试 |
| `internal/device/cell_extractor_test.go` | 多小区提取器测试 |
| `internal/device/cell_repository_test.go` | 多小区仓储测试 |

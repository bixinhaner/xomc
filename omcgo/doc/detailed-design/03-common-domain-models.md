# DD-03: 公共领域模型与错误处理

> 关联功能域：全部
> 关联 backend-design.md 章节：第十三章（数据模型抽象）
> 实施阶段：Phase 1（基础建设）
> 依赖文档：DD-01

---

## 1. 概述

### 1.1 模块定位

公共领域模型（`internal/common/`）定义了整个系统共享的核心数据类型、错误处理规范和 HTTP 中间件，是所有业务模块的类型基础。

### 1.2 核心职责

- 领域常量定义（运营商、制式、设备状态等枚举）
- 核心领域模型定义（Device、Alarm、Parameter、PMCounter、KPI）
- 统一错误类型体系（sentinel errors、业务错误码）
- HTTP 中间件（认证、日志、指标、恢复）
- 通用分页与过滤结构

---

## 2. 数据模型

### 2.1 领域常量 — `internal/common/model/constants.go`

```go
// CarrierCode 运营商代码
type CarrierCode string

const (
    CarrierCMCC CarrierCode = "cmcc" // 中国移动
    CarrierCTCC CarrierCode = "ctcc" // 中国电信
    CarrierCUCC CarrierCode = "cucc" // 中国联通
)

// Technology 无线制式
type Technology string

const (
    TechLTE Technology = "lte" // 4G LTE
    TechNR  Technology = "nr"  // 5G NR SA
)

// DeviceStatus 设备状态
type DeviceStatus string

const (
    DeviceDiscovered     DeviceStatus = "discovered"
    DeviceRegistered     DeviceStatus = "registered"
    DeviceProvisioning   DeviceStatus = "provisioning"
    DeviceActive         DeviceStatus = "active"
    DeviceMaintenance    DeviceStatus = "maintenance"
    DeviceOffline        DeviceStatus = "offline"
    DeviceDecommissioned DeviceStatus = "decommissioned"
)

// AlarmSeverity 告警严重级别
type AlarmSeverity int

const (
    AlarmCritical AlarmSeverity = 1 // 紧急
    AlarmMajor    AlarmSeverity = 2 // 重要
    AlarmMinor    AlarmSeverity = 3 // 次要
    AlarmWarning  AlarmSeverity = 4 // 提示
)

// AlarmStatus 告警状态
type AlarmStatus string

const (
    AlarmActive       AlarmStatus = "active"
    AlarmAcknowledged AlarmStatus = "acknowledged"
    AlarmCleared      AlarmStatus = "cleared"
)
```

### 2.2 Device 模型 — `internal/common/model/device.go`

```go
type Device struct {
    ID                   uuid.UUID    `json:"id" db:"id"`
    SerialNumber         string       `json:"serial_number" db:"serial_number"`
    OUI                  string       `json:"oui" db:"oui"`
    ProductClass         string       `json:"product_class" db:"product_class"`
    Manufacturer         string       `json:"manufacturer" db:"manufacturer"`
    ModelName            string       `json:"model_name" db:"model_name"`
    Carrier              CarrierCode  `json:"carrier" db:"carrier"`
    Technology           Technology   `json:"technology" db:"technology"`
    DataModelID          *uuid.UUID   `json:"data_model_id,omitempty" db:"data_model_id"`
    Status               DeviceStatus `json:"status" db:"status"`
    FirmwareVersion      string       `json:"firmware_version" db:"firmware_version"`
    IPAddress            string       `json:"ip_address" db:"ip_address"`
    ConnectionRequestURL string       `json:"connection_request_url" db:"connection_request_url"`
    LastInformAt         *time.Time   `json:"last_inform_at,omitempty" db:"last_inform_at"`
    LastInformEvents     []string     `json:"last_inform_events,omitempty" db:"last_inform_events"`
    InformInterval       int          `json:"inform_interval" db:"inform_interval"`
    SiteName             string       `json:"site_name" db:"site_name"`
    SiteID               string       `json:"site_id" db:"site_id"`
    Latitude             float64      `json:"latitude" db:"latitude"`
    Longitude            float64      `json:"longitude" db:"longitude"`
    ExtensionData        map[string]interface{} `json:"extension_data,omitempty" db:"extension_data"`
    CreatedAt            time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt            time.Time    `json:"updated_at" db:"updated_at"`
}
```

### 2.3 Alarm 模型 — `internal/common/model/alarm.go`

```go
type Alarm struct {
    ID             uuid.UUID     `json:"id" db:"id"`
    DeviceID       uuid.UUID     `json:"device_id" db:"device_id"`
    Severity       AlarmSeverity `json:"severity" db:"severity"`
    AlarmType      string        `json:"alarm_type" db:"alarm_type"`
    AlarmCode      string        `json:"alarm_code" db:"alarm_code"`
    Description    string        `json:"description" db:"description"`
    Status         AlarmStatus   `json:"status" db:"status"`
    RaisedAt       time.Time     `json:"raised_at" db:"raised_at"`
    AcknowledgedAt *time.Time    `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
    ClearedAt      *time.Time    `json:"cleared_at,omitempty" db:"cleared_at"`
    AcknowledgedBy string        `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
    AdditionalInfo map[string]string `json:"additional_info,omitempty" db:"additional_info"`
}
```

### 2.4 Parameter 模型 — `internal/common/model/parameter.go`

```go
type ParameterType string

const (
    ParamString   ParameterType = "string"
    ParamInt      ParameterType = "int"
    ParamUint     ParameterType = "unsignedInt"
    ParamBool     ParameterType = "boolean"
    ParamDateTime ParameterType = "dateTime"
)

type DeviceParameter struct {
    DeviceID       uuid.UUID     `json:"device_id" db:"device_id"`
    ParameterPath  string        `json:"parameter_path" db:"parameter_path"`
    ParameterValue string        `json:"parameter_value" db:"parameter_value"`
    ParameterType  ParameterType `json:"parameter_type" db:"parameter_type"`
    Writable       bool          `json:"writable" db:"writable"`
    LastUpdatedAt  time.Time     `json:"last_updated_at" db:"last_updated_at"`
}
```

### 2.5 PMCounter 与 KPI — `internal/common/model/pm_counter.go` & `kpi.go`

```go
type PMCounter struct {
    Time          time.Time `json:"time" db:"time"`
    DeviceID      uuid.UUID `json:"device_id" db:"device_id"`
    CellID        string    `json:"cell_id" db:"cell_id"`
    CounterGroup  string    `json:"counter_group" db:"counter_group"`
    CounterName   string    `json:"counter_name" db:"counter_name"`
    CounterValue  float64   `json:"counter_value" db:"counter_value"`
    Granularity   int       `json:"granularity" db:"granularity"` // 分钟
}

type KPIValue struct {
    Time       time.Time   `json:"time" db:"time"`
    DeviceID   uuid.UUID   `json:"device_id" db:"device_id"`
    CellID     string      `json:"cell_id" db:"cell_id"`
    KPIName    string      `json:"kpi_name" db:"kpi_name"`
    KPIValue   float64     `json:"kpi_value" db:"kpi_value"`
    Carrier    CarrierCode `json:"carrier" db:"carrier"`
    Technology Technology  `json:"technology" db:"technology"`
}
```

---

## 3. 错误类型体系 — `internal/common/errors/`

### 3.1 Sentinel Errors — `errors.go`

```go
var (
    ErrNotFound       = errors.New("resource not found")
    ErrAlreadyExists  = errors.New("resource already exists")
    ErrInvalidInput   = errors.New("invalid input")
    ErrUnauthorized   = errors.New("unauthorized")
    ErrForbidden      = errors.New("forbidden")
    ErrInternal       = errors.New("internal error")
    ErrTimeout        = errors.New("operation timed out")
    ErrUnavailable    = errors.New("service unavailable")
)
```

### 3.2 业务错误码 — `codes.go`

```go
type BusinessError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}

func (e *BusinessError) Error() string
func (e *BusinessError) Unwrap() error

// 业务错误码范围：
// 1000-1999: 设备管理
// 2000-2999: 数据模型
// 3000-3999: ACS/TR069
// 4000-4999: 性能管理
// 5000-5999: 告警管理
// 6000-6999: 自动开站
```

### 3.3 HTTP 错误响应 — `response.go`

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func NewErrorResponse(err error) ErrorResponse
func AbortWithError(c *gin.Context, statusCode int, err error)
```

---

## 4. HTTP 中间件 — `internal/common/middleware/`

### 4.1 认证中间件 — `auth.go`

```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc
// 从 Authorization header 提取 JWT token，验证后将用户信息存入 context
```

### 4.2 请求日志 — `logging.go`

```go
func RequestLogger(logger *zap.Logger) gin.HandlerFunc
// 记录请求方法、路径、状态码、耗时、客户端 IP
```

### 4.3 Prometheus 指标 — `metrics.go`

```go
func PrometheusMetrics() gin.HandlerFunc
// 记录 http_requests_total, http_request_duration_seconds
```

### 4.4 Panic 恢复 — `recovery.go`

```go
func Recovery(logger *zap.Logger) gin.HandlerFunc
// 捕获 panic，记录堆栈，返回 500
```

---

## 5. 通用分页与过滤 — `internal/common/model/pagination.go`

```go
type ListRequest struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    SortBy   string `form:"sort_by"`
    SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc"`
}

type ListResponse[T any] struct {
    Items      []T   `json:"items"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    TotalPages int   `json:"total_pages"`
}

func (r *ListRequest) Offset() int { return (r.Page - 1) * r.PageSize }
func (r *ListRequest) Limit() int  { return r.PageSize }
```

---

## 6. 实施子阶段

### 阶段 3a：领域常量 + Device 模型

**交付物**：`constants.go`、`device.go`
**验证**：编译通过，类型可用

### 阶段 3b：其余领域模型

**交付物**：`alarm.go`、`parameter.go`、`pm_counter.go`、`kpi.go`、`pagination.go`
**验证**：编译通过

### 阶段 3c：错误类型体系

**交付物**：`errors/errors.go`、`errors/codes.go`、`errors/response.go`
**验证**：单元测试覆盖错误包装与 HTTP 响应转换

### 阶段 3d：HTTP 中间件

**交付物**：`middleware/auth.go`、`middleware/logging.go`、`middleware/metrics.go`、`middleware/recovery.go`
**验证**：中间件在 Gin router 中正常工作

---

## 7. 文件清单

```
internal/common/model/constants.go
internal/common/model/device.go
internal/common/model/alarm.go
internal/common/model/parameter.go
internal/common/model/pm_counter.go
internal/common/model/kpi.go
internal/common/model/pagination.go
internal/common/errors/errors.go
internal/common/errors/codes.go
internal/common/errors/response.go
internal/common/middleware/auth.go
internal/common/middleware/logging.go
internal/common/middleware/metrics.go
internal/common/middleware/recovery.go
```

---

## 8. 测试策略

- Device 状态机转换有效性测试
- 错误类型 Wrap/Unwrap 测试
- 分页计算测试（边界值）
- 中间件功能测试（使用 httptest）

---

## 9. 参考

- backend-design.md 第十三章：数据模型抽象（13.1-13.3）
- CLAUDE.md 第 5.1 节：Go 编码规范

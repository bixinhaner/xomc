# DD-05: 运营商抽象层

> 关联功能域：全部（跨域公共层）
> 关联 backend-design.md 章节：第十二章（运营商抽象层）
> 实施阶段：Phase 2（CMCC）→ Phase 3（CTCC/CUCC）
> 依赖文档：DD-03

---

## 1. 概述

### 1.1 模块定位

运营商抽象层（`internal/carrier/`）通过接口 + 适配器模式封装三家运营商的差异，使业务模块无需硬编码运营商判断逻辑。**禁止** `if carrier == "cmcc"` 模式，所有差异通过适配器实现。

### 1.2 核心职责

- 定义统一 Carrier 接口（12+ 方法）
- CarrierRegistry 注册与查找
- CMCC/CTCC/CUCC 三家适配器实现
- 参数路径映射（运营商路径 ↔ 统一内部名称）
- KPI 公式定义、告警映射、开站模板

---

## 2. 接口设计

### 2.1 Carrier 接口 — `internal/carrier/carrier.go`

```go
type Carrier interface {
    // 标识
    Code() model.CarrierCode
    Name() string

    // 制式支持
    SupportedTechnologies() []model.Technology

    // 数据模型
    DefaultDataModelVersions(tech model.Technology) []string
    LoadDefaultDataModel(tech model.Technology, version string) (*datamodel.DataModel, error)

    // ���知厂商/产品
    KnownOUIProductClasses(tech model.Technology) []OUIProductClassInfo

    // 参数映射
    MapParameterToUnified(carrierPath string) string
    MapUnifiedToParameter(unifiedName string) string

    // 开站模板
    ProvisioningTemplates(tech model.Technology) []*ProvisionTemplate

    // KPI 定义
    KPIDefinitions(tech model.Technology) []*KPIDefinition

    // 告警映射
    AlarmSeverityMapping(carrierAlarmCode string) model.AlarmSeverity

    // Inform 事件处理
    HandleInformEvents(device *model.Device, events []tr069.EventStruct) ([]Command, error)

    // 参数校验
    ValidateParameter(path string, value string) error
}
```

### 2.2 CarrierRegistry — `internal/carrier/registry.go`

```go
type CarrierRegistry struct {
    carriers map[model.CarrierCode]Carrier
    mu       sync.RWMutex
}

func NewRegistry() *CarrierRegistry
func (r *CarrierRegistry) Register(code model.CarrierCode, carrier Carrier)
func (r *CarrierRegistry) Get(code model.CarrierCode) (Carrier, error)
func (r *CarrierRegistry) All() []Carrier
```

### 2.3 辅助类型

```go
type OUIProductClassInfo struct {
    OUI              string
    ProductClass     string
    ManufacturerName string
    Description      string
    HasCustomModel   bool
}

type ProvisionTemplate struct {
    Name       string
    Technology model.Technology
    Parameters map[string]interface{} // 参数名 → 默认值
    Required   []string               // 必填参数列表
}

type KPIDefinition struct {
    Name        string
    DisplayName string
    Formula     string   // 如 "rrc_setup_success / rrc_setup_attempt * 100"
    Unit        string   // "%", "ms", "Mbps"
    Counters    []string // 依赖的计数器名称
    Category    string   // "accessibility", "retainability", "mobility", "throughput"
}

type Command struct {
    Method string
    Params interface{}
}
```

---

## 3. 详细设计

### 3.1 CMCC 适配器 — `internal/carrier/cmcc/adapter.go`

```go
type CMCCCarrier struct {
    paramMapping   map[string]string // carrier path → unified name
    reverseMapping map[string]string // unified name → carrier path
}

func New() *CMCCCarrier

func (c *CMCCCarrier) Code() model.CarrierCode { return model.CarrierCMCC }
func (c *CMCCCarrier) Name() string            { return "中国移动" }

func (c *CMCCCarrier) SupportedTechnologies() []model.Technology {
    return []model.Technology{model.TechLTE, model.TechNR}
}

func (c *CMCCCarrier) DefaultDataModelVersions(tech model.Technology) []string {
    switch tech {
    case model.TechLTE: return []string{"V2.1", "V2.3"}
    case model.TechNR:  return []string{"V1.7", "V1.9.4"}
    }
    return nil
}

func (c *CMCCCarrier) KnownOUIProductClasses(tech model.Technology) []OUIProductClassInfo {
    // 华为、中兴、京信等已知厂商
}
```

CMCC 特有扩展：
```go
// SupportsDirectConnection 移动独有：网元直连接口
func (c *CMCCCarrier) SupportsDirectConnection() bool { return true }
```

### 3.2 CTCC 适配器 — `internal/carrier/ctcc/adapter.go`

```go
func (c *CTCCCarrier) Code() model.CarrierCode { return model.CarrierCTCC }
func (c *CTCCCarrier) Name() string            { return "中国电信" }

func (c *CTCCCarrier) SupportedTechnologies() []model.Technology {
    return []model.Technology{model.TechLTE, model.TechNR}
}
```

CTCC 特点：
- LTE + NR 双制式
- 开站模板版本迭代快（4G V1.0.2、5G V2.8.7）
- 5G 共建共享参数扩展
- 设备指标关联定义

### 3.3 CUCC 适配器 — `internal/carrier/cucc/adapter.go`

```go
func (c *CUCCCarrier) Code() model.CarrierCode { return model.CarrierCUCC }
func (c *CUCCCarrier) Name() string            { return "中国联通" }

func (c *CUCCCarrier) SupportedTechnologies() []model.Technology {
    return []model.Technology{model.TechNR} // 仅 5G NR
}
```

CUCC 特点：
- 仅 NR 制式（无 LTE）
- 最完整的北向 OSS 接口定义
- 社会化基站概念
- SA 无线���置参数独立映射

### 3.4 运营商差异对比矩阵

| 能力 | CMCC | CTCC | CUCC |
|------|------|------|------|
| LTE 支持 | ✅ | ✅ | ❌ |
| NR 支持 | ✅ | ✅ | ✅ |
| 网元直连 | ✅ | ❌ | ❌ |
| 北向 OSS 接口 | ❌ | ❌ | ✅ |
| 共建共享 | ❌ | ✅ | ❌ |
| 开站模板 | ✅ | ✅ | ❌ |
| PM 统计数据规范 | ✅ | ✅ | ✅ |
| MR 规范 | ✅ | ✅ | ✅ |
| 告警独立规范 | ❌ | ❌ | ✅ |

---

## 4. 实施子阶段

### 阶段 5a：Carrier 接口 + Registry + 常量（Phase 2）

**交付物**：`carrier.go`、`registry.go`、辅助类型
**验证**：编译通过

### 阶段 5b：CMCC 适配器（Phase 2）

**交付物**：`cmcc/adapter.go`、`cmcc/defaults.go`
**验证**：CMCC 适配器所有接口方法可调用

### 阶段 5c：CTCC 适配器（Phase 3）

**交付物**：`ctcc/adapter.go`、`ctcc/defaults.go`
**验证**：电信 LTE + NR 双制式支持

### 阶段 5d：CUCC 适配器（Phase 3）

**交付物**：`cucc/adapter.go`、`cucc/defaults.go`
**验证**：联通 NR 支持 + 北向接口特性

---

## 5. 文件清单

```
internal/carrier/carrier.go
internal/carrier/registry.go
internal/carrier/types.go
internal/carrier/cmcc/adapter.go
internal/carrier/cmcc/defaults.go
internal/carrier/ctcc/adapter.go
internal/carrier/ctcc/defaults.go
internal/carrier/cucc/adapter.go
internal/carrier/cucc/defaults.go
```

---

## 6. 测试策略

- 每个适配器的 table-driven 测试（制式支持、参数映射、KPI 定义）
- CarrierRegistry 并发安全测试
- 参数映射双向往返测试

---

## 7. 参考

- backend-design.md 第十二章：运营商抽象层
- doc/功能索引.md：运营商覆盖矩阵
- CLAUDE.md 第 5.1 节：运营商适配规范

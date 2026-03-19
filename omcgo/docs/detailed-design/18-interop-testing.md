# DD-18: 互操作测试（F10）

> 关联功能域：F10（互操作测试）
> 关联 backend-design.md 章节：第二章（模块 10 — interop）
> 实施阶段：Phase 4（北向与规模化）
> 依赖文档：DD-07, DD-08

---

## 1. 概述

### 1.1 模块定位

互操作测试（`internal/interop/`）验证不同厂商的基站设备与 OMC 网管之间的端到端互通能力，是系统质量保障的重要环节。

### 1.2 核心职责

- TR069 协议一致性测试框架
- 数据模型参数一致性验证
- 端到端联调测试工具
- 测试报告生成

---

## 2. 接口设计

### 2.1 ConformanceTestRunner — `internal/interop/conformance/`

```go
type ConformanceTestRunner struct {
    acsClient  *ACSTestClient
    dataModelReg *datamodel.DataModelRegistry
    logger     *zap.Logger
}

type TestCase struct {
    ID          string
    Name        string
    Description string
    Category    string // "protocol", "datamodel", "rpc", "inform"
    Steps       []TestStep
    Expected    interface{}
}

type TestResult struct {
    TestCaseID string
    Passed     bool
    Duration   time.Duration
    Details    string
    Error      string
}

// RunAll 运行所有一致性测试
func (r *ConformanceTestRunner) RunAll(ctx context.Context, deviceSN string) ([]TestResult, error)

// RunByCategory 按类别运行测试
func (r *ConformanceTestRunner) RunByCategory(ctx context.Context, deviceSN, category string) ([]TestResult, error)
```

### 2.2 DataModel Validator — `internal/interop/validator/`

```go
type DataModelValidator struct {
    dataModelReg *datamodel.DataModelRegistry
    acsClient    *ACSTestClient
}

// ValidateDevice 验证设备参数与数据模型的一致性
func (v *DataModelValidator) ValidateDevice(ctx context.Context, deviceSN string) (*ValidationReport, error)

type ValidationReport struct {
    DeviceSN       string
    ModelVersion   string
    TotalParams    int
    MatchedParams  int
    MismatchParams []ParamMismatch
    MissingParams  []string
    ExtraParams    []string
}

type ParamMismatch struct {
    Path     string
    Expected string
    Actual   string
    Type     string // "type_mismatch", "value_out_of_range", "not_writable"
}
```

---

## 3. 测试用例类别

### 3.1 协议一致性测试

- Inform 报文格式验证
- RPC 方法响应正确性（9 种 RPC）
- 会话管理（建立/保持/超时/断开）
- SOAP/XML 格式合规性
- HTTP 认证（Digest/Basic）

### 3.2 数据模型一致性测试

- 参数路径正确性
- 参数类型一致性
- 读写权限验证
- 取值范围边界测试
- 必选参数完整性

### 3.3 端到端联调测试

- 设备接入 → Inform → 注册
- 参数读写 → 配置下发 → 生效验证
- 固件升级完整流程
- 自动开站端到端
- 异常场景（网络中断、设备重启）

---

## 4. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 联调测试规范 | ❌ | ✅（自研网管联调方案）| ❌ |
| 接口规范说明 | ❌ | ✅（贵州地方）| ❌ |
| 一致性测试规范 | ❌ | ❌ | ✅ (V1.1) |

---

## 5. 实施子阶段

### 阶段 18a：测试框架 + 协议一致性（Phase 4）
### 阶段 18b：数据模型一致性 + 端到端（Phase 4）

---

## 6. 文件清单

```
internal/interop/conformance/runner.go
internal/interop/conformance/cases.go
internal/interop/conformance/protocol_tests.go
internal/interop/validator/validator.go
internal/interop/validator/report.go
```

---

## 7. 参考

- doc/features/10-interop-testing.md：F10 全部子功能

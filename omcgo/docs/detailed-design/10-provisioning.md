# DD-10: 自动开站（F09）

> 关联功能域：F09（自动开站/自动开通）
> 关联 backend-design.md 章节：第六章（核心处理管线 - 自动开站管线）
> 实施阶段：Phase 2（核心功能）
> 依赖文档：DD-07, DD-08, DD-09

---

## 1. 概述

### 1.1 模块定位

自动开站（`internal/provision/`）实现基站设备从上电到投入服务的自动化流程，包括设备自动发现、身份认证、模板匹配、配置下发、校验激活。

### 1.2 核心职责

- 开站状态机（bootstrap → 识别 → 模板匹配 → 配置下发 → 验证 → 激活）
- 模板匹配规则引擎
- 配置下发编排（命令序列）
- 批量开站管理

---

## 2. 接口设计

### 2.1 ProvisioningEngine — `internal/provision/workflow/engine.go`

```go
type ProvisioningEngine struct {
    deviceService   *device.DeviceService
    templateService *template.TemplateService
    commandQueue    acs.CommandQueue
    connReq         *connreq.ConnReqClient
    dataModelReg    *datamodel.DataModelRegistry
    eventBus        event.EventBus
    repo            ProvisioningTaskRepository
    logger          *zap.Logger
}

// StartProvisioning 开始开站流程
func (e *ProvisioningEngine) StartProvisioning(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error)

// HandleBootstrap 处理 Bootstrap 事件（自动触���）
func (e *ProvisioningEngine) HandleBootstrap(ctx context.Context, event event.Event) error

// AdvanceState 推进任务状态
func (e *ProvisioningEngine) AdvanceState(ctx context.Context, taskID uuid.UUID) error
```

### 2.2 开站任务模型

```go
type ProvisioningState string

const (
    StateDiscovered    ProvisioningState = "discovered"
    StateIdentifying   ProvisioningState = "identifying"
    StateMatching      ProvisioningState = "matching"
    StateConfiguring   ProvisioningState = "configuring"
    StateVerifying     ProvisioningState = "verifying"
    StateCompleted     ProvisioningState = "completed"
    StateFailed        ProvisioningState = "failed"
)

type ProvisioningTask struct {
    ID           uuid.UUID
    DeviceID     uuid.UUID
    TemplateID   *uuid.UUID
    Status       ProvisioningState
    CurrentStep  int
    TotalSteps   int
    ErrorMessage string
    RetryCount   int
    StartedAt    *time.Time
    CompletedAt  *time.Time
    CreatedAt    time.Time
}
```

---

## 3. 详细设计

### 3.1 自动开站流程

```
设备上电 → DHCP 获取 IP + ACS URL → Bootstrap Inform 到达 ACS
  │
  ├── 1. ACS 解析 Inform → 发布 device.inform.bootstrap
  │
  ├── 2. ProvisioningEngine 收到事件
  │     ├── 创建 ProvisioningTask (state=discovered)
  │     └── 创建/更新 Device 记录
  │
  ├── 3. 识别设备 (state=identifying)
  │     ├── 从 Inform 提取 OUI + ProductClass
  │     ├── 解析数据模型 (DataModelRegistry.Resolve)
  │     └── 确定运营商和制式
  │
  ├── 4. 匹配开站模板 (state=matching)
  │     ├── 按 carrier + tech + product_class 查找模板
  │     └── 模板中定义：必填参数、默认值、下发顺序
  │
  ├── 5. 配置下发 (state=configuring)
  │     ├── Step 1: GetParameterValues（读取当前配置）
  │     ├── Step 2: SetParameterValues（写入新配置）
  │     ├── Step 3: Download（固件升级，如需）
  │     ├── Step 4: Reboot（如需重启）
  │     └── 命令排入 Redis 队列，等待 CPE 下次连接执行
  │
  ├── 6. 配置校验 (state=verifying)
  │     ├── GetParameterValues（读回验证）
  │     ├── 对比期望值与实际值
  │     └── 不匹配则重试或标记失败
  │
  └── 7. 激活 (state=completed)
        ├── Device.Status = active
        └── 发布事件 provision.completed
```

### 3.2 模板匹配规则

```go
// MatchTemplate 按优先级匹配开站模板
func MatchTemplate(templates []*ConfigTemplate, device *model.Device) *ConfigTemplate {
    // 优先级从高到低：
    // 1. carrier + tech + product_class（精确匹配）
    // 2. carrier + tech（运营商默认模板）
    // 3. tech（制式默认模板）
}
```

### 3.3 配置下发命令编排

```go
type ProvisioningStep struct {
    Order    int
    Method   string // GetParameterValues, SetParameterValues, Download, Reboot
    Params   interface{}
    Timeout  time.Duration
    Required bool
}

func BuildProvisioningSteps(template *ConfigTemplate, currentParams map[string]string) []ProvisioningStep
```

### 3.4 数据库 Schema

```sql
CREATE TABLE provisioning_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id      UUID NOT NULL REFERENCES devices(id),
    template_id    UUID REFERENCES config_templates(id),
    status         VARCHAR(20) NOT NULL DEFAULT 'discovered',
    current_step   INTEGER DEFAULT 0,
    total_steps    INTEGER DEFAULT 0,
    error_message  TEXT,
    retry_count    INTEGER DEFAULT 0,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3.5 REST API

```
GET    /api/v1/provisioning/tasks           任务列表
POST   /api/v1/provisioning/tasks           创建任务（手动触发）
GET    /api/v1/provisioning/tasks/{id}      任务状态
GET    /api/v1/provisioning/templates       模板列表
POST   /api/v1/provisioning/templates       创建模板
```

---

## 4. 运营商差异

| 差异点 | CMCC | CTCC | CUCC |
|--------|------|------|------|
| 自动开站规范 | 有（独立技术要求文档）| 有（开站模板）| 无（含在 OMC-R 规范中）|
| 4G 开站模板 | 无独立模板 | 有（V1.0.2）| — |
| 5G 开站模板 | 无独立模板 | 有（V2.8.7）| 无 |

---

## 5. 实施子阶段

### 阶段 10a：开站状态机 + 基础工作流（Phase 2）

**交付物**：状态机、ProvisioningTask CRUD、Bootstrap 事件处理
**验证**：Bootstrap Inform → 创建任务 → 状态流转

### 阶段 10b：模板匹配 + 配置下发编排（Phase 2）

**交付物**：模板匹配、命令排队、端到端开站
**验证**：设备上电 → 自动配置 → 激活

### 阶段 10c：批量开站（Phase 3/4）

**交付物**：批量任务管理、并发控制
**验证**：批量创建 100 个开站任务，进度跟踪

---

## 6. 文件清单

```
internal/provision/workflow/engine.go
internal/provision/workflow/states.go
internal/provision/workflow/steps.go
internal/provision/template/service.go
internal/provision/template/matcher.go
internal/provision/batch/batch.go
internal/provision/repository.go
internal/provision/service.go
```

---

## 7. 参考

- backend-design.md 第六章：自动开站管线
- doc/features/09-auto-provisioning.md：F09 全部子功能

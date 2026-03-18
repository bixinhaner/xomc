# InformHandler 迁移至 Worker 进程的可行性评估

> 评估 `InformHandler`（设备 Inform 事件订阅者）从 App 进程迁移到 Worker 进程的合理性，基于项目实际代码分析。

---

## 一、问题背景

`InformHandler` 在 App 进程中通过 `QueueSubscribe` 订阅 `device.inform.*` 事件，属于长期驻留的 NATS 持久化订阅。App 进程同时承担 REST API 服务和事件订阅处理，而 Worker 进程专门负责异步事件处理。直觉上，将事件订阅者迁移到 Worker 似乎更合理。

**但经过对项目代码的深入分析，结论是：InformHandler 不应迁移到 Worker，应保留在 App。**

---

## 二、现状分析

### 2.1 App 与 Worker 当前的订阅者分布

**App 进程（5 个订阅者）**：

| 订阅者 | 事件 | Queue Group | 处理特征 |
|--------|------|-------------|---------|
| `InformHandler` | `device.inform.bootstrap/periodic/value_change` | `device-manager` | 轻量：1 次 DB 查询 + 1 次 DB 写入 + 1 次 Redis SET |
| `ProvisioningEngine` | `device.inform.bootstrap` + `provision.task.*` | `provisioning` | 中量：DB 查询 + 模板匹配 + 命令入队 |
| `SoftwareService` | `software.upgrade.*` | — | 轻量：状态机转换 |
| `PushEngine` | `alarm.*` + `pm.*` | — | 轻量：转发到外部 OSS |
| `ReportService` | `report.generation.requested` | — | 隐式订阅 |

**Worker 进程（6 个订阅者）**：

| 订阅者 | 事件 | Queue Group | 处理特征 |
|--------|------|-------------|---------|
| `TransferBridge` | `device.inform.autonomous_transfer_complete` | `transfer-bridge` | **重 I/O**：HTTP 下载 + MinIO 写入 |
| `PMCollector` | `pm.file.received` | `pm-workers` | **重 CPU/IO**：XML 流式解析 + 批量写入 + KPI 计算 |
| `MRCollector` | `mr.file.received` | `mr-workers` | **重 CPU/IO**：MR 文件解析 + 批量写入 |
| `AlarmReceiver` | `alarm.raw.*` | `alarm-workers` | 中量：去重 + 关联 + 存储 |
| `BackupExecutor` | `backup.task.created` | `backup-executors` | 中量：命令入队 + 唤醒设备 |
| `ReportGenerator` | `report.generation.requested` | `report-generators` | **重 CPU/IO**：数据聚合 + 报表生成 + MinIO 上传 |

### 2.2 InformHandler 依赖分析

```go
// internal/device/service.go
type DeviceService struct {
    deviceRepo DeviceRepository          // PostgreSQL
    paramRepo  DeviceParameterRepository // PostgreSQL
    heartbeat  *HeartbeatMonitor         // Redis (可选，nil 安全)
    logger     *zap.Logger
}
```

`RegisterFromInform()` 和 `UpdateFromInform()` 的依赖极简：

| 依赖 | 类型 | Worker 中是否可用 |
|------|------|------------------|
| PostgreSQL (PgPool) | 数据库连接池 | **是**，Worker 已初始化 |
| Redis | 心跳 SET 操作 | **是**，Worker 已初始化 |
| CarrierRegistry | OUI 到运营商映射 | **是**，Worker 已初始化 |
| zap.Logger | 日志 | **是** |

**技术上，InformHandler 完全可以在 Worker 中运行。依赖链没有障碍。**

### 2.3 关键发现：ProvisioningEngine 与 InformHandler 的竞态条件

**这是决定不迁移的核心原因。**

两者都订阅 `device.inform.bootstrap` 事件，使用**不��的** Queue Group：

```go
// InformHandler — queue group: "device-manager"
bus.QueueSubscribe("device.inform.bootstrap", "device-manager", h.handleBootstrap)

// ProvisioningEngine — queue group: "provisioning"
bus.QueueSubscribe("device.inform.bootstrap", "provisioning", e.HandleBootstrap)
```

不同 Queue Group = NATS 向两者各发一份副本，**并行处理，无顺序保证**。

**ProvisioningEngine.HandleBootstrap() 第一步就查询设备**：

```go
// internal/provision/engine.go — HandleBootstrap()
dev, err := e.deviceService.GetDevice(ctx, evt.DeviceID)
if err != nil {
    return e.failTask(ctx, task, fmt.Errorf("get device: %w", err))
}
```

**如果 InformHandler 还没完成设备注册，GetDevice 会失败，开站任务直接标记为 Failed。**

---

## 三、迁移评估

### 3.1 迁移的好处

| 好处 | 评价 |
|------|------|
| App 进程更纯粹（只做 REST API） | 理论正确，但 App 本身就是"设备管理中心"，事件订阅是其核心职责 |
| Worker 统一处理异步事件 | Worker 的定位是重 I/O 任务，InformHandler 的处理量级不匹配 |
| 关注点分离 | 按领域分比按技术模式分更合理 |

### 3.2 迁移的风险

| 风险 | 严重程度 | 说明 |
|------|---------|------|
| **竞态条件恶化** | **高** | InformHandler 和 ProvisioningEngine 分到不同进程，网络延迟使竞态概率大幅增加 |
| **开站流程断裂** | **高** | 设备注册（Worker）和自动开站（App）跨进程，调试和追踪变困难 |
| **领域模型拆散** | **中** | DeviceService 的 REST handler 在 App，事件 handler 在 Worker，同一服务被拆到两个进程 |
| **部署复杂度增加** | **低** | Worker 需要额外初始化 DeviceService 及其依赖 |

### 3.3 为什么"长期驻留订阅"不等于"应该在 Worker"

分析项目中两类订阅者的区别：

| 特征 | App 侧订阅者 | Worker 侧订阅者 |
|------|-------------|----------------|
| **处理量级** | 轻量（DB 读写、状态转换） | 重量（文件下载/解析、KPI 计算、报表生成） |
| **响应时效** | 实时（设备注册后立即可通过 API 查询） | 延迟容忍（PM 文件晚几秒解析没影响） |
| **领域归属** | 设备管理核心流程 | 数据处理管线 |
| **与 REST API 的关系** | 紧密（同一 Service 实例被 handler 和 subscriber 共享） | 无关（Worker 不暴露 HTTP） |

**所有 EventBus 订阅者都是长期驻留的**（NATS Durable），这是订阅模型的固有属性，不是区分 App/Worker 的依据。真正的区分标准是**处理量级**和**领域归属**。

---

## 四、结论：保留在 App

### 4.1 不迁移的理由

1. **领域内聚**：设备注册/更新是设备管理的核心流程，与 DeviceService 的 REST handler（GET/POST/PUT /devices）属于同一领域，应在同一进程内
2. **竞态安全**：InformHandler 和 ProvisioningEngine 都处理 bootstrap 事件且有时序依赖，保持在同一进程至少避免了跨进程网络延迟
3. **量级不匹配**：InformHandler 处理一次 Inform 约 1-2ms（一次 DB 查询 + 一次写入），远不是"重 I/O"任务，不符合 Worker 的定位
4. **Service 实例共享**：App 中的 DeviceService 同时服务于 REST API 和事件处理，拆分会导致同一 Service 的两个实例在不同进程中独立存在

### 4.2 已修复的竞态条件

**InformHandler 和 ProvisioningEngine 之间存在的竞态条件已通过引入 `device.registered` 事件修复。**

**修复前的隐患**：

```
ACS 发布 device.inform.bootstrap
    ↓ NATS 同时投递到两个 Queue Group
    ├─ "device-manager" → InformHandler.handleBootstrap()
    │     → RegisterFromInform()（DB INSERT）
    │
    └─ "provisioning" → ProvisioningEngine.HandleBootstrap()
          → GetDevice()  ← 如果设备还没注册，这里会失败
          → failTask()   ← 开站任务直接失败
```

### 4.3 修复方案（已实施）

通过事件级联���立明确的因果时序：

```
修复后（因果有序）:
  device.inform.bootstrap ────→ InformHandler（注册设备）
                                     ↓ DB 写入成功后发布
                              device.registered ──→ ProvisioningEngine（自动开站）← 设备一定存在
```

**实施的变更**：

| 文件 | 变更 |
|------|------|
| `internal/core/event/subjects.go` | 新增 `SubjectDeviceRegistered = "device.registered"` |
| `internal/device/service.go` | DeviceService 注入 `EventBus`；`RegisterFromInform()` 入库成功后调用 `publishDeviceRegistered()` |
| `internal/provision/engine.go` | `Subscribe()` 从 `SubjectDeviceBootstrap` 改为 `SubjectDeviceRegistered` |
| `cmd/app/router/router.go` | `NewDeviceService()` 传入 `eventBus` |

**设计要点**：
- `eventBus` 字段可为 nil（nil-safe），测试中传 nil 即可
- 事件发布失败不阻塞设备注册（仅 Warn 日志），保证核心流程健壮性
- `device.registered` 事件携带完整设备信息（ID、SN、OUI、ProductClass、Carrier、Technology）
- 其他未来的 bootstrap 后置处理也可以订阅 `device.registered`，形成清晰的事件链

---

## 五、App vs Worker 职责边界总结

```
┌─────────────────────────────────────────────────────────┐
│ App (omcgo-app) — 实时设备管理 + REST API                │
│                                                         │
│  REST API:                                              │
│    GET/POST/PUT/DELETE /devices, /config, /topology ...  │
│                                                         │
│  Event Subscribers (轻量级，实时响应):                    │
│    InformHandler        ← 设备注册/更新（1-2ms/次）      │
│    ProvisioningEngine   ← 自动开站协调                   │
│    SoftwareService      ← 固件升级状态机                 │
│    PushEngine           ← 北向 OSS 转发                  │
│                                                         │
│  特点: 低延迟、领域内聚、Service 实例共享                 │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Worker (omcgo-worker) — 重 I/O 异步数据管线              │
│                                                         │
│  Event Subscribers (重量级，延迟容忍):                    │
│    TransferBridge       ← HTTP 下载 + MinIO 存储         │
│    PMCollector          ← XML 解析 + KPI 计算（秒级）    │
│    MRCollector          ← MR 文件解析（秒级）            │
│    AlarmReceiver        ← 告警去重/关联/存储             │
│    BackupExecutor       ← 备份任务执行                   │
│    ReportGenerator      ← 报表生成 + MinIO 上传          │
│                                                         │
│  特点: 高吞吐、CPU/IO 密集、可水平扩展                   │
└─────────────────────────────────────────────────────────┘
```

**划分原则**：

| 维度 | 归 App | 归 Worker |
|------|--------|----------|
| 处理耗时 | < 10ms | > 100ms |
| 实时性要求 | 高（影响 API 可见性） | 低（延迟秒级可接受） |
| 领域 | 设备管理、配置、开站 | 数据采集、解析、计算 |
| 与 REST API 关系 | 共享 Service 实例 | 独立运行 |
| 扩展需求 | 随 API 负载扩展 | 随数据量独立扩展 |

---

## 六、关键文件参考

| 文件 | 说明 |
|------|------|
| `internal/device/inform_handler.go` | InformHandler 实现（47-59: Subscribe, 61-87: handleBootstrap） |
| `internal/device/service.go` | DeviceService（39-91: RegisterFromInform, 94-140: UpdateFromInform） |
| `internal/provision/engine.go` | ProvisioningEngine（64: Subscribe, 97: GetDevice 依赖设备存在） |
| `cmd/app/router/router.go` | App 订阅者注册（72-73: InformHandler, 94-98: ProvisioningEngine） |
| `cmd/worker/main.go` | Worker 订阅者注册（64-138: registerSubscribers） |
| `internal/core/event/subjects.go` | 事件 Subject 常量定义 |
| `internal/core/bootstrap/bootstrap.go` | App/Worker 基础设施初始化（InitForApp vs InitForWorker） |

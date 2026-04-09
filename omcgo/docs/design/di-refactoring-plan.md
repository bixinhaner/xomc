# DI 容器重构方案

> 本文档描述 `cmd/app` 依赖注入（DI）容器的重构方案，解决 `router.Setup` 函数过长、对象创建混乱的问题。

---

## 1. 问题分析

### 1.1 当前状态

| 文件 | 行数 | 职责 |
|------|------|------|
| `cmd/app/main.go` | ~126 | 入口、配置加载、启动服务 |
| `cmd/app/bootstrap.go` | ~66 | 基础设施初始化（DB、Redis、NATS 等） |
| `cmd/app/router/deps.go` | ~31 | `Deps` 结构体定义 |
| `cmd/app/router/router.go` | **~589** | 依赖注入 + 路由注册（**问题核心**） |

### 1.2 核心问题

**`router.Setup` 函数存在以下问题：**

1. **单函数过长**：~550 行代码在一个函数中
2. **对象创建无组织**：50+ 个对象在函数顶部连续创建，无逻辑分组
3. **缺乏模块化**：功能域（F02-F10）的初始化代码混在一起
4. **可测试性差**：无法独立测试单个模块的初始化
5. **可读性差**：新开发者难以理解依赖关系

### 1.3 当前代码结构示意

```
router.Setup() {
    // 设备模块对象（10+ 个）
    deviceRepo := ...
    paramRepo := ...
    deviceInfoRepo := ...
    heartbeatMonitor := ...
    connReqClient := ...
    stunStore := ...
    deviceCache := ...
    deviceService := ...
    infoSyncer := ...
    offlineDetector := ...

    // 数据模型模块对象（5+ 个）
    dmRepo := ...
    importLogRepo := ...
    ouiRepo := ...
    dmCache := ...
    dmRegistry := ...

    // ... 其他模块（40+ 个对象）

    // 中间件配置（混在对象创建中间）
    corsOrigins := cfg.CORS.AllowOrigins
    r.Use(middleware.CORS(...))

    // 路由注册（混在对象创建后面）
    deviceHandler := device.NewHandler(deviceService)
    deviceHandler.RegisterRoutes(permGroup("devices"))

    // ... 更多路由注册
}
```

---

## 2. 重构目标

1. **模块化**：按功能域（F02-F10）组织依赖注入
2. **可读性**：每个模块的初始化代码独立、自包含
3. **可测试性**：支持独立测试单个模块的初始化
4. **可维护性**：新增模块时只需添加一个初始化函数
5. **渐进式**：不改变现有架构，只是重组代码结构

---

## 3. 重构方案

### 3.1 方案概述

采用 **Provider 模式** + **功能域分组**：

```
cmd/app/
├── main.go              # 入口（保持不变）
├── bootstrap.go         # 基础设施初始化（保持不变）
└── provider/
    ├── container.go     # DI 容器定义
    ├── device.go        # F06 设备模块 Provider
    ├── config.go        # F02 配置/数据模型 Provider
    ├── pm.go            # F03 性能管理 Provider
    ├── alarm.go         # F04 告警管理 Provider
    ├── mr.go            # F05 测量报告 Provider
    ├── software.go      # F06 固件管理 Provider
    ├── topology.go      # F06 拓扑管理 Provider
    ├── admin.go         # F06 用户/RBAC Provider
    ├── backup.go        # F06 备份 Provider
    ├── task.go          # F06 任务队列 Provider
    ├── northbound.go    # F08 北向接口 Provider
    ├── provision.go     # F09 自动开站 Provider
    ├── interop.go       # F10 互操作测试 Provider
    └── router.go        # 路由注册（使用 Container）
```

### 3.2 核心数据结构

```go
// provider/container.go

package provider

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/minio/minio-go/v7"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"

    "github.com/omcgo/omcgo/internal/acs/cmdqueue"
    "github.com/omcgo/omcgo/internal/core/carrier"
    "github.com/omcgo/omcgo/internal/core/event"
    "github.com/omcgo/omcgo/internal/core/appconfig"
    "github.com/omcgo/omcgo/internal/core/components"
)

// Container 聚合所有基础设施依赖（不可变）
type Container struct {
    // 基础设施
    PgPool     *pgxpool.Pool
    TsPool     *pgxpool.Pool
    Redis      redis.UniversalClient
    MinIO      *minio.Client
    EventBus   event.EventBus
    CmdQueue   *cmdqueue.RedisCommandQueue

    // 运营商
    Carriers   *carrier.CarrierRegistry

    // 配置与日志
    Cfg        *appconfig.AppConfig
    Logger     *zap.Logger

    // 生命周期
    GS         *components.GracefulShutdown
    MetricsReg *prometheus.Registry
}

// ModuleProvider 定义模块初始化接口
type ModuleProvider interface {
    // Name 返回模块名称（用于日志）
    Name() string

    // Init 初始化模块的 Repository/Service/Handler
    // 返回需要注册到路由的 Handler
    Init(ctx *Container) error

    // RegisterRoutes 注册路由
    RegisterRoutes(router RouteRegistrar) error
}

// RouteRegistrar 路由注册接口（简化 gin.RouterGroup）
type RouteRegistrar interface {
    Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup
    PublicGroup() *gin.RouterGroup    // /api/v1 (no auth)
    ProtectedGroup() *gin.RouterGroup // /api/v1 (with auth)
    PermissionGroup(resource string) *gin.RouterGroup
}
```

### 3.3 模块 Provider 示例

```go
// provider/device.go

package provider

import (
    "context"

    "github.com/omcgo/omcgo/internal/acs/cmdqueue"
    "github.com/omcgo/omcgo/internal/acs/connreq"
    "github.com/omcgo/omcgo/internal/acs/stun"
    "github.com/omcgo/omcgo/internal/device"
    "go.uber.org/zap"
)

// DeviceModule 设备模块 Provider
type DeviceModule struct {
    // Repositories
    deviceRepo     device.DeviceRepository
    paramRepo      device.DeviceParameterRepository
    deviceInfoRepo device.DeviceInfoRepository

    // Services
    deviceService  *device.DeviceService
    heartbeatMon   *device.HeartbeatMonitor
    offlineDetect  *device.OfflineDetector
    infoSyncer     *device.InfoSyncer

    // Handlers
    deviceHandler  *device.Handler
    deviceInfoHdlr *device.DeviceInfoHandler
    regHandler     *device.RegistrationHandler
    exportHandler  *device.ExportHandler
    paramTreeHdlr  *device.ParameterTreeHandler

    // Shared dependencies (from container)
    connReqClient  *connreq.Client
    stunStore      *stun.Store
    cmdQueue       *cmdqueue.RedisCommandQueue
}

func NewDeviceModule() *DeviceModule {
    return &DeviceModule{}
}

func (m *DeviceModule) Name() string {
    return "device"
}

func (m *DeviceModule) Init(c *Container) error {
    logger := c.Logger.Named("device")

    // 1. Repositories
    m.deviceRepo = device.NewPgDeviceRepository(c.PgPool)
    m.paramRepo = device.NewPgDeviceParameterRepository(c.PgPool)
    m.deviceInfoRepo = device.NewPgDeviceInfoRepository(c.PgPool)

    // 2. Shared infrastructure
    m.connReqClient = connreq.NewClient(c.Redis, logger)
    m.stunStore = stun.NewStore(c.Redis, logger)
    m.cmdQueue = c.CmdQueue

    // 3. HeartbeatMonitor
    m.heartbeatMon = device.NewHeartbeatMonitor(c.Redis, m.deviceRepo, logger)
    m.heartbeatMon.Start()
    c.GS.Register("heartbeat", 1, func(ctx context.Context) error {
        m.heartbeatMon.Stop()
        return nil
    })

    // 4. DeviceService
    deviceCache := device.NewDeviceCache(c.Redis, logger)
    m.deviceService = device.NewDeviceService(
        m.deviceRepo, m.paramRepo, m.heartbeatMon, c.EventBus, logger,
    )
    m.deviceService.SetDeviceCache(deviceCache)
    m.deviceService.SetDeviceInfoRepo(m.deviceInfoRepo)
    m.deviceService.SetCommandQueue(m.cmdQueue)
    m.deviceService.SetConnectionRequester(m.connReqClient)
    m.deviceService.SetStunAddressUpdater(m.stunStore)
    m.deviceService.SetMetrics(device.NewDeviceMetrics(c.MetricsReg))

    // 5. InfoSyncer
    m.infoSyncer = device.NewInfoSyncer(
        m.deviceInfoRepo, m.paramRepo, c.Carriers, logger,
    )
    m.deviceService.SetInfoSyncer(m.infoSyncer)

    // 6. OfflineDetector
    m.offlineDetect = device.NewOfflineDetector(
        m.deviceRepo, m.infoSyncer, c.EventBus, logger,
    )
    go func() {
        if err := m.offlineDetect.Start(context.Background()); err != nil {
            logger.Error("offline detector stopped", zap.Error(err))
        }
    }()
    c.GS.Register("offline-detector", 2, func(ctx context.Context) error { return nil })

    // 7. Handlers
    m.deviceHandler = device.NewHandler(m.deviceService)
    m.deviceInfoHdlr = device.NewDeviceInfoHandler(m.deviceService)
    m.regHandler = device.NewRegistrationHandler(
        device.NewRegistrationService(
            device.NewPgRegistrationRepository(c.PgPool), logger,
        ),
    )
    m.exportHandler = device.NewExportHandler(
        device.NewExportService(m.deviceInfoRepo, logger),
    )
    m.paramTreeHdlr = device.NewParameterTreeHandler(
        m.deviceService, m.paramRepo, nil /* dmRegistry */, logger,
    )

    logger.Info("device module initialized")
    return nil
}

func (m *DeviceModule) RegisterRoutes(r RouteRegistrar) error {
    devices := r.PermissionGroup("devices")

    m.deviceHandler.RegisterRoutes(devices)
    m.deviceInfoHdlr.RegisterRoutes(devices)
    m.regHandler.RegisterRoutes(devices)
    m.exportHandler.RegisterRoutes(devices)
    m.paramTreeHdlr.RegisterRoutes(devices)

    return nil
}
```

### 3.4 重构后的 router.go

```go
// provider/router.go

package provider

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// Setup 初始化所有模块并注册路由
func Setup(r *gin.Engine, c *Container) error {
    logger := c.Logger

    // 注册全局中间件
    setupMiddleware(r, c)

    // 初始化所有模块（按依赖顺序）
    modules := []ModuleProvider{
        // 基础模块（其他模块依赖）
        NewConfigModule(),    // F02: 数据模型、配置模板
        NewAdminModule(),     // F06: 用户/RBAC（认证中间件依赖）

        // 核心业务模块
        NewDeviceModule(),    // F06: 设备管理
        NewTopologyModule(),  // F06: 拓扑/分组
        NewSoftwareModule(),  // F06: 固件管理
        NewBackupModule(),    // F06: 备份
        NewTaskModule(),      // F06: 任务队列

        // 数据模块
        NewPMModule(),        // F03: 性能管理
        NewAlarmModule(),     // F04: 告警管理
        NewMRModule(),        // F05: 测量报告

        // 扩展模块
        NewNorthboundModule(), // F08: 北向接口
        NewProvisionModule(),  // F09: 自动开站
        NewInteropModule(),    // F10: 互操作测试
    }

    // 初始化所有模块
    for _, mod := range modules {
        if err := mod.Init(c); err != nil {
            return fmt.Errorf("init module %s: %w", mod.Name(), err)
        }
    }

    // 创建路由注册器
    registrar := NewGinRouteRegistrar(r, c)

    // 注册所有模块的路由
    for _, mod := range modules {
        if err := mod.RegisterRoutes(registrar); err != nil {
            return fmt.Errorf("register routes %s: %w", mod.Name(), err)
        }
    }

    logger.Info("all modules initialized",
        zap.Int("modules", len(modules)))
    return nil
}
```

### 3.5 目录结构对比

**重构前：**
```
cmd/app/
├── main.go           # 126 行
├── bootstrap.go      # 66 行
└── router/
    ├── deps.go       # 31 行
    └── router.go     # 589 行 ← 问题所在
```

**重构后：**
```
cmd/app/
├── main.go           # 126 行（不变）
├── bootstrap.go      # 66 行（不变）
└── provider/
    ├── container.go  # ~80 行  # DI 容器定义
    ├── router.go     # ~100 行 # 路由注册主逻辑
    ├── middleware.go # ~60 行  # 中间件配置
    ├── device.go     # ~120 行 # F06 设备模块
    ├── config.go     # ~80 行  # F02 配置模块
    ├── admin.go      # ~100 行 # F06 用户/RBAC
    ├── pm.go         # ~60 行  # F03 性能管理
    ├── alarm.go      # ~60 行  # F04 告警管理
    ├── mr.go         # ~50 行  # F05 测量报告
    ├── topology.go   # ~80 行  # F06 拓扑管理
    ├── software.go   # ~60 行  # F06 固件管理
    ├── backup.go     # ~50 行  # F06 备份
    ├── task.go       # ~60 行  # F06 任务队列
    ├── northbound.go # ~50 行  # F08 北向接口
    ├── provision.go  # ~70 行  # F09 自动开站
    └── interop.go    # ~50 行  # F10 互操作测试
```

---

## 4. 实施步骤

### 4.1 Phase 1: 基础设施（1-2 小时）

1. 创建 `cmd/app/provider/` 目录
2. 实现 `container.go`：Container 结构体和 ModuleProvider 接口
3. 实现 `router.go`：Setup 主函数骨架
4. 实现 `middleware.go`：中间件配置抽取

### 4.2 Phase 2: 核心模块（2-3 小时）

按依赖顺序迁移：

1. **ConfigModule** (F02)：dmRepo, dmRegistry, dmImporter, templateService
2. **AdminModule** (F06)：userRepo, roleRepo, jwtService, permService

### 4.3 Phase 3: 业务模块（3-4 小时）

1. **DeviceModule** (F06)：deviceRepo, deviceService, handlers
2. **TopologyModule** (F06)：groupRepo, siteRepo, topologyHandler
3. **SoftwareModule** (F06)：firmwareRepo, softwareService

### 4.4 Phase 4: 数据模块（2-3 小时）

1. **PMModule** (F03)
2. **AlarmModule** (F04)
3. **MRModule** (F05)

### 4.5 Phase 5: 扩展模块（2-3 小时）

1. **TaskModule** (F06)
2. **BackupModule** (F06)
3. **NorthboundModule** (F08)
4. **ProvisionModule** (F09)
5. **InteropModule** (F10)

---

## 5. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 重构期间功能回归 | 高 | 每个模块迁移后运行 E2E 测试 |
| 循环依赖 | 中 | 确保 Container 不依赖任何模块 |
| 初始化顺序错误 | 中 | 在 Setup 中明确定义模块顺序 |
| 测试覆盖不足 | 低 | 重构不改变业务逻辑，仅重组代码 |

---

## 6. 额外优化建议

### 6.1 懒加载（可选）

对于不常用模块（如 InteropModule），可以实现懒加载：

```go
type LazyModuleProvider struct {
    provider ModuleProvider
    initOnce sync.Once
    initErr  error
}

func (m *LazyModuleProvider) Init(c *Container) error {
    m.initOnce.Do(func() {
        m.initErr = m.provider.Init(c)
    })
    return m.initErr
}
```

### 6.2 配置驱动的模块启用（可选）

```yaml
# config.yaml
modules:
  device:
    enabled: true
  pm:
    enabled: true
  interop:
    enabled: false  # 生产环境可禁用
```

```go
// 在 Setup 中
for _, mod := range modules {
    if !c.Cfg.Modules.IsModuleEnabled(mod.Name()) {
        logger.Debug("module disabled, skipping", zap.String("module", mod.Name()))
        continue
    }
    // ...
}
```

### 6.3 健康检查增强

为每个模块添加健康检查接口：

```go
type ModuleProvider interface {
    Name() string
    Init(ctx *Container) error
    RegisterRoutes(router RouteRegistrar) error

    // 可选：健康检查
    HealthCheck() error
}
```

---

## 7. 总结

| 指标 | 重构前 | 重构后 |
|------|--------|--------|
| 单文件最大行数 | 589 行 | ~120 行 |
| 文件数量 | 4 个 | 16 个 |
| 模块化程度 | 低（单函数） | 高（按功能域分组） |
| 可测试性 | 差 | 好（可独立测试每个 Provider） |
| 新增模块成本 | 修改 router.go | 新增一个 provider 文件 |

重构后，代码结构清晰，每个模块的初始化逻辑自包含，便于维护和测试。

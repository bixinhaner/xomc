# 方案：简化 main.go — 提取 bootstrap 启动框架

## 1. 现状分析

### 三个入口文件的代码量

| 入口 | 行数 | 职责 |
|------|------|------|
| `cmd/app/main.go` | 200 | CLI + 配置 + 11 个组件连接 + Gin + HTTP 服务 + 信号处理 |
| `cmd/acs/main.go` | 147 | CLI + 配置 + 4 个组件连接 + ACS 服务 + 信号处理 |
| `cmd/worker/main.go` | 236 | CLI + 配置 + 7 个组件连接 + 6 个业务订阅者 + 信号处理 |

### 重复代码模式

三个 main.go 中存在大量重复的**基础设施初始化模式**：

```
CLI 参数解析          ← 三处重复
配置加载              ← 三处重复
日志初始化            ← 三处重复
GracefulShutdown 创建 ← 三处重复
Redis 连接            ← 三处重复
NATS 连接 + EnsureStreams ← 三处重复
EventBus 创建         ← 三处重复
CarrierRegistry 注册  ← app + worker 重复
PostgreSQL 连接       ← app + worker 重复
TimescaleDB 连接      ← app + worker 重复
MinIO 连接            ← app + worker 重复
Prometheus 指标服务   ← 三处重复
信号等待 + 优雅关机   ← 三处重复
```

**统计**：三个文件约 583 行，其中约 300 行为重复的基础设施代码。

### app/main.go 的问题

当前 `runApp()` 函数 150+ 行，做了太多事情：
1. 加载配置
2. 初始化日志
3. 创建 GracefulShutdown
4. 逐一连接 5 个基础设施（PostgreSQL、Redis、TimescaleDB、MinIO、NATS）
5. 创建 EventBus
6. 注册运营商
7. 创建命令队列
8. 配置 Gin
9. 调用 router.Setup
10. 启动 Prometheus 指标服务
11. 启动 HTTP 服务
12. 等待信号
13. 优雅关机

---

## 2. 设计目标

参考用户给出的范式：

```go
func main() {
    initializeSystem()
    core.RunServer()
}
```

**目标**：
- `main()` 不超过 10 行
- `runApp()` 不超过 50 行
- 基础设施连接逻辑不在 main.go 中
- 三个入口共享同一套基础设施初始化代码
- 不破坏现有的 `router.Setup` + `Deps` 模式

---

## 3. 方案设计

### 核心思路：新增 `internal/bootstrap` 包

将基础设施连接和生命周期管理提取到 `internal/bootstrap` 包，提供统一的 `App` 容器。

### 3.1 目录结构变化

```
internal/
├── bootstrap/              # 新增：启动框架
│   ├── app.go              # App 容器（持有所有基础设施连接）
│   ├── infra.go            # 基础设施初始化（PostgreSQL/Redis/NATS/MinIO）
│   └── server.go           # HTTP 服务 + 信号处理 + 优雅关机
├── components/             # 保持不变（底层适配器）
└── ...
```

### 3.2 App 容器

```go
// internal/bootstrap/app.go

package bootstrap

// App 持有应用运行所需的全部基础设施连接
type App struct {
    Cfg             *appconfig.AppConfig
    Logger          *zap.Logger
    GS              *components.GracefulShutdown

    PgPool          *pgxpool.Pool       // 业务数据库
    TsPool          *pgxpool.Pool       // 时序数据库（可选）
    Redis           redis.UniversalClient
    MinIO           *minio.Client       // 可选
    NATS            *natscomp.Client
    EventBus        event.EventBus
    CarrierRegistry *carrier.CarrierRegistry
    CmdQueue        *cmdqueue.RedisCommandQueue
    MetricsReg      *prometheus.Registry
}
```

### 3.3 基础设施初始化

```go
// internal/bootstrap/infra.go

// InitInfra 根据配置初始化所有基础设施连接
// opts 控制哪些组件需要初始化（ACS 不需要 PostgreSQL/MinIO）
func InitInfra(cfg interface{}, opts InfraOpts) (*App, error) {
    // 1. 日志
    // 2. GracefulShutdown
    // 3. 按 opts 初始化各组件
    // 4. 注册 shutdown hooks
    // 5. 返回 App
}

// InfraOpts 控制初始化哪些基础设施
type InfraOpts struct {
    NeedPostgres  bool
    NeedTimescale bool
    NeedRedis     bool
    NeedMinIO     bool
    NeedNATS      bool
    NeedCarriers  bool
    NeedCmdQueue  bool
}
```

预定义常用配置：

```go
var (
    // OptsApp — 主应用需要全部组件
    OptsApp = InfraOpts{
        NeedPostgres: true, NeedTimescale: true,
        NeedRedis: true, NeedMinIO: true,
        NeedNATS: true, NeedCarriers: true, NeedCmdQueue: true,
    }

    // OptsACS — ACS 只需要 Redis + NATS
    OptsACS = InfraOpts{
        NeedRedis: true, NeedNATS: true,
    }

    // OptsWorker — Worker 需要全部组件
    OptsWorker = InfraOpts{
        NeedPostgres: true, NeedTimescale: true,
        NeedRedis: true, NeedMinIO: true,
        NeedNATS: true, NeedCarriers: true, NeedCmdQueue: true,
    }
)
```

### 3.4 HTTP 服务 + 信号处理

```go
// internal/bootstrap/server.go

// ListenAndServe 启动 HTTP 服务 + Prometheus 指标服务，阻塞等待信号后优雅关机
func (app *App) ListenAndServe(handler http.Handler, addr string) error {
    // 1. 启动 Prometheus 指标服务
    // 2. 启动主 HTTP 服务
    // 3. 等待 SIGINT/SIGTERM
    // 4. 调用 gs.Shutdown()
}
```

### 3.5 简化后的 main.go

#### cmd/app/main.go（~40 行）

```go
package main

import (
    "fmt"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/omcgo/omcgo/cmd/app/router"
    "github.com/omcgo/omcgo/internal/appconfig"
    "github.com/omcgo/omcgo/internal/bootstrap"
    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "omcgo-app",
        Short: "OMC Main Application",
        RunE:  runApp,
    }
    rootCmd.Flags().String("config", "cmd/app/etc/config.dev.yaml", "config file path")

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func runApp(cmd *cobra.Command, args []string) error {
    cfgPath, _ := cmd.Flags().GetString("config")

    // 1. 初始化全部基础设施
    app, err := bootstrap.Init[appconfig.AppConfig](cfgPath, bootstrap.OptsApp)
    if err != nil {
        return err
    }
    defer app.Logger.Sync()

    // 2. 配置 Gin + 注册路由
    if app.Cfg.Log.Level != "debug" {
        gin.SetMode(gin.ReleaseMode)
    }
    engine := gin.New()
    engine.Use(gin.Recovery())
    router.Setup(engine, app.ToDeps())

    // 3. 启动服务，阻塞直到收到退出信号
    return app.ListenAndServe(engine, app.Cfg.Server.Addr())
}
```

**从 200 行减至 ~40 行（-80%）**

#### cmd/acs/main.go（~30 行）

```go
func runACS(cmd *cobra.Command, args []string) error {
    cfgPath, _ := cmd.Flags().GetString("config")

    // 1. 初始化基础设施（仅 Redis + NATS）
    app, err := bootstrap.Init[appconfig.ACSConfig](cfgPath, bootstrap.OptsACS)
    if err != nil {
        return err
    }
    defer app.Logger.Sync()

    // 2. 创建 ACS 服务
    acsServer := acs.NewACSServer(app.Cfg, acs.NewDefaultDeps(...))

    // 3. 启动服务
    return app.ListenAndServe(acsServer, app.Cfg.Server.Addr())
}
```

**从 147 行减至 ~30 行（-80%）**

#### cmd/worker/main.go（~50 行）

```go
func runWorker(cmd *cobra.Command, args []string) error {
    cfgPath, _ := cmd.Flags().GetString("config")

    // 1. 初始化全部基础设施
    app, err := bootstrap.Init[appconfig.WorkerConfig](cfgPath, bootstrap.OptsWorker)
    if err != nil {
        return err
    }
    defer app.Logger.Sync()

    // 2. 注册业务订阅者
    registerWorkerSubscribers(app)

    // 3. 等待信号并优雅关机
    return app.WaitAndShutdown()
}

func registerWorkerSubscribers(app *bootstrap.App) {
    // PM Collector, Alarm Receiver, MR Collector, Transfer Bridge, Backup, Report
    // ...约 30 行
}
```

**从 236 行减至 ~50 行（-78%）**

---

## 4. App.ToDeps() 桥接

保持现有 `router.Deps` 结构不变，`App` 提供转换方法：

```go
// internal/bootstrap/app.go

func (app *App) ToDeps() *router.Deps {
    return &router.Deps{
        PgPool:          app.PgPool,
        TsPool:          app.TsPool,
        Redis:           app.Redis,
        MinIO:           app.MinIO,
        EventBus:        app.EventBus,
        CmdQueue:        app.CmdQueue,
        CarrierRegistry: app.CarrierRegistry,
        Cfg:             app.Cfg,
        Logger:          app.Logger,
        GS:              app.GS,
        MetricsReg:      app.MetricsReg,
    }
}
```

> **注意**：`ToDeps()` 会导致 `bootstrap` 依赖 `router` 包。为避免循环依赖，可以改为在 main.go 中手动构造 Deps（仍然很短），或让 `router.Deps` 搬到 `bootstrap` 包中。

**推荐做法**：`router.Deps` 改为直接嵌入 `*bootstrap.App`，或者 `router.Setup` 直接接受 `*bootstrap.App`。

```go
// 方案 A：router.Setup 直接接受 App
func Setup(r *gin.Engine, app *bootstrap.App) { ... }

// 方案 B：main.go 中手动转换（不引入依赖）
router.Setup(engine, &router.Deps{
    PgPool: app.PgPool,
    // ...
})
```

---

## 5. 泛型配置加载

使用 Go 泛型统一三种配置类型的加载：

```go
// internal/bootstrap/infra.go

func Init[T any](cfgPath string, opts InfraOpts) (*App[T], error) {
    var cfg T
    if err := appconfig.Load(cfgPath, &cfg); err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    // ...根据 cfg 中的字段初始化各组件
}
```

由于三个 Config 结构字段名一致（`DB`、`Redis`、`NATS`、`Log` 等），可以定义接口约束：

```go
type ConfigWithLog interface {
    GetLog() logpkg.Config
}

type ConfigWithDB interface {
    GetDB() appconfig.PostgresConfig
}
```

或者更简单地使用 `reflect` / 结构体嵌入公共基础配置：

```go
// internal/appconfig/config.go

type BaseConfig struct {
    Log     LogConfig      `mapstructure:"log"`
    Redis   RedisConfig    `mapstructure:"redis"`
    NATS    NATSConfig     `mapstructure:"nats"`
    Metrics MetricsConfig  `mapstructure:"metrics"`
}

type AppConfig struct {
    BaseConfig `mapstructure:",squash"`
    Server     ServerConfig   `mapstructure:"server"`
    DB         PostgresConfig `mapstructure:"db"`
    TSDB       PostgresConfig `mapstructure:"tsdb"`
    MinIO      MinIOConfig    `mapstructure:"minio"`
    // ...
}

type ACSConfig struct {
    BaseConfig `mapstructure:",squash"`
    Server     ServerConfig   `mapstructure:"server"`
    Session    SessionConfig  `mapstructure:"session"`
    // ...
}
```

这样 `bootstrap.Init` 可以通过 `BaseConfig` 统一处理公共组件初始化。

---

## 6. 实施步骤

分 4 步渐进实施，每步都可独立编译通过：

### Step 1: 提取 BaseConfig

**文件**: `internal/appconfig/config.go`

- 定义 `BaseConfig` 嵌入到 `AppConfig`/`ACSConfig`/`WorkerConfig`
- 确保配置加载兼容（`mapstructure:",squash"`）
- 验证：三个入口编译通过，行为不变

### Step 2: 创建 bootstrap 包

**新增文件**:
- `internal/bootstrap/app.go` — App 容器
- `internal/bootstrap/infra.go` — 基础设施初始化
- `internal/bootstrap/server.go` — HTTP + 信号 + 优雅关机

验证：编译通过，暂不接入

### Step 3: 迁移 main.go

按照 app → acs → worker 顺序，逐个改造：

1. `cmd/app/main.go` — 替换为 bootstrap.Init + router.Setup + ListenAndServe
2. `cmd/acs/main.go` — 替换为 bootstrap.Init + ACS 特有逻辑
3. `cmd/worker/main.go` — 替换为 bootstrap.Init + 业务订阅注册

每改一个文件就验证编译通过。

### Step 4: 清理

- 删除三个 main.go 中已迁移到 bootstrap 的重复代码
- 运行全量测试

---

## 7. 变更影响

| 维度 | 影响 |
|------|------|
| **新增文件** | `internal/bootstrap/` 3 个文件（~200 行） |
| **修改文件** | `internal/appconfig/config.go`（加 BaseConfig 嵌入） |
| **简化文件** | 3 个 main.go（合计减少 ~400 行） |
| **净效果** | 减少 ~200 行重复代码，main.go 可读性大幅提升 |
| **router.go** | 不变（或 Deps 改为 App） |
| **业务模块** | 不受影响 |
| **运行行为** | 完全不变 |

---

## 8. 简化前后对比

### app/main.go

| | 简化前 | 简化后 |
|--|--------|--------|
| 行数 | 200 | ~40 |
| import 数 | 18 | 6 |
| 基础设施连接 | 逐一手动连接 5 个组件 | `bootstrap.Init()` 一行 |
| 关机处理 | 手动信号监听 + gs.Shutdown | `app.ListenAndServe()` 内置 |
| 可读性 | 需滚动 4 屏 | 一屏可见 |

### 三入口总计

| | 简化前 | 简化后 |
|--|--------|--------|
| 总行数 | 583 | ~180 |
| 重复代码 | ~300 行 | 0 行（全部在 bootstrap） |

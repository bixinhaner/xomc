# OMC Go 后端目录调整 — 实施计划

> 基于 `omcgo目录调整方案.md` 中推荐方案，制定逐步骤可执行的实施计划。
> 所有争议点已按推荐方案落定，不再保留备选。

---

## 决策清单（争议项已落定）

| 争议点 | 最终决策 | 理由 |
|--------|---------|------|
| C4-C7 handler/service/model/repo 扁平化 | ❌ **不做** | 破坏封装、22 个同名 Handler 冲突、15-25 人日不值得 |
| C10 global/vars.go 全局单例 | ❌ **不做** | 保持当前 DI 模式，不引入全局可变状态 |
| C2 配置结构体归属 | 保留在 `internal/` 共享 | 3 个二进制都依赖同一套 Config struct |
| omcr 大包处理 | 12 个子模块提升为 `internal/` 一级目录 | 消除不必要的层级嵌套 |
| common 大包处理 | 5 个子包各自提升为 `internal/` 一级目录 | common 语义模糊，打散后职责更清晰 |
| common/response（0 处引用） | 删除 | 死代码，无文件导入 |

---

## 实施总览

```
第一阶段（P1）: 安全重构 ————— 1 天   ——  4 个步骤，影响 ~15 文件
第二阶段（P2）: 基础设施调整 —— 2 天   ——  4 个步骤，影响 ~30 文件
第三阶段（P3）: 模块扁平化 ——— 2-3 天  ——  3 个步骤，影响 ~200 文件
                                合计 ≈ 5-6 人日
```

### 依赖关系

```
P1-S1（配置文件迁移）     ──独立──→ 可随时执行
P1-S2（路由提取）         ──独立──→ 可随时执行
P1-S3（Middleware 提升）  ──独立──→ 可随时执行
P1-S4（utils 骨架）       ──独立──→ 可随时执行

P2-S1（infra→components） ──独立──→ P1 完成后
P2-S2（config 包分离）    ──依赖 P2-S1（logger/tracer 导入路径变了）
P2-S3（global/ 常量+错误码）──独立──→ P1 完成后
P2-S4（清理 common/response）──独立

P3-S1（omcr 解散）        ──独立──→ P2 完成后
P3-S2（common 解散）      ──依赖 P3-S1（避免并行改同一文件）
P3-S3（main.go 精简）     ──依赖 P3-S1 + P3-S2
```

---

## 第一阶段（P1）：安全重构

> 零风险变更，不改变任何运行时行为。每步完成后可独立编译验证。

---

### P1-S1：配置文件按环境隔离

**目标**：`configs/*.yaml` → `cmd/{app,acs,worker}/etc/`

#### 步骤

```bash
# 1. 创建目录
mkdir -p cmd/app/etc cmd/acs/etc cmd/worker/etc

# 2. 复制配置文件（保留原件，确认无误后再删）
cp configs/app.yaml    cmd/app/etc/config.dev.yaml
cp configs/acs.yaml    cmd/acs/etc/config.dev.yaml
cp configs/worker.yaml cmd/worker/etc/config.dev.yaml

# 3. 创建 test/prod 骨架（从 dev 复制，后续按需调整）
cp cmd/app/etc/config.dev.yaml    cmd/app/etc/config.test.yaml
cp cmd/app/etc/config.dev.yaml    cmd/app/etc/config.prod.yaml
cp cmd/acs/etc/config.dev.yaml    cmd/acs/etc/config.test.yaml
cp cmd/acs/etc/config.dev.yaml    cmd/acs/etc/config.prod.yaml
cp cmd/worker/etc/config.dev.yaml cmd/worker/etc/config.test.yaml
cp cmd/worker/etc/config.dev.yaml cmd/worker/etc/config.prod.yaml
```

#### 代码修改

| 文件 | 修改 |
|------|------|
| `cmd/app/main.go:70` | `"configs/app.yaml"` → `"cmd/app/etc/config.dev.yaml"` |
| `cmd/acs/main.go` | `"configs/acs.yaml"` → `"cmd/acs/etc/config.dev.yaml"` |
| `cmd/worker/main.go` | `"configs/worker.yaml"` → `"cmd/worker/etc/config.dev.yaml"` |

#### 部署文件修改

| 文件 | 修改 |
|------|------|
| `deployments/docker/Dockerfile.app` | `COPY configs/app.yaml` → `COPY cmd/app/etc/config.prod.yaml /etc/omcgo/app.yaml` |
| `deployments/docker/Dockerfile.acs` | `COPY configs/acs.yaml` → `COPY cmd/acs/etc/config.prod.yaml /etc/omcgo/acs.yaml` |
| `deployments/docker/Dockerfile.worker` | `COPY configs/worker.yaml` → `COPY cmd/worker/etc/config.prod.yaml /etc/omcgo/worker.yaml` |
| `deployments/k8s/configmap.yaml` | 配置内容不变，注释说明源文件路径更新 |

#### 清理

```bash
# 确认一切正常后
rm configs/app.yaml configs/acs.yaml configs/worker.yaml
# 保留 configs/acs-stress.yaml（压测专用，不参与环境隔��）
```

#### 验证

```bash
go build ./cmd/app && go build ./cmd/acs && go build ./cmd/worker
```

---

### P1-S2：路由注册提取

**目标**：将 `main.go` 中 ~300 行的 DI 组装 + 路由注册逻辑提取到 `cmd/app/router/`

#### 新建文件

**`cmd/app/router/deps.go`** — 依赖容器结构体

```go
package router

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/minio/minio-go/v7"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"

    "github.com/omcgo/omcgo/internal/acs/cmdqueue"
    "github.com/omcgo/omcgo/internal/common/event"
    "github.com/omcgo/omcgo/internal/carrier"
    "github.com/omcgo/omcgo/internal/config"
)

// Deps 聚合所有基础设施依赖，由 main.go 初始化后传入
type Deps struct {
    PgPool          *pgxpool.Pool
    TsPool          *pgxpool.Pool
    Redis           *redis.Client
    MinIO           *minio.Client
    EventBus        event.EventBus
    CmdQueue        *cmdqueue.RedisCommandQueue
    CarrierRegistry *carrier.Registry
    Cfg             *config.AppConfig
    Logger          *zap.Logger
}
```

**`cmd/app/router/router.go`** — 路由注册主入口

```go
package router

import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
)

// Setup 创建所有 repository/service/handler 并注册路由
// 返回需要注册到 GracefulShutdown 的后台组件
func Setup(r *gin.Engine, deps *Deps, metricsReg *prometheus.Registry) *Components {
    // 1. 创建所有 repository
    // 2. 创建所有 service
    // 3. 订阅 EventBus
    // 4. 注册路由
    // ... 从 main.go 第 141-446 行搬入
}

// Components 包含需要在 main 中注册 shutdown 的后台组件
type Components struct {
    HeartbeatMonitor interface{ Stop() }
    DataModelRegistry interface{ Stop() }
    // ... 其他需要 graceful shutdown 的组件
}
```

#### main.go 精简后结构

```go
func runApp(cmd *cobra.Command, args []string) error {
    // 1. Load config                    (~5 行)
    // 2. Init logger                    (~5 行)
    // 3. Graceful shutdown setup        (~3 行)
    // 4. Connect infra (PG/Redis/TS/MinIO/NATS) (~30 行)
    // 5. Create EventBus                (~3 行)
    // 6. Setup Gin + middleware          (~15 行)
    // 7. 调用 router.Setup()            (~5 行)
    // 8. Register shutdown hooks        (~10 行)
    // 9. Start HTTP/Metrics servers     (~20 行)
    // 10. Wait signal + shutdown        (~15 行)
    // 总计 ~110 行（当前 502 行 → 缩减 78%）
}
```

#### 验证

```bash
go build ./cmd/app
go test ./cmd/app/router/...
# E2E 测试确认所有路由正常
```

---

### P1-S3：Middleware 提升到 internal 一级

**目标**：`internal/common/middleware/` → `internal/middleware/`

#### 步骤

```bash
# 1. 移动目录
mv internal/common/middleware internal/middleware

# 2. 更新 package 声明（不需要，已经是 package middleware）

# 3. 全局替换导入路径
find . -name "*.go" -exec sed -i '' \
  's|"github.com/omcgo/omcgo/internal/common/middleware"|"github.com/omcgo/omcgo/internal/middleware"|g' {} +
```

#### 受影响文件（仅 3 个）

| 文件 | 变更 |
|------|------|
| `cmd/app/main.go` | 导入路径更新 |
| `cmd/acs/main.go` | 若有使用 |
| `cmd/worker/main.go` | 若有使用 |

#### 验证

```bash
go build ./...
# 确认无残留
grep -rn 'internal/common/middleware' --include="*.go" .
```

---

### P1-S4：创建 internal/utils/ 骨架

**目标**：预留工具函数目录

```bash
mkdir -p internal/utils
```

创建 `internal/utils/utils.go`：
```go
// Package utils provides internal utility functions shared across modules.
package utils
```

> 暂不迁移内容，后续各模块重构时按需添加。

#### P1 完成检查

```bash
go build ./...
go test ./... -count=1
grep -rn 'internal/common/middleware' --include="*.go" .   # 应返回 0
grep -rn '"configs/' --include="*.go" .                     # 应返回 0（除压测配置）
```

---

## 第二阶段（P2）：基础设施调整

> 重命名基础设施层 + 分离配置包 + 引入全局常量。涉及更多导入路径替换。

---

### P2-S1：Infra → Components 重命名与重组

**目标**：
```
internal/infra/           → internal/components/
internal/infra/db/        → internal/components/postgres/
internal/infra/cache/     → internal/components/redis/
internal/infra/mq/        → internal/components/nats/
internal/infra/storage/   → internal/components/minio/
internal/infra/logger.go  → internal/components/logger/logger.go
internal/infra/metrics.go → internal/components/monitor/metrics.go
```

非组件文件保留在 `internal/components/` 根目录：
```
internal/infra/health.go    → internal/components/health.go
internal/infra/sysinfo.go   → internal/components/sysinfo.go
internal/infra/tracer.go    → internal/components/tracer.go
internal/infra/shutdown.go  → internal/components/shutdown.go
```

#### 详细步骤

```bash
# 1. 创建新目录结构
mkdir -p internal/components/{postgres,redis,nats,minio,logger,monitor}

# 2. 移动子包（重命名）
mv internal/infra/db/postgres.go     internal/components/postgres/postgres.go
mv internal/infra/db/timescale.go    internal/components/postgres/timescale.go
mv internal/infra/cache/redis.go     internal/components/redis/redis.go
mv internal/infra/mq/nats.go         internal/components/nats/nats.go
mv internal/infra/storage/minio.go   internal/components/minio/minio.go

# 3. 独立文件提升为子包
mv internal/infra/logger.go          internal/components/logger/logger.go
mv internal/infra/metrics.go         internal/components/monitor/metrics.go

# 4. 非组件文件移动（保持在 components 根）
mv internal/infra/health.go          internal/components/health.go
mv internal/infra/health_test.go     internal/components/health_test.go
mv internal/infra/sysinfo.go         internal/components/sysinfo.go
mv internal/infra/shutdown.go        internal/components/shutdown.go
mv internal/infra/shutdown_test.go   internal/components/shutdown_test.go
mv internal/infra/tracer.go          internal/components/tracer.go

# 5. 删除旧目录
rm -rf internal/infra/
```

#### Package 声明更新

| 文件 | 旧 package | 新 package |
|------|-----------|-----------|
| `postgres/postgres.go` | `package db` | `package postgres` |
| `postgres/timescale.go` | `package db` | `package postgres` |
| `redis/redis.go` | `package cache` | `package redis` |
| `nats/nats.go` | `package mq` | `package nats` |
| `minio/minio.go` | `package storage` | `package minio` |
| `logger/logger.go` | `package infra` | `package logger` |
| `monitor/metrics.go` | `package infra` | `package monitor` |
| `health.go` | `package infra` | `package components` |
| `sysinfo.go` | `package infra` | `package components` |
| `tracer.go` | `package infra` | `package components` |
| `shutdown.go` | `package infra` | `package components` |

#### 导入路径替换映射

| 旧路径 | 新路径 | 受影响文件数 |
|--------|--------|------------|
| `internal/infra/db` | `internal/components/postgres` | 2（cmd/app, cmd/worker） |
| `internal/infra/cache` | `internal/components/redis` | 3（cmd/*） |
| `internal/infra/mq` | `internal/components/nats` | 3（cmd/*） |
| `internal/infra/storage` | `internal/components/minio` | 2（cmd/app, cmd/worker） |
| `internal/infra"` (根包) | `internal/components"` | 3（cmd/*） |

#### 函数名更新

包名变化导致调用方式变化：

| 旧调用 | 新调用 | 涉及文件 |
|--------|--------|---------|
| `db.NewPostgresPool(...)` | `postgres.NewPostgresPool(...)` | cmd/app/main.go, cmd/worker/main.go |
| `db.NewTimescalePool(...)` | `postgres.NewTimescalePool(...)` | cmd/app/main.go, cmd/worker/main.go |
| `cache.NewRedisClient(...)` | `redisclient.NewRedisClient(...)` | cmd/*/main.go（注意：`redis` 与 go-redis 包名冲突，改用 `redisclient` 别名或包名用 `rediscomp`） |
| `mq.NewNATSClient(...)` | `natscomp.NewNATSClient(...)` | cmd/*/main.go（同理避免与 nats 包名冲突） |
| `storage.NewMinIOClient(...)` | `miniocomp.NewMinIOClient(...)` | cmd/app/main.go, cmd/worker/main.go |
| `infra.NewLogger(...)` | `logger.NewLogger(...)` | cmd/*/main.go |
| `infra.NewGracefulShutdown(...)` | `components.NewGracefulShutdown(...)` | cmd/*/main.go |
| `infra.NewSystemInfoHandler(...)` | `components.NewSystemInfoHandler(...)` | cmd/app/main.go |
| `infra.NewTracerProvider(...)` | `components.NewTracerProvider(...)` | cmd/*/main.go（如有） |

> **注意**：`redis`, `nats`, `minio` 作为包名与第三方库冲突。解决方案有��种：
> - **方案 A**：子包名加后缀 → `rediscomp/`, `natscomp/`, `miniocomp/`
> - **方案 B**：保持原名，导入时用别名 → `import rediscomp "internal/components/redis"`
>
> **推荐方案 B**（包名保持干净，别名仅在 cmd/ 入口使用）

#### 验证

```bash
go build ./...
grep -rn 'internal/infra' --include="*.go" .   # 应返回 0
```

---

### P2-S2：Config 包分离

**目标**：将配置结构体 + 加载逻辑从 `internal/config/` 提取到独立位置，让 `internal/config/` 纯粹作为 F02 配置管理业务域。

**最终决策**：配置结构体保留在 `internal/` 下共享（3 个二进制都需要），但从 F02 业务域分开。

```
internal/config/config.go              → internal/appconfig/config.go
internal/config/datamodel/             → 保持不变
internal/config/template/              → 保持不变
internal/config/baseline/              → 保持不变
internal/config/sync_handler.go        → 保持不变
```

#### 步骤

```bash
# 1. 创建新包
mkdir -p internal/appconfig

# 2. 移动配置文件
mv internal/config/config.go internal/appconfig/config.go

# 3. 更新 package 声明
# config.go: package config → package appconfig
```

#### 导入路径替换

| 旧导入 | 新导入 | 受影响文件 |
|--------|--------|----------|
| `"github.com/omcgo/omcgo/internal/config"` (用于 Config struct) | `"github.com/omcgo/omcgo/internal/appconfig"` | 14 个文件 |

**注意区分**：部分文件同时导入 `internal/config`（用于 F02 SyncHandler）和 Config struct。替换时需确认每处 `config.` 引用是指 AppConfig 还是 F02 业务。

具体受影响文件清单：

| 文件 | 使用的类型 | 需改为 |
|------|-----------|--------|
| `cmd/app/main.go` | `config.AppConfig`, `config.Load()` | `appconfig.AppConfig`, `appconfig.Load()` |
| `cmd/acs/main.go` | `config.ACSConfig`, `config.Load()` | `appconfig.ACSConfig`, `appconfig.Load()` |
| `cmd/worker/main.go` | `config.WorkerConfig`, `config.Load()` | `appconfig.WorkerConfig`, `appconfig.Load()` |
| `internal/acs/server.go` | `config.ACSConfig` 字段引用 | `appconfig.ACSConfig` |
| `internal/components/postgres/postgres.go` | `config.PostgresConfig` | `appconfig.PostgresConfig` |
| `internal/components/postgres/timescale.go` | `config.PostgresConfig` | `appconfig.PostgresConfig` |
| `internal/components/redis/redis.go` | `config.RedisConfig` | `appconfig.RedisConfig` |
| `internal/components/nats/nats.go` | `config.NATSConfig` | `appconfig.NATSConfig` |
| `internal/components/minio/minio.go` | `config.MinIOConfig` | `appconfig.MinIOConfig` |
| `internal/components/logger/logger.go` | `config.LogConfig` | `appconfig.LogConfig` |
| `internal/components/tracer.go` | `config.TracerConfig` | `appconfig.TracerConfig` |
| `internal/nedirect/server.go` | `config.NEDirectConfig` | `appconfig.NEDirectConfig` |
| `internal/northbound/push/engine.go` | `config.PushTargetConfig` | `appconfig.PushTargetConfig` |
| `internal/northbound/push/engine_test.go` | 同上 | 同上 |

#### 验证

```bash
go build ./...
grep -rn '"github.com/omcgo/omcgo/internal/config"' --include="*.go" .
# 应仅返回 internal/config/ 下自身文件 + sync_handler.go
```

---

### P2-S3：新建 global/ — 常量与错误码

**目标**：将纯值常量和错误码提取到 `global/`，不含任何框架依赖。

#### 新建文件

**`global/consts.go`** — 从 `internal/common/model/constants.go` 提取

```go
package global

// CarrierCode 运营商编码
type CarrierCode string

const (
    CarrierCMCC CarrierCode = "cmcc"
    CarrierCTCC CarrierCode = "ctcc"
    CarrierCUCC CarrierCode = "cucc"
)

func (c CarrierCode) IsValid() bool { ... }
func ValidCarriers() []CarrierCode { ... }

// Technology 接入技术
type Technology string

const (
    TechLTE Technology = "lte"
    TechNR  Technology = "nr"
)

func (t Technology) IsValid() bool { ... }

// DeviceStatus 设备状态
type DeviceStatus string

const ( ... )   // 7 个状态常量

// AlarmSeverity 告警级别
type AlarmSeverity int

const ( ... )   // 4 个级别

// AlarmStatus 告警状态
type AlarmStatus string

const ( ... )   // 3 个状态

// ParameterType 参数类型
type ParameterType string

const ( ... )   // 6 个类型

// DataModelScope 数据模型范围
type DataModelScope string

const ( ... )   // 3 个范围
```

**`global/errors.go`** — 从 `internal/common/errors/codes.go` 提取纯错误码

```go
package global

// 错误码定义（纯数值常量，不含 gin/HTTP 依赖）
// 按领域划分范围：1000-16999

// Device Management (1000-1999)
const (
    ErrDeviceNotFound   = 1001
    ErrDeviceDuplicate  = 1002
    // ...
)

// Data Model / Configuration (2000-2999)
const ( ... )

// ... 全部 63 个错误码常量
```

#### 迁移策略

**关键决策**：`internal/common/errors/errors.go` 中的 `BusinessError` struct 和 `AbortWithError(gin.Context)` 函数**不移动**到 global/，因为它们依赖 gin 框架。保留在 `internal/errors/`（P3-S2 时从 common 提升）。

**分步执行**：

```bash
# 1. 创建 global 目录
mkdir -p global

# 2. 从 constants.go 提取类型和常量到 global/consts.go
# 3. 从 codes.go 提取错误码常量到 global/errors.go
# 4. 修改 internal/common/model/constants.go — 改为 re-export：
#    type CarrierCode = global.CarrierCode  // 类型别名，��破坏
# 5. 修改 internal/common/errors/codes.go — 改为 re-export：
#    const ErrDeviceNotFound = global.ErrDeviceNotFound
```

**过渡阶段**：通过类型别名 + 常量 re-export，让现有 185+62 个文件的导入路径**暂时不变**。在 P3-S2 解散 common 时统一替换。

#### 验证

```bash
go build ./...
go test ./... -count=1
```

---

### P2-S4：清理 common/response

**发现**：`internal/common/response/` 的 2 个文件（response.go + response_test.go）**没有任何外部文件导入**（0 处引用），属于死代码。

```bash
rm -rf internal/common/response/
```

#### 验证

```bash
go build ./...
```

---

#### P2 完成检查

```bash
go build ./...
go test ./... -count=1
grep -rn 'internal/infra' --include="*.go" .       # 应返回 0
grep -rn 'internal/common/response' --include="*.go" . # 应返回 0
grep -rn 'internal/common/middleware' --include="*.go" . # 应返回 0（P1 已处理）
```

---

## 第三阶段（P3）：模块扁平化

> 核心变更：解散 `internal/omcr/`（12 个子模块提升）和 `internal/common/`（4 个子包提升）。
> 影响范围最大，建议**冻结主线**集中执行。

---

### P3-S1：解散 omcr/ — 12 个子模块提升

**目标**：
```
internal/omcr/admin/       → internal/admin/
internal/omcr/device/      → internal/device/
internal/omcr/topology/    → internal/topology/
internal/omcr/software/    → internal/software/
internal/omcr/backup/      → internal/backup/
internal/omcr/dashboard/   → internal/dashboard/
internal/omcr/ops/         → internal/ops/
internal/omcr/report/      → internal/report/
internal/omcr/mml/         → internal/mml/
internal/omcr/filemanager/ → internal/filemanager/
internal/omcr/syslog/      → internal/syslog/
internal/omcr/license/     → internal/license/
```

#### 交叉依赖分析（已确认）

| 模块 | 依赖 | 说明 |
|------|------|------|
| dashboard | device, topology, admin | 仪表盘聚合数据 |
| software | device | 升级需要设备信息 |
| 其余 10 个 | 无交叉依赖 | 完全独立 |

**结论**：移动后交叉导入路径自然从 `internal/omcr/device` 变为 `internal/device`，不会产生循环依赖。

#### 执行步骤

```bash
# 1. 移动所有 12 个子模块
for mod in admin device topology software backup dashboard ops report mml filemanager syslog license; do
    mv internal/omcr/$mod internal/$mod
done

# 2. 删除空目录
rm -rf internal/omcr/

# 3. 全局替换导入路径（12 次替换）
for mod in admin device topology software backup dashboard ops report mml filemanager syslog license; do
    find . -name "*.go" -exec sed -i '' \
      "s|github.com/omcgo/omcgo/internal/omcr/$mod|github.com/omcgo/omcgo/internal/$mod|g" {} +
done
```

#### 受影响文件统计

| 旧导入路径 | 导入该模块的文件数 |
|-----------|-------------------|
| `internal/omcr/device` | ~6（main.go, dashboard, software, informHandler...） |
| `internal/omcr/admin` | ~4（main.go, dashboard, middleware/auth...） |
| `internal/omcr/topology` | ~3（main.go, dashboard...） |
| `internal/omcr/software` | ~2（main.go...） |
| `internal/omcr/backup` | ~2 |
| `internal/omcr/dashboard` | ~2 |
| `internal/omcr/ops` | ~2 |
| `internal/omcr/report` | ~2 |
| `internal/omcr/mml` | ~2 |
| `internal/omcr/filemanager` | ~2 |
| `internal/omcr/syslog` | ~2 |
| `internal/omcr/license` | ~2 |
| **合计** | **~31 个文件** |

> package 声明不需要改（都已经是 `package device`, `package admin` 等）。

#### 验证

```bash
go build ./...
grep -rn 'internal/omcr/' --include="*.go" .   # 应返回 0
```

---

### P3-S2：解散 common/ — 子包提升 + 全局路径切换

**目标**：
```
internal/common/model/     → internal/model/
internal/common/errors/    → internal/errors/
internal/common/event/     → internal/event/
internal/common/response/  → 已在 P2-S4 删除
internal/common/middleware/ → 已在 P1-S3 移动
```

完成后删除 `internal/common/` 目录。

#### 阶段 A：移动目录

```bash
# 1. 移动 3 个子包
mv internal/common/model   internal/model
mv internal/common/errors  internal/errors
mv internal/common/event   internal/event

# 2. 删除空目录
rm -rf internal/common/
```

#### 阶段 B：全局替换导入路径

```bash
# model（185 个文件 — 最大范围替换）
find . -name "*.go" -exec sed -i '' \
  's|"github.com/omcgo/omcgo/internal/common/model"|"github.com/omcgo/omcgo/internal/model"|g' {} +

# errors（62 个文件）
find . -name "*.go" -exec sed -i '' \
  's|github.com/omcgo/omcgo/internal/common/errors|github.com/omcgo/omcgo/internal/errors|g' {} +

# event（22 个文件）
find . -name "*.go" -exec sed -i '' \
  's|"github.com/omcgo/omcgo/internal/common/event"|"github.com/omcgo/omcgo/internal/event"|g' {} +
```

**注意**：errors 包通常使用别名导入：
```go
commonerrors "github.com/omcgo/omcgo/internal/common/errors"
```
替换后变为：
```go
commonerrors "github.com/omcgo/omcgo/internal/errors"
```
别名 `commonerrors` 可保留（避免与标准库 `errors` 冲突），或统一改为 `bizerrors`。

#### 阶段 C：更新 global/ re-export（如果 P2-S3 使用了 re-export 过渡）

P2-S3 中 `internal/common/model/constants.go` 内的 re-export 现在位于 `internal/model/constants.go`。
此时可以选择：
- **保持 re-export**：新代���直接用 `global.CarrierCode`，旧代码通过 `model.CarrierCode`（类型别名）兼容
- **一步到位**：全局替换 `model.CarrierCMCC` → `global.CarrierCMCC` 等（涉及 185 个文件）

**推荐**：保持 re-export 过渡，后续新代码直接用 `global`，旧代码自然迁移。

#### 受影响文件总计

| 替换项 | 文件数 |
|--------|--------|
| `internal/common/model` → `internal/model` | 185 |
| `internal/common/errors` → `internal/errors` | 62 |
| `internal/common/event` → `internal/event` | 22 |
| **合计（有重叠）** | **~200 个唯一文件** |

#### 验证

```bash
go build ./...
go test ./... -count=1
grep -rn 'internal/common/' --include="*.go" .   # 应返回 0
```

---

### P3-S3：main.go 最终精简

P1-S2 已将路由注册提取到 `cmd/app/router/`。经过 P2、P3 的路径变更，现在统一更新 main.go 及 router 中的所有导入路径。

#### 最终 main.go 导入表

```go
import (
    // 标准库
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    // 第三方
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/spf13/cobra"
    "go.uber.org/zap"

    // 内部 — 配置
    "github.com/omcgo/omcgo/internal/appconfig"

    // 内部 — 基础设施
    "github.com/omcgo/omcgo/internal/components"
    "github.com/omcgo/omcgo/internal/components/logger"
    "github.com/omcgo/omcgo/internal/components/postgres"
    rediscomp "github.com/omcgo/omcgo/internal/components/redis"
    natscomp "github.com/omcgo/omcgo/internal/components/nats"
    miniocomp "github.com/omcgo/omcgo/internal/components/minio"

    // 内部 — 通用
    "github.com/omcgo/omcgo/internal/event"
    "github.com/omcgo/omcgo/internal/middleware"
    bizerrors "github.com/omcgo/omcgo/internal/errors"

    // 内部 — 路由
    "github.com/omcgo/omcgo/cmd/app/router"
)
```

#### 验证

```bash
go build ./cmd/app
go build ./cmd/acs
go build ./cmd/worker
go test ./... -count=1
```

---

## 最终目标结构

```
omcgo/
├── cmd/
│   ├── app/
│   │   ├── main.go                 # ~110 行（基础设施初始化 + 调用 router.Setup）
│   │   ├── etc/                    # 配置文件
│   │   │   ├── config.dev.yaml
│   │   │   ├── config.test.yaml
│   │   │   └── config.prod.yaml
│   │   └── router/                 # 路由注册 + DI 容器
│   │       ├── deps.go
│   │       └── router.go
│   ├── acs/
│   │   ├── main.go
│   │   └── etc/
│   │       ├── config.dev.yaml
│   │       ├── config.test.yaml
│   │       └── config.prod.yaml
│   ├── worker/
│   │   ├── main.go
│   │   └── etc/
│   │       ├── config.dev.yaml
│   │       ├── config.test.yaml
│   │       └── config.prod.yaml
│   ├── migrate/
│   └── omcctl/
│
├── internal/
│   │── device/                     # ← omcr/device (handler+service+repo+model 自包含)
│   │── admin/                      # ← omcr/admin
│   │── topology/                   # ← omcr/topology
│   │── software/                   # ← omcr/software
│   │── backup/                     # ← omcr/backup
│   │── dashboard/                  # ← omcr/dashboard
│   │── ops/                        # ← omcr/ops
│   │── report/                     # ← omcr/report
│   │── mml/                        # ← omcr/mml
│   │── filemanager/                # ← omcr/filemanager
│   │── syslog/                     # ← omcr/syslog
│   │── license/                    # ← omcr/license
│   │── alarm/                      # 保持
│   │── pm/                         # 保持
│   │── mr/                         # 保持
│   │── config/                     # F02 业务域 (datamodel/template/baseline/sync)
│   │── provision/                  # 保持
│   │── acs/                        # 保持
│   │── carrier/                    # 保持
│   │── northbound/                 # 保持
│   │── interop/                    # 保持
│   │── nedirect/                   # 保持
│   │── appconfig/                  # ← config/config.go（配置结构体+加载）
│   │── model/                      # ← common/model（共享领域类型）
│   │── errors/                     # ← common/errors（业务错误+gin 集成）
│   │── event/                      # ← common/event（EventBus 抽象）
│   │── middleware/                  # ← common/middleware
│   │── components/                 # ← infra/
│   │   ├── postgres/               # ← infra/db
│   │   ├── redis/                  # ← infra/cache
│   │   ├── nats/                   # ← infra/mq
│   │   ├── minio/                  # ← infra/storage
│   │   ├── logger/                 # ← infra/logger.go
│   │   ├── monitor/                # ← infra/metrics.go
│   │   ├── health.go               # ← infra/health.go
│   │   ├── sysinfo.go              # ← infra/sysinfo.go
│   │   ├── tracer.go               # ← infra/tracer.go
│   │   └── shutdown.go             # ← infra/shutdown.go
│   └── utils/
│
├── global/
│   ├── consts.go                   # 运营商/设备/告警等全局常量
│   └── errors.go                   # 63 个错误码（纯数值）
│
├── pkg/                            # 保持不变
│   ├── soap/
│   ├── tr069/
│   └── xmlutil/
│
├── migrations/                     # 保持不变
├── deployments/                    # Dockerfile/K8s 路径更新
├── doc/                            # 路径引用更新
├── scripts/                        # 保持不变
├── test/                           # 保持不变
├── Makefile                        # 路径更新
└── go.mod                          # 不变
```

---

## 文档更新清单

每阶段完成后同步更新：

| 文档 | 更新时机 | 更新内容 |
|------|---------|---------|
| `CLAUDE.md` | P3 完成后 | 模块路径、目录结构描述全面重写 |
| `doc/architecture/backend-design.md` | P3 完成后 | 架构图、模块说明 |
| `doc/detailed-design/01-19-*.md` | P3 完成后 | 各功能域的模块路径引用 |
| `Makefile` | P1-S1 | build 路径 |
| `deployments/docker/Dockerfile.*` | P1-S1 | COPY 配置路径 |
| `deployments/k8s/configmap.yaml` | P1-S1 | 注释更新 |
| `.golangci.yml` | 无需更新 | 无路径硬编码 |
| `scripts/e2e_verify.sh` | 无需更新 | 走 HTTP 接口，无源码路径 |

---

## 每阶段验证检查清单

```bash
# ═══ 编译（必须通过）═══
go build ./...

# ═══ 单元测试 ═══
go test ./... -count=1 -timeout=300s

# ═══ 残留旧路径搜索 ═══
echo "--- P1 检查 ---"
grep -rn 'internal/common/middleware' --include="*.go" .

echo "--- P2 检查 ---"
grep -rn 'internal/infra' --include="*.go" .
grep -rn 'internal/common/response' --include="*.go" .

echo "--- P3 检查 ---"
grep -rn 'internal/omcr/' --include="*.go" .
grep -rn 'internal/common/' --include="*.go" .

# ═══ E2E 测试（需启动服务）═══
# 先重新编译
make build-app
# 启动服务
bin/omcgo-app --config cmd/app/etc/config.dev.yaml &
# 运行 331 条测试
bash ./scripts/e2e_verify.sh http://localhost:8080

# ═══ Lint ═══
golangci-lint run ./...
```

---

## 回滚策略

每阶段以独立 Git commit 提交，便于按阶段回滚：

```
commit P1-S1: "refactor: 配置文件按环境隔离到 cmd/*/etc/"
commit P1-S2: "refactor: 路由注册提取到 cmd/app/router/"
commit P1-S3: "refactor: middleware 提升到 internal/middleware/"
commit P1-S4: "refactor: 创建 internal/utils/ 骨架"
commit P2-S1: "refactor: infra/ 重命名为 components/"
commit P2-S2: "refactor: config 结构体分离到 internal/appconfig/"
commit P2-S3: "refactor: 全局常量和错误码提取到 global/"
commit P2-S4: "refactor: 清理未使用的 common/response/"
commit P3-S1: "refactor: 解散 omcr/，12 个模块提升为 internal 一级目录"
commit P3-S2: "refactor: 解散 common/，子包提升为 internal 一级目录"
commit P3-S3: "refactor: main.go 导入路径统一更新"
commit DOCS:  "docs: 更新所有文档中的模块路径引用"
```

如某阶段出问题：
```bash
git revert <commit-hash>   # 单步骤回退
# 或
git revert <P3-S3>..<P3-S1>  # 整阶段回退
```

---

## 时间线建议

```
Day 1 上午：P1（全部 4 步）→ 编译验证 → commit × 4
Day 1 下午：P2-S1（infra→components）→ 编译验证 → commit
Day 2 上午：P2-S2 + P2-S3 + P2-S4 → 编译验证 → commit × 3
Day 2 下午：P3-S1（omcr 解散）→ 编译验证 → commit
Day 3 上午：P3-S2（common 解散）→ 编译 + 全量单元测试 → commit
Day 3 下午：P3-S3 + 文档更新 + E2E 全量测试 → commit × 2
Day 4：    缓冲日 — 修复遗漏问题，更新 CI/CD 配置
```

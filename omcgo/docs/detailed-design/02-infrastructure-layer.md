# DD-02: 基础设施层

> 关联功能域：全部
> 关联 backend-design.md 章节：第三章（技术栈��型）、第���章（数据存储策略）
> 实施阶段：Phase 1（基础建设）
> 依赖文档：DD-01

---

## 1. 概述

### 1.1 模块定位

���础设施层（`internal/infra/`）封装所有外部存储组件的连接管理，为上层业务模块提供统一的数据访问接口。

### 1.2 核心职责

- PostgreSQL/TimescaleDB 连接池管理
- Redis Cluster/Standalone 连接管理
- NATS JetStream 连接与 Stream 管理
- MinIO 对象存储连接与 Bucket 管理
- 统一健康检查
- 优雅关闭编排

### 1.3 与其他模块的交互关系

所有业务模块（acs、config、pm、alarm 等）通过基础设施层访问底层存储，不直接创建连接。

---

## 2. 接口设计

### 2.1 PostgreSQL — `internal/infra/db/postgres.go`

```go
// PostgresConfig 数据库连接配置
type PostgresConfig struct {
    DSN             string        `yaml:"dsn"`
    MaxConns        int32         `yaml:"max_conns"`
    MinConns        int32         `yaml:"min_conns"`
    MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
    MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
    HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// NewPostgresPool 创建 PostgreSQL 连接池
func NewPostgresPool(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error)

// 健康检查
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error
```

### 2.2 TimescaleDB — `internal/infra/db/timescale.go`

```go
// TimescaleConfig 时序数据库配置（与 PostgresConfig 结构相同，使用独立连接池）
type TimescaleConfig = PostgresConfig

// NewTimescalePool 创建 TimescaleDB 连接池
func NewTimescalePool(ctx context.Context, cfg TimescaleConfig) (*pgxpool.Pool, error)

// EnsureExtension 确保 TimescaleDB 扩展已安装
func EnsureExtension(ctx context.Context, pool *pgxpool.Pool) error
```

### 2.3 Redis — `internal/infra/cache/redis.go`

```go
// RedisConfig Redis 连接配置
type RedisConfig struct {
    Addrs    []string `yaml:"addrs"`    // 单节点或 Cluster 节点列表
    Password string   `yaml:"password"`
    DB       int      `yaml:"db"`       // Standalone 模式使用
    PoolSize int      `yaml:"pool_size"`
}

// NewRedisClient 创建 Redis 客户端（自动判断 Cluster/Standalone）
func NewRedisClient(cfg RedisConfig) (redis.UniversalClient, error)

func HealthCheck(ctx context.Context, client redis.UniversalClient) error
```

### 2.4 NATS — `internal/infra/mq/nats.go`

```go
// NATSConfig NATS 连接配置
type NATSConfig struct {
    URL         string `yaml:"url"`
    ClusterID   string `yaml:"cluster_id"`
    MaxReconnect int   `yaml:"max_reconnect"`
    ReconnectWait time.Duration `yaml:"reconnect_wait"`
}

// NATSClient NATS 连接封装
type NATSClient struct {
    Conn *nats.Conn
    JS   nats.JetStreamContext
}

// NewNATSClient 创建 NATS 连接 + JetStream context
func NewNATSClient(cfg NATSConfig) (*NATSClient, error)

// EnsureStreams 确保所需的 JetStream Stream 已创建
func (c *NATSClient) EnsureStreams(ctx context.Context) error

func (c *NATSClient) HealthCheck() error
func (c *NATSClient) Close()
```

**JetStream Streams 定义**：

| Stream 名称 | Subjects | 保留策略 | 副本数 |
|-------------|----------|---------|-------|
| `DEVICE` | `device.>` | WorkQueue | 3 |
| `COMMAND` | `command.>` | WorkQueue | 3 |
| `PM` | `pm.>` | WorkQueue | 3 |
| `MR` | `mr.>` | WorkQueue | 3 |
| `ALARM` | `alarm.>` | WorkQueue | 3 |
| `OSS` | `oss.>` | WorkQueue | 3 |

### 2.5 MinIO — `internal/infra/storage/minio.go`

```go
// MinIOConfig MinIO 连接配置
type MinIOConfig struct {
    Endpoint  string `yaml:"endpoint"`
    AccessKey string `yaml:"access_key"`
    SecretKey string `yaml:"secret_key"`
    UseSSL    bool   `yaml:"use_ssl"`
    Buckets   BucketConfig `yaml:"buckets"`
}

type BucketConfig struct {
    PMFiles      string `yaml:"pm_files"`
    MRFiles      string `yaml:"mr_files"`
    Firmware     string `yaml:"firmware"`
    ConfigBackup string `yaml:"config_backup"`
    Logs         string `yaml:"logs"`
}

// NewMinIOClient 创建 MinIO 客户端
func NewMinIOClient(cfg MinIOConfig) (*minio.Client, error)

// EnsureBuckets 确保所有 Bucket 存在
func EnsureBuckets(ctx context.Context, client *minio.Client, cfg BucketConfig) error

func HealthCheck(ctx context.Context, client *minio.Client) error
```

### 2.6 统一健康检查 — `internal/infra/health.go`

```go
// ComponentHealth 组件健康状态
type ComponentHealth struct {
    Name    string `json:"name"`
    Status  string `json:"status"` // "healthy", "unhealthy", "degraded"
    Latency string `json:"latency"`
    Error   string `json:"error,omitempty"`
}

// HealthChecker 统一健康检查
type HealthChecker struct {
    components []func(ctx context.Context) ComponentHealth
}

func (h *HealthChecker) Register(name string, check func(ctx context.Context) error)
func (h *HealthChecker) CheckAll(ctx context.Context) []ComponentHealth
func (h *HealthChecker) IsHealthy(ctx context.Context) bool
```

### 2.7 优雅关闭 — `internal/infra/shutdown.go`

```go
// GracefulShutdown 优雅关闭编排器
type GracefulShutdown struct {
    timeout time.Duration
    hooks   []ShutdownHook
}

type ShutdownHook struct {
    Name     string
    Priority int // 数值越小越先执行
    Fn       func(ctx context.Context) error
}

// 关闭顺序：
// 1. 停止接受新连接（HTTP servers）
// 2. 等待进行中的请求完成
// 3. 关闭消息队列消费者（NATS）
// 4. 关闭 Redis 连接
// 5. 关闭数据库连接池
// 6. 关闭 MinIO 客户端
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error
```

---

## 3. 详细设计

### 3.1 连接池参数推荐

**PostgreSQL（主库）**：

| 参数 | 10万规模 | 100万规模 |
|------|---------|----------|
| MaxConns | 50 | 100 |
| MinConns | 10 | 20 |
| MaxConnLifetime | 30m | 30m |
| MaxConnIdleTime | 5m | 5m |
| HealthCheckInterval | 30s | 30s |

**Redis**：

| 参数 | 10万规模 | 100万规模 |
|------|---------|----------|
| PoolSize | 200 | 500 |
| MinIdleConns | 50 | 100 |
| ReadTimeout | 3s | 3s |
| WriteTimeout | 3s | 3s |

### 3.2 重连策略

所有组件采用指数退避重连：
- 初始间隔：100ms
- 最大间隔：30s
- 退避因子：2
- NATS: 内置重连机制，配置 `MaxReconnects=-1`（无限重连）
- pgxpool: 内置重连，配置 HealthCheckPeriod
- go-redis: 内置重连，配置 MaxRetries

---

## 4. 实施子阶段

### 阶段 2a：PostgreSQL + TimescaleDB 连接

**交付物**：
- `internal/infra/db/postgres.go` — 连接池创建、配置、健康检查
- `internal/infra/db/timescale.go` — TimescaleDB 连接、扩展检查
- 单元测试

**验证**：连接本地 PostgreSQL，执行 `SELECT 1`

### 阶段 2b：Redis 连接

**交付物**：
- `internal/infra/cache/redis.go` — 客户端创建、Cluster/Standalone 自动判断
- 单元测试

**验证**：连接本地 Redis，执行 PING

### 阶段 2c：NATS 连接

**交付物**：
- `internal/infra/mq/nats.go` — 连接、JetStream、Stream 创建
- 单元测试

**验证**：连接本地 NATS，创建 Stream，发布/消费消息

### 阶段 2d：MinIO 连接

**交付物**：
- `internal/infra/storage/minio.go` — 客户端创建、Bucket 初始化
- 单元测试

**验证**：连接本地 MinIO，创建 Bucket，上传/下载文件

### 阶段 2e：统一健康检查 + 优雅关闭

**交付物**：
- `internal/infra/health.go` — 统一健康检查
- `internal/infra/shutdown.go` — 优雅关闭编排
- 集成测试

**验证**：启动应用后访问 `/healthz` 返回所有组件状态

---

## 5. 文件清单

```
internal/infra/db/postgres.go
internal/infra/db/timescale.go
internal/infra/cache/redis.go
internal/infra/mq/nats.go
internal/infra/storage/minio.go
internal/infra/health.go
internal/infra/shutdown.go
```

---

## 6. 测试策略

- **单元测试**：每个组件的配置解析、参数校验
- **集成测试**：使用 docker-compose 启动依赖服务，验证连接、CRUD、健康检查
- **测试 tag**：集成测试使用 `//go:build integration` 标签

---

## 7. 参考

- backend-design.md 第三章：技术栈选型
- backend-design.md 第五章：数据存储策略
- CLAUDE.md 第 3 节：技术栈

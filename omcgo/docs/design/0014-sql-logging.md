# SQL 日志记录设计

## 1. 需求背景

在开发和调试过程中，需要查看应用程序执行的 SQL 语句，以便：
- 排查数据问题
- 优化查询性能
- 验证业务逻辑正确性
- 审计数据库操作
- 与请求链路关联（通过 request_id）

## 2. 设计目标

1. 通过配置开关控制是否记录 SQL
2. 记录完整的 SQL 语句和参数
3. 记录执行时间和影响行数
4. **与 Request ID 中间件联动，在 SQL 日志中记录 request_id**
5. 对生产环境性能影响最小化
6. 支持敏感数据脱敏（可选）

## 3. 技术方案

### 3.1 实现位置选择

**方案对比：**

| 方案 | 优点 | 缺点 |
|------|------|------|
| pgx tracer（推荐） | 捕获所有 SQL，包括 ORM 内部查询 | 需要实现 tracer 接口 |
| squirrel 日志 | 简单直接 | 只捕获 squirrel 构建的查询 |
| pgx 日志级别 | 无需代码 | 日志格式不统一 |

**选择：pgx tracer** - 在 pgx 连接池层面实现，捕获所有 SQL 执行。

### 3.2 配置设计

```yaml
# cmd/app/etc/config.dev.yaml
db:
  dsn: "postgres://..."
  log_sql: true           # 是否记录 SQL
  log_sql_params: true    # 是否记录参数（生产环境建议 false）
  log_sql_slow_threshold: 1000  # 慢查询阈值(ms)，0 表示不区分

# cmd/app/etc/config.prod.yaml
db:
  dsn: "postgres://..."
  log_sql: false          # 生产环境默认关闭
  log_sql_params: false   # 生产环境不记录参数
  log_sql_slow_threshold: 5000  # 只记录慢查询
```

### 3.3 配置结构体更新

```go
// internal/core/appconfig/config.go

// PostgresConfig 更新
type PostgresConfig struct {
    DSN                 string        `mapstructure:"dsn"`
    MaxConns            int32         `mapstructure:"max_conns"`
    MinConns            int32         `mapstructure:"min_conns"`
    MaxConnLifetime     time.Duration `mapstructure:"max_conn_lifetime"`
    MaxConnIdleTime     time.Duration `mapstructure:"max_conn_idle_time"`
    HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
    // SQL 日志配置
    LogSQL              bool          `mapstructure:"log_sql"`
    LogSQLParams        bool          `mapstructure:"log_sql_params"`
    LogSQLSlowThreshold int           `mapstructure:"log_sql_slow_threshold"` // ms
}
```

### 3.4 SQL Tracer 实现（支持 Request ID）

```go
// internal/components/postgres/tracer.go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/tracelog"
    "go.uber.org/zap"

    "github.com/omcgo/omcgo/internal/core/context"
)

// SQLTracer 实现 pgx 的 TraceLogger 接口
type SQLTracer struct {
    logger         *zap.Logger
    logParams      bool
    slowThreshold  time.Duration
}

// NewSQLTracer 创建 SQL 追踪器
func NewSQLTracer(logger *zap.Logger, logParams bool, slowThresholdMs int) *SQLTracer {
    var threshold time.Duration
    if slowThresholdMs > 0 {
        threshold = time.Duration(slowThresholdMs) * time.Millisecond
    }
    return &SQLTracer{
        logger:        logger,
        logParams:     logParams,
        slowThreshold: threshold,
    }
}

// TraceQueryStart 查询开始时调用
func (t *SQLTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data tracelog.TraceQueryStartData) context.Context {
    return context.WithValue(ctx, queryStartTimeKey{}, time.Now())
}

type queryStartTimeKey struct{}

// TraceQueryEnd 查询结束时调用
func (t *SQLTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data tracelog.TraceQueryEndData) {
    startTime, ok := ctx.Value(queryStartTimeKey{}).(time.Time)
    if !ok {
        return
    }

    duration := time.Since(startTime)
    fields := []zap.Field{
        zap.String("sql", data.SQL),
        zap.Duration("duration", duration),
    }

    // 关键：从 context 获取 request_id
    if requestID := context.GetRequestID(ctx); requestID != "" {
        fields = append(fields, zap.String("request_id", requestID))
    }

    // 记录参数（可配置关闭）
    if t.logParams && len(data.Args) > 0 {
        fields = append(fields, zap.Any("params", data.Args))
    }

    // 记录影响行数
    if data.CommandTag.RowsAffected() > 0 {
        fields = append(fields, zap.Int64("rows", data.CommandTag.RowsAffected()))
    }

    // 根据执行时间决定日志级别
    if t.slowThreshold > 0 && duration > t.slowThreshold {
        t.logger.Warn("SLOW SQL", fields...)
    } else {
        t.logger.Debug("SQL", fields...)
    }
}

// TraceBatchStart 批量操作开始
func (t *SQLTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data tracelog.TraceBatchStartData) context.Context {
    return context.WithValue(ctx, batchStartTimeKey{}, time.Now())
}

type batchStartTimeKey struct{}

// TraceBatchEnd 批量操作结束
func (t *SQLTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data tracelog.TraceBatchEndData) {
    startTime, ok := ctx.Value(batchStartTimeKey{}).(time.Time)
    if !ok {
        return
    }

    duration := time.Since(startTime)
    fields := []zap.Field{
        zap.String("sql", "BATCH"),
        zap.Duration("duration", duration),
        zap.Int64("rows", data.CommandTag.RowsAffected()),
    }

    // 关键：从 context 获取 request_id
    if requestID := context.GetRequestID(ctx); requestID != "" {
        fields = append(fields, zap.String("request_id", requestID))
    }

    t.logger.Debug("BATCH SQL", fields...)
}

// TraceConnect 连接建立时调用
func (t *SQLTracer) TraceConnect(ctx context.Context, conn *pgx.Conn, data tracelog.TraceQueryStartData) context.Context {
    return ctx
}

// TracePrepare 准备语句时调用
func (t *SQLTracer) TracePrepare(ctx context.Context, conn *pgx.Conn, data tracelog.TracePrepareData) context.Context {
    return ctx
}

// TraceCopyFrom COPY 操作
func (t *SQLTracer) TraceCopyFrom(ctx context.Context, conn *pgx.Conn, data tracelog.TraceCopyFromStartData) context.Context {
    return ctx
}
```

### 3.5 连接池集成

```go
// internal/components/postgres/postgres.go

func NewPostgresPool(ctx context.Context, cfg appconfig.PostgresConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
    poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("parse database DSN: %w", err)
    }

    // 连接池配置
    poolConfig.MaxConns = cfg.MaxConns
    poolConfig.MinConns = cfg.MinConns
    poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
    poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
    poolConfig.HealthCheckPeriod = cfg.HealthCheckInterval

    // SQL 日志追踪器（仅在启用时）
    if cfg.LogSQL {
        tracer := NewSQLTracer(
            logger.Named("sql"),
            cfg.LogSQLParams,
            cfg.LogSQLSlowThreshold,
        )
        poolConfig.ConnConfig.Tracer = tracer
    }

    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("create connection pool: %w", err)
    }

    return pool, nil
}
```

### 3.6 Repository 层 Context 传递

确保 Repository 层的所有方法都接收 context 并正确传递：

```go
// 示例：正确传递 context
func (r *PgDeviceRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (*model.Device, error) {
    query := squirrel.Select("*").From("devices").Where(squirrel.Eq{"serial_number": serialNumber})

    sql, args, err := query.ToSql()
    if err != nil {
        return nil, err
    }

    var device model.Device
    // ctx 会传递 request_id 到 SQL tracer
    err = r.pool.QueryRow(ctx, sql, args...).Scan(&device.ID, &device.SerialNumber, ...)
    if err != nil {
        return nil, err
    }

    return &device, nil
}
```

## 4. 日志输出示例

### 4.1 开发环境（log_sql: true, log_sql_params: true）

```json
{"level":"debug","ts":"2026-03-19T15:04:30.123+0800","logger":"sql","msg":"SQL","request_id":"req-20260319150430-a1b2c3d4","sql":"SELECT id, serial_number, carrier FROM devices WHERE carrier = $1 AND status = $2","duration":"2.345ms","params":["cmcc","active"],"rows":42}
```

### 4.2 慢查询告警

```json
{"level":"warn","ts":"2026-03-19T15:04:31.789+0800","logger":"sql","msg":"SLOW SQL","request_id":"req-20260319150430-a1b2c3d4","sql":"SELECT * FROM pm_counters WHERE timestamp > $1 ORDER BY timestamp","duration":"5.234s","params":["2026-03-19T00:00:00Z"]}
```

### 4.3 无 Request ID 的场景

对于后台任务（Worker）或定时任务，可能没有 request_id：

```json
{"level":"debug","ts":"2026-03-19T15:04:30.123+0800","logger":"sql","msg":"SQL","sql":"SELECT * FROM devices WHERE status = $1","duration":"1.234ms","params":["active"],"rows":100}
```

## 5. 完整请求链路示例

一个 HTTP 请求从进入到数据库查询的完整日志链路：

```
# 1. 请求进入（App 日志）
{"level":"info","ts":"...","msg":"request started","request_id":"req-20260319150430-a1b2c3d4","method":"GET","path":"/api/v1/devices"}

# 2. 业务处理（App 日志）
{"level":"info","ts":"...","msg":"querying devices","request_id":"req-20260319150430-a1b2c3d4","carrier":"cmcc"}

# 3. SQL 执行（SQL 日志）
{"level":"debug","ts":"...","logger":"sql","msg":"SQL","request_id":"req-20260319150430-a1b2c3d4","sql":"SELECT * FROM devices WHERE carrier = $1","duration":"2.345ms","params":["cmcc"],"rows":42}

# 4. 请求完成（App 日志）
{"level":"info","ts":"...","msg":"request completed","request_id":"req-20260319150430-a1b2c3d4","status":200,"duration":"15.234ms"}
```

通过 `request_id` 可以将所有日志串联起来，便于问题排查。

## 6. 配置文件更新

### 6.1 开发环境配置

```yaml
# cmd/app/etc/config.dev.yaml
# cmd/worker/etc/config.dev.yaml

db:
  dsn: "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
  max_conns: 50
  min_conns: 10
  max_conn_lifetime: 30m
  max_conn_idle_time: 5m
  health_check_interval: 30s
  # SQL 日志配置
  log_sql: true
  log_sql_params: true
  log_sql_slow_threshold: 1000  # 1秒

tsdb:
  dsn: "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
  max_conns: 30
  min_conns: 5
  max_conn_lifetime: 30m
  max_conn_idle_time: 5m
  health_check_interval: 30s
  # SQL 日志配置
  log_sql: true
  log_sql_params: true
  log_sql_slow_threshold: 2000  # 2秒（时序查询可能更慢）
```

### 6.2 生产环境配置

```yaml
# cmd/app/etc/config.prod.yaml
# cmd/worker/etc/config.prod.yaml

db:
  dsn: "postgres://..."
  max_conns: 50
  min_conns: 10
  max_conn_lifetime: 30m
  max_conn_idle_time: 5m
  health_check_interval: 30s
  # SQL 日志配置（生产环境默认关闭）
  log_sql: false
  log_sql_params: false
  log_sql_slow_threshold: 5000  # 只记录超过5秒的慢查询

tsdb:
  dsn: "postgres://..."
  max_conns: 30
  min_conns: 5
  max_conn_lifetime: 30m
  max_conn_idle_time: 5m
  health_check_interval: 30s
  log_sql: false
  log_sql_params: false
  log_sql_slow_threshold: 10000  # 10秒
```

## 7. 文件清单

| 文件 | 说明 |
|------|------|
| `internal/core/appconfig/config.go` | 修改 - 添加 SQL 日志配置字段 |
| `internal/components/postgres/tracer.go` | 新建 - SQL 追踪器实现 |
| `internal/components/postgres/postgres.go` | 修改 - 集成 Tracer |
| `cmd/*/etc/config.*.yaml` | 修改 - 添加 SQL 日志配置 |
| 所有 Repository 文件 | 确保正确传递 context |

## 8. 实施步骤

1. **Phase 1**: 更新配置
   - 修改 `PostgresConfig` 添加 SQL 日志配置字段
   - 更新所有配置文件

2. **Phase 2**: 实现 SQL Tracer
   - 创建 `internal/components/postgres/tracer.go`
   - 实现包含 request_id 的日志记录

3. **Phase 3**: 集成到连接池
   - 修改 `NewPostgresPool` 和 `NewTimescalePool`
   - 根据配置启用 Tracer

4. **Phase 4**: 测试验证
   - 单元测试
   - 集成测试

## 9. 注意事项

### 9.1 性能影响

- SQL 日志会增加少量开销（字符串格式化、日志写入）
- 生产环境建议关闭或仅记录慢查询
- 参数记录可能包含敏感数据，生产环境应关闭

### 9.2 Context 传递

- 确保所有数据库操作都正确传递 context
- 从 HTTP Handler → Service → Repository → pgx，context 必须一路传递
- 如果 context 中断，request_id 将无法记录到 SQL 日志

### 9.3 敏感数据处理

如果需要记录参数但保护敏感数据，可以添加脱敏逻辑：

```go
// 敏感字段列表
var sensitiveFields = []string{"password", "secret", "token", "key"}

func (t *SQLTracer) sanitizeParams(params []any) []any {
    // 实现参数脱敏逻辑
    // ...
}
```

## 10. 测试验证

```bash
# 开发环境启动后，执行任意 API 请求
curl http://localhost:8081/api/v1/devices

# 查看日志中的 SQL 记录（应包含 request_id）
tail -f ~/data/logs/omcgo/app/app.log | grep '"logger":"sql"'

# 预期输出类似：
# {"level":"debug","logger":"sql","msg":"SQL","request_id":"req-xxx","sql":"SELECT ...","duration":"1.234ms"}
```

## 11. 与 Request ID 中间件的依赖关系

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Request                              │
│                    X-Request-ID: req-xxx                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    RequestID Middleware                          │
│         提取/生成 request_id，存入 context                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ ctx (包含 request_id)
┌─────────────────────────────────────────────────────────────────┐
│                      Handler / Service                           │
│              使用 context logger 记录日志                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ ctx 传递
┌─────────────────────────────────────────────────────────────────┐
│                       Repository                                 │
│              ctx 传递给数据库操作                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ ctx 传递
┌─────────────────────────────────────────────────────────────────┐
│                      pgx Tracer                                  │
│         从 ctx 提取 request_id，记录到 SQL 日志                   │
└─────────────────────────────────────────────────────────────────┘
```

**依赖关系**：
- SQL 日志功能可以独立启用/禁用
- 启用 Request ID 中间件后，SQL 日志会自动包含 request_id
- 不启用 Request ID 中间件，SQL 日志仍可正常工作（只是没有 request_id）

# Goose + sqlc + pgx 完整使用示例

> **文档版本**: v1.0  
> **创建时间**: 2025-04-10  
> **目标**: 演示三种工具如何配合使用

---

## 一、完整工作流概览

```
┌─────────────────────────────────────────────────────────────┐
│                     完整工作流                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. 开发阶段                                                 │
│     ┌──────────────────────────────────────────────────┐   │
│     │  a. 编写 Goose 迁移文件 (DDL)                     │   │
│     │     └─> 001_create_pm_counters.up.sql            │   │
│     │                                                  │   │
│     │  b. 编写 sqlc 查询文件 (DML)                     │   │
│     │     └─> pm_counters_queries.sql                  │   │
│     │                                                  │   │
│     │  c. 配置 sqlc.yaml                               │   │
│     └──────────────────────────────────────────────────┘   │
│                        ↓                                    │
│  2. 构建阶段                                                 │
│     ┌──────────────────────────────────────────────────┐   │
│     │  a. 执行 Goose 迁移 (创建表结构)                  │   │
│     │     └─> goose up                                 │   │
│     │                                                  │   │
│     │  b. 生成 sqlc 代码 (类型安全查询)                 │   │
│     │     └─> sqlc generate                            │   │
│     └──────────────────────────────────────────────────┘   │
│                        ↓                                    │
│  3. 运行阶段                                                 │
│     ┌──────────────────────────────────────────────────┐   │
│     │  a. 使用 sqlc 生成的代码查询数据                  │   │
│     │     └─> db.GetCounterStatsByTimeRange()          │   │
│     │                                                  │   │
│     │  b. 使用 pgx 原生调用管理函数                     │   │
│     │     └─> pmAdmin.SetRetentionPolicy()             │   │
│     └──────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、步骤详解

### 步骤 1: 创建 Goose 迁移文件

**文件**: `migrations/001_create_pm_counters.up.sql`

```sql
-- +goose Up
-- +goose StatementBegin

-- 1. 创建 PM 计数器基础表
CREATE TABLE IF NOT EXISTS pm_counters (
    time TIMESTAMPTZ NOT NULL,
    device_id UUID NOT NULL,
    counter_name VARCHAR(100) NOT NULL,
    counter_value DOUBLE PRECISION,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 2. 添加主键 (必须在 create_hypertable 之前)
ALTER TABLE pm_counters ADD PRIMARY KEY (time, device_id, counter_name);

-- 3. 转换为 TimescaleDB 超表
SELECT create_hypertable(
    'pm_counters', 
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- 4. 创建索引
CREATE INDEX IF NOT EXISTS idx_pm_counters_device_time 
ON pm_counters (device_id, time DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pm_counters CASCADE;
-- +goose StatementEnd
```

**执行迁移**:

```bash
# 执行所有待处理的迁移
goose postgres "postgres://omcgo:omcgo@localhost:5432/omcgo?sslmode=disable" up

# 输出:
# OK    001_create_pm_counters.sql (45.2ms)
```

---

### 步骤 2: 编写 sqlc 查询文件

**文件**: `sql/queries/pm_counters.sql`

```sql
-- name: GetCounterStatsByTimeRange :many
SELECT 
    time_bucket($1::INTERVAL, time) AS bucket,
    device_id,
    counter_name,
    AVG(counter_value) AS avg_value,
    MAX(counter_value) AS max_value,
    MIN(counter_value) AS min_value,
    COUNT(*) AS sample_count
FROM pm_counters
WHERE time >= $2 
  AND time < $3
  AND device_id = $4
GROUP BY bucket, device_id, counter_name
ORDER BY bucket DESC;

-- name: GetLatestCounterValue :one
SELECT 
    time,
    device_id,
    counter_name,
    counter_value,
    metadata
FROM pm_counters
WHERE device_id = $1 
  AND counter_name = $2
ORDER BY time DESC
LIMIT 1;

-- name: BatchInsertCounters :copyfrom
INSERT INTO pm_counters (
    time,
    device_id,
    counter_name,
    counter_value,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
);
```

---

### 步骤 3: 配置 sqlc

**文件**: `sqlc.yaml`

```yaml
version: "2"

sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/schema"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        
        emit_json_tags: true
        emit_interface: true
        
        overrides:
          - db_type: "jsonb"
            go_type:
              import: "encoding/json"
              type: "RawMessage"
          
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
```

**生成代码**:

```bash
# 生成类型安全的 Go 代码
sqlc generate

# 输出:
# generated code for 3 queries
# internal/db/pm_counters.sql.go
# internal/db/models.go
# internal/db/db.go
```

---

### 步骤 4: 使用生成的代码

**自动生成的代码**: `internal/db/pm_counters.sql.go`

```go
// Code generated by sqlc. DO NOT EDIT.
// source: pm_counters.sql

package db

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type GetCounterStatsByTimeRangeParams struct {
    TimeBucket time.Duration
    Time       time.Time
    Time_2     time.Time
    DeviceID   uuid.UUID
}

type GetCounterStatsByTimeRangeRow struct {
    Bucket      time.Time
    DeviceID    uuid.UUID
    CounterName string
    AvgValue    float64
    MaxValue    float64
    MinValue    float64
    SampleCount int64
}

func (q *Queries) GetCounterStatsByTimeRange(ctx context.Context, arg GetCounterStatsByTimeRangeParams) ([]GetCounterStatsByTimeRangeRow, error) {
    rows, err := q.db.Query(ctx, getCounterStatsByTimeRange,
        arg.TimeBucket, arg.Time, arg.Time_2, arg.DeviceID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []GetCounterStatsByTimeRangeRow
    for rows.Next() {
        var i GetCounterStatsByTimeRangeRow
        if err := rows.Scan(
            &i.Bucket,
            &i.DeviceID,
            &i.CounterName,
            &i.AvgValue,
            &i.MaxValue,
            &i.MinValue,
            &i.SampleCount,
        ); err != nil {
            return nil, err
        }
        items = append(items, i)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return items, nil
}
```

**业务代码中使用**:

```go
// internal/pm/handler.go
package pm

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/omcgo/omcgo/internal/db"
)

type PMHandler struct {
    queries *db.Queries
}

func NewPMHandler(queries *db.Queries) *PMHandler {
    return &PMHandler{queries: queries}
}

// GetPMStats 获取 PM 统计信息
func (h *PMHandler) GetPMStats(c *gin.Context) {
    // 解析请求参数
    deviceID, _ := uuid.Parse(c.Param("device_id"))
    duration, _ := time.ParseDuration(c.Query("duration")) // e.g., "5m"
    
    now := time.Now()
    startTime := now.Add(-1 * time.Hour)

    // 调用 sqlc 生成的类型安全查询
    stats, err := h.queries.GetCounterStatsByTimeRange(c.Request.Context(), db.GetCounterStatsByTimeRangeParams{
        TimeBucket: duration,
        Time:       startTime,
        Time_2:     now,
        DeviceID:   deviceID,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  stats,
        "count": len(stats),
    })
}
```

---

### 步骤 5: 使用 pgx 原生调用管理函数

**文件**: `internal/pm/admin_service.go`

```go
package pm

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

// PMAdminService PM 管理服务
type PMAdminService struct {
    pool   *pgxpool.Pool
    logger *zap.Logger
}

// SetRetentionPolicy 设置数据保留策略
func (s *PMAdminService) SetRetentionPolicy(ctx context.Context, tableName string, retentionDays int) error {
    s.logger.Info("setting retention policy",
        zap.String("table", tableName),
        zap.Int("retention_days", retentionDays))

    _, err := s.pool.Exec(ctx, `
        SELECT add_retention_policy($1, 
            drop_after => INTERVAL $2 || ' days',
            if_not_exists => TRUE
        )
    `, tableName, retentionDays)

    if err != nil {
        return fmt.Errorf("set retention policy: %w", err)
    }

    return nil
}
```

**在 HTTP Handler 中使用**:

```go
// SetRetentionPolicy 设置保留策略 (Web 界面调用)
func (h *PMHandler) SetRetentionPolicy(c *gin.Context) {
    var req struct {
        TableName     string `json:"table_name" binding:"required"`
        RetentionDays int    `json:"retention_days" binding:"required,min=1"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    err := h.adminService.SetRetentionPolicy(c.Request.Context(), req.TableName, req.RetentionDays)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "retention policy set successfully",
        "table":   req.TableName,
        "days":    req.RetentionDays,
    })
}
```

---

## 三、完整目录结构

```
omcgo/
├── migrations/                          # Goose 迁移文件
│   ├── 001_create_pm_counters.up.sql   # 创建 PM 超表
│   └── 001_create_pm_counters.down.sql # 回滚
│
├── sql/                                 # sqlc 查询定义
│   ├── schema/                         # Schema 定义
│   │   └── pm_counters.sql
│   └── queries/                        # 查询定义
│       └── pm_counters.sql
│
├── sqlc.yaml                            # sqlc 配置
│
├── internal/
│   ├── db/                             # sqlc 生成的代码 (自动生成)
│   │   ├── db.go
│   │   ├── models.go
│   │   └── pm_counters.sql.go
│   │
│   └── pm/                             # PM 模块业务代码
│       ├── handler.go                  # HTTP Handler
│       ├── admin_service.go            # 管理服务 (pgx 原生)
│       └── model.go                    # 数据模型
│
└── cmd/
    └── app/
        └── main.go
```

---

## 四、Makefile 命令

```makefile
# 数据库迁移
migrate-up:
	goose postgres "$(DB_DSN)" up

migrate-down:
	goose postgres "$(DB_DSN)" down

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	goose create -dir migrations -seq $(name) sql

# sqlc 代码生成
sqlc-generate:
	sqlc generate

sqlc-validate:
	sqlc vet

# 完整构建流程
build-db: migrate-up sqlc-generate
```

**使用示例**:

```bash
# 1. 创建新迁移
make migrate-create name=add_pm_indexes

# 2. 执行迁移
make migrate-up

# 3. 生成 sqlc 代码
make sqlc-generate

# 4. 验证 sqlc 查询
make sqlc-validate
```

---

## 五、常见问题 FAQ

### Q1: 什么时候用 Goose,什么时候用 sqlc?

**A**: 
- **Goose**: DDL 语句 (CREATE/ALTER/DROP)
- **sqlc**: DML 语句 (SELECT/INSERT/UPDATE/DELETE)

### Q2: TimescaleDB 的 create_hypertable 用哪个工具?

**A**: 使用 **Goose** (这是 DDL 管理语句)

### Q3: time_bucket() 函数查询用哪个工具?

**A**: 使用 **sqlc** (这是查询语句)

### Q4: add_retention_policy() 用哪个工具?

**A**: 使用 **pgx 原生** (这是运行时管理操作,可能通过 Web 界面动态调整)

### Q5: 如何处理 JSONB 类型?

**A**: 在 `sqlc.yaml` 中配置 overrides:

```yaml
overrides:
  - db_type: "jsonb"
    go_type:
      import: "encoding/json"
      type: "RawMessage"
```

### Q6: 动态查询 (搜索/过滤) 怎么办?

**A**: 使用 **goqu** (sqlc 不适合动态查询)

```go
query := goqu.From("pm_counters")
if filter.DeviceID != "" {
    query = query.Where(goqu.C("device_id").Eq(filter.DeviceID))
}
```

---

## 六、性能对比

| 操作类型 | 工具 | 性能 | 说明 |
|---------|------|------|------|
| **简单查询** | sqlc | ⚡ 最快 | 零开销,直接调用 pgx |
| **批量插入** | sqlc :copyfrom | ⚡ 最快 | 使用 PostgreSQL COPY 协议 |
| **动态查询** | goqu | 快 | 运行时构建 SQL |
| **JSONB 查询** | sqlc | ⚡ 最快 | 类型安全映射 |
| **管理操作** | pgx 原生 | ⚡ 最快 | 直接执行 |

---

## 七、最佳实践总结

### ✅ **应该做的**

1. **使用 Goose 管理 Schema 版本**
   - 所有 DDL 语句放入迁移文件
   - 包含完整的回滚脚本

2. **使用 sqlc 生成查询代码**
   - 所有固定查询使用 sqlc
   - 充分利用类型安全检查
   - JSONB 配置自定义类型映射

3. **使用 pgx 原生调用管理函数**
   - TimescaleDB 策略管理
   - 运行时动态配置
   - 系统视图查询

4. **使用 goqu 处理动态查询**
   - 设备搜索/过滤
   - 多条件组合查询
   - 用户自定义报表

### ❌ **不应该做的**

1. ❌ 在 sqlc 中写 DDL 语句
2. ❌ 在 Goose 中写查询语句
3. ❌ 使用 Squirrel 新建代码
4. ❌ 手写扫描代码 (使用 sqlc 生成)

---

## 八、迁移检查清单

从 Squirrel 迁移时,使用此清单确保完整性:

- [ ] 封装 Squirrel 抽象层
- [ ] 创建 sqlc 配置文件
- [ ] 迁移固定查询到 sqlc
- [ ] 迁移动态查询到 goqu
- [ ] DDL 语句放入 Goose 迁移
- [ ] 管理操作使用 pgx 原生
- [ ] 所有查询添加单元测试
- [ ] JSONB 类型映射验证
- [ ] TimescaleDB 函数测试
- [ ] 性能基准测试

---

## 九、参考资料

- [Goose 官方文档](https://github.com/pressly/goose)
- [sqlc 官方文档](https://docs.sqlc.dev/)
- [goqu 官方文档](https://github.com/doug-martin/goqu)
- [TimescaleDB 文档](https://docs.timescale.com/)
- [pgx 官方文档](https://github.com/jackc/pgx)

---

**文档维护**: 随着项目实施持续更新  
**最后更新**: 2025-04-10

# SQL Builder 演进建议与工具分工方案

> **文档版本**: v1.0  
> **创建时间**: 2025-04-10  
> **状态**: 待决策

---

## 一、当前 SQL Builder 现状分析

### 1.1 Squirrel 使用情况

| 维度 | 数据 |
|------|------|
| **版本** | v1.5.4 (2021年发布) |
| **维护状态** | ⚠️ **基本停滞** (3+ 年未发布新版本) |
| **使用范围** | 40 个 repository 文件 |
| **Issues** | 60+ 未关闭 |
| **PR 积压** | 20+ 未合并 |

### 1.2 风险评估

| 风险项 | 影响程度 | 发生概率 | 说明 |
|--------|---------|---------|------|
| 未修复 Bug | 中 | 中 | 部分边界场景存在问题 |
| 不支持新 PG 特性 | 低 | 高 | PostgreSQL 17/18 新语法不支持 |
| 安全漏洞无人修复 | 高 | 低 | 基础库风险较低 |
| Go 新版本兼容性 | 中 | 中 | 可能需自行维护 fork |

**结论**: Squirrel 功能稳定,**短期不会成为致命问题**,但建议开始规划迁移。

---

## 二、替代方案对比

### 2.1 方案总览

| 方案 | 类型 | 维护状态 | 适用场景 | 迁移成本 |
|------|------|---------|---------|---------|
| **保持 Squirrel** | SQL Builder | ⚠️ 停滞 | 现有代码 | 无 |
| **goqu** | SQL Builder | ✅ 活跃 | 动态查询 | 中 (40文件) |
| **sqlc** | 代码生成器 | ✅ 活跃 | 固定查询 | 高 (需重写) |
| **混合方案** | 组合 | ✅ 推荐 | 全场景 | 渐进式 |

### 2.2 详细对比

#### **方案 A: goqu** (SQL Builder 替代)

```go
// 当前 Squirrel 写法
query, args, _ := sq.Insert("users").
    Columns("username", "email").
    Values("admin", "admin@example.com").
    ToSql()

// 迁移到 goqu
query, args, _ := goqu.Insert("users").
    Cols("username", "email").
    Vals(goqu.Vals{"admin", "admin@example.com"}).
    ToSQL()
```

| 对比项 | Squirrel | goqu | 优势方 |
|--------|----------|------|--------|
| **维护状态** | ⚠️ 停滞 (2021) | ✅ 活跃 (2024) | goqu |
| **GitHub Stars** | 6.1k | 2.1k | Squirrel |
| **API 设计** | 链式 | 链式 + DSL | goqu |
| **类型安全** | 弱 | 中等 | goqu |
| **复合查询** | 有限 | 优秀 (UNION/JOIN) | goqu |
| **PostgreSQL 特性** | 基础 | 丰富 (RETURNING) | goqu |
| **学习曲线** | 低 | 低 | 持平 |
| **迁移工作量** | - | 3-5 天 (40文件) | Squirrel |

**goqu 核心优势**:
- ✅ 活跃维护,响应及时
- ✅ 更好的复合查询支持
- ✅ 内置表达式构建器
- ✅ 完整支持 PostgreSQL RETURNING 子句

---

#### **方案 B: sqlc** (类型安全代码生成)

```sql
-- sql/queries/users.sql
-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, created_at;

-- name: ListUsers :many
SELECT id, username, email, status
FROM users
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
```

```go
// sqlc 自动生成的类型安全代码
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
    row := q.db.QueryRow(ctx, createUser, arg.Username, arg.Email, arg.PasswordHash)
    var u User
    err := row.Scan(&u.ID, &u.CreatedAt)
    return u, err
}
```

| 对比项 | Squirrel | sqlc | 优势方 |
|--------|----------|------|--------|
| **类型安全** | ❌ 运行时 | ✅ 编译时 | sqlc 碾压 |
| **SQL 控制** | 构建器 | 手写 SQL | sqlc |
| **重构安全** | 低 | 高 | sqlc |
| **性能** | 中 | 高 (零开销) | sqlc |
| **动态查询** | ✅ 优秀 | ⚠️ 有限 | Squirrel |
| **JSONB 支持** | ⚠️ 字符串拼接 | ✅ 类型映射 | sqlc |
| **TimescaleDB** | ⚠️ 字符串拼接 | ✅ 函数调用 | sqlc |
| **学习曲线** | 低 | 中 | Squirrel |
| **迁移工作量** | - | 高 (重写所有查询) | Squirrel |

**sqlc 核心优势**:
- ✅ **编译时类型检查**,彻底消除运行时 SQL 错误
- ✅ 手写 SQL,完全控制查询性能
- ✅ 自动生成类型安全的 Go 代码
- ✅ 更好的 IDE 支持 (SQL 语法高亮 + 自动补全)
- ✅ 零运行时开销 (直接调用 database/sql)
- ✅ **完美支持 JSONB 自定义类型映射**
- ✅ **完全支持 TimescaleDB 查询函数**

**sqlc 劣势**:
- ⚠️ 动态查询支持较弱 (需配合其他方案)
- ⚠️ 迁移成本高 (需重写 40 个文件)

---

#### **方案 C: 封装抽象层** (最低风险过渡)

```go
// internal/core/db/query.go
package db

import (
    sq "github.com/Masterminds/squirrel"
)

// 统一封装,未来可替换底层实现
var StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type QueryBuilder interface {
    Insert(table string) InsertBuilder
    Select(columns ...string) SelectBuilder
    Update(table string) UpdateBuilder
    Delete(table string) DeleteBuilder
}

// 当前使用 Squirrel 实现
type squirrelBuilder struct{}

func (b *squirrelBuilder) Insert(table string) InsertBuilder {
    return StatementBuilder.Insert(table)
}

// 未来可替换为 goqu 实现
type goquBuilder struct{}

func (b *goquBuilder) Insert(table string) InsertBuilder {
    return goqu.Insert(table)
}
```

**优势**:
- ✅ **零迁移成本**,立即可用
- ✅ 通过接口抽象,未来可随时替换
- ✅ 封装常用模式,减少重复代码
- ✅ 不影响现有业务逻辑

---

## 三、PostgreSQL 与 TimescaleDB 支持情况

### 3.1 sqlc 对 PostgreSQL 特性支持

| 特性 | 支持情况 | 说明 |
|------|---------|------|
| **JSONB** | ✅ 完全支持 | 可通过 type overrides 映射到自定义 Go struct |
| **数组类型** | ✅ 完全支持 | `INTEGER[]` → `[]int32`, `TEXT[]` → `[]string` |
| **自定义类型** | ✅ 完全支持 | 通过 `overrides` 配置自定义映射 |
| **UUID** | ✅ 完全支持 | 原生支持 `github.com/google/uuid` |
| **HSTORE** | ✅ 支持 | 可映射到 `map[string]string` |
| **枚举类型** | ✅ 完全支持 | 自动生成 Go 枚举 |
| **PG 16** | ✅ 完全支持 | - |
| **PG 17** | ✅ 支持 | 2024年发布 |
| **PG 18** | ⏳ 部分支持 | 持续跟进中 |

### 3.2 sqlc 对 TimescaleDB 支持

#### ✅ **完全支持** (查询语句)

```sql
-- time_bucket 函数
SELECT time_bucket('5 minutes', time) AS bucket,
       AVG(temperature) AS avg_temp
FROM sensor_data
WHERE time > NOW() - INTERVAL '1 hour'
GROUP BY bucket
ORDER BY bucket;

-- 连续聚合查询
SELECT * FROM metrics_1hour
WHERE time >= NOW() - INTERVAL '7 days';

-- 压缩数据查询
SELECT * FROM compressible_data
WHERE time < NOW() - INTERVAL '30 days';
```

#### ❌ **不支持** (DDL/管理语句)

| 特性 | 支持情况 | 原因 | 解决方案 |
|------|---------|------|---------|
| **CREATE_HYPERTABLE** | ❌ | DDL 语句 | 使用 Goose 迁移 |
| **ALTER TABLE 超表** | ❌ | DDL 语句 | 使用 Goose 迁移 |
| **数据保留策略** | ❌ | 管理函数 | 使用 pgx 原生调用 |
| **压缩策略** | ❌ | 管理函数 | 使用 pgx 原生调用 |

**关键结论**:
- ✅ **查询语句**: sqlc 100% 支持
- ❌ **管理/DDL 语句**: 需使用 Goose 或 pgx 原生

---

## 四、工具分工方案 (Goose + sqlc + pgx)

### 4.1 核心原则

```
┌─────────────────────────────────────────────────┐
│          数据库操作完整生命周期                    │
├─────────────────────────────────────────────────┤
│                                                 │
│  1. Schema 创建 (一次性)                         │
│     ┌─────────────────────────────────────┐     │
│     │  Goose 迁移工具                      │     │
│     │                                     │     │
│     │  • CREATE TABLE                     │     │
│     │  • CREATE_HYPERTABLE (DDL)         │     │
│     │  • CREATE INDEX                     │     │
│     │  • ALTER TABLE                      │     │
│     │  • 版本管理 & 回滚                   │     │
│     └─────────────────────────────────────┘     │
│                      ↓                          │
│  2. 查询代码生成 (开发时)                         │
│     ┌─────────────────────────────────────┐     │
│     │  sqlc 代码生成器                     │     │
│     │                                     │     │
│     │  • SELECT 查询                      │     │
│     │  • INSERT/UPDATE/DELETE             │     │
│     │  • TimescaleDB 函数调用             │     │
│     │  • time_bucket()                   │     │
│     │  • 生成类型安全的 Go 代码            │     │
│     └─────────────────────────────────────┘     │
│                      ↓                          │
│  3. 运行时管理 (按需)                            │
│     ┌─────────────────────────────────────┐     │
│     │  pgx 原生执行                        │     │
│     │                                     │     │
│     │  • add_retention_policy()          │     │
│     │  • add_compression_policy()        │     │
│     │  • 动态调整配置                      │     │
│     │  • Web 界面管理                      │     │
│     └─────────────────────────────────────┘     │
│                                                 │
└─────────────────────────────────────────────────┘
```

### 4.2 详细分工

| 操作类型 | 工具 | 示例 | 执行时机 | 频率 |
|---------|------|------|---------|------|
| **创建表/超表** | Goose | `CREATE TABLE` + `create_hypertable()` | 部署时 | 一次性 |
| **添加索引** | Goose | `CREATE INDEX` | 部署时 | 低 |
| **Schema 变更** | Goose | `ALTER TABLE` | 部署时 | 低 |
| **查询数据** | sqlc | `SELECT time_bucket(...)`, `INSERT`, `UPDATE` | 运行时 | 高 |
| **设置保留策略** | pgx 原生 | `add_retention_policy()` | 运行时 (可配置) | 低 |
| **设置压缩策略** | pgx 原生 | `add_compression_policy()` | 运行时 (可配置) | 低 |
| **动态调整** | pgx 原生 | 通过 Web 界面修改参数 | 运行时 (按需) | 中 |

### 4.3 三者关系

```
Goose (迁移管理)
  ↓ 创建 Schema
  ↓
sqlc (代码生成)
  ↓ 生成查询代码
  ↓
pgx 原生 (运行时管理)
  ↓ 动态调整配置
```

**核心原则**:
- 📦 **Goose** = 数据库 **Schema 版本管理**
- 🔧 **sqlc** = **查询代码**自动生成
- ⚙️ **pgx 原生** = **运行时管理操作**

三者**互补**,不是替代关系!

---

## 五、推荐方案: 混合架构

### 5.1 方案概述

```
┌─────────────────────────────────────────┐
│           查询类型分类                    │
├─────────────────────────────────────────┤
│                                         │
│  固定查询 (80%)     动态查询 (20%)       │
│  ┌─────────────┐    ┌──────────────┐    │
│  │  sqlc ✅    │    │   goqu ✅    │    │
│  │             │    │              │    │
│  │ • CRUD      │    │ • 设备搜索   │    │
│  │ • PM 统计   │    │ • 动态过滤   │    │
│  │ • KPI 计算  │    │ • 复杂条件   │    │
│  │ • 告警查询  │    │ • 多条件组合 │    │
│  │ • Timescale │    │              │    │
│  │   函数调用  │    │              │    │
│  └─────────────┘    └──────────────┘    │
│                                         │
│  DDL/管理语句 (手写)                     │
│  ┌──────────────────────────────┐       │
│  │  Goose 迁移 + pgx 执行        │       │
│  │  • CREATE_HYPERTABLE         │       │
│  │  • 数据保留策略               │       │
│  │  • 压缩策略                   │       │
│  └──────────────────────────────┘       │
└─────────────────────────────────────────┘
```

### 5.2 实施路线图

#### 阶段 1: 基础设施搭建 (1-2 周)

**任务清单**:
- [ ] 封装 Squirrel 抽象层 (1天)
- [ ] 引入 Goose 替代 golang-migrate (2天)
- [ ] 配置 sqlc 工具链 (1天)
- [ ] 创建示例模块验证方案 (2天)

**产出物**:
- `internal/core/db/query.go` - SQL Builder 抽象层
- `migrations/` - Goose 迁移文件目录
- `sql/` - sqlc 查询定义目录
- `sqlc.yaml` - sqlc 配置文件

---

#### 阶段 2: 新模块使用 sqlc (持续)

**规则**:
- ✅ 所有新模块的固定查询使用 sqlc
- ✅ PM/KPI 时序查询使用 sqlc (TimescaleDB 函数)
- ✅ JSONB 复杂映射使用 sqlc (类型安全)
- ⚠️ 动态搜索/过滤使用 goqu

**示例模块**: PM 文件管理

```sql
-- sql/schema/pm_files.sql
CREATE TABLE pm_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    file_name TEXT NOT NULL,
    file_size BIGINT,
    parsed_data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- sql/queries/pm_files.sql
-- name: GetPMFilesByDevice :many
SELECT id, device_id, file_name, file_size, 
       parsed_data, created_at
FROM pm_files
WHERE device_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPMStats :one
SELECT 
    COUNT(*) as total_files,
    SUM(file_size) as total_size,
    MAX(created_at) as last_upload
FROM pm_files
WHERE device_id = $1;
```

---

#### 阶段 3: 现有模块迁移 (1-2 月)

**迁移优先级**:

| 优先级 | 模块 | 文件数 | 工作量 | 收益 |
|--------|------|--------|--------|------|
| **P0** | PM/KPI 查询 | 12 | 3天 | 高 (TimescaleDB) |
| **P0** | 告警查询 | 8 | 2天 | 高 (JSONB) |
| **P1** | 设备管理 | 10 | 3天 | 中 (CRUD) |
| **P1** | 用户管理 | 6 | 2天 | 中 (CRUD) |
| **P2** | 动态搜索 | 4 | 保留 goqu | - |

**迁移策略**:
1. 按模块逐步迁移,不影响其他模块
2. 每个模块迁移后运行完整测试
3. 保留 Squirrel 抽象层,紧急时可回退

---

## 六、JSONB 类型映射示例

### 6.1 定义 JSONB 结构

```sql
-- schema.sql
CREATE TABLE pm_counters (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL,
    counter_data JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL
);
```

### 6.2 配置 sqlc 映射

```yaml
# sqlc.yaml
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
        overrides:
          - db_type: "jsonb"
            go_type:
              import: "encoding/json"
              type: "RawMessage"
          
          # 或映射到自定义类型
          - db_type: "jsonb"
            column: "counter_data"
            go_type:
              type: "CounterData"
              import: "github.com/omcgo/omcgo/internal/pm/model"
```

### 6.3 编写查询

```sql
-- query/pm.sql
-- name: GetCounterData :one
SELECT id, device_id, counter_data, created_at
FROM pm_counters
WHERE id = $1;

-- name: UpdateCounterData :exec
UPDATE pm_counters
SET counter_data = $2
WHERE id = $1;
```

### 6.4 自动生成类型安全代码

```go
// 由 sqlc 自动生成
type CounterData struct {
    CPUUsage    float64 `json:"cpu_usage"`
    MemoryUsage float64 `json:"memory_usage"`
    // ...
}

func (q *Queries) GetCounterData(ctx context.Context, id uuid.UUID) (PmCounter, error) {
    row := q.db.QueryRow(ctx, getCounterData, id)
    var c PmCounter
    err := row.Scan(&c.ID, &c.DeviceID, &c.CounterData, &c.CreatedAt)
    return c, err
}
```

---

## 七、实施决策建议

### 7.1 基于项目特点评估

| 因素 | 评估 | 建议 |
|------|------|------|
| **PostgreSQL 版本** | PG 16 | ✅ sqlc 完全支持 |
| **TimescaleDB 使用** | PM/KPI 时序 | ✅ 查询支持,DDL 用 Goose |
| **JSONB 使用量** | 高 (设备信息/配置) | ✅ sqlc 类型安全优势明显 |
| **动态查询需求** | 中等 (设备搜索/过滤) | ⚠️ goqu 更适合 |
| **团队学习成本** | 中等 | 📚 需 1-2 周适应 |
| **迁移工作量** | 40 个文件 | 🔄 建议渐进式 |

### 7.2 最终建议

#### 立即实施 (1-2周):
1. ✅ **封装 Squirrel 抽象层** (零风险,1天)
2. ✅ **引入 Goose 替代 golang-migrate** (2天)
3. ✅ **配置 sqlc 工具链** (1天)
4. ✅ **创建示例模块验证** (2天)

#### 短期优化 (1个月):
1. 🔄 **新模块使用 sqlc** (类型安全)
2. 🔄 **PM/KPI 查询迁移到 sqlc** (TimescaleDB 函数调用更直观)
3. 🔄 **动态查询引入 goqu**

#### 长期规划 (3个月):
1. 🔄 **核心 CRUD 逐步迁移到 sqlc**
2. 🔄 **保留 goqu 用于复杂动态查询**
3. 🔄 **完全弃用 Squirrel**

---

## 八、风险与注意事项

### 8.1 迁移风险

| 风险项 | 影响 | 缓解措施 |
|--------|------|---------|
| 迁移过程中引入 Bug | 高 | 每个模块迁移后运行完整测试 |
| 团队学习曲线 | 中 | 提供培训 + 示例代码 |
| sqlc 不支持的动态查询 | 中 | 保留 goqu 作为补充 |
| 迁移时间超预期 | 中 | 分阶段实施,不阻塞业务 |

### 8.2 回退方案

如果迁移过程中遇到问题:
1. Squirrel 抽象层保持兼容,可随时回退
2. 按模块迁移,不影响其他模块
3. 保留完整的测试用例,确保功能一致

---

## 九、参考资料

- [sqlc 官方文档](https://docs.sqlc.dev/)
- [goqu 官方文档](https://github.com/doug-martin/goqu)
- [Goose 迁移工具](https://github.com/pressly/goose)
- [PostgreSQL JSONB 类型](https://www.postgresql.org/docs/current/datatype-json.html)
- [TimescaleDB  hypertable](https://docs.timescale.com/use-timescale/latest/hypertables/)
- [Squirrel 维护状态](https://github.com/Masterminds/squirrel)

---

## 十、决策清单

请确认以下决策:

- [ ] 是否立即封装 Squirrel 抽象层?
- [ ] 是否引入 Goose 替代 golang-migrate?
- [ ] 是否在新模块中使用 sqlc?
- [ ] 是否开始迁移 PM/KPI 查询到 sqlc?
- [ ] 动态查询是否使用 goqu?
- [ ] 迁移优先级确认?

**决策人**: _____________  
**决策日期**: _____________

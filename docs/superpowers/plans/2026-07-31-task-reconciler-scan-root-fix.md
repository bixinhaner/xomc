# Task Reconciler Scan Root Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消除 `device_tasks` 活跃任务对账慢查询和固定最老 100 行导致的扫描饥饿。

**Architecture:** 仓储层提供 `(created_at,id)` 键集分页，对账器跨轮保存游标并在尾部回绕；数据库用仅覆盖 `pending/sent` 的部分索引同时满足过滤和排序。业务状态机与接口保持不变。

**Tech Stack:** Go 1.24、Squirrel、pgx、PostgreSQL 分区表、Testify。

## Global Constraints

- 单轮最多执行一次 `device_tasks` 活跃任务查询。
- 查询失败不得推进游标。
- 索引只包含 `pending`、`sent` 活跃态。
- 不修改任务状态机、重试和过期语义。
- 所有生产代码必须由旧实现上失败的测试驱动。

---

### Task 1: 仓储键集查询

**Files:**
- Modify: `omcgo/internal/task/pg_repository.go`
- Test: `omcgo/internal/task/pg_repository_test.go`

**Interfaces:**
- Produces: `ActiveTaskCursor{CreatedAt time.Time, ID string}`
- Produces: `ListActiveTasksAfter(context.Context, time.Time, *ActiveTaskCursor, int) ([]*Task, error)`
- Keeps: `ListActiveTasks(context.Context, time.Time, int) ([]*Task, error)`

- [ ] **Step 1: 写失败测试**

验证首批按 `(created_at,id)` 排序，传入首批末尾游标后只返回后续任务，并锁定 SQL
元组谓词 `(created_at, id) >`.

- [ ] **Step 2: 运行测试确认 RED**

Run: `cd omcgo && go test ./internal/task -run 'TestPgRepo_.*ListActiveTasksAfter' -count=1`

Expected: FAIL，因为 `ActiveTaskCursor` 和 `ListActiveTasksAfter` 尚不存在。

- [ ] **Step 3: 最小实现**

让 `ListActiveTasks` 委托给 `ListActiveTasksAfter(..., nil, ...)`；后者固定
`ORDER BY created_at ASC, id ASC`，非空游标增加元组比较。

- [ ] **Step 4: 运行目标测试确认 GREEN**

Run: `cd omcgo && go test ./internal/task -run 'TestPgRepo_.*ListActiveTasksAfter' -count=1`

Expected: PASS。

### Task 2: 对账器公平轮转

**Files:**
- Modify: `omcgo/internal/task/reconciler.go`
- Test: `omcgo/internal/task/reconciler_test.go`

**Interfaces:**
- Consumes: `ListActiveTasksAfter(context.Context, time.Time, *ActiveTaskCursor, int)`
- Produces: 对账器内部跨轮游标；尾部空批后清零。

- [ ] **Step 1: 写失败测试**

用记录游标的 fake lister 连续执行三轮：第一轮返回首批，第二轮接收首批末尾游标并返回
后续批，第三轮接收第二批末尾游标并返回空；断言第四轮从 nil 游标重新开始。另测查询
失败后游标保持原值。

- [ ] **Step 2: 运行测试确认 RED**

Run: `cd omcgo && go test ./internal/task -run 'Test_Reconciler_.*Cursor' -count=1`

Expected: FAIL，因为对账器仍调用无游标接口。

- [ ] **Step 3: 最小实现**

为 `Reconciler` 增加互斥保护的游标字段；每轮调用有游标查询，非空批推进，空批清零，
错误不改变游标。

- [ ] **Step 4: 运行目标测试确认 GREEN**

Run: `cd omcgo && go test ./internal/task -run 'Test_Reconciler_.*Cursor' -count=1`

Expected: PASS。

### Task 3: 活跃态部分索引

**Files:**
- Modify: `omcgo/migrations/000001_init_schema.sql`
- Test: `omcgo/cmd/migrate/task_active_index_migration_test.go`

**Interfaces:**
- Produces: `idx_device_tasks_active_created_id(created_at,id) WHERE status IN ('pending','sent')`

- [ ] **Step 1: 写失败迁移契约测试**

读取主 schema，断言索引名、列顺序和部分谓词；断言索引不包含终态。

- [ ] **Step 2: 运行测试确认 RED**

Run: `cd omcgo && go test ./cmd/migrate -run TestTaskActiveCreatedIndexMigration -count=1`

Expected: FAIL，因为索引尚不存在。

- [ ] **Step 3: 添加最小索引 DDL**

在 consolidated migrations 区域增加 `CREATE INDEX IF NOT EXISTS`，不改现有索引。

- [ ] **Step 4: 运行迁移测试确认 GREEN**

Run: `cd omcgo && go test ./cmd/migrate -run TestTaskActiveCreatedIndexMigration -count=1`

Expected: PASS。

### Task 4: 全量验证与交付

**Files:**
- Review: 本计划涉及的全部文件

- [ ] **Step 1: 格式与目标包**

Run: `cd omcgo && gofmt -w internal/task/pg_repository.go internal/task/pg_repository_test.go internal/task/reconciler.go internal/task/reconciler_test.go cmd/migrate/task_active_index_migration_test.go`

Run: `cd omcgo && go test ./internal/task ./cmd/migrate -count=1`

- [ ] **Step 2: 后端全量**

Run: `cd omcgo && go build ./...`

Run: `cd omcgo && go test ./... -count=1`

- [ ] **Step 3: 提交和 MR**

提交信息：`fix(perf): 根治任务对账扫描饥饿`

- [ ] **Step 4: 部署验证**

合入后从最新 main 构建部署；连续三个对账周期验证慢查询消失、Redis/PG 差异为 0、
任务队列年龄回落、HTTP 502/503/504 为 0。


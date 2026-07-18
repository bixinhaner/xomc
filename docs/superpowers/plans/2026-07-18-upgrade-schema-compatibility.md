# Upgrade Schema Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复旧 OMC 数据库升级后的参数同步 schema 漂移和设备坐标查询 SQL 别名错误。

**Architecture:** 用新的幂等 goose 迁移桥接旧库与当前基线，不改写已发布迁移历史；坐标查询沿用仓库统一的 `devices d` 软删除模式。先用回归测试证明两个故障，再做最小修改并在 78 上复测。

**Tech Stack:** Go 1.24、pgx、Squirrel、PostgreSQL 16、goose SQL migrations、Docker Compose

## Global Constraints

- 不回写或重编号已有迁移。
- SQL 继续使用 Squirrel + pgx。
- 所有升级 DDL 必须幂等。
- 部署后必须在持续 KPI 压测下验证。

---

### Task 1: 坐标查询 SQL 别名

**Files:**
- Modify: `omcgo/internal/device/device_repository.go`
- Test: `omcgo/internal/device/device_repository_test.go`

**Interfaces:**
- Consumes: `notDeleted = sq.Eq{"d.deleted_at": nil}`
- Produces: `PgDeviceRepository.GetCoordinates` 使用 `FROM devices d`

- [ ] **Step 1: 写回归测试**

增加测试构造与 `GetCoordinates` 相同的查询，断言 SQL 包含
`FROM devices d` 和 `d.deleted_at IS NULL`。

- [ ] **Step 2: 验证测试失败**

Run: `cd omcgo && go test ./internal/device -run TestGetCoordinatesQueryUsesDeviceAlias -count=1`
Expected: FAIL，实际 SQL 为 `FROM devices WHERE d.deleted_at IS NULL`。

- [ ] **Step 3: 最小修复**

把 `GetCoordinates` 的 `.From("devices")` 改为 `.From("devices d")`。

- [ ] **Step 4: 验证通过**

Run: `cd omcgo && go test ./internal/device -run TestGetCoordinatesQueryUsesDeviceAlias -count=1`
Expected: PASS。

### Task 2: 旧库 durable routing schema 桥接

**Files:**
- Create: `omcgo/migrations/000030_bridge_parameter_sync_routing_schema.sql`
- Test: `omcgo/test/integration/parameter_sync_routing_migration_test.go`

**Interfaces:**
- Consumes: `parameter_sync_requests`、`parameter_sync_runs`、`device_tasks`
- Produces: durable routing 字段、admission/recovery/model-upload 表及索引

- [ ] **Step 1: 写迁移契约测试**

断言 migration 30 存在，包含 `ADD COLUMN IF NOT EXISTS source_event_id`、
`CREATE TABLE IF NOT EXISTS model_upload_intents` 和关键 admission/recovery 表。

- [ ] **Step 2: 验证测试失败**

Run: `cd omcgo && go test ./test/integration -run TestBridgeParameterSyncRoutingSchema -count=1`
Expected: FAIL，migration 30 不存在。

- [ ] **Step 3: 编写幂等迁移**

使用 `ADD COLUMN IF NOT EXISTS`、`CREATE TABLE IF NOT EXISTS` 和
`CREATE INDEX IF NOT EXISTS` 补齐当前运行代码访问的 schema；Down 保持非破坏性。

- [ ] **Step 4: 验证迁移契约通过**

Run: `cd omcgo && go test ./test/integration -run TestBridgeParameterSyncRoutingSchema -count=1`
Expected: PASS。

### Task 3: 全量验证、提交和复测

**Files:**
- Verify: `omcgo/...`
- Deploy: `deployments/release/...`

**Interfaces:**
- Consumes: Tasks 1–2 的代码与迁移
- Produces: 可部署发布包和 78 实测数据

- [ ] **Step 1: 后端全量验证**

Run: `cd omcgo && go build ./... && go test ./...`
Expected: exit 0。

- [ ] **Step 2: 提交并推送 MR 分支**

使用 Conventional Commit 提交，推送到现有 MR !201。

- [ ] **Step 3: 构建部署并重启**

生成新 test 发布包，部署到 78，重启完整 compose 服务集合。

- [ ] **Step 4: 验证现网**

检查 migration 30、六个关键健康端点、两类错误日志，并在至少两个采样窗口中比较
IO PSI、NATS backlog、PM ACK 和 PostgreSQL 热点。

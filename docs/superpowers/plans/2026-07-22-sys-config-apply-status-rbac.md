# Built-in API Permission Drift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复严格 API RBAC 启用后新增路由不具备内置角色基线授权的问题，并阻止未来再次漂移。

**Architecture:** 增加 seed 增量修复当前数据库；在路由同步服务中注入内置角色权限对账器，每次自动或手工同步路由后幂等补齐 admin/operator 全部端点和 viewer GET 端点。对账仅触碰固定内置角色，实际新增授权后刷新本实例 Casbin 并广播其他实例。

**Tech Stack:** Go 1.24、Gin、pgx、Squirrel、Casbin、PostgreSQL 16、goose、testify/gomock。

## Global Constraints

- 保留 `(path, method)` fail-closed 权限校验。
- admin/operator 补齐全部已登记端点，viewer 只补齐 GET。
- 不修改任何自定义角色授权。
- 所有 SQL 参数化且幂等，不删除现有授权。
- 不混入文件传输、PM、ACS 或运营商识别修改。

---

### Task 1: 用失败测试锁定持续对账契约

**Files:**
- Modify: `omcgo/internal/admin/api_endpoint_service_test.go`
- Create: `omcgo/internal/admin/pg_role_repository_builtin_permissions_test.go`

**Interfaces:**
- Produces: `BuiltInAPIPermissionReconciler.ReconcileBuiltInAPIPermissions(context.Context)`。
- Produces: `BuiltInAPIPermissionGrantResult`，包含 admin/operator/viewer 的新增数量。

- [ ] **Step 1: 编写 service 失败测试**

覆盖路由 Upsert 完成后调用对账器，以及对账失败时 `SyncApiEndpoints` 返回错误。

- [ ] **Step 2: 编写 repository 失败测试**

覆盖三条基线插入、事务提交、仅新增时刷新策略、持久化失败时不刷新。

- [ ] **Step 3: 验证 RED**

Run: `cd omcgo && go test ./internal/admin -run 'TestSyncApiEndpointsReconcilesBuiltInPermissions|TestReconcileBuiltInAPIPermissions' -count=1`

Expected: FAIL，因为对账接口和实现尚不存在。

### Task 2: 实现最小运行时对账

**Files:**
- Modify: `omcgo/internal/admin/api_endpoint_service.go`
- Modify: `omcgo/internal/admin/pg_role_repository.go`
- Modify: `omcgo/cmd/app/provider/admin.go`
- Modify: `omcgo/cmd/app/provider/router.go`

**Interfaces:**
- `SetBuiltInPermissionReconciler(reconciler BuiltInAPIPermissionReconciler)` 注入角色仓储。
- `ReconcileBuiltInAPIPermissions(ctx) (BuiltInAPIPermissionGrantResult, error)` 幂等补齐权限并刷新 policy。

- [ ] **Step 1: 实现 repository 对账**

在单事务中分别对 admin、operator、viewer 执行参数化 `INSERT ... SELECT ... ON CONFLICT DO NOTHING`，viewer 增加 `method = 'GET'` 条件。

- [ ] **Step 2: 接通 service 与 provider**

路由同步结束后调用对账器；启动自动同步失败时由 `Setup` 返回错误，不再仅记录 warning。

- [ ] **Step 3: 验证 GREEN**

Run: `cd omcgo && go test ./internal/admin ./cmd/app/provider -run 'TestSyncApiEndpointsReconcilesBuiltInPermissions|TestReconcileBuiltInAPIPermissions|TestSyncApiEndpoints' -count=1`

Expected: PASS。

### Task 3: 增加存量 seed 修复

**Files:**
- Create: `omcgo/migrations/seed/000002_repair_builtin_role_api_permission_drift.sql`
- Create: `omcgo/test/integration/builtin_role_api_permission_seed_test.go`

**Interfaces:**
- Consumes: `api_endpoints`、`roles`、`role_api_permissions`。
- Produces: 当前数据库三个内置角色的完整基线授权。

- [ ] **Step 1: 编写 seed 契约失败测试**

断言三个固定 UUID、viewer GET 条件、`ON CONFLICT`、goose Up/Down 和无破坏回滚。

- [ ] **Step 2: 验证 RED**

Run: `cd omcgo && go test ./test/integration -run TestBuiltInRoleAPIPermissionSeedContract -count=1`

Expected: FAIL，因为 `000002` 尚不存在。

- [ ] **Step 3: 编写 migration 并验证 GREEN**

Run: `cd omcgo && go test ./test/integration -run TestBuiltInRoleAPIPermissionSeedContract -count=1`

Expected: PASS。

### Task 4: 数据库与全量验证

**Files:**
- Verify only.

**Interfaces:**
- Consumes: Tasks 1-3。
- Produces: 编译、测试、迁移和权限矩阵证据。

- [ ] **Step 1: 执行定向测试与编译**

Run: `cd omcgo && go test ./internal/admin ./cmd/app/provider ./test/integration -count=1`

Run: `cd omcgo && go build ./...`

Expected: 全部 PASS / exit 0。

- [ ] **Step 2: 在本地 PostgreSQL 应用 seed 并启动应用**

通过 compose 的 `migrate-seed` 应用 `000002`，重启应用让启动同步和 Casbin 刷新生效。

- [ ] **Step 3: 查询业务不变量**

Expected:

- admin 缺失端点数为 0。
- operator 缺失端点数为 0。
- viewer 缺失 GET 端点数为 0。
- 自定义角色授权数量前后不变。
- `GET /api/v1/admin/sysConfig/apply-batches/:id` 对 admin/operator/viewer 均有授权。

- [ ] **Step 4: 执行全量测试并复核 diff**

Run: `cd omcgo && go test ./...`

Expected: PASS；若存在与本修改无关的既有失败，记录完整命令和错误，不扩大修复范围。


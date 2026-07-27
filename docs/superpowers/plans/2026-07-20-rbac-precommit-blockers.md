# RBAC 提交阻断项修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复内置角色 API 权限缺失、权限写入后策略不立即生效以及 Casbin 并发刷新数据竞争这 3 个提交阻断项。

**Architecture:** 使用独立 seed migration 恢复历史内置角色权限基线；让 `PgRoleRepository` 在权限数据写成功后同步 reload 当前 Casbin 实例并广播其他实例；用 Casbin `SyncedEnforcer` 统一保护鉴权读和 policy reload 写。测试分为 SQL migration 契约/数据库行为、仓储刷新顺序与错误传播、Casbin 定向 race 三层。

**Tech Stack:** Go 1.x、PostgreSQL、Goose、pgx v5、Casbin v2.135、Testify、Go race detector。

## Global Constraints

- 在当前仓库 `/Users/hezg/Documents/work/xomc` 和当前分支 `codex/fix-system-config-security-p0` 原地修改，不创建 worktree 或额外项目目录。
- admin 获得全部已登记 API，operator 获得全部已登记 API，viewer 获得全部 GET API。
- `builtIn` 用户旁路规则和现有 Casbin model 语义保持不变。
- policy 数据写成功后必须先同步 reload 当前实例，再广播其他实例。
- reload 或广播失败不得静默忽略，错误必须说明数据已经持久化。
- watcher bus 为 nil 时保留单实例 no-op broadcast 兼容行为。
- 不修改 consolidated `000001_init_seed.sql`，只新增 `000002`。
- 不处理与这 3 项无关的既有前端 lint、Vitest 和 AuditLogger race 问题。
- 严格执行 RED → GREEN → REFACTOR。
- 不暂存、不提交、不推送；全部验证完成后等待用户评审。

---

### Task 1: 恢复内置角色 API 权限基线

**Files:**
- Create: `omcgo/test/integration/rbac_seed_repair_test.go`
- Create: `omcgo/migrations/seed/000002_repair_builtin_role_api_permissions.sql`

**Interfaces:**
- Consumes: `roles(id)`、`api_endpoints(id, method)`、`role_api_permissions(role_id, endpoint_id)`
- Produces: 可幂等执行的 Goose seed migration `000002`

- [ ] **Step 1: 写 migration 契约失败测试**

新增 `rbac_seed_repair_test.go`。静态测试必须读取 `000002`，检查三个固定角色 UUID、admin/operator 全量选择、viewer GET 选择、三处 `ON CONFLICT` 和无破坏 Down：

```go
package integration

import (
    "context"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const (
    adminRoleID    = "10000000-0000-0000-0000-000000000001"
    operatorRoleID = "10000000-0000-0000-0000-000000000002"
    viewerRoleID   = "10000000-0000-0000-0000-000000000003"
)

func rbacRepairSeedPath() string {
    return filepath.Join("..", "..", "migrations", "seed", "000002_repair_builtin_role_api_permissions.sql")
}

func readRBACRepairSeed(t *testing.T) string {
    t.Helper()
    raw, err := os.ReadFile(rbacRepairSeedPath())
    require.NoError(t, err)
    return string(raw)
}

func gooseUpSQL(t *testing.T, raw string) string {
    t.Helper()
    upMarker := "-- +goose Up"
    downMarker := "-- +goose Down"
    up := strings.Index(raw, upMarker)
    down := strings.Index(raw, downMarker)
    require.GreaterOrEqual(t, up, 0)
    require.Greater(t, down, up)
    return raw[up+len(upMarker) : down]
}

func TestRBACRepairSeedContract(t *testing.T) {
    raw := readRBACRepairSeed(t)
    up := gooseUpSQL(t, raw)

    assert.Contains(t, up, adminRoleID)
    assert.Contains(t, up, operatorRoleID)
    assert.Contains(t, up, viewerRoleID)
    assert.Equal(t, 3, strings.Count(strings.ToUpper(up), "ON CONFLICT (ROLE_ID, ENDPOINT_ID) DO NOTHING"))
    assert.Contains(t, up, "SELECT '"+adminRoleID+"'::uuid, ae.id")
    assert.Contains(t, up, "SELECT '"+operatorRoleID+"'::uuid, ae.id")
    assert.Contains(t, up, "SELECT '"+viewerRoleID+"'::uuid, ae.id")
    assert.Contains(t, up, "WHERE ae.method = 'GET'")

    down := raw[strings.Index(raw, "-- +goose Down"):]
    assert.Contains(t, down, "SELECT 1;")
    assert.NotContains(t, strings.ToUpper(down), "DELETE FROM")
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./test/integration -run TestRBACRepairSeedContract -count=1
```

Expected: FAIL，错误为无法读取 `000002_repair_builtin_role_api_permissions.sql`。

- [ ] **Step 3: 新增最小增量 seed migration**

创建：

```sql
-- +goose Up

-- admin：兼容历史行为，授权全部已登记 API。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, ae.id
FROM public.api_endpoints AS ae
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- operator：兼容历史行为，授权全部已登记 API。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, ae.id
FROM public.api_endpoints AS ae
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- viewer：只补齐只读 API，不授予写操作。
INSERT INTO public.role_api_permissions (role_id, endpoint_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid, ae.id
FROM public.api_endpoints AS ae
WHERE ae.method = 'GET'
ON CONFLICT (role_id, endpoint_id) DO NOTHING;

-- +goose Down
-- 数据修复无法安全区分既有授权与本 migration 新增授权，禁止破坏性回滚。
SELECT 1;
```

- [ ] **Step 4: 运行静态契约测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./test/integration -run TestRBACRepairSeedContract -count=1
```

Expected: PASS。

- [ ] **Step 5: 增加真实 PostgreSQL 行为测试**

在同一测试文件增加定向数据库测试。它在事务中插入一个 GET 和一个 POST fixture，执行 Up 两次，并回滚全部测试数据：

```go
func TestRBACRepairSeedDatabaseBehavior(t *testing.T) {
    pool := SetupTestDB(t)
    ctx := context.Background()
    tx, err := pool.Begin(ctx)
    require.NoError(t, err)
    defer tx.Rollback(ctx)

    roles := []struct {
        id, name string
    }{
        {adminRoleID, "admin"},
        {operatorRoleID, "operator"},
        {viewerRoleID, "viewer"},
    }
    for _, role := range roles {
        _, err = tx.Exec(ctx, `
            INSERT INTO roles (id, name, description, is_system)
            VALUES ($1, $2, 'RBAC seed repair test fixture', true)
            ON CONFLICT (id) DO NOTHING`,
            role.id, role.name)
        require.NoError(t, err)
    }

    fixtures := []struct {
        id, path, method string
    }{
        {"7f200000-0000-0000-0000-000000000001", "/test/rbac-seed/get", "GET"},
        {"7f200000-0000-0000-0000-000000000002", "/test/rbac-seed/post", "POST"},
    }
    for _, fixture := range fixtures {
        _, err = tx.Exec(ctx, `
            INSERT INTO api_endpoints (id, path, method, name, description, api_group)
            VALUES ($1, $2, $3, $4, '', 'test')
            ON CONFLICT (id) DO UPDATE SET method = EXCLUDED.method`,
            fixture.id, fixture.path, fixture.method, fixture.method+" test fixture")
        require.NoError(t, err)
    }

    up := gooseUpSQL(t, readRBACRepairSeed(t))
    _, err = tx.Exec(ctx, up)
    require.NoError(t, err)
    _, err = tx.Exec(ctx, up)
    require.NoError(t, err)

    assertions := []struct {
        roleID, endpointID string
        want               int
    }{
        {adminRoleID, fixtures[0].id, 1},
        {adminRoleID, fixtures[1].id, 1},
        {operatorRoleID, fixtures[0].id, 1},
        {operatorRoleID, fixtures[1].id, 1},
        {viewerRoleID, fixtures[0].id, 1},
        {viewerRoleID, fixtures[1].id, 0},
    }
    for _, assertion := range assertions {
        var got int
        err = tx.QueryRow(ctx, `
            SELECT count(*) FROM role_api_permissions
            WHERE role_id = $1 AND endpoint_id = $2`,
            assertion.roleID, assertion.endpointID).Scan(&got)
        require.NoError(t, err)
        assert.Equal(t, assertion.want, got)
    }
}
```

角色与 endpoint fixture 都必须在测试事务内创建，使测试在只应用 `000001_init_schema.sql`、未加载 baseline seed 的标准集成测试库中也能独立运行。

- [ ] **Step 6: 运行 migration 测试**

Run:

```bash
cd omcgo
go test ./test/integration -run 'TestRBACRepairSeed' -count=1
```

Expected: 静态测试 PASS；未配置 `OMCGO_TEST_DB_DSN` 时数据库测试明确 SKIP。配置测试数据库时两项均 PASS。

---

### Task 2: 权限数据写入后同步刷新 Casbin

**Files:**
- Modify: `omcgo/internal/admin/pg_role_repository.go`
- Create: `omcgo/internal/admin/pg_role_repository_policy_refresh_test.go`

**Interfaces:**
- Consumes: `storage.DB`、`PolicyRefresher.ReloadPolicy() error`、`PolicyRefresher.NotifyPolicyChange() error`
- Produces: `PgRoleRepository.refreshPolicyAfterPersist() error`

- [ ] **Step 1: 写三个写入口的刷新失败测试**

测试文件定义可记录 DB、transaction 和 refresher。通过嵌入接口，只覆盖被测路径实际调用的方法：

```go
type recordingPolicyRefresher struct {
    calls     []string
    reloadErr error
    notifyErr error
}

func (r *recordingPolicyRefresher) ReloadPolicy() error {
    r.calls = append(r.calls, "reload")
    return r.reloadErr
}

func (r *recordingPolicyRefresher) NotifyPolicyChange() error {
    r.calls = append(r.calls, "notify")
    return r.notifyErr
}

type policyTestDB struct {
    storage.DB
    execTag pgconn.CommandTag
    execErr error
    tx      pgx.Tx
    beginErr error
}

func (d *policyTestDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
    return d.execTag, d.execErr
}

func (d *policyTestDB) Begin(context.Context) (pgx.Tx, error) {
    return d.tx, d.beginErr
}

type policyTestTx struct {
    pgx.Tx
    calls     []string
    execErr   error
    commitErr error
}

func (tx *policyTestTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
    tx.calls = append(tx.calls, "exec")
    return pgconn.NewCommandTag("INSERT 0 1"), tx.execErr
}

func (tx *policyTestTx) Commit(context.Context) error {
    tx.calls = append(tx.calls, "commit")
    return tx.commitErr
}

func (tx *policyTestTx) Rollback(context.Context) error {
    tx.calls = append(tx.calls, "rollback")
    return nil
}
```

增加表驱动测试：

```go
func TestPgRoleRepositoryPolicyWritesRefreshCurrentThenPeers(t *testing.T) {
    userID, roleID, endpointID := uuid.New(), uuid.New(), uuid.New()
    cases := []struct {
        name string
        run  func(*PgRoleRepository) error
        db   *policyTestDB
    }{
        {
            name: "assign role",
            run: func(repo *PgRoleRepository) error {
                return repo.AssignRole(context.Background(), userID, roleID)
            },
            db: &policyTestDB{execTag: pgconn.NewCommandTag("INSERT 0 1")},
        },
        {
            name: "remove role",
            run: func(repo *PgRoleRepository) error {
                return repo.RemoveRole(context.Background(), userID, roleID)
            },
            db: &policyTestDB{execTag: pgconn.NewCommandTag("DELETE 1")},
        },
        {
            name: "set role endpoints",
            run: func(repo *PgRoleRepository) error {
                return repo.SetRoleApiEndpoints(context.Background(), roleID, []uuid.UUID{endpointID})
            },
            db: func() *policyTestDB {
                tx := &policyTestTx{}
                return &policyTestDB{tx: tx}
            }(),
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            refresher := &recordingPolicyRefresher{}
            repo := &PgRoleRepository{pool: tc.db, policyRefresher: refresher}
            require.NoError(t, tc.run(repo))
            assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
        })
    }
}
```

再覆盖失败语义：

```go
func TestPgRoleRepositoryPolicyRefreshErrorsAreReturned(t *testing.T) {
    persistedDB := &policyTestDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}

    t.Run("reload failure stops notify", func(t *testing.T) {
        refresher := &recordingPolicyRefresher{reloadErr: errors.New("reload failed")}
        repo := &PgRoleRepository{pool: persistedDB, policyRefresher: refresher}
        err := repo.AssignRole(context.Background(), uuid.New(), uuid.New())
        require.ErrorContains(t, err, "persisted")
        require.ErrorContains(t, err, "reload")
        assert.Equal(t, []string{"reload"}, refresher.calls)
    })

    t.Run("notify failure is returned", func(t *testing.T) {
        refresher := &recordingPolicyRefresher{notifyErr: errors.New("publish failed")}
        repo := &PgRoleRepository{pool: persistedDB, policyRefresher: refresher}
        err := repo.AssignRole(context.Background(), uuid.New(), uuid.New())
        require.ErrorContains(t, err, "persisted")
        require.ErrorContains(t, err, "notify")
        assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
    })
}
```

增加 DB mutation、begin、transaction exec 和 commit 失败均不触发 refresher 的断言。

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test ./internal/admin -run 'TestPgRoleRepositoryPolicy' -count=1
```

Expected: build FAIL，因为 `PgRoleRepository.pool` 仍是 `*pgxpool.Pool`、`authorizer` 仍是 `*CasbinAuthorizer`，且成功路径当前不会按 `reload, notify` 调用。

- [ ] **Step 3: 引入最小接口并实现统一刷新**

把鉴权读取和策略刷新拆成两个最小能力字段：

```go
type PolicyRefresher interface {
    ReloadPolicy() error
    NotifyPolicyChange() error
}

type apiPermissionChecker interface {
    CheckPermission(context.Context, uuid.UUID, string, string) (bool, error)
}

type PgRoleRepository struct {
    pool              storage.DB
    permissionChecker apiPermissionChecker
    policyRefresher   PolicyRefresher
}
```

保持生产构造函数签名不变：

```go
func NewPgRoleRepository(pool *pgxpool.Pool) *PgRoleRepository {
    return &PgRoleRepository{pool: storage.NewPoolDB(pool)}
}

func (r *PgRoleRepository) SetAuthorizer(auth *CasbinAuthorizer) {
    r.permissionChecker = auth
    r.policyRefresher = auth
}
```

新增统一 helper：

```go
func (r *PgRoleRepository) refreshPolicyAfterPersist() error {
    if r.policyRefresher == nil {
        return nil
    }
    if err := r.policyRefresher.ReloadPolicy(); err != nil {
        return fmt.Errorf("policy data persisted but reload current policy: %w", err)
    }
    if err := r.policyRefresher.NotifyPolicyChange(); err != nil {
        return fmt.Errorf("policy data persisted and local policy reloaded but notify peers: %w", err)
    }
    return nil
}
```

在 `AssignRole`、`RemoveRole` 的 DB 成功路径末尾返回该 helper；删除 `_ = NotifyPolicyChange()`：

```go
return r.refreshPolicyAfterPersist()
```

在 `SetRoleApiEndpoints` 中只在 commit 成功后刷新：

```go
if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit role api permissions: %w", err)
}
return r.refreshPolicyAfterPersist()
```

- [ ] **Step 4: 运行定向测试确认 GREEN**

Run:

```bash
cd omcgo
go test ./internal/admin -run 'TestPgRoleRepositoryPolicy' -count=1
```

Expected: 全部 PASS。

- [ ] **Step 5: 重构并验证 admin 包**

Run:

```bash
cd omcgo
gofmt -w internal/admin/pg_role_repository.go internal/admin/pg_role_repository_policy_refresh_test.go
go test ./internal/admin -count=1
```

Expected: PASS，生产 provider 中 `repo.SetAuthorizer(authorizer)` 类型匹配。

---

### Task 3: Casbin 鉴权与刷新并发安全

**Files:**
- Modify: `omcgo/internal/admin/casbin.go`
- Modify: `omcgo/internal/admin/casbin_test.go`

**Interfaces:**
- Consumes: `casbin.NewSyncedEnforcer(model.Model, ...interface{})`
- Produces: `CasbinAuthorizer.enforcer *casbin.SyncedEnforcer`

- [ ] **Step 1: 写并发 reload 失败测试**

先把测试 helper 的返回类型和构造改为 synced enforcer，迫使生产字段类型跟进：

```go
func newTestEnforcer(t *testing.T) *casbin.SyncedEnforcer {
    t.Helper()
    m, err := casbinModel.NewModelFromFile(testModelPath)
    require.NoError(t, err)
    e, err := casbin.NewSyncedEnforcer(m)
    require.NoError(t, err)
    return e
}
```

增加一个带互斥锁的可变 adapter：

```go
type concurrentPolicyAdapter struct {
    mu    sync.RWMutex
    allow bool
}

func (a *concurrentPolicyAdapter) LoadPolicy(m casbinModel.Model) error {
    a.mu.RLock()
    allow := a.allow
    a.mu.RUnlock()
    if allow {
        persist.LoadPolicyLine("p, role:viewer, system, /api/v1/admin/sysConfig, GET", m)
    }
    persist.LoadPolicyLine("g, 20000000-0000-0000-0000-000000000001, role:viewer, system", m)
    return nil
}

func (a *concurrentPolicyAdapter) SavePolicy(casbinModel.Model) error { return nil }
func (a *concurrentPolicyAdapter) AddPolicy(string, string, []string) error { return nil }
func (a *concurrentPolicyAdapter) RemovePolicy(string, string, []string) error { return nil }
func (a *concurrentPolicyAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
    return nil
}

func (a *concurrentPolicyAdapter) toggle() {
    a.mu.Lock()
    a.allow = !a.allow
    a.mu.Unlock()
}
```

并发测试使用生产字段和真实 `ReloadPolicy` / `CheckPermission`：

```go
func TestCasbinAuthorizerConcurrentEnforceAndReload(t *testing.T) {
    adapter := &concurrentPolicyAdapter{allow: true}
    m, err := casbinModel.NewModelFromFile(testModelPath)
    require.NoError(t, err)
    enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
    require.NoError(t, err)

    auth := &CasbinAuthorizer{enforcer: enforcer, logger: zap.NewNop()}
    userID := uuid.MustParse("20000000-0000-0000-0000-000000000001")

    var wg sync.WaitGroup
    for range 8 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for range 500 {
                _, enforceErr := auth.CheckPermission(
                    context.Background(),
                    userID,
                    "/api/v1/admin/sysConfig",
                    "GET",
                )
                assert.NoError(t, enforceErr)
            }
        }()
    }
    wg.Add(1)
    go func() {
        defer wg.Done()
        for range 200 {
            adapter.toggle()
            assert.NoError(t, auth.ReloadPolicy())
        }
    }()
    wg.Wait()
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run:

```bash
cd omcgo
go test -race ./internal/admin -run TestCasbinAuthorizerConcurrentEnforceAndReload -count=1
```

Expected: build FAIL，因为 `CasbinAuthorizer.enforcer` 仍要求 `*casbin.Enforcer`。如果先只加入并发测试而不改 helper，则普通 enforcer 版本应由 race detector 报告 Casbin model 的并发读写。

- [ ] **Step 3: 替换为 SyncedEnforcer**

修改生产字段：

```go
type CasbinAuthorizer struct {
    enforcer *casbin.SyncedEnforcer
    adapter  *pgAdapter
    watcher  persist.Watcher
    logger   *zap.Logger
}
```

修改生产构造：

```go
enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
if err != nil {
    return nil, fmt.Errorf("create casbin synced enforcer: %w", err)
}
```

watcher callback、`CheckPermission`、`ReloadPolicy` 和周期刷新继续调用相同公开方法，不增加额外锁。

- [ ] **Step 4: 运行定向 race 确认 GREEN**

Run:

```bash
cd omcgo
gofmt -w internal/admin/casbin.go internal/admin/casbin_test.go
go test -race ./internal/admin -run TestCasbinAuthorizerConcurrentEnforceAndReload -count=1
```

Expected: PASS，且无 `WARNING: DATA RACE`。

- [ ] **Step 5: 运行全部 Casbin 测试**

Run:

```bash
cd omcgo
go test ./internal/admin -run 'Test.*(Casbin|CheckPermission|Watcher)' -count=1
```

Expected: PASS。

---

### Task 4: 全量验证并更新提交前审查报告

**Files:**
- Modify: `docs/review-report/20260720/FIX_71cdbad6_admin-rbac-public-config.md`
- Modify: `docs/superpowers/plans/2026-07-20-rbac-precommit-blockers.md`

**Interfaces:**
- Consumes: Task 1–3 的代码和测试输出
- Produces: 可供用户评审的逐项修改、验证证据和剩余风险清单

- [ ] **Step 1: 运行格式与差异检查**

Run:

```bash
gofmt -w omcgo/internal/admin/casbin.go omcgo/internal/admin/casbin_test.go omcgo/internal/admin/pg_role_repository.go omcgo/internal/admin/pg_role_repository_policy_refresh_test.go omcgo/test/integration/rbac_seed_repair_test.go
git diff --check
```

Expected: 无输出、退出码 0。

- [ ] **Step 2: 运行后端定向验证**

Run:

```bash
cd omcgo
go test ./test/integration -run 'TestRBACRepairSeed' -count=1
go test ./internal/admin -run 'Test.*(Policy|RoleApi|Casbin|CheckPermission|Watcher)' -count=1
go test -race ./internal/admin -run TestCasbinAuthorizerConcurrentEnforceAndReload -count=1
```

Expected: 全部 PASS；数据库 migration 行为测试在无 DSN 时允许明确 SKIP。

- [ ] **Step 3: 运行后端全量验证**

Run:

```bash
cd omcgo
go vet ./...
go build ./...
go test ./...
```

Expected: 三条命令均退出码 0。

- [ ] **Step 4: 复核原分支已有前端修改**

Run:

```bash
cd omcmb
npm run typecheck
npm run build --workspace webcode
```

Expected: 两条命令均退出码 0。若结果变化，记录完整命令和错误，不把既有失败归到本次后端修复。

- [ ] **Step 5: 更新审查报告**

在报告中按以下结构记录：

```markdown
## 提交阻断项关闭记录

| 阻断项 | 修改 | 测试证据 | 状态 |
|---|---|---|---|
| 内置角色权限缺失 | seed `000002`：admin/operator 全 API，viewer 全 GET | migration contract + PostgreSQL transaction test | 已关闭 |
| policy 写入后不立即生效 | 三个写入口 commit 后同步 reload，再 broadcast，错误不再忽略 | repository success/failure tests | 已关闭 |
| Casbin 并发读写 | `SyncedEnforcer` | 定向 `go test -race` | 已关闭 |

### 非本次范围的既有问题

- 全量前端 lint 的历史错误；
- 全量 Vitest 的历史失败；
- AuditLogger 全量 race；
- 安全设置页 `refetchConfiguredState` 未传播 refetch error。
```

只有实际执行并通过的测试才能填写为 PASS；SKIP 和既有失败必须单列。

- [ ] **Step 6: 最终工作区审计**

Run:

```bash
git status --short --branch
git diff --stat
git diff --check
git diff -- omcgo/migrations/seed/000002_repair_builtin_role_api_permissions.sql omcgo/internal/admin/casbin.go omcgo/internal/admin/pg_role_repository.go
```

Expected:

- 分支仍为 `codex/fix-system-config-security-p0`；
- 本次文件清晰可见；
- 没有 staged changes；
- 没有新 commit 或 push；
- 原有用户修改未被回滚。

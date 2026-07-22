# System Config Upload Contract Regression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复系统配置 MR 引入的设备上传 URL 参数顺序回归和配置应用状态接口 RBAC 缺口。

**Architecture:** 下载 URL 继续使用结构化参数编码；设备上传模板使用独立构造入口，在完成基础地址和路径安全校验后原样保留业务查询串。RBAC 通过独立、幂等的 seed 增量在应用启动前注册端点并授予现有内置角色。

**Tech Stack:** Go 1.24、`net/url`、Gin、PostgreSQL、goose、testify。

## Global Constraints

- `filename=`/`fileName=` 必须保持为设备上传 URL 的最后部分。
- 生产环境允许设备网络可达的 RFC1918 地址。
- 不修改与两个系统配置 MR 无直接关系的功能。
- 所有生产代码修改必须先有会因当前回归而失败的测试。

---

### Task 1: Preserve device upload template query order

**Files:**
- Modify: `omcgo/internal/acs/transfercfg/url_builder.go`
- Modify: `omcgo/internal/acs/transfercfg/url_builder_test.go`
- Modify: `omcgo/internal/software/executor.go`
- Modify: `omcgo/internal/software/transfer_url_test.go`
- Modify: `omcgo/internal/backup/executor.go`
- Modify: `omcgo/internal/backup/executor_test.go`

**Interfaces:**
- Consumes: existing `ValidateBaseURL(string) error` and `ValidateServicePath(string) error`.
- Produces: `BuildTemplateURL(baseURL, relativeReference string) (string, error)` preserving `relativeReference` query bytes and order.

- [ ] **Step 1: Add failing exact-contract tests**

Add assertions for these literal suffixes:

```go
require.Equal(t,
    "https://edge.example.com/omc/smallcell/FileUploadService?fileType=LOG&sn=SN100&taskId=abc123&filename=",
    got,
)
require.Equal(t,
    "https://edge.example.com/omc/smallcell/FileUploadService?fileType=RL&id=task-1&sn=SN100&fileName=",
    got,
)
require.Equal(t,
    "https://edge.example.com/omc/smallcell/FileUploadService?fileType=CONFIGBACKUP_XML&sn=SN100&taskId=task-1&filename=backup.xml",
    got,
)
```

- [ ] **Step 2: Verify the tests fail for parameter reordering**

Run:

```bash
cd omcgo && go test ./internal/software ./internal/backup ./internal/acs/transfercfg
```

Expected: FAIL because current `url.Values.Encode()` moves `filename`/`fileName` before `sn` and `taskId`.

- [ ] **Step 3: Implement the upload-template URL builder**

Implement `BuildTemplateURL` by parsing the relative reference, rejecting an absolute host/user/fragment and dot-segment path, joining its escaped path with the validated base prefix, and assigning the original `RawQuery` without re-encoding it. Keep `BuildURL` unchanged for structured download URLs.

Update software task URL construction to call `BuildTemplateURL(baseURL, resolvedTransport)`. Update configuration backup construction to create the ordered query with `url.QueryEscape` in the exact order `fileType`, `sn`, `taskId`, `filename`, then call `BuildTemplateURL`.

- [ ] **Step 4: Verify focused tests pass**

Run:

```bash
cd omcgo && go test ./internal/software ./internal/backup ./internal/acs/transfercfg
```

Expected: PASS.

### Task 2: Grant config apply-status endpoint to built-in roles

**Files:**
- Create: `omcgo/migrations/seed/000002_grant_sys_config_apply_status.sql`
- Create: `omcgo/test/integration/sys_config_apply_status_seed_test.go`

**Interfaces:**
- Consumes: `api_endpoints(id, path, method, ...)`, `roles(id, name)`, and `role_api_permissions(role_id, endpoint_id)`.
- Produces: an idempotently registered `GET /api/v1/admin/sysConfig/apply-batches/:id` endpoint and grants for `admin`, `operator`, and `viewer`.

- [ ] **Step 1: Add a failing migration contract test**

Read the new migration path and assert it contains the exact route, GET method, all three built-in role names, and `ON CONFLICT DO NOTHING`. The test initially fails because the migration file does not exist.

- [ ] **Step 2: Verify the migration test fails**

Run:

```bash
cd omcgo && go test ./test/integration -run TestSysConfigApplyStatusSeedContract
```

Expected: FAIL reading `000002_grant_sys_config_apply_status.sql`.

- [ ] **Step 3: Add the idempotent seed migration**

Use a stable UUID to insert the exact endpoint with `ON CONFLICT (path, method) DO NOTHING`, then insert role/endpoint pairs selected by role names `admin`, `operator`, and `viewer`, also with `ON CONFLICT DO NOTHING`. The Down section removes only those three grants and the endpoint created by this migration when safe.

- [ ] **Step 4: Verify migration contract and focused packages**

Run:

```bash
cd omcgo && go test ./test/integration -run TestSysConfigApplyStatusSeedContract
```

Expected: PASS.

### Task 3: Regression verification

**Files:**
- Verify only; no additional production changes.

**Interfaces:**
- Consumes: Task 1 URL contract and Task 2 migration contract.
- Produces: build/test evidence for handoff.

- [ ] **Step 1: Format modified Go files**

Run `gofmt -w` on the exact modified Go source and test files.

- [ ] **Step 2: Run focused tests**

```bash
cd omcgo && go test ./internal/acs/transfercfg ./internal/software ./internal/backup ./test/integration
```

Expected: PASS, with database-dependent integration cases skipped only when `OMCGO_TEST_DB_DSN` is absent.

- [ ] **Step 3: Run backend build and full tests**

```bash
cd omcgo && go build ./... && go test ./...
```

Expected: PASS. If a test needs local listening permission, rerun that command with the documented elevated sandbox permission.

- [ ] **Step 4: Review final diff**

Confirm the diff contains only the upload-template contract fix, its tests, the RBAC seed migration, its test, and these approved design/plan documents. Confirm `AGENTS.md` and unrelated untracked files are absent.


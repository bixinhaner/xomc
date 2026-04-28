# T-0057.5 verify — topology errors.NotFound / 约束违反映射

- 任务：T-0057 / W2.D.1.b 真 bug #7 — `POST /api/v1/sites` 在 `domain_id` 触发外键违反时返回 500，应映射为 400/409。
- Worktree：`.claude/worktrees/agent-aa37a79c`
- 分支：`worktree-agent-aa37a79c`
- 章程关联：T-0056 verify-md §3 真 bug #7（POST /sites）

## 1. 根因

`omcgo/internal/topology/site_pg_repository.go::PgSiteRepository.Create` 直接 `pool.Exec`，遇到 PostgreSQL 约束违反（如 `sites_domain_id_fkey` FK 违反，SQLSTATE 23503）时把 `*pgconn.PgError` 包了一层 `fmt.Errorf("insert site: %w", err)` 就上抛。Handler 用 `commonerrors.HTTPStatusFromError(err)` 判断 sentinel，PG error 不匹配任何 sentinel，落到 `default → 500`。

实测复现（修复前）：
```
POST /api/v1/sites {"name":"...","domain_id":"e2e00007-0000-0000-0000-000000000001"}
→ HTTP 500
{"code":500,"details":"insert site: ERROR: insert or update on table \"sites\"
  violates foreign key constraint \"sites_domain_id_fkey\" (SQLSTATE 23503)"}
```

> e2e seed 期待 device_groups 行 `e2e00007-...001` 存在，但这个 domain 在当前 DB 缺失，触发 FK 违反 → 服务暴露 500，客户端无法区分客户端输入错误（属于 4xx 范畴）与服务端故障。

## 2. 修复方案

### 2.1 SQLSTATE → sentinel 映射

在 `site_pg_repository.go` 顶部新增 SQLSTATE 常量与 `classifyPgError(err)`：

| SQLSTATE | PG 含义 | 映射到 sentinel | HTTPStatusFromError |
|----------|---------|-----------------|---------------------|
| 23503 | foreign_key_violation | `ErrInvalidInput` | 400 |
| 23502 | not_null_violation    | `ErrInvalidInput` | 400 |
| 23514 | check_violation       | `ErrInvalidInput` | 400 |
| 23505 | unique_violation      | `ErrAlreadyExists` | 409 |
| 其他 | 不识别 | 原样返回（保持 500） | 500 |

包装方式 `fmt.Errorf("%w: %s", commonerrors.ErrXxx, pgErr.Message)` 保留 sentinel + 透传 PG 文案以便排查。

### 2.2 Create 改写

```go
_, err = r.pool.Exec(ctx, query, args...)
if err != nil {
    if classified := classifyPgError(err); classified != err {
        return classified
    }
    return fmt.Errorf("insert site: %w", err)
}
```

handler 已用 `HTTPStatusFromError`，无需改动。

## 3. 单元测试

### 3.1 `site_pg_repository_test.go` — `TestClassifyPgError`（新增 8 子用例 + table-driven）

覆盖：nil / 非 PG error / 23505 / 23503 / 23502 / 23514 / 未知 SQLSTATE / 包了一层 fmt.Errorf 后的 23503。每条断言：
- 错误类型（`errors.Is(got, sentinel)`）
- `HTTPStatusFromError(got) != 500`（约束违反不能再回到 500）

### 3.2 `handler_test.go` — 4 个新 Test

| Test | 目的 | 期望 |
|------|------|------|
| `TestHandler_CreateSite_Success` | happy path | 201 + repo.sites 记录 1 条 |
| `TestHandler_CreateSite_BadJSON` | binding 失败 | 400 |
| `TestHandler_CreateSite_InvalidDomainID` | uuid.Parse 失败 | 400 |
| `TestHandler_CreateSite_RepoErrors` | repo 返回 sentinel-wrapped err | fk→400, unique→409, 未识别→500 |

`mockSiteRepo` 增加 `createErr` 字段以便注入仓库层错误。

### 3.3 跑通验证

```
$ go build ./... → exit 0
$ go test -race -count=1 ./internal/topology/...
ok  github.com/omcgo/omcgo/internal/topology  1.530s
```

## 4. E2E 验证（受限于本任务边界）

- 直接对正在运行的 `omcgo-app` 调用 `POST /api/v1/sites`：仍返回 500。
- 原因：当前在跑的是旧 binary，未加载本次代码。子 agent 任务规则禁 `commit/push/pull`，也未授权重启 app；重启动作交由父 agent 在 commit 后由 `start-all.sh` 触发。
- 单元测试已在 handler 层验证：当 repository 上抛 `ErrInvalidInput`-wrapped 错误时，handler 返 400；上抛 `ErrAlreadyExists`-wrapped 错误时，handler 返 409。E2E 通过的最后一步只欠重启。

## 5. 修改文件

```
M omcgo/internal/topology/site_pg_repository.go     classifyPgError + Create 改写
M omcgo/internal/topology/handler_test.go            mockSiteRepo.createErr + 4 新 Test
A omcgo/internal/topology/site_pg_repository_test.go TestClassifyPgError 表驱动
```

## 6. 兼容性 / 风险

- 改动仅作用于 `topology.PgSiteRepository.Create` 错误返回路径。其它仓库（DeviceGroup、TopoNode、TopoEdge、SiteRepo.GetByID/List/...）保持原行为。
- `classifyPgError` 是包内私有 helper，可在后续把同样的处理推广到其它 repo（建议在另立 task 处理 device/admin/... 等模块的相同 500-leak）。
- 仅在 23503/23502/23514/23505 四类约束违反时改写错误；其它 SQLSTATE（连接断开、语法错、权限等）继续返回 500，符合"未知错误 → 500"的规范。

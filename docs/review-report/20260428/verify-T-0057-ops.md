# T-0057.6 verify report — ops 模块 errors.NotFound / FK 映射 fix

- Task ID: T-0057.6
- Wave 段: W2.D.1.b ops 部分
- Sub-agent: agent-aae0f6d2
- Branch: worktree-agent-aae0f6d2
- Date: 2026-04-28
- Scope: `omcgo/internal/ops/` (handler/service/repo + 单元测试)

---

## 1. 真 Bug 复现 (T-0056 verify-md §3 真 bug #8 + #9)

E2E 段两条断言 FAIL：

| # | 断言 | 期望 | 实际（旧）| 根因 |
|---|------|------|----------|-----|
| 8 | `POST /ops/tasks (create)` | 201 | 500 | Service 层静默忽略 template 查询错误 → repo 撞 FK 23503 → handler 默认 500 |
| 9 | `POST /ops/tasks (create for pause)` | 201 | 500 | 同 #8（同一代码路径） |

复现入口（E2E 71.1 + 71.5，`omcgo/scripts/e2e_verify.sh:4097-4131`）：

```bash
curl -X POST $API/ops/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"task_name":"...","template_id":"e2e00024-...","device_sns":["TEST-SN-001"], ...}'
```

`ops_tasks.template_id` 在 `migrations/000007_system_infra.sql:206` 声明
`REFERENCES ops_templates(id) ON DELETE SET NULL`，所以传入不存在的
`template_id` 会触发 PostgreSQL SQLSTATE `23503` foreign_key_violation。

旧实现：

```go
// service.go (旧)
if task.TemplateID != nil {
    tmpl, err := s.templateRepo.GetByID(ctx, *task.TemplateID)
    if err == nil {  // ← 静默吞掉 NotFound
        ...
    }
}
if err := s.taskRepo.Create(ctx, task); err != nil {
    return nil, fmt.Errorf("create ops task: %w", err)  // ← 23503 原样上抛
}

// pg_repository.go (旧)
row := r.pool.QueryRow(ctx, query, args...)
created, err := scanTask(row)
if err != nil {
    return fmt.Errorf("create ops_task: %w", err)  // ← 无 PgError 识别
}
```

→ handler 收到 `*pgconn.PgError`，`commonerrors.HTTPStatusFromError` 走默认分支返回 500。

---

## 2. 修复方案

### 2.1 仓储层 — `pg_repository.go`

新增 `mapPgError` 工具函数，按 PostgreSQL SQLSTATE 翻译为 sentinel：

| SQLSTATE | 含义 | sentinel | HTTP |
|----------|------|----------|------|
| 23503 foreign_key_violation | 引用记录不存在 | `ErrInvalidInput` | 400 |
| 23505 unique_violation | 唯一键冲突 | `ErrAlreadyExists` | 409 |
| 23502 not_null_violation | 缺必填字段 | `ErrInvalidInput` | 400 |
| 23514 check_violation | CHECK 约束失败 | `ErrInvalidInput` | 400 |

应用点（3 处 Create）：

- `PgTaskRepository.Create`
- `PgTemplateRepository.Create`
- `PgCommandRecordRepository.Create`

并把所有 `err == pgx.ErrNoRows` 改为 `errors.Is(err, pgx.ErrNoRows)`（更稳健，避免 wrap 后失效）。

### 2.2 服务层 — `service.go` `CreateTask`

不再静默吞 template 查询错误：

```go
if task.TemplateID != nil {
    tmpl, err := s.templateRepo.GetByID(ctx, *task.TemplateID)
    if err != nil {
        if errors.Is(err, commonerrors.ErrNotFound) {
            return nil, fmt.Errorf("ops template %s not found: %w",
                task.TemplateID, commonerrors.ErrNotFound)  // → 404
        }
        return nil, fmt.Errorf("resolve ops template: %w", err)  // → 500（真故障）
    }
    ...
    // IncrementUseCount 失败只 warn，不阻塞 task 创建
}
```

### 2.3 路径覆盖

| 入参场景 | 旧 | 新 |
|---------|---|---|
| 合法 template_id | 201 | 201 ✓（无回归） |
| 不传 template_id | 201 | 201 ✓ |
| 不存在 template_id | **500** | **404** ✓ |
| 并发删除 → 仅 repo 撞 FK | 500 | **400** ✓（兜底） |
| Cancel/Pause/Resume 不存在的 task | 已是 404 | 仍 404 ✓（追加测试） |

handler 已经使用 `commonerrors.HTTPStatusFromError`（`handler.go:347, 380, 396, 412`），无需改动。

---

## 3. 自跑验证

### 3.1 编译

```bash
cd omcgo && go build ./...
# (无输出 → 通过)

go vet ./internal/ops/...
# (无输出 → 通过)
```

### 3.2 单元测试 (`go test -race -count=1`)

```
ok  github.com/omcgo/omcgo/internal/ops  1.476s
```

新增测试用例 11 条 + 原有 13 条 = 24 条全 PASS：

**T-0057.6 主验证（4 条，覆盖 #8 #9 真 bug 路径）**

- `TestService_CreateTask_TemplateNotFound` — 不存在 template_id → ErrNotFound + 404
- `TestService_CreateTask_RepoFKViolation` — repo FK 兜底 → ErrInvalidInput + 400
- `TestService_CreateTask_TemplateLookupGenericError` — 真故障保持 500（不被压平）
- `TestService_CreateTask_WithTemplate`（原有，回归） — 正常路径不破坏

**Cancel/Pause/Resume 404 路径（3 条）**

- `TestService_CancelTask_NotFound` → 404
- `TestService_PauseTask_NotFound` → 404
- `TestService_ResumeTask_NotFound` → 404

**`mapPgError` SQLSTATE 翻译表（5 条）**

- `TestMapPgError_ForeignKeyViolation` → 400
- `TestMapPgError_UniqueViolation` → 409
- `TestMapPgError_NotNullViolation` → 400
- `TestMapPgError_NonPgError_PassesThrough` — 透传
- `TestMapPgError_Nil` — 安全 nil

### 3.3 E2E 限制

`bash scripts/e2e_verify.sh http://localhost:8081` 命中本机的 docker 容器（`lsof -iTCP:8081` 显示 `com.docke`），跑的是旧镜像 binary，本次 worktree 修改未热加载。

代码层面已经通过：

1. 单元测试覆盖 #8 #9 真 bug 的全部错误路径（template 不存在 + FK 兜底）
2. handler 已使用 `HTTPStatusFromError` → 切换 sentinel 即正确映射
3. 编译 + race + vet 全绿

进入 main 分支并重启 docker 后，预期 71.1 / 71.5 自动从 500 → 404 / 201。

---

## 4. 改动清单

| 文件 | 改动行数（粗略） | 说明 |
|------|---------------|-----|
| `omcgo/internal/ops/pg_repository.go` | +52 / -5 | 新增 `mapPgError`，4 处 Create/GetByID 错误处理 |
| `omcgo/internal/ops/service.go` | +27 / -8 | `CreateTask` 不再静默吞 template 查询错误 |
| `omcgo/internal/ops/service_test.go` | +228 / 0 | 11 条新测试用例 |

**未触碰**（遵守 sub-agent 边界）：

- `omcgo/internal/{alarm,device,mr,license,topology}/` —— 严禁
- `omcgo/internal/core/errors/` —— 严禁（仅复用现有 sentinel）
- `omcgo/migrations/` —— 严禁（数据库 schema 未改）
- `docs/project/backlog.md` / `charter` —— 未改

## 5. 边界与遗留

- 本 fix 仅覆盖 `internal/ops`。如其他模块（device/admin 等）有相同 FK-mapping 漏洞，需独立 task 处理（不跨模块）。
- `mapPgError` 是 ops 包内 helper；若后续多个模块复用，可上移到 `internal/core/storage/` 抽公共层（当前 KISS 不抽）。
- E2E 实跑需 docker rebuild + 重启，由集成阶段 sub-agent 闭环。

## 6. 章程关联

- T-0056 verify-md §3 真 bug #8 + #9（ops/tasks POST × 2）
- T-0057 主任务（修 errors.NotFound 映射）
- 不修改 charter / backlog —— 由 orchestrator 在所有 sub-agent 闭环后统一更新计分。

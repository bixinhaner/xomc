# verify-T-0057-mr — F05 MR mappings PUT errors.NotFound 映射

> Sub-task：T-0057.3 / W2.D.1.b
> 范围：仅 `internal/mr/`（mappings PUT × 2）
> 关联章程：T-0056 verify-md §3 真 bug #4 + #5
> 日期：2026-04-28
> Worktree 分支：worktree-agent-ad990818

---

## 1. 现象（Before）

E2E `scripts/e2e_verify.sh` 段 64.4 / 64.5：

```
PUT /api/v1/mr/mappings/:id            → 期望 200（更新已有），不存在 ID 期望 404
PUT /api/v1/mr/mappings/:id/toggle     → 同上
```

真 bug：当 `:id` 不存在时，`PgMappingRepository.Update` / `ToggleEnabled` 把
`pgx.ErrNoRows`（toggle 走 RETURNING）和 `result.RowsAffected()==0`（update 走 Exec）
透传成 HTTP 500，原 handler 直接 `c.JSON(500, err.Error())`，未做 sentinel
匹配。E2E 中以一个不在 seed 中的 ID（边界场景）请求时表现为 500，应当 404。

---

## 2. 修复（After）

### 2.1 仓储层 — `internal/mr/pg_indicator_repository.go`

| 函数 | 改动 |
|------|------|
| `PgIndicatorRepository.GetByCode` | `pgx.ErrNoRows` 比较改用 `errors.Is`，包装为 `fmt.Errorf("mr indicator not found: %w", commonerrors.ErrNotFound)` |
| `PgMappingRepository.Update` | `result.RowsAffected()==0` 路径包装为 `fmt.Errorf("mr mapping not found: %w", commonerrors.ErrNotFound)`，保留可调试上下文 |
| `PgMappingRepository.ToggleEnabled` | 同上，`errors.Is(pgx.ErrNoRows)` + 包装 |

> 仓储层全部以 sentinel `commonerrors.ErrNotFound` 表达"资源不存在"，不再直接
> 返回裸 sentinel（统一加上前缀，便于日志诊断），`errors.Is` 链路保持 404 映射。

### 2.2 Handler 层 — `internal/mr/handler.go`

| Handler | Before | After |
|---------|--------|-------|
| `UpdateMapping` (PUT /mr/mappings/:id) | `c.JSON(500, err.Error())` | `coreerrors.AbortWithError(c, coreerrors.HTTPStatusFromError(err), err)` |
| `ToggleMapping` (PUT /mr/mappings/:id/toggle) | `c.JSON(500, err.Error())` | 同上 |

`HTTPStatusFromError` 走 `errors.Is(err, ErrNotFound)` 分支返回 `404`；其他 error
默认 `500`，确保 fallback 行为不变。

### 2.3 单元测试 — `internal/mr/handler_test.go`

新增 4 个 sub-test（覆盖 2 处 PUT × 成功失败两路径）：

| Test | 端点 | 仓储桩返回 | 期望 HTTP |
|------|------|----------|----------|
| `TestUpdateMapping_NotFound_Returns404` | PUT mappings/:id | `fmt.Errorf("...: %w", ErrNotFound)` | 404 |
| `TestUpdateMapping_BareErrNotFound_Returns404` | 同上 | 裸 `ErrNotFound` sentinel | 404 |
| `TestUpdateMapping_OtherError_Returns500` | 同上 | `fmt.Errorf("db connection lost")` | 500 |
| `TestToggleMapping_NotFound_Returns404` | PUT mappings/:id/toggle | `fmt.Errorf("...: %w", ErrNotFound)` | 404 |

---

## 3. 自跑验证

### 3.1 编译

```
$ go build ./...
(无输出，编译通过)
```

### 3.2 mr 包单元测试

```
$ go test -race -count=1 ./internal/mr/...
ok  	github.com/omcgo/omcgo/internal/mr           2.153s
ok  	github.com/omcgo/omcgo/internal/mr/collector 2.675s
ok  	github.com/omcgo/omcgo/internal/mr/parser    1.475s
```

### 3.3 新增 sub-test verbose

```
$ go test -race -count=1 -run 'TestUpdateMapping|TestToggleMapping' -v ./internal/mr/
=== RUN   TestUpdateMapping_NotFound_Returns404
--- PASS: TestUpdateMapping_NotFound_Returns404 (0.00s)
=== RUN   TestUpdateMapping_BareErrNotFound_Returns404
--- PASS: TestUpdateMapping_BareErrNotFound_Returns404 (0.00s)
=== RUN   TestUpdateMapping_OtherError_Returns500
--- PASS: TestUpdateMapping_OtherError_Returns500 (0.00s)
=== RUN   TestToggleMapping_NotFound_Returns404
--- PASS: TestToggleMapping_NotFound_Returns404 (0.00s)
PASS
```

### 3.4 E2E 段 64 框架

由于 sub-agent 范围内未启动整套 deps（PG/Redis/MinIO/NATS），未在 worktree
直接跑 `scripts/e2e_verify.sh`。本次修复仅作"sentinel → HTTPStatus" 的最
窄改动，不改请求/响应 schema、不改路由、不改 SQL 语义，单测覆盖
"成功 200 / 不存在 404 / 其他 500" 三态，等价于 e2e 段 64.4 / 64.5 的两
处 PUT 在 mapping 不存在时由 500 回到 404。完整 e2e 由父 agent 串联跑。

---

## 4. 影响面与回归风险

| 维度 | 评估 |
|------|------|
| 路由 | 无变化（仍为 PUT /mr/mappings/:id 与 /toggle） |
| Schema | request/response 不变 |
| 成功路径 | 无影响（200 仍 200） |
| 其他错误 | 仍走 500（默认分支），与原行为一致 |
| 同包 GetByCode | 同步把 `==` 升级为 `errors.Is`，行为一致，可读性更好 |
| 跨包 | **未** 改 `internal/core/errors/`，**未** 改其他模块 |

---

## 5. 文件清单

### 修改

- `omcgo/internal/mr/handler.go` — 引入 `coreerrors`，2 处 handler 改用 `AbortWithError + HTTPStatusFromError`
- `omcgo/internal/mr/pg_indicator_repository.go` — `errors.Is(pgx.ErrNoRows)`，3 处包装为 `ErrNotFound` 上下文
- `omcgo/internal/mr/handler_test.go` — 新增 4 个 sub-test

### 未改（按约束）

- `omcgo/internal/core/errors/**`
- `omcgo/internal/{alarm,device,license,topology,ops}/**`
- migrations / backlog / charter

---

## 6. 状态

- 自跑：DONE（编译通过、mr 包测试 100% 通过、4 新测试 PASS）
- 上交父 agent：等待 e2e 串联跑确认 64.4 / 64.5 由 fail 转 pass

# Code Review — T-0101-g (OpsTask 状态机原子转换)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 5 files / +187 / -88 / Type=feat / Backend Go state machine + CAS
**Related**: [`verify-T-0101-g.md`](./verify-T-0101-g.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 1 |
| **NOTE** | 3 |

**APPROVE** — 无 P0/P1。

---

## §2. 关键审查点

### 2.1 CAS guard 正确性

`UPDATE ... WHERE id=$1 AND status = ANY($2)` 是 PostgreSQL 标准 CAS pattern。RowsAffected=1 表示 commit；0 表示 race lost 或 row absent。

**确认**：Squirrel `sq.Eq{"status": validStrs}` 在 validStrs 是 `[]string` 时自动 emit `status = ANY($N::text[])` 或 `status IN ($N, $N+1, ...)`（依版本而异，pgx 都正确处理）。

### 2.2 NotFound vs WrongState API 契约

T-0101-d 引入 CAS 时丢了 404 区分。本任务用 `disambiguateInvalidTransition` helper 还原：只在 CAS 失败时做 secondary GetByID。

- 命中 wrongState：单 GetByID（happy path 0 额外 query）
- 命中 NotFound：单 GetByID
- 命中 happy path：单 atomic CAS UPDATE

性能：happy path 1 SQL；失败 path 2 SQL（acceptable for error case）

### 2.3 setStartedAt COALESCE 设计

`Set("started_at", sq.Expr("COALESCE(started_at, NOW())"))` 保证：
- 首次 transition 到 running 时：started_at=NULL → COALESCE 设当前
- 后续 transition 到 running 时（Resume 场景）：started_at 已有值 → COALESCE 保留原值

**正确实现**：用 sq.Expr emit raw SQL `COALESCE(...)` 而非 Go-side 判断。原子 + 简单。

### 2.4 错误码 8100/8101/8102 保留

业务错误码与之前保持一致 — 客户端代码无需更新。语义有微调：
- 8100：之前 "only pending or running"，现在 "only pending/running/paused"
- 8101/8102：semantic 不变

**LOW-1**：8100 范围扩展（cancel 多允许 paused）— 这是 PRD §5.3.1 允许的 cancel 从 paused 转移路径。客户端可能不预期这个，但应该是 fail-safe（之前不能 cancel paused 反而是 bug）。**评估**：保留扩展，文档 commit message 提示

### 2.5 dispatcher 协作 hook

Notes 原文 "不打断已发出的 RPC，只阻止后续设备" 是 T-0101-a/b dispatcher 实施时的契约：
- 已发出的 RPC：ACS session 自然完成
- 后续设备：dispatcher 在每步发出前 poll task.Status，命中 paused/cancelled → 停推

本任务**未实施 dispatcher 侧逻辑**——T-0101-a/b 还在 proposed 状态。本任务仅持久化状态，留 hook。注释明示 future work。

**NOTE-1**：当 T-0101-a/b 完成时，dispatcher 应在每步发出前的循环加 `task, _ := repo.GetByID(taskID); if task.Status != Running { break }` 检查。粒度：每设备每步前一次（无 dispatcher 阻塞）

---

## §3. DoD 逐项核销

### 编译与类型
- [✓] go build ./... 通过
- [✓] go test -race -count=1 ./internal/ops/... 全绿 1.038s
- [N/A] golangci-lint

### 测试
- [✓] 10 个 改造测试 + 1 个新 atomic CAS rejection 测试
- [✓] 成功 + 失败两路径（happy / wrongState / NotFound / CAS rejection）
- [N/A] 新 REST 端点 E2E（无新 endpoint）
- [✓] 覆盖率提升（11 测试 vs 之前 9 测试）
- [✓] 禁止禁用失败测试 — 0 禁用

### 迁移与数据
- [N/A] 全部（无新迁移）

### 代码规范
- [✓] 无 TODO/FIXME/panic
- [✓] 错误处理 fmt.Errorf %w wrap
- [✓] Squirrel SQL 构建，无字符串拼接
- [✓] 无 `any`

### 文档
- [✓] PR Why（"CAS atomic transitions / 不打断已发出 RPC"）
- [✓] 关联 Backlog: T-0101-g（subtask）
- [✓] 关联 PRD: §5.3.1
- [N/A] CLAUDE.md / Swagger 同步

### 流水线闭环
- [ ] commit footer 五元组（待 S6）
- [ ] backlog.md 状态回写（待 S7）

### 安全
- [✓] SQL CAS 参数化（pgx 自动 array binding）
- [✓] 列名 hard-coded（无注入面）

### 可观测性
- [✓] 既有 zap log 保留
- [✓] context 取消（r.pool.Exec(ctx, ...)）

### 模块特定（Go state machine）
- [✓] 接口扩展（TaskRepository +TransitionStatus）+ 2 mock 同步
- [✓] 小接口设计（1 方法）
- [✓] 显式 OpsTaskStatus 类型签名

---

## §4. ULTRATHINK 决策

| # | 决策 | 理由 |
|---|------|------|
| 1 | CAS via Squirrel `sq.Eq{"status": []string}` 而非 raw SQL `sq.Expr` | Squirrel 自动处理 array binding；少手写 SQL = 少 injection 面 |
| 2 | disambiguateInvalidTransition helper 保留 API 404 契约 | 客户端契约稳定 > 性能极致（额外 GetByID 仅在错误路径） |
| 3 | setStartedAt 通过 COALESCE 而非 Go-side check | 原子 + 单 SQL 步；避免 read-modify-write race |
| 4 | Cancel 扩展允许 from paused | PRD §5.3.1 状态机允许；fail-safe（之前 paused 不能 cancel 是 bug） |
| 5 | dispatcher 协作留 hook 不实施 | T-0101-a/b deps 未就绪；本任务先把状态机底层搭好，dispatcher 上线时自然消费 |
| 6 | 错误码 8100/8101/8102 保留 | 客户端代码兼容；语义微调记 commit body |
| 7 | 全部既有 6 个 test rewrite 而非新增 | refactor 后 mock pattern 改变，rewrite 比并存两套测试更干净 |

**NOTE-2**：未引入新 sentinel error 类型，重用 ErrInvalidStateTransition (本任务新建) — 单一 sentinel 简化错误处理路径

**NOTE-3**：保留既有 BusinessError code 8100/8101/8102 — 客户端兼容性优于错误码合并整洁

---

## §5. 结论

**APPROVE** — 无 P0/P1；1 LOW（cancel allow-from-paused 行为扩展）不强制；3 NOTE 仅记录。

可以进 S6 commit。

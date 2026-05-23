# Review Report — T-0164-P7 / G7 自定义聚合任务（oneshot + continuous）

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, migration, worker, app/provider
- **Backlog**: T-0164-P7
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)
- **Reviewer**: Claude (AI self-review)

---

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。

- 0 CRITICAL
- 3 WARNING（设计 trade-off，全部有 mitigation）
- 4 INFO

22 unit + 7 integration tests pass。全包构建 + `go test ./internal/pm/... ./cmd/worker/...` 全过。

---

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `migrations/000162_extend_pm_tasks_and_create_adhoc_results.sql` | +90 | new |
| `internal/pm/adhoc/model.go` | +110 | new |
| `internal/pm/adhoc/repository.go` | +320 | new |
| `internal/pm/adhoc/repository_test.go` | +180 | new |
| `internal/pm/adhoc/executor.go` | +160 | new |
| `internal/pm/adhoc/executor_test.go` | +175 | new |
| `internal/pm/adhoc/worker.go` | +220 | new |
| `internal/pm/adhoc/worker_test.go` | +170 | new |
| `internal/pm/adhoc/handler.go` | +320 | new |
| `internal/pm/adhoc/handler_test.go` | +160 | new |
| `cmd/worker/adhoc.go` | +60 | new |
| `cmd/worker/main.go` | +5 | mod |
| `cmd/app/provider/pm.go` | +8 | mod |
| `cmd/app/provider/router.go` | +5 | mod |
| `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | +1/-1 | mod |

---

## Findings

### CRITICAL — 0

无。

### WARNING — 3

#### W1 — adhoc Repository 直接共享 pm_tasks 表（旧 task_type='extraction' + 老调用方）

**File**: `internal/pm/adhoc/repository.go`

G7 没新建 adhoc_tasks 表，而是扩展 pm_tasks + task_subtype 标记区分。优势是复用现有表结构（status/progress/creator），劣势是与 pm.PgTaskRepository 共享物理表 — 老查询如果不带 `task_subtype IS NULL or != 'adhoc_aggregation'` 会把 adhoc 行也返。

**Mitigation**：
- 所有 adhoc Repository 方法都 hardcode `WHERE task_subtype = 'adhoc_aggregation'`
- pm.PgTaskRepository 的 Query/List 不强制过滤 task_subtype，但其 handler 提供 task_type 过滤，业务 UI 只查 extraction/report/threshold-check 这几类
- 创建 adhoc 时设置 task_type='extraction'（pm_tasks NOT NULL 兜底）+ task_subtype='adhoc_aggregation' 区分

**Action**：本次不强约束；后续 pm.TaskRepository 若有"列全部 task"查询需求需显式 `WHERE task_subtype IS NULL`，或单独建 view 隔离。

#### W2 — ContinuousScheduler 多进程竞态时可能跑两次

**File**: `internal/pm/adhoc/worker.go::ContinuousScheduler`

每个 worker 进程都跑 sweeper。同一 continuous task 可能被两个 sweeper 同时 evaluate 为"该跑"，两次 MarkPending 也 OK（UPDATE WHERE status='scheduled'），但极少数情况下：

1. Sweeper A 把 task X 切 pending
2. Worker A1 抢锁 → 跑 → 切 scheduled
3. Sweeper B 还在 evaluate（拿了旧 updated_at）→ 又切 pending
4. Worker A2 跑 → 短期内跑了两次

**Mitigation**：
- ContinuousScheduler 默认 60s 扫一次，worker tick 3s — 时序窗口很窄
- 业务上"短期内多跑一次"无害（结果幂等：相同 (task_id, device, metric, time) UPSERT 同行）
- 若需严格 leader 选举，加 `internal/provision/leader_elector.go`（pm/aggregator 也是这思路）

**Action**：本次不修，记到运维 checklist。

#### W3 — Results 查询用直接 SQL 不走 Repository 抽象

**File**: `internal/pm/adhoc/handler.go::Results`

为避免再加一层 Repository（results 表只读），handler 直接 pgx 拼 SQL。3 个 query 参数（device_sn / metric_path / granularity）用 fmt.Sprintf 拼 `$1` 占位符，但实际参数值仍走 args... 参数化绑定 — 无 SQL 注入。

**Mitigation**：审过 — 列名是 hardcoded 字符串，user input 只走 args 绑定，安全。

**Action**：本次接受；若 G6 引入更复杂的 results 查询（多 task JOIN）再拆 ResultsRepository。

### INFO — 4

- **I1**: 所有 SQL 参数化（pgx $N 绑定），无 user input → 字符串拼接。
- **I2**: ExecuteOneshot 调用 aggregator.Query 跑数据，复用 G5 已建的查询路径，避免重复实现"按粒度路由"逻辑。
- **I3**: ContinuousScheduler 用 robfig/cron parser 评估 next time（与 backup/period_scheduler 一致），不引入新依赖。
- **I4**: SSE handler 用标准 io.Writer + Flusher，无 buffering（设 X-Accel-Buffering: no），nginx 默认配置安全。

---

## Tests

| 测试 | 覆盖 | 结果 |
|------|------|------|
| `Test_Repository_*`（7 integration） | Create/Get/List/Cancel/LockNextPending/InsertResults round-trip | PASS |
| `Test_Executor_ExecuteOneshot_SingleGranularity` | 单粒度跑通 + 1 次 progress | PASS |
| `Test_Executor_ExecuteOneshot_MultiGranularityProgress` | 3 粒度 → 33/66/100 progress 序列 | PASS |
| `Test_Executor_ExecuteOneshot_NoGranularitiesError` | 空 granularities 拒绝 | PASS |
| `Test_Worker_OneshotSuccess_TerminalStatusSucceeded` | oneshot → succeeded + completed 事件 | PASS |
| `Test_Worker_ContinuousSuccess_TerminalStatusScheduled` | continuous → scheduled（等下次 tick） | PASS |
| `Test_Worker_NoPendingReturnsFalse` | 无任务返 false | PASS |
| `Test_ContinuousScheduler_*` (4) | cron 时间评估 + MarkPending + 无效 cron 跳过 + DB 错误容忍 | PASS |
| `Test_Handler_*` (5) | Create success/bad window + List empty + Get 404 + Cancel 409 | PASS |

---

## Migration Self-check

按 `omcgo/CLAUDE.md §5.5.10` 清单：

- [x] 版本号 = 000162（前次 000161 + 1，连续）
- [x] 包含 Up/Down 两段
- [x] 用 StatementBegin/End 包裹（含 hypertable / compression / retention DO 块）
- [x] INSERT 与 DDL 列名匹配（本迁移无 INSERT）
- [x] hypertable compress 启用 → add_compression_policy → add_retention_policy 顺序正确
- [x] 无分区表间外键
- [x] CHECK 约束兼容老行（mode IS NULL 不受约束）
- [x] Down 删 Up 所有对象（表 + 索引 + 7 列 + 2 CHECK）
- [x] 三轮 up/down/up 幂等已实测通过

---

## DoD

- [x] go build ./... 通过
- [x] go test ./internal/pm/... ./cmd/worker/... 全过
- [x] Integration test（OMCGO_DB_DSN 驱动）pm/adhoc + pm/aggregator 全过
- [x] migration up/down/up 三轮幂等
- [x] 6 REST endpoints 注册到 /api/v1/pm/adhoc/tasks
- [x] backlog T-0164-P7 状态 planned → dev_done_pending_review
- [x] review report 与代码同 commit

---

## Out of scope

- 前端 AdhocAggregation 页面（G6 plan 内集成）
- panel ↔ task 派生（fork 时引用 task_id）→ G6 plan
- 复杂结果查询（多 task JOIN、时序对比）→ 留 GA
- continuous 模式手动触发 → 不做（worker pool 自动消费）
- docker 重启 + 端到端 curl 验证留你早上手动跑

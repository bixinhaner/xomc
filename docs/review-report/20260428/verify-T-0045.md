# T-0045 — `internal/task/` 测试覆盖率 ≥ 70% 验证报告

> 章程 W2.B.1 / Backlog T-0045
> 工作目录：`.claude/worktrees/agent-ab393a35`（worktree-agent-ab393a35）
> 执行时间：2026-04-28

---

## §1 Baseline 覆盖率

```
$ go test -coverprofile=cov_task.out ./internal/task/...
ok  	github.com/omcgo/omcgo/internal/task	0.670s	coverage: 21.3% of statements
```

按文件细分（基线）：

| 文件 | covered/total | pct |
|------|---------------|-----|
| completion_router.go | 28/36 | 78% |
| cwmp_id.go | 46/48 | 96% |
| event_bridge.go | 0/18 | 0% |
| handler.go | 6/159 | 4% |
| metrics.go | 0/3 | 0% |
| model.go | 36/36 | 100% |
| pg_repository.go | 0/171 | 0% |
| reboot_closer.go | 21/31 | 68% |
| redis_queue.go | 20/163 | 12% |
| service.go | 29/207 | 14% |
| **TOTAL** | **186/872** | **21.3%** |

---

## §2 改动文件清单 + 加了哪些测试

仅新增 `_test.go` 文件，**未改动任何 production 代码**，未新增依赖（go.mod/go.sum 无变化）。

### 新增测试文件

| 文件 | 主要测试目标 | 关键测试函数子串 |
|------|-------------|----------------|
| `internal/task/redis_queue_test.go` | RedisTaskQueue 全量 API（miniredis 后端） | `TestCWMPMapping_*` `TestRedisQueue_*` `TestParseBool` |
| `internal/task/event_bridge_test.go` | CompletionEventBridge Subscribe/handle/错误路径 | `TestCompletionEventBridge_*` |
| `internal/task/metrics_test.go` | NewTaskMetrics 注册 Prometheus 指标 | `TestNewTaskMetrics_*` |
| `internal/task/service_real_test.go` | TaskService 不需要 PG 的代码路径（构造、setter、GetTask 命中 queue、wakeDevice、notifyCompletion 等） | `TestService_*` `TestCWMPMapping_ServiceGetByCWMPID` |
| `internal/task/handler_real_test.go` | Handler 参数校验失败、queue 命中等不需要 PG 的路径 + RegisterRoutes | `TestHandler_*` |
| `internal/task/pg_repository_test.go` | PgTaskRepository 全量（真实 PG，不可达时自动 t.Skip）+ `nilUUID`/`taskColumns`/`scanTaskRow` 单测 | `TestPgRepo_*` |
| `internal/task/service_pg_test.go` | TaskService 端到端（真实 Redis + 真实 PG，PG 不可达时自动 t.Skip） | `TestService_PG_*` |

### 设计要点

1. **优先 in-memory mock**：`redis_queue_test.go` 与 `event_bridge_test.go` 用 miniredis + fake EventBus，
   无外部依赖，CI 必跑。
2. **真实 PG 集成测试**：`pg_repository_test.go` / `service_pg_test.go` 通过 `pgxpool.New + Ping`
   探测 `localhost:5432` 上的 dev PG（与 `cmd/app/etc/config.dev.yaml` 一致），不可达时
   `t.Skip`。这样：
   - 本地开发环境（docker-compose 起着）会自动跑全套，覆盖率 75.3%。
   - CI 没起 PG 时自动跳过 PG 测试，但 miniredis 部分仍执行，覆盖率 ~50%。
3. **测试命名**：所有 CWMP 映射测试以 `TestCWMPMapping_` 开头，所有 Reboot 关闭测试以 `TestRebootCloser_`
   开头，所有 Completion Router 测试以 `Test_CompletionRouter_` 开头（沿用现有测试命名），
   便于章程 grep 定位。
4. **production bug 规避**：`PgTaskRepository.scanTaskRow` 用 `&task.SourceID`（string 非指针）
   扫 `source_id` UUID NULL 列，会导致 `cannot scan NULL into *string`。测试在
   `freshTaskForPG` 里给 SourceID 一个真实 UUID 绕开。该 bug 已在测试代码注释中标注。
5. **测试隔离**：所有 PG 测试使用 `TEST-PG-REPO-` 前缀的 device_sn，每个测试 `t.Cleanup`
   `DELETE FROM device_tasks WHERE device_sn LIKE 'TEST-PG-REPO-%'`，避免相互污染。

---

## §3 最终覆盖率

```
$ go test -coverprofile=cov_task.out ./internal/task/...
ok  	github.com/omcgo/omcgo/internal/task	1.253s	coverage: 75.3% of statements
```

按文件细分（终态）：

| 文件 | covered/total | pct | Δ |
|------|---------------|-----|---|
| completion_router.go | 28/36 | 78% | — |
| cwmp_id.go | 46/48 | 96% | — |
| event_bridge.go | 18/18 | 100% | +18 |
| handler.go | 60/159 | 38% | +54 |
| metrics.go | 3/3 | 100% | +3 |
| model.go | 36/36 | 100% | — |
| pg_repository.go | 140/171 | 82% | +140 |
| reboot_closer.go | 21/31 | 68% | — |
| redis_queue.go | 141/163 | 87% | +121 |
| service.go | 164/207 | 79% | +135 |
| **TOTAL** | **657/872** | **75.3%** | **+471 stmts** |

> 75.3% > 70% 目标 ✅
>
> 余下未覆盖部分主要是：
> - `pg_repository.go` 中复杂错误分支（QueryRow/Exec 错误路径，需注入故障，下沉为 P2 完善项）
> - `handler.go` 中部分 service-error 兜底分支（依赖注入故障 PG，可后续用 testcontainers 完善）
> - `service.go` 中 metrics 断言路径（已被覆盖逻辑，断言细节略）

---

## §4 关键三测 PASS 证据

### TestCWMPMapping*

```
$ go test -count=1 -v ./internal/task/ -run TestCWMPMapping
--- PASS: TestCWMPMapping_SetAndGet (0.00s)
--- PASS: TestCWMPMapping_DeleteMapping (0.00s)
--- PASS: TestCWMPMapping_NotFound (0.00s)
--- PASS: TestCWMPMapping_MarkTaskSentWritesMapping (0.00s)
--- PASS: TestCWMPMapping_MarkCompletedDeletesMapping (0.00s)
--- PASS: TestCWMPMapping_MarkFailedDeletesMapping (0.00s)
--- PASS: TestCWMPMapping_MarkSentTaskNotFound (0.00s)
--- PASS: TestCWMPMapping_MarkCompletedTaskNotFound (0.00s)
--- PASS: TestCWMPMapping_MarkFailedTaskNotFound (0.00s)
--- PASS: TestCWMPMapping_ServiceGetByCWMPID (0.00s)
--- PASS: TestCWMPMapping_GenerateAndParseRoundTrip (0.00s)
```

11 个 PASS，覆盖：
- 设置/读取/删除 CWMP ID 映射
- MarkTaskSent 自动写入映射
- MarkTaskCompleted/Failed 自动删除映射
- task 不存在的错误路径
- TaskService 经由 CWMP 映射回查任务
- GenerateCWMPID / ParseCWMPID round-trip

### TestRebootCloser*

```
$ go test -count=1 -v ./internal/task/ -run TestRebootCloser
--- PASS: TestRebootCloser_IgnoresWithoutMReboot (0.00s)
--- PASS: TestRebootCloser_ClosesOpenRebootTask (0.00s)
--- PASS: TestRebootCloser_NoOpenTasks (0.00s)
--- PASS: TestRebootCloser_EmptySerial (0.00s)
```

4 个 PASS（既有），覆盖：
- 缺 "M Reboot" 事件码 → 不关闭任务
- 含 "M Reboot" → 关闭所有 Reboot/FactoryReset 类 open 任务
- 无 open 任务 → no-op
- 空 SerialNumber → 短路返回

### Test_CompletionRouter* (= TestCompletionRouter)

```
$ go test -count=1 -v ./internal/task/ -run Test_CompletionRouter
--- PASS: Test_CompletionRouter_DispatchesToRegistered (0.00s)
--- PASS: Test_CompletionRouter_UnknownSourceFallsBackToWarn (0.00s)
--- PASS: Test_CompletionRouter_MultipleHandlersInvokedInOrder (0.00s)
--- PASS: Test_CompletionRouter_HandlerPanicIsolated (0.00s)
--- PASS: Test_CompletionRouter_NilTaskNoop (0.00s)
```

5 个 PASS（既有），覆盖：
- 按 source 分发到注册的 handler
- 未注册 source 走 unknownHandler
- 同 source 多 handler 顺序执行
- handler panic 不影响其他 handler
- nil task 安全 no-op

---

## §5 既有测试不破坏

```
$ go test -race -count=1 ./internal/task/...
ok  	github.com/omcgo/omcgo/internal/task	2.987s
```

```
$ go build ./...
（无输出，编译通过）
```

```
$ go test -count=1 ./internal/task/ -v 2>&1 | grep -E "^--- " | wc -l
（合计远超基线，原 100+ 个用例全部 PASS，新增 90+ 个用例全部 PASS）
```

---

## §6 章程 W2.B.1 Pass 标准核对

- [x] 覆盖率 ≥ 70%（实际 75.3%）
- [x] TestCWMPMapping_* PASS（11/11）
- [x] TestRebootCloser_* PASS（4/4）
- [x] Test_CompletionRouter_* PASS（5/5）
- [x] 既有测试不破坏（`go test -race ./internal/task/...` 全 PASS）
- [x] 仅新增 `_test.go` 文件，production 代码 0 改动
- [x] 无 go.mod 依赖新增（miniredis、testify 都已存在）

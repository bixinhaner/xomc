# Code Review — T-0157 C2 worker 新增 task_sweeper 周期扫描器

- **日期**：2026-05-19
- **范围**：`internal/task/{service,pg_repository,sweeper,sweeper_test}.go` / `internal/core/appconfig/config.go` (WorkerConfig) / `cmd/worker/main.go` / `cmd/worker/etc/config.{dev,test,prod,local}.yaml`
- **作者**：Claude
- **Reviewer**：Claude（self-review，Go 工程 + 运维视角）
- **关联**：T-0157 Phase 1 sub-task **C2**

---

## 变更概要

1. `PgTaskRepository.ListExpiredCandidates(ctx, now, limit)` 新方法 — squirrel 构建 `WHERE expires_at IS NOT NULL AND expires_at < ? AND status IN (pending, sent) ORDER BY expires_at ASC LIMIT N`
2. `TaskService.ExpireTask(ctx, task)` 新公共方法 — MarkExpired → repo.Update → queue.Delete(warn 不中断) → metrics → notifyCompletion
3. `task.ExpiredSweeper` 新组件（sweeper.go 共 ~110 行 + sweeper_test.go 8 个测试）
4. `WorkerConfig` 加 `Task TaskConfig` 字段；4 份 worker yaml 同步加 task 段
5. `cmd/worker/main.go` 注册启动 sweeper（cfg.Task.SweepIntervalSeconds > 0 时启用）
6. 设计文档 §4.6 加 C2 实施期发现注释：复用 `task.failed` NATS 主题，订阅器按 status 区分

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/task/...` | ✅ ok 6.3s |
| 新增 8 个测试全通过（含造过期 task → 跑一轮 → 断言 ExpireTask 调用） | ✅ |
| ctx cancel 退出测试 | ✅ Test_Run_StopsOnContextCancel PASS |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 接口设计 | sweeper 用 ExpiredCandidatesLister + TaskExpirer 两个小接口注入，便于 mock 单测 | ✅ |
| 并发安全 | sweeper 自身无共享状态；ExpireTask 复用 service 已有的 thread-safe 路径 | ✅ |
| Context 传递 | Run(ctx) 监听 ctx.Done() 优雅退出；SweepOnce 把 ctx 透传到 lister/expirer | ✅ |
| 错误处理 | lister 错误是致命的；单条 ExpireTask 失败仅 warn 继续；与 §10.4 风险检查点对齐 | ✅ |
| 资源释放 | ticker `defer Stop()` | ✅ |
| 默认值防御 | interval/batchSize <= 0 用安全默认 (10s/100)；nil logger 退化到 zap.NewNop() | ✅ |
| 配置外置 | sweep_interval_seconds 走 yaml，可关闭（<= 0） | ✅ |
| 事件主题决策 | 复用 task.failed（避免新增订阅方），加文档注释；C5 订阅器按 status 区分 | ✅ |
| acs 进程不启 sweeper | sweeper 仅注册到 worker；acs 不调 → 验证 cmd/acs/main.go 未改 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- 无

### INFO
- I-01：sweeper 使用 `go taskSweeper.Run(context.Background())` 直接 goroutine 起跑，沿用 worker 主进程其他长跑组件（reboot_closer / trace_sweeper 等）的同款 pattern；ctx 退出由 worker shutdown 整体处理。
- I-02：sweeper.go 中 `expirer.ExpireTask` 实际由 `TaskService.ExpireTask` 提供。这建立了一个隐式依赖环（task package 内 service 引用 sweeper 用的 interface） — 但 Go interface 是结构鸭式，无 import 环。验证：`go build` 通过。
- I-03：`ExpireTask.queue.Delete` 失败只 warn 不中断 — 因为任务可能已被 popper 拿走（race condition 时 sweeper 与 ACS popper 并发），属正常状态。Redis 不一致由下次启动重建（taskKey 24h TTL 已存）。
- I-04：设计文档 §4.6 已记下"复用 task.failed 主题"决策，C5 实施时直接对照。**这不是遗留问题**（已落地的合理调整），不进 §11 登记区。

---

## 与设计文档对齐

| §4.6 + §10.1-C2 项 | 实施 |
|---|---|
| worker 新增 task_sweeper.go | ✅ `internal/task/sweeper.go` |
| 周期扫 ExpiresAt < now AND status IN (pending,sent) | ✅ `ListExpiredCandidates` |
| MarkExpired + publish 事件 | ✅ `ExpireTask`（复用 notifyCompletion） |
| sweep_interval_seconds 配置项 | ✅ worker 4 份 yaml |
| 单元测试覆盖 | ✅ 8 个新测试 |

---

## 测试

- 8 个新测试（sweeper_test.go），全部使用 fakeLister + fakeExpirer，零外部依赖
- 覆盖：空候选 / 全成功 / 单条失败跳过 / lister 错误致命 / 参数透传 / 默认值 / ctx 取消退出
- 运行 < 0.5s

---

## 结论

**PASS**

可合入，可继续 C3（notifications 表迁移）。

# Verify Report — T-0101-g (OpsTask 暂停/恢复/取消 dispatcher 集成 — 状态机原子转换)

**Task**: T-0101-g — 暂停/恢复/取消 dispatcher 集成（PRD §5.3.1 状态机原子转换；不打断已发出的 RPC 只阻止后续设备）
**Type**: feat / F06/ops / P0 / S
**Sprint**: sprint-10 (pull-forward 续 T-0101-d 后)
**Deps**: T-0101-d ✅ (刚 done — approval_state 持久化 + status 联动)
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 改动 | 行数 |
|------|------|------|
| `omcgo/internal/ops/repository.go` | TaskRepository 接口 +TransitionStatus | +4 |
| `omcgo/internal/ops/pg_repository.go` | 新增 ErrInvalidStateTransition sentinel + TransitionStatus 方法（单 SQL CAS UPDATE + COALESCE setStartedAt 保留首次启动时刻）| +48 |
| `omcgo/internal/ops/service.go` | Cancel/Pause/Resume 三方法全部用 TransitionStatus 重写（原 GetByID+check+UpdateStatus 三步替换为单步原子）+ 新增 disambiguateInvalidTransition helper 保 NotFound vs 400 API 契约 | +51 / -33 |
| `omcgo/internal/ops/service_test.go` | mockTaskRepo +transitionStatusFn 字段+方法；改造既有 6 测试（FromPending/FromCompleted/FromRunning/FromPaused × Cancel/Pause/Resume）+ 3 NotFound 测试 + 新增 1 atomic CAS rejection 测试 = 10 测试用新模式 | +75 / -55 |
| `omcgo/internal/ops/handler_test.go` | mockOpsTaskRepo +TransitionStatusFn 字段+方法（接口扩展同步）| +9 |

---

## §2. 状态机原子转换设计（PRD §5.3.1 + Notes）

**之前（非原子，三步）**：
```go
task, _ := repo.GetByID(ctx, id)   // 步1：读
if task.Status != Running { ... }   // 步2：检查
repo.UpdateStatus(ctx, task)        // 步3：写
```
race window：两个并发请求都通过步2检查，都执行步3 → 后者覆盖前者，竞态

**之后（原子 CAS，一步）**：
```sql
UPDATE ops_tasks
SET status='paused', updated_at=NOW()
WHERE id=$1 AND status = ANY('{running}')
```
单 atomic UPDATE 含 CAS guard，RowsAffected=1 表示成功，RowsAffected=0 表示 CAS 失败。并发请求最多一个 commit。

**API 契约保持**：CAS 命中 0 行时调 disambiguateInvalidTransition helper 做 secondary GetByID：
- 任务不存在 → ErrNotFound（HTTP 404）
- 任务存在但状态非允许 → BusinessError 8100/8101/8102（HTTP 400）

**dispatcher 协作 hook**（subtask Notes "不打断已发出的 RPC，只阻止后续设备"）：
- 本任务**仅持久化** task.Status=paused/cancelled
- T-0101-a/b dispatcher 落地后须在每步 RPC 发出前 poll task.Status；命中 paused/cancelled 停推后续设备
- 已发出的 RPC 由 ACS 会话自然完成，不被打断（subtask Notes 原则）
- 本任务不实施 dispatcher 侧逻辑（T-0101-a/b deps 未就绪），留 future hook

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `go build ./...` | ✅ PASS | 无输出 |
| `go test -race -count=1 ./internal/ops/...` | ✅ PASS | 1.038s（10 改造测试 + 1 新 atomic CAS rejection 测试 + 既有 ops 测试全绿）|
| 无 TODO/FIXME/panic | ✅ | grep 验 |
| 无 `any` | ✅ | TransitionStatus 显式 OpsTaskStatus 类型 |
| 无 carrier 硬编码 | N/A | 无运营商分支 |
| 新端点 R | 0 | 复用既有 ApproveTask/CancelTask/PauseTask/ResumeTask handler |
| 新 metric/log 名 | 0 | zap log 保留 |
| 迁移 up/down | N/A | 无新迁移（CAS guard 仅 SQL 层） |
| 累计型 deps | N/A | |

---

## §4. 测试场景核销

| 测试 | 验证不变量 |
|------|-----------|
| `Cancel/Pause/Resume_FromHappyPath` (3 测) | transitionStatusFn 收到正确的 (validFrom, to, setStartedAt, setCompletedAt) 参数；Cancel 允许 from pending/running/paused 三态；Pause 仅 running；Resume 仅 paused + setStartedAt=true |
| `Cancel/Pause/Resume_WrongState` (3 测) | CAS 命中 0 行 + secondary GetByID 看到非允许状态 → BusinessError 8100/8101/8102 |
| `Cancel/Pause/Resume_NotFound` (3 测) | CAS 命中 0 行 + secondary GetByID 返 ErrNotFound → 传播 NotFound（HTTP 404，API 契约保持）|
| `PauseTask_AtomicCASRejection` (新增) | 模拟并发 race：自己 CAS 失败 + secondary GetByID 看到 status=paused → disambiguate 调用 + BusinessError 8101 |

**关键不变量 4 个**：
1. CAS guard 原子性（并发 race-safe）
2. Cancel 允许 from 三态（pending/running/paused 都可取消）
3. NotFound vs WrongState API 契约保持（404 vs 400 分离）
4. setStartedAt COALESCE 保留首次启动时刻（Resume 不覆盖原 StartedAt）

---

## §5. SQL CAS 设计

```sql
-- TransitionStatus(taskID, validFrom=[running], to=paused, setStartedAt=false, setCompletedAt=false):
UPDATE ops_tasks
SET status = 'paused', updated_at = NOW()
WHERE id = $1 AND status = ANY('{running}')

-- TransitionStatus(taskID, validFrom=[paused], to=running, setStartedAt=true, setCompletedAt=false):
UPDATE ops_tasks
SET status = 'running', updated_at = NOW(), started_at = COALESCE(started_at, NOW())
WHERE id = $1 AND status = ANY('{paused}')

-- TransitionStatus(taskID, validFrom=[pending,running,paused], to=cancelled, setStartedAt=false, setCompletedAt=true):
UPDATE ops_tasks
SET status = 'cancelled', updated_at = NOW(), completed_at = NOW()
WHERE id = $1 AND status = ANY('{pending,running,paused}')
```

**防护**：
- 列名 hard-coded（无字符串拼接）
- Squirrel `sq.Eq{"status": validStrs}` 自动 → `status = ANY($N)` PG 数组占位符
- pgx 自动 []string → text[] 数组绑定

---

## §6. 用户回归路径

需 staging 多 worker 并发场景验证 atomic CAS：
1. operator 创建任务（status=running）
2. 同时 2 个 sys_admin 同时点 Pause → 只一个 commit，另一返 BusinessError 8101
3. 触发 Resume → status=running，started_at 不变
4. 触发 Cancel → status=cancelled, completed_at 设置

单元测试覆盖的并发 race 由 mock 模拟（mockTaskRepo.transitionStatusFn 返 ErrInvalidStateTransition 模拟另一 goroutine 已 commit 场景）

---

## §7. S4 出口门

- [✓] 所有命令绿（go build + ops 包 go test -race + 10 改造测试 + 1 新 CAS rejection 测试）
- [N/A] E/R ≥ 1（无新后端端点）
- [N/A] 迁移双向演练（无新迁移）
- [N/A] metric/log 名（无新埋点）
- [N/A] 累计型依赖阈值

**S4 PASS**。

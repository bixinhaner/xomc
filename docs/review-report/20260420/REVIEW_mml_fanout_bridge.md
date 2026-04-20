# Code Review Report

**Date**: 2026-04-20
**Scope**: mml, task
**Type**: feat
**Conclusion**: PASS_WITH_WARNINGS

---

## Summary

实现 MML 任务到 device_tasks 的扇出逻辑，以及旧 cmdqueue 到 device_tasks 的桥接适配器。

## Changes

| File | Type | Description |
|------|------|-------------|
| `internal/mml/fanout.go` | NEW | MML 任务扇出：commands × devices → device_tasks |
| `internal/mml/result_aggregator.go` | NEW | device_task 完成后回写 mml_task 统计并自动终态化 |
| `internal/task/bridge_queue.go` | NEW | 旧 cmdqueue.CommandQueue 接口桥接到 device_tasks |
| `internal/mml/param_*.go` | NEW | 参数库后端（handler/model/repository/service） |
| `internal/mml/service.go` | MOD | ExecuteCommand/StartTask 中集成扇出调用 |
| `internal/mml/repository.go` | MOD | TaskRepository 新增 IncrementStats 方法 |
| `internal/mml/pg_repository.go` | MOD | IncrementStats 实现（原子递增） |
| `internal/task/model.go` | MOD | 新增 TaskSourceMML、ParentTaskID/CommandIndex/DeviceIndex、CommandKey |
| `internal/task/pg_repository.go` | MOD | Create/BatchCreate/scan 支持 3 个新列，删除未用的 scanTaskFromRows |
| `internal/task/service.go` | MOD | TaskCompletionCallback 回调机制 |
| `cmd/app/provider/modules.go` | MOD | DI 接线：Fanouter + ResultAggregator |
| `migrations/000023_*.sql` | NEW | device_tasks 增加 parent_task_id/command_index/device_index |

## Findings

### WARNING (3)

1. **`result_aggregator.go:65` — 并发竞态风险**
   `IncrementStats` + `finalizeIfComplete` 非原子操作。多个 device_task 同时完成时，多个回调可能并发进入 `finalizeIfComplete`，导致对同一个 mml_task 并发 `Update`。
   **建议**: 使用 `SELECT ... FOR UPDATE` 或在 finalizeIfComplete 中用 CAS 乐观锁（检查 done == total 再更新）。
   **当前风险**: 可接受。Update 只是覆盖同值，不会数据损坏。

2. **`fanout.go:77` — 大批量扇出无分页**
   当 commands × devices 数量巨大（如 100 commands × 1000 devices = 10 万条），`BatchCreateTasks` 会一次插入大量数据。
   **建议**: 当 reqs 数量超过阈值（如 1000）时，分批调用 BatchCreateTasks。
   **当前风险**: 当前阶段设备规模有限，暂可接受。

3. **`bridge_queue.go:99` — Clear 为空实现**
   `Clear` 方法返回 nil 但不做任何事，可能误导调用方认为已清理。
   **建议**: 添加日志警告或返回 "not supported" 错误。
   **当前风险**: 低。Clear 很少被调用。

### INFO (2)

1. **`pg_repository.go` 删除了 `scanTaskFromRows`** — 该方法已被标记为 unused，删除合理。

2. **`fanout.go:69` — json.Marshal 失败静默忽略** — `params, _ = json.Marshal(p)` 忽略了序列化错误。理论上 `map[string]interface{}` 不会序列化失败，但加个日志更好。

---

## Checklist

- [x] 命名规范（PascalCase/camelCase）
- [x] 错误处理（fmt.Errorf + %w 包装）
- [x] SQL 安全（Squirrel 参数化，sq.Expr 安全使用）
- [x] 接口优先（DeviceTaskCreator 解耦循环依赖）
- [x] 迁移幂等（ADD COLUMN IF NOT EXISTS, CREATE INDEX IF NOT EXISTS）
- [x] Down 迁移完整
- [x] 无硬编码运营商逻辑
- [x] 无资源泄漏（rows.Close 有 defer）
- [x] 日志质量（zap 结构化字段）

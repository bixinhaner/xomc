# 任务对账扫描根治设计

## 背景与证据

2026-07-31 清洁部署 20,000 台 LTE 模拟设备后，worker 每分钟执行一次
`device_tasks` 活跃任务对账。查询固定返回最早的 100 行，持续耗时
1.07–1.40 秒：

```sql
SELECT <task columns>
FROM device_tasks
WHERE status IN ('pending', 'sent')
  AND created_at < $1
ORDER BY created_at ASC
LIMIT 100;
```

现有索引只覆盖 `status='pending'`，不能同时覆盖 `sent`。查询还没有游标；
只要最早 100 条一直处于正常活跃态，后续任务就不会被检查，形成对账饥饿。

## 目标

1. 活跃任务对账使用 `(created_at, id)` 稳定键集分页。
2. 每轮只读一个有界批次；到达尾部后下一轮从头开始。
3. 正常活跃任务不会永久挡住后续 Redis/PG 分叉任务。
4. PostgreSQL 使用活跃态部分索引直接完成过滤和排序。
5. 不改变任务状态机、重试、过期、Redis/PG 双写真相或业务 API。

## 方案

### 仓储查询

新增 `ActiveTaskCursor`，并提供
`ListActiveTasksAfter(ctx, olderThan, after, limit)`。查询固定按
`created_at, id` 排序；游标非空时增加：

```sql
(created_at, id) > ($cursor_created_at, $cursor_id)
```

保留 `ListActiveTasks` 作为无游标兼容入口，避免扩大调用方改动。

### 对账器轮转

`Reconciler` 保存上轮最后一行游标：

- 本轮返回数据时，把游标推进到最后一行。
- 带游标查询返回空时，清空游标；下一轮自然从头开始。
- 查询失败时不推进游标。
- 单轮最多执行一次数据库查询，不在同一轮回绕，避免空闲轮额外放大数据库访问。

游标只影响扫描顺序，不改变既有 Redis 真相检查和条件修复。

### 数据库索引

增加分区父表部分索引：

```sql
CREATE INDEX IF NOT EXISTS idx_device_tasks_active_created_id
ON public.device_tasks (created_at, id)
WHERE status IN ('pending', 'sent');
```

该索引同时满足过滤、稳定排序和键集游标；部分索引只包含活跃任务，避免给大量终态历史
记录增加无意义索引体积。

## 验证

- 单元测试先证明旧代码无法向后翻页、无法在尾部回绕。
- SQL 契约测试证明生成查询包含稳定排序和元组游标。
- 迁移契约测试证明部分索引存在且只覆盖 `pending/sent`。
- `go test ./internal/task -count=1`、`go build ./...`、`go test ./... -count=1`。
- 部署后连续观察至少三个对账周期：
  - 同一查询不再出现超过 1 秒慢查询；
  - Redis/PG 差异为 0；
  - 对账修复仍可发生；
  - 任务队列年龄和旧版本 TTL 残留继续回落。


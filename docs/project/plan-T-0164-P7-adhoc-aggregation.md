# T-0164-P7 / G7 自定义聚合任务（oneshot + continuous）Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 用户即兴选 N 个设备做异步聚合（不沉淀设备组）；支持 oneshot（一次执行后归档）+ continuous（按 cron 重复执行）两种模式；任务式提交 + 进度可见；结果落 `pm_adhoc_aggregation_results` 快照表；全局保留期统一管理（drop_chunks 整块清，默认 1 年）；panel ↔ task 独立生命周期。

**Architecture:** 复用 PM 模块的 `pm_tasks` 表（per-module，不进 G8 async_jobs，设计 §4.7 决策）。扩 pm_tasks 加 adhoc_aggregation 类型 + 模式标记 + cron expr 字段。worker 端有独立 pm_adhoc worker 池消费 pm_tasks；continuous 模式与 G5 cron 时刻对齐。

**Tech Stack:** Go + pgx + Squirrel + TimescaleDB（hypertable + drop_chunks）+ SSE（已有 events hub）+ pm_tasks 表（已有）。

**Deps:** T-0164-P3 ✅（pm_metrics 数据源）+ T-0164-P8 ✅（asyncjob 不进，但 LeaderElector 共用）。

---

## 0. 数据模型

**pm_tasks 扩展**（已有表，仅加列）：
```sql
ALTER TABLE pm_tasks ADD COLUMN task_subtype TEXT;   -- 'adhoc_aggregation' / 其他既有 subtype
ALTER TABLE pm_tasks ADD COLUMN mode TEXT;            -- 'oneshot' / 'continuous'
ALTER TABLE pm_tasks ADD COLUMN cron_expr TEXT;       -- continuous 时有值
ALTER TABLE pm_tasks ADD COLUMN device_sns TEXT[];   -- 用户选的设备列表
ALTER TABLE pm_tasks ADD COLUMN metric_paths TEXT[];
ALTER TABLE pm_tasks ADD COLUMN granularities TEXT[];  -- 多粒度多选
ALTER TABLE pm_tasks ADD COLUMN window_start TIMESTAMPTZ;
ALTER TABLE pm_tasks ADD COLUMN window_end TIMESTAMPTZ;
ALTER TABLE pm_tasks ADD COLUMN progress_pct INT DEFAULT 0;  -- 0-100
```

**新表 pm_adhoc_aggregation_results**：
```sql
CREATE TABLE pm_adhoc_aggregation_results (
  task_id        UUID NOT NULL,
  device_sn      TEXT NOT NULL,
  metric_path    TEXT NOT NULL,
  metric_type    TEXT NOT NULL,
  metric_value   DOUBLE PRECISION NOT NULL,
  statis_type    TEXT,
  granularity    TEXT NOT NULL,
  time           TIMESTAMPTZ NOT NULL,
  start_time     TIMESTAMPTZ NOT NULL,
  end_time       TIMESTAMPTZ NOT NULL,
  ingest_time    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  extra          JSONB
);

SELECT create_hypertable('pm_adhoc_aggregation_results', 'time', chunk_time_interval => INTERVAL '30 days');
CREATE INDEX idx_pm_adhoc_results_task ON pm_adhoc_aggregation_results (task_id, time DESC);
-- 全局保留期通过 G2 sys_configs 'pm.retention.adhoc_aggregation_days' 默认 365
-- retention policy 由 G2 reload 动态调整
SELECT add_retention_policy('pm_adhoc_aggregation_results', INTERVAL '365 days');
```

**生命周期**：
- task 删除 → 不级联删 results（panel ↔ task 独立）
- results drop_chunks 整块清（全局保留期）
- task 状态：pending / running / succeeded / failed / canceled / scheduled（continuous 等待下次 tick）

## 1. 文件结构

**新建**：
- `omcgo/migrations/000169_extend_pm_tasks_and_create_adhoc_results.sql`
- `omcgo/internal/pm/adhoc/model.go`
- `omcgo/internal/pm/adhoc/service.go` — 创建/列表/取消
- `omcgo/internal/pm/adhoc/oneshot.go` — 一次性执行
- `omcgo/internal/pm/adhoc/continuous.go` — cron 模式
- `omcgo/internal/pm/adhoc/worker.go` — 独立 worker 池消费 pm_tasks
- `omcgo/internal/pm/adhoc/handler.go` — REST API
- `omcgo/internal/pm/adhoc/pg_repository.go`
- `omcgo/internal/pm/adhoc/*_test.go`

**修改**：
- `omcgo/cmd/worker/main.go` — 启动 adhoc worker 池
- `omcgo/cmd/app/router/router.go` — 挂 REST 路由

**前端**（业务层在 frontend-core，G6 plan 内做集成）：
- `omcmb/frontend-core/src/services/api/adhocAggregationApi.ts`
- `omcmb/frontend-core/src/hooks/api/useAdhocAggregation.ts`
- `omcmb/webcode/src/pages/performance/AdhocAggregation/index.tsx`（G6 plan 内构建）

## 2. Tasks

### Task 1: migration

**Files:**
- Create: `omcgo/migrations/000169_extend_pm_tasks_and_create_adhoc_results.sql`

完整 Up：ALTER pm_tasks 7 列 + CREATE pm_adhoc_aggregation_results + 索引 + hypertable + retention。Down 反向。

- [ ] migration up/down/up 三轮幂等

### Task 2: model + Repository

```go
type Mode string
const (
    ModeOneshot   Mode = "oneshot"
    ModeContinuous Mode = "continuous"
)

type Status string
const (
    StatusPending   Status = "pending"
    StatusRunning   Status = "running"
    StatusSucceeded Status = "succeeded"
    StatusFailed    Status = "failed"
    StatusCanceled  Status = "canceled"
    StatusScheduled Status = "scheduled"  // continuous 等下次 tick
)

type AdhocTask struct {
    ID            uuid.UUID
    Name          string
    Mode          Mode
    CronExpr      *string
    DeviceSNs     []string
    MetricPaths   []string
    Granularities []string
    WindowStart   time.Time
    WindowEnd     time.Time
    Status        Status
    ProgressPct   int
    CreatedBy     uuid.UUID
    // ... timestamps
}

type Repository interface {
    Create(ctx, task) error
    Update(ctx, task) error
    Get(ctx, id) (*AdhocTask, error)
    List(ctx, filter) ([]AdhocTask, error)
    Cancel(ctx, id) error
    LockNextPending(ctx, lockOwner) (*AdhocTask, error)  // for worker
    InsertResults(ctx, taskID, results []ResultRow) error
}
```

- [ ] TDD 6 case → 全过

### Task 3: oneshot.go + continuous.go 执行逻辑

**Files:**
- Create: `omcgo/internal/pm/adhoc/oneshot.go`
- Create: `omcgo/internal/pm/adhoc/continuous.go`

oneshot：
```go
func (e *Executor) ExecuteOneshot(ctx context.Context, task *AdhocTask) error {
    total := len(task.DeviceSNs) * len(task.MetricPaths) * len(task.Granularities)
    done := 0
    
    for _, g := range task.Granularities {
        // 复用 aggregator.Aggregator.Query 拉 15min 原始数据
        rows, err := e.aggregator.Query(ctx, aggregator.QueryRequest{
            Granularity: metrics.Granularity(g),
            DeviceSNs:   task.DeviceSNs,
            MetricPaths: task.MetricPaths,
            StartTime:   task.WindowStart,
            EndTime:     task.WindowEnd,
        })
        // 按 statis_type 聚合到目标粒度 → 写 pm_adhoc_aggregation_results
        results := aggregateResults(rows, g)
        e.repo.InsertResults(ctx, task.ID, results)
        
        done += len(task.DeviceSNs) * len(task.MetricPaths)
        progressPct := done * 100 / total
        e.repo.UpdateProgress(ctx, task.ID, progressPct)
        e.eventBus.Publish(ctx, "pm.adhoc.progress", map[string]any{"task_id": task.ID, "pct": progressPct})
    }
    return nil
}
```

continuous：每次 cron tick 调 ExecuteOneshot；execute 完不归档（status: succeeded → scheduled 等下次）。cron 调度时刻与 G5 对齐（避免 G5 cron 还在跑时同时 adhoc 大查询挤占）。

测试：
- ExecuteOneshot 完整跑通（mock aggregator）
- progress 上报次数 = len(granularities)
- continuous task 跑完一次后状态变 scheduled，下一 tick 重新跑

- [ ] TDD 3 case → 全过

### Task 4: worker.go pool

**Files:**
- Create: `omcgo/internal/pm/adhoc/worker.go`

```go
type Worker struct {
    repo     Repository
    executor *Executor
    lockOwner string
}

func (w *Worker) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done(): return
        case <-time.After(3 * time.Second):
            task, err := w.repo.LockNextPending(ctx, w.lockOwner)
            if task == nil { continue }
            w.runOne(ctx, task)
        }
    }
}

func (w *Worker) runOne(ctx, task *AdhocTask) {
    // ① UpdateStatus(running)
    // ② executor.Execute(Oneshot or Continuous)
    // ③ UpdateStatus(succeeded / failed / scheduled)
}
```

测试：mock + 跑 worker 一轮 → 抓 1 个 pending → 跑 → 状态正确。

- [ ] TDD 2 case → 全过

### Task 5: REST API handler + 路由

**Files:**
- Create: `omcgo/internal/pm/adhoc/handler.go`

端点：
- `POST   /api/v1/pm/adhoc/tasks` — 创建任务
- `GET    /api/v1/pm/adhoc/tasks` — 列表
- `GET    /api/v1/pm/adhoc/tasks/:id` — 详情
- `DELETE /api/v1/pm/adhoc/tasks/:id` — 取消（status: running→canceled）
- `GET    /api/v1/pm/adhoc/tasks/:id/results` — 查结果（分页）
- `GET    /api/v1/pm/adhoc/tasks/:id/progress` — SSE 进度推送（订阅 pm.adhoc.progress 事件）

测试：handler 单测 + 集成（创建→列表→详情→cancel）。

- [ ] TDD 5 case → 全过

### Task 6: worker main 接线 + 路由挂载 + commit

**Files:**
- Modify: `omcgo/cmd/worker/main.go`
- Modify: `omcgo/cmd/app/router/router.go`

worker main：
```go
adhocExecutor := adhoc.NewExecutor(aggregator, eventBus, adhocRepo)
adhocWorkers := make([]*adhoc.Worker, 4)
for i := 0; i < 4; i++ {
    adhocWorkers[i] = adhoc.NewWorker(adhocRepo, adhocExecutor, fmt.Sprintf("%s-adhoc-%d", hostname, i))
    go adhocWorkers[i].Run(ctx)
}
```

router.go：
```go
adhoc := router.Group("/pm/adhoc")
adhoc.Use(middleware.RequireAuth)
adhoc.POST("/tasks", adhocHandler.Create)
adhoc.GET("/tasks", adhocHandler.List)
// ...
```

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/adhoc/...

# 集成测试：创建一个 adhoc 任务 → 等 worker 处理 → 查结果
curl -X POST localhost:8081/api/v1/pm/adhoc/tasks -H "Authorization: Bearer $TOKEN" -d '{
  "name":"test adhoc",
  "mode":"oneshot",
  "device_sns":["TEST-001"],
  "metric_paths":["L.Cell.Avail.Dur"],
  "granularities":["hourly"],
  "window_start":"2026-05-22T00:00:00+08:00",
  "window_end":"2026-05-22T23:59:59+08:00"
}'
```

commit message：
```
feat(pm): 实施 G7 自定义聚合任务（oneshot + continuous）+ 进度 SSE

What: migration 000169 扩 pm_tasks 加 7 列（task_subtype/mode/cron_expr/device_sns/metric_paths/granularities/window_start/window_end/progress_pct）+ 新建 pm_adhoc_aggregation_results hypertable（retention 365d）；新建 internal/pm/adhoc 包：model/repository/oneshot/continuous/worker/handler；4 个 adhoc worker 在 worker 进程内 LockNextPending 抢任务；REST 6 端点；SSE 进度推送。
Why: G7 设计文档 §4.7；用户即兴选 N 设备做异步聚合不沉淀设备组；oneshot 一次执行后归档 / continuous 按 cron；走 pm_tasks per-module 不进 G8（设计锁定）；panel ↔ task 独立生命周期（删 task 不级联 results）。
Impact: 新增 1 张 hypertable + pm_tasks 7 列 + 6 REST 端点 + 4 worker goroutine；continuous 模式 cron 调度时刻与 G5 对齐避免挤占；前端入口由 G6 plan 内集成。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P7
Review: <审查报告路径>
```

- [ ] go test 全过 + curl 集成测试通过 → /commit skill

## 3. 验收

- [ ] migration up + down + up 三轮幂等
- [ ] pm_tasks 7 新列在；pm_adhoc_aggregation_results hypertable + retention 在
- [ ] adhoc Repository 6 case 全过
- [ ] oneshot + continuous 单测全过
- [ ] worker 单测全过
- [ ] handler 5 case 全过
- [ ] 集成：POST 创建 task → GET 列表 1 行 → 5-10 秒后状态 succeeded → GET results 有行 → SSE 收到 progress 事件

## 4. Out of scope

- 前端 AdhocAggregation 页面 → G6 plan 集成
- panel ↔ task 派生关系（fork 时引用 task_id）→ G6 plan
- 复杂结果查询（多 task JOIN、时序对比）→ 留 GA
- continuous 模式手动触发 → 不做

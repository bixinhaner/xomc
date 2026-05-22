# T-0164-P8 / G8 通用任务框架（async_jobs）Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 提供进程级通用异步任务框架，承载 G5 cron 实例（小时/日/周/月聚合）与未来批量计算任务；支持心跳上报、僵尸检测、cron 触发补跑、PG advisory 分布式锁、进程重启续跑。

**Architecture:** 数据库表 `async_jobs` 作为任务实例总线（每个 cron 触发产生一行）+ `JobRunner` 接口由各业务模块注册（如 G5 注册 4 个 HourlyAggregator/DailyAggregator/...）+ 守护 goroutine 心跳上报 + Sweeper 定期扫表识别僵尸 + Scheduler cron 触发补跑（基于 LeaderElector PG advisory lock 单实例选主）。

**Tech Stack:** Go + pgx PG advisory lock + cron expression（github.com/robfig/cron/v3，仓库已用）+ context + ticker。

**Deps:** 无（独立；G5/G7 注册到本框架）。

**G7 不进 G8**（设计 §4.7/§4.8 锁定）：G7 走 `pm_tasks` per-module；G8 仅承载 G5 cron + 未来批量计算。

---

## 0. 表 Schema 与状态机

```sql
CREATE TABLE async_jobs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_type        TEXT NOT NULL,              -- 'pm_aggregate_hourly' / 'pm_aggregate_daily' / ...
  status          TEXT NOT NULL CHECK IN ('pending','running','succeeded','failed','zombie','canceled'),
  schedule_expr   TEXT,                       -- cron expression (only for cron-triggered)
  scheduled_at    TIMESTAMPTZ NOT NULL,       -- 计划执行时刻（cron 解析后）
  started_at      TIMESTAMPTZ,
  finished_at     TIMESTAMPTZ,
  heartbeat_at    TIMESTAMPTZ,                -- running 中每 30s 上报
  lock_owner      TEXT,                       -- worker 进程 ID
  attempt         INT NOT NULL DEFAULT 1,
  payload         JSONB,                      -- job 参数（如聚合的时间窗）
  result          JSONB,                      -- 成功结果（如聚合行数）
  error_message   TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_async_jobs_status_scheduled ON async_jobs (status, scheduled_at);
CREATE INDEX idx_async_jobs_type_status      ON async_jobs (job_type, status);
CREATE INDEX idx_async_jobs_heartbeat        ON async_jobs (heartbeat_at) WHERE status='running';
```

**状态机**：
```
pending → running → succeeded
              ↓        ↓
              ↓     failed (attempt < max → retry pending)
              ↓
            zombie (heartbeat_at < now - 5min, by Sweeper)
              ↓
            pending (Sweeper 重新置 pending 等下次拉)
```

## 1. 文件结构

**新建**：
- `omcgo/migrations/000167_create_async_jobs.sql`
- `omcgo/internal/core/asyncjob/model.go` — Job 类型 + 状态常量
- `omcgo/internal/core/asyncjob/runner.go` — JobRunner 接口 + Registry
- `omcgo/internal/core/asyncjob/scheduler.go` — cron 触发 + LeaderElector
- `omcgo/internal/core/asyncjob/heartbeat.go` — 守护 goroutine 心跳上报
- `omcgo/internal/core/asyncjob/sweeper.go` — 僵尸检测 + 重置
- `omcgo/internal/core/asyncjob/lock.go` — PG advisory lock helper
- `omcgo/internal/core/asyncjob/pg_repository.go` — Repository
- `omcgo/internal/core/asyncjob/*_test.go`

**修改**：
- `omcgo/cmd/worker/main.go` — 启动期注册 asyncjob.Scheduler + Sweeper 进 worker goroutine

## 2. Tasks

### Task 1: migration 创建 async_jobs

**Files:**
- Create: `omcgo/migrations/000167_create_async_jobs.sql`

Up + Down 配对（CREATE + 3 索引 / DROP 顺序倒置）。

- [ ] migration up + down + up 三轮幂等

### Task 2: model.go + Repository

**Files:**
- Create: `omcgo/internal/core/asyncjob/model.go`
- Create: `omcgo/internal/core/asyncjob/pg_repository.go`
- Test: pg_repository_test.go

```go
type Status string
const (
    StatusPending   Status = "pending"
    StatusRunning   Status = "running"
    StatusSucceeded Status = "succeeded"
    StatusFailed    Status = "failed"
    StatusZombie    Status = "zombie"
    StatusCanceled  Status = "canceled"
)

type Job struct {
    ID, JobType, Status, ScheduleExpr string
    ScheduledAt, StartedAt, FinishedAt, HeartbeatAt time.Time
    LockOwner string
    Attempt int
    Payload, Result json.RawMessage
    ErrorMessage string
    CreatedAt, UpdatedAt time.Time
}

type Repository interface {
    Insert(ctx, job) (uuid.UUID, error)
    UpdateStatus(ctx, id, status, errMsg) error
    UpdateHeartbeat(ctx, id) error
    LockNextPending(ctx, jobType, lockOwner) (*Job, error)   // SELECT FOR UPDATE SKIP LOCKED
    ListZombies(ctx, heartbeatThreshold time.Duration) ([]Job, error)
    ResetZombie(ctx, id) error   // status running→pending + attempt+1
}
```

LockNextPending 用 `SELECT ... FOR UPDATE SKIP LOCKED` 模式 + WHERE status='pending' AND scheduled_at<=NOW() AND job_type=$1 LIMIT 1。

- [ ] TDD 6 case：Insert / UpdateStatus / UpdateHeartbeat / LockNextPending（无可用返 nil）/ ListZombies / ResetZombie

### Task 3: runner.go + Registry

**Files:**
- Create: `omcgo/internal/core/asyncjob/runner.go`

```go
type JobRunner interface {
    JobType() string
    Run(ctx context.Context, job *Job) (result json.RawMessage, err error)
}

type Registry struct {
    runners map[string]JobRunner
    repo    Repository
    lockOwner string  // worker process ID, e.g. hostname-pid
}

func (r *Registry) Register(runner JobRunner) { r.runners[runner.JobType()] = runner }

func (r *Registry) RunNext(ctx context.Context, jobType string) error {
    job, err := r.repo.LockNextPending(ctx, jobType, r.lockOwner)
    if job == nil { return nil /* no pending */ }
    // ① UpdateStatus(running) + heartbeat start
    // ② runner.Run(ctx, job) — 在专门 goroutine
    // ③ heartbeat ticker 每 30s UpdateHeartbeat
    // ④ ctx.Done() / runner panic → UpdateStatus(failed) + 重试
    // ⑤ success → UpdateStatus(succeeded) + result
}
```

测试：
- runner.Run 成功 → status=succeeded
- runner.Run panic → 不破坏外层 + status=failed + error_message
- ctx cancel → status=failed + error_message="context canceled"

- [ ] TDD 3 case → 全过

### Task 4: heartbeat + sweeper

**Files:**
- Create: `omcgo/internal/core/asyncjob/heartbeat.go`
- Create: `omcgo/internal/core/asyncjob/sweeper.go`

heartbeat：每 30s tick → repo.UpdateHeartbeat(jobID) → 与 runner.Run 同生命周期，runner 退出停 ticker。

sweeper：每 60s tick → repo.ListZombies(heartbeatThreshold=5min) → 对每个 zombie repo.ResetZombie（status running→pending + attempt+1）。

测试：
- sweeper 识别 heartbeat 过期 5min 的 running → 重置 pending
- attempt 达上限 → 直接 failed 不重置
- 多 worker 跑 sweeper 并发 → PG row-level lock 保证一致

- [ ] TDD 3 case → 全过

### Task 5: scheduler + LeaderElector

**Files:**
- Create: `omcgo/internal/core/asyncjob/scheduler.go`
- Create: `omcgo/internal/core/asyncjob/lock.go`

scheduler 用 `github.com/robfig/cron/v3`：
```go
type Scheduler struct {
    cron        *cron.Cron
    repo        Repository
    leader      *LeaderElector  // PG advisory lock，保证多 worker 只有 1 个触发
}

func (s *Scheduler) Schedule(jobType, cronExpr string, payloadBuilder func() json.RawMessage) error {
    _, err := s.cron.AddFunc(cronExpr, func() {
        if !s.leader.IsLeader() { return }
        s.repo.Insert(ctx, Job{
            JobType: jobType, Status: StatusPending,
            ScheduleExpr: cronExpr, ScheduledAt: time.Now(),
            Payload: payloadBuilder(),
        })
    })
    return err
}
```

lock.go：
```go
type LeaderElector struct {
    db       *pgxpool.Pool
    lockKey  int64        // pg_try_advisory_lock(int64)
    isLeader atomic.Bool
}

func (l *LeaderElector) Run(ctx context.Context) {
    // 每 30s 尝试 acquire；持有期间 keepalive；ctx done 释放
}
```

测试：scheduler tick / LeaderElector 多实例竞争。

- [ ] TDD → 全过

### Task 6: worker main 接线 + commit

**Files:**
- Modify: `omcgo/cmd/worker/main.go`

启动期：
```go
// after pool/redis/nats init
asyncRepo := asyncjob.NewPGRepository(pool)
leader := asyncjob.NewLeaderElector(pool, advisoryLockKey_pmAggregator)
go leader.Run(ctx)

asyncRegistry := asyncjob.NewRegistry(asyncRepo, hostname+"-"+strconv.Itoa(os.Getpid()))
// G5 实施时注册 4 个 aggregator runner
asyncRegistry.Register(g5HourlyAggregator)
asyncRegistry.Register(g5DailyAggregator)
// ...

scheduler := asyncjob.NewScheduler(asyncRepo, leader)
// G5 实施时挂 cron 触发器
scheduler.Schedule("pm_aggregate_hourly", "5 * * * *", buildHourlyPayload)
// ...
scheduler.Start()

sweeper := asyncjob.NewSweeper(asyncRepo)
go sweeper.Run(ctx)

// 主 worker loop：轮询每个 jobType
for _, jt := range []string{"pm_aggregate_hourly", "pm_aggregate_daily", ...} {
    go func(jobType string) {
        for {
            select {
            case <-ctx.Done(): return
            case <-time.After(5 * time.Second):
                asyncRegistry.RunNext(ctx, jobType)
            }
        }
    }(jt)
}
```

跑 `go build ./... && go test ./internal/core/asyncjob/...`。

commit message：
```
feat(core): 实施 G8 通用任务框架（async_jobs + 心跳 + 僵尸检测 + cron + 分布式锁）

What: migration 000167 创建 async_jobs 表 + 3 索引；新建 internal/core/asyncjob 包：model.go / runner.go (JobRunner 接口 + Registry) / scheduler.go (cron + LeaderElector PG advisory lock) / heartbeat.go (30s 守护 ticker) / sweeper.go (60s 僵尸检测 + 重置 pending) / lock.go / pg_repository.go (LockNextPending 用 SELECT FOR UPDATE SKIP LOCKED) + 完整单测。worker/main.go 启动期接线（leader / scheduler / sweeper / 每 jobType 轮询 goroutine）。
Why: G8 设计文档 §4.8；承载 G5 cron 实例（小时/日/周/月聚合）+ 未来批量计算任务；进程重启续跑（heartbeat 过期 5min 自动重置 pending）；多 worker 部署一致（PG advisory lock 选主 + SKIP LOCKED 抢任务）。
Impact: 新增 async_jobs 表与运行时 goroutine；G5 实施时将 4 个 aggregator 注册到本框架；G7 不进本框架（走 pm_tasks per-module，设计 §4.7 锁定）。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P8
Review: <审查报告路径>
```

- [ ] go test 全过 → /commit skill

## 3. 验收

- [ ] migration up + down + up 三轮幂等
- [ ] async_jobs 表 + 3 索引在
- [ ] Repository 6 单测全过
- [ ] Runner 3 单测全过（panic 不破外层）
- [ ] Heartbeat / Sweeper 3 单测全过
- [ ] Scheduler / LeaderElector 全过
- [ ] worker 启动后 `pgrep -fa omcgo-worker` 仍单进程；DB 查 `pg_locks WHERE locktype='advisory'` 有 leader 锁
- [ ] mock 注入一个测试 JobRunner → 5 分钟后状态从 pending→running→succeeded

## 4. Out of scope

- 实际的 G5 4 个 aggregator 注册 → G5 plan 内做
- G7 接入 → 不做（G7 走 pm_tasks）
- 任务历史归档 → 未来批量计算任务实施时再做
- Prometheus 指标（job_runtime_seconds 等）→ 留 W3 G 块（已在 backlog）

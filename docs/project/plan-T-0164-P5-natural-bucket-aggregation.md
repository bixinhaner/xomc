# T-0164-P5 / G5 自然日历桶预聚合 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** OMC 后台定时任务按整点对齐的自然日历桶（小时 / 日 / 周 / 月）预聚合到独立聚合表（设备维度 + 设备组维度），按 `statis_type` 元数据驱动聚合方式；报表 / 趋势查询按粒度 + 行维度路由到对应聚合表，避免大窗口扫原始表。

**Architecture:** 4 张聚合表（pm_metrics_hourly hypertable + daily/weekly/monthly 普通表）+ 4 个 cron job 注册到 G8 任务框架（@每小时 :05 / 每日 00:05 / 每周一 00:10 / 每月 1 日 00:15）+ 聚合 SQL 按 statis_type 路由（sum/avg/max 从 15min 算；pct 从当前级 counter 取值代入 arithmetic）+ QueryAggregated 按粒度路由。

**Tech Stack:** Go + Squirrel + pgx + TimescaleDB（hourly hypertable）+ G8 asyncjob 框架 + perf_indicators_* / rela_platform_indicator_formula_* 元数据。

**Deps:** T-0164-P3 ✅（pm_metrics 表 + metric_type + statis_type 已就位）+ T-0164-P1 ✅（KPIEngine + router 拿 KPI 子集）+ T-0164-P8 ✅（async_jobs 任务框架）。

---

## 0. 表设计

```sql
-- pm_metrics_hourly：hypertable（量级 ~ 1/15 × 原始）
CREATE TABLE pm_metrics_hourly (LIKE pm_metrics INCLUDING ALL);
SELECT create_hypertable('pm_metrics_hourly', 'time', chunk_time_interval => INTERVAL '7 days');
ALTER TABLE pm_metrics_hourly SET (timescaledb.compress, timescaledb.compress_segmentby='device_sn,metric_type');
SELECT add_compression_policy('pm_metrics_hourly', INTERVAL '14 days');
SELECT add_retention_policy('pm_metrics_hourly', INTERVAL '180 days');

-- pm_metrics_daily / weekly / monthly：普通表，按 G2 cron 清理
CREATE TABLE pm_metrics_daily (...);   -- 同 pm_metrics schema 但 PRIMARY KEY (device_sn, metric_path, end_time)
CREATE TABLE pm_metrics_weekly (...);
CREATE TABLE pm_metrics_monthly (...);

-- 设备组聚合（扩展）：把维度从 device_sn 换成 device_group_id
CREATE TABLE pm_group_metrics_hourly (
  device_group_id UUID NOT NULL,
  metric_path TEXT NOT NULL, metric_type TEXT, metric_value DOUBLE PRECISION,
  statis_type TEXT, granularity TEXT, time TIMESTAMPTZ,
  start_time, end_time TIMESTAMPTZ,
  ...
);
-- daily/weekly/monthly 同理
```

**为什么 daily/weekly/monthly 不用 hypertable**：量小（年级别 ~ 1500 行 × 设备 × KPI）；按月业务查询为主，普通 B-tree 索引足够；普通表 G2 cron 清理（drop_chunks 不适用）。

## 1. 文件结构

**新建**：
- `omcgo/migrations/000168_create_pm_aggregation_tables.sql`
- `omcgo/internal/pm/aggregator/aggregator.go` — 聚合主入口
- `omcgo/internal/pm/aggregator/hourly.go` / `daily.go` / `weekly.go` / `monthly.go` — 4 runner
- `omcgo/internal/pm/aggregator/device_group.go` — 设备组维度聚合扩展
- `omcgo/internal/pm/aggregator/query.go` — 按粒度路由 QueryAggregated
- `omcgo/internal/pm/aggregator/*_test.go`

**修改**：
- `omcgo/cmd/worker/main.go` — 注册 4 个 aggregator 到 G8 scheduler
- `omcgo/internal/pm/query/handler.go` — 把 QueryAggregated 调用切到 aggregator.Query

## 2. 聚合 SQL 模板（按 statis_type 路由）

**sum/avg/max 类 counter 从 15min 聚合**：
```sql
INSERT INTO pm_metrics_hourly (
  device_sn, metric_path, metric_type, metric_value, statis_type,
  granularity, time, start_time, end_time, object_ldn, extra
)
SELECT
  m.device_sn,
  m.metric_path,
  'counter',
  CASE pi.statis_type
    WHEN 'sum' THEN SUM(m.metric_value)
    WHEN 'avg' THEN AVG(m.metric_value)
    WHEN 'max' THEN MAX(m.metric_value)
  END,
  pi.statis_type,
  'hourly',
  date_trunc('hour', m.end_time),
  date_trunc('hour', m.end_time),
  date_trunc('hour', m.end_time) + INTERVAL '1 hour',
  m.object_ldn,
  m.extra
FROM pm_metrics m
JOIN perf_indicators_enb pi ON pi.standard_path = m.metric_path   -- 用 m.extra.indicator_device_type 路由 _enb/_gsm/_gnb
WHERE m.metric_type = 'counter'
  AND m.granularity = '15min'
  AND m.end_time >= $1   -- 上一小时起
  AND m.end_time <  $2   -- 上一小时止
  AND pi.statis_type IN ('sum','avg','max')
GROUP BY m.device_sn, m.metric_path, pi.statis_type, m.object_ldn, m.extra
ON CONFLICT (device_sn, metric_path, granularity, end_time)
DO UPDATE SET metric_value = EXCLUDED.metric_value, ingest_time = NOW();
```

**pct 类 KPI 从当前级 counter 聚合表取值代入 arithmetic**：
```go
// 不能直接 SQL 聚合 — 需查 rela_platform_indicator_formula_*.arithmetic
// 步骤：
// 1) SELECT counters from pm_metrics_hourly (本级，刚算完 sum/avg/max)
// 2) 按 formula deps 抽取相应 counter 值
// 3) 在 Go 内执行 arithmetic 计算（复用 G3 calculator.go）
// 4) INSERT 到 pm_metrics_hourly metric_type='kpi' statis_type='pct'
```

## 3. Tasks

### Task 1: migration 创建 4 聚合表 + 设备组扩展

**Files:**
- Create: `omcgo/migrations/000168_create_pm_aggregation_tables.sql`

完整 Up：4 设备维度表 + 4 设备组维度表 + 索引 + hypertable + compression + retention。Down 反向。

- [ ] migration up/down/up 三轮幂等

### Task 2: aggregator.go 主类型 + Common

**Files:**
- Create: `omcgo/internal/pm/aggregator/aggregator.go`

```go
type Aggregator struct {
    db        *pgxpool.Pool
    indicatorRepo indicator.Repository  // for statis_type lookup
    calculator    kpi.Calculator         // for pct arithmetic
}

type WindowSpec struct {
    Granularity metrics.Granularity
    Start, End  time.Time   // 自然桶起止
}

func (a *Aggregator) AggregateCounters(ctx context.Context, w WindowSpec, target string /* table name */) (rows int, err error) {
    // 执行 SQL 模板，按 statis_type 路由 INSERT
}

func (a *Aggregator) AggregateKPIs(ctx context.Context, w WindowSpec, target string) (rows int, err error) {
    // 1) 拉 KPI 公式列表（formula + dependencies）
    // 2) 对每个 KPI：查 target 表 counter 行 → arithmetic 求值
    // 3) INSERT KPI 行（metric_type='kpi', statis_type=KPI 自身 statis_type）
}
```

测试：mock 数据 + AggregateCounters 单元覆盖 sum/avg/max 三路；AggregateKPIs 覆盖 pct 路径。

- [ ] TDD 4 case → 全过

### Task 3: 4 个 cron runner（hourly/daily/weekly/monthly）

**Files:**
- Create: `omcgo/internal/pm/aggregator/hourly.go` / `daily.go` / `weekly.go` / `monthly.go`

每个 runner 实现 asyncjob.JobRunner 接口：
```go
type HourlyAggregator struct { *Aggregator }
func (r *HourlyAggregator) JobType() string { return "pm_aggregate_hourly" }
func (r *HourlyAggregator) Run(ctx, job *asyncjob.Job) (json.RawMessage, error) {
    // payload 含起止时间窗
    var p struct{ Start, End time.Time }
    json.Unmarshal(job.Payload, &p)
    
    rowsCounter, err := r.AggregateCounters(ctx, WindowSpec{...}, "pm_metrics_hourly")
    rowsKPI, err := r.AggregateKPIs(ctx, WindowSpec{...}, "pm_metrics_hourly")
    
    return json.Marshal(map[string]int{"counter_rows": rowsCounter, "kpi_rows": rowsKPI})
}
```

daily/weekly/monthly 同样模式，区别仅 target 表 + WindowSpec。

测试：每个 runner 单独 unit test（mock Aggregator，断言调用次数 + payload 解析正确）。

- [ ] TDD 4 case → 全过

### Task 4: device_group 维度聚合扩展

**Files:**
- Create: `omcgo/internal/pm/aggregator/device_group.go`

```go
func (a *Aggregator) AggregateDeviceGroup(ctx context.Context, w WindowSpec, deviceTarget, groupTarget string) error {
    // SQL：JOIN devices.device_group_id + GROUP BY device_group_id 聚合 device 表
    // 写到 pm_group_metrics_hourly / _daily / _weekly / _monthly
}
```

测试：mock 数据含 2 设备 → 1 group → 聚合后 group 维度行正确。

- [ ] TDD 1 case → 全过

### Task 5: Query.go 按粒度路由

**Files:**
- Create: `omcgo/internal/pm/aggregator/query.go`

```go
type QueryRequest struct {
    Granularity      metrics.Granularity
    Dimension        string  // 'device' | 'device_group'
    DeviceSNs        []string
    DeviceGroupIDs   []uuid.UUID
    MetricPaths      []string
    MetricType       *metrics.MetricType
    StartTime, EndTime time.Time
    Limit, Offset    int
}

func (a *Aggregator) Query(ctx context.Context, q QueryRequest) ([]metrics.PMMetric, error) {
    table := selectTable(q.Granularity, q.Dimension)
    // table ∈ {pm_metrics, pm_metrics_hourly, pm_metrics_daily, pm_metrics_weekly, pm_metrics_monthly,
    //         pm_group_metrics_hourly, ..._monthly}
    // 走 Squirrel 拼 SQL
}

func selectTable(g metrics.Granularity, dim string) string {
    switch g {
    case metrics.Granularity15Min:
        if dim == "device_group" { return "" /* not supported, fallback to device + JOIN */ }
        return "pm_metrics"
    case metrics.GranularityHourly:
        return ifGroup(dim, "pm_group_metrics_hourly", "pm_metrics_hourly")
    // ...
    }
}
```

测试：每个粒度 × 每个维度 → 走对应表（mock pool 断言 SQL 拼接）。

- [ ] TDD 10 case (5 粒度 × 2 维度) → 全过

### Task 6: worker main 注册 4 cron + 集成 + commit

**Files:**
- Modify: `omcgo/cmd/worker/main.go`

```go
// after G8 scheduler init
aggregator := aggregator.New(pool, indicatorRepo, kpi.Calculator)
asyncRegistry.Register(aggregator.HourlyAggregator())
asyncRegistry.Register(aggregator.DailyAggregator())
asyncRegistry.Register(aggregator.WeeklyAggregator())
asyncRegistry.Register(aggregator.MonthlyAggregator())

scheduler.Schedule("pm_aggregate_hourly", "5 * * * *", func() json.RawMessage {
    now := time.Now()
    prev := now.Add(-time.Hour).Truncate(time.Hour)
    return mustJSON(map[string]any{"Start": prev, "End": prev.Add(time.Hour)})
})
scheduler.Schedule("pm_aggregate_daily",   "5 0 * * *",  buildDailyPayload)
scheduler.Schedule("pm_aggregate_weekly",  "10 0 * * 1", buildWeeklyPayload)
scheduler.Schedule("pm_aggregate_monthly", "15 0 1 * *", buildMonthlyPayload)
```

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/aggregator/...

# 集成测试：SQL 注入 1 小时内 15min × 4 数据 → 触发 hourly aggregator → 查 pm_metrics_hourly 应有聚合行
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
INSERT INTO pm_metrics (device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time)
VALUES
  ('TEST-001', 'L.Cell.Avail.Dur', 'counter', 100, 'sum', '15min', '2026-05-22 10:00:00+08', '2026-05-22 09:45:00+08', '2026-05-22 10:00:00+08'),
  ('TEST-001', 'L.Cell.Avail.Dur', 'counter', 200, 'sum', '15min', '2026-05-22 10:15:00+08', '2026-05-22 10:00:00+08', '2026-05-22 10:15:00+08'),
  ('TEST-001', 'L.Cell.Avail.Dur', 'counter', 150, 'sum', '15min', '2026-05-22 10:30:00+08', '2026-05-22 10:15:00+08', '2026-05-22 10:30:00+08'),
  ('TEST-001', 'L.Cell.Avail.Dur', 'counter', 250, 'sum', '15min', '2026-05-22 10:45:00+08', '2026-05-22 10:30:00+08', '2026-05-22 10:45:00+08');

-- 手动触发 hourly aggregator job
INSERT INTO async_jobs (job_type, status, scheduled_at, payload) VALUES
  ('pm_aggregate_hourly', 'pending', NOW(),
   '{\"Start\":\"2026-05-22T09:00:00+08:00\",\"End\":\"2026-05-22T10:00:00+08:00\"}');
"
# 等 5-10 秒
sleep 10
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT * FROM pm_metrics_hourly WHERE device_sn='TEST-001';
SELECT status, result FROM async_jobs WHERE job_type='pm_aggregate_hourly' ORDER BY created_at DESC LIMIT 1;
"
```

期望：pm_metrics_hourly 有 1 行 metric_value=700（4 × counter sum）；async_jobs 状态 succeeded。

commit message：
```
feat(pm): 实施 G5 自然日历桶预聚合（hourly/daily/weekly/monthly）+ 设备组维度扩展

What: migration 000168 创建 8 张聚合表（4 设备维度 + 4 设备组维度，hourly hypertable + 其余普通表）；新建 internal/pm/aggregator 包：aggregator.go（counter sum/avg/max + KPI pct arithmetic 两阶段聚合）/ hourly.go / daily.go / weekly.go / monthly.go 4 runner（实现 G8 asyncjob.JobRunner）/ device_group.go 扩展 / query.go 按粒度+维度路由；worker main 注册 4 个 cron 触发器（@:05 / @00:05 / @周一 00:10 / @月 1 日 00:15）。
Why: G5 设计文档 §4.5；避免大窗口现场扫 15min 原始表；按 statis_type 元数据驱动聚合方式；为 G6 前端仪表盘提供高效查询源；G2 默认保留期自动接管。
Impact: 新增 8 张聚合表 + 4 个定时任务；查询接口按粒度路由到聚合表（15min 直查 pm_metrics，hourly+ 走聚合表）；KPI pct 走 arithmetic 上下文。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P5
Review: <审查报告路径>
```

- [ ] go test 全过 + 集成测试通过 → /commit skill

## 4. 验收

- [ ] migration up + down + up 三轮幂等
- [ ] 8 张聚合表 + 索引 + hypertable + policy 全在
- [ ] aggregator 单测全过（counter 三路 + KPI pct + device_group）
- [ ] 4 runner 注册成功（worker main 启动 log 含"registered hourly/daily/weekly/monthly aggregator"）
- [ ] cron 触发器在 scheduler 内（`docker exec omc-docker-worker psql -c "SELECT * FROM async_jobs ORDER BY created_at DESC LIMIT 4"` 应能看到当下小时 job 已 succeeded 或 pending）
- [ ] 手动 INSERT mock data + INSERT async_jobs → 等 10s → pm_metrics_hourly 出现聚合行（metric_value 正确）

## 5. Out of scope

- 实际生产 PM 数据触发完整链路 → T-0121 阻塞（真机不推 PM）
- pct 公式复杂 case（涉及多 indicator 表 JOIN）→ 留 G7 / 早上 review
- KPI Continuous Aggregate（TimescaleDB CA）→ 不用（设计 §4.5 已说明）
- 设备组聚合的 group_membership 变化时的回算 → 留早上

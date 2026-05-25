# T-0164-P3 / G3 合并表 + 删物化视图 + 聚合元数据驱动 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 合并 `pm_counters` + `kpi_values` 两张表为单表 `pm_metrics`（含 `metric_type` 区分 counter/kpi），删除 TimescaleDB 物化视图 `pm_counters_hourly`，KPIEngine 的 counter 聚合改为按 `perf_indicators_*.statis_type` 元数据驱动（sum/avg/max/pct 四路路由）。

**Architecture:** 单一时序表 + 元数据字段 + KPIEngine 实时计算简化。pm_metrics 作为 15min 粒度 hypertable，G5 在其上建 hourly/daily/weekly/monthly 聚合表。KPIEngine 在实时计算时不再做 SQL 聚合（聚合下沉到 G5 cron）；counter 聚合方式在 G5 阶段才用上 statis_type，但 G3 阶段已经把字段消费链路打通。

**Tech Stack:** Go + Squirrel + pgx + TimescaleDB（hypertable + compression + retention policy）+ perf_indicators_* 表（T-0098 已落库）。

**Deps:** T-0164-P4 ✅（pm_counters / kpi_values 已有 start_time/end_time/ingest_time 三字段）。

**风险**：项目未上生产，可直接 DROP 旧表 + 视图（用户 2026-05-22 拍板"删了数据也没事"）；migration 前用 `pg_dump` 备份到 `/tmp/pm-kpi-pre-G3-backup-20260522.sql`，commit message 说明本地无生产数据下线属预期。

---

## 0. pm_metrics Schema 设计

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | gen_random_uuid() |
| `device_sn` | TEXT NOT NULL | 设备序列号 |
| `metric_path` | TEXT NOT NULL | 指标路径（counter 或 KPI 标识） |
| `metric_type` | TEXT NOT NULL CHECK IN ('counter','kpi') | 区分 counter / kpi |
| `metric_value` | DOUBLE PRECISION NOT NULL | 数值 |
| `statis_type` | TEXT | counter 类必填（sum/avg/max/pct）；kpi 类 NULL |
| `granularity` | TEXT NOT NULL CHECK IN ('15min','hourly','daily','weekly','monthly') | 粒度 |
| `time` | TIMESTAMPTZ NOT NULL | hypertable 时间列（与 end_time 同值） |
| `start_time` | TIMESTAMPTZ NOT NULL | 采集起（基站时钟） |
| `end_time` | TIMESTAMPTZ NOT NULL | 采集止（基站时钟） |
| `ingest_time` | TIMESTAMPTZ NOT NULL DEFAULT NOW() | 入库时（OMC 时钟） |
| `object_ldn` | TEXT | measObjLdn 解析后对象 LDN |
| `extra` | JSONB | 扩展字段（如 carrier/tech/cell_id 等） |

**唯一性维度**：`(device_sn, metric_path, granularity, end_time)` 复合 UNIQUE INDEX（避免重复入库 + 补传去重）。

**TimescaleDB**：
- hypertable on `time`，chunk_time_interval = 1 day
- compression policy 7d 后压缩（G2 常量 `pm.retention.raw_15min_days` 暂未挂，本 migration hard-code 7d 后压缩；G2 配置化时 alter_job 联动）
- retention policy 30d 后删（G2 默认值，hard-code 在 migration）

## 1. 文件结构

**新建**：
- `omcgo/migrations/000166_create_pm_metrics_and_drop_legacy.sql` — DROP 旧 + CREATE 新 + 挂 hypertable + compression + retention

**修改**：
- `omcgo/internal/pm/counter/model.go` + `pg_repository.go` → 改写到 `pm_metrics` + metric_type='counter'
- `omcgo/internal/pm/kpi/model.go` + `pg_repository.go` → 改写到 `pm_metrics` + metric_type='kpi'
- `omcgo/internal/pm/kpi/engine.go` → counter 聚合按 statis_type 路由（G5 阶段才用得上，本阶段先打通字段消费）
- `omcgo/internal/pm/kpi/calculator.go` → 工具函数 `AggregateByStatisType(values []float64, stype string) (float64, error)`
- `omcgo/internal/pm/query/handler.go` → 删物化视图相关查询路径，全部走 pm_metrics + metric_type 过滤

**删除**：
- `omcgo/internal/pm/counter/` 包合并到 `omcgo/internal/pm/metrics/`（或保留包名仅改 repository 实现，避免大改包路径影响 import）
- `omcgo/internal/pm/kpi/values_*.go` 中独立写 `kpi_values` 的逻辑

**预留**（G4 三字段已落，G3 收紧 NOT NULL）：
- pm_metrics 三字段 NOT NULL（直接在新表 schema 定）

## 2. Tasks

### Task 1: pg_dump 备份旧表 + migration up half（DROP + CREATE）

**Files:**
- Create: `omcgo/migrations/000166_create_pm_metrics_and_drop_legacy.sql`

**预先**（在 docker 容器内手工跑一次）：
```bash
docker exec omc-docker-postgres-1 pg_dump -U omcgo -d omcgo \
  --table=pm_counters --table=kpi_values --table=pm_counters_hourly \
  > /tmp/pm-kpi-pre-G3-backup-20260522.sql
ls -la /tmp/pm-kpi-pre-G3-backup-20260522.sql
```

migration：
```sql
-- +goose Up
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS pm_counters_hourly CASCADE;
DROP TABLE IF EXISTS pm_counters CASCADE;
DROP TABLE IF EXISTS kpi_values CASCADE;

CREATE TABLE pm_metrics (
  id            UUID NOT NULL DEFAULT gen_random_uuid(),
  device_sn     TEXT NOT NULL,
  metric_path   TEXT NOT NULL,
  metric_type   TEXT NOT NULL CHECK (metric_type IN ('counter','kpi')),
  metric_value  DOUBLE PRECISION NOT NULL,
  statis_type   TEXT CHECK (statis_type IS NULL OR statis_type IN ('sum','avg','max','pct')),
  granularity   TEXT NOT NULL CHECK (granularity IN ('15min','hourly','daily','weekly','monthly')),
  time          TIMESTAMPTZ NOT NULL,
  start_time    TIMESTAMPTZ NOT NULL,
  end_time      TIMESTAMPTZ NOT NULL,
  ingest_time   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  object_ldn    TEXT,
  extra         JSONB,
  PRIMARY KEY (id, time)  -- TS hypertable 要求 PK 含分区列
);

CREATE UNIQUE INDEX uq_pm_metrics_natural
  ON pm_metrics (device_sn, metric_path, granularity, end_time);

CREATE INDEX idx_pm_metrics_path_time ON pm_metrics (metric_path, time DESC);
CREATE INDEX idx_pm_metrics_device_time ON pm_metrics (device_sn, time DESC);
CREATE INDEX idx_pm_metrics_ingest_time ON pm_metrics (ingest_time DESC);
CREATE INDEX idx_pm_metrics_object_ldn ON pm_metrics (object_ldn) WHERE object_ldn IS NOT NULL;

-- hypertable
SELECT create_hypertable('pm_metrics', 'time', chunk_time_interval => INTERVAL '1 day');

-- compression（7d 后压缩，与 G2 默认 raw_15min_days=30 配套）
ALTER TABLE pm_metrics SET (
  timescaledb.compress,
  timescaledb.compress_segmentby = 'device_sn,metric_type,granularity'
);
SELECT add_compression_policy('pm_metrics', INTERVAL '7 days');

-- retention（30d 后删；G2 配置变化时 alter_job 联动）
SELECT add_retention_policy('pm_metrics', INTERVAL '30 days');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT remove_retention_policy('pm_metrics', if_exists => true);
SELECT remove_compression_policy('pm_metrics', if_exists => true);
DROP TABLE IF EXISTS pm_metrics CASCADE;

-- 兜底（项目未上生产无数据，down 仅恢复空 schema）
CREATE TABLE IF NOT EXISTS pm_counters (
  -- ... 旧 schema 占位（具体见 /tmp/pm-kpi-pre-G3-backup-20260522.sql）
);
CREATE TABLE IF NOT EXISTS kpi_values (
  -- ... 旧 schema 占位
);
-- +goose StatementEnd
```

**Down 兜底说明**：旧 schema 用 backup SQL 中的 CREATE TABLE 复制（不写在 migration 里以免太长，而是 Down 时若需还原从备份 restore）。

- [ ] migration up → 验证 pm_metrics 表 + 5 索引 + hypertable + policy（`SELECT * FROM timescaledb_information.hypertables` / `policies`）；`pm_counters` / `kpi_values` / `pm_counters_hourly` 不存在。

### Task 2: model.go 重构

**Files:**
- Modify / Create: `omcgo/internal/pm/metrics/model.go`（新统一 model）
- Modify: `omcgo/internal/pm/counter/model.go` + `omcgo/internal/pm/kpi/model.go` → 改为薄包装或 deprecated

```go
package metrics

type MetricType string

const (
    MetricTypeCounter MetricType = "counter"
    MetricTypeKPI     MetricType = "kpi"
)

type StatisType string

const (
    StatisSum StatisType = "sum"
    StatisAvg StatisType = "avg"
    StatisMax StatisType = "max"
    StatisPct StatisType = "pct"
)

type Granularity string

const (
    Granularity15Min   Granularity = "15min"
    GranularityHourly  Granularity = "hourly"
    GranularityDaily   Granularity = "daily"
    GranularityWeekly  Granularity = "weekly"
    GranularityMonthly Granularity = "monthly"
)

type PMMetric struct {
    ID          uuid.UUID
    DeviceSN    string
    MetricPath  string
    MetricType  MetricType
    MetricValue float64
    StatisType  *StatisType   // nullable, kpi 时 nil
    Granularity Granularity
    Time        time.Time
    StartTime   time.Time
    EndTime     time.Time
    IngestTime  time.Time
    ObjectLDN   *string       // nullable
    Extra       map[string]any
}
```

- [ ] 写 model + 单测（类型校验 + JSON 序列化）→ 全过

### Task 3: Repository 重写

**Files:**
- Create: `omcgo/internal/pm/metrics/pg_repository.go`
- Test: `omcgo/internal/pm/metrics/pg_repository_test.go`

```go
type Repository interface {
    Insert(ctx context.Context, m PMMetric) error
    BatchInsert(ctx context.Context, ms []PMMetric) error
    Query(ctx context.Context, q QueryRequest) ([]PMMetric, error)
}

type QueryRequest struct {
    DeviceSNs      []string
    MetricPaths    []string
    MetricType     *MetricType
    Granularity    Granularity
    StartTime      time.Time
    EndTime        time.Time
    Limit          int
    Offset         int
}

// pg_repository.go 用 Squirrel：
sq.Insert("pm_metrics").
    Columns("id","device_sn","metric_path","metric_type","metric_value","statis_type","granularity","time","start_time","end_time","ingest_time","object_ldn","extra").
    Values(...).
    Suffix("ON CONFLICT (device_sn, metric_path, granularity, end_time) DO UPDATE SET metric_value=EXCLUDED.metric_value, ingest_time=NOW()").
    PlaceholderFormat(sq.Dollar)
```

**关键点**：ON CONFLICT 走 natural key 索引，幂等性保证补传不重复入库。

测试：BatchInsert 500 行 + UNIQUE 冲突走 UPDATE + Query 按粒度过滤。

- [ ] TDD 6 case → 全过

### Task 4: KPIEngine + Calculator 重构

**Files:**
- Modify: `omcgo/internal/pm/kpi/engine.go`
- Modify: `omcgo/internal/pm/kpi/calculator.go`

calculator.go 新增按 statis_type 聚合工具：
```go
func AggregateByStatisType(values []float64, stype StatisType) (float64, error) {
    if len(values) == 0 {
        return 0, errors.New("empty values")
    }
    switch stype {
    case StatisSum:
        var s float64
        for _, v := range values { s += v }
        return s, nil
    case StatisAvg:
        var s float64
        for _, v := range values { s += v }
        return s / float64(len(values)), nil
    case StatisMax:
        m := values[0]
        for _, v := range values[1:] { if v > m { m = v } }
        return m, nil
    case StatisPct:
        return 0, errors.New("pct aggregation requires arithmetic context, not raw values")
    default:
        return 0, fmt.Errorf("unknown statis_type: %s", stype)
    }
}
```

engine.go：实时计算流程不变，但 KPI 写入时把 metric_type='kpi' / counter 写入时 metric_type='counter'。**G3 阶段实时计算层不做 SQL 聚合**（聚合下沉到 G5 cron）；保留计算单 PM 文件内的 arithmetic 公式即可。

测试：
- TestAggregateByStatisType_AllFour（sum/avg/max/pct 各一个 case + invalid）
- TestEngine_WritesCounterAndKPIToSameTable（mock repository，断言 metric_type 正确）

- [ ] TDD → 全过

### Task 5: Query handler 删物化视图路径

**Files:**
- Modify: `omcgo/internal/pm/query/handler.go`（如不存在则跳过；T-0098 后可能由 product/handler 覆盖）

把 `SELECT * FROM pm_counters_hourly` 类 SQL 改为 `SELECT * FROM pm_metrics WHERE metric_type='counter' AND granularity='hourly'`（G5 后 hourly 表才存在；G3 阶段只支持 granularity='15min'）。

如果 `QueryAggregated` 函数依然存在做现场聚合，G5 阶段会改为查 G5 聚合表；G3 阶段先保留现场聚合走 pm_metrics 15min。

- [ ] grep 全仓 `pm_counters_hourly` → 0 命中后才提交

### Task 6: 集成 + commit

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/...
# 额外验证：用 SQL 注入历史数据 + 跑 KPIEngine 计算 + 查 pm_metrics
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
INSERT INTO pm_metrics (device_sn, metric_path, metric_type, metric_value, statis_type, granularity, time, start_time, end_time)
VALUES
  ('TEST-001', 'L.Cell.Avail.Dur', 'counter', 100, 'sum', '15min', NOW(), NOW()-INTERVAL '15min', NOW()),
  ('TEST-001', 'KPI.Avail.Rate',   'kpi',     99.5, NULL, '15min', NOW(), NOW()-INTERVAL '15min', NOW());
SELECT count(*), metric_type FROM pm_metrics GROUP BY metric_type;
"
```

commit message：
```
feat(pm): 实施 G3 合并 pm_counters+kpi_values 为 pm_metrics + 删物化视图

What: migration 000166 DROP MATERIALIZED VIEW pm_counters_hourly + DROP TABLE pm_counters + kpi_values（项目未上生产，已 pg_dump 备份到 /tmp/pm-kpi-pre-G3-backup-20260522.sql）+ CREATE pm_metrics（TS hypertable + metric_type counter/kpi + statis_type + granularity + G4 三时间字段 NOT NULL + 5 索引 + compression 7d + retention 30d）；新建 internal/pm/metrics 包统一 model + repository；KPIEngine + Calculator 重写按 statis_type 路由（sum/avg/max/pct）；删 pm_counters_hourly 查询路径。
Why: G3 设计文档 §4.3；T-0098 落库的 perf_indicators_*.statis_type 启用消费；为 G5 自然日历桶预聚合（hourly/daily/weekly/monthly）打地基；单一表 + metric_type 字段简化分层。
Impact: 旧 pm_counters / kpi_values / pm_counters_hourly 全部移除（本地无生产数据）；现场聚合接口暂走 pm_metrics 15min（G5 后路由到聚合表）；KPIEngine 实时计算简化（无 SQL 聚合）；查询接口契约保持（仍按 metric_path + time 范围查）。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P3
Review: <审查报告路径>
```

- [ ] go test 全过 → /commit skill

## 3. 验收

- [ ] migration up → pm_metrics 表 + 5 索引 + hypertable + compression + retention policy 全在
- [ ] migration down → pm_metrics 删除（兜底 CREATE 空表）
- [ ] migration up 再次 → 干净状态
- [ ] grep `pm_counters_hourly` 全仓 → 0 命中（除 down 兜底注释）
- [ ] grep `pm_counters` / `kpi_values` 全仓 → 0 命中（除 down 兜底注释 + backup 文件路径）
- [ ] KPIEngine 实时计算 unit test 覆盖 counter + KPI 两路 → metric_type 正确写入
- [ ] AggregateByStatisType 5 case 全过（sum/avg/max/pct + invalid）
- [ ] SQL 注入 mock 数据后查 pm_metrics → counter + kpi 都有

## 4. Out of scope

- hourly/daily/weekly/monthly 聚合表 → G5
- 实时计算从聚合表查询路由 → G5
- 历史数据迁移 → 无（项目未上生产）
- pct 在 G5 cron 聚合时的 arithmetic 上下文 → G5
- KPIEngine 路由按设备 → 产品 → 平台公式 → G1

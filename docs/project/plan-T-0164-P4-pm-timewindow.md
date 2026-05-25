# T-0164-P4 / G4 PM 时间窗 + 入库时间字段 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 在 PM 时序数据中显式存储采集时间窗（start_time / end_time，基站时钟）+ 入库时间（ingest_time，OMC 时钟），用于上报延迟监控、时钟漂移排查、补传识别、唯一性维度扩展。

**Architecture:** 阶段做法——先在当前 `pm_counters` 和 `kpi_values` 两表加 NULLABLE 三字段（G3 合表前不破坏现状），parser.go 解析 fileHeader/fileFooter 时间窗，collector handler 填 ingest_time。G3 合表为 pm_metrics 时将三字段收紧 NOT NULL。

**Tech Stack:** Go + pgx + 3GPP 32.435 XML parser（encoding/xml）+ pgxpool。

**Deps:** 无（独立；G3 实施时引用本任务已落库字段，收紧 NOT NULL）。

---

## 0. PM 文件结构（3GPP 32.435）回顾

```xml
<measCollecFile>
  <fileHeader>
    <measCollec beginTime="2026-05-22T10:00:00+08:00"/>
  </fileHeader>
  <measData>
    <measInfo>
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">L.Cell.Avail.Dur</measType>
      <measValue measObjLdn="...">
        <r p="1">874</r>
      </measValue>
    </measInfo>
  </measData>
  <fileFooter>
    <measCollec endTime="2026-05-22T10:15:00+08:00"/>
  </fileFooter>
</measCollecFile>
```

**三时间字段语义**：
| 字段 | 来源 | 含义 |
|------|------|------|
| `start_time` | `fileHeader/measCollec/@beginTime` | 基站采集窗口起点（基站时钟） |
| `end_time` | `fileFooter/measCollec/@endTime`（或回退用 `granPeriod/@endTime`）| 基站采集窗口止点（基站时钟） |
| `ingest_time` | `time.Now()` in collector | OMC 入库时刻（OMC 时钟） |

**校验**：
- `start_time < end_time`
- `end_time - start_time ∈ [granPeriod.duration ± 60s]`（容忍时钟漂移）
- `ingest_time - end_time < 24h`（拒绝过老文件，可配置）

## 1. 文件结构

**新建**：
- `omcgo/migrations/000165_add_pm_time_window_columns.sql` — 给 pm_counters + kpi_values 加 3 字段（NULLABLE）

**修改**：
- `omcgo/internal/pm/parser/parser.go` — 补读 fileHeader.measCollec.beginTime + fileFooter.measCollec.endTime
- `omcgo/internal/pm/parser/types.go` — PMFile 结构加 StartTime / EndTime 字段
- `omcgo/internal/pm/collector/handler.go` — 填 IngestTime + 校验 + 写入
- `omcgo/internal/pm/counter/pg_repository.go` + `omcgo/internal/pm/kpi/pg_repository.go` — INSERT 列加三字段

## 2. Tasks

### Task 1: migration 加三字段（NULLABLE）

**Files:**
- Create: `omcgo/migrations/000165_add_pm_time_window_columns.sql`

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE pm_counters
  ADD COLUMN start_time  TIMESTAMPTZ,
  ADD COLUMN end_time    TIMESTAMPTZ,
  ADD COLUMN ingest_time TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE kpi_values
  ADD COLUMN start_time  TIMESTAMPTZ,
  ADD COLUMN end_time    TIMESTAMPTZ,
  ADD COLUMN ingest_time TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 索引：按 ingest_time 排序便于排查上报延迟
CREATE INDEX IF NOT EXISTS idx_pm_counters_ingest_time ON pm_counters (ingest_time DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_values_ingest_time  ON kpi_values  (ingest_time DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pm_counters_ingest_time;
DROP INDEX IF EXISTS idx_kpi_values_ingest_time;
ALTER TABLE pm_counters DROP COLUMN IF EXISTS start_time, DROP COLUMN IF EXISTS end_time, DROP COLUMN IF EXISTS ingest_time;
ALTER TABLE kpi_values  DROP COLUMN IF EXISTS start_time, DROP COLUMN IF EXISTS end_time, DROP COLUMN IF EXISTS ingest_time;
-- +goose StatementEnd
```

**注意**：因 pm_counters 是 TimescaleDB hypertable，`ALTER TABLE ... ADD COLUMN` 是支持的；如果 PostgreSQL 报错"can't add NOT NULL column to hypertable without DEFAULT"，则 ingest_time 改为 `NOT NULL DEFAULT NOW()` 已经满足。

- [ ] migration up → 查 `\d+ pm_counters` 三字段在；migration down → 字段消失；migration up → 再次入库。

### Task 2: parser.go 补读 fileHeader / fileFooter

**Files:**
- Modify: `omcgo/internal/pm/parser/parser.go`
- Modify: `omcgo/internal/pm/parser/types.go`
- Test: `omcgo/internal/pm/parser/parser_test.go`

types.go 新增：
```go
type PMFile struct {
    FileHeaderBeginTime time.Time   // fileHeader.measCollec.@beginTime
    FileFooterEndTime   time.Time   // fileFooter.measCollec.@endTime
    MeasData            []MeasInfo
    // ... 原字段
}
```

parser.go：增加 `<fileHeader>` 和 `<fileFooter>` 段的解析。当前代码（设计文档 §0 提到 `parser.go:110`）只读 `granPeriod.endTime`，需加：

```go
type measCollecFile struct {
    XMLName    xml.Name `xml:"measCollecFile"`
    FileHeader fileHeader `xml:"fileHeader"`
    MeasData   []measData `xml:"measData"`
    FileFooter fileFooter `xml:"fileFooter"`
}

type fileHeader struct {
    MeasCollec collecAttrs `xml:"measCollec"`
}
type fileFooter struct {
    MeasCollec collecAttrs `xml:"measCollec"`
}
type collecAttrs struct {
    BeginTime string `xml:"beginTime,attr"`
    EndTime   string `xml:"endTime,attr"`
}
```

解析后填到 PMFile.FileHeaderBeginTime / FileFooterEndTime。时区按 ISO 8601 with offset 解（`time.Parse(time.RFC3339, ...)`），缺失走回退（回退到 granPeriod.endTime 减 duration）。

测试用例：
1. 完整 fileHeader + fileFooter → 两字段命中
2. 缺 fileHeader → start_time 回退（用 granPeriod.endTime - duration）+ log warn
3. 缺 fileFooter → end_time 回退（用 granPeriod.endTime）+ log warn
4. beginTime 时区缺失（如 `2026-05-22T10:00:00`，无 offset）→ 按系统时区解析 + log warn

- [ ] TDD 4 case → 全过

### Task 3: collector handler 填 ingest_time + 校验

**Files:**
- Modify: `omcgo/internal/pm/collector/handler.go`
- Test: `omcgo/internal/pm/collector/handler_test.go`

handler 在写 pm_counters / kpi_values 前：
```go
ingestTime := time.Now()
if err := validateTimeWindow(pmFile.FileHeaderBeginTime, pmFile.FileFooterEndTime, pmFile.GranPeriod.Duration, ingestTime); err != nil {
    return fmt.Errorf("time window validation failed: %w", err)
}

// 把三字段填到 counter/kpi model
counter.StartTime = pmFile.FileHeaderBeginTime
counter.EndTime = pmFile.FileFooterEndTime
counter.IngestTime = ingestTime
```

`validateTimeWindow`：
```go
func validateTimeWindow(start, end time.Time, granDuration, ingest time.Time) error {
    if start.IsZero() || end.IsZero() {
        return errors.New("start or end time missing (parser fallback exhausted)")
    }
    if !start.Before(end) {
        return fmt.Errorf("start_time %v >= end_time %v", start, end)
    }
    span := end.Sub(start)
    expected := granDuration
    tolerance := 60 * time.Second
    if absDiff(span, expected) > tolerance {
        return fmt.Errorf("time window span %v deviates from granPeriod.duration %v by > 60s", span, expected)
    }
    if ingest.Sub(end) > 24*time.Hour {
        return fmt.Errorf("ingest_time - end_time > 24h, possibly stale file")
    }
    return nil
}
```

测试 5 case：合法 / start>=end / span 偏差 / 过老文件 / 缺字段。

- [ ] TDD 5 case → 全过

### Task 4: Repository INSERT 列加三字段

**Files:**
- Modify: `omcgo/internal/pm/counter/pg_repository.go`
- Modify: `omcgo/internal/pm/kpi/pg_repository.go`
- Test: 现有 repository test 扩展

把现有 INSERT 列表扩 3 列。Squirrel：
```go
sq.Insert("pm_counters").
    Columns("device_sn", "metric_path", "counter_value", "time", "start_time", "end_time", "ingest_time").
    Values(c.DeviceSN, c.MetricPath, c.CounterValue, c.Time, c.StartTime, c.EndTime, c.IngestTime)
```

- [ ] 扩展现有 repository test 覆盖三字段写入 + 读取

### Task 5: 集成 + commit

跑：
```bash
cd omcgo && go build ./... && go test ./internal/pm/parser/... ./internal/pm/collector/... ./internal/pm/counter/... ./internal/pm/kpi/...
```

commit message：
```
feat(pm): 实施 G4 PM 时间窗 + 入库时间字段（start_time/end_time/ingest_time）

What: migration 000165 给 pm_counters + kpi_values 加 3 NULLABLE 字段；parser.go 补读 fileHeader/measCollec/@beginTime + fileFooter/measCollec/@endTime；collector handler 填 ingest_time + 校验 start<end + 窗口长度 ≤ granPeriod.duration±60s + 过老文件拒收；repository INSERT 扩三字段写入。
Why: G4 设计文档 §4.4；支持上报延迟监控、时钟漂移排查、补传识别、唯一性维度扩展；G3 合表为 pm_metrics 时三字段将收紧 NOT NULL。
Impact: pm_counters / kpi_values 新增三列；老数据 NULL（项目未上生产无影响）；查询接口不破坏（现有索引仍用 time 列）。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P4
Review: <审查报告路径>
```

- [ ] go test 全过 → /commit skill

## 3. 验收

- [ ] migration up + down + up 三轮幂等
- [ ] `\d+ pm_counters` 含 start_time / end_time / ingest_time 三列
- [ ] parser 测试 4 case 全过（含 fallback 路径）
- [ ] collector 测试 5 case 全过（含异常路径）
- [ ] repository 集成测试三字段读写正常
- [ ] 单跑一个 sample PM 文件（用 `scripts/cpe_simulator.py` 或 SQL 注入 mock 文件）→ DB 查得三字段都填入合理值

## 4. Out of scope

- pm_metrics 唯一性约束（device_sn + metric_path + end_time + start_time + granularity）→ G3 合表时定
- measObjLdn 通用解析（LDN 结构）→ G3 一并处理
- 三字段 NOT NULL 收紧 → G3
- 真机 PM 文件端到端验证 → T-0121 阻塞（ACS 不下发 PM-CONFIG）

# DD-11: 性能管理与 KPI（F03）

> 关联功能域：F03（性能管理）
> 关联 backend-design.md 章节：第五章（TimescaleDB Schema）、第六章（PM 数据管线）
> 实施阶段：Phase 3（数据管线）
> 依赖文档：DD-02, DD-04, DD-05

---

## 1. 概述

### 1.1 模块定位

性能管理（`internal/pm/`）负责从基站周期性采集 PM 计数器数据、计算 KPI 指标、支撑网络质量评估。数据通过 TR069 Upload RPC 以 XML 文件上传，由 Worker 进程解析后存入 TimescaleDB。

### 1.2 核心职责

- PM 文件采集（MinIO 存储 + NATS 通知）
- XML 解析器（流式解析 PM 文件）
- 原始计数器存储（TimescaleDB pm_counters 超表）
- KPI 计算引擎（公式注册 + 依赖计数器解析）
- 时间维度聚合（15min → 1h → 24h）
- PM 数据导出（北向）

### 1.3 数据处理管线

```
ACS Upload TransferComplete
  → pm.file.received (NATS)
  → Worker: 从 MinIO 下载 PM XML 文件
  → XML 流式解析，提取计数器
  → 批量写入 TimescaleDB pm_counters
  → KPI 计算引擎
  → 写入 TimescaleDB kpi_values
  → 可选: oss.pm.export (北向推送)
```

---

## 2. 接口设计

### 2.1 PM Collector — `internal/pm/collector/collector.go`

```go
type PMCollector struct {
    minioClient *minio.Client
    parser      *PMXMLParser
    counterRepo CounterRepository
    eventBus    event.EventBus
    logger      *zap.Logger
}

// ProcessFile 处理单个 PM 文件
func (c *PMCollector) ProcessFile(ctx context.Context, filePath string) error

// HandleFileReceived 事件处理入口（pm.file.received）
func (c *PMCollector) HandleFileReceived(ctx context.Context, evt event.Event) error
```

### 2.2 PM XML 解析器 — `internal/pm/collector/parser_xml.go`

```go
type PMXMLParser struct {
    carrierRegistry *carrier.CarrierRegistry
}

type PMFileContent struct {
    DeviceSN    string
    CollectTime time.Time
    Granularity int // 分钟
    Counters    []PMCounterRecord
}

type PMCounterRecord struct {
    CellID       string
    CounterGroup string
    CounterName  string
    CounterValue float64
}

// Parse 流式解析 PM XML 文件
func (p *PMXMLParser) Parse(r io.Reader, carrier model.CarrierCode) (*PMFileContent, error)
```

### 2.3 Counter Repository — `internal/pm/counter/repository.go`

```go
type CounterRepository interface {
    BatchInsert(ctx context.Context, counters []model.PMCounter) error
    Query(ctx context.Context, filter CounterFilter) ([]model.PMCounter, error)
    QueryAggregated(ctx context.Context, filter CounterFilter, granularity string) ([]AggregatedCounter, error)
}

type CounterFilter struct {
    DeviceID     *uuid.UUID
    CellID       *string
    CounterGroup *string
    CounterName  *string
    StartTime    time.Time
    EndTime      time.Time
}
```

### 2.4 KPI 计算引擎 — `internal/pm/kpi/engine.go`

```go
type KPIEngine struct {
    formulas    map[string]*KPIFormula // key: kpi_name
    counterRepo CounterRepository
    kpiRepo     KPIRepository
    logger      *zap.Logger
}

type KPIFormula struct {
    Name        string
    DisplayName string
    Expression  string   // "counter_a / counter_b * 100"
    Counters    []string // 依赖的计数器
    Unit        string
    Category    string
    Carrier     *model.CarrierCode // nil 表示通用
    Technology  *model.Technology
}

// Calculate 计算指定时间范围的 KPI
func (e *KPIEngine) Calculate(ctx context.Context, deviceID uuid.UUID,
    startTime, endTime time.Time) ([]model.KPIValue, error)

// RegisterFormula 注册 KPI 公式
func (e *KPIEngine) RegisterFormula(formula *KPIFormula)
```

### 2.5 KPI 公式定义

**LTE 指标** — `internal/pm/kpi/formulas_lte.go`：

| KPI | 公式 | 类别 |
|-----|------|------|
| RRC 建立成功率 | rrc_setup_success / rrc_setup_attempt × 100 | 接入 |
| E-RAB 建立成功率 | erab_setup_success / erab_setup_attempt × 100 | 接入 |
| 掉话率 | abnormal_release / (normal_release + abnormal_release) × 100 | 保持 |
| 切换成功率 | handover_success / handover_attempt × 100 | 移动性 |
| PRB 利用率 | used_prb / total_prb × 100 | 流量 |
| 下行吞吐量 | dl_data_volume / measurement_period | 流量 |

**NR 指标** — `internal/pm/kpi/formulas_nr.go`：

| KPI | 公式 | 类别 |
|-----|------|------|
| 5G RRC 建立成功率 | nr_rrc_setup_success / nr_rrc_setup_attempt × 100 | 接入 |
| 5G 掉线率 | nr_abnormal_release / nr_total_release × 100 | 保持 |
| 5G 用户面时延 | nr_user_plane_latency_sum / nr_user_plane_latency_count | 时延 |
| 5G 下行平均速率 | nr_dl_throughput_sum / nr_active_ue_count | 速率 |

### 2.6 时间聚合 — `internal/pm/aggregation/`

```go
// TimescaleDB 连续聚合（自动维护）
// pm_counters_hourly: 15min → 1h
// pm_counters_daily: 1h → 24h (可选)
```

---

## 3. 数据模型

### 3.1 TimescaleDB Schema

```sql
-- PM 计数器超表
CREATE TABLE pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32),
    counter_group  VARCHAR(32),
    counter_name   VARCHAR(64),
    counter_value  DOUBLE PRECISION,
    granularity    SMALLINT DEFAULT 15
);
SELECT create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day');
SELECT add_compression_policy('pm_counters', INTERVAL '7 days');
SELECT add_retention_policy('pm_counters', INTERVAL '90 days');

-- KPI 值超表
CREATE TABLE kpi_values (
    time        TIMESTAMPTZ NOT NULL,
    device_id   UUID NOT NULL,
    cell_id     VARCHAR(32),
    kpi_name    VARCHAR(64),
    kpi_value   DOUBLE PRECISION,
    carrier     VARCHAR(4),
    technology  VARCHAR(3)
);
SELECT create_hypertable('kpi_values', 'time');

-- 连续聚合
CREATE MATERIALIZED VIEW pm_counters_hourly
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', time) AS bucket,
       device_id, counter_group, counter_name,
       SUM(counter_value) AS total,
       AVG(counter_value) AS avg_val
FROM pm_counters
GROUP BY bucket, device_id, counter_group, counter_name;
```

### 3.2 MinIO 存储路径

```
pm-files/{carrier}/{date}/{device_serial}/pm_{timestamp}.xml
```

### 3.3 REST API

```
GET  /api/v1/pm/counters          查询原始计数器
GET  /api/v1/pm/kpi               查询 KPI 值
GET  /api/v1/pm/kpi/definitions   KPI 定义列表
POST /api/v1/pm/export            触发 PM 数据导出
```

---

## 4. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| PM 文件格式 | XML (gNB PM V1.9.1) | XML | XML |
| LTE KPI 体系 | — | 2018 修订版 | — |
| NR KPI 体系 | — | SA V6 | V1.0 |
| 采集粒度 | 15min | 15min | 15min |

---

## 5. 实施子阶段

### 阶段 11a：PM Collector + XML 解析 + Counter 存储（Phase 3）
### 阶段 11b：KPI 计算引擎 + 公式（Phase 3）
### 阶段 11c：时间维度聚合（Phase 3）
### 阶段 11d：导出 + 调度（Phase 4）

---

## 6. 文件清单

```
internal/pm/collector/collector.go
internal/pm/collector/parser_xml.go
internal/pm/counter/repository.go
internal/pm/kpi/engine.go
internal/pm/kpi/formulas.go
internal/pm/kpi/formulas_lte.go
internal/pm/kpi/formulas_nr.go
internal/pm/kpi/repository.go
internal/pm/aggregation/aggregation.go
internal/pm/export/export.go
internal/pm/service.go
```

---

## 7. 参考

- backend-design.md 第五章：TimescaleDB Schema
- backend-design.md 第六章：PM 数据管线
- doc/features/03-performance-management.md：F03 全部子功能

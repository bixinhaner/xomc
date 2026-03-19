# 06 — 数据管线服务

> PM 采集、KPI 计算、告警管理、测量报告的全链路设计

---

## 1. 数据管线总览

```
                                  ┌──────────────────────┐
                                  │     ACS Engine        │
                                  │  (SOAP/XML 接收端)    │
                                  └──────┬───────────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
                    ▼                    ▼                    ▼
          pm.file.received     alarm.raised        mr.file.received
               (NATS)              (NATS)               (NATS)
                    │                    │                    │
                    ▼                    ▼                    ▼
            ┌───────────┐      ┌───────────┐       ┌───────────┐
            │  Worker   │      │ alarm-rpc │       │  Worker   │
            │ PM 解析器 │      │ 告警引擎  │       │ MR 解析器 │
            └─────┬─────┘      └─────┬─────┘       └─────┬─────┘
                  │                  │                    │
                  ▼                  ▼                    ▼
          ┌────────────┐    ┌────────────┐       ┌────────────┐
          │TimescaleDB │    │ Redis +    │       │TimescaleDB │
          │pm_counters │    │ PostgreSQL │       │measurement │
          └─────┬──────┘    │ + Timescale│       │ _reports   │
                │           └────────────┘       └────────────┘
                ▼
          ┌────────────┐
          │  Worker    │
          │ KPI 计算   │
          └─────┬──────┘
                │
                ▼
          ┌────────────┐
          │TimescaleDB │
          │ kpi_values │
          └────────────┘
                │
                ▼
         oss.pm.export (可选)
```

---

## 2. 性能管理（F03）

### 2.1 PM 数据采集管线

```
Step 1: ACS 收到 Upload TransferComplete（PM 文件上传完成）
    → 发布 NATS 事件: pm.file.received
    → Payload: {device_serial, file_url (MinIO), file_type, granularity}

Step 2: Worker 消费事件
    → 从 MinIO 下载 PM XML 文件

Step 3: PMXMLParser 流式解析
    → xml.Decoder 逐节点解析
    → 提取: device_serial, cell_id, collect_time, counter_group, counter_name, counter_value

Step 4: 批量写入 TimescaleDB pm_counters
    → pgx/v5 CopyFrom 批量插入（高性能）

Step 5: KPI 计算引擎
    → 从 pm_counters 查询依赖的计数器
    → 执行公式计算
    → 写入 kpi_values

Step 6: 可选 — 北向推送
    → 发布 NATS 事件: oss.pm.export
```

### 2.2 PM XML 文件格式示例

```xml
<?xml version="1.0" encoding="UTF-8"?>
<PMReport>
  <DeviceSerial>SN123456</DeviceSerial>
  <CollectTime>2026-03-05T10:00:00Z</CollectTime>
  <Granularity>15</Granularity>
  <MeasurementData>
    <Cell id="Cell_1">
      <CounterGroup name="RRC">
        <Counter name="RRC.ConnEstabAtt">1250</Counter>
        <Counter name="RRC.ConnEstabSucc">1200</Counter>
        <Counter name="RRC.ConnEstabFail">50</Counter>
      </CounterGroup>
      <CounterGroup name="ERAB">
        <Counter name="ERAB.EstabAtt">1100</Counter>
        <Counter name="ERAB.EstabSucc">1050</Counter>
      </CounterGroup>
    </Cell>
  </MeasurementData>
</PMReport>
```

### 2.3 pm-rpc 服务定义

```protobuf
// api/proto/pm.proto
syntax = "proto3";

package pm;
option go_package = "omcgo/service/monitor/rpc/pm/pb";

service PMService {
    // PM 计数器查询
    rpc QueryCounters(CounterQueryReq) returns (CounterQueryResp);

    // KPI 查询
    rpc QueryKPI(KPIQueryReq) returns (KPIQueryResp);

    // KPI 定义管理
    rpc GetKPIDefinitions(KPIDefReq) returns (KPIDefResp);

    // 触发 PM 数据导出（北向）
    rpc TriggerExport(TriggerExportReq) returns (TriggerExportResp);

    // 测量报告查询
    rpc QueryMeasurementReports(MRQueryReq) returns (MRQueryResp);
}

message CounterQueryReq {
    string device_serial = 1;
    string cell_id = 2;
    string counter_group = 3;
    repeated string counter_names = 4;    // 空=全部
    int64 start_time = 5;                 // Unix timestamp
    int64 end_time = 6;
    string granularity = 7;               // 15min / 1hour / 1day
    int32 page = 8;
    int32 page_size = 9;
}

message CounterQueryResp {
    repeated CounterRecord records = 1;
    int64 total = 2;
}

message CounterRecord {
    int64 timestamp = 1;
    string device_serial = 2;
    string cell_id = 3;
    string counter_group = 4;
    string counter_name = 5;
    double counter_value = 6;
    int32 granularity = 7;
}

message KPIQueryReq {
    string device_serial = 1;
    string cell_id = 2;
    repeated string kpi_names = 3;
    int64 start_time = 4;
    int64 end_time = 5;
    string granularity = 6;
    int32 page = 7;
    int32 page_size = 8;
}

message KPIQueryResp {
    repeated KPIRecord records = 1;
    int64 total = 2;
}

message KPIRecord {
    int64 timestamp = 1;
    string device_serial = 2;
    string cell_id = 3;
    string kpi_name = 4;
    string display_name = 5;
    double value = 6;
    string unit = 7;
}

message KPIDefReq {
    string carrier = 1;
    string technology = 2;
}

message KPIDefResp {
    repeated KPIDefinition definitions = 1;
}

message KPIDefinition {
    string name = 1;
    string display_name = 2;
    string expression = 3;            // 计算公式
    string unit = 4;                  // 百分比 / Mbps / 次
    string category = 5;             // 接入 / 切换 / 吞吐量 / 容量
    string carrier = 6;
    string technology = 7;
    repeated string dependencies = 8; // 依赖的计数器名
}

message TriggerExportReq {
    string device_serial = 1;         // 空=全部
    int64 start_time = 2;
    int64 end_time = 3;
    string format = 4;                // csv / json
}

message TriggerExportResp {
    string task_id = 1;
    string download_url = 2;          // MinIO 下载链接（异步完成后可用）
}

message MRQueryReq {
    string device_serial = 1;
    string mr_type = 2;               // mro / mrs / mre
    int64 start_time = 3;
    int64 end_time = 4;
    int32 page = 5;
    int32 page_size = 6;
}

message MRQueryResp {
    repeated MRRecord records = 1;
    int64 total = 2;
}

message MRRecord {
    int64 timestamp = 1;
    string device_serial = 2;
    string cell_id = 3;
    string mr_type = 4;
    bytes data = 5;                   // JSON 编码的测量数据
}
```

### 2.4 KPI 计算引擎

```go
// service/worker/kpi/engine.go

type KPIEngine struct {
    formulas map[string]KPIFormula     // name → formula
    pmRepo   PMRepository              // TimescaleDB 查询
    kpiRepo  KPIRepository             // TimescaleDB 写入
}

// KPI 公式注册
func (e *KPIEngine) RegisterFormulas() {
    // LTE KPIs
    e.Register(KPIFormula{
        Name:         "rrc_conn_estab_success_rate",
        DisplayName:  "RRC连接建立成功率",
        Expression:   "RRC.ConnEstabSucc / RRC.ConnEstabAtt * 100",
        Unit:         "%",
        Category:     "接入",
        Carrier:      "cmcc",
        Technology:   "lte",
        Dependencies: []string{"RRC.ConnEstabSucc", "RRC.ConnEstabAtt"},
    })

    e.Register(KPIFormula{
        Name:         "erab_estab_success_rate",
        DisplayName:  "E-RAB建立成功率",
        Expression:   "ERAB.EstabSucc / ERAB.EstabAtt * 100",
        Unit:         "%",
        Category:     "接入",
        Carrier:      "cmcc",
        Technology:   "lte",
        Dependencies: []string{"ERAB.EstabSucc", "ERAB.EstabAtt"},
    })

    e.Register(KPIFormula{
        Name:         "dl_prb_utilization",
        DisplayName:  "下行PRB利用率",
        Expression:   "PRB.DLUsed / PRB.DLTotal * 100",
        Unit:         "%",
        Category:     "容量",
        Carrier:      "cmcc",
        Technology:   "lte",
        Dependencies: []string{"PRB.DLUsed", "PRB.DLTotal"},
    })

    // 5G NR KPIs
    e.Register(KPIFormula{
        Name:         "nr_rrc_success_rate",
        DisplayName:  "5G RRC连接建立成功率",
        Expression:   "NR.RRC.ConnEstabSucc / NR.RRC.ConnEstabAtt * 100",
        Unit:         "%",
        Category:     "接入",
        Carrier:      "cmcc",
        Technology:   "nr",
        Dependencies: []string{"NR.RRC.ConnEstabSucc", "NR.RRC.ConnEstabAtt"},
    })
}

// 计算单个时间窗口的 KPI
func (e *KPIEngine) Calculate(ctx context.Context, deviceSerial, cellID string,
    startTime, endTime time.Time) error {

    for _, formula := range e.formulas {
        // 1. 查询依赖的计数器值
        counters, err := e.pmRepo.GetCounters(ctx, deviceSerial, cellID,
            formula.Dependencies, startTime, endTime)
        if err != nil {
            continue // 数据不完整则跳过
        }

        // 2. 执行公式计算
        value, err := formula.Evaluate(counters)
        if err != nil {
            continue
        }

        // 3. 写入 KPI 值
        e.kpiRepo.Insert(ctx, &KPIValue{
            Time:         startTime,
            DeviceSerial: deviceSerial,
            CellID:       cellID,
            KPIName:      formula.Name,
            Value:        value,
            Granularity:  15, // 分钟
        })
    }
    return nil
}
```

---

## 3. 告警管理（F04）

### 3.1 告警处理管线

```
Step 1: ACS 收到 Inform (EventCode = ALARM)
    → 发布 NATS 事件: alarm.raised
    → Payload: {device_serial, alarm_type, alarm_code, severity, additional_info}

Step 2: alarm-rpc 消费事件（或 Worker 转发）
    → AlarmEngine 处理

Step 3: AlarmEngine 三阶段处理
    → 去重: 检查 alarms_active 是否已存在 (device + alarm_code)
    → 关联: 同设备的关联告警归组
    → 抑制: 高优先级告警存在时抑制低优先级同类告警

Step 4: 持久化
    → alarms_active (PostgreSQL): 当前活跃告警
    → Redis alarms_active:{device_serial}: 快速查询
    → alarms_history (TimescaleDB): 归档（含已清除告警）

Step 5: 可选 — 北向转发
    → 发布 NATS 事件: oss.alarm.forward
```

### 3.2 alarm-rpc 服务定义

```protobuf
// api/proto/alarm.proto
syntax = "proto3";

package alarm;
option go_package = "omcgo/service/monitor/rpc/alarm/pb";

service AlarmService {
    // 处理新告警（通常由 Worker/ACS 调用）
    rpc ProcessAlarm(ProcessAlarmReq) returns (ProcessAlarmResp);

    // 查询活跃告警
    rpc ListActiveAlarms(AlarmQueryReq) returns (AlarmListResp);

    // 查询历史告警
    rpc ListHistoryAlarms(AlarmQueryReq) returns (AlarmListResp);

    // 确认告警
    rpc AcknowledgeAlarm(AcknowledgeReq) returns (AcknowledgeResp);

    // 清除告警
    rpc ClearAlarm(ClearAlarmReq) returns (ClearAlarmResp);

    // 告警统计
    rpc GetAlarmStatistics(AlarmStatsReq) returns (AlarmStatsResp);
}

message ProcessAlarmReq {
    string device_serial = 1;
    int32 severity = 2;              // 1=Critical, 2=Major, 3=Minor, 4=Warning
    string alarm_type = 3;
    string alarm_code = 4;
    string description = 5;
    bytes additional_info = 6;       // JSON
    bool is_clear = 7;              // true = 清除事件
}

message ProcessAlarmResp {
    int64 alarm_id = 1;
    bool is_duplicate = 2;          // 是否重复告警
    bool is_suppressed = 3;         // 是否被抑制
}

message AlarmQueryReq {
    string device_serial = 1;
    int32 severity = 2;             // 0=全部
    string alarm_type = 3;
    int64 start_time = 4;
    int64 end_time = 5;
    string carrier = 6;
    int32 page = 7;
    int32 page_size = 8;
}

message AlarmListResp {
    repeated AlarmRecord alarms = 1;
    int64 total = 2;
}

message AlarmRecord {
    int64 id = 1;
    string device_serial = 2;
    int32 severity = 3;
    string severity_name = 4;       // Critical / Major / Minor / Warning
    string alarm_type = 5;
    string alarm_code = 6;
    string description = 7;
    string status = 8;              // active / acknowledged / cleared
    int64 raised_at = 9;
    int64 acknowledged_at = 10;
    int64 cleared_at = 11;
    bytes additional_info = 12;
}

message AcknowledgeReq {
    int64 alarm_id = 1;
    string acknowledged_by = 2;     // 确认人
    string comment = 3;
}

message AcknowledgeResp {
    bool success = 1;
}

message ClearAlarmReq {
    int64 alarm_id = 1;
    string cleared_by = 2;
    string reason = 3;
}

message ClearAlarmResp {
    bool success = 1;
}

message AlarmStatsReq {
    string carrier = 1;
    string device_serial = 2;
    int64 start_time = 3;
    int64 end_time = 4;
}

message AlarmStatsResp {
    int32 total_active = 1;
    int32 critical_count = 2;
    int32 major_count = 3;
    int32 minor_count = 4;
    int32 warning_count = 5;
    int32 acknowledged_count = 6;
}
```

### 3.3 告警生命周期

```
  告警产生
     │
     ▼
  ┌────────┐
  │ active │ ← 新告警进入活跃状态
  └──┬──┬──┘
     │  │
     │  │ 管理员确认
     │  ▼
     │  ┌──────────────┐
     │  │ acknowledged │ ← 已确认，仍在活跃列表
     │  └──────┬───────┘
     │         │
     ├─────────┤
     │         │ 收到清除事件 / 管理员手动清除
     │         ▼
     │  ┌─────────┐
     └──│ cleared │ ← 移出活跃列表，归入历史
        └─────────┘
```

### 3.4 告警去重逻辑

```go
func (e *AlarmEngine) Deduplicate(ctx context.Context, alarm *AlarmEvent) (bool, error) {
    // 检查同设备同告警码是否已有活跃告警
    existing, err := e.activeRepo.FindByDeviceAndCode(ctx,
        alarm.DeviceSerial, alarm.AlarmCode)
    if err != nil {
        return false, err
    }

    if existing != nil {
        // 更新最后出现时间，不创建新记录
        existing.LastOccurredAt = time.Now()
        existing.OccurrenceCount++
        e.activeRepo.Update(ctx, existing)
        return true, nil // 重复
    }
    return false, nil
}
```

---

## 4. 测量报告（F05）

### 4.1 MR 类型

| 类型 | 全称 | 说明 |
|------|------|------|
| MRO | Measurement Report for Optimization | 同频测量，用于覆盖优化 |
| MRS | Measurement Report for Statistics | 异频测量，用于负荷统计 |
| MRE | Measurement Report for Events | 异系统测量，用于互操作分析 |

### 4.2 MR 处理管线

与 PM 类似：

```
ACS Upload TransferComplete
  → NATS: mr.file.received
  → Worker 从 MinIO 下载 MR 文件
  → MRParser 解析 (XML 格式)
  → 批量写入 TimescaleDB measurement_reports
```

---

## 5. Worker 服务设计

### 5.1 Worker 架构

```go
// service/worker/main.go

func main() {
    var c config.Config
    conf.MustLoad("etc/worker.yaml", &c)

    svcCtx := svc.NewServiceContext(c)

    // 使用 go-zero ServiceGroup 管理多个工作协程
    group := service.NewServiceGroup()

    // PM 文件消费者
    group.Add(pm.NewConsumer(svcCtx))

    // MR 文件消费者
    group.Add(mr.NewConsumer(svcCtx))

    // KPI 计算定时器
    group.Add(kpi.NewScheduler(svcCtx))

    // 设备离线检测定时器
    group.Add(heartbeat.NewChecker(svcCtx))

    // 时间聚合定时器
    group.Add(aggregation.NewScheduler(svcCtx))

    group.Start()
}
```

### 5.2 NATS 消费者模板

```go
// service/worker/pm/consumer.go

type PMConsumer struct {
    svcCtx *svc.ServiceContext
    parser *PMXMLParser
    repo   PMRepository
    kpi    *kpi.Engine
}

func (c *PMConsumer) Start() {
    // NATS Queue Subscribe（多 Worker 实例负载均衡）
    c.svcCtx.NATS.QueueSubscribe("pm.file.received", "pm-workers",
        func(msg *nats.Msg) {
            var event PMFileEvent
            json.Unmarshal(msg.Data, &event)

            if err := c.process(context.Background(), &event); err != nil {
                log.Errorf("PM processing failed: %v", err)
                msg.Nak() // 重试
                return
            }
            msg.Ack()
        })
}

func (c *PMConsumer) process(ctx context.Context, event *PMFileEvent) error {
    // 1. 从 MinIO 下载文件
    reader, err := c.svcCtx.MinIO.GetObject(ctx, "pm-files", event.FileKey, minio.GetObjectOptions{})
    if err != nil {
        return fmt.Errorf("download PM file: %w", err)
    }
    defer reader.Close()

    // 2. 流式解析 XML
    counters, err := c.parser.Parse(reader)
    if err != nil {
        return fmt.Errorf("parse PM XML: %w", err)
    }

    // 3. 批量写入 TimescaleDB
    if err := c.repo.BatchInsert(ctx, counters); err != nil {
        return fmt.Errorf("insert counters: %w", err)
    }

    // 4. 触发 KPI 计算
    return c.kpi.Calculate(ctx, event.DeviceSerial, event.CollectTime)
}

func (c *PMConsumer) Stop() {
    // 优雅关闭
}
```

### 5.3 Worker Pool 并发控制

```go
type WorkerPool struct {
    name     string
    workers  int
    taskCh   chan Task
    wg       sync.WaitGroup
}

func NewWorkerPool(name string, workers int) *WorkerPool {
    p := &WorkerPool{
        name:    name,
        workers: workers,
        taskCh:  make(chan Task, workers*2),
    }
    for i := 0; i < workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
    return p
}

// 配置建议:
// PM 解析: 4 workers（CPU 密集型 XML 解析）
// MR 解析: 2 workers
// KPI 计算: 2 workers
```

---

## 6. TimescaleDB Schema

### 6.1 pm_counters

```sql
CREATE TABLE pm_counters (
    time            TIMESTAMPTZ  NOT NULL,
    device_id       BIGINT       NOT NULL,
    device_serial   VARCHAR(64)  NOT NULL,
    cell_id         VARCHAR(64)  NOT NULL,
    counter_group   VARCHAR(100) NOT NULL,
    counter_name    VARCHAR(200) NOT NULL,
    counter_value   DOUBLE PRECISION NOT NULL,
    granularity     INT          NOT NULL DEFAULT 15  -- 分钟
);

SELECT create_hypertable('pm_counters', 'time');

-- 压缩策略: 7 天后压缩
SELECT add_compression_policy('pm_counters', INTERVAL '7 days');

-- 保留策略: 90 天
SELECT add_retention_policy('pm_counters', INTERVAL '90 days');

CREATE INDEX idx_pm_device_time ON pm_counters (device_serial, time DESC);
CREATE INDEX idx_pm_counter ON pm_counters (counter_group, counter_name, time DESC);
```

### 6.2 kpi_values

```sql
CREATE TABLE kpi_values (
    time            TIMESTAMPTZ  NOT NULL,
    device_id       BIGINT       NOT NULL,
    device_serial   VARCHAR(64)  NOT NULL,
    cell_id         VARCHAR(64)  NOT NULL,
    kpi_name        VARCHAR(200) NOT NULL,
    kpi_value       DOUBLE PRECISION NOT NULL,
    granularity     INT          NOT NULL DEFAULT 15
);

SELECT create_hypertable('kpi_values', 'time');
SELECT add_compression_policy('kpi_values', INTERVAL '7 days');
SELECT add_retention_policy('kpi_values', INTERVAL '180 days');

CREATE INDEX idx_kpi_device_time ON kpi_values (device_serial, time DESC);
CREATE INDEX idx_kpi_name_time ON kpi_values (kpi_name, time DESC);
```

### 6.3 alarms_history

```sql
CREATE TABLE alarms_history (
    time              TIMESTAMPTZ  NOT NULL,  -- raised_at
    alarm_id          BIGINT       NOT NULL,
    device_id         BIGINT       NOT NULL,
    device_serial     VARCHAR(64)  NOT NULL,
    severity          SMALLINT     NOT NULL,
    alarm_type        VARCHAR(100) NOT NULL,
    alarm_code        VARCHAR(100) NOT NULL,
    description       TEXT,
    status            VARCHAR(20)  NOT NULL,
    acknowledged_by   VARCHAR(100),
    acknowledged_at   TIMESTAMPTZ,
    cleared_at        TIMESTAMPTZ,
    additional_info   JSONB
);

SELECT create_hypertable('alarms_history', 'time');
SELECT add_compression_policy('alarms_history', INTERVAL '30 days');
SELECT add_retention_policy('alarms_history', INTERVAL '365 days');

CREATE INDEX idx_ah_device_time ON alarms_history (device_serial, time DESC);
CREATE INDEX idx_ah_severity ON alarms_history (severity, time DESC);
```

### 6.4 时间聚合（连续聚合）

```sql
-- 小时级聚合
CREATE MATERIALIZED VIEW pm_counters_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    device_serial,
    cell_id,
    counter_group,
    counter_name,
    SUM(counter_value) AS sum_value,
    AVG(counter_value) AS avg_value,
    MAX(counter_value) AS max_value,
    MIN(counter_value) AS min_value,
    COUNT(*) AS sample_count
FROM pm_counters
GROUP BY bucket, device_serial, cell_id, counter_group, counter_name;

-- 日级聚合
CREATE MATERIALIZED VIEW pm_counters_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS bucket,
    device_serial,
    cell_id,
    counter_group,
    counter_name,
    SUM(counter_value) AS sum_value,
    AVG(counter_value) AS avg_value,
    MAX(counter_value) AS max_value,
    MIN(counter_value) AS min_value,
    COUNT(*) AS sample_count
FROM pm_counters
GROUP BY bucket, device_serial, cell_id, counter_group, counter_name;

-- 自动刷新策略
SELECT add_continuous_aggregate_policy('pm_counters_hourly',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('pm_counters_daily',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');
```

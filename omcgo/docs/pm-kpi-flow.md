# PM/KPI 文件上传与处理完整流程分析

> 本文档分析性能管理（PM）文件从设备上传到 KPI 计算入库的完整链路，涵盖文件采集、传输、解析、计数器存储、KPI 公式计算及数据聚合。

---

## 一、总体架构

PM/KPI 数据管线是一个**事件驱动的异步处理管线**，跨越 ACS、Worker 两个部署单元，通过 EventBus（NATS JetStream）串联各处理阶段。

```
┌──────────┐  Upload RPC   ┌──────────┐  EventBus    ┌──────────────────┐
│   CPE    │ ─────────────→ │   ACS    │ ──────────→  │     Worker       │
│ (基站)   │  PM XML 文件    │  Engine  │              │                  │
└──────────┘               └──────────┘              │  TransferBridge  │
     ↑                          │                     │       ↓          │
     │                     ATC 事件发布                │  PMCollector     │
     │                                                │       ↓          │
     │  Connection Request                            │  KPIEngine       │
     └──────────────────────────────────────────────  │       ↓          │
                                                      │  TimescaleDB     │
                                                      └──────────────────┘
```

### 涉及的核心组件

| 组件 | 文件 | ���责 |
|------|------|------|
| Upload RPC | `internal/acs/rpc/dispatcher.go` | 构建 SOAP Upload 请求发给设备 |
| ACS Handler | `internal/acs/handler.go` | 接收 AutonomousTransferComplete |
| TransferBridge | `internal/transfer/bridge.go` | 从设备下载文件��存入 MinIO |
| PMCollector | `internal/pm/collector/collector.go` | 事件订阅，驱动文件处理 |
| PMXMLParser | `internal/pm/collector/parser.go` | 3GPP 32.435 XML 流式解析 |
| CounterRepository | `internal/pm/counter/pg_repository.go` | 计数器批量写入 TimescaleDB |
| KPIEngine | `internal/pm/kpi/engine.go` | 公式加载、KPI 计算 |
| Formula | `internal/pm/kpi/formula.go` | 递归下降公式解析器 |
| KPIRepository | `internal/pm/kpi/pg_repository.go` | KPI 结果写入 TimescaleDB |
| Aggregator | `internal/pm/aggregation/aggregator.go` | 小时级连续聚合刷新 |
| PM Handler | `internal/pm/handler.go` | REST API 查询接口 |

---

## 二、完整事件链路

```
事件流（从左到右）:

CPE 上传文件
    ↓ SOAP AutonomousTransferComplete
ACS handler.handleAutonomousTransferComplete()
    ↓ 发布 "device.inform.autonomous_transfer_complete"
TransferBridge.handleAutonomousTransferComplete()
    ↓ HTTP GET 下载 → MinIO 存储
    ↓ 发布 "pm.file.received"
PMCollector.handleFileReceived()
    ↓ MinIO 下载 → XML 解析 → pm_counters 入库
    ↓ KPIEngine.CalculateAndStore() → kpi_values 入库
    ↓ 发布 "pm.file.parsed"
（完成）
```

---

## 三、阶段一：PM 文件采集触发

### 3.1 Upload 命令下发

管理面通过命令队列触发设备上传 PM 文件。Upload RPC 的 SOAP 模板：

```xml
<cwmp:Upload>
  <CommandKey>{{.CommandKey}}</CommandKey>
  <FileType>{{.FileType}}</FileType>      <!-- "4"=PM文件, "5"=MR文件 -->
  <URL>{{.URL}}</URL>                      <!-- ACS 文件接收地址 -->
  <Username>{{.Username}}</Username>
  <Password>{{.Password}}</Password>
  <DelaySeconds>{{.DelaySeconds}}</DelaySeconds>
</cwmp:Upload>
```

**FileType 编码**：

| 值 | 类型 |
|----|------|
| `1` | Firmware Image |
| `2` | Web Content |
| `3` | Vendor Configuration / Log |
| `4` | **PM 文件** |
| `5` | MR 文件 |

### 3.2 设备自主上传（AutonomousTransferComplete）

设备也可根据自身配置周期性上传 PM 文件，无需 ACS 主动触发。上传完成后设备发送 `AutonomousTransferComplete` SOAP 消息：

```go
// internal/acs/handler.go — handleAutonomousTransferComplete()
// 1. 解码 SOAP 报文
atc, cwmpID, err := soap.DecodeAutonomousTransferComplete(reader)

// 2. 构建事件负载
payload := map[string]interface{}{
    "device_sn":       deviceSN,
    "transfer_url":    atc.TransferURL,      // 设备侧文件下载地址
    "file_type":       atc.FileType,         // "4"=PM, "5"=MR
    "file_size":       atc.FileSize,
    "target_filename": atc.TargetFileName,   // 如 "A00.4g01.pm"
    "start_time":      atc.StartTime,
    "complete_time":   atc.CompleteTime,
    "fault":           atc.FaultStruct,      // nil 表示成功
}

// 3. 发布事件
h.eventBus.Publish(ctx, "device.inform.autonomous_transfer_complete", evt)

// 4. 返回 AutonomousTransferCompleteResponse
```

---

## 四、阶段二：文件下载与分类（TransferBridge）

**文件**：`internal/transfer/bridge.go`

**订阅**：`device.inform.autonomous_transfer_complete`（Queue Group: `transfer-bridge`）

### 4.1 处理流程

```
handleAutonomousTransferComplete(ctx, event)
    ├─ 1. 解码事件负载
    ├─ 2. 校验：跳过有 fault 或无 transfer_url 的事件
    ├─ 3. 查询设备信息（DeviceRepository）→ 获取 carrier、technology
    ├─ 4. 文件分类（见下表）
    ├─ 5. HTTP GET 下载文件（30 秒超时）
    ├─ 6. 流式写入 MinIO
    │     路径: {bucket}/{YYYY/MM/DD}/{device_sn}/{filename}
    └─ 7. 发布下游事件
```

### 4.2 文件分类规则

| 条件 | 分类 | MinIO Bucket | 下游事件 |
|------|------|-------------|---------|
| FileType="4" 或文件名含 PM/COUNTER | PM | `pm-files` | `pm.file.received` |
| FileType="5" 或文件名含 MRO/MRS/MRE | MR | `mr-files` | `mr.file.received` |
| FileType="3" 或文件名含 LOG | Log | `logs` | — |

### 4.3 PM 文件接收事件负载

```json
{
  "minio_path": "2026/03/17/DEVICE-SN-001/A00.4g01.pm.xml",
  "device_id":  "550e8400-e29b-41d4-a716-446655440000",
  "device_sn":  "DEVICE-SN-001",
  "carrier":    "cmcc",
  "technology": "lte"
}
```

---

## 五、阶段三：PM 文件解析（PMCollector）

**文件**：`internal/pm/collector/collector.go`

**订阅**：`pm.file.received`（Queue Group: `pm-workers`）

### 5.1 处理流程

```
handleFileReceived(ctx, event)
    ├─ 1. 解码事件负载 → 获取 minio_path, device_id 等
    ├─ 2. 从 MinIO 下载 PM 文件
    ├─ 3. 记录文件元数据到 pm_files 表（parsed=false）
    ├─ 4. XML 流式解析 → PMFileContent{Counters[]}
    ├─ 5. 批量写入 pm_counters 表（pgx.CopyFrom）
    ├─ 6. 更新 pm_files（parsed=true, counter_count）
    ├─ 7. 提取所有 unique cellID
    ├─ 8. 对每个 cellID 调用 KPIEngine.CalculateAndStore()
    └─ 9. 发布 "pm.file.parsed" 事件
```

### 5.2 XML 解析器（3GPP 32.435 格式）

**文件**：`internal/pm/collector/parser.go`

采用 `encoding/xml` 流式 Decoder，不将整个 XML 加载到内存。

**PM 文件结构示例**：

```xml
<measCollecFile>
  <fileHeader dnPrefix="DC=baicells"/>
  <measData>
    <managedElement localDn="MeContext=DEVICE-SN-001"/>
    <measInfo measInfoId="LTE.CellMeasReport">
      <granPeriod duration="PT900S" endTime="2026-03-17T10:00:00Z"/>
      <measType p="1">PRB.UlAvailProcMeas</measType>
      <measType p="2">PRB.DlAvailProcMeas</measType>
      <measType p="3">RRC.ConnEstabSucc</measType>
      <measValue measObjLdn="CellId=1234">
        <r p="1">85.5</r>
        <r p="2">92.3</r>
        <r p="3">1024</r>
      </measValue>
      <measValue measObjLdn="CellId=5678">
        <r p="1">78.2</r>
        <r p="2">88.1</r>
        <r p="3">956</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>
```

**解析逻辑**：

```
1. managedElement.localDn → 提取 deviceSN（MeContext=xxx）
2. measInfo.measInfoId    → counterGroup（如 "LTE.CellMeasReport"）
3. granPeriod.duration    → 解析 ISO 8601（"PT900S" = 900秒 = 15分钟）
4. granPeriod.endTime     → collectTime 时间戳
5. measType[p]           → 构建索引：p → counterName 的映射表
6. measValue.measObjLdn  → 提取 cellID：
   ├─ LTE: "CellId=1234" → cellID="1234"
   ├─ 5G:  "NRCellDU=xxx" 或 "NRCellCU=xxx"
7. r[p]                  → 根据索引查找 counterName + counterValue
```

**输出结构**：

```go
type PMFileContent struct {
    DeviceSN    string
    CollectTime time.Time
    Granularity int            // 分钟数（如 15）
    Counters    []model.PMCounter
}

type PMCounter struct {
    Time         time.Time     // collectTime
    DeviceID     uuid.UUID
    CellID       string        // "1234"
    CounterGroup string        // "LTE.CellMeasReport"
    CounterName  string        // "PRB.UlAvailProcMeas"
    CounterValue float64       // 85.5
    Granularity  int           // 15
}
```

---

## 六、阶段四：KPI 计算（KPIEngine）

**文件**：`internal/pm/kpi/engine.go`

### 6.1 公式加载

KPIEngine 启动时从所有注册的运营商适配器加载 KPI 公式定义：

```go
func (e *KPIEngine) loadFormulas() {
    for _, carrier := range e.carrierRegistry.All() {
        for _, tech := range carrier.SupportedTechnologies() {
            defs := carrier.KPIDefinitions(tech)
            for _, def := range defs {
                formula, err := ParseFormula(def.Formula)
                // 缓存为 RegisteredFormula
            }
        }
    }
}
```

### 6.2 公式解析器

**文件**：`internal/pm/kpi/formula.go`

递归下降解析器，支持：
- **运算符**：`+`、`-`、`*`、`/`（标准优先级：`*`/`/` > `+`/`-`）
- **操作数**：浮点数、计数器标识符（字母/数字/下划线）
- **括号**：显式优先级覆盖
- **错误处理**：除以零返回 error，缺失计数器返回 error

```go
formula, _ := ParseFormula("rrc_conn_setup_succ / rrc_conn_setup_att * 100")
value, err := formula.Evaluate(map[string]float64{
    "rrc_conn_setup_succ": 950,
    "rrc_conn_setup_att":  1000,
})
// value = 95.0
```

### 6.3 计算流程

```go
func (e *KPIEngine) CalculateAndStore(ctx, deviceID, cellID, collectTime, carrier, tech) {
    // 1. 确定时间窗口（15 分钟回溯）
    startTime := collectTime.Add(-15 * time.Minute)
    endTime := collectTime

    // 2. 筛选适用的公式（匹配 carrier + technology）
    formulas := e.applicableFormulas(carrier, tech)

    // 3. 收集所有公式依赖的计数器名称
    counterNames := collectDependencies(formulas)

    // 4. 查询计数器值（聚合时间窗口内的 SUM）
    counters := counterRepo.QueryForKPI(deviceID, cellID, counterNames, startTime, endTime)
    // SQL: SELECT counter_name, SUM(counter_value) FROM pm_counters
    //      WHERE device_id=? AND cell_id=? AND counter_name IN (?)
    //        AND time BETWEEN startTime AND endTime
    //      GROUP BY counter_name

    // 5. 注入合成计数器
    counters["period_seconds"] = endTime.Sub(startTime).Seconds()  // 900.0

    // 6. 逐公式计算
    for _, f := range formulas {
        value, err := f.Formula.Evaluate(counters)
        if err != nil {
            continue  // 跳过缺失计数器或除零的公式
        }
        results = append(results, KPIValue{
            Time: endTime, DeviceID: deviceID, CellID: cellID,
            KPIName: f.Name, KPIValue: value,
            Carrier: carrier, Technology: tech,
        })
    }

    // 7. 批量写入 kpi_values 表
    kpiRepo.BatchInsert(ctx, results)
}
```

### 6.4 CMCC 运营商 KPI 定义

**文件**：`internal/core/carrier/cmcc/kpi.go`

#### LTE KPI（10 项）

| KPI 名称 | 显示名 | 公式 | 单位 | 类别 |
|----------|--------|------|------|------|
| `lte_rrc_setup_success_rate` | RRC 连接建立成功率 | `rrc_conn_setup_succ / rrc_conn_setup_att * 100` | % | accessibility |
| `lte_erab_setup_success_rate` | E-RAB 建立成功率 | `erab_setup_succ / erab_setup_att * 100` | % | accessibility |
| `lte_s1_signaling_setup_success_rate` | S1 信令连接建立成功率 | `s1_sig_conn_setup_succ / s1_sig_conn_setup_att * 100` | % | accessibility |
| `lte_erab_drop_rate` | E-RAB 掉话率 | `erab_abnormal_release / (erab_abnormal_release + erab_normal_release) * 100` | % | retainability |
| `lte_intra_freq_ho_success_rate` | 同频切换成功率 | `intra_freq_ho_succ / intra_freq_ho_att * 100` | % | mobility |
| `lte_inter_freq_ho_success_rate` | 异频切换成功率 | `inter_freq_ho_succ / inter_freq_ho_att * 100` | % | mobility |
| `lte_dl_prb_utilization` | 下行 PRB 利用率 | `dl_prb_used_avg / dl_prb_total * 100` | % | utilization |
| `lte_ul_prb_utilization` | 上行 PRB 利用率 | `ul_prb_used_avg / ul_prb_total * 100` | % | utilization |
| `lte_dl_throughput` | 下行吞吐量 | `pdcp_sdu_dl_volume * 8 / period_seconds / 1000000` | Mbps | throughput |
| `lte_ul_throughput` | 上行吞吐量 | `pdcp_sdu_ul_volume * 8 / period_seconds / 1000000` | Mbps | throughput |

#### 5G NR KPI（6 项）

| KPI 名称 | 显示名 | 公式 | 单位 | 类别 |
|----------|--------|------|------|------|
| `nr_rrc_setup_success_rate` | 5G RRC 连接建立成功率 | `nr_rrc_conn_setup_succ / nr_rrc_conn_setup_att * 100` | % | accessibility |
| `nr_ng_signaling_setup_success_rate` | NG 信令连接建立成功率 | `ng_sig_conn_setup_succ / ng_sig_conn_setup_att * 100` | % | accessibility |
| `nr_session_drop_rate` | 5G 会话掉线率 | `nr_session_abnormal_release / (nr_session_abnormal_release + nr_session_normal_release) * 100` | % | retainability |
| `nr_dl_user_throughput` | 5G 下行用户吞吐量 | `nr_pdcp_sdu_dl_volume * 8 / nr_active_ue_dl_time / 1000000` | Mbps | throughput |
| `nr_ul_user_throughput` | 5G 上行用户吞吐量 | `nr_pdcp_sdu_ul_volume * 8 / nr_active_ue_ul_time / 1000000` | Mbps | throughput |
| `nr_dl_average_latency` | 5G 下行平均时延 | `nr_dl_total_delay / nr_dl_total_packets` | ms | latency |

### 6.5 KPI 类别说明

| 类别 | 含义 | 典型 KPI |
|------|------|---------|
| `accessibility` | 接入性 | RRC 建立成功率、E-RAB 建立成功率 |
| `retainability` | 保持性 | 掉话率、会话掉线率 |
| `mobility` | 移动性 | 切换成功率 |
| `utilization` | 利用率 | PRB 利用率 |
| `throughput` | 吞吐量 | 上下行速率 |
| `latency` | 时延 | 下行平均时延 |

---

## 七、数据存储

### 7.1 数据库表结构

#### pm_counters（TimescaleDB 超表）

```sql
CREATE TABLE pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32) NOT NULL DEFAULT '',
    counter_group  VARCHAR(32) NOT NULL,
    counter_name   VARCHAR(64) NOT NULL,
    counter_value  DOUBLE PRECISION NOT NULL,
    granularity    SMALLINT NOT NULL DEFAULT 15
);

-- TimescaleDB 超表：按天分块
SELECT create_hypertable('pm_counters', 'time', chunk_time_interval => INTERVAL '1 day');

-- 压缩策略：7 天后自动压缩（约 10:1 压缩比）
SELECT add_compression_policy('pm_counters', INTERVAL '7 days');

-- 保留策略：90 天自动清理
SELECT add_retention_policy('pm_counters', INTERVAL '90 days');

-- 索引
CREATE INDEX idx_pm_counters_device_time ON pm_counters (device_id, time DESC);
CREATE INDEX idx_pm_counters_counter_time ON pm_counters (counter_group, counter_name, time DESC);
```

#### kpi_definitions（普通 PostgreSQL 表）

```sql
CREATE TABLE kpi_definitions (
    name         VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL,
    formula      TEXT NOT NULL,
    unit         VARCHAR(16) NOT NULL,
    category     VARCHAR(32) NOT NULL,
    carrier      VARCHAR(4),          -- NULL = 全局默认
    technology   VARCHAR(3),          -- NULL = 所有制式
    counters     JSONB NOT NULL DEFAULT '[]'
);
```

#### kpi_values（TimescaleDB 超表）

```sql
CREATE TABLE kpi_values (
    time       TIMESTAMPTZ NOT NULL,
    device_id  UUID NOT NULL,
    cell_id    VARCHAR(32) NOT NULL DEFAULT '',
    kpi_name   VARCHAR(64) NOT NULL,
    kpi_value  DOUBLE PRECISION NOT NULL,
    carrier    VARCHAR(4) NOT NULL,
    technology VARCHAR(3) NOT NULL
);

SELECT create_hypertable('kpi_values', 'time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_kpi_values_device_time ON kpi_values (device_id, time DESC);
CREATE INDEX idx_kpi_values_kpi_time ON kpi_values (kpi_name, time DESC);
```

#### pm_files（文件元数据表）

```sql
CREATE TABLE pm_files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     UUID NOT NULL,
    device_sn     VARCHAR(128) NOT NULL,
    carrier       VARCHAR(16) NOT NULL,
    technology    VARCHAR(16) NOT NULL,
    file_name     VARCHAR(512) NOT NULL,
    file_size     BIGINT DEFAULT 0,
    collect_time  TIMESTAMPTZ,         -- 从 PM 文件中提取的采集时间
    minio_path    VARCHAR(1024) NOT NULL,
    parsed        BOOLEAN DEFAULT FALSE,
    parsed_at     TIMESTAMPTZ,
    counter_count INT DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

#### pm_counters_hourly（连续聚合视图）

```sql
CREATE MATERIALIZED VIEW pm_counters_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    device_id, cell_id, counter_group, counter_name,
    SUM(counter_value)  AS sum_value,
    AVG(counter_value)  AS avg_value,
    MIN(counter_value)  AS min_value,
    MAX(counter_value)  AS max_value,
    COUNT(*)            AS sample_count
FROM pm_counters
GROUP BY bucket, device_id, cell_id, counter_group, counter_name;
```

### 7.2 MinIO 文件存储

```
pm-files/                          ← Bucket
  └── 2026/03/17/                  ← 日期分区
        └── DEVICE-SN-001/         ← 设备序列号
              ├── A00.4g01.pm.xml
              ├── A00.4g02.pm.xml
              └── ...
```

### 7.3 数据生命周期

| 数据 | 存储 | 保留时间 |
|------|------|---------|
| 原始 PM 文件 | MinIO | 长期保留 |
| pm_files 元数据 | PostgreSQL | 长期保留 |
| pm_counters 原始计数器 | TimescaleDB | 90 天（自动清理） |
| pm_counters 压缩数据 | TimescaleDB | 7 天后压缩，90 天后清理 |
| pm_counters_hourly 聚合 | TimescaleDB | 跟随源表 |
| kpi_values 计算结果 | TimescaleDB | 跟随 hypertable 策略 |

---

## 八、Worker 进程订阅注册

**文件**：`cmd/worker/main.go`

```go
// PM 处理链注册顺序：

// 1. 计数器存储层（TimescaleDB）
counterRepo := counter.NewPgCounterRepository(app.TsPool)

// 2. KPI 存储层（TimescaleDB）
kpiRepo := kpi.NewPgKPIRepository(app.TsPool)

// 3. KPI 引擎（加载所有运营商公式）
kpiEngine := kpi.NewKPIEngine(counterRepo, kpiRepo, app.Carriers, logger)

// 4. PM XML 解析器
pmParser := collector.NewPMXMLParser()

// 5. PM 文件元数据存储（PostgreSQL）
pmFileStore := pm.NewPgPMFileStore(app.PgPool)

// 6. PM 采集器（串联以上组件）
pmCollector := collector.NewPMCollector(
    app.MinIO, pmBucket,
    pmParser, counterRepo,
    kpiEngine, pmFileStore,
    app.EventBus, logger,
)
pmCollector.Subscribe(app.EventBus)  // → "pm.file.received" (queue: "pm-workers")

// 7. 文件传输桥（ACS 事件 → MinIO → 下游事件）
transferBridge := transfer.NewTransferBridge(
    deviceRepo, app.MinIO,
    pmBucket, mrBucket, logsBucket,
    app.EventBus, logger,
)
transferBridge.Subscribe(app.EventBus)  // → "device.inform.autonomous_transfer_complete"
```

**Queue Group 支持水平扩展**：多个 Worker 实例订阅同一 Queue Group，NATS 保证同一事件只被一个实例处理。

---

## 九、REST API 查询接口

**文件**：`internal/pm/handler.go`

| 端点 | 方法 | 说明 |
|------|------|------|
| `/pm/counters` | GET | 查询原始计数器样本（支持 device_id、cell_id、counter_name、时间范围过滤） |
| `/pm/counters/aggregated` | GET | 查询小时聚合数据（SUM/AVG/MIN/MAX/COUNT） |
| `/pm/kpi` | GET | 查询 KPI 计算结果（支持 kpi_name、carrier、technology 过滤） |
| `/pm/kpi/definitions` | GET | 查询 KPI 公式定义（可按 carrier/technology 过滤） |
| `/pm/kpi/calculate` | POST | 按需 KPI 计算（指定设备、小区、时间范围） |
| `/pm/files` | GET | 查询 PM 文件元数据列表 |
| `/pm/files/:id/download` | GET | 从 MinIO 下载原始 PM 文件 |
| `/pm/tasks` | POST | 创建 PM 批量提取任务 |
| `/pm/tasks` | GET | 查询任务列表（按 status/type 过滤） |

---

## 十、端到端完整流程示例

### 场景：CMCC LTE 基站定期上传 PM 文件

```
时间线：
T+0s   设备采集 15 分钟性能数据，生成 PM XML 文件
T+1s   设备主动上传文件到 ACS 文件服务器
T+5s   设备发送 AutonomousTransferComplete SOAP 消息
          ↓
=== ACS 引擎 (omcgo-acs) ===

T+5s   handler.handleAutonomousTransferComplete()
       ├─ 解码 SOAP: FileType="4", TransferURL="http://device:8080/pm/latest"
       ├─ 发布事件: "device.inform.autonomous_transfer_complete"
       │   Payload: {device_sn:"BST001", file_type:"4", transfer_url:...}
       └─ 返回 AutonomousTransferCompleteResponse
          ↓
=== Worker 进程 (omcgo-worker) ===

T+6s   TransferBridge.handleAutonomousTransferComplete()
       ├─ 查询设备: BST001 → {carrier:"cmcc", tech:"lte"}
       ├─ 分类: FileType="4" → PM 文件
       ├─ HTTP GET http://device:8080/pm/latest → 下载 XML
       ├─ MinIO PUT: pm-files/2026/03/17/BST001/A00.pm.xml
       └─ 发布事件: "pm.file.received"
           Payload: {minio_path:"2026/03/17/BST001/A00.pm.xml",
                     device_id:"uuid-001", carrier:"cmcc", technology:"lte"}
          ↓
T+7s   PMCollector.handleFileReceived()
       ├─ MinIO GET: pm-files/2026/03/17/BST001/A00.pm.xml
       ├─ 记录元数据: INSERT INTO pm_files (parsed=false)
       ├─ XML 流式解析:
       │   ├─ managedElement: deviceSN="BST001"
       │   ├─ measInfo: counterGroup="LTE.CellMeasReport"
       │   ├─ granPeriod: duration=PT900S(15min), endTime=2026-03-17T10:00:00Z
       │   ├─ measType: p=1→"rrc_conn_setup_att", p=2→"rrc_conn_setup_succ", ...
       │   └─ measValue: CellId=1234 → [1000, 950, ...], CellId=5678 → [800, 760, ...]
       │   结果: 2 cells × 10 counters = 20 PMCounter 记录
       │
       ├─ 批量写入: COPY TO pm_counters (20 rows)
       ├─ 更新元数据: UPDATE pm_files SET parsed=true, counter_count=20
       │
       ├─ KPI 计算 (CellId=1234):
       │   ├─ 时间窗口: [09:45:00, 10:00:00]
       │   ├─ 查询计数器: SELECT counter_name, SUM(counter_value) ...
       │   ├─ 注入 period_seconds=900
       │   ├─ 计算 10 个 LTE KPI:
       │   │   ├─ lte_rrc_setup_success_rate = 950/1000*100 = 95.0%
       │   │   ├─ lte_dl_throughput = volume*8/900/1000000 = xx Mbps
       │   │   └─ ... (8 more KPIs)
       │   └─ COPY TO kpi_values (10 rows)
       │
       ├─ KPI 计算 (CellId=5678):
       │   └─ 同上，另一个小区的 KPI
       │
       └─ 发布事件: "pm.file.parsed"
           Payload: {minio_path:..., device_id:..., counter_count:20}
          ↓
T+8s   完成。数据可通过 REST API 查询:
       GET /pm/counters?device_id=uuid-001&start_time=...
       GET /pm/kpi?device_id=uuid-001&kpi_name=lte_rrc_setup_success_rate
```

---

## 十一、数据流总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CPE 设备 (基站)                              │
│  ┌──────────┐     ┌──────────────┐     ┌──────────────────┐        │
│  │ PM 采集  │ ──→ │ 生成 XML 文件 │ ──→ │ HTTP Upload 到 ACS│        │
│  └──────────┘     └──────────────┘     └────────┬─────────┘        │
│                                                  │                  │
│  发送 AutonomousTransferComplete SOAP ←──────────┘                  │
└──────────────────────────────────────┬──────────────────────────────┘
                                       │
              ┌────────────────────────▼────────────────────────┐
              │              ACS 引擎 (:7547)                    │
              │                                                  │
              │  handleAutonomousTransferComplete()              │
              │      ↓                                          │
              │  EventBus.Publish("device.inform.autonomous_    │
              │                    transfer_complete")           │
              └────────────────────────┬────────────────────────┘
                                       │ NATS JetStream
              ┌────────────────────────▼────────────────────────┐
              │              Worker 进程                          │
              │                                                  │
              │  ┌─────────────────┐                             │
              │  │ TransferBridge  │ ← "device.inform.           │
              │  │                 │    autonomous_transfer_      │
              │  │ HTTP 下载文件   │    complete"                  │
              │  │ 分类 PM/MR/Log │                              │
              │  │ 存入 MinIO     │                              │
              │  └───────┬────────┘                              │
              │          │ "pm.file.received"                    │
              │  ┌───────▼────────┐                              │
              │  │  PMCollector   │                              │
              │  │                │                              │
              │  │ MinIO 下载     │                              │
              │  │ XML 流式解析   │                              │
              │  │ ↓              │                              │
              │  │ pm_counters    │ ──→ TimescaleDB              │
              │  │ ↓              │                              │
              │  │ KPIEngine      │                              │
              │  │ 公式计算       │                              │
              │  │ ↓              │                              │
              │  │ kpi_values     │ ──→ TimescaleDB              │
              │  │ ↓              │                              │
              │  │ pm_files 元数据│ ──→ PostgreSQL               │
              │  └───────┬────────┘                              │
              │          │ "pm.file.parsed"                      │
              └──────────┼──────────────────────────────────────┘
                         │
              ┌──────────▼──────────────────────────────────────┐
              │              App 主应用 (:8080)                   │
              │                                                  │
              │  REST API:                                       │
              │  GET /pm/counters          ← pm_counters         │
              │  GET /pm/counters/aggregated ← pm_counters_hourly│
              │  GET /pm/kpi               ← kpi_values          │
              │  GET /pm/kpi/definitions   ← kpi_definitions     │
              │  POST /pm/kpi/calculate    ← 按需计算             │
              │  GET /pm/files             ← pm_files            │
              │  GET /pm/files/:id/download ← MinIO              │
              └─────────────────────────────────────────────────┘
```

---

## 十二、关键设计总结

| 设计点 | 实现方式 | 设计理由 |
|--------|---------|---------|
| **流式 XML 解析** | `encoding/xml` Decoder | PM 文件可能很大（100K+ 计数器），避免全量加载到内存 |
| **批量写入** | `pgx.CopyFrom()` | 单次 COPY 比逐行 INSERT 快 10-50 倍 |
| **TimescaleDB 超表** | 按天分块 + 7 天压缩 + 90 天清理 | 时序数据自动管理，查询性能优异 |
| **连续聚合** | `pm_counters_hourly` 物化视图 | 小时粒度预计算，加速仪表盘查询 |
| **公式引擎** | 递归下降解析 + 运营商适配器 | 各运营商 KPI 定义不同，公式动态加载 |
| **合成计数器** | `period_seconds` 自动注入 | 吞吐量计算需要时间间隔，避免公式硬编码 |
| **事件级联** | ATC → TransferBridge → PMCollector | 每个阶段独立扩展，故障隔离 |
| **Queue Group** | `pm-workers` / `transfer-bridge` | 多 Worker 实例负载均衡，同一文件不重复处理 |
| **文件元数据** | `pm_files` 表 + `parsed` 标记 | 追踪文件处理状态，支持重试和审计 |
| **运营商适配** | `Carrier.KPIDefinitions(tech)` | 各运营商 KPI 公式不同，通过接口适配 |

---

## 十三、关键文件索引

| 文件路径 | 说明 |
|----------|------|
| `internal/acs/rpc/dispatcher.go` | Upload RPC 构建器 |
| `internal/acs/handler.go` | ATC 事件接收与发布 |
| `internal/transfer/bridge.go` | 文件下载、分类、MinIO 存储 |
| `internal/pm/collector/collector.go` | PM 采集器主逻辑 |
| `internal/pm/collector/parser.go` | 3GPP 32.435 XML 流式解析器 |
| `internal/pm/counter/pg_repository.go` | 计数器 TimescaleDB 读写 |
| `internal/pm/kpi/engine.go` | KPI 计算引擎 |
| `internal/pm/kpi/formula.go` | 公式解析与求值 |
| `internal/pm/kpi/pg_repository.go` | KPI 结果存储 |
| `internal/pm/aggregation/aggregator.go` | 连续聚合刷新 |
| `internal/pm/handler.go` | REST API 端点 |
| `internal/core/carrier/cmcc/kpi.go` | CMCC KPI 公式定义（LTE 10 项 + NR 6 项） |
| `internal/core/model/pm.go` | PMCounter / KPIValue / KPIDefinition 模型 |
| `cmd/worker/main.go` | Worker 订阅者注册 |
| `migrations/000010_create_pm_counters.up.sql` | pm_counters 超表 |
| `migrations/000011_create_kpi_tables.up.sql` | kpi_definitions + kpi_values + 连续聚合 |
| `migrations/000034_create_pm_files.up.sql` | pm_files 元数据表 |

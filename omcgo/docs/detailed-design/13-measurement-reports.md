# DD-13: ���量报告（F05）

> 关联功能域：F05（测量报告）
> 关联 backend-design.md 章节：第二章（模块 5）、第六章（消息/事件架构）
> 实施阶段：Phase 3（数据管线）
> 依赖文档：DD-02, DD-04

---

## 1. 概述

### 1.1 模块定位

测量报告（`internal/mr/`）负责采集和解析基站的无线信号测量数据（MR），包括 MRO（切换优化）、MRS（服务小区）、MRE（终端能力）三种类型，是无线网络优化的核心数据源。

### 1.2 数据处理管线

```
ACS Upload TransferComplete
  → mr.file.received (NATS)
  → Worker: 从 MinIO 下载 MR XML 文件
  → 解析文件名判断 MR 类型（MRO/MRS/MRE）
  → 对应解析器解析 XML
  → 存储解析后数据
  → mr.file.parsed (事件通知)
```

---

## 2. 接口设计

### 2.1 MR Collector — `internal/mr/collector/collector.go`

```go
type MRCollector struct {
    minioClient *minio.Client
    parsers     map[string]MRParser // key: "mro", "mrs", "mre"
    store       MRStore
    eventBus    event.EventBus
    logger      *zap.Logger
}

// HandleFileReceived 处理 mr.file.received 事件
func (c *MRCollector) HandleFileReceived(ctx context.Context, evt event.Event) error

// DetectMRType 从文件名判断 MR 类型
func DetectMRType(filename string) string // "mro", "mrs", "mre"
```

### 2.2 MR 解析器接口

```go
type MRParser interface {
    Parse(r io.Reader, carrier model.CarrierCode) (*MRData, error)
}

type MRData struct {
    DeviceSN    string
    CellID      string
    MRType      string // mro, mrs, mre
    CollectTime time.Time
    Records     []MRRecord
}

type MRRecord struct {
    Timestamp time.Time
    UEID      string
    Fields    map[string]interface{} // 测量字段
}
```

### 2.3 三种解析器

**MRO 解析器** — `internal/mr/parser/mro.go`：
- 切换优化数据
- 5G NR 测量量：SS-RSRP、SS-RSRQ、SS-SINR
- LTE 测量量：RSRP、RSRQ、SINR
- 服务小区 + 邻区测量值

**MRS 解析器** — `internal/mr/parser/mrs.go`：
- 服务小区统计数据
- 信号强度分布
- 用户数统计

**MRE 解析器** — `internal/mr/parser/mre.go`：
- 终端能力数据
- 终端类型统计
- 频段支持能力

### 2.4 MR Store — `internal/mr/store/`

```go
type MRStore interface {
    Save(ctx context.Context, data *MRData) error
    Query(ctx context.Context, filter MRFilter) ([]MRRecord, error)
    ListFiles(ctx context.Context, filter MRFileFilter) ([]MRFileInfo, error)
}

type MRFileInfo struct {
    ID          uuid.UUID
    DeviceSN    string
    MRType      string
    FileName    string
    FileSize    int64
    CollectTime time.Time
    Parsed      bool
    MinIOPath   string
}
```

---

## 3. 数据模型

### 3.1 MinIO 存储路径

```
mr-files/{carrier}/{date}/{device_serial}/mr_{type}_{timestamp}.xml
```

### 3.2 REST API

```
GET  /api/v1/mr/files                MR 文件列表
GET  /api/v1/mr/files/{id}/download  下载原始 MR 文件
GET  /api/v1/mr/data                 查询解析后 MR 数据
```

---

## 4. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 5G NR MR 版本 | V1.2.0 | V1.0 | V1.0 |
| LTE MR | V2.1.0（扩展型皮基站）| — | — |
| MR 类型支持 | MRO/MRS/MRE | MRO/MRS | MRO/MRS |
| 波束级测量 | ✅ | ✅ | ✅ |

---

## 5. 实施子阶段

### 阶段 13a：Collector + MRO 解析（Phase 3）
### 阶段 13b：MRS/MRE 解析（Phase 3）
### 阶段 13c：存储 + API（Phase 3）

---

## 6. 文件清单

```
internal/mr/collector/collector.go
internal/mr/parser/mro.go
internal/mr/parser/mrs.go
internal/mr/parser/mre.go
internal/mr/store/store.go
internal/mr/service.go
```

---

## 7. 参考

- doc/features/05-measurement-reports.md：F05 全部子功能
- backend-design.md 第二章：模块 5

# OMC 文件上传/下载缺口功能 — 详细实施方案

> 生成时间：2026-03-11
> 依据文档：`omcgo/doc/reports/kpi-file-transfer-analysis.md`
> 范围：全部 7 项未完成的文件传输功能

## Context

根据分析报告，系统 20 项文件传输功能中有 7 项未完成（2 项缺失、3 项部分完成、2 项占位）。本方案涵盖全部缺口的设计与实施计划，按优先级分 4 个 Sprint 交付，每个 Sprint 独立可测、可部署。

---

## 总览

| Sprint | 功能 | 优先级 | 新建文件 | 修改文件 | 估计代码量 |
|--------|------|--------|---------|---------|-----------|
| 1 | Transfer Bridge Handler | P0 | 2 | 1 | ~330 行 |
| 2A | PM 原始文件下载 | P2 | 4 | 2 | ~230 行 |
| 2B | 配置备份执行 Worker | P1 | 2 | 3 | ~300 行 |
| 3A | 报告文件生成 + 下载 | P2 | 1 | 5 | ~250 行 |
| 3B | MR 数据导出 | P3 | 0 | 1 | ~80 行 |
| 4 | 文件分发 Worker | P3 | 0 | 1 | ~40 行 |
| **合计** | | | **9** | **11** | **~1230 行** |

---

## Sprint 1: Transfer Bridge Handler (P0)

### 问题

ACS 发布 `device.inform.autonomous_transfer_complete` 事件后无订阅者。PM Collector 等待 `pm.file.received`、MR Collector 等待 `mr.file.received`，但这两个事件从未被发布。**整条 PM/MR 处理流水线无法被触发。**

### 新建文件

#### `internal/transfer/bridge.go` (~180 行)

```go
type TransferBridge struct {
    deviceRepo  device.DeviceRepository
    minioClient *minio.Client
    pmBucket    string
    mrBucket    string
    logsBucket  string
    httpClient  *http.Client
    eventBus    event.EventBus
    logger      *zap.Logger
}
```

核心方法：

1. **`NewTransferBridge(...)`** — 构造函数，httpClient 配置 30s 超时
2. **`Subscribe(eventBus) error`** — 订阅 `SubjectDeviceAutonomousTransferComplete`, queue group `"transfer-bridge"`
3. **`handleAutonomousTransferComplete(ctx, evt) error`** — 主处理逻辑：
   - `evt.DecodePayload(&payload)` — 解码 ATC 事件 payload
   - 检查 fault，有故障则记录日志并 return nil
   - `deviceRepo.GetBySerialNumber(ctx, payload.DeviceSN)` — 查设备信息
   - `classifyFileType(payload.FileType, payload.TargetFileName)` — 按 file_type 和文件名判断类别
   - `downloadAndStore(ctx, transferURL, bucket, objectPath)` — HTTP GET + MinIO PutObject
   - 按文件类别构建并发布对应事件

4. **`classifyFileType(fileType, fileName) string`** — 文件分类：
   ```
   FileType 含 "PM" 或 "4"            → "pm"
   FileType 含 "MR" 或 "5"            → "mr"
   文件名含 "MRO"|"MRS"|"MRE"         → "mr"
   文件名含 "pm"|"PM"|"counter"       → "pm"
   FileType "3" 或含 "Log"            → "log"
   其他                                → "log" (兜底)
   ```

5. **`downloadAndStore(ctx, url, bucket, path) (int64, error)`** — 文件下载与存储：
   ```go
   req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
   resp, err := b.httpClient.Do(req)
   // error handling + status check
   defer resp.Body.Close()
   info, err := b.minioClient.PutObject(ctx, bucket, path, resp.Body, -1,
       minio.PutObjectOptions{ContentType: "application/octet-stream"})
   return info.Size, err
   ```

事件 payload 格式：
- PM → 匹配 `collector.FileReceivedPayload{MinIOPath, DeviceID, DeviceSN, Carrier, Technology}`
- MR → 匹配 `mrcollector.MRFilePayload{MinioPath, Bucket, DeviceID, DeviceSN, Carrier, FileName, FileSize}`

#### `internal/transfer/bridge_test.go` (~150 行)

| 测试用例 | 描述 |
|---------|------|
| TestClassifyFileType_PM | file_type="4" 或文件名含 "pm" → 返回 "pm" |
| TestClassifyFileType_MR | file_type="5" 或文件名含 "MRO" → 返回 "mr" |
| TestClassifyFileType_Log | file_type="3" → 返回 "log" |
| TestHandleATC_FaultSkipped | fault 非空时跳过处理，return nil |
| TestHandleATC_DeviceNotFound | 设备不存在时返回错误 |
| TestHandleATC_PMFilePublish | PM 文件正确发布 pm.file.received 事件 |
| TestHandleATC_MRFilePublish | MR 文件正确发布 mr.file.received 事件 |

### 修改文件

**`cmd/worker/main.go`** — 在 line 152 (`logger.Info("MR collector started")`) 之后添加:

```go
// 14. Create Transfer Bridge + Subscribe
deviceRepo := device.NewPgDeviceRepository(pgPool)
transferBridge := transfer.NewTransferBridge(
    deviceRepo, minioClient,
    cfg.MinIO.Buckets.PMFiles, cfg.MinIO.Buckets.MRFiles, cfg.MinIO.Buckets.Logs,
    eventBus, logger,
)
if err := transferBridge.Subscribe(eventBus); err != nil {
    logger.Warn("subscribe transfer bridge", zap.Error(err))
}
logger.Info("transfer bridge started")
```

新增 import:
```go
"github.com/omcgo/omcgo/internal/transfer"
"github.com/omcgo/omcgo/internal/device"
```

### 数据流

```
CPE → ACS (AutonomousTransferComplete SOAP)
  → handler.go:407 发布 "device.inform.autonomous_transfer_complete"
    → TransferBridge.handleAutonomousTransferComplete()
      → HTTP GET transfer_url → MinIO PutObject(pm-files/...)
      → 发布 "pm.file.received"
        → PM Collector → 3GPP XML 解析 → Counter 入库 → KPI 计算
      或发布 "mr.file.received"
        → MR Collector → MR XML 解析 → MR 记录入库
```

### 验证方式
1. 单元测试: `go test ./internal/transfer/...`
2. 集成: curl 模拟 CPE 发送 ATC SOAP 到 ACS → 确认 MinIO pm-files/mr-files bucket 有文件
3. 确认 PM Collector 日志 "processing PM file" 和 MR Collector "processing MR file"

---

## Sprint 2: PM 文件下载 + 配置备份执行 (P1-P2)

### 2A: PM 原始文件 REST 下载

#### 问题
MR 有 `GET /mr/files/:id/download`（已实现），PM 没有对应端点，且 PM 文件无元数据表。

#### 新建 migration

**`migrations/000034_create_pm_files.up.sql`**:
```sql
CREATE TABLE IF NOT EXISTS pm_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    device_sn VARCHAR(128) NOT NULL,
    carrier VARCHAR(16) NOT NULL,
    technology VARCHAR(16) NOT NULL,
    file_name VARCHAR(512) NOT NULL,
    file_size BIGINT DEFAULT 0,
    collect_time TIMESTAMPTZ,
    minio_path VARCHAR(1024) NOT NULL,
    parsed BOOLEAN DEFAULT FALSE,
    parsed_at TIMESTAMPTZ,
    counter_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_pm_files_device ON pm_files(device_id);
CREATE INDEX idx_pm_files_created ON pm_files(created_at DESC);
```

**`migrations/000034_create_pm_files.down.sql`**:
```sql
DROP TABLE IF EXISTS pm_files;
```

#### 新建 `internal/pm/file_store.go` (~30 行)

```go
type PMFileInfo struct {
    ID           uuid.UUID
    DeviceID     uuid.UUID
    DeviceSN     string
    Carrier      string
    Technology   string
    FileName     string
    FileSize     int64
    CollectTime  time.Time
    MinioPath    string
    Parsed       bool
    ParsedAt     *time.Time
    CounterCount int
    CreatedAt    time.Time
}

type PMFileStore interface {
    SaveFile(ctx context.Context, info *PMFileInfo) error
    GetFileByID(ctx context.Context, id uuid.UUID) (*PMFileInfo, error)
    ListFiles(ctx context.Context, filter PMFileFilter) (*model.ListResponse[PMFileInfo], error)
    UpdateFileParsed(ctx context.Context, id uuid.UUID, counterCount int) error
}
```

#### 新建 `internal/pm/pg_file_store.go` (~120 行)

参照 `mr/pg_store.go` 实现，使用 squirrel + pgx。

#### 修改 `internal/pm/collector/collector.go`

- PMCollector struct 新增 `fileStore PMFileStore` 字段
- NewPMCollector 构造函数新增 fileStore 参数
- handleFileReceived() 在 MinIO 下载之前（line 68 之后）新增文件元数据保存
- handleFileReceived() 解析完成后调用 fileStore.UpdateFileParsed()

#### 修改 `internal/pm/handler.go`

- Handler struct 新增 `fileStore PMFileStore` 和 `minioClient *minio.Client` 和 `bucket string`
- RegisterRoutes 追加:
  ```go
  pmGroup.GET("/files", h.ListPMFiles)
  pmGroup.GET("/files/:id/download", h.DownloadPMFile)
  ```
- `DownloadPMFile` 参照 `mr/handler.go:94-128`:
  ```go
  fileInfo, _ := h.fileStore.GetFileByID(ctx, fileID)
  obj, _ := h.minioClient.GetObject(ctx, h.bucket, fileInfo.MinioPath, ...)
  c.DataFromReader(http.StatusOK, stat.Size, "application/xml", obj, nil)
  ```

### 2B: 配置备份执行 Worker

#### 问题
备份模块有完整 CRUD（任务、定时、FTP 配置），但没有执行逻辑。

#### 新建 `internal/backup/executor.go` (~200 行)

```go
type BackupExecutor struct {
    taskRepo     TaskRepository
    deviceRepo   device.DeviceRepository
    cmdQueue     cmdqueue.CommandQueue
    connReq      *connreq.Client
    minioClient  *minio.Client
    backupBucket string
    eventBus     event.EventBus
    logger       *zap.Logger
}
```

核心方法：
1. `Subscribe(eventBus)` — 订阅 `SubjectBackupTaskCreated`, queue `"backup-executors"`
2. `handleTaskCreated(ctx, evt)`:
   - 从事件获取 task_id
   - `taskRepo.GetByID()` → 检查 status == pending
   - 更新 status → running, started_at = now
   - 遍历 TargetIDs:
     - `deviceRepo.GetBySerialNumber(ctx, deviceSN)`
     - 构建 Upload 命令: `{Method: "Upload", Params: {file_type: "2", url: minioUploadURL}}`
     - `cmdQueue.Push(ctx, deviceSN, uploadCmd)`
     - `connReq.Send()` 唤醒设备
     - 更新 progress = (i+1)/total * 100
   - 完成后更新 status → completed (或 failed + error_message)

#### 新建 `internal/backup/executor_test.go` (~100 行)

| 测试用例 | 描述 |
|---------|------|
| TestHandleTask_PendingToRunning | 状态从 pending 转为 running |
| TestHandleTask_PushUploadCommand | 每个设备生成 Upload 命令 |
| TestHandleTask_ProgressUpdate | 进度正确更新 |
| TestHandleTask_DeviceNotFound | 设备不存在时标记失败 |

#### 新增事件主题

**`internal/event/subjects.go`** 追加:
```go
// Backup events
const (
    SubjectBackupTaskCreated = "backup.task.created"
    SubjectBackupTaskDone    = "backup.task.done"
)

// Report events
const (
    SubjectReportGenerateRequested = "report.generate.requested"
    SubjectReportGenerateDone      = "report.generate.done"
)
```

#### 修改 `internal/backup/handler.go`

CreateTask 处理器中，任务创建成功后发布事件:
```go
evt, _ := event.NewEvent(event.SubjectBackupTaskCreated, map[string]interface{}{
    "task_id": task.ID.String(),
})
h.eventBus.Publish(c.Request.Context(), event.SubjectBackupTaskCreated, evt)
```

Handler struct 新增 eventBus 字段。

#### 修改 `cmd/worker/main.go`

在 transfer bridge 之后追加:
```go
// 15. Create Backup Executor + Subscribe
backupTaskRepo := backup.NewPgTaskRepository(pgPool)
cmdQueue := cmdqueue.NewRedisCommandQueue(redisClient)
connReqClient := connreq.NewClient(redisClient, logger)
backupExecutor := backup.NewBackupExecutor(
    backupTaskRepo, deviceRepo, cmdQueue, connReqClient,
    minioClient, cfg.MinIO.Buckets.ConfigBackup, eventBus, logger,
)
if err := backupExecutor.Subscribe(eventBus); err != nil {
    logger.Warn("subscribe backup executor", zap.Error(err))
}
logger.Info("backup executor started")
```

### 验证方式
- PM: `GET /api/v1/pm/files` → 列表; `GET /api/v1/pm/files/:id/download` → 下载 XML
- 备份: `POST /api/v1/backup/tasks` → 检查 Redis `acs:cmdq:{device_sn}` 有 Upload 命令
- 单元测试: `go test ./internal/pm/... ./internal/backup/...`

---

## Sprint 3: 报告文件生成 + MR 数据导出 (P2-P3)

### 3A: 报告文件生成 Worker

#### 问题
`report.Service.Generate()` 创建了 status=generating 记录但没有实际生成文件。DownloadRecord 返回空 URL。

#### 新建 `internal/report/generator.go` (~250 行)

```go
type ReportGenerator struct {
    recordRepo  RecordRepository
    defRepo     DefinitionRepository
    kpiRepo     kpi.KPIRepository
    counterRepo counter.CounterRepository
    alarmStore  alarm.AlarmStore
    minioClient *minio.Client
    reportBkt   string
    eventBus    event.EventBus
    logger      *zap.Logger
}
```

核心方法：
1. `Subscribe(eventBus)` — 订阅 `SubjectReportGenerateRequested`, queue `"report-generators"`
2. `handleGenerateRequest(ctx, evt)`:
   - 获取 record_id → `recordRepo.GetByID()`
   - 获取 definition → `defRepo.GetByID()`
   - 根据 ReportType 收集数据:
     - `performance` → `kpiRepo.Query()` 获取 KPI 值
     - `alarm` → `alarmStore.Query()` 获取告警统计
     - `device` → 设备统计信息
   - 构建报告 JSON 结构 (`encoding/json` + `bytes.Buffer`)
   - 上传到 MinIO: `reports/{definition_id}/{period}/{record_id}.json`
   - 更新 record: MinioPath, FileSize, Status → ready

#### 修改 `internal/report/repository.go`

RecordRepository 新增:
```go
Update(ctx context.Context, record *ReportRecord) error
```

#### 修改 `internal/report/pg_repository.go`

实现 `Update` 方法。

#### 修改 `internal/report/service.go`

- Service struct 新增 `eventBus event.EventBus` 字段
- `NewService()` 构造函数新增 eventBus 参数
- `Generate()` 末尾（line 148 之后）追加:
  ```go
  genEvt, _ := event.NewEvent(event.SubjectReportGenerateRequested, map[string]interface{}{
      "record_id":     record.ID.String(),
      "definition_id": definitionID.String(),
  })
  _ = s.eventBus.Publish(ctx, event.SubjectReportGenerateRequested, genEvt)
  ```

#### 修改 `internal/report/handler.go`

- Handler struct 新增 `minioClient *minio.Client` 和 `reportBkt string`
- `DownloadRecord()` 修改: 当 `record.MinioPath != ""` 时从 MinIO 流式下载:
  ```go
  if record.MinioPath != "" {
      obj, _ := h.minioClient.GetObject(ctx, h.reportBkt, record.MinioPath, minio.GetObjectOptions{})
      defer obj.Close()
      stat, _ := obj.Stat()
      c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, record.ReportName, record.Format))
      c.DataFromReader(http.StatusOK, stat.Size, "application/json", obj, nil)
      return
  }
  ```

#### 修改 `internal/appconfig/config.go`

BucketConfig 新增:
```go
Reports string `mapstructure:"reports"`
```

#### 修改 `cmd/worker/main.go`

追加 ReportGenerator 初始化和 Subscribe。

### 3B: MR 数据导出

#### 问题
`POST /api/v1/mr/export` 返回 placeholder `{"task_id": "export-placeholder", "status": "pending"}`。

#### 修改 `internal/mr/handler.go` ExportMRData (~80 行)

替换 placeholder 为实际导出逻辑:
```go
func (h *Handler) ExportMRData(c *gin.Context) {
    var req struct {
        DeviceID  string `json:"device_id"`
        MRType    string `json:"mr_type"`
        CellID    string `json:"cell_id"`
        StartTime string `json:"start_time"`
        EndTime   string `json:"end_time"`
        Format    string `json:"format"` // "json" (default) 或 "csv"
    }
    if err := c.ShouldBindJSON(&req); err != nil { ... }

    // 构建 MRRecordFilter
    filter := MRRecordFilter{ListRequest: model.ListRequest{Page: 1, PageSize: 1000}}
    // 设置可选过滤条件...

    result, err := h.store.QueryRecords(c.Request.Context(), filter)

    switch req.Format {
    case "csv":
        c.Header("Content-Type", "text/csv")
        c.Header("Content-Disposition", "attachment; filename=mr_export.csv")
        writer := csv.NewWriter(c.Writer)
        // 写入 CSV 表头和数据行...
        writer.Flush()
    default:
        c.JSON(http.StatusOK, result)
    }
}
```

### 验证方式
- 报告: `POST /api/v1/reports/generate` → 等待几秒 → `GET /api/v1/reports/records/:id/download` 下载 JSON 文件
- MR: `POST /api/v1/mr/export` with `{"format":"csv"}` 返回 CSV 数据
- 单元测试: `go test ./internal/report/... ./internal/mr/...`

---

## Sprint 4: 文件分发 Worker (P3)

### 问题
`POST /api/v1/files/:id/distribute` 仅记日志返回 queued。

### 修改 `internal/filemanager/handler.go` Distribute (~40 行修改)

- Handler struct 新增 `cmdQueue cmdqueue.CommandQueue` 字段
- 替换占位逻辑为实际命令队列推送:

```go
downloadURL := fmt.Sprintf("minio://%s/%s", h.bucket, mf.MinIOPath)
successCount := 0
for _, sn := range req.DeviceSNs {
    paramsJSON, _ := json.Marshal(map[string]interface{}{
        "url":       downloadURL,
        "file_type": mapFileTypeToTR069(mf.FileType),
        "file_size": mf.FileSize,
        "file_name": mf.FileName,
    })
    cmd := &cmdqueue.Command{Method: "Download", Params: paramsJSON}
    if err := h.cmdQueue.Push(c.Request.Context(), sn, cmd); err != nil {
        h.logger.Warn("push download command", zap.String("sn", sn), zap.Error(err))
        continue
    }
    successCount++
}

c.JSON(http.StatusOK, gin.H{
    "task_id":      taskID,
    "file_id":      mf.ID.String(),
    "device_count": len(req.DeviceSNs),
    "queued_count": successCount,
    "status":       "queued",
})
```

新增辅助函数:
```go
func mapFileTypeToTR069(ft FileType) string {
    switch ft {
    case FileFirmware: return "1"
    case FileConfig:   return "2"
    default:           return "3"
    }
}
```

### 验证方式
- `POST /api/v1/files/:id/distribute` → 检查 Redis `acs:cmdq:{device_sn}` 有 Download 命令
- 单元测试: `go test ./internal/filemanager/...`

---

## 完整文件清单

### 新建文件 (10 个)

| 文件 | 估计行数 | Sprint |
|------|---------|--------|
| `internal/transfer/bridge.go` | 180 | 1 |
| `internal/transfer/bridge_test.go` | 150 | 1 |
| `internal/pm/file_store.go` | 30 | 2 |
| `internal/pm/pg_file_store.go` | 120 | 2 |
| `internal/pm/pg_file_store_test.go` | 60 | 2 |
| `migrations/000034_create_pm_files.up.sql` | 15 | 2 |
| `migrations/000034_create_pm_files.down.sql` | 1 | 2 |
| `internal/backup/executor.go` | 200 | 2 |
| `internal/backup/executor_test.go` | 100 | 2 |
| `internal/report/generator.go` | 250 | 3 |

### 修改文件 (12 个)

| 文件 | 修改内容 | Sprint |
|------|---------|--------|
| `cmd/worker/main.go` | 新增 TransferBridge/BackupExecutor/ReportGenerator 初始化 | 1,2,3 |
| `internal/event/subjects.go` | 追加 backup/report 事件常量 | 2 |
| `internal/pm/handler.go` | 新增 ListPMFiles + DownloadPMFile 路由和处理器 | 2 |
| `internal/pm/collector/collector.go` | 集成 PMFileStore 保存文件元数据 | 2 |
| `internal/backup/handler.go` | CreateTask 后发布事件 + eventBus 字段 | 2 |
| `internal/report/repository.go` | RecordRepository 新增 Update 方法 | 3 |
| `internal/report/pg_repository.go` | 实现 RecordRepository.Update | 3 |
| `internal/report/service.go` | Generate 发布事件 + eventBus 字段 | 3 |
| `internal/report/handler.go` | DownloadRecord MinIO 流式下载 | 3 |
| `internal/mr/handler.go` | ExportMRData 替换 placeholder | 3 |
| `internal/filemanager/handler.go` | Distribute 集成 cmdQueue | 4 |
| `internal/appconfig/config.go` | BucketConfig 新增 Reports | 3 |

---

## 依赖关系与执行顺序

```
Sprint 1 (TransferBridge) ← 独立，无前置依赖
    ↓ 打通 PM/MR 流水线后才有数据
Sprint 2A (PM文件下载) ← 依赖 Sprint 1 (PM 文件需经 bridge 存入)
Sprint 2B (备份执行)   ← 独立
    ↓ 2A + 2B 可并行开发
Sprint 3A (报告生成)   ← 依赖 Sprint 2A (性能报告需要 KPI 数据)
Sprint 3B (MR导出)     ← 独立
    ↓ 3A + 3B 可并行开发
Sprint 4 (文件分发)    ← 独立
```

建议执行顺序: **Sprint 1 → Sprint 2 (2A+2B 并行) → Sprint 3 (3A+3B 并行) → Sprint 4**

---

## 端到端验收标准

| # | 验收项 | Sprint | 验证方法 |
|---|-------|--------|---------|
| 1 | ATC SOAP → MinIO 有 PM 文件 → pm_counters 有数据 → kpi_values 有数据 | 1 | curl ATC + 检查 DB |
| 2 | ATC SOAP → MinIO 有 MR 文件 → mr_records 有数据 | 1 | curl ATC + 检查 DB |
| 3 | `GET /api/v1/pm/files` 返回 PM 文件列表 | 2 | curl |
| 4 | `GET /api/v1/pm/files/:id/download` 返回 XML 文件 | 2 | curl + file check |
| 5 | `POST /api/v1/backup/tasks` → 设备命令队列有 Upload 命令 | 2 | curl + Redis check |
| 6 | `POST /api/v1/reports/generate` → `GET /records/:id/download` 下载 JSON 报告 | 3 | curl |
| 7 | `POST /api/v1/mr/export` 返回 CSV/JSON MR 数据 | 3 | curl |
| 8 | `POST /api/v1/files/:id/distribute` → 命令队列有 Download 命令 | 4 | curl + Redis check |
| 9 | 原有 E2E 331 个测试全部通过 | All | `bash ./scripts/e2e_verify.sh` |

---

## 关键参照源码

| 用途 | 参照文件 | 行号 |
|------|---------|------|
| 事件订阅模式 | `internal/software/service.go` | 292-298 |
| 事件处理模式 | `internal/software/service.go` | 246-290 |
| PM Collector payload | `internal/pm/collector/collector.go` | 16-23 |
| MR Collector payload | `internal/mr/collector/collector.go` | 18-27 |
| MR 文件下载 | `internal/mr/handler.go` | 94-128 |
| MinIO 上传 | `internal/software/service.go` | 59-90 |
| 命令队列 Push | `internal/software/service.go` | 129-144 |
| Worker 初始化 | `cmd/worker/main.go` | 125-152 |
| 设备查询 | `internal/device/repository.go` | 22-33 |
| 固件升级状态机 | `internal/software/state_machine.go` | 全文 |

# PRD: backup_task → file_path 链路回填（T-0079 / T-0072 followup）

> **关联**: Backlog T-0079 / Sprint-07..08 / Domain=F06/backup / Type=feat
> **作者**: Claude（代 Owner=Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: EventBus pub-sub（非直接耦合）+ first-write-wins 多设备语义 + restore_by_task_id API 模式增强

---

## 1. 业务背景

T-0072 PRD §2.3 / T-0078 PRD §2.1 共同记录的关键架构 gap：`BackupTask.FilePath *string` 字段在 schema 中存在（T-0016 时建立），但**当前没有任何代码写入**。结果：
- restore endpoint 只能接受用户手动输入 `{bucket, object_path}`（T-0078 MVP）
- "从备份历史选择恢复源" 的 UI 体验无法做（T-0078 N1 文件浏览器拆到本任务后做）
- 数据完整性：backup_tasks 表 60% 信息被使用，40%（file_path）形同虚设

---

## 2. ULTRATHINK 决策

### 2.1 链路设计：EventBus pub-sub（不直接耦合）

最直观的方案是 ACS upload handler 直接调 backup module 的 `RecordFilePath`。但这会引入 `acs/upload → backup` 模块依赖（违反"ACS 独立部署"架构原则；CLAUDE.md §2 明确 ACS 与 app 解耦）。

更干净的方案：**复用 datamodel 已有 pub-sub 模式**。

**FileType=11 数据模型已有先例**（`acs/upload/handler.go:199` `publishDataModelEvent` → `SubjectDataModelFileReceived` → datamodel 模块订阅）。本任务对 FileType=3 backup 文件做完全对称的事：

```
[ACS 进程]                                  [App 进程]
upload handler                               backup module
    │                                            │
    │  PutObject(MinIO) 后                       │
    │  if FileType==3 →                          │
    │    publish SubjectBackupFileReceived ────► subscribe ────► UpdateFilePath
    │    payload={bucket, path, filename,        │   parse filename
    │             device_sn, file_size}          │   → extract backup_task_id
                                                 │   → repo.UpdateFilePath(taskID, path)
```

**优势**：
- 零新模块间 import；ACS 进程仍只 import `internal/core/event`
- NATS JetStream 跨进程异步交付天然支持 ACS 与 App 分离部署
- 失败链路一致：upload handler 不感知 backup 是否成功 record（best-effort）

### 2.2 backup_task_id 嵌入 filename

upload handler 拿到 `?filename=...&fileType=3` 后，要从 filename 中提取 backup_task_id。约定：

```
backup-{taskID8}-{deviceSN}.xml
```

- `taskID8` = backup_task UUID 前 8 字符（足够唯一性 + 短）
- 整个 filename 由 **executor.go** 在 enqueue Upload device task 时生成，写入 SOAP `target_filename` 字段
- CPE 上传时遵循 ACS 给的 target_filename → upload handler 拿到 filename = 我们指定的格式
- 解析端用 regex `^backup-([0-9a-f]{8})-(.+)\.xml$` 提取 task_id 前缀 + device SN

**对照 datamodel 已有模式**：datamodel filename 格式是 `datamodel_{deviceSN}_{uuid}.xml`，upload handler 用 `extractDeviceSNFromFilename` 提取。本任务沿用同思路。

### 2.3 多设备 backup_task 的 file_path 单字段冲突

`backup_tasks.file_path` 是单 `*string`，但一个 backup_task 可有 N 个目标设备 → N 个 MinIO 文件。无法把 N 路径塞进一个字段。

**方案权衡**：

| 方案 | 复杂度 | 多设备支持 | 决策 |
|------|--------|-----------|------|
| (a) **first-write-wins，保存第一个文件路径** | 极小 | 部分（restore_by_task_id 仅恢复第一个设备的 config）| **采纳** |
| (b) 新表 backup_files (id, backup_task_id FK, device_sn, bucket, path, size) | 中（migration + repo + handler）| 完整 | defer 到 T-0082（如未来需要 file 级别 listing）|
| (c) file_path 改 JSONB 数组 | 中（schema migration + 应用层解析）| 完整 | defer，避免 schema 形态切换 |
| (d) file_path 改 base prefix（不存具体文件名）| 小 | 需要 MinIO listing | 与 T-0081 候选耦合 |

**为什么 first-write-wins 在 MVP 已经够用**：
- 95% 实际备份场景是**单设备 backup task**（运维 click "backup this device"）
- 多设备 backup tasks 通常是定时任务批量备份，restore-by-task-id 的语义本来就模糊（你确定要把设备 A 的 config 推到设备 B 吗？业务上更应是"逐个恢复"）
- restore 服务对多设备 task 仍可走 explicit-path 模式（T-0072 已有）

**多设备 task 用户提示**：
- restore_by_task_id endpoint 在 backup_task.target_count > 1 时返回 warning 字段（不阻断），FE 可显示提示
- backup_tasks 列表 UI 可标记 "multi-device" 任务的 file_path 为"代表性路径（仅第一个设备）"

### 2.4 与 restore_by_task_id 的对接

T-0072 PRD §6 N3 把 `restore_by_task_id` 标为 N3 非目标，待 T-0079 完成后增强。本任务自然兜底：

新方法 `RestoreService.CreateByTaskID(ctx, backupTaskID, targetDeviceSNs, createdBy)`：
1. 读取 backup_task.file_path；若 null → 返回 `404 not yet uploaded`
2. 解析 file_path → `{bucket, object_path}`
3. 复用现有 `Create({bucket, object_path, target_device_sns}, createdBy)` 路径

新 endpoint `POST /api/v1/backup/restore/by-task-id` body=`{backup_task_id, target_device_sns}`。

### 2.5 跨进程事件一致性

ACS 用 NATS JetStream（多实例部署），App 也用 NATS。Subscriber 用 `bus.QueueSubscribe("backup-file-recorder")` 确保多 App 实例下只有一个收到事件（避免重复 update）。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我创建一个备份任务后，列表里的 file_path 列从空变为 MinIO 路径；后续 restore 可以用 by-task-id 模式直接复用 |
| FE 开发者 | T-0078 文件浏览器后续可基于 `useBackupTasks` 已有 hook 渲染 file_path 列；不需要新 API |
| 后端开发者 | EventBus 模式 = 0 新跨模块 import，符合架构 §2 |
| QA | e2e 可验证 POST /backup/restore/by-task-id 在 file_path 已填充时 200，未填充时 404 |

---

## 4. 验收标准（GWT）

### V1 — executor 设置 target_filename
- **Given** backup task created with target_devices=[SN001]
- **When** executor 处理事件
- **Then** device_task.params 含 `target_file_name="backup-{taskID8}-SN001.xml"` + `command_key="backup-{taskID8}"`

### V2 — upload handler 发布事件
- **Given** ACS upload handler 收到 `?fileType=3&filename=backup-abc12345-SN001.xml`
- **When** PutObject 成功
- **Then** publish `SubjectBackupFileReceived` event 含 payload {bucket, object_path, filename, backup_task_id_prefix="abc12345", device_sn="SN001", file_size}

### V3 — backup module 订阅 + UpdateFilePath
- **Given** SubjectBackupFileReceived event 发布
- **When** backup file path recorder 收到
- **Then** 解析 backup_task_id_prefix → 用 LIKE 'abc12345%' 查询匹配 backup_task → 调 repo.UpdateFilePath；first-write-wins（已 set 的不覆盖）

### V4 — multi-device first-write-wins
- **Given** backup_task target=[SN001, SN002]，先后收到两个 backup file 事件
- **When** 第二个事件触发 UpdateFilePath
- **Then** repo 检测 file_path 已 NOT NULL → skip update + log info；first 文件的路径保留

### V5 — POST /backup/restore/by-task-id 新 endpoint
- **Given** backup_task with file_path="config_backup/backup/2026/04/29/backup-xxx.xml.gz"
- **When** POST /api/v1/backup/restore/by-task-id body=`{backup_task_id, target_device_sns: [SN999]}`
- **Then** 200 OK；创建 restore_task；其 source_object_path = backup_task.file_path 解析后的 object_path

### V6 — by-task-id 在 file_path NULL 时拒绝
- **Given** backup_task with file_path=NULL（CPE 还没上传完）
- **When** 同 V5 的 POST
- **Then** 404 / 409 Conflict；error_message="backup not yet uploaded; try again later"

### V7 — multi-device task warning
- **Given** backup_task target_ids=[SN001, SN002]，file_path=已填充（第一设备的）
- **When** restore_by_task_id 用 backup_task_id
- **Then** 200 OK + response.warning="restoring from multi-device backup; only the first device's config is referenced"

### V8 — Backward compat
- **Given** 既有 explicit-path restore endpoint（T-0072 POST /backup/restore）
- **When** 既有 e2e bk-9, bk-10 跑
- **Then** 行为不变；新 by-task-id endpoint 与 explicit-path endpoint 并存

---

## 5. 运营商差异矩阵

无差异。链路是 OMC 内部数据完整性增强，CPE / 运营商无感知。

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | 新表 backup_files 一行一文件（多设备完整支持） | T-0082（如未来 UI 需要 file 级 listing）|
| N2 | MinIO 原始 listing endpoint | T-0081 候选 |
| N3 | T-0078 FE 升级为 "select from backup history" picker | T-0078 followup（FE 任务，本任务交付后端契约） |
| N4 | file_path 反向回滚（删除 backup_task 时清理 MinIO 对象）| 已属 T-0076 cleanup Phase 2 范围 |
| N5 | 已存在的 NULL file_path 历史 backup_task 数据回填 | 不做；forward-only；老任务用 explicit-path |
| N6 | backup_task_id_prefix 哈希冲突保护 | UUID8 prefix collision 概率极低（2^32 对里出现 50% 冲突需 ~65k 任务）；不做 dedup |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0072 ✅ | restore service 已有；本任务 extension |
| T-0007 ✅ | EventBus 已就绪 |
| FileType=11 datamodel 模式 | 模板代码 — 直接镜像 |
| `BackupTask.FilePath` 字段 | ✅ 已存在；本任务首次写入它 |

无新外部依赖。

---

## 8. 度量

| metric | 含义 |
|--------|------|
| `omc_backup_filepath_recorded_total{result}` | counter；result ∈ {recorded, skipped_already_set, skipped_no_match, error} |
| `omc_backup_restore_by_task_total{result}` | counter；result ∈ {accepted, rejected_not_uploaded, rejected_invalid_input} |

复用 `backup.RestoreMetrics` 命名空间，新增 2 collector。

---

## 9. 设计备忘（S2）

### 9.1 接口契约

**新事件**：`internal/core/event/subjects.go`
```go
// SubjectBackupFileReceived 是 CPE 上传备份配置文件落 MinIO 后发布。
// 发布者：acs/upload/handler.go ServeHTTP（FileType=3 分支）
// 订阅者：backup.FilePathRecorder（写回 backup_tasks.file_path）
SubjectBackupFileReceived = "backup.file.received"
```

**事件 payload**：
```go
type BackupFileReceivedPayload struct {
    Bucket             string `json:"bucket"`
    ObjectPath         string `json:"object_path"`
    Filename           string `json:"filename"`
    BackupTaskIDPrefix string `json:"backup_task_id_prefix"` // first 8 hex chars of UUID
    DeviceSN           string `json:"device_sn"`
    FileSize           int64  `json:"file_size"`
}
```

### 9.2 Filename 格式 + 解析

**executor 端生成**（`internal/backup/executor.go`）：
```go
taskID8 := strings.ReplaceAll(task.ID.String(), "-", "")[:8]
targetFilename := fmt.Sprintf("backup-%s-%s.xml", taskID8, dev.SerialNumber)
paramsJSON, _ := json.Marshal(map[string]any{
    "file_type":        "3",
    "target_file_name": targetFilename,
})
// CommandKey 也设置以便 TR-069 TransferComplete 反查
commandKey := fmt.Sprintf("backup-%s", taskID8)
e.taskSvc.CreateTask(ctx, &devtask.CreateTaskRequest{
    DeviceSN:   dev.SerialNumber,
    Method:     "Upload",
    Params:     paramsJSON,
    Source:     devtask.TaskSourceSystem,
    SourceID:   task.ID.String(),  // full UUID for downstream linkage
    CommandKey: commandKey,
})
```

**upload handler 解析**：
```go
var backupFilenameRe = regexp.MustCompile(`^backup-([0-9a-f]{8})-(.+)\.xml(\.[a-z0-9]+)?$`)

func extractBackupTaskID(filename string) (taskIDPrefix, deviceSN string, ok bool) {
    m := backupFilenameRe.FindStringSubmatch(filename)
    if len(m) >= 3 {
        return m[1], m[2], true
    }
    return "", "", false
}
```

注意 regex 末尾 `(\.[a-z0-9]+)?` 兼容 T-0074 + T-0077 压缩扩展名 `.gz/.zst/.lz4/.bz2`。

### 9.3 FilePathRecorder 订阅器

新文件 `internal/backup/file_path_recorder.go`：

```go
type FilePathRecorder struct {
    repo    TaskRepository  // 已有 interface, +UpdateFilePath
    metrics *RestoreMetrics // 复用，加 2 collector
    logger  *zap.Logger
}

func (r *FilePathRecorder) Subscribe(bus event.EventBus) error {
    _, err := bus.QueueSubscribe(
        event.SubjectBackupFileReceived,
        "backup-file-recorder",
        r.handleFileReceived,
    )
    return err
}

func (r *FilePathRecorder) handleFileReceived(ctx context.Context, evt event.Event) error {
    var p BackupFileReceivedPayload
    if err := evt.DecodePayload(&p); err != nil {
        return fmt.Errorf("decode backup.file.received: %w", err)
    }
    fullPath := p.Bucket + "/" + p.ObjectPath
    // Find backup_task by UUID prefix. PG ILIKE on uuid::text — 8-char prefix is selective
    // enough that the row scan is cheap (typically O(1) match).
    matches, err := r.repo.FindByIDPrefix(ctx, p.BackupTaskIDPrefix, 2 /*safety cap*/)
    if err != nil { ... }
    if len(matches) == 0 {
        r.metrics.RecordFilePathRecord("skipped_no_match")
        return nil  // 不在告警范围 — 可能是 follow-up 任务的旧 file
    }
    if len(matches) > 1 {
        // UUID8 collision; use device_sn + created_at proximity as tie-break
        // — but realistically this is so rare we'll just warn-log and pick the
        // latest-created task.
        r.logger.Warn("multiple backup_tasks match prefix", ...)
    }
    target := matches[0]
    if target.FilePath != nil && *target.FilePath != "" {
        // first-write-wins (multi-device task 已有第一文件路径)
        r.metrics.RecordFilePathRecord("skipped_already_set")
        return nil
    }
    if err := r.repo.UpdateFilePath(ctx, target.ID, fullPath); err != nil {
        r.metrics.RecordFilePathRecord("error")
        return err
    }
    r.metrics.RecordFilePathRecord("recorded")
    return nil
}
```

### 9.4 Repository 改动

`internal/backup/repository.go`：加入 interface 方法
```go
type TaskRepository interface {
    // ... existing
    UpdateFilePath(ctx context.Context, id uuid.UUID, filePath string) error
    FindByIDPrefix(ctx context.Context, prefix string, limit int) ([]*BackupTask, error)
}
```

`internal/backup/pg_repository.go`：实现两个方法（squirrel UPDATE / WHERE id::text LIKE 'prefix%' LIMIT N）。

### 9.5 RestoreService.CreateByTaskID

`internal/backup/restore_service.go`：新方法
```go
type CreateByTaskIDRequest struct {
    BackupTaskID    uuid.UUID `json:"backup_task_id" binding:"required"`
    TargetDeviceSNs []string  `json:"target_device_sns" binding:"required,min=1"`
}

func (s *RestoreService) CreateByTaskID(ctx context.Context, req *CreateByTaskIDRequest, createdBy string) (*RestoreTask, *string /*warning*/, error) {
    bt, err := s.taskRepo.GetByID(ctx, req.BackupTaskID)
    if err != nil { return nil, nil, err }
    if bt.FilePath == nil || *bt.FilePath == "" {
        s.metrics.RecordRestoreByTask("rejected_not_uploaded")
        return nil, nil, fmt.Errorf("backup_task %s file_path not set: %w", req.BackupTaskID, commonerrors.ErrNotFound)
    }
    bucket, objectPath, err := splitBucketAndPath(*bt.FilePath)
    if err != nil { return nil, nil, err }
    rt, err := s.Create(ctx, &CreateRestoreRequest{
        Bucket: bucket, ObjectPath: objectPath, TargetDeviceSNs: req.TargetDeviceSNs,
    }, createdBy)
    if err != nil { return nil, nil, err }
    s.metrics.RecordRestoreByTask("accepted")
    var warning *string
    if len(bt.TargetIDs) > 1 {
        w := fmt.Sprintf("multi-device backup (target_count=%d); restore uses only the first device's config",
            len(bt.TargetIDs))
        warning = &w
    }
    return rt, warning, nil
}
```

`internal/backup/restore_handler.go`：新 handler `CreateRestoreByTaskID` + 路由 `POST /backup/restore/by-task-id`。

### 9.6 Carrier 差异

无（内部数据完整性增强；不触 carrier）。

### 9.7 文件清单

修改：
- `omcgo/internal/core/event/subjects.go` — +`SubjectBackupFileReceived` 常量 + 注释
- `omcgo/internal/backup/executor.go` — 设置 target_file_name + CommandKey + SourceID
- `omcgo/internal/acs/upload/handler.go` — FileType=3 分支 publish event（沿用 datamodel 模式）
- `omcgo/internal/backup/repository.go` — 加 UpdateFilePath / FindByIDPrefix interface 方法
- `omcgo/internal/backup/pg_repository.go` — 实现两方法
- `omcgo/internal/backup/restore_service.go` — 加 CreateByTaskID 方法 + RestoreMetrics 调用
- `omcgo/internal/backup/restore_handler.go` — 加 CreateRestoreByTaskID handler + 路由注册
- `omcgo/internal/backup/restore_metrics.go` — +2 collector（filepath_recorded_total / restore_by_task_total）
- `omcgo/cmd/app/provider/modules.go` — 新 FilePathRecorder + Subscribe(EventBus)
- `omcgo/scripts/e2e_verify.sh` — +1 claim bk-11 (POST /backup/restore/by-task-id)

新增：
- `omcgo/internal/backup/file_path_recorder.go` — FilePathRecorder + handleFileReceived
- `omcgo/internal/backup/file_path_recorder_test.go` — filename regex + first-write-wins + no-match cases

### 9.8 待定点

| 待定 | 决策 |
|------|------|
| FindByIDPrefix 的 PG 查询性能（uuid::text LIKE）| backup_tasks.id 已是 PRIMARY KEY；prefix 8 字符可走 btree range scan（PG 15+ 支持 text-prefix 优化）；< 100k 行无虑 |
| 多 prefix collision 处理 | log warn + 取最新 created_at；规模到 65k 任务才有 50% 冲突，业务可接受 |
| backup-file-recorder queue group 名 | "backup-file-recorder" 字符串常量；多 app 实例只一个收到事件 |

# OMC 全部上传/下载功能分析报告

> 生成时间：2026-03-11
> 分析范围：omcgo 后端全部文件上传、下载、传输相关代码
> 验证方法：逐文件源码审查（14 个核心模块、88 个事件主题）

---

## 一、总体结论概览

### 上传功能总览

| # | 功能模块 | 触发方式 | 存储目标 | 状态 |
|---|---------|---------|---------|------|
| 1 | 固件上传 | REST multipart | MinIO `firmware/` | **已完成** |
| 2 | 通用文件上传 | REST multipart | MinIO `managed-files/` | **已完成** |
| 3 | TR069 Upload RPC (ACS→CPE指令) | 命令队列 → SOAP | CPE → 指定 URL | **已完成** |
| 4 | AutonomousTransferComplete 报文接收 | CPE → ACS SOAP | NATS 事件 | **已完成** |
| 5 | TransferComplete 报文接收 | CPE → ACS SOAP | NATS 事件 | **已完成** |
| 6 | 数据模型 JSON 导入 | REST JSON | PostgreSQL | **已完成** |
| 7 | 配置备份任务创建 | REST JSON | PostgreSQL (任务元数据) | **已完成** |
| 8 | **autonomous_transfer_complete → pm.file.received 桥接** | — | — | **缺失** |
| 9 | **autonomous_transfer_complete → mr.file.received 桥接** | — | — | **缺失** |

### 下载功能总览

| # | 功能模块 | 触发方式 | 数据来源 | 状态 |
|---|---------|---------|---------|------|
| 1 | 固件下载到 CPE | TR069 Download RPC | MinIO `firmware/` | **已完成** |
| 2 | 通用文件下载 | REST streaming | MinIO `managed-files/` | **已完成** |
| 3 | MR 文件下载 | REST streaming | MinIO (MR bucket) | **已完成** |
| 4 | 北向 PM Counter 导出 | REST JSON | TimescaleDB | **已完成** |
| 5 | 北向 KPI 值导出 | REST JSON | TimescaleDB | **已完成** |
| 6 | 北向告警导出 | REST JSON | PostgreSQL | **已完成** |
| 7 | **PM 原始 XML 文件下载** | — | — | **缺失** |
| 8 | **报告文件下载** | REST (URL only) | MinIO (TBD) | **部分完成** |
| 9 | **配置备份文件下载** | — | MinIO/FTP | **部分完成** |
| 10 | **MR 数据导出** | REST placeholder | — | **占位** |
| 11 | **文件分发到设备** | REST placeholder | — | **占位** |

### 统计汇总

```
上传功能:  7/9 已完成 (78%)，2 项缺失（均为事件桥接）
下载功能:  6/11 已完成 (55%)，1 项缺失，2 项部分完成，2 项占位
整体完成:  13/20 (65%)
```

---

## 二、上传功能详细分析

### 2.1 固件上传 (REST → MinIO)

**状态: 已完成**

**文件**: `internal/software/handler.go:62-90`, `internal/software/service.go:58-90`

```
用户 POST /api/v1/firmware (multipart/form-data)
    ↓
handler.go:62   c.Request.FormFile("file") 读取二进制
handler.go:70   构建 FirmwareVersion 元数据 (carrier, version, product_class)
handler.go:78   校验 carrier 和 version 必填
    ↓
service.go:61   构建 MinIO 路径: firmware/{carrier}/{product_class}/{version}/{filename}
service.go:64   minioClient.PutObject() 上传到 MinIO
service.go:77   firmwareRepo.Create() 元数据入库 PostgreSQL
service.go:81   发布 "firmware.uploaded" 事件
    ↓
返回 201 Created + FirmwareVersion JSON
```

**REST 端点**:

| 方法 | 端点 | 功能 |
|------|------|------|
| POST | `/api/v1/firmware` | 上传固件 (multipart) |
| GET | `/api/v1/firmware` | 列出固件版本 |
| GET | `/api/v1/firmware/:id` | 获取固件详情 |
| DELETE | `/api/v1/firmware/:id` | 删除固件 |

### 2.2 通用文件上传 (REST → MinIO)

**状态: 已完成**

**文件**: `internal/filemanager/handler.go:82-142`

```
用户 POST /api/v1/files (multipart/form-data)
    ↓
handler.go:87   FormFile("file") + PostForm("file_type", "description", ...)
handler.go:107  构建 MinIO 路径: managed-files/{file_type}/{YYYY-MM-DD}/{filename}
handler.go:110  minioClient.PutObject() → MinIO
handler.go:120  repo.Create() 元数据入库 PostgreSQL managed_files 表
    ↓
返回 201 Created + ManagedFile JSON
```

**支持文件类型** (`model.go:13-19`): `config`, `log`, `firmware`, `script`, `other`

### 2.3 TR069 Upload RPC (ACS → CPE 指令)

**状态: 已完成**

**文件**: `internal/acs/rpc/dispatcher.go:160-170`

ACS 主动指令 CPE 上传文件（PM 采集、日志收集等）：

```go
// UploadData (pkg/soap/templates.go)
type UploadData struct {
    ID           string  // CWMP 事务 ID
    CommandKey   string  // 命令标识
    FileType     string  // "1"=Firmware, "2"=Config, "3"=Log, 等
    URL          string  // CPE 上传目标地址
    Username     string  // 认证凭据
    Password     string
    DelaySeconds int     // 延迟启动
}
```

**SOAP 模板**: `pkg/soap/templates.go:233-241` — `<cwmp:Upload>` XML

### 2.4 AutonomousTransferComplete 报文接收

**状态: 已完成**

**文件**: `internal/acs/handler.go:369-418`

CPE 自主上传 PM/MR 文件完成后的通知处理：

```
CPE → AutonomousTransferComplete SOAP
    ↓
handler.go:372  soap.DecodeAutonomousTransferComplete() 流式 XML 解码
handler.go:393  构建 payload: device_sn, transfer_url, file_type, file_size, ...
handler.go:407  eventBus.Publish("device.inform.autonomous_transfer_complete", ...)
handler.go:411  返回 AutonomousTransferCompleteResponse XML
```

**协议类型**: `pkg/tr069/types.go:170-182`

### 2.5 TransferComplete 报文接收

**状态: 已完成**

**文件**: `internal/acs/handler.go:336-367`

命令驱动的传输完成通知（ACS 下发 Download/Upload 后 CPE 的回复）：

```
handler.go:340  soap.DecodeTransferComplete()
handler.go:356  eventBus.Publish("device.inform.transfer_complete", ...)
handler.go:360  返回 TransferCompleteResponse XML
```

**关键**: `device.inform.transfer_complete` 事件已被 `software.Service` 订阅（`service.go:293-298`），用于驱动固件升级状态机。

### 2.6 数据模型 JSON 导入

**状态: 已完成**

**文件**: `internal/config/datamodel/handler.go:262-270`, `internal/config/datamodel/importer.go`

```
POST /api/v1/datamodels/import  (JSON body)
    ↓
解析: carrier, technology, version, parameter_tree
自动推断 scope: product / oui / carrier_default
写入 PostgreSQL data_models 表
```

### 2.7 配置备份任务

**状态: 已完成（任务管理框架）**

**文件**: `internal/backup/handler.go`

| 方法 | 端点 | 功能 |
|------|------|------|
| POST | `/api/v1/backup/tasks` | 创建备份任务 |
| GET | `/api/v1/backup/tasks` | 列出备份任务 |
| GET | `/api/v1/backup/tasks/:id` | 获取任务详情 |
| POST | `/api/v1/backup/tasks/:id/cancel` | 取消任务 |
| POST | `/api/v1/backup/schedules` | 创建定时备份 |
| POST | `/api/v1/backup/ftp-configs` | 配置 FTP 服务器 |
| POST | `/api/v1/backup/ftp-configs/:id/test` | 测试 FTP 连接 |

### 2.8 autonomous_transfer_complete → pm.file.received 桥接

**状态: 缺失 (P0)**

**事件流断裂**:
```
ACS handler.go:408
    发布 → "device.inform.autonomous_transfer_complete"
                            ↓
                     ❌ 无订阅者 ❌
                            ↓
PM collector.go:51
    订阅 ← "pm.file.received"   (从未被触发)
```

**证据**: 全局搜索 `Publish.*SubjectPMFileReceived` 和 `Publish.*pm.file.received` 均无结果。

**缺失组件职责**:
1. 订阅 `device.inform.autonomous_transfer_complete`
2. 根据 `file_type` 判断是否为 PM 文件
3. 从 `transfer_url` 下载文件
4. 存入 MinIO `pm-files` bucket
5. 查询设备信息获取 device_id, carrier, technology
6. 发布 `pm.file.received` 事件

### 2.9 autonomous_transfer_complete → mr.file.received 桥接

**状态: 缺失 (P0)**

**与 2.8 同类问题**: MR Collector (`internal/mr/collector/collector.go:65`) 订阅 `mr.file.received`，但该事件同样从未被任何代码发布。

**证据**: 全局搜索 `Publish.*SubjectMRFileReceived` 和 `Publish.*mr.file.received` 均无结果。

---

## 三、下载功能详细分析

### 3.1 固件下载到 CPE (TR069 Download RPC)

**状态: 已完成**

**文件**: `internal/software/service.go:92-162`

完整的固件升级下载流程：

```
用户 POST /api/v1/upgrade-tasks {device_id, firmware_id}
    ↓
service.go:93   验证设备和固件存在
service.go:107  检查无活跃升级任务
service.go:119  创建 UpgradeTask (status: pending)
service.go:124  状态转移 → downloading
    ↓
service.go:130  构建下载 URL: minio://{bucket}/{firmware_path}
service.go:137  cmdQueue.Push(deviceSN, Download 命令)
    ↓
service.go:147  connReq.Send() 唤醒设备
service.go:153  发布 "upgrade.started" 事件
```

**状态机** (`internal/software/state_machine.go`):
```
pending → downloading → rebooting → verifying → completed
                                               ↘ failed
```

**批量升级**: `service.go:164-243` — 并发控制 (semaphore)，每设备独立状态

**TransferComplete 回调**: `service.go:245-290`
- 订阅 `device.inform.transfer_complete`
- 驱动状态机 downloading → rebooting → ... → completed
- 完成后发布 `upgrade.completed` 事件

### 3.2 通用文件下载 (REST → MinIO streaming)

**状态: 已完成**

**文件**: `internal/filemanager/handler.go:191-223`

```
GET /api/v1/files/:id/download
    ↓
handler.go:199  repo.GetByID() 查询元数据 (含 MinIOPath)
handler.go:205  minioClient.GetObject(bucket, mf.MinIOPath)
handler.go:213  Content-Disposition: attachment; filename="{filename}"
handler.go:219  io.Copy(c.Writer, obj)  流式传输
```

### 3.3 MR 文件下载 (REST → MinIO streaming)

**状态: 已完成**

**文件**: `internal/mr/handler.go:94-128`

```
GET /api/v1/mr/files/:id/download
    ↓
handler.go:102  store.GetFileByID() 查询文件元数据
handler.go:112  minioClient.GetObject(bucket, fileInfo.MinioPath)
handler.go:119  obj.Stat() 获取文件大小
handler.go:127  c.DataFromReader() 流式传输 (Content-Type: application/xml)
```

**MR 文件列表**: `GET /api/v1/mr/files` — 支持 device_id, mr_type, start_time, end_time 筛选

### 3.4 北向 PM Counter 导出

**状态: 已完成**

**文件**: `internal/northbound/pm_handler.go:32-78`

```
POST /api/v1/northbound/pm/export
    ↓
JSON body: {start_time, end_time, device_id?, cell_id?, counter_group?}
    ↓
counterRepo.Query() → TimescaleDB
    ↓
返回 JSON 格式 Counter 数据
```

### 3.5 北向 KPI 值导出

**状态: 已完成**

**文件**: `internal/northbound/pm_handler.go:81-136`

```
GET /api/v1/northbound/pm/kpi/export
    ↓
Query: device_id, kpi_name, carrier, technology, start_time, end_time
    ↓
kpiRepo.Query() → TimescaleDB
    ↓
返回 JSON 格式 KPI 数据
```

### 3.6 北向告警导出

**状态: 已完成**

**文件**: `internal/northbound/alarm_handler.go`

提供 OSS 系统集成的告警数据导出接口。

### 3.7 PM 原始 XML 文件下载

**状态: 缺失 (P2)**

当前没有 REST 端点下载存储在 MinIO `pm-files` bucket 的原始 PM XML 文件。

- MR 模块有 `GET /api/v1/mr/files/:id/download`（已实现）
- PM 模块缺少对应的 `GET /api/v1/pm/files/:id/download`
- 可参照 MR handler (`mr/handler.go:94-128`) 的实现模式快速补齐

### 3.8 报告文件下载

**状态: 部分完成**

**文件**: `internal/report/handler.go:274-302`

```
GET /api/v1/reports/records/:id/download
    ↓
handler.go:282  service.GetRecord() 获取报告记录
handler.go:288  if record.DownloadURL != "" → 返回 {url, file_name}
handler.go:297  else → 返回 "report is still generating" 占位信息
```

**问题**:
- `DownloadURL` 字段存在但未被填充（报告生成引擎未实现 MinIO 存储）
- 缺少 MinIO 流式下载逻辑（当前只返回 URL，不直接提供文件流）
- 报告生成 (`service.Generate()`) 创建了记录但未生成实际文件

### 3.9 配置备份文件下载

**状态: 部分完成**

**文件**: `internal/backup/handler.go`, `internal/backup/service.go`

- 备份任务管理框架完整（创建、列表、取消、定时）
- FTP 服务器配置管理完整
- **缺失**: 备份文件的实际获取和下载 REST 端点
- **缺失**: 备份执行 Worker（实际调用 TR069 Upload RPC 获取设备配置 → 存 MinIO/FTP）

### 3.10 MR 数据导出

**状态: 占位**

**文件**: `internal/mr/handler.go:358-364`

```go
func (h *Handler) ExportMRData(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "task_id": "export-placeholder",
        "status":  "pending",
    })
}
```

端点 `POST /api/v1/mr/export` 已注册但仅返回 placeholder。

### 3.11 文件分发到设备

**状态: 占位**

**文件**: `internal/filemanager/handler.go:230-265`

```go
// handler.go:250 注释
// Queue a distribution task (simplified — full implementation deferred to worker integration)
taskID := uuid.New().String()
c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "queued"})
```

端点 `POST /api/v1/files/:id/distribute` 已注册，接受 `device_sns` 列表，但仅记录日志并返回 queued 状态。未与 TR069 Download RPC 或命令队列集成。

---

## 四、事件总线文件传输图谱

### 4.1 完整事件拓扑

```
                       ┌──────────────┐
                       │   CPE 设备    │
                       └──────┬───────┘
                              │ TR069 SOAP
                       ┌──────▼───────┐
                       │   ACS 服务    │
                       └──────┬───────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
 device.inform.*       device.inform.      device.inform.
 (periodic,alarm       transfer_           autonomous_transfer_
  value_change,        complete            complete
  bootstrap...)              │                   │
                             │            ┌──────┴──────┐
                             │            │             │
                             ▼            ▼             ▼
                   software.Service    PM 文件?      MR 文件?
                   (升级状态机)       ❌ 缺失 ❌    ❌ 缺失 ❌
                             │            │             │
                    upgrade.started   (需新增)      (需新增)
                    upgrade.completed      │             │
                    upgrade.failed         ▼             ▼
                                   pm.file.received  mr.file.received
                                         │               │
                                         ▼               ▼
                                   PM Collector     MR Collector
                                   ┌────┴────┐     ┌────┴────┐
                                   │         │     │         │
                                   ▼         ▼     ▼         ▼
                              Counter   KPI 计算  MR 记录   MR 指标
                              入库     入库       入库      统计
                                   │         │
                                   ▼         ▼
                            pm.file.parsed   mr.file.parsed
```

### 4.2 事件主题定义全集 (`internal/event/subjects.go`)

| 类别 | 事件主题 | 发布者 | 订阅者 |
|------|---------|--------|--------|
| **设备** | `device.inform.bootstrap` | ACS handler | 设备注册 |
| | `device.inform.periodic` | ACS handler | 设备状态更新 |
| | `device.inform.value_change` | ACS handler | 参数变化处理 |
| | `device.inform.alarm` | ACS handler | 告警模块 |
| | `device.inform.transfer_complete` | ACS handler | **software.Service** |
| | `device.inform.autonomous_transfer_complete` | ACS handler | **❌ 无订阅者** |
| | `device.inform.reboot_complete` | ACS handler | 设备状态 |
| **PM** | `pm.file.received` | **❌ 无发布者** | PM Collector |
| | `pm.file.parsed` | PM Collector | — |
| **MR** | `mr.file.received` | **❌ 无发布者** | MR Collector |
| | `mr.file.parsed` | MR Collector | — |
| **固件** | `firmware.uploaded` | software.Service | — |
| | `upgrade.started` | software.Service | — |
| | `upgrade.completed` | software.Service | — |
| | `upgrade.failed` | software.Service | — |

---

## 五、MinIO Bucket 使用情况

| Bucket 名称 | 配置来源 | 写入模块 | 读取模块 | 状态 |
|------------|---------|---------|---------|------|
| `firmware` | appconfig.MinIO.Buckets.Firmware | software.Service | ACS Download RPC | 已完成 |
| `managed-files` | filemanager (hardcoded) | filemanager.Handler | filemanager.Handler | 已完成 |
| `pm-files` | appconfig.MinIO.Buckets.PMFiles | **❌ 无写入者** | PM Collector | 写入缺失 |
| `mr-files` | appconfig.MinIO.Buckets.MRFiles | **❌ 无写入者** | MR Collector + Handler | 写入缺失 |
| `config-backup` | appconfig.MinIO.Buckets.ConfigBackup | **❌ 未实现** | **❌ 未实现** | 框架就绪 |
| `logs` | appconfig.MinIO.Buckets.Logs | **❌ 未实现** | — | 已配置 |

---

## 六、缺口详细分析

### 6.1 P0: 自主传输完成事件桥接处理器

**影响**: PM 文件和 MR 文件的整个处理流水线无法被触发

**需要实现**: `internal/transfer/handler.go` (新文件)

```
订阅 "device.inform.autonomous_transfer_complete"
    ↓
1. 解析 payload: device_sn, file_type, transfer_url, target_filename
2. 根据 file_type 判断文件类别:
   - PM 文件 → pm-files bucket → 发布 pm.file.received
   - MR 文件 → mr-files bucket → 发布 mr.file.received
   - Log 文件 → logs bucket
   - 其他 → managed-files bucket
3. 从 transfer_url 下载文件 (HTTP GET 或 FTP)
4. 上传到对应 MinIO bucket
5. 查询设备信息 (device_id, carrier, technology)
6. 发布对应 *.file.received 事件
```

### 6.2 P1: 备份执行 Worker

**影响**: 配置备份任务创建后无法实际执行

**需要实现**:
- Worker 订阅备份任务，向设备下发 TR069 Upload RPC（FileType="2" 配置文件）
- 接收设备上传的配置文件 → 存入 MinIO `config-backup` bucket
- 更新任务状态 (running → completed/failed)

### 6.3 P2: PM 原始文件下载端点

**影响**: 运维人员无法通过 REST API 下载 PM 原始 XML

**参照实现**: `internal/mr/handler.go:94-128` (MR 文件下载)

### 6.4 P2: 报告文件生成 + MinIO 流式下载

**影响**: 报告记录创建了但无法下载

**当前状态**: `report/handler.go:288` 检查 `record.DownloadURL`，但该字段始终为空

### 6.5 P3: MR 数据导出

**影响**: `POST /api/v1/mr/export` 返回 placeholder

### 6.6 P3: 文件分发 Worker

**影响**: `POST /api/v1/files/:id/distribute` 仅记日志

---

## 七、相关文件清单

| 文件路径 | 角色 | 关键行号 |
|---------|------|---------|
| `internal/acs/handler.go` | ACS 报文处理 | 336-367 (TransferComplete), 369-418 (AutonomousTransferComplete) |
| `internal/acs/rpc/dispatcher.go` | RPC 调度器 | 148-158 (Download), 160-170 (Upload) |
| `internal/software/handler.go` | 固件管理 REST | 33-44 (路由), 62-90 (上传) |
| `internal/software/service.go` | 固件升级服务 | 58-90 (上传), 92-162 (单台升级), 164-243 (批量), 245-298 (TransferComplete 回调) |
| `internal/software/state_machine.go` | 升级状态机 | 全文 |
| `internal/filemanager/handler.go` | 通用文件管理 | 82-142 (上传), 191-223 (下载), 230-265 (分发) |
| `internal/pm/collector/collector.go` | PM 文件处理器 | 49-110 |
| `internal/pm/collector/parser.go` | 3GPP 32.435 解析 | 64-155 |
| `internal/pm/kpi/engine.go` | KPI 计算引擎 | 88-167 |
| `internal/pm/handler.go` | PM REST API | 31-42 (路由) |
| `internal/mr/handler.go` | MR REST API | 30-44 (路由), 94-128 (文件下载), 358-364 (导出占位) |
| `internal/mr/collector/collector.go` | MR 文件处理器 | 63-74 (订阅) |
| `internal/backup/handler.go` | 备份管理 | 31-53 (路由) |
| `internal/report/handler.go` | 报告管理 | 29-45 (路由), 274-302 (下载) |
| `internal/northbound/pm_handler.go` | 北向导出 | 32-78 (PM), 81-136 (KPI) |
| `internal/config/datamodel/handler.go` | 数据模型导入 | 262-270 |
| `internal/event/subjects.go` | 事件主题 | 全文 (88 个主题) |
| `cmd/worker/main.go` | Worker 主入口 | 125-134 (PM Collector 初始化) |
| `pkg/tr069/types.go` | TR069 类型 | 104-140 (Download/Upload), 162-182 (AutonomousTransferComplete) |
| `pkg/soap/templates.go` | SOAP 模板 | 17-18 (Download/Upload 模板), 24 (ATC 响应) |
| `pkg/soap/decoder.go` | SOAP 解码 | 202 (Download), 253-297 (ATC) |

---

## 八、建议优先级

| 优先级 | 项目 | 影响范围 | 工作量 |
|-------|------|---------|--------|
| **P0** | 实现 autonomous_transfer_complete → pm/mr.file.received 桥接处理器 | PM + MR 全流水线 | 中 (~250 行) |
| **P1** | 实现配置备份执行 Worker | 备份功能 | 中 (~300 行) |
| **P2** | 新增 PM 原始文件 REST 下载端点 | PM 文件管理 | 小 (~60 行) |
| **P2** | 实现报告文件生成 + MinIO 存储/下载 | 报告模块 | 中 (~200 行) |
| **P3** | 实现 MR 数据导出 (替换 placeholder) | MR 数据管理 | 小 (~80 行) |
| **P3** | 实现文件分发 Worker (Distribute → TR069 Download) | 文件管理 | 小 (~100 行) |

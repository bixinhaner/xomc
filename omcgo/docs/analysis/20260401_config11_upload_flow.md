# FileType=11 配置文件上传完整流程分析

**分析日期**：2026-04-01  
**触发场景**：通过 `rpctool upload config-11` 下发 Upload RPC，让 CPE 上传 `11 Configuration File` 到 ACS

---

## 整体流程图

```
CPE                    ACS handler.go            upload/handler.go       provision/engine.go
 │                          │                          │                       │
 │── Inform ──────────────→ │ handleInform             │                       │
 │← InformResponse ─────────│                          │                       │
 │                          │                          │                       │
 │── Empty POST ──────────→ │ handleEmpty L494         │                       │
 │                          │── PopTask L556 ──→ Redis │                       │
 │                          │←── Upload Task ──────────│                       │
 │← Upload SOAP ────────────│ BuildRequest L210        │                       │
 │                          │                          │                       │
 │── HTTP PUT/POST ─────────────────────────────────→  │ ServeHTTP L64        │
 │   (上传 xml.gz 到 ACS)   │                    PutObject → MinIO             │
 │                          │                    publishDataModelEvent L154    │
 │                          │                    Publish("datamodel.file.received") L231
 │                          │                          │  ─────────────────→  │ handleDataModelFileReceived L473
 │                          │                          │                  GetObject(MinIO) L142
 │                          │                          │                  ImportFromXMLForCPE L153
 │                          │                          │                  → data_model_definitions 表
 │                          │                          │                  StartTwoPhaseSync L520
 │← UploadResponse ─────────│                          │                       │
 │── TransferComplete ─────→│ handleTransferComplete L1095                     │
 │← TransferCompleteResp ───│                          │                       │
```

---

## 各步骤关键代码位置

### 步骤 1：CPE 发送空报文（Empty POST）

**触发条件**：CPE 收到 `InformResponse` 后，按 TR-069 规范发送 body 为空的 HTTP POST。

| 位置 | 说明 |
|------|------|
| `internal/acs/handler.go L183` | `ServeHTTP` 入口，读取 body，body 为空时路由到 `handleEmpty` |

---

### 步骤 2：ACS 检查 RPC 任务队列，弹出 Upload 任务

| 位置 | 说明 |
|------|------|
| `internal/acs/handler.go L494` | `handleEmpty` 函数入口 |
| `internal/acs/handler.go L500` | 从 Cookie 取会话，获取 `deviceSN` |
| `internal/acs/handler.go L541` | 检查单会话 RPC 上限（`sessionRPCLimitReached`） |
| `internal/acs/handler.go L556` | `h.taskService.PopTask(ctx, deviceSN)` — 从 Redis 弹出待执行任务 |
| `internal/acs/handler.go L565` | `MarkTaskSent(taskItem.ID, cwmpID)` — 标记任务已下发 |
| `internal/acs/handler.go L577` | 构造 `cmdqueue.Command`，调用 `rpcDispatcher.BuildRequest` |

---

### 步骤 3：ACS 构建 Upload SOAP 报文返回给 CPE

| 位置 | 说明 |
|------|------|
| `internal/acs/rpc/dispatcher.go L52` | `d.Register("Upload", &UploadHandler{})` — 注册 Upload 处理器 |
| `internal/acs/rpc/dispatcher.go L210` | `UploadHandler.BuildRequest`：反序列化 params（`file_type`、`url`、`username`、`password`） |
| `internal/acs/rpc/dispatcher.go L217` | `soap.RenderResponse(soap.UploadTmpl, params)` — 渲染 SOAP XML |
| `pkg/soap/templates.go L257` | `uploadXML` 模板，含 `<FileType>`、`<URL>`、`<Username>`、`<Password>`、`<DelaySeconds>` |

**生成的 SOAP 报文示例**：
```xml
<cwmp:Upload>
  <CommandKey>upload-xxx</CommandKey>
  <FileType>11 Configuration File</FileType>
  <URL>http://localhost:8080/smallcell/FileUploadService?fileType=11&amp;filename=1202000588233HB0039_20260401_063000.xml.gz</URL>
  <Username></Username>
  <Password></Password>
  <DelaySeconds>0</DelaySeconds>
</cwmp:Upload>
```

---

### 步骤 4/5：CPE 解析 FileType=11，执行文件上传

CPE 收到 Upload RPC 后，将本地配置文件以 HTTP PUT/POST 上传到 `<URL>` 指定的地址。

**ACS 文件接收端**：

| 位置 | 说明 |
|------|------|
| `internal/acs/upload/handler.go L64` | `ServeHTTP` 入口，校验 Method（支持 PUT/POST） |
| `internal/acs/upload/handler.go L72-90` | Basic Auth 认证 |
| `internal/acs/upload/handler.go L93-94` | 从 query param 取 `fileType`（"11"）和 `filename` |
| `internal/acs/upload/handler.go L120` | `normalizeFileType("11")` → `FileTypeDataModel` |
| `internal/acs/upload/handler.go L121` | `BucketAndCategory(ft)` → bucket=`omc-exchange`, category=`datamodel` |
| `internal/acs/upload/handler.go L132` | `minioClient.PutObject` — 流式写入 MinIO |
| `internal/core/storage/router.go L34` | `FileTypeDataModel` 路由：`buckets.Exchange, "datamodel"` |

**MinIO 存储路径格式**：`datamodel/2026/04/01/{filename}`

---

### 步骤 6：上传成功，ACS 发布 `datamodel.file.received` 事件

| 位置 | 说明 |
|------|------|
| `internal/acs/upload/handler.go L153` | `ft == FileTypeDataModel` 时触发事件发布 |
| `internal/acs/upload/handler.go L154` | 调用 `publishDataModelEvent(ctx, bucket, objectPath, filename, size)` |
| `internal/acs/upload/handler.go L212` | `publishDataModelEvent` 函数入口 |
| `internal/acs/upload/handler.go L216` | `extractDeviceSNFromFilename(filename)` — 从文件名提取 deviceSN |
| `internal/acs/upload/handler.go L231` | `eventBus.Publish(ctx, "datamodel.file.received", evt)` |
| `internal/core/event/subjects.go L60` | `SubjectDataModelFileReceived = "datamodel.file.received"` |

**事件 payload 字段**：

```json
{
  "minio_bucket": "omc-exchange",
  "minio_path":   "datamodel/2026/04/01/xxx.xml.gz",
  "device_sn":    "1202000588233HB0039",
  "file_size":    12345,
  "filename":     "xxx.xml.gz"
}
```

---

### 步骤 7：订阅事件

**订阅注册发生在 `app` 服务**，不是 `acs` 服务。

#### 注册入口（app 服务启动时）

| 位置 | 说明 |
|------|------|
| `cmd/app/router/router.go L163-L165` | 创建 `ProvisioningEngine` 实例，传入 `eventBus` |
| `cmd/app/router/router.go L169-L177` | 若 `cfg.Provision.ModelUpload.Enabled`，创建 `ModelUploadService` 并挂载到引擎 |
| `cmd/app/router/router.go L187` | **`provisionEngine.Subscribe(eventBus)`** — 触发所有订阅注册 |

#### 订阅方法内部实现

| 位置 | 说明 |
|------|------|
| `internal/provision/engine.go L85` | `func (e *ProvisioningEngine) Subscribe(bus event.EventBus) error` 入口 |
| `internal/provision/engine.go L105` | `bus.QueueSubscribe("datamodel.file.received", "provision-model-upload", handleDataModelFileReceived)` |

#### 架构归属说明

```
cmd/
├── acs/        ← ACS 服务：接收 CPE TR-069 报文，发布事件，不订阅 datamodel 事件
├── app/        ← App 服务：订阅 datamodel.file.received，处理参数树写入  ← 本流程发生在此
└── worker/     ← Worker 服务：订阅 PM/MR/Alarm 等事件（与本流程无关）
```

> `worker` 服务虽然也注册了大量订阅（`cmd/worker/main.go L69`），但 **`datamodel.file.received` 事件的唯一订阅者是 `app` 服务的 `ProvisioningEngine`**，worker 中的 `TransferBridge`（`L113`）只处理 `device.inform.autonomous_transfer_complete` 事件（PM/MR 主动上传场景）。

---

### 步骤 8：下载文件 → 解析 XML → 存入参数树

**事件处理函数**：

| 位置 | 说明 |
|------|------|
| `internal/provision/engine.go L473` | `handleDataModelFileReceived` 入口，解码 payload |
| `internal/provision/engine.go L492` | `deviceService.GetBySerialNumber(ctx, deviceSN)` — 查找设备 |
| `internal/provision/engine.go L507` | `modelUploadService.HandleModelFileReceived(ctx, dev, payload)` |
| `internal/provision/model_upload.go L121` | `HandleModelFileReceived` 入口 |
| `internal/provision/model_upload.go L130-138` | 查询或创建 `parameter_discovery_log` 记录 |
| `internal/provision/model_upload.go L142` | `minioClient.GetObject(bucket, path)` — 从 MinIO 下载 XML |
| `internal/provision/model_upload.go L153` | `dmImporter.ImportFromXMLForCPE(ctx, reader, carrier, productClass, firmwareVersion)` — 解析 XML，写入 `data_model_definitions` 表 |
| `internal/provision/model_upload.go L163-170` | 更新 `parameter_discovery_log` 状态为 `completed` |
| `internal/provision/model_upload.go L174` | `dmRegistry.InvalidateCache(ctx, dm)` — 失效数据模型缓存 |
| `internal/provision/engine.go L519-530` | 若 `AutoSync.Enabled`，触发 `syncService.StartTwoPhaseSync` — 通过 GPV/GPN 将参数值同步到 `device_parameters` 表 |

---

### CPE 上报 TransferComplete（异步）

CPE 文件上传完毕后，在同一会话内上报 `TransferComplete`。

| 位置 | 说明 |
|------|------|
| `internal/acs/handler.go L1095` | `handleTransferComplete` 入口 |
| `internal/acs/handler.go L1102` | `soap.DecodeTransferComplete` — 解码报文，获取 `CommandKey` |
| `internal/acs/handler.go L1130-1131` | `eventBus.Publish(SubjectDeviceTransferComplete, tc)` |
| `internal/acs/handler.go L1134` | 发送 `TransferCompleteResponse` 给 CPE |

> **注意**：`TransferComplete` 事件目前由 `software.Service` 订阅处理固件升级场景，配置文件上传场景暂无独立订阅者处理此事件。

---

## 关键事件主题一览

| 事件主题 | 发布者 | 订阅者 |
|---------|--------|--------|
| `datamodel.file.received` | `upload/handler.go L231` | `provision/engine.go L105` |
| `device.inform.transfer_complete` | `handler.go L1131` | `software.Service`（固件升级） |
| `device.inform.autonomous_transfer_complete` | `handler.go L1207` | `transfer/bridge.go L55`（PM/MR） |

---

## 数据库写入表

| 表名 | 写入时机 | 位置 |
|------|---------|------|
| `parameter_discovery_log` | 开始处理/完成/失败 | `model_upload.go L135, L167` |
| `data_model_definitions` | XML 解析成功后 | `dmImporter.ImportFromXMLForCPE L153` |
| `device_parameters` | AutoSync 完成后 | `syncService.StartTwoPhaseSync` |

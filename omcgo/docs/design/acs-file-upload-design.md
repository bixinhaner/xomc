# ACS 文件上传功能设计文档

> 创建时间: 2026-03-19
> 状态: 待确认
> 更新时间: 2026-03-19

---

## 0. 推荐方案：MinIO Presigned URL（零额外开发）

经过深入分析，**推荐使用 MinIO Presigned PUT URL 方案**，无需开发独立的文件接收服务器。

### 0.1 方案对比

| 方案 | 开发工作量 | 性能 | 安全性 | 网络要求 |
|------|-----------|------|--------|----------|
| **MinIO Presigned URL** | ⭐ 零开发 | ⭐⭐⭐ 直传 | ⭐⭐⭐ 签名验证 | CPE 能访问 MinIO |
| ACS 代理服务器 | ⭐⭐⭐ 中等 | ⭐⭐ 代理转发 | ⭐⭐⭐ | 无特殊要求 |

### 0.2 MinIO Presigned URL 工作原理

```
┌──────────┐                              ┌──────────┐
│   ACS    │                              │   CPE    │
│          │  1. Upload RPC               │          │
│          │   (含 Presigned URL)         │          │
│          │ ─────────────────────────────→│          │
│          │                              │          │
│ 生成:    │                              │          │
│ minio.   │                              │          │
│ Presigned│                              │          │
│ PutObject│                              │          │
│          │                              │          │
└──────────┘                              └──────────┘
     │                                         │
     │    2. HTTP PUT (直接到 MinIO)           │
     │   ◄─────────────────────────────────────┤
     │                                         │
┌──────────┐                              ┌──────────┐
│  MinIO   │ ◄── 文件存储                   │          │
│  :9000   │                              │          │
└──────────┘                              └──────────┘
     │                                         │
     │    3. TransferComplete                  │
     │   ◄─────────────────────────────────────┤
```

### 0.3 代码实现（极简）

```go
// internal/acs/upload/presigned.go

package upload

import (
    "context"
    "fmt"
    "time"

    "github.com/minio/minio-go/v7"
)

// PresignedUploader 生成 MinIO 预签名上传 URL
type PresignedUploader struct {
    minioClient *minio.Client
    pmBucket    string
    mrBucket    string
    logBucket   string
    urlTTL      time.Duration // URL 有效期
    // 可选：对外暴露的 MinIO 地址（如果与内部地址不同）
    externalEndpoint string
}

func NewPresignedUploader(minioClient *minio.Client, buckets BucketConfig, urlTTL time.Duration) *PresignedUploader {
    return &PresignedUploader{
        minioClient: minioClient,
        pmBucket:    buckets.PMFiles,
        mrBucket:    buckets.MRFiles,
        logBucket:   buckets.Logs,
        urlTTL:      urlTTL,
    }
}

// GenerateUploadURL 生成上传 URL
// fileType: "4"=PM, "5"=MR, "6"=Log
// 返回的 URL 可直接用于 HTTP PUT 上传
func (u *PresignedUploader) GenerateUploadURL(ctx context.Context, deviceSN, fileType, filename string) (string, string, error) {
    // 1. 确定目标 bucket
    bucket := u.bucketForFileType(fileType)

    // 2. 生成对象路径（按日期/设备组织）
    objectPath := u.objectPath(deviceSN, filename)

    // 3. 生成预签名 PUT URL
    url, err := u.minioClient.PresignedPutObject(ctx, bucket, objectPath, u.urlTTL)
    if err != nil {
        return "", "", fmt.Errorf("generate presigned URL: %w", err)
    }

    // 4. 如果配置了外部地址，替换 URL 中的 host
    uploadURL := url.String()
    if u.externalEndpoint != "" {
        // 替换内部地址为外部可访问地址
        uploadURL = strings.Replace(uploadURL, u.minioClient.EndpointURL().Host, u.externalEndpoint, 1)
    }

    return uploadURL, objectPath, nil
}

func (u *PresignedUploader) bucketForFileType(fileType string) string {
    switch fileType {
    case "4": // Vendor Configuration File / PM
        return u.pmBucket
    case "5": // Log File / MR
        return u.mrBucket
    case "6": // Log
        return u.logBucket
    default:
        return u.pmBucket
    }
}

func (u *PresignedUploader) objectPath(deviceSN, filename string) string {
    now := time.Now()
    return fmt.Sprintf("%s/%s/%s/%s",
        now.Format("2006/01/02"),
        deviceSN,
        filename,
    )
}
```

### 0.4 集成到 Upload RPC

```go
// internal/acs/rpc/dispatcher.go - 修改 UploadHandler

func (h *UploadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
    var params soap.UploadData
    if err := json.Unmarshal(cmd.Params, &params); err != nil {
        return nil, fmt.Errorf("parse Upload params: %w", err)
    }

    // 新增：如果没有提供 URL，生成 Presigned URL
    if params.URL == "" {
        uploadURL, objectPath, err := h.uploader.GenerateUploadURL(
            context.Background(),
            cmd.DeviceSN,
            params.FileType,
            params.TargetFileName,
        )
        if err != nil {
            return nil, fmt.Errorf("generate upload URL: %w", err)
        }
        params.URL = uploadURL
        // 记录 objectPath 用于后续 TC 处理
        cmd.Metadata["object_path"] = objectPath
    }

    params.ID = cmd.CommandKey
    params.CommandKey = cmd.CommandKey
    return soap.RenderResponse(soap.UploadTmpl, params)
}
```

### 0.5 配置项

```yaml
# config.yaml
minio:
  endpoint: "minio:9000"           # 内部地址
  external_endpoint: "minio.example.com:9000"  # CPE 可访问的外部地址（可选）
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  upload_url_ttl: 1h               # 预签名 URL 有效期
  buckets:
    pm_files: "pm-files"
    mr_files: "mr-files"
    logs: "logs"
```

### 0.6 网络要求

**关键前提**：CPE 设备需要能访问 MinIO 的网络地址

| 部署场景 | 网络配置 |
|---------|---------|
| CPE 与 ACS 同内网 | MinIO 内网地址直接可达 |
| CPE 在运营商网络 | MinIO 需要公网 IP 或专线 |
| CPE 在 NAT 后 | MinIO 需要 NAT 映射或使用 ACS 代理方案 |

**网络不可达时的备选方案**：使用 ACS 代理服务器（见第 3 节）

### 0.7 设备兼容性

| 要求 | TR069 规范 | 实际支持 |
|------|-----------|---------|
| HTTP PUT | TR-069 要求支持 | 大多数设备支持 |
| URL 长度 | 无限制 | Presigned URL 约 300-500 字符，通常无问题 |
| HTTPS | 可选 | MinIO 可配置 HTTPS |

---

## 1. 需求概述

实现 ACS 服务的文件上传功能，支持两种模式：

### 模式一：ACS 主动触发上传

```
┌─────┐      1. Upload RPC       ┌─────┐
│ ACS │ ───────────────────────► │ CPE │
│     │                          │     │
│     │      2. POST File        │     │
│     │ ◄─────────────────────── │     │
│     │                          │     │
│     │   3. TransferComplete    │     │
│     │ ◄─────────────────────── │     │
└─────┘                          └─────┘
```

1. ACS 发送 `Upload` RPC 给设备
2. 设备上传文件（HTTP POST）到指定的 URL
3. 设备发送 `TransferComplete` 通知 ACS

### 模式二：设备自主上传

```
┌─────┐                          ┌─────┐
│ ACS │                          │ CPE │
│     │      1. POST File        │     │
│     │ ◄─────────────────────── │     │
│     │                          │     │
│     │ 2. AutonomousTransfer    │     │
│     │    Complete              │     │
│     │ ◄─────────────────────── │     │
└─────┘                          └─────┘
```

1. 设备根据自己的配置，主动将文件上传到预配置的 URL
2. 文件上传成功后，设备发送 `AutonomousTransferComplete` 通知 ACS

---

## 2. 当前实现状态分析

### 2.1 已实现功能 ✅

| 组件 | 文件 | 状态 | 说明 |
|------|------|------|------|
| Upload RPC 构建 | `internal/acs/rpc/dispatcher.go` | ✅ 已实现 | `UploadHandler.BuildRequest()` 可生成 Upload RPC |
| TransferComplete 处理 | `internal/acs/handler.go:361-391` | ✅ 已实现 | `handleTransferComplete()` 处理设备响应 |
| AutonomousTransferComplete 处理 | `internal/acs/handler.go:393-442` | ✅ 已实现 | `handleAutonomousTransferComplete()` 处理设备自主上传通知 |
| UploadResponse 处理 | `internal/acs/handler.go` | ✅ 已实现 | 在 `handleRPCResponse()` 中处理 |
| ATC 事件发布 | `internal/acs/handler.go` | ✅ 已实现 | 发布 `SubjectDeviceAutonomousTransferComplete` 事件 |
| TransferURL 模式下载 | `internal/transfer/bridge.go` | ✅ 已实现 | 从 TransferURL 下载文件到 MinIO |

### 2.2 未实现功能 ❌

#### MinIO Presigned URL 方案（推荐）

| 组件 | 说明 | 优先级 |
|------|------|--------|
| **PresignedUploader** | 生成 MinIO 预签名上传 URL | 🔴 高 |
| **external_endpoint 配置** | MinIO 外部访问地址配置 | 🟡 中 |
| **Upload RPC 集成** | 自动生成 Presigned URL | 🔴 高 |

#### ACS 代理方案（备选，网络隔离场景）

| 组件 | 说明 | 优先级 |
|------|------|--------|
| **ACS 文件服务器端点** | 接收设备 POST 上传的 HTTP 端点 | 🔴 高 |
| **Token 管理** | 上传 Token 生成与验证 | 🟡 中 |
| **文件接收与存储** | 接收上传文件并存储到 MinIO | 🔴 高 |
| **上传会话管理** | 关联上传文件与后续的 TC/ATC 消息 | 🟡 中 |

### 2.3 代码分析

#### 2.3.1 Upload RPC 构建（已实现）

```go
// internal/acs/rpc/dispatcher.go
type UploadHandler struct{}

func (h *UploadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
    var params soap.UploadData
    if err := json.Unmarshal(cmd.Params, &params); err != nil {
        return nil, fmt.Errorf("parse Upload params: %w", err)
    }
    params.ID = cmd.CommandKey
    params.CommandKey = cmd.CommandKey
    return soap.RenderResponse(soap.UploadTmpl, params)
}
```

**问题**: 当前 `UploadData` 中的 URL 需要指向 ACS 文件服务器，但该服务器尚未实现。

#### 2.3.2 AutonomousTransferComplete 处理（已实现）

```go
// internal/acs/handler.go:393-442
func (s *Session) handleAutonomousTransferComplete(env *soap.Envelope) error {
    atc := env.Body.AutonomousTransferComplete
    // ... 解析 ATC 消息 ...

    // 发布事件
    payload := map[string]interface{}{
        "device_sn":       deviceSN,
        "announce_url":    atc.AnnounceURL,
        "transfer_url":    atc.TransferURL,
        "is_download":     atc.IsDownload,
        "file_type":       atc.FileType,
        "file_size":       atc.FileSize,
        "target_filename": atc.TargetFileName,
        // ...
    }
    return s.eventBus.Publish(ctx, event.SubjectDeviceAutonomousTransferComplete, evt)
}
```

**问题**: 当前 `TransferURL` 期望是设备提供的 URL（ACS 下载模式），而非 ACS 文件服务器的 URL。

#### 2.3.3 TransferBridge（仅支持下载模式）

```go
// internal/transfer/bridge.go
func (b *TransferBridge) downloadAndStore(ctx context.Context, url, bucket, objectPath string) (int64, error) {
    // HTTP GET 从 TransferURL 下载
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    // ...
    resp, err := b.httpClient.Do(req)
    // ...
    // 存储到 MinIO
    info, err := b.minioClient.PutObject(ctx, bucket, objectPath, resp.Body, -1, ...)
    // ...
}
```

**问题**: 只支持 HTTP GET 下载模式，不支持接收设备 POST 上传的文件。

---

## 3. 技术方案

### 3.1 架构设计

```
                    ┌──────────────────────────────────────┐
                    │            ACS Server                │
                    │                                      │
    HTTP POST ─────►│  /upload/{device_sn}/{token}         │
    (File Upload)   │       │                              │
                    │       ▼                              │
                    │  ┌─────────────┐                     │
                    │  │ UploadServer│ ──────► MinIO       │
                    │  └─────────────┘                     │
                    │                                      │
    SOAP ──────────►│  /acs                                │
    (TC/ATC)        │       │                              │
                    │       ▼                              │
                    │  ┌─────────────┐                     │
                    │  │ ACSSession  │ ──────► EventBus    │
                    │  └─────────────┘                     │
                    │                                      │
                    └──────────────────────────────────────┘
```

### 3.2 组件设计

#### 3.2.1 UploadServer（新组件）

**位置**: `internal/acs/upload/`

```
internal/acs/upload/
├── server.go       # HTTP 服务器，处理文件上传
├── handler.go      # 上传请求处理
├── token.go        # 上传 Token 生成与验证
└── store.go        # 文件存储到 MinIO
```

**职责**:
1. 监听 HTTP POST 请求
2. 验证上传 Token
3. 接收文件并存储到 MinIO
4. 发布 `file.uploaded` 事件（供 TC/ATC 处理关联）

#### 3.2.2 Upload URL 格式

```
http://acs-server:7547/upload/{device_sn}/{upload_token}
```

- `device_sn`: 设备序列号
- `upload_token`: 一次性上传令牌（JWT 或随机字符串）

**Token 内容**:
```json
{
  "device_sn": "ABC123",
  "file_type": "4",
  "expires_at": 1708329600,
  "command_key": "ck-12345"
}
```

#### 3.2.3 上传流程

**模式一：ACS 主动触发**

```
1. 管理系统调用 API 发起上传请求
   POST /api/v1/devices/{id}/upload
   { "file_type": "4" }

2. ACS 生成 Upload RPC
   - 生成 upload_token
   - 构建 Upload URL: http://acs:7547/upload/{sn}/{token}
   - 将 token 存入 Redis (key: upload:token:{token})

3. ACS 发送 Upload RPC 给设备
   <Upload>
     <CommandKey>ck-12345</CommandKey>
     <FileType>4</FileType>
     <URL>http://acs:7547/upload/ABC123/tok-xxx</URL>
   </Upload>

4. 设备收到 Upload RPC，开始上传
   POST /upload/ABC123/tok-xxx
   Content-Type: multipart/form-data
   [File Binary Data]

5. UploadServer 处理上传
   - 验证 token
   - 存储文件到 MinIO
   - 记录上传信息到 Redis
   - 返回 200 OK

6. 设备发送 TransferComplete
   <TransferComplete>
     <CommandKey>ck-12345</CommandKey>
     <StartTime>2026-03-19T10:00:00Z</StartTime>
     <CompleteTime>2026-03-19T10:00:30Z</CompleteTime>
   </TransferComplete>

7. ACS 处理 TransferComplete
   - 查询 Redis 获取上传信息
   - 发布 pm.file.received 或 mr.file.received 事件
```

**模式二：设备自主上传**

```
1. ACS 通过配置下发上传 URL
   SetParameterValues:
     Device.X_Upload.URL = "http://acs:7547/upload/{sn}"

2. 设备主动上传
   POST /upload/{sn}
   X-Upload-Token: {可选的认证 Token}
   [File Binary Data]

3. UploadServer 处理上传
   - 根据 device_sn 查找设备
   - 存储文件到 MinIO
   - 记录上传信息
   - 返回 200 OK

4. 设备发送 AutonomousTransferComplete
   <AutonomousTransferComplete>
     <AnnounceURL>http://acs:7547/upload/{sn}</AnnounceURL>
     <TransferURL>http://acs:7547/upload/{sn}</TransferURL>
     <IsDownload>false</IsDownload>
     <FileType>4</FileType>
     <FileSize>1024</FileSize>
     <TargetFileName>pm_20260319.xml</TargetFileName>
   </AutonomousTransferComplete>

5. ACS 处理 AutonomousTransferComplete
   - 查询已上传的文件
   - 发布 pm.file.received 或 mr.file.received 事件
```

### 3.3 数据结构

#### 3.3.1 上传会话（Redis）

```
Key: upload:session:{device_sn}:{command_key}
Value: {
  "file_type": "4",
  "file_path": "pm/2026/03/19/ABC123/pm_20260319.xml",
  "file_size": 1024,
  "bucket": "pm-bucket",
  "uploaded_at": "2026-03-19T10:00:30Z"
}
TTL: 24h
```

#### 3.3.2 Upload Token（Redis）

```
Key: upload:token:{token}
Value: {
  "device_sn": "ABC123",
  "command_key": "ck-12345",
  "file_type": "4",
  "created_at": "2026-03-19T10:00:00Z"
}
TTL: 1h
```

### 3.4 API 设计

#### 3.4.1 管理面 API（新增）

```
POST /api/v1/devices/{id}/upload
请求体:
{
  "file_type": "4",           // TR069 FileType: 4=PM, 5=MR
  "delay_seconds": 0          // 可选，延迟上传
}

响应:
{
  "command_key": "ck-12345",
  "upload_url": "http://acs:7547/upload/ABC123/tok-xxx",
  "status": "pending"
}
```

#### 3.4.2 ACS 文件服务器 API（新增）

```
POST /upload/{device_sn}/{token}
Content-Type: multipart/form-data

响应:
200 OK - 上传成功
400 Bad Request - Token 无效
404 Not Found - 路径不存在
500 Internal Server Error - 服务器错误
```

### 3.5 文件存储路径

```
MinIO Bucket 结构:
├── pm-bucket/
│   └── 2026/03/19/
│       └── {device_sn}/
│           └── pm_{timestamp}.xml
├── mr-bucket/
│   └── 2026/03/19/
│       └── {device_sn}/
│           └── mr_{timestamp}.xml
└── logs-bucket/
    └── 2026/03/19/
        └── {device_sn}/
            └── log_{timestamp}.txt
```

---

## 4. 实现计划（MinIO Presigned URL 方案）

### 方案优势

| 对比项 | MinIO Presigned URL | ACS 代理服务器 |
|--------|---------------------|---------------|
| **开发工作量** | ⭐ 半天 | ⭐⭐⭐ 3-4 天 |
| **新增代码** | ~100 行 | ~500 行 |
| **新增端点** | 无 | 1 个 |
| **性能** | 直传，最优 | 代理转发 |
| **依赖** | MinIO（已有） | MinIO + 新服务器 |

### 阶段一：Presigned URL 生成（半天）

**文件**：`internal/acs/upload/presigned.go`

```go
// 1. 新建 PresignedUploader 结构体
// 2. 实现 GenerateUploadURL 方法
// 3. 添加配置支持（external_endpoint, url_ttl）
```

**修改点**：
1. `internal/core/appconfig/config.go` — 添加 MinIO external_endpoint 配置
2. `cmd/acs/main.go` — 初始化 PresignedUploader

### 阶段二：集成 Upload RPC（半天）

**修改文件**：`internal/acs/rpc/dispatcher.go`

```go
// UploadHandler 添加 uploader 依赖
type UploadHandler struct {
    uploader *upload.PresignedUploader
}

// BuildRequest 中自动生成 Presigned URL（如果未提供）
func (h *UploadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
    // ...
    if params.URL == "" {
        params.URL, _ = h.uploader.GenerateUploadURL(...)
    }
    // ...
}
```

### 阶段三：TransferComplete 处理优化（半天）

**修改文件**：`internal/acs/handler.go` / `internal/transfer/bridge.go`

当 TransferComplete 的 URL 是 MinIO Presigned URL 时：
- 直接从 MinIO 读取已上传的文件
- 发布 `pm.file.received` 事件

### 阶段四：测试与文档（半天）

1. 单元测试
2. 集成测试（模拟 CPE PUT 上传）
3. 更新配置文档

---

## 4.1 实现计划（推荐：ACS 代理服务器方案）

> **已确认**：CPE 无法访问 MinIO，使用 ACS 代理方案

### 架构设计

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              ACS Server                                   │
│                                                                           │
│   CPE ──────► /acs (SOAP) ──────► Handler ──────► EventBus              │
│                  │                                                        │
│                  │ Upload RPC                                             │
│                  │ (含 ACS 上传 URL)                                       │
│                  ▼                                                        │
│   CPE ──────► /upload/{token} ──► UploadHandler ──► MinIO                │
│                (HTTP PUT)              │                                  │
│                                        │ Redis                            │
│                                        ▼                                  │
│                                   Session Store                           │
│                                        │                                  │
│   CPE ──────► /acs (SOAP) ──────► TC Handler ────► 关联 Session          │
│              TransferComplete                        │                    │
│                                                      ▼                    │
│                                              pm.file.received            │
└──────────────────────────────────────────────────────────────────────────┘
```

### 阶段一：基础设施（1 天）

#### 1.1 配置扩展

**修改文件**：`internal/core/appconfig/config.go`

```go
// ACSConfig 添加 MinIO 和 Upload 配置
type ACSConfig struct {
    // ... 现有字段 ...
    MinIO   MinIOConfig   `mapstructure:"minio"`   // 新增
    Upload  UploadConfig  `mapstructure:"upload"`  // 新增
}

// UploadConfig 文件上传配置
type UploadConfig struct {
    BaseURL     string        `mapstructure:"base_url"`      // 上传服务基础 URL，如 http://acs:7547
    Path        string        `mapstructure:"path"`          // 上传路径前缀，默认 /upload
    TokenTTL    time.Duration `mapstructure:"token_ttl"`     // Token 有效期，默认 1h
    MaxFileSize int64         `mapstructure:"max_file_size"` // 最大文件大小，默认 100MB
}
```

**配置文件示例**：`cmd/acs/etc/config.dev.yaml`

```yaml
minio:
  endpoint: "minio:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  buckets:
    pm_files: "pm-files"
    mr_files: "mr-files"
    logs: "logs"

upload:
  base_url: "http://acs:7547"    # CPE 可访问的 ACS 地址
  path: "/upload"
  token_ttl: 1h
  max_file_size: 104857600       # 100MB
```

#### 1.2 Bootstrap 修改

**修改文件**：`internal/core/bootstrap/bootstrap.go`

```go
// InitForACS 添加 MinIO 初始化
func InitForACS(ctx context.Context, cfg *appconfig.ACSConfig) (*App, error) {
    app, err := newBase(cfg.Log, cfg.Metrics.Port)
    if err != nil {
        return nil, err
    }

    if err := app.connectRedis(cfg.Redis); err != nil {
        return nil, err
    }
    if err := app.connectNATS(ctx, cfg.NATS); err != nil {
        return nil, err
    }
    // 新增: MinIO 连接
    if err := app.connectMinIO(ctx, cfg.MinIO); err != nil {
        return nil, err
    }
    app.createEventBus()

    return app, nil
}
```

#### 1.3 创建 Upload 组件

**新建目录**：`internal/acs/upload/`

```
internal/acs/upload/
├── token.go       # Token 生成与验证（JWT）
├── session.go     # 上传会话管理（Redis）
├── handler.go     # HTTP 上传处理
└── uploader.go    # 上传 URL 生成器
```

### 阶段二：核心实现（1.5 天）

#### 2.1 Token 管理

**文件**：`internal/acs/upload/token.go`

```go
package upload

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Claims 上传 Token 的 JWT Claims
type Claims struct {
    jwt.RegisteredClaims
    DeviceSN     string `json:"device_sn"`
    CommandKey   string `json:"command_key"`
    FileType     string `json:"file_type"`
    TargetFileName string `json:"target_filename"`
}

// TokenManager 管理 JWT Token
type TokenManager struct {
    secretKey []byte
    ttl       time.Duration
}

func NewTokenManager(secretKey string, ttl time.Duration) *TokenManager {
    return &TokenManager{
        secretKey: []byte(secretKey),
        ttl:       ttl,
    }
}

// Generate 生成上传 Token
func (m *TokenManager) Generate(deviceSN, commandKey, fileType, filename string) (string, error) {
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ID:        commandKey,
        },
        DeviceSN:       deviceSN,
        CommandKey:     commandKey,
        FileType:       fileType,
        TargetFileName: filename,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secretKey)
}

// Validate 验证并解析 Token
func (m *TokenManager) Validate(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return m.secretKey, nil
    })
    if err != nil {
        return nil, fmt.Errorf("parse token: %w", err)
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    return claims, nil
}
```

#### 2.2 上传会话管理

**文件**：`internal/acs/upload/session.go`

```go
package upload

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Session 上传会话
type Session struct {
    DeviceSN       string    `json:"device_sn"`
    CommandKey     string    `json:"command_key"`
    FileType       string    `json:"file_type"`
    Bucket         string    `json:"bucket"`
    ObjectPath     string    `json:"object_path"`
    FileSize       int64     `json:"file_size"`
    UploadedAt     time.Time `json:"uploaded_at"`
    TargetFileName string    `json:"target_filename"`
}

// SessionStore 上传会话存储
type SessionStore struct {
    redis *redis.Client
    ttl   time.Duration
}

func NewSessionStore(redis *redis.Client, ttl time.Duration) *SessionStore {
    return &SessionStore{redis: redis, ttl: ttl}
}

// Save 保存上传会话
func (s *SessionStore) Save(ctx context.Context, session *Session) error {
    key := s.key(session.DeviceSN, session.CommandKey)
    data, err := json.Marshal(session)
    if err != nil {
        return fmt.Errorf("marshal session: %w", err)
    }
    return s.redis.Set(ctx, key, data, s.ttl).Err()
}

// Get 获取上传会话
func (s *SessionStore) Get(ctx context.Context, deviceSN, commandKey string) (*Session, error) {
    key := s.key(deviceSN, commandKey)
    data, err := s.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return nil, fmt.Errorf("unmarshal session: %w", err)
    }
    return &session, nil
}

// Delete 删除上传会话
func (s *SessionStore) Delete(ctx context.Context, deviceSN, commandKey string) error {
    key := s.key(deviceSN, commandKey)
    return s.redis.Del(ctx, key).Err()
}

func (s *SessionStore) key(deviceSN, commandKey string) string {
    return fmt.Sprintf("upload:session:%s:%s", deviceSN, commandKey)
}
```

#### 2.3 上传 URL 生成器

**文件**：`internal/acs/upload/uploader.go`

```go
package upload

import (
    "fmt"
    "time"

    "github.com/omcgo/omcgo/internal/core/appconfig"
)

// Uploader 生成上传 URL 和相关信息
type Uploader struct {
    tokenManager *TokenManager
    sessionStore *SessionStore
    config       UploadConfig
    buckets      appconfig.BucketConfig
}

func NewUploader(
    tokenManager *TokenManager,
    sessionStore *SessionStore,
    config UploadConfig,
    buckets appconfig.BucketConfig,
) *Uploader {
    return &Uploader{
        tokenManager: tokenManager,
        sessionStore: sessionStore,
        config:       config,
        buckets:      buckets,
    }
}

// UploadInfo 上传信息
type UploadInfo struct {
    URL           string // 完整上传 URL
    Token         string // JWT Token
    ObjectPath    string // MinIO 对象路径
    Bucket        string // MinIO Bucket
}

// GenerateUploadURL 生成上传 URL
func (u *Uploader) GenerateUploadURL(deviceSN, commandKey, fileType, filename string) (*UploadInfo, error) {
    // 1. 生成 Token
    token, err := u.tokenManager.Generate(deviceSN, commandKey, fileType, filename)
    if err != nil {
        return nil, fmt.Errorf("generate token: %w", err)
    }

    // 2. 确定目标 bucket
    bucket := u.bucketForFileType(fileType)

    // 3. 生成对象路径
    objectPath := u.objectPath(deviceSN, filename)

    // 4. 构建完整 URL
    uploadURL := fmt.Sprintf("%s%s/%s", u.config.BaseURL, u.config.Path, token)

    return &UploadInfo{
        URL:        uploadURL,
        Token:      token,
        ObjectPath: objectPath,
        Bucket:     bucket,
    }, nil
}

func (u *Uploader) bucketForFileType(fileType string) string {
    switch fileType {
    case "4": // Vendor Configuration File / PM
        return u.buckets.PMFiles
    case "5": // Log File / MR
        return u.buckets.MRFiles
    case "6": // Log
        return u.buckets.Logs
    default:
        return u.buckets.PMFiles
    }
}

func (u *Uploader) objectPath(deviceSN, filename string) string {
    now := time.Now()
    return fmt.Sprintf("%s/%s/%s",
        now.Format("2006/01/02"),
        deviceSN,
        filename,
    )
}
```

#### 2.4 HTTP 上传处理

**文件**：`internal/acs/upload/handler.go`

```go
package upload

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/minio/minio-go/v7"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// Handler 处理文件上传请求
type Handler struct {
    tokenManager *TokenManager
    sessionStore *SessionStore
    minioClient  *minio.Client
    maxFileSize  int64
    logger       *zap.Logger
}

func NewHandler(
    tokenManager *TokenManager,
    sessionStore *SessionStore,
    minioClient *minio.Client,
    maxFileSize int64,
    logger *zap.Logger,
) *Handler {
    return &Handler{
        tokenManager: tokenManager,
        sessionStore: sessionStore,
        minioClient:  minioClient,
        maxFileSize:  maxFileSize,
        logger:       logger,
    }
}

// ServeHTTP 处理上传请求
// 路由: PUT /upload/{token}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut && r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 1. 提取 Token
    token := h.extractToken(r.URL.Path)
    if token == "" {
        http.Error(w, "missing token", http.StatusBadRequest)
        return
    }

    // 2. 验证 Token
    claims, err := h.tokenManager.Validate(token)
    if err != nil {
        h.logger.Warn("invalid upload token", zap.Error(err))
        http.Error(w, "invalid token", http.StatusBadRequest)
        return
    }

    // 3. 检查文件大小
    if r.ContentLength > h.maxFileSize {
        http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
        return
    }

    // 4. 生成对象路径
    bucket := h.bucketForFileType(claims.FileType)
    objectPath := h.objectPath(claims.DeviceSN, claims.TargetFileName)

    // 5. 流式上传到 MinIO
    ctx := r.Context()
    info, err := h.minioClient.PutObject(ctx, bucket, objectPath, r.Body, r.ContentLength, minio.PutObjectOptions{
        ContentType: "application/octet-stream",
    })
    if err != nil {
        h.logger.Error("upload to minio failed",
            zap.Error(err),
            zap.String("device_sn", claims.DeviceSN),
            zap.String("path", objectPath),
        )
        http.Error(w, "upload failed", http.StatusInternalServerError)
        return
    }

    // 6. 保存会话
    session := &Session{
        DeviceSN:       claims.DeviceSN,
        CommandKey:     claims.CommandKey,
        FileType:       claims.FileType,
        Bucket:         bucket,
        ObjectPath:     objectPath,
        FileSize:       info.Size,
        UploadedAt:     time.Now(),
        TargetFileName: claims.TargetFileName,
    }
    if err := h.sessionStore.Save(ctx, session); err != nil {
        h.logger.Warn("save upload session failed", zap.Error(err))
        // 不返回错误，上传已成功
    }

    h.logger.Info("file uploaded",
        zap.String("device_sn", claims.DeviceSN),
        zap.String("command_key", claims.CommandKey),
        zap.String("path", objectPath),
        zap.Int64("size", info.Size),
    )

    // 7. 返回成功
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"status":"ok","path":"%s","size":%d}`, objectPath, info.Size)
}

func (h *Handler) extractToken(path string) string {
    // 路径格式: /upload/{token}
    parts := strings.Split(strings.TrimPrefix(path, "/upload/"), "/")
    if len(parts) > 0 && parts[0] != "" {
        return parts[0]
    }
    return ""
}

func (h *Handler) bucketForFileType(fileType string) string {
    // 由 Uploader 提供，这里简化实现
    switch fileType {
    case "4":
        return "pm-files"
    case "5":
        return "mr-files"
    case "6":
        return "logs"
    default:
        return "pm-files"
    }
}

func (h *Handler) objectPath(deviceSN, filename string) string {
    return fmt.Sprintf("%s/%s/%s",
        time.Now().Format("2006/01/02"),
        deviceSN,
        filename,
    )
}
```

### 阶段三：集成 ACS（0.5 天）

#### 3.1 修改 ACS Server

**修改文件**：`internal/acs/server.go`

```go
// ServerDeps 添加 Upload 依赖
type ServerDeps struct {
    // ... 现有字段 ...
    UploadHandler *upload.Handler  // 新增
}

// NewACSServer 添加上传路由
func NewACSServer(cfg appconfig.ACSConfig, deps ServerDeps) *ACSServer {
    // ... 现有代码 ...

    mux := http.NewServeMux()
    mux.HandleFunc("/acs", h.ServeHTTP)
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

    // 新增: 文件上传路由
    if deps.UploadHandler != nil {
        mux.Handle("/upload/", deps.UploadHandler)
    }

    // ... 现有代码 ...
}
```

#### 3.2 修改 ACS main.go

**修改文件**：`cmd/acs/main.go`

```go
func runACS(cmd *cobra.Command, args []string) error {
    // ... 现有代码 ...

    // 新增: 创建 Upload 组件
    var uploadHandler *upload.Handler
    if app.MinIO != nil {
        tokenManager := upload.NewTokenManager(
            cfg.Upload.TokenSecret, // 从配置读取
            cfg.Upload.TokenTTL,
        )
        sessionStore := upload.NewSessionStore(app.Redis, 24*time.Hour)
        uploadHandler = upload.NewHandler(
            tokenManager,
            sessionStore,
            app.MinIO,
            cfg.Upload.MaxFileSize,
            app.Logger,
        )
    }

    deps := acs.NewDefaultDeps(
        // ... 现有参数 ...
    )
    deps.UploadHandler = uploadHandler  // 新增

    // ... 现有代码 ...
}
```

#### 3.3 修改 TransferComplete 处理

**修改文件**：`internal/acs/handler.go`

```go
func (s *Session) handleTransferComplete(env *soap.Envelope) error {
    tc := env.Body.TransferComplete

    // 新增: 查询上传会话（MinIO 代理模式）
    if s.uploadSessionStore != nil {
        session, err := s.uploadSessionStore.Get(s.ctx, s.deviceSN, tc.CommandKey)
        if err == nil && session != nil {
            // 文件已通过 UploadHandler 上传到 MinIO
            // 直接发布 pm.file.received 事件
            return s.publishFileReceivedEvent(session)
        }
    }

    // 原有逻辑: TransferURL 模式（设备托管文件，ACS 下载）
    // ...
}
```

### 阶段四：测试与文档（0.5 天）

1. **单元测试**
   - `upload/token_test.go`
   - `upload/session_test.go`
   - `upload/handler_test.go`

2. **集成测试**
   - 模拟 CPE PUT 上传
   - 验证 MinIO 存储
   - 验证 TransferComplete 关联

3. **E2E 测试**
   - 完整 Upload RPC 流程

---

## 4.2 实现计划（备选：ACS 代理服务器方案 - 简化版）

> 以下为原方案，供参考

---

## 5. 与现有代码的集成点

### 5.1 ACS Server 集成（MinIO Presigned URL 方案）

```go
// cmd/acs/main.go
func main() {
    // 现有代码...

    // 新增: 初始化 PresignedUploader
    uploader := upload.NewPresignedUploader(
        minioClient,
        cfg.MinIO.Buckets,
        time.Hour, // URL TTL
    )

    // 传入 RPC Dispatcher
    dispatcher := rpc.NewDispatcher(
        // ... 其他依赖
        uploader,
    )
}
```

```go
// internal/acs/rpc/dispatcher.go
type UploadHandler struct {
    uploader *upload.PresignedUploader
}

func (h *UploadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
    var params soap.UploadData
    if err := json.Unmarshal(cmd.Params, &params); err != nil {
        return nil, fmt.Errorf("parse Upload params: %w", err)
    }

    // 新增: 如果没有提供 URL，生成 Presigned URL
    if params.URL == "" {
        uploadURL, objectPath, err := h.uploader.GenerateUploadURL(
            context.Background(),
            cmd.DeviceSN,
            params.FileType,
            params.TargetFileName,
        )
        if err != nil {
            return nil, fmt.Errorf("generate upload URL: %w", err)
        }
        params.URL = uploadURL
        // 将 objectPath 存入 Redis，供后续 TC 处理使用
        h.sessionStore.SetObjectPath(cmd.DeviceSN, cmd.CommandKey, objectPath)
    }

    params.ID = cmd.CommandKey
    params.CommandKey = cmd.CommandKey
    return soap.RenderResponse(soap.UploadTmpl, params)
}
```

### 5.2 ACS Server 集成（备选：代理方案）

```go
// internal/acs/server.go
func NewServer(cfg *config.ACSConfig, ...) *Server {
    // 现有代码...

    // 新增: 初始化 UploadServer
    uploadServer := upload.NewUploadServer(
        minioClient,
        redisClient,
        pmBucket,
        mrBucket,
        logsBucket,
        logger,
    )

    // 新增: 注册路由
    mux.HandleFunc("/upload/", uploadServer.HandleUpload)

    return &Server{...}
}
```

### 5.3 TransferComplete 处理修改（MinIO Presigned URL 方案）

```go
// internal/acs/handler.go
func (s *Session) handleTransferComplete(env *soap.Envelope) error {
    tc := env.Body.TransferComplete

    // 新增: 查询 Redis 中的 objectPath（由 UploadHandler 存储）
    objectPath, err := s.sessionStore.GetObjectPath(s.ctx, s.deviceSN, tc.CommandKey)
    if err == nil && objectPath != "" {
        // MinIO Presigned URL 模式：文件已直接上传到 MinIO
        // 直接发布 pm.file.received 事件
        return s.publishFileReceivedEvent(s.deviceSN, objectPath, tc)
    }

    // 原有逻辑: TransferURL 模式
    // ...
}
```

### 5.4 TransferBridge 修改（MinIO Presigned URL 方案）

```go
// internal/transfer/bridge.go
func (b *TransferBridge) handleAutonomousTransferComplete(ctx context.Context, evt event.Event) error {
    var payload atcPayload
    // ...

    // 新增: 检查是否是 ACS 服务器模式
    if strings.HasPrefix(payload.TransferURL, b.acsUploadBaseURL) {
        // 文件已上传到 ACS 服务器，查询上传记录
        return b.handleACSUploadMode(ctx, &payload, dev)
    }

    // 原有逻辑: TransferURL 下载模式
    // ...
}
```

---

## 6. 配置项（MinIO Presigned URL 方案）

```yaml
# config.yaml
minio:
  endpoint: "minio:9000"                        # 内部地址（ACS/Worker 访问）
  external_endpoint: "minio.example.com:9000"   # CPE 可访问的外部地址（可选）
  access_key: "minioadmin"
  secret_key: "minioadmin"
  use_ssl: false
  upload_url_ttl: 1h                            # 预签名 URL 有效期
  buckets:
    pm_files: "pm-files"
    mr_files: "mr-files"
    logs: "logs"
```

**配置说明**：
- `endpoint`: 内部地址，用于 ACS/Worker 访问 MinIO
- `external_endpoint`: CPE 设备可访问的外部地址（如果与内部地址不同）
- `upload_url_ttl`: 预签名 URL 有效期，建议 1 小时

---

## 7. 安全考虑（MinIO Presigned URL 方案）

1. **签名验证**: MinIO Presigned URL 内置 AWS Signature V4 验证
2. **时效性**: URL 有过期时间（建议 1 小时）
3. **一次性**: 每次 Upload RPC 生成新的唯一 URL
4. **路径隔离**: 按设备/日期组织存储路径
5. **HTTPS**: 生产环境建议 MinIO 启用 HTTPS

---

## 8. 待确认事项

### 8.1 网络可达性（已确认 ✅）

- [x] CPE 无法访问 MinIO → **使用 ACS 代理方案**

### 8.2 方案选择（已确认 ✅）

- [x] **方案 B：ACS 代理服务器**（CPE → ACS → MinIO）

### 8.3 Token 方案

- [ ] JWT Token（推荐，无状态）
- [ ] 随机字符串 + Redis 存储（可撤销）

### 8.4 文件命名策略

- [ ] 保留原始文件名
- [ ] 生成唯一文件名（时间戳 + UUID）

### 8.5 模式二（设备自主上传）是否需要支持

- [ ] 是，完整支持
- [ ] 否，仅支持模式一（ACS 主动触发）

---

## 9. 实现工作量估算

| 阶段 | 工作内容 | 预估时间 |
|------|---------|---------|
| 阶段一 | 配置扩展 + Bootstrap 修改 | 0.5 天 |
| 阶段二 | Upload 组件实现（Token/Session/Handler） | 1 天 |
| 阶段三 | ACS Server 集成 + TC 处理修改 | 0.5 天 |
| 阶段四 | 测试与文档 | 0.5 天 |
| **总计** | | **2.5 天** |

---

## 10. 参考资料

- TR-069 Amendment 6: Section 3.2.2 (Upload), 3.2.5 (TransferComplete), 3.2.6 (AutonomousTransferComplete)
- MinIO Go SDK: https://min.io/docs/minio/linux/developers/go/minio-go.html
- JWT Go: https://github.com/golang-jwt/jwt
- `omcgo/docs/design/acs-service-flow.md` - ACS 服务流程设计
- `omcgo/docs/design/pm-kpi-flow.md` - PM/KPI 流程设计
- `internal/transfer/bridge.go` - 现有 TransferURL 模式实现
- `internal/core/bootstrap/bootstrap.go` - Bootstrap 初始化
- `internal/acs/server.go` - ACS Server 实现

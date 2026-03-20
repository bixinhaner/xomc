# CPE 文件上传认证方案

> 文档版本: 2.0 (简化版)
> 创建日期: 2026-03-20
> 状态: 已实现

---

## 1. 概述

CPE 设备通过 HTTP POST 上传文件到 ACS 服务器，使用 HTTP Basic Authentication 进行身份验证。

### 简化设计原则

1. **仅 POST 方法**：移除 PUT 支持
2. **单一端点**：`/smallcell/FileUploadService`
3. **全局凭证**：配置文件中定义统一的用户名/密码
4. **TR069 下发**：ACS 通过 Upload RPC 下发 URL、用户名和密码

---

## 2. 端点规范

### 2.1 请求格式

```http
POST /smallcell/FileUploadService?fileType=PM&filename=pm_20260320.xml HTTP/1.1
Host: 172.21.175.129:8080
Authorization: Basic {base64(username:password)}
Content-Type: application/octet-stream
Content-Length: 1234

[binary file data]
```

### 2.2 参数说明

| 参数 | 来源 | 必需 | 说明 |
|------|------|------|------|
| `fileType` | Query Param | 是 | 文件类型：`PM`(4), `MR`(5), `Log`(6) |
| `filename` | Query Param | 是 | 目标文件名 |
| `username` | Basic Auth | 是 | 设备 SN（通过 TR069 Upload RPC 下发） |
| `password` | Basic Auth | 是 | 全局密码（配置文件定义） |

### 2.3 响应格式

**成功 (200 OK)**：
```json
{"status":"ok","path":"pm/2026/03/20/ABC123456/pm_20260320.xml","size":1234}
```

**失败**：
- `401 Unauthorized` - 凭证无效或缺失
- `400 Bad Request` - 缺少必需参数
- `413 Request Entity Too Large` - 文件过大
- `500 Internal Server Error` - 服务器错误

---

## 3. TR069 Upload RPC

### 3.1 SOAP 消息示例

ACS 发送给 CPE 的 Upload RPC：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">acs-upload-001</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Upload>
      <CommandKey>upload-20260320-001</CommandKey>
      <FileType>4</FileType>
      <URL>http://172.21.175.129:8080/smallcell/FileUploadService?fileType=PM&amp;filename=pm_20260320.xml</URL>
      <Username>ABC123456</Username>
      <Password>global_upload_password</Password>
      <DelaySeconds>0</DelaySeconds>
    </cwmp:Upload>
  </soap:Body>
</soap:Envelope>
```

### 3.2 字段说明

| 字段 | 说明 |
|------|------|
| `CommandKey` | 唯一命令标识，用于 TransferComplete 关联 |
| `FileType` | TR069 文件类型：4=PM, 5=MR, 6=Log |
| `URL` | 上传目标 URL（含 fileType 和 filename 参数） |
| `Username` | 设备 SN（用于 Basic Auth 认证） |
| `Password` | 全局上传密码（配置文件定义） |
| `DelaySeconds` | 延迟执行秒数（通常为 0） |

---

## 4. 配置示例

### 4.1 ACS 配置文件

```yaml
# cmd/acs/etc/config.yaml
server:
  host: "0.0.0.0"
  port: 8080

upload:
  # 全局凭证（CPE 使用此凭证上传文件）
  username: "upload_user"
  password: "secure_upload_password_123"

  # 文件大小限制
  max_file_size: 104857600  # 100MB

  # MinIO 存储桶
  buckets:
    pm_files: "pm-files"
    mr_files: "mr-files"
    logs: "logs"
```

### 4.2 Upload RPC 参数生成

```go
// 内部函数：生成 Upload RPC 参数
func (s *ACSService) BuildUploadParams(device *Device, fileType, filename string) *UploadData {
    baseURL := s.config.Server.BaseURL // "http://172.21.175.129:8080"

    return &UploadData{
        ID:           fmt.Sprintf("upload-%s-%d", device.SerialNumber, time.Now().Unix()),
        CommandKey:   uuid.New().String(),
        FileType:     fileType,
        URL:          fmt.Sprintf("%s/smallcell/FileUploadService?fileType=%s&filename=%s",
                        baseURL, fileType, filename),
        Username:     device.SerialNumber,  // CPE 使用 SN 作为 username
        Password:     s.config.Upload.Password,
        DelaySeconds: 0,
    }
}
```

---

## 5. MinIO 存储路径

### 5.1 目录结构

```
{bucket}/
├── pm/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
├── mr/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
└── logs/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
```

### 5.2 示例

```
pm-files/pm/2026/03/20/ABC123456/pm_20260320_100000.xml
mr-files/mr/2026/03/20/ABC123456/mro_20260320_100000.xml
logs-files/logs/2026/03/20/ABC123456/device_log_20260320.txt
```

---

## 6. CPE 上传流程

```
┌─────────┐                              ┌─────────┐
│   ACS   │                              │   CPE   │
└────┬────┘                              └────┬────┘
     │                                         │
     │  1. Upload RPC (含 URL, Username,       │
     │     Password)                           │
     │ ────────────────────────────────────────►│
     │                                         │
     │                                         │ 2. 保存凭证
     │                                         │
     │  3. HTTP POST /FileUploadService        │
     │     Authorization: Basic xxx            │
     │     Body: [文件内容]                     │
     │ ◄───────────────────────────────────────│
     │                                         │
     │  4. 200 OK                              │
     │ ────────────────────────────────────────►│
     │                                         │
     │  5. TransferComplete                    │
     │ ◄───────────────────────────────────────│
     │                                         │
     │  6. TransferCompleteResponse            │
     │ ────────────────────────────────────────►│
     │                                         │
```

---

## 7. 安全建议

| 项目 | 建议 |
|------|------|
| **传输加密** | 生产环境必须使用 HTTPS |
| **密码管理** | 使用环境变量或密钥管理服务存储密码 |
| **定期轮换** | 建议每 90 天更换全局密码 |
| **日志审计** | 记录所有上传请求（设备 SN、文件类型、IP） |

---

## 8. 实现文件

| 文件 | 说明 |
|------|------|
| `internal/acs/upload/handler.go` | HTTP 处理器（Basic Auth 验证） |
| `internal/acs/server.go` | 路由注册 |
| `pkg/soap/templates.go` | Upload RPC SOAP 模板 |

---

*文档更新时间: 2026-03-20*

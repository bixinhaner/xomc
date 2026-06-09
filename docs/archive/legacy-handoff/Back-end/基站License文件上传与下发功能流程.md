# 基站 License 文件上传与下发功能流程

> 文档范围：涵盖 License 文件上传、验证、任务创建、TR-069 下发交互及结果处理全过程，不包含具体代码实现。

---

## 一、功能概述

License 下发功能将 License 授权文件从 OMC 服务器推送到对应基站（eNB），激活或更新设备的授权能力（如最大接入数量、功能特性开关等）。整体分为三个阶段：

1. **上传阶段**：将 License 文件上传到 OMC 服务器并完成验证解析。
2. **任务创建阶段**：选择目标基站，创建 License 下发任务。
3. **下发执行阶段**：通过 TR-069 Download RPC 将 License 文件推送到基站，等待结果回报。

---

## 二、系统分层架构

```
┌────────────────────────────────────────────────┐
│  前端 Web 界面                                  │  用户操作入口
├────────────────────────────────────────────────┤
│  OMCWebServer（REST API 层）                   │  License 上传、任务创建、查询
├────────────────────────────────────────────────┤
│  licenseReader（License 解析服务）             │  License 文件验证、信息提取
├────────────────────────────────────────────────┤
│  cellHandler（业务逻辑层）                     │  任务调度、状态管理、RPC调用
├────────────────────────────────────────────────┤
│  fileDownload（文件下载服务）                  │  提供 HTTP 文件下载接口供基站拉取
├────────────────────────────────────────────────┤
│  TR069Server / cpeHandler（协议层）            │  TR-069 SOAP 消息收发
├────────────────────────────────────────────────┤
│  基站设备                                      │  下载 License，激活授权，上报结果
└────────────────────────────────────────────────┘
```

---

## 三、License 文件类型（FileType）

TR-069 协议中，License 下发使用以下 `FileType` 参数：

| FileType 值 | 适用场景 |
|---|---|
| `License File` | 普通基站 License 下发（LICENSE_TASK、HALOB_AUTO_LICENSE） |
| `1588 License` | 1588 精密时钟授权（1588_LICENSE_TASK） |

License 下发使用 TR-069 **Download RPC**（设备视角：设备主动从 OMC 下载文件）。

---

## 四、License 文件命名与存储规则

- **文件命名**：`{SN}.lic`，文件名（去掉后缀）即为对应基站的序列号（SN）。
- **存储路径**：
  - 普通 License：`{cellUploadPath}/{operatorCode}/license/{fileName}`
  - 1588 License：`{omcUploadPath}/{operatorCode}/1588License/{fileName}`

---

## 五、完整流程

### 阶段一：License 文件上传与验证

```
用户选择 License 文件，通过 Web 界面上传
    │
    ▼
OMCWebServer 接收文件
  - 计算文件 MD5
  - 保存文件到 OMC 服务器本地（或 FTP）
  - 从文件名提取基站 SN（去掉 .lic 后缀）
  - 调用 licenseReader 服务进行验证
    │
    ▼
licenseReader 解析 License 文件
  - 使用 TrueLicense 库验证签名合法性
  - 提取授权信息（productType、eNBMaxNum、capability 等）
  - 将 License 信息缓存到 Redis（licenseInfo Hash 结构）
    │
    ▼
将 License 文件元数据写入数据库
  - 表：license_file_info
  - 字段：SN、文件名、MD5、存储路径、上传时间等
```

### 阶段二：创建 License 下发任务

```
用户选择目标基站 SN 列表，提交下发任务
    │
    ▼
OMCWebServer 接收请求
  - 检查每个 SN 对应的 License 文件是否已上传
  - 为每个 SN 创建任务记录 → 表：license_task_record
    （返回 taskId）
    │
    ▼
构建 ActionMethodCallEntity
  - methodName = "LicenseTask"
  - 参数：taskIdList、serialNumbers、isGnb、userCode
  - 加入 QueueActionMethodCall 执行队列
    │
    ▼
cellHandler 从队列取出任务，开始执行
```

### 阶段三：TR-069 下发执行

```
cellHandler 调用 TR069Server 构建 Download RPC
    │
    ▼
TR069Server 向基站发送 TR-069 Download RPC

下发参数：
  - CommandKey：  "Download License,{uuid}"
  - FileType：    "License File"（或 "1588 License"）
  - URL：         http://{omc_server}/{license_path}/{SN}.lic
  - Protocol：    HTTP
  - FileSize：    文件大小（字节）
  - Md5：         文件 MD5 校验值（扩展字段）
  - DelaySeconds：10（License 下发特有延迟）

同时将请求信息缓存到 Redis：
  Key：{smallCellCode}_DownloadReqMgr
  Value：{filename, filesize, filetype, md5, task_id, taskType, commandKey, uuid}
    │
    ▼
基站设备通过 HTTP 从 fileDownload 模块拉取 License 文件
  - fileDownload 验证任务是否仍有效（未被中止）
  - 提供文件流供基站下载
    │
    ▼
基站验证 License 文件完整性（MD5）
基站激活 License 授权
    │
    ▼
基站上报 TransferComplete 事件

结果参数：
  - CommandKey：  与 Download 请求中的 CommandKey 对应
  - FaultCode：   0 = 成功，非 0 = 失败错误码
  - FaultString： 失败时的错误描述
  - StartTime / CompleteTime：传输时间戳
    │
    ▼
TR069Server 处理 TransferComplete
  - 从 CommandKey 提取 uuid，从 Redis 查找原始请求信息
  - 将结果写入 Redis：
      Key：{uuid}
      Value：{result: {response, faultCode, faultString}, complete: "1"}
    │
    ▼
更新数据库任务状态
  - 表：license_task_record
  - task_progress：0→1→2
  - FaultCode=0 → 成功；FaultCode≠0 → 失败
    │
    ▼
前端查询展示下发结果
```

---

## 六、任务状态流转

```
任务创建
  task_progress = 0（待执行）
        │
        ▼ 队列触发执行
  task_progress = 1（执行中）
        │
    ┌───┴───┐
    ▼       ▼
  成功      失败
（FaultCode=0）（FaultCode≠0）
        │
        ▼
  task_progress = 2（已完成）
```

| task_progress | 含义 |
|---|---|
| 0 | 待执行 |
| 1 | 执行中 |
| 2 | 已完成（含成功/失败） |

---

## 七、核心数据库表

### 7.1 license_file_info（License 文件元数据）

| 字段 | 说明 |
|---|---|
| id | 主键 |
| sn | 基站序列号 |
| file_name | 文件名（{SN}.lic） |
| file_path | 服务器存储路径 |
| md5 | 文件 MD5 |
| upload_time | 上传时间 |
| operator_code | 运营商编码 |

### 7.2 license_task_record（下发任务记录）

| 字段 | 说明 |
|---|---|
| task_id | 主键，任务 ID |
| sn | 基站序列号 |
| file_name | License 文件名 |
| task_progress | 任务进度（0/1/2） |
| fault_code | TR-069 返回错误码 |
| fault_string | 错误描述 |
| create_time | 创建时间 |
| complete_time | 完成时间 |

---

## 八、TR-069 交互详解

### 8.1 Download RPC（OMC → 设备）

| 参数 | 值 |
|---|---|
| CommandKey | `"Download License,{uuid}"` |
| FileType | `"License File"` 或 `"1588 License"` |
| URL | `http://{omc_ip}:{port}/{license_file_path}` |
| Protocol | HTTP |
| DelaySeconds | 10 |
| Md5（扩展） | 文件 MD5 值，供设备验证 |

### 8.2 TransferComplete 事件（设备 → OMC）

| 参数 | 说明 |
|---|---|
| CommandKey | 与 Download 请求 CommandKey 一致（用于结果关联） |
| FaultCode | `0` = 成功，非 `0` = 失败 |
| FaultString | 失败原因描述 |
| StartTime | 传输开始时间 |
| CompleteTime | 传输完成时间 |

---

## 九、License 信息同步机制

下发成功后，OMC 系统内也需同步更新 License 状态：

- **定时刷新**：cellHandler 中的 `LicenseUpdateTimer` 每 30 分钟执行一次，从 Redis 刷新 License 缓存信息。
- **全局缓存**：`CurrOMCLicInfo` 保存当前有效的 License 信息（productType、eNBMaxNum、各项 capability 开关等）。
- **Redis 结构**：`licenseInfo`（Hash），由 licenseReader 服务写入，cellHandler 读取使用。

---

## 十、完整交互时序

```
用户         OMCWebServer    licenseReader   cellHandler    TR069Server     基站设备
 │               │                │              │               │              │
 │── 上传文件 ──→│                │              │               │              │
 │               │── 验证License→│              │               │              │
 │               │◄── 提取信息 ──│              │               │              │
 │               │── 写入DB ─────────────────→│               │              │
 │◄── 返回 ──────│                │              │               │              │
 │               │                │              │               │              │
 │── 创建任务 ──→│                │              │               │              │
 │               │─── 创建任务记录，加入队列 ──→│               │              │
 │◄── 返回 ──────│                │              │               │              │
 │               │                │              │               │              │
 │               │                │   任务队列触发│               │              │
 │               │                │              │── RPC调用 ───→│              │
 │               │                │              │  构建Download │              │
 │               │                │              │               │── Download ─→│
 │               │                │              │               │   RPC        │
 │               │                │              │               │              │
 │               │                │              │               │  [设备下载   │
 │               │                │              │               │   并激活]    │
 │               │                │              │               │              │
 │               │                │              │◄── TransferComplete ────────│
 │               │                │              │               │              │
 │               │◄────────────── 更新任务状态 ──│               │              │
 │               │                │              │               │              │
 │── 查询结果 ──→│                │              │               │              │
 │◄── 返回 ──────│                │              │               │              │
```

---

## 十一、Web REST API 端点

| 接口路径 | 方法 | 说明 |
|---|---|---|
| `/cell/license/uploadLicenseFile` | POST | 上传 License 文件到 OMC |
| `/cell/license/addLicenseTask` | POST | 创建 License 下发任务 |
| `/cell/license/queryLicenseFileList` | POST | 查询已上传的 License 文件列表 |
| `/cell/license/queryLicenseTaskList` | POST | 查询下发任务列表 |
| `/cell/license/deleteLicenseFile` | POST | 删除已上传的 License 文件 |

---

*文档生成时间：2026-05-18*

# 参数模型上传与解析 — 实施方案

> **日期**: 2026-03-24
> **状态**: ✅ 已实施
> **影响范围**: F01(ACS) / F02(Config/DataModel) / F09(Provision)

---

## 1. 背景与动机

### 原方案的问题

原参数模型获取依赖 **GPN (GetParameterNames) 逐级发现**：

| 问题 | 说明 |
|------|------|
| **慢** | 需要多轮 GPN 请求，每个子对象一次 RPC，大型参数树需要 50-100+ 轮 |
| **信息贫乏** | GPN 只返回 `path` + `writable`，无法获取类型、约束、通知策略等 |
| **不稳定** | 跨会话发现容易中断（设备断连/重连），需要复杂的停滞检测和恢复机制 |
| **不精确** | 多实例过滤只保留最小编号实例，可能遗漏结构不同的实例 |

### 新方案

基站支持 **TR-069 Upload RPC (FileType "11")** 上传完整参数模型 XML。

该 XML 包含完整的 objects 和 parameters 定义，字段远比 GPN 丰富：

```xml
<parameterModel generateTime="..." vendor="48BF74" networkType="LTE"
                serialNumber="..." modelVersion="1.0" totalEntries="2096">
    <objects>
        <object name="Device.Services.FAPService.{12}."
                access="READ_WRITE" maxInstances="0" isList="false" />
    </objects>
    <parameters>
        <param name="Device.DeviceInfo.SoftwareVersion"
               access="READ_ONLY" type="STRING" min="0" max="64"
               notify="ACTIVE_NOTIFICATION" forcedInform="true"
               changeApplies="Immediate" isList="false" />
    </parameters>
</parameterModel>
```

**优势**：
- **一次上传，完整模型** — 不再需要多轮 GPN
- **丰富元数据** — 类型、值域约束(min/max)、通知策略、是否强制上报、默认值等
- **可靠** — 文件传输有 TransferComplete 确认
- **标准化** — XML 格式可同时用于界面手动导入

**GPN 发现代码已完全移除**，不再作为降级方案。

---

## 2. 整体架构

### 2.1 两种获取参数模型的路径

```
路径 A：自动上传（Provision 流程触发）
┌─────────┐    ┌──────────┐    ┌────────┐    ┌──────────┐    ┌──────────┐
│ 模型匹配  │───→│ 下发Upload│───→│ CPE上传 │───→│ 解析XML  │───→│ 入库激活  │
│ 失败      │    │ RPC命令   │    │ XML文件 │    │ 参数模型  │    │ DataModel│
└─────────┘    └──────────┘    └────────┘    └──────────┘    └──────────┘

路径 B：界面手动导入
┌─────────┐    ┌──────────┐    ┌──────────┐
│ 用户上传  │───→│ 解析XML  │───→│ 入库(草稿)│
│ XML文件   │    │ 参数模型  │    │ DataModel│
└─────────┘    └──────────┘    └──────────┘
```

### 2.2 跨进程数据流

```
App 进程                          ACS 进程                        CPE
   │                                │                              │
   │ 1. 模型匹配失败                │                              │
   │ 2. 入队 Upload 命令            │                              │
   │ ──────(cmdQueue)──────────────→│                              │
   │                                │ 3. 发送 Upload SOAP RPC      │
   │                                │──────────────────────────────→│
   │                                │                              │
   │                                │ 4. CPE HTTP POST 上传 XML    │
   │                                │←─────────────────────────────│
   │                                │ 5. 存储到 MinIO              │
   │                                │ 6. 发布 datamodel.file.received
   │←──────(NATS EventBus)─────────│                              │
   │ 7. 从 MinIO 下载 XML          │                              │
   │ 8. 解析 → 创建 DataModel      │                              │
   │ 9. 自动激活 + 缓存失效         │                              │
```

### 2.3 核心组件

```
omcgo/internal/config/datamodel/
├── model.go            # ✅ 新增 SourceCPEUploaded, 扩展 Parameter 字段, ObjectInfo
├── xml_parser.go       # ✅ XML 解析器，将 XML → ParsedParameterModel
├── xml_parser_test.go  # ✅ 解析器单元测试
├── importer.go         # ✅ 新增 ImportFromXML / ImportFromXMLForCPE 方法
├── handler.go          # ✅ 新增 POST /datamodels/import-xml 端点
├── pg_repository.go    # ✅ 适配 object_tree, model_metadata 新列

omcgo/internal/provision/
├── model_upload.go     # ✅ ModelUploadService — Upload RPC 调度 + 文件接收处理
├── engine.go           # ✅ Path C 改为 model upload，订阅 datamodel.file.received
├── discovery.go        # ❌ 已删除（旧 GPN 发现代码）
├── discovery_test.go   # ❌ 已删除

omcgo/internal/acs/upload/
├── handler.go          # ✅ 识别 FileType "11" → datamodel 分类 + 发布事件

omcgo/internal/transfer/
├── bridge.go           # ✅ 处理 AutonomousTransferComplete 中的 datamodel 文件

omcgo/internal/core/
├── event/subjects.go   # ✅ 新增 datamodel.upload.* 和 datamodel.file.received 事件
├── appconfig/config.go # ✅ 新增 ModelUploadConfig，替换 AutoDiscoveryConfig

omcgo/cmd/acs/main.go   # ✅ 初始化 upload.Handler 并注入 eventBus

omcgo/migrations/
├── 000059_datamodel_extend_xml_upload.up.sql   # ✅ 扩展 source_type, 新增列
├── 000059_datamodel_extend_xml_upload.down.sql # ✅ 回滚迁移
```

---

## 3. 配置项

```yaml
# app.yaml
provision:
  enabled: true
  auto_configure: false
  task_timeout: 15m
  model_upload:
    enabled: true
    upload_url: "http://acs:7547/smallcell/FileUploadService"  # ACS 上传端点
    upload_username: "upload_user"     # CPE 上传认证用户名
    upload_password: "upload_pass"     # CPE 上传认证密码
    auto_activate_model: true          # 自动激活上传的模型
    upload_timeout: 5m                 # Upload RPC 超时时间
  auto_sync:
    enabled: true
    gpv_batch_size: 50                 # GPV 批量大小（从旧 auto_discovery 迁移）
```

---

## 4. Provision 四路分支（更新后）

| 路径 | 条件 | 行为 |
|------|------|------|
| **Path A** | 模板匹配成功 + auto_configure=true | 经典配置下发流程 |
| **Path B** | DataModel 存在 + auto_sync=true | 同步设备参数值 |
| **Path C** | DataModel 不存在 + model_upload=true | 下发 Upload RPC → CPE 上传 XML → 解析入库 |
| **Path D** | 无匹配条件 | 任务失败 |

---

## 5. 关键数据结构

### parameter_tree JSON 格式（XML 上传，向后兼容）

```json
[
  {
    "path": "Device.DeviceInfo.SoftwareVersion",
    "writable": false,
    "type": "string",
    "constraints": {"max_length": 64},
    "notify": "ACTIVE_NOTIFICATION",
    "forced_inform": true,
    "change_applies": "Immediate"
  }
]
```

### object_tree JSON 格式

```json
[
  {
    "name": "Device.Services.FAPService.{12}.",
    "access": "readWrite",
    "max_instances": 0,
    "is_list": false
  }
]
```

---

## 6. 事件定义

```go
// DataModel events
SubjectDataModelUploadRequested = "datamodel.upload.requested"
SubjectDataModelUploadCompleted = "datamodel.upload.completed"
SubjectDataModelUploadFailed    = "datamodel.upload.failed"
SubjectDataModelFileReceived    = "datamodel.file.received"
```

`datamodel.file.received` 事件 payload:
```json
{
  "minio_bucket": "logs",
  "minio_path": "datamodel/2026/03/24/device_sn/datamodel_xxx.xml",
  "device_sn": "ABC123",
  "device_id": "uuid",
  "carrier": "cmcc",
  "technology": "lte",
  "oui": "48BF74",
  "product_class": "SC100",
  "firmware_version": "1.0.0",
  "file_size": 437000
}
```

---

## 7. 已删除的旧代码

以下 GPN 发现相关代码已完全删除：

- `internal/provision/discovery.go` — 645 行 GPN 逐级发现逻辑
- `internal/provision/discovery_test.go` — GPN 辅助函数测试
- `engine.go` 中的 `handleGPNResponse()` 和 GPN 订阅
- `engine.go` 中的 `handleAutoDiscovery()` → `StartDiscovery()`
- `appconfig` 中的 `AutoDiscoveryConfig`（替换为 `ModelUploadConfig`）

保留的兼容代码：
- `model.go` 中的 `StateDiscovering` 状态 — 现在用于模型上传等待
- `ParameterDiscoveryLog` 和相关 repository — 用于记录模型获取日志
- `DiscoveryStatus` 常量 — 日志状态跟踪

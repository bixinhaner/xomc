# OMC 前端→App→Worker/ACS RPC 全链路分析报告

> 分析日期：2026-03-22
> 分析范围：前端 UI → App REST API → Redis 命令队列 → ACS TR-069 引擎 → CPE 设备
> 参与专家：架构专家、TR-069 协议栈专家、Go 工程专家、前端专家

---

## 一、架构总览

系统采用 **队列解耦架构**（非 gRPC），三个部署单元通过 Redis + NATS JetStream 通信：

```
┌─────────────────────┐     ┌──────────────────────────────┐     ┌───────────────────────┐
│  前端 (React SPA)    │     │  omcgo-app :8080 (REST API)  │     │  omcgo-worker         │
│  ├─ deviceApi.ts     │────▶│  ├─ device handler           │     │  ├─ PM 文件处理        │
│  ├─ softwareApi.ts   │     │  ├─ config sync handler      │     │  ├─ MR 文件处理        │
│  ├─ configSyncApi.ts │     │  ├─ software handler         │     │  ├─ KPI 计算           │
│  ├─ opsToolsApi.ts   │     │  ├─ task handler             │     │  ├─ 告警关联           │
│  └─ configBaselineApi│     │  └─ ops handler              │     │  └─ 报表生成           │
└─────────────────────┘     └──────────┬───────────────────┘     └───────────┬───────────┘
                                       │                                     │
                                       ▼                                     │
                            ┌──────────────────────┐                         │
                            │  Redis               │                         │
                            │  ├─ acs:cmdq:{sn}    │◀────────────────────────┘
                            │  ├─ acs:session:{sn}  │
                            │  └─ tasks:*           │
                            └──────────┬───────────┘
                                       │
                                       ▼
                            ┌──────────────────────┐
                            │  omcgo-acs :7547      │
                            │  ├─ Inform 处理       │
                            │  ├─ 命令队列弹出      │
                            │  ├─ SOAP/XML 构建     │
                            │  ├─ RPC Dispatcher    │
                            │  └─ 会话状态机        │
                            └──────────┬───────────┘
                                       │
                                       ▼
                            ┌──────────────────────┐
                            │  CPE 设备 (LTE/5G)    │
                            │  (基站/皮基站/微基站)   │
                            └──────────────────────┘
```

**关键设计决策**：不使用 gRPC，而是通过 Redis Sorted Set 命令队列解耦 App 与 ACS。原因：
1. TR-069 是 CPE 发起的会话模型，命令只能在 CPE 连接时下发
2. 队列天然支持持久化、负载均衡、崩溃恢复
3. ACS 处理 SOAP/XML 需要完全控制 HTTP，与 gRPC 代码生成不兼容

---

## 二、ACS 层 TR-069 RPC 实现状态

### 2.1 ACS→CPE RPC 方法（11 个，全部实现）

| # | RPC 方法 | 实现文件 | Handler | 用途 | 状态 |
|---|---------|---------|---------|------|------|
| 1 | `GetParameterValues` | `rpc/dispatcher.go` | `GetParameterValuesHandler` | 读取设备参数 | ✅ 完成 |
| 2 | `SetParameterValues` | `rpc/dispatcher.go` | `SetParameterValuesHandler` | 写入设备参数 | ✅ 完成 |
| 3 | `GetParameterNames` | `rpc/dispatcher.go` | `GetParameterNamesHandler` | 发现参数树 | ✅ 完成 |
| 4 | `GetParameterAttributes` | `rpc/dispatcher.go` | `GetParameterAttributesHandler` | 读取参数属性（通知/访问控制）| ✅ 完成 |
| 5 | `SetParameterAttributes` | `rpc/dispatcher.go` | `SetParameterAttributesHandler` | 设置参数通知属性 | ✅ 完成 |
| 6 | `AddObject` | `rpc/dispatcher.go` | `AddObjectHandler` | 创建多实例对象 | ✅ 完成 |
| 7 | `DeleteObject` | `rpc/dispatcher.go` | `DeleteObjectHandler` | 删除多实例对象 | ✅ 完成 |
| 8 | `Download` | `rpc/dispatcher.go` | `DownloadHandler` | 固件/配置下载到设备 | ✅ 完成 |
| 9 | `Upload` | `rpc/dispatcher.go` | `UploadHandler` | 设备上传文件（PM/MR/日志）| ✅ 完成 |
| 10 | `Reboot` | `rpc/dispatcher.go` | `RebootHandler` | 设备重启 | ✅ 完成 |
| 11 | `FactoryReset` | `rpc/dispatcher.go` | `FactoryResetHandler` | 恢复出厂设置 | ✅ 完成 |

### 2.2 CPE→ACS 通知方法（3 个，全部实现）

| # | 方法 | 处理函数 | 用途 | 状态 |
|---|------|---------|------|------|
| 1 | `Inform` | `handleInform()` | 设备上报状态/事件 | ✅ 完成 |
| 2 | `TransferComplete` | `handleTransferComplete()` | 文件传输完成通知 | ✅ 完成 |
| 3 | `AutonomousTransferComplete` | `handleAutonomousTransferComplete()` | CPE 主动上传完成（PM/MR）| ✅ 完成 |

### 2.3 TR-069 标准 RPC 覆盖度

| TR-069 标准 RPC | 本系统 | 说明 |
|----------------|--------|------|
| GetRPCMethods | ❌ 未实现 | 可选方法，用于协商能力，优先级低 |
| ScheduleInform | ❌ 未实现 | 可选方法，安排 CPE 定时上报 |
| SetVouchers | ❌ 未实现 | 可选方法，用于 Option Management，极少使用 |
| GetOptions | ❌ 未实现 | 可选方法，用于 Option Management，极少使用 |
| GetQueuedTransfers | ❌ 未实现 | 可选方法，查询排队中的传输任务 |
| GetAllQueuedTransfers | ❌ 未实现 | 可选方法，同上的完整版本 |
| ScheduleDownload | ❌ 未实现 | 可选方法，定时下载 |
| CancelTransfer | ❌ 未实现 | 可选方法，取消传输 |
| ChangeDUState | ❌ 未实现 | 可选方法，部署单元管理（用于软件模块安装） |

> **结论**：11 个核心必选 RPC 方法 + 3 个通知方法全部实现。8 个可选方法未实现，其中大部分在小基站场景中不需要。`GetRPCMethods` 和 `ScheduleInform` 可考虑后续补充。

---

## 三、Inform 事件码支持状态

### 3.1 标准事件码（14 个，全部支持）

| 事件码 | 常量 | 含义 | 处理 | 状态 |
|--------|------|------|------|------|
| 0 BOOTSTRAP | `EventBootstrap` | 首次启动/初始配置 | 发布 `domain.device.bootstrap` 事件 | ✅ |
| 1 BOOT | `EventBoot` | 设备启动完成 | 默认处理 | ✅ |
| 2 PERIODIC | `EventPeriodic` | 周期性上报 | 发布 `domain.device.periodic` 事件 | ✅ |
| 3 SCHEDULED | `EventScheduled` | 定时上报 | 默认处理 | ✅ |
| 4 VALUE CHANGE | `EventValueChange` | 参数值变更 | 发布 `domain.device.value_change` 事件 | ✅ |
| 5 KICKED | `EventKicked` | ACS 踢出后重连 | 默认处理 | ✅ |
| 6 CONNECTION REQUEST | `EventConnectionRequest` | CPE 主动连接 | 发布 `domain.device.connection_request` 事件 | ✅ |
| 7 TRANSFER COMPLETE | `EventTransferComplete` | 文件传输完成 | 发布 `domain.device.transfer_complete` 事件 | ✅ |
| 8 DIAGNOSTICS COMPLETE | `EventDiagnosticsComplete` | 诊断完成 | 默认处理 | ✅ |
| 9 REQUEST DOWNLOAD | `EventRequestDownload` | CPE 请求下载 | 默认处理 | ✅ |
| 10 AUTONOMOUS TRANSFER COMPLETE | `EventAutonomousTransferComplete` | CPE 主动上传完成 | 专门处理函数 | ✅ |
| M Reboot | `EventMReboot` | 重启命令完成 | 发布 `domain.device.reboot_complete` 事件 | ✅ |
| M Download | `EventMDownload` | 下载命令完成 | 默认处理 | ✅ |
| M Upload | `EventMUpload` | 上传命令完成 | 默认处理 | ✅ |

### 3.2 运营商扩展事件码

| 事件码 | 含义 | 运营商 | 状态 |
|--------|------|--------|------|
| 103 | AddObject 完成 | CMCC | ✅ |
| 104 | DeleteObject 完成 | CMCC | ✅ |
| 告警事件 | 设备告警 | CMCC | ✅ `IsAlarm()` |

### 3.3 辅助函数

```go
IsBootstrap(events)              // 检查 0 BOOTSTRAP
IsBoot(events)                   // 检查 1 BOOT
IsPeriodic(events)               // 检查 2 PERIODIC
IsValueChange(events)            // 检查 4 VALUE CHANGE
IsConnectionRequest(events)      // 检查 6 CONNECTION REQUEST
IsRebootComplete(events)         // 检查 M Reboot
IsDownloadComplete(events)       // 检查 M Download
IsAutonomousTransferComplete()   // 检查 10
HasEvent(events, code)           // 通用检查
IsAlarm(events)                  // 检查 CMCC 告警码
```

> **结论**：全部 14 个标准事件码 + CMCC 扩展事件码均已实现，并配备完整的辅助判断函数。

---

## 四、App 层设备命令 API 实现状态

### 4.1 REST API 端点

| 端点 | 方法 | 底层 RPC | 模块 | 状态 |
|------|------|---------|------|------|
| `/api/v1/devices/:id/reboot` | POST | Reboot | device | ✅ 完成 |
| `/api/v1/devices/tasks` | POST | 任意 RPC | task | ✅ 完成 |
| `/api/v1/devices/tasks` | GET | — | task | ✅ 完成 |
| `/api/v1/devices/tasks/:id` | GET | — | task | ✅ 完成 |
| `/api/v1/devices/tasks/:id` | DELETE | — | task | ✅ 完成 |
| `/api/v1/devices/tasks/:id/retry` | POST | 任意 RPC | task | ✅ 完成 |
| `/api/v1/devices/tasks/batch` | POST | 批量 RPC | task | ✅ 完成 |
| `/api/v1/devices/tasks/stats` | GET | — | task | ✅ 完成 |
| `/api/v1/config/sync/push/:deviceId` | POST | SetParameterValues | config | ✅ 完成 |
| `/api/v1/config/sync/pull/:deviceId` | POST | GetParameterValues | config | ✅ 完成 |
| `/api/v1/config/sync/status/:deviceId` | GET | — | config | ✅ 完成 |
| `/api/v1/upgrade-tasks` | POST | Download | software | ✅ 完成 |
| `/api/v1/upgrade-tasks/batch` | POST | 批量 Download | software | ✅ 完成 |
| `/api/v1/firmware` | POST | — | software | ✅ 完成 |

### 4.2 命令队列机制

```
App 层创建命令                Redis Sorted Set               ACS 层消费命令
─────────────────            ──────────────                ─────────────────
Command{                     acs:cmdq:{device_sn}          CPE Inform 到达
  Method: "Reboot"           Score = Priority + Timestamp   ├─ Pop 最高优先级命令
  Priority: 0                                               ├─ Dispatcher 构建 SOAP
  Params: {...}              ◄───────────────────────────    ├─ 发送给 CPE
  ExpiresAt: ...             ────────────────────────────►   ├─ 等待 RPC Response
}                                                           └─ 更新 Task 状态
```

### 4.3 Connection Request 客户端

文件：`internal/acs/connreq/client.go`

用于主动唤醒 CPE（不等周期 Inform）：
- HTTP GET 到 CPE 的 ConnectionRequestURL
- 支持 HTTP Digest 认证
- 指数退避重试（1s, 2s, 4s）
- Redis 去重（30s TTL）

---

## 五、Worker 进程任务处理

| 订阅者 | 监听事件 | 处理逻辑 | 状态 |
|--------|---------|---------|------|
| PM Collector | `pm.file.received` | 解析 PM 文件、计算 KPI | ✅ 完成 |
| Alarm Receiver | `alarm.raised` | 告警去重、关联、升级 | ✅ 完成 |
| MR Collector | `mr.file.received` | 解析 MRO/MRS/MRE 文件 | ✅ 完成 |
| Transfer Bridge | `device.inform.*` | 协调文件传输 | ✅ 完成 |
| Backup Executor | `backup.task.created` | 执行配置备份 | ✅ 完成 |
| Report Generator | `report.generate` | 生成周期性报表 | ✅ 完成 |

---

## 六、前端→后端 RPC 操作集成状态

### 6.1 完全集成（前端 API + Hook + UI 页面）

| 操作 | 前端 API | Hook | UI 页面 | 后端端点 | 底层 RPC |
|------|---------|------|---------|---------|---------|
| 参数读取 | `configSyncApi.pullConfig` | `useSyncConfig` | LiveParamConfig | `/config/sync/pull` | GetParameterValues |
| 参数写入 | `configSyncApi.pushConfig` | `useUpdateConfigParam` | LiveParamConfig | `/config/sync/push` | SetParameterValues |
| 固件上传 | `softwareApi.uploadFirmware` | `useUploadFirmware` | FirmwareUpload | `/firmware` | — |
| 固件升级 | `softwareApi.createUpgradePlan` | `useCreateUpgradePlan` | UpgradePlan | `/upgrade-tasks/batch` | Download |
| 运维任务创建 | `opsToolsApi.createTask` | `useCreateOpsTask` | TaskManagement | `/ops/tasks` | 多种 |
| 运维任务控制 | `opsToolsApi.{cancel,pause,resume}` | 对应 hooks | TaskManagement | `/ops/tasks/:id/*` | — |
| 配置基线 | `configBaselineApi.createTask` | `useCreateConfigTask` | BatchParamClass | `/config/tasks` | SetParameterValues |

### 6.2 部分集成（API 存在但 Hook/UI 缺失）

| 操作 | 前端 API | Hook | UI 页面 | 问题 |
|------|---------|------|---------|------|
| 设备重启 | ✅ `deviceApi.reboot(id)` | ✅ `useRebootDevice` + `useBatchRebootDevices` | ✅ 批量重启已接通 | **已修复** — hook 已封装，DeviceList 批量重启调用真实 API |
| 恢复出厂 | ❌ 未定义 | ❌ 无 | ❌ 批量操作 TODO | 后端 ACS 支持 FactoryReset RPC，但 App 层和前端均未暴露 |

### 6.3 未集成（后端支持但前端未涉及）

| 操作 | 后端 RPC | App 端点 | 前端状态 | 优先级 |
|------|---------|---------|---------|--------|
| GetParameterNames | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 中 — 参数树发现 |
| GetParameterAttributes | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 低 |
| SetParameterAttributes | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 低 |
| AddObject | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 中 — 多实例管理 |
| DeleteObject | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 中 |
| Upload（日志/配置） | ✅ ACS 已实现 | 可通过 task API 创建 | ❌ 无专用 API/UI | 中 — 日志采集 |

### 6.4 前端 Mock 委托（后端未对接）

| 操作 | 前端文件 | 问题 |
|------|---------|------|
| 升级预检查 `precheck` | `softwareApi.ts:308` | 委托到 mock，返回假数据 |
| 取消升级计划 `cancelUpgradePlan` | `softwareApi.ts:307` | 委托到 mock |

### 6.5 前端批量操作 TODO

DeviceList 页面定义了以下批量操作，但均标记 `TODO: 接入对应批量操作 API`：

| 批量操作 | i18n Key | 实际调用 |
|---------|---------|---------|
| 批量同步 | `common.batchSync` | ❌ 未实现 |
| 批量重启 | `batch-reboot` | ❌ 未实现 |
| TR069 数据采集 | `batch-tr069-collect` | ❌ 未实现 |
| 日志采集 | `batch-log-collect` | ❌ 未实现 |
| 重置配置 | `batch-reset-config` | ❌ 未实现 |

---

## 七、会话状态机与 SOAP 处理

### 7.1 会话状态机（6 状态，完整实现）

```
IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE
                              ↑              │              │
                              └──────────────┴──────────────┘
                                   （有更多命令时循环）
```

### 7.2 SOAP 模板系统（完整实现）

| 模板 | 类型 | 用途 |
|------|------|------|
| `InformResponseTmpl` | 响应 | 确认 CPE Inform |
| `GetParameterValuesTmpl` | 请求 | 查询参数 |
| `SetParameterValuesTmpl` | 请求 | 设置参数 |
| `GetParameterNamesTmpl` | 请求 | 发现参数树 |
| `GetParameterAttributesTmpl` | 请求 | 读取参数属性 |
| `SetParameterAttributesTmpl` | 请求 | 设置参数属性 |
| `AddObjectTmpl` | 请求 | 创建多实例 |
| `DeleteObjectTmpl` | 请求 | 删除多实例 |
| `DownloadTmpl` | 请求 | 固件/配置下载 |
| `UploadTmpl` | 请求 | PM/MR/日志上传 |
| `RebootTmpl` | 请求 | 设备重启 |
| `FactoryResetTmpl` | 请求 | 恢复出厂 |
| `FaultResponseTmpl` | 响应 | SOAP 错误 |

### 7.3 SOAP 解码器（完整实现）

- `DecodeInform` — 解析 Inform 消息
- `DecodeTransferComplete` — 解析传输完成通知
- `DecodeAutonomousTransferComplete` — 解析自主传输完成
- `DecodeGetParameterValuesResponse` — 解析参数值响应
- `DecodeSetParameterValuesResponse` — 解析参数设置响应
- 其他所有 RPC 响应解码器

---

## 八、可观测性支持

### Prometheus 指标

| 指标 | 类型 | 标签 |
|------|------|------|
| `acs_inform_total` | Counter | `event_code` |
| `acs_rpc_duration_seconds` | Histogram | `method` |
| `acs_rpc_errors_total` | Counter | `method` |
| `acs_active_sessions` | Gauge | — |
| `acs_session_duration_seconds` | Histogram | — |
| `acs_rate_limit_rejected_total` | Counter | — |

### 事件总线（NATS JetStream）

**Inform 事件主题**：`domain.device.{bootstrap|periodic|alarm|reboot_complete|connection_request|transfer_complete|value_change}`

**RPC 响应主题**：`domain.command.{get_params|set_params|download|upload|get_names|add_object|delete_object|reboot|factory_reset}_response`

---

## 九、总结评估

### 完成度评分

| 层次 | 完成度 | 说明 |
|------|--------|------|
| **ACS 引擎（TR-069 协议）** | ★★★★★ 95% | 11 个核心 RPC + 3 个通知全部实现，14 个事件码全部处理，8 个可选 RPC 未实现（合理省略） |
| **App 层（REST API + 命令队列）** | ★★★★☆ 85% | 核心端点完备，Task 管理系统完整，缺少 FactoryReset 专用端点和部分高级操作端点 |
| **Worker 进程** | ★★★★★ 90% | PM/MR/KPI/告警/备份/报表处理链路完整 |
| **前端集成** | ★★★☆☆ 68% | 参数读写、固件升级和批量重启已贯通，但 4 个批量操作 TODO、恢复出厂 hook 缺失、2 个操作仍委托 mock |
| **全链路贯通** | ★★★★☆ 82% | 核心业务流（参数配置、固件升级、运维任务、设备重启）已贯通，设备控制类操作（复位、日志采集）前端最后一公里未完成 |

### 建议优先级

| 优先级 | 待办项 | 工作量 | 影响 |
|--------|--------|--------|------|
| ~~P0 高~~ | ~~补充 `useRebootDevice` hook，接通 DeviceList 批量重启~~ | ~~0.5 天~~ | ✅ **已完成 (2026-03-22)** |
| **P0 高** | 补充 FactoryReset App 端点 + 前端 API/hook | 1 天 | 设备恢复必备功能 |
| **P1 中** | 接通 DeviceList 其余 4 个批量操作（同步/采集/日志/重置） | 2 天 | 批量运维效率 |
| **P1 中** | 实现 `softwareApi.precheck` 和 `cancelUpgradePlan` 后端接口，替换 mock | 1 天 | 升级安全性 |
| **P2 低** | 前端暴露 GetParameterNames/AddObject/DeleteObject 操作 | 2 天 | 高级配置管理 |
| **P2 低** | 前端暴露 Upload 操作（日志/配置上传） | 1 天 | 运维日志采集 |
| **P3 可选** | 实现 GetRPCMethods 和 ScheduleInform | 1 天 | 协议完整性 |
| **P3 可选** | 实现 ScheduleDownload/CancelTransfer | 1 天 | 高级传输管理 |

---

## 十、关键文件索引

### ACS 引擎

| 文件 | 职责 |
|------|------|
| `omcgo/internal/acs/handler.go` | ACS HTTP 处理器（1200+ 行，核心入口） |
| `omcgo/internal/acs/session.go` | 会话状态机 |
| `omcgo/internal/acs/rpc/dispatcher.go` | RPC 分发器（命令→SOAP） |
| `omcgo/internal/acs/cmdqueue/queue.go` | Redis 命令队列 |
| `omcgo/internal/acs/task_service.go` | Task 服务接口 |
| `omcgo/internal/acs/connreq/client.go` | Connection Request 客户端 |
| `omcgo/internal/acs/auth/authenticator.go` | CPE 认证（Basic/Digest） |
| `omcgo/internal/acs/ratelimit.go` | Per-device 速率限制 |
| `omcgo/internal/acs/admission.go` | 全局准入控制 |
| `omcgo/internal/acs/metrics.go` | Prometheus 指标 |
| `omcgo/pkg/soap/templates.go` | SOAP XML 模板 |
| `omcgo/pkg/soap/decoder.go` | SOAP XML 解码器 |
| `omcgo/pkg/tr069/events.go` | 事件码定义与辅助函数 |

### App 层

| 文件 | 职责 |
|------|------|
| `omcgo/internal/device/handler.go` | 设备管理 API（含 Reboot） |
| `omcgo/internal/device/service.go` | 设备业务逻辑 |
| `omcgo/internal/task/handler.go` | Task 管理 API |
| `omcgo/internal/task/service.go` | Task 业务逻辑 |
| `omcgo/internal/config/sync_handler.go` | 配置同步 API（push/pull） |
| `omcgo/internal/software/handler.go` | 固件管理 API |

### 前端

| 文件 | 职责 |
|------|------|
| `omcmb/webcode/src/services/api/deviceApi.ts` | 设备 API（含 reboot） |
| `omcmb/webcode/src/services/api/softwareApi.ts` | 固件 API（含 mock 委托） |
| `omcmb/webcode/src/services/api/configSyncApi.ts` | 配置同步 API |
| `omcmb/webcode/src/services/api/opsToolsApi.ts` | 运维工具 API |
| `omcmb/webcode/src/hooks/api/useDevices.ts` | 设备 hooks（缺 reboot） |
| `omcmb/webcode/src/hooks/api/useSoftware.ts` | 固件 hooks |
| `omcmb/webcode/src/hooks/api/useConfig.ts` | 配置 hooks |
| `omcmb/webcode/src/pages/device/DeviceList/index.tsx` | 设备列表（批量操作 TODO） |

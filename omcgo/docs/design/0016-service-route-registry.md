# OMC Go 服务路由注册文档

> 文档生成日期: 2026-03-20
> 分析范围: cmd/acs, cmd/app, cmd/worker

---

## 1. 服务概览

OMC Go 项目包含三个独立部署的服务进程：

| 服务 | 入口文件 | 端口 | 职责 |
|------|----------|------|------|
| **omcgo-acs** | `cmd/acs/main.go` | 7547 | TR069 ACS 引擎（南向接口） |
| **omcgo-app** | `cmd/app/main.go` | 8080 | 主应用 REST API（北向接口） |
| **omcgo-worker** | `cmd/worker/main.go` | - | 后台工作进程（事件驱动） |

---

## 2. omcgo-acs 服务

### 2.1 路由注册

**入口**: `cmd/acs/main.go`
**路由定义**: `internal/acs/server.go`

```go
mux := http.NewServeMux()
mux.HandleFunc("/smallcell/AcsService", h.ServeHTTP)
mux.HandleFunc("/healthz", healthHandler)
mux.Handle("/smallcell/upload/", uploadHandler)
```

### 2.2 路由表

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| POST | `/smallcell/AcsService` | `Handler.ServeHTTP` | TR069/CWMP SOAP 端点 |
| GET | `/healthz` | 内联函数 | 健康检查 |
| POST | `/smallcell/FileUploadService` | `upload.Handler` | CPE 文件上传（全局凭证认证） |

#### 文件上传

**URL 格式**：
```
POST /smallcell/FileUploadService?fileType=PM&filename=pm_20260320.xml
Authorization: Basic {base64(username:password)}
```

**认证方式**：HTTP Basic Authentication，使用配置文件中的全局用户名/密码

**参数说明**：
| 参数 | 必需 | 说明 |
|------|------|------|
| `fileType` | 是 | 文件类型：`PM`(4), `MR`(5), `Log`(6) |
| `filename` | 是 | 目标文件名 |
| `Authorization` | 是 | HTTP Basic Auth，用户名通常为设备 SN |

#### TR069 Upload RPC 下发

ACS 通过 TR069 Upload RPC 向 CPE 下发上传指令，包含：

```xml
<cwmp:Upload>
  <CommandKey>upload-20260320-001</CommandKey>
  <FileType>4</FileType>
  <URL>http://172.21.175.129:8080/smallcell/FileUploadService?fileType=PM&filename=pm.xml</URL>
  <Username>{设备SN}</Username>
  <Password>{全局密码}</Password>
  <DelaySeconds>0</DelaySeconds>
</cwmp:Upload>
```

#### MinIO 存储路径结构

```
{bucket}/
├── pm/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}     # PM 文件
├── mr/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}     # MR 文件
└── logs/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}   # 日志文件
```

**示例**：
```
pm-files/pm/2026/03/20/ABC123456/pm_20260320_100000.xml
```

### 2.3 TR069 消息处理

| 消息类型 | 处理方法 | 说明 |
|----------|----------|------|
| Inform | `handleInform` | 设备注册/心跳 |
| Empty POST | `handleEmpty` | 会话继续，发送 RPC |
| GetParameterValuesResponse | `handleRPCResponse` | 参数读取响应 |
| SetParameterValuesResponse | `handleRPCResponse` | 参数设置响应 |
| DownloadResponse | `handleRPCResponse` | 下载响应 |
| UploadResponse | `handleRPCResponse` | 上传响应 |
| TransferComplete | `handleTransferComplete` | 传输完成通知 |
| AutonomousTransferComplete | `handleAutonomousTransferComplete` | 自主传输完成 |

---

## 3. omcgo-app 服务

### 3.1 路由注册

**入口**: `cmd/app/main.go`
**路由定义**: `cmd/app/router/router.go`

#### 中间件链

```
RequestID → CORS → RequestLogger → PrometheusMetrics
      ↓
[Public Routes]
      ↓
JWT Auth → Carrier Context → Audit Logger
      ↓
[Protected Routes]
      ↓
Admin Permission Check
      ↓
[Admin Routes]
```

#### 公共路由（无需认证）

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/healthz` | 内联函数 | 健康检查 |
| POST | `/api/v1/auth/login` | `adminHandler.Login` | 用户登录 |
| POST | `/api/v1/auth/refresh` | `adminHandler.Refresh` | 刷新 Token |

#### 受保护路由（需要 JWT 认证）

所有 `/api/v1/*` 路由需要 JWT Token 认证。

---

### 3.2 模块路由详情

#### 3.2.1 设备管理模块 (`/api/v1/devices`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/devices` | `ListDevices` | 设备列表（分页、过滤） |
| GET | `/devices/stats` | `GetStats` | 设备统计 |
| GET | `/devices/:id` | `GetDevice` | 获取设备详情 |
| GET | `/devices/:id/parameters` | `GetDeviceParameters` | 获取 TR069 参数 |
| POST | `/devices` | `CreateDevice` | 创建设备 |
| PUT | `/devices/:id` | `UpdateDevice` | 更新设备 |
| DELETE | `/devices/:id` | `DeleteDevice` | 删除设备 |
| POST | `/devices/:id/reboot` | `RebootDevice` | 重启设备 |

**查询参数**:
- `carrier`: 运营商过滤
- `technology`: 制式过滤 (lte/nr)
- `status`: 状态过滤
- `oui`: OUI 过滤
- `sn`: 序列号搜索
- `search`: 关键字搜索

---

#### 3.2.2 用户认证模块 (`/api/v1/auth`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/auth/me` | `adminHandler.Me` | 获取当前用户信息 |

---

#### 3.2.3 用户管理模块 (`/api/v1/admin/users`)

> 需要 `users:admin` 权限

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/admin/users` | `ListUsers` | 用户列表 |
| POST | `/admin/users` | `CreateUser` | 创建用户 |
| GET | `/admin/users/:id` | `GetUser` | 用户详情 |
| PUT | `/admin/users/:id` | `UpdateUser` | 更新用户 |
| DELETE | `/admin/users/:id` | `DeleteUser` | 删除用户 |
| POST | `/admin/users/:id/roles` | `AssignRole` | 分配角色 |
| DELETE | `/admin/users/:id/roles/:roleId` | `RemoveRole` | 移除角色 |
| POST | `/admin/users/:id/reset-password` | `ResetPassword` | 重置密码 |
| POST | `/admin/users/:id/lock` | `LockUser` | 锁定用户 |
| POST | `/admin/users/:id/unlock` | `UnlockUser` | 解锁用户 |

---

#### 3.2.4 角色管理模块 (`/api/v1/admin/roles`)

> 需要 `users:admin` 权限

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/admin/roles` | `ListRoles` | 角色列表 |
| GET | `/admin/roles/:id` | `GetRole` | 角色详情 |
| POST | `/admin/roles` | `CreateRole` | 创建角色 |
| PUT | `/admin/roles/:id` | `UpdateRole` | 更新角色 |
| DELETE | `/admin/roles/:id` | `DeleteRole` | 删除角色 |
| GET | `/admin/permissions` | `ListPermissions` | 权限列表 |
| GET | `/admin/audit-logs` | `ListAuditLogs` | 审计日志 |

---

#### 3.2.5 数据模型模块 (`/api/v1/datamodels`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/datamodels` | `List` | 数据模型列表 |
| GET | `/datamodels/resolve` | `Resolve` | 三级回退解析 |
| GET | `/datamodels/statistics` | `Statistics` | 统计信息 |
| POST | `/datamodels` | `Create` | 创建数据模型 |
| POST | `/datamodels/import` | `Import` | JSON 导入 |
| POST | `/datamodels/cache/refresh` | `RefreshCache` | 刷新缓存 |
| GET | `/datamodels/:id` | `Get` | 获取详情 |
| GET | `/datamodels/:id/export` | `Export` | 导出 JSON |
| PUT | `/datamodels/:id` | `Update` | 更新 |
| DELETE | `/datamodels/:id` | `Delete` | 删除（仅 draft） |
| POST | `/datamodels/:id/activate` | `Activate` | 激活 |
| POST | `/datamodels/:id/deprecate` | `Deprecate` | 废弃 |
| GET | `/oui` | `ListOUI` | OUI 列表 |
| POST | `/oui` | `CreateOUI` | 创建 OUI |

**查询参数**:
- `carrier`: 运营商过滤
- `tech`: 制式过滤
- `oui`: OUI 过滤
- `product_class`: 产品类过滤
- `scope`: 范围过滤 (product/oui/carrier_default)
- `status`: 状态过滤 (draft/active/deprecated)

---

#### 3.2.6 配置模板模块 (`/api/v1/templates`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/templates` | `List` | 模板列表 |
| GET | `/templates/:id` | `Get` | 模板详情 |
| POST | `/templates` | `Create` | 创建模板 |
| PUT | `/templates/:id` | `Update` | 更新模板 |
| DELETE | `/templates/:id` | `Delete` | 删除模板 |

---

#### 3.2.7 自动开站模块 (`/api/v1/provisioning`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/provisioning/tasks` | `List` | 任务列表 |
| GET | `/provisioning/tasks/:id` | `Get` | 任务详情 |
| POST | `/provisioning/tasks` | `Create` | 创建任务 |
| POST | `/provisioning/tasks/:id/retry` | `Retry` | 重试失败任务 |

**查询参数**:
- `status`: 状态过滤 (pending/running/completed/failed)
- `device_id`: 设备 ID 过滤

---

#### 3.2.8 拓扑管理模块

##### 设备分组 (`/api/v1/groups`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/groups` | `ListTree` | 分组树 |
| POST | `/groups` | `Create` | 创建分组 |
| GET | `/groups/:id` | `Get` | 分组详情 |
| PUT | `/groups/:id` | `Update` | 更新分组 |
| DELETE | `/groups/:id` | `Delete` | 删除分组 |
| POST | `/groups/:id/devices` | `AddDevice` | 添加设备 |
| DELETE | `/groups/:id/devices/:deviceId` | `RemoveDevice` | 移除设备 |
| GET | `/groups/:id/devices` | `ListDevices` | 分组设备列表 |

##### 站点管理 (`/api/v1/sites`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/sites` | `ListSites` | 站点列表 |
| POST | `/sites` | `CreateSite` | 创建站点 |
| GET | `/sites/:id` | `GetSite` | 站点详情 |

##### 拓扑图 (`/api/v1/topology`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/topology/nodes` | `ListTopoNodes` | 拓扑节点 |
| GET | `/topology/edges` | `ListTopoEdges` | 拓扑边 |
| GET | `/topology/graph` | `GetTopoGraph` | 完整拓扑图 |
| GET | `/topology/geo` | `GetGeoData` | 地理数据 |

---

#### 3.2.9 性能管理模块 (`/api/v1/pm`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/pm/counters` | `ListCounters` | 计数器查询 |
| GET | `/pm/counters/aggregated` | `ListAggregatedCounters` | 聚合计数器 |
| GET | `/pm/kpi` | `ListKPIValues` | KPI 查询 |
| GET | `/pm/kpi/definitions` | `ListKPIDefinitions` | KPI 定义 |
| POST | `/pm/kpi/calculate` | `CalculateKPI` | 触发 KPI 计算 |
| GET | `/pm/tasks` | `ListTasks` | PM 任务列表 |
| POST | `/pm/tasks` | `CreateTask` | 创建 PM 任务 |
| GET | `/pm/files` | `ListPMFiles` | PM 文件列表 |
| GET | `/pm/files/:id/download` | `DownloadPMFile` | 下载 PM 文件 |

##### KPI 阈值 (`/api/v1/pm/thresholds`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/pm/thresholds` | `ListThresholds` | 阈值列表 |
| POST | `/pm/thresholds` | `CreateThreshold` | 创建阈值 |
| GET | `/pm/thresholds/:id` | `GetThreshold` | 阈值详情 |
| PUT | `/pm/thresholds/:id` | `UpdateThreshold` | 更新阈值 |
| DELETE | `/pm/thresholds/:id` | `DeleteThreshold` | 删除阈值 |

**查询参数**:
- `device_id`: 设备 ID
- `start_time`/`end_time`: 时间范围
- `granularity`: 粒度 (15min/hourly/daily)

---

#### 3.2.10 告警管理模块 (`/api/v1/alarms`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/alarms/active` | `ListActive` | 活跃告警 |
| GET | `/alarms/history` | `ListHistory` | 历史告警 |
| GET | `/alarms/statistics` | `Statistics` | 告警统计 |
| GET | `/alarms/:id` | `GetByID` | 告警详情 |
| POST | `/alarms/:id/acknowledge` | `Acknowledge` | 确认告警 |
| POST | `/alarms/:id/clear` | `ClearAlarm` | 清除告警 |

##### 告警规则 (`/api/v1/alarms/rules`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/alarms/rules` | `ListRules` | 规则列表 |
| POST | `/alarms/rules` | `CreateRule` | 创建规则 |
| GET | `/alarms/rules/:id` | `GetRule` | 规则详情 |
| PUT | `/alarms/rules/:id` | `UpdateRule` | 更新规则 |
| DELETE | `/alarms/rules/:id` | `DeleteRule` | 删除规则 |

**查询参数**:
- `device_id`: 设备 ID
- `severity`: 严重级别 (critical/major/minor/warning)
- `start_time`/`end_time`: 时间范围

---

#### 3.2.11 测量报告模块 (`/api/v1/mr`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/mr/files` | `ListFiles` | MR 文件列表 |
| GET | `/mr/files/:id/download` | `DownloadFile` | 下载 MR 文件 |
| GET | `/mr/data` | `QueryData` | MR 数据查询 |
| GET | `/mr/indicators` | `ListIndicators` | 指标列表 |
| GET | `/mr/indicators/all` | `ListAllIndicators` | 全部指标 |
| GET | `/mr/indicators/:code/stats` | `GetIndicatorStats` | 指标统计 |
| GET | `/mr/mappings` | `ListMappings` | 设备映射 |
| PUT | `/mr/mappings/:id` | `UpdateMapping` | 更新映射 |
| PUT | `/mr/mappings/:id/toggle` | `ToggleMapping` | 切换映射 |
| POST | `/mr/export` | `ExportMRData` | 导出数据 |

---

#### 3.2.12 固件管理模块

##### 固件版本 (`/api/v1/firmware`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/firmware` | `ListFirmware` | 固件列表 |
| POST | `/firmware` | `UploadFirmware` | 上传固件 |
| GET | `/firmware/:id` | `GetFirmware` | 固件详情 |
| DELETE | `/firmware/:id` | `DeleteFirmware` | 删除固件 |

##### 升级任务 (`/api/v1/upgrade-tasks`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/upgrade-tasks` | `ListUpgradeTasks` | 任务列表 |
| GET | `/upgrade-tasks/:id` | `GetUpgradeTask` | 任务详情 |
| POST | `/upgrade-tasks` | `TriggerUpgrade` | 单设备升级 |
| POST | `/upgrade-tasks/batch` | `BatchUpgrade` | 批量升级 |

---

#### 3.2.13 备份管理模块 (`/api/v1/backup`)

##### 备份任务

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/backup/tasks` | `ListTasks` | 任务列表 |
| POST | `/backup/tasks` | `CreateTask` | 创建任务 |
| GET | `/backup/tasks/:id` | `GetTask` | 任务详情 |
| DELETE | `/backup/tasks/:id` | `DeleteTask` | 删除任务 |
| POST | `/backup/tasks/:id/cancel` | `CancelTask` | 取消任务 |

##### 备份调度

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/backup/schedules` | `ListSchedules` | 调度列表 |
| POST | `/backup/schedules` | `CreateSchedule` | 创建调度 |
| PUT | `/backup/schedules/:id` | `UpdateSchedule` | 更新调度 |
| DELETE | `/backup/schedules/:id` | `DeleteSchedule` | 删除调度 |

##### FTP 配置

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/backup/ftp-configs` | `ListFTPConfigs` | FTP 配置列表 |
| POST | `/backup/ftp-configs` | `CreateFTPConfig` | 创建配置 |
| PUT | `/backup/ftp-configs/:id` | `UpdateFTPConfig` | 更新配置 |
| DELETE | `/backup/ftp-configs/:id` | `DeleteFTPConfig` | 删除配置 |
| POST | `/backup/ftp-configs/:id/test` | `TestFTPConnection` | 测试连接 |

---

#### 3.2.14 仪表盘模块 (`/api/v1/dashboard`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/dashboard/summary` | `GetSummary` | 汇总统计 |
| GET | `/dashboard/alarm-trend` | `GetAlarmTrend` | 告警趋势 |
| GET | `/dashboard/device-status` | `GetDeviceStatus` | 设备状态 |
| GET | `/dashboard/kpi-trend` | `GetKPITrend` | KPI 趋势 |
| GET | `/dashboard/region-stats` | `GetRegionStats` | 区域统计 |
| GET | `/dashboard/widgets` | `GetWidgets` | 组件布局 |
| PUT | `/dashboard/widgets` | `SaveWidgets` | 保存布局 |
| GET | `/dashboard/alarm-type-pie` | `GetAlarmTypePie` | 告警类型分布 |
| GET | `/dashboard/kpi-time-series` | `GetKPITimeSeries` | KPI 时序数据 |

---

#### 3.2.15 系统日志模块 (`/api/v1/logs`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/logs/system` | `ListSystemLogs` | 系统日志 |
| GET | `/logs/ne-messages` | `ListNEMessageLogs` | 网元消息日志 |

**查询参数**:
- `level`: 日志级别 (debug/info/warn/error)
- `source`: 来源过滤
- `start_time`/`end_time`: 时间范围

---

#### 3.2.16 配置同步模块 (`/api/v1/config/sync`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| POST | `/config/sync/push/:deviceId` | `PushConfig` | 下发配置 |
| POST | `/config/sync/pull/:deviceId` | `PullConfig` | 拉取配置 |
| GET | `/config/sync/status/:deviceId` | `GetSyncStatus` | 同步状态 |

---

#### 3.2.17 互操作测试模块 (`/api/v1/interop`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/interop/test-cases` | `ListTestCases` | 测试用例列表 |
| POST | `/interop/run` | `RunTests` | 运行测试 |
| POST | `/interop/run/:category` | `RunByCategory` | 按类别运行 |
| POST | `/interop/validate/:deviceId` | `ValidateDevice` | 验证设备数据模型 |

**测试类别**:
- `protocol`: 协议一致性
- `datamodel`: 数据模型验证
- `rpc`: RPC 方法测试

---

#### 3.2.18 文件管理模块 (`/api/v1/files`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/files` | `List` | 文件列表 |
| POST | `/files` | `Upload` | 上传文件 |
| GET | `/files/:id` | `GetByID` | 文件详情 |
| DELETE | `/files/:id` | `Delete` | 删除文件 |
| GET | `/files/:id/download` | `Download` | 下载文件 |
| POST | `/files/:id/distribute` | `Distribute` | 分发到设备 |

---

#### 3.2.19 MML 控制台模块 (`/api/v1/mml`)

##### 命令

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/mml/commands` | `ListCommands` | 命令列表 |
| GET | `/mml/commands/:id` | `GetCommand` | 命令详情 |
| POST | `/mml/execute` | `Execute` | 执行命令 |

##### 脚本

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/mml/scripts` | `ListScripts` | 脚本列表 |
| POST | `/mml/scripts` | `CreateScript` | 创建脚本 |
| PUT | `/mml/scripts/:id` | `UpdateScript` | 更新脚本 |
| DELETE | `/mml/scripts/:id` | `DeleteScript` | 删除脚本 |

##### 任务

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/mml/tasks` | `ListTasks` | 任务列表 |
| GET | `/mml/tasks/:id` | `GetTask` | 任务详情 |

---

#### 3.2.20 配置基线模块 (`/api/v1/config`)

##### 基线

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/config/baselines` | `ListBaselines` | 基线列表 |
| POST | `/config/baselines` | `CreateBaseline` | 创建基线 |
| GET | `/config/baselines/:id` | `GetBaseline` | 基线详情 |
| PUT | `/config/baselines/:id` | `UpdateBaseline` | 更新基线 |
| DELETE | `/config/baselines/:id` | `DeleteBaseline` | 删除基线 |

##### 配置任务

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/config/tasks` | `ListConfigTasks` | 配置任务列表 |
| POST | `/config/tasks` | `CreateConfigTask` | 创建配置任务 |

##### 邻区

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/config/neighbors` | `ListNeighbors` | 邻区关系列表 |

---

#### 3.2.21 许可证模块 (`/api/v1/licenses`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/licenses` | `List` | 许可证列表 |
| GET | `/licenses/summary` | `GetSummary` | 许可证汇总 |
| POST | `/licenses/activate` | `Activate` | 激活许可证 |
| POST | `/licenses/import` | `Import` | 导入许可证 |
| GET | `/licenses/:id` | `GetByID` | 许可证详情 |
| POST | `/licenses/:id/revoke` | `Revoke` | 撤销许可证 |

---

#### 3.2.22 运维工具模块 (`/api/v1/ops`)

##### 模板

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/ops/templates` | `ListTemplates` | 模板列表 |
| POST | `/ops/templates` | `CreateTemplate` | 创建模板 |
| GET | `/ops/templates/:id` | `GetTemplate` | 模板详情 |
| PUT | `/ops/templates/:id` | `UpdateTemplate` | 更新模板 |
| DELETE | `/ops/templates/:id` | `DeleteTemplate` | 删除模板 |

##### 命令记录

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/ops/command-records` | `ListCommandRecords` | 命令记录列表 |
| POST | `/ops/command-records` | `CreateCommandRecord` | 创建记录 |

##### 任务

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/ops/tasks` | `ListTasks` | 任务列表 |
| POST | `/ops/tasks` | `CreateTask` | 创建任务 |
| GET | `/ops/tasks/:id` | `GetTask` | 任务详情 |
| POST | `/ops/tasks/:id/cancel` | `CancelTask` | 取消任务 |
| POST | `/ops/tasks/:id/pause` | `PauseTask` | 暂停任务 |
| POST | `/ops/tasks/:id/resume` | `ResumeTask` | 恢复任务 |

---

#### 3.2.23 报表模块 (`/api/v1/reports`)

##### 报表定义

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/reports/definitions` | `ListDefinitions` | 定义列表 |
| POST | `/reports/definitions` | `CreateDefinition` | 创建定义 |
| GET | `/reports/definitions/:id` | `GetDefinition` | 定义详情 |
| PUT | `/reports/definitions/:id` | `UpdateDefinition` | 更新定义 |
| DELETE | `/reports/definitions/:id` | `DeleteDefinition` | 删除定义 |

##### 报表记录

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/reports/records` | `ListRecords` | 记录列表 |
| GET | `/reports/records/:id/download` | `DownloadRecord` | 下载报表 |

##### 生成

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| POST | `/reports/generate` | `GenerateReport` | 生成报表 |
| GET | `/reports/sample-data` | `GetSampleData` | 预览数据 |

---

#### 3.2.24 系统信息 (`/api/v1/system`)

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/system/info` | `sysInfoHandler.GetSystemInfo` | 系统信息 |

---

#### 3.2.25 北向接口模块 (`/api/v1/northbound`)

##### 推送目标

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/northbound/push/targets` | `listTargets` | 目标列表 |
| POST | `/northbound/push/targets` | `addTarget` | 添加目标 |
| DELETE | `/northbound/push/targets/:id` | `removeTarget` | 删除目标 |

##### 同步

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/northbound/sync/full` | `fullSync` | 全量同步 |
| GET | `/northbound/sync/incremental` | `incrementalSync` | 增量同步 |

##### 导出

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| POST | `/northbound/export/pm` | `pmHandler.ExportPM` | 导出 PM |
| POST | `/northbound/export/alarms` | `alarmHandler.ExportAlarms` | 导出告警 |
| GET | `/northbound/export/config/:deviceId` | `configHandler.ExportConfig` | 导出配置 |

---

#### 3.2.26 网元直连模块 (CMCC 专有)

> 需配置 `ne_direct.enabled: true`

**独立服务**: `internal/nedirect/server.go`

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| * | `/*` | `neHandler` | CMCC 网元直连协议 |

---

## 4. omcgo-worker 服务

**入口**: `cmd/worker/main.go`

Worker 是事件驱动进程，不暴露 HTTP 路由。它订阅 NATS 事件并处理后台任务。

### 4.1 事件订阅

| 订阅者 | 订阅事件 | 功能 |
|--------|----------|------|
| **PM Collector** | `pm.file.received` | PM 文件采集、解析、入库 |
| **Alarm Receiver** | `device.inform.alarm` | 告警接收、去重、入库 |
| **MR Collector** | `mr.file.received` | MR 文件采集、解析、入库 |
| **Transfer Bridge** | `transfer.upload.scheduled` | 文件传输桥接 |
| **Backup Executor** | `backup.task.scheduled` | 配置备份执行 |
| **Report Generator** | `report.generation.scheduled` | 报表生成 |

### 4.2 事件发布

Worker 处理完成后发布的事件：

| 事件 | 发布者 | 说明 |
|------|--------|------|
| `pm.file.parsed` | PM Collector | PM 文件解析完成 |
| `alarm.raised` | Alarm Receiver | 新告警产生 |
| `alarm.cleared` | Alarm Receiver | 告警清除 |
| `mr.file.parsed` | MR Collector | MR 文件解析完成 |

---

## 5. 路由统计

### 5.1 按模块统计

| 模块 | 路由数量 | 功能域 |
|------|----------|--------|
| Device | 8 | F06 |
| Admin (Auth) | 3 | F06 |
| Admin (Users) | 10 | F06 |
| Admin (Roles) | 7 | F06 |
| DataModel | 15 | F02 |
| ConfigTemplate | 5 | F02 |
| Provisioning | 4 | F09 |
| Topology | 15 | F06 |
| PM | 14 | F03 |
| Alarm | 12 | F04 |
| MR | 11 | F05 |
| Software | 8 | F06 |
| Backup | 15 | F06 |
| Dashboard | 9 | F06 |
| Syslog | 2 | F06 |
| ConfigSync | 3 | F02 |
| Interop | 4 | F10 |
| FileManager | 6 | F06 |
| MML | 10 | F06 |
| ConfigBaseline | 8 | F02 |
| License | 6 | F06 |
| Ops | 12 | F06 |
| Report | 8 | F06 |
| Northbound | 8 | F08 |
| System | 1 | F06 |
| **总计** | **~197** | |

### 5.2 按方法统计

| HTTP 方法 | 数量 |
|-----------|------|
| GET | ~100 |
| POST | ~60 |
| PUT | ~25 |
| DELETE | ~12 |

---

## 6. 认证与授权

### 6.1 公共路由（无需认证）

```
GET  /healthz
POST /api/v1/auth/login
POST /api/v1/auth/refresh
```

### 6.2 受保护路由（JWT 认证）

所有 `/api/v1/*` 路由（除公共路由外）需要：
1. JWT Bearer Token
2. Carrier Context（运营商上下文）
3. Audit Logger（审计日志）

### 6.3 管理员路由

`/api/v1/admin/*` 路由需要 `users:admin` 权限。

---

## 7. 中间件链

```
┌─────────────────────────────────────────────────────────────┐
│                        Request                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  RequestID (配置前缀: app-YYYYMMDDHHmmss-xxxxxxxx)          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  CORS (AllowOrigins from config)                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  RequestLogger (zap structured logging)                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  PrometheusMetrics (request duration, status codes)         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  [Public Routes] → Direct Handler                           │
│  [Protected Routes] → RequireAuth → RequireCarrier          │
│  [Admin Routes] → RequirePermission                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 8. 附录

### 8.1 完整路由树

```
/
├── healthz                                    [GET]     Public
└── api/v1
    ├── auth
    │   ├── login                              [POST]    Public
    │   ├── refresh                            [POST]    Public
    │   └── me                                 [GET]     Protected
    ├── devices                                [GET|POST]
    │   ├── stats                              [GET]
    │   ├── :id                                [GET|PUT|DELETE]
    │   │   ├── parameters                     [GET]
    │   │   └── reboot                         [POST]
    ├── datamodels                             [GET|POST]
    │   ├── resolve                            [GET]
    │   ├── statistics                         [GET]
    │   ├── import                             [POST]
    │   ├── cache/refresh                      [POST]
    │   └── :id                                [GET|PUT|DELETE]
    │       ├── export                         [GET]
    │       ├── activate                       [POST]
    │       └── deprecate                      [POST]
    ├── templates                              [GET|POST]
    │   └── :id                                [GET|PUT|DELETE]
    ├── provisioning/tasks                     [GET|POST]
    │   └── :id
    │       └── retry                          [POST]
    ├── groups                                 [GET|POST]
    │   └── :id                                [GET|PUT|DELETE]
    │       └── devices                        [GET|POST|DELETE]
    ├── sites                                  [GET|POST]
    │   └── :id                                [GET]
    ├── topology
    │   ├── nodes                              [GET]
    │   ├── edges                              [GET]
    │   ├── graph                              [GET]
    │   └── geo                                [GET]
    ├── pm
    │   ├── counters                           [GET]
    │   │   └── aggregated                     [GET]
    │   ├── kpi                                [GET]
    │   │   ├── definitions                    [GET]
    │   │   └── calculate                      [POST]
    │   ├── tasks                              [GET|POST]
    │   ├── files                              [GET]
    │   │   └── :id/download                   [GET]
    │   └── thresholds                         [GET|POST]
    │       └── :id                            [GET|PUT|DELETE]
    ├── alarms
    │   ├── active                             [GET]
    │   ├── history                            [GET]
    │   ├── statistics                         [GET]
    │   ├── :id                                [GET]
    │   │   ├── acknowledge                    [POST]
    │   │   └── clear                          [POST]
    │   └── rules                              [GET|POST]
    │       └── :id                            [GET|PUT|DELETE]
    ├── mr
    │   ├── files                              [GET]
    │   │   └── :id/download                   [GET]
    │   ├── data                               [GET]
    │   ├── indicators                         [GET]
    │   │   ├── all                            [GET]
    │   │   └── :code/stats                    [GET]
    │   ├── mappings                           [GET]
    │   │   └── :id                            [PUT]
    │   │       └── toggle                     [PUT]
    │   └── export                             [POST]
    ├── firmware                               [GET|POST]
    │   └── :id                                [GET|DELETE]
    ├── upgrade-tasks                          [GET|POST]
    │   ├── :id                                [GET]
    │   └── batch                              [POST]
    ├── backup
    │   ├── tasks                              [GET|POST]
    │   │   └── :id                            [GET|DELETE]
    │   │       └── cancel                     [POST]
    │   ├── schedules                          [GET|POST]
    │   │   └── :id                            [PUT|DELETE]
    │   └── ftp-configs                        [GET|POST]
    │       └── :id                            [PUT|DELETE]
    │           └── test                       [POST]
    ├── dashboard
    │   ├── summary                            [GET]
    │   ├── alarm-trend                        [GET]
    │   ├── device-status                      [GET]
    │   ├── kpi-trend                          [GET]
    │   ├── region-stats                       [GET]
    │   ├── widgets                            [GET|PUT]
    │   ├── alarm-type-pie                     [GET]
    │   └── kpi-time-series                    [GET]
    ├── logs
    │   ├── system                             [GET]
    │   └── ne-messages                        [GET]
    ├── config
    │   ├── sync
    │   │   ├── push/:deviceId                 [POST]
    │   │   ├── pull/:deviceId                 [POST]
    │   │   └── status/:deviceId               [GET]
    │   ├── baselines                          [GET|POST]
    │   │   └── :id                            [GET|PUT|DELETE]
    │   ├── tasks                              [GET|POST]
    │   └── neighbors                          [GET]
    ├── interop
    │   ├── test-cases                         [GET]
    │   ├── run                                [POST]
    │   ├── run/:category                      [POST]
    │   └── validate/:deviceId                 [POST]
    ├── files                                  [GET|POST]
    │   └── :id                                [GET|DELETE]
    │       ├── download                       [GET]
    │       └── distribute                     [POST]
    ├── mml
    │   ├── commands                           [GET]
    │   │   └── :id                            [GET]
    │   ├── execute                            [POST]
    │   ├── scripts                            [GET|POST]
    │   │   └── :id                            [PUT|DELETE]
    │   └── tasks                              [GET]
    │       └── :id                            [GET]
    ├── licenses                               [GET]
    │   ├── summary                            [GET]
    │   ├── activate                           [POST]
    │   ├── import                             [POST]
    │   └── :id                                [GET]
    │       └── revoke                         [POST]
    ├── ops
    │   ├── templates                          [GET|POST]
    │   │   └── :id                            [GET|PUT|DELETE]
    │   ├── command-records                    [GET|POST]
    │   └── tasks                              [GET|POST]
    │       └── :id                            [GET]
    │           ├── cancel                     [POST]
    │           ├── pause                      [POST]
    │           └── resume                     [POST]
    ├── reports
    │   ├── definitions                        [GET|POST]
    │   │   └── :id                            [GET|PUT|DELETE]
    │   ├── records                            [GET]
    │   │   └── :id/download                   [GET]
    │   ├── generate                           [POST]
    │   └── sample-data                        [GET]
    ├── northbound
    │   ├── push/targets                       [GET|POST]
    │   │   └── :id                            [DELETE]
    │   ├── sync
    │   │   ├── full                           [GET]
    │   │   └── incremental                    [GET]
    │   └── export
    │       ├── pm                             [POST]
    │       ├── alarms                         [POST]
    │       └── config/:deviceId               [GET]
    ├── system/info                            [GET]
    └── admin
        ├── users                              [GET|POST]
        │   └── :id                            [GET|PUT|DELETE]
        │       ├── roles                      [POST|DELETE]
        │       ├── reset-password             [POST]
        │       ├── lock                       [POST]
        │       └── unlock                     [POST]
        ├── roles                              [GET|POST]
        │   └── :id                            [GET|PUT|DELETE]
        ├── permissions                        [GET]
        └── audit-logs                         [GET]
```

---

*文档生成时间: 2026-03-20*

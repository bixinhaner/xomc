# OMC 系统数据流向全景分析

## Context

本文档基于 OMC 系统前后端源码的完整分析，梳理了三大数据流向路径：
- **读取路径**：前端 → 后端 → 基站（查询设备状态/参数/性能数据）
- **操作路径**：前端 → 后端 → 基站（下发配置/命令/升级）
- **上报路径**：基站 → 后端 → 数据库（设备主动上报数据入库）

系统由三个独立进程组成：

| 进程 | 端口 | 职责 |
|------|------|------|
| **omcgo-app** | :8080 | REST API 管理服务（前端交互） |
| **omcgo-acs** | :8000 | TR-069 ACS 引擎（基站交互） |
| **omcgo-worker** | - | 异步数据处理（事件消费） |

基础设施：PostgreSQL + TimescaleDB + Redis + MinIO + NATS JetStream

---

## 一、读取路径：前端 → 后端 → 基站

> 前端发起 GET 请求，后端查询数据库/缓存返回，**大部分读操作不直接访问基站**。

### 1.1 通用读取流程（不涉及基站）

```
┌──────────┐     HTTP GET      ┌──────────────┐    SQL Query    ┌────────────┐
│  React   │ ──────────────→  │  Gin Handler  │ ─────────────→ │ PostgreSQL │
│  前端    │  /api/v1/xxx     │  → Service    │                │ TimescaleDB│
│          │ ←──────────────  │  → Repository │ ←───────────── │            │
│          │   JSON Response  │               │   Result Rows  │            │
└──────────┘                  └──────────────┘                 └────────────┘
```

**涉及的业务模块及端点：**

| 业务域 | API 前缀 | 读取内容 | 数据源 |
|--------|----------|----------|--------|
| **设备管理** | `/devices` | 设备列表、详情、统计、分组 | PostgreSQL |
| **告警管理** | `/alarms/active`, `/alarms/history` | 活跃告警、历史告警、告警统计 | PG + TimescaleDB |
| **性能管理** | `/pm/counters`, `/pm/kpi` | PM 计数器、KPI 时序数据 | TimescaleDB |
| **测量报告** | `/mr/files`, `/mr/data`, `/mr/indicators` | MR 文件列表、解析记录、指标统计 | PG + TimescaleDB |
| **配置管理** | `/config/baselines`, `/config/tasks` | 基线配置、配置任务、邻区参数 | PostgreSQL |
| **数据模型** | `/datamodels` | TR-181 数据模型定义、统计 | PG + Redis 三级缓存 |
| **模板管理** | `/templates` | 配置模板列表、详情 | PostgreSQL |
| **软件管理** | `/firmware`, `/upgrade-tasks` | 固件版本列表、升级任务状态 | PostgreSQL |
| **文件管理** | `/files` | 文件列表、下载 | PG + MinIO |
| **备份管理** | `/backup/tasks`, `/backup/schedules` | 备份任务、调度计划、FTP 配置 | PostgreSQL |
| **MML 命令** | `/mml/commands`, `/mml/scripts` | 命令库、脚本库、执行任务 | PostgreSQL |
| **运维工具** | `/ops/templates`, `/ops/tasks` | 运维模板、任务、操作记录 | PostgreSQL |
| **拓扑管理** | `/topology/nodes`, `/topology/edges` | 拓扑节点、边、图、地理数据 | PostgreSQL |
| **报表管理** | `/reports/definitions`, `/reports/records` | 报表定义、生成记录、下载 | PG + MinIO |
| **Dashboard** | `/dashboard/*` | 综合概览、KPI 趋势、告警分布 | 聚合多表查询 |
| **日志管理** | `/logs/system`, `/logs/ne-messages` | 系统日志、网元消息日志 | PostgreSQL |
| **许可证** | `/licenses`, `/licenses/summary` | 许可列表、统计摘要 | PostgreSQL |
| **用户管理** | `/admin/users`, `/admin/roles` | 用户列表、角色权限、审计日志 | PostgreSQL |
| **北向接口** | `/northbound/sync/*` | 全量/增量同步数据给外部 OSS | 聚合多表查询 |
| **互操作** | `/interop/test-cases` | 测试用例列表 | PostgreSQL |

### 1.2 涉及基站的读取：参数拉取

```
┌──────────┐  POST /config/sync/pull/:id  ┌──────────────┐  EnqueueCmd   ┌───────┐
│  React   │ ──────────────────────────→  │  SyncHandler  │ ───────────→ │ Redis │
│  前端    │                              │  → CmdQueue   │  SortedSet   │ CmdQ  │
└──────────┘                              └──────────────┘               └───┬───┘
                                                                             │
    ┌────────────────────────────────────────────────────────────────────────┘
    │  基站下次 Inform 时
    ▼
┌──────────┐  Inform(EmptyPOST)  ┌──────────────┐  Dequeue    ┌───────┐
│  基站    │ ──────────────────→ │  ACS Handler  │ ─────────→ │ Redis │
│  (CPE)   │                    │               │            │ CmdQ  │
│          │ ←────────────────  │  GetParamReq  │            └───────┘
│          │  GetParameterValues│               │
│          │ ──────────────────→│  GetParamResp │ ──publish──→ NATS
│          │  ParamValues(XML)  │               │   command.get_parameters.response
└──────────┘                    └──────────────┘
                                                       │
    ┌──────────────────────────────────────────────────┘
    ▼
┌──────────────┐  存储参数  ┌────────────┐
│ DeviceService │ ────────→ │ PostgreSQL │
│ (Worker)      │           │ device_    │
│               │           │ parameters │
└──────────────┘            └────────────┘
```

**关键点：**
- 这是**间接读取**，非实时——前端下发拉取命令后，要等基站下次 Inform 时才执行
- 参数通过 Redis 命令队列 (`acs:cmdq:{device_sn}`) 中转
- ACS 引擎在基站会话中执行 `GetParameterValues` RPC
- 结果通过 NATS 事件总线回传给 Worker 进程存入数据库
- 前端后续通过 `GET /devices/:id/parameters` 读取缓存的参数值

### 1.3 数据模型三级缓存读取

```
┌──────────┐  GET /datamodels  ┌──────────────┐
│  React   │ ────────────────→│ DataModel     │
│  前端    │                  │ Handler       │
└──────────┘                  └──────┬────────┘
                                     │
                              ┌──────▼────────┐
                              │ DataModel     │
                              │ Registry      │
                              └──────┬────────┘
                                     │
                   ┌─────────────────┼─────────────────┐
                   ▼                 ▼                  ▼
            ┌──────────┐     ┌──────────┐      ┌────────────┐
            │ L1 Cache │     │ L2 Cache │      │ L3 Source  │
            │ sync.Map │     │  Redis   │      │ PostgreSQL │
            │ (进程内) │     │ (TTL 24h)│      │ (持久化)   │
            └──────────┘     └──────────┘      └────────────┘
```

---

## 二、操作路径：前端 → 后端 → 基站

> 前端发起 POST/PUT/DELETE 请求，后端执行操作并可能下发命令到基站。

### 2.1 仅操作数据库（不涉及基站）的业务

```
┌──────────┐   POST/PUT/DELETE   ┌──────────────┐   INSERT/UPDATE   ┌────────────┐
│  React   │ ─────────────────→ │  Gin Handler  │ ────────────────→│ PostgreSQL │
│  前端    │                    │  → Service    │                  │            │
│          │ ←───────────────── │  → Repository │ ←────────────────│            │
│          │   JSON Response    │               │                  │            │
└──────────┘                    └──────┬────────┘                  └────────────┘
                                       │ (部分操作)
                                       ▼
                                ┌──────────────┐
                                │  NATS Event  │  (如 alarm.acknowledged)
                                └──────────────┘
```

| 业务域 | 操作端点 | 操作内容 | 是否涉及基站 |
|--------|----------|----------|:------------:|
| **用户管理** | POST `/admin/users`, PUT/DELETE | 创建/修改/删除/锁定用户 | 否 |
| **角色权限** | POST `/admin/roles`, PUT/DELETE | 创建/修改/删除角色 | 否 |
| **告警确认** | POST `/alarms/:id/acknowledge` | 确认告警 | 否 |
| **告警清除** | POST `/alarms/:id/clear` | 清除告警 | 否 |
| **告警规则** | POST `/alarms/rules`, PUT/DELETE | 创建/修改/删除告警规则 | 否 |
| **PM 阈值** | POST `/pm/thresholds`, PUT/DELETE | 创建/修改/删除 PM 阈值 | 否 |
| **配置基线** | POST `/config/baselines`, PUT/DELETE | 创建/修改/删除配置基线 | 否 |
| **数据模型** | POST `/datamodels`, 激活/弃用 | 创建/导入/导出/激活数据模型 | 否 |
| **配置模板** | POST `/templates`, PUT/DELETE | 创建/修改/删除配置模板 | 否 |
| **文件上传** | POST `/files` (multipart) | 上传文件到 MinIO | 否 |
| **备份计划** | POST `/backup/schedules` | 创建/修改备份调度 | 否 |
| **FTP 配置** | POST `/backup/ftp-configs` | 创建/测试 FTP 连接 | 否 |
| **MML 脚本** | POST `/mml/scripts`, PUT/DELETE | 创建/修改/删除命令脚本 | 否 |
| **运维模板** | POST `/ops/templates`, PUT/DELETE | 创建/修改/删除运维模板 | 否 |
| **报表定义** | POST `/reports/definitions` | 创建/修改报表定义 | 否 |
| **报表生成** | POST `/reports/generate` | 触发报表生成 | 否 |
| **拓扑分组** | POST `/groups`, 添加/移除设备 | 创建分组、管设备归属 | 否 |
| **许可证** | POST `/licenses/activate`, 撤销 | 激活/导入/撤销许可证 | 否 |
| **北向目标** | POST `/northbound/push/targets` | 添加/删除 OSS 推送目标 | 否 |
| **Dashboard** | PUT `/dashboard/widgets` | 保存仪表板布局 | 否 |
| **设备注册** | POST `/devices` | 手动注册设备（仅入库） | 否 |
| **OUI 管理** | POST `/oui` | 添加厂商 OUI | 否 |

### 2.2 配置下发（前端 → 后端 → Redis 命令队列 → ACS → 基站）

```
┌──────────┐  POST /config/sync/push/:id  ┌──────────────┐
│  React   │ ──────────────────────────→  │  SyncHandler  │
│  前端    │  {parameters:[{name,value}]} │               │
└──────────┘                              └──────┬────────┘
                                                  │
                              ┌────────────────────┘
                              ▼
                       ┌──────────────┐   EnqueueSetParameters
                       │  CmdQueue    │ ─────────────────────→  Redis
                       │  (Redis)     │   acs:cmdq:{device_sn}
                       └──────────────┘   SortedSet
                                                  │
    ┌─────────────────────────────────────────────┘
    │ 基站下次 Inform（或 ConnectionRequest 触发）
    ▼
┌──────────┐  Inform / EmptyPOST   ┌──────────────┐
│  基站    │ ───────────────────→  │  ACS Handler  │
│  (CPE)   │                      │               │
│          │ ←──────────────────  │ Dequeue cmd   │
│          │  SetParameterValues  │ from Redis    │
│          │  (SOAP/XML)          │               │
│          │ ───────────────────→ │               │
│          │  SetParamResponse    │               │  ──publish──→ NATS
│          │  (Success/Fault)     │               │  command.set_parameters.response
└──────────┘                      └──────────────┘
                                          │
                               ┌──────────┘
                               ▼
                        ┌──────────────┐
                        │ DeviceService │  更新 device_parameters
                        │ (Worker)      │  表中的参数缓存
                        └──────────────┘
```

**涉及的操作：**
- `POST /config/sync/push/:deviceId` — 推送参数到设备
- `POST /config/tasks` (task_type=param_sync) — 批量参数同步任务

### 2.3 固件升级（前端 → 后端 → MinIO + Redis → ACS → 基站）

```
┌──────────┐  POST /firmware        ┌──────────────┐  Upload   ┌───────┐
│  React   │ (multipart upload) ─→ │ SoftwareHdlr │ ────────→│ MinIO │
│  前端    │                       │               │          │固件桶 │
└──────────┘                       └──────────────┘          └───────┘

┌──────────┐  POST /upgrade-tasks/batch  ┌──────────────┐
│  React   │ ────────────────────────→  │ SoftwareHdlr │
│  前端    │  {device_ids, firmware_id} │               │
└──────────┘                            └──────┬────────┘
                                               │
                    ┌──────────────────────────┘
                    │ 1. 创建升级任务 (PostgreSQL)
                    │ 2. 队列 Download RPC (Redis CmdQueue)
                    │ 3. 发送 ConnectionRequest 到设备 (HTTP Digest)
                    ▼
             ┌──────────────┐  HTTP POST ConnReq
             │ ConnReqClient│ ───────────────────→  基站
             └──────────────┘  (触发设备立即 Inform)
                                                      │
    ┌─────────────────────────────────────────────────┘
    │ 设备收到 ConnReq 后发送 Inform
    ▼
┌──────────┐  Inform          ┌──────────────┐  Dequeue   ┌───────┐
│  基站    │ ──────────────→ │  ACS Handler  │ ────────→ │ Redis │
│  (CPE)   │                 │               │           │ CmdQ  │
│          │ ←────────────── │  Download RPC │           └───────┘
│          │  Download cmd   │  (firmware URL│
│          │  (MinIO URL)    │   in MinIO)   │
│          │                 └──────────────┘
│          │
│          │ ... 设备下载固件并安装 ...
│          │
│          │  Inform(TransferComplete + M Download)
│          │ ──────────────→ ACS Handler ──publish──→ NATS
└──────────┘                  device.inform.transfer_complete
                                           │
                               ┌───────────┘
                               ▼
                        ┌──────────────┐
                        │ SoftwareService│  更新升级任务状态
                        │ (Worker)       │  → completed/failed
                        └──────────────┘
```

### 2.4 设备重启

```
┌──────────┐  POST /devices/:id/reboot  ┌──────────────┐  Enqueue
│  React   │ ────────────────────────→ │ DeviceHandler │ ──────→ Redis CmdQ
└──────────┘                           └──────────────┘
                                                              │
    基站 Inform 时 ─→ ACS Dequeue ─→ 发送 Reboot RPC ─→ 基站重启
    基站重启后 ─→ Inform(M Reboot) ─→ publish device.inform.reboot_complete
```

### 2.5 自动开通（Provisioning）

```
     设备首次上电 → 发送 Bootstrap Inform
                        │
┌──────────┐  Inform(BOOTSTRAP)  ┌──────────────┐
│  基站    │ ──────────────────→│  ACS Handler  │
│  (新CPE) │                    │               │
└──────────┘                    └──────┬────────┘
                                       │ publish
                                       ▼
                              device.inform.bootstrap
                                       │
                         ┌─────────────┼─────────────┐
                         ▼                            ▼
                  ┌──────────────┐            ┌──────────────┐
                  │ InformHandler │            │ Provisioning │
                  │ 注册设备入库  │            │ Engine       │
                  │ (DeviceService)│           │ 自动匹配模板  │
                  └──────────────┘            └──────┬────────┘
                                                      │
                  1. 根据 OUI+ProductClass 匹配运营商
                  2. 选择对应的 ProvisioningTemplate
                  3. 队列 SetParameterValues RPC 到 Redis
                  4. 等基站下次 Inform 时执行配置
                  5. 发布 provision.completed 事件
```

**也可手动触发：**
- `POST /provisioning/tasks` — 手动触发自动开通
- `POST /provisioning/tasks/:id/retry` — 重试失败任务

### 2.6 MML 命令执行

```
┌──────────┐  POST /mml/execute           ┌──────────────┐
│  React   │ ─────────────────────────→  │  MML Handler  │
│  前端    │  {command_code, device_sns}  │               │
└──────────┘                              └──────┬────────┘
                                                  │
                              将命令翻译为 TR-069 RPC
                              → 队列到 Redis CmdQueue
                              → 基站 Inform 时执行
                              → 结果回写 MML Task 表
```

### 2.7 文件分发到设备

```
┌──────────┐  POST /files/:id/distribute    ┌──────────────┐
│  React   │ ───────────────────────────→  │ FileHandler   │
│  前端    │  {device_sns: [...]}          │               │
└──────────┘                                └──────┬────────┘
                                                    │
                        生成 MinIO 下载 URL
                        → 队列 Download RPC 到 Redis
                        → 基站 Inform 时下载文件
```

### 2.8 操作路径汇总

| 操作类型 | 前端端点 | 后端处理 | 是否到基站 | 基站协议 |
|---------|---------|---------|:--------:|---------|
| 参数推送 | POST `/config/sync/push/:id` | CmdQueue → Redis | **是** | TR-069 SetParameterValues |
| 参数拉取 | POST `/config/sync/pull/:id` | CmdQueue → Redis | **是** | TR-069 GetParameterValues |
| 配置任务 | POST `/config/tasks` | 批量 → CmdQueue | **是** | TR-069 SetParameterValues |
| 固件升级 | POST `/upgrade-tasks/batch` | ConnReq + CmdQueue | **是** | TR-069 Download |
| 设备重启 | POST `/devices/:id/reboot` | CmdQueue → Redis | **是** | TR-069 Reboot |
| 自动开通 | POST `/provisioning/tasks` | Template → CmdQueue | **是** | TR-069 SetParameterValues |
| MML 执行 | POST `/mml/execute` | 翻译 → CmdQueue | **是** | TR-069 RPC |
| 文件分发 | POST `/files/:id/distribute` | MinIO URL → CmdQueue | **是** | TR-069 Download |
| 互操作测试 | POST `/interop/run` | TestRunner → CmdQueue | **是** | TR-069 多种 RPC |
| 备份任务 | POST `/backup/tasks` | CmdQueue → Upload | **是** | TR-069 Upload |

---

## 三、上报路径：基站 → 后端 → 数据库

> 基站通过 TR-069 Inform 主动上报数据，经 ACS 引擎 → NATS 事件 → Worker 处理 → 入库。

### 3.1 整体数据上报架构

```
┌──────────┐                    ┌──────────────┐                ┌──────────────┐
│  基站    │  SOAP/XML Inform   │  ACS Engine  │  NATS Events   │   Worker     │
│  (CPE)   │ ─────────────────→│  (omcgo-acs) │ ──────────────→│ (omcgo-wkr)  │
│          │  HTTP POST :8000   │              │                │              │
│          │                    │  解析 SOAP   │  device.inform │  事件消费者   │
│          │                    │  提取参数    │  .bootstrap    │              │
│          │                    │  分类事件    │  .periodic     │  ┌──────────┐│
│          │                    │  发布到 NATS │  .alarm        │  │PM Collector│
│          │                    │              │  .transfer_*   │  │MR Collector│
│          │                    │              │  .value_change │  │AlarmEngine │
└──────────┘                    └──────────────┘                │  │DeviceSvc   │
                                                                │  │TransferBr  │
                                                                │  └──────────┘│
                                                                └──────┬───────┘
                                                                       │
                                              ┌────────────────────────┼────────────┐
                                              ▼                        ▼            ▼
                                       ┌────────────┐          ┌────────────┐ ┌───────┐
                                       │ PostgreSQL │          │ TimescaleDB│ │ MinIO │
                                       │ 结构化数据 │          │ 时序数据   │ │ 文件  │
                                       └────────────┘          └────────────┘ └───────┘
```

### 3.2 设备注册上报（Bootstrap）

```
┌──────────┐                 ┌──────────────┐              ┌──────────────┐              ┌────────────┐
│  基站    │  Inform         │  ACS Engine  │  NATS        │   Worker     │              │ PostgreSQL │
│  (新CPE) │  (BOOTSTRAP)   │              │              │              │              │            │
│          │ ──────────────→│              │              │              │              │            │
│          │  DeviceId:      │ 解析 SOAP    │              │              │              │            │
│          │  {OUI, SN,      │ 提取事件码   │  publish     │              │              │            │
│          │   ProductClass} │ BOOTSTRAP    │ ───────────→│ InformHandler│              │            │
│          │  EventCode:     │              │  device.     │              │              │            │
│          │  "0 BOOTSTRAP"  │              │  inform.     │ 解析 OUI     │              │            │
│          │  Parameters:    │              │  bootstrap   │ 匹配运营商   │  INSERT      │            │
│          │  [IP, FW ver..] │              │              │ RegisterFrom │ ───────────→│ devices    │
│          │                 │              │              │ Inform()     │              │ 表         │
│          │ ←────────────── │ InformResp   │              │              │              │            │
│          │  (MaxEnvelope=1)│              │              │              │              │            │
└──────────┘                 └──────────────┘              └──────────────┘              └────────────┘

写入 devices 表的字段：
- serial_number, oui, product_class, manufacturer
- carrier (根据 OUI 从 CarrierRegistry 解析)
- technology (LTE/NR, 根据 product_class 判断)
- conn_status = 'online'
- ip_address, firmware_version (从 Inform 参数提取)
- connection_request_url, last_inform_at
```

### 3.3 心跳上报（Periodic Inform）

```
┌──────────┐  Inform(PERIODIC)   ┌──────────┐  device.inform.periodic  ┌──────────────┐
│  基站    │ ──────────────────→│ ACS      │ ───────────────────────→│ InformHandler │
│  (CPE)   │  EventCode:        │ Engine   │                         │              │
│          │  "2 PERIODIC"      │          │                         │ 更新设备状态  │
│          │  Parameters:       │          │                         │ last_inform_at│
│          │  [CurrentTime,     │          │                         │ conn_status   │
│          │   IP, uptime...]   │          │                         │ = 'online'   │
└──────────┘                    └──────────┘                         └──────┬───────┘
                                                                            │
                                                                     ┌──────▼───────┐
                                                                     │ Redis        │
                                                                     │ acs:heartbeat│
                                                                     │ :{device_sn} │
                                                                     │ TTL=2×Inform │
                                                                     │ Interval     │
                                                                     └──────────────┘

心跳监控器 (HeartbeatMonitor)：
- 后台持续运行
- 检测 Redis TTL 过期
- 过期 → 标记设备 offline
- 发布 device.connection.lost 事件
```

### 3.4 告警上报

```
┌──────────┐  Inform(ALARM)      ┌──────────┐  device.inform.alarm  ┌──────────────┐
│  基站    │ ──────────────────→│ ACS      │ ──────────────────→  │ AlarmEngine  │
│  (CPE)   │  EventCode:        │ Engine   │   (NATS durable      │ (Worker)     │
│          │  "ALARM xxx"       │          │    queue: alarm-      │              │
│          │  Parameters:       │          │    workers)           │ 1.运营商适配 │
│          │  [AlarmCode,       │          │                       │   映射告警级别│
│          │   Severity,        │          │                       │              │
│          │   Description...]  │          │                       │ 2.Redis 去重 │
└──────────┘                    └──────────┘                       │   alarm:active│
                                                                   │   :{device_sn}│
                                                                   │              │
                                                                   │ 3.入库       │
                                                                   └──────┬───────┘
                                                                          │
                                                    ┌─────────────────────┼──────────────────┐
                                                    ▼                     ▼                   ▼
                                             ┌────────────┐       ┌────────────┐      ┌──────────┐
                                             │ alarms_    │       │ alarms_    │      │ NATS     │
                                             │ active     │       │ history    │      │ alarm.   │
                                             │ (PG)       │       │ (TS 7天块) │      │ raised   │
                                             └────────────┘       └────────────┘      └────┬─────┘
                                                                                           │
                                                                                    ┌──────▼─────┐
                                                                                    │ PushEngine │
                                                                                    │ → OSS 推送 │
                                                                                    └────────────┘
```

### 3.5 PM 性能文件上报

```
┌──────────┐  设备生成PM文件      ┌──────────┐
│  基站    │  (每15分钟一个XML)   │ ACS      │
│  (CPE)   │                    │ Engine   │
│          │  Inform(Autonomous │          │
│          │  TransferComplete) │          │
│          │ ──────────────────→│ 解析文件信息│
│          │  FileType="4"(PM)  │ TransferURL│
│          │  TransferURL=...   │ FileName  │
│          │  FileName=...      │          │
└──────────┘                    └──────┬───┘
                                       │ publish
                                       ▼
                         device.inform.autonomous_transfer_complete
                                       │
                               ┌───────▼──────┐
                               │TransferBridge│  (Worker, durable queue: transfer-bridge)
                               │              │
                               │ 1. HTTP GET  │──→ 从设备下载 PM 文件
                               │    TransferURL│
                               │              │
                               │ 2. Upload    │──→ MinIO (pmfiles 桶)
                               │    to MinIO  │
                               │              │
                               │ 3. Publish   │──→ NATS: pm.file.received
                               └──────────────┘
                                       │
                               ┌───────▼──────┐
                               │ PM Collector │  (Worker, durable queue: pm-workers)
                               │              │
                               │ 1. Download  │←── MinIO
                               │    from MinIO│
                               │              │
                               │ 2. Parse XML │  解析 XML 提取计数器
                               │    提取:      │  <Object><Parameter>
                               │    - cell_id │  RRC.ConnAvg = 1234
                               │    - counter │
                               │    - value   │
                               │              │
                               │ 3. Store     │──→ TimescaleDB: pm_counters (1天分块)
                               │    counters  │
                               │              │
                               │ 4. Calculate │──→ KPIEngine 计算 KPI
                               │    KPIs      │   用运营商公式计算
                               │              │   如: RRC成功率 = (尝试-失败)/尝试×100
                               │              │
                               │ 5. Store     │──→ TimescaleDB: kpi_values
                               │    KPI values│
                               │              │
                               │ 6. Publish   │──→ NATS: pm.file.parsed
                               └──────────────┘

TimescaleDB 存储优化：
- pm_counters: 1天分块, 7天后压缩, 90天保留
- kpi_values:  1天分块, 同上
- 连续聚合:   pm_counters_hourly (小时级汇总视图)
```

### 3.6 MR 测量报告上报

```
┌──────────┐  Inform(Autonomous     ┌──────────┐
│  基站    │  TransferComplete)    │ ACS      │
│  (CPE)   │  FileType="5"(MR)    │ Engine   │
│          │ ─────────────────────→│          │
└──────────┘                       └──────┬───┘
                                          │ publish
                                          ▼
                            device.inform.autonomous_transfer_complete
                                          │
                                  ┌───────▼──────┐
                                  │TransferBridge│
                                  │ 下载 → MinIO │  (mrfiles 桶)
                                  │ → publish    │  mr.file.received
                                  └──────────────┘
                                          │
                                  ┌───────▼──────┐
                                  │ MR Collector │  (durable queue: mr-workers)
                                  │              │
                                  │ 1. 下载 MinIO│
                                  │              │
                                  │ 2. 检测类型  │  MRO / MRS / MRE
                                  │    (文件名)  │
                                  │              │
                                  │ 3. 选择解析器│  MROParser / MRSParser / MREParser
                                  │    (运营商   │
                                  │     特定格式)│
                                  │              │
                                  │ 4. 存储      │
                                  └──────┬───────┘
                                         │
                            ┌────────────┼────────────┐
                            ▼                         ▼
                     ┌────────────┐           ┌────────────┐
                     │ mr_files   │           │ mr_records │
                     │ (PG 元数据)│           │ (TS 1天块) │
                     │ 文件名/大小│           │ JSONB 存储 │
                     │ 解析状态   │           │ 测量数据   │
                     └────────────┘           └────────────┘
```

### 3.7 参数变更上报

```
┌──────────┐  Inform(VALUE CHANGE)  ┌──────────┐  device.inform.value_change  ┌──────────────┐
│  基站    │ ─────────────────────→│ ACS      │ ──────────────────────────→│ DeviceService │
│  (CPE)   │  变更的参数列表       │ Engine   │                            │ (Worker)      │
│          │  [{name, value}...]   │          │                            │              │
└──────────┘                       └──────────┘                            │ 更新参数缓存 │
                                                                           └──────┬───────┘
                                                                                  │
                                                                           ┌──────▼───────┐
                                                                           │ PostgreSQL   │
                                                                           │ device_      │
                                                                           │ parameters   │
                                                                           │ UPSERT       │
                                                                           └──────────────┘
```

### 3.8 设备日志/配置文件上报

```
┌──────────┐  Inform(Autonomous     ┌──────────┐  autonomous_transfer  ┌──────────────┐
│  基站    │  TransferComplete)    │ ACS      │  _complete           │TransferBridge│
│  (CPE)   │  FileType="3"(Logs)  │ Engine   │ ──────────────────→ │              │
│          │ ─────────────────────→│          │                     │ 下载 → MinIO │
└──────────┘                       └──────────┘                     │ (logs 桶)    │
                                                                    └──────────────┘
```

---

## 四、事件总线全景（NATS JetStream）

```
                              ┌─────────────────────────────────┐
                              │        NATS JetStream           │
                              │     (事件总线 + 持久化队列)       │
                              └─────────────────────────────────┘
                                           ▲  │
           发布者 (Publishers)              │  │  消费者 (Subscribers)
    ┌──────────────────────────────────────┘  └──────────────────────────────────────┐
    │                                                                                │
    │  ACS Engine 发布:                          Worker 消费:                          │
    │  ┌─────────────────────────────────┐       ┌─────────────────────────────────┐  │
    │  │ device.inform.bootstrap         │──────→│ InformHandler (注册设备)         │  │
    │  │                                 │──────→│ ProvisioningEngine (自动开通)    │  │
    │  │ device.inform.periodic          │──────→│ InformHandler (心跳更新)         │  │
    │  │ device.inform.value_change      │──────→│ DeviceService (参数缓存)         │  │
    │  │ device.inform.alarm             │──────→│ AlarmEngine (告警处理)           │  │
    │  │ device.inform.transfer_complete │──────→│ SoftwareService (升级完成)       │  │
    │  │ device.inform.autonomous_       │──────→│ TransferBridge (文件下载→MinIO)  │  │
    │  │   transfer_complete             │       │                                 │  │
    │  │ device.inform.reboot_complete   │──────→│ DeviceService (重启完成)         │  │
    │  │ device.connection.lost          │──────→│ HeartbeatMonitor (离线标记)      │  │
    │  └─────────────────────────────────┘       └─────────────────────────────────┘  │
    │                                                                                │
    │  TransferBridge 发布:                      Worker 消费:                          │
    │  ┌─────────────────────────────────┐       ┌─────────────────────────────────┐  │
    │  │ pm.file.received                │──────→│ PMCollector (解析PM→入库)        │  │
    │  │ mr.file.received                │──────→│ MRCollector (解析MR→入库)        │  │
    │  └─────────────────────────────────┘       └─────────────────────────────────┘  │
    │                                                                                │
    │  AlarmEngine 发布:                         PushEngine 消费:                      │
    │  ┌─────────────────────────────────┐       ┌─────────────────────────────────┐  │
    │  │ alarm.raised                    │──────→│ oss.alarm.forward (推送到OSS)    │  │
    │  │ alarm.cleared                   │       │                                 │  │
    │  └─────────────────────────────────┘       └─────────────────────────────────┘  │
    │                                                                                │
    └────────────────────────────────────────────────────────────────────────────────┘
```

---

## 五、数据存储分层

```
┌──────────────────────────────────────────────────────────────────┐
│                        数据存储全景                               │
├──────────────────┬───────────────────┬───────────────────────────┤
│   PostgreSQL     │   TimescaleDB     │     Redis                 │
│   (结构化数据)    │   (时序数据)       │     (缓存/队列)           │
├──────────────────┼───────────────────┼───────────────────────────┤
│ devices          │ pm_counters       │ acs:session:{sn}          │
│ device_parameters│   (1天块/90天留)   │ acs:cmdq:{sn}            │
│ alarms_active    │ kpi_values        │ acs:heartbeat:{sn}        │
│ mr_files         │   (1天块/90天留)   │ alarm:active:{sn}        │
│ firmware         │ alarms_history    │ datamodel:*               │
│ upgrade_tasks    │   (7天块/365天留)  │ ratelimit:inform:{sn}    │
│ provisioning_    │ mr_records        │                           │
│   tasks          │   (1天块/90天留)   │                           │
│ config_baselines │                   │                           │
│ config_templates │ 连续聚合:          │                           │
│ config_tasks     │ pm_counters_hourly│                           │
│ datamodels       │                   │                           │
│ users/roles      │                   │                           │
│ audit_logs       │                   │                           │
│ kpi_definitions  │                   │                           │
│ alarm_rules      │                   │                           │
│ mml_commands     │                   │                           │
│ ops_templates    │                   │                           │
│ report_defs      │                   │                           │
│ licenses         │                   │                           │
│ topology_*       │                   │                           │
│ backup_*         │                   │                           │
├──────────────────┴───────────────────┴───────────────────────────┤
│                        MinIO (S3 对象存储)                        │
├──────────────────────────────────────────────────────────────────┤
│ pmfiles/     - PM 性能数据 XML 文件                               │
│ mrfiles/     - MR 测量报告 XML 文件                               │
│ firmware/    - 固件升级包                                         │
│ logs/        - 设备日志文件                                       │
│ reports/     - 生成的报表文件                                     │
│ backups/     - 设备配置备份文件                                    │
│ files/       - 通用托管文件                                       │
└──────────────────────────────────────────────────────────────────┘
```

---

## 六、中间件处理链

每个 REST API 请求经过的处理链：

```
HTTP Request
    │
    ▼
┌──────────────┐
│ gin.Recovery │  崩溃恢复
└──────┬───────┘
       ▼
┌──────────────┐
│ CORS         │  跨域处理
└──────┬───────┘
       ▼
┌──────────────┐
│ RequestLogger│  结构化日志 (Zap)
└──────┬───────┘
       ▼
┌──────────────┐
│ Prometheus   │  HTTP 指标采集
│ Metrics      │
└──────┬───────┘
       ▼
┌──────────────┐
│ RequireAuth  │  JWT Token 验证 (公开路由跳过)
└──────┬───────┘
       ▼
┌──────────────┐
│ RequireCarrier│ 从 JWT Claims 提取运营商
└──────┬───────┘
       ▼
┌──────────────┐
│ AuditLogger  │  记录所有变更操作到审计表
└──────┬───────┘
       ▼
┌──────────────┐
│ RequirePerm  │  RBAC 权限校验 (仅管理路由)
└──────┬───────┘
       ▼
┌──────────────┐
│   Handler    │  业务处理 → Service → Repository
└──────────────┘
```

---

## 七、运营商适配层

```
┌─────────────────────────────────────────────────────┐
│              CarrierRegistry                        │
│    ┌──────────┬──────────┬──────────┐               │
│    │  CMCC    │  CTCC    │  CUCC    │               │
│    │ 中国移动  │ 中国电信  │ 中国联通  │               │
│    └────┬─────┴────┬─────┴────┬─────┘               │
│         │          │          │                      │
│    每个运营商适配器提供:                               │
│    ├─ OUI → 运营商映射                                │
│    ├─ TR-069 参数路径映射 (厂商路径 ↔ 统一名称)        │
│    ├─ KPI 公式定义 (如 RRC成功率/切换成功率)           │
│    ├─ 告警码 → 告警级别映射                           │
│    ├─ 支持的网络制式 (LTE/5G NR)                     │
│    └─ 自动开通模板                                    │
└─────────────────────────────────────────────────────┘
```

---

## 八、前端数据消费模式

```
┌─────────────────────────────────────────────────────┐
│              React 前端数据消费                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  HTTP Client (services/http.ts):                    │
│  ├─ 自动 camelCase ↔ snake_case 转换                │
│  ├─ Bearer Token 注入                               │
│  ├─ 401 自动刷新 Token                              │
│  └─ 错误提取与格式化                                 │
│                                                     │
│  React Query Hooks (hooks/api/*.ts):                │
│  ├─ useQuery  → 读取操作 (自动缓存+自动刷新)         │
│  │   ├─ refetchInterval: 15-60s (按数据类型)        │
│  │   └─ staleTime: 自动管理                         │
│  ├─ useMutation → 写操作 (手动触发)                  │
│  │   └─ onSuccess → invalidateQueries (刷新缓存)    │
│  └─ Mock/Real 切换: useMock ? mockService : realApi │
│                                                     │
│  实时更新策略:                                       │
│  ├─ 活跃告警: 每 15-30s 轮询                        │
│  ├─ Dashboard: 每 30-60s 轮询                       │
│  ├─ 设备状态: 每 30s 轮询                            │
│  ├─ 任务进度: 每 5-30s 轮询                          │
│  └─ 历史数据: 不自动刷新                             │
└─────────────────────────────────────────────────────┘
```

---

## 总结

| 维度 | 数量 |
|------|------|
| 前端 API 服务文件 | 24 个 |
| 前端总端点数 | 198 个 (读 110 / 写 88) |
| 后端模块 | 25+ 个 |
| 后端 REST 端点 | ~170 个 |
| Repository 接口 | 51 个 |
| NATS 事件主题 | 31 个 |
| 业务域 | 16 个 |
| 运营商适配器 | 3 个 (CMCC/CTCC/CUCC) |
| 南向协议 | TR-069/CWMP (SOAP/XML) |
| 时序表 (TimescaleDB) | 4 个 (pm_counters, kpi_values, alarms_history, mr_records) |
| MinIO 存储桶 | 7 个 |

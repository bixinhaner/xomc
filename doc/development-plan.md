# OMC Go 开发实施计划

> **版本**: v1.1
> **日期**: 2026-03-05
> **状态**: Phase 1 已完成
> **最后更新**: 2026-03-05

---

## 1. 文档目的

本文档基于 `doc/detailed-design/` 目录下的 21 个详细设计文档，制定完整的开发实施计划。将 4 个实施阶段细化为 17 个可交付的 Sprint，明确每个 Sprint 的任务清单、文件产出、前置依赖、验收标准和预估工作量，为开发团队提供可执行的开发路线图。

---

## 2. 项目总览

### 2.1 系统定位

OMC Go 是面向小基站/皮基站/微基站的无线操作维护中心系统，支持 TR069/CWMP 协议，覆盖中国移动/电信/联通三家运营商的 LTE (4G) 和 5G NR (SA) 制式。

### 2.2 架构决策

- **模块化单体 + 独立 ACS 引擎**（不使用微服务，不使用 go-zero）
- 三个部署单元：`omcgo-acs`（TR069 ACS 引擎）、`omcgo-app`（F02-F10 主应用）、`omcgo-worker`（后台工作进程）
- 消息队列：NATS JetStream；数据库：PostgreSQL 16 + TimescaleDB；缓存：Redis 7

### 2.3 四阶段总览

| 阶段 | 目标 | Sprint 数 | 对应设计文档 | 核心交付物 | 状态 |
|------|------|-----------|-------------|-----------|------|
| **Phase 1** 基础建设 | ACS 引擎能接收 Inform 并注册设备 | 5 | DD-01~04, 06, 07, 09, 19, 20a | 可运行的 ACS + 设备注册 | **已完成** |
| **Phase 2** 核心功能 | 完整设备管理和自动开站流程 | 4 | DD-05, 08, 10, 21 | 运营商适配 + 数据模型 + 自动开站 | 未开始 |
| **Phase 3** 数据管线 | PM/告警/MR 数据全链路 | 3 | DD-11, 12, 13 | 性能/告警/测量报告完整管线 | 未开始 |
| **Phase 4** 北向与规模化 | OSS 对接、10 万级验证、生产加固 | 5 | DD-14~18, 20b/c | 生产就绪系统 | 未开始 |

### 2.4 文档依赖 DAG

```
DD-01 (脚手架) ─────────────────────────────────────
  ├── DD-02 (基础设施) ──┐
  ├── DD-03 (领域模型) ──┤
  ├── DD-06 (TR069 库)   │
  ├── DD-19 (可观测性)   │
  └── DD-20 (部署)       │
                         │
  DD-02 + DD-03 ──→ DD-04 (事件总线)
  DD-03 ──────────→ DD-05 (运营商抽象)
                         │
  DD-02 + DD-04 + DD-06 → DD-07 (ACS 引擎)
  DD-02 + DD-03 + DD-05 → DD-08 (数据模型)
  DD-02~04 + DD-07 ─────→ DD-09 (设备管理)
                         │
  DD-07 + DD-08 + DD-09 → DD-10 (自动开站)
  DD-02 + DD-04 + DD-05 → DD-11 (性能管理)
  DD-02 + DD-04 + DD-07 → DD-12 (告警管理)
  DD-02 + DD-04 ────────→ DD-13 (测量报告)
  DD-07 + DD-09 ────────→ DD-14 (软件管理)
  DD-11 + DD-12 + DD-08 → DD-15 (北向接口)
  DD-05 + DD-09 ────────→ DD-16 (网元直连)
  DD-02 + DD-03 ────────→ DD-17 (RBAC)
  DD-07 + DD-08 ────────→ DD-18 (互操作测试)
  DD-02 ────────────────→ DD-21 (数据库迁移)
```

---

## 3. Phase 1 — 基础建设 ✅ 已完成

> **目标**：ACS 引擎能接收 CPE 的 SOAP Inform 报文，解析设备信息，完成设备注册，返回 InformResponse。
> **里程碑**：通过 CPE 模拟器验证完整 Inform → 设备注册 → 响应链路。
> **完成日期**：2026-03-05 | **Git Commit**: `d45abcc` | **完成度**: 95.7%（67/70 任务项）
> **完成报告**: `doc/reports/phase1-completion-report.md`

---

### Sprint 1.1 — 项目脚手架（DD-01）✅

**周期**: 3 人天 | **状态**: 已完成（7/8 项，README.md 延后）

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Go module 初始化，锁定 18 个核心依赖版本 | `go.mod`, `go.sum` |
| 2 | 创建完整目录结构（cmd/internal/pkg/api/migrations/configs/deployments/test） | 目录树 + `.gitkeep` |
| 3 | 编写 5 个 cmd 入口骨架（cobra + viper） | `cmd/acs/main.go`, `cmd/app/main.go`, `cmd/worker/main.go`, `cmd/migrate/main.go`, `cmd/omcctl/main.go` |
| 4 | 编写 3 份配置文件模板 | `configs/acs.yaml`, `configs/app.yaml`, `configs/worker.yaml` |
| 5 | 编写 Makefile（build/test/lint/generate/docker 目标） | `Makefile` |
| 6 | 配置 golangci-lint | `.golangci.yml` |
| 7 | 编写 .gitignore | `.gitignore` |
| 8 | 编写项目 README | `README.md` |

**前置依赖**: 无

**验收标准**:
- `go build ./cmd/...` 全部编译通过（各入口输出 "starting..." 后退出）
- `make lint` 零 warning
- `make test` 通过（即使尚无测试用例，也不报错）
- 所有目录结构与 CLAUDE.md 第 4 节定义一致

---

### Sprint 1.2 — 基础设施层 + 可观测性（DD-02 + DD-19）✅

**周期**: 5 人天 | **状态**: 已完成（11/11 项）

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | PostgreSQL 连接池（pgxpool） | `internal/infra/db/postgres.go` |
| 2 | TimescaleDB 连接池 + 扩展检查 | `internal/infra/db/timescale.go` |
| 3 | Redis 客户端（自动检测 Cluster/Standalone） | `internal/infra/cache/redis.go` |
| 4 | NATS JetStream 客户端 + 6 个 Stream 定义（DEVICE/COMMAND/PM/MR/ALARM/OSS） | `internal/infra/mq/nats.go` |
| 5 | MinIO 客户端 + Bucket 初始化（pm-files/mr-files/firmware/config-backup/logs） | `internal/infra/storage/minio.go` |
| 6 | 统一健康检查聚合器 | `internal/infra/health.go` |
| 7 | 优雅关闭编排器（按依赖逆序关闭） | `internal/infra/shutdown.go` |
| 8 | zap 日志初始化（JSON/Console 可切换） | `internal/infra/logger.go` |
| 9 | Prometheus 指标注册表 + HTTP Handler | `internal/infra/metrics.go` |
| 10 | OpenTelemetry TracerProvider 初始化 | `internal/infra/tracer.go` |
| 11 | 单元测试（各组件 mock 连接测试） | `internal/infra/*_test.go` |

**前置依赖**: Sprint 1.1

**验收标准**:
- `docker-compose up` 启动 PostgreSQL/Redis/NATS/MinIO 后，各连接组件 `HealthCheck()` 返回 nil
- 优雅关闭：发送 SIGTERM 后所有连接 30 秒内关闭，日志输出关闭顺序
- zap 日志输出格式符合字段规范（component/device_sn/carrier/duration_ms）
- Prometheus `/metrics` 端点返回默认 Go runtime 指标
- 单元测试覆盖率 ≥ 80%

---

### Sprint 1.3 — 公共领域模型 + 事件总线（DD-03 + DD-04）✅

**周期**: 5 人天 | **状态**: 已完成（14/14 项）

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | 领域常量（CarrierCode, Technology, DeviceStatus, AlarmSeverity, AlarmStatus） | `internal/common/model/constants.go` |
| 2 | Device 模型 | `internal/common/model/device.go` |
| 3 | Alarm 模型 | `internal/common/model/alarm.go` |
| 4 | Parameter 模型（DeviceParameter, ParameterDefinition） | `internal/common/model/parameter.go` |
| 5 | PMCounter + KPI 模型 | `internal/common/model/pm.go` |
| 6 | 通用分页/过滤结构（ListRequest, ListResponse） | `internal/common/model/pagination.go` |
| 7 | Sentinel errors + BusinessError（错误码体系 1000-6999） | `internal/common/errors/errors.go` |
| 8 | HTTP 中间件：认证、请求日志、Prometheus 指标、Panic 恢复 | `internal/common/middleware/auth.go`, `logging.go`, `metrics.go`, `recovery.go` |
| 9 | EventBus 接口定义 | `internal/common/event/bus.go` |
| 10 | ChannelEventBus 实现（进程内 Go channel） | `internal/common/event/channel_bus.go` |
| 11 | NATSEventBus 实现（NATS JetStream） | `internal/common/event/nats_bus.go` |
| 12 | Event Subject 常量定义 | `internal/common/event/subjects.go` |
| 13 | Event 类型定义（Event struct, EventHandler） | `internal/common/event/types.go` |
| 14 | 单元测试 | `internal/common/model/*_test.go`, `internal/common/event/*_test.go` |

**前置依赖**: Sprint 1.1, Sprint 1.2

**验收标准**:
- 所有领域常量值与 CLAUDE.md 第 5.1 节定义一致
- BusinessError 错误码范围分段合理（ACS 1000-1999, 配置 2000-2999, PM 3000-3999...）
- ChannelEventBus 测试：Publish → Subscribe 回调正确触发
- NATSEventBus 测试：连接 NATS 后 Publish/Subscribe 消息收发正常
- HTTP 中间件测试：请求日志包含 request_id/method/path/status/duration

---

### Sprint 1.4 — TR069 协议库 + ACS 引擎核心（DD-06 + DD-07）✅

**周期**: 10 人天 | **状态**: 已完成（22/23 项，gRPC 延后至 Phase 2）

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| **pkg/tr069 协议库** | | |
| 1 | TR069 核心类型（DeviceId, InformMessage, ParameterValueStruct, EventStruct） | `pkg/tr069/types.go` |
| 2 | 9 种 RPC 请求/响应类型定义 | `pkg/tr069/types.go`（续） |
| 3 | Inform 事件码常量（BOOTSTRAP/BOOT/PERIODIC/VALUE_CHANGE/ALARM/TRANSFER_COMPLETE 等） | `pkg/tr069/events.go` |
| 4 | CWMP 错误码（9000-9013） | `pkg/tr069/faults.go` |
| **pkg/soap SOAP 工具** | | |
| 5 | SOAP Envelope 结构体 | `pkg/soap/envelope.go` |
| 6 | SOAP 模板引擎（text/template 预编译模板集：InformResponse/GetParameterValues/SetParameterValues 等） | `pkg/soap/templates.go`, `pkg/soap/templates/*.xml` |
| 7 | SOAP 解码器（xml.Decoder 流式解析） | `pkg/soap/decoder.go` |
| **pkg/xmlutil XML 工具** | | |
| 8 | XML 流式读取工具 | `pkg/xmlutil/reader.go` |
| 9 | etree 导航辅助函数 | `pkg/xmlutil/etree_helper.go` |
| **internal/acs ACS 引擎** | | |
| 10 | ACS HTTP 服务器（监听 7547/7548 端口） | `internal/acs/server.go` |
| 11 | 请求处理主逻辑（限流 → 准入 → 认证 → 解析 → 分发 → 响应） | `internal/acs/handler.go` |
| 12 | 会话状态机（IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE） | `internal/acs/session.go` |
| 13 | Redis 会话存储（SessionStore 接口 + Redis 实现，TTL 5min） | `internal/acs/session_store.go` |
| 14 | CPE 认证（Digest/Basic） | `internal/acs/auth/authenticator.go` |
| 15 | RPC 调度器 + 9 种 RPC 方法实现 | `internal/acs/rpc/dispatcher.go`, `internal/acs/rpc/get_parameter_values.go` 等 |
| 16 | 设备命令队列（Redis Sorted Set 实现） | `internal/acs/cmdqueue/queue.go`, `internal/acs/cmdqueue/redis_queue.go` |
| 17 | Connection Request 客户端 | `internal/acs/connreq/client.go` |
| 18 | 设备级限流器（rate.Limiter） | `internal/acs/ratelimit.go` |
| 19 | 全局准入控制器（AdmissionController） | `internal/acs/admission.go` |
| 20 | ACS Prometheus 指标（active_sessions, inform_total, rpc_duration 等） | `internal/acs/metrics.go` |
| 21 | ~~gRPC 服务端实现（ACSControl service）~~ | ~~`api/proto/acs.proto`, `internal/acs/grpc_server.go`~~ | *延后：ACS ↔ App 通信已通过 NATS EventBus 实现* |
| 22 | 完善 cmd/acs 入口（接线所有组件） | `cmd/acs/main.go` |
| 23 | 单元测试 + 测试 fixtures（SOAP 报文样例） | `pkg/tr069/*_test.go`, `pkg/soap/*_test.go`, `internal/acs/*_test.go`, `test/fixtures/soap/*.xml` |

**前置依赖**: Sprint 1.2, Sprint 1.3

**验收标准**:
- ACS 启动后监听 7547 端口，接受 HTTP POST 请求
- 发送标准 SOAP Inform 报文，ACS 正确解析 DeviceId + EventList + ParameterList
- ACS 返回格式正确的 SOAP InformResponse（MaxEnvelopes=1）
- 会话状态机在 Redis 中正确创建和转移（可通过 redis-cli 查看 `acs:session:*`）
- 命令队列：Push/Pop/Peek 操作正确（通过 Redis `acs:cmdq:*` 验证）
- CPE 认证：Digest 认证流程通过（401 Challenge → Authorization header → 200）
- 限流器：超限请求返回 503
- 单元测试覆盖率 ≥ 75%

---

### Sprint 1.5 — 设备管理 + Docker 部署（DD-09 + DD-20a）✅

**周期**: 7 人天 | **状态**: 已完成（13/13 项）

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| **设备管理** | | |
| 1 | Phase 1 数据库迁移文件（devices + device_parameters 表） | `migrations/001_create_devices.up.sql`, `migrations/001_create_devices.down.sql`, `migrations/002_create_device_parameters.up.sql`, `migrations/002_create_device_parameters.down.sql` |
| 2 | cmd/migrate 迁移工具实现 | `cmd/migrate/main.go` |
| 3 | DeviceRepository 接口 + PostgreSQL 实现（squirrel 构建 SQL） | `internal/omcr/device/repository.go`, `internal/omcr/device/pg_repository.go` |
| 4 | DeviceParameterRepository 接口 + PostgreSQL 实现 | `internal/omcr/device/param_repository.go`, `internal/omcr/device/pg_param_repository.go` |
| 5 | DeviceService（RegisterFromInform, UpdateFromInform, TransitionStatus） | `internal/omcr/device/service.go` |
| 6 | 设备状态机（7 种状态 + 合法转移校验） | `internal/omcr/device/state_machine.go` |
| 7 | 心跳监控（Redis TTL + 离线检测 cron） | `internal/omcr/device/heartbeat.go` |
| 8 | 设备 REST API（列表/详情/参数查询/重启） | `internal/omcr/device/handler.go` |
| 9 | ACS 与设备管理的集成（Inform 事件 → 设备注册/更新） | ACS handler 中调用 DeviceService |
| 10 | App 入口接线（Gin router + 设备 API） | `cmd/app/main.go` |
| **Docker 部署** | | |
| 11 | 三个应用的 Dockerfile | `deployments/docker/Dockerfile.acs`, `Dockerfile.app`, `Dockerfile.worker` |
| 12 | docker-compose（PostgreSQL + TimescaleDB + Redis + NATS + MinIO + acs + app + worker） | `deployments/docker/docker-compose.yml` |
| 13 | 集成测试：完整 Inform → 设备注册链路 | `test/integration/inform_register_test.go` |

**前置依赖**: Sprint 1.4

**验收标准**:
- `make migrate-up` 成功创建 devices 和 device_parameters 表（含运营商分区）
- `make migrate-down` 成功回滚
- CPE 模拟器发送 Bootstrap Inform → ACS 解析 → 设备自动注册到 PostgreSQL → 状态为 `discovered`
- CPE 模拟器发送 Periodic Inform → 设备 `last_inform_at` 更新、心跳刷新
- `GET /api/v1/devices` 返回已注册设备列表（分页、按运营商/状态过滤）
- `GET /api/v1/devices/{id}` 返回设备详情
- `docker-compose up` 一键启动全部组件，ACS 可接收请求
- 设备状态转移校验：非法转移返回错误（如 discovered → active 不允许）
- 心跳过期设备自动标记为 offline

---

### Phase 1 里程碑验收 ✅

**端到端测试场景**：

| # | 场景 | 状态 | 实现位置 |
|---|------|------|---------|
| 1 | `docker-compose up` 启动全部基础设施 + 三个应用 | ✅ | `deployments/docker/docker-compose.yml` |
| 2 | CPE 模拟器向 `http://localhost:7547/acs` 发送 Bootstrap Inform（SOAP/XML） | ✅ | `internal/acs/handler.go` |
| 3 | ACS 解析 Inform，通过事件总线通知 App | ✅ | `handler.go` → `publishInformEvents` → EventBus |
| 4 | App 的 DeviceService 自动注册新设备到 PostgreSQL | ✅ | `inform_handler.go` → `service.go` → `pg_repository.go` |
| 5 | ACS 返回 InformResponse | ✅ | `pkg/soap/templates.go` InformResponseTmpl |
| 6 | CPE 模拟器发送后续 Periodic Inform | ✅ | `service.go` UpdateFromInform |
| 7 | 设备心跳刷新，`last_inform_at` 更新 | ✅ | `heartbeat.go` RecordHeartbeat |
| 8 | 通过 REST API `GET /api/v1/devices` 查询到已注册设备 | ✅ | `handler.go` HandleList |
| 9 | Prometheus `/metrics` 端点展示 `acs_inform_total` 等指标 | ✅ | `internal/acs/metrics.go` |

**构建验证**：
```
$ go build ./...    → 通过（5 个二进制，0 错误）
$ go test ./...     → 通过（6 个测试包，0 失败）
```

**未完成项（3 项，影响均为低）**：
1. `README.md` — 纯文档，不影响功能
2. `api/proto/acs.proto` + `internal/acs/grpc_server.go` — ACS ↔ App 通信已通过 NATS EventBus 实现，gRPC 为可选增强，延后至 Phase 2 按需补充

---

## 4. Phase 2 — 核心功能

> **目标**：完整设备管理和自动开站流程。设备上电后能自动发现、匹配模板、下发配置、验证激活。
> **里程碑**：CMCC LTE 设备从 Bootstrap 到 Active 的全自动开站。

---

### Sprint 2.1 — 数据库迁移框架（DD-21）

**周期**: 3 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | 完善 cmd/migrate（up/down/version/force 子命令） | `cmd/migrate/main.go`（完善） |
| 2 | Phase 2 迁移：数据模型定义表（含 partial unique index） | `migrations/003_create_data_model_definitions.up.sql` |
| 3 | Phase 2 迁移：OUI 注册表 + 种子数据 | `migrations/004_create_oui_registry.up.sql` |
| 4 | Phase 2 迁移：数据模型导入日志 | `migrations/005_create_data_model_import_log.up.sql` |
| 5 | Phase 2 迁移：配置模板表 | `migrations/006_create_config_templates.up.sql` |
| 6 | Phase 2 迁移：开站任务表 | `migrations/007_create_provisioning_tasks.up.sql` |
| 7 | Phase 2 迁移：设备分组表 | `migrations/008_create_device_groups.up.sql` |
| 8 | Phase 2 迁移：设备分组关联表 | `migrations/009_create_device_group_members.up.sql` |
| 9 | 所有迁移文件的 down 对（可回滚） | `migrations/*_*.down.sql` |
| 10 | 种子数据加载脚本 | `scripts/seed.sh`, `datamodels/seed/carrier_defaults/` |

**前置依赖**: Phase 1 完成

**验收标准**:
- `omcgo-migrate up` 执行所有迁移，`omcgo-migrate version` 显示 009
- `omcgo-migrate down` 逐条回滚到任意版本
- 所有表的索引、约束、分区正确创建
- OUI 种子数据成功导入

---

### Sprint 2.2 — 运营商抽象层（DD-05）

**周期**: 5 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Carrier 接口定义（12+ 方法） | `internal/carrier/carrier.go` |
| 2 | CarrierRegistry（注册/查找/列举） | `internal/carrier/registry.go` |
| 3 | CMCC 适配器（LTE + NR、参数映射、KPI 定义、告警映射、开站模板） | `internal/carrier/cmcc/adapter.go`, `internal/carrier/cmcc/params.go`, `internal/carrier/cmcc/kpi.go` |
| 4 | CTCC 适配器（LTE + NR 骨架实现） | `internal/carrier/ctcc/adapter.go` |
| 5 | CUCC 适配器（NR-only 骨架实现） | `internal/carrier/cucc/adapter.go` |
| 6 | 运营商差异矩阵测试 | `internal/carrier/*_test.go` |

**前置依赖**: Sprint 2.1

**验收标准**:
- `CarrierRegistry.Get("cmcc")` 返回 CMCC 适配器
- CMCC 适配器 `SupportedTechnologies()` 返回 [lte, nr]
- CMCC 参数映射：`MapParameterToUnified("Device.X_CMCC.xxx")` 正确返回统一名称
- CMCC KPI 定义包含 LTE（RRC 成功率、E-RAB 成功率等）和 NR（5G RRC 成功率等）公式
- CTCC/CUCC 骨架可编译，方法返回合理默认值
- **禁止**出现 `if carrier == "cmcc"` 硬编码

---

### Sprint 2.3 — 数据模型与配置管理（DD-08）

**周期**: 7 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | DataModel 数据结构（含参数树 JSONB） | `internal/config/datamodel/model.go` |
| 2 | DataModelRepository（PostgreSQL CRUD + FindActive + Activate/Deprecate） | `internal/config/datamodel/repository.go`, `internal/config/datamodel/pg_repository.go` |
| 3 | DataModelCache（Redis L2 缓存） | `internal/config/datamodel/cache.go` |
| 4 | DataModelRegistry（三级回退解析：product → oui → carrier_default + L1 内存缓存） | `internal/config/datamodel/registry.go` |
| 5 | 数据模型导入/导出（JSON 文件 ↔ DB） | `internal/config/datamodel/importer.go` |
| 6 | 数据模型生命周期（draft → active → deprecated，激活时自动废弃旧模型） | 集成在 Repository + Registry 中 |
| 7 | 缓存版本号机制（`datamodel:cache_version` 跨实例协调） | 集成在 Cache 中 |
| 8 | ConfigTemplate 配置模板管理 | `internal/config/template/service.go`, `internal/config/template/repository.go` |
| 9 | 数据模型 REST API（导入/导出/激活/废弃/解析测试/差异对比/统计） | `internal/config/datamodel/handler.go` |
| 10 | 配置模板 REST API | `internal/config/template/handler.go` |
| 11 | CMCC LTE/NR 默认数据模型种子 | `datamodels/seed/carrier_defaults/cmcc_lte.json`, `cmcc_nr.json` |
| 12 | 单元测试（三级回退解析、缓存命中/未命中、生命周期转换） | `internal/config/datamodel/*_test.go` |

**前置依赖**: Sprint 2.2

**验收标准**:
- 导入 CMCC LTE 数据模型 JSON → 数据库创建记录（draft 状态）
- 激活数据模型 → 同类旧模型自动置为 deprecated
- 三级回退：有 product 级别则返回 product 模型，无则回退 oui，再回退 carrier_default
- 三级缓存：首次查询走 PostgreSQL → 写入 Redis L2 + 内存 L1 → 再次查询命中 L1
- `datamodel:cache_version` 变更时所有 ACS 实例检测到并刷新缓存
- REST API `/api/v1/datamodels/resolve?carrier=cmcc&tech=lte&oui=xxx&product_class=yyy` 返回解析结果

---

### Sprint 2.4 — 自动开站（DD-10）

**周期**: 7 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | 开站任务模型（ProvisioningTask + 6 种状态） | `internal/provision/model.go` |
| 2 | ProvisioningTaskRepository（PostgreSQL） | `internal/provision/repository.go`, `internal/provision/pg_repository.go` |
| 3 | 模板匹配引擎（按 carrier + tech + product_class 匹配） | `internal/provision/workflow/matcher.go` |
| 4 | 开站状态机（discovered → identifying → matching → configuring → verifying → completed/failed） | `internal/provision/workflow/state_machine.go` |
| 5 | ProvisioningEngine（HandleBootstrap 事件入口、StartProvisioning、AdvanceState） | `internal/provision/workflow/engine.go` |
| 6 | 配置下发编排（GetParameterValues → SetParameterValues → Download → Reboot → Verify 命令序列） | `internal/provision/workflow/orchestrator.go` |
| 7 | 拓扑管理（设备分组 CRUD + 树查询 + 设备分配） | `internal/omcr/topology/service.go`, `internal/omcr/topology/repository.go`, `internal/omcr/topology/handler.go` |
| 8 | 批量开站 API | `internal/provision/handler.go` |
| 9 | EventBus 集成（订阅 device.inform.bootstrap → 自动触发开站） | 集成在 ProvisioningEngine |
| 10 | 集成测试 | `test/integration/provisioning_test.go` |

**前置依赖**: Sprint 2.3

**验收标准**:
- CPE Bootstrap Inform → 自动创建 ProvisioningTask → 状态推进到 completed
- 模板匹配：CMCC LTE 设备匹配到对应配置模板
- 命令序列正确入列到 Redis 命令队列（acs:cmdq:*）
- 状态机校验：非法状态转移返回错误
- 失败重试：配置下发失败时 RetryCount 递增，超过阈值标记 failed
- REST API 查询开站任务列表和详情
- 拓扑管理：创建分组 → 分配设备 → 查询树形结构

---

### Phase 2 里程碑验收

**端到端测试场景**：

1. 导入 CMCC LTE 默认数据模型 + 配置模板（通过种子脚本或 REST API）
2. CPE 模拟器发送 Bootstrap Inform
3. ACS 解析 → 设备注册（discovered）→ 数据模型解析（三级回退匹配）
4. 自动开站触发 → 模板匹配 → 命令序列入列
5. CPE 模拟器响应 GetParameterValues/SetParameterValues/Download 命令
6. 开站验证通过 → 设备状态 → active
7. 查询设备列表确认状态变更

---

## 5. Phase 3 — 数据管线

> **目标**：PM 计数器采集 → KPI 计算 → 时序存储、告警全生命周期、MR 文件解析，全链路打通。
> **里程碑**：Worker 进程能处理 PM/MR 文件，告警实时接收和去重。
> **说明**：Sprint 3.1 / 3.2 / 3.3 可并行开发，无相互依赖。

---

### Sprint 3.1 — 性能管理（DD-11）

**周期**: 7 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Phase 3 迁移：pm_counters 超表（TimescaleDB） | `migrations/010_create_pm_counters.up.sql` |
| 2 | Phase 3 迁移：kpi_definitions + kpi_values 超表 | `migrations/011_create_kpi_tables.up.sql` |
| 3 | PM XML 流式解析器（运营商感知，调用 Carrier 接口） | `internal/pm/collector/parser_xml.go` |
| 4 | PMCollector（订阅 pm.file.received → MinIO 下载 → 解析 → 存储） | `internal/pm/collector/collector.go` |
| 5 | CounterRepository（批量写入 TimescaleDB） | `internal/pm/counter/repository.go` |
| 6 | KPI 计算引擎（公式注册 + 计数器依赖解析 + 计算执行） | `internal/pm/kpi/engine.go` |
| 7 | KPI 公式注册（LTE: RRC 成功率/E-RAB/掉话率/切换/PRB/吞吐；NR: 5G RRC/掉话/时延/速率） | `internal/pm/kpi/formulas.go` |
| 8 | 时间维度聚合（15min → 1h，使用 TimescaleDB 连续聚合） | `internal/pm/aggregation/aggregator.go` |
| 9 | 数据保留策略（7 天压缩、90 天保留） | 迁移文件中配置 |
| 10 | PM REST API（查询计数器/KPI 值/KPI 定义/导出） | `internal/pm/handler.go` |
| 11 | Worker 入口集成 PM 采集管线 | `cmd/worker/main.go` |
| 12 | 测试 fixtures（PM XML 样例文件） + 单元测试 | `test/fixtures/pm/*.xml`, `internal/pm/*_test.go` |

**前置依赖**: Phase 2 完成

**验收标准**:
- 上传 PM XML 文件到 MinIO → 发送 pm.file.received 事件 → Worker 解析并写入 TimescaleDB
- `SELECT * FROM pm_counters WHERE device_id = ?` 返回正确的计数器记录
- KPI 计算：给定 RRC 连接尝试次数和成功次数 → 正确计算 RRC 成功率
- 聚合查询：15 分钟粒度 → 1 小时聚合值正确
- REST API 支持按设备/小区/时间范围查询 PM 数据
- 90 天保留策略配置正确

---

### Sprint 3.2 — 告警管理（DD-12）

**周期**: 5 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Phase 3 迁移：alarms_active + alarms_history 超表 | `migrations/012_create_alarms.up.sql` |
| 2 | AlarmReceiver（订阅 device.inform.alarm 事件） | `internal/alarm/receiver.go` |
| 3 | AlarmEngine（去重、关联分析、抑制） | `internal/alarm/engine.go` |
| 4 | AlarmStore（Redis 活跃告警 Hash + PostgreSQL 活跃表 + TimescaleDB 历史超表） | `internal/alarm/store.go` |
| 5 | AlarmForwarder（可选北向转发） | `internal/alarm/forwarder.go` |
| 6 | 告警生命周期（active → acknowledged → cleared，自动清除） | 集成在 AlarmEngine |
| 7 | 运营商告警严重级别映射（通过 Carrier 接口） | 集成在 AlarmEngine |
| 8 | 告警 REST API（活跃列表/历史列表/确认/清除/统计） | `internal/alarm/handler.go` |
| 9 | 单元测试 | `internal/alarm/*_test.go` |

**前置依赖**: Phase 2 完成

**验收标准**:
- ACS 接收含 ALARM 事件码的 Inform → 告警提取 → AlarmEngine 处理 → Redis + PostgreSQL 存储
- 去重：同一设备相同告警码的重复告警只更新时间戳，不创建新记录
- 生命周期：告警确认 → acknowledged；告警清除 → cleared → 移入历史表
- Redis `alarm:active:{device_sn}` 正确维护活跃告警
- REST API 查询活跃告警/历史告警，支持按严重级别过滤

---

### Sprint 3.3 — 测量报告（DD-13）

**周期**: 5 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Phase 3 迁移：mr_files + mr_data 表 | `migrations/013_create_mr_tables.up.sql` |
| 2 | MR 文件类型检测（从文件名判断 MRO/MRS/MRE） | `internal/mr/collector/detector.go` |
| 3 | MRCollector（订阅 mr.file.received → MinIO 下载 → 分发到对应解析器） | `internal/mr/collector/collector.go` |
| 4 | MRO 解析器（切换优化数据） | `internal/mr/parser/mro_parser.go` |
| 5 | MRS 解析器（服务小区数据） | `internal/mr/parser/mrs_parser.go` |
| 6 | MRE 解析器（终端能力数据） | `internal/mr/parser/mre_parser.go` |
| 7 | MR 数据存储（PostgreSQL + MinIO 原始文件） | `internal/mr/store.go` |
| 8 | MR REST API（文件列表/下载/查询解析数据） | `internal/mr/handler.go` |
| 9 | Worker 入口集成 MR 采集管线 | `cmd/worker/main.go`（增强） |
| 10 | 测试 fixtures + 单元测试 | `test/fixtures/mr/*.xml`, `internal/mr/*_test.go` |

**前置依赖**: Phase 2 完成

**验收标准**:
- 上传 MRO/MRS/MRE XML 文件到 MinIO → Worker 正确识别类型并解析
- MinIO 存储路径：`mr-files/{carrier}/{date}/{device_sn}/mr_{type}_{timestamp}.xml`
- 解析后的 MR 数据写入 PostgreSQL
- REST API 支持按设备/类型/时间范围查询 MR 文件和解析数据
- 运营商差异：CUCC 不支持 MRE 解析（跳过或返回不支持提示）

---

### Phase 3 里程碑验收

**端到端测试场景**：

1. PM 管线：上传 PM XML → NATS 通知 → Worker 解析 → TimescaleDB 存储 → KPI 计算 → API 查询
2. 告警管线：CPE Inform ALARM → 告警提取 → 去重 → 存储 → API 查询 → 确认 → 清除
3. MR 管线：上传 MRO XML → NATS 通知 → Worker 解析 → 存储 → API 查询

---

## 6. Phase 4 — 北向与规模化

> **目标**：OSS 接口对接、完整用户权限、固件升级、互操作测试、K8s 部署、10 万级压测。
> **里程碑**：系统通过 100K 设备并发压力测试，所有功能域完整可用。

---

### Sprint 4.1 — 用户管理与 RBAC（DD-17）

**周期**: 5 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Phase 4 迁移：users + roles + user_roles + permissions + audit_logs 表 | `migrations/014_create_users.up.sql`, `migrations/015_create_roles.up.sql` |
| 2 | AdminService（Login/RefreshToken/User CRUD/Role CRUD） | `internal/omcr/admin/service.go` |
| 3 | JWT 令牌管理（access_token 30min + refresh_token 7d） | `internal/omcr/admin/jwt.go` |
| 4 | UserRepository + RoleRepository（PostgreSQL） | `internal/omcr/admin/user_repo.go`, `internal/omcr/admin/role_repo.go` |
| 5 | RBAC 权限检查（resource + action 对） | `internal/omcr/admin/permission.go` |
| 6 | 运营商数据隔离（用户绑定 carrier，查询自动过滤） | 集成在 middleware |
| 7 | 操作审计日志（AuditRepository + 中间件自动记录） | `internal/omcr/admin/audit.go` |
| 8 | Gin 中间件：RequireAuth、RequirePermission、RequireCarrier | `internal/omcr/admin/middleware.go` |
| 9 | 管理 REST API（登录/用户/角色/权限/审计查询） | `internal/omcr/admin/handler.go` |
| 10 | 默认管理员种子数据 | `scripts/seed_admin.sh` |
| 11 | 单元测试 | `internal/omcr/admin/*_test.go` |

**前置依赖**: Phase 3 完成

**验收标准**:
- `POST /api/v1/auth/login` 返回 JWT token pair
- 带 token 访问 API → 正常；不带 token → 401
- RBAC：admin 角色可操作所有资源；viewer 角色只能 GET
- 运营商隔离：cmcc 用户只能看到 cmcc 设备
- 审计日志：所有写操作记录到 audit_logs 表

---

### Sprint 4.2 — 软件管理（DD-14）

**周期**: 5 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | Phase 4 迁移：firmware_versions + upgrade_tasks 表 | `migrations/016_create_firmware.up.sql` |
| 2 | SoftwareService（UploadFirmware/StartUpgrade/BatchUpgrade/HandleTransferComplete） | `internal/omcr/software/service.go` |
| 3 | FirmwareRepository + UpgradeTaskRepository（PostgreSQL） | `internal/omcr/software/firmware_repo.go`, `internal/omcr/software/upgrade_repo.go` |
| 4 | 升级状态机（pending → downloading → rebooting → verifying → completed/failed） | `internal/omcr/software/state_machine.go` |
| 5 | MinIO 固件存储（`firmware/{carrier}/{product_class}/{version}/firmware.bin`） | 集成在 SoftwareService |
| 6 | 批量升级调度（滚动升级、失败隔离） | `internal/omcr/software/batch.go` |
| 7 | 固件管理 REST API（版本列表/上传/触发升级/跟踪任务） | `internal/omcr/software/handler.go` |
| 8 | 单元测试 | `internal/omcr/software/*_test.go` |

**前置依赖**: Sprint 4.1

**验收标准**:
- 上传固件文件 → MinIO 存储 → 数据库记录版本信息
- 触发升级 → Download RPC 入列 → CPE 下载完成 → TransferComplete → 验证版本 → completed
- 批量升级：50 台设备滚动升级，失败设备不影响其他设备
- REST API 跟踪升级任务进度

---

### Sprint 4.3 — 北向/OSS 接口 + 网元直连（DD-15 + DD-16）

**周期**: 7 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| **北向接口** | | |
| 1 | NorthboundRouter（PM/Alarm/Config 三大模块路由） | `internal/northbound/api/router.go` |
| 2 | PM 数据接口（统计查询、文件导出） | `internal/northbound/api/pm_handler.go` |
| 3 | 告警数据接口（实时推送、全量/增量同步） | `internal/northbound/api/alarm_handler.go` |
| 4 | 配置数据接口（设备配置快照、变更推送） | `internal/northbound/api/config_handler.go` |
| 5 | PushEngine（HTTP 回调推送、JSON/XML 格式、重试） | `internal/northbound/push/engine.go` |
| 6 | SyncService（FullSync + IncrementalSync） | `internal/northbound/sync/service.go` |
| **网元直连（移动专有）** | | |
| 7 | NEDirectServer（独立 HTTP 服务器） | `internal/nedirect/server.go` |
| 8 | 直连 Handler（Register/Config/Status/Fault） | `internal/nedirect/handler.go` |
| 9 | 运营商门控（仅 CMCC 启用） | 集成在配置 + Carrier 接口 |
| 10 | 单元测试 | `internal/northbound/*_test.go`, `internal/nedirect/*_test.go` |

**前置依赖**: Sprint 4.1

**验收标准**:
- OSS 回调：告警触发 → PushEngine 向配置的 URL 发送 HTTP POST → 收到 200
- 全量同步：`POST /api/v1/northbound/sync/full` 返回所有 PM/告警/配置数据
- 增量同步：`POST /api/v1/northbound/sync/incremental?since=...` 返回增量数据
- 网元直连：CMCC 配置启用 → NEDirectServer 监听 → 接收 Register 请求
- 非 CMCC 运营商不启动 NEDirectServer

---

### Sprint 4.4 — 互操作测试（DD-18）

**周期**: 3 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | ConformanceTestRunner（按类别运行测试用例） | `internal/interop/runner.go` |
| 2 | 测试用例集（protocol/datamodel/rpc/inform 四类） | `internal/interop/cases/` |
| 3 | DataModelValidator（设备参数 vs 数据模型对比：类型/范围/可写性） | `internal/interop/validator.go` |
| 4 | ValidationReport（matched/mismatched/missing/extra） | `internal/interop/report.go` |
| 5 | 互操作测试 REST API | `internal/interop/handler.go` |
| 6 | 单元测试 | `internal/interop/*_test.go` |

**前置依赖**: Sprint 4.1

**验收标准**:
- 对已注册设备运行一致性测试 → 生成 ValidationReport
- 协议测试：Inform 报文格式校验通过
- 数据模型测试：对比设备实际参数与模型定义，报告匹配/不匹配/缺失

---

### Sprint 4.5 — K8s 部署与规模化（DD-20b/c）

**周期**: 7 人天

**工作项**:

| # | 任务 | 产出文件 |
|---|------|---------|
| 1 | K8s Namespace 定义（omcgo） | `deployments/k8s/namespace.yaml` |
| 2 | ACS Deployment（2-20 副本 + HPA 基于 acs_active_sessions） | `deployments/k8s/acs-deployment.yaml`, `acs-hpa.yaml` |
| 3 | App Deployment（2-5 副本 + HPA 基于 CPU） | `deployments/k8s/app-deployment.yaml`, `app-hpa.yaml` |
| 4 | Worker Deployment（2-10 副本 + KEDA ScaledObject 基于 NATS 队列深度） | `deployments/k8s/worker-deployment.yaml`, `worker-keda.yaml` |
| 5 | Service + Ingress 定义 | `deployments/k8s/services.yaml`, `ingress.yaml` |
| 6 | ConfigMap + Secret 模板 | `deployments/k8s/configmap.yaml`, `secret.yaml` |
| 7 | 基础设施 StatefulSet（PostgreSQL/Redis/NATS/MinIO 生产配置指引） | `deployments/k8s/infra/` |
| 8 | 10 万级负载测试脚本（模拟 100K 设备并发 Inform） | `scripts/loadtest.go` 或集成压测工具 |
| 9 | 监控面板（Grafana Dashboard JSON） | `deployments/monitoring/grafana-dashboard.json` |
| 10 | 生产部署文档 | `doc/operations/deployment-guide.md` |

**前置依赖**: Sprint 4.2, 4.3, 4.4

**验收标准**:
- `kubectl apply -f deployments/k8s/` 部署所有组件
- ACS HPA：并发会话超过 2000/实例时自动扩容
- Worker KEDA：NATS 队列积压时自动扩 Worker 副本
- 负载测试：10K 设备并发 Inform，ACS 平均延迟 < 100ms，99th < 500ms
- 负载测试：100K 设备 5 分钟周期 Inform（333 sessions/s），系统稳定运行
- Grafana 面板展示关键指标（活跃会话、Inform 速率、RPC 延迟、设备总数、告警数）

---

### Phase 4 里程碑验收

**端到端测试场景**：

1. 管理员登录 → JWT 认证 → RBAC 权限校验
2. 固件上传 → 触发批量升级 → 监控升级进度 → 全部完成
3. OSS 接口：告警推送到外部系统、PM 数据增量同步
4. CMCC 网元直连接口正常工作
5. 互操作一致性测试通过
6. K8s 部署全部组件 → 100K 设备压力测试通过

---

## 7. 集成测试计划

每个阶段完成后执行对应的集成测试套件：

| 阶段 | 测试场景 | 测试文件 |
|------|---------|---------|
| Phase 1 | Inform 接收 → 设备注册 → REST API 查询 | `test/integration/inform_register_test.go` |
| Phase 1 | 会话状态机完整流转 | `test/integration/session_lifecycle_test.go` |
| Phase 1 | 命令队列 Push → RPC 执行 → 响应 | `test/integration/command_queue_test.go` |
| Phase 2 | 数据模型三级回退解析 | `test/integration/datamodel_resolve_test.go` |
| Phase 2 | 完整自动开站流程 | `test/integration/provisioning_e2e_test.go` |
| Phase 3 | PM 文件 → 解析 → KPI 计算 | `test/integration/pm_pipeline_test.go` |
| Phase 3 | 告警接收 → 去重 → 确认 → 清除 | `test/integration/alarm_lifecycle_test.go` |
| Phase 3 | MR 文件 → 解析 → 存储 | `test/integration/mr_pipeline_test.go` |
| Phase 4 | JWT 认证 + RBAC + 运营商隔离 | `test/integration/auth_rbac_test.go` |
| Phase 4 | 固件升级完整流程 | `test/integration/firmware_upgrade_test.go` |
| Phase 4 | 北向数据推送 | `test/integration/northbound_push_test.go` |
| E2E | 设备从上电到全功能运行 | `test/e2e/device_full_lifecycle_test.go` |

---

## 8. 开发环境要求

### 8.1 必备工具

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.22+ | 编译运行 |
| Docker | 24+ | 容器化 |
| Docker Compose | v2+ | 本地开发环境 |
| golangci-lint | 1.56+ | 代码检查 |
| protoc | 3.21+ | gRPC 代码生成 |
| protoc-gen-go | 1.32+ | Go protobuf 生成 |
| protoc-gen-go-grpc | 1.3+ | Go gRPC 生成 |
| golang-migrate | 4.17+ | 数据库迁移 CLI |
| swag | 1.16+ | Swagger 生成 |

### 8.2 本地基础设施（docker-compose 提供）

| 组件 | 版本 | 端口 |
|------|------|------|
| PostgreSQL + TimescaleDB | 16 + latest | 5432 |
| Redis | 7 | 6379 |
| NATS | 2.10+ | 4222, 8222 |
| MinIO | latest | 9000, 9001 |

### 8.3 应用端口分配

| 服务 | HTTP | HTTPS | gRPC | Metrics |
|------|------|-------|------|---------|
| omcgo-acs | 7547 | 7548 | — | 9090 |
| omcgo-app | 8080 | 8443 | 50051 | 9091 |
| omcgo-worker | — | — | — | 9092 |

---

## 9. 质量门禁

### 9.1 代码质量

| 指标 | 标准 |
|------|------|
| 单元测试覆盖率 | ≥ 75%（核心模块 ≥ 85%：ACS 引擎、状态机、三级回退） |
| golangci-lint | 零 error/warning |
| 测试通过率 | 100%（CI 必须全绿） |
| 竞态检测 | `go test -race` 无 data race |

### 9.2 代码审查

| 要求 | 说明 |
|------|------|
| PR 必须审查 | 至少 1 人 approve |
| Commit 规范 | 遵循 Conventional Commits |
| 分支策略 | feature/fix/refactor 从 main 创建 → PR → squash merge |
| CI 检查 | lint + test + build 全部通过才可合并 |

### 9.3 文档要求

| 要求 | 说明 |
|------|------|
| 公共接口 | 必须有 GoDoc 注释 |
| REST API | 必须有 Swagger 注解 |
| 配置变更 | 必须更新对应 YAML 模板 |
| 数据库变更 | 必须通过 migration 文件 |

---

## 10. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| TR069 协议复杂度高 | ACS 引擎开发周期延长 | 高 | 优先实现 Inform/Get/Set 三个最常用 RPC；准备 CPE 模拟器；参照规范文档逐步补全 |
| 运营商规范差异大 | 运营商适配工作量超预期 | 高 | Phase 2 只做 CMCC 完整适配，CTCC/CUCC 骨架实现；Phase 3 补全 |
| TimescaleDB 性能瓶颈 | PM 数据写入延迟 | 中 | 批量写入（1000 条/批）；合理分区和压缩策略；预留分表方案 |
| SOAP/XML 解析性能 | ACS 高并发时内存/CPU 压力 | 中 | 使用 xml.Decoder 流式解析（不加载全 XML）；text/template 预编译响应模板 |
| 多进程间事件一致性 | 消息丢失或重复处理 | 中 | NATS JetStream 至少一次保证 + 业务幂等设计；关键操作加数据库事务 |
| K8s 部署复杂度 | 生产环境配置困难 | 低 | Phase 1-3 使用 docker-compose；Phase 4 逐步引入 K8s；提供详细部署文档 |
| 外部依赖升级 | 依赖库 breaking change | 低 | go.mod 锁定主版本号；CI 定期依赖检查 |

---

## 11. 工作量汇总

| 阶段 | Sprint | 预估人天 | 可并行 | 状态 |
|------|--------|---------|--------|------|
| **Phase 1** | 1.1 项目脚手架 | 3 | — | ✅ 完成 |
| | 1.2 基础设施 + 可观测性 | 5 | — | ✅ 完成 |
| | 1.3 领域模型 + 事件总线 | 5 | — | ✅ 完成 |
| | 1.4 TR069 库 + ACS 引擎 | 10 | — | ✅ 完成 |
| | 1.5 设备管理 + 部署 | 7 | — | ✅ 完成 |
| | **小计** | **30** | | **已完成** |
| **Phase 2** | 2.1 数据库迁移 | 3 | — |
| | 2.2 运营商抽象层 | 5 | — |
| | 2.3 数据模型与配置 | 7 | — |
| | 2.4 自动开站 | 7 | — |
| | **小计** | **22** | |
| **Phase 3** | 3.1 性能管理 | 7 | 3.1/3.2/3.3 可并行 |
| | 3.2 告警管理 | 5 | |
| | 3.3 测量报告 | 5 | |
| | **小计** | **17** (并行可压缩到 7) | |
| **Phase 4** | 4.1 RBAC | 5 | — |
| | 4.2 软件管理 | 5 | 4.2/4.3/4.4 可并行 |
| | 4.3 北向 + 直连 | 7 | |
| | 4.4 互操作测试 | 3 | |
| | 4.5 K8s + 规模化 | 7 | — |
| | **小计** | **27** (并行可压缩到 19) | |
| **总计** | | **96 人天** (并行优化后约 78 人天) | |

---

## 12. 附录：关联文档索引

| 文档 | 路径 |
|------|------|
| 详细设计索引 | `doc/detailed-design/README.md` |
| 后端架构设计 | `doc/architecture/backend-design.md` |
| 系统总览 | `doc/architecture/system-overview.md` |
| 接口拓扑 | `doc/architecture/interface-topology.md` |
| 框架选型分析 | `doc/architecture/framework-comparison.md` |
| 功能索引 | `doc/功能索引.md` |
| 功能域详情 | `doc/features/01~10-*.md` |
| 规范目录 | `doc/specs-inventory/document-catalog.md` |
| 运营商对比 | `doc/specs-inventory/carrier-comparison.md` |
| 项目指导 | `CLAUDE.md` |

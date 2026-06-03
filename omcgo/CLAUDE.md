# CLAUDE.md — OMC Go 项目指导

> 本文件为 AI 编码助手提供项目上下文。所有开发工作必须与本文件描述的架构决策保持一致。

---

## 1. 项目简介

**OMC**（Operations, Management and Control）是面向小基站/皮基站/微基站的无线操作维护中心系统。

| 维度 | 说明 |
|------|------|
| 核心协议 | TR069/CWMP（SOAP/XML over HTTP） |
| 运营商 | 中国移动（cmcc）、中国电信（ctcc）、中国联通（cucc） |
| 制式 | LTE (4G)、5G NR (SA) |
| 规模 | 10 万基站起步，预留 100 万级扩展 |
| 功能域 | 10 个域（F01-F10），43 项子功能 |

### 功能域速览

| 编号 | 功能域 | 核心职责 |
|------|--------|---------|
| F01 | 南向接口（TR069） | ACS 引擎，SOAP/XML 协议处理，设备通信通道 |
| F02 | 数据模型与配置 | 参数模型字典（XML→param_models / param_mappings / discovered_param_mappings）+ Translator 双向翻译 + 配置模板（T-0098 后旧 datamodel 三级回退已下线） |
| F03 | 性能管理（PM/KPI） | 计数器采集，KPI 计算，时序存储 |
| F04 | 告警管理 | 告警接收、去重、关联、生命周期 |
| F05 | 测量报告（MR） | MRO/MRS/MRE 文件采集与解析 |
| F06 | OMC-R 核心 | 拓扑管理、设备生命周期、固件升级、RBAC |
| F07 | 网元直连 | 网元与网管直连通道（移动专有） |
| F08 | 北向/OSS 接口 | 向 OSS 系统开放 PM/告警/配置数据 |
| F09 | 自动开站 | 设备自动发现、模板匹配、配置下发 |
| F10 | 互操作测试 | 设备联调与一致性验证 |

---

## 2. 架构决策

### 核心选型：模块化单体 + 独立 ACS 引擎

**不使用微服务架构，不使用 go-zero。**

| 决策 | 理由 |
|------|------|
| 选择模块化单体 | 100K 规模单 Go 进程足够（333 sessions/s）；10 个功能域跨域交互密集；避免分布式事务复杂度 |
| ACS 独立部署 | TR069 SOAP/XML 有状态会话、独立扩展需求、并发模型与管理面不同 |
| 拒绝 go-zero | go-zero 仅支持 JSON/Protobuf，无法处理 SOAP/XML；NATS 不在其生态；ACS（占 40% 代码）无法使用 goctl 代码生成 |
| 选择 NATS 而非 Kafka | Go 原生、轻量、吞吐足够，运维成本低 |

### 三个部署单元

```
omcgo-acs     — TR069 ACS 引擎（独立进程，水平可扩展）
omcgo-app     — 主应用（F02-F10 模块化单体）
omcgo-worker  — 后台工作进程（PM/MR 文件处理、KPI 计算）
```

### 演进路径

当前阶段保持模块化单体。到 100 万规模时可按功能域渐进拆分为独立服务。

---

## 3. 技术栈

### 核心框架与库

| 组件 | 选型 | 用途 |
|------|------|------|
| 语言 | Go 1.22+ | |
| HTTP（ACS） | `net/http` stdlib | TR069 SOAP/XML 处理，完全控制请求生命周期 |
| HTTP（管理面） | `github.com/gin-gonic/gin` | REST API，北向接口，Web 管理 |
| RPC | `google.golang.org/grpc` | ACS ↔ App 进程间通信 |
| XML/SOAP 发送 | `text/template` | 预编译 SOAP 模板，避免反射开销 |
| XML/SOAP 接收 | `encoding/xml` Decoder | 流式解析，不加载整个 XML 到内存 |
| XML 动态遍历 | `github.com/beevik/etree` | 参数树动态处理 |
| 配置管理 | `github.com/spf13/viper` | YAML + 环境变量 + 热重载 |
| CLI | `github.com/spf13/cobra` | omcctl 命令行管理工具 |
| 日志 | `go.uber.org/zap` | 结构化高性能日志 |
| 指标 | `github.com/prometheus/client_golang` | Prometheus 指标暴露 |
| 链路追踪 SDK | `go.opentelemetry.io/otel` | 进程内 trace 产生与 OTLP gRPC 上报 |
| Trace 采集器 | `otel/opentelemetry-collector-contrib:0.103.0` | 接收 SDK 上报、batch/retry/缓冲，转推后端（T-0155 Phase 1） |
| Trace 后端 | `grafana/tempo:2.5.0` | trace 存储与查询，Grafana Tempo 数据源原生集成 + trace-to-logs 跳 Loki + metrics_generator 生 RED 指标（T-0155 收尾） |
| 基础设施指标采集 | otelcol contrib 0.103 内置 `postgresqlreceiver` + `redisreceiver` → metricstransform 别名 → prometheusremotewrite → Prometheus | postgres-exporter / redis-exporter 已下线（T-0155 Phase 2b） |
| 日志采集 | otelcol filelog receiver (`/run/logs/*`) → loki exporter → Loki | Promtail 已下线（T-0155 Phase 3） |
| trace-to-logs 关联 | `logger.L(ctx)` 从 OTel context 抽 `trace_id` / `span_id` 注入 zap 字段（T-0157） | 经 Tracing middleware 的请求日志自动带 trace_id；Grafana Tempo → Loki 精确跳转 |
| DB 驱动 | `github.com/jackc/pgx/v5` | PostgreSQL 高性能驱动 |
| 连接池 | `github.com/jackc/pgx/v5/pgxpool` | 数据库连接池 |
| Redis | `github.com/redis/go-redis/v9` | 缓存、会话、命令队列 |
| 消息队列 | `github.com/nats-io/nats.go` | NATS JetStream |
| 对象存储 | `github.com/minio/minio-go/v7` | MinIO/S3 |
| 数据库迁移 | `pressly/goose/v3` | Schema 版本管理 |
| SQL 构建 | `github.com/Masterminds/squirrel` | 动态 SQL 构建（不使用 ORM） |
| 参数验证 | `github.com/go-playground/validator/v10` | 结构体校验 |
| UUID | `github.com/google/uuid` | UUID 生成 |
| 定时任务 | `github.com/robfig/cron/v3` | PM 采集、聚合等周期任务 |
| 限流 | `golang.org/x/time/rate` | 设备级/全局限流 |
| 测试 | `github.com/stretchr/testify` | 断言与 Mock |
| API 文档 | `github.com/swaggo/swag` | Swagger 自动生成 |

### 数据存储

| 数据类型 | 存储 |
|---------|------|
| 设备、配置、拓扑、用户 | PostgreSQL 16（JSONB 支持灵活 Schema） |
| PM 计数器 & KPI 时序 | TimescaleDB（PostgreSQL 扩展） |
| 历史告警 | TimescaleDB 超表 |
| TR069 会话状态 | Redis 7 Cluster（TTL 自动过期） |
| 设备任务队列 | Redis Sorted Set + PostgreSQL（`device_tasks` 表） |
| 数据模型缓存 | Redis + 内存 L1 |
| PM/MR/固件/备份文件 | MinIO（S3 兼容） |
| 进程内事件 | Go channel |
| 持久化消息 | NATS JetStream |

---

## 4. 项目目录结构

```
omcgo/
├── cmd/                            # 入口
│   ├── app/
│   │   ├── main.go                 # 主应用（~150 行，基础设施初始化 + 调用 router.Setup）
│   │   ├── etc/                    # 配置文件（dev/test/prod）
│   │   │   ├── config.dev.yaml
│   │   │   ├── config.test.yaml
│   │   │   └── config.prod.yaml
│   │   └── router/                 # 路由注册 + DI 容器
│   │       ├── deps.go
│   │       └── router.go
│   ├── acs/
│   │   ├── main.go                 # TR069 ACS 引擎
│   │   └── etc/                    # 配置文件（dev/test/prod）
│   ├── worker/
│   │   ├── main.go                 # 后台工作进程
│   │   └── etc/                    # 配置文件（dev/test/prod）
│   ├── migrate/main.go             # 数据库迁移
│   └── omcctl/main.go              # CLI 管理工具
│
├── global/                         # 全局常量与错误码（无框架依赖）
│   ├── consts.go                   #   运营商/设备/告警等全局常量
│   └── errors.go                   #   63 个错误码（纯数值常量）
│
├── internal/                       # 私有代码（按功能域组织，扁平化结构）
│   ├── appconfig/                  # 配置结构体 + 加载逻辑
│   │
│   ├── acs/                        # F01: TR069 ACS 引擎
│   │   ├── server.go               #   HTTP 服务器
│   │   ├── handler.go              #   请求处理
│   │   ├── session.go              #   会话状态机
│   │   ├── soap/                   #   SOAP 编解码
│   │   ├── rpc/                    #   RPC 方法
│   │   ├── connreq/                #   Connection Request
│   │   ├── rpc/                    #   RPC 方法与 Command（SOAP 渲染入参）
│   │   └── auth/                   #   CPE 认证
│   │
│   ├── config/                     # F02: 数据模型与配置管理（业务域）
│   │   ├── parammodel/             #   参数模型字典（T-0098 P1-P5 替代旧 datamodel）
│   │   │                              · Loader → param_models / param_mappings / standard_params
│   │   │                              · Registry → discovered + default 双源映射
│   │   │                              · Translator → standardPath ↔ privatePath O(1) 双向翻译
│   │   │                              · IntersectService → 写 discovered_param_mappings
│   │   │                              · MappingValidator → 元属性校验（access / range / change_applies）
│   │   ├── template/               #   配置模板
│   │   ├── baseline/               #   配置基线
│   │   └── sync_handler.go         #   配置同步
│   │
│   ├── product/                    # F02: 产品装配件 + ProductRegistry（T-0098 P2-01）
│   │                                  · productClass 全局正则路由 → product
│   │                                  · param_model_id / indicator_platform / alarm_ne_type 装配
│   │                                  · enable_filetype11 / enable_unknown_alarm 三态策略
│   │
│   ├── pm/                         # F03: 性能管理
│   ├── alarm/                      # F04: 告警管理
│   ├── mr/                         # F05: 测量报告
│   │
│   ├── device/                     # F06: 设备管理与生命周期（← omcr/device）
│   ├── admin/                      # F06: 用户管理与 RBAC（← omcr/admin）
│   ├── topology/                   # F06: 设备拓扑与分组（← omcr/topology）
│   ├── software/                   # F06: 固件管理（← omcr/software）
│   ├── backup/                     # F06: 配置备份（← omcr/backup）
│   ├── dashboard/                  # F06: 仪表盘（← omcr/dashboard）
│   ├── ops/                        # F06: 运维工具（← omcr/ops）
│   ├── report/                     # F06: 报表（← omcr/report）
│   ├── mml/                        # F06: MML 控制台（← omcr/mml）
│   ├── filemanager/                # F06: 文件管理（← omcr/filemanager）
│   ├── syslog/                     # F06: 系统日志（← omcr/syslog）
│   ├── license/                    # F06: 许可证（← omcr/license）
│   │
│   ├── nedirect/                   # F07: 网元直连
│   ├── northbound/                 # F08: 北向/OSS 接口
│   ├── provision/                  # F09: 自动开站
│   ├── interop/                    # F10: 互操作测试
│   │
│   ├── carrier/                    # 运营商抽象层
│   │   ├── cmcc/                   #   中国移动
│   │   ├── ctcc/                   #   中国电信
│   │   └── cucc/                   #   中国联通
│   │
│   ├── model/                      # 共享领域类型（← common/model）
│   ├── errors/                     # 业务错误 + gin 集成（← common/errors）
│   ├── event/                      # EventBus 抽象（← common/event）
│   ├── middleware/                  # HTTP 中间件（← common/middleware）
│   │
│   ├── components/                 # 基础设施适配器（← infra/）
│   │   ├── postgres/               #   PostgreSQL/TimescaleDB（← infra/db）
│   │   ├── redis/                  #   Redis（← infra/cache）
│   │   ├── nats/                   #   NATS（← infra/mq）
│   │   ├── minio/                  #   MinIO（← infra/storage）
│   │   ├── logger/                 #   Zap 日志（← infra/logger.go）
│   │   ├── monitor/                #   Prometheus 指标（← infra/metrics.go）
│   │   ├── health.go               #   健康检查
│   │   ├── sysinfo.go              #   系统���息
│   │   ├── tracer.go               #   OpenTelemetry
│   │   └── shutdown.go             #   优雅关机
│   │
│   └── utils/                      # 工具函数
│
├── pkg/                            # 可复用公共库
│   ├── tr069/                      #   TR069 类型、事件码
│   ├── soap/                       #   通用 SOAP 工具
│   └── xmlutil/                    #   XML 辅助工具
│
├── api/                            # API 定义
├── migrations/                     # 数据库迁移文件（goose 格式）
│   ├── 000NNN_description.sql      #   表结构迁移（DDL），严格连续递增
│   └── seed/                       #   种子数据迁移（DML），紧接主目录版本号继续递增
├── configs/                        # 压测专用配置（acs-stress.yaml）
├── datamodels/                     # TR069 数据模型种子数据
├── deployments/                    # 部署清单
├── doc/                            # 项目文档
├── scripts/                        # 脚本
├── test/                           # 集成/E2E 测试
├── go.mod
├── go.sum
├── Makefile
└── CLAUDE.md                       # 本文件
```

---

## 5. 开发规范

### 5.1 Go 编码规范

**命名**：
- 遵循 Go 标准：exported 用 PascalCase，unexported 用 camelCase
- 接口用名词或动词（`SessionStore`, `EventPublisher`），不加 `I` 前缀
- 包名简短、小写、单数（`alarm` 不是 `alarms`）

**领域常量**：
```go
// 运营商代码（全局统一使用这些常量，不要硬编码字符串）
CarrierCMCC CarrierCode = "cmcc"  // 中国移动
CarrierCTCC CarrierCode = "ctcc"  // 中国电信
CarrierCUCC CarrierCode = "cucc"  // 中国联通

// 制式
TechLTE Technology = "lte"
TechNR  Technology = "nr"

// 数据模型 Scope（解析优先级从高到低）
ScopeProduct        = "product"         // OUI + ProductClass 级
ScopeOUI            = "oui"             // 厂商级
ScopeCarrierDefault = "carrier_default" // 运营商默认级
```

**错误处理**：
- 返回 `error`，不 `panic`（除非不可恢复的初始化错误）
- Wrap error 加上下文：`fmt.Errorf("resolve data model: %w", err)`
- ACS 会话内的错误不能影响其他会话（goroutine 隔离）
- 使用 sentinel error 和自定义错误类型区分业务错误和系统错误

**接口优先**：
- 核心领域概念用接口定义，便于测试和替换实现：
  - `Carrier` — 运营商适配
  - `SessionStore` — TR069 会话存储
  - `TaskService` — 统一任务队列（Redis + PG 双写）
  - `EventBus` — 事件总线
  - `DataModelRepository` — 数据模型持久层

**运营商适配**：
- 通过 `Carrier` 接口 + 适配器模式处理运营商差异
- **禁止** `if carrier == "cmcc"` 硬编码，所有差异逻辑通过适配器实现
- 新增运营商 = 新增适配器 + 注册到 `CarrierRegistry`

**数据库访问**：
- squirrel 构建 SQL + pgx/v5 执行，**不使用 ORM**
- 每个模块有自己的 repository 层
- 复杂查询直接写 SQL，简单 CRUD 用 squirrel

**测试**：
- `testify` 断言，table-driven tests
- 单元测试与源文件同目录（`_test.go`）
- 集成测试放 `test/integration/`，E2E 测试放 `test/e2e/`
- 测试数据放 `test/fixtures/`

### 5.2 TR069/ACS 专项规范

**SOAP/XML 处理**：
- 发送方向（ACS → CPE）：`text/template` 预编译模板，避免 `encoding/xml` 反射
- 接收方向（CPE → ACS）：`xml.Decoder` 流式解析，不加载整个 XML 到内存
- 动态参数树遍历：`beevik/etree`

**会话状态机**：
```
IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE
```

**Redis Key 命名**：
```
acs:session:{device_serial}          — 会话状态 Hash（TTL 5 分钟）
acs:taskq:{device_serial}             — 任务队列 Sorted Set（member=taskID）
acs:task:{taskID}                     — 任务详情 Hash（TTL 24h）
acs:cwmp2task:{hash}                  — CWMP ID → Task ID 映射（TTL 24h）
acs:heartbeat:{device_serial}        — 心跳时间戳（TTL = 2×inform_interval）
acs:connreq:pending:{device_serial}  — Connection Request 去重（TTL 30 秒）
parammodel:default:{paramModelID}                       — ParamRegistry default mapping 缓存（TTL 24h）
parammodel:discovered:{productID}:{swVersion}           — ParamRegistry discovered mapping 缓存（TTL 1h）
product:byProductClass:{productClass}                   — ProductRegistry 路由结果缓存（TTL 1h）
parammodel:cache_version                                — 缓存版本号（跨实例协调）
alarm:active:{device_serial}         — 活跃告警 Hash
ratelimit:inform:{device_serial}     — 限流计数器
```

**流量控制**：
- 每设备限流器（`rate.Limiter`）防止 Inform 洪泛
- 全局准入控制器（`AdmissionController`）限制并发会话数
- PM/MR 文件处理使用 WorkerPool 控制并发

### 5.3 参数模型字典专项规范（T-0098 P1-P5）

旧 datamodel 三级回退（product / oui / carrier_default）+ data_model_definitions 表已下线（migrations/000063 DROP）。新栈走 ParamModel 字典：

**装配件（product / param_model / mapping）**：
1. **products** 表（migrations/000057）：装配件聚合产品类、参数模型、KPI 平台、告警 ne_type、上传开关
2. **param_models / param_mappings**（migrations/000058）：默认映射，按 paramModel 加载
3. **discovered_param_mappings**：设备 FileType=11 上传 XML 后由 IntersectService 写入；按 (product_id, sw_version) 索引
4. **standard_params**：standardPath 元属性参考表

**路由**：
- 设备上报 productClass → ProductRegistry.MatchProductClass（全局正则）→ product
- product.param_model_id → ParamRegistry.GetByProduct(productID, swVersion)
  - 优先 discovered_param_mappings（精确匹配 swVersion）
  - 退化 param_mappings（默认映射）
  - 双源合并 → MappingSet（含 Source 标记 discovered/default）

**双向翻译（Translator）**：
- standardPath（IETF / 标准化）↔ privatePath（厂商专有）
- 模板 / SPV / GPV 输入侧用 standardPath，下发 / 持久化用 privatePath
- {i} 占位符规范化：运行时实例号 ".N." 与模板 ".{i}." 折叠为同一索引键

**缓存**：
1. 内存 L1（`sync.Map`，进程内 Registry）
2. Redis L2（`parammodel:*` / `product:*` keys）
3. PostgreSQL（多表）

**Loader 启动期加载**（dictloader 框架）：
- 4 个 Loader：product / parammodel / indicator / alarm-definition
- 文件白名单 → 单事务幂等 UPSERT
- ModuleGraph 编排：dictload → productregistry → paramregistry → 等

#### 5.3.1 ParamModel 自定义 XML 分层目录（T-0178）

> **⚠️ 2026-06-03 已变更（用户决策，下方历史描述部分作废）**：
> 「导入 XML / 重载 XML / 刷新缓存」三功能合并为单个 **「导入 XML」**。
> - 取消 builtin/custom 目录区分：上传**直接写 builtin 目录** `param-mappings/`（**接受升级丢失**，不再写 `*-custom`）；同名直接覆盖（前端上传前查重 + 覆盖确认，旧文件备份 `.bak.<ts>`）。
> - 上传端点内部串联 **destructive 重载（删孤儿）+ 刷新缓存**；前端不再单独调用。
> - 删除 HTTP 端点 `POST /param-models/import-directory`、`POST /param-models/cache/refresh`（底层 reload/cache 逻辑保留，供上传流程内部调用）。
> - `source.go::IsDeletable` 恒 true（全部可删），前端去掉「来源」列、删除按钮恒可点。
> 下面 T-0178 关于 custom 分层目录、`IsDeletable` 守门、来源列的描述均为历史背景，**以本注记为准**。

**核心契约**：builtin XML 与 custom XML **物理隔离**两个目录,Loader 启动期合并扫描;后端唯一真值源 + 前端零代码同步规则改动。

| 目录 | 位置 | 来源 | 生命周期 |
|------|------|------|---------|
| `data/param-mappings/` | 镜像层 `COPY` 进 `/etc/omcgo/data` | 发版构建 | 跟随镜像版本 |
| `data/param-mappings-custom/` | host bind mount `/opt/omc/data/...` | 运维通过 UI 上传 | 跟随 host(升级不丢) |

**5 项设计决策**（5 轮 ULTRATHINK 已敲定,改前看历史而非提出新选项）：
1. `custom_overrides_builtin = true`（默认 custom 胜出;删 custom 自动回退 builtin 的 self-healing 链路依赖此项）
2. builtin 行删除按钮 UI 上**置灰 + Tooltip**(不隐藏 — 让用户知道功能存在但不可用)
3. `.deleted.<ts>` / `.bak.<ts>` 备份保留 **30 天**(worker cron `0 3 * * *`);`.tmp.<uuid>` 残留 1 小时即清
4. DELETE 备份失败 → 全流程**保守回滚**(500 `ErrCodeParamModelBackupFailed=2031`,不删 DB)
5. source 唯一真值源在后端(`source.go::IsDeletable`);前端只渲染 API 返回的 `deletable` bool

**唯一真值源链路**(改判定规则只动 `source.go`,无需改前端):
```
Loader.loadParamModelFile
  → resolveLoadedFrom(base, absPath) 写 "param-mappings/X.xml" 或 "param-mappings-custom/X.xml"
  → param_models.loaded_from 列
  → ClassifySource(loadedFrom) 返 builtin/custom/unknown
  → IsDeletable(loadedFrom) 守门 DELETE handler + 填充 modelView.deletable
  → 前端 ModelsTab 渲染来源 Tag + 删除按钮可见性
```

**端到端 commit 链路**(8 个 commit 合计 +2960/-99 LOC):
- P1 后端: 58640fb3(source 分类器) → 944c6d4c(Loader 双目录) → d1349c3c(Delete 守门 + 迁移 215) → 16f95d46(Upload + DTO source/deletable) → 61a0e5aa(*bool 三态 yaml)
- P2 worker: 2c27253b(BackupCleanup cron + Prometheus 指标)
- P3 部署: aab7245b(bind mount + deploy.sh 初始化)
- P4 前端: 84cf6950(Upload 按钮 + 来源列 + 置灰)

**单实例假设**:Upload/Delete/单文件 Reload 通过 `acquireFileLock(basename)` 进程内 `sync.Map[name]*sync.Mutex` 互斥。**多实例横扩前必须**补 PG advisory lock `pg_try_advisory_xact_lock(hashtext('parammodel:'+basename))`,否则同名 Upload 会丢更新。

**新增错误码**(`global/errors.go` Data Model 块 2030-2039 段):
- `ErrCodeParamModelBuiltinNotDeletable = 2030` — DELETE 内置返 403
- `ErrCodeParamModelBackupFailed = 2031` — DELETE 备份失败保守回滚返 500

**新增 Prometheus 指标**:
- `parammodel_backup_cleanup_total{kind="deleted|bak|tmp", result="swept|error|skipped"}` (worker)

**相关代码索引**:
- `internal/config/parammodel/source.go` — Source / ClassifySource / IsDeletable + 4 个共享常量
- `internal/config/parammodel/filelist.go` — mergeFileLists / resolveLoadedFrom / resolveLoaderFiles
- `internal/config/parammodel/upload.go` — Upload 4 个纯函数校验器(文件名/XML/路径/保留名)
- `internal/config/parammodel/handler.go` — DeleteModel 守门 / UploadXML / acquireFileLock
- `internal/config/parammodel/backup_cleanup.go` — BackupCleanup + 指标 + 3 个默认常量
- `internal/config/parammodel/loader.go` — Loader.run 双目录合并 + loadParamModelFile 写前缀
- `migrations/000215_param_models_loaded_from_prefix.sql` — `loaded_from` 历史数据回填
- `cmd/worker/main.go::startParamModelBackupCleanup` — cron 注册 + 30s catch-up
- `omcmb/frontend-core/src/types/paramModel.ts` — ParamModelSource union + source/deletable
- `omcmb/webcode/src/pages/product/param-model/{index,ModelsTab}.tsx` — Upload + 来源 + 置灰

#### 5.3.2 Indicator 自定义 XML 分层目录（T-0180）

> **⚠️ 2026-06-03 已变更（用户决策，下方历史描述部分作废）**：同 5.3.1。
> 「导入/重载/刷新缓存」合并为单个 **「导入 XML」**；上传**直接写 builtin** `indicator-library/`（ENB→`enb/<name>.xml`，GSM/GNB→根级 `GSM.xml`/`GNB.xml`，**接受升级丢失**）；上传端点内串联 **destructive 重载（`PerformReloadWithOrphans` 删孤儿）+ BumpCacheVersion**；删除 `POST /indicators/import-directory`、`POST /indicators/cache/refresh`；`IsDeletable` 恒 true，前端去「来源」列、全可删。以本注记为准。

**核心契约**:builtin XML 与 custom XML 物理隔离两套目录,Loader 启动期合并扫描;后端唯一真值源 + 前端零代码同步。结构与 T-0178 ParamModel 同范式,**关键差异**:custom 侧三制式子目录化(`enb/gsm/gnb/`)而非扁平。

| 目录 | 位置 | 来源 | 生命周期 |
|------|------|------|---------|
| `data/indicator-library/` | 镜像层 `COPY` 进 `/etc/omcgo/data` | 发版构建 | 跟随镜像版本 |
| └── `enb/*.xml` | 镜像层 | LTE 多文件(ALL/BLQ/...) | 同上 |
| └── `GSM.xml` | 镜像层 | GSM 单文件(根级) | 同上 |
| └── `GNB.xml` | 镜像层 | GNB 单文件(根级) | 同上 |
| `data/indicator-library-custom/` | host bind mount `/opt/omc/data/...` | 运维通过 UI 上传 | 跟随 host(升级不丢) |
| └── `enb/*.xml` | host | 用户自定义 LTE 指标 | 同上 |
| └── `gsm/*.xml` | host | 用户自定义 GSM 指标(全统一子目录化) | 同上 |
| └── `gnb/*.xml` | host | 用户自定义 NR 指标 | 同上 |

**5 项设计决策**(D1-D5 锁定):
1. `custom_overrides_builtin = true`(默认 custom 胜出;`indicator-library/enb/ALL.xml` ↔ `indicator-library-custom/enb/ALL.xml` 同名时 custom 替换)
2. 强制 `<indicatorModel platform="..." [deviceType="..."]>` 根元素:`platform` 必填;`deviceType` 若 present 必须(大小写不敏感)匹配 `?tech=` query,absent 则容忍(ENB legacy XML 历史无 deviceType)
3. `.deleted.<ts>` / `.bak.<ts>` 备份保留 **30 天**(worker cron `0 3 * * *`);`.tmp.<纳秒ts>` 残留 1 小时即清
4. DELETE 备份失败 → 全流程**保守回滚**(500 `ErrCodeIndicatorBackupFailed=2041`,不删 DB)
5. source 唯一真值源在后端(`source.go::IsDeletable`);前端只渲染 API 返回的 `deletable` bool

**唯一真值源链路**(改判定规则只动 `source.go`):
```
Loader.parseDocs
  → fileSource{AbsPath, LoadedFrom="indicator-library/enb/X.xml" or "indicator-library-custom/enb/Y.xml"}
  → indicatorRecord{Ind, LoadedFrom}(first-seen 锁定)
  → perf_indicators_{enb,gsm,gnb}.loaded_from 列写入
  → ClassifySource(loadedFrom) 返 builtin/custom/unknown
  → IsDeletable(loadedFrom) 守门 DELETE handler + 填充 fileEntry.deletable
  → 前端 XMLFilesModal 渲染来源 Tag + 删除按钮可见性
```

**端到端 commit 链路**(9 个 commit 合计 +5313/-211 LOC):
- 立项: `35ee4d51`(PRD + 设计文档入库)
- P1.1 后端: `69101bcb`(source 分类器 73 LOC + 29 sub-test)
- P1.2 后端: `351cf0b2`(Loader 双目录 + filelist + migration 217 加 loaded_from 三表)
- P1.3 后端: `311bde18`(Delete 守门 + 6 错误码 2040-2045 + 22 sub-test)
- P1.4 后端: `a58f7833`(Upload + Summary + Files 3 端点 + 38 sub-test)
- P1.5 后端: `c929a760`(import-directory ?mode=reload 三表孤儿删除 + 12 sub-test)
- P2 worker: `96ca95bd`(BackupCleanup 三制式子目录 + indicator_backup_cleanup_total + 16 sub-test)
- P3 部署: `1b742aa7`(dev/prod compose volume + deploy.sh mkdir 三子目录)
- P4 前端: `df2dd814`(business 层 4 件套 + UI 壳 6 文件 drill-down + URL 同步)

**单实例假设**:Upload/Delete/单文件 Reload 通过 `acquireFileLock(basename)` 进程内 `sync.Map[name]*sync.Mutex` 互斥。**多实例横扩前必须**补 PG advisory lock(R-NEW-T0180-5 Open)。

**新增错误码**(`global/errors.go` Data Model 块 2040-2049 段):
- `ErrCodeIndicatorBuiltinNotDeletable = 2040` — DELETE 内置返 403
- `ErrCodeIndicatorBackupFailed = 2041` — DELETE 备份失败保守回滚返 500
- `ErrCodeIndicatorUploadInvalidTech = 2042` — upload-xml ?tech= 非 enb/gsm/gnb → 400
- `ErrCodeIndicatorUploadInvalidName = 2043` — 文件名违规 → 400
- `ErrCodeIndicatorUploadInvalidRoot = 2044` — XML 根非 `<indicatorModel>` 或 D2 校验失败 → 400
- `ErrCodeIndicatorUploadTooLarge = 2045` — 文件 > 1 MiB → 400

**新增 Prometheus 指标**:
- `indicator_backup_cleanup_total{kind="deleted|bak|tmp", result="swept|error|skipped"}` (worker)

**新增 HTTP 端点**(/api/v1):
- `GET /indicators/summary` — 三制式聚合(builtin/custom/groups/platforms)
- `GET /indicators/files?tech=enb|gsm|gnb` — 该制式 XML 文件列表(DB GROUP BY + 盘扫描合并)
- `POST /indicators/upload-xml?tech=&force=` — multipart 上传,同步 Loader.Reload
- `POST /indicators/import-directory?mode=import|reload` — mode=reload 加孤儿删除
- `DELETE /indicators/files/{loadedFrom}` — IsDeletable 守门 + 三表级联

**相关代码索引**:
- `internal/pm/indicator/source.go` — Source / ClassifySource / IsDeletable + 4 共享常量
- `internal/pm/indicator/filelist.go` — resolveENBSources / resolveSingleTechSources(三制式双源合并)
- `internal/pm/indicator/upload.go` — Upload 4 个纯函数校验器(filename/tech/XML root+D2/path-containment)
- `internal/pm/indicator/file_handler.go` — FileHandler:DeleteFile / Summary / ListFiles / UploadXML + acquireFileLock + EnsureBaseDir
- `internal/pm/indicator/file_repository.go` — FileRepository 接口 + Pg 实现(Count/Delete/Summary/ListFiles/DeleteOrphansBefore)
- `internal/pm/indicator/backup_cleanup.go` — BackupCleanup(三制式子目录步进)+ 指标
- `internal/pm/indicator/reload.go` — PerformReloadWithOrphans + ParseReloadMode
- `internal/pm/indicator/loader.go` — Loader.run 双目录合并 + parseDocs 写前缀 + flushIndicators 写 loaded_from
- `internal/pm/indicator/rest_handler.go` — ImportDirectory 加 ?mode= 分支(P1.5)
- `migrations/000217_perf_indicators_loaded_from.sql` — perf_indicators_{enb,gsm,gnb} ADD COLUMN loaded_from
- `cmd/worker/main.go::startIndicatorBackupCleanup` — cron 注册 + 30s catch-up
- `cmd/app/provider/pm.go` — RESTHandler + FileHandler 接入 + EnsureBaseDir 启动期
- `deployments/docker/docker-compose.yml` — dev bind mount(app + worker)
- `deployments/release/bundle/deploy/docker-compose.app.yml` — prod bind mount(app + worker)
- `deployments/release/bundle/deploy/deploy.sh` — 首次部署 mkdir 三制式 + chown 10001 + chmod 0750
- `omcmb/frontend-core/src/types/indicatorLibrary.ts` — TechLower + 双向映射 + 5 个新 type
- `omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts` — +5 API 方法 + importDirectory(mode) 改造
- `omcmb/frontend-core/src/hooks/api/useIndicatorsLibrary.ts` — +5 React Query hooks
- `omcmb/webcode/src/pages/product/kpi-library/{index,SummaryTab,IndicatorsByTech,XMLFilesModal,UploadXmlModal,UnitsDrawer}.tsx` — drill-down + URL 同步

#### 5.3.3 Alarm 自定义 XML 分层目录（严格对标 T-0180 indicator）

> **⚠️ 2026-06-03 已变更（用户决策，下方历史描述部分作废）**：同 5.3.1。
> 「导入/重载/刷新缓存」合并为单个 **「导入 XML」**；上传**直接写 builtin** `alarm-definitions/<name>.xml`（**接受升级丢失**）；上传端点内串联 **destructive 重载（删孤儿 `DeleteOrphansSince`）+ RefreshCache**；删除 `POST /alarm-definitions/import-directory`、`POST /alarm-definitions/cache/refresh`；`IsDeletable` 恒 true，前端去「来源」列、全可删。以本注记为准。

**核心契约**:builtin XML 与 custom XML 物理隔离两套目录,Loader 启动期合并扫描;后端唯一真值源 + 前端零代码同步。与 T-0180 indicator 同范式,**关键差异**:告警按 ne_type 组织、custom 目录**扁平**(无 enb/gsm/gnb 子目录);"自定义"对应用户上传的 XML 文件。

| 目录 | 位置 | 来源 | 生命周期 |
|------|------|------|---------|
| `data/alarm-definitions/` | 镜像层 `COPY` 进 `/etc/omcgo/data` | 发版构建(每 ne_type 一个文件,如 ENB.xml) | 跟随镜像版本 |
| `data/alarm-definitions-custom/` | host bind mount `/opt/omc/data/...` | 运维通过 UI 上传(扁平,`<file>.xml`) | 跟随 host(升级不丢) |

**设计决策**(沿用 T-0180 D1-D5):custom 默认胜出;内置删除按钮置灰 + Tooltip;`.deleted/.bak` 保留 30 天 + `.tmp` 1 小时(worker cron `0 3 * * *`);DELETE 备份失败保守回滚;source 唯一真值源在后端。

**唯一真值源链路**(改判定只动 `source.go`):
```
Loader.resolveSources（builtin + custom 双目录合并,custom 默认胜出）
  → loadAlarmFile 写 loaded_from = "alarm-definitions/ENB.xml" 或 "alarm-definitions-custom/MY.xml"（含目录前缀）
  → alarm_definitions.loaded_from 列
  → ClassifySource(loadedFrom) 返 builtin/custom/unknown
  → IsDeletable(loadedFrom) 守门 DELETE handler + 填充 NeTypeStat.Deletable
  → 前端 alarm-library NeTypes 表渲染来源 Tag + 删除按钮可见性
```

**单实例假设**:Upload/Delete 通过 `acquireFileLock(basename)` 进程内 `sync.Map[name]*sync.Mutex` 互斥;多实例横扩前需补 PG advisory lock(与 indicator 同 Open)。

**新增错误码**(`global/errors.go` 2050 段):
- `ErrCodeAlarmBuiltinNotDeletable = 2050` — DELETE 内置返 403
- `ErrCodeAlarmBackupFailed = 2051` — DELETE 备份失败保守回滚返 500
- `ErrCodeAlarmUploadInvalidName = 2052` — 文件名违规 → 400
- `ErrCodeAlarmUploadInvalidRoot = 2053` — XML 根非 `<alarmModel>` → 400
- `ErrCodeAlarmUploadTooLarge = 2054` — 文件 > 1 MiB → 400

**新增 Prometheus 指标**:`alarm_backup_cleanup_total{kind="deleted|bak|tmp", result="swept|error|skipped"}` (worker)

**新增 HTTP 端点**(/api/v1,super_admin):
- `POST /alarm-definitions/upload-xml[?force=]` — multipart 上传,同步 Loader.Reload + RefreshCache
- `DELETE /alarm-definitions/files/{*loadedFrom}` — IsDeletable 守门 + 按 loaded_from 删 alarm_definitions
- `GET /alarm-definitions/ne-types`(既有)新增返 `source` / `deletable` 字段

**相关代码索引**:
- `internal/alarm/definition/source.go` — Source / ClassifySource / IsDeletable + 4 共享常量(前缀 `alarm-definitions(-custom)/`)
- `internal/alarm/definition/upload.go` — 上传校验器(filename / `<alarmModel>` root / path-containment / 限长读)
- `internal/alarm/definition/file_handler.go` — FileHandler:UploadXML / DeleteFile + acquireFileLock + EnsureBaseDir + validCustomPath
- `internal/alarm/definition/file_repository.go` — FileRepository(CountByLoadedFrom / DeleteByLoadedFrom,单表)
- `internal/alarm/definition/backup_cleanup.go` — BackupCleanup(扁平 customDir 单层扫描)+ 指标
- `internal/alarm/definition/loader.go` — Loader.resolveSources 双目录合并 + 写前缀 loaded_from
- `internal/alarm/definition/{repository,pg_repository}.go` — NeTypeStat.Source/Deletable 查询层派生
- `internal/core/appconfig/config.go` — AlarmDefinitionLoaderConfig 加 CustomDirectory / CustomOverrides / Backup*
- `migrations/seed/000006_alarm_definitions_loaded_from_prefix.sql` — loaded_from 历史数据回填前缀
- `cmd/app/provider/alarmdef.go` — FileHandler 接入 + EnsureBaseDir 启动期
- `cmd/worker/main.go::startAlarmBackupCleanup` — cron 注册 + 30s catch-up
- `deployments/docker/docker-compose.yml` + `release/bundle/deploy/docker-compose.app.yml` — app+worker bind mount
- `deployments/release/bundle/deploy/deploy.sh` — 首次部署 mkdir + chown 10001 + chmod 0750
- `omcmb/frontend-core/src/{types/alarmDefinition,services/api/alarmDefinitionApi,hooks/api/useAlarmDefinitions}.ts` — 类型 + uploadXml/deleteFile + hooks
- `omcmb/webcode/src/pages/product/alarm-library/{index,AlarmUploadXmlModal}.tsx` — 上传按钮 + 来源/加载源列 + 删除列

### 5.4 事件驱动规范

**EventBus 双实现**：
- `ChannelEventBus`：进程内 Go channel（单进程部署）
- `NATSEventBus`：NATS JetStream（多实例部署）

**事件 Subject 命名**（点分层级）：
```
device.inform.bootstrap          — 新设备发现
device.inform.periodic           — 心跳
device.inform.value_change       — 参数变更
device.inform.alarm              — 告警事件
command.get_parameters           — 读取参数
command.set_parameters           — 写入参数
pm.file.received / pm.file.parsed
mr.file.received / mr.file.parsed
alarm.raised / alarm.cleared / alarm.acknowledged
oss.alarm.forward / oss.pm.export
```

### 5.5 数据库迁移规范

**迁移工具**：`pressly/goose/v3`，版本记录在数据库 `goose_db_version` 表中。

**版本号规则（CRITICAL）**：

| 规则 | 说明 |
|------|------|
| **严格连续递增** | 版本号必须从现有最大值 +1，禁止跳跃、禁止重复 |
| **禁止重用已用版本号** | 即使旧迁移已删除，其版本号也不得再用 |
| **禁止插入低版本迁移** | 数据库当前版本之后才能添加新迁移 |

**文件命名格式**：

```
migrations/
├── 000NNN_description.sql          # 表结构迁移（DDL）
└── seed/                           # 种子数据迁移（DML）
    ├── 000MMM_seed_data.sql
    └── ...
```

**新增迁移步骤**：

1. 检查本地文件最大版本号：`ls migrations/ migrations/seed/ | sort | tail -5`
2. 新文件版本号 = 本地文件最大版本号 + 1
3. DDL 变更放 `migrations/`，DML 种子数据放 `migrations/seed/`

**多人协作注意事项（CRITICAL）**：

- 合并代码时务必检查是否有版本号冲突（多人同时新增相同版本号的迁移文件）
- 合并后执行前确认版本号无重复：`ls migrations/ migrations/seed/ | sort | uniq -d`
- 如果发现重复，将后合并的文件重命名为更大版本号

**迁移文件格式**：

```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE IF EXISTS ...;
```

**幂等性要求**：
- `CREATE TABLE IF NOT EXISTS`、`ADD COLUMN IF NOT EXISTS`
- 使用 `DO $$ BEGIN ... EXCEPTION WHEN ... END $$` 包裹可能重复的 DDL

#### 5.5.1 goose StatementBegin/End 注解规则（CRITICAL）

goose 默认以分号分割 SQL 语句。以下场景 **必须** 使用 `-- +goose StatementBegin` / `-- +goose StatementEnd` 包裹，否则 goose 会将内部语句截断导致解析失败：

| 场景 | 示例 |
|------|------|
| PL/pgSQL 匿名块 | `DO $$ ... END $$;` |
| 创建函数 | `CREATE OR REPLACE FUNCTION ... $$ ... $$ LANGUAGE plpgsql;` |
| 创建触发器函数 | 同上 |
| 循环/条件语句 | `FOR ... IN ... LOOP ... END LOOP;` |

```sql
-- +goose Up
-- 正确示例：DO 块必须包裹
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        CREATE EXTENSION IF NOT EXISTS timescaledb;
    END IF;
END $$;
-- +goose StatementEnd
```

> **历史教训**：6 个迁移文件因缺少此注解导致 goose panic（commit `05e0d6f`）。

#### 5.5.2 TimescaleDB 迁移规则

**必须先启用压缩再创建压缩策略**：

```sql
-- 正确顺序：
-- 1. 创建 hypertable
SELECT create_hypertable('table_name', 'time_column');
-- 2. 启用压缩（必须在压缩策略之前）
ALTER TABLE table_name SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'column_name',
    timescaledb.compress_orderby = 'time_column DESC'
);
-- 3. 添加压缩策略
SELECT add_compression_policy('table_name', INTERVAL '7 days');
```

> **历史教训**：2 个 hypertable 因缺少步骤 2 导致 `columnstore not enabled` 错误（commit `dcd4870`）。

#### 5.5.3 分区表外键限制

PostgreSQL **不支持** 对分区表建立外键引用（分区表的主键必须包含分区键，且外键目标必须是唯一约束列）。分区表之间的关联关系通过应用层保证数据一致性。

```sql
-- 错误：devices 是分区表，不支持外键
-- CONSTRAINT fk_device FOREIGN KEY (device_sn) REFERENCES devices(serial_number)

-- 正确：仅保留逻辑引用，应用层校验
device_sn VARCHAR(64) NOT NULL  -- 逻辑引用 devices.serial_number
```

> **历史教训**：device_tasks 因外键引用分区表失败（commit `c0adfa4`）。

#### 5.5.4 INSERT 语句与表 Schema 一致性

迁移文件中的 `INSERT` 语句 **必须** 与同目录下 DDL 定义的实际表结构完全匹配：

- 新增迁移前先确认目标表的实际列定义（查看同目录 DDL 文件或 `\d table_name`）
- 所有 NOT NULL 列必须提供值
- 不能引用不存在的列
- 空字符串 `''` 与 `NULL` 要区分清楚，注意 CHECK 约束
- JSON 字符串不能有尾随逗号（如 `'{"a":1,}'`）

```sql
-- 错误：引用不存在的 name/rpc_methods 列，缺少 version/is_active 列
-- INSERT INTO data_model_definitions (id, name, rpc_methods, ...) VALUES (...);

-- 正确：与 DDL 定义的列完全匹配
INSERT INTO data_model_definitions (id, carrier, tech, scope, version, is_active, ...)
VALUES (gen_random_uuid(), 'cmcc', 'lte', 'product', '1.0', true, ...);
```

> **历史教训**：2 次 INSERT 与 Schema 不匹配导致迁移失败（commits `c2e820c`, `985e469`）。

#### 5.5.5 PostgreSQL CREATE TABLE 内不支持部分唯一约束

PostgreSQL 的 `CREATE TABLE` 内联 `CONSTRAINT ... UNIQUE(...) WHERE ...` 语法 **不被支持**。部分唯一约束（带 WHERE 条件）必须用独立的 `CREATE UNIQUE INDEX` 语句：

```sql
-- 错误：CREATE TABLE 内不支持带 WHERE 的 UNIQUE 约束
CREATE TABLE t (
    name VARCHAR(100),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uniq_name UNIQUE(name) WHERE deleted_at IS NULL  -- 语法错误
);

-- 正确：用独立的 CREATE UNIQUE INDEX
CREATE TABLE t (
    name VARCHAR(100),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX uniq_name ON t(name) WHERE deleted_at IS NULL;
```

> **历史教训**：字典表迁移因此语法错误失败（commit `fce3a65`）。

#### 5.5.6 UUID 格式校验

PostgreSQL 的 UUID 类型严格校验格式（8-4-4-4-12）。种子数据中的 UUID 必须：
- 最后一段固定 12 个十六进制字符
- 使用 `gen_random_uuid()` 或经过校验的硬编码 UUID
- 批量插入前可用正则验证：`grep -P '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{11}\b'` 捕获格式错误

> **历史教训**：菜单表种子数据 UUID 最后一段仅 11 字符导致插入失败（commit `0ba37b0`）。

#### 5.5.7 种子数据与 DDL 迁移去重

DDL 迁移文件和 `seed/` 种子文件中的初始数据会重叠。规则：

- **DDL 迁移文件中的 INSERT**：使用 `ON CONFLICT DO NOTHING` 保证幂等
- **seed/ 文件中的 INSERT**：同样使用 `ON CONFLICT DO NOTHING`
- **禁止** 在 DDL 迁移中放不带 `ON CONFLICT` 的 INSERT，否则与 seed 文件重复执行时会失败

#### 5.5.8 Down 迁移完整性

Down 迁移必须清除 Up 迁移创建的 **所有** 数据库对象：

| Up 创建 | Down 必须删除 |
|---------|-------------|
| 表 | `DROP TABLE IF EXISTS` |
| 函数 | `DROP FUNCTION IF EXISTS` |
| 触发器 | `DROP TRIGGER IF EXISTS` |
| 索引 | `DROP INDEX IF EXISTS` |
| 扩展 | `DROP EXTENSION IF EXISTS`（仅限迁移专属扩展） |
| 类型 | `DROP TYPE IF EXISTS` |

**注意**：共享函数（如 `update_updated_at_column()`）在 `000001` 中创建，后续迁移不应重复创建。如需确保存在，用 `CREATE OR REPLACE FUNCTION`。

#### 5.5.9 TRUNCATE 与外键约束

PostgreSQL 规定：**被 FK 引用的表不能单独 TRUNCATE**，即使引用方为空、即使 FK 声明了 `ON DELETE CASCADE`。两条独立 `TRUNCATE` 语句会被拒绝（`SQLSTATE 0A000: cannot truncate a table referenced in a foreign key constraint`）。

种子数据 / 测试夹具里如果要清空被 FK 引用的表，必须**两选一**：

```sql
-- 方案 A（推荐）：同一条语句 truncate 所有相关表
TRUNCATE alarm_library_i18n, alarm_libraries RESTART IDENTITY;

-- 方案 B：CASCADE 递归 truncate
TRUNCATE alarm_libraries CASCADE;
```

**进一步避免 TRUNCATE 的设计建议**：种子数据用 `INSERT ... ON CONFLICT (key) DO UPDATE SET ...` 实现 upsert，不再需要先 truncate。这种 idempotent 写法既不破坏外键关系，又不会清掉运行时由用户产生的关联数据。

> **历史教训**：`seed/000028_alarm_library_import.sql` 用两条独立 TRUNCATE 清空 `alarm_library_i18n` 和 `alarm_libraries`，在已有数据的环境下 migrate-seed 直接 exit 1 → acs/app/worker 因 depends_on 全部起不来（commit `7afc3241`）。

#### 5.5.10 迁移文件自查清单

每次新增迁移文件后，按此清单自查：

- [ ] 版本号 = 现有最大版本号 + 1（无跳跃、无重复）
- [ ] 文件包含 `-- +goose Up` 和 `-- +goose Down` 两个段落
- [ ] 所有 `DO $$` / `CREATE FUNCTION` / `CREATE OR REPLACE FUNCTION` 被 `StatementBegin/End` 包裹
- [ ] INSERT 语句的列名与目标表 DDL 完全匹配
- [ ] INSERT 语句包含 `ON CONFLICT DO NOTHING`（种子数据类）
- [ ] TimescaleDB hypertable 压缩策略前已启用 `timescaledb.compress`
- [ ] 无分区表间的外键约束
- [ ] `CREATE TABLE` 内无带 WHERE 的 UNIQUE 约束
- [ ] UUID 格式正确（8-4-4-4-12）
- [ ] Down 段删除 Up 段创建的所有对象（表、函数、索引、触发器）
- [ ] 无与 `seed/` 目录的重复 INSERT（或均使用 `ON CONFLICT`）
- [ ] JSON 字符串无尾随逗号
- [ ] **TRUNCATE 被 FK 引用的表时**：与所有引用方写在同一条语句，或加 `CASCADE`
- [ ] **改了已被 seed 引用的表**（增/删/改字段或约束）：确认所有既有 seed 仍能在新结构上跑通——新增列可空或带 `DEFAULT`、收紧约束前已 backfill、删列/改名走两阶段（详见 §5.5.11）

#### 5.5.11 Schema 演进不破坏既有种子数据（CRITICAL）

**根因**：`migrate-seed` 用独立版本表 `goose_db_version_seed` 且 `depends_on migrate-schema`，执行模型是「**先全部 schema，再全部 seed**」。因此**每个旧 seed 文件总在「被后续所有 DDL 改过的最终表结构」上执行**——seed 按写入时的结构写，却跑在未来的结构上。改 schema 时只要破坏了任一既有 seed 的前提，全新库 / 存量库重跑 `migrate-seed` 就失败。本项目反复踩此坑。

**五种破坏场景与安全做法**：

| schema 变更 | 旧 seed 为何失败 | 安全做法 |
|------------|----------------|---------|
| 删列 | `INSERT (...,删掉的列,...)` 引用不存在列 | 两阶段：先废弃（保留列）→ 确认无 seed/代码引用 → 下个 release 再 `DROP` |
| 加无默认 `NOT NULL` 列 | 旧 seed 不给该列 → NOT NULL 违反 | 直接带 `DEFAULT`；或三步：加可空列 → `UPDATE` backfill（含 seed 行）→ 再加 `NOT NULL` |
| 收紧 `CHECK`/`UNIQUE`/`FK` | 旧 seed 的值不再满足新约束 | 同一迁移内**先 `UPDATE` 修存量数据**满足新约束，再加约束 |
| 改列名 | seed 引用旧列名 | 先加新列双写 → 迁移 seed/代码 → 再删旧列（跨 release） |
| 改类型 | seed 字面值无法隐式转换 | `ALTER ... TYPE ... USING <转换>`，确认 seed 值可转 |

**五条铁律**：
1. **新增列一律可空或带 `DEFAULT`**，绝不裸加「无默认 `NOT NULL`」列。
2. **收紧约束前，同一迁移内先 backfill 修存量数据**（含 seed 插入的行），再加约束。
3. **删列 / 改名走两阶段，跨 release 完成**——同 release 内删列极易打爆旧 seed。
4. **已 applied 的旧 seed 文件内容不回头改**（改了会触发 `check-schema-drift.sh` 的 checksum 漂移）——结构兼容责任放在**新的 schema 迁移**里（backfill / 改约束），而非回头改旧 seed。
5. **seed 侧防御**：显式列名（禁隐式全列 `VALUES`）+ `ON CONFLICT` + **只插稳定核心列**（易变列交给 `DEFAULT`），缩小 seed 对结构的耦合面。

**治本（治标铁律之外，强烈建议补 CI 门禁）**：现有 CI 对迁移只做静态 lint（撞号 + goose 标记），**不真跑迁移**。应在 CI 起一个全新 postgres 真跑 `make migrate-schema-up && make migrate-seed-up`，并补一个「存量库重跑」场景（先跑到上个 release + 灌数据，再跑新迁移）——让「旧 seed × 新结构」不兼容在 PR 就 fail，而非漏到部署。

> **历史教训**：`f9565e1a` —— `000004` pm_tasks 的 `dimension` CHECK 漏了 `'network'`，存量库重跑 migrate 时既有数据 / seed 不满足 CHECK 直接失败（补救是在迁移里放宽 CHECK）。

### 5.6 全局 API 密钥约定（.api-key）

**目的**：容器内任何工具/脚本调用 OMC HTTP API 时零配置可用,免去 `OMCCTL_API_KEY` env 配置负担。

**文件契约**：

| 属性 | 值 |
|------|------|
| 路径 | `/var/lib/omcgo/secrets/.api-key` |
| 格式 | 单行纯文本(`omk_` + 32 hex,共 36 字符),无尾换行 |
| 权限 | `0640`(owner rw,group r) |
| 签发者 | `omcgo-app` 启动期幂等签发(`admin.EnsureInternalAPIKey`) |
| 关联实体 | DB 中 `system` 用户(UUID `00000000-...001`)的 `omc-internal` API key 行,scopes=['*'] |
| 共享卷 | docker-compose named volume `omcgo-secrets`,app(rw) + worker(ro) |

**签发流程**(app 启动期):

1. 读 `.api-key` 文件 → 存在则 `APIKeyService.Validate` → 有效则直接 return
2. 文件缺失 / 失效 → 吊销 system 用户名下所有 `omc-internal` 旧 key → `APIKeyService.Create` 签发新 key → 原子写文件(tmp + rename)
3. 全程失败不阻塞 app 启动(`logger.Warn` 而非 fail-start)

**消费规则**:

- **omcctl**:`--api-key` flag > `OMCCTL_API_KEY` env > 文件 > 报错
- **bash 脚本**:统一写法
  ```bash
  API_KEY="${OMCCTL_API_KEY:-$(cat /var/lib/omcgo/secrets/.api-key 2>/dev/null || true)}"
  [ -z "$API_KEY" ] && { echo "no api key"; exit 1; }
  curl -H "X-API-Key: $API_KEY" ...
  ```
- **future tools**:复用 `admin.LoadInternalAPIKey(path)`(Go)或上面的 bash 5 行

**禁止**:

- 永不入 git(`.gitignore` 已加 `.api-key` / `**/.api-key`)
- 永不打日志明文(只允许 `key_prefix` = "omk_" + 前 4 hex)
- 永不入迁移文件(明文不放 SQL,seed/000204 只建 user 与 role 绑定)

**轮换**:

- **被动**:重启 app 自动检测文件失效并轮换
- **主动**(未来):`omcctl auth rotate` 调用 app 端 HTTP 显式轮换

**K8s 部署**(未来):named volume 改用 K8s Secret + projected volume,路径 `/var/lib/omcgo/secrets/.api-key` 保持不变,代码无需调整。

**测试**:`internal/admin/internal_apikey_test.go` 单测 atomic-write 与 load 行为;DB 相关签发逻辑由 docker-compose E2E 验证。

**相关代码**:

- `internal/admin/internal_apikey.go` — `EnsureInternalAPIKey` / `LoadInternalAPIKey` / `writeAPIKeyAtomic`
- `cmd/app/provider/admin.go` — 启动期接入点
- `cmd/omcctl/main.go` — fallback 三级查找
- `migrations/seed/000204_system_internal_user.sql` — `system` 用户与 `admin` 角色绑定
- `deployments/docker/docker-compose.yml` — `omcgo-secrets` named volume

---

## 6. Git 工作流

### 分支策略

| 分支 | 用途 | 来源 |
|------|------|------|
| `main` | 稳定版本，始终可部署 | — |
| `feature/{描述}` | 新功能开发 | 从 `main` 创建 |
| `fix/{描述}` | Bug 修复 | 从 `main` 创建 |
| `refactor/{描述}` | 重构 | 从 `main` 创建 |
| `release/x.x` | 发布准备 | 从 `main` 创建 |

### Commit Message 格式

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <简短描述>

[可选正文]

[可选脚注]
```

**type**：
- `feat` — 新功能
- `fix` — Bug 修复
- `refactor` — 重构（不改变行为）
- `docs` — 文档变更
- `test` — 测试
- `chore` — 构建/工具/CI 变更
- `perf` — 性能优化

**scope**（对应功能域或模块）：
`acs`, `config`, `parammodel`, `product`, `pm`, `alarm`, `mr`, `device`, `admin`, `topology`, `software`, `backup`, `dashboard`, `ops`, `report`, `mml`, `filemanager`, `syslog`, `license`, `nedirect`, `northbound`, `provision`, `interop`, `carrier`, `components`, `api`, `deploy`

**示例**：
```
feat(acs): 实现 TR069 Inform 解析与会话状态机
fix(datamodel): 修复三级回退解析在 oui 为空时的 panic
refactor(carrier): 提取公共参数映射逻辑到 Carrier 接口
docs: 更新 CLAUDE.md 添加数据模型缓存策略说明
test(pm): 添加 KPI 计算引擎的 table-driven 测试
chore(deploy): 添加 ACS 引擎的 Dockerfile 和 K8s deployment
```

### PR 流程

1. 从 `main` 创建 feature/fix 分支
2. 开发完成后提交 PR 到 `main`
3. PR 描述包含：变更说明、关联功能域编号（如 F01、F02）、测试说明
4. Code review 通过后 squash merge 到 `main`

### 版本号

遵循 [SemVer](https://semver.org/)：`MAJOR.MINOR.PATCH`

---

## 7. 关键文档索引

| 文档 | 路径 | 说明 |
|------|------|------|
| 系统总览 | `doc/architecture/system-overview.md` | 系统定位、设备类型、运营商差异 |
| 后端架构设计 | `doc/architecture/backend-design.md` | 完整技术方案：技术栈、目录结构、数据库 Schema、API 设计、部署拓扑 |
| 框架选型分析 | `doc/architecture/framework-comparison.md` | 当前方案 vs go-zero 的深度对比（**结论：不用 go-zero**） |
| 接口拓扑 | `doc/architecture/interface-topology.md` | 南向/北向/直连接口协议栈 |
| 功能索引 | `doc/功能索引.md` | 10 域 43 项子功能完整列表 + 运营商交叉矩阵 |
| 功能域详情 | `doc/features/01~10-*.md` | 每个功能域的子功能说明 |
| 规范目录 | `doc/specs-inventory/document-catalog.md` | 55 份运营商技术规范索引 |
| 运营商对比 | `doc/specs-inventory/carrier-comparison.md` | 三家运营商规范覆盖差异 |
| 运营商规范原件 | `规范/移动/`, `规范/电信/`, `规范/联通/` | docx/xlsx 原始规范文件 |

---

## 8. 实施路线图

| 阶段 | 目标 | 核心模块 |
|------|------|---------|
| **一：基础建设** | ACS 引擎能接收 Inform 并注册设备 | 项目脚手架, components 层, pkg/tr069, acs 基础, device 注册 |
| **二：核心功能** | 完整设备管理和自动开站流程 | acs/rpc 全量方法, task 统一队列, connreq, parammodel/product 字典, carrier(cmcc), provision |
| **三：数据管线** | PM/告警/MR 数据全链路 | pm, kpi, alarm, mr, carrier(ctcc/cucc) |
| **四：北向与规模化** | OSS 对接、10 万级验证、生产加固 | northbound, omcr 完整功能, 负载测试, TLS/认证/监控 |

---

## 9. 常用命令（待项目初始化后补充）

```bash
# 构建
make build              # 编译所有二进制
make build-acs          # 仅编译 ACS 引擎
make build-app          # 仅编译主应用

# 测试
make test               # 运行单元测试
make test-integration   # 运行集成测试
make lint               # 代码检查

# 数据库
make migrate-up         # 执行迁移
make migrate-down       # 回滚迁移

# Docker
make docker-build       # 构建所有镜像
make docker-up          # 启动本地开发环境（docker-compose）

# 开发工具
make generate           # 生成代码（proto, swagger, mock）
make swagger            # 生成 API 文档
```

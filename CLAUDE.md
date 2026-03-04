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
| F02 | 数据模型与配置 | TR069 参数树定义，三级回退解析，配置模板 |
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

**不使用微服务��架，不使用 go-zero。**

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
| 日志 | `go.uber.org/zap` | ���构化高性能日志 |
| 指标 | `github.com/prometheus/client_golang` | Prometheus 指标暴露 |
| 链路追踪 | `go.opentelemetry.io/otel` | 分布式链路追踪 |
| DB 驱动 | `github.com/jackc/pgx/v5` | PostgreSQL 高性能驱动 |
| 连接池 | `github.com/jackc/pgx/v5/pgxpool` | 数据库连接池 |
| Redis | `github.com/redis/go-redis/v9` | 缓存、会话、命令队列 |
| 消息队列 | `github.com/nats-io/nats.go` | NATS JetStream |
| 对象存储 | `github.com/minio/minio-go/v7` | MinIO/S3 |
| 数据库迁移 | `github.com/golang-migrate/migrate/v4` | Schema 版本管理 |
| SQL 构建 | `github.com/Masterminds/squirrel` | 动态 SQL 构建（不使用 ORM） |
| 参数验证 | `github.com/go-playground/validator/v10` | 结构体校验 |
| UUID | `github.com/google/uuid` | UUID 生成 |
| 定时任务 | `github.com/robfig/cron/v3` | PM 采集、聚合等周期任务 |
| 限流 | `golang.org/x/time/rate` | 设备级/全局限流 |
| 测试 | `github.com/stretchr/testify` | 断言�� Mock |
| API 文档 | `github.com/swaggo/swag` | Swagger 自动生成 |

### 数据存储

| 数据类型 | 存储 |
|---------|------|
| 设备、配置、拓扑、用户 | PostgreSQL 16（JSONB 支持灵活 Schema） |
| PM 计数器 & KPI 时序 | TimescaleDB（PostgreSQL 扩展） |
| 历史告警 | TimescaleDB 超表 |
| TR069 会话状态 | Redis 7 Cluster（TTL 自动过期） |
| 设备命令队列 | Redis Sorted Set |
| 数据模型缓存 | Redis + 内存 L1 |
| PM/MR/固件/备份文件 | MinIO（S3 兼容） |
| 进程内事件 | Go channel |
| 持久化消息 | NATS JetStream |

---

## 4. 项目目录结构

```
omcgo/
├── cmd/                            # 入口
│   ├── acs/main.go                 # TR069 ACS 引擎
│   ├── app/main.go                 # 主应用（F02-F10）
│   ├── worker/main.go              # 后台工作进程
│   ├── migrate/main.go             # 数据库迁移
│   └── omcctl/main.go              # CLI 管理工具
│
├── internal/                       # 私有代码（按功能域组织）
│   ├── acs/                        # F01: TR069 ACS 引擎
│   │   ├── server.go               #   HTTP 服务器
│   │   ├── handler.go              #   请求处理
│   │   ├── session.go              #   会话状态机
│   │   ├── soap/                   #   SOAP 编解码
│   │   ├── rpc/                    #   RPC 方法（Get/Set/Download/Upload/Reboot...）
│   │   ├── connreq/                #   Connection Request
│   │   ├── cmdqueue/               #   Redis 命令队列
│   │   └── auth/                   #   CPE 认证（Digest/Basic）
│   │
│   ├── config/                     # F02: 数据模型与配置管理
│   │   ├── datamodel/              #   数据模型注册表、三级回退解析、缓存
│   │   ├── template/               #   配置模板
│   │   ├── audit/                  #   配置审计
│   │   └── backup/                 #   配置备份
│   │
│   ├── pm/                         # F03: 性能管理
│   │   ├── collector/              #   PM 文件采集与 XML 解析
│   │   ├── counter/                #   计数器存储
│   │   ├── kpi/                    #   KPI 计算引擎
│   │   └── aggregation/            #   时间维度聚合
│   │
│   ├── alarm/                      # F04: 告警管理
│   ├── mr/                         # F05: 测量报告
│   ├── omcr/                       # F06: OMC-R 核心（拓扑/设备/固件/用户管理）
│   ├── nedirect/                   # F07: 网元直连（移动专有）
│   ├── northbound/                 # F08: 北向/OSS 接口
│   ├── provision/                  # F09: 自动开站
│   ├── interop/                    # F10: 互操作测试
│   │
│   ├── carrier/                    # 运营商抽象层
│   │   ├── carrier.go              #   Carrier 接口定义
│   │   ├── registry.go             #   CarrierRegistry
│   │   ├── cmcc/                   #   中国移动适配器
│   │   ├── ctcc/                   #   中国电信适配器
│   │   └── cucc/                   #   中国联通适配器
│   │
│   ├── common/                     # 公共类型
│   │   ├── model/                  #   领域模型（Device, Alarm, Parameter, PMCounter, KPI）
│   │   ├── event/                  #   EventBus（channel + NATS 双实现）
│   │   ├── errors/                 #   错误类型
│   │   └── middleware/             #   HTTP 中间件（认证/日志/指标/恢复）
│   │
│   └── infra/                      # 基础设施适配器
│       ├── db/                     #   PostgreSQL/TimescaleDB 连接
│       ├── cache/                  #   Redis 连接
│       ├── mq/                     #   NATS 连接
│       └── storage/                #   MinIO 连接
│
├── pkg/                            # 可复用公共库
│   ├── tr069/                      #   TR069 类型、事件码、CWMP 错误码
│   ├── soap/                       #   通�� SOAP 工具
│   └── xmlutil/                    #   XML 辅助工具
│
├── api/                            # API 定义
│   ├── openapi/                    #   OpenAPI/Swagger 定义
│   └── proto/                      #   gRPC Proto 定义
│
├── migrations/                     # 数据库迁移文件（golang-migrate 格式）
├── configs/                        # 配置文件模板（acs.yaml, app.yaml, worker.yaml）
├── datamodels/                     # TR069 数据模型种子数据
│   ├── seed/                       #   初次部署种子（carrier_defaults/ + product_models/）
│   └── templates/                  #   JSON Schema 模板
├── deployments/                    # 部署清单
│   ├── docker/                     #   Dockerfile + docker-compose
│   └── k8s/                        #   Kubernetes manifests
├── doc/                            # 项目文档（已有）
├── scripts/                        # 构建/部署/测试脚本
├── test/                           # 集成/E2E 测试
│   ├── integration/
│   ├── e2e/
│   └── fixtures/                   #   测试数据（SOAP 报文、PM 文件示例）
│
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

// 数据模型 Scope（解析优先级从高到低��
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
  - `CommandQueue` — 设备命令队列
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
acs:cmdq:{device_serial}             — 命令队列 Sorted Set
acs:heartbeat:{device_serial}        — 心跳时间戳（TTL = 2×inform_interval）
acs:connreq:pending:{device_serial}  — Connection Request 去重（TTL 30 秒）
datamodel:product:{carrier}:{tech}:{oui}:{product_class} — 数据模型缓存
datamodel:oui:{carrier}:{tech}:{oui}
datamodel:default:{carrier}:{tech}
datamodel:resolve:{carrier}:{tech}:{oui}:{product_class} — 解析结果缓存（TTL 1 小时）
datamodel:cache_version              — 缓存版本号（跨实例协调）
alarm:active:{device_serial}         — 活跃告警 Hash
ratelimit:inform:{device_serial}     — 限流计数器
```

**流量控制**：
- 每设备限流器（`rate.Limiter`）防止 Inform 洪泛
- 全局准入控制器（`AdmissionController`）限制并发会话数
- PM/MR 文件处理使用 WorkerPool 控制并发

### 5.3 数据模型专项规范

**三级回退解析**（查找设备对应的数据模型定义）：
1. **product 级**：carrier + tech + oui + product_class（最精确）
2. **oui 级**：carrier + tech + oui（厂商默认）
3. **carrier_default 级**：carrier + tech（运营商默认）

**三级缓存**：
1. 内存 L1（`sync.Map`，进程内）
2. Redis L2（`datamodel:*` keys，TTL 24 小时）
3. PostgreSQL（`data_model_definitions` 表）

**生命周期**：`draft` → `active` → `deprecated`
- 同分类（carrier + tech + oui + product_class + scope）仅一个 `active` 模型
- 激活新模型自动废弃旧模型
- 变更时自增 `datamodel:cache_version` 通知所有 ACS 实例

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
`acs`, `config`, `datamodel`, `pm`, `alarm`, `mr`, `omcr`, `nedirect`, `northbound`, `provision`, `interop`, `carrier`, `infra`, `api`, `deploy`

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

遵��� [SemVer](https://semver.org/)：`MAJOR.MINOR.PATCH`

---

## 7. 关键文档索引

| 文档 | 路径 | 说明 |
|------|------|------|
| 系统总览 | `doc/architecture/system-overview.md` | 系统定位、设备类型、运营商差异 |
| 后端架构设计 | `doc/architecture/backend-design.md` | 完整技术方案：技术栈、目录结构、数据库 Schema、API 设计、部署拓扑 |
| 框架选型分析 | `doc/architecture/framework-comparison.md` | 当前方案 vs go-zero 的深度对比（**���论：不用 go-zero**） |
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
| **一：基础建设** | ACS 引擎能接收 Inform 并注册设备 | 项目脚手架, infra 层, pkg/tr069, acs 基础, device 注册 |
| **二：核心功能** | 完整设备管理和自动开站流程 | acs/rpc 全量方法, cmdqueue, connreq, datamodel, carrier(cmcc), provision |
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

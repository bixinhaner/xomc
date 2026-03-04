# OMC Go 后端架构选型与设计

[← 返回目录](../README.md)

---

## 设计约束

| 项目 | 要求 |
|------|------|
| 开发语言 | Go 1.22+ |
| 初始规模 | 10 万基站管理 |
| 目标规模 | 预留 100 万级基站管理 |
| 运营商 | 中国移动、中国电信、中国联通 |
| 制式 | LTE (4G)、5G NR (SA) |
| 功能域 | 10 个功能域，43 项子功能（详见 [功能索引](../功能索引.md)） |

---

## 一、总体架构模式

### 选型：领域驱动模块化单体 + ACS 引擎独立部署

不采用纯微服务，也不采用纯单体，而是**模块化单体**组织 10 个功能域（F01-F10），具备服务拆分就绪能力。

**理由**：

1. 10 万基站规模下，单个 Go 进程完全可以承载负载
2. 系统有清晰的领域边界（10 个功能域），但跨域交互密集（F01 为 F02/F03/F04/F05/F09 提供数据通道）
3. 过早微服务化引入不必要的运维复杂度（服务网格、分布式事务、Saga 模式）
4. **唯一例外**：TR069 ACS 引擎（F01）从第一天起就作为独立进程部署，因为它的并发模型和规模需求与其他模块截然不同

### 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│                     外部接口层                                │
│    [F08 北向 REST/gRPC]          [Web Dashboard API]         │
├─────────────────────────────────────────────────────────────┤
│                     应用服务层                                │
│                                                              │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐  │
│  │ 配置管理 │ │ 性能管理 │ │ 告警管理 │ │ MR管理  │ │ OMC-R核心 │  │
│  │ (F02)  │ │ (F03)  │ │ (F04)  │ │ (F05)  │ │  (F06)   │  │
│  └────────┘ └────────┘ └────────┘ └────────┘ └──────────┘  │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                     │
│  │ 自动开站  │ │ OSS网关   │ │ 网元直连  │                     │
│  │  (F09)   │ │  (F08)   │ │  (F07)   │                     │
│  └──────────┘ └──────────┘ └──────────┘                     │
├─────────────────────────────────────────────────────────────┤
│                     领域核心层                                │
│                                                              │
│  ┌───────────────────┐  ┌──────────────────────────────┐    │
│  │  运营商抽象层       │  │  数据模型注册表               │    │
│  │  (Carrier Layer)  │  │  (TR069 Parameter Trees)     │    │
│  └───────────────────┘  └──────────────────────────────┘    │
│                                                              │
│  ┌───────────────────┐  ┌──────────────────────────────┐    │
│  │  设备注册表         │  │  事件总线 (in-process)        │    │
│  │  & 会话管理器       │  │  EventBus                    │    │
│  └───────────────────┘  └──────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                     基础设施层                                │
│                                                              │
│  ┌──────────┐ ┌───────────┐ ┌──────┐ ┌──────┐ ┌──────┐    │
│  │PostgreSQL│ │TimescaleDB│ │Redis │ │ NATS │ │MinIO │    │
│  │ (配置)   │ │ (PM/KPI) │ │(缓存) │ │(消息) │ │(文件) │    │
│  └──────────┘ └───────────┘ └──────┘ └──────┘ └──────┘    │
├─────────────────────────────────────────────────────────────┤
│           TR069 ACS 引擎 (F01) — 独立进程                    │
│           无状态、水平可扩展                                   │
│           处理 10万~100万 SOAP/HTTP 会话                      │
├─────────────────────────────────────────────────────────────┤
│           基站设备 (CPE) — 10万 ~ 100万                      │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、服务/模块分解

### 模块 1：`acs` — TR069 ACS 引擎（F01）

处理所有 TR069/CWMP 协议通信。独立进程部署。

| 子模块 | 职责 |
|--------|------|
| `acs/server` | HTTP/HTTPS 服务器，接受 CPE 连接 |
| `acs/handler` | TR069 HTTP 请求处理 |
| `acs/session` | TR069 会话状态机（每设备会话生命周期） |
| `acs/soap` | SOAP/XML 编解码、信封处理 |
| `acs/rpc` | RPC 方法实现（Get/SetParameterValues, AddObject, Download, Upload, Reboot 等） |
| `acs/connreq` | Connection Request 发起（ACS 触发 CPE 连接） |
| `acs/cmdqueue` | 每设备命令队列（Redis-backed，等待 CPE 下次连接时执行） |
| `acs/auth` | CPE 认证（HTTP Digest/Basic） |

### 模块 2：`config` — 数据模型与配置管理（F02）

| 子模块 | 职责 |
|--------|------|
| `config/datamodel` | TR069 数据模型注册表（按运营商/制式/版本加载参数树） |
| `config/template` | 配置模板管理 |
| `config/audit` | 配置审计（实际值 vs 期望值对比） |
| `config/backup` | 配置备份与恢复 |
| `config/batch` | 批量配置操作 |

### 模块 3：`pm` — 性能管理（F03）

| 子模块 | 职责 |
|--------|------|
| `pm/collector` | PM 文件采集与 XML 解析 |
| `pm/counter` | 原始计数器存储与查询 |
| `pm/kpi` | KPI 计算引擎（按运营商/制式定义公式） |
| `pm/aggregation` | 时间维度聚合（15min → 1h → 24h） |
| `pm/export` | PM 数据导出（北向） |

### 模块 4：`alarm` — 告警管理（F04）

| 子模块 | 职责 |
|--------|------|
| `alarm/receiver` | 告警事件接收（来自 TR069 Inform ALARM 事件） |
| `alarm/engine` | 关联分析、去重、抑制 |
| `alarm/lifecycle` | 告警状态机（活跃 → 确认 → 清除） |
| `alarm/forward` | 告警转发至 OSS 北向 |
| `alarm/store` | 活跃告警表 + 历史告警归档 |

### 模块 5：`mr` — 测量报告管理（F05）

| 子模块 | 职责 |
|--------|------|
| `mr/collector` | MR 文件采集（通过 TR069 Upload 触发） |
| `mr/parser` | XML 解析器（MRO/MRS/MRE，运营商差异化格式） |
| `mr/store` | 解析后 MR 数据存储 |

### 模块 6：`omcr` — OMC-R 核心功能（F06）

| 子模块 | 职责 |
|--------|------|
| `omcr/topology` | 网络拓扑管理（设备发现、注册、分组、状态监控） |
| `omcr/device` | 设备生命周期管理 |
| `omcr/software` | 固件/软件版本管理与升级编排 |
| `omcr/log` | 设备日志采集 |
| `omcr/admin` | 用户管理、RBAC、审计日志 |

### 模块 7：`nedirect` — 网元直连接口（F07，移动专有）

### 模块 8：`northbound` — 北向/OSS 接口（F08）

| 子模块 | 职责 |
|--------|------|
| `northbound/api` | RESTful API 端点 |
| `northbound/push` | 主动数据推送（文件或 API） |
| `northbound/sync` | 全量/增量数据同步 |

### 模块 9：`provision` — 自动开站（F09）

| 子模块 | 职责 |
|--------|------|
| `provision/workflow` | 开站状态机（bootstrap→识别→模板匹配→配置下发→验证→激活） |
| `provision/template` | 开站模板管理（4G/5G 分运营商） |
| `provision/batch` | 批量开站 |

### 模块 10：`interop` — 互操作测试（F10）

### 公共模块

| 模块 | 职责 |
|------|------|
| `carrier` | 运营商抽象层（接口 + 三家运营商适配器实现） |
| `common/model` | 领域模型（Device, Alarm, Parameter, PMCounter, KPI） |
| `common/event` | 事件总线（channel + NATS 双实现） |
| `common/middleware` | HTTP 中间件（认证、日志、指标、恢复） |
| `infra/db` | PostgreSQL/TimescaleDB 连接管理 |
| `infra/cache` | Redis 连接管理 |
| `infra/mq` | NATS 连接管理 |
| `infra/storage` | MinIO 对象存储管理 |

---

## 三、技术栈选型

### 核心框架与库

| 组件 | 选型 | 说明 |
|------|------|------|
| HTTP 服务器 (ACS) | `net/http` (stdlib) | Go 标准库 HTTP 原生支持大规模并发，TR069 无需 Web 框架 |
| REST API 框架 | `github.com/gin-gonic/gin` | 北向 API 和管理面板，高性能、生态成熟 |
| gRPC | `google.golang.org/grpc` | 模块间通信（ACS ↔ App 进程） |
| XML/SOAP | `encoding/xml` + `github.com/beevik/etree` | 标准库用于结构体编解码，etree 用于动态参数树遍历 |
| 配置管理 | `github.com/spf13/viper` | 多格式配置文件、环境变量、热重载 |
| CLI | `github.com/spf13/cobra` | 命令行管理工具 |
| 日志 | `go.uber.org/zap` | 结构化高性能日志 |
| 指标 | `github.com/prometheus/client_golang` | Prometheus 指标暴露 |
| 链路追踪 | `go.opentelemetry.io/otel` | 分布式链路追踪 |
| 数据库驱动 | `github.com/jackc/pgx/v5` | PostgreSQL 高性能驱动 |
| 连接池 | `github.com/jackc/pgx/v5/pgxpool` | 数据库连接池 |
| Redis | `github.com/redis/go-redis/v9` | Redis 客户端 |
| 对象存储 | `github.com/minio/minio-go/v7` | MinIO/S3 客户端 |
| 消息队列 | `github.com/nats-io/nats.go` | NATS JetStream 客户端 |
| 数据库迁移 | `github.com/golang-migrate/migrate/v4` | Schema 版本管理 |
| SQL 构建器 | `github.com/Masterminds/squirrel` | 动态 SQL 构建 |
| 参数验证 | `github.com/go-playground/validator/v10` | 结构体校验 |
| UUID | `github.com/google/uuid` | UUID 生成 |
| 定时任务 | `github.com/robfig/cron/v3` | PM 采集、聚合等周期任务 |
| 限流 | `golang.org/x/time/rate` | 设备级/全局限流 |
| 测试 | `github.com/stretchr/testify` | 断言与 Mock |
| API 文档 | `github.com/swaggo/swag` | Swagger 自动生成 |

### 数据存储选型

| 数据类型 | 存储 | 说明 |
|---------|------|------|
| 设备注册、配置、拓扑、用户 | **PostgreSQL 16** | ACID 事务，JSONB 支持灵活 Schema |
| PM 计数器 & KPI 时序数据 | **TimescaleDB** | PostgreSQL 扩展，自动分区、连续聚合、同驱动 |
| 活跃告警 | **PostgreSQL** | 中等数据量，需要事务和设备关联查询 |
| 历史告警 | **TimescaleDB** 超表 | 时间分区，高效范围查询 |
| MR 解析数据 | **TimescaleDB** / **ClickHouse** | 100 万规模时 ClickHouse 分析查询更优 |
| TR069 会话状态 | **Redis 7 Cluster** | 亚毫秒访问，TTL 自动过期 |
| 设备命令队列 | **Redis** Sorted Set | 按优先级排序的待执行命令 |
| 数据模型缓存 | **Redis** | 避免每次请求重新解析数据模型定义 |
| PM/MR 原始文件 | **MinIO** (S3 兼容) | 对象存储，适合大文件 |
| 固件/配置备份文件 | **MinIO** | 文件版本管理 |

### 消息队列选型

| 场景 | 选型 | 说明 |
|------|------|------|
| 进程内事件总线 | Go channel | 模块间低延迟通信 |
| 持久化消息（PM/MR/告警管线） | **NATS JetStream** | Go 原生、轻量、百万级吞吐，运维成本低于 Kafka |
| 100 万规模备选 | **Apache Kafka** | 若 NATS 吞吐不足或需要严格分区顺序 |

---

## 四、TR069 ACS 引擎设计

### 4.1 连接模型

```
                     ┌───────────────────┐
                     │    负载均衡器       │
                     │   (HAProxy/Nginx)  │
                     │  一致性哈希(设备SN)  │
                     └────────┬──────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
     ┌────────┴──────┐ ┌─────┴───────┐ ┌─────┴───────┐
     │ ACS 实例 1     │ │ ACS 实例 2   │ │ ACS 实例 N   │
     │ (goroutine/会话)│ │ (goroutine/会话)│ │ (goroutine/会话)│
     └───────────────┘ └─────────────┘ └─────────────┘
              │               │               │
              └───────────────┼───────────────┘
                              │
                     ┌────────┴──────────┐
                     │   Redis Cluster    │
                     │  会话状态 + 命令队列  │
                     └───────────────────┘
```

### 4.2 TR069 会话状态机

每个 TR069 会话由一个 goroutine 管理：

```
状态流转:
  IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE

CPE 发起流程（通过 Inform）:
  1. CPE 发送 HTTP POST（SOAP Inform）
  2. ACS goroutine 解析 Inform，提取事件（BOOTSTRAP/PERIODIC/VALUE_CHANGE/ALARM）
  3. ACS 将事件发布到事件总线（告警→告警模块，开站→开站模块等）
  4. ACS 从 Redis 命令队列检查该设备的待执行操作
  5. 若有命令：在 HTTP 响应中发送下一条 RPC（如 SetParameterValues）
  6. CPE 执行 RPC 后在下次 HTTP POST 中返回结果
  7. 循环直到命令队列清空
  8. ACS 发送空 HTTP 响应（204）结束会话

ACS 发起流程（通过 Connection Request）:
  1. 应用模块向 Redis 写入命令
  2. ConnReq 模块向 CPE 的 Connection Request URL 发送 HTTP GET
  3. CPE 以 Inform (CONNECTION REQUEST 事件) 连接 ACS
  4. 进入标准会话流程
```

### 4.3 SOAP/XML 处理策略

```go
// 发送方向（ACS → CPE）：使用预编译 text/template
// 避免 encoding/xml 反射开销，对已知结构使用模板

// 接收方向（CPE → ACS）：使用 encoding/xml Decoder 流式解析
// 避免将整个 XML 文档加载到内存
func parseInform(r io.Reader) (*InformMessage, error) {
    decoder := xml.NewDecoder(r)
    // 流式解析，只提取需要的字段
}

// 参数值列表（可能很大）：使用 etree 动态遍历
// 当结构因运营商而异时
```

### 4.4 容量计算

```
10 万基站（Inform 周期 = 300s）:
  ┌─────────────────────────────────────────────────────┐
  │ Inform 速率:  100,000 / 300 = 333 会话/秒            │
  │ 每会话:       3-5 次 HTTP 往返，500ms-2s 持续时间      │
  │ 并发会话:     333 × 1.5s = ~500 并发                  │
  │ 单会话内存:   ~50KB（goroutine 栈 + 缓冲区）           │
  │ 总内存:       500 × 50KB = 25MB                      │
  │ 结论:         单个 Go 进程轻松处理                      │
  └─────────────────────────────────────────────────────┘

100 万基站:
  ┌─────────────────────────────────────────────────────┐
  │ Inform 速率:  1,000,000 / 300 = 3,333 会话/秒        │
  │ 并发会话:     ~5,000                                  │
  │ 内存:         ~250MB                                  │
  │ 结论:         2-3 个 Go ACS 实例即可                   │
  │ 瓶颈:         从连接转移到 DB 写入和事件处理             │
  └─────────────────────────────────────────────────────┘

突发峰值（如掉电恢复导致 1 万设备同时重连）:
  - 需要连接节流和队列化准入控制
  - Redis-backed 背压机制
```

### 4.5 ACS 实例核心结构

```go
type ACSServer struct {
    httpServer    *http.Server
    sessionStore  SessionStore      // Redis-backed
    commandQueue  CommandQueue      // Redis-backed
    eventBus      EventPublisher    // NATS 或 channel
    deviceAuth    DeviceAuthenticator
    soapCodec     SOAPCodec
    rpcHandlers   map[string]RPCHandler
    rateLimiter   *rate.Limiter
    metrics       *ACSMetrics       // Prometheus
}

type SessionStore interface {
    Create(ctx context.Context, deviceID string, session *Session) error
    Get(ctx context.Context, deviceID string) (*Session, error)
    Update(ctx context.Context, deviceID string, session *Session) error
    Delete(ctx context.Context, deviceID string) error
    SetTTL(ctx context.Context, deviceID string, ttl time.Duration) error
}
```

---

## 五、数据存储策略

### 5.1 PostgreSQL Schema（配置/拓扑/设备）

```sql
-- 设备表
CREATE TABLE devices (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number        VARCHAR(64) NOT NULL UNIQUE,
    oui                  VARCHAR(6) NOT NULL,
    product_class        VARCHAR(64),
    manufacturer         VARCHAR(128),
    carrier              VARCHAR(4) NOT NULL,   -- cmcc, ctcc, cucc
    technology           VARCHAR(3) NOT NULL,   -- lte, nr
    status               VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version     VARCHAR(64),
    ip_address           INET,
    connection_request_url VARCHAR(256),
    last_inform_at       TIMESTAMPTZ,
    last_inform_events   JSONB,
    site_name            VARCHAR(128),
    site_id              VARCHAR(64),
    latitude             DOUBLE PRECISION,
    longitude            DOUBLE PRECISION,
    extension_data       JSONB,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY LIST (carrier);

CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');

CREATE INDEX idx_devices_status ON devices (carrier, technology, status);
CREATE INDEX idx_devices_last_inform ON devices (last_inform_at);

-- 设备参数表
CREATE TABLE device_parameters (
    device_id        UUID NOT NULL REFERENCES devices(id),
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

-- 配置模板表
CREATE TABLE config_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(128) NOT NULL,
    carrier        VARCHAR(4) NOT NULL,
    technology     VARCHAR(3) NOT NULL,
    template_type  VARCHAR(32) NOT NULL,  -- provisioning, batch_config, firmware_upgrade
    parameters     JSONB NOT NULL,
    version        INTEGER NOT NULL DEFAULT 1,
    active         BOOLEAN DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 数据模型定义表
CREATE TABLE data_model_definitions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier        VARCHAR(4) NOT NULL,
    technology     VARCHAR(3) NOT NULL,
    version        VARCHAR(16) NOT NULL,
    parameter_tree JSONB NOT NULL,
    UNIQUE (carrier, technology, version)
);

-- 开站任务表
CREATE TABLE provisioning_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id      UUID NOT NULL REFERENCES devices(id),
    template_id    UUID REFERENCES config_templates(id),
    status         VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 5.2 TimescaleDB Schema（PM/KPI 时序数据）

```sql
-- PM 计数器超表
CREATE TABLE pm_counters (
    time           TIMESTAMPTZ NOT NULL,
    device_id      UUID NOT NULL,
    cell_id        VARCHAR(32),
    counter_group  VARCHAR(32),
    counter_name   VARCHAR(64),
    counter_value  DOUBLE PRECISION,
    granularity    SMALLINT DEFAULT 15  -- 分钟
);

SELECT create_hypertable('pm_counters', 'time',
    chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_pm_device_time ON pm_counters (device_id, time DESC);

-- 压缩策略：7 天后压缩
SELECT add_compression_policy('pm_counters', INTERVAL '7 days');
-- 保留策略：原始数据保留 90 天
SELECT add_retention_policy('pm_counters', INTERVAL '90 days');

-- KPI 值超表
CREATE TABLE kpi_values (
    time        TIMESTAMPTZ NOT NULL,
    device_id   UUID NOT NULL,
    cell_id     VARCHAR(32),
    kpi_name    VARCHAR(64),
    kpi_value   DOUBLE PRECISION,
    carrier     VARCHAR(4),
    technology  VARCHAR(3)
);

SELECT create_hypertable('kpi_values', 'time');

-- 连续聚合（供面板展示）
CREATE MATERIALIZED VIEW pm_counters_hourly
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', time) AS bucket,
       device_id, counter_group, counter_name,
       SUM(counter_value) AS total,
       AVG(counter_value) AS avg_val
FROM pm_counters
GROUP BY bucket, device_id, counter_group, counter_name;

-- 告警历史超表
CREATE TABLE alarms_history (
    time             TIMESTAMPTZ NOT NULL,
    alarm_id         UUID NOT NULL,
    device_id        UUID,
    severity         SMALLINT,  -- 1=紧急 2=重要 3=次要 4=提示
    alarm_type       VARCHAR(64),
    alarm_code       VARCHAR(32),
    description      TEXT,
    status           VARCHAR(16),  -- active, acknowledged, cleared
    acknowledged_at  TIMESTAMPTZ,
    cleared_at       TIMESTAMPTZ
);

SELECT create_hypertable('alarms_history', 'time');
```

### 5.3 Redis 数据结构

```
# TR069 会话状态
acs:session:{device_serial}     → Hash {state, last_rpc, started_at, instance_id}
TTL: 5 分钟（会话超时）

# 设备命令队列
acs:cmdq:{device_serial}        → Sorted Set (score=优先级, member=命令JSON)

# 设备心跳追踪
acs:heartbeat:{device_serial}   → String (最后 Inform 时间戳)
TTL: 2 × inform_interval（用于离线检测）

# 数据模型缓存
datamodel:{carrier}:{tech}:{version} → String (压缩JSON)
TTL: 24 小时

# 活跃告警表
alarm:active:{device_serial}    → Hash {alarm_code → 告警JSON}

# 限流计数器
ratelimit:inform:{device_serial} → Counter with TTL

# Connection Request 去重
acs:connreq:pending:{device_serial} → String ("1")
TTL: 30 秒
```

### 5.4 MinIO 对象存储 Bucket 设计

```
pm-files/
  {carrier}/{date}/{device_serial}/pm_{timestamp}.xml

mr-files/
  {carrier}/{date}/{device_serial}/mr_{type}_{timestamp}.xml

firmware/
  {carrier}/{product_class}/{version}/firmware.bin

config-backup/
  {device_serial}/{timestamp}/config.xml

logs/
  {device_serial}/{date}/log_{timestamp}.gz
```

---

## 六、消息/事件架构

### 6.1 事件分类

```
1. 设备事件（来自 ACS/TR069）:
   device.inform.bootstrap         新设备发现
   device.inform.periodic          心跳
   device.inform.value_change      参数变更
   device.inform.alarm             告警事件
   device.inform.transfer_complete 文件传输完成
   device.connection.lost          心跳超时

2. 命令事件（应用→ACS）:
   command.get_parameters           读取设备参数
   command.set_parameters           写入设备参数
   command.download                 推送固件/配置文件
   command.upload                   请求 PM/MR/日志文件
   command.reboot                   远程重启
   command.factory_reset            恢复出厂

3. 数据事件（内部处理）:
   pm.file.received                PM 文件就绪
   pm.file.parsed                  PM 数据可用
   mr.file.received                MR 文件就绪
   mr.file.parsed                  MR 数据可用
   alarm.raised                    告警产生
   alarm.cleared                   告警清除
   alarm.acknowledged              告警确认

4. 北向事件（到 OSS）:
   oss.alarm.forward               告警转发
   oss.pm.export                   PM 数据导出
   oss.config.snapshot             配置快照
```

### 6.2 EventBus 接口设计

```go
// EventBus 接口 — 支持 channel 和 NATS 两种实现
type EventBus interface {
    Publish(ctx context.Context, subject string, event Event) error
    Subscribe(subject string, handler EventHandler) (Subscription, error)
    QueueSubscribe(subject string, queue string, handler EventHandler) (Subscription, error)
}

// 进程内实现（单进程部署）
type ChannelEventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
}

// NATS JetStream 实现（多实例部署）
type NATSEventBus struct {
    conn *nats.Conn
    js   nats.JetStreamContext
}
```

### 6.3 核心处理管线

```
PM 数据管线:
  ACS Upload TransferComplete
    → pm.file.received
    → PM Collector 从 MinIO 下载文件
    → XML 解析提取计数器
    → 批量写入 TimescaleDB pm_counters
    → KPI 引擎计算衍生指标
    → 写入 TimescaleDB kpi_values
    → oss.pm.export（若配置了北向推送）

告警管线:
  ACS 解析 Inform ALARM 事件
    → alarm.raised
    → 告警引擎：去重、关联、抑制
    → 写入活跃告警表（Redis + PostgreSQL）
    → oss.alarm.forward（若配置了北向推送）
    → 清除时: alarm.cleared → 更新生命周期

自动开站管线:
  ACS 收到 Bootstrap Inform
    → device.inform.bootstrap
    → 开站工作流：识别设备（OUI + 序列号）
    → 匹配开站模板
    → 排队命令：GetParameterValues, SetParameterValues, Download
    → ACS 在下次会话中执行命令
    → 验证配置
    → 标记设备为 active
```

---

## 七、可扩展性设计

### 7.1 水平扩展策略

```
Tier 1（10 万基站）— 单集群:
  ┌──────────────────────────────────────────┐
  │ 2-3 ACS 实例（LB + 一致性哈希）            │
  │ 1 应用服务器实例（模块化单体）              │
  │ 1 PostgreSQL 主 + 1 只读副本              │
  │ 1 TimescaleDB 实例                        │
  │ 3 节点 Redis Cluster                      │
  │ 3 节点 NATS Cluster                       │
  │ 1 MinIO 实例                              │
  └──────────────────────────────────────────┘

Tier 2（100 万基站）— 多集群:
  ┌──────────────────────────────────────────┐
  │ 10-20 ACS 实例                            │
  │ 3-5 应用服务器实例（可按模块拆分）           │
  │ PostgreSQL: 按运营商分区，区域只读副本       │
  │ TimescaleDB: PM 独立实例，分布式超表         │
  │ Redis: 更大集群，会话和缓存可分离            │
  │ NATS: 更大集群（或切换 Kafka）              │
  │ MinIO: 分布式模式（4+ 节点）               │
  └──────────────────────────────────────────┘
```

### 7.2 分片策略

**自然分片键**：设备序列号（device serial number / UUID）

- 均匀分布（序列号伪随机）
- 数据局部性（同一设备的所有数据在同一分片）
- 路由简单（hash(serial) % num_shards）

**运营商隔离（多租户）**：
- PostgreSQL：按运营商列表分区 `PARTITION BY LIST (carrier)`
- TimescaleDB：时间 + 运营商空间双维分区
- Redis：Key 前缀按运营商
- NATS：Subject 层级按运营商（如 `cmcc.alarm.raised`）

### 7.3 背压与限流

```go
// 每设备限流器（防止 Inform 洪泛）
type DeviceRateLimiter struct {
    limiters sync.Map // map[string]*rate.Limiter
    limit    rate.Limit
    burst    int
}

// ACS 全局准入控制
type AdmissionController struct {
    maxConcurrentSessions int64
    currentSessions       atomic.Int64
}

// PM/MR 文件处理工作池
type WorkerPool struct {
    jobs    chan Job
    workers int
    wg      sync.WaitGroup
}
```

---

## 八、高可用设计

### 8.1 各组件 HA 策略

| 组件 | HA 策略 |
|------|--------|
| ACS 引擎 | 多实例无状态部署，LB 健康检查，任意实例可处理任意设备 |
| 应用服务器 | 多实例无状态部署（状态在 DB/Redis） |
| PostgreSQL | 主从流复制 + Patroni 自动 Failover |
| TimescaleDB | 同 PostgreSQL；100 万规模考虑多节点分布式超表 |
| Redis | Redis Cluster（6 节点：3 主 + 3 从） |
| NATS | 3 节点集群 + JetStream |
| MinIO | 纠删码模式（4+ 节点） |

### 8.2 故障场景与恢复

| 故障 | 影响 | 恢复 |
|------|------|------|
| ACS 实例宕机 | LB 检测到健康检查失败 | 流量自动路由到其余实例；CPE 超时后重连；无数据丢失（命令在 Redis） |
| 应用服务器宕机 | 进行中的请求失败 | LB 路由到其余实例；后台任务由其他实例通过 NATS 消费组接管 |
| PostgreSQL 主库宕机 | 写操作暂停 | Patroni 30s 内提升备库；应用通过连接池自动重连 |
| Redis 节点故障 | 会话临时中断 | Redis Cluster 自动提升副本（~5s）；CPE 会话重建 |
| NATS 节点故障 | 消息投递短暂延迟 | JetStream 跨节点复制；消费者自动重连并从上次 ACK 恢复 |

---

## 九、部署架构

### 9.1 Docker 镜像

```
1. omcgo-acs:latest
   - TR069 ACS 引擎
   - 端口: 7547 (HTTP), 7548 (HTTPS), 9090 (metrics)
   - 环境变量: REDIS_URL, NATS_URL, DB_URL, TLS_CERT_PATH

2. omcgo-app:latest
   - 主应用（除 ACS 外所有模块）
   - 端口: 8080 (REST), 8443 (HTTPS), 9091 (metrics), 50051 (gRPC)
   - 环境变量: DB_URL, TSDB_URL, REDIS_URL, NATS_URL, MINIO_URL

3. omcgo-worker:latest
   - 后台工作进程（PM/MR 文件处理、KPI 计算）
   - 端口: 9092 (metrics)
   - 从 NATS 队列消费

4. omcgo-migrate:latest
   - 数据库迁移（init container）
```

### 9.2 Kubernetes 部署拓扑

```yaml
Namespace: omcgo

# ACS 引擎 — HPA 基于活跃连接数
Deployment: acs         (replicas: 2-20, HPA)
Service:    acs-lb      (type: LoadBalancer, port 7547/7548)

# 应用服务器 — HPA 基于 CPU/请求速率
Deployment: app         (replicas: 2-5, HPA)
Service:    app-api     (type: ClusterIP, port 8080)
Ingress:    api.omc.example.com → app-api

# 后台工作进程 — 基于队列深度弹性伸缩
Deployment: worker      (replicas: 2-10, KEDA)

# 数据存储（StatefulSet 或托管服务）
StatefulSet: postgresql  (replicas: 2, Patroni)
StatefulSet: redis       (replicas: 6, Cluster)
StatefulSet: nats        (replicas: 3, JetStream)
StatefulSet: minio       (replicas: 4, distributed)
```

---

## 十、项目目录结构

```
omcgo/
├── cmd/                            # 入口
│   ├── acs/main.go                 # TR069 ACS 引擎
│   ├── app/main.go                 # 主应用
│   ├── worker/main.go              # 后台工作进程
│   ├── migrate/main.go             # 数据库迁移
│   └── omcctl/main.go              # CLI 管理工具
│
├── internal/                       # 私有应用代码
│   ├── acs/                        # F01: TR069 ACS 引擎
│   │   ├── server.go
│   │   ├── handler.go
│   │   ├── session.go
│   │   ├── session_store.go        # Redis 会话存储
│   │   ├── soap/
│   │   │   ├── codec.go            # SOAP 编解码
│   │   │   ├── inform.go           # Inform 解析
│   │   │   └── templates.go        # 预编译 SOAP 模板
│   │   ├── rpc/
│   │   │   ├── get_parameter_values.go
│   │   │   ├── set_parameter_values.go
│   │   │   ├── add_object.go
│   │   │   ├── delete_object.go
│   │   │   ├── download.go
│   │   │   ├── upload.go
│   │   │   ├── reboot.go
│   │   │   ├── factory_reset.go
│   │   │   └── schedule_inform.go
│   │   ├── connreq/client.go       # Connection Request
│   │   ├── cmdqueue/queue.go       # Redis 命令队列
│   │   └── auth/
│   │       ├── digest.go
│   │       └── basic.go
│   │
│   ├── config/                     # F02: 数据模型与配置管理
│   │   ├── datamodel/
│   │   │   ├── registry.go         # 模型加载与缓存
│   │   │   ├── parameter.go        # 参数树类型
│   │   │   └── validator.go
│   │   ├── template/
│   │   │   ├── template.go
│   │   │   └── matcher.go          # 模板匹配
│   │   ├── audit/
│   │   ├── backup/
│   │   └── service.go
│   │
│   ├── pm/                         # F03: 性能管理
│   │   ├── collector/
│   │   │   ├── collector.go
│   │   │   └── parser_xml.go
│   │   ├── counter/repository.go
│   │   ├── kpi/
│   │   │   ├── engine.go
│   │   │   ├── formulas.go
│   │   │   ├── formulas_lte.go
│   │   │   └── formulas_nr.go
│   │   ├── aggregation/
│   │   └── service.go
│   │
│   ├── alarm/                      # F04: 告警管理
│   │   ├── receiver.go
│   │   ├── engine.go               # 关联、去重、抑制
│   │   ├── lifecycle.go
│   │   ├── store.go
│   │   ├── forward.go
│   │   └── service.go
│   │
│   ├── mr/                         # F05: 测量报告
│   │   ├── collector/collector.go
│   │   ├── parser/
│   │   │   ├── mro.go
│   │   │   ├── mrs.go
│   │   │   └── mre.go
│   │   ├── store/
│   │   └── service.go
│   │
│   ├── omcr/                       # F06: OMC-R 核心功能
│   │   ├── topology/service.go
│   │   ├── device/
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── software/service.go
│   │   └── admin/
│   │       ├── service.go
│   │       └── rbac.go
│   │
│   ├── nedirect/                   # F07: 网元直连接口
│   │   ├── server.go
│   │   └── handler.go
│   │
│   ├── northbound/                 # F08: 北向/OSS 接口
│   │   ├── api/
│   │   │   ├── router.go
│   │   │   ├── pm_handler.go
│   │   │   ├── alarm_handler.go
│   │   │   └── config_handler.go
│   │   ├── push/
│   │   └── sync/
│   │
│   ├── provision/                  # F09: 自动开站
│   │   ├── workflow/
│   │   │   ├── engine.go
│   │   │   └── states.go
│   │   ├── template/
│   │   └── service.go
│   │
│   ├── interop/                    # F10: 互操作测试
│   │   ├── conformance/
│   │   └── validator/
│   │
│   ├── carrier/                    # 运营商抽象层
│   │   ├── carrier.go              # Carrier 接口
│   │   ├── registry.go             # CarrierRegistry
│   │   ├── cmcc/                   # 中国移动
│   │   │   ├── adapter.go
│   │   │   ├── datamodel_lte.go
│   │   │   └── datamodel_nr.go
│   │   ├── ctcc/                   # 中国电信
│   │   │   ├── adapter.go
│   │   │   ├── datamodel_4g.go
│   │   │   └── datamodel_5g.go
│   │   └── cucc/                   # 中国联通
│   │       ├── adapter.go
│   │       └── datamodel_nr.go
│   │
│   ├── common/                     # 公共类型与工具
│   │   ├── model/                  # 领域模型
│   │   │   ├── device.go
│   │   │   ├── alarm.go
│   │   │   ├── parameter.go
│   │   │   ├── pm_counter.go
│   │   │   └── kpi.go
│   │   ├── event/                  # 事件总线
│   │   │   ├── bus.go
│   │   │   ├── types.go
│   │   │   └── nats.go
│   │   ├── errors/
│   │   └── middleware/
│   │       ├── auth.go
│   │       ├── logging.go
│   │       ├── metrics.go
│   │       └── recovery.go
│   │
│   └── infra/                      # 基础设施适配器
│       ├── db/
│       │   ├── postgres.go
│       │   └── timescale.go
│       ├── cache/redis.go
│       ├── mq/nats.go
│       └── storage/minio.go
│
├── pkg/                            # 公共库（可复用）
│   ├── tr069/
│   │   ├── types.go                # TR069 RPC 类型定义
│   │   ├── events.go               # Inform 事件码
│   │   └── faults.go               # CWMP 错误码
│   ├── soap/envelope.go            # 通用 SOAP 工具
│   └── xmlutil/
│
├── api/                            # API 定义
│   ├── openapi/
│   │   ├── northbound.yaml
│   │   └── dashboard.yaml
│   └── proto/
│       ├── acs.proto
│       ├── device.proto
│       └── alarm.proto
│
├── migrations/                     # 数据库迁移文件
│   ├── 001_create_devices.up.sql
│   ├── 001_create_devices.down.sql
│   ├── 002_create_pm_counters.up.sql
│   └── ...
│
├── configs/                        # 配置文件
│   ├── acs.yaml
│   ├── app.yaml
│   └── worker.yaml
│
├── deployments/                    # 部署清单
│   ├── docker/
│   │   ├── Dockerfile.acs
│   │   ├── Dockerfile.app
│   │   ├── Dockerfile.worker
│   │   └── docker-compose.yml
│   └── k8s/
│       ├── acs/
│       ├── app/
│       ├── worker/
│       └── infra/
│
├── datamodels/                     # TR069 数据模型定义文件
│   ├── cmcc/
│   │   ├── lte_v2.3.json
│   │   └── nr_v1.9.4.json
│   ├── ctcc/
│   │   ├── 4g_v1.1.json
│   │   └── 5g_v2.1.7.json
│   └── cucc/
│       └── nr_v1.1.json
│
├── doc/                            # 文档（已有）
├── scripts/                        # 构建/部署/测试脚本
├── test/                           # 集成/E2E 测试
│   ├── integration/
│   ├── e2e/
│   └── fixtures/                   # 测试数据（SOAP 报文、PM 文件示例等）
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 十一、API 设计

### 11.1 北向 REST API（F08 — 面向 OSS/上层系统）

```
Base URL: /api/v1

设备管理:
  GET    /devices                          列表（分页、按运营商/制式/状态过滤）
  GET    /devices/{id}                     详情
  GET    /devices/{id}/parameters          获取设备参数
  POST   /devices/{id}/parameters          设置设备参数（排队命令）
  POST   /devices/{id}/reboot              远程重启
  POST   /devices/{id}/factory-reset       恢复出厂
  GET    /devices/{id}/config-backup       下载配置备份
  POST   /devices/{id}/config-restore      恢复配置
  POST   /devices/{id}/firmware-upgrade    触发固件升级

性能管理:
  GET    /pm/counters                      查询 PM 计数器
  GET    /pm/kpi                           查询 KPI 值
  GET    /pm/kpi/definitions               KPI 定义列表
  POST   /pm/export                        触发 PM 数据导出

告警管理:
  GET    /alarms/active                    活跃告警列表
  GET    /alarms/history                   历史告警查询
  POST   /alarms/{id}/acknowledge          确认告警
  POST   /alarms/{id}/clear               手动清除告警
  GET    /alarms/statistics                告警统计

测量报告:
  GET    /mr/files                         MR 文件列表
  GET    /mr/files/{id}/download           下载原始 MR 文件
  GET    /mr/data                          查询解析后 MR 数据

自动开站:
  GET    /provisioning/tasks               开站任务列表
  POST   /provisioning/tasks               创建开站任务
  GET    /provisioning/tasks/{id}          任务状态
  GET    /provisioning/templates           开站模板列表
  POST   /provisioning/templates           创建模板

配置管理:
  GET    /config/templates                 配置模板列表
  POST   /config/templates                 创建配置模板
  POST   /config/batch                     执行批量配置

拓扑:
  GET    /topology/tree                    设备层级树
  GET    /topology/groups                  设备分组

系统:
  GET    /system/health                    健康检查
  GET    /system/metrics                   系统指标汇总
  GET    /system/carriers                  支持的运营商及数据模型版本
```

### 11.2 内部 gRPC API（ACS ↔ 应用服务器）

```protobuf
service ACSControl {
    // 向设备排队命令
    rpc QueueCommand(QueueCommandRequest) returns (QueueCommandResponse);
    // 触发 Connection Request
    rpc SendConnectionRequest(ConnectionRequestMsg) returns (ConnectionRequestResponse);
    // 流式设备事件
    rpc StreamDeviceEvents(StreamEventsRequest) returns (stream DeviceEvent);
    // ACS 实例状态
    rpc GetStatus(StatusRequest) returns (ACSStatus);
}

service DeviceService {
    rpc GetDevice(GetDeviceRequest) returns (Device);
    rpc ListDevices(ListDevicesRequest) returns (ListDevicesResponse);
    rpc UpdateDeviceStatus(UpdateDeviceStatusRequest) returns (Device);
}
```

---

## 十二、运营商抽象层

### 12.1 Carrier 接口定义

```go
// internal/carrier/carrier.go

type Carrier interface {
    // 标识
    Code() CarrierCode    // "cmcc", "ctcc", "cucc"
    Name() string         // "中国移动", "中国电信", "中国联通"

    // 数据模型
    SupportedTechnologies() []Technology
    DataModelVersions(tech Technology) []string
    LoadDataModel(tech Technology, version string) (*DataModel, error)

    // 参数映射（运营商路径 ↔ 统一内部名称）
    MapParameterToUnified(carrierPath string) string
    MapUnifiedToParameter(unifiedName string) string

    // 开站模板
    ProvisioningTemplates(tech Technology) []*ProvisionTemplate

    // KPI 定义
    KPIDefinitions(tech Technology) []*KPIDefinition

    // 告警严重性映射
    AlarmSeverityMapping(carrierAlarmCode string) AlarmSeverity

    // Inform 事件处理
    HandleInformEvents(device *Device, events []InformEvent) ([]Command, error)

    // 参数校验
    ValidateParameter(path string, value string) error
}

type CarrierCode string

const (
    CarrierCMCC CarrierCode = "cmcc" // 中国移动
    CarrierCTCC CarrierCode = "ctcc" // 中国电信
    CarrierCUCC CarrierCode = "cucc" // 中国联通
)

type Technology string

const (
    TechLTE Technology = "lte"
    TechNR  Technology = "nr"
)
```

### 12.2 CarrierRegistry

```go
type CarrierRegistry struct {
    carriers map[CarrierCode]Carrier
}

func NewRegistry() *CarrierRegistry {
    r := &CarrierRegistry{carriers: make(map[CarrierCode]Carrier)}
    r.Register(CarrierCMCC, cmcc.New())
    r.Register(CarrierCTCC, ctcc.New())
    r.Register(CarrierCUCC, cucc.New())
    return r
}

func (r *CarrierRegistry) Get(code CarrierCode) (Carrier, error) {
    c, ok := r.carriers[code]
    if !ok {
        return nil, fmt.Errorf("unsupported carrier: %s", code)
    }
    return c, nil
}
```

### 12.3 运营商适配器示例（中国移动）

```go
// internal/carrier/cmcc/adapter.go

type CMCCCarrier struct {
    dataModels map[Technology]map[string]*DataModel
    kpiDefs    map[Technology][]*KPIDefinition
}

func New() *CMCCCarrier {
    return &CMCCCarrier{
        dataModels: make(map[Technology]map[string]*DataModel),
        kpiDefs:    make(map[Technology][]*KPIDefinition),
    }
}

func (c *CMCCCarrier) Code() CarrierCode { return CarrierCMCC }
func (c *CMCCCarrier) Name() string      { return "中国移动" }

func (c *CMCCCarrier) SupportedTechnologies() []Technology {
    return []Technology{TechLTE, TechNR}
}

func (c *CMCCCarrier) DataModelVersions(tech Technology) []string {
    switch tech {
    case TechLTE:
        return []string{"V2.1", "V2.3"}
    case TechNR:
        return []string{"V1.7", "V1.9.4"}
    }
    return nil
}

// 中国移动特有: 支持网元直连接口（F07）
func (c *CMCCCarrier) SupportsDirectConnection() bool { return true }
```

---

## 十三、数据模型抽象

### 13.1 统一参数模型

核心挑战：各运营商对等价概念定义了不同参数路径。解决方案：**双层模型**。

```go
// pkg/tr069/types.go — 协议层类型
type ParameterValueStruct struct {
    Name  string `xml:"Name"`
    Value string `xml:"Value"`
    Type  string `xml:"type,attr"` // xsd: string, int, unsignedInt, boolean, dateTime
}

// internal/config/datamodel/parameter.go — 内部模型
type Parameter struct {
    Path        string         // TR069 路径: "Device.DeviceInfo.Manufacturer"
    UnifiedName string         // 内部名称: "device.manufacturer"
    Type        ParameterType  // string, int, uint, bool, datetime
    Writable    bool
    Description string
    Constraints *Constraints
    Category    string         // radio, network, security, management
}

type Constraints struct {
    MinValue   *int64
    MaxValue   *int64
    EnumValues []string
    Pattern    string // regex
    MaxLength  int
}

// DataModel — 完整数据模型（运营商/制式/版本）
type DataModel struct {
    Carrier    CarrierCode
    Technology Technology
    Version    string
    RootObject string                  // "Device." 或 "InternetGatewayDevice."
    Parameters map[string]*Parameter   // key: TR069 路径
    Objects    map[string]*Object      // key: TR069 对象路径
}

type Object struct {
    Path          string
    MultiInstance bool     // 可多实例（如 "Device.WiFi.Radio.{i}."）
    Writable      bool     // 支持 AddObject/DeleteObject
    Parameters    []string
    SubObjects    []string
}
```

### 13.2 统一设备视图

```go
// internal/common/model/device.go

type Device struct {
    ID                   uuid.UUID
    SerialNumber         string
    OUI                  string
    ProductClass         string
    Manufacturer         string
    ModelName            string

    // 分类
    Carrier              CarrierCode
    Technology           Technology
    DataModelVersion     string

    // 状态
    Status               DeviceStatus
    FirmwareVersion      string

    // 连接
    IPAddress            string
    ConnectionRequestURL string
    LastInformAt         time.Time
    LastInformEvents     []string
    InformInterval       int // 秒

    // 站址
    SiteName             string
    SiteID               string
    Latitude             float64
    Longitude            float64

    // 运营商扩展数据
    ExtensionData        map[string]interface{}

    CreatedAt            time.Time
    UpdatedAt            time.Time
}

type DeviceStatus string

const (
    DeviceDiscovered     DeviceStatus = "discovered"     // Bootstrap Inform
    DeviceRegistered     DeviceStatus = "registered"     // 身份验证通过
    DeviceProvisioning   DeviceStatus = "provisioning"   // 配置下发中
    DeviceActive         DeviceStatus = "active"         // 正常运行
    DeviceMaintenance    DeviceStatus = "maintenance"    // 维护中
    DeviceOffline        DeviceStatus = "offline"        // 心跳丢失
    DeviceDecommissioned DeviceStatus = "decommissioned" // 已退服
)
```

### 13.3 数据模型加载策略

数据模型定义以 JSON 文件存储在 `datamodels/` 目录（源自运营商规范 Excel/Word 文档解析），启动时加载到内存，并缓存至 Redis。

```go
type DataModelRegistry struct {
    models   map[string]*DataModel // key: "cmcc:nr:V1.9.4"
    cache    cache.Cache           // Redis
    basePath string                // datamodels/ 目录路径
}

func (r *DataModelRegistry) Get(carrier CarrierCode, tech Technology, version string) (*DataModel, error) {
    key := fmt.Sprintf("%s:%s:%s", carrier, tech, version)

    // 优先查内存
    if m, ok := r.models[key]; ok {
        return m, nil
    }
    // 查 Redis 缓存
    // 从 JSON 文件加载
    // 缓存后返回
}

// GetForDevice — 根据设备的运营商/制式/版本获取对应数据模型
func (r *DataModelRegistry) GetForDevice(device *Device) (*DataModel, error) {
    return r.Get(device.Carrier, device.Technology, device.DataModelVersion)
}
```

---

## 十四、关键架构决策汇总

| 决策项 | 选型 | 理由 |
|--------|------|------|
| 架构模式 | 模块化单体 + 独立 ACS | 降低初期复杂度，ACS 需独立扩展 |
| 主数据库 | PostgreSQL + TimescaleDB | 统一生态，配置 + 时序数据一站式 |
| 会话存储 | Redis Cluster | 亚毫秒延迟，TTL 自动过期 |
| 消息中间件 | NATS JetStream | Go 原生、轻量、吞吐足够 |
| 对象存储 | MinIO | S3 兼容、自托管、PM/MR 文件高效存取 |
| XML 处理 | stdlib + 预编译模板 | 避免重框架；发送用模板、接收用流式解析 |
| 运营商抽象 | 接口 + 适配器 | 清晰分离，便于新增运营商 |
| 数据模型存储 | JSON 文件 + Redis 缓存 | 数据模型低频变更，无需入库 |
| 外部 API | REST (北向) + gRPC (内部) | REST 兼容 OSS 生态，gRPC 保障内部性能 |
| 容器编排 | Kubernetes | HPA 弹性伸缩，StatefulSet 管理有状态组件 |
| 分片键 | 设备序列号 | 天然均匀分布，保证数据局部性 |

---

## 十五、实施路线图

### 第一阶段：基础建设

```
目标: ACS 引擎能接收 Inform 并注册设备

任务:
  1. 项目脚手架: Go module、目录结构、Makefile、Docker
  2. 基础设施层: PostgreSQL、Redis、MinIO、NATS 连接
  3. pkg/tr069: TR069 类型定义、SOAP 编解码、CWMP 错误码
  4. internal/acs: ACS HTTP 服务器、Inform 解析、会话状态机
  5. internal/common/model: Device 模型、事件类型
  6. internal/omcr/device: 从 Bootstrap Inform 注册设备
  7. 端到端集成: ACS 接收 Inform → 创建设备记录 → 响应
```

### 第二阶段：核心功能

```
目标: 完整的设备管理和自动开站流程

任务:
  8. internal/acs/rpc: 全部 RPC 方法实现
  9. internal/acs/cmdqueue: Redis 命令队列
  10. internal/acs/connreq: Connection Request 发送
  11. internal/config/datamodel: 数据模型注册表、JSON 加载
  12. internal/carrier: Carrier 接口 + 中国移动适配器（首家运营商）
  13. internal/provision: 自动开站工作流
  14. 端到端: Bootstrap → 识别 → 开站 → 激活
```

### 第三阶段：数据管线

```
目标: PM、告警、MR 数据全链路

任务:
  15. internal/pm: PM 文件采集、XML 解析、TimescaleDB 存储
  16. internal/pm/kpi: KPI 计算引擎
  17. internal/alarm: 告警管线（接收、去重、存储）
  18. internal/mr: MR 文件采集与解析
  19. internal/carrier/ctcc: 中国电信适配器
  20. internal/carrier/cucc: 中国联通适配器
```

### 第四阶段：北向接口与规模化

```
目标: OSS 对接、10 万级验证、生产加固

任务:
  21. internal/northbound: REST API for OSS
  22. internal/omcr: 完整 OMC-R 功能（拓扑、固件管理、用户管理）
  23. 10 万基站负载测试
  24. 水平扩展验证
  25. 生产加固（TLS、认证、限流、监控面板）
```

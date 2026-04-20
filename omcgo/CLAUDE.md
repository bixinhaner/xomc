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
| 链路追踪 | `go.opentelemetry.io/otel` | 分布式链路追踪 |
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
│   │   ├── cmdqueue/               #   Redis 命令队列
│   │   └── auth/                   #   CPE 认证
│   │
│   ├── config/                     # F02: 数据模型与配置管理（业务域）
│   │   ├── datamodel/              #   数据模型注册表、三级回退解析、缓存
│   │   ├── template/               #   配置模板
│   │   ├── baseline/               #   配置基线
│   │   └── sync_handler.go         #   配置同步
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

#### 5.5.9 迁移文件自查清单

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
`acs`, `config`, `datamodel`, `pm`, `alarm`, `mr`, `device`, `admin`, `topology`, `software`, `backup`, `dashboard`, `ops`, `report`, `mml`, `filemanager`, `syslog`, `license`, `nedirect`, `northbound`, `provision`, `interop`, `carrier`, `components`, `api`, `deploy`

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

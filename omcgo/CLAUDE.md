# CLAUDE.md — OMC Go 后端指导

> 本文件是根 `../CLAUDE.md` 的**后端补充**：只放后端独有内容（架构决策、TR069/ACS、参数字典、迁移、事件驱动）。
> 功能域定义 / 技术选型名 / Git 工作流 / 文档索引 / 常用命令 → **以根 CLAUDE.md 为准**，本文件不重复。

---

## 1. 后端架构决策（ADR 级，后端独有）

### 核心选型：模块化单体 + 独立 ACS 引擎，不用微服务、不用 go-zero

| 决策 | 理由 |
|------|------|
| 模块化单体 | 100K 规模单 Go 进程足够；10 个功能域跨域交互密集；避免分布式事务复杂度 |
| ACS 独立部署 | TR069 SOAP/XML 有状态会话、独立扩展需求、并发模型与管理面不同 |
| 拒绝 go-zero | go-zero 仅支持 JSON/Protobuf，无法处理 SOAP/XML；NATS 不在其生态；ACS 无法用 goctl 代码生成 |
| 选 NATS 而非 Kafka | Go 原生、轻量、吞吐足够，运维成本低 |

> 完整论证（含权衡与被否方案）见 `../docs/adr/`。三个部署单元（app/acs/worker）与端口见根 CLAUDE.md §5。演进路径：当前模块化单体，100 万规模时按功能域渐进拆分。

---

## 2. 依赖速查（import 路径 + 用途）

> 选型名见根 CLAUDE.md §4；下表补「具体 import 路径 + 后端用法」。**语言：Go 1.25**。

| 用途 | 库 |
|------|------|
| HTTP（ACS / 管理面） | `net/http` stdlib（完全控制请求生命周期）/ `github.com/gin-gonic/gin` |
| RPC | `google.golang.org/grpc`（ACS ↔ App）|
| SOAP 发送 / 接收 / 遍历 | `text/template`（预编译，避免反射）/ `encoding/xml` Decoder（流式）/ `github.com/beevik/etree` |
| 配置 / CLI | `github.com/spf13/viper` / `github.com/spf13/cobra` |
| DB / 连接池 / SQL | `github.com/jackc/pgx/v5` + `pgxpool` / `github.com/Masterminds/squirrel`（**不用 ORM**）|
| 迁移 | `pressly/goose/v3` |
| Redis / NATS / MinIO | `github.com/redis/go-redis/v9` / `github.com/nats-io/nats.go`（JetStream）/ `github.com/minio/minio-go/v7` |
| 日志 / 指标 / 追踪 | `go.uber.org/zap` / `github.com/prometheus/client_golang` / `go.opentelemetry.io/otel`（OTLP gRPC）|
| 校验 / UUID / 定时 / 限流 | `go-playground/validator/v10` / `google/uuid` / `robfig/cron/v3` / `golang.org/x/time/rate` |
| 测试 / API 文档 | `stretchr/testify` / `swaggo/swag` |

> 可观测性落地链路（otelcol → Tempo/Loki/Prometheus，trace↔log 关联）见根 CLAUDE.md §4 + `../docs/operations/OMC可观测性使用手册.md`。
> 存储选型（PG16/TimescaleDB/Redis/MinIO/NATS）见根 CLAUDE.md §12。

---

## 3. 项目目录结构

> 易漂移，改大结构后请重画。可用 `go list ./internal/...` 核对模块清单。

```
omcgo/
├── cmd/                            # 入口（7 个二进制）
│   ├── app/                        #   主应用：main.go + bootstrap.go + provider/（路由注册 + DI 容器）
│   │   ├── etc/                    #     config.{dev,test,prod}.yaml
│   │   └── provider/              #     router.go / container.go / modules.go + 各模块 wiring（admin/device/pm/...）
│   ├── acs/                        #   TR069 ACS 引擎（main.go + etc/）
│   ├── worker/                     #   后台工作进程（main.go + etc/）
│   ├── migrate/                    #   数据库迁移
│   ├── omcctl/                     #   CLI 管理工具
│   ├── tools/                      #   gen_seed_sql 等种子 SQL 生成器
│   └── backup-reencrypt/           #   备份密钥轮换 / 重加密 CLI
│
├── global/                         # 全局常量与错误码（无框架依赖）
│   ├── consts.go                   #   运营商/制式/设备等全局常量
│   └── errors.go                   #   全局错误码（数量现查 grep -c ErrCode）
│
├── internal/                       # 私有代码（35 个业务/基础设施模块，扁平结构）
│   ├── acs/                        # F01: TR069 ACS 引擎
│   │   ├── server.go handler.go session.go session_store.go   #   HTTP 服务器 / 请求处理 / 会话状态机
│   │   ├── admission.go ratelimit.go path_translator.go        #   准入控制 / 限流 / 路径翻译
│   │   ├── trace_capture.go task_service.go                    #   报文跟踪捕获 / 任务派发
│   │   └── soap/ rpc/ rpclog/ connreq/ auth/ download/ upload/ stun/ transfercfg/ pathutil/
│   ├── trace/                      # F01: TR069 报文跟踪（T-0137，capture_consumer/bulk_store/exporter）
│   │
│   ├── config/                     # F02: 配置管理 — parammodel/ template/ baseline/ audit/ backup/ + sync_handler.go
│   ├── product/                    # F02: 产品装配件 + ProductRegistry（productClass 正则路由）
│   ├── quicksettings/              # F02: 参数快速设置（进程内 Registry，启动期 Loader 从 XML 加载）
│   ├── devsweep/                   # F02: 单设备 param_mappings sweep（按 SN GPV 探测落库）
│   │
│   ├── pm/                         # F03: 性能管理（含 indicator/ 指标库）
│   ├── alarm/                      # F04: 告警管理（含 definition/ 告警定义库）
│   ├── eventlog/                   # F04: 设备事件日志（落 event_logs）
│   ├── mr/                         # F05: 测量报告
│   │
│   ├── device/ admin/ topology/ software/ backup/             # F06: 设备/RBAC/拓扑/固件/配置备份
│   ├── dashboard/ ops/ report/ mml/ filemanager/ syslog/ license/   # F06
│   ├── bundle/                     # F06: 文件管理批量下载（同步流式，不建表/不开后台 goroutine）
│   ├── ufte/                       # F06: 统一文件传输/任务引擎（OUTPUT 类：备份/日志采集/配置恢复）
│   ├── stationlog/                 # F06: 基站日志采集（运行日志 FileType 6 / 故障日志 8）
│   ├── rebootrecord/               # F06: 统一重启记录（两张互斥表合成）
│   │
│   ├── nedirect/ northbound/ provision/ interop/             # F07-F10
│   │
│   ├── notification/ task/ transfer/ events/                # 跨域基础设施（详见根 §6）
│   │
│   └── core/                       # 进程级基础设施
│       ├── carrier/                #   运营商适配（接口 carrier.go + registry + cmcc/ctcc/cucc 适配器）
│       ├── appconfig/ asyncjob/ components/ dictloader/      #   配置 / 异步任务 / 基础设施适配器 / 字典加载
│       ├── errors/ event/ health/ middleware/ model/         #   业务错误 / EventBus / 健康 / 中间件 / 领域类型
│       └── redact/ reliability/ response/ storage/ tracing/ utils/
│
├── pkg/                            # 可复用公共库：tr069/（类型、事件码）soap/ xmlutil/
├── data/                           # 活跃字典 XML/JSON：param-mappings/ indicator-library/ alarm-definitions/ quicksettings/ mml-catalog/（dictloader 启动期加载，data 外置 bind-mount）
├── datamodels/                     # 仅剩 mml-catalog/（JSON）+ templates/（空）；TR069 参数 XML 已迁 data/
├── migrations/                     # 数据库迁移（goose；consolidated baseline，见 migrations/README.md）+ seed/
├── api/ configs/ queries/ scripts/ test/ docs/ 规范/        # API 定义 / 压测配置 / SQL / 脚本 / 测试 / 后端文档 / 运营商规范原件
├── go.mod  Makefile  CLAUDE.md
```

---

## 4. 开发规范

### 4.1 Go 编码

**命名**：exported `PascalCase` / unexported `camelCase`；接口用名词或动词（`SessionStore`、`EventPublisher`），不加 `I` 前缀；包名简短小写单数（`alarm` 不是 `alarms`）。

**领域常量**（全局统一用常量，不硬编码字符串）：
```go
CarrierCMCC CarrierCode = "cmcc"   // 中国移动
CarrierCTCC CarrierCode = "ctcc"   // 中国电信
CarrierCUCC CarrierCode = "cucc"   // 中国联通
TechLTE Technology = "lte"
TechNR  Technology = "nr"
```
> 旧的 `ScopeProduct/ScopeOUI/ScopeCarrierDefault` 三级回退常量为 legacy（T-0098 后参数走 product + ParamModel 路由，不再用三级 Scope 解析）；`consts.go` 仍保留仅向后兼容，新代码勿用。

**错误处理**：返回 `error` 不 `panic`（除不可恢复的初始化）；wrap 加上下文 `fmt.Errorf("resolve param model: %w", err)`；ACS 会话内错误不能影响其他会话（goroutine 隔离）；区分 sentinel/自定义错误类型。

**接口优先**（核心领域概念用接口，便于测试）：`Carrier`（运营商适配，`internal/core/carrier`）· `SessionStore`（TR069 会话）· `TaskService`（统一任务队列 Redis+PG 双写）· `EventBus`（事件总线）· `ParamRegistry`/`ProductRegistry`/`Translator`（参数字典，T-0098）。

**运营商适配**：通过 `Carrier` 接口 + 适配器；**禁止** `if carrier == "cmcc"`；新增运营商 = 新增适配器 + 注册到 `CarrierRegistry`。

**数据库访问**：squirrel 构建 SQL + pgx/v5 执行，**不用 ORM**；每模块自己的 repository 层；复杂查询直接写 SQL，简单 CRUD 用 squirrel。

**测试**：`testify` 断言 + table-driven；单测与源文件同目录（`_test.go`）；集成测试 `test/integration/`，E2E `test/e2e/`，夹具 `test/fixtures/`。

### 4.2 TR069/ACS 专项

**SOAP/XML**：发送（ACS→CPE）用 `text/template` 预编译模板避免反射；接收（CPE→ACS）用 `xml.Decoder` 流式解析；动态参数树用 `beevik/etree`。

**会话状态机**：`IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE`

**Redis Key 命名**：
```
acs:session:{device_serial}          — 会话状态 Hash（TTL 5 分钟）
acs:taskq:{device_serial}            — 任务队列 Sorted Set（member=taskID）
acs:task:{taskID}                    — 任务详情 Hash（TTL 24h）
acs:cwmp2task:{hash}                 — CWMP ID → Task ID 映射（TTL 24h）
acs:heartbeat:{device_serial}        — 心跳时间戳（TTL = 2×inform_interval）
acs:connreq:pending:{device_serial}  — Connection Request 去重（TTL 30 秒）
acs:admission:slots                  — 全局准入 Sorted Set（member=sessionID, score=过期 unix 秒；TTL 自愈丢失的 Release）（issue #65 Option B）
acs:device:session:{device_serial}   — 设备当前活跃 sessionID 指针（STRING+TTL，跨实例孤儿会话清理）（issue #65 Option B）
acs:connreq:url:{device_serial}      — Inform 上报的 ConnectionRequestURL（STRING+TTL，HTTP 唤醒回退，镜像 acs:stun）（issue #65 Option B）
acs:auth:nonce:{nonce}               — Digest 一次性 nonce（SETEX 写 + GETDEL 消费，TTL 5min）（issue #65 Option B）
parammodel:default:{paramModelID}                  — ParamRegistry default mapping 缓存（TTL 24h）
parammodel:discovered:{productID}:{swVersion}      — ParamRegistry discovered mapping 缓存（TTL 1h）
product:byProductClass:{productClass}              — ProductRegistry 路由结果缓存（TTL 1h）
parammodel:cache_version                           — 缓存版本号（跨实例协调）
alarm:active:{device_serial}         — 活跃告警 Hash
ratelimit:inform:{device_serial}     — 限流计数器
```

**流量控制**：每设备限流器（`rate.Limiter`）防 Inform 洪泛；全局准入控制器（`AdmissionController`）限并发会话数；PM/MR 文件处理用 WorkerPool 控并发。

> **ACS 横扩去进程态（issue #65 / ADR 0005）**：ACS 4 类会话副作用状态（准入计数 / 设备孤儿会话指针 / ConnectionRequestURL / Digest nonce）已从进程内 sync.Map/atomic 迁到共享 Redis（键见上表 4 条 issue #65 标注）。`AdmissionController`/`DeviceSessionStore`/`ConnReqURLStore`/`auth.NonceStore` 均接口优先 + local/redis 双实现；`cmd/acs/main.go` 在 `inf.Redis != nil` 时注入 Redis 实现。准入用 Sorted Set + 单条 Lua（Acquire/Release 以 sessionID 配对，TTL 自愈），故全局上限真正全局、Challenge/Authenticate 可跨实例、孤儿会话任意实例可清理。**nginx/k8s 不再需要会话亲和（sticky session）**。准入 Redis 错误 fail-closed（503）。

### 4.3 参数模型字典（T-0098）

旧 datamodel 三级回退（product/oui/carrier_default）+ `data_model_definitions` 表已下线，参数走 ParamModel 字典：

- **装配件**：`products` 聚合产品类/参数模型/KPI 平台/告警 ne_type/上传开关；`param_models` + `param_mappings`（默认映射）；`discovered_param_mappings`（设备 FileType=11 上传 XML 后由 IntersectService 写，按 product_id+sw_version 索引）；`standard_params`（standardPath 元属性参考）。
- **路由**：设备上报 productClass → `ProductRegistry.MatchProductClass`（全局正则）→ product → `param_model_id` → `ParamRegistry.GetByProduct`（优先 discovered 精确匹配 swVersion，退化 default，双源合并 MappingSet 带 Source 标记）。
- **双向翻译（Translator）**：standardPath（标准化）↔ privatePath（厂商专有）；模板/SPV/GPV 输入侧用 standardPath，下发/持久化用 privatePath；`{i}` 占位符与运行时实例号 `.N.` 折叠为同一索引键。
- **缓存**：内存 L1（`sync.Map` 进程内 Registry）→ Redis L2（`parammodel:*`/`product:*`）→ PostgreSQL。
- **Loader 启动期加载**（dictloader 框架）：product/parammodel/indicator/alarm-definition 四个 Loader，文件白名单 → 单事务幂等 UPSERT；ModuleGraph 编排。

> 具体迁移号请查 `migrations/`，不在文档硬编码。

### 4.4 三库导入 XML 分层（ParamModel / Indicator / Alarm 同范式）

**当前范式（2026-06-04 定稿）= 单目录 + sidecar + 名称唯一 + 双重唯一硬拒 + data 外置 + 升级反向合并**：
- 单目录（builtin 与 custom 同住 `data/{param-mappings,indicator-library,alarm-definitions}/`，取消 `*-custom` 目录）；来源判定靠 sidecar 空文件 `X.xml.custom`（`source.go::IsCustom/IsDeletable` 读，仅 custom 可删，builtin/unknown → 403）。
- 上传 = `name` 必填 + 双重唯一硬拒（文件名唯一 + 内容主键唯一 `param_models.name`/indicator `platform`/alarm `neType`），命中 → 409；落盘 + 写 sidecar；DELETE 连带删 sidecar。
- 三库差异 + 错误码段（ParamModel 2030 / Indicator 2040 / Alarm 2050）+ 完整代码索引 + 已被推翻的双目录历史 → **`../docs/ref/three-library-xml-import-history.md`**。

> 单实例假设：Upload/Delete/Reload 走进程内 `acquireFileLock`；多实例横扩前必须补 PG advisory lock。

### 4.5 事件驱动

**EventBus 双实现**：`ChannelEventBus`（进程内 Go channel，单进程）/ `NATSEventBus`（NATS JetStream，多实例）。实现在 `internal/core/event/`。

**Core NATS 实时广播例外**：`internal/core/realtime/` 只用于可丢失、非持久化的多订阅者 fan-out；主题必须位于持久化 WorkQueue/EventBus 捕获范围之外，数据库仍是权威来源。禁止用于命令、任务分发或需要历史重放的事件。

**Subject 命名**（点分层级 `domain.action.detail`）：
```
device.inform.{bootstrap|periodic|value_change|alarm}
command.{get_parameters|set_parameters}
pm.file.{received|parsed}          mr.file.{received|parsed}
alarm.{raised|cleared|acknowledged}
oss.alarm.forward                  oss.pm.export
```

### 4.6 数据库迁移核心铁律

**工具**：`pressly/goose/v3`，版本记录在 `goose_db_version`（schema）/ `goose_db_version_seed`（seed）。**2026-05-31 已做 consolidated baseline**（schema 从 `000001` baseline + 增量；非从历史 0 连续），唯一事实源 = `migrations/README.md`。

**新增迁移铁律**：
- 版本号 = 现有最大号 + 1（无跳跃、无重复、不重用已删号、不插低版本）。`ls migrations/ migrations/seed/ | sort | uniq -d` 查撞号。
- DDL 放 `migrations/`，DML 种子放 `migrations/seed/`。
- `DO $$` / `CREATE [OR REPLACE] FUNCTION` / 循环条件 → 必须 `-- +goose StatementBegin/End` 包裹。
- 幂等：`CREATE TABLE IF NOT EXISTS`、`ADD COLUMN IF NOT EXISTS`、INSERT 带 `ON CONFLICT DO NOTHING`。
- Down 段删除 Up 段创建的所有对象。
- **改已被 seed 引用的表**：新增列可空或带 `DEFAULT`；收紧约束前同一迁移先 backfill；删列/改名走两阶段跨 release（`migrate-seed` 是「先全部 schema 再全部 seed」，旧 seed 跑在最终表结构上）。

> 详细踩坑案例库（11 条，含翻车 commit）→ **`../docs/ref/migration-pitfalls.md`**。提交前对照该文件末尾的自查清单。

### 4.7 全局 API 密钥约定（.api-key）

容器内工具/脚本调 OMC HTTP API 零配置可用。

| 属性 | 值 |
|------|------|
| 路径 / 格式 / 权限 | `/var/lib/omcgo/secrets/.api-key` · 单行 `omk_`+32hex（36 字符，无尾换行）· `0640` |
| 签发 | `omcgo-app` 启动期幂等签发（`admin.EnsureInternalAPIKey`）；绑 DB `system` 用户的 `omc-internal` key，scopes=`['*']` |
| 共享卷 | docker-compose named volume `omcgo-secrets`（app rw + worker ro） |

消费：omcctl 用 `--api-key` > `OMCCTL_API_KEY` env > 文件 > 报错；bash 脚本 `API_KEY="${OMCCTL_API_KEY:-$(cat /var/lib/omcgo/secrets/.api-key 2>/dev/null||true)}"` 后 `curl -H "X-API-Key: $API_KEY"`。
**禁止**：入 git（`.gitignore` 已加）· 打日志明文（只允许 `key_prefix`）· 入迁移文件。轮换：重启 app 自动检测失效轮换。
代码：`internal/admin/internal_apikey.go`、`cmd/app/provider/admin.go`、`cmd/omcctl/main.go`。

---

## 5. Git / 文档索引 / 常用命令

- **Git 工作流**（提交格式、scope、Claude git 操作边界）→ 根 CLAUDE.md §8.1（唯一真值源，本文件不重复）。
- **文档索引** → 根 CLAUDE.md §15 + `../docs/README.md`。
- **常用命令**（docker compose 重启规则、make 目标、前端命令）→ 根 CLAUDE.md §14。后端独有：`bash scripts/check-migrations.sh`（校验迁移号）。

# PRD: F08 OSS 北向 SNMP Trap 推送

**PRD ID**：F08-oss-protocol
**功能域**：F08 北向/OSS 接口
**作者**：Claude AI（PM 角色）+ 待人工审批
**创建日期**：2026-04-28
**最后更新**：2026-04-28
**状态**：Draft（待审批）
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`
**关联 Sprint**：Wave-4 Sprint（待立项，预计 sprint-04 / sprint-05 跨双冲刺）
**关联 Risk**：`docs/project/risk-register.md`（待登记 R-013：F08 SNMP Trap 协议联调风险）
**关联 Backlog**：T-0013 F08 SNMP Trap 骨架（本 PRD）→ T-0017 F08 OSS 联调 → T-0020 推送可靠性整链

---

## 1. 业务背景（Why）

OMC 作为运营商小基站网管系统，必须能将告警数据**主动推送**给上游 OSS（Operations Support System）。三大运营商（CMCC/CTCC/CUCC）的网管接口规范都把 **SNMPv2c/SNMPv3 Trap** 列为告警上报标准协议（IETF RFC 3416/3418/3826），同时 OSS 侧的告警网关、工单系统、综合监控平台也普遍以 SNMP Trap 作为统一入口。

当前状态：
- F08 北向已有 HTTP Webhook 推送（`internal/northbound/push/`）和文件订阅（`internal/northbound/sync/`），但**没有 SNMP Trap 通道**
- 三家运营商的入网测试和现网联调都把 SNMP Trap 列为强制项，是从 Beta 进入 RC 必须关闭的 P0 短板
- 不做的后果：T-0017（F08 OSS 联调）和 T-0020（推送可靠性整链）整条 Wave-4 路径阻塞，无法进入运营商联调

为什么现在做：
- W3 收官后 Wave-4 头号 P0；先立 PRD + 设计 + 骨架代码（T-0013），后续 PR 接入 alarm.Engine + Carrier + DI（T-0017），再做可靠性加固（T-0020）
- 骨架阶段不接生产链路，避免对现有 alarm/notification/push 三个稳定模块产生干扰

---

## 2. 用户故事（Who / What）

> **运营商 OSS 集成工程师**：As a OSS 集成工程师，I want OMC 通过 SNMPv2c Trap 把所有严重告警按本运营商网管规范的 MIB 字段推送到 OSS 告警网关（IP:Port），So that 现网告警能被统一监控平台秒级感知，与其他厂商网管对齐。

> **OMC 运维管理员**：As a OMC 运维管理员，I want 在管理面**配置多个 OSS Trap 接收端**（host/port/community/version，启用与禁用切换），So that 不同运营商、不同区域、主备 OSS 都能并行接收，且故障 OSS 不影响其他订阅方。

> **告警链路工程师**：As a 告警链路工程师，I want 当 SNMP Trap 推送失败（OSS 不可达 / 鉴权失败 / 编码异常）时**有可观测指标 + 失败日志 + 告警**，So that 链路异常能被发现，并且失败不影响 OMC 内部告警流。

> **运营商安全工程师**：As a 运营商安全工程师，I want SNMPv3 模式支持 **认证 + 加密**（authPriv，SHA + AES），So that 符合电信级安全规范，避免 community 明文泄露风险。

---

## 3. 验收标准（Given/When/Then）

### AC-1：SNMPv2c Trap 基本可用（T-0017 接入后）
```
Given: 已配置一个 OSS Trap 接收端 (host=10.0.0.1, port=162, community=public, version=v2c, enabled=true)
When:  OMC 触发一条 critical 告警（device_serial=D001, alarm_id=POWER_FAIL, occur_time=2026-04-28T10:00:00Z）
Then:  - SNMP Trap 在 5 秒内发送至 10.0.0.1:162
       - PDU 含基础 OID（OMC 私有企业号 + 告警 sub-tree）
       - 变量绑定（VarBind）含：alarm_identifier / severity / device_serial / occur_time / carrier
       - Prometheus 指标 trap_send_total{result="success"} +1
       - trap_send_duration_seconds 直方图记录耗时
```

### AC-2：多 OSS 并行推送 + 故障隔离
```
Given: 配置三个接收端 osr-cmcc / osr-ctcc / osr-internal，其中 osr-ctcc 不可达
When:  触发一条告警
Then:  - osr-cmcc / osr-internal 推送成功（trap_send_total +2）
       - osr-ctcc 推送失败（trap_send_failures_total{target="osr-ctcc"} +1）
       - osr-ctcc 失败不阻塞其他两个推送（异步并发或独立 goroutine）
       - 失败日志含 target_id / host / 错误原因（network unreachable）
```

### AC-3：禁用接收端不发送 + 配置热加载
```
Given: 接收端 osr-test 配置 enabled=false
When:  触发告警
Then:  - 不向 osr-test 发送 Trap
       - trap_send_total 不计数
When:  通过 API 把 enabled 改为 true
Then:  下一条告警发送至 osr-test，无需重启进程
```

### AC-4：SNMPv3 authPriv 加密通道（可选高级特性）
```
Given: 配置接收端 version=v3, username=omc-user, auth_protocol=SHA, auth_password=***, priv_protocol=AES, priv_password=***
When:  触发告警
Then:  - Trap 通过 USM 鉴权 + AES 加密发送
       - OSS 侧能正确解密并解析（联调验证）
       - auth_password / priv_password 在日志、API 响应、Prometheus label 中**永不出现**
```

> 注：AC-4 在骨架阶段（T-0013）仅做接口预留，不验证加密链路；T-0017 联调时执行。

---

## 4. 运营商差异矩阵

| 维度 | CMCC 中国移动 | CTCC 中国电信 | CUCC 中国联通 | 备注 |
|------|---------------|---------------|---------------|------|
| Trap 版本 | SNMPv2c（默认）+ SNMPv3 可选 | SNMPv3 强制（authPriv） | SNMPv2c（默认） | 联通规范允许 v2c，电信安全要求高 |
| 私有企业 OID | 待 CMCC 分配（占位 `1.3.6.1.4.1.99999.1.X`） | 待 CTCC 分配（占位 `1.3.6.1.4.1.99999.2.X`） | 待 CUCC 分配（占位 `1.3.6.1.4.1.99999.3.X`） | 入网测试时由运营商指定，骨架用占位常量 + TODO 注释 |
| MIB 变量字段顺序 | 设备 SN → 告警码 → 级别 → 时间 → 附加 | 告警码 → 级别 → 设备 SN → 时间 → 关联工单号 | 设备 SN → 告警码 → 级别 → 时间 → 类型 | 通过 `Carrier.MapAlarmToTrapPDU(alarm)` 适配点封装，禁止业务层 if-else |
| 告警级别编码 | 1=critical / 2=major / 3=minor / 4=warning | critical / major / minor / cleared（字符串）| 0=clear / 1=indeterminate / 2=critical / 3=major / 4=minor / 5=warning | 由 Carrier 适配器映射 |
| 时间戳格式 | Unix 秒（OctetString） | ISO8601（OctetString） | Unix 毫秒（Counter64） | 由 Carrier 适配器编码 |
| Community 命名规范 | `omc-cmcc-{region}` | 不使用 community（v3） | `OMC_CUCC` | 配置项允许任意字符串，仅约束建议命名 |
| 重试策略 | 失败重试 3 次，间隔 5/10/20 秒 | 失败重试 3 次 + Failed Trap 入持久化 outbox | 失败不重试，由 OSS 主动轮询补 | T-0020 推送可靠性整链时实现，骨架阶段只暴露指标 |
| 心跳 Trap | 每 5 分钟 heartbeatTrap | 不需要 | 每 10 分钟 heartbeatTrap | 非骨架范围，列入 T-0017 |

**关键约束**：所有差异必须通过 `internal/carrier/*` 适配器实现。骨架阶段（T-0013）**仅实现 cmcc 适配点的 stub** 并提供 Carrier 接口预留方法签名，ctcc/cucc 留 TODO 给后续 PR；**禁止** `if carrier == "cmcc"` 类硬编码。

---

## 5. 非目标（Non-Goals）

- ✂️ **不做** SNMP Get/GetNext/GetBulk/Set 主动查询能力（OMC 只发不收，OSS 不在 OMC 上做 SNMP 数据轮询）
- ✂️ **不做** SNMP Inform（带确认的 trap，需要 OSS 端响应）— 本期仅 Trap（无确认）；如运营商联调要求再补
- ✂️ **不做** Trap 转发到内部其他系统（不替代 NATS / EventBus）
- ✂️ **不做** MIB 文件管理 UI（MIB 由运营商提供，OMC 仅用 OID 常量）
- ✂️ **骨架阶段（T-0013）不做**：
  - 不接 alarm.Engine（T-0017 接入）
  - 不接 NATS subject `oss.alarm.forward`（T-0017 接入）
  - 不接 router / DI / cmd/app/provider（T-0017 接入）
  - 不写迁移文件（T-0017 创建 `snmp_trap_targets` 表）
  - 不实现持久化 outbox（T-0020 与 push 模块统一抽象）
  - 不实现重试策略（T-0020）
  - 不实现 Carrier 接口实质改造（仅 PRD 描述）

---

## 6. 依赖

### 阻塞项（必须先解决）

骨架阶段（T-0013）**无外部阻塞**——本任务故意做成"独立可编译的隔离包"。

后续阶段：
- [ ] T-0017 接入前需完成：`internal/carrier/Carrier` 接口扩展 `MapAlarmToTrapPDU`（Owner：架构 + 电信业务专家，预计 sprint-04）
- [ ] T-0017 接入前需完成：`migrations/000NNN_add_snmp_trap_targets.sql`（Owner：数据存储专家）
- [ ] T-0020 整链可靠性需先做：HTTP Webhook outbox 抽象上提到 `northbound/common/outbox`（Owner：架构）

### 被阻塞项（本功能不完成会影响什么）
- T-0017 F08 OSS 联调（直接依赖本骨架）
- T-0020 推送可靠性整链（依赖 T-0017）
- 现网入网测试 / 运营商联调（依赖 T-0017+T-0020 完成）
- RC 发布门槛（无 SNMP Trap = 无法过 release-gate F08 章节）

### 外部依赖
- **第三方库**：`github.com/gosnmp/gosnmp`（v1.x，BSD-2 licensed，社区活跃，是 Go 生态 SNMP 标准实现）
- **运营商联调环境**：T-0017 阶段需要三家运营商提供测试 OSS 接收端
- **运营商私有 OID 分配**：T-0017 阶段需运营商正式分配企业号 sub-tree（骨架阶段用占位 `1.3.6.1.4.1.99999`）

---

## 7. 度量（如何证明上线成功）

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| `trap_send_total{target,result}` | N/A（新增） | success 占比 ≥ 99.5% | Prometheus counter，按 target/result(success\|failure) 维度 |
| `trap_send_failures_total{target,reason}` | N/A | < 0.5% / 24h | Prometheus counter，按失败原因（network/encode/auth）分类 |
| `trap_send_duration_seconds{target}` | N/A | P99 < 1s（局域网内 OSS） | Prometheus histogram，bucket: 0.01/0.05/0.1/0.5/1/5s |
| `trap_targets_active` | N/A | = 配置数 | Prometheus gauge |
| 告警 → Trap E2E 时延 | N/A | P99 < 5s | E2E 测试脚本 + 时间戳对比 |

**反例监控**：
- 不应出现：goroutine 泄漏（每次 Trap 发送应有完整生命周期）
- 不应出现：失败的目标拖慢成功的目标（独立 goroutine + timeout）
- 不应出现：community / auth_password 出现在日志或 metrics label
- 不应出现：进程因单个 OSS 不可达而崩溃

---

## 8. 实施要点（非规范性，供参考）

### 阶段拆分

| 阶段 | 任务 | 范围 |
|------|------|------|
| T-0013（本 PRD） | 骨架代码 | 独立可编译 + 单测可绿，**不接生产** |
| T-0017 | 真接入 | alarm.Engine 订阅 + Carrier 改造 + 迁移 + DI + router |
| T-0020 | 可靠性 | 重试 + outbox 持久化 + 熔断 + 告警 |

### 涉及模块（最终全量，非骨架阶段）
- `internal/northbound/snmp/`（**本骨架新建**）
- `internal/alarm/`（T-0017 阶段订阅 alarm.Engine 输出）
- `internal/carrier/`（T-0017 阶段加 `MapAlarmToTrapPDU` 方法）
- `internal/core/event/`（T-0017 阶段订阅 NATS subject `oss.alarm.forward`）
- `migrations/000NNN_add_snmp_trap_targets.sql`（T-0017 阶段建表）
- `cmd/app/provider/snmp.go`（T-0017 阶段 DI）
- `cmd/app/router/router.go`（T-0017 阶段挂 admin API）

### 预计新增端点（非骨架阶段）
- `GET    /api/v1/oss/snmp-targets`        — 列表
- `POST   /api/v1/oss/snmp-targets`        — 新建
- `GET    /api/v1/oss/snmp-targets/:id`    — 详情
- `PUT    /api/v1/oss/snmp-targets/:id`    — 更新
- `DELETE /api/v1/oss/snmp-targets/:id`    — 删除
- `POST   /api/v1/oss/snmp-targets/:id/test` — 发送测试 Trap

### 预计工作量
- T-0013（骨架）：S（1 人日）— 本 PRD 范围
- T-0017（接入）：M（3 人日）
- T-0020（可靠性）：M（3-5 人日）

---

## 9. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | Claude AI（PM 角色） | 2026-04-28 | 待人工审批 |
| 架构师 | 待审批 | | |
| 电信业务专家 | 待审批 | | |
| QA/发布经理 | 待审批 | | |

---

## 10. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-28 | v1.0 | 初稿 — T-0013 骨架阶段 PRD + 设计备忘 | Claude AI |

---

## 设计备忘（S2 设计 — T-0013 骨架阶段）

> 本节为本 PRD 关联的 S2 设计输出。骨架阶段交付物 = PRD（本文档）+ 设计备忘（本节）+ 骨架代码（`internal/northbound/snmp/`）。

### D1. 数据模型 — `snmp_trap_targets` 表（T-0017 阶段才迁移建表）

```sql
-- migrations/000NNN_add_snmp_trap_targets.sql （T-0017 阶段创建，本 PRD 不写）
CREATE TABLE snmp_trap_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    oss_name        VARCHAR(64)  NOT NULL UNIQUE,    -- 例：osr-cmcc-beijing
    description     VARCHAR(255),
    host            VARCHAR(255) NOT NULL,           -- IPv4/IPv6/FQDN
    port            INTEGER      NOT NULL DEFAULT 162,
    version         VARCHAR(8)   NOT NULL,           -- 'v2c' | 'v3'
    community       VARCHAR(128),                    -- v2c only, ENCRYPTED at rest（应用层 AES）
    username        VARCHAR(64),                     -- v3 only
    auth_protocol   VARCHAR(8),                      -- v3: 'MD5' | 'SHA' | 'SHA256'
    auth_password   TEXT,                            -- v3 only, ENCRYPTED at rest
    priv_protocol   VARCHAR(8),                      -- v3: 'DES' | 'AES' | 'AES256'
    priv_password   TEXT,                            -- v3 only, ENCRYPTED at rest
    carrier         VARCHAR(8),                      -- 'cmcc' | 'ctcc' | 'cucc'，可选用于过滤
    timeout_ms      INTEGER      NOT NULL DEFAULT 3000,
    retries         INTEGER      NOT NULL DEFAULT 0, -- 骨架阶段 0；T-0020 引入策略
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snmp_trap_targets_enabled ON snmp_trap_targets (enabled) WHERE enabled = TRUE;
CREATE INDEX idx_snmp_trap_targets_carrier ON snmp_trap_targets (carrier);
```

骨架阶段（本 PRD）**不写迁移**，仅在内存 map 维护一份等价 Go struct（`TrapTarget`，定义见下文 `types.go`）。

### D2. 核心接口签名

```go
// internal/northbound/snmp/types.go
type SNMPVersion string
const (
    VersionV2c SNMPVersion = "v2c"
    VersionV3  SNMPVersion = "v3"
)

type TrapTarget struct {
    ID            string
    OSSName       string
    Host          string
    Port          uint16
    Version       SNMPVersion
    Community     string  // v2c
    Username      string  // v3
    AuthProtocol  string  // v3: MD5/SHA/SHA256
    AuthPassword  string  // v3, never logged
    PrivProtocol  string  // v3: DES/AES/AES256
    PrivPassword  string  // v3, never logged
    Carrier       string  // cmcc/ctcc/cucc，可选
    TimeoutMs     int
    Enabled       bool
}

type Variable struct {
    OID   string  // ASN.1 OID 字符串，如 "1.3.6.1.4.1.99999.1.1.1"
    Type  string  // OctetString | Integer | Counter32 | Counter64 | TimeTicks
    Value any
}

// internal/northbound/snmp/sender.go
type Sender interface {
    Send(ctx context.Context, target *TrapTarget, vars []Variable) error
}

// internal/northbound/snmp/registry.go
type TargetRegistry interface {
    Add(t *TrapTarget) error
    Remove(id string) bool
    Get(id string) (*TrapTarget, bool)
    List() []*TrapTarget
    ListEnabled() []*TrapTarget
}

// internal/northbound/snmp/engine.go
type Engine struct { ... }
func NewEngine(sender Sender, registry TargetRegistry, logger *zap.Logger) *Engine
// Process 是骨架阶段唯一对外方法。T-0017 阶段由 alarm.Engine 调用。
func (e *Engine) Process(ctx context.Context, alarm *AlarmEvent) []SendResult
```

### D3. Trap PDU 编码

**OID 规划（占位 — T-0017 阶段由运营商分配正式 OID 后替换）**：

```go
const OMC_PRIVATE_OID = "1.3.6.1.4.1.99999"  // TODO: 待运营商正式分配企业号

// 告警 Trap sub-tree
var (
    OIDAlarmIdentifier = OMC_PRIVATE_OID + ".1.1.1"  // 告警唯一标识
    OIDAlarmSeverity   = OMC_PRIVATE_OID + ".1.1.2"  // 严重级别
    OIDDeviceSerial    = OMC_PRIVATE_OID + ".1.1.3"  // 设备 SN
    OIDOccurTime       = OMC_PRIVATE_OID + ".1.1.4"  // 发生时间
    OIDAlarmType       = OMC_PRIVATE_OID + ".1.1.5"  // 告警类型
    OIDCarrierTag      = OMC_PRIVATE_OID + ".1.1.6"  // 运营商标签
)
```

PDU 类型：SNMPv2c TRAP-V2 PDU（避免 v1 已废弃的 enterprise-specific trap）。

### D4. Carrier 适配点（PRD 描述，T-0017 才实改 Carrier 接口）

```go
// internal/carrier/carrier.go （T-0017 阶段扩展，本骨架不动）
type Carrier interface {
    // ...现有方法...

    // MapAlarmToTrapPDU 把告警事件映射为 SNMP Trap 变量绑定列表。
    // 各运营商实现按各自规范决定字段顺序、字段编码、级别映射。
    MapAlarmToTrapPDU(alarm *AlarmEvent) ([]snmp.Variable, error)
}
```

骨架阶段（本 PRD）**不动 Carrier 接口**，由 `engine.go` 内置一个 default mapper（按 CMCC 字段顺序）作为占位；T-0017 阶段由 cmcc/ctcc/cucc 三家适配器分别实现并替换 default mapper。

### D5. 包依赖图（隔离性约束）

```
internal/northbound/snmp/  (骨架)
  ├── stdlib (context, time, errors, sync, fmt)
  ├── github.com/gosnmp/gosnmp
  ├── go.uber.org/zap
  └── github.com/google/uuid (id 生成)
```

**严禁**依赖：`internal/alarm`、`internal/carrier`、`internal/core/event`、`internal/core/appconfig`、`internal/northbound/push`、`cmd/`。

骨架阶段定义的 `AlarmEvent` 类型为本包私有 stub，T-0017 接入时再统一为跨模块共享类型（在 `internal/core/model/` 或 `internal/alarm/`）。

### D6. 测试覆盖（骨架阶段，≥ 5 测例）

1. `TestTargetRegistry_AddRemoveListGet` — Registry 基础 CRUD
2. `TestTargetRegistry_ListEnabled_OnlyEnabled` — enabled 过滤
3. `TestSender_BuildVarBinds_DefaultMapper` — 默认 mapper 字段映射正确性（table-driven）
4. `TestSender_Send_MockSuccess` — mock sender 成功路径
5. `TestSender_Send_MockError` — mock sender 错误传递
6. `TestEngine_Process_MultiTarget` — 多 target 全部投递、独立失败不影响其他
7. `TestEngine_Process_SkipDisabled` — 禁用 target 不发送
8. `TestEngine_Process_NoTarget` — 无 target 时不报错（degenerate case）

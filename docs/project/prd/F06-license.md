# PRD: F06 License 子系统（统一需求文档）

**PRD ID**：F06-license
**功能域**：F06 OMC-R 核心 / RBAC + License
**作者**：Claude（代 Owner = 电信业务专家 + 安全合规专家）
**创建**：2026-05-09（由 F06-license-enforcement + F06-license-management 合并而来）
**状态**：执行层 done（T-0033）+ 治理层 draft（待登记 T-NNNN）

> **本文档定位**：F06 license 子系统的**唯一需求源**，覆盖运行时执行层（已实施）与管理治理层（待实施）。
> 历史 PRD [F06-license-enforcement.md](./F06-license-enforcement.md) 保留作为执行层落地依据；F06-license-management.md（治理层 v0.2 draft）已合并入本文并删除。

---

## 1. 概述

OMC 是商用网管系统，遵循 OEM 授权模式。License 子系统两层职责正交：

| 层 | 职责 | 触发方 | 落地状态 |
|---|---|---|---|
| **执行层** | 运行时拦截：容量满拒新增、过期拒写入、阈值告警 cron | 业务调用（device.Create 等）+ 定时器 | ✅ done（commit T-0033） |
| **治理层** | 管理 UI：导入 / 激活 / 吊销 / 导出 / 审计日志 / 合规归档 | 管理员在三页面操作 | 📋 draft |

两层共享同一数据源（`licenses` 表），治理层新增 `license_logs` 表承载审计。

---

## 2. 范围与边界

### 2.1 范围（仅 OMC 侧）

OEM 颁发给运营商、用于授权"该 OMC 实例本身"的凭证。受控四维度：

| 维度 | 含义 |
|---|---|
| **容量授权** | 限制 OMC 可纳管的设备数（防超卖）|
| **特性授权** | 分级解锁 OMC 功能模块（北向推送 / 自动开站 / 高级 KPI / 报表导出 等）|
| **时效** | `perpetual` 永久 / `subscription` 订阅 / `trial` 试用 / `evaluation` 评估 + 宽限期 `grace_period_days` |
| **状态机** | `pending → active → expired/revoked` |

### 2.2 不在范围

| 项 | 不做的理由 |
|---|---|
| 设备侧许可证（基站 NVRAM 容量 / 用户数 / 频段授权）| 属设备参数，OMC 仅作"参数读取展示"消费方，归属设备参数功能（`device_param_handler`）|
| license 颁发流程（OEM 端生成 / 签名 / 分发）| 全部发生在 OMC 之外；本子系统仅消费"已颁发的 license 文件" |
| license 续期自助流程 | 线下走 OEM；UI 仅提供"导入新 license + Activate"通道 |
| 跨运营商分别授权 | OEM 颁发时已决定 |
| 复杂 license 文件离线解析格式 | 已有 Import API 走 JSON 输入 |
| 跨业务模块写操作拦截扩展 | 第一版仅 `device.CreateDevice`，后续 PR 按需扩展 |

### 2.3 OMC 侧 vs 设备侧（一句话区分）

- **设备侧** = 基站自身能服务多少用户 / 多少容量（设备 NVRAM）
- **OMC 侧** = 网管能管多少基站 / 能用哪些 OMC 功能（`licenses` 表）
- 本文档只关心后者

---

## 3. 用户故事

| 角色 | 故事 |
|---|---|
| OEM 销售方 | 客户实际部署设备数 ≤ license 授权数，超过时系统自动拒绝 |
| OEM 销售方 | 系统在 LicenseLogs 自动记录每次容量阈值告警与 enforcement 拦截，作为客户超卖证据 + 计费依据 |
| OEM 售后 | 远程协助运营商在 UI 上自助导入 / 激活 license，不再被无 UI 的客户电话淹没 |
| 运营商运维 | 从 LicenseList 一眼看出当前容量使用率、距过期天数、特性清单 |
| 运营商运维 | license 即将过期（30 / 7 / 1 天）时收到告警，提前续期 |
| 系统管理员 | 仪表盘看到当前 license 容量使用率与剩余天数，量化授权状态 |
| 系统管理员 | 从 LicenseLogs 追溯"谁在 X 时间导入了 / 吊销了哪条 license"，满足等保 2.0 重要操作日志合规 |
| 审计员 | 从 LicenseLogs 查询 enforcement 拦截记录（capacity_exceeded / license_expired），复盘超卖事件 + 准备季度合规报告 |
| 开发 / 测试人员 | 本地 dev 环境无 license 时不被拦截（默认放行 + warning 日志）|

---

## 4. 系统架构

```
┌────────────────────────────────────────────────────┐
│ 治理层（管理 UI + 审计）         management（待实施）│
│   /license/list  /license/operations  /license/logs│
│   handler.go + service.go + LogWriter              │
└──────────────┬────────────────────────────────────┘
               │ CRUD + 写 license_logs
               ▼
┌────────────────────────────────────────────────────┐
│ 数据层：licenses 表（已存在 21 列）                  │
│         license_logs 表（待建）                     │
└──────────────▲────────────────────────────────────┘
               │ 读 active license + 写 license_logs
┌──────────────┴────────────────────────────────────┐
│ 执行层（运行时 + cron）         enforcement（已实施）│
│   Enforcer.EnforceCapacity / EnforceExpiry         │
│   Monitor 容量 cron / 过期 cron                     │
└────────────────────────────────────────────────────┘
```

**模块归属**：
- 已实施：`internal/license/{service,enforcer,monitor,handler,repository,metrics}.go`
- 待实施：`internal/license/{license_log_repo,license_log_service}.go`（新建）+ `migrations/000NNN_license_logs.sql`（新建）+ `omcmb/webcode/src/pages/license/`（页面增强）

---

## 5. 功能详细需求

### 5.1 执行层（已实施）

#### 5.1.1 容量拦截

`POST /api/v1/devices` 创建前检查 active license 的 `MaxDevices` vs `UsedDevices`：
- 未满 → 放行
- 已满 → 返回 **403** + 错误码 **9101** `license_capacity_exceeded`
- Prometheus `license_enforcement_total{operation="device.create",result="denied_capacity"}` +1

#### 5.1.2 过期拦截

写操作前检查 active license 状态：
- 类型 `perpetual` → 跳过过期校验
- 已 `expired`（cron 自动转）+ 非 `perpetual` → 返回 **403** + 错误码 **9102** `license_expired`
- GET 类操作（dashboard / report / alarm 查询）**不受影响**

#### 5.1.3 过期阈值告警（daily cron）

| 距过期 | severity | alarm_identifier |
|---|---|---|
| 30 天 | warning | `license_expiring_30d` |
| 7 天 | major | `license_expiring_7d` |
| 1 天 | critical | `license_expiring_1d` |

#### 5.1.4 容量阈值告警（hourly cron）

- 三档阈值：80% / 90% / 95%
- 6 小时去重（`last_capacity_alert_at` + `last_capacity_alert_threshold`）
- 跨档（如从 80% 跨到 90%）立即发新告警，不等 6h

#### 5.1.5 自动过期（daily cron）

`expiry_date < now() - grace_period_days` 的 active license 自动转 `status=expired` + 写 system_logs。

#### 5.1.6 无 active license 默认放行

数据库无任何 active license 时不拦截，仅启动 warning 日志 + `license_active_count=0` metric。

**理由**：dev 环境友好；prod 部署 OEM Import license 后自动开始拦截。

#### 5.1.7 接口契约

```go
// internal/license/enforcer.go
type Enforcer interface {
    EnforceCapacity(ctx context.Context, additional int) error
    EnforceExpiry(ctx context.Context, operation string) error
    ActiveLicense(ctx context.Context) (*License, error)
    Quota(ctx context.Context) (*Quota, error)
}

type Quota struct {
    MaxDevices       int
    UsedDevices      int
    UsageRatio       float64
    DaysRemaining    int    // -1 = perpetual / no expiry
    LicenseType      string
    GracePeriodDays  int
    HasActiveLicense bool
}

// internal/device/device_service.go 消费侧（小接口）
type LicenseEnforcer interface {
    EnforceCapacity(ctx context.Context, additional int) error
    EnforceExpiry(ctx context.Context, operation string) error
}
```

DeviceService.CreateDevice 在 ctx check existing 之后、写入 DB 之前调：
```go
if s.licenseEnforcer != nil {
    if err := s.licenseEnforcer.EnforceExpiry(ctx, "device.create"); err != nil {
        return nil, err
    }
    if err := s.licenseEnforcer.EnforceCapacity(ctx, 1); err != nil {
        return nil, err
    }
}
```

#### 5.1.8 多 active license 决策

`GetActiveLicenseWithMaxDevices()` 返回 active 中 `MaxDevices` 最大的一条作为 enforcement 依据。其他 active license 仅供查询，不参与拦截决策（避免歧义）。

#### 5.1.9 缓存策略

- active license 进程内缓存（`sync.RWMutex` + 5 min TTL）
- Import / Activate / Revoke 时主动 Invalidate
- 容量计数实时查 DB（`SELECT COUNT(*) FROM devices` 索引下 < 1ms）

---

### 5.2 治理层 — LicenseList 列表（待实施）

入口：`/license/list` 主页面。

#### 5.2.1 列表能力

| 列 | 字段 | 说明 |
|---|---|---|
| 名称 | license_name | 主标识 |
| 编码 | license_code | UNIQUE，业务主键 |
| 产品 | product_name | OEM 产品线 |
| 类型 | license_type | subscription / perpetual / trial / evaluation |
| 状态 | status | Tag 颜色：active 绿 / pending 橙 / expired 红 / revoked 灰 / trial 蓝 |
| 容量使用 | used / max | Progress 条；≥80% 黄、≥90% 红 |
| 到期 | expiry_date | 距今天数；< 30 天黄、< 7 天红、过期红粗体 |
| 区域 | region | 选填 |
| 操作 | — | 详情 / 吊销（仅 active + 需 license-operate 权限）|

#### 5.2.2 过滤器（FilterBar）

状态（multi）/ 类型（multi）/ 设备类型（select）/ 区域（input）/ 名称 编码（keyword）。

#### 5.2.3 详情抽屉

```
┌─ 许可证 OMC-BASIC-HB-2024-001 ─────────────────┐
│ [基本信息]  [容量信息]  [功能特性]  [审计记录]  │
├──────────────────────────────────────────────┤
│ 名称：HB 基础版                               │
│ 状态：active  距过期：87 天                    │
│ 类型：subscription  授权方：OEM-XYZ            │
│ 设备类型：eNB  区域：华北                      │
│                                              │
│ 容量使用：85 / 100  [████████▒▒]  85%          │
│ 阈值告警：80% (已触发) / 90% / 95%             │
│ 上次告警：2026-05-01 12:34（80% 阈值）         │
│                                              │
│ 启用特性：                                     │
│   ✓ pm_export        ✓ alarm_export           │
│   ✓ northbound_push  ✗ auto_provision (未授权) │
│                                              │
│ 最近操作（前 10 条 → 跳转完整审计）：           │
│   2026-05-01 admin 容量告警 80%                │
│   2026-04-30 operator1 查询详情                │
└──────────────────────────────────────────────┘
```

#### 5.2.4 Summary 卡片（页面顶部 4 张统计）

1. **当前 Active license 数**（点击跳转过滤）
2. **总容量使用率**：sum(used_devices) / sum(max_devices)
3. **30 天内过期 license 数**（点击跳转过滤）
4. **enforcement 命中次数（近 7 天）**

数据源：`GET /licenses/summary` + `GET /licenses/quota`。

---

### 5.3 治理层 — LicenseOperations 操作（待实施）

入口：`/license/operations`，4 Tab。

#### 5.3.1 Tab 1：导入

- **方式 A**：上传 license 文件（`.lic` / `.json` / `.txt`）→ 后端解析 + 验证签名 + 入库
- **方式 B**：粘贴 license 字符串到 textarea
- **校验链**：
  1. 文件解析成功（JSON / 自定义二进制）
  2. 数字签名验证（**Q4=B MVP 放过 + warning**）：
     - 当前阶段：未签名 / 签名无效 / 公钥未配置 → 仍允许 import 入库，但写一条 zap.Warn 日志（含 `license_code` + 原因）+ 在 import 响应里返回 `signature_status: 'unverified' | 'invalid' | 'verified'` 字段供前端 Modal 显示警告 Tag
     - **P4-C 已实现**（commit `bddad147` / `d0cc4a0b`）：`internal/license/signature.go` `SignatureVerifier` 线程安全 keys map + `LoadKeysFromDir(.pem 文件，PKIX/PKCS1 双格式)` + `VerifyLicenseJSON` (RSA-PSS-SaltLen32 + canonicalize) + strict 模式 → handler 拒绝；`appconfig.LicenseConfig.Signing.{PublicKeyDir,Strict}` + DI 接线在 `cmd/app/provider/modules.go`
     - **P5-a W4 守卫**（commit `bcc1212c`）：`strict=true && KeyCount==0` 启动 Fatal，防 prod 误配公钥目录路径错误导致所有 import 静默被拒
  3. license_code 不与已有冲突（UNIQUE）
  4. issue_date / expiry_date 时间合理性
- **API**：`POST /api/v1/licenses/import`（已存在）+ 可选 `signed_license_json` 字段（**P5-c 前端已接**，文件模式 FileReader 读 content → POST，commit `4c3c59f7`）
- **响应**：成功 → Modal 显示导入详情 + 签名状态 Tag + "去激活" CTA；失败 → 错误码（**12109 SignatureVerifyFailed** 专属 UI 文案）+ 详细原因

#### 5.3.2 Tab 2：激活

- 输入 `license_code`（从 Tab 1 跳转可自动填）
- 提交时**先查询同 `device_type + region` 是否已有 active license**：
  - 已有 → **弹 Modal 确认**（Q1=B）："当前已有 active license `<old_license_code>`，激活新 license 将自动吊销旧的。是否继续？"
  - 用户点"继续" → 串行调 `POST /:old_id/revoke`（自动 + log_type=`auto_revoke_by_activate`）+ `POST /activate`（log_type=`activate`），两条日志均记 actor_user_id；任一步失败 → 整体回滚（前端 message.error 不静默）
  - 用户点"取消" → 中止激活，前端 toast "已取消"
  - 没有同维度旧 active → 直接 `POST /activate`
- `POST /api/v1/licenses/activate`：状态 `pending → active`
- 反馈：成功 toast + 跳转列表高亮新激活；不存在 → 404；已激活 → noop + "该 license 已是 active"

#### 5.3.3 Tab 3：吊销

- 下拉选择当前 active license（disabled 已 revoked / expired 项）
- 二次确认 Modal："吊销后该 license 立即失效；如当前为唯一 active，将进入"无 license"模式（dev 友好放行 + warning）"
- `POST /api/v1/licenses/:id/revoke`
- 限制：仅 status=active 可被吊销；其他状态返 400

#### 5.3.4 Tab 4：导出

- 单条：从下拉选择 license → 导出 PDF / JSON（含 license 全字段 + 近 30 天审计 + 当前容量使用）
- 全量：导出所有 active license 汇总表 → CSV
- API（**待加**）：`GET /licenses/:id/export?format=pdf|json` + `GET /licenses/export?format=csv`

#### 5.3.5 操作历史（页面底部表格）

显示当前用户**最近 30 天**对所有 license 的操作记录（来自 `license_logs` 表）。详细查询去 LicenseLogs 页。

---

### 5.4 治理层 — LicenseLogs 审计日志（待实施）

入口：`/license/logs`。

#### 5.4.1 日志类型（统一存于 license_logs 表）

| 类型 | 触发场景 | actor | result |
|---|---|---|---|
| `import` | Import API 调用成功 / 失败 | 用户 ID | success / failed |
| `activate` | Activate API 调用 | 用户 ID | success / failed |
| `revoke` | Revoke API 调用 | 用户 ID | success / failed |
| `query_detail` | 查 GetByID（敏感访问审计）| 用户 ID | success |
| `enforcement_capacity` | enforcer.EnforceCapacity 拒绝 | system | denied |
| `enforcement_expiry` | enforcer.EnforceExpiry 拒绝 | system | denied |
| `capacity_alert` | monitor 容量阈值告警 | system | warning |
| `expiry_alert` | monitor 过期阈值告警 | system | warning |
| `auto_expire` | monitor cron 把 expiry_date < now 转 expired | system | success |

#### 5.4.2 列能力

| 列 | 字段 | 说明 |
|---|---|---|
| 时间 | created_at | 默认按时间倒序 |
| 类型 | log_type | Tag 颜色按类型分组 |
| 许可证 | license_id → license_name | 关联跳转 |
| 操作者 | actor_user_id → username | system 显示"系统" |
| 结果 | result | success 绿 / failed 红 / denied 橙 / warning 黄 |
| 详情 | details (jsonb) | 摘要展示，详情 popover 全 JSON |
| 客户端 IP | client_ip | 仅用户操作；system 显示 "—" |

#### 5.4.3 过滤器

时间区间（24h / 7d / 30d / custom）/ 类型（multi）/ 结果（multi）/ license（search）/ 操作者（search）/ details 关键字。

#### 5.4.4 导出

选中行 / 当前过滤结果 → CSV / JSON（合规审计材料）。

#### 5.4.5 保留策略

- DB 默认保留 **6 个月**（满足等保 2.0 三级 8.1.4.7）
- 6 个月以上自动归档到 MinIO `license-logs/{YYYY-MM}.jsonl.gz`
- DB 清理 cron 周级跑一次

---

## 6. 数据模型

### 6.1 licenses 表（已存在）

DDL 见 [`migrations/000043_licenses_capacity_expiry.sql`](../../../omcgo/migrations/000043_licenses_capacity_expiry.sql)，21 列含：

- `license_code` UNIQUE / `license_name` / `product_name` / `license_type` / `region`
- `max_devices` / `used_devices` / `features` JSONB
- `issue_date` / `expiry_date`
- `status` (`pending` / `active` / `expired` / `revoked`)
- `grace_period_days` (default 0, ≤ 365)
- `capacity_alert_thresholds` JSONB (default `[80, 90, 95]`)
- `last_capacity_alert_at` / `last_capacity_alert_threshold`

索引：
- `idx_licenses_active_status (status) WHERE status='active'`
- `idx_licenses_expiry_date WHERE expiry_date IS NOT NULL`

**本文档不改 schema**。

### 6.2 license_logs 表（待建）

```sql
-- migrations/000NNN_license_logs.sql（版本号建迁移时实时确定）
-- +goose Up
CREATE TABLE IF NOT EXISTS license_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id      UUID REFERENCES licenses(id) ON DELETE SET NULL,
    log_type        VARCHAR(32) NOT NULL,
    -- 取值：import / activate / revoke / query_detail / enforcement_capacity /
    --       enforcement_expiry / capacity_alert / expiry_alert / auto_expire
    actor_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    -- NULL = system 操作（cron / enforcer 触发）
    result          VARCHAR(16) NOT NULL,
    -- success / failed / denied / warning
    details         JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- summary（一句话）+ context（device_sn / threshold / err_msg 等）
    client_ip       INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_license_logs_license_id ON license_logs(license_id);
CREATE INDEX idx_license_logs_created_at ON license_logs(created_at DESC);
CREATE INDEX idx_license_logs_log_type   ON license_logs(log_type);
CREATE INDEX idx_license_logs_actor      ON license_logs(actor_user_id) WHERE actor_user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_license_logs_actor;
DROP INDEX IF EXISTS idx_license_logs_log_type;
DROP INDEX IF EXISTS idx_license_logs_created_at;
DROP INDEX IF EXISTS idx_license_logs_license_id;
DROP TABLE IF EXISTS license_logs;
```

### 6.3 enforcement / cron 增量列（已存在）

`migrations/000043_licenses_capacity_expiry.sql` 已加：
```sql
ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS grace_period_days INT NOT NULL DEFAULT 0
        CHECK (grace_period_days >= 0 AND grace_period_days <= 365),
    ADD COLUMN IF NOT EXISTS capacity_alert_thresholds JSONB NOT NULL
        DEFAULT '[80, 90, 95]'::jsonb,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_threshold INT;
```

### 6.4 license_logs 写入约定

- **用户触发**：actor_user_id = ctx 内 user_id；client_ip / user_agent 从 gin.Context 取
- **system 触发**：actor_user_id = NULL
- **enforcement / alert** 写入由 enforcer / monitor 内部直接调（不经 handler）

---

## 7. API 设计

### 7.1 已实现（不动）

| 端点 | 方法 | 说明 |
|---|---|---|
| `/licenses` | GET | 列表（含分页 / 过滤） |
| `/licenses/:id` | GET | 详情 |
| `/licenses/summary` | GET | Summary 卡片数据 |
| `/licenses/quota` | GET | 当前 enforcement 状态 (`max/used/usage_ratio/days_remaining/license_type/grace/has_active`) |
| `/licenses/activate` | POST | 激活（body: `{license_code}`） |
| `/licenses/import` | POST | 导入（body: 完整 License 字段） |
| `/licenses/:id/revoke` | POST | 吊销 |

### 7.2 待加（治理层）

| 端点 | 方法 | 说明 |
|---|---|---|
| `/licenses/:id/logs` | GET | 单条 license 的审计日志（详情抽屉用，最近 N 条） |
| `/licenses/logs` | GET | 全量审计日志（分页 / 过滤，LicenseLogs 主页用） |
| `/licenses/:id/export` | GET | 单条导出（query: `format=pdf|json`） |
| `/licenses/export` | GET | 全量导出（query: `format=csv`，含过滤） |

---

## 8. 权限模型

### 8.1 三层权限点

| 权限 key | 端点范围 | 默认角色 |
|---|---|---|
| `system:license:view` | GET 所有 license + logs（只读） | viewer / operator / admin / super_admin |
| `system:license:operate` | POST import / activate / revoke / export | super_admin（可选 license-admin，参 Q3） |
| `system:license:audit` | GET /licenses/logs（全量审计） | super_admin（可选 auditor，参 Q3） |

### 8.2 独立编辑权限（同北向编辑模式 commit `b1cdea13`）

- 新加 menus button 节点 `permission_key='system:license:operate'`，挂在 `/license/operations` 父菜单下
- seed 给 super_admin 绑定该 button
- 前端 `usePermission('system:license:operate')` 控制 4 Tab 提交按钮 `disabled` + Tooltip
- 后端 `RequireAPIPermission` 中间件兜底（即使前端绕过直接调 PUT，仍 403）

### 8.3 错误码

| 代码 | sentinel | HTTP | 触发 |
|---|---|---|---|
| 9101 | `ErrLicenseCapacityExceeded` | 403 | EnforceCapacity 拒绝 |
| 9102 | `ErrLicenseExpired` | 403 | EnforceExpiry 拒绝 |

`internal/core/errors/errors.go` 已注册；`HTTPStatusFromError` 映射到 403。

---

## 9. 业务规则

### 9.1 状态机

```
[Import API]
     ↓
  pending ──[Activate API]──→ active ──[过期 cron]──→ expired
                              ↓ ↑
                     [Revoke API]
                              ↓
                          revoked
```

### 9.2 操作约束

- 同 `device_type + region` 维度**最多一个** `active` license（业务约束，非 DB UNIQUE）
- 激活时自动 revoke 同维度旧 active（写两条 log：`auto_revoke_by_activate` + `activate`）
- `trial` / `evaluation` 到期转 `expired`，**不可续期**
- `subscription` 到期可通过 Import 新 license + Activate 续期
- `revoke active` → 立即失效；如全网无 active → 进入"无 license"模式（dev 放行 / prod critical 告警）
- `import` 已存在 license_code → **409 Conflict**
- `activate` 不存在 license_code → **404**
- `revoke` 非 active → **400 Bad Request**

### 9.3 enforcement 命中必写日志

`enforcer.EnforceCapacity` / `EnforceExpiry` 在拒绝时**必须**写一条 license_log：
- `log_type = enforcement_capacity` / `enforcement_expiry`
- `result = denied`
- `details` 含 `device_sn` / `operation` / 当前 `used` / `max` / `expiry_date`

这是审计员复盘超卖事件的关键证据，为治理层 P0 阶段必接入点。

---

## 10. 度量与可观测性

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `license_active_count` | gauge | — | 当前 active license 数（=0 时 warning） |
| `license_capacity_used_devices` | gauge | — | 当前 used devices |
| `license_capacity_max_devices` | gauge | — | 最大授权设备数（多 license 取 max） |
| `license_capacity_usage_ratio` | gauge | — | used / max（0-1） |
| `license_expiry_days_remaining` | gauge | `license_id` | 剩余天数（perpetual = -1） |
| `license_enforcement_total` | counter | `operation, result` | result ∈ {`allowed`, `denied_capacity`, `denied_expired`, `no_active_license`} |

---

## 11. 验收标准（Given-When-Then）

### 11.1 执行层（V1-V8，已通过）

#### V1 — 容量未满放行
- **G**: active license `MaxDevices=100, UsedDevices=99`
- **W**: `POST /api/v1/devices` 添加新设备
- **T**: 200 + UsedDevices 更新为 100

#### V2 — 容量超限拦截
- **G**: `MaxDevices=100, UsedDevices=100`
- **W**: 同上
- **T**: **403** + 9101 `license_capacity_exceeded`
- **And**: Prom counter `denied_capacity` +1

#### V3 — 过期拦截写入
- **G**: active license `status=expired` + `LicenseType != perpetual`
- **W**: 同上
- **T**: **403** + 9102 `license_expired`
- **And**: GET 类操作（dashboard / report / alarm 查询）不受影响

#### V4 — 过期前告警
- **G**: active license `expiry_date = now + 30 days`，type=subscription
- **W**: daily cron 触发
- **T**: 写 alarm `severity=warning, identifier="license_expiring_30d"`，字段 `license_id` / `expiry_date` / `days_remaining=30`
- **过期前 7 天**：major + `license_expiring_7d`
- **过期前 1 天**：critical + `license_expiring_1d`

#### V5 — 容量阈值告警去重
- **G**: UsedDevices 触发 80% 阈值，6h 前已发过同阈值告警
- **W**: hourly cron
- **T**: 不重复发；跨过下一阈值（90%）立即发新告警，不等 6h

#### V6 — 永久 license 跳过过期
- **G**: active `LicenseType=perpetual, ExpiryDate=NULL`
- **W**: daily expiry checker
- **T**: 不转 expired，无告警

#### V7 — 无 active license 默认放行
- **G**: 无任何 active license
- **W**: `POST /api/v1/devices`
- **T**: 200（不拦截）+ 启动 warning 日志 + `license_active_count=0` metric

#### V8 — 容量配额查询
- **G**: active `MaxDevices=100, UsedDevices=42`
- **W**: `GET /api/v1/licenses/quota`
- **T**: 200 + JSON `{max_devices:100, used_devices:42, usage_ratio:0.42, days_remaining:N|null, license_type:"...", grace_period_days:0, has_active_license:true}`

### 11.2 治理层（V9-V14，待实施）

#### V9 — LicenseList 容量可视化
- **G**: active `MaxDevices=100, UsedDevices=85`
- **W**: 打开 `/license/list`
- **T**: 该行 Progress 显示 85% 黄（≥80%）；详情抽屉显示 "85 / 100 85%" + "上次告警 80% 阈值 (XX 时间)"

#### V10 — 导入成功 → 激活
- **G**: 上传合法 license 文件
- **W**: 提交 Import 表单
- **T**: Modal 显示 license 详情 + "立即激活" CTA
- **And**: 点击 CTA → Tab 2 自动填 → 一键激活成功
- **And**: license_logs +2 行（import success / activate success）

#### V11 — Logs 过滤 enforcement
- **G**: 数据库已有 5 条 enforcement_capacity 命中日志
- **W**: 打开 `/license/logs` + 类型过滤=`enforcement_capacity`
- **T**: 列表显示 5 条，结果列=denied 橙色 Tag
- **And**: 详情 popover 显示 `details.device_sn` / `details.current_used` / `details.max_devices`

#### V12 — 权限隔离
- **G**: 角色=viewer
- **W**: 访问 `/license/operations`
- **T**: 4 Tab 提交按钮 `disabled` + Tooltip "无权限：请联系管理员申请「许可证操作」权限"
- **And**: 即使前端绕过直接调 `POST /licenses/activate` → 返回 **403**

#### V13 — 等保 2.0 日志保留
- **G**: license_logs 中存在 7 月前的日志条目
- **W**: weekly 归档 cron
- **T**: 7 月前条目从 PostgreSQL 删除
- **And**: MinIO `license-logs/{YYYY-MM}.jsonl.gz` 含被删条目（gzip 压缩 JSONL）

#### V14 — 详情抽屉跳转 Logs
- **G**: LicenseList 详情抽屉打开 license A
- **W**: 点击"查看完整审计"链接
- **T**: 跳转 `/license/logs?license_id={A.id}`
- **And**: LicenseLogs 自动应用 license_id 过滤，列表只显示 A 的日志

---

## 12. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|---|---|---|---|
| License 颁发主体 | OEM 厂商（与运营商解耦） | 同 | 同 |
| 容量计算单位 | 按设备序列号去重 | 同 | 同 |
| 过期策略 | OEM 决定 | 同 | 同 |
| 实际差异 | **无** | **无** | **无** |

**结论**：F06 license 子系统**对运营商透明**，不需要 Carrier 适配点。OEM 颁发 license 时已决定容量与有效期，OMC 仅做执行层。

**未来扩展点**（本任务不做）：若 OEM 将来要求按设备类型（pico / femto / micro）分别授权，可在 `License.Features` JSONB 字段加 `device_type_quota` 子字段，由 Carrier 接口的 `MapDeviceTypeToQuotaKey()` 适配。

---

## 13. 非目标

- ❌ license 数字签名验证（MVP 放过 + warning，GA 前补强；参 Q4）
- ❌ license 文件离线导入解析复杂格式（已有 Import API 走 JSON）
- ❌ 按运营商分别 license（一份 license 跨运营商）
- ❌ license 续期自助流程（线下走 OEM）
- ❌ 接入 template / software upgrade / 批量配置等其它写操作拦截（第一版仅 device.Create）
- ❌ 设备侧许可证管理（基站 NVRAM；属设备参数功能）

---

## 14. 落地状态与路线图

### 14.1 已实施（执行层 — commit T-0033 done）

- 完整 enforcement 引擎：`internal/license/{enforcer,monitor}.go`
- migration 000043 落地
- 6 个 Prometheus 指标
- 错误码 9101 / 9102 + HTTP 403 映射
- 接入 DeviceService.CreateDevice 拦截点
- V1-V8 验收用例全绿
- 测试覆盖率 ≥ 70%
- 关闭 Risk R-103

### 14.2 待实施（治理层）

| 阶段 | 范围 | 工作量 |
|---|---|---|
| **P0** | license_logs 表迁移 + repository + service.LogWriter；enforcer / monitor / handler 三处接入 | 2 天 |
| **P1** | LicenseLogs 主页（后端 GET /licenses/logs + 过滤分页；前端接 API + i18n） | 2 天 |
| **P2** | LicenseList 详情抽屉 + Summary 卡片（后端 GET /licenses/:id/logs + Summary 增强；前端抽屉 + 进度条 + 跳转 Logs） | 2 天 |
| **P3** | LicenseOperations 完整 4 Tab（4 Tab UI + 操作历史接 API + 导入校验链 + 权限点 seed）— Q1/Q3/Q4 已决议 ✅ 无阻塞 | 3 天 |
| **P4** | 导出 + 归档 + 签名（PDF / CSV 导出；归档 cron；签名验证） | 3-4 天 |

**总工作量**：12-13 工作日。P0 → P1 → P2 → P3 严格顺序；P4 各项独立可并行。

---

## 15. Gap 清单

| # | Gap | 严重性 | 解决方案 / 阶段 |
|---|---|---|---|
| 1 | license_logs 表不存在 | **高** — 阻塞所有日志相关功能 | P0：新建迁移 + service.LogWriter + enforcer/monitor/handler 三处接入 |
| 2 | LicenseOperations 操作历史 mock | 高 | P3：接 `/licenses/logs?actor_user_id=current&limit=20` |
| 3 | LicenseLogs 全 mock | 高 | P1：接 `/licenses/logs` + 过滤器 |
| 4 | 导出 PDF / CSV 端点未实现 | 中 | P4：handler.Export + 前端文件下载；MVP 可先 JSON |
| 5 | license 文件签名验证未实现 | 中 — 安全合规风险 | **Q4=B 决议**：P3 阶段 import 不阻断、写 warn + signature_status 字段；P4-C：OEM 公钥配置（configs/oem_public_keys/*.pem）+ 强校验 |
| 6 | 独立"许可证操作"权限点未 seed | 中 | P3：参 commit `b1cdea13` 模式 — menus button (system:license:operate) + role_api_permissions seed；**Q3=C 决议**：不附带新增 license-admin 内置角色，企业按需在 /system/roles 自建 |
| 7 | enforcement 命中无可视化 | 中 | P2：LicenseList 顶部 "近 7 天 enforcement 命中" 卡片 + 跳转 Logs |
| 8 | 日志归档 cron 未实现 | 低 — GA 前必须 | P4：monitor.go 新增 `archiveOldLogs(ctx)` + weekly schedule |
| 9 | ~~`license-admin` / `auditor` 内置角色未建~~ | — | **Q3=C 决议关闭**：不新增内置角色，企业按 /system/roles 自定义；条目作废 |
| 10 | 等保 2.0 6 个月保留期未配置可调 | 低 | sys_configs 加 key `license.log.retention_months`，默认 6 |

---

## 16. 决议（2026-05-09 已拍板，状态 Locked）

| # | 议题 | 决议 | 落地点 |
|---|---|---|---|
| **Q1** | 激活时同维度旧 active 处理 | **B 弹确认 Modal** —— 用户点激活后弹「当前已有 active license XXX，激活新 license 将自动吊销旧的。是否继续？」二次确认；通过后自动 revoke + activate 串行执行，写两条 license_log（auto_revoke_by_activate + activate） | P3 LicenseOperations Tab 2 |
| **Q2** | 无 active license 时是否拦截 | **B 放行 + critical 告警**（与 V7 一致） | 已实施于 P0 enforcer.go |
| **Q3** | 是否新增 `license-admin` / `auditor` 内置角色 | **C 留给 RolePermission 自定义** —— 不新增内置角色；仅 seed `system:license:operate` button menu + role_api_permissions；企业按需在 /system/roles 自建带 license 权限的角色 | P3 权限点 seed |
| **Q4** | 数字签名 MVP 是否必须 | **B MVP 放过 + warning，GA 前补强为 A** —— P3 阶段 import 接受未签名 / 签名无效的 license 但记 warn 日志（包含 license_code + reason）；OEM 公钥配置 + 强校验延后到 P4-C 实施 | P3 import 校验链；GA 前 P4-C 收紧 |

**决议依据**：用户友好 / 避免膨胀 / GA 前补强 / dev 环境友好（参考 PRD §11.1 V7）。

---

## 17. DoD 对齐

- [x] PRD 七要素全 + 运营商差异矩阵填写
- [x] migration 000043 up/down 配对
- [ ] migration `000NNN_license_logs.sql` up/down 配对（P0）
- [x] 执行层 `go build ./...` 通过
- [x] 执行层 `go test -race -count=1 ./internal/license/... ./internal/device/...` 全绿
- [x] 执行层覆盖率 ≥ 70%
- [x] `golangci-lint run ./internal/license/...` 0 error
- [x] V1-V8 在 `e2e_verify.sh` 加 ≥ 1 claim（执行层）
- [ ] V9-V14 在 `e2e_verify.sh` 加 ≥ 1 claim（治理层 P3 完成时）
- [x] 6 个 metric `grep -rn` 都返回 ≥ 1
- [x] `risk-register.md` R-103 状态 Open → Closed
- [ ] `risk-register.md` 新增 R-109（合规审计缺口）→ 治理层落地后 Closed
- [x] backlog T-0015 done（执行层）
- [ ] backlog 新登记治理层任务（待 T-NNNN）

---

## 18. 关联

- **代码**：
  - 后端 service：[`omcgo/internal/license/`](../../../omcgo/internal/license/)
  - 前端三页面：[`omcmb/webcode/src/pages/license/`](../../../omcmb/webcode/src/pages/license/)
  - DeviceService 接入点：[`omcgo/internal/device/device_service.go:961`](../../../omcgo/internal/device/device_service.go#L961) `CreateDevice`
  - 错误码：[`omcgo/internal/core/errors/errors.go`](../../../omcgo/internal/core/errors/errors.go) `ErrLicenseCapacityExceeded` / `ErrLicenseExpired`
- **Migration**：
  - [`migrations/000043_licenses_capacity_expiry.sql`](../../../omcgo/migrations/000043_licenses_capacity_expiry.sql)（已落地）
  - `migrations/000NNN_license_logs.sql`（P0 待建）
- **风险登记**：[`docs/project/risk-register.md`](../risk-register.md) R-103（已闭）+ R-109（治理层合规审计缺口，本任务 P0 关闭）
- **Backlog**：[`docs/project/backlog.md`](../backlog.md) T-0015（done）+ 治理层 T-NNNN（待登记）
- **审计合规依据**：等保 2.0 三级 8.1.4.7（重要操作日志保留 ≥ 6 个月）
- **权限模型参考**：commit `b1cdea13` 北向 OSS 主备编辑权限初始化（独立 button 权限点模式）
- **菜单 button 权限点接入参考**：[`docs/prd/system/menus.md`](../../prd/system/menus.md)
- **内置角色管理**：[`docs/prd/system/roles.md`](../../prd/system/roles.md)

---

## 变更日志

| 版本 | 日期 | 作者 | 备注 |
|---|---|---|---|
| 1.0 | 2026-05-09 | Claude | **合并** F06-license-enforcement.md（v1.0 done）+ F06-license-management.md（v0.2 draft）为统一需求文档，作为 F06 license 子系统的唯一需求源。F06-license-management.md 内容全量并入本文后已删除；F06-license-enforcement.md 保留作为执行层落地依据。 |

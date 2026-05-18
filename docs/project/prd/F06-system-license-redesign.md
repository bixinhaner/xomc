# PRD: F06 System License 重构（singleton 模型）

**PRD ID**：F06-system-license-redesign
**功能域**：F06 OMC-R 核心 / License
**作者**：Claude（代 Owner = 产品经理 + 安全合规专家）
**创建**：2026-05-18
**状态**：draft v1，待用户审批
**取代**：[F06-license.md](./F06-license.md)（当前 multi-license 模型整体废弃）

> **决策记录（2026-05-18）**：
> - 用户调研老 OMC（http://172.21.175.129:8081）System > License 页面后确认：
>   当前 OMC Go 实现的 multi-license + Import/Activate/Revoke 模型与真实业务**不一致**
> - 真实模型是 **system-wide singleton license**：每个 OMC 实例 1 张 license 文件，
>   外部签发系统签发，OMC 上传覆盖
> - **方案 A 完全替代**：废弃当前 licenses 表 + LicenseList / Operations / Logs 全套，
>   新建 system_license + system_license_history 双表
> - **现有 licenses 表数据 DROP 不要**（fresh start）

---

## 1. 概述

OMC 是商用网管系统，license 是 OEM 厂商发给运营商客户、用于解锁该 OMC 实例功能与容量的授权凭证。

**新模型核心特征**：
- 每个 OMC 实例**只有 1 张有效 license**（singleton）
- license 来源是**外部签发系统**（OEM 签发服务，不在 OMC 内）
- license 覆盖范围是**整个系统**（含全部设备类型 + 全部功能模块）
- 操作只有一种：**Update**（上传新文件覆盖当前）；无"激活/撤销"概念

---

## 2. 老 OMC 现状（参考标准）

playwright 实测 http://172.21.175.129:8081 System > License 页面：

### 2.1 Basic Info
- License ID: `NO2022-03-14002`（厂商签发的全局唯一 ID）
- License Type: `Commercial` / `Trial` / etc.
- Expiry Date: `2049-01-01 00:00:00`（Remain 8263 Days）

### 2.2 Devices Support（按设备类型分别配额）
```
eNB:10000  GNB:10000  CPE:10000  WCG:1000  UPS:1000
```

### 2.3 Feature List（三级嵌套结构）
```
Dashboard:  All
MAP:        All
eNB:
  Monitor(7):           Synchronize / Settings / Active / RF Enable / Expiry Date / Traffic Limitation / TR069 Msg Exchange
  Maintenance(8):       MML / Configuration / Reset Configuration / Change Password / Reboot / Logs / Signaling Trace / Backup&Restore
  Upgrade&Rollback(5):  File Mgmt / IMAGE Upgrade / Patch Upgrade / FPGA Upgrade / Software Rollback
  Inventory(4):         Cert / Device / License / Data Model
gNB:                    Monitor(5) / Maintenance(6) / Upgrade&Rollback(3) / Inventory(4)
CPE:                    Monitor(3) / Maintenance(5) / Upgrade(2) / Inventory(3)
UPS / WCG:              All
Alarm:                  Alarm(2) — View / Library
Core Network:           Halob(1)
Performance:            KPI View / KPI Meas / Manually add kpi / Manually add counter / KPI Alarm
Advance:                SAS / Access Control
PlugAndPlay:            PlugAndPlay(1)
Performance MR:         MR(1)
SON:                    SON(1)
System:                 Resources / Setting(2) / Log(4) / User(3) / backup&restore(1) / Help(1) / IP Block List(1)
```

### 2.4 Update 操作
左下角 `Update` 按钮 → 弹 file picker → 上传 license 文件 → 覆盖当前。

---

## 3. 数据模型

### 3.1 `system_license` 表（singleton，最多 1 行 active）

```sql
CREATE TABLE system_license (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id           VARCHAR(100) NOT NULL UNIQUE,  -- 厂商签发 ID（NO2022-03-14002）
    license_type         VARCHAR(50)  NOT NULL,         -- Commercial / Trial / Evaluation / Internal
    issuer               VARCHAR(200),                  -- 签发方（如 "Baicells OEM"）
    licensee             VARCHAR(200),                  -- 授权对象（客户公司 / 部署 ID）
    issued_at            TIMESTAMPTZ NOT NULL,
    expiry_date          TIMESTAMPTZ,                   -- NULL = 永久
    devices_support      JSONB NOT NULL DEFAULT '{}',   -- {"eNB":10000,"gNB":10000,...}
    feature_list         JSONB NOT NULL DEFAULT '{}',   -- 三级嵌套见 §2.3
    raw_content          TEXT NOT NULL,                 -- 原始 license 文件全文（审计追溯 + 重新验签）
    signature            TEXT,                          -- base64 RSA-PSS 签名
    signature_key_id     VARCHAR(128),                  -- OEM 公钥 SHA-256 fingerprint
    signature_status     VARCHAR(20) NOT NULL,          -- verified / unverified / invalid
    uploaded_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by_user_id  UUID,                          -- 谁执行的 Update（操作员）
    is_current           BOOLEAN NOT NULL DEFAULT true,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- singleton 约束：只允许 1 行 is_current=true
CREATE UNIQUE INDEX uq_system_license_current ON system_license(is_current) WHERE is_current = true;

CREATE INDEX idx_system_license_expiry ON system_license(expiry_date) WHERE is_current = true;
```

### 3.2 `system_license_history` 表（每次 Update 留档）

```sql
CREATE TABLE system_license_history (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id           VARCHAR(100) NOT NULL,
    license_type         VARCHAR(50)  NOT NULL,
    devices_support      JSONB NOT NULL,
    feature_list         JSONB NOT NULL,
    raw_content          TEXT NOT NULL,
    signature_status     VARCHAR(20) NOT NULL,
    uploaded_at          TIMESTAMPTZ NOT NULL,
    uploaded_by_user_id  UUID,
    replaced_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- 被新 license 替换的时间
    replaced_by_id       UUID,                                -- 替换它的新 license PK
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_license_history_license_id ON system_license_history(license_id);
CREATE INDEX idx_system_license_history_uploaded ON system_license_history(uploaded_at DESC);
```

### 3.3 Update 业务规则

- 上传新 license → 验签（strict mode，必须 verified）→ 把当前 `system_license` 一行（如有）拷贝到 `system_license_history` 并置 `is_current=false` → INSERT 新行 `is_current=true`
- 全程在一个事务里
- 如果新 license 签名验证失败，整体回滚，旧 license 保留不变

---

## 4. License 文件格式（建议）

JSON 文件，结构基于老 OMC 页面字段反推：

```json
{
  "license_id":   "NO2022-03-14002",
  "license_type": "Commercial",
  "issuer":       "Baicells OEM",
  "licensee":     "Test Customer Deployment",
  "issued_at":    "2026-05-18T00:00:00Z",
  "expiry_date":  "2049-01-01T00:00:00Z",

  "devices_support": {
    "eNB": 10000,
    "gNB": 10000,
    "CPE": 10000,
    "WCG": 1000,
    "UPS": 1000
  },

  "feature_list": {
    "Dashboard": "All",
    "MAP": "All",
    "eNB": {
      "Monitor":          ["Synchronize","Settings","Active","RF Enable","Expiry Date","Traffic Limitation","TR069 Msg Exchange"],
      "Maintenance":      ["MML","Configuration","Reset Configuration","Change Password","Reboot","Logs","Signaling Trace","Backup&Restore"],
      "Upgrade&Rollback": ["File Mgmt","IMAGE Upgrade","Patch Upgrade","FPGA Upgrade","Software Rollback"],
      "Inventory":        ["Cert","Device","License","Data Model"]
    },
    "gNB": { "Monitor": [...], "Maintenance": [...], "Upgrade&Rollback": [...], "Inventory": [...] },
    "CPE": { "Monitor": [...], "Maintenance": [...], "Upgrade": [...], "Inventory": [...] },
    "UPS":   "All",
    "WCG":   "All",
    "Alarm": { "Alarm": ["View","Library"] },
    "Core Network": { "Halob": [] },
    "Performance": ["KPI View","KPI Meas","Manually add kpi","Manually add counter","KPI Alarm"],
    "Advance":  { "SAS": [], "Access Control": [] },
    "PlugAndPlay":    ["PlugAndPlay"],
    "Performance MR": ["MR"],
    "SON":            ["SON"],
    "System": {
      "Resources":      ["Resources"],
      "Setting":        ["Device Setting","North Interface Setting"],
      "Log":            ["Operation Logs","Security Logs","System Logs","North Interface Logs"],
      "User":           ["Role","User Group","User"],
      "Backup&Restore": ["Backup&Restore"],
      "Help":           ["License"],
      "IP Block List":  ["IP Block List"]
    }
  },

  "signature":        "<base64 RSA-PSS over canonical(去掉 signature/signature_key_id 后字典序 JSON)>",
  "signature_key_id": "<OEM 公钥 SHA-256 fingerprint hex>"
}
```

### 4.1 签名机制

复用现有 `internal/license/signature.go` 的 `SignatureVerifier`：
- 算法：RSA-PSS / SHA-256 / salt 长度 32
- canonical 算法：去掉 `signature` + `signature_key_id` 后按 key 字典序 marshal（不带空格）
- 公钥目录：`configs/oem_public_keys/*.pem`
- **strict 模式默认开**（与当前默认 false 相反）—— 系统 license 是核心授权，未签 / 签名失败必须拒绝

### 4.2 签发工具（OMC 内）

提供 `omcctl gen-license` CLI 子命令，OEM 侧用自己的私钥签：

```bash
omcctl gen-license \
  --license-id NO2026-05-001 \
  --license-type Commercial \
  --issuer "Baicells OEM" \
  --licensee "Customer Co." \
  --expiry-date 2027-05-18T00:00:00Z \
  --devices-support eNB=10000,gNB=10000,CPE=10000 \
  --feature-template configs/license/feature-templates/full.json \
  --private-key /path/to/oem-private.pem \
  --output omc-license-2026-05-001.json
```

---

## 5. API 契约

| 端点 | 方法 | 用途 | 鉴权 |
|------|------|------|------|
| `/api/v1/system-license` | `GET` | 获取当前生效的 license 详情（含 devices_support / feature_list） | 任意已登录 |
| `/api/v1/system-license` | `POST` | 上传新 license 文件覆盖当前 | `system:license:operate` |
| `/api/v1/system-license/history` | `GET` | 列出历史 license（分页） | `system:license:operate` |
| `/api/v1/system-license/feature-check` | `GET` | 内部接口，前端 / 后端中间件查"某 feature 当前是否授权" | 任意已登录 |

### 5.1 POST `/system-license` 请求体

```json
{
  "raw_content": "<license JSON 文件原始字符串>"
}
```

### 5.2 POST `/system-license` 响应

```json
{
  "data": {
    "license_id":       "NO2026-05-001",
    "license_type":     "Commercial",
    "signature_status": "verified",
    "expiry_date":      "2027-05-18T00:00:00Z",
    "uploaded_at":      "2026-05-18T16:00:00Z",
    "replaced":         { "license_id": "NO2022-03-14002", "uploaded_at": "..." }
  }
}
```

### 5.3 错误码

| code | 含义 |
|------|------|
| 12109 | signature verification failed (strict mode) — 与现有保持一致 |
| 12110 | license_id collision（新 license_id 已存在于 history 表） |
| 12111 | invalid license JSON format（必填字段缺失） |
| 12112 | downgrade refused（新 license 容量小于已用，需 force 或先减容） |

---

## 6. 前端 wireframe

**单页 `/license`**（替换 /license/list, /license/operations, /license/logs）：

```
┌─────────────────────────────────────────────────────────┐
│ License                                                 │
├─────────────────────────────────────────────────────────┤
│ [Basic Info]                                            │
│  License ID:    NO2022-03-14002                         │
│  License Type:  Commercial                              │
│  Expiry Date:   2049-01-01 (Remain 8263 Days)           │
│  Signature:     ✓ Verified (key: abc12...)              │
├─────────────────────────────────────────────────────────┤
│ [Devices Support]                                       │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ │
│  │  eNB   │ │  gNB   │ │  CPE   │ │  WCG   │ │  UPS   │ │
│  │ 10000  │ │ 10000  │ │ 10000  │ │  1000  │ │  1000  │ │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘ │
├─────────────────────────────────────────────────────────┤
│ [Feature List]                                          │
│  Dashboard │ All                                        │
│  MAP       │ All                                        │
│  eNB       │ Monitor(7)  • Synchronize • Settings • ... │
│            │ Maintenance(8) ...                         │
│            │ Upgrade&Rollback(5) ...                    │
│            │ Inventory(4) ...                           │
│  gNB       │ ...                                        │
│  ...                                                    │
├─────────────────────────────────────────────────────────┤
│ [Update]   ← 按钮，点击弹 file picker 上传新 license      │
└─────────────────────────────────────────────────────────┘
```

**Update 流程**：
1. 点 Update 按钮
2. 弹 Modal 含 Dragger（拖入 .lic / .json）+ 解析预览（新 license 的 ID / 类型 / 容量对比）
3. 用户确认 → POST /system-license → 成功 toast，页面刷新显示新 license

**额外页面**：
- `/license/history`：列表展示历史 license + 替换时间

---

## 7. 实施 Phase

| Phase | 内容 | 估时 | 依赖 |
|-------|------|------|------|
| **P1** | DB migration（DROP licenses + license_logs；CREATE system_license + history）+ model.go / repo.go 重写 | 4-6h | 无 |
| **P2** | handler（GetCurrent / Update / GetHistory）+ singleton 事务 + 签名强制验证 | 3-4h | P1 |
| **P3** | enforcer / monitor 改造：从 licenses 表多张 → system_license devices_support map 算容量 | 3-4h | P1 |
| **P4** | 前端单页 `/license` + Update Modal + 替换路由 + 删 LicenseList/Operations/Logs | 4-5h | P2 |
| **P5** | omcctl gen-license CLI 子命令（OEM 用） | 2-3h | P2 |
| **P6** | 测试样本重做：scripts/license-samples/ 重写为 system license 格式 + seed-licenses.sh 适配 | 2h | P2 + P5 |
| **P7** | RBAC feature 联动（后端 middleware + 前端菜单按 feature_list 显示/隐藏）| 6-8h | P2，可分批 |

总计：~25-32h（不含 P7 RBAC 联动）。

### 7.1 Phase 拆分原则

P1-P6 是**最小可用闭环**：用户能上传 license / 查看详情 / 历史。

P7 是**功能授权落地**：让 feature_list 真的控制 UI 可见性 + API 拦截。这是大改造，涉及前端菜单 / 后端 API gate / Casbin policy 同步——建议作为独立 sprint（T-0NNN-feature-gate）。

---

## 8. 废弃清单（破坏性）

### 8.1 后端

| 文件 | 处理 |
|------|------|
| `migrations/000006_*.sql` 中 `CREATE TABLE licenses` | 用新 migration DROP（保留 schema，破坏性 down） |
| `migrations/000043_licenses_capacity_expiry.sql` | DROP |
| `migrations/000073_license_logs.sql` | DROP |
| `migrations/000074_license_logs_auto_revoke.sql` | DROP |
| `internal/license/handler.go` | 80% 重写（保 ImportRequest 签名验证部分可借鉴）|
| `internal/license/service.go` | 重写（Import/Activate/Revoke → GetCurrent/Update/GetHistory）|
| `internal/license/model.go` | 重写 schema |
| `internal/license/pg_repository.go` | 重写 |
| `internal/license/pg_license_log_repository.go` | 删 |
| `internal/license/log_writer.go` | 删 |
| `internal/license/log_handler_test.go` | 删 |
| `internal/license/archiver.go` + `_test.go` | 删（若只服务 license_logs） |
| `internal/license/exporter.go` + `_test.go` | 保留并适配（PDF/CSV 导出 system license 详情） |
| `internal/license/enforcer.go` + `_test.go` | 改造：从 licenses 表 → system_license devices_support |
| `internal/license/monitor.go` + `_test.go` | 改造：从扫多张 license → 扫 system_license 1 张 |
| `internal/license/metrics.go` | 适配新 schema |
| `internal/license/signature.go` + `_test.go` | **保留**（验签算法不变，可能 strict 默认改 true）|

### 8.2 前端

| 文件 | 处理 |
|------|------|
| `omcmb/webcode/src/pages/license/LicenseList/index.tsx` | 删 |
| `omcmb/webcode/src/pages/license/LicenseOperations/index.tsx` | 删 |
| `omcmb/webcode/src/pages/license/LicenseLogs/index.tsx` | 删 |
| `omcmb/frontend-core/src/services/api/licenseApi.ts` | 大改：删 Import/Activate/Revoke API，加 GetCurrent/Update/GetHistory |
| `omcmb/frontend-core/src/hooks/api/useLicense.ts` | 同上 |
| `omcmb/webcode/src/router/routes.tsx` | 路由表更新：/license 单页 + /license/history |

### 8.3 测试 / 工具

| 文件 | 处理 |
|------|------|
| `omcgo/scripts/license-samples/01-08-*.json` | 重做：单文件 + system license 格式 |
| `omcgo/scripts/license-samples/seed-licenses.sh` | 重写：调 POST /system-license + 验签 |
| `omcgo/scripts/license-samples/README.md` | 重写测试矩阵 |

### 8.4 老 PRD

- `docs/project/prd/F06-license.md` — 标记 deprecated，链接到本 PRD
- `docs/project/prd/F06-license-enforcement.md` — 保留作执行层历史，新 enforcer 行为参考本 PRD §3.3

---

## 9. 需要拍板的剩余问题

| # | 问题 | 默认建议 |
|---|------|------|
| Q1 | License 文件格式按 §4 走？ | ✓ 按此方案，含 signature 字段 |
| Q2 | strict 签名模式默认开（未签 / 签名失败拒绝 Update）？ | ✓ 默认开，可通过配置 `license.strict_mode: false` 关 |
| Q3 | omcctl gen-license 工具要内置吗？还是 OEM 自己用 openssl 写脚本？ | 内置：方便 dev/staging 自签测试 |
| Q4 | feature_list 真实落地（Phase 7 RBAC 联动）作为本期还是下期？ | **下期**（独立 sprint）；本期只入库展示 |
| Q5 | history 表保留多久？ | 永久（合规归档要求）|
| Q6 | "新 license 容量比旧 license 小但已用超过新容量"的降级场景怎么处理？ | 默认拒，errcode 12112；前端弹 Modal 提示 force 选项 |
| Q7 | 已激活的旧 license 是否需要数据迁移工具（从老 OMC export → 转 system license）？ | 暂不做，fresh start |

---

## 10. 验收标准

### 10.1 后端（P1-P3）

- [ ] migration 跑通：drop 老表 + create 新表 + 0 行残留
- [ ] `POST /system-license` 上传 sample license → 201 + signature_status=verified
- [ ] 重复上传同 license_id → history 表多出 1 行，system_license 表仍 1 行
- [ ] 上传未签名 license + strict=true → 400 errcode 12109，DB 无变化
- [ ] enforcer 改造后：注册第 N+1 设备（N = system_license.devices_support[type]）触发 enforcement_capacity 审计

### 10.2 前端（P4）

- [ ] `/license` 单页显示 Basic Info / Devices Support / Feature List 三段
- [ ] Update 按钮弹 Modal → 拖入 .json → 解析预览 → 确认上传 → 刷新页面
- [ ] `/license/history` 列出历史 license
- [ ] 老页面 /license/list, /license/operations, /license/logs 全部 404 或自动 redirect

### 10.3 端到端（P6）

- [ ] `scripts/license-samples/seed-licenses.sh` 一键灌入新 sample → 详情页正确显示

---

## 11. 风险

| 风险 | 缓解 |
|------|------|
| 老 licenses 数据 DROP 不可恢复 | 本次明确"fresh start"，无迁移需求 |
| feature_list 落地涉及前端菜单 + 后端 API 大改 | 拆 Phase 7 独立 sprint，本期不阻塞 |
| 签名验证 strict 默认开可能 break dev 环境（无公钥）| 配置开关 `license.strict_mode`；dev/staging 默认 false |
| 老 PRD F06-license.md / F06-license-enforcement.md 引用关系 | 本 PRD 显式标记 deprecate；新 PRD 是唯一需求源 |

---

## 12. 决策日志

- **2026-05-18 用户决策**：方案 A 完全替代；现有 licenses 数据 DROP；本 PRD 取代 F06-license.md。

---

**审批人**：用户
**审批后立即启动 P1**（DB migration + model/repo 重写）。

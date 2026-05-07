# 系统管理 — 系统配置（System / Config）PRD

> 文档目的：梳理 `/system/config` 页面 9 个 Tab 的现状，本版本 **下线 SAS 与 LDAP 两个 Tab**，落地为 7 Tab。
>
> 本文档由 `system-config.md` v0.1 演进而来，作为 v1.0 取代之。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 9 Tab 现状梳理（SAS / LDAP 在内）— 见 `system-config.md` |
| 1.0  | 2026-05-07 | Frontend Team | **下线 SAS / LDAP**；保留 7 Tab。本文件取代 `system-config.md` |

**关联功能域**：F06 OMC-R 核心 / 全局基础设施

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面入口 | `omcmb/webcode/src/pages/system/SystemConfig/index.tsx` |
| 前端 Tab 子组件 | `omcmb/webcode/src/pages/system/SystemConfig/{Basic,Security,Device,Notification,Storage,Omc,Northbound}Settings.tsx` |
| 前端 Tab 子组件（**本版本删除**） | `omcmb/webcode/src/pages/system/SystemConfig/{Sas,Ldap}Settings.tsx` |
| i18n（zh-CN）| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` |
| 后端 Handler | `omcgo/internal/admin/sys_config_handler.go` |
| 后端 Repo | `omcgo/internal/admin/sys_config.go` |
| 数据库 | `omcgo/migrations/000009_sys_admin.sql:113-126`（`sys_configs` KV 表）|

---

## 1. 业务背景与本版本变更

`sys_configs` 是 KV 配置存储，按 `category` 分类。

**v0.1 现状**：前端有 9 个 Tab —— `basic / security / device / notify / storage / omc / northbound / sas / ldap`。

**v1.0 决议（本版本）**：**移除 `sas` 和 `ldap` 两个 Tab**。原因：

| Tab | 下线原因 |
|---|---|
| `sas` (SAS 设置) | 频谱接入系统专项；与本系统当前业务范围（移动/电信/联通 LTE/NR 网管）解耦。原 SAS 配置入口在 navConfig.ts 中已注释隐藏（参 §3.x），保留 Tab 易误导。 |
| `ldap` (LDAP 协议) | 当前 RBAC + 内置用户已满足身份管理需求；后端 `users.source='LDAP'` 字段保留兼容历史登录数据，但配置入口下线后不再支持新增 LDAP 集成。 |

**保留的 7 个 Tab**（顺序不变）：

| key | 中文 | 用途 |
|---|---|---|
| `basic` | 基本设置 | 厂商名 / 系统名 / 时区 |
| `security` | 安全设置 | 密码策略 / 登录失败锁定 / 会话超时 / 浏览器记住密码 |
| `device` | 设备设置 | Inform 周期 / 离线超时 / 命名规则 / RSRP 接入控制 / 设备离线日数 |
| `notify` | 通知设置 | 邮件服务器 SMTP 配置 |
| `storage` | 存储设置 | 日志/告警/KPI/MR 存储天数 / 磁盘告警阈值 / 日志 FTP 上传 |
| `omc` | 网管设置 | 远端 syslog / OMC 自身磁盘告警阈值 |
| `northbound` | 北向接口设置 | 北向 OSS 用户管理 |

---

## 2. 页面布局（v1.0）

### 2.1 顶部

`<ListPageLayout title="系统配置">` + 顶部 `<Tabs>`，下方为当前激活 Tab 的表单区，最底部居中放一个「保存」按钮。

### 2.2 Tab 顺序与 i18n key

```
settingsTabs: [
  { key: 'basic',      labelKey: 'system.config.basic'      },  // 基本设置
  { key: 'security',   labelKey: 'system.config.security'   },  // 安全设置
  { key: 'device',     labelKey: 'system.config.device'     },  // 设备设置
  { key: 'notify',     labelKey: 'system.config.notify'     },  // 通知设置
  { key: 'storage',    labelKey: 'system.config.storage'    },  // 存储设置
  { key: 'omc',        labelKey: 'system.config.omc'        },  // 网管设置
  { key: 'northbound', labelKey: 'system.config.northbound' },  // 北向接口设置
];
```

### 2.3 删除项（v1.0 移除）

- ~~`{ key: 'sas',  labelKey: 'system.config.sas'  }~~ // SAS 设置  → 删
- ~~`{ key: 'ldap', labelKey: 'system.config.ldap' }~~ // LDAP 协议 → 删

---

## 3. 表单字段（保留 Tab 现状速览）

> 详细字段见各子组件源文件；本节仅列各 Tab 字段名集合，便于回归测试。**字段集合不变** — 本版本仅删除 SAS / LDAP 两个 Tab，对 7 个保留 Tab 的字段无任何改动。

| Tab | 主要字段 |
|---|---|
| basic | `mrVendor / mrOMCName / timezoneCode` |
| security | `modifyPWD / defaultPasswd / passwordContent / pwdMinLength / pwdMaxLength / expires / validPeriod / promptBeforeDays / verifyEnable / attemptTimes / sumTimes / unlockMinu / limitMinus / limitCount / limitTimes / userSessionExpirationMin / isBrowserAutoRecordPass / autoLockUserDayEnable / autoLockUserDay / isOnlyOneUserLoginEnable / enabledFlag / msg` |
| device | `enbInformPeriodAdjustEnable / enbInformPeriod / enbTimeoutEnable / enbTimeout / cpeInformPeriodAdjustEnable / cpeInformPeriod / cpeTimeoutEnable / cpeTimeout / nameSettingEnable / prompt / accessContralEnable / rsrpVal0 / rsrpVal1 / uersrpVal0 / uersrpVal1 / uploadSelected / deviceOfflineEnable / deviceOfflineSaveDay / locationDetection / latitudeToleranceRange` |
| notify | `emEnabel / mailUsername / mailPassword / mailHost / mailPort` |
| storage | `logDataSaveDays / rebootLogDataSaveDays / rebootLogSaveCount / sysOperateLogDataSaveDays / logFtpEnable / logFtpType / logFtpSavePath / logFtpIpAddr / logFtpPort / logFtpUser / logFtpPassword / alarmHisMaxHoldTime / kpiFilesSaveDays / kpiReportDataSaveDays / kpiStorge15DataDays / kpiStorge60DataDays / kpiStorge1440DataDays / kpiWeekAndMonthSwitch / mrFileSaveDays / signalingTraceSaveDays / varDiskAlarmThresHold / homeDiskAlarmThresHold / usrDiskAlarmThresHold / rootDiskAlarmThresHold` |
| omc | `rsysLogEnable / rsysLogIp / rsysLogPort / varDiskAlarmThresHold / homeDiskAlarmThresHold / usrDiskAlarmThresHold / rootDiskAlarmThresHold` |
| northbound | `userName / userPwd / userEnable`（北向用户 CRUD） |

---

## 4. 操作清单

| # | 操作 | 触发位置 | 接口（现状 / 计划）|
|---|------|---------|------------------|
| 1 | 切换 Tab | 顶部 `<Tabs>` | 纯前端状态切换 |
| 2 | 保存当前 Tab | 底部「保存」按钮 | `validateFields()` → 当前桩为 `setTimeout 800ms` 假成功；待 P0 接 `POST /admin/sysConfig/batch`（见 §6） |

---

## 5. 接口契约

### 5.1 已实现

继承 v0.1 `system-config.md` §6 — 沿用 `sys_configs` 表的 KV CRUD：

```
GET    /admin/sysConfig             (按 category 列表)
GET    /admin/sysConfig/:id         (单条)
POST   /admin/sysConfig             (新增)
PUT    /admin/sysConfig/:id         (单条更新)
DELETE /admin/sysConfig/:id
```

### 5.2 待补（继承 v0.1 P0）

- 批量更新接口 `POST /admin/sysConfig/batch`，请求体：
  ```json
  { "category": "security", "items": [ { "key": "pwdMinLength", "value": "8" }, ... ] }
  ```
  避免逐字段 PUT。

### 5.3 v1.0 不涉及的接口

- ~~SAS 配置接口~~ — 不再需要（前端 SAS Tab 删除）
- ~~LDAP 配置接口~~ — 不再需要（前端 LDAP Tab 删除）

---

## 6. v1.0 落地步骤（实施清单 — **审核通过后才执行**）

### 6.1 前端

1. `omcmb/webcode/src/pages/system/SystemConfig/index.tsx`
   - 删除 `import SasSettings from './SasSettings'` 与 `import LdapSettings from './LdapSettings'`
   - `type SettingsTab` union 删除 `'sas' | 'ldap'`
   - `settingsTabs[]` 删除 `sas` / `ldap` 两项
   - `Form.useForm()` 实例 `sasForm` / `ldapForm` 删除
   - `getCurrentForm()` 的 `formMap` 删除两项
   - `renderSettingsContent()` 的 switch case 删除两项
2. **删除组件文件**：
   - `omcmb/webcode/src/pages/system/SystemConfig/SasSettings.tsx`（323 行）
   - `omcmb/webcode/src/pages/system/SystemConfig/LdapSettings.tsx`（104 行）
3. **i18n 清理**（`omcmb/frontend-core/src/i18n/zh-CN/index.ts` 与 `en-US/index.ts`）：
   - 删除 `'system.config.sas'` 和 `'system.config.ldap'`
   - 删除 `'system.sas.*'` 与 `'system.ldap.*'` 整组（约 30+ key）
4. typecheck / eslint 必须通过。

### 6.2 后端

- 代码层无改动（`sys_configs` 表与 handler/repo 都是泛 KV，无 sas/ldap 专项逻辑）。
- 数据层：新增 seed 迁移彻底清理 `sys_configs` 中 `category IN ('sas','ldap')` 的历史行（参 §6.3）。

### 6.3 数据迁移

- **新增 seed 迁移** `migrations/seed/000060_drop_sas_ldap_configs.sql`：
  - Up: `DELETE FROM sys_configs WHERE category IN ('sas','ldap');`
  - Down: 不可逆（`SELECT 1;` 占位）。
- 无 schema（DDL）变更。

### 6.4 文档

- **删除** 旧 PRD `omgo/docs/prd/system/system-config.md`（不保留为档案，遵从"全部删除不保留"决议）。
- `README.md` 第 7 行 PRD 链接由 `system-config.md` → `config.md`，备注改为「7 Tab（v1.0 删 SAS/LDAP）」。

### 6.5 测试

- 进入页面，Tabs 渲染 7 个，无 SAS / LDAP；切换无 console 报错。
- typecheck / lint clean。
- 每个保留 Tab 的字段渲染、保存按钮 (`message.success` 桩) 行为不变。

---

## 7. 风险与回滚

| 项 | 评估 |
|---|------|
| 影响面 | 前端 Tab + i18n + 旧 PRD 文件全删；后端代码无改；DB 一次性清理 |
| 回滚方式 | `git revert` 本次 commit 恢复代码 + i18n + 旧 PRD；DB 已删行需手工补 INSERT（一般无人维护过 SAS/LDAP 配置时无需补） |
| 兼容性 | LDAP 用户登录路径（`users.source='LDAP'`）不受影响 — 仅入口下线，登录通路保留 |

---

## 8. 验收清单（DoD）

- [ ] `system/config` 页面 Tab 数 = 7（不再有 SAS / LDAP）
- [ ] 删除的两个组件文件不存在；index.tsx 无悬挂 import
- [ ] zh-CN / en-US 中 `system.config.sas` / `system.config.ldap` 及对应 `system.sas.*` / `system.ldap.*` 全部清除
- [ ] `npm run typecheck` 通过
- [ ] `npm run lint` 在改动文件上 clean
- [ ] `omgo/docs/prd/system/README.md` 链接已更新
- [ ] 旧 PRD `system-config.md` 已删除
- [ ] DB 中 `sys_configs` 无 `category IN ('sas','ldap')` 的行（执行 `migrate-seed-up` 后）

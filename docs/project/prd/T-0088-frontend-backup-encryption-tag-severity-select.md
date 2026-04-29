# PRD: 前端 BackupPolicy encryption Tag + alert_severity select（T-0088）

> **关联**: Backlog T-0088 / Sprint-08 / Domain=frontend / Type=feat / Prio=P3
> **作者**: Claude（代 Owner=前端）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: encryption Tag 用 FE-only 派生（EnableEncryption + algo 即 ready）；alert_severity Select 3 选 1；多皮肤零回归

---

## 1. 业务背景

T-0084（severity policy-driven）后端落字段 + validate，**FE form 没有相应 select** → 运维通过 PUT /backup/policy 时无法显式设 alert_severity，全靠默认 "major"。
T-0075（AES-256-GCM 加密落地）后端就绪，FE 当前 BackupPolicy 加密段仅显示固定 orange "尚未生效" Tag — **算法已实施但 UI 没体现差异**。

T-0088 闭环 FE：
1. encryption Tag 条件化：`EnableEncryption=true && algo="AES-256-GCM"` → green "已生效"；其他保 orange warning
2. alert_severity Select 加入 form：3 选 1，与 T-0084 schema 完全对齐

---

## 2. ULTRATHINK 决策

### 2.1 encryption_ready 计算 — FE-only vs backend API 扩展

option A：后端加 `encryption_ready: bool` 派生字段到 GET /backup/policy 响应（基于 KeyProvider.Available()）  
option B：**FE-only 派生** — `EnableEncryption=true && algo="AES-256-GCM"` 即视为 ready；KEK 实际不可用时 PUT 时 backend 返 400 让用户重试

采纳 B。理由：
- T-0088 是 P3 / S 任务，加 backend API 字段把 scope 推到 cross-stack
- FE 当前没有读 KEK env 的路径（架构正确）
- 用户 PUT 失败时 backend 错误消息已含明确指示（`encryption_algorithm=AES-256-GCM enabled but encryption key not configured`），UX 闭环
- 未来若有"KEK 状态徽章"需求可单独任务

option B 同样会被 T-0085 兼容：CBC + ChaCha20 接入后可扩展白名单。

### 2.2 alert_severity Select 默认值 + 顺序

```tsx
<Select defaultValue="major">
  <Option value="warning">警告 / Warning</Option>
  <Option value="major">重要 / Major</Option>
  <Option value="critical">严重 / Critical</Option>
</Select>
```

顺序按严重度递增（运维心智 — 从轻到重）。tag color:
- warning → blue/yellow（视觉低调）
- major → orange（默认 / 突出）
- critical → red（最高级）

### 2.3 多皮肤影响

frontend-core/types/api 加 `alert_severity` 字段必然影响 webcode + webcode-v2 + webcode-v3：
- webcode（主皮肤）— 完整实施
- webcode-v2 / v3 — 仅需保证 typecheck 不破坏（field 可选 / pass-through 即可）

确认两个候选皮肤的 BackupPolicy 是否有 form：v2/v3 当前是骨架/占位，无 BackupPolicy 实质 form。零影响。

### 2.4 encryption Tag 条件逻辑

```tsx
const isReady = (
  values.enable_encryption === true &&
  values.encryption_algorithm === "AES-256-GCM"  // T-0085 后扩展白名单
);
const tag = isReady
  ? <Tag color="green">已生效 / Active</Tag>
  : (values.enable_encryption
     ? <Tag color="orange">尚未生效 / Not yet effective</Tag>
     : <Tag color="default">未启用 / Disabled</Tag>);
```

第三态（未启用）是新增的 — 比当前"任何 EnableEncryption=false 都没 tag"更清晰。

### 2.5 alert_severity field 在 form 哪一段

放"告警"段 — 与 alert_on_failure / alert_email / alert_threshold_percent 同段。顺序在 alert_threshold_percent 之后（按严重度逻辑链：先讲什么时候触发，再讲 触发的严重度）。

### 2.6 i18n keys

需 ~12 个新 key × 2 locale：
- `backup.policy.alertSeverity` (label)
- `backup.policy.alertSeverityWarning` / `Major` / `Critical` (3 选项)
- `backup.policy.alertSeverityHelp`
- `backup.policy.encryptionStatusActive` / `NotYetEffective` / `Disabled` (3 tag 文案)
- `backup.policy.encryptionStatusActiveHelp`

### 2.7 Mock data

`omcmb/frontend-core/src/mock/data/...` 的 mockBackupPolicy 加 `alert_severity: "major"`。否则 mock 模式下表单 default 显示空。

### 2.8 Form 字段顺序（不破坏既有）

既有 form 顺序保留：保留 / 自动清理 / 压缩 / 存储 / 加密 / 告警。仅在"告警"段末尾追加 alert_severity。"加密"段的 Tag 条件化 in-place。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我配 KEK + EnableEncryption=true + algo=AES-256-GCM，UI 显 green "已生效" Tag — 知道加密真正落地 |
| 网管运维（关键客户）| 我设 alert_severity=critical 让 backup 失败走 PagerDuty；UI Select 直观可见 |
| 前端开发 | 业务层 BackupPolicy 类型加 alert_severity field；UI 加 Select；多皮肤 v2/v3 typecheck 不破 |

---

## 4. 验收标准（GWT）

### V1 — encryption Tag green "已生效"
- **Given** form values: enable_encryption=true, algo="AES-256-GCM"
- **Then** Tag color="green" + 文案 "已生效"

### V2 — encryption Tag orange "尚未生效"
- **Given** form values: enable_encryption=true, algo="AES-256-CBC"
- **Then** Tag color="orange" + 文案 "尚未生效"（T-0085 落地后白名单扩展才会绿）

### V3 — encryption Tag default "未启用"
- **Given** form values: enable_encryption=false
- **Then** Tag color="default" + 文案 "未启用"

### V4 — alert_severity Select 默认 major
- **Given** mock policy 或 GET /backup/policy 返 alert_severity="major"
- **Then** form Select 显 "Major"

### V5 — alert_severity Select 三选项可选
- **When** 用户开 Select dropdown
- **Then** 显 warning / major / critical 三选项 + i18n 标签

### V6 — alert_severity 提交到 backend
- **Given** form 改 alert_severity → "critical"
- **When** 点保存 → PUT /backup/policy
- **Then** request body 含 `alert_severity: "critical"`

### V7 — backend 返 backend 字段 alert_severity 时正确显示
- **Given** mock 或真实 backend 返 alert_severity="warning"
- **Then** form Select 显 "Warning"

### V8 — typecheck 通过
- **When** `cd omcmb/webcode && npm run typecheck`
- **Then** 0 errors

### V9 — webcode-v2/v3 typecheck 不破
- **When** typecheck v2 + v3
- **Then** 仍通过（field 是可选 + 不强引用）

---

## 5. 运营商差异矩阵

无差异。Form 是统一 OMC 后台界面。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | encryption_ready backend API 字段 | option B 选定，T-0088 P3 不引 cross-stack 改动 |
| N2 | KEK 状态徽章独立显示 | 未来需求时单独任务 |
| N3 | severity 颜色和 alarm 引擎 dispatch 联动 | alarm 引擎 filter rule 已能 match severity；FE 仅提供输入 |
| N4 | 多皮肤同步实施 | webcode-v2/v3 是骨架，仅保证 typecheck；UI 同步拆未来 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0084 alert_severity schema + validate | ✅ done |
| T-0075 加密 backend | ✅ done |
| frontend-core BackupPolicy types | ✅ done — 仅扩字段 |

---

## 8. 度量

无新增 metric — FE 是状态展示。已有 `omc_backup_encrypted_total` 反映加密真正落地数量；FE Tag 仅展示 form values 派生。

---

## 9. 设计备忘（S2）

### 9.1 frontend-core 扩展

`frontend-core/src/services/api/backupApi.ts` BackendBackupPolicy + BackupPolicy types 加：
```ts
export interface BackupPolicy {
  // ... existing fields
  alert_severity: 'warning' | 'major' | 'critical';
}
```

mapBackendBackupPolicy / mapBackupPolicyToBackend 加 field round-trip。

`frontend-core/src/mock/data/backup.ts` mockBackupPolicy 加 `alert_severity: "major"`。

### 9.2 webcode UI 改动

`omcmb/webcode/src/pages/backup/BackupPolicy/index.tsx`：

1. 加密段 Tag 条件化逻辑：
   ```tsx
   const encryptionTag = useMemo(() => {
     if (!values.enable_encryption) return { color: "default", text: t("backup.policy.encryptionStatusDisabled") };
     if (values.encryption_algorithm === "AES-256-GCM") return { color: "green", text: t("backup.policy.encryptionStatusActive") };
     return { color: "orange", text: t("backup.policy.encryptionStatusNotYetEffective") };
   }, [values.enable_encryption, values.encryption_algorithm]);
   ```

2. 告警段末尾追加 alert_severity Select：
   ```tsx
   <Form.Item name="alert_severity" label={t("backup.policy.alertSeverity")}>
     <Select defaultValue="major">
       <Option value="warning">{t("backup.policy.alertSeverityWarning")}</Option>
       <Option value="major">{t("backup.policy.alertSeverityMajor")}</Option>
       <Option value="critical">{t("backup.policy.alertSeverityCritical")}</Option>
     </Select>
   </Form.Item>
   ```

### 9.3 i18n

`frontend-core/src/i18n/zh-CN/index.ts` + `en-US/index.ts` 各 +9 keys。

### 9.4 文件清单

修改：
- `omcmb/frontend-core/src/services/api/backupApi.ts` — 类型 + map
- `omcmb/frontend-core/src/mock/data/backup.ts` — mock 加字段
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts` + `en-US/index.ts` — i18n keys
- `omcmb/webcode/src/pages/backup/BackupPolicy/index.tsx` — Tag 条件 + Select

无新增文件，无 backend 改动。

### 9.5 多皮肤回归验证

```bash
cd omcmb/webcode    && npm run typecheck && npm run lint
cd omcmb/webcode-v2 && npm run typecheck
cd omcmb/webcode-v3 && npm run typecheck
```

---

## 10. 实施要点

预计工作量：S（约 0.5 人日）— 4 文件改 + 12 i18n key + 3 typecheck pass

预计涉及模块：`omcmb/`（4 文件）；后端零改动

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 前端 / 多皮肤 / QA | Claude | 2026-04-29 | option B FE-only encryption_ready；T-0085 落地后扩白名单 |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-29 | v1.0 | 初稿；ULTRATHINK 8 决策；FE-only 派生策略 | Claude |

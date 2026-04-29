# PRD: 前端 BackupSchedule UI 重设计（T-0070 / R-102 followup）

> **关联**: Backlog T-0070 / Sprint-06 / Domain=frontend / Type=feat
> **作者**: Claude（代 Owner=前端专家）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施

---

## 1. 业务背景

T-0016 audit 揭示 `omcmb/webcode/src/pages/backup/BackupSchedule/index.tsx` 460 行**名实不符**：

| 维度 | 现状 | 应有 |
|------|------|------|
| 路由 | `/backup/schedule` | 一致 |
| 类名 | `BackupSchedule` | 一致 |
| 菜单标签 | `nav.backup.schedule`（"备份调度"）| 一致 |
| **实际 UI** | "配置文件 import/export"（按文件名匹配设备 + Blob 下载 demo）| 应为 backup-schedule cron 任务管理 |
| **后端契约** | 当前页面消费 0 endpoint | 应消费 `/backup/schedules` 4 endpoint |

后端 `/backup/schedules` 4 endpoints + frontend-core 4 hooks（`useBackupSchedules` / `useCreateBackupSchedule` / `useUpdateBackupSchedule` / `useDeleteBackupSchedules`）已就绪 ≥ 1 个 milestone，**0 UI 消费**。

**当前误置 UI 性质评估**：
- mock 设备 12 条（`BJ朝阳基站01` 等假名）+ Blob 客户端拼接 XML 下载
- 后端无 `/backup/configfile-sync-by-device` 类端点
- **判定为 dev/demo placeholder**，未上线生产
- 因此本任务 **wholesale replace** 该页面，无需保留
- 若 future 需要"配置文件同步"功能，独立立项（不在本 followup 范围）

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望在"备份调度"菜单看到所有定时备份任务（cron 表达式），可启用/禁用/编辑/删除 |
| 网管运维 | 我希望新建调度时可选 cron 预设（每日/每周/每月）或自定义；可选目标设备组、备份类型 |
| 运营商客户 | 我希望"备份调度"菜单点进去就是 cron 调度管理，不再被混入"配置文件 import"困扰 |
| QA | 我希望前后端契约 e2e 可见 — 创建/更新/删除调度走真实 backend endpoint |

---

## 3. 验收标准（GWT）

### V1 — 列表展示真实 schedules
- **Given** 后端 `/backup/schedules` 返回 N 条调度
- **When** 用户访问 `/backup/schedule`
- **Then** 表格显示 `useBackupSchedules().data?.items`（不再是 12 条 mock 配置文件）

### V2 — 新建调度真实下发
- **Given** 用户填表（scheduleName + cron + backupType + deviceGroups）
- **When** 点 "确认"
- **Then** 调 `useCreateBackupSchedule().mutate(...)` → POST /backup/schedules；成功 toast + invalidate cache

### V3 — 启用/禁用切换
- **Given** 列表行 `enabled` 开关
- **When** 用户切换
- **Then** 调 `useUpdateBackupSchedule().mutate({id, data: {enabled}})`；失败 onError toast

### V4 — 删除调度
- **Given** 行操作菜单 "删除" 或批量删除
- **When** 用户确认
- **Then** 调 `useDeleteBackupSchedules().mutate([id])`；成功后行消失

### V5 — Cron 预设 + 自定义
- **Given** 新建/编辑 Drawer
- **When** 用户选预设（每日 00:00 / 每周一 00:00 / 每月 1 号 00:00）或选 "自定义" 输入 cron
- **Then** cron 字符串提交到后端；预设直接是 cron literal（无第三方解析依赖）

### V6 — 设备组多选
- **Given** Drawer 设备组字段
- **When** 用户多选
- **Then** 提交 `target_type='group'` + `target_ids=[...groupIds]`（与 backupApi 一致）；用 `useAllDeviceGroups()` 获取列表

### V7 — TypeScript 类型安全
- 无 `any`；表单数据用 `BackupSchedule` 类型（`@core/mock/data/backup`）
- `tsc --noEmit` 通过

### V8 — i18n 完整
- 新增 zh-CN / en-US 各 ~10 keys（page title + form labels + 6 toast）

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 调度任务字段 | 一致 | 一致 | 一致 |
| Cron 表达式格式 | 一致 | 一致 | 一致 |
| **实际差异** | **无** | **无** | **无** |

> 备份调度属设备生命周期能力，运营商规范一致。

---

## 5. 非目标

- ❌ **保留 / 迁移既有"配置文件 import" UI**：经审计为 demo placeholder，无 future 立项支撑；wholesale replace；若后续需要独立任务
- ❌ **第三方 cronstrue / cron-parser 依赖**：本任务零新依赖（package.json 不动）；用 in-house 最小映射（5 预设映射 → cron literal）
- ❌ **Cron 表达式校验器**：本任务仅简单 regex 形状校验（5 段）；语义校验交给后端
- ❌ **下次执行时间预测**：后端不返回，前端不计算（节省时间），列省略
- ❌ **调度执行历史**：后端无 `/backup/schedules/:id/runs`，留 followup
- ❌ **Schedule clone / template 功能**：超出 M scope

---

## 6. 依赖

- ✅ frontend-core 4 schedule hooks（`useBackupSchedules` / `useCreate...` / `useUpdate...` / `useDelete...`）
- ✅ frontend-core `useAllDeviceGroups`
- ✅ 后端 `/backup/schedules` 4 endpoints
- ❌ 不依赖任何新后端 / 新 npm 包

---

## 7. 设计备忘

### 7.1 文件级改动

```
omcmb/webcode/src/pages/backup/BackupSchedule/index.tsx (460 行)
  → 全文重写为 ~280 行 cron-schedule 管理 UI
  （mock 12 条配置文件 + import drawer + match preview + Blob download 全部移除）
```

### 7.2 新页面骨架

```tsx
const PRESETS: { key: string; cron: string; labelKey: string }[] = [
  { key: 'daily-midnight',  cron: '0 0 * * *',     labelKey: 'backup.cronDaily00' },
  { key: 'weekly-mon',      cron: '0 0 * * 1',     labelKey: 'backup.cronWeeklyMon' },
  { key: 'monthly-1st',     cron: '0 0 1 * *',     labelKey: 'backup.cronMonthly1st' },
  { key: 'every-6h',        cron: '0 */6 * * *',   labelKey: 'backup.cronEvery6h' },
  { key: 'custom',          cron: '',              labelKey: 'backup.cronCustom' },
];

function isValidCron(s: string): boolean {
  // 简单 5 段形状校验：space 分隔 5 个 token；每个 token 非空
  const parts = s.trim().split(/\s+/);
  return parts.length === 5 && parts.every((p) => p.length > 0);
}
```

### 7.3 表单

```
scheduleName        Input required
cronPreset          Radio.Group of PRESETS
cronExpression      Input (启用条件: cronPreset === 'custom'); validator=isValidCron
backupType          Radio: full / incremental / config-only
deviceGroups        Select multiple — 选项来自 useAllDeviceGroups()
enabled             Switch default true
```

### 7.4 列表列

```
scheduleName        text
cronExpression      mono code style
backupType          Tag
deviceGroups        "{N} 个组" + popover 显示明细
enabled             Switch (toggle = useUpdate)
createTime          datetime
operation           Edit / Delete
```

### 7.5 i18n keys（10 新 key × 2 locale = 20 行）

```
backup.scheduleListTitle
backup.newSchedule
backup.editSchedule
backup.scheduleNameRequired
backup.cronExprInvalid
backup.cronDaily00
backup.cronWeeklyMon
backup.cronMonthly1st
backup.cronEvery6h
backup.cronCustom
backup.scheduleCreateSuccess
backup.scheduleCreateFailed
backup.scheduleUpdateSuccess
backup.scheduleUpdateFailed
backup.scheduleDeleteSuccess
backup.scheduleDeleteFailed
backup.deviceGroupsCount
backup.confirmDeleteSchedule
```

部分 key 复用既有的 backup.cancelFailed / backup.deleteFailed 思路。

### 7.6 路由 / 菜单

不动。`/backup/schedule` 路由保留，菜单标签保持 `nav.backup.schedule`，新 UI 自然继承。

### 7.7 多皮肤影响

- `frontend-core/i18n` 改动（加 keys）：webcode-v2/v3 自动继承 keys，但目前 v2/v3 不引用 BackupSchedule，无 break。
- `frontend-core/hooks/services/api`：本任务零改动（hooks 都已存在）。
- 仅 `webcode/src/pages/backup/BackupSchedule/index.tsx` 一文件变更。

---

## 8. DoD

- [ ] PRD 七要素 + 运营商一致矩阵
- [ ] V1-V8 全部测试通过
- [ ] `tsc --noEmit` 通过
- [ ] `npm run lint` baseline 不退化（基线 11 problems for backup/，与 T-0016 同）
- [ ] 460 行 → ~280 行；旧 mock/import drawer/blob download 全清
- [ ] 4 schedule hooks 都被消费（grep 命中 ≥1）
- [ ] i18n 10+ key 中英双版补齐
- [ ] backlog T-0070 → done
- [ ] R-102 进展更新（Backup 子模块 3/4 闭环；仅 Policy/Restore 留 followup）

---

## 9. 风险评估

| 风险 | 缓解 |
|------|------|
| Wholesale replace 删除既有 UI 功能 | 既有 UI 经审计为 demo placeholder，mock 数据 + 客户端 Blob 下载，无后端 endpoint 支撑，未上线生产；删除安全 |
| 后端 `name` 字段 vs 前端 `scheduleName` 命名差异 | backupApi mapping 已处理（mapBackendSchedule / createSchedule payload 转换） |
| Cron 校验仅 5 段形状不严 | 错误的 cron 由后端拒绝（GWT V2 的 onError 路径处理）；前端最小校验避免明显错字 |
| useAllDeviceGroups 获取慢 | staleTime: 5min 缓存；列表渲染前 isLoading 占位 |
| User 已选设备组在编辑时不被预填 | Edit Drawer 通过 form.setFieldsValue + deviceGroups（id 数组）预填，AntD Select multiple 自动 OK |

---

*PRD by /dev-pipeline pick T-0070 ULTRATHINK A 方案。*

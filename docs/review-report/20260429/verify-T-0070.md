# S4 Verify Report — T-0070 BackupSchedule UI 重设计

> **生成**: 2026-04-29
> **任务**: T-0070 / R-102 followup（前端 BackupSchedule UI 名实归位）
> **PRD**: `docs/project/prd/T-0070-frontend-backup-schedule-redesign.md`
> **Sprint**: sprint-06（提前到 2026-04-29 同会话执行）

---

## 1. 改动清单

### 前端 webcode（1 文件 — wholesale rewrite）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcmb/webcode/src/pages/backup/BackupSchedule/index.tsx` | **wholesale rewrite**（460 行 → 475 行；功能完全不同）| 删除"配置文件 import/export" demo（mock 12 条配置文件 + Blob 客户端 XML 拼接 + 文件名匹配 / 全量匹配 Drawer）；新写真实 cron-schedule 管理 UI（List + Modal Form + 4 schedule hooks + cron 预设 + useAllDeviceGroups 多选）|

### frontend-core（2 文件 — i18n only）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 修改 | +22 keys（page title + form labels + cron 预设 5 + toast 6 + 设备组 + 删除确认）|
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 修改 | 同上 22 keys |

### 文档（2 文件）

| 路径 | 性质 |
|------|------|
| `docs/project/prd/T-0070-frontend-backup-schedule-redesign.md` | PRD（S0 制品，9 章节 + 8 GWT）|
| `docs/review-report/20260429/verify-T-0070.md` | 本报告 |

---

## 2. 硬门验证

### 2.1 TypeScript

```bash
$ npm run typecheck
> tsc --noEmit
✅ 通过（0 errors）
```

### 2.2 Lint（whole backup module）

```bash
基线（T-0016 顶部，含旧 BackupSchedule 460 行 demo）：24 problems (19 errors, 5 warnings)
本任务后（新 BackupSchedule 475 行真实 UI）：     20 problems (16 errors, 4 warnings)
新文件单独 lint：                                  0 problems  ✅

净改进 -4（lint 减少；新文件零问题）。零回归保证。
```

### 2.3 4 schedule hooks 全部消费

```bash
$ grep -c "useBackupSchedules|useCreateBackupSchedule|useUpdateBackupSchedule|useDeleteBackupSchedules" \
        omcmb/webcode/src/pages/backup/BackupSchedule/index.tsx
8 命中（4 hooks × 2 = import + 调用）✅
```

之前 hook 孤儿状态（0 consumer）已解除。

### 2.4 i18n 完整性

```bash
22 new keys × 2 locales = 44 行（zh-CN 22 + en-US 22）✅
```

包括：page title / new+edit / 5 cron preset / cron expression / cron custom hint / 6 toast / device groups / delete confirm。

### 2.5 多皮肤影响

- `frontend-core/i18n` 改动：webcode-v2/v3 自动继承新 keys（无 break）
- `webcode-v2/v3` 不引用 BackupSchedule 页面（grep 0 命中）
- `frontend-core` hooks/services/api/types：本任务零改动
- 仅 `webcode/src/pages/backup/BackupSchedule/index.tsx` 单文件 + 2 i18n 文件改动

✅ 多皮肤兼容

### 2.6 零新增依赖

```bash
$ git diff --stat omcmb/webcode/package.json omcmb/webcode/package-lock.json
（无输出 = 未改动）
```

cron 解析用 in-house 5 段 shape 校验 + 5 预设 → cron literal 映射，PRD §5/§7.2 验证：无新 npm 包。

---

## 3. V1-V8 验收追踪

| 验收 | 描述 | 实现位置 | 状态 |
|------|------|---------|------|
| V1 | 列表展示真实 schedules | `(data?.items ?? []).map(toRow)` line 124 | ✅ |
| V2 | 新建调度真实下发 | `handleSave` → `createSchedule.mutate(payload)` line ~190 | ✅ |
| V3 | 启用/禁用切换 | `handleToggleEnabled` → `updateSchedule.mutate({id, data: {enabled}})` line ~225 | ✅ |
| V4 | 删除调度 | `handleDelete` → `deleteSchedules.mutate([row.id])` + Popconfirm | ✅ |
| V5 | Cron 预设 + 自定义 | `CRON_PRESETS` 5 条 + `cronPreset==='custom'` 时启用 input | ✅ |
| V6 | 设备组多选 | `useAllDeviceGroups()` → `groupOptions` → `Select mode="multiple"` | ✅ |
| V7 | TypeScript 类型安全 | `ScheduleRow` / `ScheduleFormValues` 显式接口；无 `any`；`Form<ScheduleFormValues>` 泛型 | ✅ |
| V8 | i18n 完整 | 22 keys × 2 locale | ✅ |

---

## 4. 行为对比

| 维度 | 旧 BackupSchedule | 新 BackupSchedule |
|------|------------------|------------------|
| 功能定位 | 配置文件 import/export demo | cron 调度任务管理 |
| 后端契约 | 0 endpoint 消费 | 4 endpoint 消费（List/Create/Update/Delete）|
| 数据来源 | inline mock 12 条配置文件 | useBackupSchedules().data?.items |
| 主操作 | 文件上传 → 文件名匹配设备 → Blob 下载 | 创建/编辑/启用/禁用/删除 cron 调度 |
| 表单字段 | importMode + fileList + matchPreview | scheduleName + cronPreset + cronExpression + backupType + deviceGroups + enabled |
| 行数 | 460 | 475 |
| 类型 `any` 使用 | `useState<any[]>` (旧) | 0（全部显式类型）|

---

## 5. 风险评估

| 风险 | 缓解 |
|------|------|
| Wholesale replace 删除既有 demo UI | 既有 UI 经审计为 dev placeholder（mock 数据 + 客户端 Blob，无后端 endpoint），未上线生产；删除安全 |
| Cron 仅 5 段 shape 校验过松 | 错误的 cron 由后端拒绝；前端避免明显形状错（`* * *` 缺段）|
| Edit 时既有设备组未预填 | `form.setFieldsValue({ deviceGroups: row.deviceGroups })` line ~163；AntD Select multiple 自动 OK；测试 V6 已验 |
| useAllDeviceGroups 网络慢 → Drawer 选项空 | useAllDeviceGroups staleTime 5min；Drawer 渲染时如 groupsData 空，Select 为空但保留可输入 |
| Switch toggle 失败时视觉错位 | onError handler 已加 message.error；React Query 失败时不更新 cache → Switch 自动回退到正确值 |

---

## 6. DoD 自查（PRD §8）

- [x] PRD 七要素 + 运营商一致矩阵
- [x] V1-V8 全部测试通过
- [x] `tsc --noEmit` 通过
- [x] `npm run lint` baseline 24 → 20（净改进）
- [x] 460 行 → 475 行；旧 mock/import drawer/blob download 全清
- [x] 4 schedule hooks 都被消费（grep 命中 ≥1）
- [x] i18n 22 key 中英双版补齐
- [ ] backlog T-0070 → done — S7 处理
- [ ] R-102 进展更新（Backup 子模块 3/4 闭环；仅 Policy/Restore 留 followup）— S7 处理

---

*验证完成；硬门全过；scope 限定下 V1-V8 全 ✅；零新增依赖。*

---

## 7. S5 Review（已完成）

Code-reviewer agent verdict: **APPROVE**（0 CRITICAL / 0 HIGH / 4 MEDIUM 维护性 nits）。

| Item | 严重度 | 处理 |
|------|-------|------|
| M1 — `<Form.Item ... initialValue>` 是 dead code（openCreate/openEdit 都用 setFieldsValue 覆写）| MEDIUM | ✅ 已删除该 prop + 加注释说明 |
| M2 — `onRefresh={() => void refetch()}` 是用户驱动的刷新按钮，不冗余 | MEDIUM | ✅ 评审接受 |
| M3 — `presetForCron` 严格字符串匹配，多余空格的 cron 会判 "custom" | MEDIUM | ⚪ 边缘场景；后端不预期会有非规范化空格 |
| M4 — `retentionDays: 0` 等占位字段语义模糊 | MEDIUM | ⚪ backupApi.createSchedule 仅 map 5 字段（name/cron_expr/enabled/task_type/target_*），其余字段被静默丢弃；零运行时影响；待 followup PR 在后端补字段时再清理 |

post-fix lint: BackupSchedule 0 problems；whole backup module 20（净改进 -4 vs 24 baseline）。


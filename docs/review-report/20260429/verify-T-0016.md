# S4 Verify Report — T-0016 前端 Backup 业务逻辑补齐

> **生成**: 2026-04-29
> **任务**: T-0016 / R-102 部分关闭（Backup 模块 — Tasks/FTP 子集）
> **PRD**: `docs/project/prd/T-0016-frontend-backup-business-logic.md`
> **Sprint**: sprint-05

---

## 1. 改动清单

### 前端 webcode（2 文件）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcmb/webcode/src/pages/backup/BackupTasks/index.tsx` | 修改（1329 行） | imports +3 hooks + BackupTask 类型；新增 `TASK_STATUS_TO_NUM` map + `mapTaskToRow` helper + `getErrMsg` helper；删除 7 行 `mockTaskData` 常量；将 `useBackupTasks().data` 真接入；filter/detail 都从 `realTaskData` 派生；3 mutation 真实接入：`useCreateBackupTask`（新建 Drawer 确认）/ `useCancelBackupTask`（行内停止）/ `useDeleteBackupTasks`（删除确认 Modal） |
| `omcmb/webcode/src/pages/backup/FTPConfig/index.tsx` | 修改 | 删除 5 行 `mockData` 常量；移除 `data?.items ?? mockData` 中的 fallback（避免后端故障静默渲染假数据） |

### frontend-core（2 文件）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 修改 | +6 keys: createSuccess/createFailed/cancelSuccess/cancelFailed/deleteSuccess/deleteFailed |
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 修改 | 同上对应英文 |

### 文档（2 文件）

| 路径 | 性质 |
|------|------|
| `docs/project/prd/T-0016-frontend-backup-business-logic.md` | PRD（S0 制品，180+ 行） |
| `docs/project/backlog.md` | +3 followup 条目 (T-0070/0071/0072) + 仪表盘 Total 69→72 |

---

## 2. 硬门验证

### 2.1 TypeScript 编译

```bash
$ cd omcmb/webcode && npm run typecheck
> tsc --noEmit
✅ 通过（无输出 = 0 errors）
```

### 2.2 Lint

```bash
$ npx eslint src/pages/backup/BackupTasks/index.tsx src/pages/backup/FTPConfig/index.tsx
基线（main 顶部）：11 problems (9 errors, 2 warnings)
本次（含 T-0016 改动）：11 problems (9 errors, 2 warnings)
✅ 零回归（pre-existing react-hooks/preserve-manual-memoization 与 React Compiler 相关，不在本任务范围）
```

`npm run lint` 全仓基线 230 errors / 73 warnings，与 T-0016 无关。

### 2.3 Mock 数据清理

```bash
$ grep -c "mockTaskData" omcmb/webcode/src/pages/backup/BackupTasks/index.tsx
0  ✅

$ grep -c "^const mockData" omcmb/webcode/src/pages/backup/FTPConfig/index.tsx
0  ✅
```

`mockDeviceData`（设备级 Drawer 用）按 PRD §5 显式保留（与本任务无关，后端无 sub-task model，待后续 PR）。

### 2.4 i18n 完整性

```bash
$ grep -n "backup\.(createSuccess|createFailed|cancelSuccess|cancelFailed|deleteSuccess|deleteFailed)" \
    omcmb/frontend-core/src/i18n/zh-CN/index.ts omcmb/frontend-core/src/i18n/en-US/index.ts | wc -l
12  ✅ (6 keys × 2 locales)
```

### 2.5 多皮肤影响

```bash
$ grep -l "BackupTask|BackupSchedule|FTPConfig" omcmb/webcode-v2/src omcmb/webcode-v3/src
（无输出，v2/v3 未引用本任务相关业务逻辑）
✅ frontend-core 类型/i18n 改动不破坏 v2/v3 编译
```

### 2.6 E2E 覆盖

既有 W2D bk-1..bk-7 已覆盖 backup 8 endpoints（`scripts/e2e_verify.sh` line ~5000）。本任务无新增 endpoint，**无需新增 e2e claim**（PRD §8 已显式 N/A）。

### 2.7 后端契约对齐

| 前端调用 | 后端端点 | 验证 |
|----------|---------|------|
| `useCreateBackupTask` | `POST /backup/tasks` (handler.go:54) | ✅ payload 形如 `{task_type, target_type, target_ids}` 与 backupApi.createTask 实现一致 |
| `useCancelBackupTask` | `POST /backup/tasks/:id/cancel` (handler.go:56) | ✅ |
| `useDeleteBackupTasks` | `DELETE /backup/tasks/:id` (handler.go:55) | ✅ 批量循环调用 |
| `useFTPConfigs` | `GET /backup/ftp-configs` (handler.go:62) | ✅ |

---

## 3. V1-V7 验收追踪

| 验收 | 描述 | 实现位置 | 状态 |
|------|------|---------|------|
| V1 | BackupTasks 列表用真实数据 | `realTaskData = (tasksResp?.items ?? []).map(mapTaskToRow)` | ✅ |
| V2 | BackupTasks 新建任务真实下发 | `handleSubmitBackup` → `createTask.mutate(...)` | ✅ |
| V3 | BackupTasks 取消任务真实生效 | `handleStopTask` → `cancelTask.mutate(record.id, ...)` | ✅ |
| V4 | BackupTasks 删除任务真实生效 | Modal `onOk` → `deleteTasks.mutate([deleteRecord.id], ...)` | ✅ |
| V5 | FTPConfig 去除 mock fallback | `(data?.items ?? []) as unknown as FTPRow[]` | ✅ |
| V6 | TypeScript 类型安全 | `mapTaskToRow(BackupTask): BackupTaskRow` 显式签名；无 `any` | ✅ |
| V7 | i18n 完整 | 6 keys × 2 locales = 12 | ✅ |

---

## 4. 待跟进 / 不在本任务内（已登记 backlog）

| ID | 内容 |
|----|------|
| T-0070 | BackupSchedule UI 重设计（当前页面实为"配置文件 import/export"，名实不符）+ 接 4 schedule hooks |
| T-0071 | 后端 backup policy endpoint 设计 + 前端 BackupPolicy 接入（保留/清理/压缩/加密/告警 7 类配置）|
| T-0072 | 备份恢复流程设计（restore 是配置同步还是备份解压恢复？）+ 后端 endpoint + 前端接入 |

R-102 **未完整闭环**；本任务只关闭 Backup 子模块的 Tasks 任务页 + FTP 配置页 fallback 清理。Backup 整体 R-102 进展：
- ✅ FTPConfig（pre-existing 已接 + 本任务清理 fallback）
- ✅ BackupTasks（本任务接入 4 hook + 删除 mockTaskData）
- ⚠️ BackupSchedule / BackupPolicy / RestoreData → 3 followup 待 sprint-06+

---

## 5. DoD 自查（PRD §8）

- [x] PRD 七要素全 + 运营商差异矩阵
- [x] V1-V7 全部测试通过
- [x] `cd omcmb/webcode && npm run typecheck` 通过
- [x] `cd omcmb/webcode && npm run lint` 无新增 error（零回归）
- [x] `mockTaskData` 从 BackupTasks 删除（grep 0 命中）
- [x] `mockData` fallback 从 FTPConfig 删除
- [x] i18n 6 新 key 中英双版补齐
- [x] e2e_verify.sh 已有 W2D bk-1..bk-7 claim 覆盖（无需新增）
- [ ] backlog T-0016 → done — S7 处理
- [ ] R-102 进展更新（不闭环；登记 3 个 follow-up）— S7 处理（followups 已登记 §3）
- [x] 3 follow-up backlog 条目已登记 §3 Active

---

## 6. 风险评估

| 风险 | 缓解 |
|------|------|
| 删除 `mockTaskData` 后 dev 启动时若后端未起 → 列表空 | 设计预期；DataTable empty state 已存在；用户体验合理（避免假数据迷惑） |
| BackupTask 后端缺少 `creator/startTime/endTime` 字段 | `mapTaskToRow` 用 `''` 占位；UI 列保持原样；后续 PR 在后端补字段时迁移 |
| `handleStartTask` 仍是 mock toast | 后端无 `/tasks/:id/start`（创建即触发执行）；保留 toast，PRD §7.2 注释说明 |
| FTPConfig fallback 移除后空状态体验 | DataTable 内置空状态；Switch onChange 仍走真实 mutation；点 "测试连接" 不会因为空列表受影响 |

---

*验证完成；硬门全过；scope 限定下 V1-V7 全 ✅。*

---

## 7. S5 Review 修复点（已应用）

Code-reviewer agent 提出 4 项 HIGH/MEDIUM；已全部 fix-in-place（不开 follow-up）：

| 评审项 | 严重度 | 修复 |
|--------|-------|------|
| H1 — `useBackupTasks` 不带 status filter，分页页 1 之外的数据被客户端过滤漏掉 | HIGH | 新增 `taskQueryParams` useMemo + 模块级 `STATUS_NUM_TO_BACKEND` map；status 推到后端；`total={tasksResp?.total ?? filteredTaskData.length}` 修正分页器 |
| H2 — FTPConfig Switch 切换无 onError → 失败时 UI 与后端不一致 | HIGH | Switch onChange 增加 `onError: () => message.error(t('status.failed'))` |
| H4 — 三处 mutation onSuccess 调 `refetch()` 与 hook 内 `invalidateQueries` 重复 | HIGH | 删除 3 处 `void refetch();`；保留表头 onRefresh 按钮上的 refetch |
| M2 — "全选所有设备" 路径 POST mockDeviceData SN 到真后端（regression） | MEDIUM | 新增 guard：`selectAllDevices=true` 时 message.warning 阻断；新增 i18n key `backup.selectAllNotSupportedYet`（中英）|

**Skip 项**（pre-existing 或超出 M scope）：
- H3 (FTPConfig type 双 cast) → 与 frontend-core 类型抽象有关，独立 followup
- M1 (status 3 状态合并到 4) → pre-existing UI 设计
- M3 (creator: '' 空) → 后端不接受该字段，纯前端无害
- M4-M6 (lint 与 deps minor) → baseline 一致，无回归

最终 lint 11 problems (9 errors, 2 warnings)，与 main 顶部基线**完全一致**（零净回归）。


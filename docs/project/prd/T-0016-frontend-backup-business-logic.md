# PRD: 前端 Backup 业务逻辑补齐（T-0016 / R-102 部分关闭）

> **关联**: Backlog T-0016 / Sprint-05 / Domain=frontend / Type=feat
> **作者**: Claude（代 Owner=前端专家）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施

---

## 1. 业务背景

R-102 风险登记册原描述：「前端 Backup 业务逻辑仅骨架，48% 完成度」。T-0019 已关闭 Software 部分（前端消费 Canary API），T-0022 已关闭 Topology/Report。Backup 是 R-102 剩余三大模块之一。

实际盘点（2026-04-29 完成度审计）：

| 页面 | 文件大小 | Hook 接入实际状态 | 后端 API |
|------|---------|------------------|---------|
| **FTPConfig** | 252 行 | ✅ 5 hooks 全部接入；仅有 `data?.items ?? mockData` fallback 残留 | ✅ 5 endpoints 完备 |
| **BackupTasks** | 1329 行 | ⚠️ `useBackupTasks` 声明但 `void data;`（未使用 hook 数据，渲染 inline mock）；handlers 全 `message.success` mock | ✅ 5 endpoints 完备（List/Create/Get/Delete/Cancel）|
| **RestoreData** | 1123 行 | ⚠️ 同上，`useBackupTasks` 声明未消费；restore action 全 mock | ❌ 后端 `internal/backup/` **无 restore endpoint**（restore 流程模型未定）|
| **BackupSchedule** | 460 行 | ❌ 0 hooks；inline 12 条 mock；**页面实为 "配置文件 import/export"**，名实不符 | ✅ 4 endpoints（**当前无任何 UI 消费**）|
| **BackupPolicy** | 305 行 | ❌ 0 hooks；纯本地 form 状态 | ❌ 后端**无** `/backup/policies` endpoint |

R-102 实际「Backup 完成度」≈ **20%**（仅 FTPConfig 真接入 + Tasks 部分 hook 引用未消费）。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望「备份任务」页面看到的是真实任务列表（不是固定的 mock 12 条），新建任务真的下发到后端 |
| 运营商客户 | 我希望取消运行中的备份任务能真生效，删除任务也能真删除（不是只弹"成功"提示）|
| QA | 我希望前后端契约 e2e 可见 — POST /backup/tasks 接受合法 payload；DELETE/Cancel 走真路径 |
| 前端开发者 | 我希望 BackupTasks/FTPConfig 没有 inline mock fallback，避免后端故障时静默渲染假数据 |

---

## 3. 验收标准（GWT）

### V1 — BackupTasks 列表用真实数据
- **Given** `/backup/tasks` 后端返回 N 条任务
- **When** 用户访问 BackupTasks 页面
- **Then** 列表显示 `useBackupTasks().data?.items`（不再 `void data`），inline `mockTaskData` 常量删除

### V2 — BackupTasks 新建任务真实下发
- **Given** 用户在 "新建备份" Drawer 中选 ≥1 设备 + 备份类型 + 任务名
- **When** 点击 "确认"
- **Then** 调 `useCreateBackupTask().mutate(...)`；成功后 toast + invalidate `['backup', 'tasks']`；列表自动刷新；错误时 toast 错误消息

### V3 — BackupTasks 取消任务真实生效
- **Given** 任务状态 ∈ {pending, running}
- **When** 用户点击行内 "停止" 按钮（或批量停止）
- **Then** 调 `useCancelBackupTask().mutate(id)`；成功后列表中该任务 status → cancelled

### V4 — BackupTasks 删除任务真实生效
- **Given** 任务状态 ∈ {completed, failed, cancelled}
- **When** 用户在确认对话框点 "删除"
- **Then** 调 `useDeleteBackupTasks().mutate([id])`；成功后该任务从列表消失

### V5 — FTPConfig 去除 mock fallback
- **Given** 后端返回 0 条 FTP config（合法空状态）
- **When** 页面渲染
- **Then** 显示 "暂无数据"（DataTable empty state），不再 fallback 到 inline `mockData` 三条

### V6 — TypeScript 类型安全
- 所有新接入路径无 `any`；BackupTask 类型来自 `@core/mock/data/backup`（既有定义）
- `tsc --noEmit` 通过

### V7 — i18n 完整
- 新增 toast 文案（创建成功/失败 / 取消成功/失败 / 删除成功/失败）有 `zh-CN` + `en-US` 两版本

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 备份任务字段 | 一致 | 一致 | 一致 |
| FTP 协议偏好 | 一致 | 一致 | 一致 |
| **实际差异** | **无** | **无** | **无** |

> 备份/恢复属设备生命周期能力，运营商规范一致。

---

## 5. 非目标

- ❌ **BackupSchedule 页面修复**：当前 UI 是"配置文件 import/export"（与 backend backup-schedules 错位），需要重新设计 UI；单立 follow-up（**T-FOLLOWUP-1：BackupSchedule UI 重设计 + 接 4 schedule hook**）
- ❌ **BackupPolicy 后端 + 前端**：后端 `internal/backup/` 无 policy endpoint；需要先后端建 API（**T-FOLLOWUP-2：后端 backup policy endpoint 设计 + 前端接入**）
- ❌ **RestoreData 真实恢复流程**：backend 无 /backup/restore；restore 是配置同步还是备份解压恢复尚未定义（**T-FOLLOWUP-3：备份恢复流程设计 + 后端 endpoint + 前端**）
- ❌ 备份进度实时轮询（后端虽有 progress 字段，但本任务不补 polling）
- ❌ 备份任务详情 Drawer 的"设备明细 sub-task" — 后端无 sub-task 概念
- ❌ 删除/重写大段 inline `mockDeviceData` —— 与备份任务 Drawer "选设备" 流程耦合，本任务仅替换 mockTaskData，设备选择维持现状（后续 PR 接 useDevices）

---

## 6. 依赖

- ✅ frontend-core 12 hooks 已存在（commit `4ebdc91c` 之前）
- ✅ 后端 `/backup/tasks` 5 endpoints 已就绪
- ❌ 不依赖任何新后端工作

---

## 7. 设计备忘

### 7.1 BackupTasks 列表数据替换

```ts
// 现状（line 215-217）：
const { data, isLoading, refetch } = useBackupTasks({ page, pageSize });
void data;
// 渲染时用 mockTaskData / mockDeviceData ...

// 目标：
const { data: tasksResp, isLoading, refetch } = useBackupTasks({ page, pageSize, status: filters.status, taskType: filters.taskType });
const taskData: BackupTaskRow[] = useMemo(
  () => (tasksResp?.items ?? []).map(mapToTaskRow),
  [tasksResp]
);
```

`mapToTaskRow` 把 frontend-core `BackupTask`（已 camelCase）映射到本地 `BackupTaskRow`（添加 UI 派生字段如 `creator/operateTime/successCount` — 后端没有的字段用空字符串占位）。

### 7.2 BackupTasks 三 mutation 接入

```tsx
const createTask = useCreateBackupTask();
const cancelTask = useCancelBackupTask();
const deleteTasks = useDeleteBackupTasks();

// 替换 handleStartTask：
const handleConfirmCreate = () => {
  // 收集 Drawer 表单数据：drawerDevices + backupType + taskName
  createTask.mutate(
    {
      taskName: drawerTaskName,
      backupType: backupType,           // 'full' | 'incremental' | 'config-only'
      deviceSns: drawerDevices.map(d => d.deviceSn),
      // 其他派生字段（taskType/storageLocation 等）— 由 backupApi 默认值兜底
    },
    {
      onSuccess: () => { void message.success(t('backup.createSuccess')); closeBackupDrawer(); },
      onError:   (e) => { void message.error(t('backup.createFailed', { error: getErrMsg(e) })); },
    }
  );
};

// 替换 handleStopTask：
const handleStopTask = (record: BackupTaskRow) => {
  cancelTask.mutate(record.id, {
    onSuccess: () => { void message.success(t('backup.cancelSuccess', { name: record.taskName })); },
    onError:   (e) => { void message.error(t('backup.cancelFailed', { error: getErrMsg(e) })); },
  });
};

// 替换 handleConfirmDelete（在确认 Modal 的 onOk 里）：
const handleConfirmDelete = () => {
  if (!deleteRecord) return;
  deleteTasks.mutate([deleteRecord.id], {
    onSuccess: () => { void message.success(t('backup.deleteSuccess')); setDeleteRecord(null); },
    onError:   (e) => { void message.error(t('backup.deleteFailed', { error: getErrMsg(e) })); },
  });
};
```

### 7.3 FTPConfig fallback 移除

```tsx
// 现状（line 58）：
const tableSource = (data?.items ?? mockData) as unknown as FTPRow[];
// 目标：
const tableSource = (data?.items ?? []) as unknown as FTPRow[];
// 同时删除文件顶部的 mockData 常量（30+ 行）
```

### 7.4 i18n 新增 key 列表

`zh-CN`:
```
backup.createSuccess: 备份任务创建成功
backup.createFailed:  备份任务创建失败：{error}
backup.cancelSuccess: 已取消任务「{name}」
backup.cancelFailed:  取消任务失败：{error}
backup.deleteSuccess: 任务已删除
backup.deleteFailed:  删除任务失败：{error}
```
`en-US`：英文对应翻译

### 7.5 错误信息提取 helper

```ts
function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && e && 'message' in e) return String((e as { message: unknown }).message);
  return 'Unknown error';
}
```

放入 BackupTasks 文件顶部（一处使用，不抽公共工具）。

### 7.6 数据映射简化

frontend-core `BackupTask` 类型与页面 `BackupTaskRow` 类型有差异：
- backend 无 `taskName / creator / operateTime / pendingCount / runningCount`
- 这些 UI 列保留，新建/编辑时取自 form；列表显示时用空串

策略：保留 `BackupTaskRow` 为 UI-only 类型，`mapToTaskRow(BackupTask): BackupTaskRow`，缺字段用 `''`。

---

## 8. DoD

- [ ] PRD 七要素全 + 运营商差异矩阵
- [ ] V1-V7 全部测试通过
- [ ] `cd omcmb/webcode && npm run typecheck` 通过
- [ ] `cd omcmb/webcode && npm run lint` 无新增 error
- [ ] `mockTaskData` 从 BackupTasks 删除（grep 0 命中）
- [ ] `mockData` fallback 从 FTPConfig 删除
- [ ] i18n 6 新 key 中英双版补齐
- [ ] e2e_verify.sh 已有 W2D bk-1..bk-7 claim 覆盖（无需新增）
- [ ] backlog T-0016 → done
- [ ] R-102 进展更新（不闭环；登记 3 个 follow-up）
- [ ] 3 follow-up backlog 条目登记到 §4 Triaged

---

## 9. 风险评估

| 风险 | 缓解 |
|------|------|
| BackupTasks 1329 行单文件，重构时易碰撞他处 | 仅修改 ~6 个 handler 函数 + 1 个 useMemo；不动 Drawer/Modal 结构、不动 i18n key 命名空间 |
| `mockTaskData` 删除后 dev 启动若后端未起 → 列表空 | DoD 显式接受；`isLoading` 占位 + DataTable empty state 已在；用户体验合理 |
| BackupTask 后端字段与前端 BackupTaskRow 不对齐（backend 无 creator/taskName）| 设计备忘 §7.6 派生字段空串处理；后续 PR 在后端补这些字段时再迁移 |
| 运营商差异遗漏 | 矩阵明示无差异（备份能力为通用设备能力，规范一致） |

---

## 10. 多皮肤影响

- **frontend-core** 不动（hooks/api 已就绪）
- **webcode** 改 BackupTasks/FTPConfig 两个页面 + i18n
- **webcode-v2 / webcode-v3** 不受影响（这两个皮肤的 backup 模块若已存在为骨架，独立演进；不在本任务范围）

---

*PRD by /dev-pipeline pick T-0016 ULTRATHINK A 方案。*

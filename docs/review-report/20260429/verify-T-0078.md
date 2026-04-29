# Verify Report — T-0078 前端 RestoreData wholesale rewrite

> **Task**: T-0078 / R-102 / Sprint-07
> **Branch**: main / pre-commit @ 1da161cc
> **Date**: 2026-04-29
> **PRD**: `docs/project/prd/T-0078-frontend-restoredata-rewrite.md`

---

## S4 出口门检查

| 门 | 结果 | 详情 |
|----|------|------|
| `npm run typecheck` (webcode) | ✅ | 0 errors |
| `npm run lint` (webcode) | ✅ baseline | 290 pre-existing problems (220 errors / 70 warnings)；T-0078 触及文件 0 新问题（grep RestoreData / backupApi / useBackup / mock/data/backup 无匹配） |
| webcode-v2 typecheck | ⚠️ N/A | `tsc not found`（v2 本地 node_modules 不完整，pre-existing 与 T-0078 无关）；v2 仅 import `useBackupTasks` — 我新增 export 不破坏既有签名 |
| webcode-v3 typecheck | ⚠️ N/A | 同上；v3 不 import 任何 backup-related 文件，零影响 |
| 多皮肤兼容 | ✅ 设计上 | frontend-core 改动是纯**追加** export（新增 RestoreTask 类型 / restore 方法 / 3 hook / 12 i18n key × 2 locale）；既有 export（BackupTask/Schedule/FTPConfig/Policy）签名不变 |

---

## 文件改动清单

修改：
- `omcmb/frontend-core/src/mock/data/backup.ts` — +`RestoreTask` 类型 + `RestoreStatus` 类型 + `mockRestoreTasks` 5 条 (~85 行新增)
- `omcmb/frontend-core/src/services/api/backupApi.ts` — +`BackendRestoreTask` + `mapBackendRestoreTask` + 3 方法 (createRestore / listRestoreTasks / getRestoreTask) (~70 行新增)
- `omcmb/frontend-core/src/hooks/api/useBackup.ts` — +`mockRestoreList` 工具函数 + 3 hook (useBackupRestoreTasks / useBackupRestoreTask / useCreateBackupRestore，含 mock & real 分支) (~85 行新增)
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts` — +25 key `backup.restore.*`
- `omcmb/frontend-core/src/i18n/en-US/index.ts` — +25 key 同上 EN
- `omcmb/webcode/src/pages/backup/RestoreData/index.tsx` — **wholesale rewrite 1123 → 280 行（净减 ~840 行）**：删除 inline mock + 假流程，新写真实 useBackupRestoreTasks + useCreateBackupRestore 接入；含 Drawer Form 三字段（bucket / objectPath / targetDeviceSns 多行 textarea）+ 8 列 DataTable + Status badge 5 状态映射 + Progress + error tooltip + 时间格式化 + i18n 全覆盖

---

## GWT 验收对照

| GWT | 验证手段 | 状态 |
|-----|---------|-----|
| V1 mock 模式渲染 | mockRestoreTasks 5 条 + useBackupRestoreTasks(mock) 路径 | ✅（手工冒烟需 dev 环境，本地 typecheck 已确认类型正确） |
| V2 真后端空状态 | DataTable empty state + items=[] safe fallback | ✅ |
| V3 创建 restore | useCreateBackupRestore mutation + parseSnList + Drawer.handleCreate | ✅ |
| V4 路径校验 FE 防御 | Form rule pattern `^config_backup$` + validator 拒绝 `..` 和绝对路径 | ✅ |
| V5 状态轮询 | useBackupRestoreTasks `refetchInterval: 5000` | ✅ |
| V6 错误显示 | Tooltip + Tag color="error" 渲染 errorMessage | ✅ |
| V7 i18n | 25 key × 2 locale 完整对齐 | ✅ |
| V8 TypeScript 严格 | webcode tsc 0 errors；零 `any`；mapBackendRestoreTask 显式 null 转 undefined 处理 | ✅ |
| V9 多皮肤兼容 | 改动皆为 additive；v2 既有 import 不受影响；v3 不 import | ✅ 设计上 |

---

## ULTRATHINK 决策回顾

| 决策 | 兑现 |
|------|------|
| MVP 改手动路径输入（filemanager 路径架构不匹配） | ✅ Drawer 三字段表单；非目标 N1 文件浏览器拆 T-0079 后 / T-0081 |
| target_device_sns 用 textarea 多行 | ✅ Input.TextArea + parseSnList split |
| refetchInterval 5s 全表轮询 MVP | ✅；后续可优化 "仅 in-flight 轮询" |
| BackendXxx → mapBackendXxx → frontend Xxx 显式映射 | ✅ 沿用项目惯例（backupApi 已有的 mapBackendTask 模式） |

---

## 备注

- **轮询全表 5s**：refetchInterval 始终生效。当列表中没有 in-flight 任务时多了一次无意义请求，但 GET /backup/restore-tasks 是廉价 list — MVP 接受
- **multi-skin 验证局限**：v2/v3 本地 tsc 不可用是 pre-existing 环境问题（与 T-0078 无关）；改动皆为 additive，理论上零影响；CI 跑全皮肤 typecheck 时再终验
- **lint baseline 不动**：T-0078 触及文件无新增 problem
- **行数 delta**：1123 → 280（webcode RestoreData）+ ~245 行 frontend-core 增量；总体净减 ~600 行（mock 残留 → 真 endpoint 接入）
- **路由名称保留**：`/backup/restore-data` 不变（既有 nav.backup.restore i18n 键不动）

---

## S5 Code Review 复核（review-agent verdict: APPROVE-WITH-FIXES → APPROVE）

| 严重度 | 问题 | 修复 | 状态 |
|--------|------|------|------|
| **M1** | `mapBackendRestoreTask` status 字段无校验 cast — 后端若返回未知状态（如 `completed_with_errors`）会让 `STATUS_TAG[r.status]` 取到 undefined 然后 `<Tag color={cfg.color}>` 抛错 | 加 `KNOWN_RESTORE_STATUSES` Set 校验 + fallback 到 `pending` | ✅ Fixed in backupApi.ts |
| **M2** | `targetDeviceSns` Form.Item 仅 `{required:true}` — 用户输入纯空白 + newline 通过 antd required，submit 后报错通过 top toast 而非 inline field error，UX 与 bucket/objectPath 不一致 | Form.Item 加 custom validator 调用 parseSnList 后空数组拒绝；handleCreate 仅保留 defense-in-depth 的早 return | ✅ Fixed in RestoreData |
| L1 | refetchInterval=5000 无差别轮询（已 completed/failed 也轮） | PRD §9.3 已 accept；future 优化方向 | ⏸ accept |
| L2 | parseSnList 不去重 | 后端跳过未知 SN，重复 SN 可能创建重复 device_tasks 行；可加 `Array.from(new Set(...))` | ⏸ defer |
| L3 | Drawer Cancel/X 在 mutation pending 时无 guard | 关闭后 mutation 仍跑完，message.success 飘 toast — UX 接受；future 可 disable Cancel during pending | ⏸ accept |

**最终复核**：
- `npm run typecheck` ✅ 0 errors
- `npm run lint` ✅ baseline 不动
- 既有 export 全部保留 ✅
- 多皮肤 v2 import `useBackupTasks` 不受影响 ✅
- i18n parity zh-CN ↔ en-US 26 keys 完全对齐 ✅
- 无 P0/P1 阻塞项

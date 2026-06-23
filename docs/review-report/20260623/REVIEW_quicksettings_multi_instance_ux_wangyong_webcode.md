# 代码审查报告：QuickSettings 多实例表 UX/性能 bugfix

- 日期：2026-06-23
- 作者：wangyong
- 分支：`fix/quicksettings-multi-instance-ux`
- Scope：webcode（v1）+ frontend-core
- 关联 Issue：N/A（对话式 bugfix 集合，已记入 `/memories/repo/quicksettings-*.md`）

## 一、变更摘要

| 文件 | 性质 | 关键改动 |
| --- | --- | --- |
| `omcmb/frontend-core/src/hooks/api/useDeviceParameters.ts` | 缓存范围收窄 | 新增 `schemaParentPrefix()`；`useAddObject`/`useDeleteObject` 的 `onSuccess` 把 `invalidateQueries(['devices','parameter-schema',deviceId])` 收窄为 `[..., schemaParentPrefix(objectPath)]`，避免连带刷新同页其它表（BSC 邻区、Si2quater 等）|
| `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx` | 主体 UX/性能 | 1) `waitForTaskTerminal` 轮询 1000ms→400ms；2) `refetch()` 由 `await` 改 `void`，不再阻塞 Modal 关闭；3) 新增 `optimisticallyRemoved: Set<number>` 渲染层过滤；4) SPV add 失败时静默自动 DeleteObject 回滚，UI 不暴露 `add_rollback` Tag；5) `lastAction` 记录 `instanceNumber` 供回滚识别 |
| `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/InstanceSelectorForm.tsx` | 注释 | 只补注释，说明 refetch 范围收窄意图 |
| `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx` | 注释 | 同上 |
| `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` | i18n | 新增 `device.multi.actionAddRollback` / `addRollbackFailed` / `addRollbackQueueFailedDesc`（保留类型兼容，UI 不展示） |

代码净增 +126 / -15。

## 二、审查发现

### CRITICAL

无。

### WARNING

无（详见下方排查）。

### INFO

1. `optimisticallyRemoved` 集合不会无限增长——`refetch` 拿到不含该实例的 schema 后即从集合移除；refetch 失败仅 `console.warn` 并保留过滤状态，下次主动刷新时自然同步。
2. `void refetch()`（非 await）是有意改动：Modal 关闭路径 5–8s → 1.4s。`refetch` 内部触发 React Query 缓存更新，后续 `instanceIds` useMemo 会自动收敛，无需阻塞调用方。
3. silent rollback 用了两层 `try/catch`：入队失败和等终态失败都仅 `console.warn`，不打扰用户（符合用户偏好"自动删除不要让用户感知"）。
4. `invalidatedForTaskId` 去重写在 `await` 之前，避免 React StrictMode 双调度时重复回滚。

## 三、检查项逐项

### React/TS 前端

- [x] **类型安全**：无新增 `any` / `interface{}`；`optimisticallyRemoved` 显式 `Set<number>`；`instanceNumber` 显式 `number | undefined`。
- [x] **Hook 模式**：useEffect 依赖完整（`[lastTask?.id, lastTask?.status]`），异步路径用 `cancelled` flag 防泄漏。
- [x] **API 模式**：所有 mutation 走 `useAddObject` / `useDeleteObject` / `useUpdateParameters`，无新增直连 fetch。
- [x] **XSS / Token**：本次未涉及外部输入渲染、无 Token 处理。
- [x] **i18n**：新增 key 已同步 zh-CN / en-US。

### 通用

- [x] **运营商硬编码**：未涉及。
- [x] **SQL 拼接**：未涉及。
- [x] **panic / 异常**：仅 `console.warn`，无新增 throw。
- [x] **重复代码**：`schemaParentPrefix` 抽为模块内私有函数，避免两处 mutation 重复内联同样逻辑。
- [x] **日志质量**：silent rollback 的 warn 带 `inst` + 原始错误对象，便于排查。

### 三皮肤铁律

- v1 (webcode)：本次主体改动；浏览器侧已实测（添加 / 删除 / 编辑 / 失败回滚 / 跨表静态）。
- v2 (webcode-v2)：未直接改组件层；`frontend-core` 共享改动是把 invalidate 范围**变窄**，不引入新失败路径；typecheck 维持基线 19 条 pre-existing 不变。
- v3 (webcode-v3)：未直接改组件层；typecheck 0 错误。

## 四、机械门记录

| 门 | 结果 |
| --- | --- |
| `cd omcgo && go build ./...` | ✅ 通过 |
| `cd omcgo && go test ./...` | ⚠️ `TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren` 失败 —— **pre-existing**（main 同样失败，见 `/memories/repo/ship-preexisting-failures.md`），本次未改 omcgo |
| `cd omcmb && npm run typecheck` | ⚠️ webcode 130 / webcode-v2 19 / webcode-v3 0 —— **pre-existing**（main 同样数量，antd6 升级遗留）。本次改动的 6 个文件自身无类型错误 |
| 前端 v1 浏览器冒烟 | ✅ 在 conversation 中已多轮实测（无 `未找到/加载失败/Error/pageerror`） |
| 新端点 E/R | N/A（无新端点） |
| 迁移演练 | N/A（无迁移） |

## 五、Pre-existing 验证

按 `/memories/repo/ship-preexisting-failures.md` 选项 C：

1. `git stash push -u` + `git switch main` 复现 → main 上 webcode 130 / webcode-v2 19 / `TestParameterTreeHandler_*` 同样失败。
2. 切回 `fix/quicksettings-multi-instance-ux` + `git stash pop` → 错误数完全一致，本次未引入新失败。
3. 推进当前 PR；pre-existing 不在本次 commit 范围内。

## 六、结论

**PASS** —— 无 CRITICAL / WARNING，可进入 P8 提交。
